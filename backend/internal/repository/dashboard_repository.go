package repository

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/seeds-metrics/analytics-backend/internal/models"
)

// DashboardRepository handles dashboard data queries
type DashboardRepository struct {
	db *sql.DB
}

// NewDashboardRepository creates a new dashboard repository
func NewDashboardRepository(db *sql.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
}

// RecalculateAllLoanFields triggers recalculation of all computed fields for all loans
// This is done by updating the updated_at timestamp, which triggers the update_loan_computed_fields() function
func (r *DashboardRepository) RecalculateAllLoanFields() (int64, error) {
	query := `UPDATE loans SET updated_at = CURRENT_TIMESTAMP WHERE 1=1`

	result, err := r.db.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("failed to recalculate loan fields: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected, nil
}

// GetPortfolioLoanMetrics retrieves loan-level aggregated metrics for portfolio calculations
func (r *DashboardRepository) GetPortfolioLoanMetrics(filters map[string]interface{}) (*models.PortfolioLoanMetrics, error) {
	query := `
		SELECT
			-- Active vs Inactive Loans
			COUNT(CASE WHEN (principal_outstanding + interest_outstanding + fees_outstanding) > 2000
				AND days_since_last_repayment < 6 THEN 1 END) as active_loans_count,
			COALESCE(SUM(CASE WHEN (principal_outstanding + interest_outstanding + fees_outstanding) > 2000
				AND days_since_last_repayment < 6
				THEN (principal_outstanding + interest_outstanding + fees_outstanding) END), 0) as active_loans_volume,
			COUNT(CASE WHEN (principal_outstanding + interest_outstanding + fees_outstanding) <= 2000
				OR days_since_last_repayment > 5 THEN 1 END) as inactive_loans_count,
			COALESCE(SUM(CASE WHEN (principal_outstanding + interest_outstanding + fees_outstanding) <= 2000
				OR days_since_last_repayment > 5
				THEN (principal_outstanding + interest_outstanding + fees_outstanding) END), 0) as inactive_loans_volume,

			-- ROT (Risk of Termination) Analysis
			COUNT(CASE WHEN (CURRENT_DATE - disbursement_date::date) < 7 AND current_dpd > 4 THEN 1 END) as early_rot_count,
			COALESCE(SUM(CASE WHEN (CURRENT_DATE - disbursement_date::date) < 7 AND current_dpd > 4
				THEN (principal_outstanding + interest_outstanding + fees_outstanding) END), 0) as early_rot_volume,
			COUNT(CASE WHEN (CURRENT_DATE - disbursement_date::date) >= 7 AND current_dpd > 4 THEN 1 END) as late_rot_count,
			COALESCE(SUM(CASE WHEN (CURRENT_DATE - disbursement_date::date) >= 7 AND current_dpd > 4
				THEN (principal_outstanding + interest_outstanding + fees_outstanding) END), 0) as late_rot_volume,

			-- Portfolio Repayment Behavior Metrics (only active loans)
			COALESCE(AVG(CASE WHEN (principal_outstanding + interest_outstanding + fees_outstanding) > 2000
				THEN current_dpd END), 0) as avg_days_past_due,
			COALESCE(AVG(CASE WHEN (principal_outstanding + interest_outstanding + fees_outstanding) > 2000
				THEN timeliness_score END), 0) as avg_timeliness_score
		FROM loans
		WHERE UPPER(status) = 'ACTIVE'
	`

	args := []interface{}{}
	argCount := 1

	// Apply wave filter
	if wave, ok := filters["wave"].(string); ok && wave != "" {
		query += fmt.Sprintf(" AND wave = $%d", argCount)
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
		WHERE l.current_dpd >= 15
			AND UPPER(l.status) = 'ACTIVE'
			AND ls.due_date <= CURRENT_DATE
			AND ls.payment_status IN ('Pending', 'Partial', 'Overdue')
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
						WHEN loan_term_days > 0 THEN
							(principal_outstanding + interest_outstanding + fees_outstanding) *
							LEAST(1.0, GREATEST(0.0, (CURRENT_DATE - disbursement_date::date)::float / loan_term_days::float))
						ELSE
							(principal_outstanding + interest_outstanding + fees_outstanding)
					END
				), 0) as estimated_actual_overdue
			FROM loans
			WHERE current_dpd >= 15
				AND UPPER(status) = 'ACTIVE'
		`

		fallbackArgs := []interface{}{}
		fallbackArgCount := 1

		// Apply wave filter to fallback query
		if wave, ok := filters["wave"].(string); ok && wave != "" {
			fallbackQuery += fmt.Sprintf(" AND wave = $%d", fallbackArgCount)
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

// GetOfficers retrieves all officers with their raw metrics
func (r *DashboardRepository) GetOfficers(filters map[string]interface{}) ([]*models.DashboardOfficerMetrics, error) {
	query := `
		SELECT
			o.officer_id,
			o.officer_name,
			o.region,
			o.branch,
			COALESCE(o.primary_channel, '') as primary_channel,
			-- Raw metrics (to be aggregated from loans)
			COALESCE(SUM(CASE WHEN l.fimr_tagged THEN 1 ELSE 0 END), 0) as first_miss,
			COALESCE(COUNT(DISTINCT l.loan_id), 0) as disbursed,
			COALESCE(SUM(CASE WHEN l.current_dpd BETWEEN 1 AND 6 THEN l.principal_outstanding ELSE 0 END), 0) as dpd1to6_bal,
			COALESCE(SUM(l.principal_outstanding + l.interest_outstanding + l.fees_outstanding), 0) as amount_due_7d,
			COALESCE(SUM(CASE WHEN l.current_dpd BETWEEN 7 AND 30 THEN l.principal_outstanding ELSE 0 END), 0) as moved_to_7to30,
			COALESCE(SUM(CASE WHEN l.current_dpd BETWEEN 1 AND 6 THEN l.principal_outstanding ELSE 0 END), 0) as prev_dpd1to6_bal,
			COALESCE(SUM(l.total_fees_paid), 0) as fees_collected,
			COALESCE(SUM(l.fee_amount), 0) as fees_due,
			COALESCE(SUM(l.total_interest_paid), 0) as interest_collected,
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
			COALESCE(AVG(CASE WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) > 2000 THEN (CURRENT_DATE - l.disbursement_date::date) ELSE NULL END), 0) as avg_loan_age,
			COALESCE(COUNT(CASE WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) > 2000 THEN 1 ELSE NULL END), 0) as active_loans_count
		FROM officers o
		LEFT JOIN loans l ON o.officer_id = l.officer_id
		WHERE 1=1
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
		query += fmt.Sprintf(" AND o.region = $%d", argCount)
		args = append(args, region)
		argCount++
	}

	if channel, ok := filters["channel"].(string); ok && channel != "" {
		query += fmt.Sprintf(" AND o.primary_channel = $%d", argCount)
		args = append(args, channel)
		argCount++
	}

	if wave, ok := filters["wave"].(string); ok && wave != "" {
		query += fmt.Sprintf(" AND l.wave = $%d", argCount)
		args = append(args, wave)
		argCount++
	}

	query += " GROUP BY o.officer_id, o.officer_name, o.region, o.branch, o.primary_channel"

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

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	officers := []*models.DashboardOfficerMetrics{}
	for rows.Next() {
		officer := &models.DashboardOfficerMetrics{
			RawMetrics: &models.RawMetrics{},
		}

		err := rows.Scan(
			&officer.OfficerID,
			&officer.Name,
			&officer.Region,
			&officer.Branch,
			&officer.Channel,
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

		officers = append(officers, officer)
	}

	return officers, nil
}

