# 🚀 LLM Verifier Scoring System - Comprehensive Examples

This document provides extensive examples covering all features of the scoring system.

## 📋 Table of Contents

1. [Basic Examples](#basic-examples)
2. [Configuration Examples](#configuration-examples)
3. [Batch Processing Examples](#batch-processing-examples)
4. [Model Naming Examples](#model-naming-examples)
5. [Advanced Examples](#advanced-examples)
6. [Integration Examples](#integration-examples)
7. [Performance Examples](#performance-examples)
8. [Error Handling Examples](#error-handling-examples)

## Basic Examples

### 1. Simple Score Calculation

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "llm-verifier/database"
    "llm-verifier/logging"
    "llm-verifier/scoring"
)

func basicScoreCalculation() {
    // Setup
    db, _ := database.New(":memory:")
    defer db.Close()
    
    provider := &database.Provider{
        Name:     "OpenAI",
        Endpoint: "https://api.openai.com/v1",
        IsActive: true,
    }
    db.CreateProvider(provider)
    
    model := &database.Model{
        ProviderID:          provider.ID,
        ModelID:             "gpt-4",
        Name:                "GPT-4",
        ParameterCount:      int64Ptr(175000000000),
        ContextWindowTokens: intPtr(128000),
        ReleaseDate:         timePtr(time.Date(2023, 3, 14, 0, 0, 0, 0, time.UTC)),
        IsMultimodal:        true,
        SupportsVision:      true,
        SupportsReasoning:   true,
        VerificationStatus:  "verified",
        ResponsivenessScore: 8.5,
        CodeCapabilityScore: 9.0,
    }
    db.CreateModel(model)
    
    // Create scoring engine
    logger := &logging.Logger{}
    client, _ := scoring.NewModelsDevClient(scoring.DefaultClientConfig(), logger)
    engine := scoring.NewScoringEngine(db, client, logger)
    
    // Calculate score
    ctx := context.Background()
    config := scoring.DefaultScoringConfig()
    
    score, err := engine.CalculateComprehensiveScore(ctx, "gpt-4", config)
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Printf("=== Basic Score Calculation ===\n")
    fmt.Printf("Model: %s\n", score.ModelName)
    fmt.Printf("Overall Score: %.1f %s\n", score.OverallScore, score.ScoreSuffix)
    fmt.Printf("Last Calculated: %s\n", score.LastCalculated.Format(time.RFC3339))
}
```

### 2. Component Analysis

```go
func componentAnalysis() {
    // Setup scoring engine (reuse from basic example)
    
    score, _ := engine.CalculateComprehensiveScore(ctx, "gpt-4", config)
    
    fmt.Printf("\n=== Component Analysis ===\n")
    fmt.Printf("Speed Score: %.1f/10\n", score.Components.SpeedScore)
    fmt.Printf("Efficiency Score: %.1f/10\n", score.Components.EfficiencyScore)
    fmt.Printf("Cost Score: %.1f/10\n", score.Components.CostScore)
    fmt.Printf("Capability Score: %.1f/10\n", score.Components.CapabilityScore)
    fmt.Printf("Recency Score: %.1f/10\n", score.Components.RecencyScore)
    
    // Identify strengths and weaknesses
    components := []struct {
        name  string
        score float64
    }{
        {"Speed", score.Components.SpeedScore},
        {"Efficiency", score.Components.EfficiencyScore},
        {"Cost", score.Components.CostScore},
        {"Capability", score.Components.CapabilityScore},
        {"Recency", score.Components.RecencyScore},
    }
    
    fmt.Printf("\n=== Strengths & Weaknesses ===\n")
    for _, comp := range components {
        if comp.score >= 8.0 {
            fmt.Printf("✅ %s: Excellent (%.1f)\n", comp.name, comp.score)
        } else if comp.score >= 6.0 {
            fmt.Printf("🟡 %s: Good (%.1f)\n", comp.name, comp.score)
        } else if comp.score >= 4.0 {
            fmt.Printf("🟠 %s: Average (%.1f)\n", comp.name, comp.score)
        } else {
            fmt.Printf("🔴 %s: Needs Improvement (%.1f)\n", comp.name, comp.score)
        }
    }
}
```

## Configuration Examples

### 1. Speed-Focused Configuration

```go
func speedFocusedScoring() {
    fmt.Printf("\n=== Speed-Focused Configuration ===\n")
    
    speedConfig := scoring.ScoringConfig{
        ConfigName: "speed-focused",
        Weights: scoring.ScoreWeights{
            ResponseSpeed:     0.6,  // 60% weight on speed
            ModelEfficiency:   0.1,
            CostEffectiveness: 0.1,
            Capability:        0.1,
            Recency:           0.1,
        },
        Thresholds: scoring.ScoreThresholds{
            MinScore: 0.0,
            MaxScore: 10.0,
        },
        Enabled: true,
    }
    
    score, _ := engine.CalculateComprehensiveScore(ctx, "gpt-4", speedConfig)
    fmt.Printf("Speed-focused score: %.1f %s\n", score.OverallScore, score.ScoreSuffix)
    fmt.Printf("Speed component: %.1f/10 (weighted: %.1f)\n", 
        score.Components.SpeedScore, 
        score.Components.SpeedScore * 0.6)
}
```

### 2. Cost-Focused Configuration

```go
func costFocusedScoring() {
    fmt.Printf("\n=== Cost-Focused Configuration ===\n")
    
    costConfig := scoring.ScoringConfig{
        ConfigName: "cost-focused",
        Weights: scoring.ScoreWeights{
            ResponseSpeed:     0.1,
            ModelEfficiency:   0.1,
            CostEffectiveness: 0.6,  // 60% weight on cost
            Capability:        0.1,
            Recency:           0.1,
        },
        Thresholds: scoring.ScoreThresholds{
            MinScore: 0.0,
            MaxScore: 10.0,
        },
        Enabled: true,
    }
    
    score, _ := engine.CalculateComprehensiveScore(ctx, "gpt-4", costConfig)
    fmt.Printf("Cost-focused score: %.1f %s\n", score.OverallScore, score.ScoreSuffix)
    fmt.Printf("Cost component: %.1f/10 (weighted: %.1f)\n", 
        score.Components.CostScore, 
        score.Components.CostScore * 0.6)
}
```

### 3. Balanced Configuration

```go
func balancedScoring() {
    fmt.Printf("\n=== Balanced Configuration ===\n")
    
    balancedConfig := scoring.ScoringConfig{
        ConfigName: "balanced",
        Weights: scoring.ScoreWeights{
            ResponseSpeed:     0.2,
            ModelEfficiency:   0.2,
            CostEffectiveness: 0.2,
            Capability:        0.2,
            Recency:           0.2,
        },
        Thresholds: scoring.ScoreThresholds{
            MinScore: 0.0,
            MaxScore: 10.0,
        },
        Enabled: true,
    }
    
    score, _ := engine.CalculateComprehensiveScore(ctx, "gpt-4", balancedConfig)
    fmt.Printf("Balanced score: %.1f %s\n", score.OverallScore, score.ScoreSuffix)
    fmt.Printf("All components equally weighted at 20%%\n")
}
```

### 4. Dynamic Configuration Switching

```go
func dynamicConfiguration() {
    fmt.Printf("\n=== Dynamic Configuration Switching ===\n")
    
    configs := []struct {
        name   string
        config scoring.ScoringConfig
    }{
        {
            name: "Speed Focused",
            config: scoring.ScoringConfig{
                Weights: scoring.ScoreWeights{
                    ResponseSpeed: 0.6, ModelEfficiency: 0.1, CostEffectiveness: 0.1, Capability: 0.1, Recency: 0.1,
                },
            },
        },
        {
            name: "Cost Focused", 
            config: scoring.ScoringConfig{
                Weights: scoring.ScoreWeights{
                    ResponseSpeed: 0.1, ModelEfficiency: 0.1, CostEffectiveness: 0.6, Capability: 0.1, Recency: 0.1,
                },
            },
        },
        {
            name: "Capability Focused",
            config: scoring.ScoringConfig{
                Weights: scoring.ScoreWeights{
                    ResponseSpeed: 0.1, ModelEfficiency: 0.1, CostEffectiveness: 0.1, Capability: 0.6, Recency: 0.1,
                },
            },
        },
    }
    
    for _, cfg := range configs {
        score, _ := engine.CalculateComprehensiveScore(ctx, "gpt-4", cfg.config)
        fmt.Printf("%s: %.1f %s\n", cfg.name, score.OverallScore, score.ScoreSuffix)
    }
}
```

## Batch Processing Examples

### 1. Basic Batch Scoring

```go
func basicBatchScoring() {
    fmt.Printf("\n=== Basic Batch Scoring ===\n")
    
    modelIDs := []string{
        "gpt-4",
        "claude-3-sonnet", 
        "llama-2-70b",
        "gemini-pro",
    }
    
    scores, err := engine.CalculateBatchScores(ctx, modelIDs, &config.Weights)
    if err != nil {
        log.Printf("Batch scoring error: %v", err)
        return
    }
    
    fmt.Printf("Processed %d models\n", len(scores))
    for i, score := range scores {
        fmt.Printf("%d. %s: %.1f %s\n", 
            i+1, score.ModelName, score.OverallScore, score.ScoreSuffix)
    }
}
```

### 2. Batch with Progress Tracking

```go
func batchWithProgress() {
    fmt.Printf("\n=== Batch Scoring with Progress ===\n")
    
    modelIDs := []string{"model-1", "model-2", "model-3", "model-4", "model-5"}
    total := len(modelIDs)
    
    fmt.Printf("Processing %d models...\n", total)
    
    scores, err := engine.CalculateBatchScores(ctx, modelIDs, &config.Weights)
    if err != nil {
        log.Printf("Batch error: %v", err)
        return
    }
    
    fmt.Printf("\nResults:\n")
    for i, score := range scores {
        progress := float64(i+1) / float64(total) * 100
        fmt.Printf("[%3.0f%%] %s: %.1f %s\n", 
            progress, score.ModelName, score.OverallScore, score.ScoreSuffix)
    }
}
```

### 3. Batch with Error Handling

```go
func batchWithErrorHandling() {
    fmt.Printf("\n=== Batch Scoring with Error Handling ===\n")
    
    modelIDs := []string{"valid-model-1", "invalid-model", "valid-model-2"}
    
    scores, err := engine.CalculateBatchScores(ctx, modelIDs, &config.Weights)
    if err != nil {
        log.Printf("Batch processing error: %v", err)
        
        // Handle partial failure
        if len(scores) > 0 {
            fmt.Printf("Partial success: processed %d out of %d models\n", len(scores), len(modelIDs))
            for _, score := range scores {
                fmt.Printf("✅ %s: %.1f %s\n", score.ModelName, score.OverallScore, score.ScoreSuffix)
            }
        }
        return
    }
    
    fmt.Printf("✅ All %d models processed successfully\n", len(scores))
}
```

## Model Naming Examples

### 1. Basic Naming Operations

```go
func basicNamingOperations() {
    fmt.Printf("\n=== Basic Naming Operations ===\n")
    
    naming := scoring.NewModelNaming()
    
    // Add score suffix
    originalName := "GPT-4"
    score := 8.5
    updatedName := naming.AddScoreSuffix(originalName, score)
    fmt.Printf("'%s' → '%s' (score: %.1f)\n", originalName, updatedName, score)
    
    // Update existing suffix
    nameWithScore := "Claude-3 (SC:7.2)"
    newScore := 8.1
    updatedName = naming.AddScoreSuffix(nameWithScore, newScore)
    fmt.Printf("'%s' → '%s' (score: %.1f)\n", nameWithScore, updatedName, newScore)
    
    // Extract score from name
    testNames := []string{
        "GPT-4 (SC:8.5)",
        "Llama-2 (SC:6.9)",
        "Model Without Score",
        "Invalid (SC:abc)",
    }
    
    fmt.Printf("\nScore Extraction:\n")
    for _, name := range testNames {
        extractedScore, found := naming.ExtractScoreFromName(name)
        if found {
            fmt.Printf("'%s' → Score: %.1f\n", name, extractedScore)
        } else {
            fmt.Printf("'%s' → No score found\n", name)
        }
    }
}
```

### 2. Batch Naming Updates

```go
func batchNamingUpdates() {
    fmt.Printf("\n=== Batch Naming Updates ===\n")
    
    naming := scoring.NewModelNaming()
    
    modelScores := map[string]float64{
        "GPT-4":           8.5,
        "Claude-3":        7.8,
        "Llama-2":         6.9,
        "Gemini-Pro":      7.2,
        "Mistral-Large":   7.5,
    }
    
    fmt.Printf("Updating %d model names...\n", len(modelScores))
    
    results := naming.BatchUpdateModelNames(modelScores)
    
    fmt.Printf("\nResults:\n")
    for original, updated := range results {
        fmt.Printf("'%s' → '%s'\n", original, updated)
    }
}
```

### 3. Score Suffix Validation

```go
func scoreSuffixValidation() {
    fmt.Printf("\n=== Score Suffix Validation ===\n")
    
    naming := scoring.NewModelNaming()
    
    testSuffixes := []string{
        "(SC:8.5)",        // Valid
        " (SC:7.2) ",      // Valid with spaces
        "SC:8.5",          // Invalid - missing parentheses
        "(SC:invalid)",    // Invalid - non-numeric score
        "(SC:8.5.2)",      // Invalid - too many decimals
        "(SC:15.0)",       // Invalid - score too high
        "(SC:-1.0)",       // Invalid - negative score
    }
    
    fmt.Printf("Validation Results:\n")
    for _, suffix := range testSuffixes {
        isValid := naming.ValidateScoreSuffix(suffix)
        status := "❌ Invalid"
        if isValid {
            status = "✅ Valid"
        }
        fmt.Printf("'%s' → %s\n", suffix, status)
    }
}
```

## Advanced Examples

### 1. Score Ranking System

```go
func scoreRankingSystem() {
    fmt.Printf("\n=== Score Ranking System ===\n")
    
    // Create multiple models with different scores
    models := []*database.Model{
        {ProviderID: provider.ID, ModelID: "model-a", Name: "Model A", ParameterCount: int64Ptr(1000000000), ReleaseDate: timePtr(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))},
        {ProviderID: provider.ID, ModelID: "model-b", Name: "Model B", ParameterCount: int64Ptr(5000000000), ReleaseDate: timePtr(time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC))},
        {ProviderID: provider.ID, ModelID: "model-c", Name: "Model C", ParameterCount: int64Ptr(2000000000), ReleaseDate: timePtr(time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC))},
    }
    
    scores := make([]*scoring.ComprehensiveScore, len(models))
    for i, model := range models {
        db.CreateModel(model)
        score, _ := engine.CalculateComprehensiveScore(ctx, model.ModelID, config)
        scores[i] = score
    }
    
    // Sort by overall score (descending)
    sort.Slice(scores, func(i, j int) bool {
        return scores[i].OverallScore > scores[j].OverallScore
    })
    
    fmt.Printf("Model Rankings:\n")
    for i, score := range scores {
        fmt.Printf("%d. %s: %.1f %s\n", 
            i+1, score.ModelName, score.OverallScore, score.ScoreSuffix)
    }
}
```

### 2. Score Analytics

```go
func scoreAnalytics() {
    fmt.Printf("\n=== Score Analytics ===\n")
    
    // Calculate multiple scores for analytics
    modelIDs := []string{"gpt-4", "claude-3", "llama-2", "gemini-pro", "mistral-large"}
    scores := make([]*scoring.ComprehensiveScore, len(modelIDs))
    
    for i, modelID := range modelIDs {
        score, _ := engine.CalculateComprehensiveScore(ctx, modelID, config)
        scores[i] = score
    }
    
    // Calculate statistics
    var totalScore float64
    var minScore, maxScore float64 = 10.0, 0.0
    
    for _, score := range scores {
        totalScore += score.OverallScore
        if score.OverallScore < minScore {
            minScore = score.OverallScore
        }
        if score.OverallScore > maxScore {
            maxScore = score.OverallScore
        }
    }
    
    avgScore := totalScore / float64(len(scores))
    
    fmt.Printf("Score Statistics:\n")
    fmt.Printf("Average Score: %.2f\n", avgScore)
    fmt.Printf("Minimum Score: %.1f\n", minScore)
    fmt.Printf("Maximum Score: %.1f\n", maxScore)
    fmt.Printf("Score Range: %.1f\n", maxScore-minScore)
    
    // Component statistics
    fmt.Printf("\nComponent Statistics:\n")
    components := []string{"Speed", "Efficiency", "Cost", "Capability", "Recency"}
    componentScores := []float64{
        0, 0, 0, 0, 0, // Will accumulate
    }
    
    for _, score := range scores {
        componentScores[0] += score.Components.SpeedScore
        componentScores[1] += score.Components.EfficiencyScore
        componentScores[2] += score.Components.CostScore
        componentScores[3] += score.Components.CapabilityScore
        componentScores[4] += score.Components.RecencyScore
    }
    
    for i, comp := range components {
        avg := componentScores[i] / float64(len(scores))
        fmt.Printf("%s: Average = %.2f/10\n", comp, avg)
    }
}
```

### 3. Custom Scoring Logic

```go
func customScoringLogic() {
    fmt.Printf("\n=== Custom Scoring Logic ===\n")
    
    type EnhancedScorer struct {
        base *scoring.ScoringEngine
    }
    
    func (es *EnhancedScorer) CalculateWithBonus(ctx context.Context, modelID string) (*scoring.ComprehensiveScore, error) {
        // Get base score
        baseScore, err := es.base.CalculateComprehensiveScore(ctx, modelID, scoring.DefaultScoringConfig())
        if err != nil {
            return nil, err
        }
        
        // Apply custom bonuses
        bonus := 0.0
        
        // Bonus for high capability
        if baseScore.Components.CapabilityScore > 8.5 {
            bonus += 0.3
        }
        
        // Bonus for recent models
        if baseScore.Components.RecencyScore > 7.0 {
            bonus += 0.2
        }
        
        // Bonus for balanced performance
        avgComponent := (baseScore.Components.SpeedScore + 
                        baseScore.Components.EfficiencyScore + 
                        baseScore.Components.CostScore + 
                        baseScore.Components.CapabilityScore + 
                        baseScore.Components.RecencyScore) / 5.0
        
        if avgComponent > 7.5 && baseScore.OverallScore > 7.0 {
            bonus += 0.1
        }
        
        // Apply bonus (max 10.0)
        newScore := baseScore.OverallScore + bonus
        if newScore > 10.0 {
            newScore = 10.0
        }
        
        // Update score
        baseScore.OverallScore = newScore
        baseScore.ScoreSuffix = fmt.Sprintf("(SC:%.1f)", newScore)
        
        return baseScore, nil
    }
    
    // Use enhanced scorer
    enhanced := &EnhancedScorer{base: engine}
    
    score, _ := enhanced.CalculateWithBonus(ctx, "gpt-4")
    fmt.Printf("Enhanced Score: %.1f %s (includes custom bonuses)\n", 
        score.OverallScore, score.ScoreSuffix)
}
```

## Integration Examples

### 1. Web Service Integration

```go
package main

import (
    "encoding/json"
    "net/http"
    
    "github.com/gorilla/mux"
    "llm-verifier/scoring"
)

type ScoreRequest struct {
    ModelID     string                  `json:"model_id"`
    Config      *scoring.ScoringConfig  `json:"config,omitempty"`
}

type ScoreResponse struct {
    Success bool                           `json:"success"`
    Data    *scoring.ComprehensiveScore    `json:"data"`
    Error   string                         `json:"error,omitempty"`
}

func scoreHandler(w http.ResponseWriter, r *http.Request) {
    var req ScoreRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Use global scoring engine
    config := scoring.DefaultScoringConfig()
    if req.Config != nil {
        config = *req.Config
    }
    
    score, err := globalEngine.CalculateComprehensiveScore(r.Context(), req.ModelID, config)
    if err != nil {
        json.NewEncoder(w).Encode(ScoreResponse{
            Success: false,
            Error:   err.Error(),
        })
        return
    }
    
    json.NewEncoder(w).Encode(ScoreResponse{
        Success: true,
        Data:    score,
    })
}

func main() {
    router := mux.NewRouter()
    router.HandleFunc("/api/score", scoreHandler).Methods("POST")
    
    http.ListenAndServe(":8080", router)
}
```

### 2. CLI Tool Integration

```go
package main

import (
    "bufio"
    "fmt"
    "os"
    "strings"
    
    "llm-verifier/scoring"
)

func main() {
    fmt.Println("🚀 LLM Verifier CLI Tool")
    fmt.Println("========================")
    
    reader := bufio.NewReader(os.Stdin)
    
    for {
        fmt.Print("\nEnter model ID (or 'quit' to exit): ")
        modelID, _ := reader.ReadString('\n')
        modelID = strings.TrimSpace(modelID)
        
        if modelID == "quit" {
            break
        }
        
        score, err := globalEngine.CalculateComprehensiveScore(context.Background(), modelID, scoring.DefaultScoringConfig())
        if err != nil {
            fmt.Printf("❌ Error: %v\n", err)
            continue
        }
        
        fmt.Printf("\n📊 Results:\n")
        fmt.Printf("Model: %s\n", score.ModelName)
        fmt.Printf("Score: %.1f %s\n", score.OverallScore, score.ScoreSuffix)
        fmt.Printf("Components:\n")
        fmt.Printf("  Speed: %.1f\n", score.Components.SpeedScore)
        fmt.Printf("  Efficiency: %.1f\n", score.Components.EfficiencyScore)
        fmt.Printf("  Cost: %.1f\n", score.Components.CostScore)
        fmt.Printf("  Capability: %.1f\n", score.Components.CapabilityScore)
        fmt.Printf("  Recency: %.1f\n", score.Components.RecencyScore)
    }
}
```

### 3. Database Integration

```go
func databaseIntegration() {
    // Setup database connection
    db, err := database.New("./data/llm-verifier.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
    
    // Create tables if needed
    err = db.InitializeScoringSchema()
    if err != nil {
        log.Fatal(err)
    }
    
    // Create scoring system with database
    engine := scoring.NewScoringEngine(db, client, logger)
    
    // Store score in database
    score, _ := engine.CalculateComprehensiveScore(ctx, "gpt-4", config)
    
    // Use database extensions to store score
    dbExt := scoring.NewScoringDatabaseExtensions(db)
    err = dbExt.CreateModelScore(score)
    if err != nil {
        log.Printf("Failed to store score: %v", err)
    }
    
    // Retrieve stored score
    storedScore, err := dbExt.GetLatestModelScore(modelID)
    if err != nil {
        log.Printf("Failed to retrieve score: %v", err)
    }
    
    fmt.Printf("Stored score: %.1f\n", storedScore.Score)
}
```

## Performance Examples

### 1. Concurrent Score Calculation

```go
func concurrentScoring() {
    fmt.Printf("\n=== Concurrent Scoring ===\n")
    
    modelIDs := []string{"model-1", "model-2", "model-3", "model-4", "model-5"}
    
    // Create channels for results
    results := make(chan *scoring.ComprehensiveScore, len(modelIDs))
    errors := make(chan error, len(modelIDs))
    
    // Launch concurrent calculations
    for _, modelID := range modelIDs {
        go func(id string) {
            score, err := engine.CalculateComprehensiveScore(ctx, id, config)
            if err != nil {
                errors <- err
                return
            }
            results <- score
        }(modelID)
    }
    
    // Collect results
    var scores []*scoring.ComprehensiveScore
    var errs []error
    
    for i := 0; i < len(modelIDs); i++ {
        select {
        case score := <-results:
            scores = append(scores, score)
        case err := <-errors:
            errs = append(errs, err)
        }
    }
    
    fmt.Printf("Processed %d models concurrently\n", len(scores))
    fmt.Printf("Errors: %d\n", len(errs))
    
    // Display results
    for i, score := range scores {
        fmt.Printf("%d. %s: %.1f %s\n", 
            i+1, score.ModelName, score.OverallScore, score.ScoreSuffix)
    }
}
```

### 2. Performance Benchmarking

```go
func performanceBenchmarking() {
    fmt.Printf("\n=== Performance Benchmarking ===\n")
    
    iterations := []int{1, 10, 100, 1000}
    
    for _, iter := range iterations {
        start := time.Now()
        
        for i := 0; i < iter; i++ {
            _, _ = engine.CalculateComprehensiveScore(ctx, "gpt-4", config)
        }
        
        duration := time.Since(start)
        avgTime := duration / time.Duration(iter)
        
        fmt.Printf("%d iterations: Total = %v, Average = %v per calculation\n", 
            iter, duration, avgTime)
    }
}
```

### 3. Memory Optimization

```go
func memoryOptimizedScoring() {
    fmt.Printf("\n=== Memory Optimized Scoring ===\n")
    
    // Use object pool for scores
    scorePool := sync.Pool{
        New: func() interface{} {
            return &scoring.ComprehensiveScore{}
        },
    }
    
    // Process large batch with memory pooling
    for i := 0; i < 1000; i++ {
        // Get score from pool
        score := scorePool.Get().(*scoring.ComprehensiveScore)
        
        // Calculate score (reuse object)
        calculatedScore, _ := engine.CalculateComprehensiveScore(ctx, fmt.Sprintf("model-%d", i), config)
        
        // Copy data to pooled object
        *score = *calculatedScore
        
        // Use score...
        _ = score
        
        // Return to pool
        scorePool.Put(score)
    }
    
    fmt.Printf("Processed 1000 models with memory pooling\n")
}
```

## Error Handling Examples

### 1. Comprehensive Error Handling

```go
func comprehensiveErrorHandling() {
    fmt.Printf("\n=== Comprehensive Error Handling ===\n")
    
    testCases := []struct {
        name      string
        modelID   string
        expectErr bool
    }{
        {"Valid Model", "gpt-4", false},
        {"Invalid Model", "invalid-model-id", true},
        {"Empty Model ID", "", true},
        {"Special Characters", "model@#$%", true},
    }
    
    for _, tc := range testCases {
        fmt.Printf("\nTesting: %s\n", tc.name)
        
        score, err := engine.CalculateComprehensiveScore(ctx, tc.modelID, config)
        
        if err != nil {
            if tc.expectErr {
                fmt.Printf("✅ Expected error: %v\n", err)
                
                // Handle specific error types
                switch err.Error() {
                case "model not found":
                    fmt.Printf("  → Model '%s' not found in database\n", tc.modelID)
                case "invalid model ID":
                    fmt.Printf("  → Model ID '%s' contains invalid characters\n", tc.modelID)
                default:
                    fmt.Printf("  → Unexpected error: %v\n", err)
                }
            } else {
                fmt.Printf("❌ Unexpected error: %v\n", err)
            }
        } else {
            if tc.expectErr {
                fmt.Printf("❌ Expected error but got success\n")
            } else {
                fmt.Printf("✅ Success: %.1f %s\n", score.OverallScore, score.ScoreSuffix)
            }
        }
    }
}
```

### 2. Retry Logic

```go
func retryLogic() {
    fmt.Printf("\n=== Retry Logic ===\n")
    
    maxRetries := 3
    retryDelay := 100 * time.Millisecond
    
    for attempt := 1; attempt <= maxRetries; attempt++ {
        score, err := engine.CalculateComprehensiveScore(ctx, "problematic-model", config)
        
        if err == nil {
            fmt.Printf("✅ Success on attempt %d: %.1f %s\n", 
                attempt, score.OverallScore, score.ScoreSuffix)
            break
        }
        
        fmt.Printf("❌ Attempt %d failed: %v\n", attempt, err)
        
        if attempt < maxRetries {
            fmt.Printf("🔄 Retrying in %v...\n", retryDelay)
            time.Sleep(retryDelay)
            retryDelay *= 2 // Exponential backoff
        } else {
            fmt.Printf("❌ All %d attempts failed\n", maxRetries)
        }
    }
}
```

### 3. Fallback Mechanisms

```go
func fallbackMechanisms() {
    fmt.Printf("\n=== Fallback Mechanisms ===\n")
    
    // Primary: Full scoring
    score, err := engine.CalculateComprehensiveScore(ctx, "model-with-full-data", config)
    if err != nil {
        fmt.Printf("Primary scoring failed: %v\n", err)
        
        // Fallback 1: Simplified scoring
        fmt.Printf("🔄 Trying simplified scoring...\n")
        simpleConfig := scoring.ScoringConfig{
            Weights: scoring.ScoreWeights{
                ResponseSpeed: 0.5,
                Capability:    0.5,
            },
        }
        
        score, err = engine.CalculateComprehensiveScore(ctx, "model-with-full-data", simpleConfig)
        if err != nil {
            fmt.Printf("Simplified scoring failed: %v\n", err)
            
            // Fallback 2: Default score
            fmt.Printf("🔄 Using default score...\n")
            defaultScore := &scoring.ComprehensiveScore{
                ModelID:      "model-with-full-data",
                ModelName:    "Unknown Model",
                OverallScore: 5.0,
                ScoreSuffix:  "(SC:5.0)",
                Components: scoring.ScoreComponents{
                    SpeedScore:      5.0,
                    EfficiencyScore: 5.0,
                    CostScore:       5.0,
                    CapabilityScore: 5.0,
                    RecencyScore:    5.0,
                },
            }
            score = defaultScore
        }
    }
    
    fmt.Printf("Final result: %.1f %s\n", score.OverallScore, score.ScoreSuffix)
}
```

---

## 📚 Complete Example Application

Here's a complete example application that demonstrates all features:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "llm-verifier/database"
    "llm-verifier/logging"
    "llm-verifier/scoring"
)

func main() {
    fmt.Println("🚀 LLM Verifier Scoring System - Complete Demo")
    fmt.Println("==============================================")
    
    // Run all examples
    basicScoreCalculation()
    componentAnalysis()
    speedFocusedScoring()
    costFocusedScoring()
    balancedScoring()
    dynamicConfiguration()
    basicBatchScoring()
    basicNamingOperations()
    scoreRankingSystem()
    scoreAnalytics()
    customScoringLogic()
    
    fmt.Println("\n✅ All examples completed successfully!")
}

// Helper functions (same as defined in examples above)
func int64Ptr(i int64) *int64 { return &i }
func intPtr(i int) *int { return &i }
func timePtr(t time.Time) *time.Time { return &t }
```

---

## 🎯 Next Steps

1. **Run the Examples**: Try all examples in your environment
2. **Customize for Your Needs**: Modify examples for your specific use case
3. **Production Integration**: See [Integration Examples](#integration-examples)
4. **Performance Optimization**: Check [Performance Examples](#performance-examples)
5. **Advanced Features**: Explore [Advanced Examples](#advanced-examples)

## 🤝 Contributing Examples

Have a great example? Contribute it!

1. Fork the repository
2. Create your example in `/examples/scoring/`
3. Add comprehensive documentation
4. Submit a pull request

---

**🎉 You're now an expert in LLM Verifier Scoring System examples!**

*Next: [Advanced Usage](../guides/ADVANCED.md) →*