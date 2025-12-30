package testsuite

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Constant Tests ====================

func TestTestCaseType_Constants(t *testing.T) {
	assert.Equal(t, TestCaseType("basic"), TestCaseTypeBasic)
	assert.Equal(t, TestCaseType("comparison"), TestCaseTypeComparison)
	assert.Equal(t, TestCaseType("load"), TestCaseTypeLoad)
	assert.Equal(t, TestCaseType("stress"), TestCaseTypeStress)
	assert.Equal(t, TestCaseType("security"), TestCaseTypeSecurity)
	assert.Equal(t, TestCaseType("compliance"), TestCaseTypeCompliance)
	assert.Equal(t, TestCaseType("multimodal"), TestCaseTypeMultiModal)
	assert.Equal(t, TestCaseType("custom"), TestCaseTypeCustom)
}

func TestTestPriority_Constants(t *testing.T) {
	assert.Equal(t, TestPriority("low"), TestPriorityLow)
	assert.Equal(t, TestPriority("medium"), TestPriorityMedium)
	assert.Equal(t, TestPriority("high"), TestPriorityHigh)
	assert.Equal(t, TestPriority("critical"), TestPriorityCritical)
}

func TestExecutionMode_Constants(t *testing.T) {
	assert.Equal(t, ExecutionMode("sequential"), ExecutionModeSequential)
	assert.Equal(t, ExecutionMode("parallel"), ExecutionModeParallel)
	assert.Equal(t, ExecutionMode("distributed"), ExecutionModeDistributed)
}

func TestTestStatus_Constants(t *testing.T) {
	assert.Equal(t, TestStatus("pending"), TestStatusPending)
	assert.Equal(t, TestStatus("running"), TestStatusRunning)
	assert.Equal(t, TestStatus("passed"), TestStatusPassed)
	assert.Equal(t, TestStatus("failed"), TestStatusFailed)
	assert.Equal(t, TestStatus("skipped"), TestStatusSkipped)
	assert.Equal(t, TestStatus("error"), TestStatusError)
}

// ==================== TestSuiteBuilder Tests ====================

func TestNewTestSuiteBuilder(t *testing.T) {
	builder := NewTestSuiteBuilder("Test Suite", "Test Description")

	require.NotNil(t, builder)
	require.NotNil(t, builder.suite)
	assert.Equal(t, "Test Suite", builder.suite.Name)
	assert.Equal(t, "Test Description", builder.suite.Description)
	assert.Equal(t, "1.0.0", builder.suite.Version)
	assert.NotEmpty(t, builder.suite.ID)
	assert.Equal(t, ExecutionModeParallel, builder.suite.Configuration.ExecutionMode)
	assert.Equal(t, 5, builder.suite.Configuration.Parallelism)
	assert.Equal(t, 300*time.Second, builder.suite.Configuration.Timeout)
}

func TestTestSuiteBuilder_WithVersion(t *testing.T) {
	builder := NewTestSuiteBuilder("Test", "Test").
		WithVersion("2.0.0")

	assert.Equal(t, "2.0.0", builder.suite.Version)
}

func TestTestSuiteBuilder_WithAuthor(t *testing.T) {
	builder := NewTestSuiteBuilder("Test", "Test").
		WithAuthor("Test Author")

	assert.Equal(t, "Test Author", builder.suite.Author)
}

func TestTestSuiteBuilder_WithTags(t *testing.T) {
	builder := NewTestSuiteBuilder("Test", "Test").
		WithTags("tag1", "tag2", "tag3")

	assert.Len(t, builder.suite.Tags, 3)
	assert.Contains(t, builder.suite.Tags, "tag1")
	assert.Contains(t, builder.suite.Tags, "tag2")
	assert.Contains(t, builder.suite.Tags, "tag3")
}

