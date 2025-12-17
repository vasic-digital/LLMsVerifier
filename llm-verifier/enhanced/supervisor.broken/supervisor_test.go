package supervisor

import (
	"testing"
	"time"
)

func TestEnhancedSupervisorBasic(t *testing.T) {
	// Test supervisor configuration
	config := SupervisorConfig{
		MaxConcurrentJobs:       5,
		JobTimeout:              30 * time.Second,
		HealthCheckInterval:     10 * time.Second,
		RetryAttempts:           3,
		RetryBackoff:            5 * time.Second,
		EnableAutoScaling:       true,
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
		MetricsFlushInterval:    60 * time.Second,
		EnableDetailedMetrics:   true,
	}

	// Create mock components
	mockContextMgr := &MockContextManager{}
	mockAnalyticsEngine := &MockAnalyticsEngine{}
	mockValidator := &MockValidationGate{}
	mockMonitor := &MockMonitoringInterface{}
	mockVerifier := &MockVerifier{}

	// Create supervisor
	supervisor := NewEnhancedSupervisor(
		config,
		mockContextMgr,
		mockAnalyticsEngine,
		mockValidator,
		mockMonitor,
		mockVerifier,
	)

	if supervisor == nil {
		t.Fatal("Failed to create enhanced supervisor")
	}

	// Test starting supervisor
	err := supervisor.Start()
	if err != nil {
		t.Fatalf("Failed to start supervisor: %v", err)
	}

	// Give some time for workers to start
	time.Sleep(100 * time.Millisecond)

	// Test getting status
	status := supervisor.GetSupervisorStatus()
	if status.State != SupervisorStateActive {
		t.Errorf("Expected active state, got %v", status.State)
	}

	if status.WorkerCount != config.MinWorkers {
		t.Errorf("Expected %d workers, got %d", config.MinWorkers, status.WorkerCount)
	}

	// Test submitting a job
	job := &Job{
		ID:         "test-job-1",
		Type:       "verification",
		Priority:   1,
		Payload:    map[string]interface{}{"model_name": "test-model"},
		CreatedAt:  time.Now(),
		Status:     JobStatusPending,
		RetryCount: 0,
		MaxRetries: 3,
		Timeout:    10 * time.Second,
		Metadata:   map[string]interface{}{"test": true},
		Tags:       map[string]string{"env": "test"},
	}

	err = supervisor.SubmitJob(job)
	if err != nil {
		t.Fatalf("Failed to submit job: %v", err)
	}

	// Wait for job processing
	time.Sleep(200 * time.Millisecond)

	// Test getting job status
	retrievedJob, err := supervisor.GetJobStatus(job.ID)
	if err != nil {
		t.Fatalf("Failed to get job status: %v", err)
	}

	if retrievedJob.ID != job.ID {
		t.Errorf("Expected job ID %s, got %s", job.ID, retrievedJob.ID)
	}

	// Test stopping supervisor
	err = supervisor.Stop(5 * time.Second)
	if err != nil {
		t.Fatalf("Failed to stop supervisor: %v", err)
	}
}

func TestCircuitBreaker(t *testing.T) {
	cb := &CircuitBreaker{
		state:     CircuitBreakerClosed,
		threshold: 3,
		timeout:   5 * time.Second,
	}

	// Test initial state
	if cb.IsOpen() {
		t.Error("Circuit breaker should be closed initially")
	}

	// Record failures
	for i := 0; i < 3; i++ {
		cb.RecordFailure()
	}

	// Should be open now
	if !cb.IsOpen() {
		t.Error("Circuit breaker should be open after threshold failures")
	}

	// Record success should close it
	cb.RecordSuccess()
	if cb.IsOpen() {
		t.Error("Circuit breaker should be closed after success")
	}
}

