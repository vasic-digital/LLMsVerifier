package analytics

import (
	"math"
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

// DataPoint represents a single data point
type DataPoint struct {
	Timestamp time.Time              `json:"timestamp"`
	Value     float64                `json:"value"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// TrendDirection represents the direction of a trend
type TrendDirection string

const (
	TrendDirectionUpward   TrendDirection = "upward"
	TrendDirectionDownward TrendDirection = "downward"
	TrendDirectionStable   TrendDirection = "stable"
)

// Anomaly represents an anomaly in the data
type Anomaly struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Expected  float64   `json:"expected"`
	Severity  string    `json:"severity"`
}

// ForecastPoint represents a forecast data point
type ForecastPoint struct {
	Timestamp  time.Time `json:"timestamp"`
	Value      float64   `json:"value"`
	Confidence float64   `json:"confidence"`
}

// UsagePatternAnalyzer analyzes usage patterns
type UsagePatternAnalyzer struct {
	db *database.Database
}

// NewUsagePatternAnalyzer creates a new usage pattern analyzer
func NewUsagePatternAnalyzer(db *database.Database) *UsagePatternAnalyzer {
	return &UsagePatternAnalyzer{db: db}
}

// UsageAnalysis represents the result of usage pattern analysis
type UsageAnalysis struct {
	TimeRange       TimeRange          `json:"time_range"`
	HourlyPatterns  map[int]float64    `json:"hourly_patterns"`  // Hour -> average usage
	DailyPatterns   map[string]float64 `json:"daily_patterns"`   // Day -> average usage
	ModelPopularity map[string]int     `json:"model_popularity"` // Model -> usage count
	ErrorPatterns   map[string]int     `json:"error_patterns"`   // Error type -> count
	Recommendations []string           `json:"recommendations"`
}

// CostOptimizationAnalyzer analyzes cost optimization opportunities
type CostOptimizationAnalyzer struct {
	db *database.Database
}

// NewCostOptimizationAnalyzer creates a new cost optimization analyzer
func NewCostOptimizationAnalyzer(db *database.Database) *CostOptimizationAnalyzer {
	return &CostOptimizationAnalyzer{db: db}
}

// CostAnalysis represents the result of cost optimization analysis
type CostAnalysis struct {
	TimeRange        TimeRange          `json:"time_range"`
	TotalCost        float64            `json:"total_cost"`
	CostByModel      map[string]float64 `json:"cost_by_model"`
	CostByProvider   map[string]float64 `json:"cost_by_provider"`
	CostByEndpoint   map[string]float64 `json:"cost_by_endpoint"`
	Recommendations  []string           `json:"recommendations"`
	PotentialSavings float64            `json:"potential_savings"`
}

// SpendingBreakdown represents a breakdown of spending
type SpendingBreakdown struct {
	ByModel     map[string]float64 `json:"by_model"`
	ByProvider  map[string]float64 `json:"by_provider"`
	ByEndpoint  map[string]float64 `json:"by_endpoint"`
	ByTimeOfDay map[string]float64 `json:"by_time_of_day"`
}

// AnalyzePerformanceTrend analyzes performance trends for a specific metric
func (ta *TrendAnalyzer) AnalyzePerformanceTrend(metricName string, timeRange TimeRange, granularity time.Duration) (*PerformanceTrend, error) {
	// In a real implementation, you would query the database for actual data
	// For now, we'll generate sample data
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

	current := timeRange.From
	baseValue := 100.0
	trend := 0.1 // Slight upward trend
	noise := 5.0 // Random noise

	for current.Before(timeRange.To) {
		// Add trend
		value := baseValue + trend*float64(len(points))

		// Add seasonal pattern
		seasonal := 10 * math.Sin(2*math.Pi*float64(len(points))/24) // Daily pattern

		// Add noise
		noiseValue := (float64(time.Now().UnixNano()%1000)/1000.0 - 0.5) * noise

		points = append(points, DataPoint{
			Timestamp: current,
			Value:     value + seasonal + noiseValue,
		})

		current = current.Add(granularity)
	}

	return points
}

// calculateTrend calculates the trend direction and slope
func (ta *TrendAnalyzer) calculateTrend(dataPoints []DataPoint) (TrendDirection, float64) {
	if len(dataPoints) < 2 {
		return TrendDirectionStable, 0.0
	}

	// Simple linear regression
	n := float64(len(dataPoints))
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0

	for i, point := range dataPoints {
		x := float64(i)
		sumX += x
		sumY += point.Value
		sumXY += x * point.Value
		sumX2 += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// Determine trend direction
	direction := TrendDirectionStable
	if slope > 0.1 {
		direction = TrendDirectionUpward
	} else if slope < -0.1 {
		direction = TrendDirectionDownward
	}

	return direction, slope
}

// calculateConfidence calculates confidence in the trend
func (ta *TrendAnalyzer) calculateConfidence(dataPoints []DataPoint) float64 {
	if len(dataPoints) < 2 {
		return 0.0
	}

	// Calculate correlation coefficient (simplified)
	n := float64(len(dataPoints))
	sumX, sumY, sumXY, sumX2, sumY2 := 0.0, 0.0, 0.0, 0.0, 0.0

	for i, point := range dataPoints {
		x := float64(i)
		sumX += x
		sumY += point.Value
		sumXY += x * point.Value
		sumX2 += x * x
		sumY2 += point.Value * point.Value
	}

	numerator := n*sumXY - sumX*sumY
	denominator := math.Sqrt((n*sumX2 - sumX*sumX) * (n*sumY2 - sumY*sumY))

	if denominator == 0 {
		return 0.0
	}

	correlation := numerator / denominator
	return math.Abs(correlation) // Convert to confidence
}

// detectAnomalies detects anomalies in the data
func (ta *TrendAnalyzer) detectAnomalies(dataPoints []DataPoint) []Anomaly {
	if len(dataPoints) < 10 {
		return nil
	}

	var anomalies []Anomaly

	// Calculate moving average and standard deviation
	window := 10
	for i := window; i < len(dataPoints); i++ {
		var sum, sum2 float64
		for j := i - window; j < i; j++ {
			sum += dataPoints[j].Value
			sum2 += dataPoints[j].Value * dataPoints[j].Value
		}

		mean := sum / float64(window)
		variance := (sum2 / float64(window)) - (mean * mean)
		stdDev := math.Sqrt(variance)

		// Check if current point is an anomaly (2 standard deviations away)
		current := dataPoints[i]
		if math.Abs(current.Value-mean) > 2*stdDev {
			anomaly := Anomaly{
				Timestamp: current.Timestamp,
				Value:     current.Value,
				Expected:  mean,
				Severity:  "medium",
			}

			if math.Abs(current.Value-mean) > 3*stdDev {
				anomaly.Severity = "high"
			}

			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

// generateForecast generates forecast points
func (ta *TrendAnalyzer) generateForecast(dataPoints []DataPoint, points int) []ForecastPoint {
	if len(dataPoints) < 2 {
		return nil
	}

	// Simple linear extrapolation
	lastPoint := dataPoints[len(dataPoints)-1]
	_, slope := ta.calculateTrend(dataPoints)

	var forecast []ForecastPoint
	granularity := time.Hour // Default granularity

	if len(dataPoints) >= 2 {
		granularity = dataPoints[1].Timestamp.Sub(dataPoints[0].Timestamp)
	}

	for i := 1; i <= points; i++ {
		forecastTime := lastPoint.Timestamp.Add(time.Duration(i) * granularity)
		forecastValue := lastPoint.Value + slope*float64(i)

		// Decrease confidence for further ahead predictions
		confidence := 1.0 - (float64(i)/float64(points))*0.5

		forecast = append(forecast, ForecastPoint{
			Timestamp:  forecastTime,
			Value:      forecastValue,
			Confidence: confidence,
		})
	}

	return forecast
}

// AnalyzeUsagePatterns analyzes usage patterns over time
func (upa *UsagePatternAnalyzer) AnalyzeUsagePatterns(timeRange TimeRange) (*UsageAnalysis, error) {
	analysis := &UsageAnalysis{
		TimeRange:       timeRange,
		HourlyPatterns:  upa.analyzeHourlyPatterns(timeRange),
		DailyPatterns:   upa.analyzeDailyPatterns(timeRange),
		ModelPopularity: upa.analyzeModelPopularity(timeRange),
		ErrorPatterns:   upa.analyzeErrorPatterns(timeRange),
		Recommendations: upa.generateUsageRecommendations(),
	}

	return analysis, nil
}

// analyzeHourlyPatterns analyzes usage by hour of day
func (upa *UsagePatternAnalyzer) analyzeHourlyPatterns(timeRange TimeRange) map[int]float64 {
	patterns := make(map[int]float64)

	// In a real implementation, query the database
	// For now, generate sample patterns
	for hour := 0; hour < 24; hour++ {
		// Peak hours during business hours
		if hour >= 9 && hour <= 17 {
			patterns[hour] = 100.0 + (float64(hour%4) * 10)
		} else if hour >= 18 && hour <= 22 {
			patterns[hour] = 60.0 + (float64(hour%2) * 15)
		} else {
			patterns[hour] = 20.0 + (float64(hour%3) * 5)
		}
	}

	return patterns
}

// analyzeDailyPatterns analyzes usage by day of week
func (upa *UsagePatternAnalyzer) analyzeDailyPatterns(timeRange TimeRange) map[string]float64 {
	patterns := make(map[string]float64)

	days := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}

	// Higher usage during weekdays
	for i, day := range days {
		if i < 5 { // Weekdays
			patterns[day] = 90.0 + (float64(i) * 5)
		} else { // Weekends
			patterns[day] = 40.0 + (float64(i-5) * 10)
		}
	}

	return patterns
}

// analyzeModelPopularity analyzes model popularity
func (upa *UsagePatternAnalyzer) analyzeModelPopularity(timeRange TimeRange) map[string]int {
	popularity := map[string]int{
		"gpt-3.5-turbo":  450,
		"gpt-4":          300,
		"claude-3-opus":  200,
		"gemini-pro":     150,
		"llama-2":        100,
		"codellama":      80,
		"mistral":        60,
		"phi-2":          40,
		"qwen-turbo":     35,
		"deepseek-coder": 30,
	}

	return popularity
}

// analyzeErrorPatterns analyzes error patterns
func (upa *UsagePatternAnalyzer) analyzeErrorPatterns(timeRange TimeRange) map[string]int {
	patterns := map[string]int{
		"timeout_error":        25,
		"rate_limit_error":     18,
		"authentication_error": 12,
		"model_not_found":      8,
		"invalid_request":      6,
		"insufficient_quota":   4,
		"content_filter":       3,
		"network_error":        2,
		"server_error":         1,
		"unknown_error":        1,
	}

	return patterns
}

// generateUsageRecommendations generates recommendations based on usage patterns
func (upa *UsagePatternAnalyzer) generateUsageRecommendations() []string {
	return []string{
		"Consider implementing rate limiting during peak hours (9 AM - 5 PM)",
		"Optimize model selection based on task complexity to reduce costs",
		"Implement caching for frequently repeated requests",
		"Consider using smaller models for simple tasks during peak hours",
		"Schedule batch processing during off-peak hours (10 PM - 6 AM)",
		"Implement error retry logic for transient failures",
		"Consider load balancing across multiple providers",
	}
}

// AnalyzeCostOptimization analyzes cost optimization opportunities
func (coa *CostOptimizationAnalyzer) AnalyzeCostOptimization(timeRange TimeRange) (*CostAnalysis, error) {
	spending := coa.analyzeCurrentSpending(timeRange)

	analysis := &CostAnalysis{
		TimeRange:        timeRange,
		TotalCost:        spending.ByModel["gpt-4"]*0.03 + spending.ByModel["gpt-3.5-turbo"]*0.002 + spending.ByModel["claude-3-opus"]*0.015,
		CostByModel:      spending.ByModel,
		CostByProvider:   spending.ByProvider,
		CostByEndpoint:   spending.ByEndpoint,
		Recommendations:  coa.generateCostRecommendations(spending),
		PotentialSavings: 0.25, // 25% potential savings
	}

	return analysis, nil
}

// analyzeCurrentSpending analyzes current spending patterns
func (coa *CostOptimizationAnalyzer) analyzeCurrentSpending(timeRange TimeRange) SpendingBreakdown {
	return SpendingBreakdown{
		ByModel: map[string]float64{
			"gpt-4":         300.0,
			"gpt-3.5-turbo": 450.0,
			"claude-3-opus": 200.0,
			"gemini-pro":    150.0,
			"llama-2":       100.0,
		},
		ByProvider: map[string]float64{
			"openai":    750.0,
			"anthropic": 200.0,
			"google":    150.0,
			"meta":      100.0,
		},
		ByEndpoint: map[string]float64{
			"/verify":    600.0,
			"/compare":   300.0,
			"/analyze":   200.0,
			"/recommend": 100.0,
		},
		ByTimeOfDay: map[string]float64{
			"morning":   400.0,
			"afternoon": 500.0,
			"evening":   200.0,
			"night":     100.0,
		},
	}
}

// generateCostRecommendations generates cost optimization recommendations
func (coa *CostOptimizationAnalyzer) generateCostRecommendations(spending SpendingBreakdown) []string {
	return []string{
		"Use GPT-3.5-turbo for simple tasks, save GPT-4 for complex reasoning",
		"Implement response caching to reduce API calls",
		"Consider using open-source models for non-critical tasks",
		"Optimize prompt engineering to reduce token usage",
		"Implement batch processing to take advantage of volume discounts",
		"Use smaller models for initial drafts, larger models for refinement",
		"Consider spot instances or reserved capacity for predictable workloads",
	}
}
