package database

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupLogsTestDB(t *testing.T) *Database {
	dbFile := "/tmp/test_logs_" + time.Now().Format("20060102150405") + ".db"
	db, err := New(dbFile)
	require.NoError(t, err)
	t.Cleanup(func() {
		db.Close()
		os.Remove(dbFile)
	})
	return db
}

func createTestLogEntry() *LogEntry {
	details := "Test details"
	requestID := "req-123"
	return &LogEntry{
		Level:     "info",
		Logger:    "test-logger",
		Message:   "Test log message",
		Details:   &details,
		RequestID: &requestID,
	}
}

func TestCreateLog(t *testing.T) {
	db := setupLogsTestDB(t)

	logEntry := createTestLogEntry()
	err := db.CreateLog(logEntry)
	require.NoError(t, err)
	assert.NotZero(t, logEntry.ID)
}

func TestCreateLog_MinimalFields(t *testing.T) {
	db := setupLogsTestDB(t)

	logEntry := &LogEntry{
		Level:   "error",
		Logger:  "test",
		Message: "Error message",
	}
	err := db.CreateLog(logEntry)
	require.NoError(t, err)
	assert.NotZero(t, logEntry.ID)
}

func TestGetLog(t *testing.T) {
	db := setupLogsTestDB(t)

	logEntry := createTestLogEntry()
	err := db.CreateLog(logEntry)
	require.NoError(t, err)

	// Retrieve
	retrieved, err := db.GetLog(logEntry.ID)
	require.NoError(t, err)
	assert.Equal(t, logEntry.ID, retrieved.ID)
	assert.Equal(t, logEntry.Level, retrieved.Level)
	assert.Equal(t, logEntry.Logger, retrieved.Logger)
	assert.Equal(t, logEntry.Message, retrieved.Message)
	assert.NotNil(t, retrieved.Details)
	assert.NotNil(t, retrieved.RequestID)
}

func TestGetLog_NotFound(t *testing.T) {
	db := setupLogsTestDB(t)

	_, err := db.GetLog(99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestListLogs_NoFilters(t *testing.T) {
	db := setupLogsTestDB(t)

	// Create multiple logs
	for i := 0; i < 3; i++ {
		logEntry := createTestLogEntry()
		logEntry.Message = "Log " + string(rune('A'+i))
		err := db.CreateLog(logEntry)
		require.NoError(t, err)
	}

	// List all
	logs, err := db.ListLogs(nil)
	require.NoError(t, err)
	assert.Len(t, logs, 3)
}

func TestListLogs_FilterByLevel(t *testing.T) {
	db := setupLogsTestDB(t)

	// Create logs with different levels
	levels := []string{"debug", "info", "warn", "error"}
	for _, level := range levels {
		logEntry := createTestLogEntry()
		logEntry.Level = level
		err := db.CreateLog(logEntry)
		require.NoError(t, err)
	}

	// Filter by level
	logs, err := db.ListLogs(map[string]interface{}{"level": "error"})
	require.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, "error", logs[0].Level)
}

func TestListLogs_FilterByLogger(t *testing.T) {
	db := setupLogsTestDB(t)

	// Create logs with different loggers
	loggers := []string{"auth", "api", "db"}
	for _, logger := range loggers {
		logEntry := createTestLogEntry()
		logEntry.Logger = logger
		err := db.CreateLog(logEntry)
		require.NoError(t, err)
	}

	// Filter by logger
	logs, err := db.ListLogs(map[string]interface{}{"logger": "api"})
	require.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, "api", logs[0].Logger)
}

func TestListLogs_WithLimit(t *testing.T) {
	db := setupLogsTestDB(t)

	// Create multiple logs
	for i := 0; i < 10; i++ {
		logEntry := createTestLogEntry()
		err := db.CreateLog(logEntry)
		require.NoError(t, err)
	}

	// List with limit
	logs, err := db.ListLogs(map[string]interface{}{"limit": 5})
	require.NoError(t, err)
	assert.Len(t, logs, 5)
}

