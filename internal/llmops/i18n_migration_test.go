package llmops

import (
	"context"
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeLLMOpsTranslator returns "<TRANSLATED:msg_id>" so tests can assert the
// i18n-routed sentinel without coupling to the English bundle text.
// Anti-bluff per CONST-035 / Article XI §11.9: a test asserting the original
// literal would silently pass if the call-site bypassed the translator.
type fakeLLMOpsTranslator struct{}

func (fakeLLMOpsTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeLLMOpsTranslator installs the fakeLLMOpsTranslator, runs fn, then
// restores the prior translator.
func withFakeLLMOpsTranslator(t *testing.T, fn func()) {
	t.Helper()
	prior := translator
	translator = fakeLLMOpsTranslator{}
	defer func() { translator = prior }()
	fn()
}

// captureLogger returns a *log.Logger writing into a *strings.Builder so the
// test can read back exactly what a migrated call site emitted — real log
// output, not a mock.
func captureLogger(buf *strings.Builder) *log.Logger {
	return log.New(buf, "", 0)
}

// TestI18nSeam_NoopReturnsMessageIDVerbatim proves the default seam is a
// safe pass-through: production ships NoopTranslator, which returns the
// message ID unchanged so nothing breaks before a real backend is wired.
func TestI18nSeam_NoopReturnsMessageIDVerbatim(t *testing.T) {
	assert.Equal(t, "llmops_log_dataset_created", tr("llmops_log_dataset_created"))
	assert.Equal(t, "llmops_diff_content_changed",
		trData("llmops_diff_content_changed", map[string]any{"old_chars": 1, "new_chars": 2}))
}

// TestI18nSeam_FakeTranslatorRoutesEveryID confirms the seam genuinely
// delegates to the active translator for every migrated message ID.
func TestI18nSeam_FakeTranslatorRoutesEveryID(t *testing.T) {
	ids := []string{
		"llmops_log_dataset_created",
		"llmops_log_evaluation_run_created",
		"llmops_log_evaluation_run_completed",
		"llmops_alert_pass_rate_dropped",
		"llmops_log_alert_created",
		"llmops_log_experiment_created",
		"llmops_log_experiment_started",
		"llmops_log_experiment_paused",
		"llmops_log_experiment_completed",
		"llmops_log_system_initialized",
		"llmops_experiment_desc_ab_prompt_versions",
		"llmops_log_prompt_created",
		"llmops_log_prompt_activated",
		"llmops_log_prompt_deleted",
		"llmops_diff_content_changed",
	}
	withFakeLLMOpsTranslator(t, func() {
		for _, id := range ids {
			assert.Equal(t, "<TRANSLATED:"+id+">", tr(id), "tr(%q) must route through translator", id)
			assert.Equal(t, "<TRANSLATED:"+id+">", trData(id, nil), "trData(%q) must route through translator", id)
		}
	})
}

// TestPromptRegistry_CreateLogIsI18nRouted exercises the real Create path and
// asserts the emitted log line is the i18n sentinel, never an English literal.
func TestPromptRegistry_CreateLogIsI18nRouted(t *testing.T) {
	withFakeLLMOpsTranslator(t, func() {
		var buf strings.Builder
		registry := NewInMemoryPromptRegistry(captureLogger(&buf))

		err := registry.Create(context.Background(),
			&PromptVersion{Name: "greeting", Version: "1.0.0", Content: "Hi"})
		require.NoError(t, err)

		out := buf.String()
		assert.Contains(t, out, "<TRANSLATED:llmops_log_prompt_created>",
			"Create must emit the i18n-routed sentinel")
		// Paired-mutation guard: the pre-migration English literal must be gone.
		assert.NotContains(t, out, "Created prompt greeting version",
			"pre-migration English literal must NOT survive the migration")
	})
}

// TestExperimentManager_LogsAreI18nRouted exercises Create/Start/Pause/Complete
// and asserts every log line routes through the i18n seam.
func TestExperimentManager_LogsAreI18nRouted(t *testing.T) {
	withFakeLLMOpsTranslator(t, func() {
		var buf strings.Builder
		mgr := NewInMemoryExperimentManager(captureLogger(&buf))
		ctx := context.Background()

		exp := &Experiment{
			Name:     "ab",
			Variants: []*Variant{{Name: "Control", IsControl: true}, {Name: "Treatment"}},
		}
		require.NoError(t, mgr.Create(ctx, exp))
		require.NoError(t, mgr.Start(ctx, exp.ID))
		require.NoError(t, mgr.Pause(ctx, exp.ID))
		require.NoError(t, mgr.Complete(ctx, exp.ID, exp.Variants[0].ID))

		out := buf.String()
		for _, id := range []string{
			"<TRANSLATED:llmops_log_experiment_created>",
			"<TRANSLATED:llmops_log_experiment_started>",
			"<TRANSLATED:llmops_log_experiment_paused>",
			"<TRANSLATED:llmops_log_experiment_completed>",
		} {
			assert.Contains(t, out, id, "experiment lifecycle must route %s", id)
		}
		// Paired-mutation guard: pre-migration literals must be absent.
		assert.NotContains(t, out, "Created experiment: ab")
		assert.NotContains(t, out, "Started experiment: ab")
	})
}

// TestPromptComparator_ContentDiffIsI18nRouted proves the user-facing diff
// string returned by Compare routes through the seam — a returned value, not
// just a log line, so the assertion covers product-visible output.
func TestPromptComparator_ContentDiffIsI18nRouted(t *testing.T) {
	withFakeLLMOpsTranslator(t, func() {
		registry := NewInMemoryPromptRegistry(nil)
		ctx := context.Background()
		require.NoError(t, registry.Create(ctx,
			&PromptVersion{Name: "p", Version: "1", Content: "short"}))
		require.NoError(t, registry.Create(ctx,
			&PromptVersion{Name: "p", Version: "2", Content: "a much longer content body"}))

		cmp := NewPromptVersionComparator(registry, nil)
		diff, err := cmp.Compare(ctx, "p", "1", "2")
		require.NoError(t, err)

		assert.Equal(t, "<TRANSLATED:llmops_diff_content_changed>", diff.ContentDiff,
			"ContentDiff must be i18n-routed, not a hardcoded English literal")
		// Paired-mutation guard.
		assert.NotContains(t, diff.ContentDiff, "Content changed from")
	})
}

// TestLLMOpsSystem_InitLogIsI18nRouted exercises the real Initialize path and
// checks the system-initialized log line routes through the seam.
func TestLLMOpsSystem_InitLogIsI18nRouted(t *testing.T) {
	withFakeLLMOpsTranslator(t, func() {
		var buf strings.Builder
		sys := NewLLMOpsSystem(nil, captureLogger(&buf))
		require.NoError(t, sys.Initialize())

		out := buf.String()
		assert.Contains(t, out, "<TRANSLATED:llmops_log_system_initialized>")
		assert.NotContains(t, out, "LLMOps system initialized",
			"pre-migration English literal must NOT survive")
	})
}

// TestEvaluator_DatasetAndAlertLogsAreI18nRouted exercises CreateDataset and
// the alert-creation path, asserting both route through the i18n seam.
func TestEvaluator_DatasetAndAlertLogsAreI18nRouted(t *testing.T) {
	withFakeLLMOpsTranslator(t, func() {
		var evalBuf, alertBuf strings.Builder
		alertMgr := NewInMemoryAlertManager(captureLogger(&alertBuf))
		ev := NewInMemoryContinuousEvaluator(nil, alertMgr, nil, captureLogger(&evalBuf))
		ctx := context.Background()

		require.NoError(t, ev.CreateDataset(ctx, &Dataset{Name: "golden"}))
		assert.Contains(t, evalBuf.String(), "<TRANSLATED:llmops_log_dataset_created>")
		assert.NotContains(t, evalBuf.String(), "Created dataset: golden")

		require.NoError(t, alertMgr.Create(ctx,
			&Alert{Type: AlertTypeRegression, Message: "m"}))
		assert.Contains(t, alertBuf.String(), "<TRANSLATED:llmops_log_alert_created>")
		assert.NotContains(t, alertBuf.String(), "Created alert: regression")
	})
}
