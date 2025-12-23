# LLM Verifier - Step-by-Step Implementation Execution Guide

**Generated**: $(date -u +"%Y-%m-%d %H:%M:%S UTC")
**Project Path**: `/media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier`

---

## EXECUTION PREREQUISITES

Before starting implementation, ensure you have:

1. **Go 1.21+** installed
2. **Node.js 18+** and npm/yarn for web/desktop/mobile apps
3. **Flutter SDK** for Flutter mobile app
4. **Rust toolchain** for Tauri desktop app
5. **Android Studio** and **Xcode** for mobile app development
6. **Docker** for containerized testing
7. **Git** for version control
8. **Screen recording software** for video course production
9. **Video editing software** (e.g., DaVinci Resolve, Premiere Pro)
10. **Static site generator** (Hugo) for website

### Tool Installation Commands

```bash
# Go
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Node.js
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt install -y nodejs

# Flutter
git clone https://github.com/flutter/flutter.git -b stable
export PATH="$PATH:`pwd`/flutter/bin"

# Rust
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh

# Hugo
wget https://github.com/gohugoio/hugo/releases/download/v0.121.2/hugo_extended_0.121.2_linux-amd64.deb
sudo dpkg -i hugo_extended_0.121.2_linux-amd64.deb
```

---

## PHASE 1: CORE BACKEND TESTING & DOCUMENTATION

### Week 1: Critical Package Testing (Events & Failover)

#### Day 1: Events Package Testing

**File**: `llm-verifier/events/events_test.go`

```bash
cd /media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier/llm-verifier
```

Create test file:

