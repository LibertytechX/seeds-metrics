package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/seeds-metrics/analytics-backend/internal/config"
	"github.com/seeds-metrics/analytics-backend/internal/handlers"
	"github.com/seeds-metrics/analytics-backend/internal/repository"
	"github.com/seeds-metrics/analytics-backend/internal/services"
	"github.com/seeds-metrics/analytics-backend/pkg/database"

	_ "github.com/seeds-metrics/analytics-backend/docs" // Import generated docs
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Seeds Metrics API
// @version 1.0
// @description API for Seeds & Pennies loan portfolio metrics and analytics dashboard
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@seedsandpennies.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8081
// @BasePath /api/v1
// @schemes http https

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set Gin mode
	gin.SetMode(cfg.Server.GinMode)

	// Initialize SeedsMetrics database (read-write)
	db, err := database.NewPostgresDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to SeedsMetrics database: %v", err)
	}
	defer db.Close()

	log.Println("✅ SeedsMetrics database connection established")

	// Initialize Django database (read-only)
	djangoDB, err := database.NewPostgresDB(&cfg.DjangoDatabase)
	if err != nil {
		log.Fatalf("Failed to connect to Django database: %v", err)
	}
	defer djangoDB.Close()

	log.Println("✅ Django database connection established")

	// Initialize repositories
	loanRepo := repository.NewLoanRepository(db)
	repaymentRepo := repository.NewRepaymentRepository(db)
	officerRepo := repository.NewOfficerRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	dashboardRepo := repository.NewDashboardRepository(db.DB)

	// Initialize Django repository (read-only access to source data)
	djangoRepo := repository.NewDjangoRepository(djangoDB.DB)

	// Initialize services
	metricsService := services.NewMetricsService()
	syncService := services.NewSyncService(djangoDB.DB, db)

	// Initialize handlers
	etlHandler := handlers.NewETLHandler(loanRepo, repaymentRepo, officerRepo)
	customerHandler := handlers.NewCustomerHandler(customerRepo)
	healthHandler := handlers.NewHealthHandler(db, djangoRepo)
	dashboardHandler := handlers.NewDashboardHandler(dashboardRepo, repaymentRepo, metricsService, syncService)

	// Setup router
	router := setupRouter(cfg, etlHandler, customerHandler, healthHandler, dashboardHandler)

	// Start server
	addr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)
	log.Printf("🚀 Server starting on %s", addr)

	// Graceful shutdown
	go func() {
		if err := router.Run(addr); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")
}

func setupRouter(cfg *config.Config, etlHandler *handlers.ETLHandler, customerHandler *handlers.CustomerHandler, healthHandler *handlers.HealthHandler, dashboardHandler *handlers.DashboardHandler) *gin.Engine {
	router := gin.Default()

	// CORS middleware
	router.Use(corsMiddleware(cfg))

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check
	router.GET("/health", healthHandler.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// ETL endpoints
		etl := v1.Group("/etl")
		{
			etl.POST("/customers", customerHandler.CreateCustomer)
			etl.POST("/officers", etlHandler.CreateOfficer)
			etl.POST("/loans", etlHandler.CreateLoan)
			etl.POST("/repayments", etlHandler.CreateRepayment)
			etl.POST("/sync", etlHandler.BatchSync)
		}

		// Customer endpoints
		v1.GET("/customers", customerHandler.GetCustomers)

		// Portfolio metrics
		metrics := v1.Group("/metrics")
		{
			metrics.GET("/portfolio", dashboardHandler.GetPortfolioMetrics)
		}

		// Collections endpoints
		collections := v1.Group("/collections")
		{
			collections.GET("/branches", dashboardHandler.GetBranchCollectionsLeaderboard)
			collections.GET("/officers", dashboardHandler.GetOfficerCollectionsLeaderboard)
			collections.GET("/daily", dashboardHandler.GetDailyCollections)
		}

		// Officer endpoints
		officers := v1.Group("/officers")
		{
			officers.GET("", dashboardHandler.GetOfficers)
			officers.GET("/:officer_id", dashboardHandler.GetOfficerByID)
			officers.PUT("/:officer_id/audit", dashboardHandler.UpdateOfficerAudit)
			officers.GET("/:officer_id/audit-history", dashboardHandler.GetOfficerAuditHistory)
			officers.GET("/:officer_id/top-risk-loans", dashboardHandler.GetTopRiskLoans)
		}

		// FIMR endpoints
		fimr := v1.Group("/fimr")
		{
			fimr.GET("/loans", dashboardHandler.GetFIMRLoans)
			fimr.GET("/summary", dashboardHandler.GetFIMRSummary)
		}

		// Early indicators endpoints
		earlyIndicators := v1.Group("/early-indicators")
		{
			earlyIndicators.GET("/loans", dashboardHandler.GetEarlyIndicatorLoans)
			earlyIndicators.GET("/summary", dashboardHandler.GetEarlyIndicatorSummary)
		}

		// Branch endpoints
		branches := v1.Group("/branches")
		{
			branches.GET("", dashboardHandler.GetBranches)
		}

		// Loans endpoints
		loans := v1.Group("/loans")
		{
			loans.GET("", dashboardHandler.GetAllLoans)
			loans.GET("/:loan_id/repayments", dashboardHandler.GetLoanRepayments)
			loans.POST("/recalculate-fields", dashboardHandler.RecalculateAllLoanFields)
			loans.POST("/update-past-maturity", dashboardHandler.UpdatePastMaturityStatus)
			loans.POST("/:loan_id/sync-repayments", dashboardHandler.SyncLoanRepayments)
		}

		// Sync endpoints
		sync := v1.Group("/sync")
		{
			sync.POST("/repayments", dashboardHandler.SyncNewRepayments)
		}

		// Filter endpoints
		filters := v1.Group("/filters")
		{
			filters.GET("/:type", dashboardHandler.GetFilterOptions)
		}

		// Team management
		v1.GET("/team-members", dashboardHandler.GetTeamMembers)
	}

	return router
}

func corsMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
