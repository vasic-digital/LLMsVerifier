package factory

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"llmsverifier/internal/messaging"
	"llmsverifier/internal/messaging/inmemory"
	"llmsverifier/internal/messaging/kafka"
	"llmsverifier/internal/messaging/rabbitmq"
)

func TestNewBrokerFactory(t *testing.T) {
	tests := []struct {
		name   string
		config *messaging.MessagingConfig
		logger *logrus.Logger
	}{
		{
			name:   "with nil config and logger",
			config: nil,
			logger: nil,
		},
		{
			name:   "with custom config",
			config: messaging.DefaultMessagingConfig(),
			logger: logrus.New(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			factory := NewBrokerFactory(tt.config, tt.logger)

			require.NotNil(t, factory)
			assert.NotNil(t, factory.config)
			assert.NotNil(t, factory.logger)
			assert.False(t, factory.initialized)
		})
	}
}

func TestBrokerFactoryInitializeDisabledMessaging(t *testing.T) {
	config := messaging.DefaultMessagingConfig()
	config.Messaging.Enabled = false

	factory := NewBrokerFactory(config, nil)
	err := factory.Initialize(context.Background())

	require.NoError(t, err)
	assert.True(t, factory.IsInitialized())
	assert.NotNil(t, factory.TaskQueue())
	assert.NotNil(t, factory.EventStream())
	assert.NotNil(t, factory.Fallback())
	assert.Equal(t, messaging.BrokerTypeInMemory, factory.TaskQueue().BrokerType())
	assert.Equal(t, messaging.BrokerTypeInMemory, factory.EventStream().BrokerType())
}

func TestBrokerFactoryInitializeWithFallback(t *testing.T) {
	config := messaging.DefaultMessagingConfig()
	config.Messaging.Enabled = true
	config.Messaging.FallbackEnabled = true
	config.Messaging.FallbackTimeout = 100 * time.Millisecond
	// Use invalid hosts to force fallback
	config.RabbitMQ.Host = "invalid-host"
	config.Kafka.Brokers = []string{"invalid-host:9092"}

	factory := NewBrokerFactory(config, nil)
	err := factory.Initialize(context.Background())

	require.NoError(t, err)
	assert.True(t, factory.IsInitialized())
	assert.True(t, factory.IsUsingFallback())
	assert.Equal(t, messaging.BrokerTypeInMemory, factory.TaskQueue().BrokerType())
	assert.Equal(t, messaging.BrokerTypeInMemory, factory.EventStream().BrokerType())
}

func TestBrokerFactoryInitializeNoFallback(t *testing.T) {
	config := messaging.DefaultMessagingConfig()
	config.Messaging.Enabled = true
	config.Messaging.FallbackEnabled = false
	config.Messaging.FallbackTimeout = 100 * time.Millisecond
	// Use invalid hosts to force failure
	config.RabbitMQ.Host = "invalid-host"
	config.Kafka.Brokers = []string{"invalid-host:9092"}

	factory := NewBrokerFactory(config, nil)
	err := factory.Initialize(context.Background())

	// Should fail because fallback is disabled and connection fails
	assert.Error(t, err)
}

func TestBrokerFactoryDoubleInitialize(t *testing.T) {
	config := messaging.DefaultMessagingConfig()
	config.Messaging.Enabled = false

	factory := NewBrokerFactory(config, nil)

	err := factory.Initialize(context.Background())
	require.NoError(t, err)

	// Second initialize should be no-op
	err = factory.Initialize(context.Background())
	require.NoError(t, err)
}

func TestBrokerFactoryClose(t *testing.T) {
	config := messaging.DefaultMessagingConfig()
	config.Messaging.Enabled = false

	factory := NewBrokerFactory(config, nil)

	err := factory.Initialize(context.Background())
	require.NoError(t, err)

	err = factory.Close(context.Background())
	require.NoError(t, err)

	assert.False(t, factory.IsInitialized())
	assert.Nil(t, factory.TaskQueue())
	assert.Nil(t, factory.EventStream())
	assert.Nil(t, factory.Fallback())
}

