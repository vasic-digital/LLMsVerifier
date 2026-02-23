package messaging

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ensure require is used
var _ = require.NoError

// ============================================================================
// BrokerConfig.Validate Tests
// ============================================================================

func TestBrokerConfig_Validate_Valid(t *testing.T) {
	cfg := DefaultBrokerConfig()
	err := cfg.Validate()
	require.NoError(t, err)
}

func TestBrokerConfig_Validate_EmptyHosts(t *testing.T) {
	cfg := DefaultBrokerConfig()
	cfg.Hosts = []string{}
	err := cfg.Validate()
	assert.ErrorIs(t, err, ErrInvalidHost)
}

func TestBrokerConfig_Validate_ZeroConnectionTimeout(t *testing.T) {
	cfg := DefaultBrokerConfig()
	cfg.ConnectionTimeout = 0
	err := cfg.Validate()
	assert.ErrorIs(t, err, ErrInvalidTimeout)
}

func TestBrokerConfig_Validate_ZeroRequestTimeout(t *testing.T) {
	cfg := DefaultBrokerConfig()
	cfg.RequestTimeout = 0
	err := cfg.Validate()
	assert.ErrorIs(t, err, ErrInvalidTimeout)
}

// ============================================================================
// NewMessage and Message builder methods
// ============================================================================

func TestNewMessage(t *testing.T) {
	msg := NewMessage("test.event", []byte("payload"))

	assert.NotEmpty(t, msg.ID)
	assert.Equal(t, "test.event", msg.Type)
	assert.Equal(t, []byte("payload"), msg.Payload)
	assert.Equal(t, PriorityNormal, msg.Priority)
	assert.NotNil(t, msg.Headers)
	assert.False(t, msg.Timestamp.IsZero())
}

func TestMessage_WithHeader(t *testing.T) {
	msg := NewMessage("test", []byte{})
	result := msg.WithHeader("Content-Type", "application/json")

	assert.Same(t, msg, result)
	assert.Equal(t, "application/json", msg.Headers["Content-Type"])
}

func TestMessage_WithHeader_NilHeaders(t *testing.T) {
	msg := &Message{}
	msg.WithHeader("key", "value")
	assert.Equal(t, "value", msg.Headers["key"])
}

func TestMessage_WithPriority(t *testing.T) {
	msg := NewMessage("test", []byte{})
	result := msg.WithPriority(PriorityCritical)

	assert.Same(t, msg, result)
	assert.Equal(t, PriorityCritical, msg.Priority)
}

func TestMessage_WithTraceID(t *testing.T) {
	msg := NewMessage("test", []byte{})
	result := msg.WithTraceID("trace-123")

	assert.Same(t, msg, result)
	assert.Equal(t, "trace-123", msg.TraceID)
}

func TestMessage_WithCorrelationID(t *testing.T) {
	msg := NewMessage("test", []byte{})
	result := msg.WithCorrelationID("corr-456")

	assert.Same(t, msg, result)
	assert.Equal(t, "corr-456", msg.CorrelationID)
}

func TestMessage_WithExpiration(t *testing.T) {
	msg := NewMessage("test", []byte{})
	result := msg.WithExpiration(30 * time.Second)

	assert.Same(t, msg, result)
	assert.Equal(t, 30*time.Second, msg.Expiration)
}

func TestMessage_WithKey(t *testing.T) {
	msg := NewMessage("test", []byte{})
	result := msg.WithKey("partition-key")

	assert.Same(t, msg, result)
	assert.Equal(t, "partition-key", msg.Key)
}

func TestMessage_ChainBuilders(t *testing.T) {
	msg := NewMessage("test.event", []byte("data")).
		WithHeader("X-Source", "service-a").
		WithPriority(PriorityHigh).
		WithTraceID("trace-789").
		WithCorrelationID("corr-789").
		WithKey("my-key")

	assert.Equal(t, "service-a", msg.Headers["X-Source"])
	assert.Equal(t, PriorityHigh, msg.Priority)
	assert.Equal(t, "trace-789", msg.TraceID)
	assert.Equal(t, "corr-789", msg.CorrelationID)
	assert.Equal(t, "my-key", msg.Key)
}

// ============================================================================
// Error Types
// ============================================================================

func TestBrokerError_Error_WithTopic(t *testing.T) {
	err := &BrokerError{Op: "publish", Topic: "my-topic", Err: ErrPublishFailed}
	assert.Equal(t, "publish on my-topic: failed to publish message", err.Error())
}

