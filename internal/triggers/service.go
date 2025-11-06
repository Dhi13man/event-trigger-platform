package triggers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dhima/event-trigger-platform/internal/models"
	"github.com/dhima/event-trigger-platform/internal/storage"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

// Service encapsulates trigger business logic.
type Service struct {
	store *storage.MySQLClient
}

// NewService creates a trigger service.
func NewService(store *storage.MySQLClient) *Service {
	return &Service{
		store: store,
	}
}

// CreateTrigger validates config, persists trigger, and schedules first run if required.
func (s *Service) CreateTrigger(ctx context.Context, req models.CreateTriggerRequest) (*models.TriggerResponse, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, NewValidationError("name is required")
	}

	trigger := models.Trigger{
		ID:     uuid.New().String(),
		Name:   req.Name,
		Type:   req.Type,
		Status: models.TriggerStatusActive,
		Config: req.Config,
	}

	var schedule *models.TriggerSchedule
	var err error
	switch req.Type {
	case models.TriggerTypeWebhook:
		trigger.Config, err = normalizeWebhookConfig(req.Config)
	case models.TriggerTypeTimeScheduled:
		trigger.Config, schedule, err = s.prepareTimeSchedule(trigger.ID, req.Config)
	case models.TriggerTypeCronScheduled:
		trigger.Config, schedule, err = s.prepareCronSchedule(trigger.ID, req.Config)
	default:
		return nil, NewValidationError("unsupported trigger type: %s", req.Type)
	}
	if err != nil {
		return nil, err
	}

	if err = s.store.CreateTrigger(ctx, &trigger, schedule); err != nil {
		return nil, err
	}

	stored, next, err := s.store.GetTrigger(ctx, trigger.ID)
	if err != nil {
		return nil, err
	}

	resp := buildTriggerResponse(ctx, stored, next)
	return &resp, nil
}

// ListTriggers returns triggers response along with pagination metadata.
func (s *Service) ListTriggers(ctx context.Context, query models.ListTriggersQuery) (models.TriggerListResponse, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.Limit <= 0 {
		query.Limit = 20
	}

	triggers, nextRuns, total, err := s.store.ListTriggers(ctx, query)
	if err != nil {
		return models.TriggerListResponse{}, err
	}

	responses := make([]models.TriggerResponse, 0, len(triggers))
	for i := range triggers {
		resp := buildTriggerResponse(ctx, &triggers[i], nextRuns[i])
		responses = append(responses, resp)
	}

	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(query.Limit) - 1) / int64(query.Limit))
	}

	return models.TriggerListResponse{
		Triggers: responses,
		Pagination: models.Pagination{
			CurrentPage:  query.Page,
			PageSize:     query.Limit,
			TotalPages:   totalPages,
			TotalRecords: total,
		},
	}, nil
}

// GetTrigger fetches details for a trigger.
func (s *Service) GetTrigger(ctx context.Context, triggerID string) (*models.TriggerResponse, error) {
	trigger, next, err := s.store.GetTrigger(ctx, triggerID)
	if err != nil {
		if errors.Is(err, storage.ErrTriggerNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("get trigger: %w", err)
	}

	resp := buildTriggerResponse(ctx, trigger, next)
	return &resp, nil
}

// UpdateTrigger updates metadata/config for a trigger.
func (s *Service) UpdateTrigger(ctx context.Context, triggerID string, req models.UpdateTriggerRequest) (*models.TriggerResponse, error) {
	current, _, err := s.store.GetTrigger(ctx, triggerID)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	var schedule *models.TriggerSchedule

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, NewValidationError("name cannot be empty")
		}
		updates["name"] = name
		current.Name = name
	}

	if req.Status != nil {
		updates["status"] = *req.Status
		current.Status = *req.Status
	}

	if len(req.Config) > 0 {
		switch current.Type {
		case models.TriggerTypeWebhook:
			current.Config, err = normalizeWebhookConfig(req.Config)
		case models.TriggerTypeTimeScheduled:
			current.Config, schedule, err = s.prepareTimeSchedule(current.ID, req.Config)
		case models.TriggerTypeCronScheduled:
			current.Config, schedule, err = s.prepareCronSchedule(current.ID, req.Config)
		default:
			err = NewValidationError("unsupported trigger type: %s", current.Type)
		}
		if err != nil {
			return nil, err
		}
		updates["config"] = string(current.Config)
	}

	if len(updates) > 0 {
		if err := s.store.UpdateTrigger(ctx, triggerID, updates); err != nil {
			return nil, err
		}
	}

	// Only update schedules when config changes (not on status changes)
	// The scheduler will check trigger.status before firing, so we don't need to cancel schedules
	if schedule != nil {
		if err := s.store.UpsertTriggerSchedule(ctx, current.ID, schedule); err != nil {
			return nil, err
		}
	}

	refreshed, refreshedNext, err := s.store.GetTrigger(ctx, triggerID)
	if err != nil {
		return nil, err
	}

	resp := buildTriggerResponse(ctx, refreshed, refreshedNext)
	return &resp, nil
}

