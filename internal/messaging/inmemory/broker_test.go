package inmemory

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"llmsverifier/internal/messaging"
)

func TestNewBroker(t *testing.T) {
	t.Run("with nil config", func(t *testing.T) {
		broker := NewBroker(nil, nil)
		assert.NotNil(t, broker)
		assert.NotNil(t, broker.config)
		assert.Equal(t, 100000, broker.config.MaxQueueSize)
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &Config{
			MaxQueueSize:      1000,
			MaxTopicSize:      500,
			DefaultPartitions: 5,
			MessageTTL:        1 * time.Hour,
		}
		logger := logrus.New()
		broker := NewBroker(config, logger)
		assert.NotNil(t, broker)
		assert.Equal(t, 1000, broker.config.MaxQueueSize)
		assert.Equal(t, 5, broker.config.DefaultPartitions)
	})
}

func TestBrokerConnect(t *testing.T) {
	broker := NewBroker(nil, nil)

	t.Run("successful connect", func(t *testing.T) {
		err := broker.Connect(context.Background())
		assert.NoError(t, err)
		assert.True(t, broker.IsConnected())
	})

	t.Run("connect after close fails", func(t *testing.T) {
		broker2 := NewBroker(nil, nil)
		_ = broker2.Connect(context.Background())
		_ = broker2.Close(context.Background())
		err := broker2.Connect(context.Background())
		assert.Equal(t, messaging.ErrConnectionClosed, err)
	})
}

func TestBrokerClose(t *testing.T) {
	broker := NewBroker(nil, nil)
	_ = broker.Connect(context.Background())

	err := broker.Close(context.Background())
	assert.NoError(t, err)
	assert.False(t, broker.IsConnected())

	// Close is idempotent
	err = broker.Close(context.Background())
	assert.NoError(t, err)
}

func TestBrokerHealthCheck(t *testing.T) {
	broker := NewBroker(nil, nil)

	t.Run("not connected", func(t *testing.T) {
		err := broker.HealthCheck(context.Background())
		assert.Equal(t, messaging.ErrNotConnected, err)
	})

	t.Run("connected", func(t *testing.T) {
		_ = broker.Connect(context.Background())
		err := broker.HealthCheck(context.Background())
		assert.NoError(t, err)
	})
}

func TestBrokerType(t *testing.T) {
	broker := NewBroker(nil, nil)
	assert.Equal(t, messaging.BrokerTypeInMemory, broker.BrokerType())
}

func TestBrokerPublish(t *testing.T) {
	broker := NewBroker(nil, nil)
	ctx := context.Background()

	t.Run("publish without connect fails", func(t *testing.T) {
		msg := &messaging.Message{
			ID:      "test-1",
			Type:    "test",
			Payload: []byte("hello"),
		}
		err := broker.Publish(ctx, "test-queue", msg)
		assert.Equal(t, messaging.ErrNotConnected, err)
	})

	_ = broker.Connect(ctx)

	t.Run("publish creates queue automatically", func(t *testing.T) {
		msg := &messaging.Message{
			ID:        "test-2",
			Type:      "test",
			Payload:   []byte("hello"),
			Timestamp: time.Now(),
		}
		err := broker.Publish(ctx, "auto-queue", msg)
		assert.NoError(t, err)

		// Queue should be created
		_, exists := broker.queues["auto-queue"]
		assert.True(t, exists)
	})

	t.Run("publish to declared queue", func(t *testing.T) {
		_ = broker.DeclareQueue(ctx, "declared-queue")
		msg := &messaging.Message{
			ID:        "test-3",
			Type:      "test",
			Payload:   []byte("hello world"),
			Timestamp: time.Now(),
		}
		err := broker.Publish(ctx, "declared-queue", msg)
		assert.NoError(t, err)
	})

	t.Run("publish to topic", func(t *testing.T) {
		_ = broker.CreateTopic(ctx, "test-topic", 3, 1)
		msg := &messaging.Message{
			ID:        "test-4",
			Type:      "test",
			Payload:   []byte("topic message"),
			Timestamp: time.Now(),
		}
		err := broker.Publish(ctx, "test-topic", msg)
		assert.NoError(t, err)
	})
}

