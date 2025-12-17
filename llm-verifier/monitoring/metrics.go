package monitoring

import (
	"fmt"
	"sync"
	"time"
)

// MetricType represents different types of metrics
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

// Metric represents a single metric measurement
type Metric struct {
	Name      string            `json:"name"`
	Type      MetricType        `json:"type"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// MetricsCollector collects and manages metrics
type MetricsCollector struct {
	metrics    map[string][]Metric
	mu         sync.RWMutex
	maxSamples int
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(maxSamples int) *MetricsCollector {
	return &MetricsCollector{
		metrics:    make(map[string][]Metric),
		maxSamples: maxSamples,
	}
}

// RecordMetric records a new metric measurement
func (mc *MetricsCollector) RecordMetric(name string, value float64, labels map[string]string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metric := Metric{
		Name:      name,
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now(),
	}

	// Determine metric type based on name
	switch {
	case name == "requests_total" || name == "errors_total":
		metric.Type = MetricTypeCounter
	case name == "active_connections" || name == "memory_usage":
		metric.Type = MetricTypeGauge
	case name == "request_duration" || name == "response_time":
		metric.Type = MetricTypeHistogram
	default:
		metric.Type = MetricTypeGauge
	}

	// Add to samples
	samples := mc.metrics[name]
	samples = append(samples, metric)

	// Maintain maximum samples
	if len(samples) > mc.maxSamples {
		samples = samples[len(samples)-mc.maxSamples:]
	}

	mc.metrics[name] = samples
}

// GetMetrics returns all metrics for a given name
func (mc *MetricsCollector) GetMetrics(name string) []Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	samples := mc.metrics[name]
	result := make([]Metric, len(samples))
	copy(result, samples)
	return result
}

// GetAllMetrics returns all collected metrics
func (mc *MetricsCollector) GetAllMetrics() map[string][]Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make(map[string][]Metric)
	for name, samples := range mc.metrics {
		result[name] = make([]Metric, len(samples))
		copy(result[name], samples)
	}
	return result
}

// GetLatestMetric returns the most recent metric for a given name
func (mc *MetricsCollector) GetLatestMetric(name string) *Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	samples := mc.metrics[name]
	if len(samples) == 0 {
		return nil
	}

	latest := samples[len(samples)-1]
	return &latest
}

// CalculateStats calculates statistics for a metric over a time window
func (mc *MetricsCollector) CalculateStats(name string, window time.Duration) MetricStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	samples := mc.metrics[name]
	if len(samples) == 0 {
		return MetricStats{}
	}

	cutoff := time.Now().Add(-window)
	var relevantSamples []Metric

	for _, sample := range samples {
		if sample.Timestamp.After(cutoff) {
			relevantSamples = append(relevantSamples, sample)
		}
	}

	if len(relevantSamples) == 0 {
		return MetricStats{}
	}

	return calculateStats(relevantSamples)
}

// MetricStats represents statistical information about metrics
type MetricStats struct {
	Count      int       `json:"count"`
	Min        float64   `json:"min"`
	Max        float64   `json:"max"`
	Avg        float64   `json:"avg"`
	P50        float64   `json:"p50"`
	P95        float64   `json:"p95"`
	P99        float64   `json:"p99"`
	Latest     float64   `json:"latest"`
	LatestTime time.Time `json:"latest_time"`
}

// calculateStats calculates statistics from a slice of metrics
func calculateStats(samples []Metric) MetricStats {
	if len(samples) == 0 {
		return MetricStats{}
	}

	stats := MetricStats{
		Count:      len(samples),
		Min:        samples[0].Value,
		Max:        samples[0].Value,
		Latest:     samples[len(samples)-1].Value,
		LatestTime: samples[len(samples)-1].Timestamp,
	}

	sum := 0.0
	values := make([]float64, len(samples))

	for i, sample := range samples {
		value := sample.Value
		values[i] = value
		sum += value

		if value < stats.Min {
			stats.Min = value
		}
		if value > stats.Max {
			stats.Max = value
		}
	}

	stats.Avg = sum / float64(len(samples))

	// Calculate percentiles
	stats.P50 = percentile(values, 50)
	stats.P95 = percentile(values, 95)
	stats.P99 = percentile(values, 99)

	return stats
}

// percentile calculates the p-th percentile of a slice of values
func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}

	// Sort values
	sorted := make([]float64, len(values))
	copy(sorted, values)

	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	index := (p / 100) * float64(len(sorted)-1)
	lower := int(index)
	upper := lower + 1

	if upper >= len(sorted) {
		return sorted[lower]
	}

	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

// CriticalMetricsTracker tracks critical performance metrics
type CriticalMetricsTracker struct {
	collector *MetricsCollector
	alerts    []Alert
	mu        sync.RWMutex
}

// NewCriticalMetricsTracker creates a new critical metrics tracker
func NewCriticalMetricsTracker(collector *MetricsCollector) *CriticalMetricsTracker {
	return &CriticalMetricsTracker{
		collector: collector,
		alerts:    []Alert{},
	}
}

// Alert represents an alert condition
type Alert struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Message     string         `json:"message,omitempty"`
	Metric      string         `json:"metric"`
	Condition   AlertCondition `json:"condition"`
	Threshold   float64        `json:"threshold"`
	Severity    AlertSeverity  `json:"severity"`
	Active      bool           `json:"active"`
	TriggeredAt *time.Time     `json:"triggered_at,omitempty"`
	ResolvedAt  *time.Time     `json:"resolved_at,omitempty"`
}

// AlertCondition represents the condition for triggering an alert
type AlertCondition string

const (
	AlertConditionGreaterThan AlertCondition = ">"
	AlertConditionLessThan    AlertCondition = "<"
	AlertConditionEqual       AlertCondition = "="
)

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityCritical AlertSeverity = "critical"
)

// TrackTTFT tracks Time to First Token
func (cmt *CriticalMetricsTracker) TrackTTFT(providerID string, ttft time.Duration) {
	cmt.collector.RecordMetric("ttft_seconds", ttft.Seconds(), map[string]string{
		"provider": providerID,
	})
}

// TrackLatency tracks request latency
func (cmt *CriticalMetricsTracker) TrackLatency(providerID string, latency time.Duration) {
	cmt.collector.RecordMetric("request_latency_seconds", latency.Seconds(), map[string]string{
		"provider": providerID,
	})
}

// TrackError tracks errors
func (cmt *CriticalMetricsTracker) TrackError(providerID string, errorType string) {
	cmt.collector.RecordMetric("errors_total", 1, map[string]string{
		"provider": providerID,
		"type":     errorType,
	})
}

// TrackThroughput tracks requests per second
func (cmt *CriticalMetricsTracker) TrackThroughput(providerID string, requests int) {
	cmt.collector.RecordMetric("requests_total", float64(requests), map[string]string{
		"provider": providerID,
	})
}

// AddAlert adds an alert condition
func (cmt *CriticalMetricsTracker) AddAlert(alert Alert) {
	cmt.mu.Lock()
	defer cmt.mu.Unlock()

	alert.ID = fmt.Sprintf("alert_%d", time.Now().UnixNano())
	cmt.alerts = append(cmt.alerts, alert)
}

// CheckAlerts checks all alert conditions
func (cmt *CriticalMetricsTracker) CheckAlerts() []Alert {
	cmt.mu.RLock()
	defer cmt.mu.RUnlock()

	var triggered []Alert

	for _, alert := range cmt.alerts {
		if alert.Active {
			continue // Already triggered
		}

		latest := cmt.collector.GetLatestMetric(alert.Metric)
		if latest == nil {
			continue
		}

		isTriggered := cmt.checkAlertCondition(alert, latest.Value)
		if isTriggered {
			alert.Active = true
			alert.TriggeredAt = &time.Time{}
			*alert.TriggeredAt = time.Now()
			triggered = append(triggered, alert)
		}
	}

	return triggered
}

// checkAlertCondition checks if an alert condition is met
func (cmt *CriticalMetricsTracker) checkAlertCondition(alert Alert, value float64) bool {
	switch alert.Condition {
	case AlertConditionGreaterThan:
		return value > alert.Threshold
	case AlertConditionLessThan:
		return value < alert.Threshold
	case AlertConditionEqual:
		return value == alert.Threshold
	default:
		return false
	}
}

// ResolveAlert resolves an alert
func (cmt *CriticalMetricsTracker) ResolveAlert(alertID string) {
	cmt.mu.Lock()
	defer cmt.mu.Unlock()

	for i, alert := range cmt.alerts {
		if alert.ID == alertID {
			cmt.alerts[i].Active = false
			now := time.Now()
			cmt.alerts[i].ResolvedAt = &now
			break
		}
	}
}

// GetAlerts returns all alerts
func (cmt *CriticalMetricsTracker) GetAlerts() []Alert {
	cmt.mu.RLock()
	defer cmt.mu.RUnlock()

	alerts := make([]Alert, len(cmt.alerts))
	copy(alerts, cmt.alerts)
	return alerts
}

// GetActiveAlerts returns currently active alerts
func (cmt *CriticalMetricsTracker) GetActiveAlerts() []Alert {
	cmt.mu.RLock()
	defer cmt.mu.RUnlock()

	var active []Alert
	for _, alert := range cmt.alerts {
		if alert.Active {
			active = append(active, alert)
		}
	}
	return active
}

// GetPerformanceReport generates a performance report
func (cmt *CriticalMetricsTracker) GetPerformanceReport(window time.Duration) PerformanceReport {
	report := PerformanceReport{
		GeneratedAt: time.Now(),
		Window:      window,
		Metrics:     make(map[string]MetricStats),
	}

	// Key metrics to report
	metrics := []string{
		"ttft_seconds",
		"request_latency_seconds",
		"errors_total",
		"requests_total",
	}

	for _, metric := range metrics {
		stats := cmt.collector.CalculateStats(metric, window)
		report.Metrics[metric] = stats
	}

	// Calculate overall health score
	report.HealthScore = cmt.calculateHealthScore(report.Metrics)

	return report
}

// PerformanceReport represents a performance report
type PerformanceReport struct {
	GeneratedAt time.Time              `json:"generated_at"`
	Window      time.Duration          `json:"window"`
	Metrics     map[string]MetricStats `json:"metrics"`
	HealthScore float64                `json:"health_score"`
	Alerts      []Alert                `json:"alerts,omitempty"`
}

// calculateHealthScore calculates an overall health score from metrics
func (cmt *CriticalMetricsTracker) calculateHealthScore(metrics map[string]MetricStats) float64 {
	score := 100.0

	// TTFT score (lower is better)
	if ttftStats, exists := metrics["ttft_seconds"]; exists && ttftStats.Count > 0 {
		avgTTFT := ttftStats.Avg
		if avgTTFT > 5.0 { // More than 5 seconds is bad
			score -= 20
		} else if avgTTFT > 2.0 { // More than 2 seconds is concerning
			score -= 10
		}
	}

	// Latency score (lower is better)
	if latencyStats, exists := metrics["request_latency_seconds"]; exists && latencyStats.Count > 0 {
		avgLatency := latencyStats.Avg
		if avgLatency > 10.0 { // More than 10 seconds is bad
			score -= 30
		} else if avgLatency > 5.0 { // More than 5 seconds is concerning
			score -= 15
		}
	}

	// Error rate score (lower is better)
	if errorStats, exists := metrics["errors_total"]; exists && errorStats.Count > 0 {
		if requestStats, reqExists := metrics["requests_total"]; reqExists && requestStats.Count > 0 {
			errorRate := errorStats.Latest / requestStats.Latest
			if errorRate > 0.1 { // More than 10% error rate
				score -= 25
			} else if errorRate > 0.05 { // More than 5% error rate
				score -= 10
			}
		}
	}

	if score < 0 {
		score = 0
	}

	return score
}
