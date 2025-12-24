# Event System Comprehensive Challenge

## Overview
This challenge validates event system including WebSocket, gRPC, notifications (Slack, Email, Telegram, Matrix, WhatsApp), and event subscription management.

## Challenge Type
Integration Test + Real-time Test + Notification Test

## Test Scenarios

### 1. WebSocket Event Subscription Challenge
**Objective**: Verify WebSocket event streaming

**Steps**:
1. Connect to WebSocket endpoint
2. Subscribe to event types (score_change, model_detected)
3. Trigger events
4. Verify events received
5. Test reconnection

**Expected Results**:
- WebSocket connection established
- Subscriptions registered
- Events received in real-time
- Reconnection works after disconnect

**Test Code**:
```go
func TestWebSocketEventSubscription(t *testing.T) {
    client := NewWebSocketClient("ws://localhost:8080/api/v1/events/ws")

    err := client.Connect()
    assert.NoError(t, err)

    events := make(chan Event, 10)
    client.Subscribe([]string{"score_change"}, events)

    // Trigger event
    triggerScoreChangeEvent("gpt-4", 95)

    receivedEvent := <-events
    assert.Equal(t, "score_change", receivedEvent.Type)
    assert.Equal(t, "gpt-4", receivedEvent.Data["model_id"])
}
```

---

### 2. gRPC Event Streaming Challenge
**Objective**: Verify gRPC event streaming

**Steps**:
1. Connect to gRPC server
2. Subscribe to event stream
3. Receive events
4. Handle connection errors

**Expected Results**:
- gRPC connection established
- Event stream receives data
- Errors handled correctly

**Test Code**:
```go
func TestGRPCEventStreaming(t *testing.T) {
    conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
    assert.NoError(t, err)

    client := pb.NewEventServiceClient(conn)
    stream, err := client.SubscribeEvents(context.Background(), &pb.SubscribeRequest{
        EventTypes: []string{"score_change", "model_detected"},
    })
    assert.NoError(t, err)

    event, err := stream.Recv()
    assert.NoError(t, err)
    assert.Equal(t, "score_change", event.Type)
}
```

---

### 3. Slack Notification Challenge
**Objective**: Verify Slack notifications work

**Steps**:
1. Configure Slack webhook/bot token
2. Subscribe to Slack channel
3. Trigger event
4. Verify notification received

**Expected Results**:
- Slack configured correctly
- Notification sent
- Message appears in Slack

**Test Code**:
```go
func TestSlackNotification(t *testing.T) {
    notifier := NewSlackNotifier("xoxb-test-token")

    err := notifier.SendNotification(SlackNotification{
        Channel: "#llm-verifier",
        Message: "Model gpt-4 score changed to 95%",
        Color:   "good",
    })
    assert.NoError(t, err)
}
```

---

### 4. Email Notification Challenge
**Objective**: Verify email notifications work

**Steps**:
1. Configure SMTP settings
2. Subscribe to email notifications
3. Trigger event
4. Verify email received

**Expected Results**:
- Email configured
- Email sent
- Email received

**Test Code**:
```go
func TestEmailNotification(t *testing.T) {
    notifier := NewEmailNotifier(EmailConfig{
        SMTPHost:     "smtp.example.com",
        SMTPPort:     587,
        Username:     "test@example.com",
        Password:     "password",
        FromAddress:  "llm-verifier@example.com",
    })

    err := notifier.SendNotification(EmailNotification{
        To:      "user@example.com",
        Subject: "LLM Verifier Alert",
        Body:    "Model gpt-4 score changed to 95%",
    })
    assert.NoError(t, err)
}
```

---

### 5. Telegram Notification Challenge
**Objective**: Verify Telegram notifications work

**Steps**:
1. Configure Telegram bot token and chat ID
2. Subscribe to Telegram
3. Trigger event
4. Verify message received

**Expected Results**:
- Telegram configured
- Message sent
- Message appears in chat

**Test Code**:
```go
func TestTelegramNotification(t *testing.T) {
    notifier := NewTelegramNotifier("123456789:ABCdefGHIjklMNOpqrsTUVwxyz")

    err := notifier.SendNotification(TelegramNotification{
        ChatID: 123456789,
        Message: "Model gpt-4 score changed to 95%",
    })
    assert.NoError(t, err)
}
```

---

### 6. Matrix Notification Challenge
**Objective**: Verify Matrix notifications work

**Steps**:
1. Configure Matrix server and credentials
2. Join room
3. Trigger event
4. Verify message in room

**Expected Results**:
- Matrix configured
- Message sent
- Message appears in room

