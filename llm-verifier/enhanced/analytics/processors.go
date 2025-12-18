package analytics

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// MetricProcessor implementations

// AnomalyDetector detects anomalies in metrics
type AnomalyDetector struct {
	threshold  float64
	windowSize int
	mu         sync.RWMutex
	historical []float64
}

func NewAnomalyDetector(threshold float64, windowSize int) *AnomalyDetector {
	return &AnomalyDetector{
		threshold:  threshold,
		windowSize: windowSize,
		historical: make([]float64, 0, windowSize),
	}
}

func (ad *AnomalyDetector) Process(metric AnalyticsMetric) AnalyticsMetric {
	ad.mu.Lock()
	defer ad.mu.Unlock()

	// Detect anomaly if we have enough previous data
	if len(ad.historical) >= ad.windowSize {
		mean, std := ad.calculateStats()
		zScore := math.Abs((metric.Value - mean) / std)

		if zScore > ad.threshold {
			if metric.Tags == nil {
				metric.Tags = make(map[string]string)
			}
			metric.Tags["anomaly"] = "true"
			metric.Tags["z_score"] = fmt.Sprintf("%.2f", zScore)
		}
	}

	// Add to historical data
	ad.historical = append(ad.historical, metric.Value)
	if len(ad.historical) > ad.windowSize {
		ad.historical = ad.historical[1:]
	}

	return metric
}

func (ad *AnomalyDetector) GetName() string {
	return "anomaly_detector"
}

func (ad *AnomalyDetector) calculateStats() (float64, float64) {
	n := float64(len(ad.historical))

	// Calculate mean
	sum := 0.0
	for _, value := range ad.historical {
		sum += value
	}
	mean := sum / n

	// Calculate standard deviation
	variance := 0.0
	for _, value := range ad.historical {
		variance += math.Pow(value-mean, 2)
	}
	std := math.Sqrt(variance / n)

	return mean, std
}

// RateCalculator calculates rates (per second/minute/hour)
type RateCalculator struct {
	counters map[string]*CounterData
	mu       sync.RWMutex
}

type CounterData struct {
	Count      int64
	LastUpdate time.Time
	Window     []time.Time
}

func NewRateCalculator() *RateCalculator {
	return &RateCalculator{
		counters: make(map[string]*CounterData),
	}
}

func (rc *RateCalculator) Process(metric AnalyticsMetric) AnalyticsMetric {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	counterName := metric.Name
	if tag, exists := metric.Tags["counter"]; exists {
		counterName = tag
	}

	counter, exists := rc.counters[counterName]
	if !exists {
		counter = &CounterData{
			Window: make([]time.Time, 0),
		}
		rc.counters[counterName] = counter
	}

	counter.Count++
	counter.LastUpdate = time.Now()

	// Add timestamp to window (1 hour window)
	counter.Window = append(counter.Window, metric.Timestamp)
	cutoff := metric.Timestamp.Add(-time.Hour)

	// Remove old timestamps
	var filtered []time.Time
	for _, ts := range counter.Window {
		if ts.After(cutoff) {
			filtered = append(filtered, ts)
		}
	}
	counter.Window = filtered

	// Calculate rate per hour
	ratePerHour := float64(len(counter.Window))
	metric.Value = ratePerHour

	if metric.Tags == nil {
		metric.Tags = make(map[string]string)
	}
	metric.Tags["rate_per_hour"] = fmt.Sprintf("%.2f", ratePerHour)
	metric.Tags["total_count"] = fmt.Sprintf("%d", counter.Count)

	return metric
}

func (rc *RateCalculator) GetName() string {
	return "rate_calculator"
}

// PercentileCalculator calculates percentiles over time windows
type PercentileCalculator struct {
	windows    map[string][]float64
	mu         sync.RWMutex
	windowSize int
}

func NewPercentileCalculator(windowSize int) *PercentileCalculator {
	return &PercentileCalculator{
		windows:    make(map[string][]float64),
		windowSize: windowSize,
	}
}

