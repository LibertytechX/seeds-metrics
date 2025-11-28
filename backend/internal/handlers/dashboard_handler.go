package handlers

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/seeds-metrics/analytics-backend/internal/models"
	"github.com/seeds-metrics/analytics-backend/internal/repository"
	"github.com/seeds-metrics/analytics-backend/internal/services"
)

// DashboardHandler handles dashboard API requests
type DashboardHandler struct {
	dashboardRepo  *repository.DashboardRepository
	repaymentRepo  *repository.RepaymentRepository
	metricsService *services.MetricsService
	syncService    *services.SyncService
}

// NewDashboardHandler creates a new dashboard handler
func NewDashboardHandler(dashboardRepo *repository.DashboardRepository, repaymentRepo *repository.RepaymentRepository, metricsService *services.MetricsService, syncService *services.SyncService) *DashboardHandler {
	return &DashboardHandler{
		dashboardRepo:  dashboardRepo,
		repaymentRepo:  repaymentRepo,
		metricsService: metricsService,
		syncService:    syncService,
	}
}

// Helper function to create API error
func newAPIError(code, message string) *models.APIError {
	return &models.APIError{
		Code:    code,
		Message: message,
	}
}

// GetPortfolioMetrics handles GET /api/v1/metrics/portfolio
// @Summary Get portfolio metrics
// @Description Get aggregated portfolio-level metrics including total overdue, DQI, AYR, and risk scores
// @Tags Metrics
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse{data=models.PortfolioMetrics}
// @Failure 500 {object} models.APIResponse
// @Router /metrics/portfolio [get]
func (h *DashboardHandler) GetPortfolioMetrics(c *gin.Context) {
	// Parse filters
	filters := make(map[string]interface{})
	if wave := c.Query("wave"); wave != "" {
		filters["wave"] = wave
	}

	// Set high limit to fetch all officers for portfolio-level aggregation
	filters["limit"] = 100000

	// Get all officers with metrics
	officers, err := h.dashboardRepo.GetOfficers(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to retrieve portfolio metrics",
			Error:   newAPIError("PORTFOLIO_METRICS_ERROR", err.Error()),
		})
		return
	}

	// Calculate metrics for each officer
	for _, officer := range officers {
		officer.CalculatedMetrics = h.metricsService.CalculateOfficerMetrics(officer.RawMetrics)
		officer.RiskBand = models.GetRiskBand(officer.CalculatedMetrics.RiskScore)
	}

	// Calculate portfolio-level metrics from officers
	portfolio := h.metricsService.CalculatePortfolioMetrics(officers)

	// Get loan-level metrics for new portfolio cards
	loanMetrics, err := h.dashboardRepo.GetPortfolioLoanMetrics(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to retrieve loan metrics",
			Error:   newAPIError("LOAN_METRICS_ERROR", err.Error()),
		})
		return
	}

	// Merge loan metrics into portfolio
	portfolio.ActiveLoansCount = loanMetrics.ActiveLoansCount
	portfolio.ActiveLoansVolume = loanMetrics.ActiveLoansVolume
	portfolio.InactiveLoansCount = loanMetrics.InactiveLoansCount
	portfolio.InactiveLoansVolume = loanMetrics.InactiveLoansVolume
	portfolio.EarlyROTCount = loanMetrics.EarlyROTCount
	portfolio.EarlyROTVolume = loanMetrics.EarlyROTVolume
	portfolio.LateROTCount = loanMetrics.LateROTCount
	portfolio.LateROTVolume = loanMetrics.LateROTVolume
	portfolio.AvgDaysPastDue = loanMetrics.AvgDaysPastDue
	portfolio.AvgTimelinessScore = loanMetrics.AvgTimelinessScore

	// Get actual overdue amount (only installments due to date)
	actualOverdue15d, err := h.dashboardRepo.GetActualOverdue15d(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to retrieve actual overdue amount",
			Error:   newAPIError("ACTUAL_OVERDUE_ERROR", err.Error()),
		})
		return
	}
	portfolio.ActualOverdue15d = actualOverdue15d

	// Get total DPD loans count and actual outstanding
	totalDPDLoansCount, totalDPDActualOutstanding, err := h.dashboardRepo.GetTotalDPDLoans(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to retrieve total DPD loans",
			Error:   newAPIError("TOTAL_DPD_LOANS_ERROR", err.Error()),
		})
		return
	}
	portfolio.TotalDPDLoansCount = totalDPDLoansCount
	portfolio.TotalDPDActualOutstanding = totalDPDActualOutstanding

	c.JSON(http.StatusOK, models.APIResponse{
		Status: "success",
		Data:   portfolio,
	})
}

