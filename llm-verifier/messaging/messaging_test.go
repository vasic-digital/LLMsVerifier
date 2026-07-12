package messaging

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockBroker is a mock message broker for testing.
type mockBroker struct {
	connected    bool
	publishCalls []publishCall
	publishErr   error
	connectErr   error
	closeErr     error
	mu           sync.Mutex
}

type publishCall struct {
	topic   string
	message []byte
}

func (m *mockBroker) Connect(ctx context.Context) error {
	if m.connectErr != nil {
		return m.connectErr
	}
	m.connected = true
	return nil
}

func (m *mockBroker) Close(ctx context.Context) error {
	if m.closeErr != nil {
		return m.closeErr
	}
	m.connected = false
	return nil
}

func (m *mockBroker) Publish(ctx context.Context, topic string, message []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.publishErr != nil {
		return m.publishErr
	}
	m.publishCalls = append(m.publishCalls, publishCall{topic: topic, message: message})
	return nil
}

func (m *mockBroker) IsConnected() bool {
	return m.connected
}

func (m *mockBroker) BrokerType() BrokerType {
	return BrokerTypeKafka
}

func (m *mockBroker) GetPublishCalls() []publishCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.publishCalls
}

// Event Tests

func TestNewVerificationEvent(t *testing.T) {
	event := NewVerificationEvent(
		EventVerificationStarted,
		SeverityInfo,
		"Test Event",
		"Test message",
	)

	assert.NotEmpty(t, event.ID)
	assert.Equal(t, EventVerificationStarted, event.Type)
	assert.Equal(t, SeverityInfo, event.Severity)
	assert.Equal(t, "Test Event", event.Title)
	assert.Equal(t, "Test message", event.Message)
	assert.Equal(t, "llmsverifier", event.Source)
	assert.NotZero(t, event.Timestamp)
	assert.NotNil(t, event.Details)
}

func TestVerificationEvent_WithProvider(t *testing.T) {
	event := NewVerificationEvent(EventProviderScored, SeverityInfo, "Test", "Test").
		WithProvider("provider-123")

	assert.Equal(t, "provider-123", event.ProviderID)
	assert.Equal(t, "provider-123", event.Subject)
}

func TestVerificationEvent_WithModel(t *testing.T) {
	event := NewVerificationEvent(EventModelVerified, SeverityInfo, "Test", "Test").
		WithModel("model-456")

	assert.Equal(t, "model-456", event.ModelID)
	assert.Equal(t, "model-456", event.Subject)
}

func TestVerificationEvent_WithScore(t *testing.T) {
	event := NewVerificationEvent(EventProviderScored, SeverityInfo, "Test", "Test").
		WithScore(8.5)

	assert.Equal(t, 8.5, event.Score)
}

func TestVerificationEvent_WithTraceID(t *testing.T) {
	event := NewVerificationEvent(EventVerificationStarted, SeverityInfo, "Test", "Test").
		WithTraceID("trace-789")

	assert.Equal(t, "trace-789", event.TraceID)
}

func TestVerificationEvent_WithDetails(t *testing.T) {
	details := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}
	event := NewVerificationEvent(EventVerificationStarted, SeverityInfo, "Test", "Test").
		WithDetails(details)

	assert.Equal(t, details, event.Details)
}

func TestVerificationEvent_AddDetail(t *testing.T) {
	event := NewVerificationEvent(EventVerificationStarted, SeverityInfo, "Test", "Test").
		AddDetail("key1", "value1").
		AddDetail("key2", 123)

	assert.Equal(t, "value1", event.Details["key1"])
	assert.Equal(t, 123, event.Details["key2"])
}

// Config Tests

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.False(t, cfg.Enabled)
	assert.Equal(t, BrokerTypeNone, cfg.BrokerType)
	assert.Equal(t, 30*time.Second, cfg.PublishTimeout)
	assert.True(t, cfg.RetryOnError)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, 1*time.Second, cfg.RetryDelay)
	assert.True(t, cfg.AsyncPublish)
	assert.Equal(t, 1000, cfg.BufferSize)
}