func (pc *PercentileCalculator) Process(metric AnalyticsMetric) AnalyticsMetric {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	windowName := metric.Name
	if tag, exists := metric.Tags["percentile_window"]; exists {
		windowName = tag
	}

	window := pc.windows[windowName]
	window = append(window, metric.Value)

	// Keep only the most recent values
	if len(window) > pc.windowSize {
		window = window[1:]
	}
	pc.windows[windowName] = window

	if len(window) > 0 {
		sorted := make([]float64, len(window))
		copy(sorted, window)
		sort.Float64s(sorted)

		if metric.Tags == nil {
			metric.Tags = make(map[string]string)
		}

		// Calculate various percentiles
		percentiles := []struct {
			name       string
			percentile float64
		}{
			{"p50", 0.5},
			{"p90", 0.9},
			{"p95", 0.95},
			{"p99", 0.99},
		}

		for _, p := range percentiles {
			index := int(float64(len(sorted)) * p.percentile)
			if index >= len(sorted) {
				index = len(sorted) - 1
			}
			metric.Tags[p.name] = fmt.Sprintf("%.2f", sorted[index])
		}
	}

	return metric
}

func (pc *PercentileCalculator) GetName() string {
	return "percentile_calculator"
}

// MovingAverageCalculator calculates moving averages
type MovingAverageCalculator struct {
	windows     map[string][]float64
	mu          sync.RWMutex
	windowSizes map[string]int
}

func NewMovingAverageCalculator() *MovingAverageCalculator {
	return &MovingAverageCalculator{
		windows:     make(map[string][]float64),
		windowSizes: make(map[string]int),
	}
}

func (mac *MovingAverageCalculator) SetWindowSize(metricName string, size int) {
	mac.mu.Lock()
	defer mac.mu.Unlock()
	mac.windowSizes[metricName] = size
}

func (mac *MovingAverageCalculator) Process(metric AnalyticsMetric) AnalyticsMetric {
	mac.mu.Lock()
	defer mac.mu.Unlock()

	windowName := metric.Name
	if tag, exists := metric.Tags["ma_window"]; exists {
		windowName = tag
	}

	windowSize, hasSize := mac.windowSizes[windowName]
	if !hasSize {
		windowSize = 10 // default window size
	}

	window := mac.windows[windowName]
	window = append(window, metric.Value)

	// Keep only the most recent values
	if len(window) > windowSize {
		window = window[1:]
	}
	mac.windows[windowName] = window

	// Calculate moving average
	if len(window) > 0 {
		sum := 0.0
		for _, value := range window {
			sum += value
		}
		avg := sum / float64(len(window))

		if metric.Tags == nil {
			metric.Tags = make(map[string]string)
		}
		metric.Tags["moving_average"] = fmt.Sprintf("%.2f", avg)
		metric.Tags["ma_window_size"] = fmt.Sprintf("%d", len(window))

		// Replace metric value with moving average
		metric.Value = avg
	}

	return metric
}

func (mac *MovingAverageCalculator) GetName() string {
	return "moving_average_calculator"
}

// DerivativeCalculator calculates rate of change (derivative)
type DerivativeCalculator struct {
	lastValues map[string]LastValue
	mu         sync.RWMutex
}

type LastValue struct {
	Value     float64
	Timestamp time.Time
}

func NewDerivativeCalculator() *DerivativeCalculator {
	return &DerivativeCalculator{
		lastValues: make(map[string]LastValue),
	}
}

func (dc *DerivativeCalculator) Process(metric AnalyticsMetric) AnalyticsMetric {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	metricName := metric.Name
	lastValue, exists := dc.lastValues[metricName]

	if exists {
		// Calculate derivative (rate of change)
		timeDiff := metric.Timestamp.Sub(lastValue.Timestamp).Seconds()
		if timeDiff > 0 {
			derivative := (metric.Value - lastValue.Value) / timeDiff

			if metric.Tags == nil {
				metric.Tags = make(map[string]string)
			}
			metric.Tags["derivative"] = fmt.Sprintf("%.6f", derivative)
			metric.Tags["time_diff_seconds"] = fmt.Sprintf("%.2f", timeDiff)

			// Replace metric value with derivative
			metric.Value = derivative
		}
	}

	// Store current value for next calculation
	dc.lastValues[metricName] = LastValue{
		Value:     metric.Value,
		Timestamp: metric.Timestamp,
	}

	return metric
}

