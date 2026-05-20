package main

import (
	"context"
	"strings"
	"testing"

	"digital.vasic.llmsverifier/pkg/i18n"
)

// fakeCodeVerificationTranslator returns "<TRANSLATED:msg_id>" for every
// message ID, proving that the help-text and summary call sites route their
// user-facing text through the i18n seam rather than emitting hardcoded
// English literals (CONST-046 anti-bluff invariant per Article XI §11.9).
type fakeCodeVerificationTranslator struct{}

func (fakeCodeVerificationTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

func (fakeCodeVerificationTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeCodeVerificationTranslator swaps the package translator for the
// duration of a test and restores the prior backend afterwards.
func withFakeCodeVerificationTranslator(t *testing.T) {
	t.Helper()
	prev := translator
	translator = fakeCodeVerificationTranslator{}
	t.Cleanup(func() { translator = prev })
}

// TestTr_RoutesThroughTranslator is the positive case: with the fake backend
// installed, tr() must emit the sentinel, not the message ID verbatim.
func TestTr_RoutesThroughTranslator(t *testing.T) {
	withFakeCodeVerificationTranslator(t)
	got := tr("code_verification.help.title")
	want := "<TRANSLATED:code_verification.help.title>"
	if got != want {
		t.Fatalf("tr() did not route through translator: got %q want %q", got, want)
	}
}

// TestTr_NoopReturnsMessageID is the default-backend case: NoopTranslator
// returns the message ID verbatim, which is the graceful fallback.
func TestTr_NoopReturnsMessageID(t *testing.T) {
	prev := translator
	translator = i18n.NoopTranslator{}
	t.Cleanup(func() { translator = prev })
	if got := tr("code_verification.summary.heading"); got != "code_verification.summary.heading" {
		t.Fatalf("NoopTranslator should return id verbatim, got %q", got)
	}
}

// TestHelpAndSummaryMessageIDs_AllRouteThroughTranslator is the paired-mutation
// gate: every message ID used by printHelp and printSummary MUST resolve via
// the translator. With the fake backend installed, tr() must return the
// "<TRANSLATED:...>" sentinel for each ID — a regression that re-introduces a
// hardcoded English literal would not produce the sentinel.
func TestHelpAndSummaryMessageIDs_AllRouteThroughTranslator(t *testing.T) {
	withFakeCodeVerificationTranslator(t)

	ids := []string{
		"code_verification.help.title",
		"code_verification.help.summary",
		"code_verification.help.usage_heading",
		"code_verification.help.usage_line",
		"code_verification.help.options_heading",
		"code_verification.help.option_config",
		"code_verification.help.option_output",
		"code_verification.help.option_providers",
		"code_verification.help.option_models",
		"code_verification.help.option_concurrency",
		"code_verification.help.option_timeout",
		"code_verification.help.option_format",
		"code_verification.help.option_db",
		"code_verification.help.option_help",
		"code_verification.help.examples_heading",
		"code_verification.help.example_all_comment",
		"code_verification.help.example_all_command",
		"code_verification.help.example_providers_comment",
		"code_verification.help.example_providers_command",
		"code_verification.help.example_models_comment",
		"code_verification.help.example_models_command",
		"code_verification.help.example_output_comment",
		"code_verification.help.example_output_command",
		"code_verification.summary.heading",
		"code_verification.summary.total_models",
		"code_verification.summary.verified_models",
		"code_verification.summary.failed_models",
		"code_verification.summary.error_models",
		"code_verification.summary.average_score",
		"code_verification.summary.success_rate",
	}

	for _, id := range ids {
		got := tr(id)
		if !strings.HasPrefix(got, "<TRANSLATED:code_verification.") {
			t.Errorf("message ID %q is not i18n-routed: got %q", id, got)
		}
		if got != "<TRANSLATED:"+id+">" {
			t.Errorf("message ID %q resolved to unexpected sentinel %q", id, got)
		}
	}
}
