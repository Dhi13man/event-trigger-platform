package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/dhima/event-trigger-platform/internal/events"
	"github.com/dhima/event-trigger-platform/internal/logging"
	"github.com/dhima/event-trigger-platform/internal/scheduler"
	"github.com/dhima/event-trigger-platform/internal/storage"
	"github.com/dhima/event-trigger-platform/pkg/config"
	platformEvents "github.com/dhima/event-trigger-platform/platform/events"
	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

func main() {
	// Load configuration from environment
	cfg := config.FromEnv()

	// Initialize logger
	logger, err := logging.NewLogger(cfg.Environment, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Create zap logger for Kafka publisher (needs *zap.Logger, not our Logger interface)
	var zapLogger *zap.Logger
	var zapErr error
	if cfg.Environment == "production" {
		zapLogger, zapErr = zap.NewProduction()
	} else {
		zapLogger, zapErr = zap.NewDevelopment()
	}
	if zapErr != nil {
		log.Fatalf("failed to initialize zap logger for Kafka: %v", zapErr)
	}

	zapLogger.Info("starting scheduler service",
		zap.String("environment", cfg.Environment),
		zap.String("database_url", maskPassword(cfg.DatabaseURL)),
		zap.String("kafka_brokers", cfg.KafkaBrokers))

	// Connect to MySQL database
	db, err := sql.Open("mysql", cfg.DatabaseURL)
	if err != nil {
		zapLogger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Verify database connection
	if err := db.Ping(); err != nil {
		zapLogger.Fatal("failed to ping database", zap.Error(err))
	}
	zapLogger.Info("database connection established")

	// Initialize MySQL client
	mysqlClient := storage.NewMySQLClient(db)

	// Initialize Kafka publisher
	kafkaBrokers := parseKafkaBrokers(cfg.KafkaBrokers)
	kafkaPublisher := platformEvents.NewPublisher(kafkaBrokers, "trigger-events", zapLogger)
	defer func() {
		if err := kafkaPublisher.Close(); err != nil {
			zapLogger.Error("failed to close Kafka publisher", zap.Error(err))
		}
	}()
	zapLogger.Info("Kafka publisher initialized",
		zap.Strings("brokers", kafkaBrokers),
		zap.String("topic", "trigger-events"))

	// Initialize EventService
	eventService := events.NewService(mysqlClient, kafkaPublisher, zapLogger)
	zapLogger.Info("event service initialized")

	// Initialize Scheduler Engine (5 second polling interval)
	tickInterval := 5 * time.Second
	engine := scheduler.NewEngine(tickInterval, mysqlClient, eventService, zapLogger)

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		zapLogger.Info("received shutdown signal",
			zap.String("signal", sig.String()))
		cancel()
	}()

	// Run scheduler engine
	zapLogger.Info("scheduler engine starting",
		zap.Duration("tick_interval", tickInterval))

	if err := engine.Run(ctx); err != nil {
		if err == context.Canceled {
			zapLogger.Info("scheduler engine stopped gracefully")
		} else {
			zapLogger.Fatal("scheduler engine stopped with error", zap.Error(err))
		}
	}

	zapLogger.Info("scheduler service shut down successfully")
}

// parseKafkaBrokers parses comma-separated Kafka broker list.
func parseKafkaBrokers(brokers string) []string {
	// Split by comma and trim whitespace
	brokerList := strings.Split(brokers, ",")
	for i, broker := range brokerList {
		brokerList[i] = strings.TrimSpace(broker)
	}
	return brokerList
}

// maskPassword masks the password in the database URL for logging.
func maskPassword(dsn string) string {
	// Format: user:password@tcp(host:port)/dbname
	if idx := strings.Index(dsn, "@"); idx > 0 {
		if colonIdx := strings.Index(dsn, ":"); colonIdx > 0 && colonIdx < idx {
			return dsn[:colonIdx+1] + "****" + dsn[idx:]
		}
	}
	return dsn
}
