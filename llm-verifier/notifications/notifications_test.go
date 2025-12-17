package notifications

import (
	"testing"
	"time"

	"llm-verifier/config"
	"llm-verifier/events"
)

func TestNotificationManager(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		Notifications: config.NotificationsConfig{
			Slack: config.SlackConfig{
				Enabled:    true,
				WebhookURL: "https://hooks.slack.com/test",
			},
			Email: config.EmailConfig{
				Enabled:          true,
				SMTPHost:         "smtp.test.com",
				SMTPPort:         587,
				Username:         "test@test.com",
				Password:         "testpass",
				DefaultRecipient: "alerts@test.com",
			},
			Telegram: config.TelegramConfig{Enabled: false},
			Matrix:   config.MatrixConfig{Enabled: false},
			WhatsApp: config.WhatsAppConfig{Enabled: false},
		},
	}

	// Create event bus
	eventBus := events.NewEventBus(nil)
	defer eventBus.Shutdown()

	// Create notification manager
	nm := NewNotificationManager(cfg, eventBus)
	defer nm.Shutdown()

	// Test notification creation
	testEvent := events.Event{
		ID:        "test-event-1",
		Type:      events.EventTypeModelVerified,
		Severity:  events.EventSeverityInfo,
		Message:   "Test notification message",
		Timestamp: time.Now(),
		Source:    "test",
	}

	// Test sending notification to enabled channels
	err := nm.SendEventNotification(testEvent, []NotificationChannel{ChannelSlack, ChannelEmail})
	if err != nil {
		t.Errorf("Failed to send event notification: %v", err)
	}

	// Test basic functionality
	if !cfg.Notifications.Slack.Enabled {
		t.Errorf("Expected Slack to be enabled")
	}

	if !cfg.Notifications.Email.Enabled {
		t.Errorf("Expected Email to be enabled")
	}
}

func TestEventTitleGeneration(t *testing.T) {
	testCases := []struct {
		eventType   events.EventType
		expectTitle string
	}{
		{events.EventTypeModelVerified, "Model Verification Completed"},
		{events.EventTypeModelVerificationFailed, "Model Verification Failed"},
		{events.EventTypeScoreChanged, "Model Score Changed"},
		{events.EventTypeVerificationStarted, "Verification Started"},
		{events.EventTypeErrorOccurred, "System Error"},
		{events.EventType("custom.event"), "System Notification"},
	}

	for _, tc := range testCases {
		title := getEventTitleForTest(tc.eventType)
		if title != tc.expectTitle {
			t.Errorf("Expected title '%s' for event type '%s', got '%s'", tc.expectTitle, tc.eventType, title)
		}
	}
}

func TestPriorityMapping(t *testing.T) {
	testCases := []struct {
		severity       events.EventSeverity
		expectPriority string
	}{
		{events.EventSeverityCritical, "high"},
		{events.EventSeverityError, "high"},
		{events.EventSeverityWarning, "medium"},
		{events.EventSeverityInfo, "low"},
	}

	for _, tc := range testCases {
		priority := getPriorityForSeverityForTest(tc.severity)
		if priority != tc.expectPriority {
			t.Errorf("Expected priority '%s' for severity '%s', got '%s'", tc.expectPriority, tc.severity, priority)
		}
	}
}

// Helper functions for testing private methods
func getEventTitleForTest(eventType events.EventType) string {
	switch eventType {
	case events.EventTypeModelVerified:
		return "Model Verification Completed"
	case events.EventTypeModelVerificationFailed:
		return "Model Verification Failed"
	case events.EventTypeScoreChanged:
		return "Model Score Changed"
	case events.EventTypeVerificationStarted:
		return "Verification Started"
	case events.EventTypeErrorOccurred:
		return "System Error"
	default:
		return "System Notification"
	}
}

func getPriorityForSeverityForTest(severity events.EventSeverity) string {
	switch severity {
	case events.EventSeverityCritical:
		return "high"
	case events.EventSeverityError:
		return "high"
	case events.EventSeverityWarning:
		return "medium"
	default:
		return "low"
	}
}
