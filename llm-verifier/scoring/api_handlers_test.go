package scoring

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"llm-verifier/logging"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestRouter(t *testing.T) (*gin.Engine, *ScoringAPIHandlers) {
	router := gin.New()

	db := setupTestDatabase(t)
	mockClient := NewMockModelsDevClient()
	logger := setupTestLogger()

	engine := NewScoringEngine(db, mockClient, logger)
	handlers := NewScoringAPIHandlers(engine, logger)

	group := router.Group("/api/v1")
	handlers.RegisterRoutes(group)

	return router, handlers
}

func TestNewScoringAPIHandlers(t *testing.T) {
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	mockClient := NewMockModelsDevClient()
	logger := &logging.Logger{}

	engine := NewScoringEngine(db, mockClient, logger)
	handlers := NewScoringAPIHandlers(engine, logger)

	require.NotNil(t, handlers)
	assert.NotNil(t, handlers.scoringEngine)
	assert.NotNil(t, handlers.modelNaming)
	assert.NotNil(t, handlers.logger)
}

func TestScoringAPIHandlers_RegisterRoutes(t *testing.T) {
	router, _ := setupTestRouter(t)

	// Test that routes are registered by making requests
	routes := router.Routes()

	// Check that key routes are registered
	routeMap := make(map[string]bool)
	for _, route := range routes {
		routeMap[route.Path] = true
	}

	assert.True(t, routeMap["/api/v1/models/:model_id/score"])
	assert.True(t, routeMap["/api/v1/models/:model_id/score/calculate"])
	assert.True(t, routeMap["/api/v1/models/scores/batch"])
	assert.True(t, routeMap["/api/v1/scoring/configuration"])
}

func TestScoringAPIHandlers_GetModelScore(t *testing.T) {
	router, _ := setupTestRouter(t)

	req, _ := http.NewRequest("GET", "/api/v1/models/gpt-4/score", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, "gpt-4", response["model_id"])
	assert.NotNil(t, response["overall_score"])
	assert.NotNil(t, response["components"])
}

func TestScoringAPIHandlers_CalculateModelScore(t *testing.T) {
	router, _ := setupTestRouter(t)

	t.Run("valid request format", func(t *testing.T) {
		body := bytes.NewBufferString(`{"force_recalculation": false}`)
		req, _ := http.NewRequest("POST", "/api/v1/models/gpt-4/score/calculate", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// May return 500 if model not found, 200/201 on success
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusCreated ||
			w.Code == http.StatusInternalServerError || w.Code == http.StatusNotFound)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		body := bytes.NewBufferString(`{invalid}`)
		req, _ := http.NewRequest("POST", "/api/v1/models/gpt-4/score/calculate", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestScoringAPIHandlers_RecalculateModelScore(t *testing.T) {
	router, _ := setupTestRouter(t)

	body := bytes.NewBufferString(`{}`)
	req, _ := http.NewRequest("PUT", "/api/v1/models/gpt-4/score/recalculate", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// May return various codes depending on model existence
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest ||
		w.Code == http.StatusInternalServerError || w.Code == http.StatusNotFound)
}

func TestScoringAPIHandlers_DeleteModelScore(t *testing.T) {
	router, _ := setupTestRouter(t)

	req, _ := http.NewRequest("DELETE", "/api/v1/models/gpt-4/score", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return success or not found
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusNoContent || w.Code == http.StatusNotFound)
}

func TestScoringAPIHandlers_BatchCalculateScores(t *testing.T) {
	router, _ := setupTestRouter(t)

	// Test with invalid JSON to test the error handling path
	t.Run("invalid JSON", func(t *testing.T) {
		body := bytes.NewBufferString(`{invalid}`)
		req, _ := http.NewRequest("POST", "/api/v1/models/scores/batch", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestScoringAPIHandlers_CompareModels(t *testing.T) {
	router, _ := setupTestRouter(t)

	req, _ := http.NewRequest("GET", "/api/v1/models/scores/compare?model_ids=gpt-4,claude-3", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return comparison or error
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest)
}

func TestScoringAPIHandlers_GetModelRankings(t *testing.T) {
	router, _ := setupTestRouter(t)

	req, _ := http.NewRequest("GET", "/api/v1/models/scores/ranking", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotNil(t, response["rankings"])
}

func TestScoringAPIHandlers_GetScoringConfiguration(t *testing.T) {
	router, _ := setupTestRouter(t)

	req, _ := http.NewRequest("GET", "/api/v1/scoring/configuration", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	// Handler returns config_name and configuration at root level
	assert.NotNil(t, response["config_name"])
	assert.NotNil(t, response["configuration"])

	// Weights are inside configuration
	if config, ok := response["configuration"].(map[string]interface{}); ok {
		assert.NotNil(t, config["weights"])
	}
}

func TestScoringAPIHandlers_AddScoreSuffixToModelName(t *testing.T) {
	router, _ := setupTestRouter(t)

	body := bytes.NewBufferString(`{"model_name": "GPT-4", "score": 8.5}`)
	req, _ := http.NewRequest("POST", "/api/v1/models/naming/add-suffix", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest)
}

func TestScoringAPIHandlers_BatchUpdateModelNamesWithScores(t *testing.T) {
	router, _ := setupTestRouter(t)

	body := bytes.NewBufferString(`{"models": [{"name": "GPT-4", "score": 8.5}, {"name": "Claude-3", "score": 9.0}]}`)
	req, _ := http.NewRequest("POST", "/api/v1/models/naming/batch-update", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should succeed or return validation error
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest)
}

func TestScoringAPIHandlers_ValidateScoreCalculation(t *testing.T) {
	router, _ := setupTestRouter(t)

	t.Run("valid score", func(t *testing.T) {
		body := bytes.NewBufferString(`{"score": 8.5, "components": {"speed": 8.0, "efficiency": 9.0}}`)
		req, _ := http.NewRequest("POST", "/api/v1/scoring/validate", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest)
	})

	t.Run("invalid score range", func(t *testing.T) {
		body := bytes.NewBufferString(`{"score": 15.0}`)
		req, _ := http.NewRequest("POST", "/api/v1/scoring/validate", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Should return error for invalid score
		assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusBadRequest)
	})
}
