package notifications

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/smtp"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"llm-verifier/database"
)

// NotificationType defines the type of notification
type NotificationType string

const (
	NotificationTypeVerificationCompleted NotificationType = "verification_completed"
	NotificationTypeVerificationFailed    NotificationType = "verification_failed"
	NotificationTypeScoreChanged          NotificationType = "score_changed"
	NotificationTypeNewModel              NotificationType = "new_model"
	NotificationTypeIssueDetected         NotificationType = "issue_detected"
	NotificationTypeScheduledTest         NotificationType = "scheduled_test"
	NotificationTypeSystemError           NotificationType = "system_error"
)

// NotificationChannel defines the notification channel
type NotificationChannel string

const (
	ChannelSlack    NotificationChannel = "slack"
	ChannelEmail    NotificationChannel = "email"
	ChannelTelegram NotificationChannel = "telegram"
	ChannelMatrix   NotificationChannel = "matrix"
	ChannelWhatsApp NotificationChannel = "whatsapp"
)

// NotificationPriority defines the notification priority
type NotificationPriority string

const (
	PriorityLow      NotificationPriority = "low"
	PriorityNormal   NotificationPriority = "normal"
	PriorityHigh     NotificationPriority = "high"
	PriorityCritical NotificationPriority = "critical"
)

