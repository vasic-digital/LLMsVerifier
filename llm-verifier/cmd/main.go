package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"llm-verifier/llmverifier"
)

var (
	configFile string
	outputDir  string
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "llm-verifier",
		Short: "LLM Verifier - Verify and benchmark LLMs for coding capabilities",
		Long:  `A tool to verify, test, and benchmark LLMs based on their coding capabilities and other features.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := runVerification(); err != nil {
				log.Fatalf("Error during LLM verification: %v", err)
			}
		},
	}

	rootCmd.Flags().StringVarP(&configFile, "config", "c", "config.yaml", "Configuration file path")
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "./reports", "Output directory for reports")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runVerification() error {
	cfg, err := llmverifier.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	verifier := llmverifier.New(cfg)
	
	results, err := verifier.Verify()
	if err != nil {
		return fmt.Errorf("failed to verify models: %w", err)
	}

	if err := verifier.GenerateMarkdownReport(results, outputDir); err != nil {
		return fmt.Errorf("failed to generate markdown report: %w", err)
	}

	if err := verifier.GenerateJSONReport(results, outputDir); err != nil {
		return fmt.Errorf("failed to generate JSON report: %w", err)
	}

	return nil
}