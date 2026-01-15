// Package messaging provides message broker integration for LLMsVerifier.
package messaging

import (
	"errors"
	"time"
)

// BrokerType represents the type of message broker.
type BrokerType string

const (
	// BrokerTypeKafka uses Apache Kafka for event streaming.
	BrokerTypeKafka BrokerType = "kafka"

	// BrokerTypeRabbitMQ uses RabbitMQ for task queuing.
	BrokerTypeRabbitMQ BrokerType = "rabbitmq"

	// BrokerTypeNone disables messaging.
	BrokerTypeNone BrokerType = "none"
)

// Config holds the messaging configuration.
type Config struct {
	// Enabled determines if messaging is enabled.
	Enabled bool `json:"enabled" yaml:"enabled"`

	// BrokerType specifies which message broker to use.
	BrokerType BrokerType `json:"broker_type" yaml:"broker_type"`

	// Kafka configuration
	Kafka KafkaConfig `json:"kafka" yaml:"kafka"`

	// RabbitMQ configuration
	RabbitMQ RabbitMQConfig `json:"rabbitmq" yaml:"rabbitmq"`

	// PublishTimeout is the timeout for publishing messages.
	PublishTimeout time.Duration `json:"publish_timeout" yaml:"publish_timeout"`

	// RetryOnError determines if publishing should be retried on error.
	RetryOnError bool `json:"retry_on_error" yaml:"retry_on_error"`

	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int `json:"max_retries" yaml:"max_retries"`

	// RetryDelay is the delay between retry attempts.
	RetryDelay time.Duration `json:"retry_delay" yaml:"retry_delay"`

	// AsyncPublish determines if messages are published asynchronously.
	AsyncPublish bool `json:"async_publish" yaml:"async_publish"`

	// BufferSize is the size of the async publish buffer.
	BufferSize int `json:"buffer_size" yaml:"buffer_size"`
}

// KafkaConfig holds Kafka-specific configuration.
type KafkaConfig struct {
	// Brokers is the list of Kafka broker addresses.
	Brokers []string `json:"brokers" yaml:"brokers"`

	// Topic is the default topic for verification events.
	Topic string `json:"topic" yaml:"topic"`

	// ClientID is the client identifier.
	ClientID string `json:"client_id" yaml:"client_id"`

	// GroupID is the consumer group ID.
	GroupID string `json:"group_id" yaml:"group_id"`

	// TLS enables TLS encryption.
	TLS bool `json:"tls" yaml:"tls"`

	// TLSInsecure skips TLS certificate verification.
	TLSInsecure bool `json:"tls_insecure" yaml:"tls_insecure"`

	// SASLEnabled enables SASL authentication.
	SASLEnabled bool `json:"sasl_enabled" yaml:"sasl_enabled"`

	// SASLMechanism is the SASL mechanism (PLAIN, SCRAM-SHA-256, SCRAM-SHA-512).
	SASLMechanism string `json:"sasl_mechanism" yaml:"sasl_mechanism"`

	// SASLUsername is the SASL username.
	SASLUsername string `json:"sasl_username" yaml:"sasl_username"`

	// SASLPassword is the SASL password.
	SASLPassword string `json:"sasl_password" yaml:"sasl_password"`

	// Compression is the compression codec (none, gzip, snappy, lz4, zstd).
	Compression string `json:"compression" yaml:"compression"`

	// BatchSize is the number of messages to batch together.
	BatchSize int `json:"batch_size" yaml:"batch_size"`

	// BatchTimeout is the maximum time to wait for a batch.
	BatchTimeout time.Duration `json:"batch_timeout" yaml:"batch_timeout"`
}

// RabbitMQConfig holds RabbitMQ-specific configuration.
type RabbitMQConfig struct {
	// Host is the RabbitMQ server host.
	Host string `json:"host" yaml:"host"`

	// Port is the RabbitMQ server port.
	Port int `json:"port" yaml:"port"`

	// Username is the RabbitMQ username.
	Username string `json:"username" yaml:"username"`

	// Password is the RabbitMQ password.
	Password string `json:"password" yaml:"password"`

	// VirtualHost is the RabbitMQ virtual host.
	VirtualHost string `json:"virtual_host" yaml:"virtual_host"`

	// Exchange is the exchange for verification events.
	Exchange string `json:"exchange" yaml:"exchange"`

	// ExchangeType is the exchange type (direct, topic, fanout).
	ExchangeType string `json:"exchange_type" yaml:"exchange_type"`

	// RoutingKey is the default routing key.
	RoutingKey string `json:"routing_key" yaml:"routing_key"`

	// TLS enables TLS encryption.
	TLS bool `json:"tls" yaml:"tls"`

	// TLSInsecure skips TLS certificate verification.
	TLSInsecure bool `json:"tls_insecure" yaml:"tls_insecure"`

	// PublisherConfirm enables publisher confirms.
	PublisherConfirm bool `json:"publisher_confirm" yaml:"publisher_confirm"`

	// ConnectTimeout is the connection timeout.
	ConnectTimeout time.Duration `json:"connect_timeout" yaml:"connect_timeout"`

	// Heartbeat is the connection heartbeat interval.
	Heartbeat time.Duration `json:"heartbeat" yaml:"heartbeat"`
}

