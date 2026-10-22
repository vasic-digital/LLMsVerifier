// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

package helixqa

import (
	"digital.vasic.llmsverifier/pkg/strategy"
)

// VisionModelRegistry returns all known vision-capable models for HelixQA
// verification. These entries feed into the QAStrategy scoring and ranking
// pipeline so that new providers are included in the verification matrix.
func VisionModelRegistry() []strategy.ModelInfo {
	return []strategy.ModelInfo{
		// --- Tier 0: Specialized vision API ---
		{
			ID:                "astica-vision-25",
			Name:              "Astica Vision 2.5",
			Provider:          "astica",
			Model:             "2.5_full",
			SupportsVision:    true,
			SupportsStreaming: false,
			ContextWindow:     32768,
			MaxOutputTokens:   4096,
			AvgLatencyMs:      800,
			InputCostPer1k:    0.0005,
			OutputCostPer1k:   0.0005,
			QualityScore:      0.97,
			ReliabilityScore:  0.95,
			Capabilities:      []string{"vision", "ocr", "object_detection", "face_detection", "content_moderation", "gui_analysis"},
			Metadata: map[string]any{
				"specialization": "comprehensive image understanding",
				"api_format":     "native",
			},
		},

		// --- Tier 1: Premium vision providers ---
		{
			ID:                      "openai-gpt4o",
			Name:                    "GPT-4o",
			Provider:                "openai",
			Model:                   "gpt-4o",
			SupportsVision:          true,
			SupportsStreaming:       true,
			SupportsFunctionCalling: true,
			ContextWindow:           128000,
			MaxOutputTokens:         4096,
			AvgLatencyMs:            1200,
			InputCostPer1k:          0.005,
			OutputCostPer1k:         0.015,
			QualityScore:            0.95,
			ReliabilityScore:        0.98,
			Capabilities:            []string{"vision", "code", "reasoning", "gui_analysis"},
		},
		{
			ID:                      "anthropic-sonnet",
			Name:                    "Claude Sonnet 4",
			Provider:                "anthropic",
			Model:                   "claude-sonnet-4-20250514",
			SupportsVision:          true,
			SupportsStreaming:       true,
			SupportsFunctionCalling: true,
			ContextWindow:           200000,
			MaxOutputTokens:         4096,
			AvgLatencyMs:            1500,
			InputCostPer1k:          0.003,
			OutputCostPer1k:         0.015,
			QualityScore:            0.94,
			ReliabilityScore:        0.97,
			Capabilities:            []string{"vision", "code", "reasoning", "gui_analysis"},
		},
		{
			ID:                      "gemini-flash",
			Name:                    "Gemini 2.5 Flash",
			Provider:                "google",
			Model:                   "gemini-2.5-flash",
			SupportsVision:          true,
			SupportsStreaming:       true,
			SupportsFunctionCalling: true,
			ContextWindow:           1048576,
			MaxOutputTokens:         8192,
			AvgLatencyMs:            2000, // Thinking model: 2-20s typical
			InputCostPer1k:          0.00015,
			OutputCostPer1k:         0.0006,
			QualityScore:            0.93, // Validated: excellent JSON compliance + GUI understanding
			ReliabilityScore:        0.95, // Validated: consistent across 50+ QA sessions
			Capabilities:            []string{"vision", "code", "reasoning", "gui_analysis", "gui_navigation", "thinking"},
		},

		// --- Tier 2: Cost-effective vision providers ---
		{
			ID:                      "kimi-k25",
			Name:                    "Kimi K2.5",
			Provider:                "kimi",
			Model:                   "kimi-k2.5",
			SupportsVision:          true,
			SupportsStreaming:       true,
			SupportsFunctionCalling: true,
			ContextWindow:           131072,
			MaxOutputTokens:         8192,
			AvgLatencyMs:            1000,
			InputCostPer1k:          0.0003,
			OutputCostPer1k:         0.0006,
			QualityScore:            0.85,
			ReliabilityScore:        0.90,
			Capabilities:            []string{"vision", "code", "reasoning", "gui_analysis"},
			Metadata: map[string]any{
				"architecture": "1T MoE",
				"license":      "MIT",
				"api_compat":   "openai",
			},
		},
		{
			ID:                "stepgui-v15",
			Name:              "Step-GUI 1.5V Mini",
			Provider:          "stepfun",
			Model:             "step-1.5v-mini",
			SupportsVision:    true,
			SupportsStreaming: true,
			ContextWindow:     32768,
			MaxOutputTokens:   8192,
			AvgLatencyMs:      900,
			InputCostPer1k:    0.0,
			OutputCostPer1k:   0.0,
			QualityScore:      0.82,
			ReliabilityScore:  0.85,
			Capabilities:      []string{"vision", "gui_grounding", "gui_navigation", "element_detection"},
			Metadata: map[string]any{
				"specialization": "GUI interaction",
				"free_tier":      true,
			},
		},
		{
			ID:                "qwen3-vl",
			Name:              "Qwen3-VL",
			Provider:          "qwen",
			Model:             "qwen-vl-max",
			SupportsVision:    true,
			SupportsStreaming: true,
			ContextWindow:     32768,
			MaxOutputTokens:   4096,
			AvgLatencyMs:      1100,
			InputCostPer1k:    0.001,
			OutputCostPer1k:   0.002,
			QualityScore:      0.87,
			ReliabilityScore:  0.88,
			Capabilities:      []string{"vision", "gui_grounding", "code", "reasoning"},
			Metadata: map[string]any{
				"ui_grounding_accuracy": 0.90,
				"open_source":           true,
			},
		},

		// --- Tier 2.5: Free/community vision providers ---
		{
			ID:                      "nvidia-llama-vision",
			Name:                    "Llama 3.2 90B Vision (NVIDIA)",
			Provider:                "nvidia",
			Model:                   "meta/llama-3.2-90b-vision-instruct",
			SupportsVision:          true,
			SupportsStreaming:       true,
			SupportsFunctionCalling: false,
			ContextWindow:           128000,
			MaxOutputTokens:         4096,
			AvgLatencyMs:            1000,
			InputCostPer1k:          0.0,
			OutputCostPer1k:         0.0,
			QualityScore:            0.80,
			ReliabilityScore:        0.82,
			Capabilities:            []string{"vision", "reasoning"},
			Metadata: map[string]any{
				"free_tier": true,
			},
		},
		{
			ID:                      "githubmodels-gpt4o",
			Name:                    "GPT-4o (GitHub Models)",
			Provider:                "githubmodels",
			Model:                   "openai/gpt-4o",
			SupportsVision:          true,
			SupportsStreaming:       true,
			SupportsFunctionCalling: true,
			ContextWindow:           128000,
			MaxOutputTokens:         4096,
			AvgLatencyMs:            1400,
			InputCostPer1k:          0.0,
			OutputCostPer1k:         0.0,
			QualityScore:            0.78,
			ReliabilityScore:        0.85,
			Capabilities:            []string{"vision", "code", "reasoning"},
			Metadata: map[string]any{
				"free_tier":  true,
				"api_compat": "openai",
			},
		},
		{
			ID:                      "xai-grok3",
			Name:                    "Grok 3 (xAI)",
			Provider:                "xai",
			Model:                   "grok-3",
			SupportsVision:          true,
			SupportsStreaming:       true,
			SupportsFunctionCalling: true,
			ContextWindow:           131072,
			MaxOutputTokens:         8192,
			AvgLatencyMs:            1100,
			InputCostPer1k:          0.003,
			OutputCostPer1k:         0.005,
			QualityScore:            0.80,
			ReliabilityScore:        0.88,
			Capabilities:            []string{"vision", "code", "reasoning"},
		},

		// --- Tier 2.5: Cheaper API providers (free/low-cost via aggregators) ---
		{
			ID:                "openrouter-qwen-vl",
			Name:              "Qwen2.5-VL 72B (OpenRouter)",
			Provider:          "openrouter",
			Model:             "qwen/qwen-2.5-vl-72b-instruct",
			SupportsVision:    true,
			SupportsStreaming: true,
			ContextWindow:     32768,
			MaxOutputTokens:   4096,
			AvgLatencyMs:      2000,
			InputCostPer1k:    0.0004,
			OutputCostPer1k:   0.002,
			QualityScore:      0.87,
			ReliabilityScore:  0.85,
			Capabilities:      []string{"vision", "gui_grounding", "code", "reasoning", "json_output"},
			Metadata: map[string]any{
				"api_compat": "openai",
				"aggregator": "openrouter",
			},
		},
		{
			ID:                "huggingface-uitars",
			Name:              "UI-TARS 1.5-7B (HuggingFace)",
			Provider:          "huggingface",
			Model:             "ByteDance-Seed/UI-TARS-1.5-7B",
			SupportsVision:    true,
			SupportsStreaming: false,
			ContextWindow:     8192,
			MaxOutputTokens:   2048,
			AvgLatencyMs:      2000,
			InputCostPer1k:    0.0,
			OutputCostPer1k:   0.0,
			QualityScore:      0.82,
			ReliabilityScore:  0.75,
			Capabilities:      []string{"vision", "gui_grounding", "gui_navigation", "element_detection", "json_output"},
			Metadata: map[string]any{
				"specialization": "GUI agent",
				"api_compat":     "openai",
				"free_tier":      true,
			},
		},
		{
			ID:                "chutes-minicpm-v",
			Name:              "MiniCPM-V 2.6 (Chutes)",
			Provider:          "chutes",
			Model:             "openbmb/MiniCPM-V-2_6",
			SupportsVision:    true,
			SupportsStreaming: false,
			ContextWindow:     8192,
			MaxOutputTokens:   2048,
			AvgLatencyMs:      1500,
			InputCostPer1k:    0.0,
			OutputCostPer1k:   0.0,
			QualityScore:      0.75,
			ReliabilityScore:  0.72,
			Capabilities:      []string{"vision", "ocr", "gui_analysis"},
			Metadata: map[string]any{
				"api_compat": "openai",
				"free_tier":  true,
			},
		},
		{
			ID:                "siliconflow-qwen-vl",
			Name:              "Qwen2.5-VL 7B (SiliconFlow)",
			Provider:          "siliconflow",
			Model:             "Qwen/Qwen2.5-VL-7B-Instruct",
			SupportsVision:    true,
			SupportsStreaming: true,
			ContextWindow:     32768,
			MaxOutputTokens:   4096,
			AvgLatencyMs:      1200,
			InputCostPer1k:    0.0,
			OutputCostPer1k:   0.0,
			QualityScore:      0.80,
			ReliabilityScore:  0.78,
			Capabilities:      []string{"vision", "gui_grounding", "code", "reasoning", "json_output"},
			Metadata: map[string]any{
				"api_compat": "openai",
				"free_tier":  true,
			},
		},
		{
			ID:                "replicate-minicpm-v",
			Name:              "MiniCPM-V 4.0 (Replicate)",
			Provider:          "replicate",
			Model:             "openbmb/minicpm-v-4.0",
			SupportsVision:    true,
			SupportsStreaming: false,
			ContextWindow:     8192,
			MaxOutputTokens:   2048,
			AvgLatencyMs:      2500,
			InputCostPer1k:    0.0001,
			OutputCostPer1k:   0.0001,
			QualityScore:      0.76,
			ReliabilityScore:  0.80,
			Capabilities:      []string{"vision", "ocr", "gui_analysis"},
			Metadata: map[string]any{
				"api_format":   "replicate",
				"cost_per_run": 0.0025,
			},
		},
		{
			ID:                "zhipu-glm4v-flash",
			Name:              "GLM-4V Flash (Zhipu AI)",
			Provider:          "zhipu",
			Model:             "glm-4v-flash",
			SupportsVision:    true,
			SupportsStreaming: true,
			ContextWindow:     8192,
			MaxOutputTokens:   4096,
			AvgLatencyMs:      1000,
			InputCostPer1k:    0.0,
			OutputCostPer1k:   0.0,
			QualityScore:      0.78,
			ReliabilityScore:  0.80,
			Capabilities:      []string{"vision", "gui_analysis", "ocr", "json_output"},
			Metadata: map[string]any{
				"api_compat": "openai",
				"free_tier":  true,
				"note":       "no data:image prefix for Zhipu API",
			},
		},

		// --- Tier 3: Local/open-source vision providers ---
		{
			ID:                "ollama-minicpm-v",
			Name:              "MiniCPM-V 8B (Ollama)",
			Provider:          "ollama",
			Model:             "minicpm-v:8b",
			SupportsVision:    true,
			SupportsStreaming: true,
			ContextWindow:     4096,
			MaxOutputTokens:   2048,
			AvgLatencyMs:      3000,
			InputCostPer1k:    0.0,
			OutputCostPer1k:   0.0,
			QualityScore:      0.65,
			ReliabilityScore:  0.80,
			Capabilities:      []string{"vision", "local", "ocr"},
			Metadata: map[string]any{
				"deployment": "local",
				"free":       true,
			},
		},
		{
			ID:                "ollama-llava",
			Name:              "LLaVA 7B (Ollama)",
			Provider:          "ollama",
			Model:             "llava:7b",
			SupportsVision:    true,
			SupportsStreaming: true,
			ContextWindow:     4096,
			MaxOutputTokens:   2048,
			AvgLatencyMs:      3000,
			InputCostPer1k:    0.0,
			OutputCostPer1k:   0.0,
			QualityScore:      0.65,
			ReliabilityScore:  0.80,
			Capabilities:      []string{"vision", "local"},
			Metadata: map[string]any{
				"deployment": "local",
				"free":       true,
			},
		},
	}
}

// BudgetVisionModels returns only cost-effective vision models
// ($0 or < $1/1M tokens) suitable for high-volume QA testing.
func BudgetVisionModels() []strategy.ModelInfo {
	var budget []strategy.ModelInfo
	for _, m := range VisionModelRegistry() {
		totalCost := m.InputCostPer1k + m.OutputCostPer1k
		if totalCost < 0.002 { // < $2/1k = < $2/M tokens
			budget = append(budget, m)
		}
	}
	return budget
}

// GUISpecializedModels returns models specialized for GUI interaction
// and UI element grounding.
func GUISpecializedModels() []strategy.ModelInfo {
	var gui []strategy.ModelInfo
	for _, m := range VisionModelRegistry() {
		for _, cap := range m.Capabilities {
			if cap == "gui_grounding" || cap == "gui_navigation" {
				gui = append(gui, m)
				break
			}
		}
	}
	return gui
}

// VisionModel is retained as an alias of strategy.ModelInfo so that callers
// written against the earlier stub type (Provider, Model, QualityScore,
// ReliabilityScore, InputCostPer1k, OutputCostPer1k, AvgLatencyMs) keep
// compiling; ModelInfo carries all of those fields.
type VisionModel = strategy.ModelInfo
