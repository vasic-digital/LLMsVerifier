// Package inmemory provides an in-memory message broker for testing and fallback scenarios.
package inmemory

import (
	"container/heap"
	"errors"
	"sync"
	"time"

	"llmsverifier/internal/messaging"
)

// ErrQueueFull is returned when the queue has reached capacity.
var ErrQueueFull = errors.New("queue is full")

// ErrQueueEmpty is returned when trying to pop from an empty queue.
var ErrQueueEmpty = errors.New("queue is empty")

// Queue implements a thread-safe priority queue for messages.
type Queue struct {
	mu sync.Mutex

	name     string
	maxSize  int
	ttl      time.Duration
	messages *priorityQueue

	// Metrics
	enqueued int64
	dequeued int64
	expired  int64
}

// NewQueue creates a new Queue with the given parameters.
func NewQueue(name string, maxSize int, ttl time.Duration) *Queue {
	pq := make(priorityQueue, 0)
	heap.Init(&pq)

	return &Queue{
		name:     name,
		maxSize:  maxSize,
		ttl:      ttl,
		messages: &pq,
	}
}

// Push adds a message to the queue.
func (q *Queue) Push(msg *messaging.Message) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Remove expired messages first
	q.removeExpired()

	if q.messages.Len() >= q.maxSize {
		return ErrQueueFull
	}

	item := &queueItem{
		message:   msg,
		priority:  msg.Priority,
		timestamp: msg.Timestamp,
		expiry:    time.Now().Add(q.ttl),
	}

	heap.Push(q.messages, item)
	q.enqueued++
	return nil
}

// Pop removes and returns the highest priority message.
func (q *Queue) Pop() (*messaging.Message, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	// Remove expired messages first
	q.removeExpired()

	if q.messages.Len() == 0 {
		return nil, ErrQueueEmpty
	}

	item := heap.Pop(q.messages).(*queueItem)
	q.dequeued++
	return item.message, nil
}

// Peek returns the highest priority message without removing it.
func (q *Queue) Peek() (*messaging.Message, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.removeExpired()

	if q.messages.Len() == 0 {
		return nil, ErrQueueEmpty
	}

	return (*q.messages)[0].message, nil
}

// Len returns the number of messages in the queue.
func (q *Queue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.removeExpired()
	return q.messages.Len()
}

// Clear removes all messages from the queue.
func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()

	pq := make(priorityQueue, 0)
	heap.Init(&pq)
	q.messages = &pq
}

// Name returns the queue name.
func (q *Queue) Name() string {
	return q.name
}

// Stats returns queue statistics.
func (q *Queue) Stats() QueueStats {
	q.mu.Lock()
	defer q.mu.Unlock()

	return QueueStats{
		Name:     q.name,
		Size:     q.messages.Len(),
		MaxSize:  q.maxSize,
		Enqueued: q.enqueued,
		Dequeued: q.dequeued,
		Expired:  q.expired,
	}
}

// QueueStats contains queue statistics.
type QueueStats struct {
	Name     string
	Size     int
	MaxSize  int
	Enqueued int64
	Dequeued int64
	Expired  int64
}

// removeExpired removes expired messages from the queue.
func (q *Queue) removeExpired() {
	now := time.Now()
	newPQ := make(priorityQueue, 0, q.messages.Len())

	for _, item := range *q.messages {
		if item.expiry.After(now) {
			newPQ = append(newPQ, item)
		} else {
			q.expired++
		}
	}

	heap.Init(&newPQ)
	q.messages = &newPQ
}

// queueItem represents an item in the priority queue.
type queueItem struct {
	message   *messaging.Message
	priority  int
	timestamp time.Time
	expiry    time.Time
	index     int
}

// priorityQueue implements heap.Interface for priority-based message ordering.
type priorityQueue []*queueItem

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	// Higher priority first
	if pq[i].priority != pq[j].priority {
		return pq[i].priority > pq[j].priority
	}
	// Older messages first (FIFO within same priority)
	return pq[i].timestamp.Before(pq[j].timestamp)
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*queueItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // Avoid memory leak
	item.index = -1 // For safety
	*pq = old[0 : n-1]
	return item
}
