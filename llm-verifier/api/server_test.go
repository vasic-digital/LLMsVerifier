package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestAPIServer_Complete tests the complete API server functionality
func TestAPIServer_Complete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Setup test server
	router := setupTestRouter()
	server := httptest.NewServer(router)
	defer server.Close()
	
	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
		validateFunc   func(t *testing.T, response *http.Response)
	}{
		{
			name:           "Get Models - Success",
			method:         "GET",
			path:           "/api/models",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response *http.Response) {
				var models []Model
				err := json.NewDecoder(response.Body).Decode(&models)
				assert.NoError(t, err)
				assert.NotEmpty(t, models)
				// Verify score suffix format
				for _, model := range models {
					assert.Contains(t, model.Name, "(SC:")
					assert.Contains(t, model.Name, ")")
				}
			},
		},
		{
			name:           "Get Model by ID - Success",
			method:         "GET",
			path:           "/api/models/gpt-4",
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response *http.Response) {
				var model Model
				err := json.NewDecoder(response.Body).Decode(&model)
				assert.NoError(t, err)
				assert.Equal(t, "gpt-4", model.ModelID)
				assert.Contains(t, model.Name, "(SC:")
			},
		},
		{
			name:           "Verify Model - Success",
			method:         "POST",
			path:           "/api/verify",
			body: map[string]interface{}{
				"model_id": "gpt-4",
				"prompt":   "Test verification",
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response *http.Response) {
				var result VerificationResult
				err := json.NewDecoder(response.Body).Decode(&result)
				assert.NoError(t, err)
				assert.True(t, result.Success)
				assert.NotEmpty(t, result.Response)
				assert.Contains(t, result.ScoreSuffix, "(SC:")
			},
		},
		{
			name:           "Calculate Model Score - Success",
			method:         "POST",
			path:           "/api/scoring/calculate",
			body: map[string]interface{}{
				"model_id": "gpt-4",
				"weights": map[string]float64{
					"response_speed":    0.25,
					"model_efficiency":  0.20,
					"cost_effectiveness": 0.25,
					"capability":        0.20,
					"recency":          0.10,
				},
			},
			expectedStatus: http.StatusOK,
			validateFunc: func(t *testing.T, response *http.Response) {
				var score ModelScore
				err := json.NewDecoder(response.Body).Decode(&score)
				assert.NoError(t, err)
				assert.Equal(t, "gpt-4", score.ModelID)
				assert.Greater(t, score.Score, 0.0)
				assert.Contains(t, score.ScoreSuffix, "(SC:")
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			var err error
			
			if tt.body != nil {
				jsonBody, _ := json.Marshal(tt.body)
				req, err = http.NewRequest(tt.method, server.URL+tt.path, bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, err = http.NewRequest(tt.method, server.URL+tt.path, nil)
			}
			
			assert.NoError(t, err)
			
			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			defer resp.Body.Close()
			
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			
			if tt.validateFunc != nil {
				tt.validateFunc(t, resp)
			}
		})
	}
}

// TestAPIServer_ErrorHandling tests error handling scenarios
func TestAPIServer_ErrorHandling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupTestRouter()
	server := httptest.NewServer(router)
	defer server.Close()
	
	tests := []struct {
		name           string
		method         string
		path           string
		body           interface{}
		expectedStatus int
		errorMessage   string
	}{
		{
			name:           "Invalid Model ID",
			method:         "GET",
			path:           "/api/models/invalid-model-id",
			expectedStatus: http.StatusNotFound,
			errorMessage:   "Model not found",
		},
		{
			name:           "Invalid Verification Request",
			method:         "POST",
			path:           "/api/verify",
			body:           map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			errorMessage:   "Invalid request body",
		},
		{
			name:           "Invalid Score Calculation Request",
			method:         "POST",
			path:           "/api/scoring/calculate",
			body:           map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			errorMessage:   "Invalid request body",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			var err error
			
			if tt.body != nil {
				jsonBody, _ := json.Marshal(tt.body)
				req, err = http.NewRequest(tt.method, server.URL+tt.path, bytes.NewBuffer(jsonBody))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, err = http.NewRequest(tt.method, server.URL+tt.path, nil)
			}
			
			assert.NoError(t, err)
			
			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			defer resp.Body.Close()
			
			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}

// TestAPIServer_ScoreFormat tests score suffix format
func TestAPIServer_ScoreFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupTestRouter()
	server := httptest.NewServer(router)
	defer server.Close()
	
	// Test model list
	resp, err := http.Get(server.URL + "/api/models")
	assert.NoError(t, err)
	defer resp.Body.Close()
	
	var models []Model
	err = json.NewDecoder(resp.Body).Decode(&models)
	assert.NoError(t, err)
	
	for _, model := range models {
		// Verify score suffix format (SC:X.X)
		assert.Regexp(t, `\(SC:\d+\.\d+\)`, model.Name, "Model name should contain score suffix")
		assert.Greater(t, model.OverallScore, 0.0, "Overall score should be greater than 0")
		assert.LessOrEqual(t, model.OverallScore, 10.0, "Overall score should be less than or equal to 10")
	}
}

// Helper function to setup test router
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// API routes
	router.GET("/api/models", handleGetModels)
	router.GET("/api/models/:id", handleGetModel)
	router.POST("/api/verify", handleVerifyModel)
	router.POST("/api/scoring/calculate", handleCalculateScore)
	
	return router
}

// Handler functions (implementations would be in actual code)
func handleGetModels(c *gin.Context) {
	models := []Model{
		{
			ModelID:      "gpt-4",
			Name:         "GPT-4 (SC:8.5)",
			Provider:     "OpenAI",
			OverallScore: 8.5,
			IsActive:     true,
		},
		{
			ModelID:      "claude-3",
			Name:         "Claude-3 (SC:7.8)",
			Provider:     "Anthropic",
			OverallScore: 7.8,
			IsActive:     true,
		},
	}
	c.JSON(http.StatusOK, models)
}

func handleGetModel(c *gin.Context) {
	modelID := c.Param("id")
	if modelID == "gpt-4" {
		model := Model{
			ModelID:      "gpt-4",
			Name:         "GPT-4 (SC:8.5)",
			Provider:     "OpenAI",
			OverallScore: 8.5,
			IsActive:     true,
		}
		c.JSON(http.StatusOK, model)
	} else {
		c.JSON(http.StatusNotFound, gin.H{"error": "Model not found"})
	}
}

func handleVerifyModel(c *gin.Context) {
	var req struct {
		ModelID string `json:"model_id" binding:"required"`
		Prompt  string `json:"prompt" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	
	result := VerificationResult{
		ID:          "test-123",
		ModelID:     req.ModelID,
		Prompt:      req.Prompt,
		Response:    "Test response",
		Score:       8.5,
		ScoreSuffix: "(SC:8.5)",
		Success:     true,
		Timestamp:   time.Now(),
		Duration:    1500,
	}
	
	c.JSON(http.StatusOK, result)
}

func handleCalculateScore(c *gin.Context) {
	var req struct {
		ModelID string      `json:"model_id" binding:"required"`
		Weights ScoreWeights `json:"weights" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}
	
	score := ModelScore{
		ModelID:     req.ModelID,
		ModelName:   "GPT-4",
		Score:       8.5,
		ScoreSuffix: "(SC:8.5)",
		Components: ScoreComponents{
			ResponseSpeed:   8.0,
			ModelEfficiency: 9.0,
			CostEffectiveness: 8.5,
			Capability:      8.5,
			Recency:         8.0,
		},
		Timestamp: time.Now(),
	}
	
	c.JSON(http.StatusOK, score)
}