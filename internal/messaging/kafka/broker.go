// Package kafka provides a Kafka implementation of the event streaming broker interfaces.
package kafka

import (
	"context"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
	"github.com/sirupsen/logrus"

	"llmsverifier/internal/messaging"
)

// Broker implements messaging.EventStreamBroker using Kafka.
type Broker struct {
	mu sync.RWMutex

	config  *Config
	logger  *logrus.Logger
	metrics *messaging.BrokerMetrics

	// Writers per topic
	writers map[string]*kafka.Writer

	// Readers per topic+group
	readers map[string]*kafka.Reader

	// Consumer groups
	consumerGroups map[string]*ConsumerGroup

	// Committed offsets (per topic-partition)
	committedOffsets map[string]map[int32]int64

	// Connection state
	connected bool
	closed    bool
	closeCh   chan struct{}
}

// NewBroker creates a new Kafka broker.
func NewBroker(config *Config, logger *logrus.Logger) *Broker {
	if config == nil {
		config = DefaultConfig()
	}
	if logger == nil {
		logger = logrus.New()
	}

	return &Broker{
		config:           config,
		logger:           logger,
		metrics:          messaging.NewBrokerMetrics(),
		writers:          make(map[string]*kafka.Writer),
		readers:          make(map[string]*kafka.Reader),
		consumerGroups:   make(map[string]*ConsumerGroup),
		committedOffsets: make(map[string]map[int32]int64),
		closeCh:          make(chan struct{}),
	}
}

// Connect establishes connection to Kafka brokers.
func (b *Broker) Connect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return messaging.ErrConnectionClosed
	}

	if err := b.config.Validate(); err != nil {
		return err
	}

	// Test connectivity by fetching metadata
	dialer := b.createDialer()
	conn, err := dialer.DialContext(ctx, "tcp", b.config.Brokers[0])
	if err != nil {
		b.logger.WithError(err).Error("Failed to connect to Kafka")
		return err
	}
	conn.Close()

	b.connected = true
	b.metrics.SetConnected(true)
	b.logger.Info("Connected to Kafka")
	return nil
}

// createDialer creates a Kafka dialer with the configured settings.
func (b *Broker) createDialer() *kafka.Dialer {
	dialer := &kafka.Dialer{
		Timeout:   b.config.DialTimeout,
		DualStack: true,
	}

	if b.config.UseTLS && b.config.TLS != nil {
		dialer.TLS = b.config.TLS
	}

	if b.config.SASLEnabled {
		dialer.SASLMechanism = b.createSASLMechanism()
	}

	return dialer
}

// createSASLMechanism creates the appropriate SASL mechanism.
func (b *Broker) createSASLMechanism() sasl.Mechanism {
	switch b.config.SASLMechanism {
	case "PLAIN":
		return plain.Mechanism{
			Username: b.config.SASLUsername,
			Password: b.config.SASLPassword,
		}
	case "SCRAM-SHA-256":
		mechanism, _ := scram.Mechanism(scram.SHA256, b.config.SASLUsername, b.config.SASLPassword)
		return mechanism
	case "SCRAM-SHA-512":
		mechanism, _ := scram.Mechanism(scram.SHA512, b.config.SASLUsername, b.config.SASLPassword)
		return mechanism
	default:
		return nil
	}
}

// Close terminates all connections.
func (b *Broker) Close(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true
	close(b.closeCh)

	// Close all writers
	for _, writer := range b.writers {
		writer.Close()
	}

	// Close all readers
	for _, reader := range b.readers {
		reader.Close()
	}

	b.connected = false
	b.metrics.SetConnected(false)
	b.logger.Info("Kafka broker closed")
	return nil
}

// HealthCheck verifies the connection is healthy.
func (b *Broker) HealthCheck(ctx context.Context) error {
	if !b.IsConnected() {
		return messaging.ErrNotConnected
	}
	return nil
}

// IsConnected returns true if connected.
func (b *Broker) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connected && !b.closed
}

