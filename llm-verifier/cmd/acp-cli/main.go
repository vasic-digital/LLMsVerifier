package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"llm-verifier/config"
	"llm-verifier/llmverifier"
)

var (
	cfgFile      string
	outputFormat string
	verbose      bool
	concurrent   int
	modelsList   string
	modelsFile   string
)

// ACPMockClient implements a mock client for testing
type ACPMockClient struct {
	Provider string
}

// ChatCompletion implements the chat completion interface
func (c *ACPMockClient) ChatCompletion(ctx context.Context, request llmverifier.ChatCompletionRequest) (*llmverifier.ChatCompletionResponse, error) {
	return &llmverifier.ChatCompletionResponse{
		Choices: []llmverifier.ChatCompletionChoice{
			{
				Index: 0,
				Message: llmverifier.Message{
					Role:    "assistant",
					Content: fmt.Sprintf("ACP response from %s", c.Provider),
				},
				FinishReason: "stop",
			},
		},
	}, nil
}

// BatchResult represents the result of a batch verification
type BatchResult struct {
	Model     string        `json:"model"`
	Provider  string        `json:"provider"`
	Supported bool          `json:"supported"`
	Score     float64       `json:"score"`
	Duration  time.Duration `json:"duration"`
}

// createProviderClient creates a client for the specified provider
func createProviderClient(provider string, cfg *config.Config) (*llmverifier.LLMClient, error) {
	// This is a simplified implementation - in production this would use proper config
	var apiKey string
	var baseURL string

	switch provider {
	case "openai":
		apiKey = os.Getenv("OPENAI_API_KEY")
		baseURL = "https://api.openai.com/v1"
	case "anthropic":
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
		baseURL = "https://api.anthropic.com"
	case "deepseek":
		apiKey = os.Getenv("DEEPSEEK_API_KEY")
		baseURL = "https://api.deepseek.com"
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	if apiKey == "" {
		return nil, fmt.Errorf("API key not found for provider: %s", provider)
	}

	return llmverifier.NewLLMClient(baseURL, apiKey, 30*time.Second), nil
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "acp-cli",
		Short: "ACP (Advanced Capability Protocol) CLI tool",
		Long:  `A CLI tool for testing and verifying ACP implementations in LLM providers.`,
	}

	var verifyCmd = &cobra.Command{
		Use:   "verify",
		Short: "Verify ACP support for a specific model",
		RunE:  runVerify,
	}

	var batchCmd = &cobra.Command{
		Use:   "batch",
		Short: "Run batch verification across multiple models",
		RunE:  runBatch,
	}

	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List available models",
		RunE:  runList,
	}

	// Add flags
	verifyCmd.Flags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.acp-cli.yaml)")
	verifyCmd.Flags().StringVar(&outputFormat, "output", "json", "output format (json, yaml, table)")
	verifyCmd.Flags().BoolVar(&verbose, "verbose", false, "verbose output")
	verifyCmd.Flags().StringVar(&modelsList, "models", "", "comma-separated list of models to test")

	batchCmd.Flags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.acp-cli.yaml)")
	batchCmd.Flags().StringVar(&outputFormat, "output", "json", "output format (json, yaml, table)")
	batchCmd.Flags().BoolVar(&verbose, "verbose", false, "verbose output")
	batchCmd.Flags().StringVar(&modelsList, "models", "", "comma-separated list of models to test")
	batchCmd.Flags().IntVar(&concurrent, "concurrent", 5, "number of concurrent tests")

	listCmd.Flags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.acp-cli.yaml)")

	rootCmd.AddCommand(verifyCmd, batchCmd, listCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runVerify(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("model name required")
	}

	modelName := args[0]
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create verifier
	verifier := llmverifier.NewVerifier()

	// Get provider for model
	provider := "openai" // Default, could be detected from model name
	if strings.Contains(modelName, "claude") {
		provider = "anthropic"
	} else if strings.Contains(modelName, "deepseek") {
		provider = "deepseek"
	}

	// Create client
	client, err := createProviderClient(provider, cfg)
	if err != nil {
		return fmt.Errorf("failed to create provider client: %w", err)
	}

	// Test ACP support
	ctx := context.Background()
	acpSupported := verifier.TestACPs(client, modelName, ctx)

	// Calculate actual verification score
	verificationScore := calculateVerificationScore(acpSupported, client, modelName, ctx)

	// Measure actual latency with a test request
	latencyStart := time.Now()
	testReq := llmverifier.ChatCompletionRequest{
		Model: modelName,
		Messages: []llmverifier.Message{
			{Role: "user", Content: "Test."},
		},
		MaxTokens:   5,
		Temperature: 0.0,
	}
	_, latencyErr := client.ChatCompletion(ctx, testReq)
	actualLatency := time.Since(latencyStart)

	// Determine availability based on actual test
	responsive := latencyErr == nil

	// Create result with actual measured values
	result := llmverifier.VerificationResult{
		ModelInfo: llmverifier.ModelInfo{
			ID:      modelName,
			Object:  "model",
			Created: time.Now().Unix(),
			OwnedBy: provider,
		},
		FeatureDetection: llmverifier.FeatureDetection{
			ACPs: acpSupported,
		},
		Availability: llmverifier.AvailabilityResult{
			Exists:      true,
			Responsive:  responsive,
			Overloaded:  actualLatency > 10*time.Second,
			Latency:     actualLatency,
			LastChecked: time.Now(),
		},
		PerformanceScores: llmverifier.PerformanceScore{
			OverallScore: verificationScore / 100.0, // Convert to 0-1 scale
		},
		Timestamp: time.Now(),
	}

	// Output results
	return outputResults([]llmverifier.VerificationResult{result}, outputFormat)
}

