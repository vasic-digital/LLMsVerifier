// Package main implements the Provider Models Discovery challenge.
// This challenge discovers all available models from configured LLM providers.
package main

import (
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

// DiscoveredModel represents a model discovered from a provider.
type DiscoveredModel struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Provider      string                 `json:"provider"`
	Created       int64                  `json:"created,omitempty"`
	OwnedBy       string                 `json:"owned_by,omitempty"`
	Capabilities  []string               `json:"capabilities,omitempty"`
	ContextWindow int                    `json:"context_window,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// DiscoveryResult holds the result of model discovery for a provider.
type DiscoveryResult struct {
	Provider       string            `json:"provider"`
	Success        bool              `json:"success"`
	ModelsFound    int               `json:"models_found"`
	Models         []DiscoveredModel `json:"models,omitempty"`
	ResponseTimeMs int64             `json:"response_time_ms"`
	Error          string            `json:"error,omitempty"`
	Timestamp      time.Time         `json:"timestamp"`
}

// ChallengeResult holds the complete challenge output.
type ChallengeResult struct {
	ChallengeID   string            `json:"challenge_id"`
	ChallengeName string            `json:"challenge_name"`
	Timestamp     time.Time         `json:"timestamp"`
	Duration      time.Duration     `json:"duration"`
	Status        string            `json:"status"`
	Results       []DiscoveryResult `json:"results"`
	AllModels     []DiscoveredModel `json:"all_models"`
	Summary       ChallengeSummary  `json:"summary"`
}

// ChallengeSummary provides aggregated statistics.
type ChallengeSummary struct {
	TotalProviders     int     `json:"total_providers"`
	SuccessfulProviders int    `json:"successful_providers"`
	FailedProviders    int     `json:"failed_providers"`
	TotalModels        int     `json:"total_models"`
	UniqueModels       int     `json:"unique_models"`
	AverageResponseMs  float64 `json:"average_response_ms"`
}

// OpenAI-style models response
type ModelsResponse struct {
	Object string `json:"object"`
	Data   []struct {
		ID       string `json:"id"`
		Object   string `json:"object"`
		Created  int64  `json:"created"`
		OwnedBy  string `json:"owned_by"`
	} `json:"data"`
}

// Gemini models response
type GeminiModelsResponse struct {
	Models []struct {
		Name                       string   `json:"name"`
		DisplayName                string   `json:"displayName"`
		Description                string   `json:"description"`
		SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
		InputTokenLimit            int      `json:"inputTokenLimit"`
		OutputTokenLimit           int      `json:"outputTokenLimit"`
	} `json:"models"`
}

// Ollama tags response
type OllamaTagsResponse struct {
	Models []struct {
		Name       string `json:"name"`
		ModifiedAt string `json:"modified_at"`
		Size       int64  `json:"size"`
	} `json:"models"`
}

func discoverModels(ctx context.Context, config ProviderConfig) *DiscoveryResult {
	result := &DiscoveryResult{
		Provider:  config.Name,
		Timestamp: time.Now(),
	}

	if config.APIKey == "" && config.Name != "ollama" {
		result.Error = "API key not configured"
		return result
	}

	start := time.Now()

	var models []DiscoveredModel
	var err error

	switch config.Name {
	case "openai":
		models, err = discoverOpenAIModels(ctx, config)
	case "openrouter":
		models, err = discoverOpenRouterModels(ctx, config)
	case "deepseek":
		models, err = discoverDeepSeekModels(ctx, config)
	case "gemini":
		models, err = discoverGeminiModels(ctx, config)
	case "ollama":
		models, err = discoverOllamaModels(ctx, config)
	case "groq":
		models, err = discoverGroqModels(ctx, config)
	case "anthropic":
		models, err = getAnthropicModels(config)
	default:
		models, err = discoverGenericModels(ctx, config)
	}

	result.ResponseTimeMs = time.Since(start).Milliseconds()

	if err != nil {
		result.Error = err.Error()
		return result
	}

	result.Success = true
	result.Models = models
	result.ModelsFound = len(models)

	return result
}

func discoverOpenAIModels(ctx context.Context, config ProviderConfig) ([]DiscoveredModel, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var modelsResp ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var models []DiscoveredModel
	for _, m := range modelsResp.Data {
		model := DiscoveredModel{
			ID:       m.ID,
			Name:     m.ID,
			Provider: "openai",
			Created:  m.Created,
			OwnedBy:  m.OwnedBy,
		}
		// Infer capabilities from model name
		model.Capabilities = inferCapabilities(m.ID)
		models = append(models, model)
	}

	return models, nil
}

func discoverOpenRouterModels(ctx context.Context, config ProviderConfig) ([]DiscoveredModel, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var modelsResp ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, err
	}

	var models []DiscoveredModel
	for _, m := range modelsResp.Data {
		model := DiscoveredModel{
			ID:           m.ID,
			Name:         m.ID,
			Provider:     "openrouter",
			Created:      m.Created,
			OwnedBy:      m.OwnedBy,
			Capabilities: inferCapabilities(m.ID),
		}
		models = append(models, model)
	}

	return models, nil
}

func discoverDeepSeekModels(ctx context.Context, config ProviderConfig) ([]DiscoveredModel, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var modelsResp ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, err
	}

	var models []DiscoveredModel
	for _, m := range modelsResp.Data {
		model := DiscoveredModel{
			ID:           m.ID,
			Name:         m.ID,
			Provider:     "deepseek",
			Created:      m.Created,
			OwnedBy:      m.OwnedBy,
			Capabilities: inferCapabilities(m.ID),
		}
		models = append(models, model)
	}

	return models, nil
}

func discoverGeminiModels(ctx context.Context, config ProviderConfig) ([]DiscoveredModel, error) {
	baseURL := "https://generativelanguage.googleapis.com/v1beta"

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/models?key="+config.APIKey, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var geminiResp GeminiModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, err
	}

	var models []DiscoveredModel
	for _, m := range geminiResp.Models {
		// Extract model ID from name (format: "models/gemini-pro")
		id := m.Name
		if strings.HasPrefix(id, "models/") {
			id = strings.TrimPrefix(id, "models/")
		}

		model := DiscoveredModel{
			ID:            id,
			Name:          m.DisplayName,
			Provider:      "gemini",
			Capabilities:  m.SupportedGenerationMethods,
			ContextWindow: m.InputTokenLimit,
			Metadata: map[string]interface{}{
				"description":        m.Description,
				"output_token_limit": m.OutputTokenLimit,
			},
		}
		models = append(models, model)
	}

	return models, nil
}

func discoverOllamaModels(ctx context.Context, config ProviderConfig) ([]DiscoveredModel, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	var ollamaResp OllamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, err
	}

	var models []DiscoveredModel
	for _, m := range ollamaResp.Models {
		model := DiscoveredModel{
			ID:           m.Name,
			Name:         m.Name,
			Provider:     "ollama",
			Capabilities: inferCapabilities(m.Name),
			Metadata: map[string]interface{}{
				"size": m.Size,
			},
		}
		models = append(models, model)
	}

	return models, nil
}

func discoverGroqModels(ctx context.Context, config ProviderConfig) ([]DiscoveredModel, error) {
	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.groq.com/openai/v1"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var modelsResp ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, err
	}

	var models []DiscoveredModel
	for _, m := range modelsResp.Data {
		model := DiscoveredModel{
			ID:           m.ID,
			Name:         m.ID,
			Provider:     "groq",
			Created:      m.Created,
			OwnedBy:      m.OwnedBy,
			Capabilities: inferCapabilities(m.ID),
		}
		models = append(models, model)
	}

	return models, nil
}

func getAnthropicModels(config ProviderConfig) ([]DiscoveredModel, error) {
	// Anthropic doesn't have a models listing API, return known models
	models := []DiscoveredModel{
		{ID: "claude-3-opus-20240229", Name: "Claude 3 Opus", Provider: "anthropic", Capabilities: []string{"chat", "vision", "code"}},
		{ID: "claude-3-sonnet-20240229", Name: "Claude 3 Sonnet", Provider: "anthropic", Capabilities: []string{"chat", "vision", "code"}},
		{ID: "claude-3-haiku-20240307", Name: "Claude 3 Haiku", Provider: "anthropic", Capabilities: []string{"chat", "vision", "code"}},
		{ID: "claude-3-5-sonnet-20241022", Name: "Claude 3.5 Sonnet", Provider: "anthropic", Capabilities: []string{"chat", "vision", "code"}},
	}
	return models, nil
}

func discoverGenericModels(ctx context.Context, config ProviderConfig) ([]DiscoveredModel, error) {
	if config.BaseURL == "" {
		return nil, fmt.Errorf("no base URL configured")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", config.BaseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var modelsResp ModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, err
	}

	var models []DiscoveredModel
	for _, m := range modelsResp.Data {
		model := DiscoveredModel{
			ID:           m.ID,
			Name:         m.ID,
			Provider:     config.Name,
			Created:      m.Created,
			OwnedBy:      m.OwnedBy,
			Capabilities: inferCapabilities(m.ID),
		}
		models = append(models, model)
	}

	return models, nil
}

func inferCapabilities(modelID string) []string {
	id := strings.ToLower(modelID)
	var caps []string

	// Chat capability
	if strings.Contains(id, "chat") || strings.Contains(id, "instruct") ||
		strings.Contains(id, "gpt") || strings.Contains(id, "claude") ||
		strings.Contains(id, "gemini") || strings.Contains(id, "llama") {
		caps = append(caps, "chat")
	}

	// Code capability
	if strings.Contains(id, "code") || strings.Contains(id, "coder") ||
		strings.Contains(id, "codestral") || strings.Contains(id, "deepseek-coder") {
		caps = append(caps, "code")
	}

	// Vision capability
	if strings.Contains(id, "vision") || strings.Contains(id, "4o") ||
		strings.Contains(id, "gemini-pro-vision") {
		caps = append(caps, "vision")
	}

	// Embedding capability
	if strings.Contains(id, "embed") || strings.Contains(id, "ada") {
		caps = append(caps, "embeddings")
	}

	return caps
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
	}

	// Discover models from each provider concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex
	var discoveryResults []DiscoveryResult
	var allModels []DiscoveredModel

	for _, provider := range configuredProviders {
		wg.Add(1)
		go func(p ProviderConfig) {
			defer wg.Done()

			if *verbose {
				log.Printf("Discovering models from: %s", p.Name)
			}

			result := discoverModels(ctx, p)

			mu.Lock()
			discoveryResults = append(discoveryResults, *result)
			if result.Success {
				allModels = append(allModels, result.Models...)
			}
			mu.Unlock()
		}(provider)
	}

	wg.Wait()

	// Sort models by provider and name
	sort.Slice(allModels, func(i, j int) bool {
		if allModels[i].Provider != allModels[j].Provider {
			return allModels[i].Provider < allModels[j].Provider
		}
		return allModels[i].ID < allModels[j].ID
	})

	// Calculate summary
	var totalResponseTime int64
	successfulCount := 0
	failedCount := 0

	for _, r := range discoveryResults {
		if r.Success {
			successfulCount++
			totalResponseTime += r.ResponseTimeMs
		} else {
			failedCount++
		}
	}

	avgResponseTime := float64(0)
	if successfulCount > 0 {
		avgResponseTime = float64(totalResponseTime) / float64(successfulCount)
	}

	// Count unique models
	modelSet := make(map[string]bool)
	for _, m := range allModels {
		modelSet[m.ID] = true
	}

	// Build final result
	challengeResult := ChallengeResult{
		ChallengeID:   "provider_models_discovery",
		ChallengeName: "Provider Models Discovery",
		Timestamp:     time.Now(),
		Duration:      time.Since(start),
		Status:        "passed",
		Results:       discoveryResults,
		AllModels:     allModels,
		Summary: ChallengeSummary{
			TotalProviders:      len(configuredProviders),
			SuccessfulProviders: successfulCount,
			FailedProviders:     failedCount,
			TotalModels:         len(allModels),
			UniqueModels:        len(modelSet),
			AverageResponseMs:   avgResponseTime,
		},
	}

	if successfulCount == 0 {
		challengeResult.Status = "failed"
	}

	// Write results
	resultFile := filepath.Join(resultsPath, "discovery_results.json")
	resultData, _ := json.MarshalIndent(challengeResult, "", "  ")
	if err := os.WriteFile(resultFile, resultData, 0644); err != nil {
		log.Printf("Warning: Failed to write results: %v", err)
	}

	modelsFile := filepath.Join(resultsPath, "discovered_models.json")
	modelsData, _ := json.MarshalIndent(allModels, "", "  ")
	if err := os.WriteFile(modelsFile, modelsData, 0644); err != nil {
		log.Printf("Warning: Failed to write models: %v", err)
	}

	// Write report
	reportFile := filepath.Join(resultsPath, "discovery_report.md")
	report := generateReport(challengeResult)
	if err := os.WriteFile(reportFile, []byte(report), 0644); err != nil {
		log.Printf("Warning: Failed to write report: %v", err)
	}

	// Print summary
	fmt.Printf("\n=== Provider Models Discovery Complete ===\n")
	fmt.Printf("Status: %s\n", strings.ToUpper(challengeResult.Status))
	fmt.Printf("Providers: %d successful, %d failed\n", successfulCount, failedCount)
	fmt.Printf("Models: %d total, %d unique\n", len(allModels), len(modelSet))
	fmt.Printf("Duration: %v\n", challengeResult.Duration)
	fmt.Printf("Results: %s\n", resultsPath)

	if challengeResult.Status == "failed" {
		os.Exit(1)
	}
}

func generateReport(result ChallengeResult) string {
	var sb strings.Builder

	sb.WriteString("# Provider Models Discovery Report\n\n")
	sb.WriteString(fmt.Sprintf("Generated: %s\n\n", result.Timestamp.Format(time.RFC3339)))

	sb.WriteString("## Summary\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Status | %s |\n", strings.ToUpper(result.Status)))
	sb.WriteString(fmt.Sprintf("| Total Providers | %d |\n", result.Summary.TotalProviders))
	sb.WriteString(fmt.Sprintf("| Successful | %d |\n", result.Summary.SuccessfulProviders))
	sb.WriteString(fmt.Sprintf("| Failed | %d |\n", result.Summary.FailedProviders))
	sb.WriteString(fmt.Sprintf("| Total Models | %d |\n", result.Summary.TotalModels))
	sb.WriteString(fmt.Sprintf("| Unique Models | %d |\n", result.Summary.UniqueModels))
	sb.WriteString(fmt.Sprintf("| Avg Response Time | %.0fms |\n", result.Summary.AverageResponseMs))
	sb.WriteString(fmt.Sprintf("| Duration | %v |\n", result.Duration))

	sb.WriteString("\n## Provider Results\n\n")
	sb.WriteString("| Provider | Status | Models | Response Time |\n")
	sb.WriteString("|----------|--------|--------|---------------|\n")
	for _, r := range result.Results {
		status := "Failed"
		if r.Success {
			status = "Success"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %d | %dms |\n",
			r.Provider, status, r.ModelsFound, r.ResponseTimeMs))
	}

	sb.WriteString("\n## Discovered Models\n\n")
	sb.WriteString("| Provider | Model ID | Capabilities |\n")
	sb.WriteString("|----------|----------|-------------|\n")
	for _, m := range result.AllModels {
		caps := strings.Join(m.Capabilities, ", ")
		if caps == "" {
			caps = "-"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", m.Provider, m.ID, caps))
	}

	sb.WriteString("\n---\n\n")
	sb.WriteString("*Generated by LLMsVerifier Challenges*\n")

	return sb.String()
}
