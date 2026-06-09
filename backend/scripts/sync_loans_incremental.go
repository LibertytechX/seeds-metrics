package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/lib/pq"
	"github.com/seeds-metrics/analytics-backend/internal/config"
	"github.com/seeds-metrics/analytics-backend/internal/models"
	"github.com/seeds-metrics/analytics-backend/internal/repository"
	"github.com/seeds-metrics/analytics-backend/pkg/database"
	"github.com/shopspring/decimal"
)

func main() {
	log.Println("🚀 Starting incremental loan sync...")
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	seedsDB, err := database.NewPostgresDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to SeedsMetrics database: %v", err)
	}
	defer seedsDB.Close()

	djangoDB, err := database.NewPostgresDB(&cfg.DjangoDatabase)
	if err != nil {
		log.Fatalf("Failed to connect to Django database: %v", err)
	}
	defer djangoDB.Close()

	ctx := context.Background()
	err = syncLoansIncremental(
		ctx,
		seedsDB.DB,
		repository.NewDjangoRepository(djangoDB.DB),
		repository.NewCustomerRepository(seedsDB),
		repository.NewOfficerRepository(seedsDB),
		repository.NewLoanRepository(seedsDB),
	)
	if err != nil {
		log.Fatalf("Failed to sync loans incrementally: %v", err)
	}
	log.Println("✅ Incremental loan sync completed successfully")
}

