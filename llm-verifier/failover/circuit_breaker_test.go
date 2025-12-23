package failover

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var (
	testSuccess = errors.New("success")
	testError   = errors.New("test error")
)

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker("test-breaker")

	assert.NotNil(t, cb, "CircuitBreaker should not be nil")
	assert.Equal(t, "test-breaker", cb.name, "Name should match")
	assert.Equal(t, StateClosed, cb.state, "Initial state should be closed")
	assert.Equal(t, 0, cb.failureCount, "Initial failure count should be 0")
	assert.Equal(t, 0, cb.successCount, "Initial success count should be 0")
	assert.Equal(t, 5, cb.failureThreshold, "Default failure threshold should be 5")
	assert.Equal(t, 30*time.Second, cb.recoveryTimeout, "Default recovery timeout should be 30s")
	assert.Equal(t, 3, cb.successThreshold, "Default success threshold should be 3")
	assert.Equal(t, 10*time.Second, cb.monitoringPeriod, "Default monitoring period should be 10s")
}

func TestCircuitBreakerCallSuccess(t *testing.T) {
	cb := NewCircuitBreaker("test-breaker")

	callCount := 0
	fn := func() error {
		callCount++
		return nil
	}

	err := cb.Call(fn)
	assert.NoError(t, err, "Call should succeed")
	assert.Equal(t, 1, callCount, "Function should be called once")

	cb.mu.RLock()
	assert.Equal(t, 0, cb.failureCount, "Failure count should be 0")
	assert.Equal(t, StateClosed, cb.state, "State should remain closed")
	cb.mu.RUnlock()
}

func TestCircuitBreakerCallFailure(t *testing.T) {
	cb := NewCircuitBreaker("test-breaker")

	fn := func() error {
		return testError
	}

	err := cb.Call(fn)
	assert.Error(t, err, "Call should fail")

	cb.mu.RLock()
	assert.Equal(t, 1, cb.failureCount, "Failure count should be 1")
	assert.False(t, cb.lastFailureTime.IsZero(), "Last failure time should be set")
	cb.mu.RUnlock()
}

func TestCircuitBreakerOpensOnThreshold(t *testing.T) {
	cb := NewCircuitBreaker("test-breaker")
	cb.failureThreshold = 3 // Open after 3 failures

	fn := func() error {
		return testError
	}

	// First failure
	err := cb.Call(fn)
	assert.Error(t, err)
	assert.Equal(t, StateClosed, cb.GetState(), "State should be closed")

	// Second failure
	err = cb.Call(fn)
	assert.Error(t, err)
	assert.Equal(t, StateClosed, cb.GetState(), "State should be closed")

	// Third failure - should open circuit
	err = cb.Call(fn)
	assert.Error(t, err)
	assert.Equal(t, StateOpen, cb.GetState(), "State should be open")
	assert.False(t, cb.nextRetryTime.IsZero(), "Next retry time should be set")
}

func TestCircuitBreakerBlocksWhenOpen(t *testing.T) {
	cb := NewCircuitBreaker("test-breaker")

	// Force circuit open
	cb.mu.Lock()
	cb.state = StateOpen
	cb.mu.Unlock()

	callCount := 0
	fn := func() error {
		callCount++
		return nil
	}

	err := cb.Call(fn)
	assert.Error(t, err, "Call should fail when circuit is open")
	assert.Equal(t, ErrCircuitOpen, err, "Error should be ErrCircuitOpen")
	assert.Equal(t, 0, callCount, "Function should not be called")
}

func TestCircuitBreakerTransitionsToHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker("test-breaker")

	// Set circuit as open with retry time in past
	cb.mu.Lock()
	cb.state = StateOpen
	cb.nextRetryTime = time.Now().Add(-1 * time.Second)
	cb.mu.Unlock()

	// CheckState should transition to half-open
	cb.CheckState()
	assert.Equal(t, StateHalfOpen, cb.GetState(), "State should transition to half-open")
}

