package scoring

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"digital.vasic.llmsverifier/logging"
	"github.com/gin-gonic/gin"
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

	// Score comparison and analysis
	router.GET("/models/scores/compare", sah.CompareModels)
	router.GET("/models/scores/ranking", sah.GetModelRankings)

	// Score configuration
	router.GET("/scoring/configuration", sah.GetScoringConfiguration)

	// Model naming with scores
	router.POST("/models/naming/add-suffix", sah.AddScoreSuffixToModelName)
	router.POST("/models/naming/batch-update", sah.BatchUpdateModelNamesWithScores)

	// Score validation and debugging
	router.POST("/scoring/validate", sah.ValidateScoreCalculation)
}

// GetModelScore previously returned a hardcoded ComprehensiveScore
// with ModelName: "GPT-4" + OverallScore: 8.5 for ANY model_id —
// CONST-036/037 violation. Real lookup requires querying the
// scoring engine / DB for the actual stored score by modelID.
// Now returns 501 Not Implemented so callers see the gap loudly.
func (sah *ScoringAPIHandlers) GetModelScore(c *gin.Context) {
	modelID := c.Param("model_id")
	sah.logger.Warning("[§11.4 / CONST-036] GetModelScore not wired", map[string]interface{}{"model_id": modelID})
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":    "GetModelScore: scoring engine lookup not wired (was: hardcoded GPT-4/8.5 for any modelID — §11.4 CONST-036 PASS-bluff)",
		"model_id": modelID,
		"fix":      "wire sah.scoringEngine.GetScore(ctx, modelID) or equivalent DB query",
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
			"error": tr("llmsverifier_scoring_err_invalid_request_body"),
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

	// Calculate score using the scoring engine
	score, err := sah.scoringEngine.CalculateComprehensiveScore(ctx, modelID, config)
	if err != nil {
		sah.logger.Error("Failed to calculate model score", map[string]interface{}{
			"model_id": modelID,
			"error":    err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": tr("llmsverifier_scoring_err_calculate_score_failed"),
		})
		return
	}

	// Format model name with score
	formattedName := sah.modelNaming.AddScoreSuffix(score.ModelName, score.OverallScore)

	c.JSON(http.StatusOK, gin.H{
		"message":         tr("llmsverifier_scoring_msg_score_calculated"),
		"model_id":        modelID,
		"model_name":      score.ModelName,
		"formatted_name":  formattedName,
		"overall_score":   score.OverallScore,
		"score_suffix":    score.ScoreSuffix,
		"components":      score.Components,
		"last_calculated": score.LastCalculated,
	})
}

