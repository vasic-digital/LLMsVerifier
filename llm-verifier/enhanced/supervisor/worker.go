package supervisor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"llm-verifier/llmverifier"
)

// Task represents a unit of work to be executed
type Task struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Priority    int                    `json:"priority"` // Higher number = higher priority
	Data        map[string]interface{} `json:"data"`
	CreatedAt   time.Time              `json:"created_at"`
	Deadline    *time.Time             `json:"deadline,omitempty"`
	MaxRetries  int                    `json:"max_retries"`
	RetryCount  int                    `json:"retry_count"`
	Status      string                 `json:"status"` // pending, running, completed, failed
	Result      interface{}            `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	AssignedTo  string                 `json:"assigned_to,omitempty"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// Worker represents a worker agent
type Worker struct {
	ID            string
	Capabilities  []string
	Status        string // idle, busy, offline
	CurrentTask   *Task
	LastHeartbeat time.Time
	Performance   WorkerPerformance
}

// WorkerPerformance tracks worker performance metrics
type WorkerPerformance struct {
	TasksCompleted int
	TasksFailed    int
	AvgTaskTime    time.Duration
	SuccessRate    float64
}

// TaskHandler is a function that handles task execution
type TaskHandler func(ctx context.Context, task *Task) (interface{}, error)

// Supervisor manages the worker pool and task distribution
type Supervisor struct {
	workers   map[string]*Worker
	tasks     []*Task
	taskQueue chan *Task
	results   chan *TaskResult
	handlers  map[string]TaskHandler
	verifier  *llmverifier.Verifier
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup

	// Configuration
	maxWorkers        int
	taskTimeout       time.Duration
	heartbeatInterval time.Duration
}

// TaskResult represents the result of a task execution
type TaskResult struct {
	TaskID string
	Result interface{}
	Error  error
}

// NewSupervisor creates a new supervisor
func NewSupervisor(verifier *llmverifier.Verifier, maxWorkers int) *Supervisor {
	ctx, cancel := context.WithCancel(context.Background())

	return &Supervisor{
		workers:           make(map[string]*Worker),
		tasks:             make([]*Task, 0),
		taskQueue:         make(chan *Task, 100),
		results:           make(chan *TaskResult, 100),
		handlers:          make(map[string]TaskHandler),
		verifier:          verifier,
		ctx:               ctx,
		cancel:            cancel,
		maxWorkers:        maxWorkers,
		taskTimeout:       5 * time.Minute,
		heartbeatInterval: 30 * time.Second,
	}
}

// Start begins the supervisor operation
func (s *Supervisor) Start() error {
	log.Println("Starting supervisor...")

	// Start task dispatcher
	s.wg.Add(1)
	go s.taskDispatcher()

	// Start result processor
	s.wg.Add(1)
	go s.resultProcessor()

	// Start health checker
	s.wg.Add(1)
	go s.healthChecker()

	log.Printf("Supervisor started with capacity for %d workers", s.maxWorkers)
	return nil
}

// Stop gracefully stops the supervisor
func (s *Supervisor) Stop() {
	log.Println("Stopping supervisor...")
	s.cancel()
	close(s.taskQueue)
	close(s.results)
	s.wg.Wait()
	log.Println("Supervisor stopped")
}

// RegisterHandler registers a task handler for a specific task type
func (s *Supervisor) RegisterHandler(taskType string, handler TaskHandler) {
	s.handlers[taskType] = handler
	log.Printf("Registered handler for task type: %s", taskType)
}

// AddWorker adds a worker to the pool
func (s *Supervisor) AddWorker(workerID string, capabilities []string) {
	worker := &Worker{
		ID:            workerID,
		Capabilities:  capabilities,
		Status:        "idle",
		LastHeartbeat: time.Now(),
		Performance: WorkerPerformance{
			TasksCompleted: 0,
			TasksFailed:    0,
			AvgTaskTime:    0,
			SuccessRate:    1.0,
		},
	}

	s.workers[workerID] = worker
	log.Printf("Added worker %s with capabilities: %v", workerID, capabilities)
}

// RemoveWorker removes a worker from the pool
func (s *Supervisor) RemoveWorker(workerID string) {
	delete(s.workers, workerID)
	log.Printf("Removed worker: %s", workerID)
}

// SubmitTask submits a task for execution
func (s *Supervisor) SubmitTask(task *Task) error {
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	if task.Status == "" {
		task.Status = "pending"
	}

	s.tasks = append(s.tasks, task)

	// Try to dispatch immediately
	select {
	case s.taskQueue <- task:
		log.Printf("Dispatched task %s immediately", task.ID)
	default:
		log.Printf("Queued task %s for later dispatch", task.ID)
	}

	return nil
}

