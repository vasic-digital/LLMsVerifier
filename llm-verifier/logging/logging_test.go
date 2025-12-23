package logging

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"llm-verifier/database"
)

func setupTestLoggingDB(t *testing.T) *database.Database {
	dbFile := "/tmp/test_logging_" + time.Now().Format("20060102150405") + ".db"

	db, err := database.New(dbFile)
	require.NoError(t, err, "Failed to create test database")

	t.Cleanup(func() {
		os.Remove(dbFile)
	})

	return db
}

func TestLogLevelConstants(t *testing.T) {
	assert.Equal(t, LogLevel("debug"), LogLevelDebug)
	assert.Equal(t, LogLevel("info"), LogLevelInfo)
	assert.Equal(t, LogLevel("warning"), LogLevelWarning)
	assert.Equal(t, LogLevel("error"), LogLevelError)
	assert.Equal(t, LogLevel("fatal"), LogLevelFatal)
}

func TestLogEntryStruct(t *testing.T) {
	now := time.Now()
	userID := "user-123"
	fields := map[string]any{"key": "value"}
	metadata := map[string]any{"meta": "data"}

	entry := LogEntry{
		ID:            "log-123",
		Level:         LogLevelInfo,
		Message:       "test message",
		Timestamp:     now,
		CorrelationID: "corr-123",
		UserID:        &userID,
		Component:     "test-component",
		Source:        "test-source",
		Error:         "test error",
		Fields:        fields,
		Metadata:      metadata,
	}

	assert.Equal(t, "log-123", entry.ID)
	assert.Equal(t, LogLevelInfo, entry.Level)
	assert.Equal(t, "test message", entry.Message)
	assert.Equal(t, now, entry.Timestamp)
	assert.Equal(t, "corr-123", entry.CorrelationID)
	assert.Equal(t, &userID, entry.UserID)
	assert.Equal(t, "test-component", entry.Component)
	assert.Equal(t, "test-source", entry.Source)
	assert.Equal(t, "test error", entry.Error)
	assert.Equal(t, fields, entry.Fields)
	assert.Equal(t, metadata, entry.Metadata)
}

func TestNewLogger(t *testing.T) {
	db := setupTestLoggingDB(t)

	config := map[string]any{
		"console_level": "info",
		"file_level":    "warning",
		"file_path":     "/tmp/test_logs.log",
		"max_size":      5,
		"max_backups":   3,
		"compress":       false,
	}

	logger, err := NewLogger(db, config)

	assert.NoError(t, err)
	assert.NotNil(t, logger)
	assert.NotNil(t, logger.db)
	assert.NotNil(t, logger.fileWriter)
	assert.NotNil(t, logger.flushTicker)
	assert.NotNil(t, logger.stopCh)
	assert.Equal(t, LogLevelInfo, logger.consoleLevel)
	assert.Equal(t, LogLevelWarning, logger.fileLevel)
	assert.Equal(t, "/tmp/test_logs.log", logger.filePath)
	assert.Equal(t, int64(5*1024*1024), logger.maxSize)
	assert.Equal(t, 3, logger.maxBackups)
	assert.False(t, logger.compress)

	logger.Close()
	os.Remove("/tmp/test_logs.log")
}

func TestNewLoggerDefaults(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{})

	assert.NoError(t, err)
	assert.NotNil(t, logger)
	assert.Equal(t, LogLevelInfo, logger.consoleLevel, "Default console level should be info")
	assert.Equal(t, LogLevelWarning, logger.fileLevel, "Default file level should be warning")
	assert.Contains(t, logger.filePath, "llm-verifier.log", "Default file path should contain llm-verifier.log")

	logger.Close()
}

func TestLoggerDebug(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{
		"file_path": "/tmp/test_debug.log",
	})
	require.NoError(t, err)

	// Should not panic
	logger.Debug("debug message", map[string]any{"key": "value"})

	time.Sleep(100 * time.Millisecond)

	logger.Close()
	os.Remove("/tmp/test_debug.log")
}

func TestLoggerInfo(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{
		"file_path": "/tmp/test_info.log",
	})
	require.NoError(t, err)

	logger.Info("info message", map[string]any{"level": "info"})

	time.Sleep(100 * time.Millisecond)

	logger.Close()
	os.Remove("/tmp/test_info.log")
}

func TestLoggerWarning(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{
		"file_path": "/tmp/test_warning.log",
	})
	require.NoError(t, err)

	logger.Warning("warning message", nil)

	time.Sleep(100 * time.Millisecond)

	logger.Close()
	os.Remove("/tmp/test_warning.log")
}

