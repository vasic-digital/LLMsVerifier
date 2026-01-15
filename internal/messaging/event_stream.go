package messaging

import (
	"context"
	"time"
)

// Event represents an event to be published or consumed from a stream.
type Event struct {
	// ID is a unique identifier for the event.
	ID string `json:"id"`

	// Type identifies the event type (e.g., "user.created", "order.placed").
	Type string `json:"type"`

	// Source identifies the origin of the event.
	Source string `json:"source"`

	// Subject is the subject/entity the event relates to.
	Subject string `json:"subject,omitempty"`

	// Data contains the event payload.
	Data []byte `json:"data"`

	// DataContentType describes the data format (e.g., "application/json").
	DataContentType string `json:"data_content_type,omitempty"`

	// DataSchema is a URI reference to the schema for the data.
	DataSchema string `json:"data_schema,omitempty"`

	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp"`

	// Headers contains event metadata.
	Headers map[string]string `json:"headers,omitempty"`

	// Key is the partition key (determines partition assignment).
	Key string `json:"key,omitempty"`

	// Partition is the partition the event was read from/written to.
	Partition int32 `json:"partition,omitempty"`

	// Offset is the offset within the partition.
	Offset int64 `json:"offset,omitempty"`

	// TraceID for distributed tracing correlation.
	TraceID string `json:"trace_id,omitempty"`

	// CorrelationID for correlating related events.
	CorrelationID string `json:"correlation_id,omitempty"`

	// CausationID identifies the event that caused this event.
	CausationID string `json:"causation_id,omitempty"`

	// Version for event schema versioning.
	Version string `json:"version,omitempty"`
}

// EventHandler is a function that processes an event.
type EventHandler func(ctx context.Context, event *Event) error

// TopicMetadata contains metadata about a topic.
type TopicMetadata struct {
	Name            string           `json:"name"`
	Partitions      int32            `json:"partitions"`
	ReplicationFactor int16          `json:"replication_factor"`
	PartitionInfo   []PartitionInfo  `json:"partition_info,omitempty"`
	Config          map[string]string `json:"config,omitempty"`
}

// PartitionInfo contains information about a partition.
type PartitionInfo struct {
	Partition     int32   `json:"partition"`
	Leader        int32   `json:"leader"`
	Replicas      []int32 `json:"replicas"`
	ISR           []int32 `json:"isr"` // In-Sync Replicas
	BeginOffset   int64   `json:"begin_offset"`
	EndOffset     int64   `json:"end_offset"`
}

// ConsumerGroupInfo contains information about a consumer group.
type ConsumerGroupInfo struct {
	GroupID         string                   `json:"group_id"`
	State           string                   `json:"state"`
	Protocol        string                   `json:"protocol"`
	ProtocolType    string                   `json:"protocol_type"`
	Members         []ConsumerGroupMember    `json:"members,omitempty"`
	PartitionLag    map[string]int64         `json:"partition_lag,omitempty"`
}

// ConsumerGroupMember contains information about a consumer group member.
type ConsumerGroupMember struct {
	MemberID    string   `json:"member_id"`
	ClientID    string   `json:"client_id"`
	ClientHost  string   `json:"client_host"`
	Partitions  []int32  `json:"partitions"`
}

// EventStreamBroker extends MessageBroker with event streaming specific operations.
type EventStreamBroker interface {
	MessageBroker

	// Topic management
	CreateTopic(ctx context.Context, name string, partitions int32, replication int16, opts ...TopicOption) error
	DeleteTopic(ctx context.Context, name string) error
	ListTopics(ctx context.Context) ([]string, error)
	GetTopicMetadata(ctx context.Context, name string) (*TopicMetadata, error)
	UpdateTopicConfig(ctx context.Context, name string, config map[string]string) error

	// Consumer group management
	CreateConsumerGroup(ctx context.Context, groupID string) error
	DeleteConsumerGroup(ctx context.Context, groupID string) error
	GetConsumerGroupInfo(ctx context.Context, groupID string) (*ConsumerGroupInfo, error)
	ListConsumerGroups(ctx context.Context) ([]string, error)

	// Offset management
	CommitOffset(ctx context.Context, topic string, partition int32, offset int64) error
	GetCommittedOffset(ctx context.Context, topic string, partition int32) (int64, error)
	SeekToOffset(ctx context.Context, topic string, partition int32, offset int64) error
	SeekToTimestamp(ctx context.Context, topic string, partition int32, ts time.Time) error
	SeekToBeginning(ctx context.Context, topic string, partition int32) error
	SeekToEnd(ctx context.Context, topic string, partition int32) error

	// Streaming
	StreamEvents(ctx context.Context, topic string, opts ...StreamOption) (<-chan *Event, error)
	PublishEvent(ctx context.Context, topic string, event *Event) error
	PublishEvents(ctx context.Context, topic string, events []*Event) error

	// Exactly-once semantics
	BeginTransaction(ctx context.Context) error
	CommitTransaction(ctx context.Context) error
	AbortTransaction(ctx context.Context) error
}

