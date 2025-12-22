package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"llm-verifier/config"
	"llm-verifier/database"
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

func TestGetModelsHandler(t *testing.T) {
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

	// Create a provider first
	provider := database.Provider{
		Name:        "Test Provider",
		Endpoint:    "https://api.example.com",
		Description: "Test provider for unit tests",
		Status:      "active",
	}
	err = server.database.CreateProvider(&provider)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Create a model
	model := database.Model{
		ProviderID:  provider.ID,
		ModelID:     "test-model",
		Name:        "Test Model",
		Description: "Test model for unit tests",
	}
	err = server.database.CreateModel(&model)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	req, _ := http.NewRequest("GET", "/api/v1/models?limit=10&offset=0", nil)
	w := httptest.NewRecorder()

	router := gin.Default()
	router.GET("/api/v1/models", server.getModels)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
}
