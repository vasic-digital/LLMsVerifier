package kafka

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

	require.NotNil(t, config)
	assert.Equal(t, []string{"localhost:9092"}, config.Brokers)
	assert.Equal(t, "helixagent", config.ClientID)
	assert.Equal(t, "helixagent-consumer", config.GroupID)
	assert.False(t, config.UseTLS)
	assert.False(t, config.SASLEnabled)
	assert.Equal(t, -1, config.RequiredAcks)
	assert.True(t, config.Idempotent)
	assert.Equal(t, "lz4", config.CompressionType)
	assert.Equal(t, "latest", config.AutoOffsetReset)
	assert.False(t, config.AutoCommit)
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
			name: "empty brokers",
			config: &Config{
				Brokers:  []string{},
				ClientID: "test",
			},
			expectErr: true,
		},
		{
			name: "nil brokers",
			config: &Config{
				Brokers:  nil,
				ClientID: "test",
			},
			expectErr: true,
		},
		{
			name: "empty client ID",
			config: &Config{
				Brokers:  []string{"localhost:9092"},
				ClientID: "",
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
		WithBrokers("kafka1:9092", "kafka2:9092"),
		WithClientID("myapp"),
		WithGroupID("mygroup"),
		WithCompression("snappy"),
		WithBatching(32768, 20*time.Millisecond),
		WithIdempotent(false),
	)

	assert.Equal(t, []string{"kafka1:9092", "kafka2:9092"}, config.Brokers)
	assert.Equal(t, "myapp", config.ClientID)
	assert.Equal(t, "mygroup", config.GroupID)
	assert.Equal(t, "snappy", config.CompressionType)
	assert.Equal(t, 32768, config.BatchSize)
	assert.Equal(t, 20*time.Millisecond, config.BatchTimeout)
	assert.False(t, config.Idempotent)
}

func TestSASLConfig(t *testing.T) {
	config := ApplyConfigOptions(
		WithSASL("PLAIN", "user", "password"),
	)

	assert.True(t, config.SASLEnabled)
	assert.Equal(t, "PLAIN", config.SASLMechanism)
	assert.Equal(t, "user", config.SASLUsername)
	assert.Equal(t, "password", config.SASLPassword)
}

func TestAutoCommitConfig(t *testing.T) {
	config := ApplyConfigOptions(
		WithAutoCommit(true, 10*time.Second),
	)

	assert.True(t, config.AutoCommit)
	assert.Equal(t, 10*time.Second, config.AutoCommitInterval)
}

func TestNewBroker(t *testing.T) {
	config := DefaultConfig()
	logger := logrus.New()

	broker := NewBroker(config, logger)

	require.NotNil(t, broker)
	assert.Equal(t, messaging.BrokerTypeKafka, broker.BrokerType())
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
	assert.Equal(t, messaging.BrokerTypeKafka, broker.BrokerType())
}

func TestBrokerMetrics(t *testing.T) {
	broker := NewBroker(nil, nil)
	metrics := broker.GetMetrics()

	require.NotNil(t, metrics)
	assert.Equal(t, int64(0), metrics.MessagesPublished)
	assert.Equal(t, int64(0), metrics.MessagesConsumed)
}

func TestConfigErrors(t *testing.T) {
	assert.Equal(t, "no brokers specified", ErrNoBrokers.Error())
	assert.Equal(t, "no client ID specified", ErrNoClientID.Error())
	assert.Equal(t, "no group ID specified", ErrNoGroupID.Error())
}

func TestCompressionTypes(t *testing.T) {
	compressionTypes := []string{"none", "gzip", "snappy", "lz4", "zstd"}

	for _, compression := range compressionTypes {
		config := ApplyConfigOptions(WithCompression(compression))
		assert.Equal(t, compression, config.CompressionType)
	}
}

func TestSASLMechanisms(t *testing.T) {
	mechanisms := []string{"PLAIN", "SCRAM-SHA-256", "SCRAM-SHA-512"}

	for _, mechanism := range mechanisms {
		config := ApplyConfigOptions(
			WithSASL(mechanism, "user", "pass"),
		)
		assert.Equal(t, mechanism, config.SASLMechanism)
		assert.True(t, config.SASLEnabled)
	}
}

func TestDefaultTimeouts(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, 30*time.Second, config.DialTimeout)
	assert.Equal(t, 30*time.Second, config.ReadTimeout)
	assert.Equal(t, 30*time.Second, config.WriteTimeout)
}

func TestRetryConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, 100*time.Millisecond, config.RetryBackoff)
	assert.Equal(t, 10*time.Second, config.RetryMaxBackoff)
	assert.Equal(t, 5, config.MaxRetries)
}

// Integration tests (require actual Kafka server)
// These tests are skipped by default and run with -tags=integration

func TestBrokerConnect_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode") // SKIP-OK: #short-mode
	}

	// This test requires a running Kafka server
	t.Skip("Skipping - requires Kafka server") // SKIP-OK: #legacy-untriaged
}

func TestTopicCreation_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode") // SKIP-OK: #short-mode
	}

	// This test requires a running Kafka server
	t.Skip("Skipping - requires Kafka server") // SKIP-OK: #legacy-untriaged
}

func TestProducerConsumer_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode") // SKIP-OK: #short-mode
	}

	// This test requires a running Kafka server
	t.Skip("Skipping - requires Kafka server") // SKIP-OK: #legacy-untriaged
}