func syncLoansIncremental(ctx context.Context, seedsDB *sql.DB, djangoRepo *repository.DjangoRepository, customerRepo *repository.CustomerRepository, officerRepo *repository.OfficerRepository, loanRepo *repository.LoanRepository) error {
	startTime := time.Now()
	lookback := getLookbackDuration()
	since, err := getLoanSyncStart(ctx, seedsDB, lookback)
	if err != nil {
		return err
	}

	log.Printf("📅 Syncing Django loans changed/disbursed since %s (lookback=%s)", since.Format(time.RFC3339), lookback)

	const batchSize = 500
	offset := 0
	totalSynced := 0
	errorCount := 0
	lastLoanID := ""
	syncedLoanIDs := []string{}

	for {
		loans, err := djangoRepo.GetLoansChangedSince(ctx, since, batchSize, offset)
		if err != nil {
			return fmt.Errorf("failed to fetch changed Django loans: %w", err)
		}
		if len(loans) == 0 {
			break
		}

		log.Printf("Processing changed-loan batch: offset=%d, count=%d", offset, len(loans))
		for _, loanData := range loans {
			input, customerInput, officerInput, err := buildLoanInputs(loanData)
			if err != nil {
				log.Printf("⚠️  Skipping loan with invalid data: %v", err)
				errorCount++
				continue
			}

			if err := customerRepo.Create(ctx, customerInput); err != nil {
				log.Printf("❌ Failed to sync customer %s for loan %s: %v", customerInput.CustomerID, input.LoanID, err)
				errorCount++
				continue
			}
			if err := officerRepo.Create(ctx, officerInput); err != nil {
				log.Printf("❌ Failed to sync officer %s for loan %s: %v", officerInput.OfficerID, input.LoanID, err)
				errorCount++
				continue
			}
			if err := loanRepo.Create(ctx, input); err != nil {
				log.Printf("❌ Failed to sync loan %s: %v", input.LoanID, err)
				errorCount++
				continue
			}

			totalSynced++
			lastLoanID = input.LoanID
			syncedLoanIDs = append(syncedLoanIDs, input.LoanID)
		}

		offset += batchSize
		if len(loans) < batchSize {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	normalizedRows, err := normalizeSyncedLoanBalances(ctx, seedsDB, syncedLoanIDs)
	if err != nil {
		log.Printf("⚠️  Failed to normalize changed loan balances: %v", err)
		errorCount++
	} else if normalizedRows > 0 {
		log.Printf("🧮 Normalized computed balances for %d recently changed loans", normalizedRows)
	}

	duration := int(time.Since(startTime).Milliseconds())
	if err := updateLoanSyncTracking(ctx, seedsDB, totalSynced, errorCount, duration, lastLoanID); err != nil {
		log.Printf("⚠️  Failed to update loan sync tracking: %v", err)
	}

	log.Printf("✅ Incremental loan sync complete: %d successful, %d errors, %dms", totalSynced, errorCount, duration)
	return nil
}

func normalizeSyncedLoanBalances(ctx context.Context, seedsDB *sql.DB, loanIDs []string) (int64, error) {
	if len(loanIDs) == 0 {
		return 0, nil
	}
	result, err := seedsDB.ExecContext(ctx, `
		UPDATE loans
		SET
			principal_outstanding = GREATEST(0, COALESCE(loan_amount, 0) - COALESCE(total_principal_paid, 0)),
			interest_outstanding = GREATEST(
				0,
				(COALESCE(loan_amount, 0) * COALESCE(interest_rate, 0) * COALESCE(loan_term_days, 0) / 365)
				- COALESCE(total_interest_paid, 0)
			),
			fees_outstanding = GREATEST(0, COALESCE(fee_amount, 0) - COALESCE(total_fees_paid, 0)),
			total_outstanding = GREATEST(0, COALESCE(repayment_amount, 0) - COALESCE(total_repayments, 0)),
			actual_outstanding = LEAST(
				COALESCE(actual_outstanding, 0),
				GREATEST(0, COALESCE(repayment_amount, 0) - COALESCE(total_repayments, 0))
			),
			daily_repayment_amount = CASE
				WHEN COALESCE(loan_term_days, 0) > 0 AND COALESCE(repayment_amount, 0) > 0
					THEN COALESCE(repayment_amount, 0) / loan_term_days
				ELSE 0
			END,
			repayment_days_paid = CASE
				WHEN COALESCE(loan_term_days, 0) > 0 AND COALESCE(repayment_amount, 0) > 0
					THEN COALESCE(total_repayments, 0) / (COALESCE(repayment_amount, 0) / loan_term_days)
				ELSE 0
			END
		WHERE loan_id = ANY($1)
	`, pq.Array(loanIDs))
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func getLookbackDuration() time.Duration {
	hours := 72
	if raw := os.Getenv("LOAN_SYNC_LOOKBACK_HOURS"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			hours = parsed
		}
	}
	return time.Duration(hours) * time.Hour
}

func getLoanSyncStart(ctx context.Context, seedsDB *sql.DB, lookback time.Duration) (time.Time, error) {
	_, err := seedsDB.ExecContext(ctx, `
		INSERT INTO sync_tracking (entity_type, last_sync_timestamp, sync_count)
		VALUES ('loans', NULL, 0)
		ON CONFLICT (entity_type) DO NOTHING
	`)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to ensure loan sync tracking row: %w", err)
	}

	var lastSync sql.NullTime
	if err := seedsDB.QueryRowContext(ctx, `SELECT last_sync_timestamp FROM sync_tracking WHERE entity_type = 'loans'`).Scan(&lastSync); err != nil {
		return time.Time{}, fmt.Errorf("failed to read loan sync timestamp: %w", err)
	}
	if lastSync.Valid {
		return lastSync.Time.Add(-lookback), nil
	}

	var maxCreated sql.NullTime
	if err := seedsDB.QueryRowContext(ctx, `SELECT MAX(created_at) FROM loans`).Scan(&maxCreated); err != nil {
		return time.Time{}, fmt.Errorf("failed to read max loan created_at: %w", err)
	}
	if maxCreated.Valid {
		return maxCreated.Time.Add(-lookback), nil
	}
	return time.Now().Add(-7 * 24 * time.Hour), nil
}

func updateLoanSyncTracking(ctx context.Context, seedsDB *sql.DB, synced, errors, durationMS int, lastLoanID string) error {
	_, err := seedsDB.ExecContext(ctx, `
		INSERT INTO sync_tracking (
			entity_type, last_sync_timestamp, last_synced_record_id,
			sync_count, last_sync_records_count, last_sync_errors_count,
			last_sync_duration_ms, updated_at
		) VALUES ('loans', CURRENT_TIMESTAMP, $4, 1, $1, $2, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (entity_type) DO UPDATE SET
			last_sync_timestamp = CURRENT_TIMESTAMP,
			last_synced_record_id = EXCLUDED.last_synced_record_id,
			sync_count = sync_tracking.sync_count + 1,
			last_sync_records_count = EXCLUDED.last_sync_records_count,
			last_sync_errors_count = EXCLUDED.last_sync_errors_count,
			last_sync_duration_ms = EXCLUDED.last_sync_duration_ms,
			updated_at = CURRENT_TIMESTAMP
	`, synced, errors, durationMS, lastLoanID)
	return err
}

func buildLoanInputs(loanData map[string]interface{}) (*models.LoanInput, *models.CustomerInput, *models.OfficerInput, error) {
	loanID, _ := loanData["loan_id"].(string)
	customerID, _ := loanData["customer_id"].(string)
	customerName, _ := loanData["customer_name"].(string)
	officerID, _ := loanData["officer_id"].(string)
	officerName, _ := loanData["officer_name"].(string)
	branch, _ := loanData["branch"].(string)
	region, _ := loanData["region"].(string)
	loanAmount, _ := loanData["loan_amount"].(float64)
	loanTermDays, _ := loanData["loan_term_days"].(int)
	status, _ := loanData["status"].(string)
	channel, _ := loanData["channel"].(string)
	disbursementDate, _ := loanData["disbursement_date"].(string)
	maturityDate, _ := loanData["maturity_date"].(string)
	if loanID == "" || customerID == "" || officerID == "" || disbursementDate == "" || maturityDate == "" {
		return nil, nil, nil, fmt.Errorf("missing essential fields for loan_id=%q customer_id=%q officer_id=%q disbursement=%q maturity=%q", loanID, customerID, officerID, disbursementDate, maturityDate)
	}

	input := &models.LoanInput{LoanID: loanID, CustomerID: customerID, CustomerName: customerName, OfficerID: officerID, OfficerName: officerName, Branch: branch, Region: region, LoanAmount: decimal.NewFromFloat(loanAmount), LoanTermDays: loanTermDays, Status: status, Channel: channel, DisbursementDate: disbursementDate, MaturityDate: maturityDate}
	if v, ok := loanData["customer_phone"].(string); ok && v != "" {
		input.CustomerPhone = &v
	}
	if v, ok := loanData["officer_phone"].(string); ok && v != "" {
		input.OfficerPhone = &v
	}
	if v, ok := loanData["django_status"].(string); ok && v != "" {
		input.DjangoStatus = &v
	}
	if v, ok := loanData["performance_status"].(string); ok && v != "" {
		input.PerformanceStatus = &v
	}
	if v, ok := loanData["loan_type"].(string); ok && v != "" {
		input.LoanType = &v
	}
	if v, ok := loanData["verification_status"].(string); ok && v != "" {
		input.VerificationStatus = &v
	}
	if v, ok := loanData["first_payment_due_date"].(string); ok && v != "" {
		input.FirstPaymentDueDate = &v
	}
	if v, ok := loanData["repayment_amount"].(float64); ok && v > 0 {
		amt := decimal.NewFromFloat(v)
		input.RepaymentAmount = &amt
	}
	if v, ok := loanData["interest_rate"].(float64); ok && v > 0 {
		rate := decimal.NewFromFloat(v)
		input.InterestRate = &rate
	}
	if v, ok := loanData["fee_amount"].(float64); ok && v > 0 {
		fee := decimal.NewFromFloat(v)
		input.FeeAmount = &fee
	}
	if v, ok := loanData["is_disbursed"].(bool); ok {
		input.IsDisbursed = &v
	}
	if v, ok := loanData["supervisor_disbursement_status"].(string); ok && v != "" {
		input.SupervisorDisbursementStatus = &v
	}

	customer := &models.CustomerInput{CustomerID: customerID, CustomerName: customerName, CustomerPhone: input.CustomerPhone}
	officer := &models.OfficerInput{OfficerID: officerID, OfficerName: officerName, OfficerPhone: input.OfficerPhone, Region: region, Branch: branch, EmploymentStatus: "Active"}
	return input, customer, officer, nil
}
