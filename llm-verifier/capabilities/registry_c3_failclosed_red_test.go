package capabilities

import (
	"os"
	"testing"
)

// TestC3RegistrySeedNotSelfCertifiedVerified — §11.4.115 RED-baseline for C3
// (10b §3 C3, §2.5). Gap: GetProviderBaseCapabilities("openai") returns a seed
// self-certified .Verified==true (registry.go:11) with NO probe backing.
// Post-C3, absent a fresh probe the accessor MUST report unverified/fail-closed.
//
// Polarity switch (§11.4.115): RED_MODE=1 reproduces the defect on the broken
// (pre-C3) artifact and PASSes there; RED_MODE=0 (default) is the standing GREEN
// regression guard that FAILs on the broken artifact and PASSes once fail-closed
// C3 lands. Registered in the §11.4.135 standing regression-guard suite.
func TestC3RegistrySeedNotSelfCertifiedVerified(t *testing.T) {
	redMode := os.Getenv("RED_MODE")
	if redMode == "" {
		redMode = "0"
	}
	caps := GetProviderBaseCapabilities("openai")
	if caps == nil {
		t.Fatalf("GetProviderBaseCapabilities(\"openai\") returned nil; expected a seed entry")
	}
	selfCertifiedVerified := caps.Verified // no probe supplied
	switch redMode {
	case "1": // reproduce: PASSes on broken (current) artifact
		if !selfCertifiedVerified {
			t.Fatalf("RED_MODE=1: expected seed self-certified Verified==true absent a probe, got %v", caps.Verified)
		}
		t.Logf("RED_MODE=1 PASS: defect reproduced — unbacked seed self-certified Verified==true (§2.5).")
	case "0": // guard: FAILs on broken, PASSes once fail-closed C3 lands
		if selfCertifiedVerified {
			t.Fatalf("RED_MODE=0: fail-closed C3 not implemented — unbacked seed self-certified Verified==true; " +
				"absent a fresh probe the accessor MUST report unverified/fail-closed (10b §3 C3 part 3).")
		}
		t.Logf("RED_MODE=0 PASS: accessor reports unverified/fail-closed absent a probe.")
	default:
		t.Fatalf("unknown RED_MODE=%q (expected 0 or 1)", redMode)
	}
}
