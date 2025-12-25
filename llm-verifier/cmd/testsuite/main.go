package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"llm-verifier/testsuite"
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
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("LLM Test Suite Builder")
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
		fmt.Println("Error: create command requires name and description")
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

	fmt.Printf("Created test suite: %s (ID: %s)\n", suite.Name, suite.ID)
	fmt.Printf("Providers: %v\n", suite.Configuration.Providers)
	fmt.Printf("Test cases: %d\n", len(suite.TestCases))
}

func handleList() {
	// In a real implementation, this would load from persistent storage
	// For demo, show template suites
	suites := testsuite.CreateTemplateSuites()

	fmt.Println("Available Test Suites:")
	fmt.Println("======================")

	for _, suite := range suites {
		fmt.Printf("ID: %s\n", suite.ID)
		fmt.Printf("Name: %s\n", suite.Name)
		fmt.Printf("Description: %s\n", suite.Description)
		fmt.Printf("Test Cases: %d\n", len(suite.TestCases))
		fmt.Printf("Tags: %v\n", suite.Tags)
		fmt.Println()
	}
}

func handleRun() {
	if len(os.Args) < 3 {
		fmt.Println("Error: run command requires suite ID")
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

	fmt.Printf("Running test suite: %s\n", targetSuite.Name)
	fmt.Printf("Description: %s\n", targetSuite.Description)
	fmt.Printf("Test cases: %d\n", len(targetSuite.TestCases))
	fmt.Println()

	// Create executor and run tests
	executor := testsuite.NewTestSuiteExecutor(targetSuite)
	report, err := executor.Execute(nil) // Using nil context for demo
	if err != nil {
		log.Fatalf("Failed to execute test suite: %v", err)
	}

	// Print results
	fmt.Println("Execution Results:")
	fmt.Println("==================")
	fmt.Printf("Total Tests: %d\n", report.Summary.TotalTests)
	fmt.Printf("Passed: %d\n", report.Summary.PassedTests)
	fmt.Printf("Failed: %d\n", report.Summary.FailedTests)
	fmt.Printf("Average Score: %.2f\n", report.Summary.AvgScore)
	fmt.Printf("Average Duration: %v\n", report.Summary.AvgDuration)
	fmt.Printf("Total Duration: %v\n", report.Summary.TotalDuration)

	if report.Summary.P95Duration > 0 {
		fmt.Printf("95th Percentile: %v\n", report.Summary.P95Duration)
	}

	fmt.Println()
	fmt.Println("Individual Test Results:")
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
		fmt.Println("Error: export command requires suite ID")
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

	fmt.Printf("Test suite exported to: %s\n", outputFile)
}

func handleImport() {
	if len(os.Args) < 3 {
		fmt.Println("Error: import command requires file path")
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

	fmt.Printf("Test suite imported: %s (ID: %s)\n", suite.Name, suite.ID)
}

func handleTemplates() {
	fmt.Println("Creating template test suites...")

	// Create template suites
	suites := testsuite.CreateTemplateSuites()

	manager := testsuite.NewTestSuiteManager()

	// Save templates
	for _, suite := range suites {
		if err := manager.SaveSuite(suite); err != nil {
			log.Printf("Failed to save template %s: %v", suite.Name, err)
			continue
		}
		fmt.Printf("Created template: %s (%d test cases)\n", suite.Name, len(suite.TestCases))
	}

	fmt.Printf("\nCreated %d template test suites\n", len(suites))
	fmt.Println("Use 'testsuite list' to see available suites")
}
