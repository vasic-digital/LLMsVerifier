// Package partners provides integrations with popular AI tools and platforms
package partners

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PartnerType represents different partner integration types
type PartnerType string

const (
	PartnerTypeOpenCode   PartnerType = "opencode"
	PartnerTypeClaudeCode PartnerType = "claude_code"
	PartnerTypeCursor     PartnerType = "cursor"
	PartnerTypeVSCode     PartnerType = "vscode"
	PartnerTypeJetBrains  PartnerType = "jetbrains"
	PartnerTypeGitHub     PartnerType = "github"
	PartnerTypeCustom     PartnerType = "custom"
)

// IntegrationStatus represents the status of a partner integration
type IntegrationStatus string

const (
	IntegrationStatusActive   IntegrationStatus = "active"
	IntegrationStatusInactive IntegrationStatus = "inactive"
	IntegrationStatusError    IntegrationStatus = "error"
	IntegrationStatusPending  IntegrationStatus = "pending"
)

// PartnerIntegration represents a partner integration configuration
type PartnerIntegration struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Type          PartnerType            `json:"type"`
	Description   string                 `json:"description"`
	Version       string                 `json:"version"`
	Status        IntegrationStatus      `json:"status"`
	Configuration map[string]interface{} `json:"configuration"`
	Capabilities  []string               `json:"capabilities"`
	AuthMethods   []string               `json:"auth_methods"`
	WebhookURL    string                 `json:"webhook_url,omitempty"`
	APIEndpoints  map[string]string      `json:"api_endpoints,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	LastSync      *time.Time             `json:"last_sync,omitempty"`
	ErrorMessage  string                 `json:"error_message,omitempty"`
}

// IntegrationManager manages partner integrations
type IntegrationManager struct {
	integrations map[string]*PartnerIntegration
	httpClient   *http.Client
}

// NewIntegrationManager creates a new integration manager
func NewIntegrationManager() *IntegrationManager {
	return &IntegrationManager{
		integrations: make(map[string]*PartnerIntegration),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// RegisterIntegration registers a new partner integration
func (im *IntegrationManager) RegisterIntegration(integration *PartnerIntegration) error {
	if integration.ID == "" {
		integration.ID = uuid.New().String()
	}
	if integration.CreatedAt.IsZero() {
		integration.CreatedAt = time.Now()
	}
	integration.UpdatedAt = time.Now()

	im.integrations[integration.ID] = integration
	return nil
}

// SyncIntegration syncs data with a partner platform
func (im *IntegrationManager) SyncIntegration(ctx context.Context, id string) error {
	integration, exists := im.integrations[id]
	if !exists {
		return fmt.Errorf("integration not found: %s", id)
	}

	switch integration.Type {
	case PartnerTypeOpenCode:
		return im.syncOpenCode(ctx, integration)
	case PartnerTypeClaudeCode:
		return im.syncClaudeCode(ctx, integration)
	case PartnerTypeCursor:
		return im.syncCursor(ctx, integration)
	default:
		return fmt.Errorf("sync not supported for integration type: %s", integration.Type)
	}
}

// syncOpenCode syncs with OpenCode platform
func (im *IntegrationManager) syncOpenCode(ctx context.Context, integration *PartnerIntegration) error {
	// Get configuration
	baseURL, _ := integration.Configuration["base_url"].(string)
	apiKey, _ := integration.Configuration["api_key"].(string)

	if baseURL == "" || apiKey == "" {
		return fmt.Errorf("missing OpenCode configuration: base_url and api_key required")
	}

	// In a real implementation, this would create OpenCode integration and sync data
	// For demo, just mark as synced
	now := time.Now()
	integration.LastSync = &now

	return nil
}

// syncClaudeCode syncs with Claude Code platform
func (im *IntegrationManager) syncClaudeCode(ctx context.Context, integration *PartnerIntegration) error {
	apiKey, _ := integration.Configuration["api_key"].(string)

	if apiKey == "" {
		return fmt.Errorf("missing Claude Code configuration: api_key required")
	}

	// In a real implementation, this would create Claude Code integration and sync data
	// For demo, just mark as synced
	now := time.Now()
	integration.LastSync = &now

	return nil
}

// syncCursor syncs with Cursor IDE
func (im *IntegrationManager) syncCursor(ctx context.Context, integration *PartnerIntegration) error {
	apiKey, _ := integration.Configuration["api_key"].(string)

	if apiKey == "" {
		return fmt.Errorf("missing Cursor configuration: api_key required")
	}

	// In a real implementation, this would create Cursor integration and sync data
	// For demo, just mark as synced
	now := time.Now()
	integration.LastSync = &now

	return nil
}

// OpenCodeIntegration provides OpenCode-specific integration
type OpenCodeIntegration struct {
	baseURL    string
	apiKey     string
	projectID  string
	httpClient *http.Client
}

// NewOpenCodeIntegration creates a new OpenCode integration
func NewOpenCodeIntegration(baseURL, apiKey, projectID string) *OpenCodeIntegration {
	return &OpenCodeIntegration{
		baseURL:    strings.TrimSuffix(baseURL, "/"),
		apiKey:     apiKey,
		projectID:  projectID,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// SyncModels syncs available models from LLM Verifier to OpenCode
func (oci *OpenCodeIntegration) SyncModels(ctx context.Context, models []ModelInfo) error {
	// Create OpenCode-compatible model configurations
	opencodeModels := make([]OpenCodeModel, len(models))
	for i, model := range models {
		opencodeModels[i] = OpenCodeModel{
			ID:          model.ID,
			Name:        model.Name,
			Description: fmt.Sprintf("%s model from %s", model.Name, model.Provider),
			Provider:    string(model.Provider),
			Capabilities: []string{
				"chat",
				"completion",
			},
			ContextLength: model.ContextLength,
			Pricing: OpenCodePricing{
				InputCost:  model.InputPricing,
				OutputCost: model.OutputPricing,
				Currency:   "USD",
			},
		}
	}

	// Send to OpenCode API
	return oci.sendToOpenCode(ctx, "/api/models/sync", opencodeModels)
}

// SyncTestResults syncs test results to OpenCode
func (oci *OpenCodeIntegration) SyncTestResults(ctx context.Context, results []TestResult) error {
	// Convert to OpenCode format
	opencodeResults := make([]OpenCodeTestResult, len(results))
	for i, result := range results {
		status := "passed"
		if result.Status == "failed" {
			status = "failed"
		}

		opencodeResults[i] = OpenCodeTestResult{
			TestID:       result.TestCaseID,
			TestName:     result.TestCaseName,
			Status:       status,
			Duration:     result.Duration.Milliseconds(),
			Score:        result.Score,
			ErrorMessage: result.Error,
			Provider:     result.Provider,
			Model:        result.Model,
			Timestamp:    result.Timestamp,
		}
	}

	return oci.sendToOpenCode(ctx, "/api/tests/results", opencodeResults)
}

// SyncAnalytics syncs analytics data to OpenCode
func (oci *OpenCodeIntegration) SyncAnalytics(ctx context.Context, analytics *AnalyticsSummary) error {
	return oci.sendToOpenCode(ctx, "/api/analytics/sync", analytics)
}

// sendToOpenCode sends data to OpenCode API
func (oci *OpenCodeIntegration) sendToOpenCode(ctx context.Context, endpoint string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	url := oci.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+oci.apiKey)
	req.Header.Set("X-Project-ID", oci.projectID)

	resp, err := oci.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OpenCode API returned status: %d", resp.StatusCode)
	}

	return nil
}

// OpenCode data structures
type OpenCodeModel struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Provider      string          `json:"provider"`
	Capabilities  []string        `json:"capabilities"`
	ContextLength int             `json:"context_length"`
	Pricing       OpenCodePricing `json:"pricing"`
}

type OpenCodePricing struct {
	InputCost  float64 `json:"input_cost"`
	OutputCost float64 `json:"output_cost"`
	Currency   string  `json:"currency"`
}

type OpenCodeTestResult struct {
	TestID       string    `json:"test_id"`
	TestName     string    `json:"test_name"`
	Status       string    `json:"status"`
	Duration     int64     `json:"duration_ms"`
	Score        float64   `json:"score"`
	ErrorMessage string    `json:"error_message,omitempty"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	Timestamp    time.Time `json:"timestamp"`
}

