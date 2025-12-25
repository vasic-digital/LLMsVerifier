package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// WhatsAppNotifier handles WhatsApp notifications via Twilio API
type WhatsAppNotifier struct {
	accountSID string
	authToken  string
	fromNumber string
	toNumbers  []string
}

// NewWhatsAppNotifier creates a new WhatsApp notifier using Twilio
func NewWhatsAppNotifier(accountSID, authToken, fromNumber string, toNumbers []string) *WhatsAppNotifier {
	return &WhatsAppNotifier{
		accountSID: accountSID,
		authToken:  authToken,
		fromNumber: fromNumber,
		toNumbers:  toNumbers,
	}
}

// SendNotification sends a WhatsApp notification via Twilio
func (wn *WhatsAppNotifier) SendNotification(event *Event) error {
	message := wn.buildMessage(event)

	for _, toNumber := range wn.toNumbers {
		if err := wn.sendMessage(toNumber, message); err != nil {
			return fmt.Errorf("failed to send WhatsApp message to %s: %w", toNumber, err)
		}
	}

	return nil
}

// buildMessage builds a formatted message for WhatsApp
func (wn *WhatsAppNotifier) buildMessage(event *Event) string {
	severityEmoji := wn.getSeverityEmoji(event.Severity)

	message := fmt.Sprintf("%s *%s*\n\n%s\n\n*Severity:* %s\n*Source:* %s\n*Time:* %s",
		severityEmoji,
		event.Title,
		event.Message,
		event.Severity,
		event.Source,
		event.Timestamp.Format("2006-01-02 15:04:05"))

	if event.ModelID != nil {
		message += fmt.Sprintf("\n*Model ID:* %d", *event.ModelID)
	}
	if event.ProviderID != nil {
		message += fmt.Sprintf("\n*Provider ID:* %d", *event.ProviderID)
	}
	if event.ClientID != nil {
		message += fmt.Sprintf("\n*Client ID:* %s", *event.ClientID)
	}

	return message
}

// getSeverityEmoji returns the appropriate emoji for the severity
func (wn *WhatsAppNotifier) getSeverityEmoji(severity Severity) string {
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

// sendMessage sends a message to a specific WhatsApp number via Twilio
func (wn *WhatsAppNotifier) sendMessage(toNumber, message string) error {
	payload := map[string]interface{}{
		"From": fmt.Sprintf("whatsapp:%s", wn.fromNumber),
		"To":   fmt.Sprintf("whatsapp:%s", toNumber),
		"Body": message,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal WhatsApp payload: %w", err)
	}

	url := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", wn.accountSID)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create WhatsApp request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(wn.accountSID, wn.authToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send WhatsApp message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("Twilio API returned status %d", resp.StatusCode)
	}

	return nil
}
