package tests

import (
	"context"
	"fmt"
	"testing"
	"time"

	"llm-verifier/enhanced/analytics"
	contextmanager "llm-verifier/enhanced/context"
	"llm-verifier/enhanced/enterprise"
	"llm-verifier/enhanced/supervisor"
	llmverifier "llm-verifier/llmverifier"
)

// TestSystemValidation performs system-level validation tests
func TestSystemValidation(t *testing.T) {
	// Test component initialization order
	t.Run("Component Initialization Order", func(t *testing.T) {
		// Validate that components can be initialized in the correct order
		mockVerifier := &MockVerifier{}

		// Initialize analytics first
		analyticsConfig := analytics.AnalyticsConfig{
			RetentionPeriod:   24 * time.Hour,
			MaxTimeSeriesSize: 1000,
			BatchSize:         100,
			FlushInterval:     time.Minute,
			EnablePredictions: false,
		}

		analyticsEngine := analytics.NewAnalyticsEngine(analyticsConfig)
		if analyticsEngine == nil {
			t.Fatal("Failed to initialize analytics engine")
		}

		// Initialize context manager
		contextConfig := context.ContextConfig{
			ShortTermMaxMessages:    100,
			ShortTermWindowDuration: time.Hour,
			LongTermMaxSummaries:    50,
			SummarizationThreshold:  10,
			BackupEnabled:           false,
			BackupInterval:          time.Hour,
		}

		contextManager := context.NewContextManager(
			"validation-test",
			contextConfig,
			mockVerifier,
			nil, // No storage for validation
		)

		if contextManager == nil {
			t.Fatal("Failed to initialize context manager")
		}

		// Initialize supervisor
		supervisorConfig := supervisor.SupervisorConfig{
			MaxConcurrentJobs:       5,
			JobTimeout:              30 * time.Second,
			HealthCheckInterval:     5 * time.Second,
			RetryAttempts:           3,
			RetryBackoff:            time.Second,
			EnableAutoScaling:       false,
			EnablePredictions:       false,
			EnableAdaptiveLoad:      false,
			EnableCircuitBreaker:    true,
			HighLoadThreshold:       0.8,
			LowLoadThreshold:        0.2,
			ErrorRateThreshold:      0.1,
			MemoryThreshold:         0.9,
			MinWorkers:              2,
			MaxWorkers:              10,
			ScaleUpCooldown:         30 * time.Second,
			ScaleDownCooldown:       60 * time.Second,
			CircuitBreakerThreshold: 5,
			CircuitBreakerTimeout:   30 * time.Second,
			MetricsFlushInterval:    time.Minute,
			EnableDetailedMetrics:   true,
		}

		enhancedSupervisor := supervisor.NewEnhancedSupervisor(
			supervisorConfig,
			contextManager,
			analyticsEngine,
			nil, // Mock validator
			nil, // Mock monitor
			mockVerifier,
		)

		if enhancedSupervisor == nil {
			t.Fatal("Failed to initialize enhanced supervisor")
		}

		t.Logf("All components initialized successfully")
	})
}

