package api

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RateLimiter provides thread-safe rate limiting using token bucket algorithm
type RateLimiter struct {
	mu           sync.RWMutex
	clients      map[string]*clientRateLimit
	defaultLimit int           // requests per window
	windowSize   time.Duration // sliding window size
	cleanupTick  time.Duration // cleanup interval for expired entries
}

// clientRateLimit tracks rate limit state for a single client
type clientRateLimit struct {
	requests    []time.Time // timestamps of requests within window
	mu          sync.Mutex
	lastCleanup time.Time
}

// NewRateLimiter creates a new rate limiter with specified limits
func NewRateLimiter(requestsPerMinute int, windowSize time.Duration) *RateLimiter {
	rl := &RateLimiter{
		clients:      make(map[string]*clientRateLimit),
		defaultLimit: requestsPerMinute,
		windowSize:   windowSize,
		cleanupTick:  time.Minute * 5,
	}

	// Start background cleanup goroutine
	go rl.cleanupLoop()

	return rl
}

// Allow checks if the client is allowed to make a request
func (rl *RateLimiter) Allow(clientIP string) (allowed bool, remaining int, resetTime time.Time) {
	rl.mu.Lock()
	client, exists := rl.clients[clientIP]
	if !exists {
		client = &clientRateLimit{
			requests:    make([]time.Time, 0, rl.defaultLimit),
			lastCleanup: time.Now(),
		}
		rl.clients[clientIP] = client
	}
	rl.mu.Unlock()

	client.mu.Lock()
	defer client.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.windowSize)

	// Remove expired requests (outside the window)
	validRequests := make([]time.Time, 0, len(client.requests))
	for _, t := range client.requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}
	client.requests = validRequests

	// Calculate remaining and reset time
	remaining = rl.defaultLimit - len(client.requests)
	if len(client.requests) > 0 {
		resetTime = client.requests[0].Add(rl.windowSize)
	} else {
		resetTime = now.Add(rl.windowSize)
	}

	// Check if limit exceeded
	if len(client.requests) >= rl.defaultLimit {
		return false, 0, resetTime
	}

	// Record this request
	client.requests = append(client.requests, now)
	remaining = rl.defaultLimit - len(client.requests)

	return true, remaining, resetTime
}

// GetStatus returns current rate limit status without consuming a request
func (rl *RateLimiter) GetStatus(clientIP string) (remaining int, resetTime time.Time) {
	rl.mu.RLock()
	client, exists := rl.clients[clientIP]
	rl.mu.RUnlock()

	if !exists {
		return rl.defaultLimit, time.Now().Add(rl.windowSize)
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.windowSize)

	count := 0
	var firstRequest time.Time
	for _, t := range client.requests {
		if t.After(windowStart) {
			if count == 0 {
				firstRequest = t
			}
			count++
		}
	}

	remaining = rl.defaultLimit - count
	if count > 0 {
		resetTime = firstRequest.Add(rl.windowSize)
	} else {
		resetTime = now.Add(rl.windowSize)
	}

	return remaining, resetTime
}

// cleanupLoop periodically removes stale entries
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanupTick)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

// cleanup removes clients with no recent requests
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.windowSize)

	for ip, client := range rl.clients {
		client.mu.Lock()
		// Remove if all requests are expired
		hasValid := false
		for _, t := range client.requests {
			if t.After(windowStart) {
				hasValid = true
				break
			}
		}
		if !hasValid && now.Sub(client.lastCleanup) > rl.windowSize*2 {
			delete(rl.clients, ip)
		}
		client.mu.Unlock()
	}
}

// Global rate limiter instance
var globalRateLimiter *RateLimiter
var rateLimiterOnce sync.Once

// GetGlobalRateLimiter returns the global rate limiter, creating it if necessary
func GetGlobalRateLimiter() *RateLimiter {
	rateLimiterOnce.Do(func() {
		// Default: 100 requests per minute
		globalRateLimiter = NewRateLimiter(100, time.Minute)
	})
	return globalRateLimiter
}

// SetGlobalRateLimiter allows configuring a custom rate limiter
func SetGlobalRateLimiter(rl *RateLimiter) {
	globalRateLimiter = rl
}

