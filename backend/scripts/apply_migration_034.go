package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	// Get database connection details from environment variables
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	// Validate required environment variables
	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
		log.Fatal("❌ Missing required database environment variables (DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME)")
	}

	// Build connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	// Connect to database
	log.Println("🔌 Connecting to database...")
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("❌ Failed to ping database: %v", err)
	}
	log.Println("✅ Database connection established")

	// Read migration file
	log.Println("📖 Reading migration file...")
	migrationSQL, err := os.ReadFile("migrations/034_fix_total_outstanding_calculation.sql")
	if err != nil {
		log.Fatalf("❌ Failed to read migration file: %v", err)
	}

	// Execute migration
	log.Println("🚀 Applying migration 034: Fix total_outstanding calculation...")
	startTime := time.Now()

	_, err = db.Exec(string(migrationSQL))
	if err != nil {
		log.Fatalf("❌ Failed to apply migration: %v", err)
	}

	duration := time.Since(startTime)
	log.Printf("✅ Migration 034 applied successfully in %v", duration)

	// Verify the function was created
	log.Println("🔍 Verifying recalculate_all_loan_fields() function...")
	var functionExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 
			FROM pg_proc p
			JOIN pg_namespace n ON p.pronamespace = n.oid
			WHERE n.nspname = 'public' 
			AND p.proname = 'recalculate_all_loan_fields'
		)
	`).Scan(&functionExists)

	if err != nil {
		log.Fatalf("❌ Failed to verify function: %v", err)
	}

	if !functionExists {
		log.Fatal("❌ Function recalculate_all_loan_fields() was not created")
	}

	log.Println("✅ Function recalculate_all_loan_fields() verified")

	// Run the recalculate function to fix all loans
	log.Println("🔄 Running recalculate_all_loan_fields() to fix all loan outstanding amounts...")
	
	var totalLoans, loansUpdated, executionTimeMs int
	err = db.QueryRow("SELECT * FROM recalculate_all_loan_fields()").Scan(&totalLoans, &loansUpdated, &executionTimeMs)
	if err != nil {
		log.Fatalf("❌ Failed to run recalculate function: %v", err)
	}

	log.Printf("✅ Recalculation complete:")
	log.Printf("   - Total loans processed: %d", totalLoans)
	log.Printf("   - Loans updated: %d", loansUpdated)
	log.Printf("   - Execution time: %d ms", executionTimeMs)

	// Verify loan 8205 is now fixed
	log.Println("🔍 Verifying loan 8205 fix...")
	var loanID string
	var totalOutstanding, repaymentAmount, totalRepayments float64
	err = db.QueryRow(`
		SELECT 
			loan_id,
			total_outstanding,
			repayment_amount,
			total_repayments
		FROM loans 
		WHERE loan_id = '8205'
	`).Scan(&loanID, &totalOutstanding, &repaymentAmount, &totalRepayments)

	if err != nil {
		log.Printf("⚠️  Could not verify loan 8205: %v", err)
	} else {
		log.Printf("✅ Loan 8205 verification:")
		log.Printf("   - Repayment Amount: %.2f", repaymentAmount)
		log.Printf("   - Total Repayments: %.2f", totalRepayments)
		log.Printf("   - Total Outstanding: %.2f", totalOutstanding)
		log.Printf("   - Expected Outstanding: %.2f", repaymentAmount-totalRepayments)
		
		if totalOutstanding < 0 {
			log.Printf("⚠️  WARNING: Total outstanding is still negative!")
		} else if totalOutstanding == repaymentAmount-totalRepayments {
			log.Printf("✅ Total outstanding is correct!")
		} else {
			log.Printf("⚠️  WARNING: Total outstanding doesn't match expected value")
		}
	}

	log.Println("🎉 Migration 034 completed successfully!")
}

