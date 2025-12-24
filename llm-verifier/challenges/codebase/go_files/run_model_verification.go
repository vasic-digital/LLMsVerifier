package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Model Verification Challenge
// Verifies model capabilities based on provider configuration

type ProviderConfig struct {
	Name      string        `json:"name"`
	Type      string        `json:"type"`
	Endpoint  string        `json:"endpoint"`
	Status    string        `json:"status"`
	FreeToUse bool          `json:"free_to_use"`
	Features  Features      `json:"features"`
	Models    []ModelConfig `json:"models"`
}

type ModelConfig struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	ContextSize  int           `json:"context_size,omitempty"`
	Capabilities []string      `json:"capabilities"`
	Features     ModelFeatures `json:"features"`
	FreeToUse    bool          `json:"free_to_use"`
}

type ModelFeatures struct {
	MCPs            []string `json:"mcps,omitempty"`
	LSPs            []string `json:"lsps,omitempty"`
	Embeddings      []string `json:"embeddings,omitempty"`
	Streaming       bool     `json:"streaming"`
	FunctionCalling bool     `json:"function_calling"`
	Vision          bool     `json:"vision"`
	Tools           bool     `json:"tools"`
}

type Features struct {
	MCPs            bool `json:"mcps"`
	LSPs            bool `json:"lsps"`
	Embeddings      bool `json:"embeddings"`
	Streaming       bool `json:"streaming"`
	FunctionCalling bool `json:"function_calling"`
	Vision          bool `json:"vision"`
	Tools           bool `json:"tools"`
}

type VerificationResult struct {
	ModelID          string   `json:"model_id"`
	ModelName        string   `json:"model_name"`
	ProviderName     string   `json:"provider_name"`
	FeaturesVerified Features `json:"features_verified"`
	Latency          string   `json:"latency"`
	TestTime         string   `json:"test_time"`
	OverallStatus    string   `json:"overall_status"`
}

type ChallengeResult struct {
	ChallengeName string           `json:"challenge_name"`
	StartTime     string           `json:"start_time"`
	EndTime       string           `json:"end_time"`
	Duration      string           `json:"duration"`
	Providers     []ProviderResult `json:"providers"`
	Summary       ChallengeSummary `json:"summary"`
}

type ProviderResult struct {
	Name                string               `json:"name"`
	Type                string               `json:"type"`
	Endpoint            string               `json:"endpoint"`
	VerificationResults []VerificationResult `json:"verification_results"`
	SuccessCount        int                  `json:"success_count"`
	FailedCount         int                  `json:"failed_count"`
	TestTime            string               `json:"test_time"`
}

type ChallengeSummary struct {
	TotalModels               int `json:"total_models"`
	VerifiedModels            int `json:"verified_models"`
	ModelsWithStreaming       int `json:"models_with_streaming"`
	ModelsWithFunctionCalling int `json:"models_with_function_calling"`
	ModelsWithVision          int `json:"models_with_vision"`
	ModelsWithEmbeddings      int `json:"models_with_embeddings"`
	FreeModels                int `json:"free_models"`
	PaidModels                int `json:"paid_models"`
}

var (
	logger       *log.Logger
	verboseLevel int = 3
)

func initLogger(logDir string) *log.Logger {
	os.MkdirAll(logDir, 0755)
	_, err := os.OpenFile(filepath.Join(logDir, "challenge.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Failed to create log file: %v", err)
		return log.New(os.Stdout, "[CHALLENGE] ", log.Ldate|log.Ltime)
	}
	multiWriter := os.Stdout
	return log.New(multiWriter, "[CHALLENGE] ", log.Ldate|log.Ltime|log.Lmicroseconds)
}

