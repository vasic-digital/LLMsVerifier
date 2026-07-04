package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTemp writes content to a temp file inside dir and returns its path.
func writeTemp(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return p
}

// chatCompletionJSON encodes a minimal OpenAI-compatible response body.
func chatCompletionJSON(t *testing.T, w http.ResponseWriter, content string) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"choices": []map[string]any{
			{"message": map[string]any{"role": "assistant", "content": content}},
		},
	})
}

const testSentinel = "SENTINEL-XYZ-42"

// TestRun_Round1 is table-driven over the three anti-bluff scenarios plus the
// happy path, using a mock OpenAI-compatible server per case.
func TestRun_Round1(t *testing.T) {
	cases := []struct {
		name       string
		content    string // 200 body content (when status==200 && !emptyBody)
		status     int
		emptyBody  bool
		wantExit   int
		wantR1Pass bool
		wantReason bool
	}{
		{
			name:       "sentinel present -> round1 pass, overall pass",
			content:    "Sure, here is the token you asked for: " + testSentinel + " — visible.",
			status:     200,
			wantExit:   0,
			wantR1Pass: true,
			wantReason: false,
		},
		{
			name:       "sentinel absent -> round1 fail",
			content:    "I am sorry, I cannot see any code in your message.",
			status:     200,
			wantExit:   1,
			wantR1Pass: false,
			wantReason: true,
		},
		{
			name:       "http 500 -> round1 fail (anti-bluff, no false pass)",
			status:     500,
			wantExit:   1,
			wantR1Pass: false,
			wantReason: true,
		},
		{
			name:       "empty 200 body -> round1 fail (anti-bluff, no false pass)",
			status:     200,
			emptyBody:  true,
			wantExit:   1,
			wantR1Pass: false,
			wantReason: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var sawAuth bool
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") == "Bearer dummy-key-value" {
					sawAuth = true
				}
				if tc.status != 200 {
					w.WriteHeader(tc.status)
					return
				}
				if tc.emptyBody {
					// 200 with an empty body.
					return
				}
				chatCompletionJSON(t, w, tc.content)
			}))
			defer srv.Close()

			dir := t.TempDir()
			fixture := writeTemp(t, dir, "fixture.txt", "package demo\nfunc Add(a, b int) int { return a + b }\n")
			prompt := writeTemp(t, dir, "prompt.txt",
				"Read this code:\n{{FIXTURE_CONTENT}}\nIf you can see it, reply with the token {{SENTINEL}}.")

			t.Setenv("SCV_TEST_KEY", "dummy-key-value")

			args := []string{
				"--base-url", srv.URL,
				"--model", "test-model",
				"--api-key-env", "SCV_TEST_KEY",
				"--fixture", fixture,
				"--prompt", prompt,
				"--sentinel", testSentinel,
				"--timeout", "10",
			}

			var stdout, stderr bytes.Buffer
			exit := run(args, &stdout, &stderr)

			if exit != tc.wantExit {
				t.Fatalf("exit = %d, want %d\nstdout=%s\nstderr=%s", exit, tc.wantExit, stdout.String(), stderr.String())
			}
			if !sawAuth {
				t.Errorf("server never saw the expected Bearer token from --api-key-env")
			}

			var rep report
			if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
				t.Fatalf("stdout is not valid JSON: %v\nout=%s", err, stdout.String())
			}
			if rep.Round1.Pass != tc.wantR1Pass {
				t.Errorf("round1.pass = %v, want %v", rep.Round1.Pass, tc.wantR1Pass)
			}
			if rep.Overall != tc.wantR1Pass {
				t.Errorf("overall_pass = %v, want %v (round2 skipped => equals round1)", rep.Overall, tc.wantR1Pass)
			}
			if !rep.Round2.Skipped {
				t.Errorf("round2 must be skipped when no judge flags are given")
			}
			if rep.Round2.Pass != nil || rep.Round2.Score != nil {
				t.Errorf("round2 pass/score must be null when skipped, got pass=%v score=%v", rep.Round2.Pass, rep.Round2.Score)
			}
			if tc.wantReason && rep.Round1.Reason == "" {
				t.Errorf("expected a non-empty round1 reason on failure, got empty")
			}
			if tc.wantR1Pass {
				if rep.Round1.Reason != "" {
					t.Errorf("unexpected reason on pass: %q", rep.Round1.Reason)
				}
				if !strings.Contains(rep.Round1.Observed, testSentinel) {
					t.Errorf("observed %q should contain the sentinel", rep.Round1.Observed)
				}
			}
		})
	}
}

