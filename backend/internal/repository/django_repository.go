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
			COALESCE(user_branch, 'Unknown') as branch,
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
		var phone, dob, gender, state, lga, address, kycStatus sql.NullString

		err := rows.Scan(
			&customer.CustomerID,
			&customer.CustomerName,
			&phone,
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
	var phone, dob, gender, state, lga, address, kycStatus sql.NullString

	err := r.db.QueryRowContext(ctx, query, customerID).Scan(
		&customer.CustomerID,
		&customer.CustomerName,
		&phone,
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

// GetLoans retrieves loans from Django database with pagination
func (r *DjangoRepository) GetLoans(ctx context.Context, limit, offset int) ([]*models.Loan, error) {
	query := `
		SELECT
			l.id::VARCHAR(50) as loan_id,
			l.borrower_id::VARCHAR(50) as customer_id,
			l.agent_id::VARCHAR(50) as officer_id,
			COALESCE(u.user_branch, 'Unknown') as branch,
			CASE
				WHEN u.user_branch LIKE '%Lagos%' THEN 'Lagos'
				WHEN u.user_branch LIKE '%Abuja%' THEN 'FCT'
				WHEN u.user_branch LIKE '%Ogun%' THEN 'Ogun'
				WHEN u.user_branch LIKE '%Oyo%' THEN 'Oyo'
				ELSE 'Nigeria'
			END as region,
			l.amount as principal_amount,
			l.interest_rate / 100.0 as interest_rate,
			l.interest_amount,
			l.processing_fee,
			l.tenor_in_days as loan_term_days,
			l.date_disbursed as disbursement_date,
			l.start_date,
			l.end_date as maturity_date,
			CASE
				WHEN l.status = 'COMPLETED' THEN 'Closed'
				WHEN l.status = 'ACTIVE' THEN 'Active'
				WHEN l.status = 'DEFAULTED' THEN 'Defaulted'
				ELSE 'Active'
			END as loan_status,
			l.created_at,
			l.updated_at
		FROM loans_ajoloan l
		LEFT JOIN accounts_customuser u ON l.agent_id = u.id
		WHERE l.is_disbursed = TRUE
		ORDER BY l.date_disbursed DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query loans from Django: %w", err)
	}
	defer rows.Close()

	var loans []*models.Loan
	for rows.Next() {
		var loan models.Loan
		var disbursementDate, startDate, maturityDate sql.NullTime

		err := rows.Scan(
			&loan.LoanID,
			&loan.CustomerID,
			&loan.OfficerID,
			&loan.Branch,
			&loan.Region,
			&loan.PrincipalAmount,
			&loan.InterestRate,
			&loan.InterestAmount,
			&loan.ProcessingFee,
			&loan.LoanTermDays,
			&disbursementDate,
			&startDate,
			&maturityDate,
			&loan.LoanStatus,
			&loan.CreatedAt,
			&loan.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan loan row: %w", err)
		}

		if disbursementDate.Valid {
			loan.DisbursementDate = &disbursementDate.Time
		}
		if startDate.Valid {
			loan.StartDate = &startDate.Time
		}
		if maturityDate.Valid {
			loan.MaturityDate = &maturityDate.Time
		}

		loans = append(loans, &loan)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating loan rows: %w", err)
	}

	return loans, nil
}

// GetRepayments retrieves repayments for a specific loan from Django database
func (r *DjangoRepository) GetRepaymentsByLoanID(ctx context.Context, loanID string) ([]*models.Repayment, error) {
	query := `
		SELECT
			id::VARCHAR(50) as repayment_id,
			ajo_loan_id::VARCHAR(50) as loan_id,
			repayment_amount as amount_paid,
			paid_date as payment_date,
			created_at,
			updated_at
		FROM loans_ajoloanrepayment
		WHERE ajo_loan_id::VARCHAR(50) = $1
		AND applied_to_loan = TRUE
		ORDER BY paid_date ASC
	`

	rows, err := r.db.QueryContext(ctx, query, loanID)
	if err != nil {
		return nil, fmt.Errorf("failed to query repayments from Django: %w", err)
	}
	defer rows.Close()

	var repayments []*models.Repayment
	for rows.Next() {
		var repayment models.Repayment
		var paymentDate sql.NullTime

		err := rows.Scan(
			&repayment.RepaymentID,
			&repayment.LoanID,
			&repayment.AmountPaid,
			&paymentDate,
			&repayment.CreatedAt,
			&repayment.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan repayment row: %w", err)
		}

		if paymentDate.Valid {
			repayment.PaymentDate = &paymentDate.Time
		}

		repayments = append(repayments, &repayment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating repayment rows: %w", err)
	}

	return repayments, nil
}

// GetRepaymentsCount returns the total count of repayments in Django database
func (r *DjangoRepository) GetRepaymentsCount(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM loans_ajoloanrepayment WHERE applied_to_loan = TRUE`

	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count repayments from Django: %w", err)
	}

	return count, nil
}

// GetSchedulesByLoanID retrieves loan schedules for a specific loan from Django database
func (r *DjangoRepository) GetSchedulesByLoanID(ctx context.Context, loanID string) ([]*models.LoanSchedule, error) {
	query := `
		SELECT
			id::VARCHAR(50) as schedule_id,
			loan_id::VARCHAR(50),
			due_amount,
			amount_left,
			paid_amount,
			due_date,
			paid_date,
			fully_paid,
			fully_paid_date,
			is_late,
			late_days,
			created_at,
			updated_at
		FROM loans_ajoloanschedule
		WHERE loan_id::VARCHAR(50) = $1
		ORDER BY due_date ASC
	`

	rows, err := r.db.QueryContext(ctx, query, loanID)
	if err != nil {
		return nil, fmt.Errorf("failed to query schedules from Django: %w", err)
	}
	defer rows.Close()

	var schedules []*models.LoanSchedule
	for rows.Next() {
		var schedule models.LoanSchedule
		var dueDate, paidDate, fullyPaidDate sql.NullTime

		err := rows.Scan(
			&schedule.ScheduleID,
			&schedule.LoanID,
			&schedule.DueAmount,
			&schedule.AmountLeft,
			&schedule.PaidAmount,
			&dueDate,
			&paidDate,
			&schedule.FullyPaid,
			&fullyPaidDate,
			&schedule.IsLate,
			&schedule.LateDays,
			&schedule.CreatedAt,
			&schedule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule row: %w", err)
		}

		if dueDate.Valid {
			schedule.DueDate = &dueDate.Time
		}
		if paidDate.Valid {
			schedule.PaidDate = &paidDate.Time
		}
		if fullyPaidDate.Valid {
			schedule.FullyPaidDate = &fullyPaidDate.Time
		}

		schedules = append(schedules, &schedule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating schedule rows: %w", err)
	}

	return schedules, nil
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
