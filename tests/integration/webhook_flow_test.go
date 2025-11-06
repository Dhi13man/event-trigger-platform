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

type fakeTriggerReader struct{ resp *models.TriggerResponse; err error }
func (f *fakeTriggerReader) GetTrigger(ctx context.Context, triggerID string) (*models.TriggerResponse, error) { return f.resp, f.err }

type fakeEventFiring struct{ id string; err error }
func (f *fakeEventFiring) FireTrigger(ctx context.Context, trigger *models.Trigger, source models.EventSource, payload map[string]interface{}, isTestRun bool) (string, error) { return f.id, f.err }

func TestWebhookFlow_AcceptsAndQueues(t *testing.T) {
    gin.SetMode(gin.TestMode)
    now := time.Now()
    cfgBytes, _ := json.Marshal(map[string]any{"endpoint":"https://e"})
    tr := &models.TriggerResponse{ID: "t1", Name: "wh", Type: models.TriggerTypeWebhook, Status: models.TriggerStatusActive, Config: cfgBytes, CreatedAt: now, UpdatedAt: now}
    wh := handlers.NewWebhookHandler(&fakeTriggerReader{resp: tr}, &fakeEventFiring{id: "evt-xyz"}, logging.NewNoOpLogger())

    r := gin.New()
    r.POST("/api/v1/webhook/:trigger_id", wh.ReceiveWebhook)

    payload := map[string]any{"k":"v"}
    b,_ := json.Marshal(payload)
    w := httptest.NewRecorder()
    req := httptest.NewRequest(http.MethodPost, "/api/v1/webhook/t1", bytes.NewReader(b))
    req.Header.Set("Content-Type", "application/json")
    r.ServeHTTP(w, req)

    require.Equal(t, http.StatusAccepted, w.Code)
}

