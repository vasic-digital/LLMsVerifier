package messaging

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// HubConfig contains configuration for the MessagingHub.
type HubConfig struct {
	// TaskQueueConfig for RabbitMQ-style task queue.
	TaskQueueConfig *BrokerConfig `json:"task_queue" yaml:"task_queue"`

	// EventStreamConfig for Kafka-style event streaming.
	EventStreamConfig *BrokerConfig `json:"event_stream" yaml:"event_stream"`

	// EnableFallback enables in-memory fallback when brokers are unavailable.
	EnableFallback bool `json:"enable_fallback" yaml:"enable_fallback"`

	// FallbackTimeout is how long to try primary before using fallback.
	FallbackTimeout time.Duration `json:"fallback_timeout" yaml:"fallback_timeout"`

	// RetryConfig for retry behavior.
	RetryConfig *RetryConfig `json:"retry" yaml:"retry"`

	// CircuitBreakerConfig for circuit breaker behavior.
	CircuitBreakerConfig *CircuitBreakerConfig `json:"circuit_breaker" yaml:"circuit_breaker"`
}

// RetryConfig contains retry configuration.
type RetryConfig struct {
	MaxRetries     int           `json:"max_retries" yaml:"max_retries"`
	InitialBackoff time.Duration `json:"initial_backoff" yaml:"initial_backoff"`
	MaxBackoff     time.Duration `json:"max_backoff" yaml:"max_backoff"`
	BackoffFactor  float64       `json:"backoff_factor" yaml:"backoff_factor"`
}

// CircuitBreakerConfig contains circuit breaker configuration.
type CircuitBreakerConfig struct {
	Enabled          bool          `json:"enabled" yaml:"enabled"`
	FailureThreshold int           `json:"failure_threshold" yaml:"failure_threshold"`
	SuccessThreshold int           `json:"success_threshold" yaml:"success_threshold"`
	Timeout          time.Duration `json:"timeout" yaml:"timeout"`
}

// DefaultHubConfig returns a HubConfig with sensible defaults.
func DefaultHubConfig() *HubConfig {
	return &HubConfig{
		TaskQueueConfig:   DefaultBrokerConfig(),
		EventStreamConfig: DefaultBrokerConfig(),
		EnableFallback:    true,
		FallbackTimeout:   5 * time.Second,
		RetryConfig: &RetryConfig{
			MaxRetries:     3,
			InitialBackoff: 100 * time.Millisecond,
			MaxBackoff:     10 * time.Second,
			BackoffFactor:  2.0,
		},
		CircuitBreakerConfig: &CircuitBreakerConfig{
			Enabled:          true,
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          30 * time.Second,
		},
	}
}

// MessagingHub provides unified access to task queue and event stream brokers.
type MessagingHub struct {
	mu sync.RWMutex

	config *HubConfig
	logger *logrus.Logger

	// Brokers
	taskQueue   TaskQueueBroker
	eventStream EventStreamBroker
	fallback    MessageBroker

	// State
	initialized bool
	closed      bool

	// Metrics
	metrics *HubMetrics

	// Circuit breakers
	taskQueueCircuit   *CircuitBreaker
	eventStreamCircuit *CircuitBreaker

	// Middleware
	middleware []MessageMiddleware

	// Subscriptions tracking
	subscriptions map[string]Subscription
}

// HubMetrics contains metrics for the messaging hub.
type HubMetrics struct {
	TaskQueueMetrics   *BrokerMetrics `json:"task_queue_metrics"`
	EventStreamMetrics *BrokerMetrics `json:"event_stream_metrics"`
	FallbackMetrics    *BrokerMetrics `json:"fallback_metrics,omitempty"`
	FallbacksUsed      int64          `json:"fallbacks_used"`
	CircuitBreaksOpen  int64          `json:"circuit_breaks_open"`
}

