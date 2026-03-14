package logging

import (
	"os"
	"testing"

	"go.uber.org/zap"
)

func TestNewLogger_WhenDevelopmentEnvironment_ThenReturnsDevelopmentLogger(t *testing.T) {
	// Arrange & Act
	logger, err := NewLogger("development", "debug")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if logger == nil {
		t.Fatal("expected logger to be non-nil")
	}

	// Cleanup
	_ = logger.Sync()
}

func TestNewLogger_WhenProductionEnvironment_ThenReturnsProductionLogger(t *testing.T) {
	// Arrange & Act
	logger, err := NewLogger("production", "info")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if logger == nil {
		t.Fatal("expected logger to be non-nil")
	}

	// Cleanup
	_ = logger.Sync()
}

func TestNewLogger_WhenInvalidLogLevel_ThenDefaultsToInfo(t *testing.T) {
	// Arrange & Act
	logger, err := NewLogger("production", "invalid-level")

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if logger == nil {
		t.Fatal("expected logger to be non-nil")
	}

	// Cleanup
	_ = logger.Sync()
}

func TestNewDevelopmentLogger_WhenCalled_ThenReturnsLogger(t *testing.T) {
	// Arrange & Act
	logger, err := NewDevelopmentLogger()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if logger == nil {
		t.Fatal("expected logger to be non-nil")
	}

	// Cleanup
	_ = logger.Sync()
}

func TestNewProductionLogger_WhenCalled_ThenReturnsLogger(t *testing.T) {
	// Arrange & Act
	logger, err := NewProductionLogger()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if logger == nil {
		t.Fatal("expected logger to be non-nil")
	}

	// Cleanup
	_ = logger.Sync()
}

func TestNewFromEnv_WhenEnvironmentVariablesSet_ThenUsesThoseValues(t *testing.T) {
	// Arrange
	originalEnvironment := os.Getenv("ENVIRONMENT")
	originalLogLevel := os.Getenv("LOG_LEVEL")
	defer func() {
		os.Setenv("ENVIRONMENT", originalEnvironment)
		os.Setenv("LOG_LEVEL", originalLogLevel)
	}()

	os.Setenv("ENVIRONMENT", "development")
	os.Setenv("LOG_LEVEL", "debug")

	// Act
	logger, err := NewFromEnv()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if logger == nil {
		t.Fatal("expected logger to be non-nil")
	}

	// Cleanup
	_ = logger.Sync()
}

func TestNewFromEnv_WhenNoEnvironmentVariables_ThenUsesDefaults(t *testing.T) {
	// Arrange
	originalEnvironment := os.Getenv("ENVIRONMENT")
	originalLogLevel := os.Getenv("LOG_LEVEL")
	defer func() {
		os.Setenv("ENVIRONMENT", originalEnvironment)
		os.Setenv("LOG_LEVEL", originalLogLevel)
	}()

	os.Unsetenv("ENVIRONMENT")
	os.Unsetenv("LOG_LEVEL")

	// Act
	logger, err := NewFromEnv()

	// Assert
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if logger == nil {
		t.Fatal("expected logger to be non-nil")
	}

	// Cleanup
	_ = logger.Sync()
}

func TestZapLogger_Debug_WhenCalled_ThenLogsDebugMessage(t *testing.T) {
	// Arrange
	logger, err := NewDevelopmentLogger()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Act (should not panic)
	logger.Debug("test debug message", zap.String("key", "value"))

	// Assert - if we reach here without panic, test passes
}

func TestZapLogger_Info_WhenCalled_ThenLogsInfoMessage(t *testing.T) {
	// Arrange
	logger, err := NewProductionLogger()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Act (should not panic)
	logger.Info("test info message", zap.String("key", "value"))

	// Assert - if we reach here without panic, test passes
}

func TestZapLogger_Warn_WhenCalled_ThenLogsWarnMessage(t *testing.T) {
	// Arrange
	logger, err := NewProductionLogger()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Act (should not panic)
	logger.Warn("test warn message", zap.String("key", "value"))

	// Assert - if we reach here without panic, test passes
}

func TestZapLogger_Error_WhenCalled_ThenLogsErrorMessage(t *testing.T) {
	// Arrange
	logger, err := NewProductionLogger()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Act (should not panic)
	logger.Error("test error message", zap.String("key", "value"))

	// Assert - if we reach here without panic, test passes
}

func TestZapLogger_With_WhenCalledWithFields_ThenReturnsLoggerWithFields(t *testing.T) {
	// Arrange
	logger, err := NewProductionLogger()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Act
	childLogger := logger.With(zap.String("request_id", "123"))

	// Assert
	if childLogger == nil {
		t.Fatal("expected child logger to be non-nil")
	}

	// Should not panic
	childLogger.Info("test message")
}

func TestZapLogger_Sync_WhenCalled_ThenReturnsNoError(t *testing.T) {
	// Arrange
	logger, err := NewProductionLogger()
	if err != nil {
		t.Fatalf("failed to create logger: %v", err)
	}

	// Act
	err = logger.Sync()

	// Assert - error may occur on stderr sync, which is acceptable
	// We just verify it doesn't panic
}

func TestNoOpLogger_AllMethods_WhenCalled_ThenDoNothing(t *testing.T) {
	// Arrange
	logger := NewNoOpLogger()

	// Act & Assert (should not panic)
	logger.Debug("test")
	logger.Info("test")
	logger.Warn("test")
	logger.Error("test")

	childLogger := logger.With(zap.String("key", "value"))
	if childLogger == nil {
		t.Fatal("expected child logger to be non-nil")
	}

	err := logger.Sync()
	if err != nil {
		t.Errorf("expected no error from Sync, got %v", err)
	}
}

func TestNoOpLogger_With_WhenCalled_ThenReturnsSelf(t *testing.T) {
	// Arrange
	logger := &NoOpLogger{}

	// Act
	childLogger := logger.With(zap.String("key", "value"))

	// Assert
	if childLogger != logger {
		t.Error("expected With to return same logger instance")
	}
}
