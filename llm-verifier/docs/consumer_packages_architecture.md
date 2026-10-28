# Consumer packages architecture (Catalogizer / HelixQA)

Packages under `llm-verifier/pkg/`: `bridge`, `catalogizer`, `chat`, `helixqa`, `local`, `recipe`, `selector`, `strategy`, `vision`.
They were authored inside the Catalogizer superproject (plain directory `LLMsVerifier/`, later `submodules/llms_verifier`, 2026-03 to 2026-06) and moved here without content change except `pkg/helixqa` (see below). Module path: `digital.vasic.llmsverifier`.

## Environment variables: what the code actually reads

The Catalogizer copy of `.env.example` lists `LLMSVERIFIER_STRATEGY=default`. **No code in these packages reads `LLMSVERIFIER_STRATEGY`** (grep over `pkg/` finds nothing); it has no effect.
Provider API keys ARE read, as presence checks only (via an injectable `EnvChecker` in `selector`; `os.Getenv` in `catalogizer`), through the maps `providerEnvKeys` in `pkg/selector/selector.go` and `pkg/catalogizer/strategy.go`:
`ANTHROPIC_API_KEY`, `OPENAI_API_KEY`, `GOOGLE_API_KEY`, `KIMI_API_KEY`, `DEEPSEEK_API_KEY`, `GROQ_API_KEY`, `CEREBRAS_API_KEY` (both maps); additionally in `selector`: `STEPFUN_API_KEY`, `QWEN_API_KEY`, `ASTICA_API_KEY`, `NVIDIA_API_KEY`, `GITHUB_MODELS_API_KEY`, `XAI_API_KEY`, `OPENROUTER_API_KEY`, `HUGGINGFACE_API_KEY`, `CHUTES_API_KEY`, `SILICONFLOW_API_KEY`. `ollama` needs no key.

## `pkg/helixqa` change on import

Upstream held a stub `VisionModelRegistry() []VisionModel` (added 2026-04-18). The Catalogizer version returns `[]strategy.ModelInfo`; `type VisionModel = strategy.ModelInfo` is kept as an alias so callers using the old field names (Provider, Model, QualityScore, ReliabilityScore, InputCostPer1k, OutputCostPer1k, AvgLatencyMs) still compile. Guarded by `pkg/helixqa/vision_model_compat_test.go`. The old registry's 10 curated entries are not preserved; the registry now has 18 entries.

## Source documents (verbatim, nothing dropped)

Each appendix below is the unmodified Catalogizer file; headings inside are demoted by one level only by the enclosing "Appendix" heading, text is unchanged.

---

## Appendix: Catalogizer `docs/ARCHITECTURE.md`

````markdown
# LLMsVerifier Architecture

## Module Structure

```
LLMsVerifier/
├── pkg/
│   ├── strategy/      # Core Strategy pattern interface + default implementation
│   ├── vision/        # Vision-specialised strategies (Vision, Navigation, Analysis)
│   ├── chat/          # Chat/reasoning-specialised strategies (Planning)
│   ├── bridge/        # Bridged CLI model discovery (Claude Code, Qwen, OpenCode)
│   ├── recipe/        # Fluent builder for verification configurations
│   ├── helixqa/       # HelixQA-specific strategy (vision-weighted)
│   ├── catalogizer/   # Catalogizer-specific strategy
│   ├── local/         # Local model discovery via SSH/Ollama probing
│   └── selector/      # Unified model selection across cloud and local providers
├── go.mod
└── docs/
    └── ARCHITECTURE.md
```

## Design Pattern: Strategy

LLMsVerifier implements the Strategy pattern for LLM model selection:

```
                ┌──────────────────────────┐
                │  VerificationStrategy    │
                │  (interface)             │
                ├──────────────────────────┤
                │  Score(model) -> Score   │
                │  Validate(model) -> bool │
                │  Rank(models) -> ranked  │
                │  Select(ranked, reqs)    │
                └────────────┬─────────────┘
                             │
     ┌───────────┬───────────┼───────────┬───────────┬───────────┐
     │           │           │           │           │           │
┌────▼────┐ ┌───▼─────┐ ┌───▼─────┐ ┌───▼─────┐ ┌───▼────┐ ┌───▼──────┐
│ Default │ │ Vision  │ │ Navig-  │ │Analysis │ │Planning│ │ HelixQA  │
│Strategy │ │Strategy │ │ ation   │ │Strategy │ │Strategy│ │ /Catalog │
│(balanced│ │(vision) │ │Strategy │ │(descr.) │ │(reason)│ │ Strategy │
└─────────┘ └─────────┘ └─────────┘ └─────────┘ └────────┘ └──────────┘
 pkg/        pkg/        pkg/        pkg/        pkg/       pkg/helixqa
 strategy/   vision/     vision/     vision/     chat/      pkg/catalog
```