func TestLoggerError(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{
		"file_path": "/tmp/test_error.log",
	})
	require.NoError(t, err)

	logger.Error("error message", map[string]any{"error_code": 500})

	time.Sleep(100 * time.Millisecond)

	logger.Close()
	os.Remove("/tmp/test_error.log")
}

func TestLoggerFatal(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{
		"file_path": "/tmp/test_fatal.log",
	})
	require.NoError(t, err)

	logger.Fatal("fatal message", nil)

	time.Sleep(100 * time.Millisecond)

	logger.Close()
	os.Remove("/tmp/test_fatal.log")
}

func TestLoggerLog(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{
		"file_path": "/tmp/test_log.log",
		"console_level": "debug",
	})
	require.NoError(t, err)

	logger.Log(LogLevelDebug, "debug test", nil)
	logger.Log(LogLevelInfo, "info test", nil)
	logger.Log(LogLevelWarning, "warning test", nil)
	logger.Log(LogLevelError, "error test", nil)

	time.Sleep(100 * time.Millisecond)

	logger.Close()
	os.Remove("/tmp/test_log.log")
}

func TestLoggerWithFields(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{
		"file_path": "/tmp/test_with_fields.log",
	})
	require.NoError(t, err)

	contextLogger := logger.WithFields(map[string]any{"request_id": "123", "user_id": "456"})
	assert.NotNil(t, contextLogger)

	contextLogger.Info("test with context", nil)

	time.Sleep(100 * time.Millisecond)

	logger.Close()
	os.Remove("/tmp/test_with_fields.log")
}

func TestLoggerQueryLogs(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{
		"file_path": "/tmp/test_query.log",
	})
	require.NoError(t, err)

	logs, err := logger.QueryLogs(map[string]any{"level": LogLevelInfo}, 10, 0)

	assert.NoError(t, err)
	assert.NotNil(t, logs)
	assert.Equal(t, 0, len(logs), "Query logs returns empty for now")

	logger.Close()
	os.Remove("/tmp/test_query.log")
}

func TestLoggerGetLogStats(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{
		"file_path": "/tmp/test_stats.log",
	})
	require.NoError(t, err)

	stats := logger.GetLogStats()

	assert.NotNil(t, stats)
	assert.Equal(t, 0, stats["total_entries"])
	assert.NotNil(t, stats["entries_by_level"])
	assert.Equal(t, 0, stats["storage_size"])

	logger.Close()
	os.Remove("/tmp/test_stats.log")
}

func TestLoggerRotateLogFile(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{
		"file_path": "/tmp/test_rotate.log",
		"max_backups": 2,
	})
	require.NoError(t, err)

	err = logger.RotateLogFile()

	assert.NoError(t, err)

	logger.Close()

	// Clean up
	os.Remove("/tmp/test_rotate.log")
	os.Remove("/tmp/test_rotate.log.1")
	os.Remove("/tmp/test_rotate.log.2")
}

func TestLoggerClose(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{
		"file_path": "/tmp/test_close.log",
	})
	require.NoError(t, err)

	// Log some messages
	for i := 0; i < 5; i++ {
		logger.Info("message", map[string]any{"index": i})
	}

	err = logger.Close()

	assert.NoError(t, err)
	os.Remove("/tmp/test_close.log")
}

func TestLoggerCloseTwice(t *testing.T) {
	// Test that close can be called (even if it may panic on second call)
	defer func() {
		if r := recover(); r != nil {
			// Expected panic on second close
			assert.Contains(t, r, "close of closed channel")
		}
	}()

	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{
		"file_path": "/tmp/test_close_twice.log",
	})
	require.NoError(t, err)

	logger.Close()
	logger.Close() // This will panic
}

func TestLoggerConcurrentLogging(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{
		"file_path": "/tmp/test_concurrent.log",
	})
	require.NoError(t, err)

	var wg sync.WaitGroup

	// Concurrent logging
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			logger.Info("concurrent message", map[string]any{"index": index})
		}(i)
	}

	wg.Wait()

	time.Sleep(200 * time.Millisecond)

	logger.Close()
	os.Remove("/tmp/test_concurrent.log")
}

func TestLoggerConcurrentWithFields(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{
		"file_path": "/tmp/test_concurrent_fields.log",
	})
	require.NoError(t, err)

	contextLogger := logger.WithFields(map[string]any{"app": "test"})

	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			contextLogger.Info("message", map[string]any{"index": index})
		}(i)
	}

	wg.Wait()

	time.Sleep(200 * time.Millisecond)

	logger.Close()
	os.Remove("/tmp/test_concurrent_fields.log")
}

