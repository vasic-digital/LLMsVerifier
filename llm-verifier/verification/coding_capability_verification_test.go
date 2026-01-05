package verification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCodingCapabilityVerificationService tests service creation
func TestNewCodingCapabilityVerificationService(t *testing.T) {
	service := NewCodingCapabilityVerificationService(nil, nil)
	assert.NotNil(t, service)
}

// createMockLLMServer creates a mock LLM server for testing
func createMockLLMServer(t *testing.T, response string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": response,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

// TestCodebaseDetection tests the codebase detection capability
func TestCodebaseDetection(t *testing.T) {
	tests := []struct {
		name           string
		response       string
		expectedPassed bool
		minScore       float64
	}{
		{
			name:           "successful detection with all keywords",
			response:       "Yes, I can see your codebase. This is a Go server project with an API backend. It appears to be a web application service.",
			expectedPassed: true,
			minScore:       0.4,
		},
		{
			name:           "partial detection",
			response:       "This appears to be a Go server project with an API backend for web application.",
			expectedPassed: true,
			minScore:       0.4,
		},
		{
			name:           "failed detection - gibberish response",
			response:       "I cannot process this request at this time.",
			expectedPassed: false,
			minScore:       0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createMockLLMServer(t, tt.response)
			defer server.Close()

			service := NewCodingCapabilityVerificationService(nil, nil)
			client := &SimpleProviderClient{
				BaseURL:    server.URL,
				APIKey:     "test-key",
				HTTPClient: server.Client(),
			}

			response, err := service.testCodebaseDetection(context.Background(), "test-provider", "test-model", client)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedPassed, response.Passed)
			assert.GreaterOrEqual(t, response.CapabilityScore, tt.minScore)
		})
	}
}

// TestLanguageDetection tests the language detection capability
func TestLanguageDetection(t *testing.T) {
	tests := []struct {
		name           string
		response       string
		expectedPassed bool
	}{
		{
			name:           "correct language identified - Go",
			response:       "This code is written in Go (Golang). It demonstrates HTTP handling with context support.",
			expectedPassed: true,
		},
		{
			name:           "correct language - lowercase",
			response:       "The programming language is go.",
			expectedPassed: true,
		},
		{
			name:           "wrong language identified",
			response:       "This is Python code for web development.",
			expectedPassed: false,
		},
		{
			name:           "no language identified",
			response:       "This is some code.",
			expectedPassed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createMockLLMServer(t, tt.response)
			defer server.Close()

			service := NewCodingCapabilityVerificationService(nil, nil)
			client := &SimpleProviderClient{
				BaseURL:    server.URL,
				APIKey:     "test-key",
				HTTPClient: server.Client(),
			}

			response, err := service.testLanguageDetection(context.Background(), "test-provider", "test-model", client)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedPassed, response.Passed)
		})
	}
}

// TestCodeGeneration tests the code generation capability
func TestCodeGeneration(t *testing.T) {
	tests := []struct {
		name           string
		response       string
		expectedPassed bool
		minScore       float64
	}{
		{
			name: "complete code generation",
			response: `func isPrime(n int) bool {
				if n <= 1 {
					return false
				}
				for i := 2; i*i <= n; i++ {
					if n % i == 0 {
						return false
					}
				}
				return true
			}`,
			expectedPassed: true,
			minScore:       0.5,
		},
		{
			name:           "partial code generation",
			response:       "func isPrime(n int) bool { return n > 1 }",
			expectedPassed: true,
			minScore:       0.3,
		},
		{
			name:           "no code generated",
			response:       "I'll think about how to solve this problem.",
			expectedPassed: false,
			minScore:       0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createMockLLMServer(t, tt.response)
			defer server.Close()

			service := NewCodingCapabilityVerificationService(nil, nil)
			client := &SimpleProviderClient{
				BaseURL:    server.URL,
				APIKey:     "test-key",
				HTTPClient: server.Client(),
			}

			response, err := service.testCodeGeneration(context.Background(), "test-provider", "test-model", client)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedPassed, response.Passed)
			assert.GreaterOrEqual(t, response.CapabilityScore, tt.minScore)
		})
	}
}

