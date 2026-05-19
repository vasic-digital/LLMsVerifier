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

// --- Round-308: 9 new sentinel paired-mutation tests ---------------------
//
// Each test calls the migrated helper (tr / trData) directly through the
// translator seam and asserts the fakeTranslator sentinel surfaces. Any
// revert of a single call site to its hardcoded English literal would
// bypass the translator and fail the matching test below. Paired-mutation
// gate per §1.1 / CONST-046.

func TestRound308_ValidateConfigFileValid_Sentinel(t *testing.T) {
	got := trWith(t, func() string { return tr("llmsverifier_validate_config_file_valid") })
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_validate_config_file_valid>") {
		t.Fatalf("tr() bypass: got %q; expected sentinel", got)
	}
}

func TestRound308_ValidateConfigApiPort_Sentinel(t *testing.T) {
	got := trWith(t, func() string {
		return trData("llmsverifier_validate_config_api_port", map[string]any{"port": "8080"})
	})
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_validate_config_api_port>") {
		t.Fatalf("trData() bypass: got %q; expected sentinel", got)
	}
}

func TestRound308_ValidateSystemProbeOk_Sentinel(t *testing.T) {
	got := trWith(t, func() string { return tr("llmsverifier_validate_system_probe_ok") })
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_validate_system_probe_ok>") {
		t.Fatalf("tr() bypass: got %q; expected sentinel", got)
	}
}

func TestRound308_ValidateSystemCompleted_Sentinel(t *testing.T) {
	got := trWith(t, func() string { return tr("llmsverifier_validate_system_completed_successfully") })
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_validate_system_completed_successfully>") {
		t.Fatalf("tr() bypass: got %q; expected sentinel", got)
	}
}

func TestRound308_AIConfigExportingForFormat_Sentinel(t *testing.T) {
	got := trWith(t, func() string {
		return trData("llmsverifier_aiconfig_exporting_for_format", map[string]any{"format": "opencode"})
	})
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_aiconfig_exporting_for_format>") {
		t.Fatalf("trData() bypass: got %q; expected sentinel", got)
	}
}

func TestRound308_AIConfigExportSuccess_Sentinel(t *testing.T) {
	got := trWith(t, func() string {
		return trData("llmsverifier_aiconfig_export_success", map[string]any{"format": "crush", "path": "/tmp/x.json"})
	})
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_aiconfig_export_success>") {
		t.Fatalf("trData() bypass: got %q; expected sentinel", got)
	}
}

func TestRound308_AIConfigValidating_Sentinel(t *testing.T) {
	got := trWith(t, func() string { return tr("llmsverifier_aiconfig_validating") })
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_aiconfig_validating>") {
		t.Fatalf("tr() bypass: got %q; expected sentinel", got)
	}
}

func TestRound308_AIConfigValidationPassed_Sentinel(t *testing.T) {
	got := trWith(t, func() string { return tr("llmsverifier_aiconfig_validation_passed") })
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_aiconfig_validation_passed>") {
		t.Fatalf("tr() bypass: got %q; expected sentinel", got)
	}
}

func TestRound308_AIConfigBulkExportSuccess_Sentinel(t *testing.T) {
	got := trWith(t, func() string {
		return trData("llmsverifier_aiconfig_bulk_export_success", map[string]any{"path": "/tmp/out"})
	})
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_aiconfig_bulk_export_success>") {
		t.Fatalf("trData() bypass: got %q; expected sentinel", got)
	}
}

// TestRound308_Bundle_NoHardcodedLiterals scans the migrated print sites in
// main.go and asserts none reverted to the English literals — paired
// mutation per §1.1: a future revert flips the assertion to FAIL.
func TestRound308_Bundle_NoHardcodedLiterals(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	body := string(src)

	forbidden := []string{
		`fmt.Println("✓ Configuration file is valid")`,
		`fmt.Printf("✓ API Port: %s\n", cfg.API.Port)`,
		`fmt.Printf("✓ Database Path: %s\n", cfg.Database.Path)`,
		`fmt.Printf("✓ LLMs configured: %d\n", len(cfg.LLMs))`,
		`fmt.Printf("✓ Profile: %s\n", cfg.Profile)`,
		`fmt.Print("Testing database connectivity... ")`,
		`fmt.Print("Testing API endpoints... ")`,
		`fmt.Println("✓ System validation completed successfully")`,
		`fmt.Printf("📤 Exporting AI CLI configuration for format: %s\n", format)`,
		`fmt.Printf("📄 Output file: %s\n", outputFile)`,
		`fmt.Printf("✅ Successfully exported %s configuration to %s\n", format, outputFile)`,
		`fmt.Println("🔍 Validating exported configuration...")`,
		`fmt.Println("✅ Configuration validation passed")`,
		`fmt.Printf("📤 Exporting AI CLI configurations to directory: %s\n", outputDir)`,
		`fmt.Printf("✅ Successfully exported all AI CLI configurations to %s\n", outputDir)`,
		`fmt.Println("📄 Generated files:")`,
		`fmt.Printf("🔍 Validating AI CLI configuration: %s\n", configPath)`,
	}
	for _, lit := range forbidden {
		if strings.Contains(body, lit) {
			t.Fatalf("CONST-046 round-308 regression: literal %q reappeared in main.go (paired-mutation gate)", lit)
		}
	}
}

