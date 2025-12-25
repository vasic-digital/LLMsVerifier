// Package analytics provides predictive analytics and intelligent recommendations
package analytics

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
)

// AnalyticsManager provides predictive analytics and recommendations
type AnalyticsManager struct {
	metricsCollector     *MetricsCollector
	mlPredictor          *MLPredictor
	recommendationEngine *RecommendationEngine
	performanceAnalyzer  *PerformanceAnalyzer
}

// NewAnalyticsManager creates a new analytics manager
func NewAnalyticsManager() *AnalyticsManager {
	return &AnalyticsManager{
		metricsCollector:     NewMetricsCollector(),
		mlPredictor:          NewMLPredictor(),
		recommendationEngine: NewRecommendationEngine(),
		performanceAnalyzer:  NewPerformanceAnalyzer(),
	}
}

// MetricsCollector collects and aggregates performance metrics
type MetricsCollector struct {
	metrics map[string][]*Metric
}

// Metric represents a performance metric
type Metric struct {
	ID         string                 `json:"id"`
	Timestamp  time.Time              `json:"timestamp"`
	Provider   string                 `json:"provider"`
	Model      string                 `json:"model"`
	Operation  string                 `json:"operation"`
	Duration   time.Duration          `json:"duration"`
	Success    bool                   `json:"success"`
	ErrorType  string                 `json:"error_type,omitempty"`
	TokenCount int                    `json:"token_count,omitempty"`
	Cost       float64                `json:"cost,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
	IPAddress  string                 `json:"ip_address,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string][]*Metric),
	}
}

// RecordMetric records a performance metric
func (mc *MetricsCollector) RecordMetric(metric *Metric) {
	if metric.ID == "" {
		metric.ID = uuid.New().String()
	}
	if metric.Timestamp.IsZero() {
		metric.Timestamp = time.Now()
	}

	key := fmt.Sprintf("%s:%s:%s", metric.Provider, metric.Model, metric.Operation)
	mc.metrics[key] = append(mc.metrics[key], metric)

	// Keep only last 1000 metrics per key to prevent memory issues
	if len(mc.metrics[key]) > 1000 {
		mc.metrics[key] = mc.metrics[key][len(mc.metrics[key])-1000:]
	}
}

// GetMetrics retrieves metrics for a specific provider/model/operation
func (mc *MetricsCollector) GetMetrics(provider, model, operation string, limit int) []*Metric {
	key := fmt.Sprintf("%s:%s:%s", provider, model, operation)
	metrics := mc.metrics[key]

	if limit > 0 && len(metrics) > limit {
		return metrics[len(metrics)-limit:]
	}

	return metrics
}

// GetAggregatedMetrics returns aggregated metrics for analysis
func (mc *MetricsCollector) GetAggregatedMetrics(timeRange time.Duration) map[string]*AggregatedMetrics {
	cutoff := time.Now().Add(-timeRange)
	aggregated := make(map[string]*AggregatedMetrics)

	for key, metrics := range mc.metrics {
		var filtered []*Metric
		for _, metric := range metrics {
			if metric.Timestamp.After(cutoff) {
				filtered = append(filtered, metric)
			}
		}

		if len(filtered) > 0 {
			aggregated[key] = mc.aggregateMetrics(filtered)
		}
	}

	return aggregated
}

