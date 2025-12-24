package client

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ClientConfig represents client-specific configuration
type ClientConfig struct {
	ClientID             int64                `json:"client_id"`
	Name                 string               `json:"name"`
	Description          string               `json:"description"`
	Preferences          ClientPreferences    `json:"preferences"`
	NotificationSettings NotificationSettings `json:"notification_settings"`
	RateLimitConfig      RateLimitConfig      `json:"rate_limit_config"`
	AnalyticsConfig      AnalyticsConfig      `json:"analytics_config"`
	CreatedAt            time.Time            `json:"created_at"`
	UpdatedAt            time.Time            `json:"updated_at"`
}

// ClientPreferences holds client-specific preferences
type ClientPreferences struct {
	DefaultProvider string                 `json:"default_provider,omitempty"`
	DefaultModel    string                 `json:"default_model,omitempty"`
	Theme           string                 `json:"theme,omitempty"` // light, dark, system
	Language        string                 `json:"language,omitempty"`
	TimeZone        string                 `json:"timezone,omitempty"`
	DateFormat      string                 `json:"date_format,omitempty"`
	ResponseFormat  string                 `json:"response_format,omitempty"` // json, markdown, text
	StreamResponses bool                   `json:"stream_responses,omitempty"`
	MaxTokens       int                    `json:"max_tokens,omitempty"`
	Temperature     float64                `json:"temperature,omitempty"`
	CustomSettings  map[string]interface{} `json:"custom_settings,omitempty"`
}

// NotificationSettings controls how notifications are delivered
type NotificationSettings struct {
	EmailEnabled    bool       `json:"email_enabled"`
	EmailAddress    string     `json:"email_address,omitempty"`
	SlackEnabled    bool       `json:"slack_enabled"`
	SlackWebhook    string     `json:"slack_webhook,omitempty"`
	TelegramEnabled bool       `json:"telegram_enabled"`
	TelegramChatID  string     `json:"telegram_chat_id,omitempty"`
	SeverityFilter  []string   `json:"severity_filter"` // debug, info, warning, error, critical
	EventTypes      []string   `json:"event_types"`     // verification_started, score_changed, etc.
	QuietHours      QuietHours `json:"quiet_hours,omitempty"`
}

// QuietHours defines when notifications should be suppressed
type QuietHours struct {
	Enabled   bool   `json:"enabled"`
	StartTime string `json:"start_time"` // HH:MM format
	EndTime   string `json:"end_time"`   // HH:MM format
	TimeZone  string `json:"timezone"`
}

// RateLimitConfig defines rate limiting for clients
type RateLimitConfig struct {
	Enabled           bool   `json:"enabled"`
	RequestsPerMinute int    `json:"requests_per_minute"`
	RequestsPerHour   int    `json:"requests_per_hour"`
	RequestsPerDay    int    `json:"requests_per_day"`
	BurstLimit        int    `json:"burst_limit"`
	BackoffStrategy   string `json:"backoff_strategy"` // exponential, linear, fixed
}

// AnalyticsConfig controls analytics and reporting
type AnalyticsConfig struct {
	Enabled          bool     `json:"enabled"`
	RetentionDays    int      `json:"retention_days"`
	ReportFrequency  string   `json:"report_frequency"` // daily, weekly, monthly
	MetricsToTrack   []string `json:"metrics_to_track"` // latency, success_rate, usage, etc.
	DashboardEnabled bool     `json:"dashboard_enabled"`
	ExportFormats    []string `json:"export_formats"` // json, csv, pdf
}

// ClientManager handles client configuration management
type ClientManager struct {
	configs map[int64]*ClientConfig
	mutex   sync.RWMutex
}

// NewClientManager creates a new client configuration manager
func NewClientManager() *ClientManager {
	return &ClientManager{
		configs: make(map[int64]*ClientConfig),
	}
}

