package config

import (
	"os"
	"testing"
)

func TestFromEnv_WhenAllVariablesSet_ThenReturnsConfigWithSetValues(t *testing.T) {
	// Arrange
	originalDatabaseURL := os.Getenv("DATABASE_URL")
	originalKafkaBrokers := os.Getenv("KAFKA_BROKERS")
	originalAPIPort := os.Getenv("API_PORT")
	originalEnvironment := os.Getenv("ENVIRONMENT")
	originalLogLevel := os.Getenv("LOG_LEVEL")
	originalLogEncoding := os.Getenv("LOG_ENCODING")
	originalCORSOrigins := os.Getenv("CORS_ORIGINS")

	defer func() {
		os.Setenv("DATABASE_URL", originalDatabaseURL)
		os.Setenv("KAFKA_BROKERS", originalKafkaBrokers)
		os.Setenv("API_PORT", originalAPIPort)
		os.Setenv("ENVIRONMENT", originalEnvironment)
		os.Setenv("LOG_LEVEL", originalLogLevel)
		os.Setenv("LOG_ENCODING", originalLogEncoding)
		os.Setenv("CORS_ORIGINS", originalCORSOrigins)
	}()

	os.Setenv("DATABASE_URL", "user:pass@tcp(localhost:3306)/testdb")
	os.Setenv("KAFKA_BROKERS", "kafka1:9092,kafka2:9092")
	os.Setenv("API_PORT", "9000")
	os.Setenv("ENVIRONMENT", "development")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("LOG_ENCODING", "console")
	os.Setenv("CORS_ORIGINS", "http://localhost:3000,https://example.com")

	// Act
	config := FromEnv()

	// Assert
	if config.DatabaseURL != "user:pass@tcp(localhost:3306)/testdb" {
		t.Errorf("expected DatabaseURL to be 'user:pass@tcp(localhost:3306)/testdb', got '%s'", config.DatabaseURL)
	}
	if config.KafkaBrokers != "kafka1:9092,kafka2:9092" {
		t.Errorf("expected KafkaBrokers to be 'kafka1:9092,kafka2:9092', got '%s'", config.KafkaBrokers)
	}
	if config.APIPort != "9000" {
		t.Errorf("expected APIPort to be '9000', got '%s'", config.APIPort)
	}
	if config.Environment != "development" {
		t.Errorf("expected Environment to be 'development', got '%s'", config.Environment)
	}
	if config.LogLevel != "debug" {
		t.Errorf("expected LogLevel to be 'debug', got '%s'", config.LogLevel)
	}
	if config.LogEncoding != "console" {
		t.Errorf("expected LogEncoding to be 'console', got '%s'", config.LogEncoding)
	}
	if len(config.CORSOrigins) != 2 {
		t.Fatalf("expected 2 CORS origins, got %d", len(config.CORSOrigins))
	}
	if config.CORSOrigins[0] != "http://localhost:3000" {
		t.Errorf("expected first CORS origin to be 'http://localhost:3000', got '%s'", config.CORSOrigins[0])
	}
	if config.CORSOrigins[1] != "https://example.com" {
		t.Errorf("expected second CORS origin to be 'https://example.com', got '%s'", config.CORSOrigins[1])
	}
}

func TestFromEnv_WhenNoVariablesSet_ThenReturnsDefaults(t *testing.T) {
	// Arrange
	originalDatabaseURL := os.Getenv("DATABASE_URL")
	originalKafkaBrokers := os.Getenv("KAFKA_BROKERS")
	originalAPIPort := os.Getenv("API_PORT")
	originalEnvironment := os.Getenv("ENVIRONMENT")
	originalLogLevel := os.Getenv("LOG_LEVEL")
	originalLogEncoding := os.Getenv("LOG_ENCODING")
	originalCORSOrigins := os.Getenv("CORS_ORIGINS")

	defer func() {
		os.Setenv("DATABASE_URL", originalDatabaseURL)
		os.Setenv("KAFKA_BROKERS", originalKafkaBrokers)
		os.Setenv("API_PORT", originalAPIPort)
		os.Setenv("ENVIRONMENT", originalEnvironment)
		os.Setenv("LOG_LEVEL", originalLogLevel)
		os.Setenv("LOG_ENCODING", originalLogEncoding)
		os.Setenv("CORS_ORIGINS", originalCORSOrigins)
	}()

	os.Unsetenv("DATABASE_URL")
	os.Unsetenv("KAFKA_BROKERS")
	os.Unsetenv("API_PORT")
	os.Unsetenv("ENVIRONMENT")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("LOG_ENCODING")
	os.Unsetenv("CORS_ORIGINS")

	// Act
	config := FromEnv()

	// Assert
	if config.DatabaseURL != "" {
		t.Errorf("expected DatabaseURL to be empty, got '%s'", config.DatabaseURL)
	}
	if config.KafkaBrokers != "localhost:9092" {
		t.Errorf("expected KafkaBrokers to be 'localhost:9092', got '%s'", config.KafkaBrokers)
	}
	if config.APIPort != "8080" {
		t.Errorf("expected APIPort to be '8080', got '%s'", config.APIPort)
	}
	if config.Environment != "production" {
		t.Errorf("expected Environment to be 'production', got '%s'", config.Environment)
	}
	if config.LogLevel != "info" {
		t.Errorf("expected LogLevel to be 'info', got '%s'", config.LogLevel)
	}
	if config.LogEncoding != "json" {
		t.Errorf("expected LogEncoding to be 'json', got '%s'", config.LogEncoding)
	}
	if len(config.CORSOrigins) != 1 || config.CORSOrigins[0] != "*" {
		t.Errorf("expected CORS origins to be ['*'], got %v", config.CORSOrigins)
	}
}

