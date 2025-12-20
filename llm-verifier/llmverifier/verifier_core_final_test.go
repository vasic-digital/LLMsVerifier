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

// TestVerifier_CalculateFeatureRichnessScore_Comprehensive tests previously uncovered method
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

// TestVerifier_assessComplexityHandling_Advanced tests complexity handling
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
							"content": "def complex_algorithm(data):\n    return data"
						}
					}],
					"usage": {"total_tokens": 200}
				}`))
			} else {
				// Simple response
				w.Write([]byte(`{
					"choices": [{
						"message": {
							"content": "def simple_algorithm(data):\n    return data"
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
	ctx := context.Background()

	// Test complexity assessment
	complexity := verifier.assessComplexityHandling(client, "test-model", ctx)

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

	t.Logf("Complexity handling: MaxDepth=%d, Quality=%.1f, Logic=%.1f, Efficiency=%.1f",
		complexity.MaxHandledDepth, complexity.CodeQuality, complexity.LogicCorrectness,
		complexity.RuntimeEfficiency)
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

			var code string
			if strings.Contains(content, "Python") {
				code = "def python_function():\n    return \"Python code\""
			} else if strings.Contains(content, "JavaScript") {
				code = "function jsFunction() {\n    return \"JavaScript code\";\n}"
			} else if strings.Contains(content, "Go") {
				code = "func goFunction() string {\n    return \"Go code\"\n}"
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

	// Verify all language success rates are set and within valid ranges
	if results.PythonSuccessRate < 0 || results.PythonSuccessRate > 100 {
		t.Errorf("Invalid Python success rate: %f", results.PythonSuccessRate)
	}

	if results.JavascriptSuccessRate < 0 || results.JavascriptSuccessRate > 100 {
		t.Errorf("Invalid JavaScript success rate: %f", results.JavascriptSuccessRate)
	}

	if results.GoSuccessRate < 0 || results.GoSuccessRate > 100 {
		t.Errorf("Invalid Go success rate: %f", results.GoSuccessRate)
	}

	// Overall success rate should be calculated correctly
	if results.OverallSuccessRate < 0 || results.OverallSuccessRate > 100 {
		t.Errorf("Invalid overall success rate: %f", results.OverallSuccessRate)
	}

	t.Logf("Language test summary: Overall=%.1f%%, Python=%.1f%%, JavaScript=%.1f%%, Go=%.1f%%",
		results.OverallSuccessRate, results.PythonSuccessRate, results.JavascriptSuccessRate, results.GoSuccessRate)
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

	_ = New(cfgMinimal) // Test minimal config creation

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
		t.Logf("Expected error handling for minimal responses: %v", err)
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
	latency, responsive, status := verifierFast.checkResponsiveness(clientFast, "fast-model")
	if !responsive {
		t.Error("Expected fast model to be responsive")
	}

	if latency <= 0 {
		t.Error("Expected positive latency")
	}

	if status == "" {
		t.Error("Expected status message")
	}

	t.Logf("Fast model: Latency=%v, Responsive=%v, Status=%s", latency, responsive, status)

	t.Log("Edge cases testing completed successfully")
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
					{"id": "model-3", "object": "model"}
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
		},
		Concurrency: 2,
	}

	verifier := New(cfg)

	// Test verification with multiple models
	results, err := verifier.Verify()

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

// TestVerifier_Verify_NoModels tests with no models configured
func TestVerifier_Verify_NoModels(t *testing.T) {
	// Mock server for model discovery
	mockDiscoverServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if strings.Contains(r.URL.Path, "models") {
			// Return list of models
			w.Write([]byte(`{
				"object": "list",
				"data": [
					{"id": "discovered-model-1", "object": "model"},
					{"id": "discovered-model-2", "object": "model"}
				]
			}`))
		} else {
			// Chat completion response
			w.Write([]byte(`{
				"choices": [{
					"message": {
						"content": "Discovered model response"
					}
				}]
			}`))
		}
	}))
	defer mockDiscoverServer.Close()

	// Config with no LLMs specified (should discover models)
	cfgDiscover := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockDiscoverServer.URL,
			APIKey:      "discover-key",
			DefaultModel: "discover-model",
		},
	}

	verifierDiscover := New(cfgDiscover)

	// Test verification with model discovery
	results, err := verifierDiscover.Verify()

	if err != nil {
		t.Errorf("Expected no error with model discovery, got: %v", err)
	}

	if results == nil {
		t.Fatal("Expected results, got nil")
	}

	// Should have discovered models
	if len(results) == 0 {
		t.Error("Expected to discover at least one model")
	}

	t.Logf("Model discovery test: Results=%d", len(results))

	// Verify discovered results
	for i, result := range results {
		if result.ModelInfo.ID == "" {
			t.Errorf("Discovered result %d missing model ID", i)
		}

		t.Logf("Discovered model %d: ID=%s, Exists=%v",
			i, result.ModelInfo.ID, result.Availability.Exists)
	}
}

