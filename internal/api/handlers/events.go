package handlers

import (
	"context"

	"github.com/dhima/event-trigger-platform/internal/api/response"
	"github.com/dhima/event-trigger-platform/internal/logging"
	"github.com/dhima/event-trigger-platform/internal/models"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// EventHandler handles event log query requests.
type EventHandler struct {
	eventService EventQueryService
	logger       logging.Logger
}

// EventQueryService defines query methods used by the handler.
type EventQueryService interface {
	QueryEvents(ctx context.Context, query models.ListEventsQuery) ([]models.EventLog, models.Pagination, error)
	GetEvent(ctx context.Context, eventID string) (*models.EventLog, error)
}

// NewEventHandler creates a new event handler.
func NewEventHandler(eventService EventQueryService, logger logging.Logger) *EventHandler {
	return &EventHandler{
		eventService: eventService,
		logger:       logger.With(zap.String("handler", "event")),
	}
}

// ListEvents godoc
// @Summary List event logs
// @Description Retrieves event logs with filtering and pagination. By default shows only active events (last 2 hours).
// @Tags Events
// @Produce json
// @Param trigger_id query string false "Filter by trigger ID"
// @Param retention_status query string false "Filter by retention status" Enums(active, archived) default(active)
// @Param execution_status query string false "Filter by execution status" Enums(success, failure)
// @Param source query string false "Filter by event source" Enums(webhook, scheduler, manual-test)
// @Param page query int false "Page number" default(1) minimum(1)
// @Param limit query int false "Items per page" default(20) minimum(1) maximum(100)
// @Success 200 {object} models.EventLogListResponse
// @Failure 400 {object} response.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /events [get]
func (h *EventHandler) ListEvents(c *gin.Context) {
	var query models.ListEventsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.logger.Warn("invalid list events query",
			zap.Error(err),
			zap.String("request_id", response.GetRequestID(c)),
		)
		response.BadRequest(c, "invalid query parameters", err.Error())
		return
	}

	// Set defaults
	if query.Page == 0 {
		query.Page = 1
	}
	if query.Limit == 0 {
		query.Limit = 20
	}
	if query.RetentionStatus == "" {
		query.RetentionStatus = "active" // Default to active events
	}

	h.logger.Info("listing events",
		zap.String("trigger_id", query.TriggerID),
		zap.String("retention_status", query.RetentionStatus),
		zap.String("execution_status", query.ExecutionStatus),
		zap.String("source", query.Source),
		zap.Int("page", query.Page),
		zap.Int("limit", query.Limit),
		zap.String("request_id", response.GetRequestID(c)),
	)

	// Query events from service
	eventLogs, pagination, err := h.eventService.QueryEvents(c.Request.Context(), query)
	if err != nil {
		h.logger.Error("failed to query events",
			zap.Error(err),
			zap.String("request_id", response.GetRequestID(c)),
		)
		response.InternalServerError(c, "failed to query events")
		return
	}

	// Convert to response format
	eventResponses := make([]models.EventLogResponse, len(eventLogs))
	for i, event := range eventLogs {
		eventResponses[i] = models.EventLogResponse{
			ID:              event.ID,
			TriggerID:       event.TriggerID,
			TriggerType:     event.TriggerType,
			FiredAt:         event.FiredAt,
			Payload:         event.Payload,
			Source:          event.Source,
			ExecutionStatus: event.ExecutionStatus,
			ErrorMessage:    event.ErrorMessage,
			RetentionStatus: event.RetentionStatus,
			IsTestRun:       event.IsTestRun,
			CreatedAt:       event.CreatedAt,
		}
	}

	result := models.EventLogListResponse{
		Events:     eventResponses,
		Pagination: pagination,
	}

	h.logger.Info("events listed successfully",
		zap.Int("count", len(eventResponses)),
		zap.Int64("total", pagination.TotalRecords),
		zap.String("request_id", response.GetRequestID(c)),
	)

	response.OK(c, result)
}

// GetEvent godoc
// @Summary Get event log details
// @Description Retrieves details of a specific event log by ID, including full payload and error message if failed
// @Tags Events
// @Produce json
// @Param id path string true "Event ID"
// @Success 200 {object} models.EventLogResponse
// @Failure 404 {object} response.ErrorResponse "Event not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /events/{id} [get]
func (h *EventHandler) GetEvent(c *gin.Context) {
	eventID := c.Param("id")

	h.logger.Info("getting event",
		zap.String("event_id", eventID),
		zap.String("request_id", response.GetRequestID(c)),
	)

	// Get event from service
	event, err := h.eventService.GetEvent(c.Request.Context(), eventID)
	if err != nil {
		h.logger.Error("failed to get event",
			zap.Error(err),
			zap.String("event_id", eventID),
			zap.String("request_id", response.GetRequestID(c)),
		)
		response.InternalServerError(c, "failed to get event")
		return
	}

	if event == nil {
		h.logger.Warn("event not found",
			zap.String("event_id", eventID),
			zap.String("request_id", response.GetRequestID(c)),
		)
		response.NotFound(c, "event not found")
		return
	}

	// Convert to response format
	eventResponse := models.EventLogResponse{
		ID:              event.ID,
		TriggerID:       event.TriggerID,
		TriggerType:     event.TriggerType,
		FiredAt:         event.FiredAt,
		Payload:         event.Payload,
		Source:          event.Source,
		ExecutionStatus: event.ExecutionStatus,
		ErrorMessage:    event.ErrorMessage,
		RetentionStatus: event.RetentionStatus,
		IsTestRun:       event.IsTestRun,
		CreatedAt:       event.CreatedAt,
	}

	h.logger.Info("event retrieved successfully",
		zap.String("event_id", eventID),
		zap.String("request_id", response.GetRequestID(c)),
	)

	response.OK(c, eventResponse)
}
