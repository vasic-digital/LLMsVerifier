package events

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// EventStore handles persistence of events to database
type EventStore struct {
	db *database.Database
}

// NewEventStore creates a new event store
func NewEventStore(db *database.Database) *EventStore {
	return &EventStore{db: db}
}

// StoreEvent stores an event in the database
func (es *EventStore) StoreEvent(event *Event) error {
	return es.db.WithTransaction(func(tx *sql.Tx) error {
		detailsJSON, err := json.Marshal(event.Details)
		if err != nil {
			return fmt.Errorf("failed to marshal event details: %w", err)
		}

		query := `
			INSERT INTO events (
				event_type, severity, title, message, details, model_id, provider_id,
				verification_result_id, issue_id, client_id, user_id, source, timestamp
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`

		_, err = tx.Exec(query,
			string(event.Type),
			string(event.Severity),
			event.Title,
			event.Message,
			string(detailsJSON),
			event.ModelID,
			event.ProviderID,
			event.VerificationID,
			event.IssueID,
			event.ClientID,
			event.UserID,
			event.Source,
			event.Timestamp,
		)

		return err
	})
}

// GetEvents retrieves events with optional filtering
func (es *EventStore) GetEvents(filters map[string]interface{}) ([]*Event, error) {
	query := `
		SELECT id, event_type, severity, title, message, details, model_id, provider_id,
			verification_result_id, issue_id, client_id, user_id, source, timestamp, processed_at
		FROM events WHERE 1=1
	`

	var args []interface{}

	if eventType, ok := filters["event_type"]; ok && eventType != "" {
		query += " AND event_type = ?"
		args = append(args, eventType)
	}

	if severity, ok := filters["severity"]; ok && severity != "" {
		query += " AND severity = ?"
		args = append(args, severity)
	}

	if modelID, ok := filters["model_id"]; ok && modelID != nil {
		query += " AND model_id = ?"
		args = append(args, modelID)
	}

	if providerID, ok := filters["provider_id"]; ok && providerID != nil {
		query += " AND provider_id = ?"
		args = append(args, providerID)
	}

	if clientID, ok := filters["client_id"]; ok && clientID != "" {
		query += " AND client_id = ?"
		args = append(args, clientID)
	}

	if startTime, ok := filters["start_time"]; ok {
		query += " AND timestamp >= ?"
		args = append(args, startTime)
	}

	if endTime, ok := filters["end_time"]; ok {
		query += " AND timestamp <= ?"
		args = append(args, endTime)
	}

	limit := 100 // Default limit
	if limitVal, ok := filters["limit"]; ok {
		if l, ok := limitVal.(int); ok && l > 0 && l <= 1000 {
			limit = l
		}
	}

	query += " ORDER BY timestamp DESC LIMIT ?"
	args = append(args, limit)

	rows, err := es.db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []*Event
	for rows.Next() {
		var event Event
		var detailsJSON string
		var processedAt sql.NullTime

		err := rows.Scan(
			&event.ID,
			(*string)(&event.Type),
			(*string)(&event.Severity),
			&event.Title,
			&event.Message,
			&detailsJSON,
			&event.ModelID,
			&event.ProviderID,
			&event.VerificationID,
			&event.IssueID,
			&event.ClientID,
			&event.UserID,
			&event.Source,
			&event.Timestamp,
			&processedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		if processedAt.Valid {
			event.ProcessedAt = &processedAt.Time
		}

		// Unmarshal details
		if detailsJSON != "" {
			err = json.Unmarshal([]byte(detailsJSON), &event.Details)
			if err != nil {
				// Log error but continue
				fmt.Printf("Warning: failed to unmarshal event details: %v\n", err)
			}
		}

		events = append(events, &event)
	}

	return events, nil
}

// GetEventByID retrieves a specific event by ID
func (es *EventStore) GetEventByID(eventID string) (*Event, error) {
	query := `
		SELECT id, event_type, severity, title, message, details, model_id, provider_id,
			verification_result_id, issue_id, client_id, user_id, source, timestamp, processed_at
		FROM events WHERE id = ?
	`

	var event Event
	var detailsJSON string
	var processedAt sql.NullTime

	err := es.db.conn.QueryRow(query, eventID).Scan(
		&event.ID,
		(*string)(&event.Type),
		(*string)(&event.Severity),
		&event.Title,
		&event.Message,
		&detailsJSON,
		&event.ModelID,
		&event.ProviderID,
		&event.VerificationID,
		&event.IssueID,
		&event.ClientID,
		&event.UserID,
		&event.Source,
		&event.Timestamp,
		&processedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get event: %w", err)
	}

	if processedAt.Valid {
		event.ProcessedAt = &processedAt.Time
	}

	// Unmarshal details
	if detailsJSON != "" {
		err = json.Unmarshal([]byte(detailsJSON), &event.Details)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal event details: %w", err)
		}
	}

	return &event, nil
}

