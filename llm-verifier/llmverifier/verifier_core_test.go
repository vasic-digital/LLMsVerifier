package llmverifier

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"llm-verifier/config"
)

// TestVerifier_CalculateFeatureRichnessScore_Comprehensive tests feature richness scoring
func TestVerifier_CalculateFeatureRichnessScore_Comprehensive(t *testing.T) {
	verifier := New(&config.Config{})

	// Test with comprehensive feature set
	featuresFull := FeatureDetectionResult{
		ToolUse:          true,
		FunctionCalling:  true,
		CodeGeneration:   true,
		CodeCompletion:   true,
		CodeExplanation:  true,
		CodeReview:       true,
		Embeddings:       true,
		Reranking:        true,
		ImageGeneration:  true,
		AudioGeneration:  true,
		VideoGeneration:  true,
		MCPs:             true,
		LSPs:             true,
		Multimodal:       true,
		Streaming:        true,
		JSONMode:         true,
		StructuredOutput: true,
		Reasoning:        true,
		ParallelToolUse:  true,
		MaxParallelCalls: 5,
		BatchProcessing:  true,
	}

	resultFull := VerificationResult{
		FeatureDetection: featuresFull,
		GenerativeCapabilities: GenerativeCapabilityResult{
			CreativeWriting:        true,
			Storytelling:          true,
			ContentGeneration:     true,
			ArtisticCreativity:    true,
			ProblemSolving:       true,
			MultimodalGenerative: true,
			OriginalityScore:      95.0,
			CreativityScore:      90.0,
		},
	}

	score, breakdown := verifier.calculateFeatureRichnessScoreFromResult(resultFull)

	// Should be very high for comprehensive feature set
	if score < 80 {
		t.Errorf("Expected high feature richness score for comprehensive features, got %f", score)
	}

	// Verify breakdown
	if breakdown.CoreFeaturesScore == 0 {
		t.Error("Expected core features score > 0")
	}

	if breakdown.AdvancedFeaturesScore == 0 {
		t.Error("Expected advanced features score > 0")
	}

	t.Logf("Comprehensive features: Score=%.1f, Core=%.1f, Advanced=%.1f, Experimental=%.1f",
		score, breakdown.CoreFeaturesScore, breakdown.AdvancedFeaturesScore, breakdown.ExperimentalFeaturesScore)

	// Test with minimal feature set
	featuresMinimal := FeatureDetectionResult{
		ToolUse:        false,
		CodeGeneration: false,
		Streaming:      false,
	}

	resultMinimal := VerificationResult{
		FeatureDetection: featuresMinimal,
		GenerativeCapabilities: GenerativeCapabilityResult{
			CreativeWriting: false,
			OriginalityScore: 10.0,
			CreativityScore:   5.0,
		},
	}

	scoreMinimal, breakdownMinimal := verifier.calculateFeatureRichnessScoreFromResult(resultMinimal)

	// Should be low for minimal feature set
	if scoreMinimal > 40 {
		t.Errorf("Expected low feature richness score for minimal features, got %f", scoreMinimal)
	}

	t.Logf("Minimal features: Score=%.1f, Core=%.1f, Advanced=%.1f, Experimental=%.1f",
		scoreMinimal, breakdownMinimal.CoreFeaturesScore, breakdownMinimal.AdvancedFeaturesScore, breakdownMinimal.ExperimentalFeaturesScore)
}

