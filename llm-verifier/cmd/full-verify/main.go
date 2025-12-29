package main

import (
    "context"
    "log"
    
    "llm-verifier/llmverifier"
)

func main() {
    log.SetFlags(log.LstdFlags | log.Lshortfile)
    
    providers := []struct {
        name    string
        url     string
        key     string
    }{
        {"DeepSeek", "https://api.deepseek.com/v1", "${DEEPSEEK_API_KEY}"},
        {"NVIDIA", "https://integrate.api.nvidia.com/v1", "REDACTED_API_KEY"},
    }
    
    totalModels := 0
    
    for _, provider := range providers {
        log.Printf("Fetching from %s...", provider.name)
        
        client := llmverifier.NewLLMClient(provider.url, provider.key, nil)
        models, err := client.ListModels(context.Background())
        if err != nil {
            log.Printf("  ❌ Failed: %v", err)
            continue
        }
        
        log.Printf("  ✅ Found %d models", len(models))
        
        for i, model := range models {
            if i < 5 {
                log.Printf("    - %s", model.ID)
            }
        }
        
        if len(models) > 5 {
            log.Printf("    ... and %d more", len(models)-5)
        }
        
        totalModels += len(models)
    }
    
    log.Printf("\n🎉 Total models discovered: %d", totalModels)
}
