package monitoring

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== AdvancedMonitor Tests ====================

func TestNewAdvancedMonitor(t *testing.T) {
	monitor := NewAdvancedMonitor()
	require.NotNil(t, monitor)
	assert.NotNil(t, monitor.alerts)
	assert.NotNil(t, monitor.thresholds)
	assert.NotNil(t, monitor.metrics)
}

func TestAdvancedMonitor_SetThreshold(t *testing.T) {
	monitor := NewAdvancedMonitor()

	monitor.SetThreshold("cpu", 70.0, 90.0)

	threshold, exists := monitor.thresholds["cpu"]
	require.True(t, exists)
	assert.Equal(t, 70.0, threshold.Warning)
	assert.Equal(t, 90.0, threshold.Critical)
	assert.True(t, threshold.Enabled)
}

func TestAdvancedMonitor_SetThreshold_Multiple(t *testing.T) {
	monitor := NewAdvancedMonitor()

	monitor.SetThreshold("cpu", 70.0, 90.0)
	monitor.SetThreshold("memory", 60.0, 80.0)
	monitor.SetThreshold("disk", 75.0, 95.0)

	assert.Len(t, monitor.thresholds, 3)
}

func TestAdvancedMonitor_RecordMetric(t *testing.T) {
	monitor := NewAdvancedMonitor()

	monitor.RecordMetric("cpu", 50.0)
	monitor.RecordMetric("cpu", 60.0)
	monitor.RecordMetric("cpu", 70.0)

	metrics := monitor.GetMetrics("cpu", 10)
	assert.Len(t, metrics, 3)
	assert.Equal(t, 50.0, metrics[0].Value)
	assert.Equal(t, 60.0, metrics[1].Value)
	assert.Equal(t, 70.0, metrics[2].Value)
}

func TestAdvancedMonitor_RecordMetric_TriggersWarning(t *testing.T) {
	monitor := NewAdvancedMonitor()

	monitor.SetThreshold("cpu", 70.0, 90.0)
	monitor.RecordMetric("cpu", 75.0)

	alerts := monitor.GetActiveAlerts()
	require.Len(t, alerts, 1)
	assert.Equal(t, "warning", alerts[0].Severity)
	assert.Contains(t, alerts[0].Message, "warning threshold")
}

func TestAdvancedMonitor_RecordMetric_TriggersCritical(t *testing.T) {
	monitor := NewAdvancedMonitor()

	monitor.SetThreshold("cpu", 70.0, 90.0)
	monitor.RecordMetric("cpu", 95.0)

	alerts := monitor.GetActiveAlerts()
	require.Len(t, alerts, 1)
	assert.Equal(t, "critical", alerts[0].Severity)
	assert.Contains(t, alerts[0].Message, "critical threshold")
}

func TestAdvancedMonitor_RecordMetric_NoAlertBelowThreshold(t *testing.T) {
	monitor := NewAdvancedMonitor()

	monitor.SetThreshold("cpu", 70.0, 90.0)
	monitor.RecordMetric("cpu", 50.0)

	alerts := monitor.GetActiveAlerts()
	assert.Empty(t, alerts)
}

func TestAdvancedMonitor_RecordMetric_NoThresholdSet(t *testing.T) {
	monitor := NewAdvancedMonitor()

	// Record metric without setting threshold
	monitor.RecordMetric("cpu", 95.0)

	alerts := monitor.GetActiveAlerts()
	assert.Empty(t, alerts)
}

func TestAdvancedMonitor_RecordMetric_TrimOldMetrics(t *testing.T) {
	monitor := NewAdvancedMonitor()

	// Record more than 1000 metrics
	for i := 0; i < 1100; i++ {
		monitor.RecordMetric("test", float64(i))
	}

	// Should be trimmed to 1000
	assert.LessOrEqual(t, len(monitor.metrics["test"]), 1000)
}

func TestAdvancedMonitor_GetActiveAlerts(t *testing.T) {
	monitor := NewAdvancedMonitor()

	monitor.SetThreshold("cpu", 70.0, 90.0)
	monitor.RecordMetric("cpu", 75.0)
	monitor.RecordMetric("cpu", 95.0)

	alerts := monitor.GetActiveAlerts()
	assert.Len(t, alerts, 2)
}

func TestAdvancedMonitor_GetActiveAlerts_Empty(t *testing.T) {
	monitor := NewAdvancedMonitor()

	alerts := monitor.GetActiveAlerts()
	assert.Empty(t, alerts)
}

