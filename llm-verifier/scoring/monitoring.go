package scoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"llm-verifier/logging"
)

// ScoringMonitor handles monitoring and alerting for the scoring system
type ScoringMonitor struct {
	scoringSystem *ScoringSystem
	logger        *logging.Logger
	alerts        AlertManager
	metrics       MetricsCollector
	config        MonitoringConfig
	mu            sync.RWMutex
}

// MonitoringConfig holds configuration for monitoring
type MonitoringConfig struct {
	Enabled                    bool
	ScoreChangeThreshold       float64
	PerformanceThreshold       float64
	APIResponseTimeThreshold   time.Duration
	DatabaseLatencyThreshold   time.Duration
	AlertCooldownPeriod        time.Duration
	MetricsRetentionPeriod     time.Duration
	EnableEmailAlerts          bool
	EnableWebhookAlerts        bool
	WebhookURL                 string
	AlertRecipients            []string
}

// DefaultMonitoringConfig returns default monitoring configuration
func DefaultMonitoringConfig() MonitoringConfig {
	return MonitoringConfig{
		Enabled:                  true,
		ScoreChangeThreshold:     0.5, // 0.5 point change triggers alert
		PerformanceThreshold:     80.0, // 80% performance threshold
		APIResponseTimeThreshold: 5 * time.Second,
		DatabaseLatencyThreshold: 1 * time.Second,
		AlertCooldownPeriod:      1 * time.Hour,
		MetricsRetentionPeriod:   7 * 24 * time.Hour, // 7 days
		EnableEmailAlerts:        false,
		EnableWebhookAlerts:      false,
		AlertRecipients:          []string{},
	}
}

// NewScoringMonitor creates a new scoring monitor
func NewScoringMonitor(scoringSystem *ScoringSystem, logger *logging.Logger, config MonitoringConfig) *ScoringMonitor {
	return &ScoringMonitor{
		scoringSystem: scoringSystem,
		logger:        logger,
		alerts:        NewAlertManager(config),
		metrics:       NewMetricsCollector(config),
		config:        config,
	}
}

// Start begins monitoring processes
func (sm *ScoringMonitor) Start(ctx context.Context) error {
	if !sm.config.Enabled {
		sm.logger.Info("Monitoring is disabled")
		return nil
	}

	sm.logger.Info("Starting scoring system monitoring")

	// Start background monitoring processes
	go sm.monitorScoreChanges(ctx)
	go sm.monitorSystemPerformance(ctx)
	go sm.monitorExternalAPIs(ctx)
	go sm.monitorDatabasePerformance(ctx)
	go sm.cleanupOldMetrics(ctx)

	return nil
}

// MonitorScoreChange monitors for significant score changes
func (sm *ScoringMonitor) MonitorScoreChange(modelID string, oldScore, newScore float64, components ScoreComponents) {
	if !sm.config.Enabled {
		return
	}

	change := newScore - oldScore
	absChange := abs(change)

	if absChange >= sm.config.ScoreChangeThreshold {
		alert := ScoreChangeAlert{
			ModelID:       modelID,
			OldScore:      oldScore,
			NewScore:      newScore,
			ScoreChange:   change,
			Components:    components,
			Timestamp:     time.Now(),
			Severity:      sm.determineAlertSeverity(absChange),
			Message:       sm.generateScoreChangeMessage(modelID, oldScore, newScore, change),
		}

		if err := sm.alerts.SendScoreChangeAlert(alert); err != nil {
			sm.logger.Error("Failed to send score change alert", "error", err, "model_id", modelID)
		}

		// Record metric
		sm.metrics.RecordScoreChange(modelID, change)
	}
}

// MonitorAPIPerformance monitors external API performance
func (sm *ScoringMonitor) MonitorAPIPerformance(apiName string, responseTime time.Duration, success bool) {
	if !sm.config.Enabled {
		return
	}

	sm.metrics.RecordAPIPerformance(apiName, responseTime, success)

	if !success || responseTime > sm.config.APIResponseTimeThreshold {
		alert := APIPerformanceAlert{
			APIName:      apiName,
			ResponseTime: responseTime,
			Success:      success,
			Timestamp:    time.Now(),
			Threshold:    sm.config.APIResponseTimeThreshold,
			Message:      sm.generateAPIPerformanceMessage(apiName, responseTime, success),
		}

		if err := sm.alerts.SendAPIPerformanceAlert(alert); err != nil {
			sm.logger.Error("Failed to send API performance alert", "error", err, "api", apiName)
		}
	}
}

