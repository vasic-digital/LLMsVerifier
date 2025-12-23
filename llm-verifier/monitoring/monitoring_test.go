package monitoring

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAlertSeverityConstants(t *testing.T) {
	assert.Equal(t, AlertSeverity("info"), AlertSeverityInfo)
	assert.Equal(t, AlertSeverity("warning"), AlertSeverityWarning)
	assert.Equal(t, AlertSeverity("error"), AlertSeverityError)
	assert.Equal(t, AlertSeverity("critical"), AlertSeverityCritical)
}

func TestMetricTypeConstants(t *testing.T) {
	assert.Equal(t, MetricType("counter"), MetricTypeCounter)
	assert.Equal(t, MetricType("gauge"), MetricTypeGauge)
	assert.Equal(t, MetricType("histogram"), MetricTypeHistogram)
}

func TestNewMetricsCollector(t *testing.T) {
	mc := NewMetricsCollector()

	assert.NotNil(t, mc)
	assert.NotNil(t, mc.metrics)
}

func TestMetricsCollectorRecordCounter(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordCounter("test_counter", 1.0, map[string]string{"env": "test"}, "Test counter")

	metric, exists := mc.GetMetric("test_counter", map[string]string{"env": "test"})

	assert.True(t, exists)
	assert.NotNil(t, metric)
	assert.Equal(t, "test_counter", metric.Name)
	assert.Equal(t, MetricTypeCounter, metric.Type)
	assert.Equal(t, 1.0, metric.Value)
}

func TestMetricsCollectorRecordCounterIncrement(t *testing.T) {
	mc := NewMetricsCollector()

	labels := map[string]string{"env": "test"}
	mc.RecordCounter("test_counter", 1.0, labels, "Test counter")
	mc.RecordCounter("test_counter", 2.0, labels, "Test counter increment")

	metric, _ := mc.GetMetric("test_counter", labels)

	assert.NotNil(t, metric)
	assert.Equal(t, 3.0, metric.Value, "Counter should increment")
}

func TestMetricsCollectorRecordGauge(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordGauge("test_gauge", 42.5, map[string]string{"env": "prod"}, "Test gauge")

	metric, exists := mc.GetMetric("test_gauge", map[string]string{"env": "prod"})

	assert.True(t, exists)
	assert.NotNil(t, metric)
	assert.Equal(t, "test_gauge", metric.Name)
	assert.Equal(t, MetricTypeGauge, metric.Type)
	assert.Equal(t, 42.5, metric.Value)
}

func TestMetricsCollectorRecordGaugeReplace(t *testing.T) {
	mc := NewMetricsCollector()

	labels := map[string]string{"env": "test"}
	mc.RecordGauge("test_gauge", 10.0, labels, "Test gauge")
	mc.RecordGauge("test_gauge", 20.0, labels, "Test gauge replace")

	metric, _ := mc.GetMetric("test_gauge", labels)

	assert.NotNil(t, metric)
	assert.Equal(t, 20.0, metric.Value, "Gauge should replace value")
}

func TestMetricsCollectorRecordHistogram(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordHistogram("test_histogram", 0.5, map[string]string{"operation": "test"}, "Test histogram")

	metric, exists := mc.GetMetric("test_histogram", map[string]string{"operation": "test"})

	assert.True(t, exists)
	assert.NotNil(t, metric)
	assert.Equal(t, "test_histogram", metric.Name)
	assert.Equal(t, MetricTypeHistogram, metric.Type)
	assert.Equal(t, 0.5, metric.Value)
}

func TestMetricsCollectorGetAllMetrics(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordCounter("counter1", 1.0, map[string]string{"env": "test"}, "")
	mc.RecordGauge("gauge1", 10.0, map[string]string{"env": "test"}, "")

	metrics := mc.GetAllMetrics()

	assert.NotNil(t, metrics)
	assert.Equal(t, 2, len(metrics))
}

func TestMetricsCollectorGetAllMetricsEmpty(t *testing.T) {
	mc := NewMetricsCollector()

	metrics := mc.GetAllMetrics()

	assert.NotNil(t, metrics)
	assert.Equal(t, 0, len(metrics))
}

