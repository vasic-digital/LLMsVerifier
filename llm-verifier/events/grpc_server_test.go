package events

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGRPCServer_Creation(t *testing.T) {
	ctx := context.Background()
	em := NewEventManager(ctx, 100, 2)
	defer em.Shutdown()

	server := NewGRPCServer(em)
	assert.NotNil(t, server)
	assert.NotNil(t, server.clients)
	assert.Equal(t, 0, server.GetClientCount())
}

func TestGRPCServer_StartStop(t *testing.T) {
	ctx := context.Background()
	em := NewEventManager(ctx, 100, 2)
	defer em.Shutdown()

	server := NewGRPCServer(em)

	// Start server
	err := server.Start(0) // Use port 0 for random available port
	require.NoError(t, err)
	assert.True(t, server.isRunning())

	// Double start should error
	err = server.Start(0)
	assert.Error(t, err)

	// Stop server
	server.Stop()
	assert.False(t, server.isRunning())

	// Double stop should not panic
	assert.NotPanics(t, func() { server.Stop() })
}

func TestGRPCServer_RegisterUnregisterClient(t *testing.T) {
	ctx := context.Background()
	em := NewEventManager(ctx, 100, 2)
	defer em.Shutdown()

	server := NewGRPCServer(em)

	eventTypes := []EventType{EventVerificationCompleted, EventIssueDetected}

	// Register client
	client, err := server.RegisterClient("client-1", eventTypes)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "client-1", client.ID)
	assert.Equal(t, 1, server.GetClientCount())

	// Register same client again should fail
	_, err = server.RegisterClient("client-1", eventTypes)
	assert.Error(t, err)

	// Register another client
	_, err = server.RegisterClient("client-2", eventTypes)
	require.NoError(t, err)
	assert.Equal(t, 2, server.GetClientCount())

	// Unregister client
	server.UnregisterClient("client-1")
	assert.Equal(t, 1, server.GetClientCount())

	// Unregister non-existent client should not panic
	assert.NotPanics(t, func() { server.UnregisterClient("non-existent") })

	// Cleanup
	server.UnregisterClient("client-2")
	assert.Equal(t, 0, server.GetClientCount())
}

func TestGRPCServer_GetClientInfo(t *testing.T) {
	ctx := context.Background()
	em := NewEventManager(ctx, 100, 2)
	defer em.Shutdown()

	server := NewGRPCServer(em)

	eventTypes := []EventType{EventVerificationCompleted}

	// Register client
	_, err := server.RegisterClient("info-client", eventTypes)
	require.NoError(t, err)

	// Get client info
	info, err := server.GetClientInfo("info-client")
	require.NoError(t, err)
	assert.Equal(t, "info-client", info["id"])
	assert.NotNil(t, info["connected_at"])
	assert.NotNil(t, info["last_activity"])
	assert.NotNil(t, info["event_types"])

	// Get non-existent client info
	_, err = server.GetClientInfo("non-existent")
	assert.Error(t, err)

	// Cleanup
	server.UnregisterClient("info-client")
}

func TestGRPCServer_GetAllClientsInfo(t *testing.T) {
	ctx := context.Background()
	em := NewEventManager(ctx, 100, 2)
	defer em.Shutdown()

	server := NewGRPCServer(em)

	// No clients initially
	clients := server.GetAllClientsInfo()
	assert.Len(t, clients, 0)

	// Register multiple clients
	server.RegisterClient("client-a", []EventType{EventVerificationCompleted})
	server.RegisterClient("client-b", []EventType{EventIssueDetected})
	server.RegisterClient("client-c", []EventType{})

	clients = server.GetAllClientsInfo()
	assert.Len(t, clients, 3)

	// Cleanup
	server.UnregisterClient("client-a")
	server.UnregisterClient("client-b")
	server.UnregisterClient("client-c")
}

func TestGRPCServer_BroadcastEvent(t *testing.T) {
	ctx := context.Background()
	em := NewEventManager(ctx, 100, 2)
	defer em.Shutdown()

	server := NewGRPCServer(em)

	// Register clients with different event type interests
	server.RegisterClient("all-events", []EventType{}) // All events
	server.RegisterClient("verification-only", []EventType{EventVerificationCompleted})
	server.RegisterClient("issue-only", []EventType{EventIssueDetected})

	// Broadcast verification event
	verificationEvent := &Event{
		ID:      "test-1",
		Type:    EventVerificationCompleted,
		Title:   "Test Verification",
		Message: "Verification completed",
	}

	sent := server.BroadcastEvent(verificationEvent)
	assert.Equal(t, 2, sent) // all-events and verification-only

	// Broadcast issue event
	issueEvent := &Event{
		ID:      "test-2",
		Type:    EventIssueDetected,
		Title:   "Test Issue",
		Message: "Issue detected",
	}

	sent = server.BroadcastEvent(issueEvent)
	assert.Equal(t, 2, sent) // all-events and issue-only

	// Cleanup
	server.UnregisterClient("all-events")
	server.UnregisterClient("verification-only")
	server.UnregisterClient("issue-only")
}

func TestGRPCServer_SerializeDeserializeEvent(t *testing.T) {
	event := &Event{
		ID:       "serialize-test",
		Type:     EventVerificationCompleted,
		Severity: SeverityInfo,
		Title:    "Serialization Test",
		Message:  "Testing event serialization",
		Details: map[string]interface{}{
			"score": 8.5,
			"model": "gpt-4",
		},
		Timestamp: time.Now(),
	}

	// Serialize
	data, err := SerializeEvent(event)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Deserialize
	deserialized, err := DeserializeEvent(data)
	require.NoError(t, err)
	assert.Equal(t, event.ID, deserialized.ID)
	assert.Equal(t, event.Type, deserialized.Type)
	assert.Equal(t, event.Title, deserialized.Title)
	assert.Equal(t, event.Message, deserialized.Message)
}

func TestGRPCServer_DeserializeInvalidEvent(t *testing.T) {
	invalidData := []byte("not valid json")
	_, err := DeserializeEvent(invalidData)
	assert.Error(t, err)
}

func TestGRPCServer_WithNilEventManager(t *testing.T) {
	server := NewGRPCServer(nil)
	assert.NotNil(t, server)

	// Should still work without event manager
	client, err := server.RegisterClient("no-em-client", []EventType{})
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, 1, server.GetClientCount())

	server.UnregisterClient("no-em-client")
	assert.Equal(t, 0, server.GetClientCount())
}
