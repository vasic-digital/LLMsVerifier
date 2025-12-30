package monitoring

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"llm-verifier/database"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// ComponentHealth represents the health of a system component
type ComponentHealth struct {
	Name         string                 `json:"name"`
	Status       HealthStatus           `json:"status"`
	Message      string                 `json:"message,omitempty"`
	LastChecked  time.Time              `json:"last_checked"`
	ResponseTime time.Duration          `json:"response_time,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// SystemMetrics represents system performance metrics
type SystemMetrics struct {
	Timestamp         time.Time         `json:"timestamp"`
	Uptime            time.Duration     `json:"uptime"`
	MemoryUsage       MemoryStats       `json:"memory_usage"`
	Goroutines        int               `json:"goroutines"`
	DatabaseStats     DatabaseStats     `json:"database_stats"`
	APIMetrics        APIMetrics        `json:"api_metrics"`
	VerificationStats VerificationStats `json:"verification_stats"`
	BrotliMetrics     BrotliMetrics     `json:"brotli_metrics"`
}

// BrotliMetrics represents Brotli compression performance metrics
type BrotliMetrics struct {
	TestsPerformed     int64         `json:"tests_performed"`
	SupportedModels    int64         `json:"supported_models"`
	SupportRatePercent float64       `json:"support_rate_percent"`
	AvgDetectionTime   time.Duration `json:"avg_detection_time"`
	CacheHits          int64         `json:"cache_hits"`
	CacheMisses        int64         `json:"cache_misses"`
	CacheHitRate       float64       `json:"cache_hit_rate"`
}

// MemoryStats represents memory usage statistics
type MemoryStats struct {
	Alloc         uint64  `json:"alloc_bytes"`
	TotalAlloc    uint64  `json:"total_alloc_bytes"`
	Sys           uint64  `json:"sys_bytes"`
	Lookups       uint64  `json:"lookups"`
	Mallocs       uint64  `json:"mallocs"`
	Frees         uint64  `json:"frees"`
	HeapAlloc     uint64  `json:"heap_alloc_bytes"`
	HeapSys       uint64  `json:"heap_sys_bytes"`
	HeapIdle      uint64  `json:"heap_idle_bytes"`
	HeapInuse     uint64  `json:"heap_inuse_bytes"`
	HeapReleased  uint64  `json:"heap_released_bytes"`
	HeapObjects   uint64  `json:"heap_objects"`
	StackInuse    uint64  `json:"stack_inuse_bytes"`
	StackSys      uint64  `json:"stack_sys_bytes"`
	GCSys         uint64  `json:"gc_sys_bytes"`
	NextGC        uint64  `json:"next_gc_bytes"`
	LastGC        uint64  `json:"last_gc_timestamp"`
	NumGC         uint32  `json:"num_gc"`
	NumForcedGC   uint32  `json:"num_forced_gc"`
	GCCPUFraction float64 `json:"gc_cpu_fraction"`
}

// DatabaseStats represents database performance statistics
type DatabaseStats struct {
	ConnectionsInUse int           `json:"connections_in_use"`
	ConnectionsIdle  int           `json:"connections_idle"`
	ConnectionsOpen  int           `json:"connections_open"`
	QueryCount       int64         `json:"query_count"`
	QueryDuration    time.Duration `json:"query_duration_avg"`
	ErrorCount       int64         `json:"error_count"`
}

// APIMetrics represents API performance metrics
type APIMetrics struct {
	TotalRequests       int64                    `json:"total_requests"`
	ActiveRequests      int64                    `json:"active_requests"`
	AverageResponseTime time.Duration            `json:"average_response_time"`
	RequestRate         float64                  `json:"requests_per_second"`
	ErrorRate           float64                  `json:"error_rate"`
	EndpointStats       map[string]EndpointStats `json:"endpoint_stats"`
}

// EndpointStats represents statistics for a specific API endpoint
type EndpointStats struct {
	Requests        int64         `json:"requests"`
	Errors          int64         `json:"errors"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
	LastRequest     time.Time     `json:"last_request"`
}

