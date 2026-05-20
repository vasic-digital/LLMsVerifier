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

// TestTestsuiteTrRoutesThroughTranslator proves tr() actually delegates to
// the package translator. If tr() regressed to returning a hardcoded
// literal, the sentinel would not appear. Paired-mutation: swap the body
// of tr() to `return id` minus the translator call and this fails.
func TestTestsuiteTrRoutesThroughTranslator(t *testing.T) {
	prior := translator
	translator = fakeTranslator{}
	defer func() { translator = prior }()

	for _, id := range []string{
		"llmsverifier_testsuite_title",
		"llmsverifier_testsuite_err_create_args",
		"llmsverifier_testsuite_available_heading",
		"llmsverifier_testsuite_execution_results_heading",
		"llmsverifier_testsuite_use_list_hint",
	} {
		got := tr(id)
		want := "<TRANSLATED:" + id + ">"
		if got != want {
			t.Fatalf("tr(%q) = %q; want %q — translator seam bypassed", id, got, want)
		}
	}
}

// TestTestsuiteTrDataRoutesThroughTranslator proves trData() delegates to
// the translator for parameterised call sites.
func TestTestsuiteTrDataRoutesThroughTranslator(t *testing.T) {
	prior := translator
	translator = fakeTranslator{}
	defer func() { translator = prior }()

	id := "llmsverifier_testsuite_created"
	got := trData(id, map[string]any{"name": "demo", "id": "suite-1"})
	want := "<TRANSLATED:" + id + ">"
	if got != want {
		t.Fatalf("trData(%q) = %q; want %q — translator seam bypassed", id, got, want)
	}
}

// TestTestsuiteNoopReturnsIDVerbatim asserts the default NoopTranslator
// returns the message ID unchanged — required so call sites stay testable.
func TestTestsuiteNoopReturnsIDVerbatim(t *testing.T) {
	// translator is NoopTranslator{} by default.
	id := "llmsverifier_testsuite_title"
	if got := tr(id); got != id {
		t.Fatalf("tr(%q) with NoopTranslator = %q; want verbatim ID", id, got)
	}
}

// TestTestsuiteBundleNoRevert scans the active English bundle and asserts
// every round-384 message ID referenced by cmd/testsuite/main.go is still
// defined. A revert that drops a bundle entry (re-hardcoding the literal in
// source) is caught here. Paired-mutation: delete any
// llmsverifier_testsuite_* entry from the bundle and this fails.
func TestTestsuiteBundleNoRevert(t *testing.T) {
	bundlePath := filepath.Join("..", "..", "pkg", "i18n", "bundles", "active.en.yaml")
	data, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("read bundle %s: %v", bundlePath, err)
	}
	bundle := string(data)

	required := []string{
		"llmsverifier_testsuite_unknown_command",
		"llmsverifier_testsuite_title",
		"llmsverifier_testsuite_err_create_args",
		"llmsverifier_testsuite_err_run_args",
		"llmsverifier_testsuite_err_export_args",
		"llmsverifier_testsuite_err_import_args",
		"llmsverifier_testsuite_created",
		"llmsverifier_testsuite_test_cases_count",
		"llmsverifier_testsuite_available_heading",
		"llmsverifier_testsuite_running",
		"llmsverifier_testsuite_execution_results_heading",
		"llmsverifier_testsuite_total_tests",
		"llmsverifier_testsuite_average_score",
		"llmsverifier_testsuite_average_duration",
		"llmsverifier_testsuite_total_duration",
		"llmsverifier_testsuite_p95_duration",
		"llmsverifier_testsuite_individual_results_heading",
		"llmsverifier_testsuite_exported_to",
		"llmsverifier_testsuite_imported",
		"llmsverifier_testsuite_creating_templates",
		"llmsverifier_testsuite_created_template",
		"llmsverifier_testsuite_use_list_hint",
	}
	for _, id := range required {
		if !strings.Contains(bundle, id+":") {
			t.Fatalf("bundle %s missing message ID %q — round-384 i18n migration reverted", bundlePath, id)
		}
	}
}

// TestTestsuiteSourceNoHardcodedRevert scans main.go and asserts the
// migrated English literals are no longer present as source-code string
// literals. A re-hardcode of any migrated string is caught here.
func TestTestsuiteSourceNoHardcodedRevert(t *testing.T) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	src := string(data)

	forbidden := []string{
		`"LLM Test Suite Builder"`,
		`"Error: create command requires name and description"`,
		`"Error: run command requires suite ID"`,
		`"Error: export command requires suite ID"`,
		`"Error: import command requires file path"`,
		`"Available Test Suites:"`,
		`"Execution Results:"`,
		`"Individual Test Results:"`,
		`"Creating template test suites..."`,
		`"Use 'testsuite list' to see available suites"`,
	}
	for _, lit := range forbidden {
		if strings.Contains(src, lit) {
			t.Fatalf("main.go re-hardcoded migrated literal %s — round-384 migration reverted", lit)
		}
	}
}
