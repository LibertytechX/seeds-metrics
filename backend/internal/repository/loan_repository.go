package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/seeds-metrics/analytics-backend/internal/models"
	"github.com/seeds-metrics/analytics-backend/pkg/database"
)

type LoanRepository struct {
	db *database.DB
}

func NewLoanRepository(db *database.DB) *LoanRepository {
	return &LoanRepository{db: db}
}

// Create inserts a new loan (ETL fields only)
// Skips if loan_id already exists (ON CONFLICT DO NOTHING)
func (r *LoanRepository) Create(ctx context.Context, input *models.LoanInput) error {
	query := `
		INSERT INTO loans (
			loan_id, customer_id, customer_name, customer_phone,
			officer_id, officer_name, officer_phone,
			region, branch, state,
			loan_amount, repayment_amount, disbursement_date, first_payment_due_date, maturity_date, loan_term_days,
			interest_rate, fee_amount,
			channel, channel_partner,
			status, closed_date, wave,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, COALESCE($23, 'Wave 2'),
			NOW(), NOW()
		)
		ON CONFLICT (loan_id) DO NOTHING
	`

	disbursementDate, err := time.Parse("2006-01-02", input.DisbursementDate)
	if err != nil {
		return fmt.Errorf("invalid disbursement_date format: %w", err)
	}

	var firstPaymentDueDate *time.Time
	if input.FirstPaymentDueDate != nil {
		parsed, err := time.Parse("2006-01-02", *input.FirstPaymentDueDate)
		if err != nil {
			return fmt.Errorf("invalid first_payment_due_date format: %w", err)
		}
		firstPaymentDueDate = &parsed
	}

	maturityDate, err := time.Parse("2006-01-02", input.MaturityDate)
	if err != nil {
		return fmt.Errorf("invalid maturity_date format: %w", err)
	}

	var closedDate *time.Time
	if input.ClosedDate != nil {
		parsed, err := time.Parse("2006-01-02", *input.ClosedDate)
		if err != nil {
			return fmt.Errorf("invalid closed_date format: %w", err)
		}
		closedDate = &parsed
	}

	_, err = r.db.ExecContext(ctx, query,
		input.LoanID, input.CustomerID, input.CustomerName, input.CustomerPhone,
		input.OfficerID, input.OfficerName, input.OfficerPhone,
		input.Region, input.Branch, input.State,
		input.LoanAmount, input.RepaymentAmount, disbursementDate, firstPaymentDueDate, maturityDate, input.LoanTermDays,
		input.InterestRate, input.FeeAmount,
		input.Channel, input.ChannelPartner,
		input.Status, closedDate, input.Wave,
	)

	return err
}

