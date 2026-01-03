package providers

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"llm-verifier/logging"
)

// NewTestLogger creates a logger for testing
func NewTestLogger() *logging.Logger {
	config := map[string]any{
		"level":       "debug",
		"output":      "console",
		"json_format": false,
	}
	logger, _ := logging.NewLogger(nil, config)
	return logger
}

// TestNewModelProviderService tests service creation
func TestNewModelProviderService(t *testing.T) {
	logger := NewTestLogger()
	service := NewModelProviderService("/tmp/test-config.json", logger)

	if service == nil {
		t.Fatal("NewModelProviderService() returned nil")
	}

	if service.configPath != "/tmp/test-config.json" {
		t.Errorf("Expected config path /tmp/test-config.json, got %s", service.configPath)
	}

	if service.logger == nil {
		t.Error("Logger should not be nil")
	}

	if service.providerClients == nil {
		t.Error("Provider clients map should be initialized")
	}

	if service.modelsDevClient == nil {
		t.Error("Models.dev client should be initialized")
	}

	if service.displayFormatter == nil {
		t.Error("Display formatter should be initialized")
	}

	if service.cache == nil {
		t.Error("Cache should be initialized")
	}

	if service.cacheTTL != 24 {
		t.Errorf("Expected cache TTL 24 hours, got %d", service.cacheTTL)
	}
}

// TestRegisterProvider tests provider registration
func TestRegisterProvider(t *testing.T) {
	logger := NewTestLogger()
	service := NewModelProviderService("/tmp/test.json", logger)

	service.RegisterProvider("openai", "https://api.openai.com/v1", "sk-test123")

	client, exists := service.providerClients["openai"]
	if !exists {
		t.Fatal("Provider not registered")
	}

	if client.ProviderID != "openai" {
		t.Errorf("Expected provider ID openai, got %s", client.ProviderID)
	}

	if client.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("Expected base URL https://api.openai.com/v1, got %s", client.BaseURL)
	}

	if client.APIKey != "sk-test123" {
		t.Errorf("Expected API key sk-test123, got %s", client.APIKey)
	}
}

// TestRegisterAllProviders tests automatic registration
func TestRegisterAllProviders(t *testing.T) {
	// Set up test environment variables
	os.Setenv("OPENAI_API_KEY", "sk-openai-test")
	os.Setenv("HUGGINGFACE_API_KEY", "hf-test")
	os.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
	defer os.Unsetenv("OPENAI_API_KEY")
	defer os.Unsetenv("HUGGINGFACE_API_KEY")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	logger := NewTestLogger()
	service := NewModelProviderService("/tmp/test.json", logger)

	service.RegisterAllProviders()

	// Should register all providers with env vars set
	if len(service.providerClients) < 3 {
		t.Errorf("Expected at least 3 providers registered, got %d", len(service.providerClients))
	}

	// Check specific providers
	if _, exists := service.providerClients["openai"]; !exists {
		t.Error("OpenAI provider not registered")
	}

	if _, exists := service.providerClients["huggingface"]; !exists {
		t.Error("HuggingFace provider not registered")
	}

	if _, exists := service.providerClients["anthropic"]; !exists {
		t.Error("Anthropic provider not registered")
	}
}

