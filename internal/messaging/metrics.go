package messaging

import (
	"sync"
	"sync/atomic"
	"time"
)

// BrokerMetrics contains metrics for a message broker.
type BrokerMetrics struct {
	mu sync.RWMutex

	// Connection metrics
	Connected      bool      `json:"connected"`
	ConnectedSince time.Time `json:"connected_since,omitempty"`
	Reconnections  int64     `json:"reconnections"`

	// Message metrics
	MessagesPublished   int64 `json:"messages_published"`
	MessagesConsumed    int64 `json:"messages_consumed"`
	MessagesFailed      int64 `json:"messages_failed"`
	MessagesAcked       int64 `json:"messages_acked"`
	MessagesNacked      int64 `json:"messages_nacked"`
	MessagesRequeued    int64 `json:"messages_requeued"`
	MessagesDeadLetter  int64 `json:"messages_dead_letter"`

	// Byte metrics
	BytesPublished int64 `json:"bytes_published"`
	BytesConsumed  int64 `json:"bytes_consumed"`

	// Latency metrics (nanoseconds for precision)
	PublishLatencyTotal  int64 `json:"publish_latency_total_ns"`
	PublishLatencyCount  int64 `json:"publish_latency_count"`
	ConsumeLatencyTotal  int64 `json:"consume_latency_total_ns"`
	ConsumeLatencyCount  int64 `json:"consume_latency_count"`

	// Error metrics
	ConnectionErrors int64 `json:"connection_errors"`
	PublishErrors    int64 `json:"publish_errors"`
	ConsumeErrors    int64 `json:"consume_errors"`
	AckErrors        int64 `json:"ack_errors"`
	RetryCount       int64 `json:"retry_count"`

	// Queue/Topic metrics
	ActiveSubscriptions int64            `json:"active_subscriptions"`
	QueueDepths         map[string]int64 `json:"queue_depths,omitempty"`

	// Per-topic metrics
	TopicMetrics map[string]*TopicMetrics `json:"topic_metrics,omitempty"`
}

// TopicMetrics contains metrics for a specific topic/queue.
type TopicMetrics struct {
	Name             string `json:"name"`
	MessagesIn       int64  `json:"messages_in"`
	MessagesOut      int64  `json:"messages_out"`
	BytesIn          int64  `json:"bytes_in"`
	BytesOut         int64  `json:"bytes_out"`
	ConsumerCount    int32  `json:"consumer_count"`
	LastPublishTime  int64  `json:"last_publish_time"`
	LastConsumeTime  int64  `json:"last_consume_time"`
}

// NewBrokerMetrics creates a new BrokerMetrics instance.
func NewBrokerMetrics() *BrokerMetrics {
	return &BrokerMetrics{
		QueueDepths:  make(map[string]int64),
		TopicMetrics: make(map[string]*TopicMetrics),
	}
}

// SetConnected updates the connection status.
func (m *BrokerMetrics) SetConnected(connected bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Connected = connected
	if connected && m.ConnectedSince.IsZero() {
		m.ConnectedSince = time.Now()
	} else if !connected {
		m.ConnectedSince = time.Time{}
	}
}

// IncrementReconnections increments the reconnection counter.
func (m *BrokerMetrics) IncrementReconnections() {
	atomic.AddInt64(&m.Reconnections, 1)
}

// RecordPublish records a published message.
func (m *BrokerMetrics) RecordPublish(topic string, bytes int64, latency time.Duration) {
	atomic.AddInt64(&m.MessagesPublished, 1)
	atomic.AddInt64(&m.BytesPublished, bytes)
	atomic.AddInt64(&m.PublishLatencyTotal, int64(latency))
	atomic.AddInt64(&m.PublishLatencyCount, 1)

	m.recordTopicPublish(topic, bytes)
}

// RecordConsume records a consumed message.
func (m *BrokerMetrics) RecordConsume(topic string, bytes int64, latency time.Duration) {
	atomic.AddInt64(&m.MessagesConsumed, 1)
	atomic.AddInt64(&m.BytesConsumed, bytes)
	atomic.AddInt64(&m.ConsumeLatencyTotal, int64(latency))
	atomic.AddInt64(&m.ConsumeLatencyCount, 1)

	m.recordTopicConsume(topic, bytes)
}

// RecordAck records a message acknowledgment.
func (m *BrokerMetrics) RecordAck() {
	atomic.AddInt64(&m.MessagesAcked, 1)
}

// RecordNack records a message negative acknowledgment.
func (m *BrokerMetrics) RecordNack(requeued bool) {
	atomic.AddInt64(&m.MessagesNacked, 1)
	if requeued {
		atomic.AddInt64(&m.MessagesRequeued, 1)
	}
}

