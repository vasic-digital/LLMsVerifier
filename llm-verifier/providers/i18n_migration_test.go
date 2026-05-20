package providers

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// fakeProvidersTranslator returns "<TRANSLATED:msg_id>" so tests can assert
// the i18n-routed sentinel without coupling to the English bundle text.
// Anti-bluff per CONST-035 / Article XI §11.9: a test asserting the original
// literal would silently pass if the call-site bypassed the translator.
type fakeProvidersTranslator struct{}

func (fakeProvidersTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}
func (fakeProvidersTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeProvidersTranslator installs the fakeProvidersTranslator, runs fn,
// then restores the prior translator.
func withFakeProvidersTranslator(t *testing.T, fn func()) {
	t.Helper()
	prior := translator
	translator = fakeProvidersTranslator{}
	defer func() { translator = prior }()
	fn()
}

// newResp builds a minimal *http.Response with the given status code so the
// classifier exercises its real status-code switch — not a mock.
func newResp(status int) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{}}
}

// TestClassifyError_NilResponse_Routed proves the nil-response (network
// failure) branch emits an i18n-routed Message rather than a hardcoded
// English literal. With the fake translator installed, the Message must
// carry the "<TRANSLATED:...>" prefix.
func TestClassifyError_NilResponse_Routed(t *testing.T) {
	withFakeProvidersTranslator(t, func() {
		ec := NewErrorClassifier("openai")
		pe := ec.ClassifyError(nil, nil)
		if !strings.HasPrefix(pe.Message, "<TRANSLATED:llmsverifier_provider_err_") {
			t.Errorf("network-error message not i18n-routed: %q", pe.Message)
		}
		// Code is an identifier token — must stay verbatim, NOT translated.
		if pe.Code != "NETWORK_ERROR" {
			t.Errorf("error code unexpectedly altered: %q", pe.Code)
		}
	})
}

// TestClassifyError_AllProviders_StatusBranches_Routed walks every provider
// classifier across the full status-code matrix and asserts every emitted
// Message is i18n-routed. A surviving literal in any branch fails the test.
func TestClassifyError_AllProviders_StatusBranches_Routed(t *testing.T) {
	providers := []string{"openai", "anthropic", "deepseek", "gemini", "google", "unknownvendor"}
	statuses := []int{400, 401, 403, 404, 408, 422, 429, 500, 502, 503, 504, 529, 418}
	withFakeProvidersTranslator(t, func() {
		for _, prov := range providers {
			ec := NewErrorClassifier(prov)
			for _, st := range statuses {
				pe := ec.ClassifyError(newResp(st), nil)
				if !strings.HasPrefix(pe.Message, "<TRANSLATED:llmsverifier_provider_err_") {
					t.Errorf("provider %s status %d: message not i18n-routed: %q", prov, st, pe.Message)
				}
			}
		}
	})
}

// TestClassifyError_NoopTranslator_Default proves the production default
// (NoopTranslator) returns the messageID verbatim — the seam is wired but
// never breaks the build when no real backend is installed.
func TestClassifyError_NoopTranslator_Default(t *testing.T) {
	ec := NewErrorClassifier("openai")
	pe := ec.ClassifyError(newResp(429), nil)
	// NoopTranslator returns the id verbatim, so the Message equals the key.
	if pe.Message != "llmsverifier_provider_err_rate_limit_exceeded" {
		t.Errorf("NoopTranslator default did not return messageID verbatim: %q", pe.Message)
	}
}

// TestTr_PairedMutation is the §1.1 paired-mutation guard: it confirms tr()
// genuinely routes through the translator. If tr() were mutated to return its
// argument verbatim (bypassing the translator), this assertion fails because
// the fake translator's sentinel prefix would be absent.
func TestTr_PairedMutation(t *testing.T) {
	withFakeProvidersTranslator(t, func() {
		got := tr("llmsverifier_provider_err_invalid_api_key")
		want := "<TRANSLATED:llmsverifier_provider_err_invalid_api_key>"
		if got != want {
			t.Errorf("tr() not routed through translator: got %q want %q", got, want)
		}
	})
}