// VerificationStats represents verification performance statistics
type VerificationStats struct {
	ActiveVerifications int           `json:"active_verifications"`
	CompletedToday      int           `json:"completed_today"`
	FailedToday         int           `json:"failed_today"`
	AverageDuration     time.Duration `json:"average_duration"`
	SuccessRate         float64       `json:"success_rate"`
	QueueLength         int           `json:"queue_length"`
}

// HealthChecker manages system health monitoring
type HealthChecker struct {
	database       *database.Database
	components     map[string]*ComponentHealth
	systemMetrics  *SystemMetrics
	metricsTracker *MetricsTracker
	startTime      time.Time
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *database.Database) *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())

	hc := &HealthChecker{
		database:       db,
		components:     make(map[string]*ComponentHealth),
		systemMetrics:  &SystemMetrics{},
		metricsTracker: NewMetricsTracker(),
		startTime:      time.Now(),
		ctx:            ctx,
		cancel:         cancel,
	}

	// Initialize default components
	hc.initializeComponents()

	return hc
}

// Start begins health monitoring
func (hc *HealthChecker) Start(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-hc.ctx.Done():
				return
			case <-ticker.C:
				hc.checkAllComponents()
				hc.updateSystemMetrics()
			}
		}
	}()

	log.Printf("Health monitoring started with %v interval", interval)
}

// Stop stops health monitoring
func (hc *HealthChecker) Stop() {
	hc.cancel()
	log.Println("Health monitoring stopped")
}

// GetHealthStatus returns the overall health status
func (hc *HealthChecker) GetHealthStatus() HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	// If any critical component is unhealthy, system is unhealthy
	for _, component := range hc.components {
		if component.Status == HealthStatusUnhealthy {
			return HealthStatusUnhealthy
		}
	}

	// If any component is degraded, system is degraded
	for _, component := range hc.components {
		if component.Status == HealthStatusDegraded {
			return HealthStatusDegraded
		}
	}

	return HealthStatusHealthy
}

// GetComponentHealth returns health status for all components
func (hc *HealthChecker) GetComponentHealth() map[string]*ComponentHealth {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	components := make(map[string]*ComponentHealth)
	for name, component := range hc.components {
		components[name] = &ComponentHealth{
			Name:         component.Name,
			Status:       component.Status,
			Message:      component.Message,
			LastChecked:  component.LastChecked,
			ResponseTime: component.ResponseTime,
			Details:      component.Details,
		}
	}

	return components
}

// parseDuration parses a duration string safely
func parseDuration(durationStr string) time.Duration {
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return 0
	}
	return duration
}

// GetSystemMetrics returns current system metrics
func (hc *HealthChecker) GetSystemMetrics() *SystemMetrics {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	// Return a copy to avoid race conditions
	return &SystemMetrics{
		Timestamp:         hc.systemMetrics.Timestamp,
		Uptime:            time.Since(hc.startTime),
		MemoryUsage:       hc.systemMetrics.MemoryUsage,
		Goroutines:        hc.systemMetrics.Goroutines,
		DatabaseStats:     hc.systemMetrics.DatabaseStats,
		APIMetrics:        hc.systemMetrics.APIMetrics,
		VerificationStats: hc.systemMetrics.VerificationStats,
	}
}

// initializeComponents sets up default health check components
func (hc *HealthChecker) initializeComponents() {
	hc.components["database"] = &ComponentHealth{
		Name:        "Database",
		Status:      HealthStatusHealthy,
		LastChecked: time.Now(),
	}

	hc.components["api"] = &ComponentHealth{
		Name:        "API Server",
		Status:      HealthStatusHealthy,
		LastChecked: time.Now(),
	}

	hc.components["scheduler"] = &ComponentHealth{
		Name:        "Job Scheduler",
		Status:      HealthStatusHealthy,
		LastChecked: time.Now(),
	}

	hc.components["notifications"] = &ComponentHealth{
		Name:        "Notification System",
		Status:      HealthStatusHealthy,
		LastChecked: time.Now(),
	}
}

