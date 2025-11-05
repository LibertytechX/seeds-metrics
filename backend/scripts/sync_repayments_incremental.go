package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/seeds-metrics/analytics-backend/internal/config"
	"github.com/seeds-metrics/analytics-backend/internal/models"
	"github.com/seeds-metrics/analytics-backend/internal/repository"
	"github.com/seeds-metrics/analytics-backend/pkg/database"
	"github.com/shopspring/decimal"
)

func main() {
	log.Println("🚀 Starting incremental repayment sync...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to SeedsMetrics database (read-write)
	seedsDB, err := database.NewPostgresDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to SeedsMetrics database: %v", err)
	}
	defer seedsDB.Close()
	log.Println("✅ Connected to SeedsMetrics database")

	// Connect to Django database (read-only)
	djangoDB, err := database.NewPostgresDB(&cfg.DjangoDatabase)
	if err != nil {
		log.Fatalf("Failed to connect to Django database: %v", err)
	}
	defer djangoDB.Close()
	log.Println("✅ Connected to Django database")

	// Initialize repositories
	djangoRepo := repository.NewDjangoRepository(djangoDB.DB)
	repaymentRepo := repository.NewRepaymentRepository(seedsDB)

	ctx := context.Background()

	// Sync repayments incrementally
	log.Println("\n📊 Syncing Repayments (Incremental)...")
	if err := syncRepaymentIncremental(ctx, seedsDB.DB, djangoRepo, repaymentRepo); err != nil {
		log.Fatalf("Failed to sync repayments: %v", err)
	}

	log.Println("\n✅ Incremental repayment sync completed successfully!")
}

// syncRepaymentIncremental syncs only new repayments since last sync
func syncRepaymentIncremental(ctx context.Context, seedsDB *sql.DB, djangoRepo *repository.DjangoRepository, repaymentRepo *repository.RepaymentRepository) error {
	startTime := time.Now()

	// Get last sync timestamp
	var lastSyncTimestamp *time.Time
	err := seedsDB.QueryRowContext(ctx, `
		SELECT last_sync_timestamp
		FROM sync_tracking
		WHERE entity_type = 'repayments'
	`).Scan(&lastSyncTimestamp)

	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to get last sync timestamp: %w", err)
	}

	// If no last sync, sync all repayments
	if lastSyncTimestamp == nil {
		log.Println("⚠️  No previous sync found. Syncing ALL repayments...")
		return syncAllRepayments(ctx, seedsDB, djangoRepo, repaymentRepo)
	}

	log.Printf("📅 Last sync: %s", lastSyncTimestamp.Format("2006-01-02 15:04:05"))
	log.Printf("🔄 Syncing repayments created/updated after: %s", lastSyncTimestamp.Format("2006-01-02 15:04:05"))

	// Fetch new repayments from Django since last sync
	batchSize := 1000
	offset := 0
	totalSynced := 0
	errorCount := 0

	for {
		log.Printf("Processing batch: offset=%d, batch_size=%d", offset, batchSize)

		// Fetch repayments from Django
		repayments, err := djangoRepo.GetRepayments(ctx, batchSize, offset)
		if err != nil {
			return fmt.Errorf("failed to fetch repayments: %w", err)
		}

		if len(repayments) == 0 {
			break
		}

		// Process each repayment
		for _, repaymentData := range repayments {
			// Extract fields
			repaymentID, _ := repaymentData["repayment_id"].(string)
			loanID, _ := repaymentData["loan_id"].(string)
			paymentDate, _ := repaymentData["payment_date"].(string)
			paymentAmount, _ := repaymentData["payment_amount"].(float64)
			paymentMethod, _ := repaymentData["payment_method"].(string)
			updatedAt, _ := repaymentData["updated_at"].(time.Time)

			// Skip if essential fields are missing
			if repaymentID == "" || loanID == "" || paymentDate == "" || paymentAmount <= 0 {
				log.Printf("⚠️  Skipping repayment with missing essential fields: %v", repaymentData)
				errorCount++
				continue
			}

			// Check if this repayment was updated after last sync
			if updatedAt.Before(*lastSyncTimestamp) {
				// Skip old repayments
				continue
			}

			// Check if repayment already exists in SeedsMetrics
			existingRepayment, err := repaymentRepo.GetByID(ctx, repaymentID)
			if err != nil && err != sql.ErrNoRows {
				log.Printf("❌ Error checking repayment %s: %v", repaymentID, err)
				errorCount++
				continue
			}

			// If repayment exists and hasn't been updated, skip it
			if existingRepayment != nil && existingRepayment.UpdatedAt.After(updatedAt) {
				continue
			}

			// Create repayment input
			input := &models.RepaymentInput{
				RepaymentID:   repaymentID,
				LoanID:        loanID,
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
			if err := repaymentRepo.Create(ctx, input); err != nil {
				if err.Error() == "loan not found" {
					// Skip silently - this loan wasn't synced
					errorCount++
				} else {
					log.Printf("❌ Failed to sync repayment %s: %v", input.RepaymentID, err)
					errorCount++
				}
			} else {
				totalSynced++
				if totalSynced%500 == 0 {
					log.Printf("   Synced %d repayments...", totalSynced)
				}
			}
		}

		// Move to next batch
		offset += batchSize

		// If we got fewer than batchSize, we're done
		if len(repayments) < batchSize {
			break
		}

		// Small delay to avoid overwhelming the database
		time.Sleep(100 * time.Millisecond)
	}

	// Update sync tracking
	duration := time.Since(startTime).Milliseconds()
	updateErr := seedsDB.QueryRowContext(ctx, `
		UPDATE sync_tracking
		SET
			last_sync_timestamp = CURRENT_TIMESTAMP,
			sync_count = sync_count + 1,
			last_sync_records_count = $1,
			last_sync_errors_count = $2,
			last_sync_duration_ms = $3,
			updated_at = CURRENT_TIMESTAMP
		WHERE entity_type = 'repayments'
		RETURNING last_sync_timestamp
	`, totalSynced, errorCount, duration).Scan(&lastSyncTimestamp)

	if updateErr != nil {
		log.Printf("⚠️  Failed to update sync tracking: %v", updateErr)
	}

	log.Printf("✅ Incremental repayment sync complete: %d successful, %d errors, %dms", totalSynced, errorCount, duration)
	return nil
}