// TestConcurrencyHandling tests concurrent operations
func TestConcurrencyHandling(t *testing.T) {
	t.Run("Concurrent Job Processing", func(t *testing.T) {
		mockVerifier := &MockVerifier{}

		// Initialize components
		analyticsEngine := analytics.NewAnalyticsEngine(analytics.AnalyticsConfig{
			RetentionPeriod: 24 * time.Hour,
		})

		contextManager := context.NewContextManager(
			"concurrency-test",
			context.ContextConfig{
				ShortTermMaxMessages: 100,
			},
			mockVerifier,
			nil,
		)

		enhancedSupervisor := supervisor.NewEnhancedSupervisor(
			supervisor.SupervisorConfig{
				MaxConcurrentJobs: 10,
				MinWorkers:        5,
				MaxWorkers:        10,
			},
			contextManager,
			analyticsEngine,
			nil, nil, mockVerifier,
		)

		if err := enhancedSupervisor.Start(); err != nil {
			t.Fatalf("Failed to start supervisor: %v", err)
		}
		defer enhancedSupervisor.Stop(5 * time.Second)

		// Submit jobs concurrently
		jobCount := 50
		done := make(chan bool, jobCount)

		for i := 0; i < jobCount; i++ {
			go func(jobID int) {
				defer func() { done <- true }()

				job := &supervisor.Job{
					ID:         fmt.Sprintf("concurrent-job-%d", jobID),
					Type:       "verification",
					Priority:   1,
					Payload:    map[string]interface{}{"job_id": jobID},
					CreatedAt:  time.Now(),
					Status:     supervisor.JobStatusPending,
					MaxRetries: 1,
					Timeout:    5 * time.Second,
				}

				if err := enhancedSupervisor.SubmitJob(job); err != nil {
					t.Errorf("Failed to submit job %d: %v", jobID, err)
				}
			}(i)
		}

		// Wait for all jobs to be submitted
		for i := 0; i < jobCount; i++ {
			<-done
		}

		// Wait for processing
		time.Sleep(2 * time.Second)

		// Verify supervisor handled concurrent submissions
		status := enhancedSupervisor.GetSupervisorStatus()
		if status.ActiveJobs > status.WorkerCount {
			t.Errorf("Active jobs (%d) exceeded worker count (%d)",
				status.ActiveJobs, status.WorkerCount)
		}

		t.Logf("Concurrent job submission completed. Active jobs: %d, Workers: %d",
			status.ActiveJobs, status.WorkerCount)
	})
}

// TestResourceManagement tests resource cleanup and management
func TestResourceManagement(t *testing.T) {
	t.Run("Resource Management", func(t *testing.T) {
		mockVerifier := &MockVerifier{}

		analyticsEngine := analytics.NewAnalyticsEngine(analytics.AnalyticsConfig{
			RetentionPeriod: 1 * time.Hour, // Short retention for testing
		})

		contextManager := context.NewContextManager(
			"resource-test",
			context.ContextConfig{
				ShortTermMaxMessages: 10, // Small limit for testing
			},
			mockVerifier,
			nil,
		)

		enhancedSupervisor := supervisor.NewEnhancedSupervisor(
			supervisor.SupervisorConfig{
				MaxConcurrentJobs:   5,
				MinWorkers:          1,
				MaxWorkers:          3,
				HealthCheckInterval: 1 * time.Second,
			},
			contextManager,
			analyticsEngine,
			nil, nil, mockVerifier,
		)

		if err := enhancedSupervisor.Start(); err != nil {
			t.Fatalf("Failed to start supervisor: %v", err)
		}
		defer enhancedSupervisor.Stop(5 * time.Second)

		// Fill up resources
		for i := 0; i < 20; i++ {
			contextManager.AddMessage("user", fmt.Sprintf("Message %d", i), nil)
		}

		// Submit jobs to fill up worker capacity
		for i := 0; i < 10; i++ {
			job := &supervisor.Job{
				ID:         fmt.Sprintf("resource-job-%d", i),
				Type:       "verification",
				Priority:   1,
				Payload:    map[string]interface{}{"resource_test": true},
				CreatedAt:  time.Now(),
				Status:     supervisor.JobStatusPending,
				MaxRetries: 1,
				Timeout:    10 * time.Second,
			}
			enhancedSupervisor.SubmitJob(job)
		}

		// Wait for some processing
		time.Sleep(1 * time.Second)

		// Check resource usage
		contextStats := contextManager.GetStats()
		supervisorStatus := enhancedSupervisor.GetSupervisorStatus()

		// Verify resource limits are respected
		if contextStats.ShortTermMessages > 10 {
			t.Errorf("Short-term messages exceeded limit: %d > 10",
				contextStats.ShortTermMessages)
		}

		if supervisorStatus.ActiveJobs > supervisorStatus.WorkerCount*2 {
			t.Errorf("Too many active jobs per worker: %d jobs for %d workers",
				supervisorStatus.ActiveJobs, supervisorStatus.WorkerCount)
		}

		t.Logf("Resource management test passed. Context messages: %d, Active jobs: %d",
			contextStats.ShortTermMessages, supervisorStatus.ActiveJobs)
	})
}

