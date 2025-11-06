package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dhima/event-trigger-platform/internal/logging"
	"github.com/dhima/event-trigger-platform/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type fakeTriggerReader struct {
	resp *models.TriggerResponse
	err  error
}

func (f *fakeTriggerReader) GetTrigger(ctx context.Context, triggerID string) (*models.TriggerResponse, error) {
	return f.resp, f.err
}

type fakeEventFiring struct {
	id  string
	err error
}

func (f *fakeEventFiring) FireTrigger(ctx context.Context, trigger *models.Trigger, source models.EventSource, payload map[string]interface{}, isTestRun bool) (string, error) {
	return f.id, f.err
}

func TestReceiveWebhook_ValidatesSchemaAndFires(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now()
	cfg := map[string]any{
		"schema": map[string]any{
			"type":       "object",
			"properties": map[string]any{"x": map[string]any{"type": "number"}},
			"required":   []any{"x"},
		},
		"endpoint": "https://e",
	}
	cfgBytes, _ := json.Marshal(cfg)
	tr := &models.TriggerResponse{ID: "t1", Name: "wh", Type: models.TriggerTypeWebhook, Status: models.TriggerStatusActive, Config: cfgBytes, CreatedAt: now, UpdatedAt: now}
	h := NewWebhookHandler(&fakeTriggerReader{resp: tr}, &fakeEventFiring{id: "evt-1"}, logging.NewNoOpLogger())
	r := gin.New()
	r.POST("/api/v1/webhook/:trigger_id", h.ReceiveWebhook)

	payload := map[string]any{"x": 1}
	b, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhook/t1", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)
	assert.Contains(t, w.Body.String(), "evt-1")
}

func TestReceiveWebhook_SchemaError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now()
	cfg := map[string]any{
		"schema": map[string]any{
			"type":       "object",
			"properties": map[string]any{"x": map[string]any{"type": "number"}},
			"required":   []any{"x"},
		},
		"endpoint": "https://e",
	}
	cfgBytes, _ := json.Marshal(cfg)
	tr := &models.TriggerResponse{ID: "t1", Name: "wh", Type: models.TriggerTypeWebhook, Status: models.TriggerStatusActive, Config: cfgBytes, CreatedAt: now, UpdatedAt: now}
	h := NewWebhookHandler(&fakeTriggerReader{resp: tr}, &fakeEventFiring{id: "evt-1"}, logging.NewNoOpLogger())
	r := gin.New()
	r.POST("/api/v1/webhook/:trigger_id", h.ReceiveWebhook)

	// missing x
	payload := map[string]any{"y": 1}
	b, _ := json.Marshal(payload)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhook/t1", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestReceiveWebhook_WhenTriggerNotFound_ThenReturns404(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	mockTriggerReader := &fakeTriggerReader{
		resp: nil,
		err:  assert.AnError,
	}
	mockEventFirer := &fakeEventFiring{id: "evt-1"}
	mockHandler := NewWebhookHandler(mockTriggerReader, mockEventFirer, logging.NewNoOpLogger())
	mockRouter := gin.New()
	mockRouter.POST("/api/v1/webhook/:trigger_id", mockHandler.ReceiveWebhook)

	mockRecorder := httptest.NewRecorder()
	mockPayload := map[string]interface{}{"x": 1}
	mockBody, _ := json.Marshal(mockPayload)
	mockRequest := httptest.NewRequest(http.MethodPost, "/api/v1/webhook/nonexistent", bytes.NewReader(mockBody))
	mockRequest.Header.Set("Content-Type", "application/json")

	// Act
	mockRouter.ServeHTTP(mockRecorder, mockRequest)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, mockRecorder.Code)
}

