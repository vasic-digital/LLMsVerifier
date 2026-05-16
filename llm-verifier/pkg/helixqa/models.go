// SPDX-FileCopyrightText: 2026 Milos Vasic
// SPDX-License-Identifier: Apache-2.0

// Package helixqa exposes the LLMsVerifier-curated vision-model registry
// that HelixQA and other downstream consumers score against. The registry
// is the authoritative source of per-provider quality, reliability, cost,
// and latency metrics for vision-capable LLMs.
//
// Intended consumers:
//   - HelixQA (pkg/llm/vision_ranking.go) — mirror-checked every release
//   - Any future project that wants to pick a vision provider without
//     hard-coding model preferences.
//
// The initial registry is intentionally minimal — values come from the
// rate table documented in helix_qa/CLAUDE.md "Cost Rates" section.
// Expand per-provider entries as LLMsVerifier adds benchmark data.
package helixqa

// VisionModel describes a vision-capable LLM with the metrics HelixQA's
// scorer needs to rank providers. Zero-value fields are valid; callers
// must tolerate 0.0 scores and 0 latencies (the downstream scorer
// already handles that gracefully by assigning a low baseline).
type VisionModel struct {
	// Provider is the canonical provider name (matches Provider.Name()
	// in HelixQA's llm package — e.g. "openai", "anthropic", "google",
	// "ollama", "kimi", "astica", "qwen", "xai", "stepfun", "nvidia").
	Provider string

	// Model is the specific model identifier inside the provider's
	// catalogue (e.g. "gpt-4o", "claude-3-5-sonnet", "gemini-1.5-pro").
	// Optional for providers that expose a single vision model.
	Model string

	// QualityScore is a 0..1 benchmark-derived quality rating.
	QualityScore float64

	// ReliabilityScore is a 0..1 uptime/reliability rating.
	ReliabilityScore float64

	// InputCostPer1k is USD per 1000 input tokens.
	InputCostPer1k float64

	// OutputCostPer1k is USD per 1000 output tokens.
	OutputCostPer1k float64

	// AvgLatencyMs is the average response latency in milliseconds.
	AvgLatencyMs int
}

// VisionModelRegistry returns the curated list of vision models known to
// LLMsVerifier. The slice is a fresh copy on every call so callers may
// mutate it without affecting other readers.
//
// Rates below mirror the table in helix_qa/CLAUDE.md (§ "Cost Rates").
// Quality/reliability values are conservative starting points —
// LLMsVerifier's benchmarks override these when available.
func VisionModelRegistry() []VisionModel {
	return []VisionModel{
		{Provider: "openai", Model: "gpt-4o", QualityScore: 0.92, ReliabilityScore: 0.95, InputCostPer1k: 0.005, OutputCostPer1k: 0.015, AvgLatencyMs: 1800},
		{Provider: "anthropic", Model: "claude-3-5-sonnet", QualityScore: 0.93, ReliabilityScore: 0.94, InputCostPer1k: 0.003, OutputCostPer1k: 0.015, AvgLatencyMs: 2100},
		{Provider: "google", Model: "gemini-1.5-pro", QualityScore: 0.90, ReliabilityScore: 0.93, InputCostPer1k: 0.0001, OutputCostPer1k: 0.0004, AvgLatencyMs: 1600},
		{Provider: "kimi", Model: "moonshot-v1-vision", QualityScore: 0.85, ReliabilityScore: 0.88, InputCostPer1k: 0.0003, OutputCostPer1k: 0.0006, AvgLatencyMs: 2300},
		{Provider: "astica", Model: "astica-vision", QualityScore: 0.80, ReliabilityScore: 0.85, InputCostPer1k: 0.0005, OutputCostPer1k: 0.0005, AvgLatencyMs: 1400},
		{Provider: "qwen", Model: "qwen-vl-max", QualityScore: 0.84, ReliabilityScore: 0.88, InputCostPer1k: 0.001, OutputCostPer1k: 0.002, AvgLatencyMs: 2000},
		{Provider: "xai", Model: "grok-vision", QualityScore: 0.82, ReliabilityScore: 0.87, InputCostPer1k: 0.0025, OutputCostPer1k: 0.0025, AvgLatencyMs: 2200},
		{Provider: "ollama", Model: "llava", QualityScore: 0.72, ReliabilityScore: 0.92, InputCostPer1k: 0.0, OutputCostPer1k: 0.0, AvgLatencyMs: 3500},
		{Provider: "stepfun", Model: "step-1-8k-vision", QualityScore: 0.78, ReliabilityScore: 0.80, InputCostPer1k: 0.0, OutputCostPer1k: 0.0, AvgLatencyMs: 2100},
		{Provider: "nvidia", Model: "neva-22b", QualityScore: 0.76, ReliabilityScore: 0.78, InputCostPer1k: 0.0, OutputCostPer1k: 0.0, AvgLatencyMs: 2600},
	}
}