// GetOfficers handles GET /api/v1/officers
// @Summary Get all officers
// @Description Get list of loan officers with their performance metrics and calculated scores
// @Tags Officers
// @Accept json
// @Produce json
// @Param branch query string false "Filter by branch"
// @Param region query string false "Filter by region"
// @Param channel query string false "Filter by channel"
// @Param user_type query string false "Filter by user type"
// @Param officer_email query string false "Filter by officer email (partial match)"
// @Param sort_by query string false "Sort field (e.g., risk_score, total_portfolio)"
// @Param sort_dir query string false "Sort direction (asc/desc)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(50)
// @Success 200 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /officers [get]
func (h *DashboardHandler) GetOfficers(c *gin.Context) {
	// Parse filters from query parameters
	filters := make(map[string]interface{})

	if branch := c.Query("branch"); branch != "" {
		filters["branch"] = branch
	}
	if region := c.Query("region"); region != "" {
		// Support comma-separated regions for multi-select
		filters["region"] = region
	}
	if channel := c.Query("channel"); channel != "" {
		filters["channel"] = channel
	}
	if wave := c.Query("wave"); wave != "" {
		filters["wave"] = wave
	}
	if userType := c.Query("user_type"); userType != "" {
		filters["user_type"] = userType
	}
	if officerEmail := c.Query("officer_email"); officerEmail != "" {
		filters["officer_email"] = officerEmail
	}
	if sortBy := c.Query("sort_by"); sortBy != "" {
		filters["sort_by"] = sortBy
	}
	if sortDir := c.Query("sort_dir"); sortDir != "" {
		filters["sort_dir"] = sortDir
	}

	// Parse pagination
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	filters["page"] = page
	filters["limit"] = limit

	// Get officers
	officers, err := h.dashboardRepo.GetOfficers(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to retrieve officers",
			Error:   newAPIError("INTERNAL_ERROR", err.Error()),
		})
		return
	}

	// Calculate metrics for each officer
	for _, officer := range officers {
		officer.CalculatedMetrics = h.metricsService.CalculateOfficerMetrics(officer.RawMetrics)
		officer.RiskBand = models.GetRiskBand(officer.CalculatedMetrics.RiskScore)
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"officers": officers,
			"pagination": map[string]interface{}{
				"page":  page,
				"limit": limit,
				"total": len(officers),
			},
		},
	})
}

// GetOfficerByID handles GET /api/v1/officers/:officer_id
// @Summary Get officer by ID
// @Description Get detailed information about a specific loan officer including metrics and risk band
// @Tags Officers
// @Accept json
// @Produce json
// @Param officer_id path string true "Officer ID"
// @Success 200 {object} models.APIResponse{data=models.Officer}
// @Failure 404 {object} models.APIResponse
// @Router /officers/{officer_id} [get]
func (h *DashboardHandler) GetOfficerByID(c *gin.Context) {
	officerID := c.Param("officer_id")

	officer, err := h.dashboardRepo.GetOfficerByID(officerID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Status:  "error",
			Message: "Officer not found",
			Error:   newAPIError("INTERNAL_ERROR", err.Error()),
		})
		return
	}

	// Calculate metrics
	officer.CalculatedMetrics = h.metricsService.CalculateOfficerMetrics(officer.RawMetrics)
	officer.RiskBand = models.GetRiskBand(officer.CalculatedMetrics.RiskScore)

	c.JSON(http.StatusOK, models.APIResponse{
		Status: "success",
		Data:   officer,
	})
}

// GetFIMRLoans handles GET /api/v1/fimr/loans
func (h *DashboardHandler) GetFIMRLoans(c *gin.Context) {
	// Parse filters
	filters := make(map[string]interface{})

	if officerID := c.Query("officer_id"); officerID != "" {
		filters["officer_id"] = officerID
	}
	if branch := c.Query("branch"); branch != "" {
		filters["branch"] = branch
	}
	if region := c.Query("region"); region != "" {
		filters["region"] = region
	}
	if channel := c.Query("channel"); channel != "" {
		filters["channel"] = channel
	}
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}

	// Default django_status filter for FIMR drilldown:
	// if caller does not specify django_status explicitly, restrict to OPEN and PAST_MATURITY loans.
	if djangoStatus := c.Query("django_status"); djangoStatus != "" {
		filters["django_status"] = djangoStatus
	} else {
		filters["django_status"] = "OPEN,PAST_MATURITY"
	}
	if wave := c.Query("wave"); wave != "" {
		filters["wave"] = wave
	}
	if sortBy := c.Query("sort_by"); sortBy != "" {
		filters["sort_by"] = sortBy
	}
	if sortDir := c.Query("sort_dir"); sortDir != "" {
		filters["sort_dir"] = sortDir
	}

	loans, err := h.dashboardRepo.GetFIMRLoans(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to retrieve FIMR loans",
			Error:   newAPIError("INTERNAL_ERROR", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"loans": loans,
			"total": len(loans),
		},
	})
}

