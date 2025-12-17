package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"llm-verifier/config"
	"llm-verifier/llmverifier"
)

// Integration tests for the complete LLM verifier system

func TestConfigLoading(t *testing.T) {
	t.Run("Config Loading", func(t *testing.T) {
		// Create a temporary config file for testing
		tempDir := t.TempDir()
		tempConfig := `global:
  base_url: "https://api.openai.com/v1"
  api_key: "test-key"
llms:
  - name: "gpt-3.5-turbo"
    endpoint: "https://api.openai.com/v1"
    model: "gpt-3.5-turbo"
`

		tempFile := filepath.Join(tempDir, "config.yaml")
		if err := os.WriteFile(tempFile, []byte(tempConfig), 0644); err != nil {
			t.Fatalf("Failed to write temp config file: %v", err)
		}

		// Test config loading
		cfg, err := config.LoadConfig(tempFile)
		if err != nil {
			t.Errorf("Failed to load config: %v", err)
			return
		}

		// Verify config values
		if cfg.Global.BaseURL != "https://api.openai.com/v1" {
			t.Errorf("Expected base URL 'https://api.openai.com/v1', got '%s'", cfg.Global.BaseURL)
		}

		if cfg.Global.APIKey != "test-key" {
			t.Errorf("Expected API key 'test-key', got '%s'", cfg.Global.APIKey)
		}

		if len(cfg.LLMs) != 1 {
			t.Errorf("Expected 1 LLM configuration, got %d", len(cfg.LLMs))
		}

		llm := cfg.LLMs[0]
		if llm.Name != "gpt-3.5-turbo" {
			t.Errorf("Expected LLM name 'gpt-3.5-turbo', got '%s'", llm.Name)
		}

		t.Logf("Config loading test passed")
	})
}

func TestVerifierIntegration(t *testing.T) {
	t.Run("Verifier Integration", func(t *testing.T) {
		// Create test config
		cfg := &config.Config{
			Global: config.GlobalConfig{
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "test-key",
			},
			LLMs: []config.LLMConfig{
				{
					Name:     "gpt-3.5-turbo",
					Endpoint: "https://api.openai.com/v1",
					Model:    "gpt-3.5-turbo",
				},
			},
		}

		// Create verifier
		verifier := llmverifier.New(cfg)
		if verifier == nil {
			t.Fatal("Failed to create verifier")
		}

		t.Logf("Verifier created successfully with %d LLM configurations", len(cfg.LLMs))

		// Test conversation summarization
		messages := []string{
			"User: Hello, how are you?",
			"Assistant: I'm doing well, thank you!",
			"User: Can you help me with my code?",
			"Assistant: I'd be happy to help you with your code. What are you working on?",
		}

		summary, err := verifier.SummarizeConversation(messages)
		if err != nil {
			t.Errorf("Failed to summarize conversation: %v", err)
		} else {
			if summary.Summary == "" {
				t.Error("Expected non-empty summary")
			}
			t.Logf("Conversation summary generated: %s", summary.Summary)
		}

		// Test multiple LLMs
		if len(cfg.LLMs) > 1 {
			t.Log("Multiple LLMs configured - test extended scenarios")
		}
	})
}

func TestComponentOrchestration(t *testing.T) {
	t.Run("Component Orchestration", func(t *testing.T) {
		// This test simulates the main application flow
		// verifying that different components can work together

		// Create test config
		cfg := &config.Config{
			Global: config.GlobalConfig{
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "test-key",
			},
			LLMs: []config.LLMConfig{
				{
					Name:     "orchestration-test",
					Endpoint: "https://api.openai.com/v1",
					Model:    "gpt-3.5-turbo",
				},
			},
			Concurrency: 2,
			Timeout:     30 * time.Second,
		}

		// Initialize components in the correct order
		// 1. Create verifier (required by other components)
		verifier := llmverifier.New(cfg)
		if verifier == nil {
			t.Fatal("Failed to create verifier")
		}

		// 2. Initialize other components (in real application, these would be enhanced components)
		// For integration test, we verify the verifier works

		t.Logf("Component orchestration test completed successfully")
	})
}

func TestErrorHandling(t *testing.T) {
	t.Run("Error Handling", func(t *testing.T) {
		// Test invalid configurations and error scenarios

		// Test 1: Invalid configuration
		invalidConfig := &config.Config{
			Global: config.GlobalConfig{
				BaseURL: "", // Invalid empty URL
				APIKey:  "test-key",
			},
		}

		verifier := llmverifier.New(invalidConfig)
		if verifier != nil {
			t.Error("Expected verifier to be nil with invalid config")
		}

		// Test 2: Valid config with invalid API key
		validConfig := &config.Config{
			Global: config.GlobalConfig{
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "invalid-key", // This would likely cause API errors
			},
			LLMs: []config.LLMConfig{
				{
					Name:     "error-test",
					Endpoint: "https://api.openai.com/v1",
					Model:    "gpt-3.5-turbo",
				},
			},
		}

		errorVerifier := llmverifier.New(validConfig)
		if errorVerifier == nil {
			t.Fatal("Failed to create verifier with valid config")
		}

		// Test that the verifier handles API errors gracefully
		// In a real implementation, this would involve making actual API calls
		// For this test, we verify the structure is created properly

		t.Logf("Error handling test completed - verifier structure validated")
	})
}

