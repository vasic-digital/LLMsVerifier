package tests

import (
	"context"
	"testing"
	"time"

	"github.com/llmverifier/llmverifier"
	"github.com/llmverifier/llmverifier/config"
)

// TestACPsCapabilityDetection tests the ACP capability detection function
func TestACPsCapabilityDetection(t *testing.T) {
	// Create a mock verifier instance
	cfg := &config.Config{
		GlobalTimeout: 30 * time.Second,
	}
	verifier := llmverifier.New(cfg)

	// Test with a mock client that simulates ACP-supported responses
	mockClient := &MockLLMClient{
		Responses: map[string]string{
			"jsonrpc": `{"jsonrpc":"2.0","result":{"items":[{"label":"print","kind":"function"}]},"id":1}`,
			"tool":    `I'll use the file_read tool to analyze the Python file.`,
			"context": `Based on your project structure, I recommend adding the utility module in src/utils/database.py and importing it as: from src.utils.database import DatabaseHelper`,
			"code":    `def validate_users(users: List[Dict[str, str]]) -> List[Dict[str, str]]:
				"""Validate user data and return list of valid users."""
				valid_users = []
				for user in users:
					if '@' in user.get('email', ''):
						valid_users.append(user)
				return valid_users`,
			"error":   `Line 3: KeyError - missing 'email' key in user dictionary. Suggestion: Use user.get('email', '') with default value.`,
		},
	}

	ctx := context.Background()
	modelName := "test-model"

	// Test ACP detection
	supportsACP := verifier.TestACPs(mockClient, modelName, ctx)
	
	if !supportsACP {
		t.Error("Expected ACP support to be detected, but got false")
	}
}

// TestACPsScoringIntegration tests ACP integration in the scoring system
func TestACPsScoringIntegration(t *testing.T) {
	// Create test feature detection results
	featuresWithACP := llmverifier.FeatureDetectionResult{
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
		ACPs:             true, // ACP support enabled
		Multimodal:       false,
		Streaming:        true,
		JSONMode:         true,
		StructuredOutput: true,
		Reasoning:        false,
		FunctionCalling:  true,
		ParallelToolUse:  false,
		MaxParallelCalls: 0,
		Modalities:       []string{"text"},
		BatchProcessing:  false,
	}

	featuresWithoutACP := llmverifier.FeatureDetectionResult{
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
		ACPs:             false, // ACP support disabled
		Multimodal:       false,
		Streaming:        true,
		JSONMode:         true,
		StructuredOutput: true,
		Reasoning:        false,
		FunctionCalling:  true,
		ParallelToolUse:  false,
		MaxParallelCalls: 0,
		Modalities:       []string{"text"},
		BatchProcessing:  false,
	}

	cfg := &config.Config{}
	verifier := llmverifier.New(cfg)

	// Test scoring with ACP support
	scoreWithACP, breakdownWithACP := verifier.CalculateFeatureRichnessScore(featuresWithACP)
	
	// Test scoring without ACP support
	scoreWithoutACP, breakdownWithoutACP := verifier.CalculateFeatureRichnessScore(featuresWithoutACP)

	// Verify that ACP support increases the score
	if scoreWithACP <= scoreWithoutACP {
		t.Errorf("Expected higher score with ACP support (%.2f) vs without (%.2f)", 
			scoreWithACP, scoreWithoutACP)
	}

	// Verify that experimental features score is higher with ACP
	if breakdownWithACP.ExperimentalFeaturesScore <= breakdownWithoutACP.ExperimentalFeaturesScore {
		t.Errorf("Expected higher experimental features score with ACP (%.2f) vs without (%.2f)",
			breakdownWithACP.ExperimentalFeaturesScore, breakdownWithoutACP.ExperimentalFeaturesScore)
	}
}

