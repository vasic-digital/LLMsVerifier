package scoring

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"llm-verifier/database"
	"llm-verifier/logging"
)

// ScoringEngine handles the calculation of comprehensive LLM scores
type ScoringEngine struct {
	db                *database.Database
	modelsDevClient   *ModelsDevClient
	logger            *logging.Logger
	performanceCache  *PerformanceCache
	benchmarkData     *BenchmarkData
}

// PerformanceCache stores recent performance metrics
type PerformanceCache struct {
	responseTimes map[string][]time.Duration
	throughput    map[string]float64
	errorRates    map[string]float64
}

// BenchmarkData contains benchmarking information
type BenchmarkData struct {
	modelSizes      map[string]int64    // Parameter counts
	contextWindows  map[string]int      // Context window sizes
	architectures   map[string]string   // Architecture types
}

// ScoreComponents represents the individual scoring components
type ScoreComponents struct {
	SpeedScore      float64 `json:"speed_score"`
	EfficiencyScore float64 `json:"efficiency_score"`
	CostScore       float64 `json:"cost_score"`
	CapabilityScore float64 `json:"capability_score"`
	RecencyScore    float64 `json:"recency_score"`
}

// ComprehensiveScore contains the complete scoring information
type ComprehensiveScore struct {
	ModelID         string          `json:"model_id"`
	ModelName       string          `json:"model_name"`
	OverallScore    float64         `json:"overall_score"`
	Components      ScoreComponents `json:"components"`
	LastCalculated  time.Time       `json:"last_calculated"`
	CalculationHash string          `json:"calculation_hash"`
	ScoreSuffix     string          `json:"score_suffix"`
}

// ScoringConfig holds configuration for the scoring engine
type ScoringConfig struct {
	Weights struct {
		Speed      float64 `json:"speed"`
		Efficiency float64 `json:"efficiency"`
		Cost       float64 `json:"cost"`
		Capability float64 `json:"capability"`
		Recency    float64 `json:"recency"`
	} `json:"weights"`
	Normalization struct {
		MinScore float64 `json:"min_score"`
		MaxScore float64 `json:"max_score"`
	} `json:"normalization"`
	CacheDuration time.Duration `json:"cache_duration"`
}

// DefaultScoringConfig returns default scoring configuration
func DefaultScoringConfig() ScoringConfig {
	config := ScoringConfig{
		Normalization: struct {
			MinScore float64 `json:"min_score"`
			MaxScore float64 `json:"max_score"`
		}{
			MinScore: 0.0,
			MaxScore: 10.0,
		},
		CacheDuration: 1 * time.Hour,
	}
	
	// Set default weights
	config.Weights.Speed = 0.25
	config.Weights.Efficiency = 0.20
	config.Weights.Cost = 0.25
	config.Weights.Capability = 0.20
	config.Weights.Recency = 0.10
	
	return config
}

// NewScoringEngine creates a new scoring engine instance
func NewScoringEngine(db *database.Database, modelsDevClient *ModelsDevClient, logger *logging.Logger) *ScoringEngine {
	return &ScoringEngine{
		db:              db,
		modelsDevClient: modelsDevClient,
		logger:          logger,
		performanceCache: &PerformanceCache{
			responseTimes: make(map[string][]time.Duration),
			throughput:    make(map[string]float64),
			errorRates:    make(map[string]float64),
		},
		benchmarkData: &BenchmarkData{
			modelSizes:     make(map[string]int64),
			contextWindows: make(map[string]int),
			architectures:  make(map[string]string),
		},
	}
}

