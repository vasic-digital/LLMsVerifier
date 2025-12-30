package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== NotificationManager Tests ====================

func TestNewNotificationManager(t *testing.T) {
	nm := NewNotificationManager()
	require.NotNil(t, nm)
	assert.Empty(t, nm.notifications)
	assert.Equal(t, 3, nm.maxVisible)
	assert.True(t, nm.autoDismiss)
}

func TestNotificationManager_AddNotification(t *testing.T) {
	nm := NewNotificationManager()
	nm.autoDismiss = false // Disable auto-dismiss for testing

	nm.AddNotification(NotificationInfo, "Test message", 5*time.Second)

	notifications := nm.GetNotifications()
	require.Len(t, notifications, 1)
	assert.Equal(t, NotificationInfo, notifications[0].Type)
	assert.Equal(t, "Test message", notifications[0].Message)
	assert.Equal(t, 5*time.Second, notifications[0].Duration)
	assert.NotEmpty(t, notifications[0].ID)
}

func TestNotificationManager_AddNotification_MaxVisible(t *testing.T) {
	nm := NewNotificationManager()
	nm.autoDismiss = false
	nm.maxVisible = 3

	// Add more than max visible
	for i := 0; i < 5; i++ {
		nm.AddNotification(NotificationInfo, "Message", 0)
	}

	notifications := nm.GetNotifications()
	assert.Len(t, notifications, 3, "Should only keep maxVisible notifications")
}

func TestNotificationManager_RemoveNotification(t *testing.T) {
	nm := NewNotificationManager()
	nm.autoDismiss = false

	nm.AddNotification(NotificationInfo, "First", 0)
	nm.AddNotification(NotificationSuccess, "Second", 0)

	notifications := nm.GetNotifications()
	require.Len(t, notifications, 2)

	// Remove first notification
	firstID := notifications[0].ID
	nm.RemoveNotification(firstID)

	notifications = nm.GetNotifications()
	require.Len(t, notifications, 1)
	assert.Equal(t, "Second", notifications[0].Message)
}

func TestNotificationManager_RemoveNotification_NotFound(t *testing.T) {
	nm := NewNotificationManager()
	nm.autoDismiss = false

	nm.AddNotification(NotificationInfo, "Test", 0)

	// Try to remove non-existent notification
	nm.RemoveNotification("non-existent-id")

	// Should still have the original notification
	notifications := nm.GetNotifications()
	assert.Len(t, notifications, 1)
}

func TestNotificationManager_ClearNotifications(t *testing.T) {
	nm := NewNotificationManager()
	nm.autoDismiss = false

	nm.AddNotification(NotificationInfo, "First", 0)
	nm.AddNotification(NotificationSuccess, "Second", 0)
	nm.AddNotification(NotificationWarning, "Third", 0)

	require.Len(t, nm.GetNotifications(), 3)

	nm.ClearNotifications()

	assert.Empty(t, nm.GetNotifications())
}

func TestNotificationManager_GetNotifications(t *testing.T) {
	nm := NewNotificationManager()
	nm.autoDismiss = false

	// Initially empty
	assert.Empty(t, nm.GetNotifications())

	// Add some notifications
	nm.AddNotification(NotificationInfo, "Test", 0)
	nm.AddNotification(NotificationError, "Error", 0)

	notifications := nm.GetNotifications()
	assert.Len(t, notifications, 2)
}

func TestNotificationManager_Render_Empty(t *testing.T) {
	nm := NewNotificationManager()

	result := nm.Render()
	assert.Empty(t, result)
}

func TestNotificationManager_Render_WithNotifications(t *testing.T) {
	nm := NewNotificationManager()
	nm.autoDismiss = false

	nm.AddNotification(NotificationInfo, "Info message", 0)
	nm.AddNotification(NotificationSuccess, "Success message", 0)

	result := nm.Render()
	assert.NotEmpty(t, result)
	assert.Contains(t, result, "Info message")
	assert.Contains(t, result, "Success message")
}