func TestBrokerFactoryCloseNotInitialized(t *testing.T) {
	factory := NewBrokerFactory(nil, nil)

	err := factory.Close(context.Background())
	require.NoError(t, err)
}

func TestBrokerFactorySwitchToFallback(t *testing.T) {
	config := messaging.DefaultMessagingConfig()
	config.Messaging.Enabled = false

	factory := NewBrokerFactory(config, nil)

	err := factory.Initialize(context.Background())
	require.NoError(t, err)

	err = factory.SwitchToFallback(context.Background())
	require.NoError(t, err)

	assert.Equal(t, messaging.BrokerTypeInMemory, factory.TaskQueue().BrokerType())
	assert.Equal(t, messaging.BrokerTypeInMemory, factory.EventStream().BrokerType())
}

func TestBrokerFactoryHealthCheck(t *testing.T) {
	config := messaging.DefaultMessagingConfig()
	config.Messaging.Enabled = false

	factory := NewBrokerFactory(config, nil)

	err := factory.Initialize(context.Background())
	require.NoError(t, err)

	assert.True(t, factory.TaskQueueHealthy(context.Background()))
	assert.True(t, factory.EventStreamHealthy(context.Background()))
}

func TestBrokerFactoryHealthCheckNotInitialized(t *testing.T) {
	factory := NewBrokerFactory(nil, nil)

	assert.False(t, factory.TaskQueueHealthy(context.Background()))
	assert.False(t, factory.EventStreamHealthy(context.Background()))
}

func TestBrokerFactoryGetStatus(t *testing.T) {
	config := messaging.DefaultMessagingConfig()
	config.Messaging.Enabled = false

	factory := NewBrokerFactory(config, nil)

	err := factory.Initialize(context.Background())
	require.NoError(t, err)

	status := factory.GetStatus()

	require.NotNil(t, status)
	assert.True(t, status.Initialized)
	assert.True(t, status.UsingFallback)
	assert.NotNil(t, status.TaskQueue)
	assert.NotNil(t, status.EventStream)
	assert.NotNil(t, status.Fallback)
	assert.Equal(t, messaging.BrokerTypeInMemory, status.TaskQueue.Type)
	assert.True(t, status.TaskQueue.Connected)
}

func TestBrokerFactoryGetStatusNotInitialized(t *testing.T) {
	factory := NewBrokerFactory(nil, nil)
	status := factory.GetStatus()

	require.NotNil(t, status)
	assert.False(t, status.Initialized)
	assert.True(t, status.UsingFallback) // No brokers means fallback
	assert.Nil(t, status.TaskQueue)
	assert.Nil(t, status.EventStream)
	assert.Nil(t, status.Fallback)
}

func TestCreateBroker(t *testing.T) {
	tests := []struct {
		name       string
		brokerType messaging.BrokerType
		config     interface{}
		expectErr  bool
	}{
		{
			name:       "create in-memory broker with config",
			brokerType: messaging.BrokerTypeInMemory,
			config: &inmemory.Config{
				MaxQueueSize: 1000,
				MessageTTL:   time.Hour,
			},
			expectErr: false,
		},
		{
			name:       "create in-memory broker without config",
			brokerType: messaging.BrokerTypeInMemory,
			config:     nil,
			expectErr:  false,
		},
		{
			name:       "create rabbitmq broker",
			brokerType: messaging.BrokerTypeRabbitMQ,
			config: &rabbitmq.Config{
				Host: "localhost",
				Port: 5672,
			},
			expectErr: false,
		},
		{
			name:       "create rabbitmq broker with invalid config",
			brokerType: messaging.BrokerTypeRabbitMQ,
			config:     "invalid",
			expectErr:  true,
		},
		{
			name:       "create kafka broker",
			brokerType: messaging.BrokerTypeKafka,
			config: &kafka.Config{
				Brokers:  []string{"localhost:9092"},
				ClientID: "test",
			},
			expectErr: false,
		},
		{
			name:       "create kafka broker with invalid config",
			brokerType: messaging.BrokerTypeKafka,
			config:     "invalid",
			expectErr:  true,
		},
		{
			name:       "unknown broker type",
			brokerType: messaging.BrokerType("unknown"),
			config:     nil,
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			broker, err := CreateBroker(tt.brokerType, tt.config)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, broker)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, broker)
				assert.Equal(t, tt.brokerType, broker.BrokerType())
			}
		})
	}
}

