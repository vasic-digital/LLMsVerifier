package monitoring

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// PrometheusExporter exports metrics in Prometheus format
type PrometheusExporter struct {
	metricsCollector *MetricsCollector
	alertManager     *AlertManager
	metricsTracker   *MetricsTracker
	mu               sync.RWMutex
}

// NewPrometheusExporter creates a new Prometheus exporter
func NewPrometheusExporter(metricsCollector *MetricsCollector, alertManager *AlertManager, metricsTracker *MetricsTracker) *PrometheusExporter {
	return &PrometheusExporter{
		metricsCollector: metricsCollector,
		alertManager:     alertManager,
		metricsTracker:   metricsTracker,
	}
}

// ServeHTTP serves Prometheus metrics at /metrics endpoint
func (pe *PrometheusExporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	// Get current Brotli metrics
	brotliMetrics := pe.metricsTracker.GetBrotliMetrics()

	// Export Brotli metrics in Prometheus format
	fmt.Fprintf(w, "# HELP brotli_tests_performed Total number of Brotli compression tests performed\n")
	fmt.Fprintf(w, "# TYPE brotli_tests_performed counter\n")
	fmt.Fprintf(w, "brotli_tests_performed %d\n", brotliMetrics["tests_performed"])

	fmt.Fprintf(w, "# HELP brotli_supported_models Number of models supporting Brotli compression\n")
	fmt.Fprintf(w, "# TYPE brotli_supported_models gauge\n")
	fmt.Fprintf(w, "brotli_supported_models %d\n", brotliMetrics["supported_models"])

	fmt.Fprintf(w, "# HELP brotli_support_rate_percent Percentage of models supporting Brotli compression\n")
	fmt.Fprintf(w, "# TYPE brotli_support_rate_percent gauge\n")
	fmt.Fprintf(w, "brotli_support_rate_percent %.2f\n", brotliMetrics["support_rate_percent"])

	fmt.Fprintf(w, "# HELP brotli_cache_hits Number of Brotli cache hits\n")
	fmt.Fprintf(w, "# TYPE brotli_cache_hits counter\n")
	fmt.Fprintf(w, "brotli_cache_hits %d\n", brotliMetrics["cache_hits"])

	fmt.Fprintf(w, "# HELP brotli_cache_misses Number of Brotli cache misses\n")
	fmt.Fprintf(w, "# TYPE brotli_cache_misses counter\n")
	fmt.Fprintf(w, "brotli_cache_misses %d\n", brotliMetrics["cache_misses"])

	fmt.Fprintf(w, "# HELP brotli_cache_hit_rate Brotli cache hit rate percentage\n")
	fmt.Fprintf(w, "# TYPE brotli_cache_hit_rate gauge\n")
	fmt.Fprintf(w, "brotli_cache_hit_rate %.2f\n", brotliMetrics["cache_hit_rate"])

	// Convert detection duration to seconds for Prometheus
	if avgDurationStr, ok := brotliMetrics["avg_detection_duration"].(string); ok {
		if duration, err := time.ParseDuration(avgDurationStr); err == nil {
			fmt.Fprintf(w, "# HELP brotli_avg_detection_duration_seconds Average Brotli detection time in seconds\n")
			fmt.Fprintf(w, "# TYPE brotli_avg_detection_duration_seconds gauge\n")
			fmt.Fprintf(w, "brotli_avg_detection_duration_seconds %.6f\n", duration.Seconds())
		}
	}

	// Export other system metrics if available
	verificationStats := pe.metricsTracker.GetVerificationStats()
	fmt.Fprintf(w, "# HELP verification_active_count Number of active verifications\n")
	fmt.Fprintf(w, "# TYPE verification_active_count gauge\n")
	fmt.Fprintf(w, "verification_active_count %d\n", verificationStats.ActiveVerifications)

	fmt.Fprintf(w, "# HELP verification_success_rate Verification success rate\n")
	fmt.Fprintf(w, "# TYPE verification_success_rate gauge\n")
	fmt.Fprintf(w, "verification_success_rate %.2f\n", verificationStats.SuccessRate)
}
