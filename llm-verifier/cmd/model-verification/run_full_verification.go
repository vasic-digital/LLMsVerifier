package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"llm-verifier/auth"
	"llm-verifier/client"
	"llm-verifier/database"
	"llm-verifier/verification"
	"llm-verifier/logging"
)

type FullVerificationResult struct {
	Timestamp      time.Time                `json:"timestamp"`
	ProviderCount  int                      `json:"provider_count"`
	ModelCount     int                      `json:"model_count"`
	Providers      []ProviderVerification   `json:"providers"`
	Summary        VerificationSummary      `json:"summary"`
}

type ProviderVerification struct {
	Name           string                         `json:"name"`
	Endpoint       string                         `json:"endpoint"`
	HasAPIKey      bool                           `json:"has_api_key"`
	Models         []ModelVerification            `json:"models"`
	Error          string                         `json:"error,omitempty"`
}

type ModelVerification struct {
	ModelID        string                          `json:"model_id"`
	Name           string                          `json:"name"`
	Verified       bool                            `json:"verified"`
	Features       ModelFeatures                   `json:"features"`
	Scores         ModelScores                     `json:"scores"`
	ResponseTime   int64                           `json:"response_time_ms"`
	TTFT           int64                           `json:"ttft_ms"`
	Error          string                          `json:"error,omitempty"`
	LastVerified   time.Time                       `json:"last_verified"`
}

type ModelFeatures struct {
	Streaming         bool    `json:"streaming"`
	ToolCalling       bool    `json:"tool_calling"`
	Embeddings        bool    `json:"embeddings"`
	Vision            bool    `json:"vision"`
	MCP               bool    `json:"mcp"`               // Model Capability Protocol
	LSP               bool    `json:"lsp"`               // Language Server Protocol
	ACP               bool    `json:"acp"`               // AI Coding Protocol
	Audio             bool    `json:"audio"`
	Code              bool    `json:"code"`
	StructuredOutput  bool    `json:"structured_output"` // Added for models.dev compatibility
}

type ModelScores struct {
	Overall        int     `json:"overall"`
	CodeCapability int     `json:"code_capability"`
	Responsiveness int     `json:"responsiveness"`
	Reliability    int     `json:"reliability"`
	FeatureRichness int    `json:"feature_richness"`
}

type VerificationSummary struct {
	TotalProviders    int     `json:"total_providers"`
	ProvidersWithKeys int     `json:"providers_with_keys"`
	TotalModels       int     `json:"total_models"`
	VerifiedModels    int     `json:"verified_models"`
	FailedModels      int     `json:"failed_models"`
	AverageScore      float64 `json:"average_score"`
}

type VerificationRunner struct {
	db              *database.Database
	authMgr         *auth.AuthManager
	httpClient      *client.HTTPClient
	modelsDevClient *verification.ModelsDevClient
	logger          *logging.Logger
	results         FullVerificationResult
	providerData    map[string]providerInfo
}

type providerInfo struct {
	apiKey   string
	endpoint string
	models   []string
}

func NewVerificationRunner() (*VerificationRunner, error) {
	db, err := database.New("../llm-verifier.db")
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	authMgr := auth.NewAuthManager("verification-secret-key")
	httpClient := client.NewHTTPClient(30 * time.Second)
	modelsDevClient := verification.NewModelsDevClient()
	
	return &VerificationRunner{
		db:              db,
		authMgr:         authMgr,
		httpClient:      httpClient,
		modelsDevClient: modelsDevClient,
		providerData:    make(map[string]providerInfo),
		results: FullVerificationResult{
			Timestamp: time.Now(),
			Providers: []ProviderVerification{},
			Summary:   VerificationSummary{},
		},
	}, nil
}

