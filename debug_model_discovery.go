package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Test the models.dev API directly to understand the JSON structure
type TestModelDetails struct {
	ID               string              `json:"id"`
	Name             string              `json:"name"`
	Family           string              `json:"family,omitempty"`
	Attachment       bool                `json:"attachment"`
	Reasoning        bool                `json:"reasoning"`
	ToolCall         bool                `json:"tool_call"`
	Temperature      bool                `json:"temperature"`
	Knowledge        string              `json:"knowledge,omitempty"`
	ReleaseDate      string              `json:"release_date"`
	LastUpdated      string              `json:"last_updated"`
	Modalities       ModelModalities     `json:"modalities"`
	OpenWeights      bool                `json:"open_weights"`
	Cost             ModelCost           `json:"cost"`
	Limits           ModelLimits         `json:"limit"`
	StructuredOutput bool                `json:"structured_output,omitempty"`
	Status           string              `json:"status,omitempty"`
	ContextOver200k  json.RawMessage     `json:"context_over_200k,omitempty"` // Use RawMessage to handle any type
	Interleaved      json.RawMessage     `json:"interleaved,omitempty"`       // Use RawMessage to handle any type
}

type ModelModalities struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

type ModelCost struct {
	Input              float64 `json:"input"`
	Output             float64 `json:"output"`
	Reasoning          float64 `json:"reasoning,omitempty"`
	CacheRead          float64 `json:"cache_read,omitempty"`
	CacheWrite         float64 `json:"cache_write,omitempty"`
	InputAudio         float64 `json:"input_audio,omitempty"`
	OutputAudio        float64 `json:"output_audio,omitempty"`
}

type ModelLimits struct {
	Context uint64 `json:"context"`
	Input   uint64 `json:"input"`
	Output  uint64 `json:"output"`
}

type TestProviderData struct {
	ID     string                    `json:"id"`
	Env    []string                  `json:"env"`
	NPM    string                    `json:"npm"`
	API    string                    `json:"api,omitempty"`
	Name   string                    `json:"name"`
	Doc    string                    `json:"doc"`
	Models map[string]TestModelDetails `json:"models"`
}

type TestResponse map[string]TestProviderData

func main() {
	fmt.Println("Testing models.dev API to understand JSON structure...")
	
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get("https://models.dev/api.json")
	if err != nil {
		fmt.Printf("Error fetching models.dev: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return
	}
	
	fmt.Printf("Response size: %d bytes\n", len(body))
	
	// Try to parse with our test struct
	var testResp TestResponse
	if err := json.Unmarshal(body, &testResp); err != nil {
		fmt.Printf("Error parsing with test struct: %v\n", err)
		
		// Try to parse as generic JSON to see structure
		var generic map[string]interface{}
		if err := json.Unmarshal(body, &generic); err != nil {
			fmt.Printf("Error parsing as generic JSON: %v\n", err)
			return
		}
		
		// Look for providers with context_over_200k
		for providerID, providerData := range generic {
			if providerMap, ok := providerData.(map[string]interface{}); ok {
				if models, ok := providerMap["models"].(map[string]interface{}); ok {
					for modelID, modelData := range models {
						if modelMap, ok := modelData.(map[string]interface{}); ok {
							if contextOver200k, exists := modelMap["context_over_200k"]; exists {
								fmt.Printf("Provider %s, Model %s has context_over_200k: %v (type: %T)\n", 
									providerID, modelID, contextOver200k, contextOver200k)
							}
						}
					}
				}
			}
		}
		return
	}
	
	fmt.Printf("Successfully parsed %d providers\n", len(testResp))
	
	// Check for context_over_200k usage
	contextCount := 0
	for providerID, provider := range testResp {
		for modelID, model := range provider.Models {
			if len(model.ContextOver200k) > 0 {
				contextCount++
				if contextCount <= 5 { // Show first 5 examples
					fmt.Printf("Provider %s, Model %s has context_over_200k: %s\n", 
						providerID, modelID, string(model.ContextOver200k))
				}
			}
		}
	}
	fmt.Printf("Total models with context_over_200k: %d\n", contextCount)
	
	// Test a specific provider that's failing
	if provider, exists := testResp["nlpcloud"]; exists {
		fmt.Printf("\nNLPCloud provider data:\n")
		fmt.Printf("  ID: %s\n", provider.ID)
		fmt.Printf("  Name: %s\n", provider.Name)
		fmt.Printf("  NPM: %s\n", provider.NPM)
		fmt.Printf("  Models: %d\n", len(provider.Models))
		
		for modelID, model := range provider.Models {
			fmt.Printf("  Model %s: %s\n", modelID, model.Name)
			if len(model.ContextOver200k) > 0 {
				fmt.Printf("    context_over_200k: %s\n", string(model.ContextOver200k))
			}
		}
	}
}