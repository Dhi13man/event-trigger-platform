package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dhima/event-trigger-platform/internal/models"
)

// CreateEventLog inserts a new event log entry into the database.
func (c *MySQLClient) CreateEventLog(ctx context.Context, eventLog *models.EventLog) error {
	query := `
		INSERT INTO event_logs (
			id, trigger_id, trigger_type, fired_at, payload, source,
			execution_status, error_message, retention_status, is_test_run, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	// Convert payload to JSON bytes
	var payloadBytes []byte
	var err error
	if eventLog.Payload != nil {
		payloadBytes = eventLog.Payload
	}

	_, err = c.db.ExecContext(ctx, query,
		eventLog.ID,
		eventLog.TriggerID,
		eventLog.TriggerType,
		eventLog.FiredAt,
		payloadBytes,
		eventLog.Source,
		eventLog.ExecutionStatus,
		eventLog.ErrorMessage,
		eventLog.RetentionStatus,
		eventLog.IsTestRun,
		eventLog.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create event log: %w", err)
	}

	return nil
}

// UpdateEventLogStatus updates the execution status and error message of an event log.
// This is used when Kafka publishing fails after the event log is created.
func (c *MySQLClient) UpdateEventLogStatus(ctx context.Context, eventID string, status models.ExecutionStatus, errorMessage *string) error {
	query := `
		UPDATE event_logs
		SET execution_status = ?, error_message = ?
		WHERE id = ?
	`

	_, err := c.db.ExecContext(ctx, query, status, errorMessage, eventID)
	if err != nil {
		return fmt.Errorf("failed to update event log status: %w", err)
	}

	return nil
}

// GetEventLog retrieves a single event log by ID.
func (c *MySQLClient) GetEventLog(ctx context.Context, eventID string) (*models.EventLog, error) {
	query := `
		SELECT id, trigger_id, trigger_type, fired_at, payload, source,
		       execution_status, error_message, retention_status, is_test_run, created_at
		FROM event_logs
		WHERE id = ?
	`

	row := c.db.QueryRowContext(ctx, query, eventID)

	var eventLog models.EventLog
	var triggerID sql.NullString
	var errorMessage sql.NullString
	var payload sql.NullString

	err := row.Scan(
		&eventLog.ID,
		&triggerID,
		&eventLog.TriggerType,
		&eventLog.FiredAt,
		&payload,
		&eventLog.Source,
		&eventLog.ExecutionStatus,
		&errorMessage,
		&eventLog.RetentionStatus,
		&eventLog.IsTestRun,
		&eventLog.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // Event log not found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get event log: %w", err)
	}

	// Handle nullable fields
	if triggerID.Valid {
		eventLog.TriggerID = &triggerID.String
	}
	if errorMessage.Valid {
		eventLog.ErrorMessage = &errorMessage.String
	}
	if payload.Valid {
		eventLog.Payload = json.RawMessage(payload.String)
	}

	return &eventLog, nil
}

// ListEventLogs retrieves event logs with filtering and pagination.
// Returns the list of event logs and the total count for pagination.
func (c *MySQLClient) ListEventLogs(ctx context.Context, query models.ListEventsQuery) ([]models.EventLog, int64, error) {
	// Build WHERE clause dynamically based on filters
	whereClauses := []string{}
	args := []interface{}{}

	// Default to 'active' retention status if not specified
	retentionStatus := query.RetentionStatus
	if retentionStatus == "" {
		retentionStatus = "active"
	}
	whereClauses = append(whereClauses, "retention_status = ?")
	args = append(args, retentionStatus)

	if query.TriggerID != "" {
		whereClauses = append(whereClauses, "trigger_id = ?")
		args = append(args, query.TriggerID)
	}

	if query.ExecutionStatus != "" {
		whereClauses = append(whereClauses, "execution_status = ?")
		args = append(args, query.ExecutionStatus)
	}

	if query.Source != "" {
		whereClauses = append(whereClauses, "source = ?")
		args = append(args, query.Source)
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = "WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Get total count for pagination
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM event_logs %s", whereClause)
	var totalCount int64
	err := c.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count event logs: %w", err)
	}

	// Calculate pagination
	page := query.Page
	if page < 1 {
		page = 1
	}
	limit := query.Limit
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	// Get paginated results
	listQuery := fmt.Sprintf(`
		SELECT id, trigger_id, trigger_type, fired_at, payload, source,
		       execution_status, error_message, retention_status, is_test_run, created_at
		FROM event_logs
		%s
		ORDER BY fired_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)

	rows, err := c.db.QueryContext(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list event logs: %w", err)
	}
	defer rows.Close()

	eventLogs := []models.EventLog{}
	for rows.Next() {
		var eventLog models.EventLog
		var triggerID sql.NullString
		var errorMessage sql.NullString
		var payload sql.NullString

		err := rows.Scan(
			&eventLog.ID,
			&triggerID,
			&eventLog.TriggerType,
			&eventLog.FiredAt,
			&payload,
			&eventLog.Source,
			&eventLog.ExecutionStatus,
			&errorMessage,
			&eventLog.RetentionStatus,
			&eventLog.IsTestRun,
			&eventLog.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan event log: %w", err)
		}

		// Handle nullable fields
		if triggerID.Valid {
			eventLog.TriggerID = &triggerID.String
		}
		if errorMessage.Valid {
			eventLog.ErrorMessage = &errorMessage.String
		}
		if payload.Valid {
			eventLog.Payload = json.RawMessage(payload.String)
		}

		eventLogs = append(eventLogs, eventLog)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating event logs: %w", err)
	}

	return eventLogs, totalCount, nil
}
