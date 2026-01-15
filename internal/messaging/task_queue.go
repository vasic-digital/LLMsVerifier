package messaging

import (
	"context"
	"time"
)

// Task represents a task to be processed by workers.
type Task struct {
	// ID is a unique identifier for the task.
	ID string `json:"id"`

	// Type identifies the task type for routing to appropriate handlers.
	Type string `json:"type"`

	// Payload contains the task data.
	Payload []byte `json:"payload"`

	// Priority indicates task priority (1-10, higher = more urgent).
	Priority int `json:"priority,omitempty"`

	// MaxRetries is the maximum number of processing attempts.
	MaxRetries int `json:"max_retries,omitempty"`

	// RetryCount tracks how many times processing has been attempted.
	RetryCount int `json:"retry_count,omitempty"`

	// RetryBackoff is the delay before retrying.
	RetryBackoff time.Duration `json:"retry_backoff,omitempty"`

	// Deadline is the absolute deadline for task completion.
	Deadline time.Time `json:"deadline,omitempty"`

	// Timeout is the maximum time allowed for processing.
	Timeout time.Duration `json:"timeout,omitempty"`

	// CreatedAt is when the task was created.
	CreatedAt time.Time `json:"created_at"`

	// ScheduledAt is when the task should be executed (for delayed tasks).
	ScheduledAt time.Time `json:"scheduled_at,omitempty"`

	// StartedAt is when processing started.
	StartedAt time.Time `json:"started_at,omitempty"`

	// CompletedAt is when processing completed.
	CompletedAt time.Time `json:"completed_at,omitempty"`

	// Status is the current task status.
	Status TaskStatus `json:"status"`

	// WorkerID is the ID of the worker processing this task.
	WorkerID string `json:"worker_id,omitempty"`

	// Metadata contains additional task metadata.
	Metadata map[string]string `json:"metadata,omitempty"`

	// Result contains the task result (set on completion).
	Result []byte `json:"result,omitempty"`

	// Error contains the error message (set on failure).
	Error string `json:"error,omitempty"`

	// TraceID for distributed tracing.
	TraceID string `json:"trace_id,omitempty"`

	// CorrelationID for correlating related tasks.
	CorrelationID string `json:"correlation_id,omitempty"`

	// DeliveryTag is the broker-specific delivery identifier.
	DeliveryTag uint64 `json:"-"`
}

// TaskStatus represents the status of a task.
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusQueued     TaskStatus = "queued"
	TaskStatusScheduled  TaskStatus = "scheduled"
	TaskStatusRunning    TaskStatus = "running"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusRetrying   TaskStatus = "retrying"
	TaskStatusCancelled  TaskStatus = "cancelled"
	TaskStatusDeadLetter TaskStatus = "dead_letter"
)

// TaskHandler is a function that processes a task.
type TaskHandler func(ctx context.Context, task *Task) error

// TaskResult represents the result of task processing.
type TaskResult struct {
	TaskID      string        `json:"task_id"`
	Status      TaskStatus    `json:"status"`
	Result      []byte        `json:"result,omitempty"`
	Error       string        `json:"error,omitempty"`
	Duration    time.Duration `json:"duration"`
	RetryCount  int           `json:"retry_count"`
	CompletedAt time.Time     `json:"completed_at"`
}

// QueueStats contains statistics for a queue.
type QueueStats struct {
	Name            string `json:"name"`
	Messages        int64  `json:"messages"`
	MessagesReady   int64  `json:"messages_ready"`
	MessagesUnacked int64  `json:"messages_unacked"`
	Consumers       int    `json:"consumers"`
}

