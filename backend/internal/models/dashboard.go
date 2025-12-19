package models

import "time"

// PortfolioMetrics represents aggregated portfolio-level KPIs
type PortfolioMetrics struct {
	TotalOverdue15d    float64     `json:"totalOverdue15d"`
	ActualOverdue15d   float64     `json:"actualOverdue15d"` // Only installments due to date
	AvgDQI             int         `json:"avgDQI"`           // Will be removed from frontend
	AvgAYR             float64     `json:"avgAYR"`
	AvgRiskScore       int         `json:"avgRiskScore"`
	TopOfficer         *TopOfficer `json:"topOfficer"`
	WatchlistCount     int         `json:"watchlistCount"`
	WatchlistPortfolio float64     `json:"watchlistPortfolio"` // Total portfolio of watchlist officers
	TotalOfficers      int         `json:"totalOfficers"`
	TotalLoans         int         `json:"totalLoans"`
	TotalPortfolio     float64     `json:"totalPortfolio"`
	Trends             *Trends     `json:"trends,omitempty"`

	// Active vs Inactive Loans
	ActiveLoansCount    int     `json:"activeLoansCount"`
	ActiveLoansVolume   float64 `json:"activeLoansVolume"`
	InactiveLoansCount  int     `json:"inactiveLoansCount"`
	InactiveLoansVolume float64 `json:"inactiveLoansVolume"`

	// ROT (Risk of Termination) Analysis
	EarlyROTCount  int     `json:"earlyROTCount"`
	EarlyROTVolume float64 `json:"earlyROTVolume"`
	LateROTCount   int     `json:"lateROTCount"`
	LateROTVolume  float64 `json:"lateROTVolume"`

	// Portfolio Delinquency Risk
	AtRiskOfficersCount      int     `json:"atRiskOfficersCount"`
	AtRiskOfficersPercentage float64 `json:"atRiskOfficersPercentage"`

	// Portfolio Repayment Behavior Metrics
	AvgDaysPastDue        float64 `json:"avgDaysPastDue"`
	AvgTimelinessScore    float64 `json:"avgTimelinessScore"`
	AvgRepaymentDelayRate float64 `json:"avgRepaymentDelayRate"`

	// Total DPD Loans (current_dpd > 0 AND status in Active/Defaulted)
	TotalDPDLoansCount        int     `json:"totalDPDLoansCount"`
	TotalDPDActualOutstanding float64 `json:"totalDPDActualOutstanding"`
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

// PortfolioLoanMetrics represents loan-level aggregated metrics for portfolio
type PortfolioLoanMetrics struct {
	ActiveLoansCount    int     `json:"activeLoansCount"`
	ActiveLoansVolume   float64 `json:"activeLoansVolume"`
	InactiveLoansCount  int     `json:"inactiveLoansCount"`
	InactiveLoansVolume float64 `json:"inactiveLoansVolume"`
	EarlyROTCount       int     `json:"earlyROTCount"`
	EarlyROTVolume      float64 `json:"earlyROTVolume"`
	LateROTCount        int     `json:"lateROTCount"`
	LateROTVolume       float64 `json:"lateROTVolume"`
	AvgDaysPastDue      float64 `json:"avgDaysPastDue"`
	AvgTimelinessScore  float64 `json:"avgTimelinessScore"`
}

// DashboardOfficerMetrics represents an officer with all calculated metrics for dashboard
type DashboardOfficerMetrics struct {
	ID                int                `json:"id"`
	OfficerID         string             `json:"officer_id"`
	Name              string             `json:"name"`
	Email             string             `json:"email,omitempty"`
	Region            string             `json:"region"`
	Branch            string             `json:"branch"`
	Channel           string             `json:"channel"`
	UserType          *string            `json:"user_type,omitempty"`
	HireDate          *time.Time         `json:"hire_date,omitempty"`
	SupervisorEmail   *string            `json:"supervisor_email,omitempty"`
	SupervisorName    *string            `json:"supervisor_name,omitempty"`
	VerticalLeadEmail *string            `json:"vertical_lead_email,omitempty"`
	VerticalLeadName  *string            `json:"vertical_lead_name,omitempty"`
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
	LoanID                   string  `json:"loan_id"`
	OfficerID                string  `json:"officer_id"`
	OfficerName              string  `json:"officer_name"`
	Region                   string  `json:"region"`
	Branch                   string  `json:"branch"`
	CustomerID               string  `json:"customer_id"`
	CustomerName             string  `json:"customer_name"`
	CustomerPhone            string  `json:"customer_phone"`
	DisbursementDate         string  `json:"disbursement_date"`
	LoanAmount               float64 `json:"loan_amount"`
	MaturityDate             string  `json:"maturity_date"`
	FirstPaymentDueDate      string  `json:"first_payment_due_date"`
	FirstPaymentReceivedDate *string `json:"first_payment_received_date,omitempty"`
	DaysSinceDue             *int    `json:"days_since_due,omitempty"`
	AmountDue1stInstallment  float64 `json:"amount_due_1st_installment"`
	AmountPaid               float64 `json:"amount_paid"`
	ActualOutstanding        float64 `json:"actual_outstanding"`
	OutstandingBalance       float64 `json:"outstanding_balance"`
	TotalRepayments          float64 `json:"total_repayments"`
	CurrentDPD               int     `json:"current_dpd"`
	Channel                  string  `json:"channel"`
	Status                   string  `json:"status"`
	DjangoStatus             string  `json:"django_status,omitempty"`
	FIMRTagged               bool    `json:"fimr_tagged"`
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
	Branch                string  `json:"branch"`
	Region                string  `json:"region"`
	PortfolioTotal        float64 `json:"portfolio_total"`
	Overdue15d            float64 `json:"overdue_15d"`
	Par15Ratio            float64 `json:"par15_ratio"`
	AYR                   float64 `json:"ayr"`
	DQI                   int     `json:"dqi"`
	FIMR                  float64 `json:"fimr"`
	ActiveLoans           int     `json:"active_loans"`
	TotalOfficers         int     `json:"total_officers"`
	AvgRepaymentDelayRate float64 `json:"avg_repayment_delay_rate"`
}

// VerticalLeadMetricsRow represents aggregated loan metrics per vertical lead
// for the Credit Health by Branch "By Vertical Lead" view.
type VerticalLeadMetricsRow struct {
	VerticalLeadName  string  `json:"vertical_lead_name"`
	VerticalLeadEmail string  `json:"vertical_lead_email"`
	Branches          int     `json:"branches"`
	ActiveLOs         int     `json:"active_los"`
	Loans             int     `json:"loans"`
	Outstanding       float64 `json:"outstanding"`
	AvgDPD            float64 `json:"avg_dpd"`
	MaxDPD            int     `json:"max_dpd"`
	DPD0              int     `json:"dpd0"`
	DPD1to6           int     `json:"dpd1_6"`
	DPD7to14          int     `json:"dpd7_14"`
	DPD14to21         int     `json:"dpd14_21"`
	DPD21Plus         int     `json:"dpd21_plus"`
	Quiet             int     `json:"quiet"`
	QuietValue        float64 `json:"quiet_value"`
}

// BranchCollectionsLeaderboardRow represents per-branch collections metrics for the
// Collections Control Centre "Branch Leaderboard" table.
type BranchCollectionsLeaderboardRow struct {
	Branch         string  `json:"branch"`
	Region         string  `json:"region"`
	PortfolioTotal float64 `json:"portfolio_total"`
	Overdue15d     float64 `json:"overdue_15d"`
	DueToday       float64 `json:"due_today"`
	CollectedToday float64 `json:"collected_today"`
	TodayRate      float64 `json:"today_rate"`
	MTDRate        float64 `json:"mtd_rate"`
	ProgressRate   float64 `json:"progress_rate"`
	MissedToday    float64 `json:"missed_today"`
	NPLRatio       float64 `json:"npl_ratio"`
	Status         string  `json:"status"`
}

// OfficerCollectionsLeaderboardRow represents per-officer collections metrics for the
// Agent/Officer Leaderboard views. It mirrors BranchCollectionsLeaderboardRow but is
// grouped by officer instead of branch.
type OfficerCollectionsLeaderboardRow struct {
	OfficerID      string  `json:"officer_id"`
	OfficerName    string  `json:"officer_name"`
	OfficerEmail   string  `json:"officer_email"`
	Branch         string  `json:"branch"`
	Region         string  `json:"region"`
	PortfolioTotal float64 `json:"portfolio_total"`
	Overdue15d     float64 `json:"overdue_15d"`
	DueToday       float64 `json:"due_today"`
	CollectedToday float64 `json:"collected_today"`
	TodayRate      float64 `json:"today_rate"`
	MTDRate        float64 `json:"mtd_rate"`
	ProgressRate   float64 `json:"progress_rate"`
	MissedToday    float64 `json:"missed_today"`
	NPLRatio       float64 `json:"npl_ratio"`
	Status         string  `json:"status"`
}

// RepaymentWatchOfficerRow represents per-officer Wave 2 repayment performance for the
// Repayment Watch view in the Collections Control Centre.
type RepaymentWatchOfficerRow struct {
	OfficerID               string  `json:"officer_id"`
	OfficerName             string  `json:"officer_name"`
	OfficerEmail            string  `json:"officer_email"`
	Branch                  string  `json:"branch"`
	Region                  string  `json:"region"`
	TotalWave2OpenLoans     int     `json:"total_wave2_open_loans"`
	LoansWithRepaymentToday int     `json:"loans_with_repayment_today"`
	AmountCollectedToday    float64 `json:"amount_collected_today"`
	RepaymentRate           float64 `json:"repayment_rate"`
}

// AgentActivitySummary represents aggregated counts for the Agent Activity
// categories used in the Collections Control Centre.
type AgentActivitySummary struct {
	CriticalNoCollectionCount int `json:"critical_no_collection"`
	StoppedCollectingCount    int `json:"stopped_collecting"`
	SevereDeclineCount        int `json:"severe_decline"`
	NotYetStartedTodayCount   int `json:"not_yet_started_today"`
	StrongGrowthCount         int `json:"strong_growth"`
	StartedTodayCount         int `json:"started_today"`
}

// AgentActivityDetailRow represents a single officer row in the Agent Activity
// drilldown view. It contains 7-day per-day repayment amounts and an
// activity/repayment rate, along with officer and location metadata.
type AgentActivityDetailRow struct {
	OfficerID           string  `json:"officer_id"`
	OfficerName         string  `json:"officer_name"`
	OfficerEmail        string  `json:"officer_email"`
	Branch              string  `json:"branch"`
	Region              string  `json:"region"`
	RepaymentRate       float64 `json:"repayment_rate"`
	Amount5DaysAgo      float64 `json:"amount_5_days_ago"`
	Amount4DaysAgo      float64 `json:"amount_4_days_ago"`
	Amount3DaysAgo      float64 `json:"amount_3_days_ago"`
	Amount2DaysAgo      float64 `json:"amount_2_days_ago"`
	Amount2DaysAgoExact float64 `json:"amount_2_days_ago_exact"`
	Amount1DayAgo       float64 `json:"amount_1_day_ago"`
	AmountToday         float64 `json:"amount_today"`
	TotalCollected      float64 `json:"total_collected"`
}

// DailyCollectionsPoint represents a single day in the collections time series
// used by the Collections Control Centre daily chart.
type DailyCollectionsPoint struct {
	Date            string  `json:"date"`
	CollectedAmount float64 `json:"collected_amount"`
	RepaymentsCount int     `json:"repayments_count"`

	// Repayment type breakdown for the day. These amounts always sum to
	// CollectedAmount and are grouped using normalised payment_method values
	// (e.g. AGENT_DEBIT, TRANSFER, ESCROW_DEBIT; everything else is "other").
	AgentDebitAmount      float64 `json:"agent_debit_amount"`
	TransferAmount        float64 `json:"transfer_amount"`
	EscrowDebitAmount     float64 `json:"escrow_debit_amount"`
	OtherRepaymentsAmount float64 `json:"other_repayments_amount"`
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
	OfficerID string  `json:"officer_id"`
	Name      string  `json:"name"`
	Email     *string `json:"email,omitempty"`
	Branch    string  `json:"branch"`
	Region    string  `json:"region"`
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
