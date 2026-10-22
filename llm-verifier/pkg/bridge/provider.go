// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package bridge

import (
	"sort"

	"digital.vasic.llmsverifier/pkg/strategy"
)

// ProviderInfo describes a CLI provider's metadata for display
// and filtering purposes.
type ProviderInfo struct {
	// CLIName is the binary name (e.g. "claude").
	CLIName string `json:"cli_name"`

	// Provider is the normalized provider identifier
	// (e.g. "anthropic").
	Provider string `json:"provider"`

	// DisplayName is the human-readable provider name.
	DisplayName string `json:"display_name"`
}

// knownProviders maps CLI names to their display metadata.
var knownProviders = map[string]ProviderInfo{
	"claude": {
		CLIName:     "claude",
		Provider:    "anthropic",
		DisplayName: "Anthropic (Claude Code)",
	},
	"qwen-coder": {
		CLIName:     "qwen-coder",
		Provider:    "qwen",
		DisplayName: "Alibaba (Qwen Coder)",
	},
	"qwen": {
		CLIName:     "qwen",
		Provider:    "qwen",
		DisplayName: "Alibaba (Qwen)",
	},
	"opencode": {
		CLIName:     "opencode",
		Provider:    "opencode",
		DisplayName: "OpenCode",
	},
	"zen": {
		CLIName:     "zen",
		Provider:    "opencode",
		DisplayName: "OpenCode (Zen)",
	},
}

// GetProviderInfo returns display metadata for a CLI tool name.
// Returns a zero-value ProviderInfo with the CLI name as both
// provider and display name for unknown tools.
func GetProviderInfo(cliName string) ProviderInfo {
	if info, ok := knownProviders[cliName]; ok {
		return info
	}
	return ProviderInfo{
		CLIName:     cliName,
		Provider:    cliName,
		DisplayName: cliName,
	}
}

// BridgedToModelInfos converts a slice of BridgedModels to
// strategy.ModelInfo entries. Only available models are included.
func BridgedToModelInfos(models []BridgedModel) []strategy.ModelInfo {
	var infos []strategy.ModelInfo
	for _, m := range models {
		if m.Available {
			infos = append(infos, m.ToModelInfo())
		}
	}
	return infos
}

// SortByQuality sorts bridged models by quality score in
// descending order (highest quality first).
func SortByQuality(models []BridgedModel) {
	sort.SliceStable(models, func(i, j int) bool {
		return models[i].QualityScore > models[j].QualityScore
	})
}

// FilterVisionCapable returns only the bridged models that
// support vision input.
func FilterVisionCapable(models []BridgedModel) []BridgedModel {
	var result []BridgedModel
	for _, m := range models {
		if m.SupportsVision {
			result = append(result, m)
		}
	}
	return result
}

// FilterAvailable returns only the bridged models that are
// available (CLI binary found on PATH).
func FilterAvailable(models []BridgedModel) []BridgedModel {
	var result []BridgedModel
	for _, m := range models {
		if m.Available {
			result = append(result, m)
		}
	}
	return result
}
