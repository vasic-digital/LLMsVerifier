package monitoring

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// PrometheusExporter exports metrics in Prometheus format
type PrometheusExporter struct {
	metricsCollector *MetricsCollector
	alertManager     *AlertManager
}

// NewPrometheusExporter creates a new Prometheus exporter
func NewPrometheusExporter(metricsCollector *MetricsCollector, alertManager *AlertManager) *PrometheusExporter {
	return &PrometheusExporter{
		metricsCollector: metricsCollector,
		alertManager:     alertManager,
	}
}

// ServeHTTP serves Prometheus metrics
func (pe *PrometheusExporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	metrics := pe.generatePrometheusMetrics()
	w.Write([]byte(metrics))
}

// generatePrometheusMetrics generates metrics in Prometheus format
func (pe *PrometheusExporter) generatePrometheusMetrics() string {
	var output strings.Builder

	// Add header
	output.WriteString("# LLM Verifier Metrics\n")
	output.WriteString(fmt.Sprintf("# Generated at %s\n\n", time.Now().Format(time.RFC3339)))

	// Export metrics from collector
	allMetrics := pe.metricsCollector.GetAllMetrics()

	for metricName, samples := range allMetrics {
		if len(samples) == 0 {
			continue
		}

		// Get the latest sample
		latest := samples[len(samples)-1]

		// Convert to Prometheus format
		prometheusName := pe.convertToPrometheusName(metricName)
		metricType := pe.getPrometheusType(latest.Type)

		// Add HELP comment
		output.WriteString(fmt.Sprintf("# HELP %s %s\n", prometheusName, pe.getMetricDescription(metricName)))

		// Add TYPE
		output.WriteString(fmt.Sprintf("# TYPE %s %s\n", prometheusName, metricType))

		// Add metric value with labels
		labels := pe.formatPrometheusLabels(latest.Labels)
		output.WriteString(fmt.Sprintf("%s%s %g %d\n",
			prometheusName,
			labels,
			latest.Value,
			latest.Timestamp.Unix()*1000)) // Prometheus expects milliseconds

		output.WriteString("\n")
	}

	// Add alert metrics
	pe.addAlertMetrics(&output)

	// Add custom business metrics
	pe.addBusinessMetrics(&output)

	return output.String()
}

// convertToPrometheusName converts metric names to Prometheus format
func (pe *PrometheusExporter) convertToPrometheusName(name string) string {
	// Replace invalid characters and convert to snake_case
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, ".", "_")

	// Add namespace prefix
	return fmt.Sprintf("llm_verifier_%s", name)
}

// getPrometheusType converts metric type to Prometheus type
func (pe *PrometheusExporter) getPrometheusType(metricType MetricType) string {
	switch metricType {
	case MetricTypeCounter:
		return "counter"
	case MetricTypeGauge:
		return "gauge"
	case MetricTypeHistogram:
		return "histogram"
	case MetricTypeSummary:
		return "summary"
	default:
		return "gauge"
	}
}

// getMetricDescription returns a description for a metric
func (pe *PrometheusExporter) getMetricDescription(name string) string {
	descriptions := map[string]string{
		"requests_total":          "Total number of requests processed",
		"errors_total":            "Total number of errors encountered",
		"request_latency_seconds": "Request processing latency in seconds",
		"ttft_seconds":            "Time to first token in seconds",
		"active_connections":      "Number of active connections",
		"memory_usage":            "Memory usage in bytes",
	}

	if desc, exists := descriptions[name]; exists {
		return desc
	}
	return fmt.Sprintf("Metric: %s", name)
}

// formatPrometheusLabels formats labels for Prometheus
func (pe *PrometheusExporter) formatPrometheusLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	var labelParts []string
	for key, value := range labels {
		// Escape special characters in label values
		value = strings.ReplaceAll(value, "\\", "\\\\")
		value = strings.ReplaceAll(value, "\"", "\\\"")
		value = strings.ReplaceAll(value, "\n", "\\n")

		labelParts = append(labelParts, fmt.Sprintf(`%s="%s"`, key, value))
	}

	return fmt.Sprintf("{%s}", strings.Join(labelParts, ","))
}

