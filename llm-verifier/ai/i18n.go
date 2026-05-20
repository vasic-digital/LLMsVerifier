// Package-level i18n seam for the ai package per CONST-046.
//
// translator is a package-level variable so ai call sites (recommendation
// reasoning fragments shown to end users) can be migrated incrementally
// without threading a Translator argument through every recommender method.
// Consumers that link this package directly may override the variable at
// init time; the default NoopTranslator returns the messageID verbatim,
// which lets tests assert sentinels rather than English literals (anti-bluff
// invariant per CONST-035 / Article XI §11.9).
//
// This mirrors the providers/i18n.go, scoring/i18n.go and auth/i18n.go seams
// so the migration pattern stays uniform across the codebase.
package ai

import (
	"context"

	"digital.vasic.llmsverifier/pkg/i18n"
)

// translator is the active i18n backend for the ai package. Tests in this
// package may swap it for a fakeAITranslator that returns
// "<TRANSLATED:msg_id>" to verify the migration without coupling to the
// English bundle text.
var translator i18n.Translator = i18n.NoopTranslator{}

// tr returns the translated message for id with a fresh background context.
// The error path is discarded — a recommendation reasoning fragment has no
// graceful fallback beyond the messageID, which is what NoopTranslator
// returns anyway.
func tr(id string) string {
	s, err := translator.T(context.Background(), id, nil)
	if err != nil {
		return id
	}
	return s
}
