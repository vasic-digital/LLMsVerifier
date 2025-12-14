package tests

import (
	"testing"

	"llm-verifier/config"
	"llm-verifier/llmverifier"
)

// TestIntegrationWithMockedAPI tests integration with mocked API
func TestIntegrationWithMockedAPI(t *testing.T) {
	helper, cleanup := SetupTestEnvironment(t)
	defer cleanup()
	
	// Create test verifier
	verifier := CreateTestVerifier(helper.Config)
	
	// Test verification workflow
	t.Run("Successful verification workflow", func(t *testing.T) {
		results, err := verifier.Verify()
		AssertNoError(t, err)
		AssertTrue(t, results != nil, "Results should not be nil")
		AssertTrue(t, len(results) > 0, "Should have verification results")
		
		// Check that models were verified
		for _, result := range results {
			AssertTrue(t, result.ModelInfo.ID != "", "Model should have ID")
			AssertTrue(t, result.OverallScore >= 0, "Score should be non-negative")
		}
	})
	
	// Test with specific models
	t.Run("Verification with specific models", func(t *testing.T) {
		// Configure specific models
		helper.Config.LLMs = []config.LLMConfig{
			{
				Name:     "Test GPT-4",
				Endpoint: helper.MockServer.URL + "/v1",
				APIKey:   "test-api-key",
				Model:    "gpt-4-turbo",
			},
		}
		
		verifier := CreateTestVerifier(helper.Config)
		results, err := verifier.Verify()
		AssertNoError(t, err)
		AssertTrue(t, len(results) > 0, "Should have results for specific models")
	})
}

