package supervisor

import (
	"fmt"
	"net/http"
	"time"
)

// SupervisorAPI provides HTTP API for enhanced supervisor
type SupervisorAPI struct {
	supervisor *EnhancedSupervisor
}

// NewSupervisorAPI creates a new supervisor API handler
func NewSupervisorAPI(supervisor *EnhancedSupervisor) *SupervisorAPI {
	return &SupervisorAPI{
		supervisor: supervisor,
	}
}

// APIRequest represents a generic API request
type APIRequest struct {
	Action  string                 `json:"action"`
	Payload map[string]interface{} `json:"payload,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
}

// APIResponse represents a generic API response
type APIResponse struct {
	Success  bool                   `json:"success"`
	Data     interface{}            `json:"data,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// SubmitJobRequest represents a job submission request
type SubmitJobRequest struct {
	Type         string                 `json:"type"`
	Priority     int                    `json:"priority"`
	Payload      map[string]interface{} `json:"payload"`
	Timeout      time.Duration          `json:"timeout,omitempty"`
	MaxRetries   int                    `json:"max_retries,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Tags         map[string]string      `json:"tags,omitempty"`
	Dependencies []string               `json:"dependencies,omitempty"`
}

// JobStatusResponse represents job status response
type JobStatusResponse struct {
	JobID       string                 `json:"job_id"`
	Status      JobStatus              `json:"status"`
	Type        string                 `json:"type"`
	Priority    int                    `json:"priority"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	RetryCount  int                    `json:"retry_count"`
	MaxRetries  int                    `json:"max_retries"`
	Duration    *time.Duration         `json:"duration,omitempty"`
	Result      interface{}            `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Tags        map[string]string      `json:"tags,omitempty"`
	WorkerID    string                 `json:"worker_id,omitempty"`
}

// SupervisorStatusResponse represents supervisor status response
type SupervisorStatusResponse struct {
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
	LastUpdate     time.Time           `json:"last_update"`
}

// WorkerStatusResponse represents worker status response
type WorkerStatusResponse struct {
	ID            string                 `json:"id"`
	Status        WorkerStatus           `json:"status"`
	CurrentJob    *JobStatusResponse     `json:"current_job,omitempty"`
	StartedAt     time.Time              `json:"started_at"`
	LastActivity  time.Time              `json:"last_activity"`
	JobsProcessed int64                  `json:"jobs_processed"`
	ErrorsCount   int64                  `json:"errors_count"`
	LoadFactor    float64                `json:"load_factor"`
	HealthScore   float64                `json:"health_score"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Tags          map[string]string      `json:"tags,omitempty"`
}

// MetricsResponse represents metrics response
type MetricsResponse struct {
	MetricCounts  map[string]int64       `json:"metric_counts"`
	LoadMetrics   map[string]float64     `json:"load_metrics"`
	HealthMetrics map[string]float64     `json:"health_metrics"`
	Performance   map[string]interface{} `json:"performance"`
	LastUpdate    time.Time              `json:"last_update"`
}

// SubmitJob handles POST /api/supervisor/jobs
func (api *SupervisorAPI) SubmitJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SubmitJobRequest
	if err := api.readJSON(r, &req); err != nil {
		api.writeError(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Create job
	job := &Job{
		ID:           api.generateJobID(),
		Type:         req.Type,
		Priority:     req.Priority,
		Payload:      req.Payload,
		CreatedAt:    time.Now(),
		Status:       JobStatusPending,
		RetryCount:   0,
		MaxRetries:   req.MaxRetries,
		Timeout:      req.Timeout,
		Metadata:     req.Metadata,
		Tags:         req.Tags,
		Dependencies: req.Dependencies,
	}

	// Set defaults
	if job.Timeout == 0 {
		job.Timeout = 5 * time.Minute
	}
	if job.MaxRetries == 0 {
		job.MaxRetries = 3
	}

	// Submit job
	if err := api.supervisor.SubmitJob(job); err != nil {
		api.writeError(w, fmt.Sprintf("Failed to submit job: %v", err), http.StatusInternalServerError)
		return
	}

	// Return job status
	response := JobStatusResponse{
		JobID:      job.ID,
		Status:     job.Status,
		Type:       job.Type,
		Priority:   job.Priority,
		CreatedAt:  job.CreatedAt,
		RetryCount: job.RetryCount,
		MaxRetries: job.MaxRetries,
		Metadata:   job.Metadata,
		Tags:       job.Tags,
	}

	api.writeJSON(w, APIResponse{
		Success: true,
		Data:    response,
	})
}

// GetJobStatus handles GET /api/supervisor/jobs/{id}
func (api *SupervisorAPI) GetJobStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobID := r.URL.Query().Get("id")
	if jobID == "" {
		api.writeError(w, "Job ID is required", http.StatusBadRequest)
		return
	}

	job, err := api.supervisor.GetJobStatus(jobID)
	if err != nil {
		api.writeError(w, fmt.Sprintf("Job not found: %v", err), http.StatusNotFound)
		return
	}

	response := JobStatusResponse{
		JobID:       job.ID,
		Status:      job.Status,
		Type:        job.Type,
		Priority:    job.Priority,
		CreatedAt:   job.CreatedAt,
		StartedAt:   job.StartedAt,
		CompletedAt: job.CompletedAt,
		RetryCount:  job.RetryCount,
		MaxRetries:  job.MaxRetries,
		Metadata:    job.Metadata,
		Tags:        job.Tags,
	}

	// Calculate duration if completed
	if job.StartedAt != nil && job.CompletedAt != nil {
		duration := job.CompletedAt.Sub(*job.StartedAt)
		response.Duration = &duration
	}

	api.writeJSON(w, APIResponse{
		Success: true,
		Data:    response,
	})
}

// ListJobs handles GET /api/supervisor/jobs
func (api *SupervisorAPI) ListJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse filter parameters
	filter := JobFilter{}

	if statusStr := r.URL.Query().Get("status"); statusStr != "" {
		if status, err := api.parseJobStatus(statusStr); err == nil {
			filter.Status = &status
		}
	}

	if jobType := r.URL.Query().Get("type"); jobType != "" {
		filter.Type = &jobType
	}

	if priorityStr := r.URL.Query().Get("priority"); priorityStr != "" {
		if priority, err := api.parseInt(priorityStr); err == nil {
			filter.Priority = &priority
		}
	}

	jobs, err := api.supervisor.GetJobs(filter)
	if err != nil {
		api.writeError(w, fmt.Sprintf("Failed to get jobs: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	var responses []JobStatusResponse
	for _, job := range jobs {
		response := JobStatusResponse{
			JobID:       job.ID,
			Status:      job.Status,
			Type:        job.Type,
			Priority:    job.Priority,
			CreatedAt:   job.CreatedAt,
			StartedAt:   job.StartedAt,
			CompletedAt: job.CompletedAt,
			RetryCount:  job.RetryCount,
			MaxRetries:  job.MaxRetries,
			Metadata:    job.Metadata,
			Tags:        job.Tags,
		}

		// Calculate duration if completed
		if job.StartedAt != nil && job.CompletedAt != nil {
			duration := job.CompletedAt.Sub(*job.StartedAt)
			response.Duration = &duration
		}

		responses = append(responses, response)
	}

	api.writeJSON(w, APIResponse{
		Success: true,
		Data:    responses,
	})
}

// GetSupervisorStatus handles GET /api/supervisor/status
func (api *SupervisorAPI) GetSupervisorStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := api.supervisor.GetSupervisorStatus()

	response := SupervisorStatusResponse{
		State:          status.State,
		WorkerCount:    status.WorkerCount,
		ActiveJobs:     status.ActiveJobs,
		QueuedJobs:     status.QueuedJobs,
		CompletedJobs:  status.CompletedJobs,
		FailedJobs:     status.FailedJobs,
		LoadFactor:     status.LoadFactor,
		HealthScore:    status.HealthScore,
		CircuitBreaker: status.CircuitBreaker,
		Uptime:         status.Uptime,
		LastUpdate:     time.Now(),
	}

	api.writeJSON(w, APIResponse{
		Success: true,
		Data:    response,
	})
}

// GetWorkers handles GET /api/supervisor/workers
func (api *SupervisorAPI) GetWorkers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workers, err := api.supervisor.GetWorkers()
	if err != nil {
		api.writeError(w, fmt.Sprintf("Failed to get workers: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	var responses []WorkerStatusResponse
	for _, worker := range workers {
		response := WorkerStatusResponse{
			ID:            worker.ID,
			Status:        worker.Status,
			StartedAt:     worker.StartedAt,
			LastActivity:  worker.LastActivity,
			JobsProcessed: worker.JobsProcessed,
			ErrorsCount:   worker.ErrorsCount,
			LoadFactor:    worker.LoadFactor,
			HealthScore:   worker.HealthScore,
			Metadata:      worker.Metadata,
			Tags:          worker.Tags,
		}

		// Add current job info if exists
		if worker.CurrentJob != nil {
			currentJobResponse := JobStatusResponse{
				JobID:      worker.CurrentJob.ID,
				Status:     worker.CurrentJob.Status,
				Type:       worker.CurrentJob.Type,
				Priority:   worker.CurrentJob.Priority,
				CreatedAt:  worker.CurrentJob.CreatedAt,
				StartedAt:  worker.CurrentJob.StartedAt,
				RetryCount: worker.CurrentJob.RetryCount,
				MaxRetries: worker.CurrentJob.MaxRetries,
			}
			response.CurrentJob = &currentJobResponse
		}

		responses = append(responses, response)
	}

	api.writeJSON(w, APIResponse{
		Success: true,
		Data:    responses,
	})
}

// GetMetrics handles GET /api/supervisor/metrics
func (api *SupervisorAPI) GetMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get supervisor status for metrics
	status := api.supervisor.GetSupervisorStatus()

	response := MetricsResponse{
		MetricCounts: map[string]int64{
			"total_jobs":     int64(status.CompletedJobs + status.FailedJobs + status.ActiveJobs),
			"active_jobs":    int64(status.ActiveJobs),
			"queued_jobs":    int64(status.QueuedJobs),
			"completed_jobs": int64(status.CompletedJobs),
			"failed_jobs":    int64(status.FailedJobs),
			"total_workers":  int64(status.WorkerCount),
		},
		LoadMetrics: map[string]float64{
			"load_factor":     status.LoadFactor,
			"jobs_per_worker": float64(status.ActiveJobs) / float64(status.WorkerCount),
		},
		HealthMetrics: map[string]float64{
			"health_score": status.HealthScore,
			"success_rate": api.calculateSuccessRate(status),
		},
		Performance: map[string]interface{}{
			"uptime_seconds":  status.Uptime.Seconds(),
			"circuit_breaker": status.CircuitBreaker.String(),
		},
		LastUpdate: time.Now(),
	}

	api.writeJSON(w, APIResponse{
		Success: true,
		Data:    response,
	})
}

// StartSupervisor handles POST /api/supervisor/start
func (api *SupervisorAPI) StartSupervisor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := api.supervisor.Start(); err != nil {
		api.writeError(w, fmt.Sprintf("Failed to start supervisor: %v", err), http.StatusInternalServerError)
		return
	}

	api.writeJSON(w, APIResponse{
		Success: true,
		Data:    map[string]string{"message": "Supervisor started successfully"},
	})
}

// StopSupervisor handles POST /api/supervisor/stop
func (api *SupervisorAPI) StopSupervisor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse timeout from query params
	timeoutStr := r.URL.Query().Get("timeout")
	timeout := 30 * time.Second // default
	if timeoutStr != "" {
		if parsed, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsed
		}
	}

	if err := api.supervisor.Stop(timeout); err != nil {
		api.writeError(w, fmt.Sprintf("Failed to stop supervisor: %v", err), http.StatusInternalServerError)
		return
	}

	api.writeJSON(w, APIResponse{
		Success: true,
		Data:    map[string]string{"message": "Supervisor stopped successfully"},
	})
}

