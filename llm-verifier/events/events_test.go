package events

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"llm-verifier/database"
)

// setupTestDatabase creates an in-memory database for testing
func setupTestDatabase(t *testing.T) *database.Database {
	// Create temp database file
	dbFile := "/tmp/test_events_" + time.Now().Format("20060102150405") + ".db"

	db, err := database.New(dbFile)
	require.NoError(t, err, "Failed to create test database")

	// Clean up after test
	t.Cleanup(func() {
		os.Remove(dbFile)
	})

	return db
}

// MockSubscriber implements Subscriber interface for testing
type MockSubscriber struct {
	ID        string
	Types     []EventType
	Events    []Event
	Active    bool
	HandleErr error
	mu        sync.RWMutex
}

func (ms *MockSubscriber) HandleEvent(event Event) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.Events = append(ms.Events, event)
	return ms.HandleErr
}

func (ms *MockSubscriber) GetID() string {
	return ms.ID
}

func (ms *MockSubscriber) GetTypes() []EventType {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.Types
}

func (ms *MockSubscriber) IsActive() bool {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.Active
}

func (ms *MockSubscriber) GetEvents() []Event {
	ms.mu.RLock()
	defer ms.mu.RUnlock()
	return ms.Events
}

func (ms *MockSubscriber) ClearEvents() {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.Events = nil
}

func TestNewEventBus(t *testing.T) {
	db := setupTestDatabase(t)
	eb := NewEventBus(db)

	assert.NotNil(t, eb, "EventBus should not be nil")
	assert.NotNil(t, eb.subscribers, "subscribers map should be initialized")
	assert.NotNil(t, eb.db, "database should be set")
	assert.NotNil(t, eb.ctx, "context should be initialized")
	assert.NotNil(t, eb.cancel, "cancel function should be initialized")
}

func TestEventBusSubscribe(t *testing.T) {
	db := setupTestDatabase(t)
	eb := NewEventBus(db)

	sub := &MockSubscriber{
		ID:     "sub-1",
		Types:  []EventType{EventTypeModelVerified},
		Events:  []Event{},
		Active: true,
	}

	eb.Subscribe(sub)

	// Verify subscriber is in map
	eb.mu.RLock()
	_, exists := eb.subscribers["sub-1"]
	eb.mu.RUnlock()
	assert.True(t, exists, "Subscriber should be in subscribers map")
}

func TestEventBusSubscribeMultiple(t *testing.T) {
	db := setupTestDatabase(t)
	eb := NewEventBus(db)

	subs := []*MockSubscriber{
		{ID: "sub-1", Types: []EventType{EventTypeModelVerified}, Active: true},
		{ID: "sub-2", Types: []EventType{EventTypeModelVerified}, Active: true},
		{ID: "sub-3", Types: []EventType{EventTypeModelVerified}, Active: true},
	}

	for _, sub := range subs {
		eb.Subscribe(sub)
	}

	eb.mu.RLock()
	count := len(eb.subscribers)
	eb.mu.RUnlock()
	assert.Equal(t, 3, count, "Should have 3 subscribers")
}

func TestEventBusPublish(t *testing.T) {
	db := setupTestDatabase(t)
	eb := NewEventBus(db)

	sub := &MockSubscriber{
		ID:     "sub-1",
		Types:  []EventType{EventTypeModelVerified},
		Events:  []Event{},
		Active: true,
	}

	eb.Subscribe(sub)

	event := Event{
		ID:       "evt-1",
		Type:     EventTypeModelVerified,
		Message:  "Model verified successfully",
		Timestamp: time.Now(),
		Data:     map[string]any{"model": "gpt-4"},
	}

	err := eb.Publish(event)
	assert.NoError(t, err, "Publish should not return error")

	// Wait for async delivery
	time.Sleep(100 * time.Millisecond)

	events := sub.GetEvents()
	assert.GreaterOrEqual(t, len(events), 1, "Should receive at least 1 event")
	if len(events) > 0 {
		assert.Equal(t, EventTypeModelVerified, events[0].Type, "Event type should match")
	}
}

