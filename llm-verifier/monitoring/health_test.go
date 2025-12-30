package monitoring

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"llm-verifier/database"
)

// setupTestDB creates an in-memory database for testing
func setupTestDB(t *testing.T) *database.Database {
	db, err := database.New(":memory:")
	require.NoError(t, err, "Failed to create test database")
	return db
}

// cleanupTestDB closes the test database
func cleanupTestDB(t *testing.T, db *database.Database) {
	if db != nil {
		err := db.Close()
		if err != nil {
			t.Logf("Warning: failed to close test database: %v", err)
		}
	}
}

func TestHealthStatusConstants(t *testing.T) {
	assert.Equal(t, HealthStatus("healthy"), HealthStatusHealthy)
	assert.Equal(t, HealthStatus("degraded"), HealthStatusDegraded)
	assert.Equal(t, HealthStatus("unhealthy"), HealthStatusUnhealthy)
}

func TestComponentHealthStruct(t *testing.T) {
	now := time.Now()
	health := ComponentHealth{
		Name:         "Test Component",
		Status:       HealthStatusHealthy,
		Message:      "Component is healthy",
		LastChecked:  now,
		ResponseTime: time.Millisecond * 100,
		Details: map[string]interface{}{
			"key": "value",
		},
	}

	assert.Equal(t, "Test Component", health.Name)
	assert.Equal(t, HealthStatusHealthy, health.Status)
	assert.Equal(t, "Component is healthy", health.Message)
	assert.Equal(t, time.Millisecond*100, health.ResponseTime)
	assert.Equal(t, "value", health.Details["key"])
}

func TestSystemMetricsStruct(t *testing.T) {
	metrics := SystemMetrics{
		Timestamp: time.Now(),
		Uptime:    time.Hour,
		MemoryUsage: MemoryStats{
			Alloc:      1024,
			TotalAlloc: 2048,
		},
		Goroutines: 10,
	}

	assert.NotZero(t, metrics.Timestamp)
	assert.Equal(t, time.Hour, metrics.Uptime)
	assert.Equal(t, 10, metrics.Goroutines)
}

func TestMemoryStatsStruct(t *testing.T) {
	stats := MemoryStats{
		Alloc:     1000,
		HeapAlloc: 2000,
		HeapSys:   3000,
		NumGC:     2,
	}

	assert.Equal(t, uint64(1000), stats.Alloc)
	assert.Equal(t, uint64(2000), stats.HeapAlloc)
	assert.Equal(t, uint64(3000), stats.HeapSys)
	assert.Equal(t, uint32(2), stats.NumGC)
}

func TestDatabaseStatsStruct(t *testing.T) {
	stats := DatabaseStats{
		ConnectionsInUse: 5,
		ConnectionsIdle:  10,
		ConnectionsOpen:  15,
		QueryCount:       100,
		QueryDuration:    time.Millisecond * 50,
		ErrorCount:       2,
	}

	assert.Equal(t, 5, stats.ConnectionsInUse)
	assert.Equal(t, 10, stats.ConnectionsIdle)
	assert.Equal(t, 15, stats.ConnectionsOpen)
	assert.Equal(t, int64(100), stats.QueryCount)
}

func TestAPIMetricsStruct(t *testing.T) {
	metrics := APIMetrics{
		TotalRequests:       1000,
		ActiveRequests:      10,
		AverageResponseTime: time.Millisecond * 100,
		RequestRate:         15.5,
		ErrorRate:           0.02,
		EndpointStats: map[string]EndpointStats{
			"/api/v1/models": {
				Requests: 500,
				Errors:   5,
			},
		},
	}

	assert.Equal(t, int64(1000), metrics.TotalRequests)
	assert.Equal(t, int64(10), metrics.ActiveRequests)
	assert.Equal(t, 15.5, metrics.RequestRate)
	assert.Equal(t, 0.02, metrics.ErrorRate)
}

func TestEndpointStatsStruct(t *testing.T) {
	now := time.Now()
	stats := EndpointStats{
		Requests:        100,
		Errors:          2,
		AvgResponseTime: time.Millisecond * 50,
		LastRequest:     now,
	}

	assert.Equal(t, int64(100), stats.Requests)
	assert.Equal(t, int64(2), stats.Errors)
	assert.Equal(t, time.Millisecond*50, stats.AvgResponseTime)
}

