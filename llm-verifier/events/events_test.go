package events

import (
	"context"
	"fmt"
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
)

// TestEventManager_Complete tests complete event management functionality
func TestEventManager_Complete(t *testing.T) {
	ctx := context.Background()
	manager := NewEventManager(ctx, 100, 2)
	defer manager.Shutdown()
	
	tests := []struct {
		name        string
		eventType   EventType
		severity    Severity
		title       string
		message     string
		details     map[string]interface{}
		validateFunc func(t *testing.T, event *Event)
	}{
		{
			name:      "Model Verification Event",
			eventType: EventVerificationCompleted,
			severity:  SeverityInfo,
			title:     "Model Verification Completed",
			message:   "GPT-4 verification completed successfully",
			details: map[string]interface{}{
				"model_id": "gpt-4",
				"score":    8.5,
				"score_suffix": "(SC:8.5)",
				"duration": 1500,
			},
			validateFunc: func(t *testing.T, event *Event) {
				assert.Equal(t, EventVerificationCompleted, event.Type)
				assert.Equal(t, SeverityInfo, event.Severity)
				assert.Contains(t, event.Message, "GPT-4")
				assert.Equal(t, 8.5, event.Details["score"])
				assert.Contains(t, event.Details["score_suffix"], "(SC:")
			},
		},
		{
			name:      "Security Event",
			eventType: EventSecurityAlert,
			severity:  SeverityWarning,
			title:     "Security Alert",
			message:   "Authentication failure detected",
			details: map[string]interface{}{
				"user_id": "user456",
				"details": "Invalid API key",
				"ip_address": "192.168.1.100",
			},
			validateFunc: func(t *testing.T, event *Event) {
				assert.Equal(t, EventSecurityAlert, event.Type)
				assert.Equal(t, SeverityWarning, event.Severity)
				assert.Contains(t, event.Message, "Authentication failure")
				assert.Equal(t, "Invalid API key", event.Details["details"])
			},
		},
		{
			name:      "Score Update Event",
			eventType: EventScoreChanged,
			severity:  SeverityInfo,
			title:     "Model Score Updated",
			message:   "Model score has been updated",
			details: map[string]interface{}{
				"model_id": "claude-3",
				"old_score": 7.5,
				"new_score": 7.8,
				"score_suffix": "(SC:7.8)",
				"reason": "Performance improvement",
			},
			validateFunc: func(t *testing.T, event *Event) {
				assert.Equal(t, EventScoreChanged, event.Type)
				assert.Equal(t, SeverityInfo, event.Severity)
				assert.Contains(t, event.Message, "score has been updated")
				assert.Equal(t, 7.8, event.Details["new_score"])
				assert.Contains(t, event.Details["score_suffix"], "(SC:")
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := CreateEventWithDetails(tt.eventType, tt.severity, tt.title, tt.message, tt.details)
			err := manager.PublishEvent(event)
			assert.NoError(t, err)
			
			// Give some time for event processing
			time.Sleep(100 * time.Millisecond)
			
			tt.validateFunc(t, event)
		})
	}
}

// TestEventManager_Concurrent tests concurrent event handling
func TestEventManager_Concurrent(t *testing.T) {
	ctx := context.Background()
	manager := NewEventManager(ctx, 1000, 5)
	defer manager.Shutdown()
	
	// Test concurrent event publishing
	done := make(chan bool)
	eventCount := 100
	
	for i := 0; i < eventCount; i++ {
		go func(id int) {
			event := CreateEventWithDetails(
				EventVerificationCompleted,
				SeverityInfo,
				fmt.Sprintf("Verification %d", id),
				fmt.Sprintf("Model model-%d verification completed", id),
				map[string]interface{}{
					"model_id": fmt.Sprintf("model-%d", id),
					"score":    float64(id % 10),
					"score_suffix": fmt.Sprintf("(SC:%.1f)", float64(id%10)),
				},
			)
			err := manager.PublishEvent(event)
			assert.NoError(t, err)
			done <- true
		}(i)
	}
	
	// Wait for all goroutines to complete
	for i := 0; i < eventCount; i++ {
		<-done
	}
	
	// Give time for processing
	time.Sleep(500 * time.Millisecond)
	
	// Verify events were processed
	assert.Equal(t, 0, len(manager.eventBuffer), "All events should be processed")
}

// TestEventManager_Subscribers tests subscriber functionality
func TestEventManager_Subscribers(t *testing.T) {
	ctx := context.Background()
	manager := NewEventManager(ctx, 100, 2)
	defer manager.Shutdown()
	
	// Create a test subscriber
	subscriber := NewWebSocketSubscriber("test-connection", []EventType{EventVerificationCompleted})
	
	err := manager.Subscribe(subscriber)
	assert.NoError(t, err)
	
	assert.Equal(t, 1, manager.GetSubscriberCount())
	
	// Publish an event
	event := CreateEvent(EventVerificationCompleted, SeverityInfo, "Test Verification", "Test verification completed")
	err = manager.PublishEvent(event)
	assert.NoError(t, err)
	
	// Give time for processing
	time.Sleep(100 * time.Millisecond)
	
	// Verify subscriber received the event
	select {
	case receivedEvent := <-subscriber.ReceiveChannel:
		assert.Equal(t, EventVerificationCompleted, receivedEvent.Type)
		assert.Equal(t, "Test Verification", receivedEvent.Title)
	case <-time.After(1 * time.Second):
		t.Fatal("Subscriber did not receive event within timeout")
	}
}

// TestEventManager_ScoreFormat tests score suffix format in events
func TestEventManager_ScoreFormat(t *testing.T) {
	ctx := context.Background()
	manager := NewEventManager(ctx, 100, 2)
	defer manager.Shutdown()
	
	// Create event with score suffix
	event := CreateEventWithDetails(
		EventScoreChanged,
		SeverityInfo,
		"Score Update",
		"Model score updated",
		map[string]interface{}{
			"model_id": "gpt-4",
			"old_score": 8.0,
			"new_score": 8.5,
			"score_suffix": "(SC:8.5)",
		},
	)
	
	// Verify score suffix format
	assert.Contains(t, event.Details["score_suffix"].(string), "(SC:")
	assert.Contains(t, event.Details["score_suffix"].(string), ")")
	
	// Test numeric score validation
	assert.Greater(t, event.Details["new_score"].(float64), 0.0)
	assert.LessOrEqual(t, event.Details["new_score"].(float64), 10.0)
}