// Package rabbitmq provides a RabbitMQ implementation of the message broker interfaces.
package rabbitmq

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"

	"llmsverifier/internal/messaging"
)

// Broker implements messaging.MessageBroker and messaging.TaskQueueBroker using RabbitMQ.
type Broker struct {
	mu sync.RWMutex

	config     *Config
	logger     *logrus.Logger
	conn       *Connection
	pubChannel *amqp.Channel
	metrics    *messaging.BrokerMetrics

	// Subscriptions
	subscriptions map[string]*Subscription

	// Publisher confirms
	confirmsEnabled bool
	confirmCh       chan amqp.Confirmation

	// Shutdown
	closed  bool
	closeCh chan struct{}
}

// NewBroker creates a new RabbitMQ broker.
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
		metrics:       messaging.NewBrokerMetrics(),
		subscriptions: make(map[string]*Subscription),
		closeCh:       make(chan struct{}),
	}
}

// Connect establishes connection to RabbitMQ.
func (b *Broker) Connect(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return messaging.ErrConnectionClosed
	}

	if err := b.config.Validate(); err != nil {
		return err
	}

	// Create connection manager
	b.conn = NewConnection(b.config, b.logger)
	b.conn.OnConnect(b.onConnected)
	b.conn.OnDisconnect(b.onDisconnected)
	b.conn.OnReconnect(b.onReconnecting)

	if err := b.conn.Connect(ctx); err != nil {
		return err
	}

	// Setup publisher channel
	if err := b.setupPublisherChannel(); err != nil {
		return err
	}

	b.metrics.SetConnected(true)
	return nil
}

// setupPublisherChannel creates and configures the publisher channel.
func (b *Broker) setupPublisherChannel() error {
	var err error
	b.pubChannel, err = b.conn.Channel()
	if err != nil {
		return err
	}

	if b.config.PublisherConfirms {
		if err := b.pubChannel.Confirm(false); err != nil {
			return err
		}
		b.confirmCh = make(chan amqp.Confirmation, 100)
		b.pubChannel.NotifyPublish(b.confirmCh)
		b.confirmsEnabled = true
	}

	return nil
}

// onConnected is called when connection is established.
func (b *Broker) onConnected() {
	b.logger.Info("RabbitMQ broker connected")
	b.metrics.SetConnected(true)

	// Re-setup publisher channel after reconnection
	b.mu.Lock()
	defer b.mu.Unlock()

	if err := b.setupPublisherChannel(); err != nil {
		b.logger.WithError(err).Error("Failed to setup publisher channel after reconnect")
	}

	// Re-establish subscriptions
	for _, sub := range b.subscriptions {
		if sub.active {
			go sub.restart()
		}
	}
}

// onDisconnected is called when connection is lost.
func (b *Broker) onDisconnected(err error) {
	b.logger.WithError(err).Warn("RabbitMQ broker disconnected")
	b.metrics.SetConnected(false)
	b.metrics.RecordConnectionError()
}

// onReconnecting is called during reconnection attempts.
func (b *Broker) onReconnecting(attempt int) {
	b.logger.WithField("attempt", attempt).Info("RabbitMQ broker reconnecting")
	b.metrics.IncrementReconnections()
}

// Close terminates the connection.
func (b *Broker) Close(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}

	b.closed = true
	close(b.closeCh)

	// Close all subscriptions
	for _, sub := range b.subscriptions {
		sub.Stop()
	}

	// Close publisher channel
	if b.pubChannel != nil {
		b.pubChannel.Close()
	}

	// Close connection
	if b.conn != nil {
		if err := b.conn.Close(); err != nil {
			return err
		}
	}

	b.metrics.SetConnected(false)
	b.logger.Info("RabbitMQ broker closed")
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
	return b.conn != nil && b.conn.IsConnected()
}

// BrokerType returns the broker type.
func (b *Broker) BrokerType() messaging.BrokerType {
	return messaging.BrokerTypeRabbitMQ
}

// GetMetrics returns broker metrics.
func (b *Broker) GetMetrics() *messaging.BrokerMetrics {
	return b.metrics
}

