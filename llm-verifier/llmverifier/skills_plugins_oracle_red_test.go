package llmverifier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"digital.vasic.llmsverifier/config"
)

// keywordEchoOnlyContent is an assistant message that merely REPEATS the words
// "skill" and "plugin" without genuinely invoking anything and without echoing
// any grounded sentinel. Under the loose keyword-echo oracle (review advisory
// 2 — Contains(content,"skill") / Contains(content,"plugin")) TestSkills and
// TestPlugins FALSELY pass on this; under the tightened grounded-sentinel oracle
// (matching TestRAG's zorblax-7742 style) they correctly reject it. Served over
// a REAL loopback HTTP round-trip (no simulation — §3 root CLAUDE.md).
const keywordEchoOnlyContent = "Sure, I can help with that skill request, and I can use a plugin as well."

// TestSkillsPluginsOracleRED — §11.4.115 RED-baseline-on-the-broken-artifact for
// the weak Skills/Plugins probe oracle (review advisory 2).
//
// Gap: TestSkills/TestPlugins verdict on a loose keyword echo
// (Contains(content,"skill") / Contains(content,"plugin")), so a model that
// merely repeats the word — without genuinely invoking the advertised
// skill/plugin — trips a false positive. TestRAG (C4) already uses a strong
// grounded-sentinel oracle (zorblax-7742); this test drives the same style into
// Skills/Plugins.
//
// Polarity switch (§11.4.115): RED_MODE=1 reproduces the false positive on the
// pre-fix (loose-oracle) artifact and PASSes there; RED_MODE=0 (default) is the
// standing GREEN regression guard that FAILs on the broken oracle and PASSes
// once the grounded-sentinel oracle lands. Registered in the §11.4.135 standing
// regression-guard suite.
func TestSkillsPluginsOracleRED(t *testing.T) {
	redMode := os.Getenv("RED_MODE")
	if redMode == "" {
		redMode = "0"
	}

	// Real loopback endpoint returning keyword-echo-only content for every
	// chat/completions call.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"` +
			keywordEchoOnlyContent + `"}}]}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		Global: config.GlobalConfig{
			BaseURL:      server.URL,
			APIKey:       "loopback-key",
			DefaultModel: "probe-model",
		},
	}
	v := New(cfg)
	client := v.GetGlobalClient()
	ctx := context.Background()

	probes := map[string]bool{
		"TestSkills":  v.TestSkills(client, "probe-model", ctx),
		"TestPlugins": v.TestPlugins(client, "probe-model", ctx),
	}

	for name, passed := range probes {
		switch redMode {
		case "1": // reproduce the false positive on the broken (loose-oracle) artifact
			if !passed {
				t.Fatalf("RED_MODE=1: expected %s to FALSELY pass on keyword-echo-only content "+
					"(loose-oracle defect), got false", name)
			}
			t.Logf("RED_MODE=1 PASS: defect reproduced — %s false-positives on a bare keyword echo.", name)
		case "0": // guard: FAILs on broken oracle, PASSes once sentinel oracle lands
			if passed {
				t.Fatalf("RED_MODE=0: %s still passes on keyword-echo-only content — the oracle is not "+
					"grounded; a mere keyword echo must NOT be a positive verdict (tighten to a "+
					"grounded-sentinel oracle like TestRAG).", name)
			}
			t.Logf("RED_MODE=0 PASS: %s correctly rejects a bare keyword echo (grounded-sentinel oracle).", name)
		default:
			t.Fatalf("unknown RED_MODE=%q (expected 0 or 1)", redMode)
		}
	}
}
