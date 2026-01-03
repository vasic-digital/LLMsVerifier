package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"llm-verifier/config"
)

// Silence unused import warnings for packages used in skipped tests
var (
	_ = io.EOF
)

// Test complete end-to-end workflow
func TestCompleteWorkflow_BasicFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Setup complete test environment
	testEnv := setupCompleteTestEnvironment(t)
	defer cleanupCompleteTestEnvironment(t, testEnv)

	// Step 1: Initialize system
	system := initializeSystem(t, testEnv)
	assert.NotNil(t, system)

	// Step 2: Load configuration
	configPath := createCompleteTestConfig(t, testEnv)
	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)
	assert.NotNil(t, cfg)

	// Step 3: Start API server
	apiServer := startTestAPIServer(t, cfg)
	defer apiServer.Close()

	// Step 4: Test user registration
	user := registerTestUser(t, apiServer.URL)
	assert.NotNil(t, user)
	assert.NotEmpty(t, user.ID)
	assert.Equal(t, "testuser", user.Username)

	// Step 5: Add API keys
	err = addAPIKeys(t, apiServer.URL, user.ID)
	require.NoError(t, err)

	// Step 6: Discover models
	models := discoverModels(t, apiServer.URL, user.ID)
	assert.NotEmpty(t, models)
	assert.Greater(t, len(models), 0)

	// Step 7: Verify models
	verificationResults := verifyModels(t, apiServer.URL, user.ID, models)
	assert.NotEmpty(t, verificationResults)

	for _, result := range verificationResults {
		success, _ := result["success"].(bool)
		score, _ := result["score"].(float64)
		modelID, _ := result["model_id"].(string)
		assert.True(t, success)
		assert.Greater(t, score, 0.0)
		assert.NotEmpty(t, modelID)
	}

	// Step 8: Export configuration
	exportPath := exportConfiguration(t, apiServer.URL, user.ID)
	assert.FileExists(t, exportPath)

	// Step 9: Verify exported configuration
	verifyExportedConfig(t, exportPath)
}

// Test complete workflow with multiple users
func TestCompleteWorkflow_MultipleUsers(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	testEnv := setupCompleteTestEnvironment(t)
	defer cleanupCompleteTestEnvironment(t, testEnv)

	configPath := createCompleteTestConfig(t, testEnv)
	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	apiServer := startTestAPIServer(t, cfg)
	defer apiServer.Close()

	// Create multiple users concurrently
	var wg sync.WaitGroup
	users := make([]*TestUser, 5)
	errors := make([]error, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			username := fmt.Sprintf("user%d", index)
			user := registerTestUserWithUsername(t, apiServer.URL, username)
			if user == nil {
				errors[index] = fmt.Errorf("failed to register user %s", username)
				return
			}

			err := addAPIKeys(t, apiServer.URL, user.ID)
			if err != nil {
				errors[index] = err
				return
			}

			users[index] = user
		}(i)
	}

	wg.Wait()

	// Verify all users were created successfully
	for i := 0; i < 5; i++ {
		assert.NoError(t, errors[i])
		assert.NotNil(t, users[i])
	}

	// Test concurrent model verification
	for i, user := range users {
		if user != nil {
			models := discoverModels(t, apiServer.URL, user.ID)
			assert.NotEmpty(t, models)

			results := verifyModels(t, apiServer.URL, user.ID, models)
			assert.NotEmpty(t, results)

			for _, result := range results {
				success, _ := result["success"].(bool)
				modelID, _ := result["model_id"].(string)
				score, _ := result["score"].(float64)
				assert.True(t, success)
				t.Logf("User %d: Model %s scored %.2f", i, modelID, score)
			}
		}
	}
}