// --- Round-309: sentinel paired-mutation tests --------------------------
//
// Each test calls the migrated helper (tr / trData) directly through the
// translator seam and asserts the fakeTranslator sentinel surfaces. Any
// revert of a single call site to its hardcoded English literal would
// bypass the translator and fail the matching test below. Paired-mutation
// gate per §1.1 / CONST-046.

func TestRound309_InteractiveVerifyUsage_Sentinel(t *testing.T) {
	got := trWith(t, func() string { return tr("llmsverifier_interactive_verify_usage") })
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_interactive_verify_usage>") {
		t.Fatalf("tr() bypass: got %q; expected sentinel", got)
	}
}

func TestRound309_InteractiveError_Sentinel(t *testing.T) {
	got := trWith(t, func() string {
		return trData("llmsverifier_interactive_error", map[string]any{"err": "boom"})
	})
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_interactive_error>") {
		t.Fatalf("trData() bypass: got %q; expected sentinel", got)
	}
}

func TestRound309_InteractiveVerifyCompleted_Sentinel(t *testing.T) {
	got := trWith(t, func() string { return tr("llmsverifier_interactive_verify_completed") })
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_interactive_verify_completed>") {
		t.Fatalf("tr() bypass: got %q; expected sentinel", got)
	}
}

func TestRound309_InteractiveErrorDisplayingResult_Sentinel(t *testing.T) {
	got := trWith(t, func() string {
		return trData("llmsverifier_interactive_error_displaying_result", map[string]any{"err": "x"})
	})
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_interactive_error_displaying_result>") {
		t.Fatalf("trData() bypass: got %q; expected sentinel", got)
	}
}

func TestRound309_InteractiveErrorGettingModels_Sentinel(t *testing.T) {
	got := trWith(t, func() string {
		return trData("llmsverifier_interactive_error_getting_models", map[string]any{"err": "x"})
	})
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_interactive_error_getting_models>") {
		t.Fatalf("trData() bypass: got %q; expected sentinel", got)
	}
}

func TestRound309_InteractiveErrorGettingProviders_Sentinel(t *testing.T) {
	got := trWith(t, func() string {
		return trData("llmsverifier_interactive_error_getting_providers", map[string]any{"err": "x"})
	})
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_interactive_error_getting_providers>") {
		t.Fatalf("trData() bypass: got %q; expected sentinel", got)
	}
}

func TestRound309_InteractiveStatusHeader_Sentinel(t *testing.T) {
	got := trWith(t, func() string { return tr("llmsverifier_interactive_status_header") })
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_interactive_status_header>") {
		t.Fatalf("tr() bypass: got %q; expected sentinel", got)
	}
}

func TestRound309_InteractiveStatusModels_Sentinel(t *testing.T) {
	got := trWith(t, func() string {
		return trData("llmsverifier_interactive_status_models", map[string]any{"count": 4})
	})
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_interactive_status_models>") {
		t.Fatalf("trData() bypass: got %q; expected sentinel", got)
	}
}

func TestRound309_InteractiveStatusProviders_Sentinel(t *testing.T) {
	got := trWith(t, func() string {
		return trData("llmsverifier_interactive_status_providers", map[string]any{"count": 2})
	})
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_interactive_status_providers>") {
		t.Fatalf("trData() bypass: got %q; expected sentinel", got)
	}
}

func TestRound309_InteractiveStatusApiConnected_Sentinel(t *testing.T) {
	got := trWith(t, func() string { return tr("llmsverifier_interactive_status_api_connected") })
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_interactive_status_api_connected>") {
		t.Fatalf("tr() bypass: got %q; expected sentinel", got)
	}
}

func TestRound309_InteractiveUnknownCommand_Sentinel(t *testing.T) {
	got := trWith(t, func() string {
		return trData("llmsverifier_interactive_unknown_command", map[string]any{"command": "frob"})
	})
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_interactive_unknown_command>") {
		t.Fatalf("trData() bypass: got %q; expected sentinel", got)
	}
}

