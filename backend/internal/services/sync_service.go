package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/seeds-metrics/analytics-backend/internal/models"
	"github.com/seeds-metrics/analytics-backend/internal/repository"
	"github.com/seeds-metrics/analytics-backend/pkg/database"
	"github.com/shopspring/decimal"
)

// SyncService handles data synchronization operations
type SyncService struct {
	djangoRepo       *repository.DjangoRepository
	repaymentRepo    *repository.RepaymentRepository
	loanRepo         *repository.LoanRepository
	creditBureauRepo *repository.CreditBureauRepository
}

// NewSyncService creates a new sync service
func NewSyncService(djangoDB *sql.DB, seedsDB *database.DB) *SyncService {
	return &SyncService{
		djangoRepo:       repository.NewDjangoRepository(djangoDB),
		repaymentRepo:    repository.NewRepaymentRepository(seedsDB),
		loanRepo:         repository.NewLoanRepository(seedsDB),
		creditBureauRepo: repository.NewCreditBureauRepository(seedsDB),
	}
}

// SyncLoanRepaymentsResult contains the result of syncing repayments for a loan
type SyncLoanRepaymentsResult struct {
	LoanID      string       `json:"loan_id"`
	TotalSynced int          `json:"total_synced"`
	TotalErrors int          `json:"total_errors"`
	UpdatedLoan *models.Loan `json:"updated_loan,omitempty"`
	Message     string       `json:"message"`
}

// SyncLoanRepayments syncs repayments for a specific loan from Django to SeedsMetrics
func (s *SyncService) SyncLoanRepayments(ctx context.Context, loanID string) (*SyncLoanRepaymentsResult, error) {
	log.Printf("🔄 Starting repayment sync for loan %s", loanID)

	// Verify loan exists in SeedsMetrics
	loan, err := s.loanRepo.GetByID(ctx, loanID)
	if err != nil {
		return nil, fmt.Errorf("failed to get loan: %w", err)
	}
	if loan == nil {
		return nil, fmt.Errorf("loan %s not found", loanID)
	}

	// Fetch repayments from Django for this specific loan
	repayments, err := s.djangoRepo.GetRepaymentsByLoanID(ctx, loanID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repayments from Django: %w", err)
	}

	log.Printf("📦 Found %d repayments in Django for loan %s", len(repayments), loanID)

	totalSynced := 0
	errorCount := 0

	// Process each repayment
	for _, repaymentData := range repayments {
		// Convert map to RepaymentInput with nil-safe type assertions
		repaymentID, _ := repaymentData["repayment_id"].(string)
		loanIDStr, _ := repaymentData["loan_id"].(string)
		paymentDate, _ := repaymentData["payment_date"].(string)
		paymentAmount, _ := repaymentData["payment_amount"].(float64)
		paymentMethod, _ := repaymentData["payment_method"].(string)

		// Skip if essential fields are missing
		if repaymentID == "" || loanIDStr == "" || paymentDate == "" || paymentAmount <= 0 {
			log.Printf("⚠️  Skipping repayment with missing essential fields: %v", repaymentData)
			errorCount++
			continue
		}

		// For Django repayments, we don't have breakdown of principal/interest/fees
		// So we'll put the full amount as principal_paid and let the triggers calculate
		input := &models.RepaymentInput{
			RepaymentID:   repaymentID,
			LoanID:        loanIDStr,
			PaymentDate:   paymentDate,
			PaymentAmount: decimal.NewFromFloat(paymentAmount),
			PrincipalPaid: decimal.NewFromFloat(paymentAmount), // Full amount as principal
			InterestPaid:  decimal.Zero,
			FeesPaid:      decimal.Zero,
			PenaltyPaid:   decimal.Zero,
			PaymentMethod: paymentMethod,
			DPDAtPayment:  0,
			IsBackdated:   false,
			IsReversed:    false,
			WaiverAmount:  decimal.Zero,
		}

		// Create/update repayment
		if err := s.repaymentRepo.Create(ctx, input); err != nil {
			log.Printf("❌ Failed to sync repayment %s: %v", input.RepaymentID, err)
			errorCount++
		} else {
			totalSynced++
		}
	}

	log.Printf("✅ Repayment sync complete for loan %s: %d successful, %d errors", loanID, totalSynced, errorCount)

	// Fetch updated loan data
	updatedLoan, err := s.loanRepo.GetByID(ctx, loanID)
	if err != nil {
		log.Printf("⚠️  Failed to fetch updated loan data: %v", err)
	}

	result := &SyncLoanRepaymentsResult{
		LoanID:      loanID,
		TotalSynced: totalSynced,
		TotalErrors: errorCount,
		UpdatedLoan: updatedLoan,
		Message:     fmt.Sprintf("Synced %d repayments (%d errors)", totalSynced, errorCount),
	}

	return result, nil
}