// checkAllComponents performs health checks on all components
func (hc *HealthChecker) checkAllComponents() {
	for name := range hc.components {
		switch name {
		case "database":
			hc.checkDatabaseHealth()
		case "api":
			hc.checkAPIHealth()
		case "scheduler":
			hc.checkSchedulerHealth()
		case "notifications":
			hc.checkNotificationsHealth()
		}
	}
}

// checkDatabaseHealth checks database connectivity and performance
func (hc *HealthChecker) checkDatabaseHealth() {
	start := time.Now()

	// Check for nil database to prevent panic
	if hc.database == nil {
		hc.mu.Lock()
		defer hc.mu.Unlock()

		component := hc.components["database"]
		component.LastChecked = time.Now()
		component.Status = HealthStatusUnhealthy
		component.Message = "Database is not configured"
		component.ResponseTime = 0
		component.Details = map[string]interface{}{
			"error": "database is nil",
		}
		return
	}

	// Simple query to test database connectivity
	_, err := hc.database.ListModels(map[string]interface{}{})

	responseTime := time.Since(start)

	hc.mu.Lock()
	defer hc.mu.Unlock()

	component := hc.components["database"]
	component.LastChecked = time.Now()
	component.ResponseTime = responseTime

	if err != nil {
		component.Status = HealthStatusUnhealthy
		component.Message = fmt.Sprintf("Database connection failed: %v", err)
	} else if responseTime > 5*time.Second {
		component.Status = HealthStatusDegraded
		component.Message = fmt.Sprintf("Database response slow: %v", responseTime)
	} else {
		component.Status = HealthStatusHealthy
		component.Message = "Database is healthy"
	}

	component.Details = map[string]interface{}{
		"response_time_ms": responseTime.Milliseconds(),
		"error":            err != nil,
	}
}

// checkAPIHealth checks API server health using real metrics
func (hc *HealthChecker) checkAPIHealth() {
	// Get real API metrics from the tracker
	apiMetrics := hc.metricsTracker.GetAPIMetrics()

	hc.mu.Lock()
	defer hc.mu.Unlock()

	component := hc.components["api"]
	component.LastChecked = time.Now()
	component.ResponseTime = apiMetrics.AverageResponseTime

	// Determine health status based on real metrics
	if apiMetrics.ErrorRate > 0.5 { // More than 50% errors
		component.Status = HealthStatusUnhealthy
		component.Message = fmt.Sprintf("API error rate critical: %.1f%%", apiMetrics.ErrorRate*100)
	} else if apiMetrics.ErrorRate > 0.1 { // More than 10% errors
		component.Status = HealthStatusDegraded
		component.Message = fmt.Sprintf("API error rate elevated: %.1f%%", apiMetrics.ErrorRate*100)
	} else if apiMetrics.AverageResponseTime > 5*time.Second {
		component.Status = HealthStatusDegraded
		component.Message = fmt.Sprintf("API response slow: %v", apiMetrics.AverageResponseTime)
	} else {
		component.Status = HealthStatusHealthy
		component.Message = "API server is responding normally"
	}

	component.Details = map[string]interface{}{
		"total_requests":       apiMetrics.TotalRequests,
		"active_requests":      apiMetrics.ActiveRequests,
		"error_rate_percent":   apiMetrics.ErrorRate * 100,
		"requests_per_second":  apiMetrics.RequestRate,
		"avg_response_time_ms": apiMetrics.AverageResponseTime.Milliseconds(),
		"endpoints_tracked":    len(apiMetrics.EndpointStats),
	}
}