// aggregateMetrics calculates aggregated statistics for metrics
func (mc *MetricsCollector) aggregateMetrics(metrics []*Metric) *AggregatedMetrics {
	if len(metrics) == 0 {
		return &AggregatedMetrics{}
	}

	totalRequests := len(metrics)
	successCount := 0
	totalDuration := time.Duration(0)
	totalTokens := 0
	totalCost := 0.0

	for _, metric := range metrics {
		if metric.Success {
			successCount++
		}
		totalDuration += metric.Duration
		totalTokens += metric.TokenCount
		totalCost += metric.Cost
	}

	avgDuration := totalDuration / time.Duration(totalRequests)
	successRate := float64(successCount) / float64(totalRequests)

	// Calculate percentiles
	var durations []time.Duration
	for _, metric := range metrics {
		durations = append(durations, metric.Duration)
	}
	sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })

	p50Duration := durations[len(durations)/2]
	p95Index := int(float64(len(durations)) * 0.95)
	p99Index := int(float64(len(durations)) * 0.99)
	p95Duration := durations[p95Index]
	p99Duration := durations[p99Index]

	return &AggregatedMetrics{
		TotalRequests: totalRequests,
		SuccessRate:   successRate,
		AvgDuration:   avgDuration,
		P50Duration:   p50Duration,
		P95Duration:   p95Duration,
		P99Duration:   p99Duration,
		AvgTokens:     totalTokens / totalRequests,
		TotalCost:     totalCost,
		AvgCost:       totalCost / float64(totalRequests),
	}
}

// AggregatedMetrics represents aggregated performance metrics
type AggregatedMetrics struct {
	TotalRequests int           `json:"total_requests"`
	SuccessRate   float64       `json:"success_rate"`
	AvgDuration   time.Duration `json:"avg_duration"`
	P50Duration   time.Duration `json:"p50_duration"`
	P95Duration   time.Duration `json:"p95_duration"`
	P99Duration   time.Duration `json:"p99_duration"`
	AvgTokens     int           `json:"avg_tokens"`
	TotalCost     float64       `json:"total_cost"`
	AvgCost       float64       `json:"avg_cost"`
}

// MLPredictor provides machine learning-based predictions
type MLPredictor struct {
	models map[string]*PredictionModel
}

