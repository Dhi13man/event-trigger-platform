package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/dhima/event-trigger-platform/internal/api/response"
	"github.com/dhima/event-trigger-platform/internal/logging"
	"github.com/dhima/event-trigger-platform/internal/models"
	"github.com/dhima/event-trigger-platform/internal/storage"
	"github.com/dhima/event-trigger-platform/internal/triggers"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TriggerHandler handles trigger management requests.
type TriggerHandler struct {
	logger  logging.Logger
	service *triggers.Service
}

// NewTriggerHandler creates a new trigger handler.
func NewTriggerHandler(logger logging.Logger, service *triggers.Service) *TriggerHandler {
	return &TriggerHandler{
		logger:  logger.With(zap.String("handler", "trigger")),
		service: service,
	}
}

// CreateTrigger godoc
// @Summary Create a new trigger
// @Description Creates a trigger with configuration. Webhook triggers return a webhook URL.
// @Tags Triggers
// @Accept json
// @Produce json
// @Param trigger body models.CreateTriggerRequest true "Trigger configuration"
// @Success 201 {object} models.TriggerResponse
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/triggers [post]
func (h *TriggerHandler) CreateTrigger(c *gin.Context) {
	var req models.CreateTriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid create trigger request",
			zap.Error(err),
			zap.String("request_id", response.GetRequestID(c)),
		)
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}

	result, err := h.service.CreateTrigger(c.Request.Context(), req)
	if h.handleServiceError(c, err, "create trigger") {
		return
	}

	h.decorateWebhookURL(c, result)

	h.logger.Info("trigger created",
		zap.String("trigger_id", result.ID),
		zap.String("type", string(result.Type)),
		zap.String("request_id", response.GetRequestID(c)),
	)

	response.Created(c, result, "trigger created successfully")
}

// ListTriggers godoc
// @Summary List all triggers
// @Description Retrieves a list of triggers with optional filtering and pagination
// @Tags Triggers
// @Produce json
// @Param type query string false "Filter by trigger type" Enums(webhook, time_scheduled, cron_scheduled)
// @Param status query string false "Filter by trigger status" Enums(active, inactive)
// @Param page query int false "Page number" default(1) minimum(1)
// @Param limit query int false "Items per page" default(20) minimum(1) maximum(100)
// @Success 200 {object} models.TriggerListResponse
// @Failure 400 {object} response.ErrorResponse "Invalid query parameters"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/triggers [get]
func (h *TriggerHandler) ListTriggers(c *gin.Context) {
	var query models.ListTriggersQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		h.logger.Warn("invalid list triggers query",
			zap.Error(err),
			zap.String("request_id", response.GetRequestID(c)),
		)
		response.BadRequest(c, "invalid query parameters", err.Error())
		return
	}

	result, err := h.service.ListTriggers(c.Request.Context(), query)
	if h.handleServiceError(c, err, "list triggers") {
		return
	}

	for i := range result.Triggers {
		h.decorateWebhookURL(c, &result.Triggers[i])
	}

	response.Success(c, http.StatusOK, result, "")
}

// GetTrigger godoc
// @Summary Get trigger details
// @Description Retrieves details of a specific trigger by ID
// @Tags Triggers
// @Produce json
// @Param id path string true "Trigger ID"
// @Success 200 {object} models.TriggerResponse
// @Failure 404 {object} response.ErrorResponse "Trigger not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/triggers/{id} [get]
func (h *TriggerHandler) GetTrigger(c *gin.Context) {
	triggerID := c.Param("id")
	result, err := h.service.GetTrigger(c.Request.Context(), triggerID)
	if h.handleServiceError(c, err, "get trigger") {
		return
	}

	h.decorateWebhookURL(c, result)
	response.OK(c, result)
}

// UpdateTrigger godoc
// @Summary Update a trigger
// @Description Updates an existing trigger's metadata or configuration. Only affects future trigger firings.
// @Tags Triggers
// @Accept json
// @Produce json
// @Param id path string true "Trigger ID"
// @Param trigger body models.UpdateTriggerRequest true "Updated trigger data"
// @Success 200 {object} models.TriggerResponse
// @Failure 400 {object} response.ErrorResponse "Invalid request"
// @Failure 404 {object} response.ErrorResponse "Trigger not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/triggers/{id} [put]
func (h *TriggerHandler) UpdateTrigger(c *gin.Context) {
	triggerID := c.Param("id")

	var req models.UpdateTriggerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("invalid update trigger request",
			zap.Error(err),
			zap.String("trigger_id", triggerID),
			zap.String("request_id", response.GetRequestID(c)),
		)
		response.BadRequest(c, "invalid request body", err.Error())
		return
	}

	result, err := h.service.UpdateTrigger(c.Request.Context(), triggerID, req)
	if h.handleServiceError(c, err, "update trigger") {
		return
	}

	h.decorateWebhookURL(c, result)
	response.OK(c, result)
}

// DeleteTrigger godoc
// @Summary Delete a trigger
// @Description Deletes a trigger. Event logs are not deleted and follow their retention lifecycle.
// @Tags Triggers
// @Produce json
// @Param id path string true "Trigger ID"
// @Success 204 "Trigger deleted successfully"
// @Failure 404 {object} response.ErrorResponse "Trigger not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/triggers/{id} [delete]
func (h *TriggerHandler) DeleteTrigger(c *gin.Context) {
	triggerID := c.Param("id")

	if h.handleServiceError(c, h.service.DeleteTrigger(c.Request.Context(), triggerID), "delete trigger") {
		return
	}

	response.NoContent(c)
}

// TestTrigger godoc
// @Summary Test a trigger (manual/test run)
// @Description Fires a trigger once for testing. Creates an event log with is_test_run=true.
// @Tags Triggers
// @Produce json
// @Param id path string true "Trigger ID"
// @Success 202 {object} response.SuccessResponse "Test trigger fired successfully"
// @Failure 404 {object} response.ErrorResponse "Trigger not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/triggers/{id}/test [post]
func (h *TriggerHandler) TestTrigger(c *gin.Context) {
	// TODO: Implement actual test trigger firing via worker queue.
	response.NotFound(c, "trigger not found")
}

func (h *TriggerHandler) handleServiceError(c *gin.Context, err error, operation string) bool {
	if err == nil {
		return false
	}

	var validationErr triggers.ValidationError
	switch {
	case errors.As(err, &validationErr):
		response.BadRequest(c, "validation failed", validationErr.Error())
	case errors.Is(err, storage.ErrTriggerNotFound):
		response.NotFound(c, "trigger not found")
	default:
		h.logger.Error(operation+" failed",
			zap.Error(err),
			zap.String("request_id", response.GetRequestID(c)),
		)
		response.InternalServerError(c, "internal server error")
	}
	return true
}

func (h *TriggerHandler) decorateWebhookURL(c *gin.Context, resp *models.TriggerResponse) {
	if resp == nil || resp.Type != models.TriggerTypeWebhook {
		return
	}

	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	} else if forwarded := c.GetHeader("X-Forwarded-Proto"); forwarded != "" {
		scheme = forwarded
	}

	host := c.Request.Host
	if forwardedHost := c.GetHeader("X-Forwarded-Host"); forwardedHost != "" {
		host = forwardedHost
	}

	host = strings.TrimSpace(host)
	if host == "" {
		return
	}

	resp.WebhookURL = scheme + "://" + host + "/api/v1/webhook/" + resp.ID
}
