package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/dhima/event-trigger-platform/internal/events"
	"github.com/dhima/event-trigger-platform/internal/models"
	"github.com/dhima/event-trigger-platform/internal/storage"
	"github.com/dhima/event-trigger-platform/internal/triggers"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Engine periodically scans for triggers that are due to fire and enqueues them.
type Engine struct {
	tick         time.Duration
	db           *storage.MySQLClient
	eventService *events.Service
	logger       *zap.Logger
}

// NewEngine constructs a scheduler with the provided polling cadence and dependencies.
func NewEngine(tick time.Duration, db *storage.MySQLClient, eventService *events.Service, logger *zap.Logger) *Engine {
	return &Engine{
		tick:         tick,
		db:           db,
		eventService: eventService,
		logger:       logger,
	}
}

// Run begins the polling loop, querying due schedules and firing triggers.
// This method runs until the context is cancelled (graceful shutdown).
func (e *Engine) Run(ctx context.Context) error {
	e.logger.Info("scheduler engine started",
		zap.Duration("tick_interval", e.tick))

	ticker := time.NewTicker(e.tick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.processSchedules(ctx)
		case <-ctx.Done():
			e.logger.Info("scheduler engine shutting down")
			return ctx.Err()
		}
	}
}

// processSchedules queries and processes all due schedules in a single tick.
func (e *Engine) processSchedules(ctx context.Context) {
	// Query up to 100 due schedules (configurable limit)
	schedules, err := e.db.GetDueSchedules(ctx, 100)
	if err != nil {
		e.logger.Error("failed to query due schedules", zap.Error(err))
		return
	}

	if len(schedules) == 0 {
		e.logger.Debug("no due schedules found")
		return
	}

	e.logger.Info("processing due schedules",
		zap.Int("count", len(schedules)))

	// Process each schedule
	successCount := 0
	failureCount := 0

	for _, schedule := range schedules {
		if err := e.processSchedule(ctx, schedule); err != nil {
			e.logger.Error("failed to process schedule",
				zap.String("schedule_id", schedule.Schedule.ID),
				zap.String("trigger_id", schedule.Trigger.ID),
				zap.Error(err))
			failureCount++
		} else {
			successCount++
		}
	}

	e.logger.Info("completed processing schedules",
		zap.Int("success", successCount),
		zap.Int("failure", failureCount))
}

// processSchedule handles a single schedule: mark processing, fire trigger, update status, create next schedule.
func (e *Engine) processSchedule(ctx context.Context, scheduleWithTrigger storage.ScheduleWithTrigger) error {
	schedule := scheduleWithTrigger.Schedule
	trigger := scheduleWithTrigger.Trigger

	e.logger.Info("processing schedule",
		zap.String("schedule_id", schedule.ID),
		zap.String("trigger_id", trigger.ID),
		zap.String("trigger_name", trigger.Name),
		zap.String("trigger_type", string(trigger.Type)),
		zap.Time("fire_at", schedule.FireAt))

	// Step 1: Mark schedule as 'processing' to prevent duplicate execution
	err := e.db.UpdateScheduleStatus(ctx, schedule.ID, models.ScheduleStatusProcessing)
	if err != nil {
		return fmt.Errorf("failed to mark schedule as processing: %w", err)
	}

	// Step 2: Extract payload from trigger config
	config, err := storage.ParseTriggerConfig(&trigger)
	if err != nil {
		return fmt.Errorf("failed to parse trigger config: %w", err)
	}

	payload := storage.ExtractPayloadFromConfig(trigger.Type, config)

	// Step 3: Fire trigger via EventService (creates event log + publishes to Kafka)
	eventID, err := e.eventService.FireTrigger(ctx, &trigger, models.EventSourceScheduler, payload, false)
	if err != nil {
		// Even if firing fails, we still mark schedule as completed to avoid retry loops
		// The event log will have execution_status='failure' with error message
		e.logger.Error("failed to fire trigger, marking schedule as completed anyway",
			zap.String("schedule_id", schedule.ID),
			zap.String("trigger_id", trigger.ID),
			zap.Error(err))
	} else {
		e.logger.Info("trigger fired successfully",
			zap.String("schedule_id", schedule.ID),
			zap.String("event_id", eventID),
			zap.String("trigger_id", trigger.ID))
	}

	// Step 4: Mark schedule as 'completed'
	err = e.db.UpdateScheduleStatus(ctx, schedule.ID, models.ScheduleStatusCompleted)
	if err != nil {
		return fmt.Errorf("failed to mark schedule as completed: %w", err)
	}

	// Step 5: Handle trigger type-specific logic
	switch trigger.Type {
	case models.TriggerTypeTimeScheduled:
		// One-time trigger - deactivate after firing
		err = e.db.DeactivateTrigger(ctx, trigger.ID)
		if err != nil {
			e.logger.Error("failed to deactivate one-time trigger",
				zap.String("trigger_id", trigger.ID),
				zap.Error(err))
			return fmt.Errorf("failed to deactivate trigger: %w", err)
		}
		e.logger.Info("deactivated one-time trigger",
			zap.String("trigger_id", trigger.ID))

	case models.TriggerTypeCronScheduled:
		// Recurring trigger - create next schedule if trigger is still active
		if trigger.Status == models.TriggerStatusActive {
			err = e.createNextSchedule(ctx, &trigger)
			if err != nil {
				e.logger.Error("failed to create next schedule for CRON trigger",
					zap.String("trigger_id", trigger.ID),
					zap.Error(err))
				return fmt.Errorf("failed to create next schedule: %w", err)
			}
		} else {
			e.logger.Info("skipping next schedule creation for inactive CRON trigger",
				zap.String("trigger_id", trigger.ID))
		}
	}

	return nil
}

// createNextSchedule calculates and creates the next schedule entry for a CRON trigger.
func (e *Engine) createNextSchedule(ctx context.Context, trigger *models.Trigger) error {
	// Parse CRON config from trigger
	cronConfig, err := triggers.ParseCronConfig(trigger.Config)
	if err != nil {
		return fmt.Errorf("failed to parse cron config: %w", err)
	}

	// Calculate next fire time
	nextFireTime, err := triggers.CalculateNextFireTime(cronConfig.Cron, cronConfig.Timezone, time.Now())
	if err != nil {
		return fmt.Errorf("failed to calculate next fire time: %w", err)
	}

	// Create new schedule entry
	nextSchedule := &models.TriggerSchedule{
		ID:           uuid.New().String(),
		TriggerID:    trigger.ID,
		FireAt:       nextFireTime,
		Status:       models.ScheduleStatusPending,
		AttemptCount: 0,
	}

	err = e.db.CreateNextSchedule(ctx, nextSchedule)
	if err != nil {
		return fmt.Errorf("failed to insert next schedule: %w", err)
	}

	e.logger.Info("created next schedule for CRON trigger",
		zap.String("trigger_id", trigger.ID),
		zap.String("next_schedule_id", nextSchedule.ID),
		zap.Time("next_fire_at", nextFireTime),
		zap.String("cron_expr", cronConfig.Cron),
		zap.String("timezone", cronConfig.Timezone))

	return nil
}
