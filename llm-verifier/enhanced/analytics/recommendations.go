package analytics

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"llm-verifier/database"
)

// ModelRecommender provides AI-powered model recommendations
type ModelRecommender struct {
	db *database.Database
}

// NewModelRecommender creates a new model recommender
func NewModelRecommender(db *database.Database) *ModelRecommender {
	return &ModelRecommender{db: db}
}

// TaskRequirements represents the requirements for a task
type TaskRequirements struct {
	TaskType         string   `json:"task_type"`         // coding, writing, analysis, etc.
	Complexity       string   `json:"complexity"`        // simple, medium, complex
	SpeedRequirement string   `json:"speed_requirement"` // fast, normal, quality_first
	BudgetLimit      float64  `json:"budget_limit"`      // Maximum cost per request
	RequiredFeatures []string `json:"required_features"` // json_mode, function_calling, etc.
	ContextLength    int      `json:"context_length"`    // Required context window
}

// ModelRecommendation represents a model recommendation
type ModelRecommendation struct {
	ModelID          string   `json:"model_id"`
	Provider         string   `json:"provider"`
	Score            float64  `json:"score"`             // 0-100, higher is better
	CostEstimate     float64  `json:"cost_estimate"`     // Estimated cost per request
	PerformanceScore float64  `json:"performance_score"` // Performance rating
	ReliabilityScore float64  `json:"reliability_score"` // Reliability rating
	Reasoning        []string `json:"reasoning"`         // Why this model was recommended
}

// RecommendationResult represents the result of a recommendation query
type RecommendationResult struct {
	Requirements       TaskRequirements      `json:"requirements"`
	Recommendations    []ModelRecommendation `json:"recommendations"`
	BestChoice         ModelRecommendation   `json:"best_choice"`
	AlternativeOptions []ModelRecommendation `json:"alternative_options"`
	AnalysisTime       float64               `json:"analysis_time_ms"`
}

// RecommendModel recommends the best model for a given task
func (mr *ModelRecommender) RecommendModel(requirements TaskRequirements) (*RecommendationResult, error) {
	startTime := time.Now()

	// Get all available models
	models, err := mr.getAvailableModels()
	if err != nil {
		return nil, fmt.Errorf("failed to get available models: %w", err)
	}

	// Score each model
	var recommendations []ModelRecommendation
	for _, model := range models {
		recommendation := mr.scoreModel(model, requirements)
		if recommendation.Score > 10 { // Only include models with reasonable scores
			recommendations = append(recommendations, recommendation)
		}
	}

	// Sort by score (descending)
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	// Limit to top 5 recommendations
	if len(recommendations) > 5 {
		recommendations = recommendations[:5]
	}

	result := &RecommendationResult{
		Requirements:    requirements,
		Recommendations: recommendations,
		AnalysisTime:    float64(time.Since(startTime).Nanoseconds()) / 1e6, // Convert to milliseconds
	}

	// Set best choice and alternatives
	if len(recommendations) > 0 {
		result.BestChoice = recommendations[0]
		if len(recommendations) > 1 {
			result.AlternativeOptions = recommendations[1:]
		}
	}

	return result, nil
}

