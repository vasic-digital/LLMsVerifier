package failover

import (
	"errors"
	"sync"
	"time"
)

var (
	// ErrCircuitOpen is returned when the circuit breaker is open
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

// CircuitBreaker implements the circuit breaker pattern for provider failover
type CircuitBreaker struct {
	name            string
	state           CircuitBreakerState
	failureCount    int
	lastFailureTime time.Time
	nextRetryTime   time.Time
	successCount    int
	mu              sync.RWMutex

	// Configuration
	failureThreshold int           // Number of failures before opening
	recoveryTimeout  time.Duration // Time to wait before trying again
	successThreshold int           // Number of successes needed to close
	monitoringPeriod time.Duration // How often to check health
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string) *CircuitBreaker {
	return &CircuitBreaker{
		name:             name,
		state:            StateClosed,
		failureThreshold: 5,                // 5 failures
		recoveryTimeout:  30 * time.Second, // 30 seconds
		successThreshold: 3,                // 3 successes
		monitoringPeriod: 10 * time.Second, // 10 seconds
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mu.RLock()
	state := cb.state
	cb.mu.RUnlock()

	switch state {
	case StateOpen:
		return ErrCircuitOpen
	case StateHalfOpen:
		// Allow one request through
		err := fn()
		cb.recordResult(err == nil)
		return err
	case StateClosed:
		err := fn()
		cb.recordResult(err == nil)
		return err
	default:
		return ErrCircuitOpen
	}
}

// recordResult records the success or failure of a call
func (cb *CircuitBreaker) recordResult(success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if success {
		cb.onSuccess()
	} else {
		cb.onFailure()
	}
}

// onSuccess handles successful calls
func (cb *CircuitBreaker) onSuccess() {
	cb.failureCount = 0

	switch cb.state {
	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.state = StateClosed
			cb.successCount = 0
		}
	case StateClosed:
		// Stay closed
	}
}

// onFailure handles failed calls
func (cb *CircuitBreaker) onFailure() {
	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.state == StateClosed && cb.failureCount >= cb.failureThreshold {
		cb.state = StateOpen
		cb.nextRetryTime = time.Now().Add(cb.recoveryTimeout)
	} else if cb.state == StateHalfOpen {
		cb.state = StateOpen
		cb.nextRetryTime = time.Now().Add(cb.recoveryTimeout)
		cb.successCount = 0
	}
}

// CheckState checks if the circuit breaker should transition states
func (cb *CircuitBreaker) CheckState() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateOpen && time.Now().After(cb.nextRetryTime) {
		cb.state = StateHalfOpen
		cb.successCount = 0
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// IsAvailable returns true if the circuit breaker allows requests
func (cb *CircuitBreaker) IsAvailable() bool {
	cb.CheckState()
	state := cb.GetState()
	return state == StateClosed || state == StateHalfOpen
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failureCount = 0
	cb.successCount = 0
	cb.lastFailureTime = time.Time{}
	cb.nextRetryTime = time.Time{}
}
