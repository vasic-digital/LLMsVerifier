// Package-level i18n seam for the llmverifier library package per CONST-046.
//
// translator is a package-level variable so call sites can be migrated
// incrementally without threading a Translator argument through every
// report-writer signature. Consumers that link this package directly may
// override the variable at init time; the default NoopTranslator returns
// the messageID verbatim, which lets tests assert sentinels rather than
// English literals (anti-bluff invariant per CONST-035 / Article XI §11.9).
//
// This mirrors the cmd/i18n.go seam (round-194) so the migration pattern
// is uniform across binary and library packages.
package llmverifier

import (
	"context"

	"digital.vasic.llmsverifier/pkg/i18n"
)

// translator is the active i18n backend for the llmverifier package. Tests
// in this package may swap it for a fakeTranslator that returns
// "<TRANSLATED:msg_id>" to verify the migration without coupling to the
// English bundle text.
var translator i18n.Translator = i18n.NoopTranslator{}

// tr is a tiny helper that calls translator.T with a fresh background
// context and discards the error path — the report-writer sites have no
// graceful fallback beyond returning the messageID, which is exactly what
// NoopTranslator does on every call anyway.
func tr(id string) string {
	s, err := translator.T(context.Background(), id, nil)
	if err != nil {
		return id
	}
	return s
}

// trData is the parameterised companion to tr — used by call sites that
// previously emitted fmt.Fprintf with substitution placeholders. The
// translator backend is responsible for materialising the placeholders
// against its own template syntax (go-i18n {{.field}}, ICU {field}, etc.);
// NoopTranslator still returns the messageID verbatim so tests can assert
// the i18n-routed sentinel without coupling to a specific backend.
func trData(id string, data map[string]any) string {
	s, err := translator.T(context.Background(), id, data)
	if err != nil {
		return id
	}
	return s
}
