package events

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EventType defines the type of system event
type EventType string

const (
	EventVerificationStarted   EventType = "verification_started"
	EventVerificationCompleted EventType = "verification_completed"
	EventVerificationFailed    EventType = "verification_failed"
	EventScoreChanged          EventType = "score_changed"
	EventModelAdded            EventType = "model_added"
	EventModelRemoved          EventType = "model_removed"
	EventProviderAdded         EventType = "provider_added"
	EventProviderRemoved       EventType = "provider_removed"
	EventIssueDetected         EventType = "issue_detected"
	EventIssueResolved         EventType = "issue_resolved"
	EventConfigExported        EventType = "config_exported"
	EventDatabaseMigration     EventType = "database_migration"
	EventClientConnected       EventType = "client_connected"
	EventClientDisconnected    EventType = "client_disconnected"
	EventSystemHealthChanged   EventType = "system_health_changed"
	EventMaintenanceMode       EventType = "maintenance_mode"
	EventBackupCompleted       EventType = "backup_completed"
	EventSecurityAlert         EventType = "security_alert"
)

// Severity defines the severity level of an event
type Severity string

const (
	SeverityDebug    Severity = "debug"
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
)

// Event represents a system event
type Event struct {
	ID             string                 `json:"id"`
	Type           EventType              `json:"type"`
	Severity       Severity               `json:"severity"`
	Title          string                 `json:"title"`
	Message        string                 `json:"message"`
	Details        map[string]interface{} `json:"details,omitempty"`
	ModelID        *int64                 `json:"model_id,omitempty"`
	ProviderID     *int64                 `json:"provider_id,omitempty"`
	VerificationID *int64                 `json:"verification_id,omitempty"`
	IssueID        *int64                 `json:"issue_id,omitempty"`
	ClientID       *string                `json:"client_id,omitempty"`
	UserID         *int64                 `json:"user_id,omitempty"`
	Source         string                 `json:"source"` // Component that generated the event
	Timestamp      time.Time              `json:"timestamp"`
	ProcessedAt    *time.Time             `json:"processed_at,omitempty"`
}

// EventSubscriber represents a component that can receive events
type EventSubscriber interface {
	ReceiveEvent(event *Event) error
	GetID() string
	GetSupportedEventTypes() []EventType
	IsActive() bool
}

// WebSocketSubscriber represents a WebSocket client subscriber
type WebSocketSubscriber struct {
	ID             string
	ConnectionID   string
	SupportedTypes []EventType
	LastActivity   time.Time
	ReceiveChannel chan *Event
	IsConnected    bool
}

// NewWebSocketSubscriber creates a new WebSocket subscriber
func NewWebSocketSubscriber(connectionID string, supportedTypes []EventType) *WebSocketSubscriber {
	return &WebSocketSubscriber{
		ID:             fmt.Sprintf("ws_%s", connectionID),
		ConnectionID:   connectionID,
		SupportedTypes: supportedTypes,
		LastActivity:   time.Now(),
		ReceiveChannel: make(chan *Event, 100), // Buffered channel
		IsConnected:    true,
	}
}

// ReceiveEvent implements EventSubscriber interface
func (ws *WebSocketSubscriber) ReceiveEvent(event *Event) error {
	if !ws.IsConnected {
		return fmt.Errorf("websocket subscriber is not connected")
	}

	select {
	case ws.ReceiveChannel <- event:
		ws.LastActivity = time.Now()
		return nil
	default:
		return fmt.Errorf("subscriber channel is full, event dropped")
	}
}

// GetID implements EventSubscriber interface
func (ws *WebSocketSubscriber) GetID() string {
	return ws.ID
}

// GetSupportedEventTypes implements EventSubscriber interface
func (ws *WebSocketSubscriber) GetSupportedEventTypes() []EventType {
	return ws.SupportedTypes
}

// IsActive implements EventSubscriber interface
func (ws *WebSocketSubscriber) IsActive() bool {
	return ws.IsConnected && time.Since(ws.LastActivity) < 5*time.Minute
}

// gRPCSubscriber represents a gRPC client subscriber
type gRPCSubscriber struct {
	ID             string
	ClientID       string
	SupportedTypes []EventType
	LastActivity   time.Time
	IsActiveConn   bool
	Callback       func(*Event) error
}

// NewGRPCSubscriber creates a new gRPC subscriber
func NewGRPCSubscriber(clientID string, supportedTypes []EventType, callback func(*Event) error) *gRPCSubscriber {
	return &gRPCSubscriber{
		ID:             fmt.Sprintf("grpc_%s", clientID),
		ClientID:       clientID,
		SupportedTypes: supportedTypes,
		LastActivity:   time.Now(),
		IsActiveConn:   true,
		Callback:       callback,
	}
}