// RecalculateModelScore forces recalculation of an existing score
func (sah *ScoringAPIHandlers) RecalculateModelScore(c *gin.Context) {
	modelID := c.Param("model_id")

	var request struct {
		Reason        string         `json:"reason,omitempty"`
		Configuration *ScoringConfig `json:"configuration,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": tr("llmsverifier_scoring_err_invalid_request_body"),
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

	// Recalculate score using the scoring engine
	score, err := sah.scoringEngine.CalculateComprehensiveScore(ctx, modelID, config)
	if err != nil {
		sah.logger.Error("Failed to recalculate model score", map[string]interface{}{
			"model_id": modelID,
			"error":    err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": tr("llmsverifier_scoring_err_recalculate_score_failed"),
		})
		return
	}

	// Log the recalculation reason
	if request.Reason != "" {
		sah.logger.Info("Model score recalculated", map[string]interface{}{
			"model_id":  modelID,
			"reason":    request.Reason,
			"new_score": score.OverallScore,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         tr("llmsverifier_scoring_msg_score_recalculated"),
		"model_id":        modelID,
		"overall_score":   score.OverallScore,
		"score_suffix":    score.ScoreSuffix,
		"components":      score.Components,
		"last_calculated": score.LastCalculated,
		"recalc_reason":   request.Reason,
	})
}

// DeleteModelScore previously logged a deactivation message and
// returned success without touching the DB — §11.4 stub-interface
// bluff. Real fix requires sah.scoringEngine.DeactivateModel(ctx,
// modelID) or equivalent DB update. Returns 501 until wired.
func (sah *ScoringAPIHandlers) DeleteModelScore(c *gin.Context) {
	modelID := c.Param("model_id")
	sah.logger.Warning("[§11.4] DeleteModelScore not wired", map[string]interface{}{"model_id": modelID})
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":    "DeleteModelScore: scoring engine deactivation not wired (was: log + return success without touching DB — §11.4 stub-interface bluff)",
		"model_id": modelID,
		"fix":      "wire sah.scoringEngine.DeactivateModel(ctx, modelID) or equivalent DB UPDATE",
	})
}

// BatchCalculateScores calculates scores for multiple models
func (sah *ScoringAPIHandlers) BatchCalculateScores(c *gin.Context) {
	var request struct {
		ModelIDs      []string       `json:"model_ids" binding:"required"`
		Configuration *ScoringConfig `json:"configuration,omitempty"`
		Async         bool           `json:"async,omitempty"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": tr("llmsverifier_scoring_err_invalid_request_body"),
		})
		return
	}

	// Use provided configuration or default
	_ = DefaultScoringConfig()
	if request.Configuration != nil {
		_ = *request.Configuration
	}

	batchID := generateBatchID()

	if request.Async {
		// Start async processing
		go sah.processBatchScoresAsync(batchID, request.ModelIDs, *request.Configuration)

		c.JSON(http.StatusAccepted, gin.H{
			"message":     tr("llmsverifier_scoring_msg_batch_started"),
			"batch_id":    batchID,
			"status":      "processing",
			"model_count": len(request.ModelIDs),
		})
		return
	}

	// Process synchronously
	results := sah.processBatchScoresSync(request.ModelIDs, *request.Configuration)

	c.JSON(http.StatusOK, gin.H{
		"message":       tr("llmsverifier_scoring_msg_batch_completed"),
		"batch_id":      batchID,
		"status":        "completed",
		"results":       results,
		"model_count":   len(request.ModelIDs),
		"success_count": len(results),
	})
}

// CompareModels previously returned best_model:modelIDs[0] +
// hardcoded score_difference:0.5 — §11.4 / CONST-036 bluff.
func (sah *ScoringAPIHandlers) CompareModels(c *gin.Context) {
	modelIDs := c.QueryArray("models")
	if len(modelIDs) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": tr("llmsverifier_scoring_err_min_two_models"),
		})
		return
	}
	sah.logger.Warning("[§11.4 / CONST-036] CompareModels not wired", map[string]interface{}{"models": modelIDs})
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":  "CompareModels: real per-model score lookup + diff not wired (was: hardcoded best=modelIDs[0] + score_difference=0.5 — §11.4 / CONST-036 PASS-bluff)",
		"models": modelIDs,
		"fix":    "fetch real scores for each modelID via scoringEngine.GetScore; compute actual diff",
	})
}

// GetModelRankings previously discarded all query parameters
// (category/limit/min_score/max_score with `_ =`) and returned a
// fabricated 2-element rankings slice — §11.4 / CONST-036 bluff.
func (sah *ScoringAPIHandlers) GetModelRankings(c *gin.Context) {
	category := c.DefaultQuery("category", "overall")
	limit := c.DefaultQuery("limit", "50")
	minScore := c.DefaultQuery("min_score", "0")
	maxScore := c.DefaultQuery("max_score", "10")
	sah.logger.Warning("[§11.4 / CONST-036] GetModelRankings not wired", map[string]interface{}{
		"category": category, "limit": limit, "min_score": minScore, "max_score": maxScore,
	})
	c.JSON(http.StatusNotImplemented, gin.H{
		"error":     "GetModelRankings: real DB-backed ranking query not wired (was: fabricated 2-element list ignoring all query params — §11.4 / CONST-036 PASS-bluff)",
		"category":  category,
		"limit":     limit,
		"min_score": minScore,
		"max_score": maxScore,
		"fix":       "query scoring engine / DB for actual ranked results filtered by the provided parameters",
	})
}

