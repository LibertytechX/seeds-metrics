package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

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
	seedsDB          *database.DB
}

// NewSyncService creates a new sync service
func NewSyncService(djangoDB *sql.DB, seedsDB *database.DB) *SyncService {
	return &SyncService{
		djangoRepo:       repository.NewDjangoRepository(djangoDB),
		repaymentRepo:    repository.NewRepaymentRepository(seedsDB),
		loanRepo:         repository.NewLoanRepository(seedsDB),
		creditBureauRepo: repository.NewCreditBureauRepository(seedsDB),
		seedsDB:          seedsDB,
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
	retrySynced, retryErrors := s.syncQueuedRepayments(ctx)
	totalSynced += retrySynced
	errorCount += retryErrors
	if retrySynced > 0 {
		maxID, err = s.repaymentRepo.GetMaxRepaymentID(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh max repayment ID after retry queue sync: %w", err)
		}
		log.Printf("📊 Max repayment ID after retry queue sync: %d", maxID)
	}
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
				if isLoanMissingError(err) {
					if queueErr := s.enqueueRepaymentRetry(ctx, repaymentID, loanIDStr, err); queueErr != nil {
						log.Printf("❌ Failed to queue repayment %s for retry: %v", input.RepaymentID, queueErr)
						errorCount++
					} else {
						log.Printf("⏳ Queued repayment %s for retry because loan %s is not yet synced", input.RepaymentID, loanIDStr)
					}
				} else {
					log.Printf("❌ Failed to sync repayment %s: %v", input.RepaymentID, err)
					errorCount++
				}
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

func (s *SyncService) syncQueuedRepayments(ctx context.Context) (int, int) {
	rows, err := s.seedsDB.QueryContext(ctx, `
		SELECT repayment_id, loan_id
		FROM repayment_sync_retry_queue
		ORDER BY first_seen_at
		LIMIT 500
	`)
	if err != nil {
		log.Printf("⚠️  Failed to read repayment retry queue: %v", err)
		return 0, 1
	}
	defer rows.Close()

	queuedByLoan := map[string]map[string]bool{}
	for rows.Next() {
		var repaymentID, loanID string
		if err := rows.Scan(&repaymentID, &loanID); err != nil {
			log.Printf("⚠️  Failed to scan repayment retry queue: %v", err)
			return 0, 1
		}
		if queuedByLoan[loanID] == nil {
			queuedByLoan[loanID] = map[string]bool{}
		}
		queuedByLoan[loanID][repaymentID] = false
	}
	if err := rows.Err(); err != nil {
		log.Printf("⚠️  Failed to iterate repayment retry queue: %v", err)
		return 0, 1
	}

	synced := 0
	errors := 0
	for loanID, queuedIDs := range queuedByLoan {
		loan, err := s.loanRepo.GetByID(ctx, loanID)
		if err != nil {
			s.markRetryAttempts(ctx, queuedIDs, err.Error())
			errors += len(queuedIDs)
			continue
		}
		if loan == nil {
			s.markRetryAttempts(ctx, queuedIDs, "loan still missing")
			continue
		}

		repayments, err := s.djangoRepo.GetRepaymentsByLoanID(ctx, loanID)
		if err != nil {
			s.markRetryAttempts(ctx, queuedIDs, err.Error())
			errors += len(queuedIDs)
			continue
		}

		for _, repaymentData := range repayments {
			repaymentID, _ := repaymentData["repayment_id"].(string)
			if _, ok := queuedIDs[repaymentID]; !ok {
				continue
			}
			queuedIDs[repaymentID] = true
			if err := s.syncRepaymentData(ctx, repaymentData); err != nil {
				s.markSingleRetryAttempt(ctx, repaymentID, err.Error())
				errors++
				continue
			}
			if _, err := s.seedsDB.ExecContext(ctx, `DELETE FROM repayment_sync_retry_queue WHERE repayment_id = $1`, repaymentID); err != nil {
				log.Printf("⚠️  Failed to delete repayment %s from retry queue: %v", repaymentID, err)
				errors++
				continue
			}
			synced++
		}

		for repaymentID, found := range queuedIDs {
			if !found {
				s.markSingleRetryAttempt(ctx, repaymentID, "repayment no longer found in Django")
				errors++
			}
		}
	}
	return synced, errors
}

func (s *SyncService) syncRepaymentData(ctx context.Context, repaymentData map[string]interface{}) error {
	repaymentID, _ := repaymentData["repayment_id"].(string)
	loanID, _ := repaymentData["loan_id"].(string)
	paymentDate, _ := repaymentData["payment_date"].(string)
	paymentAmount, _ := repaymentData["payment_amount"].(float64)
	paymentMethod, _ := repaymentData["payment_method"].(string)
	if repaymentID == "" || loanID == "" || paymentDate == "" || paymentAmount <= 0 {
		return fmt.Errorf("repayment has missing essential fields")
	}
	amount := decimal.NewFromFloat(paymentAmount)
	return s.repaymentRepo.Create(ctx, &models.RepaymentInput{
		RepaymentID: repaymentID, LoanID: loanID, PaymentDate: paymentDate,
		PaymentAmount: amount, PrincipalPaid: amount, InterestPaid: decimal.Zero,
		FeesPaid: decimal.Zero, PenaltyPaid: decimal.Zero, PaymentMethod: paymentMethod,
		DPDAtPayment: 0, IsBackdated: false, IsReversed: false, WaiverAmount: decimal.Zero,
	})
}

func (s *SyncService) enqueueRepaymentRetry(ctx context.Context, repaymentID, loanID string, cause error) error {
	_, err := s.seedsDB.ExecContext(ctx, `
		INSERT INTO repayment_sync_retry_queue (repayment_id, loan_id, last_error, updated_at)
		VALUES ($1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (repayment_id) DO UPDATE SET
			loan_id = EXCLUDED.loan_id,
			last_error = EXCLUDED.last_error,
			updated_at = CURRENT_TIMESTAMP
	`, repaymentID, loanID, cause.Error())
	return err
}

func (s *SyncService) markRetryAttempts(ctx context.Context, queuedIDs map[string]bool, reason string) {
	for repaymentID := range queuedIDs {
		s.markSingleRetryAttempt(ctx, repaymentID, reason)
	}
}

func (s *SyncService) markSingleRetryAttempt(ctx context.Context, repaymentID, reason string) {
	if _, err := s.seedsDB.ExecContext(ctx, `
		UPDATE repayment_sync_retry_queue
		SET attempts = attempts + 1, last_attempt_at = CURRENT_TIMESTAMP, last_error = $2, updated_at = CURRENT_TIMESTAMP
		WHERE repayment_id = $1
	`, repaymentID, reason); err != nil {
		log.Printf("⚠️  Failed to update retry queue for repayment %s: %v", repaymentID, err)
	}
}

func isLoanMissingError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "loan not found") ||
		(strings.Contains(msg, "foreign key") && strings.Contains(msg, "loan")) ||
		strings.Contains(msg, "fk_loan")
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
	if row.LoanType.Valid {
		rec.LoanType = &row.LoanType.String
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

	// Guarantor fields
	if row.GuarantorFullName.Valid {
		rec.GuarantorFullName = &row.GuarantorFullName.String
	}
	if row.GuarantorPhone.Valid {
		rec.GuarantorPhone = &row.GuarantorPhone.String
	}
	if row.GuarantorEmail.Valid {
		rec.GuarantorEmail = &row.GuarantorEmail.String
	}
	if row.GuarantorAddress.Valid {
		rec.GuarantorAddress = &row.GuarantorAddress.String
	}
	if row.GuarantorRelationship.Valid {
		rec.GuarantorRelationship = &row.GuarantorRelationship.String
	}
	if row.GuarantorIDCardImage.Valid {
		rec.GuarantorIDCardImage = &row.GuarantorIDCardImage.String
	}
	if row.GuarantorSelfieImage.Valid {
		rec.GuarantorSelfieImage = &row.GuarantorSelfieImage.String
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
