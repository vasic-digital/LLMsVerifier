package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"llm-verifier/api"
	"llm-verifier/config"
	"llm-verifier/database"
)

// TestIntegrationSuite runs comprehensive integration tests
func TestIntegrationSuite(t *testing.T) {
	// Test database connectivity
	t.Run("DatabaseConnection", TestDatabaseConnection)

	// Test authentication flow
	t.Run("Authentication", TestAuthenticationFlow)

	// Test API endpoints integration
	t.Run("APIEndpoints", TestAPIEndpointsIntegration)

	// Test error handling
	t.Run("ErrorHandling", TestErrorHandlingIntegration)

	// Test rate limiting
	t.Run("RateLimiting", TestRateLimitingIntegration)

	// Test health checks
	t.Run("HealthChecks", TestHealthChecksIntegration)
}

// TestDatabaseConnectionIntegration tests database connectivity and basic operations
func TestDatabaseConnectionIntegration(t *testing.T) {
	// Create test database
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Run migrations
	migrationManager := database.NewMigrationManager(db)
	migrationManager.SetupDefaultMigrations()

	if err := migrationManager.InitializeMigrationTable(); err != nil {
		t.Fatalf("Failed to initialize migration table: %v", err)
	}

	if err := migrationManager.MigrateUp(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Test basic operations
	testUser := &database.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "password123",
		FullName:     "Test User",
		Role:         "user",
		IsActive:     true,
	}

	err = db.CreateUser(testUser)
	assert.NoError(t, err)
	assert.NotZero(t, testUser.ID)

	// Retrieve user
	retrieved, err := db.GetUser(testUser.ID)
	assert.NoError(t, err)
	assert.Equal(t, testUser.Username, retrieved.Username)
	assert.Equal(t, testUser.Email, retrieved.Email)
}

// TestAuthenticationFlow tests the complete authentication process
func TestAuthenticationFlow(t *testing.T) {
	// Setup test database
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Run migrations
	migrationManager := database.NewMigrationManager(db)
	migrationManager.SetupDefaultMigrations()

	if err := migrationManager.InitializeMigrationTable(); err != nil {
		t.Fatalf("Failed to initialize migration table: %v", err)
	}

	if err := migrationManager.MigrateUp(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create test config
	cfg := &config.Config{
		Profile: "test",
		Global: config.GlobalConfig{
			LogLevel: "debug",
			LogFile:  "",
		},
		Database: config.DatabaseConfig{
			Path: ":memory:",
		},
		API: config.APIConfig{
			JWTSecret:  "test-secret-key",
			RateLimit:  100,
			BurstLimit: 200,
		},
	}

	// Create a proper server instance
	server, err := api.NewServer(cfg)
	require.NoError(t, err)
	require.NotNil(t, server)

	if err := migrationManager.MigrateUp(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create test user
	testUser := &database.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "password123",
		FullName:     "Test User",
		Role:         "user",
		IsActive:     true,
	}

	err = db.CreateUser(testUser)
	require.NoError(t, err)

	// Create test server
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: ":memory:",
		},
		API: config.APIConfig{
			JWTSecret:  "test-secret-key",
			RateLimit:  100,
			BurstLimit: 200,
		},
	}

	// Use the database from earlier (in-memory)
	server := &api.Server{
		Config:   cfg,
		Database: db,
	}

	// Test login
	loginJSON := `{"username": "testuser", "password": "password123"}`
	req := httptest.NewRequest("POST", "/auth/login", bytes.NewBuffer([]byte(loginJSON)))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var loginResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &loginResp)
	assert.NoError(t, err)
	assert.Contains(t, loginResp, "token")

	// Test protected endpoint with token
	if token, ok := loginResp["token"].(string); ok {
		protectedReq := httptest.NewRequest("GET", "/api/v1/users/me", nil)
		protectedReq.Header.Set("Authorization", "Bearer "+token)

		w2 := httptest.NewRecorder()
		server.ServeHTTP(w2, protectedReq)

		assert.Equal(t, http.StatusOK, w2.Code)
	}
}