```go
package events

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEventBus(t *testing.T) {
	eb := NewEventBus()
	assert.NotNil(t, eb)
	assert.NotNil(t, eb.subscribers)
	assert.NotNil(t, eb.ctx)
	assert.NotNil(t, eb.cancel)
}

func TestSubscribe(t *testing.T) {
	eb := NewEventBus()
	sub1, err := eb.Subscribe("test-event", &TestSubscriber{
		id: "sub1",
		types: []string{"test-event"},
	})
	require.NoError(t, err)
	assert.NotNil(t, sub1)

	// Test duplicate subscription
	sub2, err := eb.Subscribe("test-event", sub1)
	assert.Error(t, err)
	assert.Nil(t, sub2)
}

func TestPublish(t *testing.T) {
	eb := NewEventBus()
	
	received := false
	sub, err := eb.Subscribe("test-event", &TestSubscriber{
		id: "sub1",
		types: []string{"test-event"},
		onEvent: func(e Event) {
			received = true
			assert.Equal(t, "test-event", e.Type)
		},
	})
	require.NoError(t, err)
	require.NotNil(t, sub)

	event := Event{
		ID:        "evt-1",
		Type:      "test-event",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"test": "data"},
	}

	err = eb.Publish(event)
	assert.NoError(t, err)
	
	// Wait for async delivery
	time.Sleep(100 * time.Millisecond)
	assert.True(t, received, "Event should be delivered to subscriber")
}

func TestUnsubscribe(t *testing.T) {
	eb := NewEventBus()
	
	sub, err := eb.Subscribe("test-event", &TestSubscriber{
		id: "sub1",
		types: []string{"test-event"},
	})
	require.NoError(t, err)

	err = eb.Unsubscribe(sub.ID)
	assert.NoError(t, err)

	// Try to publish after unsubscribe
	event := Event{
		ID:        "evt-1",
		Type:      "test-event",
		Timestamp: time.Now(),
	}
	
	err = eb.Publish(event)
	assert.NoError(t, err)
}

func TestShutdown(t *testing.T) {
	eb := NewEventBus()
	
	sub, err := eb.Subscribe("test-event", &TestSubscriber{
		id: "sub1",
		types: []string{"test-event"},
	})
	require.NoError(t, err)

	eb.Shutdown()

	// Verify subscribers are cleaned up
	eb.mu.RLock()
	_, exists := eb.subscribers[sub.ID]
	eb.mu.RUnlock()
	assert.False(t, exists, "Subscriber should be removed after shutdown")
}

// Test Helpers
type TestSubscriber struct {
	id       string
	types    []string
	onEvent  func(Event)
}

func (ts *TestSubscriber) GetTypes() []string {
	return ts.types
}

func (ts *TestSubscriber) OnEvent(e Event) {
	if ts.onEvent != nil {
		ts.onEvent(e)
	}
}

func TestEventStore(t *testing.T) {
	eb := NewEventBus()
	
	event := Event{
		ID:        "evt-1",
		Type:      "test-event",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{"test": "data"},
	}

	err := eb.Publish(event)
	assert.NoError(t, err)

	// Verify event is stored (if store is implemented)
	events := eb.GetEvents("test-event", 10)
	assert.GreaterOrEqual(t, len(events), 1)
}

func TestConcurrentPublish(t *testing.T) {
	eb := NewEventBus()
	
	count := 100
	receivedCount := 0
	
	sub, err := eb.Subscribe("test-event", &ConcurrentTestSubscriber{
		id:     "sub1",
		types:  []string{"test-event"},
		count:  &receivedCount,
	})
	require.NoError(t, err)

	// Publish multiple events concurrently
	for i := 0; i < count; i++ {
		go func(index int) {
			event := Event{
				ID:        fmt.Sprintf("evt-%d", index),
				Type:      "test-event",
				Timestamp: time.Now(),
				Data:      map[string]interface{}{"index": index},
			}
			eb.Publish(event)
		}(i)
	}

	// Wait for all events
	time.Sleep(500 * time.Millisecond)
	assert.Equal(t, count, receivedCount, "All events should be delivered")
}

type ConcurrentTestSubscriber struct {
	id     string
	types  []string
	count  *int
	mu     sync.Mutex
}

func (cts *ConcurrentTestSubscriber) GetTypes() []string {
	return cts.types
}

func (cts *ConcurrentTestSubscriber) OnEvent(e Event) {
	cts.mu.Lock()
	*cts.count++
	cts.mu.Unlock()
}

func TestEventFiltering(t *testing.T) {
	eb := NewEventBus()
	
	typeAReceived := false
	typeBReceived := false
	
	subA, _ := eb.Subscribe("type-a", &TypeTestSubscriber{
		id: "sub-a",
		received: &typeAReceived,
		expectedType: "type-a",
	})
	
	subB, _ := eb.Subscribe("type-b", &TypeTestSubscriber{
		id: "sub-b",
		received: &typeBReceived,
		expectedType: "type-b",
	})

	// Publish type A event
	eventA := Event{
		ID:        "evt-a",
		Type:      "type-a",
		Timestamp: time.Now(),
	}
	eb.Publish(eventA)

	// Publish type B event
	eventB := Event{
		ID:        "evt-b",
		Type:      "type-b",
		Timestamp: time.Now(),
	}
	eb.Publish(eventB)

	time.Sleep(100 * time.Millisecond)
	
	assert.True(t, typeAReceived, "Type A subscriber should receive type A")
	assert.True(t, typeBReceived, "Type B subscriber should receive type B")
}

type TypeTestSubscriber struct {
	id           string
	expectedType string
	received     *bool
}

func (tts *TypeTestSubscriber) GetTypes() []string {
	return []string{tts.expectedType}
}

func (tts *TypeTestSubscriber) OnEvent(e Event) {
	if e.Type == tts.expectedType {
		*tts.received = true
	}
}
```

**Execute Tests**:
```bash
go test ./events -v -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

**Coverage Target**: 100%

---

#### Day 2: Failover - Circuit Breaker Testing

**File**: `llm-verifier/failover/circuit_breaker_test.go`

```go
package failover

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker("test-cb")
	assert.NotNil(t, cb)
	assert.Equal(t, "test-cb", cb.name)
	assert.Equal(t, StateClosed, cb.state)
	assert.Equal(t, 0, cb.failureCount)
}

