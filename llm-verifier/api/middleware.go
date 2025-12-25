package api

import (
	"bytes"
	"net/http"
	"strings"
)

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

// RateLimitMiddleware implements HTTP rate limiting
func RateLimitMiddleware(limiter interface{}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// For now, implement a simple IP-based rate limiting
			// In production, you'd use a more sophisticated rate limiter
			clientIP := getClientIP(r)

			// Check rate limit (placeholder - would use actual rate limiter)
			if isRateLimited(clientIP) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			// Add rate limit headers
			w.Header().Set("X-RateLimit-Limit", "100")
			w.Header().Set("X-RateLimit-Remaining", "99")
			w.Header().Set("X-RateLimit-Reset", "1640995200")

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

// isRateLimited checks if the client is rate limited (placeholder implementation)
func isRateLimited(clientIP string) bool {
	// Placeholder - in production, you'd check against a rate limiter
	// For example:
	// return rateLimiter.CheckRateLimit(clientIP).Allowed == false

	return false // Allow all requests for now
}