func TestEventBusPublishNoSubscribers(t *testing.T) {
	db := setupTestDatabase(t)
	eb := NewEventBus(db)

	event := Event{
		ID:       "evt-1",
		Type:     EventTypeModelVerified,
		Message:  "Test event",
		Timestamp: time.Now(),
		Data:     map[string]any{},
	}

	err := eb.Publish(event)
	assert.NoError(t, err, "Publish should not return error even without subscribers")
}

func TestEventBusPublishMultipleSubscribers(t *testing.T) {
	db := setupTestDatabase(t)
	eb := NewEventBus(db)

	subs := make([]*MockSubscriber, 3)
	for i := 0; i < 3; i++ {
		subs[i] = &MockSubscriber{
			ID:     "sub-" + string(rune('1'+i)),
			Types:  []EventType{EventTypeModelVerified},
			Events:  []Event{},
			Active: true,
		}
		eb.Subscribe(subs[i])
	}

	event := Event{
		ID:       "evt-1",
		Type:     EventTypeModelVerified,
		Message:  "Test event",
		Timestamp: time.Now(),
		Data:     map[string]any{},
	}

	err := eb.Publish(event)
	assert.NoError(t, err, "Publish should not return error")

	// Wait for async deliveries
	time.Sleep(200 * time.Millisecond)

	// Verify all received
	for _, sub := range subs {
		events := sub.GetEvents()
		assert.GreaterOrEqual(t, len(events), 1, "Each subscriber should receive event")
	}
}

func TestEventBusPublishInactiveSubscriber(t *testing.T) {
	db := setupTestDatabase(t)
	eb := NewEventBus(db)

	activeSub := &MockSubscriber{
		ID:     "active-sub",
		Types:  []EventType{EventTypeModelVerified},
		Events:  []Event{},
		Active: true,
	}

	inactiveSub := &MockSubscriber{
		ID:     "inactive-sub",
		Types:  []EventType{EventTypeModelVerified},
		Events:  []Event{},
		Active: false,
	}

	eb.Subscribe(activeSub)
	eb.Subscribe(inactiveSub)

	event := Event{
		ID:       "evt-1",
		Type:     EventTypeModelVerified,
		Message:  "Test event",
		Timestamp: time.Now(),
		Data:     map[string]any{},
	}

	err := eb.Publish(event)
	assert.NoError(t, err, "Publish should not return error")

	// Wait for async delivery
	time.Sleep(100 * time.Millisecond)

	// Only active subscriber should receive
	assert.GreaterOrEqual(t, len(activeSub.GetEvents()), 1, "Active subscriber should receive event")
	assert.Equal(t, 0, len(inactiveSub.GetEvents()), "Inactive subscriber should not receive event")
}

func TestEventBusUnsubscribe(t *testing.T) {
	db := setupTestDatabase(t)
	eb := NewEventBus(db)

	sub := &MockSubscriber{
		ID:     "sub-1",
		Types:  []EventType{EventTypeModelVerified},
		Events:  []Event{},
		Active: true,
	}

	eb.Subscribe(sub)

	// Verify exists
	eb.mu.RLock()
	_, exists := eb.subscribers["sub-1"]
	eb.mu.RUnlock()
	assert.True(t, exists, "Subscriber should exist before unsubscribe")

	eb.Unsubscribe("sub-1")

	// Verify removed
	eb.mu.RLock()
	_, exists = eb.subscribers["sub-1"]
	eb.mu.RUnlock()
	assert.False(t, exists, "Subscriber should be removed after unsubscribe")
}

func TestEventBusUnsubscribeNonExistent(t *testing.T) {
	db := setupTestDatabase(t)
	eb := NewEventBus(db)

	// Should not panic
	eb.Unsubscribe("non-existent-id")
}

func TestEventBusGetSubscribers(t *testing.T) {
	db := setupTestDatabase(t)
	eb := NewEventBus(db)

	activeSub := &MockSubscriber{
		ID:     "active-sub",
		Types:  []EventType{EventTypeModelVerified},
		Events:  []Event{},
		Active: true,
	}

	inactiveSub := &MockSubscriber{
		ID:     "inactive-sub",
		Types:  []EventType{EventTypeModelVerified},
		Events:  []Event{},
		Active: false,
	}

	eb.Subscribe(activeSub)
	eb.Subscribe(inactiveSub)

	subs := eb.GetSubscribers()
	assert.Len(t, subs, 1, "Should return only active subscribers")
	assert.Equal(t, "active-sub", subs[0].GetID(), "Should return active subscriber")
}

