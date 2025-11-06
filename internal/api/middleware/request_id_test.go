package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestID_WhenClientProvidesRequestID_ThenUsesProvidedID(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)

	expectedRequestID := "client-provided-request-id"
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		// Assert
		actualRequestID, exists := c.Get(RequestIDKey)
		if !exists {
			t.Fatal("expected request ID to exist in context")
		}
		if actualRequestID != expectedRequestID {
			t.Errorf("expected request ID '%s', got '%s'", expectedRequestID, actualRequestID)
		}
		c.Status(http.StatusOK)
	})

	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set(RequestIDHeader, expectedRequestID)

	// Act
	router.ServeHTTP(w, c.Request)

	// Assert
	responseRequestID := w.Header().Get(RequestIDHeader)
	if responseRequestID != expectedRequestID {
		t.Errorf("expected response header to contain request ID '%s', got '%s'", expectedRequestID, responseRequestID)
	}
}

func TestRequestID_WhenClientDoesNotProvideRequestID_ThenGeneratesNewID(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)

	var generatedRequestID string
	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		// Assert
		actualRequestID, exists := c.Get(RequestIDKey)
		if !exists {
			t.Fatal("expected request ID to exist in context")
		}
		generatedRequestID = actualRequestID.(string)
		if generatedRequestID == "" {
			t.Error("expected generated request ID to be non-empty")
		}
		c.Status(http.StatusOK)
	})

	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	// Act
	router.ServeHTTP(w, c.Request)

	// Assert
	responseRequestID := w.Header().Get(RequestIDHeader)
	if responseRequestID != generatedRequestID {
		t.Errorf("expected response header to contain generated request ID '%s', got '%s'", generatedRequestID, responseRequestID)
	}
	if responseRequestID == "" {
		t.Error("expected response header to contain non-empty request ID")
	}
}

func TestRequestID_WhenMultipleRequests_ThenEachGetsDifferentID(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RequestID())

	requestIDs := make([]string, 0, 3)
	router.GET("/test", func(c *gin.Context) {
		requestID, _ := c.Get(RequestIDKey)
		requestIDs = append(requestIDs, requestID.(string))
		c.Status(http.StatusOK)
	})

	// Act
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		router.ServeHTTP(w, req)
	}

	// Assert
	if len(requestIDs) != 3 {
		t.Fatalf("expected 3 request IDs, got %d", len(requestIDs))
	}

	// Verify all IDs are different
	for i := 0; i < len(requestIDs); i++ {
		for j := i + 1; j < len(requestIDs); j++ {
			if requestIDs[i] == requestIDs[j] {
				t.Errorf("expected request IDs to be unique, but found duplicate: %s", requestIDs[i])
			}
		}
	}
}

func TestRequestID_WhenEmptyStringProvided_ThenGeneratesNewID(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)

	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		requestID, exists := c.Get(RequestIDKey)
		if !exists {
			t.Fatal("expected request ID to exist in context")
		}
		if requestID.(string) == "" {
			t.Error("expected request ID to be non-empty")
		}
		c.Status(http.StatusOK)
	})

	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set(RequestIDHeader, "") // Empty header

	// Act
	router.ServeHTTP(w, c.Request)

	// Assert
	responseRequestID := w.Header().Get(RequestIDHeader)
	if responseRequestID == "" {
		t.Error("expected non-empty request ID in response header")
	}
}

func TestRequestID_WhenCalled_ThenSetsResponseHeader(t *testing.T) {
	// Arrange
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, router := gin.CreateTestContext(w)

	router.Use(RequestID())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	// Act
	router.ServeHTTP(w, c.Request)

	// Assert
	responseRequestID := w.Header().Get(RequestIDHeader)
	if responseRequestID == "" {
		t.Error("expected X-Request-ID header in response")
	}
}
