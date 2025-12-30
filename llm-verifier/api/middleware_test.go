package api

import (
	"bytes"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	// Create middleware
	middleware := SecurityHeadersMiddleware()

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap handler with middleware
	handler := middleware(testHandler)

	// Create request without TLS
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify response
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify security headers are set
	assert.Equal(t, "1; mode=block", rec.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Equal(t, "strict-origin-when-cross-origin", rec.Header().Get("Referrer-Policy"))
	assert.Equal(t, "geolocation=(), microphone=(), camera=()", rec.Header().Get("Permissions-Policy"))
	assert.Equal(t, "none", rec.Header().Get("X-Permitted-Cross-Domain-Policies"))

	// Without TLS, HSTS should not be set
	assert.Empty(t, rec.Header().Get("Strict-Transport-Security"))
}

func TestSecurityHeadersMiddleware_WithTLS(t *testing.T) {
	middleware := SecurityHeadersMiddleware()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(testHandler)

	// Create HTTPS request (with TLS)
	req := httptest.NewRequest("GET", "https://example.com/test", nil)
	req.TLS = &tls.ConnectionState{} // Simulate TLS connection
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// With TLS, HSTS should be set
	assert.Equal(t, "max-age=31536000; includeSubDomains", rec.Header().Get("Strict-Transport-Security"))
}

func TestContentSecurityPolicyMiddleware(t *testing.T) {
	middleware := ContentSecurityPolicyMiddleware()

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := middleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify CSP header is set
	csp := rec.Header().Get("Content-Security-Policy")
	assert.NotEmpty(t, csp)
	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "script-src")
	assert.Contains(t, csp, "style-src")
	assert.Contains(t, csp, "frame-ancestors 'none'")
}

func TestRateLimitMiddleware(t *testing.T) {
	middleware := RateLimitMiddleware(nil)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	handler := middleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Verify request is allowed (rate limit is placeholder, always allows)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Verify rate limit headers are set
	assert.Equal(t, "100", rec.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "99", rec.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, rec.Header().Get("X-RateLimit-Reset"))
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		xff        string
		xri        string
		remoteAddr string
		expected   string
	}{
		{
			name:       "X-Forwarded-For single IP",
			xff:        "10.0.0.1",
			remoteAddr: "192.168.1.1:12345",
			expected:   "10.0.0.1",
		},
		{
			name:       "X-Forwarded-For multiple IPs",
			xff:        "10.0.0.1, 172.16.0.1, 192.168.1.1",
			remoteAddr: "127.0.0.1:12345",
			expected:   "10.0.0.1",
		},
		{
			name:       "X-Real-IP",
			xri:        "10.0.0.2",
			remoteAddr: "192.168.1.1:12345",
			expected:   "10.0.0.2",
		},
		{
			name:       "RemoteAddr with port",
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:       "RemoteAddr without port",
			remoteAddr: "192.168.1.1",
			expected:   "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}

			result := getClientIP(req)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsRateLimited(t *testing.T) {
	// Current implementation always returns false
	result := isRateLimited("192.168.1.1")
	assert.False(t, result)

	result = isRateLimited("10.0.0.1")
	assert.False(t, result)
}

func TestResponseWriterWrapper_Write(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapper := &responseWriterWrapper{
		ResponseWriter: rec,
		body:           &bytes.Buffer{},
	}

	n, err := wrapper.Write([]byte("test data"))
	assert.NoError(t, err)
	assert.Equal(t, 9, n)

	// Check that data was written to both buffer and response writer
	assert.Equal(t, "test data", wrapper.body.String())
	assert.Equal(t, "test data", rec.Body.String())
}

func TestSanitizeJSONResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{"string", "test", "test"},
		{"map", map[string]string{"key": "value"}, map[string]string{"key": "value"}},
		{"nil", nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeJSONResponse(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOutputSanitizationMiddleware(t *testing.T) {
	middleware := OutputSanitizationMiddleware()

	t.Run("JSON response", func(t *testing.T) {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"key": "value"}`))
		})

		handler := middleware(testHandler)
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Contains(t, rec.Body.String(), "key")
	})

	t.Run("HTML response", func(t *testing.T) {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<p>Hello</p>`))
		})

		handler := middleware(testHandler)
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Contains(t, rec.Body.String(), "Hello")
	})

	t.Run("Other content type", func(t *testing.T) {
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(`Plain text content`))
		})

		handler := middleware(testHandler)
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)
		assert.Contains(t, rec.Body.String(), "Plain text")
	})
}
