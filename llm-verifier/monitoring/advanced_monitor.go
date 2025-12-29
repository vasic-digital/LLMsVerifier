package monitoring

import (
	"fmt"
	"sync"
	"time"
)

// AdvancedMonitor provides advanced monitoring and alerting capabilities
type AdvancedMonitor struct {
	mu         sync.RWMutex
	alerts     []MonitorAlert
	thresholds map[string]Threshold
	metrics    map[string][]MetricPoint
}

// MonitorAlert represents a monitoring alert
type MonitorAlert struct {
	ID        string
	Type      string
	Message   string
	Severity  string
	Value     float64
	Threshold float64
	Timestamp time.Time
	Resolved  bool
}

// Threshold defines alert thresholds
type Threshold struct {
	Warning  float64
	Critical float64
	Enabled  bool
}

// MetricPoint represents a single metric measurement
type MetricPoint struct {
	Value     float64
	Timestamp time.Time
}

// NewAdvancedMonitor creates a new advanced monitor
func NewAdvancedMonitor() *AdvancedMonitor {
	return &AdvancedMonitor{
		alerts:     []MonitorAlert{},
		thresholds: make(map[string]Threshold),
		metrics:    make(map[string][]MetricPoint),
	}
}

// SetThreshold sets an alert threshold for a metric
func (am *AdvancedMonitor) SetThreshold(metric string, warning, critical float64) {
	am.mu.Lock()
	defer am.mu.Unlock()

	am.thresholds[metric] = Threshold{
		Warning:  warning,
		Critical: critical,
		Enabled:  true,
	}
}

// RecordMetric records a metric value and checks for alerts
func (am *AdvancedMonitor) RecordMetric(name string, value float64) {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Record the metric
	point := MetricPoint{
		Value:     value,
		Timestamp: time.Now(),
	}

	am.metrics[name] = append(am.metrics[name], point)

	// Keep only last 1000 points
	if len(am.metrics[name]) > 1000 {
		am.metrics[name] = am.metrics[name][1:]
	}

	// Check for alerts
	if threshold, exists := am.thresholds[name]; exists && threshold.Enabled {
		am.checkThreshold(name, value, threshold)
	}
}

// checkThreshold checks if a value exceeds thresholds
func (am *AdvancedMonitor) checkThreshold(metric string, value float64, threshold Threshold) {
	var severity string
	var message string

	if value >= threshold.Critical {
		severity = "critical"
		message = fmt.Sprintf("%s exceeded critical threshold: %.2f >= %.2f", metric, value, threshold.Critical)
	} else if value >= threshold.Warning {
		severity = "warning"
		message = fmt.Sprintf("%s exceeded warning threshold: %.2f >= %.2f", metric, value, threshold.Warning)
	} else {
		return // No alert needed
	}

	alert := MonitorAlert{
		ID:        fmt.Sprintf("%s-%d", metric, time.Now().Unix()),
		Type:      "threshold_exceeded",
		Message:   message,
		Severity:  severity,
		Value:     value,
		Threshold: threshold.Critical,
		Timestamp: time.Now(),
		Resolved:  false,
	}

	am.alerts = append(am.alerts, alert)
}

// GetActiveAlerts returns all active (unresolved) alerts
func (am *AdvancedMonitor) GetActiveAlerts() []MonitorAlert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	var active []MonitorAlert
	for _, alert := range am.alerts {
		if !alert.Resolved {
			active = append(active, alert)
		}
	}
	return active
}

// GetMetrics returns recent metrics for a given name
func (am *AdvancedMonitor) GetMetrics(name string, limit int) []MetricPoint {
	am.mu.RLock()
	defer am.mu.RUnlock()

	points := am.metrics[name]
	if len(points) == 0 {
		return []MetricPoint{}
	}

	// Return last 'limit' points
	start := len(points) - limit
	if start < 0 {
		start = 0
	}

	return points[start:]
}

// ResolveAlert marks an alert as resolved
func (am *AdvancedMonitor) ResolveAlert(alertID string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	for i := range am.alerts {
		if am.alerts[i].ID == alertID {
			am.alerts[i].Resolved = true
			break
		}
	}
}

// GetMetricStats returns statistical information about a metric
func (am *AdvancedMonitor) GetMetricStats(name string) map[string]interface{} {
	points := am.GetMetrics(name, 1000)

	if len(points) == 0 {
		return map[string]interface{}{
			"count":   0,
			"current": 0.0,
		}
	}

	var sum, min, max float64
	min = points[0].Value
	max = points[0].Value

	for _, point := range points {
		sum += point.Value
		if point.Value < min {
			min = point.Value
		}
		if point.Value > max {
			max = point.Value
		}
	}

	avg := sum / float64(len(points))
	current := points[len(points)-1].Value

	return map[string]interface{}{
		"count":   len(points),
		"current": current,
		"average": avg,
		"minimum": min,
		"maximum": max,
		"sum":     sum,
	}
}

// HealthCheck performs a comprehensive health check
func (am *AdvancedMonitor) HealthCheck() map[string]interface{} {
	stats := map[string]interface{}{
		"timestamp": time.Now(),
		"status":    "healthy",
	}

	// Check for critical alerts
	activeAlerts := am.GetActiveAlerts()
	criticalCount := 0
	for _, alert := range activeAlerts {
		if alert.Severity == "critical" {
			criticalCount++
		}
	}

	if criticalCount > 0 {
		stats["status"] = "critical"
		stats["critical_alerts"] = criticalCount
	}

	stats["active_alerts"] = len(activeAlerts)
	stats["monitored_metrics"] = len(am.metrics)

	return stats
}
