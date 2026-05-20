package messaging

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// fakeMessagingTranslator returns "<TRANSLATED:msg_id>" so tests can assert the
// i18n-routed sentinel without coupling to the English bundle text.
// Anti-bluff per CONST-035 / Article XI §11.9: a test asserting the original
// literal would silently pass if the call site bypassed the translator.
type fakeMessagingTranslator struct{}

func (fakeMessagingTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}
func (fakeMessagingTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeMessagingTranslator installs the fakeMessagingTranslator, runs fn,
// then restores the prior translator.
func withFakeMessagingTranslator(t *testing.T, fn func()) {
	t.Helper()
	prior := translator
	translator = fakeMessagingTranslator{}
	defer func() { translator = prior }()
	fn()
}

// publishedEvent decodes the last JSON payload the mockBroker captured back
// into a VerificationEvent — a real round-trip through the genuine publish
// path (NewVerificationEvent → json.Marshal → broker.Publish), not a mock of
// the event itself.
func publishedEvent(t *testing.T, broker *mockBroker) *VerificationEvent {
	t.Helper()
	calls := broker.GetPublishCalls()
	if len(calls) == 0 {
		t.Fatal("no event published to broker")
	}
	var ev VerificationEvent
	if err := json.Unmarshal(calls[len(calls)-1].message, &ev); err != nil {
		t.Fatalf("decode published event: %v", err)
	}
	return &ev
}

// newTestPublisher wires a real Publisher + mockBroker with synchronous
// publishing so the captured payload is available immediately.
func newTestPublisher(t *testing.T) (*Publisher, *mockBroker) {
	t.Helper()
	cfg := DefaultConfig()
	cfg.Enabled = true
	cfg.AsyncPublish = false
	cfg.BrokerType = BrokerTypeKafka
	pub := NewPublisher(cfg)
	broker := &mockBroker{}
	pub.SetBroker(broker)
	if err := pub.Start(context.Background()); err != nil {
		t.Fatalf("publisher start: %v", err)
	}
	t.Cleanup(func() { _ = pub.Stop(context.Background()) })
	return pub, broker
}

// TestPublisher_TitlesRouted proves every verification-event publisher routes
// its Title + Message through the i18n seam. With the fake translator
// installed, every Title/Message must carry the "<TRANSLATED:messaging.*>"
// prefix — if a branch still held an English literal, the assertion fails.
func TestPublisher_TitlesRouted(t *testing.T) {
	cases := []struct {
		name     string
		titlePfx string
		publish  func(p *Publisher) error
	}{
		{"verification_started", "<TRANSLATED:messaging.verification.started.",
			func(p *Publisher) error {
				return p.PublishVerificationStarted(context.Background(), 3, 2)
			}},
		{"verification_completed", "<TRANSLATED:messaging.verification.completed.",
			func(p *Publisher) error {
				return p.PublishVerificationCompleted(context.Background(), time.Second, 5, 1)
			}},
		{"verification_failed", "<TRANSLATED:messaging.verification.failed.",
			func(p *Publisher) error {
				return p.PublishVerificationFailed(context.Background(), "boom")
			}},
		{"provider_scored", "<TRANSLATED:messaging.provider.scored.",
			func(p *Publisher) error {
				return p.PublishProviderScored(context.Background(), &ProviderScoredEvent{
					ProviderID: "p1", ProviderName: "OpenAI", OverallScore: 91.5,
				})
			}},
		{"provider_health", "<TRANSLATED:messaging.provider.health.",
			func(p *Publisher) error {
				return p.PublishProviderHealthCheck(context.Background(), &ProviderHealthEvent{
					ProviderID: "p1", ProviderName: "OpenAI", Status: "healthy",
				})
			}},
		{"model_ranked", "<TRANSLATED:messaging.model.ranked.",
			func(p *Publisher) error {
				return p.PublishModelRanked(context.Background(), &ModelRankedEvent{
					ProviderID: "p1", ModelID: "m1", ModelName: "gpt-4", Rank: 1, Score: 95.0,
				})
			}},
		{"team_selected", "<TRANSLATED:messaging.team.selected.",
			func(p *Publisher) error {
				return p.PublishTeamSelected(context.Background(), &TeamSelectedEvent{
					TeamID: "t1", TeamSize: 3, SelectionCriteria: "code-capability",
				})
			}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			withFakeMessagingTranslator(t, func() {
				pub, broker := newTestPublisher(t)
				if err := c.publish(pub); err != nil {
					t.Fatalf("publish returned error: %v", err)
				}
				ev := publishedEvent(t, broker)
				if !strings.HasPrefix(ev.Title, c.titlePfx) {
					t.Errorf("Title not i18n-routed: got %q want prefix %q", ev.Title, c.titlePfx)
				}
				if !strings.HasPrefix(ev.Message, "<TRANSLATED:messaging.") {
					t.Errorf("Message not i18n-routed: got %q", ev.Message)
				}
			})
		})
	}
}

// TestPublisher_HealthSeverityBranches covers all three health-status severity
// branches and confirms each still routes Title/Message through the seam.
func TestPublisher_HealthSeverityBranches(t *testing.T) {
	for _, status := range []string{"healthy", "degraded", "unhealthy"} {
		s := status
		t.Run(s, func(t *testing.T) {
			withFakeMessagingTranslator(t, func() {
				pub, broker := newTestPublisher(t)
				if err := pub.PublishProviderHealthCheck(context.Background(), &ProviderHealthEvent{
					ProviderID: "p1", ProviderName: "OpenAI", Status: s,
				}); err != nil {
					t.Fatalf("publish health %q: %v", s, err)
				}
				ev := publishedEvent(t, broker)
				if !strings.HasPrefix(ev.Title, "<TRANSLATED:messaging.provider.health.") {
					t.Errorf("health status %q title not routed: %q", s, ev.Title)
				}
			})
		})
	}
}

// TestPublisher_I18nMutationGuard is the paired-mutation test per §1.1. With
// the production-default NoopTranslator, the verbatim message ID is returned —
// NOT an English literal. A regression that re-hardcoded "Model Verification
// Started" would make ev.Title differ from the message ID, failing this test.
func TestPublisher_I18nMutationGuard(t *testing.T) {
	if got := tr("messaging.verification.started.title"); got != "messaging.verification.started.title" {
		t.Fatalf("NoopTranslator must return the bare id; got %q", got)
	}
	pub, broker := newTestPublisher(t)
	if err := pub.PublishVerificationStarted(context.Background(), 1, 1); err != nil {
		t.Fatalf("publish: %v", err)
	}
	ev := publishedEvent(t, broker)
	if ev.Title != "messaging.verification.started.title" {
		t.Fatalf("title regressed to a hardcoded literal: %q", ev.Title)
	}
	if strings.Contains(ev.Message, "Starting verification of") {
		t.Fatalf("message regressed to a hardcoded English literal: %q", ev.Message)
	}
	// The trData path must also return the bare id under NoopTranslator.
	if got := trData("messaging.team.selected.message", map[string]any{"team_size": 3}); got != "messaging.team.selected.message" {
		t.Fatalf("trData NoopTranslator must return the bare id; got %q", got)
	}
}