// MessageMiddleware is a function that wraps message handling.
type MessageMiddleware func(MessageHandler) MessageHandler

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	mu sync.Mutex

	name             string
	state            CircuitState
	failures         int
	successes        int
	failureThreshold int
	successThreshold int
	timeout          time.Duration
	lastFailure      time.Time
}

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// NewMessagingHub creates a new MessagingHub.
func NewMessagingHub(config *HubConfig, logger *logrus.Logger) *MessagingHub {
	if config == nil {
		config = DefaultHubConfig()
	}
	if logger == nil {
		logger = logrus.New()
	}

	hub := &MessagingHub{
		config:        config,
		logger:        logger,
		metrics:       &HubMetrics{},
		subscriptions: make(map[string]Subscription),
		middleware:    make([]MessageMiddleware, 0),
	}

	if config.CircuitBreakerConfig != nil && config.CircuitBreakerConfig.Enabled {
		hub.taskQueueCircuit = NewCircuitBreaker("task_queue", config.CircuitBreakerConfig)
		hub.eventStreamCircuit = NewCircuitBreaker("event_stream", config.CircuitBreakerConfig)
	}

	return hub
}

// Initialize initializes the messaging hub and connects to brokers.
func (h *MessagingHub) Initialize(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.initialized {
		return nil
	}

	h.logger.Info("Initializing messaging hub")

	// Initialize task queue if configured
	if h.config.TaskQueueConfig != nil && h.config.TaskQueueConfig.Type != BrokerTypeInMemory {
		if err := h.initTaskQueue(ctx); err != nil {
			h.logger.WithError(err).Warn("Failed to initialize task queue")
			if !h.config.EnableFallback {
				return fmt.Errorf("task queue initialization failed: %w", err)
			}
		}
	}

	// Initialize event stream if configured
	if h.config.EventStreamConfig != nil && h.config.EventStreamConfig.Type != BrokerTypeInMemory {
		if err := h.initEventStream(ctx); err != nil {
			h.logger.WithError(err).Warn("Failed to initialize event stream")
			if !h.config.EnableFallback {
				return fmt.Errorf("event stream initialization failed: %w", err)
			}
		}
	}

	// Initialize fallback if enabled
	if h.config.EnableFallback {
		if err := h.initFallback(ctx); err != nil {
			return fmt.Errorf("fallback initialization failed: %w", err)
		}
	}

	h.initialized = true
	h.logger.Info("Messaging hub initialized successfully")

	return nil
}

// Close closes all broker connections.
func (h *MessagingHub) Close(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return nil
	}

	h.logger.Info("Closing messaging hub")

	var errs []error

	// Close all subscriptions
	for name, sub := range h.subscriptions {
		if err := sub.Unsubscribe(); err != nil {
			errs = append(errs, fmt.Errorf("failed to unsubscribe %s: %w", name, err))
		}
	}

	// Close task queue
	if h.taskQueue != nil {
		if err := h.taskQueue.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("task queue close failed: %w", err))
		}
	}

	// Close event stream
	if h.eventStream != nil {
		if err := h.eventStream.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("event stream close failed: %w", err))
		}
	}

	// Close fallback
	if h.fallback != nil {
		if err := h.fallback.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("fallback close failed: %w", err))
		}
	}

	h.closed = true

	if len(errs) > 0 {
		return fmt.Errorf("errors during close: %v", errs)
	}

	h.logger.Info("Messaging hub closed successfully")
	return nil
}

// PublishTask publishes a task to the task queue.
func (h *MessagingHub) PublishTask(ctx context.Context, queue string, task *Task) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.initialized {
		return ErrNotConnected
	}

	// Try primary task queue
	if h.taskQueue != nil && h.shouldUsePrimary(h.taskQueueCircuit) {
		if err := h.taskQueue.EnqueueTask(ctx, queue, task); err != nil {
			h.recordFailure(h.taskQueueCircuit)
			if !h.config.EnableFallback {
				return fmt.Errorf("task queue publish failed: %w", err)
			}
			h.logger.WithError(err).Warn("Task queue publish failed, using fallback")
		} else {
			h.recordSuccess(h.taskQueueCircuit)
			return nil
		}
	}

	// Use fallback
	if h.fallback != nil {
		msg := h.taskToMessage(task)
		if err := h.fallback.Publish(ctx, queue, msg); err != nil {
			return fmt.Errorf("fallback publish failed: %w", err)
		}
		h.metrics.FallbacksUsed++
		return nil
	}

	return ErrNotConnected
}