func TestTestSuiteBuilder_WithExecutionMode(t *testing.T) {
	t.Run("sequential", func(t *testing.T) {
		builder := NewTestSuiteBuilder("Test", "Test").
			WithExecutionMode(ExecutionModeSequential)

		assert.Equal(t, ExecutionModeSequential, builder.suite.Configuration.ExecutionMode)
	})

	t.Run("parallel", func(t *testing.T) {
		builder := NewTestSuiteBuilder("Test", "Test").
			WithExecutionMode(ExecutionModeParallel)

		assert.Equal(t, ExecutionModeParallel, builder.suite.Configuration.ExecutionMode)
	})

	t.Run("distributed", func(t *testing.T) {
		builder := NewTestSuiteBuilder("Test", "Test").
			WithExecutionMode(ExecutionModeDistributed)

		assert.Equal(t, ExecutionModeDistributed, builder.suite.Configuration.ExecutionMode)
	})
}

func TestTestSuiteBuilder_WithParallelism(t *testing.T) {
	builder := NewTestSuiteBuilder("Test", "Test").
		WithParallelism(10)

	assert.Equal(t, 10, builder.suite.Configuration.Parallelism)
}

func TestTestSuiteBuilder_WithProviders(t *testing.T) {
	builder := NewTestSuiteBuilder("Test", "Test").
		WithProviders("openai", "anthropic", "google")

	assert.Len(t, builder.suite.Configuration.Providers, 3)
	assert.Contains(t, builder.suite.Configuration.Providers, "openai")
	assert.Contains(t, builder.suite.Configuration.Providers, "anthropic")
	assert.Contains(t, builder.suite.Configuration.Providers, "google")
}

func TestTestSuiteBuilder_AddTestCase(t *testing.T) {
	testCase := TestCase{
		Name:        "Custom Test",
		Description: "Custom test case",
		Type:        TestCaseTypeCustom,
		Priority:    TestPriorityHigh,
		Enabled:     true,
	}

	builder := NewTestSuiteBuilder("Test", "Test").
		AddTestCase(testCase)

	require.Len(t, builder.suite.TestCases, 1)
	assert.NotEmpty(t, builder.suite.TestCases[0].ID)
	assert.Equal(t, "Custom Test", builder.suite.TestCases[0].Name)
	assert.True(t, builder.suite.TestCases[0].Enabled)
}

func TestTestSuiteBuilder_AddBasicTestCase(t *testing.T) {
	builder := NewTestSuiteBuilder("Test", "Test").
		AddBasicTestCase("Greeting Test", "Say hello", []string{"hello", "hi"})

	require.Len(t, builder.suite.TestCases, 1)
	tc := builder.suite.TestCases[0]

	assert.Equal(t, "Greeting Test", tc.Name)
	assert.Equal(t, TestCaseTypeBasic, tc.Type)
	assert.Equal(t, TestPriorityMedium, tc.Priority)
	assert.Equal(t, "basic", tc.Category)
	assert.Equal(t, "Say hello", tc.Inputs.Prompt)
	assert.Equal(t, []string{"hello", "hi"}, tc.Expected.Contains)
	assert.True(t, tc.Enabled)
}

func TestTestSuiteBuilder_AddComparisonTestCase(t *testing.T) {
	builder := NewTestSuiteBuilder("Test", "Test").
		AddComparisonTestCase("Provider Comparison", "Compare responses")

	require.Len(t, builder.suite.TestCases, 1)
	tc := builder.suite.TestCases[0]

	assert.Equal(t, "Provider Comparison", tc.Name)
	assert.Equal(t, TestCaseTypeComparison, tc.Type)
	assert.Equal(t, TestPriorityHigh, tc.Priority)
	assert.Equal(t, "comparison", tc.Category)
	assert.Equal(t, "Compare responses", tc.Inputs.Prompt)
}

func TestTestSuiteBuilder_AddLoadTestCase(t *testing.T) {
	builder := NewTestSuiteBuilder("Test", "Test").
		AddLoadTestCase("Load Test", 50, 120)

	require.Len(t, builder.suite.TestCases, 1)
	tc := builder.suite.TestCases[0]

	assert.Equal(t, "Load Test", tc.Name)
	assert.Equal(t, TestCaseTypeLoad, tc.Type)
	assert.Equal(t, TestPriorityHigh, tc.Priority)
	assert.Equal(t, "performance", tc.Category)
	assert.Equal(t, 50, tc.Metadata["concurrent_users"])
	assert.Equal(t, 120, tc.Metadata["duration_seconds"])
}