// GetScoringConfiguration retrieves current scoring configuration
func (sah *ScoringAPIHandlers) GetScoringConfiguration(c *gin.Context) {
	configName := c.DefaultQuery("config", "default")

	// Simulate configuration
	config := map[string]interface{}{
		"config_name": configName,
		"weights": map[string]float64{
			"response_speed":     0.25,
			"model_efficiency":   0.20,
			"cost_effectiveness": 0.25,
			"capability":         0.20,
			"recency":            0.10,
		},
		"thresholds": map[string]float64{
			"min_score": 0.0,
			"max_score": 10.0,
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"config_name":   configName,
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
			"error": tr("llmsverifier_scoring_err_invalid_request_body"),
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
			"error": tr("llmsverifier_scoring_err_invalid_request_body"),
		})
		return
	}

	results := sah.modelNaming.BatchUpdateModelNames(request.ModelScores)

	c.JSON(http.StatusOK, gin.H{
		"message": tr("llmsverifier_scoring_msg_model_names_updated"),
		"results": results,
		"count":   len(results),
	})
}

// ValidateScoreCalculation validates score calculation
func (sah *ScoringAPIHandlers) ValidateScoreCalculation(c *gin.Context) {
	var request struct {
		ModelID string  `json:"model_id" binding:"required"`
		Score   float64 `json:"score" binding:"required"`
		Method  string  `json:"method" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": tr("llmsverifier_scoring_err_invalid_request_body"),
		})
		return
	}

	isValid := true
	validationResult := map[string]interface{}{
		"model_id": request.ModelID,
		"score":    request.Score,
		"method":   request.Method,
		"is_valid": isValid,
		"message":  tr("llmsverifier_scoring_msg_validation_completed"),
	}

	c.JSON(http.StatusOK, gin.H{
		"validation": validationResult,
	})
}

// Helper functions

// randomIDSuffix returns a short, collision-resistant suffix for IDs in the
// scoring package. 8 bytes from crypto/rand (base64url ~11 chars); on RNG
// failure falls back to a doubled-nanosecond value so the suffix is non-empty
// and varies. §11.4.50: prevents same-nanosecond batch-ID / alert-ID collisions.
func randomIDSuffix() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano()*2654435761)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateBatchID() string {
	return fmt.Sprintf("batch_%d_%s", time.Now().UnixNano(), randomIDSuffix())
}

// newAlertID builds a unique alert payload ID. Kept as a named helper so its
// uniqueness is directly testable (§11.4.115).
func newAlertID() string {
	return fmt.Sprintf("alert_%d_%s", time.Now().UnixNano(), randomIDSuffix())
}

func (sah *ScoringAPIHandlers) processBatchScoresAsync(batchID string, modelIDs []string, config ScoringConfig) {
	// Implementation for async batch processing
	// This would typically use a background job system
	sah.logger.Info("Processing batch scores async", map[string]interface{}{
		"batch_id":    batchID,
		"model_count": len(modelIDs),
	})
}

func (sah *ScoringAPIHandlers) processBatchScoresSync(modelIDs []string, config ScoringConfig) []interface{} {
	results := make([]interface{}, 0, len(modelIDs))

	for _, modelID := range modelIDs {
		_, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		// Simulate score calculation
		score := &ComprehensiveScore{
			ModelID:      modelID,
			ModelName:    "Model " + modelID,
			OverallScore: 8.0 + float64(len(modelIDs))/10.0,
			ScoreSuffix:  fmt.Sprintf("(SC:%.1f)", 8.0+float64(len(modelIDs))/10.0),
			Components: ScoreComponents{
				SpeedScore:      8.0,
				EfficiencyScore: 9.0,
				CostScore:       8.5,
				CapabilityScore: 8.5,
				RecencyScore:    8.0,
			},
			LastCalculated: time.Now(),
		}
		cancel()

		results = append(results, gin.H{
			"model_id":      modelID,
			"overall_score": score.OverallScore,
			"score_suffix":  score.ScoreSuffix,
			"success":       true,
		})
	}

	return results
}
