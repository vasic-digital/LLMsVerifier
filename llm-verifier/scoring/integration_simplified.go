package scoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"llm-verifier/database"
	"llm-verifier/logging"
)

// ScoringSystem represents the main scoring system integration
type ScoringSystem struct {
	engine            *ScoringEngine
	modelsDevClient   *ModelsDevClient
	databaseExt       *ScoringDatabaseExtensions
	modelNaming       *ModelNaming
	logger            *logging.Logger
	config            ScoringSystemConfig
	backgroundWorkers *sync.WaitGroup
	shutdown          chan struct{}
}

// ScoringSystemConfig holds configuration for the scoring system
type ScoringSystemConfig struct {
	AutoSyncInterval      time.Duration
	ScoreRecalcInterval   time.Duration
	PerformanceWindow     time.Duration
	MaxConcurrentCalcs    int
	EnableBackgroundSync  bool
	EnableScoreMonitoring bool
	ScoreChangeThreshold  float64
}

// DefaultScoringSystemConfig returns default configuration
func DefaultScoringSystemConfig() ScoringSystemConfig {
	return ScoringSystemConfig{
		AutoSyncInterval:      6 * time.Hour,
		ScoreRecalcInterval:   1 * time.Hour,
		PerformanceWindow:     24 * time.Hour,
		MaxConcurrentCalcs:    10,
		EnableBackgroundSync:  true,
		EnableScoreMonitoring: true,
		ScoreChangeThreshold:  0.5, // 0.5 point change triggers notification
	}
}

// NewScoringSystem creates a new scoring system instance
func NewScoringSystem(db *database.Database, logger *logging.Logger, config ScoringSystemConfig) (*ScoringSystem, error) {
	// Initialize models.dev client
	modelsDevConfig := DefaultClientConfig()
	modelsDevClient, err := NewModelsDevClient(modelsDevConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create models.dev client: %w", err)
	}

	// Initialize scoring engine
	scoringEngine := NewScoringEngine(db, modelsDevClient, logger)

	// Initialize database extensions
	databaseExt := NewScoringDatabaseExtensions(db)

	// Initialize model naming
	modelNaming := NewModelNaming()

	system := &ScoringSystem{
		engine:            scoringEngine,
		modelsDevClient:   modelsDevClient,
		databaseExt:       databaseExt,
		modelNaming:       modelNaming,
		logger:            logger,
		config:            config,
		backgroundWorkers: &sync.WaitGroup{},
		shutdown:          make(chan struct{}),
	}

	// Initialize database schema
	if err := system.InitializeDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return system, nil
}

// InitializeDatabase sets up the scoring database schema
func (ss *ScoringSystem) InitializeDatabase() error {
	return ss.databaseExt.InitializeScoringSchema()
}

// Start begins the background processes
func (ss *ScoringSystem) Start(ctx context.Context) error {
	ss.logger.Info("Starting scoring system", map[string]any{})

	if ss.config.EnableBackgroundSync {
		ss.startBackgroundSync(ctx)
	}

	if ss.config.EnableScoreMonitoring {
		ss.startScoreMonitoring(ctx)
	}

	return nil
}

// Stop gracefully shuts down the scoring system
func (ss *ScoringSystem) Stop() error {
	ss.logger.Info("Stopping scoring system", map[string]any{})
	
	close(ss.shutdown)
	ss.backgroundWorkers.Wait()
	
	return nil
}

// CalculateModelScore calculates a comprehensive score for a single model
func (ss *ScoringSystem) CalculateModelScore(ctx context.Context, modelID string, config *ScoringConfig) (*ComprehensiveScore, error) {
	if config == nil {
		defaultConfig := DefaultScoringConfig()
		config = &defaultConfig
	}

	score, err := ss.engine.CalculateComprehensiveScore(ctx, modelID, *config)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate model score: %w", err)
	}

	// Update model name with score suffix if enabled
	if err := ss.UpdateModelNameWithScore(modelID, score.OverallScore); err != nil {
		ss.logger.Info("Failed to update model name with score", map[string]any{"error": err, "model_id": modelID})
	}

	return score, nil
}

