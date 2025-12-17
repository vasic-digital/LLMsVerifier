package analytics

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	enhancedContext "llm-verifier/enhanced/context"
)

// MetricType represents different types of metrics
type MetricType int

const (
	MetricTypeCounter MetricType = iota
	MetricTypeGauge
	MetricTypeHistogram
	MetricTypeSummary
)

// AnalyticsMetric represents a single analytics metric
type AnalyticsMetric struct {
	Name       string            `json:"name"`
	Type       MetricType        `json:"type"`
	Value      float64           `json:"value"`
	Timestamp  time.Time         `json:"timestamp"`
	Tags       map[string]string `json:"tags,omitempty"`
	Dimensions map[string]any    `json:"dimensions,omitempty"`
}

// TimeSeries represents a time series of metrics
type TimeSeries struct {
	Name       string            `json:"name"`
	Metrics    []float64         `json:"metrics"`
	Timestamps []time.Time       `json:"timestamps"`
	Tags       map[string]string `json:"tags,omitempty"`
	Unit       string            `json:"unit,omitempty"`
}

// AnalyticsQuery represents a query for analytics data
type AnalyticsQuery struct {
	MetricNames    []string          `json:"metric_names"`
	Tags           map[string]string `json:"tags,omitempty"`
	QueryTimeRange QueryTimeRange    `json:"time_range"`
	Aggregation    AggregationType   `json:"aggregation"`
	GroupBy        []string          `json:"group_by,omitempty"`
	Filters        []QueryFilter     `json:"filters,omitempty"`
}

// QueryTimeRange defines a time range for queries
type QueryTimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// TimeRange defines a time range for queries
type TimeRange struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

// AggregationType represents how to aggregate metrics
type AggregationType int

const (
	AggregationSum AggregationType = iota
	AggregationAvg
	AggregationMin
	AggregationMax
	AggregationCount
	AggregationPercentile50
	AggregationPercentile90
	AggregationPercentile95
	AggregationPercentile99
)

// QueryFilter represents a filter for analytics queries
type QueryFilter struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // eq, ne, gt, lt, gte, lte, in, contains
	Value    interface{} `json:"value"`
}

// AnalyticsResult represents the result of an analytics query
type AnalyticsResult struct {
	Query         AnalyticsQuery          `json:"query"`
	TimeSeries    []TimeSeries            `json:"time_series"`
	Aggregated    map[string]float64      `json:"aggregated,omitempty"`
	Groups        map[string][]TimeSeries `json:"groups,omitempty"`
	Metadata      map[string]interface{}  `json:"metadata,omitempty"`
	ExecutionTime time.Duration           `json:"execution_time"`
}

// AnalyticsEngine provides advanced analytics capabilities
type AnalyticsEngine struct {
	metrics    []AnalyticsMetric
	timeSeries map[string]*TimeSeries
	contextMgr *enhancedContext.ContextManager
	verifier   enhancedContext.VerifierInterface
	mu         sync.RWMutex
	config     AnalyticsConfig
	processors []MetricProcessor
}

// AnalyticsConfig holds configuration for analytics engine
type AnalyticsConfig struct {
	RetentionPeriod   time.Duration          `yaml:"retention_period"`
	MaxTimeSeriesSize int                    `yaml:"max_time_series_size"`
	BatchSize         int                    `yaml:"batch_size"`
	FlushInterval     time.Duration          `yaml:"flush_interval"`
	EnablePredictions bool                   `yaml:"enable_predictions"`
	MLModelConfig     map[string]interface{} `yaml:"ml_model_config"`
}

// MetricProcessor interface for processing metrics
type MetricProcessor interface {
	Process(metric AnalyticsMetric) AnalyticsMetric
	GetName() string
}

// NewAnalyticsEngine creates a new analytics engine
func NewAnalyticsEngine(config AnalyticsConfig, contextMgr *enhancedContext.ContextManager, verifier enhancedContext.VerifierInterface) *AnalyticsEngine {
	return &AnalyticsEngine{
		metrics:    make([]AnalyticsMetric, 0),
		timeSeries: make(map[string]*TimeSeries),
		contextMgr: contextMgr,
		verifier:   verifier,
		config:     config,
		processors: make([]MetricProcessor, 0),
	}
}

