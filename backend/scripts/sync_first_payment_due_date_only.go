package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/seeds-metrics/analytics-backend/internal/config"
	"github.com/seeds-metrics/analytics-backend/pkg/database"
)

func main() {
	log.Println("🚀 Starting first_payment_due_date sync from Django...")

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

	ctx := context.Background()

	// Sync first_payment_due_date only
	if err := syncFirstPaymentDueDate(ctx, djangoDB.DB, seedsDB.DB); err != nil {
		log.Fatalf("Failed to sync first_payment_due_date: %v", err)
	}

	log.Println("\n✅ first_payment_due_date sync completed successfully!")
}

func syncFirstPaymentDueDate(ctx context.Context, djangoDB *sql.DB, seedsDB *sql.DB) error {
	const batchSize = 500
	offset := 0
	totalUpdated := 0
	errorCount := 0

	for {
		// Fetch loans from Django with their start_date
		query := `
			SELECT 
				l.id::VARCHAR(50) as loan_id,
				l.start_date as first_payment_due_date
			FROM loans_ajoloan l
			WHERE l.is_disbursed = TRUE
			ORDER BY l.id
			LIMIT $1 OFFSET $2
		`

		rows, err := djangoDB.QueryContext(ctx, query, batchSize, offset)
		if err != nil {
			return fmt.Errorf("failed to query Django database: %w", err)
		}
		defer rows.Close()

		batchCount := 0
		for rows.Next() {
			var loanID string
			var firstPaymentDueDate sql.NullTime

			if err := rows.Scan(&loanID, &firstPaymentDueDate); err != nil {
				log.Printf("❌ Failed to scan row: %v", err)
				errorCount++
				continue
			}

			// Skip if first_payment_due_date is NULL
			if !firstPaymentDueDate.Valid {
				continue
			}

			// Update the loan in SeedsMetrics
			updateQuery := `
				UPDATE loans
				SET first_payment_due_date = $1, updated_at = NOW()
				WHERE loan_id = $2
			`

			result, err := seedsDB.ExecContext(ctx, updateQuery, firstPaymentDueDate.Time, loanID)
			if err != nil {
				log.Printf("❌ Failed to update loan %s: %v", loanID, err)
				errorCount++
				continue
			}

			rowsAffected, err := result.RowsAffected()
			if err != nil {
				log.Printf("❌ Failed to get rows affected for loan %s: %v", loanID, err)
				errorCount++
				continue
			}

			if rowsAffected > 0 {
				totalUpdated++
				batchCount++
			}
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("error iterating rows: %w", err)
		}

		if batchCount > 0 {
			log.Printf("   Updated %d loans (batch offset=%d)", batchCount, offset)
		}

		// Move to next batch
		offset += batchSize

		// If we got fewer than batchSize, we're done
		if batchCount < batchSize {
			break
		}

		// Small delay to avoid overwhelming the database
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("\n✅ first_payment_due_date sync complete: %d updated, %d errors", totalUpdated, errorCount)

	// Verify the update
	verifyQuery := `
		SELECT 
			COUNT(*) as total_loans,
			COUNT(CASE WHEN (first_payment_due_date - disbursement_date) = 30 THEN 1 END) as loans_with_30_day_gap,
			COUNT(CASE WHEN (first_payment_due_date - disbursement_date) != 30 THEN 1 END) as loans_with_correct_gap
		FROM loans
		WHERE first_payment_due_date IS NOT NULL
	`

	var totalLoans, loansWith30DayGap, loansWithCorrectGap int
	if err := seedsDB.QueryRowContext(ctx, verifyQuery).Scan(&totalLoans, &loansWith30DayGap, &loansWithCorrectGap); err != nil {
		return fmt.Errorf("failed to verify update: %w", err)
	}

	log.Printf("\n📊 Verification Results:")
	log.Printf("   Total loans with first_payment_due_date: %d", totalLoans)
	log.Printf("   Loans with 30-day gap (old default): %d", loansWith30DayGap)
	log.Printf("   Loans with correct gap (from Django): %d", loansWithCorrectGap)

	return nil
}

