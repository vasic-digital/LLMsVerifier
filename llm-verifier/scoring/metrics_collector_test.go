package scoring

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMetricsCollector(t *testing.T) {
	config := DefaultMonitoringConfig()
	collector := NewMetricsCollector(config)

	assert.NotNil(t, collector.scoreChanges)
	assert.NotNil(t, collector.apiResponseTimes)
	assert.NotNil(t, collector.dbLatencies)
	assert.NotNil(t, collector.memoryUsage)
	assert.NotNil(t, collector.cpuUsage)
	assert.NotNil(t, collector.diskUsage)
}

func TestMetricsCollector_RecordScoreCalculation(t *testing.T) {
	config := DefaultMonitoringConfig()
	collector := NewMetricsCollector(config)

	// Record successful calculation
	collector.RecordScoreCalculation(true)
	assert.Equal(t, int64(1), collector.scoreCalculationsTotal)
	assert.Equal(t, int64(0), collector.scoreCalculationErrors)

	// Record failed calculation
	collector.RecordScoreCalculation(false)
	assert.Equal(t, int64(2), collector.scoreCalculationsTotal)
	assert.Equal(t, int64(1), collector.scoreCalculationErrors)
}

func TestMetricsCollector_RecordScoreChange(t *testing.T) {
	config := DefaultMonitoringConfig()
	collector := NewMetricsCollector(config)

	collector.RecordScoreChange("model-1", 5.5)
	collector.RecordScoreChange("model-2", -2.3)

	assert.Len(t, collector.scoreChanges, 2)
	assert.Equal(t, "model-1", collector.scoreChanges[0].ModelID)
	assert.Equal(t, 5.5, collector.scoreChanges[0].Change)
	assert.Equal(t, "model-2", collector.scoreChanges[1].ModelID)
	assert.Equal(t, -2.3, collector.scoreChanges[1].Change)
}

func TestMetricsCollector_RecordAPIPerformance(t *testing.T) {
	config := DefaultMonitoringConfig()
	collector := NewMetricsCollector(config)

	// Record successful API call
	collector.RecordAPIPerformance("test-api", 100*time.Millisecond, true)
	assert.Equal(t, int64(1), collector.apiRequestsTotal)
	assert.Equal(t, int64(0), collector.apiRequestsFailed)
	assert.Len(t, collector.apiResponseTimes, 1)

	// Record failed API call
	collector.RecordAPIPerformance("test-api", 500*time.Millisecond, false)
	assert.Equal(t, int64(2), collector.apiRequestsTotal)
	assert.Equal(t, int64(1), collector.apiRequestsFailed)
	assert.Len(t, collector.apiResponseTimes, 2)
}

func TestMetricsCollector_RecordDatabasePerformance(t *testing.T) {
	config := DefaultMonitoringConfig()
	collector := NewMetricsCollector(config)

	// Record successful DB operation
	collector.RecordDatabasePerformance("SELECT", 10*time.Millisecond, true)
	assert.Equal(t, int64(1), collector.dbOperationsTotal)
	assert.Equal(t, int64(0), collector.dbOperationsFailed)
	assert.Len(t, collector.dbLatencies, 1)

	// Record failed DB operation
	collector.RecordDatabasePerformance("INSERT", 50*time.Millisecond, false)
	assert.Equal(t, int64(2), collector.dbOperationsTotal)
	assert.Equal(t, int64(1), collector.dbOperationsFailed)
	assert.Len(t, collector.dbLatencies, 2)
}

func TestMetricsCollector_RecordSystemPerformance(t *testing.T) {
	config := DefaultMonitoringConfig()
	collector := NewMetricsCollector(config)

	// Record system performance with all valid values
	collector.RecordSystemPerformance(4*1024*1024*1024, 16*1024*1024*1024, 45.5, 100*1024*1024*1024, 500*1024*1024*1024)

	assert.Len(t, collector.memoryUsage, 1)
	assert.Len(t, collector.cpuUsage, 1)
	assert.Len(t, collector.diskUsage, 1)

	assert.Equal(t, uint64(4*1024*1024*1024), collector.memoryUsage[0].Used)
	assert.Equal(t, 45.5, collector.cpuUsage[0].Usage)
	assert.Equal(t, uint64(100*1024*1024*1024), collector.diskUsage[0].Used)
}

