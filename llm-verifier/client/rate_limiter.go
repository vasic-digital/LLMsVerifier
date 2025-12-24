package client

import (
	"fmt"
	"sync"
	"time"
)

// RateLimiter handles rate limiting for clients
type RateLimiter struct {
	clientLimits map[int64]*ClientLimits
	mutex        sync.RWMutex
}

// ClientLimits stores rate limiting data for a client
type ClientLimits struct {
	ClientID          int64
	RequestsPerMinute int
	RequestsPerHour   int
	RequestsPerDay    int
	BurstLimit        int

	// Current counters
	MinuteCount int
	HourCount   int
	DayCount    int
	LastMinute  time.Time
	LastHour    time.Time
	LastDay     time.Time

	// Backoff state
	InBackoff    bool
	BackoffUntil time.Time
	BackoffCount int
}

// RateLimitResult represents the result of a rate limit check
type RateLimitResult struct {
	Allowed      bool          `json:"allowed"`
	Remaining    int           `json:"remaining"`
	ResetTime    time.Time     `json:"reset_time"`
	RetryAfter   time.Duration `json:"retry_after,omitempty"`
	InBackoff    bool          `json:"in_backoff"`
	BackoffCount int           `json:"backoff_count"`
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		clientLimits: make(map[int64]*ClientLimits),
	}

	// Start cleanup goroutine
	go rl.cleanupExpiredLimits()

	return rl
}

// SetClientLimits sets rate limits for a client
func (rl *RateLimiter) SetClientLimits(clientID int64, rpm, rph, rpd, burst int) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	rl.clientLimits[clientID] = &ClientLimits{
		ClientID:          clientID,
		RequestsPerMinute: rpm,
		RequestsPerHour:   rph,
		RequestsPerDay:    rpd,
		BurstLimit:        burst,
		MinuteCount:       0,
		HourCount:         0,
		DayCount:          0,
		LastMinute:        time.Now(),
		LastHour:          time.Now(),
		LastDay:           time.Now(),
		InBackoff:         false,
		BackoffCount:      0,
	}
}

// CheckRateLimit checks if a request should be allowed
func (rl *RateLimiter) CheckRateLimit(clientID int64) *RateLimitResult {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	limits, exists := rl.clientLimits[clientID]
	if !exists {
		// No limits set, allow request
		return &RateLimitResult{
			Allowed:   true,
			Remaining: -1, // Unlimited
		}
	}

	now := time.Now()

	// Check if in backoff period
	if limits.InBackoff && now.Before(limits.BackoffUntil) {
		return &RateLimitResult{
			Allowed:      false,
			Remaining:    0,
			ResetTime:    limits.BackoffUntil,
			RetryAfter:   limits.BackoffUntil.Sub(now),
			InBackoff:    true,
			BackoffCount: limits.BackoffCount,
		}
	}

	// Reset counters if time windows have passed
	if now.Sub(limits.LastMinute) >= time.Minute {
		limits.MinuteCount = 0
		limits.LastMinute = now
	}

	if now.Sub(limits.LastHour) >= time.Hour {
		limits.HourCount = 0
		limits.LastHour = now
	}

	if now.Sub(limits.LastDay) >= 24*time.Hour {
		limits.DayCount = 0
		limits.LastDay = now
	}

	// Check limits (most restrictive applies)
	minRemaining := -1
	var resetTime time.Time

	// Check minute limit
	if limits.RequestsPerMinute > 0 {
		remaining := limits.RequestsPerMinute - limits.MinuteCount
		if remaining <= 0 {
			return &RateLimitResult{
				Allowed:    false,
				Remaining:  0,
				ResetTime:  limits.LastMinute.Add(time.Minute),
				RetryAfter: limits.LastMinute.Add(time.Minute).Sub(now),
			}
		}
		if minRemaining == -1 || remaining < minRemaining {
			minRemaining = remaining
			resetTime = limits.LastMinute.Add(time.Minute)
		}
	}

	// Check hour limit
	if limits.RequestsPerHour > 0 {
		remaining := limits.RequestsPerHour - limits.HourCount
		if remaining <= 0 {
			return &RateLimitResult{
				Allowed:    false,
				Remaining:  0,
				ResetTime:  limits.LastHour.Add(time.Hour),
				RetryAfter: limits.LastHour.Add(time.Hour).Sub(now),
			}
		}
		if minRemaining == -1 || remaining < minRemaining {
			minRemaining = remaining
			resetTime = limits.LastHour.Add(time.Hour)
		}
	}

	// Check day limit
	if limits.RequestsPerDay > 0 {
		remaining := limits.RequestsPerDay - limits.DayCount
		if remaining <= 0 {
			return &RateLimitResult{
				Allowed:    false,
				Remaining:  0,
				ResetTime:  limits.LastDay.Add(24 * time.Hour),
				RetryAfter: limits.LastDay.Add(24 * time.Hour).Sub(now),
			}
		}
		if minRemaining == -1 || remaining < minRemaining {
			minRemaining = remaining
			resetTime = limits.LastDay.Add(24 * time.Hour)
		}
	}

	// Allow request
	return &RateLimitResult{
		Allowed:   true,
		Remaining: minRemaining - 1, // Will be decremented when RecordRequest is called
		ResetTime: resetTime,
		InBackoff: false,
	}
}

