// Package notifications handles notification management
// Integrates with the events system to send notifications through various channels
package notifications

import (
	"context"
	"fmt"
	"sync"
	"time"

	"llm-verifier/config"
	"llm-verifier/events"
)

// NotificationType represents the type of notification channel
type NotificationType string

const (
	NotificationTypeEmail    NotificationType = "email"
	NotificationTypeSlack    NotificationType = "slack"
	NotificationTypeTelegram NotificationType = "telegram"
	NotificationTypeMatrix   NotificationType = "matrix"
	NotificationTypeWhatsApp NotificationType = "whatsapp"
)

// NotificationConfig holds configuration for a notification channel
type NotificationConfig struct {
	Type        NotificationType       `json:"type"`
	Enabled     bool                   `json:"enabled"`
	MinSeverity events.Severity        `json:"min_severity"`
	EventTypes  []events.EventType     `json:"event_types"`
	Settings    map[string]interface{} `json:"settings"`
}

// NotificationManager manages notification channels and integrates with the event system
type NotificationManager struct {
	config         *config.Config
	eventManager   *events.EventManager
	subscribers    map[string]*events.NotificationSubscriber
	subscribersMux sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	running        bool
	notifyQueue    chan *Notification
	retryQueue     chan *Notification
	maxRetries     int
	retryDelay     time.Duration
}