// GetFIMRSummary handles GET /api/v1/fimr/summary
func (h *DashboardHandler) GetFIMRSummary(c *gin.Context) {
	// Parse filters
	filters := make(map[string]interface{})

	if officerID := c.Query("officer_id"); officerID != "" {
		filters["officer_id"] = officerID
	}
	if branch := c.Query("branch"); branch != "" {
		filters["branch"] = branch
	}
	if region := c.Query("region"); region != "" {
		filters["region"] = region
	}
	if wave := c.Query("wave"); wave != "" {
		filters["wave"] = wave
	}

	loans, err := h.dashboardRepo.GetFIMRLoans(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to retrieve FIMR summary",
			Error:   newAPIError("INTERNAL_ERROR", err.Error()),
		})
		return
	}

	// Calculate summary statistics
	var totalAmount float64
	var totalOutstanding float64
	for _, loan := range loans {
		totalAmount += loan.LoanAmount
		totalOutstanding += loan.OutstandingBalance
	}

	summary := map[string]interface{}{
		"total_loans":       len(loans),
		"total_amount":      totalAmount,
		"total_outstanding": totalOutstanding,
		"avg_dpd":           0,
	}

	if len(loans) > 0 {
		var totalDPD int
		for _, loan := range loans {
			totalDPD += loan.CurrentDPD
		}
		summary["avg_dpd"] = totalDPD / len(loans)
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Status: "success",
		Data:   summary,
	})
}

// GetEarlyIndicatorLoans handles GET /api/v1/early-indicators/loans
func (h *DashboardHandler) GetEarlyIndicatorLoans(c *gin.Context) {
	// Parse filters
	filters := make(map[string]interface{})

	if officerID := c.Query("officer_id"); officerID != "" {
		filters["officer_id"] = officerID
	}
	if branch := c.Query("branch"); branch != "" {
		filters["branch"] = branch
	}
	if region := c.Query("region"); region != "" {
		filters["region"] = region
	}
	if channel := c.Query("channel"); channel != "" {
		filters["channel"] = channel
	}
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if wave := c.Query("wave"); wave != "" {
		filters["wave"] = wave
	}
	if sortBy := c.Query("sort_by"); sortBy != "" {
		filters["sort_by"] = sortBy
	}
	if sortDir := c.Query("sort_dir"); sortDir != "" {
		filters["sort_dir"] = sortDir
	}

	loans, err := h.dashboardRepo.GetEarlyIndicatorLoans(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to retrieve early indicator loans",
			Error:   newAPIError("INTERNAL_ERROR", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"loans": loans,
			"total": len(loans),
		},
	})
}

// GetEarlyIndicatorSummary handles GET /api/v1/early-indicators/summary
func (h *DashboardHandler) GetEarlyIndicatorSummary(c *gin.Context) {
	// Parse filters
	filters := make(map[string]interface{})

	if officerID := c.Query("officer_id"); officerID != "" {
		filters["officer_id"] = officerID
	}
	if branch := c.Query("branch"); branch != "" {
		filters["branch"] = branch
	}
	if region := c.Query("region"); region != "" {
		filters["region"] = region
	}
	if wave := c.Query("wave"); wave != "" {
		filters["wave"] = wave
	}

	loans, err := h.dashboardRepo.GetEarlyIndicatorLoans(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to retrieve early indicator summary",
			Error:   newAPIError("INTERNAL_ERROR", err.Error()),
		})
		return
	}

	// Calculate summary statistics
	var totalAmount float64
	var totalOutstanding float64
	var worsening, stable, improving int

	for _, loan := range loans {
		totalAmount += loan.LoanAmount
		totalOutstanding += loan.OutstandingBalance

		switch loan.RollDirection {
		case "Worsening":
			worsening++
		case "Stable":
			stable++
		case "Improving":
			improving++
		}
	}

	summary := map[string]interface{}{
		"total_loans":       len(loans),
		"total_amount":      totalAmount,
		"total_outstanding": totalOutstanding,
		"worsening":         worsening,
		"stable":            stable,
		"improving":         improving,
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Status: "success",
		Data:   summary,
	})
}