// TestVerifier_assessComplexityHandling_Advanced tests complexity handling with various scenarios
func TestVerifier_assessComplexityHandling_Advanced(t *testing.T) {
	// Mock server that can handle different complexity levels
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&requestBody)

		content, _ := requestBody["messages"].([]interface{})
		if len(content) > 0 {
			msg := content[0].(map[string]interface{})
			prompt := msg["content"].(string)
			
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			if strings.Contains(prompt, "complex") {
				// Respond with complex solution
				w.Write([]byte(`{
					"choices": [{
						"message": {
							"content": "def complex_algorithm(data):\n    # Multi-step complex implementation\n    result = []\n    for item in data:\n        result.append(transform(item))\n    return optimize(result)"
						}
					}],
					"usage": {"total_tokens": 200}
				}`))
			} else if strings.Contains(prompt, "moderate") {
				// Respond with moderate complexity
				w.Write([]byte(`{
					"choices": [{
						"message": {
							"content": "def moderate_algorithm(data):\n    # Moderate implementation\n    return [item * 2 for item in data]"
						}
					}],
					"usage": {"total_tokens": 100}
				}`))
			} else {
				// Simple response
				w.Write([]byte(`{
					"choices": [{
						"message": {
							"content": "def simple_algorithm(data):\n    return sum(data)"
						}
					}],
					"usage": {"total_tokens": 50}
				}`))
			}
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

	// Test complexity assessment
	complexity := verifier.assessComplexityHandling(client, "test-model")

	if complexity == nil {
		t.Fatal("Expected complexity metrics, got nil")
	}

	// Verify complexity metrics are reasonable
	if complexity.MaxHandledDepth < 1 || complexity.MaxHandledDepth > 5 {
		t.Errorf("Invalid max handled depth: %d", complexity.MaxHandledDepth)
	}

	if complexity.CodeQuality < 0 || complexity.CodeQuality > 100 {
		t.Errorf("Invalid code quality score: %f", complexity.CodeQuality)
	}

	if complexity.LogicCorrectness < 0 || complexity.LogicCorrectness > 100 {
		t.Errorf("Invalid logic correctness score: %f", complexity.LogicCorrectness)
	}

	if complexity.RuntimeEfficiency < 0 || complexity.RuntimeEfficiency > 100 {
		t.Errorf("Invalid runtime efficiency score: %f", complexity.RuntimeEfficiency)
	}

	// Model should handle at least some complexity
	if complexity.MaxHandledDepth < 2 {
		t.Errorf("Model should handle at least moderate complexity, got depth %d", complexity.MaxHandledDepth)
	}

	t.Logf("Complexity handling: MaxDepth=%d, Quality=%.1f, Logic=%.1f, Efficiency=%.1f, MaxTokens=%d",
		complexity.MaxHandledDepth, complexity.CodeQuality, complexity.LogicCorrectness,
		complexity.RuntimeEfficiency, complexity.MaxTokens)
}

// TestVerifier_runLanguageSpecificTests_Comprehensive tests all supported languages
func TestVerifier_runLanguageSpecificTests_Comprehensive(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Analyze request to determine language-appropriate response
		var requestBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&requestBody)

		messages := requestBody["messages"].([]interface{})
		if len(messages) > 0 {
			msg := messages[0].(map[string]interface{})
			content := msg["content"].(string)

			var language string
			var code string

			if strings.Contains(content, "Python") {
				language = "python"
				code = "def python_function():\n    return \"Python code\""
			} else if strings.Contains(content, "JavaScript") {
				language = "javascript"
				code = "function jsFunction() {\n    return \"JavaScript code\";\n}"
			} else if strings.Contains(content, "Go") {
				language = "go"
				code = "func goFunction() string {\n    return \"Go code\"\n}"
			} else if strings.Contains(content, "Java") {
				language = "java"
				code = "public class JavaClass {\n    public String javaMethod() {\n        return \"Java code\";\n    }\n}"
			} else if strings.Contains(content, "C++") {
				language = "cpp"
				code = "#include <string>\nstd::string cppFunction() {\n    return \"C++ code\";\n}"
			} else if strings.Contains(content, "TypeScript") {
				language = "typescript"
				code = "function tsFunction(): string {\n    return \"TypeScript code\";\n}"
			} else {
				// Default fallback
				code = "// Default code response"
			}

			w.Write([]byte(fmt.Sprintf(`{
				"choices": [{
					"message": {
						"role": "assistant",
						"content": "%s"
					}
				}],
				"usage": {"total_tokens": 50}
			}`, code)))
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

	// Test language-specific capability assessment
	results := verifier.runLanguageSpecificTests(client, "test-model")

	if results == nil {
		t.Fatal("Expected language test results, got nil")
	}

	// Verify all language success rates are set and within valid ranges
	languages := []struct {
		name   string
		rate   float64
		field  *float64
	}{
		{"Python", results.PythonSuccessRate, &results.PythonSuccessRate},
		{"JavaScript", results.JavascriptSuccessRate, &results.JavascriptSuccessRate},
		{"Go", results.GoSuccessRate, &results.GoSuccessRate},
		{"Java", results.JavaSuccessRate, &results.JavaSuccessRate},
		{"C++", results.CppSuccessRate, &results.CppSuccessRate},
		{"TypeScript", results.TypescriptSuccessRate, &results.TypescriptSuccessRate},
	}

	for _, lang := range languages {
		if lang.rate < 0 || lang.rate > 100 {
			t.Errorf("Invalid %s success rate: %f", lang.name, lang.rate)
		}

		t.Logf("%s success rate: %.1f%%", lang.name, lang.rate)
	}

	// Overall success rate should be calculated correctly
	if results.OverallSuccessRate < 0 || results.OverallSuccessRate > 100 {
		t.Errorf("Invalid overall success rate: %f", results.OverallSuccessRate)
	}

	// Should have tested some languages
	totalLanguages := 0
	if results.PythonSuccessRate > 0 {
		totalLanguages++
	}
	if results.JavascriptSuccessRate > 0 {
		totalLanguages++
	}
	if results.GoSuccessRate > 0 {
		totalLanguages++
	}

	if totalLanguages == 0 {
		t.Error("Should have tested at least one language")
	}

	t.Logf("Language test summary: Overall=%.1f%%, Languages tested=%d, Avg response time=%v",
		results.OverallSuccessRate, totalLanguages, results.AvgResponseTime)
}

