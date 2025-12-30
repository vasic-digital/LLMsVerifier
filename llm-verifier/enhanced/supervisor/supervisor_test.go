package supervisor

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ==================== SupervisorConfig Tests ====================

func TestSupervisorConfigValidate(t *testing.T) {
	config := SupervisorConfig{
		MaxConcurrentJobs:   10,
		JobTimeout:          5 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
	}

	err := config.Validate()
	assert.NoError(t, err)
}

func TestSupervisorConfigValidateZeroMaxJobs(t *testing.T) {
	config := SupervisorConfig{
		MaxConcurrentJobs:   0,
		JobTimeout:          5 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
	}

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max concurrent jobs")
}

func TestSupervisorConfigValidateZeroTimeout(t *testing.T) {
	config := SupervisorConfig{
		MaxConcurrentJobs:   10,
		JobTimeout:          0,
		HealthCheckInterval: 30 * time.Second,
	}

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job timeout")
}

func TestSupervisorConfigValidateZeroHealthCheck(t *testing.T) {
	config := SupervisorConfig{
		MaxConcurrentJobs:   10,
		JobTimeout:          5 * time.Minute,
		HealthCheckInterval: 0,
	}

	err := config.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "health check interval")
}

// ==================== PluginManager Tests ====================

// MockPlugin for testing
type MockPlugin struct {
	name        string
	version     string
	description string
	capabilities []string
	initErr     error
	execErr     error
	execResult  interface{}
}

func (m *MockPlugin) Name() string                                         { return m.name }
func (m *MockPlugin) Version() string                                      { return m.version }
func (m *MockPlugin) Description() string                                  { return m.description }
func (m *MockPlugin) Initialize(config map[string]interface{}) error       { return m.initErr }
func (m *MockPlugin) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	if m.execErr != nil {
		return nil, m.execErr
	}
	if m.execResult != nil {
		return m.execResult, nil
	}
	return map[string]interface{}{"result": "success"}, nil
}
func (m *MockPlugin) Shutdown() error                                      { return nil }
func (m *MockPlugin) GetCapabilities() []string                            { return m.capabilities }

func TestNewPluginManager(t *testing.T) {
	pm := NewPluginManager(log.Default())
	assert.NotNil(t, pm)
	assert.NotNil(t, pm.system)
}

func TestPluginManagerRegisterPlugin(t *testing.T) {
	pm := NewPluginManager(log.Default())
	plugin := &MockPlugin{
		name:        "test_plugin",
		version:     "1.0.0",
		description: "Test plugin",
		capabilities: []string{"test"},
	}

	err := pm.RegisterPlugin(plugin)
	assert.NoError(t, err)
}

