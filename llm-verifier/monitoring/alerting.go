package monitoring

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// AlertSeverity represents alert severity levels
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertStatus represents the status of an alert
type AlertStatus string

const (
	AlertStatusActive      AlertStatus = "active"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusResolved    AlertStatus = "resolved"
)

// Alert represents an alert
type Alert struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Rule        string        `json:"rule"`
	Severity    AlertSeverity `json:"severity"`
	Message     string        `json:"message"`
	Time        string        `json:"time"`
	Active      bool          `json:"active"`
	Status      AlertStatus   `json:"status"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	ResolvedAt  *time.Time    `json:"resolved_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AlertRule defines conditions for triggering alerts
type AlertRule struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Condition   string        `json:"condition"`
	Threshold   float64       `json:"threshold"`
	Severity    AlertSeverity `json:"severity"`
	Enabled     bool          `json:"enabled"`
	CooldownPeriod time.Duration `json:"cooldown_period"`
	LastTriggered  time.Time     `json:"last_triggered"`
}

// AlertManager manages the alerting system
type AlertManager struct {
	metricsTracker *MetricsTracker
	alerts         map[string]*Alert
	rules          map[string]*AlertRule
	mu             sync.RWMutex
	listeners      []AlertListener
}

// AlertListener is called when an alert is triggered or resolved
type AlertListener func(alert *Alert, eventType string)

// CriticalMetricsTracker tracks critical system metrics for alerting
type CriticalMetricsTracker struct {
	collector *MetricsCollector
}

// GetPerformanceReport returns a performance report based on actual metrics
func (cmt *CriticalMetricsTracker) GetPerformanceReport(duration interface{}) map[string]any {
	if cmt.collector == nil {
		return map[string]any{
			"status":  "unavailable",
			"message": "Metrics collector not initialized",
		}
	}

	// Get all metrics from collector
	allMetrics := cmt.collector.GetAllMetrics()

	// Extract relevant metrics
	totalRequests := float64(0)
	errorCount := float64(0)
	avgLatency := float64(0)

	for _, m := range allMetrics {
		switch m.Name {
		case "http_requests_total":
			totalRequests += m.Value
		case "http_errors_total":
			errorCount += m.Value
		case "http_request_duration_seconds":
			avgLatency = m.Value * 1000 // Convert to milliseconds
		}
	}

	return map[string]any{
		"status":             "ok",
		"timestamp":          time.Now(),
		"total_requests":     totalRequests,
		"error_count":        errorCount,
		"average_latency_ms": avgLatency,
		"metrics_count":      len(allMetrics),
	}
}

// NewCriticalMetricsTracker creates a new critical metrics tracker
func NewCriticalMetricsTracker() *CriticalMetricsTracker {
	return &CriticalMetricsTracker{
		collector: NewMetricsCollector(),
	}
}

// NewAlertManager creates a new alert manager with real implementation
func NewAlertManager(metricsTracker *MetricsTracker) *AlertManager {
	am := &AlertManager{
		metricsTracker: metricsTracker,
		alerts:         make(map[string]*Alert),
		rules:          make(map[string]*AlertRule),
		listeners:      []AlertListener{},
	}

	// Register default alert rules
	am.registerDefaultRules()

	return am
}

// registerDefaultRules sets up default alerting rules
func (am *AlertManager) registerDefaultRules() {
	// High error rate rule
	am.AddRule(&AlertRule{
		ID:          "high_error_rate",
		Name:        "High Error Rate",
		Description: "Triggers when API error rate exceeds threshold",
		Condition:   "error_rate > threshold",
		Threshold:   0.1, // 10% error rate
		Severity:    AlertSeverityError,
		Enabled:     true,
		CooldownPeriod: 5 * time.Minute,
	})

	// High response time rule
	am.AddRule(&AlertRule{
		ID:          "high_response_time",
		Name:        "High Response Time",
		Description: "Triggers when average response time exceeds threshold",
		Condition:   "avg_response_time > threshold",
		Threshold:   5000, // 5 seconds in ms
		Severity:    AlertSeverityWarning,
		Enabled:     true,
		CooldownPeriod: 5 * time.Minute,
	})

	// Scheduler failure rate rule
	am.AddRule(&AlertRule{
		ID:          "scheduler_failures",
		Name:        "Scheduler Job Failures",
		Description: "Triggers when scheduler failure rate is high",
		Condition:   "scheduler_failure_rate > threshold",
		Threshold:   0.2, // 20% failure rate
		Severity:    AlertSeverityError,
		Enabled:     true,
		CooldownPeriod: 10 * time.Minute,
	})
}

// GetActiveAlerts returns all active alerts
func (am *AlertManager) GetActiveAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	activeAlerts := []*Alert{}
	for _, alert := range am.alerts {
		if alert.Active && alert.Status == AlertStatusActive {
			activeAlerts = append(activeAlerts, alert)
		}
	}
	return activeAlerts
}

// GetAllAlerts returns all alerts (active and resolved)
func (am *AlertManager) GetAllAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	allAlerts := make([]*Alert, 0, len(am.alerts))
	for _, alert := range am.alerts {
		allAlerts = append(allAlerts, alert)
	}
	return allAlerts
}

// GetAlertsByStatus returns alerts filtered by status
func (am *AlertManager) GetAlertsByStatus(status AlertStatus) []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	filtered := []*Alert{}
	for _, alert := range am.alerts {
		if alert.Status == status {
			filtered = append(filtered, alert)
		}
	}
	return filtered
}

