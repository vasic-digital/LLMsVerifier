package enhanced

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"llm-verifier/database"
)

// ModelComparisonEngine provides side-by-side model comparison capabilities
type ModelComparisonEngine struct {
	db *database.Database
}

// NewModelComparisonEngine creates a new model comparison engine
func NewModelComparisonEngine(db *database.Database) *ModelComparisonEngine {
	return &ModelComparisonEngine{db: db}
}

// ComparisonResult represents the result of comparing multiple models
type ComparisonResult struct {
	Models          []*database.Model           `json:"models"`
	Metrics         map[string]MetricComparison `json:"metrics"`
	Rankings        map[string][]ModelRanking   `json:"rankings"`
	Recommendations []string                    `json:"recommendations"`
	Summary         string                      `json:"summary"`
}

// MetricComparison represents comparison data for a specific metric
type MetricComparison struct {
	MetricName  string             `json:"metric_name"`
	Description string             `json:"description"`
	Values      map[string]float64 `json:"values"`  // model_id -> value
	Ranking     []string           `json:"ranking"` // model_ids ordered by metric
	BestValue   float64            `json:"best_value"`
	WorstValue  float64            `json:"worst_value"`
	AvgValue    float64            `json:"avg_value"`
}

// ModelRanking represents a model's ranking in a category
type ModelRanking struct {
	ModelID    string  `json:"model_id"`
	Rank       int     `json:"rank"`
	Score      float64 `json:"score"`
	Percentile float64 `json:"percentile"`
}

