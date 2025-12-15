package main

import (
	"fmt"
	"net/http"
	"llm-verifier/enhanced"
)

func main() {
	detector := enhanced.NewLimitsDetector()
	
	// Test with Set() to ensure proper capitalization
	headers := http.Header{}
	headers.Set("x-ratelimit-limit-requests", "60")
	headers.Set("x-ratelimit-limit-tokens", "40000")
	headers.Set("x-ratelimit-remaining-requests", "45")
	headers.Set("x-ratelimit-remaining-tokens", "30000")
	headers.Set("x-ratelimit-reset", "1700000000")
	
	fmt.Printf("Headers:\n")
	for k, v := range headers {
		fmt.Printf("  %s: %v\n", k, v)
	}
	
	limits, err := detector.DetectLimits("openai", "test-model", headers)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("\nOpenAI Limits:\n")
	fmt.Printf("  RequestsPerMinute: %v\n", limits.RequestsPerMinute)
	fmt.Printf("  TokensPerMinute: %v\n", limits.TokensPerMinute)
	fmt.Printf("  CurrentUsage: %v\n", limits.CurrentUsage)
	fmt.Printf("  ResetTime: %v\n", limits.ResetTime)
}
