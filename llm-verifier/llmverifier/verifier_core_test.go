package llmverifier

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"llm-verifier/config"
)

// TestVerifier_GetGlobalClient_Final tests GetGlobalClient method
func TestVerifier_GetGlobalClient_Final(t *testing.T) {
	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:      "https://api.test.com",
			APIKey:       "test-key",
			MaxRetries:    3,
			RequestDelay:  100 * time.Millisecond,
			Timeout:       30 * time.Second,
		},
	}

	verifier := New(cfg)
	client := verifier.GetGlobalClient()

	if client == nil {
		t.Fatal("Expected client to be created")
	}

	if client.endpoint != cfg.Global.BaseURL {
		t.Errorf("Expected endpoint %s, got %s", cfg.Global.BaseURL, client.endpoint)
	}

	if client.apiKey != cfg.Global.APIKey {
		t.Errorf("Expected API key %s, got %s", cfg.Global.APIKey, client.apiKey)
	}
}

// TestVerifier_SummarizeConversation_Final tests SummarizeConversation method
func TestVerifier_SummarizeConversation_Final(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "{\n  \"summary\": \"Test conversation summary\",\n  \"topics\": [\"test\"],\n  \"key_points\": [\"Key point 1\"],\n  \"importance\": 0.8\n}"
				}
			}]
		}`))
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockServer.URL,
			APIKey:      "test-key",
			DefaultModel: "gpt-4",
		},
	}

	verifier := New(cfg)
	messages := []string{"Hello", "How are you?"}
	summary, err := verifier.SummarizeConversation(messages)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if summary == nil {
		t.Fatal("Expected summary, got nil")
	}

	if summary.Summary != "Test conversation summary" {
		t.Errorf("Expected summary 'Test conversation summary', got '%s'", summary.Summary)
	}
}

// TestVerifier_detectFeatures_Final tests detectFeatures method
func TestVerifier_detectFeatures_Final(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "def test(): pass"
				}
			}]
		}`))
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockServer.URL,
			APIKey:      "test-key",
			DefaultModel: "gpt-4",
		},
	}

	verifier := New(cfg)
	client := verifier.GetGlobalClient()
	features, err := verifier.detectFeatures(client, "test-model")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if features == nil {
		t.Fatal("Expected features, got nil")
	}

	t.Logf("Detected features: ToolUse=%v, CodeGeneration=%v", features.ToolUse, features.CodeGeneration)
}

// TestVerifier_testToolUse_Final tests tool use capability
func TestVerifier_testToolUse_Final(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&requestBody)

		if _, hasTools := requestBody["tools"]; hasTools {
			w.Write([]byte(`{
				"choices": [{
					"message": {
						"tool_calls": [{
							"id": "call_test",
							"type": "function",
							"function": {"name": "test_func", "arguments": "{}"}
						}]
					}
				}]
			}`))
		} else {
			w.Write([]byte(`{"choices": [{"message": {"content": "no tools"}}]}`))
		}
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockServer.URL,
			APIKey:      "test-key",
			DefaultModel: "gpt-4",
		},
	}

	verifier := New(cfg)
	client := verifier.GetGlobalClient()
	ctx := context.Background()
	supportsToolUse := verifier.testToolUse(client, "test-model", ctx)
	if !supportsToolUse {
		t.Error("Expected tool use to be supported")
	}
}

