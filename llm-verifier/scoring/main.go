package scoring

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"llm-verifier/database"
	"llm-verifier/logging"
)

// ScoringService provides the main interface for the scoring system
type ScoringService struct {
	system  *ScoringSystem
	monitor *ScoringMonitor
	logger  *logging.Logger
	config  ServiceConfig
}

// ServiceConfig holds configuration for the scoring service
type ServiceConfig struct {
	SystemConfig   ScoringSystemConfig
	MonitorConfig  MonitoringConfig
	AutoStart      bool
	EnableMetrics  bool
	MetricsPort    int
	EnableTracing  bool
}

// DefaultServiceConfig returns default service configuration
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		SystemConfig:  DefaultScoringSystemConfig(),
		MonitorConfig: DefaultMonitoringConfig(),
		AutoStart:     true,
		EnableMetrics: true,
		MetricsPort:   9090,
		EnableTracing: false,
	}
}

// NewScoringService creates a new scoring service
func NewScoringService(db *database.Database, logger *logging.Logger, config ServiceConfig) (*ScoringService, error) {
	if logger == nil {
		logger = &logging.Logger{}
	}

	// Initialize scoring system
	system, err := NewScoringSystem(db, logger, config.SystemConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create scoring system: %w", err)
	}

	// Initialize monitoring
	monitor := NewScoringMonitor(system, logger, config.MonitorConfig)

	service := &ScoringService{
		system:  system,
		monitor: monitor,
		logger:  logger,
		config:  config,
	}

	logger.Info("Scoring service initialized successfully", map[string]any{})
	return service, nil
}

// Start starts the scoring service
func (ss *ScoringService) Start(ctx context.Context) error {
	ss.logger.Info("Starting scoring service", map[string]any{})

	// Start scoring system
	if err := ss.system.Start(ctx); err != nil {
		return fmt.Errorf("failed to start scoring system: %w", err)
	}

	// Start monitoring
	if err := ss.monitor.Start(ctx); err != nil {
		return fmt.Errorf("failed to start monitoring: %w", err)
	}

	// Start metrics server if enabled
	if ss.config.EnableMetrics {
		go ss.startMetricsServer()
	}

	ss.logger.Info("Scoring service started successfully", map[string]any{})
	return nil
}

// Stop gracefully stops the scoring service
func (ss *ScoringService) Stop() error {
	ss.logger.Info("Stopping scoring service", map[string]any{})

	// Stop monitoring first
	if err := ss.system.Stop(); err != nil {
		ss.logger.Info("Failed to stop scoring system", map[string]any{"error": err})
	}

	ss.logger.Info("Scoring service stopped", map[string]any{})
	return nil
}

// CalculateModelScore calculates a score for a single model
func (ss *ScoringService) CalculateModelScore(ctx context.Context, modelID string, config *ScoringConfig) (*ComprehensiveScore, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		ss.monitor.metrics.RecordScoreCalculation(true)
		ss.logger.Info("Model score calculated", map[string]any{"model_id": modelID, "duration": duration})
	}()

	score, err := ss.system.CalculateModelScore(ctx, modelID, config)
	if err != nil {
		ss.monitor.metrics.RecordScoreCalculation(false)
		return nil, fmt.Errorf("failed to calculate model score: %w", err)
	}

	return score, nil
}

// BatchCalculateScores calculates scores for multiple models
func (ss *ScoringService) BatchCalculateScores(ctx context.Context, modelIDs []string, config *ScoringConfig) ([]*ComprehensiveScore, error) {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		ss.logger.Info("Batch scores calculated", map[string]any{"model_count": len(modelIDs), "duration": duration})
	}()

	scores, err := ss.system.BatchCalculateScores(ctx, modelIDs, config)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate batch scores: %w", err)
	}

	return scores, nil
}

// GetModelScore retrieves the current score for a model
func (ss *ScoringService) GetModelScore(modelID string) (*ComprehensiveScore, error) {
	score, err := ss.system.GetModelScore(modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model score: %w", err)
	}

	return score, nil
}

// GetModelRankings retrieves model rankings
func (ss *ScoringService) GetModelRankings(category string, limit int) ([]ModelRanking, error) {
	rankings, err := ss.system.GetModelRankings(category, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get model rankings: %w", err)
	}

	return rankings, nil
}

// GetScoreDistribution retrieves the distribution of scores
func (ss *ScoringService) GetScoreDistribution() (ScoreDistribution, error) {
	distribution, err := ss.system.GetScoreDistribution()
	if err != nil {
		return ScoreDistribution{}, fmt.Errorf("failed to get score distribution: %w", err)
	}

	return distribution, nil
}

