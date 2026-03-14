package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dhima/event-trigger-platform/internal/logging"
	"github.com/gin-gonic/gin"
)

func TestNewMetricsHandler_WhenCreated_ThenReturnsHandler(t *testing.T) {
	// Arrange
	logger := logging.NewNoOpLogger()

	// Act
	handler := NewMetricsHandler(logger)

	// Assert
	if handler == nil {
		t.Fatal("expected handler to be non-nil")
	}
	if handler.logger == nil {
		t.Fatal("expected logger to be non-nil")
	}
}

func TestMetrics_WhenCalled_ThenReturns200WithMetrics(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)

	logger := logging.NewNoOpLogger()
	handler := NewMetricsHandler(logger)

	router.GET("/metrics", handler.Metrics)
	c.Request = httptest.NewRequest(http.MethodGet, "/metrics", nil)

	// Act
	router.ServeHTTP(w, c.Request)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var responseWrapper struct {
		Data MetricsResponse `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &responseWrapper)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	response := responseWrapper.Data
	// Verify response structure is correct (values are 0 as per TODO stub)
	if response.PublishedEventsCount != 0 {
		t.Errorf("expected PublishedEventsCount to be 0, got %d", response.PublishedEventsCount)
	}
	if response.EventsActiveCount != 0 {
		t.Errorf("expected EventsActiveCount to be 0, got %d", response.EventsActiveCount)
	}
	if response.EventsArchivedCount != 0 {
		t.Errorf("expected EventsArchivedCount to be 0, got %d", response.EventsArchivedCount)
	}
	if response.TriggerCountScheduled != 0 {
		t.Errorf("expected TriggerCountScheduled to be 0, got %d", response.TriggerCountScheduled)
	}
	if response.TriggerCountAPI != 0 {
		t.Errorf("expected TriggerCountAPI to be 0, got %d", response.TriggerCountAPI)
	}
}
