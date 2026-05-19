package events

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeEventsTranslator returns "<TRANSLATED:msg_id>" so tests can assert the
// i18n-routed sentinel without coupling to the English bundle text.
// Anti-bluff per CONST-035 / Article XI §11.9: a test asserting the original
// literal would silently pass if the call-site bypassed the translator.
type fakeEventsTranslator struct{}

func (fakeEventsTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}
func (fakeEventsTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeEventsTranslator installs the fakeEventsTranslator, runs fn, then
// restores the prior translator.
func withFakeEventsTranslator(t *testing.T, fn func()) {
	t.Helper()
	prior := translator
	translator = fakeEventsTranslator{}
	defer func() { translator = prior }()
	fn()
}

// recordingSubscriber implements EventSubscriber and records every event it
// receives — a real subscriber, not a mock, so the test exercises the genuine
// publish path through EventManager.
type recordingSubscriber struct {
	mu     sync.Mutex
	events []*Event
}

func (r *recordingSubscriber) ReceiveEvent(event *Event) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
	return nil
}
func (r *recordingSubscriber) GetID() string { return "i18n-migration-recorder" }
func (r *recordingSubscriber) GetSupportedEventTypes() []EventType {
	// Subscribe to every event type the publishers under test emit so
	// processEvent's interest filter delivers them all.
	return []EventType{
		EventVerificationStarted, EventVerificationCompleted, EventVerificationFailed,
		EventScoreChanged, EventIssueDetected, EventIssueResolved,
		EventConfigExported, EventDatabaseMigration,
		EventClientConnected, EventClientDisconnected,
		EventSystemHealthChanged, EventSecurityAlert,
	}
}
func (r *recordingSubscriber) IsActive() bool { return true }

func (r *recordingSubscriber) last() *Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.events) == 0 {
		return nil
	}
	return r.events[len(r.events)-1]
}

// newRecordingPublisher wires a real EventManager + recordingSubscriber and
// returns a publisher plus the recorder. The EventPublisher uses a nil db —
// publishAndStoreEvent tolerates a nil db (it only logs).
func newRecordingPublisher(t *testing.T) (*EventPublisher, *recordingSubscriber, func()) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	em := NewEventManager(ctx, 64, 1)
	rec := &recordingSubscriber{}
	if err := em.Subscribe(rec); err != nil {
		cancel()
		t.Fatalf("subscribe failed: %v", err)
	}
	return NewEventPublisher(em, nil), rec, cancel
}