func TestCircuitBreakerClosesOnSuccessInHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker("test-breaker")
	cb.successThreshold = 2 // Need 2 successes to close

	// Start in half-open state
	cb.mu.Lock()
	cb.state = StateHalfOpen
	cb.mu.Unlock()

	// First success
	err := cb.Call(func() error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, StateHalfOpen, cb.GetState(), "State should remain half-open after first success")

	// Second success - should close circuit
	err = cb.Call(func() error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.GetState(), "State should be closed after second success")
	assert.Equal(t, 0, cb.successCount, "Success count should be reset")
	assert.Equal(t, 0, cb.failureCount, "Failure count should be reset")
}

func TestCircuitBreakerReopensOnFailureInHalfOpen(t *testing.T) {
	cb := NewCircuitBreaker("test-breaker")

	// Start in half-open state
	cb.mu.Lock()
	cb.state = StateHalfOpen
	cb.mu.Unlock()

	err := cb.Call(func() error { return testError })
	assert.Error(t, err)
	assert.Equal(t, StateOpen, cb.GetState(), "State should re-open on failure")
	assert.Equal(t, 0, cb.successCount, "Success count should be reset")
}

func TestCircuitBreakerGetState(t *testing.T) {
	cb := NewCircuitBreaker("test-breaker")

	assert.Equal(t, StateClosed, cb.GetState(), "Initial state should be closed")

	cb.mu.Lock()
	cb.state = StateOpen
	cb.mu.Unlock()

	assert.Equal(t, StateOpen, cb.GetState(), "State should be open")

	cb.mu.Lock()
	cb.state = StateHalfOpen
	cb.mu.Unlock()

	assert.Equal(t, StateHalfOpen, cb.GetState(), "State should be half-open")
}

func TestCircuitBreakerIsAvailable(t *testing.T) {
	cb := NewCircuitBreaker("test-breaker")

	// Closed state
	cb.mu.Lock()
	cb.state = StateClosed
	cb.mu.Unlock()
	assert.True(t, cb.IsAvailable(), "Should be available when closed")

	// Open state
	cb.mu.Lock()
	cb.state = StateOpen
	cb.nextRetryTime = time.Now().Add(1 * time.Minute)
	cb.mu.Unlock()
	assert.False(t, cb.IsAvailable(), "Should not be available when open")

	// Half-open state
	cb.mu.Lock()
	cb.state = StateHalfOpen
	cb.mu.Unlock()
	assert.True(t, cb.IsAvailable(), "Should be available when half-open")
}

func TestCircuitBreakerIsAvailableTransitions(t *testing.T) {
	cb := NewCircuitBreaker("test-breaker")

	// Set as open with retry time in past
	cb.mu.Lock()
	cb.state = StateOpen
	cb.nextRetryTime = time.Now().Add(-1 * time.Second)
	cb.mu.Unlock()

	// IsAvailable should trigger CheckState
	assert.True(t, cb.IsAvailable(), "Should become available after retry time")
	assert.Equal(t, StateHalfOpen, cb.GetState(), "State should transition to half-open")
}

func TestCircuitBreakerReset(t *testing.T) {
	cb := NewCircuitBreaker("test-breaker")

	// Set circuit to failed state
	cb.mu.Lock()
	cb.state = StateOpen
	cb.failureCount = 10
	cb.successCount = 5
	cb.lastFailureTime = time.Now()
	cb.nextRetryTime = time.Now().Add(1 * time.Hour)
	cb.mu.Unlock()

	// Reset
	cb.Reset()

	assert.Equal(t, StateClosed, cb.state, "State should be closed")
	assert.Equal(t, 0, cb.failureCount, "Failure count should be 0")
	assert.Equal(t, 0, cb.successCount, "Success count should be 0")
	assert.True(t, cb.lastFailureTime.IsZero(), "Last failure time should be zero")
	assert.True(t, cb.nextRetryTime.IsZero(), "Next retry time should be zero")
}

