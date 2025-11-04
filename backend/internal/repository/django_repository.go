package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/seeds-metrics/analytics-backend/internal/models"
)

// DjangoRepository handles read-only queries to the Django database
type DjangoRepository struct {
	db *sql.DB
}

// NewDjangoRepository creates a new Django repository instance
func NewDjangoRepository(db *sql.DB) *DjangoRepository {
	return &DjangoRepository{db: db}
}

// GetOfficers retrieves all active officers from Django database
func (r *DjangoRepository) GetOfficers(ctx context.Context) ([]*models.Officer, error) {
	query := `
		SELECT
			id::VARCHAR(50) as officer_id,
			COALESCE(username, email) as officer_name,
			user_phone as officer_phone,
			user_branch as branch,
			CASE
				WHEN user_branch LIKE '%Lagos%' THEN 'Lagos'
				WHEN user_branch LIKE '%Abuja%' THEN 'FCT'
				WHEN user_branch LIKE '%Ogun%' THEN 'Ogun'
				WHEN user_branch LIKE '%Oyo%' THEN 'Oyo'
				ELSE 'Nigeria'
			END as region,
			CASE
				WHEN performance_status = 'Active' THEN 'Active'
				ELSE 'Inactive'
			END as employment_status,
			date_joined::DATE as hire_date,
			created_at,
			updated_at
		FROM accounts_customuser
		WHERE user_type IN ('AGENT', 'STAFF_AGENT', 'PROSPER_AGENT', 'DMO_AGENT', 'AJO_AGENT', 'RECOVERY_AGENT')
		AND is_active = TRUE
		ORDER BY officer_name
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query officers from Django: %w", err)
	}
	defer rows.Close()

	var officers []*models.Officer
	for rows.Next() {
		var officer models.Officer
		var phone sql.NullString
		var hireDate sql.NullTime

		err := rows.Scan(
			&officer.OfficerID,
			&officer.OfficerName,
			&phone,
			&officer.Branch,
			&officer.Region,
			&officer.EmploymentStatus,
			&hireDate,
			&officer.CreatedAt,
			&officer.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan officer row: %w", err)
		}

		if phone.Valid {
			officer.OfficerPhone = &phone.String
		}
		if hireDate.Valid {
			officer.HireDate = &hireDate.Time
		}

		officers = append(officers, &officer)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating officer rows: %w", err)
	}

	return officers, nil
}

// GetOfficerByID retrieves a single officer by ID from Django database
func (r *DjangoRepository) GetOfficerByID(ctx context.Context, officerID string) (*models.Officer, error) {
	query := `
		SELECT
			id::VARCHAR(50) as officer_id,
			COALESCE(username, email) as officer_name,
			user_phone as officer_phone,
			user_branch as branch,
			CASE
				WHEN user_branch LIKE '%Lagos%' THEN 'Lagos'
				WHEN user_branch LIKE '%Abuja%' THEN 'FCT'
				WHEN user_branch LIKE '%Ogun%' THEN 'Ogun'
				WHEN user_branch LIKE '%Oyo%' THEN 'Oyo'
				ELSE 'Nigeria'
			END as region,
			CASE
				WHEN performance_status = 'Active' THEN 'Active'
				ELSE 'Inactive'
			END as employment_status,
			date_joined::DATE as hire_date,
			created_at,
			updated_at
		FROM accounts_customuser
		WHERE id::VARCHAR(50) = $1
		AND user_type IN ('AGENT', 'STAFF_AGENT', 'PROSPER_AGENT', 'DMO_AGENT', 'AJO_AGENT', 'RECOVERY_AGENT')
		AND is_active = TRUE
	`

	var officer models.Officer
	var phone sql.NullString
	var hireDate sql.NullTime

	err := r.db.QueryRowContext(ctx, query, officerID).Scan(
		&officer.OfficerID,
		&officer.OfficerName,
		&phone,
		&officer.Branch,
		&officer.Region,
		&officer.EmploymentStatus,
		&hireDate,
		&officer.CreatedAt,
		&officer.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("officer not found: %s", officerID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query officer from Django: %w", err)
	}

	if phone.Valid {
		officer.OfficerPhone = &phone.String
	}
	if hireDate.Valid {
		officer.HireDate = &hireDate.Time
	}

	return &officer, nil
}

// GetCustomers retrieves all customers from Django database
func (r *DjangoRepository) GetCustomers(ctx context.Context, limit, offset int) ([]*models.Customer, error) {
	query := `
		SELECT
			id::VARCHAR(50) as customer_id,
			COALESCE(TRIM(first_name || ' ' || last_name), phone_number) as customer_name,
			phone_number as customer_phone,
			email,
			dob as date_of_birth,
			gender,
			state,
			lga,
			address,
			CASE
				WHEN bvn_verified = TRUE AND onboarding_verified = TRUE THEN 'Verified'
				WHEN bvn_verified = TRUE THEN 'Partial'
				ELSE 'Pending'
			END as kyc_status,
			created_at,
			updated_at
		FROM ajo_ajouser
		WHERE onboarding_complete = TRUE
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query customers from Django: %w", err)
	}
	defer rows.Close()

	var customers []*models.Customer
	for rows.Next() {
		var customer models.Customer
		var phone, email, dob, gender, state, lga, address, kycStatus sql.NullString

		err := rows.Scan(
			&customer.CustomerID,
			&customer.CustomerName,
			&phone,
			&email,
			&dob,
			&gender,
			&state,
			&lga,
			&address,
			&kycStatus,
			&customer.CreatedAt,
			&customer.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan customer row: %w", err)
		}

		if phone.Valid {
			customer.CustomerPhone = &phone.String
		}
		if email.Valid {
			customer.CustomerEmail = &email.String
		}
		if dob.Valid {
			dobTime, _ := time.Parse("2006-01-02", dob.String)
			customer.DateOfBirth = &dobTime
		}
		if gender.Valid {
			customer.Gender = &gender.String
		}
		if state.Valid {
			customer.State = &state.String
		}
		if lga.Valid {
			customer.LGA = &lga.String
		}
		if address.Valid {
			customer.Address = &address.String
		}
		if kycStatus.Valid {
			customer.KYCStatus = &kycStatus.String
		}

		customers = append(customers, &customer)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating customer rows: %w", err)
	}

	return customers, nil
}

