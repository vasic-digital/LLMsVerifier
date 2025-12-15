package llmverifier

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"llm-verifier/config"
)

func TestGenerateMarkdownReport(t *testing.T) {
	// Create a temporary directory for test output
	tempDir := t.TempDir()

	// Create a verifier instance
	verifier := &Verifier{
		cfg: &config.Config{
			LLMs: []config.LLMConfig{
				{
					Name:     "test-model-1",
					Endpoint: "http://localhost:8080",
				},
				{
					Name:     "test-model-2",
					Endpoint: "http://localhost:8081",
				},
			},
		},
	}

	// Create test verification results
	results := []VerificationResult{
		{
			ModelInfo: ModelInfo{
				ID:       "test-model-1",
				Endpoint: "http://localhost:8080",
			},
			Availability: AvailabilityResult{
				Exists:     true,
				Responsive: true,
				Overloaded: false,
				Latency:    100 * time.Millisecond,
			},
			PerformanceScores: PerformanceScore{
				OverallScore:     85.5,
				CodeCapability:   90.0,
				Responsiveness:   80.0,
				Reliability:      85.0,
				FeatureRichness:  75.0,
				ValueProposition: 82.0,
			},
			FeatureDetection: FeatureDetectionResult{
				ToolUse:          true,
				FunctionCalling:  true,
				CodeGeneration:   true,
				CodeCompletion:   true,
				CodeExplanation:  true,
				CodeReview:       true,
				Embeddings:       false,
				Reranking:        false,
				ImageGeneration:  false,
				AudioGeneration:  false,
				VideoGeneration:  false,
				MCPs:             false,
				LSPs:             false,
				Multimodal:       false,
				Streaming:        true,
				JSONMode:         true,
				StructuredOutput: true,
				Reasoning:        false,
				ParallelToolUse:  true,
				MaxParallelCalls: 3,
			},
			CodeCapabilities: CodeCapabilityResult{
				LanguageSupport:    []string{"Python", "JavaScript", "Go"},
				CodeGeneration:     true,
				CodeCompletion:     true,
				CodeDebugging:      true,
				CodeOptimization:   true,
				CodeReview:         true,
				TestGeneration:     true,
				Documentation:      true,
				Refactoring:        true,
				ErrorResolution:    true,
				Architecture:       true,
				SecurityAssessment: true,
				PatternRecognition: true,
				ComplexityHandling: ComplexityMetrics{
					MaxHandledDepth:   4,
					CodeQuality:       85.0,
					LogicCorrectness:  90.0,
					RuntimeEfficiency: 80.0,
				},
				PromptResponse: PromptResponseTest{
					PythonSuccessRate:     95.0,
					JavascriptSuccessRate: 90.0,
					GoSuccessRate:         85.0,
					JavaSuccessRate:       75.0,
					CppSuccessRate:        70.0,
					TypescriptSuccessRate: 88.0,
					OverallSuccessRate:    85.5,
				},
			},
			Timestamp: time.Now(),
		},
		{
			ModelInfo: ModelInfo{
				ID:       "test-model-2",
				Endpoint: "http://localhost:8081",
			},
			Error:     "connection refused",
			Timestamp: time.Now(),
		},
	}

	// Test markdown report generation
	err := verifier.GenerateMarkdownReport(results, tempDir)
	if err != nil {
		t.Fatalf("Failed to generate markdown report: %v", err)
	}

	// Verify the report file was created
	reportPath := filepath.Join(tempDir, "llm_verification_report.md")
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Fatalf("Markdown report file was not created: %v", err)
	}

	// Read and verify the report content
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read markdown report: %v", err)
	}

	// Basic content checks
	reportContent := string(content)
	if !contains(reportContent, "# LLM Verification Report") {
		t.Error("Report missing title")
	}
	if !contains(reportContent, "## Model: test-model-1") {
		t.Error("Report missing successful model section")
	}
	if !contains(reportContent, "## Model: test-model-2 (FAILED)") {
		t.Error("Report missing failed model section")
	}
	if !contains(reportContent, "## Summary") {
		t.Error("Report missing summary section")
	}
	if !contains(reportContent, "## Category Rankings") {
		t.Error("Report missing category rankings section")
	}
}

