// Package-level i18n seam for the test-suite builder CLI per CONST-046.
//
// translator is a package-level variable so call sites can be migrated
// incrementally without threading a Translator argument through every
// printer signature. The default NoopTranslator returns the messageID
// verbatim, which lets tests assert sentinels rather than English
// literals (anti-bluff invariant per CONST-035 / Article XI §11.9).
//
// This mirrors cmd/i18n.go and cmd/model-verification/i18n.go — the
// test-suite builder is a distinct `main` package, so it carries its own
// copy of the tiny seam rather than reaching into a sibling main package
// (which Go forbids anyway).
package main

import (
	"context"

	"digital.vasic.llmsverifier/pkg/i18n"
)

// translator is the active i18n backend for this binary. Tests in this
// package swap it for a fakeTranslator that returns "<TRANSLATED:msg_id>"
// to verify the migration without coupling to the English bundle text.
var translator i18n.Translator = i18n.NoopTranslator{}

// tr calls translator.T with a fresh background context and discards the
// error path — the print sites have no graceful fallback beyond returning
// the messageID, which is exactly what NoopTranslator does anyway.
func tr(id string) string {
	s, err := translator.T(context.Background(), id, nil)
	if err != nil {
		return id
	}
	return s
}

// trData is the parameterised companion to tr — used by call sites that
// previously emitted fmt.Printf with substitution placeholders. The
// translator backend materialises the placeholders against its own
// template syntax; NoopTranslator still returns the messageID verbatim.
func trData(id string, data map[string]any) string {
	s, err := translator.T(context.Background(), id, data)
	if err != nil {
		return id
	}
	return s
}
