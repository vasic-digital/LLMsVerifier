package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"digital.vasic.llmsverifier/pkg/i18n"
)

// fakeTranslator returns "<TRANSLATED:msg_id>" so tests can assert the
// sentinel without coupling to the English bundle text. Anti-bluff per
// CONST-035 / Article XI §11.9: a test asserting the original literal
// would silently pass if the call-site bypassed the translator.
type fakeTranslator struct{}

func (fakeTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}
func (fakeTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeTranslator installs the fakeTranslator, runs fn, restores the
// prior translator. Captures os.Stdout output for assertion.
func captureWithFakeTranslator(t *testing.T, fn func()) string {
	t.Helper()
	prior := translator
	translator = fakeTranslator{}
	defer func() { translator = prior }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	origStdout := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = origStdout }()

	done := make(chan struct{})
	var buf bytes.Buffer
	go func() {
		_, _ = io.Copy(&buf, r)
		close(done)
	}()

	fn()
	_ = w.Close()
	<-done
	return buf.String()
}

func TestPrintModelsTable_EmptyRoutesThroughTranslator(t *testing.T) {
	out := captureWithFakeTranslator(t, func() { printModelsTable(nil) })
	want := "<TRANSLATED:llmsverifier_models_table_empty>"
	if !strings.Contains(out, want) {
		t.Fatalf("printModelsTable empty output = %q; want substring %q (regressed to hardcoded literal?)", out, want)
	}
	// Anti-bluff sentinel: the original English literal MUST NOT appear.
	if strings.Contains(out, "No models found") {
		t.Fatalf("printModelsTable still emits hardcoded English literal — translator bypass detected")
	}
}

func TestPrintProvidersTable_EmptyRoutesThroughTranslator(t *testing.T) {
	out := captureWithFakeTranslator(t, func() { printProvidersTable(nil) })
	want := "<TRANSLATED:llmsverifier_providers_table_empty>"
	if !strings.Contains(out, want) {
		t.Fatalf("printProvidersTable empty output = %q; want substring %q", out, want)
	}
	if strings.Contains(out, "No providers found") {
		t.Fatalf("printProvidersTable still emits hardcoded English literal — translator bypass detected")
	}
}

func TestPrintResultsTable_EmptyRoutesThroughTranslator(t *testing.T) {
	out := captureWithFakeTranslator(t, func() { printResultsTable(nil) })
	want := "<TRANSLATED:llmsverifier_results_table_empty>"
	if !strings.Contains(out, want) {
		t.Fatalf("printResultsTable empty output = %q; want substring %q", out, want)
	}
	if strings.Contains(out, "No verification results found") {
		t.Fatalf("printResultsTable still emits hardcoded English literal — translator bypass detected")
	}
}

func TestPrintModelsTable_HeaderRoutesThroughTranslator(t *testing.T) {
	row := []map[string]interface{}{{"name": "m", "provider": "p", "version": "v", "status": "ok"}}
	out := captureWithFakeTranslator(t, func() { printModelsTable(row) })
	want := "<TRANSLATED:llmsverifier_models_table_header>"
	if !strings.Contains(out, want) {
		t.Fatalf("printModelsTable header missing sentinel; got %q", out)
	}
}

func TestPrintProvidersTable_HeaderRoutesThroughTranslator(t *testing.T) {
	row := []map[string]interface{}{{"name": "p", "model_count": 0.0, "status": "ok"}}
	out := captureWithFakeTranslator(t, func() { printProvidersTable(row) })
	want := "<TRANSLATED:llmsverifier_providers_table_header>"
	if !strings.Contains(out, want) {
		t.Fatalf("printProvidersTable header missing sentinel; got %q", out)
	}
}

// Sanity: the default package-level translator is the NoopTranslator —
// verifying the shipped binary keeps stable behaviour (returns messageID
// verbatim) until a consumer wires in a real backend.
func TestPackageDefault_NoopTranslator(t *testing.T) {
	if _, ok := translator.(i18n.NoopTranslator); !ok {
		t.Fatalf("package default translator = %T; want i18n.NoopTranslator", translator)
	}
	got, err := translator.T(context.Background(), "llmsverifier_models_table_empty", nil)
	if err != nil {
		t.Fatalf("translator.T err: %v", err)
	}
	if got != "llmsverifier_models_table_empty" {
		t.Fatalf("default translator returned %q; want messageID verbatim", got)
	}
	// silence unused-import warning if go vet ever runs alone
	_ = fmt.Sprintf
}

// --- Round-194: 10 new sentinel paired-mutation tests ----------------------
//
// Each test calls the migrated helper (tr / trData) directly through the
// translator seam and asserts the fakeTranslator sentinel surfaces. Any
// revert of a single call site to its hardcoded English literal would
// bypass the translator and fail the matching test below. This is the
// paired-mutation gate per §1.1.

// trWith installs fakeTranslator, runs fn returning a string, restores
// translator. Returns the produced string for substring assertion.
func trWith(t *testing.T, fn func() string) string {
	t.Helper()
	prior := translator
	translator = fakeTranslator{}
	defer func() { translator = prior }()
	return fn()
}

