// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package bridge

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLookPath returns a LookPathFunc that resolves the given
// binary names to fake paths and returns an error for all others.
func mockLookPath(available map[string]string) LookPathFunc {
	return func(file string) (string, error) {
		if path, ok := available[file]; ok {
			return path, nil
		}
		return "", fmt.Errorf("exec: %q: executable file not found in $PATH", file)
	}
}

// --- DiscoverBridgedModelsWithLookPath tests ---

func TestDiscover_AllCLIsFound(t *testing.T) {
	lookup := mockLookPath(map[string]string{
		"claude":   "/usr/bin/claude",
		"qwen":     "/usr/local/bin/qwen",
		"opencode": "/usr/bin/opencode",
	})

	models := DiscoverBridgedModelsWithLookPath(lookup)

	require.Len(t, models, 3)

	// Claude should be first (knownCLIs order).
	assert.Equal(t, "claude", models[0].CLIName)
	assert.Equal(t, "/usr/bin/claude", models[0].CLIPath)
	assert.Equal(t, "claude-sonnet-4", models[0].Model)
	assert.True(t, models[0].Available)
	assert.True(t, models[0].SupportsVision)
	assert.Equal(t, 200000, models[0].ContextWindow)
	assert.Equal(t, 0.95, models[0].QualityScore)

	// Qwen (first alternative found).
	assert.Equal(t, "qwen", models[1].CLIName)
	assert.Equal(t, "qwen3-coder", models[1].Model)

	// OpenCode.
	assert.Equal(t, "opencode", models[2].CLIName)
	assert.Equal(t, "opencode", models[2].Model)
}

func TestDiscover_NoCLIsAvailable(t *testing.T) {
	lookup := mockLookPath(map[string]string{})

	models := DiscoverBridgedModelsWithLookPath(lookup)

	assert.Empty(t, models)
}

func TestDiscover_OnlyClaudeAvailable(t *testing.T) {
	lookup := mockLookPath(map[string]string{
		"claude": "/home/user/.local/bin/claude",
	})

	models := DiscoverBridgedModelsWithLookPath(lookup)

	require.Len(t, models, 1)
	assert.Equal(t, "claude", models[0].CLIName)
	assert.Equal(t, "/home/user/.local/bin/claude", models[0].CLIPath)
	assert.True(t, models[0].SupportsVision)
}

func TestDiscover_QwenCoderPreferredOverQwen(t *testing.T) {
	// When both qwen-coder and qwen are available, qwen-coder
	// should be chosen (it appears first in the names list).
	lookup := mockLookPath(map[string]string{
		"qwen-coder": "/usr/bin/qwen-coder",
		"qwen":       "/usr/bin/qwen",
	})

	models := DiscoverBridgedModelsWithLookPath(lookup)

	require.Len(t, models, 1)
	assert.Equal(t, "qwen-coder", models[0].CLIName)
	assert.Equal(t, "/usr/bin/qwen-coder", models[0].CLIPath)
	assert.Equal(t, "qwen3-coder", models[0].Model)
}

func TestDiscover_ZenFallbackForOpenCode(t *testing.T) {
	// When opencode is missing but zen is available, zen should
	// be discovered.
	lookup := mockLookPath(map[string]string{
		"zen": "/usr/local/bin/zen",
	})

	models := DiscoverBridgedModelsWithLookPath(lookup)

	require.Len(t, models, 1)
	assert.Equal(t, "zen", models[0].CLIName)
	assert.Equal(t, "/usr/local/bin/zen", models[0].CLIPath)
	assert.Equal(t, "opencode", models[0].Model)
	assert.Equal(t, 0.80, models[0].QualityScore)
}

func TestDiscover_QwenFallbackOnly(t *testing.T) {
	// qwen-coder missing, qwen present.
	lookup := mockLookPath(map[string]string{
		"qwen": "/opt/bin/qwen",
	})

	models := DiscoverBridgedModelsWithLookPath(lookup)

	require.Len(t, models, 1)
	assert.Equal(t, "qwen", models[0].CLIName)
	assert.Equal(t, "/opt/bin/qwen", models[0].CLIPath)
}

// --- ToModelInfo tests ---

func TestToModelInfo_Claude(t *testing.T) {
	b := BridgedModel{
		CLIName:        "claude",
		CLIPath:        "/usr/bin/claude",
		Model:          "claude-sonnet-4",
		Available:      true,
		SupportsVision: true,
		ContextWindow:  200000,
		QualityScore:   0.95,
	}

	info := b.ToModelInfo()

	assert.Equal(t, "bridged-claude-claude-sonnet-4", info.ID)
	assert.Contains(t, info.Name, "claude-sonnet-4")
	assert.Contains(t, info.Name, "claude CLI")
	assert.Equal(t, "anthropic", info.Provider)
	assert.Equal(t, "claude-sonnet-4", info.Model)
	assert.True(t, info.SupportsVision)
	assert.True(t, info.SupportsStreaming)
	assert.True(t, info.SupportsFunctionCalling)
	assert.Equal(t, 200000, info.ContextWindow)
	assert.Equal(t, 16384, info.MaxOutputTokens)
	assert.Equal(t, 0.0, info.InputCostPer1k)
	assert.Equal(t, 0.0, info.OutputCostPer1k)
	assert.Equal(t, 0.95, info.QualityScore)
	assert.Equal(t, 0.85, info.ReliabilityScore)
	assert.Contains(t, info.Capabilities, "text")
	assert.Contains(t, info.Capabilities, "bridged")
	assert.Contains(t, info.Capabilities, "coding")
	assert.Contains(t, info.Capabilities, "vision")

	// Metadata checks.
	assert.Equal(t, "claude", info.Metadata["cli_name"])
	assert.Equal(t, "/usr/bin/claude", info.Metadata["cli_path"])
	assert.Equal(t, true, info.Metadata["is_bridged"])
	assert.Equal(t, false, info.Metadata["is_local"])
	assert.Equal(t, "cli", info.Metadata["access_type"])
}

