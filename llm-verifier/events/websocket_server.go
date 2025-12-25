package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketServer handles WebSocket connections for real-time event streaming
type WebSocketServer struct {
	eventManager   *EventManager
	upgrader       websocket.Upgrader
	connections    map[string]*WebSocketConnection
	connectionsMux sync.RWMutex
	server         *http.Server
	shutdownChan   chan struct{}
}

// WebSocketConnection represents a single WebSocket connection
type WebSocketConnection struct {
	ID             string
	Conn           *websocket.Conn
	Subscriber     *WebSocketSubscriber
	LastActivity   time.Time
	SupportedTypes []EventType
	IsActive       bool
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(eventManager *EventManager, addr string) *WebSocketServer {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			// Allow connections from any origin in development
			// In production, implement proper CORS checking
			return true
		},
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	server := &WebSocketServer{
		eventManager: eventManager,
		upgrader:     upgrader,
		connections:  make(map[string]*WebSocketConnection),
		shutdownChan: make(chan struct{}),
	}

	// Set up HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", server.handleWebSocket)
	mux.HandleFunc("/health", server.handleHealth)

	server.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return server
}

// Start starts the WebSocket server
func (ws *WebSocketServer) Start() error {
	log.Printf("Starting WebSocket server on %s", ws.server.Addr)

	// Start cleanup goroutine
	go ws.connectionCleanup()

	return ws.server.ListenAndServe()
}

// Stop stops the WebSocket server gracefully
func (ws *WebSocketServer) Stop(ctx context.Context) error {
	log.Println("Stopping WebSocket server...")

	// Close all connections
	ws.connectionsMux.Lock()
	for id, conn := range ws.connections {
		conn.Conn.Close()
		delete(ws.connections, id)
	}
	ws.connectionsMux.Unlock()

	close(ws.shutdownChan)

	return ws.server.Shutdown(ctx)
}

// handleWebSocket handles WebSocket upgrade requests
func (ws *WebSocketServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Get supported event types from query parameters
	supportedTypesStr := r.URL.Query().Get("types")
	var supportedTypes []EventType

	if supportedTypesStr != "" {
		// Parse comma-separated event types
		// Implementation would parse and validate types
		supportedTypes = []EventType{EventVerificationStarted, EventVerificationCompleted}
	} else {
		// Default to all types
		supportedTypes = []EventType{
			EventVerificationStarted,
			EventVerificationCompleted,
			EventVerificationFailed,
			EventScoreChanged,
			EventIssueDetected,
			EventIssueResolved,
		}
	}

	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	connectionID := fmt.Sprintf("ws_%d", time.Now().UnixNano())

	// Create WebSocket subscriber
	subscriber := NewWebSocketSubscriber(connectionID, supportedTypes)

	// Create connection wrapper
	wsConn := &WebSocketConnection{
		ID:             connectionID,
		Conn:           conn,
		Subscriber:     subscriber,
		LastActivity:   time.Now(),
		SupportedTypes: supportedTypes,
		IsActive:       true,
	}

	// Add to connections map
	ws.connectionsMux.Lock()
	ws.connections[connectionID] = wsConn
	ws.connectionsMux.Unlock()

	// Subscribe to event manager
	if err := ws.eventManager.Subscribe(subscriber); err != nil {
		log.Printf("Failed to subscribe WebSocket client: %v", err)
		conn.Close()
		return
	}

	log.Printf("WebSocket client connected: %s (types: %v)", connectionID, supportedTypes)

	// Start goroutines for this connection
	go ws.handleIncomingMessages(wsConn)
	go ws.handleOutgoingEvents(wsConn)
}