// getAvailableModels gets all available models with their metadata
func (mr *ModelRecommender) getAvailableModels() ([]ModelData, error) {
	// This is a simplified implementation - in reality, you'd query the database
	models := []ModelData{
		{
			ID:               "gpt-4",
			Name:             "GPT-4",
			Provider:         "OpenAI",
			TaskTypes:        []string{"coding", "writing", "analysis", "research"},
			PerformanceScore: 95.0,
			ReliabilityScore: 98.0,
			CostPerToken:     0.03, // $0.03 per 1K tokens
			ContextLength:    8192,
			Features:         []string{"function_calling", "json_mode", "streaming"},
			MaxTokens:        4096,
		},
		{
			ID:               "gpt-3.5-turbo",
			Name:             "GPT-3.5 Turbo",
			Provider:         "OpenAI",
			TaskTypes:        []string{"coding", "writing", "analysis", "chat"},
			PerformanceScore: 85.0,
			ReliabilityScore: 96.0,
			CostPerToken:     0.002, // $0.002 per 1K tokens
			ContextLength:    4096,
			Features:         []string{"function_calling", "json_mode", "streaming"},
			MaxTokens:        2048,
		},
		{
			ID:               "claude-3-sonnet",
			Name:             "Claude 3 Sonnet",
			Provider:         "Anthropic",
			TaskTypes:        []string{"coding", "writing", "analysis", "research", "creative"},
			PerformanceScore: 92.0,
			ReliabilityScore: 97.0,
			CostPerToken:     0.015, // $0.015 per 1K tokens
			ContextLength:    200000,
			Features:         []string{"function_calling", "json_mode", "streaming", "vision"},
			MaxTokens:        4096,
		},
		{
			ID:               "claude-3-haiku",
			Name:             "Claude 3 Haiku",
			Provider:         "Anthropic",
			TaskTypes:        []string{"coding", "writing", "chat", "analysis"},
			PerformanceScore: 82.0,
			ReliabilityScore: 95.0,
			CostPerToken:     0.00025, // $0.00025 per 1K tokens
			ContextLength:    200000,
			Features:         []string{"function_calling", "json_mode", "streaming"},
			MaxTokens:        4096,
		},
		{
			ID:               "gemini-pro",
			Name:             "Gemini Pro",
			Provider:         "Google",
			TaskTypes:        []string{"coding", "writing", "analysis", "multimodal"},
			PerformanceScore: 88.0,
			ReliabilityScore: 93.0,
			CostPerToken:     0.001, // $0.001 per 1K tokens
			ContextLength:    32768,
			Features:         []string{"function_calling", "json_mode", "streaming", "vision", "audio"},
			MaxTokens:        2048,
		},
		{
			ID:               "codellama-34b",
			Name:             "CodeLlama 34B",
			Provider:         "Meta",
			TaskTypes:        []string{"coding", "code_review", "debugging"},
			PerformanceScore: 90.0,
			ReliabilityScore: 94.0,
			CostPerToken:     0.0005, // $0.0005 per 1K tokens
			ContextLength:    16384,
			Features:         []string{"code_generation", "fill_in_middle"},
			MaxTokens:        1024,
		},
	}

	return models, nil
}

// ModelData represents model metadata for recommendation
type ModelData struct {
	ID               string
	Name             string
	Provider         string
	TaskTypes        []string
	PerformanceScore float64
	ReliabilityScore float64
	CostPerToken     float64
	ContextLength    int
	Features         []string
	MaxTokens        int
}

// scoreModel scores a model based on task requirements
func (mr *ModelRecommender) scoreModel(model ModelData, requirements TaskRequirements) ModelRecommendation {
	score := 100.0
	var reasoning []string

	// Task type matching (30% weight)
	taskMatchScore := mr.calculateTaskMatchScore(model.TaskTypes, requirements.TaskType)
	score = score*0.7 + taskMatchScore*30
	reasoning = append(reasoning, fmt.Sprintf("Task type match: %.1f/30", taskMatchScore))

	// Complexity matching (20% weight)
	complexityScore := mr.calculateComplexityScore(model.PerformanceScore, requirements.Complexity)
	score = score*0.8 + complexityScore*20
	reasoning = append(reasoning, fmt.Sprintf("Complexity match: %.1f/20", complexityScore))

	// Speed vs quality balance (15% weight)
	speedScore := mr.calculateSpeedScore(model, requirements.SpeedRequirement)
	score = score*0.85 + speedScore*15
	reasoning = append(reasoning, fmt.Sprintf("Speed/quality balance: %.1f/15", speedScore))

	// Cost efficiency (15% weight)
	costScore := mr.calculateCostScore(model, requirements)
	score = score*0.866 + costScore*15
	reasoning = append(reasoning, fmt.Sprintf("Cost efficiency: %.1f/15", costScore))

	// Feature requirements (10% weight)
	featureScore := mr.calculateFeatureScore(model.Features, requirements.RequiredFeatures)
	score = score*0.9 + featureScore*10
	reasoning = append(reasoning, fmt.Sprintf("Feature match: %.1f/10", featureScore))

	// Context length suitability (5% weight)
	contextScore := mr.calculateContextScore(model.ContextLength, requirements.ContextLength)
	score = score*0.95 + contextScore*5
	reasoning = append(reasoning, fmt.Sprintf("Context suitability: %.1f/5", contextScore))

	// Reliability bonus (5% weight)
	reliabilityBonus := model.ReliabilityScore / 20 // Convert to 0-5 scale
	score += reliabilityBonus
	reasoning = append(reasoning, fmt.Sprintf("Reliability bonus: %.1f/5", reliabilityBonus))

	// Estimate cost
	costEstimate := mr.estimateCost(model, requirements)

	return ModelRecommendation{
		ModelID:          model.ID,
		Provider:         model.Provider,
		Score:            math.Max(0, math.Min(100, score)), // Clamp to 0-100
		CostEstimate:     costEstimate,
		PerformanceScore: model.PerformanceScore,
		ReliabilityScore: model.ReliabilityScore,
		Reasoning:        reasoning,
	}
}

