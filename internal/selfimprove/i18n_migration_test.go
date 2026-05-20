package selfimprove

import (
	"context"
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeSelfimproveTranslator returns "<TRANSLATED:msg_id>" so tests can assert
// the i18n-routed sentinel without coupling to the English bundle text.
// Anti-bluff per CONST-035 / Article XI §11.9: a test asserting the original
// literal would silently pass if the call-site bypassed the translator.
type fakeSelfimproveTranslator struct{}

func (fakeSelfimproveTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeSelfimproveTranslator installs the fakeSelfimproveTranslator, runs
// fn, then restores the prior translator.
func withFakeSelfimproveTranslator(t *testing.T, fn func()) {
	t.Helper()
	prior := translator
	translator = fakeSelfimproveTranslator{}
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
	assert.Equal(t, "selfimprove.log.system_initialized", tr("selfimprove.log.system_initialized"))
	assert.Equal(t, "selfimprove.err.policy_update_not_found",
		trData("selfimprove.err.policy_update_not_found", map[string]any{"id": "x"}))
}

// TestI18nSeam_FakeTranslatorRoutesEveryID confirms the seam genuinely
// delegates to the active translator for every migrated message ID.
func TestI18nSeam_FakeTranslatorRoutesEveryID(t *testing.T) {
	ids := []string{
		"selfimprove.log.system_initialized",
		"selfimprove.log.policy_update_applied",
		"selfimprove.log.policy_update_rolled_back",
		"selfimprove.err.policy_update_not_found",
		"selfimprove.log.debate_failed_fallback",
		"selfimprove.log.training_reward_model",
	}
	withFakeSelfimproveTranslator(t, func() {
		for _, id := range ids {
			assert.Equal(t, "<TRANSLATED:"+id+">", tr(id), "tr(%q) must route through translator", id)
			assert.Equal(t, "<TRANSLATED:"+id+">",
				trData(id, map[string]any{"k": "v"}), "trData(%q) must route through translator", id)
		}
	})
}

// TestSystemInitialized_LogRoutesThroughTranslator is the paired-mutation
// guard for the SelfImprovementSystem.Initialize log line. The captured real
// log output MUST contain the i18n sentinel — if a future edit reverts the
// call site to logger.Println("Self-improvement system initialized") this
// test FAILS.
func TestSystemInitialized_LogRoutesThroughTranslator(t *testing.T) {
	var buf strings.Builder
	sys := NewSelfImprovementSystem(nil, captureLogger(&buf))
	withFakeSelfimproveTranslator(t, func() {
		require.NoError(t, sys.Initialize(nil, nil))
	})
	assert.Contains(t, buf.String(), "<TRANSLATED:selfimprove.log.system_initialized>")
	assert.NotContains(t, buf.String(), "Self-improvement system initialized",
		"raw English literal must not leak when translator is active")
}

// TestPolicyUpdateNotFound_RoutesThroughTranslator is the paired-mutation
// guard for the LLMPolicyOptimizer.Rollback error path. With the fake
// translator the surfaced error MUST carry the i18n sentinel, not the
// English "update not found" literal.
func TestPolicyUpdateNotFound_RoutesThroughTranslator(t *testing.T) {
	opt := NewLLMPolicyOptimizer(nil, nil, nil, captureLogger(&strings.Builder{}))
	withFakeSelfimproveTranslator(t, func() {
		err := opt.Rollback(context.Background(), "nonexistent-update")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "<TRANSLATED:selfimprove.err.policy_update_not_found>")
		assert.NotContains(t, err.Error(), "update not found:",
			"raw English literal must not leak when translator is active")
	})
}