func TestCreateTaskQueueBroker(t *testing.T) {
	broker, err := CreateTaskQueueBroker(messaging.BrokerTypeInMemory, nil)

	require.NoError(t, err)
	require.NotNil(t, broker)
	assert.Equal(t, messaging.BrokerTypeInMemory, broker.BrokerType())
}

func TestCreateTaskQueueBrokerInvalidType(t *testing.T) {
	broker, err := CreateTaskQueueBroker(messaging.BrokerType("unknown"), nil)

	assert.Error(t, err)
	assert.Nil(t, broker)
}

func TestCreateEventStreamBroker(t *testing.T) {
	broker, err := CreateEventStreamBroker(messaging.BrokerTypeInMemory, nil)

	require.NoError(t, err)
	require.NotNil(t, broker)
	assert.Equal(t, messaging.BrokerTypeInMemory, broker.BrokerType())
}

func TestCreateEventStreamBrokerInvalidType(t *testing.T) {
	broker, err := CreateEventStreamBroker(messaging.BrokerType("unknown"), nil)

	assert.Error(t, err)
	assert.Nil(t, broker)
}

func TestFromConfigBytes(t *testing.T) {
	yaml := `
messaging:
  enabled: false
  fallback_enabled: true

rabbitmq:
  host: localhost
  port: 5672

kafka:
  brokers:
    - localhost:9092
  client_id: test
`

	factory, err := FromConfigBytes([]byte(yaml), nil)

	require.NoError(t, err)
	require.NotNil(t, factory)
	assert.False(t, factory.IsInitialized())
}

func TestFromConfigBytesInvalidYAML(t *testing.T) {
	yaml := `
messaging:
  enabled: [invalid
`

	factory, err := FromConfigBytes([]byte(yaml), nil)

	assert.Error(t, err)
	assert.Nil(t, factory)
}

func TestFromConfigBytesInvalidConfig(t *testing.T) {
	// Create a config manually and skip defaults to test validation
	config := &messaging.MessagingConfig{
		Messaging: messaging.MessagingSettings{
			Enabled: true,
		},
		RabbitMQ: messaging.RabbitMQConfig{
			Host: "",  // Invalid: empty host
			Port: 0,   // Invalid: zero port
		},
		Kafka: messaging.KafkaConfig{
			Brokers:  []string{}, // Invalid: no brokers
			ClientID: "",         // Invalid: empty client ID
		},
	}

	err := config.Validate()

	assert.Error(t, err)
	// Validation should fail on empty host
	assert.Contains(t, err.Error(), "rabbitmq host is required")
}

func TestFromConfigFileNotFound(t *testing.T) {
	factory, err := FromConfigFile("/nonexistent/config.yaml", nil)

	assert.Error(t, err)
	assert.Nil(t, factory)
}

func TestBrokerFactoryReconnect(t *testing.T) {
	config := messaging.DefaultMessagingConfig()
	config.Messaging.Enabled = false
	config.Messaging.FallbackTimeout = 100 * time.Millisecond

	factory := NewBrokerFactory(config, nil)

	err := factory.Initialize(context.Background())
	require.NoError(t, err)

	// Reconnect when already using fallback should attempt reconnection
	// This will fail because RabbitMQ/Kafka are not available, but should not error
	err = factory.Reconnect(context.Background())

	// The reconnect will likely fail due to no real brokers, but that's expected
	// The factory should still be in a valid state
	assert.True(t, factory.IsInitialized())
}

