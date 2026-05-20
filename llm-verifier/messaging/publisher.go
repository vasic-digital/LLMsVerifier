// Package messaging provides message broker integration for LLMsVerifier.
package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// MessageBroker defines the interface for message brokers.
type MessageBroker interface {
	// Connect establishes a connection to the broker.
	Connect(ctx context.Context) error

	// Close closes the connection to the broker.
	Close(ctx context.Context) error

	// Publish publishes a message to a topic.
	Publish(ctx context.Context, topic string, message []byte) error

	// IsConnected returns true if the broker is connected.
	IsConnected() bool

	// BrokerType returns the type of broker.
	BrokerType() BrokerType
}

// Publisher publishes verification events to message brokers.
type Publisher struct {
	config      *Config
	broker      MessageBroker
	buffer      chan *publishRequest
	stopCh      chan struct{}
	wg          sync.WaitGroup
	mu          sync.RWMutex
	connected   bool
	metrics     *PublisherMetrics
	logger      Logger
}

// publishRequest represents a request to publish an event.
type publishRequest struct {
	event   *VerificationEvent
	topic   string
	errCh   chan error
	ctx     context.Context
}

// PublisherMetrics holds metrics for the publisher.
type PublisherMetrics struct {
	EventsPublished    int64
	EventsFailed       int64
	EventsBuffered     int64
	EventsDropped      int64
	PublishLatencySum  time.Duration
	PublishLatencyCount int64
	mu                 sync.RWMutex
}

// Logger interface for logging.
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// defaultLogger is a simple logger that uses the standard log package.
type defaultLogger struct{}

func (l *defaultLogger) Info(msg string, args ...interface{})  { log.Printf("[INFO] "+msg, args...) }
func (l *defaultLogger) Error(msg string, args ...interface{}) { log.Printf("[ERROR] "+msg, args...) }
func (l *defaultLogger) Warn(msg string, args ...interface{})  { log.Printf("[WARN] "+msg, args...) }
func (l *defaultLogger) Debug(msg string, args ...interface{}) { log.Printf("[DEBUG] "+msg, args...) }

// NewPublisher creates a new event publisher.
func NewPublisher(config *Config) *Publisher {
	if config == nil {
		config = DefaultConfig()
	}

	p := &Publisher{
		config:  config,
		buffer:  make(chan *publishRequest, config.BufferSize),
		stopCh:  make(chan struct{}),
		metrics: &PublisherMetrics{},
		logger:  &defaultLogger{},
	}

	return p
}

// SetLogger sets the logger for the publisher.
func (p *Publisher) SetLogger(logger Logger) {
	p.logger = logger
}

// SetBroker sets the message broker for the publisher.
func (p *Publisher) SetBroker(broker MessageBroker) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.broker = broker
}

// Start starts the publisher.
func (p *Publisher) Start(ctx context.Context) error {
	if !p.config.Enabled {
		p.logger.Info("Messaging is disabled, publisher not started")
		return nil
	}

	if p.broker == nil {
		return errors.New("message broker not set")
	}

	// Connect to broker
	if err := p.broker.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to broker: %w", err)
	}

	p.mu.Lock()
	p.connected = true
	p.mu.Unlock()

	// Start async publish worker if enabled
	if p.config.AsyncPublish {
		p.wg.Add(1)
		go p.publishWorker(ctx)
	}

	p.logger.Info("Publisher started successfully")
	return nil
}

// Stop stops the publisher.
func (p *Publisher) Stop(ctx context.Context) error {
	p.mu.Lock()
	if !p.connected {
		p.mu.Unlock()
		return nil
	}
	p.connected = false
	p.mu.Unlock()

	// Signal workers to stop
	close(p.stopCh)

	// Wait for workers to finish
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Workers finished
	case <-ctx.Done():
		p.logger.Warn("Timeout waiting for workers to stop")
	}

	// Close broker connection
	if p.broker != nil {
		if err := p.broker.Close(ctx); err != nil {
			return fmt.Errorf("failed to close broker: %w", err)
		}
	}

	p.logger.Info("Publisher stopped successfully")
	return nil
}

// publishWorker processes the publish buffer.
func (p *Publisher) publishWorker(ctx context.Context) {
	defer p.wg.Done()

	for {
		select {
		case <-p.stopCh:
			// Drain remaining messages
			p.drainBuffer(ctx)
			return
		case <-ctx.Done():
			return
		case req := <-p.buffer:
			p.processPublishRequest(req)
		}
	}
}

// drainBuffer drains remaining messages from the buffer.
func (p *Publisher) drainBuffer(ctx context.Context) {
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case req := <-p.buffer:
			p.processPublishRequest(req)
		case <-timeout.C:
			return
		default:
			return
		}
	}
}

// processPublishRequest processes a single publish request.
func (p *Publisher) processPublishRequest(req *publishRequest) {
	start := time.Now()
	err := p.doPublish(req.ctx, req.event, req.topic)
	latency := time.Since(start)

	p.metrics.mu.Lock()
	p.metrics.PublishLatencySum += latency
	p.metrics.PublishLatencyCount++
	if err != nil {
		p.metrics.EventsFailed++
	} else {
		p.metrics.EventsPublished++
	}
	p.metrics.mu.Unlock()

	if req.errCh != nil {
		select {
		case req.errCh <- err:
		default:
		}
	}
}

