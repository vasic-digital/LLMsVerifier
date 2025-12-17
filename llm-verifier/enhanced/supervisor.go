package enhanced

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	llmverifier "llm-verifier/llmverifier"
)

// Task represents a task to be executed
type Task struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Priority    int                    `json:"priority"`
	Data        map[string]interface{} `json:"data"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Status      string                 `json:"status"`
	Result      *TaskResult            `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	MaxRetries  int                    `json:"max_retries"`
	RetryCount  int                    `json:"retry_count"`
	AssignedTo  string                 `json:"assigned_to,omitempty"`
}

// TaskResult represents the result of a completed task
type TaskResult struct {
	TaskID      string                 `json:"task_id"`
	WorkerID    string                 `json:"worker_id"`
	Success     bool                   `json:"success"`
	Data        map[string]interface{} `json:"data"`
	Result      interface{}            `json:"result"`
	Error       interface{}            `json:"error,omitempty"`
	Duration    time.Duration          `json:"duration"`
	CompletedAt time.Time              `json:"completed_at"`
}

// Worker represents a worker that can execute tasks
type Worker struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Status        string                 `json:"status"`
	CurrentTask   *Task                  `json:"current_task,omitempty"`
	TasksDone     int                    `json:"tasks_done"`
	LastActive    time.Time              `json:"last_active"`
	StartTime     time.Time              `json:"start_time"`
	Capabilities  []string               `json:"capabilities"`
	LastHeartbeat time.Time              `json:"last_heartbeat"`
	Performance   map[string]interface{} `json:"performance"`
}

// TaskHandler represents a function that can handle a specific type of task
type TaskHandler func(ctx context.Context, task *Task) (interface{}, error)

// Supervisor manages task decomposition and worker coordination
type Supervisor struct {
	workers      map[string]*Worker
	tasks        map[string]*Task
	taskQueue    chan *Task
	resultChan   chan *TaskResult
	stopCh       chan struct{}
	verifier     *llmverifier.Verifier
	taskHandlers map[string]TaskHandler
	maxWorkers   int
	mu           sync.RWMutex
}

// TaskResult represents the result of a completed task

// NewSupervisor creates a new supervisor instance
func NewSupervisor(verifier *llmverifier.Verifier, maxWorkers int) *Supervisor {
	s := &Supervisor{
		workers:      make(map[string]*Worker),
		tasks:        make(map[string]*Task),
		taskQueue:    make(chan *Task, 100), // Buffer for 100 tasks
		resultChan:   make(chan *TaskResult, 100),
		stopCh:       make(chan struct{}),
		verifier:     verifier,
		taskHandlers: make(map[string]TaskHandler),
		maxWorkers:   maxWorkers,
	}

	// Register default task handlers
	s.registerDefaultHandlers()

	return s
}

// Start begins the supervisor operations
func (s *Supervisor) Start() error {
	log.Printf("Starting supervisor with max %d workers", s.maxWorkers)

	// Start worker manager
	go s.workerManager()

	// Start task dispatcher
	go s.taskDispatcher()

	// Start result processor
	go s.resultProcessor()

	return nil
}

// Stop gracefully shuts down the supervisor
func (s *Supervisor) Stop() {
	log.Println("Stopping ..")

	close(s.stopCh)

	// Wait a bit for workers to finish
	time.Sleep(2 * time.Second)

	// Clean up resources
	close(s.taskQueue)
	close(s.resultChan)

	log.Println("Supervisor stopped")
}

// DecomposeTask breaks down a complex task into smaller subtasks
func (s *Supervisor) DecomposeTask(taskDescription string, context map[string]interface{}) ([]*Task, error) {
	// Simple rule-based decomposition (can be enhanced with LLM later)
	var tasks []*Task

	// Create a primary analysis task
	task := &Task{
		ID:         fmt.Sprintf("task_%d", time.Now().UnixNano()),
		Type:       "analysis",
		Priority:   5,
		Data:       map[string]any{"description": taskDescription, "context": context},
		CreatedAt:  time.Now(),
		MaxRetries: 3,
		Status:     "pending",
	}

	tasks = append(tasks, task)

	// Create subtasks based on keywords
	description := strings.ToLower(taskDescription)

	if strings.Contains(description, "code") || strings.Contains(description, "review") {
		subtask := &Task{
			ID:         fmt.Sprintf("task_%d_sub1", time.Now().UnixNano()),
			Type:       "generation",
			Priority:   4,
			Data:       map[string]any{"description": "Code analysis subtask", "context": context},
			CreatedAt:  time.Now(),
			MaxRetries: 3,
			Status:     "pending",
		}
		tasks = append(tasks, subtask)
	}

	if strings.Contains(description, "test") || strings.Contains(description, "validate") {
		subtask := &Task{
			ID:         fmt.Sprintf("task_%d_sub2", time.Now().UnixNano()),
			Type:       "testing",
			Priority:   3,
			Data:       map[string]any{"description": "Testing subtask", "context": context},
			CreatedAt:  time.Now(),
			MaxRetries: 3,
			Status:     "pending",
		}
		tasks = append(tasks, subtask)
	}

	log.Printf("Decomposed task into %d subtasks", len(tasks))
	return tasks, nil
}

