package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/bubbletea"
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

	// Export subcommands
	rootCmd.AddCommand(aiConfigCmd())
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
	// Exports commands
	rootCmd.AddCommand(exportsCmd())
	// Logs commands
	rootCmd.AddCommand(logsCmd())
	// Config commands
	rootCmd.AddCommand(configCmd())
	// Batch commands
	rootCmd.AddCommand(batchCmd())
	// TUI commands
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
			port, _ := cmd.Flags().GetString("port")
			if err := runServer(port); err != nil {
				log.Fatalf("Error starting server: %v", err)
			}
		},
	}

	cmd.Flags().String("port", "8080", "Port to run the server on")
	return cmd
}

func tuiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Start the Terminal User Interface",
		Long:  `Start the interactive terminal user interface for managing models, providers, and verification results.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := runTUI(); err != nil {
				log.Fatalf("Error starting TUI: %v", err)
			}
		},
	}

	cmd.Flags().String("server-url", "http://localhost:8080", "Server URL to connect to")
	return cmd
}

func runServer(port string) error {
	cfg, err := llmverifier.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	server := api.NewServer(cfg, nil)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Use provided port, or config port, or default
	if port == "" {
		port = cfg.API.Port
		if port == "" {
			port = "8080"
		}
	}

	return server.Start()
}

func runTUI() error {
	// Get server URL from flag (if implemented) or use default
	serverURL := "http://localhost:8080"

	// Create client
	c := client.New(serverURL)

	// Create and run TUI app
	app := tui.NewApp(c)
	p := tea.NewProgram(app, tea.WithAltScreen())

	_, err := p.Run()
	return err
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
				log.Fatalf("Failed to fetch models: %v", err)
			}

			if len(models) == 0 {
				fmt.Println("No models found.")
				return
			}

			fmt.Printf("Found %d models:\n\n", len(models))
			for i, model := range models {
				fmt.Printf("%d. Name: %v\n", i+1, model["name"])
				fmt.Printf("   Provider: %v\n", model["provider_name"])
				fmt.Printf("   Status: %v\n", model["verification_status"])
				fmt.Printf("   Score: %.2f\n\n", model["overall_score"])
			}
		},
	}
	listCmd.Flags().String("filter", "", "Filter models by provider name")
	listCmd.Flags().String("format", "table", "Output format (table, json)")
	listCmd.Flags().Int("limit", 0, "Limit number of results (0 for unlimited)")

	createCmd := &cobra.Command{
		Use:   "create [name] [provider]",
		Short: "Create a new model",
		Args:  cobra.MinimumNArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			name := args[0]
			provider := args[1]
			description, _ := cmd.Flags().GetString("description")
			version, _ := cmd.Flags().GetString("version")

			model := map[string]interface{}{
				"name":        name,
				"provider":    provider,
				"description": description,
				"version":     version,
			}

			result, err := c.CreateModel(model)
			if err != nil {
				log.Fatalf("Failed to create model: %v", err)
			}

			fmt.Printf("Model created successfully\n")
			fmt.Printf("ID: %v\n", result["id"])
			fmt.Printf("Name: %v\n", result["name"])
			fmt.Printf("Provider: %v\n", result["provider"])
		},
	}
	createCmd.Flags().String("description", "", "Model description")
	createCmd.Flags().String("version", "1.0", "Model version")

	getCmd := &cobra.Command{
		Use:   "get [model-id]",
		Short: "Get model details",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			modelID := args[0]
			model, err := c.GetModel(modelID)
			if err != nil {
				log.Fatalf("Failed to fetch model: %v", err)
			}

			fmt.Printf("Model Details:\n")
			fmt.Printf("ID: %v\n", model["id"])
			fmt.Printf("Name: %v\n", model["name"])
			fmt.Printf("Provider: %v\n", model["provider_name"])
			fmt.Printf("Status: %v\n", model["verification_status"])
			fmt.Printf("Score: %.2f\n", model["overall_score"])
			fmt.Printf("Description: %v\n", model["description"])
		},
	}

	verifyCmd := &cobra.Command{
		Use:   "verify [model-id]",
		Short: "Trigger verification for a model",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			modelID := args[0]
			result, err := c.VerifyModel(modelID)
			if err != nil {
				log.Fatalf("Failed to verify model: %v", err)
			}

			fmt.Printf("Verification started for model %s\n", modelID)
			fmt.Printf("Status: %v\n", result["status"])
		},
	}

	cmd.AddCommand(listCmd)
	cmd.AddCommand(createCmd)
	cmd.AddCommand(getCmd)
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

// AI CLI export command
func aiConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ai-config",
		Short: "Export and validate AI CLI agent configurations",
		Long:  `Export models in AI CLI agent formats (opencode, crush, claude-code) and validate configurations.`,
	}

	// Single format export
	exportCmd := &cobra.Command{
		Use:   "export [format] [output_file]",
		Short: "Export configuration for a specific AI CLI agent",
		Long:  `Export models in a specific AI CLI agent format (opencode, crush, claude-code).`,
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			runAIConfigExport(args)
		},
	}

	// Bulk export
	bulkCmd := &cobra.Command{
		Use:   "bulk [output_directory]",
		Short: "Export configurations for all AI CLI agents",
		Long:  `Export configurations for all supported AI CLI agents (opencode, crush, claude-code) to the specified directory.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runAIBulkExport(args[0])
		},
	}

	// Validate command
	validateCmd := &cobra.Command{
		Use:   "validate [config_file]",
		Short: "Validate an exported AI CLI configuration",
		Long:  `Validate the structure and content of an exported AI CLI configuration file.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runAIConfigValidate(args[0])
		},
	}

	cmd.AddCommand(exportCmd, bulkCmd, validateCmd)
	return cmd
}

func runAIConfigExport(args []string) {
	// Parse arguments
	var format, outputFile string
	switch len(args) {
	case 0:
		format = "opencode"        // default
		outputFile = "export.json" // default
	case 1:
		format = args[0]
		outputFile = "export.json" // default
	case 2:
		format = args[0]
		outputFile = args[1]
	default:
		fmt.Printf("❌ Too many arguments. Usage: ai-config [format] [output_file]\n")
		os.Exit(1)
	}
	if len(args) >= 2 {
		outputFile = args[1]
	}

	// Validate format
	supportedFormats := []string{"opencode", "crush", "claude-code"}
	if !contains(supportedFormats, format) {
		fmt.Printf("❌ Unsupported format: %s\n", format)
		fmt.Printf("Supported formats: %v\n", supportedFormats)
		os.Exit(1)
	}

	// Initialize database
	db, err := database.New("llm-verifier.db")
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer db.Close()

	fmt.Printf("📤 Exporting AI CLI configuration for format: %s\n", format)
	fmt.Printf("📄 Output file: %s\n", outputFile)

	// Load configuration
	cfg, err := llmverifier.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	// Create export options
	options := &llmverifier.ExportOptions{
		Top:           5,
		MinScore:      40.0, // Lower threshold to include challenge-verified models
		IncludeAPIKey: false,
	}

	// Export configuration
	err = llmverifier.ExportAIConfig(db, cfg, format, outputFile, options)
	if err != nil {
		log.Fatalf("❌ Failed to export %s configuration: %v", format, err)
	}

	fmt.Printf("✅ Successfully exported %s configuration to %s\n", format, outputFile)

	// Validate exported configuration
	fmt.Println("🔍 Validating exported configuration...")
	err = llmverifier.ValidateExportedConfig(outputFile)
	if err != nil {
		log.Printf("❌ Validation failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Configuration validation passed")
}

func runAIBulkExport(outputDir string) {
	// Initialize database
	db, err := database.New("llm-verifier.db")
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Load configuration
	cfg, err := llmverifier.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	fmt.Printf("📤 Exporting AI CLI configurations to directory: %s\n", outputDir)

	// Create export options
	options := &llmverifier.ExportOptions{
		Top:           10,
		MinScore:      70.0,
		IncludeAPIKey: false,
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("❌ Failed to create output directory: %v", err)
	}

	// Export all configurations
	err = llmverifier.ExportBulkConfig(db, cfg, outputDir, options)
	if err != nil {
		log.Fatalf("❌ Failed to export configurations: %v", err)
	}

	fmt.Printf("✅ Successfully exported all AI CLI configurations to %s\n", outputDir)
	fmt.Println("📄 Generated files:")
	fmt.Println("  - export_opencode.json")
	fmt.Println("  - export_crush.json")
	fmt.Println("  - export_claude_code.json")
	fmt.Println("  - export_summary.json")
	fmt.Println("  - export_manifest.json")
}

func runAIConfigValidate(configPath string) {
	fmt.Printf("🔍 Validating AI CLI configuration: %s\n", configPath)

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("❌ Configuration file does not exist: %s", configPath)
	}

	// Validate configuration
	err := llmverifier.ValidateExportedConfig(configPath)
	if err != nil {
		log.Fatalf("❌ Validation failed: %v", err)
	}

	fmt.Println("✅ Configuration validation passed")
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func exportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export [format] [output_file]",
		Short: "Export data and AI CLI configurations",
		Long:  `Export models, providers, configurations, and verification results. Supports AI CLI formats: opencode, crush, claude-code.`,
		Args:  cobra.RangeArgs(1, 2),
	}

	// Models export command
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

	// Providers export command
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
			_, err := getClient()
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

func providersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "Manage LLM providers",
		Long:  `List, create, update, and delete LLM provider configurations.`,
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
				log.Fatalf("Failed to fetch providers: %v", err)
			}

			if len(providers) == 0 {
				fmt.Println("No providers found.")
				return
			}

			fmt.Printf("Found %d providers:\n\n", len(providers))
			for i, provider := range providers {
				fmt.Printf("%d. Name: %v\n", i+1, provider["name"])
				fmt.Printf("   Endpoint: %v\n", provider["endpoint"])
				fmt.Printf("   Status: %v\n", provider["status"])
				fmt.Printf("   Created: %v\n\n", provider["created_at"])
			}
		},
	}
	listCmd.Flags().String("filter", "", "Filter providers by name")
	listCmd.Flags().String("format", "table", "Output format (table, json)")
	listCmd.Flags().Int("limit", 0, "Limit number of results (0 for unlimited)")

	getCmd := &cobra.Command{
		Use:   "get [provider-id]",
		Short: "Get provider details",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			providerID := args[0]
			provider, err := c.GetProvider(providerID)
			if err != nil {
				log.Fatalf("Failed to fetch provider: %v", err)
			}

			fmt.Printf("Provider Details:\n")
			fmt.Printf("ID: %v\n", provider["id"])
			fmt.Printf("Name: %v\n", provider["name"])
			fmt.Printf("Status: %v\n", provider["status"])
			fmt.Printf("Model Count: %v\n", provider["model_count"])
			fmt.Printf("Average Score: %.2f\n", provider["avg_score"])
			fmt.Printf("Description: %v\n", provider["description"])
		},
	}

	statusCmd := &cobra.Command{
		Use:   "status [provider-name]",
		Short: "Get provider status",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			providers, err := c.GetProviders()
			if err != nil {
				log.Fatalf("Failed to fetch providers: %v", err)
			}

			if len(args) == 0 {
				// Show status for all providers
				for _, provider := range providers {
					fmt.Printf("Provider: %v - Status: %v\n", provider["name"], provider["status"])
				}
				return
			}

			// Show status for specific provider
			providerName := args[0]
			found := false
			for _, provider := range providers {
				if name, ok := provider["name"]; ok && name == providerName {
					fmt.Printf("Provider: %v\n", name)
					fmt.Printf("  Status: %v\n", provider["status"])
					fmt.Printf("  Endpoint: %v\n", provider["endpoint"])
					found = true
					break
				}
			}

			if !found {
				fmt.Printf("Provider '%s' not found\n", providerName)
			}
		},
	}

	cmd.AddCommand(listCmd)
	cmd.AddCommand(getCmd)
	cmd.AddCommand(statusCmd)

	return cmd
}

func resultsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "results",
		Short: "Manage verification results",
		Long:  `View, export, and manage LLM verification results.`,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List verification results",
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			results, err := c.GetVerificationResults()
			if err != nil {
				log.Fatalf("Failed to fetch verification results: %v", err)
			}

			if len(results) == 0 {
				fmt.Println("No verification results found.")
				return
			}

			fmt.Printf("Found %d verification results:\n\n", len(results))
			for i, result := range results {
				fmt.Printf("%d. ID: %v\n", i+1, result["id"])
				fmt.Printf("   Model: %v\n", result["model_name"])
				fmt.Printf("   Score: %v\n", result["score"])
				fmt.Printf("   Status: %v\n", result["status"])
				fmt.Printf("   Created: %v\n\n", result["created_at"])
			}
		},
	}
	listCmd.Flags().String("filter", "", "Filter results by model name")
	listCmd.Flags().String("format", "table", "Output format (table, json)")
	listCmd.Flags().Int("limit", 0, "Limit number of results (0 for unlimited)")

	getCmd := &cobra.Command{
		Use:   "get [result-id]",
		Short: "Get verification result details",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			resultID := args[0]
			result, err := c.GetVerificationResult(resultID)
			if err != nil {
				log.Fatalf("Failed to fetch verification result: %v", err)
			}

			fmt.Printf("Verification Result Details:\n")
			fmt.Printf("ID: %v\n", result["id"])
			fmt.Printf("Model: %v\n", result["model_name"])
			fmt.Printf("Score: %.2f\n", result["score"])
			fmt.Printf("Status: %v\n", result["status"])
			fmt.Printf("Created: %v\n", result["created_at"])
			fmt.Printf("Duration: %v\n", result["duration"])
			fmt.Printf("Error Message: %v\n", result["error_message"])
		},
	}

	exportCmd := &cobra.Command{
		Use:   "export [format] [output-file]",
		Short: "Export verification results",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			results, err := c.GetVerificationResults()
			if err != nil {
				log.Fatalf("Failed to fetch verification results: %v", err)
			}

			// Simple JSON export for now
			data, err := json.MarshalIndent(results, "", "  ")
			if err != nil {
				log.Fatalf("Failed to marshal results: %v", err)
			}

			err = os.WriteFile(args[1], data, 0644)
			if err != nil {
				log.Fatalf("Failed to write export file: %v", err)
			}

			fmt.Printf("Exported %d results to %s\n", len(results), args[1])
		},
	}
	cmd.AddCommand(listCmd)
	cmd.AddCommand(getCmd)
	cmd.AddCommand(exportCmd)

	return cmd
}

func pricingCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pricing",
		Short: "Manage pricing plans",
		Long:  `View and manage LLM service pricing plans.`,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List pricing plans",
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			pricing, err := c.GetPricing()
			if err != nil {
				log.Fatalf("Failed to fetch pricing plans: %v", err)
			}

			if len(pricing) == 0 {
				fmt.Println("No pricing plans found.")
				return
			}

			fmt.Printf("Found %d pricing plans:\n\n", len(pricing))
			for i, plan := range pricing {
				fmt.Printf("%d. Plan: %v\n", i+1, plan["name"])
				fmt.Printf("   Model: %v\n", plan["model_name"])
				fmt.Printf("   Input Cost: %v\n", plan["input_cost"])
				fmt.Printf("   Output Cost: %v\n", plan["output_cost"])
				fmt.Printf("   Currency: %v\n", plan["currency"])
				fmt.Printf("   Updated: %v\n\n", plan["updated_at"])
			}
		},
	}

	modelCmd := &cobra.Command{
		Use:   "model [model-id]",
		Short: "Get pricing for specific model",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			pricing, err := c.GetPricing()
			if err != nil {
				log.Fatalf("Failed to fetch pricing plans: %v", err)
			}

			modelID := args[0]
			found := false
			for _, plan := range pricing {
				if planID, ok := plan["model_id"]; ok && planID == modelID {
					fmt.Printf("Pricing for Model %s:\n", modelID)
					fmt.Printf("  Plan: %v\n", plan["name"])
					fmt.Printf("  Input Cost: %v\n", plan["input_cost"])
					fmt.Printf("  Output Cost: %v\n", plan["output_cost"])
					fmt.Printf("  Currency: %v\n", plan["currency"])
					found = true
					break
				}
			}

			if !found {
				fmt.Printf("No pricing found for model ID: %s\n", modelID)
			}
		},
	}

	cmd.AddCommand(listCmd)
	cmd.AddCommand(modelCmd)

	return cmd
}

func limitsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "limits",
		Short: "Manage rate limits",
		Long:  `View and configure API rate limits and quotas.`,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List rate limits",
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			limits, err := c.GetLimits()
			if err != nil {
				log.Fatalf("Failed to fetch rate limits: %v", err)
			}

			if len(limits) == 0 {
				fmt.Println("No rate limits configured. Default limits: 100 requests/minute")
				return
			}

			fmt.Printf("Found %d rate limits:\n\n", len(limits))
			for i, limit := range limits {
				fmt.Printf("%d. Type: %v\n", i+1, limit["type"])
				fmt.Printf("   Limit: %v\n", limit["limit"])
				fmt.Printf("   Window: %v\n", limit["window"])
				fmt.Printf("   Remaining: %v\n\n", limit["remaining"])
			}
		},
	}
	cmd.AddCommand(listCmd)

	return cmd
}

func issuesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "issues",
		Short: "Manage issue tracking",
		Long:  `Track and manage verification issues and anomalies.`,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List issues",
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			issues, err := c.GetIssues()
			if err != nil {
				log.Fatalf("Failed to fetch issues: %v", err)
			}

			if len(issues) == 0 {
				fmt.Println("No issues found")
				return
			}

			fmt.Printf("Found %d issues:\n\n", len(issues))
			for i, issue := range issues {
				fmt.Printf("%d. ID: %v\n", i+1, issue["id"])
				fmt.Printf("   Type: %v\n", issue["type"])
				fmt.Printf("   Severity: %v\n", issue["severity"])
				fmt.Printf("   Status: %v\n", issue["status"])
				fmt.Printf("   Description: %v\n\n", issue["description"])
			}
		},
	}
	cmd.AddCommand(listCmd)

	return cmd
}

func eventsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Manage system events",
		Long:  `View and manage system events and logs.`,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List events",
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			events, err := c.GetEvents()
			if err != nil {
				log.Fatalf("Failed to fetch events: %v", err)
			}

			if len(events) == 0 {
				fmt.Println("No events found")
				return
			}

			fmt.Printf("Found %d events:\n\n", len(events))
			for i, event := range events {
				fmt.Printf("%d. ID: %v\n", i+1, event["id"])
				fmt.Printf("   Type: %v\n", event["type"])
				fmt.Printf("   Severity: %v\n", event["severity"])
				fmt.Printf("   Timestamp: %v\n", event["timestamp"])
				fmt.Printf("   Message: %v\n\n", event["message"])
			}
		},
	}
	cmd.AddCommand(listCmd)

	return cmd
}

func schedulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schedules",
		Short: "Manage job schedules",
		Long:  `Configure and manage verification job schedules.`,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List schedules",
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			schedules, err := c.GetSchedules()
			if err != nil {
				log.Fatalf("Failed to fetch schedules: %v", err)
			}

			if len(schedules) == 0 {
				fmt.Println("No schedules found")
				return
			}

			fmt.Printf("Found %d schedules:\n\n", len(schedules))
			for i, schedule := range schedules {
				fmt.Printf("%d. ID: %v\n", i+1, schedule["id"])
				fmt.Printf("   Type: %v\n", schedule["type"])
				fmt.Printf("   Frequency: %v\n", schedule["frequency"])
				fmt.Printf("   Status: %v\n", schedule["status"])
				fmt.Printf("   Next Run: %v\n\n", schedule["next_run"])
			}
		},
	}
	cmd.AddCommand(listCmd)

	return cmd
}

func exportsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exports",
		Short: "Manage data exports",
		Long:  `Export verification results and configuration data.`,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List exports",
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			exports, err := c.GetConfigExports()
			if err != nil {
				log.Fatalf("Failed to fetch exports: %v", err)
			}

			if len(exports) == 0 {
				fmt.Println("No exports found")
				return
			}

			fmt.Printf("Found %d exports:\n\n", len(exports))
			for i, export := range exports {
				fmt.Printf("%d. ID: %v\n", i+1, export["id"])
				fmt.Printf("   Format: %v\n", export["format"])
				fmt.Printf("   Status: %v\n", export["status"])
				fmt.Printf("   Created: %v\n", export["created_at"])
				fmt.Printf("   Size: %v\n\n", export["size"])
			}
		},
	}
	cmd.AddCommand(listCmd)

	return cmd
}

func logsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Manage system logs",
		Long:  `View and manage system logs and audit trails.`,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List logs",
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			logs, err := c.GetLogs()
			if err != nil {
				log.Fatalf("Failed to fetch logs: %v", err)
			}

			if len(logs) == 0 {
				fmt.Println("No logs found")
				return
			}

			fmt.Printf("Found %d log entries:\n\n", len(logs))
			for i, logEntry := range logs {
				fmt.Printf("%d. Level: %v\n", i+1, logEntry["level"])
				fmt.Printf("   Message: %v\n", logEntry["message"])
				fmt.Printf("   Timestamp: %v\n", logEntry["timestamp"])
				fmt.Printf("   Component: %v\n\n", logEntry["component"])
			}
		},
	}
	cmd.AddCommand(listCmd)

	return cmd
}

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `View and update system configuration settings.`,
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Show configuration",
		Run: func(cmd *cobra.Command, args []string) {
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			config, err := c.GetConfig()
			if err != nil {
				log.Fatalf("Failed to fetch configuration: %v", err)
			}

			fmt.Printf("System Configuration:\n\n")
			for key, value := range config {
				fmt.Printf("%-20s: %v\n", key, value)
			}
		},
	}
	cmd.AddCommand(listCmd)

	return cmd
}

func batchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "batch",
		Short: "Manage batch operations",
		Long:  `Execute and manage batch verification operations.`,
	}

	runCmd := &cobra.Command{
		Use:   "run [batch-file]",
		Short: "Run batch verification",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Batch verification not yet fully implemented")
			fmt.Printf("Would process batch file: %s\n", args[0])
			fmt.Println("Use main verify command for individual verifications")
		},
	}
	cmd.AddCommand(runCmd)

	return cmd
}
