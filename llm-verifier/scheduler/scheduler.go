package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"llm-verifier/database"
	"llm-verifier/llmverifier"
)

// JobType defines the type of scheduled job
type JobType string

const (
	JobTypeVerification JobType = "verification"
	JobTypeExport       JobType = "export"
	JobTypeMaintenance  JobType = "maintenance"
	JobTypeCustom       JobType = "custom"
)

// JobStatus defines the status of a scheduled job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// ScheduledJob represents a scheduled job
type ScheduledJob struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Type      JobType                `json:"type"`
	Schedule  string                 `json:"schedule"` // Cron expression
	Enabled   bool                   `json:"enabled"`
	Config    map[string]interface{} `json:"config"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`

	// Runtime fields
	cronID  cron.EntryID
	lastRun *time.Time
	nextRun *time.Time
	status  JobStatus
	error   string
}

// JobExecution represents a job execution instance
type JobExecution struct {
	ID        string                 `json:"id"`
	JobID     string                 `json:"job_id"`
	StartedAt time.Time              `json:"started_at"`
	EndedAt   *time.Time             `json:"ended_at"`
	Status    JobStatus              `json:"status"`
	Error     string                 `json:"error"`
	Results   map[string]interface{} `json:"results"`
	Duration  time.Duration          `json:"duration"`
}

// Scheduler manages scheduled jobs
type Scheduler struct {
	cron       *cron.Cron
	database   *database.Database
	verifier   *llmverifier.Verifier
	jobs       map[string]*ScheduledJob
	executions map[string]*JobExecution
	mu         sync.RWMutex
	running    bool
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewScheduler creates a new scheduler
func NewScheduler(db *database.Database, verifier *llmverifier.Verifier) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		cron:       cron.New(cron.WithSeconds()),
		database:   db,
		verifier:   verifier,
		jobs:       make(map[string]*ScheduledJob),
		executions: make(map[string]*JobExecution),
		running:    false,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	// Load existing jobs from database
	if err := s.loadJobsFromDatabase(); err != nil {
		return fmt.Errorf("failed to load jobs from database: %w", err)
	}

	// Schedule all enabled jobs
	for _, job := range s.jobs {
		if job.Enabled {
			if err := s.scheduleJob(job); err != nil {
				log.Printf("Failed to schedule job %s: %v", job.Name, err)
				continue
			}
		}
	}

	// Start the cron scheduler
	s.cron.Start()
	s.running = true

	log.Printf("Scheduler started with %d jobs", len(s.jobs))
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return fmt.Errorf("scheduler is not running")
	}

	// Cancel context to stop running jobs
	s.cancel()

	// Stop cron scheduler
	ctx := s.cron.Stop()
	<-ctx.Done()

	s.running = false
	log.Printf("Scheduler stopped")
	return nil
}

// AddJob adds a new scheduled job
func (s *Scheduler) AddJob(job *ScheduledJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate job
	if err := s.validateJob(job); err != nil {
		return fmt.Errorf("invalid job: %w", err)
	}

	// Save to database
	if err := s.saveJobToDatabase(job); err != nil {
		return fmt.Errorf("failed to save job to database: %w", err)
	}

	// Add to in-memory map
	s.jobs[job.ID] = job

	// Schedule if enabled
	if job.Enabled {
		if err := s.scheduleJob(job); err != nil {
			return fmt.Errorf("failed to schedule job: %w", err)
		}
	}

	log.Printf("Added job: %s (%s)", job.Name, job.Schedule)
	return nil
}

// RemoveJob removes a scheduled job
func (s *Scheduler) RemoveJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	// Remove from cron
	if job.cronID != 0 {
		s.cron.Remove(job.cronID)
	}

	// Remove from database
	if err := s.deleteJobFromDatabase(jobID); err != nil {
		return fmt.Errorf("failed to delete job from database: %w", err)
	}

	// Remove from memory
	delete(s.jobs, jobID)

	log.Printf("Removed job: %s", job.Name)
	return nil
}

// EnableJob enables a scheduled job
func (s *Scheduler) EnableJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	if job.Enabled {
		return fmt.Errorf("job is already enabled: %s", jobID)
	}

	job.Enabled = true
	job.UpdatedAt = time.Now()

	// Update database
	if err := s.updateJobInDatabase(job); err != nil {
		return fmt.Errorf("failed to update job in database: %w", err)
	}

	// Schedule the job
	if err := s.scheduleJob(job); err != nil {
		return fmt.Errorf("failed to schedule job: %w", err)
	}

	log.Printf("Enabled job: %s", job.Name)
	return nil
}