func TestDefaultKafkaConfig(t *testing.T) {
	cfg := DefaultKafkaConfig()

	assert.Equal(t, []string{"localhost:9092"}, cfg.Brokers)
	assert.Equal(t, "llmsverifier.events", cfg.Topic)
	assert.Equal(t, "llmsverifier", cfg.ClientID)
	assert.Equal(t, "llmsverifier-group", cfg.GroupID)
	assert.False(t, cfg.TLS)
	assert.False(t, cfg.SASLEnabled)
	assert.Equal(t, "PLAIN", cfg.SASLMechanism)
	assert.Equal(t, "lz4", cfg.Compression)
	assert.Equal(t, 100, cfg.BatchSize)
	assert.Equal(t, 10*time.Millisecond, cfg.BatchTimeout)
}

func TestDefaultRabbitMQConfig(t *testing.T) {
	cfg := DefaultRabbitMQConfig()

	assert.Equal(t, "localhost", cfg.Host)
	assert.Equal(t, 5672, cfg.Port)
	assert.Equal(t, "guest", cfg.Username)
	assert.Equal(t, "guest", cfg.Password)
	assert.Equal(t, "/", cfg.VirtualHost)
	assert.Equal(t, "llmsverifier.events", cfg.Exchange)
	assert.Equal(t, "topic", cfg.ExchangeType)
	assert.Equal(t, "verification.#", cfg.RoutingKey)
	assert.False(t, cfg.TLS)
	assert.True(t, cfg.PublisherConfirm)
	assert.Equal(t, 30*time.Second, cfg.ConnectTimeout)
	assert.Equal(t, 60*time.Second, cfg.Heartbeat)
}

func TestConfig_Validate_Disabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false
	assert.NoError(t, cfg.Validate())
}

func TestConfig_Validate_Kafka(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.BrokerType = BrokerTypeKafka
	assert.NoError(t, cfg.Validate())
}

func TestConfig_Validate_RabbitMQ(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.BrokerType = BrokerTypeRabbitMQ
	assert.NoError(t, cfg.Validate())
}

func TestConfig_Validate_InvalidBrokerType(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.BrokerType = BrokerType("invalid")
	assert.Error(t, cfg.Validate())
}

func TestKafkaConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     KafkaConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			cfg:     DefaultKafkaConfig(),
			wantErr: false,
		},
		{
			name: "empty brokers",
			cfg: KafkaConfig{
				Brokers:  []string{},
				ClientID: "test",
				Topic:    "test",
			},
			wantErr: true,
		},
		{
			name: "empty broker address",
			cfg: KafkaConfig{
				Brokers:  []string{""},
				ClientID: "test",
				Topic:    "test",
			},
			wantErr: true,
		},
		{
			name: "empty topic",
			cfg: KafkaConfig{
				Brokers:  []string{"localhost:9092"},
				ClientID: "test",
				Topic:    "",
			},
			wantErr: true,
		},
		{
			name: "empty client ID",
			cfg: KafkaConfig{
				Brokers:  []string{"localhost:9092"},
				ClientID: "",
				Topic:    "test",
			},
			wantErr: true,
		},
		{
			name: "SASL without username",
			cfg: KafkaConfig{
				Brokers:      []string{"localhost:9092"},
				ClientID:     "test",
				Topic:        "test",
				SASLEnabled:  true,
				SASLUsername: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRabbitMQConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     RabbitMQConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			cfg:     DefaultRabbitMQConfig(),
			wantErr: false,
		},
		{
			name: "empty host",
			cfg: RabbitMQConfig{
				Host:     "",
				Port:     5672,
				Exchange: "test",
			},
			wantErr: true,
		},
		{
			name: "invalid port zero",
			cfg: RabbitMQConfig{
				Host:     "localhost",
				Port:     0,
				Exchange: "test",
			},
			wantErr: true,
		},
		{
			name: "invalid port too high",
			cfg: RabbitMQConfig{
				Host:     "localhost",
				Port:     70000,
				Exchange: "test",
			},
			wantErr: true,
		},
		{
			name: "empty exchange",
			cfg: RabbitMQConfig{
				Host:     "localhost",
				Port:     5672,
				Exchange: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDefaultTopics(t *testing.T) {
	topics := DefaultTopics()

	assert.Len(t, topics, 5)

	topicNames := make([]string, len(topics))
	for i, tc := range topics {
		topicNames[i] = tc.Name
	}

	assert.Contains(t, topicNames, "llmsverifier.events.verification")
	assert.Contains(t, topicNames, "llmsverifier.events.provider")
	assert.Contains(t, topicNames, "llmsverifier.events.model")
	assert.Contains(t, topicNames, "llmsverifier.events.team")
	assert.Contains(t, topicNames, "llmsverifier.events.system")
}

// Publisher Tests

func TestNewPublisher(t *testing.T) {
	cfg := DefaultConfig()
	pub := NewPublisher(cfg)

	assert.NotNil(t, pub)
	assert.Equal(t, cfg, pub.config)
	assert.NotNil(t, pub.buffer)
	assert.NotNil(t, pub.stopCh)
	assert.NotNil(t, pub.metrics)
	assert.NotNil(t, pub.logger)
}

func TestNewPublisher_NilConfig(t *testing.T) {
	pub := NewPublisher(nil)

	assert.NotNil(t, pub)
	assert.NotNil(t, pub.config)
	assert.Equal(t, DefaultConfig().Enabled, pub.config.Enabled)
}

func TestPublisher_Start_Disabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false
	pub := NewPublisher(cfg)

	err := pub.Start(context.Background())
	assert.NoError(t, err)
	assert.False(t, pub.IsConnected())
}

func TestPublisher_Start_NoBroker(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	pub := NewPublisher(cfg)

	err := pub.Start(context.Background())
	assert.Error(t, err)
}

func TestPublisher_Start_ConnectError(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	pub := NewPublisher(cfg)

	broker := &mockBroker{connectErr: errors.New("connection failed")}
	pub.SetBroker(broker)

	err := pub.Start(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "connection failed")
}

func TestPublisher_Start_Success(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.AsyncPublish = false
	pub := NewPublisher(cfg)

	broker := &mockBroker{}
	pub.SetBroker(broker)

	err := pub.Start(context.Background())
	assert.NoError(t, err)
	assert.True(t, pub.IsConnected())

	// Cleanup
	pub.Stop(context.Background())
}

func TestPublisher_Stop(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.AsyncPublish = false
	pub := NewPublisher(cfg)

	broker := &mockBroker{}
	pub.SetBroker(broker)

	pub.Start(context.Background())
	assert.True(t, pub.IsConnected())

	err := pub.Stop(context.Background())
	assert.NoError(t, err)
	assert.False(t, pub.IsConnected())
}

func TestPublisher_Publish_Disabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = false
	pub := NewPublisher(cfg)

	event := NewVerificationEvent(EventVerificationStarted, SeverityInfo, "Test", "Test")
	err := pub.Publish(context.Background(), event)
	assert.NoError(t, err)
}

func TestPublisher_Publish_NotConnected(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	pub := NewPublisher(cfg)

	event := NewVerificationEvent(EventVerificationStarted, SeverityInfo, "Test", "Test")
	err := pub.Publish(context.Background(), event)
	assert.Error(t, err)
}

func TestPublisher_Publish_Success(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.AsyncPublish = false
	cfg.BrokerType = BrokerTypeKafka
	pub := NewPublisher(cfg)

	broker := &mockBroker{}
	pub.SetBroker(broker)
	pub.Start(context.Background())
	defer pub.Stop(context.Background())

	event := NewVerificationEvent(EventVerificationStarted, SeverityInfo, "Test", "Test")
	err := pub.Publish(context.Background(), event)
	assert.NoError(t, err)

	calls := broker.GetPublishCalls()
	assert.Len(t, calls, 1)
	assert.Equal(t, "llmsverifier.events.verification", calls[0].topic)
}

func TestPublisher_PublishVerificationStarted(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.AsyncPublish = false
	cfg.BrokerType = BrokerTypeKafka
	pub := NewPublisher(cfg)

	broker := &mockBroker{}
	pub.SetBroker(broker)
	pub.Start(context.Background())
	defer pub.Stop(context.Background())

	err := pub.PublishVerificationStarted(context.Background(), 10, 3)
	assert.NoError(t, err)

	calls := broker.GetPublishCalls()
	assert.Len(t, calls, 1)
}

func TestPublisher_PublishVerificationCompleted(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.AsyncPublish = false
	cfg.BrokerType = BrokerTypeKafka
	pub := NewPublisher(cfg)

	broker := &mockBroker{}
	pub.SetBroker(broker)
	pub.Start(context.Background())
	defer pub.Stop(context.Background())

	err := pub.PublishVerificationCompleted(context.Background(), 5*time.Second, 8, 2)
	assert.NoError(t, err)

	calls := broker.GetPublishCalls()
	assert.Len(t, calls, 1)
}

func TestPublisher_PublishVerificationFailed(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.AsyncPublish = false
	cfg.BrokerType = BrokerTypeKafka
	pub := NewPublisher(cfg)

	broker := &mockBroker{}
	pub.SetBroker(broker)
	pub.Start(context.Background())
	defer pub.Stop(context.Background())

	err := pub.PublishVerificationFailed(context.Background(), "connection timeout")
	assert.NoError(t, err)

	calls := broker.GetPublishCalls()
	assert.Len(t, calls, 1)
}

func TestPublisher_GetMetrics(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.AsyncPublish = false
	cfg.BrokerType = BrokerTypeKafka
	pub := NewPublisher(cfg)

	broker := &mockBroker{}
	pub.SetBroker(broker)
	pub.Start(context.Background())
	defer pub.Stop(context.Background())

	// Publish some events
	for i := 0; i < 3; i++ {
		event := NewVerificationEvent(EventVerificationStarted, SeverityInfo, "Test", "Test")
		pub.Publish(context.Background(), event)
	}

	metrics := pub.GetMetrics()
	assert.NotNil(t, metrics)
	assert.Equal(t, int64(3), metrics.EventsPublished)
	assert.Equal(t, int64(0), metrics.EventsFailed)
}

func TestPublisher_GetKafkaTopicForEvent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BrokerType = BrokerTypeKafka
	pub := NewPublisher(cfg)

	tests := []struct {
		eventType EventType
		expected  string
	}{
		{EventVerificationStarted, "llmsverifier.events.verification"},
		{EventVerificationCompleted, "llmsverifier.events.verification"},
		{EventVerificationFailed, "llmsverifier.events.verification"},
		{EventProviderDiscovered, "llmsverifier.events.provider"},
		{EventProviderScored, "llmsverifier.events.provider"},
		{EventProviderHealthCheck, "llmsverifier.events.provider"},
		{EventProviderFailed, "llmsverifier.events.provider"},
		{EventModelVerified, "llmsverifier.events.model"},
		{EventModelRanked, "llmsverifier.events.model"},
		{EventModelFailed, "llmsverifier.events.model"},
		{EventTeamSelected, "llmsverifier.events.team"},
		{EventTeamMemberFailed, "llmsverifier.events.team"},
		{EventSystemHealthChanged, "llmsverifier.events.system"},
		{EventSecurityAlert, "llmsverifier.events.system"},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			event := NewVerificationEvent(tt.eventType, SeverityInfo, "Test", "Test")
			topic := pub.getKafkaTopicForEvent(event)
			assert.Equal(t, tt.expected, topic)
		})
	}
}