// TaskQueueBroker extends MessageBroker with task queue specific operations.
type TaskQueueBroker interface {
	MessageBroker

	// DeclareQueue declares a queue with the given options.
	DeclareQueue(ctx context.Context, name string, opts ...QueueOption) error

	// DeleteQueue deletes a queue.
	DeleteQueue(ctx context.Context, name string) error

	// PurgeQueue removes all messages from a queue.
	PurgeQueue(ctx context.Context, name string) (int64, error)

	// EnqueueTask adds a task to the queue.
	EnqueueTask(ctx context.Context, queue string, task *Task) error

	// EnqueueTasks adds multiple tasks to the queue.
	EnqueueTasks(ctx context.Context, queue string, tasks []*Task) error

	// DequeueTask retrieves and locks a task from the queue.
	DequeueTask(ctx context.Context, queue string, workerID string) (*Task, error)

	// AckTask acknowledges successful task completion.
	AckTask(ctx context.Context, task *Task) error

	// NackTask indicates task processing failure.
	NackTask(ctx context.Context, task *Task, requeue bool) error

	// MoveToDeadLetter moves a task to the dead letter queue.
	MoveToDeadLetter(ctx context.Context, task *Task, reason string) error

	// GetQueueStats returns queue statistics.
	GetQueueStats(ctx context.Context, queue string) (*QueueStats, error)

	// ScheduleTask schedules a task for future execution.
	ScheduleTask(ctx context.Context, queue string, task *Task, executeAt time.Time) error

	// CancelTask attempts to cancel a pending task.
	CancelTask(ctx context.Context, queue string, taskID string) error

	// GetTask retrieves a task by ID.
	GetTask(ctx context.Context, queue string, taskID string) (*Task, error)
}

// QueueConfig contains configuration for a task queue.
type QueueConfig struct {
	// Name is the queue name.
	Name string `json:"name" yaml:"name"`

	// Durable means the queue survives broker restart.
	Durable bool `json:"durable" yaml:"durable"`

	// Exclusive means only this connection can access the queue.
	Exclusive bool `json:"exclusive" yaml:"exclusive"`

	// AutoDelete means the queue is deleted when last consumer unsubscribes.
	AutoDelete bool `json:"auto_delete" yaml:"auto_delete"`

	// MessageTTL is the default message time-to-live.
	MessageTTL time.Duration `json:"message_ttl" yaml:"message_ttl"`

	// MaxLength is the maximum number of messages in the queue.
	MaxLength int64 `json:"max_length" yaml:"max_length"`

	// MaxLengthBytes is the maximum total size of messages in bytes.
	MaxLengthBytes int64 `json:"max_length_bytes" yaml:"max_length_bytes"`

	// DeadLetterExchange is the exchange for dead-lettered messages.
	DeadLetterExchange string `json:"dead_letter_exchange" yaml:"dead_letter_exchange"`

	// DeadLetterRoutingKey is the routing key for dead-lettered messages.
	DeadLetterRoutingKey string `json:"dead_letter_routing_key" yaml:"dead_letter_routing_key"`

	// MaxPriority enables priority queuing (0-255).
	MaxPriority int `json:"max_priority" yaml:"max_priority"`

	// Arguments contains additional queue arguments.
	Arguments map[string]interface{} `json:"arguments,omitempty" yaml:"arguments,omitempty"`
}

// QueueOption is a function that modifies QueueConfig.
type QueueOption func(*QueueConfig)

// DefaultQueueConfig returns a QueueConfig with sensible defaults.
func DefaultQueueConfig(name string) *QueueConfig {
	return &QueueConfig{
		Name:       name,
		Durable:    true,
		Exclusive:  false,
		AutoDelete: false,
		MessageTTL: 24 * time.Hour,
		MaxLength:  1000000,
		MaxPriority: 10,
		Arguments:  make(map[string]interface{}),
	}
}

// ApplyQueueOptions applies a list of options to QueueConfig.
func ApplyQueueOptions(name string, opts ...QueueOption) *QueueConfig {
	config := DefaultQueueConfig(name)
	for _, opt := range opts {
		opt(config)
	}
	return config
}

// WithDurable sets the durable flag.
func WithDurable(durable bool) QueueOption {
	return func(c *QueueConfig) {
		c.Durable = durable
	}
}

// WithExclusive sets the exclusive flag.
func WithQueueExclusive(exclusive bool) QueueOption {
	return func(c *QueueConfig) {
		c.Exclusive = exclusive
	}
}

// WithAutoDelete sets the auto-delete flag.
func WithAutoDelete(autoDelete bool) QueueOption {
	return func(c *QueueConfig) {
		c.AutoDelete = autoDelete
	}
}

// WithMessageTTL sets the message TTL.
func WithMessageTTL(ttl time.Duration) QueueOption {
	return func(c *QueueConfig) {
		c.MessageTTL = ttl
	}
}

// WithMaxLength sets the maximum queue length.
func WithMaxLength(length int64) QueueOption {
	return func(c *QueueConfig) {
		c.MaxLength = length
	}
}