// OutputSanitizationMiddleware sanitizes API responses to prevent XSS
func OutputSanitizationMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create a response writer wrapper to capture the response
			wrapper := &responseWriterWrapper{
				ResponseWriter: w,
				body:           &bytes.Buffer{},
			}

			// Call the next handler
			next.ServeHTTP(wrapper, r)

			// Get the response content type
			contentType := wrapper.Header().Get("Content-Type")

			// Sanitize based on content type
			if strings.Contains(contentType, "application/json") {
				sanitizedBody := SanitizeJSONOutput(wrapper.body.String())
				wrapper.ResponseWriter.Write([]byte(sanitizedBody.(string)))
			} else if strings.Contains(contentType, "text/html") {
				sanitizedBody := SanitizeHTMLResponse(wrapper.body.String())
				wrapper.ResponseWriter.Write([]byte(sanitizedBody))
			} else {
				// For other content types, apply basic output sanitization
				sanitizedBody := SanitizeOutput(wrapper.body.String())
				wrapper.ResponseWriter.Write([]byte(sanitizedBody))
			}
		})
	}
}

// responseWriterWrapper captures the response body
type responseWriterWrapper struct {
	http.ResponseWriter
	body *bytes.Buffer
}

func (rw *responseWriterWrapper) Write(data []byte) (int, error) {
	// Write to both the buffer and the original response writer
	rw.body.Write(data)
	return rw.ResponseWriter.Write(data)
}

// SanitizeJSONResponse sanitizes JSON responses
func SanitizeJSONResponse(data interface{}) interface{} {
	// For now, just return the data as-is
	// In production, you'd want to sanitize string fields in the JSON
	return data
}

// SecurityHeadersMiddleware adds comprehensive security headers
func SecurityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// XSS protection
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			// Prevent MIME type sniffing
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// Prevent clickjacking
			w.Header().Set("X-Frame-Options", "DENY")

			// Referrer policy
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

			// Permissions policy (formerly Feature-Policy)
			w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

			// HSTS (HTTP Strict Transport Security) - only for HTTPS
			if r.TLS != nil {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			// Remove server header to avoid information disclosure
			w.Header().Del("Server")

			// Add security-focused headers
			w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")

			next.ServeHTTP(w, r)
		})
	}
}

// ContentSecurityPolicyMiddleware adds CSP headers
func ContentSecurityPolicyMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Comprehensive CSP policy
			csp := "default-src 'self'; " +
				"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
				"style-src 'self' 'unsafe-inline'; " +
				"img-src 'self' data: https: blob:; " +
				"font-src 'self' https: data:; " +
				"connect-src 'self' https: wss:; " +
				"media-src 'self' https:; " +
				"object-src 'none'; " +
				"frame-src 'none'; " +
				"frame-ancestors 'none'; " +
				"base-uri 'self'; " +
				"form-action 'self'; " +
				"upgrade-insecure-requests;"

			w.Header().Set("Content-Security-Policy", csp)
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitMiddleware implements HTTP rate limiting using sliding window algorithm
func RateLimitMiddleware(limiter interface{}) func(http.Handler) http.Handler {
	// Use the global rate limiter or create one if custom limiter provided
	var rl *RateLimiter
	if customLimiter, ok := limiter.(*RateLimiter); ok && customLimiter != nil {
		rl = customLimiter
	} else {
		rl = GetGlobalRateLimiter()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			clientIP := getClientIP(r)

			// Check rate limit using the real rate limiter
			allowed, remaining, resetTime := rl.Allow(clientIP)

			// Add rate limit headers
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.defaultLimit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

			if !allowed {
				retryAfter := int(time.Until(resetTime).Seconds())
				if retryAfter < 1 {
					retryAfter = 1
				}
				w.Header().Set("Retry-After", fmt.Sprintf("%d", retryAfter))
				http.Error(w, "Rate limit exceeded. Please retry after the reset time.", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getClientIP extracts the real client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies/load balancers)
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if idx := strings.Index(xff, ","); idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr
	if idx := strings.LastIndex(r.RemoteAddr, ":"); idx > 0 {
		return r.RemoteAddr[:idx]
	}

	return r.RemoteAddr
}

// isRateLimited checks if the client is rate limited using the global rate limiter
func isRateLimited(clientIP string) bool {
	rl := GetGlobalRateLimiter()
	allowed, _, _ := rl.Allow(clientIP)
	return !allowed
}
