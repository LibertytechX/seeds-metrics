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

// Holiday represents a holiday record
type Holiday struct {
	ID        int64
	Date      *time.Time
	Name      *string
	CreatedAt time.Time
	UpdatedAt time.Time
	AgentID   *int64
	BranchID  *int64
	CreatedBy *int64
	Type      string
	SalaryWaver bool
}

func main() {
	log.Println("🚀 Starting holiday sync from Django to SeedsMetrics...")

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

	// Sync holidays
	log.Println("\n📊 Syncing Holidays...")
	if err := syncHolidays(ctx, seedsDB.DB, djangoDB.DB); err != nil {
		log.Fatalf("Failed to sync holidays: %v", err)
	}

	log.Println("\n✅ Holiday sync completed successfully!")
}

// syncHolidays syncs all holidays from Django to SeedsMetrics
func syncHolidays(ctx context.Context, seedsDB, djangoDB *sql.DB) error {
	startTime := time.Now()

	// Get all holidays from Django
	log.Println("📥 Fetching holidays from Django database...")
	holidays, err := getHolidaysFromDjango(ctx, djangoDB)
	if err != nil {
		return fmt.Errorf("failed to get holidays from Django: %w", err)
	}

	log.Printf("📊 Found %d holidays in Django database", len(holidays))

	if len(holidays) == 0 {
		log.Println("⚠️  No holidays found in Django database")
		return nil
	}

	// Clear existing holidays in SeedsMetrics
	log.Println("🗑️  Clearing existing holidays in SeedsMetrics...")
	_, err = seedsDB.ExecContext(ctx, "TRUNCATE TABLE holiday")
	if err != nil {
		return fmt.Errorf("failed to truncate holiday table: %w", err)
	}

	// Insert holidays into SeedsMetrics
	log.Println("📤 Inserting holidays into SeedsMetrics...")
	successCount := 0
	errorCount := 0

	for _, holiday := range holidays {
		err := insertHoliday(ctx, seedsDB, holiday)
		if err != nil {
			log.Printf("❌ Error inserting holiday %d: %v", holiday.ID, err)
			errorCount++
		} else {
			successCount++
		}
	}

	duration := time.Since(startTime)

	log.Printf("\n✅ Sync completed in %v", duration)
	log.Printf("📊 Successfully inserted: %d holidays", successCount)
	if errorCount > 0 {
		log.Printf("❌ Failed to insert: %d holidays", errorCount)
	}

	// Verify count
	var count int
	err = seedsDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM holiday").Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to verify holiday count: %w", err)
	}

	log.Printf("✅ Verification: %d holidays now in SeedsMetrics database", count)

	if count != len(holidays) {
		log.Printf("⚠️  Warning: Expected %d holidays but found %d", len(holidays), count)
	}

	return nil
}

// getHolidaysFromDjango fetches all holidays from Django database
func getHolidaysFromDjango(ctx context.Context, db *sql.DB) ([]*Holiday, error) {
	query := `
		SELECT 
			id,
			date,
			name,
			created_at,
			updated_at,
			agent_id,
			branch_id,
			created_by_id,
			type,
			salary_waver
		FROM loans_holiday
		ORDER BY date ASC
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holidays []*Holiday
	for rows.Next() {
		holiday := &Holiday{}
		err := rows.Scan(
			&holiday.ID,
			&holiday.Date,
			&holiday.Name,
			&holiday.CreatedAt,
			&holiday.UpdatedAt,
			&holiday.AgentID,
			&holiday.BranchID,
			&holiday.CreatedBy,
			&holiday.Type,
			&holiday.SalaryWaver,
		)
		if err != nil {
			return nil, err
		}
		holidays = append(holidays, holiday)
	}

	return holidays, rows.Err()
}

// insertHoliday inserts a single holiday into SeedsMetrics
func insertHoliday(ctx context.Context, db *sql.DB, holiday *Holiday) error {
	query := `
		INSERT INTO holiday (
			id, date, name, created_at, updated_at, 
			agent_id, branch_id, created_by_id, type, salary_waver
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := db.ExecContext(ctx, query,
		holiday.ID,
		holiday.Date,
		holiday.Name,
		holiday.CreatedAt,
		holiday.UpdatedAt,
		holiday.AgentID,
		holiday.BranchID,
		holiday.CreatedBy,
		holiday.Type,
		holiday.SalaryWaver,
	)

	return err
}

