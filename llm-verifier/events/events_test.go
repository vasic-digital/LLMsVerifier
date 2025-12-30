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

// TestEventManager_Unsubscribe tests unsubscribing from events
func TestEventManager_Unsubscribe(t *testing.T) {
	ctx := context.Background()
	manager := NewEventManager(ctx, 100, 2)
	defer manager.Shutdown()

	// Create and subscribe
	subscriber := NewWebSocketSubscriber("test-unsub", []EventType{EventVerificationCompleted})
	err := manager.Subscribe(subscriber)
	assert.NoError(t, err)
	assert.Equal(t, 1, manager.GetSubscriberCount())

	// Unsubscribe using the full ID (which includes ws_ prefix)
	manager.Unsubscribe(subscriber.GetID())
	assert.Equal(t, 0, manager.GetSubscriberCount())
}

// TestEventManager_GetSubscribers tests getting all subscribers
func TestEventManager_GetSubscribers(t *testing.T) {
	ctx := context.Background()
	manager := NewEventManager(ctx, 100, 2)
	defer manager.Shutdown()

	// Add multiple subscribers
	sub1 := NewWebSocketSubscriber("sub-1", []EventType{EventVerificationCompleted})
	sub2 := NewWebSocketSubscriber("sub-2", []EventType{EventScoreChanged})

	manager.Subscribe(sub1)
	manager.Subscribe(sub2)

	subscribers := manager.GetSubscribers()
	assert.Equal(t, 2, len(subscribers))
}

// TestCreateModelEvent tests creating model-specific events
func TestCreateModelEvent(t *testing.T) {
	event := CreateModelEvent(
		EventVerificationCompleted,
		SeverityInfo,
		"Model Verified",
		"Verification completed for GPT-4",
		1,
	)

	assert.Equal(t, EventVerificationCompleted, event.Type)
	assert.Equal(t, SeverityInfo, event.Severity)
	assert.NotNil(t, event.ModelID)
	assert.Equal(t, int64(1), *event.ModelID)
	assert.Contains(t, event.Message, "GPT-4")
}

// TestCreateProviderEvent tests creating provider-specific events
func TestCreateProviderEvent(t *testing.T) {
	event := CreateProviderEvent(
		EventProviderAdded,
		SeverityInfo,
		"Provider Added",
		"Provider OpenAI added successfully",
		1,
	)

	assert.Equal(t, EventProviderAdded, event.Type)
	assert.Equal(t, SeverityInfo, event.Severity)
	assert.NotNil(t, event.ProviderID)
	assert.Equal(t, int64(1), *event.ProviderID)
	assert.Contains(t, event.Message, "OpenAI")
}

// TestCreateVerificationEvent tests creating verification-specific events
func TestCreateVerificationEvent(t *testing.T) {
	event := CreateVerificationEvent(
		EventVerificationStarted,
		SeverityInfo,
		"Verification Started",
		"Starting verification for model",
		100,
	)

	assert.Equal(t, EventVerificationStarted, event.Type)
	assert.NotNil(t, event.VerificationID)
	assert.Equal(t, int64(100), *event.VerificationID)
}

// TestCreateClientEvent tests creating client-specific events
func TestCreateClientEvent(t *testing.T) {
	event := CreateClientEvent(
		EventClientConnected,
		SeverityInfo,
		"Client Connected",
		"Client connected from 192.168.1.100",
		"client-123",
	)

	assert.Equal(t, EventClientConnected, event.Type)
	assert.NotNil(t, event.ClientID)
	assert.Equal(t, "client-123", *event.ClientID)
	assert.Contains(t, event.Message, "192.168.1.100")
}

// TestNewEventPublisher tests event publisher creation
func TestNewEventPublisher(t *testing.T) {
	ctx := context.Background()
	manager := NewEventManager(ctx, 100, 2)
	defer manager.Shutdown()

	publisher := NewEventPublisher(manager, nil)
	assert.NotNil(t, publisher)
}