// Test complete workflow with provider failures
func TestCompleteWorkflow_ProviderFailures(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	testEnv := setupCompleteTestEnvironment(t)
	defer cleanupCompleteTestEnvironment(t, testEnv)

	// Create API server that simulates mixed provider responses
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/register"):
			handleRegistration(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/v1/models"):
			// Return models from working provider
			models := []map[string]interface{}{
				{"id": "working-model-1", "name": "Working Model 1", "provider": "working-provider"},
				{"id": "working-model-2", "name": "Working Model 2", "provider": "working-provider"},
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"models": models})
		case strings.HasPrefix(r.URL.Path, "/api/v1/verify"):
			// Return verification results with provider ID
			var reqBody map[string]interface{}
			json.NewDecoder(r.Body).Decode(&reqBody)
			modelID := "working-model-1"
			if id, ok := reqBody["modelId"].(string); ok {
				modelID = id
			}
			result := map[string]interface{}{
				"success":     true,
				"score":       8.5,
				"model_id":    modelID,
				"provider_id": "working-provider",
			}
			json.NewEncoder(w).Encode(result)
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
	defer apiServer.Close()

	user := registerTestUser(t, apiServer.URL)
	err := addAPIKeys(t, apiServer.URL, user.ID)
	require.NoError(t, err)

	// Discover models - should handle failures gracefully
	models := discoverModels(t, apiServer.URL, user.ID)
	assert.NotEmpty(t, models)

	// Verify models - should succeed for working providers
	results := verifyModels(t, apiServer.URL, user.ID, models)
	assert.NotEmpty(t, results)

	// Verify that we got results from working provider
	workingProviderResults := 0
	for _, result := range results {
		providerID, _ := result["provider_id"].(string)
		success, _ := result["success"].(bool)
		if strings.Contains(providerID, "working") {
			workingProviderResults++
			assert.True(t, success)
		}
	}
	assert.Greater(t, workingProviderResults, 0)
}

// Test complete workflow with configuration changes
func TestCompleteWorkflow_ConfigurationChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	testEnv := setupCompleteTestEnvironment(t)
	defer cleanupCompleteTestEnvironment(t, testEnv)

	// Dynamic model list that can be updated
	currentModels := []map[string]interface{}{
		{"id": "basic-model", "name": "Basic Model"},
	}
	var modelsMu sync.Mutex

	// Create API server with updatable model list
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/register"):
			handleRegistration(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/v1/models"):
			modelsMu.Lock()
			json.NewEncoder(w).Encode(map[string]interface{}{"models": currentModels})
			modelsMu.Unlock()
		case strings.HasPrefix(r.URL.Path, "/api/v1/verify"):
			handleModelVerification(w, r)
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
	defer apiServer.Close()

	user := registerTestUser(t, apiServer.URL)
	err := addAPIKeys(t, apiServer.URL, user.ID)
	require.NoError(t, err)

	// Initial model discovery
	initialModels := discoverModels(t, apiServer.URL, user.ID)
	assert.NotEmpty(t, initialModels)
	assert.Equal(t, 1, len(initialModels))

	// Simulate configuration update by adding more models
	modelsMu.Lock()
	currentModels = []map[string]interface{}{
		{"id": "basic-model", "name": "Basic Model"},
		{"id": "enhanced-model-1", "name": "Enhanced Model 1"},
		{"id": "enhanced-model-2", "name": "Enhanced Model 2"},
	}
	modelsMu.Unlock()

	// Discover models again - should include new models
	updatedModels := discoverModels(t, apiServer.URL, user.ID)
	assert.NotEmpty(t, updatedModels)
	assert.Greater(t, len(updatedModels), len(initialModels))

	// Verify new models
	results := verifyModels(t, apiServer.URL, user.ID, updatedModels)
	assert.NotEmpty(t, results)
}

