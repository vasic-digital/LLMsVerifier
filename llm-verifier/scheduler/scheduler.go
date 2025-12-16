package scheduler

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"llm-verifier/database"
)

// ScheduleType represents the type of scheduling
type ScheduleType string

const (
	ScheduleTypeCron     ScheduleType = "cron"
	ScheduleTypeInterval ScheduleType = "interval"
	ScheduleTypeOnce     ScheduleType = "once"
)

// JobType represents the type of job to execute
type JobType string

const (
	JobTypeVerification JobType = "verification"
	JobTypeExport       JobType = "export"
	JobTypeCleanup      JobType = "cleanup"
	JobTypeReport       JobType = "report"
)

// Schedule represents a scheduled job
type Schedule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        ScheduleType           `json:"type"`
	JobType     JobType                `json:"job_type"`
	Expression  string                 `json:"expression"` // Cron expression or interval
	Enabled     bool                   `json:"enabled"`
	Targets     []string               `json:"targets"` // Model IDs, provider IDs, or "all"
	Options     map[string]interface{} `json:"options"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	NextRun     time.Time              `json:"next_run"`
	LastRun     *time.Time             `json:"last_run,omitempty"`
	RunCount    int                    `json:"run_count"`
	ErrorCount  int                    `json:"error_count"`
}

// ScheduleRun represents a single execution of a scheduled job
type ScheduleRun struct {
	ID          string                 `json:"id"`
	ScheduleID  string                 `json:"schedule_id"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Status      string                 `json:"status"` // "running", "completed", "failed"
	Error       string                 `json:"error,omitempty"`
	Results     map[string]interface{} `json:"results,omitempty"`
}

// Scheduler manages scheduled jobs
type Scheduler struct {
	db         *database.Database
	schedules  map[string]*Schedule
	runs       map[string]*ScheduleRun
	ticker     *time.Ticker
	stopCh     chan struct{}
	running    bool
	mu         sync.RWMutex
	jobHandler func(jobType JobType, targets []string, options map[string]interface{}) error
}

// NewScheduler creates a new scheduler instance
func NewScheduler(db *database.Database) *Scheduler {
	return &Scheduler{
		db:        db,
		schedules: make(map[string]*Schedule),
		runs:      make(map[string]*ScheduleRun),
		stopCh:    make(chan struct{}),
		running:   false,
	}
}

// SetJobHandler sets the function that handles job execution
func (s *Scheduler) SetJobHandler(handler func(jobType JobType, targets []string, options map[string]interface{}) error) {
	s.jobHandler = handler
}

// Start begins the scheduler
func (s *Scheduler) Start() error {
	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	s.running = true
	s.ticker = time.NewTicker(30 * time.Second) // Check every 30 seconds

	// Load existing schedules
	if err := s.loadSchedules(); err != nil {
		return fmt.Errorf("failed to load schedules: %w", err)
	}

	// Start the scheduler loop
	go s.run()

	log.Printf("Scheduler started with %d schedules", len(s.schedules))
	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	if !s.running {
		return
	}

	s.running = false
	close(s.stopCh)
	s.ticker.Stop()

	log.Println("Scheduler stopped")
}

// CreateSchedule creates a new schedule
func (s *Scheduler) CreateSchedule(schedule *Schedule) error {
	schedule.ID = generateScheduleID()
	schedule.CreatedAt = time.Now()
	schedule.UpdatedAt = time.Now()
	schedule.NextRun = s.calculateNextRun(schedule)

	s.mu.Lock()
	s.schedules[schedule.ID] = schedule
	s.mu.Unlock()

	// Persist to database (placeholder)
	log.Printf("Created schedule: %s (%s)", schedule.Name, schedule.ID)

	return nil
}

// UpdateSchedule updates an existing schedule
func (s *Scheduler) UpdateSchedule(id string, updates *Schedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	schedule, exists := s.schedules[id]
	if !exists {
		return fmt.Errorf("schedule not found: %s", id)
	}

	// Update fields
	if updates.Name != "" {
		schedule.Name = updates.Name
	}
	if updates.Description != "" {
		schedule.Description = updates.Description
	}
	if updates.Expression != "" {
		schedule.Expression = updates.Expression
	}
	schedule.Enabled = updates.Enabled
	if updates.Targets != nil {
		schedule.Targets = updates.Targets
	}
	if updates.Options != nil {
		schedule.Options = updates.Options
	}
	schedule.UpdatedAt = time.Now()
	schedule.NextRun = s.calculateNextRun(schedule)

	log.Printf("Updated schedule: %s", id)
	return nil
}

