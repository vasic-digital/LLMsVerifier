package unit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"llm-verifier/providers"
	"llm-verifier/scoring"
	"llm-verifier/verification"
)

// Mock implementations for testing
type MockProviderService struct {
	mock.Mock
}

func (m *MockProviderService) GetModels(provider string) ([]providers.Model, error) {
	args := m.Called(provider)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]providers.Model), args.Error(1)
}

func (m *MockProviderService) VerifyModel(ctx context.Context, provider, modelID string) (*verification.Result, error) {
	args := m.Called(ctx, provider, modelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*verification.Result), args.Error(1)
}

type MockScoringEngine struct {
	mock.Mock
}

func (m *MockScoringEngine) CalculateScore(model *providers.Model, verificationResult *verification.Result) float64 {
	args := m.Called(model, verificationResult)
	return args.Get(0).(float64)
}

func (m *MockScoringEngine) GetScoreExplanation(score float64) string {
	args := m.Called(score)
	return args.String(0)
}

// Test cases for model verification
func TestModelVerification_ValidModel(t *testing.T) {
	mockService := new(MockProviderService)
	mockScoring := new(MockScoringEngine)

	model := providers.Model{
		ID:          "gpt-4",
		Name:        "GPT-4",
		Provider:    "openai",
		MaxTokens:   8192,
		ContextWindow: 128000,
	}

	verificationResult := &verification.Result{
		Success:      true,
		ResponseTime: 150 * time.Millisecond,
		Accuracy:     0.95,
		Latency:      120 * time.Millisecond,
	}

	mockService.On("VerifyModel", mock.Anything, "openai", "gpt-4").Return(verificationResult, nil)
	mockScoring.On("CalculateScore", &model, verificationResult).Return(8.5)
	mockScoring.On("GetScoreExplanation", 8.5).Return("Excellent performance")

	verifier := verification.NewModelVerifier(mockService, mockScoring)
	result, err := verifier.VerifyModel(context.Background(), "openai", "gpt-4")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 8.5, result.Score)
	assert.Equal(t, "Excellent performance", result.ScoreExplanation)

	mockService.AssertExpectations(t)
	mockScoring.AssertExpectations(t)
}

func TestModelVerification_InvalidModel(t *testing.T) {
	mockService := new(MockProviderService)
	mockScoring := new(MockScoringEngine)

	mockService.On("VerifyModel", mock.Anything, "invalid", "invalid-model").Return(nil, fmt.Errorf("model not found"))

	verifier := verification.NewModelVerifier(mockService, mockScoring)
	result, err := verifier.VerifyModel(context.Background(), "invalid", "invalid-model")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "model not found")

	mockService.AssertExpectations(t)
}

func TestModelVerification_Timeout(t *testing.T) {
	mockService := new(MockProviderService)
	mockScoring := new(MockScoringEngine)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	mockService.On("VerifyModel", ctx, "slow-provider", "slow-model").Return(nil, context.DeadlineExceeded)

	verifier := verification.NewModelVerifier(mockService, mockScoring)
	result, err := verifier.VerifyModel(ctx, "slow-provider", "slow-model")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, context.DeadlineExceeded, err)

	mockService.AssertExpectations(t)
}

func TestModelVerification_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		provider      string
		modelID       string
		setupMock     func(*MockProviderService, *MockScoringEngine)
		expectedError bool
		expectedScore float64
	}{
		{
			name:     "Empty provider",
			provider: "",
			modelID:  "gpt-4",
			setupMock: func(m *MockProviderService, s *MockScoringEngine) {
				m.On("VerifyModel", mock.Anything, "", "gpt-4").Return(nil, fmt.Errorf("invalid provider"))
			},
			expectedError: true,
		},
		{
			name:     "Empty model ID",
			provider: "openai",
			modelID:  "",
			setupMock: func(m *MockProviderService, s *MockScoringEngine) {
				m.On("VerifyModel", mock.Anything, "openai", "").Return(nil, fmt.Errorf("invalid model ID"))
			},
			expectedError: true,
		},
		{
			name:     "Zero score",
			provider: "test",
			modelID:  "test-model",
			setupMock: func(m *MockProviderService, s *MockScoringEngine) {
				result := &verification.Result{
					Success: false,
					Errors:  []string{"verification failed"},
				}
				m.On("VerifyModel", mock.Anything, "test", "test-model").Return(result, nil)
				s.On("CalculateScore", mock.Anything, result).Return(0.0)
				s.On("GetScoreExplanation", 0.0).Return("Verification failed")
			},
			expectedError: false,
			expectedScore: 0.0,
		},
		{
			name:     "Perfect score",
			provider: "test",
			modelID:  "perfect-model",
			setupMock: func(m *MockProviderService, s *MockScoringEngine) {
				result := &verification.Result{
					Success:      true,
					ResponseTime: 50 * time.Millisecond,
					Accuracy:     1.0,
					Latency:      40 * time.Millisecond,
				}
				m.On("VerifyModel", mock.Anything, "test", "perfect-model").Return(result, nil)
				s.On("CalculateScore", mock.Anything, result).Return(10.0)
				s.On("GetScoreExplanation", 10.0).Return("Perfect performance")
			},
			expectedError: false,
			expectedScore: 10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockProviderService)
			mockScoring := new(MockScoringEngine)
			tt.setupMock(mockService, mockScoring)

			verifier := verification.NewModelVerifier(mockService, mockScoring)
			result, err := verifier.VerifyModel(context.Background(), tt.provider, tt.modelID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedScore, result.Score)
			}

			mockService.AssertExpectations(t)
			mockScoring.AssertExpectations(t)
		})
	}
}

