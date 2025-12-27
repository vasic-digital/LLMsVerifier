package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/llmverifier/llmverifier"
	"github.com/llmverifier/llmverifier/config"
	"github.com/llmverifier/llmverifier/providers"
)

// ACPClient implements ACP-specific interactions with LLM providers
type ACPClient struct {
	baseClient *llmverifier.LLMClient
	provider   string
	config     *providers.ProviderConfig
	httpClient *http.Client
}

// NewACPClient creates a new ACP client for a provider
func NewACPClient(providerConfig *providers.ProviderConfig, apiKey string) (*ACPClient, error) {
	httpClient := &http.Client{
		Timeout: providerConfig.Timeouts.RequestTimeout,
	}

	// Create base LLM client
	baseClient := &llmverifier.LLMClient{
		Endpoint: providerConfig.Endpoint,
		APIKey:   apiKey,
		Client:   httpClient,
	}

	return &ACPClient{
		baseClient: baseClient,
		provider:   providerConfig.Name,
		config:     providerConfig,
		httpClient: httpClient,
	}, nil
}

// TestACPAllCapabilities runs all ACP capability tests
func (c *ACPClient) TestACPAllCapabilities(ctx context.Context, modelName string) (*ACPTestResult, error) {
	result := &ACPTestResult{
		ModelID:      modelName,
		Provider:     c.provider,
		Timestamp:    time.Now(),
		Capabilities: make(map[string]ACPCapabilityResult),
	}

	// Test 1: JSON-RPC Protocol Comprehension
	if jsonRPCResult := c.testJSONRPCCompliance(ctx, modelName); jsonRPCResult.Supported {
		result.Capabilities["jsonrpc_compliance"] = jsonRPCResult
	}

	// Test 2: Tool Calling Capability
	if toolResult := c.testToolCalling(ctx, modelName); toolResult.Supported {
		result.Capabilities["tool_calling"] = toolResult
	}

	// Test 3: Context Management
	if contextResult := c.testContextManagement(ctx, modelName); contextResult.Supported {
		result.Capabilities["context_management"] = contextResult
	}

	// Test 4: Code Assistance
	if codeResult := c.testCodeAssistance(ctx, modelName); codeResult.Supported {
		result.Capabilities["code_assistance"] = codeResult
	}

	// Test 5: Error Detection
	if errorResult := c.testErrorDetection(ctx, modelName); errorResult.Supported {
		result.Capabilities["error_detection"] = errorResult
	}

	// Calculate overall support and score
	result.calculateOverallResult()

	return result, nil
}

// ACPTestResult represents the result of ACP capability testing
type ACPTestResult struct {
	ModelID      string                         `json:"model_id"`
	Provider     string                         `json:"provider"`
	ACPSupported bool                           `json:"acp_supported"`
	ACPScore     float64                        `json:"acp_score"`
	Capabilities map[string]ACPCapabilityResult `json:"capabilities"`
	Timestamp    time.Time                      `json:"timestamp"`
	Duration     time.Duration                  `json:"duration"`
}

// ACPCapabilityResult represents the result of a specific ACP capability test
type ACPCapabilityResult struct {
	Name        string        `json:"name"`
	Supported   bool          `json:"supported"`
	Score       float64       `json:"score"`
	Details     string        `json:"details"`
	Duration    time.Duration `json:"duration"`
	Error       string        `json:"error,omitempty"`
}

// calculateOverallResult calculates the overall ACP support and score
func (r *ACPTestResult) calculateOverallResult() {
	var totalScore float64
	supportedCount := 0

	for _, capability := range r.Capabilities {
		totalScore += capability.Score
		if capability.Supported {
			supportedCount++
		}
	}

	r.ACPScore = totalScore / float64(len(r.Capabilities))
	r.ACPSupported = supportedCount >= 3 // Require at least 3 capabilities
}