// SyncWithModelsDev synchronizes data with models.dev
func (ss *ScoringService) SyncWithModelsDev(ctx context.Context, providerID, modelID string, force bool) error {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		ss.logger.Info("Models.dev sync completed", map[string]any{"provider": providerID, "model": modelID, "duration": duration})
	}()

	if err := ss.system.SyncWithModelsDev(ctx, providerID, modelID, force); err != nil {
		return fmt.Errorf("failed to sync with models.dev: %w", err)
	}

	return nil
}

// UpdateModelNameWithScore updates a model name with score suffix
func (ss *ScoringService) UpdateModelNameWithScore(modelID string, score float64) error {
	if err := ss.system.UpdateModelNameWithScore(modelID, score); err != nil {
		return fmt.Errorf("failed to update model name: %w", err)
	}

	return nil
}

// GetSystemHealth returns the current system health status
func (ss *ScoringService) GetSystemHealth() SystemHealth {
	return ss.monitor.GetSystemHealth()
}

// GetMetricsSummary returns a summary of system metrics
func (ss *ScoringService) GetMetricsSummary(duration time.Duration) MetricsSummary {
	return ss.monitor.GetMetricsSummary(duration)
}

// GetServiceStatus returns the overall service status
func (ss *ScoringService) GetServiceStatus() ServiceStatus {
	health := ss.GetSystemHealth()
	metrics := ss.GetMetricsSummary(5 * time.Minute)

	rankings, err := ss.GetModelRankings("overall", 10000)
	if err != nil {
		rankings = []ModelRanking{}
	}
	
	status := ServiceStatus{
		Status:           health.OverallStatus,
		Uptime:           time.Since(time.Now().Add(-24 * time.Hour)), // Placeholder
		TotalModels:      int64(len(rankings)),
		LastScoreCalc:    metrics.LastUpdated,
		ScoreCalcRate:    ss.monitor.metrics.GetScoreCalculationRate(),
		APIErrorRate:     ss.monitor.metrics.GetAPIErrorRate(),
		DatabaseErrorRate: ss.monitor.metrics.GetDatabaseErrorRate(),
		LastUpdated:      time.Now(),
	}

	return status
}

// ServiceStatus represents the overall service status
type ServiceStatus struct {
	Status            string        `json:"status"`
	Uptime            time.Duration `json:"uptime"`
	TotalModels       int64         `json:"total_models"`
	LastScoreCalc     time.Time     `json:"last_score_calc"`
	ScoreCalcRate     float64       `json:"score_calc_rate"`
	APIErrorRate      float64       `json:"api_error_rate"`
	DatabaseErrorRate float64       `json:"database_error_rate"`
	LastUpdated       time.Time     `json:"last_updated"`
}

// Helper methods

func (ss *ScoringService) startMetricsServer() {
	// This would start a metrics server (e.g., Prometheus metrics endpoint)
	ss.logger.Info("Starting metrics server", map[string]any{"port": ss.config.MetricsPort})
	// Implementation would depend on the metrics library being used
}

// Convenience methods for common operations

// QuickScore calculates a score with default configuration
func (ss *ScoringService) QuickScore(ctx context.Context, modelID string) (*ComprehensiveScore, error) {
	return ss.CalculateModelScore(ctx, modelID, nil)
}

// QuickBatch calculates scores for multiple models with default configuration
func (ss *ScoringService) QuickBatch(ctx context.Context, modelIDs []string) ([]*ComprehensiveScore, error) {
	return ss.BatchCalculateScores(ctx, modelIDs, nil)
}

// GetTopModels returns the top N models by overall score
func (ss *ScoringService) GetTopModels(n int) ([]ModelRanking, error) {
	return ss.GetModelRankings("overall", n)
}

// GetModelsByScoreRange returns models within a score range
func (ss *ScoringService) GetModelsByScoreRange(minScore, maxScore float64, limit int) ([]ModelRanking, error) {
	allRankings, err := ss.GetModelRankings("overall", limit*2) // Get more to filter
	if err != nil {
		return nil, err
	}

	var filtered []ModelRanking
	for _, ranking := range allRankings {
		if ranking.OverallScore >= minScore && ranking.OverallScore <= maxScore {
			filtered = append(filtered, ranking)
			if len(filtered) >= limit {
				break
			}
		}
	}

	return filtered, nil
}

// RefreshAllScores recalculates all model scores
func (ss *ScoringService) RefreshAllScores(ctx context.Context) error {
	ss.logger.Info("Refreshing all model scores", map[string]any{})

	// Get all models
	rankings, err := ss.GetModelRankings("overall", 10000)
	if err != nil {
		return fmt.Errorf("failed to get all models: %w", err)
	}

	modelIDs := make([]string, len(rankings))
	for i, ranking := range rankings {
		// Convert ranking model ID back to string model ID
		// This is a simplified implementation
		modelIDs[i] = fmt.Sprintf("model_%s", ranking.ModelID)
	}

	_, err = ss.BatchCalculateScores(ctx, modelIDs, nil)
	if err != nil {
		return fmt.Errorf("failed to refresh scores: %w", err)
	}

	ss.logger.Info("All model scores refreshed successfully", map[string]any{"count": len(modelIDs)})
	return nil
}

