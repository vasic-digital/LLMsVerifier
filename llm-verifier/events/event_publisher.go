package events

import (
	"fmt"
	"log"
	"time"

	"llm-verifier/database"
)

// EventPublisher provides high-level event publishing functions
type EventPublisher struct {
	eventManager *EventManager
	eventStore   *EventStore
}

// NewEventPublisher creates a new event publisher
func NewEventPublisher(eventManager *EventManager, db *database.Database) *EventPublisher {
	return &EventPublisher{
		eventManager: eventManager,
		eventStore:   NewEventStore(db),
	}
}

// PublishVerificationStarted publishes a verification started event
func (ep *EventPublisher) PublishVerificationStarted(modelCount int, providerCount int) error {
	event := CreateEvent(
		EventVerificationStarted,
		SeverityInfo,
		"Model Verification Started",
		fmt.Sprintf("Starting verification of %d models across %d providers", modelCount, providerCount),
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
		"Model Verification Completed",
		fmt.Sprintf("Verification completed in %s. %d models verified successfully, %d failed",
			duration, successCount, failureCount),
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
		"Model Verification Failed",
		fmt.Sprintf("Verification failed with error: %s", errorMsg),
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
		title = "Model Score Improved"
		message = fmt.Sprintf("Model score increased from %d to %d (%s)", oldScore, newScore, scoreType)
	} else if newScore < oldScore {
		severity = SeverityWarning
		title = "Model Score Decreased"
		message = fmt.Sprintf("Model score decreased from %d to %d (%s)", oldScore, newScore, scoreType)
	} else {
		// No change, don't publish
		return nil
	}

	event := CreateModelEvent(severity, title, message, modelID)

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
		Severity(severity),
		"Issue Detected",
		fmt.Sprintf("%s: %s", title, description),
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
		SeverityInfo,
		"Issue Resolved",
		fmt.Sprintf("Issue %d resolved: %s", issueID, resolution),
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
		"Client Connected",
		fmt.Sprintf("%s client connected: %s", clientType, clientID),
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
		"Client Disconnected",
		fmt.Sprintf("%s client disconnected: %s", clientType, clientID),
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
		title = "System Health: Healthy"
	case "degraded":
		severity = SeverityWarning
		title = "System Health: Degraded"
	case "unhealthy":
		severity = SeverityError
		title = "System Health: Unhealthy"
	case "critical":
		severity = SeverityCritical
		title = "System Health: Critical"
	default:
		severity = SeverityInfo
		title = "System Health Changed"
	}

	message := fmt.Sprintf("System health status changed to: %s", healthStatus)
	event := CreateEvent(severity, title, message)

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
		"Configuration Exported",
		fmt.Sprintf("Exported %s configuration for %d targets", configType, targetCount),
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
		"Security Alert",
		fmt.Sprintf("[%s] %s", alertType, message),
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
		title = "Database Migration Completed"
		message = fmt.Sprintf("Successfully applied migration %d: %s", migrationVersion, description)
	} else {
		severity = SeverityError
		title = "Database Migration Failed"
		message = fmt.Sprintf("Failed to apply migration %d: %s", migrationVersion, description)
	}

	event := CreateEvent(severity, title, message)

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
	if err := ep.eventStore.StoreEvent(event); err != nil {
		log.Printf("Failed to store event in database: %v", err)
		// Continue publishing even if storage fails
	}

	// Publish to subscribers
	return ep.eventManager.PublishEvent(event)
}
