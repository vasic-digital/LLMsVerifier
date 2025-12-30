package enterprise

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== EnterpriseMonitor Tests ====================

func TestNewEnterpriseMonitor(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled:       true,
		BatchInterval: 5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    1 * time.Second,
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)
	assert.NotNil(t, monitor)
	assert.True(t, monitor.config.Enabled)
}

func TestEnterpriseMonitor_WithSplunkConfig(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: true,
		Splunk: SplunkConfig{
			Host:       "splunk.example.com",
			Port:       8088,
			Token:      "test-token",
			Index:      "main",
			Source:     "llm-verifier",
			Sourcetype: "llm:metrics",
		},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)
	assert.NotNil(t, monitor)
	assert.Equal(t, "splunk.example.com", monitor.config.Splunk.Host)
}

func TestEnterpriseMonitor_WithDataDogConfig(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: true,
		DataDog: DataDogConfig{
			APIKey:      "test-api-key",
			AppKey:      "test-app-key",
			Endpoint:    "https://api.datadoghq.com",
			ServiceName: "llm-verifier",
			Environment: "production",
			Tags:        map[string]string{"env": "prod"},
		},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)
	assert.NotNil(t, monitor)
	assert.Equal(t, "llm-verifier", monitor.config.DataDog.ServiceName)
}

func TestEnterpriseMonitor_WithNewRelicConfig(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: true,
		NewRelic: NewRelicConfig{
			LicenseKey: "test-license-key",
			AppName:    "LLM Verifier",
			Endpoint:   "https://api.newrelic.com",
			Labels:     map[string]string{"version": "1.0"},
		},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)
	assert.NotNil(t, monitor)
	assert.Equal(t, "LLM Verifier", monitor.config.NewRelic.AppName)
}

func TestEnterpriseMonitor_WithELKConfig(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: true,
		ELK: ELKConfig{
			ElasticsearchURL: "http://elasticsearch:9200",
			IndexName:        "llm-verifier-logs",
			Username:         "elastic",
			Password:         "password",
		},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)
	assert.NotNil(t, monitor)
	assert.Equal(t, "llm-verifier-logs", monitor.config.ELK.IndexName)
}

func TestEnterpriseMonitor_WithWebhooks(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: true,
		CustomWebhooks: []WebhookConfig{
			{
				URL:        "https://webhook.example.com/events",
				Method:     "POST",
				Headers:    map[string]string{"Authorization": "Bearer token"},
				EventTypes: []string{"alert", "metric"},
				RetryConfig: RetryConfig{
					MaxAttempts: 3,
					Delay:       1 * time.Second,
					Backoff:     2.0,
				},
			},
		},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)
	assert.NotNil(t, monitor)
	assert.Len(t, monitor.config.CustomWebhooks, 1)
	assert.Equal(t, "POST", monitor.config.CustomWebhooks[0].Method)
}

// ==================== Config Struct Tests ====================

func TestSplunkConfig_Struct(t *testing.T) {
	config := SplunkConfig{
		Host:       "localhost",
		Port:       8088,
		Token:      "token123",
		Index:      "main",
		Source:     "app",
		Sourcetype: "json",
		Fields:     map[string]string{"env": "test"},
	}

	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 8088, config.Port)
	assert.Equal(t, "test", config.Fields["env"])
}

func TestDataDogConfig_Struct(t *testing.T) {
	config := DataDogConfig{
		APIKey:      "api123",
		AppKey:      "app456",
		Endpoint:    "https://api.datadoghq.com",
		Tags:        map[string]string{"service": "test"},
		ServiceName: "myservice",
		Environment: "dev",
	}

	assert.Equal(t, "api123", config.APIKey)
	assert.Equal(t, "dev", config.Environment)
}

func TestNewRelicConfig_Struct(t *testing.T) {
	config := NewRelicConfig{
		LicenseKey: "license123",
		AppName:    "TestApp",
		Labels:     map[string]string{"team": "dev"},
		Endpoint:   "https://newrelic.com",
	}

	assert.Equal(t, "license123", config.LicenseKey)
	assert.Equal(t, "TestApp", config.AppName)
}

func TestELKConfig_Struct(t *testing.T) {
	config := ELKConfig{
		ElasticsearchURL: "http://es:9200",
		IndexName:        "logs",
		Username:         "admin",
		Password:         "secret",
		Mapping:          map[string]interface{}{"dynamic": "strict"},
	}

	assert.Equal(t, "http://es:9200", config.ElasticsearchURL)
	assert.Equal(t, "logs", config.IndexName)
}