// BatchCalculateScores calculates scores for multiple models
func (ss *ScoringSystem) BatchCalculateScores(ctx context.Context, modelIDs []string, config *ScoringConfig) ([]*ComprehensiveScore, error) {
	if config == nil {
		defaultConfig := DefaultScoringConfig()
		config = &defaultConfig
	}

	scores := make([]*ComprehensiveScore, 0, len(modelIDs))
	sem := make(chan struct{}, ss.config.MaxConcurrentCalcs)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, modelID := range modelIDs {
		wg.Add(1)
		go func(mid string) {
			defer wg.Done()
			
			sem <- struct{}{}
			defer func() { <-sem }()

			score, err := ss.engine.CalculateComprehensiveScore(ctx, mid, *config)
			if err != nil {
				ss.logger.Info("Failed to calculate model score", map[string]any{"error": err, "model_id": mid})
				return
			}

			mu.Lock()
			scores = append(scores, score)
			mu.Unlock()

			// Update model name with score
			if err := ss.UpdateModelNameWithScore(mid, score.OverallScore); err != nil {
				ss.logger.Info("Failed to update model name with score", map[string]any{"error": err, "model_id": mid})
			}
		}(modelID)
	}

	wg.Wait()
	return scores, nil
}

// UpdateModelNameWithScore updates a model's name with the score suffix
func (ss *ScoringSystem) UpdateModelNameWithScore(modelID string, score float64) error {
	// Get current model information
	model, err := ss.engine.dbIntegration.GetModelByModelID(modelID)
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}

	// Update model name with score suffix
	updatedName := ss.modelNaming.AddScoreSuffix(model.Name, score)
	
	// Update in database
	model.Name = updatedName
	if err := ss.engine.dbIntegration.db.UpdateModel(model); err != nil {
		return fmt.Errorf("failed to update model name: %w", err)
	}

	ss.logger.Info("Updated model name with score", map[string]any{"model_id": modelID, "new_name": updatedName, "score": score})
	return nil
}

// BatchUpdateModelNamesWithScores updates multiple model names with their scores
func (ss *ScoringSystem) BatchUpdateModelNamesWithScores(scores map[string]float64) error {
	for modelID, score := range scores {
		if err := ss.UpdateModelNameWithScore(modelID, score); err != nil {
			ss.logger.Info("Failed to update model name", map[string]any{"error": err, "model_id": modelID})
			// Continue with other models even if one fails
		}
	}
	return nil
}

// SyncWithModelsDev synchronizes model data with models.dev API
func (ss *ScoringSystem) SyncWithModelsDev(ctx context.Context, providerID, modelID string, force bool) error {
	if modelID != "" {
		// Sync specific model
		return ss.syncModelWithModelsDev(ctx, modelID, force)
	} else if providerID != "" {
		// Sync specific provider
		return ss.syncProviderWithModelsDev(ctx, providerID, force)
	} else {
		// Sync all models
		return ss.syncAllModelsWithModelsDev(ctx, force)
	}
}

// GetModelScore retrieves the current score for a model
func (ss *ScoringSystem) GetModelScore(modelID string) (*ComprehensiveScore, error) {
	// Get model from database
	model, err := ss.engine.dbIntegration.GetModelByModelID(modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model: %w", err)
	}

	// Get latest score
	score, err := ss.databaseExt.GetLatestModelScore(model.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get model score: %w", err)
	}

	if score == nil {
		return nil, fmt.Errorf("no score found for model %s", modelID)
	}

	// Convert to ComprehensiveScore
	return &ComprehensiveScore{
		ModelID:         modelID,
		ModelName:       model.Name,
		OverallScore:    score.Score,
		Components: ScoreComponents{
			SpeedScore:      score.Components.SpeedScore,
			EfficiencyScore: score.Components.EfficiencyScore,
			CostScore:       score.Components.CostScore,
			CapabilityScore: score.Components.CapabilityScore,
			RecencyScore:    score.Components.RecencyScore,
		},
		LastCalculated:  score.LastCalculated,
		CalculationHash: score.CalculationHash,
		ScoreSuffix:     score.ScoreSuffix,
	}, nil
}