// TestAPIEndpointsIntegration tests multiple API endpoints working together
func TestAPIEndpointsIntegration(t *testing.T) {
	// Setup test environment
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Run migrations
	migrationManager := database.NewMigrationManager(db)
	migrationManager.SetupDefaultMigrations()

	if err := migrationManager.InitializeMigrationTable(); err != nil {
		t.Fatalf("Failed to initialize migration table: %v", err)
	}

	if err := migrationManager.MigrateUp(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create admin user for testing
	adminUser := &database.User{
		Username:     "admin",
		Email:        "admin@example.com",
		PasswordHash: "admin123",
		FullName:     "Admin User",
		Role:         "admin",
		IsActive:     true,
	}

	err = db.CreateUser(adminUser)
	require.NoError(t, err)

	// Create test server
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: ":memory:",
		},
		API: config.APIConfig{
			JWTSecret:  "test-secret-key",
			RateLimit:  1000, // Higher limit for testing
			BurstLimit: 2000,
		},
	}

	server := &api.Server{
		Config:   cfg,
		Database: db,
	}

	// Test provider creation
	providerJSON := `{"name": "test-provider", "endpoint": "https://api.test.com"}`
	req := httptest.NewRequest("POST", "/api/v1/providers", bytes.NewBuffer([]byte(providerJSON)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer admin-token")

	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Test model creation
	modelJSON := `{"name": "test-model", "provider_id": 1}`
	req2 := httptest.NewRequest("POST", "/api/v1/models", bytes.NewBuffer([]byte(modelJSON)))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer admin-token")

	w2 = httptest.NewRecorder()
	server.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	// Test verification flow
	verificationJSON := `{"model_id": 1, "test_data": "test"}`
	req3 := httptest.NewRequest("POST", "/api/v1/models/1/verify", bytes.NewBuffer([]byte(verificationJSON)))
	req3.Header.Set("Content-Type", "application/json")
	req3.Header.Set("Authorization", "Bearer admin-token")

	w3 := httptest.NewRecorder()
	server.ServeHTTP(w3, req3)

	// Should work even if verification fails (that's expected behavior)
	assert.Equal(t, http.StatusOK, w3.Code)
}

// TestErrorHandlingIntegration tests error consistency across the system
func TestErrorHandlingIntegration(t *testing.T) {
	// Setup test server
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: ":memory:",
		},
		API: config.APIConfig{
			JWTSecret: "test-secret-key",
		},
	}

	server := &api.Server{
		Config:   cfg,
		Database: db,
	}

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "InvalidJSON",
			method:         "POST",
			path:           "/auth/login",
			body:           `{"invalid": json}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request",
		},
		{
			name:           "MissingCredentials",
			method:         "POST",
			path:           "/auth/login",
			body:           `{}`,
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid request",
		},
		{
			name:           "UnauthorizedAccess",
			method:         "GET",
			path:           "/api/v1/users/me",
			body:           "",
			expectedStatus: http.StatusUnauthorized,
			expectedError:  "Unauthorized",
		},
		{
			name:           "ResourceNotFound",
			method:         "GET",
			path:           "/api/v1/models/999",
			body:           "",
			expectedStatus: http.StatusNotFound,
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, bytes.NewBuffer([]byte(tt.body)))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var resp map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &resp)
				assert.NoError(t, err)

				if error, ok := resp["error"]; ok {
					if errorMap, ok := error.(map[string]interface{}); ok {
						assert.Equal(t, tt.expectedError, errorMap["message"])
					}
				}
			}
		})
	}
}

// TestRateLimitingIntegration tests rate limiting behavior
func TestRateLimitingIntegration(t *testing.T) {
	// Setup test server with rate limiting
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: ":memory:",
		},
		API: config.APIConfig{
			JWTSecret:  "test-secret-key",
			RateLimit:  2, // Very low limit for testing
			BurstLimit: 5,
		},
	}

	server := &api.Server{
		Config:   cfg,
		Database: db,
	}

	// Make requests rapidly
	responses := make([]*httptest.ResponseRecorder, 10)

	for i := 0; i < 10; i++ {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		responses[i] = w
		server.ServeHTTP(w, req)
	}

	// First few should succeed (within burst)
	successCount := 0
	for i := 0; i < 5; i++ {
		assert.Equal(t, http.StatusOK, responses[i].Code)
		if responses[i].Code == http.StatusOK {
			successCount++
		}
	}

	// Later requests should be rate limited
	limitedCount := 0
	for i := 5; i < 10; i++ {
		if responses[i].Code == http.StatusTooManyRequests {
			limitedCount++
		}
	}

	// Verify rate limiting is working
	assert.True(t, successCount >= 5, "First few requests should succeed")
	assert.True(t, limitedCount >= 3, "Later requests should be rate limited")
}

// TestHealthChecksIntegration tests health check system
func TestHealthChecksIntegration(t *testing.T) {
	// Setup test server
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: ":memory:",
		},
		API: config.APIConfig{
			JWTSecret: "test-secret-key",
		},
	}

	server := &api.Server{
		Config:   cfg,
		Database: db,
	}

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var healthResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &healthResp)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", healthResp["status"])
	assert.Contains(t, healthResp, "timestamp")
	assert.Contains(t, healthResp, "version")
}

// TestConcurrency tests concurrent requests handling
func TestConcurrency(t *testing.T) {
	// Setup test server
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: ":memory:",
		},
		API: config.APIConfig{
			JWTSecret:  "test-secret-key",
			RateLimit:  100,
			BurstLimit: 200,
		},
	}

	server := &api.Server{
		Config:   cfg,
		Database: db,
	}

	// Test concurrent health checks
	const numRequests = 10
	ch := make(chan *httptest.ResponseRecorder, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			server.ServeHTTP(w, req)
			ch <- w
		}()
	}

	// Collect responses
	responses := make([]*httptest.ResponseRecorder, numRequests)
	for i := 0; i < numRequests; i++ {
		select {
		case resp := <-ch:
			responses[i] = resp
		case <-time.After(5 * time.Second):
			t.Fatalf("Timeout waiting for response %d", i)
		}
	}

	// Verify all requests completed successfully
	successCount := 0
	for _, resp := range responses {
		assert.Equal(t, http.StatusOK, resp.Code)
		if resp.Code == http.StatusOK {
			successCount++
		}
	}

	assert.Equal(t, numRequests, successCount, "All concurrent requests should succeed")
}

// TestEndToEndWorkflow tests complete user workflows
func TestEndToEndWorkflow(t *testing.T) {
	// Setup test environment
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Run migrations
	migrationManager := database.NewMigrationManager(db)
	migrationManager.SetupDefaultMigrations()

	if err := migrationManager.InitializeMigrationTable(); err != nil {
		t.Fatalf("Failed to initialize migration table: %v", err)
	}

	if err := migrationManager.MigrateUp(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create test user
	user := &database.User{
		Username:     "workflowuser",
		Email:        "workflow@example.com",
		PasswordHash: "password123",
		FullName:     "Workflow Test User",
		Role:         "user",
		IsActive:     true,
	}

	err = db.CreateUser(user)
	require.NoError(t, err)

	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Path: ":memory:",
		},
		API: config.APIConfig{
			JWTSecret: "test-secret-key",
		},
	}

	server := &api.Server{
		Config:   cfg,
		Database: db,
	}

	// 1. Login
	loginReq := httptest.NewRequest("POST", "/auth/login",
		bytes.NewBuffer([]byte(`{"username": "workflowuser", "password": "password123"}`)))
	loginReq.Header.Set("Content-Type", "application/json")

	loginW := httptest.NewRecorder()
	server.ServeHTTP(loginW, loginReq)

	assert.Equal(t, http.StatusOK, loginW.Code)

	var loginResp map[string]interface{}
	err = json.Unmarshal(loginW.Body.Bytes(), &loginResp)
	require.NoError(t, err)

	token, ok := loginResp["token"].(string)
	require.True(t, ok)

	// 2. Get current user
	meReq := httptest.NewRequest("GET", "/api/v1/users/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+token)

	meW := httptest.NewRecorder()
	server.ServeHTTP(meW, meReq)

	assert.Equal(t, http.StatusOK, meW.Code)

	// 3. Update user profile
	updateReq := httptest.NewRequest("PUT", "/api/v1/users/me",
		bytes.NewBuffer([]byte(`{"full_name": "Updated Workflow User"}`)))
	updateReq.Header.Set("Authorization", "Bearer "+token)
	updateReq.Header.Set("Content-Type", "application/json")

	updateW := httptest.NewRecorder()
	server.ServeHTTP(updateW, updateReq)

	assert.Equal(t, http.StatusOK, updateW.Code)

	// 4. Get updated user
	meReq2 := httptest.NewRequest("GET", "/api/v1/users/me", nil)
	meReq2.Header.Set("Authorization", "Bearer "+token)

	meW2 := httptest.NewRecorder()
	server.ServeHTTP(meW2, meReq2)

	assert.Equal(t, http.StatusOK, meW2.Code)

	var meResp map[string]interface{}
	err = json.Unmarshal(meW2.Body.Bytes(), &meResp)
	require.NoError(t, err)

	// Verify the update worked
	if userData, ok := meResp["full_name"]; ok {
		assert.Equal(t, "Updated Workflow User", userData)
	}

	// 5. Logout (if implemented)
	// Note: This would test token invalidation if logout endpoint existed
}
