package handlers

import (
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
	"github.com/stretchr/testify/require"
)

type fakeEventQuerySvc struct {
	list     []models.EventLog
	getEvent *models.EventLog
	getErr   error
}

func (f *fakeEventQuerySvc) QueryEvents(ctx context.Context, query models.ListEventsQuery) ([]models.EventLog, models.Pagination, error) {
	return f.list, models.Pagination{CurrentPage: 1, PageSize: 20, TotalPages: 1, TotalRecords: int64(len(f.list))}, nil
}
func (f *fakeEventQuerySvc) GetEvent(ctx context.Context, eventID string) (*models.EventLog, error) {
	return f.getEvent, f.getErr
}

func TestListEvents_DefaultsActive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewEventHandler(&fakeEventQuerySvc{list: []models.EventLog{}}, logging.NewNoOpLogger())
	r := gin.New()
	r.GET("/api/v1/events", h.ListEvents)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListEvents_WhenServiceReturnsEvent_ThenPreservesEventFields(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	triggerID := "trigger-123"
	errorMessage := "delivery failed"
	firedAt := time.Date(2026, time.July, 28, 9, 30, 0, 0, time.UTC)
	createdAt := firedAt.Add(time.Second)
	event := models.EventLog{
		ID:              "event-123",
		TriggerID:       &triggerID,
		TriggerType:     models.TriggerTypeWebhook,
		FiredAt:         firedAt,
		Payload:         json.RawMessage(`{"status":"failed"}`),
		Source:          models.EventSourceWebhook,
		ExecutionStatus: models.ExecutionStatusFailure,
		ErrorMessage:    &errorMessage,
		RetentionStatus: models.RetentionStatusActive,
		IsTestRun:       true,
		CreatedAt:       createdAt,
	}
	handler := NewEventHandler(&fakeEventQuerySvc{list: []models.EventLog{event}}, logging.NewNoOpLogger())
	router := gin.New()
	router.GET("/api/v1/events", handler.ListEvents)

	// Act
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	router.ServeHTTP(recorder, request)

	// Assert
	require.Equal(t, http.StatusOK, recorder.Code)
	var body struct {
		Data models.EventLogListResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	require.Len(t, body.Data.Events, 1)
	got := body.Data.Events[0]
	assert.Equal(t, event.ID, got.ID)
	assert.Equal(t, event.TriggerID, got.TriggerID)
	assert.Equal(t, event.TriggerType, got.TriggerType)
	assert.Equal(t, event.FiredAt, got.FiredAt)
	assert.JSONEq(t, string(event.Payload), string(got.Payload))
	assert.Equal(t, event.Source, got.Source)
	assert.Equal(t, event.ExecutionStatus, got.ExecutionStatus)
	assert.Equal(t, event.ErrorMessage, got.ErrorMessage)
	assert.Equal(t, event.RetentionStatus, got.RetentionStatus)
	assert.Equal(t, event.IsTestRun, got.IsTestRun)
	assert.Equal(t, event.CreatedAt, got.CreatedAt)
}

func TestGetEvent_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	evt := &models.EventLog{ID: "evt-123", Source: models.EventSourceWebhook}
	svc := &fakeEventQuerySvc{getEvent: evt}
	h := NewEventHandler(svc, logging.NewNoOpLogger())
	r := gin.New()
	r.GET("/api/v1/events/:id", h.GetEvent)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/evt-123", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "evt-123")
}

func TestGetEvent_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Events handler checks for nil event (line 153), not error type
	// So fake should return (nil, nil) for NotFound case
	svc := &fakeEventQuerySvc{getEvent: nil, getErr: nil}
	h := NewEventHandler(svc, logging.NewNoOpLogger())
	r := gin.New()
	r.GET("/api/v1/events/:id", h.GetEvent)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/events/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
