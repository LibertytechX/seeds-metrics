package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/seeds-metrics/analytics-backend/internal/models"
	"github.com/seeds-metrics/analytics-backend/internal/repository"
	"github.com/seeds-metrics/analytics-backend/pkg/database"
	"github.com/shopspring/decimal"
)

// SyncService handles data synchronization operations
type SyncService struct {
	djangoRepo    *repository.DjangoRepository
	repaymentRepo *repository.RepaymentRepository
	loanRepo      *repository.LoanRepository
}

// NewSyncService creates a new sync service
func NewSyncService(djangoDB *sql.DB, seedsDB *database.DB) *SyncService {
	return &SyncService{
		djangoRepo:    repository.NewDjangoRepository(djangoDB),
		repaymentRepo: repository.NewRepaymentRepository(seedsDB),
		loanRepo:      repository.NewLoanRepository(seedsDB),
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
