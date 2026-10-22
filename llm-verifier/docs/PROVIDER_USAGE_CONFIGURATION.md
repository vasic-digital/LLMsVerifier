# Provider Usage Configuration

**Version:** 1.0.0  
**Last Updated:** 2026-04-08  
**Applies To:** LLMsVerifier v1.1.0+, HelixQA v1.0.0+

## Overview

The Provider Usage Configuration system allows fine-grained control over which LLM providers are used for specific purposes. This is particularly important for managing costs while maximizing quality:

- **Vision-only providers** (e.g., Gemini) are reserved exclusively for vision tasks like screenshot analysis and UI element detection
- **LLM-only providers** (e.g., Groq, Cerebras) are optimized for text generation, reasoning, and planning
- **General providers** (e.g., Anthropic, OpenAI) can be used for any task

## Quick Start

```go
import "digital.vasic.llmsverifier/pkg/strategy"

// Use the default configuration
config := strategy.DefaultProviderUsageConfig()

// Gemini is vision-only by default
usage := config.GetUsage("google")
// usage == strategy.UsageTypeVisionOnly

// Check if provider can be used for specific tasks
visionOK := config.CanUseForVision("google", true)  // true
llmOK := config.CanUseForLLM("google")              // false
```

## Provider Usage Types

### `UsageTypeAny`

The provider can be used for any purpose (vision and LLM tasks). This is the default.

**Example Providers:**
- `anthropic` (Claude)
- `openai` (GPT)
- `openrouter`
- `ollama` (local)

### `UsageTypeVisionOnly`

The provider should **ONLY** be used for vision-related tasks. These are typically expensive but excellent at image understanding.

**Example Providers:**
- `google` / `gemini` (Gemini Flash/Pro)
- `vertex` / `vertexai`

**Use Cases:**
- Screenshot analysis
- UI element detection
- Visual regression detection
- Image-based navigation decisions

### `UsageTypeLLMOnly`

The provider should **ONLY** be used for general LLM chat tasks. These may lack vision or have cost structures optimized for text.

**Example Providers:**
- `groq` (Llama via Groq - fast, cheap)
- `cerebras` (Llama via Cerebras)
- `deepseek` (DeepSeek Chat)

**Use Cases:**
- Test planning
- Code analysis
- Reasoning tasks
- Chat-based interactions

## Default Configuration

The default configuration is designed to balance cost and quality:

```go
&ProviderUsageConfig{
    ProviderUsages: map[string]ProviderUsageType{
        // Gemini is expensive - reserve for vision tasks only
        "google":  UsageTypeVisionOnly,
        "gemini":  UsageTypeVisionOnly,

        // These providers are optimized for text/LLM tasks
        "groq":     UsageTypeLLMOnly,
        "cerebras": UsageTypeLLMOnly,
        "deepseek": UsageTypeLLMOnly,

        // Local providers can do everything
        "ollama":  UsageTypeAny,
        "ui-tars": UsageTypeAny,

        // These are good general-purpose providers
        "anthropic":  UsageTypeAny,
        "openai":     UsageTypeAny,
        "openrouter": UsageTypeAny,
    },
    DefaultUsage: UsageTypeAny,
    StrictMode:   false,
}
```

## Custom Configuration

### YAML Configuration

```yaml
# llmsverifier-config.yaml
provider_usages:
  google: vision_only      # Gemini - expensive vision
  anthropic: any           # Claude - general purpose
  groq: llm_only           # Fast, cheap text
  openai: any              # GPT - general purpose

default_usage: any
strict_mode: false
```

### Programmatic Configuration

```go
import (
    "digital.vasic.llmsverifier/pkg/strategy"
    "digital.vasic.llmsverifier/pkg/vision"
    "digital.vasic.llmsverifier/pkg/catalogizer"
)

// Create custom usage config
usageConfig := &strategy.ProviderUsageConfig{
    ProviderUsages: map[string]strategy.ProviderUsageType{
        "google":     strategy.UsageTypeVisionOnly,
        "anthropic":  strategy.UsageTypeAny,
        "openai":     strategy.UsageTypeAny,
        "groq":       strategy.UsageTypeLLMOnly,
    },
    DefaultUsage: strategy.UsageTypeAny,
    StrictMode:   false,
}

// Use with VisionStrategy
visionStrat := vision.NewVisionStrategy(func(cfg *vision.VisionStrategyConfig) {
    cfg.UsageConfig = usageConfig
    cfg.PreferVisionOnlyProviders = true  // Prioritize vision-only providers
})

// Use with CatalogizerStrategy
catStrat := catalogizer.NewCatalogizerStrategy(func(cfg *catalogizer.CatalogizerStrategyConfig) {
    cfg.UsageConfig = usageConfig
})
```

## Strategy Integration

### Vision Strategy

The Vision Strategy automatically:

1. **Excludes** providers marked as `UsageTypeLLMOnly`
2. **Prioritizes** providers marked as `UsageTypeVisionOnly` with a 10% score bonus
3. **Accepts** providers marked as `UsageTypeAny` with vision support

```go
strat := vision.NewVisionStrategy()

// Will return score=0 for LLM-only providers
score, _ := strat.Score(ctx, strategy.ModelInfo{
    Provider: "groq",
    SupportsVision: false,
})
// score.Overall == 0

// Will give bonus to vision-only providers
score, _ = strat.Score(ctx, strategy.ModelInfo{
    Provider: "google",
    SupportsVision: true,
})
// score gets +10% bonus for being vision-only designated
```

### Catalogizer Strategy

The Catalogizer Strategy automatically:

1. **Excludes** providers marked as `UsageTypeVisionOnly` from general LLM tasks
2. **Preserves** expensive vision providers for vision-specific work
3. **Accepts** all other providers based on their capabilities

```go
strat := catalogizer.NewCatalogizerStrategy()

// Will filter out vision-only providers for general tasks
models := []strategy.ModelInfo{
    {Provider: "google", ID: "gemini-flash"},
    {Provider: "anthropic", ID: "claude-sonnet"},
}
ranked, _ := strat.Rank(ctx, models)
// Gemini is excluded from general LLM ranking
```

## HelixQA Integration

HelixQA uses the Provider Usage Configuration to route tasks appropriately:

### Configuration File

```yaml
# helixqa-config.yaml
llm:
  provider: auto  # Let HelixQA choose based on task
  
provider_usage:
  gemini:
    usage: vision_only
    reason: "Expensive, excellent vision - reserve for screenshots"
  groq:
    usage: llm_only
    reason: "Fast, cheap - use for text tasks"
  anthropic:
    usage: any
    reason: "General purpose"
```

### Task-Based Routing

```go
// HelixQA automatically routes based on task type

// Vision task -> Uses Gemini (vision-only)
helixqa.Execute(ctx, VisionTask{
    Screenshot: img,
    Prompt: "Analyze this UI",
})

// Planning task -> Uses Anthropic or Groq (not Gemini)
helixqa.Execute(ctx, PlanningTask{
    Context: testContext,
    Prompt: "Plan test cases",
})
```

## Cost Optimization

### Before Provider Usage

```
All tasks used all providers:
- Screenshot analysis: Gemini ($$$)
- Test planning: Gemini ($$$)
- Code review: Gemini ($$$)

Cost: ~$0.50-1.00 per session
```

### After Provider Usage

```
Tasks routed to appropriate providers:
- Screenshot analysis: Gemini ($$$) - only when needed
- Test planning: Groq ($) - fast, cheap
- Code review: Anthropic ($$) - good balance

Cost: ~$0.10-0.30 per session (70% reduction)
```

## Provider Aliases

The system normalizes provider names for convenience:

| Input | Normalized |
|-------|------------|
| `google`, `gemini`, `vertex`, `vertexai` | `google` |
| `anthropic`, `claude` | `anthropic` |
| `openai`, `gpt`, `chatgpt` | `openai` |
| `ollama`, `local` | `ollama` |

## API Reference

### ProviderUsageConfig

```go
type ProviderUsageConfig struct {
    ProviderUsages map[string]ProviderUsageType
    DefaultUsage   ProviderUsageType
    StrictMode     bool
}

func (c *ProviderUsageConfig) GetUsage(provider string) ProviderUsageType
func (c *ProviderUsageConfig) CanUseForVision(provider string, supportsVision bool) bool
func (c *ProviderUsageConfig) CanUseForLLM(provider string) bool
func (c *ProviderUsageConfig) GetVisionProviders() []string
func (c *ProviderUsageConfig) GetLLMProviders() []string
```

### ProviderUsageRequirements

```go
type ProviderUsageRequirements struct {
    Requirements
    RequireVisionProvider bool
    RequireLLMProvider    bool
    UsageConfig          *ProviderUsageConfig
}
```

## Best Practices

1. **Reserve expensive providers for their strengths:**
   - Gemini: Vision tasks (excellent image understanding)
   - Groq: High-volume text (fast, cheap)
   - Anthropic: Complex reasoning (high quality)

2. **Use local providers when possible:**
   - Ollama: Both vision and LLM if you have GPU
   - No API costs, always available

3. **Monitor usage:**
   ```go
   // Log provider selection for cost tracking
   log.Printf("Selected %s for %s task", 
       model.Provider, 
       task.Type)
   ```

4. **Fine-tune per project:**
   ```go
   // Different projects may have different cost/quality needs
   if project.Budget == "low" {
       config.ProviderUsages["google"] = UsageTypeLLMOnly // Disable Gemini
   }
   ```

## Migration Guide

### From Legacy (No Usage Config)

**Before:**
```go
strat := vision.NewVisionStrategy()
// All providers considered for all tasks
```

**After:**
```go
strat := vision.NewVisionStrategy()
// Automatically respects default usage config
// Gemini vision-only, Groq excluded, etc.
```

No code changes needed! The default configuration is automatically applied.

### Customizing Behavior

**To make a provider vision-only:**
```go
config := strategy.DefaultProviderUsageConfig()
config.ProviderUsages["openai"] = strategy.UsageTypeVisionOnly
```

**To allow Gemini for all tasks:**
```go
config := strategy.DefaultProviderUsageConfig()
config.ProviderUsages["google"] = strategy.UsageTypeAny
```

**To disable a provider:**
```go
config := strategy.DefaultProviderUsageConfig()
delete(config.ProviderUsages, "google") // Remove from config
// Or set to LLMOnly and don't use LLM tasks
```

## Troubleshooting

### Issue: Vision-only provider not being selected

**Check:** Does the provider support vision?
```go
if !model.SupportsVision {
    // Provider must have SupportsVision=true
}
```

### Issue: Provider being used for wrong task type

**Check:** Verify usage configuration
```go
usage := config.GetUsage(provider)
fmt.Printf("Provider %s has usage type: %s\n", provider, usage)
```

### Issue: Need more control over selection

**Solution:** Use strict mode
```go
config.StrictMode = true
// Only explicitly configured providers will be used
```

## Changelog

### v1.0.0 (2026-04-08)
- Initial release
- Support for UsageTypeAny, UsageTypeVisionOnly, UsageTypeLLMOnly
- Default configuration with Gemini as vision-only
- Integration with VisionStrategy and CatalogizerStrategy
- Provider alias normalization
