package benchmark

import (
	"context"
	"log"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeBenchmarkTranslator returns "<TRANSLATED:msg_id>" so tests can assert
// the i18n-routed sentinel without coupling to the English bundle text.
// Anti-bluff per CONST-035 / Article XI §11.9: a test asserting the original
// literal would silently pass if the call-site bypassed the translator.
type fakeBenchmarkTranslator struct{}

func (fakeBenchmarkTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeBenchmarkTranslator installs the fakeBenchmarkTranslator, runs fn,
// then restores the prior translator.
func withFakeBenchmarkTranslator(t *testing.T, fn func()) {
	t.Helper()
	prior := translator
	translator = fakeBenchmarkTranslator{}
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
	assert.Equal(t, "benchmark.log.run_created", tr("benchmark.log.run_created"))
	assert.Equal(t, "benchmark.err.run_not_found",
		trData("benchmark.err.run_not_found", map[string]any{"id": "x"}))
}

// TestI18nSeam_FakeTranslatorRoutesEveryID confirms the seam genuinely
// delegates to the active translator for every migrated message ID.
func TestI18nSeam_FakeTranslatorRoutesEveryID(t *testing.T) {
	ids := []string{
		"benchmark.log.system_initialized",
		"benchmark.log.run_created",
		"benchmark.log.run_completed",
		"benchmark.err.system_not_initialized",
		"benchmark.err.benchmark_not_found",
		"benchmark.err.run_not_found",
		"benchmark.summary.run_comparison",
	}
	withFakeBenchmarkTranslator(t, func() {
		for _, id := range ids {
			assert.Equal(t, "<TRANSLATED:"+id+">", tr(id), "tr(%q) must route through translator", id)
			assert.Equal(t, "<TRANSLATED:"+id+">",
				trData(id, map[string]any{"k": "v"}), "trData(%q) must route through translator", id)
		}
	})
}

// TestRunNotFound_RoutesThroughTranslator is the paired-mutation guard for
// the GetTasks / StartRun / GetRun / CancelRun / CompareRuns error paths.
// With the fake translator the surfaced error MUST carry the i18n sentinel,
// not the English "run not found" literal. If a future edit reverts the
// call site to a raw fmt.Errorf("run not found: %s", ...) this test FAILS.
func TestRunNotFound_RoutesThroughTranslator(t *testing.T) {
	r := NewStandardBenchmarkRunner(nil, captureLoggerDiscard())
	withFakeBenchmarkTranslator(t, func() {
		_, err := r.GetRun(context.Background(), "nonexistent-run")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "<TRANSLATED:benchmark.err.run_not_found>")
		assert.NotContains(t, err.Error(), "run not found:",
			"raw English literal must not leak when translator is active")

		_, err = r.GetTasks(context.Background(), "nonexistent-benchmark", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "<TRANSLATED:benchmark.err.benchmark_not_found>")
	})
}

// TestRunCreated_LogRoutesThroughTranslator is the paired-mutation guard for
// the CreateRun log line. The captured real log output MUST contain the
// i18n sentinel.
func TestRunCreated_LogRoutesThroughTranslator(t *testing.T) {
	var buf strings.Builder
	r := NewStandardBenchmarkRunner(nil, captureLogger(&buf))
	withFakeBenchmarkTranslator(t, func() {
		require.NoError(t, r.CreateRun(context.Background(), &BenchmarkRun{}))
	})
	assert.Contains(t, buf.String(), "<TRANSLATED:benchmark.log.run_created>")
	assert.NotContains(t, buf.String(), "Created benchmark run:",
		"raw English literal must not leak when translator is active")
}

func captureLoggerDiscard() *log.Logger {
	return log.New(&strings.Builder{}, "", 0)
}
