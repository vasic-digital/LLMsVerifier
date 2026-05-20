// Package-level i18n seam for the code-verification CLI binary per CONST-046.
//
// translator is a package-level variable so the help-text and summary-report
// call sites in main.go can be migrated incrementally without threading a
// Translator argument through printHelp / printSummary. The default
// NoopTranslator returns the messageID verbatim, which lets tests assert
// sentinels rather than English literals (anti-bluff invariant per CONST-035 /
// Article XI §11.9).
//
// This mirrors the cmd/i18n.go, llmverifier/i18n.go, analytics/i18n.go,
// events/i18n.go, enhanced/i18n.go and messaging/i18n.go seams established in
// earlier CONST-046 Phase-4 rounds so the migration pattern stays uniform
// across binary and library packages.
package main

import (
	"context"

	"digital.vasic.llmsverifier/pkg/i18n"
)

// translator is the active i18n backend for the code-verification binary.
// Tests in this package swap it for a fakeCodeVerificationTranslator that
// returns "<TRANSLATED:msg_id>" to verify the migration without coupling to
// the English bundle text.
var translator i18n.Translator = i18n.NoopTranslator{}

// tr is a tiny helper that calls translator.T with a fresh background context
// and discards the error path — help-text and summary-line strings have no
// graceful fallback beyond returning the messageID, which is exactly what
// NoopTranslator does on every call anyway.
func tr(id string) string {
	s, err := translator.T(context.Background(), id, nil)
	if err != nil {
		return id
	}
	return s
}