// TestVerifier_checkOverload_Advanced tests overload detection with various scenarios
func TestVerifier_checkOverload_Advanced(t *testing.T) {
	// Server that simulates various overload scenarios
	requestCount := 0

	mockOverloadServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// Simulate latency based on request count
		if requestCount > 10 {
			time.Sleep(500 * time.Millisecond) // Slow responses under load
		} else {
			time.Sleep(50 * time.Millisecond) // Fast responses normally
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices": [{"message": {"content": "overload test"}}]}`))
	}))
	defer mockOverloadServer.Close()

	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockOverloadServer.URL,
			APIKey:      "overload-key",
			DefaultModel: "overload-model",
		},
	}

	verifier := New(cfg)
	client := verifier.GetGlobalClient()

	// Test overload detection
	overloaded, avgLatency, throughput := verifier.checkOverload(client, "overload-model")

	if avgLatency < 0 {
		t.Error("Expected positive average latency")
	}

	if throughput < 0 {
		t.Error("Expected positive throughput")
	}

	t.Logf("Overload detection: Overloaded=%v, AvgLatency=%v, Throughput=%.2f req/s",
		overloaded, avgLatency, throughput)

	if requestCount < 10 {
		t.Errorf("Expected at least 10 requests for overload testing, got %d", requestCount)
	}
}

// TestVerifier_Verify_Integration_FullWorkflow tests complete verification workflow
func TestVerifier_Verify_Integration_FullWorkflow(t *testing.T) {
	// Create comprehensive mock server that handles all verification stages
	mockWorkflowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		var requestBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&requestBody)

		if strings.Contains(r.URL.Path, "models") {
			// Model discovery
			w.Write([]byte(`{
				"data": [{"id": "workflow-model", "object": "model"}]
			}`))
		} else if len(requestBody) > 0 {
			if _, hasTools := requestBody["tools"]; hasTools {
				// Tool use test
				w.Write([]byte(`{
					"choices": [{
						"message": {
							"tool_calls": [{
								"id": "call_workflow",
								"type": "function",
								"function": {"name": "workflow_func", "arguments": "{}"}
							}]
						}
					}]
				}`))
			} else {
				// Default code generation test
				w.Write([]byte(`{
					"choices": [{
						"message": {
							"content": "def workflow_function():\n    return \"Full workflow test\""
						}
					}]
				}`))
			}
		} else {
			// Default response
			w.Write([]byte(`{
				"choices": [{
					"message": {
						"content": "Default workflow response"
					}
				}]
			}`))
		}
	}))
	defer mockWorkflowServer.Close()

	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockWorkflowServer.URL,
			APIKey:      "workflow-key",
			DefaultModel: "workflow-model",
		},
		LLMs: []config.LLMConfig{
			{
				Name:     "WorkflowLLM",
				Endpoint: mockWorkflowServer.URL,
				APIKey:  "workflow-key",
				Model:    "workflow-model",
			},
		},
	}

	verifier := New(cfg)

	// Test full workflow verification
	results, err := verifier.Verify()

	if err != nil {
		t.Errorf("Expected no error in full workflow, got: %v", err)
	}

	if results == nil {
		t.Fatal("Expected results, got nil")
	}

	if len(results) == 0 {
		t.Error("Expected at least one result from full workflow")
	}

	// Verify result has comprehensive information
	result := results[0]

	// Model info should be populated
	if result.ModelInfo.ID == "" {
		t.Error("Expected model ID to be populated")
	}

	// Availability should be tested
	if result.Availability.Exists == false {
		t.Error("Expected model existence to be detected")
	}

	// Scores should be calculated
	if result.PerformanceScores.OverallScore == 0 {
		t.Error("Expected overall score to be calculated")
	}

	// Timestamp should be set
	if result.Timestamp.IsZero() {
		t.Error("Expected timestamp to be set")
	}

	featureCount := 0
	if result.FeatureDetection.ToolUse { featureCount++ }
	if result.FeatureDetection.CodeGeneration { featureCount++ }
	if result.FeatureDetection.Streaming { featureCount++ }

	t.Logf("Full workflow test: Model=%s, Features detected=%d, Overall score=%.1f",
		result.ModelInfo.ID, featureCount, result.PerformanceScores.OverallScore)

	if featureCount == 0 {
		t.Log("Feature detection completed with minimal features (acceptable)")
	}
}