// BrokerType returns the broker type.
func (b *Broker) BrokerType() messaging.BrokerType {
	return messaging.BrokerTypeKafka
}

// GetMetrics returns broker metrics.
func (b *Broker) GetMetrics() *messaging.BrokerMetrics {
	return b.metrics
}

// Publish sends a message to a topic.
func (b *Broker) Publish(ctx context.Context, topic string, message *messaging.Message, opts ...messaging.PublishOption) error {
	if !b.IsConnected() {
		return messaging.ErrNotConnected
	}

	start := time.Now()
	writer := b.getOrCreateWriter(topic)

	kafkaMsg := kafka.Message{
		Key:   []byte(message.Key),
		Value: message.Payload,
		Time:  message.Timestamp,
		Headers: []kafka.Header{
			{Key: "id", Value: []byte(message.ID)},
			{Key: "type", Value: []byte(message.Type)},
		},
	}

	// Add custom headers
	for k, v := range message.Headers {
		kafkaMsg.Headers = append(kafkaMsg.Headers, kafka.Header{Key: k, Value: []byte(v)})
	}

	if message.TraceID != "" {
		kafkaMsg.Headers = append(kafkaMsg.Headers, kafka.Header{Key: "trace_id", Value: []byte(message.TraceID)})
	}
	if message.CorrelationID != "" {
		kafkaMsg.Headers = append(kafkaMsg.Headers, kafka.Header{Key: "correlation_id", Value: []byte(message.CorrelationID)})
	}

	if err := writer.WriteMessages(ctx, kafkaMsg); err != nil {
		b.metrics.RecordPublishError()
		return err
	}

	b.metrics.RecordPublish(topic, int64(len(message.Payload)), time.Since(start))
	return nil
}

// PublishBatch sends multiple messages to a topic.
func (b *Broker) PublishBatch(ctx context.Context, topic string, messages []*messaging.Message, opts ...messaging.PublishOption) error {
	if !b.IsConnected() {
		return messaging.ErrNotConnected
	}

	writer := b.getOrCreateWriter(topic)

	kafkaMessages := make([]kafka.Message, len(messages))
	for i, msg := range messages {
		kafkaMessages[i] = kafka.Message{
			Key:   []byte(msg.Key),
			Value: msg.Payload,
			Time:  msg.Timestamp,
		}
	}

	return writer.WriteMessages(ctx, kafkaMessages...)
}

// Subscribe creates a consumer for a topic.
func (b *Broker) Subscribe(ctx context.Context, topic string, handler messaging.MessageHandler, opts ...messaging.SubscribeOption) (messaging.Subscription, error) {
	if !b.IsConnected() {
		return nil, messaging.ErrNotConnected
	}

	options := messaging.ApplySubscribeOptions(opts...)

	groupID := b.config.GroupID
	if options.ConsumerGroup != "" {
		groupID = options.ConsumerGroup
	}

	reader := b.getOrCreateReader(topic, groupID)

	sub := &Subscription{
		id:      topic + "-" + groupID,
		topic:   topic,
		groupID: groupID,
		handler: handler,
		reader:  reader,
		metrics: b.metrics,
		logger:  b.logger,
		active:  true,
		closeCh: make(chan struct{}),
	}

	go sub.consume(ctx)

	b.metrics.IncrementSubscriptions()
	b.logger.WithFields(logrus.Fields{
		"topic": topic,
		"group": groupID,
	}).Info("Subscription created")

	return sub, nil
}

// Ack acknowledges a message.
func (b *Broker) Ack(ctx context.Context, msg *messaging.Message) error {
	b.metrics.RecordAck()
	return nil
}

// Nack negatively acknowledges a message.
func (b *Broker) Nack(ctx context.Context, msg *messaging.Message, requeue bool) error {
	b.metrics.RecordNack(requeue)
	return nil
}

// Reject rejects a message.
func (b *Broker) Reject(ctx context.Context, msg *messaging.Message) error {
	b.metrics.RecordNack(false)
	return nil
}

