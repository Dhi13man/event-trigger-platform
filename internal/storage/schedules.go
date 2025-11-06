package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/dhima/event-trigger-platform/internal/models"
)

// ScheduleWithTrigger combines a trigger schedule with its parent trigger context.
// Used by the scheduler to have all necessary information for firing triggers.
type ScheduleWithTrigger struct {
	Schedule models.TriggerSchedule
	Trigger  models.Trigger
}

// GetDueSchedules retrieves pending schedules that are due to fire, along with their trigger context.
// Uses JOIN query to fetch both schedule and trigger data in a single query.
// Results are ordered by fire_at ASC to process oldest schedules first.
func (c *MySQLClient) GetDueSchedules(ctx context.Context, limit int) ([]ScheduleWithTrigger, error) {
	query := `
		SELECT
			ts.id, ts.trigger_id, ts.fire_at, ts.status, ts.attempt_count, ts.last_attempt_at, ts.created_at, ts.updated_at,
			t.id, t.name, t.type, t.status, t.config, t.created_at, t.updated_at
		FROM trigger_schedules ts
		INNER JOIN triggers t ON ts.trigger_id = t.id
		WHERE ts.fire_at <= NOW()
		  AND ts.status = 'pending'
		  AND t.status = 'active'
		ORDER BY ts.fire_at ASC
		LIMIT ?
	`

	rows, err := c.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query due schedules: %w", err)
	}
	defer rows.Close()

	var schedules []ScheduleWithTrigger
	for rows.Next() {
		var s ScheduleWithTrigger
		var lastAttemptAt sql.NullTime

		err := rows.Scan(
			// Schedule fields
			&s.Schedule.ID,
			&s.Schedule.TriggerID,
			&s.Schedule.FireAt,
			&s.Schedule.Status,
			&s.Schedule.AttemptCount,
			&lastAttemptAt,
			&s.Schedule.CreatedAt,
			&s.Schedule.UpdatedAt,
			// Trigger fields
			&s.Trigger.ID,
			&s.Trigger.Name,
			&s.Trigger.Type,
			&s.Trigger.Status,
			&s.Trigger.Config,
			&s.Trigger.CreatedAt,
			&s.Trigger.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan schedule with trigger: %w", err)
		}

		// Handle nullable fields
		if lastAttemptAt.Valid {
			s.Schedule.LastAttemptAt = &lastAttemptAt.Time
		}

		schedules = append(schedules, s)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating due schedules: %w", err)
	}

	return schedules, nil
}

// UpdateScheduleStatus updates the status of a schedule (without incrementing attempts).
// This method uses row-level locking (implicitly via UPDATE) to prevent duplicate processing.
// Used for: pending → processing, processing → completed
func (c *MySQLClient) UpdateScheduleStatus(ctx context.Context, scheduleID string, status models.ScheduleStatus) error {
	query := `
		UPDATE trigger_schedules
		SET status = ?,
		    updated_at = NOW()
		WHERE id = ?
	`

	result, err := c.db.ExecContext(ctx, query, status, scheduleID)
	if err != nil {
		return fmt.Errorf("failed to update schedule status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("schedule not found: %s", scheduleID)
	}

	return nil
}

// IncrementScheduleAttempt increments the attempt count and updates last_attempt_at.
// Used when a trigger firing fails and needs to be retried.
func (c *MySQLClient) IncrementScheduleAttempt(ctx context.Context, scheduleID string) error {
	query := `
		UPDATE trigger_schedules
		SET attempt_count = attempt_count + 1,
		    last_attempt_at = NOW(),
		    updated_at = NOW()
		WHERE id = ?
	`

	result, err := c.db.ExecContext(ctx, query, scheduleID)
	if err != nil {
		return fmt.Errorf("failed to increment schedule attempt: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("schedule not found: %s", scheduleID)
	}

	return nil
}

// RevertScheduleToPending reverts a schedule from 'processing' to 'pending' for retry.
// Increments attempt_count and updates last_attempt_at.
// Used when trigger firing fails but hasn't exceeded max retries.
func (c *MySQLClient) RevertScheduleToPending(ctx context.Context, scheduleID string) error {
	query := `
		UPDATE trigger_schedules
		SET status = 'pending',
		    attempt_count = attempt_count + 1,
		    last_attempt_at = NOW(),
		    updated_at = NOW()
		WHERE id = ?
	`

	result, err := c.db.ExecContext(ctx, query, scheduleID)
	if err != nil {
		return fmt.Errorf("failed to revert schedule to pending: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("schedule not found: %s", scheduleID)
	}

	return nil
}

// CreateNextSchedule inserts a new schedule entry for recurring (CRON) triggers.
// This is called after successfully firing a CRON trigger to schedule the next occurrence.
func (c *MySQLClient) CreateNextSchedule(ctx context.Context, schedule *models.TriggerSchedule) error {
	query := `
		INSERT INTO trigger_schedules (id, trigger_id, fire_at, status, attempt_count)
		VALUES (?, ?, ?, ?, ?)
	`

	_, err := c.db.ExecContext(ctx, query,
		schedule.ID,
		schedule.TriggerID,
		schedule.FireAt,
		schedule.Status,
		schedule.AttemptCount,
	)

	if err != nil {
		return fmt.Errorf("failed to create next schedule: %w", err)
	}

	return nil
}

// DeactivateTrigger marks a trigger as inactive.
// This is used for one-time (time_scheduled) triggers after they fire.
func (c *MySQLClient) DeactivateTrigger(ctx context.Context, triggerID string) error {
	query := `
		UPDATE triggers
		SET status = 'inactive', updated_at = NOW()
		WHERE id = ?
	`

	result, err := c.db.ExecContext(ctx, query, triggerID)
	if err != nil {
		return fmt.Errorf("failed to deactivate trigger: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("trigger not found: %s", triggerID)
	}

	return nil
}

// ParseTriggerConfig parses a trigger's JSON config into a typed struct.
// This is a helper function to extract endpoint, payload, etc. from trigger config.
func ParseTriggerConfig(trigger *models.Trigger) (map[string]interface{}, error) {
	var config map[string]interface{}
	err := json.Unmarshal(trigger.Config, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse trigger config: %w", err)
	}
	return config, nil
}

// ExtractPayloadFromConfig extracts the payload from a trigger config for firing.
// Different trigger types have different config structures, but all have optional payload fields.
func ExtractPayloadFromConfig(triggerType models.TriggerType, config map[string]interface{}) map[string]interface{} {
	// For webhook triggers, the payload comes from the webhook request, not the config
	if triggerType == models.TriggerTypeWebhook {
		return config
	}

	// For time_scheduled and cron_scheduled, payload is in the config
	if payload, ok := config["payload"].(map[string]interface{}); ok {
		return payload
	}

	// If no payload specified, return the entire config (contains endpoint, headers, etc.)
	return config
}
