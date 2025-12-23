package enhanced

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
		RetryAttempts:       3,
		RetryBackoff:        1 * time.Second,
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

func TestNewPluginManager(t *testing.T) {
	pm := NewPluginManager(nil)

	assert.NotNil(t, pm)
	assert.NotNil(t, pm.system)
	assert.NotNil(t, pm.system.plugins)
	assert.NotNil(t, pm.system.enabled)
}

// MockPlugin for testing
type MockPlugin struct {
	name        string
	version     string
	description string
	capabilities []string
}

func (m *MockPlugin) Name() string { return m.name }
func (m *MockPlugin) Version() string { return m.version }
func (m *MockPlugin) Description() string { return m.description }
func (m *MockPlugin) Initialize(config map[string]interface{}) error { return nil }
func (m *MockPlugin) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	return map[string]interface{}{"result": "success"}, nil
}
func (m *MockPlugin) Shutdown() error { return nil }
func (m *MockPlugin) GetCapabilities() []string { return m.capabilities }

func TestPluginManagerRegisterPlugin(t *testing.T) {
	pm := NewPluginManager(log.Default())
	plugin := &MockPlugin{
		name:        "test_plugin",
		version:     "1.0.0",
		description: "Test plugin for unit tests",
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
		description: "Test plugin for unit tests",
		capabilities: []string{"test"},
	}

	// Register once
	pm.RegisterPlugin(plugin)

	// Try to register again
	err := pm.RegisterPlugin(plugin)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestPluginManagerGetPlugin(t *testing.T) {
	pm := NewPluginManager(log.Default())
	plugin := &MockPlugin{
		name:        "test_plugin",
		version:     "1.0.0",
		description: "Test plugin for unit tests",
		capabilities: []string{"test"},
	}

	pm.RegisterPlugin(plugin)

	retrieved, exists := pm.GetPlugin("test_plugin")

	assert.True(t, exists)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "test_plugin", retrieved.Name())
}

func TestPluginManagerGetPluginNotFound(t *testing.T) {
	pm := NewPluginManager(log.Default())

	_, exists := pm.GetPlugin("non_existent")

	assert.False(t, exists)
}

func TestPluginManagerListPlugins(t *testing.T) {
	pm := NewPluginManager(log.Default())

	plugin1 := &MockPlugin{
		name:        "plugin_a",
		version:     "1.0.0",
		description: "Plugin A",
		capabilities: []string{"a"},
	}
	plugin2 := &MockPlugin{
		name:        "plugin_b",
		version:     "1.0.0",
		description: "Plugin B",
		capabilities: []string{"b"},
	}

	pm.RegisterPlugin(plugin1)
	pm.RegisterPlugin(plugin2)

	plugins := pm.ListPlugins()

	assert.Equal(t, 2, len(plugins))
}

func TestPluginManagerListPluginsSorted(t *testing.T) {
	pm := NewPluginManager(log.Default())

	plugin1 := &MockPlugin{
		name:        "plugin_b",
		version:     "1.0.0",
		description: "Plugin B",
		capabilities: []string{"b"},
	}
	plugin2 := &MockPlugin{
		name:        "plugin_a",
		version:     "1.0.0",
		description: "Plugin A",
		capabilities: []string{"a"},
	}

	pm.RegisterPlugin(plugin1)
	pm.RegisterPlugin(plugin2)

	plugins := pm.ListPlugins()

	// Should be sorted alphabetically
	assert.Equal(t, "plugin_a", plugins[0]["name"])
	assert.Equal(t, "plugin_b", plugins[1]["name"])
}

