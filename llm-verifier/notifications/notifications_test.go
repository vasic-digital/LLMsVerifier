package notifications

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"llm-verifier/config"
	"llm-verifier/events"
)

// TestNotificationManager_Creation tests notification manager creation
func TestNotificationManager_Creation(t *testing.T) {
	cfg := &config.Config{}
	manager := NewNotificationManager(cfg)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.subscribers)
	assert.False(t, manager.IsRunning())
	assert.Equal(t, 0, manager.GetChannelCount())
}

// TestNotificationManager_StartStop tests start/stop lifecycle
func TestNotificationManager_StartStop(t *testing.T) {
	cfg := &config.Config{}
	manager := NewNotificationManager(cfg)

	// Start
	err := manager.Start()
	require.NoError(t, err)
	assert.True(t, manager.IsRunning())

	// Double start should return error
	err = manager.Start()
	assert.Error(t, err)

	// Stop
	manager.Stop()
	assert.False(t, manager.IsRunning())

	// Double stop should not panic
	assert.NotPanics(t, func() { manager.Stop() })
}

// TestNotificationManager_AddChannels tests adding notification channels
func TestNotificationManager_AddChannels(t *testing.T) {
	cfg := &config.Config{}
	manager := NewNotificationManager(cfg)

	t.Run("Add Slack Channel", func(t *testing.T) {
		err := manager.AddSlackChannel("https://hooks.slack.com/services/xxx", "#general", "LLM Verifier")
		assert.NoError(t, err)
		assert.Equal(t, 1, manager.GetChannelCount())
	})

	t.Run("Add Email Channel", func(t *testing.T) {
		err := manager.AddEmailChannel(
			"smtp.gmail.com",
			587,
			"user@gmail.com",
			"password",
			"noreply@llmverifier.com",
			[]string{"admin@example.com"},
		)
		assert.NoError(t, err)
		assert.Equal(t, 2, manager.GetChannelCount())
	})

	t.Run("Add Telegram Channel", func(t *testing.T) {
		err := manager.AddTelegramChannel("123456:ABC-DEF", "-100123456789")
		assert.NoError(t, err)
		assert.Equal(t, 3, manager.GetChannelCount())
	})

	t.Run("Add Matrix Channel", func(t *testing.T) {
		err := manager.AddMatrixChannel(
			"https://matrix.org",
			"syt_abc123",
			"!roomid:matrix.org",
		)
		assert.NoError(t, err)
		assert.Equal(t, 4, manager.GetChannelCount())
	})

	t.Run("Add WhatsApp Channel", func(t *testing.T) {
		err := manager.AddWhatsAppChannel(
			"ACXXXXXXXXXXXXXXXXXXXXXXX",
			"auth_token_123",
			"whatsapp:+14155551234",
			[]string{"+14155555678"},
		)
		assert.NoError(t, err)
		assert.Equal(t, 5, manager.GetChannelCount())
	})
}

// TestNotificationManager_GetActiveChannels tests getting active channels
func TestNotificationManager_GetActiveChannels(t *testing.T) {
	cfg := &config.Config{}
	manager := NewNotificationManager(cfg)

	// Add channels
	manager.AddSlackChannel("https://hooks.slack.com/services/xxx", "#general", "Bot")
	manager.AddTelegramChannel("123456:ABC", "-100123456789")

	channels := manager.GetActiveChannels()
	assert.Len(t, channels, 2)
	assert.Contains(t, channels, "slack")
	assert.Contains(t, channels, "telegram")
}

// TestNotificationManager_SendNotification tests sending notifications
func TestNotificationManager_SendNotification(t *testing.T) {
	cfg := &config.Config{}
	manager := NewNotificationManager(cfg)

	// Start manager
	err := manager.Start()
	require.NoError(t, err)
	defer manager.Stop()

	tests := []struct {
		name             string
		notificationType string
		recipient        string
		message          string
	}{
		{
			name:             "Email Notification",
			notificationType: "email",
			recipient:        "user@example.com",
			message:          "Model verification completed successfully - Score: 8.5 (SC:8.5)",
		},
		{
			name:             "Slack Notification",
			notificationType: "slack",
			recipient:        "#general",
			message:          "New model verification available - GPT-4 scored 8.5 (SC:8.5)",
		},
		{
			name:             "Push Notification",
			notificationType: "push",
			recipient:        "device_token_123",
			message:          "Model verification complete! Score: 8.5 (SC:8.5)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test notification sending (will queue but not send without configured channel)
			err := manager.SendNotification(map[string]interface{}{
				"type":      tt.notificationType,
				"recipient": tt.recipient,
				"message":   tt.message,
				"timestamp": time.Now(),
			})
			// Queue should accept the notification
			assert.NoError(t, err)
		})
	}
}

