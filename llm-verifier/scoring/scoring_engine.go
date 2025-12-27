package scoring

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"llm-verifier/database"
)

// ScoringEngine handles the core scoring logic
type ScoringEngine struct {
	modelsDevClient *ModelsDevClient
	dbIntegration   *DatabaseIntegration
	weights         ScoreWeights
}

// NewScoringEngine creates a new scoring engine
func NewScoringEngine(db *database.Database, modelsDevClient *ModelsDevClient, logger interface{}) *ScoringEngine {
	return &ScoringEngine{
		modelsDevClient: modelsDevClient,
		dbIntegration:   NewDatabaseIntegration(db),
		weights:         DefaultScoreWeights(),
	}
}

// CalculateComprehensiveScore calculates a comprehensive score for a model
func (se *ScoringEngine) CalculateComprehensiveScore(ctx context.Context, modelID string, config ScoringConfig) (*ComprehensiveScore, error) {
	weights := &config.Weights
	
	// Fetch model data from models.dev
	modelData, err := se.modelsDevClient.FetchModelByID(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch model data: %w", err)
	}
	
	// Get existing model from database to get pricing and other data
	dbModel, err := se.dbIntegration.GetModelByModelID(modelID)
	if err != nil {
		log.Printf("Warning: Could not find model %s in database: %v", modelID, err)
		// Continue with default values
		dbModel = &database.Model{
			ModelID: modelID,
			Name:    modelData.Name,
		}
	}
	
	// Calculate individual component scores
	responseScore := se.calculateResponseSpeedScore(modelData, dbModel)
	efficiencyScore := se.calculateModelEfficiencyScore(modelData, dbModel)
	costScore := se.calculateCostEffectivenessScore(modelData, dbModel)
	capabilityScore := se.calculateCapabilityScore(modelData, dbModel)
	recencyScore := se.calculateRecencyScore(modelData, dbModel)
	
	// Calculate weighted total score
	totalScore := (responseScore * weights.ResponseSpeed) +
		(efficiencyScore * weights.ModelEfficiency) +
		(costScore * weights.CostEffectiveness) +
		(capabilityScore * weights.Capability) +
		(recencyScore * weights.Recency)
	
	// Ensure score is within bounds
	totalScore = math.Max(0, math.Min(10, totalScore))
	
	score := &ComprehensiveScore{
		ModelID:     modelID,
		ModelName:   modelData.Model,
		OverallScore: totalScore,
		ScoreSuffix: fmt.Sprintf("(SC:%.1f)", totalScore),
		Components: ScoreComponents{
			SpeedScore:      responseScore,
			EfficiencyScore: efficiencyScore,
			CostScore:       costScore,
			CapabilityScore: capabilityScore,
			RecencyScore:    recencyScore,
		},
		LastCalculated: time.Now(),
		DataSource:   "models.dev",
	}
	
	// Update database with new scores
	if err := se.dbIntegration.UpdateModelScores(dbModel.ID, score); err != nil {
		log.Printf("Warning: Failed to update model scores in database: %v", err)
	}
	
	// Create verification score record
	if err := se.dbIntegration.CreateVerificationScore(dbModel.ID, score); err != nil {
		log.Printf("Warning: Failed to create verification score: %v", err)
	}
	
	// Log scoring event
	if err := se.dbIntegration.CreateScoringEvent("model_scored", 
		fmt.Sprintf("Calculated score %.1f for model %s", totalScore, modelID),
		&dbModel.ID, map[string]interface{}{
			"model_id": modelID,
			"score": totalScore,
			"components": score.Components,
		}); err != nil {
		log.Printf("Warning: Failed to create scoring event: %v", err)
	}
	
	return score, nil
}

// CalculateBatchScores calculates scores for multiple models
func (se *ScoringEngine) CalculateBatchScores(ctx context.Context, modelIDs []string, weights *ScoreWeights) ([]*ModelScore, error) {
	var scores []*ModelScore
	
	for _, modelID := range modelIDs {
		score, err := se.CalculateModelScore(ctx, modelID, weights)
		if err != nil {
			log.Printf("Warning: Failed to calculate score for model %s: %v", modelID, err)
			continue
		}
		scores = append(scores, score)
	}
	
	return scores, nil
}

// GetTopModels gets top scoring models
func (se *ScoringEngine) GetTopModels(ctx context.Context, limit int) ([]*database.Model, error) {
	return se.dbIntegration.GetTopScoringModels(limit)
}

// GetModelsByScoreRange gets models within a score range
func (se *ScoringEngine) GetModelsByScoreRange(ctx context.Context, minScore, maxScore float64, limit int) ([]*database.Model, error) {
	return se.dbIntegration.ListModelsByScore(minScore, maxScore, limit)
}

// Scoring component calculation methods

func (se *ScoringEngine) calculateResponseSpeedScore(modelData *ModelData, dbModel *database.Model) float64 {
	baseScore := 5.0
	
	// Factor in average response time if available
	if dbModel.AverageResponseTimeMs > 0 {
		if dbModel.AverageResponseTimeMs < 1000 {
			baseScore += 3.0
		} else if dbModel.AverageResponseTimeMs < 3000 {
			baseScore += 2.0
		} else if dbModel.AverageResponseTimeMs < 5000 {
			baseScore += 1.0
		} else {
			baseScore -= 1.0
		}
	}
	
	// Factor in throughput if available
	if modelData.ThroughputRPS > 0 {
		if modelData.ThroughputRPS > 10 {
			baseScore += 1.0
		} else if modelData.ThroughputRPS > 5 {
			baseScore += 0.5
		}
	}
	
	return math.Max(0, math.Min(10, baseScore))
}