// GetAlertsBySeverity returns alerts filtered by severity
func (am *AlertManager) GetAlertsBySeverity(severity AlertSeverity) []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	filtered := []*Alert{}
	for _, alert := range am.alerts {
		if alert.Severity == severity && alert.Active {
			filtered = append(filtered, alert)
		}
	}
	return filtered
}

// CreateAlert creates a new alert
func (am *AlertManager) CreateAlert(name, rule, message string, severity AlertSeverity) *Alert {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()
	alert := &Alert{
		ID:        uuid.New().String(),
		Name:      name,
		Rule:      rule,
		Severity:  severity,
		Message:   message,
		Time:      now.Format(time.RFC3339),
		Active:    true,
		Status:    AlertStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  make(map[string]interface{}),
	}

	am.alerts[alert.ID] = alert

	// Notify listeners
	am.notifyListeners(alert, "created")

	return alert
}

// AcknowledgeAlert marks an alert as acknowledged
func (am *AlertManager) AcknowledgeAlert(alertID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return nil
	}

	now := time.Now()
	alert.Status = AlertStatusAcknowledged
	alert.AcknowledgedAt = &now
	alert.UpdatedAt = now

	am.notifyListeners(alert, "acknowledged")
	return nil
}

// ResolveAlert marks an alert as resolved
func (am *AlertManager) ResolveAlert(alertID string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert, exists := am.alerts[alertID]
	if !exists {
		return nil
	}

	now := time.Now()
	alert.Status = AlertStatusResolved
	alert.Active = false
	alert.ResolvedAt = &now
	alert.UpdatedAt = now

	am.notifyListeners(alert, "resolved")
	return nil
}

// AddRule adds a new alert rule
func (am *AlertManager) AddRule(rule *AlertRule) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.rules[rule.ID] = rule
}

// GetRules returns all alert rules
func (am *AlertManager) GetRules() []*AlertRule {
	am.mu.RLock()
	defer am.mu.RUnlock()

	rules := make([]*AlertRule, 0, len(am.rules))
	for _, rule := range am.rules {
		rules = append(rules, rule)
	}
	return rules
}

// CheckRules evaluates all rules against current metrics
func (am *AlertManager) CheckRules() {
	if am.metricsTracker == nil {
		return
	}

	am.mu.Lock()
	defer am.mu.Unlock()

	apiMetrics := am.metricsTracker.GetAPIMetrics()
	schedStats := am.metricsTracker.GetSchedulerStats()

	for _, rule := range am.rules {
		if !rule.Enabled {
			continue
		}

		// Check cooldown
		if time.Since(rule.LastTriggered) < rule.CooldownPeriod {
			continue
		}

		triggered := false
		var message string

		switch rule.ID {
		case "high_error_rate":
			if apiMetrics.ErrorRate > rule.Threshold {
				triggered = true
				message = "API error rate exceeded threshold: " +
					string(rune(int(apiMetrics.ErrorRate*100))) + "%"
			}
		case "high_response_time":
			if apiMetrics.AverageResponseTime.Milliseconds() > int64(rule.Threshold) {
				triggered = true
				message = "Average response time exceeded threshold"
			}
		case "scheduler_failures":
			totalJobs := schedStats.CompletedJobs + schedStats.FailedJobs
			if totalJobs > 0 {
				failureRate := float64(schedStats.FailedJobs) / float64(totalJobs)
				if failureRate > rule.Threshold {
					triggered = true
					message = "Scheduler failure rate exceeded threshold"
				}
			}
		}

		if triggered {
			rule.LastTriggered = time.Now()
			// Create alert (using internal version without lock)
			am.createAlertInternal(rule.Name, rule.ID, message, rule.Severity)
		}
	}
}

// createAlertInternal creates an alert without acquiring lock (caller must hold lock)
func (am *AlertManager) createAlertInternal(name, rule, message string, severity AlertSeverity) *Alert {
	now := time.Now()
	alert := &Alert{
		ID:        uuid.New().String(),
		Name:      name,
		Rule:      rule,
		Severity:  severity,
		Message:   message,
		Time:      now.Format(time.RFC3339),
		Active:    true,
		Status:    AlertStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
		Metadata:  make(map[string]interface{}),
	}

	am.alerts[alert.ID] = alert
	return alert
}

// AddListener adds a listener for alert events
func (am *AlertManager) AddListener(listener AlertListener) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.listeners = append(am.listeners, listener)
}

// notifyListeners calls all registered listeners
func (am *AlertManager) notifyListeners(alert *Alert, eventType string) {
	for _, listener := range am.listeners {
		go listener(alert, eventType)
	}
}

// GetAlertCount returns count of active alerts by severity
func (am *AlertManager) GetAlertCount() map[AlertSeverity]int {
	am.mu.RLock()
	defer am.mu.RUnlock()

	counts := map[AlertSeverity]int{
		AlertSeverityInfo:     0,
		AlertSeverityWarning:  0,
		AlertSeverityError:    0,
		AlertSeverityCritical: 0,
	}

	for _, alert := range am.alerts {
		if alert.Active {
			counts[alert.Severity]++
		}
	}

	return counts
}

// CleanupOldAlerts removes resolved alerts older than the specified duration
func (am *AlertManager) CleanupOldAlerts(maxAge time.Duration) int {
	am.mu.Lock()
	defer am.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, alert := range am.alerts {
		if alert.Status == AlertStatusResolved && alert.ResolvedAt != nil && alert.ResolvedAt.Before(cutoff) {
			delete(am.alerts, id)
			removed++
		}
	}

	return removed
}
