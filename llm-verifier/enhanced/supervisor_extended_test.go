package enhanced

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Helper Function Tests ====================

func TestIntPtr(t *testing.T) {
	result := intPtr(42)
	require.NotNil(t, result)
	assert.Equal(t, 42, *result)
}

func TestFloat64Ptr(t *testing.T) {
	result := float64Ptr(3.14)
	require.NotNil(t, result)
	assert.Equal(t, 3.14, *result)
}

// ==================== Supervisor Tests ====================

func TestNewSupervisor(t *testing.T) {
	supervisor := NewSupervisor(nil, 5)

	require.NotNil(t, supervisor)
	assert.Equal(t, 5, supervisor.maxWorkers)
	assert.NotNil(t, supervisor.workers)
	assert.NotNil(t, supervisor.tasks)
	assert.NotNil(t, supervisor.taskQueue)
	assert.NotNil(t, supervisor.resultChan)
	assert.NotNil(t, supervisor.stopCh)
	assert.NotNil(t, supervisor.taskHandlers)
}

func TestSupervisorStartStop(t *testing.T) {
	supervisor := NewSupervisor(nil, 2)

	err := supervisor.Start()
	assert.NoError(t, err)

	// Give time for workers to initialize
	time.Sleep(100 * time.Millisecond)

	supervisor.Stop()

	// Test should complete without deadlock
}

func TestSupervisorDecomposeTask(t *testing.T) {
	supervisor := NewSupervisor(nil, 2)

	tasks, err := supervisor.DecomposeTask("Analyze this code", nil)

	assert.NoError(t, err)
	require.NotEmpty(t, tasks)
	assert.Equal(t, "analysis", tasks[0].Type)
}

func TestSupervisorDecomposeTaskWithCodeKeyword(t *testing.T) {
	supervisor := NewSupervisor(nil, 2)

	tasks, err := supervisor.DecomposeTask("Please review this code", nil)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(tasks), 2) // Should create code subtask
}

func TestSupervisorDecomposeTaskWithTestKeyword(t *testing.T) {
	supervisor := NewSupervisor(nil, 2)

	tasks, err := supervisor.DecomposeTask("Please test and validate", nil)

	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(tasks), 2) // Should create testing subtask
}

func TestSupervisorGetTaskStatus(t *testing.T) {
	supervisor := NewSupervisor(nil, 2)

	// Add a task directly
	task := &Task{
		ID:     "test_task_1",
		Type:   "test",
		Status: "pending",
	}
	supervisor.tasks["test_task_1"] = task

	result, err := supervisor.GetTaskStatus("test_task_1")

	assert.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "test_task_1", result.ID)
}