// SyncNewRepaymentsResult contains the result of syncing new repayments
type SyncNewRepaymentsResult struct {
	TotalSynced   int    `json:"total_synced"`
	TotalErrors   int    `json:"total_errors"`
	LastIDSynced  int64  `json:"last_id_synced"`
	PreviousMaxID int64  `json:"previous_max_id"`
	Message       string `json:"message"`
}

// SyncNewRepayments syncs only new repayments from Django that have ID > max existing ID
func (s *SyncService) SyncNewRepayments(ctx context.Context) (*SyncNewRepaymentsResult, error) {
	log.Printf("🔄 Starting incremental repayment sync...")

	// Get the max repayment ID currently in seedsmetrics
	maxID, err := s.repaymentRepo.GetMaxRepaymentID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get max repayment ID: %w", err)
	}
	log.Printf("📊 Current max repayment ID in seedsmetrics: %d", maxID)

	// Fetch new repayments from Django in batches
	batchSize := 1000
	totalSynced := 0
	errorCount := 0
	lastIDSynced := maxID

	for {
		repayments, err := s.djangoRepo.GetRepaymentsAfterID(ctx, lastIDSynced, batchSize)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch new repayments from Django: %w", err)
		}

		if len(repayments) == 0 {
			break
		}

		log.Printf("📦 Processing batch of %d new repayments (after ID %d)", len(repayments), lastIDSynced)

		for _, repaymentData := range repayments {
			repaymentID, _ := repaymentData["repayment_id"].(string)
			repaymentIDInt, _ := repaymentData["repayment_id_int"].(int64)
			loanIDStr, _ := repaymentData["loan_id"].(string)
			paymentDate, _ := repaymentData["payment_date"].(string)
			paymentAmount, _ := repaymentData["payment_amount"].(float64)
			paymentMethod, _ := repaymentData["payment_method"].(string)

			// Skip if essential fields are missing
			if repaymentID == "" || loanIDStr == "" || paymentDate == "" || paymentAmount <= 0 {
				errorCount++
				continue
			}

			input := &models.RepaymentInput{
				RepaymentID:   repaymentID,
				LoanID:        loanIDStr,
				PaymentDate:   paymentDate,
				PaymentAmount: decimal.NewFromFloat(paymentAmount),
				PrincipalPaid: decimal.NewFromFloat(paymentAmount),
				InterestPaid:  decimal.Zero,
				FeesPaid:      decimal.Zero,
				PenaltyPaid:   decimal.Zero,
				PaymentMethod: paymentMethod,
				DPDAtPayment:  0,
				IsBackdated:   false,
				IsReversed:    false,
				WaiverAmount:  decimal.Zero,
			}

			if err := s.repaymentRepo.Create(ctx, input); err != nil {
				if err.Error() != "loan not found" {
					log.Printf("❌ Failed to sync repayment %s: %v", input.RepaymentID, err)
				}
				errorCount++
			} else {
				totalSynced++
			}

			// Track the highest ID we've processed
			if repaymentIDInt > lastIDSynced {
				lastIDSynced = repaymentIDInt
			}
		}

		// If we got fewer than batchSize, we're done
		if len(repayments) < batchSize {
			break
		}
	}

	log.Printf("✅ Incremental sync complete: %d synced, %d errors (ID range: %d -> %d)", totalSynced, errorCount, maxID, lastIDSynced)

	result := &SyncNewRepaymentsResult{
		TotalSynced:   totalSynced,
		TotalErrors:   errorCount,
		LastIDSynced:  lastIDSynced,
		PreviousMaxID: maxID,
		Message:       fmt.Sprintf("Synced %d new repayments (%d errors). ID range: %d -> %d", totalSynced, errorCount, maxID, lastIDSynced),
	}

	return result, nil
}