// Test scoring system
func TestScoringSystem_CalculateScore(t *testing.T) {
	engine := scoring.NewEngine()

	tests := []struct {
		name               string
		model              providers.Model
		verificationResult *verification.Result
		expectedMinScore   float64
		expectedMaxScore   float64
	}{
		{
			name: "High performance model",
			model: providers.Model{
				ContextWindow: 128000,
				MaxTokens:     8192,
			},
			verificationResult: &verification.Result{
				Success:      true,
				ResponseTime: 100 * time.Millisecond,
				Accuracy:     0.95,
				Latency:      80 * time.Millisecond,
			},
			expectedMinScore: 8.0,
			expectedMaxScore: 10.0,
		},
		{
			name: "Low performance model",
			model: providers.Model{
				ContextWindow: 2048,
				MaxTokens:     512,
			},
			verificationResult: &verification.Result{
				Success:      true,
				ResponseTime: 2000 * time.Millisecond,
				Accuracy:     0.70,
				Latency:      1800 * time.Millisecond,
			},
			expectedMinScore: 0.0,
			expectedMaxScore: 5.0,
		},
		{
			name: "Failed verification",
			model: providers.Model{
				ContextWindow: 8192,
				MaxTokens:     2048,
			},
			verificationResult: &verification.Result{
				Success: false,
				Errors:  []string{"timeout", "invalid response"},
			},
			expectedMinScore: 0.0,
			expectedMaxScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := engine.CalculateScore(&tt.model, tt.verificationResult)
			assert.GreaterOrEqual(t, score, tt.expectedMinScore)
			assert.LessOrEqual(t, score, tt.expectedMaxScore)
		})
	}
}

func TestScoringSystem_GetScoreExplanation(t *testing.T) {
	engine := scoring.NewEngine()

	tests := []struct {
		score          float64
		expectedPhrase string
	}{
		{9.5, "Exceptional"},
		{8.5, "Excellent"},
		{7.0, "Good"},
		{5.0, "Average"},
		{3.0, "Below Average"},
		{1.0, "Poor"},
		{0.0, "Failed"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Score_%.1f", tt.score), func(t *testing.T) {
			explanation := engine.GetScoreExplanation(tt.score)
			assert.Contains(t, explanation, tt.expectedPhrase)
		})
	}
}

// Test HTTP client functionality
func TestHTTPClient_ProviderRequests(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/models":
			models := []providers.Model{
				{ID: "gpt-4", Name: "GPT-4", Provider: "openai"},
				{ID: "gpt-3.5-turbo", Name: "GPT-3.5 Turbo", Provider: "openai"},
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": models,
			})
		case "/v1/chat/completions":
			response := map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]interface{}{
							"content": "Test response",
						},
					},
				},
			}
			json.NewEncoder(w).Encode(response)
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Test model discovery
	client := providers.NewHTTPClient(server.URL, "test-api-key")
	models, err := client.GetModels()

	require.NoError(t, err)
	assert.Len(t, models, 2)
	assert.Equal(t, "gpt-4", models[0].ID)
	assert.Equal(t, "GPT-4", models[0].Name)
}

