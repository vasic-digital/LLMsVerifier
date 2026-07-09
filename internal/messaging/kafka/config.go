// Package kafka provides a Kafka implementation of the event streaming broker interfaces.
package kafka

import (
	"crypto/tls"
	"time"
)

// Config contains configuration for the Kafka broker.
type Config struct {
	// Brokers is a list of Kafka broker addresses.
	Brokers []string `json:"brokers" yaml:"brokers"`

	// ClientID identifies this client to Kafka.
	ClientID string `json:"client_id" yaml:"client_id"`

	// GroupID is the consumer group ID.
	GroupID string `json:"group_id" yaml:"group_id"`

	// TLS configuration for secure connections.
	TLS *tls.Config `json:"-" yaml:"-"`

	// UseTLS enables TLS for connections.
	UseTLS bool `json:"use_tls" yaml:"use_tls"`

	// SASL configuration
	SASLEnabled   bool   `json:"sasl_enabled" yaml:"sasl_enabled"`
	SASLMechanism string `json:"sasl_mechanism" yaml:"sasl_mechanism"` // PLAIN, SCRAM-SHA-256, SCRAM-SHA-512
	SASLUsername  string `json:"sasl_username" yaml:"sasl_username"`
	SASLPassword  string `json:"sasl_password" yaml:"sasl_password"`

	// Producer settings
	RequiredAcks    int           `json:"required_acks" yaml:"required_acks"` // -1 = all, 0 = none, 1 = leader
	Idempotent      bool          `json:"idempotent" yaml:"idempotent"`
	BatchSize       int           `json:"batch_size" yaml:"batch_size"`
	BatchTimeout    time.Duration `json:"batch_timeout" yaml:"batch_timeout"`
	CompressionType string        `json:"compression_type" yaml:"compression_type"` // none, gzip, snappy, lz4, zstd

	// Consumer settings
	FetchMinBytes      int           `json:"fetch_min_bytes" yaml:"fetch_min_bytes"`
	FetchMaxBytes      int           `json:"fetch_max_bytes" yaml:"fetch_max_bytes"`
	MaxWaitTime        time.Duration `json:"max_wait_time" yaml:"max_wait_time"`
	AutoOffsetReset    string        `json:"auto_offset_reset" yaml:"auto_offset_reset"` // earliest, latest
	AutoCommit         bool          `json:"auto_commit" yaml:"auto_commit"`
	AutoCommitInterval time.Duration `json:"auto_commit_interval" yaml:"auto_commit_interval"`

	// Connection settings
	DialTimeout        time.Duration `json:"dial_timeout" yaml:"dial_timeout"`
	ReadTimeout        time.Duration `json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout       time.Duration `json:"write_timeout" yaml:"write_timeout"`
	MetadataMaxRetries int           `json:"metadata_max_retries" yaml:"metadata_max_retries"`

	// Retry settings
	RetryBackoff    time.Duration `json:"retry_backoff" yaml:"retry_backoff"`
	RetryMaxBackoff time.Duration `json:"retry_max_backoff" yaml:"retry_max_backoff"`
	MaxRetries      int           `json:"max_retries" yaml:"max_retries"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Brokers:            []string{"localhost:9092"},
		ClientID:           "helixagent",
		GroupID:            "helixagent-consumer",
		UseTLS:             false,
		SASLEnabled:        false,
		RequiredAcks:       -1, // All replicas
		Idempotent:         true,
		BatchSize:          16384,
		BatchTimeout:       10 * time.Millisecond,
		CompressionType:    "lz4",
		FetchMinBytes:      1,
		FetchMaxBytes:      10 * 1024 * 1024, // 10MB
		MaxWaitTime:        1 * time.Second,
		AutoOffsetReset:    "latest",
		AutoCommit:         false,
		AutoCommitInterval: 5 * time.Second,
		DialTimeout:        30 * time.Second,
		ReadTimeout:        30 * time.Second,
		WriteTimeout:       30 * time.Second,
		MetadataMaxRetries: 3,
		RetryBackoff:       100 * time.Millisecond,
		RetryMaxBackoff:    10 * time.Second,
		MaxRetries:         5,
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if len(c.Brokers) == 0 {
		return ErrNoBrokers
	}
	if c.ClientID == "" {
		return ErrNoClientID
	}
	return nil
}

// ConfigOption is a function that modifies Config.
type ConfigOption func(*Config)

// WithBrokers sets the broker addresses.
func WithBrokers(brokers ...string) ConfigOption {
	return func(c *Config) {
		c.Brokers = brokers
	}
}

// WithClientID sets the client ID.
func WithClientID(id string) ConfigOption {
	return func(c *Config) {
		c.ClientID = id
	}
}

// WithGroupID sets the consumer group ID.
func WithGroupID(id string) ConfigOption {
	return func(c *Config) {
		c.GroupID = id
	}
}

// WithTLS enables TLS with the provided configuration.
func WithTLS(tlsConfig *tls.Config) ConfigOption {
	return func(c *Config) {
		c.UseTLS = true
		c.TLS = tlsConfig
	}
}

// WithSASL enables SASL authentication.
func WithSASL(mechanism, username, password string) ConfigOption {
	return func(c *Config) {
		c.SASLEnabled = true
		c.SASLMechanism = mechanism
		c.SASLUsername = username
		c.SASLPassword = password
	}
}

// WithCompression sets the compression type.
func WithCompression(compression string) ConfigOption {
	return func(c *Config) {
		c.CompressionType = compression
	}
}

// WithBatching configures producer batching.
func WithBatching(size int, timeout time.Duration) ConfigOption {
	return func(c *Config) {
		c.BatchSize = size
		c.BatchTimeout = timeout
	}
}

// WithAutoCommit configures auto-commit behavior.
func WithAutoCommit(enabled bool, interval time.Duration) ConfigOption {
	return func(c *Config) {
		c.AutoCommit = enabled
		c.AutoCommitInterval = interval
	}
}

// WithIdempotent enables or disables idempotent producer.
func WithIdempotent(enabled bool) ConfigOption {
	return func(c *Config) {
		c.Idempotent = enabled
	}
}

// ApplyConfigOptions applies options to a new Config.
func ApplyConfigOptions(opts ...ConfigOption) *Config {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}
	return config
}

// Error types for configuration validation.
var (
	ErrNoBrokers  = ConfigError("no brokers specified")
	ErrNoClientID = ConfigError("no client ID specified")
	ErrNoGroupID  = ConfigError("no group ID specified")
)

// ConfigError represents a configuration error.
type ConfigError string

func (e ConfigError) Error() string {
	return string(e)
}