func TestCircuitBreakerConcurrentCalls(t *testing.T) {
	cb := NewCircuitBreaker("test-breaker")

	var wg sync.WaitGroup
	callCount := 0
	successCount := 0

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			callCount++
			err := cb.Call(func() error { return nil })
			if err == nil {
				successCount++
			}
		}()
	}

	wg.Wait()

	assert.GreaterOrEqual(t, callCount, 95, "Most calls should be made")
	assert.GreaterOrEqual(t, successCount, 95, "Most calls should succeed")
}

func TestCircuitBreakerConcurrentFailures(t *testing.T) {
	cb := NewCircuitBreaker("test-breaker")
	cb.failureThreshold = 10

	var wg sync.WaitGroup

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.Call(func() error { return testError })
		}()
	}

	wg.Wait()

	cb.mu.RLock()
	assert.GreaterOrEqual(t, cb.failureCount, 10, "At least threshold failures should be recorded")
	cb.mu.RUnlock()
}

func TestCircuitBreakerMultipleSuccesses(t *testing.T) {
	cb := NewCircuitBreaker("test-breaker")

	for i := 0; i < 10; i++ {
		err := cb.Call(func() error { return nil })
		assert.NoError(t, err)
	}

	cb.mu.RLock()
	assert.Equal(t, 0, cb.failureCount, "Failure count should remain 0")
	assert.Equal(t, StateClosed, cb.state, "State should remain closed")
	cb.mu.RUnlock()
}

func TestCircuitBreakerMixedSuccessAndFailure(t *testing.T) {
	cb := NewCircuitBreaker("test-breaker")
	cb.failureThreshold = 5

	// 4 successes
	for i := 0; i < 4; i++ {
		err := cb.Call(func() error { return nil })
		assert.NoError(t, err)
	}

	assert.Equal(t, StateClosed, cb.GetState())

	// 5 failures - should open
	for i := 0; i < 5; i++ {
		err := cb.Call(func() error { return testError })
		assert.Error(t, err)
	}

	assert.Equal(t, StateOpen, cb.GetState())

	// Try to call when open
	err := cb.Call(func() error { return nil })
	assert.Error(t, err)
	assert.Equal(t, ErrCircuitOpen, err)
}

func TestCircuitBreakerCheckState(t *testing.T) {
	cb := NewCircuitBreaker("test-breaker")

	// Set state to open with future retry time
	cb.mu.Lock()
	cb.state = StateOpen
	cb.nextRetryTime = time.Now().Add(1 * time.Hour)
	cb.mu.Unlock()

	cb.CheckState()
	assert.Equal(t, StateOpen, cb.GetState(), "State should remain open if retry time not reached")

	// Set retry time in the past
	cb.mu.Lock()
	cb.nextRetryTime = time.Now().Add(-1 * time.Second)
	cb.mu.Unlock()

	cb.CheckState()
	assert.Equal(t, StateHalfOpen, cb.GetState(), "State should transition to half-open when retry time reached")
}

func TestCircuitBreakerFailureTracking(t *testing.T) {
	cb := NewCircuitBreaker("test-breaker")

	cb.mu.Lock()
	cb.failureThreshold = 3
	cb.mu.Unlock()

	// First failure
	cb.Call(func() error { return testError })
	cb.mu.RLock()
	assert.Equal(t, 1, cb.failureCount, "Failure count should be 1")
	cb.mu.RUnlock()

	// Second failure
	cb.Call(func() error { return testError })
	cb.mu.RLock()
	assert.Equal(t, 2, cb.failureCount, "Failure count should be 2")
	cb.mu.RUnlock()

	// Third failure - should open
	cb.Call(func() error { return testError })
	cb.mu.RLock()
	assert.Equal(t, 3, cb.failureCount, "Failure count should be 3")
	cb.mu.RUnlock()

	assert.Equal(t, StateOpen, cb.GetState())
}