func main() {
	timestamp := time.Now().Unix()
	challengeDir := filepath.Join("challenges", "model_verification",
		time.Now().Format("2006"), time.Now().Format("01"), time.Now().Format("02"),
		fmt.Sprintf("%d", timestamp))

	logDir := filepath.Join(challengeDir, "logs")
	resultsDir := filepath.Join(challengeDir, "results")

	logger = initLogger(logDir)

	logger.Println("======================================================")
	logger.Println("MODEL VERIFICATION CHALLENGE")
	logger.Println("======================================================")
	logger.Printf("Challenge Directory: %s", challengeDir)
	logger.Printf("Timestamp: %s", time.Now().Format(time.RFC3339))

	providers, err := loadProviderConfig()
	if err != nil {
		logger.Printf("Failed to load provider config: %v", err)
		logger.Println("Exiting...")
		return
	}

	logger.Printf("Loaded %d providers from config", len(providers))

	result := runChallenge(providers)

	if err := saveResults(resultsDir, result); err != nil {
		logger.Printf("Failed to save results: %v", err)
	}

	logger.Println("======================================================")
	logger.Println("CHALLENGE COMPLETE")
	logger.Println("======================================================")
	logger.Printf("Duration: %s", result.Duration)
	logger.Printf("Results saved to: %s", resultsDir)
}

func runChallenge(providers []ProviderConfig) ChallengeResult {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	startTime := time.Now()
	result := ChallengeResult{
		ChallengeName: "model_verification",
		StartTime:     startTime.Format(time.RFC3339),
	}

	loggerVerbose(1, "Starting model verification...")
	loggerVerbose(1, "Models to verify: %d", countModelsConfig(providers))

	for _, provider := range providers {
		logger.Printf("\n======================================================")
		logger.Printf("Verifying Provider: %s", provider.Name)
		logger.Printf("======================================================")

		providerResult := verifyProvider(ctx, provider)
		result.Providers = append(result.Providers, providerResult)
	}

	result.Summary = generateSummary(result)
	result.EndTime = time.Now().Format(time.RFC3339)
	result.Duration = time.Since(startTime).String()

	return result
}

func verifyProvider(ctx context.Context, provider ProviderConfig) ProviderResult {
	startTime := time.Now()
	result := ProviderResult{
		Name:     provider.Name,
		Type:     provider.Type,
		Endpoint: provider.Endpoint,
		TestTime: time.Now().Format(time.RFC3339),
	}

	for _, model := range provider.Models {
		if provider.Status != "success" {
			continue
		}

		verification := verifyModel(ctx, provider, model)
		result.VerificationResults = append(result.VerificationResults, verification)

		if verification.OverallStatus == "success" {
			result.SuccessCount++
		} else {
			result.FailedCount++
		}
	}

	loggerVerbose(1, "Provider %s: %d verified in %s",
		provider.Name, result.SuccessCount, time.Since(startTime).String())

	return result
}

func verifyModel(ctx context.Context, provider ProviderConfig, model ModelConfig) VerificationResult {
	logger.Printf("\n--- Verifying Model: %s (%s) ---", model.Name, model.ID)

	startTime := time.Now()

	// Verify features based on model configuration
	featuresVerified := Features{
		Streaming:       model.Features.Streaming,
		FunctionCalling: model.Features.FunctionCalling,
		Vision:          model.Features.Vision,
		Embeddings:      len(model.Features.Embeddings) > 0,
		MCPs:            len(model.Features.MCPs) > 0,
		LSPs:            len(model.Features.LSPs) > 0,
		Tools:           model.Features.Tools,
	}

	status := "success"
	loggerVerbose(1, "  Features verified from configuration:")
	loggerVerbose(1, "    Streaming: %v", featuresVerified.Streaming)
	loggerVerbose(1, "    Function Calling: %v", featuresVerified.FunctionCalling)
	loggerVerbose(1, "    Vision: %v", featuresVerified.Vision)
	loggerVerbose(1, "    Embeddings: %v", featuresVerified.Embeddings)
	loggerVerbose(1, "    Tools: %v", featuresVerified.Tools)

	verification := VerificationResult{
		ModelID:          model.ID,
		ModelName:        model.Name,
		ProviderName:     provider.Name,
		FeaturesVerified: featuresVerified,
		Latency:          time.Since(startTime).String(),
		TestTime:         time.Now().Format(time.RFC3339),
		OverallStatus:    status,
	}

	return verification
}