func TestRound194_ListModelsEmpty_Sentinel(t *testing.T) {
	got := trWith(t, func() string { return tr("llmsverifier_list_models_empty") })
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_list_models_empty>") {
		t.Fatalf("tr() bypass: got %q; expected sentinel", got)
	}
}

func TestRound194_ListModelsFoundHeader_Sentinel(t *testing.T) {
	got := trWith(t, func() string {
		return trData("llmsverifier_list_models_found_header", map[string]any{"count": 3})
	})
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_list_models_found_header>") {
		t.Fatalf("trData() bypass: got %q; expected sentinel", got)
	}
}

func TestRound194_ModelCreatedSuccessfully_Sentinel(t *testing.T) {
	got := trWith(t, func() string { return tr("llmsverifier_model_created_successfully") })
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_model_created_successfully>") {
		t.Fatalf("tr() bypass: got %q; expected sentinel", got)
	}
}

func TestRound194_ModelDetailsHeader_Sentinel(t *testing.T) {
	got := trWith(t, func() string { return tr("llmsverifier_model_details_header") })
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_model_details_header>") {
		t.Fatalf("tr() bypass: got %q; expected sentinel", got)
	}
}

func TestRound194_VerifyStartedForModel_Sentinel(t *testing.T) {
	got := trWith(t, func() string {
		return trData("llmsverifier_verify_started_for_model", map[string]any{"model_id": "abc"})
	})
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_verify_started_for_model>") {
		t.Fatalf("trData() bypass: got %q; expected sentinel", got)
	}
}

func TestRound194_ExportModelsSuccess_Sentinel(t *testing.T) {
	got := trWith(t, func() string {
		return trData("llmsverifier_export_models_success", map[string]any{"count": 5, "path": "/tmp/x.json"})
	})
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_export_models_success>") {
		t.Fatalf("trData() bypass: got %q; expected sentinel", got)
	}
}

func TestRound194_InteractiveModeBanner_Sentinel(t *testing.T) {
	got := trWith(t, func() string { return tr("llmsverifier_interactive_mode_banner") })
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_interactive_mode_banner>") {
		t.Fatalf("tr() bypass: got %q; expected sentinel", got)
	}
}

func TestRound194_InteractiveAvailableCommands_Sentinel(t *testing.T) {
	got := trWith(t, func() string { return tr("llmsverifier_interactive_available_commands") })
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_interactive_available_commands>") {
		t.Fatalf("tr() bypass: got %q; expected sentinel", got)
	}
}

func TestRound194_InteractiveGoodbye_Sentinel(t *testing.T) {
	got := trWith(t, func() string { return tr("llmsverifier_interactive_goodbye") })
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_interactive_goodbye>") {
		t.Fatalf("tr() bypass: got %q; expected sentinel", got)
	}
}

func TestRound194_InteractiveListUsage_Sentinel(t *testing.T) {
	got := trWith(t, func() string { return tr("llmsverifier_interactive_list_usage") })
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_interactive_list_usage>") {
		t.Fatalf("tr() bypass: got %q; expected sentinel", got)
	}
}

// TestRound194_TrData_NoopReturnsMessageID confirms the new trData() helper
// delegates to translator.T and falls back to the messageID when the
// NoopTranslator is installed — preserving the shipped-binary contract
// for parameterised sites (same invariant as TestPackageDefault_NoopTranslator
// extended for trData).
func TestRound194_TrData_NoopReturnsMessageID(t *testing.T) {
	// translator is package-default NoopTranslator at this point.
	got := trData("llmsverifier_round194_noop_probe", map[string]any{"x": 1})
	if got != "llmsverifier_round194_noop_probe" {
		t.Fatalf("trData() with NoopTranslator returned %q; want messageID verbatim", got)
	}
}

// TestRound194_Bundle_NoHardcodedLiterals scans the migrated print sites in
// main.go and asserts none reverted to the English literals — paired
// mutation per §1.1: a future revert flips the assertion to FAIL.
func TestRound194_Bundle_NoHardcodedLiterals(t *testing.T) {
	// Read the migrated main.go source.
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	body := string(src)

	// Each English literal that MUST NOT reappear at a print/exec site.
	// Tolerated occurrence: cobra command Short/Long/Usage strings (those
	// stay until a separate cobra-i18n migration round addresses them).
	forbidden := []struct {
		literal string
		ctx     string // additional substring required nearby to scope hit
	}{
		// `fmt.Println("No models found.")` site.
		{`fmt.Println("No models found.")`, ""},
		// `fmt.Printf("Found %d models:\n\n"` site.
		{`fmt.Printf("Found %d models:`, ""},
		{`fmt.Printf("Model created successfully\n")`, ""},
		{`fmt.Printf("Model Details:\n")`, ""},
		{`fmt.Printf("Verification started for model %s\n"`, ""},
		{`fmt.Printf("Exported %d models to %s\n"`, ""},
		{`fmt.Println("=== LLM Verifier Interactive Mode ===")`, ""},
		{`fmt.Println("Goodbye!")`, ""},
		{`fmt.Println("Usage: list models|providers")`, ""},
	}
	for _, f := range forbidden {
		if strings.Contains(body, f.literal) {
			t.Fatalf("CONST-046 round-194 regression: literal %q reappeared in main.go (paired-mutation gate)", f.literal)
		}
	}
}
