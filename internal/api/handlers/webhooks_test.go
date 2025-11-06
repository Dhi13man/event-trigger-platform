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