// SyncLoanCreditBureauKYCResult contains the result of syncing loan credit bureau KYC data
type SyncLoanCreditBureauKYCResult struct {
	TotalFetched      int    `json:"total_fetched"`
	TotalUpserted     int    `json:"total_upserted"`
	TotalErrors       int    `json:"total_errors"`
	LegacyConverted   int    `json:"legacy_converted"`
	LegacyConvertFail int    `json:"legacy_convert_fail"`
	Message           string `json:"message"`
}

// SyncLoanCreditBureauKYC fetches loan + KYC + credit bureau data from Django in batches and upserts into metrics DB.
// Uses batched fetching (LIMIT/OFFSET) to avoid OOM on low-memory servers.
func (s *SyncService) SyncLoanCreditBureauKYC(ctx context.Context) (*SyncLoanCreditBureauKYCResult, error) {
	log.Printf("🔄 Starting loan credit bureau KYC sync...")

	const fetchBatchSize = 500
	totalFetched := 0
	totalUpserted := 0
	totalErrors := 0
	legacyConverted := 0
	legacyConvertFail := 0

	for offset := 0; ; offset += fetchBatchSize {
		// Fetch one batch from Django
		rows, err := s.djangoRepo.GetLoanCreditBureauKYCForSyncBatch(ctx, fetchBatchSize, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch loan credit bureau KYC batch (offset=%d): %w", offset, err)
		}
		if len(rows) == 0 {
			break // No more rows
		}

		totalFetched += len(rows)
		log.Printf("📦 Fetched batch offset=%d: %d rows", offset, len(rows))

		// Convert rows to model records
		records := make([]*models.LoanCreditBureauKYC, 0, len(rows))
		for _, row := range rows {
			rec := s.convertKYCRow(row, &legacyConverted, &legacyConvertFail)
			records = append(records, rec)
		}
		// Free Django rows immediately
		rows = nil

		// Upsert this batch
		upserted, err := s.creditBureauRepo.UpsertLoanCreditBureauKYC(ctx, records)
		if err != nil {
			log.Printf("❌ Batch upsert error at offset %d: %v", offset, err)
		}
		totalUpserted += upserted
		totalErrors += len(records) - upserted
		log.Printf("📊 Upserted batch offset=%d: %d/%d records", offset, upserted, len(records))

		// Free records before next iteration
		records = nil
	}

	log.Printf("✅ Loan credit bureau KYC sync complete: %d fetched, %d upserted, %d errors, %d legacy converted, %d legacy failed",
		totalFetched, totalUpserted, totalErrors, legacyConverted, legacyConvertFail)

	return &SyncLoanCreditBureauKYCResult{
		TotalFetched:      totalFetched,
		TotalUpserted:     totalUpserted,
		TotalErrors:       totalErrors,
		LegacyConverted:   legacyConverted,
		LegacyConvertFail: legacyConvertFail,
		Message:           fmt.Sprintf("Synced %d loan credit bureau KYC records (%d errors, %d legacy converted, %d legacy failed)", totalUpserted, totalErrors, legacyConverted, legacyConvertFail),
	}, nil
}

