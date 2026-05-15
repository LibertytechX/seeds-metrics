package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
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
	queuedForRetry := 0

	retrySynced, retryErrors, err := syncQueuedRepayments(ctx, seedsDB, djangoRepo, repaymentRepo)
	if err != nil {
		return fmt.Errorf("failed to process queued repayment retries: %w", err)
	}
	totalSynced += retrySynced
	errorCount += retryErrors

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
				if isLoanMissingError(err) {
					if queueErr := enqueueRepaymentRetry(ctx, seedsDB, repaymentID, loanID, updatedAt, err); queueErr != nil {
						log.Printf("❌ Failed to queue repayment %s for retry: %v", repaymentID, queueErr)
						errorCount++
					} else {
						queuedForRetry++
						log.Printf("⏳ Queued repayment %s for retry because loan %s is not yet synced", repaymentID, loanID)
					}
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

	log.Printf("✅ Incremental repayment sync complete: %d successful, %d queued, %d errors, %dms", totalSynced, queuedForRetry, errorCount, duration)
	return nil
}

func syncQueuedRepayments(ctx context.Context, seedsDB *sql.DB, djangoRepo *repository.DjangoRepository, repaymentRepo *repository.RepaymentRepository) (int, int, error) {
	rows, err := seedsDB.QueryContext(ctx, `
		SELECT repayment_id, loan_id
		FROM repayment_sync_retry_queue
		ORDER BY first_seen_at
		LIMIT 500
	`)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	queuedByLoan := map[string]map[string]bool{}
	for rows.Next() {
		var repaymentID, loanID string
		if err := rows.Scan(&repaymentID, &loanID); err != nil {
			return 0, 0, err
		}
		if queuedByLoan[loanID] == nil {
			queuedByLoan[loanID] = map[string]bool{}
		}
		queuedByLoan[loanID][repaymentID] = false
	}
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}

	synced := 0
	errors := 0
	for loanID, queuedIDs := range queuedByLoan {
		var loanExists bool
		if err := seedsDB.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM loans WHERE loan_id = $1)`, loanID).Scan(&loanExists); err != nil {
			return synced, errors, err
		}
		if !loanExists {
			markRetryAttempt(ctx, seedsDB, queuedIDs, "loan still missing")
			continue
		}

		repayments, err := djangoRepo.GetRepaymentsByLoanID(ctx, loanID)
		if err != nil {
			markRetryAttempt(ctx, seedsDB, queuedIDs, err.Error())
			errors += len(queuedIDs)
			continue
		}

		for _, repaymentData := range repayments {
			repaymentID, _ := repaymentData["repayment_id"].(string)
			if _, ok := queuedIDs[repaymentID]; !ok {
				continue
			}
			queuedIDs[repaymentID] = true
			if err := syncRepaymentData(ctx, repaymentRepo, repaymentData); err != nil {
				markSingleRetryAttempt(ctx, seedsDB, repaymentID, err.Error())
				errors++
				continue
			}
			if _, err := seedsDB.ExecContext(ctx, `DELETE FROM repayment_sync_retry_queue WHERE repayment_id = $1`, repaymentID); err != nil {
				return synced, errors, err
			}
			synced++
		}

		for repaymentID, found := range queuedIDs {
			if !found {
				markSingleRetryAttempt(ctx, seedsDB, repaymentID, "repayment no longer found in Django")
				errors++
			}
		}
	}

	return synced, errors, nil
}

func syncRepaymentData(ctx context.Context, repaymentRepo *repository.RepaymentRepository, repaymentData map[string]interface{}) error {
	repaymentID, _ := repaymentData["repayment_id"].(string)
	loanID, _ := repaymentData["loan_id"].(string)
	paymentDate, _ := repaymentData["payment_date"].(string)
	paymentAmount, _ := repaymentData["payment_amount"].(float64)
	paymentMethod, _ := repaymentData["payment_method"].(string)
	if repaymentID == "" || loanID == "" || paymentDate == "" || paymentAmount <= 0 {
		return fmt.Errorf("repayment has missing essential fields")
	}
	amount := decimal.NewFromFloat(paymentAmount)
	return repaymentRepo.Create(ctx, &models.RepaymentInput{
		RepaymentID: repaymentID, LoanID: loanID, PaymentDate: paymentDate,
		PaymentAmount: amount, PrincipalPaid: amount, InterestPaid: decimal.Zero,
		FeesPaid: decimal.Zero, PenaltyPaid: decimal.Zero, PaymentMethod: paymentMethod,
		DPDAtPayment: 0, IsBackdated: false, IsReversed: false, WaiverAmount: decimal.Zero,
	})
}

func enqueueRepaymentRetry(ctx context.Context, seedsDB *sql.DB, repaymentID, loanID string, djangoUpdatedAt time.Time, cause error) error {
	_, err := seedsDB.ExecContext(ctx, `
		INSERT INTO repayment_sync_retry_queue (repayment_id, loan_id, django_updated_at, last_error, updated_at)
		VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
		ON CONFLICT (repayment_id) DO UPDATE SET
			loan_id = EXCLUDED.loan_id,
			django_updated_at = GREATEST(repayment_sync_retry_queue.django_updated_at, EXCLUDED.django_updated_at),
			last_error = EXCLUDED.last_error,
			updated_at = CURRENT_TIMESTAMP
	`, repaymentID, loanID, djangoUpdatedAt, cause.Error())
	return err
}

func markRetryAttempt(ctx context.Context, seedsDB *sql.DB, queuedIDs map[string]bool, reason string) {
	for repaymentID := range queuedIDs {
		markSingleRetryAttempt(ctx, seedsDB, repaymentID, reason)
	}
}

func markSingleRetryAttempt(ctx context.Context, seedsDB *sql.DB, repaymentID, reason string) {
	if _, err := seedsDB.ExecContext(ctx, `
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
