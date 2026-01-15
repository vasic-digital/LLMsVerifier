// Package inmemory provides an in-memory message broker for testing and fallback scenarios.
package inmemory

import (
	"errors"
	"hash/fnv"
	"sync"
	"time"

	"llmsverifier/internal/messaging"
)

// ErrPartitionFull is returned when a partition has reached capacity.
var ErrPartitionFull = errors.New("partition is full")

// Topic implements a Kafka-like topic with partitions.
type Topic struct {
	mu sync.RWMutex

	name       string
	partitions []*Partition
	maxSize    int

	// Consumer groups
	consumerGroups map[string]*ConsumerGroup
}

// NewTopic creates a new Topic with the given number of partitions.
func NewTopic(name string, numPartitions, maxSize int) *Topic {
	if numPartitions <= 0 {
		numPartitions = 1
	}

	partitions := make([]*Partition, numPartitions)
	for i := 0; i < numPartitions; i++ {
		partitions[i] = NewPartition(int32(i), maxSize)
	}

	return &Topic{
		name:           name,
		partitions:     partitions,
		maxSize:        maxSize,
		consumerGroups: make(map[string]*ConsumerGroup),
	}
}

// Publish appends a message to the appropriate partition.
func (t *Topic) Publish(msg *messaging.Message) error {
	partition := t.selectPartition(msg)
	return t.partitions[partition].Append(msg)
}

// PublishToPartition appends a message to a specific partition.
func (t *Topic) PublishToPartition(msg *messaging.Message, partition int32) error {
	if int(partition) >= len(t.partitions) {
		return errors.New("invalid partition")
	}
	return t.partitions[partition].Append(msg)
}

// Read reads messages from a partition starting at the given offset.
func (t *Topic) Read(partition int32, offset int64, maxMessages int) ([]*messaging.Message, error) {
	if int(partition) >= len(t.partitions) {
		return nil, errors.New("invalid partition")
	}
	return t.partitions[partition].Read(offset, maxMessages), nil
}

// NumPartitions returns the number of partitions.
func (t *Topic) NumPartitions() int32 {
	return int32(len(t.partitions))
}

// Name returns the topic name.
func (t *Topic) Name() string {
	return t.name
}

// GetPartition returns a specific partition.
func (t *Topic) GetPartition(index int32) *Partition {
	if int(index) >= len(t.partitions) {
		return nil
	}
	return t.partitions[index]
}

// selectPartition selects a partition for a message.
func (t *Topic) selectPartition(msg *messaging.Message) int32 {
	numPartitions := int32(len(t.partitions))
	if numPartitions == 0 {
		return 0
	}

	if msg.Key != "" {
		// Use consistent hashing based on key
		h := fnv.New32a()
		h.Write([]byte(msg.Key))
		return int32(h.Sum32() % uint32(numPartitions))
	}
	// Round-robin for messages without keys using absolute modulo
	t.mu.Lock()
	defer t.mu.Unlock()
	// Use timestamp and ensure non-negative result
	nano := time.Now().UnixNano()
	partition := nano % int64(numPartitions)
	if partition < 0 {
		partition = -partition
	}
	return int32(partition)
}

// GetOrCreateConsumerGroup gets or creates a consumer group.
func (t *Topic) GetOrCreateConsumerGroup(groupID string) *ConsumerGroup {
	t.mu.Lock()
	defer t.mu.Unlock()

	if group, ok := t.consumerGroups[groupID]; ok {
		return group
	}

	group := NewConsumerGroup(groupID, t)
	t.consumerGroups[groupID] = group
	return group
}

// DeleteConsumerGroup removes a consumer group.
func (t *Topic) DeleteConsumerGroup(groupID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.consumerGroups, groupID)
}

// Partition represents a single partition within a topic.
type Partition struct {
	mu sync.RWMutex

	id       int32
	messages []*messaging.Message
	maxSize  int
	offset   int64
}

// NewPartition creates a new Partition.
func NewPartition(id int32, maxSize int) *Partition {
	return &Partition{
		id:       id,
		messages: make([]*messaging.Message, 0, 1000),
		maxSize:  maxSize,
		offset:   0,
	}
}

// Append adds a message to the partition.
func (p *Partition) Append(msg *messaging.Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.messages) >= p.maxSize {
		// Remove oldest messages (compaction)
		p.messages = p.messages[p.maxSize/2:]
	}

	msg.Partition = p.id
	msg.Offset = p.offset
	p.messages = append(p.messages, msg)
	p.offset++
	return nil
}