// TestNotificationManager_WithEventManager tests integration with event manager
func TestNotificationManager_WithEventManager(t *testing.T) {
	cfg := &config.Config{}
	manager := NewNotificationManager(cfg)

	// Create event manager
	ctx := context.Background()
	em := events.NewEventManager(ctx, 100, 2)
	defer em.Shutdown()

	// Set event manager
	manager.SetEventManager(em)

	// Add channels
	err := manager.AddSlackChannel("https://hooks.slack.com/services/xxx", "#general", "Bot")
	require.NoError(t, err)

	// Verify subscriber was added to event manager
	assert.Equal(t, 1, em.GetSubscriberCount())

	// Start manager
	err = manager.Start()
	require.NoError(t, err)
	defer manager.Stop()
}

// TestNotificationManager_ScoreFormat tests score suffix format in notifications
func TestNotificationManager_ScoreFormat(t *testing.T) {
	cfg := &config.Config{}
	manager := NewNotificationManager(cfg)

	err := manager.Start()
	require.NoError(t, err)
	defer manager.Stop()

	// Test notification with score suffix
	err = manager.SendNotification(map[string]interface{}{
		"type":      "email",
		"recipient": "user@example.com",
		"message":   "Model verification completed - Score: 8.5 (SC:8.5)",
		"timestamp": time.Now(),
	})

	assert.NoError(t, err)
}

// TestNotificationManager_RateLimiting tests notification rate handling
func TestNotificationManager_RateLimiting(t *testing.T) {
	cfg := &config.Config{}
	manager := NewNotificationManager(cfg)

	err := manager.Start()
	require.NoError(t, err)
	defer manager.Stop()

	// Send multiple notifications quickly - queue should handle them
	for i := 0; i < 10; i++ {
		err := manager.SendNotification(map[string]interface{}{
			"type":      "email",
			"recipient": "test@example.com",
			"message":   "Test notification - Score: 8.5 (SC:8.5)",
			"timestamp": time.Now(),
		})
		assert.NoError(t, err)
	}
}

// TestNotification struct tests
func TestNotification_Struct(t *testing.T) {
	notification := &Notification{
		Type:      NotificationTypeSlack,
		Channel:   "#alerts",
		Priority:  "high",
		Title:     "Verification Complete",
		Message:   "Model GPT-4 verified with score 8.5",
		Recipient: "#alerts",
		CreatedAt: time.Now(),
	}

	assert.Equal(t, NotificationTypeSlack, notification.Type)
	assert.Equal(t, "high", notification.Priority)
	assert.False(t, notification.Sent)
	assert.Nil(t, notification.SentAt)
}

// TestNotificationTypes tests notification type constants
func TestNotificationTypes(t *testing.T) {
	assert.Equal(t, NotificationType("email"), NotificationTypeEmail)
	assert.Equal(t, NotificationType("slack"), NotificationTypeSlack)
	assert.Equal(t, NotificationType("telegram"), NotificationTypeTelegram)
	assert.Equal(t, NotificationType("matrix"), NotificationTypeMatrix)
	assert.Equal(t, NotificationType("whatsapp"), NotificationTypeWhatsApp)
}

// TestNotificationManager_PriorityToSeverity tests priority conversion
func TestNotificationManager_PriorityToSeverity(t *testing.T) {
	cfg := &config.Config{}
	manager := NewNotificationManager(cfg)

	tests := []struct {
		priority string
		expected events.Severity
	}{
		{"low", events.SeverityDebug},
		{"normal", events.SeverityInfo},
		{"high", events.SeverityWarning},
		{"critical", events.SeverityCritical},
		{"unknown", events.SeverityInfo},
	}

	for _, tt := range tests {
		t.Run(tt.priority, func(t *testing.T) {
			severity := manager.priorityToSeverity(tt.priority)
			assert.Equal(t, tt.expected, severity)
		})
	}
}