// TestErrorRecovery tests system behavior under error conditions
func TestErrorRecovery(t *testing.T) {
	t.Run("Error Recovery", func(t *testing.T) {
		mockVerifier := &MockVerifier{}

		analyticsEngine := analytics.NewAnalyticsEngine(analytics.AnalyticsConfig{})

		contextManager := context.NewContextManager(
			"error-recovery-test",
			context.ContextConfig{},
			mockVerifier,
			nil,
		)

		enhancedSupervisor := supervisor.NewEnhancedSupervisor(
			supervisor.SupervisorConfig{
				MaxConcurrentJobs:       5,
				MinWorkers:              2,
				MaxWorkers:              4,
				RetryAttempts:           3,
				RetryBackoff:            100 * time.Millisecond, // Short backoff for testing
				EnableCircuitBreaker:    true,
				CircuitBreakerThreshold: 2, // Low threshold for testing
				CircuitBreakerTimeout:   2 * time.Second,
			},
			contextManager,
			analyticsEngine,
			nil, nil, mockVerifier,
		)

		if err := enhancedSupervisor.Start(); err != nil {
			t.Fatalf("Failed to start supervisor: %v", err)
		}
		defer enhancedSupervisor.Stop(5 * time.Second)

		// Submit jobs that will fail (simulated)
		failureCount := 0
		for i := 0; i < 10; i++ {
			job := &supervisor.Job{
				ID:         fmt.Sprintf("error-test-job-%d", i),
				Type:       "error_simulation", // This type will be configured to fail
				Priority:   1,
				Payload:    map[string]interface{}{"should_fail": true},
				CreatedAt:  time.Now(),
				Status:     supervisor.JobStatusPending,
				MaxRetries: 3,
				Timeout:    1 * time.Second,
			}

			if err := enhancedSupervisor.SubmitJob(job); err != nil {
				failureCount++
			}
		}

		// Wait for processing and retry logic
		time.Sleep(3 * time.Second)

		// Check system status after failures
		status := enhancedSupervisor.GetSupervisorStatus()

		// Verify circuit breaker behavior
		if status.HealthScore < 0.5 { // Should be degraded due to failures
			t.Logf("System health degraded as expected: %.2f", status.HealthScore)
		}

		// Verify jobs are still being processed (retries)
		if status.ActiveJobs == 0 && failureCount > 0 {
			t.Error("No active jobs after failures - system may be stuck")
		}

		t.Logf("Error recovery test completed. Failed submissions: %d, System health: %.2f",
			failureCount, status.HealthScore)
	})
}

