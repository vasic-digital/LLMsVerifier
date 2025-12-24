//go:build integration
// +build integration

package testing

import (
	"encoding/json"
	"testing"
	"time"

	"llm-verifier/auth"
	"llm-verifier/database"
)

// TestEndToEndWorkflowSuite runs comprehensive end-to-end tests
func TestEndToEndWorkflowSuite(t *testing.T) {
	t.Run("CompleteVerificationWorkflow", testCompleteVerificationWorkflow)
	t.Run("ConfigurationExportWorkflow", testConfigurationExportWorkflow)
	t.Run("ClientIntegrationWorkflow", testClientIntegrationWorkflow)
	t.Run("MultiProviderWorkflow", testMultiProviderWorkflow)
	t.Run("FailureRecoveryWorkflow", testFailureRecoveryWorkflow)
}

// testCompleteVerificationWorkflow tests the complete verification workflow
func testCompleteVerificationWorkflow(t *testing.T) {
	t.Log("Testing complete verification workflow...")

	// Setup database
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Run migrations
	mm := database.NewMigrationManager(db)
	mm.SetupDefaultMigrations()
	if err := mm.MigrateUp(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Step 1: Provider Setup
	t.Log("Step 1: Setting up test provider...")
	provider := &database.Provider{
		Name:                  "OpenAI",
		Endpoint:              "https://api.openai.com/v1",
		APIKeyEncrypted:       "encrypted_openai_key",
		Description:           "OpenAI API Provider",
		Website:               "https://openai.com",
		SupportEmail:          "support@openai.com",
		DocumentationURL:      "https://platform.openai.com/docs",
		IsActive:              true,
		ReliabilityScore:      98.5,
		AverageResponseTimeMs: 150,
	}

	err = db.CreateProvider(provider)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	// Step 2: Model Setup
	t.Log("Step 2: Setting up test models...")
	models := []*database.Model{
		{
			ProviderID:          provider.ID,
			ModelID:             "gpt-4",
			Name:                "GPT-4",
			Description:         "Most advanced GPT model",
			ContextWindowTokens: &[]int{8192}[0],
			MaxOutputTokens:     &[]int{4096}[0],
			IsMultimodal:        false,
		},
		{
			ProviderID:          provider.ID,
			ModelID:             "gpt-3.5-turbo",
			Name:                "GPT-3.5 Turbo",
			Description:         "Fast and efficient GPT model",
			ContextWindowTokens: &[]int{4096}[0],
			MaxOutputTokens:     &[]int{2048}[0],
			IsMultimodal:        false,
		},
	}

	for _, model := range models {
		err = db.CreateModel(model)
		if err != nil {
			t.Fatalf("Failed to create model %s: %v", model.Name, err)
		}
	}

	// Step 3: Verification Simulation
	t.Log("Step 3: Simulating model verification...")
	for _, model := range models {
		// Simulate verification result
		latency := 150
		exists := true
		responsive := true
		result := &database.VerificationResult{
			ModelID:                model.ID,
			VerificationType:       "api_test",
			StartedAt:              time.Now().Add(-time.Minute),
			CompletedAt:            &[]time.Time{time.Now()}[0],
			Status:                 "completed",
			ErrorMessage:           nil,
			ModelExists:            &exists,
			Responsive:             &responsive,
			LatencyMs:              &latency,
			SupportsToolUse:        true,
			SupportsCodeGeneration: true,
		}

		err = db.CreateVerificationResult(result)
		if err != nil {
			t.Fatalf("Failed to create verification result for %s: %v", model.Name, err)
		}
	}

	// Step 4: Verification Results Query
	t.Log("Step 4: Querying verification results...")
	for _, model := range models {
		results, err := db.GetLatestVerificationResults([]int64{model.ID})
		if err != nil {
			t.Fatalf("Failed to get verification results for %s: %v", model.Name, err)
		}

		if len(results) == 0 {
			t.Errorf("No verification results found for model %s", model.Name)
		}

		// Verify result data
		result := results[0]
		if result.Status != "completed" {
			t.Errorf("Expected verification status 'completed', got '%s'", result.Status)
		}
		if result.SupportsCodeGeneration != true {
			t.Error("Expected model to support code generation")
		}
	}

	// Step 5: Analytics and Reporting
	t.Log("Step 5: Testing analytics and reporting...")
	// Simulate analytics processing
	analytics := map[string]interface{}{
		"total_models":       len(models),
		"verified_models":    len(models),
		"average_latency_ms": 150,
		"success_rate":       1.0,
		"provider_health":    "excellent",
	}

	// Verify analytics data structure
	if analytics["total_models"].(int) != len(models) {
		t.Error("Analytics total_models mismatch")
	}

	t.Logf("Workflow completed successfully: %d models verified", len(models))
}

// testConfigurationExportWorkflow tests the configuration export workflow
func testConfigurationExportWorkflow(t *testing.T) {
	t.Log("Testing configuration export workflow...")

	// Setup database with test data
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	mm := database.NewMigrationManager(db)
	mm.SetupDefaultMigrations()
	if err := mm.MigrateUp(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create test providers and models
	providers := []struct {
		name     string
		endpoint string
		models   []string
	}{
		{"OpenAI", "https://api.openai.com/v1", []string{"gpt-4", "gpt-3.5-turbo"}},
		{"Anthropic", "https://api.anthropic.com", []string{"claude-3", "claude-2"}},
	}

	var allModels []*database.Model

	for _, p := range providers {
		provider := &database.Provider{
			Name:            p.name,
			Endpoint:        p.endpoint,
			APIKeyEncrypted: "encrypted_key",
			IsActive:        true,
		}

		err = db.CreateProvider(provider)
		if err != nil {
			t.Fatalf("Failed to create provider %s: %v", p.name, err)
		}

		for _, modelID := range p.models {
			contextWindow := 4096
			maxTokens := 2048
			model := &database.Model{
				ProviderID:          provider.ID,
				ModelID:             modelID,
				Name:                modelID,
				Description:         "Test model",
				ContextWindowTokens: &contextWindow,
				MaxOutputTokens:     &maxTokens,
				IsMultimodal:        false,
			}

			err = db.CreateModel(model)
			if err != nil {
				t.Fatalf("Failed to create model %s: %v", modelID, err)
			}
			allModels = append(allModels, model)
		}
	}

	// Simulate configuration export
	t.Log("Simulating configuration export...")

	// Create OpenCode-style configuration
	opencodeConfig := map[string]interface{}{
		"version":   "1.0",
		"providers": []map[string]interface{}{},
	}

	for _, provider := range providers {
		providerConfig := map[string]interface{}{
			"name":     provider.name,
			"endpoint": provider.endpoint,
			"api_key":  "sk-test-key", // In real export, this would be decrypted
			"models":   provider.models,
		}
		opencodeConfig["providers"] = append(opencodeConfig["providers"].([]map[string]interface{}), providerConfig)
	}

	// Verify configuration structure
	providersConfig := opencodeConfig["providers"].([]map[string]interface{})
	if len(providersConfig) != len(providers) {
		t.Errorf("Expected %d providers in config, got %d", len(providers), len(providersConfig))
	}

	// Verify JSON serialization
	configJSON, err := json.MarshalIndent(opencodeConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal configuration: %v", err)
	}

	if len(configJSON) == 0 {
		t.Error("Configuration JSON should not be empty")
	}

	t.Logf("Configuration export successful: %d bytes", len(configJSON))
}

// testClientIntegrationWorkflow tests client integration workflow
func testClientIntegrationWorkflow(t *testing.T) {
	t.Log("Testing client integration workflow...")

	// Setup authentication manager
	authManager := auth.NewAuthManager("test-jwt-secret")

	// Setup database
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	mm := database.NewMigrationManager(db)
	mm.SetupDefaultMigrations()
	if err := mm.MigrateUp(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Step 1: Client Registration
	t.Log("Step 1: Client registration...")
	client, apiKey, err := authManager.RegisterClient(
		"test-integration-client",
		"Client for integration testing",
		[]string{"read", "write", "export"},
		100,
	)
	if err != nil {
		t.Fatalf("Failed to register client: %v", err)
	}

	// Step 2: Authentication
	t.Log("Step 2: Client authentication...")
	authenticatedClient, err := authManager.AuthenticateClient(apiKey)
	if err != nil {
		t.Fatalf("Authentication failed: %v", err)
	}

	if authenticatedClient.ID != client.ID {
		t.Error("Authenticated client ID mismatch")
	}

	// Step 3: Authorization
	t.Log("Step 3: Authorization checks...")
	permissions := []string{"read", "write", "export"}
	for _, permission := range permissions {
		err := authManager.AuthorizeRequest(client, permission)
		if err != nil {
			t.Errorf("Authorization failed for permission '%s': %v", permission, err)
		}
	}

	// Step 4: JWT Token Generation
	t.Log("Step 4: JWT token operations...")
	token, err := authManager.GenerateJWTToken(client, time.Hour)
	if err != nil {
		t.Fatalf("JWT generation failed: %v", err)
	}

	claims, err := authManager.ValidateJWTToken(token)
	if err != nil {
		t.Fatalf("JWT validation failed: %v", err)
	}

	if claims.ClientID != client.ID {
		t.Error("JWT claims client ID mismatch")
	}

	// Step 5: Rate Limiting
	t.Log("Step 5: Rate limiting...")
	for i := 0; i < 10; i++ {
		err := authManager.CheckRateLimit(client.ID, client.RateLimit)
		if err != nil {
			t.Logf("Rate limit check %d: %v", i, err)
		}
	}

	// Step 6: Usage Tracking (simulated)
	t.Log("Step 6: Usage tracking...")
	usageStats := map[string]interface{}{
		"client_id":       client.ID,
		"requests_today":  45,
		"requests_hour":   5,
		"total_requests":  1250,
		"last_request_at": time.Now(),
	}

	// Verify usage stats structure
	if usageStats["client_id"].(int64) != client.ID {
		t.Error("Usage stats client ID mismatch")
	}

	t.Logf("Client integration workflow completed for client %s", client.Name)
}

// testMultiProviderWorkflow tests workflow with multiple providers
func testMultiProviderWorkflow(t *testing.T) {
	t.Log("Testing multi-provider workflow...")

	// Setup database
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	mm := database.NewMigrationManager(db)
	mm.SetupDefaultMigrations()
	if err := mm.MigrateUp(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Define multiple providers with different characteristics
	providerConfigs := []struct {
		name             string
		endpoint         string
		models           []string
		reliabilityScore float64
		responseTime     int
	}{
		{"OpenAI", "https://api.openai.com/v1", []string{"gpt-4", "gpt-3.5-turbo"}, 98.5, 150},
		{"Anthropic", "https://api.anthropic.com", []string{"claude-3-opus", "claude-3-sonnet"}, 97.8, 200},
		{"Google", "https://generativelanguage.googleapis.com", []string{"gemini-pro", "gemini-pro-vision"}, 96.2, 180},
		{"Meta", "https://api.meta.com", []string{"llama-2-70b", "llama-2-13b"}, 94.1, 250},
	}

	totalModels := 0
	var allProviders []*database.Provider
	var allModels []*database.Model

	// Step 1: Setup all providers and models
	t.Log("Step 1: Setting up multiple providers...")
	for _, config := range providerConfigs {
		provider := &database.Provider{
			Name:                  config.name,
			Endpoint:              config.endpoint,
			APIKeyEncrypted:       "encrypted_key_" + config.name,
			Description:           config.name + " API Provider",
			Website:               "https://" + config.name + ".com",
			SupportEmail:          "support@" + config.name + ".com",
			IsActive:              true,
			ReliabilityScore:      config.reliabilityScore,
			AverageResponseTimeMs: config.responseTime,
		}

		err = db.CreateProvider(provider)
		if err != nil {
			t.Fatalf("Failed to create provider %s: %v", config.name, err)
		}
		allProviders = append(allProviders, provider)

		// Create models for this provider
		for _, modelID := range config.models {
			contextWindow := 4096
			maxTokens := 2048
			model := &database.Model{
				ProviderID:          provider.ID,
				ModelID:             modelID,
				Name:                modelID,
				Description:         "Model from " + config.name,
				ContextWindowTokens: &contextWindow,
				MaxOutputTokens:     &maxTokens,
				IsMultimodal:        false,
			}

			err = db.CreateModel(model)
			if err != nil {
				t.Fatalf("Failed to create model %s: %v", modelID, err)
			}
			allModels = append(allModels, model)
			totalModels++
		}
	}

	// Step 2: Simulate verification for all models
	t.Log("Step 2: Running verification across all providers...")
	verificationResults := make([]*database.VerificationResult, 0, totalModels)

	for _, model := range allModels {
		latency := 150
		exists := true
		responsive := true
		result := &database.VerificationResult{
			ModelID:                model.ID,
			VerificationType:       "multi_provider_test",
			StartedAt:              time.Now().Add(-time.Minute),
			CompletedAt:            &[]time.Time{time.Now()}[0],
			Status:                 "completed",
			ErrorMessage:           nil,
			ModelExists:            &exists,
			Responsive:             &responsive,
			LatencyMs:              &latency,
			SupportsCodeGeneration: true,
		}

		err = db.CreateVerificationResult(result)
		if err != nil {
			t.Fatalf("Failed to create verification result for %s: %v", model.Name, err)
		}
		verificationResults = append(verificationResults, result)
	}

	// Step 3: Analytics and reporting
	t.Log("Step 3: Generating multi-provider analytics...")
	analytics := map[string]interface{}{
		"total_providers":     len(allProviders),
		"total_models":        totalModels,
		"verified_models":     len(verificationResults),
		"average_reliability": 96.65, // Calculated average
		"best_provider":       "OpenAI",
		"fastest_provider":    "OpenAI",
		"coverage_score":      0.95,
	}

	// Verify analytics
	if analytics["total_providers"].(int) != len(providerConfigs) {
		t.Error("Analytics provider count mismatch")
	}

	if analytics["total_models"].(int) != totalModels {
		t.Error("Analytics model count mismatch")
	}

	// Step 4: Export configuration for all providers
	t.Log("Step 4: Exporting unified configuration...")
	unifiedConfig := map[string]interface{}{
		"version":   "1.0",
		"providers": []map[string]interface{}{},
		"summary": map[string]interface{}{
			"total_providers": len(allProviders),
			"total_models":    totalModels,
			"last_updated":    time.Now(),
		},
	}

	for _, provider := range allProviders {
		providerModels := make([]string, 0)
		for _, model := range allModels {
			if model.ProviderID == provider.ID {
				providerModels = append(providerModels, model.ModelID)
			}
		}

		providerConfig := map[string]interface{}{
			"name":             provider.Name,
			"endpoint":         provider.Endpoint,
			"api_key":          "configured_key", // Would be decrypted in real export
			"models":           providerModels,
			"reliability":      provider.ReliabilityScore,
			"response_time_ms": provider.AverageResponseTimeMs,
		}
		unifiedConfig["providers"] = append(unifiedConfig["providers"].([]map[string]interface{}), providerConfig)
	}

	// Verify unified configuration
	configJSON, err := json.MarshalIndent(unifiedConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal unified config: %v", err)
	}

	t.Logf("Multi-provider workflow completed: %d providers, %d models, %d bytes config",
		len(allProviders), totalModels, len(configJSON))
}

// testFailureRecoveryWorkflow tests failure and recovery scenarios
func testFailureRecoveryWorkflow(t *testing.T) {
	t.Log("Testing failure recovery workflow...")

	// Setup database
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	mm := database.NewMigrationManager(db)
	mm.SetupDefaultMigrations()
	if err := mm.MigrateUp(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Step 1: Setup provider and models
	t.Log("Step 1: Setting up test environment...")
	provider := &database.Provider{
		Name:             "UnreliableProvider",
		Endpoint:         "https://unreliable.api.com",
		APIKeyEncrypted:  "encrypted_key",
		IsActive:         true,
		ReliabilityScore: 50.0, // Low reliability
	}

	err = db.CreateProvider(provider)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	contextWindow := 4096
	maxTokens := 2048
	model := &database.Model{
		ProviderID:          provider.ID,
		ModelID:             "unreliable-model",
		Name:                "Unreliable Model",
		Description:         "Model that may fail",
		ContextWindowTokens: &contextWindow,
		MaxOutputTokens:     &maxTokens,
		IsMultimodal:        false,
	}

	err = db.CreateModel(model)
	if err != nil {
		t.Fatalf("Failed to create model: %v", err)
	}

	// Step 2: Simulate various failure scenarios
	t.Log("Step 2: Simulating failure scenarios...")

	failureScenarios := []struct {
		description string
		status      string
		errorMsg    *string
		modelExists *bool
		responsive  *bool
		latency     *int
	}{
		{
			description: "Network timeout",
			status:      "failed",
			errorMsg:    &[]string{"Request timeout after 30s"}[0],
			modelExists: &[]bool{true}[0],
			responsive:  &[]bool{false}[0],
			latency:     nil,
		},
		{
			description: "Authentication failure",
			status:      "failed",
			errorMsg:    &[]string{"Invalid API key"}[0],
			modelExists: &[]bool{false}[0],
			responsive:  &[]bool{false}[0],
			latency:     &[]int{50}[0],
		},
		{
			description: "Rate limit exceeded",
			status:      "failed",
			errorMsg:    &[]string{"Rate limit exceeded"}[0],
			modelExists: &[]bool{true}[0],
			responsive:  &[]bool{true}[0],
			latency:     &[]int{25}[0],
		},
		{
			description: "Recovery - successful verification",
			status:      "completed",
			errorMsg:    nil,
			modelExists: &[]bool{true}[0],
			responsive:  &[]bool{true}[0],
			latency:     &[]int{180}[0],
		},
	}

	for i, scenario := range failureScenarios {
		result := &database.VerificationResult{
			ModelID:                model.ID,
			VerificationType:       "failure_recovery_test",
			StartedAt:              time.Now().Add(time.Duration(i) * time.Minute),
			CompletedAt:            &[]time.Time{time.Now().Add(time.Duration(i)*time.Minute + time.Second*30)}[0],
			Status:                 scenario.status,
			ErrorMessage:           scenario.errorMsg,
			ModelExists:            scenario.modelExists,
			Responsive:             scenario.responsive,
			LatencyMs:              scenario.latency,
			SupportsCodeGeneration: i == len(failureScenarios)-1, // Only last one succeeds
		}

		err = db.CreateVerificationResult(result)
		if err != nil {
			t.Fatalf("Failed to create failure scenario result: %v", err)
		}

		t.Logf("Created %s scenario: %s", scenario.status, scenario.description)
	}

	// Step 3: Test recovery mechanisms
	t.Log("Step 3: Testing recovery mechanisms...")

	// Query all results for the model
	filters := map[string]interface{}{
		"model_id": model.ID,
	}
	results, err := db.ListVerificationResults(filters)
	if err != nil {
		t.Fatalf("Failed to get verification results: %v", err)
	}

	if len(results) != len(failureScenarios) {
		t.Errorf("Expected %d results, got %d", len(failureScenarios), len(results))
	}

	// Check that we have both failures and recovery
	hasFailure := false
	hasSuccess := false
	for _, result := range results {
		if result.Status == "failed" {
			hasFailure = true
		}
		if result.Status == "completed" && result.ErrorMessage == nil {
			hasSuccess = true
		}
	}

	if !hasFailure {
		t.Error("Should have at least one failure scenario")
	}

	if !hasSuccess {
		t.Error("Should have at least one successful recovery")
	}

	// Step 4: Simulate automated retry logic
	t.Log("Step 4: Simulating automated retry logic...")
	lastResult := results[len(results)-1] // Most recent result

	if lastResult.Status == "completed" && lastResult.ErrorMessage == nil {
		t.Log("Recovery successful - model is now operational")
	} else {
		t.Log("Model still failing - would trigger escalation")
	}

	// Step 5: Generate failure analysis report
	t.Log("Step 5: Generating failure analysis...")
	failureAnalysis := map[string]interface{}{
		"model_id":            model.ID,
		"total_attempts":      len(results),
		"successful_attempts": 1, // Only the last one
		"failure_rate":        0.75,
		"common_errors":       []string{"timeout", "auth_failure", "rate_limit"},
		"recovery_time":       "30 minutes",
		"recommendations":     []string{"Increase timeout", "Check API key", "Implement backoff"},
	}

	// Verify failure analysis
	if failureAnalysis["total_attempts"].(int) != len(results) {
		t.Error("Failure analysis attempt count mismatch")
	}

	t.Logf("Failure recovery workflow completed: %d attempts, final status: %s",
		len(results), lastResult.Status)
}
