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

// RecalculateAllLoanFields triggers comprehensive recalculation of all computed fields for all loans.
//
// It performs two steps:
//  1. Calls the recalculate_all_loan_fields() stored procedure which recomputes all
//     derived metrics (DPD, risk tags, timeliness scores, etc.) using the database logic.
//  2. Applies a safety normalisation pass directly on the loans table to ensure that
//     monetary fields are internally consistent, specifically:
//     - total_outstanding is always max(0, repayment_amount - total_repayments)
//     - actual_outstanding is never greater than total_outstanding
//
// This second step gives us the business guarantee that "Actual Outstanding" can
// never exceed the contractual "Outstanding" amount, even if older versions of the
// database function left inconsistent values behind.
func (r *DashboardRepository) RecalculateAllLoanFields() (int64, error) {
	// Step 1: run the main database-side recalculation.
	//
	// We intentionally call the function via Exec rather than QueryRow+Scan so that
	// this code is compatible with older deployments where
	// recalculate_all_loan_fields() may not return the
	// (total_loans_processed, loans_updated, execution_time_ms) columns.
	if _, err := r.db.Exec("SELECT recalculate_all_loan_fields()"); err != nil {
		return 0, fmt.Errorf("failed to recalculate loan fields: %w", err)
	}

	// Step 2: enforce consistent outstanding balances using a single, set-based UPDATE.
	//
	// This uses only stable columns (repayment_amount, total_repayments, total_outstanding,
	// actual_outstanding) and does NOT depend on any particular version of the
	// recalculate_all_loan_fields() implementation.
	fixQuery := `
			UPDATE loans
			SET
				-- Contractual remaining balance should always be non-negative and equal
				-- to repayment_amount - total_repayments.
				total_outstanding = GREATEST(
					0,
					COALESCE(repayment_amount, 0) - COALESCE(total_repayments, 0)
				),
				-- Actual outstanding should never exceed the contractual balance.
				actual_outstanding = LEAST(
					COALESCE(actual_outstanding, 0),
					GREATEST(0, COALESCE(repayment_amount, 0) - COALESCE(total_repayments, 0))
				)
			WHERE
				-- Only touch rows where the values are inconsistent with the business rules.
				total_outstanding != GREATEST(
					0,
					COALESCE(repayment_amount, 0) - COALESCE(total_repayments, 0)
				)
				OR actual_outstanding > GREATEST(
					0,
					COALESCE(repayment_amount, 0) - COALESCE(total_repayments, 0)
				);
		`

	result, err := r.db.Exec(fixQuery)
	if err != nil {
		return 0, fmt.Errorf("failed to normalise outstanding balances: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	return rowsAffected, nil
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

	// Quiet Loans filter: when enabled, restrict to loans with 6+ days since
	// last repayment or with no repayments at all. This keeps summary metrics
	// aligned with the All Loans table and exports when the Quiet Loans toggle
	// is active.
	if quietLoans, ok := filters["quiet_loans"].(bool); ok && quietLoans {
		query += " AND (l.days_since_last_repayment >= 6 OR l.days_since_last_repayment IS NULL)"
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

	// Quiet Loans filter for repayments aggregates so that "Collection Today"
	// and related metrics reflect the same quiet-loan population as the table.
	if quietLoans, ok := filters["quiet_loans"].(bool); ok && quietLoans {
		repaymentsWhere += " AND (l.days_since_last_repayment >= 6 OR l.days_since_last_repayment IS NULL)"
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

	// Additionally calculate total repayments for "yesterday" (exactly one
	// calendar day before today). This intentionally ignores the selected
	// period filter for the repayments date range so that the metric always
	// represents "yesterday" while still respecting all other filters
	// (branch, region, officer, loan type, etc.).
	repaymentsWhereYesterday := `
				FROM repayments r
				INNER JOIN loans l ON r.loan_id = l.loan_id
				INNER JOIN officers o ON l.officer_id = o.officer_id
				WHERE r.is_reversed = false
					AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
					AND DATE(r.payment_date) = CURRENT_DATE - INTERVAL '1 day'
			`

	repaymentsYesterdayArgs := []interface{}{}
	repaymentsYesterdayArgCount := 1

	if officerID, ok := filters["officer_id"].(string); ok && officerID != "" {
		repaymentsWhereYesterday += fmt.Sprintf(" AND l.officer_id = $%d", repaymentsYesterdayArgCount)
		repaymentsYesterdayArgs = append(repaymentsYesterdayArgs, officerID)
		repaymentsYesterdayArgCount++
	}

	if branch, ok := filters["branch"].(string); ok && branch != "" {
		repaymentsWhereYesterday += fmt.Sprintf(" AND l.branch = $%d", repaymentsYesterdayArgCount)
		repaymentsYesterdayArgs = append(repaymentsYesterdayArgs, branch)
		repaymentsYesterdayArgCount++
	}

	if region, ok := filters["region"].(string); ok && region != "" {
		regions := strings.Split(region, ",")
		if len(regions) == 1 {
			repaymentsWhereYesterday += fmt.Sprintf(" AND l.region = $%d", repaymentsYesterdayArgCount)
			repaymentsYesterdayArgs = append(repaymentsYesterdayArgs, regions[0])
			repaymentsYesterdayArgCount++
		} else {
			placeholders := []string{}
			for _, rgn := range regions {
				placeholders = append(placeholders, fmt.Sprintf("$%d", repaymentsYesterdayArgCount))
				repaymentsYesterdayArgs = append(repaymentsYesterdayArgs, strings.TrimSpace(rgn))
				repaymentsYesterdayArgCount++
			}
			repaymentsWhereYesterday += fmt.Sprintf(" AND l.region IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	if channel, ok := filters["channel"].(string); ok && channel != "" {
		repaymentsWhereYesterday += fmt.Sprintf(" AND l.channel = $%d", repaymentsYesterdayArgCount)
		repaymentsYesterdayArgs = append(repaymentsYesterdayArgs, channel)
		repaymentsYesterdayArgCount++
	}

	if status, ok := filters["status"].(string); ok && status != "" {
		statuses := strings.Split(status, ",")
		if len(statuses) == 1 {
			repaymentsWhereYesterday += fmt.Sprintf(" AND l.status = $%d", repaymentsYesterdayArgCount)
			repaymentsYesterdayArgs = append(repaymentsYesterdayArgs, statuses[0])
			repaymentsYesterdayArgCount++
		} else {
			placeholders := []string{}
			for _, s := range statuses {
				placeholders = append(placeholders, fmt.Sprintf("$%d", repaymentsYesterdayArgCount))
				repaymentsYesterdayArgs = append(repaymentsYesterdayArgs, strings.TrimSpace(s))
				repaymentsYesterdayArgCount++
			}
			repaymentsWhereYesterday += fmt.Sprintf(" AND l.status IN (%s)", strings.Join(placeholders, ", "))
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
			repaymentsWhereYesterday += fmt.Sprintf(" AND l.django_status = $%d", repaymentsYesterdayArgCount)
			repaymentsYesterdayArgs = append(repaymentsYesterdayArgs, nonMissing[0])
			repaymentsYesterdayArgCount++
		} else if len(nonMissing) > 1 {
			placeholders := make([]string, len(nonMissing))
			for i, s := range nonMissing {
				placeholders[i] = fmt.Sprintf("$%d", repaymentsYesterdayArgCount)
				repaymentsYesterdayArgs = append(repaymentsYesterdayArgs, s)
				repaymentsYesterdayArgCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.django_status IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.django_status IS NULL OR l.django_status = '')")
		}

		if len(conditions) > 0 {
			repaymentsWhereYesterday += " AND (" + strings.Join(conditions, " OR ") + ")"
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

		if len(nonMissing) == 1 {
			repaymentsWhereYesterday += fmt.Sprintf(" AND l.performance_status = $%d", repaymentsYesterdayArgCount)
			repaymentsYesterdayArgs = append(repaymentsYesterdayArgs, nonMissing[0])
			repaymentsYesterdayArgCount++
		} else if len(nonMissing) > 1 {
			placeholders := []string{}
			for _, ps := range nonMissing {
				placeholders = append(placeholders, fmt.Sprintf("$%d", repaymentsYesterdayArgCount))
				repaymentsYesterdayArgs = append(repaymentsYesterdayArgs, strings.TrimSpace(ps))
				repaymentsYesterdayArgCount++
			}
			repaymentsWhereYesterday += fmt.Sprintf(" AND l.performance_status IN (%s)", strings.Join(placeholders, ", "))
		}

		if includeMissing {
			repaymentsWhereYesterday += " AND (l.performance_status IS NULL OR l.performance_status = '')"
		}
	}

	if wave, ok := filters["wave"].(string); ok && wave != "" {
		repaymentsWhereYesterday += fmt.Sprintf(" AND l.wave = $%d", repaymentsYesterdayArgCount)
		repaymentsYesterdayArgs = append(repaymentsYesterdayArgs, wave)
		repaymentsYesterdayArgCount++
	}

	if customerPhone, ok := filters["customer_phone"].(string); ok && customerPhone != "" {
		repaymentsWhereYesterday += fmt.Sprintf(" AND l.customer_phone LIKE $%d", repaymentsYesterdayArgCount)
		repaymentsYesterdayArgs = append(repaymentsYesterdayArgs, "%"+customerPhone+"%")
		repaymentsYesterdayArgCount++
	}

	if verticalLeadEmail, ok := filters["vertical_lead_email"].(string); ok && verticalLeadEmail != "" {
		repaymentsWhereYesterday += fmt.Sprintf(" AND l.vertical_lead_email = $%d", repaymentsYesterdayArgCount)
		repaymentsYesterdayArgs = append(repaymentsYesterdayArgs, verticalLeadEmail)
		repaymentsYesterdayArgCount++
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
			repaymentsWhereYesterday += fmt.Sprintf(" AND l.loan_type = $%d", repaymentsYesterdayArgCount)
			repaymentsYesterdayArgs = append(repaymentsYesterdayArgs, nonMissing[0])
			repaymentsYesterdayArgCount++
		} else if len(nonMissing) > 1 {
			placeholders := make([]string, len(nonMissing))
			for i, lt := range nonMissing {
				placeholders[i] = fmt.Sprintf("$%d", repaymentsYesterdayArgCount)
				repaymentsYesterdayArgs = append(repaymentsYesterdayArgs, lt)
				repaymentsYesterdayArgCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.loan_type IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.loan_type IS NULL OR l.loan_type = '')")
		}

		if len(conditions) > 0 {
			repaymentsWhereYesterday += " AND (" + strings.Join(conditions, " OR ") + ")"
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
			repaymentsWhereYesterday += fmt.Sprintf(" AND l.verification_status = $%d", repaymentsYesterdayArgCount)
			repaymentsYesterdayArgs = append(repaymentsYesterdayArgs, nonMissing[0])
			repaymentsYesterdayArgCount++
		} else if len(nonMissing) > 1 {
			placeholders := make([]string, len(nonMissing))
			for i, vs := range nonMissing {
				placeholders[i] = fmt.Sprintf("$%d", repaymentsYesterdayArgCount)
				repaymentsYesterdayArgs = append(repaymentsYesterdayArgs, vs)
				repaymentsYesterdayArgCount++
			}
			conditions = append(conditions, fmt.Sprintf("l.verification_status IN (%s)", strings.Join(placeholders, ",")))
		}

		if includeMissing {
			conditions = append(conditions, "(l.verification_status IS NULL OR l.verification_status = '')")
		}

		if len(conditions) > 0 {
			repaymentsWhereYesterday += " AND (" + strings.Join(conditions, " OR ") + ")"
		}
	}

	if dpdMin, ok := filters["dpd_min"].(int); ok {
		repaymentsWhereYesterday += fmt.Sprintf(" AND l.current_dpd >= $%d", repaymentsYesterdayArgCount)
		repaymentsYesterdayArgs = append(repaymentsYesterdayArgs, dpdMin)
		repaymentsYesterdayArgCount++
	}

	if dpdMax, ok := filters["dpd_max"].(int); ok {
		repaymentsWhereYesterday += fmt.Sprintf(" AND l.current_dpd <= $%d", repaymentsYesterdayArgCount)
		repaymentsYesterdayArgs = append(repaymentsYesterdayArgs, dpdMax)
		repaymentsYesterdayArgCount++
	}

	// Apply Quiet Loans filter for yesterday's repayments as well so period
	// comparisons remain consistent when the toggle is active.
	if quietLoans, ok := filters["quiet_loans"].(bool); ok && quietLoans {
		repaymentsWhereYesterday += " AND (l.days_since_last_repayment >= 6 OR l.days_since_last_repayment IS NULL)"
	}

	repaymentsYesterdayQuery := `
				SELECT COALESCE(SUM(r.payment_amount), 0) as total_repayments_yesterday
			` + repaymentsWhereYesterday

	var totalRepaymentsYesterday float64
	err = r.db.QueryRow(repaymentsYesterdayQuery, repaymentsYesterdayArgs...).Scan(&totalRepaymentsYesterday)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate yesterday's repayments: %w", err)
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

	// Quiet Loans filter for missed repayments so that "missed today" metrics
	// are computed on the same quiet-loan subset as the table when enabled.
	if quietLoans, ok := filters["quiet_loans"].(bool); ok && quietLoans {
		missedQuery += " AND (l.days_since_last_repayment >= 6 OR l.days_since_last_repayment IS NULL)"
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
		"total_repayments_yesterday":    totalRepaymentsYesterday,
		"percentage_of_due_collected":   percentageDueCollected,
		"missed_repayments_today":       missedAmountToday,
		"missed_repayments_today_count": missedCountToday,
		"past_maturity_outstanding":     pastMaturityOutstanding,
	}

	return metrics, nil
}

// GetAllLoans retrieves all loans with pagination and filters
func (r *DashboardRepository) GetAllLoans(filters map[string]interface{}) ([]*models.AllLoan, int, error) {
	// NOTE: For the per-loan "repayments_today" field we now intentionally
	// ignore the selected period and always aggregate ONLY today's repayments
	// (DATE(r.payment_date) = CURRENT_DATE). This keeps the "Collection Today"
	// column in the Officer/Branch loan tables strictly "today-only" even when
	// the overall dashboard period is This Week/This Month/Last Month.

	// Per-loan repayments_today should always reflect today's collections only,
	// regardless of the current period filter.
	repaymentsDateCondition := "DATE(r.payment_date) = CURRENT_DATE"

	repaymentsJoin := fmt.Sprintf(`
		LEFT JOIN (
			SELECT
				r.loan_id,
				COALESCE(SUM(r.payment_amount), 0) AS repayments_in_period
			FROM repayments r
			WHERE r.is_reversed = false
				AND %s
			GROUP BY r.loan_id
		) rp ON rp.loan_id = l.loan_id
	`, repaymentsDateCondition)

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
				l.previous_dpd,
				(l.current_dpd - COALESCE(l.previous_dpd, 0)) AS dpd_change,
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
			l.verification_status,
			COALESCE(rp.repayments_in_period, 0) AS repayments_today
		FROM loans l
		JOIN officers o ON l.officer_id = o.officer_id
	` + repaymentsJoin + `
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

	// Quiet Loans filter: when enabled, restrict to loans with 6+ days since last
	// repayment or with no repayments at all. This is kept in sync with
	// GetLoansSummaryMetrics so that table rows, summary cards, and exports all
	// reflect the same filtered population.
	if quietLoans, ok := filters["quiet_loans"].(bool); ok && quietLoans {
		query += " AND (l.days_since_last_repayment >= 6 OR l.days_since_last_repayment IS NULL)"
		countQuery += " AND (l.days_since_last_repayment >= 6 OR l.days_since_last_repayment IS NULL)"
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
		var repaymentsToday sql.NullFloat64
		var daysSinceLastRepayment, repaymentDaysDueToday, businessDaysSinceDisbursement sql.NullInt64
		var previousDPD, dpdChange sql.NullInt64

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
			&previousDPD,
			&dpdChange,
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
			&repaymentsToday,
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
		if previousDPD.Valid {
			val := int(previousDPD.Int64)
			loan.PreviousDPD = &val
		}
		if dpdChange.Valid {
			val := int(dpdChange.Int64)
			loan.DPDChange = &val
		}
		if loanType.Valid {
			loan.LoanType = &loanType.String
		}
		if verificationStatus.Valid {
			loan.VerificationStatus = &verificationStatus.String
		}
		if repaymentsToday.Valid {
			val := repaymentsToday.Float64
			loan.RepaymentsToday = &val
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

// GetVerticalLeadMetrics retrieves aggregated loan metrics grouped by vertical
// lead name for the Credit Health by Branch "By Vertical Lead" view.
func (r *DashboardRepository) GetVerticalLeadMetrics(filters map[string]interface{}) ([]*models.VerticalLeadMetricsRow, error) {
	query := `
		SELECT
			COALESCE(NULLIF(l.vertical_lead_name, ''), 'Unassigned Vertical Lead') AS vertical_lead_name,
			COUNT(DISTINCT l.branch) AS branches,
			COUNT(DISTINCT l.officer_id) AS active_los,
			COUNT(*) AS loans,
			COALESCE(SUM(l.total_outstanding), 0) AS outstanding,
			COALESCE(AVG(l.current_dpd), 0) AS avg_dpd,
			COALESCE(MAX(l.max_dpd_ever), 0) AS max_dpd,
			COUNT(CASE WHEN l.current_dpd = 0 THEN 1 END) AS dpd0,
			COUNT(CASE WHEN l.current_dpd BETWEEN 1 AND 6 THEN 1 END) AS dpd1_6,
			COUNT(CASE WHEN l.current_dpd BETWEEN 7 AND 14 THEN 1 END) AS dpd7_14,
			COUNT(CASE WHEN l.current_dpd BETWEEN 14 AND 21 THEN 1 END) AS dpd14_21,
			COUNT(CASE WHEN l.current_dpd > 21 THEN 1 END) AS dpd21_plus,
			COUNT(CASE WHEN COALESCE(l.days_since_last_repayment, 0) > 7 THEN 1 END) AS quiet,
			COALESCE(SUM(CASE WHEN COALESCE(l.days_since_last_repayment, 0) > 7 THEN l.total_outstanding ELSE 0 END), 0) AS quiet_value
		FROM loans l
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 1

	// Optional filters (kept consistent with GetBranches where relevant)
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

	query += `
		GROUP BY vertical_lead_name
		ORDER BY vertical_lead_name
	`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []*models.VerticalLeadMetricsRow{}
	for rows.Next() {
		row := &models.VerticalLeadMetricsRow{}
		if err := rows.Scan(
			&row.VerticalLeadName,
			&row.Branches,
			&row.ActiveLOs,
			&row.Loans,
			&row.Outstanding,
			&row.AvgDPD,
			&row.MaxDPD,
			&row.DPD0,
			&row.DPD1to6,
			&row.DPD7to14,
			&row.DPD14to21,
			&row.DPD21Plus,
			&row.Quiet,
			&row.QuietValue,
		); err != nil {
			return nil, err
		}

		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
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

// GetAgentActivitySummary computes aggregated counts of officers in the
// Collections Control Centre Agent Activity categories over a rolling 7-day
// window (past 7 days including today). It respects the same core filters
// (branch, region, channel, wave, loan_type) used by other collections
// endpoints and applies the standard officer user_type filter. All date
// comparisons are based on DATE(r.payment_date).
func (r *DashboardRepository) GetAgentActivitySummary(filters map[string]interface{}) (*models.AgentActivitySummary, error) {
	query := `
			WITH filtered_loans AS (
				SELECT DISTINCT
					l.loan_id,
					l.officer_id
				FROM loans l
				JOIN officers o ON l.officer_id = o.officer_id
				WHERE (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
		`

	args := []interface{}{}
	argCount := 1

	// Apply filters (branch, region, channel, wave, loan_type) similar to
	// GetOfficerCollectionsLeaderboard's loanQuery.
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

	if wave, ok := filters["wave"].(string); ok && strings.TrimSpace(wave) != "" {
		waves := strings.Split(wave, ",")
		if len(waves) == 1 {
			query += fmt.Sprintf(" AND l.wave = $%d", argCount)
			args = append(args, strings.TrimSpace(waves[0]))
			argCount++
		} else {
			placeholders := make([]string, len(waves))
			for i, w := range waves {
				placeholders[i] = fmt.Sprintf("$%d", argCount)
				args = append(args, strings.TrimSpace(w))
				argCount++
			}
			query += fmt.Sprintf(" AND l.wave IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	if loanType, ok := filters["loan_type"].(string); ok && loanType != "" {
		loanTypes := strings.Split(loanType, ",")
		if len(loanTypes) == 1 {
			query += fmt.Sprintf(" AND l.loan_type = $%d", argCount)
			args = append(args, strings.TrimSpace(loanTypes[0]))
			argCount++
		} else {
			placeholders := make([]string, len(loanTypes))
			for i, lt := range loanTypes {
				placeholders[i] = fmt.Sprintf("$%d", argCount)
				args = append(args, strings.TrimSpace(lt))
				argCount++
			}
			query += fmt.Sprintf(" AND l.loan_type IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	query += `
			),
			officer_base AS (
				SELECT DISTINCT officer_id
				FROM filtered_loans
			),
			repayments_7d AS (
				SELECT
					fl.officer_id,
					DATE(r.payment_date) AS payment_date,
					SUM(r.payment_amount) AS amount
				FROM filtered_loans fl
				JOIN repayments r ON r.loan_id = fl.loan_id
				WHERE r.is_reversed = FALSE
					AND DATE(r.payment_date) >= (CURRENT_DATE - INTERVAL '6 days')
					AND DATE(r.payment_date) <= CURRENT_DATE
				GROUP BY fl.officer_id, DATE(r.payment_date)
			),
			per_officer AS (
				SELECT
					ob.officer_id,
					COALESCE(SUM(r7.amount), 0) AS total_7d,
					COALESCE(SUM(r7.amount) FILTER (
						WHERE r7.payment_date >= (CURRENT_DATE - INTERVAL '6 days')
							AND r7.payment_date <= (CURRENT_DATE - INTERVAL '3 days')
					), 0) AS amount_first4,
					COALESCE(SUM(r7.amount) FILTER (
						WHERE r7.payment_date >= (CURRENT_DATE - INTERVAL '2 days')
							AND r7.payment_date <= CURRENT_DATE
					), 0) AS amount_last3,
							-- Count of distinct business days (Mon-Fri) with at least one collection
							-- in the 7-calendar-day window. Weekends are excluded from this count but
							-- their repayments are still included in total_7d/amount_first4/amount_last3.
							COALESCE(COUNT(DISTINCT r7.payment_date) FILTER (
								WHERE r7.payment_date >= (CURRENT_DATE - INTERVAL '6 days')
									AND r7.payment_date <= CURRENT_DATE
									AND EXTRACT(ISODOW FROM r7.payment_date) BETWEEN 1 AND 5
							), 0) AS days_with_collection_7d,
					COALESCE(COUNT(DISTINCT r7.payment_date) FILTER (
						WHERE r7.payment_date = CURRENT_DATE
					), 0) AS days_with_collection_today
				FROM officer_base ob
				LEFT JOIN repayments_7d r7 ON ob.officer_id = r7.officer_id
				GROUP BY ob.officer_id
			)
			SELECT
				COALESCE(COUNT(*) FILTER (WHERE total_7d = 0), 0) AS critical_no_collection_count,
				COALESCE(COUNT(*) FILTER (WHERE amount_first4 > 0 AND amount_last3 = 0), 0) AS stopped_collecting_count,
				COALESCE(COUNT(*) FILTER (
					WHERE amount_first4 > 0
						AND amount_last3 > 0
						AND amount_last3 < 0.3 * amount_first4
				), 0) AS severe_decline_count,
				COALESCE(COUNT(*) FILTER (
					WHERE days_with_collection_7d >= 5
						AND days_with_collection_today = 0
				), 0) AS not_yet_started_today_count,
				COALESCE(COUNT(*) FILTER (
					WHERE amount_first4 > 0
						AND amount_last3 > 1.5 * amount_first4
				), 0) AS strong_growth_count,
				COALESCE(COUNT(*) FILTER (WHERE days_with_collection_today > 0), 0) AS started_today_count
			FROM per_officer;
		`

	row := r.db.QueryRow(query, args...)
	summary := &models.AgentActivitySummary{}
	if err := row.Scan(
		&summary.CriticalNoCollectionCount,
		&summary.StoppedCollectingCount,
		&summary.SevereDeclineCount,
		&summary.NotYetStartedTodayCount,
		&summary.StrongGrowthCount,
		&summary.StartedTodayCount,
	); err != nil {
		return nil, err
	}

	return summary, nil
}

// GetAgentActivityDetail returns per-officer 7-day repayment activity for a
// specific Agent Activity category. It reuses the same 7-day window and
// category logic as GetAgentActivitySummary but returns detailed rows instead
// of aggregate counts. The same core filters (branch, region, channel, wave,
// loan_type) are applied.
func (r *DashboardRepository) GetAgentActivityDetail(filters map[string]interface{}, category string) ([]*models.AgentActivityDetailRow, error) {
	query := `
				WITH filtered_loans AS (
					SELECT DISTINCT
						l.loan_id,
						l.officer_id,
						l.branch,
						l.region
					FROM loans l
					JOIN officers o ON l.officer_id = o.officer_id
					WHERE (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
			`

	args := []interface{}{}
	argCount := 1

	// Apply filters (branch, region, channel, wave, loan_type) similar to
	// GetAgentActivitySummary's loanQuery.
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

	if wave, ok := filters["wave"].(string); ok && strings.TrimSpace(wave) != "" {
		waves := strings.Split(wave, ",")
		if len(waves) == 1 {
			query += fmt.Sprintf(" AND l.wave = $%d", argCount)
			args = append(args, strings.TrimSpace(waves[0]))
			argCount++
		} else {
			placeholders := make([]string, len(waves))
			for i, w := range waves {
				placeholders[i] = fmt.Sprintf("$%d", argCount)
				args = append(args, strings.TrimSpace(w))
				argCount++
			}
			query += fmt.Sprintf(" AND l.wave IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	if loanType, ok := filters["loan_type"].(string); ok && loanType != "" {
		loanTypes := strings.Split(loanType, ",")
		if len(loanTypes) == 1 {
			query += fmt.Sprintf(" AND l.loan_type = $%d", argCount)
			args = append(args, strings.TrimSpace(loanTypes[0]))
			argCount++
		} else {
			placeholders := make([]string, len(loanTypes))
			for i, lt := range loanTypes {
				placeholders[i] = fmt.Sprintf("$%d", argCount)
				args = append(args, strings.TrimSpace(lt))
				argCount++
			}
			query += fmt.Sprintf(" AND l.loan_type IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	query += `
				),
				officer_base AS (
					SELECT DISTINCT officer_id
					FROM filtered_loans
				),
				repayments_7d AS (
					SELECT
						fl.officer_id,
						DATE(r.payment_date) AS payment_date,
						SUM(r.payment_amount) AS amount
					FROM filtered_loans fl
					JOIN repayments r ON r.loan_id = fl.loan_id
					WHERE r.is_reversed = FALSE
						AND DATE(r.payment_date) >= (CURRENT_DATE - INTERVAL '6 days')
						AND DATE(r.payment_date) <= CURRENT_DATE
					GROUP BY fl.officer_id, DATE(r.payment_date)
				),
				per_officer AS (
					SELECT
						ob.officer_id,
						COALESCE(SUM(r7.amount), 0) AS total_7d,
						COALESCE(SUM(r7.amount) FILTER (
							WHERE r7.payment_date >= (CURRENT_DATE - INTERVAL '6 days')
								AND r7.payment_date <= (CURRENT_DATE - INTERVAL '3 days')
						), 0) AS amount_first4,
						COALESCE(SUM(r7.amount) FILTER (
							WHERE r7.payment_date >= (CURRENT_DATE - INTERVAL '2 days')
								AND r7.payment_date <= CURRENT_DATE
						), 0) AS amount_last3,
						-- Count of distinct business days (Mon-Fri) with at least one collection
						-- in the 7-calendar-day window. Weekends are excluded from this count but
						-- their repayments are still included in total_7d/amount_first4/amount_last3.
						COALESCE(COUNT(DISTINCT r7.payment_date) FILTER (
							WHERE r7.payment_date >= (CURRENT_DATE - INTERVAL '6 days')
								AND r7.payment_date <= CURRENT_DATE
								AND EXTRACT(ISODOW FROM r7.payment_date) BETWEEN 1 AND 5
						), 0) AS days_with_collection_7d,
						COALESCE(COUNT(DISTINCT r7.payment_date) FILTER (
							WHERE r7.payment_date = CURRENT_DATE
						), 0) AS days_with_collection_today,
						COALESCE(SUM(r7.amount) FILTER (
							WHERE r7.payment_date = (CURRENT_DATE - INTERVAL '6 days')
						), 0) AS amount_5d_ago,
						COALESCE(SUM(r7.amount) FILTER (
							WHERE r7.payment_date = (CURRENT_DATE - INTERVAL '5 days')
						), 0) AS amount_4d_ago,
						COALESCE(SUM(r7.amount) FILTER (
							WHERE r7.payment_date = (CURRENT_DATE - INTERVAL '4 days')
						), 0) AS amount_3d_ago,
						COALESCE(SUM(r7.amount) FILTER (
							WHERE r7.payment_date = (CURRENT_DATE - INTERVAL '3 days')
						), 0) AS amount_2d_ago,
						COALESCE(SUM(r7.amount) FILTER (
							WHERE r7.payment_date = (CURRENT_DATE - INTERVAL '2 days')
						), 0) AS amount_2d_ago_exact,
						COALESCE(SUM(r7.amount) FILTER (
							WHERE r7.payment_date = (CURRENT_DATE - INTERVAL '1 day')
						), 0) AS amount_1d_ago,
						COALESCE(SUM(r7.amount) FILTER (
							WHERE r7.payment_date = CURRENT_DATE
						), 0) AS amount_today
					FROM officer_base ob
					LEFT JOIN repayments_7d r7 ON ob.officer_id = r7.officer_id
					GROUP BY ob.officer_id
				),
				officer_info AS (
					SELECT
						fl.officer_id,
						COALESCE(o.officer_name, '') AS officer_name,
						COALESCE(o.officer_email, '') AS officer_email,
						MODE() WITHIN GROUP (ORDER BY fl.branch) AS branch,
						MODE() WITHIN GROUP (ORDER BY fl.region) AS region
					FROM filtered_loans fl
					JOIN officers o ON fl.officer_id = o.officer_id
					GROUP BY fl.officer_id, o.officer_name, o.officer_email
				)
			SELECT
				po.officer_id,
				oi.officer_name,
				oi.officer_email,
				oi.branch,
				oi.region,
				CASE
							-- Repayment rate is based on business days only. There are always 5
							-- business days in any 7-calendar-day window, so we divide by 5.0
							-- instead of 7. Weekend repayments still contribute to total_7d and
							-- the per-day amounts but do not increase the denominator.
							WHEN po.days_with_collection_7d > 0 THEN (po.days_with_collection_7d::float / 5.0) * 100.0
					ELSE 0
					END AS repayment_rate,
					po.amount_5d_ago,
					po.amount_4d_ago,
					po.amount_3d_ago,
					po.amount_2d_ago,
					po.amount_2d_ago_exact,
					po.amount_1d_ago,
					po.amount_today,
					po.total_7d AS total_collected
			FROM per_officer po
			JOIN officer_info oi ON po.officer_id = oi.officer_id
		`

	// Apply category-specific filter using the same logic as GetAgentActivitySummary.
	switch category {
	case "critical_no_collection":
		query += " WHERE po.total_7d = 0"
	case "stopped_collecting":
		query += " WHERE po.amount_first4 > 0 AND po.amount_last3 = 0"
	case "severe_decline":
		query += " WHERE po.amount_first4 > 0 AND po.amount_last3 > 0 AND po.amount_last3 < 0.3 * po.amount_first4"
	case "not_yet_started_today":
		query += " WHERE po.days_with_collection_7d >= 5 AND po.days_with_collection_today = 0"
	case "strong_growth":
		query += " WHERE po.amount_first4 > 0 AND po.amount_last3 > 1.5 * po.amount_first4"
	case "started_today":
		query += " WHERE po.days_with_collection_today > 0"
	default:
		return nil, fmt.Errorf("unknown agent activity category: %s", category)
	}

	query += " ORDER BY po.total_7d DESC, oi.officer_name ASC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []*models.AgentActivityDetailRow{}
	for rows.Next() {
		row := &models.AgentActivityDetailRow{}
		if err := rows.Scan(
			&row.OfficerID,
			&row.OfficerName,
			&row.OfficerEmail,
			&row.Branch,
			&row.Region,
			&row.RepaymentRate,
			&row.Amount5DaysAgo,
			&row.Amount4DaysAgo,
			&row.Amount3DaysAgo,
			&row.Amount2DaysAgo,
			&row.Amount2DaysAgoExact,
			&row.Amount1DayAgo,
			&row.AmountToday,
			&row.TotalCollected,
		); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// GetRepaymentWatchOfficers computes per-officer Wave 2 repayment performance for the
// Repayment Watch view. It focuses on Wave 2 loans that are currently OPEN or
// PAST_MATURITY and counts non-reversed repayments made today. It respects the same
// branch/region/channel/wave/loan_type filters used elsewhere in the Collections
// Control Centre.
func (r *DashboardRepository) GetRepaymentWatchOfficers(filters map[string]interface{}) ([]*models.RepaymentWatchOfficerRow, error) {
	query := `
				SELECT
					l.officer_id,
					COALESCE(o.officer_name, '') AS officer_name,
					COALESCE(o.officer_email, '') AS officer_email,
					MODE() WITHIN GROUP (ORDER BY l.branch) AS branch,
					MODE() WITHIN GROUP (ORDER BY l.region) AS region,
					COUNT(DISTINCT l.loan_id) AS total_wave2_open_loans,
					COUNT(DISTINCT r.loan_id) AS loans_with_repayment_today,
					COALESCE(SUM(r.payment_amount), 0) AS amount_collected_today
				FROM loans l
				JOIN officers o ON l.officer_id = o.officer_id
				LEFT JOIN repayments r ON r.loan_id = l.loan_id
					AND r.is_reversed = FALSE
					AND r.payment_date::date = CURRENT_DATE
				WHERE 1=1
					AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
					AND l.django_status IN ('OPEN', 'PAST_MATURITY')
			`

	args := []interface{}{}
	argCount := 1

	// Apply filters (branch, region, channel, wave, loan_type) similar to
	// GetOfficerCollectionsLeaderboard, but with a Wave 2 default when no wave
	// filter is provided.
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

	if wave, ok := filters["wave"].(string); ok && strings.TrimSpace(wave) != "" {
		// Support comma-separated waves for completeness
		waves := strings.Split(wave, ",")
		if len(waves) == 1 {
			query += fmt.Sprintf(" AND l.wave = $%d", argCount)
			args = append(args, strings.TrimSpace(waves[0]))
			argCount++
		} else {
			placeholders := make([]string, len(waves))
			for i, w := range waves {
				placeholders[i] = fmt.Sprintf("$%d", argCount)
				args = append(args, strings.TrimSpace(w))
				argCount++
			}
			query += fmt.Sprintf(" AND l.wave IN (%s)", strings.Join(placeholders, ", "))
		}
	} else {
		// Default focus: Wave 2 loans only (case-insensitive, supports "wave2" and "wave 2").
		query += " AND LOWER(COALESCE(l.wave, '')) IN ('wave2', 'wave 2')"
	}

	if loanType, ok := filters["loan_type"].(string); ok && loanType != "" {
		loanTypes := strings.Split(loanType, ",")
		if len(loanTypes) == 1 {
			query += fmt.Sprintf(" AND l.loan_type = $%d", argCount)
			args = append(args, strings.TrimSpace(loanTypes[0]))
			argCount++
		} else {
			placeholders := make([]string, len(loanTypes))
			for i, lt := range loanTypes {
				placeholders[i] = fmt.Sprintf("$%d", argCount)
				args = append(args, strings.TrimSpace(lt))
				argCount++
			}
			query += fmt.Sprintf(" AND l.loan_type IN (%s)", strings.Join(placeholders, ", "))
		}
	}

	query += `
				GROUP BY
					l.officer_id,
					o.officer_name,
					o.officer_email
				HAVING COUNT(DISTINCT l.loan_id) > 0
				ORDER BY COUNT(DISTINCT l.loan_id) DESC
			`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []*models.RepaymentWatchOfficerRow{}
	for rows.Next() {
		row := &models.RepaymentWatchOfficerRow{}
		if err := rows.Scan(
			&row.OfficerID,
			&row.OfficerName,
			&row.OfficerEmail,
			&row.Branch,
			&row.Region,
			&row.TotalWave2OpenLoans,
			&row.LoansWithRepaymentToday,
			&row.AmountCollectedToday,
		); err != nil {
			return nil, err
		}

		if row.TotalWave2OpenLoans > 0 && row.LoansWithRepaymentToday > 0 {
			row.RepaymentRate = (float64(row.LoansWithRepaymentToday) / float64(row.TotalWave2OpenLoans)) * 100
		} else {
			row.RepaymentRate = 0
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
				COUNT(*) AS repayments_count,
				-- Repayment type breakdown (normalised using UPPER(TRIM(payment_method)))
				COALESCE(SUM(CASE WHEN UPPER(TRIM(r.payment_method)) = 'AGENT_DEBIT' THEN r.payment_amount END), 0) AS agent_debit_amount,
				COALESCE(SUM(CASE WHEN UPPER(TRIM(r.payment_method)) = 'TRANSFER' THEN r.payment_amount END), 0) AS transfer_amount,
				COALESCE(SUM(CASE WHEN UPPER(TRIM(r.payment_method)) = 'ESCROW_DEBIT' THEN r.payment_amount END), 0) AS escrow_debit_amount,
				COALESCE(SUM(CASE
					WHEN UPPER(TRIM(r.payment_method)) NOT IN ('AGENT_DEBIT', 'TRANSFER', 'ESCROW_DEBIT')
						OR r.payment_method IS NULL
						OR TRIM(r.payment_method) = ''
					THEN r.payment_amount
				END), 0) AS other_repayments_amount
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
		if err := rows.Scan(
			&point.Date,
			&point.CollectedAmount,
			&point.RepaymentsCount,
			&point.AgentDebitAmount,
			&point.TransferAmount,
			&point.EscrowDebitAmount,
			&point.OtherRepaymentsAmount,
		); err != nil {
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
	// Regions should include all configured regions, even if there are
	// currently no loans in that region yet. To achieve this we take the
	// union of regions present on loans and regions configured on officers.
	//
	// This ensures newer regions/verticals (e.g. "Saphire") that already
	// exist on officers but don't yet have active loans still appear in the
	// Regions filter dropdown.
	query := `
		SELECT DISTINCT region
		FROM (
			SELECT l.region AS region
			FROM loans l
			INNER JOIN officers o ON l.officer_id = o.officer_id
			WHERE (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)

			UNION

			SELECT o.region AS region
			FROM officers o
			WHERE (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
		) regions
		WHERE region IS NOT NULL AND region != ''
		ORDER BY region`

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

// GetVerticalLeadNames returns the distinct vertical lead names used on loans.
//
// It includes a synthetic "Unassigned Vertical Lead" bucket for loans where
// vertical_lead_name is NULL or blank, so that these loans are still
// represented in the UI.
func (r *DashboardRepository) GetVerticalLeadNames() ([]string, error) {
	query := `
		SELECT DISTINCT
			COALESCE(NULLIF(l.vertical_lead_name, ''), 'Unassigned Vertical Lead') AS vertical_lead_name
		FROM loans l
		ORDER BY vertical_lead_name`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch vertical lead names: %w", err)
	}
	defer rows.Close()

	names := []string{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan vertical lead name: %w", err)
		}
		names = append(names, name)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating vertical lead name rows: %w", err)
	}

	return names, nil
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
