package benchmark

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Round-78 §11.4 anti-bluff: HTTPBenchmarkProvider tests
//
// These tests are the load-bearing anti-bluff guard for the
// HTTPBenchmarkProvider concrete implementation introduced in
// http_provider.go. They prove:
//   (1) construction-time sentinels fire when Endpoint / Model are empty,
//   (2) a successful round-trip returns the REAL message content from the
//       mock server (not a hardcoded string),
//   (3) tokens come from the server's usage.total_tokens (not the
//       round-28 fabricated 50),
//   (4) errors are surfaced as wrapped sentinels (HTTP non-OK, invalid
//       JSON, embedded error),
//   (5) context cancellation aborts the in-flight request,
//   (6) paired-mutation: two DIFFERENT mock responses produce DIFFERENT
//       tokens — proving the value is NOT fabricated.
//
// Article XI §11.9 / CONST-035 / CONST-050(A)+(B) anchors apply.
// ============================================================================

// ----------------------------------------------------------------------------
// Construction-time sentinels
// ----------------------------------------------------------------------------

func TestHTTPBenchmarkProvider_NoEndpoint_ReturnsSentinel(t *testing.T) {
	provider, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: "",
		Model:    "gpt-4o-mini",
	})

	require.Error(t, err)
	require.Nil(t, provider)
	assert.True(t, errors.Is(err, ErrHTTPBenchmarkProviderEndpointNotConfigured),
		"empty Endpoint must return ErrHTTPBenchmarkProviderEndpointNotConfigured (got %v)", err)
}

func TestHTTPBenchmarkProvider_NoModel_ReturnsSentinel(t *testing.T) {
	provider, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: "http://localhost:8080/v1",
		Model:    "",
	})

	require.Error(t, err)
	require.Nil(t, provider)
	assert.True(t, errors.Is(err, ErrHTTPBenchmarkProviderModelNotConfigured),
		"empty Model must return ErrHTTPBenchmarkProviderModelNotConfigured (got %v)", err)
}

func TestHTTPBenchmarkProvider_WhitespaceOnlyEndpoint_ReturnsSentinel(t *testing.T) {
	provider, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: "   ",
		Model:    "gpt-4o-mini",
	})

	require.Error(t, err)
	require.Nil(t, provider)
	assert.True(t, errors.Is(err, ErrHTTPBenchmarkProviderEndpointNotConfigured))
}

// Sentinels MUST be distinct (errors.Is must NOT match cross-sentinels).
// Paired-mutation guard against accidental aliasing.
func TestHTTPBenchmarkProvider_SentinelsDistinctFromRound28(t *testing.T) {
	assert.False(t,
		errors.Is(ErrHTTPBenchmarkProviderEndpointNotConfigured, ErrBenchmarkProviderNotConfigured),
		"round-78 endpoint sentinel MUST be distinct from round-28 not-configured sentinel")
	assert.False(t,
		errors.Is(ErrHTTPBenchmarkProviderModelNotConfigured, ErrBenchmarkProviderNotConfigured),
		"round-78 model sentinel MUST be distinct from round-28 not-configured sentinel")
	assert.False(t,
		errors.Is(ErrHTTPBenchmarkProviderRequestFailed, ErrBenchmarkProviderNotConfigured),
		"round-78 request-failed sentinel MUST be distinct from round-28 not-configured sentinel")
	assert.False(t,
		errors.Is(ErrHTTPBenchmarkProviderResponseInvalid, ErrBenchmarkProviderNotConfigured),
		"round-78 response-invalid sentinel MUST be distinct from round-28 not-configured sentinel")
	// And the four round-78 sentinels are distinct from each other.
	assert.False(t,
		errors.Is(ErrHTTPBenchmarkProviderEndpointNotConfigured, ErrHTTPBenchmarkProviderModelNotConfigured))
	assert.False(t,
		errors.Is(ErrHTTPBenchmarkProviderRequestFailed, ErrHTTPBenchmarkProviderResponseInvalid))
}

func TestHTTPBenchmarkProvider_GetName_DefaultAndCustom(t *testing.T) {
	defaultProv, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: "http://localhost:8080/v1",
		Model:    "gpt-4o-mini",
	})
	require.NoError(t, err)
	assert.Equal(t, "http-benchmark-provider", defaultProv.GetName())

	customProv, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: "http://localhost:8080/v1",
		Model:    "gpt-4o-mini",
		Name:     "openai-prod",
	})
	require.NoError(t, err)
	assert.Equal(t, "openai-prod", customProv.GetName())
}