## Scoring Pipeline

```
Probe providers → Collect ModelInfo → Score each model → Rank → Filter by Requirements → Select best
```

### Dimension Weights (DefaultStrategy)

| Dimension | Weight | Source |
|-----------|--------|--------|
| Quality | 35% | Benchmark scores (0-1) |
| Speed | 25% | Inverse of latency (ms) |
| Cost | 20% | Inverse of token pricing |
| Reliability | 20% | Uptime rating (0-1) |

HelixQA strategy overrides these weights to prioritize vision capability.

### Scoring Formula

```
dimension_score = raw_value * weight
overall = sum(dimension_scores) / sum(weights)
confidence = 0.95 if verified, 0.8 otherwise (decays 10% if stale > 24h)
```

## Dual Model Types

LLMsVerifier distinguishes two model types:

| Type | `SupportsVision` | Used For | Examples |
|------|-------------------|----------|----------|
| Vision | `true` | Screenshot analysis, UI comprehension | Astica, Gemini, GPT-4o, Ollama minicpm-v |
| Chat | `false` | Planning, reasoning, report generation | GPT-4, Claude, Mistral, Groq |

The HelixQA strategy selects vision models for Execute/Curiosity phases and chat models for Learn/Plan/Analyze phases.

## Local Model Support

Ollama-hosted models are included in the scoring pool:

```
# NOTE: HELIX_OLLAMA_URL is not read by these packages; hosts are passed to ProbeLocalModels as a parameter.
                    │
                    ▼
        ┌───────────────────────┐
        │  Probe Ollama API     │
        │  GET /api/tags        │
        └───────────┬───────────┘
                    │ discovered models
                    ▼
        ┌───────────────────────┐
        │  Create ModelInfo     │
        │  cost = 0 (free)     │
        │  latency = measured   │
        │  quality = benchmark  │
        └───────────┬───────────┘
                    │
                    ▼
        ┌───────────────────────┐
        │  Score alongside      │
        │  cloud providers      │
        └───────────────────────┘
```

Local models get a perfect cost score (1.0) since they are free. They compete with cloud providers on quality, speed, and reliability dimensions.

## Thread Safety

- `DefaultStrategy` uses `sync.RWMutex` for weight access and score cache
- Score cache entries expire after 5 minutes
- All public methods are safe for concurrent use

## Tier Classification

Models are classified into tiers based on overall score:

| Tier | Score Range | Description |
|------|-------------|-------------|
| Tier 1 | >= 0.8 | Top-tier models |
| Tier 2 | 0.6 - 0.8 | Mid-tier models |
| Tier 3 | < 0.6 | Budget-tier models |

## Recipe Builder

The `pkg/recipe` package provides a fluent API for composing strategies:

```go
recipe := recipe.NewBuilder().
    WithStrategy("helix-qa").
    WithMinScore(0.6).
    WithMaxModels(5).
    WithVisionRequired(true).
    Build()
```

Presets provide pre-configured recipes for common use cases:

- `VisionPreset()` -- general vision model selection
- `NavigationPreset()` -- JSON-action navigation model selection
- `AnalysisPreset()` -- rich description analysis model selection
- `PlanningPreset()` -- reasoning/planning model selection

---

## NavigationStrategy (pkg/vision/)

Specialised strategy for UI navigation tasks. Prioritises models that produce
structured JSON action arrays from screenshots over models that return natural
language descriptions.

### Scoring Dimensions

| Dimension | Weight | Scoring Logic |
|-----------|--------|---------------|
| JSON compliance | 40% | 1.0 for json_output/structured_output capability; 0.2 otherwise |
| GUI understanding | 25% | Scaled by gui_analysis/gui_grounding/gui_navigation caps (0-4) |
| Speed | 20% | Normalised against 5s ideal (sub-1s = close to 1.0) |
| Cost | 15% | Free models = 1.0; cloud scaled by total token cost |

Non-vision models receive an overall score of 0. Vision models without
JSON output capability are heavily penalised (score 0.2 on the JSON
dimension) because navigation requires machine-parseable action arrays.

---

## AnalysisStrategy (pkg/vision/)

Specialised strategy for screenshot analysis tasks. Prioritises models that
produce rich, detailed image descriptions with OCR and object detection.

### Scoring Dimensions

| Dimension | Weight | Scoring Logic |
|-----------|--------|---------------|
| Description quality | 35% | QualityScore + bonus for vision/ocr/object_detection/face_detection caps |
| OCR accuracy | 20% | 0.95 for ocr capability; 0.4 * QualityScore otherwise |
| Object detection | 20% | 0.95 for object_detection capability; 0.35 * QualityScore otherwise |
| Comprehensiveness | 15% | Count of 6 analysis capabilities normalised to 0-1 |
| Cost | 10% | Free models = 1.0; cloud scaled by total token cost |

