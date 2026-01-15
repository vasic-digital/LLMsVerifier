package kafka

import (
	"context"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"

	"llmsverifier/internal/messaging"
)

// Subscription represents an active subscription to a Kafka topic.
type Subscription struct {
	mu sync.Mutex

	id       string
	topic    string
	groupID  string
	handler  messaging.MessageHandler
	reader   *kafka.Reader
	metrics  *messaging.BrokerMetrics
	logger   *logrus.Logger
	active   bool
	closeCh  chan struct{}
}

// consume reads messages from Kafka and processes them.
func (s *Subscription) consume(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.closeCh:
			return
		default:
			msg, err := s.reader.FetchMessage(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				// Log error and continue
				s.logger.WithError(err).Debug("Error fetching message")
				time.Sleep(100 * time.Millisecond)
				continue
			}

			s.processMessage(ctx, &msg)
		}
	}
}

// processMessage processes a single message.
func (s *Subscription) processMessage(ctx context.Context, msg *kafka.Message) {
	start := time.Now()

	// Convert to messaging.Message
	message := kafkaToMessage(msg)

	// Call handler
	err := s.handler(ctx, message)

	if err != nil {
		s.metrics.RecordFailure()
		s.logger.WithError(err).WithField("offset", msg.Offset).Error("Message handling failed")
	} else {
		// Commit offset on success
		if err := s.reader.CommitMessages(ctx, *msg); err != nil {
			s.logger.WithError(err).Error("Failed to commit offset")
		}
		s.metrics.RecordAck()
	}

	s.metrics.RecordConsume(s.topic, int64(len(msg.Value)), time.Since(start))
}

// Unsubscribe stops the subscription.
func (s *Subscription) Unsubscribe() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.active {
		return nil
	}

	s.active = false

	select {
	case <-s.closeCh:
	default:
		close(s.closeCh)
	}

	s.metrics.DecrementSubscriptions()
	s.logger.WithField("topic", s.topic).Info("Subscription stopped")
	return nil
}

// IsActive returns true if the subscription is active.
func (s *Subscription) IsActive() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active
}

// Topic returns the subscription topic.
func (s *Subscription) Topic() string {
	return s.topic
}

// ID returns the subscription ID.
func (s *Subscription) ID() string {
	return s.id
}

// kafkaToMessage converts a Kafka message to a messaging.Message.
func kafkaToMessage(msg *kafka.Message) *messaging.Message {
	message := &messaging.Message{
		Payload:   msg.Value,
		Key:       string(msg.Key),
		Timestamp: msg.Time,
		Partition: int32(msg.Partition),
		Offset:    msg.Offset,
		Headers:   make(map[string]string),
	}

	for _, h := range msg.Headers {
		switch h.Key {
		case "id":
			message.ID = string(h.Value)
		case "type":
			message.Type = string(h.Value)
		case "trace_id":
			message.TraceID = string(h.Value)
		case "correlation_id":
			message.CorrelationID = string(h.Value)
		default:
			message.Headers[h.Key] = string(h.Value)
		}
	}

	return message
}