// TestCodeAnalysis tests the code analysis capability
func TestCodeAnalysis(t *testing.T) {
	tests := []struct {
		name           string
		response       string
		expectedPassed bool
		minScore       float64
	}{
		{
			name: "comprehensive analysis",
			response: `This code implements a concurrent worker pool pattern using goroutines and channels.
				The semaphore channel limits the number of concurrent workers, implementing a rate limiter.
				The WaitGroup is used to sync completion of all goroutines before closing the results channel.`,
			expectedPassed: true,
			minScore:       0.3,
		},
		{
			name:           "partial analysis",
			response:       "The code uses goroutines for parallel processing with a channel for results.",
			expectedPassed: true,
			minScore:       0.2,
		},
		{
			name:           "no understanding",
			response:       "This is some code that does things.",
			expectedPassed: false,
			minScore:       0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createMockLLMServer(t, tt.response)
			defer server.Close()

			service := NewCodingCapabilityVerificationService(nil, nil)
			client := &SimpleProviderClient{
				BaseURL:    server.URL,
				APIKey:     "test-key",
				HTTPClient: server.Client(),
			}

			response, err := service.testCodeAnalysis(context.Background(), "test-provider", "test-model", client)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedPassed, response.Passed)
			assert.GreaterOrEqual(t, response.CapabilityScore, tt.minScore)
		})
	}
}

// TestVerifyModelCodingCapabilities tests the full verification flow
func TestVerifyModelCodingCapabilities(t *testing.T) {
	// Create a mock server that responds appropriately to different prompts
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		var response string
		if len(req.Messages) > 0 {
			content := req.Messages[0].Content
			switch {
			case contains(content, "directory structure"):
				response = "Yes, I can see your codebase. This is a Go server project with API handlers."
			case contains(content, "programming language"):
				response = "This code is written in Go (Golang)."
			case contains(content, "prime"):
				response = "func isPrime(n int) bool { if n <= 1 { return false } for i := 2; i*i <= n; i++ { if n % i == 0 { return false } } return true }"
			case contains(content, "concurrent"):
				response = "This code uses goroutines and channels for concurrent processing with a semaphore pattern for rate limiting."
			default:
				response = "I understand your request."
			}
		}

		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": response}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	service := NewCodingCapabilityVerificationService(nil, nil)
	client := &SimpleProviderClient{
		BaseURL:    server.URL,
		APIKey:     "test-key",
		HTTPClient: server.Client(),
	}

	result, err := service.VerifyModelCodingCapabilities(context.Background(), "test-model", "test-provider", client)
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify result structure
	assert.NotEmpty(t, result.VerificationID)
	assert.Equal(t, "test-model", result.ModelID)
	assert.Equal(t, "test-provider", result.ProviderID)
	assert.Contains(t, []string{"verified", "partial", "failed"}, result.Status)

	// Verify individual tests ran
	assert.NotEmpty(t, result.CodebaseDetection.TestType)
	assert.NotEmpty(t, result.LanguageDetection.TestType)
	assert.NotEmpty(t, result.CodeGeneration.TestType)
	assert.NotEmpty(t, result.CodeAnalysis.TestType)

	// Verify aggregate scores
	assert.GreaterOrEqual(t, result.OverallCapabilityScore, 0.0)
	assert.LessOrEqual(t, result.OverallCapabilityScore, 1.0)
	assert.GreaterOrEqual(t, result.ReadinessScore, 0.0)
	assert.LessOrEqual(t, result.ReadinessScore, 1.0)
}

// TestVerifyModelCodingCapabilities_NilClient tests error handling for nil client
func TestVerifyModelCodingCapabilities_NilClient(t *testing.T) {
	service := NewCodingCapabilityVerificationService(nil, nil)

	result, err := service.VerifyModelCodingCapabilities(context.Background(), "test-model", "test-provider", nil)
	assert.Error(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "error", result.Status)
	assert.Equal(t, "Provider client cannot be nil", result.ErrorMessage)
}

// TestFindMatchedKeywords tests keyword matching
func TestFindMatchedKeywords(t *testing.T) {
	service := NewCodingCapabilityVerificationService(nil, nil)

	tests := []struct {
		name     string
		response string
		keywords []string
		expected []string
	}{
		{
			name:     "all keywords found",
			response: "This is Go code with a server API",
			keywords: []string{"go", "server", "api"},
			expected: []string{"go", "server", "api"},
		},
		{
			name:     "partial keywords found",
			response: "This is Go code",
			keywords: []string{"go", "python", "java"},
			expected: []string{"go"},
		},
		{
			name:     "no keywords found",
			response: "Hello world",
			keywords: []string{"go", "python", "java"},
			expected: nil,
		},
		{
			name:     "case insensitive matching",
			response: "This is GO code written in GOLANG",
			keywords: []string{"go", "golang"},
			expected: []string{"go", "golang"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched := service.findMatchedKeywords(tt.response, tt.keywords)
			if tt.expected == nil {
				assert.Nil(t, matched)
			} else {
				assert.ElementsMatch(t, tt.expected, matched)
			}
		})
	}
}

