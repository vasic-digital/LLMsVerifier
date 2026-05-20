// Package-level i18n seam for the scoring package per CONST-046.
//
// translator is a package-level variable so scoring HTTP-handler call
// sites can be migrated incrementally without threading a Translator
// argument through every gin.H{"error": ...} / gin.H{"message": ...}
// assignment. Consumers that link this package directly may override the
// variable at init time; the default NoopTranslator returns the messageID
// verbatim, which lets tests assert sentinels rather than English literals
// (anti-bluff invariant per CONST-035 / Article XI §11.9).
//
// This mirrors the providers/i18n.go, llmverifier/i18n.go, monitoring/i18n.go
// and events/i18n.go seams (rounds 364 / 316 / 326 / earlier) so the
// migration pattern stays uniform across binary and library packages.
package scoring

import (
	"context"

	"digital.vasic.llmsverifier/pkg/i18n"
)

// translator is the active i18n backend for the scoring package. Tests in
// this package may swap it for a fakeScoringTranslator that returns
// "<TRANSLATED:msg_id>" to verify the migration without coupling to the
// English bundle text.
var translator i18n.Translator = i18n.NoopTranslator{}

// tr is a tiny helper that calls translator.T with a fresh background
// context and discards the error path — scoring API status/error messages
// have no graceful fallback beyond returning the messageID, which is
// exactly what NoopTranslator does on every call anyway.
func tr(id string) string {
	s, err := translator.T(context.Background(), id, nil)
	if err != nil {
		return id
	}
	return s
}