// RecordDeadLetter records a message sent to dead letter queue.
func (m *BrokerMetrics) RecordDeadLetter() {
	atomic.AddInt64(&m.MessagesDeadLetter, 1)
}

// RecordFailure records a failed message.
func (m *BrokerMetrics) RecordFailure() {
	atomic.AddInt64(&m.MessagesFailed, 1)
}

// RecordConnectionError records a connection error.
func (m *BrokerMetrics) RecordConnectionError() {
	atomic.AddInt64(&m.ConnectionErrors, 1)
}

// RecordPublishError records a publish error.
func (m *BrokerMetrics) RecordPublishError() {
	atomic.AddInt64(&m.PublishErrors, 1)
}

// RecordConsumeError records a consume error.
func (m *BrokerMetrics) RecordConsumeError() {
	atomic.AddInt64(&m.ConsumeErrors, 1)
}

// RecordAckError records an ack error.
func (m *BrokerMetrics) RecordAckError() {
	atomic.AddInt64(&m.AckErrors, 1)
}

// RecordRetry records a retry attempt.
func (m *BrokerMetrics) RecordRetry() {
	atomic.AddInt64(&m.RetryCount, 1)
}

// IncrementSubscriptions increments the active subscription count.
func (m *BrokerMetrics) IncrementSubscriptions() {
	atomic.AddInt64(&m.ActiveSubscriptions, 1)
}

// DecrementSubscriptions decrements the active subscription count.
func (m *BrokerMetrics) DecrementSubscriptions() {
	atomic.AddInt64(&m.ActiveSubscriptions, -1)
}

// SetQueueDepth updates the depth of a specific queue.
func (m *BrokerMetrics) SetQueueDepth(queue string, depth int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.QueueDepths == nil {
		m.QueueDepths = make(map[string]int64)
	}
	m.QueueDepths[queue] = depth
}

// GetAveragePublishLatency returns the average publish latency.
func (m *BrokerMetrics) GetAveragePublishLatency() time.Duration {
	count := atomic.LoadInt64(&m.PublishLatencyCount)
	if count == 0 {
		return 0
	}
	total := atomic.LoadInt64(&m.PublishLatencyTotal)
	return time.Duration(total / count)
}

// GetAverageConsumeLatency returns the average consume latency.
func (m *BrokerMetrics) GetAverageConsumeLatency() time.Duration {
	count := atomic.LoadInt64(&m.ConsumeLatencyCount)
	if count == 0 {
		return 0
	}
	total := atomic.LoadInt64(&m.ConsumeLatencyTotal)
	return time.Duration(total / count)
}

// GetThroughput returns messages per second (published + consumed).
func (m *BrokerMetrics) GetThroughput(duration time.Duration) float64 {
	if duration <= 0 {
		return 0
	}
	published := atomic.LoadInt64(&m.MessagesPublished)
	consumed := atomic.LoadInt64(&m.MessagesConsumed)
	return float64(published+consumed) / duration.Seconds()
}

// GetErrorRate returns the error rate as a percentage.
func (m *BrokerMetrics) GetErrorRate() float64 {
	total := atomic.LoadInt64(&m.MessagesPublished) + atomic.LoadInt64(&m.MessagesConsumed)
	if total == 0 {
		return 0
	}
	errors := atomic.LoadInt64(&m.PublishErrors) + atomic.LoadInt64(&m.ConsumeErrors)
	return float64(errors) / float64(total) * 100
}

