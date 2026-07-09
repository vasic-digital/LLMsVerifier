package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultMinIOConfig(t *testing.T) {
	config := DefaultMinIOConfig()

	assert.NotNil(t, config)
	assert.Equal(t, "localhost:9000", config.Endpoint)
	assert.Equal(t, "minioadmin", config.AccessKey)
	assert.Equal(t, "minioadmin123", config.SecretKey)
	assert.False(t, config.UseSSL)
	assert.Equal(t, "us-east-1", config.Region)
	assert.Equal(t, 30*time.Second, config.ConnectTimeout)
	assert.Equal(t, 60*time.Second, config.RequestTimeout)
	assert.Equal(t, "llmsverifier-results", config.VerificationBucket)
	assert.Equal(t, "llmsverifier-models", config.ModelsBucket)
	assert.Equal(t, "llmsverifier-logs", config.LogsBucket)
}

func TestMinIOConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		modify      func(*MinIOConfig)
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid default config",
			modify:      func(c *MinIOConfig) {},
			expectError: false,
		},
		{
			name: "empty endpoint",
			modify: func(c *MinIOConfig) {
				c.Endpoint = ""
			},
			expectError: true,
			errorMsg:    "endpoint is required",
		},
		{
			name: "empty access key",
			modify: func(c *MinIOConfig) {
				c.AccessKey = ""
			},
			expectError: true,
			errorMsg:    "access_key is required",
		},
		{
			name: "empty secret key",
			modify: func(c *MinIOConfig) {
				c.SecretKey = ""
			},
			expectError: true,
			errorMsg:    "secret_key is required",
		},
		{
			name: "invalid connect timeout",
			modify: func(c *MinIOConfig) {
				c.ConnectTimeout = 0
			},
			expectError: true,
			errorMsg:    "connect_timeout must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultMinIOConfig()
			tt.modify(config)

			err := config.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewMinIOClient(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		client, err := NewMinIOClient(nil)
		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.False(t, client.IsConnected())
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &MinIOConfig{
			Endpoint:       "minio.example.com:9000",
			AccessKey:      "access",
			SecretKey:      "secret",
			ConnectTimeout: 60 * time.Second,
		}
		client, err := NewMinIOClient(config)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("with invalid config", func(t *testing.T) {
		config := &MinIOConfig{
			Endpoint: "",
		}
		client, err := NewMinIOClient(config)
		require.Error(t, err)
		assert.Nil(t, client)
	})
}

func TestVerificationResult(t *testing.T) {
	result := &VerificationResult{
		ID:         "test-123",
		ProviderID: "openai",
		ModelID:    "gpt-4",
		Score:      0.95,
		Passed:     true,
		Details: map[string]interface{}{
			"tests_passed": 8,
			"tests_total":  8,
		},
		Timestamp:    time.Now(),
		Duration:     5 * time.Second,
		ErrorMessage: "",
	}

	assert.Equal(t, "test-123", result.ID)
	assert.Equal(t, "openai", result.ProviderID)
	assert.Equal(t, "gpt-4", result.ModelID)
	assert.Equal(t, 0.95, result.Score)
	assert.True(t, result.Passed)
	assert.Empty(t, result.ErrorMessage)
}

func TestLogEntry(t *testing.T) {
	entry := &LogEntry{
		ID:        "log-123",
		Level:     "info",
		Message:   "Test message",
		Component: "verifier",
		Metadata: map[string]interface{}{
			"provider": "openai",
		},
		Timestamp: time.Now(),
	}

	assert.Equal(t, "log-123", entry.ID)
	assert.Equal(t, "info", entry.Level)
	assert.Equal(t, "Test message", entry.Message)
	assert.Equal(t, "verifier", entry.Component)
}
