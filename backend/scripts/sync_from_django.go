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
	repaymentRepo := repository.NewRepaymentRepository(seedsDB)

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

	// Sync Repayments
	log.Println("\n📊 Syncing Repayments...")
	if err := syncRepayments(ctx, djangoRepo, repaymentRepo); err != nil {
		log.Fatalf("Failed to sync repayments: %v", err)
	}

	log.Println("\n✅ Data sync completed successfully!")
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
			UserType:         officer.UserType,
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
			// Convert map to LoanInput with nil-safe type assertions
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
			firstPaymentDueDate, _ := loanData["first_payment_due_date"].(string)
			maturityDate, _ := loanData["maturity_date"].(string)

			// Skip if essential fields are missing
			if loanID == "" || customerID == "" || officerID == "" || disbursementDate == "" || maturityDate == "" {
				log.Printf("⚠️  Skipping loan with missing essential fields: %v", loanData)
				errorCount++
				continue
			}

			input := &models.LoanInput{
				LoanID:           loanID,
				CustomerID:       customerID,
				CustomerName:     customerName,
				OfficerID:        officerID,
				OfficerName:      officerName,
				Branch:           branch,
				Region:           region,
				LoanAmount:       decimal.NewFromFloat(loanAmount),
				LoanTermDays:     loanTermDays,
				Status:           status,
				Channel:          channel,
				DisbursementDate: disbursementDate,
				MaturityDate:     maturityDate,
			}

			// Optional fields
			if customerPhone, ok := loanData["customer_phone"].(string); ok && customerPhone != "" {
				input.CustomerPhone = &customerPhone
			}
			if officerPhone, ok := loanData["officer_phone"].(string); ok && officerPhone != "" {
				input.OfficerPhone = &officerPhone
			}
			if firstPaymentDueDate != "" {
				input.FirstPaymentDueDate = &firstPaymentDueDate
			}

			// Decimal fields
			if repaymentAmt, ok := loanData["repayment_amount"].(float64); ok && repaymentAmt > 0 {
				amt := decimal.NewFromFloat(repaymentAmt)
				input.RepaymentAmount = &amt
			}

			if interestRate, ok := loanData["interest_rate"].(float64); ok && interestRate > 0 {
				rate := decimal.NewFromFloat(interestRate)
				input.InterestRate = &rate
			}

			if feeAmount, ok := loanData["fee_amount"].(float64); ok && feeAmount > 0 {
				fee := decimal.NewFromFloat(feeAmount)
				input.FeeAmount = &fee
			}

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

// syncRepayments syncs repayments from Django to SeedsMetrics
func syncRepayments(ctx context.Context, djangoRepo *repository.DjangoRepository, repaymentRepo *repository.RepaymentRepository) error {
	batchSize := 1000
	offset := 0
	totalSynced := 0
	errorCount := 0

	for {
		log.Printf("Processing batch: offset=%d, count=%d", offset, batchSize)

		// Fetch repayments from Django
		repayments, err := djangoRepo.GetRepayments(ctx, batchSize, offset)
		if err != nil {
			return fmt.Errorf("failed to fetch repayments: %w", err)
		}

		if len(repayments) == 0 {
			break
		}

		// Process each repayment
		for _, repaymentData := range repayments {
			// Convert map to RepaymentInput with nil-safe type assertions
			repaymentID, _ := repaymentData["repayment_id"].(string)
			loanID, _ := repaymentData["loan_id"].(string)
			paymentDate, _ := repaymentData["payment_date"].(string)
			paymentAmount, _ := repaymentData["payment_amount"].(float64)
			paymentMethod, _ := repaymentData["payment_method"].(string)

			// Skip if essential fields are missing
			if repaymentID == "" || loanID == "" || paymentDate == "" || paymentAmount <= 0 {
				log.Printf("⚠️  Skipping repayment with missing essential fields: %v", repaymentData)
				errorCount++
				continue
			}

			// For Django repayments, we don't have breakdown of principal/interest/fees
			// So we'll put the full amount as principal_paid and let the triggers calculate
			input := &models.RepaymentInput{
				RepaymentID:   repaymentID,
				LoanID:        loanID,
				PaymentDate:   paymentDate,
				PaymentAmount: decimal.NewFromFloat(paymentAmount),
				PrincipalPaid: decimal.NewFromFloat(paymentAmount), // Full amount as principal
				InterestPaid:  decimal.Zero,
				FeesPaid:      decimal.Zero,
				PenaltyPaid:   decimal.Zero,
				PaymentMethod: paymentMethod,
				DPDAtPayment:  0,
				IsBackdated:   false,
				IsReversed:    false,
				WaiverAmount:  decimal.Zero,
			}

			// Create/update repayment
			if err := repaymentRepo.Create(ctx, input); err != nil {
				// Check if it's a foreign key constraint error (loan doesn't exist)
				if err.Error() == "loan not found" {
					// Skip silently - this loan wasn't synced
					errorCount++
				} else {
					log.Printf("❌ Failed to sync repayment %s: %v", input.RepaymentID, err)
					errorCount++
				}
			} else {
				totalSynced++
				if totalSynced%500 == 0 {
					log.Printf("   Synced %d repayments...", totalSynced)
				}
			}
		}

		// Move to next batch
		offset += batchSize

		// If we got fewer than batchSize, we're done
		if len(repayments) < batchSize {
			break
		}

		// Small delay to avoid overwhelming the database
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("✅ Repayments sync complete: %d successful, %d errors", totalSynced, errorCount)
	return nil
}