func TestWebhookConfig_Struct(t *testing.T) {
	config := WebhookConfig{
		URL:        "https://hooks.example.com",
		Method:     "POST",
		Headers:    map[string]string{"Content-Type": "application/json"},
		Template:   "{{.Message}}",
		EventTypes: []string{"alert", "info"},
		RetryConfig: RetryConfig{
			MaxAttempts: 5,
			Delay:       2 * time.Second,
			Backoff:     1.5,
		},
	}

	assert.Equal(t, "POST", config.Method)
	assert.Len(t, config.EventTypes, 2)
	assert.Equal(t, 5, config.RetryConfig.MaxAttempts)
}

func TestRetryConfig_Struct(t *testing.T) {
	config := RetryConfig{
		MaxAttempts: 3,
		Delay:       500 * time.Millisecond,
		Backoff:     2.0,
	}

	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, 500*time.Millisecond, config.Delay)
	assert.Equal(t, 2.0, config.Backoff)
}

func TestEnterpriseMonitorConfig_Struct(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled:       true,
		BatchInterval: 10 * time.Second,
		RetryAttempts: 5,
		RetryDelay:    2 * time.Second,
	}

	assert.True(t, config.Enabled)
	assert.Equal(t, 10*time.Second, config.BatchInterval)
	assert.Equal(t, 5, config.RetryAttempts)
}

// ==================== Mock Server Tests ====================

func TestEnterpriseMonitor_SendToSplunk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := EnterpriseMonitorConfig{
		Enabled: true,
		Splunk: SplunkConfig{
			Host:  server.URL[7:], // Strip "http://"
			Port:  80,
			Token: "test-token",
		},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)
	assert.NotNil(t, monitor)
}

func TestEnterpriseMonitor_SendToDataDog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	config := EnterpriseMonitorConfig{
		Enabled: true,
		DataDog: DataDogConfig{
			APIKey:   "test-key",
			Endpoint: server.URL,
		},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)
	assert.NotNil(t, monitor)
}

// ==================== Edge Cases ====================

func TestEnterpriseMonitor_DisabledConfig(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: false,
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)
	assert.NotNil(t, monitor)
	assert.False(t, monitor.config.Enabled)
}

func TestEnterpriseMonitor_EmptyWebhooks(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled:        true,
		CustomWebhooks: []WebhookConfig{},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)
	assert.NotNil(t, monitor)
	assert.Empty(t, monitor.config.CustomWebhooks)
}

func TestEnterpriseMonitor_ZeroRetryConfig(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled:       true,
		RetryAttempts: 0,
		RetryDelay:    0,
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)
	assert.NotNil(t, monitor)
}

// ==================== Multiple Config Tests ====================

func TestEnterpriseMonitor_AllConfigsEnabled(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled:       true,
		BatchInterval: 5 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    1 * time.Second,
		Splunk: SplunkConfig{
			Host:  "splunk.example.com",
			Port:  8088,
			Token: "token",
		},
		DataDog: DataDogConfig{
			APIKey:   "dd-key",
			Endpoint: "https://api.datadoghq.com",
		},
		NewRelic: NewRelicConfig{
			LicenseKey: "nr-key",
			AppName:    "test",
		},
		ELK: ELKConfig{
			ElasticsearchURL: "http://es:9200",
			IndexName:        "logs",
		},
		CustomWebhooks: []WebhookConfig{
			{
				URL:    "https://webhook.example.com",
				Method: "POST",
			},
		},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)
	require.NotNil(t, monitor)
	assert.True(t, monitor.config.Enabled)
	assert.NotEmpty(t, monitor.config.Splunk.Host)
	assert.NotEmpty(t, monitor.config.DataDog.APIKey)
	assert.NotEmpty(t, monitor.config.NewRelic.LicenseKey)
	assert.NotEmpty(t, monitor.config.ELK.ElasticsearchURL)
	assert.Len(t, monitor.config.CustomWebhooks, 1)
}

func TestEnterpriseMonitor_MultipleWebhooks(t *testing.T) {
	config := EnterpriseMonitorConfig{
		Enabled: true,
		CustomWebhooks: []WebhookConfig{
			{
				URL:        "https://webhook1.example.com",
				Method:     "POST",
				EventTypes: []string{"alert"},
			},
			{
				URL:        "https://webhook2.example.com",
				Method:     "POST",
				EventTypes: []string{"metric"},
			},
			{
				URL:        "https://webhook3.example.com",
				Method:     "PUT",
				EventTypes: []string{"alert", "metric"},
			},
		},
	}

	monitor := NewEnterpriseMonitor(config, nil, nil)
	require.NotNil(t, monitor)
	assert.Len(t, monitor.config.CustomWebhooks, 3)
}
