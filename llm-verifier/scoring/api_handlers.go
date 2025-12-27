package scoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"llm-verifier/logging"
)

// ScoringAPIHandlers handles HTTP requests for scoring functionality
type ScoringAPIHandlers struct {
	scoringEngine *ScoringEngine
	modelNaming   *ModelNaming
	logger        *logging.Logger
}

// NewScoringAPIHandlers creates new API handlers for scoring
func NewScoringAPIHandlers(scoringEngine *ScoringEngine, logger *logging.Logger) *ScoringAPIHandlers {
	return &ScoringAPIHandlers{
		scoringEngine: scoringEngine,
		modelNaming:   NewModelNaming(),
		logger:        logger,
	}
}

// RegisterRoutes registers all scoring-related API routes
func (sah *ScoringAPIHandlers) RegisterRoutes(router *gin.RouterGroup) {
	// Model scoring endpoints
	router.GET("/models/:model_id/score", sah.GetModelScore)
	router.POST("/models/:model_id/score/calculate", sah.CalculateModelScore)
	router.PUT("/models/:model_id/score/recalculate", sah.RecalculateModelScore)
	router.DELETE("/models/:model_id/score", sah.DeleteModelScore)

	// Batch scoring endpoints
	router.POST("/models/scores/batch", sah.BatchCalculateScores)
	router.GET("/models/scores/batch/:batch_id", sah.GetBatchScoreResults)
	router.GET("/models/scores/batch/:batch_id/status", sah.GetBatchScoreStatus)

	// Score comparison and analysis
	router.GET("/models/scores/compare", sah.CompareModels)
	router.GET("/models/scores/ranking", sah.GetModelRankings)
	router.GET("/models/scores/distribution", sah.GetScoreDistribution)

	// Score configuration
	router.GET("/scoring/configuration", sah.GetScoringConfiguration)
	router.PUT("/scoring/configuration", sah.UpdateScoringConfiguration)
	router.POST("/scoring/configuration/test", sah.TestScoringConfiguration)

	// Score history and trends
	router.GET("/models/:model_id/score/history", sah.GetModelScoreHistory)
	router.GET("/models/:model_id/score/trends", sah.GetModelScoreTrends)
	router.GET("/scoring/changes", sah.GetRecentScoreChanges)

	// Model naming with scores
	router.POST("/models/naming/add-suffix", sah.AddScoreSuffixToModelName)
	router.POST("/models/naming/remove-suffix", sah.RemoveScoreSuffixFromModelName)
	router.POST("/models/naming/batch-update", sah.BatchUpdateModelNamesWithScores)

	// External data integration
	router.POST("/scoring/sync-models-dev", sah.SyncWithModelsDev)
	router.GET("/scoring/models-dev/:model_id", sah.GetModelsDevData)
	router.POST("/scoring/models-dev/fetch", sah.FetchModelsDevData)

	// Score validation and debugging
	router.POST("/scoring/validate", sah.ValidateScoreCalculation)
	router.GET("/scoring/debug/:model_id", sah.DebugScoreCalculation)
}

// GetModelScore retrieves the current score for a model
func (sah *ScoringAPIHandlers) GetModelScore(c *gin.Context) {
	modelID := c.Param("model_id")
	
	// Get latest score from database
	score, err := sah.scoringEngine.db.GetLatestModelScoreByModelID(modelID)
	if err != nil {
		sah.logger.Error("Failed to get model score", "error", err, "model_id", modelID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve model score",
		})
		return
	}

	if score == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "No score found for model",
		})
		return
	}

	// Add formatted model name with score suffix
	formattedName := sah.modelNaming.AddScoreSuffix(score.ModelName, score.OverallScore)

	c.JSON(http.StatusOK, gin.H{
		"model_id":         modelID,
		"model_name":       score.ModelName,
		"formatted_name":   formattedName,
		"overall_score":    score.OverallScore,
		"score_suffix":     score.ScoreSuffix,
		"components":       score.Components,
		"last_calculated":  score.LastCalculated,
		"calculation_hash": score.CalculationHash,
	})
}