// DisableJob disables a scheduled job
func (s *Scheduler) DisableJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	if !job.Enabled {
		return fmt.Errorf("job is already disabled: %s", jobID)
	}

	job.Enabled = false
	job.UpdatedAt = time.Now()

	// Remove from cron
	if job.cronID != 0 {
		s.cron.Remove(job.cronID)
		job.cronID = 0
	}

	// Update database
	if err := s.updateJobInDatabase(job); err != nil {
		return fmt.Errorf("failed to update job in database: %w", err)
	}

	log.Printf("Disabled job: %s", job.Name)
	return nil
}

// GetJobs returns all scheduled jobs
func (s *Scheduler) GetJobs() map[string]*ScheduledJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make(map[string]*ScheduledJob)
	for id, job := range s.jobs {
		jobs[id] = job
	}
	return jobs
}

// GetJob returns a specific job
func (s *Scheduler) GetJob(jobID string) (*ScheduledJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[jobID]
	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	return job, nil
}

// GetJobExecutions returns executions for a job
func (s *Scheduler) GetJobExecutions(jobID string, limit int) ([]*JobExecution, error) {
	// In a real implementation, this would query the database
	// For now, return from in-memory map
	s.mu.RLock()
	defer s.mu.RUnlock()

	var executions []*JobExecution
	for _, execution := range s.executions {
		if execution.JobID == jobID {
			executions = append(executions, execution)
		}
	}

	// Return most recent executions
	if len(executions) > limit {
		executions = executions[len(executions)-limit:]
	}

	return executions, nil
}

// scheduleJob schedules a job with cron
func (s *Scheduler) scheduleJob(job *ScheduledJob) error {
	// Remove existing cron entry if any
	if job.cronID != 0 {
		s.cron.Remove(job.cronID)
	}

	// Add to cron
	id, err := s.cron.AddFunc(job.Schedule, func() {
		s.executeJob(job)
	})
	if err != nil {
		return fmt.Errorf("failed to schedule job: %w", err)
	}

	job.cronID = id
	job.status = JobStatusPending
	job.nextRun = s.getNextRunTime(job.Schedule)

	return nil
}

// executeJob executes a scheduled job
func (s *Scheduler) executeJob(job *ScheduledJob) {
	execution := &JobExecution{
		ID:        generateExecutionID(),
		JobID:     job.ID,
		StartedAt: time.Now(),
		Status:    JobStatusRunning,
	}

	s.mu.Lock()
	s.executions[execution.ID] = execution
	job.status = JobStatusRunning
	job.lastRun = &execution.StartedAt
	s.mu.Unlock()

	log.Printf("Executing job: %s (%s)", job.Name, job.Type)

	var err error
	var results map[string]interface{}

	// Execute job based on type
	switch job.Type {
	case JobTypeVerification:
		results, err = s.executeVerificationJob(job)
	case JobTypeExport:
		results, err = s.executeExportJob(job)
	case JobTypeMaintenance:
		results, err = s.executeMaintenanceJob(job)
	case JobTypeCustom:
		results, err = s.executeCustomJob(job)
	default:
		err = fmt.Errorf("unknown job type: %s", job.Type)
	}

	// Update execution
	now := time.Now()
	execution.EndedAt = &now
	execution.Duration = execution.EndedAt.Sub(execution.StartedAt)
	execution.Results = results

	if err != nil {
		execution.Status = JobStatusFailed
		execution.Error = err.Error()
		log.Printf("Job failed: %s - %v", job.Name, err)
	} else {
		execution.Status = JobStatusCompleted
		log.Printf("Job completed: %s", job.Name)
	}

	s.mu.Lock()
	job.status = JobStatusPending
	job.nextRun = s.getNextRunTime(job.Schedule)
	s.mu.Unlock()

	// Save execution to database (in a real implementation)
	// For now, just keep in memory
}