// TopicConfig contains configuration for a topic.
type TopicConfig struct {
	// Name is the topic name.
	Name string `json:"name" yaml:"name"`

	// Partitions is the number of partitions.
	Partitions int32 `json:"partitions" yaml:"partitions"`

	// ReplicationFactor is the number of replicas.
	ReplicationFactor int16 `json:"replication_factor" yaml:"replication_factor"`

	// RetentionTime is how long to keep messages.
	RetentionTime time.Duration `json:"retention_time" yaml:"retention_time"`

	// RetentionBytes is the maximum size in bytes.
	RetentionBytes int64 `json:"retention_bytes" yaml:"retention_bytes"`

	// SegmentBytes is the log segment size.
	SegmentBytes int64 `json:"segment_bytes" yaml:"segment_bytes"`

	// CleanupPolicy is "delete" or "compact".
	CleanupPolicy string `json:"cleanup_policy" yaml:"cleanup_policy"`

	// CompressionType is the compression codec.
	CompressionType string `json:"compression_type" yaml:"compression_type"`

	// MinISR is the minimum in-sync replicas.
	MinISR int `json:"min_isr" yaml:"min_isr"`

	// MaxMessageBytes is the maximum message size.
	MaxMessageBytes int `json:"max_message_bytes" yaml:"max_message_bytes"`

	// CustomConfig contains additional topic configuration.
	CustomConfig map[string]string `json:"custom_config,omitempty" yaml:"custom_config,omitempty"`
}

// TopicOption is a function that modifies TopicConfig.
type TopicOption func(*TopicConfig)

// DefaultTopicConfig returns a TopicConfig with sensible defaults.
func DefaultTopicConfig(name string) *TopicConfig {
	return &TopicConfig{
		Name:              name,
		Partitions:        3,
		ReplicationFactor: 1,
		RetentionTime:     7 * 24 * time.Hour,
		RetentionBytes:    -1, // unlimited
		SegmentBytes:      1073741824, // 1GB
		CleanupPolicy:     "delete",
		CompressionType:   "lz4",
		MinISR:            1,
		MaxMessageBytes:   1048576, // 1MB
		CustomConfig:      make(map[string]string),
	}
}

// ApplyTopicOptions applies a list of options to TopicConfig.
func ApplyTopicOptions(name string, opts ...TopicOption) *TopicConfig {
	config := DefaultTopicConfig(name)
	for _, opt := range opts {
		opt(config)
	}
	return config
}

// WithTopicPartitions sets the number of partitions.
func WithTopicPartitions(partitions int32) TopicOption {
	return func(c *TopicConfig) {
		c.Partitions = partitions
	}
}

// WithTopicReplication sets the replication factor.
func WithTopicReplication(factor int16) TopicOption {
	return func(c *TopicConfig) {
		c.ReplicationFactor = factor
	}
}

// WithTopicRetention sets the retention time.
func WithTopicRetention(duration time.Duration) TopicOption {
	return func(c *TopicConfig) {
		c.RetentionTime = duration
	}
}

// WithTopicRetentionBytes sets the retention size.
func WithTopicRetentionBytes(bytes int64) TopicOption {
	return func(c *TopicConfig) {
		c.RetentionBytes = bytes
	}
}

// WithTopicCleanupPolicy sets the cleanup policy.
func WithTopicCleanupPolicy(policy string) TopicOption {
	return func(c *TopicConfig) {
		c.CleanupPolicy = policy
	}
}

// WithTopicCompression sets the compression type.
func WithTopicCompression(compression string) TopicOption {
	return func(c *TopicConfig) {
		c.CompressionType = compression
	}
}

// WithTopicMinISR sets the minimum in-sync replicas.
func WithTopicMinISR(minISR int) TopicOption {
	return func(c *TopicConfig) {
		c.MinISR = minISR
	}
}

// WithTopicConfig adds a custom configuration option.
func WithTopicConfig(key, value string) TopicOption {
	return func(c *TopicConfig) {
		if c.CustomConfig == nil {
			c.CustomConfig = make(map[string]string)
		}
		c.CustomConfig[key] = value
	}
}

// StreamOptions contains options for streaming events.
type StreamOptions struct {
	// ConsumerGroup for group consumption.
	ConsumerGroup string

	// StartOffset specifies where to start: "earliest", "latest", or offset.
	StartOffset string

	// StartTimestamp specifies a timestamp to start from.
	StartTimestamp time.Time

	// Partitions specifies which partitions to consume.
	Partitions []int32

	// BufferSize is the channel buffer size.
	BufferSize int

	// MaxPollRecords limits records per poll.
	MaxPollRecords int

	// PollTimeout is the timeout for each poll.
	PollTimeout time.Duration

	// AutoCommit enables automatic offset commits.
	AutoCommit bool

	// AutoCommitInterval is the interval for auto-commits.
	AutoCommitInterval time.Duration

	// IsolationLevel for transactional reads.
	IsolationLevel string

	// FilterTypes filters events by type.
	FilterTypes []string
}

