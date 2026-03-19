# Video Course: LLMsVerifier Strategy Pattern

## Episode 1: Strategy Pattern Overview (10 min)

**Script outline:**
1. Problem: hardcoded scoring weights don't fit all use cases
2. Solution: ScoringStrategy interface with pluggable implementations
3. Demo: DefaultStrategy vs custom QA strategy — show different weight outputs
4. Architecture diagram: ScoringStrategy → StrategyBuilder → RecipeStep hierarchy
5. Backward compatibility: existing Verify()/CalculateScores() unchanged

## Episode 2: Building Custom Strategies with StrategyBuilder (15 min)

**Script outline:**
1. NewStrategyBuilder() fluent API walkthrough
2. WeightConfig: 6 dimensions, must sum to 1.0, Validate()
3. Thresholds: MinScore, MaxLatency, MinContextWindow
4. RequireCapability: mapping to ModelInfo fields
5. Build(): validation and error handling
6. Live coding: build a strategy from scratch

## Episode 3: RecipeBuilder Walkthrough (12 min)

**Script outline:**
1. RecipeStep interface: Apply(builder) pattern
2. Built-in recipes: VisionTest, InstructionFollowingTest, ContextWindowTest, StreamingReliabilityTest, LongSessionStabilityTest
3. Composing multiple recipes in a single strategy
4. Recipe test categories and weights
5. Demo: build HelixQA strategy with all 5 recipes

## Episode 4: Writing Custom VerificationTests (15 min)

**Script outline:**
1. VerificationTest interface: ID, Name, Category, Run, Weight, Required
2. LLMClient usage in Run(): sending prompts, parsing responses
3. TestResult: Passed, Score, Details — structured output
4. Required vs optional tests: fail-fast behavior
5. Live coding: write a custom navigation command test
6. Testing your tests: mock LLMClient patterns

## Episode 5: Integrating with External Tools (10 min)

**Script outline:**
1. VerifyWithStrategy: full pipeline (filter → verify → custom tests → score)
2. CalculateScoresWithStrategy: scoring existing results
3. StrategyScore.ToPerformanceScore(): backward compatibility conversion
4. Integration with HelixQA: QAStrategy injection
5. Integration with CI/CD: strategy-based model selection
6. Best practices and common patterns