func TestBrokerError_Error_NoTopic(t *testing.T) {
	err := &BrokerError{Op: "connect", Err: ErrConnectionFailed}
	assert.Equal(t, "connect: failed to connect to broker", err.Error())
}

func TestBrokerError_Unwrap(t *testing.T) {
	err := &BrokerError{Op: "subscribe", Err: ErrSubscribeFailed}
	assert.Equal(t, ErrSubscribeFailed, err.Unwrap())
}

func TestBrokerError_Is(t *testing.T) {
	err := &BrokerError{Op: "subscribe", Err: ErrSubscribeFailed}
	assert.True(t, err.Is(ErrSubscribeFailed))
	assert.False(t, err.Is(ErrPublishFailed))
}

func TestNewBrokerError(t *testing.T) {
	err := NewBrokerError("publish", "topic-1", ErrPublishFailed)
	assert.Equal(t, "publish", err.Op)
	assert.Equal(t, "topic-1", err.Topic)
	assert.Equal(t, ErrPublishFailed, err.Err)
}

func TestBrokerError_WithCode(t *testing.T) {
	err := NewBrokerError("op", "topic", ErrPublishFailed)
	result := err.WithCode("PUB_001")

	assert.Same(t, err, result)
	assert.Equal(t, "PUB_001", err.Code)
}

func TestBrokerError_WithDetail(t *testing.T) {
	err := NewBrokerError("op", "topic", ErrPublishFailed)
	err.WithDetail("retry_count", 3)

	assert.Equal(t, 3, err.Details["retry_count"])
}

func TestBrokerError_WithDetail_InitializesMap(t *testing.T) {
	err := &BrokerError{Op: "op", Err: ErrPublishFailed}
	assert.Nil(t, err.Details)

	err.WithDetail("key", "value")
	assert.NotNil(t, err.Details)
	assert.Equal(t, "value", err.Details["key"])
}

func TestRetryableError_Error(t *testing.T) {
	err := &RetryableError{Err: ErrConnectionLost, RetryAfter: 5}
	assert.Contains(t, err.Error(), "retryable error")
	assert.Contains(t, err.Error(), "5")
}

func TestRetryableError_Unwrap(t *testing.T) {
	err := &RetryableError{Err: ErrConnectionLost}
	assert.Equal(t, ErrConnectionLost, err.Unwrap())
}

func TestNewRetryableError(t *testing.T) {
	err := NewRetryableError(ErrTimeout, 10)
	assert.Equal(t, ErrTimeout, err.Err)
	assert.Equal(t, 10, err.RetryAfter)
}

func TestIsRetryable_RetryableError(t *testing.T) {
	err := NewRetryableError(errors.New("transient"), 5)
	assert.True(t, IsRetryable(err))
}

func TestIsRetryable_ConnectionLost(t *testing.T) {
	assert.True(t, IsRetryable(ErrConnectionLost))
}

func TestIsRetryable_Timeout(t *testing.T) {
	assert.True(t, IsRetryable(ErrTimeout))
}

func TestIsRetryable_RateLimited(t *testing.T) {
	assert.True(t, IsRetryable(ErrRateLimited))
}

func TestIsRetryable_NotRetryable(t *testing.T) {
	assert.False(t, IsRetryable(ErrAuthFailed))
}

func TestIsConnectionError_True(t *testing.T) {
	assert.True(t, IsConnectionError(ErrNotConnected))
	assert.True(t, IsConnectionError(ErrConnectionFailed))
	assert.True(t, IsConnectionError(ErrConnectionClosed))
	assert.True(t, IsConnectionError(ErrConnectionLost))
}

func TestIsConnectionError_False(t *testing.T) {
	assert.False(t, IsConnectionError(ErrAuthFailed))
	assert.False(t, IsConnectionError(ErrPublishFailed))
}

func TestIsConfigError_True(t *testing.T) {
	assert.True(t, IsConfigError(ErrNilConfig))
	assert.True(t, IsConfigError(ErrInvalidConfig))
	assert.True(t, IsConfigError(ErrInvalidHost))
	assert.True(t, IsConfigError(ErrInvalidPort))
	assert.True(t, IsConfigError(ErrInvalidTimeout))
}

func TestIsConfigError_False(t *testing.T) {
	assert.False(t, IsConfigError(ErrAuthFailed))
}

