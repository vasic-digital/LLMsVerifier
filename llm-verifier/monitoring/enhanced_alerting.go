package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// AlertSeverity represents alert severity levels
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertStatus represents alert status
type AlertStatus string

const (
	AlertStatusActive     AlertStatus = "active"
	AlertStatusResolved   AlertStatus = "resolved"
	AlertStatusSuppressed AlertStatus = "suppressed"
	AlertStatusEscalated  AlertStatus = "escalated"
)

// Alert represents an alert with rich metadata
type Alert struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Rule        string                 `json:"rule"`
	Severity    AlertSeverity          `json:"severity"`
	Status      AlertStatus            `json:"status"`
	Message     string                 `json:"message"`
	Source      string                 `json:"source"`
	Category    string                 `json:"category"`
	Labels      map[string]string      `json:"labels,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	EscalatedAt *time.Time             `json:"escalated_at,omitempty"`
	Count       int                    `json:"count"` // Number of occurrences
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AlertRule represents an alerting rule
type AlertRule struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Enabled     bool              `json:"enabled"`
	Severity    AlertSeverity     `json:"severity"`
	Condition   string            `json:"condition"` // Expression or reference to condition function
	Threshold   float64           `json:"threshold"`
	Duration    time.Duration     `json:"duration"`
	Cooldown    time.Duration     `json:"cooldown"`
	Escalation  *EscalationPolicy `json:"escalation,omitempty"`
	Actions     []AlertAction     `json:"actions,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// EscalationPolicy defines escalation behavior
type EscalationPolicy struct {
	Enabled         bool              `json:"enabled"`
	Levels          []EscalationLevel `json:"levels"`
	AutoEscalate    bool              `json:"auto_escalate"`
	EscalationDelay time.Duration     `json:"escalation_delay"`
	MaxEscalations  int               `json:"max_escalations"`
}

// EscalationLevel represents an escalation level
type EscalationLevel struct {
	Level    int           `json:"level"`
	Name     string        `json:"name"`
	Severity AlertSeverity `json:"severity"`
	Actions  []AlertAction `json:"actions"`
	Delay    time.Duration `json:"delay"`
}

// AlertAction represents an action to take when alert triggers
type AlertAction struct {
	Type    string                 `json:"type"`
	Config  map[string]interface{} `json:"config,omitempty"`
	Enabled bool                   `json:"enabled"`
}

// AlertMetrics represents alert system metrics
type AlertMetrics struct {
	TotalAlerts      int            `json:"total_alerts"`
	ActiveAlerts     int            `json:"active_alerts"`
	EscalatedAlerts  int            `json:"escalated_alerts"`
	SuppressedAlerts int            `json:"suppressed_alerts"`
	ResolutionTime   time.Duration  `json:"avg_resolution_time"`
	AlertsBySeverity map[string]int `json:"alerts_by_severity"`
	AlertsByCategory map[string]int `json:"alerts_by_category"`
	TopAlerts        []*Alert       `json:"top_alerts"`
}

// MetricsCollector interface for metrics collection
type MetricsCollector interface {
	GetMetric(name string) (float64, error)
	IncrementCounter(name string, value float64) error
}

// AlertManager manages intelligent alerting with escalation
type AlertManager struct {
	alerts      map[string]*Alert
	rules       map[string]*AlertRule
	metrics     MetricsCollector
	escalations map[string][]*Alert
	mu          sync.RWMutex
	cooldowns   map[string]time.Time
	debounce    map[string]*Alert
}

// NewAlertManager creates a new intelligent alert manager
func NewAlertManager(metrics MetricsCollector) *AlertManager {
	am := &AlertManager{
		alerts:      make(map[string]*Alert),
		rules:       make(map[string]*AlertRule),
		metrics:     metrics,
		escalations: make(map[string][]*Alert),
		cooldowns:   make(map[string]time.Time),
		debounce:    make(map[string]*Alert),
	}

	// Setup default alert rules
	am.setupDefaultRules()

	// Start background processors
	go am.alertProcessor()
	go am.escalationProcessor()
	go am.cooldownManager()

	return am
}

