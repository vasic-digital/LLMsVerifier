package selection

import (
	"context"
	"strings"
	"testing"
)

// fakeSelectionTranslator returns "<TRANSLATED:msg_id>" so tests assert the
// i18n-routed sentinel rather than English literals. Anti-bluff per CONST-035:
// a test asserting the English text would still pass if the call site bypassed
// the translator entirely.
type fakeSelectionTranslator struct{}

func (fakeSelectionTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}
func (fakeSelectionTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

func withFakeSelectionTranslator(t *testing.T, fn func()) {
	t.Helper()
	prior := translator
	SetTranslator(fakeSelectionTranslator{})
	defer func() { translator = prior }()
	fn()
}

// TestSelectionTr_RoutesThroughTranslator proves tr consults the active
// translator instead of echoing the messageID.
func TestSelectionTr_RoutesThroughTranslator(t *testing.T) {
	withFakeSelectionTranslator(t, func() {
		ids := []string{
			ReasonPaidCreditAvailable,
			ReasonFreeCreditExhausted,
			ReasonFreeUnknownCredit,
			ReasonPaidUnknownCredit,
			ReasonFellBackToFree,
			ReasonFellBackToPaid,
			ReasonNoCandidates,
			ReasonNoEligibleCandidate,
			ReasonUnknownCreditRejected,
		}
		for _, id := range ids {
			got := tr(id)
			want := "<TRANSLATED:" + id + ">"
			if got != want {
				t.Fatalf("tr(%q) = %q, want %q", id, got, want)
			}
		}
	})
}

// TestDecisionReason_IsLocalised proves Decision.Reason is produced through
// the i18n seam and that ReasonID stays the stable machine-readable key.
func TestDecisionReason_IsLocalised(t *testing.T) {
	withFakeSelectionTranslator(t, func() {
		d := mustSelect(t, mixedPool(), creditAvailable(), basePolicy(UnknownCreditPreferFree))
		if d.ReasonID != ReasonPaidCreditAvailable {
			t.Fatalf("ReasonID = %q", d.ReasonID)
		}
		if !strings.HasPrefix(d.Reason, "<TRANSLATED:") {
			t.Fatalf("Reason = %q, expected it to route through the translator", d.Reason)
		}
	})
}

// TestSetTranslator_NilRestoresNoop guards the nil path so a consumer clearing
// its translator does not panic every subsequent selection.
func TestSetTranslator_NilRestoresNoop(t *testing.T) {
	prior := translator
	defer func() { translator = prior }()

	SetTranslator(nil)
	if got := tr(ReasonNoCandidates); got != ReasonNoCandidates {
		t.Fatalf("tr with nil translator = %q, want the messageID verbatim", got)
	}
}