func TestIsAuthError_True(t *testing.T) {
	assert.True(t, IsAuthError(ErrAuthFailed))
	assert.True(t, IsAuthError(ErrAccessDenied))
	assert.True(t, IsAuthError(ErrInvalidToken))
	assert.True(t, IsAuthError(ErrTokenExpired))
}

func TestIsAuthError_False(t *testing.T) {
	assert.False(t, IsAuthError(ErrPublishFailed))
}

// ============================================================================
// Topic Config Builder
// ============================================================================

func TestDefaultTopicConfig(t *testing.T) {
	cfg := DefaultTopicConfig("my-topic")

	assert.Equal(t, "my-topic", cfg.Name)
	assert.Equal(t, int32(3), cfg.Partitions)
	assert.Equal(t, int16(1), cfg.ReplicationFactor)
	assert.Equal(t, 7*24*time.Hour, cfg.RetentionTime)
	assert.Equal(t, int64(-1), cfg.RetentionBytes)
	assert.Equal(t, "delete", cfg.CleanupPolicy)
	assert.Equal(t, "lz4", cfg.CompressionType)
	assert.Equal(t, 1, cfg.MinISR)
	assert.NotNil(t, cfg.CustomConfig)
}

func TestApplyTopicOptions_NoOptions(t *testing.T) {
	cfg := ApplyTopicOptions("topic-1")
	assert.Equal(t, "topic-1", cfg.Name)
	assert.Equal(t, int32(3), cfg.Partitions) // default
}

func TestWithTopicPartitions(t *testing.T) {
	cfg := ApplyTopicOptions("topic", WithTopicPartitions(6))
	assert.Equal(t, int32(6), cfg.Partitions)
}

func TestWithTopicReplication(t *testing.T) {
	cfg := ApplyTopicOptions("topic", WithTopicReplication(3))
	assert.Equal(t, int16(3), cfg.ReplicationFactor)
}

func TestWithTopicRetention(t *testing.T) {
	cfg := ApplyTopicOptions("topic", WithTopicRetention(24*time.Hour))
	assert.Equal(t, 24*time.Hour, cfg.RetentionTime)
}

func TestWithTopicRetentionBytes(t *testing.T) {
	cfg := ApplyTopicOptions("topic", WithTopicRetentionBytes(1073741824))
	assert.Equal(t, int64(1073741824), cfg.RetentionBytes)
}

func TestWithTopicCleanupPolicy(t *testing.T) {
	cfg := ApplyTopicOptions("topic", WithTopicCleanupPolicy("compact"))
	assert.Equal(t, "compact", cfg.CleanupPolicy)
}

func TestWithTopicCompression(t *testing.T) {
	cfg := ApplyTopicOptions("topic", WithTopicCompression("snappy"))
	assert.Equal(t, "snappy", cfg.CompressionType)
}

func TestWithTopicMinISR(t *testing.T) {
	cfg := ApplyTopicOptions("topic", WithTopicMinISR(2))
	assert.Equal(t, 2, cfg.MinISR)
}

func TestWithTopicConfig(t *testing.T) {
	cfg := ApplyTopicOptions("topic", WithTopicConfig("custom.key", "custom.value"))
	assert.Equal(t, "custom.value", cfg.CustomConfig["custom.key"])
}

func TestWithTopicConfig_NilCustomConfig(t *testing.T) {
	base := DefaultTopicConfig("t")
	base.CustomConfig = nil
	opt := WithTopicConfig("k", "v")
	opt(base)
	assert.Equal(t, "v", base.CustomConfig["k"])
}

func TestApplyTopicOptions_MultipleOptions(t *testing.T) {
	cfg := ApplyTopicOptions("topic",
		WithTopicPartitions(12),
		WithTopicReplication(3),
		WithTopicCompression("gzip"),
		WithTopicCleanupPolicy("compact"),
	)
	assert.Equal(t, int32(12), cfg.Partitions)
	assert.Equal(t, int16(3), cfg.ReplicationFactor)
	assert.Equal(t, "gzip", cfg.CompressionType)
	assert.Equal(t, "compact", cfg.CleanupPolicy)
}

// ============================================================================
// Stream Options Builder
// ============================================================================

func TestDefaultStreamOptions(t *testing.T) {
	opts := DefaultStreamOptions()

	assert.Equal(t, "latest", opts.StartOffset)
	assert.Equal(t, 1000, opts.BufferSize)
	assert.Equal(t, 500, opts.MaxPollRecords)
	assert.Equal(t, 1*time.Second, opts.PollTimeout)
	assert.False(t, opts.AutoCommit)
	assert.Equal(t, "read_committed", opts.IsolationLevel)
}

