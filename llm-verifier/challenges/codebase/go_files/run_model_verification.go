package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/milosvasic/LLMsVerifier/llm-verifier/client"
)

type VerificationResult struct {
	ModelID       string `json:"model_id"`
	ModelName     string `json:"model_name"`
	ProviderName  string `json:"provider_name"`
	Exists        bool   `json:"exists"`
	ExistsError   string `json:"exists_error,omitempty"`
	Responsive    bool   `json:"responsive"`
	ResponseError string `json:"response_error,omitempty"`
	Latency       string `json:"latency"`
	TTFT          string `json:"time_to_first_token"`
	OverallStatus string `json:"overall_status"`
	TestTime      string `json:"test_time"`
	StatusCode    int    `json:"status_code,omitempty"`
}

type ChallengeResult struct {
	ChallengeName string           `json:"challenge_name"`
	Date          string           `json:"date"`
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
	SkippedCount        int                  `json:"skipped_count"`
	TestTime            string               `json:"test_time"`
}

type ChallengeSummary struct {
	TotalModels      int     `json:"total_models"`
	VerifiedModels   int     `json:"verified_models"`
	ModelsExist      int     `json:"models_exist"`
	ModelsResponsive int     `json:"models_responsive"`
	SuccessRate      float64 `json:"success_rate"`
}

var logger *log.Logger
var httpClient *client.HTTPClient

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
	challengeDir := filepath.Join("challenges", "model_verification_real",
		time.Now().Format("2006"), time.Now().Format("01"), time.Now().Format("02"),
		fmt.Sprintf("%d", timestamp))
	logDir := filepath.Join(challengeDir, "logs")
	resultsDir := filepath.Join(challengeDir, "results")

	logger = initLogger(logDir)
	httpClient = client.NewHTTPClient(30 * time.Second)

	logger.Println("======================================================")
	logger.Println("MODEL VERIFICATION CHALLENGE (REAL API TESTING)")
	logger.Println("======================================================")

	providers, err := loadProviderConfig()
	if err != nil {
		logger.Printf("Failed to load provider config: %v", err)
		logger.Println("Exiting...")
		return
	}

	logger.Printf("Loaded %d providers from discovery results\n", len(providers))
	logger.Printf("Timestamp: %s\n", time.Now().Format(time.RFC3339))

	result := runChallenge(providers)

	if err := saveResults(resultsDir, result); err != nil {
		logger.Printf("Failed to save results: %v", err)
	}

	logger.Println("======================================================")
	logger.Println("CHALLENGE COMPLETE")
	logger.Println("======================================================")
	logger.Printf("Duration: %s\n", result.Duration)
	logger.Printf("Results saved to: %s\n", resultsDir)
}

func runChallenge(providers []provider.ProviderConfig) ChallengeResult {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	startTime := time.Now()
	result := ChallengeResult{
		ChallengeName: "model_verification_real",
		StartTime:     startTime.Format(time.RFC3339),
		Date:          time.Now().Format("2006-01-02"),
	}

	logger.Println("\nStarting model verification with REAL API calls...")

	for _, provider := range providers {
		logger.Printf("\n======================================================")
		logger.Printf("Verifying Provider: %s (%d models)\n", provider.Name, len(provider.Models))
		logger.Printf("======================================================\n")

		providerResult := verifyProvider(ctx, provider)
		result.Providers = append(result.Providers, providerResult)
	}

	result.Summary = generateSummary(result)
	result.EndTime = time.Now().Format(time.RFC3339)
	result.Duration = time.Since(startTime).String()

	return result
}

func verifyProvider(ctx context.Context, provider provider.ProviderConfig) ProviderResult {
	startTime := time.Now()
	result := ProviderResult{
		Name:     provider.Name,
		Type:     provider.Type,
		Endpoint: provider.Endpoint,
		TestTime: time.Now().Format(time.RFC3339),
	}

	for _, model := range provider.Models {
		verification := verifyModel(ctx, provider, model)
		result.VerificationResults = append(result.VerificationResults, verification)

		if verification.OverallStatus == "success" {
			result.SuccessCount++
		} else {
			result.FailedCount++
		}
	}

	duration := time.Since(startTime)
	logger.Printf("Provider %s completed in %s: %d passed, %d failed\n",
		provider.Name, duration, result.SuccessCount, result.FailedCount)

	return result
}