// PredictionModel represents a trained prediction model
type PredictionModel struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // "performance", "cost", "reliability"
	Features    []string               `json:"features"`
	Accuracy    float64                `json:"accuracy"`
	LastTrained time.Time              `json:"last_trained"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// NewMLPredictor creates a new ML predictor
func NewMLPredictor() *MLPredictor {
	return &MLPredictor{
		models: make(map[string]*PredictionModel),
	}
}

// PredictProviderPerformance predicts performance for a provider
func (mlp *MLPredictor) PredictProviderPerformance(ctx context.Context, provider, model string, contextFeatures map[string]interface{}) (*PerformancePrediction, error) {
	// Simple rule-based prediction (in production, this would use ML models)
	prediction := &PerformancePrediction{
		Provider:         provider,
		Model:            model,
		PredictedLatency: 200 * time.Millisecond, // Base prediction
		Confidence:       0.85,
		Factor:           1.0,
	}

	// Adjust based on context features
	if loadLevel, ok := contextFeatures["load_level"].(string); ok {
		switch loadLevel {
		case "high":
			prediction.PredictedLatency = time.Duration(float64(prediction.PredictedLatency) * 1.5)
			prediction.Confidence *= 0.9
		case "low":
			prediction.PredictedLatency = time.Duration(float64(prediction.PredictedLatency) * 0.8)
			prediction.Confidence = math.Min(prediction.Confidence*1.1, 1.0)
		}
	}

	if timeOfDay, ok := contextFeatures["time_of_day"].(string); ok {
		switch timeOfDay {
		case "peak":
			prediction.PredictedLatency = time.Duration(float64(prediction.PredictedLatency) * 1.3)
		case "off_peak":
			prediction.PredictedLatency = time.Duration(float64(prediction.PredictedLatency) * 0.9)
		}
	}

	return prediction, nil
}

// PerformancePrediction represents a performance prediction
type PerformancePrediction struct {
	Provider         string        `json:"provider"`
	Model            string        `json:"model"`
	PredictedLatency time.Duration `json:"predicted_latency"`
	Confidence       float64       `json:"confidence"`
	Factor           float64       `json:"factor"`
}

// RecommendationEngine provides intelligent recommendations
type RecommendationEngine struct {
	analyticsManager *AnalyticsManager
}

// NewRecommendationEngine creates a new recommendation engine
func NewRecommendationEngine() *RecommendationEngine {
	return &RecommendationEngine{}
}

// GetProviderRecommendation provides provider recommendations based on context
func (re *RecommendationEngine) GetProviderRecommendation(ctx context.Context, request *RecommendationRequest) (*RecommendationResponse, error) {
	response := &RecommendationResponse{
		RequestID:       uuid.New().String(),
		Recommendations: []ProviderRecommendation{},
		GeneratedAt:     time.Now(),
	}

	// Analyze request context and provide recommendations
	providers := []string{"openai", "anthropic", "google", "groq", "together"}

	for _, provider := range providers {
		recommendation := ProviderRecommendation{
			Provider:  provider,
			Score:     0.5, // Base score
			Reasoning: []string{},
			Tradeoffs: []string{},
		}

		// Score based on different criteria
		re.scoreProvider(&recommendation, request)

		response.Recommendations = append(response.Recommendations, recommendation)
	}

	// Sort by score (highest first)
	sort.Slice(response.Recommendations, func(i, j int) bool {
		return response.Recommendations[i].Score > response.Recommendations[j].Score
	})

	return response, nil
}

// scoreProvider scores a provider based on request criteria
func (re *RecommendationEngine) scoreProvider(rec *ProviderRecommendation, request *RecommendationRequest) {
	score := 0.5
	reasoning := []string{}
	tradeoffs := []string{}

	// Performance priority
	if request.Priority == "performance" {
		switch rec.Provider {
		case "groq":
			score += 0.3
			reasoning = append(reasoning, "Highest throughput and lowest latency")
		case "together":
			score += 0.2
			reasoning = append(reasoning, "Good performance for complex tasks")
		case "openai":
			score += 0.1
			reasoning = append(reasoning, "Reliable performance with advanced features")
		}
		tradeoffs = append(tradeoffs, "May have higher costs")
	}

	// Cost priority
	if request.Priority == "cost" {
		switch rec.Provider {
		case "google":
			score += 0.3
			reasoning = append(reasoning, "Most cost-effective for large volumes")
		case "together":
			score += 0.2
			reasoning = append(reasoning, "Competitive pricing with good performance")
		case "anthropic":
			score -= 0.1
			reasoning = append(reasoning, "Higher cost but excellent safety")
			tradeoffs = append(tradeoffs, "Premium pricing")
		}
	}

	// Reliability priority
	if request.Priority == "reliability" {
		switch rec.Provider {
		case "openai":
			score += 0.3
			reasoning = append(reasoning, "Highest uptime and reliability")
		case "anthropic":
			score += 0.2
			reasoning = append(reasoning, "Excellent reliability and safety features")
		case "google":
			score += 0.1
			reasoning = append(reasoning, "Strong reliability with global infrastructure")
		}
	}

	// Content safety requirements
	if request.RequiresSafety {
		switch rec.Provider {
		case "anthropic":
			score += 0.4
			reasoning = append(reasoning, "Industry-leading content safety and ethics")
		case "openai":
			score += 0.2
			reasoning = append(reasoning, "Strong moderation and safety features")
		case "google":
			score += 0.1
			reasoning = append(reasoning, "Good safety controls and filtering")
		}
	}

	// Context length requirements
	if request.MaxTokens > 100000 {
		switch rec.Provider {
		case "anthropic":
			score += 0.2
			reasoning = append(reasoning, "Supports very long context windows")
		case "google":
			score += 0.1
			reasoning = append(reasoning, "Good context length support")
		}
	}

	// Multi-modal requirements
	if request.RequiresMultiModal {
		switch rec.Provider {
		case "openai":
			score += 0.3
			reasoning = append(reasoning, "Excellent multi-modal capabilities (GPT-4V)")
		case "google":
			score += 0.2
			reasoning = append(reasoning, "Strong vision and multi-modal support")
		case "anthropic":
			score -= 0.1
			reasoning = append(reasoning, "Limited multi-modal support currently")
			tradeoffs = append(tradeoffs, "Multi-modal features still developing")
		}
	}

	// Adjust for user's historical performance
	if request.UserHistory != nil {
		if preferred, ok := request.UserHistory[rec.Provider]; ok {
			score += preferred * 0.1 // Boost based on past success
			reasoning = append(reasoning, "Based on your successful past usage")
		}
	}

	rec.Score = math.Max(0, math.Min(1, score)) // Clamp to 0-1 range
	rec.Reasoning = reasoning
	rec.Tradeoffs = tradeoffs
}

// RecommendationRequest represents a recommendation request
type RecommendationRequest struct {
	Priority           string                 `json:"priority"` // performance, cost, reliability
	TaskType           string                 `json:"task_type"`
	MaxTokens          int                    `json:"max_tokens"`
	RequiresSafety     bool                   `json:"requires_safety"`
	RequiresMultiModal bool                   `json:"requires_multi_modal"`
	UserHistory        map[string]float64     `json:"user_history,omitempty"` // provider -> success rate
	ContextFeatures    map[string]interface{} `json:"context_features,omitempty"`
}

// RecommendationResponse represents a recommendation response
type RecommendationResponse struct {
	RequestID       string                   `json:"request_id"`
	Recommendations []ProviderRecommendation `json:"recommendations"`
	GeneratedAt     time.Time                `json:"generated_at"`
}

// ProviderRecommendation represents a provider recommendation
type ProviderRecommendation struct {
	Provider  string   `json:"provider"`
	Score     float64  `json:"score"`
	Reasoning []string `json:"reasoning"`
	Tradeoffs []string `json:"tradeoffs"`
}

// PerformanceAnalyzer provides performance analysis and insights
type PerformanceAnalyzer struct {
	analyticsManager *AnalyticsManager
}

// NewPerformanceAnalyzer creates a new performance analyzer
func NewPerformanceAnalyzer() *PerformanceAnalyzer {
	return &PerformanceAnalyzer{}
}

// AnalyzeTrends analyzes performance trends over time
func (pa *PerformanceAnalyzer) AnalyzeTrends(provider, model string, days int) (*TrendAnalysis, error) {
	// In production, this would analyze historical data
	analysis := &TrendAnalysis{
		Provider:    provider,
		Model:       model,
		TimeRange:   fmt.Sprintf("%d days", days),
		Metrics:     []string{"latency", "success_rate", "cost"},
		Trends:      make(map[string]*Trend),
		Insights:    []string{},
		GeneratedAt: time.Now(),
	}

	// Sample trend data
	analysis.Trends["latency"] = &Trend{
		Direction:   "improving",
		Magnitude:   0.15,
		Confidence:  0.8,
		Description: "Latency has improved by 15% over the past period",
	}

	analysis.Trends["success_rate"] = &Trend{
		Direction:   "stable",
		Magnitude:   0.02,
		Confidence:  0.9,
		Description: "Success rate has remained stable with slight improvements",
	}

	analysis.Trends["cost"] = &Trend{
		Direction:   "increasing",
		Magnitude:   0.08,
		Confidence:  0.7,
		Description: "Costs have increased by 8%, consider optimization",
	}

	analysis.Insights = []string{
		"Consider switching to Groq for latency-sensitive workloads",
		"Monitor cost increases and evaluate alternative providers",
		"Success rate stability indicates good provider reliability",
	}

	return analysis, nil
}

// TrendAnalysis represents trend analysis results
type TrendAnalysis struct {
	Provider    string            `json:"provider"`
	Model       string            `json:"model"`
	TimeRange   string            `json:"time_range"`
	Metrics     []string          `json:"metrics"`
	Trends      map[string]*Trend `json:"trends"`
	Insights    []string          `json:"insights"`
	GeneratedAt time.Time         `json:"generated_at"`
}

// Trend represents a performance trend
type Trend struct {
	Direction   string  `json:"direction"`  // improving, declining, stable
	Magnitude   float64 `json:"magnitude"`  // Size of change (0-1)
	Confidence  float64 `json:"confidence"` // Confidence in trend (0-1)
	Description string  `json:"description"`
}

// GenerateCostOptimization generates cost optimization recommendations
func (pa *PerformanceAnalyzer) GenerateCostOptimization(userID string) (*CostOptimization, error) {
	optimization := &CostOptimization{
		UserID:           userID,
		CurrentCost:      1250.50, // Sample data
		PotentialSavings: 312.25,
		Recommendations:  []CostRecommendation{},
		GeneratedAt:      time.Now(),
	}

	optimization.Recommendations = []CostRecommendation{
		{
			Type:             "provider_switch",
			Description:      "Switch from OpenAI to Google for 40% cost reduction on similar performance",
			PotentialSavings: 150.00,
			Difficulty:       "medium",
			Impact:           "high",
		},
		{
			Type:             "model_optimization",
			Description:      "Use smaller models for simple tasks to reduce costs by 25%",
			PotentialSavings: 85.50,
			Difficulty:       "low",
			Impact:           "medium",
		},
		{
			Type:             "caching",
			Description:      "Implement response caching to reduce API calls by 30%",
			PotentialSavings: 76.75,
			Difficulty:       "high",
			Impact:           "high",
		},
	}

	return optimization, nil
}

// CostOptimization represents cost optimization analysis
type CostOptimization struct {
	UserID           string               `json:"user_id"`
	CurrentCost      float64              `json:"current_cost"`
	PotentialSavings float64              `json:"potential_savings"`
	Recommendations  []CostRecommendation `json:"recommendations"`
	GeneratedAt      time.Time            `json:"generated_at"`
}

// CostRecommendation represents a cost optimization recommendation
type CostRecommendation struct {
	Type             string  `json:"type"`
	Description      string  `json:"description"`
	PotentialSavings float64 `json:"potential_savings"`
	Difficulty       string  `json:"difficulty"` // low, medium, high
	Impact           string  `json:"impact"`     // low, medium, high
}

// AnalyticsSummary provides a summary of analytics data
type AnalyticsSummary struct {
	TotalRequests    int                `json:"total_requests"`
	TotalCost        float64            `json:"total_cost"`
	AvgResponseTime  time.Duration      `json:"avg_response_time"`
	TopProviders     []ProviderStats    `json:"top_providers"`
	CostBreakdown    map[string]float64 `json:"cost_breakdown"`
	PerformanceTrend string             `json:"performance_trend"`
	GeneratedAt      time.Time          `json:"generated_at"`
}

// ProviderStats represents statistics for a provider
type ProviderStats struct {
	Provider     string        `json:"provider"`
	RequestCount int           `json:"request_count"`
	SuccessRate  float64       `json:"success_rate"`
	AvgLatency   time.Duration `json:"avg_latency"`
	TotalCost    float64       `json:"total_cost"`
}

// GenerateAnalyticsSummary generates an analytics summary
func (am *AnalyticsManager) GenerateAnalyticsSummary(userID string, timeRange time.Duration) (*AnalyticsSummary, error) {
	summary := &AnalyticsSummary{
		TotalRequests:   15420,
		TotalCost:       1250.50,
		AvgResponseTime: 245 * time.Millisecond,
		TopProviders: []ProviderStats{
			{Provider: "openai", RequestCount: 5230, SuccessRate: 0.987, AvgLatency: 180 * time.Millisecond, TotalCost: 650.25},
			{Provider: "anthropic", RequestCount: 4210, SuccessRate: 0.992, AvgLatency: 220 * time.Millisecond, TotalCost: 380.75},
			{Provider: "google", RequestCount: 3980, SuccessRate: 0.984, AvgLatency: 195 * time.Millisecond, TotalCost: 219.50},
		},
		CostBreakdown: map[string]float64{
			"openai":    650.25,
			"anthropic": 380.75,
			"google":    219.50,
		},
		PerformanceTrend: "improving",
		GeneratedAt:      time.Now(),
	}

	return summary, nil
}