func TestTestSuiteBuilder_AddMultiModalTestCase(t *testing.T) {
	builder := NewTestSuiteBuilder("Test", "Test").
		AddMultiModalTestCase("Image Test", "Describe the image", "image/jpeg")

	require.Len(t, builder.suite.TestCases, 1)
	tc := builder.suite.TestCases[0]

	assert.Equal(t, "Image Test", tc.Name)
	assert.Equal(t, TestCaseTypeMultiModal, tc.Type)
	assert.Equal(t, "multimodal", tc.Category)
	assert.Equal(t, "Describe the image", tc.Inputs.Prompt)
	require.Len(t, tc.Inputs.Files, 1)
	assert.Equal(t, "image/jpeg", tc.Inputs.Files[0].Type)
}

func TestTestSuiteBuilder_Build(t *testing.T) {
	builder := NewTestSuiteBuilder("Final Suite", "Final description").
		WithVersion("3.0.0").
		WithAuthor("Tester").
		WithTags("final", "test").
		AddBasicTestCase("Test 1", "Prompt 1", []string{"expected"})

	suite := builder.Build()

	require.NotNil(t, suite)
	assert.Equal(t, "Final Suite", suite.Name)
	assert.Equal(t, "3.0.0", suite.Version)
	assert.Equal(t, "Tester", suite.Author)
	assert.Len(t, suite.Tags, 2)
	assert.Len(t, suite.TestCases, 1)
	assert.NotZero(t, suite.UpdatedAt)
}

func TestTestSuiteBuilder_FluentChaining(t *testing.T) {
	suite := NewTestSuiteBuilder("Chained Suite", "Testing fluent chaining").
		WithVersion("1.0.0").
		WithAuthor("Author").
		WithTags("tag1", "tag2").
		WithExecutionMode(ExecutionModeSequential).
		WithParallelism(1).
		WithProviders("openai", "anthropic").
		AddBasicTestCase("Test 1", "Prompt 1", []string{"a"}).
		AddComparisonTestCase("Test 2", "Prompt 2").
		AddLoadTestCase("Test 3", 10, 60).
		AddMultiModalTestCase("Test 4", "Prompt 4", "image/png").
		Build()

	require.NotNil(t, suite)
	assert.Equal(t, "Chained Suite", suite.Name)
	assert.Equal(t, "1.0.0", suite.Version)
	assert.Equal(t, "Author", suite.Author)
	assert.Len(t, suite.Tags, 2)
	assert.Equal(t, ExecutionModeSequential, suite.Configuration.ExecutionMode)
	assert.Equal(t, 1, suite.Configuration.Parallelism)
	assert.Len(t, suite.Configuration.Providers, 2)
	assert.Len(t, suite.TestCases, 4)
}

// ==================== TestSuiteExecutor Tests ====================

func TestNewTestSuiteExecutor(t *testing.T) {
	suite := NewTestSuiteBuilder("Test", "Test").Build()

	executor := NewTestSuiteExecutor(suite)

	require.NotNil(t, executor)
	assert.Equal(t, suite, executor.suite)
	assert.NotNil(t, executor.results)
}

func TestNewTestSuiteExecutorWithClient(t *testing.T) {
	suite := NewTestSuiteBuilder("Test", "Test").Build()

	executor := NewTestSuiteExecutorWithClient(suite, nil)

	require.NotNil(t, executor)
	assert.Nil(t, executor.llmClient)
}

func TestTestSuiteExecutor_SetLLMClient(t *testing.T) {
	suite := NewTestSuiteBuilder("Test", "Test").Build()
	executor := NewTestSuiteExecutor(suite)

	// LLM client can be set to nil for testing
	executor.SetLLMClient(nil)

	assert.Nil(t, executor.llmClient)
}

func TestTestSuiteExecutor_Execute_Sequential_NoClient(t *testing.T) {
	suite := NewTestSuiteBuilder("Test", "Test").
		WithExecutionMode(ExecutionModeSequential).
		AddBasicTestCase("Test 1", "Hello", []string{"hi"}).
		Build()

	executor := NewTestSuiteExecutor(suite)
	ctx := context.Background()

	report, err := executor.Execute(ctx)

	require.NoError(t, err)
	require.NotNil(t, report)
	assert.Equal(t, suite.ID, report.SuiteID)
	assert.Equal(t, suite.Name, report.SuiteName)
	assert.NotZero(t, report.StartTime)
	assert.NotZero(t, report.EndTime)
	// Tests will fail without LLM client
	assert.Len(t, report.TestResults, 1)
	assert.Equal(t, TestStatusError, report.TestResults[0].Status)
}