func verifyModel(ctx context.Context, provider provider.ProviderConfig, model provider.ModelConfig) VerificationResult {
	logger.Printf("\n--- Verifying Model: %s ---", model.Name)
	startTime := time.Now()

	verification := VerificationResult{
		ModelID:       model.ID,
		ModelName:     model.Name,
		ProviderName:  provider.Name,
		TestTime:      time.Now().Format(time.RFC3339),
		OverallStatus: "unknown",
	}

	logger.Println("  Test 1: Existence (HTTP HEAD/GET)...")
	existsResp, existsErr := httpClient.TestModelExists(ctx, provider.Name, provider.APIKey, model.ID)
	verification.Exists = existsResp != nil && existsResp.Exists
	if existsErr != nil {
		verification.ExistsError = existsErr.Error()
		verification.StatusCode = existsResp.Status
		logger.Printf("  ✗ Existence failed: %s\n", existsErr.Error())
	} else if existsResp != nil && existsResp.Exists {
		verification.StatusCode = existsResp.Status
		logger.Printf("  ✓ Exists (HTTP %d, latency: %s)\n", verification.StatusCode, existsResp.Latency)
	}

	logger.Println("  Test 2: Responsiveness (HTTP POST with latency)...")
	respResp, respErr := httpClient.TestResponsiveness(ctx, provider.Name, provider.APIKey, model.ID, "test")
	verification.Responsive = respErr == nil && respResp.Success
	if respErr != nil {
		verification.ResponseError = respErr.Error()
		verification.StatusCode = respResp.StatusCode
		logger.Printf("  ✗ Responsiveness failed: %s\n", respErr.Error())
	} else if respResp.Success {
		verification.StatusCode = respResp.StatusCode
		verification.Latency = respResp.TotalTime.String()
		verification.TTFT = respResp.TTFT.String()
		logger.Printf("  ✓ Responsive (TTFT: %s, total: %s)\n", respResp.TTFT, respResp.TotalTime)
	}

	if verification.Exists && verification.Responsive {
		verification.OverallStatus = "success"
		logger.Println("  ✓ Overall status: SUCCESS\n")
	} else if !verification.Exists {
		verification.OverallStatus = "failed"
		logger.Println("  ✗ Overall status: FAILED (model does not exist)\n")
	} else if !verification.Responsive {
		verification.OverallStatus = "failed"
		logger.Println("  ✗ Overall status: FAILED (model not responsive)\n")
	} else {
		verification.OverallStatus = "failed"
		logger.Println("  ✗ Overall status: FAILED (unknown error)\n")
	}

	return verification
}

func generateSummary(result ChallengeResult) ChallengeSummary {
	summary := ChallengeSummary{
		TotalModels:      0,
		VerifiedModels:   0,
		ModelsExist:      0,
		ModelsResponsive: 0,
		SuccessRate:      0.0,
	}

	for _, provider := range result.Providers {
		summary.TotalModels += len(provider.VerificationResults)
		for _, verification := range provider.VerificationResults {
			summary.VerifiedModels++
			if verification.Exists {
				summary.ModelsExist++
			}
			if verification.Responsive {
				summary.ModelsResponsive++
			}
		}
	}

	if summary.TotalModels > 0 {
		summary.SuccessRate = float64(summary.ModelsResponsive) / float64(summary.TotalModels) * 100
	}

	return summary
}

