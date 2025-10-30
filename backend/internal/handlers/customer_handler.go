package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/seeds-metrics/analytics-backend/internal/models"
	"github.com/seeds-metrics/analytics-backend/internal/repository"
)

type CustomerHandler struct {
	customerRepo *repository.CustomerRepository
}

func NewCustomerHandler(customerRepo *repository.CustomerRepository) *CustomerHandler {
	return &CustomerHandler{
		customerRepo: customerRepo,
	}
}

// CreateCustomer handles POST /api/v1/etl/customers
// @Summary Create a new customer
// @Description Create a new customer record in the system (ETL endpoint)
// @Tags ETL
// @Accept json
// @Produce json
// @Param customer body models.CustomerInput true "Customer data"
// @Success 201 {object} models.APIResponse
// @Failure 400 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /etl/customers [post]
func (h *CustomerHandler) CreateCustomer(c *gin.Context) {
	var input models.CustomerInput
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

	// Create customer
	if err := h.customerRepo.Create(c.Request.Context(), &input); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status: "error",
			Error: &models.APIError{
				Code:    "DATABASE_ERROR",
				Message: "Failed to create customer",
				Details: map[string]interface{}{"error": err.Error()},
			},
		})
		return
	}

	c.JSON(http.StatusCreated, models.APIResponse{
		Status:  "success",
		Message: "Customer created successfully",
		Data: map[string]interface{}{
			"customer_id": input.CustomerID,
		},
	})
}

// GetCustomers handles GET /api/v1/customers
// @Summary Get all customers
// @Description Retrieve a list of all customers
// @Tags Customers
// @Produce json
// @Success 200 {object} models.APIResponse
// @Failure 500 {object} models.APIResponse
// @Router /customers [get]
func (h *CustomerHandler) GetCustomers(c *gin.Context) {
	customers, err := h.customerRepo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Status: "error",
			Error: &models.APIError{
				Code:    "DATABASE_ERROR",
				Message: "Failed to retrieve customers",
				Details: map[string]interface{}{"error": err.Error()},
			},
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Status: "success",
		Data:   customers,
	})
}

