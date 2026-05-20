package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"digital.vasic.llmsverifier/partners"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "list":
		handleList()
	case "add":
		handleAdd()
	case "remove":
		handleRemove()
	case "sync":
		handleSync()
	case "status":
		handleStatus()
	default:
		fmt.Println(trData("llmsverifier_partners_unknown_command", map[string]any{"command": command}))
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(tr("llmsverifier_partners_usage_title"))
	fmt.Println()
	fmt.Println(tr("llmsverifier_partners_usage_heading"))
	fmt.Println(tr("llmsverifier_partners_usage_list"))
	fmt.Println(tr("llmsverifier_partners_usage_add"))
	fmt.Println(tr("llmsverifier_partners_usage_remove"))
	fmt.Println(tr("llmsverifier_partners_usage_sync"))
	fmt.Println(tr("llmsverifier_partners_usage_status"))
	fmt.Println()
	fmt.Println(tr("llmsverifier_partners_types_heading"))
	fmt.Println(tr("llmsverifier_partners_type_opencode"))
	fmt.Println(tr("llmsverifier_partners_type_claude_code"))
	fmt.Println(tr("llmsverifier_partners_type_cursor"))
	fmt.Println(tr("llmsverifier_partners_type_vscode"))
	fmt.Println(tr("llmsverifier_partners_type_jetbrains"))
	fmt.Println(tr("llmsverifier_partners_type_github"))
	fmt.Println()
	fmt.Println(tr("llmsverifier_partners_examples_heading"))
	fmt.Println(tr("llmsverifier_partners_example_add"))
	fmt.Println(tr("llmsverifier_partners_example_sync"))
}

func handleList() {
	// In a real implementation, load from persistent storage
	// For demo, show predefined integrations
	integrations := getPredefinedIntegrations()

	fmt.Println(tr("llmsverifier_partners_list_heading"))
	fmt.Println("====================")

	for _, integration := range integrations {
		status := "✓"
		if integration.Status != "active" {
			status = "✗"
		}

		fmt.Printf("%s %s (%s) - %s\n", status, integration.Name, integration.Type, integration.Description)
		if integration.LastSync != nil {
			fmt.Println(trData("llmsverifier_partners_last_sync_indented", map[string]any{
				"time": integration.LastSync.Format("2006-01-02 15:04:05"),
			}))
		}
		fmt.Println()
	}
}

func handleAdd() {
	if len(os.Args) < 4 {
		fmt.Println(tr("llmsverifier_partners_err_add_args"))
		os.Exit(1)
	}

	integrationType := os.Args[2]
	name := os.Args[3]

	// Parse configuration from remaining arguments
	config := make(map[string]interface{})
	for i := 4; i < len(os.Args); i++ {
		arg := os.Args[i]
		if strings.HasPrefix(arg, "--") {
			key := strings.TrimPrefix(arg, "--")
			if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "--") {
				config[key] = os.Args[i+1]
				i++ // Skip next arg as it's the value
			} else {
				config[key] = true // Flag without value
			}
		}
	}

	// Create integration based on type
	var integration *partners.PartnerIntegration
	var err error

	switch integrationType {
	case "opencode":
		integration, err = createOpenCodeIntegration(name, config)
	case "claude_code":
		integration, err = createClaudeCodeIntegration(name, config)
	case "cursor":
		integration, err = createCursorIntegration(name, config)
	default:
		err = fmt.Errorf("unsupported integration type: %s", integrationType)
	}

	if err != nil {
		log.Fatalf("Failed to create integration: %v", err)
	}

	// In real implementation, register with manager
	// For demo, just show success message

	fmt.Println(trData("llmsverifier_partners_added_integration", map[string]any{
		"name": integration.Name, "id": integration.ID,
	}))
	fmt.Println(trData("llmsverifier_partners_field_type", map[string]any{"type": integration.Type}))
	fmt.Println(trData("llmsverifier_partners_field_capabilities", map[string]any{"capabilities": integration.Capabilities}))
}