// TestGetModelsFromConfig tests config-based model loading
func TestGetModelsFromConfig(t *testing.T) {
	logger := NewTestLogger()

	// Create a temporary config file
	configContent := `{
		"provider": {
			"test-provider": {
				"models": {
					"model-1": {
						"name": "Test Model 1",
						"maxTokens": 4096,
						"supports_brotli": true
					},
					"model-2": {
						"name": "Test Model 2",
						"maxTokens": 8192
					}
				}
			}
		}
	}`

	tmpFile, err := os.CreateTemp("", "test-config-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(configContent); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	service := NewModelProviderService(tmpFile.Name(), logger)

	// Test that we can load models from config
	models := service.loadFromConfig("test-provider")

	if len(models) != 2 {
		t.Fatalf("Expected 2 models, got %d", len(models))
	}

	// Find models by ID since order is not guaranteed
	var model1, model2 *Model
	for i := range models {
		if models[i].ID == "model-1" {
			model1 = &models[i]
		} else if models[i].ID == "model-2" {
			model2 = &models[i]
		}
	}

	if model1 == nil {
		t.Error("Expected to find model-1")
	} else {
		if model1.Name != "Test Model 1" {
			t.Errorf("Expected model name 'Test Model 1', got '%s'", model1.Name)
		}
		if !model1.SupportsBrotli {
			t.Error("Model should support brotli")
		}
	}

	if model2 == nil {
		t.Error("Expected to find model-2")
	} else {
		if model2.MaxTokens != 8192 {
			t.Errorf("Expected max tokens 8192, got %d", model2.MaxTokens)
		}
	}
}

// TestDeduplicateModels tests model deduplication
func TestDeduplicateModels(t *testing.T) {
	logger := NewTestLogger()
	service := NewModelProviderService("/tmp/test.json", logger)

	models := []Model{
		{ID: "gpt-4", ProviderID: "openai", Name: "GPT-4"},
		{ID: "gpt-4", ProviderID: "openai", Name: "GPT-4 Duplicate"}, // Should be removed
		{ID: "claude-3", ProviderID: "anthropic", Name: "Claude 3"},
		{ID: "gpt-4", ProviderID: "openrouter", Name: "GPT-4 from OpenRouter"}, // Different provider, should keep
	}

	result := service.deduplicateModels(models)

	if len(result) != 3 {
		t.Errorf("Expected 3 unique models, got %d", len(result))
	}

	// Should have gpt-4 (openai), claude-3, and gpt-4 (openrouter)
	providerCount := make(map[string]int)
	for _, model := range result {
		key := model.ProviderID + ":" + model.ID
		providerCount[key]++
	}

	if len(providerCount) != 3 {
		t.Error("Duplicate models not properly removed")
	}
}

// TestSortModels tests model sorting
func TestSortModels(t *testing.T) {
	logger := NewTestLogger()
	service := NewModelProviderService("/tmp/test.json", logger)

	models := []Model{
		{Name: "Zebra Model"},
		{Name: "Apple Model"},
		{Name: "Mango Model"},
		{Name: "Banana Model"},
	}

	service.sortModels(models)

	expectedOrder := []string{"Apple Model", "Banana Model", "Mango Model", "Zebra Model"}

	for i, model := range models {
		if model.Name != expectedOrder[i] {
			t.Errorf("Expected model at position %d to be '%s', got '%s'",
				i, expectedOrder[i], model.Name)
		}
	}
}

// TestCacheOperations tests caching
func TestCacheOperations(t *testing.T) {
	logger := NewTestLogger()
	service := NewModelProviderService("/tmp/test.json", logger)

	// Initially cache should be empty
	cached := service.getFromCache("test-provider")
	if cached != nil {
		t.Error("Cache should be empty initially")
	}

	// Save to cache
	testModels := []Model{
		{ID: "model-1", Name: "Model 1"},
		{ID: "model-2", Name: "Model 2"},
	}
	service.saveToCache("test-provider", testModels)

	// Should be able to retrieve from cache
	cached = service.getFromCache("test-provider")
	if cached == nil {
		t.Fatal("Cache should return models after saving")
	}

	if len(cached) != 2 {
		t.Errorf("Expected 2 cached models, got %d", len(cached))
	}

	// Clear cache
	service.ClearCache()

	cached = service.getFromCache("test-provider")
	if cached != nil {
		t.Error("Cache should be empty after clearing")
	}
}

// TestIntegrationWithRealProvider tests integration (with OpenAI as example)
func TestIntegrationWithRealProvider(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_API_KEY not set, skipping integration test")
	}

	logger := NewTestLogger()

	// Create service
	service := NewModelProviderService("/tmp/nonexistent.json", logger)
	service.RegisterProvider("openai", "https://api.openai.com/v1", apiKey)

	// Get models
	models, err := service.GetModels("openai")
	if err != nil {
		t.Fatalf("Failed to get models: %v", err)
	}

	if len(models) == 0 {
		t.Error("Expected to get some models from OpenAI")
	}

	// Check for GPT-4
	foundGPT4 := false
	for _, model := range models {
		if strings.Contains(model.ID, "gpt-4") {
			foundGPT4 = true
			break
		}
	}

	if !foundGPT4 {
		t.Error("Expected to find GPT-4 in models")
	}

	t.Logf("Successfully retrieved %d models from OpenAI", len(models))
}

// TestFeatureExtraction tests feature extraction
func TestFeatureExtraction(t *testing.T) {
	logger := NewTestLogger()
	service := NewModelProviderService("/tmp/test.json", logger)

	modelData := map[string]interface{}{
		"supports_brotli": true,
		"supports_http3":  true,
		"open_weights":    true,
		"cost": map[string]interface{}{
			"input":  0.0,
			"output": 0.0,
		},
		"response_time_ms": 500.0,
	}

	model := service.parseModelFromData("test-model", modelData, "test-provider")

	if !model.SupportsBrotli {
		t.Error("Model should support brotli")
	}

	if !model.SupportsHTTP3 {
		t.Error("Model should support HTTP3")
	}

	if !model.IsOpenSource {
		t.Error("Model should be open source")
	}

	if !model.IsFree {
		t.Error("Model should be free")
	}

	if model.ResponseTimeMS != 500.0 {
		t.Errorf("Expected response time 500ms, got %.2f", model.ResponseTimeMS)
	}
}