// TestConfigLoadingWithMockedPaths tests configuration loading with mocked paths
func TestConfigLoadingWithMockedPaths(t *testing.T) {
	// Create a temporary config file
	tempDir := t.TempDir()
	configContent := `
global:
  base_url: "https://api.test.com/v1"
  api_key: "test-key"
  max_retries: 3
  request_delay: 100ms
  timeout: 10s

llms:
  - name: "Test Model"
    endpoint: "https://api.test.com/v1"
    api_key: "test-key"
    model: "test-model"

concurrency: 2
timeout: 15s
`
	
	configPath := filepath.Join(tempDir, "test-config.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	AssertNoError(t, err)
	
	// Test loading the config
	cfg, err := llmverifier.LoadConfig(configPath)
	AssertNoError(t, err)
	AssertEquals(t, "https://api.test.com/v1", cfg.Global.BaseURL)
	AssertEquals(t, "test-key", cfg.Global.APIKey)
	AssertEquals(t, 2, cfg.Concurrency)
}

// TestDatabaseIntegration tests database operations
func TestDatabaseIntegration(t *testing.T) {
	helper, cleanup := SetupTestEnvironment(t)
	defer cleanup()
	
	// Test provider operations
	t.Run("Provider CRUD operations", func(t *testing.T) {
		provider := &database.Provider{
			Name:             "Test Provider",
			Endpoint:         "https://api.test.com/v1",
			APIKeyEncrypted:  "encrypted-key",
			Description:      "Test provider for integration tests",
			IsActive:         true,
			ReliabilityScore: 95.0,
		}
		
		// Create provider
		err := helper.DB.CreateProvider(provider)
		AssertNoError(t, err)
		AssertTrue(t, provider.ID > 0, "Provider should have ID after creation")
		
		// Get provider
		retrieved, err := helper.DB.GetProvider(provider.ID)
		AssertNoError(t, err)
		AssertEquals(t, provider.Name, retrieved.Name)
		
		// Update provider
		retrieved.ReliabilityScore = 98.0
		err = helper.DB.UpdateProvider(retrieved)
		AssertNoError(t, err)
		
		// Verify update
		updated, err := helper.DB.GetProvider(provider.ID)
		AssertNoError(t, err)
		AssertEquals(t, 98.0, updated.ReliabilityScore)
		
		// List providers
		providers, err := helper.DB.ListProviders(nil)
		AssertNoError(t, err)
		AssertTrue(t, len(providers) > 0, "Should have at least one provider")
	})
	
	// Test model operations
	t.Run("Model CRUD operations", func(t *testing.T) {
		// First create a provider
		provider := &database.Provider{
			Name:     "Test Provider 2",
			Endpoint: "https://api.test2.com/v1",
			IsActive: true,
		}
		err := helper.DB.CreateProvider(provider)
		AssertNoError(t, err)
		
		model := &database.Model{
			ProviderID:        provider.ID,
			ModelID:           "test-model-123",
			Name:              "Test Model",
			Description:       "Test model for integration",
			OverallScore:      85.5,
			VerificationStatus: "verified",
			Tags:              []string{"test", "integration"},
			LanguageSupport:   []string{"en", "es"},
		}
		
		// Create model
		err = helper.DB.CreateModel(model)
		AssertNoError(t, err)
		AssertTrue(t, model.ID > 0, "Model should have ID after creation")
		
		// Get model
		retrieved, err := helper.DB.GetModel(model.ID)
		AssertNoError(t, err)
		AssertEquals(t, model.Name, retrieved.Name)
		AssertEquals(t, 85.5, retrieved.OverallScore)
		
		// List models
		models, err := helper.DB.ListModels(nil)
		AssertNoError(t, err)
		AssertTrue(t, len(models) > 0, "Should have at least one model")
	})
	
	// Test verification results
	t.Run("Verification results storage", func(t *testing.T) {
		// Get a model first
		models, err := helper.DB.ListModels(nil)
		AssertNoError(t, err)
		AssertTrue(t, len(models) > 0, "Should have models for verification")
		
		result := &database.VerificationResult{
			ModelID:          models[0].ID,
			VerificationType: "full",
			Status:           "completed",
			Exists:           boolPtr(true),
			Responsive:       boolPtr(true),
			Overloaded:       boolPtr(false),
			LatencyMs:        intPtr(150),
			OverallScore:     92.5,
			CodeCapabilityScore: 95.0,
			ResponsivenessScore: 88.0,
			ReliabilityScore: 94.0,
			FeatureRichnessScore: 91.0,
			ValuePropositionScore: 89.0,
			SupportsToolUse: true,
			SupportsCodeGeneration: true,
			CodeLanguageSupport: []string{"python", "javascript", "go"},
		}
		
		err = helper.DB.CreateVerificationResult(result)
		AssertNoError(t, err)
		AssertTrue(t, result.ID > 0, "Result should have ID after creation")
		
		// Get result
		retrieved, err := helper.DB.GetVerificationResult(result.ID)
		AssertNoError(t, err)
		AssertEquals(t, result.OverallScore, retrieved.OverallScore)
		AssertTrue(t, retrieved.SupportsCodeGeneration, "Should support code generation")
	})
}

// TestReportGeneration tests report generation functionality
func TestReportGeneration(t *testing.T) {
	helper, cleanup := SetupTestEnvironment(t)
	defer cleanup()
	
	// Create some test data
	verifier := CreateTestVerifier(helper.Config)
	results := GenerateTestVerificationResults(3)
	
	// Generate Markdown report
	t.Run("Markdown report generation", func(t *testing.T) {
		err := verifier.GenerateMarkdownReport(results, t.TempDir())
		AssertNoError(t, err)
		
		// Check that report file was created
		reportPath := filepath.Join(t.TempDir(), "llm_verification_report.md")
		_, err = os.Stat(reportPath)
		AssertNoError(t, err)
	})
	
	// Generate JSON report
	t.Run("JSON report generation", func(t *testing.T) {
		err := verifier.GenerateJSONReport(results, t.TempDir())
		AssertNoError(t, err)
		
		// Check that report file was created
		reportPath := filepath.Join(t.TempDir(), "llm_verification_report.json")
		_, err = os.Stat(reportPath)
		AssertNoError(t, err)
	})
}

// TestJSONMarshaling tests JSON marshaling/unmarshaling
func TestJSONMarshaling(t *testing.T) {
	// Create test data
	result := &llmverifier.VerificationResult{
		ModelInfo: llmverifier.ModelInfo{
			ID:       "test-model",
			Object:   "model",
			Created:  time.Now().Unix(),
			Endpoint: "https://api.test.com/v1",
		},
		Availability: llmverifier.AvailabilityResult{
			Exists:      true,
			Responsive:  true,
			Overloaded:  false,
			Latency:     150 * time.Millisecond,
			LastChecked: time.Now(),
		},
		Timestamp: time.Now(),
		OverallScore: 85.5,
		PerformanceScores: llmverifier.PerformanceScore{
			OverallScore:          85.5,
			CodeCapability:        90.0,
			Responsiveness:        80.0,
			Reliability:           85.0,
			FeatureRichness:       82.0,
			ValueProposition:      88.0,
		},
	}
	
	// Marshal to JSON
	jsonData, err := json.Marshal(result)
	AssertNoError(t, err)
	AssertTrue(t, len(jsonData) > 0, "JSON data should not be empty")
	
	// Unmarshal from JSON
	var unmarshaled llmverifier.VerificationResult
	err = json.Unmarshal(jsonData, &unmarshaled)
	AssertNoError(t, err)
	AssertEquals(t, result.ModelInfo.ID, unmarshaled.ModelInfo.ID)
	AssertEquals(t, result.OverallScore, unmarshaled.OverallScore)
}

// TestVerifierWithMockedAPI tests the verifier with mocked API calls
func TestVerifierWithMockedAPI(t *testing.T) {
	helper, cleanup := SetupTestEnvironment(t)
	defer cleanup()
	
	verifier := CreateTestVerifier(helper.Config)
	
	// Test that verifier can work with mocked API
	results, err := verifier.Verify()
	AssertNoError(t, err)
	AssertTrue(t, results != nil, "Results should not be nil")
	
	// Verify scoring works correctly
	for _, result := range results {
		AssertTrue(t, result.OverallScore >= 0, "Score should be non-negative")
		AssertTrue(t, result.OverallScore <= 100, "Score should not exceed 100")
	}
}

// TestPerformanceWithMockedAPI tests performance with mocked API
func TestPerformanceWithMockedAPI(t *testing.T) {
	helper, cleanup := SetupTestEnvironment(t)
	defer cleanup()
	
	verifier := CreateTestVerifier(helper.Config)
	
	// Measure verification time
	start := time.Now()
	results, err := verifier.Verify()
	duration := time.Since(start)
	
	AssertNoError(t, err)
	AssertTrue(t, results != nil, "Results should not be nil")
	AssertTrue(t, duration < 10*time.Second, "Verification should complete within 10 seconds")
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func intPtr(i int) *int {
	return &i
}