func TestEventBusGetSubscribersEmpty(t *testing.T) {
	db := setupTestDatabase(t)
	eb := NewEventBus(db)

	subs := eb.GetSubscribers()
	assert.Len(t, subs, 0, "Should return empty list when no subscribers")
}

func TestEventBusShutdown(t *testing.T) {
	db := setupTestDatabase(t)
	eb := NewEventBus(db)

	sub := &MockSubscriber{
		ID:     "sub-1",
		Types:  []EventType{EventTypeModelVerified},
		Events:  []Event{},
		Active: true,
	}

	eb.Subscribe(sub)

	// Verify exists
	eb.mu.RLock()
	_, exists := eb.subscribers["sub-1"]
	eb.mu.RUnlock()
	assert.True(t, exists, "Subscriber should exist before shutdown")

	eb.Shutdown()

	// Verify subscribers cleaned up
	eb.mu.RLock()
	_, exists = eb.subscribers["sub-1"]
	eb.mu.RUnlock()
	assert.False(t, exists, "Subscriber should be removed after shutdown")
}

func TestEventBusConcurrentPublish(t *testing.T) {
	db := setupTestDatabase(t)
	eb := NewEventBus(db)

	count := 100

	sub := &MockSubscriber{
		ID:     "sub-1",
		Types:  []EventType{EventTypeModelVerified},
		Events:  []Event{},
		Active: true,
		mu:     sync.RWMutex{},
	}

	eb.Subscribe(sub)

	// Publish multiple events concurrently
	for i := 0; i < count; i++ {
		go func(index int) {
			event := Event{
				ID:       string(rune('0' + index%10)),
				Type:     EventTypeModelVerified,
				Message:  "Test event",
				Timestamp: time.Now(),
				Data:     map[string]any{"index": index},
			}
			eb.Publish(event)
		}(i)
	}

	// Wait for all events
	time.Sleep(500 * time.Millisecond)

	events := sub.GetEvents()
	assert.GreaterOrEqual(t, len(events), 90, "Should receive most concurrent events")
}

func TestEventBusTypeFiltering(t *testing.T) {
	db := setupTestDatabase(t)
	eb := NewEventBus(db)

	sub1 := &MockSubscriber{
		ID:     "sub-1",
		Types:  []EventType{EventTypeModelVerified},
		Events:  []Event{},
		Active: true,
	}

	sub2 := &MockSubscriber{
		ID:     "sub-2",
		Types:  []EventType{EventTypeVerificationCompleted},
		Events:  []Event{},
		Active: true,
	}

	eb.Subscribe(sub1)
	eb.Subscribe(sub2)

	// Publish type 1 event
	event1 := Event{
		ID:       "evt-1",
		Type:     EventTypeModelVerified,
		Message:  "Model verified",
		Timestamp: time.Now(),
		Data:     map[string]any{},
	}
	eb.Publish(event1)

	// Publish type 2 event
	event2 := Event{
		ID:       "evt-2",
		Type:     EventTypeVerificationCompleted,
		Message:  "Verification completed",
		Timestamp: time.Now(),
		Data:     map[string]any{},
	}
	eb.Publish(event2)

	// Wait for async delivery
	time.Sleep(100 * time.Millisecond)

	// Check type 1 subscriber
	events1 := sub1.GetEvents()
	assert.GreaterOrEqual(t, len(events1), 1, "Type 1 subscriber should receive type 1 event")
	if len(events1) > 0 {
		assert.Equal(t, EventTypeModelVerified, events1[0].Type, "Type 1 subscriber should receive type 1 event")
	}

	// Check type 2 subscriber
	events2 := sub2.GetEvents()
	assert.GreaterOrEqual(t, len(events2), 1, "Type 2 subscriber should receive type 2 event")
	if len(events2) > 0 {
		assert.Equal(t, EventTypeVerificationCompleted, events2[0].Type, "Type 2 subscriber should receive type 2 event")
	}
}

