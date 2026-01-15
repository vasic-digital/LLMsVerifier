// Package rabbitmq provides a RabbitMQ implementation of the message broker interfaces.
package rabbitmq

import (
	"crypto/tls"
	"time"
)

// Config contains configuration for the RabbitMQ broker.
type Config struct {
	// Host is the RabbitMQ server hostname.
	Host string `json:"host" yaml:"host"`

	// Port is the RabbitMQ server port.
	Port int `json:"port" yaml:"port"`

	// Username for authentication.
	Username string `json:"username" yaml:"username"`

	// Password for authentication.
	Password string `json:"password" yaml:"password"`

	// VHost is the virtual host to connect to.
	VHost string `json:"vhost" yaml:"vhost"`

	// TLS configuration for secure connections.
	TLS *tls.Config `json:"-" yaml:"-"`

	// UseTLS enables TLS for the connection.
	UseTLS bool `json:"use_tls" yaml:"use_tls"`

	// ConnectionTimeout is the timeout for establishing a connection.
	ConnectionTimeout time.Duration `json:"connection_timeout" yaml:"connection_timeout"`

	// HeartbeatInterval is the interval for heartbeat frames.
	HeartbeatInterval time.Duration `json:"heartbeat_interval" yaml:"heartbeat_interval"`

	// ReconnectInterval is the initial delay between reconnection attempts.
	ReconnectInterval time.Duration `json:"reconnect_interval" yaml:"reconnect_interval"`

	// MaxReconnectInterval is the maximum delay between reconnection attempts.
	MaxReconnectInterval time.Duration `json:"max_reconnect_interval" yaml:"max_reconnect_interval"`

	// MaxReconnectAttempts is the maximum number of reconnection attempts (0 = infinite).
	MaxReconnectAttempts int `json:"max_reconnect_attempts" yaml:"max_reconnect_attempts"`

	// ChannelMax is the maximum number of channels (0 = no limit).
	ChannelMax int `json:"channel_max" yaml:"channel_max"`

	// FrameSize is the maximum frame size in bytes.
	FrameSize int `json:"frame_size" yaml:"frame_size"`

	// PrefetchCount is the number of messages to prefetch per consumer.
	PrefetchCount int `json:"prefetch_count" yaml:"prefetch_count"`

	// PrefetchSize is the prefetch size in bytes (0 = no limit).
	PrefetchSize int `json:"prefetch_size" yaml:"prefetch_size"`

	// PublisherConfirms enables publisher confirms mode.
	PublisherConfirms bool `json:"publisher_confirms" yaml:"publisher_confirms"`

	// ConfirmTimeout is the timeout for publisher confirm acknowledgments.
	ConfirmTimeout time.Duration `json:"confirm_timeout" yaml:"confirm_timeout"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Host:                 "localhost",
		Port:                 5672,
		Username:             "guest",
		Password:             "guest",
		VHost:                "/",
		UseTLS:               false,
		ConnectionTimeout:    30 * time.Second,
		HeartbeatInterval:    10 * time.Second,
		ReconnectInterval:    1 * time.Second,
		MaxReconnectInterval: 30 * time.Second,
		MaxReconnectAttempts: 0, // Infinite
		ChannelMax:           0, // No limit
		FrameSize:            0, // Use server default
		PrefetchCount:        10,
		PrefetchSize:         0,
		PublisherConfirms:    true,
		ConfirmTimeout:       5 * time.Second,
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Host == "" {
		return ErrInvalidHost
	}
	if c.Port <= 0 || c.Port > 65535 {
		return ErrInvalidPort
	}
	if c.ConnectionTimeout <= 0 {
		return ErrInvalidTimeout
	}
	return nil
}

// AMQPURI returns the AMQP connection URI.
func (c *Config) AMQPURI() string {
	scheme := "amqp"
	if c.UseTLS {
		scheme = "amqps"
	}

	// Build URI: amqp://user:pass@host:port/vhost
	uri := scheme + "://"
	if c.Username != "" {
		uri += c.Username
		if c.Password != "" {
			uri += ":" + c.Password
		}
		uri += "@"
	}
	uri += c.Host
	if c.Port != 0 {
		uri += ":" + intToString(c.Port)
	}
	if c.VHost != "" && c.VHost != "/" {
		uri += "/" + c.VHost
	}

	return uri
}

// intToString converts an int to string without importing strconv.
func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToString(-n)
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// ConfigOption is a function that modifies Config.
type ConfigOption func(*Config)

// WithHost sets the host.
func WithHost(host string) ConfigOption {
	return func(c *Config) {
		c.Host = host
	}
}

// WithPort sets the port.
func WithPort(port int) ConfigOption {
	return func(c *Config) {
		c.Port = port
	}
}

// WithCredentials sets the username and password.
func WithCredentials(username, password string) ConfigOption {
	return func(c *Config) {
		c.Username = username
		c.Password = password
	}
}

// WithVHost sets the virtual host.
func WithVHost(vhost string) ConfigOption {
	return func(c *Config) {
		c.VHost = vhost
	}
}

// WithTLS enables TLS with the provided configuration.
func WithTLS(tlsConfig *tls.Config) ConfigOption {
	return func(c *Config) {
		c.UseTLS = true
		c.TLS = tlsConfig
	}
}

// WithPrefetch sets the prefetch count.
func WithPrefetch(count int) ConfigOption {
	return func(c *Config) {
		c.PrefetchCount = count
	}
}

// WithPublisherConfirms enables or disables publisher confirms.
func WithPublisherConfirms(enabled bool) ConfigOption {
	return func(c *Config) {
		c.PublisherConfirms = enabled
	}
}

// WithConnectionTimeout sets the connection timeout.
func WithConnectionTimeout(timeout time.Duration) ConfigOption {
	return func(c *Config) {
		c.ConnectionTimeout = timeout
	}
}

// WithReconnect configures reconnection behavior.
func WithReconnect(interval, maxInterval time.Duration, maxAttempts int) ConfigOption {
	return func(c *Config) {
		c.ReconnectInterval = interval
		c.MaxReconnectInterval = maxInterval
		c.MaxReconnectAttempts = maxAttempts
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
	ErrInvalidHost    = ConfigError("invalid or empty host")
	ErrInvalidPort    = ConfigError("invalid port number")
	ErrInvalidTimeout = ConfigError("invalid timeout value")
)

// ConfigError represents a configuration error.
type ConfigError string

func (e ConfigError) Error() string {
	return string(e)
}
