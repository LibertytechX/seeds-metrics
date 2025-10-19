package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/seeds-metrics/analytics-backend/internal/models"
	"github.com/seeds-metrics/analytics-backend/pkg/database"
)

type HealthHandler struct {
	db *database.DB
}

func NewHealthHandler(db *database.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// HealthCheck handles GET /health
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	services := make(map[string]models.ServiceHealth)

	// Check database
	dbStatus := "healthy"
	dbMessage := "Database connection is healthy"
	if err := h.db.HealthCheck(); err != nil {
		dbStatus = "unhealthy"
		dbMessage = err.Error()
	}
	services["database"] = models.ServiceHealth{
		Status:  dbStatus,
		Message: dbMessage,
	}

	// Overall status
	overallStatus := "healthy"
	if dbStatus != "healthy" {
		overallStatus = "unhealthy"
	}

	response := models.HealthCheckResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Services:  services,
	}

	statusCode := http.StatusOK
	if overallStatus != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, response)
}