// PublishEvent publishes an event to the event stream.
func (h *MessagingHub) PublishEvent(ctx context.Context, topic string, event *Event) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.initialized {
		return ErrNotConnected
	}

	// Try primary event stream
	if h.eventStream != nil && h.shouldUsePrimary(h.eventStreamCircuit) {
		if err := h.eventStream.PublishEvent(ctx, topic, event); err != nil {
			h.recordFailure(h.eventStreamCircuit)
			if !h.config.EnableFallback {
				return fmt.Errorf("event stream publish failed: %w", err)
			}
			h.logger.WithError(err).Warn("Event stream publish failed, using fallback")
		} else {
			h.recordSuccess(h.eventStreamCircuit)
			return nil
		}
	}

	// Use fallback
	if h.fallback != nil {
		msg := h.eventToMessage(event)
		if err := h.fallback.Publish(ctx, topic, msg); err != nil {
			return fmt.Errorf("fallback publish failed: %w", err)
		}
		h.metrics.FallbacksUsed++
		return nil
	}

	return ErrNotConnected
}

// SubscribeTasks subscribes to tasks from a queue.
func (h *MessagingHub) SubscribeTasks(ctx context.Context, queue string, handler TaskHandler, opts ...SubscribeOption) (Subscription, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.initialized {
		return nil, ErrNotConnected
	}

	// Wrap handler with middleware
	wrappedHandler := h.wrapTaskHandler(handler)

	// Try primary task queue
	if h.taskQueue != nil && h.taskQueue.IsConnected() {
		sub, err := h.taskQueue.Subscribe(ctx, queue, wrappedHandler, opts...)
		if err == nil {
			h.subscriptions[queue] = sub
			return sub, nil
		}
		h.logger.WithError(err).Warn("Task queue subscribe failed, using fallback")
	}

	// Use fallback
	if h.fallback != nil {
		sub, err := h.fallback.Subscribe(ctx, queue, wrappedHandler, opts...)
		if err != nil {
			return nil, fmt.Errorf("fallback subscribe failed: %w", err)
		}
		h.subscriptions[queue] = sub
		return sub, nil
	}

	return nil, ErrNotConnected
}

// SubscribeEvents subscribes to events from a topic.
func (h *MessagingHub) SubscribeEvents(ctx context.Context, topic string, handler EventHandler, opts ...SubscribeOption) (Subscription, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.initialized {
		return nil, ErrNotConnected
	}

	// Wrap handler
	wrappedHandler := h.wrapEventHandler(handler)

	// Try primary event stream
	if h.eventStream != nil && h.eventStream.IsConnected() {
		sub, err := h.eventStream.Subscribe(ctx, topic, wrappedHandler, opts...)
		if err == nil {
			h.subscriptions[topic] = sub
			return sub, nil
		}
		h.logger.WithError(err).Warn("Event stream subscribe failed, using fallback")
	}

	// Use fallback
	if h.fallback != nil {
		sub, err := h.fallback.Subscribe(ctx, topic, wrappedHandler, opts...)
		if err != nil {
			return nil, fmt.Errorf("fallback subscribe failed: %w", err)
		}
		h.subscriptions[topic] = sub
		return sub, nil
	}

	return nil, ErrNotConnected
}

// AddMiddleware adds middleware to the hub.
func (h *MessagingHub) AddMiddleware(mw MessageMiddleware) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.middleware = append(h.middleware, mw)
}

// GetMetrics returns hub metrics.
func (h *MessagingHub) GetMetrics() *HubMetrics {
	h.mu.RLock()
	defer h.mu.RUnlock()

	metrics := &HubMetrics{
		FallbacksUsed:     h.metrics.FallbacksUsed,
		CircuitBreaksOpen: h.metrics.CircuitBreaksOpen,
	}

	if h.taskQueue != nil {
		metrics.TaskQueueMetrics = h.taskQueue.GetMetrics()
	}
	if h.eventStream != nil {
		metrics.EventStreamMetrics = h.eventStream.GetMetrics()
	}
	if h.fallback != nil {
		metrics.FallbackMetrics = h.fallback.GetMetrics()
	}

	return metrics
}

// TaskQueue returns the task queue broker (may be nil).
func (h *MessagingHub) TaskQueue() TaskQueueBroker {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.taskQueue
}

// EventStream returns the event stream broker (may be nil).
func (h *MessagingHub) EventStream() EventStreamBroker {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.eventStream
}

