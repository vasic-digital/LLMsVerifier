// Package-level i18n seam for the selection package per CONST-046.
//
// translator is a package-level variable so Decision.Reason rendering can be
// migrated without threading a Translator through every Select call. The
// default NoopTranslator returns the messageID verbatim, which lets tests
// assert sentinels rather than English literals (anti-bluff invariant per
// CONST-035 / Article XI §11.9).
//
// This mirrors the capabilities/i18n.go, scoring/i18n.go, providers/i18n.go,
// monitoring/i18n.go and events/i18n.go seams so the pattern stays uniform.
package selection

import (
	"context"

	"digital.vasic.llmsverifier/pkg/i18n"
)

// translator is the active i18n backend for the selection package. Tests may
// swap it for a fake returning "<TRANSLATED:msg_id>" to verify that reason
// rendering actually routes through i18n.
var translator i18n.Translator = i18n.NoopTranslator{}

// SetTranslator installs an i18n backend for this package. It exists so a
// consuming application can localise Decision.Reason without this package
// knowing anything about that consumer.
func SetTranslator(t i18n.Translator) {
	if t == nil {
		translator = i18n.NoopTranslator{}
		return
	}
	translator = t
}

// tr resolves a message ID through the active translator, falling back to the
// ID itself — which is exactly what NoopTranslator returns anyway.
func tr(id string) string {
	s, err := translator.T(context.Background(), id, nil)
	if err != nil {
		return id
	}
	return s
}