// ClaudeCodeIntegration provides Claude Code integration
type ClaudeCodeIntegration struct {
	apiKey     string
	workspace  string
	httpClient *http.Client
}

// NewClaudeCodeIntegration creates a new Claude Code integration
func NewClaudeCodeIntegration(apiKey, workspace string) *ClaudeCodeIntegration {
	return &ClaudeCodeIntegration{
		apiKey:     apiKey,
		workspace:  workspace,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// SyncModels syncs models to Claude Code
func (cci *ClaudeCodeIntegration) SyncModels(ctx context.Context, models []ModelInfo) error {
	// Claude Code specific model format
	claudeModels := make([]ClaudeCodeModel, len(models))
	for i, model := range models {
		claudeModels[i] = ClaudeCodeModel{
			Name:          model.Name,
			Provider:      string(model.Provider),
			ContextWindow: model.ContextLength,
			MaxTokens:     model.MaxTokens,
			Pricing: ClaudeCodePricing{
				Input:  model.InputPricing,
				Output: model.OutputPricing,
			},
		}
	}

	return cci.sendToClaudeCode(ctx, "/api/models", claudeModels)
}

// SyncResults syncs test results to Claude Code
func (cci *ClaudeCodeIntegration) SyncResults(ctx context.Context, results []TestResult) error {
	claudeResults := make([]ClaudeCodeResult, len(results))
	for i, result := range results {
		claudeResults[i] = ClaudeCodeResult{
			TestName: result.TestCaseName,
			Passed:   result.Status == "passed",
			Duration: result.Duration,
			Provider: result.Provider,
			Model:    result.Model,
			Score:    result.Score,
			Error:    result.Error,
		}
	}

	return cci.sendToClaudeCode(ctx, "/api/test-results", claudeResults)
}

// sendToClaudeCode sends data to Claude Code API
func (cci *ClaudeCodeIntegration) sendToClaudeCode(ctx context.Context, endpoint string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	url := "https://api.claude.ai" + endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cci.apiKey)
	req.Header.Set("X-Workspace", cci.workspace)

	resp, err := cci.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Claude Code API returned status: %d", resp.StatusCode)
	}

	return nil
}

