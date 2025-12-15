package main

import (
	"fmt"
	"net/http"
	"llm-verifier/enhanced"
)

func main() {
	detector := enhanced.NewLimitsDetector()
	
	// Test OpenAI headers
	headers := http.Header{
		"x-ratelimit-limit-requests":     []string{"60"},
		"x-ratelimit-limit-tokens":       []string{"40000"},
		"x-ratelimit-remaining-requests": []string{"45"},
		"x-ratelimit-remaining-tokens":   []string{"30000"},
		"x-ratelimit-reset":              []string{"1700000000"},
	}
	
	limits, err := detector.DetectLimits("openai", "test-model", headers)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("OpenAI Limits:\n")
	fmt.Printf("  RequestsPerMinute: %v\n", limits.RequestsPerMinute)
	fmt.Printf("  TokensPerMinute: %v\n", limits.TokensPerMinute)
	fmt.Printf("  CurrentUsage: %v\n", limits.CurrentUsage)
	fmt.Printf("  ResetTime: %v\n", limits.ResetTime)
}
