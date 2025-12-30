package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== AnalyticsManager Tests ====================

func TestNewAnalyticsManager(t *testing.T) {
	manager := NewAnalyticsManager()
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.metricsCollector)
	assert.NotNil(t, manager.mlPredictor)
	assert.NotNil(t, manager.recommendationEngine)
	assert.NotNil(t, manager.performanceAnalyzer)
}

func TestAnalyticsManager_GenerateAnalyticsSummary(t *testing.T) {
	manager := NewAnalyticsManager()

	summary, err := manager.GenerateAnalyticsSummary("user123", 24*time.Hour)
	require.NoError(t, err)
	assert.NotNil(t, summary)
	assert.Greater(t, summary.TotalRequests, 0)
	assert.Greater(t, summary.TotalCost, 0.0)
	assert.NotEmpty(t, summary.TopProviders)
	assert.NotEmpty(t, summary.CostBreakdown)
	assert.NotEmpty(t, summary.PerformanceTrend)
	assert.False(t, summary.GeneratedAt.IsZero())
}

// ==================== MetricsCollector Tests ====================

func TestNewMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector()
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.metrics)
}

func TestMetricsCollector_RecordMetric(t *testing.T) {
	collector := NewMetricsCollector()

	metric := &Metric{
		Provider:   "openai",
		Model:      "gpt-4",
		Operation:  "completion",
		Duration:   100 * time.Millisecond,
		Success:    true,
		TokenCount: 500,
		Cost:       0.05,
	}

	collector.RecordMetric(metric)

	// ID should be assigned
	assert.NotEmpty(t, metric.ID)
	// Timestamp should be assigned
	assert.False(t, metric.Timestamp.IsZero())

	// Metric should be recorded
	metrics := collector.GetMetrics("openai", "gpt-4", "completion", 10)
	assert.Len(t, metrics, 1)
	assert.Equal(t, "openai", metrics[0].Provider)
}

func TestMetricsCollector_RecordMetric_WithExistingID(t *testing.T) {
	collector := NewMetricsCollector()

	metric := &Metric{
		ID:        "custom-id-123",
		Timestamp: time.Now().Add(-time.Hour),
		Provider:  "anthropic",
		Model:     "claude-3",
		Operation: "chat",
		Duration:  200 * time.Millisecond,
		Success:   true,
	}

	collector.RecordMetric(metric)

	// ID should remain unchanged
	assert.Equal(t, "custom-id-123", metric.ID)
}

func TestMetricsCollector_RecordMetric_TrimOldMetrics(t *testing.T) {
	collector := NewMetricsCollector()

	// Record more than 1000 metrics
	for i := 0; i < 1005; i++ {
		metric := &Metric{
			Provider:  "openai",
			Model:     "gpt-4",
			Operation: "completion",
			Duration:  100 * time.Millisecond,
			Success:   true,
		}
		collector.RecordMetric(metric)
	}

	// Should be trimmed to 1000
	metrics := collector.GetMetrics("openai", "gpt-4", "completion", 0)
	assert.Len(t, metrics, 1000)
}

func TestMetricsCollector_GetMetrics(t *testing.T) {
	collector := NewMetricsCollector()

	// Record multiple metrics
	for i := 0; i < 10; i++ {
		collector.RecordMetric(&Metric{
			Provider:  "google",
			Model:     "gemini",
			Operation: "generate",
			Duration:  time.Duration(i*100) * time.Millisecond,
			Success:   true,
		})
	}

	// Get all metrics
	allMetrics := collector.GetMetrics("google", "gemini", "generate", 0)
	assert.Len(t, allMetrics, 10)

	// Get limited metrics
	limitedMetrics := collector.GetMetrics("google", "gemini", "generate", 5)
	assert.Len(t, limitedMetrics, 5)
}

func TestMetricsCollector_GetMetrics_NoMatches(t *testing.T) {
	collector := NewMetricsCollector()

	metrics := collector.GetMetrics("nonexistent", "model", "op", 10)
	assert.Empty(t, metrics)
}