// MonitorDatabasePerformance monitors database performance
func (sm *ScoringMonitor) MonitorDatabasePerformance(operation string, latency time.Duration, success bool) {
	if !sm.config.Enabled {
		return
	}

	sm.metrics.RecordDatabasePerformance(operation, latency, success)

	if !success || latency > sm.config.DatabaseLatencyThreshold {
		alert := DatabasePerformanceAlert{
			Operation:   operation,
			Latency:     latency,
			Success:     success,
			Timestamp:   time.Now(),
			Threshold:   sm.config.DatabaseLatencyThreshold,
			Message:     sm.generateDatabasePerformanceMessage(operation, latency, success),
		}

		if err := sm.alerts.SendDatabasePerformanceAlert(alert); err != nil {
			sm.logger.Error("Failed to send database performance alert", "error", err, "operation", operation)
		}
	}
}

// GetSystemHealth returns overall system health status
func (sm *ScoringMonitor) GetSystemHealth() SystemHealth {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	metrics := sm.metrics.GetCurrentMetrics()
	
	health := SystemHealth{
		OverallStatus: "healthy",
		Timestamp:     time.Now(),
		Components:    make(map[string]ComponentHealth),
	}

	// Evaluate scoring engine health
	if metrics.ScoreCalculationErrors > 10 {
		health.OverallStatus = "degraded"
		health.Components["scoring_engine"] = ComponentHealth{
			Status:  "unhealthy",
			Message: fmt.Sprintf("High error rate: %d errors", metrics.ScoreCalculationErrors),
		}
	} else {
		health.Components["scoring_engine"] = ComponentHealth{
			Status: "healthy",
			Message: "Operating normally",
		}
	}

	// Evaluate API health
	if metrics.AverageAPIResponseTime > sm.config.APIResponseTimeThreshold {
		health.OverallStatus = "degraded"
		health.Components["external_apis"] = ComponentHealth{
			Status:  "degraded",
			Message: fmt.Sprintf("API response time: %v (threshold: %v)", metrics.AverageAPIResponseTime, sm.config.APIResponseTimeThreshold),
		}
	} else {
		health.Components["external_apis"] = ComponentHealth{
			Status: "healthy",
			Message: fmt.Sprintf("API response time: %v", metrics.AverageAPIResponseTime),
		}
	}

	// Evaluate database health
	if metrics.AverageDatabaseLatency > sm.config.DatabaseLatencyThreshold {
		health.OverallStatus = "degraded"
		health.Components["database"] = ComponentHealth{
			Status:  "degraded",
			Message: fmt.Sprintf("Database latency: %v (threshold: %v)", metrics.AverageDatabaseLatency, sm.config.DatabaseLatencyThreshold),
		}
	} else {
		health.Components["database"] = ComponentHealth{
			Status: "healthy",
			Message: fmt.Sprintf("Database latency: %v", metrics.AverageDatabaseLatency),
		}
	}

	return health
}

// GetMetricsSummary returns a summary of system metrics
func (sm *ScoringMonitor) GetMetricsSummary(duration time.Duration) MetricsSummary {
	return sm.metrics.GetSummary(duration)
}

// Background monitoring processes

func (sm *ScoringMonitor) monitorScoreChanges(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.checkForScoreChanges()
		}
	}
}

func (sm *ScoringMonitor) monitorSystemPerformance(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.checkSystemPerformance()
		}
	}
}

func (sm *ScoringMonitor) monitorExternalAPIs(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.checkExternalAPIs()
		}
	}
}

func (sm *ScoringMonitor) monitorDatabasePerformance(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.checkDatabasePerformance()
		}
	}
}

func (sm *ScoringMonitor) cleanupOldMetrics(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.metrics.CleanupOldMetrics(sm.config.MetricsRetentionPeriod)
		}
	}
}

