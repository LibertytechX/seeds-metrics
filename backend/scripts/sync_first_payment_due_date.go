package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// Connect to Django database (source)
	djangoDBURL := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		os.Getenv("DJANGO_DB_HOST"),
		os.Getenv("DJANGO_DB_PORT"),
		os.Getenv("DJANGO_DB_NAME"),
		os.Getenv("DJANGO_DB_USER"),
		os.Getenv("DJANGO_DB_PASSWORD"),
		os.Getenv("DJANGO_DB_SSLMODE"),
	)

	djangoDB, err := sql.Open("postgres", djangoDBURL)
	if err != nil {
		log.Fatalf("Failed to connect to Django database: %v", err)
	}
	defer djangoDB.Close()

	// Connect to SeedsMetrics database (target)
	seedsDBURL := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_SSLMODE"),
	)

	seedsDB, err := sql.Open("postgres", seedsDBURL)
	if err != nil {
		log.Fatalf("Failed to connect to SeedsMetrics database: %v", err)
	}
	defer seedsDB.Close()

	log.Println("✅ Connected to both databases")

	// Fetch loans with start_date from Django
	query := `
		SELECT 
			id::VARCHAR(50) as loan_id,
			start_date
		FROM loans_ajoloan
		WHERE is_disbursed = TRUE
		ORDER BY id
	`

	rows, err := djangoDB.QueryContext(context.Background(), query)
	if err != nil {
		log.Fatalf("Failed to query Django database: %v", err)
	}
	defer rows.Close()

	// Prepare update statement
	updateStmt, err := seedsDB.Prepare(`
		UPDATE loans
		SET first_payment_due_date = $1, updated_at = $2
		WHERE loan_id = $3
	`)
	if err != nil {
		log.Fatalf("Failed to prepare update statement: %v", err)
	}
	defer updateStmt.Close()

	updated := 0
	skipped := 0
	errors := 0

	for rows.Next() {
		var loanID string
		var startDate sql.NullTime

		if err := rows.Scan(&loanID, &startDate); err != nil {
			log.Printf("Error scanning row: %v", err)
			errors++
			continue
		}

		// Update the loan in SeedsMetrics database
		var firstPaymentDueDate interface{}
		if startDate.Valid {
			firstPaymentDueDate = startDate.Time.Format("2006-01-02")
		} else {
			firstPaymentDueDate = nil
		}

		result, err := updateStmt.Exec(firstPaymentDueDate, time.Now(), loanID)
		if err != nil {
			log.Printf("Error updating loan %s: %v", loanID, err)
			errors++
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			updated++
			if updated%100 == 0 {
				log.Printf("Updated %d loans...", updated)
			}
		} else {
			skipped++
		}
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Error iterating rows: %v", err)
	}

	log.Printf("\n✅ Sync complete!")
	log.Printf("   Updated: %d loans", updated)
	log.Printf("   Skipped: %d loans", skipped)
	log.Printf("   Errors: %d", errors)

	// Show sample of updated loans
	var sampleQuery = `
		SELECT loan_id, disbursement_date, first_payment_due_date, maturity_date
		FROM loans
		WHERE first_payment_due_date IS NOT NULL
		ORDER BY loan_id DESC
		LIMIT 5
	`

	sampleRows, err := seedsDB.Query(sampleQuery)
	if err != nil {
		log.Printf("Error fetching sample: %v", err)
		return
	}
	defer sampleRows.Close()

	log.Println("\n📋 Sample of updated loans:")
	for sampleRows.Next() {
		var loanID, disbursementDate, maturityDate string
		var firstPaymentDueDate sql.NullString

		if err := sampleRows.Scan(&loanID, &disbursementDate, &firstPaymentDueDate, &maturityDate); err != nil {
			log.Printf("Error scanning sample: %v", err)
			continue
		}

		fpd := "NULL"
		if firstPaymentDueDate.Valid {
			fpd = firstPaymentDueDate.String
		}

		log.Printf("   Loan %s: Disbursed=%s, FirstPaymentDue=%s, Maturity=%s",
			loanID, disbursementDate, fpd, maturityDate)
	}
}

