package main

import (
	"context"
	"fmt"
	"log"

	"github.com/seeds-metrics/analytics-backend/internal/config"
	"github.com/seeds-metrics/analytics-backend/pkg/database"
)

func main() {
	log.Println("🔍 Checking loan 19808 in Django database...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to Django database (read-only)
	djangoDB, err := database.NewPostgresDB(&cfg.DjangoDatabase)
	if err != nil {
		log.Fatalf("Failed to connect to Django database: %v", err)
	}
	defer djangoDB.Close()
	log.Println("✅ Connected to Django database")

	ctx := context.Background()

	// Query recent loans
	query := `
		SELECT
			id as loan_id,
			borrower_id,
			date_disbursed,
			start_date,
			end_date,
			status,
			amount,
			tenor_in_days
		FROM loans_ajoloan
		WHERE id IN (19808, 19807, 19806, 19805, 19804)
		ORDER BY id DESC
	`

	rows, err := djangoDB.DB.QueryContext(ctx, query)
	if err != nil {
		log.Fatalf("Failed to query loans: %v", err)
	}
	defer rows.Close()

	fmt.Println("\n📊 Recent Loans Data from Django:")
	fmt.Println("================================================================================")

	for rows.Next() {
		var loanID, borrowerID, status string
		var dateDisbursed, startDate, endDate interface{}
		var loanAmount float64
		var tenorInDays int

		err = rows.Scan(
			&loanID,
			&borrowerID,
			&dateDisbursed,
			&startDate,
			&endDate,
			&status,
			&loanAmount,
			&tenorInDays,
		)
		if err != nil {
			log.Fatalf("Failed to scan loan: %v", err)
		}

		fmt.Printf("\nLoan ID: %s\n", loanID)
		fmt.Printf("  Borrower ID: %s\n", borrowerID)
		fmt.Printf("  Date Disbursed: %v\n", dateDisbursed)
		fmt.Printf("  Start Date (First Payment Due): %v\n", startDate)
		fmt.Printf("  End Date (Maturity): %v\n", endDate)
		fmt.Printf("  Status: %s\n", status)
		fmt.Printf("  Loan Amount: %.2f\n", loanAmount)
		fmt.Printf("  Tenor (Days): %d\n", tenorInDays)
		fmt.Println("--------------------------------------------------------------------------------")
	}

	if err = rows.Err(); err != nil {
		log.Fatalf("Error iterating rows: %v", err)
	}
}
