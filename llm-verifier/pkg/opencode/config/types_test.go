package opencode_config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== Config Type Tests ====================

func TestConfig_Structure(t *testing.T) {
	config := &Config{
		Plugin:     []string{"plugin1", "plugin2"},
		Enterprise: &EnterpriseConfig{URL: "https://enterprise.example.com"},
		Instructions: []string{"instruction1"},
		Provider: map[string]ProviderConfig{
			"openai": {Model: "gpt-4"},
		},
		Mcp: map[string]McpConfig{
			"test": {Type: "local"},
		},
		Tools: map[string]interface{}{"tool1": true},
		Agent: map[string]AgentConfig{
			"build": {Model: "gpt-4"},
		},
		Command: map[string]CommandConfig{
			"test": {Template: "echo test"},
		},
		Keybinds: &KeybindsConfig{Leader: ","},
		Username: "testuser",
		Permission: &PermissionConfig{
			Edit: "ask",
		},
		Compaction: &CompactionConfig{
			Auto: boolPtr(true),
		},
		Sse: &SseConfig{
			Enabled: boolPtr(true),
		},
	}

	assert.Len(t, config.Plugin, 2)
	assert.NotNil(t, config.Enterprise)
	assert.Equal(t, "https://enterprise.example.com", config.Enterprise.URL)
	assert.Len(t, config.Instructions, 1)
	assert.Len(t, config.Provider, 1)
	assert.Len(t, config.Mcp, 1)
	assert.Len(t, config.Tools, 1)
	assert.Len(t, config.Agent, 1)
	assert.Len(t, config.Command, 1)
	assert.NotNil(t, config.Keybinds)
	assert.Equal(t, ",", config.Keybinds.Leader)
	assert.Equal(t, "testuser", config.Username)
	assert.NotNil(t, config.Permission)
	assert.Equal(t, "ask", config.Permission.Edit)
	assert.NotNil(t, config.Compaction)
	assert.True(t, *config.Compaction.Auto)
	assert.NotNil(t, config.Sse)
	assert.True(t, *config.Sse.Enabled)
}

func TestConfig_JSONSerialization(t *testing.T) {
	config := &Config{
		Provider: map[string]ProviderConfig{
			"openai": {
				Model: "gpt-4",
				Options: map[string]interface{}{
					"api_key": "sk-test123",
				},
			},
		},
	}

	data, err := json.Marshal(config)
	require.NoError(t, err)

	var parsed Config
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Contains(t, parsed.Provider, "openai")
	assert.Equal(t, "gpt-4", parsed.Provider["openai"].Model)
}

// ==================== ProviderConfig Tests ====================

func TestProviderConfig_Structure(t *testing.T) {
	pc := ProviderConfig{
		Model: "gpt-4-turbo",
		Options: map[string]interface{}{
			"api_key": "sk-test",
			"baseURL": "https://api.openai.com",
		},
	}

	assert.Equal(t, "gpt-4-turbo", pc.Model)
	assert.Contains(t, pc.Options, "api_key")
	assert.Contains(t, pc.Options, "baseURL")
}

// ==================== McpConfig Tests ====================

func TestMcpConfig_Local(t *testing.T) {
	timeout := 30
	enabled := true
	mcp := McpConfig{
		Type:    "local",
		Command: []string{"npx", "-y", "@modelcontextprotocol/server-filesystem"},
		Environment: map[string]string{
			"NODE_ENV": "production",
		},
		Enabled: &enabled,
		Timeout: &timeout,
	}

	assert.Equal(t, "local", mcp.Type)
	assert.Len(t, mcp.Command, 3)
	assert.Contains(t, mcp.Environment, "NODE_ENV")
	assert.True(t, *mcp.Enabled)
	assert.Equal(t, 30, *mcp.Timeout)
}

func TestMcpConfig_Remote(t *testing.T) {
	mcp := McpConfig{
		Type: "remote",
		URL:  "https://mcp.example.com",
		Headers: map[string]string{
			"Authorization": "Bearer token123",
		},
	}

	assert.Equal(t, "remote", mcp.Type)
	assert.Equal(t, "https://mcp.example.com", mcp.URL)
	assert.Contains(t, mcp.Headers, "Authorization")
}

// ==================== AgentConfig Tests ====================

func TestAgentConfig_Structure(t *testing.T) {
	temp := 0.7
	topP := 0.9
	maxSteps := 10
	disabled := false

	agent := AgentConfig{
		Model:       "gpt-4-turbo",
		Temperature: &temp,
		TopP:        &topP,
		Prompt:      "You are a helpful assistant.",
		Tools: map[string]bool{
			"Read":  true,
			"Write": true,
			"Bash":  false,
		},
		Disable:     &disabled,
		Description: "Build agent for development tasks",
		Mode:        "autonomous",
		Color:       "#FF5733",
		MaxSteps:    &maxSteps,
		Permission: map[string]interface{}{
			"bash": "allow",
		},
	}

	assert.Equal(t, "gpt-4-turbo", agent.Model)
	assert.Equal(t, 0.7, *agent.Temperature)
	assert.Equal(t, 0.9, *agent.TopP)
	assert.Equal(t, "You are a helpful assistant.", agent.Prompt)
	assert.Len(t, agent.Tools, 3)
	assert.True(t, agent.Tools["Read"])
	assert.False(t, agent.Tools["Bash"])
	assert.False(t, *agent.Disable)
	assert.Equal(t, "Build agent for development tasks", agent.Description)
	assert.Equal(t, "autonomous", agent.Mode)
	assert.Equal(t, "#FF5733", agent.Color)
	assert.Equal(t, 10, *agent.MaxSteps)
	assert.Contains(t, agent.Permission, "bash")
}

