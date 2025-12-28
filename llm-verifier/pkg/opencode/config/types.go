package opencode_config

import (
	"encoding/json"
	"os"
)

// Config represents the OpenCode configuration structure
type Config struct {
	Plugin       []string                `json:"plugin,omitempty"`
	Enterprise   *EnterpriseConfig       `json:"enterprise,omitempty"`
	Instructions []string               `json:"instructions,omitempty"`
	Provider     map[string]ProviderConfig  `json:"provider,omitempty"`
	Mcp          map[string]McpConfig       `json:"mcp,omitempty"`
	Tools        map[string]interface{}     `json:"tools,omitempty"`
	Agent        map[string]AgentConfig     `json:"agent,omitempty"`
	Command      map[string]CommandConfig   `json:"command,omitempty"`
	Keybinds     *KeybindsConfig            `json:"keybinds,omitempty"`
	Username     string                     `json:"username,omitempty"`
	Share        interface{}                `json:"share,omitempty"`
	Permission   *PermissionConfig          `json:"permission,omitempty"`
	Compaction   *CompactionConfig          `json:"compaction,omitempty"`
	Sse          *SseConfig                 `json:"sse,omitempty"`
	Mode         map[string]interface{}     `json:"mode,omitempty"`
	Autoshare    interface{}                `json:"autoshare,omitempty"`
}

type EnterpriseConfig struct {
	URL string `json:"url,omitempty"`
}

type ProviderConfig struct {
	Options map[string]interface{} `json:"options,omitempty"`
	Model   string                 `json:"model,omitempty"`
}

type McpConfig struct {
	Type        string            `json:"type"`
	Command     []string          `json:"command,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Enabled     *bool             `json:"enabled,omitempty"`
	Timeout     *int              `json:"timeout,omitempty"`
	URL         string            `json:"url,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	OAuth       interface{}       `json:"oauth,omitempty"`
}