func TestBrokerPublishBatch(t *testing.T) {
	broker := NewBroker(nil, nil)
	ctx := context.Background()
	_ = broker.Connect(ctx)

	messages := []*messaging.Message{
		{ID: "batch-1", Type: "test", Payload: []byte("msg1"), Timestamp: time.Now()},
		{ID: "batch-2", Type: "test", Payload: []byte("msg2"), Timestamp: time.Now()},
		{ID: "batch-3", Type: "test", Payload: []byte("msg3"), Timestamp: time.Now()},
	}

	err := broker.PublishBatch(ctx, "batch-queue", messages)
	assert.NoError(t, err)

	// Verify all messages were published
	assert.Equal(t, 3, broker.queues["batch-queue"].Len())
}

func TestBrokerSubscribe(t *testing.T) {
	broker := NewBroker(nil, nil)
	ctx := context.Background()

	t.Run("subscribe without connect fails", func(t *testing.T) {
		_, err := broker.Subscribe(ctx, "test", func(ctx context.Context, msg *messaging.Message) error {
			return nil
		})
		assert.Equal(t, messaging.ErrNotConnected, err)
	})

	_ = broker.Connect(ctx)

	t.Run("successful subscription", func(t *testing.T) {
		received := make(chan *messaging.Message, 1)
		sub, err := broker.Subscribe(ctx, "sub-queue", func(ctx context.Context, msg *messaging.Message) error {
			received <- msg
			return nil
		})
		require.NoError(t, err)
		assert.NotNil(t, sub)
		assert.True(t, sub.IsActive())

		// Publish a message
		msg := &messaging.Message{
			ID:        "sub-test",
			Type:      "test",
			Payload:   []byte("subscription test"),
			Timestamp: time.Now(),
		}
		_ = broker.Publish(ctx, "sub-queue", msg)

		// Should receive the message
		select {
		case recvMsg := <-received:
			assert.Equal(t, "sub-test", recvMsg.ID)
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for message")
		}

		// Unsubscribe
		err = sub.Unsubscribe()
		assert.NoError(t, err)
		assert.False(t, sub.IsActive())
	})
}

func TestBrokerDeclareQueue(t *testing.T) {
	broker := NewBroker(nil, nil)
	ctx := context.Background()
	_ = broker.Connect(ctx)

	t.Run("declare new queue", func(t *testing.T) {
		err := broker.DeclareQueue(ctx, "new-queue")
		assert.NoError(t, err)
		_, exists := broker.queues["new-queue"]
		assert.True(t, exists)
	})

	t.Run("declare existing queue is no-op", func(t *testing.T) {
		err := broker.DeclareQueue(ctx, "new-queue")
		assert.NoError(t, err)
	})
}

func TestBrokerDeleteQueue(t *testing.T) {
	broker := NewBroker(nil, nil)
	ctx := context.Background()
	_ = broker.Connect(ctx)
	_ = broker.DeclareQueue(ctx, "to-delete")

	err := broker.DeleteQueue(ctx, "to-delete")
	assert.NoError(t, err)
	_, exists := broker.queues["to-delete"]
	assert.False(t, exists)
}

func TestBrokerPurgeQueue(t *testing.T) {
	broker := NewBroker(nil, nil)
	ctx := context.Background()
	_ = broker.Connect(ctx)
	_ = broker.DeclareQueue(ctx, "purge-queue")

	// Add messages
	for i := 0; i < 5; i++ {
		msg := &messaging.Message{
			ID:        "purge-" + string(rune('0'+i)),
			Type:      "test",
			Payload:   []byte("test"),
			Timestamp: time.Now(),
		}
		_ = broker.Publish(ctx, "purge-queue", msg)
	}

	count, err := broker.PurgeQueue(ctx, "purge-queue")
	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)
	assert.Equal(t, 0, broker.queues["purge-queue"].Len())
}

