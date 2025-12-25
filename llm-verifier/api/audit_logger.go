package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// AuditLogger provides comprehensive audit logging for security events
type AuditLogger struct {
	logFile string
	logger  *log.Logger
	events  chan *AuditEvent
	stopCh  chan struct{}
}

// AuditEvent represents an auditable security event
type AuditEvent struct {
	ID           string                 `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	EventType    string                 `json:"event_type"`
	UserID       *int64                 `json:"user_id,omitempty"`
	ClientIP     string                 `json:"client_ip"`
	UserAgent    string                 `json:"user_agent"`
	Method       string                 `json:"method"`
	Path         string                 `json:"path"`
	StatusCode   int                    `json:"status_code"`
	RequestSize  int64                  `json:"request_size"`
	ResponseSize int64                  `json:"response_size"`
	Duration     time.Duration          `json:"duration"`
	Severity     string                 `json:"severity"`
	Details      map[string]interface{} `json:"details,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logFile string) *AuditLogger {
	al := &AuditLogger{
		logFile: logFile,
		events:  make(chan *AuditEvent, 1000), // Buffer up to 1000 events
		stopCh:  make(chan struct{}),
	}

	// Start the logging goroutine
	go al.processEvents()

	return al
}

// LogEvent logs an audit event
func (al *AuditLogger) LogEvent(event *AuditEvent) {
	select {
	case al.events <- event:
		// Event queued successfully
	default:
		// Channel is full, log immediately to avoid blocking
		log.Printf("[AUDIT] Queue full, logging directly: %+v", event)
	}
}

// LogHTTPRequest logs an HTTP request
func (al *AuditLogger) LogHTTPRequest(r *http.Request, statusCode int, duration time.Duration, responseSize int64) {
	event := &AuditEvent{
		ID:           generateAuditID(),
		Timestamp:    time.Now(),
		EventType:    "http_request",
		ClientIP:     getClientIP(r),
		UserAgent:    r.Header.Get("User-Agent"),
		Method:       r.Method,
		Path:         r.URL.Path,
		StatusCode:   statusCode,
		RequestSize:  r.ContentLength,
		ResponseSize: responseSize,
		Duration:     duration,
		Severity:     al.getSeverityForStatus(statusCode),
	}

	// Add additional details for security-relevant requests
	if al.isSecurityRelevant(r) {
		event.Details = map[string]interface{}{
			"query":   r.URL.RawQuery,
			"referer": r.Header.Get("Referer"),
			"accept":  r.Header.Get("Accept"),
			"host":    r.Host,
		}
	}

	al.LogEvent(event)
}

// LogSecurityEvent logs a security-related event
func (al *AuditLogger) LogSecurityEvent(eventType, severity, description string, details map[string]interface{}) {
	event := &AuditEvent{
		ID:        generateAuditID(),
		Timestamp: time.Now(),
		EventType: eventType,
		Severity:  severity,
		Details:   details,
	}

	if desc, ok := details["description"].(string); ok {
		// Use description from details if available
		_ = desc // Could store in event if needed
	}

	al.LogEvent(event)
}

// LogAuthentication logs authentication events
func (al *AuditLogger) LogAuthentication(clientIP, userAgent, username string, success bool, details map[string]interface{}) {
	eventType := "authentication_success"
	severity := "info"

	if !success {
		eventType = "authentication_failure"
		severity = "warning"
	}

	event := &AuditEvent{
		ID:        generateAuditID(),
		Timestamp: time.Now(),
		EventType: eventType,
		ClientIP:  clientIP,
		UserAgent: userAgent,
		Severity:  severity,
		Details:   details,
	}

	if event.Details == nil {
		event.Details = make(map[string]interface{})
	}
	event.Details["username"] = username

	al.LogEvent(event)
}

// LogAuthorization logs authorization events
func (al *AuditLogger) LogAuthorization(clientIP, userAgent, resource, action string, allowed bool, details map[string]interface{}) {
	eventType := "authorization_granted"
	severity := "info"

	if !allowed {
		eventType = "authorization_denied"
		severity = "warning"
	}

	event := &AuditEvent{
		ID:        generateAuditID(),
		Timestamp: time.Now(),
		EventType: eventType,
		ClientIP:  clientIP,
		UserAgent: userAgent,
		Severity:  severity,
		Details:   details,
	}

	if event.Details == nil {
		event.Details = make(map[string]interface{})
	}
	event.Details["resource"] = resource
	event.Details["action"] = action

	al.LogEvent(event)
}

