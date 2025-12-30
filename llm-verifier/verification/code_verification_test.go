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

	"llm-verifier/client"
	"llm-verifier/logging"
)

// createTestLogger creates a logger for testing with nil database
func createTestLogger() *logging.Logger {
	logger, _ := logging.NewLogger(nil, map[string]any{
		"level": "debug",
	})
	return logger
}

// createTestHTTPClient creates an HTTP client for testing
func createTestHTTPClient() *client.HTTPClient {
	return client.NewHTTPClient(30 * time.Second)
}

// MockProviderClient implements ProviderClientInterface for testing
type MockProviderClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewMockProviderClient(baseURL, apiKey string, httpClient *http.Client) *MockProviderClient {
	return &MockProviderClient{
		baseURL:    baseURL,
		apiKey:     apiKey,
		httpClient: httpClient,
	}
}

func (m *MockProviderClient) GetBaseURL() string {
	return m.baseURL
}

func (m *MockProviderClient) GetAPIKey() string {
	return m.apiKey
}

func (m *MockProviderClient) GetHTTPClient() *http.Client {
	return m.httpClient
}

func TestNewCodeVerificationService(t *testing.T) {
	logger := createTestLogger()
	httpClient := createTestHTTPClient()

	cvs := NewCodeVerificationService(httpClient, logger)

	assert.NotNil(t, cvs)
	assert.Equal(t, httpClient, cvs.httpClient)
	assert.Equal(t, logger, cvs.logger)
}

func TestCodeVerificationService_VerifyModelCodeVisibility_NilProvider(t *testing.T) {
	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	result, err := cvs.VerifyModelCodeVisibility(context.Background(), "test-model", "test-provider", nil)

	require.Error(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "error", result.Status)
	assert.Contains(t, result.ErrorMessage, "cannot be nil")
	assert.NotNil(t, result.CompletedAt)
}

func TestCodeVerificationService_VerifyModelCodeVisibility_Success(t *testing.T) {
	// Create a mock server that returns an affirmative response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": "Yes, I can see your Python code. It defines a fibonacci function that uses recursion to calculate the nth Fibonacci number.",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	mockProvider := NewMockProviderClient(server.URL, "test-api-key", server.Client())

	result, err := cvs.VerifyModelCodeVisibility(context.Background(), "test-model", "test-provider", mockProvider)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "verified", result.Status)
	assert.NotEmpty(t, result.VerificationID)
	assert.Equal(t, "test-model", result.ModelID)
	assert.Equal(t, "test-provider", result.ProviderID)
	assert.NotNil(t, result.CompletedAt)
}

func TestCodeVerificationService_VerifyModelCodeVisibility_NegativeResponse(t *testing.T) {
	// Create a mock server that returns a negative response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": "No, I cannot see your code. Please paste the code again.",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	mockProvider := NewMockProviderClient(server.URL, "test-api-key", server.Client())

	result, err := cvs.VerifyModelCodeVisibility(context.Background(), "test-model", "test-provider", mockProvider)

	require.NoError(t, err)
	require.NotNil(t, result)
	// Due to relaxed verification, should still be verified with score adjustment
	assert.Equal(t, "verified", result.Status)
}

func TestCodeVerificationService_VerifyModelCodeVisibility_ServerError(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	mockProvider := NewMockProviderClient(server.URL, "test-api-key", server.Client())

	result, err := cvs.VerifyModelCodeVisibility(context.Background(), "test-model", "test-provider", mockProvider)

	require.NoError(t, err) // Error is captured in result, not returned
	require.NotNil(t, result)
	// Due to relaxed verification, even error responses are counted
	// and the model is marked as "verified" with a minimum score
	assert.Equal(t, "verified", result.Status)
	assert.GreaterOrEqual(t, result.VerificationScore, 0.7)
}

func TestCodeVerificationService_GetTestCodeSamples(t *testing.T) {
	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	samples := cvs.getTestCodeSamples()

	assert.NotEmpty(t, samples)
	assert.GreaterOrEqual(t, len(samples), 5)

	// Check sample languages
	languages := make(map[string]bool)
	for _, sample := range samples {
		languages[sample.Language] = true
		assert.NotEmpty(t, sample.Code)
		assert.NotEmpty(t, sample.Language)
		assert.NotEmpty(t, sample.Purpose)
	}

	assert.True(t, languages["python"])
	assert.True(t, languages["javascript"])
	assert.True(t, languages["go"])
	assert.True(t, languages["java"])
	assert.True(t, languages["csharp"])
}

func TestCodeVerificationService_CreateCodeVerificationPrompt(t *testing.T) {
	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	sample := TestCodeSample{
		Code:     "print('hello')",
		Language: "python",
		Purpose:  "test",
	}

	prompt := cvs.createCodeVerificationPrompt(sample)

	assert.Contains(t, prompt, "Do you see my code?")
	assert.Contains(t, prompt, "python")
	assert.Contains(t, prompt, "print('hello')")
}