// CreateClientConfig creates a new client configuration with defaults
func (cm *ClientManager) CreateClientConfig(clientID int64, name, description string) (*ClientConfig, error) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if _, exists := cm.configs[clientID]; exists {
		return nil, fmt.Errorf("client configuration already exists for ID %d", clientID)
	}

	now := time.Now()
	config := &ClientConfig{
		ClientID:    clientID,
		Name:        name,
		Description: description,
		Preferences: ClientPreferences{
			Theme:           "system",
			Language:        "en",
			TimeZone:        "UTC",
			ResponseFormat:  "json",
			StreamResponses: true,
			MaxTokens:       4096,
			Temperature:     0.7,
		},
		NotificationSettings: NotificationSettings{
			EmailEnabled:    false,
			SlackEnabled:    false,
			TelegramEnabled: false,
			SeverityFilter:  []string{"info", "warning", "error", "critical"},
			EventTypes:      []string{"verification_completed", "score_changed", "issue_detected"},
		},
		RateLimitConfig: RateLimitConfig{
			Enabled:           true,
			RequestsPerMinute: 60,
			RequestsPerHour:   1000,
			RequestsPerDay:    5000,
			BurstLimit:        10,
			BackoffStrategy:   "exponential",
		},
		AnalyticsConfig: AnalyticsConfig{
			Enabled:          true,
			RetentionDays:    30,
			ReportFrequency:  "weekly",
			MetricsToTrack:   []string{"latency", "success_rate", "usage", "errors"},
			DashboardEnabled: true,
			ExportFormats:    []string{"json", "csv"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	cm.configs[clientID] = config
	return config, nil
}

// GetClientConfig retrieves client configuration
func (cm *ClientManager) GetClientConfig(clientID int64) (*ClientConfig, error) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	config, exists := cm.configs[clientID]
	if !exists {
		return nil, fmt.Errorf("client configuration not found for ID %d", clientID)
	}

	return config, nil
}

// UpdateClientConfig updates client configuration
func (cm *ClientManager) UpdateClientConfig(clientID int64, updates map[string]interface{}) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	config, exists := cm.configs[clientID]
	if !exists {
		return fmt.Errorf("client configuration not found for ID %d", clientID)
	}

	// Update fields dynamically
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal current config: %w", err)
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal(configJSON, &configMap); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply updates
	for key, value := range updates {
		configMap[key] = value
	}

	// Convert back to struct
	updatedJSON, err := json.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	if err := json.Unmarshal(updatedJSON, config); err != nil {
		return fmt.Errorf("failed to unmarshal updated config: %w", err)
	}

	config.UpdatedAt = time.Now()
	return nil
}

// DeleteClientConfig removes client configuration
func (cm *ClientManager) DeleteClientConfig(clientID int64) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	if _, exists := cm.configs[clientID]; !exists {
		return fmt.Errorf("client configuration not found for ID %d", clientID)
	}

	delete(cm.configs, clientID)
	return nil
}

// ListClientConfigs returns all client configurations
func (cm *ClientManager) ListClientConfigs() []*ClientConfig {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	configs := make([]*ClientConfig, 0, len(cm.configs))
	for _, config := range cm.configs {
		configs = append(configs, config)
	}

	return configs
}

// GetClientPreferences gets preferences for a client
func (cm *ClientManager) GetClientPreferences(clientID int64) (*ClientPreferences, error) {
	config, err := cm.GetClientConfig(clientID)
	if err != nil {
		return nil, err
	}

	return &config.Preferences, nil
}

// UpdateClientPreferences updates client preferences
func (cm *ClientManager) UpdateClientPreferences(clientID int64, preferences ClientPreferences) error {
	return cm.UpdateClientConfig(clientID, map[string]interface{}{
		"preferences": preferences,
	})
}

// GetNotificationSettings gets notification settings for a client
func (cm *ClientManager) GetNotificationSettings(clientID int64) (*NotificationSettings, error) {
	config, err := cm.GetClientConfig(clientID)
	if err != nil {
		return nil, err
	}

	return &config.NotificationSettings, nil
}

