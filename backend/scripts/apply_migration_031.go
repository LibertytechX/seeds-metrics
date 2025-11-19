package main

import (
	"fmt"
	"log"
	"os"

	"github.com/seeds-metrics/analytics-backend/internal/config"
	"github.com/seeds-metrics/analytics-backend/pkg/database"
)

func main() {
	log.Println("🚀 Applying migration 031: Fix actual_outstanding in recalculate function...")

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
	migrationSQL, err := os.ReadFile("../migrations/031_fix_actual_outstanding_in_recalculate.sql")
	if err != nil {
		log.Fatalf("Failed to read migration file: %v", err)
	}

	// Execute migration
	log.Println("📝 Executing migration...")
	_, err = db.DB.Exec(string(migrationSQL))
	if err != nil {
		log.Fatalf("Failed to execute migration: %v", err)
	}

	log.Println("✅ Migration 031 applied successfully!")
	log.Println("📊 The recalculate_all_loan_fields() function now includes actual_outstanding field updates")
	
	// Test the function
	log.Println("\n🧪 Testing the updated function...")
	var totalLoans, loansUpdated, executionTime int
	err = db.DB.QueryRow("SELECT * FROM recalculate_all_loan_fields()").Scan(&totalLoans, &loansUpdated, &executionTime)
	if err != nil {
		log.Fatalf("Failed to test recalculate function: %v", err)
	}

	log.Printf("✅ Test successful!")
	log.Printf("   - Total loans processed: %d", totalLoans)
	log.Printf("   - Loans updated: %d", loansUpdated)
	log.Printf("   - Execution time: %d ms", executionTime)
	
	fmt.Println("\n🎉 Migration completed and tested successfully!")
}