func TestPluginManagerRegisterDuplicatePlugin(t *testing.T) {
	pm := NewPluginManager(log.Default())
	plugin := &MockPlugin{
		name:        "test_plugin",
		version:     "1.0.0",
		description: "Test plugin",
		capabilities: []string{"test"},
	}

	pm.RegisterPlugin(plugin)
	err := pm.RegisterPlugin(plugin)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestPluginManagerExecutePlugin(t *testing.T) {
	pm := NewPluginManager(log.Default())
	plugin := &MockPlugin{
		name:        "test_plugin",
		version:     "1.0.0",
		description: "Test plugin",
		capabilities: []string{"test"},
	}

	pm.RegisterPlugin(plugin)

	ctx := context.Background()
	result, err := pm.ExecutePlugin(ctx, "test_plugin", "input")
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPluginManagerExecutePluginNotFound(t *testing.T) {
	pm := NewPluginManager(log.Default())

	ctx := context.Background()
	_, err := pm.ExecutePlugin(ctx, "non_existent", "input")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPluginManagerExecutePluginDisabled(t *testing.T) {
	pm := NewPluginManager(log.Default())
	plugin := &MockPlugin{
		name:        "test_plugin",
		version:     "1.0.0",
		description: "Test plugin",
		capabilities: []string{"test"},
	}

	pm.RegisterPlugin(plugin)
	pm.DisablePlugin("test_plugin")

	ctx := context.Background()
	_, err := pm.ExecutePlugin(ctx, "test_plugin", "input")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

func TestPluginManagerEnablePlugin(t *testing.T) {
	pm := NewPluginManager(log.Default())
	plugin := &MockPlugin{
		name:        "test_plugin",
		version:     "1.0.0",
		description: "Test plugin",
		capabilities: []string{"test"},
	}

	pm.RegisterPlugin(plugin)
	pm.DisablePlugin("test_plugin")
	err := pm.EnablePlugin("test_plugin")
	assert.NoError(t, err)
}

func TestPluginManagerEnablePluginNotFound(t *testing.T) {
	pm := NewPluginManager(log.Default())
	err := pm.EnablePlugin("non_existent")
	assert.Error(t, err)
}

func TestPluginManagerDisablePlugin(t *testing.T) {
	pm := NewPluginManager(log.Default())
	plugin := &MockPlugin{
		name:        "test_plugin",
		version:     "1.0.0",
		description: "Test plugin",
		capabilities: []string{"test"},
	}

	pm.RegisterPlugin(plugin)
	err := pm.DisablePlugin("test_plugin")
	assert.NoError(t, err)
}

func TestPluginManagerDisablePluginNotFound(t *testing.T) {
	pm := NewPluginManager(log.Default())
	err := pm.DisablePlugin("non_existent")
	assert.Error(t, err)
}

func TestPluginManagerListPlugins(t *testing.T) {
	pm := NewPluginManager(log.Default())
	pm.RegisterPlugin(&MockPlugin{name: "plugin_a", version: "1.0", description: "A", capabilities: []string{"a"}})
	pm.RegisterPlugin(&MockPlugin{name: "plugin_b", version: "1.0", description: "B", capabilities: []string{"b"}})

	plugins := pm.ListPlugins()
	assert.Len(t, plugins, 2)
}

// ==================== InMemoryCache Tests ====================

func TestNewInMemoryCache(t *testing.T) {
	cache := NewInMemoryCache(100)
	assert.NotNil(t, cache)
}

func TestInMemoryCacheSetAndGet(t *testing.T) {
	cache := NewInMemoryCache(100)

	err := cache.Set("key1", "value1", 5*time.Minute)
	assert.NoError(t, err)

	value, exists := cache.Get("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", value)
}

func TestInMemoryCacheGetNotFound(t *testing.T) {
	cache := NewInMemoryCache(100)

	value, exists := cache.Get("nonexistent")
	assert.False(t, exists)
	assert.Nil(t, value)
}

func TestInMemoryCacheGetExpired(t *testing.T) {
	cache := NewInMemoryCache(100)

	cache.Set("key1", "value1", 50*time.Millisecond)
	time.Sleep(100 * time.Millisecond)

	value, exists := cache.Get("key1")
	assert.False(t, exists)
	assert.Nil(t, value)
}

func TestInMemoryCacheDelete(t *testing.T) {
	cache := NewInMemoryCache(100)

	cache.Set("key1", "value1", 5*time.Minute)
	err := cache.Delete("key1")
	assert.NoError(t, err)

	_, exists := cache.Get("key1")
	assert.False(t, exists)
}

func TestInMemoryCacheClear(t *testing.T) {
	cache := NewInMemoryCache(100)

	cache.Set("key1", "value1", 5*time.Minute)
	cache.Set("key2", "value2", 5*time.Minute)

	err := cache.Clear()
	assert.NoError(t, err)

	_, exists1 := cache.Get("key1")
	_, exists2 := cache.Get("key2")
	assert.False(t, exists1)
	assert.False(t, exists2)
}

func TestInMemoryCacheClose(t *testing.T) {
	cache := NewInMemoryCache(100)
	err := cache.Close()
	assert.NoError(t, err)
}

func TestInMemoryCacheMaxSize(t *testing.T) {
	cache := NewInMemoryCache(2)

	cache.Set("key1", "value1", 5*time.Minute)
	cache.Set("key2", "value2", 5*time.Minute)
	cache.Set("key3", "value3", 5*time.Minute) // Should evict one

	// At least one key should be missing (LRU eviction)
	count := 0
	if _, exists := cache.Get("key1"); exists {
		count++
	}
	if _, exists := cache.Get("key2"); exists {
		count++
	}
	if _, exists := cache.Get("key3"); exists {
		count++
	}
	assert.LessOrEqual(t, count, 2)
}

// ==================== CacheManager Tests ====================

func TestNewCacheManager(t *testing.T) {
	backend := NewInMemoryCache(100)
	cm := NewCacheManager(backend, true)

	assert.NotNil(t, cm)
	assert.True(t, cm.IsEnabled())
}

func TestCacheManagerGet(t *testing.T) {
	backend := NewInMemoryCache(100)
	cm := NewCacheManager(backend, true)

	backend.Set("key1", "value1", 5*time.Minute)
	value, exists := cm.Get("key1")

	assert.True(t, exists)
	assert.Equal(t, "value1", value)
}

func TestCacheManagerGetDisabled(t *testing.T) {
	backend := NewInMemoryCache(100)
	cm := NewCacheManager(backend, false)

	backend.Set("key1", "value1", 5*time.Minute)
	value, exists := cm.Get("key1")

	assert.False(t, exists)
	assert.Nil(t, value)
}

func TestCacheManagerSet(t *testing.T) {
	backend := NewInMemoryCache(100)
	cm := NewCacheManager(backend, true)

	err := cm.Set("key1", "value1", 5*time.Minute)
	assert.NoError(t, err)

	value, _ := backend.Get("key1")
	assert.Equal(t, "value1", value)
}

func TestCacheManagerSetDisabled(t *testing.T) {
	backend := NewInMemoryCache(100)
	cm := NewCacheManager(backend, false)

	err := cm.Set("key1", "value1", 5*time.Minute)
	assert.NoError(t, err) // Should not error even if disabled
}

func TestCacheManagerDelete(t *testing.T) {
	backend := NewInMemoryCache(100)
	cm := NewCacheManager(backend, true)

	backend.Set("key1", "value1", 5*time.Minute)
	err := cm.Delete("key1")
	assert.NoError(t, err)
}

func TestCacheManagerDeleteDisabled(t *testing.T) {
	backend := NewInMemoryCache(100)
	cm := NewCacheManager(backend, false)

	err := cm.Delete("key1")
	assert.NoError(t, err)
}

func TestCacheManagerClear(t *testing.T) {
	backend := NewInMemoryCache(100)
	cm := NewCacheManager(backend, true)

	err := cm.Clear()
	assert.NoError(t, err)
}

func TestCacheManagerClearDisabled(t *testing.T) {
	backend := NewInMemoryCache(100)
	cm := NewCacheManager(backend, false)

	err := cm.Clear()
	assert.NoError(t, err)
}

func TestCacheManagerEnableDisable(t *testing.T) {
	backend := NewInMemoryCache(100)
	cm := NewCacheManager(backend, false)

	cm.Enable()
	assert.True(t, cm.IsEnabled())

	cm.Disable()
	assert.False(t, cm.IsEnabled())
}

// ==================== Built-in Plugins Tests ====================

func TestSentimentAnalysisPlugin(t *testing.T) {
	plugin := &SentimentAnalysisPlugin{}

	assert.Equal(t, "sentiment_analysis", plugin.Name())
	assert.Equal(t, "1.0.0", plugin.Version())
	assert.Contains(t, plugin.Description(), "sentiment")

	err := plugin.Initialize(nil)
	assert.NoError(t, err)

	ctx := context.Background()
	result, err := plugin.Execute(ctx, "This is great and awesome!")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, "positive", resultMap["sentiment"])

	err = plugin.Shutdown()
	assert.NoError(t, err)

	capabilities := plugin.GetCapabilities()
	assert.Contains(t, capabilities, "sentiment_analysis")
}

func TestSentimentAnalysisPluginNegative(t *testing.T) {
	plugin := &SentimentAnalysisPlugin{}

	ctx := context.Background()
	result, err := plugin.Execute(ctx, "This is terrible and awful")
	assert.NoError(t, err)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, "negative", resultMap["sentiment"])
}

func TestSentimentAnalysisPluginInvalidInput(t *testing.T) {
	plugin := &SentimentAnalysisPlugin{}

	ctx := context.Background()
	_, err := plugin.Execute(ctx, 12345) // Not a string
	assert.Error(t, err)
}

func TestCodeReviewPlugin(t *testing.T) {
	plugin := &CodeReviewPlugin{}

	assert.Equal(t, "code_review", plugin.Name())
	assert.Equal(t, "1.0.0", plugin.Version())
	assert.Contains(t, plugin.Description(), "code review")

	err := plugin.Initialize(nil)
	assert.NoError(t, err)

	ctx := context.Background()
	result, err := plugin.Execute(ctx, "func main() { fmt.Println(\"Hello\") }")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, "go", resultMap["language"])

	err = plugin.Shutdown()
	assert.NoError(t, err)

	capabilities := plugin.GetCapabilities()
	assert.Contains(t, capabilities, "code_review")
}

func TestCodeReviewPluginWithIssues(t *testing.T) {
	plugin := &CodeReviewPlugin{}

	ctx := context.Background()
	// Code with TODO and long line
	code := "// TODO: fix this\npassword = 'secret123' # " + string(make([]byte, 150))
	result, err := plugin.Execute(ctx, code)
	assert.NoError(t, err)

	resultMap := result.(map[string]interface{})
	issues := resultMap["issues"].([]map[string]interface{})
	assert.NotEmpty(t, issues)
}

func TestCodeReviewPluginInvalidInput(t *testing.T) {
	plugin := &CodeReviewPlugin{}

	ctx := context.Background()
	_, err := plugin.Execute(ctx, 12345) // Not a string
	assert.Error(t, err)
}

func TestPerformanceAnalysisPlugin(t *testing.T) {
	plugin := &PerformanceAnalysisPlugin{}

	assert.Equal(t, "performance_analysis", plugin.Name())
	assert.Equal(t, "1.0.0", plugin.Version())
	assert.Contains(t, plugin.Description(), "performance")

	err := plugin.Initialize(nil)
	assert.NoError(t, err)

	ctx := context.Background()
	metrics := map[string]interface{}{
		"response_time_avg": 500.0,
		"error_rate":        0.01,
	}
	result, err := plugin.Execute(ctx, metrics)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	err = plugin.Shutdown()
	assert.NoError(t, err)

	capabilities := plugin.GetCapabilities()
	assert.Contains(t, capabilities, "performance_analysis")
}

func TestPerformanceAnalysisPluginHighResponseTime(t *testing.T) {
	plugin := &PerformanceAnalysisPlugin{}

	ctx := context.Background()
	metrics := map[string]interface{}{
		"response_time_avg": 3000.0, // High
	}
	result, err := plugin.Execute(ctx, metrics)
	assert.NoError(t, err)

	resultMap := result.(map[string]interface{})
	analysis := resultMap["analysis"].(map[string]interface{})
	assert.Equal(t, "poor", analysis["overall_health"])
}

func TestPerformanceAnalysisPluginInvalidInput(t *testing.T) {
	plugin := &PerformanceAnalysisPlugin{}

	ctx := context.Background()
	_, err := plugin.Execute(ctx, "invalid") // Not a map
	assert.Error(t, err)
}

// ==================== Helper Functions Tests ====================

func TestAnalyzeSentiment(t *testing.T) {
	tests := []struct {
		text     string
		expected string
	}{
		{"This is great and excellent", "positive"},
		{"This is terrible and awful", "negative"},
		{"This is just normal", "neutral"},
		{"Good bad good bad", "neutral"}, // Mixed
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			score := analyzeSentiment(tt.text)
			label := getSentimentLabel(score)
			assert.Equal(t, tt.expected, label)
		})
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"func main() { }", "go"},
		{"package main", "go"},
		{"def hello():", "python"},
		{"import os", "python"},
		{"function test() { }", "javascript"},
		{"const x = 1", "javascript"},
		{"public class Main { }", "java"},
		{"class Test { }", "java"},
		{"unknown code", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, detectLanguage(tt.code))
		})
	}
}

