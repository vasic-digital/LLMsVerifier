package scoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"llm-verifier/database"
)

// DatabaseIntegration provides database operations for scoring using existing CRUD methods
type DatabaseIntegration struct {
	db *database.Database
}

// NewDatabaseIntegration creates new database integration
func NewDatabaseIntegration(db *database.Database) *DatabaseIntegration {
	return &DatabaseIntegration{
		db: db,
	}
}

// GetModelByID retrieves a model by ID
func (di *DatabaseIntegration) GetModelByID(modelID int64) (*database.Model, error) {
	return di.db.GetModel(modelID)
}

// GetModelByModelID retrieves a model by its model_id field
func (di *DatabaseIntegration) GetModelByModelID(modelID string) (*database.Model, error) {
	models, err := di.db.ListModels(map[string]interface{}{
		"model_id": modelID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get model by model_id: %w", err)
	}
	
	if len(models) == 0 {
		return nil, fmt.Errorf("model not found: %s", modelID)
	}
	
	return models[0], nil
}

// UpdateModelScores updates model scores in the database
func (di *DatabaseIntegration) UpdateModelScores(modelID int64, score interface{}) error {
	model, err := di.db.GetModel(modelID)
	if err != nil {
		return fmt.Errorf("failed to get model: %w", err)
	}
	
	// Handle both ModelScore and ComprehensiveScore types
	switch s := score.(type) {
	case *ModelScore:
		model.OverallScore = s.Score
		model.ResponsivenessScore = s.Components.SpeedScore
		model.ReliabilityScore = s.Components.EfficiencyScore
		model.CodeCapabilityScore = s.Components.CapabilityScore
		model.FeatureRichnessScore = s.Components.CostScore
		model.ValuePropositionScore = s.Components.RecencyScore
	case *ComprehensiveScore:
		model.OverallScore = s.OverallScore
		model.ResponsivenessScore = s.Components.SpeedScore
		model.ReliabilityScore = s.Components.EfficiencyScore
		model.CodeCapabilityScore = s.Components.CapabilityScore
		model.FeatureRichnessScore = s.Components.CostScore
		model.ValuePropositionScore = s.Components.RecencyScore
	default:
		return fmt.Errorf("unsupported score type: %T", score)
	}
	
	return di.db.UpdateModel(model)
}

// CreateVerificationScore creates a verification score record
func (di *DatabaseIntegration) CreateVerificationScore(modelID int64, score interface{}) error {
	// Handle both ModelScore and ComprehensiveScore types
	var overallScore float64
	var components ScoreComponents
	
	switch s := score.(type) {
	case *ModelScore:
		overallScore = s.Score
		components = s.Components
	case *ComprehensiveScore:
		overallScore = s.OverallScore
		components = s.Components
	default:
		return fmt.Errorf("unsupported score type: %T", score)
	}
	
	// Convert our score to verification score format
	verificationScore := &database.VerificationScore{
		ModelID:              modelID,
		Score:                int(overallScore * 10), // Convert 0-10 to 0-100
		ScoreType:            "comprehensive_scoring",
		ScoringMethod:        "weighted_algorithm",
		Category:             di.getScoreCategory(overallScore),
		CodeCorrectnessScore: intPtr(int(components.CapabilityScore * 10)),
		CodeQualityScore:     intPtr(int(components.EfficiencyScore * 10)),
		CodeSpeedScore:       intPtr(int(components.SpeedScore * 10)),
		ErrorHandlingScore:   intPtr(int(components.RecencyScore * 10)),
		ContextUnderstandingScore: intPtr(int(components.CostScore * 10)),
		Evidence:             di.createScoreEvidence(score),
		BenchmarkVersion:     "1.0",
		ScoredBy:             "scoring_system",
		ConfidenceLevel:      85,
		ScoredAt:             time.Now(),
	}
	
	_, err := di.db.CreateVerificationScore(verificationScore)
	return err
}

// GetLatestVerificationScore gets the latest verification score for a model
func (di *DatabaseIntegration) GetLatestVerificationScore(modelID int64) (*database.VerificationScore, error) {
	return di.db.GetLatestVerificationScore(modelID, "comprehensive_scoring")
}

// ListModelsByScore gets models filtered by score range
func (di *DatabaseIntegration) ListModelsByScore(minScore, maxScore float64, limit int) ([]*database.Model, error) {
	filters := map[string]interface{}{
		"min_score": minScore,
	}
	if limit > 0 {
		filters["limit"] = limit
	}
	
	return di.db.ListModels(filters)
}

// GetTopScoringModels gets the top scoring models
func (di *DatabaseIntegration) GetTopScoringModels(limit int) ([]*database.Model, error) {
	// This is a simplified implementation - in a real scenario you'd have a more complex query
	models, err := di.db.ListModels(map[string]interface{}{
		"limit": limit,
	})
	if err != nil {
		return nil, err
	}
	
	// Sort by overall score (descending)
	for i := 0; i < len(models)-1; i++ {
		for j := i + 1; j < len(models); j++ {
			if models[i].OverallScore < models[j].OverallScore {
				models[i], models[j] = models[j], models[i]
			}
		}
	}
	
	return models, nil
}

// CreateScoringEvent creates an event for scoring operations
func (di *DatabaseIntegration) CreateScoringEvent(eventType, message string, modelID *int64, details map[string]interface{}) error {
	detailsJSON, err := json.Marshal(details)
	if err != nil {
		return fmt.Errorf("failed to marshal details: %w", err)
	}
	
	event := &database.Event{
		EventType: eventType,
		Severity:  "info",
		Title:     "Scoring Operation",
		Message:   message,
		Details:   stringPtr(string(detailsJSON)),
		ModelID:   modelID,
		CreatedAt: time.Now(),
	}
	
	return di.db.CreateEvent(event)
}

// GetVerificationResults gets verification results for models
func (di *DatabaseIntegration) GetVerificationResults(modelIDs []int64) ([]*database.VerificationResult, error) {
	if len(modelIDs) == 0 {
		return []*database.VerificationResult{}, nil
	}
	
	// Get latest verification results for the models
	return di.db.GetLatestVerificationResults(modelIDs)
}

// Helper methods

func (di *DatabaseIntegration) getScoreCategory(score float64) string {
	switch {
	case score >= 8.5:
		return "fully_coding_capable"
	case score >= 7.0:
		return "coding_with_tools"
	case score >= 5.0:
		return "chat_with_tooling"
	default:
		return "chat_only"
	}
}

func (di *DatabaseIntegration) createScoreEvidence(score interface{}) string {
	var overallScore float64
	var components ScoreComponents
	var scoreSuffix string
	
	switch s := score.(type) {
	case *ModelScore:
		overallScore = s.Score
		components = s.Components
		scoreSuffix = s.ScoreSuffix
	case *ComprehensiveScore:
		overallScore = s.OverallScore
		components = s.Components
		scoreSuffix = s.ScoreSuffix
	default:
		overallScore = 0.0
		components = ScoreComponents{}
		scoreSuffix = ""
	}
	
	evidence := map[string]interface{}{
		"overall_score": overallScore,
		"components": map[string]float64{
			"speed_score":      components.SpeedScore,
			"efficiency_score": components.EfficiencyScore,
			"cost_score":       components.CostScore,
			"capability_score": components.CapabilityScore,
			"recency_score":    components.RecencyScore,
		},
		"score_suffix": scoreSuffix,
		"calculated_at": time.Now().Format(time.RFC3339),
	}
	
	jsonBytes, _ := json.Marshal(evidence)
	return string(jsonBytes)
}

// Helper functions

func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}