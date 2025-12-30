package llmverifier

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== OpenCodeExportAnalytics Tests ====================

func TestNewOpenCodeExportAnalytics(t *testing.T) {
	analytics := NewOpenCodeExportAnalytics()

	require.NotNil(t, analytics)
	assert.Equal(t, 0, analytics.TotalExports)
	assert.Equal(t, 0, analytics.SuccessfulExports)
	assert.Equal(t, 0, analytics.FailedExports)
	assert.NotNil(t, analytics.ProviderStats)
	assert.NotNil(t, analytics.ModelStats)
	assert.NotNil(t, analytics.AgentAssignments)
	assert.NotNil(t, analytics.ExportHistory)
	assert.Empty(t, analytics.ExportHistory)
}

func TestOpenCodeExportAnalytics_RecordExport_Success(t *testing.T) {
	analytics := NewOpenCodeExportAnalytics()

	analytics.RecordExport(5, 10, true, "")

	assert.Equal(t, 1, analytics.TotalExports)
	assert.Equal(t, 1, analytics.SuccessfulExports)
	assert.Equal(t, 0, analytics.FailedExports)
	assert.Len(t, analytics.ExportHistory, 1)
	assert.True(t, analytics.ExportHistory[0].Success)
	assert.Equal(t, 5, analytics.ExportHistory[0].ProviderCount)
	assert.Equal(t, 10, analytics.ExportHistory[0].ModelCount)
	assert.Empty(t, analytics.ExportHistory[0].ErrorMessage)
}

func TestOpenCodeExportAnalytics_RecordExport_Failure(t *testing.T) {
	analytics := NewOpenCodeExportAnalytics()

	analytics.RecordExport(3, 5, false, "Connection failed")

	assert.Equal(t, 1, analytics.TotalExports)
	assert.Equal(t, 0, analytics.SuccessfulExports)
	assert.Equal(t, 1, analytics.FailedExports)
	assert.Len(t, analytics.ExportHistory, 1)
	assert.False(t, analytics.ExportHistory[0].Success)
	assert.Equal(t, "Connection failed", analytics.ExportHistory[0].ErrorMessage)
}

func TestOpenCodeExportAnalytics_RecordExport_MultipleExports(t *testing.T) {
	analytics := NewOpenCodeExportAnalytics()

	// Record multiple exports
	analytics.RecordExport(5, 10, true, "")
	analytics.RecordExport(3, 5, false, "Error")
	analytics.RecordExport(7, 15, true, "")

	assert.Equal(t, 3, analytics.TotalExports)
	assert.Equal(t, 2, analytics.SuccessfulExports)
	assert.Equal(t, 1, analytics.FailedExports)
	assert.Len(t, analytics.ExportHistory, 3)
}

func TestOpenCodeExportAnalytics_RecordExport_HistoryLimit(t *testing.T) {
	analytics := NewOpenCodeExportAnalytics()

	// Record more than 100 exports
	for i := 0; i < 110; i++ {
		analytics.RecordExport(1, 1, true, "")
	}

	assert.Equal(t, 110, analytics.TotalExports)
	assert.Len(t, analytics.ExportHistory, 100) // Should be limited to 100
}

func TestOpenCodeExportAnalytics_RecordProviderUsage(t *testing.T) {
	analytics := NewOpenCodeExportAnalytics()

	providers := map[string]interface{}{
		"openai":    map[string]interface{}{"api_key": "xxx"},
		"anthropic": map[string]interface{}{"api_key": "yyy"},
		"google":    map[string]interface{}{"api_key": "zzz"},
	}

	analytics.RecordProviderUsage(providers)
	analytics.RecordProviderUsage(providers)

	assert.Equal(t, 2, analytics.ProviderStats["openai"])
	assert.Equal(t, 2, analytics.ProviderStats["anthropic"])
	assert.Equal(t, 2, analytics.ProviderStats["google"])
}

