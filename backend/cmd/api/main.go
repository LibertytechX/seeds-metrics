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
	"github.com/seeds-metrics/analytics-backend/pkg/database"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set Gin mode
	gin.SetMode(cfg.Server.GinMode)

	// Initialize database
	db, err := database.NewPostgresDB(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	log.Println("✅ Database connection established")

	// Initialize repositories
	loanRepo := repository.NewLoanRepository(db)
	repaymentRepo := repository.NewRepaymentRepository(db)

	// Initialize handlers
	etlHandler := handlers.NewETLHandler(loanRepo, repaymentRepo)
	healthHandler := handlers.NewHealthHandler(db)

	// Setup router
	router := setupRouter(cfg, etlHandler, healthHandler)

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

func setupRouter(cfg *config.Config, etlHandler *handlers.ETLHandler, healthHandler *handlers.HealthHandler) *gin.Engine {
	router := gin.Default()

	// CORS middleware
	router.Use(corsMiddleware(cfg))

	// Health check
	router.GET("/health", healthHandler.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// ETL endpoints
		etl := v1.Group("/etl")
		{
			etl.POST("/loans", etlHandler.CreateLoan)
			etl.POST("/repayments", etlHandler.CreateRepayment)
			etl.POST("/sync", etlHandler.BatchSync)
		}
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

