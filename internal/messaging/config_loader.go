package messaging

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// MessagingConfig represents the complete messaging configuration.
type MessagingConfig struct {
	Messaging       MessagingSettings       `yaml:"messaging"`
	RabbitMQ        RabbitMQConfig          `yaml:"rabbitmq"`
	Kafka           KafkaConfig             `yaml:"kafka"`
	CircuitBreaker  CircuitBreakerSettings  `yaml:"circuit_breaker"`
	Retry           RetrySettings           `yaml:"retry"`
	HealthCheck     HealthCheckSettings     `yaml:"health_check"`
}

// MessagingSettings contains general messaging settings.
type MessagingSettings struct {
	Enabled         bool          `yaml:"enabled"`
	FallbackEnabled bool          `yaml:"fallback_enabled"`
	FallbackTimeout time.Duration `yaml:"fallback_timeout"`
}

// RabbitMQConfig contains RabbitMQ-specific configuration.
type RabbitMQConfig struct {
	Host                 string              `yaml:"host"`
	Port                 int                 `yaml:"port"`
	Username             string              `yaml:"username"`
	Password             string              `yaml:"password"`
	VHost                string              `yaml:"vhost"`
	UseTLS               bool                `yaml:"use_tls"`
	TLSCertFile          string              `yaml:"tls_cert_file"`
	TLSKeyFile           string              `yaml:"tls_key_file"`
	TLSCAFile            string              `yaml:"tls_ca_file"`
	ConnectionTimeout    time.Duration       `yaml:"connection_timeout"`
	HeartbeatInterval    time.Duration       `yaml:"heartbeat_interval"`
	ReconnectInterval    time.Duration       `yaml:"reconnect_interval"`
	MaxReconnectInterval time.Duration       `yaml:"max_reconnect_interval"`
	MaxReconnectAttempts int                 `yaml:"max_reconnect_attempts"`
	ChannelMax           int                 `yaml:"channel_max"`
	FrameSize            int                 `yaml:"frame_size"`
	PrefetchCount        int                 `yaml:"prefetch_count"`
	PrefetchSize         int                 `yaml:"prefetch_size"`
	PublisherConfirms    bool                `yaml:"publisher_confirms"`
	ConfirmTimeout       time.Duration       `yaml:"confirm_timeout"`
	Queues               []QueueDefinition   `yaml:"queues"`
	Exchanges            []ExchangeDefinition `yaml:"exchanges"`
}

// QueueDefinition defines a RabbitMQ queue.
type QueueDefinition struct {
	Name                 string `yaml:"name"`
	Durable              bool   `yaml:"durable"`
	AutoDelete           bool   `yaml:"auto_delete"`
	DeadLetterExchange   string `yaml:"dead_letter_exchange"`
	DeadLetterRoutingKey string `yaml:"dead_letter_routing_key"`
	MessageTTL           int64  `yaml:"message_ttl"`
}

// ExchangeDefinition defines a RabbitMQ exchange.
type ExchangeDefinition struct {
	Name       string `yaml:"name"`
	Type       string `yaml:"type"`
	Durable    bool   `yaml:"durable"`
	AutoDelete bool   `yaml:"auto_delete"`
}

// KafkaConfig contains Kafka-specific configuration.
type KafkaConfig struct {
	Brokers            []string          `yaml:"brokers"`
	ClientID           string            `yaml:"client_id"`
	GroupID            string            `yaml:"group_id"`
	UseTLS             bool              `yaml:"use_tls"`
	TLSCertFile        string            `yaml:"tls_cert_file"`
	TLSKeyFile         string            `yaml:"tls_key_file"`
	TLSCAFile          string            `yaml:"tls_ca_file"`
	SASLEnabled        bool              `yaml:"sasl_enabled"`
	SASLMechanism      string            `yaml:"sasl_mechanism"`
	SASLUsername       string            `yaml:"sasl_username"`
	SASLPassword       string            `yaml:"sasl_password"`
	RequiredAcks       int               `yaml:"required_acks"`
	Idempotent         bool              `yaml:"idempotent"`
	BatchSize          int               `yaml:"batch_size"`
	BatchTimeout       time.Duration     `yaml:"batch_timeout"`
	CompressionType    string            `yaml:"compression_type"`
	FetchMinBytes      int               `yaml:"fetch_min_bytes"`
	FetchMaxBytes      int               `yaml:"fetch_max_bytes"`
	MaxWaitTime        time.Duration     `yaml:"max_wait_time"`
	AutoOffsetReset    string            `yaml:"auto_offset_reset"`
	AutoCommit         bool              `yaml:"auto_commit"`
	AutoCommitInterval time.Duration     `yaml:"auto_commit_interval"`
	DialTimeout        time.Duration     `yaml:"dial_timeout"`
	ReadTimeout        time.Duration     `yaml:"read_timeout"`
	WriteTimeout       time.Duration     `yaml:"write_timeout"`
	MetadataMaxRetries int               `yaml:"metadata_max_retries"`
	RetryBackoff       time.Duration     `yaml:"retry_backoff"`
	RetryMaxBackoff    time.Duration     `yaml:"retry_max_backoff"`
	MaxRetries         int               `yaml:"max_retries"`
	Topics             []TopicDefinition `yaml:"topics"`
}

