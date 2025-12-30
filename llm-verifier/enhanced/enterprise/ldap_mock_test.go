package enterprise

// MockLDAPUsers represents a mock LDAP user directory for testing
var MockLDAPUsers = map[string]*LDAPUser{
	"admin": {
		ID:        "ldap_admin",
		Username:  "admin",
		Email:     "admin@company.com",
		FirstName: "Admin",
		LastName:  "User",
		Roles:     []RBACRole{RBACRoleAdmin},
		Enabled:   true,
	},
	"analyst": {
		ID:        "ldap_analyst",
		Username:  "analyst",
		Email:     "analyst@company.com",
		FirstName: "Analyst",
		LastName:  "User",
		Roles:     []RBACRole{RBACRoleAnalyst},
		Enabled:   true,
	},
	"viewer": {
		ID:        "ldap_viewer",
		Username:  "viewer",
		Email:     "viewer@company.com",
		FirstName: "Viewer",
		LastName:  "User",
		Roles:     []RBACRole{RBACRoleViewer},
		Enabled:   true,
	},
}