// setupDefaultRules creates default intelligent alerting rules
func (am *AlertManager) setupDefaultRules() {
	rules := []*AlertRule{
		{
			ID:          "high_error_rate",
			Name:        "High Error Rate",
			Description: "Triggers when error rate exceeds threshold",
			Enabled:     true,
			Severity:    AlertSeverityWarning,
			Condition:   "error_rate",
			Threshold:   0.1, // 10% error rate
			Duration:    5 * time.Minute,
			Cooldown:    15 * time.Minute,
			Escalation: &EscalationPolicy{
				Enabled: true,
				Levels: []EscalationLevel{
					{Level: 1, Name: "Team Lead", Severity: AlertSeverityWarning, Delay: 5 * time.Minute},
					{Level: 2, Name: "Manager", Severity: AlertSeverityError, Delay: 10 * time.Minute},
					{Level: 3, Name: "Director", Severity: AlertSeverityCritical, Delay: 15 * time.Minute},
				},
				AutoEscalate:    true,
				EscalationDelay: 5 * time.Minute,
				MaxEscalations:  3,
			},
			Actions: []AlertAction{
				{Type: "email", Config: map[string]interface{}{"template": "error_rate_alert"}, Enabled: true},
				{Type: "slack", Config: map[string]interface{}{"channel": "#alerts"}, Enabled: true},
			},
		},
		{
			ID:          "high_latency",
			Name:        "High Latency",
			Description: "Triggers when response latency exceeds threshold",
			Enabled:     true,
			Severity:    AlertSeverityWarning,
			Condition:   "latency",
			Threshold:   5000, // 5 seconds
			Duration:    5 * time.Minute,
			Cooldown:    10 * time.Minute,
			Escalation: &EscalationPolicy{
				Enabled: true,
				Levels: []EscalationLevel{
					{Level: 1, Name: "Performance Team", Severity: AlertSeverityError, Delay: 2 * time.Minute},
				},
				AutoEscalate:    true,
				EscalationDelay: 2 * time.Minute,
				MaxEscalations:  2,
			},
			Actions: []AlertAction{
				{Type: "webhook", Config: map[string]interface{}{"url": "https://api.company.com/alerts"}, Enabled: true},
				{Type: "pagerduty", Config: map[string]interface{}{"service_key": "pagerduty_key"}, Enabled: false},
			},
		},
		{
			ID:          "system_resources",
			Name:        "System Resources",
			Description: "Triggers when system resources are critically low",
			Enabled:     true,
			Severity:    AlertSeverityCritical,
			Condition:   "resource_usage",
			Threshold:   0.95, // 95% resource usage
			Duration:    3 * time.Minute,
			Cooldown:    5 * time.Minute,
			Escalation: &EscalationPolicy{
				Enabled: true,
				Levels: []EscalationLevel{
					{Level: 1, Name: "On-call Engineer", Severity: AlertSeverityCritical, Delay: 0},
					{Level: 2, Name: "Systems Administrator", Severity: AlertSeverityCritical, Delay: 0},
				},
				AutoEscalate:    true,
				EscalationDelay: 0,
				MaxEscalations:  2,
			},
			Actions: []AlertAction{
				{Type: "sms", Config: map[string]interface{}{"numbers": []string{"+1234567890"}}, Enabled: true},
				{Type: "phone_call", Config: map[string]interface{}{"numbers": []string{"+1234567890"}}, Enabled: true},
			},
		},
		{
			ID:          "provider_health",
			Name:        "Provider Health",
			Description: "Triggers when LLM provider health degrades",
			Enabled:     true,
			Severity:    AlertSeverityError,
			Condition:   "provider_health",
			Threshold:   50.0, // Health score below 50%
			Duration:    2 * time.Minute,
			Cooldown:    5 * time.Minute,
			Escalation: &EscalationPolicy{
				Enabled: true,
				Levels: []EscalationLevel{
					{Level: 1, Name: "API Team", Severity: AlertSeverityWarning, Delay: 10 * time.Minute},
					{Level: 2, Name: "Infrastructure Team", Severity: AlertSeverityError, Delay: 5 * time.Minute},
				},
				AutoEscalate:    true,
				EscalationDelay: 10 * time.Minute,
				MaxEscalations:  3,
			},
			Actions: []AlertAction{
				{Type: "email", Config: map[string]interface{}{"template": "provider_health"}, Enabled: true},
				{Type: "jira", Config: map[string]interface{}{"project": "INFRA", "issue_type": "PROVIDER_ISSUE"}, Enabled: true},
			},
		},
	}

	for _, rule := range rules {
		am.rules[rule.ID] = rule
	}
}