func TestTestSuiteExecutor_Execute_Parallel_NoClient(t *testing.T) {
	suite := NewTestSuiteBuilder("Test", "Test").
		WithExecutionMode(ExecutionModeParallel).
		WithParallelism(2).
		AddBasicTestCase("Test 1", "Hello", []string{"hi"}).
		AddBasicTestCase("Test 2", "World", []string{"world"}).
		Build()

	executor := NewTestSuiteExecutor(suite)
	ctx := context.Background()

	report, err := executor.Execute(ctx)

	require.NoError(t, err)
	require.NotNil(t, report)
	// Tests fail without LLM client but still execute
	assert.GreaterOrEqual(t, len(report.TestResults), 1)
}

func TestTestSuiteExecutor_Execute_Distributed_NoClient(t *testing.T) {
	suite := NewTestSuiteBuilder("Test", "Test").
		WithExecutionMode(ExecutionModeDistributed).
		AddBasicTestCase("Test 1", "Hello", []string{"hi"}).
		Build()

	executor := NewTestSuiteExecutor(suite)
	ctx := context.Background()

	report, err := executor.Execute(ctx)

	require.NoError(t, err)
	require.NotNil(t, report)
}

func TestTestSuiteExecutor_Execute_DisabledTestCase(t *testing.T) {
	suite := NewTestSuiteBuilder("Test", "Test").
		WithExecutionMode(ExecutionModeSequential).
		Build()

	// Add a disabled test case manually
	suite.TestCases = append(suite.TestCases, TestCase{
		ID:      "disabled-test",
		Name:    "Disabled Test",
		Enabled: false,
	})

	executor := NewTestSuiteExecutor(suite)
	ctx := context.Background()

	report, err := executor.Execute(ctx)

	require.NoError(t, err)
	require.NotNil(t, report)
	// Disabled tests should be skipped
	assert.Empty(t, report.TestResults)
}

func TestTestSuiteExecutor_Execute_UnsupportedTestType(t *testing.T) {
	suite := NewTestSuiteBuilder("Test", "Test").
		WithExecutionMode(ExecutionModeSequential).
		Build()

	// Add a test case with unsupported type
	suite.TestCases = append(suite.TestCases, TestCase{
		ID:      "unsupported-test",
		Name:    "Unsupported Test",
		Type:    TestCaseType("unsupported"),
		Enabled: true,
	})

	executor := NewTestSuiteExecutor(suite)
	ctx := context.Background()

	report, err := executor.Execute(ctx)

	require.NoError(t, err)
	require.Len(t, report.TestResults, 1)
	assert.Equal(t, TestStatusError, report.TestResults[0].Status)
	assert.Contains(t, report.TestResults[0].Error, "Unsupported test case type")
}