func TestCircuitBreakerClosedState(t *testing.T) {
	cb := NewCircuitBreaker("test-cb")
	
	callCount := 0
	fn := func() error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	// First call - failure
	err := cb.Call(fn)
	assert.Error(t, err)
	assert.Equal(t, 1, cb.failureCount)
	assert.Equal(t, StateClosed, cb.state)

	// Second call - failure
	err = cb.Call(fn)
	assert.Error(t, err)
	assert.Equal(t, 2, cb.failureCount)
	assert.Equal(t, StateClosed, cb.state)

	// Third call - failure (threshold reached, should open)
	err = cb.Call(fn)
	assert.Error(t, err)
	assert.Equal(t, 3, cb.failureCount)
	assert.Equal(t, StateOpen, cb.state)
}

func TestCircuitBreakerOpenState(t *testing.T) {
	cb := NewCircuitBreaker("test-cb")
	
	// Force open state
	for i := 0; i < 5; i++ {
		cb.Call(func() error {
			return errors.New("error")
		})
	}

	assert.Equal(t, StateOpen, cb.state)

	// Try to call when open - should fail with ErrCircuitOpen
	err := cb.Call(func() error {
		return nil
	})
	assert.Error(t, err)
	assert.Equal(t, ErrCircuitOpen, err)
}

func TestCircuitBreakerRecovery(t *testing.T) {
	cb := NewCircuitBreaker("test-cb")
	cb.recoveryTimeout = 100 * time.Millisecond

	// Force open state
	for i := 0; i < 5; i++ {
		cb.Call(func() error {
			return errors.New("error")
		})
	}

	assert.Equal(t, StateOpen, cb.state)

	// Wait for recovery timeout
	time.Sleep(150 * time.Millisecond)

	// Should be in half-open state
	cb.mu.RLock()
	state := cb.state
	cb.mu.RUnlock()
	assert.Equal(t, StateHalfOpen, state)

	// Successful call should close circuit
	err := cb.Call(func() error {
		return nil
	})
	assert.NoError(t, err)

	assert.Equal(t, StateClosed, cb.state)
}

func TestCircuitBreakerSuccessThreshold(t *testing.T) {
	cb := NewCircuitBreaker("test-cb")
	cb.successThreshold = 2
	cb.recoveryTimeout = 100 * time.Millisecond

	// Force open state
	for i := 0; i < 5; i++ {
		cb.Call(func() error {
			return errors.New("error")
		})
	}

	assert.Equal(t, StateOpen, cb.state)

	// Wait for recovery
	time.Sleep(150 * time.Millisecond)

	// Two successful calls should close circuit
	cb.Call(func() error {
		return nil
	})
	
	assert.Equal(t, StateHalfOpen, cb.state)
	
	cb.Call(func() error {
		return nil
	})

	assert.Equal(t, StateClosed, cb.state)
}

func TestCircuitBreakerConcurrentCalls(t *testing.T) {
	cb := NewCircuitBreaker("test-cb")
	
	callCount := make(map[int]int)
	var mu sync.Mutex

	// Concurrent calls
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			cb.Call(func() error {
				mu.Lock()
				callCount[index]++
				mu.Unlock()
				return nil
			})
		}(i)
	}

	wg.Wait()
	assert.Equal(t, 100, len(callCount))
	assert.Equal(t, StateClosed, cb.state)
}

func TestCircuitBreakerStateTransitions(t *testing.T) {
	cb := NewCircuitBreaker("test-cb")
	cb.failureThreshold = 2
	cb.recoveryTimeout = 100 * time.Millisecond
	cb.successThreshold = 1

	// Start: Closed
	assert.Equal(t, StateClosed, cb.state)

	// Failure 1: Still Closed
	cb.Call(func() error { return errors.New("error") })
	assert.Equal(t, StateClosed, cb.state)

	// Failure 2: Should Open
	cb.Call(func() error { return errors.New("error") })
	assert.Equal(t, StateOpen, cb.state)

	// Call while Open: Should fail
	err := cb.Call(func() error { return nil })
	assert.Error(t, err)
	assert.Equal(t, ErrCircuitOpen, err)

	// Wait for recovery
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, StateHalfOpen, cb.state)

	// Success: Should Close
	err = cb.Call(func() error { return nil })
	assert.NoError(t, err)
	assert.Equal(t, StateClosed, cb.state)
}
```

---

#### Day 3: Failover - Manager Testing

**File**: `llm-verifier/failover/failover_manager_test.go`

```go
package failover

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFailoverManager(t *testing.T) {
	fm := NewFailoverManager()
	assert.NotNil(t, fm)
	assert.NotNil(t, fm.providers)
	assert.NotNil(t, fm.circuitBreakers)
	assert.NotNil(t, fm.healthChecker)
}