// GetAllLoans handles GET /api/v1/loans
// @Summary Get all loans
// @Description Get list of all loans with filtering, sorting, and pagination
// @Tags Loans
// @Accept json
// @Produce json
// @Param officer_id query string false "Filter by officer ID"
// @Param branch query string false "Filter by branch"
// @Param region query string false "Filter by region"
// @Param channel query string false "Filter by channel"
// @Param status query string false "Filter by normalized status"
// @Param django_status query string false "Filter by raw Django status (comma-separated list; use __MISSING__ for missing)"
// @Param customer_phone query string false "Filter by customer phone (partial match)"
// @Param sort_by query string false "Sort field"
// @Param sort_dir query string false "Sort direction (asc/desc)"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(50)
// @Success 200 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /loans [get]
func (h *DashboardHandler) GetAllLoans(c *gin.Context) {
	// Parse filters
	filters := make(map[string]interface{})

	if officerID := c.Query("officer_id"); officerID != "" {
		filters["officer_id"] = officerID
	}
	if branch := c.Query("branch"); branch != "" {
		filters["branch"] = branch
	}
	if region := c.Query("region"); region != "" {
		filters["region"] = region
	}
	if channel := c.Query("channel"); channel != "" {
		filters["channel"] = channel
	}
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if djangoStatus := c.Query("django_status"); djangoStatus != "" {
		filters["django_status"] = djangoStatus
	}
	if performanceStatus := c.Query("performance_status"); performanceStatus != "" {
		filters["performance_status"] = performanceStatus
	}
	if wave := c.Query("wave"); wave != "" {
		filters["wave"] = wave
	}
	if customerPhone := c.Query("customer_phone"); customerPhone != "" {
		filters["customer_phone"] = customerPhone
	}
	if verticalLeadEmail := c.Query("vertical_lead_email"); verticalLeadEmail != "" {
		filters["vertical_lead_email"] = verticalLeadEmail
	}
	if loanType := c.Query("loan_type"); loanType != "" {
		filters["loan_type"] = loanType
	}
	if verificationStatus := c.Query("verification_status"); verificationStatus != "" {
		filters["verification_status"] = verificationStatus
	}
	// Optional period filter used primarily by Collections Control Centre.
	// Supported values (for now): today, this_week, this_month, last_month.
	if period := c.Query("period"); period != "" {
		filters["period"] = period
	}
	// Behavior-based filters used by All Loans UI (implemented server-side so
	// dashboard totals and CSV exports stay consistent)
	if behaviorLoanType := c.Query("behavior_loan_type"); behaviorLoanType != "" {
		filters["behavior_loan_type"] = behaviorLoanType
	}
	if rotType := c.Query("rot_type"); rotType != "" {
		filters["rot_type"] = rotType
	}
	if delayType := c.Query("delay_type"); delayType != "" {
		filters["delay_type"] = delayType
	}
	if dpdMin := c.Query("dpd_min"); dpdMin != "" {
		if min, err := strconv.Atoi(dpdMin); err == nil {
			filters["dpd_min"] = min
		}
	}
	if dpdMax := c.Query("dpd_max"); dpdMax != "" {
		if max, err := strconv.Atoi(dpdMax); err == nil {
			filters["dpd_max"] = max
		}
	}
	if sortBy := c.Query("sort_by"); sortBy != "" {
		filters["sort_by"] = sortBy
	}
	if sortDir := c.Query("sort_dir"); sortDir != "" {
		filters["sort_dir"] = sortDir
	}

	// Parse pagination
	page := 1
	limit := 50
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	filters["page"] = page
	filters["limit"] = limit

	loans, total, err := h.dashboardRepo.GetAllLoans(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to retrieve loans",
			Error:   newAPIError("INTERNAL_ERROR", err.Error()),
		})
		return
	}

	// Calculate summary metrics for all filtered loans (not just current page)
	summaryMetrics, err := h.dashboardRepo.GetLoansSummaryMetrics(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to calculate summary metrics",
			Error:   newAPIError("INTERNAL_ERROR", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"loans":           loans,
			"total":           total,
			"page":            page,
			"limit":           limit,
			"pages":           (total + limit - 1) / limit,
			"summary_metrics": summaryMetrics,
		},
	})
}