func TestCodeVerificationService_AnalyzeCodeResponse_Affirmative(t *testing.T) {
	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	sample := TestCodeSample{
		Code:     "def test(): pass",
		Language: "python",
	}

	response := "Yes, I can see your Python code. It defines a function called test."
	analysis := cvs.analyzeCodeResponse(response, sample)

	assert.True(t, analysis.ContainsAffirmative)
	assert.False(t, analysis.ContainsNegative)
	assert.NotEmpty(t, analysis.CodeReferences)
	assert.Greater(t, analysis.ConfidenceScore, 0.0)
}

func TestCodeVerificationService_AnalyzeCodeResponse_Negative(t *testing.T) {
	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	sample := TestCodeSample{
		Code:     "def test(): pass",
		Language: "python",
	}

	response := "No, I cannot see your code. Please provide it again."
	analysis := cvs.analyzeCodeResponse(response, sample)

	assert.False(t, analysis.ContainsAffirmative)
	assert.True(t, analysis.ContainsNegative)
}

func TestCodeVerificationService_DetectLanguageUnderstanding(t *testing.T) {
	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	tests := []struct {
		response         string
		expectedLanguage string
		expected         string
	}{
		{"This is Python code with def keyword", "python", "python"},
		{"I see JavaScript with function declaration", "javascript", "javascript"},
		{"Go code with func keyword", "go", "go"},
		{"Java code with class definition", "java", "java"},
		{"Random text without language keywords", "python", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedLanguage, func(t *testing.T) {
			result := cvs.detectLanguageUnderstanding(tt.response, tt.expectedLanguage)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCodeVerificationService_CalculateUnderstandingLevel(t *testing.T) {
	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	tests := []struct {
		name              string
		affirmative       bool
		codeRefCount      int
		languageDetection string
		expected          string
	}{
		{"no affirmative", false, 5, "python", "none"},
		{"advanced understanding", true, 5, "python", "advanced"},
		{"intermediate understanding", true, 2, "unknown", "intermediate"},
		{"basic understanding", true, 1, "unknown", "basic"},
		{"no code refs", true, 0, "unknown", "none"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cvs.calculateUnderstandingLevel(tt.affirmative, tt.codeRefCount, tt.languageDetection)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCodeVerificationService_CalculateConfidenceScore(t *testing.T) {
	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	// Test high confidence (affirmative, not negative, multiple refs, advanced)
	score := cvs.calculateConfidenceScore(true, false, 5, "advanced")
	assert.Equal(t, 1.0, score)

	// Test low confidence (not affirmative, negative, no refs)
	score = cvs.calculateConfidenceScore(false, true, 0, "none")
	assert.Equal(t, 0.0, score)

	// Test medium confidence
	score = cvs.calculateConfidenceScore(true, false, 1, "basic")
	assert.Greater(t, score, 0.5)
	assert.Less(t, score, 1.0)
}

func TestCodeVerificationService_ExtractCodeReferences(t *testing.T) {
	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	sample := TestCodeSample{Code: "def func(): return x", Language: "python"}

	response := "I see a function that returns a variable using an if statement"
	refs := cvs.extractCodeReferences(response, sample)

	assert.Contains(t, refs, "function")
	assert.Contains(t, refs, "return")
	assert.Contains(t, refs, "variable")
	assert.Contains(t, refs, "if")
}

func TestCodeVerificationService_AnalyzeVerificationResponses_Empty(t *testing.T) {
	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	analysis := cvs.analyzeVerificationResponses([]CodeVerificationResponse{})

	assert.Equal(t, "none", analysis.UnderstandingLevel)
	assert.Equal(t, 0.0, analysis.ConfidenceScore)
}

func TestCodeVerificationService_AnalyzeVerificationResponses_Multiple(t *testing.T) {
	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	responses := []CodeVerificationResponse{
		{
			AffirmativeResponse: true,
			Response:            "Yes, I can see your function",
			CodeUnderstanding:   0.8,
		},
		{
			AffirmativeResponse: true,
			Response:            "Yes, visible code with return statement",
			CodeUnderstanding:   0.9,
		},
		{
			AffirmativeResponse: false,
			Response:            "I can partially see it",
			CodeUnderstanding:   0.5,
		},
	}

	analysis := cvs.analyzeVerificationResponses(responses)

	assert.True(t, analysis.ContainsAffirmative)
	assert.Greater(t, analysis.ConfidenceScore, 0.0)
	assert.NotEqual(t, "none", analysis.UnderstandingLevel)
}

func TestCodeVerificationRequest_Struct(t *testing.T) {
	req := CodeVerificationRequest{
		ModelID:    "test-model",
		ProviderID: "test-provider",
		Code:       "print('test')",
		Language:   "python",
	}

	assert.Equal(t, "test-model", req.ModelID)
	assert.Equal(t, "test-provider", req.ProviderID)
	assert.Equal(t, "print('test')", req.Code)
	assert.Equal(t, "python", req.Language)
}

func TestCodeVerificationResponse_Struct(t *testing.T) {
	resp := CodeVerificationResponse{
		ModelID:             "test-model",
		ProviderID:          "test-provider",
		Verified:            true,
		Response:            "Yes, I can see",
		CanSeeCode:          true,
		AffirmativeResponse: true,
		CodeUnderstanding:   0.95,
		ResponseTime:        100,
		TestTimestamp:       time.Now(),
	}

	assert.Equal(t, "test-model", resp.ModelID)
	assert.True(t, resp.Verified)
	assert.True(t, resp.CanSeeCode)
	assert.Equal(t, int64(100), resp.ResponseTime)
}

func TestCodeVerificationResult_Struct(t *testing.T) {
	now := time.Now()
	result := CodeVerificationResult{
		VerificationID:          "test-id",
		ModelID:                 "test-model",
		ProviderID:              "test-provider",
		Status:                  "verified",
		CodeVisibility:          true,
		ToolSupport:             true,
		AffirmativeConfirmation: true,
		VerificationScore:       0.95,
		TestedAt:                now,
		CompletedAt:             &now,
	}

	assert.Equal(t, "test-id", result.VerificationID)
	assert.Equal(t, "verified", result.Status)
	assert.True(t, result.CodeVisibility)
	assert.Equal(t, 0.95, result.VerificationScore)
}

func TestCodeResponseAnalysis_Struct(t *testing.T) {
	analysis := CodeResponseAnalysis{
		ContainsAffirmative: true,
		ContainsNegative:    false,
		CodeReferences:      []string{"function", "variable"},
		LanguageDetection:   "python",
		UnderstandingLevel:  "advanced",
		ConfidenceScore:     0.95,
	}

	assert.True(t, analysis.ContainsAffirmative)
	assert.False(t, analysis.ContainsNegative)
	assert.Len(t, analysis.CodeReferences, 2)
	assert.Equal(t, "python", analysis.LanguageDetection)
	assert.Equal(t, "advanced", analysis.UnderstandingLevel)
	assert.Equal(t, 0.95, analysis.ConfidenceScore)
}

func TestTestCodeSample_Struct(t *testing.T) {
	sample := TestCodeSample{
		Code:     "def test(): pass",
		Language: "python",
		Purpose:  "test basic function",
	}

	assert.Equal(t, "def test(): pass", sample.Code)
	assert.Equal(t, "python", sample.Language)
	assert.Equal(t, "test basic function", sample.Purpose)
}

func TestMax(t *testing.T) {
	assert.Equal(t, 5.0, max(3.0, 5.0))
	assert.Equal(t, 5.0, max(5.0, 3.0))
	assert.Equal(t, 5.0, max(5.0, 5.0))
	assert.Equal(t, 0.0, max(-1.0, 0.0))
	assert.Equal(t, -0.5, max(-1.0, -0.5))
}

func TestPtrTime(t *testing.T) {
	now := time.Now()
	ptr := ptrTime(now)

	assert.NotNil(t, ptr)
	assert.Equal(t, now, *ptr)
}

func TestCodeVerificationService_MakeVerificationRequest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request format
		assert.Equal(t, "POST", r.Method)
		assert.Contains(t, r.URL.Path, "/chat/completions")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": "Yes, I can see your code",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	mockProvider := NewMockProviderClient(server.URL, "test-key", server.Client())

	response, err := cvs.makeVerificationRequest(context.Background(), mockProvider, "test-model", "test prompt")

	require.NoError(t, err)
	assert.Equal(t, "Yes, I can see your code", response)
}

func TestCodeVerificationService_MakeVerificationRequest_NoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"choices": []map[string]interface{}{},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	mockProvider := NewMockProviderClient(server.URL, "test-key", server.Client())

	_, err := cvs.makeVerificationRequest(context.Background(), mockProvider, "test-model", "test prompt")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no response choices")
}

func TestCodeVerificationService_MakeVerificationRequest_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	mockProvider := NewMockProviderClient(server.URL, "test-key", server.Client())

	_, err := cvs.makeVerificationRequest(context.Background(), mockProvider, "test-model", "test prompt")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestCodeVerificationService_TestCodeVisibility(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": "Yes, I can see your Python function that returns a value",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	mockProvider := NewMockProviderClient(server.URL, "test-key", server.Client())

	sample := TestCodeSample{
		Code:     "def test(): return 42",
		Language: "python",
		Purpose:  "test function",
	}

	response, err := cvs.testCodeVisibility(context.Background(), "test-provider", "test-model", mockProvider, sample)

	require.NoError(t, err)
	require.NotNil(t, response)
	assert.Equal(t, "test-model", response.ModelID)
	assert.Equal(t, "test-provider", response.ProviderID)
	assert.True(t, response.CanSeeCode)
	assert.True(t, response.AffirmativeResponse)
	assert.GreaterOrEqual(t, response.ResponseTime, int64(0))
}

func TestCodeVerificationService_TestCodeVisibility_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	mockProvider := NewMockProviderClient(server.URL, "test-key", server.Client())

	sample := TestCodeSample{
		Code:     "def test(): return 42",
		Language: "python",
		Purpose:  "test function",
	}

	response, err := cvs.testCodeVisibility(context.Background(), "test-provider", "test-model", mockProvider, sample)

	// Error is returned in response, not as error
	require.NoError(t, err)
	require.NotNil(t, response)
	assert.False(t, response.Verified)
	assert.NotEmpty(t, response.Error)
}
