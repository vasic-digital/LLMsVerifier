package llmverifier

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"

	"digital.vasic.llmsverifier/config"
)

// const040ProbePositiveContent is the assistant message body a loopback
// endpoint returns so that ALL THREE CONST-040 capability probes added by
// change C4 (RAG / Skills / Plugins) reach a positive verdict:
//   - RAG      → cites the injected-document sentinel "zorblax-7742"
//   - Skills   → names the invoked skill "code_formatter"
//   - Plugins  → names the invoked plugin "weather_lookup"
//
// A model genuinely supporting these capabilities returns exactly this kind
// of grounded, skill/plugin-citing answer; a model that ignores them does
// not. The loopback server serving this content is a REAL HTTP round-trip
// (no simulation — §3 root CLAUDE.md), so the probe exercised here makes an
// actual wire call, not a stubbed one.
const const040ProbePositiveContent = "Using the code_formatter skill and the weather_lookup plugin, " +
	"the grounded answer from the provided document is zorblax-7742."

// TestCapabilityProbesCONST040 is the §11.4.115 RED-baseline-on-the-broken-
// artifact polarity test for change C4 (the per-capability probe path for
// the CONST-040 capabilities that had NO probe: RAG, Skills, Plugins —
// 10b_code_exact_change_spec.md §3 C4).
//
// Gap reproduced: before C4 the probe engine (llmverifier/verifier.go) has
// probes for MCP/LSP/ACP/Embeddings but NONE for RAG / Skills / Plugins, so
// a model that genuinely supports them can never be detected — a CONST-040
// hole (capabilities MUST be sourced from a real probe, never absent).
//
// Compile-agnostic oracle (§11.4.115): the probe methods are resolved by
// STRING NAME via reflection (reflect.Value.MethodByName), so this SAME test
// source carries NO compile-time reference to the new TestRAG/TestSkills/
// TestPlugins methods and therefore compiles on BOTH the pre-C4 (broken) and
// post-C4 (fixed) artifacts. On the broken artifact the methods are absent
// (reflect returns an invalid Value); on the fixed artifact they exist and
// are invoked against a real loopback endpoint that returns a positive
// answer, and MUST return true.
//
// Polarity switch (§11.4.115):
//
//	RED_MODE=1           — reproduce the defect: assert the probe methods are
//	    ABSENT. PASSes on the BROKEN (pre-C4) artifact (proof the gap is
//	    real), FAILs on the FIXED artifact.
//	RED_MODE=0 (default) — standing GREEN regression guard: assert the probe
//	    methods EXIST and each returns true against a capability-positive
//	    loopback endpoint. FAILs on BROKEN, PASSes on FIXED. Default so
//	    `go test ./...` runs the standing guard on the fixed tree per
//	    §11.4.135 (guard runs on every build).
func TestCapabilityProbesCONST040(t *testing.T) {
	redMode := os.Getenv("RED_MODE")
	if redMode == "" {
		redMode = "0"
	}

	// Real loopback endpoint returning a CONST-040-positive answer for every
	// probe's chat/completions call.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"` +
			const040ProbePositiveContent + `"}}]}`))
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

	// The three CONST-040 capabilities that had no probe before C4.
	probeMethods := []string{"TestRAG", "TestSkills", "TestPlugins"}

	vv := reflect.ValueOf(v)
	callArgs := []reflect.Value{
		reflect.ValueOf(client),
		reflect.ValueOf("probe-model"),
		reflect.ValueOf(ctx),
	}

	for _, name := range probeMethods {
		m := vv.MethodByName(name)

		switch redMode {
		case "1":
			// Reproduce-and-assert-defect-present on the broken artifact: the
			// probe does not exist yet.
			if m.IsValid() {
				t.Fatalf("RED_MODE=1: expected CONST-040 probe %q to be ABSENT "+
					"(defect present pre-C4) but it exists", name)
			}
			t.Logf("RED_MODE=1 PASS: defect reproduced — CONST-040 probe %q absent (no producer).", name)
		case "0":
			// Standing GREEN regression guard on the fixed artifact.
			if !m.IsValid() {
				t.Fatalf("RED_MODE=0: CONST-040 probe %q not implemented — the "+
					"capability has no probe producer, so a model supporting it "+
					"can never be detected (10b spec §3 C4).", name)
			}
			// Real loopback wire call: the probe must reach a positive verdict
			// against a capability-positive endpoint.
			out := m.Call(callArgs)
			if len(out) != 1 || out[0].Kind() != reflect.Bool {
				t.Fatalf("RED_MODE=0: probe %q must return a single bool, got %v", name, out)
			}
			if !out[0].Bool() {
				t.Fatalf("RED_MODE=0: probe %q returned false against a "+
					"capability-positive loopback endpoint; it must detect the "+
					"capability from the real response (10b spec §3 C4).", name)
			}
			t.Logf("RED_MODE=0 PASS: CONST-040 probe %q present and detected the capability via a real wire call.", name)
		default:
			t.Fatalf("unknown RED_MODE=%q (expected 0 or 1)", redMode)
		}
	}
}
