package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/seeds-metrics/analytics-backend/internal/models"
	"github.com/seeds-metrics/analytics-backend/internal/repository"
	"github.com/shopspring/decimal"
)

type ETLHandler struct {
	loanRepo      *repository.LoanRepository
	repaymentRepo *repository.RepaymentRepository
	officerRepo   *repository.OfficerRepository
}

func NewETLHandler(loanRepo *repository.LoanRepository, repaymentRepo *repository.RepaymentRepository, officerRepo *repository.OfficerRepository) *ETLHandler {
	return &ETLHandler{
		loanRepo:      loanRepo,
		repaymentRepo: repaymentRepo,
		officerRepo:   officerRepo,
	}
}

// CreateLoan handles POST /api/v1/etl/loans
// @Summary Create a new loan
// @Description Create a new loan record in the system (ETL endpoint)
// @Tags ETL
// @Accept json
// @Produce json
// @Param loan body models.LoanInput true "Loan data"
// @Success 201 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /etl/loans [post]
func (h *ETLHandler) CreateLoan(c *gin.Context) {
	var input models.LoanInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Status: "error",
			Error: &models.APIError{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request payload",
				Details: map[string]interface{}{"error": err.Error()},
			},
		})
		return
	}

	// Convert interest rate from percentage to decimal (e.g., 15 -> 0.15)
	if input.InterestRate != nil {
		percentageRate := *input.InterestRate
		decimalRate := percentageRate.Div(decimal.NewFromInt(100))
		input.InterestRate = &decimalRate
	}

	// Create loan
	if err := h.loanRepo.Create(c.Request.Context(), &input); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status: "error",
			Error: &models.APIError{
				Code:    "DATABASE_ERROR",
				Message: "Failed to create loan",
				Details: map[string]interface{}{"error": err.Error()},
			},
		})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{
		Status:  "success",
		Message: "Loan created successfully",
		Data: map[string]interface{}{
			"loan_id": input.LoanID,
		},
	})
}

// CreateRepayment handles POST /api/v1/etl/repayments
// @Summary Create a new repayment
// @Description Create a new repayment record for a loan (ETL endpoint)
// @Tags ETL
// @Accept json
// @Produce json
// @Param repayment body models.RepaymentInput true "Repayment data"
// @Success 201 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 404 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /etl/repayments [post]
func (h *ETLHandler) CreateRepayment(c *gin.Context) {
	var input models.RepaymentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Status: "error",
			Error: &models.APIError{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request payload",
				Details: map[string]interface{}{"error": err.Error()},
			},
		})
		return
	}

	// Verify loan exists
	loan, err := h.loanRepo.GetByID(c.Request.Context(), input.LoanID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status: "error",
			Error: &models.APIError{
				Code:    "DATABASE_ERROR",
				Message: "Failed to verify loan",
				Details: map[string]interface{}{"error": err.Error()},
			},
		})
		return
	}
	if loan == nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Status: "error",
			Error: &models.APIError{
				Code:    "LOAN_NOT_FOUND",
				Message: fmt.Sprintf("Loan with ID %s does not exist", input.LoanID),
			},
		})
		return
	}

	// Create repayment
	if err := h.repaymentRepo.Create(c.Request.Context(), &input); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status: "error",
			Error: &models.APIError{
				Code:    "DATABASE_ERROR",
				Message: "Failed to create repayment",
				Details: map[string]interface{}{"error": err.Error()},
			},
		})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{
		Status:  "success",
		Message: "Repayment created successfully. Loan computed fields will be updated automatically.",
		Data: map[string]interface{}{
			"repayment_id": input.RepaymentID,
			"loan_id":      input.LoanID,
		},
	})
}

// BatchSync handles POST /api/v1/etl/sync
func (h *ETLHandler) BatchSync(c *gin.Context) {
	var request models.ETLSyncRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Status: "error",
			Error: &models.APIError{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request payload",
				Details: map[string]interface{}{"error": err.Error()},
			},
		})
		return
	}

	startTime := time.Now()
	syncID := uuid.New().String()

	results := models.ETLSyncResults{
		Loans:      models.ETLEntityResult{},
		Repayments: models.ETLEntityResult{},
	}
	var errors []models.ETLSyncError

	// Process loans
	for _, loanInput := range request.Data.Loans {
		err := h.loanRepo.Create(c.Request.Context(), &loanInput)
		if err != nil {
			results.Loans.Failed++
			errors = append(errors, models.ETLSyncError{
				EntityType:   "loan",
				EntityID:     loanInput.LoanID,
				ErrorCode:    "CREATE_FAILED",
				ErrorMessage: err.Error(),
			})
		} else {
			results.Loans.Inserted++
		}
	}

	// Process repayments
	for _, repaymentInput := range request.Data.Repayments {
		err := h.repaymentRepo.Create(c.Request.Context(), &repaymentInput)
		if err != nil {
			results.Repayments.Failed++
			errors = append(errors, models.ETLSyncError{
				EntityType:   "repayment",
				EntityID:     repaymentInput.RepaymentID,
				ErrorCode:    "CREATE_FAILED",
				ErrorMessage: err.Error(),
			})
		} else {
			results.Repayments.Inserted++
		}
	}

	computationTime := time.Since(startTime).Milliseconds()

	// Determine status
	status := "success"
	if results.Loans.Failed > 0 || results.Repayments.Failed > 0 {
		if results.Loans.Inserted > 0 || results.Repayments.Inserted > 0 {
			status = "partial_success"
		} else {
			status = "error"
		}
	}

	// Calculate next sync time (15 minutes from now)
	nextSync := time.Now().Add(15 * time.Minute)

	response := models.ETLSyncResponse{
		Status:    status,
		SyncID:    syncID,
		Timestamp: time.Now(),
		Results:   results,
		ComputedFieldsUpdated: models.ComputedFieldsUpdate{
			LoansAffected:     results.Repayments.Inserted,
			ComputationTimeMs: int(computationTime),
		},
		NextSyncRecommended: nextSync,
		Errors:              errors,
	}

	statusCode := http.StatusOK
	if status == "partial_success" {
		statusCode = http.StatusMultiStatus
	} else if status == "error" {
		statusCode = http.StatusBadRequest
	}

	c.JSON(statusCode, response)
}

// CreateOfficer handles POST /api/v1/etl/officers
// @Summary Create a new officer
// @Description Create a new loan officer record in the system (ETL endpoint)
// @Tags ETL
// @Accept json
// @Produce json
// @Param officer body models.OfficerInput true "Officer data"
// @Success 201 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /etl/officers [post]
func (h *ETLHandler) CreateOfficer(c *gin.Context) {
	var input models.OfficerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Status: "error",
			Error: &models.APIError{
				Code:    "VALIDATION_ERROR",
				Message: "Invalid request payload",
				Details: map[string]interface{}{"error": err.Error()},
			},
		})
		return
	}

	// Create officer
	if err := h.officerRepo.Create(c.Request.Context(), &input); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status: "error",
			Error: &models.APIError{
				Code:    "DATABASE_ERROR",
				Message: "Failed to create officer",
				Details: map[string]interface{}{"error": err.Error()},
			},
		})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{
		Status:  "success",
		Message: "Officer created successfully",
		Data: map[string]interface{}{
			"officer_id": input.OfficerID,
		},
	})
}