// addAlertMetrics adds alert-related metrics
func (pe *PrometheusExporter) addAlertMetrics(output *strings.Builder) {
	stats := pe.alertManager.GetAlertStats()

	// Active alerts by severity
	if severityCounts, ok := stats["severity_counts"].(map[AlertSeverity]int); ok {
		for severity, count := range severityCounts {
			output.WriteString(fmt.Sprintf("# HELP llm_verifier_active_alerts Active alerts by severity\n"))
			output.WriteString(fmt.Sprintf("# TYPE llm_verifier_active_alerts gauge\n"))
			output.WriteString(fmt.Sprintf("llm_verifier_active_alerts{severity=\"%s\"} %d\n",
				strings.ToLower(string(severity)), count))
		}
	}

	// Total alerts
	if activeCount, ok := stats["active_alerts"].(int); ok {
		output.WriteString(fmt.Sprintf("# HELP llm_verifier_alerts_total Total number of active alerts\n"))
		output.WriteString(fmt.Sprintf("# TYPE llm_verifier_alerts_total gauge\n"))
		output.WriteString(fmt.Sprintf("llm_verifier_alerts_total %d\n", activeCount))
	}

	output.WriteString("\n")
}

// addBusinessMetrics adds business-specific metrics
func (pe *PrometheusExporter) addBusinessMetrics(output *strings.Builder) {
	// System health score
	report := pe.alertManager.metricsTracker.GetPerformanceReport(1 * time.Hour)

	output.WriteString(fmt.Sprintf("# HELP llm_verifier_health_score Overall system health score (0-100)\n"))
	output.WriteString(fmt.Sprintf("# TYPE llm_verifier_health_score gauge\n"))
	output.WriteString(fmt.Sprintf("llm_verifier_health_score %g\n", report.HealthScore))

	// Verification success rate
	if errorStats, exists := report.Metrics["errors_total"]; exists && errorStats.Count > 0 {
		if requestStats, reqExists := report.Metrics["requests_total"]; reqExists && requestStats.Count > 0 {
			successRate := 1.0 - (errorStats.Latest / requestStats.Latest)
			if successRate < 0 {
				successRate = 0
			}
			if successRate > 1 {
				successRate = 1
			}

			output.WriteString(fmt.Sprintf("# HELP llm_verifier_success_rate Request success rate (0-1)\n"))
			output.WriteString(fmt.Sprintf("# TYPE llm_verifier_success_rate gauge\n"))
			output.WriteString(fmt.Sprintf("llm_verifier_success_rate %g\n", successRate))
		}
	}

	// Average response time
	if latencyStats, exists := report.Metrics["request_latency_seconds"]; exists && latencyStats.Count > 0 {
		output.WriteString(fmt.Sprintf("# HELP llm_verifier_avg_response_time_seconds Average response time in seconds\n"))
		output.WriteString(fmt.Sprintf("# TYPE llm_verifier_avg_response_time_seconds gauge\n"))
		output.WriteString(fmt.Sprintf("llm_verifier_avg_response_time_seconds %g\n", latencyStats.Avg))
	}

	output.WriteString("\n")
}

// PrometheusCollector provides a collector interface for Prometheus
type PrometheusCollector struct {
	exporter *PrometheusExporter
}

// NewPrometheusCollector creates a new Prometheus collector
func NewPrometheusCollector(exporter *PrometheusExporter) *PrometheusCollector {
	return &PrometheusCollector{
		exporter: exporter,
	}
}

// Describe returns all descriptors (required by Prometheus interface)
func (pc *PrometheusCollector) Describe(ch chan<- *PrometheusDesc) {
	// This is a simplified implementation
	// In a real implementation, you would define proper metric descriptors
}

// Collect returns current metrics (required by Prometheus interface)
func (pc *PrometheusCollector) Collect(ch chan<- PrometheusMetric) {
	// This is a simplified implementation
	// In a real implementation, you would convert metrics to Prometheus format
}

