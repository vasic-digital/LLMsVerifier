package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"llm-verifier/database"
)

// EventType represents the type of system event
type EventType string

const (
	EventTypeModelVerified           EventType = "model.verified"
	EventTypeModelVerificationFailed EventType = "model.verification.failed"
	EventTypeScoreChanged            EventType = "model.score.changed"
	EventTypeProviderAdded           EventType = "provider.added"
	EventTypeProviderUpdated         EventType = "provider.updated"
	EventTypeSystemHealthChanged     EventType = "system.health.changed"
	EventTypeVerificationStarted     EventType = "verification.started"
	EventTypeVerificationCompleted   EventType = "verification.completed"
	EventTypeScheduleExecuted        EventType = "schedule.executed"
	EventTypeErrorOccurred           EventType = "error.occurred"
)

// EventSeverity represents the severity level of an event
type EventSeverity string

const (
	EventSeverityInfo     EventSeverity = "info"
	EventSeverityWarning  EventSeverity = "warning"
	EventSeverityError    EventSeverity = "error"
	EventSeverityCritical EventSeverity = "critical"
)

// Event represents a system event
type Event struct {
	ID        string                 `json:"id" db:"id"`
	Type      EventType              `json:"type" db:"type"`
	Severity  EventSeverity          `json:"severity" db:"severity"`
	Message   string                 `json:"message" db:"message"`
	Data      map[string]interface{} `json:"data,omitempty" db:"data"`
	Timestamp time.Time              `json:"timestamp" db:"timestamp"`
	Source    string                 `json:"source" db:"source"`
	UserID    *int64                 `json:"user_id,omitempty" db:"user_id"`
	SessionID string                 `json:"session_id,omitempty" db:"session_id"`
}

// Subscriber represents an event subscriber
type Subscriber interface {
	HandleEvent(event Event) error
	GetID() string
	GetTypes() []EventType
	IsActive() bool
}

// WebSocketSubscriber handles WebSocket event delivery
type WebSocketSubscriber struct {
	ID     string
	Conn   interface{} // WebSocket connection
	Types  []EventType
	Active bool
	mu     sync.RWMutex
}

// GRPCSubscriber handles gRPC streaming event delivery
type GRPCSubscriber struct {
	ID     string
	Stream interface{} // gRPC stream
	Types  []EventType
	Active bool
	mu     sync.RWMutex
}

// NotificationSubscriber handles notification-based event delivery
type NotificationSubscriber struct {
	ID       string
	Channels []string // ["slack", "email", "telegram"]
	Types    []EventType
	Active   bool
	mu       sync.RWMutex
}

// EventBus manages event publishing and subscription
type EventBus struct {
	subscribers map[string]Subscriber
	db          *database.Database
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
}

// NewEventBus creates a new event bus
func NewEventBus(db *database.Database) *EventBus {
	ctx, cancel := context.WithCancel(context.Background())

	return &EventBus{
		subscribers: make(map[string]Subscriber),
		db:          db,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Publish publishes an event to all interested subscribers
func (eb *EventBus) Publish(event Event) error {
	// Store event in database
	if err := eb.storeEvent(event); err != nil {
		log.Printf("Failed to store event %s: %v", event.ID, err)
	}

	// Publish to subscribers
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	published := 0
	for _, subscriber := range eb.subscribers {
		if subscriber.IsActive() && eb.shouldReceiveEvent(subscriber, event) {
			go func(sub Subscriber, evt Event) {
				if err := sub.HandleEvent(evt); err != nil {
					log.Printf("Failed to deliver event %s to subscriber %s: %v",
						evt.ID, sub.GetID(), err)
				}
			}(subscriber, event)
			published++
		}
	}

	log.Printf("Event %s published to %d subscribers", event.ID, published)
	return nil
}

// Subscribe adds a new subscriber
func (eb *EventBus) Subscribe(subscriber Subscriber) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.subscribers[subscriber.GetID()] = subscriber
	log.Printf("Subscriber %s registered for event types: %v",
		subscriber.GetID(), subscriber.GetTypes())
}

// Unsubscribe removes a subscriber
func (eb *EventBus) Unsubscribe(subscriberID string) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if _, exists := eb.subscribers[subscriberID]; exists {
		delete(eb.subscribers, subscriberID)
		log.Printf("Subscriber %s unsubscribed", subscriberID)
	}
}

// GetSubscribers returns all active subscribers
func (eb *EventBus) GetSubscribers() []Subscriber {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	var active []Subscriber
	for _, sub := range eb.subscribers {
		if sub.IsActive() {
			active = append(active, sub)
		}
	}
	return active
}

