package repository

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/seeds-metrics/analytics-backend/internal/models"
)

const MissingValueSentinel = "__MISSING__"

// DashboardRepository handles dashboard data queries
type DashboardRepository struct {
	db *sql.DB
}

// NewDashboardRepository creates a new dashboard repository
func NewDashboardRepository(db *sql.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
}

// RecalculateAllLoanFields triggers comprehensive recalculation of all computed fields for all loans
// This calls the recalculate_all_loan_fields() stored procedure which:
// - Recalculates all outstanding balances (principal, interest, fees)
// - Recalculates DPD and risk indicators (fimr_tagged, early_indicator_tagged)
// - Recalculates repayment delay rates
// - Recalculates timeliness scores and repayment health scores
// - Updates all 17,419 loans in a single efficient operation
func (r *DashboardRepository) RecalculateAllLoanFields() (int64, error) {
	query := `
		SELECT total_loans_processed, loans_updated, execution_time_ms
		FROM recalculate_all_loan_fields()
	`

	var totalLoans, loansUpdated, executionTimeMs int64
	err := r.db.QueryRow(query).Scan(&totalLoans, &loansUpdated, &executionTimeMs)
	if err != nil {
		return 0, fmt.Errorf("failed to recalculate loan fields: %w", err)
	}

	return loansUpdated, nil
}

// GetPortfolioLoanMetrics retrieves loan-level aggregated metrics for portfolio calculations
func (r *DashboardRepository) GetPortfolioLoanMetrics(filters map[string]interface{}) (*models.PortfolioLoanMetrics, error) {
	query := `
		SELECT
			-- Active vs Inactive Loans
			COUNT(CASE WHEN total_outstanding > 2000
				AND days_since_last_repayment < 6 THEN 1 END) as active_loans_count,
			COALESCE(SUM(CASE WHEN total_outstanding > 2000
				AND days_since_last_repayment < 6
				THEN total_outstanding END), 0) as active_loans_volume,
			COUNT(CASE WHEN total_outstanding <= 2000
				OR days_since_last_repayment > 5 THEN 1 END) as inactive_loans_count,
			COALESCE(SUM(CASE WHEN total_outstanding <= 2000
				OR days_since_last_repayment > 5
				THEN total_outstanding END), 0) as inactive_loans_volume,

			-- ROT (Risk of Termination) Analysis
			COUNT(CASE WHEN (CURRENT_DATE - disbursement_date::date) < 7 AND current_dpd > 4 THEN 1 END) as early_rot_count,
			COALESCE(SUM(CASE WHEN (CURRENT_DATE - disbursement_date::date) < 7 AND current_dpd > 4
				THEN total_outstanding END), 0) as early_rot_volume,
			COUNT(CASE WHEN (CURRENT_DATE - disbursement_date::date) >= 7 AND current_dpd > 4 THEN 1 END) as late_rot_count,
			COALESCE(SUM(CASE WHEN (CURRENT_DATE - disbursement_date::date) >= 7 AND current_dpd > 4
				THEN total_outstanding END), 0) as late_rot_volume,

			-- Portfolio Repayment Behavior Metrics (only active loans)
			COALESCE(AVG(CASE WHEN total_outstanding > 2000
				THEN current_dpd END), 0) as avg_days_past_due,
			COALESCE(AVG(CASE WHEN total_outstanding > 2000
				THEN timeliness_score END), 0) as avg_timeliness_score
		FROM loans l
		INNER JOIN officers o ON l.officer_id = o.officer_id
		WHERE UPPER(l.status) = 'ACTIVE'
			AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
	`

	args := []interface{}{}
	argCount := 1

	// Apply wave filter
	if wave, ok := filters["wave"].(string); ok && wave != "" {
		query += fmt.Sprintf(" AND l.wave = $%d", argCount)
		args = append(args, wave)
		argCount++
	}

	metrics := &models.PortfolioLoanMetrics{}
	err := r.db.QueryRow(query, args...).Scan(
		&metrics.ActiveLoansCount,
		&metrics.ActiveLoansVolume,
		&metrics.InactiveLoansCount,
		&metrics.InactiveLoansVolume,
		&metrics.EarlyROTCount,
		&metrics.EarlyROTVolume,
		&metrics.LateROTCount,
		&metrics.LateROTVolume,
		&metrics.AvgDaysPastDue,
		&metrics.AvgTimelinessScore,
	)

	if err != nil {
		return nil, err
	}

	return metrics, nil
}

// GetActualOverdue15d calculates the actual overdue amount (only installments due to date)
// for loans with current_dpd >= 15
func (r *DashboardRepository) GetActualOverdue15d(filters map[string]interface{}) (float64, error) {
	// First, try to get from loan_schedule table (most accurate)
	scheduleQuery := `
		SELECT
			COALESCE(SUM(ls.total_due - ls.amount_paid), 0) as actual_overdue_15d
		FROM loan_schedule ls
		INNER JOIN loans l ON ls.loan_id = l.loan_id
		INNER JOIN officers o ON l.officer_id = o.officer_id
		WHERE l.current_dpd >= 15
			AND UPPER(l.status) = 'ACTIVE'
			AND ls.due_date <= CURRENT_DATE
			AND ls.payment_status IN ('Pending', 'Partial', 'Overdue')
			AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
	`

	args := []interface{}{}
	argCount := 1

	// Apply wave filter to schedule query
	if wave, ok := filters["wave"].(string); ok && wave != "" {
		scheduleQuery += fmt.Sprintf(" AND l.wave = $%d", argCount)
		args = append(args, wave)
		argCount++
	}

	var actualOverdue15d float64
	err := r.db.QueryRow(scheduleQuery, args...).Scan(&actualOverdue15d)
	if err != nil {
		return 0, err
	}

	// If loan_schedule has no data, calculate from loans table
	// Estimate based on loan age and total outstanding
	if actualOverdue15d == 0 {
		fallbackQuery := `
			SELECT
				COALESCE(SUM(
					CASE
						-- Calculate proportion of loan term that has elapsed
						WHEN l.loan_term_days > 0 THEN
							(l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) *
							LEAST(1.0, GREATEST(0.0, (CURRENT_DATE - l.disbursement_date::date)::float / l.loan_term_days::float))
						ELSE
							(l.principal_outstanding + l.interest_outstanding + l.fees_outstanding)
					END
				), 0) as estimated_actual_overdue
			FROM loans l
			INNER JOIN officers o ON l.officer_id = o.officer_id
			WHERE l.current_dpd >= 15
				AND UPPER(l.status) = 'ACTIVE'
				AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
		`

		fallbackArgs := []interface{}{}
		fallbackArgCount := 1

		// Apply wave filter to fallback query
		if wave, ok := filters["wave"].(string); ok && wave != "" {
			fallbackQuery += fmt.Sprintf(" AND l.wave = $%d", fallbackArgCount)
			fallbackArgs = append(fallbackArgs, wave)
			fallbackArgCount++
		}

		err = r.db.QueryRow(fallbackQuery, fallbackArgs...).Scan(&actualOverdue15d)
		if err != nil {
			return 0, err
		}
	}

	return actualOverdue15d, nil
}

