package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"digital.vasic.llmsverifier/client"
	"digital.vasic.llmsverifier/tui"
)

func main() {
	var (
		serverURL = flag.String("server", "http://localhost:8080", "LLM Verifier server URL")
		token     = flag.String("token", "", "Authentication token")
		username  = flag.String("username", "", "Username for authentication")
		password  = flag.String("password", "", "Password for authentication")
	)
	flag.Parse()

	// Create API client
	client := client.New(*serverURL)

	// Handle authentication
	if *token != "" {
		client.SetToken(*token)
	} else if *username != "" && *password != "" {
		fmt.Println("🔐 Authenticating...")
		if err := client.Login(*username, *password); err != nil {
			fmt.Printf("❌ Authentication failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ Authentication successful")
	} else {
		fmt.Println("⚠️  Running without authentication")
	}

	// Create TUI application
	app := tui.NewApp(client)

	// Run the TUI
	fmt.Println("🚀 Starting LLM Verifier TUI...")
	fmt.Println("Press 'q' to quit, '1-4' to switch screens, '←/→' to navigate")
	fmt.Println()

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("❌ Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
