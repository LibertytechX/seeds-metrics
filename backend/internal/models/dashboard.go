package models

import "time"

// PortfolioMetrics represents aggregated portfolio-level KPIs
type PortfolioMetrics struct {
	TotalOverdue15d float64     `json:"totalOverdue15d"`
	AvgDQI          int         `json:"avgDQI"`
	AvgAYR          float64     `json:"avgAYR"`
	AvgRiskScore    int         `json:"avgRiskScore"`
	TopOfficer      *TopOfficer `json:"topOfficer"`
	WatchlistCount  int         `json:"watchlistCount"`
	TotalOfficers   int         `json:"totalOfficers"`
	TotalLoans      int         `json:"totalLoans"`
	TotalPortfolio  float64     `json:"totalPortfolio"`
	Trends          *Trends     `json:"trends,omitempty"`
}

type TopOfficer struct {
	OfficerID string  `json:"officer_id"`
	Name      string  `json:"name"`
	AYR       float64 `json:"ayr"`
}

type Trends struct {
	Overdue15dWoW float64 `json:"overdue15d_wow"`
	DQIChange     int     `json:"dqi_change"`
	AYRChange     float64 `json:"ayr_change"`
}

// DashboardOfficerMetrics represents an officer with all calculated metrics for dashboard
type DashboardOfficerMetrics struct {
	ID                int                `json:"id"`
	OfficerID         string             `json:"officer_id"`
	Name              string             `json:"name"`
	Region            string             `json:"region"`
	Branch            string             `json:"branch"`
	Channel           string             `json:"channel"`
	RawMetrics        *RawMetrics        `json:"rawMetrics"`
	CalculatedMetrics *CalculatedMetrics `json:"calculatedMetrics"`
	RiskBand          string             `json:"riskBand"`
}

type RawMetrics struct {
	FirstMiss                 int     `json:"firstMiss"`
	Disbursed                 int     `json:"disbursed"`
	Dpd1to6Bal                float64 `json:"dpd1to6Bal"`
	AmountDue7d               float64 `json:"amountDue7d"`
	MovedTo7to30              float64 `json:"movedTo7to30"`
	PrevDpd1to6Bal            float64 `json:"prevDpd1to6Bal"`
	FeesCollected             float64 `json:"feesCollected"`
	FeesDue                   float64 `json:"feesDue"`
	InterestCollected         float64 `json:"interestCollected"`
	Overdue15d                float64 `json:"overdue15d"`
	TotalPortfolio            float64 `json:"totalPortfolio"`
	Par15MidMonth             float64 `json:"par15MidMonth"`
	Waivers                   float64 `json:"waivers"`
	Backdated                 int     `json:"backdated"`
	Entries                   int     `json:"entries"`
	Reversals                 int     `json:"reversals"`
	HadFloatGap               bool    `json:"hadFloatGap"`
	AvgTimelinessScore        float64 `json:"avgTimelinessScore"`
	AvgRepaymentHealth        float64 `json:"avgRepaymentHealth"`
	AvgDaysSinceLastRepayment float64 `json:"avgDaysSinceLastRepayment"`
	AvgLoanAge                float64 `json:"avgLoanAge"`
	ActiveLoansCount          int     `json:"activeLoansCount"`
}

type CalculatedMetrics struct {
	FIMR                      float64 `json:"fimr"`
	Slippage                  float64 `json:"slippage"`
	Roll                      float64 `json:"roll"`
	FRR                       float64 `json:"frr"`
	AYR                       float64 `json:"ayr"`
	DQI                       int     `json:"dqi"`
	RiskScore                 int     `json:"riskScore"`
	Yield                     float64 `json:"yield"`
	Overdue15dVolume          float64 `json:"overdue15dVolume"`
	RiskScoreNorm             float64 `json:"riskScoreNorm"`
	OnTimeRate                float64 `json:"onTimeRate"`
	ChannelPurity             float64 `json:"channelPurity"`
	PORR                      float64 `json:"porr"`
	AvgTimelinessScore        float64 `json:"avgTimelinessScore"`
	AvgRepaymentHealth        float64 `json:"avgRepaymentHealth"`
	AvgDaysSinceLastRepayment float64 `json:"avgDaysSinceLastRepayment"`
	AvgLoanAge                float64 `json:"avgLoanAge"`
	RepaymentDelayRate        float64 `json:"repaymentDelayRate"`
}

// FIMRLoan represents a loan that missed first installment
type FIMRLoan struct {
	LoanID                  string  `json:"loan_id"`
	OfficerID               string  `json:"officer_id"`
	OfficerName             string  `json:"officer_name"`
	Region                  string  `json:"region"`
	Branch                  string  `json:"branch"`
	CustomerID              string  `json:"customer_id"`
	CustomerName            string  `json:"customer_name"`
	CustomerPhone           string  `json:"customer_phone"`
	DisbursementDate        string  `json:"disbursement_date"`
	LoanAmount              float64 `json:"loan_amount"`
	FirstPaymentDueDate     string  `json:"first_payment_due_date"`
	DaysSinceDue            int     `json:"days_since_due"`
	AmountDue1stInstallment float64 `json:"amount_due_1st_installment"`
	AmountPaid              float64 `json:"amount_paid"`
	OutstandingBalance      float64 `json:"outstanding_balance"`
	CurrentDPD              int     `json:"current_dpd"`
	Channel                 string  `json:"channel"`
	Status                  string  `json:"status"`
	FIMRTagged              bool    `json:"fimr_tagged"`
}

