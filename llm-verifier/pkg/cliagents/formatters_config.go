package cliagents

import "fmt"

// FormatterPreferences defines preferred formatters per language
type FormatterPreferences map[string]string

// FormatterFallbacks defines fallback chains per language
type FormatterFallbacks map[string][]string

// FormatterOverrides defines project-specific overrides
type FormatterOverrides map[string]FormatterOverrideConfig

// FormatterOverrideConfig defines overrides for specific file patterns
type FormatterOverrideConfig struct {
	Formatter  string `json:"formatter"`
	LineLength int    `json:"line_length,omitempty"`
	IndentSize int    `json:"indent_size,omitempty"`
	UseTabs    bool   `json:"use_tabs,omitempty"`
}

// FormattersConfig is the unified formatters configuration
// Used by all CLI agents for code formatting integration
type FormattersConfig struct {
	// Enable formatters integration
	Enabled bool `json:"enabled"`

	// Auto-format on save
	AutoFormat bool `json:"auto_format"`

	// Auto-format AI debate outputs
	FormatOnDebate bool `json:"format_on_debate"`

	// Default line length (typically 88 or 120)
	DefaultLineLength int `json:"default_line_length"`

	// Default indent size (typically 2 or 4)
	DefaultIndentSize int `json:"default_indent_size"`

	// Use tabs vs spaces
	UseTabs bool `json:"use_tabs"`

	// Preferred formatter per language
	Preferences FormatterPreferences `json:"preferences"`

	// Fallback chain per language
	Fallback FormatterFallbacks `json:"fallback,omitempty"`

	// Project-specific overrides by file pattern
	Overrides FormatterOverrides `json:"overrides,omitempty"`

	// HelixAgent formatters API endpoint
	ServiceURL string `json:"service_url,omitempty"`

	// Timeout for formatting requests (seconds)
	Timeout int `json:"timeout,omitempty"`
}

// DefaultFormattersConfig returns the default formatters configuration
func DefaultFormattersConfig(helixAgentHost string, helixAgentPort int) FormattersConfig {
	return FormattersConfig{
		Enabled:           true,
		AutoFormat:        true,
		FormatOnDebate:    true,
		DefaultLineLength: 88,
		DefaultIndentSize: 4,
		UseTabs:           false,
		ServiceURL:        fmt.Sprintf("http://%s:%d/v1/format", helixAgentHost, helixAgentPort),
		Timeout:           30,
		Preferences: FormatterPreferences{
			"python":     "ruff",            // 30x faster than black
			"javascript": "biome",           // 35x faster than prettier
			"typescript": "biome",
			"rust":       "rustfmt",
			"go":         "gofmt",
			"c":          "clang-format",
			"cpp":        "clang-format",
			"java":       "google-java-format",
			"kotlin":     "ktlint",
			"scala":      "scalafmt",
			"ruby":       "rubocop",
			"php":        "php-cs-fixer",
			"swift":      "swift-format",
			"shell":      "shfmt",
			"bash":       "shfmt",
			"sql":        "sqlfluff",
			"yaml":       "yamlfmt",
			"json":       "prettier",
			"toml":       "taplo",
			"markdown":   "prettier",
			"html":       "prettier",
			"css":        "prettier",
			"lua":        "stylua",
			"perl":       "perltidy",
			"clojure":    "cljfmt",
			"groovy":     "npm-groovy-lint",
			"r":          "air",             // 300x faster than styler
			"powershell": "psscriptanalyzer",
		},
		Fallback: FormatterFallbacks{
			"python":     []string{"black", "autopep8", "yapf"},
			"javascript": []string{"prettier", "dprint"},
			"typescript": []string{"prettier", "dprint"},
			"ruby":       []string{"standardrb"},
			"php":        []string{"laravel-pint"},
			"java":       []string{"spotless"},
			"kotlin":     []string{"ktfmt", "spotless"},
			"r":          []string{"styler"},
		},
		Overrides: FormatterOverrides{
			"src/**/*.py": {
				Formatter:  "black",
				LineLength: 120,
			},
			"tests/**/*.py": {
				Formatter:  "black",
				LineLength: 100,
			},
			"*.js": {
				Formatter:  "prettier",
			},
		},
	}
}