func TestSupervisorGetTaskStatusNotFound(t *testing.T) {
	supervisor := NewSupervisor(nil, 2)

	_, err := supervisor.GetTaskStatus("nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSupervisorGetWorkerStatus(t *testing.T) {
	supervisor := NewSupervisor(nil, 2)

	// Add a worker directly
	worker := &Worker{
		ID:            "worker_1",
		Status:        "idle",
		Capabilities:  []string{"test"},
		LastHeartbeat: time.Now(),
		Performance:   map[string]interface{}{"tasks": 0},
	}
	supervisor.workers["worker_1"] = worker

	status := supervisor.GetWorkerStatus()

	require.NotNil(t, status)
	assert.Contains(t, status, "workers")
	assert.Contains(t, status, "total")
	assert.Contains(t, status, "max_workers")
	assert.Equal(t, 2, status["max_workers"])
}

func TestSupervisorGetSystemStatus(t *testing.T) {
	supervisor := NewSupervisor(nil, 2)

	// Add various tasks
	supervisor.tasks["pending_task"] = &Task{ID: "pending_task", Status: "pending"}
	supervisor.tasks["running_task"] = &Task{ID: "running_task", Status: "running"}
	supervisor.tasks["completed_task"] = &Task{ID: "completed_task", Status: "completed"}
	supervisor.tasks["failed_task"] = &Task{ID: "failed_task", Status: "failed"}

	// Add a busy worker
	supervisor.workers["busy_worker"] = &Worker{
		ID:          "busy_worker",
		Status:      "busy",
		Performance: map[string]interface{}{},
	}

	status := supervisor.GetSystemStatus()

	require.NotNil(t, status)
	assert.Contains(t, status, "tasks")
	assert.Contains(t, status, "workers")
	assert.Contains(t, status, "queue_size")

	tasks := status["tasks"].(map[string]int)
	assert.Equal(t, 1, tasks["pending"])
	assert.Equal(t, 1, tasks["running"])
	assert.Equal(t, 1, tasks["completed"])
	assert.Equal(t, 1, tasks["failed"])
	assert.Equal(t, 4, tasks["total"])
}

// ==================== AIAssistant Helper Tests ====================

func TestGetSuccessRateMessage(t *testing.T) {
	// Create AI assistant without database to test helper methods
	ai := &AIAssistant{
		context: make(map[string][]string),
	}

	// CONST-046 round-431: getSuccessRateMessage routes through the
	// i18n seam. Under the default NoopTranslator the helper returns
	// the stable message ID verbatim, which is what we assert here.
	tests := []struct {
		rate     float64
		expected string
	}{
		{0.95, "enhanced.supervisor.success_rate.excellent"},
		{0.90, "enhanced.supervisor.success_rate.good"},
		{0.85, "enhanced.supervisor.success_rate.good"},
		{0.80, "enhanced.supervisor.success_rate.acceptable"},
		{0.75, "enhanced.supervisor.success_rate.acceptable"},
		{0.70, "enhanced.supervisor.success_rate.needs_attention"},
	}

	for _, tt := range tests {
		result := ai.getSuccessRateMessage(tt.rate)
		assert.Equal(t, tt.expected, result)
	}
}

func TestGetScoreMessage(t *testing.T) {
	ai := &AIAssistant{
		context: make(map[string][]string),
	}

	// CONST-046 round-431: getScoreMessage routes through the i18n
	// seam; NoopTranslator returns the stable message ID verbatim.
	tests := []struct {
		score    float64
		expected string
	}{
		{95, "enhanced.supervisor.score_quality.high"},
		{90, "enhanced.supervisor.score_quality.high"},
		{85, "enhanced.supervisor.score_quality.good"},
		{80, "enhanced.supervisor.score_quality.good"},
		{75, "enhanced.supervisor.score_quality.moderate"},
		{70, "enhanced.supervisor.score_quality.moderate"},
		{60, "enhanced.supervisor.score_quality.variable"},
	}

	for _, tt := range tests {
		result := ai.getScoreMessage(tt.score)
		assert.Equal(t, tt.expected, result)
	}
}

func TestGetRecommendations(t *testing.T) {
	ai := &AIAssistant{
		context: make(map[string][]string),
	}

	// CONST-046 round-431: getRecommendations routes every line
	// through the i18n seam; NoopTranslator returns message IDs.
	// Test with low score and failures.
	result := ai.getRecommendations(70, 5)
	assert.Contains(t, result, "enhanced.supervisor.recommendation.upgrade_models")
	assert.Contains(t, result, "enhanced.supervisor.recommendation.investigate_failures")

	// Test with good score and no failures.
	result = ai.getRecommendations(90, 0)
	assert.Contains(t, result, "enhanced.supervisor.recommendation.performing_well")
}

func TestAnalyzeIntent(t *testing.T) {
	ai := &AIAssistant{
		context: make(map[string][]string),
	}

	tests := []struct {
		message  string
		expected string
	}{
		{"Can you help me?", "help"},      // Contains "?" and "help"
		{"What is the status", "status"},  // No "?", contains "status"
		{"How are the systems", "status"}, // No "?", contains "how are"
		{"Suggest a model", "suggest"},
		{"Recommend something", "suggest"},
		{"Analyze this data", "analyze"},
		{"Check my setup", "analyze"},
		{"Configure settings", "configure"},
		{"Setting up", "configure"},
		{"Hello there", "general"},
	}

	for _, tt := range tests {
		result := ai.analyzeIntent(tt.message)
		assert.Equal(t, tt.expected, result, "For message: %s", tt.message)
	}
}

func TestAddToContext(t *testing.T) {
	ai := &AIAssistant{
		context: make(map[string][]string),
	}

	userID := "user1"

	// Add messages
	for i := 0; i < 15; i++ {
		ai.addToContext(userID, "message")
	}

	// Should only keep last 10
	assert.Len(t, ai.context[userID], 10)
}

func TestGenerateHelpResponse(t *testing.T) {
	ai := &AIAssistant{
		context: make(map[string][]string),
	}

	response := ai.generateHelpResponse()

	// CONST-046 round-431: help text routes through the i18n seam;
	// NoopTranslator returns the stable message ID verbatim.
	assert.Equal(t, "enhanced.supervisor.help.body", response)
}

func TestGenerateStatusResponse(t *testing.T) {
	ai := &AIAssistant{
		context: make(map[string][]string),
	}

	response := ai.generateStatusResponse()

	// CONST-046 round-431: status block routes through the i18n seam.
	assert.Equal(t, "enhanced.supervisor.status.body", response)
}

func TestGenerateSuggestionResponse(t *testing.T) {
	ai := &AIAssistant{
		context: make(map[string][]string),
	}

	// CONST-046 round-431: both suggestion branches route through the
	// i18n seam; NoopTranslator returns the stable message IDs.
	// Test with model keyword.
	response := ai.generateSuggestionResponse("suggest a model")
	assert.Equal(t, "enhanced.supervisor.suggestion.model", response)

	// Test without model keyword.
	response = ai.generateSuggestionResponse("suggest something")
	assert.Equal(t, "enhanced.supervisor.suggestion.general", response)
}

func TestGenerateConfigurationResponse(t *testing.T) {
	ai := &AIAssistant{
		context: make(map[string][]string),
	}

	response := ai.generateConfigurationResponse("configure something")

	// CONST-046 round-431: configuration block routes through the i18n seam.
	assert.Equal(t, "enhanced.supervisor.configuration.body", response)
}

func TestGenerateGeneralResponse(t *testing.T) {
	ai := &AIAssistant{
		context: make(map[string][]string),
	}

	response := ai.generateGeneralResponse("hello")

	// Should return one of the pre-defined responses
	assert.NotEmpty(t, response)
}

// ==================== Sentiment Plugin Tests ====================

func TestSimpleSentimentPluginWithGood(t *testing.T) {
	plugin := &SimpleSentimentPlugin{}
	ctx := context.Background()

	result, err := plugin.Execute(ctx, "This is good work")

	assert.NoError(t, err)
	require.NotNil(t, result)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, 0.8, resultMap["score"])
}