// GetBranchCollectionsLeaderboard handles GET /api/v1/collections/branches
// It provides the data needed for the Collections Control Centre "Branch
// Leaderboard" table – per-branch portfolio, expected due today, collections
// today, collection rates and a simple NPL proxy with status banding.
//
// @Summary Get branch collections leaderboard
// @Description Get per-branch collections metrics for the Branch Leaderboard table
// @Tags Collections
// @Accept json
// @Produce json
// @Param branch query string false "Filter by branch"
// @Param region query string false "Filter by region (supports comma-separated multi-select)"
// @Param channel query string false "Filter by channel"
// @Param wave query string false "Filter by wave"
// @Param loan_type query string false "Filter by loan type (supports comma-separated multi-select)"
// @Success 200 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /collections/branches [get]
func (h *DashboardHandler) GetBranchCollectionsLeaderboard(c *gin.Context) {
	filters := make(map[string]interface{})

	if branch := c.Query("branch"); branch != "" {
		filters["branch"] = branch
	}
	if region := c.Query("region"); region != "" {
		filters["region"] = region
	}
	if channel := c.Query("channel"); channel != "" {
		filters["channel"] = channel
	}
	if wave := c.Query("wave"); wave != "" {
		filters["wave"] = wave
	}
	if loanType := c.Query("loan_type"); loanType != "" {
		filters["loan_type"] = loanType
	}
	if djangoStatus := c.Query("django_status"); djangoStatus != "" {
		filters["django_status"] = djangoStatus
	}

	branches, err := h.dashboardRepo.GetBranchCollectionsLeaderboard(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to retrieve branch collections leaderboard",
			Error:   newAPIError("INTERNAL_ERROR", err.Error()),
		})
		return
	}

	// Simple summary rollup to make it easy for future aggregate cards if needed.
	var totalPortfolio, totalDueToday, totalCollectedToday, totalMissedToday float64
	for _, b := range branches {
		totalPortfolio += b.PortfolioTotal
		totalDueToday += b.DueToday
		totalCollectedToday += b.CollectedToday
		totalMissedToday += b.MissedToday
	}

	var collectionRate float64
	if totalDueToday > 0 {
		collectionRate = totalCollectedToday / totalDueToday
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"branches": branches,
			"summary": map[string]interface{}{
				"total_branches":        len(branches),
				"total_portfolio":       totalPortfolio,
				"total_due_today":       totalDueToday,
				"total_collected_today": totalCollectedToday,
				"total_missed_today":    totalMissedToday,
				"collection_rate_today": collectionRate,
			},
		},
	})
}

// GetOfficerCollectionsLeaderboard handles GET /api/v1/collections/officers
// It provides per-officer collections metrics (portfolio, due today, collections
// today, collection rates and NPL proxy) for Agent/Officer Leaderboard views.
//
// @Summary Get officer collections leaderboard
// @Description Get per-officer collections metrics for the Agent Leaderboard table
// @Tags Collections
// @Accept json
// @Produce json
// @Param branch query string false "Filter by branch"
// @Param region query string false "Filter by region (supports comma-separated multi-select)"
// @Param channel query string false "Filter by channel"
// @Param wave query string false "Filter by wave"
// @Param loan_type query string false "Filter by loan type (supports comma-separated multi-select)"
// @Success 200 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /collections/officers [get]
func (h *DashboardHandler) GetOfficerCollectionsLeaderboard(c *gin.Context) {
	filters := make(map[string]interface{})

	if branch := c.Query("branch"); branch != "" {
		filters["branch"] = branch
	}
	if region := c.Query("region"); region != "" {
		filters["region"] = region
	}
	if channel := c.Query("channel"); channel != "" {
		filters["channel"] = channel
	}
	if wave := c.Query("wave"); wave != "" {
		filters["wave"] = wave
	}
	if loanType := c.Query("loan_type"); loanType != "" {
		filters["loan_type"] = loanType
	}
	if djangoStatus := c.Query("django_status"); djangoStatus != "" {
		filters["django_status"] = djangoStatus
	}

	officers, err := h.dashboardRepo.GetOfficerCollectionsLeaderboard(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to retrieve officer collections leaderboard",
			Error:   newAPIError("INTERNAL_ERROR", err.Error()),
		})
		return
	}

	var totalPortfolio, totalDueToday, totalCollectedToday, totalMissedToday float64
	for _, o := range officers {
		totalPortfolio += o.PortfolioTotal
		totalDueToday += o.DueToday
		totalCollectedToday += o.CollectedToday
		totalMissedToday += o.MissedToday
	}

	var collectionRate float64
	if totalDueToday > 0 {
		collectionRate = totalCollectedToday / totalDueToday
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"officers": officers,
			"summary": map[string]interface{}{
				"total_officers":        len(officers),
				"total_portfolio":       totalPortfolio,
				"total_due_today":       totalDueToday,
				"total_collected_today": totalCollectedToday,
				"total_missed_today":    totalMissedToday,
				"collection_rate_today": collectionRate,
			},
		},
	})
}