func TestAdvancedMonitor_GetMetrics_Empty(t *testing.T) {
	monitor := NewAdvancedMonitor()

	metrics := monitor.GetMetrics("nonexistent", 10)
	assert.Empty(t, metrics)
}

func TestAdvancedMonitor_GetMetrics_WithLimit(t *testing.T) {
	monitor := NewAdvancedMonitor()

	for i := 0; i < 20; i++ {
		monitor.RecordMetric("test", float64(i))
	}

	metrics := monitor.GetMetrics("test", 5)
	assert.Len(t, metrics, 5)
	// Should return last 5 values
	assert.Equal(t, 15.0, metrics[0].Value)
	assert.Equal(t, 19.0, metrics[4].Value)
}

func TestAdvancedMonitor_GetMetrics_LimitLargerThanData(t *testing.T) {
	monitor := NewAdvancedMonitor()

	monitor.RecordMetric("test", 1.0)
	monitor.RecordMetric("test", 2.0)

	metrics := monitor.GetMetrics("test", 100)
	assert.Len(t, metrics, 2)
}

func TestAdvancedMonitor_ResolveAlert(t *testing.T) {
	monitor := NewAdvancedMonitor()

	monitor.SetThreshold("cpu", 70.0, 90.0)
	monitor.RecordMetric("cpu", 75.0)

	alerts := monitor.GetActiveAlerts()
	require.Len(t, alerts, 1)
	alertID := alerts[0].ID

	monitor.ResolveAlert(alertID)

	// Alert should no longer be active
	activeAlerts := monitor.GetActiveAlerts()
	assert.Empty(t, activeAlerts)
}

func TestAdvancedMonitor_ResolveAlert_NonExistent(t *testing.T) {
	monitor := NewAdvancedMonitor()

	// Should not panic
	monitor.ResolveAlert("nonexistent-id")
}

func TestAdvancedMonitor_GetMetricStats(t *testing.T) {
	monitor := NewAdvancedMonitor()

	// Record some metrics
	values := []float64{10.0, 20.0, 30.0, 40.0, 50.0}
	for _, v := range values {
		monitor.RecordMetric("test", v)
	}

	stats := monitor.GetMetricStats("test")

	assert.Equal(t, 5, stats["count"])
	assert.Equal(t, 50.0, stats["current"])
	assert.Equal(t, 30.0, stats["average"])
	assert.Equal(t, 10.0, stats["minimum"])
	assert.Equal(t, 50.0, stats["maximum"])
	assert.Equal(t, 150.0, stats["sum"])
}

func TestAdvancedMonitor_GetMetricStats_Empty(t *testing.T) {
	monitor := NewAdvancedMonitor()

	stats := monitor.GetMetricStats("nonexistent")

	assert.Equal(t, 0, stats["count"])
	assert.Equal(t, 0.0, stats["current"])
}

func TestAdvancedMonitor_HealthCheck_Healthy(t *testing.T) {
	monitor := NewAdvancedMonitor()

	health := monitor.HealthCheck()

	assert.Equal(t, "healthy", health["status"])
	assert.Equal(t, 0, health["active_alerts"])
	assert.NotNil(t, health["timestamp"])
}

func TestAdvancedMonitor_HealthCheck_WithWarnings(t *testing.T) {
	monitor := NewAdvancedMonitor()

	monitor.SetThreshold("cpu", 70.0, 90.0)
	monitor.RecordMetric("cpu", 75.0) // Warning

	health := monitor.HealthCheck()

	assert.Equal(t, "healthy", health["status"]) // Warnings don't change status to critical
	assert.Equal(t, 1, health["active_alerts"])
}

func TestAdvancedMonitor_HealthCheck_Critical(t *testing.T) {
	monitor := NewAdvancedMonitor()

	monitor.SetThreshold("cpu", 70.0, 90.0)
	monitor.RecordMetric("cpu", 95.0) // Critical

	health := monitor.HealthCheck()

	assert.Equal(t, "critical", health["status"])
	assert.Equal(t, 1, health["critical_alerts"])
	assert.Equal(t, 1, health["active_alerts"])
}

// ==================== MetricsTracker Extended Tests ====================

