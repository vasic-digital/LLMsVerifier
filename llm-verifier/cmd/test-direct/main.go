package main

import (
	"log"

	"llm-verifier/llmverifier"
)

func main() {
	cfg, err := llmverifier.LoadConfig("config_minimal.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Config loaded with %d LLMs", len(cfg.LLMs))

	verifier := llmverifier.New(cfg)

	log.Println("Starting verification...")
	results, err := verifier.Verify()
	if err != nil {
		log.Fatalf("Verification failed: %v", err)
	}

	log.Printf("✅ SUCCESS! Generated %d verification results", len(results))
}
