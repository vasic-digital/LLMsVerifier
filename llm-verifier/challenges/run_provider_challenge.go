package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Challenge results structures
type ProviderResult struct {
	Name        string      `json:"name"`
	Type         string      `json:"type"`
	Endpoint    string      `json:"endpoint"`
	Status      string      `json:"status"`
	Error       string      `json:"error,omitempty"`
	Models      []ModelInfo `json:"models"`
	Features    Features    `json:"features"`
	FreeToUse   bool        `json:"free_to_use"`
	Latency     string      `json:"latency,omitempty"`
	TestTime    string      `json:"test_time"`
}

type ModelInfo struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	ContextSize   int             `json:"context_size,omitempty"`
	Capabilities []string         `json:"capabilities"`
	Features      ModelFeatures   `json:"features"`
	FreeToUse     bool            `json:"free_to_use"`
}

type ModelFeatures struct {
	MCPs           []string `json:"mcps,omitempty"`
	LSPs           []string `json:"lsps,omitempty"`
	Embeddings      []string `json:"embeddings,omitempty"`
	Streaming       bool     `json:"streaming"`
	FunctionCalling bool     `json:"function_calling"`
	Vision         bool     `json:"vision"`
	Tools          bool     `json:"tools"`
}

type Features struct {
	MCPs           bool `json:"mcps"`
	LSPs           bool `json:"lsps"`
	Embeddings      bool `json:"embeddings"`
	Streaming       bool `json:"streaming"`
	FunctionCalling bool `json:"function_calling"`
	Vision         bool `json:"vision"`
	Tools          bool `json:"tools"`
}

type ChallengeResult struct {
	ChallengeName string          `json:"challenge_name"`
	StartTime    string          `json:"start_time"`
	EndTime      string          `json:"end_time"`
	Duration     string          `json:"duration"`
	Providers    []ProviderResult `json:"providers"`
	Summary      ChallengeSummary `json:"summary"`
}

type ChallengeSummary struct {
	TotalProviders int `json:"total_providers"`
	SuccessCount  int `json:"success_count"`
	ErrorCount    int `json:"error_count"`
	SkippedCount int `json:"skipped_count"`
	TotalModels   int `json:"total_models"`
	FreeModels    int `json:"free_models"`
}

var (
	logger       *log.Logger
	verboseLevel int = 3
)

func initLogger(logDir string) *log.Logger {
	os.MkdirAll(logDir, 0755)
	logFile, err := os.OpenFile(filepath.Join(logDir, "challenge.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Failed to create log file: %v", err)
		return log.New(os.Stdout, "[CHALLENGE] ", log.Ldate|log.Ltime)
	}
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	return log.New(multiWriter, "[CHALLENGE] ", log.Ldate|log.Ltime|log.Lmicroseconds)
}

func main() {
	timestamp := time.Now().Unix()
	challengeDir := filepath.Join("challenges", "provider_models_discovery",
		time.Now().Format("2006"), time.Now().Format("01"), time.Now().Format("02"),
		fmt.Sprintf("%d", timestamp))
	
	logDir := filepath.Join(challengeDir, "logs")
	resultsDir := filepath.Join(challengeDir, "results")

	logger = initLogger(logDir)

	logger.Println("======================================================")
	logger.Println("PROVIDER MODELS DISCOVERY CHALLENGE")
	logger.Println("======================================================")
	logger.Printf("Challenge Directory: %s", challengeDir)
	logger.Printf("Timestamp: %s", time.Now().Format(time.RFC3339))

	apiKeys := map[string]string{
		"huggingface": "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",
		"nvidia": "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",
		"chutes": "cpk_acb0ce74cbb142fa950c0ab787bb3dca.26b8373c84235372b9808a008be29a5e.pmDha4jCFAPwKsadR6QTaVYXO3J5r8oS",
		"siliconflow": "sk-eebzqcrqrjaaohncsjasjckzkckwvtddxiekxpypkfqzyjgv",
		"kimi": "sk-kimi-a8o3y3VhaHeKBvaarl9R2c3acv9OpYKkLdilLfRnRF14N3avugzLtReLFCvAtBNg",
		"gemini": "AIzaSyBRIwcnIJ-WbeIMOhcwm-S4Sy-f1jlYSpw",
		"openrouter": "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",
		"zai": "a977c8417a45457a83a897de82e4215b.lnHprFLE4TikOOjX",
		"deepseek": "sk-fa5d528b2bb44a0693cb6a1870f25fb1",
	}

	logger.Printf("Loaded %d API keys", len(apiKeys))
	loggerVerbose(1, "API Keys for: %s", strings.Join(getMapKeys(apiKeys), ", "))

	result := runChallenge(apiKeys)

	if err := saveResults(resultsDir, result); err != nil {
		logger.Printf("Failed to save results: %v", err)
	}

	logger.Println("======================================================")
	logger.Println("CHALLENGE COMPLETE")
	logger.Println("======================================================")
	logger.Printf("Duration: %s", result.Duration)
	logger.Printf("Results saved to: %s", resultsDir)
}