// ReceiveEvent implements EventSubscriber interface
func (grpc *gRPCSubscriber) ReceiveEvent(event *Event) error {
	if !grpc.IsActiveConn || grpc.Callback == nil {
		return fmt.Errorf("grpc subscriber is not active")
	}

	err := grpc.Callback(event)
	if err == nil {
		grpc.LastActivity = time.Now()
	}
	return err
}

// GetID implements EventSubscriber interface
func (grpc *gRPCSubscriber) GetID() string {
	return grpc.ID
}

// GetSupportedEventTypes implements EventSubscriber interface
func (grpc *gRPCSubscriber) GetSupportedEventTypes() []EventType {
	return grpc.SupportedTypes
}

// IsActive implements EventSubscriber interface
func (grpc *gRPCSubscriber) IsActive() bool {
	return grpc.IsActiveConn && time.Since(grpc.LastActivity) < 10*time.Minute
}

// NotificationSubscriber represents a notification service subscriber (Slack, Email, etc.)
type NotificationSubscriber struct {
	ID             string
	ServiceType    string // "slack", "email", "telegram", etc.
	SupportedTypes []EventType
	MinSeverity    Severity
	Config         map[string]interface{} // API keys, webhook URLs, etc.
	LastActivity   time.Time
	IsEnabled      bool
}

// NewNotificationSubscriber creates a new notification subscriber
func NewNotificationSubscriber(serviceType string, supportedTypes []EventType, minSeverity Severity, config map[string]interface{}) *NotificationSubscriber {
	return &NotificationSubscriber{
		ID:             fmt.Sprintf("notify_%s_%d", serviceType, time.Now().Unix()),
		ServiceType:    serviceType,
		SupportedTypes: supportedTypes,
		MinSeverity:    minSeverity,
		Config:         config,
		LastActivity:   time.Now(),
		IsEnabled:      true,
	}
}

// ReceiveEvent implements EventSubscriber interface
func (ns *NotificationSubscriber) ReceiveEvent(event *Event) error {
	if !ns.IsEnabled {
		return fmt.Errorf("notification subscriber is disabled")
	}

	// Check if severity meets minimum threshold
	if getSeverityLevel(event.Severity) < getSeverityLevel(ns.MinSeverity) {
		return nil // Skip events below minimum severity
	}

	// Send notification based on service type
	return ns.sendNotification(event)
}

// sendNotification sends the actual notification
func (ns *NotificationSubscriber) sendNotification(event *Event) error {
	switch ns.ServiceType {
	case "slack":
		return ns.sendSlackNotification(event)
	case "email":
		return ns.sendEmailNotification(event)
	case "telegram":
		return ns.sendTelegramNotification(event)
	case "matrix":
		return ns.sendMatrixNotification(event)
	case "whatsapp":
		return ns.sendWhatsAppNotification(event)
	default:
		return fmt.Errorf("unsupported notification service: %s", ns.ServiceType)
	}
}

func (ns *NotificationSubscriber) sendSlackNotification(event *Event) error {
	webhookURL, ok := ns.Config["webhook_url"].(string)
	if !ok {
		return fmt.Errorf("slack webhook_url not configured")
	}

	channel, _ := ns.Config["channel"].(string)
	username, _ := ns.Config["username"].(string)

	notifier := NewSlackNotifier(webhookURL, channel, username)
	return notifier.SendNotification(event)
}

func (ns *NotificationSubscriber) sendEmailNotification(event *Event) error {
	smtpServer, ok := ns.Config["smtp_server"].(string)
	if !ok {
		return fmt.Errorf("email smtp_server not configured")
	}

	smtpPortFloat, ok := ns.Config["smtp_port"].(float64)
	if !ok {
		return fmt.Errorf("email smtp_port not configured")
	}
	smtpPort := int(smtpPortFloat)

	username, ok := ns.Config["username"].(string)
	if !ok {
		return fmt.Errorf("email username not configured")
	}

	password, ok := ns.Config["password"].(string)
	if !ok {
		return fmt.Errorf("email password not configured")
	}

	fromAddress, ok := ns.Config["from_address"].(string)
	if !ok {
		return fmt.Errorf("email from_address not configured")
	}

	toAddressesInterface, ok := ns.Config["to_addresses"]
	if !ok {
		return fmt.Errorf("email to_addresses not configured")
	}

	var toAddresses []string
	if toAddressesSlice, ok := toAddressesInterface.([]interface{}); ok {
		for _, addr := range toAddressesSlice {
			if addrStr, ok := addr.(string); ok {
				toAddresses = append(toAddresses, addrStr)
			}
		}
	}

	notifier := NewEmailNotifier(smtpServer, smtpPort, username, password, fromAddress, toAddresses)
	return notifier.SendNotification(event)
}

