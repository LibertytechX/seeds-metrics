package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	// Get database connection string from environment
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	if dbHost == "" || dbPort == "" || dbUser == "" || dbPassword == "" || dbName == "" {
		log.Fatal("Missing required database environment variables")
	}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=require",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("✅ Connected to database")

	// Read migration file
	migrationSQL, err := os.ReadFile("migrations/033_add_performance_status_to_loans.sql")
	if err != nil {
		log.Fatalf("Failed to read migration file: %v", err)
	}

	log.Println("📄 Applying migration 033: Add performance_status to loans table...")

	// Execute migration
	_, err = db.Exec(string(migrationSQL))
	if err != nil {
		log.Fatalf("❌ Failed to apply migration: %v", err)
	}

	log.Println("✅ Migration 033 applied successfully!")

	// Verify the column was added
	var columnExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 
			FROM information_schema.columns 
			WHERE table_name = 'loans' 
			AND column_name = 'performance_status'
		)
	`).Scan(&columnExists)

	if err != nil {
		log.Fatalf("Failed to verify column: %v", err)
	}

	if columnExists {
		log.Println("✅ Verified: performance_status column exists in loans table")
	} else {
		log.Println("⚠️  Warning: performance_status column not found after migration")
	}

	// Check if index was created
	var indexExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 
			FROM pg_indexes 
			WHERE tablename = 'loans' 
			AND indexname = 'idx_loans_performance_status'
		)
	`).Scan(&indexExists)

	if err != nil {
		log.Fatalf("Failed to verify index: %v", err)
	}

	if indexExists {
		log.Println("✅ Verified: idx_loans_performance_status index created")
	} else {
		log.Println("⚠️  Warning: idx_loans_performance_status index not found after migration")
	}

	log.Println("🎉 Migration 033 completed successfully!")
}

