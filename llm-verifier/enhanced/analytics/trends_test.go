package analytics

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== TrendAnalyzer Tests ====================

func TestNewTrendAnalyzer(t *testing.T) {
	analyzer := NewTrendAnalyzer(nil)
	assert.NotNil(t, analyzer)
}

func TestTrendAnalyzer_AnalyzePerformanceTrend(t *testing.T) {
	analyzer := NewTrendAnalyzer(nil)

	timeRange := TimeRange{
		From: time.Now().Add(-24 * time.Hour),
		To:   time.Now(),
	}

	result, err := analyzer.AnalyzePerformanceTrend("response_time", timeRange, time.Hour)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "response_time", result.MetricName)
	assert.NotEmpty(t, result.DataPoints)
}

func TestTrendAnalyzer_GenerateSampleData(t *testing.T) {
	analyzer := NewTrendAnalyzer(nil)

	timeRange := TimeRange{
		From: time.Now().Add(-6 * time.Hour),
		To:   time.Now(),
	}

	// Private method test via AnalyzePerformanceTrend
	result, err := analyzer.AnalyzePerformanceTrend("test_metric", timeRange, time.Hour)
	require.NoError(t, err)
	assert.NotEmpty(t, result.DataPoints)
}

func TestTrendAnalyzer_CalculateTrend(t *testing.T) {
	analyzer := NewTrendAnalyzer(nil)

	timeRange := TimeRange{
		From: time.Now().Add(-12 * time.Hour),
		To:   time.Now(),
	}

	result, err := analyzer.AnalyzePerformanceTrend("upward_metric", timeRange, time.Hour)
	require.NoError(t, err)

	// Should have a valid trend direction
	assert.Contains(t, []TrendDirection{TrendDirectionUpward, TrendDirectionDownward, TrendDirectionStable}, result.Trend)
}

// ==================== UsagePatternAnalyzer Tests ====================

func TestNewUsagePatternAnalyzer(t *testing.T) {
	analyzer := NewUsagePatternAnalyzer(nil)
	assert.NotNil(t, analyzer)
}

func TestUsagePatternAnalyzer_AnalyzeUsagePatterns(t *testing.T) {
	analyzer := NewUsagePatternAnalyzer(nil)

	timeRange := TimeRange{
		From: time.Now().Add(-24 * time.Hour),
		To:   time.Now(),
	}

	result, err := analyzer.AnalyzeUsagePatterns(timeRange)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.HourlyPatterns)
	assert.NotNil(t, result.DailyPatterns)
}

// ==================== CostOptimizationAnalyzer Tests ====================

func TestNewCostOptimizationAnalyzer(t *testing.T) {
	analyzer := NewCostOptimizationAnalyzer(nil)
	assert.NotNil(t, analyzer)
}

func TestCostOptimizationAnalyzer_AnalyzeCostOptimization(t *testing.T) {
	analyzer := NewCostOptimizationAnalyzer(nil)

	timeRange := TimeRange{
		From: time.Now().Add(-24 * time.Hour),
		To:   time.Now(),
	}

	result, err := analyzer.AnalyzeCostOptimization(timeRange)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.CostByModel)
	assert.NotNil(t, result.CostByProvider)
}

// ==================== Struct Tests ====================

func TestPerformanceTrend_Struct(t *testing.T) {
	trend := PerformanceTrend{
		MetricName: "test_metric",
		Trend:      TrendDirectionUpward,
		Slope:      0.5,
		Confidence: 0.95,
	}

	assert.Equal(t, "test_metric", trend.MetricName)
	assert.Equal(t, TrendDirectionUpward, trend.Trend)
	assert.Equal(t, 0.5, trend.Slope)
}

func TestDataPoint_Struct(t *testing.T) {
	dp := DataPoint{
		Timestamp: time.Now(),
		Value:     42.5,
		Metadata:  map[string]interface{}{"key": "value"},
	}

	assert.Equal(t, 42.5, dp.Value)
	assert.NotNil(t, dp.Metadata)
}

func TestAnomaly_Struct(t *testing.T) {
	anomaly := Anomaly{
		Timestamp: time.Now(),
		Value:     100.0,
		Expected:  50.0,
		Severity:  "high",
	}

	assert.Equal(t, 100.0, anomaly.Value)
	assert.Equal(t, 50.0, anomaly.Expected)
	assert.Equal(t, "high", anomaly.Severity)
}

func TestForecastPoint_Struct(t *testing.T) {
	forecast := ForecastPoint{
		Timestamp:  time.Now().Add(24 * time.Hour),
		Value:      75.0,
		Confidence: 0.85,
	}

	assert.Equal(t, 75.0, forecast.Value)
	assert.Equal(t, 0.85, forecast.Confidence)
}

func TestUsageAnalysis_Struct(t *testing.T) {
	analysis := UsageAnalysis{
		HourlyPatterns:  map[int]float64{0: 10.0, 12: 50.0},
		DailyPatterns:   map[string]float64{"Monday": 100.0},
		ModelPopularity: map[string]int{"gpt-4": 100},
		ErrorPatterns:   map[string]int{"timeout": 5},
		Recommendations: []string{"Optimize morning usage"},
	}

	assert.Len(t, analysis.HourlyPatterns, 2)
	assert.Equal(t, 100.0, analysis.DailyPatterns["Monday"])
	assert.Len(t, analysis.Recommendations, 1)
}