func TestGetCORSOrigins_WhenMultipleOriginsWithWhitespace_ThenTrimsCorrectly(t *testing.T) {
	// Arrange
	originalCORSOrigins := os.Getenv("CORS_ORIGINS")
	defer os.Setenv("CORS_ORIGINS", originalCORSOrigins)

	os.Setenv("CORS_ORIGINS", " http://localhost:3000 , https://example.com ,  ")

	// Act
	origins := getCORSOrigins()

	// Assert
	if len(origins) != 2 {
		t.Fatalf("expected 2 origins after trimming, got %d", len(origins))
	}
	if origins[0] != "http://localhost:3000" {
		t.Errorf("expected first origin to be 'http://localhost:3000', got '%s'", origins[0])
	}
	if origins[1] != "https://example.com" {
		t.Errorf("expected second origin to be 'https://example.com', got '%s'", origins[1])
	}
}

func TestGetCORSOrigins_WhenEmpty_ThenReturnsWildcard(t *testing.T) {
	// Arrange
	originalCORSOrigins := os.Getenv("CORS_ORIGINS")
	defer os.Setenv("CORS_ORIGINS", originalCORSOrigins)

	os.Setenv("CORS_ORIGINS", "")

	// Act
	origins := getCORSOrigins()

	// Assert
	if len(origins) != 1 || origins[0] != "*" {
		t.Errorf("expected ['*'], got %v", origins)
	}
}

func TestGetCORSOrigins_WhenOnlyWhitespace_ThenReturnsEmpty(t *testing.T) {
	// Arrange
	originalCORSOrigins := os.Getenv("CORS_ORIGINS")
	defer os.Setenv("CORS_ORIGINS", originalCORSOrigins)

	os.Setenv("CORS_ORIGINS", "   ,  ,  ")

	// Act
	origins := getCORSOrigins()

	// Assert
	if len(origins) != 0 {
		t.Errorf("expected empty slice, got %v", origins)
	}
}

func TestGetEnv_WhenVariableSet_ThenReturnsValue(t *testing.T) {
	// Arrange
	originalValue := os.Getenv("TEST_VAR")
	defer os.Setenv("TEST_VAR", originalValue)

	os.Setenv("TEST_VAR", "custom_value")

	// Act
	result := getEnv("TEST_VAR", "default_value")

	// Assert
	if result != "custom_value" {
		t.Errorf("expected 'custom_value', got '%s'", result)
	}
}

func TestGetEnv_WhenVariableNotSet_ThenReturnsDefault(t *testing.T) {
	// Arrange
	os.Unsetenv("NONEXISTENT_VAR")

	// Act
	result := getEnv("NONEXISTENT_VAR", "default_value")

	// Assert
	if result != "default_value" {
		t.Errorf("expected 'default_value', got '%s'", result)
	}
}

func TestGetEnv_WhenVariableEmpty_ThenReturnsDefault(t *testing.T) {
	// Arrange
	originalValue := os.Getenv("EMPTY_VAR")
	defer os.Setenv("EMPTY_VAR", originalValue)

	os.Setenv("EMPTY_VAR", "")

	// Act
	result := getEnv("EMPTY_VAR", "default_value")

	// Assert
	if result != "default_value" {
		t.Errorf("expected 'default_value', got '%s'", result)
	}
}
