package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeTranslator returns "<TRANSLATED:msg_id>" so tests assert the sentinel
// rather than the English bundle text. Anti-bluff per CONST-035 / Article
// XI §11.9: a test asserting the original literal would silently pass if a
// call-site bypassed the translator.
type fakeTranslator struct{}

func (fakeTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}
func (fakeTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// TestModelVerifyTrRoutesThroughTranslator proves tr() actually delegates to
// the package translator. If tr() regressed to returning a hardcoded
// literal, the sentinel would not appear. Paired-mutation: swap the body
// of tr() to `return id` minus the translator call and this fails.
func TestModelVerifyTrRoutesThroughTranslator(t *testing.T) {
	prior := translator
	translator = fakeTranslator{}
	defer func() { translator = prior }()

	for _, id := range []string{
		"llmsverifier_modelverify_banner",
		"llmsverifier_modelverify_no_providers",
		"llmsverifier_modelverify_model_passed",
		"llmsverifier_modelverify_model_failed",
		"llmsverifier_modelverify_config_complete",
	} {
		got := tr(id)
		want := "<TRANSLATED:" + id + ">"
		if got != want {
			t.Fatalf("tr(%q) = %q; want %q — translator seam bypassed", id, got, want)
		}
	}
}

// TestModelVerifyTrDataRoutesThroughTranslator proves trData() delegates to
// the translator for parameterised call sites.
func TestModelVerifyTrDataRoutesThroughTranslator(t *testing.T) {
	prior := translator
	translator = fakeTranslator{}
	defer func() { translator = prior }()

	id := "llmsverifier_modelverify_total_providers"
	got := trData(id, map[string]any{"count": 7})
	want := "<TRANSLATED:" + id + ">"
	if got != want {
		t.Fatalf("trData(%q) = %q; want %q — translator seam bypassed", id, got, want)
	}
}

// TestModelVerifyNoopReturnsIDVerbatim asserts the default NoopTranslator
// returns the message ID unchanged — required so call sites stay testable.
func TestModelVerifyNoopReturnsIDVerbatim(t *testing.T) {
	// translator is NoopTranslator{} by default.
	id := "llmsverifier_modelverify_banner"
	if got := tr(id); got != id {
		t.Fatalf("tr(%q) with NoopTranslator = %q; want verbatim ID", id, got)
	}
}

// TestModelVerifyBundleNoRevert scans the active English bundle and asserts
// every round-312 message ID referenced by cmd/model-verification/main.go
// is still defined. A revert that drops a bundle entry (re-hardcoding the
// literal in source) is caught here. Paired-mutation: delete any
// llmsverifier_modelverify_* entry from the bundle and this fails.
func TestModelVerifyBundleNoRevert(t *testing.T) {
	// Locate the bundle from the test working directory (cmd/model-verification).
	bundlePath := filepath.Join("..", "..", "pkg", "i18n", "bundles", "active.en.yaml")
	data, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("read bundle %s: %v", bundlePath, err)
	}
	bundle := string(data)

	required := []string{
		"llmsverifier_modelverify_banner",
		"llmsverifier_modelverify_providers_heading",
		"llmsverifier_modelverify_no_providers",
		"llmsverifier_modelverify_provider_no_key",
		"llmsverifier_modelverify_provider_configured",
		"llmsverifier_modelverify_total_providers",
		"llmsverifier_modelverify_model_not_found",
		"llmsverifier_modelverify_provider_client_not_found",
		"llmsverifier_modelverify_model_passed",
		"llmsverifier_modelverify_model_failed",
		"llmsverifier_modelverify_found_verified_models",
		"llmsverifier_modelverify_summary_models",
		"llmsverifier_modelverify_verifying_all",
		"llmsverifier_modelverify_summary_across_providers",
		"llmsverifier_modelverify_generating_config",
		"llmsverifier_modelverify_config_saved_to",
		"llmsverifier_modelverify_stats_heading",
		"llmsverifier_modelverify_stats_section",
		"llmsverifier_modelverify_stat_total_scanned",
		"llmsverifier_modelverify_stat_verified",
		"llmsverifier_modelverify_stat_rate",
		"llmsverifier_modelverify_stat_providers",
		"llmsverifier_modelverify_stat_enabled",
		"llmsverifier_modelverify_stat_strict",
		"llmsverifier_modelverify_config_complete",
		"llmsverifier_modelverify_provider_breakdown",
	}
	for _, id := range required {
		if !strings.Contains(bundle, id+":") {
			t.Fatalf("bundle %s missing message ID %q — round-312 i18n migration reverted", bundlePath, id)
		}
	}
}

// TestModelVerifySourceNoHardcodedRevert scans main.go and asserts the
// migrated English literals are no longer present as source-code string
// literals. A re-hardcode of any migrated string is caught here.
func TestModelVerifySourceNoHardcodedRevert(t *testing.T) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	src := string(data)

	// These exact literals were migrated out — they MUST NOT reappear as
	// quoted string literals in source.
	forbidden := []string{
		`"📋 Available Providers:"`,
		`"No providers registered. Please set API keys in environment variables."`,
		`"\n🎉 Model PASSED mandatory verification!"`,
		`"\n❌ Model FAILED mandatory verification!"`,
		`"\n🎉 Verified configuration generation complete!"`,
		`"\n🔧 Generating Verified Configuration"`,
	}
	for _, lit := range forbidden {
		if strings.Contains(src, lit) {
			t.Fatalf("main.go re-hardcoded migrated literal %s — round-312 migration reverted", lit)
		}
	}
}
