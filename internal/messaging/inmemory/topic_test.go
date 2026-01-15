package inmemory

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"llmsverifier/internal/messaging"
)

func TestNewTopic(t *testing.T) {
	t.Run("create topic with partitions", func(t *testing.T) {
		topic := NewTopic("test-topic", 3, 1000)

		assert.NotNil(t, topic)
		assert.Equal(t, "test-topic", topic.Name())
		assert.Equal(t, int32(3), topic.NumPartitions())
	})

	t.Run("create topic with zero partitions defaults to 1", func(t *testing.T) {
		topic := NewTopic("default", 0, 1000)

		assert.Equal(t, int32(1), topic.NumPartitions())
	})

	t.Run("create topic with negative partitions defaults to 1", func(t *testing.T) {
		topic := NewTopic("negative", -5, 1000)

		assert.Equal(t, int32(1), topic.NumPartitions())
	})
}

func TestTopicPublish(t *testing.T) {
	topic := NewTopic("publish", 3, 1000)

	t.Run("publish message without key", func(t *testing.T) {
		msg := &messaging.Message{
			ID:        "msg-1",
			Type:      "test",
			Payload:   []byte("hello"),
			Timestamp: time.Now(),
		}

		err := topic.Publish(msg)
		assert.NoError(t, err)
	})

	t.Run("publish message with key", func(t *testing.T) {
		msg := &messaging.Message{
			ID:        "msg-2",
			Type:      "test",
			Payload:   []byte("keyed message"),
			Key:       "user-123",
			Timestamp: time.Now(),
		}

		err := topic.Publish(msg)
		assert.NoError(t, err)
	})

	t.Run("same key goes to same partition", func(t *testing.T) {
		topic := NewTopic("keyed", 10, 1000)

		// Publish multiple messages with same key
		var partitions []int32
		for i := 0; i < 5; i++ {
			msg := &messaging.Message{
				ID:        "keyed-" + string(rune('0'+i)),
				Type:      "test",
				Payload:   []byte("test"),
				Key:       "consistent-key",
				Timestamp: time.Now(),
			}
			_ = topic.Publish(msg)
			partitions = append(partitions, msg.Partition)
		}

		// All messages should be in the same partition
		for _, p := range partitions {
			assert.Equal(t, partitions[0], p)
		}
	})
}

func TestTopicPublishToPartition(t *testing.T) {
	topic := NewTopic("direct", 5, 1000)

	t.Run("publish to valid partition", func(t *testing.T) {
		msg := &messaging.Message{
			ID:        "direct-1",
			Type:      "test",
			Payload:   []byte("direct"),
			Timestamp: time.Now(),
		}

		err := topic.PublishToPartition(msg, 2)
		assert.NoError(t, err)
		assert.Equal(t, int32(2), msg.Partition)
	})

	t.Run("publish to invalid partition", func(t *testing.T) {
		msg := &messaging.Message{
			ID:        "invalid",
			Type:      "test",
			Payload:   []byte("invalid"),
			Timestamp: time.Now(),
		}

		err := topic.PublishToPartition(msg, 10)
		assert.Error(t, err)
	})
}

func TestTopicRead(t *testing.T) {
	topic := NewTopic("read", 1, 1000)

	// Publish messages
	for i := 0; i < 10; i++ {
		msg := &messaging.Message{
			ID:        "read-" + string(rune('0'+i)),
			Type:      "test",
			Payload:   []byte("test data"),
			Timestamp: time.Now(),
		}
		_ = topic.PublishToPartition(msg, 0)
	}

	t.Run("read from beginning", func(t *testing.T) {
		messages, err := topic.Read(0, 0, 5)
		assert.NoError(t, err)
		assert.Len(t, messages, 5)
		assert.Equal(t, "read-0", messages[0].ID)
	})

	t.Run("read from offset", func(t *testing.T) {
		messages, err := topic.Read(0, 3, 5)
		assert.NoError(t, err)
		assert.Len(t, messages, 5)
		assert.Equal(t, "read-3", messages[0].ID)
	})

	t.Run("read with limit exceeding available", func(t *testing.T) {
		messages, err := topic.Read(0, 7, 10)
		assert.NoError(t, err)
		assert.Len(t, messages, 3) // Only 3 messages from offset 7
	})

	t.Run("read from invalid partition", func(t *testing.T) {
		_, err := topic.Read(5, 0, 10)
		assert.Error(t, err)
	})
}

func TestPartition(t *testing.T) {
	partition := NewPartition(0, 100)

	t.Run("partition properties", func(t *testing.T) {
		assert.Equal(t, int32(0), partition.ID())
		assert.Equal(t, 0, partition.Size())
		assert.Equal(t, int64(0), partition.BeginOffset())
		assert.Equal(t, int64(0), partition.EndOffset())
	})

	t.Run("append and read", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			msg := &messaging.Message{
				ID:        "partition-" + string(rune('0'+i)),
				Type:      "test",
				Payload:   []byte("test"),
				Timestamp: time.Now(),
			}
			err := partition.Append(msg)
			assert.NoError(t, err)
			assert.Equal(t, int64(i), msg.Offset)
		}

		assert.Equal(t, 5, partition.Size())
		assert.Equal(t, int64(0), partition.BeginOffset())
		assert.Equal(t, int64(5), partition.EndOffset())

		messages := partition.Read(0, 10)
		assert.Len(t, messages, 5)
	})

	t.Run("partition compaction", func(t *testing.T) {
		partition := NewPartition(0, 10) // Small size to trigger compaction

		// Fill partition beyond max
		for i := 0; i < 15; i++ {
			msg := &messaging.Message{
				ID:        "compact-" + string(rune('0'+i)),
				Type:      "test",
				Payload:   []byte("test"),
				Timestamp: time.Now(),
			}
			_ = partition.Append(msg)
		}

		// Should have compacted to maxSize/2 + remaining
		assert.LessOrEqual(t, partition.Size(), 15)
	})
}