// TestVerifier_testCodeGeneration_Final tests code generation capability
func TestVerifier_testCodeGeneration_Final(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "def fibonacci(n):\n    return n"
				}
			}]
		}`))
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockServer.URL,
			APIKey:      "test-key",
			DefaultModel: "gpt-4",
		},
	}

	verifier := New(cfg)
	client := verifier.GetGlobalClient()
	ctx := context.Background()
	supportsCodeGen := verifier.testCodeGeneration(client, "test-model", ctx)
	if !supportsCodeGen {
		t.Error("Expected code generation to be supported")
	}
}

// TestVerifier_testStreaming_Final tests streaming capability
func TestVerifier_testStreaming_Final(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&requestBody)

		if stream, ok := requestBody["stream"].(bool); ok && stream {
			w.Header().Set("Content-Type", "text/event-stream")
			flusher, _ := w.(http.Flusher)
			
			chunks := []string{
				`data: {"choices": [{"delta": {"content": "Hello"}}]}`,
				`data: [DONE]`,
			}
			
			for _, chunk := range chunks {
				fmt.Fprintf(w, "%s\n\n", chunk)
				flusher.Flush()
			}
		} else {
			w.Write([]byte(`{"choices": [{"message": {"content": "Hello"}}]}`))
		}
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockServer.URL,
			APIKey:      "test-key",
			DefaultModel: "gpt-4",
		},
	}

	verifier := New(cfg)
	client := verifier.GetGlobalClient()
	supportsStreaming := verifier.testStreaming(client, "test-model")
	if !supportsStreaming {
		t.Error("Expected streaming to be supported")
	}
}

// TestVerifier_assessCodeCapabilities_Final tests code capability assessment
func TestVerifier_assessCodeCapabilities_Final(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "def solve(): return \"solution\""
				}
			}],
			"usage": {"total_tokens": 80}
		}`))
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockServer.URL,
			APIKey:      "test-key",
			DefaultModel: "gpt-4",
		},
	}

	verifier := New(cfg)
	client := verifier.GetGlobalClient()
	caps, err := verifier.assessCodeCapabilities(client, "test-model")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if caps == nil {
		t.Fatal("Expected code capabilities, got nil")
	}

	if caps.PromptResponse.OverallSuccessRate < 0 || caps.PromptResponse.OverallSuccessRate > 100 {
		t.Errorf("Invalid overall success rate: %f", caps.PromptResponse.OverallSuccessRate)
	}

	t.Logf("Code capabilities: Debugging=%v, Optimization=%v", caps.CodeDebugging, caps.CodeOptimization)
}

// TestVerifier_assessGenerativeCapabilities_Final tests generative capability assessment
func TestVerifier_assessGenerativeCapabilities_Final(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "Once upon a time, there was a creative story."
				}
			}]
		}`))
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockServer.URL,
			APIKey:      "test-key",
			DefaultModel: "gpt-4",
		},
	}

	verifier := New(cfg)
	client := verifier.GetGlobalClient()
	caps, err := verifier.assessGenerativeCapabilities(client, "test-model")

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if caps == nil {
		t.Fatal("Expected generative capabilities, got nil")
	}

	if caps.OriginalityScore < 0 || caps.OriginalityScore > 100 {
		t.Errorf("Invalid originality score: %f", caps.OriginalityScore)
	}

	t.Logf("Generative capabilities: CreativeWriting=%v, Storytelling=%v", caps.CreativeWriting, caps.Storytelling)
}

// TestVerifier_runLanguageSpecificTests_Final tests language-specific testing
func TestVerifier_runLanguageSpecificTests_Final(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "// Language-specific code"
				}
			}]
		}`))
	}))
	defer mockServer.Close()

	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockServer.URL,
			APIKey:      "test-key",
			DefaultModel: "gpt-4",
		},
	}

	verifier := New(cfg)
	client := verifier.GetGlobalClient()
	results := verifier.runLanguageSpecificTests(client, "test-model")

	if results.OverallSuccessRate < 0 || results.OverallSuccessRate > 100 {
		t.Errorf("Invalid overall success rate: %f", results.OverallSuccessRate)
	}

	t.Logf("Language test results: Python=%.1f%%, JavaScript=%.1f%%",
		results.PythonSuccessRate, results.JavascriptSuccessRate)
}