// checkSchedulerHealth checks job scheduler health using real metrics
func (hc *HealthChecker) checkSchedulerHealth() {
	// Get real scheduler metrics from the tracker
	schedStats := hc.metricsTracker.GetSchedulerStats()

	hc.mu.Lock()
	defer hc.mu.Unlock()

	component := hc.components["scheduler"]
	component.LastChecked = time.Now()

	// Calculate failure rate
	totalJobs := schedStats.CompletedJobs + schedStats.FailedJobs
	failureRate := float64(0)
	if totalJobs > 0 {
		failureRate = float64(schedStats.FailedJobs) / float64(totalJobs)
	}

	// Determine health status based on real metrics
	if !schedStats.IsRunning {
		component.Status = HealthStatusUnhealthy
		component.Message = "Scheduler is not running"
		component.ResponseTime = 0
	} else if failureRate > 0.5 { // More than 50% failures
		component.Status = HealthStatusUnhealthy
		component.Message = fmt.Sprintf("Scheduler failure rate critical: %.1f%%", failureRate*100)
		component.ResponseTime = time.Millisecond * 100
	} else if failureRate > 0.1 { // More than 10% failures
		component.Status = HealthStatusDegraded
		component.Message = fmt.Sprintf("Scheduler failure rate elevated: %.1f%%", failureRate*100)
		component.ResponseTime = time.Millisecond * 50
	} else if schedStats.QueuedJobs > 100 { // Large queue backlog
		component.Status = HealthStatusDegraded
		component.Message = fmt.Sprintf("Scheduler queue backlog: %d jobs", schedStats.QueuedJobs)
		component.ResponseTime = time.Millisecond * 20
	} else {
		component.Status = HealthStatusHealthy
		component.Message = "Scheduler is running normally"
		component.ResponseTime = time.Millisecond * 5
	}

	component.Details = map[string]interface{}{
		"is_running":      schedStats.IsRunning,
		"active_jobs":     schedStats.ActiveJobs,
		"queued_jobs":     schedStats.QueuedJobs,
		"completed_jobs":  schedStats.CompletedJobs,
		"failed_jobs":     schedStats.FailedJobs,
		"failure_rate":    failureRate * 100,
		"last_check_time": schedStats.LastCheckTime,
	}
}

// checkNotificationsHealth checks notification system health (placeholder)
func (hc *HealthChecker) checkNotificationsHealth() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	component := hc.components["notifications"]
	component.LastChecked = time.Now()
	component.Status = HealthStatusHealthy
	component.Message = "Notification system is operational"
	component.ResponseTime = time.Millisecond * 8

	component.Details = hc.metricsTracker.GetNotificationStats()
}

// updateSystemMetrics collects current system metrics
func (hc *HealthChecker) updateSystemMetrics() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.systemMetrics.Timestamp = time.Now()
	hc.systemMetrics.Goroutines = runtime.NumGoroutine()

	// Memory stats
	hc.systemMetrics.MemoryUsage = MemoryStats{
		Alloc:         m.Alloc,
		TotalAlloc:    m.TotalAlloc,
		Sys:           m.Sys,
		Lookups:       m.Lookups,
		Mallocs:       m.Mallocs,
		Frees:         m.Frees,
		HeapAlloc:     m.HeapAlloc,
		HeapSys:       m.HeapSys,
		HeapIdle:      m.HeapIdle,
		HeapInuse:     m.HeapInuse,
		HeapReleased:  m.HeapReleased,
		HeapObjects:   m.HeapObjects,
		StackInuse:    m.StackInuse,
		StackSys:      m.StackSys,
		GCSys:         m.GCSys,
		NextGC:        m.NextGC,
		LastGC:        m.LastGC,
		NumGC:         m.NumGC,
		NumForcedGC:   m.NumForcedGC,
		GCCPUFraction: m.GCCPUFraction,
	}

	// Database stats - get from tracker
	hc.systemMetrics.DatabaseStats = hc.metricsTracker.GetDatabaseStats()

	// API metrics - get from tracker
	hc.systemMetrics.APIMetrics = hc.metricsTracker.GetAPIMetrics()

	// Verification stats - get from tracker
	hc.systemMetrics.VerificationStats = hc.metricsTracker.GetVerificationStats()

	// Brotli metrics - get from tracker
	brotliStats := hc.metricsTracker.GetBrotliMetrics()
	hc.systemMetrics.BrotliMetrics = BrotliMetrics{
		TestsPerformed:     brotliStats["tests_performed"].(int64),
		SupportedModels:    brotliStats["supported_models"].(int64),
		SupportRatePercent: brotliStats["support_rate_percent"].(float64),
		AvgDetectionTime:   parseDuration(brotliStats["avg_detection_duration"].(string)),
		CacheHits:          brotliStats["cache_hits"].(int64),
		CacheMisses:        brotliStats["cache_misses"].(int64),
		CacheHitRate:       brotliStats["cache_hit_rate"].(float64),
	}
}