func TestToModelInfo_NonVisionModel(t *testing.T) {
	b := BridgedModel{
		CLIName:        "qwen-coder",
		CLIPath:        "/usr/bin/qwen-coder",
		Model:          "qwen3-coder",
		Available:      true,
		SupportsVision: false,
		ContextWindow:  128000,
		QualityScore:   0.85,
	}

	info := b.ToModelInfo()

	assert.Equal(t, "bridged-qwen-coder-qwen3-coder", info.ID)
	assert.Equal(t, "qwen", info.Provider)
	assert.False(t, info.SupportsVision)
	assert.NotContains(t, info.Capabilities, "vision")
	assert.Contains(t, info.Capabilities, "text")
	assert.Contains(t, info.Capabilities, "bridged")
	assert.Contains(t, info.Capabilities, "coding")
}

func TestToModelInfo_OpenCodeProvider(t *testing.T) {
	b := BridgedModel{
		CLIName:        "zen",
		CLIPath:        "/usr/bin/zen",
		Model:          "opencode",
		Available:      true,
		SupportsVision: false,
		ContextWindow:  128000,
		QualityScore:   0.80,
	}

	info := b.ToModelInfo()

	assert.Equal(t, "bridged-zen-opencode", info.ID)
	assert.Equal(t, "opencode", info.Provider)
	assert.Contains(t, info.Name, "zen CLI")
}

// --- cliToProvider tests ---

func TestCliToProvider(t *testing.T) {
	tests := []struct {
		cliName  string
		expected string
	}{
		{"claude", "anthropic"},
		{"qwen-coder", "qwen"},
		{"qwen", "qwen"},
		{"opencode", "opencode"},
		{"zen", "opencode"},
		{"unknown-tool", "unknown-tool"},
	}

	for _, tt := range tests {
		t.Run(tt.cliName, func(t *testing.T) {
			assert.Equal(t, tt.expected, cliToProvider(tt.cliName))
		})
	}
}

// --- Provider helper tests ---

func TestGetProviderInfo_KnownCLI(t *testing.T) {
	info := GetProviderInfo("claude")

	assert.Equal(t, "claude", info.CLIName)
	assert.Equal(t, "anthropic", info.Provider)
	assert.Equal(t, "Anthropic (Claude Code)", info.DisplayName)
}

func TestGetProviderInfo_UnknownCLI(t *testing.T) {
	info := GetProviderInfo("some-new-tool")

	assert.Equal(t, "some-new-tool", info.CLIName)
	assert.Equal(t, "some-new-tool", info.Provider)
	assert.Equal(t, "some-new-tool", info.DisplayName)
}

func TestBridgedToModelInfos_OnlyAvailable(t *testing.T) {
	models := []BridgedModel{
		{CLIName: "claude", Model: "claude-sonnet-4", Available: true, SupportsVision: true, ContextWindow: 200000, QualityScore: 0.95},
		{CLIName: "missing", Model: "missing-model", Available: false},
	}

	infos := BridgedToModelInfos(models)

	require.Len(t, infos, 1)
	assert.Equal(t, "claude-sonnet-4", infos[0].Model)
}

func TestBridgedToModelInfos_Empty(t *testing.T) {
	infos := BridgedToModelInfos(nil)
	assert.Empty(t, infos)
}

func TestSortByQuality(t *testing.T) {
	models := []BridgedModel{
		{CLIName: "opencode", QualityScore: 0.80},
		{CLIName: "claude", QualityScore: 0.95},
		{CLIName: "qwen", QualityScore: 0.85},
	}

	SortByQuality(models)

	assert.Equal(t, "claude", models[0].CLIName)
	assert.Equal(t, "qwen", models[1].CLIName)
	assert.Equal(t, "opencode", models[2].CLIName)
}

func TestFilterVisionCapable(t *testing.T) {
	models := []BridgedModel{
		{CLIName: "claude", SupportsVision: true},
		{CLIName: "qwen", SupportsVision: false},
		{CLIName: "opencode", SupportsVision: false},
	}

	vision := FilterVisionCapable(models)

	require.Len(t, vision, 1)
	assert.Equal(t, "claude", vision[0].CLIName)
}

func TestFilterVisionCapable_NoneCapable(t *testing.T) {
	models := []BridgedModel{
		{CLIName: "qwen", SupportsVision: false},
	}

	vision := FilterVisionCapable(models)
	assert.Empty(t, vision)
}

func TestFilterAvailable(t *testing.T) {
	models := []BridgedModel{
		{CLIName: "claude", Available: true},
		{CLIName: "qwen", Available: false},
		{CLIName: "opencode", Available: true},
	}

	available := FilterAvailable(models)

	require.Len(t, available, 2)
	assert.Equal(t, "claude", available[0].CLIName)
	assert.Equal(t, "opencode", available[1].CLIName)
}