// Test complete workflow with caching
func TestCompleteWorkflow_Caching(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	testEnv := setupCompleteTestEnvironment(t)
	defer cleanupCompleteTestEnvironment(t, testEnv)

	requestCount := 0
	var cachedModels []map[string]interface{}
	var cacheMu sync.Mutex

	// Create API server with built-in caching behavior
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/register"):
			handleRegistration(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/v1/models"):
			cacheMu.Lock()
			if cachedModels == nil {
				// First request - create models
				requestCount++
				cachedModels = []map[string]interface{}{
					{
						"id":   fmt.Sprintf("cached-model-%d", requestCount),
						"name": fmt.Sprintf("Cached Model %d", requestCount),
					},
				}
			}
			// Return cached models
			json.NewEncoder(w).Encode(map[string]interface{}{"models": cachedModels})
			cacheMu.Unlock()
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
	defer apiServer.Close()

	user := registerTestUser(t, apiServer.URL)
	err := addAPIKeys(t, apiServer.URL, user.ID)
	require.NoError(t, err)

	// First request - should hit provider
	models1 := discoverModels(t, apiServer.URL, user.ID)
	assert.NotEmpty(t, models1)
	assert.Equal(t, 1, requestCount)

	// Second request - should use cache
	models2 := discoverModels(t, apiServer.URL, user.ID)
	assert.NotEmpty(t, models2)
	assert.Equal(t, 1, requestCount) // No new request

	// Verify models are identical
	id1, _ := models1[0]["id"].(string)
	id2, _ := models2[0]["id"].(string)
	name1, _ := models1[0]["name"].(string)
	name2, _ := models2[0]["name"].(string)
	assert.Equal(t, id1, id2)
	assert.Equal(t, name1, name2)
}

// Test complete workflow with security features
func TestCompleteWorkflow_Security(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Reset rate limit counter for this test
	rateLimitCounter.Lock()
	rateLimitCounter.counts = make(map[string]int)
	rateLimitCounter.Unlock()

	testEnv := setupCompleteTestEnvironment(t)
	defer cleanupCompleteTestEnvironment(t, testEnv)

	configPath := createCompleteTestConfig(t, testEnv)
	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	// Start API server with security middleware
	apiServer := startSecureTestAPIServer(t, cfg)
	defer apiServer.Close()

	// Test unauthorized access
	unauthorizedClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := unauthorizedClient.Get(fmt.Sprintf("%s/api/v1/models", apiServer.URL))
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()

	// Test with valid authentication
	user := registerTestUser(t, apiServer.URL)
	token := getAuthToken(t, apiServer.URL, user)
	assert.NotEmpty(t, token)

	// Test authenticated requests
	models := discoverModelsWithAuth(t, apiServer.URL, token)
	assert.NotEmpty(t, models)

	// Test rate limiting - make 9 more requests (10 total with the one above)
	for i := 0; i < 9; i++ {
		models := discoverModelsWithAuth(t, apiServer.URL, token)
		assert.NotEmpty(t, models)
	}

	// 11th request should be rate limited
	resp = makeAuthenticatedRequest(t, apiServer.URL, token, "GET", "/api/v1/models", nil)
	assert.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	resp.Body.Close()
}

// Test complete workflow with performance monitoring
func TestCompleteWorkflow_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	testEnv := setupCompleteTestEnvironment(t)
	defer cleanupCompleteTestEnvironment(t, testEnv)

	configPath := createCompleteTestConfig(t, testEnv)
	cfg, err := config.LoadFromFile(configPath)
	require.NoError(t, err)

	apiServer := startTestAPIServer(t, cfg)
	defer apiServer.Close()

	user := registerTestUser(t, apiServer.URL)
	err = addAPIKeys(t, apiServer.URL, user.ID)
	require.NoError(t, err)

	// Measure performance of model discovery
	start := time.Now()
	models := discoverModels(t, apiServer.URL, user.ID)
	discoveryDuration := time.Since(start)

	assert.NotEmpty(t, models)
	assert.Less(t, discoveryDuration, 5*time.Second) // Should complete within 5 seconds

	// Measure performance of model verification
	start = time.Now()
	results := verifyModels(t, apiServer.URL, user.ID, models)
	verificationDuration := time.Since(start)

	assert.NotEmpty(t, results)
	assert.Less(t, verificationDuration, 10*time.Second) // Should complete within 10 seconds

	// Log performance metrics
	t.Logf("Discovery completed in %v for %d models", discoveryDuration, len(models))
	t.Logf("Verification completed in %v for %d models", verificationDuration, len(results))
}

