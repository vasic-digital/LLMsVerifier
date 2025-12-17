package database

import (
	"database/sql"
	"fmt"
	"strings"
)

// ==================== Event CRUD Operations ====================

// CreateEvent creates a new event
func (d *Database) CreateEvent(event *Event) error {
	var detailsJSON string
	if event.Details != nil {
		detailsJSON = *event.Details
	}

	query := `
		INSERT INTO events (
			event_type, severity, title, message, details, model_id,
			provider_id, verification_result_id, issue_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := d.conn.Exec(query,
		event.EventType,
		event.Severity,
		event.Title,
		event.Message,
		detailsJSON,
		event.ModelID,
		event.ProviderID,
		event.VerificationResultID,
		event.IssueID,
	)

	if err != nil {
		return fmt.Errorf("failed to create event: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert ID: %w", err)
	}

	event.ID = id
	return nil
}

// GetEvent retrieves an event by ID
func (d *Database) GetEvent(id int64) (*Event, error) {
	query := `
		SELECT id, event_type, severity, title, message, details, model_id,
			provider_id, verification_result_id, issue_id, created_at
		FROM events WHERE id = ?
	`

	var event Event
	var detailsJSON sql.NullString

	err := d.conn.QueryRow(query, id).Scan(
		&event.ID,
		&event.EventType,
		&event.Severity,
		&event.Title,
		&event.Message,
		&detailsJSON,
		&event.ModelID,
		&event.ProviderID,
		&event.VerificationResultID,
		&event.IssueID,
		&event.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("event not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	// Parse details
	if detailsJSON.Valid && detailsJSON.String != "" {
		detailsStr := detailsJSON.String
		event.Details = &detailsStr
	}

	return &event, nil
}

// ListEvents retrieves events with optional filtering
func (d *Database) ListEvents(filters map[string]any) ([]*Event, error) {
	query := `
		SELECT id, event_type, severity, title, message, details, model_id,
			provider_id, verification_result_id, issue_id, created_at
		FROM events
	`

	var conditions []string
	var args []any

	// Add conditions based on filters
	if eventType, ok := filters["event_type"]; ok {
		conditions = append(conditions, "event_type = ?")
		args = append(args, eventType)
	}

	if severity, ok := filters["severity"]; ok {
		conditions = append(conditions, "severity = ?")
		args = append(args, severity)
	}

	if modelID, ok := filters["model_id"]; ok {
		conditions = append(conditions, "model_id = ?")
		args = append(args, modelID)
	}

	if providerID, ok := filters["provider_id"]; ok {
		conditions = append(conditions, "provider_id = ?")
		args = append(args, providerID)
	}

	if fromDate, ok := filters["from_date"]; ok {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, fromDate)
	}

	if toDate, ok := filters["to_date"]; ok {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, toDate)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC"

	if limit, ok := filters["limit"]; ok {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		var event Event
		var detailsJSON sql.NullString

		err := rows.Scan(
			&event.ID,
			&event.EventType,
			&event.Severity,
			&event.Title,
			&event.Message,
			&detailsJSON,
			&event.ModelID,
			&event.ProviderID,
			&event.VerificationResultID,
			&event.IssueID,
			&event.CreatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		// Parse details
		if detailsJSON.Valid && detailsJSON.String != "" {
			detailsStr := detailsJSON.String
			event.Details = &detailsStr
		}

		events = append(events, &event)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating events: %w", err)
	}

	return events, nil
}

// UpdateEvent updates an existing event
func (d *Database) UpdateEvent(event *Event) error {
	var detailsJSON string
	if event.Details != nil {
		detailsJSON = *event.Details
	}

	query := `
		UPDATE events SET
			event_type = ?, severity = ?, title = ?, message = ?, details = ?,
			model_id = ?, provider_id = ?, verification_result_id = ?,
			issue_id = ?
		WHERE id = ?
	`

	_, err := d.conn.Exec(query,
		event.EventType,
		event.Severity,
		event.Title,
		event.Message,
		detailsJSON,
		event.ModelID,
		event.ProviderID,
		event.VerificationResultID,
		event.IssueID,
		event.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update event: %w", err)
	}

	return nil
}

// DeleteEvent deletes an event by ID
func (d *Database) DeleteEvent(id int64) error {
	query := `DELETE FROM events WHERE id = ?`

	_, err := d.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	return nil
}

// GetEventsByType gets events filtered by type
func (d *Database) GetEventsByType(eventType string, limit int) ([]*Event, error) {
	filters := map[string]any{
		"event_type": eventType,
		"limit":      limit,
	}

	return d.ListEvents(filters)
}

// GetRecentEvents gets recent events across all types
func (d *Database) GetRecentEvents(limit int) ([]*Event, error) {
	filters := map[string]any{
		"limit": limit,
	}

	return d.ListEvents(filters)
}
