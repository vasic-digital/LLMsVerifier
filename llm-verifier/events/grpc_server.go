package events

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// GRPCServer handles gRPC connections for event streaming
type GRPCServer struct {
	eventManager *EventManager
	server       *grpc.Server
	port         string
	subscribers  map[string]*gRPCSubscriber
}

// NewGRPCServer creates a new gRPC server
func NewGRPCServer(eventManager *EventManager, port string) *GRPCServer {
	return &GRPCServer{
		eventManager: eventManager,
		port:         port,
		subscribers:  make(map[string]*gRPCSubscriber),
	}
}

// Start starts the gRPC server
func (gs *GRPCServer) Start() error {
	lis, err := net.Listen("tcp", gs.port)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", gs.port, err)
	}

	gs.server = grpc.NewServer()
	RegisterEventServiceServer(gs.server, gs)

	// Enable reflection for debugging
	reflection.Register(gs.server)

	log.Printf("Starting gRPC server on port %s", gs.port)
	return gs.server.Serve(lis)
}

// Stop stops the gRPC server
func (gs *GRPCServer) Stop() {
	if gs.server != nil {
		log.Println("Stopping gRPC server...")
		gs.server.GracefulStop()
	}
}

// Subscribe handles client subscription requests
func (gs *GRPCServer) Subscribe(ctx context.Context, req *SubscribeRequest) (*SubscribeResponse, error) {
	clientID := req.ClientId
	if clientID == "" {
		clientID = fmt.Sprintf("grpc_%d", time.Now().UnixNano())
	}

	// Convert string event types to EventType
	supportedTypes := make([]EventType, 0, len(req.SupportedEventTypes))
	for _, eventTypeStr := range req.SupportedEventTypes {
		eventType := EventType(eventTypeStr)
		supportedTypes = append(supportedTypes, eventType)
	}

	// Create gRPC subscriber
	subscriber := NewGRPCSubscriber(clientID, supportedTypes, func(event *Event) error {
		// Send event to client via stream
		// This would be implemented in the streaming method
		return nil
	})

	// Store subscriber
	gs.subscribers[clientID] = subscriber

	// Subscribe to event manager
	if err := gs.eventManager.Subscribe(subscriber); err != nil {
		delete(gs.subscribers, clientID)
		return nil, fmt.Errorf("failed to subscribe client: %w", err)
	}

	log.Printf("gRPC client subscribed: %s (types: %v)", clientID, supportedTypes)

	return &SubscribeResponse{
		ClientId:       clientID,
		Status:         "subscribed",
		SupportedTypes: req.SupportedEventTypes,
	}, nil
}

// Unsubscribe handles client unsubscription requests
func (gs *GRPCServer) Unsubscribe(ctx context.Context, req *UnsubscribeRequest) (*UnsubscribeResponse, error) {
	clientID := req.ClientId

	if subscriber, exists := gs.subscribers[clientID]; exists {
		gs.eventManager.Unsubscribe(subscriber.GetID())
		delete(gs.subscribers, clientID)

		log.Printf("gRPC client unsubscribed: %s", clientID)

		return &UnsubscribeResponse{
			ClientId: clientID,
			Status:   "unsubscribed",
		}, nil
	}

	return nil, fmt.Errorf("client not found: %s", clientID)
}

// StreamEvents provides a streaming endpoint for events
func (gs *GRPCServer) StreamEvents(req *StreamRequest, stream EventService_StreamEventsServer) error {
	clientID := req.ClientId

	subscriber, exists := gs.subscribers[clientID]
	if !exists {
		return fmt.Errorf("client not subscribed: %s", clientID)
	}

	log.Printf("Starting event stream for client: %s", clientID)

	// Update subscriber callback to send events via stream
	subscriber.Callback = func(event *Event) error {
		eventProto := &EventProto{
			Id:             event.ID,
			Type:           string(event.Type),
			Severity:       string(event.Severity),
			Title:          event.Title,
			Message:        event.Message,
			Source:         event.Source,
			Timestamp:      event.Timestamp.Unix(),
			ModelId:        event.ModelID,
			ProviderId:     event.ProviderID,
			VerificationId: event.VerificationID,
			IssueId:        event.IssueID,
			ClientId:       event.ClientID,
			UserId:         event.UserID,
		}

		// Marshal details to JSON
		if event.Details != nil {
			detailsJSON, err := json.Marshal(event.Details)
			if err == nil {
				eventProto.Details = string(detailsJSON)
			}
		}

		return stream.Send(eventProto)
	}

	// Keep stream alive until client disconnects or server shuts down
	<-stream.Context().Done()

	log.Printf("Event stream ended for client: %s", clientID)
	return stream.Context().Err()
}

// GetSubscriberCount returns the number of active gRPC subscribers
func (gs *GRPCServer) GetSubscriberCount() int {
	activeCount := 0
	for _, subscriber := range gs.subscribers {
		if subscriber.IsActive() {
			activeCount++
		}
	}
	return activeCount
}

// GetSubscribers returns information about all gRPC subscribers
func (gs *GRPCServer) GetSubscribers() []map[string]interface{} {
	subscribers := make([]map[string]interface{}, 0, len(gs.subscribers))
	for id, subscriber := range gs.subscribers {
		subscribers = append(subscribers, map[string]interface{}{
			"id":              id,
			"active":          subscriber.IsActive(),
			"last_activity":   subscriber.LastActivity,
			"supported_types": subscriber.SupportedTypes,
		})
	}
	return subscribers
}

// CleanInactiveSubscribers removes inactive subscribers
func (gs *GRPCServer) CleanInactiveSubscribers() {
	for id, subscriber := range gs.subscribers {
		if !subscriber.IsActive() {
			log.Printf("Cleaning up inactive gRPC subscriber: %s", id)
			gs.eventManager.Unsubscribe(subscriber.GetID())
			delete(gs.subscribers, id)
		}
	}
}
