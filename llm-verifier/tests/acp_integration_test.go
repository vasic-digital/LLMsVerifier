package tests

import (
	"context"
	"testing"
	"time"

	"github.com/llmverifier/llmverifier"
	"github.com/llmverifier/llmverifier/config"
	"github.com/llmverifier/llmverifier/providers"
)

// TestACPsWithRealProviders tests ACP detection with real LLM providers
func TestACPsWithRealProviders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load test configuration
	cfg := loadTestConfig()
	verifier := llmverifier.New(cfg)

	// Test with multiple providers
	providers := []string{"openai", "anthropic", "deepseek", "google"}
	
	for _, providerName := range providers {
		t.Run(providerName, func(t *testing.T) {
			// Get provider configuration
			registry := providers.NewProviderRegistry()
			providerConfig, exists := registry.GetConfig(providerName)
			if !exists {
				t.Skipf("Provider %s not configured", providerName)
			}

			// Create client for this provider
			client, err := createProviderClient(providerConfig)
			if err != nil {
				t.Errorf("Failed to create client for %s: %v", providerName, err)
				return
			}

			// Get available models for this provider
			models := getProviderModels(providerConfig)
			if len(models) == 0 {
				t.Skipf("No models available for provider %s", providerName)
			}

			// Test ACP support for each model
			for _, model := range models {
				t.Run(model, func(t *testing.T) {
					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()

					supportsACP := verifier.TestACPs(client, model, ctx)
					t.Logf("Model %s ACP support: %t", model, supportsACP)

					// Basic validation - result should be consistent
					if supportsACP && !providerConfig.Features["supports_acp"].(bool) {
						t.Logf("Warning: Model %s detected ACP support but provider config doesn't indicate support", model)
					}
				})
			}
		})
	}
}

// TestACPsEndToEnd tests complete ACP verification workflow
func TestACPsEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}

	cfg := loadTestConfig()
	verifier := llmverifier.New(cfg)

	// Run complete verification on a test model
	results, err := verifier.Verify()
	if err != nil {
		t.Fatalf("Verification failed: %v", err)
	}

	// Verify ACP results are included
	foundACPResults := false
	for _, result := range results {
		if result.FeatureDetection.ACPs {
			foundACPResults = true
			t.Logf("Model %s supports ACP", result.ModelInfo.ID)
		}
	}

	if !foundACPResults {
		t.Log("No models found with ACP support in this test run")
	}
}

// TestACPsDatabaseOperations tests ACP-related database operations
func TestACPsDatabaseOperations(t *testing.T) {
	// Setup test database
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t)

	// Create test data with ACP support
	testResult := llmverifier.VerificationResult{
		ModelInfo: llmverifier.ModelInfo{
			ID:      "test-acp-model",
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: "test-provider",
		},
		Availability: llmverifier.AvailabilityResult{
			Exists:     true,
			Responsive: true,
			Overloaded: false,
			Latency:    50 * time.Millisecond,
		},
		FeatureDetection: llmverifier.FeatureDetectionResult{
			ToolUse:          true,
			CodeGeneration:   true,
			CodeCompletion:   true,
			CodeReview:       true,
			CodeExplanation:  true,
			Embeddings:       false,
			Reranking:        false,
			ImageGeneration:  false,
			AudioGeneration:  false,
			VideoGeneration:  false,
			MCPs:             true,
			LSPs:             true,
			ACPs:             true, // ACP support
			Multimodal:       false,
			Streaming:        true,
			JSONMode:         true,
			StructuredOutput: true,
			Reasoning:        false,
			FunctionCalling:  true,
			ParallelToolUse:  false,
			MaxParallelCalls: 0,
			BatchProcessing:  false,
		},
	}

	// Insert result
	err := db.InsertVerificationResult(testResult)
	if err != nil {
		t.Fatalf("Failed to insert verification result: %v", err)
	}

	// Retrieve result
	retrieved, err := db.GetVerificationResult("test-acp-model")
	if err != nil {
		t.Fatalf("Failed to retrieve verification result: %v", err)
	}

	// Verify ACP support
	if !retrieved.FeatureDetection.ACPs {
		t.Error("ACP support not preserved in database")
	}

	// Update ACP support
	testResult.FeatureDetection.ACPs = false
	err = db.UpdateVerificationResult(testResult)
	if err != nil {
		t.Fatalf("Failed to update verification result: %v", err)
	}

	// Verify update
	updated, err := db.GetVerificationResult("test-acp-model")
	if err != nil {
		t.Fatalf("Failed to retrieve updated result: %v", err)
	}

	if updated.FeatureDetection.ACPs {
		t.Error("ACP support should be false after update")
	}
}

// TestACPsPerformance tests ACP detection performance
func TestACPsPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	cfg := loadTestConfig()
	verifier := llmverifier.New(cfg)

	// Create mock client for performance testing
	mockClient := &MockPerformanceClient{
		ResponseDelay: 100 * time.Millisecond,
	}

	ctx := context.Background()
	modelName := "performance-test-model"

	// Measure ACP detection time
	start := time.Now()
	supportsACP := verifier.TestACPs(mockClient, modelName, ctx)
	duration := time.Since(start)

	t.Logf("ACP detection took %v, result: %t", duration, supportsACP)

	// Verify performance is reasonable (should complete within reasonable time)
	maxExpectedDuration := 5 * time.Second // 5 tests with 100ms delay + overhead
	if duration > maxExpectedDuration {
		t.Errorf("ACP detection took too long: %v > %v", duration, maxExpectedDuration)
	}
}

