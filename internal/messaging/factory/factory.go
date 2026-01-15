// Package factory provides broker factory for creating and managing message broker instances.
package factory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"llmsverifier/internal/messaging"
	"llmsverifier/internal/messaging/inmemory"
	"llmsverifier/internal/messaging/kafka"
	"llmsverifier/internal/messaging/rabbitmq"
)

// BrokerFactory creates and manages message broker instances.
type BrokerFactory struct {
	config      *messaging.MessagingConfig
	logger      *logrus.Logger
	mu          sync.RWMutex
	taskQueue   messaging.TaskQueueBroker
	eventStream messaging.EventStreamBroker
	fallback    messaging.MessageBroker
	initialized bool
}

// NewBrokerFactory creates a new BrokerFactory with the given configuration.
func NewBrokerFactory(config *messaging.MessagingConfig, logger *logrus.Logger) *BrokerFactory {
	if config == nil {
		config = messaging.DefaultMessagingConfig()
	}
	if logger == nil {
		logger = logrus.New()
	}
	return &BrokerFactory{
		config: config,
		logger: logger,
	}
}

// Initialize creates and connects all brokers based on configuration.
func (f *BrokerFactory) Initialize(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.initialized {
		return nil
	}

	f.logger.Info("Initializing message brokers...")

	// Create fallback broker first (always available)
	f.fallback = f.createInMemoryBroker()
	if err := f.fallback.Connect(ctx); err != nil {
		return fmt.Errorf("failed to initialize fallback broker: %w", err)
	}
	f.logger.Info("Fallback broker (in-memory) initialized")

	// If messaging is disabled, only use fallback
	if !f.config.Messaging.Enabled {
		f.logger.Info("Messaging disabled, using in-memory fallback only")
		f.taskQueue = f.fallback.(messaging.TaskQueueBroker)
		f.eventStream = f.fallback.(messaging.EventStreamBroker)
		f.initialized = true
		return nil
	}

	// Try to create and connect RabbitMQ broker for task queue
	if err := f.initializeTaskQueue(ctx); err != nil {
		f.logger.WithError(err).Warn("Failed to initialize RabbitMQ, using fallback")
		if !f.config.Messaging.FallbackEnabled {
			return fmt.Errorf("task queue initialization failed and fallback disabled: %w", err)
		}
		f.taskQueue = f.fallback.(messaging.TaskQueueBroker)
	}

	// Try to create and connect Kafka broker for event stream
	if err := f.initializeEventStream(ctx); err != nil {
		f.logger.WithError(err).Warn("Failed to initialize Kafka, using fallback")
		if !f.config.Messaging.FallbackEnabled {
			return fmt.Errorf("event stream initialization failed and fallback disabled: %w", err)
		}
		f.eventStream = f.fallback.(messaging.EventStreamBroker)
	}

	f.initialized = true
	f.logger.Info("Message brokers initialized successfully")
	return nil
}

// initializeTaskQueue creates and connects the RabbitMQ broker.
func (f *BrokerFactory) initializeTaskQueue(ctx context.Context) error {
	config := f.createRabbitMQConfig()
	broker := rabbitmq.NewBroker(config, f.logger)

	// Create a context with timeout for connection
	connectCtx, cancel := context.WithTimeout(ctx, f.config.Messaging.FallbackTimeout)
	defer cancel()

	if err := broker.Connect(connectCtx); err != nil {
		return fmt.Errorf("rabbitmq connection failed: %w", err)
	}

	f.taskQueue = broker
	f.logger.Info("RabbitMQ task queue broker initialized")
	return nil
}

// initializeEventStream creates and connects the Kafka broker.
func (f *BrokerFactory) initializeEventStream(ctx context.Context) error {
	config := f.createKafkaConfig()
	broker := kafka.NewBroker(config, f.logger)

	// Create a context with timeout for connection
	connectCtx, cancel := context.WithTimeout(ctx, f.config.Messaging.FallbackTimeout)
	defer cancel()

	if err := broker.Connect(connectCtx); err != nil {
		return fmt.Errorf("kafka connection failed: %w", err)
	}

	f.eventStream = broker
	f.logger.Info("Kafka event stream broker initialized")
	return nil
}