func TestBrokerTaskOperations(t *testing.T) {
	broker := NewBroker(nil, nil)
	ctx := context.Background()
	_ = broker.Connect(ctx)
	_ = broker.DeclareQueue(ctx, "task-queue")

	t.Run("enqueue and dequeue task", func(t *testing.T) {
		task := &messaging.Task{
			ID:        "task-1",
			Type:      "process",
			Payload:   []byte(`{"action":"test"}`),
			Priority:  5,
			CreatedAt: time.Now(),
		}

		err := broker.EnqueueTask(ctx, "task-queue", task)
		assert.NoError(t, err)

		dequeued, err := broker.DequeueTask(ctx, "task-queue", "worker-1")
		assert.NoError(t, err)
		assert.Equal(t, "task-1", dequeued.ID)
		assert.Equal(t, "worker-1", dequeued.WorkerID)
		assert.Equal(t, messaging.TaskStatusRunning, dequeued.Status)
	})

	t.Run("dequeue from empty queue", func(t *testing.T) {
		_ = broker.DeclareQueue(ctx, "empty-queue")
		_, err := broker.DequeueTask(ctx, "empty-queue", "worker-1")
		assert.Error(t, err)
	})

	t.Run("dequeue from non-existent queue", func(t *testing.T) {
		_, err := broker.DequeueTask(ctx, "non-existent", "worker-1")
		assert.Equal(t, messaging.ErrQueueNotFound, err)
	})

	t.Run("ack task", func(t *testing.T) {
		task := &messaging.Task{ID: "task-ack"}
		err := broker.AckTask(ctx, task)
		assert.NoError(t, err)
	})

	t.Run("nack task", func(t *testing.T) {
		task := &messaging.Task{ID: "task-nack"}
		err := broker.NackTask(ctx, task, false)
		assert.NoError(t, err)
	})
}

func TestBrokerMoveToDeadLetter(t *testing.T) {
	broker := NewBroker(nil, nil)
	ctx := context.Background()
	_ = broker.Connect(ctx)

	task := &messaging.Task{
		ID:      "dead-task",
		Type:    "process",
		Payload: []byte("failed task"),
	}

	err := broker.MoveToDeadLetter(ctx, task, "processing failed")
	assert.NoError(t, err)
	assert.Equal(t, messaging.TaskStatusDeadLetter, task.Status)
	assert.Equal(t, "processing failed", task.Error)
}

func TestBrokerGetQueueStats(t *testing.T) {
	broker := NewBroker(nil, nil)
	ctx := context.Background()
	_ = broker.Connect(ctx)
	_ = broker.DeclareQueue(ctx, "stats-queue")

	// Add some messages
	for i := 0; i < 3; i++ {
		msg := &messaging.Message{
			ID:        "stats-" + string(rune('0'+i)),
			Type:      "test",
			Payload:   []byte("test"),
			Timestamp: time.Now(),
		}
		_ = broker.Publish(ctx, "stats-queue", msg)
	}

	stats, err := broker.GetQueueStats(ctx, "stats-queue")
	assert.NoError(t, err)
	assert.Equal(t, "stats-queue", stats.Name)
	assert.Equal(t, int64(3), stats.Messages)

	// Non-existent queue
	_, err = broker.GetQueueStats(ctx, "non-existent")
	assert.Equal(t, messaging.ErrQueueNotFound, err)
}

func TestBrokerScheduleTask(t *testing.T) {
	broker := NewBroker(nil, nil)
	ctx := context.Background()
	_ = broker.Connect(ctx)
	_ = broker.DeclareQueue(ctx, "schedule-queue")

	task := &messaging.Task{
		ID:      "scheduled-task",
		Type:    "delayed",
		Payload: []byte("delayed task"),
	}

	executeAt := time.Now().Add(100 * time.Millisecond)
	err := broker.ScheduleTask(ctx, "schedule-queue", task, executeAt)
	assert.NoError(t, err)
	assert.Equal(t, messaging.TaskStatusScheduled, task.Status)

	// Wait for task to be enqueued
	time.Sleep(200 * time.Millisecond)

	// Task should now be in queue
	stats, _ := broker.GetQueueStats(ctx, "schedule-queue")
	assert.Equal(t, int64(1), stats.Messages)
}