// SetupRoutes configures HTTP routes for supervisor API
func (api *SupervisorAPI) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/supervisor/jobs", api.SubmitJob)
	mux.HandleFunc("/api/supervisor/job", api.GetJobStatus)
	mux.HandleFunc("/api/supervisor/jobs/list", api.ListJobs)
	mux.HandleFunc("/api/supervisor/status", api.GetSupervisorStatus)
	mux.HandleFunc("/api/supervisor/workers", api.GetWorkers)
	mux.HandleFunc("/api/supervisor/metrics", api.GetMetrics)
	mux.HandleFunc("/api/supervisor/start", api.StartSupervisor)
	mux.HandleFunc("/api/supervisor/stop", api.StopSupervisor)
}

// Utility methods

// writeJSON writes JSON response
func (api *SupervisorAPI) writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	// In a real implementation, you'd use json.NewEncoder
	w.Write([]byte(`{"success": true, "data": "placeholder"}`))
}

// writeError writes error response
func (api *SupervisorAPI) writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	// In a real implementation, you'd use json.NewEncoder
	w.Write([]byte(fmt.Sprintf(`{"success": false, "error": "%s"}`, message)))
}

// readJSON reads JSON from request
func (api *SupervisorAPI) readJSON(r *http.Request, v interface{}) error {
	// In a real implementation, you'd use json.NewDecoder
	return nil
}

