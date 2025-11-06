package logging

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger defines the interface for structured logging operations.
// This abstraction allows for testing and swapping implementations.
type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Fatal(msg string, fields ...zap.Field)
	With(fields ...zap.Field) Logger
	Sync() error
}

// zapLogger wraps zap.Logger to implement our Logger interface.
type zapLogger struct {
	logger *zap.Logger
}

// NewLogger creates a new production-ready logger based on environment.
// Environment can be "development" or "production".
func NewLogger(environment, logLevel string) (Logger, error) {
	var config zap.Config

	if environment == "development" {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// Parse log level
	level, err := zapcore.ParseLevel(logLevel)
	if err != nil {
		level = zapcore.InfoLevel // default to info
	}
	config.Level = zap.NewAtomicLevelAt(level)

	// Enable sampling to prevent log storms in production
	if environment == "production" {
		config.Sampling = &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		}
	}

	logger, err := config.Build(
		zap.AddCallerSkip(1), // Skip one level to show correct caller
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return nil, err
	}

	return &zapLogger{logger: logger}, nil
}

// NewDevelopmentLogger creates a logger optimized for development.
func NewDevelopmentLogger() (Logger, error) {
	return NewLogger("development", "debug")
}

// NewProductionLogger creates a logger optimized for production.
func NewProductionLogger() (Logger, error) {
	return NewLogger("production", "info")
}

// NewFromEnv creates a logger based on environment variables.
// Reads LOG_LEVEL and ENVIRONMENT from env.
func NewFromEnv() (Logger, error) {
	environment := os.Getenv("ENVIRONMENT")
	if environment == "" {
		environment = "production"
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}

	return NewLogger(environment, logLevel)
}

// Debug logs a debug-level message with structured fields.
func (l *zapLogger) Debug(msg string, fields ...zap.Field) {
	l.logger.Debug(msg, fields...)
}

// Info logs an info-level message with structured fields.
func (l *zapLogger) Info(msg string, fields ...zap.Field) {
	l.logger.Info(msg, fields...)
}

// Warn logs a warning-level message with structured fields.
func (l *zapLogger) Warn(msg string, fields ...zap.Field) {
	l.logger.Warn(msg, fields...)
}

// Error logs an error-level message with structured fields.
func (l *zapLogger) Error(msg string, fields ...zap.Field) {
	l.logger.Error(msg, fields...)
}

// Fatal logs a fatal-level message and exits the application.
func (l *zapLogger) Fatal(msg string, fields ...zap.Field) {
	l.logger.Fatal(msg, fields...)
}

// With creates a child logger with additional fields.
// Returns a new Logger instance with fields permanently attached.
func (l *zapLogger) With(fields ...zap.Field) Logger {
	return &zapLogger{logger: l.logger.With(fields...)}
}

// Sync flushes any buffered log entries.
// Should be called before application exits.
func (l *zapLogger) Sync() error {
	return l.logger.Sync()
}

// NoOpLogger is a logger that does nothing. Useful for testing.
type NoOpLogger struct{}

func (l *NoOpLogger) Debug(msg string, fields ...zap.Field) {}
func (l *NoOpLogger) Info(msg string, fields ...zap.Field)  {}
func (l *NoOpLogger) Warn(msg string, fields ...zap.Field)  {}
func (l *NoOpLogger) Error(msg string, fields ...zap.Field) {}
func (l *NoOpLogger) Fatal(msg string, fields ...zap.Field) {}
func (l *NoOpLogger) With(fields ...zap.Field) Logger       { return l }
func (l *NoOpLogger) Sync() error                           { return nil }

// NewNoOpLogger creates a no-op logger for testing.
func NewNoOpLogger() Logger {
	return &NoOpLogger{}
}