func TestGenerateJSONReport(t *testing.T) {
	// Create a temporary directory for test output
	tempDir := t.TempDir()

	// Create a verifier instance
	verifier := &Verifier{
		cfg: &config.Config{
			LLMs: []config.LLMConfig{
				{
					Name:     "test-model-1",
					Endpoint: "http://localhost:8080",
				},
			},
		},
	}

	// Create test verification results
	results := []VerificationResult{
		{
			ModelInfo: ModelInfo{
				ID:       "test-model-1",
				Endpoint: "http://localhost:8080",
			},
			Availability: AvailabilityResult{
				Exists:     true,
				Responsive: true,
				Overloaded: false,
				Latency:    100 * time.Millisecond,
			},
			PerformanceScores: PerformanceScore{
				OverallScore:     85.5,
				CodeCapability:   90.0,
				Responsiveness:   80.0,
				Reliability:      85.0,
				FeatureRichness:  75.0,
				ValueProposition: 82.0,
			},
			Timestamp: time.Now(),
		},
	}

	// Test JSON report generation
	err := verifier.GenerateJSONReport(results, tempDir)
	if err != nil {
		t.Fatalf("Failed to generate JSON report: %v", err)
	}

	// Verify the report file was created
	reportPath := filepath.Join(tempDir, "llm_verification_report.json")
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Fatalf("JSON report file was not created: %v", err)
	}

	// Read and parse the JSON report
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read JSON report: %v", err)
	}

	var jsonReport map[string]interface{}
	err = json.Unmarshal(content, &jsonReport)
	if err != nil {
		t.Fatalf("Failed to parse JSON report: %v", err)
	}

	// Verify JSON structure
	if _, ok := jsonReport["summary"]; !ok {
		t.Error("JSON report missing summary")
	}
	if _, ok := jsonReport["results"]; !ok {
		t.Error("JSON report missing results")
	}
	if _, ok := jsonReport["metadata"]; !ok {
		t.Error("JSON report missing metadata")
	}

	// Verify metadata
	metadata, ok := jsonReport["metadata"].(map[string]interface{})
	if !ok {
		t.Error("Metadata is not a map")
	}
	if totalModels, ok := metadata["total_models"].(float64); !ok || totalModels != 1 {
		t.Errorf("Expected total_models to be 1, got %v", totalModels)
	}
	if _, ok := metadata["generated_at"]; !ok {
		t.Error("Metadata missing generated_at")
	}
}

func TestGenerateSummary(t *testing.T) {
	verifier := &Verifier{}

	// Create test results with mixed success/failure
	results := []VerificationResult{
		{
			ModelInfo:         ModelInfo{ID: "model-1"},
			PerformanceScores: PerformanceScore{OverallScore: 90.0},
		},
		{
			ModelInfo:         ModelInfo{ID: "model-2"},
			PerformanceScores: PerformanceScore{OverallScore: 80.0},
		},
		{
			ModelInfo: ModelInfo{ID: "model-3"},
			Error:     "connection failed",
		},
	}

	summary := verifier.generateSummary(results)

	// Verify summary calculations
	if summary.TotalModels != 3 {
		t.Errorf("Expected TotalModels=3, got %d", summary.TotalModels)
	}
	if summary.AvailableModels != 2 {
		t.Errorf("Expected AvailableModels=2, got %d", summary.AvailableModels)
	}
	if summary.FailedModels != 1 {
		t.Errorf("Expected FailedModels=1, got %d", summary.FailedModels)
	}
	if summary.AverageScore != 85.0 {
		t.Errorf("Expected AverageScore=85.0, got %.2f", summary.AverageScore)
	}
	if len(summary.CategoryRankings.ByCodeCapability) != 2 {
		t.Errorf("Expected 2 code capability rankings, got %d", len(summary.CategoryRankings.ByCodeCapability))
	}
}

func TestGenerateCategoryRankings(t *testing.T) {
	verifier := &Verifier{}

	// Create test results with different scores
	results := []VerificationResult{
		{
			ModelInfo: ModelInfo{ID: "model-high-code"},
			PerformanceScores: PerformanceScore{
				OverallScore:     85.0,
				CodeCapability:   95.0,
				Responsiveness:   75.0,
				Reliability:      80.0,
				FeatureRichness:  70.0,
				ValueProposition: 85.0,
			},
		},
		{
			ModelInfo: ModelInfo{ID: "model-high-responsive"},
			PerformanceScores: PerformanceScore{
				OverallScore:     80.0,
				CodeCapability:   75.0,
				Responsiveness:   95.0,
				Reliability:      85.0,
				FeatureRichness:  80.0,
				ValueProposition: 75.0,
			},
		},
		{
			ModelInfo: ModelInfo{ID: "model-failed"},
			Error:     "connection failed",
		},
	}

	rankings := verifier.generateCategoryRankings(results)

	// Test code capability rankings
	if len(rankings.ByCodeCapability) != 2 {
		t.Errorf("Expected 2 code capability rankings, got %d", len(rankings.ByCodeCapability))
	}
	if rankings.ByCodeCapability[0].ModelName != "model-high-code" {
		t.Errorf("Expected top code capability model to be 'model-high-code', got %s", rankings.ByCodeCapability[0].ModelName)
	}
	if rankings.ByCodeCapability[0].Score != 95.0 {
		t.Errorf("Expected top code capability score to be 95.0, got %.2f", rankings.ByCodeCapability[0].Score)
	}

	// Test responsiveness rankings
	if len(rankings.ByResponsiveness) != 2 {
		t.Errorf("Expected 2 responsiveness rankings, got %d", len(rankings.ByResponsiveness))
	}
	if rankings.ByResponsiveness[0].ModelName != "model-high-responsive" {
		t.Errorf("Expected top responsiveness model to be 'model-high-responsive', got %s", rankings.ByResponsiveness[0].ModelName)
	}
	if rankings.ByResponsiveness[0].Score != 95.0 {
		t.Errorf("Expected top responsiveness score to be 95.0, got %.2f", rankings.ByResponsiveness[0].Score)
	}

	// Test reliability rankings
	if len(rankings.ByReliability) != 2 {
		t.Errorf("Expected 2 reliability rankings, got %d", len(rankings.ByReliability))
	}

	// Test feature richness rankings
	if len(rankings.ByFeatureRichness) != 2 {
		t.Errorf("Expected 2 feature richness rankings, got %d", len(rankings.ByFeatureRichness))
	}

	// Test value proposition rankings
	if len(rankings.ByValue) != 2 {
		t.Errorf("Expected 2 value proposition rankings, got %d", len(rankings.ByValue))
	}
}

