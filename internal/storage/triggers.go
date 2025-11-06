package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dhima/event-trigger-platform/internal/models"
)

// ErrTriggerNotFound is returned when a trigger is not found.
var ErrTriggerNotFound = errors.New("trigger not found")

// CreateTrigger inserts a trigger (and optional first schedule) atomically.
func (c *MySQLClient) CreateTrigger(ctx context.Context, trigger *models.Trigger, schedule *models.TriggerSchedule) error {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(
		ctx,
		`INSERT INTO triggers (id, name, type, status, config) VALUES (?, ?, ?, ?, ?)`,
		trigger.ID,
		trigger.Name,
		trigger.Type,
		trigger.Status,
		string(trigger.Config),
	); err != nil {
		return fmt.Errorf("insert trigger: %w", err)
	}

	if schedule != nil {
		if _, err = tx.ExecContext(
			ctx,
			`INSERT INTO trigger_schedules (id, trigger_id, fire_at, status, attempt_count) VALUES (?, ?, ?, ?, ?)`,
			schedule.ID,
			trigger.ID,
			schedule.FireAt,
			schedule.Status,
			0,
		); err != nil {
			return fmt.Errorf("insert trigger schedule: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetTrigger fetches a trigger and its next scheduled run if any.
func (c *MySQLClient) GetTrigger(ctx context.Context, triggerID string) (*models.Trigger, *time.Time, error) {
	row := c.db.QueryRowContext(
		ctx,
		`SELECT id, name, type, status, config, created_at, updated_at
		 FROM triggers WHERE id = ?`,
		triggerID,
	)

	var t models.Trigger
	var config string
	if err := row.Scan(&t.ID, &t.Name, &t.Type, &t.Status, &config, &t.CreatedAt, &t.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, ErrTriggerNotFound
		}
		return nil, nil, fmt.Errorf("scan trigger: %w", err)
	}

	t.Config = jsonRawMessage(config)

	next, err := c.getNextSchedule(ctx, triggerID)
	if err != nil {
		return nil, nil, err
	}

	return &t, next, nil
}

// ListTriggers returns triggers matching the filters with pagination information.
func (c *MySQLClient) ListTriggers(ctx context.Context, query models.ListTriggersQuery) ([]models.Trigger, []*time.Time, int64, error) {
	criteria := make([]string, 0, 2)
	args := make([]interface{}, 0, 4)

	if query.Type != "" {
		criteria = append(criteria, "type = ?")
		args = append(args, query.Type)
	}
	if query.Status != "" {
		criteria = append(criteria, "status = ?")
		args = append(args, query.Status)
	}

	where := ""
	if len(criteria) > 0 {
		where = "WHERE " + strings.Join(criteria, " AND ")
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM triggers %s", where)

	var total int64
	if err := c.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, nil, 0, fmt.Errorf("count triggers: %w", err)
	}

	offset := (query.Page - 1) * query.Limit
	argsWithPagination := append(append([]interface{}{}, args...), query.Limit, offset)

	dataQuery := fmt.Sprintf(`
		SELECT id, name, type, status, config, created_at, updated_at,
			(
				SELECT fire_at FROM trigger_schedules
				WHERE trigger_id = triggers.id
				  AND status IN ('pending', 'processing')
				ORDER BY fire_at ASC
				LIMIT 1
			) AS next_fire_at
		FROM triggers
		%s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?`, where)

	rows, err := c.db.QueryContext(ctx, dataQuery, argsWithPagination...)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("query triggers: %w", err)
	}
	defer rows.Close()

	triggers := make([]models.Trigger, 0)
	nextRuns := make([]*time.Time, 0)

	for rows.Next() {
		var trigger models.Trigger
		var config string
		var nextFire sql.NullTime
		if err := rows.Scan(&trigger.ID, &trigger.Name, &trigger.Type, &trigger.Status, &config, &trigger.CreatedAt, &trigger.UpdatedAt, &nextFire); err != nil {
			return nil, nil, 0, fmt.Errorf("scan trigger row: %w", err)
		}
		trigger.Config = jsonRawMessage(config)

		triggers = append(triggers, trigger)
		if nextFire.Valid {
			t := nextFire.Time
			nextRuns = append(nextRuns, &t)
		} else {
			nextRuns = append(nextRuns, nil)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, 0, fmt.Errorf("iterate triggers: %w", err)
	}

	return triggers, nextRuns, total, nil
}

// UpdateTrigger updates the mutable fields of a trigger.
func (c *MySQLClient) UpdateTrigger(ctx context.Context, triggerID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}

	setParts := make([]string, 0, len(updates)+1)
	args := make([]interface{}, 0, len(updates)+1)

	for column, value := range updates {
		setParts = append(setParts, fmt.Sprintf("%s = ?", column))
		args = append(args, value)
	}

	setParts = append(setParts, "updated_at = NOW()")
	args = append(args, triggerID)

	query := fmt.Sprintf("UPDATE triggers SET %s WHERE id = ?", strings.Join(setParts, ", "))
	res, err := c.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update trigger: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrTriggerNotFound
	}

	return nil
}

// DeleteTrigger removes a trigger and its schedules.
func (c *MySQLClient) DeleteTrigger(ctx context.Context, triggerID string) error {
	res, err := c.db.ExecContext(ctx, "DELETE FROM triggers WHERE id = ?", triggerID)
	if err != nil {
		return fmt.Errorf("delete trigger: %w", err)
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}

	if rows == 0 {
		return ErrTriggerNotFound
	}

	return nil
}

// UpsertTriggerSchedule replaces all pending schedules for a trigger and inserts the provided one.
func (c *MySQLClient) UpsertTriggerSchedule(ctx context.Context, triggerID string, schedule *models.TriggerSchedule) error {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.ExecContext(
		ctx,
		`UPDATE trigger_schedules
		 SET status = 'cancelled', updated_at = NOW()
		 WHERE trigger_id = ? AND status IN ('pending', 'processing')`,
		triggerID,
	); err != nil {
		return fmt.Errorf("cancel existing schedules: %w", err)
	}

	if schedule == nil {
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("commit transaction: %w", err)
		}
		return nil
	}

	if _, err = tx.ExecContext(
		ctx,
		`INSERT INTO trigger_schedules (id, trigger_id, fire_at, status, attempt_count)
		 VALUES (?, ?, ?, ?, 0)`,
		schedule.ID,
		triggerID,
		schedule.FireAt,
		schedule.Status,
	); err != nil {
		return fmt.Errorf("insert schedule: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (c *MySQLClient) getNextSchedule(ctx context.Context, triggerID string) (*time.Time, error) {
	row := c.db.QueryRowContext(
		ctx,
		`SELECT fire_at
		 FROM trigger_schedules
		 WHERE trigger_id = ? AND status IN ('pending', 'processing')
		 ORDER BY fire_at ASC
		 LIMIT 1`,
		triggerID,
	)

	var next sql.NullTime
	if err := row.Scan(&next); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("scan next schedule: %w", err)
	}

	if !next.Valid {
		return nil, nil
	}
	t := next.Time
	return &t, nil
}

func jsonRawMessage(value string) json.RawMessage {
	if value == "" {
		return nil
	}
	return json.RawMessage(value)
}
