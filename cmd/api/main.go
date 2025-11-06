package main

import (
	"log"

	_ "github.com/dhima/event-trigger-platform/docs" // Import generated docs
	"github.com/dhima/event-trigger-platform/internal/api"
)

// @title Event Trigger Platform API
// @version 1.0
// @description Production-ready event trigger management system supporting scheduled and API-triggered events with Kafka integration.
// @description
// @description ## Features
// @description - **Scheduled Triggers**: One-time and recurring schedules with ISO 8601 and interval-based scheduling
// @description - **API Triggers**: Webhook-based triggers with JSON schema validation
// @description - **Event Logs**: Comprehensive logging with retention lifecycle (active → archived → deleted)
// @description - **Kafka Integration**: Reliable event publishing for external consumers
// @description
// @description ## Architecture
// @description This platform is an event publisher, not an executor. We publish trigger events to Kafka, and external consumers (owned by users) subscribe and execute business logic.

// @contact.name API Support
// @contact.url https://github.com/Dhi13man/event-trigger-platform
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1
// @schemes http https

// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name X-API-Key
// @description Optional API key authentication for management operations

func main() {
	srv := api.NewServer()
	if err := srv.Serve(); err != nil {
		log.Fatalf("api server stopped: %v", err)
	}
}
