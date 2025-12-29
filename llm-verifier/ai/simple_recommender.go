package ai

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// SimpleRecommender provides intelligent model recommendations
type SimpleRecommender struct {
	models map[string]ModelProfile
}

// ModelProfile contains information about a model's capabilities
type ModelProfile struct {
	ID               string
	Provider         string
	TaskScores       map[string]float64 // task -> score (0-1)
	ComplexityScores map[string]float64 // complexity -> score (0-1)
	CostPerToken     float64
	ResponseTime     time.Duration
	Features         map[string]bool
	Reliability      float64
}

// RecRequest represents a recommendation request
type RecRequest struct {
	TaskType         string
	Complexity       string
	MaxCost          float64
	RequiredFeatures []string
}

// Recommendation represents a model recommendation
type Recommendation struct {
	ModelID   string
	Provider  string
	Score     float64
	Cost      float64
	Time      time.Duration
	Reasoning string
}

// NewSimpleRecommender creates a new simple recommender
func NewSimpleRecommender() *SimpleRecommender {
	return &SimpleRecommender{
		models: initializeModelProfiles(),
	}
}

// Recommend provides model recommendations based on requirements
func (sr *SimpleRecommender) Recommend(req RecRequest) []Recommendation {
	var recommendations []Recommendation

	for _, model := range sr.models {
		score := sr.calculateScore(model, req)
		if score > 0 {
			recommendations = append(recommendations, Recommendation{
				ModelID:   model.ID,
				Provider:  model.Provider,
				Score:     score,
				Cost:      sr.estimateCost(model, req),
				Time:      sr.estimateTime(model, req),
				Reasoning: sr.generateReasoning(model, req, score),
			})
		}
	}

	// Sort by score descending
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	// Return top 3 recommendations
	if len(recommendations) > 3 {
		return recommendations[:3]
	}
	return recommendations
}

// calculateScore calculates how well a model matches the request
func (sr *SimpleRecommender) calculateScore(model ModelProfile, req RecRequest) float64 {
	score := 0.0

	// Task suitability (40% weight)
	taskScore, exists := model.TaskScores[req.TaskType]
	if !exists {
		taskScore = 0.3 // Default neutral score
	}
	score += taskScore * 0.4

	// Complexity match (30% weight)
	complexityScore, exists := model.ComplexityScores[req.Complexity]
	if !exists {
		complexityScore = 0.5
	}
	score += complexityScore * 0.3

	// Cost efficiency (15% weight) - prefer reasonable cost models
	if req.MaxCost > 0 {
		estimatedCost := sr.estimateCost(model, req)
		if estimatedCost <= req.MaxCost {
			// Prefer models that cost between 10-50% of budget
			optimalCost := req.MaxCost * 0.3
			costDistance := math.Abs(estimatedCost-optimalCost) / optimalCost
			costEfficiency := 1.0 - math.Min(costDistance, 1.0)
			score += costEfficiency * 0.15
		} else {
			return 0 // Over budget, not recommended
		}
	} else {
		// For no budget constraint, slightly prefer reasonably priced models
		if model.CostPerToken > 0.0001 && model.CostPerToken < 0.01 {
			score += 0.8 * 0.15
		} else {
			score += 0.5 * 0.15
		}
	}

	// Feature requirements (10% weight)
	featureScore := sr.checkFeatures(model, req.RequiredFeatures)
	score += featureScore * 0.1

	return score
}

// checkFeatures checks if model has required features
func (sr *SimpleRecommender) checkFeatures(model ModelProfile, required []string) float64 {
	if len(required) == 0 {
		return 1.0
	}

	matched := 0
	for _, feature := range required {
		if model.Features[feature] {
			matched++
		}
	}

	return float64(matched) / float64(len(required))
}

// estimateCost provides cost estimation
func (sr *SimpleRecommender) estimateCost(model ModelProfile, req RecRequest) float64 {
	// Estimate tokens based on task complexity
	var tokenEstimate float64
	switch req.Complexity {
	case "simple":
		tokenEstimate = 500
	case "medium":
		tokenEstimate = 1500
	case "complex":
		tokenEstimate = 3000
	default:
		tokenEstimate = 1000
	}

	// Adjust for task type
	switch req.TaskType {
	case "coding":
		tokenEstimate *= 1.5
	case "research":
		tokenEstimate *= 2.0
	}

	return model.CostPerToken * tokenEstimate
}

