package models

import (
	"time"

	"github.com/shopspring/decimal"
)

// Officer represents a loan officer
type Officer struct {
	OfficerID    string     `json:"officer_id" db:"officer_id"`
	OfficerName  string     `json:"officer_name" db:"officer_name"`
	OfficerPhone *string    `json:"officer_phone,omitempty" db:"officer_phone"`
	Region       string     `json:"region" db:"region"`
	Branch       string     `json:"branch" db:"branch"`
	Status       string     `json:"status" db:"status"`
	HireDate     *time.Time `json:"hire_date,omitempty" db:"hire_date"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// OfficerMetrics represents calculated metrics for an officer
type OfficerMetrics struct {
	OfficerID         string          `json:"officer_id" db:"officer_id"`
	OfficerName       string          `json:"officer_name" db:"officer_name"`
	Region            string          `json:"region" db:"region"`
	Branch            string          `json:"branch" db:"branch"`
	CalculationDate   time.Time       `json:"calculation_date" db:"calculation_date"`
	
	// Portfolio Metrics
	TotalDisbursed    int             `json:"total_disbursed" db:"total_disbursed"`
	TotalAmount       decimal.Decimal `json:"total_amount" db:"total_amount"`
	ActiveLoans       int             `json:"active_loans" db:"active_loans"`
	ClosedLoans       int             `json:"closed_loans" db:"closed_loans"`
	
	// Quality Metrics
	FIMRRate          decimal.Decimal `json:"fimr_rate" db:"fimr_rate"`
	PAR15Rate         decimal.Decimal `json:"par15_rate" db:"par15_rate"`
	PAR30Rate         decimal.Decimal `json:"par30_rate" db:"par30_rate"`
	D0to6Slippage     decimal.Decimal `json:"d0_6_slippage" db:"d0_6_slippage"`
	RollRate          decimal.Decimal `json:"roll_rate" db:"roll_rate"`
	
	// Financial Metrics
	FRR               decimal.Decimal `json:"frr" db:"frr"`
	AYR               decimal.Decimal `json:"ayr" db:"ayr"`
	
	// Composite Scores
	DQI               decimal.Decimal `json:"dqi" db:"dqi"`
	RiskScore         decimal.Decimal `json:"risk_score" db:"risk_score"`
	
	CreatedAt         time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at" db:"updated_at"`
}

// OfficerSummary represents a summary view of officer metrics
type OfficerSummary struct {
	OfficerID       string          `json:"officer_id" db:"officer_id"`
	OfficerName     string          `json:"officer_name" db:"officer_name"`
	Region          string          `json:"region" db:"region"`
	Branch          string          `json:"branch" db:"branch"`
	TotalDisbursed  int             `json:"total_disbursed" db:"total_disbursed"`
	ActiveLoans     int             `json:"active_loans" db:"active_loans"`
	FIMRRate        decimal.Decimal `json:"fimr_rate" db:"fimr_rate"`
	PAR15Rate       decimal.Decimal `json:"par15_rate" db:"par15_rate"`
	DQI             decimal.Decimal `json:"dqi" db:"dqi"`
	RiskScore       decimal.Decimal `json:"risk_score" db:"risk_score"`
}

// BranchMetrics represents aggregated metrics for a branch
type BranchMetrics struct {
	Region          string          `json:"region" db:"region"`
	Branch          string          `json:"branch" db:"branch"`
	CalculationDate time.Time       `json:"calculation_date" db:"calculation_date"`
	
	// Portfolio Metrics
	TotalOfficers     int             `json:"total_officers" db:"total_officers"`
	TotalDisbursed    int             `json:"total_disbursed" db:"total_disbursed"`
	TotalAmount       decimal.Decimal `json:"total_amount" db:"total_amount"`
	ActiveLoans       int             `json:"active_loans" db:"active_loans"`
	
	// Quality Metrics
	FIMRRate          decimal.Decimal `json:"fimr_rate" db:"fimr_rate"`
	PAR15Rate         decimal.Decimal `json:"par15_rate" db:"par15_rate"`
	PAR30Rate         decimal.Decimal `json:"par30_rate" db:"par30_rate"`
	D0to6Slippage     decimal.Decimal `json:"d0_6_slippage" db:"d0_6_slippage"`
	RollRate          decimal.Decimal `json:"roll_rate" db:"roll_rate"`
	
	// Financial Metrics
	FRR               decimal.Decimal `json:"frr" db:"frr"`
	AYR               decimal.Decimal `json:"ayr" db:"ayr"`
	
	// Composite Scores
	DQI               decimal.Decimal `json:"dqi" db:"dqi"`
	RiskScore         decimal.Decimal `json:"risk_score" db:"risk_score"`
	
	CreatedAt         time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at" db:"updated_at"`
}