// SubmitTask submits a task for execution
func (s *Supervisor) SubmitTask(task *Task) error {
	s.mu.Lock()
	s.tasks[task.ID] = task
	s.mu.Unlock()

	select {
	case s.taskQueue <- task:
		log.Printf("Submitted task %s of type %s", task.ID, task.Type)
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("task queue is full")
	}
}

// SubmitTasks submits multiple tasks for execution
func (s *Supervisor) SubmitTasks(tasks []*Task) error {
	for _, task := range tasks {
		if err := s.SubmitTask(task); err != nil {
			return err
		}
	}
	return nil
}

// GetTaskStatus returns the status of a task
func (s *Supervisor) GetTaskStatus(taskID string) (*Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	return task, nil
}

// GetWorkerStatus returns the status of all workers
func (s *Supervisor) GetWorkerStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	workers := make(map[string]interface{})
	for id, worker := range s.workers {
		workers[id] = map[string]interface{}{
			"status":         worker.Status,
			"capabilities":   worker.Capabilities,
			"current_task":   worker.CurrentTask,
			"last_heartbeat": worker.LastHeartbeat,
			"performance":    worker.Performance,
		}
	}

	return map[string]interface{}{
		"workers":     workers,
		"total":       len(workers),
		"max_workers": s.maxWorkers,
	}
}

// GetSystemStatus returns overall system status
func (s *Supervisor) GetSystemStatus() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

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

	activeWorkers := 0
	for _, worker := range s.workers {
		if worker.Status == "busy" {
			activeWorkers++
		}
	}

	return map[string]interface{}{
		"tasks": map[string]int{
			"pending":   pendingTasks,
			"running":   runningTasks,
			"completed": completedTasks,
			"failed":    failedTasks,
			"total":     len(s.tasks),
		},
		"workers": map[string]interface{}{
			"active": activeWorkers,
			"total":  len(s.workers),
			"max":    s.maxWorkers,
		},
		"queue_size": len(s.taskQueue),
	}
}

// registerDefaultHandlers registers built-in task handlers
func (s *Supervisor) registerDefaultHandlers() {
	// Analysis task handler
	s.taskHandlers["analysis"] = func(ctx context.Context, task *Task) (interface{}, error) {
		description := task.Data["description"].(string)
		log.Printf("Executing analysis task: %s", description)

		// Simulate analysis work
		time.Sleep(2 * time.Second)

		return map[string]any{
			"analysis_result": "Analysis completed",
			"description":     description,
		}, nil
	}

	// Generation task handler
	s.taskHandlers["generation"] = func(ctx context.Context, task *Task) (interface{}, error) {
		description := task.Data["description"].(string)
		log.Printf("Executing generation task: %s", description)

		// Simulate generation work
		time.Sleep(3 * time.Second)

		return map[string]interface{}{
			"generated_content": "Generated content placeholder",
			"description":       description,
		}, nil
	}

	// Testing task handler
	s.taskHandlers["testing"] = func(ctx context.Context, task *Task) (interface{}, error) {
		description := task.Data["description"].(string)
		log.Printf("Executing testing task: %s", description)

		// Simulate testing work
		time.Sleep(1 * time.Second)

		return map[string]interface{}{
			"test_results": "All tests passed",
			"description":  description,
		}, nil
	}

	// General task handler (fallback)
	s.taskHandlers["general"] = func(ctx context.Context, task *Task) (interface{}, error) {
		description := task.Data["description"].(string)
		log.Printf("Executing general task: %s", description)

		// Simulate general work
		time.Sleep(2 * time.Second)

		return map[string]interface{}{
			"result":      "Task completed",
			"description": description,
		}, nil
	}
}

// workerManager manages the worker pool
func (s *Supervisor) workerManager() {
	log.Println("Worker manager started")

	// Create initial workers
	for i := 0; i < s.maxWorkers; i++ {
		workerID := fmt.Sprintf("worker_%d", i+1)
		worker := &Worker{
			ID:            workerID,
			Capabilities:  []string{"analysis", "generation", "testing", "general"},
			Status:        "idle",
			LastHeartbeat: time.Now(),
		}
		s.workers[workerID] = worker

		// Start worker goroutine
		go s.workerLoop(worker)
	}

	log.Printf("Created %d workers", s.maxWorkers)
}

