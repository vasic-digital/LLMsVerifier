package llmverifier

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
// prior translator.
func withFakeTranslator(t *testing.T, fn func()) {
	t.Helper()
	prior := translator
	translator = fakeTranslator{}
	defer func() { translator = prior }()
	fn()
}

// generateReportText drives GenerateMarkdownReport against a temp directory
// and returns the rendered markdown so tests can assert routed sentinels.
func generateReportText(t *testing.T, results []VerificationResult) string {
	t.Helper()
	v := New(nil)
	dir := t.TempDir()
	if err := v.GenerateMarkdownReport(results, dir); err != nil {
		t.Fatalf("GenerateMarkdownReport: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "llm_verification_report.md"))
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	return string(data)
}

// TestReporter_HeaderRoutesThroughTranslator asserts the report header and
// summary section headers are routed through the i18n seam. Paired-mutation
// anti-bluff: if a call-site regressed to a hardcoded English literal, the
// sentinel "<TRANSLATED:report.title>" would be absent and this fails.
func TestReporter_HeaderRoutesThroughTranslator(t *testing.T) {
	withFakeTranslator(t, func() {
		out := generateReportText(t, nil)
		for _, want := range []string{
			"<TRANSLATED:report.title>",
			"<TRANSLATED:report.generated_on>",
			"<TRANSLATED:report.section.summary>",
			"<TRANSLATED:report.summary.total_models>",
			"<TRANSLATED:report.summary.average_overall_score>",
			"<TRANSLATED:report.section.top_performers_overall>",
			"<TRANSLATED:report.section.category_rankings>",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("report missing routed sentinel %q\n--- report ---\n%s", want, out)
			}
		}
	})
}

// TestReporter_ModelSectionRoutesThroughTranslator drives a successful and a
// failed model report and asserts the per-model field labels route through
// the translator.
func TestReporter_ModelSectionRoutesThroughTranslator(t *testing.T) {
	results := []VerificationResult{
		{ModelInfo: ModelInfo{ID: "test-model", Endpoint: "http://localhost"}},
		{ModelInfo: ModelInfo{ID: "broken-model", Endpoint: "http://localhost"}, Error: "boom"},
	}
	withFakeTranslator(t, func() {
		out := generateReportText(t, results)
		for _, want := range []string{
			"<TRANSLATED:report.model.label>",
			"<TRANSLATED:report.section.basic_information>",
			"<TRANSLATED:report.field.endpoint>",
			"<TRANSLATED:report.section.performance_scores>",
			"<TRANSLATED:report.field.overall_score>",
			"<TRANSLATED:report.section.supported_features>",
			"<TRANSLATED:report.field.tool_use>",
			"<TRANSLATED:report.section.code_capabilities>",
			"<TRANSLATED:report.section.language_specific_performance>",
			"<TRANSLATED:report.field.python_success_rate>",
			"<TRANSLATED:report.model.failed>",
			"<TRANSLATED:report.field.error>",
			"<TRANSLATED:report.field.attempted_at>",
			"<TRANSLATED:report.section.overall_performance>",
			"<TRANSLATED:report.section.by_value_proposition>",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("report missing routed sentinel %q", want)
			}
		}
	})
}

// TestReporter_NoopTranslatorReturnsMessageID confirms the default
// NoopTranslator emits the messageID verbatim (the seam contract relied on
// by every other consumer that does not install a real bundle).
func TestReporter_NoopTranslatorReturnsMessageID(t *testing.T) {
	if got := tr("report.title"); got != "report.title" {
		t.Errorf("tr() with NoopTranslator = %q, want verbatim id", got)
	}
	got := trData("report.value.requests_per_sec", map[string]any{"value": 1.0})
	if got != "report.value.requests_per_sec" {
		t.Errorf("trData() with NoopTranslator = %q, want verbatim id", got)
	}
}