// TestEventPublisher_PublishVerificationStarted tests publishing verification started events
func TestEventPublisher_PublishVerificationStarted(t *testing.T) {
	ctx := context.Background()
	manager := NewEventManager(ctx, 100, 2)
	defer manager.Shutdown()

	publisher := NewEventPublisher(manager, nil)

	err := publisher.PublishVerificationStarted(10, 3)
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
}

// TestEventPublisher_PublishVerificationCompleted tests publishing verification completed events
func TestEventPublisher_PublishVerificationCompleted(t *testing.T) {
	ctx := context.Background()
	manager := NewEventManager(ctx, 100, 2)
	defer manager.Shutdown()

	publisher := NewEventPublisher(manager, nil)

	err := publisher.PublishVerificationCompleted(1500*time.Millisecond, 8, 2)
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
}

// TestEventPublisher_PublishVerificationFailed tests publishing verification failed events
func TestEventPublisher_PublishVerificationFailed(t *testing.T) {
	ctx := context.Background()
	manager := NewEventManager(ctx, 100, 2)
	defer manager.Shutdown()

	publisher := NewEventPublisher(manager, nil)

	err := publisher.PublishVerificationFailed("Timeout error")
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
}

// TestEventPublisher_PublishScoreChanged tests publishing score changed events
func TestEventPublisher_PublishScoreChanged(t *testing.T) {
	ctx := context.Background()
	manager := NewEventManager(ctx, 100, 2)
	defer manager.Shutdown()

	publisher := NewEventPublisher(manager, nil)

	t.Run("score increased", func(t *testing.T) {
		err := publisher.PublishScoreChanged(1, 80, 85, "overall")
		assert.NoError(t, err)
	})

	t.Run("score decreased", func(t *testing.T) {
		err := publisher.PublishScoreChanged(1, 85, 80, "overall")
		assert.NoError(t, err)
	})

	t.Run("no change", func(t *testing.T) {
		err := publisher.PublishScoreChanged(1, 80, 80, "overall")
		assert.NoError(t, err)
	})

	time.Sleep(50 * time.Millisecond)
}

// TestEventPublisher_PublishIssueDetected tests publishing issue detected events
func TestEventPublisher_PublishIssueDetected(t *testing.T) {
	ctx := context.Background()
	manager := NewEventManager(ctx, 100, 2)
	defer manager.Shutdown()

	publisher := NewEventPublisher(manager, nil)

	err := publisher.PublishIssueDetected(1, "performance_degradation", "warning", "Performance Issue", "Response time increased")
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
}

// TestEventPublisher_PublishIssueResolved tests publishing issue resolved events
func TestEventPublisher_PublishIssueResolved(t *testing.T) {
	ctx := context.Background()
	manager := NewEventManager(ctx, 100, 2)
	defer manager.Shutdown()

	publisher := NewEventPublisher(manager, nil)

	err := publisher.PublishIssueResolved(1, 100, "Response time normalized")
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
}

// TestEventPublisher_PublishClientConnected tests publishing client connected events
func TestEventPublisher_PublishClientConnected(t *testing.T) {
	ctx := context.Background()
	manager := NewEventManager(ctx, 100, 2)
	defer manager.Shutdown()

	publisher := NewEventPublisher(manager, nil)

	err := publisher.PublishClientConnected("client-123", "web")
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
}

// TestEventPublisher_PublishClientDisconnected tests publishing client disconnected events
func TestEventPublisher_PublishClientDisconnected(t *testing.T) {
	ctx := context.Background()
	manager := NewEventManager(ctx, 100, 2)
	defer manager.Shutdown()

	publisher := NewEventPublisher(manager, nil)

	err := publisher.PublishClientDisconnected("client-123", "web")
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
}

