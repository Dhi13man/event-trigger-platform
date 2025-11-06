package models

import (
	"encoding/json"
	"time"
)

// TriggerType represents the type of trigger.
type TriggerType string

const (
	TriggerTypeWebhook       TriggerType = "webhook"
	TriggerTypeTimeScheduled TriggerType = "time_scheduled"
	TriggerTypeCronScheduled TriggerType = "cron_scheduled"
)

// TriggerStatus represents the status of a trigger.
type TriggerStatus string

const (
	TriggerStatusActive   TriggerStatus = "active"
	TriggerStatusInactive TriggerStatus = "inactive"
)

// ScheduleStatus is the processing status for schedule rows.
type ScheduleStatus string

const (
	ScheduleStatusPending    ScheduleStatus = "pending"
	ScheduleStatusProcessing ScheduleStatus = "processing"
	ScheduleStatusCompleted  ScheduleStatus = "completed"
	ScheduleStatusCancelled  ScheduleStatus = "cancelled"
)

// Trigger represents a trigger entity from the database.
type Trigger struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Type      TriggerType     `json:"type"`
	Status    TriggerStatus   `json:"status"`
	Config    json.RawMessage `json:"config"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// TriggerSchedule represents pending or processed occurrences for a trigger.
type TriggerSchedule struct {
	ID            string         `json:"id"`
	TriggerID     string         `json:"trigger_id"`
	FireAt        time.Time      `json:"fire_at"`
	Status        ScheduleStatus `json:"status"`
	AttemptCount  int            `json:"attempt_count"`
	LastAttemptAt *time.Time     `json:"last_attempt_at,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// CreateTriggerRequest represents the request to create a trigger.
type CreateTriggerRequest struct {
	Name   string          `json:"name" binding:"required" example:"Daily metrics push"`
	Type   TriggerType     `json:"type" binding:"required,oneof=webhook time_scheduled cron_scheduled" example:"time_scheduled"`
	Config json.RawMessage `json:"config" binding:"required" swaggertype:"object"`
} // @name CreateTriggerRequest

// UpdateTriggerRequest represents the request to update a trigger.
type UpdateTriggerRequest struct {
	Name   *string         `json:"name,omitempty" example:"Daily metrics push"`
	Status *TriggerStatus  `json:"status,omitempty" binding:"omitempty,oneof=active inactive" example:"active"`
	Config json.RawMessage `json:"config,omitempty" swaggertype:"object"`
} // @name UpdateTriggerRequest

// TriggerResponse represents the response for a single trigger.
type TriggerResponse struct {
	ID               string          `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Name             string          `json:"name" example:"Daily metrics push"`
	Type             TriggerType     `json:"type" example:"time_scheduled"`
	Status           TriggerStatus   `json:"status" example:"active"`
	Config           json.RawMessage `json:"config" swaggertype:"object"`
	NextScheduledRun *time.Time      `json:"next_scheduled_run,omitempty" example:"2025-11-05T15:00:00Z"`
	WebhookURL       string          `json:"webhook_url,omitempty" example:"http://localhost:8080/api/v1/webhook/550e8400-e29b-41d4-a716-446655440000"`
	CreatedAt        time.Time       `json:"created_at" example:"2025-11-05T10:00:00Z"`
	UpdatedAt        time.Time       `json:"updated_at" example:"2025-11-05T10:00:00Z"`
} // @name TriggerResponse

// WebhookTriggerConfig holds configuration for webhook triggers that run on inbound HTTP calls.
type WebhookTriggerConfig struct {
	Schema     map[string]interface{} `json:"schema"` // JSON schema for payload validation
	Endpoint   string                 `json:"endpoint" example:"https://webhook.site/xyz"`
	HTTPMethod string                 `json:"http_method" example:"POST"`
	Headers    map[string]string      `json:"headers,omitempty"`
}

// TimeScheduledTriggerConfig configures a one-shot trigger.
type TimeScheduledTriggerConfig struct {
	RunAt      time.Time              `json:"run_at" example:"2025-11-05T15:00:00Z"`
	Endpoint   string                 `json:"endpoint" example:"https://webhook.site/xyz"`
	HTTPMethod string                 `json:"http_method" example:"POST"`
	Headers    map[string]string      `json:"headers,omitempty"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
	Timezone   string                 `json:"timezone,omitempty" example:"America/New_York"`
}

// CronScheduledTriggerConfig configures a recurring trigger based on a cron expression.
type CronScheduledTriggerConfig struct {
	Cron       string                 `json:"cron" example:"0 9 * * *"`
	Timezone   string                 `json:"timezone,omitempty" example:"America/New_York"`
	Endpoint   string                 `json:"endpoint" example:"https://webhook.site/xyz"`
	HTTPMethod string                 `json:"http_method" example:"POST"`
	Headers    map[string]string      `json:"headers,omitempty"`
	Payload    map[string]interface{} `json:"payload,omitempty"`
}

// ListTriggersQuery represents query parameters for listing triggers.
type ListTriggersQuery struct {
	Type   string `form:"type" binding:"omitempty,oneof=webhook time_scheduled cron_scheduled" example:"time_scheduled"`
	Status string `form:"status" binding:"omitempty,oneof=active inactive" example:"active"`
	Page   int    `form:"page" binding:"omitempty,min=1" example:"1"`
	Limit  int    `form:"limit" binding:"omitempty,min=1,max=100" example:"20"`
} // @name ListTriggersQuery

// TriggerListResponse represents the response for listing triggers.
type TriggerListResponse struct {
	Triggers   []TriggerResponse `json:"triggers"`
	Pagination Pagination        `json:"pagination"`
} // @name TriggerListResponse

// Pagination represents pagination metadata.
type Pagination struct {
	CurrentPage  int   `json:"current_page" example:"1"`
	PageSize     int   `json:"page_size" example:"20"`
	TotalPages   int   `json:"total_pages" example:"5"`
	TotalRecords int64 `json:"total_records" example:"100"`
} // @name Pagination
