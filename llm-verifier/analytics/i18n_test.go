package analytics

import (
	"context"
	"strings"
	"testing"

	"digital.vasic.llmsverifier/pkg/i18n"
)

// fakeAnalyticsTranslator returns "<TRANSLATED:msg_id>" for every message ID,
// proving that recommendation / trend / cost-optimization call sites route
// their user-facing text through the i18n seam rather than emitting hardcoded
// English literals (CONST-046 anti-bluff invariant per Article XI §11.9).
type fakeAnalyticsTranslator struct{}

func (fakeAnalyticsTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

func (fakeAnalyticsTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeAnalyticsTranslator swaps the package translator for the duration of
// a test and restores the prior backend afterwards.
func withFakeAnalyticsTranslator(t *testing.T) {
	t.Helper()
	prev := translator
	translator = fakeAnalyticsTranslator{}
	t.Cleanup(func() { translator = prev })
}

// TestTr_RoutesThroughTranslator is the positive case: with the fake backend
// installed, tr() must emit the sentinel, not the message ID verbatim.
func TestTr_RoutesThroughTranslator(t *testing.T) {
	withFakeAnalyticsTranslator(t)
	got := tr("analytics.recommendation.performance.groq")
	want := "<TRANSLATED:analytics.recommendation.performance.groq>"
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
	if got := tr("analytics.insight.switch_groq"); got != "analytics.insight.switch_groq" {
		t.Fatalf("NoopTranslator should return id verbatim, got %q", got)
	}
}

// TestScoreProvider_RecommendationsAreI18nRouted is the paired-mutation gate
// for predictive.go: every recommendation reasoning string MUST resolve via
// the translator. With the fake backend, no reasoning entry may contain a
// raw English literal — every entry must be a "<TRANSLATED:...>" sentinel.
func TestScoreProvider_RecommendationsAreI18nRouted(t *testing.T) {
	withFakeAnalyticsTranslator(t)
	re := NewRecommendationEngine()

	cases := []*RecommendationRequest{
		{Priority: "performance", MaxTokens: 1000},
		{Priority: "cost", MaxTokens: 1000},
		{Priority: "reliability", MaxTokens: 1000},
		{RequiresSafety: true, MaxTokens: 1000},
		{MaxTokens: 200000},
		{RequiresMultiModal: true, MaxTokens: 1000},
		{MaxTokens: 1000, UserHistory: map[string]float64{"openai": 0.9, "groq": 0.8}},
	}

	for _, req := range cases {
		resp, err := re.GetProviderRecommendation(context.Background(), req)
		if err != nil {
			t.Fatalf("GetProviderRecommendation(%+v): %v", req, err)
		}
		for _, rec := range resp.Recommendations {
			for _, reason := range rec.Reasoning {
				if !strings.HasPrefix(reason, "<TRANSLATED:analytics.") {
					t.Errorf("priority=%q provider=%q reasoning %q is not i18n-routed",
						req.Priority, rec.Provider, reason)
				}
			}
			for _, tradeoff := range rec.Tradeoffs {
				if !strings.HasPrefix(tradeoff, "<TRANSLATED:analytics.") {
					t.Errorf("priority=%q provider=%q tradeoff %q is not i18n-routed",
						req.Priority, rec.Provider, tradeoff)
				}
			}
		}
	}
}

// TestAnalyzeTrends_DescriptionsAndInsightsAreI18nRouted is the paired-mutation
// gate for AnalyzeTrends: every Trend.Description and every Insights entry MUST
// resolve via the translator.
func TestAnalyzeTrends_DescriptionsAndInsightsAreI18nRouted(t *testing.T) {
	withFakeAnalyticsTranslator(t)
	pa := NewPerformanceAnalyzer()

	analysis, err := pa.AnalyzeTrends("openai", "gpt-4", 30)
	if err != nil {
		t.Fatalf("AnalyzeTrends: %v", err)
	}
	for name, trend := range analysis.Trends {
		if !strings.HasPrefix(trend.Description, "<TRANSLATED:analytics.trend.") {
			t.Errorf("trend %q description %q is not i18n-routed", name, trend.Description)
		}
	}
	for _, insight := range analysis.Insights {
		if !strings.HasPrefix(insight, "<TRANSLATED:analytics.insight.") {
			t.Errorf("insight %q is not i18n-routed", insight)
		}
	}
}

// TestGenerateCostOptimization_DescriptionsAreI18nRouted is the paired-mutation
// gate for GenerateCostOptimization: every CostRecommendation.Description MUST
// resolve via the translator.
func TestGenerateCostOptimization_DescriptionsAreI18nRouted(t *testing.T) {
	withFakeAnalyticsTranslator(t)
	pa := NewPerformanceAnalyzer()

	opt, err := pa.GenerateCostOptimization("user-123")
	if err != nil {
		t.Fatalf("GenerateCostOptimization: %v", err)
	}
	if len(opt.Recommendations) == 0 {
		t.Fatal("expected at least one cost recommendation")
	}
	for _, rec := range opt.Recommendations {
		if !strings.HasPrefix(rec.Description, "<TRANSLATED:analytics.cost_rec.") {
			t.Errorf("cost recommendation %q description %q is not i18n-routed",
				rec.Type, rec.Description)
		}
	}
}