// TriggerAlert triggers an alert based on conditions
func (am *AlertManager) TriggerAlert(ruleID, source, message string, metadata map[string]interface{}) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	rule, exists := am.rules[ruleID]
	if !exists || !rule.Enabled {
		return fmt.Errorf("rule %s not found or disabled", ruleID)
	}

	// Check cooldown
	if lastTrigger, inCooldown := am.cooldowns[ruleID]; inCooldown {
		if time.Since(lastTrigger) < rule.Cooldown {
			return fmt.Errorf("rule %s is in cooldown", ruleID)
		}
	}

	// Check condition
	if !am.evaluateCondition(rule.Condition, rule.Threshold, metadata) {
		return nil // Condition not met
	}

	// Create alert
	alert := &Alert{
		ID:        fmt.Sprintf("alert_%d_%d", ruleID, time.Now().UnixNano()),
		Name:      rule.Name,
		Rule:      ruleID,
		Severity:  rule.Severity,
		Status:    AlertStatusActive,
		Message:   message,
		Source:    source,
		Category:  am.getCategoryFromRule(ruleID),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Count:     1,
		Metadata:  metadata,
	}

	// Check if this is a duplicate (debounce)
	if existingAlert, exists := am.debounce[ruleID]; exists {
		alert.Count = existingAlert.Count + 1
		alert.ID = existingAlert.ID // Keep same ID for the same alert
	}

	am.alerts[alert.ID] = alert
	am.debounce[ruleID] = alert
	am.cooldowns[ruleID] = time.Now()

	log.Printf("Alert triggered: %s - %s", alert.Name, alert.Message)

	// Execute alert actions
	go am.executeActions(rule.Actions, alert)

	return nil
}

// evaluateCondition evaluates alert condition
func (am *AlertManager) evaluateCondition(condition string, threshold float64, metadata map[string]interface{}) bool {
	switch condition {
	case "error_rate":
		if errorRate, ok := metadata["error_rate"].(float64); ok {
			return errorRate >= threshold
		}
	case "latency":
		if latency, ok := metadata["latency_ms"].(float64); ok {
			return latency >= threshold
		}
	case "resource_usage":
		if usage, ok := metadata["usage_percent"].(float64); ok {
			return usage >= threshold
		}
	case "provider_health":
		if health, ok := metadata["health_score"].(float64); ok {
			return health <= threshold
		}
	}
	return false
}

// getCategoryFromRule returns category based on rule ID
func (am *AlertManager) getCategoryFromRule(ruleID string) string {
	switch {
	case ruleID == "high_error_rate", ruleID == "low_success_rate":
		return "performance"
	case ruleID == "system_resources":
		return "infrastructure"
	case ruleID == "provider_health":
		return "availability"
	default:
		return "general"
	}
}

// executeActions executes alert actions
func (am *AlertManager) executeActions(actions []AlertAction, alert *Alert) {
	for _, action := range actions {
		if !action.Enabled {
			continue
		}

		switch action.Type {
		case "email":
			go am.sendEmailAlert(action.Config, alert)
		case "slack":
			go am.sendSlackAlert(action.Config, alert)
		case "webhook":
			go am.sendWebhookAlert(action.Config, alert)
		case "sms":
			go am.sendSMSAlert(action.Config, alert)
		case "phone_call":
			go am.sendPhoneCallAlert(action.Config, alert)
		case "pagerduty":
			go am.sendPagerDutyAlert(action.Config, alert)
		case "jira":
			go am.createJiraIssue(action.Config, alert)
		}
	}
}

// sendEmailAlert sends email alert
func (am *AlertManager) sendEmailAlert(config map[string]interface{}, alert *Alert) {
	template, _ := config["template"].(string)
	log.Printf("Sending email alert: %s to template %s", alert.Message, template)
}

// sendSlackAlert sends Slack alert
func (am *AlertManager) sendSlackAlert(config map[string]interface{}, alert *Alert) {
	channel, _ := config["channel"].(string)
	log.Printf("Sending Slack alert to %s: %s", channel, alert.Message)
}