func (dc *DerivativeCalculator) GetName() string {
	return "derivative_calculator"
}

// AlertProcessor triggers alerts based on thresholds
type AlertProcessor struct {
	thresholds   map[string]ThresholdConfig
	alertHandler AlertHandler
	mu           sync.RWMutex
}

type ThresholdConfig struct {
	Min     *float64 `json:"min,omitempty"`
	Max     *float64 `json:"max,omitempty"`
	Enabled bool     `json:"enabled"`
}

type AlertHandler interface {
	HandleAlert(metric AnalyticsMetric, threshold ThresholdConfig, violation string)
}

func NewAlertProcessor(alertHandler AlertHandler) *AlertProcessor {
	return &AlertProcessor{
		thresholds:   make(map[string]ThresholdConfig),
		alertHandler: alertHandler,
	}
}

func (ap *AlertProcessor) SetThreshold(metricName string, config ThresholdConfig) {
	ap.mu.Lock()
	defer ap.mu.Unlock()
	ap.thresholds[metricName] = config
}

func (ap *AlertProcessor) Process(metric AnalyticsMetric) AnalyticsMetric {
	ap.mu.RLock()
	defer ap.mu.RUnlock()

	config, exists := ap.thresholds[metric.Name]
	if !exists || !config.Enabled {
		return metric
	}

	violations := []string{}

	if config.Min != nil && metric.Value < *config.Min {
		violations = append(violations, fmt.Sprintf("below minimum %.2f", *config.Min))
	}

	if config.Max != nil && metric.Value > *config.Max {
		violations = append(violations, fmt.Sprintf("above maximum %.2f", *config.Max))
	}

	if len(violations) > 0 {
		if ap.alertHandler != nil {
			ap.alertHandler.HandleAlert(metric, config, violations[0])
		}

		if metric.Tags == nil {
			metric.Tags = make(map[string]string)
		}
		metric.Tags["alert"] = "true"
		metric.Tags["alert_violation"] = violations[0]
	}

	return metric
}

func (ap *AlertProcessor) GetName() string {
	return "alert_processor"
}

// DefaultAlertHandler implements a simple alert handler
type DefaultAlertHandler struct {
	alerts []Alert
	mu     sync.RWMutex
}

type Alert struct {
	Metric    AnalyticsMetric
	Threshold ThresholdConfig
	Violation string
	Timestamp time.Time
}

func NewDefaultAlertHandler() *DefaultAlertHandler {
	return &DefaultAlertHandler{
		alerts: make([]Alert, 0),
	}
}

func (dah *DefaultAlertHandler) HandleAlert(metric AnalyticsMetric, threshold ThresholdConfig, violation string) {
	dah.mu.Lock()
	defer dah.mu.Unlock()

	alert := Alert{
		Metric:    metric,
		Threshold: threshold,
		Violation: violation,
		Timestamp: time.Now(),
	}

	dah.alerts = append(dah.alerts, alert)

	// Log the alert (in real implementation, this would send to notification system)
	fmt.Printf("ALERT: %s - %s at %.2f (%s)\n", metric.Name, violation, metric.Value, metric.Timestamp.Format(time.RFC3339))
}

func (dah *DefaultAlertHandler) GetAlerts() []Alert {
	dah.mu.RLock()
	defer dah.mu.RUnlock()

	alerts := make([]Alert, len(dah.alerts))
	copy(alerts, dah.alerts)
	return alerts
}

func (dah *DefaultAlertHandler) ClearAlerts() {
	dah.mu.Lock()
	defer dah.mu.Unlock()
	dah.alerts = make([]Alert, 0)
}