// TestEventPublisher_PublishSystemHealthChanged tests publishing system health changed events
func TestEventPublisher_PublishSystemHealthChanged(t *testing.T) {
	ctx := context.Background()
	manager := NewEventManager(ctx, 100, 2)
	defer manager.Shutdown()

	publisher := NewEventPublisher(manager, nil)

	t.Run("healthy", func(t *testing.T) {
		err := publisher.PublishSystemHealthChanged("healthy", map[string]interface{}{"cpu": 30.0})
		assert.NoError(t, err)
	})

	t.Run("degraded", func(t *testing.T) {
		err := publisher.PublishSystemHealthChanged("degraded", map[string]interface{}{"cpu": 80.0})
		assert.NoError(t, err)
	})

	t.Run("unhealthy", func(t *testing.T) {
		err := publisher.PublishSystemHealthChanged("unhealthy", map[string]interface{}{"cpu": 95.0})
		assert.NoError(t, err)
	})

	t.Run("critical", func(t *testing.T) {
		err := publisher.PublishSystemHealthChanged("critical", map[string]interface{}{"cpu": 100.0})
		assert.NoError(t, err)
	})

	t.Run("unknown status", func(t *testing.T) {
		err := publisher.PublishSystemHealthChanged("unknown", map[string]interface{}{})
		assert.NoError(t, err)
	})

	time.Sleep(50 * time.Millisecond)
}

// TestEventPublisher_PublishConfigExported tests publishing config exported events
func TestEventPublisher_PublishConfigExported(t *testing.T) {
	ctx := context.Background()
	manager := NewEventManager(ctx, 100, 2)
	defer manager.Shutdown()

	publisher := NewEventPublisher(manager, nil)

	err := publisher.PublishConfigExported("model_scores", 10)
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
}

// TestEventPublisher_PublishSecurityAlert tests publishing security alert events
func TestEventPublisher_PublishSecurityAlert(t *testing.T) {
	ctx := context.Background()
	manager := NewEventManager(ctx, 100, 2)
	defer manager.Shutdown()

	publisher := NewEventPublisher(manager, nil)

	err := publisher.PublishSecurityAlert("unauthorized_access", "Failed login attempt", map[string]interface{}{
		"ip_address": "192.168.1.100",
		"user_id":    "admin",
	})
	assert.NoError(t, err)

	time.Sleep(50 * time.Millisecond)
}

// TestEventPublisher_PublishDatabaseMigration tests publishing database migration events
func TestEventPublisher_PublishDatabaseMigration(t *testing.T) {
	ctx := context.Background()
	manager := NewEventManager(ctx, 100, 2)
	defer manager.Shutdown()

	publisher := NewEventPublisher(manager, nil)

	t.Run("successful migration", func(t *testing.T) {
		err := publisher.PublishDatabaseMigration(5, "Add scoring tables", true)
		assert.NoError(t, err)
	})

	t.Run("failed migration", func(t *testing.T) {
		err := publisher.PublishDatabaseMigration(6, "Add invalid column", false)
		assert.NoError(t, err)
	})

	time.Sleep(50 * time.Millisecond)
}

// TestGRPCSubscriber tests GRPC subscriber functionality
func TestGRPCSubscriber(t *testing.T) {
	t.Run("creation with nil callback", func(t *testing.T) {
		subscriber := NewGRPCSubscriber("grpc-client-1", []EventType{EventVerificationCompleted}, nil)
		assert.NotNil(t, subscriber)
		assert.Contains(t, subscriber.GetID(), "grpc-client-1")
	})

	t.Run("is active check", func(t *testing.T) {
		subscriber := NewGRPCSubscriber("grpc-client-2", []EventType{EventVerificationCompleted}, nil)
		// Initially active but with nil callback, receive will fail
		assert.True(t, subscriber.IsActive())
	})

	t.Run("supported event types", func(t *testing.T) {
		eventTypes := []EventType{EventVerificationCompleted, EventScoreChanged}
		subscriber := NewGRPCSubscriber("grpc-client-3", eventTypes, nil)
		supportedTypes := subscriber.GetSupportedEventTypes()
		assert.Equal(t, eventTypes, supportedTypes)
	})

	t.Run("receive event with callback", func(t *testing.T) {
		eventReceived := false
		callback := func(e *Event) error {
			eventReceived = true
			return nil
		}
		subscriber := NewGRPCSubscriber("grpc-client-4", []EventType{EventVerificationCompleted}, callback)
		event := CreateEvent(EventVerificationCompleted, SeverityInfo, "Test", "Test message")

		err := subscriber.ReceiveEvent(event)
		assert.NoError(t, err)
		assert.True(t, eventReceived)
	})

	t.Run("receive event with nil callback fails", func(t *testing.T) {
		subscriber := NewGRPCSubscriber("grpc-client-5", []EventType{EventVerificationCompleted}, nil)
		event := CreateEvent(EventVerificationCompleted, SeverityInfo, "Test", "Test message")

		err := subscriber.ReceiveEvent(event)
		assert.Error(t, err)
	})
}

