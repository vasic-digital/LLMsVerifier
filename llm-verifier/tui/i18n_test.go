package tui

import (
	"context"
	"strings"
	"testing"
)

// fakeTUITranslator returns "<TRANSLATED:msg_id>" for every message ID,
// proving that the App render call sites and notification helpers route their
// user-facing text through the i18n seam rather than emitting hardcoded
// English literals (CONST-046 anti-bluff invariant per Article XI §11.9).
type fakeTUITranslator struct{}

func (fakeTUITranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

func (fakeTUITranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeTUITranslator swaps the package translator for the duration of a
// test and restores the prior backend afterwards.
func withFakeTUITranslator(t *testing.T) {
	t.Helper()
	prev := translator
	translator = fakeTUITranslator{}
	t.Cleanup(func() { translator = prev })
}

// TestTrTUI_RoutesThroughTranslator is the positive case: with the fake
// backend installed, trTUI() must emit the sentinel, not the message ID
// verbatim.
func TestTrTUI_RoutesThroughTranslator(t *testing.T) {
	withFakeTUITranslator(t)
	got := trTUI("tui.app.title")
	if got != "<TRANSLATED:tui.app.title>" {
		t.Fatalf("trTUI did not route through translator: got %q", got)
	}
}

// TestTrTUI_NoopReturnsMessageIDVerbatim is the paired-mutation case: with the
// default NoopTranslator, trTUI() returns the message ID unchanged — so the
// positive test above genuinely exercises the seam swap rather than passing
// trivially.
func TestTrTUI_NoopReturnsMessageIDVerbatim(t *testing.T) {
	if got := trTUI("tui.app.footer.help"); got != "tui.app.footer.help" {
		t.Fatalf("NoopTranslator path: got %q; want messageID verbatim", got)
	}
}

// TestTrTUIData_RoutesThroughTranslator verifies the parameterised helper used
// by notification messages routes through the translator.
func TestTrTUIData_RoutesThroughTranslator(t *testing.T) {
	withFakeTUITranslator(t)
	got := trTUIData("tui.notification.model_verified", map[string]any{"model": "gpt-4"})
	if got != "<TRANSLATED:tui.notification.model_verified>" {
		t.Fatalf("trTUIData did not route through translator: got %q", got)
	}
}

// TestTrTUIData_NoopReturnsMessageIDVerbatim is the paired-mutation case for
// the parameterised helper.
func TestTrTUIData_NoopReturnsMessageIDVerbatim(t *testing.T) {
	got := trTUIData("tui.notification.connection_error", map[string]any{"x": 1})
	if got != "tui.notification.connection_error" {
		t.Fatalf("NoopTranslator path: got %q; want messageID verbatim", got)
	}
}

// TestAppView_InitializingRoutesThroughTranslator proves the App.View()
// uninitialised-window branch emits the translated sentinel rather than a
// hardcoded "Initializing..." literal.
func TestAppView_InitializingRoutesThroughTranslator(t *testing.T) {
	withFakeTUITranslator(t)
	a := &App{} // width and height zero
	got := a.View()
	if !strings.Contains(got, "<TRANSLATED:tui.app.initializing>") {
		t.Fatalf("App.View() did not route initializing string through translator: got %q", got)
	}
	if strings.Contains(got, "Initializing...") {
		t.Fatalf("App.View() still emits the hardcoded English literal: got %q", got)
	}
}

// TestRenderHeader_RoutesTitleAndNavThroughTranslator proves the header title
// and navigation items go through the i18n seam.
func TestRenderHeader_RoutesTitleAndNavThroughTranslator(t *testing.T) {
	withFakeTUITranslator(t)
	a := &App{width: 120, height: 40}
	got := a.renderHeader()
	for _, want := range []string{
		"<TRANSLATED:tui.app.title>",
		"<TRANSLATED:tui.app.nav.dashboard>",
		"<TRANSLATED:tui.app.nav.models>",
		"<TRANSLATED:tui.app.nav.providers>",
		"<TRANSLATED:tui.app.nav.verification>",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("renderHeader missing translated sentinel %q in: %q", want, got)
		}
	}
	if strings.Contains(got, "LLM Verifier TUI") {
		t.Fatalf("renderHeader still emits hardcoded English title: %q", got)
	}
}

// TestRenderFooter_RoutesHelpThroughTranslator proves the footer help text
// goes through the i18n seam.
func TestRenderFooter_RoutesHelpThroughTranslator(t *testing.T) {
	withFakeTUITranslator(t)
	a := &App{width: 120, height: 40}
	got := a.renderFooter()
	if !strings.Contains(got, "<TRANSLATED:tui.app.footer.help>") {
		t.Fatalf("renderFooter did not route help text through translator: %q", got)
	}
}
