package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// SuccessResponse represents a successful API response.
type SuccessResponse struct {
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
}

// ErrorResponse represents an error API response.
type ErrorResponse struct {
	Error   string      `json:"error"`
	Details interface{} `json:"details,omitempty"`
	TraceID string      `json:"trace_id,omitempty"`
}

// PaginatedResponse represents a paginated API response.
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// Pagination contains pagination metadata.
type Pagination struct {
	CurrentPage  int   `json:"current_page"`
	PageSize     int   `json:"page_size"`
	TotalPages   int   `json:"total_pages"`
	TotalRecords int64 `json:"total_records"`
}

// Success sends a successful response with data.
func Success(c *gin.Context, statusCode int, data interface{}, message string) {
	c.JSON(statusCode, SuccessResponse{
		Data:    data,
		Message: message,
	})
}

// Error sends an error response with details.
func Error(c *gin.Context, statusCode int, err string, details interface{}) {
	traceID := GetRequestID(c)
	c.JSON(statusCode, ErrorResponse{
		Error:   err,
		Details: details,
		TraceID: traceID,
	})
}

// BadRequest sends a 400 Bad Request response.
func BadRequest(c *gin.Context, err string, details interface{}) {
	Error(c, http.StatusBadRequest, err, details)
}

// NotFound sends a 404 Not Found response.
func NotFound(c *gin.Context, err string) {
	Error(c, http.StatusNotFound, err, nil)
}

// InternalServerError sends a 500 Internal Server Error response.
func InternalServerError(c *gin.Context, err string) {
	Error(c, http.StatusInternalServerError, err, nil)
}

// Unauthorized sends a 401 Unauthorized response.
func Unauthorized(c *gin.Context, err string) {
	Error(c, http.StatusUnauthorized, err, nil)
}

// Forbidden sends a 403 Forbidden response.
func Forbidden(c *gin.Context, err string) {
	Error(c, http.StatusForbidden, err, nil)
}

// Conflict sends a 409 Conflict response.
func Conflict(c *gin.Context, err string, details interface{}) {
	Error(c, http.StatusConflict, err, details)
}

// Created sends a 201 Created response.
func Created(c *gin.Context, data interface{}, message string) {
	Success(c, http.StatusCreated, data, message)
}

// OK sends a 200 OK response.
func OK(c *gin.Context, data interface{}) {
	Success(c, http.StatusOK, data, "")
}

// Accepted sends a 202 Accepted response.
func Accepted(c *gin.Context, message string) {
	Success(c, http.StatusAccepted, nil, message)
}

// NoContent sends a 204 No Content response.
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Paginated sends a paginated response.
func Paginated(c *gin.Context, data interface{}, pagination Pagination) {
	c.JSON(http.StatusOK, PaginatedResponse{
		Data:       data,
		Pagination: pagination,
	})
}

// GetRequestID retrieves the request ID from context.
func GetRequestID(c *gin.Context) string {
	if requestID, exists := c.Get("request_id"); exists {
		if id, ok := requestID.(string); ok {
			return id
		}
	}
	return uuid.New().String()
}

// ValidationError represents a field validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrors sends a 400 Bad Request with field validation errors.
func ValidationErrors(c *gin.Context, errors []ValidationError) {
	BadRequest(c, "validation failed", errors)
}
