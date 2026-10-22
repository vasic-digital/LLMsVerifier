// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

// Package bridge provides discovery and scoring for LLM models
// accessible via CLI coding assistants (Claude Code, Qwen Coder,
// OpenCode, etc.). These "bridged" models are not called via HTTP
// APIs but through their CLI tools, which handle authentication,
// context management, and streaming internally. The bridge package
// discovers which CLI tools are installed, maps them to their
// underlying models, and converts them to strategy.ModelInfo so
// they can be ranked alongside cloud and local providers.
package bridge

import (
	"fmt"
	"os/exec"
	"time"

	"digital.vasic.llmsverifier/pkg/strategy"
)

// BridgedModel represents a model available via a CLI coding
// assistant tool. Unlike cloud API models (which require API keys)
// or local Ollama models (which require a running server), bridged
// models are accessed by invoking a CLI binary that manages its
// own authentication and model routing.
type BridgedModel struct {
	// CLIName is the canonical name of the CLI tool
	// (e.g. "claude", "qwen-coder", "opencode").
	CLIName string `json:"cli_name"`

	// CLIPath is the resolved absolute path to the CLI binary.
	CLIPath string `json:"cli_path"`

	// Model is the underlying model identifier
	// (e.g. "claude-sonnet-4", "qwen3-coder").
	Model string `json:"model"`

	// Available indicates the CLI binary was found on PATH.
	Available bool `json:"available"`

	// SupportsVision indicates whether the CLI supports image
	// input (e.g. Claude Code can process screenshots).
	SupportsVision bool `json:"supports_vision"`

	// ContextWindow is the maximum context size in tokens.
	ContextWindow int `json:"context_window"`

	// QualityScore is the estimated quality rating (0-1).
	QualityScore float64 `json:"quality_score"`
}

// cliSpec defines the discovery parameters for a single CLI tool.
type cliSpec struct {
	// names are the binary names to search for (first match wins).
	names []string

	// model is the underlying model identifier.
	model string

	// supportsVision indicates image input capability.
	supportsVision bool

	// contextWindow is the context size in tokens.
	contextWindow int

	// qualityScore is the estimated quality (0-1).
	qualityScore float64
}

// knownCLIs defines all CLI tools that can be discovered.
// Order determines priority when multiple tools back the same
// provider.
var knownCLIs = []cliSpec{
	{
		names:          []string{"claude"},
		model:          "claude-sonnet-4",
		supportsVision: true,
		contextWindow:  200000,
		qualityScore:   0.95,
	},
	{
		names:          []string{"qwen-coder", "qwen"},
		model:          "qwen3-coder",
		supportsVision: false,
		contextWindow:  128000,
		qualityScore:   0.85,
	},
	{
		names:          []string{"opencode", "zen"},
		model:          "opencode",
		supportsVision: false,
		contextWindow:  128000,
		qualityScore:   0.80,
	},
}

// LookPathFunc abstracts exec.LookPath for testing. It takes a
// binary name and returns the resolved path or an error if not
// found.
type LookPathFunc func(file string) (string, error)

// DiscoverBridgedModels checks for available CLI coding assistant
// tools on the system PATH and returns a BridgedModel for each
// discovered tool.
func DiscoverBridgedModels() []BridgedModel {
	return DiscoverBridgedModelsWithLookPath(exec.LookPath)
}

// DiscoverBridgedModelsWithLookPath is like DiscoverBridgedModels
// but accepts a custom LookPathFunc for testing.
func DiscoverBridgedModelsWithLookPath(lookPath LookPathFunc) []BridgedModel {
	var models []BridgedModel

	for _, spec := range knownCLIs {
		if m, found := discoverCLI(spec, lookPath); found {
			models = append(models, m)
		}
	}

	return models
}

// discoverCLI tries each binary name in a cliSpec and returns
// the first match. Returns (model, true) on success.
func discoverCLI(spec cliSpec, lookPath LookPathFunc) (BridgedModel, bool) {
	for _, name := range spec.names {
		path, err := lookPath(name)
		if err == nil {
			return BridgedModel{
				CLIName:        name,
				CLIPath:        path,
				Model:          spec.model,
				Available:      true,
				SupportsVision: spec.supportsVision,
				ContextWindow:  spec.contextWindow,
				QualityScore:   spec.qualityScore,
			}, true
		}
	}
	return BridgedModel{}, false
}

// ToModelInfo converts a BridgedModel to the unified
// strategy.ModelInfo used by the scoring and selection system.
// Bridged models have zero token cost (the CLI handles billing)
// and are marked with a "bridged" capability.
func (b BridgedModel) ToModelInfo() strategy.ModelInfo {
	capabilities := []string{"text", "bridged", "coding"}
	if b.SupportsVision {
		capabilities = append(capabilities, "vision")
	}

	// Provider is derived from the CLI name.
	provider := cliToProvider(b.CLIName)

	return strategy.ModelInfo{
		ID:                      fmt.Sprintf("bridged-%s-%s", b.CLIName, b.Model),
		Name:                    fmt.Sprintf("%s (via %s CLI)", b.Model, b.CLIName),
		Provider:                provider,
		Model:                   b.Model,
		SupportsVision:          b.SupportsVision,
		SupportsStreaming:       true,
		SupportsFunctionCalling: true,
		ContextWindow:           b.ContextWindow,
		MaxOutputTokens:         16384,
		AvgLatencyMs:            2000, // CLI overhead adds latency
		InputCostPer1k:          0.0,  // CLI handles billing
		OutputCostPer1k:         0.0,
		QualityScore:            b.QualityScore,
		ReliabilityScore:        0.85, // CLI tools are generally reliable when installed
		LastVerified:            time.Now(),
		Capabilities:            capabilities,
		Metadata: map[string]any{
			"cli_name":    b.CLIName,
			"cli_path":    b.CLIPath,
			"is_bridged":  true,
			"is_local":    false,
			"access_type": "cli",
		},
	}
}

// cliToProvider maps CLI tool names to provider identifiers.
func cliToProvider(cliName string) string {
	switch cliName {
	case "claude":
		return "anthropic"
	case "qwen-coder", "qwen":
		return "qwen"
	case "opencode", "zen":
		return "opencode"
	default:
		return cliName
	}
}