func TestPluginManagerEnablePlugin(t *testing.T) {
	pm := NewPluginManager(log.Default())
	plugin := &MockPlugin{
		name:        "test_plugin",
		version:     "1.0.0",
		description: "Test plugin for unit tests",
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
	assert.Contains(t, err.Error(), "not found")
}

func TestPluginManagerDisablePlugin(t *testing.T) {
	pm := NewPluginManager(log.Default())
	plugin := &MockPlugin{
		name:        "test_plugin",
		version:     "1.0.0",
		description: "Test plugin for unit tests",
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
	assert.Contains(t, err.Error(), "not found")
}

func TestPluginManagerExecutePlugin(t *testing.T) {
	pm := NewPluginManager(log.Default())
	plugin := &MockPlugin{
		name:        "test_plugin",
		version:     "1.0.0",
		description: "Test plugin for unit tests",
		capabilities: []string{"test"},
	}

	pm.RegisterPlugin(plugin)

	ctx := context.Background()
	result, err := pm.ExecutePlugin(ctx, "test_plugin", "test input")

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPluginManagerExecutePluginNotFound(t *testing.T) {
	pm := NewPluginManager(log.Default())

	ctx := context.Background()
	_, err := pm.ExecutePlugin(ctx, "non_existent", "test input")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPluginManagerExecutePluginDisabled(t *testing.T) {
	pm := NewPluginManager(log.Default())
	plugin := &MockPlugin{
		name:        "test_plugin",
		version:     "1.0.0",
		description: "Test plugin for unit tests",
		capabilities: []string{"test"},
	}

	pm.RegisterPlugin(plugin)
	pm.DisablePlugin("test_plugin")

	ctx := context.Background()
	_, err := pm.ExecutePlugin(ctx, "test_plugin", "test input")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "disabled")
}

// ==================== Cache Tests ====================

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

// ==================== CacheManager Tests ====================

func TestNewCacheManager(t *testing.T) {
	backend := NewInMemoryCache(100)
	cm := NewCacheManager(backend, true)

	assert.NotNil(t, cm)
	assert.NotNil(t, cm.backend)
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

	// Should not error even if disabled
	assert.NoError(t, err)
}

func TestCacheManagerDelete(t *testing.T) {
	backend := NewInMemoryCache(100)
	cm := NewCacheManager(backend, true)

	backend.Set("key1", "value1", 5*time.Minute)
	err := cm.Delete("key1")

	assert.NoError(t, err)
}

func TestCacheManagerClear(t *testing.T) {
	backend := NewInMemoryCache(100)
	cm := NewCacheManager(backend, true)

	err := cm.Clear()

	assert.NoError(t, err)
}

func TestCacheManagerEnable(t *testing.T) {
	backend := NewInMemoryCache(100)
	cm := NewCacheManager(backend, false)

	cm.Enable()

	assert.True(t, cm.IsEnabled())
}

func TestCacheManagerDisable(t *testing.T) {
	backend := NewInMemoryCache(100)
	cm := NewCacheManager(backend, true)

	cm.Disable()

	assert.False(t, cm.IsEnabled())
}

// ==================== Task Tests ====================

func TestTaskStruct(t *testing.T) {
	now := time.Now()
	task := &Task{
		ID:         "task_1",
		Type:       "test",
		Priority:   5,
		Data:       map[string]interface{}{"test": "data"},
		CreatedAt:  now,
		MaxRetries: 3,
		Status:     "pending",
	}

	assert.Equal(t, "task_1", task.ID)
	assert.Equal(t, "test", task.Type)
	assert.Equal(t, 5, task.Priority)
	assert.Equal(t, "pending", task.Status)
}

func TestTaskResultStruct(t *testing.T) {
	now := time.Now()
	result := &TaskResult{
		TaskID:      "task_1",
		WorkerID:    "worker_1",
		Success:     true,
		Data:        map[string]interface{}{"result": "data"},
		Result:      "completed",
		Duration:    100 * time.Millisecond,
		CompletedAt: now,
	}

	assert.Equal(t, "task_1", result.TaskID)
	assert.Equal(t, "worker_1", result.WorkerID)
	assert.True(t, result.Success)
}

// ==================== Worker Tests ====================

func TestWorkerStruct(t *testing.T) {
	now := time.Now()
	worker := &Worker{
		ID:            "worker_1",
		Type:          "test",
		Status:        "idle",
		TasksDone:     10,
		LastActive:    now,
		StartTime:     now,
		Capabilities:  []string{"test", "general"},
		Performance: map[string]interface{}{
			"tasks_completed": 10,
		},
	}

	assert.Equal(t, "worker_1", worker.ID)
	assert.Equal(t, "idle", worker.Status)
	assert.Equal(t, 10, worker.TasksDone)
}

// ==================== Simple Plugins Tests ====================

func TestSimpleSentimentPlugin(t *testing.T) {
	plugin := &SimpleSentimentPlugin{}

	assert.Equal(t, "sentiment_analysis", plugin.Name())
	assert.Equal(t, "1.0.0", plugin.Version())
	assert.Contains(t, plugin.Description(), "sentiment")

	err := plugin.Initialize(nil)
	assert.NoError(t, err)

	ctx := context.Background()
	result, err := plugin.Execute(ctx, "This is great")

	assert.NoError(t, err)
	assert.NotNil(t, result)

	err = plugin.Shutdown()
	assert.NoError(t, err)

	capabilities := plugin.GetCapabilities()
	assert.Contains(t, capabilities, "sentiment_analysis")
}

func TestSimpleCodeReviewPlugin(t *testing.T) {
	plugin := &SimpleCodeReviewPlugin{}

	assert.Equal(t, "code_review", plugin.Name())
	assert.Equal(t, "1.0.0", plugin.Version())
	assert.Contains(t, plugin.Description(), "code review")

	err := plugin.Initialize(nil)
	assert.NoError(t, err)

	ctx := context.Background()
	result, err := plugin.Execute(ctx, "function test() {}")

	assert.NoError(t, err)
	assert.NotNil(t, result)

	err = plugin.Shutdown()
	assert.NoError(t, err)

	capabilities := plugin.GetCapabilities()
	assert.Contains(t, capabilities, "code_review")
}

func TestSimplePerformancePlugin(t *testing.T) {
	plugin := &SimplePerformancePlugin{}

	assert.Equal(t, "performance_analysis", plugin.Name())
	assert.Equal(t, "1.0.0", plugin.Version())
	assert.Contains(t, plugin.Description(), "performance")

	err := plugin.Initialize(nil)
	assert.NoError(t, err)

	ctx := context.Background()
	input := map[string]interface{}{"response_time": 150.0}
	result, err := plugin.Execute(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, result)

	err = plugin.Shutdown()
	assert.NoError(t, err)

	capabilities := plugin.GetCapabilities()
	assert.Contains(t, capabilities, "performance_analysis")
}

// ==================== Plugin Interface Compliance ====================

func TestPluginInterfaceImplementation(t *testing.T) {
	var _ Plugin = (*SimpleSentimentPlugin)(nil)
	var _ Plugin = (*SimpleCodeReviewPlugin)(nil)
	var _ Plugin = (*SimplePerformancePlugin)(nil)

	assert.True(t, true)
}