type AgentConfig struct {
	Model        string                 `json:"model,omitempty"`
	Temperature  *float64               `json:"temperature,omitempty"`
	TopP         *float64               `json:"top_p,omitempty"`
	Prompt       string                 `json:"prompt,omitempty"`
	Tools        map[string]bool        `json:"tools,omitempty"`
	Disable      *bool                  `json:"disable,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Mode         string                 `json:"mode,omitempty"`
	Color        string                 `json:"color,omitempty"`
	MaxSteps     *int                   `json:"maxSteps,omitempty"`
	Permission   map[string]interface{} `json:"permission,omitempty"`
}

type CommandConfig struct {
	Template    string `json:"template"`
	Description string `json:"description,omitempty"`
	Agent       string `json:"agent,omitempty"`
	Model       string `json:"model,omitempty"`
	Subtask     *bool  `json:"subtask,omitempty"`
}

type KeybindsConfig struct {
	Leader                      string `json:"leader,omitempty"`
	AppExit                     string `json:"app_exit,omitempty"`
	EditorOpen                  string `json:"editor_open,omitempty"`
	ThemeList                   string `json:"theme_list,omitempty"`
	SidebarToggle               string `json:"sidebar_toggle,omitempty"`
	ScrollbarToggle             string `json:"scrollbar_toggle,omitempty"`
	UsernameToggle              string `json:"username_toggle,omitempty"`
	StatusView                  string `json:"status_view,omitempty"`
	SessionExport               string `json:"session_export,omitempty"`
	SessionNew                  string `json:"session_new,omitempty"`
	SessionList                 string `json:"session_list,omitempty"`
	SessionTimeline             string `json:"session_timeline,omitempty"`
	SessionFork                 string `json:"session_fork,omitempty"`
	SessionRename               string `json:"session_rename,omitempty"`
	SessionShare                string `json:"session_share,omitempty"`
	SessionUnshare              string `json:"session_unshare,omitempty"`
	SessionInterrupt            string `json:"session_interrupt,omitempty"`
	SessionCompact              string `json:"session_compact,omitempty"`
	MessagesPageUp              string `json:"messages_page_up,omitempty"`
	MessagesPageDown            string `json:"messages_page_down,omitempty"`
	MessagesHalfPageUp          string `json:"messages_half_page_up,omitempty"`
	MessagesHalfPageDown        string `json:"messages_half_page_down,omitempty"`
	MessagesFirst               string `json:"messages_first,omitempty"`
	MessagesLast                string `json:"messages_last,omitempty"`
	MessagesNext                string `json:"messages_next,omitempty"`
	MessagesPrevious            string `json:"messages_previous,omitempty"`
	MessagesLastUser            string `json:"messages_last_user,omitempty"`
	MessagesCopy                string `json:"messages_copy,omitempty"`
	MessagesUndo                string `json:"messages_undo,omitempty"`
	MessagesRedo                string `json:"messages_redo,omitempty"`
	MessagesToggleConceal       string `json:"messages_toggle_conceal,omitempty"`
	ToolDetails                 string `json:"tool_details,omitempty"`
	ModelList                   string `json:"model_list,omitempty"`
	ModelCycleRecent            string `json:"model_cycle_recent,omitempty"`
	ModelCycleRecentReverse     string `json:"model_cycle_recent_reverse,omitempty"`
	ModelCycleFavorite          string `json:"model_cycle_favorite,omitempty"`
	ModelCycleFavoriteReverse   string `json:"model_cycle_favorite_reverse,omitempty"`
	CommandList                 string `json:"command_list,omitempty"`
	AgentList                   string `json:"agent_list,omitempty"`
	AgentCycle                  string `json:"agent_cycle,omitempty"`
	AgentCycleReverse           string `json:"agent_cycle_reverse,omitempty"`
	InputClear                  string `json:"input_clear,omitempty"`
	InputPaste                  string `json:"input_paste,omitempty"`
	InputSubmit                 string `json:"input_submit,omitempty"`
	InputNewline                string `json:"input_newline,omitempty"`
	InputMoveLeft               string `json:"input_move_left,omitempty"`
	InputMoveRight              string `json:"input_move_right,omitempty"`
	InputMoveUp                 string `json:"input_move_up,omitempty"`
	InputMoveDown               string `json:"input_move_down,omitempty"`
	InputSelectLeft             string `json:"input_select_left,omitempty"`
	InputSelectRight            string `json:"input_select_right,omitempty"`
	InputSelectUp               string `json:"input_select_up,omitempty"`
	InputSelectDown             string `json:"input_select_down,omitempty"`
	InputLineHome               string `json:"input_line_home,omitempty"`
	InputLineEnd                string `json:"input_line_end,omitempty"`
	InputSelectLineHome         string `json:"input_select_line_home,omitempty"`
	InputSelectLineEnd          string `json:"input_select_line_end,omitempty"`
	InputVisualLineHome         string `json:"input_visual_line_home,omitempty"`
	InputVisualLineEnd          string `json:"input_visual_line_end,omitempty"`
	InputSelectVisualLineHome   string `json:"input_select_visual_line_home,omitempty"`
	InputSelectVisualLineEnd    string `json:"input_select_visual_line_end,omitempty"`
	InputBufferHome             string `json:"input_buffer_home,omitempty"`
	InputBufferEnd              string `json:"input_buffer_end,omitempty"`
	InputSelectBufferHome       string `json:"input_select_buffer_home,omitempty"`
	InputSelectBufferEnd        string `json:"input_select_buffer_end,omitempty"`
	InputDeleteLine             string `json:"input_delete_line,omitempty"`
	InputDeleteToLineEnd        string `json:"input_delete_to_line_end,omitempty"`
	InputDeleteToLineStart      string `json:"input_delete_to_line_start,omitempty"`
	InputBackspace              string `json:"input_backspace,omitempty"`
	InputDelete                 string `json:"input_delete,omitempty"`
	InputUndo                   string `json:"input_undo,omitempty"`
	InputRedo                   string `json:"input_redo,omitempty"`
	InputWordForward            string `json:"input_word_forward,omitempty"`
	InputWordBackward           string `json:"input_word_backward,omitempty"`
	InputSelectWordForward      string `json:"input_select_word_forward,omitempty"`
	InputSelectWordBackward     string `json:"input_select_word_backward,omitempty"`
	InputDeleteWordForward      string `json:"input_delete_word_forward,omitempty"`
	InputDeleteWordBackward     string `json:"input_delete_word_backward,omitempty"`
	HistoryPrevious             string `json:"history_previous,omitempty"`
	HistoryNext                 string `json:"history_next,omitempty"`
	SessionChildCycle           string `json:"session_child_cycle,omitempty"`
	SessionChildCycleReverse    string `json:"session_child_cycle_reverse,omitempty"`
	SessionParent               string `json:"session_parent,omitempty"`
	TerminalSuspend             string `json:"terminal_suspend,omitempty"`
	TerminalTitleToggle         string `json:"terminal_title_toggle,omitempty"`
	TipsToggle                  string `json:"tips_toggle,omitempty"`
}

type PermissionConfig struct {
	Edit              string      `json:"edit,omitempty"`
	Bash              interface{} `json:"bash,omitempty"`
	Skill             interface{} `json:"skill,omitempty"`
	Webfetch          string      `json:"webfetch,omitempty"`
	DoomLoop          string      `json:"doom_loop,omitempty"`
	ExternalDirectory string      `json:"external_directory,omitempty"`
}

type CompactionConfig struct {
	Auto  *bool `json:"auto,omitempty"`
	Prune *bool `json:"prune,omitempty"`
}

type SseConfig struct {
	Enabled *bool `json:"enabled,omitempty"`
}

// LoadFromFile loads a config from a file
type ConfigLoader struct{}

func (cl *ConfigLoader) LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	
	return &config, nil
}

// SaveToFile saves a config to a file
func (cl *ConfigLoader) SaveToFile(config *Config, path string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(path, data, 0644)
}