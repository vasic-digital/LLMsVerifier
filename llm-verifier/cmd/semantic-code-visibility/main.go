// Command semantic-code-visibility is a generic, consumer-agnostic verifier
// that probes whether an OpenAI-compatible chat model can actually SEE a piece
// of code shown to it, and (optionally) whether it can DESCRIBE that code well
// enough to satisfy an independent judge model.
//
// It is deliberately project-not-aware: it hardcodes no consumer project name,
// path, prompt, fixture, or sentinel. Everything consumer-specific is supplied
// at runtime through CLI flags (the code fixture, the prompt template, the
// sentinel token, the provider endpoint/model, and — for round 2 — the judge
// endpoint/model). This keeps it reusable as shared infrastructure by any
// number of unrelated consumer projects.
//
// It depends only on the Go standard library (flag, os, net/http,
// encoding/json, ...) — it does NOT import the verification/providers/database
// packages, so it builds without the module's cgo/sqlite surface.
//
// Behaviour:
//
//	Round 1 (sentinel): interpolate the prompt template ({{FIXTURE_CONTENT}} ->
//	fixture contents, {{SENTINEL}} -> sentinel), POST it to
//	{base-url}/v1/chat/completions, and pass iff the reply contains the exact
//	sentinel substring.
//
//	Round 2 (judge, optional): only attempted when round 1 passes AND a full
//	judge flag set (--judge-base-url/--judge-model/--judge-api-key-env) is given.
//	The model-under-test is asked to describe the code, then the judge model
//	scores that description 0-3; round 2 passes iff score >= --judge-threshold.
//	Without judge flags, round 2 is reported "skipped" (never "failed").
//
// Anti-bluff: a transport error, timeout, non-200 status, or empty body ALWAYS
// yields pass=false with a reason — never a false pass. The bearer token is
// read ONLY from the env var named by --api-key-env (never taken as a flag
// value, never echoed into argv or the JSON output).
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Generic (consumer-agnostic) defaults used only when the optional round-2
// flags are omitted. They name no consumer project and carry no branding.
const (
	defaultRound2Instruction = "Based on the code you were shown, describe in detail what it does, how it is structured, and its purpose."
	defaultJudgePrompt       = "You are grading whether a description accurately reflects some reference code.\n\n" +
		"Reference code:\n{{FIXTURE_CONTENT}}\n\n" +
		"Description to grade:\n{{DESCRIPTION}}\n\n" +
		"Rate how accurately the description reflects the reference code on an integer scale from 0 to 3 " +
		"(0 = unrelated, 1 = mostly wrong, 2 = mostly right, 3 = fully accurate). Reply with ONLY the single integer."

	maxResponseBytes = 1 << 20 // 1 MiB cap on any provider response body
)

// chatMessage / chatRequest / chatResponse model the minimal OpenAI-compatible
// chat-completions wire shapes this command needs.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model     string        `json:"model"`
	Messages  []chatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// round1Result is the sentinel-visibility outcome.
type round1Result struct {
	Pass     bool   `json:"pass"`
	Observed string `json:"observed"`
	Reason   string `json:"reason,omitempty"`
}

// round2Result is the judge outcome. Pass/Score are pointers so they serialize
// as JSON null when round 2 was not evaluated.
type round2Result struct {
	Pass    *bool  `json:"pass"`
	Score   *int   `json:"score"`
	Skipped bool   `json:"skipped"`
	Reason  string `json:"reason,omitempty"`
}