// GetByID retrieves a loan by ID
func (r *LoanRepository) GetByID(ctx context.Context, loanID string) (*models.Loan, error) {
	query := `
		SELECT
			loan_id, customer_id, customer_name, customer_phone,
			officer_id, officer_name, officer_phone,
			region, branch, state,
			loan_amount, repayment_amount, disbursement_date, maturity_date, loan_term_days,
			interest_rate, fee_amount,
			channel, channel_partner,
			status, closed_date,
			current_dpd, max_dpd_ever, first_payment_missed,
			first_payment_due_date, first_payment_received_date,
			principal_outstanding, interest_outstanding, fees_outstanding, total_outstanding,
			total_principal_paid, total_interest_paid, total_fees_paid,
			fimr_tagged, early_indicator_tagged,
			created_at, updated_at
		FROM loans
		WHERE loan_id = $1
	`

	var loan models.Loan
	err := r.db.QueryRowContext(ctx, query, loanID).Scan(
		&loan.LoanID, &loan.CustomerID, &loan.CustomerName, &loan.CustomerPhone,
		&loan.OfficerID, &loan.OfficerName, &loan.OfficerPhone,
		&loan.Region, &loan.Branch, &loan.State,
		&loan.LoanAmount, &loan.RepaymentAmount, &loan.DisbursementDate, &loan.MaturityDate, &loan.LoanTermDays,
		&loan.InterestRate, &loan.FeeAmount,
		&loan.Channel, &loan.ChannelPartner,
		&loan.Status, &loan.ClosedDate,
		&loan.CurrentDPD, &loan.MaxDPDEver, &loan.FirstPaymentMissed,
		&loan.FirstPaymentDueDate, &loan.FirstPaymentReceivedDate,
		&loan.PrincipalOutstanding, &loan.InterestOutstanding, &loan.FeesOutstanding, &loan.TotalOutstanding,
		&loan.TotalPrincipalPaid, &loan.TotalInterestPaid, &loan.TotalFeesPaid,
		&loan.FIMRTagged, &loan.EarlyIndicatorTagged,
		&loan.CreatedAt, &loan.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &loan, nil
}

// List retrieves loans with optional filters
func (r *LoanRepository) List(ctx context.Context, filter *models.LoanFilter) ([]*models.LoanDrilldown, int, error) {
	// Build query with filters
	query := `
		SELECT
			l.loan_id, l.customer_name, l.customer_phone, l.officer_name, l.branch,
			l.loan_amount, l.disbursement_date, l.current_dpd, l.total_outstanding,
			l.fimr_tagged, l.status
		FROM loans l
		INNER JOIN officers o ON l.officer_id = o.officer_id
		WHERE 1=1
			AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)
	`
	countQuery := `SELECT COUNT(*) FROM loans l
		INNER JOIN officers o ON l.officer_id = o.officer_id
		WHERE 1=1
			AND (o.user_type IN ('AGENT', 'AJO_AGENT', 'DMO_AGENT', 'MERCHANT', 'MERCHANT_AGENT', 'MICRO_SAVER', 'PERSONAL', 'PROSPER_AGENT', 'STAFF_AGENT') OR o.user_type IS NULL)`
	args := []interface{}{}
	argCount := 1

	// Apply filters
	if filter.OfficerID != nil {
		query += fmt.Sprintf(" AND l.officer_id = $%d", argCount)
		countQuery += fmt.Sprintf(" AND l.officer_id = $%d", argCount)
		args = append(args, *filter.OfficerID)
		argCount++
	}

	if filter.Region != nil {
		query += fmt.Sprintf(" AND l.region = $%d", argCount)
		countQuery += fmt.Sprintf(" AND l.region = $%d", argCount)
		args = append(args, *filter.Region)
		argCount++
	}

	if filter.Branch != nil {
		query += fmt.Sprintf(" AND l.branch = $%d", argCount)
		countQuery += fmt.Sprintf(" AND l.branch = $%d", argCount)
		args = append(args, *filter.Branch)
		argCount++
	}

	if filter.Status != nil {
		query += fmt.Sprintf(" AND l.status = $%d", argCount)
		countQuery += fmt.Sprintf(" AND l.status = $%d", argCount)
		args = append(args, *filter.Status)
		argCount++
	}

	if filter.FIMRTagged != nil {
		query += fmt.Sprintf(" AND l.fimr_tagged = $%d", argCount)
		countQuery += fmt.Sprintf(" AND l.fimr_tagged = $%d", argCount)
		args = append(args, *filter.FIMRTagged)
		argCount++
	}

	if filter.EarlyIndicator != nil {
		query += fmt.Sprintf(" AND l.early_indicator_tagged = $%d", argCount)
		countQuery += fmt.Sprintf(" AND l.early_indicator_tagged = $%d", argCount)
		args = append(args, *filter.EarlyIndicator)
		argCount++
	}

	if filter.MinDPD != nil {
		query += fmt.Sprintf(" AND l.current_dpd >= $%d", argCount)
		countQuery += fmt.Sprintf(" AND l.current_dpd >= $%d", argCount)
		args = append(args, *filter.MinDPD)
		argCount++
	}

	if filter.MaxDPD != nil {
		query += fmt.Sprintf(" AND l.current_dpd <= $%d", argCount)
		countQuery += fmt.Sprintf(" AND l.current_dpd <= $%d", argCount)
		args = append(args, *filter.MaxDPD)
		argCount++
	}

	// Get total count
	var total int
	err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Add pagination
	query += " ORDER BY disbursement_date DESC"
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
		argCount++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
	}

	// Execute query
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var loans []*models.LoanDrilldown
	for rows.Next() {
		var loan models.LoanDrilldown
		err := rows.Scan(
			&loan.LoanID, &loan.CustomerName, &loan.CustomerPhone, &loan.OfficerName, &loan.Branch,
			&loan.LoanAmount, &loan.DisbursementDate, &loan.CurrentDPD, &loan.TotalOutstanding,
			&loan.FIMRTagged, &loan.Status,
		)
		if err != nil {
			return nil, 0, err
		}
		loans = append(loans, &loan)
	}

	return loans, total, nil
}