func TestApplyStreamOptions_NoOptions(t *testing.T) {
	opts := ApplyStreamOptions()
	assert.Equal(t, "latest", opts.StartOffset) // default
}

func TestWithStreamConsumerGroup(t *testing.T) {
	opts := ApplyStreamOptions(WithStreamConsumerGroup("my-group"))
	assert.Equal(t, "my-group", opts.ConsumerGroup)
}

func TestWithStreamStartOffset(t *testing.T) {
	opts := ApplyStreamOptions(WithStreamStartOffset("earliest"))
	assert.Equal(t, "earliest", opts.StartOffset)
}

func TestWithStreamStartTimestamp(t *testing.T) {
	ts := time.Now()
	opts := ApplyStreamOptions(WithStreamStartTimestamp(ts))
	assert.Equal(t, ts, opts.StartTimestamp)
}

func TestWithStreamPartitions(t *testing.T) {
	opts := ApplyStreamOptions(WithStreamPartitions(0, 1, 2))
	assert.Equal(t, []int32{0, 1, 2}, opts.Partitions)
}

func TestWithStreamBufferSize(t *testing.T) {
	opts := ApplyStreamOptions(WithStreamBufferSize(500))
	assert.Equal(t, 500, opts.BufferSize)
}

func TestApplyStreamOptions_Multiple(t *testing.T) {
	opts := ApplyStreamOptions(
		WithStreamConsumerGroup("grp"),
		WithStreamStartOffset("earliest"),
		WithStreamPartitions(0, 1),
		WithStreamBufferSize(200),
	)
	assert.Equal(t, "grp", opts.ConsumerGroup)
	assert.Equal(t, "earliest", opts.StartOffset)
	assert.Len(t, opts.Partitions, 2)
	assert.Equal(t, 200, opts.BufferSize)
}

// ============================================================================
// DefaultBrokerConfig
// ============================================================================

func TestDefaultBrokerConfig(t *testing.T) {
	cfg := DefaultBrokerConfig()

	assert.Equal(t, BrokerTypeInMemory, cfg.Type)
	assert.Equal(t, []string{"localhost"}, cfg.Hosts)
	assert.Equal(t, 30*time.Second, cfg.ConnectionTimeout)
	assert.Equal(t, 60*time.Second, cfg.RequestTimeout)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, 10, cfg.MaxIdleConns)
	assert.Equal(t, 100, cfg.MaxOpenConns)
}

// ============================================================================
// generateMessageID (indirect: via NewMessage)
// ============================================================================

func TestNewMessage_UniqueIDs(t *testing.T) {
	msg1 := NewMessage("type", []byte{})
	msg2 := NewMessage("type", []byte{})

	assert.NotEmpty(t, msg1.ID)
	assert.NotEmpty(t, msg2.ID)
	assert.NotEqual(t, msg1.ID, msg2.ID)
}

// ============================================================================
// WithStreamAutoCommit, WithStreamFilterTypes
// ============================================================================

func TestWithStreamAutoCommit(t *testing.T) {
	opts := ApplyStreamOptions(WithStreamAutoCommit(true, 5*time.Second))
	assert.True(t, opts.AutoCommit)
	assert.Equal(t, 5*time.Second, opts.AutoCommitInterval)
}

func TestWithStreamAutoCommit_Disabled(t *testing.T) {
	opts := ApplyStreamOptions(WithStreamAutoCommit(false, 0))
	assert.False(t, opts.AutoCommit)
}

func TestWithStreamFilterTypes(t *testing.T) {
	opts := ApplyStreamOptions(WithStreamFilterTypes("order.created", "order.updated"))
	assert.Equal(t, []string{"order.created", "order.updated"}, opts.FilterTypes)
}

// ============================================================================
// NewEvent and Event builder methods
// ============================================================================

func TestNewEvent(t *testing.T) {
	evt := NewEvent("user.created", "auth-service", []byte(`{"id":"1"}`))

	assert.NotEmpty(t, evt.ID)
	assert.Equal(t, "user.created", evt.Type)
	assert.Equal(t, "auth-service", evt.Source)
	assert.Equal(t, []byte(`{"id":"1"}`), evt.Data)
	assert.Equal(t, "application/json", evt.DataContentType)
	assert.Equal(t, "1.0", evt.Version)
	assert.NotNil(t, evt.Headers)
	assert.False(t, evt.Timestamp.IsZero())
}