func TestCircuitBreakerSuccessTracking(t *testing.T) {
	cb := NewCircuitBreaker("test-breaker")
	cb.successThreshold = 5

	// Start in half-open state
	cb.mu.Lock()
	cb.state = StateHalfOpen
	cb.mu.Unlock()

	// Success 1
	cb.Call(func() error { return nil })
	cb.mu.RLock()
	assert.Equal(t, 1, cb.successCount, "Success count should be 1")
	cb.mu.RUnlock()

	// Success 2
	cb.Call(func() error { return nil })
	cb.mu.RLock()
	assert.Equal(t, 2, cb.successCount, "Success count should be 2")
	cb.mu.RUnlock()

	// Success 3
	cb.Call(func() error { return nil })
	cb.mu.RLock()
	assert.Equal(t, 3, cb.successCount, "Success count should be 3")
	cb.mu.RUnlock()

	// Success 4
	cb.Call(func() error { return nil })
	cb.mu.RLock()
	assert.Equal(t, 4, cb.successCount, "Success count should be 4")
	cb.mu.RUnlock()

	// Success 5 - should close
	cb.Call(func() error { return nil })
	cb.mu.RLock()
	assert.Equal(t, 0, cb.successCount, "Success count should be reset after closing")
	cb.mu.RUnlock()

	assert.Equal(t, StateClosed, cb.GetState())
}

func TestCircuitBreakerRecovery(t *testing.T) {
	cb := NewCircuitBreaker("test-breaker")
	cb.failureThreshold = 3
	cb.recoveryTimeout = 1 * time.Second

	// Force open
	for i := 0; i < 3; i++ {
		cb.Call(func() error { return testError })
	}
	assert.Equal(t, StateOpen, cb.GetState())

	// Wait for recovery time
	time.Sleep(1 * time.Second + 100*time.Millisecond)

	// Check state should transition to half-open
	cb.CheckState()
	assert.Equal(t, StateHalfOpen, cb.GetState())

	// Successful call in half-open
	err := cb.Call(func() error { return nil })
	assert.NoError(t, err)
}

func TestCircuitBreakerStateConstants(t *testing.T) {
	assert.Equal(t, 0, int(StateClosed), "StateClosed should be 0")
	assert.Equal(t, 1, int(StateOpen), "StateOpen should be 1")
	assert.Equal(t, 2, int(StateHalfOpen), "StateHalfOpen should be 2")
}

func TestErrCircuitOpen(t *testing.T) {
	assert.NotNil(t, ErrCircuitOpen, "ErrCircuitOpen should not be nil")
	assert.Equal(t, "circuit breaker is open", ErrCircuitOpen.Error(), "Error message should match")
}

func TestCircuitBreakerName(t *testing.T) {
	cb := NewCircuitBreaker("my-breaker")

	// Access private name through state checks
	assert.Equal(t, StateClosed, cb.GetState())

	// Name is private but we can verify circuit breaker works
	cb.Call(func() error { return nil })
	assert.Equal(t, StateClosed, cb.GetState())
}

func TestCircuitBreakerDefaultConfigurations(t *testing.T) {
	cb := NewCircuitBreaker("test")

	cb.mu.RLock()
	assert.Equal(t, 5, cb.failureThreshold, "Default failure threshold should be 5")
	assert.Equal(t, 30*time.Second, cb.recoveryTimeout, "Default recovery timeout should be 30s")
	assert.Equal(t, 3, cb.successThreshold, "Default success threshold should be 3")
	assert.Equal(t, 10*time.Second, cb.monitoringPeriod, "Default monitoring period should be 10s")
	cb.mu.RUnlock()
}

func TestCircuitBreakerMultipleInstances(t *testing.T) {
	cb1 := NewCircuitBreaker("breaker-1")
	cb2 := NewCircuitBreaker("breaker-2")

	// Open first circuit breaker
	for i := 0; i < 5; i++ {
		cb1.Call(func() error { return testError })
	}

	// Second should still be closed
	assert.Equal(t, StateOpen, cb1.GetState(), "First breaker should be open")
	assert.Equal(t, StateClosed, cb2.GetState(), "Second breaker should still be closed")

	// Reset first
	cb1.Reset()
	assert.Equal(t, StateClosed, cb1.GetState(), "First breaker should be closed after reset")

	// Second should still be closed
	assert.Equal(t, StateClosed, cb2.GetState(), "Second breaker should still be closed")
}