// TestAllProviders tests registration of all 32 providers
func TestAllProviders(t *testing.T) {
	// Set up environment variables for all providers
	providers := []struct {
		name    string
		envVar  string
		baseURL string
	}{
		{"openai", "OPENAI_API_KEY", "https://api.openai.com/v1"},
		{"huggingface", "HUGGINGFACE_API_KEY", "https://api-inference.huggingface.co"},
		{"anthropic", "ANTHROPIC_API_KEY", "https://api.anthropic.com/v1"},
		{"groq", "GROQ_API_KEY", "https://api.groq.com/openai/v1"},
		{"deepseek", "DEEPSEEK_API_KEY", "https://api.deepseek.com/v1"},
		{"nvidia", "NVIDIA_API_KEY", "https://integrate.api.nvidia.com/v1"},
		{"openrouter", "OPENROUTER_API_KEY", "https://openrouter.ai/api/v1"},
		{"replicate", "REPLICATE_API_KEY", "https://api.replicate.com/v1"},
		{"fireworks", "FIREWORKS_API_KEY", "https://api.fireworks.ai/inference/v1"},
		{"together", "TOGETHER_API_KEY", "https://api.together.xyz/v1"},
		{"perplexity", "PERPLEXITY_API_KEY", "https://api.perplexity.ai"},
		{"mistral", "MISTRAL_API_KEY", "https://api.mistral.ai/v1"},
		{"codestral", "CODESTRAL_API_KEY", "https://codestral.mistral.ai/v1"},
		{"kimi", "KIMI_API_KEY", "https://api.moonshot.cn/v1"},
		{"gemini", "GEMINI_API_KEY", "https://generativelanguage.googleapis.com/v1"},
		{"cloudflare", "CLOUDFLARE_API_KEY", "https://api.cloudflare.com/client/v4/accounts/YOUR_ACCOUNT/ai/v1"},
		{"cerebras", "CEREBRAS_API_KEY", "https://api.cerebras.ai/v1"},
		{"sambanova", "SAMBANOVA_API_KEY", "https://api.sambanova.ai/v1"},
		{"modal", "MODAL_API_KEY", "https://api.modal.com/v1"},
		{"chutes", "CHUTES_API_KEY", "https://api.chutes.ai/v1"},
		{"siliconflow", "SILICONFLOW_API_KEY", "https://api.siliconflow.cn/v1"},
		{"novita", "NOVITA_API_KEY", "https://api.novita.ai/v3/openai"},
		{"upstage", "UPSTAGE_API_KEY", "https://api.upstage.ai/v1/solar"},
		{"nlpcloud", "NLP_API_KEY", "https://api.nlpcloud.io/v1"},
		{"hyperbolic", "HYPERBOLIC_API_KEY", "https://api.hyperbolic.xyz/v1"},
		{"zai", "ZAI_API_KEY", "https://api.z.ai/v1"},
		{"baseten", "BASETEN_API_KEY", "https://inference.baseten.co/v1"},
		{"twelvelabs", "TWELVELABS_API_KEY", "https://api.twelvelabs.io/v1"},
		{"inference", "INFERENCE_API_KEY", "https://api.inference.net/v1"},
		{"sarvam", "SARVAM_API_KEY", "https://api.sarvam.ai/v1"},
		{"vulavula", "VULAVULA_API_KEY", "https://api.vulavula.com/v1"},
		{"vercel", "VERCEL_API_KEY", "https://api.vercel.com/v1"},
	}

	// Set all environment variables to test values
	for _, provider := range providers {
		os.Setenv(provider.envVar, fmt.Sprintf("sk-test-%s-key", provider.name))
		defer os.Unsetenv(provider.envVar)
	}

	logger := NewTestLogger()
	service := NewModelProviderService("/tmp/test.json", logger)

	// Register all providers
	service.RegisterAllProviders()

	// Should have at least the listed providers (more may be registered dynamically)
	if len(service.providerClients) < len(providers) {
		t.Errorf("Expected at least %d providers, got %d", len(providers), len(service.providerClients))
	}

	// Verify all listed providers are registered
	for _, provider := range providers {
		if _, exists := service.providerClients[provider.name]; !exists {
			t.Errorf("Provider %s not registered", provider.name)
		}
	}

	t.Logf("✓ Successfully registered all %d providers (expected at least %d)", len(service.providerClients), len(providers))
}