// DeleteOldEvents removes events older than the specified duration
func (es *EventStore) DeleteOldEvents(olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)

	return es.db.WithTransaction(func(tx *sql.Tx) error {
		query := "DELETE FROM events WHERE timestamp < ?"
		_, err := tx.Exec(query, cutoffTime)
		return err
	})
}

// GetEventStats returns statistics about events
func (es *EventStore) GetEventStats() (map[string]interface{}, error) {
	query := `
		SELECT
			COUNT(*) as total_events,
			COUNT(CASE WHEN processed_at IS NOT NULL THEN 1 END) as processed_events,
			COUNT(CASE WHEN severity = 'critical' THEN 1 END) as critical_events,
			COUNT(CASE WHEN severity = 'error' THEN 1 END) as error_events,
			COUNT(CASE WHEN severity = 'warning' THEN 1 END) as warning_events,
			MIN(timestamp) as oldest_event,
			MAX(timestamp) as newest_event
		FROM events
	`

	var stats struct {
		TotalEvents     int        `json:"total_events"`
		ProcessedEvents int        `json:"processed_events"`
		CriticalEvents  int        `json:"critical_events"`
		ErrorEvents     int        `json:"error_events"`
		WarningEvents   int        `json:"warning_events"`
		OldestEvent     *time.Time `json:"oldest_event"`
		NewestEvent     *time.Time `json:"newest_event"`
	}

	var oldestEvent, newestEvent sql.NullTime

	err := es.db.conn.QueryRow(query).Scan(
		&stats.TotalEvents,
		&stats.ProcessedEvents,
		&stats.CriticalEvents,
		&stats.ErrorEvents,
		&stats.WarningEvents,
		&oldestEvent,
		&newestEvent,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get event stats: %w", err)
	}

	if oldestEvent.Valid {
		stats.OldestEvent = &oldestEvent.Time
	}
	if newestEvent.Valid {
		stats.NewestEvent = &newestEvent.Time
	}

	result, _ := json.Marshal(stats)
	var resultMap map[string]interface{}
	json.Unmarshal(result, &resultMap)

	return resultMap, nil
}

// ReplayEvents replays events from a specific time for recovery
func (es *EventStore) ReplayEvents(fromTime time.Time, eventTypes []EventType, callback func(*Event) error) error {
	query := `
		SELECT id, event_type, severity, title, message, details, model_id, provider_id,
			verification_result_id, issue_id, client_id, user_id, source, timestamp, processed_at
		FROM events WHERE timestamp >= ?
	`

	args := []interface{}{fromTime}

	if len(eventTypes) > 0 {
		placeholders := ""
		for i, eventType := range eventTypes {
			if i > 0 {
				placeholders += ","
			}
			placeholders += "?"
			args = append(args, string(eventType))
		}
		query += " AND event_type IN (" + placeholders + ")"
	}

	query += " ORDER BY timestamp ASC"

	rows, err := es.db.conn.Query(query, args...)
	if err != nil {
		return fmt.Errorf("failed to query events for replay: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var event Event
		var detailsJSON string
		var processedAt sql.NullTime

		err := rows.Scan(
			&event.ID,
			(*string)(&event.Type),
			(*string)(&event.Severity),
			&event.Title,
			&event.Message,
			&detailsJSON,
			&event.ModelID,
			&event.ProviderID,
			&event.VerificationID,
			&event.IssueID,
			&event.ClientID,
			&event.UserID,
			&event.Source,
			&event.Timestamp,
			&processedAt,
		)

		if err != nil {
			return fmt.Errorf("failed to scan event during replay: %w", err)
		}

		if processedAt.Valid {
			event.ProcessedAt = &processedAt.Time
		}

		// Unmarshal details
		if detailsJSON != "" {
			err = json.Unmarshal([]byte(detailsJSON), &event.Details)
			if err != nil {
				// Log error but continue
				fmt.Printf("Warning: failed to unmarshal event details during replay: %v\n", err)
				continue
			}
		}

		// Call callback for each event
		if err := callback(&event); err != nil {
			return fmt.Errorf("callback failed during event replay: %w", err)
		}
	}

	return nil
}