Models like Astica with specialised vision APIs (OCR, object detection, face
detection, content moderation) score highest. General-purpose LLMs with vision
(Gemini, GPT-4o) score lower.

---

## PlanningStrategy (pkg/chat/)

Specialised strategy for test plan generation. Prioritises models with strong
reasoning, large context windows, and structured output ability.

### Scoring Dimensions

| Dimension | Weight | Scoring Logic |
|-----------|--------|---------------|
| Reasoning quality | 35% | QualityScore + 0.10 bonus for "reasoning" capability |
| Context window | 25% | Normalised against 1M tokens (larger = better) |
| Structured output | 20% | 0.9 for json_output/structured_output; 0.7 for code; 0.3 baseline |
| Speed | 10% | Normalised against 10s (planning tolerates higher latency) |
| Cost | 10% | Free models = 1.0; cloud scaled by total token cost |

Claude and GPT-4 score highest due to strong reasoning and large context. Gemini
scores close second (1M context). Local Ollama models score lower (smaller
context, weaker reasoning).

---

## Bridged CLI Model Discovery (pkg/bridge/)

The bridge package discovers LLM models accessible via CLI coding assistants
installed on the developer's machine. Unlike cloud API models (API key required)
or local Ollama models (server required), bridged models are accessed by invoking
a CLI binary that manages its own authentication and routing.

### Discovery Flow

```
DiscoverBridgedModels()
  └─ For each known CLI spec (claude, qwen-coder, opencode):
       └─ exec.LookPath(binary_name)
            ├─ found → BridgedModel{Available: true, CLIPath: path}
            └─ not found → skip
  └─ Return []BridgedModel

BridgedModel.ToModelInfo()
  └─ strategy.ModelInfo{
       ID:           "bridged-claude-claude-sonnet-4",
       Provider:     "anthropic",
       Cost:         0 (CLI handles billing),
       Capabilities: ["text", "bridged", "coding", "vision"],
     }
```

### Known CLI Tools

| CLI | Binary Names | Provider | Model | Vision | Context |
|-----|-------------|----------|-------|--------|---------|
| Claude Code | `claude` | anthropic | claude-sonnet-4 | Yes | 200K |
| Qwen Coder | `qwen-coder`, `qwen` | qwen | qwen3-coder | No | 128K |
| OpenCode | `opencode`, `zen` | opencode | opencode | No | 128K |

Bridged models are scored alongside cloud and local providers by all strategies.
They get a perfect cost score (1.0) since they are free to the scoring system.

---

## Phase-Specific Strategy Mapping (HelixQA Integration)

When used by HelixQA's autonomous pipeline, each phase selects the optimal
strategy for its task:

```
┌──────────┐     ┌─────────────────────┐     ┌────────────────┐
│  Learn   │────►│  PlanningStrategy   │────►│ Chat model     │
│  Phase   │     │  (reasoning+context)│     │ (Gemini, Claude│
└──────────┘     └─────────────────────┘     └────────────────┘

┌──────────┐     ┌─────────────────────┐     ┌────────────────┐
│  Plan    │────►│  PlanningStrategy   │────►│ Chat model     │
│  Phase   │     │  (reasoning+struct) │     │ (Claude, GPT)  │
└──────────┘     └─────────────────────┘     └────────────────┘

┌──────────┐     ┌─────────────────────┐     ┌────────────────┐
│ Execute  │────►│ NavigationStrategy  │────►│ Vision model   │
│  Phase   │     │  (JSON+GUI+speed)   │     │ (Gemini Flash) │
└──────────┘     └─────────────────────┘     └────────────────┘

┌──────────┐     ┌─────────────────────┐     ┌────────────────┐
│Curiosity │────►│ NavigationStrategy  │────►│ Vision model   │
│  Phase   │     │  (JSON+GUI+speed)   │     │ (Gemini Flash) │
└──────────┘     └─────────────────────┘     └────────────────┘

┌──────────┐     ┌─────────────────────┐     ┌────────────────┐
│ Analyze  │────►│  AnalysisStrategy   │────►│ Vision model   │
│  Phase   │     │  (descr+OCR+detect) │     │ (Astica, GPT4o)│
└──────────┘     └─────────────────────┘     └────────────────┘
```

````

---

## Appendix: Catalogizer `ARCHITECTURE.md`

````markdown
# Architecture -- LLMsVerifier

## Purpose

Go module providing the Strategy pattern for LLM model verification, scoring, ranking, and selection. Dynamically evaluates available LLM providers and selects the best model for a given use case based on configurable criteria. Supports phase-specific strategies for vision, navigation, analysis, and planning tasks.

## Structure