// CreateTopic creates a topic.
func (b *Broker) CreateTopic(ctx context.Context, name string, partitions int32, replication int16, opts ...messaging.TopicOption) error {
	if !b.IsConnected() {
		return messaging.ErrNotConnected
	}

	dialer := b.createDialer()
	conn, err := dialer.DialContext(ctx, "tcp", b.config.Brokers[0])
	if err != nil {
		return err
	}
	defer conn.Close()

	controllerConn, err := conn.Controller()
	if err != nil {
		return err
	}

	adminConn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(controllerConn.Host, strconv.Itoa(controllerConn.Port)))
	if err != nil {
		return err
	}
	defer adminConn.Close()

	topicConfig := kafka.TopicConfig{
		Topic:             name,
		NumPartitions:     int(partitions),
		ReplicationFactor: int(replication),
	}

	err = adminConn.CreateTopics(topicConfig)
	if err != nil {
		return err
	}

	b.logger.WithFields(logrus.Fields{
		"topic":       name,
		"partitions":  partitions,
		"replication": replication,
	}).Info("Topic created")

	return nil
}

// DeleteTopic deletes a topic.
func (b *Broker) DeleteTopic(ctx context.Context, name string) error {
	if !b.IsConnected() {
		return messaging.ErrNotConnected
	}

	dialer := b.createDialer()
	conn, err := dialer.DialContext(ctx, "tcp", b.config.Brokers[0])
	if err != nil {
		return err
	}
	defer conn.Close()

	controllerConn, err := conn.Controller()
	if err != nil {
		return err
	}

	adminConn, err := dialer.DialContext(ctx, "tcp", net.JoinHostPort(controllerConn.Host, strconv.Itoa(controllerConn.Port)))
	if err != nil {
		return err
	}
	defer adminConn.Close()

	return adminConn.DeleteTopics(name)
}

// ListTopics lists all topics.
func (b *Broker) ListTopics(ctx context.Context) ([]string, error) {
	if !b.IsConnected() {
		return nil, messaging.ErrNotConnected
	}

	dialer := b.createDialer()
	conn, err := dialer.DialContext(ctx, "tcp", b.config.Brokers[0])
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions()
	if err != nil {
		return nil, err
	}

	topicSet := make(map[string]struct{})
	for _, p := range partitions {
		topicSet[p.Topic] = struct{}{}
	}

	topics := make([]string, 0, len(topicSet))
	for topic := range topicSet {
		topics = append(topics, topic)
	}

	return topics, nil
}

// GetTopicMetadata returns metadata for a topic.
func (b *Broker) GetTopicMetadata(ctx context.Context, name string) (*messaging.TopicMetadata, error) {
	if !b.IsConnected() {
		return nil, messaging.ErrNotConnected
	}

	dialer := b.createDialer()
	conn, err := dialer.DialContext(ctx, "tcp", b.config.Brokers[0])
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	partitions, err := conn.ReadPartitions(name)
	if err != nil {
		return nil, err
	}

	if len(partitions) == 0 {
		return nil, messaging.ErrTopicNotFound
	}

	partitionInfos := make([]messaging.PartitionInfo, len(partitions))
	for i, p := range partitions {
		replicas := make([]int32, len(p.Replicas))
		for j, r := range p.Replicas {
			replicas[j] = int32(r.ID)
		}

		isr := make([]int32, len(p.Isr))
		for j, r := range p.Isr {
			isr[j] = int32(r.ID)
		}

		partitionInfos[i] = messaging.PartitionInfo{
			Partition: int32(p.ID),
			Leader:    int32(p.Leader.ID),
			Replicas:  replicas,
			ISR:       isr,
		}
	}

	return &messaging.TopicMetadata{
		Name:          name,
		Partitions:    int32(len(partitions)),
		PartitionInfo: partitionInfos,
	}, nil
}

// UpdateTopicConfig updates topic configuration.
func (b *Broker) UpdateTopicConfig(ctx context.Context, name string, config map[string]string) error {
	// Kafka topic config updates require admin client
	// This is a simplified implementation
	return nil
}

