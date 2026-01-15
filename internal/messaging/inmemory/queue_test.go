package inmemory

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"llmsverifier/internal/messaging"
)

func TestNewQueue(t *testing.T) {
	queue := NewQueue("test-queue", 100, 1*time.Hour)

	assert.NotNil(t, queue)
	assert.Equal(t, "test-queue", queue.Name())
	assert.Equal(t, 0, queue.Len())
}

func TestQueuePush(t *testing.T) {
	queue := NewQueue("test", 10, 1*time.Hour)

	t.Run("push single message", func(t *testing.T) {
		msg := &messaging.Message{
			ID:        "msg-1",
			Type:      "test",
			Payload:   []byte("hello"),
			Timestamp: time.Now(),
		}

		err := queue.Push(msg)
		assert.NoError(t, err)
		assert.Equal(t, 1, queue.Len())
	})

	t.Run("push multiple messages", func(t *testing.T) {
		queue := NewQueue("multi", 100, 1*time.Hour)

		for i := 0; i < 5; i++ {
			msg := &messaging.Message{
				ID:        "multi-" + string(rune('0'+i)),
				Type:      "test",
				Payload:   []byte("test"),
				Timestamp: time.Now(),
			}
			err := queue.Push(msg)
			assert.NoError(t, err)
		}

		assert.Equal(t, 5, queue.Len())
	})

	t.Run("push to full queue", func(t *testing.T) {
		queue := NewQueue("full", 3, 1*time.Hour)

		for i := 0; i < 3; i++ {
			msg := &messaging.Message{
				ID:        "full-" + string(rune('0'+i)),
				Type:      "test",
				Payload:   []byte("test"),
				Timestamp: time.Now(),
			}
			_ = queue.Push(msg)
		}

		msg := &messaging.Message{
			ID:        "overflow",
			Type:      "test",
			Payload:   []byte("overflow"),
			Timestamp: time.Now(),
		}

		err := queue.Push(msg)
		assert.Equal(t, ErrQueueFull, err)
	})
}

func TestQueuePop(t *testing.T) {
	t.Run("pop from empty queue", func(t *testing.T) {
		queue := NewQueue("empty", 10, 1*time.Hour)

		_, err := queue.Pop()
		assert.Equal(t, ErrQueueEmpty, err)
	})

	t.Run("pop messages", func(t *testing.T) {
		queue := NewQueue("pop", 10, 1*time.Hour)

		// Push messages with different priorities
		msgs := []*messaging.Message{
			{ID: "low", Type: "test", Payload: []byte("low"), Priority: 1, Timestamp: time.Now()},
			{ID: "high", Type: "test", Payload: []byte("high"), Priority: 10, Timestamp: time.Now()},
			{ID: "medium", Type: "test", Payload: []byte("medium"), Priority: 5, Timestamp: time.Now()},
		}

		for _, msg := range msgs {
			_ = queue.Push(msg)
		}

		// Should get highest priority first
		msg, err := queue.Pop()
		require.NoError(t, err)
		assert.Equal(t, "high", msg.ID)

		msg, err = queue.Pop()
		require.NoError(t, err)
		assert.Equal(t, "medium", msg.ID)

		msg, err = queue.Pop()
		require.NoError(t, err)
		assert.Equal(t, "low", msg.ID)
	})

	t.Run("FIFO within same priority", func(t *testing.T) {
		queue := NewQueue("fifo", 10, 1*time.Hour)

		// Push messages with same priority
		for i := 0; i < 3; i++ {
			msg := &messaging.Message{
				ID:        "fifo-" + string(rune('0'+i)),
				Type:      "test",
				Payload:   []byte("test"),
				Priority:  5,
				Timestamp: time.Now().Add(time.Duration(i) * time.Second),
			}
			_ = queue.Push(msg)
		}

		// Should get messages in FIFO order
		for i := 0; i < 3; i++ {
			msg, err := queue.Pop()
			require.NoError(t, err)
			assert.Equal(t, "fifo-"+string(rune('0'+i)), msg.ID)
		}
	})
}