func TestReceiveWebhook_WhenInvalidJSON_ThenReturns400(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	now := time.Now()
	mockConfig := map[string]interface{}{
		"schema": map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{"x": map[string]interface{}{"type": "number"}},
		},
	}
	mockConfigBytes, _ := json.Marshal(mockConfig)
	mockTriggerResponse := &models.TriggerResponse{
		ID:        "t1",
		Name:      "webhook",
		Type:      models.TriggerTypeWebhook,
		Status:    models.TriggerStatusActive,
		Config:    mockConfigBytes,
		CreatedAt: now,
		UpdatedAt: now,
	}
	mockHandler := NewWebhookHandler(&fakeTriggerReader{resp: mockTriggerResponse}, &fakeEventFiring{id: "evt-1"}, logging.NewNoOpLogger())
	mockRouter := gin.New()
	mockRouter.POST("/api/v1/webhook/:trigger_id", mockHandler.ReceiveWebhook)

	mockRecorder := httptest.NewRecorder()
	mockRequest := httptest.NewRequest(http.MethodPost, "/api/v1/webhook/t1", bytes.NewReader([]byte("invalid json")))
	mockRequest.Header.Set("Content-Type", "application/json")

	// Act
	mockRouter.ServeHTTP(mockRecorder, mockRequest)

	// Assert
	assert.Equal(t, http.StatusBadRequest, mockRecorder.Code)
}

func TestReceiveWebhook_WhenFireTriggerFails_ThenReturns500(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	now := time.Now()
	mockConfig := map[string]interface{}{
		"schema": map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{"x": map[string]interface{}{"type": "number"}},
			"required":   []interface{}{"x"},
		},
	}
	mockConfigBytes, _ := json.Marshal(mockConfig)
	mockTriggerResponse := &models.TriggerResponse{
		ID:        "t1",
		Name:      "webhook",
		Type:      models.TriggerTypeWebhook,
		Status:    models.TriggerStatusActive,
		Config:    mockConfigBytes,
		CreatedAt: now,
		UpdatedAt: now,
	}
	mockHandler := NewWebhookHandler(
		&fakeTriggerReader{resp: mockTriggerResponse},
		&fakeEventFiring{err: assert.AnError},
		logging.NewNoOpLogger(),
	)
	mockRouter := gin.New()
	mockRouter.POST("/api/v1/webhook/:trigger_id", mockHandler.ReceiveWebhook)

	mockPayload := map[string]interface{}{"x": 1}
	mockBody, _ := json.Marshal(mockPayload)
	mockRecorder := httptest.NewRecorder()
	mockRequest := httptest.NewRequest(http.MethodPost, "/api/v1/webhook/t1", bytes.NewReader(mockBody))
	mockRequest.Header.Set("Content-Type", "application/json")

	// Act
	mockRouter.ServeHTTP(mockRecorder, mockRequest)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, mockRecorder.Code)
}

func TestReceiveWebhook_WhenTriggerInactive_ThenReturns400(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	now := time.Now()
	mockConfig := map[string]interface{}{
		"schema": map[string]interface{}{
			"type": "object",
		},
	}
	mockConfigBytes, _ := json.Marshal(mockConfig)
	mockTriggerResponse := &models.TriggerResponse{
		ID:        "t1",
		Name:      "webhook",
		Type:      models.TriggerTypeWebhook,
		Status:    models.TriggerStatusInactive,
		Config:    mockConfigBytes,
		CreatedAt: now,
		UpdatedAt: now,
	}
	mockHandler := NewWebhookHandler(&fakeTriggerReader{resp: mockTriggerResponse}, &fakeEventFiring{id: "evt-1"}, logging.NewNoOpLogger())
	mockRouter := gin.New()
	mockRouter.POST("/api/v1/webhook/:trigger_id", mockHandler.ReceiveWebhook)

	mockPayload := map[string]interface{}{"x": 1}
	mockBody, _ := json.Marshal(mockPayload)
	mockRecorder := httptest.NewRecorder()
	mockRequest := httptest.NewRequest(http.MethodPost, "/api/v1/webhook/t1", bytes.NewReader(mockBody))
	mockRequest.Header.Set("Content-Type", "application/json")

	// Act
	mockRouter.ServeHTTP(mockRecorder, mockRequest)

	// Assert
	assert.Equal(t, http.StatusBadRequest, mockRecorder.Code)
}