// GetDailyCollections handles GET /api/v1/collections/daily
// It returns a per-day time series of collections amounts suitable for the
// Collections Control Centre daily chart.
//
// @Summary Get daily collections time series
// @Description Get per-day collections amounts for the selected period and filters
// @Tags Collections
// @Accept json
// @Produce json
// @Param period query string false "Period (today, this_week, this_month, last_month)"
// @Param branch query string false "Filter by branch"
// @Param region query string false "Filter by region (supports comma-separated multi-select)"
// @Param channel query string false "Filter by channel"
// @Param wave query string false "Filter by wave"
// @Param loan_type query string false "Filter by loan type (supports comma-separated multi-select)"
// @Success 200 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /collections/daily [get]
func (h *DashboardHandler) GetDailyCollections(c *gin.Context) {
	filters := make(map[string]interface{})

	if period := c.Query("period"); period != "" {
		filters["period"] = period
	}
	if branch := c.Query("branch"); branch != "" {
		filters["branch"] = branch
	}
	if region := c.Query("region"); region != "" {
		filters["region"] = region
	}
	if channel := c.Query("channel"); channel != "" {
		filters["channel"] = channel
	}
	if wave := c.Query("wave"); wave != "" {
		filters["wave"] = wave
	}
	if loanType := c.Query("loan_type"); loanType != "" {
		filters["loan_type"] = loanType
	}

	points, err := h.dashboardRepo.GetDailyCollections(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to retrieve daily collections",
			Error:   newAPIError("INTERNAL_ERROR", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"points": points,
		},
	})
}

// GetBranches handles GET /api/v1/branches
// @Summary Get all branches
// @Description Get list of branches with their portfolio metrics and PAR15 ratios
// @Tags Branches
// @Accept json
// @Produce json
// @Param branch query string false "Filter by branch"
// @Param region query string false "Filter by region"
// @Param channel query string false "Filter by channel"
// @Param user_type query string false "Filter by user type"
// @Param wave query string false "Filter by wave"
// @Param sort_by query string false "Sort field"
// @Param sort_dir query string false "Sort direction (asc/desc)"
// @Success 200 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /branches [get]
func (h *DashboardHandler) GetBranches(c *gin.Context) {
	// Parse filters
	filters := make(map[string]interface{})

	if branch := c.Query("branch"); branch != "" {
		filters["branch"] = branch
	}
	if region := c.Query("region"); region != "" {
		filters["region"] = region
	}
	if channel := c.Query("channel"); channel != "" {
		filters["channel"] = channel
	}
	if userType := c.Query("user_type"); userType != "" {
		filters["user_type"] = userType
	}
	if wave := c.Query("wave"); wave != "" {
		filters["wave"] = wave
	}
	if sortBy := c.Query("sort_by"); sortBy != "" {
		filters["sort_by"] = sortBy
	}
	if sortDir := c.Query("sort_dir"); sortDir != "" {
		filters["sort_dir"] = sortDir
	}

	branches, err := h.dashboardRepo.GetBranches(filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to retrieve branches",
			Error:   newAPIError("INTERNAL_ERROR", err.Error()),
		})
		return
	}

	// Calculate summary
	var totalPortfolio, totalOverdue15d float64
	for _, branch := range branches {
		totalPortfolio += branch.PortfolioTotal
		totalOverdue15d += branch.Overdue15d
	}

	avgPar15 := 0.0
	if totalPortfolio > 0 {
		avgPar15 = totalOverdue15d / totalPortfolio
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"branches": branches,
			"summary": map[string]interface{}{
				"total_branches":    len(branches),
				"total_portfolio":   totalPortfolio,
				"total_overdue_15d": totalOverdue15d,
				"avg_par15_ratio":   avgPar15,
			},
		},
	})
}

// GetFilterOptions handles GET /api/v1/filters/:type
func (h *DashboardHandler) GetFilterOptions(c *gin.Context) {
	filterType := c.Param("type")

	// Parse additional filters
	filters := make(map[string]interface{})
	if region := c.Query("region"); region != "" {
		filters["region"] = region
	}
	if branch := c.Query("branch"); branch != "" {
		filters["branch"] = branch
	}

	options, err := h.dashboardRepo.GetFilterOptions(filterType, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to retrieve filter options",
			Error:   newAPIError("INTERNAL_ERROR", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			filterType: options,
		},
	})
}