// RecordMetric records a new analytics metric
func (ae *AnalyticsEngine) RecordMetric(name string, metricType MetricType, value float64, tags map[string]string, dimensions map[string]any) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	metric := AnalyticsMetric{
		Name:       name,
		Type:       metricType,
		Value:      value,
		Timestamp:  time.Now(),
		Tags:       tags,
		Dimensions: dimensions,
	}

	// Apply metric processors
	for _, processor := range ae.processors {
		metric = processor.Process(metric)
	}

	// Store metric
	ae.metrics = append(ae.metrics, metric)

	// Update time series
	if ts, exists := ae.timeSeries[name]; exists {
		ts.Metrics = append(ts.Metrics, value)
		ts.Timestamps = append(ts.Timestamps, metric.Timestamp)

		// Trim time series if it exceeds max size
		if len(ts.Metrics) > ae.config.MaxTimeSeriesSize {
			ts.Metrics = ts.Metrics[1:]
			ts.Timestamps = ts.Timestamps[1:]
		}
	} else {
		ae.timeSeries[name] = &TimeSeries{
			Name:       name,
			Metrics:    []float64{value},
			Timestamps: []time.Time{metric.Timestamp},
			Tags:       tags,
		}
	}

	return nil
}

// Query executes an analytics query
func (ae *AnalyticsEngine) Query(ctx context.Context, query AnalyticsQuery) (*AnalyticsResult, error) {
	start := time.Now()
	defer func() {
		fmt.Printf("Analytics query executed in %v\n", time.Since(start))
	}()

	ae.mu.RLock()
	defer ae.mu.RUnlock()

	result := &AnalyticsResult{
		Query:         query,
		TimeSeries:    make([]TimeSeries, 0),
		Aggregated:    make(map[string]float64),
		Groups:        make(map[string][]TimeSeries),
		ExecutionTime: time.Since(start),
	}

	// Filter metrics based on query
	filteredMetrics := ae.filterMetrics(query.MetricNames, query.Tags, query.QueryTimeRange)

	// Group metrics if needed
	if len(query.GroupBy) > 0 {
		groups := ae.groupMetrics(filteredMetrics, query.GroupBy)
		for groupKey, groupMetrics := range groups {
			ts := ae.metricsToTimeSeries(groupMetrics, query.MetricNames[0])
			result.Groups[groupKey] = append(result.Groups[groupKey], ts)
		}
	} else {
		// Create time series for each metric name
		for _, metricName := range query.MetricNames {
			metricList := ae.getMetricsByName(filteredMetrics, metricName)
			if len(metricList) > 0 {
				ts := ae.metricsToTimeSeries(metricList, metricName)
				result.TimeSeries = append(result.TimeSeries, ts)
			}
		}
	}

	// Apply aggregation
	if len(result.TimeSeries) > 0 {
		result.Aggregated = ae.aggregateMetrics(result.TimeSeries, query.Aggregation)
	}

	// Add metadata
	result.Metadata = map[string]interface{}{
		"metric_count":      len(filteredMetrics),
		"time_series_count": len(result.TimeSeries),
		"groups_count":      len(result.Groups),
		"retention_period":  ae.config.RetentionPeriod,
	}

	return result, nil
}

// GetMetricsSummary returns a summary of all metrics
func (ae *AnalyticsEngine) GetMetricsSummary(ctx context.Context) map[string]interface{} {
	ae.mu.RLock()
	defer ae.mu.RUnlock()

	summary := map[string]interface{}{
		"total_metrics":     len(ae.metrics),
		"time_series_count": len(ae.timeSeries),
		"processors_count":  len(ae.processors),
	}

	// Metric type distribution
	typeCount := make(map[MetricType]int)
	for _, metric := range ae.metrics {
		typeCount[metric.Type]++
	}
	summary["type_distribution"] = typeCount

	// Time series statistics
	tsStats := make(map[string]interface{})
	for name, ts := range ae.timeSeries {
		tsStats[name] = map[string]interface{}{
			"data_points": len(ts.Metrics),
			"oldest":      ts.Timestamps[0],
			"newest":      ts.Timestamps[len(ts.Timestamps)-1],
		}
	}
	summary["time_series_stats"] = tsStats

	// Recent activity
	recentCount := 0
	cutoff := time.Now().Add(-time.Hour)
	for _, metric := range ae.metrics {
		if metric.Timestamp.After(cutoff) {
			recentCount++
		}
	}
	summary["recent_metrics_hour"] = recentCount

	return summary
}

