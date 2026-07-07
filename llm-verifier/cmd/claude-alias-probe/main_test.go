package main

// main_test.go -- unit tests for the pure config/judge functions
// (buildAliasConfig, judgeOutcome) plus the flag-validation (exit-2)
// paths of run(). These are UNIT tests (§11.4.27(A)): no real `claude`
// process is invoked, no network call is made -- they exercise this
// command's own argument handling and verdict-mapping logic in
// isolation, quota-free. The REAL end-to-end exec path (real `claude -p`
// via the adapter) is exercised by main_integration_test.go
// (build tag integration) and by the constitution/scripts/
// llm-alias-health util's own integration test, per the incorporation
// plan's quota-minimal design (PART G risk 1).

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"

	"digital.vasic.llmsverifier/providers"
)

func TestBuildAliasConfig_Native_RequiresConfigDir(t *testing.T) {
	if _, err := buildAliasConfig("native", "", "", "", "", ""); err == nil {
		t.Fatal("expected error when --config-dir is missing for kind=native")
	}
	cfg, err := buildAliasConfig("native", "/tmp/.claude-x", "", "", "", "opus")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Kind != providers.ClaudeCodeAliasNative || cfg.ConfigDir != "/tmp/.claude-x" || cfg.AnthropicModel != "opus" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestBuildAliasConfig_ProviderNative_RequiresBaseURLAndToken(t *testing.T) {
	if _, err := buildAliasConfig("provider-native", "", "", "", "", ""); err == nil {
		t.Fatal("expected error when --base-url is missing for kind=provider-native")
	}
	if _, err := buildAliasConfig("provider-native", "", "https://api.example.com", "", "", ""); err == nil {
		t.Fatal("expected error when --auth-token-env is missing for kind=provider-native")
	}

	const envName = "TEST_CLAUDE_ALIAS_PROBE_TOKEN_UNSET"
	if _, err := buildAliasConfig("provider-native", "", "https://api.example.com", envName, "", ""); err == nil {
		t.Fatal("expected error when the named env var is unset/empty")
	}

	t.Setenv("TEST_CLAUDE_ALIAS_PROBE_TOKEN_SET", "secret-token-value")
	cfg, err := buildAliasConfig("provider-native", "", "https://api.example.com", "TEST_CLAUDE_ALIAS_PROBE_TOKEN_SET", "", "some-model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Kind != providers.ClaudeCodeAliasProviderNative || cfg.AnthropicAuthToken != "secret-token-value" || cfg.AnthropicBaseURL != "https://api.example.com" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestBuildAliasConfig_ProviderRouter_DefaultsToCCRPlaceholder(t *testing.T) {
	if _, err := buildAliasConfig("provider-router", "", "", "", "", ""); err == nil {
		t.Fatal("expected error when --base-url is missing for kind=provider-router")
	}

	// No --api-key-env given: falls back to the documented non-secret
	// "ccr" placeholder (ccr's own front accepts any non-empty key).
	cfg, err := buildAliasConfig("provider-router", "", "http://127.0.0.1:3456", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AnthropicAPIKey != "ccr" {
		t.Fatalf("expected default ccr placeholder key, got %q", cfg.AnthropicAPIKey)
	}

	const envName = "TEST_CLAUDE_ALIAS_PROBE_APIKEY_UNSET"
	if _, err := buildAliasConfig("provider-router", "", "http://127.0.0.1:3456", "", envName, ""); err == nil {
		t.Fatal("expected error when the named api-key env var is set as a flag but unset in the environment")
	}
}

func TestBuildAliasConfig_UnknownKind(t *testing.T) {
	if _, err := buildAliasConfig("not-a-real-kind", "", "", "", "", ""); err == nil {
		t.Fatal("expected error for an unrecognised --kind value")
	}
}

func TestJudgeOutcome_SentinelMatched_IsOK_NotInfra(t *testing.T) {
	const sentinel = "SENTINEL_ABC123"
	rep := judgeOutcome("here is your token: SENTINEL_ABC123 done", nil, sentinel)
	if rep.Verdict != "ok" || !rep.SentinelMatched || rep.infra {
		t.Fatalf("unexpected report: %+v (infra=%v)", rep, rep.infra)
	}
}

func TestJudgeOutcome_SentinelMissing_IsFailed_NotInfra(t *testing.T) {
	// A completed call with a genuinely wrong reply is a real
	// determination, not an infra condition -- exit code 1, not 3.
	rep := judgeOutcome("I have no idea what you mean", nil, "SENTINEL_XYZ")
	if rep.Verdict != "failed" || rep.SentinelMatched || rep.infra {
		t.Fatalf("unexpected report: %+v (infra=%v)", rep, rep.infra)
	}
	if rep.Detail == "" {
		t.Fatal("expected a non-empty Detail explaining the sentinel mismatch")
	}
}

// TestJudgeOutcome_CappedAlias_RED_GREEN is the RED->GREEN regression
// test named by the PWU-3 task brief: a capped-alias fixture (an error
// wrapping providers.ErrClaudeCodeRateLimited / ErrClaudeCodeQuotaExceeded,
// the exact sentinels providers/claudecode.go returns for a real
// Fair-Usage/quota-exhausted CLI response) MUST emit a DISTINCT
// rate_limited/quota_exceeded verdict -- NEVER "ok", and NEVER folded
// into the generic opaque "failed" bucket that existed before PWU-2's
// verdict.go wiring (see providers/verdict.go's doc comment: the
// pre-PWU-2 RED state was "zero non-self callers" for ClassifyError, so
// a capped alias collapsed into an indistinguishable generic failure).
func TestJudgeOutcome_CappedAlias_RED_GREEN(t *testing.T) {
	cases := []struct {
		name         string
		err          error
		wantVerdict  string
		wantInfra    bool
		wantRetryVal int
	}{
		{
			name:        "rate_limited",
			err:         fmt.Errorf("%w: 429 fair usage policy", providers.ErrClaudeCodeRateLimited),
			wantVerdict: "rate_limited",
			wantInfra:   true,
		},
		{
			name:        "quota_exceeded",
			err:         fmt.Errorf("%w: weekly limit reached", providers.ErrClaudeCodeQuotaExceeded),
			wantVerdict: "quota_exceeded",
			wantInfra:   true,
		},
		{
			name:        "generic_failure_not_misclassified_as_capacity",
			err:         errors.New("claude code CLI failed: exit status 1 (stderr: unexpected auth error)"),
			wantVerdict: "failed",
			wantInfra:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rep := judgeOutcome("", tc.err, "IRRELEVANT_SENTINEL")
			// RED assertion: a capped alias must NEVER report "ok".
			if rep.Verdict == "ok" {
				t.Fatalf("PASS-BLUFF: capped/failing alias reported verdict=ok, report=%+v", rep)
			}
			if rep.Verdict != tc.wantVerdict {
				t.Fatalf("verdict = %q, want %q (report=%+v)", rep.Verdict, tc.wantVerdict, rep)
			}
			if rep.infra != tc.wantInfra {
				t.Fatalf("infra = %v, want %v (report=%+v)", rep.infra, tc.wantInfra, rep)
			}
		})
	}
}

func TestRun_ConfigErrors_Exit2(t *testing.T) {
	cases := [][]string{
		{}, // no flags at all: missing --sentinel
		{"--sentinel", "X"},                                    // missing --kind
		{"--kind", "native", "--sentinel", "X"},                // missing --config-dir
		{"--kind", "bogus", "--sentinel", "X"},                 // unknown kind
		{"--kind", "native", "--config-dir", "/x", "--sentinel", "X", "--timeout", "0"}, // bad timeout
		{"--kind", "native", "--config-dir", "/x", "--sentinel", "X", "--format", "yaml"}, // bad format
	}
	for i, args := range cases {
		var stdout, stderr bytes.Buffer
		got := run(args, &stdout, &stderr)
		if got != 2 {
			t.Fatalf("case %d (%v): exit code = %d, want 2; stderr=%s", i, args, got, stderr.String())
		}
		if stdout.Len() != 0 {
			t.Fatalf("case %d (%v): expected no stdout on a config error, got %q", i, args, stdout.String())
		}
	}
}

func TestFirstNChars_UTF8Bounded(t *testing.T) {
	s := strings.Repeat("é", 200) // multi-byte rune
	got := firstNChars(s, 10)
	if len([]rune(got)) != 10 {
		t.Fatalf("expected 10 runes, got %d (%q)", len([]rune(got)), got)
	}
}