func TestCircuitBreakerRaceCondition(t *testing.T) {
	cb := NewCircuitBreaker("race-test")
	cb.failureThreshold = 100

	var wg sync.WaitGroup

	// Concurrent successes
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.Call(func() error { return nil })
		}()
	}

	// Concurrent failures
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.Call(func() error { return testError })
		}()
	}

	wg.Wait()

	// Should not panic and state should be consistent
	cb.mu.RLock()
	state := cb.state
	failureCount := cb.failureCount
	cb.mu.RUnlock()

	assert.True(t, state == StateClosed || state == StateOpen, "State should be valid")
	assert.GreaterOrEqual(t, failureCount, 0, "Failure count should be valid")
}

func TestCircuitBreakerPanicsInFunction(t *testing.T) {
	cb := NewCircuitBreaker("panic-test")

	// This will panic
	assert.Panics(t, func() {
		cb.Call(func() error {
			panic("test panic")
		})
	}, "Should panic when function panics")

	// Circuit breaker should still work after panic
	err := cb.Call(func() error { return nil })
	assert.NoError(t, err)
}

func TestCircuitBreakerResetWhileOpen(t *testing.T) {
	cb := NewCircuitBreaker("reset-test")
	cb.failureThreshold = 3

	// Open circuit
	for i := 0; i < 3; i++ {
		cb.Call(func() error { return testError })
	}
	assert.Equal(t, StateOpen, cb.GetState())

	// Reset
	cb.Reset()
	assert.Equal(t, StateClosed, cb.GetState())

	// Should work normally after reset
	err := cb.Call(func() error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.GetState())
}

func TestCircuitBreakerFullCycle(t *testing.T) {
	cb := NewCircuitBreaker("cycle-test")
	cb.failureThreshold = 3
	cb.successThreshold = 2
	cb.recoveryTimeout = 500 * time.Millisecond

	// 1. Start closed
	assert.Equal(t, StateClosed, cb.GetState())

	// 2. Success in closed state
	err := cb.Call(func() error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.GetState())

	// 3. Failures lead to open
	for i := 0; i < 3; i++ {
		cb.Call(func() error { return testError })
	}
	assert.Equal(t, StateOpen, cb.GetState())

	// 4. Calls blocked when open
	err = cb.Call(func() error { return nil })
	assert.Error(t, err)
	assert.Equal(t, ErrCircuitOpen, err)

	// 5. Wait for recovery
	time.Sleep(600 * time.Millisecond)
	cb.CheckState()
	assert.Equal(t, StateHalfOpen, cb.GetState())

	// 6. Success in half-open (1 of 2)
	err = cb.Call(func() error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, StateHalfOpen, cb.GetState())

	// 7. Success in half-open (2 of 2) - should close
	err = cb.Call(func() error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.GetState())

	// 8. Normal operation restored
	err = cb.Call(func() error { return nil })
	assert.NoError(t, err)
}

func TestCircuitBreakerPartialFailure(t *testing.T) {
	cb := NewCircuitBreaker("partial-failure")
	cb.failureThreshold = 10

	// Mix of successes and failures
	for i := 0; i < 20; i++ {
		var err error
		if i%3 == 0 {
			err = cb.Call(func() error { return testError })
		} else {
			err = cb.Call(func() error { return nil })
		}
		if i < 10 {
			// Should succeed in first 10 calls
			if i%3 == 0 {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		}
	}

	cb.mu.RLock()
	assert.Equal(t, 0, cb.failureCount, "Successes reset failures")
	state := cb.state
	cb.mu.RUnlock()

	// Should still be closed (less than threshold)
	assert.Equal(t, StateClosed, state, "Should still be closed with partial failures")
}