// TestWebSocketSubscriber tests WebSocket subscriber functionality
func TestWebSocketSubscriber(t *testing.T) {
	t.Run("creation", func(t *testing.T) {
		subscriber := NewWebSocketSubscriber("ws-client-1", []EventType{EventVerificationCompleted})
		assert.NotNil(t, subscriber)
		// ID has ws_ prefix
		assert.Equal(t, "ws_ws-client-1", subscriber.GetID())
	})

	t.Run("is active", func(t *testing.T) {
		subscriber := NewWebSocketSubscriber("ws-client-2", []EventType{EventVerificationCompleted})
		assert.True(t, subscriber.IsActive())
	})

	t.Run("supported event types", func(t *testing.T) {
		eventTypes := []EventType{EventVerificationCompleted, EventScoreChanged}
		subscriber := NewWebSocketSubscriber("ws-client-3", eventTypes)
		supportedTypes := subscriber.GetSupportedEventTypes()
		assert.Equal(t, eventTypes, supportedTypes)
	})

	t.Run("receive event", func(t *testing.T) {
		subscriber := NewWebSocketSubscriber("ws-client-4", []EventType{EventVerificationCompleted})
		event := CreateEvent(EventVerificationCompleted, SeverityInfo, "Test", "Test message")

		go func() {
			err := subscriber.ReceiveEvent(event)
			assert.NoError(t, err)
		}()

		select {
		case received := <-subscriber.ReceiveChannel:
			assert.Equal(t, event.ID, received.ID)
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for event")
		}
	})
}

// ==================== Email Notifier Tests ====================

func TestNewEmailNotifier(t *testing.T) {
	notifier := NewEmailNotifier(
		"smtp.example.com",
		587,
		"user@example.com",
		"password",
		"from@example.com",
		[]string{"to@example.com", "to2@example.com"},
	)

	assert.NotNil(t, notifier)
	assert.Equal(t, "smtp.example.com", notifier.smtpServer)
	assert.Equal(t, 587, notifier.smtpPort)
	assert.Equal(t, "user@example.com", notifier.username)
	assert.Equal(t, "from@example.com", notifier.fromAddress)
	assert.Len(t, notifier.toAddresses, 2)
}

func TestEmailNotifier_SendNotification_BuildsMessage(t *testing.T) {
	notifier := NewEmailNotifier("smtp.example.com", 587, "user", "pass", "from@test.com", []string{"to@test.com"})

	modelID := int64(42)
	providerID := int64(99)
	clientID := "client-123"

	event := &Event{
		ID:         "test-event",
		Type:       EventVerificationCompleted,
		Severity:   SeverityInfo,
		Title:      "Test Title",
		Message:    "Test Message",
		Source:     "test",
		Timestamp:  time.Now(),
		ModelID:    &modelID,
		ProviderID: &providerID,
		ClientID:   &clientID,
	}

	// SendNotification will fail due to no SMTP server, but we verify the notifier is set up correctly
	err := notifier.SendNotification(event)
	// Expected to fail as there's no actual SMTP server
	assert.Error(t, err)
}

// ==================== Slack Notifier Tests ====================

func TestNewSlackNotifier(t *testing.T) {
	notifier := NewSlackNotifier(
		"https://hooks.slack.com/services/xxx",
		"#general",
		"LLM-Verifier",
	)

	assert.NotNil(t, notifier)
	assert.Equal(t, "https://hooks.slack.com/services/xxx", notifier.webhookURL)
	assert.Equal(t, "#general", notifier.channel)
	assert.Equal(t, "LLM-Verifier", notifier.username)
}