// report is the machine-readable output document.
type report struct {
	Round1  round1Result `json:"round1_sentinel"`
	Round2  round2Result `json:"round2_judge"`
	Overall bool         `json:"overall_pass"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run is the testable entry point. Exit codes: 0 overall pass, 1 verification
// fail, 2 usage/config error.
func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("semantic-code-visibility", flag.ContinueOnError)
	fs.SetOutput(stderr)

	baseURL := fs.String("base-url", "", "Provider chat endpoint base (e.g. https://api.deepseek.com)")
	model := fs.String("model", "", "Model id to send to the provider")
	apiKeyEnv := fs.String("api-key-env", "", "NAME of the env var holding the bearer token (read via os.Getenv; never the key itself)")
	fixturePath := fs.String("fixture", "", "Path to a file whose contents are the code to show the model")
	promptPath := fs.String("prompt", "", "Path to a prompt-template file containing {{FIXTURE_CONTENT}} and {{SENTINEL}}")
	sentinel := fs.String("sentinel", "", "Exact token to look for in the round-1 reply")
	timeoutSec := fs.Int("timeout", 60, "Per-request timeout in seconds")
	format := fs.String("format", "json", "Output format (only 'json' is supported)")

	round2PromptPath := fs.String("round2-prompt", "", "Optional path to a round-2 describe-prompt template (defaults to a generic instruction)")
	judgeBaseURL := fs.String("judge-base-url", "", "Optional judge provider chat endpoint base (enables round 2)")
	judgeModel := fs.String("judge-model", "", "Optional judge model id (enables round 2)")
	judgeKeyEnv := fs.String("judge-api-key-env", "", "Optional NAME of the env var holding the judge bearer token (enables round 2)")
	judgePromptPath := fs.String("judge-prompt", "", "Optional path to a judge scoring-prompt template with {{FIXTURE_CONTENT}} and {{DESCRIPTION}} (defaults to a generic rubric)")
	judgeThreshold := fs.Int("judge-threshold", 2, "Minimum judge score (0-3) required for round 2 to pass")

	if err := fs.Parse(args); err != nil {
		// flag already printed usage/err to stderr (includes -h ErrHelp).
		return 2
	}

	// --- config validation (exit 2 on any config/usage error) ---
	var missing []string
	for name, val := range map[string]string{
		"--base-url":    *baseURL,
		"--model":       *model,
		"--api-key-env": *apiKeyEnv,
		"--fixture":     *fixturePath,
		"--prompt":      *promptPath,
		"--sentinel":    *sentinel,
	} {
		if val == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		fmt.Fprintf(stderr, "config error: missing required flag(s): %s\n", strings.Join(missing, ", "))
		return 2
	}
	if *format != "json" {
		fmt.Fprintf(stderr, "config error: unsupported --format %q (only 'json' is supported)\n", *format)
		return 2
	}
	if *timeoutSec <= 0 {
		fmt.Fprintf(stderr, "config error: --timeout must be positive, got %d\n", *timeoutSec)
		return 2
	}

	apiKey := os.Getenv(*apiKeyEnv)
	if apiKey == "" {
		fmt.Fprintf(stderr, "config error: env var %q (from --api-key-env) is empty or unset\n", *apiKeyEnv)
		return 2
	}

	fixtureBytes, err := os.ReadFile(*fixturePath)
	if err != nil {
		fmt.Fprintf(stderr, "config error: cannot read --fixture: %v\n", err)
		return 2
	}
	promptBytes, err := os.ReadFile(*promptPath)
	if err != nil {
		fmt.Fprintf(stderr, "config error: cannot read --prompt: %v\n", err)
		return 2
	}

	// Judge flags are all-or-none. Any subset that is not the full set is a
	// config error rather than a silent partial-round-2.
	judgeAny := *judgeBaseURL != "" || *judgeModel != "" || *judgeKeyEnv != ""
	judgeAll := *judgeBaseURL != "" && *judgeModel != "" && *judgeKeyEnv != ""
	if judgeAny && !judgeAll {
		fmt.Fprintf(stderr, "config error: --judge-base-url, --judge-model and --judge-api-key-env must be set together (or all omitted)\n")
		return 2
	}

	var (
		judgeKey            string
		round2Instruction   = defaultRound2Instruction
		judgePromptTemplate = defaultJudgePrompt
	)
	if judgeAll {
		judgeKey = os.Getenv(*judgeKeyEnv)
		if judgeKey == "" {
			fmt.Fprintf(stderr, "config error: judge env var %q (from --judge-api-key-env) is empty or unset\n", *judgeKeyEnv)
			return 2
		}
		if *round2PromptPath != "" {
			b, rerr := os.ReadFile(*round2PromptPath)
			if rerr != nil {
				fmt.Fprintf(stderr, "config error: cannot read --round2-prompt: %v\n", rerr)
				return 2
			}
			round2Instruction = string(b)
		}
		if *judgePromptPath != "" {
			b, rerr := os.ReadFile(*judgePromptPath)
			if rerr != nil {
				fmt.Fprintf(stderr, "config error: cannot read --judge-prompt: %v\n", rerr)
				return 2
			}
			judgePromptTemplate = string(b)
		}
	}

	// --- build round-1 prompt ---
	round1Prompt := strings.ReplaceAll(string(promptBytes), "{{FIXTURE_CONTENT}}", string(fixtureBytes))
	round1Prompt = strings.ReplaceAll(round1Prompt, "{{SENTINEL}}", *sentinel)

	hc := &http.Client{Timeout: time.Duration(*timeoutSec) * time.Second}
	ctx := context.Background()

	rep := report{Round2: round2Result{Skipped: true}}

	// --- round 1 ---
	r1Content, err := chatComplete(ctx, hc, *baseURL, apiKey, *model, []chatMessage{
		{Role: "user", Content: round1Prompt},
	}, 256)
	if err != nil {
		// Anti-bluff: any failed/empty call is a fail with a reason, never a pass.
		rep.Round1 = round1Result{Pass: false, Observed: "", Reason: err.Error()}
	} else {
		rep.Round1 = round1Result{
			Pass:     strings.Contains(r1Content, *sentinel),
			Observed: firstNChars(r1Content, 80),
		}
		if !rep.Round1.Pass {
			rep.Round1.Reason = "sentinel not found in response"
		}
	}

	// --- round 2 (judge), only if enabled AND round 1 passed ---
	switch {
	case !judgeAll:
		rep.Round2 = round2Result{Skipped: true}
	case !rep.Round1.Pass:
		rep.Round2 = round2Result{Skipped: true, Reason: "round 1 did not pass; round 2 not attempted"}
	default:
		rep.Round2 = runRound2(ctx, hc, round2Params{
			modelBaseURL: *baseURL,
			modelKey:     apiKey,
			model:        *model,
			round1Prompt: round1Prompt,
			round1Reply:  r1Content,
			round2Instr:  interpolate(round2Instruction, string(fixtureBytes), *sentinel),
			judgeBaseURL: *judgeBaseURL,
			judgeKey:     judgeKey,
			judgeModel:   *judgeModel,
			judgePrompt:  judgePromptTemplate,
			fixture:      string(fixtureBytes),
			threshold:    *judgeThreshold,
		})
	}

	// --- overall verdict ---
	if rep.Round2.Skipped {
		rep.Overall = rep.Round1.Pass
	} else {
		rep.Overall = rep.Round1.Pass && rep.Round2.Pass != nil && *rep.Round2.Pass
	}

	out, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		fmt.Fprintf(stderr, "internal error: marshal report: %v\n", err)
		return 2
	}
	fmt.Fprintln(stdout, string(out))

	if rep.Overall {
		return 0
	}
	return 1
}

type round2Params struct {
	modelBaseURL string
	modelKey     string
	model        string
	round1Prompt string
	round1Reply  string
	round2Instr  string
	judgeBaseURL string
	judgeKey     string
	judgeModel   string
	judgePrompt  string
	fixture      string
	threshold    int
}

// runRound2 continues the round-1 conversation to obtain a description, then
// asks the judge model to score it. Every failure yields pass=false with a
// reason (anti-bluff) — never a null/pass on a broken call.
func runRound2(ctx context.Context, hc *http.Client, p round2Params) round2Result {
	fail := func(reason string) round2Result {
		f := false
		return round2Result{Pass: &f, Skipped: false, Reason: reason}
	}

	// Ask the model-under-test to describe the code (continued conversation).
	description, err := chatComplete(ctx, hc, p.modelBaseURL, p.modelKey, p.model, []chatMessage{
		{Role: "user", Content: p.round1Prompt},
		{Role: "assistant", Content: p.round1Reply},
		{Role: "user", Content: p.round2Instr},
	}, 512)
	if err != nil {
		return fail("round-2 model call failed: " + err.Error())
	}

	// Ask the judge to score the description.
	judgePrompt := strings.ReplaceAll(p.judgePrompt, "{{FIXTURE_CONTENT}}", p.fixture)
	judgePrompt = strings.ReplaceAll(judgePrompt, "{{DESCRIPTION}}", description)
	judgeReply, err := chatComplete(ctx, hc, p.judgeBaseURL, p.judgeKey, p.judgeModel, []chatMessage{
		{Role: "user", Content: judgePrompt},
	}, 64)
	if err != nil {
		return fail("judge call failed: " + err.Error())
	}

	score, ok := parseScore(judgeReply)
	if !ok {
		return fail("could not parse judge score from reply: " + firstNChars(judgeReply, 80))
	}

	pass := score >= p.threshold
	res := round2Result{Pass: &pass, Score: &score, Skipped: false}
	if !pass {
		res.Reason = fmt.Sprintf("judge score %d below threshold %d", score, p.threshold)
	}
	return res
}

// chatComplete POSTs an OpenAI-compatible chat completion to
// {baseURL}/v1/chat/completions with an Authorization: Bearer header and
// returns choices[0].message.content. It treats transport errors, non-200
// statuses, empty bodies, and empty content as errors (anti-bluff).
func chatComplete(ctx context.Context, hc *http.Client, baseURL, apiKey, model string, messages []chatMessage, maxTokens int) (string, error) {
	url := strings.TrimRight(baseURL, "/") + "/v1/chat/completions"
	payload, err := json.Marshal(chatRequest{Model: model, Messages: messages, MaxTokens: maxTokens})
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := hc.Do(req)
	if err != nil {
		return "", fmt.Errorf("http call failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", fmt.Errorf("read response body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non-200 status %d: %s", resp.StatusCode, firstNChars(string(body), 200))
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return "", fmt.Errorf("empty response body")
	}

	var cr chatResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return "", fmt.Errorf("decode response json: %w", err)
	}
	if len(cr.Choices) == 0 {
		return "", fmt.Errorf("response had no choices")
	}
	content := cr.Choices[0].Message.Content
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("response message content was empty")
	}
	return content, nil
}

// interpolate applies the same {{FIXTURE_CONTENT}}/{{SENTINEL}} substitutions
// used for the round-1 prompt (so a consumer-supplied round-2 template may
// reference them too).
func interpolate(tmpl, fixture, sentinel string) string {
	s := strings.ReplaceAll(tmpl, "{{FIXTURE_CONTENT}}", fixture)
	return strings.ReplaceAll(s, "{{SENTINEL}}", sentinel)
}

// parseScore extracts the first integer found in the judge reply.
func parseScore(s string) (int, bool) {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			continue
		}
		j := i
		for j < len(s) && s[j] >= '0' && s[j] <= '9' {
			j++
		}
		n, err := strconv.Atoi(s[i:j])
		if err != nil {
			return 0, false
		}
		return n, true
	}
	return 0, false
}

// firstNChars returns the first n runes of s (UTF-8 aware).
func firstNChars(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
