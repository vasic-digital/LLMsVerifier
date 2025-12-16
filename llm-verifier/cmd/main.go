package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"llm-verifier/api"
	"llm-verifier/client"
	"llm-verifier/database"
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

	// Export subcommands
	rootCmd.AddCommand(aiConfigCmd())
	// rootCmd.AddCommand(modelsCmd()) // TODO: Implement modelsCmd
	// Providers commands
	// rootCmd.AddCommand(providersCmd()) // TODO: Implement providersCmd
	// Verification results commands
	// rootCmd.AddCommand(resultsCmd()) // TODO: Implement resultsCmd
	// Pricing commands
	// rootCmd.AddCommand(pricingCmd()) // TODO: Implement pricingCmd
	// Limits commands
	// rootCmd.AddCommand(limitsCmd()) // TODO: Implement limitsCmd
	// Issues commands
	// rootCmd.AddCommand(issuesCmd()) // TODO: Implement issuesCmd
	// Events commands
	// rootCmd.AddCommand(eventsCmd()) // TODO: Implement eventsCmd
	// Schedules commands
	// rootCmd.AddCommand(schedulesCmd()) // TODO: Implement schedulesCmd
	// Exports commands
	// rootCmd.AddCommand(exportsCmd()) // TODO: Implement exportsCmd
	// Logs commands
	// rootCmd.AddCommand(logsCmd()) // TODO: Implement logsCmd
	// Config commands
	// rootCmd.AddCommand(configCmd()) // TODO: Implement configCmd
	// Batch commands
	// rootCmd.AddCommand(batchCmd()) // TODO: Implement batchCmd
	// TUI commands
	// rootCmd.AddCommand(tuiCmd()) // TODO: Implement tuiCmd

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

	_ = &cobra.Command{
		Use:   "list",
		Short: "List all models",
		Run: func(cmd *cobra.Command, args []string) {
			_, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

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
			_, err := getClient()
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
			_, err := getClient()
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
		Use:   "ai-config [format] [output_file]",
		Short: "Export AI CLI agent configurations",
		Long:  `Export models in AI CLI agent formats (opencode, crush, claude-code).`,
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			runAIConfigExport(args)
		},
	}

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

	// Load configuration from database or use mock data
	// For now, use mock data since we don't have database integration yet
	fmt.Printf("📤 Exporting AI CLI configuration for format: %s\n", format)
	fmt.Printf("📄 Output file: %s\n", outputFile)

	// Create export options
	options := &llmverifier.ExportOptions{
		Top:           5,
		MinScore:      70.0,
		IncludeAPIKey: false,
	}

	// Export configuration
	err := llmverifier.ExportAIConfig(nil, format, outputFile, options)
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
			_, err := getClient()
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
			_, err := getClient()
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

	modelsCmd := &cobra.Command{
		Use:   "models [output_file]",
		Short: "Export models data",
		Long:  `Export all models data to a JSON file.`,
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			_, err := getClient()
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
			_, err := getClient()
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
			_, err := getClient()
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