// convertKYCRow converts a raw Django row into a LoanCreditBureauKYC model record
func (s *SyncService) convertKYCRow(row *repository.LoanCreditBureauKYCRow, legacyConverted, legacyConvertFail *int) *models.LoanCreditBureauKYC {
	rec := &models.LoanCreditBureauKYC{
		DjangoLoanID: row.DjangoLoanID,
		LoanRef:      row.LoanRef,
		CBDataSource: row.CBDataSource,
	}

	// Loan fields
	if row.LoanAmount.Valid {
		v := row.LoanAmount.Float64
		rec.LoanAmount = &v
	}
	if row.Tenor.Valid {
		rec.Tenor = &row.Tenor.String
	}
	if row.TenorInDays.Valid {
		v := int(row.TenorInDays.Int32)
		rec.TenorInDays = &v
	}
	if row.BorrowerFullName.Valid {
		rec.BorrowerFullName = &row.BorrowerFullName.String
	}
	if row.BorrowerPhone.Valid {
		rec.BorrowerPhone = &row.BorrowerPhone.String
	}
	if row.LoanStatus.Valid {
		rec.LoanStatus = &row.LoanStatus.String
	}
	if row.DateDisbursed.Valid {
		rec.DateDisbursed = &row.DateDisbursed.Time
	}

	// KYC fields
	if row.VerificationNumber.Valid {
		rec.VerificationNumber = &row.VerificationNumber.String
	}
	if row.VerificationType.Valid {
		rec.VerificationType = &row.VerificationType.String
	}
	if row.NIN.Valid {
		rec.NIN = &row.NIN.String
	}
	if row.DateOfBirth.Valid {
		rec.DateOfBirth = &row.DateOfBirth.String
	}
	if row.IsVerified.Valid {
		rec.IsVerified = &row.IsVerified.Bool
	}
	if row.Address.Valid {
		rec.Address = &row.Address.String
	}
	if row.FaceMatch.Valid {
		rec.FaceMatch = &row.FaceMatch.Bool
	}
	if row.IDCardImage.Valid {
		rec.IDCardImage = &row.IDCardImage.String
	}
	if row.SelfieImage.Valid {
		rec.SelfieImage = &row.SelfieImage.String
	}

	// Credit bureau JSON fields
	if len(row.CBResult) > 0 {
		rec.CBResult = json.RawMessage(row.CBResult)
	}
	if len(row.CBDecision) > 0 {
		rec.CBDecision = json.RawMessage(row.CBDecision)
	}
	if row.CBDecisionStatus.Valid {
		rec.CBDecisionStatus = &row.CBDecisionStatus.String
	}
	if len(row.CBCredibility) > 0 {
		rec.CBCredibility = json.RawMessage(row.CBCredibility)
	}
	if row.CBStatus.Valid {
		rec.CBStatus = &row.CBStatus.Bool
	}
	if row.CBReason.Valid {
		rec.CBReason = &row.CBReason.String
	}
	if len(row.CBBadLoansInstitutions) > 0 {
		rec.CBBadLoansInstitutions = json.RawMessage(row.CBBadLoansInstitutions)
	}
	if row.CBBadLoansInstitutionsCount.Valid {
		v := int(row.CBBadLoansInstitutionsCount.Int32)
		rec.CBBadLoansInstitutionsCount = &v
	}
	if row.CBCountOfOpenLoans.Valid {
		v := int(row.CBCountOfOpenLoans.Int32)
		rec.CBCountOfOpenLoans = &v
	}
	if row.CBTotalOutstanding.Valid {
		v := row.CBTotalOutstanding.Float64
		rec.CBTotalOutstanding = &v
	}
	if row.CBDebtThreshold.Valid {
		v := row.CBDebtThreshold.Float64
		rec.CBDebtThreshold = &v
	}
	if row.CBHighOutstandingDebt.Valid {
		rec.CBHighOutstandingDebt = &row.CBHighOutstandingDebt.Bool
	}
	if len(row.CBOpenLoanInstitutions) > 0 {
		rec.CBOpenLoanInstitutions = json.RawMessage(row.CBOpenLoanInstitutions)
	}
	if row.CBMaxDebtInstitutionCount.Valid {
		v := int(row.CBMaxDebtInstitutionCount.Int32)
		rec.CBMaxDebtInstitutionCount = &v
	}

	// Legacy metadata: convert Python repr to JSON
	if row.CBLegacyResponse.Valid && row.CBLegacyResponse.String != "" {
		jsonStr, err := pythonReprToJSON(row.CBLegacyResponse.String)
		if err != nil {
			log.Printf("⚠️  Failed to convert legacy response for loan %d: %v", row.DjangoLoanID, err)
			*legacyConvertFail++
		} else {
			rec.CBLegacyResponse = json.RawMessage(jsonStr)
			*legacyConverted++
		}
	}
	if row.CBNoOfDefaultedLoans.Valid {
		v := int(row.CBNoOfDefaultedLoans.Int32)
		rec.CBNoOfDefaultedLoans = &v
	}
	if row.CBMonthlyRepaymentAmount.Valid {
		v := row.CBMonthlyRepaymentAmount.Float64
		rec.CBMonthlyRepaymentAmount = &v
	}
	if row.CBCreatedAt.Valid {
		rec.CBCreatedAt = &row.CBCreatedAt.Time
	}

	return rec
}
