package messaging

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultMessagingConfig(t *testing.T) {
	config := DefaultMessagingConfig()

	require.NotNil(t, config)
	assert.True(t, config.Messaging.Enabled)
	assert.True(t, config.Messaging.FallbackEnabled)
	assert.Equal(t, 5*time.Second, config.Messaging.FallbackTimeout)

	// RabbitMQ defaults
	assert.Equal(t, "localhost", config.RabbitMQ.Host)
	assert.Equal(t, 5672, config.RabbitMQ.Port)
	assert.Equal(t, "guest", config.RabbitMQ.Username)
	assert.Equal(t, "guest", config.RabbitMQ.Password)
	assert.Equal(t, "/", config.RabbitMQ.VHost)
	assert.Equal(t, 10, config.RabbitMQ.PrefetchCount)

	// Kafka defaults
	assert.Equal(t, []string{"localhost:9092"}, config.Kafka.Brokers)
	assert.Equal(t, "llmsverifier", config.Kafka.ClientID)
	assert.Equal(t, "llmsverifier-consumer", config.Kafka.GroupID)
	assert.Equal(t, -1, config.Kafka.RequiredAcks)
	assert.Equal(t, "lz4", config.Kafka.CompressionType)
	assert.Equal(t, "latest", config.Kafka.AutoOffsetReset)

	// Circuit breaker defaults
	assert.Equal(t, 5, config.CircuitBreaker.FailureThreshold)
	assert.Equal(t, 2, config.CircuitBreaker.SuccessThreshold)
	assert.Equal(t, 30*time.Second, config.CircuitBreaker.Timeout)

	// Retry defaults
	assert.Equal(t, 3, config.Retry.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, config.Retry.InitialBackoff)
	assert.Equal(t, 10*time.Second, config.Retry.MaxBackoff)
	assert.Equal(t, 2.0, config.Retry.BackoffFactor)

	// Health check defaults
	assert.Equal(t, 30*time.Second, config.HealthCheck.Interval)
	assert.Equal(t, 5*time.Second, config.HealthCheck.Timeout)
}

func TestLoadConfigFromBytes(t *testing.T) {
	yaml := `
messaging:
  enabled: true
  fallback_enabled: false
  fallback_timeout: 10s

rabbitmq:
  host: rabbitmq.example.com
  port: 5673
  username: admin
  password: secret
  vhost: myapp
  prefetch_count: 20

kafka:
  brokers:
    - kafka1:9092
    - kafka2:9092
  client_id: myapp
  group_id: myapp-consumer
  compression_type: snappy

circuit_breaker:
  enabled: true
  failure_threshold: 10
  success_threshold: 3
  timeout: 60s
`

	config, err := LoadConfigFromBytes([]byte(yaml))

	require.NoError(t, err)
	require.NotNil(t, config)

	// Messaging settings
	assert.True(t, config.Messaging.Enabled)
	assert.False(t, config.Messaging.FallbackEnabled)
	assert.Equal(t, 10*time.Second, config.Messaging.FallbackTimeout)

	// RabbitMQ settings
	assert.Equal(t, "rabbitmq.example.com", config.RabbitMQ.Host)
	assert.Equal(t, 5673, config.RabbitMQ.Port)
	assert.Equal(t, "admin", config.RabbitMQ.Username)
	assert.Equal(t, "secret", config.RabbitMQ.Password)
	assert.Equal(t, "myapp", config.RabbitMQ.VHost)
	assert.Equal(t, 20, config.RabbitMQ.PrefetchCount)

	// Kafka settings
	assert.Equal(t, []string{"kafka1:9092", "kafka2:9092"}, config.Kafka.Brokers)
	assert.Equal(t, "myapp", config.Kafka.ClientID)
	assert.Equal(t, "myapp-consumer", config.Kafka.GroupID)
	assert.Equal(t, "snappy", config.Kafka.CompressionType)

	// Circuit breaker settings
	assert.True(t, config.CircuitBreaker.Enabled)
	assert.Equal(t, 10, config.CircuitBreaker.FailureThreshold)
	assert.Equal(t, 3, config.CircuitBreaker.SuccessThreshold)
	assert.Equal(t, 60*time.Second, config.CircuitBreaker.Timeout)
}

