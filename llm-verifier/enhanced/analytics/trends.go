package analytics

import (
	"fmt"
	"math"
	"sort"
	"time"

	"llm-verifier/database"
)

// TrendAnalyzer analyzes trends in verification data
type TrendAnalyzer struct {
	db *database.Database
}

// NewTrendAnalyzer creates a new trend analyzer
func NewTrendAnalyzer(db *database.Database) *TrendAnalyzer {
	return &TrendAnalyzer{db: db}
}

// PerformanceTrend represents a performance trend over time
type PerformanceTrend struct {
	MetricName string          `json:"metric_name"`
	TimeRange  TimeRange       `json:"time_range"`
	DataPoints []DataPoint     `json:"data_points"`
	Trend      TrendDirection  `json:"trend"`
	Slope      float64         `json:"slope"`
	Confidence float64         `json:"confidence"`
	Anomalies  []Anomaly       `json:"anomalies,omitempty"`
	Forecast   []ForecastPoint `json:"forecast,omitempty"`
}

// TimeRange represents a time range for analysis
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// DataPoint represents a single data point
type DataPoint struct {
	Timestamp time.Time              `json:"timestamp"`
	Value     float64                `json:"value"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// TrendDirection represents the direction of a trend
type TrendDirection string

const (
	TrendIncreasing    TrendDirection = "increasing"
	TrendDecreasing    TrendDirection = "decreasing"
	TrendStable        TrendDirection = "stable"
	TrendUnpredictable TrendDirection = "unpredictable"
)

// Anomaly represents a detected anomaly
type Anomaly struct {
	Timestamp     time.Time `json:"timestamp"`
	Value         float64   `json:"value"`
	ExpectedValue float64   `json:"expected_value"`
	Severity      string    `json:"severity"` // low, medium, high, critical
	Description   string    `json:"description"`
}

// ForecastPoint represents a forecast data point
type ForecastPoint struct {
	Timestamp  time.Time `json:"timestamp"`
	Value      float64   `json:"value"`
	Confidence float64   `json:"confidence"`
}

// AnalyzePerformanceTrend analyzes performance trends for a metric
func (ta *TrendAnalyzer) AnalyzePerformanceTrend(metricName string, timeRange TimeRange, granularity time.Duration) (*PerformanceTrend, error) {
	// Get metric data from monitoring system
	// This is a simplified implementation - in reality, you'd query the monitoring system

	// Generate sample data for demonstration
	dataPoints := ta.generateSampleData(timeRange, granularity)

	trend := &PerformanceTrend{
		MetricName: metricName,
		TimeRange:  timeRange,
		DataPoints: dataPoints,
	}

	// Calculate trend
	trend.Trend, trend.Slope = ta.calculateTrend(dataPoints)

	// Calculate confidence
	trend.Confidence = ta.calculateConfidence(dataPoints)

	// Detect anomalies
	trend.Anomalies = ta.detectAnomalies(dataPoints)

	// Generate forecast
	trend.Forecast = ta.generateForecast(dataPoints, 10) // 10 points ahead

	return trend, nil
}

// generateSampleData generates sample data for demonstration
func (ta *TrendAnalyzer) generateSampleData(timeRange TimeRange, granularity time.Duration) []DataPoint {
	var points []DataPoint

	current := timeRange.Start
	baseValue := 100.0
	trend := 0.1 // Slight upward trend
	noise := 5.0 // Random noise

	for current.Before(timeRange.End) {
		// Add trend
		value := baseValue + trend*float64(len(points))

		// Add seasonal pattern
		seasonal := 10 * math.Sin(2*math.Pi*float64(len(points))/24) // Daily pattern

		// Add noise
		noiseValue := (float64(time.Now().UnixNano()%1000)/1000.0 - 0.5) * noise

		finalValue := value + seasonal + noiseValue

		points = append(points, DataPoint{
			Timestamp: current,
			Value:     finalValue,
			Metadata: map[string]interface{}{
				"source": "synthetic",
			},
		})

		current = current.Add(granularity)
	}

	return points
}

// calculateTrend calculates the trend direction and slope
func (ta *TrendAnalyzer) calculateTrend(points []DataPoint) (TrendDirection, float64) {
	if len(points) < 2 {
		return TrendStable, 0
	}

	// Simple linear regression
	n := float64(len(points))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	for i, point := range points {
		x := float64(i)
		y := point.Value

		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)

	// Determine direction
	var direction TrendDirection
	absSlope := math.Abs(slope)

	if absSlope < 0.01 {
		direction = TrendStable
	} else if absSlope < 0.1 {
		if slope > 0 {
			direction = TrendIncreasing
		} else {
			direction = TrendDecreasing
		}
	} else {
		if slope > 0 {
			direction = TrendIncreasing
		} else {
			direction = TrendDecreasing
		}
	}

	return direction, slope
}

// calculateConfidence calculates confidence in the trend analysis
func (ta *TrendAnalyzer) calculateConfidence(points []DataPoint) float64 {
	if len(points) < 3 {
		return 0.5
	}

	// Calculate R-squared (coefficient of determination)
	n := float64(len(points))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0
	sumYY := 0.0

	for i, point := range points {
		x := float64(i)
		y := point.Value

		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
		sumYY += y * y
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	intercept := (sumY - slope*sumX) / n

	// Calculate R-squared
	ssRes := 0.0
	ssTot := 0.0

	for i, point := range points {
		x := float64(i)
		y := point.Value
		predicted := slope*x + intercept

		ssRes += (y - predicted) * (y - predicted)
		ssTot += (y - sumY/n) * (y - sumY/n)
	}

	rSquared := 1 - (ssRes / ssTot)
	if rSquared < 0 {
		rSquared = 0
	}

	return math.Sqrt(rSquared) // Return correlation coefficient
}

// detectAnomalies detects anomalies in the data
func (ta *TrendAnalyzer) detectAnomalies(points []DataPoint) []Anomaly {
	var anomalies []Anomaly

	if len(points) < 10 {
		return anomalies
	}

	// Calculate rolling average and standard deviation
	windowSize := 5
	values := make([]float64, len(points))
	for i, point := range points {
		values[i] = point.Value
	}

	// Simple anomaly detection based on standard deviation
	for i := windowSize; i < len(values); i++ {
		window := values[i-windowSize : i]

		// Calculate mean and std dev of window
		sum := 0.0
		for _, v := range window {
			sum += v
		}
		mean := sum / float64(windowSize)

		sumSq := 0.0
		for _, v := range window {
			sumSq += (v - mean) * (v - mean)
		}
		stdDev := math.Sqrt(sumSq / float64(windowSize))

		// Check if current value is an outlier
		currentValue := values[i]
		zScore := math.Abs(currentValue-mean) / stdDev

		if zScore > 3.0 { // 3 standard deviations
			severity := "low"
			if zScore > 4.0 {
				severity = "medium"
			}
			if zScore > 5.0 {
				severity = "high"
			}
			if zScore > 6.0 {
				severity = "critical"
			}

			anomalies = append(anomalies, Anomaly{
				Timestamp:     points[i].Timestamp,
				Value:         currentValue,
				ExpectedValue: mean,
				Severity:      severity,
				Description:   fmt.Sprintf("Value deviates by %.2f standard deviations", zScore),
			})
		}
	}

	return anomalies
}

// generateForecast generates a simple forecast
func (ta *TrendAnalyzer) generateForecast(points []DataPoint, numPoints int) []ForecastPoint {
	if len(points) < 2 {
		return nil
	}

	// Use simple linear regression for forecasting
	n := float64(len(points))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	for i, point := range points {
		x := float64(i)
		y := point.Value

		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	intercept := (sumY - slope*sumX) / n

	// Calculate confidence interval (simplified)
	confidences := []float64{0.95, 0.90, 0.85, 0.80, 0.75, 0.70, 0.65, 0.60, 0.55, 0.50}

	var forecast []ForecastPoint
	lastTimestamp := points[len(points)-1].Timestamp

	for i := 1; i <= numPoints; i++ {
		x := float64(len(points) + i - 1)
		predictedValue := slope*x + intercept

		confidence := 0.8
		if i < len(confidences) {
			confidence = confidences[i-1]
		}

		forecast = append(forecast, ForecastPoint{
			Timestamp:  lastTimestamp.Add(time.Duration(i) * time.Hour),
			Value:      predictedValue,
			Confidence: confidence,
		})
	}

	return forecast
}

// UsagePatternAnalyzer analyzes usage patterns
type UsagePatternAnalyzer struct {
	db *database.Database
}

// NewUsagePatternAnalyzer creates a new usage pattern analyzer
func NewUsagePatternAnalyzer(db *database.Database) *UsagePatternAnalyzer {
	return &UsagePatternAnalyzer{db: db}
}

// AnalyzeUsagePatterns analyzes usage patterns over time
func (upa *UsagePatternAnalyzer) AnalyzeUsagePatterns(timeRange TimeRange) (*UsageAnalysis, error) {
	analysis := &UsageAnalysis{
		TimeRange: timeRange,
	}

	// Analyze hourly patterns
	analysis.HourlyPatterns = upa.analyzeHourlyPatterns(timeRange)

	// Analyze daily patterns
	analysis.DailyPatterns = upa.analyzeDailyPatterns(timeRange)

	// Analyze model popularity
	analysis.ModelPopularity = upa.analyzeModelPopularity(timeRange)

	// Analyze error patterns
	analysis.ErrorPatterns = upa.analyzeErrorPatterns(timeRange)

	// Generate recommendations
	analysis.Recommendations = upa.generateRecommendations(analysis)

	return analysis, nil
}

// UsageAnalysis represents usage pattern analysis
type UsageAnalysis struct {
	TimeRange       TimeRange          `json:"time_range"`
	HourlyPatterns  map[int]float64    `json:"hourly_patterns"`  // Hour -> average usage
	DailyPatterns   map[string]float64 `json:"daily_patterns"`   // Day -> average usage
	ModelPopularity map[string]int     `json:"model_popularity"` // Model -> usage count
	ErrorPatterns   map[string]int     `json:"error_patterns"`   // Error type -> count
	Recommendations []string           `json:"recommendations"`
}

// analyzeHourlyPatterns analyzes usage by hour
func (upa *UsagePatternAnalyzer) analyzeHourlyPatterns(timeRange TimeRange) map[int]float64 {
	patterns := make(map[int]float64)

	// Sample data - in real implementation, query database
	for hour := 0; hour < 24; hour++ {
		// Simulate typical usage patterns
		baseUsage := 50.0
		if hour >= 9 && hour <= 17 { // Business hours
			baseUsage = 100.0
		}
		if hour >= 12 && hour <= 13 { // Lunch time
			baseUsage = 80.0
		}
		if hour >= 18 && hour <= 6 { // Off hours
			baseUsage = 20.0
		}

		// Add some variation
		patterns[hour] = baseUsage + (float64(hour%5) * 10)
	}

	return patterns
}

// analyzeDailyPatterns analyzes usage by day of week
func (upa *UsagePatternAnalyzer) analyzeDailyPatterns(timeRange TimeRange) map[string]float64 {
	patterns := map[string]float64{
		"Monday":    90.0,
		"Tuesday":   95.0,
		"Wednesday": 100.0,
		"Thursday":  98.0,
		"Friday":    85.0,
		"Saturday":  40.0,
		"Sunday":    35.0,
	}

	return patterns
}

// analyzeModelPopularity analyzes which models are most used
func (upa *UsagePatternAnalyzer) analyzeModelPopularity(timeRange TimeRange) map[string]int {
	popularity := map[string]int{
		"gpt-4":           150,
		"gpt-3.5-turbo":   120,
		"claude-3-sonnet": 100,
		"gemini-pro":      80,
		"claude-2":        60,
		"other":           40,
	}

	return popularity
}

// analyzeErrorPatterns analyzes error patterns
func (upa *UsagePatternAnalyzer) analyzeErrorPatterns(timeRange TimeRange) map[string]int {
	patterns := map[string]int{
		"rate_limit_exceeded":  25,
		"model_unavailable":    15,
		"authentication_error": 10,
		"timeout":              8,
		"parsing_error":        5,
	}

	return patterns
}

// generateRecommendations generates recommendations based on analysis
func (upa *UsagePatternAnalyzer) generateRecommendations(analysis *UsageAnalysis) []string {
	var recommendations []string

	// Check peak hours
	peakHour := 0
	maxUsage := 0.0
	for hour, usage := range analysis.HourlyPatterns {
		if usage > maxUsage {
			maxUsage = usage
			peakHour = hour
		}
	}

	if peakHour >= 9 && peakHour <= 17 {
		recommendations = append(recommendations,
			fmt.Sprintf("Scale up resources during peak business hours (%d:00)", peakHour))
	}

	// Check model popularity
	var popularModels []string
	for model, count := range analysis.ModelPopularity {
		if count > 100 {
			popularModels = append(popularModels, model)
		}
	}

	if len(popularModels) > 0 {
		recommendations = append(recommendations,
			fmt.Sprintf("Ensure high availability for popular models: %v", popularModels))
	}

	// Check error patterns
	totalErrors := 0
	for _, count := range analysis.ErrorPatterns {
		totalErrors += count
	}

	if totalErrors > 20 {
		recommendations = append(recommendations,
			"Investigate and resolve high error rates to improve user experience")
	}

	// Add general recommendations
	recommendations = append(recommendations,
		"Consider implementing request queuing during peak hours",
		"Monitor model performance metrics for early issue detection",
		"Implement automatic scaling based on usage patterns")

	return recommendations
}

// CostOptimizationAnalyzer analyzes cost optimization opportunities
type CostOptimizationAnalyzer struct {
	db *database.Database
}

// NewCostOptimizationAnalyzer creates a new cost optimization analyzer
func NewCostOptimizationAnalyzer(db *database.Database) *CostOptimizationAnalyzer {
	return &CostOptimizationAnalyzer{db: db}
}

// AnalyzeCostOptimization analyzes cost optimization opportunities
func (coa *CostOptimizationAnalyzer) AnalyzeCostOptimization(timeRange TimeRange) (*CostAnalysis, error) {
	analysis := &CostAnalysis{
		TimeRange: timeRange,
	}

	// Analyze current spending
	analysis.CurrentSpending = coa.analyzeCurrentSpending(timeRange)

	// Identify optimization opportunities
	analysis.OptimizationOpportunities = coa.identifyOptimizationOpportunities()

	// Generate recommendations
	analysis.Recommendations = coa.generateCostRecommendations(analysis)

	// Calculate potential savings
	analysis.PotentialSavings = coa.calculatePotentialSavings(analysis)

	return analysis, nil
}

// CostAnalysis represents cost analysis and optimization opportunities
type CostAnalysis struct {
	TimeRange                  TimeRange                 `json:"time_range"`
	CurrentSpending            SpendingBreakdown         `json:"current_spending"`
	OptimizationOpportunities  []OptimizationOpportunity `json:"optimization_opportunities"`
	Recommendations            []string                  `json:"recommendations"`
	PotentialSavings           float64                   `json:"potential_savings"`
	PotentialSavingsPercentage float64                   `json:"potential_savings_percentage"`
}

// SpendingBreakdown represents spending breakdown by category
type SpendingBreakdown struct {
	ByProvider map[string]float64 `json:"by_provider"`
	ByModel    map[string]float64 `json:"by_model"`
	ByTime     map[string]float64 `json:"by_time"` // hourly breakdown
	Total      float64            `json:"total"`
}

// OptimizationOpportunity represents a cost optimization opportunity
type OptimizationOpportunity struct {
	Type             string  `json:"type"` // model_switch, batch_processing, caching, etc.
	Description      string  `json:"description"`
	PotentialSavings float64 `json:"potential_savings"`
	Difficulty       string  `json:"difficulty"` // low, medium, high
	Impact           string  `json:"impact"`     // low, medium, high
}

// analyzeCurrentSpending analyzes current spending patterns
func (coa *CostOptimizationAnalyzer) analyzeCurrentSpending(timeRange TimeRange) SpendingBreakdown {
	breakdown := SpendingBreakdown{
		ByProvider: make(map[string]float64),
		ByModel:    make(map[string]float64),
		ByTime:     make(map[string]float64),
	}

	// Sample data - in real implementation, query pricing and usage data
	breakdown.ByProvider = map[string]float64{
		"OpenAI":    1500.0,
		"Anthropic": 800.0,
		"Google":    400.0,
		"Others":    200.0,
	}

	breakdown.ByModel = map[string]float64{
		"gpt-4":           1200.0,
		"claude-3-sonnet": 700.0,
		"gpt-3.5-turbo":   600.0,
		"gemini-pro":      300.0,
		"others":          100.0,
	}

	// Hourly spending (simulate daily pattern)
	for hour := 0; hour < 24; hour++ {
		multiplier := 1.0
		if hour >= 9 && hour <= 17 { // Business hours
			multiplier = 2.0
		}
		breakdown.ByTime[fmt.Sprintf("%02d:00", hour)] = 75.0 * multiplier
	}

	// Calculate total
	for _, amount := range breakdown.ByProvider {
		breakdown.Total += amount
	}

	return breakdown
}

// identifyOptimizationOpportunities identifies cost optimization opportunities
func (coa *CostOptimizationAnalyzer) identifyOptimizationOpportunities() []OptimizationOpportunity {
	opportunities := []OptimizationOpportunity{
		{
			Type:             "model_switch",
			Description:      "Switch from GPT-4 to GPT-3.5-turbo for non-critical tasks",
			PotentialSavings: 500.0,
			Difficulty:       "medium",
			Impact:           "high",
		},
		{
			Type:             "batch_processing",
			Description:      "Implement batch processing for multiple similar requests",
			PotentialSavings: 200.0,
			Difficulty:       "high",
			Impact:           "medium",
		},
		{
			Type:             "response_caching",
			Description:      "Implement intelligent response caching for repeated queries",
			PotentialSavings: 150.0,
			Difficulty:       "medium",
			Impact:           "high",
		},
		{
			Type:             "usage_optimization",
			Description:      "Optimize prompt length and reduce unnecessary tokens",
			PotentialSavings: 100.0,
			Difficulty:       "low",
			Impact:           "medium",
		},
		{
			Type:             "off_peak_scheduling",
			Description:      "Schedule non-urgent tasks during off-peak hours",
			PotentialSavings: 75.0,
			Difficulty:       "low",
			Impact:           "low",
		},
	}

	return opportunities
}

// generateCostRecommendations generates cost optimization recommendations
func (coa *CostOptimizationAnalyzer) generateCostRecommendations(analysis *CostAnalysis) []string {
	var recommendations []string

	// Sort opportunities by potential savings
	opportunities := analysis.OptimizationOpportunities
	sort.Slice(opportunities, func(i, j int) bool {
		return opportunities[i].PotentialSavings > opportunities[j].PotentialSavings
	})

	// Add top recommendations
	for _, opp := range opportunities[:3] {
		recommendations = append(recommendations,
			fmt.Sprintf("%s: Potential savings $%.2f (%s difficulty, %s impact)",
				opp.Description, opp.PotentialSavings, opp.Difficulty, opp.Impact))
	}

	// Add general recommendations
	recommendations = append(recommendations,
		"Implement usage monitoring and set up spending alerts",
		"Consider reserved instances or committed use discounts where available",
		"Regularly review and optimize model selection based on task requirements",
		"Implement request deduplication to avoid redundant API calls")

	return recommendations
}

// calculatePotentialSavings calculates total potential savings
func (coa *CostOptimizationAnalyzer) calculatePotentialSavings(analysis *CostAnalysis) float64 {
	totalSavings := 0.0
	for _, opp := range analysis.OptimizationOpportunities {
		totalSavings += opp.PotentialSavings
	}

	analysis.PotentialSavingsPercentage = (totalSavings / analysis.CurrentSpending.Total) * 100

	return totalSavings
}