// executeVerificationJob executes a verification job
func (s *Scheduler) executeVerificationJob(job *ScheduledJob) (map[string]interface{}, error) {
	log.Printf("Running scheduled verification for job: %s", job.Name)

	// Run verification
	results, err := s.verifier.Verify()
	if err != nil {
		return nil, fmt.Errorf("verification failed: %w", err)
	}

	// Process results
	totalModels := len(results)
	successfulModels := 0
	failedModels := 0

	for _, result := range results {
		if result.Error == "" {
			successfulModels++
		} else {
			failedModels++
		}
	}

	return map[string]interface{}{
		"total_models":      totalModels,
		"successful_models": successfulModels,
		"failed_models":     failedModels,
		"success_rate":      float64(successfulModels) / float64(totalModels) * 100,
	}, nil
}

// executeExportJob executes an export job
func (s *Scheduler) executeExportJob(job *ScheduledJob) (map[string]interface{}, error) {
	log.Printf("Running scheduled export for job: %s", job.Name)

	// Get export format from job config
	format, ok := job.Config["format"].(string)
	if !ok {
		format = "json"
	}

	outputPath, ok := job.Config["output_path"].(string)
	if !ok {
		outputPath = "./scheduled_exports"
	}

	// Perform export (simplified - would need actual config)
	err := llmverifier.ExportConfig(nil, format, outputPath)
	if err != nil {
		return nil, fmt.Errorf("export failed: %w", err)
	}

	return map[string]interface{}{
		"format":      format,
		"output_path": outputPath,
		"exported_at": time.Now(),
	}, nil
}

// executeMaintenanceJob executes a maintenance job
func (s *Scheduler) executeMaintenanceJob(job *ScheduledJob) (map[string]interface{}, error) {
	log.Printf("Running scheduled maintenance for job: %s", job.Name)

	// Run database maintenance tasks
	// This would include cleanup, optimization, etc.

	return map[string]interface{}{
		"maintenance_type": "database_cleanup",
		"executed_at":      time.Now(),
	}, nil
}

// executeCustomJob executes a custom job
func (s *Scheduler) executeCustomJob(job *ScheduledJob) (map[string]interface{}, error) {
	log.Printf("Running custom job: %s", job.Name)

	// Execute custom logic based on job config
	command, ok := job.Config["command"].(string)
	if !ok {
		return nil, fmt.Errorf("custom job missing command")
	}

	// Execute command (simplified)
	log.Printf("Would execute command: %s", command)

	return map[string]interface{}{
		"command":     command,
		"executed_at": time.Now(),
	}, nil
}

// validateJob validates a scheduled job
func (s *Scheduler) validateJob(job *ScheduledJob) error {
	if job.Name == "" {
		return fmt.Errorf("job name is required")
	}
	if job.Schedule == "" {
		return fmt.Errorf("job schedule is required")
	}

	// Validate cron expression (allow both 5 and 6 field formats)
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := parser.Parse(job.Schedule); err != nil {
		// Try standard 5-field format
		if _, err2 := cron.ParseStandard(job.Schedule); err2 != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
	}

	return nil
}

// loadJobsFromDatabase loads jobs from the database
func (s *Scheduler) loadJobsFromDatabase() error {
	// In a real implementation, this would load from database
	// For now, just ensure we have a clean state
	return nil
}

// saveJobToDatabase saves a job to the database
func (s *Scheduler) saveJobToDatabase(job *ScheduledJob) error {
	// In a real implementation, this would save to database
	return nil
}

// updateJobInDatabase updates a job in the database
func (s *Scheduler) updateJobInDatabase(job *ScheduledJob) error {
	// In a real implementation, this would update in database
	return nil
}

// deleteJobFromDatabase deletes a job from the database
func (s *Scheduler) deleteJobFromDatabase(jobID string) error {
	// In a real implementation, this would delete from database
	return nil
}

// getNextRunTime calculates the next run time for a cron schedule
func (s *Scheduler) getNextRunTime(schedule string) *time.Time {
	// Try 6-field format first (with seconds)
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	scheduleEntry, err := parser.Parse(schedule)
	if err != nil {
		// Fall back to 5-field format
		scheduleEntry, err = cron.ParseStandard(schedule)
		if err != nil {
			return nil
		}
	}

	next := scheduleEntry.Next(time.Now())
	return &next
}

// generateExecutionID generates a unique execution ID
func generateExecutionID() string {
	return fmt.Sprintf("exec_%d", time.Now().UnixNano())
}