// TestVerifier_EdgeCases tests edge cases and boundary conditions
func TestVerifier_EdgeCases(t *testing.T) {
	// Test with minimal valid configuration
	cfgMinimal := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     "https://api.minimal.com",
			APIKey:      "minimal-key",
			DefaultModel: "minimal-model",
		},
	}

	verifierMinimal := New(cfgMinimal)
	client := verifierMinimal.GetGlobalClient()

	// Test with server that returns minimal responses
	mockMinimalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices": [{"message": {"content": "minimal"}}]}`))
	}))
	defer mockMinimalServer.Close()

	cfgMinimal.Global.BaseURL = mockMinimalServer.URL
	verifierEdge := New(cfgMinimal)
	clientEdge := verifierEdge.GetGlobalClient()

	// Test edge case handling
	features, err := verifierEdge.detectFeatures(clientEdge, "edge-model")
	if err != nil {
		t.Fatalf("Expected no error with minimal responses, got: %v", err)
	}

	if features == nil {
		t.Error("Expected features object even with minimal responses")
	}

	// Test with very rapid responses (fast model)
	mockFastServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices": [{"message": {"content": "fast response"}}]}`))
	}))
	defer mockFastServer.Close()

	cfgFast := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockFastServer.URL,
			APIKey:      "fast-key",
			DefaultModel: "fast-model",
		},
	}

	verifierFast := New(cfgFast)
	clientFast := verifierFast.GetGlobalClient()

	// Test responsiveness with fast model
	responsive := verifierFast.checkResponsiveness(clientFast, "fast-model")
	if !responsive {
		t.Error("Expected fast model to be responsive")
	}
}

// TestVerifier_ConcurrentVerification tests concurrent verification scenarios
func TestVerifier_ConcurrentVerification(t *testing.T) {
	// Mock server that can handle concurrent requests
	requestCount := 0
	mockConcurrentServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`{
			"choices": [{
				"message": {
					"role": "assistant",
					"content": "Response %d"
				}
			}],
			"usage": {"total_tokens": 50}
		}`, requestCount)))
	}))
	defer mockConcurrentServer.Close()

	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockConcurrentServer.URL,
			APIKey:      "concurrent-key",
			DefaultModel: "concurrent-model",
		},
		Concurrency: 3, // Allow concurrent processing
	}

	verifier := New(cfg)

	// Test concurrent verification
	ctx := context.Background()
	results, err := verifier.Verify(ctx, 2*time.Second)

	if err != nil {
		t.Errorf("Expected no error with concurrent verification, got: %v", err)
	}

	if results == nil {
		t.Fatal("Expected results, got nil")
	}

	// Verify all requests were processed
	if requestCount == 0 {
		t.Error("Expected at least one request to be processed")
	}

	t.Logf("Concurrent verification: Requests processed=%d, Results=%d",
		requestCount, len(results))
}

// TestVerifier_Verify_WithTimeout tests timeout handling
func TestVerifier_Verify_WithTimeout(t *testing.T) {
	// Server that responds slowly
	mockSlowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Slow response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices": [{"message": {"content": "slow response"}}]}`))
	}))
	defer mockSlowServer.Close()

	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockSlowServer.URL,
			APIKey:      "slow-key",
			DefaultModel: "slow-model",
		},
		Timeout: 1 * time.Second, // Short timeout
	}

	verifier := New(cfg)

	// Test verification with timeout
	ctx := context.Background()
	start := time.Now()
	results, err := verifier.Verify(ctx, 3*time.Second)
	duration := time.Since(start)

	// Should timeout gracefully
	if err == nil && duration < 2*time.Second {
		t.Error("Expected timeout or delay with slow server")
	}

	if duration > 5*time.Second {
		t.Error("Should have timed out much earlier")
	}

	t.Logf("Timeout test: Duration=%v, Error=%v", duration, err)
}