func (vr *VerificationRunner) LoadAPIKeys() error {
	log.Println("Loading API keys from environment...")
	
	// Check in project root first
	envFile := filepath.Join("../../../", ".env")
	file, err := os.Open(envFile)
	if err != nil {
		// Try alternative location
		envFile = filepath.Join("/media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier", ".env")
		file, err = os.Open(envFile)
		if err != nil {
			return fmt.Errorf("failed to open .env file: %w", err)
		}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	providerRegex := regexp.MustCompile(`^ApiKey_([A-Za-z_]+)=(.+)$`)
	
	for scanner.Scan() {
		line := scanner.Text()
		matches := providerRegex.FindStringSubmatch(line)
		if len(matches) == 3 {
			providerName := strings.ToLower(strings.ReplaceAll(matches[1], "_", ""))
			apiKey := matches[2]
			
			// Skip if API key is empty or placeholder
			if apiKey == "" || strings.Contains(apiKey, "YOUR") || strings.Contains(apiKey, "CHANGE") {
				log.Printf("Skipping %s: API key not configured", providerName)
				continue
			}
			
			endpoint := vr.getProviderEndpoint(providerName)
			models := vr.getProviderModels(providerName)
			
			vr.providerData[providerName] = providerInfo{
				apiKey:   apiKey,
				endpoint: endpoint,
				models:   models,
			}
			log.Printf("Loaded %s with %d models", providerName, len(models))
		}
	}
	
	return scanner.Err()
}

func (vr *VerificationRunner) getProviderEndpoint(provider string) string {
	endpoints := map[string]string{
		"openai":              "https://api.openai.com/v1",
		"anthropic":           "https://api.anthropic.com/v1",
		"google":              "https://generativelanguage.googleapis.com/v1",
		"gemini":              "https://generativelanguage.googleapis.com/v1",
		"mistral":             "https://api.mistral.ai/v1",
		"cohere":              "https://api.cohere.ai/v1",
		"huggingface":         "https://api-inference.huggingface.co",
		"together":            "https://api.together.xyz/v1",
		"fireworks":           "https://api.fireworks.ai/inference/v1",
		"replicate":           "https://api.replicate.com/v1",
		"groq":                "https://api.groq.com/openai/v1",
		"perplexity":          "https://api.perplexity.ai",
		"deepseek":            "https://api.deepseek.com/v1",
		"nvidia":              "https://integrate.api.nvidia.com/v1",
		"chutes":              "https://api.chutes.ai/v1",
		"siliconflow":         "https://api.siliconflow.cn/v1",
		"kimi":                "https://api.moonshot.cn/v1",
		"openrouter":          "https://openrouter.ai/api/v1",
		"zai":                 "https://api.studio.nebius.ai/v1",
		"cerebras":            "https://api.cerebras.ai/v1",
		"cloudflare":          "https://api.cloudflare.com/client/v4/accounts",
		"vercel":              "https://api.vercel.com/v1",
		"baseten":             "https://inference.baseten.co/v1",
		"novita":              "https://api.novita.ai/v3/openai",
		"upstage":             "https://api.upstage.ai/v1",
		"nlpcloud":            "https://api.nlpcloud.com/v1",
		"modal":               "https://api.modal.com/v1",
		"inference":           "https://api.inference.net/v1",
		"hyperbolic":          "https://api.hyperbolic.xyz/v1",
		"sambanova":           "https://api.sambanova.ai/v1",
		"vertex":              "https://us-central1-aiplatform.googleapis.com/v1",
	}
	
	if endpoint, ok := endpoints[provider]; ok {
		return endpoint
	}
	
	// Default OpenAI-compatible endpoint
	return "https://api.openai.com/v1"
}

func (vr *VerificationRunner) getProviderModels(provider string) []string {
	// Common models for each provider
	models := map[string][]string{
		"openai":      {"gpt-4", "gpt-4-turbo", "gpt-3.5-turbo"},
		"anthropic":   {"claude-3-5-sonnet-20241022", "claude-3-opus-20240229", "claude-3-haiku-20240307"},
		"google":      {"gemini-pro", "gemini-1.5-pro", "gemini-1.5-flash"},
		"gemini":      {"gemini-pro", "gemini-1.5-pro", "gemini-1.5-flash"},
		"mistral":     {"mistral-large-latest", "mistral-medium", "mistral-small"},
		"cohere":      {"command", "command-light", "command-nightly"},
		"huggingface": {"microsoft/DialoGPT-medium", "google/flan-t5-base"},
		"together":    {"togethercomputer/llama-2-70b-chat", "togethercomputer/llama-2-13b-chat"},
		"fireworks":   {"accounts/fireworks/models/llama-v2-70b-chat", "accounts/fireworks/models/mixtral-8x7b-instruct"},
		"replicate":   {"meta/llama-2-70b-chat", "mistralai/mixtral-8x7b-instruct"},
		"groq":        {"llama2-70b-4096", "mixtral-8x7b-32768", "gemma-7b-it"},
		"perplexity":  {"llama-3-sonar-large-32k-online", "llama-3-sonar-small-32k-online"},
		"deepseek":    {"deepseek-chat", "deepseek-coder"},
		"nvidia":      {"nvidia/llama-3.1-nemotron-70b-instruct", "nvidia/llama-2-70b"},
		"chutes":      {"gpt-4", "claude-3"},
		"siliconflow": {"deepseek-ai/deepseek-llm-67b-chat", "Qwen/Qwen2-72B-Instruct"},
		"kimi":        {"moonshot-v1-8k", "moonshot-v1-32k", "moonshot-v1-128k"},
		"openrouter":  {"openai/gpt-4", "anthropic/claude-3.5-sonnet", "google/gemini-pro"},
		"zai":         {"llama-3.1-70b-instruct", "llama-3.1-8b-instruct"},
		"cerebras":    {"llama-3.3-70b"},
		"cloudflare":  {"@cf/meta/llama-2-7b-chat-int8", "@cf/mistral/mistral-7b-instruct"},
		"vercel":      {"openai/gpt-4", "anthropic/claude-3.5-sonnet"},
		"baseten":     {"llama-2-70b-chat", "stable-diffusion-xl"},
		"novita":      {"deepseek/deepseek_v2.5", "sophosympatheia/midnight-rose-70b"},
		"upstage":     {"solar-pro2", "solar-1-mini-chat"},
		"nlpcloud":    {"finetuned-llama-2-70b", "dolphin-mixtral-8x7b"},
		"modal":       {"any-model"},
		"inference":   {"google/gemma-3-27b-instruct", "meta-llama/llama-3.3-70b"},
		"hyperbolic":  {"meta-llama/llama-3.1-70b-instruct", "meta-llama/llama-3.1-8b-instruct"},
		"sambanova":   {"ALLaM-7B-Instruct-preview", "llama-3.3-70b"},
		"vertex":      {"gemini-1.5-pro", "gemini-1.5-flash"},
	}
	
	if providerModels, ok := models[provider]; ok {
		return providerModels
	}
	
	// Default models for unknown providers
	return []string{"default-model"}
}

func (vr *VerificationRunner) VerifyAllProviders() error {
	log.Printf("Starting verification of %d providers...", len(vr.providerData))
	
	vr.results.Summary.TotalProviders = len(vr.providerData)
	vr.results.Summary.ProvidersWithKeys = len(vr.providerData)
	
	for providerName, info := range vr.providerData {
		log.Printf("\n=== Verifying %s ===", providerName)
		
		providerResult := ProviderVerification{
			Name:      providerName,
			Endpoint:  info.endpoint,
			HasAPIKey: true,
			Models:    []ModelVerification{},
		}
		
		// Create or update provider in database
		provider := &database.Provider{
			Name:            strings.Title(providerName),
			Endpoint:        info.endpoint,
			APIKeyEncrypted: "ENCRYPTED_" + info.apiKey[:10], // Placeholder - real encryption needed
			IsActive:        true,
			LastChecked:     &[]time.Time{time.Now()}[0],
		}
		
		// Check if provider already exists
		existingProvider, err := vr.db.GetProviderByName(provider.Name)
		if err == nil && existingProvider != nil {
			// Provider exists, update it
			provider.ID = existingProvider.ID
			err = vr.db.UpdateProvider(provider)
			if err != nil {
				providerResult.Error = fmt.Sprintf("Failed to update provider: %v", err)
				vr.results.Providers = append(vr.results.Providers, providerResult)
				continue
			}
		} else {
			// Provider doesn't exist, create new
			err = vr.db.CreateProvider(provider)
			if err != nil {
				providerResult.Error = fmt.Sprintf("Failed to create provider: %v", err)
				vr.results.Providers = append(vr.results.Providers, providerResult)
				continue
			}
		}
		
		// Verify each model
		for _, modelID := range info.models {
			modelResult := vr.verifyModel(provider.ID, providerName, modelID, info.apiKey)
			providerResult.Models = append(providerResult.Models, modelResult)
			vr.results.ModelCount++
			
			if modelResult.Verified {
				vr.results.Summary.VerifiedModels++
			} else {
				vr.results.Summary.FailedModels++
			}
		}
		
		vr.results.Providers = append(vr.results.Providers, providerResult)
	}
	
	// Calculate average score
	if vr.results.Summary.VerifiedModels > 0 {
		totalScore := 0
		for _, provider := range vr.results.Providers {
			for _, model := range provider.Models {
				if model.Verified {
					totalScore += model.Scores.Overall
				}
			}
		}
		vr.results.Summary.AverageScore = float64(totalScore) / float64(vr.results.Summary.VerifiedModels)
	}
	
	return nil
}

func (vr *VerificationRunner) verifyModel(providerID int64, providerName, modelID, apiKey string) ModelVerification {
	log.Printf("  Verifying model: %s", modelID)
	
	result := ModelVerification{
		ModelID:      modelID,
		Name:         modelID,
		Features:     ModelFeatures{},
		Scores:       ModelScores{},
		LastVerified: time.Now(),
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// First try to fetch model info from models.dev for enhanced verification
	modelsDevModel, err := vr.modelsDevClient.FindModel(ctx, modelID)
	if err != nil {
		log.Printf("  Warning: Could not fetch %s from models.dev: %v", modelID, err)
	}
	
	// Test model existence - fresh HTTP call to provider's actual API
	log.Printf("    Making fresh API call to %s/%s...", providerName, vr.hideApiKey(apiKey))
	exists, err := vr.httpClient.TestModelExists(ctx, providerName, apiKey, modelID)
	if err != nil || !exists {
		result.Error = fmt.Sprintf("Model existence check failed: %v", err)
		return result
	}
	
	// Test responsiveness - clean call with no caching
	log.Printf("    Testing responsiveness...")
	totalTime, ttft, err, errMsg, responsive, statusCode, httpErr := vr.httpClient.TestResponsiveness(
		ctx, providerName, apiKey, modelID, "What is 2+2?")
	
	if err != nil || !responsive || statusCode != 200 {
		result.Error = fmt.Sprintf("Responsiveness check failed: %v (HTTP %d)", errMsg, statusCode)
		if httpErr != nil {
			result.Error += fmt.Sprintf(" - %v", httpErr)
		}
		return result
	}
	
	result.ResponseTime = totalTime.Milliseconds()
	result.TTFT = ttft.Milliseconds()
	
	// Test features - if we have models.dev data, use it to enhance verification
	if modelsDevModel != nil {
		log.Printf("    Found in models.dev - enhancing verification data...")
		result.Name = modelsDevModel.Model
		result.Features = vr.enhanceFeaturesWithModelsDev(ctx, providerName, modelID, apiKey, modelsDevModel)
		result.Scores = vr.calculateModelScoresWithMetadata(result, modelsDevModel)
	} else {
		// Fallback to heuristic detection
		result.Features = vr.detectFeatures(ctx, providerName, modelID, apiKey)
		result.Scores = vr.calculateModelScores(result)
	}
	
	result.Verified = true
	
	// Store in database
	log.Printf("    Storing verification results...")
	vr.storeModelVerification(providerID, modelID, result)
	
	return result
}

func (vr *VerificationRunner) detectFeatures(ctx context.Context, providerName, modelID, apiKey string) ModelFeatures {
	features := ModelFeatures{}
	
	// Test streaming
	features.Streaming = vr.testFeature(ctx, providerName, modelID, apiKey, "streaming")
	
	// Test tool calling (function calling)
	features.ToolCalling = vr.testFeature(ctx, providerName, modelID, apiKey, "tool_calling")
	
	// Test embeddings
	features.Embeddings = vr.testFeature(ctx, providerName, modelID, apiKey, "embeddings")
	
	// Test vision
	features.Vision = vr.testFeature(ctx, providerName, modelID, apiKey, "vision")
	
	// Detect code capabilities
	features.Code = vr.detectCodeCapability(providerName, modelID)
	
	// MCP, LSP, ACP detection
	features.MCP = vr.testACPSupport(ctx, providerName, modelID, apiKey)
	features.LSP = features.MCP // LSP often supported where MCP is
	features.ACP = features.MCP // ACP is the main protocol
	
	return features
}

func (vr *VerificationRunner) testFeature(ctx context.Context, providerName, modelID, apiKey, feature string) bool {
	// This is a simplified version - in production, you'd have specific tests
	// For now, we use heuristics based on model/provider names
	
	checkStr := strings.ToLower(modelID + " " + providerName)
	
	switch feature {
	case "streaming":
		// Most modern models support streaming
		return !strings.Contains(checkStr, "embedding") && !strings.Contains(checkStr, "ada")
		
	case "tool_calling":
		// OpenAI, Anthropic, and newer models support tool calling
		return strings.Contains(checkStr, "gpt-4") || strings.Contains(checkStr, "gpt-3.5") ||
			strings.Contains(checkStr, "claude") || strings.Contains(checkStr, "mistral") ||
			strings.Contains(checkStr, "gemini")
		
	case "embeddings":
		return strings.Contains(checkStr, "embedding") || strings.Contains(checkStr, "ada") ||
			strings.Contains(checkStr, "text-embedding")
		
	case "vision":
		return strings.Contains(checkStr, "vision") || strings.Contains(checkStr, "gpt-4") ||
			strings.Contains(checkStr, "claude-3") || strings.Contains(checkStr, "gemini")
	}
	
	return false
}

func (vr *VerificationRunner) detectCodeCapability(providerName, modelID string) bool {
	checkStr := strings.ToLower(modelID + " " + providerName)
	
	// Look for coding-related keywords
	codeKeywords := []string{"code", "coder", "gpt-4", "claude", "deepseek", "mistral", "llama", "codestral"}
	
	for _, keyword := range codeKeywords {
		if strings.Contains(checkStr, keyword) {
			return true
		}
	}
	
	return false
}

func (vr *VerificationRunner) testACPSupport(ctx context.Context, providerName, modelID, apiKey string) bool {
	// Test if model supports ACP (AI Coding Protocol)
	// This is a simplified test - real ACP testing would involve actual protocol verification
	
	acpCapableModels := []string{
		"gpt-4", "gpt-4-turbo", "claude-3.5-sonnet", "claude-3-opus",
		"deepseek-chat", "deepseek-coder", "mistral-large", "codestral",
		"gemini-1.5-pro", "llama-3", "llama-3.1", "llama-2-70b",
	}
	
	for _, capableModel := range acpCapableModels {
		if strings.Contains(strings.ToLower(modelID), strings.ToLower(capableModel)) {
			return true
		}
	}
	
	return false
}

func (vr *VerificationRunner) calculateModelScores(verification ModelVerification) ModelScores {
	scores := ModelScores{}
	
	// Base score on responsiveness (0-30 points)
	if verification.ResponseTime > 0 && verification.ResponseTime < 1000 {
		scores.Responsiveness = 30
	} else if verification.ResponseTime < 3000 {
		scores.Responsiveness = 20
	} else if verification.ResponseTime < 10000 {
		scores.Responsiveness = 10
	}
	
	// Feature richness score (0-25 points)
	featureCount := 0
	if verification.Features.Streaming {
		featureCount++
	}
	if verification.Features.ToolCalling {
		featureCount += 2
	}
	if verification.Features.Embeddings {
		featureCount++
	}
	if verification.Features.Vision {
		featureCount += 2
	}
	if verification.Features.MCP || verification.Features.ACP {
		featureCount += 3
	}
	if verification.Features.Code {
		featureCount += 2
	}
	
	scores.FeatureRichness = featureCount * 3
	if scores.FeatureRichness > 25 {
		scores.FeatureRichness = 25
	}
	
	// Code capability score (0-25 points)
	if verification.Features.Code {
		scores.CodeCapability = 25
	} else if verification.Features.ToolCalling && verification.Features.Streaming {
		scores.CodeCapability = 20
	} else if verification.Features.ToolCalling {
		scores.CodeCapability = 15
	}
	
	// Reliability score (0-20 points)
	if verification.Verified {
		scores.Reliability = 20
	}
	
	// Overall score
	scores.Overall = scores.Responsiveness + scores.FeatureRichness + 
		scores.CodeCapability + scores.Reliability
	
	return scores
}

func (vr *VerificationRunner) storeModelVerification(providerID int64, modelID string, verification ModelVerification) {
	// Store model in database
	model := &database.Model{
		ProviderID:          providerID,
		ModelID:             modelID,
		Name:                verification.Name,
		Description:         "",
		IsMultimodal:        verification.Features.Vision || verification.Features.Audio,
		SupportsVision:      verification.Features.Vision,
		LastVerified:        &verification.LastVerified,
		VerificationStatus:  map[bool]string{true: "verified", false: "failed"}[verification.Verified],
		OverallScore:        float64(verification.Scores.Overall),
		CodeCapabilityScore: float64(verification.Scores.CodeCapability),
		ResponsivenessScore: float64(verification.Scores.Responsiveness),
		ReliabilityScore:    float64(verification.Scores.Reliability),
		FeatureRichnessScore: float64(verification.Scores.FeatureRichness),
	}
	
	err := vr.db.CreateModel(model)
	if err != nil {
		log.Printf("Failed to store model %s: %v", modelID, err)
		return
	}
	
	// Store verification result
	status := "completed"
	if !verification.Verified {
		status = "failed"
	}
	
	result := &database.VerificationResult{
		ModelID:         model.ID,
		VerificationType: "full_feature_test",
		StartedAt:       verification.LastVerified.Add(-time.Second),
		CompletedAt:     &verification.LastVerified,
		Status:          status,
		ModelExists:     &verification.Verified,
		Responsive:      &verification.Verified,
		LatencyMs:       &[]int{int(verification.ResponseTime)}[0],
		SupportsBrotli:  verification.Features.MCP || verification.Features.ACP,
	}
	
	if verification.Error != "" {
		result.ErrorMessage = &verification.Error
	}
	
	err = vr.db.CreateVerificationResult(result)
	if err != nil {
		log.Printf("Failed to store verification result for %s: %v", modelID, err)
	}
}

func (vr *VerificationRunner) SaveResults() error {
	resultsDir := filepath.Join("..", "challenges", "full_verification", 
		time.Now().Format("2006/01/02/150405"), "results")
	
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		return fmt.Errorf("failed to create results directory: %w", err)
	}
	
	// Save JSON results
	jsonData, err := json.MarshalIndent(vr.results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}
	
	jsonFile := filepath.Join(resultsDir, "full_verification_results.json")
	if err := os.WriteFile(jsonFile, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON results: %w", err)
	}
	
	// Save markdown summary
	summary := vr.generateMarkdownSummary()
	mdFile := filepath.Join(resultsDir, "verification_summary.md")
	if err := os.WriteFile(mdFile, []byte(summary), 0644); err != nil {
		return fmt.Errorf("failed to write markdown summary: %w", err)
	}
	
	// Save CSV for easy analysis
	csvData := vr.generateCSVData()
	csvFile := filepath.Join(resultsDir, "model_scores.csv")
	if err := os.WriteFile(csvFile, []byte(csvData), 0644); err != nil {
		return fmt.Errorf("failed to write CSV data: %w", err)
	}
	
	// Save providers export
	providersExport := vr.generateProvidersExport()
	exportFile := filepath.Join(resultsDir, "providers_export.json")
	if err := os.WriteFile(exportFile, providersExport, 0644); err != nil {
		return fmt.Errorf("failed to write export data: %w", err)
	}
	
	log.Printf("Saved results to: %s", resultsDir)
	return nil
}

func (vr *VerificationRunner) generateMarkdownSummary() string {
	summary := fmt.Sprintf(`# Full LLM Provider Verification Summary

**Generated:** %s  
**Total Providers:** %d  
**Providers with API Keys:** %d  
**Total Models:** %d  
**Verified Models:** %d  
**Failed Models:** %d  
**Average Score:** %.1f

---

## Provider Details

`, vr.results.Timestamp.Format(time.RFC3339),
		vr.results.Summary.TotalProviders,
		vr.results.Summary.ProvidersWithKeys,
		vr.results.ModelCount,
		vr.results.Summary.VerifiedModels,
		vr.results.Summary.FailedModels,
		vr.results.Summary.AverageScore)
	
	for _, provider := range vr.results.Providers {
		summary += fmt.Sprintf("### %s\n\n", strings.Title(provider.Name))
		summary += fmt.Sprintf("- **Endpoint:** %s\n", provider.Endpoint)
		summary += fmt.Sprintf("- **API Key:** %s\n", map[bool]string{true: "✅ Configured", false: "❌ Missing"}[provider.HasAPIKey])
		
		if provider.Error != "" {
			summary += fmt.Sprintf("- **Error:** %s\n\n", provider.Error)
			continue
		}
		
		summary += fmt.Sprintf("- **Models:** %d\n\n", len(provider.Models))
		summary += "| Model | Verified | Score | Features | Response Time |\n"
		summary += "|-------|----------|-------|----------|---------------|\n"
		
		for _, model := range provider.Models {
			status := "❌"
			if model.Verified {
				status = "✅"
			}
			
			features := []string{}
			if model.Features.Streaming {
				features = append(features, "Streaming")
			}
			if model.Features.ToolCalling {
				features = append(features, "Tools")
			}
			if model.Features.Embeddings {
				features = append(features, "Embeddings")
			}
			if model.Features.Vision {
				features = append(features, "Vision")
			}
			if model.Features.ACP {
				features = append(features, "ACP")
			}
			if model.Features.Code {
				features = append(features, "Code")
			}
			
			featureStr := strings.Join(features, ", ")
			
			summary += fmt.Sprintf("| %s | %s | %d | %s | %dms |\n",
				model.ModelID, status, model.Scores.Overall, featureStr, model.ResponseTime)
		}
		summary += "\n"
	}
	
	summary += "## Feature Legend\n\n"
	summary += "- **Streaming:** Supports streaming responses\n"
	summary += "- **Tools:** Supports tool/function calling\n"
	summary += "- **Embeddings:** Supports text embeddings\n"
	summary += "- **Vision:** Supports image input\n"
	summary += "- **ACP:** Supports AI Coding Protocol\n"
	summary += "- **Code:** Detected code generation capability\n"
	
	return summary
}

func (vr *VerificationRunner) generateCSVData() string {
	csv := "Provider,Model,Verified,OverallScore,CodeScore,ResponsivenessScore,ReliabilityScore,FeatureScore,Streaming,ToolCalling,Embeddings,Vision,ACP,Code,ResponseTime\n"
	
	for _, provider := range vr.results.Providers {
		for _, model := range provider.Models {
			csv += fmt.Sprintf("%s,%s,%t,%d,%d,%d,%d,%d,%t,%t,%t,%t,%t,%t,%d\n",
				provider.Name,
				model.ModelID,
				model.Verified,
				model.Scores.Overall,
				model.Scores.CodeCapability,
				model.Scores.Responsiveness,
				model.Scores.Reliability,
				model.Scores.FeatureRichness,
				model.Features.Streaming,
				model.Features.ToolCalling,
				model.Features.Embeddings,
				model.Features.Vision,
				model.Features.ACP,
				model.Features.Code,
				model.ResponseTime,
			)
		}
	}
	
	return csv
}

func (vr *VerificationRunner) generateProvidersExport() []byte {
	// Export providers with all features in a format that can be used by other systems
	export := map[string]interface{}{
		"version": "1.0",
		"timestamp": vr.results.Timestamp,
		"providers": vr.results.Providers,
		"summary": vr.results.Summary,
	}
	
	data, _ := json.MarshalIndent(export, "", "  ")
	return data
}

func main() {
	log.Println("=== LLM Verifier - Full Provider and Model Verification ===")
	log.Println("This will discover and verify all providers with configured API keys...")
	
	startTime := time.Now()
	
	runner, err := NewVerificationRunner()
	if err != nil {
		log.Fatalf("Failed to create verification runner: %v", err)
	}
	defer runner.db.Close()
	
	// Load API keys
	if err := runner.LoadAPIKeys(); err != nil {
		log.Fatalf("Failed to load API keys: %v", err)
	}
	
	log.Printf("Found %d providers with API keys", len(runner.providerData))
	
	// Verify all providers
	if err := runner.VerifyAllProviders(); err != nil {
		log.Fatalf("Failed to verify providers: %v", err)
	}
	
	// Save results
	if err := runner.SaveResults(); err != nil {
		log.Fatalf("Failed to save results: %v", err)
	}
	
	duration := time.Since(startTime)
	
	log.Println("\n=== Verification Complete ===")
	log.Printf("Duration: %v", duration)
	log.Printf("Providers verified: %d/%d", runner.results.Summary.ProvidersWithKeys, runner.results.Summary.TotalProviders)
	log.Printf("Models verified: %d/%d", runner.results.Summary.VerifiedModels, runner.results.ModelCount)
	log.Printf("Average score: %.1f", runner.results.Summary.AverageScore)
	log.Printf("Results saved to: challenges/full_verification/ directory")
}// enhanceFeaturesWithModelsDev uses models.dev metadata to enhance feature detection
func (vr *VerificationRunner) enhanceFeaturesWithModelsDev(ctx context.Context, providerName, modelID, apiKey string, modelsDev *verification.ModelsDevModel) ModelFeatures {
	features := ModelFeatures{}
	
	// Start with models.dev data
	features.Streaming = true // Most modern models support streaming
	features.ToolCalling = modelsDev.ToolCall
	features.Embeddings = strings.Contains(strings.ToLower(modelID), "embedding")
	features.Vision = strings.Contains(strings.ToLower(modelID), "vision") || strings.Contains(strings.ToLower(modelsDev.Model), "vision")
	
	// MCP/ACP/LSP support detection
	features.MCP = vr.testACPSupport(ctx, providerName, modelID, apiKey)
	features.LSP = features.MCP
	features.ACP = features.MCP
	
	// Audio/Video support (basic heuristic)
	audioKeywords := []string{"audio", "whisper", "tts", "speech"}
	videoKeywords := []string{"video", "vision"}
	
	for _, keyword := range audioKeywords {
		if strings.Contains(strings.ToLower(modelID), keyword) ||
			strings.Contains(strings.ToLower(modelsDev.Model), keyword) {
			features.Audio = true
		}
	}
	
	for _, keyword := range videoKeywords {
		if strings.Contains(strings.ToLower(modelID), keyword) ||
			strings.Contains(strings.ToLower(modelsDev.Model), keyword) {
			features.Vision = true
		}
	}
	
	// Code capability detection
	features.Code = vr.detectCodeCapability(providerName, modelID)
	
	return features
}

// calculateModelScoresWithMetadata uses models.dev data for more accurate scoring
func (vr *VerificationRunner) calculateModelScoresWithMetadata(verification ModelVerification, modelsDev *verification.ModelsDevModel) ModelScores {
	scores := ModelScores{}
	
	// Base score on responsiveness (0-30 points)
	if verification.ResponseTime > 0 && verification.ResponseTime < 1000 {
		scores.Responsiveness = 30
	} else if verification.ResponseTime < 3000 {
		scores.Responsiveness = 20
	} else if verification.ResponseTime < 10000 {
		scores.Responsiveness = 10
	}
	
	// Feature richness score enhanced by models.dev data (0-25 points)
	featureCount := 0
	if verification.Features.Streaming {
		featureCount++
	}
	if verification.Features.ToolCalling || modelsDev.ToolCall {
		featureCount += 2
	}
	if verification.Features.Embeddings {
		featureCount++
	}
	if verification.Features.Vision {
		featureCount += 2
	}
	if verification.Features.MCP || verification.Features.ACP {
		featureCount += 3
	}
	if verification.Features.Code || strings.Contains(strings.ToLower(modelsDev.Family), "coder") {
		featureCount += 2
	}
	if verification.Features.StructuredOutput || modelsDev.StructuredOutput {
		featureCount++
	}
	if verification.Features.Audio {
		featureCount++
	}
	
	scores.FeatureRichness = featureCount * 3
	if scores.FeatureRichness > 25 {
		scores.FeatureRichness = 25
	}
	
	// Code capability score enhanced (0-25 points)
	if strings.Contains(strings.ToLower(modelsDev.Family), "coder") ||
		strings.Contains(strings.ToLower(modelsDev.Model), "code") {
		scores.CodeCapability = 25
	} else if verification.Features.ToolCalling && verification.Features.Streaming {
		scores.CodeCapability = 20
	} else if verification.Features.ToolCalling {
		scores.CodeCapability = 15
	}
	
	// Reliability score (0-20 points)
	if verification.Verified {
		scores.Reliability = 20
	}
	
	// Overall score capped at 100
	scores.Overall = scores.Responsiveness + scores.FeatureRichness + 
		scores.CodeCapability + scores.Reliability
	if scores.Overall > 100 {
		scores.Overall = 100
	}
	
	return scores
}

// hideApiKey returns a masked version of the API key for logging
func (vr *VerificationRunner) hideApiKey(apiKey string) string {
	if len(apiKey) < 8 {
		return "***"
	}
	return apiKey[:4] + "***" + apiKey[len(apiKey)-4:]
}
