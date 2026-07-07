//go:build integration
// +build integration

package providers

// claudecode_integration_test.go drives a REAL `claude -p` invocation
// through the local claude-code-router (ccr) — a genuine §11.4.69
// sink-side probe, no mocks (§11.4.27(A) no-fakes-beyond-unit — this
// file is NOT a unit test, so it MUST exercise the real system). Two
// cases: a positive live-sentinel PASS, and a negative case proving a
// broken/unreachable route can NEVER be mistaken for a sentinel PASS
// (the exact anti-bluff property §11.4.69/§11.4.107 mandate).
//
// SKIP (never FAIL, per §11.4.3 topology dispatch) when the `claude`
// binary is absent or ccr is not reachable on 127.0.0.1:3456 — this is
// the correct behaviour for a host that has not installed the CLI or
// started the router, not a test failure.

import (
	"context"
	"errors"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// ccrBaseURL is the local claude-code-router front the incorporation
// plan (PART B.1) documents: Anthropic-protocol in, OpenAI-compatible
// backend out.
const ccrBaseURL = "http://127.0.0.1:3456"

func claudeCLIAvailable(t *testing.T) bool {
	t.Helper()
	_, err := exec.LookPath("claude")
	return err == nil
}

func ccrReachable(t *testing.T) bool {
	t.Helper()
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(ccrBaseURL + "/")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return true
}

// TestClaudeCodeCLIAdapter_ChatCompletion_LiveSentinel_ProviderRouter is
// the positive case: a real, minimal `claude -p` completion routed
// through ccr, judged by a deterministic sentinel substring — the same
// judging discipline
// cmd/semantic-code-visibility/main.go's round-1 uses (fetched read-only
// from origin/main; see the PWU-1 report for the fetch-first finding
// that this file is ahead of the checked-out submodule commit).
func TestClaudeCodeCLIAdapter_ChatCompletion_LiveSentinel_ProviderRouter(t *testing.T) {
	if !claudeCLIAvailable(t) {
		t.Skip("SKIP: claude CLI not found on PATH")
	}
	if !ccrReachable(t) {
		t.Skip("SKIP: claude-code-router (ccr) not reachable on " + ccrBaseURL)
	}

	alias := ClaudeCodeAliasConfig{
		Kind:             ClaudeCodeAliasProviderRouter,
		AnthropicBaseURL: ccrBaseURL,
		AnthropicAPIKey:  "ccr",
	}
	// Timeout matches NewClaudeCodeCLIAdapter's own natural default
	// (180s, mirrors kimicode.go) rather than an artificially tighter
	// value — this host runs other concurrent `claude -p` workers
	// sharing the same local ccr instance (§11.4.103 continuous
	// parallel-stream routine), so queueing latency on a shared router
	// is expected, not a code defect (§11.4.102: root-cause BEFORE
	// tightening any timeout further).
	adapter := NewClaudeCodeCLIAdapter(alias, 180*time.Second)

	const sentinel = "ATMOSPHERE_LLMSVERIFIER_PWU1_SENTINEL_7f3a91"
	ctx, cancel := context.WithTimeout(context.Background(), 190*time.Second)
	defer cancel()

	resp, err := adapter.ChatCompletion(ctx, OpenAIChatRequest{
		Messages: []Message{
			{Role: "user", Content: "Reply with EXACTLY this token and nothing else: " + sentinel},
		},
		MaxTokens: 32,
	})
	if err != nil {
		if errors.Is(err, ErrClaudeCodeRateLimited) {
			t.Skipf("SKIP: rate-limited/Fair-Usage-capped during live probe (infra, not a test failure): %v", err)
		}
		t.Fatalf("live ChatCompletion via ccr failed: %v", err)
	}
	if resp == nil || len(resp.Choices) == 0 {
		t.Fatalf("live ChatCompletion via ccr returned no choices (response=%+v)", resp)
	}

	content := resp.Choices[0].Message.Content
	if !strings.Contains(content, sentinel) {
		t.Fatalf("SENTINEL PASS-BLUFF GUARD TRIPPED: expected reply to contain %q, got: %q", sentinel, content)
	}

	t.Logf("PASS (real §11.4.69 sink-side probe via ccr): sentinel found in live reply=%q model=%q", firstNChars(content, 120), resp.Model)
}

// TestClaudeCodeCLIAdapter_ChatCompletion_NegativeCase_UnreachableRoute_NoFalsePass
// is the required NEGATIVE case: point the adapter at a port nothing
// listens on. `claude -p` MUST fail to connect, and the adapter MUST
// NOT report a sentinel-matching PASS. Absence of this test would leave
// the sentinel judge unproven against network failure — the exact class
// of PASS-bluff §11.4.69/§11.4.107 forbid.
func TestClaudeCodeCLIAdapter_ChatCompletion_NegativeCase_UnreachableRoute_NoFalsePass(t *testing.T) {
	if !claudeCLIAvailable(t) {
		t.Skip("SKIP: claude CLI not found on PATH")
	}

	alias := ClaudeCodeAliasConfig{
		Kind:             ClaudeCodeAliasProviderRouter,
		AnthropicBaseURL: "http://127.0.0.1:1", // reserved port; nothing listens here
		AnthropicAPIKey:  "ccr",
	}
	adapter := NewClaudeCodeCLIAdapter(alias, 20*time.Second)

	const sentinel = "ATMOSPHERE_LLMSVERIFIER_PWU1_NEGATIVE_SENTINEL_9c1d4e"
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	resp, err := adapter.ChatCompletion(ctx, OpenAIChatRequest{
		Messages: []Message{
			{Role: "user", Content: "Reply with EXACTLY: " + sentinel},
		},
	})

	if err == nil {
		content := ""
		if resp != nil && len(resp.Choices) > 0 {
			content = resp.Choices[0].Message.Content
		}
		if strings.Contains(content, sentinel) {
			t.Fatalf("FALSE-PASS DETECTED: an unreachable route produced a sentinel-matching reply: %q", content)
		}
		t.Fatalf("expected ChatCompletion to fail against an unreachable route (127.0.0.1:1), got success with content=%q", content)
	}

	t.Logf("PASS (negative case correctly rejected, no false-pass): error=%v", err)
}