// Publish publishes an event to the message broker.
func (p *Publisher) Publish(ctx context.Context, event *VerificationEvent) error {
	return p.PublishToTopic(ctx, event, p.getTopicForEvent(event))
}

// PublishToTopic publishes an event to a specific topic.
func (p *Publisher) PublishToTopic(ctx context.Context, event *VerificationEvent, topic string) error {
	if !p.config.Enabled {
		return nil
	}

	p.mu.RLock()
	if !p.connected {
		p.mu.RUnlock()
		return errors.New("publisher not connected")
	}
	p.mu.RUnlock()

	if p.config.AsyncPublish {
		return p.asyncPublish(ctx, event, topic)
	}
	return p.doPublish(ctx, event, topic)
}

// asyncPublish adds an event to the publish buffer.
func (p *Publisher) asyncPublish(ctx context.Context, event *VerificationEvent, topic string) error {
	req := &publishRequest{
		event: event,
		topic: topic,
		ctx:   ctx,
	}

	select {
	case p.buffer <- req:
		p.metrics.mu.Lock()
		p.metrics.EventsBuffered++
		p.metrics.mu.Unlock()
		return nil
	default:
		p.metrics.mu.Lock()
		p.metrics.EventsDropped++
		p.metrics.mu.Unlock()
		return errors.New("publish buffer full")
	}
}

// doPublish performs the actual publish operation.
func (p *Publisher) doPublish(ctx context.Context, event *VerificationEvent, topic string) error {
	start := time.Now()

	// Serialize event
	data, err := json.Marshal(event)
	if err != nil {
		p.metrics.mu.Lock()
		p.metrics.EventsFailed++
		p.metrics.mu.Unlock()
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	// Publish with retries
	var lastErr error
	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(p.config.RetryDelay)
		}

		publishCtx, cancel := context.WithTimeout(ctx, p.config.PublishTimeout)
		err = p.broker.Publish(publishCtx, topic, data)
		cancel()

		if err == nil {
			latency := time.Since(start)
			p.metrics.mu.Lock()
			p.metrics.EventsPublished++
			p.metrics.PublishLatencySum += latency
			p.metrics.PublishLatencyCount++
			p.metrics.mu.Unlock()
			return nil
		}
		lastErr = err

		if !p.config.RetryOnError {
			break
		}

		p.logger.Warn("Publish attempt %d failed: %v", attempt+1, err)
	}

	p.metrics.mu.Lock()
	p.metrics.EventsFailed++
	p.metrics.mu.Unlock()
	return fmt.Errorf("failed to publish event after %d attempts: %w", p.config.MaxRetries+1, lastErr)
}

// getTopicForEvent returns the appropriate topic for an event.
func (p *Publisher) getTopicForEvent(event *VerificationEvent) string {
	switch p.config.BrokerType {
	case BrokerTypeKafka:
		return p.getKafkaTopicForEvent(event)
	case BrokerTypeRabbitMQ:
		return p.getRabbitMQRoutingKeyForEvent(event)
	default:
		return "llmsverifier.events"
	}
}

// getKafkaTopicForEvent returns the Kafka topic for an event.
func (p *Publisher) getKafkaTopicForEvent(event *VerificationEvent) string {
	switch {
	case event.Type == EventVerificationStarted ||
		event.Type == EventVerificationCompleted ||
		event.Type == EventVerificationFailed:
		return "llmsverifier.events.verification"
	case event.Type == EventProviderDiscovered ||
		event.Type == EventProviderScored ||
		event.Type == EventProviderHealthCheck ||
		event.Type == EventProviderFailed:
		return "llmsverifier.events.provider"
	case event.Type == EventModelVerified ||
		event.Type == EventModelRanked ||
		event.Type == EventModelFailed:
		return "llmsverifier.events.model"
	case event.Type == EventTeamSelected ||
		event.Type == EventTeamMemberFailed:
		return "llmsverifier.events.team"
	default:
		return "llmsverifier.events.system"
	}
}

// getRabbitMQRoutingKeyForEvent returns the RabbitMQ routing key for an event.
func (p *Publisher) getRabbitMQRoutingKeyForEvent(event *VerificationEvent) string {
	return string(event.Type)
}

// PublishVerificationStarted publishes a verification started event.
func (p *Publisher) PublishVerificationStarted(ctx context.Context, modelCount, providerCount int) error {
	event := NewVerificationEvent(
		EventVerificationStarted,
		SeverityInfo,
		tr("messaging.verification.started.title"),
		trData("messaging.verification.started.message", map[string]any{
			"model_count":    modelCount,
			"provider_count": providerCount,
		}),
	).AddDetail("model_count", modelCount).
		AddDetail("provider_count", providerCount).
		AddDetail("start_time", time.Now().UTC())

	return p.Publish(ctx, event)
}