func TestFailoverManagerAddProvider(t *testing.T) {
	fm := NewFailoverManager()
	
	provider := ProviderConfig{
		ID:       "provider-1",
		Endpoint: "https://api.example.com",
		Priority: 1,
	}

	err := fm.AddProvider(provider)
	assert.NoError(t, err)

	providers, err := fm.GetProviders()
	assert.NoError(t, err)
	assert.Len(t, providers, 1)
	assert.Equal(t, "provider-1", providers[0].ID)
}

func TestFailoverManagerRemoveProvider(t *testing.T) {
	fm := NewFailoverManager()
	
	provider := ProviderConfig{
		ID:       "provider-1",
		Endpoint: "https://api.example.com",
		Priority: 1,
	}

	fm.AddProvider(provider)
	
	err := fm.RemoveProvider("provider-1")
	assert.NoError(t, err)

	providers, err := fm.GetProviders()
	assert.NoError(t, err)
	assert.Len(t, providers, 0)
}

func TestFailoverManagerPrimarySelection(t *testing.T) {
	fm := NewFailoverManager()
	
	providers := []ProviderConfig{
		{ID: "provider-1", Endpoint: "https://api1.com", Priority: 1},
		{ID: "provider-2", Endpoint: "https://api2.com", Priority: 2},
		{ID: "provider-3", Endpoint: "https://api3.com", Priority: 3},
	}

	for _, p := range providers {
		fm.AddProvider(p)
	}

	primary, err := fm.GetPrimaryProvider()
	assert.NoError(t, err)
	assert.Equal(t, "provider-1", primary.ID)
}

func TestFailoverManagerFailover(t *testing.T) {
	fm := NewFailoverManager()
	
	providers := []ProviderConfig{
		{ID: "provider-1", Endpoint: "https://api1.com", Priority: 1},
		{ID: "provider-2", Endpoint: "https://api2.com", Priority: 2},
		{ID: "provider-3", Endpoint: "https://api3.com", Priority: 3},
	}

	for _, p := range providers {
		fm.AddProvider(p)
	}

	// Mark provider-1 as failed
	fm.MarkProviderFailed("provider-1")

	// Should failover to provider-2
	primary, err := fm.GetPrimaryProvider()
	assert.NoError(t, err)
	assert.Equal(t, "provider-2", primary.ID)
}

func TestFailoverManagerAllProvidersFailed(t *testing.T) {
	fm := NewFailoverManager()
	
	providers := []ProviderConfig{
		{ID: "provider-1", Endpoint: "https://api1.com", Priority: 1},
		{ID: "provider-2", Endpoint: "https://api2.com", Priority: 2},
	}

	for _, p := range providers {
		fm.AddProvider(p)
	}

	// Mark all as failed
	fm.MarkProviderFailed("provider-1")
	fm.MarkProviderFailed("provider-2")

	// Should return error
	_, err := fm.GetPrimaryProvider()
	assert.Error(t, err)
	assert.Equal(t, ErrNoHealthyProviders, err)
}

func TestFailoverManagerRecovery(t *testing.T) {
	fm := NewFailoverManager()
	
	providers := []ProviderConfig{
		{ID: "provider-1", Endpoint: "https://api1.com", Priority: 1},
		{ID: "provider-2", Endpoint: "https://api2.com", Priority: 2},
	}

	for _, p := range providers {
		fm.AddProvider(p)
	}

	// Mark provider-1 as failed
	fm.MarkProviderFailed("provider-1")

	// Should failover to provider-2
	primary, _ := fm.GetPrimaryProvider()
	assert.Equal(t, "provider-2", primary.ID)

	// Recover provider-1
	fm.MarkProviderRecovered("provider-1")

	// Should switch back to provider-1 (higher priority)
	primary, err := fm.GetPrimaryProvider()
	assert.NoError(t, err)
	assert.Equal(t, "provider-1", primary.ID)
}