// DeleteSchedule deletes a schedule
func (s *Scheduler) DeleteSchedule(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.schedules[id]; !exists {
		return fmt.Errorf("schedule not found: %s", id)
	}

	delete(s.schedules, id)

	// Clean up associated runs
	for runID, run := range s.runs {
		if run.ScheduleID == id {
			delete(s.runs, runID)
		}
	}

	log.Printf("Deleted schedule: %s", id)
	return nil
}

// GetSchedules returns all schedules
func (s *Scheduler) GetSchedules() []*Schedule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var schedules []*Schedule
	for _, schedule := range s.schedules {
		schedules = append(schedules, schedule)
	}

	// Sort by next run time
	sort.Slice(schedules, func(i, j int) bool {
		return schedules[i].NextRun.Before(schedules[j].NextRun)
	})

	return schedules
}

// GetSchedule returns a specific schedule
func (s *Scheduler) GetSchedule(id string) (*Schedule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	schedule, exists := s.schedules[id]
	if !exists {
		return nil, fmt.Errorf("schedule not found: %s", id)
	}

	return schedule, nil
}

// GetScheduleRuns returns runs for a specific schedule
func (s *Scheduler) GetScheduleRuns(scheduleID string, limit int) []*ScheduleRun {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var runs []*ScheduleRun
	for _, run := range s.runs {
		if run.ScheduleID == scheduleID {
			runs = append(runs, run)
		}
	}

	// Sort by start time (newest first)
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].StartedAt.After(runs[j].StartedAt)
	})

	if limit > 0 && len(runs) > limit {
		runs = runs[:limit]
	}

	return runs
}

// EnableSchedule enables a schedule
func (s *Scheduler) EnableSchedule(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	schedule, exists := s.schedules[id]
	if !exists {
		return fmt.Errorf("schedule not found: %s", id)
	}

	schedule.Enabled = true
	schedule.UpdatedAt = time.Now()
	schedule.NextRun = s.calculateNextRun(schedule)

	log.Printf("Enabled schedule: %s", id)
	return nil
}

// DisableSchedule disables a schedule
func (s *Scheduler) DisableSchedule(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	schedule, exists := s.schedules[id]
	if !exists {
		return fmt.Errorf("schedule not found: %s", id)
	}

	schedule.Enabled = false
	schedule.UpdatedAt = time.Now()

	log.Printf("Disabled schedule: %s", id)
	return nil
}

// RunNow executes a schedule immediately
func (s *Scheduler) RunNow(id string) error {
	s.mu.RLock()
	schedule, exists := s.schedules[id]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("schedule not found: %s", id)
	}

	return s.executeSchedule(schedule)
}

// Private methods

func (s *Scheduler) run() {
	log.Println("Scheduler main loop started")

	for {
		select {
		case <-s.stopCh:
			return
		case <-s.ticker.C:
			s.checkSchedules()
		}
	}
}

func (s *Scheduler) checkSchedules() {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, schedule := range s.schedules {
		if !schedule.Enabled {
			continue
		}

		if now.After(schedule.NextRun) || now.Equal(schedule.NextRun) {
			go func(sched *Schedule) {
				if err := s.executeSchedule(sched); err != nil {
					log.Printf("Failed to execute schedule %s: %v", sched.ID, err)
				}
			}(schedule)

			// Update next run time
			schedule.NextRun = s.calculateNextRun(schedule)
			schedule.LastRun = &now
			schedule.RunCount++
		}
	}
}

func (s *Scheduler) executeSchedule(schedule *Schedule) error {
	runID := generateRunID()

	run := &ScheduleRun{
		ID:         runID,
		ScheduleID: schedule.ID,
		StartedAt:  time.Now(),
		Status:     "running",
	}

	s.mu.Lock()
	s.runs[runID] = run
	s.mu.Unlock()

	log.Printf("Executing schedule: %s (%s)", schedule.Name, schedule.ID)

	// Execute the job
	var err error
	if s.jobHandler != nil {
		err = s.jobHandler(schedule.JobType, schedule.Targets, schedule.Options)
	} else {
		err = fmt.Errorf("no job handler configured")
	}

	// Update run status
	now := time.Now()
	run.CompletedAt = &now

	if err != nil {
		run.Status = "failed"
		run.Error = err.Error()

		s.mu.Lock()
		schedule.ErrorCount++
		s.mu.Unlock()

		log.Printf("Schedule %s failed: %v", schedule.ID, err)
	} else {
		run.Status = "completed"
		run.Results = map[string]interface{}{
			"message": "Schedule executed successfully",
		}

		log.Printf("Schedule %s completed successfully", schedule.ID)
	}

	return err
}