// GetTotalDPDLoans calculates the total count and actual_outstanding for loans with current_dpd > 0
// and status in ('Active', 'Defaulted')
func (r *DashboardRepository) GetTotalDPDLoans(filters map[string]interface{}) (int, float64, error) {
	query := `
		SELECT
			COUNT(*) as total_dpd_loans_count,
			COALESCE(SUM(l.actual_outstanding), 0) as total_dpd_actual_outstanding
		FROM loans l
		INNER JOIN officers o ON l.officer_id = o.officer_id
		WHERE l.current_dpd > 0
			AND UPPER(l.status) IN ('ACTIVE', 'DEFAULTED')
			AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
	`

	args := []interface{}{}
	argCount := 1

	// Apply wave filter
	if wave, ok := filters["wave"].(string); ok && wave != "" {
		query += fmt.Sprintf(" AND l.wave = $%d", argCount)
		args = append(args, wave)
		argCount++
	}

	// Apply region filter (multi-select support)
	if region, ok := filters["region"].(string); ok && region != "" {
		regions := strings.Split(region, ",")
		if len(regions) == 1 {
			query += fmt.Sprintf(" AND l.region = $%d", argCount)
			args = append(args, regions[0])
			argCount++
		} else {
			// Build IN clause for multiple regions
			placeholders := []string{}
			for _, r := range regions {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
				args = append(args, strings.TrimSpace(r))
				argCount++
			}
			query += fmt.Sprintf(" AND l.region IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	var count int
	var actualOutstanding float64
	err := r.db.QueryRow(query, args...).Scan(&count, &actualOutstanding)
	if err != nil {
		return 0, 0, err
	}

	return count, actualOutstanding, nil
}

// GetOfficers retrieves all officers with their raw metrics
func (r *DashboardRepository) GetOfficers(filters map[string]interface{}) ([]*models.DashboardOfficerMetrics, error) {
	query := `
		WITH loan_repayments AS (
			SELECT
				l.loan_id,
				l.officer_id,
				l.loan_amount,
				l.interest_rate,
				l.fee_amount,
				SUM(r.payment_amount) as total_repayments
			FROM loans l
			LEFT JOIN repayments r ON l.loan_id = r.loan_id AND r.is_reversed = false
			GROUP BY l.loan_id, l.officer_id, l.loan_amount, l.interest_rate, l.fee_amount
		)
		SELECT
			o.officer_id,
			o.officer_name,
			COALESCE(o.officer_email, '') as officer_email,
			o.region,
			o.branch,
			COALESCE(o.primary_channel, '') as primary_channel,
		o.user_type,
			o.hire_date,
			o.supervisor_email,
			o.supervisor_name,
			o.vertical_lead_email,
			o.vertical_lead_name,
			-- Raw metrics (to be aggregated from loans)
			COALESCE(SUM(CASE WHEN l.fimr_tagged THEN 1 ELSE 0 END), 0) as first_miss,
			COALESCE(COUNT(DISTINCT l.loan_id), 0) as disbursed,
			COALESCE(SUM(CASE WHEN l.current_dpd BETWEEN 1 AND 6 THEN l.principal_outstanding ELSE 0 END), 0) as dpd1to6_bal,
			COALESCE(SUM(l.principal_outstanding + l.interest_outstanding + l.fees_outstanding), 0) as amount_due_7d,
			COALESCE(SUM(CASE WHEN l.current_dpd BETWEEN 7 AND 30 THEN l.principal_outstanding ELSE 0 END), 0) as moved_to_7to30,
			COALESCE(SUM(CASE WHEN l.current_dpd BETWEEN 1 AND 6 THEN l.principal_outstanding ELSE 0 END), 0) as prev_dpd1to6_bal,
			-- Calculate fees collected from repayments (proportional allocation)
			COALESCE(SUM(
				CASE
					WHEN lr.loan_amount * (1 + lr.interest_rate) + lr.fee_amount > 0 THEN
						lr.total_repayments * lr.fee_amount / (lr.loan_amount * (1 + lr.interest_rate) + lr.fee_amount)
					ELSE 0
				END
			), 0) as fees_collected,
			COALESCE(SUM(l.fee_amount), 0) as fees_due,
			-- Calculate interest collected from repayments (proportional allocation)
			COALESCE(SUM(
				CASE
					WHEN lr.loan_amount * (1 + lr.interest_rate) + lr.fee_amount > 0 THEN
						lr.total_repayments * (lr.loan_amount * lr.interest_rate) / (lr.loan_amount * (1 + lr.interest_rate) + lr.fee_amount)
					ELSE 0
				END
			), 0) as interest_collected,
			COALESCE(SUM(CASE WHEN l.current_dpd >= 15 THEN l.principal_outstanding ELSE 0 END), 0) as overdue_15d,
			COALESCE(SUM(l.principal_outstanding), 0) as total_portfolio,
			COALESCE(SUM(l.principal_outstanding), 0) as par15_mid_month,
			0 as waivers,
			0 as backdated,
			0 as entries,
			0 as reversals,
			false as had_float_gap,
			-- NEW: Repayment behavior metrics (only for loans with total_outstanding > 2000)
			COALESCE(AVG(CASE WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) > 2000 THEN l.timeliness_score ELSE NULL END), 0) as avg_timeliness_score,
			COALESCE(AVG(CASE WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) > 2000 THEN l.repayment_health ELSE NULL END), 0) as avg_repayment_health,
			COALESCE(AVG(CASE WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) > 2000 THEN l.days_since_last_repayment ELSE NULL END), 0) as avg_days_since_last_repayment,
			COALESCE(AVG(CASE WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) > 2000 THEN l.loan_age ELSE NULL END), 0) as avg_loan_age,
			COALESCE(COUNT(CASE WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) > 2000 THEN 1 ELSE NULL END), 0) as active_loans_count
		FROM officers o
		LEFT JOIN loans l ON o.officer_id = l.officer_id
		LEFT JOIN loan_repayments lr ON l.loan_id = lr.loan_id
		WHERE 1=1
			AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
	`

	args := []interface{}{}
	argCount := 1

	// Apply filters
	if branch, ok := filters["branch"].(string); ok && branch != "" {
		query += fmt.Sprintf(" AND o.branch = $%d", argCount)
		args = append(args, branch)
		argCount++
	}

	if region, ok := filters["region"].(string); ok && region != "" {
		// Support comma-separated regions for multi-select
		regions := strings.Split(region, ",")
		if len(regions) == 1 {
			query += fmt.Sprintf(" AND o.region = $%d", argCount)
			args = append(args, regions[0])
			argCount++
		} else {
			// Build IN clause for multiple regions
			placeholders := []string{}
			for _, r := range regions {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
				args = append(args, strings.TrimSpace(r))
				argCount++
			}
			query += fmt.Sprintf(" AND o.region IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	if channel, ok := filters["channel"].(string); ok && channel != "" {
		query += fmt.Sprintf(" AND o.primary_channel = $%d", argCount)
		args = append(args, channel)
		argCount++
	}

	if userType, ok := filters["user_type"].(string); ok && userType != "" {
		query += fmt.Sprintf(" AND o.user_type = $%d", argCount)
		args = append(args, userType)
		argCount++
	}

	if wave, ok := filters["wave"].(string); ok && wave != "" {
		query += fmt.Sprintf(" AND l.wave = $%d", argCount)
		args = append(args, wave)
		argCount++
	}

	if officerEmail, ok := filters["officer_email"].(string); ok && officerEmail != "" {
		query += fmt.Sprintf(" AND (o.officer_email ILIKE $%d OR o.officer_name ILIKE $%d)", argCount, argCount)
		args = append(args, "%"+officerEmail+"%")
		argCount++
	}

	query += " GROUP BY o.officer_id, o.officer_name, o.officer_email, o.region, o.branch, o.primary_channel, o.user_type, o.hire_date"

	// Apply sorting
	sortBy := "o.officer_name"
	if sort, ok := filters["sort_by"].(string); ok && sort != "" {
		sortBy = sort
	}
	sortDir := "ASC"
	if dir, ok := filters["sort_dir"].(string); ok && strings.ToUpper(dir) == "DESC" {
		sortDir = "DESC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortDir)

	// Apply pagination
	limit := 50
	if l, ok := filters["limit"].(int); ok && l > 0 {
		limit = l
	}
	offset := 0
	if page, ok := filters["page"].(int); ok && page > 0 {
		offset = (page - 1) * limit
	}
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	// Log the query for debugging
	log.Printf("🔍 GetOfficers SQL Query: %s", query)
	log.Printf("🔍 GetOfficers SQL Args: %v", args)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		log.Printf("❌ GetOfficers SQL Error: %v", err)
		return nil, err
	}
	defer rows.Close()

	officers := []*models.DashboardOfficerMetrics{}
	for rows.Next() {
		officer := &models.DashboardOfficerMetrics{
			RawMetrics: &models.RawMetrics{},
		}

		var supervisorEmail, supervisorName, verticalLeadEmail, verticalLeadName sql.NullString

		err := rows.Scan(
			&officer.OfficerID,
			&officer.Name,
			&officer.Email,
			&officer.Region,
			&officer.Branch,
			&officer.Channel,
			&officer.UserType,
			&officer.HireDate,
			&supervisorEmail,
			&supervisorName,
			&verticalLeadEmail,
			&verticalLeadName,
			&officer.RawMetrics.FirstMiss,
			&officer.RawMetrics.Disbursed,
			&officer.RawMetrics.Dpd1to6Bal,
			&officer.RawMetrics.AmountDue7d,
			&officer.RawMetrics.MovedTo7to30,
			&officer.RawMetrics.PrevDpd1to6Bal,
			&officer.RawMetrics.FeesCollected,
			&officer.RawMetrics.FeesDue,
			&officer.RawMetrics.InterestCollected,
			&officer.RawMetrics.Overdue15d,
			&officer.RawMetrics.TotalPortfolio,
			&officer.RawMetrics.Par15MidMonth,
			&officer.RawMetrics.Waivers,
			&officer.RawMetrics.Backdated,
			&officer.RawMetrics.Entries,
			&officer.RawMetrics.Reversals,
			&officer.RawMetrics.HadFloatGap,
			&officer.RawMetrics.AvgTimelinessScore,
			&officer.RawMetrics.AvgRepaymentHealth,
			&officer.RawMetrics.AvgDaysSinceLastRepayment,
			&officer.RawMetrics.AvgLoanAge,
			&officer.RawMetrics.ActiveLoansCount,
		)
		if err != nil {
			return nil, err
		}

		// Handle NULL values for supervisor and vertical lead fields
		if supervisorEmail.Valid {
			officer.SupervisorEmail = &supervisorEmail.String
		}
		if supervisorName.Valid {
			officer.SupervisorName = &supervisorName.String
		}
		if verticalLeadEmail.Valid {
			officer.VerticalLeadEmail = &verticalLeadEmail.String
		}
		if verticalLeadName.Valid {
			officer.VerticalLeadName = &verticalLeadName.String
		}

		officers = append(officers, officer)
	}

	return officers, nil
}

// GetOfficerByID retrieves a single officer by ID
func (r *DashboardRepository) GetOfficerByID(officerID string) (*models.DashboardOfficerMetrics, error) {
	query := `
		WITH loan_repayments AS (
			SELECT
				l.loan_id,
				l.officer_id,
				l.loan_amount,
				l.interest_rate,
				l.fee_amount,
				SUM(r.payment_amount) as total_repayments
			FROM loans l
			LEFT JOIN repayments r ON l.loan_id = r.loan_id AND r.is_reversed = false
			WHERE l.officer_id = $1
			GROUP BY l.loan_id, l.officer_id, l.loan_amount, l.interest_rate, l.fee_amount
		)
		SELECT
			o.officer_id,
			o.officer_name,
			COALESCE(o.officer_email, '') as officer_email,
			o.region,
			o.branch,
			COALESCE(o.primary_channel, '') as primary_channel,
			o.user_type,
			o.hire_date,
			o.supervisor_email,
			o.supervisor_name,
			o.vertical_lead_email,
			o.vertical_lead_name,
			COALESCE(SUM(CASE WHEN l.fimr_tagged THEN 1 ELSE 0 END), 0) as first_miss,
			COALESCE(COUNT(DISTINCT l.loan_id), 0) as disbursed,
			COALESCE(SUM(CASE WHEN l.current_dpd BETWEEN 1 AND 6 THEN l.principal_outstanding ELSE 0 END), 0) as dpd1to6_bal,
			COALESCE(SUM(l.principal_outstanding + l.interest_outstanding + l.fees_outstanding), 0) as amount_due_7d,
			COALESCE(SUM(CASE WHEN l.current_dpd BETWEEN 7 AND 30 THEN l.principal_outstanding ELSE 0 END), 0) as moved_to_7to30,
			COALESCE(SUM(CASE WHEN l.current_dpd BETWEEN 1 AND 6 THEN l.principal_outstanding ELSE 0 END), 0) as prev_dpd1to6_bal,
			-- Calculate fees collected from repayments (proportional allocation)
			COALESCE(SUM(
				CASE
					WHEN lr.loan_amount * (1 + lr.interest_rate) + lr.fee_amount > 0 THEN
						lr.total_repayments * lr.fee_amount / (lr.loan_amount * (1 + lr.interest_rate) + lr.fee_amount)
					ELSE 0
				END
			), 0) as fees_collected,
			COALESCE(SUM(l.fee_amount), 0) as fees_due,
			-- Calculate interest collected from repayments (proportional allocation)
			COALESCE(SUM(
				CASE
					WHEN lr.loan_amount * (1 + lr.interest_rate) + lr.fee_amount > 0 THEN
						lr.total_repayments * (lr.loan_amount * lr.interest_rate) / (lr.loan_amount * (1 + lr.interest_rate) + lr.fee_amount)
					ELSE 0
				END
			), 0) as interest_collected,
			COALESCE(SUM(CASE WHEN l.current_dpd >= 15 THEN l.principal_outstanding ELSE 0 END), 0) as overdue_15d,
			COALESCE(SUM(l.principal_outstanding), 0) as total_portfolio,
			COALESCE(SUM(l.principal_outstanding), 0) as par15_mid_month,
			0 as waivers,
			0 as backdated,
			0 as entries,
			0 as reversals,
			false as had_float_gap,
			-- Repayment behavior metrics (align with list query)
			COALESCE(AVG(CASE WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) > 2000 THEN l.timeliness_score ELSE NULL END), 0) as avg_timeliness_score,
			COALESCE(AVG(CASE WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) > 2000 THEN l.repayment_health ELSE NULL END), 0) as avg_repayment_health,
			COALESCE(AVG(CASE WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) > 2000 THEN l.days_since_last_repayment ELSE NULL END), 0) as avg_days_since_last_repayment,
			COALESCE(AVG(CASE WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) > 2000 THEN l.loan_age ELSE NULL END), 0) as avg_loan_age,
			COALESCE(COUNT(CASE WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) > 2000 THEN 1 ELSE NULL END), 0) as active_loans_count
		FROM officers o
		LEFT JOIN loans l ON o.officer_id = l.officer_id
		LEFT JOIN loan_repayments lr ON l.loan_id = lr.loan_id
		WHERE o.officer_id = $1
			AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
		GROUP BY o.officer_id, o.officer_name, o.region, o.branch, o.primary_channel, o.user_type, o.hire_date, o.supervisor_email, o.supervisor_name, o.vertical_lead_email, o.vertical_lead_name
	`

	officer := &models.DashboardOfficerMetrics{
		RawMetrics: &models.RawMetrics{},
	}

	var supervisorEmail, supervisorName, verticalLeadEmail, verticalLeadName sql.NullString

	err := r.db.QueryRow(query, officerID).Scan(
		&officer.OfficerID,
		&officer.Name,
		&officer.Email,
		&officer.Region,
		&officer.Branch,
		&officer.Channel,
		&officer.UserType,
		&officer.HireDate,
		&supervisorEmail,
		&supervisorName,
		&verticalLeadEmail,
		&verticalLeadName,
		&officer.RawMetrics.FirstMiss,
		&officer.RawMetrics.Disbursed,
		&officer.RawMetrics.Dpd1to6Bal,
		&officer.RawMetrics.AmountDue7d,
		&officer.RawMetrics.MovedTo7to30,
		&officer.RawMetrics.PrevDpd1to6Bal,
		&officer.RawMetrics.FeesCollected,
		&officer.RawMetrics.FeesDue,
		&officer.RawMetrics.InterestCollected,
		&officer.RawMetrics.Overdue15d,
		&officer.RawMetrics.TotalPortfolio,
		&officer.RawMetrics.Par15MidMonth,
		&officer.RawMetrics.Waivers,
		&officer.RawMetrics.Backdated,
		&officer.RawMetrics.Entries,
		&officer.RawMetrics.Reversals,
		&officer.RawMetrics.HadFloatGap,
		&officer.RawMetrics.AvgTimelinessScore,
		&officer.RawMetrics.AvgRepaymentHealth,
		&officer.RawMetrics.AvgDaysSinceLastRepayment,
		&officer.RawMetrics.AvgLoanAge,
		&officer.RawMetrics.ActiveLoansCount,
	)

	if err != nil {
		return nil, err
	}

	// Handle NULL values for supervisor and vertical lead fields
	if supervisorEmail.Valid {
		officer.SupervisorEmail = &supervisorEmail.String
	}
	if supervisorName.Valid {
		officer.SupervisorName = &supervisorName.String
	}
	if verticalLeadEmail.Valid {
		officer.VerticalLeadEmail = &verticalLeadEmail.String
	}
	if verticalLeadName.Valid {
		officer.VerticalLeadName = &verticalLeadName.String
	}

	return officer, nil
}

// GetFIMRLoans retrieves loans that missed first installment
func (r *DashboardRepository) GetFIMRLoans(filters map[string]interface{}) ([]*models.FIMRLoan, error) {
	query := `
		SELECT
			l.loan_id,
			l.officer_id,
			o.officer_name as officer_name,
			l.region,
			l.branch,
			l.customer_id,
			l.customer_name,
			l.customer_phone,
			l.disbursement_date,
				l.loan_amount,
				l.maturity_date,
				l.actual_outstanding,
				l.total_repayments,
				l.first_payment_due_date as first_payment_due_date,
			l.first_payment_received_date,
			l.days_since_due,
			CASE
				WHEN l.loan_term_days > 0 THEN l.loan_amount / l.loan_term_days
				ELSE 0
			END as amount_due_1st_installment,
			l.total_principal_paid as amount_paid,
			l.principal_outstanding as outstanding_balance,
			l.current_dpd,
			l.channel,
				l.status,
				l.django_status,
				l.fimr_tagged as fimr_tagged
		FROM loans l
		JOIN officers o ON l.officer_id = o.officer_id
		WHERE l.fimr_tagged = true
	`

	args := []interface{}{}
	argCount := 1

	// Apply filters
	if officerID, ok := filters["officer_id"].(string); ok && officerID != "" {
		query += fmt.Sprintf(" AND l.officer_id = $%d", argCount)
		args = append(args, officerID)
		argCount++
	}

	if branch, ok := filters["branch"].(string); ok && branch != "" {
		query += fmt.Sprintf(" AND l.branch = $%d", argCount)
		args = append(args, branch)
		argCount++
	}

	if region, ok := filters["region"].(string); ok && region != "" {
		// Support comma-separated regions for multi-select
		regions := strings.Split(region, ",")
		if len(regions) == 1 {
			query += fmt.Sprintf(" AND l.region = $%d", argCount)
			args = append(args, regions[0])
			argCount++
		} else {
			// Build IN clause for multiple regions
			placeholders := []string{}
			for _, r := range regions {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
				args = append(args, strings.TrimSpace(r))
				argCount++
			}
			query += fmt.Sprintf(" AND l.region IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	if channel, ok := filters["channel"].(string); ok && channel != "" {
		query += fmt.Sprintf(" AND l.channel = $%d", argCount)
		args = append(args, channel)
		argCount++
	}

	if status, ok := filters["status"].(string); ok && status != "" {
		query += fmt.Sprintf(" AND l.status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	// Raw Django status filter - supports comma-separated values and optional missing sentinel
	if djangoStatus, ok := filters["django_status"].(string); ok && djangoStatus != "" {
		statuses := strings.Split(djangoStatus, ",")
		nonMissing := []string{}
		includeMissing := false

		for _, s := range statuses {
			value := strings.TrimSpace(s)
			if value == "" {
				continue
			}
			if value == MissingValueSentinel {
				includeMissing = true
			} else {
				nonMissing = append(nonMissing, value)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.django_status = $%d", argCount))
			args = append(args, nonMissing[0])
			argCount++
		} else if len(nonMissing) > 1 {
			placeholders := make([]string, len(nonMissing))
			for i, s := range nonMissing {
				placeholders[i] = fmt.Sprintf("$%d", argCount)
				args = append(args, s)
				argCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.django_status IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.django_status IS NULL OR l.django_status = '')")
		}

		if len(conditions) > 0 {
			query += " AND (" + strings.Join(conditions, " OR ") + ")"
		}
	}

	if wave, ok := filters["wave"].(string); ok && wave != "" {
		query += fmt.Sprintf(" AND l.wave = $%d", argCount)
		args = append(args, wave)
		argCount++
	}

	// Apply sorting
	sortBy := "l.disbursement_date"
	if sort, ok := filters["sort_by"].(string); ok && sort != "" {
		sortBy = sort
	}
	sortDir := "DESC"
	if dir, ok := filters["sort_dir"].(string); ok && strings.ToUpper(dir) == "ASC" {
		sortDir = "ASC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortDir)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loans := []*models.FIMRLoan{}
	for rows.Next() {
		loan := &models.FIMRLoan{}
		var firstPaymentDueDate sql.NullString
		var maturityDate sql.NullString
		var firstPaymentReceivedDate sql.NullString
		var daysSinceDue sql.NullInt64
		var djangoStatus sql.NullString
		err := rows.Scan(
			&loan.LoanID,
			&loan.OfficerID,
			&loan.OfficerName,
			&loan.Region,
			&loan.Branch,
			&loan.CustomerID,
			&loan.CustomerName,
			&loan.CustomerPhone,
			&loan.DisbursementDate,
			&loan.LoanAmount,
			&maturityDate,
			&loan.ActualOutstanding,
			&loan.TotalRepayments,
			&firstPaymentDueDate,
			&firstPaymentReceivedDate,
			&daysSinceDue,
			&loan.AmountDue1stInstallment,
			&loan.AmountPaid,
			&loan.OutstandingBalance,
			&loan.CurrentDPD,
			&loan.Channel,
			&loan.Status,
			&djangoStatus,
			&loan.FIMRTagged,
		)
		if err != nil {
			return nil, err
		}
		if firstPaymentDueDate.Valid {
			loan.FirstPaymentDueDate = firstPaymentDueDate.String
		}
		if maturityDate.Valid {
			loan.MaturityDate = maturityDate.String
		}
		if firstPaymentReceivedDate.Valid {
			loan.FirstPaymentReceivedDate = &firstPaymentReceivedDate.String
		}
		if daysSinceDue.Valid {
			days := int(daysSinceDue.Int64)
			loan.DaysSinceDue = &days
		}
		if djangoStatus.Valid {
			loan.DjangoStatus = djangoStatus.String
		}
		loans = append(loans, loan)
	}

	return loans, nil
}

// GetEarlyIndicatorLoans retrieves loans in early delinquency (DPD 1-30)
func (r *DashboardRepository) GetEarlyIndicatorLoans(filters map[string]interface{}) ([]*models.EarlyIndicatorLoan, error) {
	query := `
		SELECT
			l.loan_id,
			l.officer_id,
			o.officer_name as officer_name,
			l.region,
			l.branch,
			l.customer_id,
			l.customer_name,
			l.customer_phone,
			l.disbursement_date,
			l.loan_amount,
			l.current_dpd,
			'Current' as previous_dpd_status,
			0 as days_in_current_status,
			l.total_outstanding as amount_due,
			l.total_principal_paid + l.total_interest_paid + l.total_fees_paid as amount_paid,
			l.principal_outstanding as outstanding_balance,
			l.channel,
			l.status,
			l.fimr_tagged as fimr_tagged,
			'Stable' as roll_direction,
			(SELECT MAX(r.payment_date) FROM repayments r WHERE r.loan_id = l.loan_id AND NOT r.is_reversed) as last_payment_date
		FROM loans l
		JOIN officers o ON l.officer_id = o.officer_id
		WHERE l.current_dpd BETWEEN 1 AND 30
	`

	args := []interface{}{}
	argCount := 1

	// Apply filters
	if officerID, ok := filters["officer_id"].(string); ok && officerID != "" {
		query += fmt.Sprintf(" AND l.officer_id = $%d", argCount)
		args = append(args, officerID)
		argCount++
	}

	if branch, ok := filters["branch"].(string); ok && branch != "" {
		query += fmt.Sprintf(" AND l.branch = $%d", argCount)
		args = append(args, branch)
		argCount++
	}

	if region, ok := filters["region"].(string); ok && region != "" {
		// Support comma-separated regions for multi-select
		regions := strings.Split(region, ",")
		if len(regions) == 1 {
			query += fmt.Sprintf(" AND l.region = $%d", argCount)
			args = append(args, regions[0])
			argCount++
		} else {
			// Build IN clause for multiple regions
			placeholders := []string{}
			for _, r := range regions {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
				args = append(args, strings.TrimSpace(r))
				argCount++
			}
			query += fmt.Sprintf(" AND l.region IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	if channel, ok := filters["channel"].(string); ok && channel != "" {
		query += fmt.Sprintf(" AND l.channel = $%d", argCount)
		args = append(args, channel)
		argCount++
	}

	if status, ok := filters["status"].(string); ok && status != "" {
		// Status filter for DPD ranges
		switch status {
		case "D1-3":
			query += " AND l.current_dpd BETWEEN 1 AND 3"
		case "D4-6":
			query += " AND l.current_dpd BETWEEN 4 AND 6"
		case "D7-15":
			query += " AND l.current_dpd BETWEEN 7 AND 15"
		case "D16-30":
			query += " AND l.current_dpd BETWEEN 16 AND 30"
		}
	}

	if wave, ok := filters["wave"].(string); ok && wave != "" {
		query += fmt.Sprintf(" AND l.wave = $%d", argCount)
		args = append(args, wave)
		argCount++
	}

	// Apply sorting
	sortBy := "l.current_dpd"
	if sort, ok := filters["sort_by"].(string); ok && sort != "" {
		sortBy = sort
	}
	sortDir := "DESC"
	if dir, ok := filters["sort_dir"].(string); ok && strings.ToUpper(dir) == "ASC" {
		sortDir = "ASC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortDir)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loans := []*models.EarlyIndicatorLoan{}
	for rows.Next() {
		loan := &models.EarlyIndicatorLoan{}
		var lastPaymentDate sql.NullString

		err := rows.Scan(
			&loan.LoanID,
			&loan.OfficerID,
			&loan.OfficerName,
			&loan.Region,
			&loan.Branch,
			&loan.CustomerID,
			&loan.CustomerName,
			&loan.CustomerPhone,
			&loan.DisbursementDate,
			&loan.LoanAmount,
			&loan.CurrentDPD,
			&loan.PreviousDPDStatus,
			&loan.DaysInCurrentStatus,
			&loan.AmountDue,
			&loan.AmountPaid,
			&loan.OutstandingBalance,
			&loan.Channel,
			&loan.Status,
			&loan.FIMRTagged,
			&loan.RollDirection,
			&lastPaymentDate,
		)
		if err != nil {
			return nil, err
		}

		if lastPaymentDate.Valid {
			loan.LastPaymentDate = lastPaymentDate.String
		}

		loans = append(loans, loan)
	}

	return loans, nil
}

// GetLoansSummaryMetrics calculates summary metrics for all loans matching the given filters
func (r *DashboardRepository) GetLoansSummaryMetrics(filters map[string]interface{}) (map[string]interface{}, error) {
	// Determine requested period for period-based metrics (currently used for
	// repayments-only aggregates further down). Defaults to "today" semantics
	// if not specified.
	period := ""
	if p, ok := filters["period"].(string); ok {
		period = strings.TrimSpace(strings.ToLower(p))
	}

	// Base query for summary metrics. Past maturity outstanding here is defined
	// purely as "all loans where today is past the maturity_date" (i.e.
	// maturity_date < CURRENT_DATE) and actual_outstanding is still positive,
	// independent of the selected period filter.
	query := `
			SELECT
				COUNT(*) as total_loans,
				COALESCE(SUM(l.loan_amount), 0) as total_portfolio_amount,
				COALESCE(SUM(CASE WHEN l.current_dpd > 14 THEN 1 ELSE 0 END), 0) as at_risk_count,
				COALESCE(SUM(CASE WHEN l.current_dpd > 14 THEN l.loan_amount ELSE 0 END), 0) as at_risk_amount,
				COALESCE(SUM(CASE WHEN l.current_dpd > 14 THEN l.actual_outstanding ELSE 0 END), 0) as at_risk_outstanding,
				COALESCE(SUM(CASE WHEN l.current_dpd > 0 THEN l.actual_outstanding ELSE 0 END), 0) as total_amount_in_dpd,
				COALESCE(SUM(CASE WHEN l.current_dpd > 21 THEN 1 ELSE 0 END), 0) as critical_count,
				COALESCE(SUM(CASE WHEN l.repayment_delay_rate >= 80 THEN 1 ELSE 0 END), 0) as excellent_delay_count,
				COALESCE(SUM(CASE WHEN l.repayment_delay_rate >= 40 AND l.repayment_delay_rate < 80 THEN 1 ELSE 0 END), 0) as okay_delay_count,
				COALESCE(SUM(CASE WHEN l.repayment_delay_rate < 40 THEN 1 ELSE 0 END), 0) as critical_delay_count,
				COALESCE(SUM(CASE WHEN l.actual_outstanding > 0 THEN l.daily_repayment_amount ELSE 0 END), 0) as total_due_for_today,
				COALESCE(SUM(
					CASE
						-- Past maturity outstanding: all loans for which today is past
						-- the contractual end date (maturity_date) and which still have a
						-- positive actual_outstanding balance.
						WHEN l.maturity_date IS NOT NULL
							AND l.maturity_date < CURRENT_DATE
							AND l.actual_outstanding > 0
							THEN l.actual_outstanding
						ELSE 0
					END
				), 0) as past_maturity_outstanding,
				COALESCE(SUM(CASE WHEN UPPER(l.performance_status) = 'PERFORMING' THEN 1 ELSE 0 END), 0) as performing_loans_count,
				COALESCE(SUM(CASE WHEN UPPER(l.performance_status) = 'PERFORMING' THEN l.actual_outstanding ELSE 0 END), 0) as performing_actual_outstanding
			FROM loans l
			JOIN officers o ON l.officer_id = o.officer_id
			WHERE 1=1
				AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
			`

	args := []interface{}{}
	argCount := 1

	// Apply the same filters as GetAllLoans
	if officerID, ok := filters["officer_id"].(string); ok && officerID != "" {
		query += fmt.Sprintf(" AND l.officer_id = $%d", argCount)
		args = append(args, officerID)
		argCount++
	}

	if branch, ok := filters["branch"].(string); ok && branch != "" {
		query += fmt.Sprintf(" AND l.branch = $%d", argCount)
		args = append(args, branch)
		argCount++
	}

	if region, ok := filters["region"].(string); ok && region != "" {
		regions := strings.Split(region, ",")
		if len(regions) == 1 {
			query += fmt.Sprintf(" AND l.region = $%d", argCount)
			args = append(args, regions[0])
			argCount++
		} else {
			placeholders := []string{}
			for _, r := range regions {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
				args = append(args, strings.TrimSpace(r))
				argCount++
			}
			query += fmt.Sprintf(" AND l.region IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	if channel, ok := filters["channel"].(string); ok && channel != "" {
		query += fmt.Sprintf(" AND l.channel = $%d", argCount)
		args = append(args, channel)
		argCount++
	}

	if status, ok := filters["status"].(string); ok && status != "" {
		statuses := strings.Split(status, ",")
		if len(statuses) == 1 {
			query += fmt.Sprintf(" AND l.status = $%d", argCount)
			args = append(args, statuses[0])
			argCount++
		} else {
			placeholders := []string{}
			for _, s := range statuses {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
				args = append(args, strings.TrimSpace(s))
				argCount++
			}
			query += fmt.Sprintf(" AND l.status IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	// Raw Django status filter - supports comma-separated values and optional missing sentinel
	if djangoStatus, ok := filters["django_status"].(string); ok && djangoStatus != "" {
		statuses := strings.Split(djangoStatus, ",")
		nonMissing := []string{}
		includeMissing := false

		for _, s := range statuses {
			value := strings.TrimSpace(s)
			if value == "" {
				continue
			}
			if value == MissingValueSentinel {
				includeMissing = true
			} else {
				nonMissing = append(nonMissing, value)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.django_status = $%d", argCount))
			args = append(args, nonMissing[0])
			argCount++
		} else if len(nonMissing) > 1 {
			placeholders := make([]string, len(nonMissing))
			for i, s := range nonMissing {
				placeholders[i] = fmt.Sprintf("$%d", argCount)
				args = append(args, s)
				argCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.django_status IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.django_status IS NULL OR l.django_status = '')")
		}

		if len(conditions) > 0 {
			query += " AND (" + strings.Join(conditions, " OR ") + ")"
		}
	}

	if performanceStatus, ok := filters["performance_status"].(string); ok && performanceStatus != "" {
		performanceStatuses := strings.Split(performanceStatus, ",")
		nonMissing := []string{}
		includeMissing := false

		for _, ps := range performanceStatuses {
			value := strings.TrimSpace(ps)
			if value == "" {
				continue
			}
			if value == MissingValueSentinel {
				includeMissing = true
			} else {
				nonMissing = append(nonMissing, value)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.performance_status = $%d", argCount))
			args = append(args, nonMissing[0])
			argCount++
		} else if len(nonMissing) > 1 {
			placeholders := make([]string, len(nonMissing))
			for i, ps := range nonMissing {
				placeholders[i] = fmt.Sprintf("$%d", argCount)
				args = append(args, ps)
				argCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.performance_status IN (%s)", strings.Join(placeholders, ", ")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.performance_status IS NULL OR l.performance_status = '')")
		}

		if len(conditions) > 0 {
			query += " AND (" + strings.Join(conditions, " OR ") + ")"
		}
	}

	if wave, ok := filters["wave"].(string); ok && wave != "" {
		query += fmt.Sprintf(" AND l.wave = $%d", argCount)
		args = append(args, wave)
		argCount++
	}

	if customerPhone, ok := filters["customer_phone"].(string); ok && customerPhone != "" {
		query += fmt.Sprintf(" AND l.customer_phone LIKE $%d", argCount)
		args = append(args, "%"+customerPhone+"%")
		argCount++
	}

	if verticalLeadEmail, ok := filters["vertical_lead_email"].(string); ok && verticalLeadEmail != "" {
		query += fmt.Sprintf(" AND l.vertical_lead_email = $%d", argCount)
		args = append(args, verticalLeadEmail)
		argCount++
	}

	if loanType, ok := filters["loan_type"].(string); ok && loanType != "" {
		// Support comma-separated values for multiple loan types, including a sentinel for missing values
		loanTypes := strings.Split(loanType, ",")
		nonMissing := []string{}
		includeMissing := false

		for _, lt := range loanTypes {
			value := strings.TrimSpace(lt)
			if value == "" {
				continue
			}
			if value == MissingValueSentinel {
				includeMissing = true
			} else {
				nonMissing = append(nonMissing, value)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.loan_type = $%d", argCount))
			args = append(args, nonMissing[0])
			argCount++
		} else if len(nonMissing) > 1 {
			placeholders := make([]string, len(nonMissing))
			for i, lt := range nonMissing {
				placeholders[i] = fmt.Sprintf("$%d", argCount)
				args = append(args, lt)
				argCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.loan_type IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.loan_type IS NULL OR l.loan_type = '')")
		}

		if len(conditions) > 0 {
			query += " AND (" + strings.Join(conditions, " OR ") + ")"
		}
	}

	if verificationStatus, ok := filters["verification_status"].(string); ok && verificationStatus != "" {
		// Support comma-separated values for multiple verification statuses, including a sentinel for missing values
		verificationStatuses := strings.Split(verificationStatus, ",")
		nonMissing := []string{}
		includeMissing := false

		for _, vs := range verificationStatuses {
			value := strings.TrimSpace(vs)
			if value == "" {
				continue
			}
			if value == MissingValueSentinel {
				includeMissing = true
			} else {
				nonMissing = append(nonMissing, value)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.verification_status = $%d", argCount))
			args = append(args, nonMissing[0])
			argCount++
		} else if len(nonMissing) > 1 {
			placeholders := make([]string, len(nonMissing))
			for i, vs := range nonMissing {
				placeholders[i] = fmt.Sprintf("$%d", argCount)
				args = append(args, vs)
				argCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.verification_status IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.verification_status IS NULL OR l.verification_status = '')")
		}

		if len(conditions) > 0 {
			query += " AND (" + strings.Join(conditions, " OR ") + ")"
		}
	}

	if dpdMin, ok := filters["dpd_min"].(int); ok {
		query += fmt.Sprintf(" AND l.current_dpd >= $%d", argCount)
		args = append(args, dpdMin)
		argCount++
	}

	if dpdMax, ok := filters["dpd_max"].(int); ok {
		query += fmt.Sprintf(" AND l.current_dpd <= $%d", argCount)
		args = append(args, dpdMax)
		argCount++
	}

	// Behavior-based filters (active/inactive/overdue_15d, early/late ROT, risky delay)
	// kept in sync with GetAllLoans so summary metrics match the table and exports.
	if behaviorLoanType, ok := filters["behavior_loan_type"].(string); ok && behaviorLoanType != "" {
		switch behaviorLoanType {
		case "active":
			query += " AND l.total_outstanding > 2000 AND COALESCE(l.days_since_last_repayment, 0) < 6"
		case "inactive":
			query += " AND (l.total_outstanding <= 2000 OR COALESCE(l.days_since_last_repayment, 0) > 5)"
		case "overdue_15d":
			query += " AND l.current_dpd > 15"
		}
	}

	if rotType, ok := filters["rot_type"].(string); ok && rotType != "" {
		switch rotType {
		case "early":
			query += " AND (CURRENT_DATE - l.disbursement_date::date) < 7 AND l.current_dpd > 4"
		case "late":
			query += " AND (CURRENT_DATE - l.disbursement_date::date) >= 7 AND l.current_dpd > 4"
		}
	}

	if delayType, ok := filters["delay_type"].(string); ok && delayType != "" {
		if delayType == "risky" {
			query += " AND l.status = 'Active' AND l.total_outstanding > 2000 AND l.repayment_delay_rate IS NOT NULL AND l.repayment_delay_rate < 60"
		}
	}

	// Execute query
	var totalLoans, atRiskCount, criticalCount, excellentDelayCount, okayDelayCount, criticalDelayCount, performingLoansCount int
	var totalPortfolioAmount, atRiskAmount, atRiskOutstanding, totalAmountInDPD, totalDueForToday, pastMaturityOutstanding, performingActualOutstanding float64

	err := r.db.QueryRow(query, args...).Scan(
		&totalLoans,
		&totalPortfolioAmount,
		&atRiskCount,
		&atRiskAmount,
		&atRiskOutstanding,
		&totalAmountInDPD,
		&criticalCount,
		&excellentDelayCount,
		&okayDelayCount,
		&criticalDelayCount,
		&totalDueForToday,
		&pastMaturityOutstanding,
		&performingLoansCount,
		&performingActualOutstanding,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate summary metrics: %w", err)
	}

	// Build a base WHERE clause to sum repayments made in the requested period for
	// loans matching the filters. We keep this reusable so we can calculate both
	// the overall total and a breakdown by django_status using the same filters.
	repaymentsWhere := `
			FROM repayments r
			INNER JOIN loans l ON r.loan_id = l.loan_id
			INNER JOIN officers o ON l.officer_id = o.officer_id
			WHERE r.is_reversed = false
				AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
		`

	// Apply period restriction on repayment dates. This affects only the repayments
	// aggregation; loan-level metrics (e.g. total_due_for_today) remain as currently
	// defined until collections-specific period handling is implemented for them.
	switch period {
	case "this_week":
		repaymentsWhere += `
				AND DATE(r.payment_date) >= DATE_TRUNC('week', CURRENT_DATE)::date
				AND DATE(r.payment_date) <= CURRENT_DATE
			`
	case "this_month":
		repaymentsWhere += `
				AND DATE(r.payment_date) >= DATE_TRUNC('month', CURRENT_DATE)::date
				AND DATE(r.payment_date) <= CURRENT_DATE
			`
	case "last_month":
		repaymentsWhere += `
				AND DATE(r.payment_date) >= (DATE_TRUNC('month', CURRENT_DATE) - INTERVAL '1 month')::date
				AND DATE(r.payment_date) < DATE_TRUNC('month', CURRENT_DATE)::date
			`
	default: // "today" or any unrecognised value
		repaymentsWhere += `
				AND DATE(r.payment_date) = CURRENT_DATE
			`
	}

	// Apply the same filters to the repayments WHERE clause
	repaymentsArgs := []interface{}{}
	repaymentsArgCount := 1

	if officerID, ok := filters["officer_id"].(string); ok && officerID != "" {
		repaymentsWhere += fmt.Sprintf(" AND l.officer_id = $%d", repaymentsArgCount)
		repaymentsArgs = append(repaymentsArgs, officerID)
		repaymentsArgCount++
	}

	if branch, ok := filters["branch"].(string); ok && branch != "" {
		repaymentsWhere += fmt.Sprintf(" AND l.branch = $%d", repaymentsArgCount)
		repaymentsArgs = append(repaymentsArgs, branch)
		repaymentsArgCount++
	}

	if region, ok := filters["region"].(string); ok && region != "" {
		regions := strings.Split(region, ",")
		if len(regions) == 1 {
			repaymentsWhere += fmt.Sprintf(" AND l.region = $%d", repaymentsArgCount)
			repaymentsArgs = append(repaymentsArgs, regions[0])
			repaymentsArgCount++
		} else {
			placeholders := []string{}
			for _, r := range regions {
				placeholders = append(placeholders, fmt.Sprintf("$%d", repaymentsArgCount))
				repaymentsArgs = append(repaymentsArgs, strings.TrimSpace(r))
				repaymentsArgCount++
			}
			repaymentsWhere += fmt.Sprintf(" AND l.region IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	if channel, ok := filters["channel"].(string); ok && channel != "" {
		repaymentsWhere += fmt.Sprintf(" AND l.channel = $%d", repaymentsArgCount)
		repaymentsArgs = append(repaymentsArgs, channel)
		repaymentsArgCount++
	}

	if status, ok := filters["status"].(string); ok && status != "" {
		statuses := strings.Split(status, ",")
		if len(statuses) == 1 {
			repaymentsWhere += fmt.Sprintf(" AND l.status = $%d", repaymentsArgCount)
			repaymentsArgs = append(repaymentsArgs, statuses[0])
			repaymentsArgCount++
		} else {
			placeholders := []string{}
			for _, s := range statuses {
				placeholders = append(placeholders, fmt.Sprintf("$%d", repaymentsArgCount))
				repaymentsArgs = append(repaymentsArgs, strings.TrimSpace(s))
				repaymentsArgCount++
			}
			repaymentsWhere += fmt.Sprintf(" AND l.status IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	// Raw Django status filter - supports comma-separated values and optional missing sentinel
	if djangoStatus, ok := filters["django_status"].(string); ok && djangoStatus != "" {
		statuses := strings.Split(djangoStatus, ",")
		nonMissing := []string{}
		includeMissing := false

		for _, s := range statuses {
			value := strings.TrimSpace(s)
			if value == "" {
				continue
			}
			if value == MissingValueSentinel {
				includeMissing = true
			} else {
				nonMissing = append(nonMissing, value)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.django_status = $%d", repaymentsArgCount))
			repaymentsArgs = append(repaymentsArgs, nonMissing[0])
			repaymentsArgCount++
		} else if len(nonMissing) > 1 {
			placeholders := make([]string, len(nonMissing))
			for i, s := range nonMissing {
				placeholders[i] = fmt.Sprintf("$%d", repaymentsArgCount)
				repaymentsArgs = append(repaymentsArgs, s)
				repaymentsArgCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.django_status IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.django_status IS NULL OR l.django_status = '')")
		}

		if len(conditions) > 0 {
			repaymentsWhere += " AND (" + strings.Join(conditions, " OR ") + ")"
		}
	}

	if performanceStatus, ok := filters["performance_status"].(string); ok && performanceStatus != "" {
		performanceStatuses := strings.Split(performanceStatus, ",")
		if len(performanceStatuses) == 1 {
			repaymentsWhere += fmt.Sprintf(" AND l.performance_status = $%d", repaymentsArgCount)
			repaymentsArgs = append(repaymentsArgs, performanceStatuses[0])
			repaymentsArgCount++
		} else {
			placeholders := []string{}
			for _, ps := range performanceStatuses {
				placeholders = append(placeholders, fmt.Sprintf("$%d", repaymentsArgCount))
				repaymentsArgs = append(repaymentsArgs, strings.TrimSpace(ps))
				repaymentsArgCount++
			}
			repaymentsWhere += fmt.Sprintf(" AND l.performance_status IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	if wave, ok := filters["wave"].(string); ok && wave != "" {
		repaymentsWhere += fmt.Sprintf(" AND l.wave = $%d", repaymentsArgCount)
		repaymentsArgs = append(repaymentsArgs, wave)
		repaymentsArgCount++
	}

	if customerPhone, ok := filters["customer_phone"].(string); ok && customerPhone != "" {
		repaymentsWhere += fmt.Sprintf(" AND l.customer_phone LIKE $%d", repaymentsArgCount)
		repaymentsArgs = append(repaymentsArgs, "%"+customerPhone+"%")
		repaymentsArgCount++
	}

	if verticalLeadEmail, ok := filters["vertical_lead_email"].(string); ok && verticalLeadEmail != "" {
		repaymentsWhere += fmt.Sprintf(" AND l.vertical_lead_email = $%d", repaymentsArgCount)
		repaymentsArgs = append(repaymentsArgs, verticalLeadEmail)
		repaymentsArgCount++
	}

	if loanType, ok := filters["loan_type"].(string); ok && loanType != "" {
		loanTypes := strings.Split(loanType, ",")
		nonMissing := []string{}
		includeMissing := false

		for _, lt := range loanTypes {
			value := strings.TrimSpace(lt)
			if value == "" {
				continue
			}
			if value == MissingValueSentinel {
				includeMissing = true
			} else {
				nonMissing = append(nonMissing, value)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.loan_type = $%d", repaymentsArgCount))
			repaymentsArgs = append(repaymentsArgs, nonMissing[0])
			repaymentsArgCount++
		} else if len(nonMissing) > 1 {
			placeholders := make([]string, len(nonMissing))
			for i, lt := range nonMissing {
				placeholders[i] = fmt.Sprintf("$%d", repaymentsArgCount)
				repaymentsArgs = append(repaymentsArgs, lt)
				repaymentsArgCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.loan_type IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.loan_type IS NULL OR l.loan_type = '')")
		}

		if len(conditions) > 0 {
			repaymentsWhere += " AND (" + strings.Join(conditions, " OR ") + ")"
		}
	}

	if verificationStatus, ok := filters["verification_status"].(string); ok && verificationStatus != "" {
		verificationStatuses := strings.Split(verificationStatus, ",")
		nonMissing := []string{}
		includeMissing := false

		for _, vs := range verificationStatuses {
			value := strings.TrimSpace(vs)
			if value == "" {
				continue
			}
			if value == MissingValueSentinel {
				includeMissing = true
			} else {
				nonMissing = append(nonMissing, value)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.verification_status = $%d", repaymentsArgCount))
			repaymentsArgs = append(repaymentsArgs, nonMissing[0])
			repaymentsArgCount++
		} else if len(nonMissing) > 1 {
			placeholders := make([]string, len(nonMissing))
			for i, vs := range nonMissing {
				placeholders[i] = fmt.Sprintf("$%d", repaymentsArgCount)
				repaymentsArgs = append(repaymentsArgs, vs)
				repaymentsArgCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.verification_status IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.verification_status IS NULL OR l.verification_status = '')")
		}

		if len(conditions) > 0 {
			repaymentsWhere += " AND (" + strings.Join(conditions, " OR ") + ")"
		}
	}

	if dpdMin, ok := filters["dpd_min"].(int); ok {
		repaymentsWhere += fmt.Sprintf(" AND l.current_dpd >= $%d", repaymentsArgCount)
		repaymentsArgs = append(repaymentsArgs, dpdMin)
		repaymentsArgCount++
	}

	if dpdMax, ok := filters["dpd_max"].(int); ok {
		repaymentsWhere += fmt.Sprintf(" AND l.current_dpd <= $%d", repaymentsArgCount)
		repaymentsArgs = append(repaymentsArgs, dpdMax)
		repaymentsArgCount++
	}

	// Overall total repayments in the period
	repaymentsTotalQuery := `
			SELECT COALESCE(SUM(r.payment_amount), 0) as total_repayments_today
		` + repaymentsWhere

	var totalRepaymentsToday float64
	err = r.db.QueryRow(repaymentsTotalQuery, repaymentsArgs...).Scan(&totalRepaymentsToday)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate today's repayments: %w", err)
	}

	// Breakdown of repayments by django_status using the same filters and period.
	repaymentsByStatusQuery := fmt.Sprintf(`
			SELECT
				COALESCE(NULLIF(l.django_status, ''), '%s') AS django_status,
				COALESCE(SUM(r.payment_amount), 0) AS amount
		%s
			GROUP BY COALESCE(NULLIF(l.django_status, ''), '%s')
			ORDER BY amount DESC
		`, MissingValueSentinel, repaymentsWhere, MissingValueSentinel)

	repaymentsByStatus := []map[string]interface{}{}
	rows, err := r.db.Query(repaymentsByStatusQuery, repaymentsArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate repayments by django_status: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status string
		var amount float64
		if scanErr := rows.Scan(&status, &amount); scanErr != nil {
			return nil, fmt.Errorf("failed to scan repayments by django_status row: %w", scanErr)
		}
		repaymentsByStatus = append(repaymentsByStatus, map[string]interface{}{
			"django_status": status,
			"amount":        amount,
		})
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate repayments by django_status rows: %w", err)
	}

	// Calculate missed repayments today: loans that have a scheduled daily repayment
	// today (same population as total_due_for_today) but have no repayment recorded
	// for the current day. This uses the same filters as the loans and repayments
	// summary so that amounts and counts stay aligned.
	missedQuery := `
			SELECT
				COALESCE(SUM(CASE WHEN l.actual_outstanding > 0 THEN l.daily_repayment_amount ELSE 0 END), 0) AS missed_amount_today,
				COUNT(*) AS missed_count_today
			FROM loans l
			INNER JOIN officers o ON l.officer_id = o.officer_id
			WHERE 1=1
				AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
				AND l.actual_outstanding > 0
				AND NOT EXISTS (
					SELECT 1
					FROM repayments r
					WHERE r.loan_id = l.loan_id
						AND r.is_reversed = false
						AND DATE(r.payment_date) = CURRENT_DATE
				)
		`

	missedArgs := []interface{}{}
	missedArgCount := 1

	if officerID, ok := filters["officer_id"].(string); ok && officerID != "" {
		missedQuery += fmt.Sprintf(" AND l.officer_id = $%d", missedArgCount)
		missedArgs = append(missedArgs, officerID)
		missedArgCount++
	}

	if branch, ok := filters["branch"].(string); ok && branch != "" {
		missedQuery += fmt.Sprintf(" AND l.branch = $%d", missedArgCount)
		missedArgs = append(missedArgs, branch)
		missedArgCount++
	}

	if region, ok := filters["region"].(string); ok && region != "" {
		regions := strings.Split(region, ",")
		if len(regions) == 1 {
			missedQuery += fmt.Sprintf(" AND l.region = $%d", missedArgCount)
			missedArgs = append(missedArgs, regions[0])
			missedArgCount++
		} else {
			placeholders := []string{}
			for _, r := range regions {
				placeholders = append(placeholders, fmt.Sprintf("$%d", missedArgCount))
				missedArgs = append(missedArgs, strings.TrimSpace(r))
				missedArgCount++
			}
			missedQuery += fmt.Sprintf(" AND l.region IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	if channel, ok := filters["channel"].(string); ok && channel != "" {
		missedQuery += fmt.Sprintf(" AND l.channel = $%d", missedArgCount)
		missedArgs = append(missedArgs, channel)
		missedArgCount++
	}

	if status, ok := filters["status"].(string); ok && status != "" {
		statuses := strings.Split(status, ",")
		if len(statuses) == 1 {
			missedQuery += fmt.Sprintf(" AND l.status = $%d", missedArgCount)
			missedArgs = append(missedArgs, statuses[0])
			missedArgCount++
		} else {
			placeholders := []string{}
			for _, s := range statuses {
				placeholders = append(placeholders, fmt.Sprintf("$%d", missedArgCount))
				missedArgs = append(missedArgs, strings.TrimSpace(s))
				missedArgCount++
			}
			missedQuery += fmt.Sprintf(" AND l.status IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	// Raw Django status filter - supports comma-separated values and optional missing sentinel
	if djangoStatus, ok := filters["django_status"].(string); ok && djangoStatus != "" {
		statuses := strings.Split(djangoStatus, ",")
		nonMissing := []string{}
		includeMissing := false

		for _, s := range statuses {
			value := strings.TrimSpace(s)
			if value == "" {
				continue
			}
			if value == MissingValueSentinel {
				includeMissing = true
			} else {
				nonMissing = append(nonMissing, value)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.django_status = $%d", missedArgCount))
			missedArgs = append(missedArgs, nonMissing[0])
			missedArgCount++
		} else if len(nonMissing) > 1 {
			placeholders := make([]string, len(nonMissing))
			for i, s := range nonMissing {
				placeholders[i] = fmt.Sprintf("$%d", missedArgCount)
				missedArgs = append(missedArgs, s)
				missedArgCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.django_status IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.django_status IS NULL OR l.django_status = '')")
		}

		if len(conditions) > 0 {
			missedQuery += " AND (" + strings.Join(conditions, " OR ") + ")"
		}
	}

	if performanceStatus, ok := filters["performance_status"].(string); ok && performanceStatus != "" {
		performanceStatuses := strings.Split(performanceStatus, ",")
		if len(performanceStatuses) == 1 {
			missedQuery += fmt.Sprintf(" AND l.performance_status = $%d", missedArgCount)
			missedArgs = append(missedArgs, performanceStatuses[0])
			missedArgCount++
		} else {
			placeholders := []string{}
			for _, ps := range performanceStatuses {
				placeholders = append(placeholders, fmt.Sprintf("$%d", missedArgCount))
				missedArgs = append(missedArgs, strings.TrimSpace(ps))
				missedArgCount++
			}
			missedQuery += fmt.Sprintf(" AND l.performance_status IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	if wave, ok := filters["wave"].(string); ok && wave != "" {
		missedQuery += fmt.Sprintf(" AND l.wave = $%d", missedArgCount)
		missedArgs = append(missedArgs, wave)
		missedArgCount++
	}

	if customerPhone, ok := filters["customer_phone"].(string); ok && customerPhone != "" {
		missedQuery += fmt.Sprintf(" AND l.customer_phone LIKE $%d", missedArgCount)
		missedArgs = append(missedArgs, "%"+customerPhone+"%")
		missedArgCount++
	}

	if verticalLeadEmail, ok := filters["vertical_lead_email"].(string); ok && verticalLeadEmail != "" {
		missedQuery += fmt.Sprintf(" AND l.vertical_lead_email = $%d", missedArgCount)
		missedArgs = append(missedArgs, verticalLeadEmail)
		missedArgCount++
	}

	if loanType, ok := filters["loan_type"].(string); ok && loanType != "" {
		loanTypes := strings.Split(loanType, ",")
		nonMissing := []string{}
		includeMissing := false

		for _, lt := range loanTypes {
			value := strings.TrimSpace(lt)
			if value == "" {
				continue
			}
			if value == MissingValueSentinel {
				includeMissing = true
			} else {
				nonMissing = append(nonMissing, value)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.loan_type = $%d", missedArgCount))
			missedArgs = append(missedArgs, nonMissing[0])
			missedArgCount++
		} else if len(nonMissing) > 1 {
			placeholders := make([]string, len(nonMissing))
			for i, lt := range nonMissing {
				placeholders[i] = fmt.Sprintf("$%d", missedArgCount)
				missedArgs = append(missedArgs, lt)
				missedArgCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.loan_type IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.loan_type IS NULL OR l.loan_type = '')")
		}

		if len(conditions) > 0 {
			missedQuery += " AND (" + strings.Join(conditions, " OR ") + ")"
		}
	}

	if verificationStatus, ok := filters["verification_status"].(string); ok && verificationStatus != "" {
		verificationStatuses := strings.Split(verificationStatus, ",")
		nonMissing := []string{}
		includeMissing := false

		for _, vs := range verificationStatuses {
			value := strings.TrimSpace(vs)
			if value == "" {
				continue
			}
			if value == MissingValueSentinel {
				includeMissing = true
			} else {
				nonMissing = append(nonMissing, value)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.verification_status = $%d", missedArgCount))
			missedArgs = append(missedArgs, nonMissing[0])
			missedArgCount++
		} else if len(nonMissing) > 1 {
			placeholders := make([]string, len(nonMissing))
			for i, vs := range nonMissing {
				placeholders[i] = fmt.Sprintf("$%d", missedArgCount)
				missedArgs = append(missedArgs, vs)
				missedArgCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.verification_status IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.verification_status IS NULL OR l.verification_status = '')")
		}

		if len(conditions) > 0 {
			missedQuery += " AND (" + strings.Join(conditions, " OR ") + ")"
		}
	}

	if dpdMin, ok := filters["dpd_min"].(int); ok {
		missedQuery += fmt.Sprintf(" AND l.current_dpd >= $%d", missedArgCount)
		missedArgs = append(missedArgs, dpdMin)
		missedArgCount++
	}

	if dpdMax, ok := filters["dpd_max"].(int); ok {
		missedQuery += fmt.Sprintf(" AND l.current_dpd <= $%d", missedArgCount)
		missedArgs = append(missedArgs, dpdMax)
		missedArgCount++
	}

	var missedAmountToday float64
	var missedCountToday int
	err = r.db.QueryRow(missedQuery, missedArgs...).Scan(&missedAmountToday, &missedCountToday)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate missed repayments today: %w", err)
	}

	// Calculate percentages
	atRiskPercentage := 0.0
	if totalLoans > 0 {
		atRiskPercentage = (float64(atRiskCount) / float64(totalLoans)) * 100
	}

	criticalPercentage := 0.0
	if totalLoans > 0 {
		criticalPercentage = (float64(criticalCount) / float64(totalLoans)) * 100
	}

	// Calculate percentage of due collected
	percentageDueCollected := 0.0
	if totalDueForToday > 0 {
		percentageDueCollected = (totalRepaymentsToday / totalDueForToday) * 100
	}

	// Build response
	metrics := map[string]interface{}{
		"total_loans":            totalLoans,
		"total_portfolio_amount": totalPortfolioAmount,
		"at_risk_loans": map[string]interface{}{
			"count":              atRiskCount,
			"amount":             atRiskAmount,
			"actual_outstanding": atRiskOutstanding,
			"percentage":         atRiskPercentage,
		},
		"portfolio_health": map[string]interface{}{
			"performing_loans_count":        performingLoansCount,
			"performing_actual_outstanding": performingActualOutstanding,
		},
		"total_amount_in_dpd": totalAmountInDPD,
		"critical_loans": map[string]interface{}{
			"count":      criticalCount,
			"percentage": criticalPercentage,
		},
		"repayment_delay_categories": map[string]interface{}{
			"excellent": excellentDelayCount,
			"okay":      okayDelayCount,
			"critical":  criticalDelayCount,
		},
		"repayments_by_django_status":   repaymentsByStatus,
		"total_due_for_today":           totalDueForToday,
		"total_repayments_today":        totalRepaymentsToday,
		"percentage_of_due_collected":   percentageDueCollected,
		"missed_repayments_today":       missedAmountToday,
		"missed_repayments_today_count": missedCountToday,
		"past_maturity_outstanding":     pastMaturityOutstanding,
	}

	return metrics, nil
}

// GetAllLoans retrieves all loans with pagination and filters
func (r *DashboardRepository) GetAllLoans(filters map[string]interface{}) ([]*models.AllLoan, int, error) {
	// Base query
	query := `
		SELECT
			l.loan_id,
			l.customer_name,
			l.customer_phone,
			l.officer_id,
			o.officer_name as officer_name,
			l.region,
			l.branch,
			l.vertical_lead_name,
			l.vertical_lead_email,
			l.channel,
			l.loan_amount,
			l.repayment_amount,
			TO_CHAR(l.disbursement_date, 'YYYY-MM-DD') as disbursement_date,
			TO_CHAR(l.first_payment_due_date, 'YYYY-MM-DD') as first_payment_due_date,
			TO_CHAR(l.maturity_date, 'YYYY-MM-DD') as maturity_date,
			l.loan_term_days,
			l.current_dpd,
			l.principal_outstanding,
			l.interest_outstanding,
			l.fees_outstanding,
			l.total_outstanding,
			l.actual_outstanding,
				l.total_repayments,
				l.status,
				l.django_status,
				l.performance_status,
			l.fimr_tagged,
			l.timeliness_score,
			l.repayment_health,
			l.days_since_last_repayment,
			l.repayment_delay_rate,
			l.wave,
			l.daily_repayment_amount,
			l.repayment_days_due_today,
			l.repayment_days_paid,
			l.business_days_since_disbursement,
			l.loan_type,
			l.verification_status
		FROM loans l
		JOIN officers o ON l.officer_id = o.officer_id
		WHERE 1=1
			AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
	`

	countQuery := `
		SELECT COUNT(*)
		FROM loans l
		JOIN officers o ON l.officer_id = o.officer_id
		WHERE 1=1
			AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
	`

	args := []interface{}{}
	argCount := 1

	// Apply filters
	if officerID, ok := filters["officer_id"].(string); ok && officerID != "" {
		query += fmt.Sprintf(" AND l.officer_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND l.officer_id = $%d", argCount)
		args = append(args, officerID)
		argCount++
	}

	if branch, ok := filters["branch"].(string); ok && branch != "" {
		query += fmt.Sprintf(" AND l.branch = $%d", argCount)
		countQuery += fmt.Sprintf(" AND l.branch = $%d", argCount)
		args = append(args, branch)
		argCount++
	}

	if region, ok := filters["region"].(string); ok && region != "" {
		// Support comma-separated regions for multi-select
		regions := strings.Split(region, ",")
		if len(regions) == 1 {
			query += fmt.Sprintf(" AND l.region = $%d", argCount)
			countQuery += fmt.Sprintf(" AND l.region = $%d", argCount)
			args = append(args, regions[0])
			argCount++
		} else {
			// Build IN clause for multiple regions
			placeholders := []string{}
			for _, r := range regions {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
				args = append(args, strings.TrimSpace(r))
				argCount++
			}
			inClause := fmt.Sprintf(" AND l.region IN (%s)", strings.Join(placeholders, ", "))
			query += inClause
			countQuery += inClause
		}
	}

	if channel, ok := filters["channel"].(string); ok && channel != "" {
		query += fmt.Sprintf(" AND l.channel = $%d", argCount)
		countQuery += fmt.Sprintf(" AND l.channel = $%d", argCount)
		args = append(args, channel)
		argCount++
	}

	if status, ok := filters["status"].(string); ok && status != "" {
		// Support comma-separated statuses for multi-select
		statuses := strings.Split(status, ",")
		if len(statuses) == 1 {
			query += fmt.Sprintf(" AND l.status = $%d", argCount)
			countQuery += fmt.Sprintf(" AND l.status = $%d", argCount)
			args = append(args, statuses[0])
			argCount++
		} else {
			// Build IN clause for multiple statuses
			placeholders := []string{}
			for _, s := range statuses {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
				args = append(args, strings.TrimSpace(s))
				argCount++
			}
			inClause := fmt.Sprintf(" AND l.status IN (%s)", strings.Join(placeholders, ", "))
			query += inClause
			countQuery += inClause
		}
	}

	// Raw Django status filter - supports comma-separated values and optional missing sentinel
	if djangoStatus, ok := filters["django_status"].(string); ok && djangoStatus != "" {
		statuses := strings.Split(djangoStatus, ",")
		nonMissing := []string{}
		includeMissing := false

		for _, s := range statuses {
			value := strings.TrimSpace(s)
			if value == "" {
				continue
			}
			if value == MissingValueSentinel {
				includeMissing = true
			} else {
				nonMissing = append(nonMissing, value)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.django_status = $%d", argCount))
			args = append(args, nonMissing[0])
			argCount++
		} else if len(nonMissing) > 1 {
			placeholders := make([]string, len(nonMissing))
			for i, s := range nonMissing {
				placeholders[i] = fmt.Sprintf("$%d", argCount)
				args = append(args, s)
				argCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.django_status IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.django_status IS NULL OR l.django_status = '')")
		}

		if len(conditions) > 0 {
			clause := " AND (" + strings.Join(conditions, " OR ") + ")"
			query += clause
			countQuery += clause
		}
	}

	if performanceStatus, ok := filters["performance_status"].(string); ok && performanceStatus != "" {
		// Support comma-separated performance statuses for multi-select, including a sentinel for missing values
		performanceStatuses := strings.Split(performanceStatus, ",")
		nonMissing := []string{}
		includeMissing := false

		for _, ps := range performanceStatuses {
			value := strings.TrimSpace(ps)
			if value == "" {
				continue
			}
			if value == MissingValueSentinel {
				includeMissing = true
			} else {
				nonMissing = append(nonMissing, value)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.performance_status = $%d", argCount))
			args = append(args, nonMissing[0])
			argCount++
		} else if len(nonMissing) > 1 {
			placeholders := make([]string, len(nonMissing))
			for i, ps := range nonMissing {
				placeholders[i] = fmt.Sprintf("$%d", argCount)
				args = append(args, ps)
				argCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.performance_status IN (%s)", strings.Join(placeholders, ", ")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.performance_status IS NULL OR l.performance_status = '')")
		}

		if len(conditions) > 0 {
			clause := " AND (" + strings.Join(conditions, " OR ") + ")"
			query += clause
			countQuery += clause
		}
	}

	if wave, ok := filters["wave"].(string); ok && wave != "" {
		query += fmt.Sprintf(" AND l.wave = $%d", argCount)
		countQuery += fmt.Sprintf(" AND l.wave = $%d", argCount)
		args = append(args, wave)
		argCount++
	}

	if customerPhone, ok := filters["customer_phone"].(string); ok && customerPhone != "" {
		query += fmt.Sprintf(" AND l.customer_phone LIKE $%d", argCount)
		countQuery += fmt.Sprintf(" AND l.customer_phone LIKE $%d", argCount)
		args = append(args, "%"+customerPhone+"%")
		argCount++
	}

	// Vertical lead filter - support comma-separated values for multi-select
	if verticalLeadEmail, ok := filters["vertical_lead_email"].(string); ok && verticalLeadEmail != "" {
		emails := strings.Split(verticalLeadEmail, ",")
		if len(emails) == 1 {
			query += fmt.Sprintf(" AND l.vertical_lead_email = $%d", argCount)
			countQuery += fmt.Sprintf(" AND l.vertical_lead_email = $%d", argCount)
			args = append(args, strings.TrimSpace(emails[0]))
			argCount++
		} else {
			placeholders := []string{}
			for _, e := range emails {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
				args = append(args, strings.TrimSpace(e))
				argCount++
			}
			inClause := fmt.Sprintf(" AND l.vertical_lead_email IN (%s)", strings.Join(placeholders, ", "))
			query += inClause
			countQuery += inClause
		}
	}

	// Loan type filter - support comma-separated values for multiple loan types, including a sentinel for missing values
	if loanType, ok := filters["loan_type"].(string); ok && loanType != "" {
		loanTypes := strings.Split(loanType, ",")
		fmt.Printf("DEBUG GetAllLoans: loan_type filter - raw input: '%s', split into %d values: %v\n", loanType, len(loanTypes), loanTypes)

		nonMissing := []string{}
		includeMissing := false

		for _, lt := range loanTypes {
			value := strings.TrimSpace(lt)
			if value == "" {
				continue
			}
			if value == MissingValueSentinel {
				includeMissing = true
			} else {
				nonMissing = append(nonMissing, value)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.loan_type = $%d", argCount))
			args = append(args, nonMissing[0])
			argCount++
		} else if len(nonMissing) > 1 {
			placeholders := make([]string, len(nonMissing))
			for i, lt := range nonMissing {
				placeholders[i] = fmt.Sprintf("$%d", argCount)
				args = append(args, lt)
				argCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.loan_type IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.loan_type IS NULL OR l.loan_type = '')")
		}

		if len(conditions) > 0 {
			clause := " AND (" + strings.Join(conditions, " OR ") + ")"
			fmt.Printf("DEBUG GetAllLoans: loan_type WHERE clause: %s, total args: %d\n", clause, len(args))
			query += clause
			countQuery += clause
		}
	}

	// Verification status filter - support comma-separated values for multiple verification statuses, including a sentinel for missing values
	if verificationStatus, ok := filters["verification_status"].(string); ok && verificationStatus != "" {
		verificationStatuses := strings.Split(verificationStatus, ",")
		nonMissing := []string{}
		includeMissing := false

		for _, vs := range verificationStatuses {
			value := strings.TrimSpace(vs)
			if value == "" {
				continue
			}
			if value == MissingValueSentinel {
				includeMissing = true
			} else {
				nonMissing = append(nonMissing, value)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.verification_status = $%d", argCount))
			args = append(args, nonMissing[0])
			argCount++
		} else if len(nonMissing) > 1 {
			placeholders := make([]string, len(nonMissing))
			for i, vs := range nonMissing {
				placeholders[i] = fmt.Sprintf("$%d", argCount)
				args = append(args, vs)
				argCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.verification_status IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.verification_status IS NULL OR l.verification_status = '')")
		}

		if len(conditions) > 0 {
			clause := " AND (" + strings.Join(conditions, " OR ") + ")"
			query += clause
			countQuery += clause
		}
	}

	// DPD range filter
	if dpdMin, ok := filters["dpd_min"].(int); ok {
		query += fmt.Sprintf(" AND l.current_dpd >= $%d", argCount)
		countQuery += fmt.Sprintf(" AND l.current_dpd >= $%d", argCount)
		args = append(args, dpdMin)
		argCount++
	}

	if dpdMax, ok := filters["dpd_max"].(int); ok {
		query += fmt.Sprintf(" AND l.current_dpd <= $%d", argCount)
		countQuery += fmt.Sprintf(" AND l.current_dpd <= $%d", argCount)
		args = append(args, dpdMax)
		argCount++
	}

	// Behavior-based filters that were previously applied only on the frontend
	// so that dashboard totals and CSV exports now use identical logic.
	if behaviorLoanType, ok := filters["behavior_loan_type"].(string); ok && behaviorLoanType != "" {
		switch behaviorLoanType {
		case "active":
			// Active: significant outstanding and recent repayment
			query += " AND l.total_outstanding > 2000 AND COALESCE(l.days_since_last_repayment, 0) < 6"
			countQuery += " AND l.total_outstanding > 2000 AND COALESCE(l.days_since_last_repayment, 0) < 6"
		case "inactive":
			// Inactive: low outstanding or no recent repayment
			query += " AND (l.total_outstanding <= 2000 OR COALESCE(l.days_since_last_repayment, 0) > 5)"
			countQuery += " AND (l.total_outstanding <= 2000 OR COALESCE(l.days_since_last_repayment, 0) > 5)"
		case "overdue_15d":
			// Overdue: DPD strictly greater than 15 days
			query += " AND l.current_dpd > 15"
			countQuery += " AND l.current_dpd > 15"
		}
	}

	if rotType, ok := filters["rot_type"].(string); ok && rotType != "" {
		switch rotType {
		case "early":
			// Early ROT: young loan with emerging DPD
			query += " AND (CURRENT_DATE - l.disbursement_date::date) < 7 AND l.current_dpd > 4"
			countQuery += " AND (CURRENT_DATE - l.disbursement_date::date) < 7 AND l.current_dpd > 4"
		case "late":
			// Late ROT: older loan with DPD
			query += " AND (CURRENT_DATE - l.disbursement_date::date) >= 7 AND l.current_dpd > 4"
			countQuery += " AND (CURRENT_DATE - l.disbursement_date::date) >= 7 AND l.current_dpd > 4"
		}
	}

	if delayType, ok := filters["delay_type"].(string); ok && delayType != "" {
		// Risky loans based on repayment delay rate
		if delayType == "risky" {
			query += " AND l.status = 'Active' AND l.total_outstanding > 2000 AND l.repayment_delay_rate IS NOT NULL AND l.repayment_delay_rate < 60"
			countQuery += " AND l.status = 'Active' AND l.total_outstanding > 2000 AND l.repayment_delay_rate IS NOT NULL AND l.repayment_delay_rate < 60"
		}
	}

	// Get total count
	var total int
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Apply sorting
	sortBy := "l.disbursement_date"
	if sort, ok := filters["sort_by"].(string); ok && sort != "" {
		sortBy = "l." + sort
	}
	sortDir := "DESC"
	if dir, ok := filters["sort_dir"].(string); ok && dir != "" {
		sortDir = dir
	}
	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortDir)

	// Apply pagination
	page := 1
	limit := 50
	if p, ok := filters["page"].(int); ok && p > 0 {
		page = p
	}
	if l, ok := filters["limit"].(int); ok && l > 0 {
		limit = l
	}
	offset := (page - 1) * limit

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args = append(args, limit, offset)

	// Execute query
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	loans := []*models.AllLoan{}
	for rows.Next() {
		loan := &models.AllLoan{}
		var customerPhone, officerID, firstPaymentDueDate, maturityDate sql.NullString
		var verticalLeadName, verticalLeadEmail, performanceStatus sql.NullString
		var loanType, verificationStatus, djangoStatus sql.NullString
		var repaymentAmount, timelinessScore, repaymentHealth, repaymentDelayRate sql.NullFloat64
		var dailyRepaymentAmount, repaymentDaysPaid sql.NullFloat64
		var daysSinceLastRepayment, repaymentDaysDueToday, businessDaysSinceDisbursement sql.NullInt64

		err := rows.Scan(
			&loan.LoanID,
			&loan.CustomerName,
			&customerPhone,
			&officerID,
			&loan.OfficerName,
			&loan.Region,
			&loan.Branch,
			&verticalLeadName,
			&verticalLeadEmail,
			&loan.Channel,
			&loan.LoanAmount,
			&repaymentAmount,
			&loan.DisbursementDate,
			&firstPaymentDueDate,
			&maturityDate,
			&loan.LoanTermDays,
			&loan.CurrentDPD,
			&loan.PrincipalOutstanding,
			&loan.InterestOutstanding,
			&loan.FeesOutstanding,
			&loan.TotalOutstanding,
			&loan.ActualOutstanding,
			&loan.TotalRepayments,
			&loan.Status,
			&djangoStatus,
			&performanceStatus,
			&loan.FIMRTagged,
			&timelinessScore,
			&repaymentHealth,
			&daysSinceLastRepayment,
			&repaymentDelayRate,
			&loan.Wave,
			&dailyRepaymentAmount,
			&repaymentDaysDueToday,
			&repaymentDaysPaid,
			&businessDaysSinceDisbursement,
			&loanType,
			&verificationStatus,
		)
		if err != nil {
			return nil, 0, err
		}

		if customerPhone.Valid {
			loan.CustomerPhone = customerPhone.String
		}
		if officerID.Valid {
			loan.OfficerID = officerID.String
		}
		if verticalLeadName.Valid {
			loan.VerticalLeadName = &verticalLeadName.String
		}
		if verticalLeadEmail.Valid {
			loan.VerticalLeadEmail = &verticalLeadEmail.String
		}
		if performanceStatus.Valid {
			loan.PerformanceStatus = &performanceStatus.String
		}
		if djangoStatus.Valid {
			loan.DjangoStatus = &djangoStatus.String
		}
		if loanType.Valid {
			loan.LoanType = &loanType.String
		}
		if verificationStatus.Valid {
			loan.VerificationStatus = &verificationStatus.String
		}
		if repaymentAmount.Valid {
			val := repaymentAmount.Float64
			loan.RepaymentAmount = &val
		}
		if firstPaymentDueDate.Valid {
			loan.FirstPaymentDueDate = &firstPaymentDueDate.String
		}
		if maturityDate.Valid {
			loan.MaturityDate = maturityDate.String
		}
		if timelinessScore.Valid {
			val := timelinessScore.Float64
			loan.TimelinessScore = &val
		}
		if repaymentHealth.Valid {
			val := repaymentHealth.Float64
			loan.RepaymentHealth = &val
		}
		if daysSinceLastRepayment.Valid {
			val := int(daysSinceLastRepayment.Int64)
			loan.DaysSinceLastRepayment = &val
		}
		if repaymentDelayRate.Valid {
			val := repaymentDelayRate.Float64
			loan.RepaymentDelayRate = &val
		}
		if dailyRepaymentAmount.Valid {
			val := dailyRepaymentAmount.Float64
			loan.DailyRepaymentAmount = &val
		}
		if repaymentDaysDueToday.Valid {
			val := int(repaymentDaysDueToday.Int64)
			loan.RepaymentDaysDueToday = &val
		}
		if repaymentDaysPaid.Valid {
			val := repaymentDaysPaid.Float64
			loan.RepaymentDaysPaid = &val
		}
		if businessDaysSinceDisbursement.Valid {
			val := int(businessDaysSinceDisbursement.Int64)
			loan.BusinessDaysSinceDisbursement = &val
		}

		loans = append(loans, loan)
	}

	return loans, total, nil
}

// GetTopRiskLoans retrieves the top N highest-risk loans for a specific officer
func (r *DashboardRepository) GetTopRiskLoans(officerID string, limit int) ([]*models.TopRiskLoan, error) {
	query := `
		SELECT
			l.loan_id,
			l.customer_name,
			COALESCE(l.customer_phone, '') as customer_phone,
			l.loan_amount::float as loan_amount,
			TO_CHAR(l.disbursement_date, 'YYYY-MM-DD') as disbursement_date,
			l.current_dpd,
			l.max_dpd_ever,
			l.total_outstanding::float as total_outstanding,
			l.principal_outstanding::float as principal_outstanding,
			l.interest_outstanding::float as interest_outstanding,
			l.fees_outstanding::float as fees_outstanding,
			l.status,
			l.fimr_tagged,
			l.channel,
			(CURRENT_DATE - l.disbursement_date::date)::int as days_since_disbursement,
			-- Calculate risk score based on multiple factors
			(
				-- DPD weight (40%): Higher DPD = higher risk
				(CASE
					WHEN l.current_dpd >= 90 THEN 40
					WHEN l.current_dpd >= 60 THEN 35
					WHEN l.current_dpd >= 30 THEN 30
					WHEN l.current_dpd >= 15 THEN 25
					WHEN l.current_dpd >= 7 THEN 15
					WHEN l.current_dpd >= 1 THEN 10
					ELSE 0
				END) +
				-- Outstanding balance weight (30%): Higher balance = higher risk
				(CASE
					WHEN l.total_outstanding >= 5000000 THEN 30
					WHEN l.total_outstanding >= 3000000 THEN 25
					WHEN l.total_outstanding >= 2000000 THEN 20
					WHEN l.total_outstanding >= 1000000 THEN 15
					WHEN l.total_outstanding >= 500000 THEN 10
					ELSE 5
				END) +
				-- FIMR weight (15%): FIMR tagged = higher risk
				(CASE WHEN l.fimr_tagged THEN 15 ELSE 0 END) +
				-- Max DPD Ever weight (10%): Historical delinquency
				(CASE
					WHEN l.max_dpd_ever >= 90 THEN 10
					WHEN l.max_dpd_ever >= 60 THEN 8
					WHEN l.max_dpd_ever >= 30 THEN 6
					WHEN l.max_dpd_ever >= 15 THEN 4
					ELSE 0
				END) +
				-- Loan age weight (5%): Newer loans with issues = higher risk
				(CASE
					WHEN (CURRENT_DATE - l.disbursement_date::date) <= 30 AND l.current_dpd > 0 THEN 5
					WHEN (CURRENT_DATE - l.disbursement_date::date) <= 60 AND l.current_dpd > 0 THEN 3
					ELSE 0
				END)
			) as risk_score
		FROM loans l
		WHERE l.officer_id = $1
			AND l.status = 'Active'
			AND (l.current_dpd > 0 OR l.fimr_tagged = true OR l.total_outstanding > 0)
		ORDER BY risk_score DESC, l.current_dpd DESC, total_outstanding DESC
		LIMIT $2
	`

	rows, err := r.db.Query(query, officerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var loans []*models.TopRiskLoan
	for rows.Next() {
		var loan models.TopRiskLoan
		err := rows.Scan(
			&loan.LoanID,
			&loan.CustomerName,
			&loan.CustomerPhone,
			&loan.LoanAmount,
			&loan.DisbursementDate,
			&loan.CurrentDPD,
			&loan.MaxDPDEver,
			&loan.TotalOutstanding,
			&loan.PrincipalOutstanding,
			&loan.InterestOutstanding,
			&loan.FeesOutstanding,
			&loan.Status,
			&loan.FIMRTagged,
			&loan.Channel,
			&loan.DaysSinceDisbursement,
			&loan.RiskScore,
		)
		if err != nil {
			return nil, err
		}

		// Determine risk category based on risk score
		if loan.RiskScore >= 80 {
			loan.RiskCategory = "Critical"
		} else if loan.RiskScore >= 60 {
			loan.RiskCategory = "High"
		} else if loan.RiskScore >= 40 {
			loan.RiskCategory = "Medium"
		} else {
			loan.RiskCategory = "Low"
		}

		loans = append(loans, &loan)
	}

	return loans, nil
}

// GetBranches retrieves branch-level aggregated metrics
func (r *DashboardRepository) GetBranches(filters map[string]interface{}) ([]*models.DashboardBranchMetrics, error) {
	query := `
		SELECT
			l.branch,
			l.region,
			COALESCE(SUM(l.principal_outstanding), 0) as portfolio_total,
			COALESCE(SUM(CASE WHEN l.current_dpd >= 15 THEN l.principal_outstanding ELSE 0 END), 0) as overdue_15d,
			CASE
				WHEN SUM(l.principal_outstanding) > 0
				THEN SUM(CASE WHEN l.current_dpd >= 15 THEN l.principal_outstanding ELSE 0 END) / SUM(l.principal_outstanding)
				ELSE 0
			END as par15_ratio,
			COUNT(DISTINCT l.loan_id) as active_loans,
			COUNT(DISTINCT l.officer_id) as total_officers,
			COALESCE(AVG(l.repayment_delay_rate), 0) as avg_repayment_delay_rate
		FROM loans l
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 1

	// Apply filters
	if branch, ok := filters["branch"].(string); ok && branch != "" {
		query += fmt.Sprintf(" AND l.branch = $%d", argCount)
		args = append(args, branch)
		argCount++
	}

	if region, ok := filters["region"].(string); ok && region != "" {
		// Support comma-separated regions for multi-select
		regions := strings.Split(region, ",")
		if len(regions) == 1 {
			query += fmt.Sprintf(" AND l.region = $%d", argCount)
			args = append(args, regions[0])
			argCount++
		} else {
			// Build IN clause for multiple regions
			placeholders := []string{}
			for _, r := range regions {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
				args = append(args, strings.TrimSpace(r))
				argCount++
			}
			query += fmt.Sprintf(" AND l.region IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	if channel, ok := filters["channel"].(string); ok && channel != "" {
		query += fmt.Sprintf(" AND l.channel = $%d", argCount)
		args = append(args, channel)
		argCount++
	}

	if userType, ok := filters["user_type"].(string); ok && userType != "" {
		query += fmt.Sprintf(" AND l.user_type = $%d", argCount)
		args = append(args, userType)
		argCount++
	}

	if wave, ok := filters["wave"].(string); ok && wave != "" {
		query += fmt.Sprintf(" AND l.wave = $%d", argCount)
		args = append(args, wave)
		argCount++
	}

	query += " GROUP BY l.branch, l.region"

	// Apply sorting
	sortBy := "l.branch"
	if sort, ok := filters["sort_by"].(string); ok && sort != "" {
		sortBy = sort
	}
	sortDir := "ASC"
	if dir, ok := filters["sort_dir"].(string); ok && strings.ToUpper(dir) == "DESC" {
		sortDir = "DESC"
	}
	query += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortDir)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	branches := []*models.DashboardBranchMetrics{}
	for rows.Next() {
		branch := &models.DashboardBranchMetrics{}
		err := rows.Scan(
			&branch.Branch,
			&branch.Region,
			&branch.PortfolioTotal,
			&branch.Overdue15d,
			&branch.Par15Ratio,
			&branch.ActiveLoans,
			&branch.TotalOfficers,
			&branch.AvgRepaymentDelayRate,
		)
		if err != nil {
			return nil, err
		}

		// Calculate AYR, DQI, FIMR for branch (simplified - would need more complex query)
		branch.AYR = 0.0
		branch.DQI = 0
		branch.FIMR = 0.0

		branches = append(branches, branch)
	}

	return branches, nil
}

// GetBranchCollectionsLeaderboard returns per-branch collections metrics for the
// Collections Control Centre "Branch Leaderboard" table. It focuses on
// "today" collections behaviour (expected due today vs collected today) and
// a simple NPL proxy based on PAR15 (overdue >= 15 days / portfolio).
func (r *DashboardRepository) GetBranchCollectionsLeaderboard(filters map[string]interface{}) ([]*models.BranchCollectionsLeaderboardRow, error) {
	// --- First query: loan-based metrics per branch (portfolio, due today, PAR15) ---
	// NOTE: Group by branch only. Use MODE() to get the most common region for display.
	loanQuery := `
		SELECT
			l.branch,
			MODE() WITHIN GROUP (ORDER BY l.region) AS region,
			COALESCE(SUM(l.repayment_amount), 0) AS portfolio_total,
			COALESCE(SUM(CASE WHEN l.actual_outstanding > 0 THEN l.daily_repayment_amount ELSE 0 END), 0) AS due_today,
			COALESCE(SUM(CASE WHEN l.current_dpd >= 15 THEN l.principal_outstanding ELSE 0 END), 0) AS overdue_15d
		FROM loans l
		JOIN officers o ON l.officer_id = o.officer_id
		WHERE 1=1
			AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
	`

	loanArgs := []interface{}{}
	loanArgCount := 1

	// Apply filters (same semantics as Collections Control Centre where relevant).
	if branch, ok := filters["branch"].(string); ok && branch != "" {
		loanQuery += fmt.Sprintf(" AND l.branch = $%d", loanArgCount)
		loanArgs = append(loanArgs, branch)
		loanArgCount++
	}

	if region, ok := filters["region"].(string); ok && region != "" {
		regions := strings.Split(region, ",")
		if len(regions) == 1 {
			loanQuery += fmt.Sprintf(" AND l.region = $%d", loanArgCount)
			loanArgs = append(loanArgs, regions[0])
			loanArgCount++
		} else {
			placeholders := []string{}
			for _, rgn := range regions {
				placeholders = append(placeholders, fmt.Sprintf("$%d", loanArgCount))
				loanArgs = append(loanArgs, strings.TrimSpace(rgn))
				loanArgCount++
			}
			loanQuery += fmt.Sprintf(" AND l.region IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	if channel, ok := filters["channel"].(string); ok && channel != "" {
		loanQuery += fmt.Sprintf(" AND l.channel = $%d", loanArgCount)
		loanArgs = append(loanArgs, channel)
		loanArgCount++
	}

	if wave, ok := filters["wave"].(string); ok && wave != "" {
		loanQuery += fmt.Sprintf(" AND l.wave = $%d", loanArgCount)
		loanArgs = append(loanArgs, wave)
		loanArgCount++
	}

	if loanType, ok := filters["loan_type"].(string); ok && loanType != "" {
		// Support comma-separated loan types for multi-select
		loanTypes := strings.Split(loanType, ",")
		if len(loanTypes) == 1 {
			loanQuery += fmt.Sprintf(" AND l.loan_type = $%d", loanArgCount)
			loanArgs = append(loanArgs, strings.TrimSpace(loanTypes[0]))
			loanArgCount++
		} else {
			placeholders := make([]string, len(loanTypes))
			for i, lt := range loanTypes {
				placeholders[i] = fmt.Sprintf("$%d", loanArgCount)
				loanArgs = append(loanArgs, strings.TrimSpace(lt))
				loanArgCount++
			}
			loanQuery += fmt.Sprintf(" AND l.loan_type IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	// Django status filter - supports comma-separated values and optional missing sentinel
	if djangoStatus, ok := filters["django_status"].(string); ok && djangoStatus != "" {
		statuses := strings.Split(djangoStatus, ",")
		nonMissing := []string{}
		includeMissing := false
		for _, s := range statuses {
			trimmed := strings.TrimSpace(s)
			if trimmed == "__MISSING__" {
				includeMissing = true
			} else if trimmed != "" {
				nonMissing = append(nonMissing, trimmed)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.django_status = $%d", loanArgCount))
			loanArgs = append(loanArgs, nonMissing[0])
			loanArgCount++
		} else if len(nonMissing) > 1 {
			placeholders := []string{}
			for _, s := range nonMissing {
				placeholders = append(placeholders, fmt.Sprintf("$%d", loanArgCount))
				loanArgs = append(loanArgs, s)
				loanArgCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.django_status IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.django_status IS NULL OR l.django_status = '')")
		}

		if len(conditions) > 0 {
			loanQuery += " AND (" + strings.Join(conditions, " OR ") + ")"
		}
	}

	loanQuery += " GROUP BY l.branch"

	loanRows, err := r.db.Query(loanQuery, loanArgs...)
	if err != nil {
		return nil, err
	}
	defer loanRows.Close()

	branchMap := make(map[string]*models.BranchCollectionsLeaderboardRow)

	for loanRows.Next() {
		row := &models.BranchCollectionsLeaderboardRow{}
		if err := loanRows.Scan(
			&row.Branch,
			&row.Region,
			&row.PortfolioTotal,
			&row.DueToday,
			&row.Overdue15d,
		); err != nil {
			return nil, err
		}

		// Initialise numeric fields explicitly (Go defaults to zero, but keep clear)
		row.CollectedToday = 0
		row.TodayRate = 0
		row.MTDRate = 0
		row.ProgressRate = 0
		row.MissedToday = 0
		row.NPLRatio = 0
		row.Status = ""

		branchMap[row.Branch] = row
	}

	// --- Second query: repayment-based metrics per branch (collections today) ---
	repayQuery := `
		SELECT
			l.branch,
			COALESCE(SUM(r.payment_amount), 0) AS collected_today
		FROM repayments r
		JOIN loans l ON r.loan_id = l.loan_id
		JOIN officers o ON l.officer_id = o.officer_id
		WHERE 1=1
			AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
			AND r.is_reversed = FALSE
			AND r.payment_date::date = CURRENT_DATE
	`

	repayArgs := []interface{}{}
	repayArgCount := 1

	if branch, ok := filters["branch"].(string); ok && branch != "" {
		repayQuery += fmt.Sprintf(" AND l.branch = $%d", repayArgCount)
		repayArgs = append(repayArgs, branch)
		repayArgCount++
	}

	if region, ok := filters["region"].(string); ok && region != "" {
		regions := strings.Split(region, ",")
		if len(regions) == 1 {
			repayQuery += fmt.Sprintf(" AND l.region = $%d", repayArgCount)
			repayArgs = append(repayArgs, regions[0])
			repayArgCount++
		} else {
			placeholders := []string{}
			for _, rgn := range regions {
				placeholders = append(placeholders, fmt.Sprintf("$%d", repayArgCount))
				repayArgs = append(repayArgs, strings.TrimSpace(rgn))
				repayArgCount++
			}
			repayQuery += fmt.Sprintf(" AND l.region IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	if channel, ok := filters["channel"].(string); ok && channel != "" {
		repayQuery += fmt.Sprintf(" AND l.channel = $%d", repayArgCount)
		repayArgs = append(repayArgs, channel)
		repayArgCount++
	}

	if wave, ok := filters["wave"].(string); ok && wave != "" {
		repayQuery += fmt.Sprintf(" AND l.wave = $%d", repayArgCount)
		repayArgs = append(repayArgs, wave)
		repayArgCount++
	}

	if loanType, ok := filters["loan_type"].(string); ok && loanType != "" {
		loanTypes := strings.Split(loanType, ",")
		if len(loanTypes) == 1 {
			repayQuery += fmt.Sprintf(" AND l.loan_type = $%d", repayArgCount)
			repayArgs = append(repayArgs, strings.TrimSpace(loanTypes[0]))
			repayArgCount++
		} else {
			placeholders := make([]string, len(loanTypes))
			for i, lt := range loanTypes {
				placeholders[i] = fmt.Sprintf("$%d", repayArgCount)
				repayArgs = append(repayArgs, strings.TrimSpace(lt))
				repayArgCount++
			}
			repayQuery += fmt.Sprintf(" AND l.loan_type IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	// Django status filter for repayments - supports comma-separated values and optional missing sentinel
	if djangoStatus, ok := filters["django_status"].(string); ok && djangoStatus != "" {
		statuses := strings.Split(djangoStatus, ",")
		nonMissing := []string{}
		includeMissing := false
		for _, s := range statuses {
			trimmed := strings.TrimSpace(s)
			if trimmed == "__MISSING__" {
				includeMissing = true
			} else if trimmed != "" {
				nonMissing = append(nonMissing, trimmed)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.django_status = $%d", repayArgCount))
			repayArgs = append(repayArgs, nonMissing[0])
			repayArgCount++
		} else if len(nonMissing) > 1 {
			placeholders := []string{}
			for _, s := range nonMissing {
				placeholders = append(placeholders, fmt.Sprintf("$%d", repayArgCount))
				repayArgs = append(repayArgs, s)
				repayArgCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.django_status IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.django_status IS NULL OR l.django_status = '')")
		}

		if len(conditions) > 0 {
			repayQuery += " AND (" + strings.Join(conditions, " OR ") + ")"
		}
	}

	repayQuery += " GROUP BY l.branch"

	repayRows, err := r.db.Query(repayQuery, repayArgs...)
	if err != nil {
		return nil, err
	}
	defer repayRows.Close()

	for repayRows.Next() {
		var branchName string
		var collectedToday float64
		if err := repayRows.Scan(&branchName, &collectedToday); err != nil {
			return nil, err
		}

		row, exists := branchMap[branchName]
		if !exists {
			row = &models.BranchCollectionsLeaderboardRow{Branch: branchName}
			branchMap[branchName] = row
		}
		row.CollectedToday = collectedToday
	}

	// --- Finalise metrics: rates, missed amount, NPL proxy & status ---
	result := make([]*models.BranchCollectionsLeaderboardRow, 0, len(branchMap))
	for _, row := range branchMap {
		if row.DueToday > 0 {
			row.TodayRate = row.CollectedToday / row.DueToday
			if row.TodayRate < 0 {
				row.TodayRate = 0
			}
		} else if row.CollectedToday > 0 {
			// No explicit due but collections recorded; treat as fully collected.
			row.TodayRate = 1
		} else {
			row.TodayRate = 0
		}

		// For now, use today's collection rate as both MTD and progress indicators.
		row.MTDRate = row.TodayRate
		row.ProgressRate = row.TodayRate

		row.MissedToday = row.DueToday - row.CollectedToday
		if row.MissedToday < 0 {
			row.MissedToday = 0
		}

		if row.PortfolioTotal > 0 {
			row.NPLRatio = row.Overdue15d / row.PortfolioTotal
		} else {
			row.NPLRatio = 0
		}

		// Simple status banding based on NPL ratio (inspired by sample: OK/Watch/Critical).
		if row.NPLRatio < 0.12 {
			row.Status = "OK"
		} else if row.NPLRatio < 0.18 {
			row.Status = "Watch"
		} else {
			row.Status = "Critical"
		}

		result = append(result, row)
	}

	return result, nil
}

// GetOfficerCollectionsLeaderboard returns per-officer collections metrics for the
// Agent/Officer Leaderboard views. It mirrors GetBranchCollectionsLeaderboard but
// groups by officer instead of branch.
func (r *DashboardRepository) GetOfficerCollectionsLeaderboard(filters map[string]interface{}) ([]*models.OfficerCollectionsLeaderboardRow, error) {
	// --- First query: loan-based metrics per officer (portfolio, due today, PAR15) ---
	loanQuery := `
			SELECT
				l.officer_id,
				COALESCE(o.officer_name, '') AS officer_name,
				COALESCE(o.officer_email, '') AS officer_email,
				MODE() WITHIN GROUP (ORDER BY l.branch) AS branch,
				MODE() WITHIN GROUP (ORDER BY l.region) AS region,
				COALESCE(SUM(l.repayment_amount), 0) AS portfolio_total,
				COALESCE(SUM(CASE WHEN l.actual_outstanding > 0 THEN l.daily_repayment_amount ELSE 0 END), 0) AS due_today,
				COALESCE(SUM(CASE WHEN l.current_dpd >= 15 THEN l.principal_outstanding ELSE 0 END), 0) AS overdue_15d
			FROM loans l
			JOIN officers o ON l.officer_id = o.officer_id
			WHERE 1=1
				AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
		`

	loanArgs := []interface{}{}
	loanArgCount := 1

	// Apply filters (same semantics as branch collections leaderboard where relevant).
	if branch, ok := filters["branch"].(string); ok && branch != "" {
		loanQuery += fmt.Sprintf(" AND l.branch = $%d", loanArgCount)
		loanArgs = append(loanArgs, branch)
		loanArgCount++
	}

	if region, ok := filters["region"].(string); ok && region != "" {
		regions := strings.Split(region, ",")
		if len(regions) == 1 {
			loanQuery += fmt.Sprintf(" AND l.region = $%d", loanArgCount)
			loanArgs = append(loanArgs, regions[0])
			loanArgCount++
		} else {
			placeholders := []string{}
			for _, rgn := range regions {
				placeholders = append(placeholders, fmt.Sprintf("$%d", loanArgCount))
				loanArgs = append(loanArgs, strings.TrimSpace(rgn))
				loanArgCount++
			}
			loanQuery += fmt.Sprintf(" AND l.region IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	if channel, ok := filters["channel"].(string); ok && channel != "" {
		loanQuery += fmt.Sprintf(" AND l.channel = $%d", loanArgCount)
		loanArgs = append(loanArgs, channel)
		loanArgCount++
	}

	if wave, ok := filters["wave"].(string); ok && wave != "" {
		loanQuery += fmt.Sprintf(" AND l.wave = $%d", loanArgCount)
		loanArgs = append(loanArgs, wave)
		loanArgCount++
	}

	if loanType, ok := filters["loan_type"].(string); ok && loanType != "" {
		loanTypes := strings.Split(loanType, ",")
		if len(loanTypes) == 1 {
			loanQuery += fmt.Sprintf(" AND l.loan_type = $%d", loanArgCount)
			loanArgs = append(loanArgs, strings.TrimSpace(loanTypes[0]))
			loanArgCount++
		} else {
			placeholders := make([]string, len(loanTypes))
			for i, lt := range loanTypes {
				placeholders[i] = fmt.Sprintf("$%d", loanArgCount)
				loanArgs = append(loanArgs, strings.TrimSpace(lt))
				loanArgCount++
			}
			loanQuery += fmt.Sprintf(" AND l.loan_type IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	// Django status filter - supports comma-separated values and optional missing sentinel
	if djangoStatus, ok := filters["django_status"].(string); ok && djangoStatus != "" {
		statuses := strings.Split(djangoStatus, ",")
		nonMissing := []string{}
		includeMissing := false
		for _, s := range statuses {
			trimmed := strings.TrimSpace(s)
			if trimmed == "__MISSING__" {
				includeMissing = true
			} else if trimmed != "" {
				nonMissing = append(nonMissing, trimmed)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.django_status = $%d", loanArgCount))
			loanArgs = append(loanArgs, nonMissing[0])
			loanArgCount++
		} else if len(nonMissing) > 1 {
			placeholders := []string{}
			for _, s := range nonMissing {
				placeholders = append(placeholders, fmt.Sprintf("$%d", loanArgCount))
				loanArgs = append(loanArgs, s)
				loanArgCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.django_status IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.django_status IS NULL OR l.django_status = '')")
		}

		if len(conditions) > 0 {
			loanQuery += " AND (" + strings.Join(conditions, " OR ") + ")"
		}
	}

	loanQuery += " GROUP BY l.officer_id, o.officer_name, o.officer_email"

	loanRows, err := r.db.Query(loanQuery, loanArgs...)
	if err != nil {
		return nil, err
	}
	defer loanRows.Close()

	officerMap := make(map[string]*models.OfficerCollectionsLeaderboardRow)

	for loanRows.Next() {
		row := &models.OfficerCollectionsLeaderboardRow{}
		if err := loanRows.Scan(
			&row.OfficerID,
			&row.OfficerName,
			&row.OfficerEmail,
			&row.Branch,
			&row.Region,
			&row.PortfolioTotal,
			&row.DueToday,
			&row.Overdue15d,
		); err != nil {
			return nil, err
		}

		row.CollectedToday = 0
		row.TodayRate = 0
		row.MTDRate = 0
		row.ProgressRate = 0
		row.MissedToday = 0
		row.NPLRatio = 0
		row.Status = ""

		officerMap[row.OfficerID] = row
	}

	// --- Second query: repayment-based metrics per officer (collections today) ---
	repayQuery := `
			SELECT
				l.officer_id,
				COALESCE(SUM(r.payment_amount), 0) AS collected_today
			FROM repayments r
			JOIN loans l ON r.loan_id = l.loan_id
			JOIN officers o ON l.officer_id = o.officer_id
			WHERE 1=1
				AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
				AND r.is_reversed = FALSE
				AND r.payment_date::date = CURRENT_DATE
		`

	repayArgs := []interface{}{}
	repayArgCount := 1

	if branch, ok := filters["branch"].(string); ok && branch != "" {
		repayQuery += fmt.Sprintf(" AND l.branch = $%d", repayArgCount)
		repayArgs = append(repayArgs, branch)
		repayArgCount++
	}

	if region, ok := filters["region"].(string); ok && region != "" {
		regions := strings.Split(region, ",")
		if len(regions) == 1 {
			repayQuery += fmt.Sprintf(" AND l.region = $%d", repayArgCount)
			repayArgs = append(repayArgs, regions[0])
			repayArgCount++
		} else {
			placeholders := []string{}
			for _, rgn := range regions {
				placeholders = append(placeholders, fmt.Sprintf("$%d", repayArgCount))
				repayArgs = append(repayArgs, strings.TrimSpace(rgn))
				repayArgCount++
			}
			repayQuery += fmt.Sprintf(" AND l.region IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	if channel, ok := filters["channel"].(string); ok && channel != "" {
		repayQuery += fmt.Sprintf(" AND l.channel = $%d", repayArgCount)
		repayArgs = append(repayArgs, channel)
		repayArgCount++
	}

	if wave, ok := filters["wave"].(string); ok && wave != "" {
		repayQuery += fmt.Sprintf(" AND l.wave = $%d", repayArgCount)
		repayArgs = append(repayArgs, wave)
		repayArgCount++
	}

	if loanType, ok := filters["loan_type"].(string); ok && loanType != "" {
		loanTypes := strings.Split(loanType, ",")
		if len(loanTypes) == 1 {
			repayQuery += fmt.Sprintf(" AND l.loan_type = $%d", repayArgCount)
			repayArgs = append(repayArgs, strings.TrimSpace(loanTypes[0]))
			repayArgCount++
		} else {
			placeholders := make([]string, len(loanTypes))
			for i, lt := range loanTypes {
				placeholders[i] = fmt.Sprintf("$%d", repayArgCount)
				repayArgs = append(repayArgs, strings.TrimSpace(lt))
				repayArgCount++
			}
			repayQuery += fmt.Sprintf(" AND l.loan_type IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	// Django status filter for repayments - supports comma-separated values and optional missing sentinel
	if djangoStatus, ok := filters["django_status"].(string); ok && djangoStatus != "" {
		statuses := strings.Split(djangoStatus, ",")
		nonMissing := []string{}
		includeMissing := false
		for _, s := range statuses {
			trimmed := strings.TrimSpace(s)
			if trimmed == "__MISSING__" {
				includeMissing = true
			} else if trimmed != "" {
				nonMissing = append(nonMissing, trimmed)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.django_status = $%d", repayArgCount))
			repayArgs = append(repayArgs, nonMissing[0])
			repayArgCount++
		} else if len(nonMissing) > 1 {
			placeholders := []string{}
			for _, s := range nonMissing {
				placeholders = append(placeholders, fmt.Sprintf("$%d", repayArgCount))
				repayArgs = append(repayArgs, s)
				repayArgCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.django_status IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.django_status IS NULL OR l.django_status = '')")
		}

		if len(conditions) > 0 {
			repayQuery += " AND (" + strings.Join(conditions, " OR ") + ")"
		}
	}

	repayQuery += " GROUP BY l.officer_id"

	repayRows, err := r.db.Query(repayQuery, repayArgs...)
	if err != nil {
		return nil, err
	}
	defer repayRows.Close()

	for repayRows.Next() {
		var officerID string
		var collectedToday float64
		if err := repayRows.Scan(&officerID, &collectedToday); err != nil {
			return nil, err
		}

		row, exists := officerMap[officerID]
		if !exists {
			row = &models.OfficerCollectionsLeaderboardRow{OfficerID: officerID}
			officerMap[officerID] = row
		}
		row.CollectedToday = collectedToday
	}

	// --- Finalise metrics: rates, missed amount, NPL proxy & status ---
	result := make([]*models.OfficerCollectionsLeaderboardRow, 0, len(officerMap))
	for _, row := range officerMap {
		if row.DueToday > 0 {
			row.TodayRate = row.CollectedToday / row.DueToday
			if row.TodayRate < 0 {
				row.TodayRate = 0
			}
		} else if row.CollectedToday > 0 {
			row.TodayRate = 1
		} else {
			row.TodayRate = 0
		}

		row.MTDRate = row.TodayRate
		row.ProgressRate = row.TodayRate

		row.MissedToday = row.DueToday - row.CollectedToday
		if row.MissedToday < 0 {
			row.MissedToday = 0
		}

		if row.PortfolioTotal > 0 {
			row.NPLRatio = row.Overdue15d / row.PortfolioTotal
		} else {
			row.NPLRatio = 0
		}

		if row.NPLRatio < 0.12 {
			row.Status = "OK"
		} else if row.NPLRatio < 0.18 {
			row.Status = "Watch"
		} else {
			row.Status = "Critical"
		}

		result = append(result, row)
	}

	return result, nil
}

// GetFilterOptions retrieves filter dropdown options
func (r *DashboardRepository) GetFilterOptions(filterType string, filters map[string]interface{}) (interface{}, error) {
	switch filterType {
	case "branches":
		return r.getBranches(filters)
	case "regions":
		return r.getRegions()
	case "waves":
		return r.getWaves()
	case "channels":
		return r.getChannels()
	case "user-types":
		return r.getUserTypes()
	case "officers":
		return r.getOfficerOptions(filters)
	case "statuses":
		return r.getStatuses()
	case "loan-types":
		return r.getLoanTypes()
	case "verification-statuses":
		return r.getVerificationStatuses()
	case "django-statuses":
		return r.getDjangoStatuses()
	case "vertical-leads":
		return r.getVerticalLeads()
	default:
		return nil, fmt.Errorf("unknown filter type: %s", filterType)
	}
}

func (r *DashboardRepository) getBranches(filters map[string]interface{}) ([]string, error) {
	query := `SELECT DISTINCT l.branch FROM loans l
		INNER JOIN officers o ON l.officer_id = o.officer_id
		WHERE (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)`
	args := []interface{}{}
	argCount := 1

	if region, ok := filters["region"].(string); ok && region != "" {
		// Support comma-separated regions for multi-select
		regions := strings.Split(region, ",")
		if len(regions) == 1 {
			query += fmt.Sprintf(" AND l.region = $%d", argCount)
			args = append(args, regions[0])
			argCount++
		} else {
			// Build IN clause for multiple regions
			placeholders := []string{}
			for _, r := range regions {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
				args = append(args, strings.TrimSpace(r))
				argCount++
			}
			query += fmt.Sprintf(" AND l.region IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	query += " ORDER BY l.branch"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	branches := []string{}
	for rows.Next() {
		var branch string
		if err := rows.Scan(&branch); err != nil {
			return nil, err
		}
		branches = append(branches, branch)
	}

	return branches, nil
}

// GetDailyCollections returns a per-day time series of collections amounts for the
// Collections Control Centre daily chart. It aggregates repayments by payment_date
// and applies the same officer and loan filters as other collections metrics.
func (r *DashboardRepository) GetDailyCollections(filters map[string]interface{}) ([]*models.DailyCollectionsPoint, error) {
	// Determine requested period, defaulting to "today".
	period := "today"
	if p, ok := filters["period"].(string); ok && strings.TrimSpace(p) != "" {
		period = strings.ToLower(strings.TrimSpace(p))
	}

	query := `
		SELECT
			DATE(r.payment_date) AS payment_date,
			COALESCE(SUM(r.payment_amount), 0) AS collected_amount,
			COUNT(*) AS repayments_count
		FROM repayments r
		INNER JOIN loans l ON r.loan_id = l.loan_id
		INNER JOIN officers o ON l.officer_id = o.officer_id
		WHERE r.is_reversed = false
			AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
	`

	// Apply period restriction on repayment dates.
	switch period {
	case "this_week":
		query += `
				AND DATE(r.payment_date) >= DATE_TRUNC('week', CURRENT_DATE)::date
				AND DATE(r.payment_date) <= CURRENT_DATE
			`
	case "this_month":
		query += `
				AND DATE(r.payment_date) >= DATE_TRUNC('month', CURRENT_DATE)::date
				AND DATE(r.payment_date) <= CURRENT_DATE
			`
	case "last_month":
		query += `
				AND DATE(r.payment_date) >= (DATE_TRUNC('month', CURRENT_DATE) - INTERVAL '1 month')::date
				AND DATE(r.payment_date) < DATE_TRUNC('month', CURRENT_DATE)::date
			`
	case "last_7_days":
		// Custom period for the Collections Control Centre daily chart:
		// always show the last 7 calendar days (including today).
		query += `
				AND DATE(r.payment_date) >= (CURRENT_DATE - INTERVAL '6 days')
				AND DATE(r.payment_date) <= CURRENT_DATE
			`
	default: // "today" or any unrecognised value
		query += `
				AND DATE(r.payment_date) = CURRENT_DATE
			`
	}

	args := []interface{}{}
	argCount := 1

	// Apply the same filters as other collections repayments aggregations so that
	// the chart stays aligned with the KPI cards.
	if officerID, ok := filters["officer_id"].(string); ok && officerID != "" {
		query += fmt.Sprintf(" AND l.officer_id = $%d", argCount)
		args = append(args, officerID)
		argCount++
	}

	if branch, ok := filters["branch"].(string); ok && branch != "" {
		query += fmt.Sprintf(" AND l.branch = $%d", argCount)
		args = append(args, branch)
		argCount++
	}

	if region, ok := filters["region"].(string); ok && region != "" {
		regions := strings.Split(region, ",")
		if len(regions) == 1 {
			query += fmt.Sprintf(" AND l.region = $%d", argCount)
			args = append(args, regions[0])
			argCount++
		} else {
			placeholders := []string{}
			for _, rgn := range regions {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
				args = append(args, strings.TrimSpace(rgn))
				argCount++
			}
			query += fmt.Sprintf(" AND l.region IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	if channel, ok := filters["channel"].(string); ok && channel != "" {
		query += fmt.Sprintf(" AND l.channel = $%d", argCount)
		args = append(args, channel)
		argCount++
	}

	if status, ok := filters["status"].(string); ok && status != "" {
		statuses := strings.Split(status, ",")
		if len(statuses) == 1 {
			query += fmt.Sprintf(" AND l.status = $%d", argCount)
			args = append(args, statuses[0])
			argCount++
		} else {
			placeholders := []string{}
			for _, s := range statuses {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
				args = append(args, strings.TrimSpace(s))
				argCount++
			}
			query += fmt.Sprintf(" AND l.status IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	// Django status filter with MissingValueSentinel support.
	if djangoStatus, ok := filters["django_status"].(string); ok && djangoStatus != "" {
		statuses := strings.Split(djangoStatus, ",")
		nonMissing := []string{}
		includeMissing := false

		for _, s := range statuses {
			value := strings.TrimSpace(s)
			if value == "" {
				continue
			}
			if value == MissingValueSentinel {
				includeMissing = true
			} else {
				nonMissing = append(nonMissing, value)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.django_status = $%d", argCount))
			args = append(args, nonMissing[0])
			argCount++
		} else if len(nonMissing) > 1 {
			placeholders := make([]string, len(nonMissing))
			for i, s := range nonMissing {
				placeholders[i] = fmt.Sprintf("$%d", argCount)
				args = append(args, s)
				argCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.django_status IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.django_status IS NULL OR l.django_status = '')")
		}

		if len(conditions) > 0 {
			query += " AND (" + strings.Join(conditions, " OR ") + ")"
		}
	}

	if performanceStatus, ok := filters["performance_status"].(string); ok && performanceStatus != "" {
		performanceStatuses := strings.Split(performanceStatus, ",")
		if len(performanceStatuses) == 1 {
			query += fmt.Sprintf(" AND l.performance_status = $%d", argCount)
			args = append(args, performanceStatuses[0])
			argCount++
		} else {
			placeholders := []string{}
			for _, ps := range performanceStatuses {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
				args = append(args, strings.TrimSpace(ps))
				argCount++
			}
			query += fmt.Sprintf(" AND l.performance_status IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	if wave, ok := filters["wave"].(string); ok && wave != "" {
		query += fmt.Sprintf(" AND l.wave = $%d", argCount)
		args = append(args, wave)
		argCount++
	}

	if customerPhone, ok := filters["customer_phone"].(string); ok && customerPhone != "" {
		query += fmt.Sprintf(" AND l.customer_phone LIKE $%d", argCount)
		args = append(args, "%"+customerPhone+"%")
		argCount++
	}

	if verticalLeadEmail, ok := filters["vertical_lead_email"].(string); ok && verticalLeadEmail != "" {
		query += fmt.Sprintf(" AND l.vertical_lead_email = $%d", argCount)
		args = append(args, verticalLeadEmail)
		argCount++
	}

	if loanType, ok := filters["loan_type"].(string); ok && loanType != "" {
		loanTypes := strings.Split(loanType, ",")
		nonMissing := []string{}
		includeMissing := false

		for _, lt := range loanTypes {
			value := strings.TrimSpace(lt)
			if value == "" {
				continue
			}
			if value == MissingValueSentinel {
				includeMissing = true
			} else {
				nonMissing = append(nonMissing, value)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.loan_type = $%d", argCount))
			args = append(args, nonMissing[0])
			argCount++
		} else if len(nonMissing) > 1 {
			placeholders := make([]string, len(nonMissing))
			for i, lt := range nonMissing {
				placeholders[i] = fmt.Sprintf("$%d", argCount)
				args = append(args, lt)
				argCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.loan_type IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.loan_type IS NULL OR l.loan_type = '')")
		}

		if len(conditions) > 0 {
			query += " AND (" + strings.Join(conditions, " OR ") + ")"
		}
	}

	if verificationStatus, ok := filters["verification_status"].(string); ok && verificationStatus != "" {
		verificationStatuses := strings.Split(verificationStatus, ",")
		nonMissing := []string{}
		includeMissing := false

		for _, vs := range verificationStatuses {
			value := strings.TrimSpace(vs)
			if value == "" {
				continue
			}
			if value == MissingValueSentinel {
				includeMissing = true
			} else {
				nonMissing = append(nonMissing, value)
			}
		}

		conditions := []string{}
		if len(nonMissing) == 1 {
			conditions = append(conditions, fmt.Sprintf("l.verification_status = $%d", argCount))
			args = append(args, nonMissing[0])
			argCount++
		} else if len(nonMissing) > 1 {
			placeholders := make([]string, len(nonMissing))
			for i, vs := range nonMissing {
				placeholders[i] = fmt.Sprintf("$%d", argCount)
				args = append(args, vs)
				argCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.verification_status IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.verification_status IS NULL OR l.verification_status = '')")
		}

		if len(conditions) > 0 {
			query += " AND (" + strings.Join(conditions, " OR ") + ")"
		}
	}

	if dpdMin, ok := filters["dpd_min"].(int); ok {
		query += fmt.Sprintf(" AND l.current_dpd >= $%d", argCount)
		args = append(args, dpdMin)
		argCount++
	}

	if dpdMax, ok := filters["dpd_max"].(int); ok {
		query += fmt.Sprintf(" AND l.current_dpd <= $%d", argCount)
		args = append(args, dpdMax)
		argCount++
	}

	query += `
		GROUP BY DATE(r.payment_date)
		ORDER BY DATE(r.payment_date)
	`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve daily collections: %w", err)
	}
	defer rows.Close()

	results := []*models.DailyCollectionsPoint{}
	for rows.Next() {
		point := &models.DailyCollectionsPoint{}
		if err := rows.Scan(&point.Date, &point.CollectedAmount, &point.RepaymentsCount); err != nil {
			return nil, fmt.Errorf("failed to scan daily collections row: %w", err)
		}
		results = append(results, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate daily collections rows: %w", err)
	}

	return results, nil
}

func (r *DashboardRepository) getRegions() ([]string, error) {
	query := `SELECT DISTINCT l.region FROM loans l
		INNER JOIN officers o ON l.officer_id = o.officer_id
		WHERE (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
		ORDER BY l.region`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	regions := []string{}
	for rows.Next() {
		var region string
		if err := rows.Scan(&region); err != nil {
			return nil, err
		}
		regions = append(regions, region)
	}

	return regions, nil
}

func (r *DashboardRepository) getWaves() ([]string, error) {
	query := `SELECT DISTINCT l.wave FROM loans l
		INNER JOIN officers o ON l.officer_id = o.officer_id
		WHERE l.wave IS NOT NULL
		AND l.wave != ''
		AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
		ORDER BY l.wave`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	waves := []string{}
	for rows.Next() {
		var wave string
		if err := rows.Scan(&wave); err != nil {
			return nil, err
		}
		waves = append(waves, wave)
	}

	return waves, nil
}

func (r *DashboardRepository) getChannels() ([]string, error) {
	query := `SELECT DISTINCT l.channel FROM loans l
		INNER JOIN officers o ON l.officer_id = o.officer_id
		WHERE (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
		ORDER BY l.channel`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	channels := []string{}
	for rows.Next() {
		var channel string
		if err := rows.Scan(&channel); err != nil {
			return nil, err
		}
		channels = append(channels, channel)
	}

	return channels, nil
}

func (r *DashboardRepository) getUserTypes() ([]string, error) {
	query := `SELECT DISTINCT user_type FROM officers
		WHERE user_type IS NOT NULL
		AND user_type != ''
		AND user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT')
		ORDER BY user_type`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userTypes := []string{}
	for rows.Next() {
		var userType string
		if err := rows.Scan(&userType); err != nil {
			return nil, err
		}
		userTypes = append(userTypes, userType)
	}

	return userTypes, nil
}

func (r *DashboardRepository) getStatuses() ([]string, error) {
	query := `SELECT DISTINCT l.status FROM loans l
		INNER JOIN officers o ON l.officer_id = o.officer_id
		WHERE l.status IS NOT NULL
		AND l.status != ''
		AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
		ORDER BY l.status`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	statuses := []string{}
	for rows.Next() {
		var status string
		if err := rows.Scan(&status); err != nil {
			return nil, err
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

func (r *DashboardRepository) getLoanTypes() ([]string, error) {
	query := `SELECT DISTINCT l.loan_type FROM loans l
		INNER JOIN officers o ON l.officer_id = o.officer_id
		WHERE l.loan_type IS NOT NULL
		AND l.loan_type != ''
		AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
		ORDER BY l.loan_type`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	loanTypes := []string{}
	for rows.Next() {
		var loanType string
		if err := rows.Scan(&loanType); err != nil {
			return nil, err
		}
		loanTypes = append(loanTypes, loanType)
	}

	return loanTypes, nil
}

func (r *DashboardRepository) getVerificationStatuses() ([]string, error) {
	query := `SELECT DISTINCT l.verification_status FROM loans l
		INNER JOIN officers o ON l.officer_id = o.officer_id
		WHERE l.verification_status IS NOT NULL
		AND l.verification_status != ''
		AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
		ORDER BY l.verification_status`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	verificationStatuses := []string{}
	for rows.Next() {
		var status string
		if err := rows.Scan(&status); err != nil {
			return nil, err
		}
		verificationStatuses = append(verificationStatuses, status)
	}

	return verificationStatuses, nil
}

// getVerticalLeads returns the distinct vertical lead emails used on loans
func (r *DashboardRepository) getVerticalLeads() ([]string, error) {
	query := `SELECT DISTINCT l.vertical_lead_email FROM loans l
		INNER JOIN officers o ON l.officer_id = o.officer_id
		WHERE l.vertical_lead_email IS NOT NULL
		AND l.vertical_lead_email != ''
		AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
		ORDER BY l.vertical_lead_email`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	verticalLeads := []string{}
	for rows.Next() {
		var email string
		if err := rows.Scan(&email); err != nil {
			return nil, err
		}
		verticalLeads = append(verticalLeads, email)
	}

	return verticalLeads, nil
}

// getDjangoStatuses returns the distinct raw Django status values stored on loans.django_status
func (r *DashboardRepository) getDjangoStatuses() ([]string, error) {
	query := `SELECT DISTINCT l.django_status FROM loans l
		INNER JOIN officers o ON l.officer_id = o.officer_id
		WHERE l.django_status IS NOT NULL
		AND l.django_status != ''
		AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
		ORDER BY l.django_status`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	statuses := []string{}
	for rows.Next() {
		var status string
		if err := rows.Scan(&status); err != nil {
			return nil, err
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

func (r *DashboardRepository) getOfficerOptions(filters map[string]interface{}) ([]*models.OfficerOption, error) {
	query := `SELECT DISTINCT l.officer_id, l.officer_name, o.officer_email, l.branch, l.region FROM loans l
		INNER JOIN officers o ON l.officer_id = o.officer_id
		WHERE (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)`
	args := []interface{}{}
	argCount := 1

	if branch, ok := filters["branch"].(string); ok && branch != "" {
		query += fmt.Sprintf(" AND l.branch = $%d", argCount)
		args = append(args, branch)
		argCount++
	}

	if region, ok := filters["region"].(string); ok && region != "" {
		// Support comma-separated regions for multi-select
		regions := strings.Split(region, ",")
		if len(regions) == 1 {
			query += fmt.Sprintf(" AND l.region = $%d", argCount)
			args = append(args, regions[0])
			argCount++
		} else {
			// Build IN clause for multiple regions
			placeholders := []string{}
			for _, r := range regions {
				placeholders = append(placeholders, fmt.Sprintf("$%d", argCount))
				args = append(args, strings.TrimSpace(r))
				argCount++
			}
			query += fmt.Sprintf(" AND l.region IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	query += " ORDER BY l.officer_name"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	officers := []*models.OfficerOption{}
	for rows.Next() {
		officer := &models.OfficerOption{}
		if err := rows.Scan(&officer.OfficerID, &officer.Name, &officer.Email, &officer.Branch, &officer.Region); err != nil {
			return nil, err
		}
		officers = append(officers, officer)
	}

	return officers, nil
}

// GetTeamMembers retrieves team members for audit assignment
func (r *DashboardRepository) GetTeamMembers() ([]*models.TeamMember, error) {
	query := "SELECT member_id, member_name, role FROM team_members WHERE is_active = true ORDER BY role, member_name"

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := []*models.TeamMember{
		{ID: 0, Name: "Unassigned", Role: ""},
		{ID: "me", Name: "Assigned to Me", Role: "Current User"},
	}

	for rows.Next() {
		member := &models.TeamMember{}
		if err := rows.Scan(&member.ID, &member.Name, &member.Role); err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return members, nil
}

// UpdateOfficerAudit updates audit assignment for an officer
func (r *DashboardRepository) UpdateOfficerAudit(officerID string, update *models.AuditUpdate) error {
	query := `
		INSERT INTO audit_tracking (officer_id, assignee_id, assignee_name, audit_status, audit_date)
		VALUES ($1, $2, $3, $4, CURRENT_DATE)
		ON CONFLICT (officer_id)
		DO UPDATE SET
			assignee_id = $2,
			assignee_name = $3,
			audit_status = $4,
			audit_date = CURRENT_DATE,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := r.db.Exec(query, officerID, update.AssigneeID, update.AssigneeName, update.AuditStatus)
	return err
}

// GetOfficerAuditHistory retrieves audit history for an officer
func (r *DashboardRepository) GetOfficerAuditHistory(officerID string, limit int) ([]*models.AuditHistory, error) {
	query := `
		SELECT id, officer_id, assignee_id, assignee_name, audit_status, audit_date, notes, created_at
		FROM audit_tracking
		WHERE officer_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := r.db.Query(query, officerID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := []*models.AuditHistory{}
	for rows.Next() {
		item := &models.AuditHistory{}
		var notes sql.NullString
		err := rows.Scan(
			&item.ID,
			&item.OfficerID,
			&item.AssigneeID,
			&item.AssigneeName,
			&item.AuditStatus,
			&item.AuditDate,
			&notes,
			&item.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if notes.Valid {
			item.Notes = notes.String
		}

		history = append(history, item)
	}

	return history, nil
}

// UpdatePastMaturityStatus updates django_status to 'PAST_MATURITY' for eligible loans.
// It only affects loans that are currently marked as OPEN and have a maturity_date
// earlier than the current date. Other django_status values (COMPLETED, DECLINED, etc.)
// are left unchanged. Returns the count of loans updated.
func (r *DashboardRepository) UpdatePastMaturityStatus() (int64, error) {
	query := `
		UPDATE loans
		SET django_status = 'PAST_MATURITY'
		WHERE maturity_date < CURRENT_DATE
		  AND django_status = 'OPEN'
	`

	result, err := r.db.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("failed to update past maturity status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}
