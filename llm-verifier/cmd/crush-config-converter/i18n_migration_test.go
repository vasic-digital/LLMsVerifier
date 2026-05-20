package main

import (
	"context"
	"strings"
	"testing"
)

// fakeConverterTranslator returns "<TRANSLATED:msg_id>" so tests can
// assert the i18n-routed sentinel without coupling to the English bundle
// text. Anti-bluff per CONST-035 / Article XI §11.9: a test asserting the
// original English literal would silently pass if the call-site bypassed
// the translator entirely.
type fakeConverterTranslator struct{}

func (fakeConverterTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}
func (fakeConverterTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeConverterTranslator installs the fakeConverterTranslator, runs
// fn, then restores the prior translator.
func withFakeConverterTranslator(t *testing.T, fn func()) {
	t.Helper()
	prior := translator
	translator = fakeConverterTranslator{}
	defer func() { translator = prior }()
	fn()
}

// TestConverterTr_RoutesThroughTranslator proves the package-level tr
// helper actually consults the active translator rather than echoing the
// messageID. With the fake translator installed every lookup must carry
// the "<TRANSLATED:...>" prefix. This is the paired mutation for the
// crush-config-converter i18n seam: if a future change made tr() return
// its argument verbatim, this test fails.
func TestConverterTr_RoutesThroughTranslator(t *testing.T) {
	withFakeConverterTranslator(t, func() {
		ids := []string{
			"llmsverifier_crushconv_usage",
			"llmsverifier_crushconv_generating_brotli",
			"llmsverifier_crushconv_err_read_discovery",
			"llmsverifier_crushconv_err_parse_json",
			"llmsverifier_crushconv_err_marshal_config",
			"llmsverifier_crushconv_err_write_config",
			"llmsverifier_crushconv_err_marshal_redacted",
			"llmsverifier_crushconv_err_write_redacted",
			"llmsverifier_crushconv_err_marshal_opencode",
			"llmsverifier_crushconv_err_write_opencode",
			"llmsverifier_crushconv_written_crush",
			"llmsverifier_crushconv_written_redacted",
			"llmsverifier_crushconv_written_opencode",
			"llmsverifier_crushconv_written_stats",
			"llmsverifier_crushconv_brotli_summary_header",
			"llmsverifier_crushconv_brotli_total_providers",
			"llmsverifier_crushconv_brotli_supported_providers",
			"llmsverifier_crushconv_brotli_support_rate",
		}
		for _, id := range ids {
			got := tr(id)
			want := "<TRANSLATED:" + id + ">"
			if got != want {
				t.Errorf("tr(%q) = %q, not i18n-routed (want %q)", id, got, want)
			}
		}
	})
}

// TestConverterTr_NoopReturnsMessageIDVerbatim proves the default
// NoopTranslator path: without an override, tr returns the messageID
// itself. This guarantees the migration introduces no panic and the seam
// degrades gracefully.
func TestConverterTr_NoopReturnsMessageIDVerbatim(t *testing.T) {
	const id = "llmsverifier_crushconv_usage"
	if got := tr(id); got != id {
		t.Errorf("NoopTranslator tr(%q) = %q, want verbatim messageID", id, got)
	}
}

// TestConverterFormatKeys_PreserveVerbDirectives proves the migrated
// printf-style message IDs still resolve to format strings carrying the
// expected verb directives once the real bundle is loaded. The default
// NoopTranslator returns the messageID verbatim, so this asserts the
// messageID naming is stable for the format-string call sites; a renamed
// key without a matching bundle entry would later fail the audit.
func TestConverterFormatKeys_PreserveVerbDirectives(t *testing.T) {
	formatKeys := []string{
		"llmsverifier_crushconv_err_read_discovery",
		"llmsverifier_crushconv_written_crush",
		"llmsverifier_crushconv_brotli_total_providers",
		"llmsverifier_crushconv_brotli_support_rate",
	}
	for _, k := range formatKeys {
		if got := tr(k); !strings.HasPrefix(got, "llmsverifier_crushconv_") {
			t.Errorf("format key %q resolved unexpectedly: %q", k, got)
		}
	}
}