// testJSONRPCCompliance tests JSON-RPC protocol understanding
func (c *ACPClient) testJSONRPCCompliance(ctx context.Context, modelName string) ACPCapabilityResult {
	start := time.Now()

	result := ACPCapabilityResult{
		Name:      "JSON-RPC Protocol Comprehension",
		Supported: false,
		Score:     0.0,
	}

	// Create JSON-RPC test message
	jsonRPCMessage := `{"jsonrpc":"2.0","method":"textDocument/completion","params":{"textDocument":{"uri":"file:///test.py"},"position":{"line":0,"character":10}},"id":1}`

	request := llmverifier.ChatCompletionRequest{
		Model: modelName,
		Messages: []llmverifier.Message{
			{
				Role:    "user",
				Content: fmt.Sprintf("You are an ACP-compatible AI coding agent. Please respond to this JSON-RPC request:\n%s\n\nWhat would be an appropriate response for a code completion request? Please provide a valid JSON-RPC response.", jsonRPCMessage),
			},
		},
	}

	response, err := c.sendRequest(ctx, modelName, request)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	// Evaluate JSON-RPC response
	responseText := strings.ToLower(response.Choices[0].Message.Content)
	if strings.Contains(responseText, "jsonrpc") &&
		(strings.Contains(responseText, "2.0") || strings.Contains(responseText, "2")) &&
		(strings.Contains(responseText, "result") || strings.Contains(responseText, "items")) {
		result.Supported = true
		result.Score = 0.9
		result.Details = "Model correctly interpreted JSON-RPC and provided valid response structure"
	} else if strings.Contains(responseText, "completion") {
		result.Supported = true
		result.Score = 0.7
		result.Details = "Model showed basic JSON-RPC understanding but response format could be improved"
	} else {
		result.Score = 0.3
		result.Details = "Model did not demonstrate clear JSON-RPC comprehension"
	}

	result.Duration = time.Since(start)
	return result
}

// testToolCalling tests tool calling capabilities
func (c *ACPClient) testToolCalling(ctx context.Context, modelName string) ACPCapabilityResult {
	start := time.Now()

	result := ACPCapabilityResult{
		Name:      "Tool Calling Capability",
		Supported: false,
		Score:     0.0,
	}

	request := llmverifier.ChatCompletionRequest{
		Model: modelName,
		Messages: []llmverifier.Message{
			{
				Role: "user",
				Content: `As an ACP agent, you have access to tools like "file_read", "file_write", and "execute_command". 
Please demonstrate how you would call the "file_read" tool to read the content of a Python file named "main.py" 
and then suggest improvements based on the content.`,
			},
		},
	}

	response, err := c.sendRequest(ctx, modelName, request)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	// Evaluate tool calling response
	responseText := strings.ToLower(response.Choices[0].Message.Content)
	if strings.Contains(responseText, "file_read") &&
		(strings.Contains(responseText, "tool") || strings.Contains(responseText, "function")) {
		result.Supported = true
		result.Score = 0.85
		result.Details = "Model demonstrated proper tool calling understanding and usage"
	} else if strings.Contains(responseText, "tool") || strings.Contains(responseText, "function") {
		result.Supported = true
		result.Score = 0.6
		result.Details = "Model showed basic tool awareness but could be more specific"
	} else {
		result.Score = 0.2
		result.Details = "Model did not demonstrate clear tool calling capability"
	}

	result.Duration = time.Since(start)
	return result
}

// testContextManagement tests context management across conversation turns
func (c *ACPClient) testContextManagement(ctx context.Context, modelName string) ACPCapabilityResult {
	start := time.Now()

	result := ACPCapabilityResult{
		Name:      "Context Management",
		Supported: false,
		Score:     0.0,
	}

	// Multi-turn conversation test
	conversation := []llmverifier.Message{
		{
			Role: "user",
			Content: `I'm working on a Python project with the following structure: src/main.py, tests/test_main.py, requirements.txt. The main.py file contains a Flask web application. Remember this project structure and context.`,
		},
		{
			Role:    "assistant",
			Content: `I've noted your Python project structure and will remember it for our conversation.`,
		},
		{
			Role:    "user",
			Content: `Based on this project structure, where should I add a new utility module for database operations, and what would be the appropriate import statement in my Flask app?`,
		},
	}

	request := llmverifier.ChatCompletionRequest{
		Model:    modelName,
		Messages: conversation,
	}

	response, err := c.sendRequest(ctx, modelName, request)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	// Evaluate context retention
	responseText := strings.ToLower(response.Choices[0].Message.Content)
	if strings.Contains(responseText, "src") && 
		(strings.Contains(responseText, "utility") || strings.Contains(responseText, "utils")) &&
		strings.Contains(responseText, "import") {
		result.Supported = true
		result.Score = 0.85
		result.Details = "Model successfully maintained context across conversation turns"
	} else if strings.Contains(responseText, "src") || strings.Contains(responseText, "import") {
		result.Supported = true
		result.Score = 0.6
		result.Details = "Model showed some context awareness but response could be more specific"
	} else {
		result.Score = 0.3
		result.Details = "Model did not demonstrate strong context management capabilities"
	}

	result.Duration = time.Since(start)
	return result
}

