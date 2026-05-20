package ai

import (
	"context"
	"strings"
	"testing"
	"time"
)

// fakeAITranslator returns "<TRANSLATED:msg_id>" so tests can assert the
// i18n-routed sentinel without coupling to the English bundle text.
// Anti-bluff per CONST-035 / Article XI §11.9: a test asserting the original
// literal would silently pass if the call-site bypassed the translator.
type fakeAITranslator struct{}

func (fakeAITranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}
func (fakeAITranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeAITranslator installs the fakeAITranslator, runs fn, then restores
// the prior translator.
func withFakeAITranslator(t *testing.T, fn func()) {
	t.Helper()
	prior := translator
	translator = fakeAITranslator{}
	defer func() { translator = prior }()
	fn()
}

// TestGenerateReasoning_Routed proves every recommendation reasoning
// fragment emitted by generateReasoning is i18n-routed rather than a
// hardcoded English literal. The model profile below triggers every
// reasoning branch (task suitability, cost, speed, reliability).
func TestGenerateReasoning_Routed(t *testing.T) {
	withFakeAITranslator(t, func() {
		sr := &SimpleRecommender{}
		model := ModelProfile{
			ID:           "test-model",
			Provider:     "test",
			TaskScores:   map[string]float64{"coding": 0.95},
			ResponseTime: 1 * time.Second,
			Reliability:  0.99,
			CostPerToken: 0.000001,
		}
		req := RecRequest{TaskType: "coding", MaxCost: 1.0}
		reasoning := sr.generateReasoning(model, req, 0.9)
		for _, frag := range strings.Split(reasoning, ", ") {
			if !strings.HasPrefix(frag, "<TRANSLATED:llmsverifier_ai_rec_reason_") {
				t.Errorf("reasoning fragment not i18n-routed: %q (full: %q)", frag, reasoning)
			}
		}
	})
}

// TestGenerateReasoning_FallbackRouted proves the empty-reasons fallback
// branch ("Good general-purpose model") is also i18n-routed.
func TestGenerateReasoning_FallbackRouted(t *testing.T) {
	withFakeAITranslator(t, func() {
		sr := &SimpleRecommender{}
		// No task scores, slow, unreliable, no cost limit => no reasons added.
		model := ModelProfile{
			ID:           "weak-model",
			Provider:     "test",
			TaskScores:   map[string]float64{},
			ResponseTime: 30 * time.Second,
			Reliability:  0.1,
		}
		req := RecRequest{TaskType: "coding"}
		reasoning := sr.generateReasoning(model, req, 0.1)
		if reasoning != "<TRANSLATED:llmsverifier_ai_rec_reason_good_general_purpose>" {
			t.Errorf("fallback reasoning not i18n-routed: %q", reasoning)
		}
	})
}

// TestGenerateReasoning_NoopDefault proves the production default
// (NoopTranslator) returns the messageID verbatim.
func TestGenerateReasoning_NoopDefault(t *testing.T) {
	sr := &SimpleRecommender{}
	model := ModelProfile{
		ID:           "weak-model",
		TaskScores:   map[string]float64{},
		ResponseTime: 30 * time.Second,
		Reliability:  0.1,
	}
	req := RecRequest{TaskType: "coding"}
	reasoning := sr.generateReasoning(model, req, 0.1)
	if reasoning != "llmsverifier_ai_rec_reason_good_general_purpose" {
		t.Errorf("NoopTranslator default did not return messageID verbatim: %q", reasoning)
	}
}

// TestTr_PairedMutation is the §1.1 paired-mutation guard: it confirms tr()
// genuinely routes through the translator. If tr() were mutated to return its
// argument verbatim (bypassing the translator), this assertion fails.
func TestTr_PairedMutation(t *testing.T) {
	withFakeAITranslator(t, func() {
		got := tr("llmsverifier_ai_rec_reason_fast_response")
		want := "<TRANSLATED:llmsverifier_ai_rec_reason_fast_response>"
		if got != want {
			t.Errorf("tr() not routed through translator: got %q want %q", got, want)
		}
	})
}