func (ns *NotificationSubscriber) sendTelegramNotification(event *Event) error {
	botToken, ok := ns.Config["bot_token"].(string)
	if !ok {
		return fmt.Errorf("telegram bot_token not configured")
	}

	chatID, ok := ns.Config["chat_id"].(string)
	if !ok {
		return fmt.Errorf("telegram chat_id not configured")
	}

	notifier := NewTelegramNotifier(botToken, chatID)
	return notifier.SendNotification(event)
}

func (ns *NotificationSubscriber) sendMatrixNotification(event *Event) error {
	homeserverURL, ok := ns.Config["homeserver_url"].(string)
	if !ok {
		return fmt.Errorf("matrix homeserver_url not configured")
	}

	accessToken, ok := ns.Config["access_token"].(string)
	if !ok {
		return fmt.Errorf("matrix access_token not configured")
	}

	roomID, ok := ns.Config["room_id"].(string)
	if !ok {
		return fmt.Errorf("matrix room_id not configured")
	}

	notifier := NewMatrixNotifier(homeserverURL, accessToken, roomID)
	return notifier.SendNotification(event)
}

func (ns *NotificationSubscriber) sendWhatsAppNotification(event *Event) error {
	accountSID, ok := ns.Config["account_sid"].(string)
	if !ok {
		return fmt.Errorf("whatsapp account_sid not configured")
	}

	authToken, ok := ns.Config["auth_token"].(string)
	if !ok {
		return fmt.Errorf("whatsapp auth_token not configured")
	}

	fromNumber, ok := ns.Config["from_number"].(string)
	if !ok {
		return fmt.Errorf("whatsapp from_number not configured")
	}

	toNumbersInterface, ok := ns.Config["to_numbers"]
	if !ok {
		return fmt.Errorf("whatsapp to_numbers not configured")
	}

	var toNumbers []string
	if toNumbersSlice, ok := toNumbersInterface.([]interface{}); ok {
		for _, num := range toNumbersSlice {
			if numStr, ok := num.(string); ok {
				toNumbers = append(toNumbers, numStr)
			}
		}
	}

	notifier := NewWhatsAppNotifier(accountSID, authToken, fromNumber, toNumbers)
	return notifier.SendNotification(event)
}

// GetID implements EventSubscriber interface
func (ns *NotificationSubscriber) GetID() string {
	return ns.ID
}

// GetSupportedEventTypes implements EventSubscriber interface
func (ns *NotificationSubscriber) GetSupportedEventTypes() []EventType {
	return ns.SupportedTypes
}

// IsActive implements EventSubscriber interface
func (ns *NotificationSubscriber) IsActive() bool {
	return ns.IsEnabled && time.Since(ns.LastActivity) < 24*time.Hour
}

// getSeverityLevel returns numeric severity level for comparison
func getSeverityLevel(severity Severity) int {
	switch severity {
	case SeverityDebug:
		return 0
	case SeverityInfo:
		return 1
	case SeverityWarning:
		return 2
	case SeverityError:
		return 3
	case SeverityCritical:
		return 4
	default:
		return 1
	}
}