// RecordRequest records a successful request (increments counters)
func (rl *RateLimiter) RecordRequest(clientID int64) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	limits, exists := rl.clientLimits[clientID]
	if !exists {
		return
	}

	now := time.Now()

	// Reset backoff if expired
	if limits.InBackoff && now.After(limits.BackoffUntil) {
		limits.InBackoff = false
		limits.BackoffCount = 0
	}

	// Increment counters
	limits.MinuteCount++
	limits.HourCount++
	limits.DayCount++
}

// RecordRateLimitViolation records a rate limit violation and applies backoff
func (rl *RateLimiter) RecordRateLimitViolation(clientID int64, backoffStrategy string) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	limits, exists := rl.clientLimits[clientID]
	if !exists {
		return
	}

	limits.BackoffCount++

	// Calculate backoff duration
	var backoffDuration time.Duration
	switch backoffStrategy {
	case "exponential":
		// 2^backoff_count seconds, max 300 seconds (5 minutes)
		backoffDuration = time.Duration(min(1<<uint(limits.BackoffCount-1), 300)) * time.Second
	case "linear":
		// 30 seconds * backoff_count, max 5 minutes
		backoffDuration = time.Duration(min(30*limits.BackoffCount, 300)) * time.Second
	default: // "fixed"
		backoffDuration = 30 * time.Second
	}

	limits.InBackoff = true
	limits.BackoffUntil = time.Now().Add(backoffDuration)
}

// ResetClientLimits resets all counters for a client
func (rl *RateLimiter) ResetClientLimits(clientID int64) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	if limits, exists := rl.clientLimits[clientID]; exists {
		now := time.Now()
		limits.MinuteCount = 0
		limits.HourCount = 0
		limits.DayCount = 0
		limits.LastMinute = now
		limits.LastHour = now
		limits.LastDay = now
		limits.InBackoff = false
		limits.BackoffCount = 0
	}
}

// GetClientLimits returns current limits and counters for a client
func (rl *RateLimiter) GetClientLimits(clientID int64) (*ClientLimits, error) {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	limits, exists := rl.clientLimits[clientID]
	if !exists {
		return nil, fmt.Errorf("no rate limits configured for client %d", clientID)
	}

	// Return a copy to prevent external modification
	limitsCopy := *limits
	return &limitsCopy, nil
}

// GetAllClientLimits returns limits for all clients
func (rl *RateLimiter) GetAllClientLimits() map[int64]*ClientLimits {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	limits := make(map[int64]*ClientLimits)
	for clientID, clientLimits := range rl.clientLimits {
		// Return copies
		limitsCopy := *clientLimits
		limits[clientID] = &limitsCopy
	}

	return limits
}

// RemoveClientLimits removes rate limits for a client
func (rl *RateLimiter) RemoveClientLimits(clientID int64) {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	delete(rl.clientLimits, clientID)
}

