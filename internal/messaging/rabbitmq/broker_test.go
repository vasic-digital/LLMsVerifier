package rabbitmq

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"llmsverifier/internal/messaging"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 5672, config.Port)
	assert.Equal(t, "guest", config.Username)
	assert.Equal(t, "guest", config.Password)
	assert.Equal(t, "/", config.VHost)
	assert.Equal(t, 10, config.PrefetchCount)
	assert.True(t, config.PublisherConfirms)
	assert.Equal(t, 1*time.Second, config.ReconnectInterval)
	assert.Equal(t, 0, config.MaxReconnectAttempts) // 0 = infinite
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		expectErr bool
	}{
		{
			name:      "valid default config",
			config:    DefaultConfig(),
			expectErr: false,
		},
		{
			name: "empty host",
			config: &Config{
				Host:              "",
				Port:              5672,
				Username:          "guest",
				Password:          "guest",
				ConnectionTimeout: 30 * time.Second,
			},
			expectErr: true,
		},
		{
			name: "invalid port",
			config: &Config{
				Host:              "localhost",
				Port:              0,
				Username:          "guest",
				Password:          "guest",
				ConnectionTimeout: 30 * time.Second,
			},
			expectErr: true,
		},
		{
			name: "negative port",
			config: &Config{
				Host:              "localhost",
				Port:              -1,
				Username:          "guest",
				Password:          "guest",
				ConnectionTimeout: 30 * time.Second,
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigOptions(t *testing.T) {
	config := ApplyConfigOptions(
		WithHost("rabbitmq.example.com"),
		WithPort(5673),
		WithCredentials("admin", "secret"),
		WithVHost("myapp"),
		WithPrefetch(20),
		WithPublisherConfirms(false),
		WithReconnect(60*time.Second, 120*time.Second, 10),
	)

	assert.Equal(t, "rabbitmq.example.com", config.Host)
	assert.Equal(t, 5673, config.Port)
	assert.Equal(t, "admin", config.Username)
	assert.Equal(t, "secret", config.Password)
	assert.Equal(t, "myapp", config.VHost)
	assert.Equal(t, 20, config.PrefetchCount)
	assert.False(t, config.PublisherConfirms)
	assert.Equal(t, 60*time.Second, config.ReconnectInterval)
	assert.Equal(t, 10, config.MaxReconnectAttempts)
}

func TestNewBroker(t *testing.T) {
	config := DefaultConfig()
	logger := logrus.New()

	broker := NewBroker(config, logger)

	require.NotNil(t, broker)
	assert.Equal(t, messaging.BrokerTypeRabbitMQ, broker.BrokerType())
	assert.False(t, broker.IsConnected())
}

func TestNewBrokerWithNilConfig(t *testing.T) {
	broker := NewBroker(nil, nil)

	require.NotNil(t, broker)
	assert.NotNil(t, broker.config)
	assert.NotNil(t, broker.logger)
}

func TestBrokerType(t *testing.T) {
	broker := NewBroker(nil, nil)
	assert.Equal(t, messaging.BrokerTypeRabbitMQ, broker.BrokerType())
}

func TestBrokerMetrics(t *testing.T) {
	broker := NewBroker(nil, nil)
	metrics := broker.GetMetrics()

	require.NotNil(t, metrics)
	assert.Equal(t, int64(0), metrics.MessagesPublished)
	assert.Equal(t, int64(0), metrics.MessagesConsumed)
}

func TestAMQPURI(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected string
	}{
		{
			name:     "default config",
			config:   DefaultConfig(),
			expected: "amqp://guest:guest@localhost:5672",
		},
		{
			name: "custom vhost",
			config: &Config{
				Host:     "localhost",
				Port:     5672,
				Username: "admin",
				Password: "secret",
				VHost:    "myapp",
			},
			expected: "amqp://admin:secret@localhost:5672/myapp",
		},
		{
			name: "with TLS",
			config: &Config{
				Host:     "localhost",
				Port:     5671,
				Username: "admin",
				Password: "secret",
				UseTLS:   true,
			},
			expected: "amqps://admin:secret@localhost:5671",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.AMQPURI()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExchangeConfig(t *testing.T) {
	config := DefaultExchangeConfig("test.exchange", ExchangeTopic)

	assert.Equal(t, "test.exchange", config.Name)
	assert.Equal(t, ExchangeTopic, config.Type)
	assert.True(t, config.Durable)
	assert.False(t, config.AutoDelete)
	assert.False(t, config.Internal)
	assert.False(t, config.NoWait)
}

func TestExchangeTypes(t *testing.T) {
	assert.Equal(t, ExchangeType("direct"), ExchangeDirect)
	assert.Equal(t, ExchangeType("fanout"), ExchangeFanout)
	assert.Equal(t, ExchangeType("topic"), ExchangeTopic)
	assert.Equal(t, ExchangeType("headers"), ExchangeHeaders)
}

func TestConfigError(t *testing.T) {
	err := ConfigError("test error")
	assert.Equal(t, "test error", err.Error())
}

func TestErrNotConnected(t *testing.T) {
	assert.Equal(t, "not connected", ErrNotConnected.Error())
}

// Integration tests (require actual RabbitMQ server)
// These tests are skipped by default and run with -tags=integration

func TestBrokerConnect_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test requires a running RabbitMQ server
	t.Skip("Skipping - requires RabbitMQ server")
}
