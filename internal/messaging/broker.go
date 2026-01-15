// Package messaging provides unified message broker abstractions for RabbitMQ and Kafka integration.
package messaging

import (
	"context"
	"time"
)

// BrokerType identifies the type of message broker.
type BrokerType string

const (
	// BrokerTypeRabbitMQ represents a RabbitMQ broker.
	BrokerTypeRabbitMQ BrokerType = "rabbitmq"
	// BrokerTypeKafka represents a Kafka broker.
	BrokerTypeKafka BrokerType = "kafka"
	// BrokerTypeInMemory represents an in-memory broker for testing/fallback.
	BrokerTypeInMemory BrokerType = "inmemory"
)

// Priority levels for messages.
const (
	PriorityLow      = 1
	PriorityNormal   = 5
	PriorityHigh     = 8
	PriorityCritical = 10
)

// Message represents a message to be published or consumed.
type Message struct {
	// ID is a unique identifier for the message.
	ID string `json:"id"`

	// Type identifies the message type for routing and handling.
	Type string `json:"type"`

	// Payload contains the message data.
	Payload []byte `json:"payload"`

	// Headers contains metadata key-value pairs.
	Headers map[string]string `json:"headers,omitempty"`

	// Timestamp is when the message was created.
	Timestamp time.Time `json:"timestamp"`

	// Priority indicates message priority (1-10, higher = more urgent).
	Priority int `json:"priority,omitempty"`

	// RetryCount tracks how many times delivery has been attempted.
	RetryCount int `json:"retry_count,omitempty"`

	// MaxRetries is the maximum number of delivery attempts.
	MaxRetries int `json:"max_retries,omitempty"`

	// TraceID for distributed tracing correlation.
	TraceID string `json:"trace_id,omitempty"`

	// CorrelationID for request-response patterns.
	CorrelationID string `json:"correlation_id,omitempty"`

	// ReplyTo specifies where responses should be sent.
	ReplyTo string `json:"reply_to,omitempty"`

	// Expiration is when the message should expire (zero = never).
	Expiration time.Duration `json:"expiration,omitempty"`

	// Key is the partition key for Kafka (determines partition assignment).
	Key string `json:"key,omitempty"`

	// DeliveryTag is the broker-specific delivery identifier (for ack/nack).
	DeliveryTag uint64 `json:"-"`

	// Partition is the Kafka partition the message was received from.
	Partition int32 `json:"partition,omitempty"`

	// Offset is the Kafka offset of the message.
	Offset int64 `json:"offset,omitempty"`
}

// MessageHandler is a function that processes a message.
type MessageHandler func(ctx context.Context, msg *Message) error

// Subscription represents an active subscription to a topic or queue.
type Subscription interface {
	// Unsubscribe cancels the subscription.
	Unsubscribe() error

	// IsActive returns true if the subscription is still active.
	IsActive() bool

	// Topic returns the topic or queue name being subscribed to.
	Topic() string
}

// MessageBroker is the core interface for message broker implementations.
type MessageBroker interface {
	// Connect establishes a connection to the broker.
	Connect(ctx context.Context) error

	// Close terminates the connection to the broker.
	Close(ctx context.Context) error

	// HealthCheck verifies the broker is healthy and accessible.
	HealthCheck(ctx context.Context) error

	// IsConnected returns true if currently connected to the broker.
	IsConnected() bool

	// Publish sends a single message to a topic or queue.
	Publish(ctx context.Context, topic string, message *Message, opts ...PublishOption) error

	// PublishBatch sends multiple messages to a topic or queue.
	PublishBatch(ctx context.Context, topic string, messages []*Message, opts ...PublishOption) error

	// Subscribe registers a handler for messages from a topic or queue.
	Subscribe(ctx context.Context, topic string, handler MessageHandler, opts ...SubscribeOption) (Subscription, error)

	// BrokerType returns the type of this broker.
	BrokerType() BrokerType

	// GetMetrics returns current broker metrics.
	GetMetrics() *BrokerMetrics
}

// AcknowledgableBroker extends MessageBroker with acknowledgment capabilities.
type AcknowledgableBroker interface {
	MessageBroker

	// Ack acknowledges successful processing of a message.
	Ack(ctx context.Context, msg *Message) error

	// Nack indicates failed processing; requeue determines if message is redelivered.
	Nack(ctx context.Context, msg *Message, requeue bool) error

	// Reject permanently rejects a message (moves to dead letter if configured).
	Reject(ctx context.Context, msg *Message) error
}

// TransactionalBroker extends MessageBroker with transaction support.
type TransactionalBroker interface {
	MessageBroker

	// BeginTx starts a new transaction.
	BeginTx(ctx context.Context) (Transaction, error)
}

// Transaction represents a message broker transaction.
type Transaction interface {
	// Publish sends a message within the transaction.
	Publish(ctx context.Context, topic string, message *Message) error

	// Commit commits all messages in the transaction.
	Commit(ctx context.Context) error

	// Rollback aborts all messages in the transaction.
	Rollback(ctx context.Context) error
}