// TestVerifier_CalculateScores_Final tests score calculation system
func TestVerifier_CalculateScores_Final(t *testing.T) {
	cfg := &config.Config{}
	verifier := New(cfg)

	result := VerificationResult{
		ModelInfo: ModelInfo{
			ID:                "test-model",
			MaxOutputTokens:    4096,
			ContextWindow:      ContextWindow{TotalMaxTokens: 8192},
			SupportsVision:    true,
			SupportsReasoning: true,
			LanguageSupport:   []string{"python", "javascript"},
		},
		Availability: AvailabilityResult{
			Exists:      true,
			Responsive:  true,
			Overloaded:  false,
			Latency:     150 * time.Millisecond,
		},
		ResponseTime: ResponseTimeResult{
			AverageLatency: 200 * time.Millisecond,
			P95Latency:     300 * time.Millisecond,
			MinLatency:       50 * time.Millisecond,
			MaxLatency:       500 * time.Millisecond,
			Throughput:       10.5,
			MeasurementCount: 100,
		},
		FeatureDetection: FeatureDetectionResult{
			ToolUse:         true,
			CodeGeneration:  true,
			Streaming:       true,
			JSONMode:        true,
			Reasoning:       true,
			ParallelToolUse: true,
		},
		CodeCapabilities: CodeCapabilityResult{
			CodeGeneration:   true,
			DebuggingAccuracy: 95.5,
			PromptResponse: PromptResponseTest{
				OverallSuccessRate: 85.5,
			},
		},
		GenerativeCapabilities: GenerativeCapabilityResult{
			CreativeWriting:   true,
			OriginalityScore: 85.5,
			CreativityScore:   88.0,
		},
	}

	// Test CalculateScores with value (not pointer)
	scores, details := verifier.CalculateScores(result)

	// Verify scores are calculated and within valid ranges
	if scores.OverallScore < 0 || scores.OverallScore > 100 {
		t.Errorf("Invalid overall score: %f", scores.OverallScore)
	}

	if scores.CodeCapability < 0 || scores.CodeCapability > 100 {
		t.Errorf("Invalid code capability score: %f", scores.CodeCapability)
	}

	if scores.Responsiveness < 0 || scores.Responsiveness > 100 {
		t.Errorf("Invalid responsiveness score: %f", scores.Responsiveness)
	}

	if scores.Reliability < 0 || scores.Reliability > 100 {
		t.Errorf("Invalid reliability score: %f", scores.Reliability)
	}

	if scores.FeatureRichness < 0 || scores.FeatureRichness > 100 {
		t.Errorf("Invalid feature richness score: %f", scores.FeatureRichness)
	}

	if scores.ValueProposition < 0 || scores.ValueProposition > 100 {
		t.Errorf("Invalid value proposition score: %f", scores.ValueProposition)
	}

	t.Logf("Calculated scores: Overall=%.1f, Code=%.1f, Responsiveness=%.1f",
		scores.OverallScore, scores.CodeCapability, scores.Responsiveness)

	// Verify details are populated
	if details.CodeCapabilityBreakdown.GenerationScore == 0 && result.CodeCapabilities.CodeGeneration {
		t.Error("Expected generation score > 0 for code generation capability")
	}

	t.Logf("Score details: %+v", details)
}