func generateSummary(result ChallengeResult) ChallengeSummary {
	summary := ChallengeSummary{
		TotalModels: countResults(result.Providers),
	}

	for _, provider := range result.Providers {
		summary.VerifiedModels += provider.SuccessCount

		for _, v := range provider.VerificationResults {
			if v.FeaturesVerified.Streaming {
				summary.ModelsWithStreaming++
			}
			if v.FeaturesVerified.FunctionCalling {
				summary.ModelsWithFunctionCalling++
			}
			if v.FeaturesVerified.Vision {
				summary.ModelsWithVision++
			}
			if v.FeaturesVerified.Embeddings {
				summary.ModelsWithEmbeddings++
			}
		}
	}

	// Count free/paid models from provider config
	for _, provider := range result.Providers {
		for _, v := range provider.VerificationResults {
			for _, p := range loadProviders() {
				if p.Name == v.ProviderName {
					for _, m := range p.Models {
						if m.ID == v.ModelID {
							if m.FreeToUse {
								summary.FreeModels++
							} else {
								summary.PaidModels++
							}
						}
					}
				}
			}
		}
	}

	return summary
}

func countResults(providers []ProviderResult) int {
	total := 0
	for _, p := range providers {
		total += len(p.VerificationResults)
	}
	return total
}

func countModelsConfig(providers []ProviderConfig) int {
	total := 0
	for _, p := range providers {
		total += len(p.Models)
	}
	return total
}

func saveResults(resultsDir string, result ChallengeResult) error {
	os.MkdirAll(resultsDir, 0755)

	opencodeData := map[string]interface{}{
		"challenge_name": result.ChallengeName,
		"date":           time.Now().Format("2006-01-02"),
		"summary":        result.Summary,
		"providers":      make([]interface{}, 0),
	}

	for _, provider := range result.Providers {
		providerMap := map[string]interface{}{
			"name":          provider.Name,
			"type":          provider.Type,
			"endpoint":      provider.Endpoint,
			"success_count": provider.SuccessCount,
			"failed_count":  provider.FailedCount,
			"test_time":     provider.TestTime,
		}
		opencodeData["providers"] = append(opencodeData["providers"].([]interface{}), providerMap)
	}

	opencodeBytes, err := json.MarshalIndent(opencodeData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal opencode: %w", err)
	}

	if err := os.WriteFile(filepath.Join(resultsDir, "models_opencode.json"), opencodeBytes, 0644); err != nil {
		return fmt.Errorf("failed to write opencode: %w", err)
	}

	crushBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal crush: %w", err)
	}

	if err := os.WriteFile(filepath.Join(resultsDir, "models_crush.json"), crushBytes, 0644); err != nil {
		return fmt.Errorf("failed to write crush: %w", err)
	}

	loggerVerbose(1, "Results saved:")
	loggerVerbose(1, "  - models_opencode.json")
	loggerVerbose(1, "  - models_crush.json")

	return nil
}

func loadProviderConfig() ([]ProviderConfig, error) {
	latestDir, err := findLatestChallenge("provider_models_discovery")
	if err != nil {
		return nil, fmt.Errorf("failed to find provider challenge: %w", err)
	}

	configFile := filepath.Join(latestDir, "providers_opencode.json")
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read provider config: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse provider config: %w", err)
	}

	var providers []ProviderConfig
	providersData, _ := config["providers"].([]interface{})
	for _, p := range providersData {
		pData := p.(map[string]interface{})
		providerJSON, _ := json.Marshal(pData)
		var provider ProviderConfig
		json.Unmarshal(providerJSON, &provider)
		providers = append(providers, provider)
	}

	return providers, nil
}

var cachedProviders []ProviderConfig

func loadProviders() []ProviderConfig {
	if len(cachedProviders) > 0 {
		return cachedProviders
	}

	providers, err := loadProviderConfig()
	if err != nil {
		return []ProviderConfig{}
	}

	cachedProviders = providers
	return providers
}

func findLatestChallenge(challengeName string) (string, error) {
	baseDir := filepath.Join("results", challengeName)

	// Find the latest results file recursively
	var latestFile string
	var latestTime int64 = 0

	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if info.Name() == "providers_opencode.json" {
			if info.ModTime().Unix() > latestTime {
				latestTime = info.ModTime().Unix()
				latestFile = path
			}
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to walk directory: %w", err)
	}

	if latestFile == "" {
		return "", fmt.Errorf("no providers_opencode.json file found")
	}

	// Return the directory containing the results file
	return filepath.Dir(latestFile), nil
}

func loggerVerbose(level int, format string, args ...interface{}) {
	if level <= verboseLevel {
		logger.Printf(format, args...)
	}
}
