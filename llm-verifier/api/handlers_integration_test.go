package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"llm-verifier/config"
	"llm-verifier/database"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupIntegrationTestDB creates a temporary database for integration testing
func setupIntegrationTestDB(t *testing.T) (*database.Database, func()) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "integration_test.db")

	db, err := database.New(dbPath)
	require.NoError(t, err)

	cleanup := func() {
		db.Close()
		os.RemoveAll(tempDir)
	}

	return db, cleanup
}

// createTestServerWithDB creates a server with a real database
func createTestServerWithDB(t *testing.T, db *database.Database) *Server {
	cfg := &config.Config{}
	return NewServer(cfg, db)
}

// ==================== ListModels Tests ====================

func TestListModelsHandler_EmptyDatabase(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	server := createTestServerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/models", nil)
	w := httptest.NewRecorder()

	server.ListModelsHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	models := response["models"].([]interface{})
	assert.Empty(t, models)
	assert.Equal(t, float64(0), response["count"])
}

func TestListModelsHandler_WithData(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	// Create test provider
	provider := &database.Provider{
		Name:     "TestProvider",
		Endpoint: "https://api.test.com/v1",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	// Create test models
	model1 := &database.Model{
		ProviderID:         provider.ID,
		ModelID:            "test-model-1",
		Name:               "Test Model 1",
		Description:        "First test model",
		VerificationStatus: "verified",
		OverallScore:       85.5,
		SupportsReasoning:  true,
	}
	err = db.CreateModel(model1)
	require.NoError(t, err)

	model2 := &database.Model{
		ProviderID:         provider.ID,
		ModelID:            "test-model-2",
		Name:               "Test Model 2",
		Description:        "Second test model",
		VerificationStatus: "pending",
		OverallScore:       0,
		IsMultimodal:       true,
	}
	err = db.CreateModel(model2)
	require.NoError(t, err)

	server := createTestServerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/models", nil)
	w := httptest.NewRecorder()

	server.ListModelsHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	models := response["models"].([]interface{})
	assert.Len(t, models, 2)
	assert.Equal(t, float64(2), response["count"])

	// Verify first model
	m1 := models[0].(map[string]interface{})
	assert.Equal(t, "Test Model 1", m1["name"])
	assert.Equal(t, "TestProvider", m1["provider"])
	assert.Equal(t, "verified", m1["status"])
}

func TestListModelsHandler_WithFilters(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	// Create test provider
	provider := &database.Provider{
		Name:     "TestProvider",
		Endpoint: "https://api.test.com/v1",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	// Create models with different statuses
	verifiedModel := &database.Model{
		ProviderID:         provider.ID,
		ModelID:            "verified-model",
		Name:               "Verified Model",
		VerificationStatus: "verified",
		OverallScore:       90.0,
	}
	err = db.CreateModel(verifiedModel)
	require.NoError(t, err)

	pendingModel := &database.Model{
		ProviderID:         provider.ID,
		ModelID:            "pending-model",
		Name:               "Pending Model",
		VerificationStatus: "pending",
		OverallScore:       0,
	}
	err = db.CreateModel(pendingModel)
	require.NoError(t, err)

	server := createTestServerWithDB(t, db)

	// Filter by status
	req := httptest.NewRequest(http.MethodGet, "/api/models?status=verified", nil)
	w := httptest.NewRecorder()

	server.ListModelsHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	models := response["models"].([]interface{})
	assert.Len(t, models, 1)
	m := models[0].(map[string]interface{})
	assert.Equal(t, "Verified Model", m["name"])
}

func TestListModelsHandler_MinScoreFilter(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	provider := &database.Provider{
		Name:     "TestProvider",
		Endpoint: "https://api.test.com/v1",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	// Create models with different scores
	highScoreModel := &database.Model{
		ProviderID:         provider.ID,
		ModelID:            "high-score",
		Name:               "High Score Model",
		VerificationStatus: "verified",
		OverallScore:       95.0,
	}
	err = db.CreateModel(highScoreModel)
	require.NoError(t, err)

	lowScoreModel := &database.Model{
		ProviderID:         provider.ID,
		ModelID:            "low-score",
		Name:               "Low Score Model",
		VerificationStatus: "verified",
		OverallScore:       50.0,
	}
	err = db.CreateModel(lowScoreModel)
	require.NoError(t, err)

	server := createTestServerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/models?min_score=80", nil)
	w := httptest.NewRecorder()

	server.ListModelsHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	models := response["models"].([]interface{})
	assert.Len(t, models, 1)
	m := models[0].(map[string]interface{})
	assert.Equal(t, "High Score Model", m["name"])
}

func TestListModelsHandler_NoDatabaseReturnsError(t *testing.T) {
	cfg := &config.Config{}
	server := NewServer(cfg, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/models", nil)
	w := httptest.NewRecorder()

	server.ListModelsHandler(w, req)

	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

// ==================== GetModel Tests ====================

func TestGetModelHandler_Found(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	provider := &database.Provider{
		Name:     "TestProvider",
		Endpoint: "https://api.test.com/v1",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	contextTokens := 128000
	paramCount := int64(175000000000)
	model := &database.Model{
		ProviderID:          provider.ID,
		ModelID:             "gpt-4",
		Name:                "GPT-4",
		Description:         "Most capable model",
		VerificationStatus:  "verified",
		OverallScore:        95.0,
		ContextWindowTokens: &contextTokens,
		ParameterCount:      &paramCount,
		SupportsReasoning:   true,
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	server := createTestServerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/models/1", nil)
	w := httptest.NewRecorder()

	server.GetModelHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "GPT-4", response["name"])
	assert.Equal(t, "TestProvider", response["provider"])
	assert.Equal(t, "verified", response["status"])
	assert.Equal(t, 95.0, response["score"])
	assert.Contains(t, response["context"], "128K tokens")
	assert.Contains(t, response["parameters"], "billion")
}

func TestGetModelHandler_NotFound(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	server := createTestServerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/models/999", nil)
	w := httptest.NewRecorder()

	server.GetModelHandler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetModelHandler_InvalidID(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	server := createTestServerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/models/invalid", nil)
	w := httptest.NewRecorder()

	server.GetModelHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ==================== ListProviders Tests ====================

func TestListProvidersHandler_EmptyDatabase(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	server := createTestServerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/providers", nil)
	w := httptest.NewRecorder()

	server.ListProvidersHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	providers := response["providers"].([]interface{})
	assert.Empty(t, providers)
}

func TestListProvidersHandler_WithData(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	// Create test providers
	provider1 := &database.Provider{
		Name:        "OpenAI",
		Endpoint:    "https://api.openai.com/v1",
		Description: "OpenAI API",
		IsActive:    true,
	}
	err := db.CreateProvider(provider1)
	require.NoError(t, err)

	provider2 := &database.Provider{
		Name:        "Anthropic",
		Endpoint:    "https://api.anthropic.com/v1",
		Description: "Anthropic API",
		IsActive:    true,
	}
	err = db.CreateProvider(provider2)
	require.NoError(t, err)

	// Add a model to OpenAI provider
	model := &database.Model{
		ProviderID: provider1.ID,
		ModelID:    "gpt-4",
		Name:       "GPT-4",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	server := createTestServerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/providers", nil)
	w := httptest.NewRecorder()

	server.ListProvidersHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	providers := response["providers"].([]interface{})
	assert.Len(t, providers, 2)

	// Find OpenAI provider and verify model count
	for _, p := range providers {
		prov := p.(map[string]interface{})
		if prov["name"] == "OpenAI" {
			assert.Equal(t, float64(1), prov["models"])
			assert.Equal(t, "active", prov["status"])
		}
	}
}

func TestListProvidersHandler_ActiveFilter(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	activeProvider := &database.Provider{
		Name:     "Active",
		Endpoint: "https://api.active.com",
		IsActive: true,
	}
	err := db.CreateProvider(activeProvider)
	require.NoError(t, err)

	inactiveProvider := &database.Provider{
		Name:     "Inactive",
		Endpoint: "https://api.inactive.com",
		IsActive: false,
	}
	err = db.CreateProvider(inactiveProvider)
	require.NoError(t, err)

	server := createTestServerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/providers?is_active=true", nil)
	w := httptest.NewRecorder()

	server.ListProvidersHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	providers := response["providers"].([]interface{})
	assert.Len(t, providers, 1)
	p := providers[0].(map[string]interface{})
	assert.Equal(t, "Active", p["name"])
}

// ==================== AddProvider Tests ====================

func TestAddProviderHandler_Success(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	server := createTestServerWithDB(t, db)

	providerData := map[string]interface{}{
		"name":        "NewProvider",
		"endpoint":    "https://api.new.com/v1",
		"description": "A new provider",
		"is_active":   true,
	}
	body, _ := json.Marshal(providerData)

	req := httptest.NewRequest(http.MethodPost, "/api/providers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.AddProviderHandler(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "provider_added", response["status"])
	assert.Equal(t, "NewProvider", response["name"])
	assert.NotNil(t, response["id"])

	// Verify in database
	providers, err := db.ListProviders(nil)
	require.NoError(t, err)
	assert.Len(t, providers, 1)
	assert.Equal(t, "NewProvider", providers[0].Name)
}

func TestAddProviderHandler_MissingName(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	server := createTestServerWithDB(t, db)

	providerData := map[string]interface{}{
		"endpoint": "https://api.new.com/v1",
	}
	body, _ := json.Marshal(providerData)

	req := httptest.NewRequest(http.MethodPost, "/api/providers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.AddProviderHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "name")
}

func TestAddProviderHandler_MissingEndpoint(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	server := createTestServerWithDB(t, db)

	providerData := map[string]interface{}{
		"name": "NewProvider",
	}
	body, _ := json.Marshal(providerData)

	req := httptest.NewRequest(http.MethodPost, "/api/providers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.AddProviderHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "endpoint")
}

func TestAddProviderHandler_DuplicateName(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	// Create initial provider
	provider := &database.Provider{
		Name:     "ExistingProvider",
		Endpoint: "https://api.existing.com",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	server := createTestServerWithDB(t, db)

	providerData := map[string]interface{}{
		"name":     "ExistingProvider",
		"endpoint": "https://api.new.com/v1",
	}
	body, _ := json.Marshal(providerData)

	req := httptest.NewRequest(http.MethodPost, "/api/providers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.AddProviderHandler(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestAddProviderHandler_WithApiUrl(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	server := createTestServerWithDB(t, db)

	// Use api_url instead of endpoint (fallback)
	providerData := map[string]interface{}{
		"name":    "NewProvider",
		"api_url": "https://api.new.com/v1",
	}
	body, _ := json.Marshal(providerData)

	req := httptest.NewRequest(http.MethodPost, "/api/providers", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.AddProviderHandler(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

// ==================== VerifyModel Tests ====================

func TestVerifyModelHandler_Success(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	provider := &database.Provider{
		Name:     "TestProvider",
		Endpoint: "https://api.test.com/v1",
		IsActive: true,
	}
	err := db.CreateProvider(provider)
	require.NoError(t, err)

	model := &database.Model{
		ProviderID: provider.ID,
		ModelID:    "test-model",
		Name:       "Test Model",
	}
	err = db.CreateModel(model)
	require.NoError(t, err)

	server := createTestServerWithDB(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/models/1/verify", nil)
	w := httptest.NewRecorder()

	server.VerifyModelHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "verification_started", response["status"])
	assert.Equal(t, float64(1), response["model_id"])
	assert.Equal(t, "Test Model", response["model_name"])
	assert.NotNil(t, response["verification_id"])
	assert.NotNil(t, response["job_id"])
	assert.NotNil(t, response["started_at"])
}

func TestVerifyModelHandler_ModelNotFound(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	server := createTestServerWithDB(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/models/999/verify", nil)
	w := httptest.NewRecorder()

	server.VerifyModelHandler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestVerifyModelHandler_InvalidID(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	server := createTestServerWithDB(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/models/invalid/verify", nil)
	w := httptest.NewRecorder()

	server.VerifyModelHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ==================== Health Handler Tests ====================

func TestHealthHandler_Success(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	w := httptest.NewRecorder()

	server.HealthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "healthy", response["status"])
	assert.NotNil(t, response["timestamp"])
}

func TestHealthHandler_MethodNotAllowed(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(http.MethodPost, "/api/health", nil)
	w := httptest.NewRecorder()

	server.HealthHandler(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// ==================== Helper Function Tests ====================

func TestBuildCapabilitiesList(t *testing.T) {
	tests := []struct {
		name     string
		model    *database.Model
		expected []string
	}{
		{
			name: "all capabilities",
			model: &database.Model{
				IsMultimodal:      true,
				SupportsVision:    true,
				SupportsAudio:     true,
				SupportsVideo:     true,
				SupportsReasoning: true,
			},
			expected: []string{"multimodal", "vision", "audio", "video", "reasoning", "text"},
		},
		{
			name:     "text only",
			model:    &database.Model{},
			expected: []string{"text"},
		},
		{
			name: "vision and reasoning",
			model: &database.Model{
				SupportsVision:    true,
				SupportsReasoning: true,
			},
			expected: []string{"vision", "reasoning", "text"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capabilities := buildCapabilitiesList(tt.model)
			assert.Equal(t, tt.expected, capabilities)
		})
	}
}

func TestFormatTokenCount(t *testing.T) {
	tests := []struct {
		tokens   int
		expected string
	}{
		{tokens: 100, expected: "100 tokens"},
		{tokens: 1000, expected: "1K tokens"},
		{tokens: 8000, expected: "8K tokens"},
		{tokens: 128000, expected: "128K tokens"},
		{tokens: 1000000, expected: "1.0M tokens"},
		{tokens: 2000000, expected: "2.0M tokens"},
	}

	for _, tt := range tests {
		result := formatTokenCount(tt.tokens)
		assert.Equal(t, tt.expected, result)
	}
}

func TestFormatParameterCount(t *testing.T) {
	tests := []struct {
		params   int64
		expected string
	}{
		{params: 1000, expected: "1000"},
		{params: 7000000, expected: "7.0 million"},
		{params: 70000000000, expected: "70.0 billion"},
		{params: 175000000000, expected: "175.0 billion"},
		{params: 1760000000000, expected: "1.76 trillion"},
	}

	for _, tt := range tests {
		result := formatParameterCount(tt.params)
		assert.Equal(t, tt.expected, result)
	}
}