func TestTestSuiteExecutor_checkExpectations(t *testing.T) {
	suite := NewTestSuiteBuilder("Test", "Test").Build()
	executor := NewTestSuiteExecutor(suite)

	t.Run("passes all expectations", func(t *testing.T) {
		testCase := TestCase{
			Expected: TestExpected{
				Contains:       []string{"hello"},
				NotContains:    []string{"error"},
				MinLength:      5,
				MaxLength:      100,
				ResponseTime:   10 * time.Second,
				ScoreThreshold: 0.5,
			},
		}
		result := TestResult{
			Response: "hello world",
			Duration: 1 * time.Second,
			Score:    1.0,
		}

		passed := executor.checkExpectations(testCase, result)
		assert.True(t, passed)
	})

	t.Run("fails response time", func(t *testing.T) {
		testCase := TestCase{
			Expected: TestExpected{
				ResponseTime: 1 * time.Second,
			},
		}
		result := TestResult{
			Duration: 2 * time.Second,
		}

		passed := executor.checkExpectations(testCase, result)
		assert.False(t, passed)
	})

	t.Run("fails score threshold", func(t *testing.T) {
		testCase := TestCase{
			Expected: TestExpected{
				ScoreThreshold: 0.8,
			},
		}
		result := TestResult{
			Score: 0.5,
		}

		passed := executor.checkExpectations(testCase, result)
		assert.False(t, passed)
	})

	t.Run("fails contains check", func(t *testing.T) {
		testCase := TestCase{
			Expected: TestExpected{
				Contains: []string{"missing"},
			},
		}
		result := TestResult{
			Response: "hello world",
		}

		passed := executor.checkExpectations(testCase, result)
		assert.False(t, passed)
	})

	t.Run("fails not contains check", func(t *testing.T) {
		testCase := TestCase{
			Expected: TestExpected{
				NotContains: []string{"hello"},
			},
		}
		result := TestResult{
			Response: "hello world",
		}

		passed := executor.checkExpectations(testCase, result)
		assert.False(t, passed)
	})

	t.Run("fails min length", func(t *testing.T) {
		testCase := TestCase{
			Expected: TestExpected{
				MinLength: 100,
			},
		}
		result := TestResult{
			Response: "short",
		}

		passed := executor.checkExpectations(testCase, result)
		assert.False(t, passed)
	})

	t.Run("fails max length", func(t *testing.T) {
		testCase := TestCase{
			Expected: TestExpected{
				MaxLength: 5,
			},
		}
		result := TestResult{
			Response: "this is too long",
		}

		passed := executor.checkExpectations(testCase, result)
		assert.False(t, passed)
	})
}

func TestTestSuiteExecutor_generateSummary(t *testing.T) {
	suite := NewTestSuiteBuilder("Test", "Test").Build()
	executor := NewTestSuiteExecutor(suite)

	t.Run("empty results", func(t *testing.T) {
		summary := executor.generateSummary([]TestResult{})

		assert.Equal(t, 0, summary.TotalTests)
		assert.Equal(t, 0, summary.PassedTests)
		assert.Equal(t, 0, summary.FailedTests)
		assert.Equal(t, 0.0, summary.AvgScore)
	})

	t.Run("mixed results", func(t *testing.T) {
		results := []TestResult{
			{Status: TestStatusPassed, Score: 1.0, Duration: 1 * time.Second},
			{Status: TestStatusFailed, Score: 0.5, Duration: 2 * time.Second},
			{Status: TestStatusSkipped, Score: 0.0, Duration: 0},
			{Status: TestStatusError, Score: 0.0, Duration: 500 * time.Millisecond},
		}

		summary := executor.generateSummary(results)

		assert.Equal(t, 4, summary.TotalTests)
		assert.Equal(t, 1, summary.PassedTests)
		assert.Equal(t, 1, summary.FailedTests)
		assert.Equal(t, 1, summary.SkippedTests)
		assert.Equal(t, 1, summary.ErrorTests)
		assert.InDelta(t, 0.375, summary.AvgScore, 0.001)
		assert.NotZero(t, summary.TotalDuration)
	})

	t.Run("all passed", func(t *testing.T) {
		results := []TestResult{
			{Status: TestStatusPassed, Score: 1.0, Duration: 1 * time.Second},
			{Status: TestStatusPassed, Score: 0.9, Duration: 2 * time.Second},
			{Status: TestStatusPassed, Score: 0.8, Duration: 3 * time.Second},
		}

		summary := executor.generateSummary(results)

		assert.Equal(t, 3, summary.TotalTests)
		assert.Equal(t, 3, summary.PassedTests)
		assert.Equal(t, 0, summary.FailedTests)
		assert.InDelta(t, 0.9, summary.AvgScore, 0.001)
		assert.NotZero(t, summary.P50Duration)
		assert.NotZero(t, summary.P95Duration)
	})
}

// ==================== TestSuiteManager Tests ====================

func TestNewTestSuiteManager(t *testing.T) {
	manager := NewTestSuiteManager()

	require.NotNil(t, manager)
	assert.NotNil(t, manager.suites)
	assert.Len(t, manager.suites, 0)
}

