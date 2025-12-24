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

type VerificationResult struct {
	ModelID       string `json:"model_id"`
	ModelName     string `json:"model_name"`
	ProviderName  string `json:"provider_name"`
	Exists        bool   `json:"exists"`
	Responsive    bool   `json:"responsive"`
	Latency       string `json:"latency"`
	OverallStatus string `json:"overall_status"`
	TestTime      string `json:"test_time"`
}

type ChallengeResult struct {
	ChallengeName string       `json:"challenge_name"`
	Date          string       `json:"date"`
	Duration      string       `json:"duration"`
	Providers     []ProviderResult `json:"providers"`
	Summary       ChallengeSummary `json:"summary"`
}

type ProviderResult struct {
	Name                string               `json:"name"`
	VerificationResults []VerificationResult `json:"verification_results"`
	SuccessCount        int                  `json:"success_count"`
	FailedCount         int                  `json:"failed_count"`
	SkippedCount       int                  `json:"skipped_count"`
	TestTime            string               `json:"test_time"`
}

type ChallengeSummary struct {
	TotalModels      int     `json:"total_models"`
	ModelsExist      int     `json:"models_exist"`
	ModelsResponsive int     `json:"models_responsive"`
	SuccessRate     float64 `json:"success_rate"`
}

var logger *log.Logger

func initLogger(logDir string) *log.Logger {
	os.MkdirAll(logDir, 0755)
	logFile, _ := os.OpenFile(filepath.Join(logDir, "challenge.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	defer logFile.Close()
	log.Printf("Initialized logger at: %s\n", logFile.Name())
	return log.New(io.MultiWriter(os.Stdout, logFile), "[CHALLENGE] ", log.Ldate|log.Ltime|log.Lmicroseconds)
}

func main() {
	timestamp := time.Now().Unix()
	challengeDir := filepath.Join("challenges", "model_verification_simple",
		time.Now().Format("2006"), time.Now().Format("01"), time.Now().Format("02"),
		fmt.Sprintf("%d", timestamp))
	logDir := filepath.Join(challengeDir, "logs")
	resultsDir := filepath.Join(challengeDir, "results")

	logger = initLogger(logDir)

	logger.Println("======================================================")
	logger.Println("MODEL VERIFICATION CHALLENGE (SIMPLE VERSION)")
	logger.Println("======================================================")

	providers := loadProviderConfig()
	logger.Printf("Loaded %d providers, starting verification\n", len(providers))

	startTime := time.Now()
	result := runChallenge(providers, providers)

	if err := saveResults(resultsDir, result); err != nil {
		logger.Printf("Failed to save results: %v\n", err)
	}

	logger.Println("======================================================")
	logger.Println("CHALLENGE COMPLETE")
	logger.Println("======================================================")
	logger.Printf("Duration: %s\n", time.Since(startTime).String())
	logger.Printf("Results saved to: %s\n", resultsDir)
}

func runChallenge(providers []ProviderConfig, providers []ProviderResult) ChallengeResult {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	result := ChallengeResult{
		ChallengeName: "model_verification_simple",
		StartTime:     time.Now().Format(time.RFC3339),
		Date:          time.Now().Format("2006-01-02"),
	}

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

func verifyProvider(ctx context.Context, provider ProviderConfig, providers []ProviderResult) ProviderResult {
	startTime := time.Now()
	result := ProviderResult{
		Name:     provider.Name,
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
	logger.Printf("Provider %s: %d verified, %d failed, %d skipped (duration: %s)\n",
		provider.Name, result.SuccessCount, result.FailedCount, result.SkippedCount, duration.String())

	return result
}

func verifyModel(ctx context.Context, provider ProviderConfig, providers []ProviderResult) VerificationResult {
	logger.Printf("\n--- Verifying Model: %s (%s) ---\n", model.Name, model.ID)

	verification := VerificationResult{
		ModelID:       model.ID,
		ModelName:     model.Name,
		ProviderName:  provider.Name,
		TestTime:      time.Now().Format(time.RFC3339),
		OverallStatus: "simulated_success",
	}

	logger.Printf("  Status: %s\n", verification.OverallStatus)
	logger.Printf("  Exists: %v\n", verification.Exists)
	logger.Printf("  Responsive: %v\n", verification.Responsive)
	logger.Printf("  Latency: %s\n", verification.Latency)

	return verification
}

func generateSummary(result ChallengeResult) ChallengeSummary {
	summary := ChallengeSummary{
		TotalModels:      0,
		ModelsExist:      0,
		ModelsResponsive: 0,
		SuccessRate:     0.0,
	}

	for _, provider := range result.Providers {
		for _, verification := range provider.VerificationResults {
			summary.TotalModels++

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

func countModels(providers []ProviderConfig) int {
	count := 0
	for _, provider := range providers {
		count += len(provider.Models)
	}
	return count
}

func loadProviderConfig() []ProviderConfig {
	providers := []ProviderConfig{
		{
			Name:      "huggingface",
			Type:      "openai-compatible",
			Endpoint:  "https://api-inference.huggingface.co",
			Models: []ModelConfig{
				{ID: "sentence-transformers/all-MiniLM-L6-v2", Name: "Sentence Transformers MiniLM L6 v2", FreeToUse: true},
				{ID: "Falconsai/nsfw_image_detection", Name: "NSFW Image Detection", FreeToUse: true},
				{ID: "openai/clip-vit-base-patch32", Name: "CLIP ViT Base Patch32", FreeToUse: true},
			},
		},
		{
			Name:      "deepseek",
			Type:      "openai-compatible",
			Endpoint:  "https://api.deepseek.com",
			Models: []ModelConfig{
				{ID: "deepseek-chat", Name: "DeepSeek Chat", FreeToUse: false},
				{ID: "deepseek-coder", Name: "DeepSeek Coder", FreeToUse: false},
			},
		},
	}

	return providers
}

func saveResults(resultsDir string, result ChallengeResult) error {
	os.MkdirAll(resultsDir, 0755)

	resultFile := filepath.Join(resultsDir, "verification_results.json")
	resultData, _ := json.MarshalIndent(result, "", "  ")
	
	if err := os.WriteFile(resultFile, resultData, 0644); err != nil {
		return fmt.Errorf("failed to write results: %w", err)
	}

	summaryFile := filepath.Join(resultsDir, "summary.md")
	summary := fmt.Sprintf("# Model Verification Challenge Results (Simple Version)\n\n")
	summary += fmt.Sprintf("**Date**: %s\n\n", result.Date)
	summary += fmt.Sprintf("**Duration**: %s\n\n", result.Duration)
	summary += "## Providers\n\n"
	for _, provider := range result.Providers {
		summary += fmt.Sprintf("### %s\n", provider.Name)
		summary += fmt.Sprintf("- **Status**: %d tested, %d passed, %d failed\n",
			provider.SuccessCount+provider.FailedCount+provider.SkippedCount,
			provider.SuccessCount, provider.FailedCount, provider.SkippedCount)
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

type ModelConfig struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	FreeToUse bool   `json:"free_to_use"`
}
