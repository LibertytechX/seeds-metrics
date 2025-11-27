package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/seeds-metrics/analytics-backend/internal/models"
	"github.com/seeds-metrics/analytics-backend/pkg/database"
)

type RepaymentRepository struct {
	db *database.DB
}

func NewRepaymentRepository(db *database.DB) *RepaymentRepository {
	return &RepaymentRepository{db: db}
}

// Create inserts a new repayment
func (r *RepaymentRepository) Create(ctx context.Context, input *models.RepaymentInput) error {
	query := `
		INSERT INTO repayments (
			repayment_id, loan_id, payment_date, payment_amount,
			principal_paid, interest_paid, fees_paid, penalty_paid,
			payment_method, payment_reference, payment_channel,
			dpd_at_payment, is_backdated, is_reversed,
			reversal_date, reversal_reason,
			waiver_amount, waiver_type, waiver_approved_by,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15, $16, $17, $18, $19,
			NOW(), NOW()
		)
		ON CONFLICT (repayment_id) DO UPDATE SET
			loan_id = EXCLUDED.loan_id,
			payment_date = EXCLUDED.payment_date,
			payment_amount = EXCLUDED.payment_amount,
			principal_paid = EXCLUDED.principal_paid,
			interest_paid = EXCLUDED.interest_paid,
			fees_paid = EXCLUDED.fees_paid,
			penalty_paid = EXCLUDED.penalty_paid,
			payment_method = EXCLUDED.payment_method,
			payment_reference = EXCLUDED.payment_reference,
			payment_channel = EXCLUDED.payment_channel,
			dpd_at_payment = EXCLUDED.dpd_at_payment,
			is_backdated = EXCLUDED.is_backdated,
			is_reversed = EXCLUDED.is_reversed,
			reversal_date = EXCLUDED.reversal_date,
			reversal_reason = EXCLUDED.reversal_reason,
			waiver_amount = EXCLUDED.waiver_amount,
			waiver_type = EXCLUDED.waiver_type,
			waiver_approved_by = EXCLUDED.waiver_approved_by,
			updated_at = NOW()
	`

	paymentDate, err := time.Parse("2006-01-02", input.PaymentDate)
	if err != nil {
		return fmt.Errorf("invalid payment_date format: %w", err)
	}

	var reversalDate *time.Time
	if input.ReversalDate != nil {
		parsed, err := time.Parse("2006-01-02", *input.ReversalDate)
		if err != nil {
			return fmt.Errorf("invalid reversal_date format: %w", err)
		}
		reversalDate = &parsed
	}

	// Validate payment amount
	totalPaid := input.PrincipalPaid.Add(input.InterestPaid).Add(input.FeesPaid).Add(input.PenaltyPaid)
	if !totalPaid.Equal(input.PaymentAmount) {
		return fmt.Errorf("payment_amount must equal sum of principal_paid + interest_paid + fees_paid + penalty_paid")
	}

	_, err = r.db.ExecContext(ctx, query,
		input.RepaymentID, input.LoanID, paymentDate, input.PaymentAmount,
		input.PrincipalPaid, input.InterestPaid, input.FeesPaid, input.PenaltyPaid,
		input.PaymentMethod, input.PaymentReference, input.PaymentChannel,
		input.DPDAtPayment, input.IsBackdated, input.IsReversed,
		reversalDate, input.ReversalReason,
		input.WaiverAmount, input.WaiverType, input.WaiverApprovedBy,
	)

	return err
}

// GetByID retrieves a repayment by ID
func (r *RepaymentRepository) GetByID(ctx context.Context, repaymentID string) (*models.Repayment, error) {
	query := `
		SELECT
			repayment_id, loan_id, payment_date, payment_amount,
			principal_paid, interest_paid, fees_paid, penalty_paid,
			payment_method, payment_reference, payment_channel,
			dpd_at_payment, is_backdated, is_reversed,
			reversal_date, reversal_reason,
			waiver_amount, waiver_type, waiver_approved_by,
			created_at, updated_at
		FROM repayments
		WHERE repayment_id = $1
	`

	var repayment models.Repayment
	err := r.db.QueryRowContext(ctx, query, repaymentID).Scan(
		&repayment.RepaymentID, &repayment.LoanID, &repayment.PaymentDate, &repayment.PaymentAmount,
		&repayment.PrincipalPaid, &repayment.InterestPaid, &repayment.FeesPaid, &repayment.PenaltyPaid,
		&repayment.PaymentMethod, &repayment.PaymentReference, &repayment.PaymentChannel,
		&repayment.DPDAtPayment, &repayment.IsBackdated, &repayment.IsReversed,
		&repayment.ReversalDate, &repayment.ReversalReason,
		&repayment.WaiverAmount, &repayment.WaiverType, &repayment.WaiverApprovedBy,
		&repayment.CreatedAt, &repayment.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &repayment, nil
}

// GetByLoanID retrieves all repayments for a loan
func (r *RepaymentRepository) GetByLoanID(ctx context.Context, loanID string) ([]*models.Repayment, error) {
	query := `
		SELECT
			repayment_id, loan_id, payment_date, payment_amount,
			principal_paid, interest_paid, fees_paid, penalty_paid,
			payment_method, payment_reference, payment_channel,
			dpd_at_payment, is_backdated, is_reversed,
			reversal_date, reversal_reason,
			waiver_amount, waiver_type, waiver_approved_by,
			created_at, updated_at
		FROM repayments
		WHERE loan_id = $1
		ORDER BY payment_date DESC
	`

	rows, err := r.db.QueryContext(ctx, query, loanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repayments []*models.Repayment
	for rows.Next() {
		var repayment models.Repayment
		err := rows.Scan(
			&repayment.RepaymentID, &repayment.LoanID, &repayment.PaymentDate, &repayment.PaymentAmount,
			&repayment.PrincipalPaid, &repayment.InterestPaid, &repayment.FeesPaid, &repayment.PenaltyPaid,
			&repayment.PaymentMethod, &repayment.PaymentReference, &repayment.PaymentChannel,
			&repayment.DPDAtPayment, &repayment.IsBackdated, &repayment.IsReversed,
			&repayment.ReversalDate, &repayment.ReversalReason,
			&repayment.WaiverAmount, &repayment.WaiverType, &repayment.WaiverApprovedBy,
			&repayment.CreatedAt, &repayment.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		repayments = append(repayments, &repayment)
	}

	return repayments, nil
}

// GetMaxRepaymentID returns the highest repayment_id (as integer) currently in the database
// This is used for incremental sync to determine which repayments are new
func (r *RepaymentRepository) GetMaxRepaymentID(ctx context.Context) (int64, error) {
	query := `
		SELECT COALESCE(MAX(CAST(repayment_id AS BIGINT)), 0)
		FROM repayments
		WHERE repayment_id ~ '^[0-9]+$'
	`

	var maxID int64
	err := r.db.QueryRowContext(ctx, query).Scan(&maxID)
	if err != nil {
		return 0, fmt.Errorf("failed to get max repayment ID: %w", err)
	}

	return maxID, nil
}
