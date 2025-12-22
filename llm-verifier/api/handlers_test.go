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

func TestCreateModelHandler(t *testing.T) {
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

	t.Run("Success", func(t *testing.T) {
		modelData := map[string]interface{}{
			"provider_id":              provider.ID,
			"model_id":                 "test-model-123",
			"name":                     "Test Model",
			"description":              "A test model for unit tests",
			"version":                  "1.0",
			"architecture":             "transformer",
			"parameter_count":          7000000000,
			"context_window_tokens":    8192,
			"max_output_tokens":        4096,
			"is_multimodal":            false,
			"supports_vision":          false,
			"supports_audio":           false,
			"supports_video":           false,
			"supports_reasoning":       true,
			"open_source":              true,
			"deprecated":               false,
			"tags":                     []string{"test", "unit"},
			"language_support":         []string{"en", "es"},
			"use_case":                 "testing",
			"verification_status":      "pending",
			"overall_score":            85.5,
			"code_capability_score":    80.0,
			"responsiveness_score":     90.0,
			"reliability_score":        95.0,
			"feature_richness_score":   75.0,
			"value_proposition_score":  88.0,
		}
		body, _ := json.Marshal(modelData)
		req, _ := http.NewRequest("POST", "/api/v1/models", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.POST("/api/v1/models", server.createModel)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("ValidationError", func(t *testing.T) {
		// Invalid values for required fields
		modelData := map[string]interface{}{
			"provider_id":              0, // invalid
			"model_id":                 "", // empty
			"name":                     "", // empty
			"overall_score":            -5, // negative
		}
		body, _ := json.Marshal(modelData)
		req, _ := http.NewRequest("POST", "/api/v1/models", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.POST("/api/v1/models", server.createModel)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGetModelHandler(t *testing.T) {
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
		ModelID:     "test-model-456",
		Name:        "Test Model",
		Description: "Test model for unit tests",
	}
	err = server.database.CreateModel(&model)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/models/%d", model.ID), nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.GET("/api/v1/models/:id", server.getModel)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// Optionally parse response and verify fields
		var response database.Model
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, model.ID, response.ID)
		assert.Equal(t, model.Name, response.Name)
	})

	t.Run("NotFound", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/models/99999", nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.GET("/api/v1/models/:id", server.getModel)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUpdateModelHandler(t *testing.T) {
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
		ModelID:     "test-model-789",
		Name:        "Original Model",
		Description: "Original description",
	}
	err = server.database.CreateModel(&model)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		newName := "Updated Model Name"
		newDesc := "Updated description"
		modelData := map[string]interface{}{
			"name":                  newName,
			"description":           newDesc,
			"version":               "2.0",
			"parameter_count":       0,
			"context_window_tokens": 0,
			"max_output_tokens":     0,
		}
		body, _ := json.Marshal(modelData)
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/models/%d", model.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.PUT("/api/v1/models/:id", server.updateModel)

		router.ServeHTTP(w, req)

		t.Logf("Response status: %d, body: %s", w.Code, w.Body.String())
		assert.Equal(t, http.StatusOK, w.Code)
		// Verify update
		var response struct {
			Data database.Model `json:"data"`
		}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, newName, response.Data.Name)
		assert.Equal(t, newDesc, response.Data.Description)
		assert.Equal(t, "2.0", response.Data.Version)
	})

	t.Run("ValidationError", func(t *testing.T) {
		modelData := map[string]interface{}{
			"overall_score": -10, // invalid negative
		}
		body, _ := json.Marshal(modelData)
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/models/%d", model.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.PUT("/api/v1/models/:id", server.updateModel)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("NotFound", func(t *testing.T) {
		modelData := map[string]interface{}{
			"name": "New Name",
		}
		body, _ := json.Marshal(modelData)
		req, _ := http.NewRequest("PUT", "/api/v1/models/99999", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.PUT("/api/v1/models/:id", server.updateModel)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestDeleteModelHandler(t *testing.T) {
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
		ModelID:     "test-model-delete",
		Name:        "Model to delete",
		Description: "Will be deleted",
	}
	err = server.database.CreateModel(&model)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/models/%d", model.ID), nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.DELETE("/api/v1/models/:id", server.deleteModel)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		// Verify deletion by attempting to fetch
		req2, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/models/%d", model.ID), nil)
		w2 := httptest.NewRecorder()
		router.GET("/api/v1/models/:id", server.getModel)
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusNotFound, w2.Code)
	})

	t.Run("NotFound", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/v1/models/99999", nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.DELETE("/api/v1/models/:id", server.deleteModel)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

func TestUpdateProviderHandler(t *testing.T) {
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
		Name:        "Original Provider",
		Endpoint:    "https://api.example.com",
		Description: "Original description",
		IsActive:    true,
	}
	err = server.database.CreateProvider(&provider)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		newName := "Updated Provider Name"
		newDesc := "Updated description"
		providerData := map[string]interface{}{
			"name":        newName,
			"description": newDesc,
			"website":     "https://updated.example.com",
		}
		body, _ := json.Marshal(providerData)
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/providers/%d", provider.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.PUT("/api/v1/providers/:id", server.updateProvider)

		router.ServeHTTP(w, req)

		t.Logf("Response status: %d, body: %s", w.Code, w.Body.String())
		assert.Equal(t, http.StatusOK, w.Code)
		// Verify update
		var response database.Provider
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, newName, response.Name)
		assert.Equal(t, newDesc, response.Description)
		assert.Equal(t, "https://updated.example.com", response.Website)
	})

	t.Run("ValidationError", func(t *testing.T) {
		providerData := map[string]interface{}{
			"reliability_score": -10, // invalid negative
		}
		body, _ := json.Marshal(providerData)
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/providers/%d", provider.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.PUT("/api/v1/providers/:id", server.updateProvider)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("NotFound", func(t *testing.T) {
		providerData := map[string]interface{}{
			"name": "New Name",
		}
		body, _ := json.Marshal(providerData)
		req, _ := http.NewRequest("PUT", "/api/v1/providers/99999", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.PUT("/api/v1/providers/:id", server.updateProvider)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestDeleteProviderHandler(t *testing.T) {
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
		Name:        "Provider to delete",
		Endpoint:    "https://api.example.com",
		Description: "Will be deleted",
		IsActive:    true,
	}
	err = server.database.CreateProvider(&provider)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/providers/%d", provider.ID), nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.DELETE("/api/v1/providers/:id", server.deleteProvider)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		// Verify deletion by attempting to fetch
		req2, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/providers/%d", provider.ID), nil)
		w2 := httptest.NewRecorder()
		router.GET("/api/v1/providers/:id", server.getProvider)
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusNotFound, w2.Code)
	})

	t.Run("NotFound", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/v1/providers/99999", nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.DELETE("/api/v1/providers/:id", server.deleteProvider)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

func TestCreateVerificationResultHandler(t *testing.T) {
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

	// Create provider and model
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
	model := database.Model{
		ProviderID:  provider.ID,
		ModelID:     "test-model-verif",
		Name:        "Test Model",
		Description: "Test model for verification",
	}
	err = server.database.CreateModel(&model)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		startedAt := time.Now().Add(-time.Hour)
		completedAt := time.Now()
		verificationData := map[string]interface{}{
			"model_id":           model.ID,
			"verification_type":  "basic",
			"started_at":         startedAt.Format(time.RFC3339),
			"completed_at":       completedAt.Format(time.RFC3339),
			"status":             "completed",
			"supports_tool_use":  true,
			"supports_reasoning": false,
		}
		body, _ := json.Marshal(verificationData)
		req, _ := http.NewRequest("POST", "/api/v1/verification-results", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.POST("/api/v1/verification-results", server.createVerificationResult)

		router.ServeHTTP(w, req)

		t.Logf("Response status: %d, body: %s", w.Code, w.Body.String())
		assert.Equal(t, http.StatusCreated, w.Code)
		// Optionally parse response and verify fields
	})

	t.Run("ValidationError", func(t *testing.T) {
		verificationData := map[string]interface{}{
			"model_id":           0, // invalid
			"verification_type":  "invalid-type", // invalid
			"started_at":         "invalid-date",
			"status":             "invalid-status",
		}
		body, _ := json.Marshal(verificationData)
		req, _ := http.NewRequest("POST", "/api/v1/verification-results", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.POST("/api/v1/verification-results", server.createVerificationResult)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGetVerificationResultHandler(t *testing.T) {
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
		ModelID:     "test-model-verif-get",
		Name:        "Test Model",
		Description: "Test model for verification get",
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

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/verification-results/%d", verificationResult.ID), nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.GET("/api/v1/verification-results/:id", server.getVerificationResult)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response database.VerificationResult
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, verificationResult.ID, response.ID)
		assert.Equal(t, verificationResult.ModelID, response.ModelID)
	})

	t.Run("NotFound", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/verification-results/99999", nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.GET("/api/v1/verification-results/:id", server.getVerificationResult)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
func TestUpdateVerificationResultHandler(t *testing.T) {
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
		ModelID:     "test-model-verif-update",
		Name:        "Test Model",
		Description: "Test model for verification update",
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

	t.Run("Success", func(t *testing.T) {
		newStatus := "failed"
		updateData := map[string]interface{}{
			"status": newStatus,
		}
		body, _ := json.Marshal(updateData)
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/verification-results/%d", verificationResult.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.PUT("/api/v1/verification-results/:id", server.updateVerificationResult)

		router.ServeHTTP(w, req)

		t.Logf("Response status: %d, body: %s", w.Code, w.Body.String())
		assert.Equal(t, http.StatusOK, w.Code)
		var response database.VerificationResult
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, newStatus, response.Status)
	})

	t.Run("ValidationError", func(t *testing.T) {
		updateData := map[string]interface{}{
			"debugging_accuracy": 150, // invalid > 100
		}
		body, _ := json.Marshal(updateData)
		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/verification-results/%d", verificationResult.ID), bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.PUT("/api/v1/verification-results/:id", server.updateVerificationResult)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("NotFound", func(t *testing.T) {
		updateData := map[string]interface{}{
			"status": "failed",
		}
		body, _ := json.Marshal(updateData)
		req, _ := http.NewRequest("PUT", "/api/v1/verification-results/99999", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.PUT("/api/v1/verification-results/:id", server.updateVerificationResult)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
func TestDeleteVerificationResultHandler(t *testing.T) {
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
		ModelID:     "test-model-verif-delete",
		Name:        "Test Model",
		Description: "Test model for verification delete",
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

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/verification-results/%d", verificationResult.ID), nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.DELETE("/api/v1/verification-results/:id", server.deleteVerificationResult)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		// Verify deletion by attempting to fetch
		req2, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/verification-results/%d", verificationResult.ID), nil)
		w2 := httptest.NewRecorder()
		router.GET("/api/v1/verification-results/:id", server.getVerificationResult)
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusNotFound, w2.Code)
	})

	t.Run("NotFound", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/v1/verification-results/99999", nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.DELETE("/api/v1/verification-results/:id", server.deleteVerificationResult)

		router.ServeHTTP(w, req)

		// Delete endpoint returns 204 even if not found? Let's see actual behavior
		// We'll just assert no panic
		assert.NotEqual(t, http.StatusInternalServerError, w.Code)
	})
}

func TestGetPricingHandler(t *testing.T) {
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
		ModelID:     "test-model-pricing",
		Name:        "Test Model",
		Description: "Test model for pricing tests",
	}
	err = server.database.CreateModel(&model)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Create a pricing entry
	effectiveFrom := time.Now()
	pricing := database.Pricing{
		ModelID:              model.ID,
		InputTokenCost:       0.001,
		OutputTokenCost:      0.002,
		CachedInputTokenCost: 0.0005,
		StorageCost:          0.01,
		RequestCost:          0.1,
		Currency:             "USD",
		PricingModel:         "per_token",
		EffectiveFrom:        &effectiveFrom,
	}
	err = server.database.CreatePricing(&pricing)
	if err != nil {
		t.Fatalf("Failed to create pricing: %v", err)
	}

	req, _ := http.NewRequest("GET", "/api/v1/pricing?limit=10&offset=0", nil)
	w := httptest.NewRecorder()

	router := gin.Default()
	router.GET("/api/v1/pricing", server.getPricing)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCreatePricingHandler(t *testing.T) {
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
		ModelID:     "test-model-pricing-create",
		Name:        "Test Model",
		Description: "Test model for pricing create tests",
	}
	err = server.database.CreateModel(&model)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		effectiveFrom := time.Now()
		requestBody := map[string]interface{}{
			"model_id":               model.ID,
			"input_token_cost":       0.001,
			"output_token_cost":      0.002,
			"cached_input_token_cost": 0.0005,
			"storage_cost":           0.01,
			"request_cost":           0.1,
			"currency":               "USD",
			"pricing_model":          "per_token",
			"effective_from":         effectiveFrom.Format(time.RFC3339),
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/v1/pricing", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.POST("/api/v1/pricing", server.createPricing)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response["id"])
		assert.Equal(t, model.ID, int64(response["model_id"].(float64)))
	})

	t.Run("ValidationError", func(t *testing.T) {
		// Missing required fields
		requestBody := map[string]interface{}{
			"model_id": model.ID,
			// missing input_token_cost
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/v1/pricing", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.POST("/api/v1/pricing", server.createPricing)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestGetPricingByIDHandler(t *testing.T) {
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
		ModelID:     "test-model-pricing-getbyid",
		Name:        "Test Model",
		Description: "Test model for pricing get by ID tests",
	}
	err = server.database.CreateModel(&model)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Create a pricing entry
	effectiveFrom := time.Now()
	pricing := database.Pricing{
		ModelID:              model.ID,
		InputTokenCost:       0.001,
		OutputTokenCost:      0.002,
		CachedInputTokenCost: 0.0005,
		StorageCost:          0.01,
		RequestCost:          0.1,
		Currency:             "USD",
		PricingModel:         "per_token",
		EffectiveFrom:        &effectiveFrom,
	}
	err = server.database.CreatePricing(&pricing)
	if err != nil {
		t.Fatalf("Failed to create pricing: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/pricing/%d", pricing.ID), nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.GET("/api/v1/pricing/:id", server.getPricingByID)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, pricing.ID, int64(response["id"].(float64)))
		assert.Equal(t, model.ID, int64(response["model_id"].(float64)))
	})

	t.Run("NotFound", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/pricing/99999", nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.GET("/api/v1/pricing/:id", server.getPricingByID)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestUpdatePricingHandler(t *testing.T) {
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
		ModelID:     "test-model-pricing-update",
		Name:        "Test Model",
		Description: "Test model for pricing update tests",
	}
	err = server.database.CreateModel(&model)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Create a pricing entry
	effectiveFrom := time.Now()
	pricing := database.Pricing{
		ModelID:              model.ID,
		InputTokenCost:       0.001,
		OutputTokenCost:      0.002,
		CachedInputTokenCost: 0.0005,
		StorageCost:          0.01,
		RequestCost:          0.1,
		Currency:             "USD",
		PricingModel:         "per_token",
		EffectiveFrom:        &effectiveFrom,
	}
	err = server.database.CreatePricing(&pricing)
	if err != nil {
		t.Fatalf("Failed to create pricing: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"input_token_cost": 0.002, // Update price
			"currency":        "EUR",
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/pricing/%d", pricing.ID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.PUT("/api/v1/pricing/:id", server.updatePricing)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, pricing.ID, int64(response["id"].(float64)))
		assert.Equal(t, "EUR", response["currency"])
	})

	t.Run("ValidationError", func(t *testing.T) {
		// Invalid currency length
		requestBody := map[string]interface{}{
			"currency": "US", // 2 chars, should be 3
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/pricing/%d", pricing.ID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.PUT("/api/v1/pricing/:id", server.updatePricing)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("NotFound", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"input_token_cost": 0.002,
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("PUT", "/api/v1/pricing/99999", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.PUT("/api/v1/pricing/:id", server.updatePricing)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestDeletePricingHandler(t *testing.T) {
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
		ModelID:     "test-model-pricing-delete",
		Name:        "Model for pricing deletion",
		Description: "Model to test pricing deletion",
	}
	err = server.database.CreateModel(&model)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Create pricing
	effectiveFrom := time.Now()
	pricing := database.Pricing{
		ModelID:              model.ID,
		InputTokenCost:       0.001,
		OutputTokenCost:      0.002,
		CachedInputTokenCost: 0.0005,
		StorageCost:          0.01,
		RequestCost:          0.1,
		Currency:             "USD",
		PricingModel:         "per_token",
		EffectiveFrom:        &effectiveFrom,
	}
	err = server.database.CreatePricing(&pricing)
	if err != nil {
		t.Fatalf("Failed to create pricing: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/pricing/%d", pricing.ID), nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.DELETE("/api/v1/pricing/:id", server.deletePricing)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		// Verify deletion by attempting to fetch
		req2, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/pricing/%d", pricing.ID), nil)
		w2 := httptest.NewRecorder()
		router.GET("/api/v1/pricing/:id", server.getPricingByID)
		router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusNotFound, w2.Code)
	})

	t.Run("NotFound", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/v1/pricing/99999", nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.DELETE("/api/v1/pricing/:id", server.deletePricing)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

func TestGetLimitsHandler(t *testing.T) {
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
		ModelID:     "test-model-limits",
		Name:        "Test Model",
		Description: "Test model for limits tests",
	}
	err = server.database.CreateModel(&model)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Create a limit entry
	resetTime := time.Now().Add(time.Hour)
	limit := database.Limit{
		ModelID:      model.ID,
		LimitType:    "requests_per_minute",
		LimitValue:   100,
		CurrentUsage: 0,
		ResetPeriod:  "minute",
		ResetTime:    &resetTime,
		IsHardLimit:  true,
	}
	err = server.database.CreateLimit(&limit)
	if err != nil {
		t.Fatalf("Failed to create limit: %v", err)
	}

	req, _ := http.NewRequest("GET", "/api/v1/limits?limit=10&offset=0", nil)
	w := httptest.NewRecorder()

	router := gin.Default()
	router.GET("/api/v1/limits", server.getLimits)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetLimitByIDHandler(t *testing.T) {
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
		ModelID:     "test-model-limits-by-id",
		Name:        "Test Model",
		Description: "Test model for limits by ID tests",
	}
	err = server.database.CreateModel(&model)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Create a limit entry
	resetTime := time.Now().Add(time.Hour)
	limit := database.Limit{
		ModelID:      model.ID,
		LimitType:    "requests_per_minute",
		LimitValue:   100,
		CurrentUsage: 0,
		ResetPeriod:  "minute",
		ResetTime:    &resetTime,
		IsHardLimit:  true,
	}
	err = server.database.CreateLimit(&limit)
	if err != nil {
		t.Fatalf("Failed to create limit: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/limits/%d", limit.ID), nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.GET("/api/v1/limits/:id", server.getLimitByID)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("NotFound", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/limits/99999", nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.GET("/api/v1/limits/:id", server.getLimitByID)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestCreateLimitHandler(t *testing.T) {
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
		ModelID:     "test-model-limits-create",
		Name:        "Test Model",
		Description: "Test model for limits create tests",
	}
	err = server.database.CreateModel(&model)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		resetTime := time.Now().Add(time.Hour)
		requestBody := map[string]interface{}{
			"model_id":      model.ID,
			"limit_type":    "requests_per_minute",
			"limit_value":   100,
			"current_usage": 0,
			"reset_period":  "minute",
			"reset_time":    resetTime.Format(time.RFC3339),
			"is_hard_limit": true,
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/v1/limits", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.POST("/api/v1/limits", server.createLimit)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response["id"])
		assert.Equal(t, model.ID, int64(response["model_id"].(float64)))
	})

	t.Run("ValidationError", func(t *testing.T) {
		// Missing required fields
		requestBody := map[string]interface{}{
			"model_id": model.ID,
			// missing limit_type, limit_value, reset_period
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/v1/limits", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.POST("/api/v1/limits", server.createLimit)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUpdateLimitHandler(t *testing.T) {
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
		ModelID:     "test-model-limits-update",
		Name:        "Test Model",
		Description: "Test model for limits update tests",
	}
	err = server.database.CreateModel(&model)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Create a limit entry
	resetTime := time.Now().Add(time.Hour)
	limit := database.Limit{
		ModelID:      model.ID,
		LimitType:    "requests_per_minute",
		LimitValue:   100,
		CurrentUsage: 0,
		ResetPeriod:  "minute",
		ResetTime:    &resetTime,
		IsHardLimit:  true,
	}
	err = server.database.CreateLimit(&limit)
	if err != nil {
		t.Fatalf("Failed to create limit: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"limit_value":   200, // Update limit value
			"reset_period":  "hour",
			"is_hard_limit": false,
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/limits/%d", limit.ID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.PUT("/api/v1/limits/:id", server.updateLimit)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, limit.ID, int64(response["id"].(float64)))
		assert.Equal(t, 200.0, response["limit_value"])
		assert.Equal(t, "hour", response["reset_period"])
		assert.Equal(t, false, response["is_hard_limit"])
	})

	t.Run("ValidationError", func(t *testing.T) {
		// Invalid limit value (negative)
		requestBody := map[string]interface{}{
			"limit_value": -10,
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/limits/%d", limit.ID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.PUT("/api/v1/limits/:id", server.updateLimit)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("NotFound", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"limit_value": 300,
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("PUT", "/api/v1/limits/99999", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.PUT("/api/v1/limits/:id", server.updateLimit)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestDeleteLimitHandler(t *testing.T) {
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
		ModelID:     "test-model-limits-delete",
		Name:        "Test Model",
		Description: "Test model for limits delete tests",
	}
	err = server.database.CreateModel(&model)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Create a limit entry
	resetTime := time.Now().Add(time.Hour)
	limit := database.Limit{
		ModelID:      model.ID,
		LimitType:    "requests_per_minute",
		LimitValue:   100,
		CurrentUsage: 0,
		ResetPeriod:  "minute",
		ResetTime:    &resetTime,
		IsHardLimit:  true,
	}
	err = server.database.CreateLimit(&limit)
	if err != nil {
		t.Fatalf("Failed to create limit: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/limits/%d", limit.ID), nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.DELETE("/api/v1/limits/:id", server.deleteLimit)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("NotFound", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/v1/limits/99999", nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.DELETE("/api/v1/limits/:id", server.deleteLimit)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

func TestGetEventsHandler(t *testing.T) {
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

	// Create an event entry
	event := database.Event{
		EventType: "system_error",
		Severity:  "info",
		Title:     "Test Event",
		Message:   "This is a test event for unit tests",
	}
	err = server.database.CreateEvent(&event)
	if err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	req, _ := http.NewRequest("GET", "/api/v1/events?limit=10&offset=0", nil)
	w := httptest.NewRecorder()

	router := gin.Default()
	router.GET("/api/v1/events", server.getEvents)

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetEventByIDHandler(t *testing.T) {
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

	// Create an event entry
	event := database.Event{
		EventType: "system_error",
		Severity:  "info",
		Title:     "Test Event",
		Message:   "This is a test event for unit tests",
	}
	err = server.database.CreateEvent(&event)
	if err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fmt.Sprintf("/api/v1/events/%d", event.ID), nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.GET("/api/v1/events/:id", server.getEventByID)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("NotFound", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/events/99999", nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.GET("/api/v1/events/:id", server.getEventByID)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestCreateEventHandler(t *testing.T) {
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
		requestBody := map[string]interface{}{
			"event_type":  "system_error",
			"severity":    "info",
			"title":       "Test Event Created",
			"message":     "This is a test event created via API",
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/v1/events", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.POST("/api/v1/events", server.createEvent)

		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Logf("Response body: %s", w.Body.String())
		}
		assert.Equal(t, http.StatusCreated, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotNil(t, response["id"])
		assert.Equal(t, "system_error", response["event_type"])
		assert.Equal(t, "info", response["severity"])
	})

	t.Run("ValidationError", func(t *testing.T) {
		// Missing required fields
		requestBody := map[string]interface{}{
			"event_type": "system_error",
			// missing severity, title, message
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("POST", "/api/v1/events", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.POST("/api/v1/events", server.createEvent)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestUpdateEventHandler(t *testing.T) {
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

	// Create an event entry
	event := database.Event{
		EventType: "system_error",
		Severity:  "info",
		Title:     "Test Event",
		Message:   "This is a test event for unit tests",
	}
	err = server.database.CreateEvent(&event)
	if err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"severity": "warning",
			"title":    "Updated Event Title",
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/events/%d", event.ID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.PUT("/api/v1/events/:id", server.updateEvent)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, event.ID, int64(response["id"].(float64)))
		assert.Equal(t, "warning", response["severity"])
		assert.Equal(t, "Updated Event Title", response["title"])
	})

	t.Run("ValidationError", func(t *testing.T) {
		// Invalid severity (not in enum)
		requestBody := map[string]interface{}{
			"severity": "invalid_severity",
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("PUT", fmt.Sprintf("/api/v1/events/%d", event.ID), bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.PUT("/api/v1/events/:id", server.updateEvent)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("NotFound", func(t *testing.T) {
		requestBody := map[string]interface{}{
			"title": "Updated Title",
		}
		jsonBody, _ := json.Marshal(requestBody)

		req, _ := http.NewRequest("PUT", "/api/v1/events/99999", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router := gin.Default()
		router.PUT("/api/v1/events/:id", server.updateEvent)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestDeleteEventHandler(t *testing.T) {
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

	// Create an event entry
	event := database.Event{
		EventType: "system_error",
		Severity:  "info",
		Title:     "Test Event",
		Message:   "This is a test event for unit tests",
	}
	err = server.database.CreateEvent(&event)
	if err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	t.Run("Success", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("/api/v1/events/%d", event.ID), nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.DELETE("/api/v1/events/:id", server.deleteEvent)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("NotFound", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/api/v1/events/99999", nil)
		w := httptest.NewRecorder()

		router := gin.Default()
		router.DELETE("/api/v1/events/:id", server.deleteEvent)

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}
