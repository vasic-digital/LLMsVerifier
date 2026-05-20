package enhanced

import (
	"context"
	"strings"
	"testing"
	"time"

	"digital.vasic.llmsverifier/database"
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

// TestIssueTemplate_LocalizedRoutesThroughTranslator drives every common
// issue template through Localized() and asserts that Name, Title,
// Description, Symptoms, and Workarounds route through the i18n seam.
// Paired-mutation anti-bluff: if Localized() regressed to returning the
// stored message ID verbatim (or a hardcoded English literal), the
// "<TRANSLATED:...>" sentinel would be absent and this fails.
func TestIssueTemplate_LocalizedRoutesThroughTranslator(t *testing.T) {
	withFakeTranslator(t, func() {
		for _, raw := range IssueTemplates {
			loc := raw.Localized()
			if !strings.HasPrefix(loc.Name, "<TRANSLATED:") {
				t.Errorf("template %s Name not routed: %q", raw.ID, loc.Name)
			}
			if !strings.HasPrefix(loc.Title, "<TRANSLATED:") {
				t.Errorf("template %s Title not routed: %q", raw.ID, loc.Title)
			}
			if !strings.HasPrefix(loc.Description, "<TRANSLATED:") {
				t.Errorf("template %s Description not routed: %q", raw.ID, loc.Description)
			}
			if len(loc.Symptoms) != len(raw.Symptoms) {
				t.Errorf("template %s Symptoms length changed: %d != %d", raw.ID, len(loc.Symptoms), len(raw.Symptoms))
			}
			for i, s := range loc.Symptoms {
				if !strings.HasPrefix(s, "<TRANSLATED:") {
					t.Errorf("template %s Symptoms[%d] not routed: %q", raw.ID, i, s)
				}
			}
			for i, w := range loc.Workarounds {
				if !strings.HasPrefix(w, "<TRANSLATED:") {
					t.Errorf("template %s Workarounds[%d] not routed: %q", raw.ID, i, w)
				}
			}
		}
	})
}

// TestIssueTemplate_StoresMessageIDsNotLiterals asserts the package-level
// IssueTemplates registry stores stable i18n message IDs (CONST-046), not
// English display text. A regression that re-introduced a hardcoded
// literal would store text containing spaces, which fails this check.
func TestIssueTemplate_StoresMessageIDsNotLiterals(t *testing.T) {
	for _, tpl := range IssueTemplates {
		for _, id := range []string{tpl.Name, tpl.Title, tpl.Description} {
			if strings.Contains(id, " ") {
				t.Errorf("template %s stores a literal, not a message ID: %q", tpl.ID, id)
			}
		}
		for _, s := range tpl.Symptoms {
			if strings.Contains(s, " ") {
				t.Errorf("template %s symptom stores a literal: %q", tpl.ID, s)
			}
		}
		for _, w := range tpl.Workarounds {
			if strings.Contains(w, " ") {
				t.Errorf("template %s workaround stores a literal: %q", tpl.ID, w)
			}
		}
	}
}

// TestNoopTranslatorReturnsMessageID confirms the default NoopTranslator
// emits the messageID verbatim — the seam contract relied on by every
// consumer that does not install a real bundle.
func TestNoopTranslatorReturnsMessageID(t *testing.T) {
	if got := tr("issue.report.title"); got != "issue.report.title" {
		t.Errorf("tr() with NoopTranslator = %q, want verbatim id", got)
	}
	got := trData("issue.report.field.severity", map[string]any{"x": 1})
	if got != "issue.report.field.severity" {
		t.Errorf("trData() with NoopTranslator = %q, want verbatim id", got)
	}
	list := trList("a.b", "c.d")
	if len(list) != 2 || list[0] != "a.b" || list[1] != "c.d" {
		t.Errorf("trList() with NoopTranslator = %v, want verbatim ids", list)
	}
}

// TestModelComparison_GenerateSummary_RoutesThroughTranslator drives
// generateSummary with the fakeTranslator installed and asserts every
// emitted summary fragment routes through the i18n seam (CONST-046
// round-398). Paired-mutation anti-bluff: if generateSummary regressed
// to a hardcoded English literal, the "<TRANSLATED:...>" sentinel would
// be absent and this fails.
func TestModelComparison_GenerateSummary_RoutesThroughTranslator(t *testing.T) {
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("database.New: %v", err)
	}
	defer db.Close()
	engine := NewModelComparisonEngine(db)

	withFakeTranslator(t, func() {
		// Empty model set: the no-models branch.
		empty := engine.generateSummary(&ComparisonResult{Models: []*database.Model{}})
		if !strings.HasPrefix(empty, "<TRANSLATED:enhanced.model_comparison.summary.no_models") {
			t.Errorf("no-models summary not routed: %q", empty)
		}

		// Populated set: count + best-performer + range + differentiators.
		full := engine.generateSummary(&ComparisonResult{
			Models: []*database.Model{{ModelID: "m1"}, {ModelID: "m2"}},
			Rankings: map[string][]ModelRanking{"composite": {
				{ModelID: "m1", Score: 90.0, Rank: 1},
				{ModelID: "m2", Score: 70.0, Rank: 2},
			}},
			Metrics: map[string]MetricComparison{
				"context_window": {Values: map[string]float64{"m1": 8192, "m2": 32768}, BestValue: 32768, WorstValue: 8192},
				"parameters":     {Values: map[string]float64{"m1": 7e9, "m2": 70e9}, BestValue: 70e9, WorstValue: 7e9},
			},
		})
		for _, want := range []string{
			"<TRANSLATED:enhanced.model_comparison.summary.count",
			"<TRANSLATED:enhanced.model_comparison.summary.best_performer",
			"<TRANSLATED:enhanced.model_comparison.summary.performance_range",
			"<TRANSLATED:enhanced.model_comparison.differentiator.context_window",
			"<TRANSLATED:enhanced.model_comparison.differentiator.model_size",
		} {
			if !strings.Contains(full, want) {
				t.Errorf("summary missing routed fragment %q:\n%s", want, full)
			}
		}
	})
}

