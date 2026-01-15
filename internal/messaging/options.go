package messaging

import (
	"time"
)

// PublishOptions contains options for publishing messages.
type PublishOptions struct {
	// Timeout for the publish operation.
	Timeout time.Duration

	// Persistent indicates if the message should survive broker restart.
	Persistent bool

	// Mandatory requires the message to be routed to at least one queue (RabbitMQ).
	Mandatory bool

	// Immediate requires immediate delivery to a consumer (RabbitMQ, deprecated).
	Immediate bool

	// WaitForConfirm waits for broker confirmation before returning.
	WaitForConfirm bool

	// ConfirmTimeout is the timeout for waiting for confirmation.
	ConfirmTimeout time.Duration

	// Exchange specifies the exchange to publish to (RabbitMQ).
	Exchange string

	// RoutingKey specifies the routing key for exchange routing (RabbitMQ).
	RoutingKey string

	// Partition specifies the target partition (Kafka).
	Partition *int32

	// CompressionType specifies message compression (Kafka).
	CompressionType string

	// Headers are additional headers to include.
	Headers map[string]string

	// DeduplicationID for idempotent publishing.
	DeduplicationID string
}

// PublishOption is a function that modifies PublishOptions.
type PublishOption func(*PublishOptions)

// DefaultPublishOptions returns default publish options.
func DefaultPublishOptions() *PublishOptions {
	return &PublishOptions{
		Timeout:        30 * time.Second,
		Persistent:     true,
		WaitForConfirm: true,
		ConfirmTimeout: 10 * time.Second,
		Headers:        make(map[string]string),
	}
}

// ApplyPublishOptions applies a list of options to PublishOptions.
func ApplyPublishOptions(opts ...PublishOption) *PublishOptions {
	options := DefaultPublishOptions()
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// WithPublishTimeout sets the publish timeout.
func WithPublishTimeout(timeout time.Duration) PublishOption {
	return func(o *PublishOptions) {
		o.Timeout = timeout
	}
}

// WithPersistent sets whether the message should be persistent.
func WithPersistent(persistent bool) PublishOption {
	return func(o *PublishOptions) {
		o.Persistent = persistent
	}
}

// WithMandatory sets the mandatory flag.
func WithMandatory(mandatory bool) PublishOption {
	return func(o *PublishOptions) {
		o.Mandatory = mandatory
	}
}

// WithWaitForConfirm sets whether to wait for broker confirmation.
func WithWaitForConfirm(wait bool) PublishOption {
	return func(o *PublishOptions) {
		o.WaitForConfirm = wait
	}
}

// WithConfirmTimeout sets the confirmation timeout.
func WithConfirmTimeout(timeout time.Duration) PublishOption {
	return func(o *PublishOptions) {
		o.ConfirmTimeout = timeout
	}
}

// WithPartition sets the target partition for Kafka.
func WithPartition(partition int32) PublishOption {
	return func(o *PublishOptions) {
		o.Partition = &partition
	}
}

// WithCompression sets the compression type.
func WithCompression(compressionType string) PublishOption {
	return func(o *PublishOptions) {
		o.CompressionType = compressionType
	}
}

// WithPublishHeaders adds headers to the publish options.
func WithPublishHeaders(headers map[string]string) PublishOption {
	return func(o *PublishOptions) {
		for k, v := range headers {
			o.Headers[k] = v
		}
	}
}

// WithDeduplicationID sets the deduplication ID.
func WithDeduplicationID(id string) PublishOption {
	return func(o *PublishOptions) {
		o.DeduplicationID = id
	}
}

// WithExchange sets the exchange to publish to (RabbitMQ).
func WithExchange(exchange string) PublishOption {
	return func(o *PublishOptions) {
		o.Exchange = exchange
	}
}

// WithRoutingKey sets the routing key (RabbitMQ).
func WithRoutingKey(key string) PublishOption {
	return func(o *PublishOptions) {
		o.RoutingKey = key
	}
}

// WithPersistence sets the persistence flag (alias for WithPersistent).
func WithPersistence(persistent bool) PublishOption {
	return func(o *PublishOptions) {
		o.Persistent = persistent
	}
}

// SubscribeOptions contains options for subscribing to topics/queues.
type SubscribeOptions struct {
	// ConsumerGroup is the consumer group name (Kafka).
	ConsumerGroup string

	// ConsumerTag is the consumer identifier (RabbitMQ).
	ConsumerTag string

	// AutoAck enables automatic acknowledgment.
	AutoAck bool

	// Exclusive makes this the only consumer for the queue (RabbitMQ).
	Exclusive bool

	// PrefetchCount limits the number of unacknowledged messages (RabbitMQ).
	PrefetchCount int

	// PrefetchSize limits the total size of unacknowledged messages in bytes.
	PrefetchSize int

	// StartOffset specifies where to start consuming (Kafka).
	// Possible values: "earliest", "latest", or a specific offset.
	StartOffset string

	// MaxConcurrency limits concurrent message processing.
	MaxConcurrency int

	// RequeueOnFailure requeues messages that fail processing.
	RequeueOnFailure bool

	// DeadLetterTopic specifies where to send failed messages.
	DeadLetterTopic string

	// MaxRetries limits delivery attempts before dead-lettering.
	MaxRetries int

	// RetryBackoff is the initial backoff between retries.
	RetryBackoff time.Duration

	// MaxRetryBackoff is the maximum backoff between retries.
	MaxRetryBackoff time.Duration

	// Headers are additional subscription headers.
	Headers map[string]string

	// FilterExpression for message filtering (if broker supports).
	FilterExpression string

	// PartitionAssignment specifies explicit partition assignment (Kafka).
	PartitionAssignment []int32

	// SessionTimeout for consumer group sessions (Kafka).
	SessionTimeout time.Duration

	// HeartbeatInterval for consumer group heartbeats (Kafka).
	HeartbeatInterval time.Duration

	// RebalanceTimeout for consumer group rebalancing (Kafka).
	RebalanceTimeout time.Duration
}

// SubscribeOption is a function that modifies SubscribeOptions.
type SubscribeOption func(*SubscribeOptions)

// DefaultSubscribeOptions returns default subscribe options.
func DefaultSubscribeOptions() *SubscribeOptions {
	return &SubscribeOptions{
		AutoAck:          false,
		Exclusive:        false,
		PrefetchCount:    10,
		StartOffset:      "latest",
		MaxConcurrency:   1,
		RequeueOnFailure: true,
		MaxRetries:       3,
		RetryBackoff:     1 * time.Second,
		MaxRetryBackoff:  30 * time.Second,
		SessionTimeout:   30 * time.Second,
		HeartbeatInterval: 10 * time.Second,
		RebalanceTimeout: 60 * time.Second,
		Headers:          make(map[string]string),
	}
}

// ApplySubscribeOptions applies a list of options to SubscribeOptions.
func ApplySubscribeOptions(opts ...SubscribeOption) *SubscribeOptions {
	options := DefaultSubscribeOptions()
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// WithConsumerGroup sets the consumer group.
func WithConsumerGroup(group string) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.ConsumerGroup = group
	}
}

