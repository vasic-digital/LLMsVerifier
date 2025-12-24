package monitoring

import (
	"sync"
	"time"
)

// MetricsTracker tracks real metrics throughout the application
type MetricsTracker struct {
	mu sync.RWMutex

	// Database metrics
	dbConnectionsInUse int
	dbConnectionsIdle  int
	dbConnectionsOpen  int
	dbQueryCount       int64
	dbQueryDuration    time.Duration
	dbErrorCount       int64

	// API metrics
	apiTotalRequests       int64
	apiActiveRequests      int64
	apiAverageResponseTime time.Duration
	apiErrorRate           float64
	apiEndpointStats       map[string]*EndpointStatsInternal

	// Verification metrics
	verActiveVerifications int
	verCompletedToday      int
	verFailedToday         int
	verAverageDuration     time.Duration
	verSuccessRate         float64
	verQueueLength         int

	// Brotli compression metrics
	brotliTestsPerformed    int64
	brotliSupportedModels   int64
	brotliDetectionDuration time.Duration
	brotliCacheHits         int64
	brotliCacheMisses       int64

	// Notifications metrics
	notifChannelsConfigured int
	notifMessagesSent       int
	notifDeliveryRate       float64

	startTime time.Time
}

// EndpointStatsInternal tracks internal stats for endpoints
type EndpointStatsInternal struct {
	mu            sync.RWMutex
	Requests      int64
	Errors        int64
	TotalDuration time.Duration
	LastRequest   time.Time
}

// NewMetricsTracker creates a new metrics tracker
func NewMetricsTracker() *MetricsTracker {
	return &MetricsTracker{
		apiEndpointStats: make(map[string]*EndpointStatsInternal),
		startTime:        time.Now(),
	}
}

// Database Stats Methods

func (mt *MetricsTracker) GetDatabaseStats() DatabaseStats {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	return DatabaseStats{
		ConnectionsInUse: mt.dbConnectionsInUse,
		ConnectionsIdle:  mt.dbConnectionsIdle,
		ConnectionsOpen:  mt.dbConnectionsOpen,
		QueryCount:       mt.dbQueryCount,
		QueryDuration:    mt.dbQueryDuration,
		ErrorCount:       mt.dbErrorCount,
	}
}

func (mt *MetricsTracker) UpdateDatabaseStats(inUse, idle, open int) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	mt.dbConnectionsInUse = inUse
	mt.dbConnectionsIdle = idle
	mt.dbConnectionsOpen = open
}

func (mt *MetricsTracker) RecordQuery(duration time.Duration) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	mt.dbQueryCount++

	// Update average duration
	if mt.dbQueryCount == 1 {
		mt.dbQueryDuration = duration
	} else {
		// Rolling average
		mt.dbQueryDuration = (mt.dbQueryDuration*time.Duration(mt.dbQueryCount-1) + duration) / time.Duration(mt.dbQueryCount)
	}
}

func (mt *MetricsTracker) RecordQueryError() {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	mt.dbErrorCount++
}

// API Stats Methods

func (mt *MetricsTracker) GetAPIMetrics() APIMetrics {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	// Build endpoint stats map
	endpointStats := make(map[string]EndpointStats)
	for path, stats := range mt.apiEndpointStats {
		stats.mu.RLock()
		avgResponse := time.Duration(0)
		if stats.Requests > 0 {
			avgResponse = stats.TotalDuration / time.Duration(stats.Requests)
		}

		endpointStats[path] = EndpointStats{
			Requests:        stats.Requests,
			Errors:          stats.Errors,
			AvgResponseTime: avgResponse,
			LastRequest:     stats.LastRequest,
		}
		stats.mu.RUnlock()
	}

	return APIMetrics{
		TotalRequests:       mt.apiTotalRequests,
		ActiveRequests:      mt.apiActiveRequests,
		AverageResponseTime: mt.apiAverageResponseTime,
		RequestRate:         mt.calculateRequestRate(),
		ErrorRate:           mt.apiErrorRate,
		EndpointStats:       endpointStats,
	}
}

func (mt *MetricsTracker) RecordAPIRequest(endpoint string) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	mt.apiTotalRequests++
	mt.apiActiveRequests++
	mt.startTime = time.Now() // Reset for rate calculation

	// Track endpoint stats
	stats, exists := mt.apiEndpointStats[endpoint]
	if !exists {
		stats = &EndpointStatsInternal{}
		mt.apiEndpointStats[endpoint] = stats
	}

	stats.mu.Lock()
	stats.Requests++
	stats.LastRequest = time.Now()
	stats.mu.Unlock()
}

func (mt *MetricsTracker) RecordAPIResponse(endpoint string, duration time.Duration) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	if mt.apiActiveRequests > 0 {
		mt.apiActiveRequests--
	}

	// Update average response time
	if mt.apiTotalRequests == 1 {
		mt.apiAverageResponseTime = duration
	} else {
		mt.apiAverageResponseTime = (mt.apiAverageResponseTime*time.Duration(mt.apiTotalRequests-1) + duration) / time.Duration(mt.apiTotalRequests)
	}

	// Track endpoint stats
	if stats, exists := mt.apiEndpointStats[endpoint]; exists {
		stats.mu.Lock()
		stats.TotalDuration += duration
		stats.mu.Unlock()
	}
}