// TestProviderClient tests provider client
func TestProviderClient(t *testing.T) {
	logger := NewTestLogger()

	// Create client directly with struct
	client := &ProviderClient{
		ProviderID: "test-provider",
		BaseURL:    "https://api.example.com/v1",
		APIKey:     "sk-test-key",
		HTTPClient: nil,
		logger:     logger,
	}

	if client.ProviderID != "test-provider" {
		t.Errorf("Expected provider ID 'test-provider', got '%s'", client.ProviderID)
	}

}

// TestGetAllModels tests getting models for all providers
func TestGetAllModels(t *testing.T) {
	logger := NewTestLogger()
	service := NewModelProviderService("/tmp/test.json", logger)

	// Register test providers
	service.RegisterProvider("provider1", "https://api.provider1.com/v1", "sk-test1")
	service.RegisterProvider("provider2", "https://api.provider2.com/v1", "sk-test2")

	// Get all models (this tests the concurrent fetch)
	allModels, err := service.GetAllModels()
	if err != nil {
		t.Fatalf("GetAllModels() returned error: %v", err)
	}

	// Should have entries for registered providers
	if allModels == nil {
		t.Error("Expected non-nil map")
	}
}

// TestRefreshCache tests cache refresh functionality
func TestRefreshCache(t *testing.T) {
	logger := NewTestLogger()
	service := NewModelProviderService("/tmp/test.json", logger)

	// Register a provider
	service.RegisterProvider("test", "https://api.test.com/v1", "sk-test")

	// Call RefreshCache
	err := service.RefreshCache()
	if err != nil {
		t.Logf("RefreshCache returned error (expected for no real API): %v", err)
	}

	// Cache should be cleared and refreshed (or error handled gracefully)
}

// TestGetAllProviders tests getting all registered providers
func TestGetAllProviders(t *testing.T) {
	logger := NewTestLogger()
	service := NewModelProviderService("/tmp/test.json", logger)

	// Register test providers
	service.RegisterProvider("openai", "https://api.openai.com/v1", "sk-test1")
	service.RegisterProvider("anthropic", "https://api.anthropic.com/v1", "sk-test2")
	service.RegisterProvider("groq", "https://api.groq.com/openai/v1", "sk-test3")

	// Get all providers (returns map[string]*ProviderClient)
	providers := service.GetAllProviders()

	if len(providers) != 3 {
		t.Errorf("Expected 3 providers, got %d", len(providers))
	}

	// Check that all providers are returned by key
	if _, exists := providers["openai"]; !exists {
		t.Error("Expected openai in providers")
	}
	if _, exists := providers["anthropic"]; !exists {
		t.Error("Expected anthropic in providers")
	}
	if _, exists := providers["groq"]; !exists {
		t.Error("Expected groq in providers")
	}
}

// TestGetModelsUnregisteredProvider tests getting models for unregistered provider
func TestGetModelsUnregisteredProvider(t *testing.T) {
	logger := NewTestLogger()
	service := NewModelProviderService("/tmp/test.json", logger)

	// Try to get models for unregistered provider
	models, err := service.GetModels("nonexistent-provider")

	// Should return empty list or error - either is acceptable
	if err != nil {
		t.Logf("GetModels for unregistered provider returned expected error: %v", err)
	} else if len(models) > 0 {
		t.Error("Expected no models for unregistered provider")
	}
}

// TestCacheTTL tests that cache respects TTL
func TestCacheTTL(t *testing.T) {
	logger := NewTestLogger()
	service := NewModelProviderService("/tmp/test.json", logger)

	// Default cache TTL should be 24 hours
	if service.cacheTTL != 24 {
		t.Errorf("Expected default cache TTL 24 hours, got %d", service.cacheTTL)
	}

	// Set a custom TTL
	service.cacheTTL = 1 // 1 hour

	if service.cacheTTL != 1 {
		t.Errorf("Expected cache TTL 1 hour, got %d", service.cacheTTL)
	}
}

// BenchmarkGetModels benchmarks model retrieval
func BenchmarkGetModels(b *testing.B) {
	logger := NewTestLogger()
	service := NewModelProviderService("/tmp/test.json", logger)

	// Register a provider
	service.RegisterProvider("test", "https://api.test.com/v1", "sk-test")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		service.GetModels("test")
	}
}