// CalculateModelScore calculates a new score for a model
func (sah *ScoringAPIHandlers) CalculateModelScore(c *gin.Context) {
	modelID := c.Param("model_id")
	
	var request struct {
		Configuration *ScoringConfig `json:"configuration,omitempty"`
		ForceRecalc   bool           `json:"force_recalculation,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Use provided configuration or default
	config := DefaultScoringConfig()
	if request.Configuration != nil {
		config = *request.Configuration
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Calculate score
	score, err := sah.scoringEngine.CalculateComprehensiveScore(ctx, modelID, config)
	if err != nil {
		sah.logger.Error("Failed to calculate model score", "error", err, "model_id", modelID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to calculate score: %v", err),
		})
		return
	}

	// Format model name with score
	formattedName := sah.modelNaming.AddScoreSuffix(score.ModelName, score.OverallScore)

	c.JSON(http.StatusOK, gin.H{
		"message":          "Score calculated successfully",
		"model_id":         modelID,
		"model_name":       score.ModelName,
		"formatted_name":   formattedName,
		"overall_score":    score.OverallScore,
		"score_suffix":     score.ScoreSuffix,
		"components":       score.Components,
		"last_calculated":  score.LastCalculated,
		"calculation_hash": score.CalculationHash,
	})
}

// RecalculateModelScore forces recalculation of an existing score
func (sah *ScoringAPIHandlers) RecalculateModelScore(c *gin.Context) {
	modelID := c.Param("model_id")
	
	var request struct {
		Reason      string         `json:"reason,omitempty"`
		Configuration *ScoringConfig `json:"configuration,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Use provided configuration or default
	config := DefaultScoringConfig()
	if request.Configuration != nil {
		config = *request.Configuration
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Force recalculation
	score, err := sah.scoringEngine.CalculateComprehensiveScore(ctx, modelID, config)
	if err != nil {
		sah.logger.Error("Failed to recalculate model score", "error", err, "model_id", modelID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to recalculate score: %v", err),
		})
		return
	}

	// Log the recalculation reason
	if request.Reason != "" {
		sah.logger.Info("Model score recalculated", "model_id", modelID, "reason", request.Reason, "new_score", score.OverallScore)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "Score recalculated successfully",
		"model_id":         modelID,
		"overall_score":    score.OverallScore,
		"score_suffix":     score.ScoreSuffix,
		"components":       score.Components,
		"last_calculated":  score.LastCalculated,
		"calculation_hash": score.CalculationHash,
		"recalc_reason":    request.Reason,
	})
}

// DeleteModelScore removes a model score (soft delete)
func (sah *ScoringAPIHandlers) DeleteModelScore(c *gin.Context) {
	modelID := c.Param("model_id")
	
	// Mark score as inactive instead of deleting
	err := sah.scoringEngine.db.DeactivateModelScore(modelID)
	if err != nil {
		sah.logger.Error("Failed to deactivate model score", "error", err, "model_id", modelID)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to deactivate model score",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Model score deactivated successfully",
		"model_id":  modelID,
	})
}