func (se *ScoringEngine) calculateModelEfficiencyScore(modelData *ModelData, dbModel *database.Model) float64 {
	baseScore := 5.0
	
	// Factor in parameter count efficiency
	if dbModel.ParameterCount != nil && *dbModel.ParameterCount > 0 {
		params := *dbModel.ParameterCount
		if params < 1000000000 { // Less than 1B parameters
			baseScore += 2.0
		} else if params < 10000000000 { // Less than 10B parameters
			baseScore += 1.0
		} else {
			baseScore -= 1.0
		}
	}
	
	// Factor in context window efficiency
	if dbModel.ContextWindowTokens != nil && *dbModel.ContextWindowTokens > 0 {
		context := *dbModel.ContextWindowTokens
		if context > 128000 {
			baseScore += 2.0
		} else if context > 32000 {
			baseScore += 1.0
		}
	}
	
	// Factor in multimodal capabilities
	if dbModel.IsMultimodal {
		baseScore += 1.0
	}
	
	return math.Max(0, math.Min(10, baseScore))
}

func (se *ScoringEngine) calculateCostEffectivenessScore(modelData *ModelData, dbModel *database.Model) float64 {
	baseScore := 5.0
	
	// This would ideally use pricing data from the database
	// For now, we'll use a basic heuristic based on model size and capabilities
	
	if dbModel.ParameterCount != nil && *dbModel.ParameterCount > 0 {
		params := *dbModel.ParameterCount
		if params < 1000000000 { // Smaller models are more cost-effective
			baseScore += 2.0
		} else if params < 5000000000 {
			baseScore += 1.0
		} else {
			baseScore -= 1.0
		}
	}
	
	// Factor in open source (more cost-effective)
	if dbModel.OpenSource {
		baseScore += 2.0
	}
	
	// Factor in multimodal capabilities (better value proposition)
	if dbModel.IsMultimodal {
		baseScore += 1.0
	}
	
	return math.Max(0, math.Min(10, baseScore))
}

func (se *ScoringEngine) calculateCapabilityScore(modelData *ModelData, dbModel *database.Model) float64 {
	baseScore := 5.0
	
	// Factor in code capabilities from verification results
	verificationResults, err := se.dbIntegration.GetVerificationResults([]int64{dbModel.ID})
	if err == nil && len(verificationResults) > 0 {
		latestResult := verificationResults[0]
		
		if latestResult.SupportsCodeGeneration {
			baseScore += 2.0
		}
		if latestResult.SupportsCodeCompletion {
			baseScore += 1.0
		}
		if latestResult.SupportsCodeReview {
			baseScore += 1.0
		}
		if latestResult.SupportsCodeExplanation {
			baseScore += 0.5
		}
		if latestResult.SupportsDebugging {
			baseScore += 1.0
		}
		if latestResult.SupportsReasoning {
			baseScore += 1.0
		}
	}
	
	// Factor in multimodal capabilities
	if dbModel.IsMultimodal {
		baseScore += 1.0
	}
	
	// Factor in reasoning capabilities
	if dbModel.SupportsReasoning {
		baseScore += 1.0
	}
	
	return math.Max(0, math.Min(10, baseScore))
}

func (se *ScoringEngine) calculateRecencyScore(modelData *ModelData, dbModel *database.Model) float64 {
	baseScore := 5.0
	
	// Factor in release date
	if dbModel.ReleaseDate != nil {
		age := time.Since(*dbModel.ReleaseDate).Days()
		if age < 365 { // Less than 1 year old
			baseScore += 3.0
		} else if age < 730 { // Less than 2 years old
			baseScore += 2.0
		} else if age < 1095 { // Less than 3 years old
			baseScore += 1.0
		} else {
			baseScore -= 1.0
		}
	}
	
	// Factor in training data cutoff
	if dbModel.TrainingDataCutoff != nil {
		cutoffAge := time.Since(*dbModel.TrainingDataCutoff).Days()
		if cutoffAge < 730 { // Less than 2 years old
			baseScore += 1.0
		} else {
			baseScore -= 0.5
		}
	}
	
	// Factor in last verification date
	if dbModel.LastVerified != nil {
		verificationAge := time.Since(*dbModel.LastVerified).Days()
		if verificationAge < 30 { // Verified within last month
			baseScore += 1.0
		} else if verificationAge < 90 { // Verified within last 3 months
			baseScore += 0.5
		}
	}
	
	return math.Max(0, math.Min(10, baseScore))
}

// DefaultScoreWeights returns default scoring weights
func DefaultScoreWeights() ScoreWeights {
	return ScoreWeights{
		SpeedScore:      0.25,
		EfficiencyScore: 0.20,
		CostScore:       0.25,
		CapabilityScore: 0.20,
		RecencyScore:    0.10,
	}
}

// SetWeights updates the scoring weights
func (se *ScoringEngine) SetWeights(weights ScoreWeights) {
	se.weights = weights
}

// GetWeights returns the current scoring weights
func (se *ScoringEngine) GetWeights() ScoreWeights {
	return se.weights
}