// ValidateConfiguration validates the service configuration
func (ss *ScoringService) ValidateConfiguration() error {
	// Validate system config
	if err := ss.validateSystemConfig(ss.config.SystemConfig); err != nil {
		return fmt.Errorf("invalid system configuration: %w", err)
	}

	// Validate monitor config
	if err := ss.validateMonitorConfig(ss.config.MonitorConfig); err != nil {
		return fmt.Errorf("invalid monitor configuration: %w", err)
	}

	return nil
}

func (ss *ScoringService) validateSystemConfig(config ScoringSystemConfig) error {
	if config.MaxConcurrentCalcs <= 0 {
		return fmt.Errorf("max_concurrent_calcs must be positive")
	}

	if config.AutoSyncInterval <= 0 {
		return fmt.Errorf("auto_sync_interval must be positive")
	}

	if config.ScoreRecalcInterval <= 0 {
		return fmt.Errorf("score_recalc_interval must be positive")
	}

	if config.PerformanceWindow <= 0 {
		return fmt.Errorf("performance_window must be positive")
	}

	return nil
}

func (ss *ScoringService) validateMonitorConfig(config MonitoringConfig) error {
	if config.ScoreChangeThreshold < 0 {
		return fmt.Errorf("score_change_threshold must be non-negative")
	}

	if config.PerformanceThreshold < 0 || config.PerformanceThreshold > 100 {
		return fmt.Errorf("performance_threshold must be between 0 and 100")
	}

	if config.APIResponseTimeThreshold <= 0 {
		return fmt.Errorf("api_response_time_threshold must be positive")
	}

	if config.DatabaseLatencyThreshold <= 0 {
		return fmt.Errorf("database_latency_threshold must be positive")
	}

	if config.AlertCooldownPeriod <= 0 {
		return fmt.Errorf("alert_cooldown_period must be positive")
	}

	if config.MetricsRetentionPeriod <= 0 {
		return fmt.Errorf("metrics_retention_period must be positive")
	}

	return nil
}

// ExportScores exports scores to various formats
func (ss *ScoringService) ExportScores(format string, filters map[string]interface{}) ([]byte, error) {
	switch format {
	case "json":
		return ss.exportScoresJSON(filters)
	case "csv":
		return ss.exportScoresCSV(filters)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

func (ss *ScoringService) exportScoresJSON(filters map[string]interface{}) ([]byte, error) {
	rankings, err := ss.GetModelRankings("overall", 10000)
	if err != nil {
		return nil, fmt.Errorf("failed to get rankings: %w", err)
	}

	// Apply filters if provided
	filteredRankings := rankings
	if len(filters) > 0 {
		filteredRankings = ss.applyFilters(rankings, filters)
	}

	// Convert to JSON
	return json.Marshal(filteredRankings)
}

func (ss *ScoringService) exportScoresCSV(filters map[string]interface{}) ([]byte, error) {
	rankings, err := ss.GetModelRankings("overall", 10000)
	if err != nil {
		return nil, fmt.Errorf("failed to get rankings: %w", err)
	}

	// Apply filters if provided
	filteredRankings := rankings
	if len(filters) > 0 {
		filteredRankings = ss.applyFilters(rankings, filters)
	}

	// Convert to CSV
	var csvData []byte
	csvData = append(csvData, []byte("Rank,Model ID,Model Name,Overall Score,Score Suffix,Category Score,Last Updated\n")...)
	
	for _, ranking := range filteredRankings {
		line := fmt.Sprintf("%d,%s,%s,%.1f,%s,%.1f,%s\n",
			ranking.Rank,
			ranking.ModelID,
			ranking.ModelName,
			ranking.OverallScore,
			ranking.ScoreSuffix,
			ranking.CategoryScore,
			ranking.LastUpdated.Format(time.RFC3339),
		)
		csvData = append(csvData, []byte(line)...)
	}

	return csvData, nil
}

func (ss *ScoringService) applyFilters(rankings []ModelRanking, filters map[string]interface{}) []ModelRanking {
	// This is a simplified filter implementation
	// In a real implementation, you would handle various filter types
	filtered := make([]ModelRanking, 0)
	
	for _, ranking := range rankings {
		// Apply filters based on filter map
		// For now, just return all rankings
		filtered = append(filtered, ranking)
	}
	
	return filtered
}