// createRabbitMQConfig converts MessagingConfig to RabbitMQ-specific config.
func (f *BrokerFactory) createRabbitMQConfig() *rabbitmq.Config {
	return &rabbitmq.Config{
		Host:              f.config.RabbitMQ.Host,
		Port:              f.config.RabbitMQ.Port,
		Username:          f.config.RabbitMQ.Username,
		Password:          f.config.RabbitMQ.Password,
		VHost:             f.config.RabbitMQ.VHost,
		UseTLS:            f.config.RabbitMQ.UseTLS,
		ConnectionTimeout: f.config.RabbitMQ.ConnectionTimeout,
		HeartbeatInterval: f.config.RabbitMQ.HeartbeatInterval,
		PrefetchCount:     f.config.RabbitMQ.PrefetchCount,
		PrefetchSize:      f.config.RabbitMQ.PrefetchSize,
		PublisherConfirms: f.config.RabbitMQ.PublisherConfirms,
		ConfirmTimeout:    f.config.RabbitMQ.ConfirmTimeout,
	}
}

// createKafkaConfig converts MessagingConfig to Kafka-specific config.
func (f *BrokerFactory) createKafkaConfig() *kafka.Config {
	return &kafka.Config{
		Brokers:            f.config.Kafka.Brokers,
		ClientID:           f.config.Kafka.ClientID,
		GroupID:            f.config.Kafka.GroupID,
		UseTLS:             f.config.Kafka.UseTLS,
		SASLEnabled:        f.config.Kafka.SASLEnabled,
		SASLMechanism:      f.config.Kafka.SASLMechanism,
		SASLUsername:       f.config.Kafka.SASLUsername,
		SASLPassword:       f.config.Kafka.SASLPassword,
		RequiredAcks:       f.config.Kafka.RequiredAcks,
		Idempotent:         f.config.Kafka.Idempotent,
		BatchSize:          f.config.Kafka.BatchSize,
		BatchTimeout:       f.config.Kafka.BatchTimeout,
		CompressionType:    f.config.Kafka.CompressionType,
		FetchMinBytes:      f.config.Kafka.FetchMinBytes,
		FetchMaxBytes:      f.config.Kafka.FetchMaxBytes,
		MaxWaitTime:        f.config.Kafka.MaxWaitTime,
		AutoOffsetReset:    f.config.Kafka.AutoOffsetReset,
		AutoCommit:         f.config.Kafka.AutoCommit,
		AutoCommitInterval: f.config.Kafka.AutoCommitInterval,
		DialTimeout:        f.config.Kafka.DialTimeout,
		ReadTimeout:        f.config.Kafka.ReadTimeout,
		WriteTimeout:       f.config.Kafka.WriteTimeout,
	}
}

// createInMemoryBroker creates a new in-memory broker.
func (f *BrokerFactory) createInMemoryBroker() *inmemory.Broker {
	return inmemory.NewBroker(&inmemory.Config{
		MaxQueueSize:      10000,
		MaxTopicSize:      10000,
		DefaultPartitions: 3,
		MessageTTL:        time.Hour,
	}, f.logger)
}

// TaskQueue returns the task queue broker (RabbitMQ or fallback).
func (f *BrokerFactory) TaskQueue() messaging.TaskQueueBroker {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.taskQueue
}

// EventStream returns the event stream broker (Kafka or fallback).
func (f *BrokerFactory) EventStream() messaging.EventStreamBroker {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.eventStream
}

// Fallback returns the fallback in-memory broker.
func (f *BrokerFactory) Fallback() messaging.MessageBroker {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.fallback
}

// IsInitialized returns true if the factory has been initialized.
func (f *BrokerFactory) IsInitialized() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.initialized
}

// IsUsingFallback returns true if either broker is using the fallback.
func (f *BrokerFactory) IsUsingFallback() bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	if f.taskQueue == nil || f.eventStream == nil {
		return true
	}

	// Check if task queue is using fallback
	if f.taskQueue.BrokerType() == messaging.BrokerTypeInMemory {
		return true
	}

	// Check if event stream is using fallback
	if f.eventStream.BrokerType() == messaging.BrokerTypeInMemory {
		return true
	}

	return false
}

// TaskQueueHealthy returns true if the task queue broker is healthy.
func (f *BrokerFactory) TaskQueueHealthy(ctx context.Context) bool {
	f.mu.RLock()
	tq := f.taskQueue
	f.mu.RUnlock()

	if tq == nil {
		return false
	}
	return tq.HealthCheck(ctx) == nil
}