func TestQueuePeek(t *testing.T) {
	queue := NewQueue("peek", 10, 1*time.Hour)

	t.Run("peek empty queue", func(t *testing.T) {
		_, err := queue.Peek()
		assert.Equal(t, ErrQueueEmpty, err)
	})

	t.Run("peek does not remove message", func(t *testing.T) {
		msg := &messaging.Message{
			ID:        "peek-1",
			Type:      "test",
			Payload:   []byte("peek"),
			Timestamp: time.Now(),
		}
		_ = queue.Push(msg)

		// Peek multiple times
		for i := 0; i < 3; i++ {
			peeked, err := queue.Peek()
			require.NoError(t, err)
			assert.Equal(t, "peek-1", peeked.ID)
		}

		// Message should still be in queue
		assert.Equal(t, 1, queue.Len())
	})
}

func TestQueueClear(t *testing.T) {
	queue := NewQueue("clear", 10, 1*time.Hour)

	// Add messages
	for i := 0; i < 5; i++ {
		msg := &messaging.Message{
			ID:        "clear-" + string(rune('0'+i)),
			Type:      "test",
			Payload:   []byte("test"),
			Timestamp: time.Now(),
		}
		_ = queue.Push(msg)
	}

	assert.Equal(t, 5, queue.Len())

	queue.Clear()

	assert.Equal(t, 0, queue.Len())
}

func TestQueueStats(t *testing.T) {
	queue := NewQueue("stats", 100, 1*time.Hour)

	// Push some messages
	for i := 0; i < 10; i++ {
		msg := &messaging.Message{
			ID:        "stats-" + string(rune('0'+i)),
			Type:      "test",
			Payload:   []byte("test"),
			Timestamp: time.Now(),
		}
		_ = queue.Push(msg)
	}

	// Pop some messages
	for i := 0; i < 3; i++ {
		_, _ = queue.Pop()
	}

	stats := queue.Stats()
	assert.Equal(t, "stats", stats.Name)
	assert.Equal(t, 7, stats.Size)
	assert.Equal(t, 100, stats.MaxSize)
	assert.Equal(t, int64(10), stats.Enqueued)
	assert.Equal(t, int64(3), stats.Dequeued)
}

func TestQueueMessageExpiry(t *testing.T) {
	// Create queue with short TTL
	queue := NewQueue("expiry", 100, 100*time.Millisecond)

	msg := &messaging.Message{
		ID:        "expiring",
		Type:      "test",
		Payload:   []byte("test"),
		Timestamp: time.Now(),
	}
	_ = queue.Push(msg)

	assert.Equal(t, 1, queue.Len())

	// Wait for message to expire
	time.Sleep(150 * time.Millisecond)

	// Message should be removed on next access
	_, err := queue.Pop()
	assert.Equal(t, ErrQueueEmpty, err)
}

func TestPriorityQueueOrdering(t *testing.T) {
	queue := NewQueue("priority", 100, 1*time.Hour)

	// Add messages with various priorities
	priorities := []int{3, 7, 1, 9, 5, 2, 8, 4, 6, 10}
	for i, p := range priorities {
		msg := &messaging.Message{
			ID:        "p-" + string(rune('0'+i)),
			Type:      "test",
			Payload:   []byte("test"),
			Priority:  p,
			Timestamp: time.Now().Add(time.Duration(i) * time.Millisecond),
		}
		_ = queue.Push(msg)
	}

	// Messages should come out in priority order (highest first)
	expectedOrder := []int{10, 9, 8, 7, 6, 5, 4, 3, 2, 1}
	for _, expectedPriority := range expectedOrder {
		msg, err := queue.Pop()
		require.NoError(t, err)
		assert.Equal(t, expectedPriority, msg.Priority)
	}
}
