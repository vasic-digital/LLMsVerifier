package tui

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"digital.vasic.llmsverifier/client"
)

// randomIDSuffix returns a short, collision-resistant suffix for IDs.
// 8 bytes from crypto/rand (base64url ~11 chars); on RNG failure falls back to
// a doubled-nanosecond value so the suffix is non-empty and varies. §11.4.50:
// two TUI notifications added within the same nanosecond must not share an ID.
func randomIDSuffix() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano()*2654435761)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// newNotificationID builds a unique TUI notification ID. Kept as a named helper
// so its uniqueness is directly testable (§11.4.115).
func newNotificationID() string {
	return fmt.Sprintf("notif-%d-%s", time.Now().UnixNano(), randomIDSuffix())
}

type NotificationType string

const (
	NotificationInfo    NotificationType = "info"
	NotificationSuccess NotificationType = "success"
	NotificationWarning NotificationType = "warning"
	NotificationError   NotificationType = "error"
)

type Notification struct {
	ID        string
	Type      NotificationType
	Message   string
	Timestamp time.Time
	Duration  time.Duration
}

type NotificationManager struct {
	notifications []Notification
	maxVisible    int
	autoDismiss   bool
}

func NewNotificationManager() *NotificationManager {
	return &NotificationManager{
		notifications: []Notification{},
		maxVisible:    3,
		autoDismiss:   true,
	}
}

func (nm *NotificationManager) AddNotification(ntype NotificationType, message string, duration time.Duration) {
	id := newNotificationID()
	notification := Notification{
		ID:        id,
		Type:      ntype,
		Message:   message,
		Timestamp: time.Now(),
		Duration:  duration,
	}

	nm.notifications = append(nm.notifications, notification)

	// Remove oldest notifications if we exceed max visible
	if len(nm.notifications) > nm.maxVisible {
		nm.notifications = nm.notifications[len(nm.notifications)-nm.maxVisible:]
	}

	// Auto-dismiss after duration
	if nm.autoDismiss && duration > 0 {
		go func(id string) {
			time.Sleep(duration)
			nm.RemoveNotification(id)
		}(id)
	}
}

func (nm *NotificationManager) RemoveNotification(id string) {
	for i, notif := range nm.notifications {
		if notif.ID == id {
			nm.notifications = append(nm.notifications[:i], nm.notifications[i+1:]...)
			break
		}
	}
}

func (nm *NotificationManager) ClearNotifications() {
	nm.notifications = []Notification{}
}

func (nm *NotificationManager) GetNotifications() []Notification {
	return nm.notifications
}

func (nm *NotificationManager) Render() string {
	if len(nm.notifications) == 0 {
		return ""
	}

	var notificationViews []string
	visibleNotifications := nm.notifications
	if len(visibleNotifications) > nm.maxVisible {
		visibleNotifications = visibleNotifications[len(visibleNotifications)-nm.maxVisible:]
	}

	for _, notif := range visibleNotifications {
		notificationViews = append(notificationViews, nm.renderNotification(notif))
	}

	return lipgloss.JoinVertical(
		lipgloss.Top,
		notificationViews...,
	)
}

func (nm *NotificationManager) renderNotification(notif Notification) string {
	var (
		borderColor string
		bgColor     string
		icon        string
	)

	switch notif.Type {
	case NotificationInfo:
		borderColor = "39"
		bgColor = "234"
		icon = "ℹ️"
	case NotificationSuccess:
		borderColor = "46"
		bgColor = "234"
		icon = "✅"
	case NotificationWarning:
		borderColor = "214"
		bgColor = "234"
		icon = "⚠️"
	case NotificationError:
		borderColor = "196"
		bgColor = "234"
		icon = "❌"
	}

	style := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Background(lipgloss.Color(bgColor)).
		Padding(0, 1).
		Margin(0, 0, 1, 0)

	content := lipgloss.JoinHorizontal(
		lipgloss.Left,
		lipgloss.NewStyle().Width(3).Render(icon),
		lipgloss.NewStyle().Width(2).Render(""),
		lipgloss.NewStyle().Render(notif.Message),
		lipgloss.NewStyle().Width(5).Render(""),
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Render(notif.Timestamp.Format("15:04:05")),
	)

	return style.Render(content)
}

