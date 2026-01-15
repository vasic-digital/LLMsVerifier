package rabbitmq

import (
	"context"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"

	"llmsverifier/internal/messaging"
)

// Subscription represents an active subscription to a RabbitMQ queue.
type Subscription struct {
	mu sync.Mutex

	id       string
	queue    string
	handler  messaging.MessageHandler
	options  *messaging.SubscribeOptions
	channel  *amqp.Channel
	conn     *Connection
	metrics  *messaging.BrokerMetrics
	logger   *logrus.Logger
	active   bool
	closeCh  chan struct{}
	prefetch int

	// Consumer tag
	consumerTag string
}

// start begins consuming messages.
func (s *Subscription) start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.consumerTag = "helixagent-" + s.id

	deliveries, err := s.channel.Consume(
		s.queue,
		s.consumerTag,
		s.options.AutoAck,
		s.options.Exclusive,
		false, // no-local
		false, // no-wait
		nil,   // args
	)
	if err != nil {
		return err
	}

	go s.consume(ctx, deliveries)
	return nil
}

// consume processes messages from the delivery channel.
func (s *Subscription) consume(ctx context.Context, deliveries <-chan amqp.Delivery) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.closeCh:
			return
		case delivery, ok := <-deliveries:
			if !ok {
				// Channel closed, subscription ended
				s.logger.WithField("queue", s.queue).Warn("Delivery channel closed")
				return
			}

			s.processDelivery(ctx, &delivery)
		}
	}
}

// processDelivery processes a single delivery.
func (s *Subscription) processDelivery(ctx context.Context, delivery *amqp.Delivery) {
	start := time.Now()

	// Convert to Message
	msg := deliveryToMessage(delivery)

	// Call handler
	err := s.handler(ctx, msg)

	if err != nil {
		s.metrics.RecordFailure()
		s.logger.WithError(err).WithField("message_id", msg.ID).Error("Message handling failed")

		// Nack message if auto-ack is disabled
		if !s.options.AutoAck {
			if s.options.RequeueOnFailure {
				delivery.Nack(false, true)
				s.metrics.RecordNack(true)
			} else {
				delivery.Reject(false)
				s.metrics.RecordNack(false)
			}
		}
	} else {
		// Ack message if auto-ack is disabled
		if !s.options.AutoAck {
			delivery.Ack(false)
			s.metrics.RecordAck()
		}
	}

	s.metrics.RecordConsume(s.queue, int64(len(msg.Payload)), time.Since(start))
}

// restart re-establishes the subscription after reconnection.
func (s *Subscription) restart() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return
	}

	// Close old channel if exists
	if s.channel != nil {
		s.channel.Close()
	}

	// Create new channel
	var err error
	s.channel, err = s.conn.Channel()
	if err != nil {
		s.logger.WithError(err).Error("Failed to create channel during restart")
		return
	}

	// Set QoS
	if err := s.channel.Qos(s.prefetch, 0, false); err != nil {
		s.logger.WithError(err).Error("Failed to set QoS during restart")
		return
	}

	// Start consuming
	ctx := context.Background()
	if err := s.start(ctx); err != nil {
		s.logger.WithError(err).Error("Failed to restart subscription")
	}

	s.logger.WithField("queue", s.queue).Info("Subscription restarted")
}

// Stop stops the subscription.
func (s *Subscription) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return
	}

	s.active = false

	// Signal close
	select {
	case <-s.closeCh:
		// Already closed
	default:
		close(s.closeCh)
	}

	// Cancel consumer
	if s.channel != nil && s.consumerTag != "" {
		s.channel.Cancel(s.consumerTag, false)
		s.channel.Close()
	}

	s.metrics.DecrementSubscriptions()
	s.logger.WithField("queue", s.queue).Info("Subscription stopped")
}

// Unsubscribe stops the subscription.
func (s *Subscription) Unsubscribe() error {
	s.Stop()
	return nil
}

// IsActive returns true if the subscription is active.
func (s *Subscription) IsActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active
}

// Topic returns the subscription queue name.
func (s *Subscription) Topic() string {
	return s.queue
}

// ID returns the subscription ID.
func (s *Subscription) ID() string {
	return s.id
}

// deliveryToMessage converts an AMQP delivery to a Message.
func deliveryToMessage(delivery *amqp.Delivery) *messaging.Message {
	msg := &messaging.Message{
		ID:            delivery.MessageId,
		Type:          delivery.Type,
		Payload:       delivery.Body,
		Priority:      int(delivery.Priority),
		Timestamp:     delivery.Timestamp,
		CorrelationID: delivery.CorrelationId,
		Headers:       make(map[string]string),
		DeliveryTag:   delivery.DeliveryTag,
	}

	// Extract headers
	for k, v := range delivery.Headers {
		if str, ok := v.(string); ok {
			msg.Headers[k] = str
		}
		if k == "trace_id" {
			if str, ok := v.(string); ok {
				msg.TraceID = str
			}
		}
	}

	return msg
}
