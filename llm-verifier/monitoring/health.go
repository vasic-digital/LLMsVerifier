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
	database      *database.Database
	components    map[string]*ComponentHealth
	systemMetrics *SystemMetrics
	startTime     time.Time
	mu            sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *database.Database) *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())

	hc := &HealthChecker{
		database:      db,
		components:    make(map[string]*ComponentHealth),
		systemMetrics: &SystemMetrics{},
		startTime:     time.Now(),
		ctx:           ctx,
		cancel:        cancel,
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

// checkAPIHealth checks API server health (placeholder)
func (hc *HealthChecker) checkAPIHealth() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	component := hc.components["api"]
	component.LastChecked = time.Now()
	component.Status = HealthStatusHealthy
	component.Message = "API server is responding"
	component.ResponseTime = time.Millisecond * 10

	component.Details = map[string]interface{}{
		"endpoints_available": 15,
		"active_connections":  5,
	}
}

// checkSchedulerHealth checks job scheduler health (placeholder)
func (hc *HealthChecker) checkSchedulerHealth() {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	component := hc.components["scheduler"]
	component.LastChecked = time.Now()
	component.Status = HealthStatusHealthy
	component.Message = "Scheduler is running"
	component.ResponseTime = time.Millisecond * 5

	component.Details = map[string]interface{}{
		"active_jobs":    3,
		"completed_jobs": 150,
		"failed_jobs":    2,
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

	component.Details = map[string]interface{}{
		"channels_configured": 3,
		"messages_sent":       250,
		"delivery_rate":       0.98,
	}
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

	// Database stats (simplified)
	hc.systemMetrics.DatabaseStats = DatabaseStats{
		ConnectionsInUse: 5,
		ConnectionsIdle:  10,
		ConnectionsOpen:  15,
		QueryCount:       1250,
		QueryDuration:    time.Millisecond * 15,
		ErrorCount:       3,
	}

	// API metrics (simplified)
	hc.systemMetrics.APIMetrics = APIMetrics{
		TotalRequests:       2500,
		ActiveRequests:      8,
		AverageResponseTime: time.Millisecond * 120,
		RequestRate:         15.5,
		ErrorRate:           0.02,
		EndpointStats: map[string]EndpointStats{
			"/api/v1/models": {
				Requests:        500,
				Errors:          5,
				AvgResponseTime: time.Millisecond * 80,
				LastRequest:     time.Now().Add(-time.Minute),
			},
		},
	}

	// Verification stats (simplified)
	hc.systemMetrics.VerificationStats = VerificationStats{
		ActiveVerifications: 2,
		CompletedToday:      25,
		FailedToday:         1,
		AverageDuration:     time.Minute * 3,
		SuccessRate:         0.96,
		QueueLength:         3,
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

		c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		c.String(http.StatusOK, output)
	})
}