// Helper methods

func (sm *ScoringMonitor) checkForScoreChanges() {
	// Implementation would check for recent score changes in database
	sm.logger.Debug("Checking for score changes")
}

func (sm *ScoringMonitor) checkSystemPerformance() {
	// Implementation would check system resource usage
	sm.logger.Debug("Checking system performance")
}

func (sm *ScoringMonitor) checkExternalAPIs() {
	// Implementation would test external API connectivity
	sm.logger.Debug("Checking external APIs")
}

func (sm *ScoringMonitor) checkDatabasePerformance() {
	// Implementation would check database performance metrics
	sm.logger.Debug("Checking database performance")
}

func (sm *ScoringMonitor) determineAlertSeverity(change float64) string {
	absChange := abs(change)
	switch {
	case absChange >= 2.0:
		return "critical"
	case absChange >= 1.0:
		return "high"
	case absChange >= 0.5:
		return "medium"
	default:
		return "low"
	}
}

func (sm *ScoringMonitor) generateScoreChangeMessage(modelID string, oldScore, newScore, change float64) string {
	changeType := "increased"
	if change < 0 {
		changeType = "decreased"
	}

	return fmt.Sprintf("Model %s score %s from %.1f to %.1f (change: %.1f)",
		modelID, changeType, oldScore, newScore, change)
}

func (sm *ScoringMonitor) generateAPIPerformanceMessage(apiName string, responseTime time.Duration, success bool) string {
	if !success {
		return fmt.Sprintf("API %s request failed", apiName)
	}

	return fmt.Sprintf("API %s response time %v exceeds threshold %v",
		apiName, responseTime, sm.config.APIResponseTimeThreshold)
}

func (sm *ScoringMonitor) generateDatabasePerformanceMessage(operation string, latency time.Duration, success bool) string {
	if !success {
		return fmt.Sprintf("Database operation %s failed", operation)
	}

	return fmt.Sprintf("Database operation %s latency %v exceeds threshold %v",
		operation, latency, sm.config.DatabaseLatencyThreshold)
}

// Alert types

type ScoreChangeAlert struct {
	ModelID     string        `json:"model_id"`
	OldScore    float64       `json:"old_score"`
	NewScore    float64       `json:"new_score"`
	ScoreChange float64       `json:"score_change"`
	Components  ScoreComponents `json:"components"`
	Timestamp   time.Time     `json:"timestamp"`
	Severity    string        `json:"severity"`
	Message     string        `json:"message"`
}

type APIPerformanceAlert struct {
	APIName      string        `json:"api_name"`
	ResponseTime time.Duration `json:"response_time"`
	Success      bool          `json:"success"`
	Timestamp    time.Time     `json:"timestamp"`
	Threshold    time.Duration `json:"threshold"`
	Message      string        `json:"message"`
}

type DatabasePerformanceAlert struct {
	Operation string        `json:"operation"`
	Latency   time.Duration `json:"latency"`
	Success   bool          `json:"success"`
	Timestamp time.Time     `json:"timestamp"`
	Threshold time.Duration `json:"threshold"`
	Message   string        `json:"message"`
}

// System health types

type SystemHealth struct {
	OverallStatus string                      `json:"overall_status"`
	Timestamp     time.Time                   `json:"timestamp"`
	Components    map[string]ComponentHealth `json:"components"`
}

type ComponentHealth struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Metrics types

type MetricsSummary struct {
	TotalScoreCalculations   int64         `json:"total_score_calculations"`
	AverageScoreChange       float64       `json:"average_score_change"`
	APIRequestsTotal         int64         `json:"api_requests_total"`
	APIRequestsFailed        int64         `json:"api_requests_failed"`
	AverageAPIResponseTime   time.Duration `json:"average_api_response_time"`
	DatabaseOperationsTotal  int64         `json:"database_operations_total"`
	DatabaseOperationsFailed int64         `json:"database_operations_failed"`
	AverageDatabaseLatency   time.Duration `json:"average_database_latency"`
	LastUpdated              time.Time     `json:"last_updated"`
}

// Helper functions

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}