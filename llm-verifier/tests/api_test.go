package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"

	"llm-verifier/api"
	"llm-verifier/config"
)

func setupTestServer(t *testing.T) *gin.Engine {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create test config
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: "./test.db",
		},
		API: config.APIConfig{
			Port:       "8080",
			JWTSecret:  "test-secret",
			RateLimit:  100,
			EnableCORS: true,
		},
	}

	// Create server
	server, err := api.NewServer(cfg)
	assert.NoError(t, err)

	return server.Router()
}

func TestHealthCheck(t *testing.T) {
	router := setupTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.Contains(t, response, "timestamp")
	assert.Equal(t, "1.0.0", response["version"])
}

func TestLogin(t *testing.T) {
	router := setupTestServer(t)

	t.Run("successful login", func(t *testing.T) {
		loginData := map[string]string{
			"username": "admin",
			"password": "password",
		}
		jsonData, _ := json.Marshal(loginData)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.Contains(t, response, "token")
		assert.Contains(t, response, "expires_in")
		assert.Contains(t, response, "user")
	})

	t.Run("invalid credentials", func(t *testing.T) {
		loginData := map[string]string{
			"username": "",
			"password": "",
		}
		jsonData, _ := json.Marshal(loginData)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.Equal(t, "Invalid credentials", response["error"])
	})
}

func TestCORSHeaders(t *testing.T) {
	router := setupTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/api/v1/models", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")
}

func TestRateLimiting(t *testing.T) {
	router := setupTestServer(t)

	// Make multiple requests to test rate limiting
	for i := 0; i < 105; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/health", nil)
		router.ServeHTTP(w, req)

		if i < 100 {
			assert.Equal(t, http.StatusOK, w.Code)
		} else {
			// Should be rate limited after 100 requests
			assert.Equal(t, http.StatusTooManyRequests, w.Code)
		}
	}
}

func TestJWTAuthentication(t *testing.T) {
	router := setupTestServer(t)

	t.Run("missing authorization header", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/models", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.Equal(t, "Authorization header required", response["error"])
	})

	t.Run("invalid authorization header format", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/models", nil)
		req.Header.Set("Authorization", "InvalidFormat")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.Equal(t, "Invalid authorization header format", response["error"])
	})

	t.Run("invalid token", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/models", nil)
		req.Header.Set("Authorization", "Bearer invalid.token.here")
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)

		assert.Equal(t, "Invalid token", response["error"])
	})
}

func TestGetModels(t *testing.T) {
	router := setupTestServer(t)

	// First login to get a token
	loginData := map[string]string{
		"username": "admin",
		"password": "password",
	}
	jsonData, _ := json.Marshal(loginData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	var loginResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResponse)
	token := loginResponse["token"].(string)

	// Now test the protected endpoint
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/models", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Contains(t, response, "models")
	assert.Contains(t, response, "pagination")
}

func TestGetConfig(t *testing.T) {
	router := setupTestServer(t)

	// First login to get a token
	loginData := map[string]string{
		"username": "admin",
		"password": "password",
	}
	jsonData, _ := json.Marshal(loginData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	var loginResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResponse)
	token := loginResponse["token"].(string)

	// Test the config endpoint
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/config", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Contains(t, response, "concurrency")
	assert.Contains(t, response, "timeout")
	assert.Contains(t, response, "api")
}

func TestInvalidJSON(t *testing.T) {
	router := setupTestServer(t)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "Invalid request format", response["error"])
}

func TestInvalidParameters(t *testing.T) {
	router := setupTestServer(t)

	// Test invalid limit parameter
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/models?limit=invalid", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "Invalid limit parameter", response["error"])
}