func TestConsumerGroup(t *testing.T) {
	topic := NewTopic("cg-topic", 3, 1000)
	cg := NewConsumerGroup("test-group", topic)

	t.Run("consumer group properties", func(t *testing.T) {
		assert.Equal(t, "test-group", cg.GroupID())
		assert.Equal(t, int32(0), cg.Generation())
		assert.Empty(t, cg.Members())
	})

	t.Run("commit and get offset", func(t *testing.T) {
		cg.CommitOffset(0, 100)
		cg.CommitOffset(1, 200)
		cg.CommitOffset(2, 300)

		assert.Equal(t, int64(100), cg.GetCommittedOffset(0))
		assert.Equal(t, int64(200), cg.GetCommittedOffset(1))
		assert.Equal(t, int64(300), cg.GetCommittedOffset(2))
	})

	t.Run("add member", func(t *testing.T) {
		member := cg.AddMember("member-1", "client-1")

		assert.Equal(t, "member-1", member.MemberID)
		assert.Equal(t, "client-1", member.ClientID)
		assert.NotEmpty(t, member.Partitions) // Should have partitions assigned
		assert.Equal(t, int32(1), cg.Generation())

		members := cg.Members()
		assert.Len(t, members, 1)
	})

	t.Run("add multiple members triggers rebalance", func(t *testing.T) {
		cg := NewConsumerGroup("rebalance-group", topic)

		cg.AddMember("member-1", "client-1")
		gen1 := cg.Generation()

		cg.AddMember("member-2", "client-2")
		gen2 := cg.Generation()

		assert.Greater(t, gen2, gen1)

		// Partitions should be distributed
		members := cg.Members()
		totalPartitions := 0
		for _, m := range members {
			totalPartitions += len(m.Partitions)
		}
		assert.Equal(t, 3, totalPartitions) // All partitions assigned
	})

	t.Run("remove member", func(t *testing.T) {
		cg := NewConsumerGroup("remove-group", topic)

		cg.AddMember("member-1", "client-1")
		cg.AddMember("member-2", "client-2")

		cg.RemoveMember("member-1")

		members := cg.Members()
		assert.Len(t, members, 1)
		assert.Equal(t, "member-2", members[0].MemberID)
	})
}

func TestTopicConsumerGroups(t *testing.T) {
	topic := NewTopic("multi-cg", 3, 1000)

	t.Run("get or create consumer group", func(t *testing.T) {
		cg1 := topic.GetOrCreateConsumerGroup("group-1")
		assert.NotNil(t, cg1)

		cg1Again := topic.GetOrCreateConsumerGroup("group-1")
		assert.Equal(t, cg1, cg1Again) // Same instance

		cg2 := topic.GetOrCreateConsumerGroup("group-2")
		assert.NotEqual(t, cg1, cg2)
	})

	t.Run("delete consumer group", func(t *testing.T) {
		topic.GetOrCreateConsumerGroup("to-delete")
		topic.DeleteConsumerGroup("to-delete")

		// Group should be recreated as new
		cg := topic.GetOrCreateConsumerGroup("to-delete")
		assert.Empty(t, cg.Members())
	})
}

func TestGetPartition(t *testing.T) {
	topic := NewTopic("get-partition", 3, 1000)

	t.Run("valid partition", func(t *testing.T) {
		p := topic.GetPartition(0)
		assert.NotNil(t, p)
		assert.Equal(t, int32(0), p.ID())

		p2 := topic.GetPartition(2)
		assert.NotNil(t, p2)
		assert.Equal(t, int32(2), p2.ID())
	})

	t.Run("invalid partition", func(t *testing.T) {
		p := topic.GetPartition(10)
		assert.Nil(t, p)
	})
}

func TestPartitionConsistentHashing(t *testing.T) {
	topic := NewTopic("hash", 100, 1000)

	// Test that same key always goes to same partition
	key := "test-key-12345"
	var partition int32 = -1

	for i := 0; i < 100; i++ {
		msg := &messaging.Message{
			ID:        "hash-" + string(rune('0'+i)),
			Type:      "test",
			Payload:   []byte("test"),
			Key:       key,
			Timestamp: time.Now(),
		}
		_ = topic.Publish(msg)

		if partition == -1 {
			partition = msg.Partition
		} else {
			assert.Equal(t, partition, msg.Partition, "Consistent hashing should always put same key in same partition")
		}
	}
}

func TestPartitionOffsets(t *testing.T) {
	partition := NewPartition(0, 1000)

	// Empty partition
	assert.Equal(t, int64(0), partition.BeginOffset())
	assert.Equal(t, int64(0), partition.EndOffset())

	// Add messages
	for i := 0; i < 5; i++ {
		msg := &messaging.Message{
			ID:        "offset-" + string(rune('0'+i)),
			Type:      "test",
			Payload:   []byte("test"),
			Timestamp: time.Now(),
		}
		err := partition.Append(msg)
		require.NoError(t, err)
	}

	assert.Equal(t, int64(0), partition.BeginOffset())
	assert.Equal(t, int64(5), partition.EndOffset())
}