// ----------------------------------------------------------------------------
// Successful round-trip: real content + real tokens from response
// ----------------------------------------------------------------------------

// fakeChatCompletionsServer is a unit-test-only httptest server. Permitted
// under CONST-050(A): in-process loopback for unit-test scope; production
// code never imports this helper.
func fakeChatCompletionsServer(t *testing.T, content string, totalTokens int, latency time.Duration) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if latency > 0 {
			time.Sleep(latency)
		}
		// Validate the request shape so a regression in client-side
		// serialisation is caught immediately.
		if r.URL.Path != "/chat/completions" {
			http.Error(w, fmt.Sprintf("unexpected path %s", r.URL.Path), http.StatusNotFound)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, fmt.Sprintf("unexpected method %s", r.Method), http.StatusMethodNotAllowed)
			return
		}
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "could not read body", http.StatusBadRequest)
			return
		}
		var parsed map[string]any
		if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
			http.Error(w, "could not parse body", http.StatusBadRequest)
			return
		}
		if parsed["model"] == nil || parsed["messages"] == nil {
			http.Error(w, "missing model or messages", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{
					"message": map[string]any{
						"role":    "assistant",
						"content": content,
					},
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     7,
				"completion_tokens": totalTokens - 7,
				"total_tokens":      totalTokens,
			},
		})
	}))
}

func TestHTTPBenchmarkProvider_Execute_SuccessfulRoundtrip(t *testing.T) {
	server := fakeChatCompletionsServer(t, "the answer is 42", 38, 0)
	defer server.Close()

	provider, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: server.URL,
		Model:    "test-model",
		APIKey:   "sk-test-not-real",
		Timeout:  5 * time.Second,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	content, tokens, err := provider.Complete(ctx, "what is the answer to life?", "")
	require.NoError(t, err)

	// Anti-bluff: assert the content came from the mock, not a fabricated string.
	assert.Equal(t, "the answer is 42", content,
		"content MUST come from server response body, not a fabricated string")
	assert.Equal(t, 38, tokens,
		"tokens MUST come from server.usage.total_tokens, not the round-28 fabricated 50")
}

// ----------------------------------------------------------------------------
// Real-latency proof: timing IS measured, not fabricated.
//
// Round-28 bluff fabricated Latency=100ms regardless of input. This test
// stands up a slow mock server and asserts the wall-clock duration of
// Complete() reflects the server's induced delay — proving the measurement
// is real. Because HTTPBenchmarkProvider itself doesn't return Latency
// (that's measured by runner.executeTask via time.Since around Complete),
// we measure here at the caller layer — exactly the same pattern the runner
// uses.
// ----------------------------------------------------------------------------

func TestHTTPBenchmarkProvider_Execute_LatencyIsReal(t *testing.T) {
	const inducedDelay = 100 * time.Millisecond
	server := fakeChatCompletionsServer(t, "ok", 5, inducedDelay)
	defer server.Close()

	provider, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: server.URL,
		Model:    "test-model",
		Timeout:  5 * time.Second,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	start := time.Now()
	_, _, err = provider.Complete(ctx, "ping", "")
	elapsed := time.Since(start)
	require.NoError(t, err)

	// Allow some scheduling slack but the elapsed time MUST be >= inducedDelay
	// minus a small margin. The round-28 fabricated 100ms could have been any
	// value; this test proves the measurement reflects real behaviour.
	assert.GreaterOrEqual(t, elapsed, inducedDelay-10*time.Millisecond,
		"measured latency (%s) MUST reflect server's induced delay (%s) — round-28 bluff fabricated 100ms regardless of input",
		elapsed, inducedDelay)
}

// ----------------------------------------------------------------------------
// Tokens-from-response proof: usage.total_tokens flows through unchanged.
//
// Round-28 bluff fabricated TokensUsed=50 regardless of input. Asserts the
// returned tokens equal a specific server-reported value (42) NOT the
// fabricated 50.
// ----------------------------------------------------------------------------

func TestHTTPBenchmarkProvider_Execute_TokensUsedFromResponse(t *testing.T) {
	const serverReportedTokens = 42
	server := fakeChatCompletionsServer(t, "response body", serverReportedTokens, 0)
	defer server.Close()

	provider, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: server.URL,
		Model:    "test-model",
		Timeout:  5 * time.Second,
	})
	require.NoError(t, err)

	_, tokens, err := provider.Complete(context.Background(), "test", "")
	require.NoError(t, err)

	assert.Equal(t, serverReportedTokens, tokens,
		"tokens MUST equal server.usage.total_tokens (%d), not the round-28 fabricated 50",
		serverReportedTokens)
	assert.NotEqual(t, 50, tokens,
		"tokens MUST NOT equal the round-28 fabricated constant 50")
}

