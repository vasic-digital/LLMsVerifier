// Package-level i18n seam for the crush-config-converter CLI per CONST-046.
//
// translator is a package-level variable so the converter's user-facing
// status lines and fatal error messages can be migrated incrementally
// without threading a Translator argument through main(). The default
// NoopTranslator returns the messageID verbatim, which lets tests assert
// sentinels rather than English literals (anti-bluff invariant per
// CONST-035 / Article XI §11.9).
//
// This mirrors the capabilities/i18n.go, scoring/i18n.go and cmd/i18n.go
// seams so the migration pattern stays uniform across the codebase.
package main

import (
	"context"

	"digital.vasic.llmsverifier/pkg/i18n"
)

// translator is the active i18n backend for the crush-config-converter
// CLI. Tests in this package may swap it for a fakeConverterTranslator
// that returns "<TRANSLATED:msg_id>" to verify the migration without
// coupling to the English bundle text.
var translator i18n.Translator = i18n.NoopTranslator{}

// tr is a tiny helper that calls translator.T with a fresh background
// context and discards the error path — converter CLI messages have no
// graceful fallback beyond returning the messageID, which is exactly
// what NoopTranslator does on every call anyway.
func tr(id string) string {
	s, err := translator.T(context.Background(), id, nil)
	if err != nil {
		return id
	}
	return s
}
