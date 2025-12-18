package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"llm-verifier/config"
)

func TestHealthCheckHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup minimal config
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: ":memory:",
		},
		API: config.APIConfig{
			JWTSecret: "test-secret",
		},
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Shutdown()

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router := gin.Default()
	router.GET("/health", server.healthCheck)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRefreshTokenHandler_BasicValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: ":memory:",
		},
		API: config.APIConfig{
			JWTSecret: "test-secret",
		},
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Shutdown()

	tests := []struct {
		name          string
		authHeader    string
		statusCode    int
		expectedError string
	}{
		{
			name:          "Missing auth header",
			authHeader:    "",
			statusCode:    http.StatusUnauthorized,
			expectedError: "Authorization header required",
		},
		{
			name:          "Invalid auth format",
			authHeader:    "InvalidFormat token",
			statusCode:    http.StatusUnauthorized,
			expectedError: "Invalid authorization header format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("POST", "/auth/refresh", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()

			router := gin.Default()
			router.POST("/auth/refresh", server.refreshToken)

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.statusCode, w.Code)
		})
	}
}