func TestFailoverManagerLatencyRouting(t *testing.T) {
	fm := NewFailoverManager()
	
	providers := []ProviderConfig{
		{ID: "provider-1", Endpoint: "https://api1.com", Priority: 1, Latency: 100},
		{ID: "provider-2", Endpoint: "https://api2.com", Priority: 2, Latency: 50},
		{ID: "provider-3", Endpoint: "https://api3.com", Priority: 3, Latency: 75},
	}

	for _, p := range providers {
		fm.AddProvider(p)
	}

	// Update latencies
	fm.UpdateProviderLatency("provider-1", 100)
	fm.UpdateProviderLatency("provider-2", 50)
	fm.UpdateProviderLatency("provider-3", 75)

	// Should route to lowest latency among healthy providers
	best, err := fm.GetBestProvider()
	assert.NoError(t, err)
	assert.Equal(t, "provider-2", best.ID)
}
```

---

#### Day 4: Failover - Health Checker Testing

**File**: `llm-verifier/failover/health_checker_test.go`

```go
package failover

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHealthChecker(t *testing.T) {
	hc := NewHealthChecker(10 * time.Second)
	assert.NotNil(t, hc)
	assert.NotNil(t, hc.checkInterval)
	assert.NotNil(t, hc.stopCh)
	assert.NotNil(t, hc.wg)
}

func TestHealthCheckerStartStop(t *testing.T) {
	hc := NewHealthChecker(100 * time.Millisecond)
	
	started := make(chan bool)
	go func() {
		hc.Start()
		started <- true
	}()

	// Wait for start
	select {
	case <-started:
		assert.True(t, true)
	case <-time.After(1 * time.Second):
		assert.Fail(t, "Health checker did not start")
	}

	// Wait for health checks to run
	time.Sleep(200 * time.Millisecond)

	// Stop health checker
	hc.Stop()
}

func TestHealthCheckerProviderRegistration(t *testing.T) {
	hc := NewHealthChecker(10 * time.Second)
	
	provider := &ProviderConfig{
		ID:       "provider-1",
		Endpoint: "https://api.example.com",
	}

	err := hc.RegisterProvider(provider.ID, provider.Endpoint)
	assert.NoError(t, err)

	providers := hc.GetProviders()
	assert.Contains(t, providers, "provider-1")
}

func TestHealthCheckerHealthCheck(t *testing.T) {
	// Create test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"healthy"}`))
		}
	}))
	defer server.Close()

	hc := NewHealthChecker(100 * time.Millisecond)
	
	provider := &ProviderConfig{
		ID:       "provider-1",
		Endpoint: server.URL,
	}

	hc.RegisterProvider(provider.ID, provider.Endpoint)
	hc.Start()

	// Wait for health check
	time.Sleep(200 * time.Millisecond)

	// Check if provider is marked as healthy
	healthy := hc.IsHealthy(provider.ID)
	assert.True(t, healthy)

	hc.Stop()
}

func TestHealthCheckerUnhealthyEndpoint(t *testing.T) {
	// Create test HTTP server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	hc := NewHealthChecker(100 * time.Millisecond)
	
	provider := &ProviderConfig{
		ID:       "provider-1",
		Endpoint: server.URL,
	}

	hc.RegisterProvider(provider.ID, provider.Endpoint)
	hc.Start()

	// Wait for health checks
	time.Sleep(200 * time.Millisecond)

	// Check if provider is marked as unhealthy
	healthy := hc.IsHealthy(provider.ID)
	assert.False(t, healthy)

	hc.Stop()
}

