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

// WebSocketConfig holds configuration for the WebSocket server
type WebSocketConfig struct {
	AllowedOrigins    []string // List of allowed origins, empty means allow all (development only)
	ReadBufferSize    int
	WriteBufferSize   int
	HandshakeTimeout  time.Duration
	ReadDeadline      time.Duration
	WriteDeadline     time.Duration
	PingInterval      time.Duration
	MaxMessageSize    int64
	EnableCompression bool
}

// DefaultWebSocketConfig returns a secure default configuration
func DefaultWebSocketConfig() *WebSocketConfig {
	return &WebSocketConfig{
		AllowedOrigins:    []string{}, // Must be configured for production
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		HandshakeTimeout:  10 * time.Second,
		ReadDeadline:      60 * time.Second,
		WriteDeadline:     10 * time.Second,
		PingInterval:      54 * time.Second, // Before read deadline
		MaxMessageSize:    512 * 1024,       // 512KB max message
		EnableCompression: true,
	}
}

// WebSocketServer handles WebSocket connections for real-time event streaming
type WebSocketServer struct {
	eventManager   *EventManager
	upgrader       websocket.Upgrader
	connections    map[string]*WebSocketConnection
	connectionsMux sync.RWMutex
	server         *http.Server
	shutdownChan   chan struct{}
	config         *WebSocketConfig
	metrics        *WebSocketMetrics
}

// WebSocketMetrics tracks WebSocket server metrics
type WebSocketMetrics struct {
	mu                  sync.RWMutex
	TotalConnections    int64
	ActiveConnections   int64
	MessagesReceived    int64
	MessagesSent        int64
	ConnectionsRejected int64
	Errors              int64
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

// NewWebSocketServer creates a new WebSocket server with default configuration
func NewWebSocketServer(eventManager *EventManager, addr string) *WebSocketServer {
	return NewWebSocketServerWithConfig(eventManager, addr, DefaultWebSocketConfig())
}

// NewWebSocketServerWithConfig creates a new WebSocket server with custom configuration
func NewWebSocketServerWithConfig(eventManager *EventManager, addr string, config *WebSocketConfig) *WebSocketServer {
	if config == nil {
		config = DefaultWebSocketConfig()
	}

	server := &WebSocketServer{
		eventManager: eventManager,
		connections:  make(map[string]*WebSocketConnection),
		shutdownChan: make(chan struct{}),
		config:       config,
		metrics:      &WebSocketMetrics{},
	}

	upgrader := websocket.Upgrader{
		CheckOrigin:       server.checkOrigin,
		ReadBufferSize:    config.ReadBufferSize,
		WriteBufferSize:   config.WriteBufferSize,
		HandshakeTimeout:  config.HandshakeTimeout,
		EnableCompression: config.EnableCompression,
	}

	server.upgrader = upgrader

	// Set up HTTP routes
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", server.handleWebSocket)
	mux.HandleFunc("/health", server.handleHealth)
	mux.HandleFunc("/metrics", server.handleMetrics)

	server.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return server
}

// checkOrigin validates the origin of WebSocket connections
func (ws *WebSocketServer) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")

	// If no allowed origins configured, allow all (development mode)
	if len(ws.config.AllowedOrigins) == 0 {
		log.Printf("WARNING: WebSocket accepting connections from any origin (development mode)")
		return true
	}

	// Empty origin is typically from same-origin requests (non-browser)
	if origin == "" {
		return true
	}

	// Check against allowed origins
	for _, allowed := range ws.config.AllowedOrigins {
		if allowed == "*" {
			return true
		}
		if origin == allowed {
			return true
		}
		// Support wildcard subdomains (e.g., "https://*.example.com")
		if len(allowed) > 2 && allowed[:2] == "*." {
			suffix := allowed[1:] // ".example.com"
			if len(origin) > len(suffix) {
				// Check if origin ends with the suffix after the protocol
				originWithoutProtocol := origin
				if idx := findProtocolEnd(origin); idx > 0 {
					originWithoutProtocol = origin[idx:]
				}
				if len(originWithoutProtocol) > len(suffix) &&
					originWithoutProtocol[len(originWithoutProtocol)-len(suffix):] == suffix {
					return true
				}
			}
		}
	}

	log.Printf("WebSocket connection rejected: origin '%s' not in allowed list", origin)
	ws.metrics.mu.Lock()
	ws.metrics.ConnectionsRejected++
	ws.metrics.mu.Unlock()
	return false
}

