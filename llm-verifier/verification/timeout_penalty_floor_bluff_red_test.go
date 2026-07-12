package verification

import "testing"

// TestApplyVerificationTimeoutPenalty_BlindModelMustNotBeFlooredTo0Point3 is
// the §11.4.115 RED-on-broken-artifact regression guard for the SAME
// score-floor-for-a-failed-model bluff class fixed for the initial 0.7 floor
// (see verification_score_floor_bluff_red_test.go), reproduced in the
// timeout-penalty branch of VerifyModelCodeVisibility.
//
// ROOT CAUSE (independent-audit finding, 2026-07-10): the pre-fix line
//
//	result.VerificationScore = max(result.VerificationScore-timeoutPenalty, 0.3)
//
// applied a 0.3 floor UNCONDITIONALLY, regardless of whether the model ever
// demonstrated code visibility. A model that never demonstrated code
// visibility (VerificationScore == 0.0, CodeVisibility == false) and also
// happened to be slow (avg response time > 20s) would be bumped from its
// true 0.0 score up to 0.3 -- the identical inflate-a-failed-model's-score
// bug fixed above, reproduced in this second code path.
func TestApplyVerificationTimeoutPenalty_BlindModelMustNotBeFlooredTo0Point3(t *testing.T) {
	// A code-blind model (CodeVisibility == false) that measured 0.0
	// confidence, and was also slow (25s avg response time -- inside the
	// 20001-30000ms "penalty=0.1" tier).
	adjusted, penalty, applied := applyVerificationTimeoutPenalty(0.0, 25000, false /* codeVisible */)

	if !applied {
		t.Fatalf("expected the timeout penalty to apply for a 25000ms avg response time")
	}
	if penalty != 0.1 {
		t.Fatalf("expected penalty=0.1 for the 20001-30000ms tier, got %.2f", penalty)
	}

	// BLUFF CHECK: 0.0 - 0.1 = -0.1, which must clamp at 0 (a score can never
	// go negative) -- NOT float back up to the 0.3 "verified but slow" floor,
	// because this model was never verified in the first place.
	if adjusted != 0.0 {
		t.Fatalf("SCORE-FLOOR BLUFF: a code-blind (CodeVisibility=false) model's timeout-penalised "+
			"score must clamp at 0.0, not float up to the 0.3 floor reserved for models that "+
			"actually demonstrated code visibility; got adjusted=%.2f", adjusted)
	}

	// Control: the SAME slow response time for a model that DID demonstrate
	// code visibility (codeVisible == true, starting score 0.4) still gets
	// the original 0.3 floor behaviour -- this fix must not regress the
	// legitimate "slow but verified" case.
	adjustedVisible, _, appliedVisible := applyVerificationTimeoutPenalty(0.4, 25000, true /* codeVisible */)
	if !appliedVisible {
		t.Fatalf("expected the timeout penalty to apply for a 25000ms avg response time (visible case)")
	}
	// float64 0.4-0.1 == 0.30000000000000004, not exactly 0.3 -- compare with
	// an epsilon rather than exact equality.
	const epsilon = 1e-9
	if diff := adjustedVisible - 0.3; diff > epsilon || diff < -epsilon {
		t.Fatalf("expected a verified-but-slow model (0.4 - 0.1 ~= 0.3) to stay at its computed score "+
			"(already at the 0.3 floor); got %.17f", adjustedVisible)
	}
}
