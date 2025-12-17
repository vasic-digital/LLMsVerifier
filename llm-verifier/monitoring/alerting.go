package monitoring

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// AlertStrategy defines the alerting strategy configuration
type AlertStrategy struct {
	Name            string                `json:"name"`
	Description     string                `json:"description"`
	Enabled         bool                  `json:"enabled"`
	Rules           []AlertRule           `json:"rules"`
	Channels        []NotificationChannel `json:"channels"`
	EscalationRules []EscalationRule      `json:"escalation_rules"`
	CooldownPeriod  time.Duration         `json:"cooldown_period"`
}

// AlertRule defines a rule for triggering alerts
type AlertRule struct {
	Name        string         `json:"name"`
	Metric      string         `json:"metric"`
	Condition   AlertCondition `json:"condition"`
	Threshold   float64        `json:"threshold"`
	Duration    time.Duration  `json:"duration"` // How long condition must be true
	Severity    AlertSeverity  `json:"severity"`
	Description string         `json:"description"`
	Enabled     bool           `json:"enabled"`
}

// NotificationChannel defines how alerts are sent
type NotificationChannel struct {
	Type   string            `json:"type"` // email, slack, webhook, etc.
	Config map[string]string `json:"config"`
}

// EscalationRule defines rules for escalating alerts
type EscalationRule struct {
	TriggerAfter time.Duration         `json:"trigger_after"`
	NewSeverity  AlertSeverity         `json:"new_severity"`
	NewChannels  []NotificationChannel `json:"new_channels"`
}

// AlertManager manages the alerting system
type AlertManager struct {
	strategies     map[string]*AlertStrategy
	activeAlerts   map[string]*ActiveAlert
	alertHistory   []*AlertHistory
	metricsTracker *CriticalMetricsTracker
	cooldowns      map[string]time.Time
	maxHistory     int
}