func TestCostAnalysis_Struct(t *testing.T) {
	analysis := CostAnalysis{
		TotalCost:        1000.0,
		CostByModel:      map[string]float64{"gpt-4": 500.0},
		CostByProvider:   map[string]float64{"openai": 800.0},
		CostByEndpoint:   map[string]float64{"/chat": 600.0},
		Recommendations:  []string{"Use cheaper models for simple tasks"},
		PotentialSavings: 200.0,
	}

	assert.Equal(t, 1000.0, analysis.TotalCost)
	assert.Equal(t, 500.0, analysis.CostByModel["gpt-4"])
	assert.Equal(t, 200.0, analysis.PotentialSavings)
}

func TestSpendingBreakdown_Struct(t *testing.T) {
	breakdown := SpendingBreakdown{
		ByModel:     map[string]float64{"gpt-4": 100.0},
		ByProvider:  map[string]float64{"openai": 200.0},
		ByEndpoint:  map[string]float64{"/api/chat": 150.0},
		ByTimeOfDay: map[string]float64{"morning": 80.0},
	}

	assert.Equal(t, 100.0, breakdown.ByModel["gpt-4"])
	assert.Equal(t, 200.0, breakdown.ByProvider["openai"])
}

// ==================== TrendDirection Tests ====================

func TestTrendDirection_Constants(t *testing.T) {
	assert.Equal(t, TrendDirection("upward"), TrendDirectionUpward)
	assert.Equal(t, TrendDirection("downward"), TrendDirectionDownward)
	assert.Equal(t, TrendDirection("stable"), TrendDirectionStable)
}

// ==================== Helper Function Tests ====================

func TestCalculateTrend_Upward(t *testing.T) {
	// Simulate upward trend data
	dataPoints := []DataPoint{
		{Value: 10.0, Timestamp: time.Now().Add(-5 * time.Hour)},
		{Value: 20.0, Timestamp: time.Now().Add(-4 * time.Hour)},
		{Value: 30.0, Timestamp: time.Now().Add(-3 * time.Hour)},
		{Value: 40.0, Timestamp: time.Now().Add(-2 * time.Hour)},
		{Value: 50.0, Timestamp: time.Now().Add(-1 * time.Hour)},
	}

	slope := calculateSlope(dataPoints)
	assert.Greater(t, slope, 0.0) // Upward trend should have positive slope
}

func TestCalculateTrend_Downward(t *testing.T) {
	dataPoints := []DataPoint{
		{Value: 50.0, Timestamp: time.Now().Add(-5 * time.Hour)},
		{Value: 40.0, Timestamp: time.Now().Add(-4 * time.Hour)},
		{Value: 30.0, Timestamp: time.Now().Add(-3 * time.Hour)},
		{Value: 20.0, Timestamp: time.Now().Add(-2 * time.Hour)},
		{Value: 10.0, Timestamp: time.Now().Add(-1 * time.Hour)},
	}

	slope := calculateSlope(dataPoints)
	assert.Less(t, slope, 0.0) // Downward trend should have negative slope
}

func TestCalculateTrend_Stable(t *testing.T) {
	dataPoints := []DataPoint{
		{Value: 50.0, Timestamp: time.Now().Add(-5 * time.Hour)},
		{Value: 50.0, Timestamp: time.Now().Add(-4 * time.Hour)},
		{Value: 50.0, Timestamp: time.Now().Add(-3 * time.Hour)},
		{Value: 50.0, Timestamp: time.Now().Add(-2 * time.Hour)},
		{Value: 50.0, Timestamp: time.Now().Add(-1 * time.Hour)},
	}

	slope := calculateSlope(dataPoints)
	assert.InDelta(t, 0.0, slope, 0.1) // Stable trend should have ~0 slope
}

// calculateSlope is a helper to calculate the slope of data points
func calculateSlope(points []DataPoint) float64 {
	if len(points) < 2 {
		return 0.0
	}

	n := float64(len(points))
	var sumX, sumY, sumXY, sumX2 float64

	for i, p := range points {
		x := float64(i)
		y := p.Value
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	denominator := n*sumX2 - sumX*sumX
	if math.Abs(denominator) < 1e-10 {
		return 0.0
	}

	return (n*sumXY - sumX*sumY) / denominator
}

// ==================== Edge Case Tests ====================

func TestTrendAnalyzer_EmptyTimeRange(t *testing.T) {
	analyzer := NewTrendAnalyzer(nil)

	timeRange := TimeRange{
		From: time.Now(),
		To:   time.Now(),
	}

	result, err := analyzer.AnalyzePerformanceTrend("test", timeRange, time.Hour)
	// Should handle empty range gracefully
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestUsagePatternAnalyzer_EmptyTimeRange(t *testing.T) {
	analyzer := NewUsagePatternAnalyzer(nil)

	timeRange := TimeRange{
		From: time.Now(),
		To:   time.Now(),
	}

	result, err := analyzer.AnalyzeUsagePatterns(timeRange)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestCostOptimizationAnalyzer_EmptyTimeRange(t *testing.T) {
	analyzer := NewCostOptimizationAnalyzer(nil)

	timeRange := TimeRange{
		From: time.Now(),
		To:   time.Now(),
	}

	result, err := analyzer.AnalyzeCostOptimization(timeRange)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestTrendAnalyzer_LargeTimeRange(t *testing.T) {
	analyzer := NewTrendAnalyzer(nil)

	timeRange := TimeRange{
		From: time.Now().Add(-30 * 24 * time.Hour), // 30 days
		To:   time.Now(),
	}

	result, err := analyzer.AnalyzePerformanceTrend("monthly_metric", timeRange, 24*time.Hour)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.DataPoints)
}