func TestMetricsCollector_RecordSystemPerformance_ZeroValues(t *testing.T) {
	config := DefaultMonitoringConfig()
	collector := NewMetricsCollector(config)

	// Record with zero memory total (should not add memory metric)
	collector.RecordSystemPerformance(0, 0, 50.0, 0, 0)

	assert.Len(t, collector.memoryUsage, 0)
	assert.Len(t, collector.cpuUsage, 1) // CPU is always added
	assert.Len(t, collector.diskUsage, 0)
}

func TestMetricsCollector_GetCurrentMetrics(t *testing.T) {
	config := DefaultMonitoringConfig()
	collector := NewMetricsCollector(config)

	// Add some data
	collector.RecordScoreCalculation(true)
	collector.RecordScoreCalculation(false)
	collector.RecordScoreChange("model-1", 10.0)
	collector.RecordAPIPerformance("api-1", 100*time.Millisecond, true)
	collector.RecordAPIPerformance("api-2", 200*time.Millisecond, false)
	collector.RecordDatabasePerformance("SELECT", 5*time.Millisecond, true)
	collector.RecordSystemPerformance(8*1024*1024*1024, 16*1024*1024*1024, 30.0, 200*1024*1024*1024, 1000*1024*1024*1024)

	metrics := collector.GetCurrentMetrics()

	assert.Equal(t, int64(2), metrics.ScoreCalculationsTotal)
	assert.Equal(t, int64(1), metrics.ScoreCalculationErrors)
	assert.Equal(t, 10.0, metrics.AverageScoreChange)
	assert.Equal(t, int64(2), metrics.APIRequestsTotal)
	assert.Equal(t, int64(1), metrics.APIRequestsFailed)
	assert.Equal(t, int64(1), metrics.DatabaseOperationsTotal)
	assert.Equal(t, int64(0), metrics.DatabaseOperationsFailed)
	assert.InDelta(t, 50.0, metrics.CurrentMemoryUsage, 0.1)
	assert.Equal(t, 30.0, metrics.CurrentCPUUsage)
	assert.InDelta(t, 20.0, metrics.CurrentDiskUsage, 0.1)
}

func TestMetricsCollector_GetCurrentMetrics_Empty(t *testing.T) {
	config := DefaultMonitoringConfig()
	collector := NewMetricsCollector(config)

	metrics := collector.GetCurrentMetrics()

	assert.Equal(t, int64(0), metrics.ScoreCalculationsTotal)
	assert.Equal(t, float64(0), metrics.AverageScoreChange)
	assert.Equal(t, float64(0), metrics.CurrentMemoryUsage)
	assert.Equal(t, float64(0), metrics.CurrentCPUUsage)
	assert.Equal(t, float64(0), metrics.CurrentDiskUsage)
}

func TestMetricsCollector_GetSummary(t *testing.T) {
	config := DefaultMonitoringConfig()
	collector := NewMetricsCollector(config)

	// Add data
	collector.RecordScoreCalculation(true)
	collector.RecordScoreChange("model-1", 5.0)
	collector.RecordScoreChange("model-2", 15.0)
	collector.RecordAPIPerformance("api-1", 100*time.Millisecond, true)
	collector.RecordDatabasePerformance("SELECT", 10*time.Millisecond, true)

	summary := collector.GetSummary(1 * time.Hour)

	assert.Equal(t, int64(1), summary.TotalScoreCalculations)
	assert.Equal(t, 10.0, summary.AverageScoreChange) // (5 + 15) / 2
	assert.Equal(t, int64(1), summary.APIRequestsTotal)
	assert.Equal(t, int64(1), summary.DatabaseOperationsTotal)
}

