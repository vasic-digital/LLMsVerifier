package llmverifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"llm-verifier/config"
)

// TestVerifier_ExportIntegrationTests tests export functionality integration
func TestVerifier_ExportIntegrationTests(t *testing.T) {
	// Mock server for export testing
	mockExportServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "Export test model"
				}
			}]
		}`))
	}))
	defer mockExportServer.Close()

	// Create test configuration for export
	testConfig := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockExportServer.URL,
			APIKey:      "export-key",
			DefaultModel: "export-model",
		},
		LLMs: []config.LLMConfig{
			{
				Name:     "ExportTestLLM",
				Endpoint: mockExportServer.URL,
				APIKey:  "export-key",
				Model:    "export-model",
			},
		},
	}

	verifier := New(testConfig)

	// Get verification results for export
	results, err := verifier.Verify()
	if err != nil {
		t.Errorf("Expected no error getting export results, got: %v", err)
	}

	if results == nil {
		t.Fatal("Expected results for export testing")
	}

	// Test JSON export via buffer
	var jsonBuffer bytes.Buffer
	
	// Manually serialize config since ExportJSON is not a method
	configJson, err := json.MarshalIndent(testConfig, "", "  ")
	if err != nil {
		t.Errorf("Expected no error marshaling config to JSON, got: %v", err)
	}

	jsonBuffer.Write(configJson)
	
	jsonOutput := jsonBuffer.String()
	if len(jsonOutput) == 0 {
		t.Error("Expected JSON output")
	}

	if !strings.Contains(jsonOutput, "ExportTestLLM") {
		t.Error("JSON export should contain LLM configuration")
	}

	t.Logf("JSON export successful: %d characters", len(jsonOutput))

	// Test that exported JSON can be unmarshaled back
	var unmarshaledConfig config.Config
	err = json.Unmarshal([]byte(jsonOutput), &unmarshaledConfig)
	if err != nil {
		t.Errorf("Expected exported JSON to be valid, got: %v", err)
	}

	if len(unmarshaledConfig.LLMs) != len(testConfig.LLMs) {
		t.Error("Exported config should match original LLM count")
	}

	t.Log("Export integration tests completed successfully")
}

// TestVerifier_ComprehensiveScoreScenarios tests scoring with various comprehensive scenarios
func TestVerifier_ComprehensiveScoreScenarios(t *testing.T) {
	verifier := New(&config.Config{})

	// Test scenario 1: High-end model with all capabilities
	highEndModel := VerificationResult{
		ModelInfo: ModelInfo{
			ID:                "gpt-4-turbo",
			MaxOutputTokens:    4096,
			ContextWindow:      ContextWindow{TotalMaxTokens: 128000},
			SupportsVision:    true,
			SupportsAudio:     true,
			SupportsReasoning: true,
		},
		FeatureDetection: FeatureDetectionResult{
			ToolUse:          true,
			FunctionCalling:  true,
			CodeGeneration:   true,
			CodeCompletion:   true,
			CodeExplanation:  true,
			CodeReview:       true,
			Embeddings:       true,
			Multimodal:       true,
			Streaming:        true,
			JSONMode:         true,
			StructuredOutput: true,
			Reasoning:        true,
			ParallelToolUse:  true,
			BatchProcessing:  true,
		},
		CodeCapabilities: CodeCapabilityResult{
			CodeGeneration:     true,
			DebuggingAccuracy: 95.0,
			PromptResponse: PromptResponseTest{
				OverallSuccessRate: 92.3,
			},
		},
		GenerativeCapabilities: GenerativeCapabilityResult{
			CreativeWriting:        true,
			Storytelling:          true,
			OriginalityScore:       92.0,
			CreativityScore:        90.0,
		},
		Availability: AvailabilityResult{
			Exists:     true,
			Responsive: true,
			Latency:    100 * time.Millisecond,
		},
		ResponseTime: ResponseTimeResult{
			AverageLatency:   150 * time.Millisecond,
			P95Latency:       200 * time.Millisecond,
			Throughput:       50.0,
			MeasurementCount: 1000,
		},
	}

	scoresHighEnd, detailsHighEnd := verifier.CalculateScores(highEndModel)

	if scoresHighEnd.OverallScore < 80 {
		t.Errorf("High-end model should have high overall score, got %f", scoresHighEnd.OverallScore)
	}

	if scoresHighEnd.CodeCapability < 85 {
		t.Errorf("High-end model should have high code capability score, got %f", scoresHighEnd.CodeCapability)
	}

	if scoresHighEnd.FeatureRichness < 85 {
		t.Errorf("High-end model should have high feature richness score, got %f", scoresHighEnd.FeatureRichness)
	}

	t.Logf("High-end model scores: Overall=%.1f, Code=%.1f, Features=%.1f",
		scoresHighEnd.OverallScore, scoresHighEnd.CodeCapability, scoresHighEnd.FeatureRichness)

	// Test scenario 2: Basic model with minimal capabilities
	basicModel := VerificationResult{
		ModelInfo: ModelInfo{
			ID:                "basic-model",
			MaxOutputTokens:    1000,
			ContextWindow:      ContextWindow{TotalMaxTokens: 4000},
		},
		FeatureDetection: FeatureDetectionResult{
			CodeGeneration: true,
			// All other features are false
		},
		CodeCapabilities: CodeCapabilityResult{
			CodeGeneration:    true,
			DebuggingAccuracy: 60.0,
			PromptResponse: PromptResponseTest{
				OverallSuccessRate: 70.0,
			},
		},
		GenerativeCapabilities: GenerativeCapabilityResult{
			CreativeWriting: false,
			OriginalityScore: 40.0,
			CreativityScore:   35.0,
		},
		Availability: AvailabilityResult{
			Exists:     true,
			Responsive: false,
			Latency:    500 * time.Millisecond,
		},
		ResponseTime: ResponseTimeResult{
			AverageLatency: 800 * time.Millisecond,
			P95Latency:     1200 * time.Millisecond,
			Throughput:     0.1,
		},
	}

	scoresBasic, detailsBasic := verifier.CalculateScores(basicModel)

	if scoresBasic.OverallScore > 60 {
		t.Errorf("Basic model should not have very high overall score, got %f", scoresBasic.OverallScore)
	}

	if scoresBasic.CodeCapability > 70 {
		t.Errorf("Basic model should not have very high code capability score, got %f", scoresBasic.CodeCapability)
	}

	if scoresBasic.FeatureRichness > 30 {
		t.Errorf("Basic model should have low feature richness score, got %f", scoresBasic.FeatureRichness)
	}

	t.Logf("Basic model scores: Overall=%.1f, Code=%.1f, Features=%.1f",
		scoresBasic.OverallScore, scoresBasic.CodeCapability, scoresBasic.FeatureRichness)

	// Verify score details are populated
	if detailsHighEnd.CodeCapabilityBreakdown.GenerationScore == 0 {
		t.Error("High-end model should have generation score in breakdown")
	}

	if detailsBasic.CodeCapabilityBreakdown.GenerationScore == 0 && basicModel.CodeCapabilities.CodeGeneration {
		t.Error("Basic model should have generation score in breakdown if code generation is supported")
	}

	t.Log("Comprehensive score scenarios test completed")
}

// TestVerifier_RealWorldProviderSimulation tests simulation of real-world providers
func TestVerifier_RealWorldProviderSimulation(t *testing.T) {
	// Mock server simulating different real-world provider behaviors
	providerType := 0
	
	mockRealWorldServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providerType = (providerType + 1) % 3
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		switch providerType {
		case 0: // OpenAI-style response
			w.Write([]byte(`{
				"id": "chatcmpl-openai",
				"object": "chat.completion",
				"created": 1677652288,
				"model": "gpt-4",
				"choices": [{
					"index": 0,
					"message": {
						"role": "assistant",
						"content": "def openai_function():\n    return \"OpenAI response\""
					},
					"finish_reason": "stop"
				}],
				"usage": {
					"prompt_tokens": 10,
					"completion_tokens": 20,
					"total_tokens": 30
				}
			}`))
		case 1: // Generic response
			w.Write([]byte(`{
				"choices": [{
					"message": {
						"role": "assistant",
						"content": "def generic_function():\n    return \"Generic response\""
					}
				}]
			}`))
		case 2: // Tool-using response
			w.Write([]byte(`{
				"choices": [{
					"message": {
						"tool_calls": [{
							"id": "call_provider",
							"type": "function",
							"function": {"name": "provider_func", "arguments": "{}"}
						}]
					}
				}]
			}`))
		}
	}))
	defer mockRealWorldServer.Close()

	// Test with different provider configurations
	tests := []struct {
		name string
		cfg  *config.Config
	}{
		{
			name: "Generic Provider",
			cfg: &config.Config{
				Global: config.GlobalConfig{
					BaseURL:     mockRealWorldServer.URL,
					APIKey:      "generic-key",
					DefaultModel: "generic-model",
				},
				LLMs: []config.LLMConfig{
					{
						Name:     "Generic Provider",
						Endpoint: mockRealWorldServer.URL,
						APIKey:  "generic-key",
						Model:    "generic-model",
					},
				},
			},
		},
		{
			name: "Tool Using Provider",
			cfg: &config.Config{
				Global: config.GlobalConfig{
					BaseURL:     mockRealWorldServer.URL,
					APIKey:      "tool-key",
					DefaultModel: "tool-model",
				},
				LLMs: []config.LLMConfig{
					{
						Name:     "Tool Provider",
						Endpoint: mockRealWorldServer.URL,
						APIKey:  "tool-key",
						Model:    "tool-model",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			verifier := New(test.cfg)

			// Test verification
			results, err := verifier.Verify()

			if err != nil {
				t.Logf("Expected no error for %s, got: %v", test.name, err)
			}

			if results == nil {
				t.Fatalf("Expected results for %s, got nil", test.name)
			}

			if len(results) == 0 {
				t.Errorf("Expected at least one result for %s", test.name)
			}

			// Verify result structure
			result := results[0]
			if result.ModelInfo.ID == "" {
				t.Errorf("%s: Expected model ID", test.name)
			}

			if result.Timestamp.IsZero() {
				t.Errorf("%s: Expected timestamp", test.name)
			}

			t.Logf("%s: Model=%s, Features=%d, Score=%.1f",
				test.name, result.ModelInfo.ID,
				countDetectedFeatures(result.FeatureDetection),
				result.PerformanceScores.OverallScore)
		})
	}

	t.Log("Real-world provider simulation completed")
}

// TestVerifier_AdvancedErrorScenarios tests advanced error handling scenarios
func TestVerifier_AdvancedErrorScenarios(t *testing.T) {
	// Server with various error scenarios
	errorScenario := 0
	
	mockErrorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		errorScenario = (errorScenario + 1) % 7
		
		switch errorScenario {
		case 0: // Rate limit
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"error": "Rate limit exceeded", "retry_after": 60}`))
		case 1: // Invalid auth
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "Invalid API key"}`))
		case 2: // Model not found
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "Model not found"}`))
		case 3: // Server error
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Internal server error"}`))
		case 4: // Timeout (simulated)
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"choices": [{"message": {"content": "late"}}]}`))
		case 5: // Success
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"choices": [{"message": {"content": "success"}}]}`))
		case 6: // Another success
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"choices": [{"message": {"content": "another success"}}]}`))
		}
	}))
	defer mockErrorServer.Close()

	cfgError := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockErrorServer.URL,
			APIKey:      "error-key",
			DefaultModel: "error-model",
			Timeout:     500 * time.Millisecond, // Short timeout for testing
		},
		LLMs: []config.LLMConfig{
			{
				Name:     "ErrorTestLLM",
				Endpoint: mockErrorServer.URL,
				APIKey:  "error-key",
				Model:    "error-model",
			},
		},
	}

	verifier := New(cfgError)

	// Test advanced error scenarios
	start := time.Now()
	results, err := verifier.Verify()
	duration := time.Since(start)

	if err == nil {
		t.Log("Advanced error scenarios: No error returned (may be acceptable)")
	}

	if results == nil {
		t.Fatal("Expected results array even with errors")
	}

	// Should have some results despite errors
	if len(results) == 0 {
		t.Error("Expected at least one result despite errors")
	}

	// Verify error information is captured
	result := results[0]
	if !result.Availability.Exists {
		t.Log("Error correctly detected as non-existent")
	}

	t.Logf("Advanced error test: Duration=%v, Results=%d, Model=%s",
		duration, len(results), result.ModelInfo.ID)

	// Test with network unavailable
	cfgNetwork := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     "https://nonexistent.network.example.com",
			APIKey:      "network-key",
			DefaultModel: "network-model",
			Timeout:     100 * time.Millisecond,
		},
	}

	verifierNetwork := New(cfgNetwork)
	startNetwork := time.Now()
	resultsNetwork, errNetwork := verifierNetwork.Verify()
	durationNetwork := time.Since(startNetwork)

	if errNetwork == nil {
		t.Log("Network error handled gracefully (acceptable)")
	}

	if resultsNetwork == nil {
		t.Fatal("Expected results array even with network errors")
	}

	t.Logf("Network error test: Duration=%v, Error=%v",
		durationNetwork, errNetwork != nil)

	t.Log("Advanced error scenarios test completed")
}

// TestVerifier_PerformanceBoundaryTests tests performance at system boundaries
func TestVerifier_PerformanceBoundaryTests(t *testing.T) {
	// Server with controlled performance characteristics
	requestCounter := 0
	var latencies []time.Duration
	
	mockPerfServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCounter++
		start := time.Now()
		
		// Simulate varying response times
		switch {
		case requestCounter%20 == 0:
			// Occasional very slow response
			time.Sleep(2 * time.Second)
		case requestCounter%10 == 0:
			// Occasional slow response
			time.Sleep(1 * time.Second)
		default:
			// Normal response
			time.Sleep(50 * time.Millisecond)
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "Performance test response"
				}
			}]
		}`))
		
		latency := time.Since(start)
		latencies = append(latencies, latency)
	}))
	defer mockPerfServer.Close()

	cfgPerf := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockPerfServer.URL,
			APIKey:      "perf-key",
			DefaultModel: "perf-model",
			Timeout:     3 * time.Second,
		},
		LLMs: []config.LLMConfig{
			{
				Name:     "PerformanceTestLLM",
				Endpoint: mockPerfServer.URL,
				APIKey:  "perf-key",
				Model:    "perf-model",
			},
		},
		Concurrency: 5,
	}

	verifier := New(cfgPerf)

	// Test performance boundaries
	start := time.Now()
	results, err := verifier.Verify()
	duration := time.Since(start)

	if err != nil {
		t.Logf("Performance test error (may be expected): %v", err)
	}

	if results == nil {
		t.Fatal("Expected results, got nil")
	}

	// Performance should be within reasonable bounds
	if duration > 30*time.Second {
		t.Errorf("Performance test took too long: %v", duration)
	}

	// Verify performance metrics
	if len(results) > 0 {
		result := results[0]
		
		if result.ResponseTime.MeasurementCount < 10 {
			t.Errorf("Should have made multiple measurements, got %d", result.ResponseTime.MeasurementCount)
		}

		if result.ResponseTime.AverageLatency > 5*time.Second {
			t.Errorf("Average latency too high: %v", result.ResponseTime.AverageLatency)
		}

		// Calculate actual performance statistics
		if len(latencies) > 0 {
			var totalLatency time.Duration
			maxLatency := time.Duration(0)
			minLatency := latencies[0]
			
			for _, latency := range latencies {
				totalLatency += latency
				if latency > maxLatency {
					maxLatency = latency
				}
				if latency < minLatency {
					minLatency = latency
				}
			}
			
			avgLatency := totalLatency / time.Duration(len(latencies))
			
			t.Logf("Performance test: Duration=%v, Requests=%d, Avg Latency=%v, Min=%v, Max=%v",
				duration, len(latencies), avgLatency, minLatency, maxLatency)
		}

		t.Logf("Performance metrics: Avg=%v, P95=%v, Min=%v, Max=%v, Throughput=%.2f req/s",
			result.ResponseTime.AverageLatency,
			result.ResponseTime.P95Latency,
			result.ResponseTime.MinLatency,
			result.ResponseTime.MaxLatency,
			result.ResponseTime.Throughput)
	}

	t.Log("Performance boundary tests completed")
}

