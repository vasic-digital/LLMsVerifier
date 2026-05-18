// Package i18n provides the LLMsVerifier i18n seam per CONST-046.
//
// LLMsVerifier exposes a Translator interface plus a NoopTranslator default
// so call sites can be migrated incrementally without consumers breaking.
// Real translation backends (go-i18n bundles, ICU, etc.) are injected by
// consuming projects via constructor parameter — never resolved by hardcoded
// reach into a parent project (CONST-051(B) decoupling).
//
// Migrated call sites use exactly this pattern:
//
//	msg, _ := tr.T(ctx, "llmsverifier_<descriptor>", map[string]any{...})
//	fmt.Println(msg)
//
// When tr is NoopTranslator{}, the message ID is returned verbatim — which
// is intentional: callers stay testable against a sentinel
// ("<TRANSLATED:msg_id>" via fakeTranslator) without depending on the
// English text. Anti-bluff invariant: tests assert the sentinel, NEVER the
// original literal.
package i18n

import "context"

// Translator is the minimal i18n surface every LLMsVerifier call site uses.
// Implementations MUST be safe for concurrent use across goroutines.
type Translator interface {
	// T returns the translated message for messageID. templateData is merged
	// into the message template per the backend's rules; nil is permitted.
	T(ctx context.Context, messageID string, templateData map[string]any) (string, error)

	// TPlural returns a count-aware translation. Backends pick the
	// appropriate plural form (CLDR rules) from count and templateData.
	TPlural(ctx context.Context, messageID string, count int, templateData map[string]any) (string, error)
}

// NoopTranslator returns the messageID verbatim. It is the default for
// callers that have not yet been wired to a real backend, and it is the
// only Translator implementation that production code may ship by default.
// Consuming projects substitute a real backend at construction time.
type NoopTranslator struct{}

// T satisfies Translator.T by returning the messageID unchanged.
func (NoopTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return id, nil
}

// TPlural satisfies Translator.TPlural by returning the messageID unchanged.
func (NoopTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return id, nil
}