```
pkg/
  strategy/      Core VerificationStrategy interface, DefaultStrategy with configurable dimension weights
  vision/        Vision-specialized strategies: VisionStrategy, NavigationStrategy, AnalysisStrategy
  chat/          Chat/reasoning-specialized: PlanningStrategy for test plan generation
  bridge/        Bridged CLI model discovery (Claude Code, Qwen Coder, OpenCode)
  recipe/        Fluent builder API for composing verification configurations with presets
  helixqa/       HelixQA-specific QA session strategy and known vision model registry
  catalogizer/   Catalogizer-specific strategy configuration
  local/         Local model discovery via SSH/Ollama probing
  selector/      Unified model selection across cloud and local providers
```

## Key Components

- **`strategy.VerificationStrategy`** -- Interface: Score, Validate, Rank, Select
- **`strategy.ModelInfo`** -- Model metadata: provider, capabilities, benchmarks, costs, context window
- **`strategy.DefaultStrategy`** -- Balanced scoring across Quality (35%), Speed (25%), Cost (20%), Reliability (20%)
- **`vision.VisionStrategy`** -- Image quality (35%), GUI detection (25%), OCR (15%), Speed (15%), Cost (10%)
- **`vision.NavigationStrategy`** -- JSON compliance (40%), GUI understanding (25%), Speed (20%), Cost (15%)
- **`vision.AnalysisStrategy`** -- Description quality (35%), OCR (20%), Object detection (20%), Comprehensiveness (15%), Cost (10%)
- **`chat.PlanningStrategy`** -- Reasoning (35%), Context window (25%), Structured output (20%), Speed (10%), Cost (10%)
- **`bridge.DiscoverBridgedModels()`** -- Discovers CLI tools on PATH, converts to ModelInfo for unified scoring
- **`local.*`** -- Probes Ollama hosts via SSH, estimates quality from parameter count

## Data Flow

```
strategy.Select(models, requirements)
    |
    filter by requirements (needs vision, max latency, min quality)
    |
    for each model: Score(model) -> weighted dimension scores -> StrategyScore
    |
    Rank(scored models) -> sort by total score -> assign tiers
    |
    Select() -> return highest-ranked model

Bridged models: bridge.DiscoverBridgedModels() -> check PATH for claude/qwen/opencode
    -> BridgedModel.ToModelInfo() -> compete alongside cloud/local in scoring pipeline
```

## Dependencies

- `github.com/stretchr/testify` -- Test assertions (only dependency)

## Testing Strategy

Table-driven tests with `testify` and race detection. Tests cover scoring dimension calculations, ranking order, requirement filtering, strategy weight customization, bridged model discovery with injectable LookPathFunc, and preset configuration validation.

````

---

## Appendix: Catalogizer `README.md`

````markdown
# LLMsVerifier

Strategy-based LLM model verification, scoring, ranking, and selection for Go. Dynamically evaluates available LLM providers and selects the best model for a given use case based on configurable criteria.

**Module:** `digital.vasic.llmsverifier`

## Features

- **Strategy pattern** -- Pluggable verification strategies with configurable dimension weights
- **7 built-in strategies** -- Default, Vision, Navigation, Analysis, Planning, QA, Catalogizer
- **Dynamic scoring** -- Models probed, scored, ranked, and selected at runtime (no hardcoded preferences)
- **Bridged CLI discovery** -- Discovers Claude Code, Qwen Coder, OpenCode on PATH
- **Local model support** -- Probes Ollama instances via SSH across distributed hosts
- **Unified selection** -- Cloud, local, and bridged models compete in a single scoring pipeline
- **Recipe builder** -- Fluent API with presets for quick strategy configuration

## Quick Start

```go
import (
    "digital.vasic.llmsverifier/pkg/strategy"
    "digital.vasic.llmsverifier/pkg/vision"
)

// Create a vision strategy for screenshot analysis
vs := vision.NewVisionStrategy()

// Score and rank available models
models := []strategy.ModelInfo{
    {Name: "gemini-2.0-flash", Provider: "google", SupportsVision: true, ...},
    {Name: "gpt-4o", Provider: "openai", SupportsVision: true, ...},
}

ranked := vs.Rank(models)
best := vs.Select(models, &strategy.Requirements{NeedsVision: true})
```

## Strategies

| Strategy | Package | Use Case |
|----------|---------|----------|
| `DefaultStrategy` | `pkg/strategy/` | General-purpose balanced selection (quality/speed/cost/reliability) |
| `VisionStrategy` | `pkg/vision/` | Screenshot analysis, UI detection, OCR |
| `NavigationStrategy` | `pkg/vision/` | JSON-action producing models for UI navigation |
| `AnalysisStrategy` | `pkg/vision/` | Rich image descriptions for screenshot analysis |
| `PlanningStrategy` | `pkg/chat/` | Test plan generation with strong reasoning |
| `QAStrategy` | `pkg/helixqa/` | Autonomous QA sessions (vision + speed + quality) |
| `CatalogizerStrategy` | `pkg/catalogizer/` | Metadata enrichment and content analysis |

