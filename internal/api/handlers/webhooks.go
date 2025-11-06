package handlers

import (
	"github.com/dhima/event-trigger-platform/internal/api/response"
	"github.com/dhima/event-trigger-platform/internal/logging"
	"github.com/dhima/event-trigger-platform/internal/models"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// WebhookHandler handles webhook receiver requests for API triggers.
type WebhookHandler struct {
	logger logging.Logger
}

// NewWebhookHandler creates a new webhook handler.
func NewWebhookHandler(logger logging.Logger) *WebhookHandler {
	return &WebhookHandler{
		logger: logger.With(zap.String("handler", "webhook")),
	}
}

// ReceiveWebhook godoc
// @Summary Receive webhook payload for API trigger
// @Description Receives a payload from external systems, validates it against the trigger's JSON schema, and fires the trigger
// @Tags Webhooks
// @Accept json
// @Produce json
// @Param trigger_id path string true "Trigger ID"
// @Param payload body models.WebhookPayload true "Webhook payload"
// @Success 202 {object} response.SuccessResponse "Webhook accepted and trigger queued"
// @Failure 400 {object} response.ErrorResponse "Invalid payload or schema validation failed"
// @Failure 404 {object} response.ErrorResponse "Trigger not found"
// @Failure 500 {object} response.ErrorResponse "Internal server error"
// @Router /api/v1/webhook/{trigger_id} [post]
func (h *WebhookHandler) ReceiveWebhook(c *gin.Context) {
	triggerID := c.Param("trigger_id")

	var payload models.WebhookPayload
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
		zap.String("request_id", response.GetRequestID(c)),
	)

	// TODO: Implement webhook processing:
	// 1. Fetch trigger from database
	// 2. Validate payload against trigger's JSON schema
	// 3. Publish event to Kafka
	// 4. Create event log in database
	// 5. Return 202 Accepted

	response.NotFound(c, "trigger not found")
}
