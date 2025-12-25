package monitoring

import (
	"fmt"
	"log"
	"time"

	"llm-verifier/database"
	"llm-verifier/events"
)

// LimitsMonitor monitors usage against configured limits and triggers alerts
type LimitsMonitor struct {
	db            *database.Database
	eventManager  *events.EventManager
	alertManager  *AlertManager
	checkInterval time.Duration
	stopCh        chan struct{}
}

// NewLimitsMonitor creates a new limits monitor
func NewLimitsMonitor(db *database.Database, eventManager *events.EventManager, alertManager *AlertManager) *LimitsMonitor {
	return &LimitsMonitor{
		db:            db,
		eventManager:  eventManager,
		alertManager:  alertManager,
		checkInterval: 5 * time.Minute, // Check every 5 minutes
		stopCh:        make(chan struct{}),
	}
}

// Start begins the limits monitoring process
func (lm *LimitsMonitor) Start() {
	log.Println("Starting limits monitor...")
	go lm.monitorLoop()
}

// Stop stops the limits monitoring process
func (lm *LimitsMonitor) Stop() {
	close(lm.stopCh)
	log.Println("Limits monitor stopped")
}

func (lm *LimitsMonitor) monitorLoop() {
	ticker := time.NewTicker(lm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-lm.stopCh:
			return
		case <-ticker.C:
			lm.checkLimits()
		}
	}
}

func (lm *LimitsMonitor) checkLimits() {
	// Get all active limits from database
	limits, err := lm.db.ListLimits(map[string]interface{}{})
	if err != nil {
		log.Printf("Failed to fetch limits: %v", err)
		return
	}

	for _, limit := range limits {
		lm.checkLimit(limit)
	}
}

func (lm *LimitsMonitor) checkLimit(limit *database.Limit) {
	// Calculate current usage percentage
	usagePercent := float64(limit.CurrentUsage) / float64(limit.LimitValue) * 100

	// Check different threshold levels
	switch {
	case usagePercent >= 100 && limit.IsHardLimit:
		// Hard limit exceeded - critical alert
		lm.triggerLimitAlert(limit, usagePercent, events.SeverityCritical, "Hard limit exceeded")

	case usagePercent >= 95:
		// Approaching hard limit - warning
		lm.triggerLimitAlert(limit, usagePercent, events.SeverityWarning, "Approaching hard limit")

	case usagePercent >= 80:
		// High usage - info alert
		lm.triggerLimitAlert(limit, usagePercent, events.SeverityInfo, "High usage detected")

	case limit.ResetTime != nil && time.Now().After(*limit.ResetTime):
		// Reset time has passed, reset usage
		lm.resetLimit(limit)
	}
}

func (lm *LimitsMonitor) triggerLimitAlert(limit *database.Limit, usagePercent float64, severity events.Severity, message string) {
	// Get model information for context
	model, err := lm.db.GetModel(limit.ModelID)
	modelName := "Unknown"
	if err == nil {
		modelName = model.ModelID
	}

	alertMessage := fmt.Sprintf("%s for model %s (%s): %.1f%% usage (%d/%d)",
		message, modelName, limit.LimitType, usagePercent, limit.CurrentUsage, limit.LimitValue)

	// Create event
	event := events.CreateEventWithDetails(
		events.EventIssueDetected,
		severity,
		fmt.Sprintf("Limit Alert: %s", limit.LimitType),
		alertMessage,
		map[string]interface{}{
			"model_id":      limit.ModelID,
			"model_name":    modelName,
			"limit_type":    limit.LimitType,
			"current_usage": limit.CurrentUsage,
			"limit_value":   limit.LimitValue,
			"usage_percent": usagePercent,
			"is_hard_limit": limit.IsHardLimit,
		},
	)

	// Publish event
	if err := lm.eventManager.PublishEvent(event); err != nil {
		log.Printf("Failed to publish limit alert event: %v", err)
	}
}

func (lm *LimitsMonitor) resetLimit(limit *database.Limit) {
	// Reset usage to 0 and update reset time
	limit.CurrentUsage = 0

	// Calculate next reset time based on reset period
	if limit.ResetPeriod != "" {
		resetDuration, err := time.ParseDuration(limit.ResetPeriod)
		if err == nil {
			nextReset := time.Now().Add(resetDuration)
			limit.ResetTime = &nextReset
		}
	}

	// Update in database
	if err := lm.db.UpdateLimit(limit); err != nil {
		log.Printf("Failed to reset limit %d: %v", limit.ID, err)
	}
}

// UpdateUsage updates the usage for a specific limit
func (lm *LimitsMonitor) UpdateUsage(modelID int64, limitType string, usageIncrement int) error {
	// Find the relevant limit
	limits, err := lm.db.ListLimits(map[string]interface{}{
		"model_id":   modelID,
		"limit_type": limitType,
	})
	if err != nil {
		return fmt.Errorf("failed to find limit: %w", err)
	}

	if len(limits) == 0 {
		return fmt.Errorf("no limit found for model %d, type %s", modelID, limitType)
	}

	limit := limits[0]

	// Check if we need to reset based on time
	if limit.ResetTime != nil && time.Now().After(*limit.ResetTime) {
		lm.resetLimit(limit)
	}

	// Update usage
	limit.CurrentUsage += usageIncrement

	// Save to database
	if err := lm.db.UpdateLimit(limit); err != nil {
		return fmt.Errorf("failed to update limit usage: %w", err)
	}

	// Check if this update triggers any alerts
	lm.checkLimit(limit)

	return nil
}