func TestMetricsCollector_GetAggregatedMetrics(t *testing.T) {
	collector := NewMetricsCollector()

	// Record metrics
	for i := 0; i < 10; i++ {
		collector.RecordMetric(&Metric{
			Provider:   "openai",
			Model:      "gpt-4",
			Operation:  "completion",
			Duration:   time.Duration(100+i*10) * time.Millisecond,
			Success:    i != 5, // 1 failure
			TokenCount: 500 + i*50,
			Cost:       0.05 + float64(i)*0.01,
		})
	}

	aggregated := collector.GetAggregatedMetrics(time.Hour)
	require.NotEmpty(t, aggregated)

	stats := aggregated["openai:gpt-4:completion"]
	require.NotNil(t, stats)
	assert.Equal(t, 10, stats.TotalRequests)
	assert.Equal(t, 0.9, stats.SuccessRate) // 9/10
	assert.Greater(t, stats.AvgDuration, time.Duration(0))
	assert.Greater(t, stats.P50Duration, time.Duration(0))
	assert.Greater(t, stats.P95Duration, time.Duration(0))
	assert.Greater(t, stats.P99Duration, time.Duration(0))
	assert.Greater(t, stats.AvgTokens, 0)
	assert.Greater(t, stats.TotalCost, 0.0)
	assert.Greater(t, stats.AvgCost, 0.0)
}

func TestMetricsCollector_GetAggregatedMetrics_FilterByTime(t *testing.T) {
	collector := NewMetricsCollector()

	// Record old metric (outside time range)
	oldMetric := &Metric{
		Provider:  "openai",
		Model:     "gpt-4",
		Operation: "completion",
		Duration:  100 * time.Millisecond,
		Success:   true,
		Timestamp: time.Now().Add(-2 * time.Hour),
	}
	collector.metrics["openai:gpt-4:completion"] = append(collector.metrics["openai:gpt-4:completion"], oldMetric)

	// Record new metric
	collector.RecordMetric(&Metric{
		Provider:  "openai",
		Model:     "gpt-4",
		Operation: "completion",
		Duration:  100 * time.Millisecond,
		Success:   true,
	})

	// Get aggregated within last hour
	aggregated := collector.GetAggregatedMetrics(time.Hour)
	stats := aggregated["openai:gpt-4:completion"]
	require.NotNil(t, stats)
	assert.Equal(t, 1, stats.TotalRequests) // Only the new metric
}

func TestMetricsCollector_aggregateMetrics_Empty(t *testing.T) {
	collector := NewMetricsCollector()
	result := collector.aggregateMetrics([]*Metric{})
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.TotalRequests)
}

// ==================== MLPredictor Tests ====================

func TestNewMLPredictor(t *testing.T) {
	predictor := NewMLPredictor()
	assert.NotNil(t, predictor)
	assert.NotNil(t, predictor.models)
}

func TestMLPredictor_PredictProviderPerformance(t *testing.T) {
	predictor := NewMLPredictor()
	ctx := context.Background()

	prediction, err := predictor.PredictProviderPerformance(ctx, "openai", "gpt-4", nil)
	require.NoError(t, err)
	assert.NotNil(t, prediction)
	assert.Equal(t, "openai", prediction.Provider)
	assert.Equal(t, "gpt-4", prediction.Model)
	assert.Greater(t, prediction.PredictedLatency, time.Duration(0))
	assert.Greater(t, prediction.Confidence, 0.0)
	assert.LessOrEqual(t, prediction.Confidence, 1.0)
}

