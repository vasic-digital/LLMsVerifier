package monitoring

import (
	"context"
	"strings"
	"testing"
)

// fakeMonitorTranslator returns "<TRANSLATED:msg_id>" so tests can assert the
// sentinel without coupling to the English bundle text. Anti-bluff per
// CONST-035 / Article XI §11.9: a test asserting the original literal would
// silently pass if the call-site bypassed the translator.
type fakeMonitorTranslator struct{}

func (fakeMonitorTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}
func (fakeMonitorTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeMonitorTranslator installs the fakeMonitorTranslator, runs fn, then
// restores the prior translator.
func withFakeMonitorTranslator(t *testing.T, fn func()) {
	t.Helper()
	prior := translator
	translator = fakeMonitorTranslator{}
	defer func() { translator = prior }()
	fn()
}

// TestHealthChecker_DatabaseMessages_Routed proves the database health-check
// branches emit i18n-routed sentinels rather than hardcoded English. With the
// fake translator installed, every ComponentHealth.Message must carry the
// "<TRANSLATED:...>" prefix — if a branch still held a literal, the assertion
// would fail.
func TestHealthChecker_DatabaseMessages_Routed(t *testing.T) {
	withFakeMonitorTranslator(t, func() {
		// nil-database branch → not_configured
		hc := NewHealthChecker(nil)
		hc.checkDatabaseHealth()
		got := hc.components["database"].Message
		if !strings.HasPrefix(got, "<TRANSLATED:monitoring.database.") {
			t.Errorf("database message not i18n-routed: %q", got)
		}
	})
}

// TestHealthChecker_ComponentName_Routed proves the notification component's
// display name is i18n-routed at initialization time.
func TestHealthChecker_ComponentName_Routed(t *testing.T) {
	withFakeMonitorTranslator(t, func() {
		hc := NewHealthChecker(nil)
		name := hc.components["notifications"].Name
		if name != "<TRANSLATED:monitoring.component.notification_system>" {
			t.Errorf("notification component name not routed: %q", name)
		}
	})
}

// TestHealthChecker_SchedulerAndNotifications_Routed exercises the scheduler
// and notification health-check branches and asserts every emitted message is
// i18n-routed. The metricsTracker starts empty, so the scheduler reports
// not_running and notifications reports no_channels_configured — both
// migrated message IDs.
func TestHealthChecker_SchedulerAndNotifications_Routed(t *testing.T) {
	withFakeMonitorTranslator(t, func() {
		hc := NewHealthChecker(nil)
		hc.checkSchedulerHealth()
		hc.checkNotificationsHealth()

		sched := hc.components["scheduler"].Message
		if !strings.HasPrefix(sched, "<TRANSLATED:monitoring.scheduler.") {
			t.Errorf("scheduler message not i18n-routed: %q", sched)
		}
		notif := hc.components["notifications"].Message
		if !strings.HasPrefix(notif, "<TRANSLATED:monitoring.notifications.") {
			t.Errorf("notification message not i18n-routed: %q", notif)
		}
	})
}

// TestMonitoring_NoopTranslatorReturnsMessageID confirms the default
// NoopTranslator emits the messageID verbatim — the seam contract relied on
// by every consumer that has not installed a real bundle.
func TestMonitoring_NoopTranslatorReturnsMessageID(t *testing.T) {
	if got := tr("monitoring.database.healthy"); got != "monitoring.database.healthy" {
		t.Errorf("tr() with NoopTranslator = %q, want verbatim id", got)
	}
	got := trData("monitoring.api.error_rate_critical", map[string]any{"percent": 75.0})
	if got != "monitoring.api.error_rate_critical" {
		t.Errorf("trData() with NoopTranslator = %q, want verbatim id", got)
	}
}

// TestMonitoring_MutationGuard is the paired-mutation guard per §1.1: it
// asserts that the migrated health.go call sites genuinely route through the
// package translator. If a future edit reverts any Message assignment back to
// a hardcoded literal, the fake translator's sentinel would NOT appear and
// this test fails — the mutation is caught.
func TestMonitoring_MutationGuard(t *testing.T) {
	withFakeMonitorTranslator(t, func() {
		hc := NewHealthChecker(nil)
		// Run every component check; each must produce a routed Message.
		hc.checkAllComponents()
		for name, c := range hc.components {
			if c.Message == "" {
				continue
			}
			if !strings.HasPrefix(c.Message, "<TRANSLATED:") {
				t.Errorf("component %q Message bypassed i18n translator: %q", name, c.Message)
			}
		}
	})
}
