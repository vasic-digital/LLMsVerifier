// Package-level i18n seam for the enhanced/enterprise package per CONST-046.
//
// translator is a package-level variable so the enterprise HTTP handlers can
// be migrated incrementally without threading a Translator argument through
// every writeError / writeJSON call site. Consumers that link this package
// directly may override the variable at init time; the default NoopTranslator
// returns the messageID verbatim, which lets tests assert sentinels rather
// than English literals (anti-bluff invariant per CONST-035 / Article XI
// §11.9).
//
// This mirrors the llmverifier/i18n.go, enhanced/i18n.go, enhanced/analytics/
// i18n.go, monitoring/i18n.go and events/i18n.go seams (rounds 316 / 319 /
// 326 / 328 / 340) so the migration pattern stays uniform across binary and
// library packages.
package enterprise

import (
	"context"

	"digital.vasic.llmsverifier/pkg/i18n"
)

// translator is the active i18n backend for the enterprise package. Tests in
// this package may swap it for a fakeEnterpriseTranslator that returns
// "<TRANSLATED:msg_id>" to verify the migration without coupling to the
// English bundle text.
var translator i18n.Translator = i18n.NoopTranslator{}

// tr is a tiny helper that calls translator.T with a fresh background context
// and discards the error path — HTTP status messages have no graceful
// fallback beyond returning the messageID, which is exactly what
// NoopTranslator does on every call anyway.
func tr(id string) string {
	s, err := translator.T(context.Background(), id, nil)
	if err != nil {
		return id
	}
	return s
}
