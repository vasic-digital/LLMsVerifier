package monitoring

// No imports needed for stub

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