// WithMaxLengthBytes sets the maximum queue size in bytes.
func WithMaxLengthBytes(bytes int64) QueueOption {
	return func(c *QueueConfig) {
		c.MaxLengthBytes = bytes
	}
}

// WithDeadLetterQueue configures dead letter routing.
func WithDeadLetterQueue(exchange, routingKey string) QueueOption {
	return func(c *QueueConfig) {
		c.DeadLetterExchange = exchange
		c.DeadLetterRoutingKey = routingKey
	}
}

// WithMaxPriority sets the maximum priority level.
func WithQueueMaxPriority(priority int) QueueOption {
	return func(c *QueueConfig) {
		c.MaxPriority = priority
	}
}

// WithQueueArgument adds a custom argument.
func WithQueueArgument(key string, value interface{}) QueueOption {
	return func(c *QueueConfig) {
		if c.Arguments == nil {
			c.Arguments = make(map[string]interface{})
		}
		c.Arguments[key] = value
	}
}

// NewTask creates a new Task with default values.
func NewTask(taskType string, payload []byte) *Task {
	return &Task{
		ID:        generateUUID(),
		Type:      taskType,
		Payload:   payload,
		Priority:  PriorityNormal,
		MaxRetries: 3,
		CreatedAt: time.Now(),
		Status:    TaskStatusPending,
		Metadata:  make(map[string]string),
	}
}

// WithTaskPriority sets the task priority.
func (t *Task) WithTaskPriority(priority int) *Task {
	t.Priority = priority
	return t
}

// WithTaskMaxRetries sets the maximum retry count.
func (t *Task) WithTaskMaxRetries(max int) *Task {
	t.MaxRetries = max
	return t
}

// WithTaskDeadline sets the task deadline.
func (t *Task) WithTaskDeadline(deadline time.Time) *Task {
	t.Deadline = deadline
	return t
}

// WithTaskTimeout sets the task timeout.
func (t *Task) WithTaskTimeout(timeout time.Duration) *Task {
	t.Timeout = timeout
	return t
}

// WithTaskSchedule sets when the task should be executed.
func (t *Task) WithTaskSchedule(executeAt time.Time) *Task {
	t.ScheduledAt = executeAt
	t.Status = TaskStatusScheduled
	return t
}

// WithTaskMetadata adds metadata to the task.
func (t *Task) WithTaskMetadata(key, value string) *Task {
	if t.Metadata == nil {
		t.Metadata = make(map[string]string)
	}
	t.Metadata[key] = value
	return t
}

// WithTaskTraceID sets the trace ID.
func (t *Task) WithTaskTraceID(traceID string) *Task {
	t.TraceID = traceID
	return t
}

// WithTaskCorrelationID sets the correlation ID.
func (t *Task) WithTaskCorrelationID(correlationID string) *Task {
	t.CorrelationID = correlationID
	return t
}

// IsExpired checks if the task has exceeded its deadline.
func (t *Task) IsExpired() bool {
	if t.Deadline.IsZero() {
		return false
	}
	return time.Now().After(t.Deadline)
}

// CanRetry checks if the task can be retried.
func (t *Task) CanRetry() bool {
	return t.RetryCount < t.MaxRetries
}

// ShouldExecute checks if a scheduled task should be executed now.
func (t *Task) ShouldExecute() bool {
	if t.ScheduledAt.IsZero() {
		return true
	}
	return time.Now().After(t.ScheduledAt) || time.Now().Equal(t.ScheduledAt)
}

// Queue name constants for HelixAgent.
const (
	// Task queues
	QueueBackgroundTasks = "helixagent.tasks.background"
	QueueLLMRequests     = "helixagent.tasks.llm"
	QueueDebateRounds    = "helixagent.tasks.debate"
	QueueVerification    = "helixagent.tasks.verification"
	QueueNotifications   = "helixagent.tasks.notifications"

	// Dead letter queues
	QueueDeadLetter = "helixagent.dlq"
	QueueRetry      = "helixagent.retry"

	// Exchanges
	ExchangeTasks         = "helixagent.tasks"
	ExchangeEvents        = "helixagent.events"
	ExchangeNotifications = "helixagent.notifications"
	ExchangeDeadLetter    = "helixagent.dlx"
)