// PredictMetrics uses ML to predict future metric values
func (ae *AnalyticsEngine) PredictMetrics(ctx context.Context, metricName string, horizon time.Duration) (*AnalyticsResult, error) {
	if !ae.config.EnablePredictions {
		return nil, fmt.Errorf("predictions are disabled in configuration")
	}

	ae.mu.RLock()
	defer ae.mu.RUnlock()

	ts, exists := ae.timeSeries[metricName]
	if !exists {
		return nil, fmt.Errorf("time series not found: %s", metricName)
	}

	if len(ts.Metrics) < 10 {
		return nil, fmt.Errorf("insufficient data for prediction (need at least 10 data points, have %d)", len(ts.Metrics))
	}

	// Simple linear regression prediction
	prediction := ae.linearRegressionPredict(ts, horizon)

	result := &AnalyticsResult{
		Query: AnalyticsQuery{
			MetricNames: []string{metricName},
		},
		TimeSeries: []TimeSeries{
			{
				Name:    metricName + "_predicted",
				Metrics: prediction.Values,
				Tags:    map[string]string{"type": "prediction"},
			},
		},
		Metadata: map[string]interface{}{
			"prediction_horizon": horizon,
			"model_type":         "linear_regression",
			"accuracy_score":     prediction.Accuracy,
		},
	}

	return result, nil
}

// AddProcessor adds a metric processor
func (ae *AnalyticsEngine) AddProcessor(processor MetricProcessor) {
	ae.mu.Lock()
	defer ae.mu.Unlock()
	ae.processors = append(ae.processors, processor)
}

// Cleanup removes old metrics beyond retention period
func (ae *AnalyticsEngine) Cleanup(ctx context.Context) error {
	ae.mu.Lock()
	defer ae.mu.Unlock()

	cutoff := time.Now().Add(-ae.config.RetentionPeriod)

	// Clean up metrics
	var filtered []AnalyticsMetric
	for _, metric := range ae.metrics {
		if metric.Timestamp.After(cutoff) {
			filtered = append(filtered, metric)
		}
	}
	ae.metrics = filtered

	// Clean up time series
	for name, ts := range ae.timeSeries {
		var filteredMetrics []float64
		var filteredTimestamps []time.Time

		for i, timestamp := range ts.Timestamps {
			if timestamp.After(cutoff) {
				filteredMetrics = append(filteredMetrics, ts.Metrics[i])
				filteredTimestamps = append(filteredTimestamps, timestamp)
			}
		}

		if len(filteredMetrics) > 0 {
			ts.Metrics = filteredMetrics
			ts.Timestamps = filteredTimestamps
		} else {
			delete(ae.timeSeries, name)
		}
	}

	return nil
}

