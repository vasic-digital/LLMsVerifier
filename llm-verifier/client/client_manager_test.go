package client

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClientManager(t *testing.T) {
	cm := NewClientManager()
	require.NotNil(t, cm)
	assert.NotNil(t, cm.configs)
	assert.Equal(t, 0, cm.GetClientCount())
}

func TestClientManager_CreateClientConfig(t *testing.T) {
	cm := NewClientManager()

	t.Run("create new config", func(t *testing.T) {
		config, err := cm.CreateClientConfig(1, "Test Client", "A test client")
		require.NoError(t, err)
		require.NotNil(t, config)

		assert.Equal(t, int64(1), config.ClientID)
		assert.Equal(t, "Test Client", config.Name)
		assert.Equal(t, "A test client", config.Description)

		// Check defaults
		assert.Equal(t, "system", config.Preferences.Theme)
		assert.Equal(t, "en", config.Preferences.Language)
		assert.Equal(t, 60, config.RateLimitConfig.RequestsPerMinute)
		assert.True(t, config.AnalyticsConfig.Enabled)
	})

	t.Run("duplicate config fails", func(t *testing.T) {
		_, err := cm.CreateClientConfig(1, "Duplicate", "Duplicate client")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

func TestClientManager_GetClientConfig(t *testing.T) {
	cm := NewClientManager()

	t.Run("get existing config", func(t *testing.T) {
		_, err := cm.CreateClientConfig(2, "Test Client 2", "Description")
		require.NoError(t, err)

		config, err := cm.GetClientConfig(2)
		require.NoError(t, err)
		assert.Equal(t, "Test Client 2", config.Name)
	})

	t.Run("get non-existing config", func(t *testing.T) {
		_, err := cm.GetClientConfig(999)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestClientManager_UpdateClientConfig(t *testing.T) {
	cm := NewClientManager()
	_, err := cm.CreateClientConfig(3, "Update Test", "To be updated")
	require.NoError(t, err)

	t.Run("update existing config", func(t *testing.T) {
		updates := map[string]interface{}{
			"name": "Updated Name",
		}
		err := cm.UpdateClientConfig(3, updates)
		require.NoError(t, err)

		config, _ := cm.GetClientConfig(3)
		assert.Equal(t, "Updated Name", config.Name)
	})

	t.Run("update non-existing config", func(t *testing.T) {
		err := cm.UpdateClientConfig(999, map[string]interface{}{"name": "test"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestClientManager_DeleteClientConfig(t *testing.T) {
	cm := NewClientManager()

	t.Run("delete existing config", func(t *testing.T) {
		_, err := cm.CreateClientConfig(4, "To Delete", "Will be deleted")
		require.NoError(t, err)

		err = cm.DeleteClientConfig(4)
		require.NoError(t, err)

		_, err = cm.GetClientConfig(4)
		assert.Error(t, err)
	})

	t.Run("delete non-existing config", func(t *testing.T) {
		err := cm.DeleteClientConfig(999)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestClientManager_ListClientConfigs(t *testing.T) {
	cm := NewClientManager()

	// Create multiple configs
	cm.CreateClientConfig(10, "Client 10", "Desc 10")
	cm.CreateClientConfig(11, "Client 11", "Desc 11")
	cm.CreateClientConfig(12, "Client 12", "Desc 12")

	configs := cm.ListClientConfigs()
	assert.Equal(t, 3, len(configs))
}

func TestClientManager_GetClientPreferences(t *testing.T) {
	cm := NewClientManager()
	_, err := cm.CreateClientConfig(20, "Pref Client", "Preferences test")
	require.NoError(t, err)

	t.Run("get preferences", func(t *testing.T) {
		prefs, err := cm.GetClientPreferences(20)
		require.NoError(t, err)
		assert.Equal(t, "system", prefs.Theme)
	})

	t.Run("get preferences for non-existing client", func(t *testing.T) {
		_, err := cm.GetClientPreferences(999)
		assert.Error(t, err)
	})
}

func TestClientManager_UpdateClientPreferences(t *testing.T) {
	cm := NewClientManager()
	_, err := cm.CreateClientConfig(21, "Update Pref Client", "Preferences update test")
	require.NoError(t, err)

	newPrefs := ClientPreferences{
		Theme:      "dark",
		Language:   "es",
		MaxTokens:  8192,
		Temperature: 0.5,
	}

	err = cm.UpdateClientPreferences(21, newPrefs)
	require.NoError(t, err)

	prefs, _ := cm.GetClientPreferences(21)
	assert.Equal(t, "dark", prefs.Theme)
	assert.Equal(t, "es", prefs.Language)
}

func TestClientManager_GetNotificationSettings(t *testing.T) {
	cm := NewClientManager()
	_, err := cm.CreateClientConfig(30, "Notif Client", "Notification test")
	require.NoError(t, err)

	t.Run("get notification settings", func(t *testing.T) {
		settings, err := cm.GetNotificationSettings(30)
		require.NoError(t, err)
		assert.False(t, settings.EmailEnabled)
	})

	t.Run("get settings for non-existing client", func(t *testing.T) {
		_, err := cm.GetNotificationSettings(999)
		assert.Error(t, err)
	})
}

func TestClientManager_UpdateNotificationSettings(t *testing.T) {
	cm := NewClientManager()
	_, err := cm.CreateClientConfig(31, "Update Notif Client", "Notification update test")
	require.NoError(t, err)

	newSettings := NotificationSettings{
		EmailEnabled:   true,
		EmailAddress:   "test@example.com",
		SlackEnabled:   true,
		SlackWebhook:   "https://hooks.slack.com/test",
		SeverityFilter: []string{"error", "critical"},
	}

	err = cm.UpdateNotificationSettings(31, newSettings)
	require.NoError(t, err)

	settings, _ := cm.GetNotificationSettings(31)
	assert.True(t, settings.EmailEnabled)
	assert.Equal(t, "test@example.com", settings.EmailAddress)
}

func TestClientManager_GetRateLimitConfig(t *testing.T) {
	cm := NewClientManager()
	_, err := cm.CreateClientConfig(40, "Rate Limit Client", "Rate limit test")
	require.NoError(t, err)

	t.Run("get rate limit config", func(t *testing.T) {
		config, err := cm.GetRateLimitConfig(40)
		require.NoError(t, err)
		assert.True(t, config.Enabled)
		assert.Equal(t, 60, config.RequestsPerMinute)
	})

	t.Run("get config for non-existing client", func(t *testing.T) {
		_, err := cm.GetRateLimitConfig(999)
		assert.Error(t, err)
	})
}

func TestClientManager_UpdateRateLimitConfig(t *testing.T) {
	cm := NewClientManager()
	_, err := cm.CreateClientConfig(41, "Update Rate Client", "Rate limit update test")
	require.NoError(t, err)

	newConfig := RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 120,
		RequestsPerHour:   2000,
		BurstLimit:        20,
		BackoffStrategy:   "linear",
	}

	err = cm.UpdateRateLimitConfig(41, newConfig)
	require.NoError(t, err)

	config, _ := cm.GetRateLimitConfig(41)
	assert.Equal(t, 120, config.RequestsPerMinute)
	assert.Equal(t, "linear", config.BackoffStrategy)
}

func TestClientManager_GetAnalyticsConfig(t *testing.T) {
	cm := NewClientManager()
	_, err := cm.CreateClientConfig(50, "Analytics Client", "Analytics test")
	require.NoError(t, err)

	t.Run("get analytics config", func(t *testing.T) {
		config, err := cm.GetAnalyticsConfig(50)
		require.NoError(t, err)
		assert.True(t, config.Enabled)
		assert.Equal(t, 30, config.RetentionDays)
	})

	t.Run("get config for non-existing client", func(t *testing.T) {
		_, err := cm.GetAnalyticsConfig(999)
		assert.Error(t, err)
	})
}

func TestClientManager_UpdateAnalyticsConfig(t *testing.T) {
	cm := NewClientManager()
	_, err := cm.CreateClientConfig(51, "Update Analytics Client", "Analytics update test")
	require.NoError(t, err)

	newConfig := AnalyticsConfig{
		Enabled:          true,
		RetentionDays:    90,
		ReportFrequency:  "daily",
		MetricsToTrack:   []string{"latency", "errors"},
		DashboardEnabled: false,
	}

	err = cm.UpdateAnalyticsConfig(51, newConfig)
	require.NoError(t, err)

	config, _ := cm.GetAnalyticsConfig(51)
	assert.Equal(t, 90, config.RetentionDays)
	assert.Equal(t, "daily", config.ReportFrequency)
}

func TestClientManager_ExportClientConfig(t *testing.T) {
	cm := NewClientManager()
	_, err := cm.CreateClientConfig(60, "Export Client", "Export test")
	require.NoError(t, err)

	t.Run("export existing config", func(t *testing.T) {
		data, err := cm.ExportClientConfig(60)
		require.NoError(t, err)
		assert.NotEmpty(t, data)
		assert.Contains(t, string(data), "Export Client")
	})

	t.Run("export non-existing config", func(t *testing.T) {
		_, err := cm.ExportClientConfig(999)
		assert.Error(t, err)
	})
}

func TestClientManager_ImportClientConfig(t *testing.T) {
	cm := NewClientManager()

	jsonData := `{
		"client_id": 0,
		"name": "Imported Client",
		"description": "Imported from JSON",
		"preferences": {
			"theme": "light",
			"language": "fr"
		},
		"rate_limit_config": {
			"enabled": true,
			"requests_per_minute": 100
		},
		"analytics_config": {
			"enabled": true,
			"retention_days": 60
		}
	}`

	err := cm.ImportClientConfig(70, []byte(jsonData))
	require.NoError(t, err)

	config, err := cm.GetClientConfig(70)
	require.NoError(t, err)
	assert.Equal(t, "Imported Client", config.Name)
	assert.Equal(t, int64(70), config.ClientID) // Should be overwritten with provided ID
}

func TestClientManager_ImportClientConfig_InvalidJSON(t *testing.T) {
	cm := NewClientManager()

	err := cm.ImportClientConfig(71, []byte("invalid json"))
	assert.Error(t, err)
}

func TestClientManager_ResetClientConfigToDefaults(t *testing.T) {
	cm := NewClientManager()
	_, err := cm.CreateClientConfig(80, "Reset Client", "Will be reset")
	require.NoError(t, err)

	// Modify the config
	cm.UpdateClientPreferences(80, ClientPreferences{
		Theme:     "dark",
		MaxTokens: 10000,
	})

	// Reset to defaults
	err = cm.ResetClientConfigToDefaults(80)
	require.NoError(t, err)

	config, _ := cm.GetClientConfig(80)
	assert.Equal(t, "system", config.Preferences.Theme)
	assert.Equal(t, 4096, config.Preferences.MaxTokens)
}

func TestClientManager_ResetClientConfigToDefaults_NonExisting(t *testing.T) {
	cm := NewClientManager()

	err := cm.ResetClientConfigToDefaults(999)
	assert.Error(t, err)
}

func TestClientManager_ValidateClientConfig(t *testing.T) {
	cm := NewClientManager()

	t.Run("valid config", func(t *testing.T) {
		config := &ClientConfig{
			ClientID: 1,
			Name:     "Valid Client",
			RateLimitConfig: RateLimitConfig{
				RequestsPerMinute: 60,
			},
			AnalyticsConfig: AnalyticsConfig{
				RetentionDays: 30,
			},
		}
		err := cm.ValidateClientConfig(config)
		assert.NoError(t, err)
	})

	t.Run("invalid client ID", func(t *testing.T) {
		config := &ClientConfig{
			ClientID: 0,
			Name:     "Invalid ID",
		}
		err := cm.ValidateClientConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid client ID")
	})

	t.Run("empty name", func(t *testing.T) {
		config := &ClientConfig{
			ClientID: 1,
			Name:     "",
		}
		err := cm.ValidateClientConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})

	t.Run("invalid requests per minute", func(t *testing.T) {
		config := &ClientConfig{
			ClientID: 1,
			Name:     "Test",
			RateLimitConfig: RateLimitConfig{
				RequestsPerMinute: 0,
			},
		}
		err := cm.ValidateClientConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "requests per minute")
	})

	t.Run("negative retention days", func(t *testing.T) {
		config := &ClientConfig{
			ClientID: 1,
			Name:     "Test",
			RateLimitConfig: RateLimitConfig{
				RequestsPerMinute: 60,
			},
			AnalyticsConfig: AnalyticsConfig{
				RetentionDays: -1,
			},
		}
		err := cm.ValidateClientConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "retention days")
	})
}

func TestClientManager_GetClientCount(t *testing.T) {
	cm := NewClientManager()

	assert.Equal(t, 0, cm.GetClientCount())

	cm.CreateClientConfig(100, "Client 1", "Desc")
	assert.Equal(t, 1, cm.GetClientCount())

	cm.CreateClientConfig(101, "Client 2", "Desc")
	assert.Equal(t, 2, cm.GetClientCount())

	cm.DeleteClientConfig(100)
	assert.Equal(t, 1, cm.GetClientCount())
}

func TestClientManager_GetActiveClients(t *testing.T) {
	cm := NewClientManager()

	// Create some clients
	cm.CreateClientConfig(110, "Inactive Client", "No features enabled")

	// Create active client with email notifications
	cm.CreateClientConfig(111, "Email Client", "Has email enabled")
	cm.UpdateNotificationSettings(111, NotificationSettings{EmailEnabled: true})

	// Create active client with analytics
	cm.CreateClientConfig(112, "Analytics Client", "Has analytics enabled")
	cm.UpdateAnalyticsConfig(112, AnalyticsConfig{Enabled: true})

	activeClients := cm.GetActiveClients()

	// Should have 2 active clients (one with email, one with analytics)
	// Note: First client also has analytics enabled by default
	assert.GreaterOrEqual(t, len(activeClients), 2)
}