// EventManager manages event publishing and subscriptions
type EventManager struct {
	subscribers    map[string]EventSubscriber
	subscribersMux sync.RWMutex
	eventBuffer    chan *Event
	bufferSize     int
	workers        int
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// NewEventManager creates a new event manager
func NewEventManager(ctx context.Context, bufferSize int, workers int) *EventManager {
	ctx, cancel := context.WithCancel(ctx)

	em := &EventManager{
		subscribers: make(map[string]EventSubscriber),
		eventBuffer: make(chan *Event, bufferSize),
		bufferSize:  bufferSize,
		workers:     workers,
		ctx:         ctx,
		cancel:      cancel,
	}

	// Start worker goroutines
	for i := 0; i < workers; i++ {
		em.wg.Add(1)
		go em.eventProcessor(i)
	}

	return em
}

// PublishEvent publishes an event to all interested subscribers
func (em *EventManager) PublishEvent(event *Event) error {
	if event == nil {
		return fmt.Errorf("cannot publish nil event")
	}

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Generate ID if not set
	if event.ID == "" {
		event.ID = fmt.Sprintf("evt_%d_%s", event.Timestamp.Unix(), event.Type)
	}

	// Set source if not set
	if event.Source == "" {
		event.Source = "system"
	}

	select {
	case em.eventBuffer <- event:
		return nil
	case <-em.ctx.Done():
		return fmt.Errorf("event manager is shutting down")
	default:
		return fmt.Errorf("event buffer is full, event dropped")
	}
}

// Subscribe adds a new subscriber
func (em *EventManager) Subscribe(subscriber EventSubscriber) error {
	em.subscribersMux.Lock()
	defer em.subscribersMux.Unlock()

	subscriberID := subscriber.GetID()
	if _, exists := em.subscribers[subscriberID]; exists {
		return fmt.Errorf("subscriber with ID %s already exists", subscriberID)
	}

	em.subscribers[subscriberID] = subscriber
	return nil
}

// Unsubscribe removes a subscriber
func (em *EventManager) Unsubscribe(subscriberID string) {
	em.subscribersMux.Lock()
	defer em.subscribersMux.Unlock()

	delete(em.subscribers, subscriberID)
}

// GetSubscribers returns all active subscribers
func (em *EventManager) GetSubscribers() []EventSubscriber {
	em.subscribersMux.RLock()
	defer em.subscribersMux.RUnlock()

	subscribers := make([]EventSubscriber, 0, len(em.subscribers))
	for _, subscriber := range em.subscribers {
		if subscriber.IsActive() {
			subscribers = append(subscribers, subscriber)
		}
	}

	return subscribers
}

// GetSubscriberCount returns the count of active subscribers
func (em *EventManager) GetSubscriberCount() int {
	em.subscribersMux.RLock()
	defer em.subscribersMux.RUnlock()

	count := 0
	for _, subscriber := range em.subscribers {
		if subscriber.IsActive() {
			count++
		}
	}

	return count
}

// Shutdown gracefully shuts down the event manager
func (em *EventManager) Shutdown() error {
	em.cancel()
	close(em.eventBuffer)
	em.wg.Wait()

	em.subscribersMux.Lock()
	em.subscribers = make(map[string]EventSubscriber)
	em.subscribersMux.Unlock()

	return nil
}

// eventProcessor processes events from the buffer
func (em *EventManager) eventProcessor(workerID int) {
	defer em.wg.Done()

	for {
		select {
		case event, ok := <-em.eventBuffer:
			if !ok {
				// Channel closed, exit
				return
			}

			em.processEvent(event)

		case <-em.ctx.Done():
			return
		}
	}
}

// processEvent distributes an event to interested subscribers
func (em *EventManager) processEvent(event *Event) {
	em.subscribersMux.RLock()
	defer em.subscribersMux.RUnlock()

	delivered := 0
	failed := 0

	for _, subscriber := range em.subscribers {
		if !subscriber.IsActive() {
			continue
		}

		// Check if subscriber is interested in this event type
		interested := false
		for _, eventType := range subscriber.GetSupportedEventTypes() {
			if eventType == event.Type {
				interested = true
				break
			}
		}

		if interested {
			if err := subscriber.ReceiveEvent(event); err != nil {
				failed++
			} else {
				delivered++
			}
		}
	}

	// Mark event as processed
	now := time.Now()
	event.ProcessedAt = &now
}

// CreateEvent creates a new event with the given parameters
func CreateEvent(eventType EventType, severity Severity, title, message string) *Event {
	return &Event{
		Type:      eventType,
		Severity:  severity,
		Title:     title,
		Message:   message,
		Timestamp: time.Now(),
		Source:    "system",
	}
}

// CreateEventWithDetails creates a new event with additional details
func CreateEventWithDetails(eventType EventType, severity Severity, title, message string, details map[string]interface{}) *Event {
	event := CreateEvent(eventType, severity, title, message)
	event.Details = details
	return event
}

// CreateModelEvent creates an event related to a specific model
func CreateModelEvent(eventType EventType, severity Severity, title, message string, modelID int64) *Event {
	event := CreateEvent(eventType, severity, title, message)
	event.ModelID = &modelID
	return event
}

// CreateProviderEvent creates an event related to a specific provider
func CreateProviderEvent(eventType EventType, severity Severity, title, message string, providerID int64) *Event {
	event := CreateEvent(eventType, severity, title, message)
	event.ProviderID = &providerID
	return event
}

// CreateVerificationEvent creates an event related to verification
func CreateVerificationEvent(eventType EventType, severity Severity, title, message string, verificationID int64) *Event {
	event := CreateEvent(eventType, severity, title, message)
	event.VerificationID = &verificationID
	return event
}

// CreateClientEvent creates an event related to a client
func CreateClientEvent(eventType EventType, severity Severity, title, message string, clientID string) *Event {
	event := CreateEvent(eventType, severity, title, message)
	event.ClientID = &clientID
	return event
}
