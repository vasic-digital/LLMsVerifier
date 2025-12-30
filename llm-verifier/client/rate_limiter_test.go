package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRateLimiter(t *testing.T) {
	rl := NewRateLimiter()
	require.NotNil(t, rl)
	assert.NotNil(t, rl.clientLimits)
}

func TestRateLimiter_SetClientLimits(t *testing.T) {
	rl := NewRateLimiter()

	rl.SetClientLimits(1, 60, 1000, 5000, 10)

	limits, err := rl.GetClientLimits(1)
	require.NoError(t, err)

	assert.Equal(t, int64(1), limits.ClientID)
	assert.Equal(t, 60, limits.RequestsPerMinute)
	assert.Equal(t, 1000, limits.RequestsPerHour)
	assert.Equal(t, 5000, limits.RequestsPerDay)
	assert.Equal(t, 10, limits.BurstLimit)
	assert.Equal(t, 0, limits.MinuteCount)
	assert.False(t, limits.InBackoff)
}

func TestRateLimiter_CheckRateLimit_NoLimits(t *testing.T) {
	rl := NewRateLimiter()

	result := rl.CheckRateLimit(999) // Client with no limits

	assert.True(t, result.Allowed)
	assert.Equal(t, -1, result.Remaining) // Unlimited
}

func TestRateLimiter_CheckRateLimit_WithLimits(t *testing.T) {
	rl := NewRateLimiter()
	rl.SetClientLimits(1, 5, 100, 1000, 10)

	// First request should be allowed
	result := rl.CheckRateLimit(1)
	assert.True(t, result.Allowed)
	assert.Equal(t, 4, result.Remaining) // 5 - 1 = 4
}

func TestRateLimiter_CheckRateLimit_MinuteLimit(t *testing.T) {
	rl := NewRateLimiter()
	rl.SetClientLimits(1, 3, 100, 1000, 10)

	// Make requests up to the limit
	for i := 0; i < 3; i++ {
		result := rl.CheckRateLimit(1)
		assert.True(t, result.Allowed)
		rl.RecordRequest(1)
	}

	// Next request should be denied
	result := rl.CheckRateLimit(1)
	assert.False(t, result.Allowed)
	assert.Equal(t, 0, result.Remaining)
}

func TestRateLimiter_RecordRequest(t *testing.T) {
	rl := NewRateLimiter()
	rl.SetClientLimits(1, 60, 1000, 5000, 10)

	// Record a request
	rl.RecordRequest(1)

	limits, _ := rl.GetClientLimits(1)
	assert.Equal(t, 1, limits.MinuteCount)
	assert.Equal(t, 1, limits.HourCount)
	assert.Equal(t, 1, limits.DayCount)
}

func TestRateLimiter_RecordRequest_NonExistingClient(t *testing.T) {
	rl := NewRateLimiter()

	// Should not panic
	rl.RecordRequest(999)
}

func TestRateLimiter_RecordRateLimitViolation_Exponential(t *testing.T) {
	rl := NewRateLimiter()
	rl.SetClientLimits(1, 60, 1000, 5000, 10)

	rl.RecordRateLimitViolation(1, "exponential")

	limits, _ := rl.GetClientLimits(1)
	assert.True(t, limits.InBackoff)
	assert.Equal(t, 1, limits.BackoffCount)
	assert.True(t, limits.BackoffUntil.After(time.Now()))
}

func TestRateLimiter_RecordRateLimitViolation_Linear(t *testing.T) {
	rl := NewRateLimiter()
	rl.SetClientLimits(1, 60, 1000, 5000, 10)

	rl.RecordRateLimitViolation(1, "linear")

	limits, _ := rl.GetClientLimits(1)
	assert.True(t, limits.InBackoff)
	assert.Equal(t, 1, limits.BackoffCount)
}

func TestRateLimiter_RecordRateLimitViolation_Fixed(t *testing.T) {
	rl := NewRateLimiter()
	rl.SetClientLimits(1, 60, 1000, 5000, 10)

	rl.RecordRateLimitViolation(1, "fixed")

	limits, _ := rl.GetClientLimits(1)
	assert.True(t, limits.InBackoff)
}

func TestRateLimiter_RecordRateLimitViolation_NonExistingClient(t *testing.T) {
	rl := NewRateLimiter()

	// Should not panic
	rl.RecordRateLimitViolation(999, "exponential")
}

func TestRateLimiter_CheckRateLimit_InBackoff(t *testing.T) {
	rl := NewRateLimiter()
	rl.SetClientLimits(1, 60, 1000, 5000, 10)

	// Put client in backoff
	rl.RecordRateLimitViolation(1, "fixed")

	// Request should be denied due to backoff
	result := rl.CheckRateLimit(1)
	assert.False(t, result.Allowed)
	assert.True(t, result.InBackoff)
	assert.True(t, result.RetryAfter > 0)
}

func TestRateLimiter_ResetClientLimits(t *testing.T) {
	rl := NewRateLimiter()
	rl.SetClientLimits(1, 60, 1000, 5000, 10)

	// Make some requests
	rl.RecordRequest(1)
	rl.RecordRequest(1)
	rl.RecordRateLimitViolation(1, "exponential")

	// Reset limits
	rl.ResetClientLimits(1)

	limits, _ := rl.GetClientLimits(1)
	assert.Equal(t, 0, limits.MinuteCount)
	assert.Equal(t, 0, limits.HourCount)
	assert.Equal(t, 0, limits.DayCount)
	assert.False(t, limits.InBackoff)
	assert.Equal(t, 0, limits.BackoffCount)
}