// CalculateComprehensiveScore calculates a comprehensive score for a model
func (se *ScoringEngine) CalculateComprehensiveScore(ctx context.Context, modelID string, config ScoringConfig) (*ComprehensiveScore, error) {
	// Fetch model data from database
	model, err := se.db.GetModelByModelID(modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch model data: %w", err)
	}

	// Fetch additional data from models.dev
	modelsDevModel, err := se.modelsDevClient.FetchModelByID(ctx, modelID)
	if err != nil {
		se.logger.Warn("Failed to fetch models.dev data, using database data only", "error", err)
		modelsDevModel = &ModelsDevModel{}
	}

	// Calculate individual score components
	components := ScoreComponents{}

	// 1. Calculate Speed Score (25% weight)
	components.SpeedScore = se.calculateSpeedScore(model, modelsDevModel)

	// 2. Calculate Efficiency Score (20% weight)
	components.EfficiencyScore = se.calculateEfficiencyScore(model, modelsDevModel)

	// 3. Calculate Cost Score (25% weight)
	components.CostScore = se.calculateCostScore(model, modelsDevModel)

	// 4. Calculate Capability Score (20% weight)
	components.CapabilityScore = se.calculateCapabilityScore(model, modelsDevModel)

	// 5. Calculate Recency Score (10% weight)
	components.RecencyScore = se.calculateRecencyScore(model, modelsDevModel)

	// Calculate overall score
	overallScore := se.calculateOverallScore(components, config)

	// Create comprehensive score
	score := &ComprehensiveScore{
		ModelID:         modelID,
		ModelName:       model.Name,
		OverallScore:    overallScore,
		Components:      components,
		LastCalculated:  time.Now(),
		CalculationHash: se.generateCalculationHash(components),
		ScoreSuffix:     se.generateScoreSuffix(overallScore),
	}

	// Store the score in database
	if err := se.storeScore(score); err != nil {
		se.logger.Error("Failed to store score", "error", err)
	}

	return score, nil
}

// calculateSpeedScore calculates the speed/performance score
func (se *ScoringEngine) calculateSpeedScore(model *database.Model, devModel *ModelsDevModel) float64 {
	// Get recent performance metrics
	responseTimes := se.performanceCache.responseTimes[model.ModelID]
	throughput := se.performanceCache.throughput[model.ModelID]
	errorRate := se.performanceCache.errorRates[model.ModelID]

	// If no cached data, use default values
	if len(responseTimes) == 0 {
		responseTimes = []time.Duration{500 * time.Millisecond, 600 * time.Millisecond, 550 * time.Millisecond}
	}
	if throughput == 0 {
		throughput = 2.0 // 2 requests per second default
	}
	if errorRate == 0 {
		errorRate = 0.01 // 1% error rate default
	}

	// Calculate average response time
	avgResponseTime := se.calculateAverageResponseTime(responseTimes)
	
	// Calculate P95 response time
	p95ResponseTime := se.calculateP95ResponseTime(responseTimes)

	// Normalize response times (lower is better)
	avgScore := se.normalizeResponseTime(avgResponseTime)
	p95Score := se.normalizeResponseTime(p95ResponseTime)

	// Normalize throughput (higher is better)
	throughputScore := se.normalizeThroughput(throughput)

	// Error rate penalty (lower is better)
	errorScore := se.normalizeErrorRate(errorRate)

	// Weighted combination
	speedScore := (avgScore*0.4 + p95Score*0.3 + throughputScore*0.2 + errorScore*0.1)

	return math.Max(0, math.Min(10, speedScore))
}

// calculateEfficiencyScore calculates the model efficiency score
func (se *ScoringEngine) calculateEfficiencyScore(model *database.Model, devModel *ModelsDevModel) float64 {
	// Parameter count efficiency (smaller models get higher scores for same performance)
	parameterCount := se.getParameterCount(model, devModel)
	parameterScore := se.normalizeParameterCount(parameterCount)

	// Context window efficiency
	contextWindow := se.getContextWindow(model, devModel)
	contextScore := se.normalizeContextWindow(contextWindow)

	// Architecture efficiency bonus
	architectureScore := se.getArchitectureScore(devModel)

	// Memory efficiency (context per parameter)
	memoryEfficiency := float64(contextWindow) / float64(parameterCount)
	memoryScore := se.normalizeMemoryEfficiency(memoryEfficiency)

	// Weighted combination
	efficiencyScore := (parameterScore*0.4 + contextScore*0.3 + architectureScore*0.2 + memoryScore*0.1)

	return math.Max(0, math.Min(10, efficiencyScore))
}