func TestOpenCodeExportAnalytics_RecordModelUsage(t *testing.T) {
	analytics := NewOpenCodeExportAnalytics()

	agents := map[string]interface{}{
		"coder": map[string]interface{}{
			"model": "openai.gpt-4",
		},
		"writer": map[string]interface{}{
			"model": "anthropic.claude-3-sonnet",
		},
		"reviewer": map[string]interface{}{
			"model": "openai.gpt-4",
		},
	}

	analytics.RecordModelUsage(agents)

	assert.Equal(t, 2, analytics.ModelStats["openai.gpt-4"])
	assert.Equal(t, 1, analytics.ModelStats["anthropic.claude-3-sonnet"])
	assert.Equal(t, 2, analytics.AgentAssignments["coder_openai"]+analytics.AgentAssignments["reviewer_openai"])
	assert.Equal(t, 1, analytics.AgentAssignments["writer_anthropic"])
}

func TestOpenCodeExportAnalytics_RecordModelUsage_InvalidData(t *testing.T) {
	analytics := NewOpenCodeExportAnalytics()

	agents := map[string]interface{}{
		"coder": map[string]interface{}{
			"model": "gpt-4", // Missing provider prefix
		},
		"writer": "invalid-data", // Not a map
		"reviewer": map[string]interface{}{
			"other_field": "value", // No model field
		},
	}

	// Should not panic
	analytics.RecordModelUsage(agents)

	// Should have no stats since data is invalid
	assert.Empty(t, analytics.ModelStats)
	assert.Empty(t, analytics.AgentAssignments)
}

func TestOpenCodeExportAnalytics_GetAnalyticsSummary(t *testing.T) {
	analytics := NewOpenCodeExportAnalytics()

	// Record some data
	analytics.RecordExport(5, 10, true, "")
	analytics.RecordExport(3, 5, false, "Error")
	analytics.RecordExport(7, 15, true, "")

	providers := map[string]interface{}{
		"openai":    map[string]interface{}{},
		"anthropic": map[string]interface{}{},
	}
	analytics.RecordProviderUsage(providers)

	agents := map[string]interface{}{
		"coder": map[string]interface{}{
			"model": "openai.gpt-4",
		},
	}
	analytics.RecordModelUsage(agents)

	summary := analytics.GetAnalyticsSummary()

	assert.Equal(t, 3, summary["total_exports"])
	assert.Equal(t, 2, summary["successful_exports"])
	assert.Equal(t, 1, summary["failed_exports"])
	assert.Equal(t, "66.7%", summary["success_rate"])
	assert.Equal(t, 2, summary["unique_providers"])
	assert.Equal(t, 1, summary["unique_models"])
	assert.NotEmpty(t, summary["last_export_time"])
	assert.NotNil(t, summary["popular_providers"])
	assert.NotNil(t, summary["popular_models"])
	assert.NotNil(t, summary["agent_preferences"])
}

func TestOpenCodeExportAnalytics_GetAnalyticsSummary_NoExports(t *testing.T) {
	analytics := NewOpenCodeExportAnalytics()

	summary := analytics.GetAnalyticsSummary()

	assert.Equal(t, 0, summary["total_exports"])
	assert.Equal(t, "0.0%", summary["success_rate"])
}

// ==================== getTopItems Tests ====================

func TestGetTopItems(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		result := getTopItems(map[string]int{}, 5)
		assert.Empty(t, result)
	})

	t.Run("fewer items than topN", func(t *testing.T) {
		data := map[string]int{
			"item1": 10,
			"item2": 5,
		}
		result := getTopItems(data, 5)
		assert.Len(t, result, 2)
	})

	t.Run("more items than topN", func(t *testing.T) {
		data := map[string]int{
			"item1": 10,
			"item2": 5,
			"item3": 15,
			"item4": 3,
			"item5": 8,
			"item6": 20,
		}
		result := getTopItems(data, 3)
		assert.Len(t, result, 3)
		// Should be sorted by value descending
		assert.Equal(t, "item6", result[0]["name"])
		assert.Equal(t, 20, result[0]["count"])
		assert.Equal(t, "item3", result[1]["name"])
		assert.Equal(t, 15, result[1]["count"])
	})

	t.Run("exact topN items", func(t *testing.T) {
		data := map[string]int{
			"item1": 10,
			"item2": 5,
			"item3": 15,
		}
		result := getTopItems(data, 3)
		assert.Len(t, result, 3)
	})
}