// findProtocolEnd finds the position after "://" in a URL
func findProtocolEnd(url string) int {
	for i := 0; i < len(url)-2; i++ {
		if url[i:i+3] == "://" {
			return i + 3
		}
	}
	return 0
}

// AddAllowedOrigin adds an origin to the allowed list
func (ws *WebSocketServer) AddAllowedOrigin(origin string) {
	ws.config.AllowedOrigins = append(ws.config.AllowedOrigins, origin)
}

// SetAllowedOrigins sets the allowed origins list
func (ws *WebSocketServer) SetAllowedOrigins(origins []string) {
	ws.config.AllowedOrigins = origins
}

// handleMetrics provides metrics about the WebSocket server
func (ws *WebSocketServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	ws.metrics.mu.RLock()
	metrics := map[string]interface{}{
		"total_connections":    ws.metrics.TotalConnections,
		"active_connections":   ws.metrics.ActiveConnections,
		"messages_received":    ws.metrics.MessagesReceived,
		"messages_sent":        ws.metrics.MessagesSent,
		"connections_rejected": ws.metrics.ConnectionsRejected,
		"errors":               ws.metrics.Errors,
		"timestamp":            time.Now(),
	}
	ws.metrics.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
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
		supportedTypes = ws.parseEventTypes(supportedTypesStr)
	}

	// If no valid types parsed, use defaults
	if len(supportedTypes) == 0 {
		supportedTypes = []EventType{
			EventVerificationStarted,
			EventVerificationCompleted,
			EventVerificationFailed,
			EventScoreChanged,
			EventIssueDetected,
			EventIssueResolved,
			EventClientConnected,
			EventClientDisconnected,
			EventSystemHealthChanged,
		}
	}

	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		ws.metrics.mu.Lock()
		ws.metrics.Errors++
		ws.metrics.mu.Unlock()
		return
	}

	// Configure connection limits
	conn.SetReadLimit(ws.config.MaxMessageSize)

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

	// Update metrics
	ws.metrics.mu.Lock()
	ws.metrics.TotalConnections++
	ws.metrics.ActiveConnections++
	ws.metrics.mu.Unlock()

	// Subscribe to event manager
	if err := ws.eventManager.Subscribe(subscriber); err != nil {
		log.Printf("Failed to subscribe WebSocket client: %v", err)
		conn.Close()
		ws.metrics.mu.Lock()
		ws.metrics.ActiveConnections--
		ws.metrics.Errors++
		ws.metrics.mu.Unlock()
		return
	}

	log.Printf("WebSocket client connected: %s (types: %v)", connectionID, supportedTypes)

	// Publish client connected event
	ws.eventManager.PublishEvent(&Event{
		Type:     EventClientConnected,
		Severity: SeverityInfo,
		Title:    "WebSocket Client Connected",
		Message:  fmt.Sprintf("Client %s connected", connectionID),
		ClientID: &connectionID,
	})

	// Start goroutines for this connection
	go ws.handleIncomingMessages(wsConn)
	go ws.handleOutgoingEvents(wsConn)
}

// parseEventTypes parses comma-separated event types string
func (ws *WebSocketServer) parseEventTypes(typesStr string) []EventType {
	var eventTypes []EventType
	validTypes := map[string]EventType{
		"verification_started":   EventVerificationStarted,
		"verification_completed": EventVerificationCompleted,
		"verification_failed":    EventVerificationFailed,
		"score_changed":          EventScoreChanged,
		"model_added":            EventModelAdded,
		"model_removed":          EventModelRemoved,
		"provider_added":         EventProviderAdded,
		"provider_removed":       EventProviderRemoved,
		"issue_detected":         EventIssueDetected,
		"issue_resolved":         EventIssueResolved,
		"config_exported":        EventConfigExported,
		"database_migration":     EventDatabaseMigration,
		"client_connected":       EventClientConnected,
		"client_disconnected":    EventClientDisconnected,
		"system_health_changed":  EventSystemHealthChanged,
		"maintenance_mode":       EventMaintenanceMode,
		"backup_completed":       EventBackupCompleted,
		"security_alert":         EventSecurityAlert,
	}

	// Split by comma and trim
	for _, typeStr := range splitAndTrim(typesStr, ",") {
		if eventType, ok := validTypes[typeStr]; ok {
			eventTypes = append(eventTypes, eventType)
		}
	}

	return eventTypes
}