// BreakDownTask breaks down a complex task into smaller subtasks
func (s *Supervisor) BreakDownTask(complexTask *Task) ([]*Task, error) {
	// TODO: Use LLM to break down the task into subtasks
	// For now, create mock subtasks

	// This would normally call the LLM, but for now we'll create mock subtasks
	subtasks := []*Task{
		{
			ID:         fmt.Sprintf("%s_sub_1", complexTask.ID),
			Type:       "analysis",
			Priority:   8,
			Data:       map[string]interface{}{"parent_task": complexTask.ID, "step": "analyze_requirements"},
			CreatedAt:  time.Now(),
			MaxRetries: 3,
			Status:     "pending",
		},
		{
			ID:         fmt.Sprintf("%s_sub_2", complexTask.ID),
			Type:       "implementation",
			Priority:   7,
			Data:       map[string]interface{}{"parent_task": complexTask.ID, "step": "implement_solution"},
			CreatedAt:  time.Now(),
			MaxRetries: 3,
			Status:     "pending",
		},
		{
			ID:         fmt.Sprintf("%s_sub_3", complexTask.ID),
			Type:       "testing",
			Priority:   6,
			Data:       map[string]interface{}{"parent_task": complexTask.ID, "step": "test_implementation"},
			CreatedAt:  time.Now(),
			MaxRetries: 2,
			Status:     "pending",
		},
	}

	log.Printf("Broke down task %s into %d subtasks", complexTask.ID, len(subtasks))
	return subtasks, nil
}

// taskDispatcher dispatches tasks to available workers
func (s *Supervisor) taskDispatcher() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case task := <-s.taskQueue:
			if task == nil {
				return
			}

			// Find an available worker
			workerID := s.findAvailableWorker(task)
			if workerID == "" {
				// No worker available, put back in queue
				time.Sleep(1 * time.Second)
				select {
				case s.taskQueue <- task:
				default:
					log.Printf("Task queue full, dropping task %s", task.ID)
				}
				continue
			}

			// Assign task to worker
			s.assignTaskToWorker(task, workerID)

			// Start task execution in goroutine
			go s.executeTask(task, workerID)
		}
	}
}

// findAvailableWorker finds the best available worker for a task
func (s *Supervisor) findAvailableWorker(task *Task) string {
	var bestWorker string
	var bestScore float64

	for workerID, worker := range s.workers {
		if worker.Status != "idle" {
			continue
		}

		// Check if worker has required capabilities
		if !s.workerHasCapability(worker, task.Type) {
			continue
		}

		// Calculate worker score based on performance
		score := s.calculateWorkerScore(worker)

		if score > bestScore {
			bestScore = score
			bestWorker = workerID
		}
	}

	return bestWorker
}

// workerHasCapability checks if a worker has the required capability
func (s *Supervisor) workerHasCapability(worker *Worker, capability string) bool {
	for _, cap := range worker.Capabilities {
		if cap == capability || cap == "general" {
			return true
		}
	}
	return false
}

// calculateWorkerScore calculates a score for worker selection
func (s *Supervisor) calculateWorkerScore(worker *Worker) float64 {
	if worker.Performance.TasksCompleted == 0 {
		return 0.5 // Default score for new workers
	}

	// Score based on success rate and average task time
	successRateScore := worker.Performance.SuccessRate
	timeScore := 1.0 / (1.0 + worker.Performance.AvgTaskTime.Seconds()/60.0) // Prefer faster workers

	return (successRateScore * 0.7) + (timeScore * 0.3)
}

// assignTaskToWorker assigns a task to a worker
func (s *Supervisor) assignTaskToWorker(task *Task, workerID string) {
	worker := s.workers[workerID]
	worker.Status = "busy"
	worker.CurrentTask = task
	task.AssignedTo = workerID
	task.Status = "running"
	now := time.Now()
	task.StartedAt = &now

	log.Printf("Assigned task %s to worker %s", task.ID, workerID)
}

// executeTask executes a task on a worker
func (s *Supervisor) executeTask(task *Task, workerID string) {
	ctx, cancel := context.WithTimeout(s.ctx, s.taskTimeout)
	defer cancel()

	// Get handler for task type
	handler, exists := s.handlers[task.Type]
	if !exists {
		s.reportTaskResult(&TaskResult{
			TaskID: task.ID,
			Error:  fmt.Errorf("no handler registered for task type: %s", task.Type),
		})
		return
	}

	// Execute task
	startTime := time.Now()
	result, err := handler(ctx, task)
	executionTime := time.Now().Sub(startTime)

	// Report result
	s.reportTaskResult(&TaskResult{
		TaskID: task.ID,
		Result: result,
		Error:  err,
	})

	// Update worker performance
	s.updateWorkerPerformance(workerID, err == nil, executionTime)

	log.Printf("Task %s completed in %v", task.ID, executionTime)
}