// calculateCostScore calculates the cost-effectiveness score
func (se *ScoringEngine) calculateCostScore(model *database.Model, devModel *ModelsDevModel) float64 {
	// Get pricing data
	inputCost := devModel.InputCostPer1M
	outputCost := devModel.OutputCostPer1M
	reasoningCost := devModel.ReasoningCostPer1M
	cacheReadCost := devModel.CacheReadCostPer1M

	// Normalize costs (lower cost = higher score)
	inputScore := se.normalizeCost(inputCost)
	outputScore := se.normalizeCost(outputCost)
	reasoningScore := se.normalizeCost(reasoningCost)
	cacheScore := se.normalizeCost(cacheReadCost)

	// Value ratio (performance per dollar)
	valueRatio := se.calculateValueRatio(model, devModel)
	valueScore := se.normalizeValueRatio(valueRatio)

	// Weighted combination
	costScore := (inputScore*0.3 + outputScore*0.3 + reasoningScore*0.2 + cacheScore*0.1 + valueScore*0.1)

	return math.Max(0, math.Min(10, costScore))
}

// calculateCapabilityScore calculates the feature capability score
func (se *ScoringEngine) calculateCapabilityScore(model *database.Model, devModel *ModelsDevModel) float64 {
	// Feature support
	featureScore := 0.0
	if devModel.ToolCall {
		featureScore += 2.0
	}
	if devModel.Reasoning {
		featureScore += 2.0
	}
	if devModel.SupportsStructuredOutput {
		featureScore += 1.5
	}
	if devModel.Multimodal {
		featureScore += 1.5
	}
	if devModel.Vision {
		featureScore += 1.0
	}
	if devModel.Audio {
		featureScore += 1.0
	}
	if devModel.Video {
		featureScore += 1.0
	}

	// Context handling capability
	contextScore := se.calculateContextCapabilityScore(devModel)

	// Reliability score from database
	reliabilityScore := model.ReliabilityScore

	// Architecture bonus
	architectureScore := se.getArchitectureCapabilityScore(devModel)

	// Weighted combination
	capabilityScore := (featureScore*0.4 + contextScore*0.3 + reliabilityScore*0.2 + architectureScore*0.1)

	return math.Max(0, math.Min(10, capabilityScore))
}

// calculateRecencyScore calculates the recency and maintenance score
func (se *ScoringEngine) calculateRecencyScore(model *database.Model, devModel *ModelsDevModel) float64 {
	// Release date recency
	releaseScore := se.calculateReleaseRecency(devModel.ReleaseDate)

	// Update frequency
	updateScore := se.calculateUpdateRecency(devModel.LastUpdated)

	// Maintenance status
	maintenanceScore := 10.0
	if model.Deprecated {
		maintenanceScore = 2.0
	}

	// Open source bonus
	openSourceBonus := 0.0
	if devModel.OpenWeights {
		openSourceBonus = 2.0
	}

	// Weighted combination
	recencyScore := (releaseScore*0.4 + updateScore*0.3 + maintenanceScore*0.2 + openSourceBonus*0.1)

	return math.Max(0, math.Min(10, recencyScore))
}

// calculateOverallScore calculates the final weighted score
func (se *ScoringEngine) calculateOverallScore(components ScoreComponents, config ScoringConfig) float64 {
	overallScore := (components.SpeedScore*config.Weights.Speed +
		components.EfficiencyScore*config.Weights.Efficiency +
		components.CostScore*config.Weights.Cost +
		components.CapabilityScore*config.Weights.Capability +
		components.RecencyScore*config.Weights.Recency)

	// Normalize to 0-10 scale
	return math.Max(0, math.Min(10, overallScore))
}

// Helper functions for score calculations

func (se *ScoringEngine) calculateAverageResponseTime(times []time.Duration) time.Duration {
	if len(times) == 0 {
		return 500 * time.Millisecond
	}
	
	sum := time.Duration(0)
	for _, t := range times {
		sum += t
	}
	return sum / time.Duration(len(times))
}

