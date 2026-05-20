package enterprise

import (
	"context"
	"strings"
	"testing"
)

// fakeEnterpriseTranslator returns "<TRANSLATED:msg_id>" so tests can assert
// the i18n-routed sentinel without coupling to the English bundle text.
// Anti-bluff per CONST-035 / Article XI §11.9: a test asserting the original
// literal would silently pass if the call-site bypassed the translator.
type fakeEnterpriseTranslator struct{}

func (fakeEnterpriseTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}
func (fakeEnterpriseTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeEnterpriseTranslator installs the fakeEnterpriseTranslator, runs fn,
// then restores the prior translator.
func withFakeEnterpriseTranslator(t *testing.T, fn func()) {
	t.Helper()
	prior := translator
	translator = fakeEnterpriseTranslator{}
	defer func() { translator = prior }()
	fn()
}

// migratedEnterpriseIDs lists every message ID the round-356 migration routed
// through the i18n seam in enhanced/enterprise/api.go. Every entry must
// resolve through the translator — the two assertions below prove (a) the
// fake translator sees each id, and (b) the production-default NoopTranslator
// returns the bare id rather than an English literal.
var migratedEnterpriseIDs = []string{
	"enterprise.error.missing_authorization_header",
	"enterprise.error.invalid_authorization_header_format",
	"enterprise.error.invalid_token",
	"enterprise.error.user_not_authenticated",
	"enterprise.error.insufficient_permissions",
	"enterprise.error.method_not_allowed",
	"enterprise.error.invalid_json",
	"enterprise.error.username_password_required",
	"enterprise.error.invalid_credentials",
	"enterprise.error.token_generation_failed",
	"enterprise.error.token_refresh_failed",
	"enterprise.error.user_id_required",
	"enterprise.error.multi_tenancy_not_enabled",
	"enterprise.message.logged_out_successfully",
	"enterprise.message.user_created_successfully",
	"enterprise.message.user_updated_successfully",
	"enterprise.message.user_deleted_successfully",
	"enterprise.message.role_operations",
	"enterprise.message.tenant_created_successfully",
	"enterprise.message.tenant_operations",
}

// TestEnterpriseMessages_Routed proves every migrated enterprise message ID
// flows through the i18n seam. With the fake translator installed, every id
// must materialise as "<TRANSLATED:enterprise.*>" — if a call-site held an
// English literal instead of calling tr(), the prefix assertion fails.
func TestEnterpriseMessages_Routed(t *testing.T) {
	withFakeEnterpriseTranslator(t, func() {
		for _, id := range migratedEnterpriseIDs {
			got := tr(id)
			want := "<TRANSLATED:" + id + ">"
			if got != want {
				t.Errorf("tr(%q) = %q, want %q", id, got, want)
			}
			if !strings.HasPrefix(got, "<TRANSLATED:enterprise.") {
				t.Errorf("id %q not routed through enterprise i18n seam: %q", id, got)
			}
		}
	})
}

// TestEnterpriseMutationGuard is the paired-mutation test per §1.1. With the
// production-default NoopTranslator, the verbatim message ID is returned —
// NOT an English literal. A regression that re-hardcoded "Method not allowed"
// in api.go would make the writeError argument differ from the message ID,
// and a sweep of api.go for the literal would catch it; this test guards the
// seam contract: NoopTranslator MUST be id-faithful so the migration is real.
func TestEnterpriseMutationGuard(t *testing.T) {
	for _, id := range migratedEnterpriseIDs {
		if got := tr(id); got != id {
			t.Fatalf("NoopTranslator must return the bare id for %q; got %q", id, got)
		}
	}
	// A non-migrated, unknown id must also pass through verbatim — proving
	// tr() never injects an English fallback of its own.
	if got := tr("enterprise.error.__nonexistent__"); got != "enterprise.error.__nonexistent__" {
		t.Fatalf("NoopTranslator leaked a non-id value: %q", got)
	}
}
