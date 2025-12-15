package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

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
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			models, err := c.GetModels()
			if err != nil {
				log.Fatalf("Failed to get models: %v", err)
			}

			if err := printJSON(models); err != nil {
				log.Fatalf("Failed to print models: %v", err)
			}
		},
	})

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
		Use:   "create",
		Short: "Create a new model",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Creating model... (interactive creation not implemented)")
			fmt.Println("Use the API directly or implement interactive creation")
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

	cmd.AddCommand(&cobra.Command{
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

			if err := printJSON(providers); err != nil {
				log.Fatalf("Failed to print providers: %v", err)
			}
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
			c, err := getClient()
			if err != nil {
				log.Fatalf("Failed to create client: %v", err)
			}

			results, err := c.GetVerificationResults()
			if err != nil {
				log.Fatalf("Failed to get verification results: %v", err)
			}

			if err := printJSON(results); err != nil {
				log.Fatalf("Failed to print verification results: %v", err)
			}
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
	// Start the TUI application
	tui.Run()
	return nil
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