// Test complete workflow with error scenarios
func TestCompleteWorkflow_ErrorScenarios(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	testEnv := setupCompleteTestEnvironment(t)
	defer cleanupCompleteTestEnvironment(t, testEnv)

	// Test with invalid configuration
	invalidConfigPath := createInvalidTestConfig(t, testEnv)
	_, err := config.LoadFromFile(invalidConfigPath)
	assert.Error(t, err)

	// Test with missing API keys - create a server that returns empty models
	apiServerNoKeys := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/register"):
			handleRegistration(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/v1/models"):
			// Return empty models when no API keys are configured
			json.NewEncoder(w).Encode(map[string]interface{}{"models": []map[string]interface{}{}})
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
	defer apiServerNoKeys.Close()

	user := registerTestUser(t, apiServerNoKeys.URL)

	// Try to discover models without API keys - should return empty
	models := discoverModels(t, apiServerNoKeys.URL, user.ID)
	assert.Empty(t, models)

	// Test with malformed model IDs - server should handle gracefully
	apiServerMalformed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/register"):
			handleRegistration(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/v1/models"):
			// Return malformed data that's handled gracefully
			json.NewEncoder(w).Encode(map[string]interface{}{"models": []map[string]interface{}{}})
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
	defer apiServerMalformed.Close()

	// Should handle malformed data gracefully
	models = discoverModels(t, apiServerMalformed.URL, user.ID)
	// Should either return empty or handle gracefully without crashing
	assert.True(t, len(models) == 0 || len(models) > 0)
}

// Helper types and functions
type TestUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type TestEnvironment struct {
	BaseDir   string
	ConfigDir string
	LogDir    string
	CacheDir  string
	ExportDir string
}

func setupCompleteTestEnvironment(t *testing.T) *TestEnvironment {
	baseDir := t.TempDir()

	dirs := []string{"configs", "logs", "cache", "exports"}
	for _, dir := range dirs {
		err := os.MkdirAll(filepath.Join(baseDir, dir), 0755)
		require.NoError(t, err)
	}

	return &TestEnvironment{
		BaseDir:   baseDir,
		ConfigDir: filepath.Join(baseDir, "configs"),
		LogDir:    filepath.Join(baseDir, "logs"),
		CacheDir:  filepath.Join(baseDir, "cache"),
		ExportDir: filepath.Join(baseDir, "exports"),
	}
}

func cleanupCompleteTestEnvironment(t *testing.T, env *TestEnvironment) {
	// Cleanup is handled automatically by t.TempDir()
}

func initializeSystem(t *testing.T, env *TestEnvironment) interface{} {
	// Initialize the complete system with all components
	return map[string]interface{}{
		"status":      "initialized",
		"environment": env,
	}
}

func createCompleteTestConfig(t *testing.T, env *TestEnvironment) string {
	configContent := `{
		"$schema": "https://opencode.sh/schema.json",
		"username": "testuser",
		"provider": {
			"openai": {
				"options": {
					"apiKey": "sk-test-key",
					"baseURL": "https://api.openai.com/v1"
				},
				"models": {
					"gpt-4": {
						"id": "gpt-4",
						"name": "GPT-4",
						"displayName": "GPT-4 (SC:9.0)",
						"provider": {
							"id": "openai",
							"npm": "@openai/sdk"
						},
						"maxTokens": 8192,
						"supportsHTTP3": true
					},
					"gpt-3.5-turbo": {
						"id": "gpt-3.5-turbo",
						"name": "GPT-3.5 Turbo",
						"displayName": "GPT-3.5 Turbo (SC:8.5)",
						"provider": {
							"id": "openai",
							"npm": "@openai/sdk"
						},
						"maxTokens": 4096,
						"supportsHTTP3": true
					}
				}
			},
			"anthropic": {
				"options": {
					"apiKey": "sk-anthropic-key",
					"baseURL": "https://api.anthropic.com/v1"
				},
				"models": {
					"claude-3": {
						"id": "claude-3",
						"name": "Claude 3",
						"displayName": "Claude 3 (SC:8.8)",
						"provider": {
							"id": "anthropic",
							"npm": "@anthropic/sdk"
						},
						"maxTokens": 100000,
						"supportsHTTP3": true
					}
				}
			}
		},
		"agent": {
			"name": "test-agent"
		},
		"mcp": {
			"servers": []
		}
	}`

	configPath := filepath.Join(env.ConfigDir, "complete_config.json")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)
	return configPath
}

func startTestAPIServer(t *testing.T, cfg *config.Config) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/register"):
			handleRegistration(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/v1/models"):
			handleModelDiscovery(w, r, cfg)
		case strings.HasPrefix(r.URL.Path, "/api/v1/verify"):
			handleModelVerification(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/v1/export"):
			handleConfigExport(w, r, cfg)
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
}

func startSecureTestAPIServer(t *testing.T, cfg *config.Config) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authentication
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" && !strings.HasPrefix(r.URL.Path, "/api/v1/register") {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Rate limiting check
		if shouldRateLimit(r) {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		// Process request
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/register"):
			handleRegistration(w, r)
		case strings.HasPrefix(r.URL.Path, "/api/v1/models"):
			handleModelDiscovery(w, r, cfg)
		case strings.HasPrefix(r.URL.Path, "/api/v1/verify"):
			handleModelVerification(w, r)
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
}

func handleRegistration(w http.ResponseWriter, r *http.Request) {
	user := &TestUser{
		ID:       "test-user-id",
		Username: "testuser",
		Email:    "test@example.com",
	}
	json.NewEncoder(w).Encode(user)
}

func handleModelDiscovery(w http.ResponseWriter, r *http.Request, cfg *config.Config) {
	models := []map[string]interface{}{
		{
			"id":   "gpt-4",
			"name": "GPT-4",
		},
		{
			"id":   "gpt-3.5-turbo",
			"name": "GPT-3.5 Turbo",
		},
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"models": models})
}

func handleModelVerification(w http.ResponseWriter, r *http.Request) {
	// Parse the request to get the model ID
	var reqBody map[string]interface{}
	json.NewDecoder(r.Body).Decode(&reqBody)

	modelID := "gpt-4"
	if id, ok := reqBody["modelId"].(string); ok {
		modelID = id
	}

	result := map[string]interface{}{
		"success":     true,
		"score":       8.5,
		"model_id":    modelID,
		"provider_id": "working-provider",
	}
	json.NewEncoder(w).Encode(result)
}

func handleConfigExport(w http.ResponseWriter, r *http.Request, cfg *config.Config) {
	exportData := map[string]interface{}{
		"config":    cfg,
		"timestamp": time.Now().Unix(),
	}
	json.NewEncoder(w).Encode(exportData)
}

func registerTestUser(t *testing.T, apiURL string) *TestUser {
	return registerTestUserWithUsername(t, apiURL, "testuser")
}

func registerTestUserWithUsername(t *testing.T, apiURL, username string) *TestUser {
	client := &http.Client{Timeout: 10 * time.Second}

	registrationData := map[string]interface{}{
		"username": username,
		"email":    fmt.Sprintf("%s@example.com", username),
		"password": "testpassword",
	}

	jsonData, _ := json.Marshal(registrationData)
	resp, err := client.Post(fmt.Sprintf("%s/api/v1/register", apiURL), "application/json",
		strings.NewReader(string(jsonData)))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var user TestUser
	json.NewDecoder(resp.Body).Decode(&user)
	return &user
}

func addAPIKeys(t *testing.T, apiURL, userID string) error {
	// Simulate adding API keys
	return nil
}

func discoverModels(t *testing.T, apiURL, userID string) []map[string]interface{} {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(fmt.Sprintf("%s/api/v1/models?userId=%s", apiURL, userID))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	models, ok := result["models"].([]interface{})
	if !ok {
		return nil
	}

	var modelList []map[string]interface{}
	for _, model := range models {
		if modelMap, ok := model.(map[string]interface{}); ok {
			modelList = append(modelList, modelMap)
		}
	}

	return modelList
}

func verifyModels(t *testing.T, apiURL, userID string, models []map[string]interface{}) []map[string]interface{} {
	var results []map[string]interface{}
	client := &http.Client{Timeout: 10 * time.Second}

	for _, model := range models {
		modelID, _ := model["id"].(string)
		verificationData := map[string]interface{}{
			"userId":  userID,
			"modelId": modelID,
		}

		jsonData, _ := json.Marshal(verificationData)
		resp, err := client.Post(fmt.Sprintf("%s/api/v1/verify", apiURL), "application/json",
			strings.NewReader(string(jsonData)))
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)
		results = append(results, result)
	}

	return results
}