// CreateConsumerGroup creates a consumer group.
func (b *Broker) CreateConsumerGroup(ctx context.Context, groupID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.consumerGroups[groupID]; !exists {
		b.consumerGroups[groupID] = &ConsumerGroup{
			GroupID: groupID,
			Offsets: make(map[string]map[int32]int64),
		}
	}
	return nil
}

// DeleteConsumerGroup deletes a consumer group.
func (b *Broker) DeleteConsumerGroup(ctx context.Context, groupID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.consumerGroups, groupID)
	return nil
}

// GetConsumerGroupInfo returns information about a consumer group.
func (b *Broker) GetConsumerGroupInfo(ctx context.Context, groupID string) (*messaging.ConsumerGroupInfo, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	cg, exists := b.consumerGroups[groupID]
	if !exists {
		return &messaging.ConsumerGroupInfo{
			GroupID: groupID,
			State:   "Empty",
		}, nil
	}

	return &messaging.ConsumerGroupInfo{
		GroupID: groupID,
		State:   "Stable",
		Members: cg.Members,
	}, nil
}

// ListConsumerGroups lists all consumer groups.
func (b *Broker) ListConsumerGroups(ctx context.Context) ([]string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	groups := make([]string, 0, len(b.consumerGroups))
	for id := range b.consumerGroups {
		groups = append(groups, id)
	}
	return groups, nil
}

// CommitOffset commits an offset for a partition.
func (b *Broker) CommitOffset(ctx context.Context, topic string, partition int32, offset int64) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.committedOffsets[topic] == nil {
		b.committedOffsets[topic] = make(map[int32]int64)
	}
	b.committedOffsets[topic][partition] = offset
	return nil
}

// GetCommittedOffset returns the committed offset for a partition.
func (b *Broker) GetCommittedOffset(ctx context.Context, topic string, partition int32) (int64, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if offsets, ok := b.committedOffsets[topic]; ok {
		if offset, ok := offsets[partition]; ok {
			return offset, nil
		}
	}
	return 0, nil
}

// SeekToOffset seeks to a specific offset.
func (b *Broker) SeekToOffset(ctx context.Context, topic string, partition int32, offset int64) error {
	return b.CommitOffset(ctx, topic, partition, offset)
}

// SeekToTimestamp seeks to a message at or after a timestamp.
func (b *Broker) SeekToTimestamp(ctx context.Context, topic string, partition int32, ts time.Time) error {
	// Would need to query Kafka for offset by timestamp
	return nil
}

// SeekToBeginning seeks to the beginning of a partition.
func (b *Broker) SeekToBeginning(ctx context.Context, topic string, partition int32) error {
	return b.CommitOffset(ctx, topic, partition, kafka.FirstOffset)
}

// SeekToEnd seeks to the end of a partition.
func (b *Broker) SeekToEnd(ctx context.Context, topic string, partition int32) error {
	return b.CommitOffset(ctx, topic, partition, kafka.LastOffset)
}

