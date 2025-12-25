package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// MatrixNotifier handles Matrix notifications
type MatrixNotifier struct {
	homeserverURL string
	accessToken   string
	roomID        string
}

// NewMatrixNotifier creates a new Matrix notifier
func NewMatrixNotifier(homeserverURL, accessToken, roomID string) *MatrixNotifier {
	return &MatrixNotifier{
		homeserverURL: homeserverURL,
		accessToken:   accessToken,
		roomID:        roomID,
	}
}

// SendNotification sends a Matrix notification
func (mn *MatrixNotifier) SendNotification(event *Event) error {
	message := mn.buildMessage(event)

	payload := map[string]interface{}{
		"msgtype":        "m.text",
		"body":           message,
		"format":         "org.matrix.custom.html",
		"formatted_body": mn.buildHTMLMessage(event),
	}

	return mn.sendMessage(payload)
}

// buildMessage builds a plain text message for Matrix
func (mn *MatrixNotifier) buildMessage(event *Event) string {
	severityEmoji := mn.getSeverityEmoji(event.Severity)

	message := fmt.Sprintf("%s [%s] %s\n\n%s\n\nSeverity: %s\nSource: %s\nTime: %s",
		severityEmoji,
		strings.ToUpper(string(event.Severity)),
		event.Title,
		event.Message,
		event.Severity,
		event.Source,
		event.Timestamp.Format("2006-01-02 15:04:05"))

	if event.ModelID != nil {
		message += fmt.Sprintf("\nModel ID: %d", *event.ModelID)
	}
	if event.ProviderID != nil {
		message += fmt.Sprintf("\nProvider ID: %d", *event.ProviderID)
	}
	if event.ClientID != nil {
		message += fmt.Sprintf("\nClient ID: %s", *event.ClientID)
	}

	return message
}

// buildHTMLMessage builds an HTML formatted message for Matrix
func (mn *MatrixNotifier) buildHTMLMessage(event *Event) string {
	severityColor := mn.getSeverityColor(event.Severity)
	severityEmoji := mn.getSeverityEmoji(event.Severity)

	message := fmt.Sprintf("<strong>%s <span style=\"color: %s;\">[%s]</span> %s</strong><br><br>%s<br><br>",
		severityEmoji,
		severityColor,
		strings.ToUpper(string(event.Severity)),
		event.Title,
		event.Message)

	message += fmt.Sprintf("<strong>Severity:</strong> %s<br>", event.Severity)
	message += fmt.Sprintf("<strong>Source:</strong> %s<br>", event.Source)
	message += fmt.Sprintf("<strong>Time:</strong> %s<br>", event.Timestamp.Format("2006-01-02 15:04:05"))

	if event.ModelID != nil {
		message += fmt.Sprintf("<strong>Model ID:</strong> %d<br>", *event.ModelID)
	}
	if event.ProviderID != nil {
		message += fmt.Sprintf("<strong>Provider ID:</strong> %d<br>", *event.ProviderID)
	}
	if event.ClientID != nil {
		message += fmt.Sprintf("<strong>Client ID:</strong> %s<br>", *event.ClientID)
	}

	return message
}

// getSeverityEmoji returns the appropriate emoji for the severity
func (mn *MatrixNotifier) getSeverityEmoji(severity Severity) string {
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

// getSeverityColor returns the appropriate color for the severity
func (mn *MatrixNotifier) getSeverityColor(severity Severity) string {
	switch severity {
	case SeverityCritical:
		return "#FF0000"
	case SeverityError:
		return "#FF4444"
	case SeverityWarning:
		return "#FFA500"
	case SeverityInfo:
		return "#008000"
	case SeverityDebug:
		return "#808080"
	default:
		return "#808080"
	}
}

// sendMessage sends the message to Matrix
func (mn *MatrixNotifier) sendMessage(payload map[string]interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Matrix payload: %w", err)
	}

	url := fmt.Sprintf("%s/_matrix/client/r0/rooms/%s/send/m.room.message?access_token=%s",
		mn.homeserverURL, mn.roomID, mn.accessToken)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Matrix request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Matrix message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Matrix API returned status %d", resp.StatusCode)
	}

	return nil
}