func TestEvent_WithEventSubject(t *testing.T) {
	evt := NewEvent("test", "src", []byte{})
	result := evt.WithEventSubject("users/123")
	assert.Same(t, evt, result)
	assert.Equal(t, "users/123", evt.Subject)
}

func TestEvent_WithEventKey(t *testing.T) {
	evt := NewEvent("test", "src", []byte{})
	result := evt.WithEventKey("partition-key-1")
	assert.Same(t, evt, result)
	assert.Equal(t, "partition-key-1", evt.Key)
}

func TestEvent_WithEventTraceID(t *testing.T) {
	evt := NewEvent("test", "src", []byte{})
	result := evt.WithEventTraceID("trace-abc")
	assert.Same(t, evt, result)
	assert.Equal(t, "trace-abc", evt.TraceID)
}

func TestEvent_WithEventCorrelationID(t *testing.T) {
	evt := NewEvent("test", "src", []byte{})
	result := evt.WithEventCorrelationID("corr-xyz")
	assert.Same(t, evt, result)
	assert.Equal(t, "corr-xyz", evt.CorrelationID)
}

func TestEvent_WithEventCausationID(t *testing.T) {
	evt := NewEvent("test", "src", []byte{})
	result := evt.WithEventCausationID("cause-001")
	assert.Same(t, evt, result)
	assert.Equal(t, "cause-001", evt.CausationID)
}

func TestEvent_WithEventHeader(t *testing.T) {
	evt := NewEvent("test", "src", []byte{})
	result := evt.WithEventHeader("X-Tenant", "tenant-1")
	assert.Same(t, evt, result)
	assert.Equal(t, "tenant-1", evt.Headers["X-Tenant"])
}

func TestEvent_WithEventHeader_NilHeaders(t *testing.T) {
	evt := &Event{}
	evt.WithEventHeader("key", "value")
	assert.Equal(t, "value", evt.Headers["key"])
}

func TestEvent_WithEventSchema(t *testing.T) {
	evt := NewEvent("test", "src", []byte{})
	result := evt.WithEventSchema("https://schema.example.com/v1/user")
	assert.Same(t, evt, result)
	assert.Equal(t, "https://schema.example.com/v1/user", evt.DataSchema)
}

func TestEvent_ChainBuilders(t *testing.T) {
	evt := NewEvent("order.placed", "order-service", []byte(`{}`)).
		WithEventSubject("orders/456").
		WithEventKey("customer-42").
		WithEventTraceID("trace-1").
		WithEventCorrelationID("corr-1").
		WithEventCausationID("cause-1").
		WithEventHeader("X-Region", "us-west").
		WithEventSchema("https://schema.example.com/order")

	assert.Equal(t, "orders/456", evt.Subject)
	assert.Equal(t, "customer-42", evt.Key)
	assert.Equal(t, "trace-1", evt.TraceID)
	assert.Equal(t, "corr-1", evt.CorrelationID)
	assert.Equal(t, "cause-1", evt.CausationID)
	assert.Equal(t, "us-west", evt.Headers["X-Region"])
	assert.Equal(t, "https://schema.example.com/order", evt.DataSchema)
}

// ============================================================================
// PublishOptions builder functions
// ============================================================================

func TestDefaultPublishOptions(t *testing.T) {
	opts := DefaultPublishOptions()
	assert.Equal(t, 30*time.Second, opts.Timeout)
	assert.True(t, opts.Persistent)
	assert.True(t, opts.WaitForConfirm)
	assert.Equal(t, 10*time.Second, opts.ConfirmTimeout)
	assert.NotNil(t, opts.Headers)
}

func TestApplyPublishOptions_NoOptions(t *testing.T) {
	opts := ApplyPublishOptions()
	assert.Equal(t, 30*time.Second, opts.Timeout)
}

func TestWithPublishTimeout(t *testing.T) {
	opts := ApplyPublishOptions(WithPublishTimeout(5 * time.Second))
	assert.Equal(t, 5*time.Second, opts.Timeout)
}

func TestWithPersistent(t *testing.T) {
	opts := ApplyPublishOptions(WithPersistent(false))
	assert.False(t, opts.Persistent)
}

func TestWithPersistence(t *testing.T) {
	opts := ApplyPublishOptions(WithPersistence(true))
	assert.True(t, opts.Persistent)
}

