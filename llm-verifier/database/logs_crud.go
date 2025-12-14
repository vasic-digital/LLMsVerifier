package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// ==================== Log CRUD Operations ====================

// CreateLog creates a new log entry
func (d *Database) CreateLog(logEntry *LogEntry) error {
	query := `
		INSERT INTO logs (
			level, logger, message, details, request_id, user_id,
			model_id, provider_id, verification_result_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := d.conn.Exec(query,
		logEntry.Level,
		logEntry.Logger,
		logEntry.Message,
		logEntry.Details,
		logEntry.RequestID,
		logEntry.UserID,
		logEntry.ModelID,
		logEntry.ProviderID,
		logEntry.VerificationResultID,
	)

	if err != nil {
		return fmt.Errorf("failed to create log entry: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	logEntry.ID = id
	return nil
}

// GetLog retrieves a log entry by ID
func (d *Database) GetLog(id int64) (*LogEntry, error) {
	query := `
		SELECT id, timestamp, level, logger, message, details, request_id,
			user_id, model_id, provider_id, verification_result_id
		FROM logs WHERE id = ?
	`

	var logEntry LogEntry
	var details, requestID sql.NullString

	err := d.conn.QueryRow(query, id).Scan(
		&logEntry.ID,
		&logEntry.Timestamp,
		&logEntry.Level,
		&logEntry.Logger,
		&logEntry.Message,
		&details,
		&requestID,
		&logEntry.UserID,
		&logEntry.ModelID,
		&logEntry.ProviderID,
		&logEntry.VerificationResultID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("log entry not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get log entry: %w", err)
	}

	// Handle nullable fields
	if details.Valid {
		logEntry.Details = &details.String
	}
	if requestID.Valid {
		logEntry.RequestID = &requestID.String
	}

	return &logEntry, nil
}

// ListLogs retrieves log entries with optional filtering
func (d *Database) ListLogs(filters map[string]interface{}) ([]*LogEntry, error) {
	query := `
		SELECT id, timestamp, level, logger, message, details, request_id,
			user_id, model_id, provider_id, verification_result_id
		FROM logs
	`

	var conditions []string
	var args []interface{}

	// Add conditions based on filters
	if level, ok := filters["level"]; ok {
		conditions = append(conditions, "level = ?")
		args = append(args, level)
	}

	if logger, ok := filters["logger"]; ok {
		conditions = append(conditions, "logger = ?")
		args = append(args, logger)
	}

	if requestID, ok := filters["request_id"]; ok {
		conditions = append(conditions, "request_id = ?")
		args = append(args, requestID)
	}

	if modelID, ok := filters["model_id"]; ok {
		conditions = append(conditions, "model_id = ?")
		args = append(args, modelID)
	}

	if providerID, ok := filters["provider_id"]; ok {
		conditions = append(conditions, "provider_id = ?")
		args = append(args, providerID)
	}

	if verificationResultID, ok := filters["verification_result_id"]; ok {
		conditions = append(conditions, "verification_result_id = ?")
		args = append(args, verificationResultID)
	}

	if fromDate, ok := filters["from_date"]; ok {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, fromDate)
	}

	if toDate, ok := filters["to_date"]; ok {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, toDate)
	}

	if search, ok := filters["search"]; ok {
		conditions = append(conditions, "(message LIKE ? OR logger LIKE ?)")
		searchPattern := fmt.Sprintf("%%%s%%", search)
		args = append(args, searchPattern, searchPattern)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY timestamp DESC"

	if limit, ok := filters["limit"]; ok {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list logs: %w", err)
	}
	defer rows.Close()

	var logs []*LogEntry
	for rows.Next() {
		var logEntry LogEntry
		var details, requestID sql.NullString

		err := rows.Scan(
			&logEntry.ID,
			&logEntry.Timestamp,
			&logEntry.Level,
			&logEntry.Logger,
			&logEntry.Message,
			&details,
			&requestID,
			&logEntry.UserID,
			&logEntry.ModelID,
			&logEntry.ProviderID,
			&logEntry.VerificationResultID,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan log entry: %w", err)
		}

		// Handle nullable fields
		if details.Valid {
			logEntry.Details = &details.String
		}
		if requestID.Valid {
			logEntry.RequestID = &requestID.String
		}

		logs = append(logs, &logEntry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating logs: %w", err)
	}

	return logs, nil
}

// DeleteLog deletes a log entry by ID
func (d *Database) DeleteLog(id int64) error {
	query := `DELETE FROM logs WHERE id = ?`

	_, err := d.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete log entry: %w", err)
	}

	return nil
}

// GetLogsByLevel gets log entries filtered by level
func (d *Database) GetLogsByLevel(level string, limit int) ([]*LogEntry, error) {
	filters := map[string]interface{}{
		"level": level,
		"limit": limit,
	}

	return d.ListLogs(filters)
}

// GetLogsByRequest gets all logs for a specific request
func (d *Database) GetLogsByRequest(requestID string) ([]*LogEntry, error) {
	filters := map[string]interface{}{
		"request_id": requestID,
	}

	return d.ListLogs(filters)
}

// GetRecentLogs gets recent log entries across all levels
func (d *Database) GetRecentLogs(limit int) ([]*LogEntry, error) {
	filters := map[string]interface{}{
		"limit": limit,
	}

	return d.ListLogs(filters)
}

// GetLogsSince gets log entries since a specific timestamp
func (d *Database) GetLogsSince(since time.Time) ([]*LogEntry, error) {
	filters := map[string]interface{}{
		"from_date": since,
	}

	return d.ListLogs(filters)
}

// GetErrorLogs gets all error and critical level logs
func (d *Database) GetErrorLogs(limit int) ([]*LogEntry, error) {
	query := `
		SELECT id, timestamp, level, logger, message, details, request_id,
			user_id, model_id, provider_id, verification_result_id
		FROM logs
		WHERE level IN ('ERROR', 'CRITICAL')
		ORDER BY timestamp DESC
		LIMIT ?
	`

	rows, err := d.conn.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get error logs: %w", err)
	}
	defer rows.Close()

	var logs []*LogEntry
	for rows.Next() {
		var logEntry LogEntry
		var details, requestID sql.NullString

		err := rows.Scan(
			&logEntry.ID,
			&logEntry.Timestamp,
			&logEntry.Level,
			&logEntry.Logger,
			&logEntry.Message,
			&details,
			&requestID,
			&logEntry.UserID,
			&logEntry.ModelID,
			&logEntry.ProviderID,
			&logEntry.VerificationResultID,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan error log: %w", err)
		}

		// Handle nullable fields
		if details.Valid {
			logEntry.Details = &details.String
		}
		if requestID.Valid {
			logEntry.RequestID = &requestID.String
		}

		logs = append(logs, &logEntry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating error logs: %w", err)
	}

	return logs, nil
}

// CleanOldLogs removes log entries older than the specified duration
func (d *Database) CleanOldLogs(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	query := `DELETE FROM logs WHERE timestamp < ?`

	_, err := d.conn.Exec(query, cutoff)
	if err != nil {
		return fmt.Errorf("failed to clean old logs: %w", err)
	}

	return nil
}
