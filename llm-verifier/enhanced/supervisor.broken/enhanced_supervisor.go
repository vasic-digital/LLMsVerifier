package supervisor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"llm-verifier/enhanced/analytics"
	enhancedContext "llm-verifier/enhanced/context"
	"llm-verifier/enhanced/validation"
	llmverifier "llm-verifier/llmverifier"
	"llm-verifier/monitoring"
)

// SupervisorState represents the current state of the supervisor
type SupervisorState int

const (
	SupervisorStateInactive SupervisorState = iota
	SupervisorStateActive
	SupervisorStateDegraded
	SupervisorStateMaintenance
	SupervisorStateError
)

// SupervisorConfig holds configuration for the enhanced supervisor
type SupervisorConfig struct {
	// Core settings
	MaxConcurrentJobs   int           `yaml:"max_concurrent_jobs"`
	JobTimeout          time.Duration `yaml:"job_timeout"`
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`
	RetryAttempts       int           `yaml:"retry_attempts"`
	RetryBackoff        time.Duration `yaml:"retry_backoff"`

	// Enhanced features
	EnableAutoScaling    bool `yaml:"enable_auto_scaling"`
	EnablePredictions    bool `yaml:"enable_predictions"`
	EnableAdaptiveLoad   bool `yaml:"enable_adaptive_load"`
	EnableCircuitBreaker bool `yaml:"enable_circuit_breaker"`

	// Thresholds
	HighLoadThreshold  float64 `yaml:"high_load_threshold"`
	LowLoadThreshold   float64 `yaml:"low_load_threshold"`
	ErrorRateThreshold float64 `yaml:"error_rate_threshold"`
	MemoryThreshold    float64 `yaml:"memory_threshold"`

	// Auto-scaling
	MinWorkers        int           `yaml:"min_workers"`
	MaxWorkers        int           `yaml:"max_workers"`
	ScaleUpCooldown   time.Duration `yaml:"scale_up_cooldown"`
	ScaleDownCooldown time.Duration `yaml:"scale_down_cooldown"`

	// Circuit breaker
	CircuitBreakerThreshold int           `yaml:"circuit_breaker_threshold"`
	CircuitBreakerTimeout   time.Duration `yaml:"circuit_breaker_timeout"`

	// Analytics
	MetricsFlushInterval  time.Duration `yaml:"metrics_flush_interval"`
	EnableDetailedMetrics bool          `yaml:"enable_detailed_metrics"`
}

// Job represents a unit of work for the supervisor
type Job struct {
	ID           string                 `json:"id"`
	Type         string                 `json:"type"`
	Priority     int                    `json:"priority"`
	Payload      map[string]interface{} `json:"payload"`
	CreatedAt    time.Time              `json:"created_at"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Status       JobStatus              `json:"status"`
	RetryCount   int                    `json:"retry_count"`
	MaxRetries   int                    `json:"max_retries"`
	Timeout      time.Duration          `json:"timeout"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Tags         map[string]string      `json:"tags,omitempty"`
	Dependencies []string               `json:"dependencies,omitempty"`
}

// JobStatus represents the status of a job
type JobStatus int

const (
	JobStatusPending JobStatus = iota
	JobStatusRunning
	JobStatusCompleted
	JobStatusFailed
	JobStatusCancelled
	JobStatusTimeout
)

// JobResult represents the result of a completed job
type JobResult struct {
	JobID     string                 `json:"job_id"`
	Status    JobStatus              `json:"status"`
	Result    interface{}            `json:"result,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	WorkerID  string                 `json:"worker_id,omitempty"`
}