func TestSortResultsByScore(t *testing.T) {
	verifier := &Verifier{}

	results := []VerificationResult{
		{
			ModelInfo:         ModelInfo{ID: "model-low"},
			PerformanceScores: PerformanceScore{OverallScore: 70.0},
		},
		{
			ModelInfo:         ModelInfo{ID: "model-high"},
			PerformanceScores: PerformanceScore{OverallScore: 90.0},
		},
		{
			ModelInfo:         ModelInfo{ID: "model-medium"},
			PerformanceScores: PerformanceScore{OverallScore: 80.0},
		},
	}

	// Test sorting by overall score
	sorted := verifier.SortResultsByScore(results, func(r VerificationResult) float64 {
		return r.PerformanceScores.OverallScore
	})

	if len(sorted) != 3 {
		t.Fatalf("Expected 3 sorted results, got %d", len(sorted))
	}

	// Verify descending order
	if sorted[0].ModelInfo.ID != "model-high" {
		t.Errorf("Expected first model to be 'model-high', got %s", sorted[0].ModelInfo.ID)
	}
	if sorted[1].ModelInfo.ID != "model-medium" {
		t.Errorf("Expected second model to be 'model-medium', got %s", sorted[1].ModelInfo.ID)
	}
	if sorted[2].ModelInfo.ID != "model-low" {
		t.Errorf("Expected third model to be 'model-low', got %s", sorted[2].ModelInfo.ID)
	}

	// Test sorting with failed models
	resultsWithFailure := []VerificationResult{
		{
			ModelInfo:         ModelInfo{ID: "model-ok"},
			PerformanceScores: PerformanceScore{OverallScore: 85.0},
		},
		{
			ModelInfo: ModelInfo{ID: "model-failed"},
			Error:     "connection failed",
		},
	}

	sortedWithFailure := verifier.SortResultsByScore(resultsWithFailure, func(r VerificationResult) float64 {
		return r.PerformanceScores.OverallScore
	})

	if len(sortedWithFailure) != 2 {
		t.Fatalf("Expected 2 sorted results with failure, got %d", len(sortedWithFailure))
	}
	if sortedWithFailure[0].ModelInfo.ID != "model-ok" {
		t.Errorf("Expected first model to be 'model-ok', got %s", sortedWithFailure[0].ModelInfo.ID)
	}
}

func TestReportGenerationWithEmptyResults(t *testing.T) {
	tempDir := t.TempDir()
	verifier := &Verifier{}

	// Test with empty results
	emptyResults := []VerificationResult{}

	// Should not error with empty results
	err := verifier.GenerateMarkdownReport(emptyResults, tempDir)
	if err != nil {
		t.Fatalf("Failed to generate markdown report with empty results: %v", err)
	}

	// Verify file was created
	reportPath := filepath.Join(tempDir, "llm_verification_report.md")
	if _, err := os.Stat(reportPath); os.IsNotExist(err) {
		t.Fatalf("Markdown report file was not created with empty results: %v", err)
	}

	// Test JSON report with empty results
	err = verifier.GenerateJSONReport(emptyResults, tempDir)
	if err != nil {
		t.Fatalf("Failed to generate JSON report with empty results: %v", err)
	}

	jsonPath := filepath.Join(tempDir, "llm_verification_report.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Fatalf("JSON report file was not created with empty results: %v", err)
	}
}

func TestReportGenerationWithAllFailedModels(t *testing.T) {
	tempDir := t.TempDir()
	verifier := &Verifier{}

	// Create results with all failed models
	results := []VerificationResult{
		{
			ModelInfo: ModelInfo{ID: "model-1"},
			Error:     "connection refused",
		},
		{
			ModelInfo: ModelInfo{ID: "model-2"},
			Error:     "timeout",
		},
	}

	err := verifier.GenerateMarkdownReport(results, tempDir)
	if err != nil {
		t.Fatalf("Failed to generate markdown report with all failed models: %v", err)
	}

	// Verify file was created
	reportPath := filepath.Join(tempDir, "llm_verification_report.md")
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("Failed to read markdown report: %v", err)
	}

	reportContent := string(content)
	if !contains(reportContent, "Available Models: 0") {
		t.Error("Report should show 0 available models")
	}
	if !contains(reportContent, "Failed Models: 2") {
		t.Error("Report should show 2 failed models")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || contains(s[1:], substr)))
}