// filterMetrics filters metrics based on query criteria
func (ae *AnalyticsEngine) filterMetrics(metricNames []string, tags map[string]string, timeRange QueryTimeRange) []AnalyticsMetric {
	var filtered []AnalyticsMetric

	for _, metric := range ae.metrics {
		// Filter by metric name
		if len(metricNames) > 0 {
			found := false
			for _, name := range metricNames {
				if metric.Name == name {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Filter by tags
		if len(tags) > 0 {
			allMatch := true
			for tagKey, tagValue := range tags {
				if metricValue, exists := metric.Tags[tagKey]; !exists || metricValue != tagValue {
					allMatch = false
					break
				}
			}
			if !allMatch {
				continue
			}
		}

		// Filter by time range
		if !timeRange.From.IsZero() && metric.Timestamp.Before(timeRange.From) {
			continue
		}
		if !timeRange.To.IsZero() && metric.Timestamp.After(timeRange.To) {
			continue
		}

		filtered = append(filtered, metric)
	}

	return filtered
}

// groupMetrics groups metrics by specified fields
func (ae *AnalyticsEngine) groupMetrics(metrics []AnalyticsMetric, groupBy []string) map[string][]AnalyticsMetric {
	groups := make(map[string][]AnalyticsMetric)

	for _, metric := range metrics {
		groupKey := ""
		for i, field := range groupBy {
			if i > 0 {
				groupKey += "|"
			}

			switch field {
			case "name":
				groupKey += metric.Name
			case "type":
				groupKey += fmt.Sprintf("%d", metric.Type)
			default:
				if tagValue, exists := metric.Tags[field]; exists {
					groupKey += tagValue
				} else {
					groupKey += "unknown"
				}
			}
		}

		groups[groupKey] = append(groups[groupKey], metric)
	}

	return groups
}

// metricsToTimeSeries converts metrics to time series
func (ae *AnalyticsEngine) metricsToTimeSeries(metrics []AnalyticsMetric, metricName string) TimeSeries {
	ts := TimeSeries{
		Name:       metricName,
		Metrics:    make([]float64, len(metrics)),
		Timestamps: make([]time.Time, len(metrics)),
		Tags:       make(map[string]string),
	}

	if len(metrics) > 0 {
		ts.Tags = metrics[0].Tags
	}

	for i, metric := range metrics {
		ts.Metrics[i] = metric.Value
		ts.Timestamps[i] = metric.Timestamp
	}

	return ts
}

// getMetricsByName filters metrics by name
func (ae *AnalyticsEngine) getMetricsByName(metrics []AnalyticsMetric, name string) []AnalyticsMetric {
	var filtered []AnalyticsMetric
	for _, metric := range metrics {
		if metric.Name == name {
			filtered = append(filtered, metric)
		}
	}
	return filtered
}

// aggregateMetrics aggregates time series data
func (ae *AnalyticsEngine) aggregateMetrics(timeSeries []TimeSeries, aggregationType AggregationType) map[string]float64 {
	result := make(map[string]float64)

	for _, ts := range timeSeries {
		if len(ts.Metrics) == 0 {
			continue
		}

		var value float64
		switch aggregationType {
		case AggregationSum:
			for _, v := range ts.Metrics {
				value += v
			}
		case AggregationAvg:
			for _, v := range ts.Metrics {
				value += v
			}
			value /= float64(len(ts.Metrics))
		case AggregationMin:
			value = ts.Metrics[0]
			for _, v := range ts.Metrics {
				if v < value {
					value = v
				}
			}
		case AggregationMax:
			value = ts.Metrics[0]
			for _, v := range ts.Metrics {
				if v > value {
					value = v
				}
			}
		case AggregationCount:
			value = float64(len(ts.Metrics))
		default:
			value = ts.Metrics[len(ts.Metrics)-1] // Default to last value
		}

		result[ts.Name] = value
	}

	return result
}

// linearRegressionPredict performs simple linear regression prediction
func (ae *AnalyticsEngine) linearRegressionPredict(ts *TimeSeries, horizon time.Duration) struct {
	Values   []float64
	Accuracy float64
} {
	if len(ts.Metrics) < 2 {
		return struct {
			Values   []float64
			Accuracy float64
		}{Values: []float64{}, Accuracy: 0}
	}

	// Calculate linear regression parameters
	n := float64(len(ts.Metrics))
	var sumX, sumY, sumXY, sumX2 float64

	for i, value := range ts.Metrics {
		x := float64(i)
		sumX += x
		sumY += value
		sumXY += x * value
		sumX2 += x * x
	}

	// Calculate slope and intercept
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	intercept := (sumY - slope*sumX) / n

	// Calculate R-squared for accuracy
	var ssTotal, ssResidual float64
	meanY := sumY / n

	for i, value := range ts.Metrics {
		x := float64(i)
		predicted := slope*x + intercept
		ssTotal += math.Pow(value-meanY, 2)
		ssResidual += math.Pow(value-predicted, 2)
	}

	rSquared := 1 - (ssResidual / ssTotal)
	if rSquared < 0 {
		rSquared = 0
	}

	// Predict future values
	steps := int(horizon.Hours()) // Predict hourly
	predictions := make([]float64, steps)

	for i := 0; i < steps; i++ {
		x := float64(len(ts.Metrics) + i)
		predictions[i] = slope*x + intercept
	}

	return struct {
		Values   []float64
		Accuracy float64
	}{
		Values:   predictions,
		Accuracy: rSquared,
	}
}