func (mt *MetricsTracker) RecordAPIError(endpoint string) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	// Update error rate
	if mt.apiTotalRequests > 0 {
		totalErrors := float64(mt.calculateTotalErrors() + 1)
		mt.apiErrorRate = totalErrors / float64(mt.apiTotalRequests)
	}

	// Track endpoint stats
	if stats, exists := mt.apiEndpointStats[endpoint]; exists {
		stats.mu.Lock()
		stats.Errors++
		stats.mu.Unlock()
	}
}

func (mt *MetricsTracker) calculateRequestRate() float64 {
	elapsed := time.Since(mt.startTime).Seconds()
	if elapsed > 0 {
		return float64(mt.apiTotalRequests) / elapsed
	}
	return 0
}

func (mt *MetricsTracker) calculateTotalErrors() int64 {
	total := int64(0)
	for _, stats := range mt.apiEndpointStats {
		total += stats.Errors
	}
	return total
}

// Verification Stats Methods

func (mt *MetricsTracker) GetVerificationStats() VerificationStats {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	return VerificationStats{
		ActiveVerifications: mt.verActiveVerifications,
		CompletedToday:      mt.verCompletedToday,
		FailedToday:         mt.verFailedToday,
		AverageDuration:     mt.verAverageDuration,
		SuccessRate:         mt.verSuccessRate,
		QueueLength:         mt.verQueueLength,
	}
}

func (mt *MetricsTracker) RecordVerificationStarted() {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	mt.verActiveVerifications++
	if mt.verQueueLength > 0 {
		mt.verQueueLength--
	}
}

func (mt *MetricsTracker) RecordVerificationCompleted(success bool, duration time.Duration) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	mt.verActiveVerifications--
	mt.verCompletedToday++

	if !success {
		mt.verFailedToday++
	}

	// Update average duration
	totalCompleted := mt.verCompletedToday
	if totalCompleted == 1 {
		mt.verAverageDuration = duration
	} else {
		mt.verAverageDuration = (mt.verAverageDuration*time.Duration(totalCompleted-1) + duration) / time.Duration(totalCompleted)
	}

	// Update success rate
	if totalCompleted > 0 {
		mt.verSuccessRate = float64(totalCompleted-mt.verFailedToday) / float64(totalCompleted)
	}
}

func (mt *MetricsTracker) QueueVerification() {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.verQueueLength++
}

// Brotli Compression Stats Methods

func (mt *MetricsTracker) RecordBrotliTest(supportsBrotli bool, duration time.Duration) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	mt.brotliTestsPerformed++
	if supportsBrotli {
		mt.brotliSupportedModels++
	}

	// Update average detection duration
	if mt.brotliTestsPerformed == 1 {
		mt.brotliDetectionDuration = duration
	} else {
		mt.brotliDetectionDuration = (mt.brotliDetectionDuration*time.Duration(mt.brotliTestsPerformed-1) + duration) / time.Duration(mt.brotliTestsPerformed)
	}
}

func (mt *MetricsTracker) RecordBrotliCacheHit() {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.brotliCacheHits++
}

func (mt *MetricsTracker) RecordBrotliCacheMiss() {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.brotliCacheMisses++
}

func (mt *MetricsTracker) GetBrotliMetrics() map[string]interface{} {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	totalTests := mt.brotliTestsPerformed
	cacheHitRate := 0.0
	if totalTests > 0 {
		cacheHitRate = float64(mt.brotliCacheHits) / float64(mt.brotliCacheHits+mt.brotliCacheMisses) * 100
	}

	supportRate := 0.0
	if totalTests > 0 {
		supportRate = float64(mt.brotliSupportedModels) / float64(totalTests) * 100
	}

	return map[string]interface{}{
		"tests_performed":        totalTests,
		"supported_models":       mt.brotliSupportedModels,
		"support_rate_percent":   supportRate,
		"avg_detection_duration": mt.brotliDetectionDuration.String(),
		"cache_hits":             mt.brotliCacheHits,
		"cache_misses":           mt.brotliCacheMisses,
		"cache_hit_rate":         cacheHitRate,
	}
}

// Notification Stats Methods

func (mt *MetricsTracker) GetNotificationStats() map[string]interface{} {
	mt.mu.RLock()
	defer mt.mu.RUnlock()

	return map[string]interface{}{
		"channels_configured": mt.notifChannelsConfigured,
		"messages_sent":       mt.notifMessagesSent,
		"delivery_rate":       mt.notifDeliveryRate,
	}
}

func (mt *MetricsTracker) SetNotificationChannels(count int) {
	mt.mu.Lock()
	defer mt.mu.Unlock()
	mt.notifChannelsConfigured = count
}

func (mt *MetricsTracker) RecordNotificationSent(delivered bool) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	mt.notifMessagesSent++

	// Update delivery rate
	totalSent := mt.notifMessagesSent
	if totalSent > 0 {
		deliveredCount := float64(totalSent)
		if delivered {
			deliveredCount = float64(mt.notifMessagesSent-1) + 1
		}
		mt.notifDeliveryRate = deliveredCount / float64(totalSent)
	}
}