// ----------------------------------------------------------------------------
// Paired-mutation anti-bluff guard: two different mock responses MUST
// produce two different returned values. If the implementation regressed
// to fabricating a constant, this test would fail.
// ----------------------------------------------------------------------------

func TestHTTPBenchmarkProvider_NotFabricated(t *testing.T) {
	server1 := fakeChatCompletionsServer(t, "first answer", 13, 0)
	defer server1.Close()
	server2 := fakeChatCompletionsServer(t, "second longer answer", 27, 0)
	defer server2.Close()

	provider1, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: server1.URL,
		Model:    "test-model",
		Timeout:  5 * time.Second,
	})
	require.NoError(t, err)
	provider2, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: server2.URL,
		Model:    "test-model",
		Timeout:  5 * time.Second,
	})
	require.NoError(t, err)

	content1, tokens1, err := provider1.Complete(context.Background(), "q", "")
	require.NoError(t, err)
	content2, tokens2, err := provider2.Complete(context.Background(), "q", "")
	require.NoError(t, err)

	// Paired-mutation: DIFFERENT inputs MUST produce DIFFERENT outputs.
	// A fabricated implementation would return identical values for both
	// servers regardless of what they reported.
	assert.NotEqual(t, content1, content2,
		"two different mock servers MUST produce two different contents — fabrication guard")
	assert.NotEqual(t, tokens1, tokens2,
		"two different mock servers MUST produce two different token counts — fabrication guard")
	assert.Equal(t, "first answer", content1)
	assert.Equal(t, "second longer answer", content2)
	assert.Equal(t, 13, tokens1)
	assert.Equal(t, 27, tokens2)
}

// ----------------------------------------------------------------------------
// Error-path coverage
// ----------------------------------------------------------------------------

func TestHTTPBenchmarkProvider_Execute_HTTPNonOK_ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid api key"}}`))
	}))
	defer server.Close()

	provider, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: server.URL,
		Model:    "test-model",
		Timeout:  5 * time.Second,
	})
	require.NoError(t, err)

	content, tokens, err := provider.Complete(context.Background(), "q", "")
	require.Error(t, err)
	assert.Empty(t, content)
	assert.Zero(t, tokens)
	assert.True(t, errors.Is(err, ErrHTTPBenchmarkProviderRequestFailed),
		"401 must wrap ErrHTTPBenchmarkProviderRequestFailed (got %v)", err)
	// CONST-042: error message MUST NOT leak APIKey (we didn't set one,
	// but check the error doesn't accidentally serialise one).
	assert.NotContains(t, err.Error(), "sk-",
		"error message MUST NOT include any API key material")
}

func TestHTTPBenchmarkProvider_Execute_InvalidJSON_ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not valid json {{{`))
	}))
	defer server.Close()

	provider, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: server.URL,
		Model:    "test-model",
		Timeout:  5 * time.Second,
	})
	require.NoError(t, err)

	content, tokens, err := provider.Complete(context.Background(), "q", "")
	require.Error(t, err)
	assert.Empty(t, content)
	assert.Zero(t, tokens)
	assert.True(t, errors.Is(err, ErrHTTPBenchmarkProviderResponseInvalid),
		"invalid JSON must wrap ErrHTTPBenchmarkProviderResponseInvalid (got %v)", err)
}

func TestHTTPBenchmarkProvider_Execute_EmptyChoices_ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()

	provider, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: server.URL,
		Model:    "test-model",
		Timeout:  5 * time.Second,
	})
	require.NoError(t, err)

	_, _, err = provider.Complete(context.Background(), "q", "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrHTTPBenchmarkProviderResponseInvalid),
		"empty choices must wrap ErrHTTPBenchmarkProviderResponseInvalid (got %v)", err)
}

func TestHTTPBenchmarkProvider_Execute_EmbeddedError_ReturnsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}],"error":{"message":"rate limited","type":"rate_limit","code":"429"}}`))
	}))
	defer server.Close()

	provider, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: server.URL,
		Model:    "test-model",
		Timeout:  5 * time.Second,
	})
	require.NoError(t, err)

	_, _, err = provider.Complete(context.Background(), "q", "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrHTTPBenchmarkProviderResponseInvalid),
		"embedded error must wrap ErrHTTPBenchmarkProviderResponseInvalid (got %v)", err)
	assert.Contains(t, err.Error(), "rate limited")
}

// ----------------------------------------------------------------------------
// Context cancellation
// ----------------------------------------------------------------------------