// DeleteTrigger removes the trigger.
func (s *Service) DeleteTrigger(ctx context.Context, triggerID string) error {
	return s.store.DeleteTrigger(ctx, triggerID)
}

func (s *Service) prepareTimeSchedule(triggerID string, config json.RawMessage) (json.RawMessage, *models.TriggerSchedule, error) {
	var payload struct {
		RunAt      string                 `json:"run_at"`
		Endpoint   string                 `json:"endpoint"`
		HTTPMethod string                 `json:"http_method"`
		Headers    map[string]string      `json:"headers,omitempty"`
		Payload    map[string]interface{} `json:"payload,omitempty"`
		Timezone   string                 `json:"timezone,omitempty"`
	}
	if err := json.Unmarshal(config, &payload); err != nil {
		return nil, nil, fmt.Errorf("invalid time_scheduled config: %w", err)
	}

	if payload.RunAt == "" {
		return nil, nil, NewValidationError("run_at is required for time_scheduled triggers")
	}
	if payload.Endpoint == "" {
		return nil, nil, NewValidationError("endpoint is required for time_scheduled triggers")
	}
	if payload.HTTPMethod == "" {
		payload.HTTPMethod = "POST"
	}

	loc, err := resolveLocation(payload.Timezone)
	if err != nil {
		return nil, nil, err
	}

	runAt, err := time.ParseInLocation(time.RFC3339, payload.RunAt, loc)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid run_at: %w", err)
	}

	if runAt.Before(time.Now().Add(-1 * time.Minute)) {
		return nil, nil, NewValidationError("run_at must be in the future")
	}

	payload.Timezone = loc.String()
	normalized, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal time_scheduled config: %w", err)
	}

	return normalized, &models.TriggerSchedule{
		ID:        uuid.New().String(),
		TriggerID: triggerID,
		FireAt:    runAt.UTC(),
		Status:    models.ScheduleStatusPending,
	}, nil
}

func (s *Service) prepareCronSchedule(triggerID string, config json.RawMessage) (json.RawMessage, *models.TriggerSchedule, error) {
	var payload struct {
		Cron       string                 `json:"cron"`
		Timezone   string                 `json:"timezone,omitempty"`
		Endpoint   string                 `json:"endpoint"`
		HTTPMethod string                 `json:"http_method"`
		Headers    map[string]string      `json:"headers,omitempty"`
		Payload    map[string]interface{} `json:"payload,omitempty"`
	}
	if err := json.Unmarshal(config, &payload); err != nil {
		return nil, nil, fmt.Errorf("invalid cron_scheduled config: %w", err)
	}

	if payload.Cron == "" {
		return nil, nil, NewValidationError("cron expression is required")
	}
	if payload.Endpoint == "" {
		return nil, nil, NewValidationError("endpoint is required for cron_scheduled triggers")
	}
	if payload.HTTPMethod == "" {
		payload.HTTPMethod = "POST"
	}

	loc, err := resolveLocation(payload.Timezone)
	if err != nil {
		return nil, nil, err
	}
	payload.Timezone = loc.String()

	parser := cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(payload.Cron)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	nextRun := schedule.Next(time.Now().In(loc)).UTC()
	normalized, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal cron_scheduled config: %w", err)
	}

	return normalized, &models.TriggerSchedule{
		ID:        uuid.New().String(),
		TriggerID: triggerID,
		FireAt:    nextRun,
		Status:    models.ScheduleStatusPending,
	}, nil
}

func normalizeWebhookConfig(config json.RawMessage) (json.RawMessage, error) {
	var payload struct {
		Schema     map[string]interface{} `json:"schema"`
		Endpoint   string                 `json:"endpoint"`
		HTTPMethod string                 `json:"http_method"`
		Headers    map[string]string      `json:"headers,omitempty"`
	}
	if err := json.Unmarshal(config, &payload); err != nil {
		return nil, fmt.Errorf("invalid webhook config: %w", err)
	}

	if payload.Endpoint == "" {
		return nil, NewValidationError("endpoint is required for webhook triggers")
	}
	if payload.HTTPMethod == "" {
		payload.HTTPMethod = "POST"
	}

	normalized, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal webhook config: %w", err)
	}

	return normalized, nil
}

func buildTriggerResponse(_ context.Context, trigger *models.Trigger, next *time.Time) models.TriggerResponse {
	resp := models.TriggerResponse{
		ID:               trigger.ID,
		Name:             trigger.Name,
		Type:             trigger.Type,
		Status:           trigger.Status,
		Config:           trigger.Config,
		NextScheduledRun: next,
		CreatedAt:        trigger.CreatedAt,
		UpdatedAt:        trigger.UpdatedAt,
	}
	return resp
}

func resolveLocation(tz string) (*time.Location, error) {
	if tz == "" {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, NewValidationError("invalid timezone: %v", err)
	}
	return loc, nil
}