func TestMetricsTracker_QueueVerification(t *testing.T) {
	tracker := NewMetricsTracker()

	tracker.QueueVerification()
	tracker.QueueVerification()
	tracker.QueueVerification()

	stats := tracker.GetVerificationStats()
	assert.Equal(t, 3, stats.QueueLength)
}

func TestMetricsTracker_RecordBrotliTest(t *testing.T) {
	tracker := NewMetricsTracker()

	tracker.RecordBrotliTest(true, 100*time.Millisecond)
	tracker.RecordBrotliTest(false, 200*time.Millisecond)
	tracker.RecordBrotliTest(true, 150*time.Millisecond)

	metrics := tracker.GetBrotliMetrics()

	assert.Equal(t, int64(3), metrics["tests_performed"])
	assert.Equal(t, int64(2), metrics["supported_models"])
}

func TestMetricsTracker_RecordBrotliTest_FirstTest(t *testing.T) {
	tracker := NewMetricsTracker()

	tracker.RecordBrotliTest(true, 100*time.Millisecond)

	metrics := tracker.GetBrotliMetrics()
	assert.Equal(t, int64(1), metrics["tests_performed"])
	assert.Equal(t, int64(1), metrics["supported_models"])
}

func TestMetricsTracker_RecordBrotliCacheHit(t *testing.T) {
	tracker := NewMetricsTracker()

	tracker.RecordBrotliCacheHit()
	tracker.RecordBrotliCacheHit()
	tracker.RecordBrotliCacheHit()

	metrics := tracker.GetBrotliMetrics()
	assert.Equal(t, int64(3), metrics["cache_hits"])
}

func TestMetricsTracker_RecordBrotliCacheMiss(t *testing.T) {
	tracker := NewMetricsTracker()

	tracker.RecordBrotliCacheMiss()
	tracker.RecordBrotliCacheMiss()

	metrics := tracker.GetBrotliMetrics()
	assert.Equal(t, int64(2), metrics["cache_misses"])
}

func TestMetricsTracker_GetBrotliMetrics_CacheHitRate(t *testing.T) {
	tracker := NewMetricsTracker()

	// Simulate some tests first
	tracker.RecordBrotliTest(true, 100*time.Millisecond)

	// 3 hits, 1 miss = 75% hit rate
	tracker.RecordBrotliCacheHit()
	tracker.RecordBrotliCacheHit()
	tracker.RecordBrotliCacheHit()
	tracker.RecordBrotliCacheMiss()

	metrics := tracker.GetBrotliMetrics()
	assert.Equal(t, 75.0, metrics["cache_hit_rate"])
}

func TestMetricsTracker_GetBrotliMetrics_Empty(t *testing.T) {
	tracker := NewMetricsTracker()

	metrics := tracker.GetBrotliMetrics()

	assert.Equal(t, int64(0), metrics["tests_performed"])
	assert.Equal(t, int64(0), metrics["supported_models"])
	assert.Equal(t, 0.0, metrics["support_rate_percent"])
}

func TestMetricsTracker_RecordNotificationSent(t *testing.T) {
	tracker := NewMetricsTracker()

	tracker.RecordNotificationSent(true)
	tracker.RecordNotificationSent(true)
	tracker.RecordNotificationSent(false)

	stats := tracker.GetNotificationStats()
	assert.Equal(t, 3, stats["messages_sent"])
}

func TestMetricsTracker_RecordSchedulerJobStarted(t *testing.T) {
	tracker := NewMetricsTracker()

	// Queue some jobs first
	tracker.QueueSchedulerJob()
	tracker.QueueSchedulerJob()

	// Start a job
	tracker.RecordSchedulerJobStarted()

	stats := tracker.GetSchedulerStats()
	assert.Equal(t, 1, stats.ActiveJobs)
	assert.Equal(t, 1, stats.QueuedJobs) // 2 queued - 1 started = 1
}

func TestMetricsTracker_RecordSchedulerJobCompleted_Success(t *testing.T) {
	tracker := NewMetricsTracker()

	tracker.RecordSchedulerJobStarted()
	tracker.RecordSchedulerJobCompleted(true)

	stats := tracker.GetSchedulerStats()
	assert.Equal(t, 0, stats.ActiveJobs)
	assert.Equal(t, int64(1), stats.CompletedJobs)
	assert.Equal(t, int64(0), stats.FailedJobs)
}

