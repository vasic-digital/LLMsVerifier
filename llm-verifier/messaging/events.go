// Package messaging provides message broker integration for LLMsVerifier.
package messaging

import (
	"fmt"
	"time"
)

// EventType defines the type of verification event.
type EventType string

// Event types for LLMsVerifier
const (
	// Verification lifecycle events
	EventVerificationStarted   EventType = "verification.started"
	EventVerificationCompleted EventType = "verification.completed"
	EventVerificationFailed    EventType = "verification.failed"

	// Provider events
	EventProviderDiscovered  EventType = "provider.discovered"
	EventProviderScored      EventType = "provider.scored"
	EventProviderHealthCheck EventType = "provider.health_check"
	EventProviderFailed      EventType = "provider.failed"

	// Model events
	EventModelVerified EventType = "model.verified"
	EventModelRanked   EventType = "model.ranked"
	EventModelFailed   EventType = "model.failed"

	// Team selection events
	EventTeamSelected     EventType = "team.selected"
	EventTeamMemberFailed EventType = "team.member_failed"

	// Configuration events
	EventConfigExported  EventType = "config.exported"
	EventConfigGenerated EventType = "config.generated"

	// System events
	EventSystemHealthChanged EventType = "system.health_changed"
	EventSecurityAlert       EventType = "system.security_alert"
)

// Severity represents the severity level of an event.
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// VerificationEvent represents a verification event to be published.
type VerificationEvent struct {
	// ID is the unique identifier for this event.
	ID string `json:"id"`

	// Type is the event type.
	Type EventType `json:"type"`

	// Source identifies the component that generated the event.
	Source string `json:"source"`

	// Subject is the entity this event is about (e.g., provider ID, model ID).
	Subject string `json:"subject,omitempty"`

	// Severity is the event severity level.
	Severity Severity `json:"severity"`

	// Title is a short title for the event.
	Title string `json:"title"`

	// Message is a detailed description of the event.
	Message string `json:"message"`

	// ProviderID is the provider this event is related to.
	ProviderID string `json:"provider_id,omitempty"`

	// ModelID is the model this event is related to.
	ModelID string `json:"model_id,omitempty"`

	// Score is the verification score (if applicable).
	Score float64 `json:"score,omitempty"`

	// Details contains additional event-specific information.
	Details map[string]interface{} `json:"details,omitempty"`

	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp"`

	// TraceID for distributed tracing.
	TraceID string `json:"trace_id,omitempty"`
}

// NewVerificationEvent creates a new verification event.
func NewVerificationEvent(eventType EventType, severity Severity, title, message string) *VerificationEvent {
	return &VerificationEvent{
		ID:        generateEventID(),
		Type:      eventType,
		Source:    "llmsverifier",
		Severity:  severity,
		Title:     title,
		Message:   message,
		Timestamp: time.Now().UTC(),
		Details:   make(map[string]interface{}),
	}
}

// WithProvider sets the provider ID for the event.
func (e *VerificationEvent) WithProvider(providerID string) *VerificationEvent {
	e.ProviderID = providerID
	e.Subject = providerID
	return e
}

// WithModel sets the model ID for the event.
func (e *VerificationEvent) WithModel(modelID string) *VerificationEvent {
	e.ModelID = modelID
	if e.Subject == "" {
		e.Subject = modelID
	}
	return e
}

// WithScore sets the verification score for the event.
func (e *VerificationEvent) WithScore(score float64) *VerificationEvent {
	e.Score = score
	return e
}

// WithTraceID sets the trace ID for distributed tracing.
func (e *VerificationEvent) WithTraceID(traceID string) *VerificationEvent {
	e.TraceID = traceID
	return e
}

// WithDetails sets additional details for the event.
func (e *VerificationEvent) WithDetails(details map[string]interface{}) *VerificationEvent {
	e.Details = details
	return e
}

// AddDetail adds a single detail to the event.
func (e *VerificationEvent) AddDetail(key string, value interface{}) *VerificationEvent {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// ProviderHealthEvent represents a provider health check result.
type ProviderHealthEvent struct {
	ProviderID   string    `json:"provider_id"`
	ProviderName string    `json:"provider_name"`
	Status       string    `json:"status"` // "healthy", "degraded", "unhealthy"
	Latency      int64     `json:"latency_ms"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// ProviderScoredEvent represents a provider scoring result.
type ProviderScoredEvent struct {
	ProviderID       string    `json:"provider_id"`
	ProviderName     string    `json:"provider_name"`
	OverallScore     float64   `json:"overall_score"`
	ResponseSpeed    float64   `json:"response_speed_score"`
	ModelEfficiency  float64   `json:"model_efficiency_score"`
	CostEffectiveness float64  `json:"cost_effectiveness_score"`
	Capability       float64   `json:"capability_score"`
	Recency          float64   `json:"recency_score"`
	Timestamp        time.Time `json:"timestamp"`
}

// ModelRankedEvent represents a model ranking update.
type ModelRankedEvent struct {
	ProviderID string    `json:"provider_id"`
	ModelID    string    `json:"model_id"`
	ModelName  string    `json:"model_name"`
	Rank       int       `json:"rank"`
	Score      float64   `json:"score"`
	PrevRank   int       `json:"prev_rank,omitempty"`
	PrevScore  float64   `json:"prev_score,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// TeamSelectedEvent represents an AI debate team selection.
type TeamSelectedEvent struct {
	TeamID       string            `json:"team_id"`
	TeamSize     int               `json:"team_size"`
	PrimaryLLMs  []TeamMember      `json:"primary_llms"`
	FallbackLLMs []TeamMember      `json:"fallback_llms"`
	SelectionCriteria string       `json:"selection_criteria"`
	Timestamp    time.Time         `json:"timestamp"`
}

// TeamMember represents a member of the AI debate team.
type TeamMember struct {
	ProviderID string  `json:"provider_id"`
	ModelID    string  `json:"model_id"`
	Position   string  `json:"position"`
	Score      float64 `json:"score"`
	IsPrimary  bool    `json:"is_primary"`
}

// generateEventID generates a unique event ID.
func generateEventID() string {
	return fmt.Sprintf("evt-%d-%d", time.Now().UnixNano()/1e6, time.Now().Nanosecond()%1000)
}