// estimateTime provides time estimation
func (sr *SimpleRecommender) estimateTime(model ModelProfile, req RecRequest) time.Duration {
	baseTime := model.ResponseTime

	// Adjust for complexity
	switch req.Complexity {
	case "simple":
		baseTime = baseTime / 2
	case "complex":
		baseTime = baseTime * 2
	}

	return baseTime
}

// generateReasoning creates reasoning for the recommendation
func (sr *SimpleRecommender) generateReasoning(model ModelProfile, req RecRequest, score float64) string {
	reasons := []string{}

	// Task suitability
	if taskScore, exists := model.TaskScores[req.TaskType]; exists && taskScore > 0.7 {
		reasons = append(reasons, fmt.Sprintf("Excellent for %s tasks", req.TaskType))
	}

	// Cost efficiency
	if req.MaxCost > 0 {
		estimatedCost := sr.estimateCost(model, req)
		if estimatedCost <= req.MaxCost*0.5 {
			reasons = append(reasons, "Cost-effective choice")
		}
	}

	// Speed
	if model.ResponseTime < 3*time.Second {
		reasons = append(reasons, "Fast response time")
	}

	// Reliability
	if model.Reliability > 0.9 {
		reasons = append(reasons, "Highly reliable")
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "Good general-purpose model")
	}

	return strings.Join(reasons, ", ")
}

// initializeModelProfiles creates initial model profiles
func initializeModelProfiles() map[string]ModelProfile {
	return map[string]ModelProfile{
		"gpt-4": {
			ID:       "gpt-4",
			Provider: "openai",
			TaskScores: map[string]float64{
				"coding":   0.95,
				"writing":  0.90,
				"analysis": 0.95,
				"research": 0.90,
				"chat":     0.85,
			},
			ComplexityScores: map[string]float64{
				"simple":  0.8,
				"medium":  0.95,
				"complex": 0.90,
			},
			CostPerToken: 0.00003,
			ResponseTime: 3 * time.Second,
			Features: map[string]bool{
				"streaming":        true,
				"brotli":           true,
				"function_calling": true,
				"vision":           false,
			},
			Reliability: 0.98,
		},
		"claude-3-opus": {
			ID:       "claude-3-opus",
			Provider: "anthropic",
			TaskScores: map[string]float64{
				"coding":   0.90,
				"writing":  0.95,
				"analysis": 0.90,
				"research": 0.85,
				"chat":     0.90,
			},
			ComplexityScores: map[string]float64{
				"simple":  0.85,
				"medium":  0.90,
				"complex": 0.95,
			},
			CostPerToken: 0.000015,
			ResponseTime: 2 * time.Second,
			Features: map[string]bool{
				"streaming":        true,
				"brotli":           true,
				"toon":             true,
				"function_calling": true,
			},
			Reliability: 0.97,
		},
		"gemini-pro": {
			ID:       "gemini-pro",
			Provider: "google",
			TaskScores: map[string]float64{
				"coding":   0.85,
				"writing":  0.85,
				"analysis": 0.85,
				"research": 0.80,
				"chat":     0.80,
			},
			ComplexityScores: map[string]float64{
				"simple":  0.90,
				"medium":  0.85,
				"complex": 0.80,
			},
			CostPerToken: 0.000001,
			ResponseTime: 1 * time.Second,
			Features: map[string]bool{
				"streaming":        true,
				"brotli":           true,
				"http3":            true,
				"function_calling": false,
			},
			Reliability: 0.92,
		},
		"deepseek-chat": {
			ID:       "deepseek-chat",
			Provider: "deepseek",
			TaskScores: map[string]float64{
				"coding":   0.80,
				"writing":  0.75,
				"analysis": 0.85,
				"research": 0.80,
				"chat":     0.85,
			},
			ComplexityScores: map[string]float64{
				"simple":  0.85,
				"medium":  0.80,
				"complex": 0.75,
			},
			CostPerToken: 0.000001,
			ResponseTime: 2 * time.Second,
			Features: map[string]bool{
				"streaming":        true,
				"brotli":           true,
				"function_calling": false,
			},
			Reliability: 0.88,
		},
	}
}