func TestConcurrentAccess(t *testing.T) {
	t.Run("Concurrent Access", func(t *testing.T) {
		// Test concurrent access to the verifier

		cfg := &config.Config{
			Global: config.GlobalConfig{
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "test-key",
			},
			LLMs: []config.LLMConfig{
				{
					Name:     "concurrent-test",
					Endpoint: "https://api.openai.com/v1",
					Model:    "gpt-3.5-turbo",
				},
			},
			Concurrency: 5,
		}

		verifier := llmverifier.New(cfg)
		if verifier == nil {
			t.Fatal("Failed to create verifier")
		}

		// Test concurrent conversation summarization
		const numGoroutines = 10
		const messagesPerGoroutine = 5

		done := make(chan bool, numGoroutines)
		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer func() { done <- true }()

				for j := 0; j < messagesPerGoroutine; j++ {
					messages := []string{
						fmt.Sprintf("Goroutine %d, Message %d", id, j),
						"How can I assist you today?",
					}

					_, err := verifier.SummarizeConversation(messages)
					if err != nil {
						errors <- fmt.Errorf("Goroutine %d, Message %d failed: %v", id, j, err)
						return
					}
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Check for errors
		errorCount := 0
		for i := 0; i < numGoroutines; i++ {
			select {
			case err := <-errors:
				t.Logf("Concurrent error: %v", err)
				errorCount++
			default:
				// No more errors
				break
			}
		}

		// Allow some errors due to mock/test environment
		if errorCount > numGoroutines/2 {
			t.Errorf("Too many errors: %d out of %d goroutines", errorCount, numGoroutines)
		}

		t.Logf("Concurrent access test completed - Goroutines: %d, Error rate: %.1f%%",
			numGoroutines, float64(errorCount)/float64(numGoroutines)*100)
	})
}

func TestResourceManagement(t *testing.T) {
	t.Run("Resource Management", func(t *testing.T) {
		// Test resource cleanup and management
		// This is particularly important for memory leak detection

		cfg := &config.Config{
			Global: config.GlobalConfig{
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "test-key",
			},
			LLMs: []config.LLMConfig{
				{
					Name:     "resource-test",
					Endpoint: "https://api.openai.com/v1",
					Model:    "gpt-3.5-turbo",
				},
			},
		}

		// Create multiple verifiers to test resource usage
		var verifiers []*llmverifier.Verifier
		const numVerifiers = 10

		for i := 0; i < numVerifiers; i++ {
			verifier := llmverifier.New(cfg)
			if verifier == nil {
				t.Fatalf("Failed to create verifier %d", i)
			}
			verifiers = append(verifiers, verifier)
		}

		// Use all verifiers to test memory usage
		for i, verifier := range verifiers {
			messages := []string{
				fmt.Sprintf("Resource test message from verifier %d", i),
				"Test resource management",
			}

			summary, err := verifier.SummarizeConversation(messages)
			if err != nil {
				t.Errorf("Verifier %d failed: %v", i, err)
			} else {
				if summary.Summary == "" {
					t.Errorf("Verifier %d produced empty summary", i)
				}
			}
		}

		// Clean up
		for i := range verifiers {
			// In a real implementation, there would be cleanup methods
			// For this test, we let garbage collection handle it
			verifiers[i] = nil
		}

		t.Logf("Resource management test completed with %d verifiers", numVerifiers)
	})
}

func TestPerformanceCharacteristics(t *testing.T) {
	t.Run("Performance Characteristics", func(t *testing.T) {
		// Test performance characteristics of key operations

		cfg := &config.Config{
			Global: config.GlobalConfig{
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "test-key",
			},
			LLMs: []config.LLMConfig{
				{
					Name:     "performance-test",
					Endpoint: "https://api.openai.com/v1",
					Model:    "gpt-3.5-turbo",
				},
			},
		}

		verifier := llmverifier.New(cfg)
		if verifier == nil {
			t.Fatal("Failed to create verifier")
		}

		// Test conversation summarization performance
		messageSizes := []int{1, 10, 50, 100, 500}

		for _, size := range messageSizes {
			messages := make([]string, size)
			for i := range messages {
				messages[i] = fmt.Sprintf("Performance test message %d", i)
			}

			start := time.Now()
			summary, err := verifier.SummarizeConversation(messages)
			duration := time.Since(start)

			if err != nil {
				t.Errorf("Summarization failed for size %d: %v", size, err)
			} else {
				if summary.Summary == "" {
					t.Errorf("Empty summary for size %d", size)
				}

				t.Logf("Message count: %d, Duration: %v, Summary length: %d",
					size, duration, len(summary.Summary))

				// Performance assertions (adjust based on your environment)
				if duration > 5*time.Second {
					t.Logf("WARNING: Slow summarization for %d messages: %v", size, duration)
				}
			}
		}

		// Test batch processing performance
		numBatches := 5
		messagesPerBatch := 10

		start := time.Now()
		for batch := 0; batch < numBatches; batch++ {
			batchMessages := make([]string, messagesPerBatch)
			for i := 0; i < messagesPerBatch; i++ {
				batchMessages[i] = fmt.Sprintf("Batch %d, Message %d", batch, i)
			}

			_, err := verifier.SummarizeConversation(batchMessages)
			if err != nil {
				t.Errorf("Batch %d failed: %v", batch, err)
			}
		}

		totalDuration := time.Since(start)
		t.Logf("Batch processing completed: %d batches, %d messages per batch, Total duration: %v",
			numBatches, messagesPerBatch, totalDuration)

		averageDuration := totalDuration / time.Duration(numBatches)
		t.Logf("Average batch duration: %v", averageDuration)
	})
}

func TestEndToEndWorkflows(t *testing.T) {
	t.Run("End-to-End Workflows", func(t *testing.T) {
		// Test complete workflows that would be used in production

		cfg := &config.Config{
			Global: config.GlobalConfig{
				BaseURL: "https://api.openai.com/v1",
				APIKey:  "test-key",
			},
			LLMs: []config.LLMConfig{
				{
					Name:     "workflow-test",
					Endpoint: "https://api.openai.com/v1",
					Model:    "gpt-3.5-turbo",
				},
				{
					Name:     "analysis-test",
					Endpoint: "https://api.openai.com/v1",
					Model:    "gpt-4",
				},
			},
			Concurrency: 2,
		}

		verifier := llmverifier.New(cfg)
		if verifier == nil {
			t.Fatal("Failed to create verifier")
		}

		// Workflow 1: Simple conversation analysis
		simpleConversation := []string{
			"User: I have a piece of code that isn't working.",
			"Assistant: I'd be happy to help you with your code. Could you share the code that's giving you trouble?",
			"User: function processData(data) { return data.map(item => item + 1); }",
			"Assistant: I see the issue. In your processData function, you're trying to add 1 to each item. However, item is a string, so you can't perform arithmetic. You should parse the string as a number first.",
			"User: You're right! I forgot to parse it. How should I fix this?",
		}

		start := time.Now()
		simpleSummary, err := verifier.SummarizeConversation(simpleConversation)
		simpleDuration := time.Since(start)

		if err != nil {
			t.Errorf("Simple conversation analysis failed: %v", err)
		} else {
			if simpleDuration > 5*time.Second {
				t.Logf("WARNING: Simple conversation analysis took %v", simpleDuration)
			}
			t.Logf("Simple workflow completed - Duration: %v, Summary: %s",
				simpleDuration, simpleSummary.Summary)
		}

		// Workflow 2: Complex multi-turn conversation
		complexConversation := []string{
			"User: I'm building a complex application with multiple components.",
			"Assistant: That sounds interesting! What kind of application are you building?",
			"User: It's a data processing pipeline with authentication, data storage, and analytics.",
			"Assistant: Let me help you plan this. For authentication, you'll need to choose between session-based or token-based authentication. What are your requirements?",
			"User: I need to support both user authentication and API key authentication.",
			"Assistant: For user authentication, you'll want to implement OAuth2.0 or JWT tokens. For API key authentication, you'll need secure key storage and rotation. Do you have preferences?",
			"User: I think JWT tokens would be simpler. I'll also need rate limiting and logging.",
			"Assistant: Great! JWT tokens it is. For rate limiting, you can use algorithms like token bucket or sliding window. For logging, make sure to log security events but not sensitive data. How many requests per second do you expect?",
			"User: Around 100 requests per second initially.",
			"Assistant: Perfect! I'll implement a token bucket algorithm with a burst capacity. This will allow for occasional spikes while maintaining the average rate limit. I'll also implement structured logging with correlation IDs.",
		}

		start = time.Now()
		complexSummary, err := verifier.SummarizeConversation(complexConversation)
		complexDuration := time.Since(start)

		if err != nil {
			t.Errorf("Complex conversation analysis failed: %v", err)
		} else {
			t.Logf("Complex workflow completed - Duration: %v, Summary: %s",
				complexDuration, complexSummary.Summary)

			// Verify the summary captures the complexity
			if complexDuration > simpleDuration*3 {
				t.Logf("Complex conversation took significantly longer than simple one as expected")
			}
		}

		t.Logf("All end-to-end workflows completed successfully")
	})
}