// sendWebhookAlert sends webhook alert
func (am *AlertManager) sendWebhookAlert(config map[string]interface{}, alert *Alert) {
	url, _ := config["url"].(string)
	log.Printf("Sending webhook alert to %s: %s", url, alert.Message)
}

// sendSMSAlert sends SMS alert
func (am *AlertManager) sendSMSAlert(config map[string]interface{}, alert *Alert) {
	numbers, _ := config["numbers"].([]string)
	log.Printf("Sending SMS alert to %v: %s", numbers, alert.Message)
}

// sendPhoneCallAlert sends phone call alert
func (am *AlertManager) sendPhoneCallAlert(config map[string]interface{}, alert *Alert) {
	numbers, _ := config["numbers"].([]string)
	log.Printf("Making phone call alert to %v: %s", numbers, alert.Message)
}

// sendPagerDutyAlert sends PagerDuty alert
func (am *AlertManager) sendPagerDutyAlert(config map[string]interface{}, alert *Alert) {
	serviceKey, _ := config["service_key"].(string)
	log.Printf("Sending PagerDuty alert: %s (key: %s)", alert.Message, serviceKey)
}

// createJiraIssue creates JIRA issue
func (am *AlertManager) createJiraIssue(config map[string]interface{}, alert *Alert) {
	project, _ := config["project"].(string)
	issueType, _ := config["issue_type"].(string)
	log.Printf("Creating JIRA issue in %s (%s): %s", project, issueType, alert.Message)
}

// EscalateAlert escalates an alert to the next level
func (am *AlertManager) EscalateAlert(alertID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert %s not found", alertID)
	}

	rule := am.rules[alert.Rule]
	if rule.Escalation == nil || !rule.Escalation.Enabled {
		return fmt.Errorf("escalation not enabled for rule %s", alert.Rule)
	}

	currentLevel := am.getCurrentEscalationLevel(alertID)
	if currentLevel >= rule.Escalation.MaxEscalations {
		return fmt.Errorf("maximum escalation level reached for alert %s", alertID)
	}

	nextLevel := currentLevel + 1
	if nextLevel >= len(rule.Escalation.Levels) {
		nextLevel = len(rule.Escalation.Levels) - 1
	}

	escalationLevel := rule.Escalation.Levels[nextLevel]
	now := time.Now()

	// Update alert
	alert.Status = AlertStatusEscalated
	alert.EscalatedAt = &now
	alert.UpdatedAt = now

	am.escalations[alertID] = append(am.escalations[alertID], alert)

	log.Printf("Alert %s escalated to level %d: %s", alertID, nextLevel, escalationLevel.Name)

	// Execute escalation actions
	go am.executeActions(escalationLevel.Actions, alert)

	return nil
}

// getCurrentEscalationLevel gets current escalation level for an alert
func (am *AlertManager) getCurrentEscalationLevel(alertID string) int {
	escalations := am.escalations[alertID]
	return len(escalations)
}

// ResolveAlert resolves an alert
func (am *AlertManager) ResolveAlert(alertID string, resolution string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert %s not found", alertID)
	}

	now := time.Now()
	alert.Status = AlertStatusResolved
	alert.ResolvedAt = &now
	alert.UpdatedAt = now
	alert.Metadata["resolution"] = resolution

	log.Printf("Alert %s resolved: %s", alertID, resolution)

	// Clean up escalations
	delete(am.escalations, alertID)
	delete(am.debounce, alertID)

	return nil
}

// SuppressAlert suppresses an alert
func (am *AlertManager) SuppressAlert(alertID string, reason string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return fmt.Errorf("alert %s not found", alertID)
	}

	now := time.Now()
	alert.Status = AlertStatusSuppressed
	alert.UpdatedAt = now
	alert.Metadata["suppression_reason"] = reason

	log.Printf("Alert %s suppressed: %s", alertID, reason)

	return nil
}

// GetActiveAlerts returns all active alerts
func (am *AlertManager) GetActiveAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var activeAlerts []*Alert
	for _, alert := range am.alerts {
		if alert.Status == AlertStatusActive {
			activeAlerts = append(activeAlerts, alert)
		}
	}

	return activeAlerts
}

