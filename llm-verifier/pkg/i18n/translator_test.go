package i18n

import (
	"context"
	"testing"
)

func TestNoopTranslator_TReturnsMessageIDVerbatim(t *testing.T) {
	tr := NoopTranslator{}
	got, err := tr.T(context.Background(), "llmsverifier_models_table_empty", map[string]any{"k": "v"})
	if err != nil {
		t.Fatalf("NoopTranslator.T returned err: %v", err)
	}
	if got != "llmsverifier_models_table_empty" {
		t.Fatalf("NoopTranslator.T = %q; want messageID verbatim", got)
	}
}

func TestNoopTranslator_TPluralReturnsMessageIDVerbatim(t *testing.T) {
	tr := NoopTranslator{}
	got, err := tr.TPlural(context.Background(), "llmsverifier_models_count", 5, nil)
	if err != nil {
		t.Fatalf("NoopTranslator.TPlural returned err: %v", err)
	}
	if got != "llmsverifier_models_count" {
		t.Fatalf("NoopTranslator.TPlural = %q; want messageID verbatim", got)
	}
}