func TestMetricsCollectorGetMetricsByType(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordCounter("counter1", 1.0, nil, "")
	mc.RecordCounter("counter2", 2.0, nil, "")
	mc.RecordGauge("gauge1", 10.0, nil, "")

	counters := mc.GetMetricsByType(MetricTypeCounter)

	assert.NotNil(t, counters)
	assert.Equal(t, 2, len(counters))
}

func TestMetricsCollectorGetMetricsByName(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordCounter("test_metric", 1.0, map[string]string{"env": "test"}, "")
	mc.RecordCounter("test_metric", 2.0, map[string]string{"env": "prod"}, "")

	metrics := mc.GetMetricsByName("test_metric")

	assert.NotNil(t, metrics)
	assert.Equal(t, 2, len(metrics))
}

func TestMetricsCollectorClearMetrics(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordCounter("counter1", 1.0, nil, "")
	mc.ClearMetrics()

	metrics := mc.GetAllMetrics()

	assert.Equal(t, 0, len(metrics))
}

func TestMetricsCollectorGetMetricNotExists(t *testing.T) {
	mc := NewMetricsCollector()

	_, exists := mc.GetMetric("nonexistent", nil)

	assert.False(t, exists)
}

func TestMetricsCollectorMetricTimestamp(t *testing.T) {
	mc := NewMetricsCollector()

	before := time.Now()
	mc.RecordCounter("test_counter", 1.0, nil, "")
	after := time.Now()

	metric, _ := mc.GetMetric("test_counter", nil)

	assert.True(t, metric.Timestamp.After(before) || metric.Timestamp.Equal(before))
	assert.True(t, metric.Timestamp.Before(after) || metric.Timestamp.Equal(after))
}

func TestMetricsCollectorMetricDescription(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordCounter("test_counter", 1.0, nil, "This is a test counter")

	metric, _ := mc.GetMetric("test_counter", nil)

	assert.Equal(t, "This is a test counter", metric.Description)
}

func TestMetricsCollectorGetMetricKey(t *testing.T) {
	mc := NewMetricsCollector()

	key := mc.getMetricKey("test", nil)
	assert.Equal(t, "test", key)

	key = mc.getMetricKey("test", map[string]string{"env": "prod"})
	assert.Equal(t, "test{env=prod}", key)

	key = mc.getMetricKey("test", map[string]string{"env": "prod", "region": "us-west"})
	assert.Contains(t, key, "test{")
	assert.Contains(t, key, "env=prod")
	assert.Contains(t, key, "region=us-west")
}

func TestMetricsCollectorConcurrentRecording(t *testing.T) {
	mc := NewMetricsCollector()

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			mc.RecordCounter("concurrent_counter", 1.0, nil, "")
		}(i)
	}

	wg.Wait()

	metric, _ := mc.GetMetric("concurrent_counter", nil)
	assert.NotNil(t, metric)
	assert.Equal(t, 100.0, metric.Value)
}

func TestPerformanceMonitorNew(t *testing.T) {
	pm := NewPerformanceMonitor()

	assert.NotNil(t, pm)
	assert.NotNil(t, pm.collector)
}

func TestPerformanceMonitorRecordOperation(t *testing.T) {
	pm := NewPerformanceMonitor()

	duration := 100 * time.Millisecond
	pm.RecordOperation("test_operation", duration, map[string]string{"env": "test"})

	metrics := pm.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "operation_duration")
	assert.Contains(t, metrics, "operation_count")
}

func TestPerformanceMonitorRecordError(t *testing.T) {
	pm := NewPerformanceMonitor()

	pm.RecordError("test_operation", assert.AnError, map[string]string{"env": "test"})

	metrics := pm.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "operation_errors")
}

func TestPerformanceMonitorGetMetrics(t *testing.T) {
	pm := NewPerformanceMonitor()

	pm.RecordOperation("op1", 100*time.Millisecond, nil)
	pm.RecordOperation("op2", 200*time.Millisecond, nil)

	metrics := pm.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Greater(t, len(metrics), 0)
}

func TestLLMVerifierMetricsNew(t *testing.T) {
	lvm := NewLLMVerifierMetrics()

	assert.NotNil(t, lvm)
	assert.NotNil(t, lvm.performanceMonitor)
	assert.NotNil(t, lvm.collector)
}

