package supervisor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"llm-verifier/enhanced/analytics"
	"llm-verifier/enhanced/context"
	"llm-verifier/enhanced/validation"
	llmverifier "llm-verifier/llmverifier"
	"llm-verifier/monitoring"
)

// EnhancedSupervisor provides advanced job supervision with analytics and context management
type EnhancedSupervisor struct {
	config  SupervisorConfig
	state   SupervisorState
	workers map[string]*EnhancedWorker
	jobs    map[string]*Job
	results map[string]*JobResult
	mu      sync.RWMutex

	// Enhanced components
	contextManager  context.ContextManagerInterface
	analyticsEngine analytics.AnalyticsEngineInterface
	validator       validation.ValidationGateInterface
	monitor         monitoring.MonitoringInterface
	verifier        llmverifier.VerifierInterface

	// Job management
	jobQueue    chan *Job
	resultQueue chan *JobResult
	activeJobs  map[string]*Job

	// Auto-scaling
	currentWorkers int32
	scaleUpTime    time.Time
	scaleDownTime  time.Time

	// Circuit breaker
	circuitBreaker *CircuitBreaker

	// Shutdown
	shutdownChan chan struct{}
	shutdownWG   sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
}

// EnhancedWorker represents a worker in enhanced supervisor
type EnhancedWorker struct {
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

// NewEnhancedSupervisor creates a new enhanced supervisor
func NewEnhancedSupervisor(
	config SupervisorConfig,
	contextManager context.ContextManagerInterface,
	analyticsEngine analytics.AnalyticsEngineInterface,
	validator validation.ValidationGateInterface,
	monitor monitoring.MonitoringInterface,
	verifier llmverifier.VerifierInterface,
) *EnhancedSupervisor {
	ctx, cancel := context.WithCancel(context.Background())

	supervisor := &EnhancedSupervisor{
		config:          config,
		state:           SupervisorStateInactive,
		workers:         make(map[string]*EnhancedWorker),
		jobs:            make(map[string]*Job),
		results:         make(map[string]*JobResult),
		contextManager:  contextManager,
		analyticsEngine: analyticsEngine,
		validator:       validator,
		monitor:         monitor,
		jobQueue:        make(chan *Job, 1000),
		resultQueue:     make(chan *JobResult, 1000),
		activeJobs:      make(map[string]*Job),
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

	// Start initial workers
	for i := 0; i < es.config.MinWorkers; i++ {
		worker := &EnhancedWorker{
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
	go es.runAutoScaler()

	es.shutdownWG.Add(1)
	go es.runHealthChecker()

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
	return nil
}

// SubmitJob submits a new job to the supervisor
func (es *EnhancedSupervisor) SubmitJob(job *Job) error {
	// Validate job
	if es.validator != nil {
		if err := es.validator.ValidateJob(job); err != nil {
			return fmt.Errorf("job validation failed: %w", err)
		}
	}

	// Add job to context if available
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

// runWorker executes jobs from the queue
func (es *EnhancedSupervisor) runWorker(worker *EnhancedWorker) {
	defer es.shutdownWG.Done()

	for {
		select {
		case <-es.ctx.Done():
			return

		case job := <-es.jobQueue:
			es.processJob(worker, job)
		}
	}
}

// processJob processes a single job
func (es *EnhancedSupervisor) processJob(worker *EnhancedWorker, job *Job) {
	startTime := time.Now()

	// Update worker status
	worker.Status = WorkerStatusBusy
	worker.CurrentJob = job
	worker.LastActivity = startTime

	// Update job status
	job.Status = JobStatusRunning
	now := time.Now()
	job.StartedAt = &now

	es.mu.Lock()
	es.activeJobs[job.ID] = job
	es.mu.Unlock()

	// Process job with timeout
	resultChan := make(chan *JobResult, 1)
	go func() {
		result := es.executeJob(job, worker)
		resultChan <- result
	}()

	var result *JobResult
	select {
	case result = <-resultChan:
		// Job completed normally
	case <-time.After(job.Timeout):
		// Job timed out
		result = &JobResult{
			JobID:     job.ID,
			Status:    JobStatusTimeout,
			Error:     "job execution timeout",
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
			WorkerID:  worker.ID,
		}
	}

	// Record metrics
	if es.analyticsEngine != nil {
		es.recordJobMetrics(job, result, worker)
	}

	// Update worker and job status
	worker.Status = WorkerStatusIdle
	worker.CurrentJob = nil
	atomic.AddInt64(&worker.JobsProcessed, 1)

	if result.Status == JobStatusFailed || result.Status == JobStatusTimeout {
		atomic.AddInt64(&worker.ErrorsCount, 1)
	}

	job.Status = result.Status
	completedAt := time.Now()
	job.CompletedAt = &completedAt

	es.mu.Lock()
	delete(es.activeJobs, job.ID)
	es.results[job.ID] = result
	es.mu.Unlock()

	// Add result to queue
	select {
	case es.resultQueue <- result:
	default:
		log.Printf("Result queue full, dropping result for job %s", job.ID)
	}

	// Add to context
	if es.contextManager != nil {
		if err := es.contextManager.AddMessage("supervisor",
			fmt.Sprintf("Job completed: %s with status: %v", job.ID, result.Status),
			map[string]interface{}{
				"job_id":    job.ID,
				"status":    result.Status,
				"duration":  result.Duration,
				"worker_id": worker.ID,
			}); err != nil {
			log.Printf("Failed to add job completion to context: %v", err)
		}
	}
}

// executeJob executes the actual job logic
func (es *EnhancedSupervisor) executeJob(job *Job, worker *EnhancedWorker) *JobResult {
	startTime := time.Now()

	// Check circuit breaker
	if es.circuitBreaker != nil && es.circuitBreaker.IsOpen() {
		return &JobResult{
			JobID:     job.ID,
			Status:    JobStatusFailed,
			Error:     "circuit breaker is open",
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
			WorkerID:  worker.ID,
		}
	}

	// Execute job based on type
	var result interface{}
	var err error

	switch job.Type {
	case "verification":
		result, err = es.executeVerificationJob(job)
	case "analysis":
		result, err = es.executeAnalysisJob(job)
	case "context_summary":
		result, err = es.executeContextSummaryJob(job)
	default:
		result, err = es.executeGenericJob(job)
	}

	// Update circuit breaker based on result
	if es.circuitBreaker != nil {
		if err != nil {
			es.circuitBreaker.RecordFailure()
		} else {
			es.circuitBreaker.RecordSuccess()
		}
	}

	jobResult := &JobResult{
		JobID:     job.ID,
		Status:    JobStatusCompleted,
		Result:    result,
		Duration:  time.Since(startTime),
		Timestamp: time.Now(),
		WorkerID:  worker.ID,
	}

	if err != nil {
		jobResult.Status = JobStatusFailed
		jobResult.Error = err.Error()
	}

	return jobResult
}

// executeVerificationJob executes a verification job
func (es *EnhancedSupervisor) executeVerificationJob(job *Job) (interface{}, error) {
	// Extract verification parameters
	modelName, ok := job.Payload["model_name"].(string)
	if !ok {
		return nil, fmt.Errorf("model_name required for verification job")
	}

	// Use verifier to check model
	if es.verifier == nil {
		return nil, fmt.Errorf("verifier not available")
	}

	// This would use the actual verifier logic
	// For now, return a mock result
	result := map[string]interface{}{
		"model_name": modelName,
		"verified":   true,
		"timestamp":  time.Now(),
	}

	return result, nil
}

// executeAnalysisJob executes an analysis job
func (es *EnhancedSupervisor) executeAnalysisJob(job *Job) (interface{}, error) {
	// Extract analysis parameters
	metricName, ok := job.Payload["metric_name"].(string)
	if !ok {
		return nil, fmt.Errorf("metric_name required for analysis job")
	}

	// Use analytics engine for analysis
	if es.analyticsEngine == nil {
		return nil, fmt.Errorf("analytics engine not available")
	}

	// This would use actual analytics logic
	result := map[string]interface{}{
		"metric_name":   metricName,
		"analysis_type": "trend_analysis",
		"timestamp":     time.Now(),
	}

	return result, nil
}

// executeContextSummaryJob executes a context summary job
func (es *EnhancedSupervisor) executeContextSummaryJob(job *Job) (interface{}, error) {
	// Use context manager for summary
	if es.contextManager == nil {
		return nil, fmt.Errorf("context manager not available")
	}

	// This would use actual context logic
	result := map[string]interface{}{
		"job_type":  "context_summary",
		"summary":   "Context summary job completed",
		"timestamp": time.Now(),
	}

	return result, nil
}

// executeGenericJob executes a generic job
func (es *EnhancedSupervisor) executeGenericJob(job *Job) (interface{}, error) {
	// Default job execution
	result := map[string]interface{}{
		"job_type":  job.Type,
		"status":    "completed",
		"payload":   job.Payload,
		"timestamp": time.Now(),
	}

	return result, nil
}

// runSupervisor manages the main supervisor loop
func (es *EnhancedSupervisor) runSupervisor() {
	defer es.shutdownWG.Done()

	ticker := time.NewTicker(es.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-es.ctx.Done():
			return

		case result := <-es.resultQueue:
			es.handleJobResult(result)

		case <-ticker.C:
			es.performMaintenance()
		}
	}
}

// handleJobResult processes completed job results
func (es *EnhancedSupervisor) handleJobResult(result *JobResult) {
	// Handle job retries if needed
	if result.Status == JobStatusFailed {
		job, exists := es.jobs[result.JobID]
		if exists && job.RetryCount < job.MaxRetries {
			job.RetryCount++
			job.Status = JobStatusPending

			// Schedule retry with backoff
			go func() {
				time.Sleep(es.config.RetryBackoff * time.Duration(job.RetryCount))
				select {
				case es.jobQueue <- job:
				case <-es.ctx.Done():
				}
			}()

			return
		}
	}

	// Clean up completed jobs
	es.mu.Lock()
	delete(es.jobs, result.JobID)
	es.mu.Unlock()
}

// performMaintenance performs periodic maintenance tasks
func (es *EnhancedSupervisor) performMaintenance() {
	// Clean up old jobs and results
	cutoff := time.Now().Add(-24 * time.Hour)

	es.mu.Lock()
	for jobID, job := range es.jobs {
		if job.CreatedAt.Before(cutoff) {
			delete(es.jobs, jobID)
		}
	}

	for resultID, result := range es.results {
		if result.Timestamp.Before(cutoff) {
			delete(es.results, resultID)
		}
	}
	es.mu.Unlock()

	// Update worker health scores
	es.updateWorkerHealth()
}

// updateWorkerHealth updates health scores for all workers
func (es *EnhancedSupervisor) updateWorkerHealth() {
	es.mu.RLock()
	defer es.mu.RUnlock()

	for _, worker := range es.workers {
		// Calculate health score based on various factors
		jobsProcessed := atomic.LoadInt64(&worker.JobsProcessed)
		errorsCount := atomic.LoadInt64(&worker.ErrorsCount)

		var successRate float64
		if jobsProcessed > 0 {
			successRate = float64(jobsProcessed-errorsCount) / float64(jobsProcessed)
		}

		// Update load factor based on recent activity
		timeSinceActivity := time.Since(worker.LastActivity)
		loadFactor := 1.0 - (timeSinceActivity.Hours() / 24.0) // Decay over 24 hours
		if loadFactor < 0 {
			loadFactor = 0
		}

		worker.LoadFactor = loadFactor
		worker.HealthScore = (successRate * 0.7) + (loadFactor * 0.3)
	}
}

// runAutoScaler manages automatic worker scaling
func (es *EnhancedSupervisor) runAutoScaler() {
	defer es.shutdownWG.Done()

	if !es.config.EnableAutoScaling {
		return
	}

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-es.ctx.Done():
			return
		case <-ticker.C:
			es.checkScaling()
		}
	}
}

// checkScaling determines if scaling is needed
func (es *EnhancedSupervisor) checkScaling() {
	loadFactor := es.calculateLoadFactor()
	currentWorkers := atomic.LoadInt32(&es.currentWorkers)
	now := time.Now()

	// Scale up if needed
	if loadFactor > es.config.HighLoadThreshold &&
		int(currentWorkers) < es.config.MaxWorkers &&
		now.Sub(es.scaleUpTime) > es.config.ScaleUpCooldown {

		es.scaleUp()
		es.scaleUpTime = now
	}

	// Scale down if needed
	if loadFactor < es.config.LowLoadThreshold &&
		int(currentWorkers) > es.config.MinWorkers &&
		now.Sub(es.scaleDownTime) > es.config.ScaleDownCooldown {

		es.scaleDown()
		es.scaleDownTime = now
	}
}

// scaleUp adds a new worker
func (es *EnhancedSupervisor) scaleUp() {
	currentWorkers := atomic.LoadInt32(&es.currentWorkers)
	if int(currentWorkers) >= es.config.MaxWorkers {
		return
	}

	workerID := fmt.Sprintf("worker-%d", currentWorkers)
	worker := &EnhancedWorker{
		ID:        workerID,
		Status:    WorkerStatusIdle,
		StartedAt: time.Now(),
		Tags:      map[string]string{"auto_scaled": "true"},
		Metadata:  make(map[string]interface{}),
	}

	es.mu.Lock()
	es.workers[workerID] = worker
	es.mu.Unlock()

	atomic.AddInt32(&es.currentWorkers, 1)
	es.shutdownWG.Add(1)
	go es.runWorker(worker)

	log.Printf("Scaled up: added worker %s (total: %d)", workerID, currentWorkers+1)
}

// scaleDown removes a worker
func (es *EnhancedSupervisor) scaleDown() {
	currentWorkers := atomic.LoadInt32(&es.currentWorkers)
	if int(currentWorkers) <= es.config.MinWorkers {
		return
	}

	// Find an idle worker to remove
	es.mu.RLock()
	var idleWorker *EnhancedWorker
	for _, worker := range es.workers {
		if worker.Status == WorkerStatusIdle && worker.Tags["auto_scaled"] == "true" {
			idleWorker = worker
			break
		}
	}
	es.mu.RUnlock()

	if idleWorker == nil {
		return
	}

	// Remove worker
	es.mu.Lock()
	delete(es.workers, idleWorker.ID)
	es.mu.Unlock()

	atomic.AddInt32(&es.currentWorkers, -1)

	log.Printf("Scaled down: removed worker %s (total: %d)", idleWorker.ID, currentWorkers-1)
}

// runHealthChecker monitors health of the supervisor
func (es *EnhancedSupervisor) runHealthChecker() {
	defer es.shutdownWG.Done()

	ticker := time.NewTicker(es.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-es.ctx.Done():
			return
		case <-ticker.C:
			es.checkHealth()
		}
	}
}

// checkHealth performs health checks
func (es *EnhancedSupervisor) checkHealth() {
	healthScore := es.calculateHealthScore()

	// Update state based on health
	es.mu.Lock()
	switch {
	case healthScore < 0.3:
		if es.state != SupervisorStateError {
			log.Printf("Supervisor health critical: %.2f", healthScore)
			es.state = SupervisorStateError
		}
	case healthScore < 0.6:
		if es.state != SupervisorStateDegraded {
			log.Printf("Supervisor health degraded: %.2f", healthScore)
			es.state = SupervisorStateDegraded
		}
	default:
		if es.state != SupervisorStateActive {
			log.Printf("Supervisor health recovered: %.2f", healthScore)
			es.state = SupervisorStateActive
		}
	}
	es.mu.Unlock()
}

// calculateLoadFactor calculates current load factor
func (es *EnhancedSupervisor) calculateLoadFactor() float64 {
	es.mu.RLock()
	defer es.mu.RUnlock()

	currentWorkers := atomic.LoadInt32(&es.currentWorkers)
	if currentWorkers == 0 {
		return 1.0
	}

	activeJobs := len(es.activeJobs)
	queuedJobs := len(es.jobQueue)

	// Calculate load based on active jobs and queue length
	activeLoad := float64(activeJobs) / float64(currentWorkers)
	queueLoad := float64(queuedJobs) / 100.0 // Normalize queue length

	loadFactor := (activeLoad * 0.7) + (queueLoad * 0.3)
	if loadFactor > 1.0 {
		loadFactor = 1.0
	}

	return loadFactor
}

// calculateHealthScore calculates overall health score
func (es *EnhancedSupervisor) calculateHealthScore() float64 {
	es.mu.RLock()
	defer es.mu.RUnlock()

	var totalHealth float64
	var workerCount int

	for _, worker := range es.workers {
		totalHealth += worker.HealthScore
		workerCount++
	}

	if workerCount == 0 {
		return 0.0
	}

	avgWorkerHealth := totalHealth / float64(workerCount)

	// Factor in circuit breaker state
	var circuitHealth float64
	if es.circuitBreaker != nil {
		switch es.circuitBreaker.state {
		case CircuitBreakerClosed:
			circuitHealth = 1.0
		case CircuitBreakerHalfOpen:
			circuitHealth = 0.5
		case CircuitBreakerOpen:
			circuitHealth = 0.0
		}
	}

	// Combine factors
	healthScore := (avgWorkerHealth * 0.7) + (circuitHealth * 0.3)
	return healthScore
}

// recordJobMetrics records metrics for job execution
func (es *EnhancedSupervisor) recordJobMetrics(job *Job, result *JobResult, worker *EnhancedWorker) {
	if es.analyticsEngine == nil {
		return
	}

	// Record job duration metric
	es.analyticsEngine.RecordMetric(
		"job_duration",
		analytics.MetricTypeHistogram,
		result.Duration.Seconds(),
		map[string]string{
			"job_type": job.Type,
			"status":   fmt.Sprintf("%d", result.Status),
			"worker":   worker.ID,
		},
		nil,
	)

	// Record job completion metric
	es.analyticsEngine.RecordMetric(
		"job_completed",
		analytics.MetricTypeCounter,
		1,
		map[string]string{
			"job_type": job.Type,
			"status":   fmt.Sprintf("%d", result.Status),
		},
		nil,
	)
}

// CircuitBreaker implementation
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitBreakerOpen:
		return time.Since(cb.lastFailureTime) > cb.timeout
	case CircuitBreakerHalfOpen:
		return false
	default:
		return false
	}
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0
	cb.state = CircuitBreakerClosed
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.failureCount >= cb.threshold {
		cb.state = CircuitBreakerOpen
	}
}