func TestVerificationStatsStruct(t *testing.T) {
	stats := VerificationStats{
		ActiveVerifications: 5,
		CompletedToday:      20,
		FailedToday:         1,
		AverageDuration:     time.Minute * 3,
		SuccessRate:         0.95,
		QueueLength:         3,
	}

	assert.Equal(t, 5, stats.ActiveVerifications)
	assert.Equal(t, 20, stats.CompletedToday)
	assert.Equal(t, time.Minute*3, stats.AverageDuration)
	assert.Equal(t, 0.95, stats.SuccessRate)
}

func TestNewHealthChecker(t *testing.T) {
	db := (*database.Database)(nil)
	hc := NewHealthChecker(db)

	assert.NotNil(t, hc)
	assert.NotNil(t, hc.components)
	assert.NotNil(t, hc.systemMetrics)
	assert.NotZero(t, hc.startTime)
	assert.NotNil(t, hc.ctx)

}

func TestHealthCheckerStart(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	hc := NewHealthChecker(db)

	// Start with a short interval for testing
	hc.Start(100 * time.Millisecond)

	// Wait for at least one check cycle
	time.Sleep(150 * time.Millisecond)

	// Check that health status is available
	status := hc.GetHealthStatus()
	assert.NotEmpty(t, string(status))

	hc.Stop()
}

func TestHealthCheckerStop(t *testing.T) {
	db := (*database.Database)(nil)
	hc := NewHealthChecker(db)

	hc.Start(time.Second)
	hc.Stop()

}

func TestHealthCheckerGetHealthStatus(t *testing.T) {
	db := (*database.Database)(nil)
	hc := NewHealthChecker(db)

	status := hc.GetHealthStatus()

	assert.Equal(t, HealthStatusHealthy, status)

}

func TestHealthCheckerGetComponentHealth(t *testing.T) {
	db := (*database.Database)(nil)
	hc := NewHealthChecker(db)

	components := hc.GetComponentHealth()

	assert.NotNil(t, components)
	assert.NotEmpty(t, components)
	assert.Contains(t, components, "database")
	assert.Contains(t, components, "api")
	assert.Contains(t, components, "scheduler")

}

func TestHealthCheckerGetSystemMetrics(t *testing.T) {
	db := (*database.Database)(nil)
	hc := NewHealthChecker(db)
	// Update metrics first
	hc.updateSystemMetrics()

	metrics := hc.GetSystemMetrics()
	assert.NotZero(t, metrics.Timestamp)
	assert.NotZero(t, metrics.Uptime)
	assert.GreaterOrEqual(t, metrics.Goroutines, 0)
	assert.NotZero(t, metrics.MemoryUsage.Alloc)

}

func TestHealthCheckerComponentDetails(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	hc := NewHealthChecker(db)

	components := hc.GetComponentHealth()
	assert.NotNil(t, components)

	// Check that database component exists
	dbComponent, exists := components["database"]
	assert.True(t, exists, "database component should exist")
	assert.Equal(t, "Database", dbComponent.Name)
}

func TestHealthCheckerInitializeComponents(t *testing.T) {
	db := (*database.Database)(nil)
	hc := NewHealthChecker(db)

	expectedComponents := []string{"database", "api", "scheduler", "notifications"}

	for _, name := range expectedComponents {
		_, exists := hc.components[name]
		assert.True(t, exists, "Component %s should exist", name)
	}

}

func TestHealthCheckerCheckAllComponents(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	hc := NewHealthChecker(db)

	// Run check on all components
	hc.checkAllComponents()

	// Verify all components have been checked
	components := hc.GetComponentHealth()
	for name, component := range components {
		assert.NotEmpty(t, component.Name, "Component %s should have a name", name)
		assert.NotEmpty(t, string(component.Status), "Component %s should have a status", name)
	}
}

func TestHealthCheckerUpdateSystemMetrics(t *testing.T) {
	db := (*database.Database)(nil)
	hc := NewHealthChecker(db)

	hc.updateSystemMetrics()

	metrics := hc.GetSystemMetrics()
	assert.NotZero(t, metrics.Timestamp)
	assert.GreaterOrEqual(t, metrics.Goroutines, 0)
	assert.NotZero(t, metrics.MemoryUsage.Alloc)

}

