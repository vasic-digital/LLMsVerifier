// Package main implements the Model Verification Real challenge.
// This challenge tests actual model responses with real prompts to verify
// model quality, accuracy, and response characteristics.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ProviderConfig holds configuration for an LLM provider.
type ProviderConfig struct {
	Name    string `json:"name"`
	APIKey  string `json:"-"` // Never serialize
	BaseURL string `json:"base_url,omitempty"`
	Enabled bool   `json:"enabled"`
}

// TestPrompt represents a test prompt with expected characteristics.
type TestPrompt struct {
	ID          string   `json:"id"`
	Category    string   `json:"category"`
	Prompt      string   `json:"prompt"`
	ExpectedMin int      `json:"expected_min_tokens"`
	ExpectedMax int      `json:"expected_max_tokens"`
	Keywords    []string `json:"keywords,omitempty"`
	Description string   `json:"description"`
}

// PromptTestResult holds the result of testing a prompt.
type PromptTestResult struct {
	PromptID      string  `json:"prompt_id"`
	Category      string  `json:"category"`
	Success       bool    `json:"success"`
	Response      string  `json:"response"`
	ResponseLen   int     `json:"response_length"`
	LatencyMs     int64   `json:"latency_ms"`
	TokensUsed    int     `json:"tokens_used,omitempty"`
	KeywordsFound int     `json:"keywords_found"`
	KeywordsTotal int     `json:"keywords_total"`
	QualityScore  float64 `json:"quality_score"`
	Error         string  `json:"error,omitempty"`
}

// ModelTestResult holds test results for a model.
type ModelTestResult struct {
	ModelID        string             `json:"model_id"`
	Provider       string             `json:"provider"`
	Success        bool               `json:"success"`
	PromptResults  []PromptTestResult `json:"prompt_results"`
	TotalPrompts   int                `json:"total_prompts"`
	PassedPrompts  int                `json:"passed_prompts"`
	FailedPrompts  int                `json:"failed_prompts"`
	AverageLatency float64            `json:"average_latency_ms"`
	AverageQuality float64            `json:"average_quality"`
	OverallScore   float64            `json:"overall_score"`
	ResponseTimeMs int64              `json:"response_time_ms"`
	Error          string             `json:"error,omitempty"`
	Timestamp      time.Time          `json:"timestamp"`
}

// ProviderTestResult holds all model test results for a provider.
type ProviderTestResult struct {
	Provider       string            `json:"provider"`
	Success        bool              `json:"success"`
	ModelsTested   int               `json:"models_tested"`
	Models         []ModelTestResult `json:"models"`
	AverageScore   float64           `json:"average_score"`
	ResponseTimeMs int64             `json:"response_time_ms"`
	Error          string            `json:"error,omitempty"`
	Timestamp      time.Time         `json:"timestamp"`
}

// ChallengeResult holds the complete challenge output.
type ChallengeResult struct {
	ChallengeID   string               `json:"challenge_id"`
	ChallengeName string               `json:"challenge_name"`
	Timestamp     time.Time            `json:"timestamp"`
	Duration      time.Duration        `json:"duration"`
	Status        string               `json:"status"`
	Results       []ProviderTestResult `json:"results"`
	AllModels     []ModelTestResult    `json:"all_models"`
	Summary       ChallengeSummary     `json:"summary"`
}

// ChallengeSummary provides aggregated statistics.
type ChallengeSummary struct {
	TotalProviders      int     `json:"total_providers"`
	SuccessfulProviders int     `json:"successful_providers"`
	FailedProviders     int     `json:"failed_providers"`
	TotalModels         int     `json:"total_models"`
	PassedModels        int     `json:"passed_models"`
	TotalPrompts        int     `json:"total_prompts"`
	PassedPrompts       int     `json:"passed_prompts"`
	AverageLatency      float64 `json:"average_latency_ms"`
	AverageQuality      float64 `json:"average_quality"`
	HighestScore        float64 `json:"highest_score"`
	TopModel            string  `json:"top_model"`
}

// API structures
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
}

type ChatCompletionResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int         `json:"index"`
		Message      ChatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// Anthropic structures
type AnthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	Messages  []AnthropicMessage `json:"messages"`
}

type AnthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type AnthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Gemini structures
type GeminiRequest struct {
	Contents         []GeminiContent        `json:"contents"`
	GenerationConfig GeminiGenerationConfig `json:"generationConfig,omitempty"`
}

type GeminiContent struct {
	Parts []GeminiPart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type GeminiPart struct {
	Text string `json:"text,omitempty"`
}

type GeminiGenerationConfig struct {
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
	Temperature     float64 `json:"temperature,omitempty"`
}

type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// Test prompts for real verification
var testPrompts = []TestPrompt{
	{
		ID:          "code_gen_simple",
		Category:    "code_generation",
		Prompt:      "Write a Python function that calculates the factorial of a number using recursion. Include a docstring.",
		ExpectedMin: 50,
		ExpectedMax: 500,
		Keywords:    []string{"def", "factorial", "return", "if", "else"},
		Description: "Simple code generation test",
	},
	{
		ID:          "reasoning_math",
		Category:    "reasoning",
		Prompt:      "If a train travels at 60 mph for 2.5 hours, then at 80 mph for 1.5 hours, what is the total distance traveled? Show your work step by step.",
		ExpectedMin: 50,
		ExpectedMax: 400,
		Keywords:    []string{"60", "80", "distance", "miles", "270"},
		Description: "Mathematical reasoning test",
	},
	{
		ID:          "knowledge_tech",
		Category:    "knowledge",
		Prompt:      "Explain the difference between a compiler and an interpreter in programming. Give one example of each.",
		ExpectedMin: 100,
		ExpectedMax: 600,
		Keywords:    []string{"compiler", "interpreter", "code", "execution"},
		Description: "Technical knowledge test",
	},
	{
		ID:          "creative_writing",
		Category:    "creative",
		Prompt:      "Write a haiku about artificial intelligence.",
		ExpectedMin: 10,
		ExpectedMax: 100,
		Keywords:    []string{},
		Description: "Creative writing test - haiku format",
	},
	{
		ID:          "instruction_follow",
		Category:    "instruction",
		Prompt:      "List exactly 5 programming languages, one per line, without any additional explanation or numbering.",
		ExpectedMin: 20,
		ExpectedMax: 100,
		Keywords:    []string{},
		Description: "Instruction following test",
	},
}

// Known models per provider
var knownModels = map[string][]string{
	"openai":     {"gpt-4o-mini"},
	"anthropic":  {"claude-3-haiku-20240307"},
	"openrouter": {"openai/gpt-4o-mini"},
	"deepseek":   {"deepseek-chat"},
	"gemini":     {"gemini-1.5-flash"},
	"groq":       {"llama-3.1-8b-instant"},
}

func sendPrompt(ctx context.Context, config ProviderConfig, modelID, prompt string, maxTokens int) (string, int, error) {
	switch config.Name {
	case "anthropic":
		return sendAnthropicPrompt(ctx, config, modelID, prompt, maxTokens)
	case "gemini":
		return sendGeminiPrompt(ctx, config, modelID, prompt, maxTokens)
	case "ollama":
		return sendOllamaPrompt(ctx, config, modelID, prompt, maxTokens)
	default:
		return sendOpenAICompatiblePrompt(ctx, config, modelID, prompt, maxTokens)
	}
}

func sendOpenAICompatiblePrompt(ctx context.Context, config ProviderConfig, modelID, prompt string, maxTokens int) (string, int, error) {
	baseURL := getBaseURL(config)

	reqBody := ChatCompletionRequest{
		Model:       modelID,
		Messages:    []ChatMessage{{Role: "user", Content: prompt}},
		MaxTokens:   maxTokens,
		Temperature: 0.7,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("Authorization", "Bearer "+config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", 0, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", 0, fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, chatResp.Usage.CompletionTokens, nil
}

func sendAnthropicPrompt(ctx context.Context, config ProviderConfig, modelID, prompt string, maxTokens int) (string, int, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}

	reqBody := AnthropicRequest{
		Model:     modelID,
		MaxTokens: maxTokens,
		Messages:  []AnthropicMessage{{Role: "user", Content: prompt}},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/messages", bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("x-api-key", config.APIKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var anthropicResp AnthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
		return "", 0, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(anthropicResp.Content) == 0 {
		return "", 0, fmt.Errorf("no content in response")
	}

	return anthropicResp.Content[0].Text, anthropicResp.Usage.OutputTokens, nil
}

func sendGeminiPrompt(ctx context.Context, config ProviderConfig, modelID, prompt string, maxTokens int) (string, int, error) {
	baseURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", modelID, config.APIKey)

	reqBody := GeminiRequest{
		Contents: []GeminiContent{
			{Parts: []GeminiPart{{Text: prompt}}},
		},
		GenerationConfig: GeminiGenerationConfig{
			MaxOutputTokens: maxTokens,
			Temperature:     0.7,
		},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL, bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var geminiResp GeminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", 0, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", 0, fmt.Errorf("no content in response")
	}

	text := geminiResp.Candidates[0].Content.Parts[0].Text
	// Estimate tokens (rough approximation)
	tokens := len(strings.Fields(text))

	return text, tokens, nil
}

func sendOllamaPrompt(ctx context.Context, config ProviderConfig, modelID, prompt string, maxTokens int) (string, int, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	type OllamaRequest struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
		Stream bool   `json:"stream"`
	}

	type OllamaResponse struct {
		Response string `json:"response"`
	}

	reqBody := OllamaRequest{
		Model:  modelID,
		Prompt: prompt,
		Stream: false,
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", 0, err
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", 0, fmt.Errorf("failed to decode response: %w", err)
	}

	tokens := len(strings.Fields(ollamaResp.Response))

	return ollamaResp.Response, tokens, nil
}

func getBaseURL(config ProviderConfig) string {
	if config.BaseURL != "" {
		return config.BaseURL
	}

	switch config.Name {
	case "openai":
		return "https://api.openai.com/v1"
	case "openrouter":
		return "https://openrouter.ai/api/v1"
	case "deepseek":
		return "https://api.deepseek.com"
	case "groq":
		return "https://api.groq.com/openai/v1"
	default:
		return "https://api.openai.com/v1"
	}
}

func testPromptOnModel(ctx context.Context, config ProviderConfig, modelID string, testPrompt TestPrompt) PromptTestResult {
	result := PromptTestResult{
		PromptID:      testPrompt.ID,
		Category:      testPrompt.Category,
		KeywordsTotal: len(testPrompt.Keywords),
	}

	start := time.Now()
	response, tokens, err := sendPrompt(ctx, config, modelID, testPrompt.Prompt, testPrompt.ExpectedMax)
	result.LatencyMs = time.Since(start).Milliseconds()

	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.Success = true
	result.Response = response
	result.ResponseLen = len(response)
	result.TokensUsed = tokens

	// Check for keywords
	responseLower := strings.ToLower(response)
	for _, keyword := range testPrompt.Keywords {
		if strings.Contains(responseLower, strings.ToLower(keyword)) {
			result.KeywordsFound++
		}
	}

	// Calculate quality score (0-100)
	score := 0.0

	// Length appropriateness (30%)
	if result.ResponseLen >= testPrompt.ExpectedMin && result.ResponseLen <= testPrompt.ExpectedMax*2 {
		score += 30.0
	} else if result.ResponseLen >= testPrompt.ExpectedMin/2 {
		score += 15.0
	}

	// Keywords found (40%)
	if result.KeywordsTotal > 0 {
		keywordScore := float64(result.KeywordsFound) / float64(result.KeywordsTotal) * 40.0
		score += keywordScore
	} else {
		// No keywords to check, give full points if response is reasonable
		if result.ResponseLen >= testPrompt.ExpectedMin {
			score += 40.0
		}
	}

	// Response existence and coherence (30%)
	if result.ResponseLen > 0 {
		score += 15.0
		// Check for common signs of a good response
		if strings.Contains(response, ".") || strings.Contains(response, "\n") {
			score += 15.0
		}
	}

	result.QualityScore = score

	return result
}

func testModel(ctx context.Context, config ProviderConfig, modelID string) *ModelTestResult {
	result := &ModelTestResult{
		ModelID:   modelID,
		Provider:  config.Name,
		Timestamp: time.Now(),
	}

	start := time.Now()
	var promptResults []PromptTestResult
	var totalLatency int64
	var totalQuality float64
	passedCount := 0

	for _, testPrompt := range testPrompts {
		promptResult := testPromptOnModel(ctx, config, modelID, testPrompt)
		promptResults = append(promptResults, promptResult)

		if promptResult.Success {
			passedCount++
			totalLatency += promptResult.LatencyMs
			totalQuality += promptResult.QualityScore
		}
	}

	result.PromptResults = promptResults
	result.TotalPrompts = len(testPrompts)
	result.PassedPrompts = passedCount
	result.FailedPrompts = len(testPrompts) - passedCount
	result.ResponseTimeMs = time.Since(start).Milliseconds()

	if passedCount > 0 {
		result.AverageLatency = float64(totalLatency) / float64(passedCount)
		result.AverageQuality = totalQuality / float64(passedCount)
		result.Success = true
	}

	// Calculate overall score (quality weighted by success rate)
	if result.TotalPrompts > 0 {
		successRate := float64(passedCount) / float64(result.TotalPrompts)
		result.OverallScore = result.AverageQuality * successRate
	}

	return result
}

func testProvider(ctx context.Context, config ProviderConfig) *ProviderTestResult {
	result := &ProviderTestResult{
		Provider:  config.Name,
		Timestamp: time.Now(),
	}

	if config.APIKey == "" && config.Name != "ollama" {
		result.Error = "API key not configured"
		return result
	}

	start := time.Now()

	models, exists := knownModels[config.Name]
	if !exists || len(models) == 0 {
		result.Error = "No known models for provider"
		return result
	}

	var modelResults []ModelTestResult
	var totalScore float64

	// Test first model only to save time and API costs
	modelID := models[0]
	modelResult := testModel(ctx, config, modelID)
	modelResults = append(modelResults, *modelResult)
	totalScore += modelResult.OverallScore

	result.ResponseTimeMs = time.Since(start).Milliseconds()
	result.Models = modelResults
	result.ModelsTested = len(modelResults)

	if result.ModelsTested > 0 {
		result.AverageScore = totalScore / float64(result.ModelsTested)
		result.Success = true
	}

	return result
}

func main() {
	resultsDir := flag.String("results-dir", "", "Directory to store results")
	verbose := flag.Bool("verbose", false, "Enable verbose output")
	flag.Parse()

	if *resultsDir == "" {
		log.Fatal("--results-dir is required")
	}

	ctx := context.Background()
	start := time.Now()

	// Create results directory
	resultsPath := filepath.Join(*resultsDir, "results")
	logsPath := filepath.Join(*resultsDir, "logs")
	if err := os.MkdirAll(resultsPath, 0755); err != nil {
		log.Fatalf("Failed to create results directory: %v", err)
	}
	if err := os.MkdirAll(logsPath, 0755); err != nil {
		log.Fatalf("Failed to create logs directory: %v", err)
	}

	// Configure providers from environment
	providers := []ProviderConfig{
		{Name: "anthropic", APIKey: os.Getenv("ANTHROPIC_API_KEY"), Enabled: true},
		{Name: "openai", APIKey: os.Getenv("OPENAI_API_KEY"), Enabled: true},
		{Name: "openrouter", APIKey: os.Getenv("OPENROUTER_API_KEY"), Enabled: true},
		{Name: "deepseek", APIKey: os.Getenv("DEEPSEEK_API_KEY"), Enabled: true},
		{Name: "gemini", APIKey: os.Getenv("GEMINI_API_KEY"), Enabled: true},
		{Name: "groq", APIKey: os.Getenv("GROQ_API_KEY"), Enabled: true},
		{Name: "ollama", BaseURL: os.Getenv("OLLAMA_BASE_URL"), Enabled: true},
	}

	// Filter to configured providers
	var configuredProviders []ProviderConfig
	for _, p := range providers {
		if p.APIKey != "" || p.Name == "ollama" {
			configuredProviders = append(configuredProviders, p)
		}
	}

	if *verbose {
		log.Printf("Found %d configured providers", len(configuredProviders))
		log.Printf("Running %d test prompts per model", len(testPrompts))
	}

	// Test each provider concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	var testResults []ProviderTestResult
	var allModels []ModelTestResult

	for _, provider := range configuredProviders {
		wg.Add(1)
		go func(p ProviderConfig) {
			defer wg.Done()

			if *verbose {
				log.Printf("Testing provider: %s", p.Name)
			}

			result := testProvider(ctx, p)

			mu.Lock()
			testResults = append(testResults, *result)
			if result.Success {
				allModels = append(allModels, result.Models...)
			}
			mu.Unlock()
		}(provider)
	}

	wg.Wait()

	// Sort models by overall score (descending)
	sort.Slice(allModels, func(i, j int) bool {
		return allModels[i].OverallScore > allModels[j].OverallScore
	})

	// Calculate summary
	successfulCount := 0
	failedCount := 0
	var totalLatency float64
	var totalQuality float64
	var highestScore float64
	var topModel string
	totalPrompts := 0
	passedPrompts := 0

	for _, r := range testResults {
		if r.Success {
			successfulCount++
		} else {
			failedCount++
		}
	}

	for _, m := range allModels {
		totalLatency += m.AverageLatency
		totalQuality += m.AverageQuality
		totalPrompts += m.TotalPrompts
		passedPrompts += m.PassedPrompts
		if m.OverallScore > highestScore {
			highestScore = m.OverallScore
			topModel = m.ModelID
		}
	}

	avgLatency := float64(0)
	avgQuality := float64(0)
	if len(allModels) > 0 {
		avgLatency = totalLatency / float64(len(allModels))
		avgQuality = totalQuality / float64(len(allModels))
	}

	passedModels := 0
	for _, m := range allModels {
		if m.Success && m.OverallScore >= 50 {
			passedModels++
		}
	}

	// Build final result
	challengeResult := ChallengeResult{
		ChallengeID:   "run_model_verification_real",
		ChallengeName: "Model Verification Real",
		Timestamp:     time.Now(),
		Duration:      time.Since(start),
		Status:        "passed",
		Results:       testResults,
		AllModels:     allModels,
		Summary: ChallengeSummary{
			TotalProviders:      len(configuredProviders),
			SuccessfulProviders: successfulCount,
			FailedProviders:     failedCount,
			TotalModels:         len(allModels),
			PassedModels:        passedModels,
			TotalPrompts:        totalPrompts,
			PassedPrompts:       passedPrompts,
			AverageLatency:      avgLatency,
			AverageQuality:      avgQuality,
			HighestScore:        highestScore,
			TopModel:            topModel,
		},
	}

	if successfulCount == 0 {
		challengeResult.Status = "failed"
	}

	// Write results
	resultFile := filepath.Join(resultsPath, "real_verification_results.json")
	resultData, _ := json.MarshalIndent(challengeResult, "", "  ")
	if err := os.WriteFile(resultFile, resultData, 0644); err != nil {
		log.Printf("Warning: Failed to write results: %v", err)
	}

	modelsFile := filepath.Join(resultsPath, "tested_models.json")
	modelsData, _ := json.MarshalIndent(allModels, "", "  ")
	if err := os.WriteFile(modelsFile, modelsData, 0644); err != nil {
		log.Printf("Warning: Failed to write models: %v", err)
	}

	// Write report
	reportFile := filepath.Join(resultsPath, "real_verification_report.md")
	report := generateReport(challengeResult)
	if err := os.WriteFile(reportFile, []byte(report), 0644); err != nil {
		log.Printf("Warning: Failed to write report: %v", err)
	}

	// Print summary
	fmt.Printf("\n=== Model Verification Real Complete ===\n")
	fmt.Printf("Status: %s\n", strings.ToUpper(challengeResult.Status))
	fmt.Printf("Providers: %d successful, %d failed\n", successfulCount, failedCount)
	fmt.Printf("Models: %d passed out of %d (score >= 50%%)\n", passedModels, len(allModels))
	fmt.Printf("Prompts: %d passed out of %d\n", passedPrompts, totalPrompts)
	if topModel != "" {
		fmt.Printf("Top Model: %s (score: %.1f)\n", topModel, highestScore)
	}
	fmt.Printf("Average Quality: %.1f\n", avgQuality)
	fmt.Printf("Average Latency: %.0fms\n", avgLatency)
	fmt.Printf("Duration: %v\n", challengeResult.Duration)
	fmt.Printf("Results: %s\n", resultsPath)

	if challengeResult.Status == "failed" {
		os.Exit(1)
	}
}

func generateReport(result ChallengeResult) string {
	var sb strings.Builder

	sb.WriteString("# Model Verification Real Report\n\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", result.Timestamp.Format(time.RFC3339)))

	sb.WriteString("## Summary\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Status | %s |\n", strings.ToUpper(result.Status)))
	sb.WriteString(fmt.Sprintf("| Total Providers | %d |\n", result.Summary.TotalProviders))
	sb.WriteString(fmt.Sprintf("| Successful Providers | %d |\n", result.Summary.SuccessfulProviders))
	sb.WriteString(fmt.Sprintf("| Failed Providers | %d |\n", result.Summary.FailedProviders))
	sb.WriteString(fmt.Sprintf("| Total Models | %d |\n", result.Summary.TotalModels))
	sb.WriteString(fmt.Sprintf("| Passed Models | %d |\n", result.Summary.PassedModels))
	sb.WriteString(fmt.Sprintf("| Total Prompts Tested | %d |\n", result.Summary.TotalPrompts))
	sb.WriteString(fmt.Sprintf("| Passed Prompts | %d |\n", result.Summary.PassedPrompts))
	sb.WriteString(fmt.Sprintf("| Average Quality | %.1f |\n", result.Summary.AverageQuality))
	sb.WriteString(fmt.Sprintf("| Average Latency | %.0fms |\n", result.Summary.AverageLatency))
	sb.WriteString(fmt.Sprintf("| Highest Score | %.1f |\n", result.Summary.HighestScore))
	sb.WriteString(fmt.Sprintf("| Top Model | %s |\n", result.Summary.TopModel))
	sb.WriteString(fmt.Sprintf("| Duration | %v |\n", result.Duration))

	sb.WriteString("\n## Test Prompts Used\n\n")
	sb.WriteString("| ID | Category | Description |\n")
	sb.WriteString("|----|----------|-------------|\n")
	for _, p := range testPrompts {
		sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", p.ID, p.Category, p.Description))
	}

	sb.WriteString("\n## Provider Results\n\n")
	sb.WriteString("| Provider | Status | Models Tested | Average Score | Response Time |\n")
	sb.WriteString("|----------|--------|---------------|---------------|---------------|\n")
	for _, r := range result.Results {
		status := "Failed"
		if r.Success {
			status = "Success"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %d | %.1f | %dms |\n",
			r.Provider, status, r.ModelsTested, r.AverageScore, r.ResponseTimeMs))
	}

	sb.WriteString("\n## Model Results\n\n")
	for _, m := range result.AllModels {
		status := "FAILED"
		if m.Success && m.OverallScore >= 50 {
			status = "PASSED"
		}
		sb.WriteString(fmt.Sprintf("### %s (%s) - %s\n\n", m.ModelID, m.Provider, status))
		sb.WriteString(fmt.Sprintf("**Overall Score**: %.1f | **Prompts**: %d/%d passed | **Avg Latency**: %.0fms | **Avg Quality**: %.1f\n\n",
			m.OverallScore, m.PassedPrompts, m.TotalPrompts, m.AverageLatency, m.AverageQuality))

		sb.WriteString("| Prompt | Category | Status | Quality | Latency | Keywords |\n")
		sb.WriteString("|--------|----------|--------|---------|---------|----------|\n")
		for _, pr := range m.PromptResults {
			status := "Failed"
			if pr.Success {
				status = "Passed"
			}
			keywords := fmt.Sprintf("%d/%d", pr.KeywordsFound, pr.KeywordsTotal)
			if pr.KeywordsTotal == 0 {
				keywords = "N/A"
			}
			sb.WriteString(fmt.Sprintf("| %s | %s | %s | %.1f | %dms | %s |\n",
				pr.PromptID, pr.Category, status, pr.QualityScore, pr.LatencyMs, keywords))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("---\n\n")
	sb.WriteString("*Generated by LLMsVerifier Model Verification Real Challenge*\n")

	return sb.String()
}