// Read reads messages starting from an offset.
func (p *Partition) Read(offset int64, maxMessages int) []*messaging.Message {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.messages) == 0 {
		return nil
	}

	// Find starting index
	startIdx := 0
	for i, msg := range p.messages {
		if msg.Offset >= offset {
			startIdx = i
			break
		}
	}

	endIdx := startIdx + maxMessages
	if endIdx > len(p.messages) {
		endIdx = len(p.messages)
	}

	result := make([]*messaging.Message, endIdx-startIdx)
	copy(result, p.messages[startIdx:endIdx])
	return result
}

// ID returns the partition ID.
func (p *Partition) ID() int32 {
	return p.id
}

// Size returns the number of messages.
func (p *Partition) Size() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.messages)
}

// BeginOffset returns the first offset.
func (p *Partition) BeginOffset() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if len(p.messages) == 0 {
		return 0
	}
	return p.messages[0].Offset
}

// EndOffset returns the next offset to be written.
func (p *Partition) EndOffset() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.offset
}

// ConsumerGroup tracks offsets for a group of consumers.
type ConsumerGroup struct {
	mu sync.RWMutex

	groupID    string
	topic      *Topic
	offsets    map[int32]int64 // partition -> committed offset
	members    map[string]*ConsumerMember
	generation int32
}

// ConsumerMember represents a member of a consumer group.
type ConsumerMember struct {
	MemberID   string
	ClientID   string
	Partitions []int32
	JoinedAt   time.Time
}

// NewConsumerGroup creates a new ConsumerGroup.
func NewConsumerGroup(groupID string, topic *Topic) *ConsumerGroup {
	offsets := make(map[int32]int64)
	for i := int32(0); i < topic.NumPartitions(); i++ {
		offsets[i] = 0
	}

	return &ConsumerGroup{
		groupID: groupID,
		topic:   topic,
		offsets: offsets,
		members: make(map[string]*ConsumerMember),
	}
}

// CommitOffset commits an offset for a partition.
func (cg *ConsumerGroup) CommitOffset(partition int32, offset int64) {
	cg.mu.Lock()
	defer cg.mu.Unlock()
	cg.offsets[partition] = offset
}

// GetCommittedOffset returns the committed offset for a partition.
func (cg *ConsumerGroup) GetCommittedOffset(partition int32) int64 {
	cg.mu.RLock()
	defer cg.mu.RUnlock()
	return cg.offsets[partition]
}

// AddMember adds a member to the consumer group.
func (cg *ConsumerGroup) AddMember(memberID, clientID string) *ConsumerMember {
	cg.mu.Lock()
	defer cg.mu.Unlock()

	member := &ConsumerMember{
		MemberID: memberID,
		ClientID: clientID,
		JoinedAt: time.Now(),
	}
	cg.members[memberID] = member
	cg.generation++

	// Rebalance partitions
	cg.rebalance()

	return member
}

// RemoveMember removes a member from the consumer group.
func (cg *ConsumerGroup) RemoveMember(memberID string) {
	cg.mu.Lock()
	defer cg.mu.Unlock()

	delete(cg.members, memberID)
	cg.generation++
	cg.rebalance()
}

// rebalance redistributes partitions among members.
func (cg *ConsumerGroup) rebalance() {
	if len(cg.members) == 0 {
		return
	}

	// Clear existing assignments
	for _, member := range cg.members {
		member.Partitions = nil
	}

	// Simple round-robin assignment
	members := make([]*ConsumerMember, 0, len(cg.members))
	for _, member := range cg.members {
		members = append(members, member)
	}

	numPartitions := cg.topic.NumPartitions()
	for i := int32(0); i < numPartitions; i++ {
		member := members[int(i)%len(members)]
		member.Partitions = append(member.Partitions, i)
	}
}

// GroupID returns the group ID.
func (cg *ConsumerGroup) GroupID() string {
	return cg.groupID
}

// Members returns all members.
func (cg *ConsumerGroup) Members() []*ConsumerMember {
	cg.mu.RLock()
	defer cg.mu.RUnlock()

	members := make([]*ConsumerMember, 0, len(cg.members))
	for _, member := range cg.members {
		members = append(members, member)
	}
	return members
}

// Generation returns the current generation.
func (cg *ConsumerGroup) Generation() int32 {
	cg.mu.RLock()
	defer cg.mu.RUnlock()
	return cg.generation
}