func handleRemove() {
	if len(os.Args) < 3 {
		fmt.Println(tr("llmsverifier_partners_err_remove_args"))
		os.Exit(1)
	}

	id := os.Args[2]
	fmt.Println(trData("llmsverifier_partners_removed_integration", map[string]any{"id": id}))
	// In real implementation, remove from storage
}

func handleSync() {
	if len(os.Args) < 3 {
		fmt.Println(tr("llmsverifier_partners_err_sync_args"))
		os.Exit(1)
	}

	nameOrID := os.Args[2]

	// Find integration (in real implementation, load from storage)
	integrations := getPredefinedIntegrations()
	var targetIntegration *partners.PartnerIntegration

	for _, integration := range integrations {
		if integration.ID == nameOrID || strings.Contains(integration.Name, nameOrID) {
			targetIntegration = integration
			break
		}
	}

	if targetIntegration == nil {
		log.Fatalf("Integration not found: %s", nameOrID)
	}

	fmt.Println(trData("llmsverifier_partners_syncing_integration", map[string]any{"name": targetIntegration.Name}))

	// Create manager and register integration for sync
	manager := partners.NewIntegrationManager()
	if err := manager.RegisterIntegration(targetIntegration); err != nil {
		log.Fatalf("Failed to register integration: %v", err)
	}

	// Perform sync
	err := manager.SyncIntegration(nil, targetIntegration.ID)
	if err != nil {
		log.Fatalf("Sync failed: %v", err)
	}

	fmt.Println(trData("llmsverifier_partners_synced_success", map[string]any{"name": targetIntegration.Name}))
	if targetIntegration.LastSync != nil {
		fmt.Println(trData("llmsverifier_partners_last_sync", map[string]any{
			"time": targetIntegration.LastSync.Format("2006-01-02 15:04:05"),
		}))
	}
}

func handleStatus() {
	if len(os.Args) < 3 {
		fmt.Println(tr("llmsverifier_partners_err_status_args"))
		os.Exit(1)
	}

	id := os.Args[2]

	// In real implementation, get from storage
	integrations := getPredefinedIntegrations()
	var targetIntegration *partners.PartnerIntegration

	for _, integration := range integrations {
		if integration.ID == id || strings.Contains(integration.Name, id) {
			targetIntegration = integration
			break
		}
	}

	if targetIntegration == nil {
		log.Fatalf("Integration not found: %s", id)
	}

	fmt.Println(trData("llmsverifier_partners_status_heading", map[string]any{"name": targetIntegration.Name}))
	fmt.Println(trData("llmsverifier_partners_field_id", map[string]any{"id": targetIntegration.ID}))
	fmt.Println(trData("llmsverifier_partners_field_type", map[string]any{"type": targetIntegration.Type}))
	fmt.Println(trData("llmsverifier_partners_field_status", map[string]any{"status": targetIntegration.Status}))
	fmt.Println(trData("llmsverifier_partners_field_version", map[string]any{"version": targetIntegration.Version}))

	if targetIntegration.LastSync != nil {
		fmt.Println(trData("llmsverifier_partners_field_last_sync", map[string]any{
			"time": targetIntegration.LastSync.Format("2006-01-02 15:04:05"),
		}))
	}

	if targetIntegration.ErrorMessage != "" {
		fmt.Println(trData("llmsverifier_partners_field_error", map[string]any{"error": targetIntegration.ErrorMessage}))
	}

	fmt.Println(trData("llmsverifier_partners_field_capabilities", map[string]any{"capabilities": targetIntegration.Capabilities}))
}

// Helper functions

func createOpenCodeIntegration(name string, config map[string]interface{}) (*partners.PartnerIntegration, error) {
	baseURL, _ := config["base-url"].(string)
	apiKey, _ := config["api-key"].(string)
	projectID, _ := config["project-id"].(string)

	if baseURL == "" || apiKey == "" {
		return nil, fmt.Errorf("base-url and api-key are required for OpenCode integration")
	}

	return &partners.PartnerIntegration{
		Name:        name,
		Type:        partners.PartnerTypeOpenCode,
		Description: "OpenCode AI assistant integration",
		Version:     "1.0.0",
		Status:      partners.IntegrationStatusActive,
		Configuration: map[string]interface{}{
			"base_url":   baseURL,
			"api_key":    apiKey,
			"project_id": projectID,
		},
		Capabilities: []string{
			"model_sync",
			"test_results",
			"analytics",
			"real_time_updates",
		},
		AuthMethods: []string{"api_key", "oauth"},
		APIEndpoints: map[string]string{
			"models":    "/api/models",
			"tests":     "/api/tests",
			"analytics": "/api/analytics",
		},
	}, nil
}

