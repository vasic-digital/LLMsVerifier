// Package-level i18n seam for the TUI root package per CONST-046.
//
// translator is a package-level variable so the App header / footer / help
// render call sites and the notification helper messages can be migrated
// incrementally without threading a Translator argument through every
// tea.Model constructor. Consumers that link this package directly may
// override the variable at init time; the default NoopTranslator returns the
// messageID verbatim, which lets tests assert sentinels rather than English
// literals (anti-bluff invariant per CONST-035 / Article XI §11.9).
//
// This mirrors the tui/screens/i18n.go, llmverifier/i18n.go, enhanced/i18n.go,
// monitoring/i18n.go, events/i18n.go and other per-package seams established
// across rounds 316-439 so the migration pattern stays uniform across binary,
// library and terminal-UI packages.
package tui

import (
	"context"

	"digital.vasic.llmsverifier/pkg/i18n"
)

// translator is the active i18n backend for the TUI root package. Tests in
// this package may swap it for a fakeTUITranslator that returns
// "<TRANSLATED:msg_id>" to verify the migration without coupling to the
// English bundle text.
var translator i18n.Translator = i18n.NoopTranslator{}

// trTUI is a tiny helper that calls translator.T with a fresh background
// context and discards the error path — TUI label/heading strings have no
// graceful fallback beyond returning the messageID, which is exactly what
// NoopTranslator does on every call anyway.
func trTUI(id string) string {
	s, err := translator.T(context.Background(), id, nil)
	if err != nil {
		return id
	}
	return s
}

// trTUIData is the parameterised variant of trTUI for notification messages
// that interpolate runtime values (model name, error text). The templateData
// map is merged into the message template by a real backend; NoopTranslator
// ignores it and returns the messageID verbatim, so tests assert the sentinel
// rather than English text. This mirrors monitoring/i18n.go's trData helper so
// the parameterised migration pattern stays uniform across packages.
func trTUIData(id string, data map[string]any) string {
	s, err := translator.T(context.Background(), id, data)
	if err != nil {
		return id
	}
	return s
}