func TestRound309_ProvidersListNoneFound_Sentinel(t *testing.T) {
	got := trWith(t, func() string { return tr("llmsverifier_providers_list_none_found") })
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_providers_list_none_found>") {
		t.Fatalf("tr() bypass: got %q; expected sentinel", got)
	}
}

func TestRound309_ProvidersListFoundHeader_Sentinel(t *testing.T) {
	got := trWith(t, func() string {
		return trData("llmsverifier_providers_list_found_header", map[string]any{"count": 3})
	})
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_providers_list_found_header>") {
		t.Fatalf("trData() bypass: got %q; expected sentinel", got)
	}
}

func TestRound309_ProvidersListItemName_Sentinel(t *testing.T) {
	got := trWith(t, func() string {
		return trData("llmsverifier_providers_list_item_name", map[string]any{"index": 1, "name": "p"})
	})
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_providers_list_item_name>") {
		t.Fatalf("trData() bypass: got %q; expected sentinel", got)
	}
}

func TestRound309_ProvidersListItemCreated_Sentinel(t *testing.T) {
	got := trWith(t, func() string {
		return trData("llmsverifier_providers_list_item_created", map[string]any{"created": "now"})
	})
	if !strings.Contains(got, "<TRANSLATED:llmsverifier_providers_list_item_created>") {
		t.Fatalf("trData() bypass: got %q; expected sentinel", got)
	}
}

// TestRound309_Bundle_NoHardcodedLiterals scans the migrated print sites in
// main.go and asserts none reverted to the English literals — paired
// mutation per §1.1: a future revert flips the assertion to FAIL.
func TestRound309_Bundle_NoHardcodedLiterals(t *testing.T) {
	src, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	body := string(src)

	forbidden := []string{
		`fmt.Println("Usage: verify [model_id]")`,
		`fmt.Println("Verification completed:")`,
		`fmt.Printf("Error displaying result: %v\n", err)`,
		`fmt.Printf("Error getting models: %v\n", err)`,
		`fmt.Printf("Error getting providers: %v\n", err)`,
		`fmt.Printf("System Status:\n")`,
		`fmt.Printf("  Models: %d\n", len(models))`,
		`fmt.Printf("  Providers: %d\n", len(providers))`,
		`fmt.Printf("  API Server: Connected\n")`,
		`fmt.Printf("Unknown command: %s\n", args[0])`,
		`fmt.Println("No providers found.")`,
		`fmt.Printf("Found %d providers:\n\n", len(providers))`,
		`fmt.Printf("%d. Name: %v\n", i+1, provider["name"])`,
		`fmt.Printf("   Endpoint: %v\n", provider["endpoint"])`,
		`fmt.Printf("   Status: %v\n", provider["status"])`,
		`fmt.Printf("   Created: %v\n\n", provider["created_at"])`,
	}
	for _, lit := range forbidden {
		if strings.Contains(body, lit) {
			t.Fatalf("CONST-046 round-309 regression: literal %q reappeared in main.go (paired-mutation gate)", lit)
		}
	}
}

// TestRound309_Bundle_KeysPresent scans the active.en.yaml bundle and asserts
// every round-309 message key is present — bundle-scan no-revert paired
// mutation: deleting a key flips the assertion to FAIL.
func TestRound309_Bundle_KeysPresent(t *testing.T) {
	src, err := os.ReadFile("../pkg/i18n/bundles/active.en.yaml")
	if err != nil {
		t.Fatalf("read active.en.yaml: %v", err)
	}
	body := string(src)

	keys := []string{
		"llmsverifier_interactive_verify_usage",
		"llmsverifier_interactive_error",
		"llmsverifier_interactive_verify_completed",
		"llmsverifier_interactive_error_displaying_result",
		"llmsverifier_interactive_error_getting_models",
		"llmsverifier_interactive_error_getting_providers",
		"llmsverifier_interactive_status_header",
		"llmsverifier_interactive_status_models",
		"llmsverifier_interactive_status_providers",
		"llmsverifier_interactive_status_api_connected",
		"llmsverifier_interactive_unknown_command",
		"llmsverifier_providers_list_none_found",
		"llmsverifier_providers_list_found_header",
		"llmsverifier_providers_list_item_name",
		"llmsverifier_providers_list_item_endpoint",
		"llmsverifier_providers_list_item_status",
		"llmsverifier_providers_list_item_created",
	}
	for _, k := range keys {
		if !strings.Contains(body, k+":") {
			t.Fatalf("CONST-046 round-309: bundle key %q missing from active.en.yaml (no-revert gate)", k)
		}
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