func TestRateLimiter_ResetClientLimits_NonExisting(t *testing.T) {
	rl := NewRateLimiter()

	// Should not panic
	rl.ResetClientLimits(999)
}

func TestRateLimiter_GetClientLimits(t *testing.T) {
	rl := NewRateLimiter()
	rl.SetClientLimits(1, 60, 1000, 5000, 10)

	t.Run("existing client", func(t *testing.T) {
		limits, err := rl.GetClientLimits(1)
		require.NoError(t, err)
		assert.Equal(t, int64(1), limits.ClientID)
	})

	t.Run("non-existing client", func(t *testing.T) {
		_, err := rl.GetClientLimits(999)
		assert.Error(t, err)
	})
}

func TestRateLimiter_GetAllClientLimits(t *testing.T) {
	rl := NewRateLimiter()
	rl.SetClientLimits(1, 60, 1000, 5000, 10)
	rl.SetClientLimits(2, 120, 2000, 10000, 20)
	rl.SetClientLimits(3, 30, 500, 2500, 5)

	allLimits := rl.GetAllClientLimits()
	assert.Equal(t, 3, len(allLimits))
	assert.Contains(t, allLimits, int64(1))
	assert.Contains(t, allLimits, int64(2))
	assert.Contains(t, allLimits, int64(3))
}

func TestRateLimiter_RemoveClientLimits(t *testing.T) {
	rl := NewRateLimiter()
	rl.SetClientLimits(1, 60, 1000, 5000, 10)

	// Verify exists
	_, err := rl.GetClientLimits(1)
	require.NoError(t, err)

	// Remove
	rl.RemoveClientLimits(1)

	// Verify removed
	_, err = rl.GetClientLimits(1)
	assert.Error(t, err)
}

func TestRateLimiter_GetStats(t *testing.T) {
	rl := NewRateLimiter()
	rl.SetClientLimits(1, 60, 1000, 5000, 10)
	rl.SetClientLimits(2, 120, 2000, 10000, 20)

	// Make some requests
	rl.RecordRequest(1)
	rl.RecordRequest(1)
	rl.RecordRequest(2)

	// Put one client in backoff
	rl.RecordRateLimitViolation(1, "exponential")

	stats := rl.GetStats()

	assert.Equal(t, 2, stats["total_clients"])
	assert.Equal(t, 1, stats["clients_in_backoff"])
	assert.Equal(t, 3, stats["total_requests_today"])
	assert.NotNil(t, stats["timestamp"])
}

// Test SlidingWindowLimiter

func TestNewSlidingWindowLimiter(t *testing.T) {
	swl := NewSlidingWindowLimiter(time.Minute, 10)
	require.NotNil(t, swl)
	assert.Equal(t, time.Minute, swl.windowSize)
	assert.Equal(t, 10, swl.maxRequests)
}

func TestSlidingWindowLimiter_Allow(t *testing.T) {
	swl := NewSlidingWindowLimiter(time.Minute, 3)

	// First 3 requests should be allowed
	assert.True(t, swl.Allow())
	assert.True(t, swl.Allow())
	assert.True(t, swl.Allow())

	// 4th request should be denied
	assert.False(t, swl.Allow())
}

func TestSlidingWindowLimiter_GetRemainingRequests(t *testing.T) {
	swl := NewSlidingWindowLimiter(time.Minute, 5)

	assert.Equal(t, 5, swl.GetRemainingRequests())

	swl.Allow()
	swl.Allow()

	assert.Equal(t, 3, swl.GetRemainingRequests())
}

// Test TokenBucketLimiter

func TestNewTokenBucketLimiter(t *testing.T) {
	tbl := NewTokenBucketLimiter(10, time.Second)
	require.NotNil(t, tbl)
	assert.Equal(t, int64(10), tbl.capacity)
	assert.Equal(t, int64(10), tbl.tokens)
}

func TestTokenBucketLimiter_Allow(t *testing.T) {
	tbl := NewTokenBucketLimiter(3, time.Second)

	// First 3 requests should be allowed
	assert.True(t, tbl.Allow())
	assert.True(t, tbl.Allow())
	assert.True(t, tbl.Allow())

	// 4th request should be denied (no tokens left)
	assert.False(t, tbl.Allow())
}

func TestTokenBucketLimiter_GetAvailableTokens(t *testing.T) {
	tbl := NewTokenBucketLimiter(5, time.Second)

	assert.Equal(t, int64(5), tbl.GetAvailableTokens())

	tbl.Allow()
	tbl.Allow()

	assert.Equal(t, int64(3), tbl.GetAvailableTokens())
}

func TestTokenBucketLimiter_Refill(t *testing.T) {
	tbl := NewTokenBucketLimiter(3, 10*time.Millisecond)

	// Use all tokens
	tbl.Allow()
	tbl.Allow()
	tbl.Allow()
	assert.Equal(t, int64(0), tbl.GetAvailableTokens())

	// Wait for refill
	time.Sleep(50 * time.Millisecond)

	// Should have some tokens now
	tokens := tbl.GetAvailableTokens()
	assert.True(t, tokens > 0)
}

// Test helper functions

func TestMin(t *testing.T) {
	assert.Equal(t, 5, min(5, 10))
	assert.Equal(t, 5, min(10, 5))
	assert.Equal(t, 5, min(5, 5))
}

func TestMin64(t *testing.T) {
	assert.Equal(t, int64(5), min64(5, 10))
	assert.Equal(t, int64(5), min64(10, 5))
	assert.Equal(t, int64(5), min64(5, 5))
}