// EventStreamHealthy returns true if the event stream broker is healthy.
func (f *BrokerFactory) EventStreamHealthy(ctx context.Context) bool {
	f.mu.RLock()
	es := f.eventStream
	f.mu.RUnlock()

	if es == nil {
		return false
	}
	return es.HealthCheck(ctx) == nil
}

// SwitchToFallback switches both brokers to the fallback in-memory broker.
func (f *BrokerFactory) SwitchToFallback(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Close existing brokers if they're not the fallback
	if f.taskQueue != nil && f.taskQueue != f.fallback {
		if closer, ok := f.taskQueue.(interface{ Close(context.Context) error }); ok {
			_ = closer.Close(ctx)
		}
	}
	if f.eventStream != nil && f.eventStream != f.fallback {
		if closer, ok := f.eventStream.(interface{ Close(context.Context) error }); ok {
			_ = closer.Close(ctx)
		}
	}

	f.taskQueue = f.fallback.(messaging.TaskQueueBroker)
	f.eventStream = f.fallback.(messaging.EventStreamBroker)
	f.logger.Warn("Switched to fallback in-memory broker")
	return nil
}

// Reconnect attempts to reconnect to primary brokers.
func (f *BrokerFactory) Reconnect(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.logger.Info("Attempting to reconnect to primary brokers...")

	var taskQueueErr, eventStreamErr error

	// Try to reconnect task queue
	if f.taskQueue == nil || f.taskQueue.BrokerType() == messaging.BrokerTypeInMemory {
		config := f.createRabbitMQConfig()
		broker := rabbitmq.NewBroker(config, f.logger)

		connectCtx, cancel := context.WithTimeout(ctx, f.config.Messaging.FallbackTimeout)
		taskQueueErr = broker.Connect(connectCtx)
		cancel()

		if taskQueueErr == nil {
			f.taskQueue = broker
			f.logger.Info("Reconnected to RabbitMQ")
		}
	}

	// Try to reconnect event stream
	if f.eventStream == nil || f.eventStream.BrokerType() == messaging.BrokerTypeInMemory {
		config := f.createKafkaConfig()
		broker := kafka.NewBroker(config, f.logger)

		connectCtx, cancel := context.WithTimeout(ctx, f.config.Messaging.FallbackTimeout)
		eventStreamErr = broker.Connect(connectCtx)
		cancel()

		if eventStreamErr == nil {
			f.eventStream = broker
			f.logger.Info("Reconnected to Kafka")
		}
	}

	if taskQueueErr != nil && eventStreamErr != nil {
		return fmt.Errorf("reconnection failed: taskqueue=%v, eventstream=%v", taskQueueErr, eventStreamErr)
	}

	return nil
}

// Close closes all broker connections.
func (f *BrokerFactory) Close(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if !f.initialized {
		return nil
	}

	f.logger.Info("Closing message brokers...")

	var errs []error

	// Close task queue if it's not the fallback
	if f.taskQueue != nil && f.taskQueue != f.fallback {
		if closer, ok := f.taskQueue.(interface{ Close(context.Context) error }); ok {
			if err := closer.Close(ctx); err != nil {
				errs = append(errs, fmt.Errorf("task queue close error: %w", err))
			}
		}
	}

	// Close event stream if it's not the fallback
	if f.eventStream != nil && f.eventStream != f.fallback {
		if closer, ok := f.eventStream.(interface{ Close(context.Context) error }); ok {
			if err := closer.Close(ctx); err != nil {
				errs = append(errs, fmt.Errorf("event stream close error: %w", err))
			}
		}
	}

	// Close fallback broker
	if f.fallback != nil {
		if closer, ok := f.fallback.(interface{ Close(context.Context) error }); ok {
			if err := closer.Close(ctx); err != nil {
				errs = append(errs, fmt.Errorf("fallback close error: %w", err))
			}
		}
	}

	f.taskQueue = nil
	f.eventStream = nil
	f.fallback = nil
	f.initialized = false

	if len(errs) > 0 {
		return fmt.Errorf("close errors: %v", errs)
	}

	f.logger.Info("Message brokers closed successfully")
	return nil
}

