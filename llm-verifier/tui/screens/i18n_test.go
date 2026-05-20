package screens

import (
	"context"
	"strings"
	"testing"

	"digital.vasic.llmsverifier/client"
	"digital.vasic.llmsverifier/pkg/i18n"
)

// fakeScreensTranslator returns "<TRANSLATED:msg_id>" for every message ID,
// proving that dashboard / verification screen render call sites route their
// user-facing text through the i18n seam rather than emitting hardcoded
// English literals (CONST-046 anti-bluff invariant per Article XI §11.9).
type fakeScreensTranslator struct{}

func (fakeScreensTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

func (fakeScreensTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeScreensTranslator swaps the package translator for the duration of a
// test and restores the prior backend afterwards.
func withFakeScreensTranslator(t *testing.T) {
	t.Helper()
	prev := translator
	translator = fakeScreensTranslator{}
	t.Cleanup(func() { translator = prev })
}

// TestTrScreen_RoutesThroughTranslator is the positive case: with the fake
// backend installed, trScreen() must emit the sentinel, not the message ID
// verbatim.
func TestTrScreen_RoutesThroughTranslator(t *testing.T) {
	withFakeScreensTranslator(t)
	got := trScreen("screens.dashboard.title")
	want := "<TRANSLATED:screens.dashboard.title>"
	if got != want {
		t.Fatalf("trScreen routed wrong: got %q want %q", got, want)
	}
}

// TestTrScreen_NoopReturnsID confirms the default NoopTranslator returns the
// message ID verbatim — the documented incremental-migration fallback.
func TestTrScreen_NoopReturnsID(t *testing.T) {
	translator = i18n.NoopTranslator{}
	if got := trScreen("screens.verification.title"); got != "screens.verification.title" {
		t.Fatalf("NoopTranslator must return id verbatim, got %q", got)
	}
}

// TestDashboardView_RoutesThroughTranslator is the paired-mutation guard for
// dashboard.go: with the fake backend installed, the rendered View() MUST
// contain the translated sentinels for the migrated labels and MUST NOT
// contain the original English literals. If a future edit reintroduces a
// hardcoded literal, this test fails.
func TestDashboardView_RoutesThroughTranslator(t *testing.T) {
	withFakeScreensTranslator(t)
	d := NewDashboardScreen(&client.Client{})
	d.width = 120
	d.height = 40
	d.stats.TotalModels = 10
	d.stats.VerifiedModels = 7
	out := d.View()

	// lipgloss word-wraps long sentinels inside fixed-width boxes; assert on
	// the wrapped-but-stable fragments rather than the full sentinel string.
	mustContain := []string{
		"<TRANSLATED:screens.dashboard.title>",
		"<TRANSLATED:scre", // statBox + actionButton sentinels wrap at box width
		"total_models",     // proves total-models stat ID routed
		"at.verified",      // proves verified stat ID routed
		"<TRANSLATED:screens.dashboard.progress_overview>",
		"<TRANSLATED:screens.dashboard.quick_actions>",
		"desc>", // proves action-button .desc sentinels routed (unique suffix)
	}
	for _, s := range mustContain {
		if !strings.Contains(out, s) {
			t.Errorf("dashboard View() missing translated sentinel fragment %q", s)
		}
	}

	mustNotContain := []string{
		"Total Models", "Quick Actions:", "Progress Overview", "Run Verification",
	}
	for _, s := range mustNotContain {
		if strings.Contains(out, s) {
			t.Errorf("dashboard View() leaked hardcoded literal %q — CONST-046 violation", s)
		}
	}
}

// TestVerificationView_RoutesThroughTranslator is the paired-mutation guard
// for verification.go: the new-verification form and action buttons MUST
// render translated sentinels, never the English literals.
func TestVerificationView_RoutesThroughTranslator(t *testing.T) {
	withFakeScreensTranslator(t)
	v := NewVerificationScreen(&client.Client{})
	v.width = 120
	v.height = 40
	v.showNewForm = true
	out := v.View()

	mustContain := []string{
		"<TRANSLATED:screens.verification.title>",
		"<TRANSLATED:screens.verification.no_history>",
		"<TRANSLATED:screens.verification.actions_heading>",
		"<TRANSLATED:screens.verification.new_form.heading>",
		"<TRANSLATED:screens.verification.new_form.hint>",
	}
	for _, s := range mustContain {
		if !strings.Contains(out, s) {
			t.Errorf("verification View() missing translated sentinel %q", s)
		}
	}

	mustNotContain := []string{
		"No verification history", "New Verification:", "Press Enter to start verification",
	}
	for _, s := range mustNotContain {
		if strings.Contains(out, s) {
			t.Errorf("verification View() leaked hardcoded literal %q — CONST-046 violation", s)
		}
	}
}