func TestLLMVerifierMetricsRecordVerification(t *testing.T) {
	lvm := NewLLMVerifierMetrics()

	lvm.RecordVerification("gpt-4", "openai", 100*time.Millisecond, true)

	metrics := lvm.performanceMonitor.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "operation_duration")
}

func TestLLMVerifierMetricsRecordVerificationFailure(t *testing.T) {
	lvm := NewLLMVerifierMetrics()

	lvm.RecordVerification("gpt-4", "openai", 100*time.Millisecond, false)

	metrics := lvm.performanceMonitor.GetMetrics()

	assert.NotNil(t, metrics)
}

func TestLLMVerifierMetricsRecordModelScore(t *testing.T) {
	lvm := NewLLMVerifierMetrics()

	lvm.RecordModelScore("gpt-4", "openai", 95.5)

	metric, _ := lvm.collector.GetMetric("model_score", map[string]string{"model": "gpt-4", "provider": "openai"})

	assert.NotNil(t, metric)
	assert.Equal(t, 95.5, metric.Value)
}

func TestLLMVerifierMetricsRecordAPICall(t *testing.T) {
	lvm := NewLLMVerifierMetrics()

	lvm.RecordAPICall("openai", 100*time.Millisecond, true, 100)

	metrics := lvm.performanceMonitor.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "operation_duration")
}

func TestLLMVerifierMetricsRecordAPICallFailure(t *testing.T) {
	lvm := NewLLMVerifierMetrics()

	lvm.RecordAPICall("openai", 100*time.Millisecond, false, 0)

	metrics := lvm.performanceMonitor.GetMetrics()

	assert.NotNil(t, metrics)
}

func TestLLMVerifierMetricsGetSummary(t *testing.T) {
	lvm := NewLLMVerifierMetrics()

	lvm.RecordVerification("gpt-4", "openai", 100*time.Millisecond, true)
	lvm.RecordVerification("gpt-4", "openai", 100*time.Millisecond, false)

	summary := lvm.GetSummary()

	assert.NotNil(t, summary)
	assert.Contains(t, summary, "performance")
	assert.Contains(t, summary, "summary")
}

func TestLLMVerifierMetricsSummaryCalculations(t *testing.T) {
	lvm := NewLLMVerifierMetrics()

	// Record 3 successful and 1 failed verification
	lvm.RecordVerification("gpt-4", "openai", 100*time.Millisecond, true)
	lvm.RecordVerification("gpt-4", "openai", 100*time.Millisecond, true)
	lvm.RecordVerification("gpt-4", "openai", 100*time.Millisecond, true)
	lvm.RecordVerification("gpt-4", "openai", 100*time.Millisecond, false)

	summary := lvm.GetSummary()

	summaryData := summary["summary"].(map[string]any)

	assert.Equal(t, 4.0, summaryData["total_verifications"])
	assert.Equal(t, 3.0, summaryData["total_successes"])
	assert.Equal(t, 1.0, summaryData["total_failures"])
	assert.Equal(t, 75.0, summaryData["success_rate"])
}

func TestLLMVerifierMetricsSummaryEmpty(t *testing.T) {
	lvm := NewLLMVerifierMetrics()

	summary := lvm.GetSummary()

	assert.NotNil(t, summary)
	summaryData := summary["summary"].(map[string]any)
	assert.Equal(t, 0.0, summaryData["total_verifications"])
	assert.Equal(t, 0.0, summaryData["success_rate"])
}

func TestNewCriticalMetricsTracker(t *testing.T) {
	cmt := NewCriticalMetricsTracker()

	assert.NotNil(t, cmt)
	assert.NotNil(t, cmt.collector)
}

func TestCriticalMetricsTrackerGetPerformanceReport(t *testing.T) {
	cmt := NewCriticalMetricsTracker()

	report := cmt.GetPerformanceReport(nil)

	assert.NotNil(t, report)
	assert.Equal(t, "ok", report["status"])
}

func TestNewAlertManager(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)

	assert.NotNil(t, am)
	assert.NotNil(t, am.metricsTracker)
}

func TestAlertManagerGetActiveAlerts(t *testing.T) {
	mc := NewMetricsCollector()
	am := NewAlertManager(mc)

	alerts := am.GetActiveAlerts()

	assert.NotNil(t, alerts)
	assert.Equal(t, 0, len(alerts))
}

