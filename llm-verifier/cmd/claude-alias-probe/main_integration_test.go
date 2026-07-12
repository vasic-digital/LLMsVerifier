//go:build integration
// +build integration

package main

// main_integration_test.go -- REAL end-to-end exercise of this command's
// run() against the real `claude` CLI (§11.4.27(A): non-unit tests MUST
// use the real system, no mocks). Mirrors providers/
// claudecode_integration_test.go's SKIP-not-FAIL topology dispatch
// (§11.4.3) and its quota-minimal design (PART G risk 1 of the
// incorporation plan): exactly ONE real completion for the positive
// case, plus a near-zero-cost negative case against an unreachable port
// (the CLI fails to connect before any token is spent).

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func claudeCLIAvailableForTest(t *testing.T) bool {
	t.Helper()
	_, err := exec.LookPath("claude")
	return err == nil
}

// TestRun_LiveSentinel_ProviderRouter_ViaCCR is the ONE real, billed
// `claude -p` call this test file makes (quota-minimal per PART G risk 1)
// -- routed through the local claude-code-router (ccr) exactly like
// providers/claudecode_integration_test.go's positive case, but driven
// through THIS command's run() (flags -> JSON stdout -> exit code),
// proving the CLI wrapper itself -- not just the underlying adapter --
// is wired correctly end to end.
func TestRun_LiveSentinel_ProviderRouter_ViaCCR(t *testing.T) {
	if !claudeCLIAvailableForTest(t) {
		t.Skip("SKIP: claude CLI not found on PATH")
	}

	const sentinel = "ATMOSPHERE_LLMSVERIFIER_PWU3_BRIDGE_CLI_SENTINEL_2c6b1a"
	args := []string{
		"--kind", "provider-router",
		"--base-url", "http://127.0.0.1:3456",
		"--sentinel", sentinel,
		"--timeout", "180",
	}

	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)

	// A ccr-unreachable OR heavily-contended host is an honest SKIP, not a
	// FAIL (§11.4.3): the bridge classifies "could not connect" / "did
	// not complete in time" as an infra/failed verdict (exit 3), which is
	// indistinguishable at the exit-code layer from a genuinely
	// rate-limited/quota-exhausted route without inspecting the JSON --
	// so inspect stdout for the specific unreachable-router / timeout
	// signal before deciding SKIP vs FAIL. A timeout is an expected,
	// non-defect outcome on a host running multiple concurrent `claude -p`
	// workers sharing the same local ccr instance (§11.4.103 continuous
	// parallel-stream routine; see the sibling
	// providers/claudecode_integration_test.go's identical rationale for
	// its own 180s live-probe timeout).
	out := stdout.String()
	if code == 3 && (strings.Contains(out, "connection refused") ||
		strings.Contains(out, "no such host") ||
		strings.Contains(out, "claude command not found") ||
		strings.Contains(out, "timed out")) {
		t.Skipf("SKIP: ccr / claude CLI not reachable, or the shared router is too contended to complete within budget, in this environment: %s", out)
	}

	if code != 0 {
		t.Fatalf("expected exit 0 (ok) for a live sentinel round-trip, got %d; stdout=%s stderr=%s", code, out, stderr.String())
	}
	if !strings.Contains(out, `"verdict": "ok"`) {
		t.Fatalf("expected verdict==ok in JSON output, got: %s", out)
	}
	if !strings.Contains(out, `"sentinel_matched": true`) {
		t.Fatalf("SENTINEL PASS-BLUFF GUARD TRIPPED: expected sentinel_matched=true, got: %s", out)
	}

	t.Logf("PASS (real §11.4.69 sink-side probe via the claude-alias-probe CLI, routed through ccr): %s", out)
}

// TestRun_NegativeCase_UnreachableRoute_NoFalsePass is the required
// negative case (near-zero cost -- the CLI fails to connect before any
// token is spent): pointing the bridge at a port nothing listens on MUST
// NEVER produce verdict=="ok".
func TestRun_NegativeCase_UnreachableRoute_NoFalsePass(t *testing.T) {
	if !claudeCLIAvailableForTest(t) {
		t.Skip("SKIP: claude CLI not found on PATH")
	}

	const sentinel = "ATMOSPHERE_LLMSVERIFIER_PWU3_NEGATIVE_SENTINEL_9f2e7d"
	args := []string{
		"--kind", "provider-router",
		"--base-url", "http://127.0.0.1:1", // reserved port; nothing listens here
		"--sentinel", sentinel,
		"--timeout", "10",
	}

	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)

	out := stdout.String()
	if strings.Contains(out, `"verdict": "ok"`) {
		t.Fatalf("FALSE-PASS DETECTED: an unreachable route produced verdict==ok: %s", out)
	}
	if code == 0 {
		t.Fatalf("expected a non-zero exit code for an unreachable route, got 0; stdout=%s", out)
	}

	t.Logf("PASS (negative case correctly rejected, no false-pass): exit=%d stdout=%s stderr=%s", code, out, stderr.String())
}