// Notification represents a notification to be sent
type Notification struct {
	ID         string                 `json:"id"`
	Type       NotificationType       `json:"type"`
	Channel    NotificationChannel    `json:"channel"`
	Priority   NotificationPriority   `json:"priority"`
	Title      string                 `json:"title"`
	Message    string                 `json:"message"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Recipient  string                 `json:"recipient"`
	Sent       bool                   `json:"sent"`
	Error      string                 `json:"error,omitempty"`
	RetryCount int                    `json:"retry_count"`
}

// NotificationConfig holds configuration for notification channels
type NotificationConfig struct {
	Slack    SlackConfig    `yaml:"slack" json:"slack"`
	Email    EmailConfig    `yaml:"email" json:"email"`
	Telegram TelegramConfig `yaml:"telegram" json:"telegram"`
	Matrix   MatrixConfig   `yaml:"matrix" json:"matrix"`
	WhatsApp WhatsAppConfig `yaml:"whatsapp" json:"whatsapp"`
}

// SlackConfig holds Slack notification configuration
type SlackConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	WebhookURL string `yaml:"webhook_url" json:"webhook_url"`
	Channel    string `yaml:"channel" json:"channel"`
	Username   string `yaml:"username" json:"username"`
	IconEmoji  string `yaml:"icon_emoji" json:"icon_emoji"`
	IconURL    string `yaml:"icon_url" json:"icon_url"`
	Timeout    int    `yaml:"timeout" json:"timeout"` // seconds
}

// EmailConfig holds email notification configuration
type EmailConfig struct {
	Enabled      bool     `yaml:"enabled" json:"enabled"`
	SMTPHost     string   `yaml:"smtp_host" json:"smtp_host"`
	SMTPPort     int      `yaml:"smtp_port" json:"smtp_port"`
	Username     string   `yaml:"username" json:"username"`
	Password     string   `yaml:"password" json:"password"`
	From         string   `yaml:"from" json:"from"`
	To           []string `yaml:"to" json:"to"`
	UseTLS       bool     `yaml:"use_tls" json:"use_tls"`
	SkipVerify   bool     `yaml:"skip_verify" json:"skip_verify"`
	TemplatesDir string   `yaml:"templates_dir" json:"templates_dir"`
}

// TelegramConfig holds Telegram notification configuration
type TelegramConfig struct {
	Enabled   bool   `yaml:"enabled" json:"enabled"`
	BotToken  string `yaml:"bot_token" json:"bot_token"`
	ChatID    string `yaml:"chat_id" json:"chat_id"`
	ParseMode string `yaml:"parse_mode" json:"parse_mode"`
	Timeout   int    `yaml:"timeout" json:"timeout"` // seconds
}

// MatrixConfig holds Matrix notification configuration
type MatrixConfig struct {
	Enabled     bool   `yaml:"enabled" json:"enabled"`
	Homeserver  string `yaml:"homeserver" json:"homeserver"`
	UserID      string `yaml:"user_id" json:"user_id"`
	AccessToken string `yaml:"access_token" json:"access_token"`
	RoomID      string `yaml:"room_id" json:"room_id"`
	Timeout     int    `yaml:"timeout" json:"timeout"` // seconds
}

// WhatsAppConfig holds WhatsApp notification configuration
type WhatsAppConfig struct {
	Enabled     bool   `yaml:"enabled" json:"enabled"`
	APIKey      string `yaml:"api_key" json:"api_key"`
	PhoneNumber string `yaml:"phone_number" json:"phone_number"`
	Timeout     int    `yaml:"timeout" json:"timeout"` // seconds
}

// NotificationManager manages sending notifications through various channels
type NotificationManager struct {
	config    *NotificationConfig
	database  *database.Database
	templates map[string]*template.Template
}

// NewNotificationManager creates a new notification manager
func NewNotificationManager(config *NotificationConfig, db *database.Database) *NotificationManager {
	manager := &NotificationManager{
		config:    config,
		database:  db,
		templates: make(map[string]*template.Template),
	}

	// Load email templates if email is enabled
	if config.Email.Enabled {
		manager.loadTemplates()
	}

	return manager
}

// SendNotification sends a notification through the specified channel
func (nm *NotificationManager) SendNotification(notification *Notification) error {
	var err error

	switch notification.Channel {
	case ChannelSlack:
		err = nm.sendSlackNotification(notification)
	case ChannelEmail:
		err = nm.sendEmailNotification(notification)
	case ChannelTelegram:
		err = nm.sendTelegramNotification(notification)
	case ChannelMatrix:
		err = nm.sendMatrixNotification(notification)
	case ChannelWhatsApp:
		err = nm.sendWhatsAppNotification(notification)
	default:
		return fmt.Errorf("unsupported notification channel: %s", notification.Channel)
	}

	// Update notification status in database
	if nm.database != nil {
		notification.Sent = (err == nil)
		if err != nil {
			notification.Error = err.Error()
			notification.RetryCount++
		}

		// Store notification in database
		if saveErr := nm.database.CreateNotification(&database.Notification{
			Type:       string(notification.Type),
			Channel:    string(notification.Channel),
			Priority:   string(notification.Priority),
			Title:      notification.Title,
			Message:    notification.Message,
			Data:       notification.Data,
			Recipient:  notification.Recipient,
			Sent:       notification.Sent,
			Error:      notification.Error,
			RetryCount: notification.RetryCount,
			CreatedAt:  notification.Timestamp,
		}); saveErr != nil {
			return fmt.Errorf("failed to save notification: %w", saveErr)
		}
	}

	return err
}

// sendSlackNotification sends a notification via Slack webhook
func (nm *NotificationManager) sendSlackNotification(notification *Notification) error {
	if !nm.config.Slack.Enabled || nm.config.Slack.WebhookURL == "" {
		return fmt.Errorf("slack notifications not configured")
	}

	payload := map[string]interface{}{
		"text":       fmt.Sprintf("*%s*\n%s", notification.Title, notification.Message),
		"username":   nm.config.Slack.Username,
		"icon_emoji": nm.config.Slack.IconEmoji,
		"icon_url":   nm.config.Slack.IconURL,
		"channel":    nm.config.Slack.Channel,
	}

	// Add attachments for richer formatting if data is provided
	if notification.Data != nil {
		attachments := []map[string]interface{}{
			{
				"color":  nm.getSlackColor(notification.Priority),
				"fields": nm.formatSlackFields(notification.Data),
			},
		}
		payload["attachments"] = attachments
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	client := &http.Client{Timeout: time.Duration(nm.config.Slack.Timeout) * time.Second}
	resp, err := client.Post(nm.config.Slack.WebhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}

// getSlackColor returns color based on notification priority
func (nm *NotificationManager) getSlackColor(priority NotificationPriority) string {
	switch priority {
	case PriorityCritical:
		return "danger"
	case PriorityHigh:
		return "warning"
	case PriorityNormal:
		return "good"
	case PriorityLow:
		return "#808080"
	default:
		return "#808080"
	}
}

// formatSlackFields formats notification data as Slack attachment fields
func (nm *NotificationManager) formatSlackFields(data map[string]interface{}) []map[string]interface{} {
	fields := make([]map[string]interface{}, 0, len(data))

	for key, value := range data {
		field := map[string]interface{}{
			"title": key,
			"value": fmt.Sprintf("%v", value),
			"short": true,
		}
		fields = append(fields, field)
	}

	return fields
}

// sendEmailNotification sends a notification via email
func (nm *NotificationManager) sendEmailNotification(notification *Notification) error {
	if !nm.config.Email.Enabled {
		return fmt.Errorf("email notifications not configured")
	}

	// Prepare email content
	subject := fmt.Sprintf("[%s] %s", strings.ToUpper(string(notification.Type)), notification.Title)

	// Use template if available, otherwise use plain message
	body := notification.Message
	if tmpl, exists := nm.templates[string(notification.Type)]; exists {
		var buf bytes.Buffer
		data := map[string]interface{}{
			"Title":     notification.Title,
			"Message":   notification.Message,
			"Data":      notification.Data,
			"Priority":  notification.Priority,
			"Timestamp": notification.Timestamp.Format("2006-01-02 15:04:05"),
			"Type":      notification.Type,
		}
		if err := tmpl.Execute(&buf, data); err == nil {
			body = buf.String()
		}
	}

	// Create email message
	message := nm.buildEmailMessage(subject, body, notification)

	// Send email via SMTP
	return nm.sendSMTPMail(subject, message)
}

// buildEmailMessage constructs the email message
func (nm *NotificationManager) buildEmailMessage(subject, body string, notification *Notification) []byte {
	var message bytes.Buffer

	// Email headers
	message.WriteString(fmt.Sprintf("From: %s\r\n", nm.config.Email.From))
	message.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(nm.config.Email.To, ",")))
	message.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	message.WriteString("MIME-Version: 1.0\r\n")
	message.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	message.WriteString("\r\n")

	// Email body
	if strings.Contains(body, "<html>") || strings.Contains(body, "<!DOCTYPE") {
		// Already HTML
		message.WriteString(body)
	} else {
		// Convert plain text to HTML
		htmlBody := strings.ReplaceAll(body, "\n", "<br>")
		message.WriteString(fmt.Sprintf("<html><body>%s</body></html>", htmlBody))
	}

	return message.Bytes()
}

// sendSMTPMail sends email via SMTP
func (nm *NotificationManager) sendSMTPMail(subject string, message []byte) error {
	// SMTP server configuration
	addr := fmt.Sprintf("%s:%d", nm.config.Email.SMTPHost, nm.config.Email.SMTPPort)

	// Authentication
	auth := smtp.PlainAuth("", nm.config.Email.Username, nm.config.Email.Password, nm.config.Email.SMTPHost)

	// TLS configuration
	tlsConfig := &tls.Config{
		InsecureSkipVerify: nm.config.Email.SkipVerify,
		ServerName:         nm.config.Email.SMTPHost,
	}

	// Establish connection
	conn, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	// Start TLS if enabled
	if nm.config.Email.UseTLS {
		if err = conn.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	// Authenticate
	if err = conn.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	// Set sender
	if err = conn.Mail(nm.config.Email.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, to := range nm.config.Email.To {
		if err = conn.Rcpt(to); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", to, err)
		}
	}

	// Send data
	w, err := conn.Data()
	if err != nil {
		return fmt.Errorf("failed to initiate data transfer: %w", err)
	}

	_, err = w.Write(message)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close data transfer: %w", err)
	}

	// Send QUIT
	err = conn.Quit()
	if err != nil {
		return fmt.Errorf("failed to quit SMTP connection: %w", err)
	}

	return nil
}

// sendTelegramNotification sends a notification via Telegram bot
func (nm *NotificationManager) sendTelegramNotification(notification *Notification) error {
	if !nm.config.Telegram.Enabled || nm.config.Telegram.BotToken == "" {
		return fmt.Errorf("telegram notifications not configured")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", nm.config.Telegram.BotToken)

	message := fmt.Sprintf("*%s*\n%s", notification.Title, notification.Message)
	if notification.Data != nil {
		message += "\n\nDetails:"
		for key, value := range notification.Data {
			message += fmt.Sprintf("\n• %s: %v", key, value)
		}
	}

	payload := map[string]interface{}{
		"chat_id":    nm.config.Telegram.ChatID,
		"text":       message,
		"parse_mode": nm.config.Telegram.ParseMode,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram payload: %w", err)
	}

	client := &http.Client{Timeout: time.Duration(nm.config.Telegram.Timeout) * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send telegram notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}

// sendMatrixNotification sends a notification via Matrix
func (nm *NotificationManager) sendMatrixNotification(notification *Notification) error {
	if !nm.config.Matrix.Enabled {
		return fmt.Errorf("matrix notifications not configured")
	}

	// This is a simplified Matrix implementation
	// In a real implementation, you'd use a Matrix client library
	url := fmt.Sprintf("%s/_matrix/client/r0/rooms/%s/send/m.room.message",
		nm.config.Matrix.Homeserver, nm.config.Matrix.RoomID)

	message := fmt.Sprintf("**%s**\n%s", notification.Title, notification.Message)
	if notification.Data != nil {
		message += "\n\nDetails:"
		for key, value := range notification.Data {
			message += fmt.Sprintf("\n• %s: %v", key, value)
		}
	}

	payload := map[string]interface{}{
		"msgtype":        "m.text",
		"body":           message,
		"format":         "org.matrix.custom.html",
		"formatted_body": strings.ReplaceAll(message, "\n", "<br>"),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal matrix payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create matrix request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+nm.config.Matrix.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: time.Duration(nm.config.Matrix.Timeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send matrix notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("matrix API returned status %d", resp.StatusCode)
	}

	return nil
}

// sendWhatsAppNotification sends a notification via WhatsApp
func (nm *NotificationManager) sendWhatsAppNotification(notification *Notification) error {
	if !nm.config.WhatsApp.Enabled {
		return fmt.Errorf("whatsapp notifications not configured")
	}

	// This is a simplified WhatsApp implementation
	// In a real implementation, you'd use WhatsApp Business API
	message := fmt.Sprintf("*%s*\n%s", notification.Title, notification.Message)
	if notification.Data != nil {
		message += "\n\nDetails:"
		for key, value := range notification.Data {
			message += fmt.Sprintf("\n• %s: %v", key, value)
		}
	}

	fmt.Printf("Would send WhatsApp message to %s: %s\n",
		nm.config.WhatsApp.PhoneNumber, message)

	return nil // Placeholder - actual WhatsApp API implementation would go here
}

// loadTemplates loads email templates from the templates directory
func (nm *NotificationManager) loadTemplates() {
	if nm.config.Email.TemplatesDir == "" {
		return
	}

	// Template names correspond to notification types
	templateNames := []string{
		string(NotificationTypeVerificationCompleted),
		string(NotificationTypeVerificationFailed),
		string(NotificationTypeScoreChanged),
		string(NotificationTypeNewModel),
		string(NotificationTypeIssueDetected),
		string(NotificationTypeScheduledTest),
		string(NotificationTypeSystemError),
	}

	for _, templateName := range templateNames {
		templatePath := filepath.Join(nm.config.Email.TemplatesDir, templateName+".html")
		if tmpl, err := template.ParseFiles(templatePath); err == nil {
			nm.templates[templateName] = tmpl
		}
	}
}

// CreateVerificationCompletedNotification creates a verification completed notification
func CreateVerificationCompletedNotification(totalModels, successfulModels int, duration time.Duration) *Notification {
	return &Notification{
		ID:       generateNotificationID(),
		Type:     NotificationTypeVerificationCompleted,
		Channel:  ChannelSlack, // Default to Slack
		Priority: PriorityNormal,
		Title:    "Verification Completed",
		Message:  fmt.Sprintf("Successfully verified %d out of %d models in %v", successfulModels, totalModels, duration.Round(time.Second)),
		Data: map[string]interface{}{
			"total_models":      totalModels,
			"successful_models": successfulModels,
			"failed_models":     totalModels - successfulModels,
			"duration_seconds":  duration.Seconds(),
			"success_rate":      float64(successfulModels) / float64(totalModels) * 100,
		},
		Timestamp: time.Now(),
	}
}

// CreateVerificationFailedNotification creates a verification failed notification
func CreateVerificationFailedNotification(modelName, error string) *Notification {
	return &Notification{
		ID:       generateNotificationID(),
		Type:     NotificationTypeVerificationFailed,
		Channel:  ChannelSlack, // Default to Slack
		Priority: PriorityHigh,
		Title:    "Verification Failed",
		Message:  fmt.Sprintf("Verification failed for model %s: %s", modelName, error),
		Data: map[string]interface{}{
			"model_name": modelName,
			"error":      error,
		},
		Timestamp: time.Now(),
	}
}

// CreateScoreChangedNotification creates a score changed notification
func CreateScoreChangedNotification(modelName string, oldScore, newScore float64) *Notification {
	change := newScore - oldScore
	changeDirection := "increased"
	if change < 0 {
		changeDirection = "decreased"
	}

	return &Notification{
		ID:       generateNotificationID(),
		Type:     NotificationTypeScoreChanged,
		Channel:  ChannelSlack, // Default to Slack
		Priority: PriorityNormal,
		Title:    "Model Score Changed",
		Message: fmt.Sprintf("Score for model %s %s by %.1f points (%.1f → %.1f)",
			modelName, changeDirection, change, oldScore, newScore),
		Data: map[string]interface{}{
			"model_name":       modelName,
			"old_score":        oldScore,
			"new_score":        newScore,
			"change":           change,
			"change_direction": changeDirection,
		},
		Timestamp: time.Now(),
	}
}

// generateNotificationID generates a unique notification ID
func generateNotificationID() string {
	return fmt.Sprintf("notif_%d", time.Now().UnixNano())
}