// Event Type Constants Tests

func TestEventTypes(t *testing.T) {
	assert.Equal(t, EventType("verification.started"), EventVerificationStarted)
	assert.Equal(t, EventType("verification.completed"), EventVerificationCompleted)
	assert.Equal(t, EventType("verification.failed"), EventVerificationFailed)
	assert.Equal(t, EventType("provider.discovered"), EventProviderDiscovered)
	assert.Equal(t, EventType("provider.scored"), EventProviderScored)
	assert.Equal(t, EventType("provider.health_check"), EventProviderHealthCheck)
	assert.Equal(t, EventType("provider.failed"), EventProviderFailed)
	assert.Equal(t, EventType("model.verified"), EventModelVerified)
	assert.Equal(t, EventType("model.ranked"), EventModelRanked)
	assert.Equal(t, EventType("model.failed"), EventModelFailed)
	assert.Equal(t, EventType("team.selected"), EventTeamSelected)
	assert.Equal(t, EventType("team.member_failed"), EventTeamMemberFailed)
}

func TestSeverityLevels(t *testing.T) {
	assert.Equal(t, Severity("info"), SeverityInfo)
	assert.Equal(t, Severity("warning"), SeverityWarning)
	assert.Equal(t, Severity("error"), SeverityError)
	assert.Equal(t, Severity("critical"), SeverityCritical)
}

func TestBrokerTypes(t *testing.T) {
	assert.Equal(t, BrokerType("kafka"), BrokerTypeKafka)
	assert.Equal(t, BrokerType("rabbitmq"), BrokerTypeRabbitMQ)
	assert.Equal(t, BrokerType("none"), BrokerTypeNone)
}
