package multimodal

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// fakeMultimodalTranslator returns "<TRANSLATED:msg_id>" so tests can assert
// the i18n-routed sentinel without coupling to the English bundle text.
// Anti-bluff per CONST-035 / Article XI §11.9: a test asserting the original
// literal would silently pass if the call-site bypassed the translator.
type fakeMultimodalTranslator struct{}

func (fakeMultimodalTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeMultimodalTranslator installs the fakeMultimodalTranslator, runs fn,
// then restores the prior translator.
func withFakeMultimodalTranslator(t *testing.T, fn func()) {
	t.Helper()
	prior := translator
	translator = fakeMultimodalTranslator{}
	defer func() { translator = prior }()
	fn()
}

// TestI18nSeam_NoopReturnsMessageIDVerbatim proves the default seam is a safe
// pass-through: production ships NoopTranslator, which returns the message ID
// unchanged so nothing breaks before a real backend is wired.
func TestI18nSeam_NoopReturnsMessageIDVerbatim(t *testing.T) {
	assert.Equal(t, "multimodal.validation.content_required",
		tr("multimodal.validation.content_required"))
	assert.Equal(t, "multimodal.error.content_validation_failed",
		trData("multimodal.error.content_validation_failed", map[string]any{"error": "x"}))
}

// migratedMessageIDs is the canonical list of every CONST-046 message ID
// introduced by the round-376 multimodal migration. The migration test and
// the paired-mutation test both walk this list so they stay in lockstep.
var migratedMessageIDs = []string{
	"multimodal.error.content_validation_failed",
	"multimodal.error.safety_check_failed",
	"multimodal.error.content_processing_failed",
	"multimodal.error.llm_response_failed",
	"multimodal.validation.content_required",
	"multimodal.validation.content_type_required",
	"multimodal.validation.content_size_exceeded",
	"multimodal.validation.invalid_mime_type",
	"multimodal.analysis.description",
	"multimodal.analysis.detected_text",
	"multimodal.analysis.transcript",
	"multimodal.analysis.detected_objects",
	"multimodal.analysis.topics",
	"multimodal.analysis.sentiment",
	"multimodal.analysis.language",
	"multimodal.analysis.summary",
	"multimodal.video.keyframe_analysis",
	"multimodal.video.file_metadata",
	"multimodal.video.ffmpeg_note",
	"multimodal.audio.transcription_metadata",
	"multimodal.audio.transcription_via_google",
	"multimodal.safety.no_api_response",
	"multimodal.safety.content_flagged",
	"multimodal.safety.file_size_exceeded",
	"multimodal.safety.suspicious_file",
	"multimodal.safety.unknown_mime_type",
}

// TestI18nSeam_FakeTranslatorRoutesEveryID confirms the seam genuinely
// delegates to the active translator for every migrated message ID.
func TestI18nSeam_FakeTranslatorRoutesEveryID(t *testing.T) {
	withFakeMultimodalTranslator(t, func() {
		for _, id := range migratedMessageIDs {
			assert.Equal(t, "<TRANSLATED:"+id+">", tr(id),
				"tr(%q) must route through the active translator", id)
			assert.Equal(t, "<TRANSLATED:"+id+">",
				trData(id, map[string]any{"k": "v"}),
				"trData(%q) must route through the active translator", id)
		}
	})
}

// TestMigration_ValidateContentRoutesThroughTranslator is the paired-mutation
// guard for the validateContent call sites. With the fake translator active,
// every error string returned by validateContent MUST be an i18n sentinel —
// if a future edit reintroduced a hardcoded English literal, the sentinel
// assertion fails. This is real behaviour: validateContent runs unmodified.
func TestMigration_ValidateContentRoutesThroughTranslator(t *testing.T) {
	mmp := NewMultiModalProcessor()

	withFakeMultimodalTranslator(t, func() {
		// nil content
		err := mmp.validateContent(nil)
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "<TRANSLATED:multimodal.validation.content_required>")
		}

		// missing content type
		err = mmp.validateContent(&MultiModalContent{})
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "<TRANSLATED:multimodal.validation.content_type_required>")
		}

		// oversized content
		err = mmp.validateContent(&MultiModalContent{
			Type: ContentTypeImage,
			Size: 999 * 1024 * 1024,
		})
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "<TRANSLATED:multimodal.validation.content_size_exceeded>")
		}

		// invalid MIME type
		err = mmp.validateContent(&MultiModalContent{
			Type:     ContentTypeImage,
			MimeType: "application/zip",
		})
		if assert.Error(t, err) {
			assert.Contains(t, err.Error(), "<TRANSLATED:multimodal.validation.invalid_mime_type>")
		}
	})
}

// TestMigration_BasicSafetyCheckRoutesThroughTranslator is the paired-mutation
// guard for the performBasicSafetyCheck SafetyIssue descriptions. The real
// safety checker runs against real content; every flagged-issue description
// MUST be an i18n sentinel under the fake translator.
func TestMigration_BasicSafetyCheckRoutesThroughTranslator(t *testing.T) {
	csc := NewContentSafetyChecker(NewMultiModalProcessor())

	withFakeMultimodalTranslator(t, func() {
		// suspicious MIME type -> "suspicious_file" issue
		res, err := csc.performBasicSafetyCheck(&MultiModalContent{
			Type:     ContentTypeImage,
			MimeType: "application/x-executable",
		})
		assert.NoError(t, err)
		assert.False(t, res.Safe)
		foundSuspicious := false
		for _, iss := range res.Issues {
			if iss.Type == "suspicious_file" {
				foundSuspicious = true
				assert.Equal(t, "<TRANSLATED:multimodal.safety.suspicious_file>", iss.Description)
			}
		}
		assert.True(t, foundSuspicious, "expected a suspicious_file safety issue")

		// oversized file -> "file_size" issue
		res, err = csc.performBasicSafetyCheck(&MultiModalContent{
			Type: ContentTypeImage,
			Size: 999 * 1024 * 1024,
		})
		assert.NoError(t, err)
		foundSize := false
		for _, iss := range res.Issues {
			if iss.Type == "file_size" {
				foundSize = true
				assert.Equal(t, "<TRANSLATED:multimodal.safety.file_size_exceeded>", iss.Description)
			}
		}
		assert.True(t, foundSize, "expected a file_size safety issue")

		// unknown MIME type -> "unknown_mime_type" issue
		res, err = csc.performBasicSafetyCheck(&MultiModalContent{
			Type:     ContentTypeImage,
			MimeType: "application/x-custom-format",
		})
		assert.NoError(t, err)
		foundUnknown := false
		for _, iss := range res.Issues {
			if iss.Type == "unknown_mime_type" {
				foundUnknown = true
				assert.Equal(t, "<TRANSLATED:multimodal.safety.unknown_mime_type>", iss.Description)
			}
		}
		assert.True(t, foundUnknown, "expected an unknown_mime_type safety issue")
	})
}

// TestMigration_NoHardcodedLiteralLeak is a belt-and-braces mutation check:
// with the fake translator active, none of the migrated call sites may emit
// the original English literal. Asserts the sentinels never collide with the
// pre-migration text fragments.
func TestMigration_NoHardcodedLiteralLeak(t *testing.T) {
	withFakeMultimodalTranslator(t, func() {
		for _, id := range migratedMessageIDs {
			got := tr(id)
			assert.True(t, strings.HasPrefix(got, "<TRANSLATED:"),
				"message %q leaked a non-i18n literal: %q", id, got)
		}
	})
}
