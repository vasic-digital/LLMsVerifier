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

	"digital.vasic.llmsverifier/client"
	"digital.vasic.llmsverifier/logging"
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
	// HXV-status-bluff reconciliation (§11.4.120): the previous assertion
	// expected Status == "verified" for a model that answers "No, I cannot see
	// your code" to every sample — that certified a code-blind model as
	// verified (a §11.4 / §11.4.1 PASS-bluff in the source-of-truth gate). A
	// model that does NOT establish code visibility MUST be "failed", never
	// "verified".
	assert.False(t, result.CodeVisibility,
		"a model that denies code visibility must have CodeVisibility=false")
	assert.NotEqual(t, "verified", result.Status,
		"a model that cannot see the code must not be certified verified")
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

	require.NoError(t, err) // top-level error nil; outcome captured in result.Status
	require.NotNil(t, result)
	// Honest contract (HXV-002, round-348): a provider that returns
	// HTTP 500 for every code sample yields zero successful
	// verification responses, so the model is NOT verified — Status
	// MUST be "failed". The previous assertion expected "verified"
	// with score >= 0.7, which certified the "relaxed verification"
	// bluff: a model that was never successfully exercised was still
	// reported as verified to consumers (a CONST-036/037 single-
	// source-of-truth lie). A server error must never produce a
	// passing verification.
	assert.Equal(t, "failed", result.Status,
		"a model whose provider returns HTTP 500 for every sample must not be verified")
	assert.NotEmpty(t, result.ErrorMessage,
		"the failure cause must be recorded so consumers see why verification did not pass")
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

// TestCodeVerificationService_AnalyzeCodeResponse_WordBoundaries pins the
// word-boundary keyword matching: the bare keyword "no" must NOT match inside
// "know"/"not"/"now", and "can see" must NOT match across the "cannot"
// boundary. Bare substring matching misclassified all of these.
func TestCodeVerificationService_AnalyzeCodeResponse_WordBoundaries(t *testing.T) {
	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	sample := TestCodeSample{Code: "def test(): pass", Language: "python"}

	tests := []struct {
		name            string
		response        string
		wantAffirmative bool
		wantNegative    bool
	}{
		{"plain affirmative", "Yes, I can see your code", true, false},
		{"know is not no", "I know the answer", false, false},
		{"now is not no", "Yes, I see it now", true, false},
		{"note is not no", "Yes, I note the code", true, false},
		{"cannot see is negative, not affirmative", "I cannot see it", false, true},
		{"explicit no is negative", "No, I cannot see your code", false, true},
		{"not visible is negative", "The code is not visible to me", true, true},
		{"do not see is negative", "I do not see any code here", false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analysis := cvs.analyzeCodeResponse(tt.response, sample)
			assert.Equal(t, tt.wantAffirmative, analysis.ContainsAffirmative,
				"ContainsAffirmative for %q", tt.response)
			assert.Equal(t, tt.wantNegative, analysis.ContainsNegative,
				"ContainsNegative for %q", tt.response)
		})
	}
}

// TestCodeVerificationService_AnalyzeCodeResponse_BareAffirmativeBelowThreshold
// pins the strict-verification calibration: a bare "Yes, I can see it" with
// zero code-specific content must NOT reach the 0.5 confidence threshold on
// its own — otherwise a bluffing model passes the gate by parroting the
// requested confirmation phrase.
func TestCodeVerificationService_AnalyzeCodeResponse_BareAffirmativeBelowThreshold(t *testing.T) {
	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	sample := TestCodeSample{Code: "def test(): pass", Language: "python"}

	analysis := cvs.analyzeCodeResponse("Yes, I can see it.", sample)

	assert.True(t, analysis.ContainsAffirmative)
	assert.False(t, analysis.ContainsNegative)
	assert.Empty(t, analysis.CodeReferences)
	assert.Less(t, analysis.ConfidenceScore, 0.5,
		"a bare affirmative with no code-specific content must stay below the 0.5 strict threshold")
}