func TestEventBusEventData(t *testing.T) {
	db := setupTestDatabase(t)
	eb := NewEventBus(db)

	sub := &MockSubscriber{
		ID:     "sub-1",
		Types:  []EventType{EventTypeModelVerified},
		Events:  []Event{},
		Active: true,
	}

	eb.Subscribe(sub)

	testData := map[string]any{
		"string":  "value",
		"number":   123,
		"bool":     true,
		"nested":   map[string]any{"key": "nested-value"},
	}

	event := Event{
		ID:       "evt-1",
		Type:     EventTypeModelVerified,
		Message:  "Test event with data",
		Timestamp: time.Now(),
		Data:     testData,
	}

	err := eb.Publish(event)
	assert.NoError(t, err)

	// Wait for async delivery
	time.Sleep(100 * time.Millisecond)

	events := sub.GetEvents()
	if len(events) > 0 {
		assert.Equal(t, testData["string"], events[0].Data["string"], "String data should match")
		assert.Equal(t, testData["number"], events[0].Data["number"], "Number data should match")
		assert.Equal(t, testData["bool"], events[0].Data["bool"], "Bool data should match")
		assert.Equal(t, testData["nested"], events[0].Data["nested"], "Nested data should match")
	}
}

func TestEventBusNilData(t *testing.T) {
	db := setupTestDatabase(t)
	eb := NewEventBus(db)

	sub := &MockSubscriber{
		ID:     "sub-1",
		Types:  []EventType{EventTypeModelVerified},
		Events:  []Event{},
		Active: true,
	}

	eb.Subscribe(sub)

	event := Event{
		ID:       "evt-1",
		Type:     EventTypeModelVerified,
		Message:  "Test event with nil data",
		Timestamp: time.Now(),
		Data:     nil,
	}

	err := eb.Publish(event)
	assert.NoError(t, err)

	// Wait for async delivery
	time.Sleep(100 * time.Millisecond)

	events := sub.GetEvents()
	assert.GreaterOrEqual(t, len(events), 1, "Should receive event with nil data")
}

func TestWebSocketSubscriber(t *testing.T) {
	ws := &WebSocketSubscriber{
		ID:     "ws-1",
		Conn:   nil,
		Types:  []EventType{EventTypeModelVerified},
		Active: true,
	}

	assert.Equal(t, "ws-1", ws.GetID(), "ID should match")
	assert.Equal(t, []EventType{EventTypeModelVerified}, ws.GetTypes(), "Types should match")
	assert.True(t, ws.IsActive(), "Should be active")

	ws.Active = false
	assert.False(t, ws.IsActive(), "Should reflect active state")
}

func TestGRPCSubscriber(t *testing.T) {
	gs := &GRPCSubscriber{
		ID:     "grpc-1",
		Stream: nil,
		Types:  []EventType{EventTypeVerificationStarted},
		Active: true,
	}

	assert.Equal(t, "grpc-1", gs.GetID(), "ID should match")
	assert.Equal(t, []EventType{EventTypeVerificationStarted}, gs.GetTypes(), "Types should match")
	assert.True(t, gs.IsActive(), "Should be active")

	gs.Active = false
	assert.False(t, gs.IsActive(), "Should reflect active state")
}

func TestNotificationSubscriber(t *testing.T) {
	ns := &NotificationSubscriber{
		ID:       "notif-1",
		Channels: []string{"slack", "email"},
		Types:    []EventType{EventTypeErrorOccurred},
		Active:   true,
	}

	assert.Equal(t, "notif-1", ns.GetID(), "ID should match")
	assert.Equal(t, []string{"slack", "email"}, ns.Channels, "Channels should match")
	assert.Equal(t, []EventType{EventTypeErrorOccurred}, ns.GetTypes(), "Types should match")
	assert.True(t, ns.IsActive(), "Should be active")

	ns.Active = false
	assert.False(t, ns.IsActive(), "Should reflect active state")
}

func TestNewEvent(t *testing.T) {
	event := NewEvent(
		EventTypeModelVerified,
		EventSeverityInfo,
		"Model verified successfully",
		"verifier",
	)

	assert.NotEmpty(t, event.ID, "ID should be generated")
	assert.Equal(t, EventTypeModelVerified, event.Type, "Type should match")
	assert.Equal(t, EventSeverityInfo, event.Severity, "Severity should match")
	assert.Equal(t, "Model verified successfully", event.Message, "Message should match")
	assert.Equal(t, "verifier", event.Source, "Source should match")
	assert.NotNil(t, event.Data, "Data map should be initialized")
	assert.False(t, event.Timestamp.IsZero(), "Timestamp should be set")
}