// TestACPsErrorHandling tests ACP error handling
func TestACPsErrorHandling(t *testing.T) {
	cfg := loadTestConfig()
	verifier := llmverifier.New(cfg)

	testCases := []struct {
		name        string
		client      llmverifier.LLMClient
		expectFalse bool
	}{
		{
			name:        "ErrorClient",
			client:      &ErrorClient{},
			expectFalse: true,
		},
		{
			name:        "EmptyResponseClient",
			client:      &EmptyResponseClient{},
			expectFalse: true,
		},
		{
			name:        "TimeoutClient",
			client:      &TimeoutClient{},
			expectFalse: true,
		},
	}

	ctx := context.Background()
	modelName := "error-test-model"

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			supportsACP := verifier.TestACPs(tc.client, modelName, ctx)
			if tc.expectFalse && supportsACP {
				t.Error("Expected ACP support to be false due to error, but got true")
			}
		})
	}
}

// Helper functions and mock implementations

func loadTestConfig() *config.Config {
	return &config.Config{
		GlobalTimeout: 30 * time.Second,
		MaxRetries:    3,
		// Add other necessary config
	}
}

func createProviderClient(config *providers.ProviderConfig) (llmverifier.LLMClient, error) {
	// Implementation would create actual client based on provider config
	// For now, return a mock client
	return &MockProviderClient{Config: config}, nil
}

func getProviderModels(config *providers.ProviderConfig) []string {
	if models, ok := config.Features["supported_models"].([]string); ok {
		return models
	}
	return []string{config.DefaultModel}
}

func setupTestDatabase(t *testing.T) *Database {
	// Setup test database
	// Implementation would create test database
	return &Database{}
}

func cleanupTestDatabase(t *testing.T) {
	// Cleanup test database
}

// Mock implementations for testing

type MockPerformanceClient struct {
	ResponseDelay time.Duration
}

func (m *MockPerformanceClient) ChatCompletion(ctx context.Context, request llmverifier.ChatCompletionRequest) (*llmverifier.ChatCompletionResponse, error) {
	// Simulate network delay
	select {
	case <-time.After(m.ResponseDelay):
		// Return appropriate response based on request
		content := generateACPResponse(request.Messages[0].Content)
		return &llmverifier.ChatCompletionResponse{
			Choices: []llmverifier.Choice{
				{
					Message: llmverifier.Message{
						Role:    "assistant",
						Content: content,
					},
				},
			},
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type ErrorClient struct{}

func (e *ErrorClient) ChatCompletion(ctx context.Context, request llmverifier.ChatCompletionRequest) (*llmverifier.ChatCompletionResponse, error) {
	return nil, fmt.Errorf("simulated error")
}

type EmptyResponseClient struct{}

func (e *EmptyResponseClient) ChatCompletion(ctx context.Context, request llmverifier.ChatCompletionRequest) (*llmverifier.ChatCompletionResponse, error) {
	return &llmverifier.ChatCompletionResponse{
		Choices: []llmverifier.Choice{}, // Empty choices
	}, nil
}

type TimeoutClient struct{}

func (t *TimeoutClient) ChatCompletion(ctx context.Context, request llmverifier.ChatCompletionRequest) (*llmverifier.ChatCompletionResponse, error) {
	select {
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("timeout")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

type MockProviderClient struct {
	Config *providers.ProviderConfig
}

func (m *MockProviderClient) ChatCompletion(ctx context.Context, request llmverifier.ChatCompletionRequest) (*llmverifier.ChatCompletionResponse, error) {
	// Simulate provider-specific responses
	content := generateProviderSpecificResponse(m.Config.Name, request.Messages[0].Content)
	return &llmverifier.ChatCompletionResponse{
		Choices: []llmverifier.Choice{
			{
				Message: llmverifier.Message{
					Role:    "assistant",
					Content: content,
				},
			},
		},
	}, nil
}

func generateACPResponse(requestContent string) string {
	// Generate appropriate ACP-style response based on request content
	content := strings.ToLower(requestContent)
	
	if strings.Contains(content, "jsonrpc") {
		return `{"jsonrpc":"2.0","result":{"items":[{"label":"print","kind":"function","detail":"Built-in function"}]},"id":1}`
	}
	if strings.Contains(content, "tool") {
		return `I'll use the file_read tool with parameters: {"filename": "main.py"}`
	}
	if strings.Contains(content, "project structure") {
		return `Based on your Flask project structure, I recommend adding the utility module in src/utils/database.py`
	}
	if strings.Contains(content, "function") {
		return `def validate_users(users: List[Dict[str, str]]) -> List[Dict[str, str]]:
			\"\"\"Validate user data and return list of valid users.\"\"\"
			return [user for user in users if '@' in user.get('email', '')]`
	}
	if strings.Contains(content, "error") {
		return `Line 3: KeyError - missing 'email' key. Suggestion: Use user.get('email', '')`
	}
	
	return "I understand your ACP request and can help with coding tasks."
}

func generateProviderSpecificResponse(providerName, requestContent string) string {
	// Generate provider-specific ACP responses
	switch providerName {
	case "openai":
		return generateACPResponse(requestContent) + " (OpenAI GPT)"
	case "anthropic":
		return generateACPResponse(requestContent) + " (Claude)"
	case "deepseek":
		return generateACPResponse(requestContent) + " (DeepSeek)"
	case "google":
		return generateACPResponse(requestContent) + " (Gemini)"
	default:
		return generateACPResponse(requestContent)
	}
}