func TestSlackNotifier_getSeverityIcon(t *testing.T) {
	notifier := NewSlackNotifier("", "", "")

	testCases := []struct {
		severity Severity
		expected string
	}{
		{SeverityCritical, ":fire:"},
		{SeverityError, ":x:"},
		{SeverityWarning, ":warning:"},
		{SeverityInfo, ":information_source:"},
		{SeverityDebug, ":bug:"},
		{Severity("unknown"), ":bell:"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.severity), func(t *testing.T) {
			icon := notifier.getSeverityIcon(tc.severity)
			assert.Equal(t, tc.expected, icon)
		})
	}
}

func TestSlackNotifier_getSeverityColor(t *testing.T) {
	notifier := NewSlackNotifier("", "", "")

	testCases := []struct {
		severity Severity
		expected string
	}{
		{SeverityCritical, "danger"},
		{SeverityError, "danger"},
		{SeverityWarning, "warning"},
		{SeverityInfo, "good"},
		{SeverityDebug, "#808080"},
		{Severity("unknown"), "#808080"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.severity), func(t *testing.T) {
			color := notifier.getSeverityColor(tc.severity)
			assert.Equal(t, tc.expected, color)
		})
	}
}

func TestSlackNotifier_buildFields(t *testing.T) {
	notifier := NewSlackNotifier("", "", "")

	t.Run("basic fields", func(t *testing.T) {
		event := &Event{
			Severity: SeverityInfo,
			Source:   "test-source",
		}
		fields := notifier.buildFields(event)
		assert.Len(t, fields, 2)
	})

	t.Run("with model id", func(t *testing.T) {
		modelID := int64(42)
		event := &Event{
			Severity: SeverityInfo,
			Source:   "test-source",
			ModelID:  &modelID,
		}
		fields := notifier.buildFields(event)
		assert.Len(t, fields, 3)
	})

	t.Run("with provider id", func(t *testing.T) {
		providerID := int64(10)
		event := &Event{
			Severity:   SeverityInfo,
			Source:     "test-source",
			ProviderID: &providerID,
		}
		fields := notifier.buildFields(event)
		assert.Len(t, fields, 3)
	})

	t.Run("with client id", func(t *testing.T) {
		clientID := "client-123"
		event := &Event{
			Severity: SeverityInfo,
			Source:   "test-source",
			ClientID: &clientID,
		}
		fields := notifier.buildFields(event)
		assert.Len(t, fields, 3)
	})

	t.Run("with all ids", func(t *testing.T) {
		modelID := int64(42)
		providerID := int64(10)
		clientID := "client-123"
		event := &Event{
			Severity:   SeverityInfo,
			Source:     "test-source",
			ModelID:    &modelID,
			ProviderID: &providerID,
			ClientID:   &clientID,
		}
		fields := notifier.buildFields(event)
		assert.Len(t, fields, 5)
	})
}

// ==================== Telegram Notifier Tests ====================

func TestNewTelegramNotifier(t *testing.T) {
	notifier := NewTelegramNotifier("bot-token-123", "chat-id-456")

	assert.NotNil(t, notifier)
	assert.Equal(t, "bot-token-123", notifier.botToken)
	assert.Equal(t, "chat-id-456", notifier.chatID)
}

func TestTelegramNotifier_getSeverityEmoji(t *testing.T) {
	notifier := NewTelegramNotifier("", "")

	testCases := []struct {
		severity Severity
		expected string
	}{
		{SeverityCritical, "🔴"},
		{SeverityError, "❌"},
		{SeverityWarning, "⚠️"},
		{SeverityInfo, "ℹ️"},
		{SeverityDebug, "🐛"},
		{Severity("unknown"), "🔔"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.severity), func(t *testing.T) {
			emoji := notifier.getSeverityEmoji(tc.severity)
			assert.Equal(t, tc.expected, emoji)
		})
	}
}

