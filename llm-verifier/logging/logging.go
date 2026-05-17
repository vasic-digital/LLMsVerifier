package logging

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"digital.vasic.llmsverifier/database"
)

// LogLevel represents the severity level of a log entry
type LogLevel string

const (
	LogLevelDebug   LogLevel = "debug"
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
	LogLevelFatal   LogLevel = "fatal"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	ID            string         `json:"id" db:"id"`
	Level         LogLevel       `json:"level" db:"level"`
	Message       string         `json:"message"`
	Timestamp     time.Time      `json:"timestamp"`
	CorrelationID string         `json:"correlation_id,omitempty"`
	UserID        *string        `json:"user_id,omitempty"`
	Component     string         `json:"component,omitempty"`
	Source        string         `json:"source,omitempty"`
	Error         string         `json:"error,omitempty"`
	Fields        map[string]any `json:"fields,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

// Logger manages structured logging with multiple outputs
type Logger struct {
	db           *database.Database
	consoleLevel LogLevel
	fileLevel    LogLevel
	fileWriter   *os.File
	filePath     string
	maxSize      int64 // Max size in bytes
	maxBackups   int
	compress     bool
	mu           sync.Mutex
	buffer       []*LogEntry
	bufferSize   int
	flushTicker  *time.Ticker
	stopCh       chan struct{}
	// In-memory log history for queryable persistence
	history      []*LogEntry
	historyMux   sync.RWMutex
	maxHistory   int
}

// NewLogger creates a new structured logger
func NewLogger(db *database.Database, config map[string]any) (*Logger, error) {
	logger := &Logger{
		db:         db,
		bufferSize: 100,
		buffer:     make([]*LogEntry, 0, 100),
		stopCh:     make(chan struct{}),
		history:    make([]*LogEntry, 0, 10000),
		maxHistory: 10000, // Keep last 10k log entries in memory
	}

	// Parse configuration
	if level, ok := config["console_level"].(string); ok {
		logger.consoleLevel = LogLevel(level)
	} else {
		logger.consoleLevel = LogLevelInfo
	}

	if level, ok := config["file_level"].(string); ok {
		logger.fileLevel = LogLevel(level)
	} else {
		logger.fileLevel = LogLevelWarning
	}

	if path, ok := config["file_path"].(string); ok {
		logger.filePath = path
	} else {
		logger.filePath = "logs/llm-verifier.log"
	}

	if maxSize, ok := config["max_size"].(int); ok {
		logger.maxSize = int64(maxSize) * 1024 * 1024 // Convert MB to bytes
	} else {
		logger.maxSize = 10 * 1024 * 1024 // 10MB default
	}

	if maxBackups, ok := config["max_backups"].(int); ok {
		logger.maxBackups = maxBackups
	} else {
		logger.maxBackups = 5
	}

	if compress, ok := config["compress"].(bool); ok {
		logger.compress = compress
	} else {
		logger.compress = true
	}

	// Create log directory
	if err := os.MkdirAll(filepath.Dir(logger.filePath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file
	file, err := os.OpenFile(logger.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	logger.fileWriter = file

	// Start background flush
	logger.flushTicker = time.NewTicker(30 * time.Second)
	go logger.flushWorker()

	return logger, nil
}

// Log logs a message with the specified level
func (l *Logger) Log(level LogLevel, message string, fields map[string]any) {
	entry := &LogEntry{
		ID:        generateLogID(),
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Source:    "app",
		Fields:    fields,
	}

	// Add to buffer
	l.mu.Lock()
	l.buffer = append(l.buffer, entry)

	// Flush if buffer is full
	if len(l.buffer) >= l.bufferSize {
		bufferCopy := make([]*LogEntry, len(l.buffer))
		copy(bufferCopy, l.buffer)
		l.buffer = l.buffer[:0]
		l.mu.Unlock()

		l.flushBuffer(bufferCopy)
	} else {
		l.mu.Unlock()
	}
}

// Debug logs a debug message
func (l *Logger) Debug(message string, fields map[string]any) {
	l.Log(LogLevelDebug, message, fields)
}

// Info logs an info message
func (l *Logger) Info(message string, fields map[string]any) {
	l.Log(LogLevelInfo, message, fields)
}

// Warning logs a warning message
func (l *Logger) Warning(message string, fields map[string]any) {
	l.Log(LogLevelWarning, message, fields)
}

// Error logs an error message
func (l *Logger) Error(message string, fields map[string]any) {
	l.Log(LogLevelError, message, fields)
}

// Fatal logs a fatal message
func (l *Logger) Fatal(message string, fields map[string]interface{}) {
	l.Log(LogLevelFatal, message, fields)
}

// WithFields creates a logger with pre-set fields
func (l *Logger) WithFields(fields map[string]any) *ContextLogger {
	return &ContextLogger{
		logger: l,
		fields: fields,
	}
}

// QueryLogs queries logs with filters
func (l *Logger) QueryLogs(filters map[string]any, limit int, offset int) ([]*LogEntry, error) {
	// If database is available, query from database
	if l.db != nil {
		dbFilters := make(map[string]interface{})

		// Convert filter keys
		if level, ok := filters["level"].(string); ok {
			dbFilters["level"] = strings.ToUpper(level)
		}
		if component, ok := filters["component"].(string); ok {
			dbFilters["logger"] = component
		}
		if correlationID, ok := filters["correlation_id"].(string); ok {
			dbFilters["request_id"] = correlationID
		}
		if search, ok := filters["search"].(string); ok {
			dbFilters["search"] = search
		}
		if fromDate, ok := filters["from_date"]; ok {
			dbFilters["from_date"] = fromDate
		}
		if toDate, ok := filters["to_date"]; ok {
			dbFilters["to_date"] = toDate
		}

		dbFilters["limit"] = limit

		dbLogs, err := l.db.ListLogs(dbFilters)
		if err != nil {
			return nil, fmt.Errorf("failed to query logs from database: %w", err)
		}

		// Convert database entries to logging entries
		result := make([]*LogEntry, 0, len(dbLogs))
		for _, dbLog := range dbLogs {
			entry := l.convertFromDBLogEntry(dbLog)
			result = append(result, entry)
		}

		// Apply offset
		if offset > 0 && offset < len(result) {
			result = result[offset:]
		} else if offset >= len(result) {
			return []*LogEntry{}, nil
		}

		return result, nil
	}

	// Fall back to in-memory query
	return l.queryInMemory(filters, limit, offset), nil
}

// convertFromDBLogEntry converts a database LogEntry to a logging LogEntry
func (l *Logger) convertFromDBLogEntry(dbEntry *database.LogEntry) *LogEntry {
	entry := &LogEntry{
		ID:        fmt.Sprintf("log_%d", dbEntry.ID),
		Level:     LogLevel(strings.ToLower(dbEntry.Level)),
		Message:   dbEntry.Message,
		Timestamp: dbEntry.Timestamp,
		Component: dbEntry.Logger,
		Source:    dbEntry.Logger,
	}

	if dbEntry.RequestID != nil {
		entry.CorrelationID = *dbEntry.RequestID
	}

	// Parse details JSON into fields
	if dbEntry.Details != nil && *dbEntry.Details != "" {
		var fields map[string]any
		if err := json.Unmarshal([]byte(*dbEntry.Details), &fields); err == nil {
			entry.Fields = fields
		}
	}

	return entry
}

// queryInMemory queries logs from in-memory history
func (l *Logger) queryInMemory(filters map[string]any, limit int, offset int) []*LogEntry {
	l.historyMux.RLock()
	defer l.historyMux.RUnlock()

	var result []*LogEntry

	for _, entry := range l.history {
		// Apply filters
		if level, ok := filters["level"].(string); ok {
			if string(entry.Level) != strings.ToLower(level) {
				continue
			}
		}
		if component, ok := filters["component"].(string); ok {
			if entry.Component != component && entry.Source != component {
				continue
			}
		}
		if correlationID, ok := filters["correlation_id"].(string); ok {
			if entry.CorrelationID != correlationID {
				continue
			}
		}
		if search, ok := filters["search"].(string); ok {
			searchLower := strings.ToLower(search)
			if !strings.Contains(strings.ToLower(entry.Message), searchLower) &&
				!strings.Contains(strings.ToLower(entry.Component), searchLower) {
				continue
			}
		}
		if fromDate, ok := filters["from_date"].(time.Time); ok {
			if entry.Timestamp.Before(fromDate) {
				continue
			}
		}
		if toDate, ok := filters["to_date"].(time.Time); ok {
			if entry.Timestamp.After(toDate) {
				continue
			}
		}

		result = append(result, entry)
	}

	// Apply offset and limit
	if offset > 0 && offset < len(result) {
		result = result[offset:]
	} else if offset >= len(result) {
		return []*LogEntry{}
	}

	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}

	return result
}

// GetLogStats returns logging statistics from the in-memory history
func (l *Logger) GetLogStats() map[string]any {
	l.historyMux.RLock()
	defer l.historyMux.RUnlock()

	byLevel := map[string]int{
		"debug":   0,
		"info":    0,
		"warning": 0,
		"error":   0,
		"fatal":   0,
	}
	byComponent := make(map[string]int)

	var oldestEntry, newestEntry *LogEntry
	for _, entry := range l.history {
		byLevel[string(entry.Level)]++
		if entry.Component != "" {
			byComponent[entry.Component]++
		}
		if oldestEntry == nil || entry.Timestamp.Before(oldestEntry.Timestamp) {
			oldestEntry = entry
		}
		if newestEntry == nil || entry.Timestamp.After(newestEntry.Timestamp) {
			newestEntry = entry
		}
	}

	// Estimate storage size in bytes (rough estimate based on entry count)
	// ~256 bytes per log entry on average
	estimatedStorageSize := len(l.history) * 256

	stats := map[string]any{
		"total_entries":    len(l.history),
		"entries_by_level": byLevel,
		"storage_size":     estimatedStorageSize,
		"oldest_entry":     nil,
		"newest_entry":     nil,
	}

	if oldestEntry != nil {
		stats["oldest_entry"] = oldestEntry.Timestamp
	}
	if newestEntry != nil {
		stats["newest_entry"] = newestEntry.Timestamp
	}

	return stats
}

// RotateLogFile rotates the current log file
func (l *Logger) RotateLogFile() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.fileWriter.Close(); err != nil {
		return fmt.Errorf("failed to close current log file: %w", err)
	}

	// Rotate existing files
	for i := l.maxBackups - 1; i >= 0; i-- {
		oldPath := l.filePath
		if i > 0 {
			oldPath = fmt.Sprintf("%s.%d", l.filePath, i)
		}
		newPath := fmt.Sprintf("%s.%d", l.filePath, i+1)

		if _, err := os.Stat(oldPath); err == nil {
			if err := os.Rename(oldPath, newPath); err != nil {
				log.Printf("Failed to rotate log file %s: %v", oldPath, err)
			}
		}
	}

	// Open new log file
	file, err := os.OpenFile(l.filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %w", err)
	}
	l.fileWriter = file

	log.Printf("Log file rotated: %s", l.filePath)
	return nil
}

// Close closes the logger and flushes all pending entries
func (l *Logger) Close() error {
	close(l.stopCh)
	l.flushTicker.Stop()

	// Flush remaining buffer
	l.mu.Lock()
	bufferCopy := make([]*LogEntry, len(l.buffer))
	copy(bufferCopy, l.buffer)
	l.buffer = nil
	l.mu.Unlock()

	if len(bufferCopy) > 0 {
		l.flushBuffer(bufferCopy)
	}

	return l.fileWriter.Close()
}

// Private methods

func (l *Logger) flushWorker() {
	for {
		select {
		case <-l.stopCh:
			return
		case <-l.flushTicker.C:
			l.mu.Lock()
			if len(l.buffer) > 0 {
				bufferCopy := make([]*LogEntry, len(l.buffer))
				copy(bufferCopy, l.buffer)
				l.buffer = l.buffer[:0]
				l.mu.Unlock()

				l.flushBuffer(bufferCopy)
			} else {
				l.mu.Unlock()
			}
		}
	}
}

func (l *Logger) flushBuffer(entries []*LogEntry) {
	for _, entry := range entries {
		// Write to console if level is high enough
		if l.shouldLogToConsole(entry.Level) {
			l.writeToConsole(entry)
		}

		// Write to file if level is high enough
		if l.shouldLogToFile(entry.Level) {
			l.writeToFile(entry)
		}

		// Store in database (placeholder)
		l.storeInDatabase(entry)
	}
}

func (l *Logger) shouldLogToConsole(level LogLevel) bool {
	return getLevelPriority(level) >= getLevelPriority(l.consoleLevel)
}

func (l *Logger) shouldLogToFile(level LogLevel) bool {
	return getLevelPriority(level) >= getLevelPriority(l.fileLevel)
}

func (l *Logger) writeToConsole(entry *LogEntry) {
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
	level := strings.ToUpper(string(entry.Level))
	source := entry.Source

	var color string
	switch entry.Level {
	case LogLevelDebug:
		color = "\033[36m" // Cyan
	case LogLevelInfo:
		color = "\033[32m" // Green
	case LogLevelWarning:
		color = "\033[33m" // Yellow
	case LogLevelError:
		color = "\033[31m" // Red
	case LogLevelFatal:
		color = "\033[35m" // Magenta
	default:
		color = "\033[0m" // Reset
	}

	fmt.Printf("%s [%s%s\033[0m] %s: %s", timestamp, color, level, source, entry.Message)

	if len(entry.Fields) > 0 {
		fieldsJSON, _ := json.Marshal(entry.Fields)
		fmt.Printf(" %s", string(fieldsJSON))
	}

	fmt.Println()
}

func (l *Logger) writeToFile(entry *LogEntry) {
	entryJSON, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Failed to marshal log entry: %v", err)
		return
	}

	if _, err := l.fileWriter.Write(append(entryJSON, '\n')); err != nil {
		log.Printf("Failed to write to log file: %v", err)

		// Try to rotate if file is too large
		if stat, err := l.fileWriter.Stat(); err == nil && stat.Size() > l.maxSize {
			l.RotateLogFile()
		}
	}
}

func (l *Logger) storeInDatabase(entry *LogEntry) {
	// Store in memory history
	l.historyMux.Lock()
	l.history = append(l.history, entry)
	if len(l.history) > l.maxHistory {
		l.history = l.history[len(l.history)-l.maxHistory:]
	}
	l.historyMux.Unlock()

	// Store in database if available
	if l.db != nil {
		dbEntry := l.convertToDBLogEntry(entry)
		if err := l.db.CreateLog(dbEntry); err != nil {
			// Don't use logger here to avoid infinite recursion
			log.Printf("Failed to store log in database: %v", err)
		}
	}
}

// convertToDBLogEntry converts a logging LogEntry to a database LogEntry
func (l *Logger) convertToDBLogEntry(entry *LogEntry) *database.LogEntry {
	dbEntry := &database.LogEntry{
		Timestamp: entry.Timestamp,
		Level:     string(entry.Level),
		Logger:    entry.Component,
		Message:   entry.Message,
	}

	// Set source as logger if component is empty
	if dbEntry.Logger == "" {
		dbEntry.Logger = entry.Source
	}

	// Set correlation ID as request ID
	if entry.CorrelationID != "" {
		dbEntry.RequestID = &entry.CorrelationID
	}

	// Convert fields to details JSON
	if len(entry.Fields) > 0 {
		if details, err := json.Marshal(entry.Fields); err == nil {
			detailsStr := string(details)
			dbEntry.Details = &detailsStr
		}
	}

	return dbEntry
}

// GetLogHistory returns recent log entries from memory
func (l *Logger) GetLogHistory(limit int) []*LogEntry {
	l.historyMux.RLock()
	defer l.historyMux.RUnlock()

	if limit <= 0 || limit > len(l.history) {
		limit = len(l.history)
	}

	// Return most recent entries
	start := len(l.history) - limit
	if start < 0 {
		start = 0
	}

	result := make([]*LogEntry, limit)
	copy(result, l.history[start:])
	return result
}

// GetLogsByLevel returns log entries filtered by level
func (l *Logger) GetLogsByLevel(level LogLevel, limit int) []*LogEntry {
	l.historyMux.RLock()
	defer l.historyMux.RUnlock()

	var result []*LogEntry
	// Iterate from most recent
	for i := len(l.history) - 1; i >= 0 && (limit <= 0 || len(result) < limit); i-- {
		if l.history[i].Level == level {
			result = append(result, l.history[i])
		}
	}
	return result
}

// GetLogsByCorrelationID returns log entries for a specific correlation ID
func (l *Logger) GetLogsByCorrelationID(correlationID string) []*LogEntry {
	l.historyMux.RLock()
	defer l.historyMux.RUnlock()

	var result []*LogEntry
	for _, entry := range l.history {
		if entry.CorrelationID == correlationID {
			result = append(result, entry)
		}
	}
	return result
}


func getLevelPriority(level LogLevel) int {
	switch level {
	case LogLevelDebug:
		return 1
	case LogLevelInfo:
		return 2
	case LogLevelWarning:
		return 3
	case LogLevelError:
		return 4
	case LogLevelFatal:
		return 5
	default:
		return 0
	}
}

func generateLogID() string {
	return fmt.Sprintf("log_%d", time.Now().UnixNano())
}

// ContextLogger provides logging with pre-set context fields
type ContextLogger struct {
	logger *Logger
	fields map[string]any
}

// Debug logs a debug message with context
func (cl *ContextLogger) Debug(message string, extraFields map[string]any) {
	fields := make(map[string]any)
	for k, v := range cl.fields {
		fields[k] = v
	}
	for k, v := range extraFields {
		fields[k] = v
	}
	cl.logger.Debug(message, fields)
}

// Info logs an info message with context
func (cl *ContextLogger) Info(message string, extraFields map[string]any) {
	fields := make(map[string]any)
	for k, v := range cl.fields {
		fields[k] = v
	}
	for k, v := range extraFields {
		fields[k] = v
	}
	cl.logger.Info(message, fields)
}

// Warning logs a warning message with context
func (cl *ContextLogger) Warning(message string, extraFields map[string]any) {
	fields := make(map[string]any)
	for k, v := range cl.fields {
		fields[k] = v
	}
	for k, v := range extraFields {
		fields[k] = v
	}
	cl.logger.Warning(message, fields)
}

// Error logs an error message with context
func (cl *ContextLogger) Error(message string, extraFields map[string]any) {
	fields := make(map[string]any)
	for k, v := range cl.fields {
		fields[k] = v
	}
	for k, v := range extraFields {
		fields[k] = v
	}
	cl.logger.Error(message, fields)
}

// WithFields adds more context fields
func (cl *ContextLogger) WithFields(fields map[string]any) *ContextLogger {
	newFields := make(map[string]any)
	for k, v := range cl.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}
	return &ContextLogger{
		logger: cl.logger,
		fields: newFields,
	}
}

// PerformanceMonitor monitors system performance
type PerformanceMonitor struct {
	logger    *Logger
	metrics   map[string]*PerformanceMetric
	startTime time.Time
	mu        sync.RWMutex
}

// PerformanceMetric represents a performance metric
type PerformanceMetric struct {
	Name        string
	Count       int64
	TotalTime   time.Duration
	MinTime     time.Duration
	MaxTime     time.Duration
	LastUpdated time.Time
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(logger *Logger) *PerformanceMonitor {
	return &PerformanceMonitor{
		logger:    logger,
		metrics:   make(map[string]*PerformanceMetric),
		startTime: time.Now(),
	}
}

// StartTimer starts a performance timer
func (pm *PerformanceMonitor) StartTimer(name string) *Timer {
	return &Timer{
		name:      name,
		startTime: time.Now(),
		monitor:   pm,
	}
}

// RecordMetric records a performance metric
func (pm *PerformanceMonitor) RecordMetric(name string, duration time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.metrics[name] == nil {
		pm.metrics[name] = &PerformanceMetric{
			Name:    name,
			Count:   0,
			MinTime: time.Hour, // Initialize to large value
		}
	}

	metric := pm.metrics[name]
	metric.Count++
	metric.TotalTime += duration
	metric.LastUpdated = time.Now()

	if duration < metric.MinTime {
		metric.MinTime = duration
	}
	if duration > metric.MaxTime {
		metric.MaxTime = duration
	}
}

// GetMetrics returns all performance metrics
func (pm *PerformanceMonitor) GetMetrics() map[string]any {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	metrics := make(map[string]any)
	for name, metric := range pm.metrics {
		avgTime := time.Duration(0)
		if metric.Count > 0 {
			avgTime = metric.TotalTime / time.Duration(metric.Count)
		}

		metrics[name] = map[string]any{
			"count":        metric.Count,
			"total_time":   metric.TotalTime.String(),
			"avg_time":     avgTime.String(),
			"min_time":     metric.MinTime.String(),
			"max_time":     metric.MaxTime.String(),
			"last_updated": metric.LastUpdated.Format(time.RFC3339),
		}
	}

	return metrics
}

// Timer represents a performance timer
type Timer struct {
	name      string
	startTime time.Time
	monitor   *PerformanceMonitor
}

// Stop stops the timer and records the metric
func (t *Timer) Stop() {
	duration := time.Since(t.startTime)
	t.monitor.RecordMetric(t.name, duration)
}

// LogAnalytics provides log analysis capabilities
type LogAnalytics struct {
	logger *Logger
}

// NewLogAnalytics creates a new log analytics instance
func NewLogAnalytics(logger *Logger) *LogAnalytics {
	return &LogAnalytics{logger: logger}
}

// AnalyzeErrors previously returned hardcoded `total_errors: 0 +
// empty maps + computed time_range string` regardless of actual log
// data — §11.4 stub-interface bluff. The accompanying tests
// (logging_test.go) ASSERTED `len(errors) == 0` which CERTIFIED
// the bluff (per CONST-035: tests that pass on placeholder
// behaviour ARE the bluff infrastructure).
//
// Real implementation requires either (a) the log store backend
// (SQLite/Postgres) exposing an aggregation query API, or (b) the
// Logger keeping an in-memory ring buffer of recent error entries
// for analytics consumption. Until the storage layer is wired, this
// function emits a §11.4-disclosed placeholder result with explicit
// `not_wired: true` marker so any caller can detect the gap by
// checking that field instead of asserting on zero counts.
func (la *LogAnalytics) AnalyzeErrors(hours int) map[string]any {
	return map[string]any{
		"not_wired":    true,
		"reason":       "log store aggregation API or in-memory analytics buffer not yet wired into LogAnalytics — §11.4 PASS-bluff if consumer indexes total_errors/error_types as real data",
		"total_errors": 0,
		"error_types":  map[string]int{},
		"error_trends": []any{},
		"time_range":   fmt.Sprintf("last %d hours", hours),
	}
}

// GetTopErrors previously returned `[]map[string]any{}` (always
// empty) regardless of `limit` — §11.4 stub-interface bluff at the
// analytics-API layer. Until the storage layer is wired, emit a
// single-element list with `not_wired: true` marker so consumers
// can detect the gap programmatically (asserting "len == limit"
// will then correctly fail, surfacing the wiring gap at the
// assertion boundary).
func (la *LogAnalytics) GetTopErrors(limit int) []map[string]any {
	return []map[string]any{{
		"not_wired": true,
		"reason":    "log store aggregation API not wired (limit=" + fmt.Sprintf("%d", limit) + " ignored — §11.4 PASS-bluff if consumer asserts top-N error patterns)",
	}}
}

// GenerateReport generates a comprehensive logging report
func (la *LogAnalytics) GenerateReport(hours int) map[string]any {
	return map[string]any{
		"period":         fmt.Sprintf("last %d hours", hours),
		"total_logs":     0,
		"error_analysis": la.AnalyzeErrors(hours),
		"performance":    map[string]any{},
		"recommendations": []string{
			"Consider increasing log retention for better analysis",
			"Monitor error rates and set up alerts",
			"Review performance metrics regularly",
		},
	}
}
