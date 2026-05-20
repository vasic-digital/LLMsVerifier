package analytics

import (
	"context"
	"strings"
	"testing"
)

// fakeAnalyticsTranslator returns "<TRANSLATED:msg_id>" so tests can assert
// the i18n-routed sentinel without coupling to the English bundle text.
// Anti-bluff per CONST-035 / Article XI §11.9: a test asserting the original
// literal would silently pass if the call-site bypassed the translator.
type fakeAnalyticsTranslator struct{}

func (fakeAnalyticsTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}
func (fakeAnalyticsTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeAnalyticsTranslator installs the fakeAnalyticsTranslator, runs fn,
// then restores the prior translator.
func withFakeAnalyticsTranslator(t *testing.T, fn func()) {
	t.Helper()
	prior := translator
	translator = fakeAnalyticsTranslator{}
	defer func() { translator = prior }()
	fn()
}

// TestExecutiveSummary_LabelsRouted proves the executive-summary builder
// routes every metric name, trend insight, alert title/description and
// recommendation through the i18n seam. With the fake translator installed,
// every migrated string must carry the "<TRANSLATED:analytics.*>" prefix —
// if a branch still held an English literal, the assertion fails.
func TestExecutiveSummary_LabelsRouted(t *testing.T) {
	withFakeAnalyticsTranslator(t, func() {
		ar := &AdvancedReporting{}
		summary, err := ar.GenerateExecutiveSummary(context.Background(), QueryTimeRange{})
		if err != nil {
			t.Fatalf("GenerateExecutiveSummary returned error: %v", err)
		}

		for key, m := range summary.KeyMetrics {
			if !strings.HasPrefix(m.Name, "<TRANSLATED:analytics.metric.") {
				t.Errorf("metric %q name not i18n-routed: %q", key, m.Name)
			}
		}
		if len(summary.Trends) == 0 {
			t.Fatal("expected trend analyses, got none")
		}
		for _, tr := range summary.Trends {
			if !strings.HasPrefix(tr.Insight, "<TRANSLATED:analytics.insight.") {
				t.Errorf("trend %q insight not i18n-routed: %q", tr.Metric, tr.Insight)
			}
		}
		if len(summary.Alerts) == 0 {
			t.Fatal("expected system alerts, got none")
		}
		for _, al := range summary.Alerts {
			if !strings.HasPrefix(al.Title, "<TRANSLATED:analytics.alert.") {
				t.Errorf("alert title not i18n-routed: %q", al.Title)
			}
			if !strings.HasPrefix(al.Description, "<TRANSLATED:analytics.alert.") {
				t.Errorf("alert description not i18n-routed: %q", al.Description)
			}
		}
		if len(summary.Recommendations) == 0 {
			t.Fatal("expected recommendations, got none")
		}
		for _, rec := range summary.Recommendations {
			if !strings.HasPrefix(rec, "<TRANSLATED:analytics.recommendation.") {
				t.Errorf("recommendation not i18n-routed: %q", rec)
			}
		}
	})
}

// TestDetailedReport_SectionTitlesRouted proves every detailed-report section
// title and chart label routes through the i18n seam.
func TestDetailedReport_SectionTitlesRouted(t *testing.T) {
	withFakeAnalyticsTranslator(t, func() {
		ar := &AdvancedReporting{}
		report, err := ar.GenerateDetailedReport(context.Background(), QueryTimeRange{})
		if err != nil {
			t.Fatalf("GenerateDetailedReport returned error: %v", err)
		}
		if len(report.Sections) == 0 {
			t.Fatal("expected report sections, got none")
		}
		for _, sec := range report.Sections {
			if !strings.HasPrefix(sec.Title, "<TRANSLATED:analytics.section.") {
				t.Errorf("section title not i18n-routed: %q", sec.Title)
			}
			for _, ch := range sec.Charts {
				if !strings.HasPrefix(ch.Title, "<TRANSLATED:analytics.chart.") {
					t.Errorf("chart title not i18n-routed: %q", ch.Title)
				}
			}
		}
	})
}

// TestAnalyticsMutationGuard is the paired-mutation test per §1.1. With the
// production-default NoopTranslator, the verbatim message ID is returned —
// NOT an English literal. A regression that re-hardcoded "Executive Summary"
// would make the section title differ from the message ID, failing this test.
func TestAnalyticsMutationGuard(t *testing.T) {
	if got := tr("analytics.section.executive_summary"); got != "analytics.section.executive_summary" {
		t.Fatalf("NoopTranslator must return the bare id; got %q", got)
	}
	if got := trData("analytics.recommendation.metric_critical", map[string]any{"metric": "x"}); got != "analytics.recommendation.metric_critical" {
		t.Fatalf("NoopTranslator (trData) must return the bare id; got %q", got)
	}

	ar := &AdvancedReporting{}
	report, err := ar.GenerateDetailedReport(context.Background(), QueryTimeRange{})
	if err != nil {
		t.Fatalf("GenerateDetailedReport: %v", err)
	}
	for _, sec := range report.Sections {
		if strings.Contains(sec.Title, "Executive Summary") ||
			strings.Contains(sec.Title, "Performance Analysis") ||
			strings.Contains(sec.Title, "Strategic Recommendations") {
			t.Fatalf("section title regressed to a hardcoded English literal: %q", sec.Title)
		}
		// Under NoopTranslator the title is the bare message ID.
		if !strings.HasPrefix(sec.Title, "analytics.section.") {
			t.Fatalf("section title not routed through the i18n seam: %q", sec.Title)
		}
	}
}

// TestUsagePatternRecommendations_Routed (round-439, CONST-046) proves the
// UsagePatternAnalyzer recommendation list routes every string through the
// i18n seam. With the fake translator installed, every entry must carry the
// "<TRANSLATED:analytics.usage_rec.*>" sentinel — a regression that re-pinned
// an English literal would fail the prefix assertion.
func TestUsagePatternRecommendations_Routed(t *testing.T) {
	withFakeAnalyticsTranslator(t, func() {
		upa := &UsagePatternAnalyzer{}
		recs := upa.generateUsageRecommendations()
		if len(recs) != 7 {
			t.Fatalf("expected 7 usage recommendations, got %d", len(recs))
		}
		for _, r := range recs {
			if !strings.HasPrefix(r, "<TRANSLATED:analytics.usage_rec.") {
				t.Errorf("usage recommendation not i18n-routed: %q", r)
			}
		}
	})
}

// TestCostRecommendations_Routed (round-439, CONST-046) proves the
// CostOptimizationAnalyzer recommendation list routes every string through
// the i18n seam.
func TestCostRecommendations_Routed(t *testing.T) {
	withFakeAnalyticsTranslator(t, func() {
		coa := &CostOptimizationAnalyzer{}
		recs := coa.generateCostRecommendations(SpendingBreakdown{})
		if len(recs) != 7 {
			t.Fatalf("expected 7 cost recommendations, got %d", len(recs))
		}
		for _, r := range recs {
			if !strings.HasPrefix(r, "<TRANSLATED:analytics.cost_rec.") {
				t.Errorf("cost recommendation not i18n-routed: %q", r)
			}
		}
	})
}

// TestRound439MutationGuard is the paired-mutation test (§1.1) for the
// round-439 migration. Under the production-default NoopTranslator every
// migrated id returns verbatim — a regression that re-hardcoded an English
// literal would make the returned string differ from the id, failing here.
func TestRound439MutationGuard(t *testing.T) {
	ids := []string{
		"analytics.usage_rec.rate_limiting",
		"analytics.usage_rec.model_selection",
		"analytics.usage_rec.caching",
		"analytics.usage_rec.smaller_models_peak",
		"analytics.usage_rec.batch_off_peak",
		"analytics.usage_rec.error_retry",
		"analytics.usage_rec.load_balancing",
		"analytics.cost_rec.tier_models",
		"analytics.cost_rec.response_caching",
		"analytics.cost_rec.open_source",
		"analytics.cost_rec.prompt_engineering",
		"analytics.cost_rec.batch_discounts",
		"analytics.cost_rec.draft_refine",
		"analytics.cost_rec.reserved_capacity",
		"analytics.insight.gpt4_routing",
		"analytics.insight.codellama_underused",
		"analytics.insight.task_complexity_logic",
		"analytics.insight.token_usage_monitor",
	}
	for _, id := range ids {
		if got := tr(id); got != id {
			t.Fatalf("NoopTranslator must return the bare id %q; got %q", id, got)
		}
	}

	upa := &UsagePatternAnalyzer{}
	for _, r := range upa.generateUsageRecommendations() {
		if strings.Contains(r, "Consider") || strings.Contains(r, "Implement") ||
			strings.Contains(r, "Optimize") || strings.Contains(r, "Schedule") {
			t.Fatalf("usage recommendation regressed to a hardcoded English literal: %q", r)
		}
		if !strings.HasPrefix(r, "analytics.usage_rec.") {
			t.Fatalf("usage recommendation not routed through the i18n seam: %q", r)
		}
	}
	coa := &CostOptimizationAnalyzer{}
	for _, r := range coa.generateCostRecommendations(SpendingBreakdown{}) {
		if strings.Contains(r, "GPT") || strings.Contains(r, "Implement") ||
			strings.Contains(r, "Consider") || strings.Contains(r, "Optimize") {
			t.Fatalf("cost recommendation regressed to a hardcoded English literal: %q", r)
		}
		if !strings.HasPrefix(r, "analytics.cost_rec.") {
			t.Fatalf("cost recommendation not routed through the i18n seam: %q", r)
		}
	}
}
