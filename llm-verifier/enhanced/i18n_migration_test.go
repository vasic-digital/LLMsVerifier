package enhanced

import (
	"context"
	"strings"
	"testing"

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