func TestMetricsTracker_RecordSchedulerJobCompleted_Failure(t *testing.T) {
	tracker := NewMetricsTracker()

	tracker.RecordSchedulerJobStarted()
	tracker.RecordSchedulerJobCompleted(false)

	stats := tracker.GetSchedulerStats()
	assert.Equal(t, 0, stats.ActiveJobs)
	assert.Equal(t, int64(0), stats.CompletedJobs)
	assert.Equal(t, int64(1), stats.FailedJobs)
}

func TestMetricsTracker_QueueSchedulerJob(t *testing.T) {
	tracker := NewMetricsTracker()

	tracker.QueueSchedulerJob()
	tracker.QueueSchedulerJob()
	tracker.QueueSchedulerJob()

	stats := tracker.GetSchedulerStats()
	assert.Equal(t, 3, stats.QueuedJobs)
}

func TestMetricsTracker_SetSchedulerJobCount(t *testing.T) {
	tracker := NewMetricsTracker()

	tracker.SetSchedulerJobCount(5, 10)

	stats := tracker.GetSchedulerStats()
	assert.Equal(t, 5, stats.ActiveJobs)
	assert.Equal(t, 10, stats.QueuedJobs)
}

func TestMetricsTracker_RecordQueryError(t *testing.T) {
	tracker := NewMetricsTracker()

	tracker.RecordQueryError()
	tracker.RecordQueryError()

	// RecordQueryError should increment error count
	// The function doesn't return anything, so we just verify it doesn't panic
}

// ==================== PrometheusExporter Tests ====================

func TestNewPrometheusExporter(t *testing.T) {
	metricsCollector := NewMetricsCollector()
	metricsTracker := NewMetricsTracker()
	alertManager := NewAlertManager(metricsTracker)

	exporter := NewPrometheusExporter(metricsCollector, alertManager, metricsTracker)
	require.NotNil(t, exporter)
	assert.NotNil(t, exporter.metricsCollector)
	assert.NotNil(t, exporter.alertManager)
	assert.NotNil(t, exporter.metricsTracker)
}