func TestCalculateCodeQuality(t *testing.T) {
	// No issues = 100%
	assert.Equal(t, 100.0, calculateCodeQuality(nil))
	assert.Equal(t, 100.0, calculateCodeQuality([]map[string]interface{}{}))

	// With issues
	issues := []map[string]interface{}{
		{"severity": "low"},
		{"severity": "high"},
	}
	quality := calculateCodeQuality(issues)
	assert.Less(t, quality, 100.0)
}

func TestAnalyzePerformance(t *testing.T) {
	// Good metrics
	goodMetrics := map[string]interface{}{
		"response_time_avg": 500.0,
		"error_rate":        0.001,
	}
	analysis := analyzePerformance(goodMetrics)
	assert.Equal(t, "good", analysis["overall_health"])

	// Poor metrics - high response time
	poorMetrics := map[string]interface{}{
		"response_time_avg": 3000.0,
	}
	analysis = analyzePerformance(poorMetrics)
	assert.Equal(t, "poor", analysis["overall_health"])

	// Fair metrics - elevated error rate
	fairMetrics := map[string]interface{}{
		"error_rate": 0.03,
	}
	analysis = analyzePerformance(fairMetrics)
	assert.Equal(t, "fair", analysis["overall_health"])
}

func TestIdentifyBottlenecks(t *testing.T) {
	metrics := map[string]interface{}{
		"cpu_usage":           90.0,
		"memory_usage":        90.0,
		"db_connections_used": 95.0,
		"db_connections_max":  100.0,
	}

	bottlenecks := identifyBottlenecks(metrics)
	assert.Len(t, bottlenecks, 3) // CPU, memory, and database

	// Verify components detected
	components := make([]string, 0)
	for _, b := range bottlenecks {
		components = append(components, b["component"].(string))
	}
	assert.Contains(t, components, "cpu")
	assert.Contains(t, components, "memory")
	assert.Contains(t, components, "database")
}