func TestBrokerTopicOperations(t *testing.T) {
	broker := NewBroker(nil, nil)
	ctx := context.Background()
	_ = broker.Connect(ctx)

	t.Run("create topic", func(t *testing.T) {
		err := broker.CreateTopic(ctx, "topic-1", 3, 1)
		assert.NoError(t, err)
		_, exists := broker.topics["topic-1"]
		assert.True(t, exists)
	})

	t.Run("list topics", func(t *testing.T) {
		_ = broker.CreateTopic(ctx, "topic-2", 2, 1)
		topics, err := broker.ListTopics(ctx)
		assert.NoError(t, err)
		assert.Len(t, topics, 2)
	})

	t.Run("get topic metadata", func(t *testing.T) {
		meta, err := broker.GetTopicMetadata(ctx, "topic-1")
		assert.NoError(t, err)
		assert.Equal(t, "topic-1", meta.Name)
		assert.Equal(t, int32(3), meta.Partitions)
	})

	t.Run("get non-existent topic metadata", func(t *testing.T) {
		_, err := broker.GetTopicMetadata(ctx, "non-existent")
		assert.Equal(t, messaging.ErrTopicNotFound, err)
	})

	t.Run("delete topic", func(t *testing.T) {
		err := broker.DeleteTopic(ctx, "topic-2")
		assert.NoError(t, err)
		_, exists := broker.topics["topic-2"]
		assert.False(t, exists)
	})
}

func TestBrokerEventOperations(t *testing.T) {
	broker := NewBroker(nil, nil)
	ctx := context.Background()
	_ = broker.Connect(ctx)
	_ = broker.CreateTopic(ctx, "events", 3, 1)

	t.Run("publish event", func(t *testing.T) {
		event := &messaging.Event{
			ID:        "event-1",
			Type:      "user.created",
			Source:    "test",
			Data:      []byte(`{"user_id":"123"}`),
			Timestamp: time.Now(),
		}

		err := broker.PublishEvent(ctx, "events", event)
		assert.NoError(t, err)
	})

	t.Run("publish multiple events", func(t *testing.T) {
		events := []*messaging.Event{
			{ID: "event-2", Type: "user.updated", Source: "test", Data: []byte("{}"), Timestamp: time.Now()},
			{ID: "event-3", Type: "user.deleted", Source: "test", Data: []byte("{}"), Timestamp: time.Now()},
		}

		err := broker.PublishEvents(ctx, "events", events)
		assert.NoError(t, err)
	})
}

func TestBrokerConsumerGroupOperations(t *testing.T) {
	broker := NewBroker(nil, nil)
	ctx := context.Background()
	_ = broker.Connect(ctx)
	_ = broker.CreateTopic(ctx, "cg-topic", 3, 1)

	t.Run("create consumer group", func(t *testing.T) {
		err := broker.CreateConsumerGroup(ctx, "group-1")
		assert.NoError(t, err)
	})

	t.Run("get consumer group info", func(t *testing.T) {
		info, err := broker.GetConsumerGroupInfo(ctx, "group-1")
		assert.NoError(t, err)
		assert.Equal(t, "group-1", info.GroupID)
	})

	t.Run("list consumer groups", func(t *testing.T) {
		groups, err := broker.ListConsumerGroups(ctx)
		assert.NoError(t, err)
		assert.Contains(t, groups, "group-1")
	})

	t.Run("delete consumer group", func(t *testing.T) {
		err := broker.DeleteConsumerGroup(ctx, "group-1")
		assert.NoError(t, err)
	})
}

func TestBrokerOffsetOperations(t *testing.T) {
	broker := NewBroker(nil, nil)
	ctx := context.Background()
	_ = broker.Connect(ctx)
	_ = broker.CreateTopic(ctx, "offset-topic", 3, 1)

	// Publish some messages
	for i := 0; i < 5; i++ {
		event := &messaging.Event{
			ID:        "offset-" + string(rune('0'+i)),
			Type:      "test",
			Data:      []byte("test"),
			Timestamp: time.Now(),
		}
		_ = broker.PublishEvent(ctx, "offset-topic", event)
	}

	t.Run("commit offset", func(t *testing.T) {
		err := broker.CommitOffset(ctx, "offset-topic", 0, 10)
		assert.NoError(t, err)
	})

	t.Run("get committed offset", func(t *testing.T) {
		offset, err := broker.GetCommittedOffset(ctx, "offset-topic", 0)
		assert.NoError(t, err)
		assert.Equal(t, int64(10), offset)
	})

	t.Run("seek to offset", func(t *testing.T) {
		err := broker.SeekToOffset(ctx, "offset-topic", 0, 5)
		assert.NoError(t, err)
	})

	t.Run("seek to beginning", func(t *testing.T) {
		err := broker.SeekToBeginning(ctx, "offset-topic", 0)
		assert.NoError(t, err)
	})

	t.Run("seek to end", func(t *testing.T) {
		err := broker.SeekToEnd(ctx, "offset-topic", 0)
		assert.NoError(t, err)
	})

	t.Run("seek to timestamp", func(t *testing.T) {
		err := broker.SeekToTimestamp(ctx, "offset-topic", 0, time.Now().Add(-1*time.Hour))
		assert.NoError(t, err)
	})

	t.Run("operations on non-existent topic", func(t *testing.T) {
		_, err := broker.GetCommittedOffset(ctx, "non-existent", 0)
		assert.Equal(t, messaging.ErrTopicNotFound, err)

		err = broker.SeekToOffset(ctx, "non-existent", 0, 0)
		assert.Equal(t, messaging.ErrTopicNotFound, err)
	})
}