// TopicDefinition defines a Kafka topic.
type TopicDefinition struct {
	Name              string `yaml:"name"`
	Partitions        int    `yaml:"partitions"`
	ReplicationFactor int    `yaml:"replication_factor"`
	RetentionHours    int    `yaml:"retention_hours"`
}

// CircuitBreakerSettings contains circuit breaker configuration.
type CircuitBreakerSettings struct {
	Enabled          bool          `yaml:"enabled"`
	FailureThreshold int           `yaml:"failure_threshold"`
	SuccessThreshold int           `yaml:"success_threshold"`
	Timeout          time.Duration `yaml:"timeout"`
}

// RetrySettings contains retry configuration.
type RetrySettings struct {
	MaxRetries     int           `yaml:"max_retries"`
	InitialBackoff time.Duration `yaml:"initial_backoff"`
	MaxBackoff     time.Duration `yaml:"max_backoff"`
	BackoffFactor  float64       `yaml:"backoff_factor"`
}

// HealthCheckSettings contains health check configuration.
type HealthCheckSettings struct {
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
}

// LoadConfig loads messaging configuration from a YAML file.
func LoadConfig(path string) (*MessagingConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables
	expanded := os.ExpandEnv(string(data))

	config := &MessagingConfig{}
	if err := yaml.Unmarshal([]byte(expanded), config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults for missing values
	config.setDefaults()

	return config, nil
}

// LoadConfigFromBytes loads messaging configuration from YAML bytes.
func LoadConfigFromBytes(data []byte) (*MessagingConfig, error) {
	// Expand environment variables
	expanded := os.ExpandEnv(string(data))

	config := &MessagingConfig{}
	if err := yaml.Unmarshal([]byte(expanded), config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set defaults for missing values
	config.setDefaults()

	return config, nil
}

// setDefaults sets default values for missing configuration.
func (c *MessagingConfig) setDefaults() {
	// Messaging defaults
	if c.Messaging.FallbackTimeout == 0 {
		c.Messaging.FallbackTimeout = 5 * time.Second
	}

	// RabbitMQ defaults
	if c.RabbitMQ.Host == "" {
		c.RabbitMQ.Host = "localhost"
	}
	if c.RabbitMQ.Port == 0 {
		c.RabbitMQ.Port = 5672
	}
	if c.RabbitMQ.Username == "" {
		c.RabbitMQ.Username = "guest"
	}
	if c.RabbitMQ.Password == "" {
		c.RabbitMQ.Password = "guest"
	}
	if c.RabbitMQ.VHost == "" {
		c.RabbitMQ.VHost = "/"
	}
	if c.RabbitMQ.ConnectionTimeout == 0 {
		c.RabbitMQ.ConnectionTimeout = 30 * time.Second
	}
	if c.RabbitMQ.HeartbeatInterval == 0 {
		c.RabbitMQ.HeartbeatInterval = 10 * time.Second
	}
	if c.RabbitMQ.ReconnectInterval == 0 {
		c.RabbitMQ.ReconnectInterval = 1 * time.Second
	}
	if c.RabbitMQ.MaxReconnectInterval == 0 {
		c.RabbitMQ.MaxReconnectInterval = 30 * time.Second
	}
	if c.RabbitMQ.PrefetchCount == 0 {
		c.RabbitMQ.PrefetchCount = 10
	}
	if c.RabbitMQ.ConfirmTimeout == 0 {
		c.RabbitMQ.ConfirmTimeout = 5 * time.Second
	}

	// Kafka defaults
	if len(c.Kafka.Brokers) == 0 {
		c.Kafka.Brokers = []string{"localhost:9092"}
	}
	if c.Kafka.ClientID == "" {
		c.Kafka.ClientID = "llmsverifier"
	}
	if c.Kafka.GroupID == "" {
		c.Kafka.GroupID = "llmsverifier-consumer"
	}
	if c.Kafka.RequiredAcks == 0 {
		c.Kafka.RequiredAcks = -1 // All replicas
	}
	if c.Kafka.BatchSize == 0 {
		c.Kafka.BatchSize = 16384
	}
	if c.Kafka.BatchTimeout == 0 {
		c.Kafka.BatchTimeout = 10 * time.Millisecond
	}
	if c.Kafka.CompressionType == "" {
		c.Kafka.CompressionType = "lz4"
	}
	if c.Kafka.FetchMinBytes == 0 {
		c.Kafka.FetchMinBytes = 1
	}
	if c.Kafka.FetchMaxBytes == 0 {
		c.Kafka.FetchMaxBytes = 10 * 1024 * 1024 // 10MB
	}
	if c.Kafka.MaxWaitTime == 0 {
		c.Kafka.MaxWaitTime = 1 * time.Second
	}
	if c.Kafka.AutoOffsetReset == "" {
		c.Kafka.AutoOffsetReset = "latest"
	}
	if c.Kafka.AutoCommitInterval == 0 {
		c.Kafka.AutoCommitInterval = 5 * time.Second
	}
	if c.Kafka.DialTimeout == 0 {
		c.Kafka.DialTimeout = 30 * time.Second
	}
	if c.Kafka.ReadTimeout == 0 {
		c.Kafka.ReadTimeout = 30 * time.Second
	}
	if c.Kafka.WriteTimeout == 0 {
		c.Kafka.WriteTimeout = 30 * time.Second
	}
	if c.Kafka.MetadataMaxRetries == 0 {
		c.Kafka.MetadataMaxRetries = 3
	}
	if c.Kafka.RetryBackoff == 0 {
		c.Kafka.RetryBackoff = 100 * time.Millisecond
	}
	if c.Kafka.RetryMaxBackoff == 0 {
		c.Kafka.RetryMaxBackoff = 10 * time.Second
	}
	if c.Kafka.MaxRetries == 0 {
		c.Kafka.MaxRetries = 5
	}

	// Circuit breaker defaults
	if c.CircuitBreaker.FailureThreshold == 0 {
		c.CircuitBreaker.FailureThreshold = 5
	}
	if c.CircuitBreaker.SuccessThreshold == 0 {
		c.CircuitBreaker.SuccessThreshold = 2
	}
	if c.CircuitBreaker.Timeout == 0 {
		c.CircuitBreaker.Timeout = 30 * time.Second
	}

	// Retry defaults
	if c.Retry.MaxRetries == 0 {
		c.Retry.MaxRetries = 3
	}
	if c.Retry.InitialBackoff == 0 {
		c.Retry.InitialBackoff = 100 * time.Millisecond
	}
	if c.Retry.MaxBackoff == 0 {
		c.Retry.MaxBackoff = 10 * time.Second
	}
	if c.Retry.BackoffFactor == 0 {
		c.Retry.BackoffFactor = 2.0
	}

	// Health check defaults
	if c.HealthCheck.Interval == 0 {
		c.HealthCheck.Interval = 30 * time.Second
	}
	if c.HealthCheck.Timeout == 0 {
		c.HealthCheck.Timeout = 5 * time.Second
	}
}

// Validate validates the messaging configuration.
func (c *MessagingConfig) Validate() error {
	if c.Messaging.Enabled {
		// Validate RabbitMQ config
		if c.RabbitMQ.Host == "" {
			return fmt.Errorf("rabbitmq host is required")
		}
		if c.RabbitMQ.Port <= 0 || c.RabbitMQ.Port > 65535 {
			return fmt.Errorf("invalid rabbitmq port: %d", c.RabbitMQ.Port)
		}

		// Validate Kafka config
		if len(c.Kafka.Brokers) == 0 {
			return fmt.Errorf("at least one kafka broker is required")
		}
		if c.Kafka.ClientID == "" {
			return fmt.Errorf("kafka client_id is required")
		}
	}

	return nil
}

// DefaultMessagingConfig returns a configuration with sensible defaults.
func DefaultMessagingConfig() *MessagingConfig {
	config := &MessagingConfig{
		Messaging: MessagingSettings{
			Enabled:         true,
			FallbackEnabled: true,
			FallbackTimeout: 5 * time.Second,
		},
	}
	config.setDefaults()
	return config
}