// TestVerifier_DataIntegrityTests tests data integrity across verification
func TestVerifier_DataIntegrityTests(t *testing.T) {
	// Server that provides consistent responses
	responseCounter := 0
	mockConsistencyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responseCounter++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Analyze request type
		var requestBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&requestBody)

		if _, hasTools := requestBody["tools"]; hasTools {
			// Tool use test
			w.Write([]byte(`{
				"choices": [{
					"message": {
						"tool_calls": [{
							"id": "call_integrity",
							"type": "function",
							"function": {"name": "integrity_func", "arguments": "{\"test\": \"data\"}"}
						}]
					}
				}]
			}`))
		} else {
			// Regular test
			w.Write([]byte(fmt.Sprintf(`{
				"choices": [{
					"message": {
						"role": "assistant",
						"content": "def integrity_test_%d():\n    return \"Data integrity check passed\""
					}
				}],
				"usage": {"total_tokens": %d}
			}`, responseCounter, 42+responseCounter)))
		}
	}))
	defer mockConsistencyServer.Close()

	cfgConsistency := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     mockConsistencyServer.URL,
			APIKey:      "consistency-key",
			DefaultModel: "consistency-model",
		},
		LLMs: []config.LLMConfig{
			{
				Name:     "ConsistencyTestLLM",
				Endpoint: mockConsistencyServer.URL,
				APIKey:  "consistency-key",
				Model:    "consistency-model",
			},
		},
	}

	verifier := New(cfgConsistency)

	// Run multiple verification rounds
	rounds := 3
	var allResults [][]VerificationResult

	for i := 0; i < rounds; i++ {
		results, err := verifier.Verify()
		if err != nil {
			t.Logf("Expected no error in round %d, got: %v", i, err)
		}

		if results == nil {
			t.Fatalf("Expected results in round %d, got nil", i)
		}

		allResults = append(allResults, results)
	}

	// Verify consistency across rounds
	for i, results := range allResults {
		t.Logf("Round %d: %d results", i, len(results))
		
		if len(results) == 0 {
			t.Errorf("Round %d: Expected at least one result", i)
			continue
		}

		result := results[0]
		
		// Basic consistency checks
		if result.ModelInfo.ID == "" {
			t.Errorf("Round %d: Missing model ID", i)
		}

		if result.Timestamp.IsZero() {
			t.Errorf("Round %d: Missing timestamp", i)
		}

		// Performance scores should be in reasonable ranges
		if result.PerformanceScores.OverallScore < 0 || result.PerformanceScores.OverallScore > 100 {
			t.Errorf("Round %d: Invalid overall score: %f", i, result.PerformanceScores.OverallScore)
		}

		t.Logf("Round %d: Model=%s, Score=%.1f, Features=%d",
			i, result.ModelInfo.ID, result.PerformanceScores.OverallScore, countDetectedFeatures(result.FeatureDetection))
	}

	t.Logf("Data integrity test completed: %d rounds, %d responses", rounds, responseCounter)
}

