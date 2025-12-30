// Package events contains event management functionality
// This file implements the gRPC server for event streaming
package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
)

// GRPCServer handles gRPC event streaming
type GRPCServer struct {
	eventManager *EventManager
	server       *grpc.Server
	listener     net.Listener
	clients      map[string]*grpcClient
	clientsMux   sync.RWMutex
	running      bool
	runningMux   sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
}

// grpcClient represents a connected gRPC client
type grpcClient struct {
	ID           string
	ConnectedAt  time.Time
	LastActivity time.Time
	EventTypes   []EventType
	EventChan    chan *Event
	Done         chan struct{}
}

// EventStreamRequest represents a request to stream events
type EventStreamRequest struct {
	ClientID   string      `json:"client_id"`
	EventTypes []EventType `json:"event_types"`
}

// EventStreamResponse represents a streamed event
type EventStreamResponse struct {
	Event     *Event `json:"event"`
	Timestamp int64  `json:"timestamp"`
}

// NewGRPCServer creates a new gRPC event server
func NewGRPCServer(eventManager *EventManager) *GRPCServer {
	ctx, cancel := context.WithCancel(context.Background())
	return &GRPCServer{
		eventManager: eventManager,
		clients:      make(map[string]*grpcClient),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start starts the gRPC server on the specified port
func (gs *GRPCServer) Start(port int) error {
	gs.runningMux.Lock()
	if gs.running {
		gs.runningMux.Unlock()
		return fmt.Errorf("gRPC server already running")
	}

	// Create listener
	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		gs.runningMux.Unlock()
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}
	gs.listener = listener

	// Configure gRPC server with keepalive
	serverOpts := []grpc.ServerOption{
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     5 * time.Minute,
			MaxConnectionAge:      30 * time.Minute,
			MaxConnectionAgeGrace: 5 * time.Second,
			Time:                  1 * time.Minute,
			Timeout:               20 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             30 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.MaxConcurrentStreams(100),
	}

	gs.server = grpc.NewServer(serverOpts...)
	gs.running = true
	gs.runningMux.Unlock()

	// Start accepting connections in a goroutine
	go func() {
		log.Printf("gRPC server listening on %s", address)
		if err := gs.server.Serve(listener); err != nil && gs.isRunning() {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	// Start client cleanup routine
	go gs.cleanupInactiveClients()

	return nil
}

// Stop stops the gRPC server gracefully
func (gs *GRPCServer) Stop() {
	gs.runningMux.Lock()
	if !gs.running {
		gs.runningMux.Unlock()
		return
	}
	gs.running = false
	gs.runningMux.Unlock()

	// Cancel context to stop all goroutines
	gs.cancel()

	// Close all client connections
	gs.clientsMux.Lock()
	for _, client := range gs.clients {
		close(client.Done)
	}
	gs.clients = make(map[string]*grpcClient)
	gs.clientsMux.Unlock()

	// Gracefully stop the server
	if gs.server != nil {
		gs.server.GracefulStop()
	}

	// Close the listener
	if gs.listener != nil {
		gs.listener.Close()
	}

	log.Println("gRPC server stopped")
}

// isRunning returns whether the server is running
func (gs *GRPCServer) isRunning() bool {
	gs.runningMux.RLock()
	defer gs.runningMux.RUnlock()
	return gs.running
}

// GetClientCount returns the number of active gRPC clients
func (gs *GRPCServer) GetClientCount() int {
	gs.clientsMux.RLock()
	defer gs.clientsMux.RUnlock()
	return len(gs.clients)
}

// RegisterClient registers a new client for event streaming
func (gs *GRPCServer) RegisterClient(clientID string, eventTypes []EventType) (*grpcClient, error) {
	gs.clientsMux.Lock()
	defer gs.clientsMux.Unlock()

	if _, exists := gs.clients[clientID]; exists {
		return nil, fmt.Errorf("client %s already registered", clientID)
	}

	client := &grpcClient{
		ID:           clientID,
		ConnectedAt:  time.Now(),
		LastActivity: time.Now(),
		EventTypes:   eventTypes,
		EventChan:    make(chan *Event, 100),
		Done:         make(chan struct{}),
	}

	gs.clients[clientID] = client

	// Register as event subscriber
	if gs.eventManager != nil {
		subscriber := NewGRPCSubscriber(clientID, eventTypes, func(event *Event) error {
			select {
			case client.EventChan <- event:
				client.LastActivity = time.Now()
				return nil
			case <-client.Done:
				return fmt.Errorf("client disconnected")
			default:
				return fmt.Errorf("event channel full")
			}
		})
		gs.eventManager.Subscribe(subscriber)
	}

	log.Printf("gRPC client registered: %s", clientID)
	return client, nil
}

// UnregisterClient removes a client from event streaming
func (gs *GRPCServer) UnregisterClient(clientID string) {
	gs.clientsMux.Lock()
	defer gs.clientsMux.Unlock()

	if client, exists := gs.clients[clientID]; exists {
		close(client.Done)
		delete(gs.clients, clientID)

		// Unregister from event manager
		if gs.eventManager != nil {
			gs.eventManager.Unsubscribe(fmt.Sprintf("grpc_%s", clientID))
		}

		log.Printf("gRPC client unregistered: %s", clientID)
	}
}

// StreamEvents streams events to a client
func (gs *GRPCServer) StreamEvents(clientID string, eventTypes []EventType, sendFunc func(*Event) error) error {
	client, err := gs.RegisterClient(clientID, eventTypes)
	if err != nil {
		return err
	}
	defer gs.UnregisterClient(clientID)

	for {
		select {
		case event := <-client.EventChan:
			if err := sendFunc(event); err != nil {
				return err
			}
		case <-client.Done:
			return nil
		case <-gs.ctx.Done():
			return status.Error(codes.Unavailable, "server shutting down")
		}
	}
}

// cleanupInactiveClients removes clients that haven't been active
func (gs *GRPCServer) cleanupInactiveClients() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			gs.clientsMux.Lock()
			now := time.Now()
			for id, client := range gs.clients {
				if now.Sub(client.LastActivity) > 10*time.Minute {
					close(client.Done)
					delete(gs.clients, id)
					if gs.eventManager != nil {
						gs.eventManager.Unsubscribe(fmt.Sprintf("grpc_%s", id))
					}
					log.Printf("Cleaned up inactive gRPC client: %s", id)
				}
			}
			gs.clientsMux.Unlock()
		case <-gs.ctx.Done():
			return
		}
	}
}