// TestACPsAPIValidation tests ACP fields in API validation
func TestACPsAPIValidation(t *testing.T) {
	// Test validation request with ACP support
	validationRequest := ValidationRequest{
		ModelName:                "test-model",
		Provider:                 "test-provider",
		Endpoint:                 "https://api.test.com",
		SupportsToolUse:          true,
		SupportsFunctionCalling:  true,
		SupportsCodeGeneration:   true,
		SupportsCodeCompletion:   true,
		SupportsCodeReview:       true,
		SupportsCodeExplanation:  true,
		SupportsEmbeddings:       false,
		SupportsReranking:        false,
		SupportsImageGeneration:  false,
		SupportsAudioGeneration:  false,
		SupportsVideoGeneration:  false,
		SupportsMCPs:             true,
		SupportsLSPs:             true,
		SupportsACPs:             true, // ACP support
		SupportsMultimodal:       false,
		SupportsStreaming:        true,
		SupportsJSONMode:         true,
		SupportsStructuredOutput: true,
		SupportsReasoning:        false,
		SupportsParallelToolUse:  false,
		MaxParallelCalls:         0,
		SupportsBatchProcessing:  false,
		SupportsBrotli:           false,
	}

	// Validate the request
	err := validateValidationRequest(validationRequest)
	if err != nil {
		t.Errorf("Validation request with ACP support should be valid: %v", err)
	}
}

// TestACPsDatabaseIntegration tests ACP fields in database operations
func TestACPsDatabaseIntegration(t *testing.T) {
	// Create a test verification result with ACP support
	verificationResult := llmverifier.VerificationResult{
		ModelInfo: llmverifier.ModelInfo{
			ID:      "test-model",
			Object:  "model",
			Created: 1234567890,
			OwnedBy: "test-provider",
		},
		Availability: llmverifier.AvailabilityResult{
			Exists:     true,
			Responsive: true,
			Overloaded: false,
			Latency:    100 * time.Millisecond,
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

	// Test database insertion
	err := insertVerificationResult(verificationResult)
	if err != nil {
		t.Errorf("Failed to insert verification result with ACP support: %v", err)
	}

	// Test database retrieval
	retrievedResult, err := getVerificationResult("test-model")
	if err != nil {
		t.Errorf("Failed to retrieve verification result: %v", err)
	}

	// Verify ACP support is preserved
	if !retrievedResult.FeatureDetection.ACPs {
		t.Error("ACP support was not preserved in database retrieval")
	}
}

// TestACPsReporting tests ACP inclusion in reporting
func TestACPsReporting(t *testing.T) {
	// Create a test result with ACP support
	result := llmverifier.VerificationResult{
		ModelInfo: llmverifier.ModelInfo{
			ID:      "test-model",
			Object:  "model",
			Created: 1234567890,
			OwnedBy: "test-provider",
		},
		FeatureDetection: llmverifier.FeatureDetectionResult{
			MCPs: true,
			LSPs: true,
			ACPs: true, // ACP support
		},
	}

	// Generate report
	report, err := generateMarkdownReport(result)
	if err != nil {
		t.Errorf("Failed to generate report: %v", err)
	}

	// Verify ACP is mentioned in report
	if !strings.Contains(report, "ACPs") {
		t.Error("Report should mention ACP support")
	}

	if !strings.Contains(report, "**ACPs**: true") {
		t.Error("Report should show ACP support as true")
	}
}

// MockLLMClient is a mock implementation for testing
type MockLLMClient struct {
	Responses map[string]string
}

func (m *MockLLMClient) ChatCompletion(ctx context.Context, request llmverifier.ChatCompletionRequest) (*llmverifier.ChatCompletionResponse, error) {
	// Simple mock implementation - return appropriate response based on request content
	content := ""
	for key, response := range m.Responses {
		if contains(request.Messages[0].Content, key) {
			content = response
			break
		}
	}

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

func contains(text, substring string) bool {
	return len(text) > 0 && len(substring) > 0 && strings.Contains(strings.ToLower(text), substring)
}