**Test Code**:
```go
func TestMatrixNotification(t *testing.T) {
    notifier := NewMatrixNotifier(MatrixConfig{
        Server:   "matrix.org",
        Username: "@bot:matrix.org",
        Password: "password",
        RoomID:   "!abcdef:matrix.org",
    })

    err := notifier.SendNotification(MatrixNotification{
        Message: "Model gpt-4 score changed to 95%",
    })
    assert.NoError(t, err)
}
```

---

### 7. WhatsApp Notification Challenge
**Objective**: Verify WhatsApp notifications work

**Steps**:
1. Configure WhatsApp Business API
2. Subscribe to WhatsApp
3. Trigger event
4. Verify message received

**Expected Results**:
- WhatsApp configured
- Message sent
- Message appears

**Test Code**:
```go
func TestWhatsAppNotification(t *testing.T) {
    notifier := NewWhatsAppNotifier(WhatsAppConfig{
        PhoneID:      "123456789",
        AccessToken:  "test-access-token",
    })

    err := notifier.SendNotification(WhatsAppNotification{
        To:      "+1234567890",
        Message: "Model gpt-4 score changed to 95%",
    })
    assert.NoError(t, err)
}
```

---

### 8. Multi-Channel Notification Challenge
**Objective**: Verify notifications sent to multiple channels

**Steps**:
1. Configure multiple channels (Slack, Email, Telegram)
2. Subscribe to all channels
3. Trigger event
4. Verify all channels received

**Expected Results**:
- All channels configured
- All channels receive notification
- Notifications consistent

**Test Code**:
```go
func TestMultiChannelNotification(t *testing.T) {
    channels := []NotificationChannel{
        NewSlackNotifier("xoxb-test-token"),
        NewEmailNotifier(emailConfig),
        NewTelegramNotifier("123456789:ABC"),
    }

    event := Event{
        Type: "score_change",
        Data: map[string]interface{}{
            "model_id": "gpt-4",
            "score":    95,
        },
    }

    for _, channel := range channels {
        err := channel.Send(event)
        assert.NoError(t, err)
    }
}
```

---

### 9. Event Registration Challenge
**Objective**: Verify event registration works

**Steps**:
1. Register event subscriber via API
2. Verify registration in database
3. List subscribers
4. Unregister subscriber

**Expected Results**:
- Subscriber registered
- Record in database
- List shows subscriber
- Unregistration works

**Test Code**:
```go
func TestEventRegistration(t *testing.T) {
    registry := NewEventRegistry()

    subscriber := EventSubscriber{
        ID:         "sub-001",
        WebhookURL: "http://example.com/webhook",
        Channels:   []string{"score_change"},
    }

    err := registry.Register(subscriber)
    assert.NoError(t, err)

    retrieved, err := registry.Get("sub-001")
    assert.NoError(t, err)
    assert.Equal(t, "sub-001", retrieved.ID)
}
```

---

### 10. Event Filtering Challenge
**Objective**: Verify events can be filtered

**Steps**:
1. Subscribe with filters (provider, model, score threshold)
2. Trigger multiple events
3. Verify only matching events received

**Expected Results**:
- Filters work correctly
- Only matching events received
- Non-matching events ignored

**Test Code**:
```go
func TestEventFiltering(t *testing.T) {
    client := NewWebSocketClient("ws://localhost:8080/api/v1/events/ws")

    filter := EventFilter{
        Provider:     "openai",
        MinScore:     90,
        EventTypes:   []string{"score_change"},
    }

    events := make(chan Event, 10)
    client.SubscribeWithFilter(filter, events)

    // Trigger multiple events
    triggerScoreChangeEvent("gpt-4", 95) // Should receive
    triggerScoreChangeEvent("claude-3-opus", 92) // Should NOT receive (wrong provider)

    received := <-events
    assert.Equal(t, "gpt-4", received.Data["model_id"])
}
```

---

## Success Criteria

### Functional Requirements
- [ ] WebSocket streaming works
- [ ] gRPC streaming works
- [ ] Slack notifications work
- [ ] Email notifications work
- [ ] Telegram notifications work
- [ ] Matrix notifications work
- [ ] WhatsApp notifications work
- [ ] Multi-channel notifications work
- [ ] Event registration works
- [ ] Event filtering works

### Reliability Requirements
- [ ] Reconnections work
- [ ] Failed notifications retried
- [ ] Events not lost
- [ ] Order preserved

### Security Requirements
- [ ] API keys protected
- [ ] Webhooks validated
- [ ] Sensitive data not in notifications

## Dependencies
- Notification service configured
- Valid API keys for all channels

## Cleanup
- Remove test subscribers
- Clear test notifications