// GetStats returns rate limiting statistics
func (rl *RateLimiter) GetStats() map[string]interface{} {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()

	totalClients := len(rl.clientLimits)
	inBackoff := 0
	totalRequests := 0

	for _, limits := range rl.clientLimits {
		if limits.InBackoff {
			inBackoff++
		}
		totalRequests += limits.DayCount
	}

	return map[string]interface{}{
		"total_clients":        totalClients,
		"clients_in_backoff":   inBackoff,
		"total_requests_today": totalRequests,
		"timestamp":            time.Now(),
	}
}

// cleanupExpiredLimits periodically cleans up expired limits and backoffs
func (rl *RateLimiter) cleanupExpiredLimits() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mutex.Lock()
			now := time.Now()

			for clientID, limits := range rl.clientLimits {
				// Reset expired backoffs
				if limits.InBackoff && now.After(limits.BackoffUntil) {
					limits.InBackoff = false
					limits.BackoffCount = 0
				}

				// Reset old counters (keep last 24 hours)
				if now.Sub(limits.LastDay) > 24*time.Hour {
					delete(rl.clientLimits, clientID)
				}
			}

			rl.mutex.Unlock()
		}
	}
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// SlidingWindowLimiter implements sliding window rate limiting
type SlidingWindowLimiter struct {
	windowSize  time.Duration
	maxRequests int
	requests    []time.Time
	mutex       sync.Mutex
}

// NewSlidingWindowLimiter creates a new sliding window rate limiter
func NewSlidingWindowLimiter(windowSize time.Duration, maxRequests int) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		windowSize:  windowSize,
		maxRequests: maxRequests,
		requests:    make([]time.Time, 0),
	}
}

// Allow checks if a request should be allowed and records it
func (swl *SlidingWindowLimiter) Allow() bool {
	swl.mutex.Lock()
	defer swl.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-swl.windowSize)

	// Remove old requests outside the window
	validRequests := make([]time.Time, 0)
	for _, req := range swl.requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}
	swl.requests = validRequests

	// Check if we can allow this request
	if len(swl.requests) >= swl.maxRequests {
		return false
	}

	// Add this request
	swl.requests = append(swl.requests, now)
	return true
}

// GetRemainingRequests returns how many more requests are allowed in the current window
func (swl *SlidingWindowLimiter) GetRemainingRequests() int {
	swl.mutex.Lock()
	defer swl.mutex.Unlock()

	now := time.Now()
	cutoff := now.Add(-swl.windowSize)

	// Count valid requests
	validCount := 0
	for _, req := range swl.requests {
		if req.After(cutoff) {
			validCount++
		}
	}

	return swl.maxRequests - validCount
}

// TokenBucketLimiter implements token bucket rate limiting
type TokenBucketLimiter struct {
	capacity   int64         // Maximum tokens
	refillRate time.Duration // Time between token refills
	tokens     int64         // Current tokens
	lastRefill time.Time
	mutex      sync.Mutex
}

// NewTokenBucketLimiter creates a new token bucket rate limiter
func NewTokenBucketLimiter(capacity int64, refillRate time.Duration) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		capacity:   capacity,
		refillRate: refillRate,
		tokens:     capacity,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request should be allowed and consumes a token
func (tbl *TokenBucketLimiter) Allow() bool {
	tbl.mutex.Lock()
	defer tbl.mutex.Unlock()

	tbl.refillTokens()

	if tbl.tokens > 0 {
		tbl.tokens--
		return true
	}

	return false
}

// refillTokens adds tokens based on elapsed time
func (tbl *TokenBucketLimiter) refillTokens() {
	now := time.Now()
	elapsed := now.Sub(tbl.lastRefill)

	// Calculate how many tokens to add
	tokensToAdd := int64(elapsed / tbl.refillRate)
	if tokensToAdd > 0 {
		tbl.tokens = min64(tbl.tokens+tokensToAdd, tbl.capacity)
		tbl.lastRefill = now
	}
}

// GetAvailableTokens returns the current number of available tokens
func (tbl *TokenBucketLimiter) GetAvailableTokens() int64 {
	tbl.mutex.Lock()
	defer tbl.mutex.Unlock()

	tbl.refillTokens()
	return tbl.tokens
}

// min64 returns the minimum of two int64 values
func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}