// StreamOption is a function that modifies StreamOptions.
type StreamOption func(*StreamOptions)

// DefaultStreamOptions returns default streaming options.
func DefaultStreamOptions() *StreamOptions {
	return &StreamOptions{
		StartOffset:         "latest",
		BufferSize:          1000,
		MaxPollRecords:      500,
		PollTimeout:         1 * time.Second,
		AutoCommit:          false,
		AutoCommitInterval:  5 * time.Second,
		IsolationLevel:      "read_committed",
	}
}

// ApplyStreamOptions applies a list of options to StreamOptions.
func ApplyStreamOptions(opts ...StreamOption) *StreamOptions {
	options := DefaultStreamOptions()
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// WithStreamConsumerGroup sets the consumer group.
func WithStreamConsumerGroup(group string) StreamOption {
	return func(o *StreamOptions) {
		o.ConsumerGroup = group
	}
}

// WithStreamStartOffset sets the start offset.
func WithStreamStartOffset(offset string) StreamOption {
	return func(o *StreamOptions) {
		o.StartOffset = offset
	}
}

// WithStreamStartTimestamp sets the start timestamp.
func WithStreamStartTimestamp(ts time.Time) StreamOption {
	return func(o *StreamOptions) {
		o.StartTimestamp = ts
	}
}

// WithStreamPartitions sets the partitions to consume.
func WithStreamPartitions(partitions ...int32) StreamOption {
	return func(o *StreamOptions) {
		o.Partitions = partitions
	}
}

// WithStreamBufferSize sets the channel buffer size.
func WithStreamBufferSize(size int) StreamOption {
	return func(o *StreamOptions) {
		o.BufferSize = size
	}
}

// WithStreamAutoCommit enables automatic offset commits.
func WithStreamAutoCommit(auto bool, interval time.Duration) StreamOption {
	return func(o *StreamOptions) {
		o.AutoCommit = auto
		o.AutoCommitInterval = interval
	}
}

// WithStreamFilterTypes sets event type filters.
func WithStreamFilterTypes(types ...string) StreamOption {
	return func(o *StreamOptions) {
		o.FilterTypes = types
	}
}

// NewEvent creates a new Event with default values.
func NewEvent(eventType, source string, data []byte) *Event {
	return &Event{
		ID:              generateUUID(),
		Type:            eventType,
		Source:          source,
		Data:            data,
		DataContentType: "application/json",
		Timestamp:       time.Now(),
		Headers:         make(map[string]string),
		Version:         "1.0",
	}
}

// WithEventSubject sets the event subject.
func (e *Event) WithEventSubject(subject string) *Event {
	e.Subject = subject
	return e
}

// WithEventKey sets the partition key.
func (e *Event) WithEventKey(key string) *Event {
	e.Key = key
	return e
}

// WithEventTraceID sets the trace ID.
func (e *Event) WithEventTraceID(traceID string) *Event {
	e.TraceID = traceID
	return e
}

// WithEventCorrelationID sets the correlation ID.
func (e *Event) WithEventCorrelationID(correlationID string) *Event {
	e.CorrelationID = correlationID
	return e
}

// WithEventCausationID sets the causation ID.
func (e *Event) WithEventCausationID(causationID string) *Event {
	e.CausationID = causationID
	return e
}

// WithEventHeader adds a header to the event.
func (e *Event) WithEventHeader(key, value string) *Event {
	if e.Headers == nil {
		e.Headers = make(map[string]string)
	}
	e.Headers[key] = value
	return e
}

// WithEventSchema sets the data schema URI.
func (e *Event) WithEventSchema(schema string) *Event {
	e.DataSchema = schema
	return e
}

// Topic name constants for HelixAgent.
const (
	// Event topics
	TopicLLMResponses        = "helixagent.events.llm.responses"
	TopicDebateRounds        = "helixagent.events.debate.rounds"
	TopicVerificationResults = "helixagent.events.verification.results"
	TopicProviderHealth      = "helixagent.events.provider.health"
	TopicAuditLog            = "helixagent.events.audit"
	TopicMetricsEvents       = "helixagent.events.metrics"
	TopicErrors              = "helixagent.events.errors"

	// Streaming topics (high-throughput)
	TopicTokenStream        = "helixagent.stream.tokens"
	TopicSSEEvents          = "helixagent.stream.sse"
	TopicWebSocketMessages  = "helixagent.stream.websocket"

	// Change data capture
	TopicCDCProviders = "helixagent.cdc.providers"
	TopicCDCModels    = "helixagent.cdc.models"
	TopicCDCDebates   = "helixagent.cdc.debates"
)