func TestHTTPBenchmarkProvider_Execute_HonoursContextCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hold the request open long enough for the test to cancel
		// the context.
		select {
		case <-r.Context().Done():
			return
		case <-time.After(5 * time.Second):
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"too late"}}]}`))
	}))
	defer server.Close()

	provider, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: server.URL,
		Model:    "test-model",
		Timeout:  10 * time.Second,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, _, err = provider.Complete(ctx, "q", "")
	elapsed := time.Since(start)

	require.Error(t, err, "cancelled context must surface an error")
	assert.True(t, errors.Is(err, ErrHTTPBenchmarkProviderRequestFailed),
		"context cancel must wrap ErrHTTPBenchmarkProviderRequestFailed (got %v)", err)
	// Cancellation must abort BEFORE the 5s server delay completes.
	assert.Less(t, elapsed, 2*time.Second,
		"Complete MUST honour ctx cancel (elapsed=%s should be <2s, not the server's 5s)", elapsed)
}

// ----------------------------------------------------------------------------
// Trailing-slash endpoint normalisation
// ----------------------------------------------------------------------------

func TestHTTPBenchmarkProvider_Execute_TrailingSlashEndpointWorks(t *testing.T) {
	server := fakeChatCompletionsServer(t, "ok", 5, 0)
	defer server.Close()

	provider, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: server.URL + "/", // trailing slash
		Model:    "test-model",
		Timeout:  5 * time.Second,
	})
	require.NoError(t, err)

	content, _, err := provider.Complete(context.Background(), "q", "")
	require.NoError(t, err)
	assert.Equal(t, "ok", content,
		"endpoint with trailing slash must still resolve /chat/completions correctly")
}

// ----------------------------------------------------------------------------
// SystemPrompt propagation
// ----------------------------------------------------------------------------

func TestHTTPBenchmarkProvider_Execute_SystemPromptSentWhenNonEmpty(t *testing.T) {
	var capturedBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}],"usage":{"total_tokens":3}}`))
	}))
	defer server.Close()

	provider, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: server.URL,
		Model:    "test-model",
		Timeout:  5 * time.Second,
	})
	require.NoError(t, err)

	_, _, err = provider.Complete(context.Background(), "user msg", "you are a helpful assistant")
	require.NoError(t, err)

	bodyStr := string(capturedBody)
	assert.Contains(t, bodyStr, "you are a helpful assistant",
		"systemPrompt MUST be propagated as a system-role message when non-empty")
	assert.Contains(t, bodyStr, "user msg",
		"prompt MUST be propagated as a user-role message")
	assert.Contains(t, bodyStr, `"role":"system"`,
		"non-empty systemPrompt MUST produce a system-role message")
	assert.Contains(t, bodyStr, `"role":"user"`,
		"prompt MUST always produce a user-role message")
}

// ----------------------------------------------------------------------------
// CONST-042: APIKey is sent as Bearer but never leaked in errors
// ----------------------------------------------------------------------------

func TestHTTPBenchmarkProvider_Execute_APIKeySentAsBearerButNeverLeaked(t *testing.T) {
	const apiKey = "sk-secret-must-not-leak"
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		// Trigger an error path to check the error message does NOT
		// contain the key.
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"upstream failed"}`))
	}))
	defer server.Close()

	provider, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: server.URL,
		Model:    "test-model",
		APIKey:   apiKey,
		Timeout:  5 * time.Second,
	})
	require.NoError(t, err)

	_, _, err = provider.Complete(context.Background(), "q", "")
	require.Error(t, err)

	// Bearer header MUST have been set with the key.
	assert.Equal(t, "Bearer "+apiKey, capturedAuth,
		"APIKey must be sent as Authorization: Bearer <key>")
	// Error message MUST NOT contain the key (CONST-042).
	assert.NotContains(t, err.Error(), apiKey,
		"error message MUST NOT include API key material (CONST-042)")
	assert.NotContains(t, err.Error(), "sk-secret",
		"error message MUST NOT include any fragment of API key material")
}

// When APIKey is empty (local llama.cpp / Ollama), no Authorization header
// is sent.
func TestHTTPBenchmarkProvider_Execute_NoAuthHeaderWhenAPIKeyEmpty(t *testing.T) {
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"ok"}}],"usage":{"total_tokens":3}}`))
	}))
	defer server.Close()

	provider, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: server.URL,
		Model:    "test-model",
		APIKey:   "",
		Timeout:  5 * time.Second,
	})
	require.NoError(t, err)

	_, _, err = provider.Complete(context.Background(), "q", "")
	require.NoError(t, err)

	assert.Empty(t, capturedAuth,
		"empty APIKey MUST NOT produce an Authorization header (local-server use case)")
}

