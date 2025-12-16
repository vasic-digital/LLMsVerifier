package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/smtp"
	"time"

	"llm-verifier/config"
	"llm-verifier/events"
)

// NotificationChannel represents the type of notification channel
type NotificationChannel string

const (
	ChannelSlack    NotificationChannel = "slack"
	ChannelEmail    NotificationChannel = "email"
	ChannelTelegram NotificationChannel = "telegram"
	ChannelMatrix   NotificationChannel = "matrix"
	ChannelWhatsApp NotificationChannel = "whatsapp"
)

// Notification represents a notification to be sent
type Notification struct {
	ID        string
	Title     string
	Message   string
	Channel   NotificationChannel
	Recipient string
	Event     events.Event
	Priority  string
	Timestamp time.Time
	Template  string
	Data      map[string]interface{}
}

// NotificationManager manages all notification channels
type NotificationManager struct {
	config   *config.Config
	channels map[NotificationChannel]NotificationChannel
	queue    chan Notification
	workers  int
	stopCh   chan struct{}
	eventBus *events.EventBus
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager(cfg *config.Config, eventBus *events.EventBus) *NotificationManager {
	nm := &NotificationManager{
		config:   cfg,
		channels: make(map[NotificationChannel]NotificationChannel),
		queue:    make(chan Notification, 1000), // Buffer for 1000 notifications
		workers:  3,                             // 3 worker goroutines
		stopCh:   make(chan struct{}),
		eventBus: eventBus,
	}

	// Initialize channels
	nm.initializeChannels()

	// Start workers
	nm.startWorkers()

	// Subscribe to events
	nm.subscribeToEvents()

	return nm
}

// SendNotification sends a notification via the specified channel
func (nm *NotificationManager) SendNotification(notif Notification) error {
	select {
	case nm.queue <- notif:
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("notification queue is full")
	}
}

// SendEventNotification creates and sends a notification for an event
func (nm *NotificationManager) SendEventNotification(event events.Event, channels []NotificationChannel) error {
	for _, channel := range channels {
		notif := Notification{
			ID:        fmt.Sprintf("notif_%d", time.Now().UnixNano()),
			Title:     nm.getEventTitle(event),
			Message:   event.Message,
			Channel:   channel,
			Recipient: nm.getRecipientForChannel(channel),
			Event:     event,
			Priority:  nm.getPriorityForSeverity(event.Severity),
			Timestamp: time.Now(),
			Template:  nm.getTemplateForEvent(event),
			Data:      event.Data,
		}

		if err := nm.SendNotification(notif); err != nil {
			log.Printf("Failed to queue notification for event %s: %v", event.ID, err)
			return err
		}
	}

	return nil
}

// Shutdown gracefully shuts down the notification manager
func (nm *NotificationManager) Shutdown() {
	close(nm.stopCh)

	// Wait for workers to finish
	time.Sleep(2 * time.Second)

	log.Println("Notification manager shutdown complete")
}

// Private methods

func (nm *NotificationManager) initializeChannels() {
	// Initialize Slack channel
	if nm.config.Notifications.Slack.Enabled {
		nm.channels[ChannelSlack] = ChannelSlack
	}

	// Initialize Email channel
	if nm.config.Notifications.Email.Enabled {
		nm.channels[ChannelEmail] = ChannelEmail
	}

	// Initialize Telegram channel
	if nm.config.Notifications.Telegram.Enabled {
		nm.channels[ChannelTelegram] = ChannelTelegram
	}

	log.Printf("Initialized notification channels: %v", nm.getActiveChannels())
}

func (nm *NotificationManager) startWorkers() {
	for i := 0; i < nm.workers; i++ {
		go nm.worker(i)
	}
	log.Printf("Started %d notification workers", nm.workers)
}

func (nm *NotificationManager) worker(id int) {
	log.Printf("Notification worker %d started", id)

	for {
		select {
		case notif := <-nm.queue:
			if err := nm.sendNotification(notif); err != nil {
				log.Printf("Worker %d failed to send notification %s: %v",
					id, notif.ID, err)
			}
		case <-nm.stopCh:
			log.Printf("Notification worker %d stopped", id)
			return
		}
	}
}

func (nm *NotificationManager) subscribeToEvents() {
	// Subscribe to all event types for notification processing
	subscriber := &NotificationEventSubscriber{
		ID:      "notification-manager",
		Manager: nm,
	}

	nm.eventBus.Subscribe(subscriber)
	log.Println("Notification manager subscribed to event bus")
}

func (nm *NotificationManager) sendNotification(notif Notification) error {
	switch notif.Channel {
	case ChannelSlack:
		return nm.sendSlackNotification(notif)
	case ChannelEmail:
		return nm.sendEmailNotification(notif)
	case ChannelTelegram:
		return nm.sendTelegramNotification(notif)
	default:
		return fmt.Errorf("unsupported notification channel: %s", notif.Channel)
	}
}

func (nm *NotificationManager) sendSlackNotification(notif Notification) error {
	if !nm.config.Notifications.Slack.Enabled || nm.config.Notifications.Slack.WebhookURL == "" {
		return fmt.Errorf("slack notifications not configured")
	}

	payload := map[string]interface{}{
		"text":       fmt.Sprintf("*%s*\n%s", notif.Title, notif.Message),
		"username":   "LLM Verifier",
		"icon_emoji": ":robot_face:",
		"attachments": []map[string]interface{}{
			{
				"color": nm.getColorForPriority(notif.Priority),
				"fields": []map[string]interface{}{
					{
						"title": "Event Type",
						"value": string(notif.Event.Type),
						"short": true,
					},
					{
						"title": "Severity",
						"value": string(notif.Event.Severity),
						"short": true,
					},
					{
						"title": "Source",
						"value": notif.Event.Source,
						"short": true,
					},
					{
						"title": "Timestamp",
						"value": notif.Event.Timestamp.Format(time.RFC3339),
						"short": true,
					},
				},
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	resp, err := http.Post(nm.config.Notifications.Slack.WebhookURL,
		"application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to send slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack notification failed with status: %d", resp.StatusCode)
	}

	log.Printf("Slack notification sent for event %s", notif.Event.ID)
	return nil
}

func (nm *NotificationManager) sendEmailNotification(notif Notification) error {
	if !nm.config.Notifications.Email.Enabled {
		return fmt.Errorf("email notifications not configured")
	}

	// Parse recipient email
	recipient := notif.Recipient
	if recipient == "" {
		recipient = nm.config.Notifications.Email.DefaultRecipient
		if recipient == "" {
			return fmt.Errorf("no recipient specified for email notification")
		}
	}

	// Create email content
	subject := fmt.Sprintf("[LLM Verifier] %s", notif.Title)
	body := nm.generateEmailBody(notif)

	// Send email
	auth := smtp.PlainAuth("",
		nm.config.Notifications.Email.Username,
		nm.config.Notifications.Email.Password,
		nm.config.Notifications.Email.SMTPHost)

	addr := fmt.Sprintf("%s:%d",
		nm.config.Notifications.Email.SMTPHost,
		nm.config.Notifications.Email.SMTPPort)

	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s", recipient, subject, body))

	err := smtp.SendMail(addr, auth, nm.config.Notifications.Email.Username,
		[]string{recipient}, msg)
	if err != nil {
		return fmt.Errorf("failed to send email notification: %w", err)
	}

	log.Printf("Email notification sent to %s for event %s", recipient, notif.Event.ID)
	return nil
}

func (nm *NotificationManager) sendTelegramNotification(notif Notification) error {
	if !nm.config.Notifications.Telegram.Enabled || nm.config.Notifications.Telegram.BotToken == "" {
		return fmt.Errorf("telegram notifications not configured")
	}

	// Parse chat ID
	chatID := notif.Recipient
	if chatID == "" {
		chatID = nm.config.Notifications.Telegram.ChatID
		if chatID == "" {
			return fmt.Errorf("no chat ID specified for telegram notification")
		}
	}

	// Create message
	message := fmt.Sprintf("🚨 *%s*\n\n%s\n\n📅 %s\n🔍 Type: %s\n⚠️ Severity: %s",
		notif.Title,
		notif.Message,
		notif.Event.Timestamp.Format("2006-01-02 15:04:05"),
		notif.Event.Type,
		notif.Event.Severity)

	// Send via Telegram Bot API
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage",
		nm.config.Notifications.Telegram.BotToken)

	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "Markdown",
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram payload: %w", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to send telegram notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram notification failed with status: %d", resp.StatusCode)
	}

	log.Printf("Telegram notification sent to chat %s for event %s", chatID, notif.Event.ID)
	return nil
}