func TestTelegramNotifier_buildMessage(t *testing.T) {
	notifier := NewTelegramNotifier("", "")

	t.Run("basic message", func(t *testing.T) {
		event := &Event{
			Title:     "Test Title",
			Message:   "Test Message",
			Severity:  SeverityInfo,
			Source:    "test",
			Timestamp: time.Now(),
		}
		msg := notifier.buildMessage(event)
		assert.Contains(t, msg, "Test Title")
		assert.Contains(t, msg, "Test Message")
		assert.Contains(t, msg, "ℹ️")
	})

	t.Run("with model id", func(t *testing.T) {
		modelID := int64(42)
		event := &Event{
			Title:     "Test",
			Message:   "Test",
			Severity:  SeverityInfo,
			Source:    "test",
			Timestamp: time.Now(),
			ModelID:   &modelID,
		}
		msg := notifier.buildMessage(event)
		assert.Contains(t, msg, "Model ID")
		assert.Contains(t, msg, "42")
	})

	t.Run("with provider id", func(t *testing.T) {
		providerID := int64(10)
		event := &Event{
			Title:      "Test",
			Message:    "Test",
			Severity:   SeverityInfo,
			Source:     "test",
			Timestamp:  time.Now(),
			ProviderID: &providerID,
		}
		msg := notifier.buildMessage(event)
		assert.Contains(t, msg, "Provider ID")
	})

	t.Run("with client id", func(t *testing.T) {
		clientID := "client-abc"
		event := &Event{
			Title:     "Test",
			Message:   "Test",
			Severity:  SeverityInfo,
			Source:    "test",
			Timestamp: time.Now(),
			ClientID:  &clientID,
		}
		msg := notifier.buildMessage(event)
		assert.Contains(t, msg, "Client ID")
		assert.Contains(t, msg, "client-abc")
	})
}

// ==================== Matrix Notifier Tests ====================

func TestNewMatrixNotifier(t *testing.T) {
	notifier := NewMatrixNotifier(
		"https://matrix.example.com",
		"access-token-xyz",
		"!room:example.com",
	)

	assert.NotNil(t, notifier)
	assert.Equal(t, "https://matrix.example.com", notifier.homeserverURL)
	assert.Equal(t, "access-token-xyz", notifier.accessToken)
	assert.Equal(t, "!room:example.com", notifier.roomID)
}

func TestMatrixNotifier_getSeverityEmoji(t *testing.T) {
	notifier := NewMatrixNotifier("", "", "")

	testCases := []struct {
		severity Severity
		expected string
	}{
		{SeverityCritical, "🔴"},
		{SeverityError, "❌"},
		{SeverityWarning, "⚠️"},
		{SeverityInfo, "ℹ️"},
		{SeverityDebug, "🐛"},
		{Severity("unknown"), "🔔"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.severity), func(t *testing.T) {
			emoji := notifier.getSeverityEmoji(tc.severity)
			assert.Equal(t, tc.expected, emoji)
		})
	}
}

func TestMatrixNotifier_getSeverityColor(t *testing.T) {
	notifier := NewMatrixNotifier("", "", "")

	testCases := []struct {
		severity Severity
		expected string
	}{
		{SeverityCritical, "#FF0000"},
		{SeverityError, "#FF4444"},
		{SeverityWarning, "#FFA500"},
		{SeverityInfo, "#008000"},
		{SeverityDebug, "#808080"},
		{Severity("unknown"), "#808080"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.severity), func(t *testing.T) {
			color := notifier.getSeverityColor(tc.severity)
			assert.Equal(t, tc.expected, color)
		})
	}
}

func TestMatrixNotifier_buildMessage(t *testing.T) {
	notifier := NewMatrixNotifier("", "", "")

	t.Run("basic message", func(t *testing.T) {
		event := &Event{
			Title:     "Test Title",
			Message:   "Test Message",
			Severity:  SeverityWarning,
			Source:    "test-source",
			Timestamp: time.Now(),
		}
		msg := notifier.buildMessage(event)
		assert.Contains(t, msg, "Test Title")
		assert.Contains(t, msg, "Test Message")
		assert.Contains(t, msg, "⚠️")
		assert.Contains(t, msg, "WARNING")
	})

	t.Run("with all ids", func(t *testing.T) {
		modelID := int64(1)
		providerID := int64(2)
		clientID := "client"
		event := &Event{
			Title:      "Test",
			Message:    "Test",
			Severity:   SeverityInfo,
			Source:     "test",
			Timestamp:  time.Now(),
			ModelID:    &modelID,
			ProviderID: &providerID,
			ClientID:   &clientID,
		}
		msg := notifier.buildMessage(event)
		assert.Contains(t, msg, "Model ID: 1")
		assert.Contains(t, msg, "Provider ID: 2")
		assert.Contains(t, msg, "Client ID: client")
	})
}