// CompareModels compares multiple models across various metrics
func (mce *ModelComparisonEngine) CompareModels(modelIDs []string, includeVerificationResults bool) (*ComparisonResult, error) {
	if len(modelIDs) < 2 {
		return nil, fmt.Errorf("need at least 2 models to compare")
	}

	// Load model data - for now, we'll need to get models by ID
	// In a real implementation, we'd have a method to get models by model_id string
	models := make([]*database.Model, 0, len(modelIDs))

	// Try to find models - this is a simplified version
	allModels, err := mce.db.ListModels(map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	for _, modelID := range modelIDs {
		for _, model := range allModels {
			if model.ModelID == modelID {
				models = append(models, model)
				break
			}
		}
	}

	if len(models) < 2 {
		return nil, fmt.Errorf("could not load sufficient model data for comparison")
	}

	result := &ComparisonResult{
		Models:          models,
		Metrics:         make(map[string]MetricComparison),
		Rankings:        make(map[string][]ModelRanking),
		Recommendations: []string{},
	}

	// Compare basic model attributes
	mce.compareBasicAttributes(result, models)

	// Compare performance metrics if requested
	if includeVerificationResults {
		mce.comparePerformanceMetrics(result, models)
	}

	// Generate rankings
	mce.generateRankings(result)

	// Generate recommendations
	mce.generateRecommendations(result)

	// Generate summary
	result.Summary = mce.generateSummary(result)

	return result, nil
}

// compareBasicAttributes compares basic model attributes
func (mce *ModelComparisonEngine) compareBasicAttributes(result *ComparisonResult, models []*database.Model) {
	// Context window comparison
	contextWindows := make(map[string]float64)
	for _, model := range models {
		if model.ContextWindowTokens != nil {
			contextWindows[model.ModelID] = float64(*model.ContextWindowTokens)
		}
	}
	if len(contextWindows) > 0 {
		result.Metrics["context_window"] = mce.createMetricComparison(
			"Context Window", "Maximum context length in tokens", contextWindows, true,
		)
	}

	// Parameter count comparison
	parameters := make(map[string]float64)
	for _, model := range models {
		if model.ParameterCount != nil {
			parameters[model.ModelID] = float64(*model.ParameterCount)
		}
	}
	if len(parameters) > 0 {
		result.Metrics["parameters"] = mce.createMetricComparison(
			"Parameters", "Number of model parameters", parameters, true,
		)
	}

	// Release date comparison (newer is better)
	releaseDates := make(map[string]float64)
	for _, model := range models {
		if model.ReleaseDate != nil {
			releaseDates[model.ModelID] = float64(model.ReleaseDate.Unix())
		}
	}
	if len(releaseDates) > 0 {
		result.Metrics["release_date"] = mce.createMetricComparison(
			"Release Date", "Model release date (newer is better)", releaseDates, true,
		)
	}
}

// comparePerformanceMetrics compares performance metrics from verification results
func (mce *ModelComparisonEngine) comparePerformanceMetrics(result *ComparisonResult, models []*database.Model) {
	// Get verification results for each model
	// Note: In a real implementation, we'd get the latest results per model
	allResults, err := mce.db.ListVerificationResults(map[string]interface{}{})
	if err != nil {
		return // Skip performance comparison if we can't get results
	}

	// Group by model and find the latest result for each
	latestResults := make(map[string]*database.VerificationResult)
	for _, result := range allResults {
		if existing, exists := latestResults[fmt.Sprintf("%d", result.ModelID)]; !exists ||
			(result.CompletedAt != nil && (existing.CompletedAt == nil || result.CompletedAt.After(*existing.CompletedAt))) {
			latestResults[fmt.Sprintf("%d", result.ModelID)] = result
		}
	}

	// Convert to model ID mapping
	modelResults := make(map[string]*database.VerificationResult)
	for _, model := range models {
		if result, exists := latestResults[fmt.Sprintf("%d", model.ID)]; exists {
			modelResults[model.ModelID] = result
		}
	}

	if len(latestResults) == 0 {
		return // No verification results available
	}

	// Overall score comparison
	overallScores := make(map[string]float64)
	for modelID, result := range latestResults {
		overallScores[modelID] = result.OverallScore
	}
	result.Metrics["overall_score"] = mce.createMetricComparison(
		"Overall Score", "Comprehensive performance score", overallScores, true,
	)

	// Code capability comparison
	codeScores := make(map[string]float64)
	for modelID, result := range latestResults {
		codeScores[modelID] = result.CodeCapabilityScore
	}
	result.Metrics["code_capability"] = mce.createMetricComparison(
		"Code Capability", "Ability to handle coding tasks", codeScores, true,
	)

	// Response time comparison (lower is better)
	responseTimes := make(map[string]float64)
	for modelID, result := range latestResults {
		if result.ResponsivenessScore > 0 {
			responseTimes[modelID] = result.ResponsivenessScore
		}
	}
	if len(responseTimes) > 0 {
		result.Metrics["responsiveness"] = mce.createMetricComparison(
			"Responsiveness", "Response time performance (lower is better)", responseTimes, false,
		)
	}
}

// createMetricComparison creates a metric comparison from values
func (mce *ModelComparisonEngine) createMetricComparison(name, description string, values map[string]float64, higherIsBetter bool) MetricComparison {
	if len(values) == 0 {
		return MetricComparison{MetricName: name, Description: description}
	}

	// Calculate statistics
	var sum, min, max float64
	min = math.MaxFloat64
	max = -math.MaxFloat64

	for _, value := range values {
		sum += value
		if value < min {
			min = value
		}
		if value > max {
			max = value
		}
	}

	avg := sum / float64(len(values))

	// Create ranking
	type modelScore struct {
		modelID string
		score   float64
	}

	scores := make([]modelScore, 0, len(values))
	for modelID, value := range values {
		scores = append(scores, modelScore{modelID, value})
	}

	// Sort by score (higher or lower depending on metric)
	sort.Slice(scores, func(i, j int) bool {
		if higherIsBetter {
			return scores[i].score > scores[j].score
		}
		return scores[i].score < scores[j].score
	})

	ranking := make([]string, len(scores))
	for i, score := range scores {
		ranking[i] = score.modelID
	}

	return MetricComparison{
		MetricName:  name,
		Description: description,
		Values:      values,
		Ranking:     ranking,
		BestValue:   scores[0].score,
		WorstValue:  scores[len(scores)-1].score,
		AvgValue:    avg,
	}
}

// generateRankings generates overall rankings for models
func (mce *ModelComparisonEngine) generateRankings(result *ComparisonResult) {
	// Calculate composite scores
	compositeScores := make(map[string]float64)

	for _, model := range result.Models {
		score := 0.0
		weight := 0.0

		// Weight different metrics
		if overallMetric, exists := result.Metrics["overall_score"]; exists {
			if val, hasVal := overallMetric.Values[model.ModelID]; hasVal {
				score += val * 0.4 // 40% weight for overall score
				weight += 0.4
			}
		}

		if codeMetric, exists := result.Metrics["code_capability"]; exists {
			if val, hasVal := codeMetric.Values[model.ModelID]; hasVal {
				score += val * 0.3 // 30% weight for code capability
				weight += 0.3
			}
		}

		if contextMetric, exists := result.Metrics["context_window"]; exists {
			if val, hasVal := contextMetric.Values[model.ModelID]; hasVal {
				// Normalize context window (higher is better)
				normalized := math.Min(val/32768.0, 1.0) // Cap at 32K tokens
				score += normalized * 0.2                // 20% weight for context
				weight += 0.2
			}
		}

		if responsivenessMetric, exists := result.Metrics["responsiveness"]; exists {
			if val, hasVal := responsivenessMetric.Values[model.ModelID]; hasVal {
				// Lower responsiveness score is better, so invert
				normalized := math.Max(0, 1.0-(val/100.0))
				score += normalized * 0.1 // 10% weight for responsiveness
				weight += 0.1
			}
		}

		if weight > 0 {
			compositeScores[model.ModelID] = score / weight
		}
	}

	// Create rankings
	rankings := make([]ModelRanking, 0, len(compositeScores))
	for modelID, score := range compositeScores {
		rankings = append(rankings, ModelRanking{
			ModelID:    modelID,
			Score:      score,
			Percentile: mce.calculatePercentile(score, compositeScores),
		})
	}

	// Sort by score (descending)
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].Score > rankings[j].Score
	})

	// Assign ranks
	for i, ranking := range rankings {
		ranking.Rank = i + 1
		rankings[i] = ranking
	}

	result.Rankings["composite"] = rankings
}