// testCodeAssistance tests code generation and assistance capabilities
func (c *ACPClient) testCodeAssistance(ctx context.Context, modelName string) ACPCapabilityResult {
	start := time.Now()

	result := ACPCapabilityResult{
		Name:      "Code Assistance",
		Supported: false,
		Score:     0.0,
	}

	request := llmverifier.ChatCompletionRequest{
		Model: modelName,
		Messages: []llmverifier.Message{
			{
				Role: "user",
				Content: `As an ACP coding agent, help me write a Python function that:
1. Takes a list of user dictionaries (each with 'name' and 'email' keys)
2. Validates that all emails are in proper format
3. Returns a list of valid users
4. Includes proper error handling
5. Has type hints and docstring

Please provide the complete implementation.`,
			},
		},
	}

	response, err := c.sendRequest(ctx, modelName, request)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	// Evaluate code generation quality
	responseText := response.Choices[0].Message.Content
	qualityScore := evaluateCodeQuality(responseText)

	result.Supported = qualityScore > 0.6
	result.Score = qualityScore
	result.Details = fmt.Sprintf("Code generation quality score: %.2f", qualityScore)
	result.Duration = time.Since(start)
	return result
}

// testErrorDetection tests error detection and diagnostic capabilities
func (c *ACPClient) testErrorDetection(ctx context.Context, modelName string) ACPCapabilityResult {
	start := time.Now()

	result := ACPCapabilityResult{
		Name:      "Error Detection",
		Supported: false,
		Score:     0.0,
	}

	request := llmverifier.ChatCompletionRequest{
		Model: modelName,
		Messages: []llmverifier.Message{
			{
				Role: "user",
				Content: `As an ACP agent, analyze this Python code and provide diagnostic information:
def process_user_data(users):
    valid_users = []
    for user in users:
        if user['email'].contains('@'):
            valid_users.append(user)
    return valid_users

result = process_user_data([{'name': 'John', 'email': 'john@example.com'}, {'name': 'Jane'}])

What errors or issues do you detect? Provide specific line numbers and suggestions.`,
			},
		},
	}

	response, err := c.sendRequest(ctx, modelName, request)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	// Evaluate error detection capability
	responseText := strings.ToLower(response.Choices[0].Message.Content)
	detectionScore := evaluateErrorDetection(responseText)

	result.Supported = detectionScore > 0.5
	result.Score = detectionScore
	result.Details = fmt.Sprintf("Error detection score: %.2f", detectionScore)
	result.Duration = time.Since(start)
	return result
}

// sendRequest sends a request to the LLM provider
func (c *ACPClient) sendRequest(ctx context.Context, modelName string, request llmverifier.ChatCompletionRequest) (*llmverifier.ChatCompletionResponse, error) {
	// Convert to provider-specific format
	providerRequest := c.convertToProviderFormat(request)

	jsonData, err := json.Marshal(providerRequest)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.config.Endpoint+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set authentication
	switch c.config.AuthType {
	case "bearer":
		if c.baseClient.APIKey != "" {
			req.Header.Set("Authorization", "Bearer "+c.baseClient.APIKey)
		}
	case "api_key":
		if c.baseClient.APIKey != "" {
			req.Header.Set("X-API-Key", c.baseClient.APIKey)
		}
	}

	req.Header.Set("Content-Type", "application/json")

	// Add provider-specific headers
	c.addProviderHeaders(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var providerResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&providerResponse); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return c.convertFromProviderFormat(providerResponse)
}