func TestMLPredictor_PredictProviderPerformance_HighLoad(t *testing.T) {
	predictor := NewMLPredictor()
	ctx := context.Background()

	features := map[string]interface{}{
		"load_level": "high",
	}

	prediction, err := predictor.PredictProviderPerformance(ctx, "openai", "gpt-4", features)
	require.NoError(t, err)

	// High load should increase predicted latency
	basePrediction, _ := predictor.PredictProviderPerformance(ctx, "openai", "gpt-4", nil)
	assert.Greater(t, prediction.PredictedLatency, basePrediction.PredictedLatency)
}

func TestMLPredictor_PredictProviderPerformance_LowLoad(t *testing.T) {
	predictor := NewMLPredictor()
	ctx := context.Background()

	features := map[string]interface{}{
		"load_level": "low",
	}

	prediction, err := predictor.PredictProviderPerformance(ctx, "openai", "gpt-4", features)
	require.NoError(t, err)

	// Low load should decrease predicted latency
	basePrediction, _ := predictor.PredictProviderPerformance(ctx, "openai", "gpt-4", nil)
	assert.Less(t, prediction.PredictedLatency, basePrediction.PredictedLatency)
}

func TestMLPredictor_PredictProviderPerformance_PeakTime(t *testing.T) {
	predictor := NewMLPredictor()
	ctx := context.Background()

	features := map[string]interface{}{
		"time_of_day": "peak",
	}

	prediction, err := predictor.PredictProviderPerformance(ctx, "anthropic", "claude-3", features)
	require.NoError(t, err)

	basePrediction, _ := predictor.PredictProviderPerformance(ctx, "anthropic", "claude-3", nil)
	assert.Greater(t, prediction.PredictedLatency, basePrediction.PredictedLatency)
}

func TestMLPredictor_PredictProviderPerformance_OffPeakTime(t *testing.T) {
	predictor := NewMLPredictor()
	ctx := context.Background()

	features := map[string]interface{}{
		"time_of_day": "off_peak",
	}

	prediction, err := predictor.PredictProviderPerformance(ctx, "google", "gemini", features)
	require.NoError(t, err)

	basePrediction, _ := predictor.PredictProviderPerformance(ctx, "google", "gemini", nil)
	assert.Less(t, prediction.PredictedLatency, basePrediction.PredictedLatency)
}

// ==================== RecommendationEngine Tests ====================

func TestNewRecommendationEngine(t *testing.T) {
	engine := NewRecommendationEngine()
	assert.NotNil(t, engine)
}

func TestRecommendationEngine_GetProviderRecommendation_Performance(t *testing.T) {
	engine := NewRecommendationEngine()
	ctx := context.Background()

	request := &RecommendationRequest{
		Priority: "performance",
	}

	response, err := engine.GetProviderRecommendation(ctx, request)
	require.NoError(t, err)
	assert.NotNil(t, response)
	assert.NotEmpty(t, response.RequestID)
	assert.NotEmpty(t, response.Recommendations)
	assert.False(t, response.GeneratedAt.IsZero())

	// Top recommendation should be performance-oriented
	top := response.Recommendations[0]
	assert.Greater(t, top.Score, 0.0)
}

func TestRecommendationEngine_GetProviderRecommendation_Cost(t *testing.T) {
	engine := NewRecommendationEngine()
	ctx := context.Background()

	request := &RecommendationRequest{
		Priority: "cost",
	}

	response, err := engine.GetProviderRecommendation(ctx, request)
	require.NoError(t, err)
	assert.NotEmpty(t, response.Recommendations)

	// Google should score well for cost
	for _, rec := range response.Recommendations {
		if rec.Provider == "google" {
			assert.Greater(t, rec.Score, 0.5)
			break
		}
	}
}

func TestRecommendationEngine_GetProviderRecommendation_Reliability(t *testing.T) {
	engine := NewRecommendationEngine()
	ctx := context.Background()

	request := &RecommendationRequest{
		Priority: "reliability",
	}

	response, err := engine.GetProviderRecommendation(ctx, request)
	require.NoError(t, err)
	assert.NotEmpty(t, response.Recommendations)

	// OpenAI should score well for reliability
	for _, rec := range response.Recommendations {
		if rec.Provider == "openai" {
			assert.Greater(t, rec.Score, 0.5)
			break
		}
	}
}

