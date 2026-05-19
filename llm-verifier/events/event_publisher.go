package events

import (
	"encoding/json"
	"log"
	"time"

	"digital.vasic.llmsverifier/database"
)

// EventPublisher provides high-level event publishing functions
type EventPublisher struct {
	eventManager *EventManager
	db           *database.Database
}

// NewEventPublisher creates a new event publisher
func NewEventPublisher(eventManager *EventManager, db *database.Database) *EventPublisher {
	return &EventPublisher{
		eventManager: eventManager,
		db:           db,
	}
}

// PublishVerificationStarted publishes a verification started event
func (ep *EventPublisher) PublishVerificationStarted(modelCount int, providerCount int) error {
	event := CreateEvent(
		EventVerificationStarted,
		SeverityInfo,
		tr("events.verification.started.title"),
		trData("events.verification.started.message", map[string]any{
			"model_count":    modelCount,
			"provider_count": providerCount,
		}),
	)

	event.Details = map[string]interface{}{
		"model_count":    modelCount,
		"provider_count": providerCount,
		"start_time":     time.Now(),
	}

	return ep.publishAndStoreEvent(event)
}

// PublishVerificationCompleted publishes a verification completed event
func (ep *EventPublisher) PublishVerificationCompleted(duration time.Duration, successCount, failureCount int) error {
	event := CreateEvent(
		EventVerificationCompleted,
		SeverityInfo,
		tr("events.verification.completed.title"),
		trData("events.verification.completed.message", map[string]any{
			"duration":      duration.String(),
			"success_count": successCount,
			"failure_count": failureCount,
		}),
	)

	event.Details = map[string]interface{}{
		"duration_seconds": duration.Seconds(),
		"success_count":    successCount,
		"failure_count":    failureCount,
		"completion_time":  time.Now(),
	}

	return ep.publishAndStoreEvent(event)
}

// PublishVerificationFailed publishes a verification failed event
func (ep *EventPublisher) PublishVerificationFailed(errorMsg string) error {
	event := CreateEvent(
		EventVerificationFailed,
		SeverityError,
		tr("events.verification.failed.title"),
		trData("events.verification.failed.message", map[string]any{"error": errorMsg}),
	)

	event.Details = map[string]interface{}{
		"error_message": errorMsg,
		"failure_time":  time.Now(),
	}

	return ep.publishAndStoreEvent(event)
}

// PublishScoreChanged publishes a score change event
func (ep *EventPublisher) PublishScoreChanged(modelID int64, oldScore, newScore int, scoreType string) error {
	var severity Severity
	var title, message string

	if newScore > oldScore {
		severity = SeverityInfo
		title = tr("events.score.improved.title")
		message = trData("events.score.improved.message", map[string]any{
			"old_score":  oldScore,
			"new_score":  newScore,
			"score_type": scoreType,
		})
	} else if newScore < oldScore {
		severity = SeverityWarning
		title = tr("events.score.decreased.title")
		message = trData("events.score.decreased.message", map[string]any{
			"old_score":  oldScore,
			"new_score":  newScore,
			"score_type": scoreType,
		})
	} else {
		// No change, don't publish
		return nil
	}

	event := CreateModelEvent(EventScoreChanged, severity, title, message, modelID)

	event.Details = map[string]interface{}{
		"old_score":  oldScore,
		"new_score":  newScore,
		"score_type": scoreType,
		"change":     newScore - oldScore,
	}

	return ep.publishAndStoreEvent(event)
}

// PublishIssueDetected publishes an issue detection event
func (ep *EventPublisher) PublishIssueDetected(modelID int64, issueType, severity, title, description string) error {
	event := CreateModelEvent(
		EventIssueDetected,
		Severity(severity),
		tr("events.issue.detected.title"),
		trData("events.issue.detected.message", map[string]any{
			"title":       title,
			"description": description,
		}),
		modelID,
	)

	event.Details = map[string]interface{}{
		"issue_type":  issueType,
		"severity":    severity,
		"title":       title,
		"description": description,
		"detected_at": time.Now(),
	}

	return ep.publishAndStoreEvent(event)
}

// PublishIssueResolved publishes an issue resolution event
func (ep *EventPublisher) PublishIssueResolved(modelID int64, issueID int64, resolution string) error {
	event := CreateModelEvent(
		EventIssueResolved,
		SeverityInfo,
		tr("events.issue.resolved.title"),
		trData("events.issue.resolved.message", map[string]any{
			"issue_id":   issueID,
			"resolution": resolution,
		}),
		modelID,
	)

	event.Details = map[string]interface{}{
		"issue_id":    issueID,
		"resolution":  resolution,
		"resolved_at": time.Now(),
	}

	return ep.publishAndStoreEvent(event)
}

// PublishClientConnected publishes a client connection event
func (ep *EventPublisher) PublishClientConnected(clientID, clientType string) error {
	event := CreateClientEvent(
		EventClientConnected,
		SeverityInfo,
		tr("events.client.connected.title"),
		trData("events.client.connected.message", map[string]any{
			"client_type": clientType,
			"client_id":   clientID,
		}),
		clientID,
	)

	event.Details = map[string]interface{}{
		"client_type":  clientType,
		"client_id":    clientID,
		"connected_at": time.Now(),
	}

	return ep.publishAndStoreEvent(event)
}