// GetCustomerByID retrieves a single customer by ID from Django database
func (r *DjangoRepository) GetCustomerByID(ctx context.Context, customerID string) (*models.Customer, error) {
	query := `
		SELECT
			id::VARCHAR(50) as customer_id,
			COALESCE(TRIM(first_name || ' ' || last_name), phone_number) as customer_name,
			phone_number as customer_phone,
			email,
			dob as date_of_birth,
			gender,
			state,
			lga,
			address,
			CASE
				WHEN bvn_verified = TRUE AND onboarding_verified = TRUE THEN 'Verified'
				WHEN bvn_verified = TRUE THEN 'Partial'
				ELSE 'Pending'
			END as kyc_status,
			created_at,
			updated_at
		FROM ajo_ajouser
		WHERE id::VARCHAR(50) = $1
		AND onboarding_complete = TRUE
	`

	var customer models.Customer
	var phone, email, dob, gender, state, lga, address, kycStatus sql.NullString

	err := r.db.QueryRowContext(ctx, query, customerID).Scan(
		&customer.CustomerID,
		&customer.CustomerName,
		&phone,
		&email,
		&dob,
		&gender,
		&state,
		&lga,
		&address,
		&kycStatus,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("customer not found: %s", customerID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query customer from Django: %w", err)
	}

	if phone.Valid {
		customer.CustomerPhone = &phone.String
	}
	if email.Valid {
		customer.CustomerEmail = &email.String
	}
	if dob.Valid {
		dobTime, _ := time.Parse("2006-01-02", dob.String)
		customer.DateOfBirth = &dobTime
	}
	if gender.Valid {
		customer.Gender = &gender.String
	}
	if state.Valid {
		customer.State = &state.String
	}
	if lga.Valid {
		customer.LGA = &lga.String
	}
	if address.Valid {
		customer.Address = &address.String
	}
	if kycStatus.Valid {
		customer.KYCStatus = &kycStatus.String
	}

	return &customer, nil
}

// GetLoansCount returns the total count of loans in Django database
func (r *DjangoRepository) GetLoansCount(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM loans_ajoloan WHERE is_disbursed = TRUE`

	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count loans from Django: %w", err)
	}

	return count, nil
}

// HealthCheck verifies the Django database connection is healthy
func (r *DjangoRepository) HealthCheck(ctx context.Context) error {
	query := `SELECT 1`
	var result int

	err := r.db.QueryRowContext(ctx, query).Scan(&result)
	if err != nil {
		return fmt.Errorf("Django database health check failed: %w", err)
	}

	return nil
}