// EarlyIndicatorLoan represents a loan in early delinquency
type EarlyIndicatorLoan struct {
	LoanID              string  `json:"loan_id"`
	OfficerID           string  `json:"officer_id"`
	OfficerName         string  `json:"officer_name"`
	Region              string  `json:"region"`
	Branch              string  `json:"branch"`
	CustomerID          string  `json:"customer_id"`
	CustomerName        string  `json:"customer_name"`
	CustomerPhone       string  `json:"customer_phone"`
	DisbursementDate    string  `json:"disbursement_date"`
	LoanAmount          float64 `json:"loan_amount"`
	CurrentDPD          int     `json:"current_dpd"`
	PreviousDPDStatus   string  `json:"previous_dpd_status"`
	DaysInCurrentStatus int     `json:"days_in_current_status"`
	AmountDue           float64 `json:"amount_due"`
	AmountPaid          float64 `json:"amount_paid"`
	OutstandingBalance  float64 `json:"outstanding_balance"`
	Channel             string  `json:"channel"`
	Status              string  `json:"status"`
	FIMRTagged          bool    `json:"fimr_tagged"`
	RollDirection       string  `json:"roll_direction"`
	LastPaymentDate     string  `json:"last_payment_date"`
}

// DashboardBranchMetrics represents branch-level aggregated metrics for dashboard
type DashboardBranchMetrics struct {
	Branch         string  `json:"branch"`
	Region         string  `json:"region"`
	PortfolioTotal float64 `json:"portfolio_total"`
	Overdue15d     float64 `json:"overdue_15d"`
	Par15Ratio     float64 `json:"par15_ratio"`
	AYR            float64 `json:"ayr"`
	DQI            int     `json:"dqi"`
	FIMR           float64 `json:"fimr"`
	ActiveLoans    int     `json:"active_loans"`
	TotalOfficers  int     `json:"total_officers"`
}

// TeamMember represents a team member for audit assignment
type TeamMember struct {
	ID   interface{} `json:"id"` // Can be int, string, or 0
	Name string      `json:"name"`
	Role string      `json:"role"`
}

// AuditUpdate represents an audit assignment update
type AuditUpdate struct {
	AssigneeID   int    `json:"assignee_id"`
	AssigneeName string `json:"assignee_name"`
	AuditStatus  string `json:"audit_status"`
}

// AuditHistory represents audit history for an officer
type AuditHistory struct {
	ID           int       `json:"id"`
	OfficerID    string    `json:"officer_id"`
	AssigneeID   int       `json:"assignee_id"`
	AssigneeName string    `json:"assignee_name"`
	AuditStatus  string    `json:"audit_status"`
	AuditDate    string    `json:"audit_date"`
	Notes        string    `json:"notes"`
	CreatedAt    time.Time `json:"created_at"`
}

// DashboardPagination represents pagination metadata for dashboard
type DashboardPagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

// FilterOptions represents filter dropdown options
type FilterOptions struct {
	Branches []string        `json:"branches,omitempty"`
	Regions  []string        `json:"regions,omitempty"`
	Channels []string        `json:"channels,omitempty"`
	Officers []OfficerOption `json:"officers,omitempty"`
}

type OfficerOption struct {
	OfficerID string `json:"officer_id"`
	Name      string `json:"name"`
	Branch    string `json:"branch"`
	Region    string `json:"region"`
}

// LoanDetail represents detailed loan information
type LoanDetail struct {
	Loan       *Loan              `json:"loan"`
	Repayments []Repayment        `json:"repayments"`
	Schedule   []LoanScheduleItem `json:"schedule"`
}

// LoanScheduleItem represents a single installment in the loan schedule
type LoanScheduleItem struct {
	InstallmentNumber int     `json:"installment_number"`
	DueDate           string  `json:"due_date"`
	PrincipalDue      float64 `json:"principal_due"`
	InterestDue       float64 `json:"interest_due"`
	FeesDue           float64 `json:"fees_due"`
	TotalDue          float64 `json:"total_due"`
	PrincipalPaid     float64 `json:"principal_paid"`
	InterestPaid      float64 `json:"interest_paid"`
	FeesPaid          float64 `json:"fees_paid"`
	BalanceAfter      float64 `json:"balance_after"`
	Status            string  `json:"status"`
	DaysOverdue       int     `json:"days_overdue"`
}

// Helper function to calculate risk band
func GetRiskBand(riskScore int) string {
	if riskScore >= 80 {
		return "Green"
	} else if riskScore >= 60 {
		return "Watch"
	} else if riskScore >= 40 {
		return "Amber"
	}
	return "Red"
}

// Helper function to calculate DPD status
func GetDPDStatus(dpd int) string {
	if dpd == 0 {
		return "Current"
	} else if dpd >= 1 && dpd <= 3 {
		return "D1-3"
	} else if dpd >= 4 && dpd <= 6 {
		return "D4-6"
	} else if dpd >= 7 && dpd <= 15 {
		return "Rolled to D7-15"
	} else if dpd >= 16 && dpd <= 30 {
		return "Rolled to D16-30"
	}
	return "Overdue"
}