func TestHealthCheckerTimeout(t *testing.T) {
	// Create test HTTP server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	hc := NewHealthChecker(50 * time.Millisecond)
	
	provider := &ProviderConfig{
		ID:       "provider-1",
		Endpoint: server.URL,
	}

	hc.RegisterProvider(provider.ID, provider.Endpoint)
	hc.Start()

	// Wait for health checks
	time.Sleep(200 * time.Millisecond)

	// Check if provider is marked as unhealthy (timeout)
	healthy := hc.IsHealthy(provider.ID)
	assert.False(t, healthy)

	hc.Stop()
}

func TestHealthCheckerConcurrentChecks(t *testing.T) {
	// Create multiple test servers
	var servers []*httptest.Server
	var endpoints []string

	for i := 0; i < 5; i++ {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		servers = append(servers, server)
		endpoints = append(endpoints, server.URL)
	}

	defer func() {
		for _, server := range servers {
			server.Close()
		}
	}()

	hc := NewHealthChecker(100 * time.Millisecond)

	// Register multiple providers
	for i := 0; i < 5; i++ {
		provider := &ProviderConfig{
			ID:       fmt.Sprintf("provider-%d", i),
			Endpoint: endpoints[i],
		}
		hc.RegisterProvider(provider.ID, provider.Endpoint)
	}

	hc.Start()

	// Wait for health checks
	time.Sleep(200 * time.Millisecond)

	// Check all providers
	for i := 0; i < 5; i++ {
		healthy := hc.IsHealthy(fmt.Sprintf("provider-%d", i))
		assert.True(t, healthy, fmt.Sprintf("Provider %d should be healthy", i))
	}

	hc.Stop()
}
```

---

#### Day 5: Failover - Latency Router Testing

**File**: `llm-verifier/failover/latency_router_test.go`

```go
package failover

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewLatencyRouter(t *testing.T) {
	lr := NewLatencyRouter()
	assert.NotNil(t, lr)
	assert.NotNil(t, lr.providerLatencies)
}

func TestLatencyRouterRecord(t *testing.T) {
	lr := NewLatencyRouter()
	
	providerID := "provider-1"
	latency := 150 * time.Millisecond

	lr.RecordLatency(providerID, latency)

	recorded := lr.GetAverageLatency(providerID)
	assert.Equal(t, latency, recorded)
}

func TestLatencyRouterAverage(t *testing.T) {
	lr := NewLatencyRouter()
	
	providerID := "provider-1"
	
	// Record multiple latencies
	latencies := []time.Duration{
		100 * time.Millisecond,
		150 * time.Millisecond,
		200 * time.Millisecond,
	}

	for _, lat := range latencies {
		lr.RecordLatency(providerID, lat)
	}

	average := lr.GetAverageLatency(providerID)
	expected := 150 * time.Millisecond
	assert.Equal(t, expected, average)
}

func TestLatencyRouterGetLowestLatencyProvider(t *testing.T) {
	lr := NewLatencyRouter()
	
	providers := map[string]time.Duration{
		"provider-1": 100 * time.Millisecond,
		"provider-2": 200 * time.Millisecond,
		"provider-3": 150 * time.Millisecond,
		"provider-4": 300 * time.Millisecond,
	}

	for providerID, latency := range providers {
		lr.RecordLatency(providerID, latency)
	}

	best := lr.GetLowestLatencyProvider()
	assert.Equal(t, "provider-1", best)
}

func TestLatencyRouterEmpty(t *testing.T) {
	lr := NewLatencyRouter()
	
	best := lr.GetLowestLatencyProvider()
	assert.Equal(t, "", best)
}

func TestLatencyRouterReset(t *testing.T) {
	lr := NewLatencyRouter()
	
	providerID := "provider-1"
	lr.RecordLatency(providerID, 100*time.Millisecond)

	// Reset latency
	lr.ResetLatency(providerID)

	average := lr.GetAverageLatency(providerID)
	assert.Equal(t, time.Duration(0), average)
}
```

---

#### Day 6-7: Failover Documentation

**File**: `llm-verifier/failover/README.md`

```markdown
# Failover Package

## Overview

The failover package provides circuit breaker patterns, health checking, and latency routing for LLM provider failover and high availability.

## Components

### Circuit Breaker

The circuit breaker prevents cascading failures by automatically opening when a provider fails repeatedly.