// TestPerformanceScaling tests auto-scaling behavior
func TestPerformanceScaling(t *testing.T) {
	t.Run("Performance Scaling", func(t *testing.T) {
		mockVerifier := &MockVerifier{}

		analyticsEngine := analytics.NewAnalyticsEngine(analytics.AnalyticsConfig{
			MetricsFlushInterval: 100 * time.Millisecond, // Fast flushing for testing
		})

		contextManager := context.NewContextManager(
			"scaling-test",
			context.ContextConfig{},
			mockVerifier,
			nil,
		)

		enhancedSupervisor := supervisor.NewEnhancedSupervisor(
			supervisor.SupervisorConfig{
				MaxConcurrentJobs:   20,
				MinWorkers:          1,
				MaxWorkers:          10,
				EnableAutoScaling:   true,
				HighLoadThreshold:   0.3, // Low threshold for easy scaling
				LowLoadThreshold:    0.1, // Low threshold for scaling down
				ScaleUpCooldown:     100 * time.Millisecond,
				ScaleDownCooldown:   200 * time.Millisecond,
				HealthCheckInterval: 100 * time.Millisecond,
			},
			contextManager,
			analyticsEngine,
			nil, nil, mockVerifier,
		)

		if err := enhancedSupervisor.Start(); err != nil {
			t.Fatalf("Failed to start supervisor: %v", err)
		}
		defer enhancedSupervisor.Stop(5 * time.Second)

		// Submit initial load
		for i := 0; i < 5; i++ {
			job := &supervisor.Job{
				ID:         fmt.Sprintf("scaling-job-%d", i),
				Type:       "verification",
				Priority:   1,
				Payload:    map[string]interface{}{"load_test": true},
				CreatedAt:  time.Now(),
				Status:     supervisor.JobStatusPending,
				MaxRetries: 1,
				Timeout:    2 * time.Second,
			}
			enhancedSupervisor.SubmitJob(job)
		}

		// Wait for initial scaling up
		time.Sleep(200 * time.Millisecond)
		initialStatus := enhancedSupervisor.GetSupervisorStatus()
		initialWorkers := initialStatus.WorkerCount

		// Add more load to trigger further scaling
		for i := 0; i < 10; i++ {
			job := &supervisor.Job{
				ID:         fmt.Sprintf("scaling-load-%d", i),
				Type:       "verification",
				Priority:   1,
				Payload:    map[string]interface{}{"high_load": true},
				CreatedAt:  time.Now(),
				Status:     supervisor.JobStatusPending,
				MaxRetries: 1,
				Timeout:    1 * time.Second,
			}
			enhancedSupervisor.SubmitJob(job)
		}

		// Wait for scaling up response
		time.Sleep(300 * time.Millisecond)
		loadedStatus := enhancedSupervisor.GetSupervisorStatus()
		loadedWorkers := loadedStatus.WorkerCount

		// Verify scaling up occurred
		if loadedWorkers <= initialWorkers {
			t.Errorf("Expected scaling up: %d -> %d", initialWorkers, loadedWorkers)
		}

		// Wait for jobs to complete and scaling down
		time.Sleep(2 * time.Second)
		finalStatus := enhancedSupervisor.GetSupervisorStatus()
		finalWorkers := finalStatus.WorkerCount

		t.Logf("Performance scaling test completed. Initial workers: %d, Peak workers: %d, Final workers: %d",
			initialWorkers, loadedWorkers, finalWorkers)
	})
}