// PublishClientDisconnected publishes a client disconnection event
func (ep *EventPublisher) PublishClientDisconnected(clientID, clientType string) error {
	event := CreateClientEvent(
		EventClientDisconnected,
		SeverityInfo,
		tr("events.client.disconnected.title"),
		trData("events.client.disconnected.message", map[string]any{
			"client_type": clientType,
			"client_id":   clientID,
		}),
		clientID,
	)

	event.Details = map[string]interface{}{
		"client_type":     clientType,
		"client_id":       clientID,
		"disconnected_at": time.Now(),
	}

	return ep.publishAndStoreEvent(event)
}

// PublishSystemHealthChanged publishes a system health change event
func (ep *EventPublisher) PublishSystemHealthChanged(healthStatus string, details map[string]interface{}) error {
	var severity Severity
	var title string

	switch healthStatus {
	case "healthy":
		severity = SeverityInfo
		title = tr("events.health.healthy.title")
	case "degraded":
		severity = SeverityWarning
		title = tr("events.health.degraded.title")
	case "unhealthy":
		severity = SeverityError
		title = tr("events.health.unhealthy.title")
	case "critical":
		severity = SeverityCritical
		title = tr("events.health.critical.title")
	default:
		severity = SeverityInfo
		title = tr("events.health.changed.title")
	}

	message := trData("events.health.changed.message", map[string]any{"health_status": healthStatus})
	event := CreateEvent(EventSystemHealthChanged, severity, title, message)

	event.Details = details
	event.Details["health_status"] = healthStatus
	event.Details["changed_at"] = time.Now()

	return ep.publishAndStoreEvent(event)
}

// PublishConfigExported publishes a configuration export event
func (ep *EventPublisher) PublishConfigExported(configType string, targetCount int) error {
	event := CreateEvent(
		EventConfigExported,
		SeverityInfo,
		tr("events.config.exported.title"),
		trData("events.config.exported.message", map[string]any{
			"config_type":  configType,
			"target_count": targetCount,
		}),
	)

	event.Details = map[string]interface{}{
		"config_type":  configType,
		"target_count": targetCount,
		"exported_at":  time.Now(),
	}

	return ep.publishAndStoreEvent(event)
}

// PublishSecurityAlert publishes a security alert event
func (ep *EventPublisher) PublishSecurityAlert(alertType, message string, details map[string]interface{}) error {
	event := CreateEvent(
		EventSecurityAlert,
		SeverityCritical,
		tr("events.security.alert.title"),
		trData("events.security.alert.message", map[string]any{
			"alert_type": alertType,
			"message":    message,
		}),
	)

	event.Details = details
	event.Details["alert_type"] = alertType
	event.Details["alert_time"] = time.Now()

	return ep.publishAndStoreEvent(event)
}

// PublishDatabaseMigration publishes a database migration event
func (ep *EventPublisher) PublishDatabaseMigration(migrationVersion int, description string, success bool) error {
	var severity Severity
	var title, message string

	if success {
		severity = SeverityInfo
		title = tr("events.migration.completed.title")
		message = trData("events.migration.completed.message", map[string]any{
			"migration_version": migrationVersion,
			"description":       description,
		})
	} else {
		severity = SeverityError
		title = tr("events.migration.failed.title")
		message = trData("events.migration.failed.message", map[string]any{
			"migration_version": migrationVersion,
			"description":       description,
		})
	}

	event := CreateEvent(EventDatabaseMigration, severity, title, message)

	event.Details = map[string]interface{}{
		"migration_version": migrationVersion,
		"description":       description,
		"success":           success,
		"migration_time":    time.Now(),
	}

	return ep.publishAndStoreEvent(event)
}

// publishAndStoreEvent publishes an event and stores it in the database
func (ep *EventPublisher) publishAndStoreEvent(event *Event) error {
	// Store in database first
	if ep.db != nil {
		dbEvent := convertToDatabaseEvent(event)
		if err := ep.db.CreateEvent(dbEvent); err != nil {
			// Log error but don't fail the publish
			log.Printf("Failed to store event in database: %v", err)
		}
	}

	// Publish to subscribers
	return ep.eventManager.PublishEvent(event)
}

// convertToDatabaseEvent converts an events.Event to database.Event
func convertToDatabaseEvent(event *Event) *database.Event {
	var details *string
	if event.Details != nil {
		// Convert map to JSON string
		if detailsBytes, err := json.Marshal(event.Details); err == nil {
			detailsStr := string(detailsBytes)
			details = &detailsStr
		}
	}

	return &database.Event{
		EventType:            string(event.Type),
		Severity:             string(event.Severity),
		Title:                event.Title,
		Message:              event.Message,
		Details:              details,
		ModelID:              event.ModelID,
		ProviderID:           event.ProviderID,
		VerificationResultID: event.VerificationID,
		IssueID:              event.IssueID,
	}
}
