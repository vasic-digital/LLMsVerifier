package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"digital.vasic.llmsverifier/testsuite"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "create":
		handleCreate()
	case "list":
		handleList()
	case "run":
		handleRun()
	case "export":
		handleExport()
	case "import":
		handleImport()
	case "templates":
		handleTemplates()
	default:
		fmt.Println(trData("llmsverifier_testsuite_unknown_command", map[string]any{"command": command}))
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(tr("llmsverifier_testsuite_title"))
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  testsuite create <name> <description> [options]  Create a new test suite")
	fmt.Println("  testsuite list                                    List all test suites")
	fmt.Println("  testsuite run <suite-id>                          Run a test suite")
	fmt.Println("  testsuite export <suite-id> [file]               Export a test suite")
	fmt.Println("  testsuite import <file>                          Import a test suite")
	fmt.Println("  testsuite templates                              Create template test suites")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  testsuite create \"My Tests\" \"Custom test suite\" --providers openai,anthropic")
	fmt.Println("  testsuite run suite-123 --parallel 5")
	fmt.Println("  testsuite export suite-123 tests.json")
}

func handleCreate() {
	if len(os.Args) < 4 {
		fmt.Println(tr("llmsverifier_testsuite_err_create_args"))
		os.Exit(1)
	}

	name := os.Args[2]
	description := os.Args[3]

	// Parse additional options
	providers := []string{"openai", "anthropic"} // default
	parallelism := 5
	executionMode := "parallel"

	for i := 4; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch {
		case strings.HasPrefix(arg, "--providers="):
			providers = strings.Split(strings.TrimPrefix(arg, "--providers="), ",")
		case strings.HasPrefix(arg, "--parallel="):
			fmt.Sscanf(strings.TrimPrefix(arg, "--parallel="), "%d", &parallelism)
		case strings.HasPrefix(arg, "--mode="):
			executionMode = strings.TrimPrefix(arg, "--mode=")
		}
	}

	// Create test suite
	builder := testsuite.NewTestSuiteBuilder(name, description).
		WithAuthor("CLI User").
		WithProviders(providers...)

	if executionMode == "sequential" {
		builder.WithExecutionMode(testsuite.ExecutionModeSequential)
	} else {
		builder.WithExecutionMode(testsuite.ExecutionModeParallel).WithParallelism(parallelism)
	}

	// Add some default test cases
	builder.
		AddBasicTestCase("Greeting Test", "Say hello in a friendly way", []string{"hello", "hi"}).
		AddBasicTestCase("Math Test", "What is 2 + 2?", []string{"4"}).
		AddBasicTestCase("Code Test", "Write a simple Python function to add two numbers", []string{"def", "return"})

	suite := builder.Build()

	// Save to file (in-memory for demo)
	_ = suite // In real implementation, save to persistent storage

	fmt.Println(trData("llmsverifier_testsuite_created", map[string]any{"name": suite.Name, "id": suite.ID}))
	fmt.Printf("Providers: %v\n", suite.Configuration.Providers)
	fmt.Println(trData("llmsverifier_testsuite_test_cases_count", map[string]any{"count": len(suite.TestCases)}))
}

func handleList() {
	// In a real implementation, this would load from persistent storage
	// For demo, show template suites
	suites := testsuite.CreateTemplateSuites()

	fmt.Println(tr("llmsverifier_testsuite_available_heading"))
	fmt.Println("======================")

	for _, suite := range suites {
		fmt.Printf("ID: %s\n", suite.ID)
		fmt.Printf("Name: %s\n", suite.Name)
		fmt.Printf("Description: %s\n", suite.Description)
		fmt.Println(trData("llmsverifier_testsuite_test_cases_count", map[string]any{"count": len(suite.TestCases)}))
		fmt.Printf("Tags: %v\n", suite.Tags)
		fmt.Println()
	}
}