// syncAllRepayments syncs all repayments from Django
func syncAllRepayments(ctx context.Context, seedsDB *sql.DB, djangoRepo *repository.DjangoRepository, repaymentRepo *repository.RepaymentRepository) error {
	startTime := time.Now()
	batchSize := 1000
	offset := 0
	totalSynced := 0
	errorCount := 0

	for {
		log.Printf("Processing batch: offset=%d, batch_size=%d", offset, batchSize)

		// Fetch repayments from Django
		repayments, err := djangoRepo.GetRepayments(ctx, batchSize, offset)
		if err != nil {
			return fmt.Errorf("failed to fetch repayments: %w", err)
		}

		if len(repayments) == 0 {
			break
		}

		// Process each repayment
		for _, repaymentData := range repayments {
			repaymentID, _ := repaymentData["repayment_id"].(string)
			loanID, _ := repaymentData["loan_id"].(string)
			paymentDate, _ := repaymentData["payment_date"].(string)
			paymentAmount, _ := repaymentData["payment_amount"].(float64)
			paymentMethod, _ := repaymentData["payment_method"].(string)

			if repaymentID == "" || loanID == "" || paymentDate == "" || paymentAmount <= 0 {
				log.Printf("⚠️  Skipping repayment with missing essential fields: %v", repaymentData)
				errorCount++
				continue
			}

			input := &models.RepaymentInput{
				RepaymentID:   repaymentID,
				LoanID:        loanID,
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

			if err := repaymentRepo.Create(ctx, input); err != nil {
				if err.Error() != "loan not found" {
					log.Printf("❌ Failed to sync repayment %s: %v", input.RepaymentID, err)
				}
				errorCount++
			} else {
				totalSynced++
				if totalSynced%500 == 0 {
					log.Printf("   Synced %d repayments...", totalSynced)
				}
			}
		}

		offset += batchSize
		if len(repayments) < batchSize {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	// Update sync tracking
	duration := time.Since(startTime).Milliseconds()
	seedsDB.QueryRowContext(ctx, `
		UPDATE sync_tracking
		SET
			last_sync_timestamp = CURRENT_TIMESTAMP,
			sync_count = sync_count + 1,
			last_sync_records_count = $1,
			last_sync_errors_count = $2,
			last_sync_duration_ms = $3,
			updated_at = CURRENT_TIMESTAMP
		WHERE entity_type = 'repayments'
	`, totalSynced, errorCount, duration)

	log.Printf("✅ Full repayment sync complete: %d successful, %d errors, %dms", totalSynced, errorCount, duration)
	return nil
}