func (s *Scheduler) calculateNextRun(schedule *Schedule) time.Time {
	now := time.Now()

	switch schedule.Type {
	case ScheduleTypeCron:
		return s.parseCronExpression(schedule.Expression, now)
	case ScheduleTypeInterval:
		return s.parseIntervalExpression(schedule.Expression, now)
	case ScheduleTypeOnce:
		// For one-time schedules, set next run to far future
		return now.Add(365 * 24 * time.Hour)
	default:
		// Default to hourly
		return now.Add(time.Hour)
	}
}

func (s *Scheduler) parseCronExpression(expression string, now time.Time) time.Time {
	// Simple cron parser (supports basic expressions)
	parts := strings.Fields(expression)
	if len(parts) != 5 {
		log.Printf("Invalid cron expression: %s", expression)
		return now.Add(time.Hour)
	}

	minute := s.parseCronField(parts[0], 0, 59)
	hour := s.parseCronField(parts[1], 0, 23)
	day := s.parseCronField(parts[2], 1, 31)
	month := s.parseCronField(parts[3], 1, 12)
	weekday := s.parseCronField(parts[4], 0, 6)

	// Find next matching time
	next := now.Add(time.Minute)
	for i := 0; i < 10000; i++ { // Prevent infinite loop
		if s.matchesCronField(next.Minute(), minute) &&
			s.matchesCronField(next.Hour(), hour) &&
			s.matchesCronField(next.Day(), day) &&
			s.matchesCronField(int(next.Month()), month) &&
			s.matchesCronField(int(next.Weekday()), weekday) {
			return next
		}
		next = next.Add(time.Minute)
	}

	return now.Add(time.Hour)
}

func (s *Scheduler) parseIntervalExpression(expression string, now time.Time) time.Time {
	// Parse expressions like "1h", "30m", "2h30m", etc.
	duration, err := time.ParseDuration(expression)
	if err != nil {
		log.Printf("Invalid interval expression: %s", expression)
		return now.Add(time.Hour)
	}

	return now.Add(duration)
}

func (s *Scheduler) parseCronField(field string, min, max int) []int {
	if field == "*" {
		var values []int
		for i := min; i <= max; i++ {
			values = append(values, i)
		}
		return values
	}

	// Handle comma-separated values and ranges
	var values []int
	parts := strings.Split(field, ",")

	for _, part := range parts {
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) == 2 {
				start, err1 := strconv.Atoi(rangeParts[0])
				end, err2 := strconv.Atoi(rangeParts[1])
				if err1 == nil && err2 == nil {
					for i := start; i <= end; i++ {
						values = append(values, i)
					}
				}
			}
		} else {
			if val, err := strconv.Atoi(part); err == nil {
				values = append(values, val)
			}
		}
	}

	return values
}

func (s *Scheduler) matchesCronField(value int, allowed []int) bool {
	for _, a := range allowed {
		if value == a {
			return true
		}
	}
	return false
}

func (s *Scheduler) loadSchedules() error {
	// Load schedules from database (placeholder)
	// In a real implementation, this would query the database

	log.Println("Loaded schedules from database")
	return nil
}

// Helper functions

func generateScheduleID() string {
	return fmt.Sprintf("sched_%d", time.Now().UnixNano())
}

func generateRunID() string {
	return fmt.Sprintf("run_%d", time.Now().UnixNano())
}

// Predefined schedule templates

// CreateDailyVerificationSchedule creates a daily verification schedule
func CreateDailyVerificationSchedule(name string, targets []string) *Schedule {
	return &Schedule{
		Name:        name,
		Description: "Daily verification of all models",
		Type:        ScheduleTypeCron,
		JobType:     JobTypeVerification,
		Expression:  "0 2 * * *", // Daily at 2 AM
		Enabled:     true,
		Targets:     targets,
		Options: map[string]interface{}{
			"full_verification": true,
		},
	}
}

// CreateHourlyHealthCheckSchedule creates an hourly health check schedule
func CreateHourlyHealthCheckSchedule(name string) *Schedule {
	return &Schedule{
		Name:        name,
		Description: "Hourly system health check",
		Type:        ScheduleTypeCron,
		JobType:     JobTypeCleanup,
		Expression:  "0 * * * *", // Every hour
		Enabled:     true,
		Targets:     []string{"system"},
		Options: map[string]interface{}{
			"check_databases":   true,
			"check_connections": true,
		},
	}
}

// CreateWeeklyReportSchedule creates a weekly report generation schedule
func CreateWeeklyReportSchedule(name string) *Schedule {
	return &Schedule{
		Name:        name,
		Description: "Weekly comprehensive report",
		Type:        ScheduleTypeCron,
		JobType:     JobTypeReport,
		Expression:  "0 3 * * 1", // Every Monday at 3 AM
		Enabled:     true,
		Targets:     []string{"all"},
		Options: map[string]interface{}{
			"report_type":    "comprehensive",
			"include_charts": true,
		},
	}
}