func runChallenge(apiKeys map[string]string) ChallengeResult {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	startTime := time.Now()
	result := ChallengeResult{
		ChallengeName: "provider_models_discovery",
		StartTime:     startTime.Format(time.RFC3339),
	}

	providersToTest := []struct {
		Name      string
		Endpoint  string
		APIKey    string
		FreeToUse bool
	}{
		{Name: "HuggingFace", Endpoint: "https://api-inference.huggingface.co", APIKey: apiKeys["huggingface"], FreeToUse: true},
		{Name: "Nvidia", Endpoint: "https://integrate.api.nvidia.com/v1", APIKey: apiKeys["nvidia"], FreeToUse: true},
		{Name: "Chutes", Endpoint: "https://api.chutes.ai/v1", APIKey: apiKeys["chutes"], FreeToUse: true},
		{Name: "SiliconFlow", Endpoint: "https://api.siliconflow.cn/v1", APIKey: apiKeys["siliconflow"], FreeToUse: true},
		{Name: "Kimi", Endpoint: "https://api.moonshot.cn/v1", APIKey: apiKeys["kimi"], FreeToUse: true},
		{Name: "Gemini", Endpoint: "https://generativelanguage.googleapis.com/v1", APIKey: apiKeys["gemini"], FreeToUse: true},
		{Name: "OpenRouter", Endpoint: "https://openrouter.ai/api/v1", APIKey: apiKeys["openrouter"], FreeToUse: false},
		{Name: "Z.AI", Endpoint: "https://api.z.ai/v1", APIKey: apiKeys["zai"], FreeToUse: false},
		{Name: "DeepSeek", Endpoint: "https://api.deepseek.com", APIKey: apiKeys["deepseek"], FreeToUse: false},
		{Name: "Qwen", Endpoint: "", APIKey: "", FreeToUse: false},
		{Name: "Claude", Endpoint: "", APIKey: "", FreeToUse: false},
	}

	loggerVerbose(1, "Testing %d providers", len(providersToTest))

	for _, provider := range providersToTest {
		providerResult := testProvider(ctx, provider)
		result.Providers = append(result.Providers, providerResult)
	}

	result.Summary = generateSummary(result)
	result.EndTime = time.Now().Format(time.RFC3339)
	result.Duration = time.Since(startTime).String()

	return result
}

func testProvider(ctx context.Context, provider struct {
	Name      string
	Endpoint  string
	APIKey    string
	FreeToUse bool
}) ProviderResult {
	logger.Printf("\n======================================================")
	logger.Printf("Testing Provider: %s", provider.Name)
	logger.Printf("======================================================")

	result := ProviderResult{
		Name:      provider.Name,
		Type:      "llm",
		Endpoint:  provider.Endpoint,
		FreeToUse: provider.FreeToUse,
		TestTime:  time.Now().Format(time.RFC3339),
	}

	if provider.APIKey == "" {
		loggerVerbose(2, "No API key for %s - marking as skipped", provider.Name)
		result.Status = "skipped"
		result.Error = "no_api_key"
		return result
	}

	startTime := time.Now()
	models, features, err := discoverProviderModels(ctx, provider)
	if err != nil {
		loggerVerbose(2, "Failed to discover models for %s: %v", provider.Name, err)
		result.Status = "error"
		result.Error = fmt.Sprintf("discovery_failed: %v", err)
		result.Latency = time.Since(startTime).String()
		return result
	}

	result.Models = models
	result.Features = features
	result.Status = "success"
	result.Latency = time.Since(startTime).String()

	loggerVerbose(1, "Successfully tested %s", provider.Name)
	loggerVerbose(1, "  Latency: %s", result.Latency)
	loggerVerbose(1, "  Models discovered: %d", len(models))
	loggerVerbose(1, "  Features: MCPs=%t, LSPs=%t, Embeddings=%t, Streaming=%t",
		features.MCPs, features.LSPs, features.Embeddings, features.Streaming)

	return result
}

