package messaging

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewBrokerMetrics(t *testing.T) {
	metrics := NewBrokerMetrics()

	assert.NotNil(t, metrics)
	assert.NotNil(t, metrics.QueueDepths)
	assert.NotNil(t, metrics.TopicMetrics)
	assert.False(t, metrics.Connected)
}

func TestBrokerMetricsSetConnected(t *testing.T) {
	metrics := NewBrokerMetrics()

	t.Run("set connected", func(t *testing.T) {
		metrics.SetConnected(true)
		assert.True(t, metrics.Connected)
		assert.False(t, metrics.ConnectedSince.IsZero())
	})

	t.Run("set disconnected", func(t *testing.T) {
		metrics.SetConnected(false)
		assert.False(t, metrics.Connected)
		assert.True(t, metrics.ConnectedSince.IsZero())
	})
}

func TestBrokerMetricsRecordPublish(t *testing.T) {
	metrics := NewBrokerMetrics()

	metrics.RecordPublish("test-topic", 100, 50*time.Millisecond)
	metrics.RecordPublish("test-topic", 200, 30*time.Millisecond)

	assert.Equal(t, int64(2), metrics.MessagesPublished)
	assert.Equal(t, int64(300), metrics.BytesPublished)
	assert.Equal(t, int64(2), metrics.PublishLatencyCount)
}

func TestBrokerMetricsRecordConsume(t *testing.T) {
	metrics := NewBrokerMetrics()

	metrics.RecordConsume("test-topic", 150, 40*time.Millisecond)

	assert.Equal(t, int64(1), metrics.MessagesConsumed)
	assert.Equal(t, int64(150), metrics.BytesConsumed)
}

func TestBrokerMetricsAckNack(t *testing.T) {
	metrics := NewBrokerMetrics()

	metrics.RecordAck()
	metrics.RecordAck()
	assert.Equal(t, int64(2), metrics.MessagesAcked)

	metrics.RecordNack(false)
	assert.Equal(t, int64(1), metrics.MessagesNacked)
	assert.Equal(t, int64(0), metrics.MessagesRequeued)

	metrics.RecordNack(true)
	assert.Equal(t, int64(2), metrics.MessagesNacked)
	assert.Equal(t, int64(1), metrics.MessagesRequeued)
}

func TestBrokerMetricsRecordDeadLetter(t *testing.T) {
	metrics := NewBrokerMetrics()

	metrics.RecordDeadLetter()
	metrics.RecordDeadLetter()

	assert.Equal(t, int64(2), metrics.MessagesDeadLetter)
}

func TestBrokerMetricsRecordFailure(t *testing.T) {
	metrics := NewBrokerMetrics()

	metrics.RecordFailure()
	assert.Equal(t, int64(1), metrics.MessagesFailed)
}

func TestBrokerMetricsRecordErrors(t *testing.T) {
	metrics := NewBrokerMetrics()

	metrics.RecordConnectionError()
	metrics.RecordPublishError()
	metrics.RecordConsumeError()
	metrics.RecordAckError()
	metrics.RecordRetry()

	assert.Equal(t, int64(1), metrics.ConnectionErrors)
	assert.Equal(t, int64(1), metrics.PublishErrors)
	assert.Equal(t, int64(1), metrics.ConsumeErrors)
	assert.Equal(t, int64(1), metrics.AckErrors)
	assert.Equal(t, int64(1), metrics.RetryCount)
}

func TestBrokerMetricsSubscriptions(t *testing.T) {
	metrics := NewBrokerMetrics()

	metrics.IncrementSubscriptions()
	metrics.IncrementSubscriptions()
	assert.Equal(t, int64(2), metrics.ActiveSubscriptions)

	metrics.DecrementSubscriptions()
	assert.Equal(t, int64(1), metrics.ActiveSubscriptions)
}

func TestBrokerMetricsSetQueueDepth(t *testing.T) {
	metrics := NewBrokerMetrics()

	metrics.SetQueueDepth("queue-1", 100)
	metrics.SetQueueDepth("queue-2", 200)

	assert.Equal(t, int64(100), metrics.QueueDepths["queue-1"])
	assert.Equal(t, int64(200), metrics.QueueDepths["queue-2"])
}

func TestBrokerMetricsIncrementReconnections(t *testing.T) {
	metrics := NewBrokerMetrics()

	metrics.IncrementReconnections()
	metrics.IncrementReconnections()

	assert.Equal(t, int64(2), metrics.Reconnections)
}