// TestCodeVerificationService_AnalyzeCodeResponse_CodeSpecificContentAboveThreshold
// pins the positive side of the calibration: an affirmative answer that
// references code elements and the language crosses the 0.5 threshold.
func TestCodeVerificationService_AnalyzeCodeResponse_CodeSpecificContentAboveThreshold(t *testing.T) {
	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	sample := TestCodeSample{Code: "def test(): pass", Language: "python"}

	analysis := cvs.analyzeCodeResponse(
		"Yes, I can see your Python code. It defines a function called test.", sample)

	assert.True(t, analysis.ContainsAffirmative)
	assert.False(t, analysis.ContainsNegative)
	assert.NotEmpty(t, analysis.CodeReferences)
	assert.Greater(t, analysis.ConfidenceScore, 0.5,
		"an affirmative with code-specific content must cross the 0.5 strict threshold")
}

// TestCodeVerificationService_CalculateConfidenceScore_BareAffirmative pins the
// weight calibration at the unit level: affirmative + not-negative alone = 0.4.
func TestCodeVerificationService_CalculateConfidenceScore_BareAffirmative(t *testing.T) {
	logger := createTestLogger()
	httpClient := createTestHTTPClient()
	cvs := NewCodeVerificationService(httpClient, logger)

	score := cvs.calculateConfidenceScore(true, false, 0, "none")
	assert.InDelta(t, 0.4, score, 1e-9,
		"affirmative + not-negative with no code evidence must be exactly 0.4 (< 0.5 threshold)")
}

// TestCodeVerificationService_VerifyModelCodeVisibility_BareAffirmativeFails is
// the end-to-end strict-gate test: a model that answers every sample with a
// bare "Yes, I can see it." (zero code-specific content) must NOT be verified,
// and the reported score must be the real computed score (no 0.7 floor).
func TestCodeVerificationService_VerifyModelCodeVisibility_BareAffirmativeFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]string{
						"content": "Yes, I can see it.",
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

	result, err := cvs.VerifyModelCodeVisibility(context.Background(), "bluff-model", "test-provider", mockProvider)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.CodeVisibility,
		"a bare affirmative with no code-specific content must not establish code visibility")
	assert.Equal(t, "failed", result.Status,
		"a model that only parrots the confirmation phrase must fail strict verification")
	assert.Less(t, result.VerificationScore, 0.5,
		"the real computed score must be reported (below the 0.5 threshold), with no floor")
	assert.Equal(t, result.ResponseAnalysis.ConfidenceScore, result.VerificationScore,
		"VerificationScore must equal the real ConfidenceScore (no hard score floor)")
}

// TestCodeVerificationService_VerifyModelCodeVisibility_RealScoreReported pins
// the floor removal on the positive path: a genuinely confident model reports
// its real computed score, which here is BELOW the old 0.7 hard floor — if the
// floor still existed, VerificationScore would read 0.7 instead.
func TestCodeVerificationService_VerifyModelCodeVisibility_RealScoreReported(t *testing.T) {
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
	assert.True(t, result.CodeVisibility)
	assert.Greater(t, result.VerificationScore, 0.5)
	assert.Equal(t, result.ResponseAnalysis.ConfidenceScore, result.VerificationScore,
		"VerificationScore must be the real computed score, not max(score, 0.7)")
	assert.NotEqual(t, 0.7, result.VerificationScore,
		"the old 0.7 hard floor must be gone")
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

	// Honest contract (HXV-002, round-348): an API failure (HTTP 503)
	// is propagated as a real error so the caller can distinguish an
	// API failure from a genuine negative verification. The previous
	// assertion expected require.NoError, which certified the bluff
	// where API failures were silently swallowed and a model could
	// still be scored as if it had been tested. The response is still
	// returned alongside the error with Verified=false and Error set.
	require.Error(t, err, "API failure must be propagated, never swallowed")
	assert.Contains(t, err.Error(), "503", "propagated error must name the real HTTP status")
	require.NotNil(t, response, "response must still be returned alongside the error")
	assert.False(t, response.Verified, "an API failure must never count as verified")
	assert.NotEmpty(t, response.Error, "the failure cause must be recorded in the response")
}
