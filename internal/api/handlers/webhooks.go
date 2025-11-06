package handlers

import (
	"encoding/json"
	"fmt"

	"github.com/dhima/event-trigger-platform/internal/api/response"
	"github.com/dhima/event-trigger-platform/internal/events"
	"github.com/dhima/event-trigger-platform/internal/logging"
	"github.com/dhima/event-trigger-platform/internal/models"
	"github.com/dhima/event-trigger-platform/internal/triggers"
	"github.com/gin-gonic/gin"
	"github.com/xeipuuv/gojsonschema"
	"go.uber.org/zap"
)

// WebhookHandler handles webhook receiver requests for API triggers.
type WebhookHandler struct {
	triggerService *triggers.Service
	eventService   *events.Service
	logger         logging.Logger
}

// NewWebhookHandler creates a new webhook handler.
func NewWebhookHandler(triggerService *triggers.Service, eventService *events.Service, logger logging.Logger) *WebhookHandler {
	return &WebhookHandler{
		triggerService: triggerService,
		eventService:   eventService,
		logger:         logger.With(zap.String("handler", "webhook")),
	}
}

// ReceiveWebhook godoc
// @Summary Receive webhook payload for webhook trigger
// @Description Receives a payload from external systems, validates it against the trigger's JSON schema, and fires the trigger
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param trigger_id path string true "Trigger ID"
// @Param payload body map[string]interface{} true "Webhook payload (validated against trigger's schema)"
// @Success 202 {object} response.SuccessResponse{data=map[string]string} "Webhook accepted and trigger queued"
// @Failure 400 {object} response.ErrorResponse "Invalid payload or schema validation failed"
// @Failure 404 {object} response.ErrorResponse "Trigger not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /webhook/{trigger_id} [post]
func (h *WebhookHandler) ReceiveWebhook(c *gin.Context) {
	triggerID := c.Param("trigger_id")

	// Parse payload as generic JSON
	var payload map[string]interface{}
	if err := c.ShouldBindJSON(&payload); err != nil {
		h.logger.Warn("invalid webhook payload",
			zap.Error(err),
			zap.String("trigger_id", triggerID),
			zap.String("request_id", response.GetRequestID(c)),
		)
		response.BadRequest(c, "invalid payload", err.Error())
		return
	}

	h.logger.Info("webhook received",
		zap.String("trigger_id", triggerID),
		zap.Int("payload_size", len(payload)),
		zap.String("request_id", response.GetRequestID(c)),
	)

	// Step 1: Fetch trigger from database
	trigger, err := h.triggerService.GetTrigger(c.Request.Context(), triggerID)
	if err != nil {
		h.logger.Error("failed to fetch trigger",
			zap.Error(err),
			zap.String("trigger_id", triggerID),
			zap.String("request_id", response.GetRequestID(c)),
		)
		response.InternalServerError(c, "failed to fetch trigger")
		return
	}

	if trigger == nil {
		h.logger.Warn("trigger not found",
			zap.String("trigger_id", triggerID),
			zap.String("request_id", response.GetRequestID(c)),
		)
		response.NotFound(c, "trigger not found")
		return
	}

	// Verify trigger is a webhook type
	if trigger.Type != models.TriggerTypeWebhook {
		h.logger.Warn("trigger is not a webhook type",
			zap.String("trigger_id", triggerID),
			zap.String("trigger_type", string(trigger.Type)),
			zap.String("request_id", response.GetRequestID(c)),
		)
		response.BadRequest(c, "trigger is not a webhook type", fmt.Sprintf("expected 'webhook', got '%s'", trigger.Type))
		return
	}

	// Verify trigger is active
	if trigger.Status != models.TriggerStatusActive {
		h.logger.Warn("trigger is not active",
			zap.String("trigger_id", triggerID),
			zap.String("trigger_status", string(trigger.Status)),
			zap.String("request_id", response.GetRequestID(c)),
		)
		response.BadRequest(c, "trigger is inactive", "cannot fire inactive trigger")
		return
	}

	// Step 2: Parse trigger config from stored JSON (NOT from request body)
	var webhookConfig models.WebhookTriggerConfig
	if err := json.Unmarshal(trigger.Config, &webhookConfig); err != nil {
		h.logger.Error("failed to parse webhook config",
			zap.Error(err),
			zap.String("trigger_id", triggerID),
			zap.String("request_id", response.GetRequestID(c)),
		)
		response.InternalServerError(c, "failed to parse trigger config")
		return
	}

	// Step 3: Validate payload against JSON schema (if schema is defined)
	if webhookConfig.Schema != nil && len(webhookConfig.Schema) > 0 {
		schemaLoader := gojsonschema.NewGoLoader(webhookConfig.Schema)
		payloadLoader := gojsonschema.NewGoLoader(payload)

		result, err := gojsonschema.Validate(schemaLoader, payloadLoader)
		if err != nil {
			h.logger.Error("failed to validate JSON schema",
				zap.Error(err),
				zap.String("trigger_id", triggerID),
				zap.String("request_id", response.GetRequestID(c)),
			)
			response.InternalServerError(c, "failed to validate payload schema")
			return
		}

		if !result.Valid() {
			// Collect validation errors
			var errorMessages []string
			for _, desc := range result.Errors() {
				errorMessages = append(errorMessages, desc.String())
			}

			h.logger.Warn("payload schema validation failed",
				zap.String("trigger_id", triggerID),
				zap.Strings("errors", errorMessages),
				zap.String("request_id", response.GetRequestID(c)),
			)
			response.BadRequest(c, "payload schema validation failed", fmt.Sprintf("validation errors: %v", errorMessages))
			return
		}

		h.logger.Info("payload schema validation passed",
			zap.String("trigger_id", triggerID),
			zap.String("request_id", response.GetRequestID(c)),
		)
	}

	// Step 4: Fire trigger via EventService (creates event log + publishes to Kafka)
	// Reconstruct trigger from response to pass to event service
	triggerModel := &models.Trigger{
		ID:     trigger.ID,
		Name:   trigger.Name,
		Type:   trigger.Type,
		Status: trigger.Status,
		Config: trigger.Config,
	}

	eventID, err := h.eventService.FireTrigger(c.Request.Context(), triggerModel, models.EventSourceWebhook, payload, false)
	if err != nil {
		h.logger.Error("failed to fire trigger",
			zap.Error(err),
			zap.String("trigger_id", triggerID),
			zap.String("request_id", response.GetRequestID(c)),
		)
		response.InternalServerError(c, "failed to fire trigger")
		return
	}

	h.logger.Info("trigger fired successfully",
		zap.String("trigger_id", triggerID),
		zap.String("event_id", eventID),
		zap.String("request_id", response.GetRequestID(c)),
	)

	// Step 5: Return 202 Accepted with event_id
	response.Success(c, 202, gin.H{
		"event_id":   eventID,
		"trigger_id": triggerID,
	}, "webhook accepted and trigger queued")
}