func handleRun() {
	if len(os.Args) < 3 {
		fmt.Println(tr("llmsverifier_testsuite_err_run_args"))
		os.Exit(1)
	}

	suiteID := os.Args[2]

	// In a real implementation, load the suite by ID
	// For demo, use template suites
	suites := testsuite.CreateTemplateSuites()

	var targetSuite *testsuite.TestSuite
	for _, suite := range suites {
		if suite.ID == suiteID || strings.Contains(suite.Name, suiteID) {
			targetSuite = suite
			break
		}
	}

	if targetSuite == nil {
		log.Fatalf("Test suite not found: %s", suiteID)
	}

	fmt.Println(trData("llmsverifier_testsuite_running", map[string]any{"name": targetSuite.Name}))
	fmt.Printf("Description: %s\n", targetSuite.Description)
	fmt.Println(trData("llmsverifier_testsuite_test_cases_count", map[string]any{"count": len(targetSuite.TestCases)}))
	fmt.Println()

	// Create executor and run tests
	executor := testsuite.NewTestSuiteExecutor(targetSuite)
	report, err := executor.Execute(nil) // Using nil context for demo
	if err != nil {
		log.Fatalf("Failed to execute test suite: %v", err)
	}

	// Print results
	fmt.Println(tr("llmsverifier_testsuite_execution_results_heading"))
	fmt.Println("==================")
	fmt.Println(trData("llmsverifier_testsuite_total_tests", map[string]any{"count": report.Summary.TotalTests}))
	fmt.Printf("Passed: %d\n", report.Summary.PassedTests)
	fmt.Printf("Failed: %d\n", report.Summary.FailedTests)
	fmt.Println(trData("llmsverifier_testsuite_average_score", map[string]any{"score": fmt.Sprintf("%.2f", report.Summary.AvgScore)}))
	fmt.Println(trData("llmsverifier_testsuite_average_duration", map[string]any{"duration": report.Summary.AvgDuration}))
	fmt.Println(trData("llmsverifier_testsuite_total_duration", map[string]any{"duration": report.Summary.TotalDuration}))

	if report.Summary.P95Duration > 0 {
		fmt.Println(trData("llmsverifier_testsuite_p95_duration", map[string]any{"duration": report.Summary.P95Duration}))
	}

	fmt.Println()
	fmt.Println(tr("llmsverifier_testsuite_individual_results_heading"))
	fmt.Println("========================")

	for _, result := range report.TestResults {
		status := "✓"
		if result.Status != "passed" {
			status = "✗"
		}
		fmt.Printf("%s %s (%.2f) - %v\n", status, result.TestCaseName, result.Score, result.Duration)
	}
}

func handleExport() {
	if len(os.Args) < 3 {
		fmt.Println(tr("llmsverifier_testsuite_err_export_args"))
		os.Exit(1)
	}

	suiteID := os.Args[2]
	outputFile := fmt.Sprintf("%s.json", suiteID)

	if len(os.Args) >= 4 {
		outputFile = os.Args[3]
	}

	// In a real implementation, load the suite
	// For demo, use template suites
	suites := testsuite.CreateTemplateSuites()

	var targetSuite *testsuite.TestSuite
	for _, suite := range suites {
		if suite.ID == suiteID || strings.Contains(suite.Name, suiteID) {
			targetSuite = suite
			break
		}
	}

	if targetSuite == nil {
		log.Fatalf("Test suite not found: %s", suiteID)
	}

	// Export to JSON
	data, err := json.MarshalIndent(targetSuite, "", "  ")
	if err != nil {
		log.Fatalf("Failed to export test suite: %v", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(outputFile)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory: %v", err)
		}
	}

	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		log.Fatalf("Failed to write file: %v", err)
	}

	fmt.Println(trData("llmsverifier_testsuite_exported_to", map[string]any{"file": outputFile}))
}

func handleImport() {
	if len(os.Args) < 3 {
		fmt.Println(tr("llmsverifier_testsuite_err_import_args"))
		os.Exit(1)
	}

	filePath := os.Args[2]

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
	}

	// Import suite
	manager := testsuite.NewTestSuiteManager()
	suite, err := manager.ImportSuite(data)
	if err != nil {
		log.Fatalf("Failed to import test suite: %v", err)
	}

	fmt.Println(trData("llmsverifier_testsuite_imported", map[string]any{"name": suite.Name, "id": suite.ID}))
}

func handleTemplates() {
	fmt.Println(tr("llmsverifier_testsuite_creating_templates"))

	// Create template suites
	suites := testsuite.CreateTemplateSuites()

	manager := testsuite.NewTestSuiteManager()

	// Save templates
	for _, suite := range suites {
		if err := manager.SaveSuite(suite); err != nil {
			log.Printf("Failed to save template %s: %v", suite.Name, err)
			continue
		}
		fmt.Println(trData("llmsverifier_testsuite_created_template", map[string]any{"name": suite.Name, "count": len(suite.TestCases)}))
	}

	fmt.Printf("\nCreated %d template test suites\n", len(suites))
	fmt.Println(tr("llmsverifier_testsuite_use_list_hint"))
}