// Notification commands for Bubble Tea

type AddNotificationMsg struct {
	Type     NotificationType
	Message  string
	Duration time.Duration
}

type RemoveNotificationMsg struct {
	ID string
}

type ClearNotificationsMsg struct{}

func AddNotificationCmd(ntype NotificationType, message string, duration time.Duration) tea.Cmd {
	return func() tea.Msg {
		return AddNotificationMsg{
			Type:     ntype,
			Message:  message,
			Duration: duration,
		}
	}
}

func RemoveNotificationCmd(id string) tea.Cmd {
	return func() tea.Msg {
		return RemoveNotificationMsg{ID: id}
	}
}

func ClearNotificationsCmd() tea.Cmd {
	return func() tea.Msg {
		return ClearNotificationsMsg{}
	}
}

// Notification tick for auto-dismissal

type NotificationTickMsg struct{}

func NotificationTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return NotificationTickMsg{}
	})
}

// Enhanced App with notifications

type AppWithNotifications struct {
	*App
	notificationManager *NotificationManager
	notificationArea    string
}

func NewAppWithNotifications(client *client.Client) *AppWithNotifications {
	return &AppWithNotifications{
		App:                 NewApp(client),
		notificationManager: NewNotificationManager(),
	}
}

func (a *AppWithNotifications) Init() tea.Cmd {
	return tea.Batch(
		a.App.Init(),
		NotificationTickCmd(),
	)
}

func (a *AppWithNotifications) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case AddNotificationMsg:
		a.notificationManager.AddNotification(msg.Type, msg.Message, msg.Duration)
		return a, nil
	case RemoveNotificationMsg:
		a.notificationManager.RemoveNotification(msg.ID)
		return a, nil
	case ClearNotificationsMsg:
		a.notificationManager.ClearNotifications()
		return a, nil
	case NotificationTickMsg:
		return a, NotificationTickCmd()
	}

	// Handle regular app messages
	model, cmd := a.App.Update(msg)
	if app, ok := model.(*App); ok {
		a.App = app
	}
	return a, cmd
}

func (a *AppWithNotifications) View() string {
	// Render notifications at the top
	notificationsView := a.notificationManager.Render()

	// Render main app content
	appView := a.App.View()

	if notificationsView == "" {
		return appView
	}

	// Combine notifications and app view
	return lipgloss.JoinVertical(
		lipgloss.Top,
		notificationsView,
		appView,
	)
}

// Helper functions for common notification patterns

func (a *AppWithNotifications) NotifyInfo(message string) {
	a.notificationManager.AddNotification(NotificationInfo, message, 5*time.Second)
}

func (a *AppWithNotifications) NotifySuccess(message string) {
	a.notificationManager.AddNotification(NotificationSuccess, message, 3*time.Second)
}

func (a *AppWithNotifications) NotifyWarning(message string) {
	a.notificationManager.AddNotification(NotificationWarning, message, 10*time.Second)
}

func (a *AppWithNotifications) NotifyError(message string) {
	a.notificationManager.AddNotification(NotificationError, message, 15*time.Second)
}

func (a *AppWithNotifications) NotifyModelVerified(modelName string) {
	a.NotifySuccess(trTUIData("tui.notification.model_verified", map[string]any{"model": modelName}))
}

func (a *AppWithNotifications) NotifyVerificationFailed(modelName string, err error) {
	a.NotifyError(trTUIData("tui.notification.verification_failed", map[string]any{"model": modelName, "error": err}))
}

func (a *AppWithNotifications) NotifyDataRefreshed() {
	a.NotifyInfo(trTUI("tui.notification.data_refreshed"))
}

func (a *AppWithNotifications) NotifyConnectionError() {
	a.NotifyError(trTUI("tui.notification.connection_error"))
}