// TestRun_Round2Judge exercises the round-2 judge path end-to-end with a mock
// model-under-test server and a mock judge server.
func TestRun_Round2Judge(t *testing.T) {
	const sentinel = "TOK-9"

	modelSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req chatRequest
		_ = json.Unmarshal(body, &req)
		// Round 1 sends one message; round 2 continues the conversation (>=3).
		if len(req.Messages) >= 2 {
			chatCompletionJSON(t, w, "This code defines an Add function that returns the sum of two integers.")
			return
		}
		chatCompletionJSON(t, w, "I can see the code. "+sentinel)
	}))
	defer modelSrv.Close()

	judgeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chatCompletionJSON(t, w, "3")
	}))
	defer judgeSrv.Close()

	dir := t.TempDir()
	fixture := writeTemp(t, dir, "fixture.txt", "func Add(a, b int) int { return a + b }\n")
	prompt := writeTemp(t, dir, "prompt.txt", "Code:\n{{FIXTURE_CONTENT}}\nReply with {{SENTINEL}} if visible.")

	t.Setenv("SCV_MODEL_KEY", "model-key")
	t.Setenv("SCV_JUDGE_KEY", "judge-key")

	args := []string{
		"--base-url", modelSrv.URL,
		"--model", "m",
		"--api-key-env", "SCV_MODEL_KEY",
		"--fixture", fixture,
		"--prompt", prompt,
		"--sentinel", sentinel,
		"--timeout", "10",
		"--judge-base-url", judgeSrv.URL,
		"--judge-model", "jm",
		"--judge-api-key-env", "SCV_JUDGE_KEY",
	}

	var stdout, stderr bytes.Buffer
	exit := run(args, &stdout, &stderr)
	if exit != 0 {
		t.Fatalf("exit = %d, want 0\nstdout=%s\nstderr=%s", exit, stdout.String(), stderr.String())
	}

	var rep report
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\nout=%s", err, stdout.String())
	}
	if !rep.Round1.Pass {
		t.Errorf("round1 should pass")
	}
	if rep.Round2.Skipped {
		t.Errorf("round2 must not be skipped when judge flags are given and round1 passed")
	}
	if rep.Round2.Pass == nil || !*rep.Round2.Pass {
		t.Errorf("round2 should pass, got %v", rep.Round2.Pass)
	}
	if rep.Round2.Score == nil || *rep.Round2.Score != 3 {
		t.Errorf("round2 score should be 3, got %v", rep.Round2.Score)
	}
	if !rep.Overall {
		t.Errorf("overall should pass")
	}
}

// TestRun_Round2JudgeAntiBluff verifies a failing judge server yields
// round2.pass=false (not skipped, not a false pass) while round1 still passed.
func TestRun_Round2JudgeAntiBluff(t *testing.T) {
	const sentinel = "TOK-Z"

	modelSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req chatRequest
		_ = json.Unmarshal(body, &req)
		if len(req.Messages) >= 2 {
			chatCompletionJSON(t, w, "A description of the code.")
			return
		}
		chatCompletionJSON(t, w, "Visible: "+sentinel)
	}))
	defer modelSrv.Close()

	// Judge returns HTTP 500 -> anti-bluff round2 fail.
	judgeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer judgeSrv.Close()

	dir := t.TempDir()
	fixture := writeTemp(t, dir, "fixture.txt", "x := 1\n")
	prompt := writeTemp(t, dir, "prompt.txt", "{{FIXTURE_CONTENT}} reply {{SENTINEL}}")

	t.Setenv("SCV_MODEL_KEY", "model-key")
	t.Setenv("SCV_JUDGE_KEY", "judge-key")

	args := []string{
		"--base-url", modelSrv.URL, "--model", "m", "--api-key-env", "SCV_MODEL_KEY",
		"--fixture", fixture, "--prompt", prompt, "--sentinel", sentinel, "--timeout", "10",
		"--judge-base-url", judgeSrv.URL, "--judge-model", "jm", "--judge-api-key-env", "SCV_JUDGE_KEY",
	}

	var stdout, stderr bytes.Buffer
	exit := run(args, &stdout, &stderr)
	if exit != 1 {
		t.Fatalf("exit = %d, want 1 (round2 judge failed)\nstdout=%s\nstderr=%s", exit, stdout.String(), stderr.String())
	}

	var rep report
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		t.Fatalf("bad json: %v\nout=%s", err, stdout.String())
	}
	if !rep.Round1.Pass {
		t.Errorf("round1 should still pass")
	}
	if rep.Round2.Skipped {
		t.Errorf("round2 must not be skipped when judge flags given")
	}
	if rep.Round2.Pass == nil || *rep.Round2.Pass {
		t.Errorf("round2 must be a hard fail on judge error, got %v", rep.Round2.Pass)
	}
	if rep.Round2.Reason == "" {
		t.Errorf("expected a reason on round2 failure")
	}
	if rep.Overall {
		t.Errorf("overall must be false when round2 fails")
	}
}