// workerLoop runs the worker's main loop
func (s *Supervisor) workerLoop(worker *Worker) {
	log.Printf("Worker %s started", worker.ID)

	for {
		select {
		case <-s.stopCh:
			log.Printf("Worker %s stopping", worker.ID)
			return

		default:
			// Check if worker should process a task
			if worker.Status == "idle" {
				// Try to get a task
				select {
				case task := <-s.taskQueue:
					s.executeTask(worker, task)
				case <-time.After(1 * time.Second):
					// No task available, continue
				}
			}

			// Update heartbeat
			worker.LastHeartbeat = time.Now()
			time.Sleep(5 * time.Second)
		}
	}
}

// executeTask executes a task on a worker
func (s *Supervisor) executeTask(worker *Worker, task *Task) {
	worker.Status = "busy"
	worker.CurrentTask = task
	task.Status = "running"
	task.AssignedTo = worker.ID
	now := time.Now()
	task.StartedAt = &now

	log.Printf("Worker %s executing task %s", worker.ID, task.ID)

	// Execute the task
	ctx := context.Background()
	handler, exists := s.taskHandlers[task.Type]
	if !exists {
		handler = s.taskHandlers["general"] // Fallback handler
	}

	result, err := handler(ctx, task)

	// Update task status
	task.CompletedAt = &now
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		task.RetryCount++

		if task.RetryCount < task.MaxRetries {
			// Re-queue for retry
			task.Status = "pending"
			select {
			case s.taskQueue <- task:
			default:
				log.Printf("Failed to re-queue task %s for retry", task.ID)
			}
		}
	} else {
		task.Status = "completed"
		task.Result = result
	}

	// Send result
	select {
	case s.resultChan <- &TaskResult{
		TaskID: task.ID,
		Result: result,
		Error:  fmt.Errorf(task.Error),
	}:
	default:
		log.Printf("Result channel full, dropping result for task %s", task.ID)
	}

	// Reset worker
	worker.Status = "idle"
	worker.CurrentTask = nil

	// Update performance metrics
	if err == nil {
		worker.Performance.TasksCompleted++
	} else {
		worker.Performance.TasksFailed++
	}

	totalTasks := worker.Performance.TasksCompleted + worker.Performance.TasksFailed
	if totalTasks > 0 {
		worker.Performance.SuccessRate = float64(worker.Performance.TasksCompleted) / float64(totalTasks)
	}

	log.Printf("Worker %s completed task %s (success: %v)", worker.ID, task.ID, err == nil)
}

// taskDispatcher dispatches tasks to available workers
func (s *Supervisor) taskDispatcher() {
	log.Println("Task dispatcher started")

	for {
		select {
		case <-s.stopCh:
			return
		case task := <-s.taskQueue:
			// Find best worker for this task
			worker := s.findBestWorker(task)
			if worker != nil {
				go s.executeTask(worker, task)
			} else {
				// No worker available, re-queue
				log.Printf("No worker available for task %s, re-queuing", task.ID)
				select {
				case s.taskQueue <- task:
				case <-time.After(1 * time.Second):
					log.Printf("Failed to re-queue task %s", task.ID)
				}
			}
		}
	}
}

// findBestWorker finds the best worker for a task
func (s *Supervisor) findBestWorker(task *Task) *Worker {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var candidates []*Worker

	// Find workers that can handle this task type and are idle
	for _, worker := range s.workers {
		if worker.Status == "idle" {
			// Check if worker has the required capability
			for _, cap := range worker.Capabilities {
				if cap == task.Type || cap == "general" {
					candidates = append(candidates, worker)
					break
				}
			}
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// Sort by performance (success rate, then task completion count)
	sort.Slice(candidates, func(i, j int) bool {
		wi, wj := candidates[i], candidates[j]

		// Higher success rate first
		if wi.Performance.SuccessRate != wj.Performance.SuccessRate {
			return wi.Performance.SuccessRate > wj.Performance.SuccessRate
		}

		// Higher completion count first
		return wi.Performance.TasksCompleted > wj.Performance.TasksCompleted
	})

	return candidates[0]
}

// resultProcessor processes completed task results
func (s *Supervisor) resultProcessor() {
	log.Println("Result processor started")

	for {
		select {
		case <-s.stopCh:
			return
		case result := <-s.resultChan:
			success := result.Error == nil
			log.Printf("Processed result for task %s (success: %v)",
				result.TaskID, success)
			// Additional result processing can be added here
		}
	}
}