func TestRecommendationEngine_GetProviderRecommendation_Safety(t *testing.T) {
	engine := NewRecommendationEngine()
	ctx := context.Background()

	request := &RecommendationRequest{
		RequiresSafety: true,
	}

	response, err := engine.GetProviderRecommendation(ctx, request)
	require.NoError(t, err)
	assert.NotEmpty(t, response.Recommendations)

	// Anthropic should score highest for safety
	for _, rec := range response.Recommendations {
		if rec.Provider == "anthropic" {
			assert.Greater(t, rec.Score, 0.7)
			break
		}
	}
}

func TestRecommendationEngine_GetProviderRecommendation_LongContext(t *testing.T) {
	engine := NewRecommendationEngine()
	ctx := context.Background()

	request := &RecommendationRequest{
		MaxTokens: 150000, // Very long context
	}

	response, err := engine.GetProviderRecommendation(ctx, request)
	require.NoError(t, err)
	assert.NotEmpty(t, response.Recommendations)

	// Anthropic should score well for long context
	for _, rec := range response.Recommendations {
		if rec.Provider == "anthropic" {
			assert.NotEmpty(t, rec.Reasoning)
			break
		}
	}
}

func TestRecommendationEngine_GetProviderRecommendation_MultiModal(t *testing.T) {
	engine := NewRecommendationEngine()
	ctx := context.Background()

	request := &RecommendationRequest{
		RequiresMultiModal: true,
	}

	response, err := engine.GetProviderRecommendation(ctx, request)
	require.NoError(t, err)
	assert.NotEmpty(t, response.Recommendations)

	// OpenAI should score well for multimodal
	for _, rec := range response.Recommendations {
		if rec.Provider == "openai" {
			assert.Greater(t, rec.Score, 0.6)
			break
		}
	}
}

func TestRecommendationEngine_GetProviderRecommendation_WithUserHistory(t *testing.T) {
	engine := NewRecommendationEngine()
	ctx := context.Background()

	request := &RecommendationRequest{
		UserHistory: map[string]float64{
			"groq": 0.95, // High success rate with Groq
		},
	}

	response, err := engine.GetProviderRecommendation(ctx, request)
	require.NoError(t, err)
	assert.NotEmpty(t, response.Recommendations)

	// Groq should get a boost from user history
	for _, rec := range response.Recommendations {
		if rec.Provider == "groq" {
			// Should have reasoning mentioning past usage
			found := false
			for _, reason := range rec.Reasoning {
				if reason == "Based on your successful past usage" {
					found = true
					break
				}
			}
			assert.True(t, found)
			break
		}
	}
}

func TestRecommendationEngine_scoreProvider_ScoreClamping(t *testing.T) {
	engine := NewRecommendationEngine()

	rec := &ProviderRecommendation{Provider: "test"}
	request := &RecommendationRequest{}

	engine.scoreProvider(rec, request)

	// Score should be clamped between 0 and 1
	assert.GreaterOrEqual(t, rec.Score, 0.0)
	assert.LessOrEqual(t, rec.Score, 1.0)
}

// ==================== PerformanceAnalyzer Tests ====================

func TestNewPerformanceAnalyzer(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()
	assert.NotNil(t, analyzer)
}

func TestPerformanceAnalyzer_AnalyzeTrends(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()

	analysis, err := analyzer.AnalyzeTrends("openai", "gpt-4", 7)
	require.NoError(t, err)
	assert.NotNil(t, analysis)
	assert.Equal(t, "openai", analysis.Provider)
	assert.Equal(t, "gpt-4", analysis.Model)
	assert.Equal(t, "7 days", analysis.TimeRange)
	assert.NotEmpty(t, analysis.Metrics)
	assert.NotEmpty(t, analysis.Trends)
	assert.NotEmpty(t, analysis.Insights)
	assert.False(t, analysis.GeneratedAt.IsZero())

	// Check trend data
	latencyTrend, ok := analysis.Trends["latency"]
	assert.True(t, ok)
	assert.NotEmpty(t, latencyTrend.Direction)
	assert.GreaterOrEqual(t, latencyTrend.Confidence, 0.0)
	assert.LessOrEqual(t, latencyTrend.Confidence, 1.0)
}