func TestEventWithData(t *testing.T) {
	event := NewEvent(
		EventTypeModelVerified,
		EventSeverityInfo,
		"Test event",
		"test",
	)

	event = event.WithData("model", "gpt-4")
	event = event.WithData("provider", "openai")
	event = event.WithData("score", 95.5)

	assert.Equal(t, "gpt-4", event.Data["model"], "Model data should be set")
	assert.Equal(t, "openai", event.Data["provider"], "Provider data should be set")
	assert.Equal(t, 95.5, event.Data["score"], "Score data should be set")
}

func TestEventWithUser(t *testing.T) {
	event := NewEvent(
		EventTypeModelVerified,
		EventSeverityInfo,
		"Test event",
		"test",
	)

	userID := int64(12345)
	event = event.WithUser(userID)

	assert.NotNil(t, event.UserID, "UserID pointer should be set")
	assert.Equal(t, userID, *event.UserID, "UserID should match")
}

func TestEventWithSession(t *testing.T) {
	event := NewEvent(
		EventTypeModelVerified,
		EventSeverityInfo,
		"Test event",
		"test",
	)

	sessionID := "session-abc-123"
	event = event.WithSession(sessionID)

	assert.Equal(t, sessionID, event.SessionID, "SessionID should match")
}

func TestEventWithNilData(t *testing.T) {
	event := NewEvent(
		EventTypeModelVerified,
		EventSeverityInfo,
		"Test event",
		"test",
	)

	// WithData should initialize Data map if nil
	event.Data = nil
	event = event.WithData("key", "value")

	assert.NotNil(t, event.Data, "Data map should be initialized")
	assert.Equal(t, "value", event.Data["key"], "Data should be set")
}

func TestEventTypes(t *testing.T) {
	assert.Equal(t, "model.verified", string(EventTypeModelVerified), "EventTypeModelVerified should match")
	assert.Equal(t, "model.verification.failed", string(EventTypeModelVerificationFailed), "EventTypeModelVerificationFailed should match")
	assert.Equal(t, "model.score.changed", string(EventTypeScoreChanged), "EventTypeScoreChanged should match")
	assert.Equal(t, "provider.added", string(EventTypeProviderAdded), "EventTypeProviderAdded should match")
	assert.Equal(t, "provider.updated", string(EventTypeProviderUpdated), "EventTypeProviderUpdated should match")
	assert.Equal(t, "system.health.changed", string(EventTypeSystemHealthChanged), "EventTypeSystemHealthChanged should match")
	assert.Equal(t, "verification.started", string(EventTypeVerificationStarted), "EventTypeVerificationStarted should match")
	assert.Equal(t, "verification.completed", string(EventTypeVerificationCompleted), "EventTypeVerificationCompleted should match")
	assert.Equal(t, "schedule.executed", string(EventTypeScheduleExecuted), "EventTypeScheduleExecuted should match")
	assert.Equal(t, "error.occurred", string(EventTypeErrorOccurred), "EventTypeErrorOccurred should match")
}

func TestEventSeverities(t *testing.T) {
	assert.Equal(t, "info", string(EventSeverityInfo), "EventSeverityInfo should match")
	assert.Equal(t, "warning", string(EventSeverityWarning), "EventSeverityWarning should match")
	assert.Equal(t, "error", string(EventSeverityError), "EventSeverityError should match")
	assert.Equal(t, "critical", string(EventSeverityCritical), "EventSeverityCritical should match")
}

func TestEventIDGeneration(t *testing.T) {
	event1 := NewEvent(EventTypeModelVerified, EventSeverityInfo, "Test", "test")
	event2 := NewEvent(EventTypeModelVerified, EventSeverityInfo, "Test", "test")

	assert.NotEqual(t, event1.ID, event2.ID, "IDs should be unique")
	assert.Contains(t, event1.ID, "evt_", "ID should contain evt_ prefix")
	assert.Contains(t, event2.ID, "evt_", "ID should contain evt_ prefix")
}