// GetTeamMembers handles GET /api/v1/team-members
func (h *DashboardHandler) GetTeamMembers(c *gin.Context) {
	members, err := h.dashboardRepo.GetTeamMembers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to retrieve team members",
			Error:   newAPIError("INTERNAL_ERROR", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"team_members": members,
		},
	})
}

// UpdateOfficerAudit handles PUT /api/v1/officers/:officer_id/audit
func (h *DashboardHandler) UpdateOfficerAudit(c *gin.Context) {
	officerID := c.Param("officer_id")

	var update models.AuditUpdate
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Status:  "error",
			Message: "Invalid request body",
			Error:   newAPIError("INTERNAL_ERROR", err.Error()),
		})
		return
	}

	err := h.dashboardRepo.UpdateOfficerAudit(officerID, &update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to update audit assignment",
			Error:   newAPIError("INTERNAL_ERROR", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Status:  "success",
		Message: "Audit assignment updated successfully",
		Data: map[string]interface{}{
			"officer_id":    officerID,
			"assignee_id":   update.AssigneeID,
			"assignee_name": update.AssigneeName,
			"audit_status":  update.AuditStatus,
		},
	})
}

// GetOfficerAuditHistory handles GET /api/v1/officers/:officer_id/audit-history
func (h *DashboardHandler) GetOfficerAuditHistory(c *gin.Context) {
	officerID := c.Param("officer_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	history, err := h.dashboardRepo.GetOfficerAuditHistory(officerID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to retrieve audit history",
			Error:   newAPIError("INTERNAL_ERROR", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"audit_history": history,
		},
	})
}

// GetTopRiskLoans handles GET /api/v1/officers/:officer_id/top-risk-loans
func (h *DashboardHandler) GetTopRiskLoans(c *gin.Context) {
	officerID := c.Param("officer_id")
	if officerID == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Status:  "error",
			Message: "Officer ID is required",
			Error:   newAPIError("INVALID_OFFICER_ID", "Officer ID parameter is missing"),
		})
		return
	}

	// Parse limit parameter (default to 20)
	limit := 20
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Fetch top risk loans from repository
	loans, err := h.dashboardRepo.GetTopRiskLoans(officerID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to retrieve top risk loans",
			Error:   newAPIError("TOP_RISK_LOANS_ERROR", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"officer_id": officerID,
			"limit":      limit,
			"count":      len(loans),
			"loans":      loans,
		},
	})
}

// GetLoanRepayments handles GET /api/v1/loans/:loan_id/repayments
func (h *DashboardHandler) GetLoanRepayments(c *gin.Context) {
	loanID := c.Param("loan_id")

	// Fetch repayments for the loan
	repayments, err := h.repaymentRepo.GetByLoanID(c.Request.Context(), loanID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to retrieve loan repayments",
			Error:   newAPIError("LOAN_REPAYMENTS_ERROR", err.Error()),
		})
		return
	}

	// Calculate running balance for each repayment
	type RepaymentWithBalance struct {
		*models.Repayment
		BalanceAfterPayment float64 `json:"balance_after_payment"`
	}

	repaymentsWithBalance := make([]RepaymentWithBalance, len(repayments))

	// Note: We're showing repayments in DESC order (most recent first)
	// To calculate running balance, we'd need the loan's original amount and all previous payments
	// For now, we'll just return the repayments without calculated balance
	// The frontend can display the payment breakdown instead

	for i, r := range repayments {
		repaymentsWithBalance[i] = RepaymentWithBalance{
			Repayment:           r,
			BalanceAfterPayment: 0, // TODO: Calculate if needed
		}
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Status: "success",
		Data: map[string]interface{}{
			"loan_id":    loanID,
			"count":      len(repayments),
			"repayments": repayments,
		},
	})
}

// RecalculateAllLoanFields handles POST /api/v1/loans/recalculate-fields
// @Summary Recalculate all loan computed fields
// @Description Manually trigger recalculation of all computed fields (actual_outstanding, total_outstanding, current_dpd, etc.) for all loans. This operation runs asynchronously.
// @Tags Loans
// @Accept json
// @Produce json
// @Success 202 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /loans/recalculate-fields [post]
func (h *DashboardHandler) RecalculateAllLoanFields(c *gin.Context) {
	// Run recalculation asynchronously to avoid timeout
	go func() {
		log.Println("🔄 Starting loan fields recalculation...")
		rowsAffected, err := h.dashboardRepo.RecalculateAllLoanFields()
		if err != nil {
			log.Printf("❌ Failed to recalculate loan fields: %v", err)
			return
		}
		log.Printf("✅ Successfully recalculated %d loans", rowsAffected)
	}()

	// Return immediately with 202 Accepted
	c.JSON(http.StatusAccepted, models.APIResponse{
		Status:  "success",
		Message: "Loan field recalculation started. This process will run in the background and may take several minutes to complete.",
		Data: map[string]interface{}{
			"status": "processing",
		},
	})
}

