package capabilities

import (
	"context"
	"strings"
	"testing"
)

// fakeCapsTranslator returns "<TRANSLATED:msg_id>" so tests can assert the
// i18n-routed sentinel without coupling to the English bundle text.
// Anti-bluff per CONST-035 / Article XI §11.9: a test asserting the
// original English literal would silently pass if the call-site bypassed
// the translator entirely.
type fakeCapsTranslator struct{}

func (fakeCapsTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}
func (fakeCapsTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeCapsTranslator installs the fakeCapsTranslator, runs fn, then
// restores the prior translator.
func withFakeCapsTranslator(t *testing.T, fn func()) {
	t.Helper()
	prior := translator
	translator = fakeCapsTranslator{}
	defer func() { translator = prior }()
	fn()
}

// TestCapsTr_RoutesThroughTranslator proves the package-level tr helper
// actually consults the active translator rather than echoing the
// messageID. With the fake translator installed every lookup must carry
// the "<TRANSLATED:...>" prefix. This is the paired mutation for the
// capabilities i18n seam: if a future change made tr() return its
// argument verbatim, this test fails.
func TestCapsTr_RoutesThroughTranslator(t *testing.T) {
	withFakeCapsTranslator(t, func() {
		ids := []string{
			"llmsverifier_capabilities_rec_http3_unsupported_agent",
			"llmsverifier_capabilities_rec_http3_unsupported_any",
			"llmsverifier_capabilities_rec_deepseek_no_streaming",
			"llmsverifier_capabilities_rec_save_config_failed",
		}
		for _, id := range ids {
			got := tr(id)
			want := "<TRANSLATED:" + id + ">"
			if got != want {
				t.Errorf("tr(%q) = %q, not i18n-routed (want %q)", id, got, want)
			}
		}
	})
}

// TestCapsTr_NoopReturnsMessageIDVerbatim proves the default
// NoopTranslator path: without an override, tr returns the messageID
// itself. This guarantees the migration introduces no panic and the seam
// degrades gracefully.
func TestCapsTr_NoopReturnsMessageIDVerbatim(t *testing.T) {
	const id = "llmsverifier_capabilities_rec_deepseek_no_streaming"
	if got := tr(id); got != id {
		t.Errorf("NoopTranslator tr(%q) = %q, want verbatim messageID", id, got)
	}
}

// TestGenerateForAgent_HTTP3RecommendationIsRouted proves the
// GenerateForAgent HTTP/3 recommendation funnels through the translator
// rather than appending a hardcoded English literal. With the fake
// translator installed the emitted recommendation must be the i18n-routed
// sentinel for an agent that does not support HTTP/3.
func TestGenerateForAgent_HTTP3RecommendationIsRouted(t *testing.T) {
	withFakeCapsTranslator(t, func() {
		cg := NewConfigGenerator("localhost", 8080)
		agents := GetAllCLIAgents()
		checked := false
		for _, name := range agents {
			caps := GetCLIAgentCapabilities(name)
			if caps == nil || caps.Network.HTTP3Supported {
				continue
			}
			cfg, err := cg.GenerateForAgent(name, nil)
			if err != nil {
				t.Fatalf("GenerateForAgent(%q) error: %v", name, err)
			}
			found := false
			for _, rec := range cfg.Recommendations {
				if strings.Contains(rec, "<TRANSLATED:llmsverifier_capabilities_rec_http3") {
					found = true
				}
				if strings.Contains(rec, "HTTP/3 is not supported by this CLI agent") {
					t.Errorf("agent %q recommendation bypassed translator: %q", name, rec)
				}
			}
			if !found {
				t.Errorf("agent %q missing i18n-routed HTTP/3 recommendation", name)
			}
			checked = true
		}
		if !checked {
			t.Skip("SKIP-OK: no registered CLI agent lacks HTTP/3 support to exercise the recommendation path")
		}
	})
}
