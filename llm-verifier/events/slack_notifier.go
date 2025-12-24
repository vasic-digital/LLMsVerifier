package events

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SlackNotifier handles Slack webhook notifications
type SlackNotifier struct {
	webhookURL string
	channel    string
	username   string
}

// NewSlackNotifier creates a new Slack notifier
func NewSlackNotifier(webhookURL, channel, username string) *SlackNotifier {
	return &SlackNotifier{
		webhookURL: webhookURL,
		channel:    channel,
		username:   username,
	}
}

// SendNotification sends a Slack notification
func (sn *SlackNotifier) SendNotification(event *Event) error {
	// Create Slack message payload
	payload := map[string]interface{}{
		"channel":    sn.channel,
		"username":   sn.username,
		"icon_emoji": sn.getSeverityIcon(event.Severity),
		"attachments": []map[string]interface{}{
			{
				"color":  sn.getSeverityColor(event.Severity),
				"title":  event.Title,
				"text":   event.Message,
				"fields": sn.buildFields(event),
				"footer": fmt.Sprintf("LLM Verifier | %s", event.Source),
				"ts":     event.Timestamp.Unix(),
			},
		},
	}

	return sn.sendWebhook(payload)
}

// getSeverityIcon returns the appropriate Slack icon for the severity
func (sn *SlackNotifier) getSeverityIcon(severity Severity) string {
	switch severity {
	case SeverityCritical:
		return ":fire:"
	case SeverityError:
		return ":x:"
	case SeverityWarning:
		return ":warning:"
	case SeverityInfo:
		return ":information_source:"
	case SeverityDebug:
		return ":bug:"
	default:
		return ":bell:"
	}
}

// getSeverityColor returns the appropriate color for the severity
func (sn *SlackNotifier) getSeverityColor(severity Severity) string {
	switch severity {
	case SeverityCritical:
		return "danger"
	case SeverityError:
		return "danger"
	case SeverityWarning:
		return "warning"
	case SeverityInfo:
		return "good"
	case SeverityDebug:
		return "#808080"
	default:
		return "#808080"
	}
}

// buildFields builds Slack attachment fields from event data
func (sn *SlackNotifier) buildFields(event *Event) []map[string]interface{} {
	fields := []map[string]interface{}{
		{
			"title": "Severity",
			"value": string(event.Severity),
			"short": true,
		},
		{
			"title": "Source",
			"value": event.Source,
			"short": true,
		},
	}

	if event.ModelID != nil {
		fields = append(fields, map[string]interface{}{
			"title": "Model ID",
			"value": fmt.Sprintf("%d", *event.ModelID),
			"short": true,
		})
	}

	if event.ProviderID != nil {
		fields = append(fields, map[string]interface{}{
			"title": "Provider ID",
			"value": fmt.Sprintf("%d", *event.ProviderID),
			"short": true,
		})
	}

	if event.ClientID != nil {
		fields = append(fields, map[string]interface{}{
			"title": "Client ID",
			"value": *event.ClientID,
			"short": true,
		})
	}

	return fields
}

// sendWebhook sends the payload to the Slack webhook
func (sn *SlackNotifier) sendWebhook(payload map[string]interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	req, err := http.NewRequest("POST", sn.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create Slack request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send Slack webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}