// Shutdown gracefully shuts down the event bus
func (eb *EventBus) Shutdown() {
	eb.cancel()

	eb.mu.Lock()
	defer eb.mu.Unlock()

	// Clean up subscribers
	for id := range eb.subscribers {
		delete(eb.subscribers, id)
	}

	log.Println("Event bus shutdown complete")
}

// Helper methods

func (eb *EventBus) shouldReceiveEvent(subscriber Subscriber, event Event) bool {
	for _, eventType := range subscriber.GetTypes() {
		if eventType == event.Type {
			return true
		}
	}
	return false
}

func (eb *EventBus) storeEvent(event Event) error {
	// Convert event data to JSON for storage
	dataJSON, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	// Create database event struct
	dbEvent := &database.Event{
		EventType: string(event.Type),
		Severity:  string(event.Severity),
		Title:     string(event.Type), // Use event type as title for now
		Message:   event.Message,
		Details:   stringPtr(string(dataJSON)),
		// ModelID, ProviderID, etc. can be set based on event data if needed
	}

	// Store in database
	err = eb.db.CreateEvent(dbEvent)
	if err != nil {
		return fmt.Errorf("failed to store event in database: %w", err)
	}

	log.Printf("Stored event: %s (%s) - %s", event.ID, event.Type, event.Message)
	return nil
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}

// WebSocketSubscriber implementation
func (ws *WebSocketSubscriber) HandleEvent(event Event) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if !ws.Active {
		return fmt.Errorf("subscriber not active")
	}

	// Send event via WebSocket
	// This is a placeholder - actual WebSocket sending would go here
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return err
	}

	log.Printf("WebSocket subscriber %s received event: %s", ws.ID, string(eventJSON))
	return nil
}

func (ws *WebSocketSubscriber) GetID() string {
	return ws.ID
}

func (ws *WebSocketSubscriber) GetTypes() []EventType {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.Types
}

func (ws *WebSocketSubscriber) IsActive() bool {
	ws.mu.RLock()
	defer ws.mu.RUnlock()
	return ws.Active
}

// GRPCSubscriber implementation
func (gs *GRPCSubscriber) HandleEvent(event Event) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if !gs.Active {
		return fmt.Errorf("subscriber not active")
	}

	// Send event via gRPC stream
	// This is a placeholder - actual gRPC streaming would go here
	log.Printf("gRPC subscriber %s received event: %s", gs.ID, event.ID)
	return nil
}

func (gs *GRPCSubscriber) GetID() string {
	return gs.ID
}

func (gs *GRPCSubscriber) GetTypes() []EventType {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.Types
}

func (gs *GRPCSubscriber) IsActive() bool {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.Active
}

// NotificationSubscriber implementation
func (ns *NotificationSubscriber) HandleEvent(event Event) error {
	ns.mu.Lock()
	defer ns.mu.Unlock()

	if !ns.Active {
		return fmt.Errorf("subscriber not active")
	}

	// Send notifications via configured channels
	// This is a placeholder - actual notification sending would go here
	log.Printf("Notification subscriber %s sending event %s via channels: %v",
		ns.ID, event.ID, ns.Channels)
	return nil
}

func (ns *NotificationSubscriber) GetID() string {
	return ns.ID
}

func (ns *NotificationSubscriber) GetTypes() []EventType {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	return ns.Types
}

func (ns *NotificationSubscriber) IsActive() bool {
	ns.mu.RLock()
	defer ns.mu.RUnlock()
	return ns.Active
}

// Event creation helpers

// NewEvent creates a new event with generated ID
func NewEvent(eventType EventType, severity EventSeverity, message string, source string) Event {
	return Event{
		ID:        generateEventID(),
		Type:      eventType,
		Severity:  severity,
		Message:   message,
		Data:      make(map[string]interface{}),
		Timestamp: time.Now(),
		Source:    source,
	}
}

// WithData adds data to an event
func (e Event) WithData(key string, value interface{}) Event {
	if e.Data == nil {
		e.Data = make(map[string]interface{})
	}
	e.Data[key] = value
	return e
}

// WithUser adds user information to an event
func (e Event) WithUser(userID int64) Event {
	e.UserID = &userID
	return e
}

// WithSession adds session information to an event
func (e Event) WithSession(sessionID string) Event {
	e.SessionID = sessionID
	return e
}

func generateEventID() string {
	return fmt.Sprintf("evt_%d", time.Now().UnixNano())
}
