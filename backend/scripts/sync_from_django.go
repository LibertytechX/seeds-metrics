package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/seeds-metrics/analytics-backend/internal/config"
	"github.com/seeds-metrics/analytics-backend/internal/models"
	"github.com/seeds-metrics/analytics-backend/internal/repository"
	"github.com/seeds-metrics/analytics-backend/pkg/database"
	"github.com/shopspring/decimal"
)

func main() {
	log.Println("🚀 Starting Django to SeedsMetrics data sync...")

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
	officerRepo := repository.NewOfficerRepository(seedsDB)
	customerRepo := repository.NewCustomerRepository(seedsDB)
	loanRepo := repository.NewLoanRepository(seedsDB)

	ctx := context.Background()

	// Sync Officers
	log.Println("\n📊 Syncing Officers...")
	if err := syncOfficers(ctx, djangoRepo, officerRepo); err != nil {
		log.Fatalf("Failed to sync officers: %v", err)
	}

	// Sync Customers
	log.Println("\n📊 Syncing Customers...")
	if err := syncCustomers(ctx, djangoRepo, customerRepo); err != nil {
		log.Fatalf("Failed to sync customers: %v", err)
	}

	// Sync Loans
	log.Println("\n📊 Syncing Loans...")
	if err := syncLoans(ctx, djangoRepo, loanRepo); err != nil {
		log.Fatalf("Failed to sync loans: %v", err)
	}

	log.Println("\n✅ Data sync completed successfully!")
	log.Println("\n📝 Note: Repayments and schedules sync will be implemented in Phase 2")
}

func syncOfficers(ctx context.Context, djangoRepo *repository.DjangoRepository, officerRepo *repository.OfficerRepository) error {
	// Get officers from Django
	officers, err := djangoRepo.GetOfficers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get officers from Django: %w", err)
	}

	log.Printf("Found %d officers in Django database", len(officers))

	// Insert/update officers in SeedsMetrics
	successCount := 0
	errorCount := 0

	for _, officer := range officers {
		// Convert to OfficerInput
		input := &models.OfficerInput{
			OfficerID:        officer.OfficerID,
			OfficerName:      officer.OfficerName,
			OfficerPhone:     officer.OfficerPhone,
			Region:           officer.Region,
			Branch:           officer.Branch,
			EmploymentStatus: officer.EmploymentStatus,
		}

		// Convert hire date to string if present
		if officer.HireDate != nil {
			hireDateStr := officer.HireDate.Format("2006-01-02")
			input.HireDate = &hireDateStr
		}

		// Create/update officer
		if err := officerRepo.Create(ctx, input); err != nil {
			log.Printf("❌ Failed to sync officer %s: %v", officer.OfficerID, err)
			errorCount++
		} else {
			successCount++
			if successCount%100 == 0 {
				log.Printf("   Synced %d officers...", successCount)
			}
		}
	}

	log.Printf("✅ Officers sync complete: %d successful, %d errors", successCount, errorCount)
	return nil
}

func syncCustomers(ctx context.Context, djangoRepo *repository.DjangoRepository, customerRepo *repository.CustomerRepository) error {
	// Get customers from Django in batches
	limit := 1000
	offset := 0
	totalSynced := 0
	errorCount := 0

	for {
		customers, err := djangoRepo.GetCustomers(ctx, limit, offset)
		if err != nil {
			return fmt.Errorf("failed to get customers from Django: %w", err)
		}

		if len(customers) == 0 {
			break
		}

		log.Printf("Processing batch: offset=%d, count=%d", offset, len(customers))

		// Insert/update customers in SeedsMetrics
		for _, customer := range customers {
			// Convert to CustomerInput
			input := &models.CustomerInput{
				CustomerID:    customer.CustomerID,
				CustomerName:  customer.CustomerName,
				CustomerPhone: customer.CustomerPhone,
				CustomerEmail: customer.CustomerEmail,
			}

			// Create/update customer
			if err := customerRepo.Create(ctx, input); err != nil {
				log.Printf("❌ Failed to sync customer %s: %v", customer.CustomerID, err)
				errorCount++
			} else {
				totalSynced++
				if totalSynced%500 == 0 {
					log.Printf("   Synced %d customers...", totalSynced)
				}
			}
		}

		// Move to next batch
		offset += limit

		// If we got fewer than limit, we're done
		if len(customers) < limit {
			break
		}

		// Small delay to avoid overwhelming the database
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("✅ Customers sync complete: %d successful, %d errors", totalSynced, errorCount)
	return nil
}

func syncLoans(ctx context.Context, djangoRepo *repository.DjangoRepository, loanRepo *repository.LoanRepository) error {
	const batchSize = 500
	offset := 0
	totalSynced := 0
	errorCount := 0

	for {
		// Get loans in batches
		loans, err := djangoRepo.GetLoans(ctx, batchSize, offset)
		if err != nil {
			return fmt.Errorf("failed to get loans from Django: %w", err)
		}

		if len(loans) == 0 {
			break
		}

		log.Printf("Processing batch: offset=%d, count=%d", offset, len(loans))

		for _, loanData := range loans {
			// Convert map to LoanInput
			input := &models.LoanInput{
				LoanID:           loanData["loan_id"].(string),
				CustomerID:       loanData["customer_id"].(string),
				CustomerName:     loanData["customer_name"].(string),
				OfficerID:        loanData["officer_id"].(string),
				OfficerName:      loanData["officer_name"].(string),
				Branch:           loanData["branch"].(string),
				Region:           loanData["region"].(string),
				LoanAmount:       decimal.NewFromFloat(loanData["loan_amount"].(float64)),
				LoanTermDays:     loanData["loan_term_days"].(int),
				Status:           loanData["status"].(string),
				Channel:          loanData["channel"].(string),
				DisbursementDate: loanData["disbursement_date"].(string),
				MaturityDate:     loanData["maturity_date"].(string),
			}

			// Optional fields
			if customerPhone, ok := loanData["customer_phone"].(string); ok {
				input.CustomerPhone = &customerPhone
			}
			if officerPhone, ok := loanData["officer_phone"].(string); ok {
				input.OfficerPhone = &officerPhone
			}

			// Decimal fields
			repaymentAmt := decimal.NewFromFloat(loanData["repayment_amount"].(float64))
			input.RepaymentAmount = &repaymentAmt

			interestRate := decimal.NewFromFloat(loanData["interest_rate"].(float64))
			input.InterestRate = &interestRate

			feeAmount := decimal.NewFromFloat(loanData["fee_amount"].(float64))
			input.FeeAmount = &feeAmount

			// Create/update loan
			if err := loanRepo.Create(ctx, input); err != nil {
				log.Printf("❌ Failed to sync loan %s: %v", input.LoanID, err)
				errorCount++
			} else {
				totalSynced++
				if totalSynced%100 == 0 {
					log.Printf("   Synced %d loans...", totalSynced)
				}
			}
		}

		// Move to next batch
		offset += batchSize

		// If we got fewer than batchSize, we're done
		if len(loans) < batchSize {
			break
		}

		// Small delay to avoid overwhelming the database
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("✅ Loans sync complete: %d successful, %d errors", totalSynced, errorCount)
	return nil
}