func TestWithMandatory(t *testing.T) {
	opts := ApplyPublishOptions(WithMandatory(true))
	assert.True(t, opts.Mandatory)
}

func TestWithWaitForConfirm(t *testing.T) {
	opts := ApplyPublishOptions(WithWaitForConfirm(false))
	assert.False(t, opts.WaitForConfirm)
}

func TestWithConfirmTimeout(t *testing.T) {
	opts := ApplyPublishOptions(WithConfirmTimeout(5 * time.Second))
	assert.Equal(t, 5*time.Second, opts.ConfirmTimeout)
}

func TestWithPartition(t *testing.T) {
	opts := ApplyPublishOptions(WithPartition(3))
	assert.NotNil(t, opts.Partition)
	assert.Equal(t, int32(3), *opts.Partition)
}

func TestWithCompression(t *testing.T) {
	opts := ApplyPublishOptions(WithCompression("snappy"))
	assert.Equal(t, "snappy", opts.CompressionType)
}

func TestWithPublishHeaders(t *testing.T) {
	opts := ApplyPublishOptions(WithPublishHeaders(map[string]string{"X-App": "test"}))
	assert.Equal(t, "test", opts.Headers["X-App"])
}

func TestWithDeduplicationID(t *testing.T) {
	opts := ApplyPublishOptions(WithDeduplicationID("dedup-123"))
	assert.Equal(t, "dedup-123", opts.DeduplicationID)
}

func TestWithExchange(t *testing.T) {
	opts := ApplyPublishOptions(WithExchange("my-exchange"))
	assert.Equal(t, "my-exchange", opts.Exchange)
}

func TestWithRoutingKey(t *testing.T) {
	opts := ApplyPublishOptions(WithRoutingKey("my.routing.key"))
	assert.Equal(t, "my.routing.key", opts.RoutingKey)
}

// ============================================================================
// SubscribeOptions builder functions
// ============================================================================

func TestDefaultSubscribeOptions(t *testing.T) {
	opts := DefaultSubscribeOptions()
	assert.False(t, opts.AutoAck)
	assert.Equal(t, 10, opts.PrefetchCount)
	assert.Equal(t, "latest", opts.StartOffset)
	assert.Equal(t, 1, opts.MaxConcurrency)
	assert.True(t, opts.RequeueOnFailure)
	assert.Equal(t, 3, opts.MaxRetries)
}

func TestApplySubscribeOptions_NoOptions(t *testing.T) {
	opts := ApplySubscribeOptions()
	assert.Equal(t, "latest", opts.StartOffset)
}

func TestWithConsumerGroup_Subscribe(t *testing.T) {
	opts := ApplySubscribeOptions(WithConsumerGroup("my-group"))
	assert.Equal(t, "my-group", opts.ConsumerGroup)
}

func TestWithConsumerTag(t *testing.T) {
	opts := ApplySubscribeOptions(WithConsumerTag("consumer-1"))
	assert.Equal(t, "consumer-1", opts.ConsumerTag)
}

func TestWithAutoAck(t *testing.T) {
	opts := ApplySubscribeOptions(WithAutoAck(true))
	assert.True(t, opts.AutoAck)
}

func TestWithExclusive_Subscribe(t *testing.T) {
	opts := ApplySubscribeOptions(WithExclusive(true))
	assert.True(t, opts.Exclusive)
}

func TestWithPrefetchCount(t *testing.T) {
	opts := ApplySubscribeOptions(WithPrefetchCount(50))
	assert.Equal(t, 50, opts.PrefetchCount)
}

func TestWithStartOffset(t *testing.T) {
	opts := ApplySubscribeOptions(WithStartOffset("earliest"))
	assert.Equal(t, "earliest", opts.StartOffset)
}

func TestWithMaxConcurrency(t *testing.T) {
	opts := ApplySubscribeOptions(WithMaxConcurrency(5))
	assert.Equal(t, 5, opts.MaxConcurrency)
}

func TestWithRequeueOnFailure(t *testing.T) {
	opts := ApplySubscribeOptions(WithRequeueOnFailure(false))
	assert.False(t, opts.RequeueOnFailure)
}

func TestWithDeadLetterTopic(t *testing.T) {
	opts := ApplySubscribeOptions(WithDeadLetterTopic("dead-letter"))
	assert.Equal(t, "dead-letter", opts.DeadLetterTopic)
}