func TestSimpleSentimentPluginWithBad(t *testing.T) {
	plugin := &SimpleSentimentPlugin{}
	ctx := context.Background()

	result, err := plugin.Execute(ctx, "This is bad")

	assert.NoError(t, err)
	require.NotNil(t, result)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, 0.2, resultMap["score"])
}

func TestSimpleSentimentPluginWithNeutral(t *testing.T) {
	plugin := &SimpleSentimentPlugin{}
	ctx := context.Background()

	result, err := plugin.Execute(ctx, "This is a sentence")

	assert.NoError(t, err)
	require.NotNil(t, result)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, 0.5, resultMap["score"])
}

func TestSimpleSentimentPluginInvalidInput(t *testing.T) {
	plugin := &SimpleSentimentPlugin{}
	ctx := context.Background()

	_, err := plugin.Execute(ctx, 123) // Invalid input

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a string")
}

// ==================== Code Review Plugin Tests ====================

func TestSimpleCodeReviewPluginWithTODO(t *testing.T) {
	plugin := &SimpleCodeReviewPlugin{}
	ctx := context.Background()

	code := "function test() { // TODO: implement }"
	result, err := plugin.Execute(ctx, code)

	assert.NoError(t, err)
	require.NotNil(t, result)

	resultMap := result.(map[string]interface{})
	issues := resultMap["issues"].([]string)
	assert.Contains(t, issues, "Contains TODO comments")
}

func TestSimpleCodeReviewPluginLongCode(t *testing.T) {
	plugin := &SimpleCodeReviewPlugin{}
	ctx := context.Background()

	// Create long code
	code := ""
	for i := 0; i < 1001; i++ {
		code += "x"
	}

	result, err := plugin.Execute(ctx, code)

	assert.NoError(t, err)
	require.NotNil(t, result)

	resultMap := result.(map[string]interface{})
	issues := resultMap["issues"].([]string)
	assert.Contains(t, issues, "Long function/method")
}

func TestSimpleCodeReviewPluginInvalidInput(t *testing.T) {
	plugin := &SimpleCodeReviewPlugin{}
	ctx := context.Background()

	_, err := plugin.Execute(ctx, 123) // Invalid input

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a string")
}

// ==================== Performance Plugin Tests ====================

func TestSimplePerformancePluginGood(t *testing.T) {
	plugin := &SimplePerformancePlugin{}
	ctx := context.Background()

	input := map[string]interface{}{"response_time": 100.0}
	result, err := plugin.Execute(ctx, input)

	assert.NoError(t, err)
	require.NotNil(t, result)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, "good", resultMap["performance"])
}