// calculateTaskMatchScore calculates how well the model matches the task type
func (mr *ModelRecommender) calculateTaskMatchScore(modelTasks []string, requiredTask string) float64 {
	for _, task := range modelTasks {
		if strings.EqualFold(task, requiredTask) {
			return 30.0 // Perfect match
		}
		if strings.Contains(strings.ToLower(task), strings.ToLower(requiredTask)) ||
			strings.Contains(strings.ToLower(requiredTask), strings.ToLower(task)) {
			return 20.0 // Partial match
		}
	}
	return 5.0 // No match
}

// calculateComplexityScore calculates complexity suitability
func (mr *ModelRecommender) calculateComplexityScore(performanceScore float64, complexity string) float64 {
	switch strings.ToLower(complexity) {
	case "simple":
		// For simple tasks, any model works, but prefer cost-effective ones
		return 20.0
	case "medium":
		// Medium complexity needs good performance
		if performanceScore >= 85 {
			return 20.0
		} else if performanceScore >= 75 {
			return 15.0
		}
		return 10.0
	case "complex":
		// Complex tasks need high performance
		if performanceScore >= 90 {
			return 20.0
		} else if performanceScore >= 80 {
			return 15.0
		}
		return 5.0
	default:
		return 15.0
	}
}

// calculateSpeedScore calculates speed vs quality balance
func (mr *ModelRecommender) calculateSpeedScore(model ModelData, speedReq string) float64 {
	switch strings.ToLower(speedReq) {
	case "fast":
		// Prefer faster, potentially less expensive models
		if model.CostPerToken < 0.01 {
			return 15.0
		} else if model.CostPerToken < 0.02 {
			return 12.0
		}
		return 8.0
	case "normal":
		// Balance speed and quality
		if model.PerformanceScore >= 85 && model.CostPerToken < 0.02 {
			return 15.0
		}
		return 10.0
	case "quality_first":
		// Prefer high-performance models
		if model.PerformanceScore >= 90 {
			return 15.0
		} else if model.PerformanceScore >= 85 {
			return 12.0
		}
		return 8.0
	default:
		return 12.0
	}
}

// calculateCostScore calculates cost efficiency
func (mr *ModelRecommender) calculateCostScore(model ModelData, requirements TaskRequirements) float64 {
	if requirements.BudgetLimit > 0 {
		estimatedCost := mr.estimateCost(model, requirements)
		if estimatedCost > requirements.BudgetLimit {
			return 0.0 // Over budget
		}
		// Score based on how much budget is left
		budgetUtilization := estimatedCost / requirements.BudgetLimit
		return 15.0 * (1.0 - budgetUtilization) // Higher score for lower utilization
	}

	// No budget limit, score based on absolute cost
	if model.CostPerToken < 0.001 {
		return 15.0 // Very cheap
	} else if model.CostPerToken < 0.01 {
		return 12.0 // Reasonably priced
	} else if model.CostPerToken < 0.02 {
		return 8.0 // Expensive
	}
	return 4.0 // Very expensive
}

// calculateFeatureScore calculates feature matching
func (mr *ModelRecommender) calculateFeatureScore(modelFeatures []string, requiredFeatures []string) float64 {
	if len(requiredFeatures) == 0 {
		return 10.0 // No specific requirements
	}

	matched := 0
	for _, required := range requiredFeatures {
		for _, available := range modelFeatures {
			if strings.EqualFold(available, required) {
				matched++
				break
			}
		}
	}

	matchRatio := float64(matched) / float64(len(requiredFeatures))
	return 10.0 * matchRatio
}