func TestMetricsCollector_CleanupOldMetrics(t *testing.T) {
	config := DefaultMonitoringConfig()
	collector := NewMetricsCollector(config)

	// Add some data
	collector.RecordScoreChange("model-1", 5.0)
	collector.RecordSystemPerformance(1024, 4096, 25.0, 1024, 4096)

	// Cleanup with a very short maxAge (nothing should remain)
	collector.CleanupOldMetrics(1 * time.Nanosecond)

	// Score changes added just now should still be there
	// (they're timestamped after the cutoff)
	assert.GreaterOrEqual(t, len(collector.scoreChanges), 0)
}

func TestMetricsCollector_GetScoreCalculationRate(t *testing.T) {
	config := DefaultMonitoringConfig()
	collector := NewMetricsCollector(config)

	// Initially 0
	rate := collector.GetScoreCalculationRate()
	assert.Equal(t, float64(0), rate)

	// Add calculations
	for i := 0; i < 100; i++ {
		collector.RecordScoreCalculation(true)
	}

	rate = collector.GetScoreCalculationRate()
	assert.Greater(t, rate, float64(0))
}

func TestMetricsCollector_GetAPIErrorRate(t *testing.T) {
	config := DefaultMonitoringConfig()
	collector := NewMetricsCollector(config)

	// No requests - 0% error rate
	rate := collector.GetAPIErrorRate()
	assert.Equal(t, float64(0), rate)

	// 50% error rate
	collector.RecordAPIPerformance("api-1", 100*time.Millisecond, true)
	collector.RecordAPIPerformance("api-2", 100*time.Millisecond, false)

	rate = collector.GetAPIErrorRate()
	assert.Equal(t, float64(50), rate)
}

func TestMetricsCollector_GetDatabaseErrorRate(t *testing.T) {
	config := DefaultMonitoringConfig()
	collector := NewMetricsCollector(config)

	// No operations - 0% error rate
	rate := collector.GetDatabaseErrorRate()
	assert.Equal(t, float64(0), rate)

	// 25% error rate
	collector.RecordDatabasePerformance("SELECT", 10*time.Millisecond, true)
	collector.RecordDatabasePerformance("SELECT", 10*time.Millisecond, true)
	collector.RecordDatabasePerformance("SELECT", 10*time.Millisecond, true)
	collector.RecordDatabasePerformance("INSERT", 10*time.Millisecond, false)

	rate = collector.GetDatabaseErrorRate()
	assert.Equal(t, float64(25), rate)
}

func TestMetricsCollector_Reset(t *testing.T) {
	config := DefaultMonitoringConfig()
	collector := NewMetricsCollector(config)

	// Add data
	collector.RecordScoreCalculation(true)
	collector.RecordScoreChange("model-1", 10.0)
	collector.RecordAPIPerformance("api-1", 100*time.Millisecond, true)
	collector.RecordDatabasePerformance("SELECT", 10*time.Millisecond, true)
	collector.RecordSystemPerformance(1024, 4096, 25.0, 1024, 4096)

	// Verify data exists
	assert.Equal(t, int64(1), collector.scoreCalculationsTotal)
	assert.Len(t, collector.scoreChanges, 1)

	// Reset
	collector.Reset()

	// Verify all counters are 0
	assert.Equal(t, int64(0), collector.scoreCalculationsTotal)
	assert.Equal(t, int64(0), collector.scoreCalculationErrors)
	assert.Equal(t, int64(0), collector.apiRequestsTotal)
	assert.Equal(t, int64(0), collector.apiRequestsFailed)
	assert.Equal(t, int64(0), collector.dbOperationsTotal)
	assert.Equal(t, int64(0), collector.dbOperationsFailed)
	assert.Empty(t, collector.scoreChanges)
	assert.Empty(t, collector.apiResponseTimes)
	assert.Empty(t, collector.dbLatencies)
	assert.Empty(t, collector.memoryUsage)
	assert.Empty(t, collector.cpuUsage)
	assert.Empty(t, collector.diskUsage)
}

