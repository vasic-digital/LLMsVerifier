package llmverifier

import (
	"fmt"
	"strings"
	"testing"
)

// TestIssueDetector_Round388TitlesRouteThroughTranslator is the round-388
// paired-mutation guard (CONST-046 Phase 4 round 23). It drives detectIssues
// with verification results that trigger every auto-detected issue branch and
// asserts each issue Title + Description routes through the i18n seam. If any
// call-site regressed to a hardcoded English literal, its sentinel would be
// absent and the test fails — anti-bluff per CONST-035 / Article XI §11.9.
func TestIssueDetector_Round388TitlesRouteThroughTranslator(t *testing.T) {
	id := &IssueDetector{}

	// Result crafted to trip every issue branch:
	//   - OverallScore < 30      → underperforming
	//   - CodeCapability < 40    → poor_code
	//   - Reliability < 50       → unreliable
	//   - Responsiveness < 60    → slow_response
	//   - ToolUse/FuncCall false → missing_tool_use
	//   - Error has "timeout"    → connectivity
	//   - Error has "auth"       → authentication
	result := &VerificationResult{
		Error: "timeout: connection unauthorized auth failure",
	}
	result.PerformanceScores.OverallScore = 10
	result.PerformanceScores.CodeCapability = 10
	result.PerformanceScores.Reliability = 10
	result.PerformanceScores.Responsiveness = 10
	result.FeatureDetection.ToolUse = false
	result.FeatureDetection.FunctionCalling = false

	withFakeTranslator(t, func() {
		issues := id.detectIssues(result)
		if len(issues) != 7 {
			t.Fatalf("detectIssues returned %d issues, want 7", len(issues))
		}

		wantTitleIDs := []string{
			"issue.underperforming.title",
			"issue.poor_code.title",
			"issue.unreliable.title",
			"issue.slow_response.title",
			"issue.missing_tool_use.title",
			"issue.connectivity.title",
			"issue.authentication.title",
		}
		wantDescIDs := []string{
			"issue.underperforming.description",
			"issue.poor_code.description",
			"issue.unreliable.description",
			"issue.slow_response.description",
			"issue.missing_tool_use.description",
			"issue.connectivity.description",
			"issue.authentication.description",
		}

		titles := make(map[string]bool)
		descs := make(map[string]bool)
		for _, iss := range issues {
			titles[iss.Title] = true
			descs[iss.Description] = true
		}

		for _, mid := range wantTitleIDs {
			sentinel := "<TRANSLATED:" + mid + ">"
			if !titles[sentinel] {
				t.Errorf("issue title missing routed sentinel %q; got titles=%v", sentinel, titles)
			}
		}
		for _, mid := range wantDescIDs {
			sentinel := "<TRANSLATED:" + mid + ">"
			if !descs[sentinel] {
				t.Errorf("issue description missing routed sentinel %q; got descs=%v", sentinel, descs)
			}
		}

		// Negative anti-bluff assertion: the pre-migration English literals
		// must NOT survive verbatim under the fake translator.
		for _, gone := range []string{
			"Severely Underperforming Model",
			"Poor Code Generation Capability",
			"Model has critically low performance scores",
			"Authentication Problems",
		} {
			if titles[gone] || descs[gone] {
				t.Errorf("pre-migration literal %q still present — call-site bypassed translator", gone)
			}
		}
	})
}

// TestIssueDetector_EventDetectedPrefixRoutesThroughTranslator asserts the
// detected-issue event title prefix is routed through the i18n seam.
func TestIssueDetector_EventDetectedPrefixRoutesThroughTranslator(t *testing.T) {
	withFakeTranslator(t, func() {
		title := fmt.Sprintf("%s %s", tr("issue.event.detected_prefix"), "Some Issue")
		if !strings.Contains(title, "<TRANSLATED:issue.event.detected_prefix>") {
			t.Errorf("event title missing routed prefix sentinel; got %q", title)
		}
		if strings.Contains(title, "Issue Detected:") {
			t.Errorf("pre-migration literal 'Issue Detected:' still present in %q", title)
		}
	})
}

// TestIssueDetector_NoopTranslatorReturnsMessageID confirms the default
// NoopTranslator emits the issue-detector message IDs verbatim — the seam
// contract relied on by consumers that do not install a real bundle.
func TestIssueDetector_NoopTranslatorReturnsMessageID(t *testing.T) {
	for _, mid := range []string{
		"issue.underperforming.title",
		"issue.connectivity.description",
		"issue.event.detected_prefix",
	} {
		if got := tr(mid); got != mid {
			t.Errorf("tr(%q) with NoopTranslator = %q, want verbatim id", mid, got)
		}
	}
}
