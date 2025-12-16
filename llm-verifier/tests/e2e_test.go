package tests

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"llm-verifier/config"
	"llm-verifier/llmverifier"
)

// End-to-end tests for the LLM verifier
// These tests verify the complete workflow from config to report generation

func TestEndToEndWithEmptyConfig(t *testing.T) {
	// Test the complete workflow with minimal config
	// This test verifies the case where no specific LLMs are configured,
	// so the tool should attempt to discover all available models

	// Create test helper to get mock server
	helper := NewTestHelper(t)
	defer helper.Cleanup()

	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:    helper.Config.Global.BaseURL, // Use mock server URL
			APIKey:     "test-api-key",               // This must match what mock server expects
			MaxRetries: 1,
			Timeout:    5, // Short timeout for tests
		},
		Concurrency: 1,
		Timeout:     10,
	}

	verifier := llmverifier.New(cfg)

	// This should return results (possibly with errors due to fake API key)
	results, err := verifier.Verify()
	if err != nil {
		// For this test, we expect an error due to the fake API key
		// The important thing is that the process completes without panicking
		t.Logf("Expected error with fake API key: %v", err)
	}

	// Verify that results is not nil (even if empty or with errors)
	if results == nil {
		t.Error("Expected results to be non-nil, even with errors")
	}
}

func TestReportGeneration(t *testing.T) {
	// Test that reports can be generated with mock data
	outputDir := t.TempDir()

	// Create mock verification results
	mockResults := []llmverifier.VerificationResult{
		{
			ModelInfo: llmverifier.ModelInfo{
				ID:       "gpt-4-mock",
				Endpoint: "https://api.openai.com/v1",
			},
			Availability: llmverifier.AvailabilityResult{
				Exists:     true,
				Responsive: true,
				Overloaded: false,
			},
			PerformanceScores: llmverifier.PerformanceScore{
				OverallScore:     85.5,
				CodeCapability:   90.0,
				Responsiveness:   80.0,
				Reliability:      85.0,
				FeatureRichness:  95.0,
				ValueProposition: 82.0,
			},
			Timestamp: time.Now(),
		},
		{
			ModelInfo: llmverifier.ModelInfo{
				ID:       "gpt-3.5-turbo-mock",
				Endpoint: "https://api.openai.com/v1",
			},
			Availability: llmverifier.AvailabilityResult{
				Exists:     true,
				Responsive: true,
				Overloaded: false,
			},
			PerformanceScores: llmverifier.PerformanceScore{
				OverallScore:     75.2,
				CodeCapability:   70.0,
				Responsiveness:   85.0,
				Reliability:      70.0,
				FeatureRichness:  75.0,
				ValueProposition: 72.0,
			},
			Timestamp: time.Now(),
		},
	}

	// Create a verifier with minimal config
	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:    "https://api.openai.com/v1",
			APIKey:     "fake-key",
			MaxRetries: 1,
		},
		Concurrency: 1,
	}

	verifier := llmverifier.New(cfg)

	// Test markdown report generation
	err := verifier.GenerateMarkdownReport(mockResults, outputDir)
	if err != nil {
		t.Fatalf("Failed to generate markdown report: %v", err)
	}

	// Verify markdown file was created
	markdownPath := filepath.Join(outputDir, "llm_verification_report.md")
	if _, err := os.Stat(markdownPath); os.IsNotExist(err) {
		t.Errorf("Markdown report file was not created at %s", markdownPath)
	}

	// Test JSON report generation
	err = verifier.GenerateJSONReport(mockResults, outputDir)
	if err != nil {
		t.Fatalf("Failed to generate JSON report: %v", err)
	}

	// Verify JSON file was created
	jsonPath := filepath.Join(outputDir, "llm_verification_report.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Errorf("JSON report file was not created at %s", jsonPath)
	}
}

func TestFullWorkflow(t *testing.T) {
	// Test the full workflow from config to reports with mock data

	// Create a config that would cause verification to use auto-discovery
	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:    "https://api.openai.com/v1",
			APIKey:     "fake-key",
			MaxRetries: 1,
			Timeout:    5,
		},
		Concurrency: 1,
		Timeout:     10,
	}

	verifier := llmverifier.New(cfg)

	// This will fail due to fake key, but it should still generate empty reports
	_, err := verifier.Verify()
	if err == nil {
		// If no error occurred, it means we had valid credentials (which is unexpected)
		t.Log("Unexpected: verification succeeded with fake credentials")
	} else {
		// Expected behavior: verification fails with invalid credentials
		t.Logf("Expected error with fake credentials: %v", err)
	}

	// We can't properly test the full workflow without real credentials,
	// but we can verify that the components connect properly by checking
	// that the method doesn't panic or crash
}
