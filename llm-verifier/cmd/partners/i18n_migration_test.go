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

// partnersMsgIDs is the full set of message IDs introduced by the round-411
// cmd/partners migration. Used by both the routing test and the bundle
// no-revert test so they stay in lockstep.
var partnersMsgIDs = []string{
	"llmsverifier_partners_unknown_command",
	"llmsverifier_partners_usage_title",
	"llmsverifier_partners_usage_heading",
	"llmsverifier_partners_usage_list",
	"llmsverifier_partners_usage_add",
	"llmsverifier_partners_usage_remove",
	"llmsverifier_partners_usage_sync",
	"llmsverifier_partners_usage_status",
	"llmsverifier_partners_types_heading",
	"llmsverifier_partners_type_opencode",
	"llmsverifier_partners_type_claude_code",
	"llmsverifier_partners_type_cursor",
	"llmsverifier_partners_type_vscode",
	"llmsverifier_partners_type_jetbrains",
	"llmsverifier_partners_type_github",
	"llmsverifier_partners_examples_heading",
	"llmsverifier_partners_example_add",
	"llmsverifier_partners_example_sync",
	"llmsverifier_partners_list_heading",
	"llmsverifier_partners_last_sync_indented",
	"llmsverifier_partners_err_add_args",
	"llmsverifier_partners_added_integration",
	"llmsverifier_partners_err_remove_args",
	"llmsverifier_partners_removed_integration",
	"llmsverifier_partners_err_sync_args",
	"llmsverifier_partners_syncing_integration",
	"llmsverifier_partners_synced_success",
	"llmsverifier_partners_last_sync",
	"llmsverifier_partners_err_status_args",
	"llmsverifier_partners_status_heading",
	"llmsverifier_partners_field_id",
	"llmsverifier_partners_field_type",
	"llmsverifier_partners_field_status",
	"llmsverifier_partners_field_version",
	"llmsverifier_partners_field_last_sync",
	"llmsverifier_partners_field_error",
	"llmsverifier_partners_field_capabilities",
}

// TestPartnersTrRoutesThroughTranslator proves tr() actually delegates to
// the package translator. If tr() regressed to returning a hardcoded
// literal, the sentinel would not appear. Paired-mutation: swap the body
// of tr() to `return id` minus the translator call and this fails.
func TestPartnersTrRoutesThroughTranslator(t *testing.T) {
	prior := translator
	translator = fakeTranslator{}
	defer func() { translator = prior }()

	for _, id := range partnersMsgIDs {
		got := tr(id)
		want := "<TRANSLATED:" + id + ">"
		if got != want {
			t.Fatalf("tr(%q) = %q; want %q — translator seam bypassed", id, got, want)
		}
	}
}

// TestPartnersTrDataRoutesThroughTranslator proves trData() delegates to
// the translator for parameterised call sites.
func TestPartnersTrDataRoutesThroughTranslator(t *testing.T) {
	prior := translator
	translator = fakeTranslator{}
	defer func() { translator = prior }()

	id := "llmsverifier_partners_added_integration"
	got := trData(id, map[string]any{"name": "X", "id": "y-1"})
	want := "<TRANSLATED:" + id + ">"
	if got != want {
		t.Fatalf("trData(%q) = %q; want %q — translator seam bypassed", id, got, want)
	}
}

// TestPartnersNoopReturnsIDVerbatim asserts the default NoopTranslator
// returns the message ID unchanged — required so call sites stay testable.
func TestPartnersNoopReturnsIDVerbatim(t *testing.T) {
	// translator is NoopTranslator{} by default.
	id := "llmsverifier_partners_usage_title"
	if got := tr(id); got != id {
		t.Fatalf("tr(%q) with NoopTranslator = %q; want verbatim ID", id, got)
	}
}

// TestPartnersBundleNoRevert scans the active English bundle and asserts
// every round-411 message ID is still defined. A revert that drops a
// bundle entry (re-hardcoding the literal in source) is caught here.
// Paired-mutation: delete any llmsverifier_partners_* entry and this fails.
func TestPartnersBundleNoRevert(t *testing.T) {
	bundlePath := filepath.Join("..", "..", "pkg", "i18n", "bundles", "active.en.yaml")
	data, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("read bundle %s: %v", bundlePath, err)
	}
	bundle := string(data)

	for _, id := range partnersMsgIDs {
		if !strings.Contains(bundle, id+":") {
			t.Fatalf("bundle %s missing message ID %q — round-411 i18n migration reverted", bundlePath, id)
		}
	}
}

// TestPartnersSourceNoHardcodedRevert scans main.go and asserts the
// migrated English literals are no longer present as source-code string
// literals. A re-hardcode of any migrated string is caught here.
func TestPartnersSourceNoHardcodedRevert(t *testing.T) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go: %v", err)
	}
	src := string(data)

	forbidden := []string{
		`"LLM Verifier Partner Integrations"`,
		`"Supported integration types:"`,
		`"Partner Integrations:"`,
		`"Error: add command requires type and name"`,
		`"Error: remove command requires integration ID"`,
		`"Error: sync command requires integration name or ID"`,
		`"Error: status command requires integration ID"`,
		`"Added integration: %s (%s)\n"`,
		`"Syncing integration: %s...\n"`,
		`"Successfully synced: %s\n"`,
		`"Integration Status: %s\n"`,
	}
	for _, lit := range forbidden {
		if strings.Contains(src, lit) {
			t.Fatalf("main.go re-hardcoded migrated literal %s — round-411 migration reverted", lit)
		}
	}
}