// calculateContextScore calculates context length suitability
func (mr *ModelRecommender) calculateContextScore(available int, required int) float64 {
	if required == 0 {
		return 5.0 // No specific requirement
	}

	if available >= required {
		// Perfect fit
		return 5.0
	} else if available >= required/2 {
		// Can handle with some limitations
		return 3.0
	}
	// Insufficient context
	return 0.0
}

// estimateCost estimates the cost for a request
func (mr *ModelRecommender) estimateCost(model ModelData, requirements TaskRequirements) float64 {
	// Estimate based on typical usage patterns
	avgInputTokens := 1000.0 // Assume 1K input tokens
	avgOutputTokens := 500.0 // Assume 500 output tokens

	// Adjust based on complexity
	switch strings.ToLower(requirements.Complexity) {
	case "simple":
		avgInputTokens = 500
		avgOutputTokens = 200
	case "complex":
		avgInputTokens = 2000
		avgOutputTokens = 1000
	}

	// Adjust for context length
	if requirements.ContextLength > 0 {
		contextMultiplier := float64(requirements.ContextLength) / 4000.0 // Normalize to 4K baseline
		avgInputTokens *= math.Min(contextMultiplier, 3.0)                // Cap at 3x
	}

	totalTokens := avgInputTokens + avgOutputTokens
	costPerThousand := model.CostPerToken

	return (totalTokens / 1000.0) * costPerThousand
}

// GetModelComparison compares multiple models for a task
func (mr *ModelRecommender) GetModelComparison(requirements TaskRequirements, modelIDs []string) (*ModelComparison, error) {
	models, err := mr.getAvailableModels()
	if err != nil {
		return nil, err
	}

	// Filter to requested models
	var selectedModels []ModelData
	for _, model := range models {
		for _, requestedID := range modelIDs {
			if model.ID == requestedID {
				selectedModels = append(selectedModels, model)
				break
			}
		}
	}

	if len(selectedModels) == 0 {
		return nil, fmt.Errorf("no requested models found")
	}

	// Generate recommendations for comparison
	comparison := &ModelComparison{
		Requirements: requirements,
		Models:       make([]ModelRecommendation, len(selectedModels)),
	}

	for i, model := range selectedModels {
		comparison.Models[i] = mr.scoreModel(model, requirements)
	}

	// Sort by score
	sort.Slice(comparison.Models, func(i, j int) bool {
		return comparison.Models[i].Score > comparison.Models[j].Score
	})

	return comparison, nil
}

// ModelComparison represents a comparison of multiple models
type ModelComparison struct {
	Requirements TaskRequirements      `json:"requirements"`
	Models       []ModelRecommendation `json:"models"`
}

// GetUsageInsights provides insights about model usage patterns
func (mr *ModelRecommender) GetUsageInsights(timeRange TimeRange) (*UsageInsights, error) {
	insights := &UsageInsights{
		TimeRange: timeRange,
	}

	// Analyze current usage patterns (simplified)
	insights.MostUsedModels = []string{"gpt-4", "claude-3-sonnet", "gpt-3.5-turbo"}
	insights.UnderUtilizedModels = []string{"codellama-34b", "gemini-pro"}
	insights.CostDrivers = []string{"gpt-4", "claude-3-sonnet"}

	// Generate insights
	insights.Insights = []string{
		"GPT-4 is heavily used for complex tasks but expensive - consider task routing",
		"CodeLlama models are underutilized for coding tasks - good cost-saving opportunity",
		"Consider implementing model selection logic based on task complexity",
		"Monitor token usage patterns to optimize context window usage",
	}

	return insights, nil
}

// UsageInsights represents usage insights and recommendations
type UsageInsights struct {
	TimeRange           TimeRange `json:"time_range"`
	MostUsedModels      []string  `json:"most_used_models"`
	UnderUtilizedModels []string  `json:"under_utilized_models"`
	CostDrivers         []string  `json:"cost_drivers"`
	Insights            []string  `json:"insights"`
}