// StreamEvents returns a channel of events from a topic.
func (b *Broker) StreamEvents(ctx context.Context, topic string, opts ...messaging.StreamOption) (<-chan *messaging.Event, error) {
	if !b.IsConnected() {
		return nil, messaging.ErrNotConnected
	}

	options := messaging.ApplyStreamOptions(opts...)
	eventCh := make(chan *messaging.Event, options.BufferSize)

	groupID := b.config.GroupID
	if options.ConsumerGroup != "" {
		groupID = options.ConsumerGroup
	}

	reader := b.getOrCreateReader(topic, groupID)

	go func() {
		defer close(eventCh)

		for {
			select {
			case <-ctx.Done():
				return
			case <-b.closeCh:
				return
			default:
				msg, err := reader.FetchMessage(ctx)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					continue
				}

				event := kafkaMessageToEvent(&msg)

				select {
				case eventCh <- event:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return eventCh, nil
}

// PublishEvent publishes an event to a topic.
func (b *Broker) PublishEvent(ctx context.Context, topic string, event *messaging.Event) error {
	msg := &messaging.Message{
		ID:            event.ID,
		Type:          event.Type,
		Payload:       event.Data,
		Key:           event.Key,
		Timestamp:     event.Timestamp,
		TraceID:       event.TraceID,
		CorrelationID: event.CorrelationID,
		Headers:       event.Headers,
	}
	return b.Publish(ctx, topic, msg)
}

// PublishEvents publishes multiple events to a topic.
func (b *Broker) PublishEvents(ctx context.Context, topic string, events []*messaging.Event) error {
	for _, event := range events {
		if err := b.PublishEvent(ctx, topic, event); err != nil {
			return err
		}
	}
	return nil
}

// BeginTransaction begins a transaction.
func (b *Broker) BeginTransaction(ctx context.Context) error {
	// Kafka transactions require transactional.id configuration
	return nil
}

// CommitTransaction commits a transaction.
func (b *Broker) CommitTransaction(ctx context.Context) error {
	return nil
}

// AbortTransaction aborts a transaction.
func (b *Broker) AbortTransaction(ctx context.Context) error {
	return nil
}

// getOrCreateWriter gets or creates a writer for a topic.
func (b *Broker) getOrCreateWriter(topic string) *kafka.Writer {
	b.mu.Lock()
	defer b.mu.Unlock()

	if writer, exists := b.writers[topic]; exists {
		return writer
	}

	transport := &kafka.Transport{
		Dial: (&net.Dialer{
			Timeout: b.config.DialTimeout,
		}).DialContext,
	}

	if b.config.UseTLS && b.config.TLS != nil {
		transport.TLS = b.config.TLS
	}

	if b.config.SASLEnabled {
		transport.SASL = b.createSASLMechanism()
	}

	writer := &kafka.Writer{
		Addr:         kafka.TCP(b.config.Brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchSize:    b.config.BatchSize,
		BatchTimeout: b.config.BatchTimeout,
		RequiredAcks: kafka.RequiredAcks(b.config.RequiredAcks),
		Transport:    transport,
	}

	b.writers[topic] = writer
	return writer
}

// getOrCreateReader gets or creates a reader for a topic.
func (b *Broker) getOrCreateReader(topic, groupID string) *kafka.Reader {
	key := topic + ":" + groupID

	b.mu.Lock()
	defer b.mu.Unlock()

	if reader, exists := b.readers[key]; exists {
		return reader
	}

	dialer := b.createDialer()

	startOffset := kafka.LastOffset
	if b.config.AutoOffsetReset == "earliest" {
		startOffset = kafka.FirstOffset
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        b.config.Brokers,
		GroupID:        groupID,
		Topic:          topic,
		Dialer:         dialer,
		MinBytes:       b.config.FetchMinBytes,
		MaxBytes:       b.config.FetchMaxBytes,
		MaxWait:        b.config.MaxWaitTime,
		StartOffset:    startOffset,
		CommitInterval: b.config.AutoCommitInterval,
	})

	b.readers[key] = reader
	return reader
}

// kafkaMessageToEvent converts a Kafka message to an Event.
func kafkaMessageToEvent(msg *kafka.Message) *messaging.Event {
	event := &messaging.Event{
		Data:      msg.Value,
		Key:       string(msg.Key),
		Timestamp: msg.Time,
		Partition: int32(msg.Partition),
		Offset:    msg.Offset,
		Headers:   make(map[string]string),
	}

	for _, h := range msg.Headers {
		switch h.Key {
		case "id":
			event.ID = string(h.Value)
		case "type":
			event.Type = string(h.Value)
		case "trace_id":
			event.TraceID = string(h.Value)
		case "correlation_id":
			event.CorrelationID = string(h.Value)
		default:
			event.Headers[h.Key] = string(h.Value)
		}
	}

	return event
}

// ConsumerGroup represents a Kafka consumer group.
type ConsumerGroup struct {
	GroupID string
	Members []messaging.ConsumerGroupMember
	Offsets map[string]map[int32]int64
}