func runBatch(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	models := []string{}
	if modelsList != "" {
		models = strings.Split(modelsList, ",")
	}

	if len(models) == 0 {
		return fmt.Errorf("no models specified. Use --models flag with comma-separated model names")
	}

	results := []BatchResult{}
	verifier := llmverifier.NewVerifier()
	ctx := context.Background()

	for _, model := range models {
		modelName := strings.TrimSpace(model)
		if verbose {
			fmt.Printf("Testing model: %s\n", modelName)
		}

		start := time.Now()

		// Detect provider from model name
		provider := detectProvider(modelName)

		// Create client for the provider
		client, err := createProviderClient(provider, cfg)
		if err != nil {
			// Record failure but continue with other models
			results = append(results, BatchResult{
				Model:     modelName,
				Provider:  provider,
				Supported: false,
				Score:     0.0,
				Duration:  time.Since(start),
			})
			if verbose {
				fmt.Printf("  Error creating client for %s: %v\n", provider, err)
			}
			continue
		}

		// Test ACP support with actual verification
		acpSupported := verifier.TestACPs(client, modelName, ctx)

		// Calculate actual score based on verification results
		score := calculateVerificationScore(acpSupported, client, modelName, ctx)

		result := BatchResult{
			Model:     modelName,
			Provider:  provider,
			Supported: acpSupported,
			Score:     score,
			Duration:  time.Since(start),
		}
		results = append(results, result)

		if verbose {
			fmt.Printf("  Completed: supported=%t, score=%.2f, duration=%s\n",
				acpSupported, score, result.Duration)
		}
	}

	return outputBatchResults(results, outputFormat)
}

// detectProvider determines the provider based on model name
func detectProvider(modelName string) string {
	modelLower := strings.ToLower(modelName)
	switch {
	case strings.Contains(modelLower, "claude"):
		return "anthropic"
	case strings.Contains(modelLower, "deepseek"):
		return "deepseek"
	case strings.Contains(modelLower, "gemini"):
		return "google"
	case strings.Contains(modelLower, "gpt") || strings.Contains(modelLower, "o1") || strings.Contains(modelLower, "davinci"):
		return "openai"
	case strings.Contains(modelLower, "llama") || strings.Contains(modelLower, "mistral"):
		return "openai" // Often served via OpenAI-compatible APIs
	default:
		return "openai"
	}
}

// calculateVerificationScore calculates actual score based on verification results
func calculateVerificationScore(acpSupported bool, client *llmverifier.LLMClient, modelName string, ctx context.Context) float64 {
	if !acpSupported {
		return 0.0
	}

	// Base score for ACP support
	score := 50.0

	// Test response quality
	testReq := llmverifier.ChatCompletionRequest{
		Model: modelName,
		Messages: []llmverifier.Message{
			{Role: "user", Content: "Reply with just the word 'verified' if you can read this."},
		},
		MaxTokens:   10,
		Temperature: 0.0,
	}

	start := time.Now()
	resp, err := client.ChatCompletion(ctx, testReq)
	latency := time.Since(start)

	if err != nil {
		return score // Return base score on error
	}

	// Add points for successful response
	if len(resp.Choices) > 0 && resp.Choices[0].Message.Content != "" {
		score += 25.0

		// Check if response contains expected content
		if strings.Contains(strings.ToLower(resp.Choices[0].Message.Content), "verif") {
			score += 15.0
		}
	}

	// Add points for fast response (under 2 seconds)
	if latency < 2*time.Second {
		score += 10.0
	} else if latency < 5*time.Second {
		score += 5.0
	}

	return score
}

