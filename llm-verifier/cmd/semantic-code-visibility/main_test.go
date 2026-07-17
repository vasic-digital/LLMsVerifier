package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
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

// TestRun_Round1 is table-driven over the anti-bluff scenarios plus the
// happy path, using a mock OpenAI-compatible server per case. It pins the
// frozen exit-code contract: 0 pass, 1 genuine negative determination
// (including definitive provider rejections HTTP 401/402/403/404 and prompt
// echo / bluff), 3 transient/infra (HTTP 429/5xx, empty body).
func TestRun_Round1(t *testing.T) {
	cases := []struct {
		name       string
		content    string // 200 body content (when status==200 && !emptyBody)
		status     int
		emptyBody  bool
		fixture    string // fixture content; empty -> default short fixture
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
			name:       "http 500 -> round1 infra error (exit 3, anti-bluff, no false pass)",
			status:     500,
			wantExit:   3,
			wantR1Pass: false,
			wantReason: true,
		},
		{
			name:       "empty 200 body -> round1 infra error (exit 3, anti-bluff, no false pass)",
			status:     200,
			emptyBody:  true,
			wantExit:   3,
			wantR1Pass: false,
			wantReason: true,
		},
		{
			name:       "http 401 -> definitive rejection, genuine fail (exit 1)",
			status:     401,
			wantExit:   1,
			wantR1Pass: false,
			wantReason: true,
		},
		{
			name:       "http 402 -> depleted credits, genuine fail (exit 1)",
			status:     402,
			wantExit:   1,
			wantR1Pass: false,
			wantReason: true,
		},
		{
			name:       "http 404 -> model not found, genuine fail (exit 1)",
			status:     404,
			wantExit:   1,
			wantR1Pass: false,
			wantReason: true,
		},
		{
			name:       "http 429 -> rate limit is transient infra (exit 3)",
			status:     429,
			wantExit:   3,
			wantR1Pass: false,
			wantReason: true,
		},
		{
			name: "prompt echo / bluff -> sentinel plus verbatim fixture slice, genuine fail (exit 1)",
			content: "package demo\n\nfunc Add(a, b int) int { return a + b }\n" +
				"func Sub(a, b int) int { return a - b }\n" + testSentinel,
			status: 200,
			fixture: "package demo\n\nfunc Add(a, b int) int { return a + b }\n" +
				"func Sub(a, b int) int { return a - b }\n",
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

			fixtureContent := tc.fixture
			if fixtureContent == "" {
				fixtureContent = "package demo\nfunc Add(a, b int) int { return a + b }\n"
			}

			dir := t.TempDir()
			fixture := writeTemp(t, dir, "fixture.txt", fixtureContent)
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

// TestRun_Round2JudgeInfraError verifies a judge server that cannot complete
// the call (HTTP 500) yields round2.pass=false AND exit 3 (infra/transport
// error) — never exit 1 (which is reserved for a completed call that yields a
// genuine negative verdict) and never a false pass.
func TestRun_Round2JudgeInfraError(t *testing.T) {
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

	// Judge returns HTTP 500 -> the judge call could not complete -> infra error.
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
	if exit != 3 {
		t.Fatalf("exit = %d, want 3 (round2 judge call could not complete)\nstdout=%s\nstderr=%s", exit, stdout.String(), stderr.String())
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

// TestRun_Round2JudgeGenuineLowScore verifies that a judge call which
// COMPLETES (HTTP 200, parseable score) but returns a score below threshold is
// a genuine determination -> exit 1, NOT exit 3. This is the precise boundary
// the exit-3 introduction must not blur: a completed call with a real
// negative verdict stays exit 1.
func TestRun_Round2JudgeGenuineLowScore(t *testing.T) {
	const sentinel = "TOK-LOW"

	modelSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req chatRequest
		_ = json.Unmarshal(body, &req)
		if len(req.Messages) >= 2 {
			chatCompletionJSON(t, w, "A vague, mostly-wrong description of the code.")
			return
		}
		chatCompletionJSON(t, w, "Visible: "+sentinel)
	}))
	defer modelSrv.Close()

	// Judge call COMPLETES (200, parseable) with a genuinely low score.
	judgeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chatCompletionJSON(t, w, "0")
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
		t.Fatalf("exit = %d, want 1 (completed call, genuine low score)\nstdout=%s\nstderr=%s", exit, stdout.String(), stderr.String())
	}

	var rep report
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		t.Fatalf("bad json: %v\nout=%s", err, stdout.String())
	}
	if rep.Round2.Pass == nil || *rep.Round2.Pass {
		t.Errorf("round2 must fail on a genuinely low score, got %v", rep.Round2.Pass)
	}
	if rep.Round2.Score == nil || *rep.Round2.Score != 0 {
		t.Errorf("round2 score should be 0, got %v", rep.Round2.Score)
	}
	if rep.Overall {
		t.Errorf("overall must be false")
	}
}

