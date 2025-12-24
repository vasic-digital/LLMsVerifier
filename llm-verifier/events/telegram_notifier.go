package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// TelegramNotifier handles Telegram bot notifications
type TelegramNotifier struct {
	botToken string
	chatID   string
}

// NewTelegramNotifier creates a new Telegram notifier
func NewTelegramNotifier(botToken, chatID string) *TelegramNotifier {
	return &TelegramNotifier{
		botToken: botToken,
		chatID:   chatID,
	}
}

// SendNotification sends a Telegram notification
func (tn *TelegramNotifier) SendNotification(event *Event) error {
	message := tn.buildMessage(event)

	payload := map[string]interface{}{
		"chat_id":    tn.chatID,
		"text":       message,
		"parse_mode": "Markdown",
	}

	return tn.sendMessage(payload)
}

// buildMessage builds a formatted message for Telegram
func (tn *TelegramNotifier) buildMessage(event *Event) string {
	severityEmoji := tn.getSeverityEmoji(event.Severity)

	message := fmt.Sprintf("%s *%s*\n\n%s\n\n", severityEmoji, event.Title, event.Message)
	message += fmt.Sprintf("🔍 *Severity:* %s\n", event.Severity)
	message += fmt.Sprintf("📡 *Source:* %s\n", event.Source)
	message += fmt.Sprintf("🕒 *Time:* %s\n", event.Timestamp.Format("2006-01-02 15:04:05"))

	if event.ModelID != nil {
		message += fmt.Sprintf("🤖 *Model ID:* %d\n", *event.ModelID)
	}
	if event.ProviderID != nil {
		message += fmt.Sprintf("🏢 *Provider ID:* %d\n", *event.ProviderID)
	}
	if event.ClientID != nil {
		message += fmt.Sprintf("👤 *Client ID:* %s\n", *event.ClientID)
	}

	return message
}

// getSeverityEmoji returns the appropriate emoji for the severity
func (tn *TelegramNotifier) getSeverityEmoji(severity Severity) string {
	switch severity {
	case SeverityCritical:
		return "🔴"
	case SeverityError:
		return "❌"
	case SeverityWarning:
		return "⚠️"
	case SeverityInfo:
		return "ℹ️"
	case SeverityDebug:
		return "🐛"
	default:
		return "🔔"
	}
}

// sendMessage sends the message to Telegram
func (tn *TelegramNotifier) sendMessage(payload map[string]interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Telegram payload: %w", err)
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", tn.botToken)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Telegram request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Telegram message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Telegram API returned status %d", resp.StatusCode)
	}

	return nil
}