func TestPrometheusExporter_ServeHTTP(t *testing.T) {
	metricsCollector := NewMetricsCollector()
	metricsTracker := NewMetricsTracker()
	alertManager := NewAlertManager(metricsTracker)

	exporter := NewPrometheusExporter(metricsCollector, alertManager, metricsTracker)

	// Record some metrics
	metricsTracker.RecordBrotliTest(true, 100*time.Millisecond)
	metricsTracker.RecordBrotliCacheHit()

	// Create request and response recorder
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	exporter.ServeHTTP(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/plain; version=0.0.4", resp.Header.Get("Content-Type"))

	body := w.Body.String()
	assert.Contains(t, body, "brotli_tests_performed")
	assert.Contains(t, body, "brotli_supported_models")
	assert.Contains(t, body, "brotli_cache_hits")
	assert.Contains(t, body, "verification_active_count")
	assert.Contains(t, body, "verification_success_rate")
}

func TestPrometheusExporter_ServeHTTP_Empty(t *testing.T) {
	metricsCollector := NewMetricsCollector()
	metricsTracker := NewMetricsTracker()
	alertManager := NewAlertManager(metricsTracker)

	exporter := NewPrometheusExporter(metricsCollector, alertManager, metricsTracker)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	exporter.ServeHTTP(w, req)

	resp := w.Result()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body := w.Body.String()
	assert.Contains(t, body, "brotli_tests_performed 0")
}

func TestPrometheusExporter_ServeHTTP_WithDuration(t *testing.T) {
	metricsCollector := NewMetricsCollector()
	metricsTracker := NewMetricsTracker()
	alertManager := NewAlertManager(metricsTracker)

	exporter := NewPrometheusExporter(metricsCollector, alertManager, metricsTracker)

	// Record metrics with specific duration
	metricsTracker.RecordBrotliTest(true, 100*time.Millisecond)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	exporter.ServeHTTP(w, req)

	body := w.Body.String()
	assert.Contains(t, body, "brotli_avg_detection_duration_seconds")
}

// ==================== AlertManager CheckRules Tests ====================

func TestAlertManager_CheckRules(t *testing.T) {
	tracker := NewMetricsTracker()
	am := NewAlertManager(tracker)

	// Add a custom rule with condition that triggers
	rule := &AlertRule{
		ID:          "test-rule-1",
		Name:        "test-rule",
		Description: "Test rule for unit tests",
		Condition:   "cpu_usage > 90",
		Threshold:   90.0,
		Severity:    AlertSeverityCritical,
		Enabled:     true,
	}
	am.AddRule(rule)

	am.CheckRules()

	// Verify the rule was added
	rules := am.GetRules()
	assert.GreaterOrEqual(t, len(rules), 1)
}

func TestAlertManager_CheckRules_Disabled(t *testing.T) {
	tracker := NewMetricsTracker()
	am := NewAlertManager(tracker)

	// Add a disabled rule
	rule := &AlertRule{
		ID:          "disabled-rule",
		Name:        "disabled-rule",
		Description: "Disabled rule",
		Condition:   "always",
		Threshold:   0,
		Severity:    AlertSeverityInfo,
		Enabled:     false,
	}
	am.AddRule(rule)

	initialCount := len(am.GetActiveAlerts())
	am.CheckRules()

	// Should not have created new alerts
	assert.Equal(t, initialCount, len(am.GetActiveAlerts()))
}

// ==================== MonitorAlert Struct Tests ====================

func TestMonitorAlert_Struct(t *testing.T) {
	alert := MonitorAlert{
		ID:        "alert-123",
		Type:      "threshold_exceeded",
		Message:   "CPU exceeded critical threshold",
		Severity:  "critical",
		Value:     95.5,
		Threshold: 90.0,
		Timestamp: time.Now(),
		Resolved:  false,
	}

	assert.Equal(t, "alert-123", alert.ID)
	assert.Equal(t, "threshold_exceeded", alert.Type)
	assert.Equal(t, "critical", alert.Severity)
	assert.Equal(t, 95.5, alert.Value)
	assert.False(t, alert.Resolved)
}

func TestThreshold_Struct(t *testing.T) {
	threshold := Threshold{
		Warning:  70.0,
		Critical: 90.0,
		Enabled:  true,
	}

	assert.Equal(t, 70.0, threshold.Warning)
	assert.Equal(t, 90.0, threshold.Critical)
	assert.True(t, threshold.Enabled)
}

func TestMetricPoint_Struct(t *testing.T) {
	point := MetricPoint{
		Value:     42.5,
		Timestamp: time.Now(),
	}

	assert.Equal(t, 42.5, point.Value)
	assert.False(t, point.Timestamp.IsZero())
}

func TestSchedulerStats_Struct(t *testing.T) {
	stats := SchedulerStats{
		ActiveJobs:    5,
		CompletedJobs: 100,
		FailedJobs:    3,
		QueuedJobs:    10,
		IsRunning:     true,
		LastCheckTime: time.Now(),
	}

	assert.Equal(t, 5, stats.ActiveJobs)
	assert.Equal(t, int64(100), stats.CompletedJobs)
	assert.Equal(t, int64(3), stats.FailedJobs)
	assert.True(t, stats.IsRunning)
}

// ==================== Integration Tests ====================

func TestAdvancedMonitor_FullWorkflow(t *testing.T) {
	monitor := NewAdvancedMonitor()

	// Set up thresholds
	monitor.SetThreshold("cpu", 70.0, 90.0)
	monitor.SetThreshold("memory", 60.0, 80.0)

	// Record normal values
	monitor.RecordMetric("cpu", 50.0)
	monitor.RecordMetric("memory", 40.0)
	assert.Empty(t, monitor.GetActiveAlerts())

	// Record warning level
	monitor.RecordMetric("cpu", 75.0)
	alerts := monitor.GetActiveAlerts()
	assert.Len(t, alerts, 1)
	assert.Equal(t, "warning", alerts[0].Severity)

	// Record critical level
	monitor.RecordMetric("memory", 85.0)
	alerts = monitor.GetActiveAlerts()
	assert.Len(t, alerts, 2)

	// Check health
	health := monitor.HealthCheck()
	assert.Equal(t, "critical", health["status"])
	assert.Equal(t, 2, health["active_alerts"])

	// Get stats
	cpuStats := monitor.GetMetricStats("cpu")
	assert.Equal(t, 2, cpuStats["count"])

	// Resolve an alert
	warningAlertID := alerts[0].ID
	monitor.ResolveAlert(warningAlertID)
	assert.Len(t, monitor.GetActiveAlerts(), 1)
}

func TestMetricsTracker_SchedulerWorkflow(t *testing.T) {
	tracker := NewMetricsTracker()

	// Set scheduler running
	tracker.SetSchedulerRunning(true)

	// Queue some jobs
	tracker.QueueSchedulerJob()
	tracker.QueueSchedulerJob()
	tracker.QueueSchedulerJob()

	// Start processing
	tracker.RecordSchedulerJobStarted()
	tracker.RecordSchedulerJobStarted()

	stats := tracker.GetSchedulerStats()
	assert.True(t, stats.IsRunning)
	assert.Equal(t, 2, stats.ActiveJobs)
	assert.Equal(t, 1, stats.QueuedJobs)

	// Complete jobs
	tracker.RecordSchedulerJobCompleted(true)
	tracker.RecordSchedulerJobCompleted(false)

	stats = tracker.GetSchedulerStats()
	assert.Equal(t, 0, stats.ActiveJobs)
	assert.Equal(t, int64(1), stats.CompletedJobs)
	assert.Equal(t, int64(1), stats.FailedJobs)

	// Stop scheduler
	tracker.SetSchedulerRunning(false)
	stats = tracker.GetSchedulerStats()
	assert.False(t, stats.IsRunning)
}

func TestMetricsTracker_BrotliWorkflow(t *testing.T) {
	tracker := NewMetricsTracker()

	// Simulate testing multiple models
	tracker.RecordBrotliTest(true, 50*time.Millisecond)
	tracker.RecordBrotliTest(true, 60*time.Millisecond)
	tracker.RecordBrotliTest(false, 40*time.Millisecond)
	tracker.RecordBrotliTest(true, 55*time.Millisecond)

	// Simulate cache usage
	tracker.RecordBrotliCacheHit()
	tracker.RecordBrotliCacheHit()
	tracker.RecordBrotliCacheMiss()

	metrics := tracker.GetBrotliMetrics()
	assert.Equal(t, int64(4), metrics["tests_performed"])
	assert.Equal(t, int64(3), metrics["supported_models"])
	assert.InDelta(t, 75.0, metrics["support_rate_percent"], 0.1)
	assert.Equal(t, int64(2), metrics["cache_hits"])
	assert.Equal(t, int64(1), metrics["cache_misses"])
}

// ==================== Concurrency Tests ====================

func TestAdvancedMonitor_ConcurrentAccess(t *testing.T) {
	monitor := NewAdvancedMonitor()
	monitor.SetThreshold("test", 50.0, 80.0)

	done := make(chan bool)

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			monitor.RecordMetric("test", float64(i))
		}
		done <- true
	}()

	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			_ = monitor.GetMetrics("test", 10)
		}
		done <- true
	}()

	// Concurrent health checks
	go func() {
		for i := 0; i < 50; i++ {
			_ = monitor.HealthCheck()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
}

func TestMetricsTracker_ConcurrentBrotli(t *testing.T) {
	tracker := NewMetricsTracker()

	done := make(chan bool)

	// Concurrent Brotli recording
	go func() {
		for i := 0; i < 100; i++ {
			tracker.RecordBrotliTest(i%2 == 0, time.Duration(i)*time.Millisecond)
		}
		done <- true
	}()

	// Concurrent cache recording
	go func() {
		for i := 0; i < 100; i++ {
			if i%3 == 0 {
				tracker.RecordBrotliCacheHit()
			} else {
				tracker.RecordBrotliCacheMiss()
			}
		}
		done <- true
	}()

	// Concurrent reads
	go func() {
		for i := 0; i < 50; i++ {
			_ = tracker.GetBrotliMetrics()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	metrics := tracker.GetBrotliMetrics()
	assert.Equal(t, int64(100), metrics["tests_performed"])
}

func TestMetricsTracker_ConcurrentScheduler(t *testing.T) {
	tracker := NewMetricsTracker()

	done := make(chan bool)

	// Concurrent job queueing
	go func() {
		for i := 0; i < 50; i++ {
			tracker.QueueSchedulerJob()
		}
		done <- true
	}()

	// Concurrent job starting
	go func() {
		for i := 0; i < 30; i++ {
			tracker.RecordSchedulerJobStarted()
		}
		done <- true
	}()

	// Concurrent job completion
	go func() {
		for i := 0; i < 20; i++ {
			tracker.RecordSchedulerJobCompleted(i%2 == 0)
		}
		done <- true
	}()

	// Concurrent reads
	go func() {
		for i := 0; i < 50; i++ {
			_ = tracker.GetSchedulerStats()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}
}