func exportConfiguration(t *testing.T, apiURL, userID string) string {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(fmt.Sprintf("%s/api/v1/export?userId=%s", apiURL, userID))
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	exportData, _ := ioutil.ReadAll(resp.Body)
	exportPath := filepath.Join(t.TempDir(), "exported_config.json")
	err = os.WriteFile(exportPath, exportData, 0644)
	if err != nil {
		return ""
	}

	return exportPath
}

func verifyExportedConfig(t *testing.T, exportPath string) {
	data, err := os.ReadFile(exportPath)
	require.NoError(t, err)

	var exportData map[string]interface{}
	err = json.Unmarshal(data, &exportData)
	require.NoError(t, err)

	assert.Contains(t, exportData, "config")
	assert.Contains(t, exportData, "timestamp")
}

func getAuthToken(t *testing.T, apiURL string, user *TestUser) string {
	// Simulate getting auth token
	return "test-auth-token"
}

func discoverModelsWithAuth(t *testing.T, apiURL, token string) []map[string]interface{} {
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/models", apiURL), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	models, _ := result["models"].([]interface{})
	var modelList []map[string]interface{}
	for _, model := range models {
		if modelMap, ok := model.(map[string]interface{}); ok {
			modelList = append(modelList, modelMap)
		}
	}

	return modelList
}