// BroadcastEvent sends an event to all connected clients
func (gs *GRPCServer) BroadcastEvent(event *Event) int {
	gs.clientsMux.RLock()
	defer gs.clientsMux.RUnlock()

	sent := 0
	for _, client := range gs.clients {
		// Check if client is interested in this event type
		interested := len(client.EventTypes) == 0 // Empty means all events
		for _, et := range client.EventTypes {
			if et == event.Type {
				interested = true
				break
			}
		}

		if interested {
			select {
			case client.EventChan <- event:
				sent++
			default:
				// Channel full, skip this client
			}
		}
	}

	return sent
}

// GetClientInfo returns information about a connected client
func (gs *GRPCServer) GetClientInfo(clientID string) (map[string]interface{}, error) {
	gs.clientsMux.RLock()
	defer gs.clientsMux.RUnlock()

	client, exists := gs.clients[clientID]
	if !exists {
		return nil, fmt.Errorf("client not found: %s", clientID)
	}

	return map[string]interface{}{
		"id":            client.ID,
		"connected_at":  client.ConnectedAt,
		"last_activity": client.LastActivity,
		"event_types":   client.EventTypes,
		"queue_size":    len(client.EventChan),
	}, nil
}

// GetAllClientsInfo returns information about all connected clients
func (gs *GRPCServer) GetAllClientsInfo() []map[string]interface{} {
	gs.clientsMux.RLock()
	defer gs.clientsMux.RUnlock()

	clients := make([]map[string]interface{}, 0, len(gs.clients))
	for _, client := range gs.clients {
		clients = append(clients, map[string]interface{}{
			"id":            client.ID,
			"connected_at":  client.ConnectedAt,
			"last_activity": client.LastActivity,
			"event_types":   client.EventTypes,
			"queue_size":    len(client.EventChan),
		})
	}

	return clients
}

// SerializeEvent converts an event to JSON for transmission
func SerializeEvent(event *Event) ([]byte, error) {
	return json.Marshal(event)
}

// DeserializeEvent converts JSON to an event
func DeserializeEvent(data []byte) (*Event, error) {
	var event Event
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}