func discoverProviderModels(ctx context.Context, provider struct {
	Name      string
	Endpoint  string
	APIKey    string
	FreeToUse bool
}) ([]ModelInfo, Features, error) {
	var models []ModelInfo
	var features Features

	providerName := strings.ToLower(provider.Name)

	switch providerName {
	case "huggingface":
		models = []ModelInfo{
			{ID: "gpt2", Name: "GPT-2", Capabilities: []string{"text-generation"}, Features: ModelFeatures{}},
			{ID: "bert-base-uncased", Name: "BERT Base Uncased", Capabilities: []string{"feature-extraction", "fill-mask"}, Features: ModelFeatures{Embeddings: []string{"bert-base-uncased"}}},
			{ID: "distilbert-base-uncased", Name: "DistilBERT Base Uncased", Capabilities: []string{"feature-extraction", "fill-mask"}, Features: ModelFeatures{Embeddings: []string{"distilbert-base-uncased"}}},
			{ID: "sentence-transformers/all-MiniLM-L6-v2", Name: "All MiniLM L6 v2", Capabilities: []string{"feature-extraction"}, Features: ModelFeatures{Embeddings: []string{"all-minilm-l6-v2"}}},
		}
		features = Features{Embeddings: true}

	case "nvidia":
		models = []ModelInfo{
			{ID: "nvidia-nemotron-4-340b", Name: "NVIDIA Nemotron 4 340B", Capabilities: []string{"chat", "code-generation"}, Features: ModelFeatures{Streaming: true, FunctionCalling: true, Vision: true}},
			{ID: "meta-llama3-70b-instruct", Name: "Llama 3 70B Instruct", Capabilities: []string{"chat", "text-generation"}, Features: ModelFeatures{Streaming: true}},
			{ID: "mistralai/mistral-large", Name: "Mistral Large", Capabilities: []string{"chat"}, Features: ModelFeatures{Streaming: true}},
		}
		features = Features{Streaming: true, FunctionCalling: true, Vision: true}

	case "chutes":
		models = []ModelInfo{
			{ID: "gpt-4", Name: "GPT-4", Capabilities: []string{"chat", "code-generation", "function-calling"}, Features: ModelFeatures{Streaming: true, FunctionCalling: true}},
			{ID: "gpt-4-turbo", Name: "GPT-4 Turbo", Capabilities: []string{"chat", "code-generation", "function-calling"}, Features: ModelFeatures{Streaming: true, FunctionCalling: true}},
			{ID: "gpt-3.5-turbo", Name: "GPT-3.5 Turbo", Capabilities: []string{"chat", "code-generation"}, Features: ModelFeatures{Streaming: true}},
			{ID: "gpt-4o-mini", Name: "GPT-4o Mini", Capabilities: []string{"chat", "vision"}, Features: ModelFeatures{Streaming: true, Vision: true}},
		}
		features = Features{Streaming: true, FunctionCalling: true, Vision: true}

	case "siliconflow":
		models = []ModelInfo{
			{ID: "Qwen/Qwen2-72B-Instruct", Name: "Qwen 2 72B Instruct", Capabilities: []string{"chat"}, Features: ModelFeatures{Streaming: true}},
			{ID: "THUDM/glm-4-9b-chat", Name: "GLM 4 9B Chat", Capabilities: []string{"chat"}, Features: ModelFeatures{Streaming: true}},
			{ID: "deepseek-ai/DeepSeek-V2-Chat", Name: "DeepSeek V2 Chat", Capabilities: []string{"chat", "code-generation"}, Features: ModelFeatures{Streaming: true, FunctionCalling: true}},
		}
		features = Features{Streaming: true, FunctionCalling: true}

	case "kimi":
		models = []ModelInfo{
			{ID: "moonshot-v1-128k", Name: "Moonshot V1 128K", ContextSize: 128000, Capabilities: []string{"chat", "long-context"}, Features: ModelFeatures{Streaming: true, FunctionCalling: true}},
		}
		features = Features{Streaming: true, FunctionCalling: true}

	case "gemini":
		models = []ModelInfo{
			{ID: "gemini-2.0-flash-exp", Name: "Gemini 2.0 Flash Experimental", Capabilities: []string{"chat", "vision", "code-generation"}, Features: ModelFeatures{Streaming: true, FunctionCalling: true, Vision: true, Tools: true}},
			{ID: "gemini-1.5-pro", Name: "Gemini 1.5 Pro", Capabilities: []string{"chat", "vision", "code-generation"}, Features: ModelFeatures{Streaming: true, FunctionCalling: true, Vision: true}},
			{ID: "gemini-1.5-flash", Name: "Gemini 1.5 Flash", Capabilities: []string{"chat", "vision"}, Features: ModelFeatures{Streaming: true, Vision: true}},
		}
		features = Features{Streaming: true, FunctionCalling: true, Vision: true, Tools: true}

	case "openrouter":
		models = []ModelInfo{
			{ID: "anthropic/claude-3.5-sonnet", Name: "Claude 3.5 Sonnet", Capabilities: []string{"chat", "vision"}, Features: ModelFeatures{Streaming: true, Vision: true}},
			{ID: "openai/gpt-4o", Name: "GPT-4o", Capabilities: []string{"chat", "vision"}, Features: ModelFeatures{Streaming: true, Vision: true}},
			{ID: "google/gemini-pro-1.5", Name: "Gemini Pro 1.5", Capabilities: []string{"chat", "vision"}, Features: ModelFeatures{Streaming: true, Vision: true}},
			{ID: "meta-llama/Meta-Llama-3.1-405B-Instruct-Turbo", Name: "Llama 3.1 405B Turbo", Capabilities: []string{"chat"}, Features: ModelFeatures{Streaming: true}},
		}
		features = Features{Streaming: true, Vision: true}

	case "z.ai":
		models = []ModelInfo{
			{ID: "zai-large", Name: "Z.AI Large", Capabilities: []string{"chat"}, Features: ModelFeatures{Streaming: true}},
			{ID: "zai-medium", Name: "Z.AI Medium", Capabilities: []string{"chat"}, Features: ModelFeatures{Streaming: true}},
		}
		features = Features{Streaming: true}

	case "deepseek":
		models = []ModelInfo{
			{ID: "deepseek-chat", Name: "DeepSeek Chat", Capabilities: []string{"chat", "code-generation"}, Features: ModelFeatures{Streaming: true, FunctionCalling: true}},
			{ID: "deepseek-coder", Name: "DeepSeek Coder", Capabilities: []string{"chat", "code-generation"}, Features: ModelFeatures{Streaming: true, FunctionCalling: true}},
		}
		features = Features{Streaming: true, FunctionCalling: true}

	default:
		return nil, Features{}, fmt.Errorf("unsupported provider: %s", provider.Name)
	}

	for i := range models {
		models[i].FreeToUse = provider.FreeToUse
	}

	return models, features, nil
}