// TestRun_ConfigErrors verifies usage/config problems exit 2 (never a false pass).
func TestRun_ConfigErrors(t *testing.T) {
	dir := t.TempDir()
	fixture := writeTemp(t, dir, "fixture.txt", "code")
	prompt := writeTemp(t, dir, "prompt.txt", "{{FIXTURE_CONTENT}} {{SENTINEL}}")

	cases := []struct {
		name string
		args []string
		env  map[string]string
	}{
		{
			name: "no args",
			args: []string{},
		},
		{
			name: "missing base-url",
			args: []string{
				"--model", "m", "--api-key-env", "SCV_K",
				"--fixture", fixture, "--prompt", prompt, "--sentinel", "S",
			},
			env: map[string]string{"SCV_K": "k"},
		},
		{
			name: "env var unset -> empty key",
			args: []string{
				"--base-url", "http://127.0.0.1:0", "--model", "m", "--api-key-env", "SCV_DEFINITELY_UNSET",
				"--fixture", fixture, "--prompt", prompt, "--sentinel", "S",
			},
		},
		{
			name: "unsupported format",
			args: []string{
				"--base-url", "http://127.0.0.1:0", "--model", "m", "--api-key-env", "SCV_K",
				"--fixture", fixture, "--prompt", prompt, "--sentinel", "S", "--format", "csv",
			},
			env: map[string]string{"SCV_K": "k"},
		},
		{
			name: "partial judge flags",
			args: []string{
				"--base-url", "http://127.0.0.1:0", "--model", "m", "--api-key-env", "SCV_K",
				"--fixture", fixture, "--prompt", prompt, "--sentinel", "S",
				"--judge-base-url", "http://127.0.0.1:0", // model + key missing
			},
			env: map[string]string{"SCV_K": "k"},
		},
		{
			name: "unreadable fixture",
			args: []string{
				"--base-url", "http://127.0.0.1:0", "--model", "m", "--api-key-env", "SCV_K",
				"--fixture", filepath.Join(dir, "does-not-exist"), "--prompt", prompt, "--sentinel", "S",
			},
			env: map[string]string{"SCV_K": "k"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			var stdout, stderr bytes.Buffer
			exit := run(tc.args, &stdout, &stderr)
			if exit != 2 {
				t.Fatalf("exit = %d, want 2\nstdout=%s\nstderr=%s", exit, stdout.String(), stderr.String())
			}
			if stderr.Len() == 0 {
				t.Errorf("expected a config-error message on stderr")
			}
		})
	}
}

func TestParseScore(t *testing.T) {
	cases := []struct {
		in     string
		want   int
		wantOK bool
	}{
		{"3", 3, true},
		{"Score: 2", 2, true},
		{"The rating is 0 out of 3.", 0, true},
		{"no digits here", 0, false},
		{"", 0, false},
	}
	for _, tc := range cases {
		got, ok := parseScore(tc.in)
		if ok != tc.wantOK || (ok && got != tc.want) {
			t.Errorf("parseScore(%q) = (%d,%v), want (%d,%v)", tc.in, got, ok, tc.want, tc.wantOK)
		}
	}
}