## Scoring Dimensions

Each strategy weights dimensions differently. For example, VisionStrategy:

- **Image quality** (35%) -- Image understanding ability
- **GUI detection** (25%) -- UI element detection capabilities
- **OCR accuracy** (15%) -- Text recognition capability
- **Speed** (15%) -- Response latency normalized to 6s window
- **Cost** (10%) -- Free local models score 1.0; cloud models scaled by token cost

## Bridged CLI Models

`pkg/bridge/` discovers LLM models available via CLI coding assistants:

| CLI Binary | Provider | Model | Vision | Context |
|------------|----------|-------|--------|---------|
| `claude` | Anthropic | claude-sonnet-4 | Yes | 200K |
| `qwen-coder` / `qwen` | Qwen | qwen3-coder | No | 128K |
| `opencode` / `zen` | OpenCode | opencode | No | 128K |

Bridged models get zero token cost (CLI handles billing) and compete alongside cloud and local models.

## Local Model Discovery

Models running on Ollama instances are discovered via SSH probing:

```go
import "digital.vasic.llmsverifier/pkg/local"

// Probe hosts for available models
hosts := []string{"thinker.local", "amber.local"}
localModels, err := local.ProbeLocalModels(hosts, "") // or ProbeLocalModelsWithOptions
```

SSH probing always uses `StrictHostKeyChecking=yes`: hosts must already be in
known_hosts. Point at a managed file with env `LLMSVERIFIER_SSH_KNOWN_HOSTS`
(absolute path) or `ProbeOptions.KnownHostsFile`. Hosts and users are validated
and passed after `--`; option-like values are rejected.

Vision-capable local models (LLaVA, MiniCPM-V, Qwen-VL) are auto-detected by name patterns.

## Recipe Builder

```go
import "digital.vasic.llmsverifier/pkg/recipe"

// Use presets
r := recipe.VisionPreset()     // Pre-configured for vision model selection
r := recipe.NavigationPreset() // Pre-configured for navigation
r := recipe.AnalysisPreset()   // Pre-configured for analysis
r := recipe.PlanningPreset()   // Pre-configured for planning
```

## Packages

| Package | Description |
|---------|-------------|
| `pkg/strategy` | Core VerificationStrategy interface, DefaultStrategy, ModelInfo, Requirements |
| `pkg/vision` | VisionStrategy, NavigationStrategy, AnalysisStrategy |
| `pkg/chat` | PlanningStrategy for reasoning/planning tasks |
| `pkg/bridge` | Bridged CLI model discovery (Claude Code, Qwen Coder, OpenCode) |
| `pkg/recipe` | Fluent builder API with strategy presets |
| `pkg/helixqa` | HelixQA-specific QA session strategy and model registry |
| `pkg/catalogizer` | Catalogizer-specific strategy configuration |
| `pkg/local` | Local model discovery via SSH/Ollama probing |
| `pkg/selector` | Unified model selection across cloud and local providers |

## Build

```bash
go build ./...
go test ./... -race -count=1
go vet ./...
```

## License

Apache-2.0

````

---

## Appendix: Catalogizer `CLAUDE.md`

````markdown
# CLAUDE.md - LLMsVerifier Module

## MANDATORY: No CI/CD Pipelines

**NO GitHub Actions, GitLab CI/CD, or any automated pipeline may exist in this repository!**

- No `.github/workflows/` directory
- No `.gitlab-ci.yml` file
- No Jenkinsfile, .travis.yml, .circleci, or any other CI configuration
- All builds and tests are run manually or via Makefile targets
- This rule is permanent and non-negotiable

## Overview

`digital.vasic.llmsverifier` is a Go module providing the Strategy pattern for LLM model verification, scoring, ranking, and selection. It dynamically evaluates available LLM providers and selects the best model for a given use case based on configurable criteria.

**Module**: `digital.vasic.llmsverifier` (Go 1.24+)

## Build and Test

```bash
go build ./...
go test ./... -race -count=1
go vet ./...
```

## Architecture

- `pkg/strategy/` - Strategy pattern interface and default implementation
  - `interface.go` - `VerificationStrategy` interface, `ModelInfo`, `StrategyScore`, `Requirements` types
  - `default.go` - `DefaultStrategy` with configurable dimension weights
  - `default_test.go` - Table-driven tests for scoring, ranking, selection
- `pkg/vision/` - Vision-specialised strategies for model selection
  - `strategy.go` - `VisionStrategy` scoring: image quality, GUI detection, OCR, speed, cost
  - `navigation.go` - `NavigationStrategy` scoring: JSON compliance, GUI understanding, speed, cost
  - `analysis.go` - `AnalysisStrategy` scoring: description quality, OCR, object detection, comprehensiveness, cost
  - `strategy_test.go`, `navigation_test.go`, `analysis_test.go` - Comprehensive tests
