package handlers

import (
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/seeds-metrics/analytics-backend/internal/models"
	"github.com/seeds-metrics/analytics-backend/internal/repository"
	"github.com/seeds-metrics/analytics-backend/internal/services"
)

// CreditBureauHandler handles credit bureau API requests
type CreditBureauHandler struct {
	creditBureauRepo *repository.CreditBureauRepository
	syncService      *services.SyncService
}

// NewCreditBureauHandler creates a new credit bureau handler
func NewCreditBureauHandler(creditBureauRepo *repository.CreditBureauRepository, syncService *services.SyncService) *CreditBureauHandler {
	return &CreditBureauHandler{
		creditBureauRepo: creditBureauRepo,
		syncService:      syncService,
	}
}

// GetLoanCreditBureauKYC handles GET /api/v1/loans/credit-bureau-kyc
// @Summary Get paginated loan credit bureau KYC data
// @Description Returns denormalized loan + KYC + credit bureau records with pagination
// @Tags Loans
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Results per page" default(5)
// @Param source query string false "Filter by CB data source (credit_bureau_result, borrower_worthiness, credit_bureau_metadata)"
// @Param include_images query bool false "Include base64 images in response" default(true)
// @Success 200 {object} models.PaginatedResponse
// @Failure 500 {object} models.APIResponse
// @Router /loans/credit-bureau-kyc [get]
func (h *CreditBureauHandler) GetLoanCreditBureauKYC(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))
	source := c.Query("source")
	includeImages := c.DefaultQuery("include_images", "true") == "true"

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 5
	}

	// Get total count
	total, err := h.creditBureauRepo.GetLoanCreditBureauKYCCount(c.Request.Context(), source)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to count records",
			Error:   newAPIError("COUNT_ERROR", err.Error()),
		})
		return
	}

	// Get paginated records
	records, err := h.creditBureauRepo.GetLoanCreditBureauKYCPaginated(c.Request.Context(), page, limit, source, includeImages)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to fetch records",
			Error:   newAPIError("FETCH_ERROR", err.Error()),
		})
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	c.JSON(http.StatusOK, models.PaginatedResponse{
		Status: "success",
		Data:   records,
		Pagination: models.Pagination{
			Total:      total,
			Page:       page,
			PageSize:   limit,
			TotalPages: totalPages,
		},
	})
}

// SyncLoanCreditBureauKYC handles POST /api/v1/loans/credit-bureau-kyc/sync
// @Summary Trigger loan credit bureau KYC sync
// @Description Manually triggers a sync of loan + KYC + credit bureau data from Django
// @Tags Loans
// @Produce json
// @Success 200 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /loans/credit-bureau-kyc/sync [post]
func (h *CreditBureauHandler) SyncLoanCreditBureauKYC(c *gin.Context) {
	result, err := h.syncService.SyncLoanCreditBureauKYC(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status:  "error",
			Message: "Failed to sync loan credit bureau KYC data",
			Error:   newAPIError("SYNC_ERROR", err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Status:  "success",
		Message: result.Message,
		Data:    result,
	})
}