// RegisterHealthEndpoints registers health check endpoints with the API server
func (hc *HealthChecker) RegisterHealthEndpoints(router *gin.Engine) {
	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		status := hc.GetHealthStatus()

		response := gin.H{
			"status":    status,
			"timestamp": time.Now(),
			"version":   "1.0.0",
			"uptime":    time.Since(hc.startTime).String(),
		}

		httpStatus := http.StatusOK
		if status == HealthStatusDegraded {
			httpStatus = http.StatusOK // Still return 200 for degraded
		} else if status == HealthStatusUnhealthy {
			httpStatus = http.StatusServiceUnavailable
		}

		c.JSON(httpStatus, response)
	})

	// Detailed health check endpoint
	router.GET("/health/detailed", func(c *gin.Context) {
		status := hc.GetHealthStatus()
		components := hc.GetComponentHealth()
		metrics := hc.GetSystemMetrics()

		response := gin.H{
			"status":     status,
			"timestamp":  time.Now(),
			"uptime":     time.Since(hc.startTime).String(),
			"components": components,
			"metrics":    metrics,
		}

		c.JSON(http.StatusOK, response)
	})

	// Readiness check endpoint
	router.GET("/health/ready", func(c *gin.Context) {
		status := hc.GetHealthStatus()

		// For readiness, we want to ensure critical components are healthy
		ready := status != HealthStatusUnhealthy

		response := gin.H{
			"ready":     ready,
			"status":    status,
			"timestamp": time.Now(),
		}

		httpStatus := http.StatusOK
		if !ready {
			httpStatus = http.StatusServiceUnavailable
		}

		c.JSON(httpStatus, response)
	})

	// Liveness check endpoint
	router.GET("/health/live", func(c *gin.Context) {
		response := gin.H{
			"alive":     true,
			"timestamp": time.Now(),
			"uptime":    time.Since(hc.startTime).String(),
		}

		c.JSON(http.StatusOK, response)
	})

	// Metrics endpoint (Prometheus format)
	router.GET("/metrics", func(c *gin.Context) {
		metrics := hc.GetSystemMetrics()

		// Generate Prometheus metrics format
		var output string

		// Memory metrics
		output += fmt.Sprintf("# HELP llm_verifier_memory_usage_bytes Current memory usage in bytes\n")
		output += fmt.Sprintf("# TYPE llm_verifier_memory_usage_bytes gauge\n")
		output += fmt.Sprintf("llm_verifier_memory_usage_bytes{type=\"alloc\"} %d\n", metrics.MemoryUsage.Alloc)
		output += fmt.Sprintf("llm_verifier_memory_usage_bytes{type=\"heap\"} %d\n", metrics.MemoryUsage.HeapAlloc)
		output += fmt.Sprintf("llm_verifier_memory_usage_bytes{type=\"sys\"} %d\n", metrics.MemoryUsage.Sys)

		// Goroutines metric
		output += fmt.Sprintf("# HELP llm_verifier_goroutines_number Current number of goroutines\n")
		output += fmt.Sprintf("# TYPE llm_verifier_goroutines_number gauge\n")
		output += fmt.Sprintf("llm_verifier_goroutines_number %d\n", metrics.Goroutines)

		// API metrics
		output += fmt.Sprintf("# HELP llm_verifier_api_requests_total Total number of API requests\n")
		output += fmt.Sprintf("# TYPE llm_verifier_api_requests_total counter\n")
		output += fmt.Sprintf("llm_verifier_api_requests_total %d\n", metrics.APIMetrics.TotalRequests)

		output += fmt.Sprintf("# HELP llm_verifier_api_response_time_seconds Average API response time in seconds\n")
		output += fmt.Sprintf("# TYPE llm_verifier_api_response_time_seconds gauge\n")
		output += fmt.Sprintf("llm_verifier_api_response_time_seconds %f\n", metrics.APIMetrics.AverageResponseTime.Seconds())

		// Brotli metrics
		output += fmt.Sprintf("# HELP llm_verifier_brotli_tests_performed Total number of Brotli compression tests performed\n")
		output += fmt.Sprintf("# TYPE llm_verifier_brotli_tests_performed counter\n")
		output += fmt.Sprintf("llm_verifier_brotli_tests_performed %d\n", metrics.BrotliMetrics.TestsPerformed)

		output += fmt.Sprintf("# HELP llm_verifier_brotli_supported_models Number of models supporting Brotli compression\n")
		output += fmt.Sprintf("# TYPE llm_verifier_brotli_supported_models gauge\n")
		output += fmt.Sprintf("llm_verifier_brotli_supported_models %d\n", metrics.BrotliMetrics.SupportedModels)

		output += fmt.Sprintf("# HELP llm_verifier_brotli_support_rate_percent Percentage of models supporting Brotli compression\n")
		output += fmt.Sprintf("# TYPE llm_verifier_brotli_support_rate_percent gauge\n")
		output += fmt.Sprintf("llm_verifier_brotli_support_rate_percent %.2f\n", metrics.BrotliMetrics.SupportRatePercent)

		output += fmt.Sprintf("# HELP llm_verifier_brotli_cache_hits Number of Brotli cache hits\n")
		output += fmt.Sprintf("# TYPE llm_verifier_brotli_cache_hits counter\n")
		output += fmt.Sprintf("llm_verifier_brotli_cache_hits %d\n", metrics.BrotliMetrics.CacheHits)

		output += fmt.Sprintf("# HELP llm_verifier_brotli_cache_misses Number of Brotli cache misses\n")
		output += fmt.Sprintf("# TYPE llm_verifier_brotli_cache_misses counter\n")
		output += fmt.Sprintf("llm_verifier_brotli_cache_misses %d\n", metrics.BrotliMetrics.CacheMisses)

		output += fmt.Sprintf("# HELP llm_verifier_brotli_cache_hit_rate Brotli cache hit rate percentage\n")
		output += fmt.Sprintf("# TYPE llm_verifier_brotli_cache_hit_rate gauge\n")
		output += fmt.Sprintf("llm_verifier_brotli_cache_hit_rate %.2f\n", metrics.BrotliMetrics.CacheHitRate)

		output += fmt.Sprintf("# HELP llm_verifier_brotli_avg_detection_time_seconds Average Brotli detection time in seconds\n")
		output += fmt.Sprintf("# TYPE llm_verifier_brotli_avg_detection_time_seconds gauge\n")
		output += fmt.Sprintf("llm_verifier_brotli_avg_detection_time_seconds %f\n", metrics.BrotliMetrics.AvgDetectionTime.Seconds())

		c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		c.String(http.StatusOK, output)
	})
}