func TestJobFilter(t *testing.T) {
	// Create test jobs
	jobs := []*Job{
		{
			ID:        "job-1",
			Type:      "verification",
			Priority:  1,
			CreatedAt: time.Now().Add(-2 * time.Hour),
			Status:    JobStatusCompleted,
			Tags:      map[string]string{"env": "test"},
		},
		{
			ID:        "job-2",
			Type:      "analysis",
			Priority:  2,
			CreatedAt: time.Now().Add(-1 * time.Hour),
			Status:    JobStatusRunning,
			Tags:      map[string]string{"env": "prod"},
		},
		{
			ID:        "job-3",
			Type:      "verification",
			Priority:  1,
			CreatedAt: time.Now(),
			Status:    JobStatusPending,
			Tags:      map[string]string{"env": "test"},
		},
	}

	// Test filtering by status
	completedStatus := JobStatusCompleted
	statusFilter := JobFilter{
		Status: &completedStatus,
	}

	var filtered []*Job
	for _, job := range jobs {
		if statusFilter.matches(job) {
			filtered = append(filtered, job)
		}
	}

	if len(filtered) != 1 {
		t.Errorf("Expected 1 completed job, got %d", len(filtered))
	}

	// Test filtering by type
	verificationType := "verification"
	typeFilter := JobFilter{
		Type: &verificationType,
	}

	filtered = []*Job{}
	for _, job := range jobs {
		if typeFilter.matches(job) {
			filtered = append(filtered, job)
		}
	}

	if len(filtered) != 2 {
		t.Errorf("Expected 2 verification jobs, got %d", len(filtered))
	}

	// Test filtering by tags
	tagFilter := JobFilter{
		Tags: map[string]string{"env": "test"},
	}

	filtered = []*Job{}
	for _, job := range jobs {
		if tagFilter.matches(job) {
			filtered = append(filtered, job)
		}
	}

	if len(filtered) != 2 {
		t.Errorf("Expected 2 test env jobs, got %d", len(filtered))
	}
}

func TestJobStatusString(t *testing.T) {
	testCases := []struct {
		status   JobStatus
		expected string
	}{
		{JobStatusPending, "pending"},
		{JobStatusRunning, "running"},
		{JobStatusCompleted, "completed"},
		{JobStatusFailed, "failed"},
		{JobStatusCancelled, "cancelled"},
		{JobStatusTimeout, "timeout"},
	}

	for _, tc := range testCases {
		if tc.status.String() != tc.expected {
			t.Errorf("Expected %s, got %s", tc.expected, tc.status.String())
		}
	}
}

func TestWorkerStatusString(t *testing.T) {
	testCases := []struct {
		status   WorkerStatus
		expected string
	}{
		{WorkerStatusIdle, "idle"},
		{WorkerStatusBusy, "busy"},
		{WorkerStatusError, "error"},
		{WorkerStatusMaintenance, "maintenance"},
		{WorkerStatusOffline, "offline"},
	}

	for _, tc := range testCases {
		if tc.status.String() != tc.expected {
			t.Errorf("Expected %s, got %s", tc.expected, tc.status.String())
		}
	}
}

func TestSupervisorStateString(t *testing.T) {
	testCases := []struct {
		state    SupervisorState
		expected string
	}{
		{SupervisorStateInactive, "inactive"},
		{SupervisorStateActive, "active"},
		{SupervisorStateDegraded, "degraded"},
		{SupervisorStateMaintenance, "maintenance"},
		{SupervisorStateError, "error"},
	}

	for _, tc := range testCases {
		if tc.state.String() != tc.expected {
			t.Errorf("Expected %s, got %s", tc.expected, tc.state.String())
		}
	}
}

func TestCircuitBreakerStateString(t *testing.T) {
	testCases := []struct {
		state    CircuitBreakerState
		expected string
	}{
		{CircuitBreakerClosed, "closed"},
		{CircuitBreakerOpen, "open"},
		{CircuitBreakerHalfOpen, "half_open"},
	}

	for _, tc := range testCases {
		if tc.state.String() != tc.expected {
			t.Errorf("Expected %s, got %s", tc.expected, tc.state.String())
		}
	}
}

// Mock implementations for testing

type MockContextManager struct{}

func (m *MockContextManager) AddMessage(role, content string, metadata map[string]interface{}) error {
	return nil
}

type MockAnalyticsEngine struct{}

func (m *MockAnalyticsEngine) RecordMetric(name string, metricType int, value float64, tags map[string]string, dimensions map[string]interface{}) error {
	return nil
}

type MockValidationGate struct{}

func (m *MockValidationGate) ValidateJob(job *Job) error {
	return nil
}

type MockMonitoringInterface struct{}

func (m *MockMonitoringInterface) RecordMetric(name string, value float64, tags map[string]string) error {
	return nil
}

func (m *MockMonitoringInterface) CheckHealth() error {
	return nil
}

type MockVerifier struct{}

func (m *MockVerifier) SummarizeConversation(messages []string) (interface{}, error) {
	return map[string]interface{}{
		"summary":    "Mock summary",
		"topics":     []string{"mock"},
		"key_points": []string{"mock point"},
		"importance": 0.5,
	}, nil
}