func TestLoadConfigFromBytesWithEnvVars(t *testing.T) {
	// Set environment variables
	os.Setenv("TEST_RABBITMQ_HOST", "env-rabbitmq.example.com")
	os.Setenv("TEST_KAFKA_BROKER", "env-kafka:9092")
	defer func() {
		os.Unsetenv("TEST_RABBITMQ_HOST")
		os.Unsetenv("TEST_KAFKA_BROKER")
	}()

	yaml := `
messaging:
  enabled: true

rabbitmq:
  host: ${TEST_RABBITMQ_HOST}
  port: 5672

kafka:
  brokers:
    - ${TEST_KAFKA_BROKER}
`

	config, err := LoadConfigFromBytes([]byte(yaml))

	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "env-rabbitmq.example.com", config.RabbitMQ.Host)
	assert.Equal(t, []string{"env-kafka:9092"}, config.Kafka.Brokers)
}

func TestLoadConfigFromBytesWithDefaults(t *testing.T) {
	yaml := `
messaging:
  enabled: true
`

	config, err := LoadConfigFromBytes([]byte(yaml))

	require.NoError(t, err)
	require.NotNil(t, config)

	// Should have defaults applied
	assert.Equal(t, "localhost", config.RabbitMQ.Host)
	assert.Equal(t, 5672, config.RabbitMQ.Port)
	assert.Equal(t, []string{"localhost:9092"}, config.Kafka.Brokers)
}

func TestLoadConfigFromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "messaging.yaml")

	yaml := `
messaging:
  enabled: true
  fallback_enabled: true
  fallback_timeout: 5s

rabbitmq:
  host: test-rabbitmq
  port: 5672
  username: testuser
  password: testpass

kafka:
  brokers:
    - test-kafka:9092
  client_id: test-client
`

	err := os.WriteFile(configPath, []byte(yaml), 0644)
	require.NoError(t, err)

	config, err := LoadConfig(configPath)

	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "test-rabbitmq", config.RabbitMQ.Host)
	assert.Equal(t, "testuser", config.RabbitMQ.Username)
	assert.Equal(t, []string{"test-kafka:9092"}, config.Kafka.Brokers)
	assert.Equal(t, "test-client", config.Kafka.ClientID)
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	yaml := `
messaging:
  enabled: [invalid yaml
`

	_, err := LoadConfigFromBytes([]byte(yaml))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config")
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    *MessagingConfig
		expectErr bool
		errMsg    string
	}{
		{
			name:      "valid default config",
			config:    DefaultMessagingConfig(),
			expectErr: false,
		},
		{
			name: "disabled messaging",
			config: &MessagingConfig{
				Messaging: MessagingSettings{
					Enabled: false,
				},
			},
			expectErr: false,
		},
		{
			name: "empty rabbitmq host when enabled",
			config: &MessagingConfig{
				Messaging: MessagingSettings{
					Enabled: true,
				},
				RabbitMQ: RabbitMQConfig{
					Host: "",
					Port: 5672,
				},
				Kafka: KafkaConfig{
					Brokers:  []string{"localhost:9092"},
					ClientID: "test",
				},
			},
			expectErr: true,
			errMsg:    "rabbitmq host is required",
		},
		{
			name: "invalid rabbitmq port",
			config: &MessagingConfig{
				Messaging: MessagingSettings{
					Enabled: true,
				},
				RabbitMQ: RabbitMQConfig{
					Host: "localhost",
					Port: 0,
				},
				Kafka: KafkaConfig{
					Brokers:  []string{"localhost:9092"},
					ClientID: "test",
				},
			},
			expectErr: true,
			errMsg:    "invalid rabbitmq port",
		},
		{
			name: "empty kafka brokers",
			config: &MessagingConfig{
				Messaging: MessagingSettings{
					Enabled: true,
				},
				RabbitMQ: RabbitMQConfig{
					Host: "localhost",
					Port: 5672,
				},
				Kafka: KafkaConfig{
					Brokers:  []string{},
					ClientID: "test",
				},
			},
			expectErr: true,
			errMsg:    "at least one kafka broker is required",
		},
		{
			name: "empty kafka client_id",
			config: &MessagingConfig{
				Messaging: MessagingSettings{
					Enabled: true,
				},
				RabbitMQ: RabbitMQConfig{
					Host: "localhost",
					Port: 5672,
				},
				Kafka: KafkaConfig{
					Brokers:  []string{"localhost:9092"},
					ClientID: "",
				},
			},
			expectErr: true,
			errMsg:    "kafka client_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()

			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestQueueDefinition(t *testing.T) {
	yaml := `
messaging:
  enabled: true

rabbitmq:
  host: localhost
  port: 5672
  queues:
    - name: test.queue.1
      durable: true
      auto_delete: false
      dead_letter_exchange: dlx
      dead_letter_routing_key: dlq
      message_ttl: 3600000
    - name: test.queue.2
      durable: false
      auto_delete: true
`

	config, err := LoadConfigFromBytes([]byte(yaml))

	require.NoError(t, err)
	require.Len(t, config.RabbitMQ.Queues, 2)

	q1 := config.RabbitMQ.Queues[0]
	assert.Equal(t, "test.queue.1", q1.Name)
	assert.True(t, q1.Durable)
	assert.False(t, q1.AutoDelete)
	assert.Equal(t, "dlx", q1.DeadLetterExchange)
	assert.Equal(t, "dlq", q1.DeadLetterRoutingKey)
	assert.Equal(t, int64(3600000), q1.MessageTTL)

	q2 := config.RabbitMQ.Queues[1]
	assert.Equal(t, "test.queue.2", q2.Name)
	assert.False(t, q2.Durable)
	assert.True(t, q2.AutoDelete)
}

func TestExchangeDefinition(t *testing.T) {
	yaml := `
messaging:
  enabled: true

rabbitmq:
  host: localhost
  port: 5672
  exchanges:
    - name: test.direct
      type: direct
      durable: true
    - name: test.topic
      type: topic
      durable: true
    - name: test.fanout
      type: fanout
      durable: false
      auto_delete: true
`

	config, err := LoadConfigFromBytes([]byte(yaml))

	require.NoError(t, err)
	require.Len(t, config.RabbitMQ.Exchanges, 3)

	assert.Equal(t, "test.direct", config.RabbitMQ.Exchanges[0].Name)
	assert.Equal(t, "direct", config.RabbitMQ.Exchanges[0].Type)
	assert.True(t, config.RabbitMQ.Exchanges[0].Durable)

	assert.Equal(t, "test.topic", config.RabbitMQ.Exchanges[1].Name)
	assert.Equal(t, "topic", config.RabbitMQ.Exchanges[1].Type)

	assert.Equal(t, "test.fanout", config.RabbitMQ.Exchanges[2].Name)
	assert.Equal(t, "fanout", config.RabbitMQ.Exchanges[2].Type)
	assert.False(t, config.RabbitMQ.Exchanges[2].Durable)
	assert.True(t, config.RabbitMQ.Exchanges[2].AutoDelete)
}

func TestTopicDefinition(t *testing.T) {
	yaml := `
messaging:
  enabled: true

kafka:
  brokers:
    - localhost:9092
  topics:
    - name: test.events
      partitions: 6
      replication_factor: 3
      retention_hours: 168
    - name: test.metrics
      partitions: 3
      replication_factor: 1
      retention_hours: 72
`

	config, err := LoadConfigFromBytes([]byte(yaml))

	require.NoError(t, err)
	require.Len(t, config.Kafka.Topics, 2)

	t1 := config.Kafka.Topics[0]
	assert.Equal(t, "test.events", t1.Name)
	assert.Equal(t, 6, t1.Partitions)
	assert.Equal(t, 3, t1.ReplicationFactor)
	assert.Equal(t, 168, t1.RetentionHours)

	t2 := config.Kafka.Topics[1]
	assert.Equal(t, "test.metrics", t2.Name)
	assert.Equal(t, 3, t2.Partitions)
	assert.Equal(t, 1, t2.ReplicationFactor)
	assert.Equal(t, 72, t2.RetentionHours)
}
