package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dhima/event-trigger-platform/internal/api/handlers"
	"github.com/dhima/event-trigger-platform/internal/api/middleware"
	"github.com/dhima/event-trigger-platform/internal/logging"
	"github.com/dhima/event-trigger-platform/internal/storage"
	"github.com/dhima/event-trigger-platform/internal/triggers"
	"github.com/dhima/event-trigger-platform/pkg/config"
	"github.com/gin-contrib/cors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

// Server orchestrates HTTP routing and dependencies for the API service.
type Server struct {
	config config.App
	logger logging.Logger
	router *gin.Engine
	db     *sql.DB

	triggerService *triggers.Service
}

// NewServer wires the API dependencies together.
func NewServer() *Server {
	cfg := config.FromEnv()

	// Initialize logger
	logger, err := logging.NewLogger(cfg.Environment, cfg.LogLevel)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize logger: %v", err))
	}

	// Set Gin mode based on environment
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	db := connectDatabase(cfg, logger)
	mysqlClient := storage.NewMySQLClient(db)

	server := &Server{
		config:         cfg,
		logger:         logger,
		db:             db,
		triggerService: triggers.NewService(mysqlClient),
	}

	server.setupRouter()
	return server
}

// setupRouter configures the Gin router with middleware and routes.
func (s *Server) setupRouter() {
	router := gin.New()

	// Get underlying zap logger for gin-contrib/zap middleware
	zapLogger := s.getZapLogger()

	// Global middleware (order matters!)
	// 1. Recovery - must be first to catch panics from other middleware
	router.Use(ginzap.RecoveryWithZap(zapLogger, true))

	// 2. Request ID - inject unique ID for tracing
	router.Use(middleware.RequestID())

	// 3. Logging - log all requests with structured fields
	router.Use(ginzap.Ginzap(zapLogger, time.RFC3339, true))

	// 4. CORS - handle cross-origin requests
	router.Use(cors.New(cors.Config{
		AllowOrigins:     s.config.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID", "X-API-Key"},
		ExposeHeaders:    []string{"Content-Length", "X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Health and metrics endpoints (no /api/v1 prefix)
	router.GET("/health", handlers.NewHealthHandler(s.logger).Health)
	router.GET("/metrics", handlers.NewMetricsHandler(s.logger).Metrics)

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Trigger management
		triggerHandler := handlers.NewTriggerHandler(s.logger, s.triggerService)
		triggers := v1.Group("/triggers")
		{
			triggers.POST("", triggerHandler.CreateTrigger)
			triggers.GET("", triggerHandler.ListTriggers)
			triggers.GET("/:id", triggerHandler.GetTrigger)
			triggers.PUT("/:id", triggerHandler.UpdateTrigger)
			triggers.DELETE("/:id", triggerHandler.DeleteTrigger)
			triggers.POST("/:id/test", triggerHandler.TestTrigger)
		}

		// Event log queries
		eventHandler := handlers.NewEventHandler(s.logger)
		events := v1.Group("/events")
		{
			events.GET("", eventHandler.ListEvents)
			events.GET("/:id", eventHandler.GetEvent)
		}

		// Webhook receiver
		webhookHandler := handlers.NewWebhookHandler(s.logger)
		v1.POST("/webhook/:trigger_id", webhookHandler.ReceiveWebhook)
	}

	s.router = router
}

// getZapLogger extracts the underlying *zap.Logger from our Logger interface.
// This is needed for gin-contrib/zap middleware.
func (s *Server) getZapLogger() *zap.Logger {
	// Create a new zap logger for middleware (gin-contrib/zap needs *zap.Logger)
	var zapLogger *zap.Logger
	if s.config.Environment == "production" {
		zapLogger, _ = zap.NewProduction()
	} else {
		zapLogger, _ = zap.NewDevelopment()
	}
	return zapLogger
}

// Serve starts the HTTP server with graceful shutdown support.
func (s *Server) Serve() error {
	addr := ":" + s.config.APIPort
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to listen for interrupt signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		s.logger.Info("starting API server",
			zap.String("address", addr),
			zap.String("environment", s.config.Environment),
			zap.String("log_level", s.config.LogLevel),
		)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	<-quit
	s.logger.Info("shutting down server gracefully...")

	// Graceful shutdown with 30 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		s.logger.Error("server forced to shutdown", zap.Error(err))
		return err
	}

	if s.db != nil {
		if err := s.db.Close(); err != nil {
			s.logger.Error("failed to close database connection", zap.Error(err))
		}
	}

	// Flush logger before exit
	if err := s.logger.Sync(); err != nil {
		// Ignore sync errors on stdout/stderr
		if err.Error() != "sync /dev/stdout: invalid argument" &&
			err.Error() != "sync /dev/stderr: invalid argument" {
			return err
		}
	}

	s.logger.Info("server stopped")
	return nil
}

func connectDatabase(cfg config.App, logger logging.Logger) *sql.DB {
	if cfg.DatabaseURL == "" {
		logger.Fatal("DATABASE_URL is required")
	}

	db, err := sql.Open("mysql", cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("failed to open database connection", zap.Error(err))
	}

	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(20)
	db.SetConnMaxLifetime(60 * time.Minute)

	if err := db.Ping(); err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}

	return db
}
