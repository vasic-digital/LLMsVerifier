package llmverifier

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// OpenCodeExportAnalytics tracks analytics for OpenCode configuration exports
type OpenCodeExportAnalytics struct {
	TotalExports      int                  `json:"total_exports"`
	SuccessfulExports int                  `json:"successful_exports"`
	FailedExports     int                  `json:"failed_exports"`
	ProviderStats     map[string]int       `json:"provider_stats"`
	ModelStats        map[string]int       `json:"model_stats"`
	AgentAssignments  map[string]int       `json:"agent_assignments"`
	LastExportTime    time.Time            `json:"last_export_time"`
	ExportHistory     []ExportHistoryEntry `json:"export_history"`
}

// ExportHistoryEntry represents a single export event
type ExportHistoryEntry struct {
	Timestamp     time.Time `json:"timestamp"`
	ProviderCount int       `json:"provider_count"`
	ModelCount    int       `json:"model_count"`
	Success       bool      `json:"success"`
	ErrorMessage  string    `json:"error_message,omitempty"`
}

// NewOpenCodeExportAnalytics creates a new analytics tracker
func NewOpenCodeExportAnalytics() *OpenCodeExportAnalytics {
	return &OpenCodeExportAnalytics{
		ProviderStats:    make(map[string]int),
		ModelStats:       make(map[string]int),
		AgentAssignments: make(map[string]int),
		ExportHistory:    make([]ExportHistoryEntry, 0),
	}
}

// RecordExport records an export event
func (a *OpenCodeExportAnalytics) RecordExport(providerCount, modelCount int, success bool, errorMsg string) {
	a.TotalExports++
	a.LastExportTime = time.Now()

	if success {
		a.SuccessfulExports++
	} else {
		a.FailedExports++
	}

	entry := ExportHistoryEntry{
		Timestamp:     a.LastExportTime,
		ProviderCount: providerCount,
		ModelCount:    modelCount,
		Success:       success,
		ErrorMessage:  errorMsg,
	}

	a.ExportHistory = append(a.ExportHistory, entry)

	// Keep only last 100 entries
	if len(a.ExportHistory) > 100 {
		a.ExportHistory = a.ExportHistory[len(a.ExportHistory)-100:]
	}
}

// RecordProviderUsage records which providers are being used
func (a *OpenCodeExportAnalytics) RecordProviderUsage(providers map[string]interface{}) {
	for providerName := range providers {
		a.ProviderStats[providerName]++
	}
}

// RecordModelUsage records which models are being assigned to agents
func (a *OpenCodeExportAnalytics) RecordModelUsage(agents map[string]interface{}) {
	for agentName, agentData := range agents {
		if agentMap, ok := agentData.(map[string]interface{}); ok {
			if model, hasModel := agentMap["model"]; hasModel {
				if modelStr, ok := model.(string); ok {
					// Extract provider from model reference (provider.model)
					if parts := strings.SplitN(modelStr, ".", 2); len(parts) == 2 {
						a.AgentAssignments[agentName+"_"+parts[0]]++
						a.ModelStats[modelStr]++
					}
				}
			}
		}
	}
}

// GetAnalyticsSummary returns a summary of export analytics
func (a *OpenCodeExportAnalytics) GetAnalyticsSummary() map[string]interface{} {
	successRate := float64(0)
	if a.TotalExports > 0 {
		successRate = float64(a.SuccessfulExports) / float64(a.TotalExports) * 100
	}

	return map[string]interface{}{
		"total_exports":      a.TotalExports,
		"successful_exports": a.SuccessfulExports,
		"failed_exports":     a.FailedExports,
		"success_rate":       fmt.Sprintf("%.1f%%", successRate),
		"unique_providers":   len(a.ProviderStats),
		"unique_models":      len(a.ModelStats),
		"last_export_time":   a.LastExportTime.Format(time.RFC3339),
		"popular_providers":  getTopItems(a.ProviderStats, 5),
		"popular_models":     getTopItems(a.ModelStats, 5),
		"agent_preferences":  getTopItems(a.AgentAssignments, 10),
	}
}

// getTopItems returns the top N items from a map by value
func getTopItems(data map[string]int, topN int) []map[string]interface{} {
	type item struct {
		key   string
		value int
	}

	var items []item
	for k, v := range data {
		items = append(items, item{k, v})
	}

	// Sort by value descending
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].value < items[j].value {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	// Take top N
	result := make([]map[string]interface{}, 0, topN)
	for i := 0; i < len(items) && i < topN; i++ {
		result = append(result, map[string]interface{}{
			"name":  items[i].key,
			"count": items[i].value,
		})
	}

	return result
}

// SaveAnalytics saves analytics to a JSON file
func (a *OpenCodeExportAnalytics) SaveAnalytics(filePath string) error {
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal analytics: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write analytics file: %w", err)
	}

	return nil
}

// LoadAnalytics loads analytics from a JSON file
func LoadAnalytics(filePath string) (*OpenCodeExportAnalytics, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return new analytics if file doesn't exist
			return NewOpenCodeExportAnalytics(), nil
		}
		return nil, fmt.Errorf("failed to read analytics file: %w", err)
	}

	var analytics OpenCodeExportAnalytics
	if err := json.Unmarshal(data, &analytics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal analytics: %w", err)
	}

	return &analytics, nil
}

// GetAnalyticsFilePath returns the standard analytics file path
func GetAnalyticsFilePath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ".opencode_analytics.json"
	}
	return filepath.Join(homeDir, ".opencode_analytics.json")
}

// RecordOpenCodeExport records analytics for an OpenCode export
func RecordOpenCodeExport(config map[string]interface{}, success bool, errorMsg string) {
	analyticsFile := GetAnalyticsFilePath()

	// Load existing analytics
	analytics, err := LoadAnalytics(analyticsFile)
	if err != nil {
		// Create new analytics if loading fails
		analytics = NewOpenCodeExportAnalytics()
	}

	// Count providers and models
	providerCount := 0
	modelCount := 0

	if providers, ok := config["providers"].(map[string]interface{}); ok {
		providerCount = len(providers)
		analytics.RecordProviderUsage(providers)
	}

	if agents, ok := config["agents"].(map[string]interface{}); ok {
		// Estimate model count from agents
		modelCount = len(agents)
		analytics.RecordModelUsage(agents)
	}

	// Record the export
	analytics.RecordExport(providerCount, modelCount, success, errorMsg)

	// Save analytics (ignore errors to not interrupt main flow)
	analytics.SaveAnalytics(analyticsFile)
}
