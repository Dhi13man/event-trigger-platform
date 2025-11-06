package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dhima/event-trigger-platform/internal/logging"
	"github.com/gin-gonic/gin"
)

func TestNewHealthHandler_WhenCreated_ThenReturnsHandler(t *testing.T) {
	// Arrange
	logger := logging.NewNoOpLogger()

	// Act
	handler := NewHealthHandler(logger)

	// Assert
	if handler == nil {
		t.Fatal("expected handler to be non-nil")
	}
	if handler.logger == nil {
		t.Fatal("expected logger to be non-nil")
	}
}

func TestHealth_WhenCalled_ThenReturns200WithHealthStatus(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)

	logger := logging.NewNoOpLogger()
	handler := NewHealthHandler(logger)

	router.GET("/health", handler.Health)
	c.Request = httptest.NewRequest(http.MethodGet, "/health", nil)

	// Act
	router.ServeHTTP(w, c.Request)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var responseWrapper struct {
		Data HealthResponse `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &responseWrapper)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	response := responseWrapper.Data
	if response.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", response.Status)
	}
	if response.Service != "event-trigger-platform" {
		t.Errorf("expected service 'event-trigger-platform', got '%s'", response.Service)
	}
	if response.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", response.Version)
	}
}