// ----------------------------------------------------------------------------
// Wired-into-runner: end-to-end proof that HTTPBenchmarkProvider closes the
// round-28 ErrBenchmarkProviderNotConfigured gap when wired via SetProvider.
//
// This is the load-bearing integration between round-78 and round-28: round
// 28 made the nil-provider runner surface a sentinel; round 78 provides a
// concrete provider the runner accepts via SetProvider so executeTask
// produces REAL evidence-bearing results instead of the sentinel.
// ----------------------------------------------------------------------------

func TestHTTPBenchmarkProvider_WiredIntoRunner_ProducesRealResult(t *testing.T) {
	server := fakeChatCompletionsServer(t, "B", 11, 0)
	defer server.Close()

	provider, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: server.URL,
		Model:    "test-model",
		Timeout:  5 * time.Second,
	})
	require.NoError(t, err)

	runner := NewStandardBenchmarkRunner(nil, nil)
	runner.SetProvider(provider)

	task := &BenchmarkTask{
		ID:       "mmlu-test",
		Prompt:   "Which option? A) wrong B) right C) other D) none",
		Expected: "B",
	}

	result := runner.executeTask(context.Background(), &BenchmarkRun{}, task)
	require.NotNil(t, result)

	// Anti-bluff: result.Response MUST be the mock server's content, not
	// a fabricated string.
	assert.Equal(t, "B", result.Response,
		"runner result Response MUST be the real server response, not fabricated")
	// TokensUsed MUST be the server's usage.total_tokens, NOT the round-28
	// fabricated 50.
	assert.Equal(t, 11, result.TokensUsed,
		"runner result TokensUsed MUST be the server-reported value, not round-28 fabricated 50")
	assert.NotEqual(t, 50, result.TokensUsed,
		"runner result TokensUsed MUST NOT equal round-28 fabricated constant 50")
	// Latency MUST be a real wall-clock measurement (> 0).
	assert.Greater(t, result.Latency, time.Duration(0),
		"runner result Latency MUST be a real measurement (>0)")
	// Task expected "B"; mock returned "B"; should pass.
	assert.True(t, result.Passed,
		"task expecting 'B' with response 'B' MUST pass")
	// Error field MUST be empty (no round-28 sentinel surfaced).
	assert.Empty(t, result.Error,
		"successful runner result MUST NOT carry round-28 ErrBenchmarkProviderNotConfigured")
	assert.NotContains(t, result.Error, "provider has not been wired",
		"round-28 sentinel MUST NOT fire when round-78 HTTPBenchmarkProvider is wired")
}

// ----------------------------------------------------------------------------
// Integration test against a real OpenAI-compatible endpoint
//
// Gated by LLMSVERIFIER_TEST_OPENAI_KEY env var. Default = SKIP-OK per
// §11.4.20 marker discipline. When the operator sets the env var, this test
// hits the real provider and asserts a real model response — satisfying
// CONST-050(B) integration-tier coverage.
// ----------------------------------------------------------------------------

func TestHTTPBenchmarkProvider_RealOpenAI(t *testing.T) {
	key := os.Getenv("LLMSVERIFIER_TEST_OPENAI_KEY")
	if strings.TrimSpace(key) == "" {
		t.Skip("SKIP-OK: #LLMSVERIFIER-BENCHMARK-REAL-ROUND78 — set LLMSVERIFIER_TEST_OPENAI_KEY=<sk-...> to exercise real OpenAI endpoint")
	}

	endpoint := os.Getenv("LLMSVERIFIER_TEST_OPENAI_ENDPOINT")
	if strings.TrimSpace(endpoint) == "" {
		endpoint = "https://api.openai.com/v1"
	}
	model := os.Getenv("LLMSVERIFIER_TEST_OPENAI_MODEL")
	if strings.TrimSpace(model) == "" {
		model = "gpt-4o-mini"
	}

	provider, err := NewHTTPBenchmarkProvider(HTTPBenchmarkProviderConfig{
		Endpoint: endpoint,
		Model:    model,
		APIKey:   key,
		Timeout:  60 * time.Second,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	content, tokens, err := provider.Complete(ctx, "What is 2+2? Reply with only the digit.", "")
	require.NoError(t, err, "real OpenAI call failed — check endpoint/model/key")

	t.Logf("real OpenAI roundtrip: content=%q tokens=%d", content, tokens)
	assert.NotEmpty(t, content, "real OpenAI response must have non-empty content")
	assert.Greater(t, tokens, 0, "real OpenAI response must report >0 tokens")
}