func TestIdentifyBottlenecksNoIssues(t *testing.T) {
	metrics := map[string]interface{}{
		"cpu_usage":           50.0,
		"memory_usage":        50.0,
		"db_connections_used": 10.0,
		"db_connections_max":  100.0,
	}

	bottlenecks := identifyBottlenecks(metrics)
	assert.Empty(t, bottlenecks)
}

func TestGeneratePerformanceRecommendations(t *testing.T) {
	// Good health
	goodAnalysis := map[string]interface{}{
		"overall_health": "good",
		"issues":         []string{},
	}
	recs := generatePerformanceRecommendations(goodAnalysis)
	assert.Contains(t, recs, "System performance is optimal")

	// Response time issues
	timeAnalysis := map[string]interface{}{
		"overall_health": "fair",
		"issues":         []string{"High response time"},
	}
	recs = generatePerformanceRecommendations(timeAnalysis)
	assert.NotEmpty(t, recs)

	// Error rate issues
	errorAnalysis := map[string]interface{}{
		"overall_health": "poor",
		"issues":         []string{"High error rate"},
	}
	recs = generatePerformanceRecommendations(errorAnalysis)
	assert.NotEmpty(t, recs)
}

// ==================== Interface Implementation Tests ====================

func TestPluginInterfaceCompliance(t *testing.T) {
	var _ Plugin = (*SentimentAnalysisPlugin)(nil)
	var _ Plugin = (*CodeReviewPlugin)(nil)
	var _ Plugin = (*PerformanceAnalysisPlugin)(nil)
}

func TestCacheBackendInterfaceCompliance(t *testing.T) {
	var _ CacheBackend = (*InMemoryCache)(nil)
}
