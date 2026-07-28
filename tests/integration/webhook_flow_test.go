//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dhima/event-trigger-platform/internal/api/handlers"
	"github.com/dhima/event-trigger-platform/internal/logging"
	"github.com/dhima/event-trigger-platform/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type fakeTriggerReader struct {
	resp *models.TriggerResponse
	err  error
}

func (f *fakeTriggerReader) GetTrigger(ctx context.Context, triggerID string) (*models.TriggerResponse, error) {
	return f.resp, f.err
}

type fakeEventFiring struct {
	id        string
	err       error
	calls     int
	trigger   *models.Trigger
	source    models.EventSource
	payload   map[string]interface{}
	isTestRun bool
}

func (f *fakeEventFiring) FireTrigger(ctx context.Context, trigger *models.Trigger, source models.EventSource, payload map[string]interface{}, isTestRun bool) (string, error) {
	f.calls++
	f.trigger = trigger
	f.source = source
	f.payload = payload
	f.isTestRun = isTestRun
	return f.id, f.err
}

func TestWebhookFlow_AcceptsAndQueues(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, time.July, 28, 9, 30, 0, 0, time.UTC)
	cfgBytes, err := json.Marshal(map[string]any{"endpoint": "https://e"})
	require.NoError(t, err)
	tr := &models.TriggerResponse{ID: "t1", Name: "wh", Type: models.TriggerTypeWebhook, Status: models.TriggerStatusActive, Config: cfgBytes, CreatedAt: now, UpdatedAt: now}
	firer := &fakeEventFiring{id: "evt-xyz"}
	wh := handlers.NewWebhookHandler(&fakeTriggerReader{resp: tr}, firer, logging.NewNoOpLogger())

	r := gin.New()
	r.POST("/api/v1/webhook/:trigger_id", wh.ReceiveWebhook)

	payload := map[string]any{"k": "v"}
	b, err := json.Marshal(payload)
	require.NoError(t, err)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhook/t1", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	require.Equal(t, http.StatusAccepted, w.Code)
	require.Equal(t, 1, firer.calls)
	require.Equal(t, "t1", firer.trigger.ID)
	require.Equal(t, models.EventSourceWebhook, firer.source)
	require.Equal(t, payload, firer.payload)
	require.False(t, firer.isTestRun)
}
