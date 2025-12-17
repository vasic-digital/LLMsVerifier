package monitoring

import (
	"sync"
	"time"
)

// MetricType represents different types of metrics
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
)

// Metric represents a single metric
type Metric struct {
	Name        string            `json:"name"`
	Type        MetricType        `json:"type"`
	Value       float64           `json:"value"`
	Labels      map[string]string `json:"labels,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
	Description string            `json:"description,omitempty"`
}

// MetricsCollector collects and manages metrics
type MetricsCollector struct {
	metrics map[string]*Metric
	mu      sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]*Metric),
	}
}

// RecordCounter increments a counter metric
func (mc *MetricsCollector) RecordCounter(name string, value float64, labels map[string]string, description string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := mc.getMetricKey(name, labels)
	if metric, exists := mc.metrics[key]; exists {
		metric.Value += value
		metric.Timestamp = time.Now()
	} else {
		mc.metrics[key] = &Metric{
			Name:        name,
			Type:        MetricTypeCounter,
			Value:       value,
			Labels:      labels,
			Timestamp:   time.Now(),
			Description: description,
		}
	}
}

// RecordGauge sets a gauge metric value
func (mc *MetricsCollector) RecordGauge(name string, value float64, labels map[string]string, description string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := mc.getMetricKey(name, labels)
	mc.metrics[key] = &Metric{
		Name:        name,
		Type:        MetricTypeGauge,
		Value:       value,
		Labels:      labels,
		Timestamp:   time.Now(),
		Description: description,
	}
}

// RecordHistogram records a histogram observation
func (mc *MetricsCollector) RecordHistogram(name string, value float64, labels map[string]string, description string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	key := mc.getMetricKey(name, labels)
	// For histogram, we store the latest value
	// In a real implementation, you'd maintain buckets and percentiles
	mc.metrics[key] = &Metric{
		Name:        name,
		Type:        MetricTypeHistogram,
		Value:       value,
		Labels:      labels,
		Timestamp:   time.Now(),
		Description: description,
	}
}

// GetMetric retrieves a metric by name and labels
func (mc *MetricsCollector) GetMetric(name string, labels map[string]string) (*Metric, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	key := mc.getMetricKey(name, labels)
	metric, exists := mc.metrics[key]
	return metric, exists
}

// GetAllMetrics returns all collected metrics
func (mc *MetricsCollector) GetAllMetrics() []*Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metrics := make([]*Metric, 0, len(mc.metrics))
	for _, metric := range mc.metrics {
		metrics = append(metrics, metric)
	}
	return metrics
}

// GetMetricsByType returns metrics filtered by type
func (mc *MetricsCollector) GetMetricsByType(metricType MetricType) []*Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var metrics []*Metric
	for _, metric := range mc.metrics {
		if metric.Type == metricType {
			metrics = append(metrics, metric)
		}
	}
	return metrics
}

// GetMetricsByName returns metrics filtered by name
func (mc *MetricsCollector) GetMetricsByName(name string) []*Metric {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var metrics []*Metric
	for _, metric := range mc.metrics {
		if metric.Name == name {
			metrics = append(metrics, metric)
		}
	}
	return metrics
}

// ClearMetrics removes all metrics
func (mc *MetricsCollector) ClearMetrics() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.metrics = make(map[string]*Metric)
}

// getMetricKey generates a unique key for a metric based on name and labels
func (mc *MetricsCollector) getMetricKey(name string, labels map[string]string) string {
	key := name
	if len(labels) > 0 {
		key += "{"
		for k, v := range labels {
			key += k + "=" + v + ","
		}
		key = key[:len(key)-1] + "}" // Remove last comma and add closing brace
	}
	return key
}

// PerformanceMonitor tracks performance metrics for operations
type PerformanceMonitor struct {
	collector *MetricsCollector
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		collector: NewMetricsCollector(),
	}
}

// RecordOperation records the duration of an operation
func (pm *PerformanceMonitor) RecordOperation(operation string, duration time.Duration, labels map[string]string) {
	// Record as histogram for duration tracking
	pm.collector.RecordHistogram("operation_duration", duration.Seconds(), labels, "Duration of operations in seconds")

	// Record count
	pm.collector.RecordCounter("operation_count", 1, labels, "Number of operations performed")
}

// RecordError records an error occurrence
func (pm *PerformanceMonitor) RecordError(operation string, err error, labels map[string]string) {
	errorLabels := make(map[string]string)
	for k, v := range labels {
		errorLabels[k] = v
	}
	errorLabels["error_type"] = err.Error()

	pm.collector.RecordCounter("operation_errors", 1, errorLabels, "Number of operation errors")
}

// GetMetrics returns all performance metrics
func (pm *PerformanceMonitor) GetMetrics() map[string]any {
	metrics := pm.collector.GetAllMetrics()

	result := make(map[string]any)
	for _, metric := range metrics {
		result[metric.Name] = map[string]any{
			"type":      metric.Type,
			"value":     metric.Value,
			"labels":    metric.Labels,
			"timestamp": metric.Timestamp,
		}
	}
	return result
}

// LLMVerifierMetrics provides metrics specific to LLM verification operations
type LLMVerifierMetrics struct {
	performanceMonitor *PerformanceMonitor
	collector          *MetricsCollector
}

// NewLLMVerifierMetrics creates metrics for LLM verifier operations
func NewLLMVerifierMetrics() *LLMVerifierMetrics {
	return &LLMVerifierMetrics{
		performanceMonitor: NewPerformanceMonitor(),
		collector:          NewMetricsCollector(),
	}
}

// RecordVerification records a verification operation
func (lvm *LLMVerifierMetrics) RecordVerification(modelName, provider string, duration time.Duration, success bool) {
	labels := map[string]string{
		"model":    modelName,
		"provider": provider,
	}

	lvm.performanceMonitor.RecordOperation("llm_verification", duration, labels)

	if success {
		lvm.collector.RecordCounter("verification_success", 1, labels, "Successful verifications")
	} else {
		lvm.collector.RecordCounter("verification_failure", 1, labels, "Failed verifications")
	}

	// Record score as gauge (would be set separately)
}

// RecordModelScore records the score for a model
func (lvm *LLMVerifierMetrics) RecordModelScore(modelName, provider string, score float64) {
	labels := map[string]string{
		"model":    modelName,
		"provider": provider,
	}

	lvm.collector.RecordGauge("model_score", score, labels, "Model performance score (0-100)")
}

// RecordAPICall records an API call to an LLM provider
func (lvm *LLMVerifierMetrics) RecordAPICall(provider string, duration time.Duration, success bool, tokensUsed int) {
	labels := map[string]string{
		"provider": provider,
	}

	lvm.performanceMonitor.RecordOperation("api_call", duration, labels)

	if success {
		lvm.collector.RecordCounter("api_call_success", 1, labels, "Successful API calls")
		lvm.collector.RecordCounter("tokens_used", float64(tokensUsed), labels, "Total tokens consumed")
	} else {
		lvm.collector.RecordCounter("api_call_failure", 1, labels, "Failed API calls")
	}
}

// GetSummary returns a summary of LLM verifier metrics
func (lvm *LLMVerifierMetrics) GetSummary() map[string]any {
	summary := make(map[string]any)

	// Get performance metrics
	perfMetrics := lvm.performanceMonitor.GetMetrics()
	summary["performance"] = perfMetrics

	// Calculate totals
	totalVerifications := 0.0
	totalSuccesses := 0.0
	totalFailures := 0.0

	for _, metric := range lvm.collector.GetAllMetrics() {
		switch metric.Name {
		case "verification_success":
			totalSuccesses += metric.Value
		case "verification_failure":
			totalFailures += metric.Value
		}
	}

	totalVerifications = totalSuccesses + totalFailures
	successRate := 0.0
	if totalVerifications > 0 {
		successRate = (totalSuccesses / totalVerifications) * 100
	}

	summary["summary"] = map[string]any{
		"total_verifications": totalVerifications,
		"success_rate":        successRate,
		"total_successes":     totalSuccesses,
		"total_failures":      totalFailures,
	}

	return summary
}