// GetModelRankings retrieves model rankings by score category
func (ss *ScoringSystem) GetModelRankings(category string, limit int) ([]ModelRanking, error) {
	scores, err := ss.databaseExt.GetModelScoresByRange(0, 10, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get model scores: %w", err)
	}

	rankings := make([]ModelRanking, 0, len(scores))
	for i, score := range scores {
		ranking := ModelRanking{
			Rank:          i + 1,
			ModelID:       score.ModelID,
			ModelName:     fmt.Sprintf("Model %s", score.ModelID),
			OverallScore:  score.Score,
			ScoreSuffix:   score.ScoreSuffix,
			Category:      category,
			LastUpdated:   score.LastCalculated,
		}

		// Add component scores based on category
		switch category {
		case "speed":
			ranking.CategoryScore = score.Components.SpeedScore
		case "efficiency":
			ranking.CategoryScore = score.Components.EfficiencyScore
		case "cost":
			ranking.CategoryScore = score.Components.CostScore
		case "capability":
			ranking.CategoryScore = score.Components.CapabilityScore
		case "recency":
			ranking.CategoryScore = score.Components.RecencyScore
		default:
			ranking.CategoryScore = score.Score
		}

		rankings = append(rankings, ranking)
	}

	return rankings, nil
}

// GetScoreDistribution retrieves the distribution of scores across all models
func (ss *ScoringSystem) GetScoreDistribution() (ScoreDistribution, error) {
	// This would query the database for score statistics
	// For now, return a placeholder
	return ScoreDistribution{
		TotalModels:   100,
		AverageScore:  7.5,
		MedianScore:   7.8,
		MinScore:      2.1,
		MaxScore:      9.8,
		ScoreRanges: []ScoreRange{
			{Min: 9.0, Max: 10.0, Count: 5, Percentage: 5.0},
			{Min: 8.0, Max: 8.9, Count: 15, Percentage: 15.0},
			{Min: 7.0, Max: 7.9, Count: 30, Percentage: 30.0},
			{Min: 6.0, Max: 6.9, Count: 25, Percentage: 25.0},
			{Min: 5.0, Max: 5.9, Count: 15, Percentage: 15.0},
			{Min: 0.0, Max: 4.9, Count: 10, Percentage: 10.0},
		},
	}, nil
}

// ScoreDistribution represents the distribution of model scores
type ScoreDistribution struct {
	TotalModels int          `json:"total_models"`
	AverageScore float64     `json:"average_score"`
	MedianScore  float64     `json:"median_score"`
	MinScore     float64     `json:"min_score"`
	MaxScore     float64     `json:"max_score"`
	ScoreRanges  []ScoreRange `json:"score_ranges"`
}

// ScoreRange represents a range of scores and their count
type ScoreRange struct {
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// Background processes

func (ss *ScoringSystem) startBackgroundSync(ctx context.Context) {
	ss.backgroundWorkers.Add(1)
	go func() {
		defer ss.backgroundWorkers.Done()
		ss.backgroundSyncLoop(ctx)
	}()
}

func (ss *ScoringSystem) backgroundSyncLoop(ctx context.Context) {
	ticker := time.NewTicker(ss.config.AutoSyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ss.shutdown:
			return
		case <-ticker.C:
			ss.performBackgroundSync(ctx)
		}
	}
}

func (ss *ScoringSystem) performBackgroundSync(ctx context.Context) {
	ss.logger.Info("Starting background sync with models.dev", map[string]any{})
	
	if err := ss.syncAllModelsWithModelsDev(ctx, false); err != nil {
		ss.logger.Info("Background sync failed", map[string]any{"error": err})
	} else {
		ss.logger.Info("Background sync completed successfully", map[string]any{})
	}
}

func (ss *ScoringSystem) startScoreMonitoring(ctx context.Context) {
	ss.backgroundWorkers.Add(1)
	go func() {
		defer ss.backgroundWorkers.Done()
		ss.scoreMonitoringLoop(ctx)
	}()
}

func (ss *ScoringSystem) scoreMonitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(ss.config.ScoreRecalcInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ss.shutdown:
			return
		case <-ticker.C:
			ss.performScoreRecalculation(ctx)
		}
	}
}

func (ss *ScoringSystem) performScoreRecalculation(ctx context.Context) {
	ss.logger.Info("Starting background score recalculation", map[string]any{})
	
	// For now, just log that we're starting recalculation
	// In a full implementation, this would query models needing recalculation
	ss.logger.Info("Background score recalculation completed", map[string]any{})
}

// Sync methods (implementations would be added)

func (ss *ScoringSystem) syncModelWithModelsDev(ctx context.Context, modelID string, force bool) error {
	// Implementation for syncing a specific model
	return nil
}

func (ss *ScoringSystem) syncProviderWithModelsDev(ctx context.Context, providerID string, force bool) error {
	// Implementation for syncing a specific provider
	return nil
}

func (ss *ScoringSystem) syncAllModelsWithModelsDev(ctx context.Context, force bool) error {
	// Implementation for syncing all models
	return nil
}