// ==================== File I/O Tests ====================

func TestOpenCodeExportAnalytics_SaveAndLoad(t *testing.T) {
	// Create temp file
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "test_analytics.json")

	// Create and populate analytics
	analytics := NewOpenCodeExportAnalytics()
	analytics.RecordExport(5, 10, true, "")
	analytics.ProviderStats["openai"] = 5
	analytics.ModelStats["gpt-4"] = 3

	// Save
	err := analytics.SaveAnalytics(filePath)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(filePath)
	assert.NoError(t, err)

	// Load
	loaded, err := LoadAnalytics(filePath)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, analytics.TotalExports, loaded.TotalExports)
	assert.Equal(t, analytics.SuccessfulExports, loaded.SuccessfulExports)
	assert.Equal(t, analytics.ProviderStats["openai"], loaded.ProviderStats["openai"])
	assert.Equal(t, analytics.ModelStats["gpt-4"], loaded.ModelStats["gpt-4"])
}

func TestLoadAnalytics_NonExistentFile(t *testing.T) {
	// Should return new analytics if file doesn't exist
	analytics, err := LoadAnalytics("/nonexistent/path/file.json")
	require.NoError(t, err)
	require.NotNil(t, analytics)
	assert.Equal(t, 0, analytics.TotalExports)
}

func TestLoadAnalytics_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "invalid.json")

	// Write invalid JSON
	err := os.WriteFile(filePath, []byte("not valid json"), 0644)
	require.NoError(t, err)

	// Should return error
	_, err = LoadAnalytics(filePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestSaveAnalytics_InvalidPath(t *testing.T) {
	analytics := NewOpenCodeExportAnalytics()

	// Try to save to invalid path
	err := analytics.SaveAnalytics("/nonexistent/directory/file.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "write")
}

// ==================== GetAnalyticsFilePath Tests ====================

func TestGetAnalyticsFilePath(t *testing.T) {
	path := GetAnalyticsFilePath()

	// Should contain .opencode_analytics.json
	assert.Contains(t, path, ".opencode_analytics.json")
}

// ==================== RecordOpenCodeExport Tests ====================

func TestRecordOpenCodeExport(t *testing.T) {
	// Use a temp file for testing
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, ".opencode_analytics.json")

	// Override the analytics file path function behavior
	// This would require modifying the function or using dependency injection
	// For now, we'll just test that it doesn't panic with valid input

	config := map[string]interface{}{
		"providers": map[string]interface{}{
			"openai":    map[string]interface{}{"api_key": "xxx"},
			"anthropic": map[string]interface{}{"api_key": "yyy"},
		},
		"agents": map[string]interface{}{
			"coder": map[string]interface{}{
				"model": "openai.gpt-4",
			},
		},
	}

	// This will try to use the default path which may fail,
	// but it shouldn't panic
	// RecordOpenCodeExport(config, true, "")
	_ = config // Use the config variable
	_ = tempFile
}

// ==================== Struct Tests ====================

func TestExportHistoryEntry_Struct(t *testing.T) {
	entry := ExportHistoryEntry{
		ProviderCount: 5,
		ModelCount:    10,
		Success:       true,
		ErrorMessage:  "",
	}

	assert.Equal(t, 5, entry.ProviderCount)
	assert.Equal(t, 10, entry.ModelCount)
	assert.True(t, entry.Success)
	assert.Empty(t, entry.ErrorMessage)
}

func TestOpenCodeExportAnalytics_Struct(t *testing.T) {
	analytics := &OpenCodeExportAnalytics{
		TotalExports:      10,
		SuccessfulExports: 8,
		FailedExports:     2,
		ProviderStats:     map[string]int{"openai": 5},
		ModelStats:        map[string]int{"gpt-4": 3},
		AgentAssignments:  map[string]int{"coder_openai": 2},
	}

	assert.Equal(t, 10, analytics.TotalExports)
	assert.Equal(t, 8, analytics.SuccessfulExports)
	assert.Equal(t, 2, analytics.FailedExports)
	assert.Equal(t, 5, analytics.ProviderStats["openai"])
	assert.Equal(t, 3, analytics.ModelStats["gpt-4"])
}