func TestSimplePerformancePluginFair(t *testing.T) {
	plugin := &SimplePerformancePlugin{}
	ctx := context.Background()

	input := map[string]interface{}{"response_time": 300.0}
	result, err := plugin.Execute(ctx, input)

	assert.NoError(t, err)
	require.NotNil(t, result)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, "fair", resultMap["performance"])
}

func TestSimplePerformancePluginPoor(t *testing.T) {
	plugin := &SimplePerformancePlugin{}
	ctx := context.Background()

	input := map[string]interface{}{"response_time": 600.0}
	result, err := plugin.Execute(ctx, input)

	assert.NoError(t, err)
	require.NotNil(t, result)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, "poor", resultMap["performance"])
}

func TestSimplePerformancePluginInvalidInput(t *testing.T) {
	plugin := &SimplePerformancePlugin{}
	ctx := context.Background()

	_, err := plugin.Execute(ctx, "invalid") // Invalid input

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a map")
}

func TestSimplePerformancePluginDefaultResponseTime(t *testing.T) {
	plugin := &SimplePerformancePlugin{}
	ctx := context.Background()

	input := map[string]interface{}{} // No response_time
	result, err := plugin.Execute(ctx, input)

	assert.NoError(t, err)
	require.NotNil(t, result)

	resultMap := result.(map[string]interface{})
	assert.Equal(t, 100.0, resultMap["response_time_ms"]) // Default value
}

// ==================== CacheManager Extended Tests ====================

func TestCacheManagerDeleteDisabled(t *testing.T) {
	backend := NewInMemoryCache(100)
	cm := NewCacheManager(backend, false)

	// Set via backend directly
	backend.Set("key1", "value1", 5*time.Minute)

	// Delete via manager (disabled)
	err := cm.Delete("key1")

	assert.NoError(t, err)

	// Value should still exist in backend
	_, exists := backend.Get("key1")
	assert.True(t, exists)
}

func TestCacheManagerClearDisabled(t *testing.T) {
	backend := NewInMemoryCache(100)
	cm := NewCacheManager(backend, false)

	// Set via backend directly
	backend.Set("key1", "value1", 5*time.Minute)

	// Clear via manager (disabled)
	err := cm.Clear()

	assert.NoError(t, err)

	// Value should still exist in backend
	_, exists := backend.Get("key1")
	assert.True(t, exists)
}

// ==================== Plugin Manager Extended Tests ====================

func TestPluginManagerWithNilLogger(t *testing.T) {
	// Should not panic
	pm := NewPluginManager(nil)
	require.NotNil(t, pm)
}

func TestPluginManagerWithLogger(t *testing.T) {
	logger := log.Default()
	pm := NewPluginManager(logger)

	require.NotNil(t, pm)
	assert.Equal(t, logger, pm.logger)
}

// ==================== SupervisorConfig Extended Tests ====================

func TestSupervisorConfigNegativeMaxJobs(t *testing.T) {
	config := SupervisorConfig{
		MaxConcurrentJobs:   -1,
		JobTimeout:          5 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
	}

	err := config.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max concurrent jobs")
}

func TestSupervisorConfigNegativeTimeout(t *testing.T) {
	config := SupervisorConfig{
		MaxConcurrentJobs:   10,
		JobTimeout:          -5 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
	}

	err := config.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "job timeout")
}

func TestSupervisorConfigNegativeHealthCheck(t *testing.T) {
	config := SupervisorConfig{
		MaxConcurrentJobs:   10,
		JobTimeout:          5 * time.Minute,
		HealthCheckInterval: -30 * time.Second,
	}

	err := config.Validate()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "health check interval")
}

func TestSupervisorConfigAllOptions(t *testing.T) {
	config := SupervisorConfig{
		MaxConcurrentJobs:    10,
		JobTimeout:           5 * time.Minute,
		HealthCheckInterval:  30 * time.Second,
		RetryAttempts:        3,
		RetryBackoff:         1 * time.Second,
		EnableAutoScaling:    true,
		EnablePredictions:    true,
		EnableAdaptiveLoad:   true,
		EnableCircuitBreaker: true,
		HighLoadThreshold:    0.8,
		LowLoadThreshold:     0.2,
		ErrorRateThreshold:   0.1,
		MemoryThreshold:      0.9,
	}

	err := config.Validate()

	assert.NoError(t, err)
}
