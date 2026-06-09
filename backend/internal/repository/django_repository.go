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

// GetOfficers retrieves all users from Django database (officers, merchants, personal users, etc.)
// Note: We sync ALL users because loans can be assigned to any user_type (MERCHANT, PERSONAL, etc.)
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
			user_type,
			CASE
				WHEN performance_status = 'Active' THEN 'Active'
				ELSE 'Inactive'
			END as employment_status,
			date_joined::DATE as hire_date,
			created_at,
			updated_at
		FROM accounts_customuser
		WHERE is_active = TRUE
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
		var userType sql.NullString
		var hireDate sql.NullTime

		err := rows.Scan(
			&officer.OfficerID,
			&officer.OfficerName,
			&phone,
			&officer.Branch,
			&officer.Region,
			&userType,
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
		if userType.Valid {
			officer.UserType = &userType.String
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
			user_type,
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
	var userType sql.NullString
	var hireDate sql.NullTime

	err := r.db.QueryRowContext(ctx, query, officerID).Scan(
		&officer.OfficerID,
		&officer.OfficerName,
		&phone,
		&officer.Branch,
		&officer.Region,
		&userType,
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
	if userType.Valid {
		officer.UserType = &userType.String
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
		WHERE TRUE
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
// Returns basic loan data that will be used to create LoanInput
func (r *DjangoRepository) GetLoans(ctx context.Context, limit, offset int) ([]map[string]interface{}, error) {
	query := `
		SELECT
			l.id::VARCHAR(50) as loan_id,
			l.borrower_id::VARCHAR(50) as customer_id,
			COALESCE(TRIM(c.first_name || ' ' || c.last_name), c.phone_number) as customer_name,
			c.phone_number as customer_phone,
			l.agent_id::VARCHAR(50) as officer_id,
			COALESCE(u.username, u.email) as officer_name,
			u.user_phone as officer_phone,
			COALESCE(u.user_branch, 'Unknown') as branch,
			CASE
				WHEN u.user_branch LIKE '%Lagos%' THEN 'Lagos'
				WHEN u.user_branch LIKE '%Abuja%' THEN 'FCT'
				WHEN u.user_branch LIKE '%Ogun%' THEN 'Ogun'
				WHEN u.user_branch LIKE '%Oyo%' THEN 'Oyo'
				ELSE 'Nigeria'
			END as region,
			l.amount as loan_amount,
			l.repayment_amount as repayment_amount,
			l.interest_rate / 100.0 as interest_rate,
			l.processing_fee as fee_amount,
			l.tenor_in_days as loan_term_days,
			l.date_disbursed as disbursement_date,
			l.start_date as first_payment_due_date,
				l.end_date as maturity_date,
				CASE
				-- Completed/Closed loans
				WHEN l.status = 'COMPLETED' THEN 'Closed'
				WHEN l.status = 'CLOSED' THEN 'Closed'

				-- Active/Open loans (loans in progress)
				WHEN l.status = 'OPEN' THEN 'Active'
				WHEN l.status = 'OPEN_TO_SUPERVISOR' THEN 'Active'
				WHEN l.status = 'APPROVED' THEN 'Active'
				WHEN l.status = 'ACTIVE' THEN 'Active'

				-- Defaulted/Past Due loans
				WHEN l.status = 'PAST_MATURITY' THEN 'Defaulted'
				WHEN l.status = 'DEFAULTED' THEN 'Defaulted'

				-- Rejected/Cancelled loans (loans that were not disbursed or were declined)
				WHEN l.status = 'DECLINED_BY_SUPERVISOR' THEN 'Rejected'
				WHEN l.status = 'REJECTED' THEN 'Rejected'
				WHEN l.status = 'NOT_TAKEN' THEN 'Cancelled'

					-- Default fallback
					ELSE 'Active'
				END as status,
				l.status as django_status,
				l.performance_status,
			l.loan_type,
			l.verification_stage as verification_status,
			l.is_disbursed,
			l.supervisor_disbursement_status,
			l.created_at,
			l.updated_at
		FROM loans_ajoloan l
		LEFT JOIN accounts_customuser u ON l.agent_id = u.id
		LEFT JOIN ajo_ajouser c ON l.borrower_id = c.id
		WHERE l.is_disbursed = TRUE
		ORDER BY l.date_disbursed DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query loans from Django: %w", err)
	}
	defer rows.Close()

	return scanDjangoLoanRows(rows)
}

// GetLoansChangedSince retrieves disbursed loans created/updated/disbursed since a timestamp.
func (r *DjangoRepository) GetLoansChangedSince(ctx context.Context, since time.Time, limit, offset int) ([]map[string]interface{}, error) {
	query := `
		SELECT
			l.id::VARCHAR(50) as loan_id,
			l.borrower_id::VARCHAR(50) as customer_id,
			COALESCE(TRIM(c.first_name || ' ' || c.last_name), c.phone_number) as customer_name,
			c.phone_number as customer_phone,
			l.agent_id::VARCHAR(50) as officer_id,
			COALESCE(u.username, u.email) as officer_name,
			u.user_phone as officer_phone,
			COALESCE(u.user_branch, 'Unknown') as branch,
			CASE
				WHEN u.user_branch LIKE '%Lagos%' THEN 'Lagos'
				WHEN u.user_branch LIKE '%Abuja%' THEN 'FCT'
				WHEN u.user_branch LIKE '%Ogun%' THEN 'Ogun'
				WHEN u.user_branch LIKE '%Oyo%' THEN 'Oyo'
				ELSE 'Nigeria'
			END as region,
			l.amount as loan_amount,
			l.repayment_amount as repayment_amount,
			l.interest_rate / 100.0 as interest_rate,
			l.processing_fee as fee_amount,
			l.tenor_in_days as loan_term_days,
			l.date_disbursed as disbursement_date,
			l.start_date as first_payment_due_date,
			l.end_date as maturity_date,
			CASE
				WHEN l.status = 'COMPLETED' THEN 'Closed'
				WHEN l.status = 'CLOSED' THEN 'Closed'
				WHEN l.status = 'OPEN' THEN 'Active'
				WHEN l.status = 'OPEN_TO_SUPERVISOR' THEN 'Active'
				WHEN l.status = 'APPROVED' THEN 'Active'
				WHEN l.status = 'ACTIVE' THEN 'Active'
				WHEN l.status = 'PAST_MATURITY' THEN 'Defaulted'
				WHEN l.status = 'DEFAULTED' THEN 'Defaulted'
				WHEN l.status = 'DECLINED_BY_SUPERVISOR' THEN 'Rejected'
				WHEN l.status = 'REJECTED' THEN 'Rejected'
				WHEN l.status = 'NOT_TAKEN' THEN 'Cancelled'
				ELSE 'Active'
			END as status,
			l.status as django_status,
			l.performance_status,
			l.loan_type,
			l.verification_stage as verification_status,
			l.is_disbursed,
			l.supervisor_disbursement_status,
			l.created_at,
			l.updated_at
		FROM loans_ajoloan l
		LEFT JOIN accounts_customuser u ON l.agent_id = u.id
		LEFT JOIN ajo_ajouser c ON l.borrower_id = c.id
		WHERE l.is_disbursed = TRUE
		  AND (
			l.updated_at >= $1
			OR l.created_at >= $1
			OR l.date_disbursed::date >= $1::date
		  )
		ORDER BY l.updated_at ASC NULLS LAST, l.created_at ASC NULLS LAST, l.id ASC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, since, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query changed loans from Django: %w", err)
	}
	defer rows.Close()

	return scanDjangoLoanRows(rows)
}

func scanDjangoLoanRows(rows *sql.Rows) ([]map[string]interface{}, error) {
	var loans []map[string]interface{}
	for rows.Next() {
		var loanID, customerID, customerName, officerID, officerName, branch, region, status string
		var customerPhone, officerPhone, performanceStatus, loanType, verificationStatus sql.NullString
		var djangoStatus sql.NullString
		var loanAmount, repaymentAmount, interestRate, feeAmount float64
		var loanTermDays int
		var disbursementDate, firstPaymentDueDate, maturityDate sql.NullTime
		var isDisbursed bool
		var supervisorDisbursementStatus sql.NullString
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&loanID,
			&customerID,
			&customerName,
			&customerPhone,
			&officerID,
			&officerName,
			&officerPhone,
			&branch,
			&region,
			&loanAmount,
			&repaymentAmount,
			&interestRate,
			&feeAmount,
			&loanTermDays,
			&disbursementDate,
			&firstPaymentDueDate,
			&maturityDate,
			&status,
			&djangoStatus,
			&performanceStatus,
			&loanType,
			&verificationStatus,
			&isDisbursed,
			&supervisorDisbursementStatus,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan loan row: %w", err)
		}

		loan := map[string]interface{}{
			"loan_id":          loanID,
			"customer_id":      customerID,
			"customer_name":    customerName,
			"officer_id":       officerID,
			"officer_name":     officerName,
			"branch":           branch,
			"region":           region,
			"loan_amount":      loanAmount,
			"repayment_amount": repaymentAmount,
			"interest_rate":    interestRate,
			"fee_amount":       feeAmount,
			"loan_term_days":   loanTermDays,
			"status":           status,
			"channel":          "AJO", // Default channel
			"created_at":       createdAt,
			"updated_at":       updatedAt,
		}

		if djangoStatus.Valid {
			loan["django_status"] = djangoStatus.String
		}

		if customerPhone.Valid {
			loan["customer_phone"] = customerPhone.String
		}
		if officerPhone.Valid {
			loan["officer_phone"] = officerPhone.String
		}
		if performanceStatus.Valid {
			loan["performance_status"] = performanceStatus.String
		}
		if loanType.Valid {
			loan["loan_type"] = loanType.String
		}
		if verificationStatus.Valid {
			loan["verification_status"] = verificationStatus.String
		}
		loan["is_disbursed"] = isDisbursed
		if supervisorDisbursementStatus.Valid {
			loan["supervisor_disbursement_status"] = supervisorDisbursementStatus.String
		}
		if disbursementDate.Valid {
			loan["disbursement_date"] = disbursementDate.Time.Format("2006-01-02")
		}
		if firstPaymentDueDate.Valid {
			loan["first_payment_due_date"] = firstPaymentDueDate.Time.Format("2006-01-02")
		}
		if maturityDate.Valid {
			loan["maturity_date"] = maturityDate.Time.Format("2006-01-02")
		}

		loans = append(loans, loan)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating loan rows: %w", err)
	}

	return loans, nil
}

// GetRepayments retrieves repayments from Django database
func (r *DjangoRepository) GetRepayments(ctx context.Context, limit, offset int) ([]map[string]interface{}, error) {
	query := `
		SELECT
			r.id::VARCHAR(50) as repayment_id,
			r.ajo_loan_id::VARCHAR(50) as loan_id,
			r.paid_date as payment_date,
			r.repayment_amount as payment_amount,
			COALESCE(r.repayment_type, 'TRANSFER') as payment_method,
			r.created_at,
			r.updated_at
		FROM loans_ajoloanrepayment r
		WHERE r.paid_date IS NOT NULL
		ORDER BY r.paid_date DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query repayments: %w", err)
	}
	defer rows.Close()

	var repayments []map[string]interface{}
	for rows.Next() {
		repayment := make(map[string]interface{})
		var repaymentID, loanID, paymentMethod string
		var paymentDate, createdAt, updatedAt time.Time
		var paymentAmount float64

		if err := rows.Scan(
			&repaymentID,
			&loanID,
			&paymentDate,
			&paymentAmount,
			&paymentMethod,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan repayment: %w", err)
		}

		repayment["repayment_id"] = repaymentID
		repayment["loan_id"] = loanID
		repayment["payment_date"] = paymentDate.Format("2006-01-02")
		repayment["payment_amount"] = paymentAmount
		repayment["payment_method"] = paymentMethod
		repayment["created_at"] = createdAt
		repayment["updated_at"] = updatedAt

		repayments = append(repayments, repayment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating repayments: %w", err)
	}

	return repayments, nil
}

// GetRepaymentsAfterID retrieves repayments from Django database where ID > afterID
// This is used for incremental sync to fetch only new repayments
func (r *DjangoRepository) GetRepaymentsAfterID(ctx context.Context, afterID int64, limit int) ([]map[string]interface{}, error) {
	query := `
		SELECT
			r.id::VARCHAR(50) as repayment_id,
			r.id as repayment_id_int,
			r.ajo_loan_id::VARCHAR(50) as loan_id,
			r.paid_date as payment_date,
			r.repayment_amount as payment_amount,
			COALESCE(r.repayment_type, 'TRANSFER') as payment_method,
			r.created_at,
			r.updated_at
		FROM loans_ajoloanrepayment r
		WHERE r.paid_date IS NOT NULL
		  AND r.id > $1
		ORDER BY r.id ASC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query repayments after ID %d: %w", afterID, err)
	}
	defer rows.Close()

	var repayments []map[string]interface{}
	for rows.Next() {
		repayment := make(map[string]interface{})
		var repaymentID, loanID, paymentMethod string
		var repaymentIDInt int64
		var paymentDate, createdAt, updatedAt time.Time
		var paymentAmount float64

		if err := rows.Scan(
			&repaymentID,
			&repaymentIDInt,
			&loanID,
			&paymentDate,
			&paymentAmount,
			&paymentMethod,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan repayment: %w", err)
		}

		repayment["repayment_id"] = repaymentID
		repayment["repayment_id_int"] = repaymentIDInt
		repayment["loan_id"] = loanID
		repayment["payment_date"] = paymentDate.Format("2006-01-02")
		repayment["payment_amount"] = paymentAmount
		repayment["payment_method"] = paymentMethod
		repayment["created_at"] = createdAt
		repayment["updated_at"] = updatedAt

		repayments = append(repayments, repayment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating repayments: %w", err)
	}

	return repayments, nil
}

// GetRepaymentsByLoanID retrieves repayments for a specific loan from Django database
func (r *DjangoRepository) GetRepaymentsByLoanID(ctx context.Context, loanID string) ([]map[string]interface{}, error) {
	query := `
		SELECT
			r.id::VARCHAR(50) as repayment_id,
			r.ajo_loan_id::VARCHAR(50) as loan_id,
			r.paid_date as payment_date,
			r.repayment_amount as payment_amount,
			COALESCE(r.repayment_type, 'TRANSFER') as payment_method,
			r.created_at,
			r.updated_at
		FROM loans_ajoloanrepayment r
		WHERE r.ajo_loan_id = $1::BIGINT
			AND r.paid_date IS NOT NULL
		ORDER BY r.paid_date DESC
	`

	rows, err := r.db.QueryContext(ctx, query, loanID)
	if err != nil {
		return nil, fmt.Errorf("failed to query repayments for loan %s: %w", loanID, err)
	}
	defer rows.Close()

	var repayments []map[string]interface{}
	for rows.Next() {
		repayment := make(map[string]interface{})
		var repaymentID, loanIDStr, paymentMethod string
		var paymentDate, createdAt, updatedAt time.Time
		var paymentAmount float64

		if err := rows.Scan(
			&repaymentID,
			&loanIDStr,
			&paymentDate,
			&paymentAmount,
			&paymentMethod,
			&createdAt,
			&updatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan repayment: %w", err)
		}

		repayment["repayment_id"] = repaymentID
		repayment["loan_id"] = loanIDStr
		repayment["payment_date"] = paymentDate.Format("2006-01-02")
		repayment["payment_amount"] = paymentAmount
		repayment["payment_method"] = paymentMethod
		repayment["created_at"] = createdAt
		repayment["updated_at"] = updatedAt

		repayments = append(repayments, repayment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating repayments: %w", err)
	}

	return repayments, nil
}

// LoanCreditBureauKYCRow represents a raw row from the Django sync query
type LoanCreditBureauKYCRow struct {
	DjangoLoanID     int64
	LoanRef          string
	LoanAmount       sql.NullFloat64
	Tenor            sql.NullString
	TenorInDays      sql.NullInt32
	BorrowerFullName sql.NullString
	BorrowerPhone    sql.NullString
	LoanStatus       sql.NullString
	LoanType         sql.NullString
	DateDisbursed    sql.NullTime

	VerificationNumber sql.NullString
	VerificationType   sql.NullString
	NIN                sql.NullString
	DateOfBirth        sql.NullString
	IsVerified         sql.NullBool
	Address            sql.NullString

	FaceMatch   sql.NullBool
	IDCardImage sql.NullString
	SelfieImage sql.NullString

	GuarantorFullName     sql.NullString
	GuarantorPhone        sql.NullString
	GuarantorEmail        sql.NullString
	GuarantorAddress      sql.NullString
	GuarantorRelationship sql.NullString
	GuarantorIDCardImage  sql.NullString
	GuarantorSelfieImage  sql.NullString

	CBResult         []byte
	CBDecision       []byte
	CBDecisionStatus sql.NullString
	CBCredibility    []byte

	CBStatus                    sql.NullBool
	CBReason                    sql.NullString
	CBBadLoansInstitutions      []byte
	CBBadLoansInstitutionsCount sql.NullInt32
	CBCountOfOpenLoans          sql.NullInt32
	CBTotalOutstanding          sql.NullFloat64
	CBDebtThreshold             sql.NullFloat64
	CBHighOutstandingDebt       sql.NullBool
	CBOpenLoanInstitutions      []byte
	CBMaxDebtInstitutionCount   sql.NullInt32

	CBLegacyResponse         sql.NullString
	CBNoOfDefaultedLoans     sql.NullInt32
	CBMonthlyRepaymentAmount sql.NullFloat64

	CBDataSource string
	CBCreatedAt  sql.NullTime
}

// GetLoanCreditBureauKYCForSyncBatch retrieves a batch of loan + KYC + credit bureau data from Django
// using a priority chain: credit_bureau_result > borrower_worthiness > credit_bureau_metadata.
// Uses LIMIT/OFFSET to avoid loading all rows into memory at once.
func (r *DjangoRepository) GetLoanCreditBureauKYCForSyncBatch(ctx context.Context, limit, offset int) ([]*LoanCreditBureauKYCRow, error) {
	query := `
		SELECT
			l.id AS django_loan_id,
			l.loan_ref,
			l.amount AS loan_amount,
			l.tenor,
			l.tenor_in_days,
			l.borrower_full_name,
			l.borrower_phone_number,
			l.status AS loan_status,
			l.loan_type,
			l.date_disbursed,
			bi.verification_number,
			bi.verification_type,
			bi.nin,
			bi.date_of_birth,
			bi.is_verified,
			u.address,
			bi.face_match,
			bi.base_64_img_string AS id_card_image,
			bi.snapped_image AS selfie_image,
			COALESCE(g.surname || ' ' || g.last_name, g.surname, g.last_name) AS guarantor_full_name,
			g.phone_number AS guarantor_phone,
			g.email AS guarantor_email,
			g.address AS guarantor_address,
			g.relationship_to_borrower AS guarantor_relationship,
			g.base_64_img_string AS guarantor_id_card_image,
			g.snapped_image AS guarantor_selfie_image,
			cbr.result AS cb_result,
			cbr.decision AS cb_decision,
			cbr.decision_status AS cb_decision_status,
			cbr.credibility AS cb_credibility,
			COALESCE(cbr.status, w.credit_worthy) AS cb_status,
			COALESCE(cbr.reason, w.reason) AS cb_reason,
			COALESCE(cbr.bad_loans_institutions, array_to_json(w.bad_loans_institions)::jsonb) AS cb_bad_loans_institutions,
			COALESCE(cbr.bad_loans_institutions_count, w.bad_loans_institions_count) AS cb_bad_loans_institutions_count,
			COALESCE(cbr.count_of_open_loans, w.count_of_open_loans) AS cb_count_of_open_loans,
			COALESCE(cbr.total_outstanding, w.total_outstanding) AS cb_total_outstanding,
			COALESCE(cbr.debt_threshold, w.debt_threshold) AS cb_debt_threshold,
			COALESCE(cbr.high_outstanding_debt, w.high_outstanding_debt) AS cb_high_outstanding_debt,
			COALESCE(cbr.open_loan_institutions, array_to_json(w.open_loan_institutions)::jsonb) AS cb_open_loan_institutions,
			COALESCE(cbr.max_debt_institution_count, w.max_debt_institution_count) AS cb_max_debt_institution_count,
			m.response AS cb_legacy_response,
			m.no_of_defaulted_loans AS cb_no_of_defaulted_loans,
			m.monthly_repayment_amount AS cb_monthly_repayment_amount,
			CASE
				WHEN cbr.id IS NOT NULL THEN 'credit_bureau_result'
				WHEN w.id IS NOT NULL THEN 'borrower_worthiness'
				WHEN m.id IS NOT NULL THEN 'credit_bureau_metadata'
				ELSE 'none'
			END AS cb_data_source,
			COALESCE(cbr.created_at, w.created_at, m.created_at) AS cb_created_at
		FROM loans_ajoloan l
		LEFT JOIN loans_borrowerinfo bi ON bi.loan_id = l.id
		LEFT JOIN accounts_customuser u ON u.id = l.borrower_id
		LEFT JOIN loans_loanguarantor g ON g.borrower_info_id = bi.id
		LEFT JOIN credit_bureau_creditbureauresult cbr ON cbr.loan_reference = l.loan_ref
		LEFT JOIN LATERAL (
			SELECT ww.*
			FROM loans_borrowercreditbureauworthiness ww
			WHERE ww.borrower_id = l.borrower_id
			ORDER BY ww.created_at DESC LIMIT 1
		) w ON true
		LEFT JOIN LATERAL (
			SELECT mm.*
			FROM loans_creditbureaumetadata mm
			WHERE mm.ajo_user_id = l.borrower_id AND mm.status = 'SUCCESS'
			ORDER BY mm.created_at DESC LIMIT 1
		) m ON true
		WHERE (cbr.id IS NOT NULL OR w.id IS NOT NULL OR m.id IS NOT NULL)
			AND l.date_disbursed IS NOT NULL
			AND COALESCE(l.loan_type, '') NOT IN ('BNPL', 'RNPL', 'MERCHANT_OVERDRAFT')
			AND EXISTS (
				SELECT 1 FROM payment_transaction pt
				WHERE pt.quotation_id = l.loan_ref::text
				AND pt.transfer_provider = 'VFD_PROVIDER'
			)
		ORDER BY l.id
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query loan credit bureau KYC batch (offset=%d): %w", offset, err)
	}
	defer rows.Close()

	var results []*LoanCreditBureauKYCRow
	for rows.Next() {
		row := &LoanCreditBureauKYCRow{}
		err := rows.Scan(
			&row.DjangoLoanID, &row.LoanRef, &row.LoanAmount,
			&row.Tenor, &row.TenorInDays, &row.BorrowerFullName,
			&row.BorrowerPhone, &row.LoanStatus, &row.LoanType, &row.DateDisbursed,
			&row.VerificationNumber, &row.VerificationType, &row.NIN,
			&row.DateOfBirth, &row.IsVerified, &row.Address,
			&row.FaceMatch, &row.IDCardImage, &row.SelfieImage,
			&row.GuarantorFullName, &row.GuarantorPhone, &row.GuarantorEmail,
			&row.GuarantorAddress, &row.GuarantorRelationship,
			&row.GuarantorIDCardImage, &row.GuarantorSelfieImage,
			&row.CBResult, &row.CBDecision, &row.CBDecisionStatus, &row.CBCredibility,
			&row.CBStatus, &row.CBReason,
			&row.CBBadLoansInstitutions, &row.CBBadLoansInstitutionsCount,
			&row.CBCountOfOpenLoans, &row.CBTotalOutstanding,
			&row.CBDebtThreshold, &row.CBHighOutstandingDebt,
			&row.CBOpenLoanInstitutions, &row.CBMaxDebtInstitutionCount,
			&row.CBLegacyResponse, &row.CBNoOfDefaultedLoans, &row.CBMonthlyRepaymentAmount,
			&row.CBDataSource, &row.CBCreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan loan credit bureau KYC row: %w", err)
		}
		results = append(results, row)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating loan credit bureau KYC rows: %w", err)
	}

	return results, nil
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