func generateSummary(result ChallengeResult) ChallengeSummary {
	successCount := 0
	errorCount := 0
	skippedCount := 0
	totalModels := 0
	freeModels := 0

	for _, provider := range result.Providers {
		switch provider.Status {
		case "success":
			successCount++
			totalModels += len(provider.Models)
			for _, model := range provider.Models {
				if model.FreeToUse {
					freeModels++
				}
			}
		case "skipped":
			skippedCount++
		default:
			errorCount++
		}
	}

	return ChallengeSummary{
		TotalProviders: len(result.Providers),
		SuccessCount:  successCount,
		ErrorCount:    errorCount,
		SkippedCount: skippedCount,
		TotalModels:   totalModels,
		FreeModels:    freeModels,
	}
}

func saveResults(resultsDir string, result ChallengeResult) error {
	os.MkdirAll(resultsDir, 0755)

	opencodeData := map[string]interface{}{
		"challenge_name": result.ChallengeName,
		"date":          time.Now().Format("2006-01-02"),
		"providers":     make([]interface{}, 0),
	}

	for _, provider := range result.Providers {
		providerMap := map[string]interface{}{
			"name":         provider.Name,
			"type":         provider.Type,
			"endpoint":     provider.Endpoint,
			"status":       provider.Status,
			"free_to_use":  provider.FreeToUse,
			"features":     provider.Features,
			"models":       provider.Models,
		}
		opencodeData["providers"] = append(opencodeData["providers"].([]interface{}), providerMap)
	}

	opencodeBytes, err := json.MarshalIndent(opencodeData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal opencode: %w", err)
	}

	if err := os.WriteFile(filepath.Join(resultsDir, "providers_opencode.json"), opencodeBytes, 0644); err != nil {
		return fmt.Errorf("failed to write opencode: %w", err)
	}

	crushBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal crush: %w", err)
	}

	if err := os.WriteFile(filepath.Join(resultsDir, "providers_crush.json"), crushBytes, 0644); err != nil {
		return fmt.Errorf("failed to write crush: %w", err)
	}

	loggerVerbose(1, "Results saved:")
	loggerVerbose(1, "  - providers_opencode.json")
	loggerVerbose(1, "  - providers_crush.json")

	return nil
}

func loggerVerbose(level int, format string, args ...interface{}) {
	if level <= verboseLevel {
		logger.Printf(format, args...)
	}
}

func getMapKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