func TestHealthCheckerDatabaseHealth(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	hc := NewHealthChecker(db)

	// Check database health
	hc.checkDatabaseHealth()

	// Verify database component status
	components := hc.GetComponentHealth()
	dbComponent, exists := components["database"]
	assert.True(t, exists, "database component should exist")
	assert.Equal(t, HealthStatusHealthy, dbComponent.Status, "database should be healthy with a valid connection")
}

func TestHealthCheckerAPIHealth(t *testing.T) {
	db := (*database.Database)(nil)
	hc := NewHealthChecker(db)

	hc.checkAPIHealth()

	component := hc.GetComponentHealth()["api"]
	assert.Equal(t, "API Server", component.Name)
	assert.Equal(t, HealthStatusHealthy, component.Status)
	assert.NotNil(t, component.Details)

}

func TestHealthCheckerSchedulerHealth(t *testing.T) {
	db := (*database.Database)(nil)
	hc := NewHealthChecker(db)

	// Set scheduler as running for the test
	hc.metricsTracker.SetSchedulerRunning(true)

	hc.checkSchedulerHealth()

	component := hc.GetComponentHealth()["scheduler"]
	assert.Equal(t, "Job Scheduler", component.Name)
	assert.Equal(t, HealthStatusHealthy, component.Status)
	assert.NotNil(t, component.Details)

}

func TestHealthCheckerNotificationsHealth(t *testing.T) {
	db := (*database.Database)(nil)
	hc := NewHealthChecker(db)

	hc.checkNotificationsHealth()

	component := hc.GetComponentHealth()["notifications"]
	assert.Equal(t, "Notification System", component.Name)
	assert.Equal(t, HealthStatusHealthy, component.Status)
	assert.NotNil(t, component.Details)

}

func TestHealthCheckerMetricsFields(t *testing.T) {
	db := (*database.Database)(nil)
	hc := NewHealthChecker(db)

	// Record some test data for metrics
	hc.metricsTracker.UpdateDatabaseStats(5, 10, 15)
	hc.metricsTracker.RecordQuery(time.Millisecond * 15)
	hc.metricsTracker.RecordAPIRequest("/api/v1/test")
	hc.metricsTracker.RecordAPIResponse("/api/v1/test", time.Millisecond*50)
	hc.metricsTracker.RecordVerificationStarted()
	hc.metricsTracker.RecordVerificationCompleted(true, time.Minute*3)
	hc.metricsTracker.SetNotificationChannels(3)

	hc.updateSystemMetrics()
	metrics := hc.GetSystemMetrics()

	// Test MemoryStats (from runtime, always non-zero)
	assert.NotZero(t, metrics.MemoryUsage.Alloc)
	assert.NotZero(t, metrics.MemoryUsage.Sys)
	assert.NotZero(t, metrics.MemoryUsage.HeapAlloc)

	// Test DatabaseStats (now from tracker)
	assert.NotZero(t, metrics.DatabaseStats.ConnectionsOpen)
	assert.NotZero(t, metrics.DatabaseStats.QueryCount)

	// Test APIMetrics (now from tracker)
	assert.NotZero(t, metrics.APIMetrics.TotalRequests)
	assert.NotEmpty(t, metrics.APIMetrics.EndpointStats)

	// Test VerificationStats (now from tracker)
	assert.NotNil(t, metrics.VerificationStats)

}

func TestHealthCheckerUptimeCalculation(t *testing.T) {
	db := (*database.Database)(nil)
	hc := NewHealthChecker(db)

	time.Sleep(time.Millisecond * 10)
	metrics := hc.GetSystemMetrics()

	assert.Greater(t, metrics.Uptime, time.Duration(0))
	assert.Less(t, metrics.Uptime, time.Second)

}

func TestHealthCheckerConcurrentAccess(t *testing.T) {
	db := (*database.Database)(nil)
	hc := NewHealthChecker(db)

	done := make(chan bool)

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			_ = hc.GetHealthStatus()
			_ = hc.GetComponentHealth()
			_ = hc.GetSystemMetrics()
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

}

func TestHealthCheckerStartStopMultipleTimes(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	hc := NewHealthChecker(db)

	// Start and stop multiple times
	for i := 0; i < 3; i++ {
		hc.Start(100 * time.Millisecond)
		time.Sleep(50 * time.Millisecond)
		hc.Stop()
	}

	// Final verification - should still be functional
	hc.Start(100 * time.Millisecond)
	status := hc.GetHealthStatus()
	assert.NotEmpty(t, string(status))
	hc.Stop()
}