// SyncLoanRepayments handles POST /api/v1/loans/:loan_id/sync-repayments
// @Summary Sync repayments for a specific loan
// @Description Syncs repayment data for a single loan from Django source database to SeedsMetrics
// @Tags Loans
// @Accept json
// @Produce json
// @Param loan_id path string true "Loan ID"
// @Success 200 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /loans/{loan_id}/sync-repayments [post]
func (h *DashboardHandler) SyncLoanRepayments(c *gin.Context) {
	loanID := c.Param("loan_id")

	if loanID == "" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Status:  "error",
			Message: "Loan ID is required",
			Error:   newAPIError("INVALID_REQUEST", "loan_id parameter is missing"),
		})
		return
	}

	log.Printf("🔄 API: Starting repayment sync for loan %s", loanID)

	// Call sync service
	result, err := h.syncService.SyncLoanRepayments(c.Request.Context(), loanID)
	if err != nil {
		// Check if it's a "not found" error
		if err.Error() == "loan "+loanID+" not found" {
			c.JSON(http.StatusNotFound, models.APIResponse{
				Status:  "error",
				Message: fmt.Sprintf("Loan %s not found", loanID),
				Error:   newAPIError("LOAN_NOT_FOUND", err.Error()),
			})
			return
		}

		log.Printf("❌ Failed to sync repayments for loan %s: %v", loanID, err)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to sync repayments",
			Error:   newAPIError("SYNC_ERROR", err.Error()),
		})
		return
	}

	log.Printf("✅ Successfully synced repayments for loan %s: %d synced, %d errors", loanID, result.TotalSynced, result.TotalErrors)

	c.JSON(http.StatusOK, models.APIResponse{
		Status:  "success",
		Message: result.Message,
		Data: map[string]interface{}{
			"loan_id":      result.LoanID,
			"total_synced": result.TotalSynced,
			"total_errors": result.TotalErrors,
			"updated_loan": result.UpdatedLoan,
		},
	})
}

// UpdatePastMaturityStatus handles POST /api/v1/loans/update-past-maturity
// @Summary Update past maturity loan statuses
// @Description Updates django_status to 'PAST_MATURITY' for all loans where current date exceeds maturity_date
// @Tags Loans
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /loans/update-past-maturity [post]
func (h *DashboardHandler) UpdatePastMaturityStatus(c *gin.Context) {
	log.Println("📅 Updating past maturity loan statuses...")

	rowsUpdated, err := h.dashboardRepo.UpdatePastMaturityStatus()
	if err != nil {
		log.Printf("❌ Error updating past maturity status: %v", err)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to update past maturity statuses",
			Error:   newAPIError("UPDATE_FAILED", err.Error()),
		})
		return
	}

	log.Printf("✅ Updated %d loans to PAST_MATURITY status", rowsUpdated)

	c.JSON(http.StatusOK, models.APIResponse{
		Status:  "success",
		Message: fmt.Sprintf("Updated %d loans to PAST_MATURITY status", rowsUpdated),
		Data: map[string]interface{}{
			"loans_updated": rowsUpdated,
		},
	})
}

// SyncNewRepayments handles POST /api/v1/sync/repayments
// @Summary Sync new repayments incrementally
// @Description Syncs only new repayments from Django (where ID > max existing ID). Much faster than full sync.
// @Tags Sync
// @Accept json
// @Produce json
// @Success 200 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /sync/repayments [post]
func (h *DashboardHandler) SyncNewRepayments(c *gin.Context) {
	log.Println("🔄 Starting incremental repayment sync...")

	result, err := h.syncService.SyncNewRepayments(c.Request.Context())
	if err != nil {
		log.Printf("❌ Error syncing new repayments: %v", err)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to sync new repayments",
			Error:   newAPIError("SYNC_ERROR", err.Error()),
		})
		return
	}

	log.Printf("✅ Incremental sync complete: %d synced, %d errors", result.TotalSynced, result.TotalErrors)

	c.JSON(http.StatusOK, models.APIResponse{
		Status:  "success",
		Message: result.Message,
		Data: map[string]interface{}{
			"total_synced":    result.TotalSynced,
			"total_errors":    result.TotalErrors,
			"last_id_synced":  result.LastIDSynced,
			"previous_max_id": result.PreviousMaxID,
		},
	})
}
