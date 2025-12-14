package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"llm-verifier/api"
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

	// Server command
	var serverCmd = &cobra.Command{
		Use:   "server",
		Short: "Start the REST API server",
		Long:  `Start the REST API server with all endpoints for managing models, providers, and verification results.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := runServer(); err != nil {
				log.Fatalf("Error starting server: %v", err)
			}
		},
	}

	serverCmd.Flags().StringVarP(&configFile, "config", "c", "config.yaml", "Configuration file path")
	serverCmd.Flags().StringP("port", "p", "8080", "Port to run the server on")

	rootCmd.AddCommand(serverCmd)

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

func runServer() error {
	cfg, err := llmverifier.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	server, err := api.NewServer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Use port from config or flag
	port := cfg.API.Port
	if port == "" {
		port = "8080"
	}

	return server.Start(port)
}
