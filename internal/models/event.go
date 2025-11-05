package models

import (
	"encoding/json"
	"time"
)

// EventSource represents the source of an event.
type EventSource string

const (
	EventSourceWebhook    EventSource = "webhook"
	EventSourceScheduler  EventSource = "scheduler"
	EventSourceManualTest EventSource = "manual-test"
)

// ExecutionStatus represents the execution status of an event.
type ExecutionStatus string

const (
	ExecutionStatusSuccess ExecutionStatus = "success"
	ExecutionStatusFailure ExecutionStatus = "failure"
)

// RetentionStatus represents the retention lifecycle status.
type RetentionStatus string

const (
	RetentionStatusActive   RetentionStatus = "active"
	RetentionStatusArchived RetentionStatus = "archived"
	RetentionStatusDeleted  RetentionStatus = "deleted"
)

// EventLog represents an event log entity from the database.
type EventLog struct {
	ID              string          `json:"id"`
	TriggerID       *string         `json:"trigger_id,omitempty"` // NULL for manual test runs
	TriggerType     TriggerType     `json:"trigger_type"`
	FiredAt         time.Time       `json:"fired_at"`
	Payload         json.RawMessage `json:"payload,omitempty"`
	Source          EventSource     `json:"source"`
	ExecutionStatus ExecutionStatus `json:"execution_status"`
	ErrorMessage    *string         `json:"error_message,omitempty"`
	RetentionStatus RetentionStatus `json:"retention_status"`
	IsTestRun       bool            `json:"is_test_run"`
	CreatedAt       time.Time       `json:"created_at"`
}

// EventLogResponse represents the response for a single event log.
type EventLogResponse struct {
	ID              string          `json:"id" example:"660e8400-e29b-41d4-a716-446655440000"`
	TriggerID       *string         `json:"trigger_id,omitempty" example:"550e8400-e29b-41d4-a716-446655440000"`
	TriggerType     TriggerType     `json:"trigger_type" example:"scheduled"`
	FiredAt         time.Time       `json:"fired_at" example:"2025-11-05T10:30:00Z"`
	Payload         json.RawMessage `json:"payload,omitempty" swaggertype:"object"`
	Source          EventSource     `json:"source" example:"scheduled"`
	ExecutionStatus ExecutionStatus `json:"execution_status" example:"success"`
	ErrorMessage    *string         `json:"error_message,omitempty" example:"connection timeout"`
	RetentionStatus RetentionStatus `json:"retention_status" example:"active"`
	IsTestRun       bool            `json:"is_test_run" example:"false"`
	CreatedAt       time.Time       `json:"created_at" example:"2025-11-05T10:30:00Z"`
} // @name EventLogResponse

// ListEventsQuery represents query parameters for listing event logs.
type ListEventsQuery struct {
	TriggerID       string `form:"trigger_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	RetentionStatus string `form:"retention_status" binding:"omitempty,oneof=active archived" example:"active"`
	ExecutionStatus string `form:"execution_status" binding:"omitempty,oneof=success failure" example:"success"`
	Source          string `form:"source" binding:"omitempty,oneof=api scheduled manual-test" example:"scheduled"`
	Page            int    `form:"page" binding:"omitempty,min=1" example:"1"`
	Limit           int    `form:"limit" binding:"omitempty,min=1,max=100" example:"20"`
} // @name ListEventsQuery

// EventLogListResponse represents the response for listing event logs.
type EventLogListResponse struct {
	Events     []EventLogResponse `json:"events"`
	Pagination Pagination         `json:"pagination"`
} // @name EventLogListResponse

// WebhookPayload represents the payload sent to webhook endpoint.
type WebhookPayload struct {
	TriggerID string                 `json:"trigger_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Data      map[string]interface{} `json:"data"`
} // @name WebhookPayload
