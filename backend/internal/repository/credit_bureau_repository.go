package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/seeds-metrics/analytics-backend/internal/models"
	"github.com/seeds-metrics/analytics-backend/pkg/database"
)

// CreditBureauRepository handles credit bureau data operations on the local metrics DB
type CreditBureauRepository struct {
	db *sql.DB
}

// NewCreditBureauRepository creates a new credit bureau repository
func NewCreditBureauRepository(db *database.DB) *CreditBureauRepository {
	return &CreditBureauRepository{db: db.DB}
}

// UpsertLoanCreditBureauKYC upserts a batch of loan credit bureau KYC records
func (r *CreditBureauRepository) UpsertLoanCreditBureauKYC(ctx context.Context, records []*models.LoanCreditBureauKYC) (int, error) {
	if len(records) == 0 {
		return 0, nil
	}

	query := `
		INSERT INTO loan_credit_bureau_kyc (
			django_loan_id, loan_ref, loan_amount, tenor, tenor_in_days,
			borrower_full_name, borrower_phone_number, loan_status, date_disbursed,
			verification_number, verification_type, nin, date_of_birth, is_verified, address,
			face_match, id_card_image, selfie_image,
			cb_result, cb_status, cb_reason, cb_decision, cb_decision_status, cb_credibility,
			cb_bad_loans_institutions, cb_bad_loans_institutions_count,
			cb_count_of_open_loans, cb_total_outstanding, cb_debt_threshold,
			cb_high_outstanding_debt, cb_open_loan_institutions, cb_max_debt_institution_count,
			cb_legacy_response, cb_no_of_defaulted_loans, cb_monthly_repayment_amount,
			cb_data_source, cb_created_at, synced_at, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,
			$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,NOW(),NOW()
		)
		ON CONFLICT (django_loan_id) DO UPDATE SET
			loan_ref = EXCLUDED.loan_ref,
			loan_amount = EXCLUDED.loan_amount,
			tenor = EXCLUDED.tenor,
			tenor_in_days = EXCLUDED.tenor_in_days,
			borrower_full_name = EXCLUDED.borrower_full_name,
			borrower_phone_number = EXCLUDED.borrower_phone_number,
			loan_status = EXCLUDED.loan_status,
			date_disbursed = EXCLUDED.date_disbursed,
			verification_number = EXCLUDED.verification_number,
			verification_type = EXCLUDED.verification_type,
			nin = EXCLUDED.nin,
			date_of_birth = EXCLUDED.date_of_birth,
			is_verified = EXCLUDED.is_verified,
			address = EXCLUDED.address,
			face_match = EXCLUDED.face_match,
			id_card_image = EXCLUDED.id_card_image,
			selfie_image = EXCLUDED.selfie_image,
			cb_result = EXCLUDED.cb_result,
			cb_status = EXCLUDED.cb_status,
			cb_reason = EXCLUDED.cb_reason,
			cb_decision = EXCLUDED.cb_decision,
			cb_decision_status = EXCLUDED.cb_decision_status,
			cb_credibility = EXCLUDED.cb_credibility,
			cb_bad_loans_institutions = EXCLUDED.cb_bad_loans_institutions,
			cb_bad_loans_institutions_count = EXCLUDED.cb_bad_loans_institutions_count,
			cb_count_of_open_loans = EXCLUDED.cb_count_of_open_loans,
			cb_total_outstanding = EXCLUDED.cb_total_outstanding,
			cb_debt_threshold = EXCLUDED.cb_debt_threshold,
			cb_high_outstanding_debt = EXCLUDED.cb_high_outstanding_debt,
			cb_open_loan_institutions = EXCLUDED.cb_open_loan_institutions,
			cb_max_debt_institution_count = EXCLUDED.cb_max_debt_institution_count,
			cb_legacy_response = EXCLUDED.cb_legacy_response,
			cb_no_of_defaulted_loans = EXCLUDED.cb_no_of_defaulted_loans,
			cb_monthly_repayment_amount = EXCLUDED.cb_monthly_repayment_amount,
			cb_data_source = EXCLUDED.cb_data_source,
			cb_created_at = EXCLUDED.cb_created_at,
			synced_at = NOW(),
			updated_at = NOW()
	`

	upserted := 0
	for _, rec := range records {
		_, err := r.db.ExecContext(ctx, query,
			rec.DjangoLoanID, rec.LoanRef, rec.LoanAmount, rec.Tenor, rec.TenorInDays,
			rec.BorrowerFullName, rec.BorrowerPhone, rec.LoanStatus, rec.DateDisbursed,
			rec.VerificationNumber, rec.VerificationType, rec.NIN, rec.DateOfBirth, rec.IsVerified, rec.Address,
			rec.FaceMatch, rec.IDCardImage, rec.SelfieImage,
			nullableJSON(rec.CBResult), rec.CBStatus, rec.CBReason,
			nullableJSON(rec.CBDecision), rec.CBDecisionStatus, nullableJSON(rec.CBCredibility),
			nullableJSON(rec.CBBadLoansInstitutions), rec.CBBadLoansInstitutionsCount,
			rec.CBCountOfOpenLoans, rec.CBTotalOutstanding, rec.CBDebtThreshold,
			rec.CBHighOutstandingDebt, nullableJSON(rec.CBOpenLoanInstitutions), rec.CBMaxDebtInstitutionCount,
			nullableJSON(rec.CBLegacyResponse), rec.CBNoOfDefaultedLoans, rec.CBMonthlyRepaymentAmount,
			rec.CBDataSource, rec.CBCreatedAt,
		)
		if err != nil {
			log.Printf("Failed to upsert loan_credit_bureau_kyc django_loan_id=%d: %v", rec.DjangoLoanID, err)
			continue
		}
		upserted++
	}

	return upserted, nil
}