// TestVerifier_Verify_CancelContext tests context cancellation
func TestVerifier_Verify_CancelContext(t *testing.T) {
	// Server that responds slowly
	mockCancelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Slow response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices": [{"message": {"content": "cancel test"}}]}`))
	}))
	defer mockCancelServer.Close()

	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockCancelServer.URL,
			APIKey:      "cancel-key",
			DefaultModel: "cancel-model",
		},
	}

	verifier := New(cfg)

	// Create cancelable context
	ctx, cancel := context.WithCancel(context.Background())
	
	// Cancel after short delay
	go func() {
		time.Sleep(500 * time.Millisecond)
		cancel()
	}()

	// Test verification with cancellation
	start := time.Now()
	results, err := verifier.Verify(ctx, 3*time.Second)
	duration := time.Since(start)

	// Should be cancelled
	if err == nil {
		t.Error("Expected cancellation error")
	}

	if duration > 2*time.Second {
		t.Error("Should have been cancelled much earlier")
	}

	t.Logf("Cancel test: Duration=%v, Error=%v, Results=%d", duration, err, len(results))
}

// TestVerifier_Verify_LargeNumberOfModels tests with many model configurations
func TestVerifier_Verify_LargeNumberOfModels(t *testing.T) {
	// Mock server that responds with many models
	mockMultiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if strings.Contains(r.URL.Path, "models") {
			// Return list of models
			w.Write([]byte(`{
				"object": "list",
				"data": [
					{"id": "model-1", "object": "model"},
					{"id": "model-2", "object": "model"},
					{"id": "model-3", "object": "model"},
					{"id": "model-4", "object": "model"},
					{"id": "model-5", "object": "model"}
				]
			}`))
		} else {
			// Chat completion response
			w.Write([]byte(`{
				"choices": [{
					"message": {
						"content": "Model test response"
					}
				}]
			}`))
		}
	}))
	defer mockMultiServer.Close()

	// Create config with multiple LLM entries
	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockMultiServer.URL,
			APIKey:      "multi-key",
			DefaultModel: "model-1",
		},
		LLMs: []config.LLMConfig{
			{
				Name:     "MultiLLM-1",
				Endpoint: mockMultiServer.URL,
				APIKey:  "multi-key",
				Model:    "model-1",
			},
			{
				Name:     "MultiLLM-2",
				Endpoint: mockMultiServer.URL,
				APIKey:  "multi-key",
				Model:    "model-2",
			},
			{
				Name:     "MultiLLM-3",
				Endpoint: mockMultiServer.URL,
				APIKey:  "multi-key",
				Model:    "model-3",
			},
		},
		Concurrency: 2,
	}

	verifier := New(cfg)

	// Test verification with multiple models
	ctx := context.Background()
	results, err := verifier.Verify(ctx, 5*time.Second)

	if err != nil {
		t.Errorf("Expected no error with multiple models, got: %v", err)
	}

	if results == nil {
		t.Fatal("Expected results, got nil")
	}

	// Should have results for multiple models
	if len(results) < 2 {
		t.Errorf("Expected results for multiple models, got %d", len(results))
	}

	t.Logf("Multiple models test: Results=%d, Configured models=%d",
		len(results), len(cfg.LLMs))

	// Verify each result has basic fields
	for i, result := range results {
		if result.ModelInfo.ID == "" {
			t.Errorf("Result %d missing model ID", i)
		}

		if result.Timestamp.IsZero() {
			t.Errorf("Result %d missing timestamp", i)
		}

		t.Logf("Model %d: ID=%s, Responsive=%v",
			i, result.ModelInfo.ID, result.Availability.Responsive)
	}
}

// Helper function to count detected features
func countDetectedFeatures(features FeatureDetectionResult) int {
	count := 0
	if features.ToolUse { count++ }
	if features.FunctionCalling { count++ }
	if features.CodeGeneration { count++ }
	if features.CodeCompletion { count++ }
	if features.CodeExplanation { count++ }
	if features.CodeReview { count++ }
	if features.Embeddings { count++ }
	if features.Reranking { count++ }
	if features.ImageGeneration { count++ }
	if features.AudioGeneration { count++ }
	if features.VideoGeneration { count++ }
	if features.Multimodal { count++ }
	if features.Streaming { count++ }
	if features.JSONMode { count++ }
	if features.StructuredOutput { count++ }
	if features.Reasoning { count++ }
	if features.ParallelToolUse { count++ }
	if features.BatchProcessing { count++ }
	return count
}