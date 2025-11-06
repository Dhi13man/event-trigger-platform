package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// RequestIDHeader is the header name for request ID.
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey is the context key for request ID.
	RequestIDKey = "request_id"
)

// RequestID is a middleware that injects a unique request ID into each request.
// If the client provides an X-Request-ID header, it will be used; otherwise, a new UUID is generated.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID is provided by client
		requestID := c.GetHeader(RequestIDHeader)

		// Generate new UUID if not provided
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Store in context for handlers and other middleware
		c.Set(RequestIDKey, requestID)

		// Add to response headers for tracing
		c.Writer.Header().Set(RequestIDHeader, requestID)

		c.Next()
	}
}