// convertToProviderFormat converts generic request to provider-specific format
func (c *ACPClient) convertToProviderFormat(request llmverifier.ChatCompletionRequest) map[string]interface{} {
	result := map[string]interface{}{
		"model":    request.Model,
		"messages": request.Messages,
	}

	// Add provider-specific fields
	switch c.provider {
	case "openai":
		// OpenAI format
		result["temperature"] = 0.7
		result["max_tokens"] = 1000
	case "anthropic":
		// Anthropic format
		result["max_tokens_to_sample"] = 1000
		result["temperature"] = 0.7
	case "google":
		// Google format
		result["generationConfig"] = map[string]interface{}{
			"temperature":     0.7,
			"maxOutputTokens": 1000,
		}
	}

	return result
}

// convertFromProviderFormat converts provider response to generic format
func (c *ACPClient) convertFromProviderFormat(providerResponse map[string]interface{}) (*llmverifier.ChatCompletionResponse, error) {
	response := &llmverifier.ChatCompletionResponse{}

	switch c.provider {
	case "openai":
		// OpenAI format
		if choices, ok := providerResponse["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if message, ok := choice["message"].(map[string]interface{}); ok {
					response.Choices = []llmverifier.Choice{
						{
							Message: llmverifier.Message{
								Role:    getString(message, "role"),
								Content: getString(message, "content"),
							},
						},
					}
				}
			}
		}
	case "anthropic":
		// Anthropic format
		if content, ok := providerResponse["completion"].(string); ok {
			response.Choices = []llmverifier.Choice{
				{
					Message: llmverifier.Message{
						Role:    "assistant",
						Content: content,
					},
				},
			}
		}
	default:
		// Generic format
		if choices, ok := providerResponse["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if message, ok := choice["message"].(map[string]interface{}); ok {
					response.Choices = []llmverifier.Choice{
						{
							Message: llmverifier.Message{
								Role:    getString(message, "role"),
								Content: getString(message, "content"),
							},
						},
					}
				}
			}
		}
	}

	return response, nil
}

// addProviderHeaders adds provider-specific headers to the request
func (c *ACPClient) addProviderHeaders(req *http.Request) {
	switch c.provider {
	case "anthropic":
		req.Header.Set("anthropic-version", "2023-06-01")
		if c.baseClient.APIKey != "" {
			req.Header.Set("x-api-key", c.baseClient.APIKey)
		}
	case "google":
		if c.baseClient.APIKey != "" {
			req.URL.RawQuery = "key=" + c.baseClient.APIKey
		}
	}
}

// Helper functions

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func evaluateCodeQuality(code string) float64 {
	// Simple heuristic-based code quality evaluation
	score := 0.0

	// Check for function definition
	if strings.Contains(code, "def ") {
		score += 0.2
	}

	// Check for type hints
	if strings.Contains(code, ": ") && strings.Contains(code, "->") {
		score += 0.2
	}

	// Check for docstring
	if strings.Contains(code, `"""`) || strings.Contains(code, `'''`) {
		score += 0.2
	}

	// Check for error handling
	if strings.Contains(code, "try:") && strings.Contains(code, "except") {
		score += 0.2
	}

	// Check for return statement
	if strings.Contains(code, "return") {
		score += 0.2
	}

	return score
}

func evaluateErrorDetection(response string) float64 {
	// Simple heuristic-based error detection evaluation
	responseLower := strings.ToLower(response)
	score := 0.0

	// Check for error mentions
	if strings.Contains(responseLower, "error") || strings.Contains(responseLower, "issue") {
		score += 0.3
	}

	// Check for line number references
	if strings.Contains(responseLower, "line") {
		score += 0.3
	}

	// Check for specific error types
	errorTypes := []string{"keyerror", "syntax", "typeerror", "valueerror", "indexerror"}
	for _, errType := range errorTypes {
		if strings.Contains(responseLower, errType) {
			score += 0.4
			break
		}
	}

	return score
}