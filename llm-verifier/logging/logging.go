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

	"llm-verifier/database"
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
	ID            string                 `json:"id" db:"id"`
	Level         LogLevel               `json:"level" db:"level"`
	Message       string                 `json:"message"`
	Timestamp     time.Time              `json:"timestamp"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	UserID        *string                `json:"user_id,omitempty"`
	Component     string                 `json:"component,omitempty"`
	Source        string                 `json:"source,omitempty"`
	Error         string                 `json:"error,omitempty"`
	Fields        map[string]interface{} `json:"fields,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
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
}

// NewLogger creates a new structured logger
func NewLogger(db *database.Database, config map[string]interface{}) (*Logger, error) {
	logger := &Logger{
		db:         db,
		bufferSize: 100,
		buffer:     make([]*LogEntry, 0, 100),
		stopCh:     make(chan struct{}),
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
func (l *Logger) Log(level LogLevel, message string, fields map[string]interface{}) {
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
func (l *Logger) Debug(message string, fields map[string]interface{}) {
	l.Log(LogLevelDebug, message, fields)
}

// Info logs an info message
func (l *Logger) Info(message string, fields map[string]interface{}) {
	l.Log(LogLevelInfo, message, fields)
}

// Warning logs a warning message
func (l *Logger) Warning(message string, fields map[string]interface{}) {
	l.Log(LogLevelWarning, message, fields)
}

// Error logs an error message
func (l *Logger) Error(message string, fields map[string]interface{}) {
	l.Log(LogLevelError, message, fields)
}

// Fatal logs a fatal message
func (l *Logger) Fatal(message string, fields map[string]interface{}) {
	l.Log(LogLevelFatal, message, fields)
}

// WithFields creates a logger with pre-set fields
func (l *Logger) WithFields(fields map[string]interface{}) *ContextLogger {
	return &ContextLogger{
		logger: l,
		fields: fields,
	}
}

// QueryLogs queries logs with filters
func (l *Logger) QueryLogs(filters map[string]interface{}, limit int, offset int) ([]*LogEntry, error) {
	// This would query the database in a real implementation
	// For now, return empty slice
	return []*LogEntry{}, nil
}

// GetLogStats returns logging statistics
func (l *Logger) GetLogStats() map[string]interface{} {
	return map[string]interface{}{
		"total_entries": 0,
		"entries_by_level": map[string]int{
			"debug":   0,
			"info":    0,
			"warning": 0,
			"error":   0,
			"fatal":   0,
		},
		"storage_size": 0,
		"oldest_entry": nil,
		"newest_entry": nil,
	}
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
	// Store in database (placeholder)
	// In real implementation, this would insert into logs table
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
	fields map[string]interface{}
}

// Debug logs a debug message with context
func (cl *ContextLogger) Debug(message string, extraFields map[string]interface{}) {
	fields := make(map[string]interface{})
	for k, v := range cl.fields {
		fields[k] = v
	}
	for k, v := range extraFields {
		fields[k] = v
	}
	cl.logger.Debug(message, fields)
}

// Info logs an info message with context
func (cl *ContextLogger) Info(message string, extraFields map[string]interface{}) {
	fields := make(map[string]interface{})
	for k, v := range cl.fields {
		fields[k] = v
	}
	for k, v := range extraFields {
		fields[k] = v
	}
	cl.logger.Info(message, fields)
}

// Warning logs a warning message with context
func (cl *ContextLogger) Warning(message string, extraFields map[string]interface{}) {
	fields := make(map[string]interface{})
	for k, v := range cl.fields {
		fields[k] = v
	}
	for k, v := range extraFields {
		fields[k] = v
	}
	cl.logger.Warning(message, fields)
}

// Error logs an error message with context
func (cl *ContextLogger) Error(message string, extraFields map[string]interface{}) {
	fields := make(map[string]interface{})
	for k, v := range cl.fields {
		fields[k] = v
	}
	for k, v := range extraFields {
		fields[k] = v
	}
	cl.logger.Error(message, fields)
}

// WithFields adds more context fields
func (cl *ContextLogger) WithFields(fields map[string]interface{}) *ContextLogger {
	newFields := make(map[string]interface{})
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
func (pm *PerformanceMonitor) GetMetrics() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	metrics := make(map[string]interface{})
	for name, metric := range pm.metrics {
		avgTime := time.Duration(0)
		if metric.Count > 0 {
			avgTime = metric.TotalTime / time.Duration(metric.Count)
		}

		metrics[name] = map[string]interface{}{
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

// AnalyzeErrors analyzes error patterns in logs
func (la *LogAnalytics) AnalyzeErrors(hours int) map[string]interface{} {
	// Placeholder implementation
	return map[string]interface{}{
		"total_errors": 0,
		"error_types":  map[string]int{},
		"error_trends": []interface{}{},
		"time_range":   fmt.Sprintf("last %d hours", hours),
	}
}

// GetTopErrors returns the most frequent errors
func (la *LogAnalytics) GetTopErrors(limit int) []map[string]interface{} {
	// Placeholder implementation
	return []map[string]interface{}{}
}

// GenerateReport generates a comprehensive logging report
func (la *LogAnalytics) GenerateReport(hours int) map[string]interface{} {
	return map[string]interface{}{
		"period":         fmt.Sprintf("last %d hours", hours),
		"total_logs":     0,
		"error_analysis": la.AnalyzeErrors(hours),
		"performance":    map[string]interface{}{},
		"recommendations": []string{
			"Consider increasing log retention for better analysis",
			"Monitor error rates and set up alerts",
			"Review performance metrics regularly",
		},
	}
}