// waitFor polls until the recorder has captured at least one event.
func waitFor(t *testing.T, rec *recordingSubscriber) *Event {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if ev := rec.last(); ev != nil {
			return ev
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("no event recorded within deadline")
	return nil
}

// TestEventPublisher_TitlesRouted proves event publishers route their title +
// message through the i18n seam. With the fake translator installed, every
// Title/Message must carry the "<TRANSLATED:events.*>" prefix — if a branch
// still held an English literal, the assertion fails.
func TestEventPublisher_TitlesRouted(t *testing.T) {
	cases := []struct {
		name     string
		titlePfx string
		publish  func(ep *EventPublisher) error
	}{
		{"verification_started", "<TRANSLATED:events.verification.started.",
			func(ep *EventPublisher) error { return ep.PublishVerificationStarted(3, 2) }},
		{"verification_completed", "<TRANSLATED:events.verification.completed.",
			func(ep *EventPublisher) error { return ep.PublishVerificationCompleted(time.Second, 5, 1) }},
		{"verification_failed", "<TRANSLATED:events.verification.failed.",
			func(ep *EventPublisher) error { return ep.PublishVerificationFailed("boom") }},
		{"issue_detected", "<TRANSLATED:events.issue.detected.",
			func(ep *EventPublisher) error { return ep.PublishIssueDetected(1, "perf", "warning", "Slow", "took too long") }},
		{"issue_resolved", "<TRANSLATED:events.issue.resolved.",
			func(ep *EventPublisher) error { return ep.PublishIssueResolved(1, 9, "patched") }},
		{"client_connected", "<TRANSLATED:events.client.connected.",
			func(ep *EventPublisher) error { return ep.PublishClientConnected("c1", "cli") }},
		{"client_disconnected", "<TRANSLATED:events.client.disconnected.",
			func(ep *EventPublisher) error { return ep.PublishClientDisconnected("c1", "cli") }},
		{"config_exported", "<TRANSLATED:events.config.exported.",
			func(ep *EventPublisher) error { return ep.PublishConfigExported("openai", 4) }},
		{"security_alert", "<TRANSLATED:events.security.alert.",
			func(ep *EventPublisher) error {
				return ep.PublishSecurityAlert("intrusion", "blocked", map[string]interface{}{})
			}},
		{"migration_completed", "<TRANSLATED:events.migration.completed.",
			func(ep *EventPublisher) error { return ep.PublishDatabaseMigration(7, "add index", true) }},
		{"migration_failed", "<TRANSLATED:events.migration.failed.",
			func(ep *EventPublisher) error { return ep.PublishDatabaseMigration(7, "add index", false) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			withFakeEventsTranslator(t, func() {
				ep, rec, cancel := newRecordingPublisher(t)
				defer cancel()
				if err := c.publish(ep); err != nil {
					t.Fatalf("publish returned error: %v", err)
				}
				ev := waitFor(t, rec)
				if !strings.HasPrefix(ev.Title, c.titlePfx) {
					t.Errorf("Title not i18n-routed: got %q want prefix %q", ev.Title, c.titlePfx)
				}
				if !strings.HasPrefix(ev.Message, "<TRANSLATED:events.") {
					t.Errorf("Message not i18n-routed: got %q", ev.Message)
				}
			})
		})
	}
}

// TestEventPublisher_ScoreChangedRouted covers both score-direction branches.
func TestEventPublisher_ScoreChangedRouted(t *testing.T) {
	withFakeEventsTranslator(t, func() {
		ep, rec, cancel := newRecordingPublisher(t)
		defer cancel()

		if err := ep.PublishScoreChanged(1, 10, 20, "code"); err != nil {
			t.Fatalf("publish improved: %v", err)
		}
		if up := waitFor(t, rec); !strings.HasPrefix(up.Title, "<TRANSLATED:events.score.improved.") {
			t.Errorf("score-improved title not routed: %q", up.Title)
		}

		if err := ep.PublishScoreChanged(1, 20, 10, "code"); err != nil {
			t.Fatalf("publish decreased: %v", err)
		}
		if down := waitFor(t, rec); !strings.HasPrefix(down.Title, "<TRANSLATED:events.score.") {
			t.Errorf("score-decreased title not routed: %q", down.Title)
		}
	})
}

// TestEventPublisher_HealthBranchesRouted covers every system-health branch.
func TestEventPublisher_HealthBranchesRouted(t *testing.T) {
	for _, status := range []string{"healthy", "degraded", "unhealthy", "critical", "weird"} {
		s := status
		t.Run(s, func(t *testing.T) {
			withFakeEventsTranslator(t, func() {
				ep, rec, cancel := newRecordingPublisher(t)
				defer cancel()
				if err := ep.PublishSystemHealthChanged(s, map[string]interface{}{}); err != nil {
					t.Fatalf("publish health %q: %v", s, err)
				}
				ev := waitFor(t, rec)
				if !strings.HasPrefix(ev.Title, "<TRANSLATED:events.health.") {
					t.Errorf("health status %q title not routed: %q", s, ev.Title)
				}
			})
		})
	}
}

// TestEventPublisher_MutationGuard is the paired-mutation test per §1.1. With
// the production-default NoopTranslator, the verbatim message ID is returned —
// NOT an English literal. A regression that re-hardcoded "Model Verification
// Started" would make ev.Title differ from the message ID, failing this test.
func TestEventPublisher_MutationGuard(t *testing.T) {
	if got := tr("events.verification.started.title"); got != "events.verification.started.title" {
		t.Fatalf("NoopTranslator must return the bare id; got %q", got)
	}
	ep, rec, cancel := newRecordingPublisher(t)
	defer cancel()
	if err := ep.PublishVerificationStarted(1, 1); err != nil {
		t.Fatalf("publish: %v", err)
	}
	ev := waitFor(t, rec)
	if ev.Title != "events.verification.started.title" {
		t.Fatalf("title regressed to a hardcoded literal: %q", ev.Title)
	}
	if strings.Contains(ev.Message, "Starting verification of") {
		t.Fatalf("message regressed to a hardcoded English literal: %q", ev.Message)
	}
}