// splitAndTrim splits a string and trims whitespace from each part
func splitAndTrim(s, sep string) []string {
	var result []string
	parts := make([]string, 0)

	// Simple split implementation
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			parts = append(parts, s[start:i])
			start = i + len(sep)
		}
	}
	parts = append(parts, s[start:])

	// Trim each part
	for _, part := range parts {
		trimmed := trimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}

// trimSpace removes leading and trailing whitespace
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
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

		// Update metrics
		ws.metrics.mu.Lock()
		ws.metrics.ActiveConnections--
		ws.metrics.mu.Unlock()

		log.Printf("WebSocket client disconnected: %s", wsConn.ID)

		// Publish client disconnected event
		ws.eventManager.PublishEvent(&Event{
			Type:     EventClientDisconnected,
			Severity: SeverityInfo,
			Title:    "WebSocket Client Disconnected",
			Message:  fmt.Sprintf("Client %s disconnected", wsConn.ID),
			ClientID: &wsConn.ID,
		})
	}()

	wsConn.Conn.SetReadDeadline(time.Now().Add(ws.config.ReadDeadline))
	wsConn.Conn.SetPongHandler(func(string) error {
		wsConn.Conn.SetReadDeadline(time.Now().Add(ws.config.ReadDeadline))
		wsConn.LastActivity = time.Now()
		return nil
	})

	for {
		messageType, message, err := wsConn.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
				ws.metrics.mu.Lock()
				ws.metrics.Errors++
				ws.metrics.mu.Unlock()
			}
			break
		}

		wsConn.LastActivity = time.Now()
		ws.metrics.mu.Lock()
		ws.metrics.MessagesReceived++
		ws.metrics.mu.Unlock()

		// Handle ping messages
		if messageType == websocket.PingMessage {
			wsConn.Conn.WriteMessage(websocket.PongMessage, nil)
			continue
		}

		// Handle text messages (commands from client)
		if messageType == websocket.TextMessage {
			ws.handleClientMessage(wsConn, message)
		}
	}
}

