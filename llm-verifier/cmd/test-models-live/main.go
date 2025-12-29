package main

import (
    "context"
    "fmt"
    "time"
    "llm-verifier/client"
)

func main() {
    httpClient := client.NewHTTPClient(30 * time.Second)
    ctx := context.Background()
    
    testCases := []struct{
        name string
        provider string
        model string
        apiKey string
    }{
        {"GPT-4", "openrouter", "openai/gpt-4", "REDACTED_API_KEY"},
        {"Claude", "openrouter", "anthropic/claude-3.5-sonnet", "REDACTED_API_KEY"},
        {"DeepSeek Chat", "deepseek", "deepseek-chat", "${DEEPSEEK_API_KEY}"},
    }
    
    verified := 0
    for _, tc := range testCases {
        fmt.Printf("\n=== Testing %s ===\n", tc.name)
        
        exists, err := httpClient.TestModelExists(ctx, tc.provider, tc.apiKey, tc.model)
        if err != nil {
            fmt.Printf("  ❌ Existence: %v\n", err)
            continue
        }
        fmt.Printf("  ✅ Exists: %v\n", exists)
        
        totalTime, ttft, err, errMsg, responsive, statusCode, httpErr := httpClient.TestResponsiveness(
            ctx, tc.provider, tc.apiKey, tc.model, "What is 2+2?")
        
        if err != nil || !responsive {
            fmt.Printf("  ❌ Responsiveness: %v (HTTP %d)\n", errMsg, statusCode)
            if httpErr != nil {
                fmt.Printf("     HTTP Error: %v\n", httpErr)
            }
            continue
        }
        
        fmt.Printf("  ✅ Responsive: HTTP %d\n", statusCode)
        fmt.Printf("  ⏱️  Time: %v, TTFT: %v\n", totalTime, ttft)
        verified++
    }
    
    fmt.Printf("\n=== Summary ===\n")
    fmt.Printf("✅ Verified: %d/%d\n", verified, len(testCases))
}