// reportTaskResult reports a task execution result
func (s *Supervisor) reportTaskResult(result *TaskResult) {
	select {
	case s.results <- result:
	default:
		log.Printf("Results channel full, dropping result for task %s", result.TaskID)
	}
}

// resultProcessor processes task execution results
func (s *Supervisor) resultProcessor() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case result := <-s.results:
			if result == nil {
				return
			}

			s.processTaskResult(result)
		}
	}
}

// processTaskResult processes a completed task result
func (s *Supervisor) processTaskResult(result *TaskResult) {
	// Find the task
	var task *Task
	for _, t := range s.tasks {
		if t.ID == result.TaskID {
			task = t
			break
		}
	}

	if task == nil {
		log.Printf("Task not found: %s", result.TaskID)
		return
	}

	// Update task status
	now := time.Now()
	task.CompletedAt = &now

	if result.Error != nil {
		task.Status = "failed"
		task.Error = result.Error.Error()
		task.RetryCount++

		// Check if we should retry
		if task.RetryCount < task.MaxRetries {
			task.Status = "pending"
			task.AssignedTo = ""
			log.Printf("Task %s failed, will retry (%d/%d)", task.ID, task.RetryCount, task.MaxRetries)
		} else {
			log.Printf("Task %s failed permanently after %d retries", task.ID, task.RetryCount)
		}
	} else {
		task.Status = "completed"
		task.Result = result.Result
		log.Printf("Task %s completed successfully", task.ID)
	}

	// Free up the worker
	if task.AssignedTo != "" {
		if worker, exists := s.workers[task.AssignedTo]; exists {
			worker.Status = "idle"
			worker.CurrentTask = nil
		}
	}
}

// updateWorkerPerformance updates a worker's performance metrics
func (s *Supervisor) updateWorkerPerformance(workerID string, success bool, executionTime time.Duration) {
	worker, exists := s.workers[workerID]
	if !exists {
		return
	}

	// Update task counts
	if success {
		worker.Performance.TasksCompleted++
	} else {
		worker.Performance.TasksFailed++
	}

	// Update success rate
	totalTasks := worker.Performance.TasksCompleted + worker.Performance.TasksFailed
	if totalTasks > 0 {
		worker.Performance.SuccessRate = float64(worker.Performance.TasksCompleted) / float64(totalTasks)
	}

	// Update average task time (exponential moving average)
	if worker.Performance.AvgTaskTime == 0 {
		worker.Performance.AvgTaskTime = executionTime
	} else {
		alpha := 0.1 // Smoothing factor
		worker.Performance.AvgTaskTime = time.Duration(
			float64(worker.Performance.AvgTaskTime)*(1-alpha) + float64(executionTime)*alpha,
		)
	}
}

// healthChecker monitors worker health
func (s *Supervisor) healthChecker() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkWorkerHealth()
		}
	}
}

// checkWorkerHealth checks the health of all workers
func (s *Supervisor) checkWorkerHealth() {
	now := time.Now()
	timeout := s.heartbeatInterval * 3 // 3 missed heartbeats = offline

	for workerID, worker := range s.workers {
		if now.Sub(worker.LastHeartbeat) > timeout {
			if worker.Status != "offline" {
				log.Printf("Worker %s marked as offline", workerID)
				worker.Status = "offline"

				// Reassign any tasks from offline worker
				if worker.CurrentTask != nil {
					task := worker.CurrentTask
					task.AssignedTo = ""
					task.Status = "pending"
					select {
					case s.taskQueue <- task:
					default:
						log.Printf("Failed to reassign task %s from offline worker", task.ID)
					}
				}
			}
		}
	}
}

// GetStats returns supervisor statistics
func (s *Supervisor) GetStats() map[string]interface{} {
	activeWorkers := 0
	idleWorkers := 0
	busyWorkers := 0
	offlineWorkers := 0

	for _, worker := range s.workers {
		switch worker.Status {
		case "idle":
			activeWorkers++
			idleWorkers++
		case "busy":
			activeWorkers++
			busyWorkers++
		case "offline":
			offlineWorkers++
		}
	}

	pendingTasks := 0
	runningTasks := 0
	completedTasks := 0
	failedTasks := 0

	for _, task := range s.tasks {
		switch task.Status {
		case "pending":
			pendingTasks++
		case "running":
			runningTasks++
		case "completed":
			completedTasks++
		case "failed":
			failedTasks++
		}
	}

	return map[string]interface{}{
		"workers": map[string]int{
			"total":   len(s.workers),
			"active":  activeWorkers,
			"idle":    idleWorkers,
			"busy":    busyWorkers,
			"offline": offlineWorkers,
		},
		"tasks": map[string]int{
			"total":     len(s.tasks),
			"pending":   pendingTasks,
			"running":   runningTasks,
			"completed": completedTasks,
			"failed":    failedTasks,
		},
		"handlers": len(s.handlers),
	}
}