// PrometheusDesc represents a Prometheus metric descriptor (simplified)
type PrometheusDesc struct {
	Name string
	Help string
	Type string
}

// PrometheusMetric represents a Prometheus metric (simplified)
type PrometheusMetric struct {
	Desc   *PrometheusDesc
	Value  float64
	Labels map[string]string
}

// StartPrometheusServer starts a Prometheus metrics server
func StartPrometheusServer(exporter *PrometheusExporter, port int) error {
	http.Handle("/metrics", exporter)
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting Prometheus metrics server on %s", addr)
	return http.ListenAndServe(addr, nil)
}

// GeneratePrometheusConfig generates a Prometheus configuration snippet
func GeneratePrometheusConfig(jobName, metricsURL string) string {
	config := fmt.Sprintf(`
# LLM Verifier Prometheus Configuration
scrape_configs:
  - job_name: '%s'
    scrape_interval: 15s
    scrape_timeout: 10s
    metrics_path: /metrics
    static_configs:
      - targets: ['%s']
    # Relabeling rules for better organization
    relabel_configs:
      - source_labels: [__address__]
        target_label: instance
        replacement: '${1}'
`, jobName, metricsURL)

	return config
}

// CreateGrafanaDashboard creates a basic Grafana dashboard configuration
func CreateGrafanaDashboard(dashboardTitle string) map[string]interface{} {
	dashboard := map[string]interface{}{
		"dashboard": map[string]interface{}{
			"id":            nil,
			"title":         dashboardTitle,
			"tags":          []string{"llm-verifier", "generated"},
			"timezone":      "browser",
			"panels":        []map[string]interface{}{},
			"time":          map[string]string{"from": "now-1h", "to": "now"},
			"timepicker":    map[string]interface{}{},
			"templating":    map[string]interface{}{"list": []interface{}{}},
			"annotations":   map[string]interface{}{"list": []interface{}{}},
			"refresh":       "5s",
			"schemaVersion": 27,
			"version":       0,
			"links":         []interface{}{},
		},
	}

	panels := []map[string]interface{}{
		// System Health Score
		{
			"id":    1,
			"title": "System Health Score",
			"type":  "stat",
			"targets": []map[string]interface{}{
				{
					"expr":         "llm_verifier_health_score",
					"legendFormat": "Health Score",
				},
			},
			"fieldConfig": map[string]interface{}{
				"defaults": map[string]interface{}{
					"color": map[string]interface{}{
						"mode": "thresholds",
					},
					"thresholds": map[string]interface{}{
						"mode": "absolute",
						"steps": []map[string]interface{}{
							{"color": "red", "value": nil},
							{"color": "orange", "value": 50},
							{"color": "yellow", "value": 75},
							{"color": "green", "value": 90},
						},
					},
				},
			},
		},
		// Request Rate
		{
			"id":    2,
			"title": "Request Rate",
			"type":  "graph",
			"targets": []map[string]interface{}{
				{
					"expr":         "rate(llm_verifier_requests_total[5m])",
					"legendFormat": "Requests/sec",
				},
			},
		},
		// Error Rate
		{
			"id":    3,
			"title": "Error Rate",
			"type":  "graph",
			"targets": []map[string]interface{}{
				{
					"expr":         "rate(llm_verifier_errors_total[5m])",
					"legendFormat": "Errors/sec",
				},
			},
		},
		// Response Time
		{
			"id":    4,
			"title": "Response Time",
			"type":  "graph",
			"targets": []map[string]interface{}{
				{
					"expr":         "llm_verifier_request_latency_seconds",
					"legendFormat": "Latency (s)",
				},
			},
		},
		// Active Alerts
		{
			"id":    5,
			"title": "Active Alerts",
			"type":  "table",
			"targets": []map[string]interface{}{
				{
					"expr":         "llm_verifier_active_alerts",
					"legendFormat": "Active Alerts",
				},
			},
		},
	}

	dashboard["dashboard"].(map[string]interface{})["panels"] = panels

	return dashboard
}
