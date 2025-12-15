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
	serverURL  string
	username   string
	password   string
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

	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "config.yaml", "Configuration file path")
	rootCmd.PersistentFlags().StringVarP(&serverURL, "server", "s", "http://localhost:8080", "API server URL")
	rootCmd.PersistentFlags().StringVarP(&username, "username", "u", "", "Username for authentication")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "Password for authentication")
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", "./reports", "Output directory for reports")

	// Server command
	rootCmd.AddCommand(serverCmd())

	// Models commands
	rootCmd.AddCommand(modelsCmd())

	// Providers commands
	rootCmd.AddCommand(providersCmd())

	// Verification results commands
	rootCmd.AddCommand(resultsCmd())

	// Pricing commands
	rootCmd.AddCommand(pricingCmd())

	// Limits commands
	rootCmd.AddCommand(limitsCmd())

	// Issues commands
	rootCmd.AddCommand(issuesCmd())

	// Events commands
	rootCmd.AddCommand(eventsCmd())

	// Schedules commands
	rootCmd.AddCommand(schedulesCmd())

	// Config exports commands
	rootCmd.AddCommand(exportsCmd())

	// Logs commands
	rootCmd.AddCommand(logsCmd())

	// Config commands
	rootCmd.AddCommand(configCmd())

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

func serverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the REST API server",
		Long:  `Start the REST API server with all endpoints for managing models, providers, and verification results.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := runServer(); err != nil {
				log.Fatalf("Error starting server: %v", err)
			}
		},
	}

	cmd.Flags().StringP("port", "p", "8080", "Port to run the server on")
	return cmd
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

func modelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "models",
		Short: "Manage LLM models",
		Long:  `List, create, update, delete, and verify LLM models.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all models",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Listing models... (implementation pending)")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "get [id]",
		Short: "Get model details",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Getting model %s... (implementation pending)\n", args[0])
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "create",
		Short: "Create a new model",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Creating model... (implementation pending)")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "verify [id]",
		Short: "Verify a model",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Verifying model %s... (implementation pending)\n", args[0])
		},
	})

	return cmd
}

func providersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "Manage LLM providers",
		Long:  `List, create, update, and delete LLM providers.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all providers",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Listing providers... (implementation pending)")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "get [id]",
		Short: "Get provider details",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Getting provider %s... (implementation pending)\n", args[0])
		},
	})

	return cmd
}

func resultsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "results",
		Short: "Manage verification results",
		Long:  `List, create, update, and delete verification results.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all verification results",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Listing verification results... (implementation pending)")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "get [id]",
		Short: "Get verification result details",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Getting verification result %s... (implementation pending)\n", args[0])
		},
	})

	return cmd
}

func pricingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pricing",
		Short: "Manage pricing information",
		Long:  `List, create, update, and delete pricing information for models.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all pricing entries",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Listing pricing entries... (implementation pending)")
		},
	})

	return cmd
}

func limitsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "limits",
		Short: "Manage rate limits",
		Long:  `List, create, update, and delete rate limit information for models.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all limit entries",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Listing limit entries... (implementation pending)")
		},
	})

	return cmd
}

func issuesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issues",
		Short: "Manage issues",
		Long:  `List, create, update, and delete issue reports for models.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all issues",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Listing issues... (implementation pending)")
		},
	})

	return cmd
}

func eventsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Manage events",
		Long:  `List, create, update, and delete system events.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all events",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Listing events... (implementation pending)")
		},
	})

	return cmd
}

func schedulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedules",
		Short: "Manage schedules",
		Long:  `List, create, update, and delete verification schedules.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all schedules",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Listing schedules... (implementation pending)")
		},
	})

	return cmd
}

func exportsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exports",
		Short: "Manage configuration exports",
		Long:  `List, create, update, delete, and download configuration exports.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all configuration exports",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Listing configuration exports... (implementation pending)")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "download [id]",
		Short: "Download a configuration export",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Downloading configuration export %s... (implementation pending)\n", args[0])
		},
	})

	return cmd
}

func logsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Manage logs",
		Long:  `List and view system logs.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List all logs",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Listing logs... (implementation pending)")
		},
	})

	return cmd
}

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `View, update, and export system configuration.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Showing configuration... (implementation pending)")
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "export [format]",
		Short: "Export configuration in specified format",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Exporting configuration in %s format... (implementation pending)\n", args[0])
		},
	})

	return cmd
}
