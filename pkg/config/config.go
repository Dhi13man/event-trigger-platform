package config

import "os"

// App holds runtime configuration derived from env vars or files.
type App struct {
	DatabaseURL  string
	KafkaBrokers string
}

// FromEnv loads the application configuration from environment variables.
func FromEnv() App {
	return App{
		DatabaseURL:  os.Getenv("DATABASE_URL"),
		KafkaBrokers: os.Getenv("KAFKA_BROKERS"),
	}
}