// Snapshot returns a copy of the metrics.
func (m *BrokerMetrics) Snapshot() *BrokerMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := &BrokerMetrics{
		Connected:           m.Connected,
		ConnectedSince:      m.ConnectedSince,
		Reconnections:       atomic.LoadInt64(&m.Reconnections),
		MessagesPublished:   atomic.LoadInt64(&m.MessagesPublished),
		MessagesConsumed:    atomic.LoadInt64(&m.MessagesConsumed),
		MessagesFailed:      atomic.LoadInt64(&m.MessagesFailed),
		MessagesAcked:       atomic.LoadInt64(&m.MessagesAcked),
		MessagesNacked:      atomic.LoadInt64(&m.MessagesNacked),
		MessagesRequeued:    atomic.LoadInt64(&m.MessagesRequeued),
		MessagesDeadLetter:  atomic.LoadInt64(&m.MessagesDeadLetter),
		BytesPublished:      atomic.LoadInt64(&m.BytesPublished),
		BytesConsumed:       atomic.LoadInt64(&m.BytesConsumed),
		PublishLatencyTotal: atomic.LoadInt64(&m.PublishLatencyTotal),
		PublishLatencyCount: atomic.LoadInt64(&m.PublishLatencyCount),
		ConsumeLatencyTotal: atomic.LoadInt64(&m.ConsumeLatencyTotal),
		ConsumeLatencyCount: atomic.LoadInt64(&m.ConsumeLatencyCount),
		ConnectionErrors:    atomic.LoadInt64(&m.ConnectionErrors),
		PublishErrors:       atomic.LoadInt64(&m.PublishErrors),
		ConsumeErrors:       atomic.LoadInt64(&m.ConsumeErrors),
		AckErrors:           atomic.LoadInt64(&m.AckErrors),
		RetryCount:          atomic.LoadInt64(&m.RetryCount),
		ActiveSubscriptions: atomic.LoadInt64(&m.ActiveSubscriptions),
	}

	// Copy maps
	if m.QueueDepths != nil {
		snapshot.QueueDepths = make(map[string]int64)
		for k, v := range m.QueueDepths {
			snapshot.QueueDepths[k] = v
		}
	}

	if m.TopicMetrics != nil {
		snapshot.TopicMetrics = make(map[string]*TopicMetrics)
		for k, v := range m.TopicMetrics {
			snapshot.TopicMetrics[k] = &TopicMetrics{
				Name:            v.Name,
				MessagesIn:      atomic.LoadInt64(&v.MessagesIn),
				MessagesOut:     atomic.LoadInt64(&v.MessagesOut),
				BytesIn:         atomic.LoadInt64(&v.BytesIn),
				BytesOut:        atomic.LoadInt64(&v.BytesOut),
				ConsumerCount:   atomic.LoadInt32(&v.ConsumerCount),
				LastPublishTime: atomic.LoadInt64(&v.LastPublishTime),
				LastConsumeTime: atomic.LoadInt64(&v.LastConsumeTime),
			}
		}
	}

	return snapshot
}

// Reset resets all metrics to zero.
func (m *BrokerMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	atomic.StoreInt64(&m.Reconnections, 0)
	atomic.StoreInt64(&m.MessagesPublished, 0)
	atomic.StoreInt64(&m.MessagesConsumed, 0)
	atomic.StoreInt64(&m.MessagesFailed, 0)
	atomic.StoreInt64(&m.MessagesAcked, 0)
	atomic.StoreInt64(&m.MessagesNacked, 0)
	atomic.StoreInt64(&m.MessagesRequeued, 0)
	atomic.StoreInt64(&m.MessagesDeadLetter, 0)
	atomic.StoreInt64(&m.BytesPublished, 0)
	atomic.StoreInt64(&m.BytesConsumed, 0)
	atomic.StoreInt64(&m.PublishLatencyTotal, 0)
	atomic.StoreInt64(&m.PublishLatencyCount, 0)
	atomic.StoreInt64(&m.ConsumeLatencyTotal, 0)
	atomic.StoreInt64(&m.ConsumeLatencyCount, 0)
	atomic.StoreInt64(&m.ConnectionErrors, 0)
	atomic.StoreInt64(&m.PublishErrors, 0)
	atomic.StoreInt64(&m.ConsumeErrors, 0)
	atomic.StoreInt64(&m.AckErrors, 0)
	atomic.StoreInt64(&m.RetryCount, 0)
	atomic.StoreInt64(&m.ActiveSubscriptions, 0)

	m.QueueDepths = make(map[string]int64)
	m.TopicMetrics = make(map[string]*TopicMetrics)
}

// recordTopicPublish records a publish to a specific topic.
func (m *BrokerMetrics) recordTopicPublish(topic string, bytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.TopicMetrics == nil {
		m.TopicMetrics = make(map[string]*TopicMetrics)
	}

	tm, ok := m.TopicMetrics[topic]
	if !ok {
		tm = &TopicMetrics{Name: topic}
		m.TopicMetrics[topic] = tm
	}

	atomic.AddInt64(&tm.MessagesIn, 1)
	atomic.AddInt64(&tm.BytesIn, bytes)
	atomic.StoreInt64(&tm.LastPublishTime, time.Now().UnixNano())
}

// recordTopicConsume records consumption from a specific topic.
func (m *BrokerMetrics) recordTopicConsume(topic string, bytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.TopicMetrics == nil {
		m.TopicMetrics = make(map[string]*TopicMetrics)
	}

	tm, ok := m.TopicMetrics[topic]
	if !ok {
		tm = &TopicMetrics{Name: topic}
		m.TopicMetrics[topic] = tm
	}

	atomic.AddInt64(&tm.MessagesOut, 1)
	atomic.AddInt64(&tm.BytesOut, bytes)
	atomic.StoreInt64(&tm.LastConsumeTime, time.Now().UnixNano())
}