// DefaultConfig returns the default messaging configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled:        false,
		BrokerType:     BrokerTypeNone,
		PublishTimeout: 30 * time.Second,
		RetryOnError:   true,
		MaxRetries:     3,
		RetryDelay:     1 * time.Second,
		AsyncPublish:   true,
		BufferSize:     1000,
		Kafka:          DefaultKafkaConfig(),
		RabbitMQ:       DefaultRabbitMQConfig(),
	}
}

// DefaultKafkaConfig returns the default Kafka configuration.
func DefaultKafkaConfig() KafkaConfig {
	return KafkaConfig{
		Brokers:       []string{"localhost:9092"},
		Topic:         "llmsverifier.events",
		ClientID:      "llmsverifier",
		GroupID:       "llmsverifier-group",
		TLS:           false,
		SASLEnabled:   false,
		SASLMechanism: "PLAIN",
		Compression:   "lz4",
		BatchSize:     100,
		BatchTimeout:  10 * time.Millisecond,
	}
}

// DefaultRabbitMQConfig returns the default RabbitMQ configuration.
func DefaultRabbitMQConfig() RabbitMQConfig {
	return RabbitMQConfig{
		Host:             "localhost",
		Port:             5672,
		Username:         "guest",
		Password:         "guest",
		VirtualHost:      "/",
		Exchange:         "llmsverifier.events",
		ExchangeType:     "topic",
		RoutingKey:       "verification.#",
		TLS:              false,
		PublisherConfirm: true,
		ConnectTimeout:   30 * time.Second,
		Heartbeat:        60 * time.Second,
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	switch c.BrokerType {
	case BrokerTypeKafka:
		return c.Kafka.Validate()
	case BrokerTypeRabbitMQ:
		return c.RabbitMQ.Validate()
	case BrokerTypeNone:
		return nil
	default:
		return errors.New("invalid broker type")
	}
}

// Validate validates the Kafka configuration.
func (c *KafkaConfig) Validate() error {
	if len(c.Brokers) == 0 {
		return errors.New("kafka: at least one broker is required")
	}
	for _, broker := range c.Brokers {
		if broker == "" {
			return errors.New("kafka: broker address cannot be empty")
		}
	}
	if c.Topic == "" {
		return errors.New("kafka: topic is required")
	}
	if c.ClientID == "" {
		return errors.New("kafka: client ID is required")
	}
	if c.SASLEnabled && c.SASLUsername == "" {
		return errors.New("kafka: SASL username is required when SASL is enabled")
	}
	return nil
}

// Validate validates the RabbitMQ configuration.
func (c *RabbitMQConfig) Validate() error {
	if c.Host == "" {
		return errors.New("rabbitmq: host is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return errors.New("rabbitmq: invalid port")
	}
	if c.Exchange == "" {
		return errors.New("rabbitmq: exchange is required")
	}
	return nil
}

// TopicConfig holds configuration for a Kafka topic.
type TopicConfig struct {
	Name              string `json:"name" yaml:"name"`
	NumPartitions     int    `json:"num_partitions" yaml:"num_partitions"`
	ReplicationFactor int    `json:"replication_factor" yaml:"replication_factor"`
}

// DefaultTopics returns the default Kafka topics for LLMsVerifier.
func DefaultTopics() []TopicConfig {
	return []TopicConfig{
		{Name: "llmsverifier.events.verification", NumPartitions: 3, ReplicationFactor: 1},
		{Name: "llmsverifier.events.provider", NumPartitions: 3, ReplicationFactor: 1},
		{Name: "llmsverifier.events.model", NumPartitions: 3, ReplicationFactor: 1},
		{Name: "llmsverifier.events.team", NumPartitions: 3, ReplicationFactor: 1},
		{Name: "llmsverifier.events.system", NumPartitions: 3, ReplicationFactor: 1},
	}
}