func TestTestSuiteManager_SaveSuite(t *testing.T) {
	manager := NewTestSuiteManager()
	suite := NewTestSuiteBuilder("Test Suite", "Description").Build()

	err := manager.SaveSuite(suite)

	require.NoError(t, err)
	assert.Len(t, manager.suites, 1)
}

func TestTestSuiteManager_GetSuite(t *testing.T) {
	manager := NewTestSuiteManager()
	suite := NewTestSuiteBuilder("Test Suite", "Description").Build()
	manager.SaveSuite(suite)

	t.Run("existing suite", func(t *testing.T) {
		retrieved, err := manager.GetSuite(suite.ID)

		require.NoError(t, err)
		require.NotNil(t, retrieved)
		assert.Equal(t, suite.ID, retrieved.ID)
		assert.Equal(t, suite.Name, retrieved.Name)
	})

	t.Run("non-existing suite", func(t *testing.T) {
		retrieved, err := manager.GetSuite("non-existent-id")

		assert.Error(t, err)
		assert.Nil(t, retrieved)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestTestSuiteManager_ListSuites(t *testing.T) {
	manager := NewTestSuiteManager()

	t.Run("empty list", func(t *testing.T) {
		suites := manager.ListSuites()

		assert.NotNil(t, suites)
		assert.Len(t, suites, 0)
	})

	t.Run("multiple suites", func(t *testing.T) {
		suite1 := NewTestSuiteBuilder("Suite 1", "Description 1").Build()
		suite2 := NewTestSuiteBuilder("Suite 2", "Description 2").Build()

		manager.SaveSuite(suite1)
		manager.SaveSuite(suite2)

		suites := manager.ListSuites()

		assert.Len(t, suites, 2)
	})
}

func TestTestSuiteManager_DeleteSuite(t *testing.T) {
	manager := NewTestSuiteManager()
	suite := NewTestSuiteBuilder("Test Suite", "Description").Build()
	manager.SaveSuite(suite)

	t.Run("delete existing", func(t *testing.T) {
		err := manager.DeleteSuite(suite.ID)

		require.NoError(t, err)
		assert.Len(t, manager.suites, 0)
	})

	t.Run("delete non-existing", func(t *testing.T) {
		err := manager.DeleteSuite("non-existent-id")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestTestSuiteManager_ExportSuite(t *testing.T) {
	manager := NewTestSuiteManager()
	suite := NewTestSuiteBuilder("Test Suite", "Description").
		WithAuthor("Author").
		WithTags("tag1").
		Build()
	manager.SaveSuite(suite)

	t.Run("export existing", func(t *testing.T) {
		data, err := manager.ExportSuite(suite.ID)

		require.NoError(t, err)
		require.NotEmpty(t, data)

		// Verify JSON is valid
		var exported TestSuite
		err = json.Unmarshal(data, &exported)
		require.NoError(t, err)
		assert.Equal(t, suite.Name, exported.Name)
		assert.Equal(t, suite.Author, exported.Author)
	})

	t.Run("export non-existing", func(t *testing.T) {
		data, err := manager.ExportSuite("non-existent-id")

		assert.Error(t, err)
		assert.Nil(t, data)
	})
}

func TestTestSuiteManager_ImportSuite(t *testing.T) {
	manager := NewTestSuiteManager()

	t.Run("valid import", func(t *testing.T) {
		jsonData := `{
			"id": "original-id",
			"name": "Imported Suite",
			"description": "Imported description",
			"version": "1.0.0",
			"author": "Importer",
			"tags": ["imported"],
			"test_cases": [],
			"configuration": {
				"execution_mode": "sequential",
				"parallelism": 1
			}
		}`

		imported, err := manager.ImportSuite([]byte(jsonData))

		require.NoError(t, err)
		require.NotNil(t, imported)
		assert.NotEqual(t, "original-id", imported.ID) // New ID generated
		assert.Equal(t, "Imported Suite", imported.Name)
		assert.Equal(t, "Importer", imported.Author)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		imported, err := manager.ImportSuite([]byte("invalid json"))

		assert.Error(t, err)
		assert.Nil(t, imported)
	})
}

// ==================== CreateTemplateSuites Tests ====================

func TestCreateTemplateSuites(t *testing.T) {
	suites := CreateTemplateSuites()

	require.NotEmpty(t, suites)
	assert.Len(t, suites, 4) // Basic, Performance, Comparison, Multi-modal

	// Check basic suite
	basicSuite := suites[0]
	assert.Equal(t, "Basic LLM Tests", basicSuite.Name)
	assert.Contains(t, basicSuite.Tags, "basic")
	assert.NotEmpty(t, basicSuite.TestCases)

	// Check performance suite
	perfSuite := suites[1]
	assert.Equal(t, "Performance Tests", perfSuite.Name)
	assert.Contains(t, perfSuite.Tags, "performance")
	assert.Equal(t, ExecutionModeParallel, perfSuite.Configuration.ExecutionMode)
	assert.Equal(t, 10, perfSuite.Configuration.Parallelism)

	// Check comparison suite
	compSuite := suites[2]
	assert.Equal(t, "Provider Comparison", compSuite.Name)
	assert.Contains(t, compSuite.Tags, "comparison")
	assert.NotEmpty(t, compSuite.Configuration.Providers)

	// Check multi-modal suite
	mmSuite := suites[3]
	assert.Equal(t, "Multi-Modal Tests", mmSuite.Name)
	assert.Contains(t, mmSuite.Tags, "multimodal")
}

// ==================== Struct Tests ====================

func TestTestSuite_Structure(t *testing.T) {
	suite := &TestSuite{
		ID:          "test-id",
		Name:        "Test Suite",
		Description: "Test description",
		Version:     "1.0.0",
		Author:      "Tester",
		Tags:        []string{"tag1", "tag2"},
		TestCases:   []TestCase{},
		Metadata:    map[string]interface{}{"key": "value"},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	assert.Equal(t, "test-id", suite.ID)
	assert.Equal(t, "Test Suite", suite.Name)
	assert.Len(t, suite.Tags, 2)
}

func TestTestCase_Structure(t *testing.T) {
	tc := &TestCase{
		ID:          "test-case-id",
		Name:        "Test Case",
		Description: "Description",
		Type:        TestCaseTypeBasic,
		Priority:    TestPriorityHigh,
		Category:    "category",
		Tags:        []string{"tag"},
		Enabled:     true,
	}

	assert.Equal(t, "test-case-id", tc.ID)
	assert.Equal(t, TestCaseTypeBasic, tc.Type)
	assert.Equal(t, TestPriorityHigh, tc.Priority)
}

func TestTestConfig_Structure(t *testing.T) {
	config := &TestConfig{
		Provider:    "openai",
		Model:       "gpt-4",
		Temperature: 0.7,
		MaxTokens:   1000,
		Timeout:     30 * time.Second,
		RetryCount:  3,
	}

	assert.Equal(t, "openai", config.Provider)
	assert.Equal(t, "gpt-4", config.Model)
	assert.Equal(t, 0.7, config.Temperature)
}

func TestTestInputs_Structure(t *testing.T) {
	inputs := &TestInputs{
		Prompt:        "Test prompt",
		SystemMessage: "System message",
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		Variables: map[string]interface{}{"var": "value"},
	}

	assert.Equal(t, "Test prompt", inputs.Prompt)
	assert.Len(t, inputs.Messages, 1)
}

func TestTestExpected_Structure(t *testing.T) {
	expected := &TestExpected{
		ResponsePattern: "pattern",
		Contains:        []string{"expected"},
		NotContains:     []string{"error"},
		MinLength:       10,
		MaxLength:       1000,
		ResponseTime:    10 * time.Second,
		SuccessRate:     0.95,
		ScoreThreshold:  0.8,
	}

	assert.Equal(t, "pattern", expected.ResponsePattern)
	assert.Len(t, expected.Contains, 1)
	assert.Equal(t, 0.95, expected.SuccessRate)
}

func TestTestResult_Structure(t *testing.T) {
	result := &TestResult{
		TestCaseID:   "case-id",
		TestCaseName: "Test Case",
		Status:       TestStatusPassed,
		Duration:     1 * time.Second,
		Response:     "Response text",
		Score:        0.95,
		Provider:     "openai",
		Model:        "gpt-4",
		Timestamp:    time.Now(),
	}

	assert.Equal(t, "case-id", result.TestCaseID)
	assert.Equal(t, TestStatusPassed, result.Status)
	assert.Equal(t, 0.95, result.Score)
}

func TestExecutionReport_Structure(t *testing.T) {
	report := &ExecutionReport{
		SuiteID:     "suite-id",
		SuiteName:   "Suite Name",
		StartTime:   time.Now(),
		EndTime:     time.Now(),
		Duration:    10 * time.Second,
		TestResults: []TestResult{},
		Summary: ExecutionSummary{
			TotalTests:  10,
			PassedTests: 8,
		},
	}

	assert.Equal(t, "suite-id", report.SuiteID)
	assert.Equal(t, 8, report.Summary.PassedTests)
}

func TestExecutionSummary_Structure(t *testing.T) {
	summary := &ExecutionSummary{
		TotalTests:    100,
		PassedTests:   80,
		FailedTests:   10,
		SkippedTests:  5,
		ErrorTests:    5,
		AvgScore:      0.85,
		AvgDuration:   2 * time.Second,
		P50Duration:   1 * time.Second,
		P95Duration:   5 * time.Second,
		TotalDuration: 200 * time.Second,
	}

	assert.Equal(t, 100, summary.TotalTests)
	assert.Equal(t, 80, summary.PassedTests)
	assert.Equal(t, 0.85, summary.AvgScore)
}

func TestMessage_Structure(t *testing.T) {
	msg := &Message{
		Role:    "user",
		Content: "Hello, world!",
	}

	assert.Equal(t, "user", msg.Role)
	assert.Equal(t, "Hello, world!", msg.Content)
}

func TestFileInput_Structure(t *testing.T) {
	file := &FileInput{
		Name:    "image.jpg",
		Content: []byte{0x89, 0x50, 0x4E, 0x47},
		URL:     "https://example.com/image.jpg",
		Type:    "image/jpeg",
	}

	assert.Equal(t, "image.jpg", file.Name)
	assert.Equal(t, "image/jpeg", file.Type)
}

func TestSuiteConfig_Structure(t *testing.T) {
	config := &SuiteConfig{
		ExecutionMode: ExecutionModeParallel,
		Parallelism:   10,
		Timeout:       300 * time.Second,
		RetryPolicy: RetryPolicy{
			MaxRetries:    3,
			InitialDelay:  1 * time.Second,
			BackoffFactor: 2.0,
		},
		Providers: []string{"openai", "anthropic"},
	}

	assert.Equal(t, ExecutionModeParallel, config.ExecutionMode)
	assert.Equal(t, 10, config.Parallelism)
}

func TestRetryPolicy_Structure(t *testing.T) {
	policy := &RetryPolicy{
		MaxRetries:    5,
		InitialDelay:  1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
	}

	assert.Equal(t, 5, policy.MaxRetries)
	assert.Equal(t, 2.0, policy.BackoffFactor)
}

func TestReportingConfig_Structure(t *testing.T) {
	config := &ReportingConfig{
		Format:        []string{"json", "html"},
		OutputDir:     "./reports",
		IncludeRaw:    true,
		Metrics:       []string{"latency", "success_rate"},
		CustomReports: map[string]interface{}{"custom": true},
	}

	assert.Len(t, config.Format, 2)
	assert.Equal(t, "./reports", config.OutputDir)
	assert.True(t, config.IncludeRaw)
}

func TestCustomValidator_Structure(t *testing.T) {
	validator := &CustomValidator{
		Name:   "custom_validator",
		Type:   "regex",
		Config: map[string]interface{}{"pattern": ".*"},
	}

	assert.Equal(t, "custom_validator", validator.Name)
	assert.Equal(t, "regex", validator.Type)
}

// ==================== Helper Function Tests ====================

func TestHelperFunctions(t *testing.T) {
	t.Run("testIntPtr", func(t *testing.T) {
		ptr := testIntPtr(42)
		require.NotNil(t, ptr)
		assert.Equal(t, 42, *ptr)
	})

	t.Run("testFloat64Ptr", func(t *testing.T) {
		ptr := testFloat64Ptr(0.5)
		require.NotNil(t, ptr)
		assert.Equal(t, 0.5, *ptr)
	})
}
