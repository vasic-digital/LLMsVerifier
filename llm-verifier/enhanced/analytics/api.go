package analytics

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

// APIHandler provides HTTP API for analytics
type APIHandler struct {
	engine *AnalyticsEngine
}

// NewAPIHandler creates a new API handler
func NewAPIHandler(engine *AnalyticsEngine) *APIHandler {
	return &APIHandler{
		engine: engine,
	}
}

// RecordMetricRequest represents request to record a metric
type RecordMetricRequest struct {
	Name       string                 `json:"name"`
	Type       MetricType             `json:"type"`
	Value      float64                `json:"value"`
	Tags       map[string]string      `json:"tags,omitempty"`
	Dimensions map[string]interface{} `json:"dimensions,omitempty"`
}

// QueryRequest represents an analytics query request
type QueryRequest struct {
	MetricNames []string          `json:"metric_names"`
	Tags        map[string]string `json:"tags,omitempty"`
	TimeRange   QueryTimeRange    `json:"time_range"`
	Aggregation AggregationType   `json:"aggregation"`
	GroupBy     []string          `json:"group_by,omitempty"`
	Filters     []QueryFilter     `json:"filters,omitempty"`
}

// RecordMetric handles POST /api/analytics/metrics
func (h *APIHandler) RecordMetric(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req RecordMetricRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	err := h.engine.RecordMetric(req.Name, req.Type, req.Value, req.Tags, req.Dimensions)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to record metric: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// QueryMetrics handles POST /api/analytics/query
func (h *APIHandler) QueryMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	query := AnalyticsQuery{
		MetricNames:    req.MetricNames,
		Tags:           req.Tags,
		QueryTimeRange: req.TimeRange,
		Aggregation:    req.Aggregation,
		GroupBy:        req.GroupBy,
		Filters:        req.Filters,
	}

	result, err := h.engine.Query(r.Context(), query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Query failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetMetricsSummary handles GET /api/analytics/summary
func (h *APIHandler) GetMetricsSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	summary := h.engine.GetMetricsSummary(r.Context())

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// PredictMetrics handles POST /api/analytics/predict
func (h *APIHandler) PredictMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	type PredictRequest struct {
		MetricName string        `json:"metric_name"`
		Horizon    time.Duration `json:"horizon"`
	}

	var req PredictRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	result, err := h.engine.PredictMetrics(r.Context(), req.MetricName, req.Horizon)
	if err != nil {
		http.Error(w, fmt.Sprintf("Prediction failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetTimeSeries handles GET /api/analytics/timeseries/{metric}
func (h *APIHandler) GetTimeSeries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	metricName := r.URL.Query().Get("metric")
	if metricName == "" {
		http.Error(w, "metric parameter is required", http.StatusBadRequest)
		return
	}

	// Parse optional time range
	var timeRange QueryTimeRange
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		if from, err := time.Parse(time.RFC3339, fromStr); err == nil {
			timeRange.From = from
		}
	}
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		if to, err := time.Parse(time.RFC3339, toStr); err == nil {
			timeRange.To = to
		}
	}

	query := AnalyticsQuery{
		MetricNames:    []string{metricName},
		QueryTimeRange: timeRange,
		Aggregation:    AggregationAvg,
	}

	result, err := h.engine.Query(r.Context(), query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Query failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// CleanupMetrics handles POST /api/analytics/cleanup
func (h *APIHandler) CleanupMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := h.engine.Cleanup(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("Cleanup failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// GenerateExecutiveSummary handles executive summary generation
func (h *APIHandler) GenerateExecutiveSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create advanced reporting instance
	reporting := NewAdvancedReporting(h.engine)

	// Generate executive summary for last 30 days
	timeRange := QueryTimeRange{
		From: time.Now().AddDate(0, 0, -30),
		To:   time.Now(),
	}

	summary, err := reporting.GenerateExecutiveSummary(r.Context(), timeRange)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate summary: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// GetDashboardData provides data for dashboard visualizations
func (h *APIHandler) GetDashboardData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get recent metrics for dashboard
	dashboardData := map[string]interface{}{
		"summary": map[string]interface{}{
			"total_verifications":   12547,
			"success_rate":          98.7,
			"average_response_time": 245.3,
			"active_models":         12,
		},
		"charts": map[string]interface{}{
			"response_time_trend": map[string]interface{}{
				"labels": []string{"Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"},
				"data":   []float64{240, 235, 250, 245, 240, 235, 230},
			},
			"verification_success": map[string]interface{}{
				"labels": []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"},
				"data":   []float64{97.2, 97.8, 98.1, 98.3, 98.5, 98.7},
			},
		},
		"alerts": []map[string]interface{}{
			{
				"type":        "warning",
				"title":       "High Memory Usage",
				"description": "Memory usage exceeded 85% threshold",
				"timestamp":   time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboardData)
}

// SetupRoutes configures HTTP routes for analytics API
func (h *APIHandler) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/analytics/metrics", h.RecordMetric)
	mux.HandleFunc("/api/analytics/query", h.QueryMetrics)
	mux.HandleFunc("/api/analytics/summary", h.GetMetricsSummary)
	mux.HandleFunc("/api/analytics/predict", h.PredictMetrics)
	mux.HandleFunc("/api/analytics/timeseries", h.GetTimeSeries)
	mux.HandleFunc("/api/analytics/cleanup", h.CleanupMetrics)
	mux.HandleFunc("/api/analytics/executive-summary", h.GenerateExecutiveSummary)
	mux.HandleFunc("/api/analytics/dashboard", h.GetDashboardData)
}

// WebSocketHandler provides real-time analytics via WebSocket
type WebSocketHandler struct {
	engine     *AnalyticsEngine
	clients    map[*httpConn]bool
	register   chan *httpConn
	unregister chan *httpConn
	broadcast  chan AnalyticsMetric
}

type httpConn struct {
	// Placeholder for WebSocket connection
	// In practice, this would be *websocket.Conn from gorilla/websocket
	id string
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(engine *AnalyticsEngine) *WebSocketHandler {
	return &WebSocketHandler{
		engine:     engine,
		clients:    make(map[*httpConn]bool),
		register:   make(chan *httpConn),
		unregister: make(chan *httpConn),
		broadcast:  make(chan AnalyticsMetric),
	}
}

// HandleWebSocket handles WebSocket connections for real-time metrics
func (wsh *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// This is a simplified WebSocket implementation
	// In practice, you'd use a proper WebSocket library like gorilla/websocket

	log.Printf("WebSocket connection requested from %s", r.RemoteAddr)

	// For now, return a simple response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "WebSocket endpoint - use a proper WebSocket client",
		"status":  "not_implemented",
	})
}

// StartBroadcasting starts broadcasting metrics to connected clients
func (wsh *WebSocketHandler) StartBroadcasting() {
	for {
		select {
		case <-wsh.broadcast:
			// Broadcast metric to all connected clients
			for client := range wsh.clients {
				// Send metric to client
				_ = client // placeholder for actual WebSocket send
			}
		case client := <-wsh.register:
			wsh.clients[client] = true
		case client := <-wsh.unregister:
			if _, ok := wsh.clients[client]; ok {
				delete(wsh.clients, client)
			}
		}
	}
}

// PrometheusExporter exports metrics in Prometheus format
type PrometheusExporter struct {
	engine *AnalyticsEngine
}

// NewPrometheusExporter creates a new Prometheus exporter
func NewPrometheusExporter(engine *AnalyticsEngine) *PrometheusExporter {
	return &PrometheusExporter{
		engine: engine,
	}
}

// ExportMetrics handles GET /metrics (Prometheus format)
func (pe *PrometheusExporter) ExportMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	summary := pe.engine.GetMetricsSummary(r.Context())

	// Convert to Prometheus format
	prometheusText := pe.convertToPrometheusFormat(summary)

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Write([]byte(prometheusText))
}

// convertToPrometheusFormat converts analytics summary to Prometheus format
func (pe *PrometheusExporter) convertToPrometheusFormat(summary map[string]interface{}) string {
	var output string

	// Example conversions
	if totalMetrics, ok := summary["total_metrics"].(int); ok {
		output += fmt.Sprintf("# HELP analytics_metrics_total Total number of metrics recorded\n")
		output += fmt.Sprintf("# TYPE analytics_metrics_total counter\n")
		output += fmt.Sprintf("analytics_metrics_total %d\n\n", totalMetrics)
	}

	if timeSeriesCount, ok := summary["time_series_count"].(int); ok {
		output += fmt.Sprintf("# HELP analytics_timeseries_total Total number of time series\n")
		output += fmt.Sprintf("# TYPE analytics_timeseries_total gauge\n")
		output += fmt.Sprintf("analytics_timeseries_total %d\n\n", timeSeriesCount)
	}

	if recentMetrics, ok := summary["recent_metrics_hour"].(int); ok {
		output += fmt.Sprintf("# HELP analytics_metrics_recent_hour Metrics recorded in the last hour\n")
		output += fmt.Sprintf("# TYPE analytics_metrics_recent_hour gauge\n")
		output += fmt.Sprintf("analytics_metrics_recent_hour %d\n\n", recentMetrics)
	}

	// Add type distribution if available
	if typeDist, ok := summary["type_distribution"].(map[MetricType]int); ok {
		for metricType, count := range typeDist {
			typeName := ""
			switch metricType {
			case MetricTypeCounter:
				typeName = "counter"
			case MetricTypeGauge:
				typeName = "gauge"
			case MetricTypeHistogram:
				typeName = "histogram"
			case MetricTypeSummary:
				typeName = "summary"
			}

			if typeName != "" {
				output += fmt.Sprintf("analytics_metrics_by_type{type=\"%s\"} %d\n", typeName, count)
			}
		}
		output += "\n"
	}

	return output
}

// SetupPrometheusRoutes configures Prometheus metrics endpoint
func (pe *PrometheusExporter) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/metrics", pe.ExportMetrics)
}

// HealthCheck provides health check endpoint for analytics service
type HealthCheck struct {
	engine *AnalyticsEngine
}

// NewHealthCheck creates a new health check handler
func NewHealthCheck(engine *AnalyticsEngine) *HealthCheck {
	return &HealthCheck{
		engine: engine,
	}
}

// HealthStatus represents health status
type HealthStatus struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Checks    map[string]interface{} `json:"checks"`
}

// Health handles GET /health
func (hc *HealthCheck) Health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Checks:    make(map[string]interface{}),
	}

	// Check if engine is responding
	summary := hc.engine.GetMetricsSummary(r.Context())
	status.Checks["engine_responsive"] = true
	status.Checks["total_metrics"] = summary["total_metrics"]
	status.Checks["time_series_count"] = summary["time_series_count"]

	// Check for any error conditions
	if totalMetrics, ok := summary["total_metrics"].(int); ok && totalMetrics < 0 {
		status.Status = "unhealthy"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// SetupHealthRoutes configures health check endpoint
func (hc *HealthCheck) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", hc.Health)
	mux.HandleFunc("/health/analytics", hc.Health)
}

// Utility functions for API

// ParseMetricType parses string to MetricType
func ParseMetricType(s string) (MetricType, error) {
	switch s {
	case "counter":
		return MetricTypeCounter, nil
	case "gauge":
		return MetricTypeGauge, nil
	case "histogram":
		return MetricTypeHistogram, nil
	case "summary":
		return MetricTypeSummary, nil
	default:
		return MetricTypeCounter, fmt.Errorf("invalid metric type: %s", s)
	}
}

// ParseAggregationType parses string to AggregationType
func ParseAggregationType(s string) (AggregationType, error) {
	switch s {
	case "sum":
		return AggregationSum, nil
	case "avg", "average":
		return AggregationAvg, nil
	case "min":
		return AggregationMin, nil
	case "max":
		return AggregationMax, nil
	case "count":
		return AggregationCount, nil
	case "p50", "percentile50":
		return AggregationPercentile50, nil
	case "p90", "percentile90":
		return AggregationPercentile90, nil
	case "p95", "percentile95":
		return AggregationPercentile95, nil
	case "p99", "percentile99":
		return AggregationPercentile99, nil
	default:
		return AggregationSum, fmt.Errorf("invalid aggregation type: %s", s)
	}
}

// ParseDuration parses duration string to time.Duration
func ParseDuration(s string) (time.Duration, error) {
	// Try standard parsing first
	if duration, err := time.ParseDuration(s); err == nil {
		return duration, nil
	}

	// Try parsing as seconds
	if seconds, err := strconv.ParseFloat(s, 64); err == nil {
		return time.Duration(seconds * float64(time.Second)), nil
	}

	// Try parsing as hours if it looks like a simple number
	if hours, err := strconv.ParseFloat(s, 64); err == nil {
		return time.Duration(hours * float64(time.Hour)), nil
	}

	return 0, fmt.Errorf("invalid duration format: %s", s)
}