func loadProviderConfig() ([]provider.ProviderConfig, error) {
	discoveryDir := "challenges/providers_models_discovery/20251224"

	dirs, err := os.ReadDir(discoveryDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read discovery dir: %w", err)
	}

	if len(dirs) == 0 {
		return nil, fmt.Errorf("no discovery results found in %s", discoveryDir)
	}

	latestDir := dirs[len(dirs)-1].Name()
	discoveryFile := filepath.Join(discoveryDir, latestDir, "results", "providers_opencode.json")

	data, err := os.ReadFile(discoveryFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read discovery results: %w", err)
	}

	var discoveryData map[string]interface{}
	if err := json.Unmarshal(data, &discoveryData); err != nil {
		return nil, fmt.Errorf("failed to parse discovery JSON: %w", err)
	}

	providersArray, ok := discoveryData["providers"]
	if !ok {
		return nil, fmt.Errorf("providers array not found in discovery data")
	}

	providers, ok := providersArray.([]interface{})
	if !ok {
		return nil, fmt.Errorf("providers is not an array")
	}

	config := make([]provider.ProviderConfig, 0)
	for _, p := range providers {
		provider, ok := p.(map[string]interface{})
		if !ok {
			continue
		}

		name, _ := provider["name"].(string)
		apiEndpoint, _ := provider["api_endpoint"].(string)
		freeToUse, _ := provider["free_to_use"].(bool)

		modelsArray, ok := provider["models"].([]interface{})
		if !ok {
			continue
		}

		models := make([]provider.ModelConfig, 0)
		for _, m := range modelsArray {
			model, ok := m.(map[string]interface{})
			if !ok {
				continue
			}

			id, _ := model["id"].(string)
			modelName, _ := model["name"].(string)
			modelFreeToUse, _ := model["free_to_use"].(bool)

			models = append(models, provider.ModelConfig{
				ID:        id,
				Name:      modelName,
				FreeToUse: modelFreeToUse,
			})
		}

		features, ok := provider["features"].(map[string]interface{})
		var modelFeatures provider.ModelFeatures
		if ok {
			streaming, _ := features["streaming"].(bool)
			functionCalling, _ := features["function_calling"].(bool)
			vision, _ := features["vision"].(bool)
			embeddingsArray, _ := features["embeddings"].([]interface{})
			embeddings := make([]string, 0)
			for _, e := range embeddingsArray {
				if eStr, ok := e.(string); ok {
					embeddings = append(embeddings, eStr)
				}
			}

			modelFeatures = provider.ModelFeatures{
				Streaming:       streaming,
				FunctionCalling: functionCalling,
				Vision:          vision,
				Embeddings:      embeddings,
			}
		}

		config = append(config, provider.ProviderConfig{
			Name:      name,
			Type:      "openai-compatible",
			Endpoint:  apiEndpoint,
			Status:    "success",
			FreeToUse: freeToUse,
			APIKey:    "", // Will be loaded separately
			Features:  modelFeatures,
			Models:    models,
		})
	}

	logger.Printf("Loaded %d providers with %d total models\n", len(config), countModels(config))

	// Load API keys from config
	apiKeys := loadAPIKeys()
	for i := range config {
		providerName := strings.ToLower(config[i].Name)
		if apiKey, ok := apiKeys[providerName]; ok {
			config[i].APIKey = apiKey
			logger.Printf("API key loaded for %s: %s...\n", config[i].Name, strings.Repeat("*", len(apiKey)))
		}
	}

	return config, nil
}

func loadAPIKeys() map[string]string {
	apiKeys := make(map[string]string)

	configFile := "config.yaml.example"
	data, err := os.ReadFile(configFile)
	if err != nil {
		return apiKeys
	}

	if err := yaml.Unmarshal(data, &config); err == nil {
		for providerName, providerData := range config.Providers {
			if p, ok := providerData.(map[string]interface{}); ok {
				if apiKey, ok := p["api_key"]; ok {
					if keyStr, ok := apiKey.(string); ok {
						apiKeys[strings.ToLower(providerName)] = keyStr
					}
				}
			}
		}
	}

	return apiKeys
}

func countModels(config []provider.ProviderConfig) int {
	count := 0
	for _, provider := range config {
		count += len(provider.Models)
	}
	return count
}

func saveResults(resultsDir string, result ChallengeResult) error {
	os.MkdirAll(resultsDir, 0755)

	resultFile := filepath.Join(resultsDir, "verification_results.json")
	resultData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := os.WriteFile(resultFile, resultData, 0644); err != nil {
		return fmt.Errorf("failed to write results: %w", err)
	}

	summaryFile := filepath.Join(resultsDir, "summary.md")
	summary := fmt.Sprintf("# Model Verification Challenge Results (Real API Testing)\n\n")
	summary += fmt.Sprintf("**Date**: %s\n\n", result.Date)
	summary += fmt.Sprintf("**Duration**: %s\n\n", result.Duration)
	summary += "## Providers\n\n"
	for _, provider := range result.Providers {
		summary += fmt.Sprintf("### %s\n", provider.Name)
		summary += fmt.Sprintf("- **Status**: %d tested, %d passed, %d failed\n",
			provider.SuccessCount+provider.FailedCount,
			provider.SuccessCount, provider.FailedCount)
	}
	summary += "\n## Summary\n\n"
	summary += fmt.Sprintf("- **Total Models**: %d\n", result.Summary.TotalModels)
	summary += fmt.Sprintf("- **Models Exist**: %d\n", result.Summary.ModelsExist)
	summary += fmt.Sprintf("- **Models Responsive**: %d\n", result.Summary.ModelsResponsive)
	summary += fmt.Sprintf("- **Success Rate**: %.1f%%\n", result.Summary.SuccessRate)

	if err := os.WriteFile(summaryFile, []byte(summary), 0644); err != nil {
		return fmt.Errorf("failed to write summary: %w", err)
	}

	return nil
}