// handleIncomingMessages handles messages from the client
func (ws *WebSocketServer) handleIncomingMessages(wsConn *WebSocketConnection) {
	defer func() {
		wsConn.IsActive = false
		wsConn.Conn.Close()
		ws.eventManager.Unsubscribe(wsConn.Subscriber.GetID())

		ws.connectionsMux.Lock()
		delete(ws.connections, wsConn.ID)
		ws.connectionsMux.Unlock()

		log.Printf("WebSocket client disconnected: %s", wsConn.ID)
	}()

	wsConn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	wsConn.Conn.SetPongHandler(func(string) error {
		wsConn.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		wsConn.LastActivity = time.Now()
		return nil
	})

	for {
		messageType, message, err := wsConn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		wsConn.LastActivity = time.Now()

		// Handle ping messages
		if messageType == websocket.PingMessage {
			wsConn.Conn.WriteMessage(websocket.PongMessage, nil)
			continue
		}

		// Handle text messages (could be subscription updates, etc.)
		if messageType == websocket.TextMessage {
			log.Printf("Received message from client %s: %s", wsConn.ID, string(message))
			// Handle client messages here if needed
		}
	}
}

// handleOutgoingEvents sends events to the client
func (ws *WebSocketServer) handleOutgoingEvents(wsConn *WebSocketConnection) {
	ticker := time.NewTicker(54 * time.Second) // Send ping before read deadline
	defer ticker.Stop()

	for {
		select {
		case event := <-wsConn.Subscriber.ReceiveChannel:
			if !wsConn.IsActive {
				return
			}

			// Convert event to JSON
			eventJSON, err := json.Marshal(event)
			if err != nil {
				log.Printf("Failed to marshal event for WebSocket: %v", err)
				continue
			}

			// Send event to client
			if err := wsConn.Conn.WriteMessage(websocket.TextMessage, eventJSON); err != nil {
				log.Printf("Failed to send event to WebSocket client %s: %v", wsConn.ID, err)
				wsConn.IsActive = false
				return
			}

			wsConn.LastActivity = time.Now()

		case <-ticker.C:
			// Send ping to keep connection alive
			if err := wsConn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Failed to ping WebSocket client %s: %v", wsConn.ID, err)
				wsConn.IsActive = false
				return
			}

		case <-ws.shutdownChan:
			return
		}
	}
}

// handleHealth provides a health check endpoint
func (ws *WebSocketServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	ws.connectionsMux.RLock()
	activeConnections := 0
	for _, conn := range ws.connections {
		if conn.IsActive {
			activeConnections++
		}
	}
	ws.connectionsMux.RUnlock()

	health := map[string]interface{}{
		"status":             "healthy",
		"active_connections": activeConnections,
		"total_connections":  len(ws.connections),
		"timestamp":          time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// connectionCleanup periodically cleans up inactive connections
func (ws *WebSocketServer) connectionCleanup() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ws.connectionsMux.Lock()
			for id, conn := range ws.connections {
				if !conn.IsActive || time.Since(conn.LastActivity) > 5*time.Minute {
					log.Printf("Cleaning up inactive WebSocket connection: %s", id)
					conn.Conn.Close()
					delete(ws.connections, id)
				}
			}
			ws.connectionsMux.Unlock()

		case <-ws.shutdownChan:
			return
		}
	}
}

// GetConnectionCount returns the number of active connections
func (ws *WebSocketServer) GetConnectionCount() int {
	ws.connectionsMux.RLock()
	defer ws.connectionsMux.RUnlock()

	count := 0
	for _, conn := range ws.connections {
		if conn.IsActive {
			count++
		}
	}

	return count
}

// GetConnections returns information about all connections
func (ws *WebSocketServer) GetConnections() []map[string]interface{} {
	ws.connectionsMux.RLock()
	defer ws.connectionsMux.RUnlock()

	connections := make([]map[string]interface{}, 0, len(ws.connections))
	for id, conn := range ws.connections {
		connections = append(connections, map[string]interface{}{
			"id":              id,
			"active":          conn.IsActive,
			"last_activity":   conn.LastActivity,
			"supported_types": conn.SupportedTypes,
		})
	}

	return connections
}
