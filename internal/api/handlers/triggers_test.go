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
	"github.com/dhima/event-trigger-platform/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type fakeTriggerSvc struct {
	createResp *models.TriggerResponse
	createErr  error
	listResp   models.TriggerListResponse
	listErr    error
	getResp    *models.TriggerResponse
	getErr     error
	updateResp *models.TriggerResponse
	updateErr  error
	deleteErr  error
}

func (f *fakeTriggerSvc) CreateTrigger(ctx context.Context, req models.CreateTriggerRequest) (*models.TriggerResponse, error) {
	return f.createResp, f.createErr
}
func (f *fakeTriggerSvc) ListTriggers(ctx context.Context, query models.ListTriggersQuery) (models.TriggerListResponse, error) {
	if f.listErr != nil {
		return models.TriggerListResponse{}, f.listErr
	}
	if f.listResp.Triggers != nil {
		return f.listResp, nil
	}
	return models.TriggerListResponse{Triggers: []models.TriggerResponse{}, Pagination: models.Pagination{}}, nil
}
func (f *fakeTriggerSvc) GetTrigger(ctx context.Context, id string) (*models.TriggerResponse, error) {
	return f.getResp, f.getErr
}
func (f *fakeTriggerSvc) UpdateTrigger(ctx context.Context, id string, req models.UpdateTriggerRequest) (*models.TriggerResponse, error) {
	return f.updateResp, f.updateErr
}
func (f *fakeTriggerSvc) DeleteTrigger(ctx context.Context, id string) error { return f.deleteErr }

func TestCreateTrigger_BadJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewTriggerHandler(logging.NewNoOpLogger(), &fakeTriggerSvc{})
	r := gin.New()
	r.POST("/api/v1/triggers", h.CreateTrigger)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/triggers", bytes.NewBufferString("{"))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCreateTrigger_WebhookURLDecoration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now()
	svc := &fakeTriggerSvc{createResp: &models.TriggerResponse{ID: "123", Name: "wh", Type: models.TriggerTypeWebhook, CreatedAt: now, UpdatedAt: now}}
	h := NewTriggerHandler(logging.NewNoOpLogger(), svc)
	r := gin.New()
	r.POST("/api/v1/triggers", h.CreateTrigger)

	body := map[string]any{"name": "wh", "type": "webhook", "config": map[string]any{"endpoint": "https://e"}}
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/triggers", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Host = "example.com"
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "\"webhook_url\":\"http://example.com/api/v1/webhook/123\"")
}

func TestListTriggers_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now()
	listResp := models.TriggerListResponse{
		Triggers: []models.TriggerResponse{
			{ID: "1", Name: "t1", Type: models.TriggerTypeWebhook, CreatedAt: now, UpdatedAt: now},
			{ID: "2", Name: "t2", Type: models.TriggerTypeCronScheduled, CreatedAt: now, UpdatedAt: now},
		},
		Pagination: models.Pagination{CurrentPage: 1, PageSize: 10, TotalPages: 1, TotalRecords: 2},
	}
	svc := &fakeTriggerSvc{listResp: listResp}
	h := NewTriggerHandler(logging.NewNoOpLogger(), svc)
	r := gin.New()
	r.GET("/api/v1/triggers", h.ListTriggers)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/triggers?page=1&limit=10", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"id\":\"1\"")
	assert.Contains(t, w.Body.String(), "\"id\":\"2\"")
}

func TestGetTrigger_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now()
	getResp := &models.TriggerResponse{ID: "123", Name: "test", Type: models.TriggerTypeWebhook, CreatedAt: now, UpdatedAt: now}
	svc := &fakeTriggerSvc{getResp: getResp}
	h := NewTriggerHandler(logging.NewNoOpLogger(), svc)
	r := gin.New()
	r.GET("/api/v1/triggers/:id", h.GetTrigger)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/triggers/123", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"id\":\"123\"")
}

