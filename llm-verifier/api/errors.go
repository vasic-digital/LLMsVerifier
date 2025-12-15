package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// Error codes
const (
	ErrCodeValidation      = "VALIDATION_ERROR"
	ErrCodeNotFound        = "NOT_FOUND"
	ErrCodeUnauthorized    = "UNAUTHORIZED"
	ErrCodeForbidden       = "FORBIDDEN"
	ErrCodeInternal        = "INTERNAL_ERROR"
	ErrCodeBadRequest      = "BAD_REQUEST"
	ErrCodeConflict        = "CONFLICT"
	ErrCodeRateLimit       = "RATE_LIMIT_EXCEEDED"
	ErrCodeDatabase        = "DATABASE_ERROR"
	ErrCodeExternalService = "EXTERNAL_SERVICE_ERROR"
)

// ErrorResponse represents a standardized error response
type ErrorResponse struct {
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Details   map[string]string `json:"details,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
	Timestamp int64             `json:"timestamp"`
}

// SuccessResponse represents a standardized success response
type SuccessResponse struct {
	Data    interface{} `json:"data"`
	Message string      `json:"message,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// PaginatedResponse represents a paginated response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Pagination struct {
		Page       int  `json:"page"`
		PageSize   int  `json:"page_size"`
		TotalItems int  `json:"total_items"`
		TotalPages int  `json:"total_pages"`
		HasNext    bool `json:"has_next"`
		HasPrev    bool `json:"has_prev"`
	} `json:"pagination"`
}

// Response helpers

// SendError sends a standardized error response
func SendError(c *gin.Context, status int, code, message string, details map[string]string) {
	response := ErrorResponse{
		Code:      code,
		Message:   message,
		Details:   details,
		Timestamp: time.Now().Unix(),
	}

	// Add request ID if available
	if requestID := c.GetString("request_id"); requestID != "" {
		response.RequestID = requestID
	}

	c.JSON(status, response)
}

// SendSuccess sends a standardized success response
func SendSuccess(c *gin.Context, status int, data interface{}, message string) {
	response := SuccessResponse{
		Data:    data,
		Message: message,
	}
	c.JSON(status, response)
}

// SendPaginated sends a paginated response
func SendPaginated(c *gin.Context, data interface{}, page, pageSize, totalItems int) {
	totalPages := (totalItems + pageSize - 1) / pageSize

	response := PaginatedResponse{
		Data: data,
	}
	response.Pagination.Page = page
	response.Pagination.PageSize = pageSize
	response.Pagination.TotalItems = totalItems
	response.Pagination.TotalPages = totalPages
	response.Pagination.HasNext = page < totalPages
	response.Pagination.HasPrev = page > 1

	c.JSON(http.StatusOK, response)
}

// Error handlers for common scenarios

// HandleValidationError handles validation errors
func HandleValidationError(c *gin.Context, err error) {
	if _, ok := err.(validator.ValidationErrors); ok {
		details := GetValidationErrors(err)
		SendError(c, http.StatusBadRequest, ErrCodeValidation, "Validation failed", details)
		return
	}

	// Generic binding error
	SendError(c, http.StatusBadRequest, ErrCodeValidation, "Invalid request format", map[string]string{
		"error": err.Error(),
	})
}

// HandleNotFoundError handles not found errors
func HandleNotFoundError(c *gin.Context, resource string, id interface{}) {
	SendError(c, http.StatusNotFound, ErrCodeNotFound,
		resource+" not found",
		map[string]string{
			"resource": resource,
			"id":       fmt.Sprintf("%v", id),
		})
}

// HandleUnauthorizedError handles unauthorized access
func HandleUnauthorizedError(c *gin.Context, message string) {
	if message == "" {
		message = "Authentication required"
	}
	SendError(c, http.StatusUnauthorized, ErrCodeUnauthorized, message, nil)
}

// HandleForbiddenError handles forbidden access
func HandleForbiddenError(c *gin.Context, message string) {
	if message == "" {
		message = "Insufficient permissions"
	}
	SendError(c, http.StatusForbidden, ErrCodeForbidden, message, nil)
}

// HandleInternalError handles internal server errors
func HandleInternalError(c *gin.Context, err error) {
	// Log the error internally
	log.Printf("Internal server error: %v", err)

	// Send generic error to client
	SendError(c, http.StatusInternalServerError, ErrCodeInternal,
		"Internal server error", nil)
}

// HandleDatabaseError handles database errors
func HandleDatabaseError(c *gin.Context, err error) {
	// Log the error internally
	log.Printf("Database error: %v", err)

	SendError(c, http.StatusInternalServerError, ErrCodeDatabase,
		"Database operation failed", nil)
}

// HandleConflictError handles conflict errors (e.g., duplicate entries)
func HandleConflictError(c *gin.Context, resource string, field string, value string) {
	SendError(c, http.StatusConflict, ErrCodeConflict,
		resource+" already exists",
		map[string]string{
			"resource": resource,
			"field":    field,
			"value":    value,
		})
}

// HandleRateLimitError handles rate limiting errors
func HandleRateLimitError(c *gin.Context, retryAfter int) {
	headers := map[string]string{
		"Retry-After": fmt.Sprintf("%d", retryAfter),
	}

	for k, v := range headers {
		c.Header(k, v)
	}

	SendError(c, http.StatusTooManyRequests, ErrCodeRateLimit,
		"Rate limit exceeded",
		map[string]string{
			"retry_after_seconds": fmt.Sprintf("%d", retryAfter),
		})
}

// Middleware to add request ID
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateUUID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// Helper function to generate UUID (simplified version)
func generateUUID() string {
	return fmt.Sprintf("%x", time.Now().UnixNano())
}
