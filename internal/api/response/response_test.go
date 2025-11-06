package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSuccess_WhenCalled_ThenReturnsSuccessResponse(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	testData := map[string]string{"key": "value"}

	// Act
	Success(c, http.StatusOK, testData, "success message")

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Message != "success message" {
		t.Errorf("expected message 'success message', got '%s'", response.Message)
	}
}

func TestError_WhenCalledWithRequestID_ThenIncludesTraceID(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("request_id", "test-trace-id")

	// Act
	Error(c, http.StatusBadRequest, "test error", nil)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Error != "test error" {
		t.Errorf("expected error 'test error', got '%s'", response.Error)
	}
	if response.TraceID != "test-trace-id" {
		t.Errorf("expected trace ID 'test-trace-id', got '%s'", response.TraceID)
	}
}

func TestError_WhenCalledWithoutRequestID_ThenGeneratesTraceID(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Act
	Error(c, http.StatusInternalServerError, "test error", "details")

	// Assert
	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.TraceID == "" {
		t.Error("expected trace ID to be generated")
	}
}

func TestBadRequest_WhenCalled_ThenReturns400(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Act
	BadRequest(c, "bad request", "details")

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestNotFound_WhenCalled_ThenReturns404(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Act
	NotFound(c, "not found")

	// Assert
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestInternalServerError_WhenCalled_ThenReturns500(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Act
	InternalServerError(c, "internal error")

	// Assert
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestUnauthorized_WhenCalled_ThenReturns401(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Act
	Unauthorized(c, "unauthorized")

	// Assert
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestForbidden_WhenCalled_ThenReturns403(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Act
	Forbidden(c, "forbidden")

	// Assert
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestConflict_WhenCalled_ThenReturns409(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Act
	Conflict(c, "conflict", "details")

	// Assert
	if w.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", w.Code)
	}
}

func TestCreated_WhenCalled_ThenReturns201(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Act
	Created(c, map[string]string{"id": "123"}, "created")

	// Assert
	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}
}

func TestOK_WhenCalled_ThenReturns200(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Act
	OK(c, map[string]string{"result": "ok"})

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestAccepted_WhenCalled_ThenReturns202(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Act
	Accepted(c, "accepted")

	// Assert
	if w.Code != http.StatusAccepted {
		t.Errorf("expected status 202, got %d", w.Code)
	}
}

func TestNoContent_WhenCalled_ThenReturns204(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)

	router.GET("/test", func(c *gin.Context) {
		NoContent(c)
	})

	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	// Act
	router.ServeHTTP(w, c.Request)

	// Assert
	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}
}

func TestPaginated_WhenCalled_ThenReturnsPaginatedResponse(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	data := []string{"item1", "item2"}
	pagination := Pagination{
		CurrentPage:  1,
		PageSize:     10,
		TotalPages:   5,
		TotalRecords: 50,
	}

	// Act
	Paginated(c, data, pagination)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response PaginatedResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Pagination.CurrentPage != 1 {
		t.Errorf("expected current page 1, got %d", response.Pagination.CurrentPage)
	}
	if response.Pagination.TotalRecords != 50 {
		t.Errorf("expected total records 50, got %d", response.Pagination.TotalRecords)
	}
}

func TestGetRequestID_WhenRequestIDExists_ThenReturnsIt(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("request_id", "existing-request-id")

	// Act
	requestID := GetRequestID(c)

	// Assert
	if requestID != "existing-request-id" {
		t.Errorf("expected 'existing-request-id', got '%s'", requestID)
	}
}

func TestGetRequestID_WhenRequestIDDoesNotExist_ThenGeneratesNew(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	// Act
	requestID := GetRequestID(c)

	// Assert
	if requestID == "" {
		t.Error("expected request ID to be generated")
	}
}

func TestGetRequestID_WhenRequestIDIsNotString_ThenGeneratesNew(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Set("request_id", 12345) // Invalid type

	// Act
	requestID := GetRequestID(c)

	// Assert
	if requestID == "" {
		t.Error("expected request ID to be generated")
	}
}

func TestValidationErrors_WhenCalled_ThenReturnsBadRequestWithErrors(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	errors := []ValidationError{
		{Field: "email", Message: "invalid email format"},
		{Field: "name", Message: "name is required"},
	}

	// Act
	ValidationErrors(c, errors)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if response.Error != "validation failed" {
		t.Errorf("expected error 'validation failed', got '%s'", response.Error)
	}
}
