package models

import (
	"encoding/json"
	"time"
)

// CreditBureauResult represents a processed credit bureau evaluation result
type CreditBureauResult struct {
	ID                        int64           `json:"id" db:"id"`
	LoanReference             *string         `json:"loan_reference" db:"loan_reference"`
	BorrowerFullName          string          `json:"borrower_full_name" db:"borrower_full_name"`
	PhoneNumber               string          `json:"phone_number" db:"phone_number"`
	Result                    json.RawMessage `json:"result" db:"result"`
	BadLoansInstitutions      json.RawMessage `json:"bad_loans_institutions" db:"bad_loans_institutions"`
	BadLoansInstitutionsCount int             `json:"bad_loans_institutions_count" db:"bad_loans_institutions_count"`
	CountOfOpenLoans          int             `json:"count_of_open_loans" db:"count_of_open_loans"`
	DebtThreshold             float64         `json:"debt_threshold" db:"debt_threshold"`
	HighOutstandingDebt       bool            `json:"high_outstanding_debt" db:"high_outstanding_debt"`
	MaxDebtInstitutionCount   int             `json:"max_debt_institution_count" db:"max_debt_institution_count"`
	OpenLoanInstitutions      json.RawMessage `json:"open_loan_institutions" db:"open_loan_institutions"`
	Reason                    *string         `json:"reason" db:"reason"`
	Status                    bool            `json:"status" db:"status"`
	TotalOutstanding          float64         `json:"total_outstanding" db:"total_outstanding"`
	Credibility               json.RawMessage `json:"credibility" db:"credibility"`
	Decision                  json.RawMessage `json:"decision" db:"decision"`
	DecisionStatus            *string         `json:"decision_status" db:"decision_status"`
	CreatedAt                 time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt                 time.Time       `json:"updated_at" db:"updated_at"`
}

// CreditBureauRawDump represents a raw API response from a credit bureau
type CreditBureauRawDump struct {
	ID               int64           `json:"id" db:"id"`
	LoanReference    *string         `json:"loan_reference" db:"loan_reference"`
	BorrowerFullName *string         `json:"borrower_full_name" db:"borrower_full_name"`
	PhoneNumber      *string         `json:"phone_number" db:"phone_number"`
	BVN              *string         `json:"bvn" db:"bvn"`
	Bureau           *string         `json:"bureau" db:"bureau"`
	StageTrace       json.RawMessage `json:"stage_trace" db:"stage_trace"`
	FullResponse     json.RawMessage `json:"full_response" db:"full_response"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at" db:"updated_at"`
}

// CreditBureauResultWithDumps is the API response combining a result with its raw dumps
type CreditBureauResultWithDumps struct {
	CreditBureauResult
	RawDumps []CreditBureauRawDump `json:"raw_dumps,omitempty"`
}

// LoanCreditBureauKYC represents a denormalized loan + KYC + credit bureau record
type LoanCreditBureauKYC struct {
	ID int64 `json:"id" db:"id"`

	// Loan Info
	DjangoLoanID     int64      `json:"django_loan_id" db:"django_loan_id"`
	LoanRef          string     `json:"loan_ref" db:"loan_ref"`
	LoanAmount       *float64   `json:"loan_amount" db:"loan_amount"`
	Tenor            *string    `json:"tenor" db:"tenor"`
	TenorInDays      *int       `json:"tenor_in_days" db:"tenor_in_days"`
	BorrowerFullName *string    `json:"borrower_full_name" db:"borrower_full_name"`
	BorrowerPhone    *string    `json:"borrower_phone_number" db:"borrower_phone_number"`
	LoanStatus       *string    `json:"loan_status" db:"loan_status"`
	LoanType         *string    `json:"loan_type" db:"loan_type"`
	DateDisbursed    *time.Time `json:"date_disbursed" db:"date_disbursed"`

	// Borrower KYC
	VerificationNumber *string `json:"verification_number" db:"verification_number"`
	VerificationType   *string `json:"verification_type" db:"verification_type"`
	NIN                *string `json:"nin" db:"nin"`
	DateOfBirth        *string `json:"date_of_birth" db:"date_of_birth"`
	IsVerified         *bool   `json:"is_verified" db:"is_verified"`
	Address            *string `json:"address" db:"address"`

	// Face Match & Images
	FaceMatch   *bool   `json:"face_match" db:"face_match"`
	IDCardImage *string `json:"id_card_image,omitempty" db:"id_card_image"`
	SelfieImage *string `json:"selfie_image,omitempty" db:"selfie_image"`

	// Guarantor Info
	GuarantorFullName     *string `json:"guarantor_full_name" db:"guarantor_full_name"`
	GuarantorPhone        *string `json:"guarantor_phone" db:"guarantor_phone"`
	GuarantorEmail        *string `json:"guarantor_email" db:"guarantor_email"`
	GuarantorAddress      *string `json:"guarantor_address" db:"guarantor_address"`
	GuarantorRelationship *string `json:"guarantor_relationship" db:"guarantor_relationship"`
	GuarantorIDCardImage  *string `json:"guarantor_id_card_image,omitempty" db:"guarantor_id_card_image"`
	GuarantorSelfieImage  *string `json:"guarantor_selfie_image,omitempty" db:"guarantor_selfie_image"`

	// Credit Bureau Fields
	CBResult                    json.RawMessage `json:"cb_result" db:"cb_result"`
	CBStatus                    *bool           `json:"cb_status" db:"cb_status"`
	CBReason                    *string         `json:"cb_reason" db:"cb_reason"`
	CBDecision                  json.RawMessage `json:"cb_decision" db:"cb_decision"`
	CBDecisionStatus            *string         `json:"cb_decision_status" db:"cb_decision_status"`
	CBCredibility               json.RawMessage `json:"cb_credibility" db:"cb_credibility"`
	CBBadLoansInstitutions      json.RawMessage `json:"cb_bad_loans_institutions" db:"cb_bad_loans_institutions"`
	CBBadLoansInstitutionsCount *int            `json:"cb_bad_loans_institutions_count" db:"cb_bad_loans_institutions_count"`
	CBCountOfOpenLoans          *int            `json:"cb_count_of_open_loans" db:"cb_count_of_open_loans"`
	CBTotalOutstanding          *float64        `json:"cb_total_outstanding" db:"cb_total_outstanding"`
	CBDebtThreshold             *float64        `json:"cb_debt_threshold" db:"cb_debt_threshold"`
	CBHighOutstandingDebt       *bool           `json:"cb_high_outstanding_debt" db:"cb_high_outstanding_debt"`
	CBOpenLoanInstitutions      json.RawMessage `json:"cb_open_loan_institutions" db:"cb_open_loan_institutions"`
	CBMaxDebtInstitutionCount   *int            `json:"cb_max_debt_institution_count" db:"cb_max_debt_institution_count"`

	// Legacy metadata
	CBLegacyResponse         json.RawMessage `json:"cb_legacy_response" db:"cb_legacy_response"`
	CBNoOfDefaultedLoans     *int            `json:"cb_no_of_defaulted_loans" db:"cb_no_of_defaulted_loans"`
	CBMonthlyRepaymentAmount *float64        `json:"cb_monthly_repayment_amount" db:"cb_monthly_repayment_amount"`

	// Data source
	CBDataSource string `json:"cb_data_source" db:"cb_data_source"`

	// Timestamps
	CBCreatedAt *time.Time `json:"cb_created_at" db:"cb_created_at"`
	SyncedAt    time.Time  `json:"synced_at" db:"synced_at"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
}