func TestGetTrigger_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeTriggerSvc{getErr: storage.ErrTriggerNotFound}
	h := NewTriggerHandler(logging.NewNoOpLogger(), svc)
	r := gin.New()
	r.GET("/api/v1/triggers/:id", h.GetTrigger)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/triggers/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdateTrigger_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Now()
	updateResp := &models.TriggerResponse{ID: "123", Name: "updated", Type: models.TriggerTypeWebhook, CreatedAt: now, UpdatedAt: now}
	svc := &fakeTriggerSvc{updateResp: updateResp}
	h := NewTriggerHandler(logging.NewNoOpLogger(), svc)
	r := gin.New()
	r.PUT("/api/v1/triggers/:id", h.UpdateTrigger)

	body := map[string]any{"name": "updated"}
	b, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/triggers/123", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "\"name\":\"updated\"")
}

func TestDeleteTrigger_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeTriggerSvc{deleteErr: nil}
	h := NewTriggerHandler(logging.NewNoOpLogger(), svc)
	r := gin.New()
	r.DELETE("/api/v1/triggers/:id", h.DeleteTrigger)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/triggers/123", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestDeleteTrigger_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := &fakeTriggerSvc{deleteErr: storage.ErrTriggerNotFound}
	h := NewTriggerHandler(logging.NewNoOpLogger(), svc)
	r := gin.New()
	r.DELETE("/api/v1/triggers/:id", h.DeleteTrigger)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/triggers/nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestTestTrigger_WhenCalled_ThenReturnsNotImplemented(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	mockSvc := &fakeTriggerSvc{}
	mockHandler := NewTriggerHandler(logging.NewNoOpLogger(), mockSvc)
	mockRouter := gin.New()
	mockRouter.POST("/api/v1/triggers/:id/test", mockHandler.TestTrigger)

	mockRecorder := httptest.NewRecorder()
	mockRequest := httptest.NewRequest(http.MethodPost, "/api/v1/triggers/test-id-123/test", nil)

	// Act
	mockRouter.ServeHTTP(mockRecorder, mockRequest)

	// Assert
	assert.Equal(t, http.StatusNotImplemented, mockRecorder.Code)
}

func TestListTriggers_WhenPageSizeInvalid_ThenUsesDefault(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	mockTriggers := []models.TriggerResponse{
		{ID: "1", Name: "Test Trigger", Type: models.TriggerTypeWebhook},
	}
	mockSvc := &fakeTriggerSvc{
		listResp: models.TriggerListResponse{
			Triggers: mockTriggers,
			Pagination: models.Pagination{
				CurrentPage:  1,
				PageSize:     10,
				TotalPages:   1,
				TotalRecords: 1,
			},
		},
	}
	mockHandler := NewTriggerHandler(logging.NewNoOpLogger(), mockSvc)
	mockRouter := gin.New()
	mockRouter.GET("/api/v1/triggers", mockHandler.ListTriggers)

	mockRecorder := httptest.NewRecorder()
	mockRequest := httptest.NewRequest(http.MethodGet, "/api/v1/triggers?page_size=-1", nil)

	// Act
	mockRouter.ServeHTTP(mockRecorder, mockRequest)

	// Assert
	assert.Equal(t, http.StatusOK, mockRecorder.Code)
}

func TestListTriggers_WhenServiceReturnsError_ThenReturns500(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	mockSvc := &fakeTriggerSvc{
		listErr: assert.AnError,
	}
	mockHandler := NewTriggerHandler(logging.NewNoOpLogger(), mockSvc)
	mockRouter := gin.New()
	mockRouter.GET("/api/v1/triggers", mockHandler.ListTriggers)

	mockRecorder := httptest.NewRecorder()
	mockRequest := httptest.NewRequest(http.MethodGet, "/api/v1/triggers", nil)

	// Act
	mockRouter.ServeHTTP(mockRecorder, mockRequest)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, mockRecorder.Code)
}

