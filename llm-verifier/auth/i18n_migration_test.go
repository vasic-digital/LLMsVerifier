package auth

import (
	"context"
	"strings"
	"testing"
)

// fakeAuthTranslator returns "<TRANSLATED:msg_id>" so tests can assert the
// i18n-routed sentinel without coupling to the English bundle text.
// Anti-bluff per CONST-035 / Article XI §11.9: a test asserting the original
// literal would silently pass if the call-site bypassed the translator.
type fakeAuthTranslator struct{}

func (fakeAuthTranslator) T(_ context.Context, id string, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}
func (fakeAuthTranslator) TPlural(_ context.Context, id string, _ int, _ map[string]any) (string, error) {
	return "<TRANSLATED:" + id + ">", nil
}

// withFakeAuthTranslator installs the fakeAuthTranslator, runs fn, then
// restores the prior translator.
func withFakeAuthTranslator(t *testing.T, fn func()) {
	t.Helper()
	prior := translator
	translator = fakeAuthTranslator{}
	defer func() { translator = prior }()
	fn()
}

// TestRBACRoleDescriptions_Routed proves every built-in RBAC role's
// Description field is i18n-routed rather than a hardcoded English literal.
// With the fake translator installed, each Description must carry the
// "<TRANSLATED:...>" prefix.
func TestRBACRoleDescriptions_Routed(t *testing.T) {
	withFakeAuthTranslator(t, func() {
		rbac := NewRBACManager()
		for name, role := range rbac.GetRoles() {
			if !strings.HasPrefix(role.Description, "<TRANSLATED:llmsverifier_auth_role_desc_") {
				t.Errorf("role %q description not i18n-routed: %q", name, role.Description)
			}
			// Name is an identifier token — must stay verbatim, NOT translated.
			if strings.HasPrefix(role.Name, "<TRANSLATED:") {
				t.Errorf("role %q name unexpectedly translated: %q", name, role.Name)
			}
		}
	})
}

// TestRBACRoleDescriptions_NoopDefault proves the production default
// (NoopTranslator) returns the messageID verbatim — the seam is wired but
// never breaks the build when no real backend is installed.
func TestRBACRoleDescriptions_NoopDefault(t *testing.T) {
	rbac := NewRBACManager()
	admin, err := rbac.GetRole("admin")
	if err != nil {
		t.Fatalf("GetRole(admin): %v", err)
	}
	if admin.Description != "llmsverifier_auth_role_desc_admin" {
		t.Errorf("NoopTranslator default did not return messageID verbatim: %q", admin.Description)
	}
}

// TestTr_PairedMutation is the §1.1 paired-mutation guard: it confirms tr()
// genuinely routes through the translator. If tr() were mutated to return its
// argument verbatim (bypassing the translator), this assertion fails because
// the fake translator's sentinel prefix would be absent.
func TestTr_PairedMutation(t *testing.T) {
	withFakeAuthTranslator(t, func() {
		got := tr("llmsverifier_auth_role_desc_admin")
		want := "<TRANSLATED:llmsverifier_auth_role_desc_admin>"
		if got != want {
			t.Errorf("tr() not routed through translator: got %q want %q", got, want)
		}
	})
}