- `pkg/chat/` - Chat/reasoning-specialised strategies
  - `planning.go` - `PlanningStrategy` scoring: reasoning quality, context window, structured output, speed, cost
  - `planning_test.go` - Comprehensive tests
- `pkg/bridge/` - Bridged CLI model discovery and scoring
  - `discovery.go` - Discovers CLI coding assistants on PATH (Claude Code, Qwen Coder, OpenCode)
  - `provider.go` - Converts bridged models to `strategy.ModelInfo`, filtering and sorting utilities
  - `discovery_test.go` - Tests with injectable `LookPathFunc`
- `pkg/recipe/` - Recipe builder for composing verification configurations
  - `builder.go` - Fluent builder API for strategy configuration
  - `presets.go` - Pre-built strategy presets (`VisionPreset`, `NavigationPreset`, `AnalysisPreset`, `PlanningPreset`)
  - `validator.go` - Recipe validation
- `pkg/helixqa/` - HelixQA-specific strategy and model definitions
  - `recipe.go` - HelixQA QA session strategy (vision-weighted)
  - `strategy.go` - QA-optimized strategy implementation
  - `models.go` - Known vision model registry with capabilities (12 models across 11 providers)
- `pkg/catalogizer/` - Catalogizer-specific strategy configuration
- `pkg/local/` - Local model discovery via SSH/Ollama probing
- `pkg/selector/` - Unified model selection across cloud and local providers

## Dynamic Model Selection (No Hardcoded Preferences)

LLMsVerifier uses a scoring system with configurable dimension weights:
- **Quality** (default 35%) - Model benchmark scores
- **Speed** (default 25%) - Response latency
- **Cost** (default 20%) - Token pricing
- **Reliability** (default 20%) - Uptime and consistency

Models are probed, scored, ranked, and selected at runtime. There are no hardcoded model preferences. The `Requirements` struct allows callers to specify constraints (needs vision, max latency, min quality, etc.) and the strategy filters and ranks accordingly.

## Strategy Selection Guide

LLMsVerifier provides four strategies, each optimised for a different use case:

| Strategy | Package | When to Use |
|----------|---------|-------------|
| `DefaultStrategy` | `pkg/strategy/` | General-purpose model selection with balanced quality/speed/cost/reliability weights |
| `VisionStrategy` | `pkg/vision/` | Vision model selection: screenshot analysis, UI element detection, OCR. Used by HelixQA Execute/Curiosity phases |
| `NavigationStrategy` | `pkg/vision/` | JSON-action producing vision models for UI navigation. Prioritises JSON compliance, GUI understanding, speed, cost |
| `AnalysisStrategy` | `pkg/vision/` | Rich image description models for screenshot analysis. Prioritises description quality, OCR accuracy, object detection |
| `PlanningStrategy` | `pkg/chat/` | Strong reasoning models for test plan generation. Prioritises reasoning quality, context window, structured output |
| `QAStrategy` | `pkg/helixqa/` | Autonomous QA sessions: weighs vision + speed + quality for interactive testing |
| `CatalogizerStrategy` | `pkg/catalogizer/` | Catalog-api tasks: metadata enrichment, content analysis, recommendations |

The `VisionStrategy` scores models across five vision-specific dimensions:
- **Image quality** (35%) -- image understanding ability, bonused for specialised vision capabilities
- **GUI detection** (25%) -- UI element detection (gui_analysis, gui_grounding, gui_navigation)
- **OCR accuracy** (15%) -- text recognition capability
- **Speed** (15%) -- response latency normalised to 6s window
- **Cost** (10%) -- free local models get 1.0; cloud models scaled by total token cost

Access strategies via presets or direct constructors:
- `recipe.VisionPreset()` or `vision.NewVisionStrategy()` -- general vision model selection
- `recipe.NavigationPreset()` or `vision.NewNavigationStrategy()` -- JSON-action navigation model selection
- `recipe.AnalysisPreset()` or `vision.NewAnalysisStrategy()` -- rich description analysis model selection
- `recipe.PlanningPreset()` or `chat.NewPlanningStrategy()` -- reasoning/planning model selection

The `NavigationStrategy` scores models across four navigation-specific dimensions:
- **JSON compliance** (40%) -- ability to produce structured JSON action arrays (not descriptions)
- **GUI understanding** (25%) -- GUI element detection, focus tracking, layout comprehension
- **Speed** (20%) -- response latency normalised to 5s window (interactive navigation needs fast responses)
- **Cost** (15%) -- free local models get 1.0; cloud models scaled by total token cost