func TestMatrixNotifier_buildHTMLMessage(t *testing.T) {
	notifier := NewMatrixNotifier("", "", "")

	event := &Event{
		Title:     "Test Title",
		Message:   "Test Message",
		Severity:  SeverityError,
		Source:    "test-source",
		Timestamp: time.Now(),
	}

	html := notifier.buildHTMLMessage(event)
	assert.Contains(t, html, "<strong>")
	assert.Contains(t, html, "Test Title")
	assert.Contains(t, html, "#FF4444") // Error color
	assert.Contains(t, html, "❌")
}

// ==================== WhatsApp Notifier Tests ====================

func TestNewWhatsAppNotifier(t *testing.T) {
	notifier := NewWhatsAppNotifier(
		"account-sid",
		"auth-token",
		"+1234567890",
		[]string{"+0987654321", "+1122334455"},
	)

	assert.NotNil(t, notifier)
	assert.Equal(t, "account-sid", notifier.accountSID)
	assert.Equal(t, "auth-token", notifier.authToken)
	assert.Equal(t, "+1234567890", notifier.fromNumber)
	assert.Len(t, notifier.toNumbers, 2)
}

func TestWhatsAppNotifier_getSeverityEmoji(t *testing.T) {
	notifier := NewWhatsAppNotifier("", "", "", nil)

	testCases := []struct {
		severity Severity
		expected string
	}{
		{SeverityCritical, "🔴"},
		{SeverityError, "❌"},
		{SeverityWarning, "⚠️"},
		{SeverityInfo, "ℹ️"},
		{SeverityDebug, "🐛"},
		{Severity("unknown"), "🔔"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.severity), func(t *testing.T) {
			emoji := notifier.getSeverityEmoji(tc.severity)
			assert.Equal(t, tc.expected, emoji)
		})
	}
}

func TestWhatsAppNotifier_buildMessage(t *testing.T) {
	notifier := NewWhatsAppNotifier("", "", "", nil)

	t.Run("basic message", func(t *testing.T) {
		event := &Event{
			Title:     "Test Title",
			Message:   "Test Message",
			Severity:  SeverityCritical,
			Source:    "test",
			Timestamp: time.Now(),
		}
		msg := notifier.buildMessage(event)
		assert.Contains(t, msg, "*Test Title*")
		assert.Contains(t, msg, "Test Message")
		assert.Contains(t, msg, "🔴")
		assert.Contains(t, msg, "*Severity:*")
	})

	t.Run("with model id", func(t *testing.T) {
		modelID := int64(123)
		event := &Event{
			Title:     "Test",
			Message:   "Test",
			Severity:  SeverityInfo,
			Source:    "test",
			Timestamp: time.Now(),
			ModelID:   &modelID,
		}
		msg := notifier.buildMessage(event)
		assert.Contains(t, msg, "*Model ID:* 123")
	})

	t.Run("with provider id", func(t *testing.T) {
		providerID := int64(456)
		event := &Event{
			Title:      "Test",
			Message:    "Test",
			Severity:   SeverityInfo,
			Source:     "test",
			Timestamp:  time.Now(),
			ProviderID: &providerID,
		}
		msg := notifier.buildMessage(event)
		assert.Contains(t, msg, "*Provider ID:* 456")
	})

	t.Run("with client id", func(t *testing.T) {
		clientID := "client-xyz"
		event := &Event{
			Title:     "Test",
			Message:   "Test",
			Severity:  SeverityInfo,
			Source:    "test",
			Timestamp: time.Now(),
			ClientID:  &clientID,
		}
		msg := notifier.buildMessage(event)
		assert.Contains(t, msg, "*Client ID:* client-xyz")
	})
}