// TestModelComparison_GenerateRecommendations_RoutesThroughTranslator
// asserts the ModelComparisonEngine recommendation lines route through
// the i18n seam (CONST-046 round-398).
func TestModelComparison_GenerateRecommendations_RoutesThroughTranslator(t *testing.T) {
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("database.New: %v", err)
	}
	defer db.Close()
	engine := NewModelComparisonEngine(db)

	withFakeTranslator(t, func() {
		result := &ComparisonResult{
			Rankings: map[string][]ModelRanking{"composite": {
				{ModelID: "best", Score: 95.0, Rank: 1},
				{ModelID: "other", Score: 80.0, Rank: 2},
			}},
			Metrics: map[string]MetricComparison{
				"context_window":  {Ranking: []string{"other", "best"}},
				"code_capability": {Ranking: []string{"other", "best"}},
			},
		}
		engine.generateRecommendations(result)
		joined := strings.Join(result.Recommendations, "\n")
		for _, want := range []string{
			"<TRANSLATED:enhanced.model_comparison.recommendation.best_overall",
			"<TRANSLATED:enhanced.model_comparison.recommendation.long_conversations",
			"<TRANSLATED:enhanced.model_comparison.recommendation.coding_tasks",
		} {
			if !strings.Contains(joined, want) {
				t.Errorf("recommendations missing routed fragment %q:\n%s", want, joined)
			}
		}
	})
}

// TestModelComparison_MetricLabels_RoutesThroughTranslator asserts the
// metric name/description pairs emitted by compareBasicAttributes route
// through the i18n seam (CONST-046 round-398).
func TestModelComparison_MetricLabels_RoutesThroughTranslator(t *testing.T) {
	db, err := database.New(":memory:")
	if err != nil {
		t.Fatalf("database.New: %v", err)
	}
	defer db.Close()
	engine := NewModelComparisonEngine(db)

	cw := 32768
	pc := int64(7_000_000_000)
	rd := time.Now()
	models := []*database.Model{
		{ModelID: "m1", ContextWindowTokens: &cw, ParameterCount: &pc, ReleaseDate: &rd},
		{ModelID: "m2", ContextWindowTokens: &cw, ParameterCount: &pc, ReleaseDate: &rd},
	}

	withFakeTranslator(t, func() {
		result := &ComparisonResult{Metrics: map[string]MetricComparison{}}
		engine.compareBasicAttributes(result, models)
		for key, prefix := range map[string]string{
			"context_window": "enhanced.model_comparison.metric.context_window",
			"parameters":     "enhanced.model_comparison.metric.parameters",
			"release_date":   "enhanced.model_comparison.metric.release_date",
		} {
			m, ok := result.Metrics[key]
			if !ok {
				t.Errorf("metric %s not produced", key)
				continue
			}
			if !strings.HasPrefix(m.MetricName, "<TRANSLATED:"+prefix+".name") {
				t.Errorf("metric %s name not routed: %q", key, m.MetricName)
			}
			if !strings.HasPrefix(m.Description, "<TRANSLATED:"+prefix+".description") {
				t.Errorf("metric %s description not routed: %q", key, m.Description)
			}
		}
	})
}