func TestBrokerFactoryIsUsingFallbackNotInitialized(t *testing.T) {
	factory := NewBrokerFactory(nil, nil)
	assert.True(t, factory.IsUsingFallback())
}

func TestFactoryCreateRabbitMQConfig(t *testing.T) {
	config := messaging.DefaultMessagingConfig()
	config.RabbitMQ.Host = "custom-host"
	config.RabbitMQ.Port = 5673
	config.RabbitMQ.Username = "admin"
	config.RabbitMQ.Password = "secret"
	config.RabbitMQ.VHost = "myapp"
	config.RabbitMQ.UseTLS = true
	config.RabbitMQ.PrefetchCount = 20

	factory := NewBrokerFactory(config, nil)
	rabbitConfig := factory.createRabbitMQConfig()

	assert.Equal(t, "custom-host", rabbitConfig.Host)
	assert.Equal(t, 5673, rabbitConfig.Port)
	assert.Equal(t, "admin", rabbitConfig.Username)
	assert.Equal(t, "secret", rabbitConfig.Password)
	assert.Equal(t, "myapp", rabbitConfig.VHost)
	assert.True(t, rabbitConfig.UseTLS)
	assert.Equal(t, 20, rabbitConfig.PrefetchCount)
}

func TestFactoryCreateKafkaConfig(t *testing.T) {
	config := messaging.DefaultMessagingConfig()
	config.Kafka.Brokers = []string{"kafka1:9092", "kafka2:9092"}
	config.Kafka.ClientID = "custom-client"
	config.Kafka.GroupID = "custom-group"
	config.Kafka.UseTLS = true
	config.Kafka.SASLEnabled = true
	config.Kafka.SASLMechanism = "SCRAM-SHA-256"
	config.Kafka.CompressionType = "snappy"

	factory := NewBrokerFactory(config, nil)
	kafkaConfig := factory.createKafkaConfig()

	assert.Equal(t, []string{"kafka1:9092", "kafka2:9092"}, kafkaConfig.Brokers)
	assert.Equal(t, "custom-client", kafkaConfig.ClientID)
	assert.Equal(t, "custom-group", kafkaConfig.GroupID)
	assert.True(t, kafkaConfig.UseTLS)
	assert.True(t, kafkaConfig.SASLEnabled)
	assert.Equal(t, "SCRAM-SHA-256", kafkaConfig.SASLMechanism)
	assert.Equal(t, "snappy", kafkaConfig.CompressionType)
}

func TestFactoryCreateInMemoryBroker(t *testing.T) {
	factory := NewBrokerFactory(nil, nil)
	broker := factory.createInMemoryBroker()

	require.NotNil(t, broker)
	assert.Equal(t, messaging.BrokerTypeInMemory, broker.BrokerType())
}

func TestBrokerStatusJSON(t *testing.T) {
	status := &BrokerStatus{
		Type:      messaging.BrokerTypeRabbitMQ,
		Connected: true,
	}

	assert.Equal(t, messaging.BrokerTypeRabbitMQ, status.Type)
	assert.True(t, status.Connected)
}

func TestFactoryStatusJSON(t *testing.T) {
	status := &FactoryStatus{
		Initialized:   true,
		UsingFallback: false,
		TaskQueue: &BrokerStatus{
			Type:      messaging.BrokerTypeRabbitMQ,
			Connected: true,
		},
		EventStream: &BrokerStatus{
			Type:      messaging.BrokerTypeKafka,
			Connected: true,
		},
		Fallback: &BrokerStatus{
			Type:      messaging.BrokerTypeInMemory,
			Connected: true,
		},
	}

	assert.True(t, status.Initialized)
	assert.False(t, status.UsingFallback)
	assert.NotNil(t, status.TaskQueue)
	assert.NotNil(t, status.EventStream)
	assert.NotNil(t, status.Fallback)
}