// nullableJSON returns nil for empty/null JSON, otherwise the raw bytes
func nullableJSON(data json.RawMessage) interface{} {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	return []byte(data)
}

// GetLoanCreditBureauKYCCount returns the total count of records, optionally filtered by source
func (r *CreditBureauRepository) GetLoanCreditBureauKYCCount(ctx context.Context, source string) (int, error) {
	query := `SELECT COUNT(*) FROM loan_credit_bureau_kyc`
	args := []interface{}{}

	if source != "" {
		query += ` WHERE cb_data_source = $1`
		args = append(args, source)
	}

	var count int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count loan_credit_bureau_kyc: %w", err)
	}
	return count, nil
}

// GetLoanCreditBureauKYCPaginated returns paginated records ordered by date_disbursed DESC
func (r *CreditBureauRepository) GetLoanCreditBureauKYCPaginated(ctx context.Context, page, limit int, source string, includeImages bool) ([]*models.LoanCreditBureauKYC, error) {
	// Build column list - exclude images by default
	cols := []string{
		"id", "django_loan_id", "loan_ref", "loan_amount", "tenor", "tenor_in_days",
		"borrower_full_name", "borrower_phone_number", "loan_status", "date_disbursed",
		"verification_number", "verification_type", "nin", "date_of_birth", "is_verified", "address",
		"face_match",
	}
	if includeImages {
		cols = append(cols, "id_card_image", "selfie_image")
	}
	cols = append(cols,
		"cb_result", "cb_status", "cb_reason", "cb_decision", "cb_decision_status", "cb_credibility",
		"cb_bad_loans_institutions", "cb_bad_loans_institutions_count",
		"cb_count_of_open_loans", "cb_total_outstanding", "cb_debt_threshold",
		"cb_high_outstanding_debt", "cb_open_loan_institutions", "cb_max_debt_institution_count",
		"cb_legacy_response", "cb_no_of_defaulted_loans", "cb_monthly_repayment_amount",
		"cb_data_source", "cb_created_at", "synced_at", "created_at", "updated_at",
	)

	query := fmt.Sprintf("SELECT %s FROM loan_credit_bureau_kyc", strings.Join(cols, ", "))
	args := []interface{}{}
	argIdx := 1

	if source != "" {
		query += fmt.Sprintf(" WHERE cb_data_source = $%d", argIdx)
		args = append(args, source)
		argIdx++
	}

	query += " ORDER BY date_disbursed DESC NULLS LAST"
	offset := (page - 1) * limit
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query loan_credit_bureau_kyc: %w", err)
	}
	defer rows.Close()

	var results []*models.LoanCreditBureauKYC
	for rows.Next() {
		rec := &models.LoanCreditBureauKYC{}
		dest := []interface{}{
			&rec.ID, &rec.DjangoLoanID, &rec.LoanRef, &rec.LoanAmount, &rec.Tenor, &rec.TenorInDays,
			&rec.BorrowerFullName, &rec.BorrowerPhone, &rec.LoanStatus, &rec.DateDisbursed,
			&rec.VerificationNumber, &rec.VerificationType, &rec.NIN, &rec.DateOfBirth, &rec.IsVerified, &rec.Address,
			&rec.FaceMatch,
		}
		if includeImages {
			dest = append(dest, &rec.IDCardImage, &rec.SelfieImage)
		}
		dest = append(dest,
			&rec.CBResult, &rec.CBStatus, &rec.CBReason, &rec.CBDecision, &rec.CBDecisionStatus, &rec.CBCredibility,
			&rec.CBBadLoansInstitutions, &rec.CBBadLoansInstitutionsCount,
			&rec.CBCountOfOpenLoans, &rec.CBTotalOutstanding, &rec.CBDebtThreshold,
			&rec.CBHighOutstandingDebt, &rec.CBOpenLoanInstitutions, &rec.CBMaxDebtInstitutionCount,
			&rec.CBLegacyResponse, &rec.CBNoOfDefaultedLoans, &rec.CBMonthlyRepaymentAmount,
			&rec.CBDataSource, &rec.CBCreatedAt, &rec.SyncedAt, &rec.CreatedAt, &rec.UpdatedAt,
		)

		if err := rows.Scan(dest...); err != nil {
			return nil, fmt.Errorf("failed to scan loan_credit_bureau_kyc row: %w", err)
		}
		results = append(results, rec)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating loan_credit_bureau_kyc rows: %w", err)
	}

	return results, nil
}
