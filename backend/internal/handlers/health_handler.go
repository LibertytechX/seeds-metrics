package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/seeds-metrics/analytics-backend/internal/models"
	"github.com/seeds-metrics/analytics-backend/internal/repository"
	"github.com/seeds-metrics/analytics-backend/pkg/database"
)

type HealthHandler struct {
	db         *database.DB
	djangoRepo *repository.DjangoRepository
}

func NewHealthHandler(db *database.DB, djangoRepo *repository.DjangoRepository) *HealthHandler {
	return &HealthHandler{
		db:         db,
		djangoRepo: djangoRepo,
	}
}

// HealthCheck handles GET /health
// @Summary Health check
// @Description Check the health status of the API and its dependencies (database)
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} models.HealthCheckResponse
// @Failure 503 {object} models.HealthCheckResponse
// @Router /health [get]
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	services := make(map[string]models.ServiceHealth)

	// Check SeedsMetrics database
	dbStatus := "healthy"
	dbMessage := "SeedsMetrics database connection is healthy"
	if err := h.db.HealthCheck(); err != nil {
		dbStatus = "unhealthy"
		dbMessage = err.Error()
	}
	services["seedsmetrics_database"] = models.ServiceHealth{
		Status:  dbStatus,
		Message: dbMessage,
	}

	// Check Django database
	djangoStatus := "healthy"
	djangoMessage := "Django database connection is healthy"
	if h.djangoRepo != nil {
		if err := h.djangoRepo.HealthCheck(c.Request.Context()); err != nil {
			djangoStatus = "unhealthy"
			djangoMessage = err.Error()
		}
	} else {
		djangoStatus = "not_configured"
		djangoMessage = "Django database not configured"
	}
	services["django_database"] = models.ServiceHealth{
		Status:  djangoStatus,
		Message: djangoMessage,
	}

	// Overall status
	overallStatus := "healthy"
	if dbStatus != "healthy" || djangoStatus != "healthy" {
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