// ProviderModels represents models discovered from a provider
type ProviderModels struct {
	Provider string   `json:"provider"`
	Models   []string `json:"models"`
	Error    string   `json:"error,omitempty"`
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Discover models from available providers
	providers := []string{"openai", "anthropic", "deepseek"}
	allProviderModels := []ProviderModels{}
	allModels := []string{}

	for _, provider := range providers {
		client, err := createProviderClient(provider, cfg)
		if err != nil {
			if verbose {
				fmt.Printf("Skipping %s: %v\n", provider, err)
			}
			allProviderModels = append(allProviderModels, ProviderModels{
				Provider: provider,
				Models:   []string{},
				Error:    err.Error(),
			})
			continue
		}

		// Try to list models from the provider
		models, discoverErr := discoverProviderModels(client, provider)
		if discoverErr != nil {
			if verbose {
				fmt.Printf("Failed to discover models from %s: %v\n", provider, discoverErr)
			}
			allProviderModels = append(allProviderModels, ProviderModels{
				Provider: provider,
				Models:   []string{},
				Error:    discoverErr.Error(),
			})
			continue
		}

		allProviderModels = append(allProviderModels, ProviderModels{
			Provider: provider,
			Models:   models,
		})
		allModels = append(allModels, models...)
	}

	// If no models discovered, provide fallback with clear indication
	if len(allModels) == 0 {
		fmt.Println("No models discovered from any provider.")
		fmt.Println("Ensure API keys are set: OPENAI_API_KEY, ANTHROPIC_API_KEY, DEEPSEEK_API_KEY")
		return nil
	}

	switch outputFormat {
	case "json":
		data, err := json.MarshalIndent(allProviderModels, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	case "yaml":
		fmt.Println("providers:")
		for _, pm := range allProviderModels {
			fmt.Printf("  - provider: %s\n", pm.Provider)
			if pm.Error != "" {
				fmt.Printf("    error: %s\n", pm.Error)
			} else {
				fmt.Println("    models:")
				for _, model := range pm.Models {
					fmt.Printf("      - %s\n", model)
				}
			}
		}
	default:
		fmt.Println("Available Models by Provider:")
		fmt.Println("=============================")
		for _, pm := range allProviderModels {
			fmt.Printf("\n%s:\n", pm.Provider)
			if pm.Error != "" {
				fmt.Printf("  Error: %s\n", pm.Error)
			} else if len(pm.Models) == 0 {
				fmt.Println("  No models found")
			} else {
				for _, model := range pm.Models {
					fmt.Printf("  - %s\n", model)
				}
			}
		}
	}

	return nil
}

// discoverProviderModels attempts to discover available models from a provider
func discoverProviderModels(client *llmverifier.LLMClient, provider string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	models, err := client.ListModels(ctx)
	if err != nil {
		// If ListModels fails, return known models for the provider
		return getKnownModelsForProvider(provider), nil
	}

	return models, nil
}

// getKnownModelsForProvider returns commonly known models when discovery fails
// These are used as fallback only when API discovery is not possible
func getKnownModelsForProvider(provider string) []string {
	switch provider {
	case "openai":
		return []string{
			"gpt-4o", "gpt-4o-mini", "gpt-4-turbo", "gpt-4",
			"gpt-3.5-turbo", "o1-preview", "o1-mini",
		}
	case "anthropic":
		return []string{
			"claude-3-5-sonnet-latest", "claude-3-5-haiku-latest",
			"claude-3-opus-latest", "claude-3-sonnet-20240229",
		}
	case "deepseek":
		return []string{
			"deepseek-chat", "deepseek-coder", "deepseek-reasoner",
		}
	default:
		return []string{}
	}
}

func loadConfig() (*config.Config, error) {
	// Create a default config
	cfg := &config.Config{
		Profile:     "default",
		Concurrency: concurrent,
		Timeout:     30 * time.Second,
		Global: config.GlobalConfig{
			MaxRetries: 3,
			Timeout:    30 * time.Second,
		},
		LLMs: []config.LLMConfig{
			{
				Name:     "openai",
				Endpoint: "https://api.openai.com/v1",
				APIKey:   os.Getenv("OPENAI_API_KEY"),
				Model:    "gpt-4",
			},
		},
	}

	return cfg, nil
}

func outputResults(results []llmverifier.VerificationResult, format string) error {
	switch format {
	case "json":
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	case "yaml":
		fmt.Printf("results:\n")
		for _, result := range results {
			fmt.Printf("  - model: %s\n", result.ModelInfo.ID)
			fmt.Printf("    available: %t\n", result.Availability.Exists && result.Availability.Responsive)
			fmt.Printf("    score: %.2f\n", result.PerformanceScores.OverallScore)
		}
	default:
		fmt.Println("Verification Results:")
		fmt.Println("====================")
		for _, result := range results {
			fmt.Printf("Model: %s\n", result.ModelInfo.ID)
			fmt.Printf("  Available: %t\n", result.Availability.Exists && result.Availability.Responsive)
			fmt.Printf("  Score: %.2f\n", result.PerformanceScores.OverallScore)
			fmt.Println()
		}
	}
	return nil
}

func outputBatchResults(results []BatchResult, format string) error {
	switch format {
	case "json":
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
	case "yaml":
		fmt.Printf("results:\n")
		for _, result := range results {
			fmt.Printf("  - model: %s\n", result.Model)
			fmt.Printf("    provider: %s\n", result.Provider)
			fmt.Printf("    supported: %t\n", result.Supported)
			fmt.Printf("    score: %.2f\n", result.Score)
			fmt.Printf("    duration: %s\n", result.Duration)
		}
	default:
		fmt.Println("Batch Verification Results:")
		fmt.Println("===========================")
		for _, result := range results {
			fmt.Printf("Model: %s\n", result.Model)
			fmt.Printf("  Provider: %s\n", result.Provider)
			fmt.Printf("  Supported: %t\n", result.Supported)
			fmt.Printf("  Score: %.2f\n", result.Score)
			fmt.Printf("  Duration: %s\n", result.Duration)
			fmt.Println()
		}
	}
	return nil
}