// Test configuration validation
func TestConfiguration_Validation(t *testing.T) {
	tests := []struct {
		name          string
		config        map[string]interface{}
		expectValid   bool
		expectedError string
	}{
		{
			name: "Valid configuration",
			config: map[string]interface{}{
				"providers": map[string]interface{}{
					"openai": map[string]interface{}{
						"api_key": "sk-test-key",
						"base_url": "https://api.openai.com/v1",
					},
				},
			},
			expectValid: true,
		},
		{
			name: "Missing API key",
			config: map[string]interface{}{
				"providers": map[string]interface{}{
					"openai": map[string]interface{}{
						"base_url": "https://api.openai.com/v1",
					},
				},
			},
			expectValid:   false,
			expectedError: "missing API key",
		},
		{
			name: "Invalid base URL",
			config: map[string]interface{}{
				"providers": map[string]interface{}{
					"openai": map[string]interface{}{
						"api_key": "sk-test-key",
						"base_url": "not-a-url",
					},
				},
			},
			expectValid:   false,
			expectedError: "invalid base URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := providers.NewConfigValidator()
			err := validator.Validate(tt.config)

			if tt.expectValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				if tt.expectedError != "" {
					assert.Contains(t, err.Error(), tt.expectedError)
				}
			}
		})
	}
}

// Test error handling and recovery
func TestErrorHandling_Recovery(t *testing.T) {
	tests := []struct {
		name          string
		errorType     string
		setupScenario func(*MockProviderService)
		expectRecovery bool
	}{
		{
			name:      "Network timeout recovery",
			errorType: "timeout",
			setupScenario: func(m *MockProviderService) {
				// First call fails with timeout
				m.On("VerifyModel", mock.Anything, "unreliable", "model").
					Return(nil, fmt.Errorf("timeout: request exceeded 30s")).Once()
				// Retry succeeds
				result := &verification.Result{Success: true, ResponseTime: 200 * time.Millisecond}
				m.On("VerifyModel", mock.Anything, "unreliable", "model").
					Return(result, nil).Once()
			},
			expectRecovery: true,
		},
		{
			name:      "Rate limit recovery",
			errorType: "rate_limit",
			setupScenario: func(m *MockProviderService) {
				// First call fails with rate limit
				m.On("VerifyModel", mock.Anything, "rate-limited", "model").
					Return(nil, fmt.Errorf("rate limit exceeded")).Once()
				// After delay, retry succeeds
				result := &verification.Result{Success: true, ResponseTime: 150 * time.Millisecond}
				m.On("VerifyModel", mock.Anything, "rate-limited", "model").
					Return(result, nil).Once()
			},
			expectRecovery: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := new(MockProviderService)
			mockScoring := new(MockScoringEngine)
			tt.setupScenario(mockService)

			verifier := verification.NewModelVerifier(mockService, mockScoring)
			verifier.SetRetryPolicy(2, 100*time.Millisecond) // 2 retries with 100ms delay

			result, err := verifier.VerifyModel(context.Background(), "test", "model")

			if tt.expectRecovery {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.True(t, result.Success)
			} else {
				assert.Error(t, err)
			}

			mockService.AssertExpectations(t)
		})
	}
}

// Test concurrent verification
func TestConcurrentVerification(t *testing.T) {
	mockService := new(MockProviderService)
	mockScoring := new(MockScoringEngine)

	// Setup multiple successful verifications
	for i := 0; i < 10; i++ {
		modelID := fmt.Sprintf("model-%d", i)
		result := &verification.Result{
			Success:      true,
			ResponseTime: time.Duration(100+i*10) * time.Millisecond,
			Accuracy:     0.90 + float64(i)*0.01,
		}
		mockService.On("VerifyModel", mock.Anything, "openai", modelID).Return(result, nil)
		mockScoring.On("CalculateScore", mock.Anything, result).Return(float64(8.0 + i*0.1))
	}

	verifier := verification.NewModelVerifier(mockService, mockScoring)

	// Run concurrent verifications
	var wg sync.WaitGroup
	results := make([]*verification.CompleteResult, 10)
	errors := make([]error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			modelID := fmt.Sprintf("model-%d", index)
			result, err := verifier.VerifyModel(context.Background(), "openai", modelID)
			results[index] = result
			errors[index] = err
		}(i)
	}

	wg.Wait()

	// Verify all results
	for i := 0; i < 10; i++ {
		assert.NoError(t, errors[i])
		assert.NotNil(t, results[i])
		assert.True(t, results[i].Success)
	}

	mockService.AssertExpectations(t)
	mockScoring.AssertExpectations(t)
}