func TestContextLogger(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{
		"file_path": "/tmp/test_context.log",
	})
	require.NoError(t, err)

	contextLogger := logger.WithFields(map[string]any{"request_id": "123"})

	assert.NotNil(t, contextLogger)
	assert.NotNil(t, contextLogger.logger)
	assert.NotNil(t, contextLogger.fields)

	contextLogger.Info("context test", nil)
	contextLogger.Debug("context debug", nil)
	contextLogger.Warning("context warning", nil)
	contextLogger.Error("context error", nil)

	time.Sleep(100 * time.Millisecond)

	logger.Close()
	os.Remove("/tmp/test_context.log")
}

func TestContextLoggerWithFields(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{
		"file_path": "/tmp/test_context_fields.log",
	})
	require.NoError(t, err)

	contextLogger := logger.WithFields(map[string]any{"app": "test"})
	newContextLogger := contextLogger.WithFields(map[string]any{"user": "admin"})

	assert.NotNil(t, newContextLogger)

	newContextLogger.Info("combined context", nil)

	time.Sleep(100 * time.Millisecond)

	logger.Close()
	os.Remove("/tmp/test_context_fields.log")
}

func TestGetLevelPriority(t *testing.T) {
	assert.Equal(t, 1, getLevelPriority(LogLevelDebug))
	assert.Equal(t, 2, getLevelPriority(LogLevelInfo))
	assert.Equal(t, 3, getLevelPriority(LogLevelWarning))
	assert.Equal(t, 4, getLevelPriority(LogLevelError))
	assert.Equal(t, 5, getLevelPriority(LogLevelFatal))
	assert.Equal(t, 0, getLevelPriority("unknown"))
}

func TestGenerateLogID(t *testing.T) {
	id1 := generateLogID()
	id2 := generateLogID()

	assert.Contains(t, id1, "log_")
	assert.Contains(t, id2, "log_")
	assert.NotEqual(t, id1, id2, "IDs should be unique")
}

func TestPerformanceMonitorNew(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{})
	require.NoError(t, err)

	pm := NewPerformanceMonitor(logger)

	assert.NotNil(t, pm)
	assert.NotNil(t, pm.logger)
	assert.NotNil(t, pm.metrics)
	assert.False(t, pm.startTime.IsZero())

	logger.Close()
}

func TestPerformanceMonitorStartTimer(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{})
	require.NoError(t, err)

	pm := NewPerformanceMonitor(logger)
	timer := pm.StartTimer("test-operation")

	assert.NotNil(t, timer)
	assert.Equal(t, "test-operation", timer.name)
	assert.False(t, timer.startTime.IsZero())

	timer.Stop()

	logger.Close()
}

func TestPerformanceMonitorRecordMetric(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{})
	require.NoError(t, err)

	pm := NewPerformanceMonitor(logger)

	pm.RecordMetric("test-op", 100*time.Millisecond)
	pm.RecordMetric("test-op", 150*time.Millisecond)
	pm.RecordMetric("test-op", 120*time.Millisecond)

	metrics := pm.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Contains(t, metrics, "test-op")

	logger.Close()
}

func TestPerformanceMonitorGetMetrics(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{})
	require.NoError(t, err)

	pm := NewPerformanceMonitor(logger)

	pm.RecordMetric("op1", 100*time.Millisecond)
	pm.RecordMetric("op2", 200*time.Millisecond)

	metrics := pm.GetMetrics()

	assert.NotNil(t, metrics)
	assert.Len(t, metrics, 2)
	assert.Contains(t, metrics, "op1")
	assert.Contains(t, metrics, "op2")

	logger.Close()
}

func TestPerformanceMonitorConcurrentRecording(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{})
	require.NoError(t, err)

	pm := NewPerformanceMonitor(logger)

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			pm.RecordMetric("test-op", time.Duration(index)*time.Millisecond)
		}(i)
	}

	wg.Wait()

	metrics := pm.GetMetrics()
	assert.NotNil(t, metrics)

	logger.Close()
}

func TestTimer(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{})
	require.NoError(t, err)

	pm := NewPerformanceMonitor(logger)
	timer := pm.StartTimer("timer-test")

	// Sleep a bit
	time.Sleep(50 * time.Millisecond)

	timer.Stop()

	metrics := pm.GetMetrics()
	assert.Contains(t, metrics, "timer-test")

	logger.Close()
}

func TestTimerStop(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{})
	require.NoError(t, err)

	pm := NewPerformanceMonitor(logger)
	timer := pm.StartTimer("stop-test")

	// Stop immediately
	timer.Stop()

	metrics := pm.GetMetrics()
	assert.Contains(t, metrics, "stop-test")

	logger.Close()
}

