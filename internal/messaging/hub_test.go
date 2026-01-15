package messaging

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultHubConfig(t *testing.T) {
	config := DefaultHubConfig()

	require.NotNil(t, config)
	assert.NotNil(t, config.TaskQueueConfig)
	assert.NotNil(t, config.EventStreamConfig)
	assert.True(t, config.EnableFallback)
	assert.Equal(t, 5*time.Second, config.FallbackTimeout)
	assert.NotNil(t, config.RetryConfig)
	assert.NotNil(t, config.CircuitBreakerConfig)
}

func TestRetryConfig(t *testing.T) {
	config := DefaultHubConfig()

	assert.Equal(t, 3, config.RetryConfig.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, config.RetryConfig.InitialBackoff)
	assert.Equal(t, 10*time.Second, config.RetryConfig.MaxBackoff)
	assert.Equal(t, 2.0, config.RetryConfig.BackoffFactor)
}

func TestCircuitBreakerConfig(t *testing.T) {
	config := DefaultHubConfig()

	assert.True(t, config.CircuitBreakerConfig.Enabled)
	assert.Equal(t, 5, config.CircuitBreakerConfig.FailureThreshold)
	assert.Equal(t, 2, config.CircuitBreakerConfig.SuccessThreshold)
	assert.Equal(t, 30*time.Second, config.CircuitBreakerConfig.Timeout)
}

func TestNewMessagingHub(t *testing.T) {
	hub := NewMessagingHub(nil, nil)

	require.NotNil(t, hub)
	assert.NotNil(t, hub.config)
	assert.NotNil(t, hub.logger)
	assert.NotNil(t, hub.metrics)
	assert.NotNil(t, hub.subscriptions)
	assert.False(t, hub.initialized)
	assert.False(t, hub.closed)
}

func TestNewMessagingHubWithConfig(t *testing.T) {
	config := &HubConfig{
		EnableFallback:  false,
		FallbackTimeout: 10 * time.Second,
	}
	logger := logrus.New()

	hub := NewMessagingHub(config, logger)

	require.NotNil(t, hub)
	assert.False(t, hub.config.EnableFallback)
	assert.Equal(t, 10*time.Second, hub.config.FallbackTimeout)
}

func TestMessagingHubSetBrokers(t *testing.T) {
	hub := NewMessagingHub(nil, nil)

	// Test SetTaskQueue
	hub.SetTaskQueue(nil)
	assert.Nil(t, hub.TaskQueue())

	// Test SetEventStream
	hub.SetEventStream(nil)
	assert.Nil(t, hub.EventStream())

	// Test SetFallback
	hub.SetFallback(nil)
}

func TestMessagingHubAddMiddleware(t *testing.T) {
	hub := NewMessagingHub(nil, nil)

	middleware := func(next MessageHandler) MessageHandler {
		return func(ctx context.Context, msg *Message) error {
			return next(ctx, msg)
		}
	}

	hub.AddMiddleware(middleware)
	assert.Len(t, hub.middleware, 1)

	hub.AddMiddleware(middleware)
	assert.Len(t, hub.middleware, 2)
}

func TestMessagingHubGetMetrics(t *testing.T) {
	hub := NewMessagingHub(nil, nil)
	metrics := hub.GetMetrics()

	require.NotNil(t, metrics)
	assert.Equal(t, int64(0), metrics.FallbacksUsed)
	assert.Equal(t, int64(0), metrics.CircuitBreaksOpen)
}

func TestMessagingHubInitializeWithoutBrokers(t *testing.T) {
	config := DefaultHubConfig()
	config.TaskQueueConfig.Type = BrokerTypeInMemory
	config.EventStreamConfig.Type = BrokerTypeInMemory
	config.EnableFallback = false

	hub := NewMessagingHub(config, nil)
	err := hub.Initialize(context.Background())

	assert.NoError(t, err)
	assert.True(t, hub.initialized)
}

func TestMessagingHubClose(t *testing.T) {
	hub := NewMessagingHub(nil, nil)
	hub.initialized = true

	err := hub.Close(context.Background())

	assert.NoError(t, err)
	assert.True(t, hub.closed)
}

func TestMessagingHubCloseIdempotent(t *testing.T) {
	hub := NewMessagingHub(nil, nil)
	hub.initialized = true

	// First close
	err := hub.Close(context.Background())
	assert.NoError(t, err)

	// Second close should be no-op
	err = hub.Close(context.Background())
	assert.NoError(t, err)
}

func TestMessagingHubPublishTaskNotInitialized(t *testing.T) {
	hub := NewMessagingHub(nil, nil)

	task := &Task{
		ID:   "test-task",
		Type: "test",
	}

	err := hub.PublishTask(context.Background(), "test-queue", task)
	assert.Error(t, err)
	assert.Equal(t, ErrNotConnected, err)
}

func TestMessagingHubPublishEventNotInitialized(t *testing.T) {
	hub := NewMessagingHub(nil, nil)

	event := &Event{
		ID:   "test-event",
		Type: "test",
	}

	err := hub.PublishEvent(context.Background(), "test-topic", event)
	assert.Error(t, err)
	assert.Equal(t, ErrNotConnected, err)
}

func TestMessagingHubSubscribeTasksNotInitialized(t *testing.T) {
	hub := NewMessagingHub(nil, nil)

	handler := func(ctx context.Context, task *Task) error {
		return nil
	}

	sub, err := hub.SubscribeTasks(context.Background(), "test-queue", handler)
	assert.Error(t, err)
	assert.Nil(t, sub)
	assert.Equal(t, ErrNotConnected, err)
}