func TestUpdateTrigger_WhenServiceReturnsError_ThenReturns500(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	mockSvc := &fakeTriggerSvc{
		updateErr: assert.AnError,
	}
	mockHandler := NewTriggerHandler(logging.NewNoOpLogger(), mockSvc)
	mockRouter := gin.New()
	mockRouter.PUT("/api/v1/triggers/:id", mockHandler.UpdateTrigger)

	mockStatus := models.TriggerStatusActive
	mockRequestBody := models.UpdateTriggerRequest{
		Name:   stringPtr("Updated Trigger"),
		Status: &mockStatus,
	}
	mockBody, _ := json.Marshal(mockRequestBody)

	mockRecorder := httptest.NewRecorder()
	mockRequest := httptest.NewRequest(http.MethodPut, "/api/v1/triggers/123", bytes.NewBuffer(mockBody))
	mockRequest.Header.Set("Content-Type", "application/json")

	// Act
	mockRouter.ServeHTTP(mockRecorder, mockRequest)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, mockRecorder.Code)
}

func TestUpdateTrigger_WhenNotFound_ThenReturns404(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	mockSvc := &fakeTriggerSvc{
		updateErr: storage.ErrTriggerNotFound,
	}
	mockHandler := NewTriggerHandler(logging.NewNoOpLogger(), mockSvc)
	mockRouter := gin.New()
	mockRouter.PUT("/api/v1/triggers/:id", mockHandler.UpdateTrigger)

	mockRequestBody := models.UpdateTriggerRequest{
		Name: stringPtr("Updated Trigger"),
	}
	mockBody, _ := json.Marshal(mockRequestBody)

	mockRecorder := httptest.NewRecorder()
	mockRequest := httptest.NewRequest(http.MethodPut, "/api/v1/triggers/nonexistent", bytes.NewBuffer(mockBody))
	mockRequest.Header.Set("Content-Type", "application/json")

	// Act
	mockRouter.ServeHTTP(mockRecorder, mockRequest)

	// Assert
	assert.Equal(t, http.StatusNotFound, mockRecorder.Code)
}

func TestDecorateWebhookURL_WhenNilResponse_ThenDoesNothing(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	mockRecorder := httptest.NewRecorder()
	mockContext, _ := gin.CreateTestContext(mockRecorder)
	mockRequest := httptest.NewRequest(http.MethodGet, "/test", nil)
	mockContext.Request = mockRequest

	mockHandler := NewTriggerHandler(logging.NewNoOpLogger(), &fakeTriggerSvc{})

	// Act (should not panic)
	mockHandler.decorateWebhookURL(mockContext, nil)
}

func TestDecorateWebhookURL_WhenNotWebhookType_ThenDoesNothing(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	mockRecorder := httptest.NewRecorder()
	mockContext, _ := gin.CreateTestContext(mockRecorder)
	mockRequest := httptest.NewRequest(http.MethodGet, "/test", nil)
	mockContext.Request = mockRequest

	mockHandler := NewTriggerHandler(logging.NewNoOpLogger(), &fakeTriggerSvc{})
	mockResponse := &models.TriggerResponse{
		ID:   "123",
		Type: models.TriggerTypeTimeScheduled,
	}

	// Act
	mockHandler.decorateWebhookURL(mockContext, mockResponse)

	// Assert - WebhookURL should remain empty
	if mockResponse.WebhookURL != "" {
		t.Errorf("expected WebhookURL to be empty, got %s", mockResponse.WebhookURL)
	}
}

func TestDecorateWebhookURL_WhenHTTPSRequest_ThenUsesHTTPS(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	mockRecorder := httptest.NewRecorder()
	mockContext, _ := gin.CreateTestContext(mockRecorder)
	mockRequest := httptest.NewRequest(http.MethodGet, "/test", nil)
	mockRequest.Host = "example.com"
	mockRequest.Header.Set("X-Forwarded-Proto", "https")
	mockContext.Request = mockRequest

	mockHandler := NewTriggerHandler(logging.NewNoOpLogger(), &fakeTriggerSvc{})
	mockResponse := &models.TriggerResponse{
		ID:   "trigger-123",
		Type: models.TriggerTypeWebhook,
	}

	// Act
	mockHandler.decorateWebhookURL(mockContext, mockResponse)

	// Assert
	assert.Contains(t, mockResponse.WebhookURL, "https://")
	assert.Contains(t, mockResponse.WebhookURL, "trigger-123")
}

func stringPtr(s string) *string {
	return &s
}