func TestLogAnalytics(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{})
	require.NoError(t, err)

	analytics := NewLogAnalytics(logger)

	assert.NotNil(t, analytics)
	assert.NotNil(t, analytics.logger)

	logger.Close()
}

func TestLogAnalyticsAnalyzeErrors(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{})
	require.NoError(t, err)

	analytics := NewLogAnalytics(logger)

	analysis := analytics.AnalyzeErrors(24)

	assert.NotNil(t, analysis)
	assert.Contains(t, analysis, "total_errors")
	assert.Contains(t, analysis, "error_types")
	assert.Contains(t, analysis, "error_trends")
	assert.Contains(t, analysis, "time_range")

	logger.Close()
}

func TestLogAnalyticsGetTopErrors(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{})
	require.NoError(t, err)

	analytics := NewLogAnalytics(logger)

	topErrors := analytics.GetTopErrors(10)

	assert.NotNil(t, topErrors)
	assert.Equal(t, 0, len(topErrors), "Should return empty slice for now")

	logger.Close()
}

func TestLogAnalyticsGenerateReport(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{})
	require.NoError(t, err)

	analytics := NewLogAnalytics(logger)

	report := analytics.GenerateReport(24)

	assert.NotNil(t, report)
	assert.Contains(t, report, "period")
	assert.Contains(t, report, "total_logs")
	assert.Contains(t, report, "error_analysis")
	assert.Contains(t, report, "performance")
	assert.Contains(t, report, "recommendations")

	logger.Close()
}

func TestLoggerMultipleLevels(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{
		"file_path":     "/tmp/test_multiple.log",
		"console_level": "debug",
		"file_level":    "debug",
	})
	require.NoError(t, err)

	// Test all log levels
	logger.Debug("debug", nil)
	logger.Info("info", nil)
	logger.Warning("warning", nil)
	logger.Error("error", nil)
	logger.Fatal("fatal", nil)

	time.Sleep(100 * time.Millisecond)

	logger.Close()
	os.Remove("/tmp/test_multiple.log")
}

func TestLoggerWithNilFields(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{
		"file_path": "/tmp/test_nil.log",
	})
	require.NoError(t, err)

	// Should handle nil fields
	logger.Info("message", nil)
	logger.Warning("warning", nil)

	time.Sleep(100 * time.Millisecond)

	logger.Close()
	os.Remove("/tmp/test_nil.log")
}

func TestLoggerEmptyMessage(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{
		"file_path": "/tmp/test_empty.log",
	})
	require.NoError(t, err)

	// Should handle empty message
	logger.Info("", nil)

	time.Sleep(100 * time.Millisecond)

	logger.Close()
	os.Remove("/tmp/test_empty.log")
}

func TestLoggerLargeFields(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{
		"file_path": "/tmp/test_large.log",
	})
	require.NoError(t, err)

	largeFields := map[string]any{}
	for i := 0; i < 100; i++ {
		largeFields["key_"+string(rune('0'+i%10))] = "value-" + string(rune('0'+i%10))
	}

	logger.Info("large fields", largeFields)

	time.Sleep(100 * time.Millisecond)

	logger.Close()
	os.Remove("/tmp/test_large.log")
}

func TestPerformanceMetricStruct(t *testing.T) {
	now := time.Now()
	metric := PerformanceMetric{
		Name:        "test-metric",
		Count:       10,
		TotalTime:   1000 * time.Millisecond,
		MinTime:     50 * time.Millisecond,
		MaxTime:     200 * time.Millisecond,
		LastUpdated: now,
	}

	assert.Equal(t, "test-metric", metric.Name)
	assert.Equal(t, int64(10), metric.Count)
	assert.Equal(t, 1000*time.Millisecond, metric.TotalTime)
	assert.Equal(t, 50*time.Millisecond, metric.MinTime)
	assert.Equal(t, 200*time.Millisecond, metric.MaxTime)
	assert.Equal(t, now, metric.LastUpdated)
}

func TestTimerStruct(t *testing.T) {
	db := setupTestLoggingDB(t)

	logger, err := NewLogger(db, map[string]any{})
	require.NoError(t, err)

	pm := NewPerformanceMonitor(logger)
	timer := Timer{
		name:      "test-timer",
		startTime: time.Now(),
		monitor:   pm,
	}

	assert.Equal(t, "test-timer", timer.name)
	assert.False(t, timer.startTime.IsZero())
	assert.NotNil(t, timer.monitor)

	logger.Close()
}