func TestMetricStruct(t *testing.T) {
	now := time.Now()
	labels := map[string]string{"env": "test", "region": "us-west"}

	metric := Metric{
		Name:        "test_metric",
		Type:        MetricTypeCounter,
		Value:       100.0,
		Labels:      labels,
		Timestamp:   now,
		Description: "Test metric",
	}

	assert.Equal(t, "test_metric", metric.Name)
	assert.Equal(t, MetricTypeCounter, metric.Type)
	assert.Equal(t, 100.0, metric.Value)
	assert.Equal(t, labels, metric.Labels)
	assert.Equal(t, now, metric.Timestamp)
	assert.Equal(t, "Test metric", metric.Description)
}

func TestAlertStruct(t *testing.T) {
	alert := Alert{
		ID:       "alert-123",
		Name:     "Test Alert",
		Rule:     "test_rule",
		Severity: AlertSeverityWarning,
		Message:  "This is a test alert",
		Time:     time.Now().Format(time.RFC3339),
		Active:   true,
	}

	assert.Equal(t, "alert-123", alert.ID)
	assert.Equal(t, "Test Alert", alert.Name)
	assert.Equal(t, "test_rule", alert.Rule)
	assert.Equal(t, AlertSeverityWarning, alert.Severity)
	assert.Equal(t, "This is a test alert", alert.Message)
	assert.True(t, alert.Active)
}

func TestMetricsCollectorWithDifferentLabels(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordCounter("test_counter", 1.0, map[string]string{"env": "test"}, "")
	mc.RecordCounter("test_counter", 2.0, map[string]string{"env": "prod"}, "")

	metrics := mc.GetMetricsByName("test_counter")

	assert.Equal(t, 2, len(metrics))
}

func TestPerformanceMonitorMultipleOperations(t *testing.T) {
	pm := NewPerformanceMonitor()

	pm.RecordOperation("op1", 100*time.Millisecond, nil)
	pm.RecordOperation("op1", 150*time.Millisecond, nil)
	pm.RecordOperation("op2", 200*time.Millisecond, nil)

	metrics := pm.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Greater(t, len(metrics), 0)
}

func TestMetricsCollectorGetMetricsByTypeMultiple(t *testing.T) {
	mc := NewMetricsCollector()

	mc.RecordCounter("c1", 1.0, nil, "")
	mc.RecordCounter("c2", 2.0, nil, "")
	mc.RecordGauge("g1", 10.0, nil, "")
	mc.RecordGauge("g2", 20.0, nil, "")

	counters := mc.GetMetricsByType(MetricTypeCounter)
	gauges := mc.GetMetricsByType(MetricTypeGauge)

	assert.Equal(t, 2, len(counters))
	assert.Equal(t, 2, len(gauges))
}

func TestPerformanceMonitorConcurrentOperations(t *testing.T) {
	pm := NewPerformanceMonitor()

	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func(index int) {
			defer wg.Done()
			pm.RecordOperation("op1", time.Duration(index)*time.Millisecond, nil)
		}(i)
		go func(index int) {
			defer wg.Done()
			pm.RecordError("op1", assert.AnError, nil)
		}(i)
	}

	wg.Wait()

	metrics := pm.GetMetrics()
	assert.NotNil(t, metrics)
}

func TestLLMVerifierMetricsMultipleProviders(t *testing.T) {
	lvm := NewLLMVerifierMetrics()

	providers := []string{"openai", "anthropic", "google"}
	for _, provider := range providers {
		lvm.RecordVerification("gpt-4", provider, 100*time.Millisecond, true)
		lvm.RecordAPICall(provider, 50*time.Millisecond, true, 100)
	}

	metrics := lvm.performanceMonitor.GetMetrics()
	assert.NotNil(t, metrics)
}

func TestLLMVerifierMetricsSuccessRate(t *testing.T) {
	lvm := NewLLMVerifierMetrics()

	// Record many verifications
	for i := 0; i < 100; i++ {
		success := i%4 != 0 // 25% failure rate
		lvm.RecordVerification("gpt-4", "openai", 100*time.Millisecond, success)
	}

	summary := lvm.GetSummary()
	summaryData := summary["summary"].(map[string]any)

	assert.Equal(t, 100.0, summaryData["total_verifications"])
	assert.InDelta(t, 75.0, summaryData["success_rate"], 1.0)
}
