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
