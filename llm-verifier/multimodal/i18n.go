// Package-level i18n seam for the multimodal package per CONST-046.
//
// The root `llmsverifier` module does not yet ship a shared i18n package
// importable from here without a module dependency. This file therefore
// declares a minimal, self-contained Translator interface plus a
// NoopTranslator default — identical in shape to the llmops seam
// (rounds 308-372) so the migration pattern stays uniform across the
// module.
//
// translator is a package-level variable so user-facing
// validation/error/description call sites can be migrated incrementally
// without threading a Translator argument through every constructor.
// Consumers that link this package directly may override the variable
// at init time; the default NoopTranslator returns the messageID
// verbatim, which lets tests assert sentinels rather than English
// literals (anti-bluff invariant per CONST-035 / Article XI §11.9).
package multimodal

import "context"

// Translator is the minimal i18n surface every multimodal call site uses.
// Implementations MUST be safe for concurrent use across goroutines.
type Translator interface {
	// T returns the translated message for messageID. templateData is merged
	// into the message template per the backend's rules; nil is permitted.
	T(ctx context.Context, messageID string, templateData map[string]any) (string, error)
}

// NoopTranslator returns the messageID verbatim. It is the default for
// callers that have not yet been wired to a real backend, and the only
// Translator implementation production code ships by default. Consuming
// projects substitute a real backend (go-i18n bundles, ICU, etc.) at
// construction time.
type NoopTranslator struct{}

// T satisfies Translator.T by returning the messageID unchanged.
func (NoopTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return id, nil
}

// translator is the active i18n backend for the multimodal package. Tests
// in this package may swap it for a fakeTranslator that returns
// "<TRANSLATED:msg_id>" to verify the migration without coupling to the
// English bundle text.
var translator Translator = NoopTranslator{}

// tr is a tiny helper that calls translator.T with a fresh background
// context and discards the error path — validation/description messages
// have no graceful fallback beyond returning the messageID, which is
// exactly what NoopTranslator does on every call anyway.
func tr(id string) string {
	s, err := translator.T(context.Background(), id, nil)
	if err != nil {
		return id
	}
	return s
}

// trData is the parameterised companion to tr — used by call sites that
// previously emitted fmt.Sprintf with substitution placeholders. The
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