// TestRun_Round2ModelCallInfraError verifies that the round-2 model-under-test
// "describe" call failing to complete (HTTP 500 on the continued
// conversation) yields exit 3, distinguishing an infra failure occurring
// AFTER round 1 has already genuinely passed.
func TestRun_Round2ModelCallInfraError(t *testing.T) {
	const sentinel = "TOK-R2M"

	modelSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req chatRequest
		_ = json.Unmarshal(body, &req)
		if len(req.Messages) >= 2 {
			// Round-2 describe call: fail to complete.
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		chatCompletionJSON(t, w, "Visible: "+sentinel)
	}))
	defer modelSrv.Close()

	judgeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chatCompletionJSON(t, w, "3")
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
	if exit != 3 {
		t.Fatalf("exit = %d, want 3 (round2 model call could not complete)\nstdout=%s\nstderr=%s", exit, stdout.String(), stderr.String())
	}

	var rep report
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		t.Fatalf("bad json: %v\nout=%s", err, stdout.String())
	}
	if !rep.Round1.Pass {
		t.Errorf("round1 should still pass")
	}
	if rep.Round2.Pass == nil || *rep.Round2.Pass {
		t.Errorf("round2 must be a hard fail, got %v", rep.Round2.Pass)
	}
}

// TestRun_Round2JudgeDefinitiveRejection verifies the frozen invariant that
// ALL judge-call failures stay exit 3 regardless of status code: even a
// definitive rejection (HTTP 402 — depleted judge credits) must NEVER demote
// the model-under-test to exit 1.
func TestRun_Round2JudgeDefinitiveRejection(t *testing.T) {
	const sentinel = "TOK-J402"

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

	// Judge returns HTTP 402 -> still an infra error (exit 3), never exit 1.
	judgeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusPaymentRequired)
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
	if exit != 3 {
		t.Fatalf("exit = %d, want 3 (judge-call failures are ALWAYS infra, even HTTP 402)\nstdout=%s\nstderr=%s", exit, stdout.String(), stderr.String())
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

// TestRun_Round2ModelCallDefinitiveRejection verifies that a definitive
// provider rejection (HTTP 402) on the round-2 MODEL-UNDER-TEST describe call
// is a genuine negative determination -> exit 1, because depleted billing
// credits are a deterministic state, not transient infra.
func TestRun_Round2ModelCallDefinitiveRejection(t *testing.T) {
	const sentinel = "TOK-M402"

	modelSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req chatRequest
		_ = json.Unmarshal(body, &req)
		if len(req.Messages) >= 2 {
			// Round-2 describe call: definitive rejection.
			w.WriteHeader(http.StatusPaymentRequired)
			return
		}
		chatCompletionJSON(t, w, "Visible: "+sentinel)
	}))
	defer modelSrv.Close()

	judgeSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chatCompletionJSON(t, w, "3")
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
		t.Fatalf("exit = %d, want 1 (definitive provider rejection on the model-under-test call)\nstdout=%s\nstderr=%s", exit, stdout.String(), stderr.String())
	}

	var rep report
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		t.Fatalf("bad json: %v\nout=%s", err, stdout.String())
	}
	if !rep.Round1.Pass {
		t.Errorf("round1 should still pass")
	}
	if rep.Round2.Pass == nil || *rep.Round2.Pass {
		t.Errorf("round2 must be a hard fail, got %v", rep.Round2.Pass)
	}
	if rep.Overall {
		t.Errorf("overall must be false")
	}
}

// TestEchoBluffDetected pins the echo/bluff guard semantics: any >=60-rune
// verbatim slice of the (whitespace-normalized) fixture appearing in the
// response marks it as a prompt echo; anything shorter or absent does not.
func TestEchoBluffDetected(t *testing.T) {
	fixture := "package demo\n\nfunc Add(a, b int) int { return a + b }\nfunc Sub(a, b int) int { return a - b }\n"
	normFixture := normalizeWhitespace(fixture)
	if len([]rune(normFixture)) < echoBluffRunes+1 {
		t.Fatalf("test fixture must normalize to more than %d runes, got %d", echoBluffRunes, len([]rune(normFixture)))
	}
	overlap60 := string([]rune(normFixture)[:echoBluffRunes])
	overlap59 := string([]rune(normFixture)[:echoBluffRunes-1])

	cases := []struct {
		name     string
		response string
		fixture  string
		want     bool
	}{
		{
			name:     "verbatim echo of the whole fixture",
			response: fixture + " SENTINEL-XYZ-42",
			fixture:  fixture,
			want:     true,
		},
		{
			name:     "whitespace-reformatted echo is still caught",
			response: "package demo   func Add(a, b int) int { return a + b }   func Sub(a, b int) int { return a - b } SENTINEL-XYZ-42",
			fixture:  fixture,
			want:     true,
		},
		{
			name:     "exactly 60-rune overlap",
			response: overlap60,
			fixture:  fixture,
			want:     true,
		},
		{
			name:     "59-rune overlap is below the echo threshold",
			response: overlap59,
			fixture:  fixture,
			want:     false,
		},
		{
			name:     "no fixture overlap",
			response: "I can see the code. SENTINEL-XYZ-42",
			fixture:  fixture,
			want:     false,
		},
		{
			name:     "fixture shorter than 60 runes never triggers",
			response: "x := 1 x := 1 x := 1 SENTINEL-XYZ-42",
			fixture:  "x := 1\n",
			want:     false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := echoBluffDetected(tc.response, tc.fixture); got != tc.want {
				t.Errorf("echoBluffDetected(%q, %q) = %v, want %v", tc.response, tc.fixture, got, tc.want)
			}
		})
	}
}