// BrokerConfig contains common configuration for message brokers.
type BrokerConfig struct {
	// Type identifies the broker type.
	Type BrokerType `json:"type" yaml:"type"`

	// Hosts is a list of broker hosts.
	Hosts []string `json:"hosts" yaml:"hosts"`

	// Port is the default port to connect to.
	Port int `json:"port" yaml:"port"`

	// Username for authentication.
	Username string `json:"username" yaml:"username"`

	// Password for authentication.
	Password string `json:"password" yaml:"password"`

	// TLS configuration.
	TLS *TLSConfig `json:"tls,omitempty" yaml:"tls,omitempty"`

	// ConnectionTimeout is the timeout for establishing connections.
	ConnectionTimeout time.Duration `json:"connection_timeout" yaml:"connection_timeout"`

	// RequestTimeout is the default timeout for operations.
	RequestTimeout time.Duration `json:"request_timeout" yaml:"request_timeout"`

	// MaxRetries is the maximum number of retry attempts for operations.
	MaxRetries int `json:"max_retries" yaml:"max_retries"`

	// RetryBackoff is the initial backoff duration between retries.
	RetryBackoff time.Duration `json:"retry_backoff" yaml:"retry_backoff"`

	// HeartbeatInterval is the interval between heartbeat messages.
	HeartbeatInterval time.Duration `json:"heartbeat_interval" yaml:"heartbeat_interval"`

	// MaxIdleConns is the maximum number of idle connections in the pool.
	MaxIdleConns int `json:"max_idle_conns" yaml:"max_idle_conns"`

	// MaxOpenConns is the maximum number of open connections.
	MaxOpenConns int `json:"max_open_conns" yaml:"max_open_conns"`
}

// TLSConfig contains TLS/SSL configuration.
type TLSConfig struct {
	// Enabled enables TLS connections.
	Enabled bool `json:"enabled" yaml:"enabled"`

	// CertFile is the path to the client certificate file.
	CertFile string `json:"cert_file,omitempty" yaml:"cert_file,omitempty"`

	// KeyFile is the path to the client private key file.
	KeyFile string `json:"key_file,omitempty" yaml:"key_file,omitempty"`

	// CAFile is the path to the CA certificate file.
	CAFile string `json:"ca_file,omitempty" yaml:"ca_file,omitempty"`

	// InsecureSkipVerify skips certificate verification (not recommended for production).
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty" yaml:"insecure_skip_verify,omitempty"`

	// ServerName is the expected server name for verification.
	ServerName string `json:"server_name,omitempty" yaml:"server_name,omitempty"`
}

// DefaultBrokerConfig returns a BrokerConfig with sensible defaults.
func DefaultBrokerConfig() *BrokerConfig {
	return &BrokerConfig{
		Type:              BrokerTypeInMemory,
		Hosts:             []string{"localhost"},
		ConnectionTimeout: 30 * time.Second,
		RequestTimeout:    60 * time.Second,
		MaxRetries:        3,
		RetryBackoff:      1 * time.Second,
		HeartbeatInterval: 30 * time.Second,
		MaxIdleConns:      10,
		MaxOpenConns:      100,
	}
}

// Validate checks if the configuration is valid.
func (c *BrokerConfig) Validate() error {
	if c == nil {
		return ErrNilConfig
	}
	if len(c.Hosts) == 0 {
		return ErrInvalidHost
	}
	if c.ConnectionTimeout <= 0 {
		return ErrInvalidTimeout
	}
	if c.RequestTimeout <= 0 {
		return ErrInvalidTimeout
	}
	return nil
}

// NewMessage creates a new Message with default values.
func NewMessage(messageType string, payload []byte) *Message {
	return &Message{
		ID:        generateMessageID(),
		Type:      messageType,
		Payload:   payload,
		Headers:   make(map[string]string),
		Timestamp: time.Now(),
		Priority:  PriorityNormal,
	}
}

// WithHeader adds a header to the message and returns the message for chaining.
func (m *Message) WithHeader(key, value string) *Message {
	if m.Headers == nil {
		m.Headers = make(map[string]string)
	}
	m.Headers[key] = value
	return m
}

// WithPriority sets the message priority and returns the message for chaining.
func (m *Message) WithPriority(priority int) *Message {
	m.Priority = priority
	return m
}

// WithTraceID sets the trace ID and returns the message for chaining.
func (m *Message) WithTraceID(traceID string) *Message {
	m.TraceID = traceID
	return m
}

// WithCorrelationID sets the correlation ID and returns the message for chaining.
func (m *Message) WithCorrelationID(correlationID string) *Message {
	m.CorrelationID = correlationID
	return m
}

// WithExpiration sets the message expiration and returns the message for chaining.
func (m *Message) WithExpiration(expiration time.Duration) *Message {
	m.Expiration = expiration
	return m
}

// WithKey sets the partition key (for Kafka) and returns the message for chaining.
func (m *Message) WithKey(key string) *Message {
	m.Key = key
	return m
}

// generateMessageID creates a unique message identifier.
func generateMessageID() string {
	return generateUUID()
}