// GetOfficerByID retrieves a single officer by ID
func (r *DashboardRepository) GetOfficerByID(officerID string) (*models.DashboardOfficerMetrics, error) {
	query := `
		SELECT
			o.officer_id,
			o.officer_name,
			o.region,
			o.branch,
			COALESCE(o.primary_channel, '') as primary_channel,
			COALESCE(SUM(CASE WHEN l.fimr_tagged THEN 1 ELSE 0 END), 0) as first_miss,
			COALESCE(COUNT(DISTINCT l.loan_id), 0) as disbursed,
			COALESCE(SUM(CASE WHEN l.current_dpd BETWEEN 1 AND 6 THEN l.principal_outstanding ELSE 0 END), 0) as dpd1to6_bal,
			COALESCE(SUM(l.principal_outstanding + l.interest_outstanding + l.fees_outstanding), 0) as amount_due_7d,
			COALESCE(SUM(CASE WHEN l.current_dpd BETWEEN 7 AND 30 THEN l.principal_outstanding ELSE 0 END), 0) as moved_to_7to30,
			COALESCE(SUM(CASE WHEN l.current_dpd BETWEEN 1 AND 6 THEN l.principal_outstanding ELSE 0 END), 0) as prev_dpd1to6_bal,
			COALESCE(SUM(l.total_fees_paid), 0) as fees_collected,
			COALESCE(SUM(l.fee_amount), 0) as fees_due,
			COALESCE(SUM(l.total_interest_paid), 0) as interest_collected,
			COALESCE(SUM(CASE WHEN l.current_dpd >= 15 THEN l.principal_outstanding ELSE 0 END), 0) as overdue_15d,
			COALESCE(SUM(l.principal_outstanding), 0) as total_portfolio,
			COALESCE(SUM(l.principal_outstanding), 0) as par15_mid_month,
			0 as waivers,
			0 as backdated,
			0 as entries,
			0 as reversals,
			false as had_float_gap
		FROM officers o
		LEFT JOIN loans l ON o.officer_id = l.officer_id
		WHERE o.officer_id = $1
		GROUP BY o.officer_id, o.officer_name, o.region, o.branch, o.primary_channel
	`

	officer := &models.DashboardOfficerMetrics{
		RawMetrics: &models.RawMetrics{},
	}

	err := r.db.QueryRow(query, officerID).Scan(
		&officer.OfficerID,
		&officer.Name,
		&officer.Region,
		&officer.Branch,
		&officer.Channel,
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
	)

	if err != nil {
		return nil, err
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
			l.maturity_date as first_payment_due_date,
			l.current_dpd as days_since_due,
			l.loan_amount / l.loan_term_days as amount_due_1st_installment,
			l.total_principal_paid as amount_paid,
			l.principal_outstanding as outstanding_balance,
			l.current_dpd,
			l.channel,
			l.status,
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
		query += fmt.Sprintf(" AND l.region = $%d", argCount)
		args = append(args, region)
		argCount++
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
			&loan.FirstPaymentDueDate,
			&loan.DaysSinceDue,
			&loan.AmountDue1stInstallment,
			&loan.AmountPaid,
			&loan.OutstandingBalance,
			&loan.CurrentDPD,
			&loan.Channel,
			&loan.Status,
			&loan.FIMRTagged,
		)
		if err != nil {
			return nil, err
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
			l.principal_outstanding + l.interest_outstanding + l.fees_outstanding as amount_due,
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
		query += fmt.Sprintf(" AND l.region = $%d", argCount)
		args = append(args, region)
		argCount++
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
			l.channel,
			l.loan_amount,
			l.repayment_amount,
			l.disbursement_date,
			l.maturity_date,
			l.loan_term_days,
			l.current_dpd,
			l.principal_outstanding,
			l.interest_outstanding,
			l.fees_outstanding,
			l.principal_outstanding + l.interest_outstanding + l.fees_outstanding as total_outstanding,
			l.actual_outstanding,
			l.total_repayments,
			l.status,
			l.fimr_tagged,
			l.timeliness_score,
			l.repayment_health,
			l.days_since_last_repayment,
			l.repayment_delay_rate,
			l.wave
		FROM loans l
		JOIN officers o ON l.officer_id = o.officer_id
		WHERE 1=1
	`

	countQuery := `
		SELECT COUNT(*)
		FROM loans l
		JOIN officers o ON l.officer_id = o.officer_id
		WHERE 1=1
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
		query += fmt.Sprintf(" AND l.region = $%d", argCount)
		countQuery += fmt.Sprintf(" AND l.region = $%d", argCount)
		args = append(args, region)
		argCount++
	}

	if channel, ok := filters["channel"].(string); ok && channel != "" {
		query += fmt.Sprintf(" AND l.channel = $%d", argCount)
		countQuery += fmt.Sprintf(" AND l.channel = $%d", argCount)
		args = append(args, channel)
		argCount++
	}

	if status, ok := filters["status"].(string); ok && status != "" {
		query += fmt.Sprintf(" AND l.status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND l.status = $%d", argCount)
		args = append(args, status)
		argCount++
	}

	if wave, ok := filters["wave"].(string); ok && wave != "" {
		query += fmt.Sprintf(" AND l.wave = $%d", argCount)
		countQuery += fmt.Sprintf(" AND l.wave = $%d", argCount)
		args = append(args, wave)
		argCount++
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
		var customerPhone, officerID sql.NullString
		var repaymentAmount, timelinessScore, repaymentHealth, repaymentDelayRate sql.NullFloat64
		var daysSinceLastRepayment sql.NullInt64

		err := rows.Scan(
			&loan.LoanID,
			&loan.CustomerName,
			&customerPhone,
			&officerID,
			&loan.OfficerName,
			&loan.Region,
			&loan.Branch,
			&loan.Channel,
			&loan.LoanAmount,
			&repaymentAmount,
			&loan.DisbursementDate,
			&loan.MaturityDate,
			&loan.LoanTermDays,
			&loan.CurrentDPD,
			&loan.PrincipalOutstanding,
			&loan.InterestOutstanding,
			&loan.FeesOutstanding,
			&loan.TotalOutstanding,
			&loan.ActualOutstanding,
			&loan.TotalRepayments,
			&loan.Status,
			&loan.FIMRTagged,
			&timelinessScore,
			&repaymentHealth,
			&daysSinceLastRepayment,
			&repaymentDelayRate,
			&loan.Wave,
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
		if repaymentAmount.Valid {
			val := repaymentAmount.Float64
			loan.RepaymentAmount = &val
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
			(l.principal_outstanding + l.interest_outstanding + l.fees_outstanding)::float as total_outstanding,
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
					WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) >= 5000000 THEN 30
					WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) >= 3000000 THEN 25
					WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) >= 2000000 THEN 20
					WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) >= 1000000 THEN 15
					WHEN (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) >= 500000 THEN 10
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
			AND (l.current_dpd > 0 OR l.fimr_tagged = true OR (l.principal_outstanding + l.interest_outstanding + l.fees_outstanding) > 0)
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
			COUNT(DISTINCT l.officer_id) as total_officers
		FROM loans l
		WHERE 1=1
	`

	args := []interface{}{}
	argCount := 1

	// Apply filters
	if region, ok := filters["region"].(string); ok && region != "" {
		query += fmt.Sprintf(" AND l.region = $%d", argCount)
		args = append(args, region)
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

// GetFilterOptions retrieves filter dropdown options
func (r *DashboardRepository) GetFilterOptions(filterType string, filters map[string]interface{}) (interface{}, error) {
	switch filterType {
	case "branches":
		return r.getBranches(filters)
	case "regions":
		return r.getRegions()
	case "channels":
		return r.getChannels()
	case "officers":
		return r.getOfficerOptions(filters)
	default:
		return nil, fmt.Errorf("unknown filter type: %s", filterType)
	}
}

func (r *DashboardRepository) getBranches(filters map[string]interface{}) ([]string, error) {
	query := "SELECT DISTINCT branch FROM loans WHERE 1=1"
	args := []interface{}{}
	argCount := 1

	if region, ok := filters["region"].(string); ok && region != "" {
		query += fmt.Sprintf(" AND region = $%d", argCount)
		args = append(args, region)
		argCount++
	}

	query += " ORDER BY branch"

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

func (r *DashboardRepository) getRegions() ([]string, error) {
	query := "SELECT DISTINCT region FROM loans ORDER BY region"

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

func (r *DashboardRepository) getChannels() ([]string, error) {
	query := "SELECT DISTINCT channel FROM loans ORDER BY channel"

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

func (r *DashboardRepository) getOfficerOptions(filters map[string]interface{}) ([]*models.OfficerOption, error) {
	query := "SELECT DISTINCT officer_id, officer_name, branch, region FROM loans WHERE 1=1"
	args := []interface{}{}
	argCount := 1

	if branch, ok := filters["branch"].(string); ok && branch != "" {
		query += fmt.Sprintf(" AND branch = $%d", argCount)
		args = append(args, branch)
		argCount++
	}

	if region, ok := filters["region"].(string); ok && region != "" {
		query += fmt.Sprintf(" AND region = $%d", argCount)
		args = append(args, region)
		argCount++
	}

	query += " ORDER BY officer_name"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	officers := []*models.OfficerOption{}
	for rows.Next() {
		officer := &models.OfficerOption{}
		if err := rows.Scan(&officer.OfficerID, &officer.Name, &officer.Branch, &officer.Region); err != nil {
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
