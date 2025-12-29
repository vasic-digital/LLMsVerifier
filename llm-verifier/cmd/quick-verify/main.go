package main

import (
    "context"
    "log"
    
    "llm-verifier/llmverifier"
)

func main() {
    log.SetFlags(log.LstdFlags | log.Lshortfile)
    
    log.Println("Starting direct API test...")
    
    client := llmverifier.NewLLMClient("https://api.deepseek.com/v1", "${DEEPSEEK_API_KEY}", nil)
    
    models, err := client.ListModels(context.Background())
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }
    
    log.Printf("✅ SUCCESS! Found %d models from DeepSeek", len(models))
    for _, m := range models {
        log.Printf("  - %s", m.ID)
    }
}
