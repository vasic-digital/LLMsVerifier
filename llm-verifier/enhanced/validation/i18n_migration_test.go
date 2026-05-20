package validation

import (
	"context"
	"strings"
	"testing"
)

// fakeValidationTranslator returns "<TRANSLATED:msg_id>" so tests can assert
// the i18n-routed sentinel without coupling to the English bundle text.
// Anti-bluff per CONST-035 / Article XI §11.9: a test asserting the original
// literal would silently pass if the call-site bypassed the translator.
type fakeValidationTranslator struct{}

func (fakeValidationTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}
func (fakeValidationTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeValidationTranslator installs the fakeValidationTranslator, runs fn,
// then restores the prior translator.
func withFakeValidationTranslator(t *testing.T, fn func()) {
	t.Helper()
	prior := translator
	translator = fakeValidationTranslator{}
	defer func() { translator = prior }()
	fn()
}

func ptrInt(v int) *int           { return &v }
func ptrFloat(v float64) *float64 { return &v }

// TestSyntaxValidator_MessagesRouted proves the SyntaxValidator routes every
// error and warning string through the i18n seam. With the fake translator
// installed, every migrated string must carry the "<TRANSLATED:validation.*>"
// prefix — if a branch still held an English literal, the assertion fails.
func TestSyntaxValidator_MessagesRouted(t *testing.T) {
	withFakeValidationTranslator(t, func() {
		sv := NewSyntaxValidator("syntax")
		ctx := context.Background()

		// Empty prompt -> error.
		res := sv.Validate(ctx, "   ")
		if len(res.Errors) == 0 {
			t.Fatal("expected an error for empty prompt")
		}
		for _, e := range res.Errors {
			if !strings.HasPrefix(e, "<TRANSLATED:validation.") {
				t.Errorf("empty-prompt error not i18n-routed: %q", e)
			}
		}

		// Harmful content -> error.
		res = sv.Validate(ctx, "please DROP TABLE users now")
		if len(res.Errors) == 0 {
			t.Fatal("expected an error for harmful content")
		}
		for _, e := range res.Errors {
			if !strings.HasPrefix(e, "<TRANSLATED:validation.") {
				t.Errorf("harmful-content error not i18n-routed: %q", e)
			}
		}

		// Unbalanced brackets -> warning.
		res = sv.Validate(ctx, "what does ( this do")
		for _, w := range res.Warnings {
			if !strings.HasPrefix(w, "<TRANSLATED:validation.") {
				t.Errorf("unbalanced-brackets warning not i18n-routed: %q", w)
			}
		}

		// Long prompt -> warning.
		res = sv.Validate(ctx, strings.Repeat("a ", 6000))
		for _, w := range res.Warnings {
			if !strings.HasPrefix(w, "<TRANSLATED:validation.") {
				t.Errorf("long-prompt warning not i18n-routed: %q", w)
			}
		}

		// Unsupported input type -> error.
		res = sv.Validate(ctx, 42)
		if len(res.Errors) == 0 {
			t.Fatal("expected an error for unsupported input type")
		}
		if !strings.HasPrefix(res.Errors[0], "<TRANSLATED:validation.") {
			t.Errorf("unsupported-input error not i18n-routed: %q", res.Errors[0])
		}

		// LLMRequest with bad messages, temperature, max tokens.
		req := &LLMRequest{
			Messages: []Message{
				{Role: "user", Content: ""},
				{Role: "wizard", Content: "hi"},
			},
			Temperature: ptrFloat(9.0),
			MaxTokens:   ptrInt(-1),
		}
		res = sv.Validate(ctx, req)
		for _, e := range res.Errors {
			if !strings.HasPrefix(e, "<TRANSLATED:validation.") {
				t.Errorf("LLMRequest error not i18n-routed: %q", e)
			}
		}
		for _, w := range res.Warnings {
			if !strings.HasPrefix(w, "<TRANSLATED:validation.") {
				t.Errorf("LLMRequest warning not i18n-routed: %q", w)
			}
		}
	})
}

// TestSemanticValidator_MessagesRouted proves the SemanticValidator routes
// every warning and error through the i18n seam.
func TestSemanticValidator_MessagesRouted(t *testing.T) {
	withFakeValidationTranslator(t, func() {
		sv := NewSemanticValidator("semantic", nil)
		ctx := context.Background()

		// Very brief prompt -> warning.
		res := sv.Validate(ctx, "hi")
		for _, w := range res.Warnings {
			if !strings.HasPrefix(w, "<TRANSLATED:validation.") {
				t.Errorf("brief-prompt warning not i18n-routed: %q", w)
			}
		}

		// Excessive repetition -> warning.
		res = sv.Validate(ctx, strings.Repeat("automation ", 30))
		for _, w := range res.Warnings {
			if !strings.HasPrefix(w, "<TRANSLATED:validation.") {
				t.Errorf("repetition warning not i18n-routed: %q", w)
			}
		}

		// Ambiguous pronouns -> warning.
		res = sv.Validate(ctx, "it does this and that with them and these and those.")
		for _, w := range res.Warnings {
			if !strings.HasPrefix(w, "<TRANSLATED:validation.") {
				t.Errorf("ambiguous-pronoun warning not i18n-routed: %q", w)
			}
		}

		// Unsupported input type -> error.
		res = sv.Validate(ctx, 42)
		if len(res.Errors) == 0 {
			t.Fatal("expected an error for unsupported input type")
		}
		if !strings.HasPrefix(res.Errors[0], "<TRANSLATED:validation.") {
			t.Errorf("unsupported-input error not i18n-routed: %q", res.Errors[0])
		}

		// Conversation flow warnings (consecutive + very long messages).
		req := &LLMRequest{Messages: []Message{
			{Role: "user", Content: "a"},
			{Role: "user", Content: "b"},
			{Role: "user", Content: "c"},
			{Role: "user", Content: strings.Repeat("x", 6000)},
		}}
		res = sv.Validate(ctx, req)
		for _, w := range res.Warnings {
			if !strings.HasPrefix(w, "<TRANSLATED:validation.") {
				t.Errorf("conversation-flow warning not i18n-routed: %q", w)
			}
		}
	})
}

// TestValidationMutationGuard is the paired-mutation test per §1.1. With the
// production-default NoopTranslator, the verbatim message ID is returned —
// NOT an English literal. A regression that re-hardcoded "Prompt cannot be
// empty" would make the error differ from the message ID, failing this test.
func TestValidationMutationGuard(t *testing.T) {
	if got := tr("validation.error.prompt_empty"); got != "validation.error.prompt_empty" {
		t.Fatalf("NoopTranslator must return the bare id; got %q", got)
	}
	if got := trData("validation.error.message_empty_content", map[string]any{"index": 0}); got != "validation.error.message_empty_content" {
		t.Fatalf("NoopTranslator (trData) must return the bare id; got %q", got)
	}

	sv := NewSyntaxValidator("syntax")
	res := sv.Validate(context.Background(), "   ")
	if len(res.Errors) == 0 {
		t.Fatal("expected an error for empty prompt")
	}
	for _, e := range res.Errors {
		if strings.Contains(e, "Prompt cannot be empty") ||
			strings.Contains(e, "potentially harmful") {
			t.Fatalf("validation error regressed to a hardcoded English literal: %q", e)
		}
		// Under NoopTranslator the error is the bare message ID.
		if !strings.HasPrefix(e, "validation.") {
			t.Fatalf("validation error not routed through the i18n seam: %q", e)
		}
	}
}
