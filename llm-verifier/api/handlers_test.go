package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
		IsActive:    true,
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

func TestGetProvidersHandler(t *testing.T) {
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

	// Create a provider
	provider := database.Provider{
		Name:        "Test Provider",
		Endpoint:    "https://api.example.com",
		Description: "Test provider for unit tests",
		IsActive:    true,
	}
	err = server.database.CreateProvider(&provider)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	req, _ := http.NewRequest("GET", "/api/v1/providers?limit=10&offset=0", nil)
	w := httptest.NewRecorder()

	router := gin.Default()
	router.GET("/api/v1/providers", server.getProviders)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetProviderHandler(t *testing.T) {
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

	// Create a provider
	provider := database.Provider{
		Name:        "Test Provider",
		Endpoint:    "https://api.example.com",
		Description: "Test provider for unit tests",
		IsActive:    true,
	}
	err = server.database.CreateProvider(&provider)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/providers/%d", provider.ID), nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.GET("/api/v1/providers/:id", server.getProvider)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("NotFound", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/providers/99999", nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.GET("/api/v1/providers/:id", server.getProvider)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestCreateProviderHandler(t *testing.T) {
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

	t.Run("Success", func(t *testing.T) {
		providerData := map[string]interface{}{
			"name":                    "New Provider",
			"endpoint":                "https://api.newprovider.com",
			"api_key_encrypted":       "encrypted-key-1234567890",
			"description":             "A new provider",
			"website":                 "https://newprovider.com",
			"support_email":           "support@newprovider.com",
			"documentation_url":       "https://docs.newprovider.com",
			"is_active":               true,
			"reliability_score":       95.5,
			"average_response_time_ms": 150,
		}
		body, _ := json.Marshal(providerData)
		req, _ := http.NewRequest("POST", "/api/v1/providers", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.POST("/api/v1/providers", server.createProvider)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("ValidationError", func(t *testing.T) {
		// Invalid values for required fields
		providerData := map[string]interface{}{
			"name":              "", // empty name
			"endpoint":          "invalid-url",
			"api_key_encrypted": "short", // less than min 10
		}
		body, _ := json.Marshal(providerData)
		req, _ := http.NewRequest("POST", "/api/v1/providers", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.POST("/api/v1/providers", server.createProvider)

		router.ServeHTTP(w, req)

		t.Logf("Response status: %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestValidation(t *testing.T) {
	t.Run("Required fields", func(t *testing.T) {
		req := CreateProviderRequest{
			Name:              "",
			Endpoint:          "invalid-url",
			APIKeyEncrypted:   "short",
		}
		err := ValidateRequest(req)
		assert.Error(t, err)
	})

	t.Run("URL validation", func(t *testing.T) {
		req := CreateProviderRequest{
			Name:              "Test",
			Endpoint:          "invalid-url",
			APIKeyEncrypted:   "longenough123",
		}
		err := ValidateRequest(req)
		assert.Error(t, err)
	})

	t.Run("Min length", func(t *testing.T) {
		req := CreateProviderRequest{
			Name:              "Test",
			Endpoint:          "https://example.com",
			APIKeyEncrypted:   "short",
		}
		err := ValidateRequest(req)
		assert.Error(t, err)
	})

	t.Run("Valid request", func(t *testing.T) {
		req := CreateProviderRequest{
			Name:              "Test Provider",
			Endpoint:          "https://api.example.com",
			APIKeyEncrypted:   "encrypted-key-1234567890",
			Description:       "Description",
			Website:           "https://example.com",
			SupportEmail:      "support@example.com",
			DocumentationURL:  "https://docs.example.com",
			IsActive:          true,
			ReliabilityScore:  95.5,
			AverageResponseTimeMs: 150,
		}
		err := ValidateRequest(req)
		assert.NoError(t, err)
	})
}

func TestGetVerificationResultsHandler(t *testing.T) {
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

	// Create a provider
	provider := database.Provider{
		Name:        "Test Provider",
		Endpoint:    "https://api.example.com",
		Description: "Test provider for unit tests",
		IsActive:    true,
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

	// Create a verification result
	completedAt := time.Now()
	verificationResult := database.VerificationResult{
		ModelID:          model.ID,
		VerificationType: "basic",
		StartedAt:        time.Now().Add(-time.Hour),
		CompletedAt:      &completedAt,
		Status:           "completed",
	}
	err = server.database.CreateVerificationResult(&verificationResult)
	if err != nil {
		t.Fatalf("Failed to create verification result: %v", err)
	}

	req, _ := http.NewRequest("GET", "/api/v1/verification-results?limit=10&offset=0", nil)
	w := httptest.NewRecorder()

	router := gin.Default()
	router.GET("/api/v1/verification-results", server.getVerificationResults)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