// Claude Code data structures
type ClaudeCodeModel struct {
	Name          string            `json:"name"`
	Provider      string            `json:"provider"`
	ContextWindow int               `json:"context_window"`
	MaxTokens     int               `json:"max_tokens"`
	Pricing       ClaudeCodePricing `json:"pricing"`
}

type ClaudeCodePricing struct {
	Input  float64 `json:"input"`
	Output float64 `json:"output"`
}

type ClaudeCodeResult struct {
	TestName string        `json:"test_name"`
	Passed   bool          `json:"passed"`
	Duration time.Duration `json:"duration"`
	Provider string        `json:"provider"`
	Model    string        `json:"model"`
	Score    float64       `json:"score"`
	Error    string        `json:"error,omitempty"`
}

// CursorIntegration provides Cursor IDE integration
type CursorIntegration struct {
	apiKey     string
	userID     string
	httpClient *http.Client
}

// NewCursorIntegration creates a new Cursor integration
func NewCursorIntegration(apiKey, userID string) *CursorIntegration {
	return &CursorIntegration{
		apiKey:     apiKey,
		userID:     userID,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// SyncModels syncs models to Cursor
func (ci *CursorIntegration) SyncModels(ctx context.Context, models []ModelInfo) error {
	cursorModels := make([]CursorModel, len(models))
	for i, model := range models {
		cursorModels[i] = CursorModel{
			ID:            model.ID,
			Name:          model.Name,
			Provider:      string(model.Provider),
			Capabilities:  model.Capabilities,
			ContextLength: model.ContextLength,
			MaxTokens:     model.MaxTokens,
		}
	}

	return ci.sendToCursor(ctx, "/api/models/sync", cursorModels)
}

// SyncCompletions syncs completion data to Cursor
func (ci *CursorIntegration) SyncCompletions(ctx context.Context, completions []CompletionData) error {
	return ci.sendToCursor(ctx, "/api/completions/sync", completions)
}

// sendToCursor sends data to Cursor API
func (ci *CursorIntegration) sendToCursor(ctx context.Context, endpoint string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	url := "https://api.cursor.sh" + endpoint
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ci.apiKey)
	req.Header.Set("X-User-ID", ci.userID)

	resp, err := ci.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Cursor API returned status: %d", resp.StatusCode)
	}

	return nil
}

// Cursor data structures
type CursorModel struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Provider      string   `json:"provider"`
	Capabilities  []string `json:"capabilities"`
	ContextLength int      `json:"context_length"`
	MaxTokens     int      `json:"max_tokens"`
}

type CompletionData struct {
	Prompt       string  `json:"prompt"`
	Completion   string  `json:"completion"`
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	TokensUsed   int     `json:"tokens_used"`
	ResponseTime float64 `json:"response_time"`
}

// Helper types (these would be imported from other packages in real implementation)
type ModelInfo struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Provider      string   `json:"provider"`
	Capabilities  []string `json:"capabilities"`
	ContextLength int      `json:"context_length"`
	MaxTokens     int      `json:"max_tokens"`
	InputPricing  float64  `json:"input_pricing"`
	OutputPricing float64  `json:"output_pricing"`
}

type TestResult struct {
	TestCaseID   string                 `json:"test_case_id"`
	TestCaseName string                 `json:"test_case_name"`
	Status       string                 `json:"status"`
	Duration     time.Duration          `json:"duration"`
	Response     string                 `json:"response,omitempty"`
	Error        string                 `json:"error,omitempty"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
	Score        float64                `json:"score,omitempty"`
	Provider     string                 `json:"provider,omitempty"`
	Model        string                 `json:"model,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
}

type AnalyticsSummary struct {
	TotalRequests    int                `json:"total_requests"`
	TotalCost        float64            `json:"total_cost"`
	AvgResponseTime  time.Duration      `json:"avg_response_time"`
	TopProviders     []ProviderStats    `json:"top_providers"`
	CostBreakdown    map[string]float64 `json:"cost_breakdown"`
	PerformanceTrend string             `json:"performance_trend"`
	GeneratedAt      time.Time          `json:"generated_at"`
}

type ProviderStats struct {
	Provider     string        `json:"provider"`
	RequestCount int           `json:"request_count"`
	SuccessRate  float64       `json:"success_rate"`
	AvgLatency   time.Duration `json:"avg_latency"`
	TotalCost    float64       `json:"total_cost"`
}