func createClaudeCodeIntegration(name string, config map[string]interface{}) (*partners.PartnerIntegration, error) {
	apiKey, _ := config["api-key"].(string)
	workspace, _ := config["workspace"].(string)

	if apiKey == "" {
		return nil, fmt.Errorf("api-key is required for Claude Code integration")
	}

	return &partners.PartnerIntegration{
		Name:        name,
		Type:        partners.PartnerTypeClaudeCode,
		Description: "Anthropic Claude Code integration",
		Version:     "1.0.0",
		Status:      partners.IntegrationStatusActive,
		Configuration: map[string]interface{}{
			"api_key":   apiKey,
			"workspace": workspace,
		},
		Capabilities: []string{
			"model_sync",
			"code_generation",
			"code_review",
			"real_time_collaboration",
		},
		AuthMethods: []string{"api_key"},
		APIEndpoints: map[string]string{
			"models": "/api/models",
			"code":   "/api/code",
			"review": "/api/review",
		},
	}, nil
}

func createCursorIntegration(name string, config map[string]interface{}) (*partners.PartnerIntegration, error) {
	apiKey, _ := config["api-key"].(string)
	userID, _ := config["user-id"].(string)

	if apiKey == "" {
		return nil, fmt.Errorf("api-key is required for Cursor integration")
	}

	return &partners.PartnerIntegration{
		Name:        name,
		Type:        partners.PartnerTypeCursor,
		Description: "Cursor IDE integration",
		Version:     "1.0.0",
		Status:      partners.IntegrationStatusActive,
		Configuration: map[string]interface{}{
			"api_key": apiKey,
			"user_id": userID,
		},
		Capabilities: []string{
			"model_sync",
			"completion_sync",
			"code_intelligence",
			"real_time_suggestions",
		},
		AuthMethods: []string{"api_key", "oauth"},
		APIEndpoints: map[string]string{
			"models":       "/api/models",
			"completions":  "/api/completions",
			"intelligence": "/api/intelligence",
		},
	}, nil
}

func getPredefinedIntegrations() []*partners.PartnerIntegration {
	return []*partners.PartnerIntegration{
		{
			ID:          "opencode-demo",
			Name:        "OpenCode Demo",
			Type:        partners.PartnerTypeOpenCode,
			Description: "Demo OpenCode integration",
			Version:     "1.0.0",
			Status:      partners.IntegrationStatusActive,
			Configuration: map[string]interface{}{
				"base_url":   "https://api.opencode.example.com",
				"api_key":    "demo-api-key-123",
				"project_id": "demo-project",
			},
			Capabilities: []string{"model_sync", "test_results", "analytics"},
			AuthMethods:  []string{"api_key"},
		},
		{
			ID:          "claude-code-demo",
			Name:        "Claude Code Demo",
			Type:        partners.PartnerTypeClaudeCode,
			Description: "Demo Claude Code integration",
			Version:     "1.0.0",
			Status:      partners.IntegrationStatusActive,
			Configuration: map[string]interface{}{
				"api_key":   "demo-claude-key-456",
				"workspace": "demo-workspace",
			},
			Capabilities: []string{"model_sync", "code_generation"},
			AuthMethods:  []string{"api_key"},
		},
		{
			ID:          "cursor-demo",
			Name:        "Cursor Demo",
			Type:        partners.PartnerTypeCursor,
			Description: "Demo Cursor integration",
			Version:     "1.0.0",
			Status:      partners.IntegrationStatusActive,
			Configuration: map[string]interface{}{
				"api_key": "demo-cursor-key-789",
				"user_id": "demo-user",
			},
			Capabilities: []string{"model_sync", "completions"},
			AuthMethods:  []string{"api_key"},
		},
	}
}