func (nm *NotificationManager) generateEmailBody(notif Notification) string {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>{{.Title}}</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
    <div style="background-color: #f8f9fa; padding: 20px; border-radius: 5px; margin: 20px 0;">
        <h2 style="color: #333; margin-top: 0;">{{.Title}}</h2>
        <p style="font-size: 16px; line-height: 1.5; color: #666;">{{.Message}}</p>

        <div style="background-color: white; padding: 15px; border-radius: 3px; margin: 15px 0;">
            <h3 style="margin-top: 0; color: #333;">Event Details</h3>
            <table style="width: 100%; border-collapse: collapse;">
                <tr>
                    <td style="padding: 5px 10px; border-bottom: 1px solid #eee;"><strong>Type:</strong></td>
                    <td style="padding: 5px 10px; border-bottom: 1px solid #eee;">{{.Event.Type}}</td>
                </tr>
                <tr>
                    <td style="padding: 5px 10px; border-bottom: 1px solid #eee;"><strong>Severity:</strong></td>
                    <td style="padding: 5px 10px; border-bottom: 1px solid #eee;">{{.Event.Severity}}</td>
                </tr>
                <tr>
                    <td style="padding: 5px 10px; border-bottom: 1px solid #eee;"><strong>Source:</strong></td>
                    <td style="padding: 5px 10px; border-bottom: 1px solid #eee;">{{.Event.Source}}</td>
                </tr>
                <tr>
                    <td style="padding: 5px 10px;"><strong>Timestamp:</strong></td>
                    <td style="padding: 5px 10px;">{{.Event.Timestamp.Format "2006-01-02 15:04:05"}}</td>
                </tr>
            </table>
        </div>

        <div style="text-align: center; margin-top: 20px; color: #666; font-size: 12px;">
            This notification was generated by LLM Verifier
        </div>
    </div>
</body>
</html>
`

	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		log.Printf("Failed to parse email template: %v", err)
		return fmt.Sprintf("%s\n\n%s", notif.Title, notif.Message)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, notif); err != nil {
		log.Printf("Failed to execute email template: %v", err)
		return fmt.Sprintf("%s\n\n%s", notif.Title, notif.Message)
	}

	return buf.String()
}

// Helper methods

func (nm *NotificationManager) getActiveChannels() []NotificationChannel {
	var active []NotificationChannel
	for channel := range nm.channels {
		active = append(active, channel)
	}
	return active
}

func (nm *NotificationManager) getEventTitle(event events.Event) string {
	switch event.Type {
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

func (nm *NotificationManager) getPriorityForSeverity(severity events.EventSeverity) string {
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

func (nm *NotificationManager) getColorForPriority(priority string) string {
	switch priority {
	case "high":
		return "danger"
	case "medium":
		return "warning"
	default:
		return "good"
	}
}

func (nm *NotificationManager) getRecipientForChannel(channel NotificationChannel) string {
	switch channel {
	case ChannelSlack:
		return nm.config.Notifications.Slack.WebhookURL
	case ChannelEmail:
		return nm.config.Notifications.Email.DefaultRecipient
	case ChannelTelegram:
		return nm.config.Notifications.Telegram.ChatID
	default:
		return ""
	}
}

func (nm *NotificationManager) getTemplateForEvent(event events.Event) string {
	switch event.Type {
	case events.EventTypeModelVerified:
		return "model_verified"
	case events.EventTypeModelVerificationFailed:
		return "verification_failed"
	case events.EventTypeScoreChanged:
		return "score_changed"
	default:
		return "default"
	}
}

// NotificationEventSubscriber implements the event subscriber interface
type NotificationEventSubscriber struct {
	ID      string
	Manager *NotificationManager
}

func (nes *NotificationEventSubscriber) HandleEvent(event events.Event) error {
	// Determine which channels to notify based on event type and severity
	channels := nes.getChannelsForEvent(event)

	if len(channels) > 0 {
		return nes.Manager.SendEventNotification(event, channels)
	}

	return nil
}

func (nes *NotificationEventSubscriber) GetID() string {
	return nes.ID
}

func (nes *NotificationEventSubscriber) GetTypes() []events.EventType {
	// Subscribe to all event types
	return []events.EventType{
		events.EventTypeModelVerified,
		events.EventTypeModelVerificationFailed,
		events.EventTypeScoreChanged,
		events.EventTypeVerificationStarted,
		events.EventTypeVerificationCompleted,
		events.EventTypeErrorOccurred,
	}
}

func (nes *NotificationEventSubscriber) IsActive() bool {
	return true
}

func (nes *NotificationEventSubscriber) getChannelsForEvent(event events.Event) []NotificationChannel {
	var channels []NotificationChannel

	// Critical and error events go to all channels
	if event.Severity == events.EventSeverityCritical || event.Severity == events.EventSeverityError {
		for channel := range nes.Manager.channels {
			channels = append(channels, channel)
		}
		return channels
	}

	// Warning events go to Slack and email
	if event.Severity == events.EventSeverityWarning {
		if _, ok := nes.Manager.channels[ChannelSlack]; ok {
			channels = append(channels, ChannelSlack)
		}
		if _, ok := nes.Manager.channels[ChannelEmail]; ok {
			channels = append(channels, ChannelEmail)
		}
	}

	// Info events only go to Slack
	if event.Severity == events.EventSeverityInfo {
		if _, ok := nes.Manager.channels[ChannelSlack]; ok {
			channels = append(channels, ChannelSlack)
		}
	}

	return channels
}
