// Package inmemory provides an in-memory message broker for testing and fallback scenarios.
package inmemory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"llmsverifier/internal/messaging"
)

// Broker implements MessageBroker using in-memory data structures.
type Broker struct {
	mu sync.RWMutex

	config    *Config
	logger    *logrus.Logger
	connected bool

	// Queues and topics
	queues map[string]*Queue
	topics map[string]*Topic

	// Subscriptions
	subscriptions map[string][]*subscription

	// Metrics
	metrics *messaging.BrokerMetrics

	// Shutdown
	closed  bool
	closeCh chan struct{}
}

// Config contains configuration for the in-memory broker.
type Config struct {
	// MaxQueueSize is the maximum number of messages per queue.
	MaxQueueSize int `json:"max_queue_size" yaml:"max_queue_size"`

	// MaxTopicSize is the maximum number of messages per topic partition.
	MaxTopicSize int `json:"max_topic_size" yaml:"max_topic_size"`

	// DefaultPartitions is the default number of partitions for topics.
	DefaultPartitions int `json:"default_partitions" yaml:"default_partitions"`

	// MessageTTL is the default message time-to-live.
	MessageTTL time.Duration `json:"message_ttl" yaml:"message_ttl"`

	// DeliveryDelay simulates network latency.
	DeliveryDelay time.Duration `json:"delivery_delay" yaml:"delivery_delay"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		MaxQueueSize:      100000,
		MaxTopicSize:      100000,
		DefaultPartitions: 3,
		MessageTTL:        24 * time.Hour,
		DeliveryDelay:     0,
	}
}

// NewBroker creates a new in-memory broker.
func NewBroker(config *Config, logger *logrus.Logger) *Broker {
	if config == nil {
		config = DefaultConfig()
	}
	if logger == nil {
		logger = logrus.New()
	}

	return &Broker{
		config:        config,
		logger:        logger,
		queues:        make(map[string]*Queue),
		topics:        make(map[string]*Topic),
		subscriptions: make(map[string][]*subscription),
		metrics:       messaging.NewBrokerMetrics(),
		closeCh:       make(chan struct{}),
	}
}

// Connect establishes a connection (no-op for in-memory).
func (b *Broker) Connect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return messaging.ErrConnectionClosed
	}

	b.connected = true
	b.metrics.SetConnected(true)
	b.logger.Info("In-memory broker connected")
	return nil
}

// Close terminates the broker.
func (b *Broker) Close(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true
	b.connected = false
	close(b.closeCh)

	// Close all subscriptions
	for _, subs := range b.subscriptions {
		for _, sub := range subs {
			sub.close()
		}
	}

	b.metrics.SetConnected(false)
	b.logger.Info("In-memory broker closed")
	return nil
}

// HealthCheck verifies the broker is healthy.
func (b *Broker) HealthCheck(ctx context.Context) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if !b.connected {
		return messaging.ErrNotConnected
	}
	return nil
}

// IsConnected returns true if connected.
func (b *Broker) IsConnected() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.connected
}

// Publish sends a message to a topic or queue.
func (b *Broker) Publish(ctx context.Context, target string, message *messaging.Message, opts ...messaging.PublishOption) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.connected {
		return messaging.ErrNotConnected
	}

	start := time.Now()
	_ = messaging.ApplyPublishOptions(opts...) // Options reserved for future use

	// Simulate delivery delay
	if b.config.DeliveryDelay > 0 {
		time.Sleep(b.config.DeliveryDelay)
	}

	// Try to publish to queue first, then topic
	if queue, ok := b.queues[target]; ok {
		if err := queue.Push(message); err != nil {
			b.metrics.RecordPublishError()
			return err
		}
	} else if topic, ok := b.topics[target]; ok {
		if err := topic.Publish(message); err != nil {
			b.metrics.RecordPublishError()
			return err
		}
	} else {
		// Auto-create queue
		queue := NewQueue(target, b.config.MaxQueueSize, b.config.MessageTTL)
		b.queues[target] = queue
		if err := queue.Push(message); err != nil {
			b.metrics.RecordPublishError()
			return err
		}
	}

	b.metrics.RecordPublish(target, int64(len(message.Payload)), time.Since(start))

	// Deliver to subscribers
	b.deliverToSubscribers(target, message)

	return nil
}

// PublishBatch sends multiple messages.
func (b *Broker) PublishBatch(ctx context.Context, target string, messages []*messaging.Message, opts ...messaging.PublishOption) error {
	for _, msg := range messages {
		if err := b.Publish(ctx, target, msg, opts...); err != nil {
			return err
		}
	}
	return nil
}

// Subscribe registers a handler for messages.
func (b *Broker) Subscribe(ctx context.Context, target string, handler messaging.MessageHandler, opts ...messaging.SubscribeOption) (messaging.Subscription, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.connected {
		return nil, messaging.ErrNotConnected
	}

	options := messaging.ApplySubscribeOptions(opts...)

	sub := &subscription{
		id:       fmt.Sprintf("sub-%s-%d", target, time.Now().UnixNano()),
		topic:    target,
		handler:  handler,
		options:  options,
		active:   true,
		messages: make(chan *messaging.Message, 1000),
		closeCh:  make(chan struct{}),
	}

	b.subscriptions[target] = append(b.subscriptions[target], sub)
	b.metrics.IncrementSubscriptions()

	// Start message delivery goroutine
	go sub.run(ctx, b.metrics, b.logger)

	// If subscribing to a queue, start consuming existing messages
	if queue, ok := b.queues[target]; ok {
		go func() {
			for {
				select {
				case <-sub.closeCh:
					return
				default:
					msg, err := queue.Pop()
					if err != nil {
						time.Sleep(100 * time.Millisecond)
						continue
					}
					select {
					case sub.messages <- msg:
					case <-sub.closeCh:
						return
					}
				}
			}
		}()
	}

	b.logger.WithField("target", target).Info("Subscription created")
	return sub, nil
}

// BrokerType returns the broker type.
func (b *Broker) BrokerType() messaging.BrokerType {
	return messaging.BrokerTypeInMemory
}

// GetMetrics returns broker metrics.
func (b *Broker) GetMetrics() *messaging.BrokerMetrics {
	return b.metrics
}

// Ack acknowledges a message (no-op for in-memory).
func (b *Broker) Ack(ctx context.Context, msg *messaging.Message) error {
	b.metrics.RecordAck()
	return nil
}

// Nack negatively acknowledges a message.
func (b *Broker) Nack(ctx context.Context, msg *messaging.Message, requeue bool) error {
	b.metrics.RecordNack(requeue)
	if requeue {
		// Re-publish the message
		return b.Publish(ctx, "", msg) // Would need the original queue name
	}
	return nil
}

// Reject permanently rejects a message.
func (b *Broker) Reject(ctx context.Context, msg *messaging.Message) error {
	b.metrics.RecordNack(false)
	return nil
}

// DeclareQueue creates a queue if it doesn't exist.
func (b *Broker) DeclareQueue(ctx context.Context, name string, opts ...messaging.QueueOption) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.connected {
		return messaging.ErrNotConnected
	}

	if _, exists := b.queues[name]; !exists {
		b.queues[name] = NewQueue(name, b.config.MaxQueueSize, b.config.MessageTTL)
	}

	b.logger.WithField("queue", name).Info("Queue declared")
	return nil
}

// DeleteQueue deletes a queue.
func (b *Broker) DeleteQueue(ctx context.Context, name string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.queues, name)
	return nil
}

// PurgeQueue removes all messages from a queue.
func (b *Broker) PurgeQueue(ctx context.Context, name string) (int64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if queue, ok := b.queues[name]; ok {
		count := queue.Len()
		queue.Clear()
		return int64(count), nil
	}
	return 0, nil
}

// EnqueueTask adds a task to a queue.
func (b *Broker) EnqueueTask(ctx context.Context, queue string, task *messaging.Task) error {
	msg := &messaging.Message{
		ID:            task.ID,
		Type:          task.Type,
		Payload:       task.Payload,
		Priority:      task.Priority,
		Timestamp:     task.CreatedAt,
		TraceID:       task.TraceID,
		CorrelationID: task.CorrelationID,
		Headers:       task.Metadata,
	}
	return b.Publish(ctx, queue, msg)
}

// EnqueueTasks adds multiple tasks to a queue.
func (b *Broker) EnqueueTasks(ctx context.Context, queue string, tasks []*messaging.Task) error {
	for _, task := range tasks {
		if err := b.EnqueueTask(ctx, queue, task); err != nil {
			return err
		}
	}
	return nil
}

// DequeueTask retrieves a task from a queue.
func (b *Broker) DequeueTask(ctx context.Context, queue string, workerID string) (*messaging.Task, error) {
	b.mu.Lock()
	q, ok := b.queues[queue]
	b.mu.Unlock()

	if !ok {
		return nil, messaging.ErrQueueNotFound
	}

	msg, err := q.Pop()
	if err != nil {
		return nil, err
	}

	return &messaging.Task{
		ID:            msg.ID,
		Type:          msg.Type,
		Payload:       msg.Payload,
		Priority:      msg.Priority,
		CreatedAt:     msg.Timestamp,
		TraceID:       msg.TraceID,
		CorrelationID: msg.CorrelationID,
		Metadata:      msg.Headers,
		WorkerID:      workerID,
		Status:        messaging.TaskStatusRunning,
		StartedAt:     time.Now(),
		DeliveryTag:   msg.DeliveryTag,
	}, nil
}

// AckTask acknowledges task completion.
func (b *Broker) AckTask(ctx context.Context, task *messaging.Task) error {
	b.metrics.RecordAck()
	return nil
}

// NackTask indicates task failure.
func (b *Broker) NackTask(ctx context.Context, task *messaging.Task, requeue bool) error {
	b.metrics.RecordNack(requeue)
	if requeue {
		task.RetryCount++
		task.Status = messaging.TaskStatusRetrying
		return b.EnqueueTask(ctx, "", task) // Would need original queue
	}
	return nil
}

// MoveToDeadLetter moves a task to dead letter queue.
func (b *Broker) MoveToDeadLetter(ctx context.Context, task *messaging.Task, reason string) error {
	task.Status = messaging.TaskStatusDeadLetter
	task.Error = reason
	b.metrics.RecordDeadLetter()
	return b.EnqueueTask(ctx, messaging.QueueDeadLetter, task)
}

// GetQueueStats returns queue statistics.
func (b *Broker) GetQueueStats(ctx context.Context, queue string) (*messaging.QueueStats, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	q, ok := b.queues[queue]
	if !ok {
		return nil, messaging.ErrQueueNotFound
	}

	subs := b.subscriptions[queue]

	return &messaging.QueueStats{
		Name:          queue,
		Messages:      int64(q.Len()),
		MessagesReady: int64(q.Len()),
		Consumers:     len(subs),
	}, nil
}

// ScheduleTask schedules a task for future execution.
func (b *Broker) ScheduleTask(ctx context.Context, queue string, task *messaging.Task, executeAt time.Time) error {
	task.ScheduledAt = executeAt
	task.Status = messaging.TaskStatusScheduled

	// Start a goroutine to enqueue at the right time
	go func() {
		delay := time.Until(executeAt)
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
				b.EnqueueTask(ctx, queue, task)
			case <-b.closeCh:
				timer.Stop()
			}
		} else {
			b.EnqueueTask(ctx, queue, task)
		}
	}()

	return nil
}

// CancelTask attempts to cancel a task.
func (b *Broker) CancelTask(ctx context.Context, queue string, taskID string) error {
	// In-memory doesn't support task cancellation
	return nil
}

// GetTask retrieves a task by ID.
func (b *Broker) GetTask(ctx context.Context, queue string, taskID string) (*messaging.Task, error) {
	// In-memory doesn't support task lookup
	return nil, messaging.ErrNotConnected
}

// CreateTopic creates a topic.
func (b *Broker) CreateTopic(ctx context.Context, name string, partitions int32, replication int16, opts ...messaging.TopicOption) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.connected {
		return messaging.ErrNotConnected
	}

	if _, exists := b.topics[name]; !exists {
		b.topics[name] = NewTopic(name, int(partitions), b.config.MaxTopicSize)
	}

	b.logger.WithFields(logrus.Fields{
		"topic":      name,
		"partitions": partitions,
	}).Info("Topic created")
	return nil
}

// DeleteTopic deletes a topic.
func (b *Broker) DeleteTopic(ctx context.Context, name string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	delete(b.topics, name)
	return nil
}

// ListTopics lists all topics.
func (b *Broker) ListTopics(ctx context.Context) ([]string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	topics := make([]string, 0, len(b.topics))
	for name := range b.topics {
		topics = append(topics, name)
	}
	return topics, nil
}

// GetTopicMetadata returns topic metadata.
func (b *Broker) GetTopicMetadata(ctx context.Context, name string) (*messaging.TopicMetadata, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	topic, ok := b.topics[name]
	if !ok {
		return nil, messaging.ErrTopicNotFound
	}

	return &messaging.TopicMetadata{
		Name:       name,
		Partitions: topic.NumPartitions(),
	}, nil
}

// UpdateTopicConfig updates topic configuration.
func (b *Broker) UpdateTopicConfig(ctx context.Context, name string, config map[string]string) error {
	// Not supported for in-memory
	return nil
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

// PublishEvents publishes multiple events.
func (b *Broker) PublishEvents(ctx context.Context, topic string, events []*messaging.Event) error {
	for _, event := range events {
		if err := b.PublishEvent(ctx, topic, event); err != nil {
			return err
		}
	}
	return nil
}

// CreateConsumerGroup creates a consumer group.
func (b *Broker) CreateConsumerGroup(ctx context.Context, groupID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.connected {
		return messaging.ErrNotConnected
	}

	// Consumer groups are automatically created when subscribing
	return nil
}

// DeleteConsumerGroup deletes a consumer group.
func (b *Broker) DeleteConsumerGroup(ctx context.Context, groupID string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Remove consumer group from all topics
	for _, topic := range b.topics {
		topic.DeleteConsumerGroup(groupID)
	}
	return nil
}

// GetConsumerGroupInfo returns information about a consumer group.
func (b *Broker) GetConsumerGroupInfo(ctx context.Context, groupID string) (*messaging.ConsumerGroupInfo, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Look for consumer group in topics
	for _, topic := range b.topics {
		cg := topic.GetOrCreateConsumerGroup(groupID)
		if cg != nil {
			members := cg.Members()
			cgMembers := make([]messaging.ConsumerGroupMember, len(members))
			for i, m := range members {
				cgMembers[i] = messaging.ConsumerGroupMember{
					MemberID:   m.MemberID,
					ClientID:   m.ClientID,
					Partitions: m.Partitions,
				}
			}

			return &messaging.ConsumerGroupInfo{
				GroupID:  groupID,
				State:    "Stable",
				Protocol: "range",
				Members:  cgMembers,
			}, nil
		}
	}

	return &messaging.ConsumerGroupInfo{
		GroupID: groupID,
		State:   "Empty",
	}, nil
}

// ListConsumerGroups lists all consumer groups.
func (b *Broker) ListConsumerGroups(ctx context.Context) ([]string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	groupSet := make(map[string]struct{})
	for _, topic := range b.topics {
		topic.mu.RLock()
		for groupID := range topic.consumerGroups {
			groupSet[groupID] = struct{}{}
		}
		topic.mu.RUnlock()
	}

	groups := make([]string, 0, len(groupSet))
	for group := range groupSet {
		groups = append(groups, group)
	}
	return groups, nil
}

// CommitOffset commits an offset for a partition.
func (b *Broker) CommitOffset(ctx context.Context, topicName string, partition int32, offset int64) error {
	b.mu.RLock()
	topic, ok := b.topics[topicName]
	b.mu.RUnlock()

	if !ok {
		return messaging.ErrTopicNotFound
	}

	// Use a default consumer group for non-group commits
	cg := topic.GetOrCreateConsumerGroup("default")
	cg.CommitOffset(partition, offset)
	return nil
}

// GetCommittedOffset returns the committed offset for a partition.
func (b *Broker) GetCommittedOffset(ctx context.Context, topicName string, partition int32) (int64, error) {
	b.mu.RLock()
	topic, ok := b.topics[topicName]
	b.mu.RUnlock()

	if !ok {
		return 0, messaging.ErrTopicNotFound
	}

	cg := topic.GetOrCreateConsumerGroup("default")
	return cg.GetCommittedOffset(partition), nil
}

// SeekToOffset seeks to a specific offset.
func (b *Broker) SeekToOffset(ctx context.Context, topicName string, partition int32, offset int64) error {
	b.mu.RLock()
	topic, ok := b.topics[topicName]
	b.mu.RUnlock()

	if !ok {
		return messaging.ErrTopicNotFound
	}

	cg := topic.GetOrCreateConsumerGroup("default")
	cg.CommitOffset(partition, offset)
	return nil
}

// SeekToTimestamp seeks to a message at or after a timestamp.
func (b *Broker) SeekToTimestamp(ctx context.Context, topicName string, partition int32, ts time.Time) error {
	b.mu.RLock()
	topic, ok := b.topics[topicName]
	b.mu.RUnlock()

	if !ok {
		return messaging.ErrTopicNotFound
	}

	p := topic.GetPartition(partition)
	if p == nil {
		return messaging.ErrPartitionNotFound
	}

	// Scan partition to find offset at timestamp
	messages, _ := topic.Read(partition, p.BeginOffset(), p.Size())
	for _, msg := range messages {
		if !msg.Timestamp.Before(ts) {
			cg := topic.GetOrCreateConsumerGroup("default")
			cg.CommitOffset(partition, msg.Offset)
			return nil
		}
	}

	// If no message found, seek to end
	cg := topic.GetOrCreateConsumerGroup("default")
	cg.CommitOffset(partition, p.EndOffset())
	return nil
}

// SeekToBeginning seeks to the beginning of a partition.
func (b *Broker) SeekToBeginning(ctx context.Context, topicName string, partition int32) error {
	b.mu.RLock()
	topic, ok := b.topics[topicName]
	b.mu.RUnlock()

	if !ok {
		return messaging.ErrTopicNotFound
	}

	p := topic.GetPartition(partition)
	if p == nil {
		return messaging.ErrPartitionNotFound
	}

	cg := topic.GetOrCreateConsumerGroup("default")
	cg.CommitOffset(partition, p.BeginOffset())
	return nil
}

// SeekToEnd seeks to the end of a partition.
func (b *Broker) SeekToEnd(ctx context.Context, topicName string, partition int32) error {
	b.mu.RLock()
	topic, ok := b.topics[topicName]
	b.mu.RUnlock()

	if !ok {
		return messaging.ErrTopicNotFound
	}

	p := topic.GetPartition(partition)
	if p == nil {
		return messaging.ErrPartitionNotFound
	}

	cg := topic.GetOrCreateConsumerGroup("default")
	cg.CommitOffset(partition, p.EndOffset())
	return nil
}

// StreamEvents returns a channel of events from a topic.
func (b *Broker) StreamEvents(ctx context.Context, topicName string, opts ...messaging.StreamOption) (<-chan *messaging.Event, error) {
	b.mu.RLock()
	topic, ok := b.topics[topicName]
	b.mu.RUnlock()

	if !ok {
		return nil, messaging.ErrTopicNotFound
	}

	options := messaging.ApplyStreamOptions(opts...)
	eventCh := make(chan *messaging.Event, options.BufferSize)

	// Start a goroutine to stream events
	go func() {
		defer close(eventCh)

		// Determine starting offset
		var currentOffsets = make(map[int32]int64)
		for i := int32(0); i < topic.NumPartitions(); i++ {
			if options.StartOffset == "earliest" {
				p := topic.GetPartition(i)
				if p != nil {
					currentOffsets[i] = p.BeginOffset()
				}
			} else {
				p := topic.GetPartition(i)
				if p != nil {
					currentOffsets[i] = p.EndOffset()
				}
			}
		}

		ticker := time.NewTicker(options.PollTimeout)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-b.closeCh:
				return
			case <-ticker.C:
				// Poll each partition
				for i := int32(0); i < topic.NumPartitions(); i++ {
					messages, err := topic.Read(i, currentOffsets[i], options.MaxPollRecords)
					if err != nil {
						continue
					}

					for _, msg := range messages {
						event := &messaging.Event{
							ID:            msg.ID,
							Type:          msg.Type,
							Data:          msg.Payload,
							Key:           msg.Key,
							Timestamp:     msg.Timestamp,
							Partition:     msg.Partition,
							Offset:        msg.Offset,
							TraceID:       msg.TraceID,
							CorrelationID: msg.CorrelationID,
							Headers:       msg.Headers,
						}

						select {
						case eventCh <- event:
							currentOffsets[i] = msg.Offset + 1
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}
	}()

	return eventCh, nil
}

// BeginTransaction begins a transaction.
func (b *Broker) BeginTransaction(ctx context.Context) error {
	// In-memory broker doesn't support true transactions
	return nil
}

// CommitTransaction commits a transaction.
func (b *Broker) CommitTransaction(ctx context.Context) error {
	// In-memory broker doesn't support true transactions
	return nil
}

// AbortTransaction aborts a transaction.
func (b *Broker) AbortTransaction(ctx context.Context) error {
	// In-memory broker doesn't support true transactions
	return nil
}

// deliverToSubscribers delivers a message to all subscribers.
func (b *Broker) deliverToSubscribers(target string, msg *messaging.Message) {
	subs := b.subscriptions[target]
	for _, sub := range subs {
		if sub.active {
			select {
			case sub.messages <- msg:
			default:
				// Channel full, message dropped
				b.logger.Warn("Subscriber channel full, message dropped")
			}
		}
	}
}

// subscription represents an active subscription.
type subscription struct {
	mu       sync.Mutex
	id       string
	topic    string
	handler  messaging.MessageHandler
	options  *messaging.SubscribeOptions
	active   bool
	messages chan *messaging.Message
	closeCh  chan struct{}
}

// Unsubscribe cancels the subscription.
func (s *subscription) Unsubscribe() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return nil
	}

	s.active = false
	close(s.closeCh)
	return nil
}

// IsActive returns true if subscription is active.
func (s *subscription) IsActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active
}

// Topic returns the subscription topic.
func (s *subscription) Topic() string {
	return s.topic
}

// close closes the subscription.
func (s *subscription) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active {
		s.active = false
		close(s.closeCh)
	}
}

// run processes messages for the subscription.
func (s *subscription) run(ctx context.Context, metrics *messaging.BrokerMetrics, logger *logrus.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.closeCh:
			return
		case msg := <-s.messages:
			start := time.Now()
			err := s.handler(ctx, msg)
			if err != nil {
				metrics.RecordFailure()
				logger.WithError(err).WithField("message_id", msg.ID).Error("Message handling failed")
			}
			metrics.RecordConsume(s.topic, int64(len(msg.Payload)), time.Since(start))
		}
	}
}