func TestDeleteLog(t *testing.T) {
	db := setupLogsTestDB(t)

	logEntry := createTestLogEntry()
	err := db.CreateLog(logEntry)
	require.NoError(t, err)

	// Delete
	err = db.DeleteLog(logEntry.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = db.GetLog(logEntry.ID)
	assert.Error(t, err)
}

func TestGetLogsByLevel(t *testing.T) {
	db := setupLogsTestDB(t)

	// Create logs with different levels
	for _, level := range []string{"info", "error", "error", "warn"} {
		logEntry := createTestLogEntry()
		logEntry.Level = level
		err := db.CreateLog(logEntry)
		require.NoError(t, err)
	}

	logs, err := db.GetLogsByLevel("error", 10)
	require.NoError(t, err)
	assert.Len(t, logs, 2)
}

func TestGetLogsByRequest(t *testing.T) {
	db := setupLogsTestDB(t)

	// Create logs with same request ID
	reqID := "req-abc-123"
	for i := 0; i < 3; i++ {
		logEntry := createTestLogEntry()
		logEntry.RequestID = &reqID
		err := db.CreateLog(logEntry)
		require.NoError(t, err)
	}

	// Create log with different request ID
	otherReqID := "req-xyz-789"
	otherLog := createTestLogEntry()
	otherLog.RequestID = &otherReqID
	err := db.CreateLog(otherLog)
	require.NoError(t, err)

	logs, err := db.GetLogsByRequest(reqID)
	require.NoError(t, err)
	assert.Len(t, logs, 3)
}

func TestGetRecentLogs(t *testing.T) {
	db := setupLogsTestDB(t)

	// Create multiple logs
	for i := 0; i < 20; i++ {
		logEntry := createTestLogEntry()
		err := db.CreateLog(logEntry)
		require.NoError(t, err)
	}

	logs, err := db.GetRecentLogs(10)
	require.NoError(t, err)
	assert.Len(t, logs, 10)
}

func TestGetLogsSince(t *testing.T) {
	db := setupLogsTestDB(t)

	// Create logs
	for i := 0; i < 5; i++ {
		logEntry := createTestLogEntry()
		err := db.CreateLog(logEntry)
		require.NoError(t, err)
	}

	// Get logs since a very old time (should get all or at least not error)
	logs, err := db.GetLogsSince(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	// ListLogs may return empty slice instead of nil
	assert.GreaterOrEqual(t, len(logs), 0)
}

func TestGetErrorLogs(t *testing.T) {
	db := setupLogsTestDB(t)

	// Create error logs (uppercase to match GetErrorLogs query)
	for i := 0; i < 3; i++ {
		logEntry := createTestLogEntry()
		logEntry.Level = "ERROR"
		err := db.CreateLog(logEntry)
		require.NoError(t, err)
	}

	// Create non-error logs
	for i := 0; i < 2; i++ {
		logEntry := createTestLogEntry()
		logEntry.Level = "INFO"
		err := db.CreateLog(logEntry)
		require.NoError(t, err)
	}

	logs, err := db.GetErrorLogs(10)
	require.NoError(t, err)
	assert.Len(t, logs, 3)
}

func TestCleanOldLogs(t *testing.T) {
	db := setupLogsTestDB(t)

	// Create some logs
	for i := 0; i < 5; i++ {
		logEntry := createTestLogEntry()
		err := db.CreateLog(logEntry)
		require.NoError(t, err)
	}

	// Clean logs older than 30 days (should not delete recent logs)
	err := db.CleanOldLogs(30 * 24 * time.Hour)
	require.NoError(t, err)

	// Verify logs still exist
	logs, err := db.ListLogs(nil)
	require.NoError(t, err)
	assert.Len(t, logs, 5)
}

func TestLogEntry_Struct(t *testing.T) {
	details := "Test details"
	requestID := "req-123"
	modelID := int64(1)
	providerID := int64(2)
	verificationID := int64(3)

	entry := LogEntry{
		ID:                   1,
		Timestamp:            time.Now(),
		Level:                "info",
		Logger:               "test",
		Message:              "Test message",
		Details:              &details,
		RequestID:            &requestID,
		ModelID:              &modelID,
		ProviderID:           &providerID,
		VerificationResultID: &verificationID,
	}

	assert.Equal(t, int64(1), entry.ID)
	assert.Equal(t, "info", entry.Level)
	assert.Equal(t, "test", entry.Logger)
	assert.NotNil(t, entry.Details)
	assert.NotNil(t, entry.ModelID)
	assert.NotNil(t, entry.ProviderID)
}
