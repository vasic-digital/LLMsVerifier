package messaging

import (
	"errors"
	"fmt"
)

// Standard messaging errors.
var (
	// Connection errors
	ErrNotConnected     = errors.New("not connected to broker")
	ErrConnectionFailed = errors.New("failed to connect to broker")
	ErrConnectionClosed = errors.New("connection closed")
	ErrConnectionLost   = errors.New("connection lost")

	// Configuration errors
	ErrNilConfig      = errors.New("config is nil")
	ErrInvalidConfig  = errors.New("invalid configuration")
	ErrInvalidHost    = errors.New("invalid or empty host")
	ErrInvalidPort    = errors.New("invalid port number")
	ErrInvalidTimeout = errors.New("invalid timeout value")
	ErrInvalidTopic   = errors.New("invalid or empty topic name")
	ErrInvalidQueue   = errors.New("invalid or empty queue name")
	ErrInvalidMessage = errors.New("invalid message")
	ErrEmptyPayload   = errors.New("message payload is empty")

	// Operation errors
	ErrPublishFailed     = errors.New("failed to publish message")
	ErrSubscribeFailed   = errors.New("failed to subscribe")
	ErrAckFailed         = errors.New("failed to acknowledge message")
	ErrNackFailed        = errors.New("failed to nack message")
	ErrTimeout           = errors.New("operation timed out")
	ErrQueueFull         = errors.New("queue is full")
	ErrQueueNotFound     = errors.New("queue not found")
	ErrTopicNotFound     = errors.New("topic not found")
	ErrPartitionNotFound = errors.New("partition not found")
	ErrConsumerNotFound  = errors.New("consumer not found")
	ErrNoConsumers       = errors.New("no consumers available")
	ErrMaxRetriesReached = errors.New("maximum retries reached")

	// Transaction errors
	ErrTxNotStarted = errors.New("transaction not started")
	ErrTxFailed     = errors.New("transaction failed")
	ErrTxRollback   = errors.New("transaction rolled back")

	// Dead letter errors
	ErrDeadLetterFull   = errors.New("dead letter queue is full")
	ErrDeadLetterFailed = errors.New("failed to move message to dead letter queue")

	// Authentication/Authorization errors
	ErrAuthFailed   = errors.New("authentication failed")
	ErrAccessDenied = errors.New("access denied")
	ErrInvalidToken = errors.New("invalid authentication token")
	ErrTokenExpired = errors.New("authentication token expired")

	// Serialization errors
	ErrSerializationFailed   = errors.New("failed to serialize message")
	ErrDeserializationFailed = errors.New("failed to deserialize message")

	// Resource errors
	ErrResourceExhausted = errors.New("resource exhausted")
	ErrRateLimited       = errors.New("rate limited")

	// State errors
	ErrAlreadyConnected   = errors.New("already connected")
	ErrAlreadySubscribed  = errors.New("already subscribed to topic")
	ErrSubscriptionClosed = errors.New("subscription is closed")
)

// BrokerError wraps errors with additional context.
type BrokerError struct {
	Op      string // Operation that failed
	Topic   string // Topic/queue involved
	Err     error  // Underlying error
	Code    string // Error code for classification
	Details map[string]interface{}
}

// Error implements the error interface.
func (e *BrokerError) Error() string {
	if e.Topic != "" {
		return fmt.Sprintf("%s on %s: %v", e.Op, e.Topic, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.Op, e.Err)
}

// Unwrap implements the errors.Unwrap interface.
func (e *BrokerError) Unwrap() error {
	return e.Err
}

// Is implements the errors.Is interface for custom error comparison.
func (e *BrokerError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// NewBrokerError creates a new BrokerError.
func NewBrokerError(op, topic string, err error) *BrokerError {
	return &BrokerError{
		Op:    op,
		Topic: topic,
		Err:   err,
	}
}

// WithCode adds an error code and returns the error for chaining.
func (e *BrokerError) WithCode(code string) *BrokerError {
	e.Code = code
	return e
}

// WithDetail adds a detail and returns the error for chaining.
func (e *BrokerError) WithDetail(key string, value interface{}) *BrokerError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// RetryableError indicates an error that may be resolved by retrying.
type RetryableError struct {
	Err        error
	RetryAfter int // Seconds to wait before retrying
}

// Error implements the error interface.
func (e *RetryableError) Error() string {
	return fmt.Sprintf("retryable error (retry after %ds): %v", e.RetryAfter, e.Err)
}

// Unwrap implements the errors.Unwrap interface.
func (e *RetryableError) Unwrap() error {
	return e.Err
}

// NewRetryableError creates a new RetryableError.
func NewRetryableError(err error, retryAfter int) *RetryableError {
	return &RetryableError{
		Err:        err,
		RetryAfter: retryAfter,
	}
}

// IsRetryable checks if an error is retryable.
func IsRetryable(err error) bool {
	var retryable *RetryableError
	if errors.As(err, &retryable) {
		return true
	}
	// Also check for specific error types that are inherently retryable
	return errors.Is(err, ErrConnectionLost) ||
		errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrRateLimited)
}

// IsConnectionError checks if an error is related to connection issues.
func IsConnectionError(err error) bool {
	return errors.Is(err, ErrNotConnected) ||
		errors.Is(err, ErrConnectionFailed) ||
		errors.Is(err, ErrConnectionClosed) ||
		errors.Is(err, ErrConnectionLost)
}

// IsConfigError checks if an error is related to configuration.
func IsConfigError(err error) bool {
	return errors.Is(err, ErrNilConfig) ||
		errors.Is(err, ErrInvalidConfig) ||
		errors.Is(err, ErrInvalidHost) ||
		errors.Is(err, ErrInvalidPort) ||
		errors.Is(err, ErrInvalidTimeout)
}

// IsAuthError checks if an error is related to authentication/authorization.
func IsAuthError(err error) bool {
	return errors.Is(err, ErrAuthFailed) ||
		errors.Is(err, ErrAccessDenied) ||
		errors.Is(err, ErrInvalidToken) ||
		errors.Is(err, ErrTokenExpired)
}