// TestCalculateOverallScore tests score calculation
func TestCalculateOverallScore(t *testing.T) {
	service := NewCodingCapabilityVerificationService(nil, nil)

	result := &CodingCapabilityResult{
		CodebaseDetection: CodingCapabilityResponse{TestType: "codebase_detection", CapabilityScore: 0.8},
		LanguageDetection: CodingCapabilityResponse{TestType: "language_detection", CapabilityScore: 1.0},
		CodeGeneration:    CodingCapabilityResponse{TestType: "code_generation", CapabilityScore: 0.6},
		CodeAnalysis:      CodingCapabilityResponse{TestType: "code_analysis", CapabilityScore: 0.4},
	}

	score := service.calculateOverallScore(result)
	expectedScore := (0.8 + 1.0 + 0.6 + 0.4) / 4.0
	assert.InDelta(t, expectedScore, score, 0.001)
}

// TestCalculateReadinessScore tests readiness score calculation
func TestCalculateReadinessScore(t *testing.T) {
	service := NewCodingCapabilityVerificationService(nil, nil)

	tests := []struct {
		name          string
		result        *CodingCapabilityResult
		expectedScore float64
	}{
		{
			name: "all capabilities present",
			result: &CodingCapabilityResult{
				CanDetectCodebase:   true,
				CanIdentifyLanguage: true,
				CanGenerateCode:     true,
				CanAnalyzeCode:      true,
			},
			expectedScore: 1.0,
		},
		{
			name: "no capabilities",
			result: &CodingCapabilityResult{
				CanDetectCodebase:   false,
				CanIdentifyLanguage: false,
				CanGenerateCode:     false,
				CanAnalyzeCode:      false,
			},
			expectedScore: 0.0,
		},
		{
			name: "essential capabilities only",
			result: &CodingCapabilityResult{
				CanDetectCodebase:   true,
				CanIdentifyLanguage: true,
				CanGenerateCode:     false,
				CanAnalyzeCode:      false,
			},
			expectedScore: 0.55, // 0.3 + 0.25
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := service.calculateReadinessScore(tt.result)
			assert.InDelta(t, tt.expectedScore, score, 0.001)
		})
	}
}

// TestGetCodingCapabilityTestSuite tests that test suite is properly defined
func TestGetCodingCapabilityTestSuite(t *testing.T) {
	service := NewCodingCapabilityVerificationService(nil, nil)
	suite := service.GetCodingCapabilityTestSuite()

	assert.Len(t, suite, 4)

	testTypes := make(map[string]bool)
	for _, test := range suite {
		testTypes[test.TestType] = true
		assert.NotEmpty(t, test.ExpectedHints)
	}

	assert.True(t, testTypes["codebase_detection"])
	assert.True(t, testTypes["language_detection"])
	assert.True(t, testTypes["code_generation"])
	assert.True(t, testTypes["code_analysis"])
}

// TestAPIError tests handling of API errors
func TestAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	service := NewCodingCapabilityVerificationService(nil, nil)
	client := &SimpleProviderClient{
		BaseURL:    server.URL,
		APIKey:     "test-key",
		HTTPClient: server.Client(),
	}

	response, err := service.testCodebaseDetection(context.Background(), "test-provider", "test-model", client)
	// The function returns both a response and an error when API fails
	assert.Error(t, err)
	assert.NotNil(t, response)
	assert.False(t, response.Passed)
	assert.NotEmpty(t, response.Error)
}

// TestResponseTime tests that response time is properly measured
func TestResponseTime(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{"message": map[string]string{"content": "Go code with server API"}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	service := NewCodingCapabilityVerificationService(nil, nil)
	client := &SimpleProviderClient{
		BaseURL:    server.URL,
		APIKey:     "test-key",
		HTTPClient: server.Client(),
	}

	response, err := service.testCodebaseDetection(context.Background(), "test-provider", "test-model", client)
	require.NoError(t, err)
	assert.Greater(t, response.ResponseTime, int64(40)) // Should be at least 40ms
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsLower(s, substr))
}

func containsLower(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	// Also check case-insensitive
	for i := 0; i <= len(s)-len(substr); i++ {
		found := true
		for j := 0; j < len(substr); j++ {
			if toLower(s[i+j]) != toLower(substr[j]) {
				found = false
				break
			}
		}
		if found {
			return true
		}
	}
	return false
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}