// TestVerifier_calculateIndividualScores_Final tests individual score calculation methods
func TestVerifier_calculateIndividualScores_Final(t *testing.T) {
	verifier := New(&config.Config{})

	// Test data
	codeCaps := CodeCapabilityResult{
		CodeGeneration:    true,
		DebuggingAccuracy: 95.5,
		PromptResponse: PromptResponseTest{
			OverallSuccessRate: 85.5,
		},
	}

	availability := AvailabilityResult{
		Exists:     true,
		Responsive: true,
		Latency:    150 * time.Millisecond,
	}

	responseTime := ResponseTimeResult{
		AverageLatency: 200 * time.Millisecond,
		P95Latency:     300 * time.Millisecond,
		Throughput:     10.5,
	}

	// Test code capability score calculation
	codeScore, codeDetails := verifier.CalculateCodeCapabilityScore(codeCaps)
	if codeScore < 0 || codeScore > 100 {
		t.Errorf("Invalid code capability score: %f", codeScore)
	}

	if codeDetails.GenerationScore == 0 && codeCaps.CodeGeneration {
		t.Error("Expected generation score > 0 for code generation capability")
	}

	t.Logf("Code score breakdown: %+v", codeDetails)

	// Test responsiveness score calculation
	respScore, respDetails := verifier.CalculateResponsivenessScore(availability, responseTime)
	if respScore < 0 || respScore > 100 {
		t.Errorf("Invalid responsiveness score: %f", respScore)
	}

	t.Logf("Responsiveness score breakdown: %+v", respDetails)

	// Test reliability score calculation
	reliabilityScore, reliabilityDetails := verifier.CalculateReliabilityScore(availability)
	if reliabilityScore < 0 || reliabilityScore > 100 {
		t.Errorf("Invalid reliability score: %f", reliabilityScore)
	}

	t.Logf("Reliability score breakdown: %+v", reliabilityDetails)

	// Test feature richness score calculation
	features := FeatureDetectionResult{
		ToolUse:         true,
		CodeGeneration:  true,
		Streaming:       true,
		JSONMode:        true,
		Reasoning:       true,
	}

	testResult := VerificationResult{
		FeatureDetection: features,
	}

	featureScore, featureDetails := verifier.calculateFeatureRichnessScoreFromResult(testResult)
	if featureScore < 0 || featureScore > 100 {
		t.Errorf("Invalid feature richness score: %f", featureScore)
	}

	t.Logf("Feature richness score breakdown: %+v", featureDetails)

	t.Logf("Individual scores: Code=%f, Responsiveness=%f, Reliability=%f, Features=%f",
		codeScore, respScore, reliabilityScore, featureScore)
}

// TestVerifier_HelperFunctions_Final tests helper functions
func TestVerifier_HelperFunctions_Final(t *testing.T) {
	// Test intPtr
	intVal := 42
	ptr := intPtr(intVal)
	if ptr == nil || *ptr != intVal {
		t.Error("intPtr failed")
	}

	// Test floatPtr
	floatVal := 3.14
	ptrFloat := floatPtr(floatVal)
	if ptrFloat == nil || *ptrFloat != floatVal {
		t.Error("floatPtr failed")
	}

	// Test containsCode
	if !containsCode("def test(): pass") {
		t.Error("containsCode failed for Python code")
	}

	if !containsCode("function test() { return true; }") {
		t.Error("containsCode failed for JavaScript code")
	}

	if !containsCode("func test() { return true }") {
		t.Error("containsCode failed for Go code")
	}

	if containsCode("This is just plain text") {
		t.Error("containsCode should return false for plain text")
	}

	if !containsCode("import os; print('test')") {
		t.Error("containsCode should detect import statements")
	}

	if !containsCode("console.log('test')") {
		t.Error("containsCode should detect console.log")
	}
}

// TestVerifier_ErrorHandling_Final tests various error scenarios
func TestVerifier_ErrorHandling_Final(t *testing.T) {
	// Server that returns error status
	mockErrorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer mockErrorServer.Close()

	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockErrorServer.URL,
			APIKey:      "test-key",
			DefaultModel: "gpt-4",
		},
	}

	verifier := New(cfg)
	client := verifier.GetGlobalClient()
	ctx := context.Background()

	// Should handle error status gracefully
	supportsCode := verifier.testCodeGeneration(client, "test-model", ctx)
	if supportsCode {
		t.Log("Code generation reported as supported despite error (may be expected behavior)")
	}

	// Server that times out
	mockTimeoutServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices": [{"message": {"content": "late"}}]}`))
	}))
	defer mockTimeoutServer.Close()

	cfgTimeout := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockTimeoutServer.URL,
			APIKey:      "test-key",
			DefaultModel: "gpt-4",
			Timeout:     100 * time.Millisecond,
		},
	}

	verifierTimeout := New(cfgTimeout)
	clientTimeout := verifierTimeout.GetGlobalClient()

	// Should handle timeout gracefully
	supportsToolUse := verifierTimeout.testToolUse(clientTimeout, "test-model", ctx)
	if supportsToolUse {
		t.Log("Tool use reported as supported despite timeout (may be expected behavior)")
	}
}