func TestWithMaxRetries_Subscribe(t *testing.T) {
	opts := ApplySubscribeOptions(WithMaxRetries(5))
	assert.Equal(t, 5, opts.MaxRetries)
}

func TestWithRetryBackoff(t *testing.T) {
	opts := ApplySubscribeOptions(WithRetryBackoff(2*time.Second, 60*time.Second))
	assert.Equal(t, 2*time.Second, opts.RetryBackoff)
	assert.Equal(t, 60*time.Second, opts.MaxRetryBackoff)
}

func TestWithSubscribeHeaders(t *testing.T) {
	opts := ApplySubscribeOptions(WithSubscribeHeaders(map[string]string{"X-Env": "prod"}))
	assert.Equal(t, "prod", opts.Headers["X-Env"])
}

func TestWithFilterExpression(t *testing.T) {
	opts := ApplySubscribeOptions(WithFilterExpression("type = 'order.created'"))
	assert.Equal(t, "type = 'order.created'", opts.FilterExpression)
}

func TestWithPartitions_Subscribe(t *testing.T) {
	opts := ApplySubscribeOptions(WithPartitions(0, 1, 2))
	assert.Equal(t, []int32{0, 1, 2}, opts.PartitionAssignment)
}

// ============================================================================
// QueueConfig builder functions
// ============================================================================

func TestDefaultQueueConfig(t *testing.T) {
	cfg := DefaultQueueConfig("my-queue")
	assert.Equal(t, "my-queue", cfg.Name)
	assert.True(t, cfg.Durable)
	assert.False(t, cfg.Exclusive)
	assert.False(t, cfg.AutoDelete)
	assert.Equal(t, 24*time.Hour, cfg.MessageTTL)
	assert.Equal(t, int64(1000000), cfg.MaxLength)
	assert.NotNil(t, cfg.Arguments)
}

func TestApplyQueueOptions_NoOptions(t *testing.T) {
	cfg := ApplyQueueOptions("q")
	assert.Equal(t, "q", cfg.Name)
}

func TestWithDurable(t *testing.T) {
	cfg := ApplyQueueOptions("q", WithDurable(false))
	assert.False(t, cfg.Durable)
}

func TestWithQueueExclusive(t *testing.T) {
	cfg := ApplyQueueOptions("q", WithQueueExclusive(true))
	assert.True(t, cfg.Exclusive)
}

func TestWithAutoDelete(t *testing.T) {
	cfg := ApplyQueueOptions("q", WithAutoDelete(true))
	assert.True(t, cfg.AutoDelete)
}

func TestWithMessageTTL(t *testing.T) {
	cfg := ApplyQueueOptions("q", WithMessageTTL(1*time.Hour))
	assert.Equal(t, 1*time.Hour, cfg.MessageTTL)
}

func TestWithMaxLength(t *testing.T) {
	cfg := ApplyQueueOptions("q", WithMaxLength(5000))
	assert.Equal(t, int64(5000), cfg.MaxLength)
}

func TestWithMaxLengthBytes(t *testing.T) {
	cfg := ApplyQueueOptions("q", WithMaxLengthBytes(1<<20))
	assert.Equal(t, int64(1<<20), cfg.MaxLengthBytes)
}

func TestWithDeadLetterQueue(t *testing.T) {
	cfg := ApplyQueueOptions("q", WithDeadLetterQueue("dlx", "dlq"))
	assert.Equal(t, "dlx", cfg.DeadLetterExchange)
	assert.Equal(t, "dlq", cfg.DeadLetterRoutingKey)
}

func TestWithQueueMaxPriority(t *testing.T) {
	cfg := ApplyQueueOptions("q", WithQueueMaxPriority(255))
	assert.Equal(t, 255, cfg.MaxPriority)
}

func TestWithQueueArgument(t *testing.T) {
	cfg := ApplyQueueOptions("q", WithQueueArgument("x-custom", "value"))
	assert.Equal(t, "value", cfg.Arguments["x-custom"])
}

func TestWithQueueArgument_NilArguments(t *testing.T) {
	base := DefaultQueueConfig("q")
	base.Arguments = nil
	opt := WithQueueArgument("key", 42)
	opt(base)
	assert.Equal(t, 42, base.Arguments["key"])
}

// ============================================================================
// Task builder functions
// ============================================================================