// Notification represents a notification to be sent
type Notification struct {
	ID          string                 `json:"id"`
	Type        NotificationType       `json:"type"`
	Channel     string                 `json:"channel"`
	Priority    string                 `json:"priority"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Data        map[string]interface{} `json:"data,omitempty"`
	Recipient   string                 `json:"recipient"`
	Sent        bool                   `json:"sent"`
	Error       string                 `json:"error,omitempty"`
	RetryCount  int                    `json:"retry_count"`
	CreatedAt   time.Time              `json:"created_at"`
	SentAt      *time.Time             `json:"sent_at,omitempty"`
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager(cfg *config.Config) *NotificationManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &NotificationManager{
		config:      cfg,
		subscribers: make(map[string]*events.NotificationSubscriber),
		ctx:         ctx,
		cancel:      cancel,
		notifyQueue: make(chan *Notification, 1000),
		retryQueue:  make(chan *Notification, 100),
		maxRetries:  3,
		retryDelay:  5 * time.Second,
	}
}


// SetEventManager sets the event manager for receiving events
func (nm *NotificationManager) SetEventManager(em *events.EventManager) {
	nm.eventManager = em
}

// Start starts the notification manager
func (nm *NotificationManager) Start() error {
	nm.subscribersMux.Lock()
	if nm.running {
		nm.subscribersMux.Unlock()
		return fmt.Errorf("notification manager already running")
	}
	nm.running = true
	nm.subscribersMux.Unlock()

	// Start notification processor
	go nm.processNotifications()

	// Start retry processor
	go nm.processRetries()

	// Initialize configured notification channels
	if nm.config != nil {
		nm.initializeChannels()
	}

	return nil
}

// Stop stops the notification manager
func (nm *NotificationManager) Stop() {
	nm.subscribersMux.Lock()
	if !nm.running {
		nm.subscribersMux.Unlock()
		return
	}
	nm.running = false
	nm.subscribersMux.Unlock()

	// Cancel context first to signal goroutines to stop
	nm.cancel()

	// Give goroutines time to exit before closing channels
	time.Sleep(10 * time.Millisecond)

	// Unsubscribe all notification subscribers from event manager
	if nm.eventManager != nil {
		nm.subscribersMux.RLock()
		for id := range nm.subscribers {
			nm.eventManager.Unsubscribe(id)
		}
		nm.subscribersMux.RUnlock()
	}
}

// initializeChannels initializes notification channels from config
func (nm *NotificationManager) initializeChannels() {
	if nm.config.Notifications.Slack.Enabled {
		nm.AddSlackChannel(
			nm.config.Notifications.Slack.WebhookURL,
			"#general", // Default channel
			"LLM Verifier",
		)
	}

	if nm.config.Notifications.Email.Enabled {
		nm.AddEmailChannel(
			nm.config.Notifications.Email.SMTPHost,
			nm.config.Notifications.Email.SMTPPort,
			nm.config.Notifications.Email.Username,
			nm.config.Notifications.Email.Password,
			nm.config.Notifications.Email.Username, // Use username as from address
			[]string{nm.config.Notifications.Email.DefaultRecipient},
		)
	}

	if nm.config.Notifications.Telegram.Enabled {
		nm.AddTelegramChannel(
			nm.config.Notifications.Telegram.BotToken,
			nm.config.Notifications.Telegram.ChatID,
		)
	}

	if nm.config.Notifications.Matrix.Enabled {
		nm.AddMatrixChannel(
			nm.config.Notifications.Matrix.HomeserverURL,
			nm.config.Notifications.Matrix.AccessToken,
			nm.config.Notifications.Matrix.RoomID,
		)
	}
}

// AddSlackChannel adds a Slack notification channel
func (nm *NotificationManager) AddSlackChannel(webhookURL, channel, username string) error {
	config := map[string]interface{}{
		"webhook_url": webhookURL,
		"channel":     channel,
		"username":    username,
	}

	eventTypes := []events.EventType{
		events.EventVerificationCompleted,
		events.EventVerificationFailed,
		events.EventIssueDetected,
		events.EventSecurityAlert,
		events.EventSystemHealthChanged,
	}

	subscriber := events.NewNotificationSubscriber("slack", eventTypes, events.SeverityInfo, config)

	nm.subscribersMux.Lock()
	nm.subscribers[subscriber.GetID()] = subscriber
	nm.subscribersMux.Unlock()

	if nm.eventManager != nil {
		return nm.eventManager.Subscribe(subscriber)
	}
	return nil
}

// AddEmailChannel adds an Email notification channel
func (nm *NotificationManager) AddEmailChannel(smtpServer string, smtpPort int, username, password, fromAddress string, toAddresses []string) error {
	config := map[string]interface{}{
		"smtp_server":  smtpServer,
		"smtp_port":    float64(smtpPort),
		"username":     username,
		"password":     password,
		"from_address": fromAddress,
		"to_addresses": toAddresses,
	}

	eventTypes := []events.EventType{
		events.EventVerificationCompleted,
		events.EventVerificationFailed,
		events.EventIssueDetected,
		events.EventSecurityAlert,
		events.EventBackupCompleted,
	}

	subscriber := events.NewNotificationSubscriber("email", eventTypes, events.SeverityWarning, config)

	nm.subscribersMux.Lock()
	nm.subscribers[subscriber.GetID()] = subscriber
	nm.subscribersMux.Unlock()

	if nm.eventManager != nil {
		return nm.eventManager.Subscribe(subscriber)
	}
	return nil
}

// AddTelegramChannel adds a Telegram notification channel
func (nm *NotificationManager) AddTelegramChannel(botToken, chatID string) error {
	config := map[string]interface{}{
		"bot_token": botToken,
		"chat_id":   chatID,
	}

	eventTypes := []events.EventType{
		events.EventVerificationCompleted,
		events.EventVerificationFailed,
		events.EventIssueDetected,
		events.EventSecurityAlert,
	}

	subscriber := events.NewNotificationSubscriber("telegram", eventTypes, events.SeverityInfo, config)

	nm.subscribersMux.Lock()
	nm.subscribers[subscriber.GetID()] = subscriber
	nm.subscribersMux.Unlock()

	if nm.eventManager != nil {
		return nm.eventManager.Subscribe(subscriber)
	}
	return nil
}

// AddMatrixChannel adds a Matrix notification channel
func (nm *NotificationManager) AddMatrixChannel(homeserverURL, accessToken, roomID string) error {
	config := map[string]interface{}{
		"homeserver_url": homeserverURL,
		"access_token":   accessToken,
		"room_id":        roomID,
	}

	eventTypes := []events.EventType{
		events.EventVerificationCompleted,
		events.EventVerificationFailed,
		events.EventIssueDetected,
		events.EventSecurityAlert,
	}

	subscriber := events.NewNotificationSubscriber("matrix", eventTypes, events.SeverityInfo, config)

	nm.subscribersMux.Lock()
	nm.subscribers[subscriber.GetID()] = subscriber
	nm.subscribersMux.Unlock()

	if nm.eventManager != nil {
		return nm.eventManager.Subscribe(subscriber)
	}
	return nil
}

// AddWhatsAppChannel adds a WhatsApp notification channel
func (nm *NotificationManager) AddWhatsAppChannel(accountSID, authToken, fromNumber string, toNumbers []string) error {
	config := map[string]interface{}{
		"account_sid": accountSID,
		"auth_token":  authToken,
		"from_number": fromNumber,
		"to_numbers":  toNumbers,
	}

	eventTypes := []events.EventType{
		events.EventSecurityAlert,
		events.EventIssueDetected,
	}

	subscriber := events.NewNotificationSubscriber("whatsapp", eventTypes, events.SeverityError, config)

	nm.subscribersMux.Lock()
	nm.subscribers[subscriber.GetID()] = subscriber
	nm.subscribersMux.Unlock()

	if nm.eventManager != nil {
		return nm.eventManager.Subscribe(subscriber)
	}
	return nil
}

// SendNotification sends a notification through the appropriate channel
func (nm *NotificationManager) SendNotification(notification interface{}) error {
	switch n := notification.(type) {
	case *Notification:
		return nm.queueNotification(n)
	case map[string]interface{}:
		return nm.queueFromMap(n)
	default:
		return fmt.Errorf("unsupported notification type: %T", notification)
	}
}

// queueNotification adds a notification to the processing queue
func (nm *NotificationManager) queueNotification(notification *Notification) error {
	if notification.ID == "" {
		notification.ID = fmt.Sprintf("notify_%d", time.Now().UnixNano())
	}
	if notification.CreatedAt.IsZero() {
		notification.CreatedAt = time.Now()
	}

	select {
	case nm.notifyQueue <- notification:
		return nil
	case <-nm.ctx.Done():
		return fmt.Errorf("notification manager is shutting down")
	default:
		return fmt.Errorf("notification queue is full")
	}
}

// queueFromMap creates a notification from a map and queues it
func (nm *NotificationManager) queueFromMap(data map[string]interface{}) error {
	notification := &Notification{
		ID:        fmt.Sprintf("notify_%d", time.Now().UnixNano()),
		CreatedAt: time.Now(),
	}

	if typeVal, ok := data["type"].(string); ok {
		notification.Type = NotificationType(typeVal)
	}
	if channel, ok := data["channel"].(string); ok {
		notification.Channel = channel
	}
	if priority, ok := data["priority"].(string); ok {
		notification.Priority = priority
	} else {
		notification.Priority = "normal"
	}
	if title, ok := data["title"].(string); ok {
		notification.Title = title
	}
	if message, ok := data["message"].(string); ok {
		notification.Message = message
	}
	if recipient, ok := data["recipient"].(string); ok {
		notification.Recipient = recipient
	}
	if dataMap, ok := data["data"].(map[string]interface{}); ok {
		notification.Data = dataMap
	}

	return nm.queueNotification(notification)
}

// processNotifications processes queued notifications
func (nm *NotificationManager) processNotifications() {
	for {
		select {
		case notification, ok := <-nm.notifyQueue:
			if !ok {
				return
			}
			if err := nm.sendNotificationDirect(notification); err != nil {
				notification.Error = err.Error()
				notification.RetryCount++
				if notification.RetryCount <= nm.maxRetries {
					select {
					case nm.retryQueue <- notification:
					case <-nm.ctx.Done():
						return
					default:
						// Retry queue full, skip
					}
				}
			} else {
				notification.Sent = true
				now := time.Now()
				notification.SentAt = &now
				// Store in database if available
				nm.storeNotification(notification)
			}
		case <-nm.ctx.Done():
			return
		}
	}
}

// processRetries processes retry queue with delay
func (nm *NotificationManager) processRetries() {
	for {
		select {
		case notification, ok := <-nm.retryQueue:
			if !ok {
				return
			}
			time.Sleep(nm.retryDelay * time.Duration(notification.RetryCount))
			select {
			case nm.notifyQueue <- notification:
			case <-nm.ctx.Done():
				return
			default:
				// Queue full, notification dropped
			}
		case <-nm.ctx.Done():
			return
		}
	}
}

// sendNotificationDirect sends a notification directly
func (nm *NotificationManager) sendNotificationDirect(notification *Notification) error {
	// Create an event and send through event manager
	event := &events.Event{
		ID:        notification.ID,
		Type:      events.EventType("notification"),
		Severity:  nm.priorityToSeverity(notification.Priority),
		Title:     notification.Title,
		Message:   notification.Message,
		Details:   notification.Data,
		Timestamp: notification.CreatedAt,
		Source:    "notification_manager",
	}

	// Find matching subscriber by type
	nm.subscribersMux.RLock()
	defer nm.subscribersMux.RUnlock()

	for _, subscriber := range nm.subscribers {
		if subscriber.ServiceType == string(notification.Type) {
			return subscriber.ReceiveEvent(event)
		}
	}

	return fmt.Errorf("no subscriber found for notification type: %s", notification.Type)
}

// priorityToSeverity converts notification priority to event severity
func (nm *NotificationManager) priorityToSeverity(priority string) events.Severity {
	switch priority {
	case "low":
		return events.SeverityDebug
	case "normal":
		return events.SeverityInfo
	case "high":
		return events.SeverityWarning
	case "critical":
		return events.SeverityCritical
	default:
		return events.SeverityInfo
	}
}

// storeNotification stores notification in database (placeholder for future implementation)
func (nm *NotificationManager) storeNotification(notification *Notification) {
	// Database storage is optional - notifications are already sent
	// This is a placeholder for future persistence requirements
	_ = notification
}

// GetActiveChannels returns the list of active notification channels
func (nm *NotificationManager) GetActiveChannels() []string {
	nm.subscribersMux.RLock()
	defer nm.subscribersMux.RUnlock()

	channels := make([]string, 0, len(nm.subscribers))
	for _, subscriber := range nm.subscribers {
		if subscriber.IsActive() {
			channels = append(channels, subscriber.ServiceType)
		}
	}
	return channels
}

// GetChannelCount returns the number of configured channels
func (nm *NotificationManager) GetChannelCount() int {
	nm.subscribersMux.RLock()
	defer nm.subscribersMux.RUnlock()
	return len(nm.subscribers)
}

// IsRunning returns whether the notification manager is running
func (nm *NotificationManager) IsRunning() bool {
	nm.subscribersMux.RLock()
	defer nm.subscribersMux.RUnlock()
	return nm.running
}