// WithConsumerTag sets the consumer tag.
func WithConsumerTag(tag string) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.ConsumerTag = tag
	}
}

// WithAutoAck enables automatic acknowledgment.
func WithAutoAck(autoAck bool) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.AutoAck = autoAck
	}
}

// WithExclusive makes the consumer exclusive.
func WithExclusive(exclusive bool) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.Exclusive = exclusive
	}
}

// WithPrefetchCount sets the prefetch count.
func WithPrefetchCount(count int) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.PrefetchCount = count
	}
}

// WithStartOffset sets the start offset for Kafka.
func WithStartOffset(offset string) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.StartOffset = offset
	}
}

// WithMaxConcurrency sets the maximum concurrent handlers.
func WithMaxConcurrency(max int) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.MaxConcurrency = max
	}
}

// WithRequeueOnFailure sets whether to requeue on failure.
func WithRequeueOnFailure(requeue bool) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.RequeueOnFailure = requeue
	}
}

// WithDeadLetterTopic sets the dead letter topic.
func WithDeadLetterTopic(topic string) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.DeadLetterTopic = topic
	}
}

// WithMaxRetries sets the maximum number of retries.
func WithMaxRetries(max int) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.MaxRetries = max
	}
}

// WithRetryBackoff sets the retry backoff configuration.
func WithRetryBackoff(initial, max time.Duration) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.RetryBackoff = initial
		o.MaxRetryBackoff = max
	}
}

// WithSubscribeHeaders adds headers to the subscribe options.
func WithSubscribeHeaders(headers map[string]string) SubscribeOption {
	return func(o *SubscribeOptions) {
		for k, v := range headers {
			o.Headers[k] = v
		}
	}
}

// WithFilterExpression sets the message filter expression.
func WithFilterExpression(expr string) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.FilterExpression = expr
	}
}

// WithPartitions sets explicit partition assignment.
func WithPartitions(partitions ...int32) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.PartitionAssignment = partitions
	}
}

// WithSessionTimeout sets the session timeout.
func WithSessionTimeout(timeout time.Duration) SubscribeOption {
	return func(o *SubscribeOptions) {
		o.SessionTimeout = timeout
	}
}