func TestNotificationManager_RenderNotification_Types(t *testing.T) {
	nm := NewNotificationManager()
	nm.autoDismiss = false

	tests := []struct {
		ntype    NotificationType
		message  string
	}{
		{NotificationInfo, "Info"},
		{NotificationSuccess, "Success"},
		{NotificationWarning, "Warning"},
		{NotificationError, "Error"},
	}

	for _, tt := range tests {
		t.Run(string(tt.ntype), func(t *testing.T) {
			nm.ClearNotifications()
			nm.AddNotification(tt.ntype, tt.message, 0)

			result := nm.Render()
			assert.NotEmpty(t, result)
			assert.Contains(t, result, tt.message)
		})
	}
}

// ==================== Notification Type Tests ====================

func TestNotificationType_Constants(t *testing.T) {
	assert.Equal(t, NotificationType("info"), NotificationInfo)
	assert.Equal(t, NotificationType("success"), NotificationSuccess)
	assert.Equal(t, NotificationType("warning"), NotificationWarning)
	assert.Equal(t, NotificationType("error"), NotificationError)
}

// ==================== Notification Struct Tests ====================

func TestNotification_Fields(t *testing.T) {
	now := time.Now()
	notif := Notification{
		ID:        "test-id",
		Type:      NotificationSuccess,
		Message:   "Test message",
		Timestamp: now,
		Duration:  10 * time.Second,
	}

	assert.Equal(t, "test-id", notif.ID)
	assert.Equal(t, NotificationSuccess, notif.Type)
	assert.Equal(t, "Test message", notif.Message)
	assert.Equal(t, now, notif.Timestamp)
	assert.Equal(t, 10*time.Second, notif.Duration)
}

// ==================== Notification Command Tests ====================

func TestAddNotificationCmd(t *testing.T) {
	cmd := AddNotificationCmd(NotificationInfo, "Test", 5*time.Second)
	require.NotNil(t, cmd)

	msg := cmd()
	addMsg, ok := msg.(AddNotificationMsg)
	require.True(t, ok)
	assert.Equal(t, NotificationInfo, addMsg.Type)
	assert.Equal(t, "Test", addMsg.Message)
	assert.Equal(t, 5*time.Second, addMsg.Duration)
}

func TestRemoveNotificationCmd(t *testing.T) {
	cmd := RemoveNotificationCmd("test-id")
	require.NotNil(t, cmd)

	msg := cmd()
	removeMsg, ok := msg.(RemoveNotificationMsg)
	require.True(t, ok)
	assert.Equal(t, "test-id", removeMsg.ID)
}

func TestClearNotificationsCmd(t *testing.T) {
	cmd := ClearNotificationsCmd()
	require.NotNil(t, cmd)

	msg := cmd()
	_, ok := msg.(ClearNotificationsMsg)
	require.True(t, ok)
}

func TestNotificationTickCmd(t *testing.T) {
	cmd := NotificationTickCmd()
	require.NotNil(t, cmd)
	// The tick command returns a tea.Cmd that will produce NotificationTickMsg after 1 second
	// We can't easily test the tick behavior without running the tea program
}

// ==================== Message Type Tests ====================

func TestAddNotificationMsg(t *testing.T) {
	msg := AddNotificationMsg{
		Type:     NotificationWarning,
		Message:  "Warning message",
		Duration: 10 * time.Second,
	}

	assert.Equal(t, NotificationWarning, msg.Type)
	assert.Equal(t, "Warning message", msg.Message)
	assert.Equal(t, 10*time.Second, msg.Duration)
}

func TestRemoveNotificationMsg(t *testing.T) {
	msg := RemoveNotificationMsg{ID: "notif-123"}
	assert.Equal(t, "notif-123", msg.ID)
}

func TestClearNotificationsMsg(t *testing.T) {
	msg := ClearNotificationsMsg{}
	assert.NotNil(t, msg)
}
