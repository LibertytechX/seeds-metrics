package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Repayment represents a repayment record in the database
type Repayment struct {
	RepaymentID       string          `json:"repayment_id" db:"repayment_id"`
	LoanID            string          `json:"loan_id" db:"loan_id"`
	PaymentDate       time.Time       `json:"payment_date" db:"payment_date"`
	PaymentAmount     decimal.Decimal `json:"payment_amount" db:"payment_amount"`
	PrincipalPaid     decimal.Decimal `json:"principal_paid" db:"principal_paid"`
	InterestPaid      decimal.Decimal `json:"interest_paid" db:"interest_paid"`
	FeesPaid          decimal.Decimal `json:"fees_paid" db:"fees_paid"`
	PenaltyPaid       decimal.Decimal `json:"penalty_paid" db:"penalty_paid"`
	PaymentMethod     string          `json:"payment_method" db:"payment_method"`
	PaymentReference  *string         `json:"payment_reference,omitempty" db:"payment_reference"`
	PaymentChannel    *string         `json:"payment_channel,omitempty" db:"payment_channel"`
	DPDAtPayment      int             `json:"dpd_at_payment" db:"dpd_at_payment"`
	IsBackdated       bool            `json:"is_backdated" db:"is_backdated"`
	IsReversed        bool            `json:"is_reversed" db:"is_reversed"`
	ReversalDate      *time.Time      `json:"reversal_date,omitempty" db:"reversal_date"`
	ReversalReason    *string         `json:"reversal_reason,omitempty" db:"reversal_reason"`
	WaiverAmount      decimal.Decimal `json:"waiver_amount" db:"waiver_amount"`
	WaiverType        *string         `json:"waiver_type,omitempty" db:"waiver_type"`
	WaiverApprovedBy  *string         `json:"waiver_approved_by,omitempty" db:"waiver_approved_by"`
	CreatedAt         time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at" db:"updated_at"`
}

// RepaymentInput represents the input payload for creating a repayment
type RepaymentInput struct {
	RepaymentID      string          `json:"repayment_id" binding:"required"`
	LoanID           string          `json:"loan_id" binding:"required"`
	PaymentDate      string          `json:"payment_date" binding:"required"` // YYYY-MM-DD
	PaymentAmount    decimal.Decimal `json:"payment_amount" binding:"required"`
	PrincipalPaid    decimal.Decimal `json:"principal_paid" binding:"required"`
	InterestPaid     decimal.Decimal `json:"interest_paid" binding:"required"`
	FeesPaid         decimal.Decimal `json:"fees_paid" binding:"required"`
	PenaltyPaid      decimal.Decimal `json:"penalty_paid"`
	PaymentMethod    string          `json:"payment_method" binding:"required"`
	PaymentReference *string         `json:"payment_reference"`
	PaymentChannel   *string         `json:"payment_channel"`
	DPDAtPayment     int             `json:"dpd_at_payment"`
	IsBackdated      bool            `json:"is_backdated"`
	IsReversed       bool            `json:"is_reversed"`
	ReversalDate     *string         `json:"reversal_date"` // YYYY-MM-DD
	ReversalReason   *string         `json:"reversal_reason"`
	WaiverAmount     decimal.Decimal `json:"waiver_amount"`
	WaiverType       *string         `json:"waiver_type"`
	WaiverApprovedBy *string         `json:"waiver_approved_by"`
}

// RepaymentFilter represents filter criteria for querying repayments
type RepaymentFilter struct {
	LoanID         *string    `json:"loan_id"`
	OfficerID      *string    `json:"officer_id"`
	PaymentAfter   *time.Time `json:"payment_after"`
	PaymentBefore  *time.Time `json:"payment_before"`
	PaymentMethod  *string    `json:"payment_method"`
	IsReversed     *bool      `json:"is_reversed"`
	Limit          int        `json:"limit"`
	Offset         int        `json:"offset"`
}