// ClientMessage represents a message from a WebSocket client
type ClientMessage struct {
	Type    string                 `json:"type"`
	Action  string                 `json:"action"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

// handleClientMessage processes messages from clients
func (ws *WebSocketServer) handleClientMessage(wsConn *WebSocketConnection, message []byte) {
	var msg ClientMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Invalid message from client %s: %v", wsConn.ID, err)
		ws.sendError(wsConn, "invalid_message", "Failed to parse message")
		return
	}

	switch msg.Type {
	case "subscribe":
		// Handle subscription updates
		if typesRaw, ok := msg.Payload["types"].([]interface{}); ok {
			var newTypes []EventType
			for _, t := range typesRaw {
				if typeStr, ok := t.(string); ok {
					parsedTypes := ws.parseEventTypes(typeStr)
					newTypes = append(newTypes, parsedTypes...)
				}
			}
			if len(newTypes) > 0 {
				wsConn.SupportedTypes = newTypes
				wsConn.Subscriber.SupportedTypes = newTypes
				ws.sendAck(wsConn, "subscribed", map[string]interface{}{"types": newTypes})
			}
		}

	case "unsubscribe":
		// Handle unsubscription
		if typesRaw, ok := msg.Payload["types"].([]interface{}); ok {
			for _, t := range typesRaw {
				if typeStr, ok := t.(string); ok {
					parsedTypes := ws.parseEventTypes(typeStr)
					wsConn.SupportedTypes = removeEventTypes(wsConn.SupportedTypes, parsedTypes)
					wsConn.Subscriber.SupportedTypes = wsConn.SupportedTypes
				}
			}
			ws.sendAck(wsConn, "unsubscribed", map[string]interface{}{"types": wsConn.SupportedTypes})
		}

	case "ping":
		// Client-initiated ping
		ws.sendAck(wsConn, "pong", nil)

	case "status":
		// Return connection status
		ws.sendAck(wsConn, "status", map[string]interface{}{
			"connection_id":   wsConn.ID,
			"active":          wsConn.IsActive,
			"subscribed_types": wsConn.SupportedTypes,
			"last_activity":   wsConn.LastActivity,
		})

	default:
		log.Printf("Unknown message type from client %s: %s", wsConn.ID, msg.Type)
	}
}

// sendAck sends an acknowledgment message to the client
func (ws *WebSocketServer) sendAck(wsConn *WebSocketConnection, ackType string, data map[string]interface{}) {
	response := map[string]interface{}{
		"type":      "ack",
		"ack_type":  ackType,
		"timestamp": time.Now(),
	}
	if data != nil {
		response["data"] = data
	}

	if jsonData, err := json.Marshal(response); err == nil {
		wsConn.Conn.SetWriteDeadline(time.Now().Add(ws.config.WriteDeadline))
		wsConn.Conn.WriteMessage(websocket.TextMessage, jsonData)
	}
}

// sendError sends an error message to the client
func (ws *WebSocketServer) sendError(wsConn *WebSocketConnection, code, message string) {
	response := map[string]interface{}{
		"type":      "error",
		"code":      code,
		"message":   message,
		"timestamp": time.Now(),
	}

	if jsonData, err := json.Marshal(response); err == nil {
		wsConn.Conn.SetWriteDeadline(time.Now().Add(ws.config.WriteDeadline))
		wsConn.Conn.WriteMessage(websocket.TextMessage, jsonData)
	}
}

// removeEventTypes removes specified event types from a slice
func removeEventTypes(types []EventType, toRemove []EventType) []EventType {
	result := make([]EventType, 0)
	removeMap := make(map[EventType]bool)
	for _, t := range toRemove {
		removeMap[t] = true
	}
	for _, t := range types {
		if !removeMap[t] {
			result = append(result, t)
		}
	}
	return result
}

// handleOutgoingEvents sends events to the client
func (ws *WebSocketServer) handleOutgoingEvents(wsConn *WebSocketConnection) {
	ticker := time.NewTicker(ws.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case event := <-wsConn.Subscriber.ReceiveChannel:
			if !wsConn.IsActive {
				return
			}

			// Wrap event in a message envelope
			message := map[string]interface{}{
				"type":      "event",
				"event":     event,
				"timestamp": time.Now(),
			}

			// Convert to JSON
			eventJSON, err := json.Marshal(message)
			if err != nil {
				log.Printf("Failed to marshal event for WebSocket: %v", err)
				ws.metrics.mu.Lock()
				ws.metrics.Errors++
				ws.metrics.mu.Unlock()
				continue
			}

			// Send event to client with write deadline
			wsConn.Conn.SetWriteDeadline(time.Now().Add(ws.config.WriteDeadline))
			if err := wsConn.Conn.WriteMessage(websocket.TextMessage, eventJSON); err != nil {
				log.Printf("Failed to send event to WebSocket client %s: %v", wsConn.ID, err)
				ws.metrics.mu.Lock()
				ws.metrics.Errors++
				ws.metrics.mu.Unlock()
				wsConn.IsActive = false
				return
			}

			ws.metrics.mu.Lock()
			ws.metrics.MessagesSent++
			ws.metrics.mu.Unlock()
			wsConn.LastActivity = time.Now()

		case <-ticker.C:
			// Send ping to keep connection alive
			wsConn.Conn.SetWriteDeadline(time.Now().Add(ws.config.WriteDeadline))
			if err := wsConn.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("Failed to ping WebSocket client %s: %v", wsConn.ID, err)
				ws.metrics.mu.Lock()
				ws.metrics.Errors++
				ws.metrics.mu.Unlock()
				wsConn.IsActive = false
				return
			}

		case <-ws.shutdownChan:
			// Send close message before shutdown
			wsConn.Conn.SetWriteDeadline(time.Now().Add(ws.config.WriteDeadline))
			wsConn.Conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseGoingAway, "Server shutting down"))
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