func TestNewTask(t *testing.T) {
	task := NewTask("email.send", []byte(`{"to":"user@example.com"}`))
	assert.NotEmpty(t, task.ID)
	assert.Equal(t, "email.send", task.Type)
	assert.Equal(t, TaskStatusPending, task.Status)
	assert.Equal(t, PriorityNormal, task.Priority)
	assert.Equal(t, 3, task.MaxRetries)
	assert.NotNil(t, task.Metadata)
}

func TestTask_WithTaskPriority(t *testing.T) {
	task := NewTask("t", nil)
	result := task.WithTaskPriority(PriorityCritical)
	assert.Same(t, task, result)
	assert.Equal(t, PriorityCritical, task.Priority)
}

func TestTask_WithTaskMaxRetries(t *testing.T) {
	task := NewTask("t", nil)
	result := task.WithTaskMaxRetries(5)
	assert.Same(t, task, result)
	assert.Equal(t, 5, task.MaxRetries)
}

func TestTask_WithTaskDeadline(t *testing.T) {
	deadline := time.Now().Add(1 * time.Hour)
	task := NewTask("t", nil)
	result := task.WithTaskDeadline(deadline)
	assert.Same(t, task, result)
	assert.Equal(t, deadline, task.Deadline)
}

func TestTask_WithTaskTimeout(t *testing.T) {
	task := NewTask("t", nil)
	result := task.WithTaskTimeout(30 * time.Second)
	assert.Same(t, task, result)
	assert.Equal(t, 30*time.Second, task.Timeout)
}

func TestTask_WithTaskSchedule(t *testing.T) {
	future := time.Now().Add(1 * time.Hour)
	task := NewTask("t", nil)
	result := task.WithTaskSchedule(future)
	assert.Same(t, task, result)
	assert.Equal(t, TaskStatusScheduled, task.Status)
	assert.Equal(t, future, task.ScheduledAt)
}

func TestTask_WithTaskMetadata(t *testing.T) {
	task := NewTask("t", nil)
	result := task.WithTaskMetadata("env", "prod")
	assert.Same(t, task, result)
	assert.Equal(t, "prod", task.Metadata["env"])
}

func TestTask_WithTaskMetadata_NilMap(t *testing.T) {
	task := &Task{}
	task.WithTaskMetadata("k", "v")
	assert.Equal(t, "v", task.Metadata["k"])
}

func TestTask_WithTaskTraceID(t *testing.T) {
	task := NewTask("t", nil)
	result := task.WithTaskTraceID("trace-xyz")
	assert.Same(t, task, result)
	assert.Equal(t, "trace-xyz", task.TraceID)
}

func TestTask_WithTaskCorrelationID(t *testing.T) {
	task := NewTask("t", nil)
	result := task.WithTaskCorrelationID("corr-abc")
	assert.Same(t, task, result)
	assert.Equal(t, "corr-abc", task.CorrelationID)
}

func TestTask_IsExpired_NoDeadline(t *testing.T) {
	task := NewTask("t", nil)
	assert.False(t, task.IsExpired())
}

func TestTask_IsExpired_PastDeadline(t *testing.T) {
	task := NewTask("t", nil)
	task.Deadline = time.Now().Add(-1 * time.Hour)
	assert.True(t, task.IsExpired())
}

func TestTask_IsExpired_FutureDeadline(t *testing.T) {
	task := NewTask("t", nil)
	task.Deadline = time.Now().Add(1 * time.Hour)
	assert.False(t, task.IsExpired())
}

func TestTask_CanRetry_True(t *testing.T) {
	task := NewTask("t", nil)
	task.RetryCount = 1
	task.MaxRetries = 3
	assert.True(t, task.CanRetry())
}

func TestTask_CanRetry_False(t *testing.T) {
	task := NewTask("t", nil)
	task.RetryCount = 3
	task.MaxRetries = 3
	assert.False(t, task.CanRetry())
}

func TestTask_ShouldExecute_NoSchedule(t *testing.T) {
	task := NewTask("t", nil)
	assert.True(t, task.ShouldExecute())
}

func TestTask_ShouldExecute_PastSchedule(t *testing.T) {
	task := NewTask("t", nil)
	task.ScheduledAt = time.Now().Add(-1 * time.Hour)
	assert.True(t, task.ShouldExecute())
}

func TestTask_ShouldExecute_FutureSchedule(t *testing.T) {
	task := NewTask("t", nil)
	task.ScheduledAt = time.Now().Add(1 * time.Hour)
	assert.False(t, task.ShouldExecute())
}