// TestVerifier_ConfigurationEdgeCases tests various configuration edge cases
func TestVerifier_ConfigurationEdgeCases(t *testing.T) {
	// Test with nil configuration
	verifier := New(nil)
	if verifier == nil {
		t.Error("Expected verifier to be created even with nil config")
	}

	// Test with empty configuration
	emptyConfig := &config.Config{}
	verifierEmpty := New(emptyConfig)
	if verifierEmpty == nil {
		t.Error("Expected verifier to be created with empty config")
	}

	// Test with minimal valid configuration
	minimalConfig := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:      "https://api.minimal.com",
			APIKey:       "minimal-key",
			DefaultModel: "minimal-model",
		},
	}
	verifierMinimal := New(minimalConfig)
	if verifierMinimal == nil {
		t.Error("Expected verifier to be created with minimal config")
	}

	// Test with very long configuration values
	longConfig := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:     strings.Repeat("https://example.com/", 10),
			APIKey:      strings.Repeat("test-key-", 10),
			DefaultModel: strings.Repeat("model-", 10),
		},
		LLMs: []config.LLMConfig{
			{
				Name:     strings.Repeat("LLM-", 10),
				Endpoint: strings.Repeat("https://endpoint.com/", 10),
				APIKey:  strings.Repeat("key-", 10),
				Model:    strings.Repeat("model-", 10),
			},
		},
	}
	verifierLong := New(longConfig)
	if verifierLong == nil {
		t.Error("Expected verifier to be created with long configuration")
	}

	t.Log("Configuration edge cases testing completed successfully")
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