// ActiveAlert represents an currently active alert
type ActiveAlert struct {
	ID          string                 `json:"id"`
	Strategy    string                 `json:"strategy"`
	Rule        string                 `json:"rule"`
	Severity    AlertSeverity          `json:"severity"`
	Message     string                 `json:"message"`
	StartedAt   time.Time              `json:"started_at"`
	LastUpdated time.Time              `json:"last_updated"`
	Occurrences int                    `json:"occurrences"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AlertHistory represents historical alert data
type AlertHistory struct {
	ID        string        `json:"id"`
	AlertID   string        `json:"alert_id"`
	Severity  AlertSeverity `json:"severity"`
	Message   string        `json:"message"`
	StartedAt time.Time     `json:"started_at"`
	EndedAt   *time.Time    `json:"ended_at,omitempty"`
	Duration  time.Duration `json:"duration"`
	Resolved  bool          `json:"resolved"`
}

// NewAlertManager creates a new alert manager
func NewAlertManager(metricsTracker *CriticalMetricsTracker, maxHistory int) *AlertManager {
	return &AlertManager{
		strategies:     make(map[string]*AlertStrategy),
		activeAlerts:   make(map[string]*ActiveAlert),
		alertHistory:   make([]*AlertHistory, 0),
		metricsTracker: metricsTracker,
		cooldowns:      make(map[string]time.Time),
		maxHistory:     maxHistory,
	}
}

// AddStrategy adds an alerting strategy
func (am *AlertManager) AddStrategy(strategy *AlertStrategy) {
	am.strategies[strategy.Name] = strategy
	log.Printf("Added alert strategy: %s", strategy.Name)
}

// RemoveStrategy removes an alerting strategy
func (am *AlertManager) RemoveStrategy(name string) {
	delete(am.strategies, name)
}

// EvaluateStrategies evaluates all enabled strategies
func (am *AlertManager) EvaluateStrategies() {
	for _, strategy := range am.strategies {
		if strategy.Enabled {
			am.evaluateStrategy(strategy)
		}
	}
}

// evaluateStrategy evaluates a single strategy
func (am *AlertManager) evaluateStrategy(strategy *AlertStrategy) {
	for _, rule := range strategy.Rules {
		if !rule.Enabled {
			continue
		}

		// Check cooldown
		cooldownKey := fmt.Sprintf("%s:%s", strategy.Name, rule.Name)
		if lastAlert, exists := am.cooldowns[cooldownKey]; exists {
			if time.Since(lastAlert) < strategy.CooldownPeriod {
				continue // Still in cooldown
			}
		}

		// Evaluate rule
		if am.evaluateRule(rule) {
			alert := am.createAlert(strategy, rule)
			if alert != nil {
				am.sendAlert(strategy, alert)
			}
		}
	}
}

// evaluateRule evaluates if a rule condition is met
func (am *AlertManager) evaluateRule(rule AlertRule) bool {
	// Get recent metric data
	stats := am.metricsTracker.collector.CalculateStats(rule.Metric, rule.Duration)

	if stats.Count == 0 {
		return false // No data available
	}

	latestValue := stats.Latest

	// Check condition
	switch rule.Condition {
	case AlertConditionGreaterThan:
		return latestValue > rule.Threshold
	case AlertConditionLessThan:
		return latestValue < rule.Threshold
	case AlertConditionEqual:
		return latestValue == rule.Threshold
	default:
		return false
	}
}

// createAlert creates an alert from a triggered rule
func (am *AlertManager) createAlert(strategy *AlertStrategy, rule AlertRule) *ActiveAlert {
	alertID := fmt.Sprintf("alert_%s_%s_%d", strategy.Name, rule.Name, time.Now().Unix())

	// Check if alert already exists
	if existing, exists := am.activeAlerts[alertID]; exists {
		existing.Occurrences++
		existing.LastUpdated = time.Now()
		return nil // Don't create duplicate
	}

	alert := &ActiveAlert{
		ID:          alertID,
		Strategy:    strategy.Name,
		Rule:        rule.Name,
		Severity:    rule.Severity,
		Message:     am.formatAlertMessage(strategy, rule),
		StartedAt:   time.Now(),
		LastUpdated: time.Now(),
		Occurrences: 1,
		Metadata: map[string]interface{}{
			"metric":    rule.Metric,
			"threshold": rule.Threshold,
			"condition": rule.Condition,
		},
	}

	am.activeAlerts[alertID] = alert

	// Set cooldown
	cooldownKey := fmt.Sprintf("%s:%s", strategy.Name, rule.Name)
	am.cooldowns[cooldownKey] = time.Now()

	return alert
}

// formatAlertMessage formats an alert message
func (am *AlertManager) formatAlertMessage(strategy *AlertStrategy, rule AlertRule) string {
	stats := am.metricsTracker.collector.CalculateStats(rule.Metric, rule.Duration)

	message := fmt.Sprintf("[%s] %s - %s", strings.ToUpper(string(rule.Severity)),
		strategy.Name, rule.Description)

	if stats.Count > 0 {
		message += fmt.Sprintf(" (Current: %.2f, Threshold: %.2f)", stats.Latest, rule.Threshold)
	}

	return message
}

// sendAlert sends an alert through configured channels
func (am *AlertManager) sendAlert(strategy *AlertStrategy, alert *ActiveAlert) {
	log.Printf("Sending alert: %s", alert.Message)

	// Send through each configured channel
	for _, channel := range strategy.Channels {
		go am.sendToChannel(channel, alert)
	}

	// Check escalation rules
	am.checkEscalation(strategy, alert)
}

// sendToChannel sends an alert to a specific channel
func (am *AlertManager) sendToChannel(channel NotificationChannel, alert *ActiveAlert) {
	switch channel.Type {
	case "log":
		log.Printf("ALERT: %s", alert.Message)
	case "email":
		am.sendEmailAlert(channel, alert)
	case "slack":
		am.sendSlackAlert(channel, alert)
	case "webhook":
		am.sendWebhookAlert(channel, alert)
	default:
		log.Printf("Unknown channel type: %s", channel.Type)
	}
}

// sendEmailAlert sends an alert via email
func (am *AlertManager) sendEmailAlert(channel NotificationChannel, alert *ActiveAlert) {
	// Implementation would integrate with email service
	log.Printf("Email alert to %s: %s", channel.Config["to"], alert.Message)
}

// sendSlackAlert sends an alert via Slack
func (am *AlertManager) sendSlackAlert(channel NotificationChannel, alert *ActiveAlert) {
	// Implementation would integrate with Slack API
	log.Printf("Slack alert to %s: %s", channel.Config["channel"], alert.Message)
}

// sendWebhookAlert sends an alert via webhook
func (am *AlertManager) sendWebhookAlert(channel NotificationChannel, alert *ActiveAlert) {
	// Implementation would send HTTP POST to webhook URL
	log.Printf("Webhook alert to %s: %s", channel.Config["url"], alert.Message)
}

// checkEscalation checks if alert should be escalated
func (am *AlertManager) checkEscalation(strategy *AlertStrategy, alert *ActiveAlert) {
	for _, escalation := range strategy.EscalationRules {
		if time.Since(alert.StartedAt) >= escalation.TriggerAfter {
			// Escalate alert
			alert.Severity = escalation.NewSeverity
			alert.LastUpdated = time.Now()

			log.Printf("Escalating alert %s to severity %s", alert.ID, escalation.NewSeverity)

			// Send through escalation channels
			for _, channel := range escalation.NewChannels {
				go am.sendToChannel(channel, alert)
			}
		}
	}
}

// ResolveAlert resolves an active alert
func (am *AlertManager) ResolveAlert(alertID string, resolutionMessage string) {
	alert, exists := am.activeAlerts[alertID]
	if !exists {
		return
	}

	now := time.Now()
	duration := now.Sub(alert.StartedAt)

	// Create history entry
	history := &AlertHistory{
		ID:        fmt.Sprintf("hist_%d", time.Now().UnixNano()),
		AlertID:   alertID,
		Severity:  alert.Severity,
		Message:   alert.Message,
		StartedAt: alert.StartedAt,
		EndedAt:   &now,
		Duration:  duration,
		Resolved:  true,
	}

	am.alertHistory = append(am.alertHistory, history)

	// Remove from active alerts
	delete(am.activeAlerts, alertID)

	// Maintain history limit
	if len(am.alertHistory) > am.maxHistory {
		am.alertHistory = am.alertHistory[len(am.alertHistory)-am.maxHistory:]
	}

	log.Printf("Resolved alert %s after %v", alertID, duration)
}

// GetActiveAlerts returns all active alerts
func (am *AlertManager) GetActiveAlerts() []*ActiveAlert {
	alerts := make([]*ActiveAlert, 0, len(am.activeAlerts))
	for _, alert := range am.activeAlerts {
		alerts = append(alerts, alert)
	}
	return alerts
}

// GetAlertHistory returns alert history
func (am *AlertManager) GetAlertHistory(limit int) []*AlertHistory {
	history := am.alertHistory
	if len(history) > limit {
		history = history[len(history)-limit:]
	}

	// Return copy
	result := make([]*AlertHistory, len(history))
	copy(result, history)
	return result
}

// SetupDefaultStrategies sets up default alerting strategies
func (am *AlertManager) SetupDefaultStrategies() {
	// Performance Alerting Strategy
	performanceStrategy := &AlertStrategy{
		Name:        "performance_monitoring",
		Description: "Monitor system performance metrics",
		Enabled:     true,
		Rules: []AlertRule{
			{
				Name:        "high_latency",
				Metric:      "request_latency_seconds",
				Condition:   AlertConditionGreaterThan,
				Threshold:   5.0, // 5 seconds
				Duration:    5 * time.Minute,
				Severity:    AlertSeverityWarning,
				Description: "Request latency is too high",
				Enabled:     true,
			},
			{
				Name:        "high_error_rate",
				Metric:      "errors_total",
				Condition:   AlertConditionGreaterThan,
				Threshold:   10.0, // 10 errors
				Duration:    10 * time.Minute,
				Severity:    AlertSeverityError,
				Description: "Error rate is too high",
				Enabled:     true,
			},
			{
				Name:        "ttft_degraded",
				Metric:      "ttft_seconds",
				Condition:   AlertConditionGreaterThan,
				Threshold:   3.0, // 3 seconds
				Duration:    5 * time.Minute,
				Severity:    AlertSeverityWarning,
				Description: "Time to first token is degraded",
				Enabled:     true,
			},
		},
		Channels: []NotificationChannel{
			{Type: "log"},
		},
		EscalationRules: []EscalationRule{
			{
				TriggerAfter: 15 * time.Minute,
				NewSeverity:  AlertSeverityCritical,
				NewChannels: []NotificationChannel{
					{Type: "email", Config: map[string]string{"to": "admin@example.com"}},
				},
			},
		},
		CooldownPeriod: 10 * time.Minute,
	}

	// Security Alerting Strategy
	securityStrategy := &AlertStrategy{
		Name:        "security_monitoring",
		Description: "Monitor security-related events",
		Enabled:     true,
		Rules: []AlertRule{
			{
				Name:        "suspicious_requests",
				Metric:      "suspicious_requests_total",
				Condition:   AlertConditionGreaterThan,
				Threshold:   5.0,
				Duration:    1 * time.Hour,
				Severity:    AlertSeverityCritical,
				Description: "Multiple suspicious requests detected",
				Enabled:     true,
			},
		},
		Channels: []NotificationChannel{
			{Type: "log"},
			{Type: "email", Config: map[string]string{"to": "security@example.com"}},
		},
		CooldownPeriod: 1 * time.Hour,
	}

	am.AddStrategy(performanceStrategy)
	am.AddStrategy(securityStrategy)
}

// Start begins the alerting evaluation loop
func (am *AlertManager) Start(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				am.EvaluateStrategies()
			}
		}
	}()
}

// GetAlertStats returns alerting statistics
func (am *AlertManager) GetAlertStats() map[string]interface{} {
	activeCount := len(am.activeAlerts)
	historyCount := len(am.alertHistory)

	severityCounts := make(map[AlertSeverity]int)
	for _, alert := range am.activeAlerts {
		severityCounts[alert.Severity]++
	}

	strategyCounts := make(map[string]int)
	for _, alert := range am.activeAlerts {
		strategyCounts[alert.Strategy]++
	}

	return map[string]interface{}{
		"active_alerts":   activeCount,
		"total_history":   historyCount,
		"severity_counts": severityCounts,
		"strategy_counts": strategyCounts,
		"strategies":      len(am.strategies),
	}
}