func TestHealthCheckerLongRunning(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	hc := NewHealthChecker(db)

	// Start with short interval
	hc.Start(50 * time.Millisecond)

	// Let it run for a few cycles
	time.Sleep(200 * time.Millisecond)

	// Verify metrics are being updated
	metrics := hc.GetSystemMetrics()
	assert.NotZero(t, metrics.Timestamp)
	assert.NotZero(t, metrics.Uptime)

	hc.Stop()
}

func TestHealthCheckerRegisterHealthEndpoints(t *testing.T) {
	db := (*database.Database)(nil)
	hc := NewHealthChecker(db)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	hc.RegisterHealthEndpoints(router)

	// Test /health endpoint
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "healthy")

	// Test /health/detailed endpoint
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/health/detailed", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "components")
	assert.Contains(t, w.Body.String(), "metrics")

	// Test /health/ready endpoint
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/health/ready", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ready")

	// Test /health/live endpoint
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/health/live", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "alive")

	// Test /metrics endpoint
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/metrics", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "# HELP")
	assert.Contains(t, w.Body.String(), "# TYPE")

}

func TestHealthCheckerEmptyDatabase(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	hc := NewHealthChecker(db)

	// Check database health with empty database (no data)
	hc.checkDatabaseHealth()

	// Database should still be healthy even if empty
	components := hc.GetComponentHealth()
	dbComponent, exists := components["database"]
	assert.True(t, exists, "database component should exist")
	assert.Equal(t, HealthStatusHealthy, dbComponent.Status, "empty database should still be healthy")
}

func TestHealthCheckerMemoryMetrics(t *testing.T) {
	db := (*database.Database)(nil)
	hc := NewHealthChecker(db)

	hc.updateSystemMetrics()
	metrics := hc.GetSystemMetrics()

	// Check various memory fields
	assert.NotZero(t, metrics.MemoryUsage.Alloc)
	assert.NotZero(t, metrics.MemoryUsage.TotalAlloc)
	assert.NotZero(t, metrics.MemoryUsage.Sys)
	assert.NotNil(t, metrics.MemoryUsage.HeapAlloc)
	assert.NotNil(t, metrics.MemoryUsage.HeapSys)
	assert.NotNil(t, metrics.MemoryUsage.HeapIdle)
	assert.NotNil(t, metrics.MemoryUsage.NumGC)

}

func TestHealthCheckerAPIMetricsDetails(t *testing.T) {
	db := (*database.Database)(nil)
	hc := NewHealthChecker(db)

	// Note: Since we now use real MetricsTracker, record test data first
	// Note: Since we now use real MetricsTracker, record test data first
	hc.metricsTracker.RecordAPIRequest("/api/v1/models")
	hc.metricsTracker.RecordAPIResponse("/api/v1/models", time.Millisecond*50)
	hc.metricsTracker.RecordAPIRequest("/api/v1/providers")
	hc.metricsTracker.RecordAPIResponse("/api/v1/providers", time.Millisecond*80)
	hc.metricsTracker.RecordAPIRequest("/api/v1/models")
	hc.metricsTracker.RecordAPIResponse("/api/v1/models", time.Millisecond*60)
	hc.metricsTracker.RecordAPIError("/api/v1/models")

	hc.updateSystemMetrics()
	metrics := hc.GetSystemMetrics()

	assert.NotZero(t, metrics.APIMetrics.TotalRequests)
	assert.NotZero(t, metrics.APIMetrics.AverageResponseTime)
	assert.NotZero(t, metrics.APIMetrics.RequestRate)
	assert.NotZero(t, metrics.APIMetrics.ErrorRate)
	assert.NotEmpty(t, metrics.APIMetrics.EndpointStats)

	// Check endpoint stats
	for endpoint, stats := range metrics.APIMetrics.EndpointStats {
		assert.NotEmpty(t, endpoint)
		assert.NotZero(t, stats.Requests)
		assert.NotNil(t, stats.LastRequest)
	}

}

func TestHealthCheckerVerificationMetrics(t *testing.T) {
	db := (*database.Database)(nil)
	hc := NewHealthChecker(db)

	hc.updateSystemMetrics()
	metrics := hc.GetSystemMetrics()

	assert.NotNil(t, metrics.VerificationStats)
	assert.NotNil(t, metrics.VerificationStats.AverageDuration)
	assert.NotNil(t, metrics.VerificationStats.SuccessRate)

}