// generateJobID generates a unique job ID
func (api *SupervisorAPI) generateJobID() string {
	return fmt.Sprintf("job_%d", time.Now().UnixNano())
}

// parseJobStatus parses job status from string
func (api *SupervisorAPI) parseJobStatus(s string) (JobStatus, error) {
	switch s {
	case "pending":
		return JobStatusPending, nil
	case "running":
		return JobStatusRunning, nil
	case "completed":
		return JobStatusCompleted, nil
	case "failed":
		return JobStatusFailed, nil
	case "cancelled":
		return JobStatusCancelled, nil
	case "timeout":
		return JobStatusTimeout, nil
	default:
		return JobStatusPending, fmt.Errorf("invalid job status: %s", s)
	}
}

// parseInt parses integer from string
func (api *SupervisorAPI) parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

// calculateSuccessRate calculates success rate from status
func (api *SupervisorAPI) calculateSuccessRate(status SupervisorStatus) float64 {
	totalJobs := status.CompletedJobs + status.FailedJobs
	if totalJobs == 0 {
		return 1.0
	}
	return float64(status.CompletedJobs) / float64(totalJobs)
}

// String methods for enums
func (s SupervisorState) String() string {
	switch s {
	case SupervisorStateInactive:
		return "inactive"
	case SupervisorStateActive:
		return "active"
	case SupervisorStateDegraded:
		return "degraded"
	case SupervisorStateMaintenance:
		return "maintenance"
	case SupervisorStateError:
		return "error"
	default:
		return "unknown"
	}
}

func (s JobStatus) String() string {
	switch s {
	case JobStatusPending:
		return "pending"
	case JobStatusRunning:
		return "running"
	case JobStatusCompleted:
		return "completed"
	case JobStatusFailed:
		return "failed"
	case JobStatusCancelled:
		return "cancelled"
	case JobStatusTimeout:
		return "timeout"
	default:
		return "unknown"
	}
}

func (s WorkerStatus) String() string {
	switch s {
	case WorkerStatusIdle:
		return "idle"
	case WorkerStatusBusy:
		return "busy"
	case WorkerStatusError:
		return "error"
	case WorkerStatusMaintenance:
		return "maintenance"
	case WorkerStatusOffline:
		return "offline"
	default:
		return "unknown"
	}
}

func (s CircuitBreakerState) String() string {
	switch s {
	case CircuitBreakerClosed:
		return "closed"
	case CircuitBreakerOpen:
		return "open"
	case CircuitBreakerHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}