// ==================== CommandConfig Tests ====================

func TestCommandConfig_Structure(t *testing.T) {
	subtask := true
	cmd := CommandConfig{
		Template:    "echo Hello, ${name}!",
		Description: "Greets a user by name",
		Agent:       "build",
		Model:       "gpt-4",
		Subtask:     &subtask,
	}

	assert.Equal(t, "echo Hello, ${name}!", cmd.Template)
	assert.Equal(t, "Greets a user by name", cmd.Description)
	assert.Equal(t, "build", cmd.Agent)
	assert.Equal(t, "gpt-4", cmd.Model)
	assert.True(t, *cmd.Subtask)
}

// ==================== KeybindsConfig Tests ====================

func TestKeybindsConfig_Structure(t *testing.T) {
	kb := KeybindsConfig{
		Leader:       ",",
		AppExit:      "q",
		EditorOpen:   "e",
		SessionNew:   "n",
		SessionList:  "l",
		InputSubmit:  "ctrl+enter",
		InputNewline: "enter",
	}

	assert.Equal(t, ",", kb.Leader)
	assert.Equal(t, "q", kb.AppExit)
	assert.Equal(t, "e", kb.EditorOpen)
	assert.Equal(t, "n", kb.SessionNew)
	assert.Equal(t, "l", kb.SessionList)
	assert.Equal(t, "ctrl+enter", kb.InputSubmit)
	assert.Equal(t, "enter", kb.InputNewline)
}

// ==================== PermissionConfig Tests ====================

func TestPermissionConfig_Structure(t *testing.T) {
	pc := PermissionConfig{
		Edit:              "ask",
		Bash:              "allow",
		Skill:             map[string]string{"test": "allow"},
		Webfetch:          "ask",
		DoomLoop:          "deny",
		ExternalDirectory: "ask",
	}

	assert.Equal(t, "ask", pc.Edit)
	assert.Equal(t, "allow", pc.Bash)
	assert.NotNil(t, pc.Skill)
	assert.Equal(t, "ask", pc.Webfetch)
	assert.Equal(t, "deny", pc.DoomLoop)
	assert.Equal(t, "ask", pc.ExternalDirectory)
}

// ==================== CompactionConfig Tests ====================

func TestCompactionConfig_Structure(t *testing.T) {
	autoVal := true
	pruneVal := false

	cc := CompactionConfig{
		Auto:  &autoVal,
		Prune: &pruneVal,
	}

	assert.True(t, *cc.Auto)
	assert.False(t, *cc.Prune)
}

// ==================== SseConfig Tests ====================

func TestSseConfig_Structure(t *testing.T) {
	enabled := true
	sc := SseConfig{
		Enabled: &enabled,
	}

	assert.True(t, *sc.Enabled)
}

// ==================== EnterpriseConfig Tests ====================

func TestEnterpriseConfig_Structure(t *testing.T) {
	ec := EnterpriseConfig{
		URL: "https://enterprise.claude.ai",
	}

	assert.Equal(t, "https://enterprise.claude.ai", ec.URL)
}

// ==================== Full Config JSON Round-Trip Tests ====================

func TestConfig_CompleteJSONRoundTrip(t *testing.T) {
	temp := 0.5
	enabled := true
	timeout := 60

	original := &Config{
		Plugin: []string{"plugin1"},
		Enterprise: &EnterpriseConfig{
			URL: "https://enterprise.example.com",
		},
		Instructions: []string{"Be helpful"},
		Provider: map[string]ProviderConfig{
			"openai": {
				Model: "gpt-4",
				Options: map[string]interface{}{
					"api_key": "sk-test",
				},
			},
		},
		Mcp: map[string]McpConfig{
			"filesystem": {
				Type:    "local",
				Command: []string{"npx", "-y", "@modelcontextprotocol/server-filesystem"},
				Timeout: &timeout,
			},
		},
		Agent: map[string]AgentConfig{
			"coder": {
				Model:       "gpt-4",
				Temperature: &temp,
				Prompt:      "You are a coding assistant.",
			},
		},
		Command: map[string]CommandConfig{
			"build": {
				Template:    "npm run build",
				Description: "Build the project",
			},
		},
		Username: "testuser",
		Sse: &SseConfig{
			Enabled: &enabled,
		},
	}

	// Marshal
	data, err := json.MarshalIndent(original, "", "  ")
	require.NoError(t, err)

	// Unmarshal
	var parsed Config
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	// Verify
	assert.Len(t, parsed.Plugin, 1)
	assert.Equal(t, "https://enterprise.example.com", parsed.Enterprise.URL)
	assert.Len(t, parsed.Instructions, 1)
	assert.Contains(t, parsed.Provider, "openai")
	assert.Contains(t, parsed.Mcp, "filesystem")
	assert.Contains(t, parsed.Agent, "coder")
	assert.Contains(t, parsed.Command, "build")
	assert.Equal(t, "testuser", parsed.Username)
	assert.True(t, *parsed.Sse.Enabled)
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}