// Worker represents a worker in the supervisor
type Worker struct {
	ID            string                 `json:"id"`
	Status        WorkerStatus           `json:"status"`
	CurrentJob    *Job                   `json:"current_job,omitempty"`
	StartedAt     time.Time              `json:"started_at"`
	LastActivity  time.Time              `json:"last_activity"`
	JobsProcessed int64                  `json:"jobs_processed"`
	ErrorsCount   int64                  `json:"errors_count"`
	LoadFactor    float64                `json:"load_factor"`
	HealthScore   float64                `json:"health_score"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Tags          map[string]string      `json:"tags,omitempty"`
}

// WorkerStatus represents the status of a worker
type WorkerStatus int

const (
	WorkerStatusIdle WorkerStatus = iota
	WorkerStatusBusy
	WorkerStatusError
	WorkerStatusMaintenance
	WorkerStatusOffline
)

// EnhancedSupervisor provides advanced job supervision with analytics and context management
type EnhancedSupervisor struct {
	config  SupervisorConfig
	state   SupervisorState
	workers map[string]*Worker
	jobs    map[string]*Job
	results map[string]*JobResult
	mu      sync.RWMutex

	// Enhanced components
	contextManager  *enhancedContext.ContextManager
	analyticsEngine *analytics.AnalyticsEngine
	validator       *validation.ValidationGate
	monitor         *monitoring.MonitoringEngine
	verifier        llmverifier.VerifierInterface

	// Job management
	jobQueue    chan *Job
	resultQueue chan *JobResult
	activeJobs  map[string]*Job // Currently running jobs

	// Auto-scaling
	currentWorkers int32
	scaleUpTime    time.Time
	scaleDownTime  time.Time

	// Circuit breaker
	circuitBreaker *CircuitBreaker

	// Analytics
	metrics     map[string]analytics.AnalyticsMetric
	metricsLock sync.RWMutex

	// Shutdown
	shutdownChan chan struct{}
	shutdownWG   sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
}

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
	state           CircuitBreakerState
	failureCount    int64
	lastFailureTime time.Time
	threshold       int
	timeout         time.Duration
	mu              sync.RWMutex
}

// CircuitBreakerState represents circuit breaker state
type CircuitBreakerState int

const (
	CircuitBreakerClosed CircuitBreakerState = iota
	CircuitBreakerOpen
	CircuitBreakerHalfOpen
)

// NewEnhancedSupervisor creates a new enhanced supervisor
func NewEnhancedSupervisor(
	config SupervisorConfig,
	contextManager enhancedContext.ContextManagerInterface,
	analyticsEngine analytics.AnalyticsEngineInterface,
	validator validation.ValidationGateInterface,
	monitor monitoring.MonitoringInterface,
	verifier llmverifier.VerifierInterface,
) *EnhancedSupervisor {
	ctx, cancel := context.WithCancel(context.Background())

	supervisor := &EnhancedSupervisor{
		config:          config,
		state:           SupervisorStateInactive,
		workers:         make(map[string]*Worker),
		jobs:            make(map[string]*Job),
		results:         make(map[string]*JobResult),
		contextManager:  contextManager,
		analyticsEngine: analyticsEngine,
		validator:       validator,
		monitor:         monitor,
		verifier:        verifier,
		jobQueue:        make(chan *Job, 1000),
		resultQueue:     make(chan *JobResult, 1000),
		activeJobs:      make(map[string]*Job),
		metrics:         make(map[string]analytics.AnalyticsMetric),
		shutdownChan:    make(chan struct{}),
		currentWorkers:  int32(config.MinWorkers),
		ctx:             ctx,
		cancel:          cancel,
	}

	// Initialize circuit breaker
	supervisor.circuitBreaker = &CircuitBreaker{
		state:     CircuitBreakerClosed,
		threshold: config.CircuitBreakerThreshold,
		timeout:   config.CircuitBreakerTimeout,
	}

	return supervisor
}

// Start starts the enhanced supervisor
func (es *EnhancedSupervisor) Start() error {
	es.mu.Lock()
	defer es.mu.Unlock()

	if es.state == SupervisorStateActive {
		return fmt.Errorf("supervisor is already running")
	}

	es.state = SupervisorStateActive
	es.cancel()
	es.ctx, es.cancel = context.WithCancel(context.Background())

	// Start initial workers
	for i := 0; i < es.config.MinWorkers; i++ {
		worker := &Worker{
			ID:        fmt.Sprintf("worker-%d", i),
			Status:    WorkerStatusIdle,
			StartedAt: time.Now(),
			Tags:      make(map[string]string),
			Metadata:  make(map[string]interface{}),
		}
		es.workers[worker.ID] = worker
		es.shutdownWG.Add(1)
		go es.runWorker(worker)
	}

	// Start supervisor loops
	es.shutdownWG.Add(1)
	go es.runSupervisor()

	es.shutdownWG.Add(1)
	go es.runMetricsCollector()

	es.shutdownWG.Add(1)
	go es.runAutoScaler()

	es.shutdownWG.Add(1)
	go es.runHealthChecker()

	// Record startup metrics
	es.recordMetric("supervisor.start", analytics.MetricTypeGauge, 1,
		map[string]string{"state": "active"}, nil)

	log.Printf("Enhanced supervisor started with %d initial workers", es.config.MinWorkers)
	return nil
}

// Stop stops the enhanced supervisor gracefully
func (es *EnhancedSupervisor) Stop(timeout time.Duration) error {
	es.mu.Lock()
	defer es.mu.Unlock()

	if es.state != SupervisorStateActive {
		return fmt.Errorf("supervisor is not running")
	}

	es.state = SupervisorStateMaintenance
	es.cancel()

	// Signal shutdown
	close(es.shutdownChan)

	// Wait for graceful shutdown with timeout
	done := make(chan struct{})
	go func() {
		es.shutdownWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("Enhanced supervisor stopped gracefully")
	case <-time.After(timeout):
		log.Printf("Enhanced supervisor stop timeout, forcing shutdown")
	}

	es.state = SupervisorStateInactive

	// Record shutdown metrics
	es.recordMetric("supervisor.stop", analytics.MetricTypeGauge, 1,
		map[string]string{"state": "inactive"}, nil)

	return nil
}

// SubmitJob submits a new job to the supervisor
func (es *EnhancedSupervisor) SubmitJob(job *Job) error {
	// Validate job
	if err := es.validator.ValidateJob(job); err != nil {
		return fmt.Errorf("job validation failed: %w", err)
	}

	// Add job to queue with context
	if es.contextManager != nil {
		if err := es.contextManager.AddMessage("supervisor",
			fmt.Sprintf("Job submitted: %s", job.ID),
			map[string]interface{}{
				"job_id":   job.ID,
				"job_type": job.Type,
				"priority": job.Priority,
			}); err != nil {
			log.Printf("Failed to add job to context: %v", err)
		}
	}

	// Record submission metric
	es.recordMetric("job.submitted", analytics.MetricTypeCounter, 1,
		map[string]string{
			"job_type": job.Type,
			"priority": fmt.Sprintf("%d", job.Priority),
		}, nil)

	select {
	case es.jobQueue <- job:
		es.mu.Lock()
		es.jobs[job.ID] = job
		es.mu.Unlock()
		return nil
	case <-es.ctx.Done():
		return fmt.Errorf("supervisor is shutting down")
	case <-time.After(es.config.JobTimeout):
		return fmt.Errorf("job submission timeout")
	}
}

// GetJobStatus returns the status of a specific job
func (es *EnhancedSupervisor) GetJobStatus(jobID string) (*Job, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	job, exists := es.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	// Return a copy to prevent external modification
	jobCopy := *job
	return &jobCopy, nil
}

// GetJobs returns all jobs with optional filtering
func (es *EnhancedSupervisor) GetJobs(filter JobFilter) ([]*Job, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	var filtered []*Job
	for _, job := range es.jobs {
		if filter.matches(job) {
			jobCopy := *job
			filtered = append(filtered, &jobCopy)
		}
	}

	return filtered, nil
}

// GetWorkers returns all workers
func (es *EnhancedSupervisor) GetWorkers() ([]*Worker, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()

	workers := make([]*Worker, 0, len(es.workers))
	for _, worker := range es.workers {
		workerCopy := *worker
		workers = append(workers, &workerCopy)
	}

	return workers, nil
}

// GetSupervisorStatus returns the current status of the supervisor
func (es *EnhancedSupervisor) GetSupervisorStatus() SupervisorStatus {
	es.mu.RLock()
	defer es.mu.RUnlock()

	status := SupervisorStatus{
		State:          es.state,
		WorkerCount:    int(es.currentWorkers),
		ActiveJobs:     len(es.activeJobs),
		QueuedJobs:     len(es.jobQueue),
		CompletedJobs:  0,
		FailedJobs:     0,
		LoadFactor:     es.calculateLoadFactor(),
		HealthScore:    es.calculateHealthScore(),
		CircuitBreaker: es.circuitBreaker.state,
		Uptime:         time.Since(time.Time{}), // Will be calculated properly
	}

	// Count completed and failed jobs
	for _, job := range es.jobs {
		switch job.Status {
		case JobStatusCompleted:
			status.CompletedJobs++
		case JobStatusFailed, JobStatusTimeout:
			status.FailedJobs++
		}
	}

	return status
}

// SupervisorStatus represents the overall status of the supervisor
type SupervisorStatus struct {
	State          SupervisorState     `json:"state"`
	WorkerCount    int                 `json:"worker_count"`
	ActiveJobs     int                 `json:"active_jobs"`
	QueuedJobs     int                 `json:"queued_jobs"`
	CompletedJobs  int                 `json:"completed_jobs"`
	FailedJobs     int                 `json:"failed_jobs"`
	LoadFactor     float64             `json:"load_factor"`
	HealthScore    float64             `json:"health_score"`
	CircuitBreaker CircuitBreakerState `json:"circuit_breaker"`
	Uptime         time.Duration       `json:"uptime"`
}

// JobFilter provides filtering options for jobs
type JobFilter struct {
	Status   *JobStatus        `json:"status,omitempty"`
	Type     *string           `json:"type,omitempty"`
	Priority *int              `json:"priority,omitempty"`
	From     *time.Time        `json:"from,omitempty"`
	To       *time.Time        `json:"to,omitempty"`
	Tags     map[string]string `json:"tags,omitempty"`
}

// matches checks if a job matches the filter criteria
func (f JobFilter) matches(job *Job) bool {
	if f.Status != nil && job.Status != *f.Status {
		return false
	}

	if f.Type != nil && job.Type != *f.Type {
		return false
	}

	if f.Priority != nil && job.Priority != *f.Priority {
		return false
	}

	if f.From != nil && job.CreatedAt.Before(*f.From) {
		return false
	}

	if f.To != nil && job.CreatedAt.After(*f.To) {
		return false
	}

	if len(f.Tags) > 0 {
		for key, value := range f.Tags {
			if jobValue, exists := job.Tags[key]; !exists || jobValue != value {
				return false
			}
		}
	}

	return true
}