// LogDataAccess logs data access events
func (al *AuditLogger) LogDataAccess(clientIP, userAgent, operation, resource string, recordCount int, details map[string]interface{}) {
	event := &AuditEvent{
		ID:        generateAuditID(),
		Timestamp: time.Now(),
		EventType: "data_access",
		ClientIP:  clientIP,
		UserAgent: userAgent,
		Severity:  "info",
		Details:   details,
	}

	if event.Details == nil {
		event.Details = make(map[string]interface{})
	}
	event.Details["operation"] = operation
	event.Details["resource"] = resource
	event.Details["record_count"] = recordCount

	al.LogEvent(event)
}

// LogError logs error events
func (al *AuditLogger) LogError(clientIP, userAgent, operation string, err error, details map[string]interface{}) {
	event := &AuditEvent{
		ID:        generateAuditID(),
		Timestamp: time.Now(),
		EventType: "error",
		ClientIP:  clientIP,
		UserAgent: userAgent,
		Severity:  "error",
		Error:     err.Error(),
		Details:   details,
	}

	if event.Details == nil {
		event.Details = make(map[string]interface{})
	}
	event.Details["operation"] = operation

	al.LogEvent(event)
}

// processEvents processes audit events from the queue
func (al *AuditLogger) processEvents() {
	for {
		select {
		case event := <-al.events:
			al.writeEvent(event)
		case <-al.stopCh:
			return
		}
	}
}

// writeEvent writes an audit event to the log
func (al *AuditLogger) writeEvent(event *AuditEvent) {
	// Convert to JSON for structured logging
	jsonData, err := json.Marshal(event)
	if err != nil {
		log.Printf("[AUDIT ERROR] Failed to marshal event: %v", err)
		return
	}

	// Log to stdout (in production, would write to file/database)
	log.Printf("[AUDIT] %s", string(jsonData))
}

// getSeverityForStatus returns severity level based on HTTP status code
func (al *AuditLogger) getSeverityForStatus(statusCode int) string {
	switch {
	case statusCode >= 500:
		return "error"
	case statusCode >= 400:
		return "warning"
	case statusCode >= 300:
		return "info"
	default:
		return "info"
	}
}

// isSecurityRelevant checks if an HTTP request is security-relevant
func (al *AuditLogger) isSecurityRelevant(r *http.Request) bool {
	securityPaths := []string{
		"/api/auth", "/api/login", "/api/logout",
		"/api/admin", "/api/users", "/api/config",
		"/api/export", "/api/import",
	}

	path := r.URL.Path
	for _, securityPath := range securityPaths {
		if len(path) >= len(securityPath) && path[:len(securityPath)] == securityPath {
			return true
		}
	}

	// Also consider non-GET methods as security-relevant
	if r.Method != "GET" && r.Method != "HEAD" {
		return true
	}

	return false
}

// Stop stops the audit logger
func (al *AuditLogger) Stop() {
	close(al.stopCh)
}

// generateAuditID generates a unique audit event ID
func generateAuditID() string {
	return fmt.Sprintf("audit_%d", time.Now().UnixNano())
}

// AuditMiddleware creates middleware for audit logging
func AuditMiddleware(auditLogger *AuditLogger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			startTime := time.Now()

			// Create a response writer wrapper to capture status and size
			wrapper := &auditResponseWriter{
				ResponseWriter: w,
				statusCode:     200, // Default status
			}

			// Call the next handler
			next.ServeHTTP(wrapper, r)

			// Log the request
			duration := time.Since(startTime)
			auditLogger.LogHTTPRequest(r, wrapper.statusCode, duration, int64(wrapper.size))
		})
	}
}

// auditResponseWriter wraps http.ResponseWriter to capture status code and response size
type auditResponseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (arw *auditResponseWriter) WriteHeader(code int) {
	arw.statusCode = code
	arw.ResponseWriter.WriteHeader(code)
}

func (arw *auditResponseWriter) Write(data []byte) (int, error) {
	size, err := arw.ResponseWriter.Write(data)
	arw.size += size
	return size, err
}
