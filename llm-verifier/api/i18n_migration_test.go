package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// fakeTranslator returns "<TRANSLATED:msg_id>" so tests can assert the
// sentinel without coupling to the English bundle text. Anti-bluff per
// CONST-035 / Article XI §11.9: a test asserting the original literal
// would silently pass if the call-site bypassed the translator.
type fakeTranslator struct{}

func (fakeTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}
func (fakeTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeTranslator installs the fakeTranslator, runs fn, restores the
// prior translator.
func withFakeTranslator(t *testing.T, fn func()) {
	t.Helper()
	prior := translator
	translator = fakeTranslator{}
	defer func() { translator = prior }()
	fn()
}

// TestComplianceChecker_DescriptionsRouteThroughTranslator drives the
// compliance checker against data that triggers every migrated violation
// path and asserts each Description + Remediation routes through the i18n
// seam. Paired-mutation anti-bluff: if a call-site regressed to a
// hardcoded English literal, the "<TRANSLATED:...>" sentinel would be
// absent and this fails.
func TestComplianceChecker_DescriptionsRouteThroughTranslator(t *testing.T) {
	withFakeTranslator(t, func() {
		cc := NewComplianceChecker(true, 1)

		// Trigger: sensitive field + PII (email) + no gdpr_consent.
		data := map[string]interface{}{
			"password": "x",
			"email":    "user@example.com",
		}
		res := cc.CheckDataCompliance(data)
		if len(res.Violations) == 0 {
			t.Fatal("expected violations from sensitive field + PII without consent")
		}
		assertAllRouted(t, "CheckDataCompliance", res.Violations)

		// Trigger: retention violation via expired created_at.
		expired := map[string]interface{}{
			"created_at": "2000-01-01T00:00:00Z",
		}
		resR := cc.CheckDataCompliance(expired)
		assertAllRouted(t, "retention", resR.Violations)
		if !containsTranslated(resR.Violations) {
			t.Error("retention violation Descriptions not routed")
		}

		// Trigger: data minimization (>20 fields).
		big := make(map[string]interface{}, 25)
		for i := 0; i < 25; i++ {
			big[string(rune('a'+i))+"field"] = i
		}
		resM := cc.CheckDataCompliance(big)
		assertAllRouted(t, "minimization", resM.Violations)

		// Trigger: request compliance — PII in query.
		req := httptest.NewRequest(http.MethodGet, "/x?contact=user@example.com", nil)
		req.Header.Set("password", "secret")
		resReq := cc.CheckRequestCompliance(req)
		if len(resReq.Violations) == 0 {
			t.Fatal("expected request-compliance violations")
		}
		assertAllRouted(t, "CheckRequestCompliance", resReq.Violations)
	})
}

// TestContentFilter_DescriptionsRouteThroughTranslator drives the content
// filter against banned-word, toxic-word, and pattern-match content and
// asserts each Description routes through the i18n seam. "forbidden" is a
// default banned word, "hate" a default toxicity word, and an email-like
// token matches a default banned pattern.
func TestContentFilter_DescriptionsRouteThroughTranslator(t *testing.T) {
	withFakeTranslator(t, func() {
		cf := NewContentFilter()

		res, err := cf.FilterContent("this forbidden hate user@example.com text")
		if err != nil {
			t.Fatalf("FilterContent: %v", err)
		}
		if len(res.Violations) < 3 {
			t.Fatalf("expected 3+ content violations, got %d", len(res.Violations))
		}
		var sawBanned, sawToxic, sawPattern bool
		for i, v := range res.Violations {
			if !strings.HasPrefix(v.Description, "<TRANSLATED:") {
				t.Errorf("content violation[%d] Description not routed: %q", i, v.Description)
			}
			switch v.Type {
			case "banned_word":
				sawBanned = true
			case "toxicity":
				sawToxic = true
			case "pattern_match":
				sawPattern = true
			}
		}
		if !sawBanned || !sawToxic || !sawPattern {
			t.Errorf("expected all three violation types, got banned=%v toxic=%v pattern=%v",
				sawBanned, sawToxic, sawPattern)
		}
	})
}

func assertAllRouted(t *testing.T, scope string, violations []ComplianceViolation) {
	t.Helper()
	for i, v := range violations {
		if !strings.HasPrefix(v.Description, "<TRANSLATED:") {
			t.Errorf("%s violation[%d] Description not routed: %q", scope, i, v.Description)
		}
		if v.Remediation != "" && !strings.HasPrefix(v.Remediation, "<TRANSLATED:") {
			t.Errorf("%s violation[%d] Remediation not routed: %q", scope, i, v.Remediation)
		}
	}
}

func containsTranslated(violations []ComplianceViolation) bool {
	for _, v := range violations {
		if strings.HasPrefix(v.Description, "<TRANSLATED:") {
			return true
		}
	}
	return false
}

// TestGetValidationErrors_RouteThroughTranslator drives GetValidationErrors
// (api/validation.go) through every validator tag branch and asserts each
// produced field-error message routes through the i18n seam. Paired-mutation
// anti-bluff per CONST-035 / CONST-046: if any branch regressed to a
// hardcoded English literal, the "<TRANSLATED:...>" sentinel would be absent
// and the corresponding sub-assertion fails.
func TestGetValidationErrors_RouteThroughTranslator(t *testing.T) {
	type tagStruct struct {
		Req   string `binding:"required"`
		Min   string `binding:"omitempty,min=5"`
		Max   string `binding:"omitempty,max=2"`
		Len   string `binding:"omitempty,len=4"`
		Email string `binding:"omitempty,email"`
		URL   string `binding:"omitempty,url"`
		OneOf string `binding:"omitempty,oneof=a b c"`
		Gt    int    `binding:"omitempty,gt=10"`
		Gte   int    `binding:"omitempty,gte=10"`
		Lt    int    `binding:"omitempty,lt=1"`
		Lte   int    `binding:"omitempty,lte=1"`
		Eq    int    `binding:"omitempty,eq=7"`
		Ne    int    `binding:"omitempty,ne=3"`
	}

	withFakeTranslator(t, func() {
		// Values deliberately chosen so every constraint is violated.
		bad := tagStruct{
			Min: "ab", Max: "abcdef", Len: "abcdefg",
			Email: "not-an-email", URL: "not a url", OneOf: "z",
			Gt: 1, Gte: 1, Lt: 99, Lte: 99, Eq: 1, Ne: 3,
		}
		err := ValidateRequest(bad)
		if err == nil {
			t.Fatal("expected validation errors from constraint-violating struct")
		}
		errs := GetValidationErrors(err)
		if len(errs) == 0 {
			t.Fatal("GetValidationErrors returned no field errors")
		}
		for field, msg := range errs {
			if !strings.HasPrefix(msg, "<TRANSLATED:") {
				t.Errorf("field %q error message not routed through translator: %q", field, msg)
			}
		}
	})
}

// TestErrorHandlers_RouteThroughTranslator exercises the REST error helpers
// in api/errors.go and asserts each user-facing error message routes through
// the i18n seam. Paired-mutation: a regressed call-site emitting a hardcoded
// literal drops the sentinel and fails the matching sub-assertion.
func TestErrorHandlers_RouteThroughTranslator(t *testing.T) {
	gin.SetMode(gin.TestMode)

	type errCase struct {
		name   string
		invoke func(*gin.Context)
	}
	cases := []errCase{
		{"unauthorized_default", func(c *gin.Context) { HandleUnauthorizedError(c, "") }},
		{"forbidden_default", func(c *gin.Context) { HandleForbiddenError(c, "") }},
		{"internal", func(c *gin.Context) { HandleInternalError(c, context.DeadlineExceeded) }},
		{"database", func(c *gin.Context) { HandleDatabaseError(c, context.DeadlineExceeded) }},
		{"rate_limit", func(c *gin.Context) { HandleRateLimitError(c, 30) }},
	}

	withFakeTranslator(t, func() {
		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				rec := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(rec)
				c.Request = httptest.NewRequest(http.MethodGet, "/x", nil)
				tc.invoke(c)
				// gin's JSON renderer HTML-escapes the sentinel's
				// leading "<" into a < unicode sequence, so we
				// assert on the "TRANSLATED:<id>" core — its presence
				// proves the message routed through the translator
				// rather than a hardcoded English literal.
				body := rec.Body.String()
				if !strings.Contains(body, "TRANSLATED:") {
					t.Errorf("%s: response body has no translator-routed message: %s", tc.name, body)
				}
			})
		}
	})
}