The `AnalysisStrategy` scores models across five analysis-specific dimensions:
- **Description quality** (35%) -- rich image description ability with vision capability bonuses
- **OCR accuracy** (20%) -- text recognition capability (dedicated OCR models score highest)
- **Object detection** (20%) -- UI element and object identification
- **Comprehensiveness** (15%) -- breadth of vision analysis capabilities
- **Cost** (10%) -- free local models get 1.0; cloud models scaled by total token cost

The `PlanningStrategy` scores models across five planning-specific dimensions:
- **Reasoning quality** (35%) -- multi-step reasoning ability with bonus for "reasoning" capability
- **Context window** (25%) -- normalised against 1M tokens (larger = better for knowledge bases)
- **Structured output** (20%) -- JSON/structured output or code generation capability
- **Speed** (10%) -- normalised against 10s (planning tolerates higher latency)
- **Cost** (10%) -- free local models get 1.0; cloud models scaled by total token cost

## Bridged CLI Models

`pkg/bridge/` discovers LLM models available via CLI coding assistant tools installed on the developer's machine. These models are not called via HTTP APIs but through their CLI binaries, which handle authentication and billing internally.

### Discovered CLI Tools

| CLI Binary | Provider | Model | Vision | Context |
|------------|----------|-------|--------|---------|
| `claude` | Anthropic | claude-sonnet-4 | Yes | 200K |
| `qwen-coder` / `qwen` | Qwen | qwen3-coder | No | 128K |
| `opencode` / `zen` | OpenCode | opencode | No | 128K |

### How It Works

1. `DiscoverBridgedModels()` searches `PATH` for known CLI binaries
2. Found CLIs are converted to `BridgedModel` structs with capability metadata
3. `BridgedModel.ToModelInfo()` converts to `strategy.ModelInfo` for unified scoring
4. Bridged models get zero token cost (CLI handles billing) and a "bridged" capability tag
5. They compete alongside cloud and local models in the scoring/ranking pipeline

### Key Functions

- `bridge.DiscoverBridgedModels()` -- discover all available CLI tools
- `bridge.BridgedToModelInfos(models)` -- convert to `[]strategy.ModelInfo`
- `bridge.FilterVisionCapable(models)` -- filter to vision-capable CLIs only
- `bridge.GetProviderInfo(cliName)` -- get display metadata for a CLI tool

## Dual Model Types

LLMsVerifier distinguishes between two model types for HelixQA:
- **Vision models** (`SupportsVision: true`) - For screenshot analysis (Astica, Gemini, OpenAI GPT-4o, Ollama local models)
- **Chat models** - For reasoning, planning, and report generation (any text-capable model)

The HelixQA QAStrategy weights vision capability higher when selecting models for the Execute and Curiosity phases. The VisionStrategy is available for pure vision model selection without the QA-specific adjustments.

## Local Model Support

LLMsVerifier can probe and score locally-running models via Ollama:
- Models on hosts passed to `ProbeLocalModels` are discovered and included in the scoring pool
- Local models get a cost score of 1.0 (free) and compete on quality/speed/reliability
- Distributed hosts (caller-supplied list) are each probed for available models via SSH
- `pkg/local/` probes hosts, parses `ollama list` output, estimates quality from parameter count
- `pkg/selector/` combines cloud and local candidates into a unified ranked list
- Vision-capable local models (LLaVA, MiniCPM-V, Qwen-VL, etc.) are auto-detected by name patterns

## Key Interfaces

- `VerificationStrategy` - Core interface: Score, Validate, Rank, Select
- `ModelInfo` - Model metadata (provider, capabilities, benchmarks, costs)
- `StrategyScore` - Detailed scoring breakdown with confidence and reasoning
- `Requirements` - Caller-specified constraints for model selection
- `RankedModel` - Model with rank position, score, tier, and selection probability

## Functional Options

Strategies accept `StrategyOption` functions:
- `WithWeights(map[string]float64)` - Custom dimension weights
- `WithConstraints([]Constraint)` - Hard/soft verification constraints
- `WithFallbacks([]FallbackRule)` - Degradation rules for provider failures

## Code Style

- Standard Go conventions, `gofmt` formatting
- SPDX headers on every .go file
- Table-driven tests with `testify`
- Thread-safe scoring cache with `sync.RWMutex`


## ⚠️ MANDATORY: NO SUDO OR ROOT EXECUTION

**ALL operations MUST run at local user level ONLY.**

This is a PERMANENT and NON-NEGOTIABLE security constraint:

- **NEVER** use `sudo` in ANY command
- **NEVER** use `su` in ANY command
- **NEVER** execute operations as `root` user
- **NEVER** elevate privileges for file operations
- **ALL** infrastructure commands MUST use user-level container runtimes (rootless podman/docker)
- **ALL** file operations MUST be within user-accessible directories
- **ALL** service management MUST be done via user systemd or local process management
- **ALL** builds, tests, and deployments MUST run as the current user

### Container-Based Solutions
When a build or runtime environment requires system-level dependencies, use containers instead of elevation:

- **Use the `Containers` submodule** (`https://github.com/vasic-digital/Containers`) for containerized build and runtime environments
- **Add the `Containers` submodule as a Git dependency** and configure it for local use within the project
- **Build and run inside containers** to avoid any need for privilege escalation
- **Rootless Podman/Docker** is the preferred container runtime

### Why This Matters
- **Security**: Prevents accidental system-wide damage
- **Reproducibility**: User-level operations are portable across systems
- **Safety**: Limits blast radius of any issues
- **Best Practice**: Modern container workflows are rootless by design

### When You See SUDO
If any script or command suggests using `sudo` or `su`:
1. STOP immediately
2. Find a user-level alternative
3. Use rootless container runtimes
4. Use the `Containers` submodule for containerized builds
5. Modify commands to work within user permissions

**VIOLATION OF THIS CONSTRAINT IS STRICTLY PROHIBITED.**



````

---

## Appendix: Catalogizer `AGENTS.md`

````markdown
# AGENTS.md - LLMsVerifier

## MANDATORY: No CI/CD Pipelines

**NO GitHub Actions, GitLab CI/CD, or any automated pipeline may exist in this repository!**

- No `.github/workflows/` directory
- No `.gitlab-ci.yml` file
- No Jenkinsfile, .travis.yml, .circleci, or any other CI configuration
- All builds and tests are run manually or via Makefile targets
- This rule is permanent and non-negotiable

## For AI Agents Working on This Codebase

### Module Purpose
LLMsVerifier provides the Strategy pattern for dynamic LLM model verification, scoring, ranking, and selection. It is used by HelixQA to choose the best vision and chat models for QA sessions.

### Key Packages
- `pkg/strategy` — Core `VerificationStrategy` interface, `DefaultStrategy` implementation, types (`ModelInfo`, `StrategyScore`, `Requirements`, `RankedModel`)
- `pkg/recipe` — Fluent builder for composing verification configurations with presets
- `pkg/helixqa` — HelixQA-specific strategy (vision-weighted scoring), known model registry
- `pkg/catalogizer` — Catalogizer-specific strategy configuration

### Dynamic Scoring (No Hardcoded Preferences)
All model selection is score-based with configurable dimension weights:
- Quality (35%), Speed (25%), Cost (20%), Reliability (20%) — defaults
- HelixQA strategy overrides weights to prioritize vision capability
- Models are probed, scored, ranked, and selected at runtime

### Dual Model Selection
- **Vision models** (`SupportsVision: true`) — selected for screenshot analysis phases
- **Chat models** — selected for reasoning, planning, and report generation phases
- Both types go through the same scoring pipeline

### Local Model Probing
- Ollama instances are discovered via the host list passed to `ProbeLocalModels`
- Local models receive cost=1.0 (free) and compete on other dimensions
- Distributed hosts (caller-supplied list) are probed individually

### Testing
```bash
go test ./... -race -count=1
```

### Key Interfaces
- `strategy.VerificationStrategy` — Score, Validate, Rank, Select (6 methods)
- `strategy.ModelInfo` — Model metadata with capabilities and benchmarks
- `strategy.Requirements` — Constraint specification for model selection


## ⚠️ MANDATORY: NO SUDO OR ROOT EXECUTION

**ALL operations MUST run at local user level ONLY.**

This is a PERMANENT and NON-NEGOTIABLE security constraint:

- **NEVER** use `sudo` in ANY command
- **NEVER** use `su` in ANY command
- **NEVER** execute operations as `root` user
- **NEVER** elevate privileges for file operations
- **ALL** infrastructure commands MUST use user-level container runtimes (rootless podman/docker)
- **ALL** file operations MUST be within user-accessible directories
- **ALL** service management MUST be done via user systemd or local process management
- **ALL** builds, tests, and deployments MUST run as the current user

### Container-Based Solutions
When a build or runtime environment requires system-level dependencies, use containers instead of elevation:

- **Use the `Containers` submodule** (`https://github.com/vasic-digital/Containers`) for containerized build and runtime environments
- **Add the `Containers` submodule as a Git dependency** and configure it for local use within the project
- **Build and run inside containers** to avoid any need for privilege escalation
- **Rootless Podman/Docker** is the preferred container runtime

### Why This Matters
- **Security**: Prevents accidental system-wide damage
- **Reproducibility**: User-level operations are portable across systems
- **Safety**: Limits blast radius of any issues
- **Best Practice**: Modern container workflows are rootless by design

### When You See SUDO
If any script or command suggests using `sudo` or `su`:
1. STOP immediately
2. Find a user-level alternative
3. Use rootless container runtimes
4. Use the `Containers` submodule for containerized builds
5. Modify commands to work within user permissions

**VIOLATION OF THIS CONSTRAINT IS STRICTLY PROHIBITED.**



````
