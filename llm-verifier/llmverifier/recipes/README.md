# Recipes — Composable Strategy Building Blocks

The `recipes` package provides pre-built `RecipeStep` implementations for use with the `StrategyBuilder`. Each recipe adds a `VerificationTest` that evaluates a specific LLM capability.

## Available Recipes

| Recipe | Category | Weight | Required | Purpose |
|--------|----------|--------|----------|---------|
| `VisionTest()` | vision | 1.0 | Yes | Tests screenshot analysis capability |
| `InstructionFollowingTest()` | instruction | 0.9 | No | Tests action precision and command following |
| `ContextWindowTest(minTokens)` | feature | 0.8 | No | Validates minimum context window size |
| `StreamingReliabilityTest()` | stability | 0.85 | No | Tests streaming response consistency |
| `LongSessionStabilityTest(duration)` | stability | 0.7 | No | Tests multi-hour session stability |

## Usage

```go
import "llm-verifier/llmverifier/recipes"

strategy, err := llmverifier.NewStrategyBuilder("my-strategy").
    WithWeights(llmverifier.WeightConfig{
        VisionCapability:     0.25,
        InstructionFollowing: 0.15,
        Reliability:          0.20,
        Responsiveness:       0.15,
        FeatureRichness:      0.15,
        CodeCapability:       0.10,
    }).
    WithRecipe(recipes.VisionTest()).
    WithRecipe(recipes.InstructionFollowingTest()).
    WithRecipe(recipes.ContextWindowTest(100000)).
    WithRecipe(recipes.StreamingReliabilityTest()).
    WithRecipe(recipes.LongSessionStabilityTest(30 * time.Minute)).
    RequireCapability("vision").
    MinScore(0.6).
    Build()
```

## Writing Custom Recipes

Implement the `RecipeStep` interface:

```go
type MyRecipe struct{}

func (r *MyRecipe) Apply(builder *llmverifier.StrategyBuilder) *llmverifier.StrategyBuilder {
    return builder.WithTest(&myTest{})
}

type myTest struct{}

func (t *myTest) ID() string                        { return "my-test" }
func (t *myTest) Name() string                      { return "My Custom Test" }
func (t *myTest) Category() llmverifier.TestCategory { return llmverifier.TestCategoryVision }
func (t *myTest) Weight() float64                    { return 1.0 }
func (t *myTest) Required() bool                     { return false }
func (t *myTest) Run(ctx context.Context, client *llmverifier.LLMClient) (llmverifier.TestResult, error) {
    // Your test logic here
    return llmverifier.TestResult{Passed: true, Score: 1.0}, nil
}
```