func TestBrokerMetricsGetAveragePublishLatency(t *testing.T) {
	metrics := NewBrokerMetrics()

	// No publishes
	assert.Equal(t, time.Duration(0), metrics.GetAveragePublishLatency())

	// With publishes
	metrics.RecordPublish("topic", 100, 100*time.Millisecond)
	metrics.RecordPublish("topic", 100, 200*time.Millisecond)

	avg := metrics.GetAveragePublishLatency()
	assert.Equal(t, 150*time.Millisecond, avg)
}

func TestBrokerMetricsGetAverageConsumeLatency(t *testing.T) {
	metrics := NewBrokerMetrics()

	// No consumes
	assert.Equal(t, time.Duration(0), metrics.GetAverageConsumeLatency())

	// With consumes
	metrics.RecordConsume("topic", 100, 50*time.Millisecond)
	metrics.RecordConsume("topic", 100, 150*time.Millisecond)

	avg := metrics.GetAverageConsumeLatency()
	assert.Equal(t, 100*time.Millisecond, avg)
}

func TestBrokerMetricsGetThroughput(t *testing.T) {
	metrics := NewBrokerMetrics()

	// Zero duration
	assert.Equal(t, float64(0), metrics.GetThroughput(0))

	// With messages
	for i := 0; i < 100; i++ {
		metrics.RecordPublish("topic", 100, time.Millisecond)
	}
	for i := 0; i < 50; i++ {
		metrics.RecordConsume("topic", 100, time.Millisecond)
	}

	throughput := metrics.GetThroughput(10 * time.Second)
	assert.Equal(t, float64(15), throughput) // 150 messages in 10 seconds = 15/s
}

func TestBrokerMetricsGetErrorRate(t *testing.T) {
	metrics := NewBrokerMetrics()

	// No messages
	assert.Equal(t, float64(0), metrics.GetErrorRate())

	// With messages and errors
	for i := 0; i < 100; i++ {
		metrics.RecordPublish("topic", 100, time.Millisecond)
	}
	metrics.RecordPublishError()
	metrics.RecordPublishError()

	errorRate := metrics.GetErrorRate()
	assert.Equal(t, float64(2), errorRate) // 2/100 = 2%
}

func TestBrokerMetricsSnapshot(t *testing.T) {
	metrics := NewBrokerMetrics()

	metrics.SetConnected(true)
	metrics.RecordPublish("topic-1", 100, time.Millisecond)
	metrics.RecordConsume("topic-2", 200, time.Millisecond)
	metrics.SetQueueDepth("queue-1", 50)

	snapshot := metrics.Snapshot()

	// Snapshot should have same values
	assert.True(t, snapshot.Connected)
	assert.Equal(t, int64(1), snapshot.MessagesPublished)
	assert.Equal(t, int64(1), snapshot.MessagesConsumed)
	assert.Equal(t, int64(50), snapshot.QueueDepths["queue-1"])
	assert.Contains(t, snapshot.TopicMetrics, "topic-1")
	assert.Contains(t, snapshot.TopicMetrics, "topic-2")

	// Modifying snapshot shouldn't affect original
	snapshot.MessagesPublished = 999
	assert.NotEqual(t, int64(999), metrics.MessagesPublished)
}

func TestBrokerMetricsReset(t *testing.T) {
	metrics := NewBrokerMetrics()

	// Add some data
	metrics.SetConnected(true)
	metrics.RecordPublish("topic", 100, time.Millisecond)
	metrics.RecordConsume("topic", 100, time.Millisecond)
	metrics.RecordFailure()
	metrics.SetQueueDepth("queue", 100)

	metrics.Reset()

	// All counters should be zero
	assert.Equal(t, int64(0), metrics.MessagesPublished)
	assert.Equal(t, int64(0), metrics.MessagesConsumed)
	assert.Equal(t, int64(0), metrics.MessagesFailed)
	assert.Empty(t, metrics.QueueDepths)
	assert.Empty(t, metrics.TopicMetrics)

	// Connection status is NOT reset
	assert.True(t, metrics.Connected)
}

func TestTopicMetricsRecording(t *testing.T) {
	metrics := NewBrokerMetrics()

	// Record multiple publishes to same topic
	metrics.RecordPublish("topic-a", 100, time.Millisecond)
	metrics.RecordPublish("topic-a", 200, time.Millisecond)

	tm := metrics.TopicMetrics["topic-a"]
	assert.NotNil(t, tm)
	assert.Equal(t, "topic-a", tm.Name)
	assert.Equal(t, int64(2), tm.MessagesIn)
	assert.Equal(t, int64(300), tm.BytesIn)

	// Record consume
	metrics.RecordConsume("topic-a", 150, time.Millisecond)
	assert.Equal(t, int64(1), tm.MessagesOut)
	assert.Equal(t, int64(150), tm.BytesOut)
}
