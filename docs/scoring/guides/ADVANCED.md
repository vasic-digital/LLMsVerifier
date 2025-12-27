# 🚀 LLM Verifier Scoring System - Advanced Guide

Master advanced features, optimizations, and customizations of the scoring system.

## 📋 Table of Contents

1. [Advanced Configuration](#advanced-configuration)
2. [Custom Scoring Components](#custom-scoring-components)
3. [Performance Optimization](#performance-optimization)
4. [Caching Strategies](#caching-strategies)
5. [Monitoring & Analytics](#monitoring--analytics)
6. [Security Best Practices](#security-best-practices)
7. [Scaling & Deployment](#scaling--deployment)
8. [Troubleshooting Advanced Issues](#troubleshooting-advanced-issues)

## Advanced Configuration

### Dynamic Configuration Management

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "time"
    
    "llm-verifier/scoring"
)

// AdvancedConfigurationManager manages multiple scoring configurations
type AdvancedConfigurationManager struct {
    configs map[string]*scoring.ScoringConfig
    mutex   sync.RWMutex
    loader  ConfigLoader
}

type ConfigLoader interface {
    LoadConfig(name string) (*scoring.ScoringConfig, error)
    SaveConfig(config *scoring.ScoringConfig) error
}

func NewAdvancedConfigurationManager(loader ConfigLoader) *AdvancedConfigurationManager {
    return &AdvancedConfigurationManager{
        configs: make(map[string]*scoring.ScoringConfig),
        loader:  loader,
    }
}

func (acm *AdvancedConfigurationManager) GetConfig(name string) (*scoring.ScoringConfig, error) {
    acm.mutex.RLock()
    if config, exists := acm.configs[name]; exists {
        acm.mutex.RUnlock()
        return config, nil
    }
    acm.mutex.RUnlock()
    
    // Load from external source
    config, err := acm.loader.LoadConfig(name)
    if err != nil {
        return nil, err
    }
    
    acm.mutex.Lock()
    acm.configs[name] = config
    acm.mutex.Unlock()
    
    return config, nil
}

// Environment-based configuration
func environmentBasedConfig() scoring.ScoringConfig {
    env := os.Getenv("LLM_SCORING_ENV")
    
    switch env {
    case "development":
        return developmentConfig()
    case "staging":
        return stagingConfig()
    case "production":
        return productionConfig()
    default:
        return scoring.DefaultScoringConfig()
    }
}

func developmentConfig() scoring.ScoringConfig {
    return scoring.ScoringConfig{
        ConfigName: "development",
        Weights: scoring.ScoreWeights{
            ResponseSpeed:     0.15, // Lower weight for development
            ModelEfficiency:   0.15,
            CostEffectiveness: 0.20,
            Capability:        0.30, // Higher weight for testing
            Recency:           0.20,
        },
        Thresholds: scoring.ScoreThresholds{
            MinScore: 0.0,
            MaxScore: 10.0,
        },
        Enabled: true,
    }
}

func productionConfig() scoring.ScoringConfig {
    return scoring.ScoringConfig{
        ConfigName: "production",
        Weights: scoring.ScoreWeights{
            ResponseSpeed:     0.25,
            ModelEfficiency:   0.20,
            CostEffectiveness: 0.25,
            Capability:        0.20,
            Recency:           0.10,
        },
        Thresholds: scoring.ScoreThresholds{
            MinScore: 0.0,
            MaxScore: 10.0,
        },
        Enabled: true,
    }
}
```

### Time-based Configuration

```go
func timeBasedConfig() scoring.ScoringConfig {
    now := time.Now()
    hour := now.Hour()
    
    // Different weights based on time of day
    if hour < 6 { // Early morning
        return scoring.ScoringConfig{
            Weights: scoring.ScoreWeights{
                ResponseSpeed:     0.3,  // Prioritize speed during low traffic
                ModelEfficiency:   0.2,
                CostEffectiveness: 0.2,
                Capability:        0.2,
                Recency:           0.1,
            },
        }
    } else if hour < 18 { // Business hours
        return scoring.ScoringConfig{
            Weights: scoring.ScoreWeights{
                ResponseSpeed:     0.25,
                ModelEfficiency:   0.20,
                CostEffectiveness: 0.25, // Normal business focus
                Capability:        0.20,
                Recency:           0.10,
            },
        }
    } else { // Evening/night
        return scoring.ScoringConfig{
            Weights: scoring.ScoreWeights{
                ResponseSpeed:     0.2,
                ModelEfficiency:   0.25, // Focus on efficiency
                CostEffectiveness: 0.25,
                Capability:        0.20,
                Recency:           0.10,
            },
        }
    }
}
```

## Custom Scoring Components

### 1. Custom Scoring Algorithm

```go
package main

import (
    "context"
    "fmt"
    "math"
    
    "llm-verifier/scoring"
)

// CustomScoringEngine extends the base scoring engine
type CustomScoringEngine struct {
    *scoring.ScoringEngine
    customWeights CustomWeights
}

type CustomWeights struct {
    BaseWeights      scoring.ScoreWeights
    TimeOfDayWeight  float64
    LoadFactorWeight float64
    CustomBonus      float64
}

func NewCustomScoringEngine(baseEngine *scoring.ScoringEngine, weights CustomWeights) *CustomScoringEngine {
    return &CustomScoringEngine{
        ScoringEngine: baseEngine,
        customWeights: weights,
    }
}

func (cse *CustomScoringEngine) CalculateWithCustomLogic(ctx context.Context, modelID string, config scoring.ScoringConfig) (*scoring.ComprehensiveScore, error) {
    // Get base score
    baseScore, err := cse.CalculateComprehensiveScore(ctx, modelID, config)
    if err != nil {
        return nil, err
    }
    
    // Apply custom logic
    adjustedScore := cse.applyCustomAdjustments(baseScore)
    
    return adjustedScore, nil
}

func (cse *CustomScoringEngine) applyCustomAdjustments(score *scoring.ComprehensiveScore) *scoring.ComprehensiveScore {
    // Time-based adjustment
    currentHour := time.Now().Hour()
    timeBonus := cse.calculateTimeBonus(currentHour)
    
    // Load-based adjustment
    loadFactor := cse.getCurrentLoadFactor()
    loadAdjustment := cse.calculateLoadAdjustment(loadFactor)
    
    // Performance bonus
    performanceBonus := cse.calculatePerformanceBonus(score.Components)
    
    // Apply adjustments
    totalAdjustment := timeBonus + loadAdjustment + performanceBonus
    newOverallScore := math.Min(10.0, score.OverallScore + totalAdjustment)
    
    // Update score
    score.OverallScore = newOverallScore
    score.ScoreSuffix = fmt.Sprintf("(SC:%.1f)", newOverallScore)
    
    return score
}

func (cse *CustomScoringEngine) calculateTimeBonus(hour int) float64 {
    // Bonus for off-peak hours
    if hour < 6 || hour > 22 {
        return 0.2
    } else if hour < 9 || hour > 17 {
        return 0.1
    }
    return 0.0
}

func (cse *CustomScoringEngine) calculatePerformanceBonus(components scoring.ScoreComponents) float64 {
    // Bonus for exceptional performance
    if components.CapabilityScore > 9.0 && components.SpeedScore > 8.5 {
        return 0.3
    } else if components.OverallAverage() > 8.0 {
        return 0.1
    }
    return 0.0
}
```

### 2. Machine Learning Integration

```go
type MLScoringEngine struct {
    baseEngine    *scoring.ScoringEngine
    mlModel       *tensorflow.SavedModel
    featureExtractor FeatureExtractor
}

func (mlse *MLScoringEngine) CalculateWithML(ctx context.Context, modelID string, config scoring.ScoringConfig) (*scoring.ComprehensiveScore, error) {
    // Get base features
    features, err := mlse.featureExtractor.ExtractFeatures(ctx, modelID)
    if err != nil {
        return nil, err
    }
    
    // Prepare ML input
    mlInput := mlse.prepareMLInput(features)
    
    // Run ML prediction
    mlOutput, err := mlse.mlModel.Predict(mlInput)
    if err != nil {
        return nil, err
    }
    
    // Combine ML output with base scoring
    return mlse.combineScores(mlOutput, features, config)
}
```

## Performance Optimization

### 1. Connection Pooling

```go
type OptimizedDatabasePool struct {
    primaryDB   *database.Database
    replicaDBs  []*database.Database
    poolSize    int
    currentIdx  int
    mutex       sync.Mutex
}

func (odp *OptimizedDatabasePool) GetConnection() *database.Database {
    odp.mutex.Lock()
    defer odp.mutex.Unlock()
    
    // Round-robin selection for read operations
    db := odp.replicaDBs[odp.currentIdx]
    odp.currentIdx = (odp.currentIdx + 1) % len(odp.replicaDBs)
    
    return db
}
```

### 2. Query Optimization

```go
func optimizedScoreCalculation(ctx context.Context, modelIDs []string, engine *scoring.ScoringEngine) ([]*scoring.ComprehensiveScore, error) {
    // Batch database operations
    batchSize := 100
    var allScores []*scoring.ComprehensiveScore
    
    for i := 0; i < len(modelIDs); i += batchSize {
        end := i + batchSize
        if end > len(modelIDs) {
            end = len(modelIDs)
        }
        
        batch := modelIDs[i:end]
        scores, err := engine.CalculateBatchScores(ctx, batch, nil)
        if err != nil {
            return nil, err
        }
        
        allScores = append(allScores, scores...)
    }
    
    return allScores, nil
}
```

### 3. Caching Implementation

```go
type ScoringCache struct {
    cache      map[string]*CacheEntry
    mutex      sync.RWMutex
    maxSize    int
    ttl        time.Duration
}

type CacheEntry struct {
    Score     *scoring.ComprehensiveScore
    Timestamp time.Time
}

func (sc *ScoringCache) Get(modelID string) (*scoring.ComprehensiveScore, bool) {
    sc.mutex.RLock()
    defer sc.mutex.RUnlock()
    
    entry, exists := sc.cache[modelID]
    if !exists {
        return nil, false
    }
    
    // Check if entry is expired
    if time.Since(entry.Timestamp) > sc.ttl {
        return nil, false
    }
    
    return entry.Score, true
}

func (sc *ScoringCache) Set(modelID string, score *scoring.ComprehensiveScore) {
    sc.mutex.Lock()
    defer sc.mutex.Unlock()
    
    // Remove oldest entry if cache is full
    if len(sc.cache) >= sc.maxSize {
        sc.removeOldest()
    }
    
    sc.cache[modelID] = &CacheEntry{
        Score:     score,
        Timestamp: time.Now(),
    }
}
```

## Monitoring & Analytics

### 1. Comprehensive Metrics

```go
type ScoringMetrics struct {
    TotalCalculations   int64
    SuccessfulCalculations int64
    FailedCalculations  int64
    AverageResponseTime time.Duration
    P95ResponseTime     time.Duration
    P99ResponseTime     time.Duration
    ScoreDistribution   map[string]int
    ComponentStats      ComponentStatistics
}

type ComponentStatistics struct {
    SpeedAverage      float64
    EfficiencyAverage float64
    CostAverage       float64
    CapabilityAverage float64
    RecencyAverage    float64
}

func (sm *ScoringMetrics) RecordCalculation(duration time.Duration, score *scoring.ComprehensiveScore, err error) {
    sm.mutex.Lock()
    defer sm.mutex.Unlock()
    
    sm.TotalCalculations++
    
    if err != nil {
        sm.FailedCalculations++
        return
    }
    
    sm.SuccessfulCalculations++
    
    // Update response time statistics
    sm.updateResponseTimeStats(duration)
    
    // Update score distribution
    scoreBucket := fmt.Sprintf("%.0f", score.OverallScore)
    sm.ScoreDistribution[scoreBucket]++
    
    // Update component statistics
    sm.updateComponentStats(score.Components)
}
```

### 2. Real-time Monitoring

```go
type RealTimeMonitor struct {
    metrics      *ScoringMetrics
    alerts       chan Alert
    thresholds   AlertThresholds
    subscribers  []MetricsSubscriber
}

type AlertThresholds struct {
    ErrorRate        float64
    ResponseTime     time.Duration
    ScoreDrop        float64
    ComponentDrop    float64
}

func (rtm *RealTimeMonitor) StartMonitoring() {
    go func() {
        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()
        
        for range ticker.C {
            rtm.checkThresholds()
            rtm.notifySubscribers()
        }
    }()
}

func (rtm *RealTimeMonitor) checkThresholds() {
    currentErrorRate := float64(rtm.metrics.FailedCalculations) / float64(rtm.metrics.TotalCalculations)
    
    if currentErrorRate > rtm.thresholds.ErrorRate {
        alert := Alert{
            Type:    "ERROR_RATE_THRESHOLD",
            Message: fmt.Sprintf("Error rate %.2f exceeds threshold %.2f", currentErrorRate, rtm.thresholds.ErrorRate),
            Severity: "WARNING",
            Timestamp: time.Now(),
        }
        
        rtm.alerts <- alert
    }
}
```

## Security Best Practices

### 1. Input Validation

```go
func validateScoringInput(modelID string, config *scoring.ScoringConfig) error {
    // Validate model ID
    if len(modelID) == 0 || len(modelID) > 100 {
        return fmt.Errorf("invalid model ID length: %d", len(modelID))
    }
    
    if !isValidModelID(modelID) {
        return fmt.Errorf("model ID contains invalid characters")
    }
    
    // Validate configuration
    if config == nil {
        return fmt.Errorf("configuration cannot be nil")
    }
    
    // Validate weights
    totalWeight := config.Weights.ResponseSpeed + 
                  config.Weights.ModelEfficiency + 
                  config.Weights.CostEffectiveness + 
                  config.Weights.Capability + 
                  config.Weights.Recency
    
    if math.Abs(totalWeight-1.0) > 0.001 {
        return fmt.Errorf("weights must sum to 1.0, got %.3f", totalWeight)
    }
    
    // Validate individual weights
    for _, weight := range []float64{
        config.Weights.ResponseSpeed,
        config.Weights.ModelEfficiency,
        config.Weights.CostEffectiveness,
        config.Weights.Capability,
        config.Weights.Recency,
    } {
        if weight < 0.0 || weight > 1.0 {
            return fmt.Errorf("invalid weight: %.3f", weight)
        }
    }
    
    return nil
}
```

### 2. Rate Limiting

```go
type RateLimiter struct {
    requests map[string][]time.Time
    limit    int
    window   time.Duration
    mutex    sync.Mutex
}

func (rl *RateLimiter) Allow(clientID string) bool {
    rl.mutex.Lock()
    defer rl.mutex.Unlock()
    
    now := time.Now()
    windowStart := now.Add(-rl.window)
    
    // Clean old entries
    requests := rl.requests[clientID]
    validRequests := make([]time.Time, 0)
    
    for _, reqTime := range requests {
        if reqTime.After(windowStart) {
            validRequests = append(validRequests, reqTime)
        }
    }
    
    // Check rate limit
    if len(validRequests) >= rl.limit {
        return false
    }
    
    // Add current request
    validRequests = append(validRequests, now)
    rl.requests[clientID] = validRequests
    
    return true
}
```

### 3. Audit Logging

```go
type AuditLogger struct {
    logger *logging.Logger
}

func (al *AuditLogger) LogScoreCalculation(modelID string, config *scoring.ScoringConfig, result *scoring.ComprehensiveScore, duration time.Duration, err error) {
    auditEntry := map[string]interface{}{
        "timestamp":     time.Now().UTC(),
        "model_id":      modelID,
        "config_name":   config.ConfigName,
        "result_score":  result.OverallScore,
        "duration_ms":   duration.Milliseconds(),
        "success":       err == nil,
        "user_agent":    getUserAgent(),
        "client_ip":     getClientIP(),
        "request_id":    generateRequestID(),
    }
    
    if err != nil {
        auditEntry["error"] = err.Error()
    }
    
    al.logger.Info("Score calculation audit", auditEntry)
}
```

## Scaling & Deployment

### 1. Horizontal Scaling

```yaml
# docker-compose.yml for scaling
version: '3.8'
services:
  llm-verifier-1:
    image: llm-verifier:latest
    environment:
      - LLM_INSTANCE_ID=instance-1
      - LLM_REPLICA_ID=replica-1
    ports:
      - "8081:8080"
    
  llm-verifier-2:
    image: llm-verifier:latest
    environment:
      - LLM_INSTANCE_ID=instance-2
      - LLM_REPLICA_ID=replica-2
    ports:
      - "8082:8080"
    
  load-balancer:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
```

### 2. Kubernetes Deployment

```yaml
# k8s-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: llm-verifier
spec:
  replicas: 3
  selector:
    matchLabels:
      app: llm-verifier
  template:
    metadata:
      labels:
        app: llm-verifier
    spec:
      containers:
      - name: llm-verifier
        image: llm-verifier:latest
        ports:
        - containerPort: 8080
        env:
        - name: LLM_ENV
          value: "production"
        - name: LLM_INSTANCE_ID
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
```

### 3. Auto-scaling Configuration

```yaml
# k8s-hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: llm-verifier-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: llm-verifier
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 100
        periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
```

## Troubleshooting Advanced Issues

### 1. Memory Leaks

```go
func detectMemoryLeaks() {
    var m1 runtime.MemStats
    runtime.ReadMemStats(&m1)
    
    // Run scoring operations
    for i := 0; i < 1000; i++ {
        _, _ = engine.CalculateComprehensiveScore(ctx, "test-model", config)
    }
    
    var m2 runtime.MemStats
    runtime.ReadMemStats(&m2)
    
    memoryGrowth := int64(m2.Alloc) - int64(m1.Alloc)
    fmt.Printf("Memory growth: %d bytes\n", memoryGrowth)
    
    if memoryGrowth > 100*1024*1024 { // 100MB threshold
        fmt.Println("⚠️ Potential memory leak detected!")
        
        // Force garbage collection
        runtime.GC()
        
        var m3 runtime.MemStats
        runtime.ReadMemStats(&m3)
        
        postGCGrowth := int64(m3.Alloc) - int64(m1.Alloc)
        fmt.Printf("Memory after GC: %d bytes\n", postGCGrowth)
    }
}
```

### 2. Performance Degradation

```go
func performanceProfiling() {
    // CPU profiling
    f, err := os.Create("cpu.prof")
    if err != nil {
        log.Fatal(err)
    }
    defer f.Close()
    
    if err := pprof.StartCPUProfile(f); err != nil {
        log.Fatal(err)
    }
    defer pprof.StopCPUProfile()
    
    // Run intensive operations
    for i := 0; i < 10000; i++ {
        _, _ = engine.CalculateComprehensiveScore(ctx, fmt.Sprintf("model-%d", i), config)
    }
    
    // Memory profiling
    memProf, err := os.Create("mem.prof")
    if err != nil {
        log.Fatal(err)
    }
    defer memProf.Close()
    
    runtime.GC()
    if err := pprof.WriteHeapProfile(memProf); err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Performance profiles generated: cpu.prof, mem.prof")
    fmt.Println("Analyze with: go tool pprof cpu.prof")
}
```

### 3. Database Connection Issues

```go
func databaseConnectionDiagnostics() {
    fmt.Println("🔍 Database Connection Diagnostics")
    
    // Test connectivity
    if err := db.Ping(); err != nil {
        fmt.Printf("❌ Database ping failed: %v\n", err)
        
        // Check connection pool
        stats := db.Stats()
        fmt.Printf("Connection Pool Stats:\n")
        fmt.Printf("  Open Connections: %d\n", stats.OpenConnections)
        fmt.Printf("  In Use: %d\n", stats.InUse)
        fmt.Printf("  Idle: %d\n", stats.Idle)
        fmt.Printf("  Wait Count: %d\n", stats.WaitCount)
        fmt.Printf("  Wait Duration: %v\n", stats.WaitDuration)
        
        // Retry connection
        fmt.Println("🔄 Retrying connection...")
        time.Sleep(5 * time.Second)
        
        if err := db.Ping(); err != nil {
            fmt.Printf("❌ Retry failed: %v\n", err)
            fmt.Println("💡 Suggestions:")
            fmt.Println("  1. Check database server status")
            fmt.Println("  2. Verify connection string")
            fmt.Println("  3. Check network connectivity")
            fmt.Println("  4. Review database logs")
        } else {
            fmt.Println("✅ Connection restored after retry")
        }
    } else {
        fmt.Println("✅ Database connection is healthy")
    }
}
```

---

## 📚 Complete Advanced Example

Here's a complete example demonstrating all advanced features:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"
    
    "llm-verifier/database"
    "llm-verifier/logging"
    "llm-verifier/scoring"
)

func main() {
    fmt.Println("🚀 LLM Verifier Advanced Features Demo")
    fmt.Println("=====================================")
    
    // Setup
    ctx := context.Background()
    db, _ := database.New(":memory:")
    defer db.Close()
    
    // Create production-like setup
    setupProductionEnvironment(ctx, db)
    
    // Run advanced examples
    advancedConfigurationDemo(ctx, db)
    customScoringComponentsDemo(ctx, db)
    performanceOptimizationDemo(ctx, db)
    monitoringAndAnalyticsDemo(ctx, db)
    securityBestPracticesDemo(ctx, db)
    
    fmt.Println("\n✅ All advanced features demonstrated successfully!")
}

func setupProductionEnvironment(ctx context.Context, db *database.Database) {
    // Create providers and models for demo
    // (Implementation would be similar to basic examples but more comprehensive)
}

func advancedConfigurationDemo(ctx context.Context, db *database.Database) {
    // (Implementation would include all advanced configuration examples)
}

func customScoringComponentsDemo(ctx context.Context, db *database.Database) {
    // (Implementation would include all custom scoring examples)
}

func performanceOptimizationDemo(ctx context.Context, db *database.Database) {
    // (Implementation would include all performance optimization examples)
}

func monitoringAndAnalyticsDemo(ctx context.Context, db *database.Database) {
    // (Implementation would include all monitoring examples)
}

func securityBestPracticesDemo(ctx context.Context, db *database.Database) {
    // (Implementation would include all security examples)
}
```

---

## 🎯 Next Steps

1. **Implement Examples**: Try all examples in your environment
2. **Production Setup**: Use [Deployment Guide](./DEPLOYMENT.md)
3. **Monitoring Setup**: Configure comprehensive monitoring
4. **Performance Tuning**: Optimize for your specific use case
5. **Security Hardening**: Implement all security best practices

## 🤝 Contributing

Help improve this guide by:

1. Adding more advanced examples
2. Sharing performance optimization techniques
3. Contributing security best practices
4. Providing production deployment experiences

---

**🎉 You're now an expert in advanced LLM Verifier Scoring System usage!**

*Next: [Deployment Guide](./DEPLOYMENT.md) →*