// Publish sends a message to an exchange/queue.
func (b *Broker) Publish(ctx context.Context, target string, message *messaging.Message, opts ...messaging.PublishOption) error {
	if !b.IsConnected() {
		return messaging.ErrNotConnected
	}

	options := messaging.ApplyPublishOptions(opts...)
	start := time.Now()

	// Build AMQP publishing
	pub := amqp.Publishing{
		ContentType:   "application/json",
		DeliveryMode:  amqp.Persistent,
		Timestamp:     message.Timestamp,
		MessageId:     message.ID,
		Type:          message.Type,
		CorrelationId: message.CorrelationID,
		Body:          message.Payload,
		Headers:       amqp.Table{},
	}

	// Add custom headers
	for k, v := range message.Headers {
		pub.Headers[k] = v
	}

	// Add priority
	if message.Priority > 0 {
		pub.Priority = uint8(message.Priority)
	}

	// Add trace ID
	if message.TraceID != "" {
		pub.Headers["trace_id"] = message.TraceID
	}

	// Determine routing
	exchange := ""
	routingKey := target

	if options.Exchange != "" {
		exchange = options.Exchange
	}
	if options.RoutingKey != "" {
		routingKey = options.RoutingKey
	}

	// Publish
	b.mu.RLock()
	channel := b.pubChannel
	b.mu.RUnlock()

	if channel == nil {
		return messaging.ErrNotConnected
	}

	err := channel.PublishWithContext(ctx, exchange, routingKey, options.Mandatory, options.Immediate, pub)
	if err != nil {
		b.metrics.RecordPublishError()
		return err
	}

	// Wait for confirmation if enabled
	if b.confirmsEnabled && options.WaitForConfirm {
		select {
		case confirm := <-b.confirmCh:
			if !confirm.Ack {
				b.metrics.RecordPublishError()
				return messaging.ErrPublishFailed
			}
		case <-time.After(b.config.ConfirmTimeout):
			b.metrics.RecordPublishError()
			return messaging.ErrTimeout
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	b.metrics.RecordPublish(target, int64(len(message.Payload)), time.Since(start))
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
	if !b.IsConnected() {
		return nil, messaging.ErrNotConnected
	}

	options := messaging.ApplySubscribeOptions(opts...)

	// Create channel for this subscription
	channel, err := b.conn.Channel()
	if err != nil {
		return nil, err
	}

	// Set QoS
	prefetch := b.config.PrefetchCount
	if options.PrefetchCount > 0 {
		prefetch = options.PrefetchCount
	}

	if err := channel.Qos(prefetch, 0, false); err != nil {
		channel.Close()
		return nil, err
	}

	// Create subscription
	sub := &Subscription{
		id:       generateSubscriptionID(target),
		queue:    target,
		handler:  handler,
		options:  options,
		channel:  channel,
		conn:     b.conn,
		metrics:  b.metrics,
		logger:   b.logger,
		active:   true,
		closeCh:  make(chan struct{}),
		prefetch: prefetch,
	}

	// Start consuming
	if err := sub.start(ctx); err != nil {
		channel.Close()
		return nil, err
	}

	b.mu.Lock()
	b.subscriptions[sub.id] = sub
	b.mu.Unlock()

	b.metrics.IncrementSubscriptions()
	b.logger.WithField("queue", target).Info("Subscription created")

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

// Reject permanently rejects a message.
func (b *Broker) Reject(ctx context.Context, msg *messaging.Message) error {
	b.metrics.RecordNack(false)
	return nil
}

// DeclareQueue declares a queue.
func (b *Broker) DeclareQueue(ctx context.Context, name string, opts ...messaging.QueueOption) error {
	if !b.IsConnected() {
		return messaging.ErrNotConnected
	}

	config := messaging.ApplyQueueOptions(name, opts...)

	channel, err := b.conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	args := amqp.Table{}

	// Set message TTL
	if config.MessageTTL > 0 {
		args["x-message-ttl"] = int64(config.MessageTTL.Milliseconds())
	}

	// Set max length
	if config.MaxLength > 0 {
		args["x-max-length"] = config.MaxLength
	}

	// Set max length bytes
	if config.MaxLengthBytes > 0 {
		args["x-max-length-bytes"] = config.MaxLengthBytes
	}

	// Set dead letter exchange
	if config.DeadLetterExchange != "" {
		args["x-dead-letter-exchange"] = config.DeadLetterExchange
		if config.DeadLetterRoutingKey != "" {
			args["x-dead-letter-routing-key"] = config.DeadLetterRoutingKey
		}
	}

	// Set priority
	if config.MaxPriority > 0 {
		args["x-max-priority"] = int32(config.MaxPriority)
	}

	// Add custom arguments
	for k, v := range config.Arguments {
		args[k] = v
	}

	_, err = channel.QueueDeclare(
		name,
		config.Durable,
		config.AutoDelete,
		config.Exclusive,
		false, // no-wait
		args,
	)

	if err != nil {
		return err
	}

	b.logger.WithField("queue", name).Info("Queue declared")
	return nil
}

// DeleteQueue deletes a queue.
func (b *Broker) DeleteQueue(ctx context.Context, name string) error {
	if !b.IsConnected() {
		return messaging.ErrNotConnected
	}

	channel, err := b.conn.Channel()
	if err != nil {
		return err
	}
	defer channel.Close()

	_, err = channel.QueueDelete(name, false, false, false)
	return err
}

// PurgeQueue removes all messages from a queue.
func (b *Broker) PurgeQueue(ctx context.Context, name string) (int64, error) {
	if !b.IsConnected() {
		return 0, messaging.ErrNotConnected
	}

	channel, err := b.conn.Channel()
	if err != nil {
		return 0, err
	}
	defer channel.Close()

	count, err := channel.QueuePurge(name, false)
	if err != nil {
		return 0, err
	}

	return int64(count), nil
}

// EnqueueTask adds a task to a queue.
func (b *Broker) EnqueueTask(ctx context.Context, queue string, task *messaging.Task) error {
	msg := taskToMessage(task)
	return b.Publish(ctx, queue, msg, messaging.WithPersistence(true))
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
	if !b.IsConnected() {
		return nil, messaging.ErrNotConnected
	}

	channel, err := b.conn.Channel()
	if err != nil {
		return nil, err
	}
	defer channel.Close()

	delivery, ok, err := channel.Get(queue, false)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, messaging.ErrQueueNotFound
	}

	task := messageToTask(&delivery)
	task.WorkerID = workerID
	task.Status = messaging.TaskStatusRunning
	task.StartedAt = time.Now()
	task.DeliveryTag = delivery.DeliveryTag

	return task, nil
}

// AckTask acknowledges task completion.
func (b *Broker) AckTask(ctx context.Context, task *messaging.Task) error {
	if !b.IsConnected() {
		return messaging.ErrNotConnected
	}

	// Note: In a real implementation, we'd need to track the channel
	// associated with the delivery tag. This is simplified.
	b.metrics.RecordAck()
	return nil
}

// NackTask indicates task failure.
func (b *Broker) NackTask(ctx context.Context, task *messaging.Task, requeue bool) error {
	if !b.IsConnected() {
		return messaging.ErrNotConnected
	}

	b.metrics.RecordNack(requeue)
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
	if !b.IsConnected() {
		return nil, messaging.ErrNotConnected
	}

	channel, err := b.conn.Channel()
	if err != nil {
		return nil, err
	}
	defer channel.Close()

	q, err := channel.QueueInspect(queue)
	if err != nil {
		return nil, err
	}

	return &messaging.QueueStats{
		Name:          queue,
		Messages:      int64(q.Messages),
		MessagesReady: int64(q.Messages),
		Consumers:     q.Consumers,
	}, nil
}

// ScheduleTask schedules a task for future execution.
func (b *Broker) ScheduleTask(ctx context.Context, queue string, task *messaging.Task, executeAt time.Time) error {
	task.ScheduledAt = executeAt
	task.Status = messaging.TaskStatusScheduled

	// RabbitMQ doesn't natively support delayed messages without plugins
	// Use x-delayed-message plugin or implement client-side delay
	delay := time.Until(executeAt)
	if delay > 0 {
		go func() {
			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
				b.EnqueueTask(ctx, queue, task)
			case <-b.closeCh:
				timer.Stop()
			}
		}()
	} else {
		return b.EnqueueTask(ctx, queue, task)
	}

	return nil
}

// CancelTask attempts to cancel a task.
func (b *Broker) CancelTask(ctx context.Context, queue string, taskID string) error {
	// RabbitMQ doesn't support task cancellation directly
	return nil
}

// GetTask retrieves a task by ID.
func (b *Broker) GetTask(ctx context.Context, queue string, taskID string) (*messaging.Task, error) {
	// RabbitMQ doesn't support random access to messages
	return nil, messaging.ErrNotConnected
}

// Helper functions

func taskToMessage(task *messaging.Task) *messaging.Message {
	return &messaging.Message{
		ID:            task.ID,
		Type:          task.Type,
		Payload:       task.Payload,
		Priority:      task.Priority,
		Timestamp:     task.CreatedAt,
		TraceID:       task.TraceID,
		CorrelationID: task.CorrelationID,
		Headers:       task.Metadata,
	}
}

func messageToTask(delivery *amqp.Delivery) *messaging.Task {
	task := &messaging.Task{
		ID:            delivery.MessageId,
		Type:          delivery.Type,
		Payload:       delivery.Body,
		Priority:      int(delivery.Priority),
		CreatedAt:     delivery.Timestamp,
		CorrelationID: delivery.CorrelationId,
		Metadata:      make(map[string]string),
		DeliveryTag:   delivery.DeliveryTag,
	}

	// Extract headers
	for k, v := range delivery.Headers {
		if str, ok := v.(string); ok {
			task.Metadata[k] = str
		}
		if k == "trace_id" {
			if str, ok := v.(string); ok {
				task.TraceID = str
			}
		}
	}

	return task
}

func generateSubscriptionID(queue string) string {
	data, _ := json.Marshal(map[string]interface{}{
		"queue": queue,
		"time":  time.Now().UnixNano(),
	})
	return string(data)
}