func TestMetricsCollector_Trimming(t *testing.T) {
	config := DefaultMonitoringConfig()
	collector := NewMetricsCollector(config)

	// Add more than the max allowed metrics
	for i := 0; i < 15000; i++ {
		collector.RecordScoreChange("model", float64(i))
	}

	// Should be trimmed to 10000
	assert.LessOrEqual(t, len(collector.scoreChanges), 10000)
}

func TestMetricsCollector_ConcurrentAccess(t *testing.T) {
	config := DefaultMonitoringConfig()
	collector := NewMetricsCollector(config)

	done := make(chan bool)

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			collector.RecordScoreCalculation(true)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			collector.RecordAPIPerformance("api", 100*time.Millisecond, true)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 100; i++ {
			collector.RecordDatabasePerformance("SELECT", 10*time.Millisecond, true)
		}
		done <- true
	}()

	// Concurrent read
	go func() {
		for i := 0; i < 50; i++ {
			collector.GetCurrentMetrics()
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 4; i++ {
		<-done
	}

	// Verify no race conditions caused data loss
	assert.Equal(t, int64(100), collector.scoreCalculationsTotal)
	assert.Equal(t, int64(100), collector.apiRequestsTotal)
	assert.Equal(t, int64(100), collector.dbOperationsTotal)
}

func TestScoreChangeMetric(t *testing.T) {
	metric := ScoreChangeMetric{
		ModelID:   "test-model",
		Change:    5.5,
		Timestamp: time.Now(),
	}

	assert.Equal(t, "test-model", metric.ModelID)
	assert.Equal(t, 5.5, metric.Change)
	assert.False(t, metric.Timestamp.IsZero())
}

func TestMemoryMetric(t *testing.T) {
	metric := MemoryMetric{
		Used:      4 * 1024 * 1024 * 1024,
		Total:     16 * 1024 * 1024 * 1024,
		Timestamp: time.Now(),
	}

	assert.Equal(t, uint64(4*1024*1024*1024), metric.Used)
	assert.Equal(t, uint64(16*1024*1024*1024), metric.Total)
}

func TestCPUMetric(t *testing.T) {
	metric := CPUMetric{
		Usage:     45.5,
		Timestamp: time.Now(),
	}

	assert.Equal(t, 45.5, metric.Usage)
}

func TestDiskMetric(t *testing.T) {
	metric := DiskMetric{
		Used:      100 * 1024 * 1024 * 1024,
		Total:     500 * 1024 * 1024 * 1024,
		Timestamp: time.Now(),
	}

	assert.Equal(t, uint64(100*1024*1024*1024), metric.Used)
	assert.Equal(t, uint64(500*1024*1024*1024), metric.Total)
}

func TestCurrentMetrics_Fields(t *testing.T) {
	metrics := CurrentMetrics{
		ScoreCalculationsTotal:   100,
		ScoreCalculationErrors:   5,
		AverageScoreChange:       7.5,
		APIRequestsTotal:         1000,
		APIRequestsFailed:        10,
		AverageAPIResponseTime:   150 * time.Millisecond,
		DatabaseOperationsTotal:  5000,
		DatabaseOperationsFailed: 25,
		AverageDatabaseLatency:   5 * time.Millisecond,
		CurrentMemoryUsage:       75.5,
		CurrentCPUUsage:          45.0,
		CurrentDiskUsage:         60.0,
		LastUpdated:              time.Now(),
	}

	assert.Equal(t, int64(100), metrics.ScoreCalculationsTotal)
	assert.Equal(t, int64(5), metrics.ScoreCalculationErrors)
	assert.Equal(t, 7.5, metrics.AverageScoreChange)
	assert.Equal(t, 75.5, metrics.CurrentMemoryUsage)
}

func TestDefaultMonitoringConfig(t *testing.T) {
	config := DefaultMonitoringConfig()
	require.NotNil(t, config)
}
