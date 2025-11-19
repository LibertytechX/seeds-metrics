package main

import (
	"fmt"
	"log"
	"os"

	"github.com/seeds-metrics/analytics-backend/internal/config"
	"github.com/seeds-metrics/analytics-backend/pkg/database"
)

func main() {
	log.Println("🚀 Applying migration 032: Fix actual_outstanding to use business days...")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to SeedsMetrics database
	db, err := database.NewPostgresDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to SeedsMetrics database: %v", err)
	}
	defer db.Close()
	log.Println("✅ Connected to SeedsMetrics database")

	// Read migration file
	migrationSQL, err := os.ReadFile("migrations/032_fix_actual_outstanding_use_business_days.sql")
	if err != nil {
		log.Fatalf("Failed to read migration file: %v", err)
	}

	// Execute migration
	log.Println("📝 Executing migration...")
	_, err = db.DB.Exec(string(migrationSQL))
	if err != nil {
		log.Fatalf("Failed to execute migration: %v", err)
	}

	log.Println("✅ Migration 032 applied successfully!")
	log.Println("📊 The recalculate_all_loan_fields() function now uses business days for actual_outstanding calculation")

	// Test the function
	log.Println("\n🧪 Testing the updated function and recalculating all loans...")
	var totalLoans, loansUpdated, executionTime int
	err = db.DB.QueryRow("SELECT * FROM recalculate_all_loan_fields()").Scan(&totalLoans, &loansUpdated, &executionTime)
	if err != nil {
		log.Fatalf("Failed to test recalculate function: %v", err)
	}

	log.Printf("✅ Recalculation completed successfully!")
	log.Printf("   - Total loans processed: %d", totalLoans)
	log.Printf("   - Loans updated: %d", loansUpdated)
	log.Printf("   - Execution time: %d ms", executionTime)

	// Verify the changes by checking a sample loan
	log.Println("\n🔍 Verifying actual_outstanding values for sample loans...")
	type LoanSample struct {
		LoanID                       int
		LoanAmount                   float64
		TotalRepayments              float64
		ActualOutstanding            float64
		BusinessDaysSinceDisbursement int
		LoanTermDays                 int
	}

	rows, err := db.DB.Query(`
		SELECT 
			loan_id,
			loan_amount,
			total_repayments,
			actual_outstanding,
			business_days_since_disbursement,
			loan_term_days
		FROM loans
		WHERE status = 'Active' AND actual_outstanding > 0
		ORDER BY actual_outstanding DESC
		LIMIT 5
	`)
	if err != nil {
		log.Fatalf("Failed to query sample loans: %v", err)
	}
	defer rows.Close()

	log.Println("\nTop 5 loans by actual_outstanding (using business days):")
	log.Println("┌──────────┬──────────────┬──────────────────┬────────────────────┬──────────────────────────────┬──────────────┐")
	log.Println("│ Loan ID  │ Loan Amount  │ Total Repayments │ Actual Outstanding │ Business Days Since Disb.    │ Loan Term    │")
	log.Println("├──────────┼──────────────┼──────────────────┼────────────────────┼──────────────────────────────┼──────────────┤")

	for rows.Next() {
		var loan LoanSample
		err := rows.Scan(
			&loan.LoanID,
			&loan.LoanAmount,
			&loan.TotalRepayments,
			&loan.ActualOutstanding,
			&loan.BusinessDaysSinceDisbursement,
			&loan.LoanTermDays,
		)
		if err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		log.Printf("│ %-8d │ %12.2f │ %16.2f │ %18.2f │ %28d │ %12d │",
			loan.LoanID,
			loan.LoanAmount,
			loan.TotalRepayments,
			loan.ActualOutstanding,
			loan.BusinessDaysSinceDisbursement,
			loan.LoanTermDays,
		)
	}
	log.Println("└──────────┴──────────────┴──────────────────┴────────────────────┴──────────────────────────────┴──────────────┘")

	fmt.Println("\n🎉 Migration 032 completed and verified successfully!")
	fmt.Println("📌 The actual_outstanding field now correctly uses business days instead of calendar days")
}

