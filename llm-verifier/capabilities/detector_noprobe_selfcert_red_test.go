package capabilities

import (
	"context"
	"os"
	"testing"
)

// TestDetectorNoProbeSelfCertRED — §11.4.115 RED-baseline-on-the-broken-artifact
// for the detector no-probe self-certification bug (review advisory 1).
//
// Gap: DetectProviderCapabilities sets caps.Verified = true UNCONDITIONALLY
// (detector.go:61) — including the apiKey=="" branch where the probe block
// (detector.go:56-59) is skipped, so it stamps Verified=true with NO probe ever
// run. Fail-closed (consistent with the C3 seed invariant enforced by
// TestC3RegistrySeedNotSelfCertifiedVerified) requires Verified to stay false
// when no probe ran.
//
// Polarity switch (§11.4.115): RED_MODE=1 reproduces the defect on the pre-fix
// artifact (a no-key call yields Verified==true) and PASSes there; RED_MODE=0
// (default) is the standing GREEN regression guard that FAILs on the broken
// artifact and PASSes once the no-probe branch stays fail-closed. Registered in
// the §11.4.135 standing regression-guard suite.
func TestDetectorNoProbeSelfCertRED(t *testing.T) {
	redMode := os.Getenv("RED_MODE")
	if redMode == "" {
		redMode = "0"
	}

	d := NewDetector()
	// Empty apiKey => the probe block (detector.go:56-59) is skipped, so NO
	// probe runs against any provider endpoint. No network call is made.
	caps, err := d.DetectProviderCapabilities(context.Background(), "openai", "")
	if err != nil {
		t.Fatalf("DetectProviderCapabilities returned unexpected error: %v", err)
	}
	if caps == nil {
		t.Fatalf("DetectProviderCapabilities returned nil caps")
	}

	noProbeVerified := caps.Verified // no probe was run (empty apiKey)
	switch redMode {
	case "1": // reproduce: PASSes on the broken (pre-fix) artifact
		if !noProbeVerified {
			t.Fatalf("RED_MODE=1: expected Verified==true with NO probe run (empty apiKey), got %v", caps.Verified)
		}
		t.Logf("RED_MODE=1 PASS: defect reproduced — Verified stamped true with no probe run.")
	case "0": // guard: FAILs on broken, PASSes once fail-closed lands
		if noProbeVerified {
			t.Fatalf("RED_MODE=0: no-probe self-cert not fixed — Verified==true with empty apiKey and no probe run; " +
				"absent a probe the detector MUST leave Verified false (fail-closed, C3 seed invariant).")
		}
		t.Logf("RED_MODE=0 PASS: no-probe branch fail-closed — Verified stays false absent a probe.")
	default:
		t.Fatalf("unknown RED_MODE=%q (expected 0 or 1)", redMode)
	}
}
