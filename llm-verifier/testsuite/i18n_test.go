package testsuite

import (
	"context"
	"strings"
	"testing"
)

// fakeTestsuiteTranslator returns "<TRANSLATED:msg_id>" for every message ID,
// proving that the TestSuiteBuilder description templates and the
// TestSuiteExecutor error/status messages route their user-facing text
// through the i18n seam rather than emitting hardcoded English literals
// (CONST-046 anti-bluff invariant per Article XI §11.9).
type fakeTestsuiteTranslator struct{}

func (fakeTestsuiteTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

func (fakeTestsuiteTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeTestsuiteTranslator swaps the package translator for the duration of
// a test and restores the prior backend afterwards.
func withFakeTestsuiteTranslator(t *testing.T) {
	t.Helper()
	prev := translator
	translator = fakeTestsuiteTranslator{}
	t.Cleanup(func() { translator = prev })
}

// TestTrSuite_RoutesThroughTranslator is the positive case: with the fake
// backend installed, trSuite() must emit the sentinel, not the message ID
// verbatim.
func TestTrSuite_RoutesThroughTranslator(t *testing.T) {
	withFakeTestsuiteTranslator(t)
	got := trSuite("testsuite.error.unsupported_case_type")
	if got != "<TRANSLATED:testsuite.error.unsupported_case_type>" {
		t.Fatalf("trSuite did not route through translator: got %q", got)
	}
}

// TestTrSuite_NoopReturnsMessageIDVerbatim is the paired-mutation case: with
// the default NoopTranslator, trSuite() returns the message ID unchanged.
func TestTrSuite_NoopReturnsMessageIDVerbatim(t *testing.T) {
	if got := trSuite("testsuite.error.no_client_basic"); got != "testsuite.error.no_client_basic" {
		t.Fatalf("NoopTranslator path: got %q; want messageID verbatim", got)
	}
}

// TestTrSuiteData_RoutesThroughTranslator verifies the parameterised helper
// used by description templates routes through the translator.
func TestTrSuiteData_RoutesThroughTranslator(t *testing.T) {
	withFakeTestsuiteTranslator(t)
	got := trSuiteData("testsuite.case.basic.description", map[string]any{"name": "smoke"})
	if got != "<TRANSLATED:testsuite.case.basic.description>" {
		t.Fatalf("trSuiteData did not route through translator: got %q", got)
	}
}

// TestTrSuiteData_NoopReturnsMessageIDVerbatim is the paired-mutation case for
// the parameterised helper.
func TestTrSuiteData_NoopReturnsMessageIDVerbatim(t *testing.T) {
	got := trSuiteData("testsuite.case.load.description", map[string]any{"users": 5, "duration": 30})
	if got != "testsuite.case.load.description" {
		t.Fatalf("NoopTranslator path: got %q; want messageID verbatim", got)
	}
}

// TestAddBasicTestCase_DescriptionRoutesThroughTranslator proves the basic
// test-case description goes through the i18n seam rather than emitting a
// hardcoded English literal.
func TestAddBasicTestCase_DescriptionRoutesThroughTranslator(t *testing.T) {
	withFakeTestsuiteTranslator(t)
	b := NewTestSuiteBuilder("suite", "desc")
	b.AddBasicTestCase("smoke", "say hi", []string{"hi"})
	suite := b.Build()
	if len(suite.TestCases) != 1 {
		t.Fatalf("expected 1 test case, got %d", len(suite.TestCases))
	}
	desc := suite.TestCases[0].Description
	if !strings.Contains(desc, "<TRANSLATED:testsuite.case.basic.description>") {
		t.Fatalf("AddBasicTestCase description not routed through translator: %q", desc)
	}
	if strings.Contains(desc, "Basic test case:") {
		t.Fatalf("AddBasicTestCase still emits hardcoded English literal: %q", desc)
	}
}

// TestAddLoadTestCase_DescriptionRoutesThroughTranslator proves the load
// test-case description goes through the i18n seam.
func TestAddLoadTestCase_DescriptionRoutesThroughTranslator(t *testing.T) {
	withFakeTestsuiteTranslator(t)
	b := NewTestSuiteBuilder("suite", "desc")
	b.AddLoadTestCase("load", 10, 60)
	suite := b.Build()
	if len(suite.TestCases) != 1 {
		t.Fatalf("expected 1 test case, got %d", len(suite.TestCases))
	}
	desc := suite.TestCases[0].Description
	if !strings.Contains(desc, "<TRANSLATED:testsuite.case.load.description>") {
		t.Fatalf("AddLoadTestCase description not routed through translator: %q", desc)
	}
	if strings.Contains(desc, "Load test:") {
		t.Fatalf("AddLoadTestCase still emits hardcoded English literal: %q", desc)
	}
}