func TestMessagingHubSubscribeEventsNotInitialized(t *testing.T) {
	hub := NewMessagingHub(nil, nil)

	handler := func(ctx context.Context, event *Event) error {
		return nil
	}

	sub, err := hub.SubscribeEvents(context.Background(), "test-topic", handler)
	assert.Error(t, err)
	assert.Nil(t, sub)
	assert.Equal(t, ErrNotConnected, err)
}

// Circuit Breaker Tests

func TestNewCircuitBreaker(t *testing.T) {
	config := &CircuitBreakerConfig{
		Enabled:          true,
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Timeout:          30 * time.Second,
	}

	cb := NewCircuitBreaker("test", config)

	require.NotNil(t, cb)
	assert.Equal(t, "test", cb.name)
	assert.Equal(t, CircuitClosed, cb.State())
	assert.Equal(t, 5, cb.failureThreshold)
	assert.Equal(t, 2, cb.successThreshold)
	assert.Equal(t, 30*time.Second, cb.timeout)
}

func TestCircuitBreakerAllow(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}

	cb := NewCircuitBreaker("test", config)

	// Closed circuit should allow
	assert.True(t, cb.Allow())
	assert.Equal(t, CircuitClosed, cb.State())
}

func TestCircuitBreakerOpenOnFailures(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}

	cb := NewCircuitBreaker("test", config)

	// Record failures until threshold
	cb.RecordFailure()
	assert.Equal(t, CircuitClosed, cb.State())

	cb.RecordFailure()
	assert.Equal(t, CircuitClosed, cb.State())

	cb.RecordFailure()
	assert.Equal(t, CircuitOpen, cb.State())

	// Open circuit should not allow
	assert.False(t, cb.Allow())
}

func TestCircuitBreakerHalfOpen(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	}

	cb := NewCircuitBreaker("test", config)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, CircuitOpen, cb.State())

	// Wait for timeout
	time.Sleep(60 * time.Millisecond)

	// Should transition to half-open
	assert.True(t, cb.Allow())
	assert.Equal(t, CircuitHalfOpen, cb.State())
}

func TestCircuitBreakerCloseAfterSuccesses(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 2,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
	}

	cb := NewCircuitBreaker("test", config)

	// Open the circuit
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, CircuitOpen, cb.State())

	// Wait for timeout and transition to half-open
	time.Sleep(60 * time.Millisecond)
	cb.Allow()
	assert.Equal(t, CircuitHalfOpen, cb.State())

	// Record successes to close
	cb.RecordSuccess()
	assert.Equal(t, CircuitHalfOpen, cb.State())

	cb.RecordSuccess()
	assert.Equal(t, CircuitClosed, cb.State())
}

func TestCircuitBreakerSuccessResetsFailures(t *testing.T) {
	config := &CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          100 * time.Millisecond,
	}

	cb := NewCircuitBreaker("test", config)

	// Record some failures
	cb.RecordFailure()
	cb.RecordFailure()

	// Success should reset failures
	cb.RecordSuccess()

	// Need 3 more failures to open
	cb.RecordFailure()
	cb.RecordFailure()
	assert.Equal(t, CircuitClosed, cb.State())

	cb.RecordFailure()
	assert.Equal(t, CircuitOpen, cb.State())
}

// Task/Event Conversion Tests

func TestTaskToMessage(t *testing.T) {
	hub := NewMessagingHub(nil, nil)

	task := &Task{
		ID:            "task-123",
		Type:          "verification",
		Payload:       []byte(`{"test": true}`),
		Priority:      5,
		CreatedAt:     time.Now(),
		TraceID:       "trace-456",
		CorrelationID: "corr-789",
		Metadata:      map[string]string{"key": "value"},
	}

	msg := hub.taskToMessage(task)

	assert.Equal(t, task.ID, msg.ID)
	assert.Equal(t, task.Type, msg.Type)
	assert.Equal(t, task.Payload, msg.Payload)
	assert.Equal(t, task.Priority, msg.Priority)
	assert.Equal(t, task.CreatedAt, msg.Timestamp)
	assert.Equal(t, task.TraceID, msg.TraceID)
	assert.Equal(t, task.CorrelationID, msg.CorrelationID)
	assert.Equal(t, task.Metadata, msg.Headers)
}

func TestEventToMessage(t *testing.T) {
	hub := NewMessagingHub(nil, nil)

	event := &Event{
		ID:            "event-123",
		Type:          "provider.scored",
		Data:          []byte(`{"score": 8.5}`),
		Key:           "provider-456",
		Timestamp:     time.Now(),
		TraceID:       "trace-789",
		CorrelationID: "corr-012",
		Headers:       map[string]string{"key": "value"},
	}

	msg := hub.eventToMessage(event)

	assert.Equal(t, event.ID, msg.ID)
	assert.Equal(t, event.Type, msg.Type)
	assert.Equal(t, event.Data, msg.Payload)
	assert.Equal(t, event.Timestamp, msg.Timestamp)
	assert.Equal(t, event.Key, msg.Key)
	assert.Equal(t, event.TraceID, msg.TraceID)
	assert.Equal(t, event.CorrelationID, msg.CorrelationID)
	assert.Equal(t, event.Headers, msg.Headers)
}

// Circuit State Tests

func TestCircuitStateValues(t *testing.T) {
	assert.Equal(t, CircuitState(0), CircuitClosed)
	assert.Equal(t, CircuitState(1), CircuitOpen)
	assert.Equal(t, CircuitState(2), CircuitHalfOpen)
}
