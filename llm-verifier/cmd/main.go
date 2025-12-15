package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"llm-verifier/api"
	"llm-verifier/client"
	"llm-verifier/database"
	"llm-verifier/llmverifier"
	"llm-verifier/tui"
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

	// Users commands
	rootCmd.AddCommand(usersCmd())

	// Batch operations command
	rootCmd.AddCommand(batchCmd())

	// Interactive mode command
	rootCmd.AddCommand(interactiveCmd())

	// Validation commands
	rootCmd.AddCommand(validateCmd())

	// Export/Import commands
	rootCmd.AddCommand(exportCmd())
	rootCmd.AddCommand(importCmd())

	// TUI command
	rootCmd.AddCommand(tuiCmd())

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

	cmd.Flags().String("port", "8080", "Port to run the server on")
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

func getClient() (*client.Client, error) {
	c := client.New(serverURL)

	// If username and password are provided, try to login
	if username != "" && password != "" {
		if err := c.Login(username, password); err != nil {
			return nil, fmt.Errorf("authentication failed: %w", err)
		}
	}

	return c, nil
}

func printJSON(data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(jsonData))
	return nil
}

func printModelsTable(models []map[string]interface{}) {
	if len(models) == 0 {
		fmt.Println("No models found")
		return
	}

	// Print header
	fmt.Printf("%-20s %-15s %-10s %-8s %-6s\n", "NAME", "PROVIDER", "VERSION", "SCORE", "STATUS")
	fmt.Println(strings.Repeat("-", 60))

	// Print rows
	for _, model := range models {
		name := getStringField(model, "name")
		provider := getStringField(model, "provider")
		version := getStringField(model, "version")
		score := getFloatField(model, "score")
		status := getStringField(model, "status")

		if len(name) > 18 {
			name = name[:18] + "..."
		}
		if len(provider) > 13 {
			provider = provider[:13] + "..."
		}
		if len(version) > 8 {
			version = version[:8] + "..."
		}

		scoreStr := ""
		if score > 0 {
			scoreStr = fmt.Sprintf("%.1f", score)
		}

		fmt.Printf("%-20s %-15s %-10s %-8s %-6s\n", name, provider, version, scoreStr, status)
	}
}