func TestPerformanceAnalyzer_GenerateCostOptimization(t *testing.T) {
	analyzer := NewPerformanceAnalyzer()

	optimization, err := analyzer.GenerateCostOptimization("user123")
	require.NoError(t, err)
	assert.NotNil(t, optimization)
	assert.Equal(t, "user123", optimization.UserID)
	assert.Greater(t, optimization.CurrentCost, 0.0)
	assert.Greater(t, optimization.PotentialSavings, 0.0)
	assert.NotEmpty(t, optimization.Recommendations)
	assert.False(t, optimization.GeneratedAt.IsZero())

	// Check recommendation structure
	for _, rec := range optimization.Recommendations {
		assert.NotEmpty(t, rec.Type)
		assert.NotEmpty(t, rec.Description)
		assert.Greater(t, rec.PotentialSavings, 0.0)
		assert.NotEmpty(t, rec.Difficulty)
		assert.NotEmpty(t, rec.Impact)
	}
}

// ==================== Struct Tests ====================

func TestMetric_Struct(t *testing.T) {
	metric := Metric{
		ID:         "metric-123",
		Timestamp:  time.Now(),
		Provider:   "openai",
		Model:      "gpt-4",
		Operation:  "completion",
		Duration:   150 * time.Millisecond,
		Success:    true,
		TokenCount: 1000,
		Cost:       0.03,
		UserID:     "user123",
		IPAddress:  "192.168.1.1",
		Metadata:   map[string]interface{}{"key": "value"},
	}

	assert.Equal(t, "metric-123", metric.ID)
	assert.Equal(t, "openai", metric.Provider)
	assert.Equal(t, 1000, metric.TokenCount)
}

func TestAggregatedMetrics_Struct(t *testing.T) {
	metrics := AggregatedMetrics{
		TotalRequests: 100,
		SuccessRate:   0.95,
		AvgDuration:   200 * time.Millisecond,
		P50Duration:   180 * time.Millisecond,
		P95Duration:   350 * time.Millisecond,
		P99Duration:   500 * time.Millisecond,
		AvgTokens:     750,
		TotalCost:     25.50,
		AvgCost:       0.255,
	}

	assert.Equal(t, 100, metrics.TotalRequests)
	assert.Equal(t, 0.95, metrics.SuccessRate)
}

func TestPredictionModel_Struct(t *testing.T) {
	model := PredictionModel{
		Name:        "latency_predictor",
		Type:        "performance",
		Features:    []string{"provider", "model", "time_of_day"},
		Accuracy:    0.89,
		LastTrained: time.Now(),
		Parameters:  map[string]interface{}{"learning_rate": 0.01},
	}

	assert.Equal(t, "latency_predictor", model.Name)
	assert.Equal(t, "performance", model.Type)
	assert.Len(t, model.Features, 3)
}

func TestPerformancePrediction_Struct(t *testing.T) {
	prediction := PerformancePrediction{
		Provider:         "anthropic",
		Model:            "claude-3",
		PredictedLatency: 180 * time.Millisecond,
		Confidence:       0.92,
		Factor:           1.1,
	}

	assert.Equal(t, "anthropic", prediction.Provider)
	assert.Equal(t, 0.92, prediction.Confidence)
}

