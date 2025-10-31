package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Loan represents a loan record in the database
type Loan struct {
	// Primary Key
	LoanID string `json:"loan_id" db:"loan_id"`

	// ========================================
	// FROM ETL SOURCE (Main Backend)
	// ========================================
	CustomerID       string           `json:"customer_id" db:"customer_id"`
	CustomerName     string           `json:"customer_name" db:"customer_name"`
	CustomerPhone    *string          `json:"customer_phone,omitempty" db:"customer_phone"`
	OfficerID        string           `json:"officer_id" db:"officer_id"`
	OfficerName      string           `json:"officer_name" db:"officer_name"`
	OfficerPhone     *string          `json:"officer_phone,omitempty" db:"officer_phone"`
	Region           string           `json:"region" db:"region"`
	Branch           string           `json:"branch" db:"branch"`
	State            *string          `json:"state,omitempty" db:"state"`
	LoanAmount       decimal.Decimal  `json:"loan_amount" db:"loan_amount"`
	DisbursementDate time.Time        `json:"disbursement_date" db:"disbursement_date"`
	MaturityDate     time.Time        `json:"maturity_date" db:"maturity_date"`
	LoanTermDays     int              `json:"loan_term_days" db:"loan_term_days"`
	InterestRate     *decimal.Decimal `json:"interest_rate,omitempty" db:"interest_rate"`
	FeeAmount        *decimal.Decimal `json:"fee_amount,omitempty" db:"fee_amount"`
	Channel          string           `json:"channel" db:"channel"`
	ChannelPartner   *string          `json:"channel_partner,omitempty" db:"channel_partner"`
	Status           string           `json:"status" db:"status"`
	ClosedDate       *time.Time       `json:"closed_date,omitempty" db:"closed_date"`
	TimelinessScore  *decimal.Decimal `json:"timeliness_score,omitempty" db:"timeliness_score"`
	RepaymentHealth  *decimal.Decimal `json:"repayment_health,omitempty" db:"repayment_health"`

	// ========================================
	// [COMPUTED] FROM REPAYMENTS
	// ========================================
	CurrentDPD               int             `json:"current_dpd" db:"current_dpd"`
	MaxDPDEver               int             `json:"max_dpd_ever" db:"max_dpd_ever"`
	FirstPaymentMissed       *bool           `json:"first_payment_missed,omitempty" db:"first_payment_missed"`
	FirstPaymentDueDate      *time.Time      `json:"first_payment_due_date,omitempty" db:"first_payment_due_date"`
	FirstPaymentReceivedDate *time.Time      `json:"first_payment_received_date,omitempty" db:"first_payment_received_date"`
	PrincipalOutstanding     decimal.Decimal `json:"principal_outstanding" db:"principal_outstanding"`
	InterestOutstanding      decimal.Decimal `json:"interest_outstanding" db:"interest_outstanding"`
	FeesOutstanding          decimal.Decimal `json:"fees_outstanding" db:"fees_outstanding"`
	TotalOutstanding         decimal.Decimal `json:"total_outstanding" db:"total_outstanding"`
	ActualOutstanding        decimal.Decimal `json:"actual_outstanding" db:"actual_outstanding"`
	TotalPrincipalPaid       decimal.Decimal `json:"total_principal_paid" db:"total_principal_paid"`
	TotalInterestPaid        decimal.Decimal `json:"total_interest_paid" db:"total_interest_paid"`
	TotalFeesPaid            decimal.Decimal `json:"total_fees_paid" db:"total_fees_paid"`
	TotalRepayments          decimal.Decimal `json:"total_repayments" db:"total_repayments"`
	FIMRTagged               *bool           `json:"fimr_tagged,omitempty" db:"fimr_tagged"`
	EarlyIndicatorTagged     *bool           `json:"early_indicator_tagged,omitempty" db:"early_indicator_tagged"`
	DaysSinceLastRepayment   *int            `json:"days_since_last_repayment,omitempty" db:"days_since_last_repayment"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// LoanInput represents the input payload for creating/updating a loan (ETL only)
type LoanInput struct {
	LoanID           string           `json:"loan_id" binding:"required"`
	CustomerID       string           `json:"customer_id" binding:"required"`
	CustomerName     string           `json:"customer_name" binding:"required"`
	CustomerPhone    *string          `json:"customer_phone"`
	OfficerID        string           `json:"officer_id" binding:"required"`
	OfficerName      string           `json:"officer_name" binding:"required"`
	OfficerPhone     *string          `json:"officer_phone"`
	Region           string           `json:"region" binding:"required"`
	Branch           string           `json:"branch" binding:"required"`
	State            *string          `json:"state"`
	LoanAmount       decimal.Decimal  `json:"loan_amount" binding:"required"`
	DisbursementDate string           `json:"disbursement_date" binding:"required"` // YYYY-MM-DD
	MaturityDate     string           `json:"maturity_date" binding:"required"`     // YYYY-MM-DD
	LoanTermDays     int              `json:"loan_term_days" binding:"required"`
	InterestRate     *decimal.Decimal `json:"interest_rate"`
	FeeAmount        *decimal.Decimal `json:"fee_amount"`
	Channel          string           `json:"channel" binding:"required"`
	ChannelPartner   *string          `json:"channel_partner"`
	Status           string           `json:"status" binding:"required"`
	ClosedDate       *string          `json:"closed_date"` // YYYY-MM-DD
}

// LoanFilter represents filter criteria for querying loans
type LoanFilter struct {
	OfficerID       *string    `json:"officer_id"`
	Region          *string    `json:"region"`
	Branch          *string    `json:"branch"`
	Status          *string    `json:"status"`
	Channel         *string    `json:"channel"`
	FIMRTagged      *bool      `json:"fimr_tagged"`
	EarlyIndicator  *bool      `json:"early_indicator_tagged"`
	MinDPD          *int       `json:"min_dpd"`
	MaxDPD          *int       `json:"max_dpd"`
	DisbursedAfter  *time.Time `json:"disbursed_after"`
	DisbursedBefore *time.Time `json:"disbursed_before"`
	Limit           int        `json:"limit"`
	Offset          int        `json:"offset"`
}

// LoanDrilldown represents a loan record for drilldown views
type LoanDrilldown struct {
	LoanID           string          `json:"loan_id" db:"loan_id"`
	CustomerName     string          `json:"customer_name" db:"customer_name"`
	CustomerPhone    *string         `json:"customer_phone" db:"customer_phone"`
	OfficerName      string          `json:"officer_name" db:"officer_name"`
	Branch           string          `json:"branch" db:"branch"`
	LoanAmount       decimal.Decimal `json:"loan_amount" db:"loan_amount"`
	DisbursementDate time.Time       `json:"disbursement_date" db:"disbursement_date"`
	CurrentDPD       int             `json:"current_dpd" db:"current_dpd"`
	TotalOutstanding decimal.Decimal `json:"total_outstanding" db:"total_outstanding"`
	FIMRTagged       *bool           `json:"fimr_tagged,omitempty" db:"fimr_tagged"`
	Status           string          `json:"status" db:"status"`
}

// AllLoan represents a comprehensive loan record for the All Loans view
type AllLoan struct {
	LoanID                 string   `json:"loan_id"`
	CustomerName           string   `json:"customer_name"`
	CustomerPhone          string   `json:"customer_phone"`
	OfficerID              string   `json:"officer_id"`
	OfficerName            string   `json:"officer_name"`
	Region                 string   `json:"region"`
	Branch                 string   `json:"branch"`
	Channel                string   `json:"channel"`
	LoanAmount             float64  `json:"loan_amount"`
	DisbursementDate       string   `json:"disbursement_date"`
	MaturityDate           string   `json:"maturity_date"`
	LoanTermDays           int      `json:"loan_term_days"`
	CurrentDPD             int      `json:"current_dpd"`
	PrincipalOutstanding   float64  `json:"principal_outstanding"`
	InterestOutstanding    float64  `json:"interest_outstanding"`
	FeesOutstanding        float64  `json:"fees_outstanding"`
	TotalOutstanding       float64  `json:"total_outstanding"`
	ActualOutstanding      float64  `json:"actual_outstanding"`
	TotalRepayments        float64  `json:"total_repayments"`
	Status                 string   `json:"status"`
	FIMRTagged             *bool    `json:"fimr_tagged,omitempty"`
	TimelinessScore        *float64 `json:"timeliness_score"`
	RepaymentHealth        *float64 `json:"repayment_health"`
	DaysSinceLastRepayment *int     `json:"days_since_last_repayment"`
}

// TopRiskLoan represents a high-risk loan for audit purposes
type TopRiskLoan struct {
	LoanID                string  `json:"loan_id"`
	CustomerName          string  `json:"customer_name"`
	CustomerPhone         string  `json:"customer_phone"`
	LoanAmount            float64 `json:"loan_amount"`
	DisbursementDate      string  `json:"disbursement_date"`
	CurrentDPD            int     `json:"current_dpd"`
	MaxDPDEver            int     `json:"max_dpd_ever"`
	TotalOutstanding      float64 `json:"total_outstanding"`
	PrincipalOutstanding  float64 `json:"principal_outstanding"`
	InterestOutstanding   float64 `json:"interest_outstanding"`
	FeesOutstanding       float64 `json:"fees_outstanding"`
	Status                string  `json:"status"`
	FIMRTagged            *bool   `json:"fimr_tagged,omitempty"`
	RiskScore             float64 `json:"risk_score"`
	RiskCategory          string  `json:"risk_category"`
	Channel               string  `json:"channel"`
	DaysSinceDisbursement int     `json:"days_since_disbursement"`
}