// PublishVerificationCompleted publishes a verification completed event.
func (p *Publisher) PublishVerificationCompleted(ctx context.Context, duration time.Duration, successCount, failureCount int) error {
	event := NewVerificationEvent(
		EventVerificationCompleted,
		SeverityInfo,
		tr("messaging.verification.completed.title"),
		trData("messaging.verification.completed.message", map[string]any{
			"duration":      duration.String(),
			"success_count": successCount,
			"failure_count": failureCount,
		}),
	).AddDetail("duration_seconds", duration.Seconds()).
		AddDetail("success_count", successCount).
		AddDetail("failure_count", failureCount).
		AddDetail("completion_time", time.Now().UTC())

	return p.Publish(ctx, event)
}

// PublishVerificationFailed publishes a verification failed event.
func (p *Publisher) PublishVerificationFailed(ctx context.Context, errorMsg string) error {
	event := NewVerificationEvent(
		EventVerificationFailed,
		SeverityError,
		tr("messaging.verification.failed.title"),
		trData("messaging.verification.failed.message", map[string]any{
			"error": errorMsg,
		}),
	).AddDetail("error_message", errorMsg).
		AddDetail("failure_time", time.Now().UTC())

	return p.Publish(ctx, event)
}

// PublishProviderScored publishes a provider scored event.
func (p *Publisher) PublishProviderScored(ctx context.Context, scored *ProviderScoredEvent) error {
	event := NewVerificationEvent(
		EventProviderScored,
		SeverityInfo,
		tr("messaging.provider.scored.title"),
		trData("messaging.provider.scored.message", map[string]any{
			"provider": scored.ProviderName,
			"score":    fmt.Sprintf("%.2f", scored.OverallScore),
		}),
	).WithProvider(scored.ProviderID).
		WithScore(scored.OverallScore).
		AddDetail("response_speed_score", scored.ResponseSpeed).
		AddDetail("model_efficiency_score", scored.ModelEfficiency).
		AddDetail("cost_effectiveness_score", scored.CostEffectiveness).
		AddDetail("capability_score", scored.Capability).
		AddDetail("recency_score", scored.Recency)

	return p.Publish(ctx, event)
}

// PublishProviderHealthCheck publishes a provider health check event.
func (p *Publisher) PublishProviderHealthCheck(ctx context.Context, health *ProviderHealthEvent) error {
	severity := SeverityInfo
	if health.Status == "unhealthy" {
		severity = SeverityError
	} else if health.Status == "degraded" {
		severity = SeverityWarning
	}

	event := NewVerificationEvent(
		EventProviderHealthCheck,
		severity,
		tr("messaging.provider.health.title"),
		trData("messaging.provider.health.message", map[string]any{
			"provider": health.ProviderName,
			"status":   health.Status,
		}),
	).WithProvider(health.ProviderID).
		AddDetail("status", health.Status).
		AddDetail("latency_ms", health.Latency)

	if health.ErrorMessage != "" {
		event.AddDetail("error_message", health.ErrorMessage)
	}

	return p.Publish(ctx, event)
}

// PublishModelRanked publishes a model ranked event.
func (p *Publisher) PublishModelRanked(ctx context.Context, ranked *ModelRankedEvent) error {
	event := NewVerificationEvent(
		EventModelRanked,
		SeverityInfo,
		tr("messaging.model.ranked.title"),
		trData("messaging.model.ranked.message", map[string]any{
			"model": ranked.ModelName,
			"rank":  ranked.Rank,
			"score": fmt.Sprintf("%.2f", ranked.Score),
		}),
	).WithProvider(ranked.ProviderID).
		WithModel(ranked.ModelID).
		WithScore(ranked.Score).
		AddDetail("rank", ranked.Rank).
		AddDetail("prev_rank", ranked.PrevRank).
		AddDetail("prev_score", ranked.PrevScore)

	return p.Publish(ctx, event)
}

// PublishTeamSelected publishes a team selected event.
func (p *Publisher) PublishTeamSelected(ctx context.Context, team *TeamSelectedEvent) error {
	event := NewVerificationEvent(
		EventTeamSelected,
		SeverityInfo,
		tr("messaging.team.selected.title"),
		trData("messaging.team.selected.message", map[string]any{
			"team_size": team.TeamSize,
			"criteria":  team.SelectionCriteria,
		}),
	).AddDetail("team_id", team.TeamID).
		AddDetail("team_size", team.TeamSize).
		AddDetail("primary_llms", team.PrimaryLLMs).
		AddDetail("fallback_llms", team.FallbackLLMs).
		AddDetail("selection_criteria", team.SelectionCriteria)

	return p.Publish(ctx, event)
}

// GetMetrics returns the publisher metrics.
func (p *Publisher) GetMetrics() *PublisherMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()

	return &PublisherMetrics{
		EventsPublished:     p.metrics.EventsPublished,
		EventsFailed:        p.metrics.EventsFailed,
		EventsBuffered:      p.metrics.EventsBuffered,
		EventsDropped:       p.metrics.EventsDropped,
		PublishLatencySum:   p.metrics.PublishLatencySum,
		PublishLatencyCount: p.metrics.PublishLatencyCount,
	}
}

// IsConnected returns true if the publisher is connected.
func (p *Publisher) IsConnected() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.connected
}