func TestRecommendationRequest_Struct(t *testing.T) {
	request := RecommendationRequest{
		Priority:           "performance",
		TaskType:           "chat",
		MaxTokens:          4096,
		RequiresSafety:     true,
		RequiresMultiModal: false,
		UserHistory:        map[string]float64{"openai": 0.95},
		ContextFeatures:    map[string]interface{}{"region": "us-east"},
	}

	assert.Equal(t, "performance", request.Priority)
	assert.True(t, request.RequiresSafety)
}

func TestRecommendationResponse_Struct(t *testing.T) {
	response := RecommendationResponse{
		RequestID: "req-123",
		Recommendations: []ProviderRecommendation{
			{Provider: "openai", Score: 0.9},
		},
		GeneratedAt: time.Now(),
	}

	assert.Equal(t, "req-123", response.RequestID)
	assert.Len(t, response.Recommendations, 1)
}

func TestProviderRecommendation_Struct(t *testing.T) {
	rec := ProviderRecommendation{
		Provider:  "openai",
		Score:     0.85,
		Reasoning: []string{"High performance", "Good reliability"},
		Tradeoffs: []string{"Higher cost"},
	}

	assert.Equal(t, "openai", rec.Provider)
	assert.Len(t, rec.Reasoning, 2)
	assert.Len(t, rec.Tradeoffs, 1)
}

func TestTrendAnalysis_Struct(t *testing.T) {
	analysis := TrendAnalysis{
		Provider:  "google",
		Model:     "gemini",
		TimeRange: "30 days",
		Metrics:   []string{"latency", "cost"},
		Trends: map[string]*Trend{
			"latency": {Direction: "improving", Magnitude: 0.1},
		},
		Insights:    []string{"Consider caching"},
		GeneratedAt: time.Now(),
	}

	assert.Equal(t, "google", analysis.Provider)
	assert.NotNil(t, analysis.Trends["latency"])
}

func TestTrend_Struct(t *testing.T) {
	trend := Trend{
		Direction:   "declining",
		Magnitude:   0.25,
		Confidence:  0.85,
		Description: "Performance is declining",
	}

	assert.Equal(t, "declining", trend.Direction)
	assert.Equal(t, 0.25, trend.Magnitude)
}

func TestCostOptimization_Struct(t *testing.T) {
	optimization := CostOptimization{
		UserID:           "user456",
		CurrentCost:      500.00,
		PotentialSavings: 125.00,
		Recommendations: []CostRecommendation{
			{Type: "provider_switch", PotentialSavings: 75.00},
		},
		GeneratedAt: time.Now(),
	}

	assert.Equal(t, "user456", optimization.UserID)
	assert.Equal(t, 125.00, optimization.PotentialSavings)
}

func TestCostRecommendation_Struct(t *testing.T) {
	rec := CostRecommendation{
		Type:             "model_optimization",
		Description:      "Use smaller models",
		PotentialSavings: 50.00,
		Difficulty:       "low",
		Impact:           "medium",
	}

	assert.Equal(t, "model_optimization", rec.Type)
	assert.Equal(t, "low", rec.Difficulty)
}

func TestAnalyticsSummary_Struct(t *testing.T) {
	summary := AnalyticsSummary{
		TotalRequests:   1000,
		TotalCost:       150.00,
		AvgResponseTime: 250 * time.Millisecond,
		TopProviders: []ProviderStats{
			{Provider: "openai", RequestCount: 500},
		},
		CostBreakdown:    map[string]float64{"openai": 100.00},
		PerformanceTrend: "stable",
		GeneratedAt:      time.Now(),
	}

	assert.Equal(t, 1000, summary.TotalRequests)
	assert.Equal(t, "stable", summary.PerformanceTrend)
}

func TestProviderStats_Struct(t *testing.T) {
	stats := ProviderStats{
		Provider:     "anthropic",
		RequestCount: 2500,
		SuccessRate:  0.98,
		AvgLatency:   180 * time.Millisecond,
		TotalCost:    350.75,
	}

	assert.Equal(t, "anthropic", stats.Provider)
	assert.Equal(t, 0.98, stats.SuccessRate)
}