func TestBrokerStreamEvents(t *testing.T) {
	broker := NewBroker(nil, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_ = broker.Connect(ctx)
	_ = broker.CreateTopic(ctx, "stream-topic", 1, 1)

	// Publish some events first
	for i := 0; i < 3; i++ {
		event := &messaging.Event{
			ID:        "stream-" + string(rune('0'+i)),
			Type:      "test",
			Data:      []byte("stream data"),
			Timestamp: time.Now(),
		}
		_ = broker.PublishEvent(ctx, "stream-topic", event)
	}

	t.Run("stream events from earliest", func(t *testing.T) {
		eventCh, err := broker.StreamEvents(ctx, "stream-topic",
			messaging.WithStreamStartOffset("earliest"),
		)
		assert.NoError(t, err)

		var received []*messaging.Event
		timeout := time.After(2 * time.Second)

	loop:
		for {
			select {
			case event, ok := <-eventCh:
				if !ok {
					break loop
				}
				received = append(received, event)
				if len(received) >= 3 {
					break loop
				}
			case <-timeout:
				break loop
			}
		}

		assert.GreaterOrEqual(t, len(received), 3)
	})

	t.Run("stream from non-existent topic", func(t *testing.T) {
		_, err := broker.StreamEvents(ctx, "non-existent")
		assert.Equal(t, messaging.ErrTopicNotFound, err)
	})
}

func TestBrokerTransactions(t *testing.T) {
	broker := NewBroker(nil, nil)
	ctx := context.Background()
	_ = broker.Connect(ctx)

	// In-memory broker doesn't support real transactions, but methods should not error
	err := broker.BeginTransaction(ctx)
	assert.NoError(t, err)

	err = broker.CommitTransaction(ctx)
	assert.NoError(t, err)

	err = broker.AbortTransaction(ctx)
	assert.NoError(t, err)
}

func TestBrokerMetrics(t *testing.T) {
	broker := NewBroker(nil, nil)
	ctx := context.Background()
	_ = broker.Connect(ctx)

	metrics := broker.GetMetrics()
	assert.NotNil(t, metrics)
	assert.True(t, metrics.Connected)

	// Publish some messages to generate metrics
	for i := 0; i < 10; i++ {
		msg := &messaging.Message{
			ID:        "metrics-" + string(rune('0'+i)),
			Type:      "test",
			Payload:   []byte("metrics test"),
			Timestamp: time.Now(),
		}
		_ = broker.Publish(ctx, "metrics-queue", msg)
	}

	assert.Equal(t, int64(10), metrics.MessagesPublished)
}

func TestBrokerConcurrency(t *testing.T) {
	broker := NewBroker(nil, nil)
	ctx := context.Background()
	_ = broker.Connect(ctx)
	_ = broker.DeclareQueue(ctx, "concurrent-queue")

	var wg sync.WaitGroup
	numGoroutines := 100
	messagesPerGoroutine := 10

	// Concurrent publishers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < messagesPerGoroutine; j++ {
				msg := &messaging.Message{
					ID:        "concurrent-" + string(rune('0'+id)) + "-" + string(rune('0'+j)),
					Type:      "test",
					Payload:   []byte("concurrent message"),
					Timestamp: time.Now(),
				}
				_ = broker.Publish(ctx, "concurrent-queue", msg)
			}
		}(i)
	}

	wg.Wait()

	// Verify all messages were received
	stats, _ := broker.GetQueueStats(ctx, "concurrent-queue")
	assert.Equal(t, int64(numGoroutines*messagesPerGoroutine), stats.Messages)
}