// calculatePercentile calculates the percentile of a score
func (mce *ModelComparisonEngine) calculatePercentile(score float64, allScores map[string]float64) float64 {
	total := len(allScores)
	if total <= 1 {
		return 100.0
	}

	better := 0
	for _, s := range allScores {
		if s > score {
			better++
		}
	}

	return (1.0 - float64(better)/float64(total)) * 100.0
}

// generateRecommendations generates recommendations based on comparison
func (mce *ModelComparisonEngine) generateRecommendations(result *ComparisonResult) {
	if rankings, exists := result.Rankings["composite"]; exists && len(rankings) > 0 {
		bestModel := rankings[0]

		result.Recommendations = append(result.Recommendations,
			fmt.Sprintf("Best overall model: %s (Score: %.2f)", bestModel.ModelID, bestModel.Score))

		// Check for specific strengths
		if contextMetric, exists := result.Metrics["context_window"]; exists {
			if len(contextMetric.Ranking) > 0 {
				bestContext := contextMetric.Ranking[0]
				if bestContext != bestModel.ModelID {
					result.Recommendations = append(result.Recommendations,
						fmt.Sprintf("For long conversations, consider %s (largest context window)", bestContext))
				}
			}
		}

		if codeMetric, exists := result.Metrics["code_capability"]; exists {
			if len(codeMetric.Ranking) > 0 {
				bestCode := codeMetric.Ranking[0]
				if bestCode != bestModel.ModelID {
					result.Recommendations = append(result.Recommendations,
						fmt.Sprintf("For coding tasks, consider %s (best code capability)", bestCode))
				}
			}
		}
	}
}

// generateSummary generates a summary of the comparison
func (mce *ModelComparisonEngine) generateSummary(result *ComparisonResult) string {
	if len(result.Models) == 0 {
		return "No models to compare"
	}

	summary := fmt.Sprintf("Comparison of %d models", len(result.Models))

	if rankings, exists := result.Rankings["composite"]; exists && len(rankings) > 0 {
		best := rankings[0]
		summary += fmt.Sprintf(". Best performer: %s with score %.2f", best.ModelID, best.Score)

		if len(rankings) > 1 {
			worst := rankings[len(rankings)-1]
			summary += fmt.Sprintf(". Performance range: %.2f points", best.Score-worst.Score)
		}
	}

	// Add key differentiators
	differentiators := []string{}

	if metric, exists := result.Metrics["context_window"]; exists && len(metric.Values) > 1 {
		maxContext := metric.BestValue
		minContext := metric.WorstValue
		if maxContext > minContext*1.5 { // Significant difference
			differentiators = append(differentiators, "context window sizes")
		}
	}

	if metric, exists := result.Metrics["parameters"]; exists && len(metric.Values) > 1 {
		maxParams := metric.BestValue
		minParams := metric.WorstValue
		if maxParams > minParams*2 { // Significant difference
			differentiators = append(differentiators, "model sizes")
		}
	}

	if len(differentiators) > 0 {
		summary += ". Key differentiators: " + strings.Join(differentiators, ", ")
	}

	return summary
}
