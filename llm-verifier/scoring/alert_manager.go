package scoring

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"llm-verifier/logging"
)

// AlertManagerFixed handles sending alerts for various system events
type AlertManagerFixed struct {
	config         MonitoringConfig
	logger         *logging.Logger
	sentAlerts     map[string]time.Time
	mu             sync.RWMutex
	httpClient     *http.Client
}

// NewAlertManagerFixed creates a new alert manager
func NewAlertManagerFixed(config MonitoringConfig) *AlertManagerFixed {
	return &AlertManagerFixed{
		config:     config,
		logger:     &logging.Logger{},
		sentAlerts: make(map[string]time.Time),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendScoreChangeAlert sends an alert for significant score changes
func (am *AlertManagerFixed) SendScoreChangeAlert(alert ScoreChangeAlert) error {
	if !am.config.Enabled {
		return nil
	}

	// Check cooldown period
	alertKey := fmt.Sprintf("score_change_%s", alert.ModelID)
	if am.isInCooldown(alertKey) {
		am.logger.Debug("Alert in cooldown period", map[string]interface{}{"key": alertKey})
		return nil
	}

	am.logger.Info("Sending score change alert", map[string]interface{}{
		"model_id": alert.ModelID,
		"old_score": alert.OldScore,
		"new_score": alert.NewScore,
		"change": alert.ScoreChange,
		"severity": alert.Severity,
	})

	// Send email alert if enabled
	if am.config.EnableEmailAlerts {
		if err := am.sendEmailAlert(alert); err != nil {
			am.logger.Error("Failed to send email alert", map[string]interface{}{"error": err})
		}
	}

	// Send webhook alert if enabled
	if am.config.EnableWebhookAlerts && am.config.WebhookURL != "" {
		if err := am.sendWebhookAlert(alert); err != nil {
			am.logger.Error("Failed to send webhook alert", map[string]interface{}{"error": err})
		}
	}

	// Record alert sent time
	am.recordAlertSent(alertKey)

	return nil
}

// SendAPIPerformanceAlert sends an alert for API performance issues
func (am *AlertManagerFixed) SendAPIPerformanceAlert(alert APIPerformanceAlert) error {
	if !am.config.Enabled {
		return nil
	}

	// Check cooldown period
	alertKey := fmt.Sprintf("api_performance_%s", alert.APIName)
	if am.isInCooldown(alertKey) {
		am.logger.Debug("Alert in cooldown period", map[string]interface{}{"key": alertKey})
		return nil
	}

	am.logger.Info("Sending API performance alert", map[string]interface{}{
		"api_name": alert.APIName,
		"response_time": alert.ResponseTime,
		"success": alert.Success,
		"threshold": alert.Threshold,
	})

	// Send email alert if enabled
	if am.config.EnableEmailAlerts {
		if err := am.sendEmailAlert(alert); err != nil {
			am.logger.Error("Failed to send email alert", map[string]interface{}{"error": err})
		}
	}

	// Send webhook alert if enabled
	if am.config.EnableWebhookAlerts && am.config.WebhookURL != "" {
		if err := am.sendWebhookAlert(alert); err != nil {
			am.logger.Error("Failed to send webhook alert", map[string]interface{}{"error": err})
		}
	}

	// Record alert sent time
	am.recordAlertSent(alertKey)

	return nil
}

// SendDatabasePerformanceAlert sends an alert for database performance issues
func (am *AlertManagerFixed) SendDatabasePerformanceAlert(alert DatabasePerformanceAlert) error {
	if !am.config.Enabled {
		return nil
	}

	// Check cooldown period
	alertKey := fmt.Sprintf("db_performance_%s", alert.Operation)
	if am.isInCooldown(alertKey) {
		am.logger.Debug("Alert in cooldown period", map[string]interface{}{"key": alertKey})
		return nil
	}

	am.logger.Info("Sending database performance alert", map[string]interface{}{
		"operation": alert.Operation,
		"latency": alert.Latency,
		"success": alert.Success,
		"threshold": alert.Threshold,
	})

	// Send email alert if enabled
	if am.config.EnableEmailAlerts {
		if err := am.sendEmailAlert(alert); err != nil {
			am.logger.Error("Failed to send email alert", map[string]interface{}{"error": err})
		}
	}

	// Send webhook alert if enabled
	if am.config.EnableWebhookAlerts && am.config.WebhookURL != "" {
		if err := am.sendWebhookAlert(alert); err != nil {
			am.logger.Error("Failed to send webhook alert", map[string]interface{}{"error": err})
		}
	}

	// Record alert sent time
	am.recordAlertSent(alertKey)

	return nil
}

// Email alert methods

func (am *AlertManagerFixed) sendEmailAlert(alert interface{}) error {
	// This is a placeholder for email sending functionality
	// In a real implementation, you would integrate with an email service
	
	var subject, body string
	
	switch a := alert.(type) {
	case ScoreChangeAlert:
		subject = fmt.Sprintf("LLM Score Change Alert: %s", a.ModelID)
		body = am.formatScoreChangeEmail(a)
	case APIPerformanceAlert:
		subject = fmt.Sprintf("API Performance Alert: %s", a.APIName)
		body = am.formatAPIPerformanceEmail(a)
	case DatabasePerformanceAlert:
		subject = fmt.Sprintf("Database Performance Alert: %s", a.Operation)
		body = am.formatDatabasePerformanceEmail(a)
	default:
		return fmt.Errorf("unknown alert type: %T", alert)
	}

	am.logger.Info("Email alert prepared", map[string]interface{}{
		"subject": subject,
		"recipients": len(am.config.AlertRecipients),
	})

	// Log the email content for debugging
	am.logger.Debug("Email alert content", map[string]interface{}{
		"subject": subject,
		"body": body,
	})

	// In a real implementation, you would send the email here
	// For now, we'll just log it
	return nil
}

func (am *AlertManagerFixed) formatScoreChangeEmail(alert ScoreChangeAlert) string {
	changeType := "increased"
	if alert.ScoreChange < 0 {
		changeType = "decreased"
	}

	return fmt.Sprintf(`
Model Score Change Alert

Model: %s
Old Score: %.1f
New Score: %.1f
Change: %.1f (%s)
Severity: %s
Time: %s

Component Scores:
- Speed: %.1f
- Efficiency: %.1f
- Cost: %.1f
- Capability: %.1f
- Recency: %.1f

%s
`,
		alert.ModelID,
		alert.OldScore,
		alert.NewScore,
		alert.ScoreChange,
		changeType,
		alert.Severity,
		alert.Timestamp.Format(time.RFC3339),
		alert.Components.SpeedScore,
		alert.Components.EfficiencyScore,
		alert.Components.CostScore,
		alert.Components.CapabilityScore,
		alert.Components.RecencyScore,
		alert.Message,
	)
}

func (am *AlertManagerFixed) formatAPIPerformanceEmail(alert APIPerformanceAlert) string {
	status := "Successful"
	if !alert.Success {
		status = "Failed"
	}

	return fmt.Sprintf(`
API Performance Alert

API: %s
Status: %s
Response Time: %v
Threshold: %v
Time: %s

%s
`,
		alert.APIName,
		status,
		alert.ResponseTime,
		alert.Threshold,
		alert.Timestamp.Format(time.RFC3339),
		alert.Message,
	)
}

func (am *AlertManagerFixed) formatDatabasePerformanceEmail(alert DatabasePerformanceAlert) string {
	status := "Successful"
	if !alert.Success {
		status = "Failed"
	}

	return fmt.Sprintf(`
Database Performance Alert

Operation: %s
Status: %s
Latency: %v
Threshold: %v
Time: %s

%s
`,
		alert.Operation,
		status,
		alert.Latency,
		alert.Threshold,
		alert.Timestamp.Format(time.RFC3339),
		alert.Message,
	)
}

// Webhook alert methods

func (am *AlertManagerFixed) sendWebhookAlert(alert interface{}) error {
	if am.config.WebhookURL == "" {
		return fmt.Errorf("webhook URL not configured")
	}

	payload, err := am.createWebhookPayload(alert)
	if err != nil {
		return fmt.Errorf("failed to create webhook payload: %w", err)
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}

	req, err := http.NewRequest("POST", am.config.WebhookURL, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "LLM-Verifier-Scoring/1.0")

	resp, err := am.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook request failed with status %d", resp.StatusCode)
	}

	am.logger.Info("Webhook alert sent successfully", map[string]interface{}{"status": resp.StatusCode})
	return nil
}

func (am *AlertManagerFixed) createWebhookPayload(alert interface{}) (map[string]interface{}, error) {
	var payload map[string]interface{}
	
	switch a := alert.(type) {
	case ScoreChangeAlert:
		payload = map[string]interface{}{
			"type":         "score_change",
			"model_id":     a.ModelID,
			"old_score":    a.OldScore,
			"new_score":    a.NewScore,
			"change":       a.ScoreChange,
			"severity":     a.Severity,
			"timestamp":    a.Timestamp.Format(time.RFC3339),
			"message":      a.Message,
			"components":   a.Components,
		}
	case APIPerformanceAlert:
		payload = map[string]interface{}{
			"type":          "api_performance",
			"api_name":      a.APIName,
			"response_time": a.ResponseTime.Milliseconds(),
			"success":       a.Success,
			"threshold":     a.Threshold.Milliseconds(),
			"timestamp":     a.Timestamp.Format(time.RFC3339),
			"message":       a.Message,
		}
	case DatabasePerformanceAlert:
		payload = map[string]interface{}{
			"type":      "database_performance",
			"operation": a.Operation,
			"latency":   a.Latency.Milliseconds(),
			"success":   a.Success,
			"threshold": a.Threshold.Milliseconds(),
			"timestamp": a.Timestamp.Format(time.RFC3339),
			"message":   a.Message,
		}
	default:
		return nil, fmt.Errorf("unknown alert type: %T", alert)
	}

	// Add common fields
	payload["alert_id"] = fmt.Sprintf("alert_%d", time.Now().UnixNano())
	payload["version"] = "1.0"
	
	return payload, nil
}

// Cooldown management

func (am *AlertManagerFixed) isInCooldown(alertKey string) bool {
	am.mu.RLock()
	lastSent, exists := am.sentAlerts[alertKey]
	am.mu.RUnlock()

	if !exists {
		return false
	}

	return time.Since(lastSent) < am.config.AlertCooldownPeriod
}

func (am *AlertManagerFixed) recordAlertSent(alertKey string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.sentAlerts[alertKey] = time.Now()
}

// Cleanup old alert records
func (am *AlertManagerFixed) CleanupOldRecords(maxAge time.Duration) {
	am.mu.Lock()
	defer am.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	for key, sentTime := range am.sentAlerts {
		if sentTime.Before(cutoff) {
			delete(am.sentAlerts, key)
		}
	}

	am.logger.Debug("Cleaned up old alert records", map[string]interface{}{"max_age": maxAge})
}

// GetAlertStats returns statistics about sent alerts
func (am *AlertManagerFixed) GetAlertStats() AlertStats {
	am.mu.RLock()
	defer am.mu.RUnlock()

	stats := AlertStats{
		TotalAlerts: len(am.sentAlerts),
		ByType:      make(map[string]int),
		BySeverity:  make(map[string]int),
	}

	// This is a simplified implementation
	// In a real implementation, you would track these separately
	for key := range am.sentAlerts {
		if len(key) > 0 {
			if key[0] == 's' {
				stats.ByType["score_change"]++
			} else if key[0] == 'a' {
				stats.ByType["api_performance"]++
			} else if key[0] == 'd' {
				stats.ByType["database_performance"]++
			}
		}
	}

	return stats
}

// AlertStats represents alert statistics
type AlertStats struct {
	TotalAlerts int            `json:"total_alerts"`
	ByType      map[string]int `json:"by_type"`
	BySeverity  map[string]int `json:"by_severity"`
}