// GetStatus returns the current status of all brokers.
func (f *BrokerFactory) GetStatus() *FactoryStatus {
	f.mu.RLock()
	defer f.mu.RUnlock()

	status := &FactoryStatus{
		Initialized:   f.initialized,
		UsingFallback: f.isUsingFallbackLocked(),
	}

	if f.taskQueue != nil {
		status.TaskQueue = &BrokerStatus{
			Type:      f.taskQueue.BrokerType(),
			Connected: f.taskQueue.IsConnected(),
		}
	}

	if f.eventStream != nil {
		status.EventStream = &BrokerStatus{
			Type:      f.eventStream.BrokerType(),
			Connected: f.eventStream.IsConnected(),
		}
	}

	if f.fallback != nil {
		status.Fallback = &BrokerStatus{
			Type:      f.fallback.BrokerType(),
			Connected: f.fallback.IsConnected(),
		}
	}

	return status
}

// isUsingFallbackLocked checks fallback usage without locking (caller must hold lock)
func (f *BrokerFactory) isUsingFallbackLocked() bool {
	if f.taskQueue == nil || f.eventStream == nil {
		return true
	}
	if f.taskQueue.BrokerType() == messaging.BrokerTypeInMemory {
		return true
	}
	if f.eventStream.BrokerType() == messaging.BrokerTypeInMemory {
		return true
	}
	return false
}

// FactoryStatus represents the current status of the broker factory.
type FactoryStatus struct {
	Initialized   bool          `json:"initialized"`
	UsingFallback bool          `json:"using_fallback"`
	TaskQueue     *BrokerStatus `json:"task_queue,omitempty"`
	EventStream   *BrokerStatus `json:"event_stream,omitempty"`
	Fallback      *BrokerStatus `json:"fallback,omitempty"`
}

// BrokerStatus represents the status of a single broker.
type BrokerStatus struct {
	Type      messaging.BrokerType `json:"type"`
	Connected bool                 `json:"connected"`
}

// CreateBroker creates a broker of the specified type.
func CreateBroker(brokerType messaging.BrokerType, config interface{}) (messaging.MessageBroker, error) {
	switch brokerType {
	case messaging.BrokerTypeRabbitMQ:
		cfg, ok := config.(*rabbitmq.Config)
		if !ok {
			return nil, fmt.Errorf("invalid config type for RabbitMQ broker")
		}
		return rabbitmq.NewBroker(cfg, nil), nil
	case messaging.BrokerTypeKafka:
		cfg, ok := config.(*kafka.Config)
		if !ok {
			return nil, fmt.Errorf("invalid config type for Kafka broker")
		}
		return kafka.NewBroker(cfg, nil), nil
	case messaging.BrokerTypeInMemory:
		cfg, ok := config.(*inmemory.Config)
		if !ok {
			// Use default config if not provided
			cfg = &inmemory.Config{
				MaxQueueSize:      10000,
				MaxTopicSize:      10000,
				DefaultPartitions: 3,
				MessageTTL:        time.Hour,
			}
		}
		return inmemory.NewBroker(cfg, nil), nil
	default:
		return nil, fmt.Errorf("unknown broker type: %s", brokerType)
	}
}

// CreateTaskQueueBroker creates a task queue broker of the specified type.
func CreateTaskQueueBroker(brokerType messaging.BrokerType, config interface{}) (messaging.TaskQueueBroker, error) {
	broker, err := CreateBroker(brokerType, config)
	if err != nil {
		return nil, err
	}

	tqb, ok := broker.(messaging.TaskQueueBroker)
	if !ok {
		return nil, fmt.Errorf("broker type %s does not implement TaskQueueBroker", brokerType)
	}

	return tqb, nil
}

// CreateEventStreamBroker creates an event stream broker of the specified type.
func CreateEventStreamBroker(brokerType messaging.BrokerType, config interface{}) (messaging.EventStreamBroker, error) {
	broker, err := CreateBroker(brokerType, config)
	if err != nil {
		return nil, err
	}

	esb, ok := broker.(messaging.EventStreamBroker)
	if !ok {
		return nil, fmt.Errorf("broker type %s does not implement EventStreamBroker", brokerType)
	}

	return esb, nil
}

// FromConfigFile creates a BrokerFactory from a configuration file.
func FromConfigFile(path string, logger *logrus.Logger) (*BrokerFactory, error) {
	config, err := messaging.LoadConfig(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return NewBrokerFactory(config, logger), nil
}

// FromConfigBytes creates a BrokerFactory from configuration bytes.
func FromConfigBytes(data []byte, logger *logrus.Logger) (*BrokerFactory, error) {
	config, err := messaging.LoadConfigFromBytes(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return NewBrokerFactory(config, logger), nil
}
