package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dhima/event-trigger-platform/internal/logging"
	"github.com/dhima/event-trigger-platform/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
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
