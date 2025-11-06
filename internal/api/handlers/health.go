package handlers

import (
	"github.com/dhima/event-trigger-platform/internal/api/response"
	"github.com/dhima/event-trigger-platform/internal/logging"
	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check requests.
type HealthHandler struct {
	logger logging.Logger
}

// NewHealthHandler creates a new health check handler.
func NewHealthHandler(logger logging.Logger) *HealthHandler {
	return &HealthHandler{logger: logger}
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status  string `json:"status" example:"ok"`
	Service string `json:"service" example:"event-trigger-platform"`
	Version string `json:"version" example:"1.0.0"`
} // @name HealthResponse

// Health godoc
// @Summary Health check endpoint
// @Description Returns the health status of the API service
// @Tags System
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	response.OK(c, HealthResponse{
		Status:  "ok",
		Service: "event-trigger-platform",
		Version: "1.0.0",
	})
}