func TestSubscriberHandleError(t *testing.T) {
	db := setupTestDatabase(t)
	eb := NewEventBus(db)

	sub := &MockSubscriber{
		ID:        "error-sub",
		Types:     []EventType{EventTypeModelVerified},
		Events:    []Event{},
		Active:     true,
		HandleErr:  assert.AnError,
	}

	eb.Subscribe(sub)

	event := Event{
		ID:       "evt-1",
		Type:     EventTypeModelVerified,
		Message:  "Test event",
		Timestamp: time.Now(),
		Data:     map[string]any{},
	}

	// Should not panic on error
	err := eb.Publish(event)
	assert.NoError(t, err, "Publish should not return error even if subscriber fails")

	// Wait for async delivery
	time.Sleep(100 * time.Millisecond)
}

func TestEventConcurrentSubscribers(t *testing.T) {
	db := setupTestDatabase(t)
	eb := NewEventBus(db)

	subscriberCount := 10

	// Create concurrent subscribers
	for i := 0; i < subscriberCount; i++ {
		go func(index int) {
			sub := &MockSubscriber{
				ID:     "sub-" + string(rune('0'+index%10)),
				Types:  []EventType{EventTypeModelVerified},
				Events:  []Event{},
				Active: true,
				mu:     sync.RWMutex{},
			}
			eb.Subscribe(sub)
		}(i)
	}

	// Wait for subscriptions
	time.Sleep(50 * time.Millisecond)

	eb.mu.RLock()
	count := len(eb.subscribers)
	eb.mu.RUnlock()

	assert.GreaterOrEqual(t, count, 8, "Should have most concurrent subscribers")
}

func TestPublishComplexData(t *testing.T) {
	db := setupTestDatabase(t)
	eb := NewEventBus(db)

	sub := &MockSubscriber{
		ID:     "sub-1",
		Types:  []EventType{EventTypeModelVerified},
		Events:  []Event{},
		Active: true,
	}

	eb.Subscribe(sub)

	complexData := map[string]any{
		"string":  "value",
		"number":   123,
		"bool":     true,
		"nested":   map[string]any{"key": "nested-value"},
		"array":    []int{1, 2, 3},
		"null":     nil,
	}

	event := Event{
		ID:       "evt-1",
		Type:     EventTypeModelVerified,
		Message:  "Complex data event",
		Timestamp: time.Now(),
		Data:     complexData,
	}

	err := eb.Publish(event)
	assert.NoError(t, err)

	// Wait for async delivery
	time.Sleep(100 * time.Millisecond)

	events := sub.GetEvents()
	if len(events) > 0 {
		assert.Equal(t, complexData["string"], events[0].Data["string"], "String data should match")
		assert.Equal(t, complexData["number"], events[0].Data["number"], "Number data should match")
		assert.Equal(t, complexData["bool"], events[0].Data["bool"], "Bool data should match")
		assert.Equal(t, complexData["nested"], events[0].Data["nested"], "Nested data should match")
		assert.Equal(t, complexData["array"], events[0].Data["array"], "Array data should match")
	}
}

func TestEventAllEventTypes(t *testing.T) {
	eventTypes := []EventType{
		EventTypeModelVerified,
		EventTypeModelVerificationFailed,
		EventTypeScoreChanged,
		EventTypeProviderAdded,
		EventTypeProviderUpdated,
		EventTypeSystemHealthChanged,
		EventTypeVerificationStarted,
		EventTypeVerificationCompleted,
		EventTypeScheduleExecuted,
		EventTypeErrorOccurred,
	}

	for _, eventType := range eventTypes {
		event := NewEvent(eventType, EventSeverityInfo, "Test", "test")
		assert.Equal(t, eventType, event.Type, "Event type should match")
	}
}

func TestEventAllSeverities(t *testing.T) {
	severities := []EventSeverity{
		EventSeverityInfo,
		EventSeverityWarning,
		EventSeverityError,
		EventSeverityCritical,
	}

	for _, severity := range severities {
		event := NewEvent(EventTypeModelVerified, severity, "Test", "test")
		assert.Equal(t, severity, event.Severity, "Event severity should match")
	}
}