// TestRun_Round1ConnectionRefused verifies a round-1 call that cannot even
// reach a server (connection refused on a closed port) is an infra error ->
// exit 3, exercising the network-error branch of chatComplete distinctly from
// the non-200 and empty-body branches already covered above.
func TestRun_Round1ConnectionRefused(t *testing.T) {
	// Open then immediately close a listener to obtain a port nothing is
	// listening on, guaranteeing a connection-refused error.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to allocate a port: %v", err)
	}
	closedAddr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("failed to close listener: %v", err)
	}

	dir := t.TempDir()
	fixture := writeTemp(t, dir, "fixture.txt", "code")
	prompt := writeTemp(t, dir, "prompt.txt", "{{FIXTURE_CONTENT}} {{SENTINEL}}")

	t.Setenv("SCV_TEST_KEY", "dummy-key-value")

	args := []string{
		"--base-url", "http://" + closedAddr,
		"--model", "test-model",
		"--api-key-env", "SCV_TEST_KEY",
		"--fixture", fixture,
		"--prompt", prompt,
		"--sentinel", testSentinel,
		"--timeout", "5",
	}

	var stdout, stderr bytes.Buffer
	exit := run(args, &stdout, &stderr)
	if exit != 3 {
		t.Fatalf("exit = %d, want 3 (connection refused is an infra error)\nstdout=%s\nstderr=%s", exit, stdout.String(), stderr.String())
	}

	var rep report
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		t.Fatalf("bad json: %v\nout=%s", err, stdout.String())
	}
	if rep.Round1.Pass {
		t.Errorf("round1 must not pass on connection refused")
	}
	if rep.Round1.Reason == "" {
		t.Errorf("expected a non-empty reason on connection failure")
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

// TestChatCompletionsURL pins the base-URL resolution rule: a base already
// ending in /chat/completions is used verbatim, a base ending in a version
// segment /v[0-9]+ gets /chat/completions appended (never a doubled /v1),
// anything else gets /v1/chat/completions appended.
func TestChatCompletionsURL(t *testing.T) {
	cases := []struct {
		name string
		base string
		want string
	}{
		{
			name: "bare host gets /v1/chat/completions",
			base: "https://api.deepseek.com",
			want: "https://api.deepseek.com/v1/chat/completions",
		},
		{
			name: "bare host with trailing slash",
			base: "https://api.deepseek.com/",
			want: "https://api.deepseek.com/v1/chat/completions",
		},
		{
			name: "/v1 base gets only /chat/completions",
			base: "https://api.deepseek.com/v1",
			want: "https://api.deepseek.com/v1/chat/completions",
		},
		{
			name: "/v1 base with trailing slash",
			base: "https://api.deepseek.com/v1/",
			want: "https://api.deepseek.com/v1/chat/completions",
		},
		{
			name: "/v4 versioned path (z.ai coding endpoint)",
			base: "https://api.z.ai/api/coding/paas/v4",
			want: "https://api.z.ai/api/coding/paas/v4/chat/completions",
		},
		{
			name: "/v12 multi-digit version segment",
			base: "https://example.com/api/v12",
			want: "https://example.com/api/v12/chat/completions",
		},
		{
			name: "base already ending in /chat/completions is used verbatim",
			base: "https://example.com/v1/chat/completions",
			want: "https://example.com/v1/chat/completions",
		},
		{
			name: "/chat/completions with trailing slash",
			base: "https://example.com/v1/chat/completions/",
			want: "https://example.com/v1/chat/completions",
		},
		{
			name: "unversioned path gets /v1/chat/completions",
			base: "https://example.com/api",
			want: "https://example.com/api/v1/chat/completions",
		},
		{
			name: "non-numeric version-ish suffix is not a version segment",
			base: "https://example.com/v1beta",
			want: "https://example.com/v1beta/v1/chat/completions",
		},
		{
			name: "bare /v without digits is not a version segment",
			base: "https://example.com/v",
			want: "https://example.com/v/v1/chat/completions",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := chatCompletionsURL(tc.base); got != tc.want {
				t.Errorf("chatCompletionsURL(%q) = %q, want %q", tc.base, got, tc.want)
			}
		})
	}
}
