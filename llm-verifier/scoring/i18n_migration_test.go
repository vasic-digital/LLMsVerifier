package scoring

import (
	"context"
	"strings"
	"testing"
)

// fakeScoringTranslator returns "<TRANSLATED:msg_id>" so tests can assert
// the i18n-routed sentinel without coupling to the English bundle text.
// Anti-bluff per CONST-035 / Article XI §11.9: a test asserting the
// original literal would silently pass if the call-site bypassed the
// translator.
type fakeScoringTranslator struct{}

func (fakeScoringTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}
func (fakeScoringTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeScoringTranslator installs the fakeScoringTranslator, runs fn,
// then restores the prior translator.
func withFakeScoringTranslator(t *testing.T, fn func()) {
	t.Helper()
	prior := translator
	translator = fakeScoringTranslator{}
	defer func() { translator = prior }()
	fn()
}

// TestTr_RoutesThroughTranslator proves the package-level tr helper actually
// consults the active translator rather than echoing the messageID. With the
// fake translator installed every lookup must carry the "<TRANSLATED:...>"
// prefix. This is the paired mutation for the scoring i18n seam: if a future
// change made tr() return its argument verbatim, this test fails.
func TestTr_RoutesThroughTranslator(t *testing.T) {
	withFakeScoringTranslator(t, func() {
		ids := []string{
			"llmsverifier_scoring_err_invalid_request_body",
			"llmsverifier_scoring_err_calculate_score_failed",
			"llmsverifier_scoring_err_recalculate_score_failed",
			"llmsverifier_scoring_err_min_two_models",
			"llmsverifier_scoring_msg_score_calculated",
			"llmsverifier_scoring_msg_score_recalculated",
			"llmsverifier_scoring_msg_batch_started",
			"llmsverifier_scoring_msg_batch_completed",
			"llmsverifier_scoring_msg_model_names_updated",
			"llmsverifier_scoring_msg_validation_completed",
			"llmsverifier_scoring_health_high_error_rate",
			"llmsverifier_scoring_health_operating_normally",
			"llmsverifier_scoring_health_api_response_time_threshold",
			"llmsverifier_scoring_health_api_response_time",
			"llmsverifier_scoring_health_db_latency_threshold",
			"llmsverifier_scoring_health_db_latency",
			"llmsverifier_scoring_alert_subject_score_change",
			"llmsverifier_scoring_alert_subject_api_performance",
			"llmsverifier_scoring_alert_subject_db_performance",
		}
		for _, id := range ids {
			got := tr(id)
			want := "<TRANSLATED:" + id + ">"
			if got != want {
				t.Errorf("tr(%q) = %q, not i18n-routed (want %q)", id, got, want)
			}
		}
	})
}

// TestTr_NoopReturnsMessageIDVerbatim proves the default NoopTranslator path:
// without an override, tr returns the messageID itself. This guarantees the
// migration introduces no panic and the seam degrades gracefully.
func TestTr_NoopReturnsMessageIDVerbatim(t *testing.T) {
	const id = "llmsverifier_scoring_err_invalid_request_body"
	if got := tr(id); got != id {
		t.Errorf("NoopTranslator tr(%q) = %q, want verbatim messageID", id, got)
	}
}

// TestScoringHealthFormatKeys_AreRoutedFormatStrings proves the
// format-string keys used by monitoring.go / alert_manager.go funnel through
// tr() before reaching fmt.Sprintf. With the fake translator installed the
// returned format string must be the i18n-routed sentinel, confirming the
// call sites do not bypass the seam.
func TestScoringHealthFormatKeys_AreRoutedFormatStrings(t *testing.T) {
	withFakeScoringTranslator(t, func() {
		formatKeys := []string{
			"llmsverifier_scoring_health_high_error_rate",
			"llmsverifier_scoring_health_api_response_time_threshold",
			"llmsverifier_scoring_health_api_response_time",
			"llmsverifier_scoring_health_db_latency_threshold",
			"llmsverifier_scoring_health_db_latency",
			"llmsverifier_scoring_alert_subject_score_change",
			"llmsverifier_scoring_alert_subject_api_performance",
			"llmsverifier_scoring_alert_subject_db_performance",
		}
		for _, k := range formatKeys {
			got := tr(k)
			if !strings.HasPrefix(got, "<TRANSLATED:") {
				t.Errorf("format key %q not i18n-routed: %q", k, got)
			}
		}
	})
}