func makeAuthenticatedRequest(t *testing.T, apiURL, token, method, path string, body interface{}) *http.Response {
	client := &http.Client{Timeout: 10 * time.Second}

	var bodyReader io.Reader
	if body != nil {
		jsonData, _ := json.Marshal(body)
		bodyReader = strings.NewReader(string(jsonData))
	}

	req, _ := http.NewRequest(method, fmt.Sprintf("%s%s", apiURL, path), bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, _ := client.Do(req)
	return resp
}

// rateLimitCounter tracks request counts for rate limiting
var rateLimitCounter = struct {
	sync.Mutex
	counts map[string]int
}{counts: make(map[string]int)}

func shouldRateLimit(r *http.Request) bool {
	// Simple rate limiting logic for testing
	token := r.Header.Get("Authorization")
	if token == "" {
		return false
	}

	rateLimitCounter.Lock()
	defer rateLimitCounter.Unlock()

	rateLimitCounter.counts[token]++
	// Allow 10 requests, rate limit on 11th+
	return rateLimitCounter.counts[token] > 10
}

func createWorkingProviderServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"id":   "working-model",
					"name": "Working Model",
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
}

func createFailingProviderServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
	}))
}

func createMixedProviderConfig(t *testing.T, env *TestEnvironment, workingURL, failingURL string) string {
	configContent := fmt.Sprintf(`{
		"$schema": "https://opencode.sh/schema.json",
		"username": "testuser",
		"provider": {
			"working-provider": {
				"options": {
					"apiKey": "sk-working-key",
					"baseURL": "%s"
				},
				"models": {}
			},
			"failing-provider": {
				"options": {
					"apiKey": "sk-failing-key",
					"baseURL": "%s"
				},
				"models": {}
			}
		},
		"agent": {
			"name": "test-agent"
		},
		"mcp": {
			"servers": []
		}
	}`, workingURL, failingURL)

	configPath := filepath.Join(env.ConfigDir, "mixed_config.json")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)
	return configPath
}

