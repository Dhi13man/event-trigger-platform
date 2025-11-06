package config

import (
	"os"
	"strings"
)

// App holds runtime configuration derived from env vars or files.
type App struct {
	// Database & Queue
	DatabaseURL  string
	KafkaBrokers string

	// Server
	APIPort     string
	Environment string // development, production

	// Logging
	LogLevel    string
	LogEncoding string // json, console

	// CORS
	CORSOrigins []string
}

// FromEnv loads the application configuration from environment variables.
func FromEnv() App {
	return App{
		DatabaseURL:  getEnv("DATABASE_URL", ""),
		KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:9092"),
		APIPort:      getEnv("API_PORT", "8080"),
		Environment:  getEnv("ENVIRONMENT", "production"),
		LogLevel:     getEnv("LOG_LEVEL", "info"),
		LogEncoding:  getEnv("LOG_ENCODING", "json"),
		CORSOrigins:  getCORSOrigins(),
	}
}

// getEnv retrieves an environment variable with a fallback default value.
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getCORSOrigins parses CORS origins from environment variable.
// Expected format: comma-separated list (e.g., "http://localhost:3000,https://app.example.com")
func getCORSOrigins() []string {
	origins := os.Getenv("CORS_ORIGINS")
	if origins == "" {
		return []string{"*"} // Allow all origins by default in dev
	}

	parsed := strings.Split(origins, ",")
	result := make([]string, 0, len(parsed))
	for _, origin := range parsed {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