// GetAlerts returns all alerts with optional filtering
func (am *AlertManager) GetAlerts(severity AlertSeverity, category string) []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var filteredAlerts []*Alert
	for _, alert := range am.alerts {
		if (severity == "" || alert.Severity == severity) &&
			(category == "" || alert.Category == category) {
			filteredAlerts = append(filteredAlerts, alert)
		}
	}

	return filteredAlerts
}

// GetAlertMetrics returns comprehensive alert metrics
func (am *AlertManager) GetAlertMetrics() *AlertMetrics {
	am.mu.RLock()
	defer am.mu.RUnlock()

	metrics := &AlertMetrics{
		AlertsBySeverity: make(map[string]int),
		AlertsByCategory: make(map[string]int),
	}

	for _, alert := range am.alerts {
		metrics.TotalAlerts++

		switch alert.Status {
		case AlertStatusActive:
			metrics.ActiveAlerts++
		case AlertStatusEscalated:
			metrics.EscalatedAlerts++
		case AlertStatusSuppressed:
			metrics.SuppressedAlerts++
		}

		metrics.AlertsBySeverity[alert.Severity]++
		metrics.AlertsByCategory[alert.Category]++
	}

	return metrics
}

// alertProcessor runs background alert processing
func (am *AlertManager) alertProcessor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			am.processAlertAging()
		}
	}
}

// processAlertAging ages alerts and takes automatic actions
func (am *AlertManager) processAlertAging() {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()

	for _, alert := range am.alerts {
		if alert.Status != AlertStatusActive {
			continue
		}

		// Auto-escalate old alerts
		if alert.EscalatedAt == nil && now.Sub(alert.CreatedAt) > 30*time.Minute {
			rule := am.rules[alert.Rule]
			if rule.Escalation != nil && rule.Escalation.AutoEscalate {
				am.EscalateAlert(alert.ID)
			}
		}

		// Auto-resolve very old alerts (could be noise)
		if now.Sub(alert.CreatedAt) > 24*time.Hour {
			am.ResolveAlert(alert.ID, "auto-resolved after 24 hours")
		}
	}
}

// escalationProcessor handles escalation logic
func (am *AlertManager) escalationProcessor() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			am.checkPendingEscalations()
		}
	}
}

// checkPendingEscalations processes delayed escalations
func (am *AlertManager) checkPendingEscalations() {
	am.mu.RLock()
	defer am.mu.RUnlock()

	now := time.Now()

	for alertID, escalations := range am.escalations {
		rule := am.rules[am.alerts[alertID].Rule]
		if rule.Escalation == nil {
			continue
		}

		for i, escalation := range escalations {
			escalationLevel := rule.Escalation.Levels[i]
			if now.Sub(*escalation.EscalatedAt) >= escalationLevel.Delay {
				// Process next escalation level
				if i+1 < len(rule.Escalation.Levels) {
					nextLevel := rule.Escalation.Levels[i+1]
					log.Printf("Processing delayed escalation for alert %s to level %d", alertID, i+2)
					go am.executeActions(nextLevel.Actions, escalation)
				}
			}
		}
	}
}

// cooldownManager manages alert cooldowns
func (am *AlertManager) cooldownManager() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			am.cleanupCooldowns()
		}
	}
}

// cleanupCooldowns removes expired cooldowns
func (am *AlertManager) cleanupCooldowns() {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()
	for ruleID, lastTrigger := range am.cooldowns {
		rule := am.rules[ruleID]
		if rule.Enabled && now.Sub(lastTrigger) > rule.Cooldown {
			delete(am.cooldowns, ruleID)
		}
	}
}

// UpdateRule updates an alert rule
func (am *AlertManager) UpdateRule(rule *AlertRule) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if rule.ID == "" {
		return fmt.Errorf("rule ID cannot be empty")
	}

	am.rules[rule.ID] = rule
	log.Printf("Updated alert rule: %s", rule.ID)

	return nil
}

// DeleteRule deletes an alert rule
func (am *AlertManager) DeleteRule(ruleID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	delete(am.rules, ruleID)
	delete(am.cooldowns, ruleID)

	log.Printf("Deleted alert rule: %s", ruleID)

	return nil
}

// GetRules returns all alert rules
func (am *AlertManager) GetRules() []*AlertRule {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var rules []*AlertRule
	for _, rule := range am.rules {
		rules = append(rules, rule)
	}

	return rules
}