// SetTaskQueue sets the task queue broker.
func (h *MessagingHub) SetTaskQueue(broker TaskQueueBroker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.taskQueue = broker
}

// SetEventStream sets the event stream broker.
func (h *MessagingHub) SetEventStream(broker EventStreamBroker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.eventStream = broker
}

// SetFallback sets the fallback broker.
func (h *MessagingHub) SetFallback(broker MessageBroker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.fallback = broker
}

// Private methods

func (h *MessagingHub) initTaskQueue(ctx context.Context) error {
	// This will be implemented when we add RabbitMQ broker
	return nil
}

func (h *MessagingHub) initEventStream(ctx context.Context) error {
	// This will be implemented when we add Kafka broker
	return nil
}

func (h *MessagingHub) initFallback(ctx context.Context) error {
	// This will be implemented when we add in-memory broker
	return nil
}

func (h *MessagingHub) shouldUsePrimary(cb *CircuitBreaker) bool {
	if cb == nil {
		return true
	}
	return cb.Allow()
}

func (h *MessagingHub) recordFailure(cb *CircuitBreaker) {
	if cb != nil {
		cb.RecordFailure()
	}
}

func (h *MessagingHub) recordSuccess(cb *CircuitBreaker) {
	if cb != nil {
		cb.RecordSuccess()
	}
}

func (h *MessagingHub) taskToMessage(task *Task) *Message {
	return &Message{
		ID:            task.ID,
		Type:          task.Type,
		Payload:       task.Payload,
		Timestamp:     task.CreatedAt,
		Priority:      task.Priority,
		TraceID:       task.TraceID,
		CorrelationID: task.CorrelationID,
		Headers:       task.Metadata,
	}
}

func (h *MessagingHub) eventToMessage(event *Event) *Message {
	return &Message{
		ID:            event.ID,
		Type:          event.Type,
		Payload:       event.Data,
		Timestamp:     event.Timestamp,
		Key:           event.Key,
		TraceID:       event.TraceID,
		CorrelationID: event.CorrelationID,
		Headers:       event.Headers,
	}
}

func (h *MessagingHub) wrapTaskHandler(handler TaskHandler) MessageHandler {
	return func(ctx context.Context, msg *Message) error {
		task := &Task{
			ID:            msg.ID,
			Type:          msg.Type,
			Payload:       msg.Payload,
			Priority:      msg.Priority,
			CreatedAt:     msg.Timestamp,
			TraceID:       msg.TraceID,
			CorrelationID: msg.CorrelationID,
			Metadata:      msg.Headers,
			DeliveryTag:   msg.DeliveryTag,
		}
		return handler(ctx, task)
	}
}

func (h *MessagingHub) wrapEventHandler(handler EventHandler) MessageHandler {
	return func(ctx context.Context, msg *Message) error {
		event := &Event{
			ID:            msg.ID,
			Type:          msg.Type,
			Data:          msg.Payload,
			Timestamp:     msg.Timestamp,
			Key:           msg.Key,
			TraceID:       msg.TraceID,
			CorrelationID: msg.CorrelationID,
			Headers:       msg.Headers,
			Partition:     msg.Partition,
			Offset:        msg.Offset,
		}
		return handler(ctx, event)
	}
}

// Circuit Breaker implementation

// NewCircuitBreaker creates a new CircuitBreaker.
func NewCircuitBreaker(name string, config *CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		name:             name,
		state:            CircuitClosed,
		failureThreshold: config.FailureThreshold,
		successThreshold: config.SuccessThreshold,
		timeout:          config.Timeout,
	}
}

// Allow checks if the circuit allows requests.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(cb.lastFailure) > cb.timeout {
			cb.state = CircuitHalfOpen
			cb.successes = 0
			return true
		}
		return false
	case CircuitHalfOpen:
		return true
	}
	return true
}

// RecordSuccess records a successful request.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0

	if cb.state == CircuitHalfOpen {
		cb.successes++
		if cb.successes >= cb.successThreshold {
			cb.state = CircuitClosed
		}
	}
}

// RecordFailure records a failed request.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.state == CircuitHalfOpen || cb.failures >= cb.failureThreshold {
		cb.state = CircuitOpen
	}
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}