// UpdateNotificationSettings updates notification settings
func (cm *ClientManager) UpdateNotificationSettings(clientID int64, settings NotificationSettings) error {
	return cm.UpdateClientConfig(clientID, map[string]interface{}{
		"notification_settings": settings,
	})
}

// GetRateLimitConfig gets rate limit configuration for a client
func (cm *ClientManager) GetRateLimitConfig(clientID int64) (*RateLimitConfig, error) {
	config, err := cm.GetClientConfig(clientID)
	if err != nil {
		return nil, err
	}

	return &config.RateLimitConfig, nil
}

// UpdateRateLimitConfig updates rate limit configuration
func (cm *ClientManager) UpdateRateLimitConfig(clientID int64, rateConfig RateLimitConfig) error {
	return cm.UpdateClientConfig(clientID, map[string]interface{}{
		"rate_limit_config": rateConfig,
	})
}

// GetAnalyticsConfig gets analytics configuration for a client
func (cm *ClientManager) GetAnalyticsConfig(clientID int64) (*AnalyticsConfig, error) {
	config, err := cm.GetClientConfig(clientID)
	if err != nil {
		return nil, err
	}

	return &config.AnalyticsConfig, nil
}

// UpdateAnalyticsConfig updates analytics configuration
func (cm *ClientManager) UpdateAnalyticsConfig(clientID int64, analyticsConfig AnalyticsConfig) error {
	return cm.UpdateClientConfig(clientID, map[string]interface{}{
		"analytics_config": analyticsConfig,
	})
}

// ExportClientConfig exports client configuration as JSON
func (cm *ClientManager) ExportClientConfig(clientID int64) ([]byte, error) {
	config, err := cm.GetClientConfig(clientID)
	if err != nil {
		return nil, err
	}

	return json.MarshalIndent(config, "", "  ")
}

// ImportClientConfig imports client configuration from JSON
func (cm *ClientManager) ImportClientConfig(clientID int64, data []byte) error {
	var config ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	config.ClientID = clientID
	config.UpdatedAt = time.Now()

	cm.mutex.Lock()
	cm.configs[clientID] = &config
	cm.mutex.Unlock()

	return nil
}

// ResetClientConfigToDefaults resets client configuration to defaults
func (cm *ClientManager) ResetClientConfigToDefaults(clientID int64) error {
	config, err := cm.GetClientConfig(clientID)
	if err != nil {
		return err
	}

	name := config.Name
	description := config.Description

	// Delete current config
	cm.DeleteClientConfig(clientID)

	// Create new config with defaults
	_, err = cm.CreateClientConfig(clientID, name, description)
	return err
}

// ValidateClientConfig validates client configuration
func (cm *ClientManager) ValidateClientConfig(config *ClientConfig) error {
	if config.ClientID <= 0 {
		return fmt.Errorf("invalid client ID: %d", config.ClientID)
	}

	if config.Name == "" {
		return fmt.Errorf("client name cannot be empty")
	}

	if config.RateLimitConfig.RequestsPerMinute <= 0 {
		return fmt.Errorf("requests per minute must be positive")
	}

	if config.AnalyticsConfig.RetentionDays < 0 {
		return fmt.Errorf("retention days cannot be negative")
	}

	return nil
}

// GetClientCount returns the number of configured clients
func (cm *ClientManager) GetClientCount() int {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return len(cm.configs)
}

// GetActiveClients returns clients with enabled configurations
func (cm *ClientManager) GetActiveClients() []*ClientConfig {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	active := make([]*ClientConfig, 0)
	for _, config := range cm.configs {
		// Consider client active if they have any enabled features
		if config.NotificationSettings.EmailEnabled ||
			config.NotificationSettings.SlackEnabled ||
			config.NotificationSettings.TelegramEnabled ||
			config.AnalyticsConfig.Enabled {
			active = append(active, config)
		}
	}

	return active
}