// BatchCalculateScores calculates scores for multiple models
func (sah *ScoringAPIHandlers) BatchCalculateScores(c *gin.Context) {
	var request struct {
		ModelIDs    []string       `json:"model_ids" binding:"required"`
		Configuration *ScoringConfig `json:"configuration,omitempty"`
		Async       bool           `json:"async,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	// Use provided configuration or default
	config := DefaultScoringConfig()
	if request.Configuration != nil {
		config = *request.Configuration
	}

	batchID := generateBatchID()
	
	if request.Async {
		// Start async processing
		go sah.processBatchScoresAsync(batchID, request.ModelIDs, config)
		
		c.JSON(http.StatusAccepted, gin.H{
			"message":   "Batch score calculation started",
			"batch_id":  batchID,
			"status":    "processing",
			"model_count": len(request.ModelIDs),
		})
		return
	}

	// Process synchronously
	results := sah.processBatchScoresSync(request.ModelIDs, config)
	
	c.JSON(http.StatusOK, gin.H{
		"message":      "Batch score calculation completed",
		"batch_id":     batchID,
		"status":       "completed",
		"results":      results,
		"model_count":  len(request.ModelIDs),
		"success_count": len(results),
	})
}

// CompareModels compares scores between multiple models
func (sah *ScoringAPIHandlers) CompareModels(c *gin.Context) {
	modelIDs := c.QueryArray("models")
	if len(modelIDs) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "At least 2 models are required for comparison",
		})
		return
	}

	comparison, err := sah.scoringEngine.CompareModels(modelIDs)
	if err != nil {
		sah.logger.Error("Failed to compare models", "error", err, "models", modelIDs)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to compare models: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"models":      modelIDs,
		"comparison":  comparison,
	})
}

// GetModelRankings retrieves model rankings by score
func (sah *ScoringAPIHandlers) GetModelRankings(c *gin.Context) {
	category := c.DefaultQuery("category", "overall")
	limitStr := c.DefaultQuery("limit", "50")
	minScoreStr := c.DefaultQuery("min_score", "0")
	maxScoreStr := c.DefaultQuery("max_score", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid limit parameter",
		})
		return
	}

	minScore, err := strconv.ParseFloat(minScoreStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid min_score parameter",
		})
		return
	}

	maxScore, err := strconv.ParseFloat(maxScoreStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid max_score parameter",
		})
		return
	}

	rankings, err := sah.scoringEngine.GetModelRankings(category, minScore, maxScore, limit)
	if err != nil {
		sah.logger.Error("Failed to get model rankings", "error", err, "category", category)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to get rankings: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"category":  category,
		"limit":     limit,
		"min_score": minScore,
		"max_score": maxScore,
		"rankings":  rankings,
	})
}

// GetScoringConfiguration retrieves current scoring configuration
func (sah *ScoringAPIHandlers) GetScoringConfiguration(c *gin.Context) {
	configName := c.DefaultQuery("config", "default")
	
	config, err := sah.scoringEngine.db.GetScoringConfiguration(configName)
	if err != nil {
		sah.logger.Error("Failed to get scoring configuration", "error", err, "config", configName)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve scoring configuration",
		})
		return
	}

	if config == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Scoring configuration not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"config_name": configName,
		"configuration": config,
	})
}

// AddScoreSuffixToModelName adds score suffix to a model name
func (sah *ScoringAPIHandlers) AddScoreSuffixToModelName(c *gin.Context) {
	var request struct {
		ModelName string  `json:"model_name" binding:"required"`
		Score     float64 `json:"score" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	updatedName := sah.modelNaming.AddScoreSuffix(request.ModelName, request.Score)

	c.JSON(http.StatusOK, gin.H{
		"original_name": request.ModelName,
		"updated_name":  updatedName,
		"score":         request.Score,
		"score_suffix":  sah.modelNaming.GenerateScoreSuffix(request.Score),
	})
}

// BatchUpdateModelNamesWithScores updates multiple model names with scores
func (sah *ScoringAPIHandlers) BatchUpdateModelNamesWithScores(c *gin.Context) {
	var request struct {
		ModelScores map[string]float64 `json:"model_scores" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	results := sah.modelNaming.BatchUpdateModelNames(request.ModelScores)

	c.JSON(http.StatusOK, gin.H{
		"message": "Model names updated successfully",
		"results": results,
		"count":   len(results),
	})
}

// SyncWithModelsDev synchronizes data with models.dev API
func (sah *ScoringAPIHandlers) SyncWithModelsDev(c *gin.Context) {
	var request struct {
		ProviderID string `json:"provider_id,omitempty"`
		ModelID    string `json:"model_id,omitempty"`
		ForceSync  bool   `json:"force_sync,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
		})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var syncResult interface{}
	var err error

	if request.ModelID != "" {
		// Sync specific model
		syncResult, err = sah.scoringEngine.SyncModelWithModelsDev(ctx, request.ModelID, request.ForceSync)
	} else if request.ProviderID != "" {
		// Sync specific provider
		syncResult, err = sah.scoringEngine.SyncProviderWithModelsDev(ctx, request.ProviderID, request.ForceSync)
	} else {
		// Sync all models
		syncResult, err = sah.scoringEngine.SyncAllModelsWithModelsDev(ctx, request.ForceSync)
	}

	if err != nil {
		sah.logger.Error("Failed to sync with models.dev", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Sync failed: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Sync completed successfully",
		"result":  syncResult,
	})
}

// Helper functions

func generateBatchID() string {
	return fmt.Sprintf("batch_%d", time.Now().UnixNano())
}

func (sah *ScoringAPIHandlers) processBatchScoresAsync(batchID string, modelIDs []string, config ScoringConfig) {
	// Implementation for async batch processing
	// This would typically use a background job system
}

func (sah *ScoringAPIHandlers) processBatchScoresSync(modelIDs []string, config ScoringConfig) []interface{} {
	results := make([]interface{}, 0, len(modelIDs))
	
	for _, modelID := range modelIDs {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		
		score, err := sah.scoringEngine.CalculateComprehensiveScore(ctx, modelID, config)
		cancel()
		
		if err != nil {
			results = append(results, gin.H{
				"model_id": modelID,
				"error":    err.Error(),
				"success":  false,
			})
		} else {
			results = append(results, gin.H{
				"model_id":      modelID,
				"overall_score": score.OverallScore,
				"score_suffix":  score.ScoreSuffix,
				"success":       true,
			})
		}
	}
	
	return results
}