**States**:
- **Closed**: Normal operation, requests pass through
- **Open**: Circuit is open, requests fail immediately
- **Half-Open**: Recovery mode, limited requests allowed

**Usage**:
```go
import "llm-verifier/failover"

cb := NewCircuitBreaker("openai-provider")
err := cb.Call(func() error {
    // Make API request
    return makeRequest()
})

if errors.Is(err, ErrCircuitOpen) {
    // Handle circuit open
}
```

### Failover Manager

Manages multiple providers and automatic failover based on health and priority.

**Features**:
- Priority-based provider selection
- Automatic failover on failure
- Recovery detection
- Latency-aware routing

**Usage**:
```go
fm := NewFailoverManager()

// Add providers
fm.AddProvider(ProviderConfig{
    ID: "openai",
    Endpoint: "https://api.openai.com/v1",
    Priority: 1,
})

fm.AddProvider(ProviderConfig{
    ID: "anthropic",
    Endpoint: "https://api.anthropic.com/v1",
    Priority: 2,
})

// Get primary provider
provider, err := fm.GetPrimaryProvider()
```

### Health Checker

Monitors provider health at regular intervals and updates circuit breaker state.

**Features**:
- Configurable check interval
- Timeout handling
- Custom health endpoints
- Circuit breaker integration

**Usage**:
```go
hc := NewHealthChecker(30 * time.Second)

hc.RegisterProvider("openai", "https://api.openai.com/v1")
hc.RegisterProvider("anthropic", "https://api.anthropic.com/v1")

hc.Start()
defer hc.Stop()

// Check health
healthy := hc.IsHealthy("openai")
```

### Latency Router

Tracks request latency and routes to fastest healthy provider.

**Features**:
- Latency tracking
- Average calculation
- Lowest latency selection
- Automatic updates

**Usage**:
```go
lr := NewLatencyRouter()

// Record latency after request
start := time.Now()
makeRequest()
latency := time.Since(start)
lr.RecordLatency(providerID, latency)

// Get best provider
best := lr.GetLowestLatencyProvider()
```

## Configuration

### Circuit Breaker Configuration

```go
cb := NewCircuitBreaker("provider-name")
cb.FailureThreshold = 5          // Failures before opening
cb.RecoveryTimeout = 30 * time.Second
cb.SuccessThreshold = 3           // Successes before closing
```

### Health Checker Configuration

```go
hc := NewHealthChecker(30 * time.Second)
hc.HealthEndpoint = "/health"
hc.Timeout = 10 * time.Second
```

## Best Practices

1. **Configure appropriate thresholds**: Adjust based on your API rate limits and reliability
2. **Monitor circuit state**: Log state changes for debugging
3. **Use exponential backoff**: When circuit is open, increase retry delay
4. **Regular health checks**: Balance between responsiveness and API load
5. **Latency routing**: Use in conjunction with priority routing for optimal performance

## Testing

Run tests:
```bash
go test ./failover -v
```

Coverage:
```bash
go test ./failover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Architecture

```
┌─────────────────┐
│ Failover Manager │
└────────┬────────┘
         │
    ┌────┴────┬────────┐
    │           │        │
┌───▼───┐ ┌───▼────┐ ┌▼────────┐
│ Circuit │ │  Health │ │  Latency │
│ Breaker │ │ Checker │ │  Router  │
└─────────┘ └─────────┘ └──────────┘
```
```

---

## EXECUTION SUMMARY

This execution guide provides detailed steps for Phase 1 (Weeks 1-3) of the comprehensive project completion plan.

**What's in this guide**:
- Complete test implementations with 100% coverage targets
- Detailed test code for all zero-coverage packages
- Documentation templates
- Execution commands

**Next steps after this guide**:
1. Continue with remaining packages in Phase 1
2. Move to Phase 2 (Database & Provider Testing)
3. Follow the comprehensive report timeline

**Remember**:
- No interactive commands (no sudo/password prompts)
- Run tests before committing
- Update documentation after each change
- Verify 100% coverage before moving on

---

*Continue with the next phase in the comprehensive implementation plan.*