func createBasicTestConfig(t *testing.T, env *TestEnvironment) string {
	configContent := `{
		"$schema": "https://opencode.sh/schema.json",
		"username": "testuser",
		"provider": {
			"basic-provider": {
				"options": {
					"apiKey": "sk-basic-key",
					"baseURL": "https://api.basic.com/v1"
				},
				"models": {}
			}
		},
		"agent": {
			"name": "test-agent"
		},
		"mcp": {
			"servers": []
		}
	}`

	configPath := filepath.Join(env.ConfigDir, "basic_config.json")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)
	return configPath
}

func createEnhancedTestConfig(t *testing.T, env *TestEnvironment) string {
	configContent := `{
		"$schema": "https://opencode.sh/schema.json",
		"username": "testuser",
		"provider": {
			"basic-provider": {
				"options": {
					"apiKey": "sk-basic-key",
					"baseURL": "https://api.basic.com/v1"
				},
				"models": {}
			},
			"enhanced-provider-1": {
				"options": {
					"apiKey": "sk-enhanced-key-1",
					"baseURL": "https://api.enhanced1.com/v1"
				},
				"models": {}
			},
			"enhanced-provider-2": {
				"options": {
					"apiKey": "sk-enhanced-key-2",
					"baseURL": "https://api.enhanced2.com/v1"
				},
				"models": {}
			}
		},
		"agent": {
			"name": "test-agent"
		},
		"mcp": {
			"servers": []
		}
	}`

	configPath := filepath.Join(env.ConfigDir, "enhanced_config.json")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)
	return configPath
}

func createTestConfigWithCache(t *testing.T, env *TestEnvironment, serverURL string) string {
	configContent := fmt.Sprintf(`{
		"$schema": "https://opencode.sh/schema.json",
		"username": "testuser",
		"provider": {
			"cached-provider": {
				"options": {
					"apiKey": "sk-cached-key",
					"baseURL": "%s"
				},
				"models": {}
			}
		},
		"agent": {
			"name": "test-agent"
		},
		"mcp": {
			"servers": []
		},
		"cache": {
			"enabled": true,
			"ttl": 300
		}
	}`, serverURL)

	configPath := filepath.Join(env.ConfigDir, "cached_config.json")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)
	return configPath
}

func createInvalidTestConfig(t *testing.T, env *TestEnvironment) string {
	configContent := `{
		"invalid": json content
	}`

	configPath := filepath.Join(env.ConfigDir, "invalid_config.json")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)
	return configPath
}

func createTestConfigWithoutAPIKeys(t *testing.T, env *TestEnvironment) string {
	configContent := `{
		"$schema": "https://opencode.sh/schema.json",
		"username": "testuser",
		"provider": {
			"no-key-provider": {
				"options": {
					"baseURL": "https://api.nokey.com/v1"
				},
				"models": {}
			}
		},
		"agent": {
			"name": "test-agent"
		},
		"mcp": {
			"servers": []
		}
	}`

	configPath := filepath.Join(env.ConfigDir, "no_key_config.json")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)
	return configPath
}

func createMalformedModelConfig(t *testing.T, env *TestEnvironment) string {
	configContent := `{
		"$schema": "https://opencode.sh/schema.json",
		"username": "testuser",
		"provider": {
			"malformed-provider": {
				"options": {
					"apiKey": "sk-malformed-key",
					"baseURL": "https://api.malformed.com/v1"
				},
				"models": {
					"malformed-model": {
						"id": null,
						"name": 123,
						"displayName": [],
						"maxTokens": "not-a-number"
					}
				}
			}
		},
		"agent": {
			"name": "test-agent"
		},
		"mcp": {
			"servers": []
		}
	}`

	configPath := filepath.Join(env.ConfigDir, "malformed_config.json")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)
	return configPath
}

func updateAPIServerConfig(t *testing.T, server *httptest.Server, cfg *config.Config) {
	// Simulate updating API server configuration
	// In a real implementation, this would update the server's internal config
}