func (se *ScoringEngine) calculateP95ResponseTime(times []time.Duration) time.Duration {
	if len(times) == 0 {
		return 1000 * time.Millisecond
	}
	
	sorted := make([]time.Duration, len(times))
	copy(sorted, times)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})
	
	index := int(float64(len(sorted)) * 0.95)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func (se *ScoringEngine) normalizeResponseTime(responseTime time.Duration) float64 {
	// Normalize to 0-10 scale (lower response time = higher score)
	milliseconds := float64(responseTime.Milliseconds())
	
	// Use exponential decay for normalization
	// 100ms = 10 points, 1000ms = 7 points, 5000ms = 3 points, 10000ms = 1 point
	score := 10.0 * math.Exp(-milliseconds/2000.0)
	return math.Max(0, math.Min(10, score))
}

func (se *ScoringEngine) normalizeThroughput(throughput float64) float64 {
	// Normalize requests per second to 0-10 scale
	// 10 rps = 10 points, 1 rps = 5 points, 0.1 rps = 1 point
	score := 10.0 * (1.0 - math.Exp(-throughput/3.0))
	return math.Max(0, math.Min(10, score))
}

func (se *ScoringEngine) normalizeErrorRate(errorRate float64) float64 {
	// Convert error rate to reliability score (lower error rate = higher score)
	reliability := 1.0 - errorRate
	return math.Max(0, math.Min(10, reliability*10))
}

func (se *ScoringEngine) normalizeParameterCount(count int64) float64 {
	// Normalize parameter count (smaller models get higher efficiency scores)
	// 1B params = 10 points, 10B params = 7 points, 100B params = 3 points, 500B params = 1 point
	billions := float64(count) / 1e9
	score := 10.0 * math.Exp(-billions/20.0)
	return math.Max(0, math.Min(10, score))
}

func (se *ScoringEngine) normalizeContextWindow(window int) float64 {
	// Normalize context window size
	// 1M tokens = 10 points, 100K tokens = 7 points, 10K tokens = 3 points
	thousands := float64(window) / 1000.0
	score := 10.0 * (1.0 - math.Exp(-thousands/50.0))
	return math.Max(0, math.Min(10, score))
}

func (se *ScoringEngine) normalizeCost(cost float64) float64 {
	// Normalize cost (lower cost = higher score)
	// $0.10/1M tokens = 10 points, $1.00/1M tokens = 7 points, $10.00/1M tokens = 3 points
	score := 10.0 * math.Exp(-cost/2.0)
	return math.Max(0, math.Min(10, score))
}

func (se *ScoringEngine) generateScoreSuffix(score float64) string {
	return fmt.Sprintf("(SC:%.1f)", score)
}

func (se *ScoringEngine) generateCalculationHash(components ScoreComponents) string {
	// Generate a hash of the score components for change detection
	data := fmt.Sprintf("%.2f-%.2f-%.2f-%.2f-%.2f",
		components.SpeedScore,
		components.EfficiencyScore,
		components.CostScore,
		components.CapabilityScore,
		components.RecencyScore)
	
	// Simple hash function (could use crypto/sha256 for production)
	hash := 0
	for _, char := range data {
		hash = hash*31 + int(char)
	}
	return fmt.Sprintf("%x", hash)
}

// Additional helper functions would be implemented here...

func (se *ScoringEngine) getParameterCount(model *database.Model, devModel *ModelsDevModel) int64 {
	if devModel.AdditionalData.ParameterCount > 0 {
		return devModel.AdditionalData.ParameterCount
	}
	if model.ParameterCount != nil && *model.ParameterCount > 0 {
		return *model.ParameterCount
	}
	return 1e9 // Default 1B parameters
}

func (se *ScoringEngine) getContextWindow(model *database.Model, devModel *ModelsDevModel) int {
	if devModel.ContextLimit > 0 {
		return devModel.ContextLimit
	}
	if model.ContextWindowTokens != nil && *model.ContextWindowTokens > 0 {
		return *model.ContextWindowTokens
	}
	return 128000 // Default 128K context
}

func (se *ScoringEngine) storeScore(score *ComprehensiveScore) error {
	// Implementation to store score in database
	// This would use the existing database CRUD operations
	return nil
}