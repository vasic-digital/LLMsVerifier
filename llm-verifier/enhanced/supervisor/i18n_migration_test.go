package supervisor

import (
	"context"
	"strings"
	"testing"
)

// fakeSupervisorTranslator returns "<TRANSLATED:msg_id>" so tests can assert
// the i18n-routed sentinel without coupling to the English bundle text.
// Anti-bluff per CONST-035 / Article XI §11.9: a test asserting the original
// literal would silently pass if the call-site bypassed the translator.
type fakeSupervisorTranslator struct{}

func (fakeSupervisorTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}
func (fakeSupervisorTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeSupervisorTranslator installs the fakeSupervisorTranslator, runs fn,
// then restores the prior translator.
func withFakeSupervisorTranslator(t *testing.T, fn func()) {
	t.Helper()
	prior := translator
	translator = fakeSupervisorTranslator{}
	defer func() { translator = prior }()
	fn()
}

// TestAssistantResponses_Routed proves the AI assistant's conversational
// responses route every help/status/suggestion/configuration string through
// the i18n seam. With the fake translator installed, every migrated string
// must carry the "<TRANSLATED:supervisor.*>" prefix — if a branch still held
// an English literal, the assertion fails.
func TestAssistantResponses_Routed(t *testing.T) {
	withFakeSupervisorTranslator(t, func() {
		ai := &AIAssistant{context: make(map[string][]string)}

		if got := ai.generateHelpResponse(); !strings.HasPrefix(got, "<TRANSLATED:supervisor.assistant.help") {
			t.Errorf("help response not i18n-routed: %q", got)
		}
		if got := ai.generateStatusResponse(); !strings.HasPrefix(got, "<TRANSLATED:supervisor.assistant.status") {
			t.Errorf("status response not i18n-routed: %q", got)
		}
		if got := ai.generateSuggestionResponse("recommend a model"); !strings.HasPrefix(got, "<TRANSLATED:supervisor.assistant.model_recommendations") {
			t.Errorf("model suggestion not i18n-routed: %q", got)
		}
		if got := ai.generateSuggestionResponse("any tips"); !strings.HasPrefix(got, "<TRANSLATED:supervisor.assistant.smart_suggestions") {
			t.Errorf("smart suggestion not i18n-routed: %q", got)
		}
		if got := ai.generateConfigurationResponse("config help"); !strings.HasPrefix(got, "<TRANSLATED:supervisor.assistant.configuration") {
			t.Errorf("configuration response not i18n-routed: %q", got)
		}
		// generateGeneralResponse selects one of five routed strings.
		for _, msg := range []string{"a", "ab", "abc", "abcd", "abcde"} {
			if got := ai.generateGeneralResponse(msg); !strings.HasPrefix(got, "<TRANSLATED:supervisor.assistant.general.") {
				t.Errorf("general response not i18n-routed for %q: %q", msg, got)
			}
		}
	})
}

// TestRecommendations_Routed proves getRecommendations routes every
// recommendation bullet through the i18n seam across all score/failure
// permutations.
func TestRecommendations_Routed(t *testing.T) {
	withFakeSupervisorTranslator(t, func() {
		ai := &AIAssistant{}
		cases := []struct {
			score    float64
			failures int
		}{
			{50, 3}, // low score + failures
			{95, 0}, // good score, no failures
			{70, 0}, // low score, no failures
			{95, 2}, // good score, failures
		}
		for _, c := range cases {
			recs := ai.getRecommendations(c.score, c.failures)
			for _, line := range strings.Split(recs, "\n") {
				if !strings.HasPrefix(line, "<TRANSLATED:supervisor.recommendation.") {
					t.Errorf("recommendation bullet not i18n-routed (score=%.0f fail=%d): %q", c.score, c.failures, line)
				}
			}
		}
	})
}

// TestPluginDescriptions_Routed proves each built-in plugin's Description
// routes through the i18n seam.
func TestPluginDescriptions_Routed(t *testing.T) {
	withFakeSupervisorTranslator(t, func() {
		descs := map[string]string{
			"sentiment":   (&SentimentAnalysisPlugin{}).Description(),
			"code_review": (&CodeReviewPlugin{}).Description(),
			"performance": (&PerformanceAnalysisPlugin{}).Description(),
		}
		for name, d := range descs {
			if !strings.HasPrefix(d, "<TRANSLATED:supervisor.plugin.") {
				t.Errorf("%s plugin description not i18n-routed: %q", name, d)
			}
		}
	})
}

// TestSupervisorMutationGuard is the paired-mutation test per §1.1. With the
// production-default NoopTranslator, the verbatim message ID is returned —
// NOT an English literal. A regression that re-hardcoded "🤖 **LLM Verifier
// Assistant**" would make the help response differ from the message ID,
// failing this test.
func TestSupervisorMutationGuard(t *testing.T) {
	if got := tr("supervisor.assistant.help"); got != "supervisor.assistant.help" {
		t.Fatalf("NoopTranslator must return the bare id; got %q", got)
	}
	if got := trData("supervisor.plugin.enabled", map[string]any{"name": "x"}); got != "supervisor.plugin.enabled" {
		t.Fatalf("NoopTranslator (trData) must return the bare id; got %q", got)
	}

	ai := &AIAssistant{context: make(map[string][]string)}
	// Under NoopTranslator the help response is the bare message ID — proving
	// the literal "🤖 **LLM Verifier Assistant**" no longer lives in code.
	if help := ai.generateHelpResponse(); help != "supervisor.assistant.help" {
		t.Fatalf("help response not routed through the i18n seam: %q", help)
	}
	if strings.Contains(help(ai), "LLM Verifier Assistant") {
		t.Fatalf("help response regressed to a hardcoded English literal")
	}
	status := ai.generateStatusResponse()
	if strings.Contains(status, "Core Services") || !strings.HasPrefix(status, "supervisor.") {
		t.Fatalf("status response not routed through the i18n seam: %q", status)
	}
	for name, d := range map[string]string{
		"sentiment":   (&SentimentAnalysisPlugin{}).Description(),
		"code_review": (&CodeReviewPlugin{}).Description(),
		"performance": (&PerformanceAnalysisPlugin{}).Description(),
	} {
		if !strings.HasPrefix(d, "supervisor.plugin.") {
			t.Fatalf("%s plugin description not routed through the i18n seam: %q", name, d)
		}
	}
}

// help is a tiny indirection so the mutation guard can re-invoke the help
// generator without re-declaring the assistant.
func help(ai *AIAssistant) string { return ai.generateHelpResponse() }