func getStringField(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getFloatField(data map[string]interface{}, key string) float64 {
	if val, ok := data[key]; ok {
		if num, ok := val.(float64); ok {
			return num
		}
	}
	return 0.0
}

func printProvidersTable(providers []map[string]interface{}) {
	if len(providers) == 0 {
		fmt.Println("No providers found")
		return
	}

	// Print header
	fmt.Printf("%-20s %-10s %-8s %-6s\n", "NAME", "MODELS", "SCORE", "STATUS")
	fmt.Println(strings.Repeat("-", 45))

	// Print rows
	for _, provider := range providers {
		name := getStringField(provider, "name")
		models := getIntField(provider, "model_count")
		score := getFloatField(provider, "avg_score")
		status := getStringField(provider, "status")

		if len(name) > 18 {
			name = name[:18] + "..."
		}

		scoreStr := ""
		if score > 0 {
			scoreStr = fmt.Sprintf("%.1f", score)
		}

		fmt.Printf("%-20s %-10d %-8s %-6s\n", name, models, scoreStr, status)
	}
}

func getIntField(data map[string]interface{}, key string) int {
	if val, ok := data[key]; ok {
		if num, ok := val.(float64); ok {
			return int(num)
		}
	}
	return 0
}

func printResultsTable(results []map[string]interface{}) {
	if len(results) == 0 {
		fmt.Println("No verification results found")
		return
	}

	// Print header
	fmt.Printf("%-20s %-12s %-8s %-10s %-8s\n", "MODEL", "STATUS", "SCORE", "STARTED", "DURATION")
	fmt.Println(strings.Repeat("-", 60))

	// Print rows
	for _, result := range results {
		model := getStringField(result, "model_name")
		status := getStringField(result, "status")
		score := getFloatField(result, "score")
		started := getStringField(result, "created_at")
		duration := getStringField(result, "duration")

		if len(model) > 18 {
			model = model[:18] + "..."
		}
		if len(status) > 10 {
			status = status[:10]
		}
		if len(started) > 10 {
			started = started[:10]
		}
		if len(duration) > 8 {
			duration = duration[:8]
		}

		scoreStr := ""
		if score > 0 {
			scoreStr = fmt.Sprintf("%.1f", score)
		}

		fmt.Printf("%-20s %-12s %-8s %-10s %-8s\n", model, status, scoreStr, started, duration)
	}
}

func modelsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "models",
		Short: "Manage LLM models",
		Long:  `List, create, update, delete, and verify LLM models.`,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all models",
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			models, err := c.GetModels()
			if err != nil {
				log.Fatalf("Failed to get models: %v", err)
			}

			// Apply filtering if specified
			filter, _ := cmd.Flags().GetString("filter")
			if filter != "" {
				filtered := make([]map[string]interface{}, 0)
				for _, model := range models {
					if name, ok := model["name"].(string); ok && strings.Contains(strings.ToLower(name), strings.ToLower(filter)) {
						filtered = append(filtered, model)
					}
				}
				models = filtered
			}

			// Apply limit if specified
			limit, _ := cmd.Flags().GetInt("limit")
			if limit > 0 && limit < len(models) {
				models = models[:limit]
			}

			// Output format
			format, _ := cmd.Flags().GetString("format")
			switch format {
			case "table":
				printModelsTable(models)
			default:
				if err := printJSON(models); err != nil {
					log.Fatalf("Failed to print models: %v", err)
				}
			}
		},
	}
	listCmd.Flags().String("filter", "", "Filter models by name")
	listCmd.Flags().Int("limit", 0, "Limit number of results")
	listCmd.Flags().String("format", "json", "Output format (json, table)")

	cmd.AddCommand(listCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "get [id]",
		Short: "Get model details",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			model, err := c.GetModel(args[0])
			if err != nil {
				log.Fatalf("Failed to get model: %v", err)
			}

			if err := printJSON(model); err != nil {
				log.Fatalf("Failed to print model: %v", err)
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "create [provider_id] [model_id] [name]",
		Short: "Create a new model",
		Args:  cobra.ExactArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			model := map[string]interface{}{
				"provider_id": args[0],
				"model_id":    args[1],
				"name":        args[2],
			}

			result, err := c.CreateModel(model)
			if err != nil {
				log.Fatalf("Failed to create model: %v", err)
			}

			if err := printJSON(result); err != nil {
				log.Fatalf("Failed to print create result: %v", err)
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "verify [id]",
		Short: "Verify a model",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			result, err := c.VerifyModel(args[0])
			if err != nil {
				log.Fatalf("Failed to verify model: %v", err)
			}

			if err := printJSON(result); err != nil {
				log.Fatalf("Failed to print verification result: %v", err)
			}
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

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all providers",
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			providers, err := c.GetProviders()
			if err != nil {
				log.Fatalf("Failed to get providers: %v", err)
			}

			// Apply filtering if specified
			filter, _ := cmd.Flags().GetString("filter")
			if filter != "" {
				filtered := make([]map[string]interface{}, 0)
				for _, provider := range providers {
					if name, ok := provider["name"].(string); ok && strings.Contains(strings.ToLower(name), strings.ToLower(filter)) {
						filtered = append(filtered, provider)
					}
				}
				providers = filtered
			}

			// Apply limit if specified
			limit, _ := cmd.Flags().GetInt("limit")
			if limit > 0 && limit < len(providers) {
				providers = providers[:limit]
			}

			// Output format
			format, _ := cmd.Flags().GetString("format")
			switch format {
			case "table":
				printProvidersTable(providers)
			default:
				if err := printJSON(providers); err != nil {
					log.Fatalf("Failed to print providers: %v", err)
				}
			}
		},
	}
	listCmd.Flags().String("filter", "", "Filter providers by name")
	listCmd.Flags().Int("limit", 0, "Limit number of results")
	listCmd.Flags().String("format", "json", "Output format (json, table)")

	cmd.AddCommand(listCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "get [id]",
		Short: "Get provider details",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			provider, err := c.GetProvider(args[0])
			if err != nil {
				log.Fatalf("Failed to get provider: %v", err)
			}

			if err := printJSON(provider); err != nil {
				log.Fatalf("Failed to print provider: %v", err)
			}
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

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all verification results",
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			results, err := c.GetVerificationResults()
			if err != nil {
				log.Fatalf("Failed to get verification results: %v", err)
			}

			// Apply filtering if specified
			filter, _ := cmd.Flags().GetString("filter")
			if filter != "" {
				filtered := make([]map[string]interface{}, 0)
				for _, result := range results {
					if model, ok := result["model_name"].(string); ok && strings.Contains(strings.ToLower(model), strings.ToLower(filter)) {
						filtered = append(filtered, result)
					}
				}
				results = filtered
			}

			// Apply limit if specified
			limit, _ := cmd.Flags().GetInt("limit")
			if limit > 0 && limit < len(results) {
				results = results[:limit]
			}

			// Output format
			format, _ := cmd.Flags().GetString("format")
			switch format {
			case "table":
				printResultsTable(results)
			default:
				if err := printJSON(results); err != nil {
					log.Fatalf("Failed to print verification results: %v", err)
				}
			}
		},
	}
	listCmd.Flags().String("filter", "", "Filter results by model name")
	listCmd.Flags().Int("limit", 0, "Limit number of results")
	listCmd.Flags().String("format", "json", "Output format (json, table)")

	cmd.AddCommand(listCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "get [id]",
		Short: "Get verification result details",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			result, err := c.GetVerificationResult(args[0])
			if err != nil {
				log.Fatalf("Failed to get verification result: %v", err)
			}

			if err := printJSON(result); err != nil {
				log.Fatalf("Failed to print verification result: %v", err)
			}
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
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			pricing, err := c.GetPricing()
			if err != nil {
				log.Fatalf("Failed to get pricing: %v", err)
			}

			if err := printJSON(pricing); err != nil {
				log.Fatalf("Failed to print pricing: %v", err)
			}
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
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			limits, err := c.GetLimits()
			if err != nil {
				log.Fatalf("Failed to get limits: %v", err)
			}

			if err := printJSON(limits); err != nil {
				log.Fatalf("Failed to print limits: %v", err)
			}
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
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			issues, err := c.GetIssues()
			if err != nil {
				log.Fatalf("Failed to get issues: %v", err)
			}

			if err := printJSON(issues); err != nil {
				log.Fatalf("Failed to print issues: %v", err)
			}
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
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			events, err := c.GetEvents()
			if err != nil {
				log.Fatalf("Failed to get events: %v", err)
			}

			if err := printJSON(events); err != nil {
				log.Fatalf("Failed to print events: %v", err)
			}
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
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			schedules, err := c.GetSchedules()
			if err != nil {
				log.Fatalf("Failed to get schedules: %v", err)
			}

			if err := printJSON(schedules); err != nil {
				log.Fatalf("Failed to print schedules: %v", err)
			}
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
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			exports, err := c.GetConfigExports()
			if err != nil {
				log.Fatalf("Failed to get config exports: %v", err)
			}

			if err := printJSON(exports); err != nil {
				log.Fatalf("Failed to print config exports: %v", err)
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "download [id]",
		Short: "Download a configuration export",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			data, err := c.DownloadConfigExport(args[0])
			if err != nil {
				log.Fatalf("Failed to download config export: %v", err)
			}

			fmt.Println(string(data))
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
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			logs, err := c.GetLogs()
			if err != nil {
				log.Fatalf("Failed to get logs: %v", err)
			}

			if err := printJSON(logs); err != nil {
				log.Fatalf("Failed to print logs: %v", err)
			}
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
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			config, err := c.GetConfig()
			if err != nil {
				log.Fatalf("Failed to get config: %v", err)
			}

			if err := printJSON(config); err != nil {
				log.Fatalf("Failed to print config: %v", err)
			}
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "export [format]",
		Short: "Export configuration in specified format",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			result, err := c.ExportConfig(args[0])
			if err != nil {
				log.Fatalf("Failed to export config: %v", err)
			}

			if err := printJSON(result); err != nil {
				log.Fatalf("Failed to print export result: %v", err)
			}
		},
	})

	return cmd
}

func tuiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Start the Terminal User Interface",
		Long:  `Start the interactive Terminal User Interface for managing models, providers, and verification results.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := runTUI(); err != nil {
				log.Fatalf("Error starting TUI: %v", err)
			}
		},
	}
	return cmd
}

func runTUI() error {
	// Create client for TUI
	client, err := getClient()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Start the TUI application
	app := tui.NewApp(client)
	p := tea.NewProgram(app, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}

	return nil
}

func batchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch",
		Short: "Execute batch operations",
		Long:  `Execute multiple operations in batch mode for efficiency.`,
	}

	verifyCmd := &cobra.Command{
		Use:   "verify [file]",
		Short: "Batch verify models from a file",
		Long:  `Verify multiple models from a JSON file containing model configurations.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			// Read and parse batch file
			data, err := os.ReadFile(args[0])
			if err != nil {
				log.Fatalf("Failed to read batch file: %v", err)
			}

			var models []map[string]interface{}
			if err := json.Unmarshal(data, &models); err != nil {
				log.Fatalf("Failed to parse batch file: %v", err)
			}

			fmt.Printf("Starting batch verification of %d models...\n", len(models))

			results := make([]map[string]interface{}, 0, len(models))
			for i, model := range models {
				fmt.Printf("Verifying model %d/%d: %v\n", i+1, len(models), model["name"])

				result, err := c.VerifyModel(fmt.Sprintf("%v", model["id"]))
				if err != nil {
					fmt.Printf("Error verifying model %v: %v\n", model["name"], err)
					continue
				}

				results = append(results, result)
			}

			fmt.Printf("Batch verification completed. %d models verified successfully.\n", len(results))

			// Save results
			outputFile := "batch_results.json"
			data, _ = json.MarshalIndent(results, "", "  ")
			if err := os.WriteFile(outputFile, data, 0644); err != nil {
				log.Fatalf("Failed to save results: %v", err)
			}

			fmt.Printf("Results saved to %s\n", outputFile)
		},
	}

	cmd.AddCommand(verifyCmd)
	return cmd
}

func interactiveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "interactive",
		Short: "Start interactive mode",
		Long:  `Start an interactive session for managing models and providers.`,
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			runInteractiveMode(c)
		},
	}
	return cmd
}

func validateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration and setup",
		Long:  `Validate system configuration, database connectivity, and API endpoints.`,
	}

	configCmd := &cobra.Command{
		Use:   "config [file]",
		Short: "Validate configuration file",
		Long:  `Validate the syntax and structure of a configuration file.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := llmverifier.LoadConfig(args[0])
			if err != nil {
				log.Fatalf("Configuration validation failed: %v", err)
			}

			fmt.Println("✓ Configuration file is valid")
			fmt.Printf("✓ API Port: %s\n", cfg.API.Port)
			fmt.Printf("✓ Database Path: %s\n", cfg.Database.Path)
			fmt.Printf("✓ LLMs configured: %d\n", len(cfg.LLMs))
			fmt.Printf("✓ Profile: %s\n", cfg.Profile)
		},
	}

	systemCmd := &cobra.Command{
		Use:   "system",
		Short: "Validate system setup",
		Long:  `Validate database connectivity, API endpoints, and system health.`,
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			// Test database connectivity
			fmt.Print("Testing database connectivity... ")
			_, err = c.GetModels()
			if err != nil {
				fmt.Printf("✗ Failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✓ OK")

			// Test API endpoints
			fmt.Print("Testing API endpoints... ")
			_, err = c.GetProviders()
			if err != nil {
				fmt.Printf("✗ Failed: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("✓ OK")

			fmt.Println("✓ System validation completed successfully")
		},
	}

	cmd.AddCommand(configCmd)
	cmd.AddCommand(systemCmd)
	return cmd
}

func exportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export data and configurations",
		Long:  `Export models, providers, configurations, and verification results.`,
	}

	modelsCmd := &cobra.Command{
		Use:   "models [output_file]",
		Short: "Export models data",
		Long:  `Export all models data to a JSON file.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			models, err := c.GetModels()
			if err != nil {
				log.Fatalf("Failed to get models: %v", err)
			}

			data, err := json.MarshalIndent(models, "", "  ")
			if err != nil {
				log.Fatalf("Failed to marshal models: %v", err)
			}

			if err := os.WriteFile(args[0], data, 0644); err != nil {
				log.Fatalf("Failed to write file: %v", err)
			}

			fmt.Printf("Exported %d models to %s\n", len(models), args[0])
		},
	}

	providersCmd := &cobra.Command{
		Use:   "providers [output_file]",
		Short: "Export providers data",
		Long:  `Export all providers data to a JSON file.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			providers, err := c.GetProviders()
			if err != nil {
				log.Fatalf("Failed to get providers: %v", err)
			}

			data, err := json.MarshalIndent(providers, "", "  ")
			if err != nil {
				log.Fatalf("Failed to marshal providers: %v", err)
			}

			if err := os.WriteFile(args[0], data, 0644); err != nil {
				log.Fatalf("Failed to write file: %v", err)
			}

			fmt.Printf("Exported %d providers to %s\n", len(providers), args[0])
		},
	}

	cmd.AddCommand(modelsCmd)
	cmd.AddCommand(providersCmd)
	return cmd
}

func importCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import data and configurations",
		Long:  `Import models, providers, and configurations from files.`,
	}

	modelsCmd := &cobra.Command{
		Use:   "models [input_file]",
		Short: "Import models data",
		Long:  `Import models data from a JSON file.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			data, err := os.ReadFile(args[0])
			if err != nil {
				log.Fatalf("Failed to read file: %v", err)
			}

			var models []map[string]interface{}
			if err := json.Unmarshal(data, &models); err != nil {
				log.Fatalf("Failed to parse models: %v", err)
			}

			imported := 0
			for _, model := range models {
				_, err := c.CreateModel(model)
				if err != nil {
					fmt.Printf("Failed to import model %v: %v\n", model["name"], err)
					continue
				}
				imported++
			}

			fmt.Printf("Imported %d/%d models successfully\n", imported, len(models))
		},
	}

	providersCmd := &cobra.Command{
		Use:   "providers [input_file]",
		Short: "Import providers data",
		Long:  `Import providers data from a JSON file.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			data, err := os.ReadFile(args[0])
			if err != nil {
				log.Fatalf("Failed to read file: %v", err)
			}

			var providers []map[string]interface{}
			if err := json.Unmarshal(data, &providers); err != nil {
				log.Fatalf("Failed to parse providers: %v", err)
			}

			// Note: Provider import not implemented yet - requires API endpoint
			fmt.Printf("Provider import not yet available. Found %d providers in file.\n", len(providers))
		},
	}

	cmd.AddCommand(modelsCmd)
	cmd.AddCommand(providersCmd)
	return cmd
}

func runInteractiveMode(client *client.Client) {
	fmt.Println("=== LLM Verifier Interactive Mode ===")
	fmt.Println("Available commands:")
	fmt.Println("  list models     - List all models")
	fmt.Println("  list providers  - List all providers")
	fmt.Println("  verify [id]     - Verify a specific model")
	fmt.Println("  status          - Show system status")
	fmt.Println("  help            - Show this help")
	fmt.Println("  quit            - Exit interactive mode")
	fmt.Println()

	for {
		fmt.Print("> ")
		var command string
		fmt.Scanln(&command)

		args := strings.Fields(command)
		if len(args) == 0 {
			continue
		}

		switch args[0] {
		case "quit", "q", "exit":
			fmt.Println("Goodbye!")
			return
		case "help", "h":
			fmt.Println("Available commands:")
			fmt.Println("  list models     - List all models")
			fmt.Println("  list providers  - List all providers")
			fmt.Println("  verify [id]     - Verify a specific model")
			fmt.Println("  status          - Show system status")
			fmt.Println("  help            - Show this help")
			fmt.Println("  quit            - Exit interactive mode")
		case "list":
			if len(args) < 2 {
				fmt.Println("Usage: list models|providers")
				continue
			}
			switch args[1] {
			case "models":
				models, err := client.GetModels()
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					continue
				}
				printModelsTable(models)
			case "providers":
				providers, err := client.GetProviders()
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					continue
				}
				printProvidersTable(providers)
			default:
				fmt.Println("Usage: list models|providers")
			}
		case "verify":
			if len(args) < 2 {
				fmt.Println("Usage: verify [model_id]")
				continue
			}
			result, err := client.VerifyModel(args[1])
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				continue
			}
			fmt.Println("Verification completed:")
			if err := printJSON(result); err != nil {
				fmt.Printf("Error displaying result: %v\n", err)
			}
		case "status":
			models, err := client.GetModels()
			if err != nil {
				fmt.Printf("Error getting models: %v\n", err)
				continue
			}
			providers, err := client.GetProviders()
			if err != nil {
				fmt.Printf("Error getting providers: %v\n", err)
				continue
			}
			fmt.Printf("System Status:\n")
			fmt.Printf("  Models: %d\n", len(models))
			fmt.Printf("  Providers: %d\n", len(providers))
			fmt.Printf("  API Server: Connected\n")
		default:
			fmt.Printf("Unknown command: %s\n", args[0])
		}
		fmt.Println()
	}
}

func usersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "users",
		Short: "Manage users",
		Long:  `Create, list, and manage system users.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "create [username] [password] [email] [full_name]",
		Short: "Create a new user",
		Args:  cobra.MinimumNArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			username := args[0]
			password := args[1]
			email := args[2]
			fullName := ""
			if len(args) > 3 {
				fullName = args[3]
			}

			// Initialize database
			db, err := database.New("llm-verifier.db")
			if err != nil {
				log.Fatalf("Failed to initialize database: %v", err)
			}
			defer db.Close()

			// Create user with plain text password
			// The CreateUser method will hash it
			user := &database.User{
				Username:     username,
				Email:        email,
				PasswordHash: password,
				FullName:     fullName,
				Role:         "admin",
				IsActive:     true,
			}

			err = db.CreateUser(user)
			if err != nil {
				log.Fatalf("Failed to create user: %v", err)
			}

			fmt.Printf("User '%s' created successfully with ID: %d\n", username, user.ID)
			fmt.Println("Role: admin")
			fmt.Println("Status: active")
		},
	})

	return cmd
}