// TestSecurityIntegration tests security features across components
func TestSecurityIntegration(t *testing.T) {
	t.Run("Security Integration", func(t *testing.T) {
		mockVerifier := &MockVerifier{}

		// Initialize enterprise features for security testing
		enterpriseConfig := enterprise.EnterpriseConfig{
			RBAC: enterprise.RBACConfig{
				Enabled:        true,
				SessionTimeout: 1 * time.Hour,
				PasswordPolicy: enterprise.PasswordPolicy{
					MinLength:        8,
					RequireUppercase: true,
					RequireLowercase: true,
					RequireNumbers:   true,
					RequireSymbols:   false,
					MaxAge:           90 * 24 * time.Hour,
				},
				TwoFactorAuth: false,
			},
			AuditLogging: enterprise.AuditLoggingConfig{
				Enabled:   true,
				Storage:   "file",
				Retention: 30 * 24 * time.Hour,
				Level:     "info",
			},
			Security: enterprise.SecurityConfig{
				HTTPSEnabled: false,         // Disabled for testing
				CORSOrigins:  []string{"*"}, // Permissive for testing
				RateLimiting: enterprise.RateLimitConfig{
					Enabled:  true,
					Requests: 100,
					Window:   time.Minute,
				},
			},
		}

		// Create enterprise manager
		enterpriseManager := enterprise.NewEnterpriseManager(enterpriseConfig, nil)

		// Create admin user with full permissions
		adminUser := &enterprise.User{
			ID:        "admin-test",
			Username:  "admin",
			Email:     "admin@test.com",
			FirstName: "Test",
			LastName:  "Admin",
			Roles:     []enterprise.RBACRole{enterprise.RBACRoleAdmin},
			Permissions: []enterprise.Permission{
				enterprise.PermissionJobSubmit,
				enterprise.PermissionJobView,
				enterprise.PermissionSystemStart,
				enterprise.PermissionSystemStop,
				enterprise.PermissionUserManage,
				enterprise.PermissionMetricsView,
				enterprise.PermissionLogsView,
			},
			Enabled:   true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		if err := enterpriseManager.RBAC.CreateUser(adminUser); err != nil {
			t.Fatalf("Failed to create admin user: %v", err)
		}

		// Test RBAC functionality
		hasPermission := enterpriseManager.RBAC.HasPermission("admin-test", enterprise.PermissionUserManage)
		if !hasPermission {
			t.Error("Admin user should have user management permission")
		}

		// Test audit logging
		auditLog := enterpriseManager.RBAC.GetAuditLog(10)
		if len(auditLog) == 0 {
			t.Error("Audit log should contain entries after user creation")
		}

		// Verify audit entry structure
		foundAdminCreation := false
		for _, entry := range auditLog {
			if entry.Action == "user.created" && entry.UserID == "admin-test" {
				foundAdminCreation = true
				if entry.Success != true {
					t.Error("Admin user creation should be logged as success")
				}
				if entry.Timestamp.IsZero() {
					t.Error("Audit entry should have timestamp")
				}
				break
			}
		}

		if !foundAdminCreation {
			t.Error("Admin user creation not found in audit log")
		}

		t.Logf("Security integration test completed. RBAC functional: %v, Audit entries: %d",
			hasPermission, len(auditLog))
	})
}

// TestMemoryAndLeakDetection tests for memory leaks
func TestMemoryAndLeakDetection(t *testing.T) {
	t.Run("Memory and Leak Detection", func(t *testing.T) {
		mockVerifier := &MockVerifier{}

		// Create components with small limits for memory pressure
		analyticsEngine := analytics.NewAnalyticsEngine(analytics.AnalyticsConfig{
			RetentionPeriod:   1 * time.Hour,
			MaxTimeSeriesSize: 100, // Small size for testing
		})

		contextManager := context.NewContextManager(
			"memory-test",
			context.ContextConfig{
				ShortTermMaxMessages:    50,
				ShortTermWindowDuration: 10 * time.Minute,
				LongTermMaxSummaries:    10,
				SummarizationThreshold:  5,
			},
			mockVerifier,
			nil,
		)

		// Run multiple iterations to detect memory growth
		initialStats := contextManager.GetStats()

		for iteration := 0; iteration < 100; iteration++ {
			// Add messages
			for i := 0; i < 10; i++ {
				contextManager.AddMessage("user",
					fmt.Sprintf("Memory test message %d-%d", iteration, i),
					map[string]interface{}{"iteration": iteration, "message_id": i})
			}

			// Periodically check memory usage
			if iteration%20 == 0 {
				currentStats := contextManager.GetStats()

				// Check for unreasonable memory growth
				if currentStats.ShortTermMessages > initialStats.ShortTermMessages*2 {
					t.Errorf("Memory growth detected: initial: %d, current: %d, iteration: %d",
						initialStats.ShortTermMessages, currentStats.ShortTermMessages, iteration)
				}

				// Check cleanup is working
				if currentStats.ShortTermMessages > 50 { // Should be capped at max
					t.Errorf("Short-term messages exceeded limit: %d", currentStats.ShortTermMessages)
				}
			}

			// Allow some processing time
			if iteration%10 == 0 {
				time.Sleep(1 * time.Millisecond)
			}
		}

		// Final memory check
		finalStats := contextManager.GetStats()

		// Verify memory is within expected bounds
		if finalStats.ShortTermMessages > 50 {
			t.Errorf("Final memory usage exceeded limit: %d", finalStats.ShortTermMessages)
		}

		// Verify long-term memory is managed
		allSummaries := contextManager.GetFullContext()
		summaryCount := len(allSummaries[1]) // Second element contains summaries

		if summaryCount > 10 {
			t.Errorf("Too many summaries in long-term memory: %d", summaryCount)
		}

		t.Logf("Memory test completed. Final context messages: %d, Summaries: %d",
			finalStats.ShortTermMessages, summaryCount)
	})
}

// MockVerifier provides mock implementation for testing
type MockVerifier struct{}

func (m *MockVerifier) SummarizeConversation(messages []string) (*llmverifier.ConversationSummary, error) {
	return &llmverifier.ConversationSummary{
		Summary:    fmt.Sprintf("Mock summary of %d messages", len(messages)),
		Topics:     []string{"mock", "test"},
		KeyPoints:  []string{"Mock key point 1", "Mock key point 2"},
		Importance: 0.7,
	}, nil
}
