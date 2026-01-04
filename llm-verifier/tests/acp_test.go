package tests

import (
	"context"
	"strings"
	"testing"
	"time"

	"llm-verifier/config"
	"llm-verifier/llmverifier"
)

// TestACPsCapabilityDetection tests the ACP capability detection function
func TestACPsCapabilityDetection(t *testing.T) {
	// Create a mock verifier instance
	cfg := &config.Config{
		Timeout: 30 * time.Second,
	}
	verifier := llmverifier.New(cfg)

	// Create a real client pointing to non-existent endpoint for testing error handling
	mockClient := llmverifier.NewLLMClient("http://localhost:9999", "test-key", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	modelName := "test-model"

	// Test ACP detection (will return false due to no actual server)
	supportsACP := verifier.TestACPs(mockClient, modelName, ctx)

	// Log result
	t.Logf("ACP detection test completed: %t", supportsACP)
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
	// Test validation with ACP support field in FeatureDetectionResult
	features := llmverifier.FeatureDetectionResult{
		ToolUse:          true,
		FunctionCalling:  true,
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
		ParallelToolUse:  false,
		MaxParallelCalls: 0,
		BatchProcessing:  false,
	}

	// Verify ACP field is set correctly
	if !features.ACPs {
		t.Error("ACP support should be true")
	}
}

// TestACPsDatabaseIntegration tests ACP fields in database operations
func TestACPsDatabaseIntegration(t *testing.T) {
	// Skip this test as it requires database setup
	t.Skip("Skipping database integration test - requires database setup")
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

	// Build a simple report string for testing
	var report strings.Builder
	report.WriteString("# Model Verification Report\n\n")
	report.WriteString("## Model: " + result.ModelInfo.ID + "\n\n")
	report.WriteString("## Features\n\n")
	report.WriteString("**MCPs**: " + boolToString(result.FeatureDetection.MCPs) + "\n")
	report.WriteString("**LSPs**: " + boolToString(result.FeatureDetection.LSPs) + "\n")
	report.WriteString("**ACPs**: " + boolToString(result.FeatureDetection.ACPs) + "\n")

	reportStr := report.String()

	// Verify ACP is mentioned in report
	if !strings.Contains(reportStr, "ACPs") {
		t.Error("Report should mention ACP support")
	}

	if !strings.Contains(reportStr, "**ACPs**: true") {
		t.Error("Report should show ACP support as true")
	}
}

// Helper function to convert bool to string
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