// TestGenerateRecommendations_RoutesThroughTranslator drives
// generateRecommendations for every issue type and asserts the emitted
// recommendation lines route through the i18n seam.
func TestGenerateRecommendations_RoutesThroughTranslator(t *testing.T) {
	gir := &GitHubIssueReporter{}
	withFakeTranslator(t, func() {
		for _, it := range []IssueType{
			IssueTypeAvailability, IssueTypePerformance, IssueTypeAccuracy,
			IssueTypeSecurity, IssueTypeCost,
		} {
			issue := &database.Issue{IssueType: string(it)}
			out := gir.generateRecommendations(issue)
			if !strings.Contains(out, "<TRANSLATED:issue.recommendation.") {
				t.Errorf("recommendations for %s not routed:\n%s", it, out)
			}
		}
	})
}

// TestAIAssistant_StaticResponses_RouteThroughTranslator drives the
// AIAssistant chat-reply generators (round-431, CONST-046 Phase 4
// round 33) and asserts every user-facing assistant reply routes
// through the i18n seam. Paired-mutation anti-bluff per §1.1: if any
// generator regressed to a hardcoded English literal, the
// "<TRANSLATED:...>" sentinel would be absent and this fails.
func TestAIAssistant_StaticResponses_RouteThroughTranslator(t *testing.T) {
	ai := &AIAssistant{context: map[string][]string{}}
	withFakeTranslator(t, func() {
		checks := map[string]string{
			"help":                ai.generateHelpResponse(),
			"status":              ai.generateStatusResponse(),
			"suggestion-model":    ai.generateSuggestionResponse("which model is best"),
			"suggestion-general":  ai.generateSuggestionResponse("give me tips"),
			"configuration":       ai.generateConfigurationResponse("help config"),
			"general":             ai.generateGeneralResponse("hello there"),
		}
		for name, got := range checks {
			if !strings.HasPrefix(got, "<TRANSLATED:enhanced.supervisor.") {
				t.Errorf("%s response not routed through i18n seam: %q", name, got)
			}
		}
	})
}

// TestAIAssistant_SuggestionResponse_BranchesRouteDistinctIDs confirms
// the model-specific and general suggestion branches resolve to
// distinct message IDs — a regression that collapsed both branches to
// one literal would produce identical sentinels and fail this.
func TestAIAssistant_SuggestionResponse_BranchesRouteDistinctIDs(t *testing.T) {
	ai := &AIAssistant{}
	withFakeTranslator(t, func() {
		model := ai.generateSuggestionResponse("recommend a model")
		general := ai.generateSuggestionResponse("any advice")
		if model == general {
			t.Errorf("model and general suggestion branches collapsed to same ID: %q", model)
		}
		if model != "<TRANSLATED:enhanced.supervisor.suggestion.model>" {
			t.Errorf("model branch routed unexpected ID: %q", model)
		}
		if general != "<TRANSLATED:enhanced.supervisor.suggestion.general>" {
			t.Errorf("general branch routed unexpected ID: %q", general)
		}
	})
}

// TestAIAssistant_QualitativeHelpers_RouteThroughTranslator drives the
// getSuccessRateMessage / getScoreMessage / getRecommendations helpers
// across every threshold band and asserts each qualitative phrase
// routes through the i18n seam.
func TestAIAssistant_QualitativeHelpers_RouteThroughTranslator(t *testing.T) {
	ai := &AIAssistant{}
	withFakeTranslator(t, func() {
		for _, rate := range []float64{0.99, 0.90, 0.78, 0.50} {
			if got := ai.getSuccessRateMessage(rate); !strings.HasPrefix(got, "<TRANSLATED:enhanced.supervisor.success_rate.") {
				t.Errorf("getSuccessRateMessage(%v) not routed: %q", rate, got)
			}
		}
		for _, score := range []float64{95, 85, 75, 40} {
			if got := ai.getScoreMessage(score); !strings.HasPrefix(got, "<TRANSLATED:enhanced.supervisor.score_quality.") {
				t.Errorf("getScoreMessage(%v) not routed: %q", score, got)
			}
		}
		// Low score + failures: exercises upgrade + investigate + maintenance branches.
		recs := ai.getRecommendations(60, 3)
		for _, want := range []string{
			"<TRANSLATED:enhanced.supervisor.recommendation.upgrade_models",
			"<TRANSLATED:enhanced.supervisor.recommendation.investigate_failures",
			"<TRANSLATED:enhanced.supervisor.recommendation.regular_maintenance",
		} {
			if !strings.Contains(recs, want) {
				t.Errorf("getRecommendations missing routed fragment %q:\n%s", want, recs)
			}
		}
		// High score + zero failures: exercises the performing-well branch.
		recsClean := ai.getRecommendations(95, 0)
		if !strings.Contains(recsClean, "<TRANSLATED:enhanced.supervisor.recommendation.performing_well") {
			t.Errorf("getRecommendations(clean) missing performing-well fragment:\n%s", recsClean)
		}
	})
}
