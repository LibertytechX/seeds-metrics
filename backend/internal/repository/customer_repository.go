package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/seeds-metrics/analytics-backend/internal/models"
	"github.com/seeds-metrics/analytics-backend/pkg/database"
)

type CustomerRepository struct {
	db *database.DB
}

func NewCustomerRepository(db *database.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

// Create inserts a new customer
func (r *CustomerRepository) Create(ctx context.Context, input *models.CustomerInput) error {
	query := `
		INSERT INTO customers (
			customer_id, customer_name, customer_phone, customer_email,
			date_of_birth, gender, state, lga, address,
			kyc_status, kyc_verified_date,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW()
		)
		ON CONFLICT (customer_id) DO UPDATE SET
			customer_name = EXCLUDED.customer_name,
			customer_phone = EXCLUDED.customer_phone,
			customer_email = EXCLUDED.customer_email,
			date_of_birth = EXCLUDED.date_of_birth,
			gender = EXCLUDED.gender,
			state = EXCLUDED.state,
			lga = EXCLUDED.lga,
			address = EXCLUDED.address,
			kyc_status = EXCLUDED.kyc_status,
			kyc_verified_date = EXCLUDED.kyc_verified_date,
			updated_at = NOW()
	`

	// Parse date of birth if provided
	var dateOfBirth *time.Time
	if input.DateOfBirth != nil {
		parsed, err := time.Parse("2006-01-02", *input.DateOfBirth)
		if err != nil {
			return fmt.Errorf("invalid date_of_birth format: %w", err)
		}
		dateOfBirth = &parsed
	}

	// Parse KYC verified date if provided
	var kycVerifiedDate *time.Time
	if input.KYCVerifiedDate != nil {
		parsed, err := time.Parse("2006-01-02", *input.KYCVerifiedDate)
		if err != nil {
			return fmt.Errorf("invalid kyc_verified_date format: %w", err)
		}
		kycVerifiedDate = &parsed
	}

	_, err := r.db.ExecContext(ctx, query,
		input.CustomerID, input.CustomerName, input.CustomerPhone, input.CustomerEmail,
		dateOfBirth, input.Gender, input.State, input.LGA, input.Address,
		input.KYCStatus, kycVerifiedDate,
	)

	return err
}

// GetByID retrieves a customer by ID
func (r *CustomerRepository) GetByID(ctx context.Context, customerID string) (*models.Customer, error) {
	query := `
		SELECT 
			customer_id, customer_name, customer_phone, customer_email,
			date_of_birth, gender, state, lga, address,
			kyc_status, kyc_verified_date,
			created_at, updated_at
		FROM customers
		WHERE customer_id = $1
	`

	var customer models.Customer
	err := r.db.QueryRowContext(ctx, query, customerID).Scan(
		&customer.CustomerID, &customer.CustomerName, &customer.CustomerPhone, &customer.CustomerEmail,
		&customer.DateOfBirth, &customer.Gender, &customer.State, &customer.LGA, &customer.Address,
		&customer.KYCStatus, &customer.KYCVerifiedDate,
		&customer.CreatedAt, &customer.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &customer, nil
}

// List retrieves all customers
func (r *CustomerRepository) List(ctx context.Context) ([]*models.Customer, error) {
	query := `
		SELECT 
			customer_id, customer_name, customer_phone, customer_email,
			date_of_birth, gender, state, lga, address,
			kyc_status, kyc_verified_date,
			created_at, updated_at
		FROM customers
		ORDER BY customer_name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var customers []*models.Customer
	for rows.Next() {
		var customer models.Customer
		err := rows.Scan(
			&customer.CustomerID, &customer.CustomerName, &customer.CustomerPhone, &customer.CustomerEmail,
			&customer.DateOfBirth, &customer.Gender, &customer.State, &customer.LGA, &customer.Address,
			&customer.KYCStatus, &customer.KYCVerifiedDate,
			&customer.CreatedAt, &customer.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		customers = append(customers, &customer)
	}

	return customers, nil
}

