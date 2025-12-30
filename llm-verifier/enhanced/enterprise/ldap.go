package enterprise

import (
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-ldap/ldap/v3"
)

// LDAPConfig holds LDAP configuration
type LDAPConfig struct {
	Enabled         bool          `yaml:"enabled"`
	URL             string        `yaml:"url"`
	BaseDN          string        `yaml:"base_dn"`
	BindDN          string        `yaml:"bind_dn"`
	BindPassword    string        `yaml:"bind_password"`
	UserFilter      string        `yaml:"user_filter"`
	UserAttributes  []string      `yaml:"user_attributes"`
	GroupFilter     string        `yaml:"group_filter"`
	GroupAttributes []string      `yaml:"group_attributes"`
	TLS             TLSConfig     `yaml:"tls"`
	Timeout         time.Duration `yaml:"timeout"`
}

// LDAPUser represents an LDAP user
type LDAPUser struct {
	ID        string     `json:"id"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	FirstName string     `json:"first_name"`
	LastName  string     `json:"last_name"`
	Roles     []RBACRole `json:"roles"`
	Enabled   bool       `json:"enabled"`
}

// LDAPAuthenticator provides LDAP authentication
type LDAPAuthenticator struct {
	config LDAPConfig
}

// NewLDAPAuthenticator creates a new LDAP authenticator
func NewLDAPAuthenticator(config LDAPConfig) *LDAPAuthenticator {
	return &LDAPAuthenticator{
		config: config,
	}
}

// Authenticate authenticates user against LDAP
func (ldap *LDAPAuthenticator) Authenticate(username, password string) (*LDAPUser, error) {
	if !ldap.config.Enabled {
		return nil, fmt.Errorf("LDAP authentication is disabled")
	}

	// Connect to LDAP server
	conn, err := ldap.dial()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP server: %w", err)
	}
	defer conn.Close()

	// Bind with user credentials
	userDN := fmt.Sprintf(ldap.config.UserFilter, username)
	err = conn.Bind(userDN, password)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials: %w", err)
	}

	// Search for user details
	searchRequest := ldap.createUserSearchRequest(username)
	sr, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to search user: %w", err)
	}

	if len(sr.Entries) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	entry := sr.Entries[0]
	user, err := ldap.parseLDAPEntry(entry)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user entry: %w", err)
	}

	return user, nil
}

// dial establishes connection to LDAP server
func (auth *LDAPAuthenticator) dial() (*ldap.Conn, error) {
	var conn *ldap.Conn
	var err error

	if auth.config.TLS.Enabled {
		// TLS connection
		tlsConfig := &tls.Config{
			InsecureSkipVerify: auth.config.TLS.InsecureSkipVerify,
			ServerName:         auth.config.TLS.ServerName,
		}
		conn, err = ldap.DialTLS("tcp", auth.config.URL, tlsConfig)
	} else {
		// Plain connection
		conn, err = ldap.Dial("tcp", auth.config.URL)
	}

	if err != nil {
		return nil, err
	}

	conn.SetTimeout(auth.config.Timeout)

	return conn, nil
}

// createUserSearchRequest creates a search request for user details
func (auth *LDAPAuthenticator) createUserSearchRequest(username string) *ldap.SearchRequest {
	filter := fmt.Sprintf(auth.config.UserFilter, username)

	return ldap.NewSearchRequest(
		auth.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		int(auth.config.Timeout.Seconds()),
		false,
		filter,
		auth.config.UserAttributes,
		nil,
	)
}

// parseLDAPEntry parses LDAP entry into LDAPUser struct
func (ldap *LDAPAuthenticator) parseLDAPEntry(entry *ldap.Entry) (*LDAPUser, error) {
	user := &LDAPUser{
		ID:       entry.DN,
		Username: getLDAPAttribute(entry, "uid", "cn", "sAMAccountName"),
		Email:    getLDAPAttribute(entry, "mail", "email"),
		Enabled:  true, // Assume enabled unless disabled attribute exists
	}

	// Parse name attributes
	if givenName := getLDAPAttribute(entry, "givenName"); givenName != "" {
		user.FirstName = givenName
	}
	if sn := getLDAPAttribute(entry, "sn", "surname"); sn != "" {
		user.LastName = sn
	}

	// Parse group memberships for roles
	groups, err := ldap.getUserGroups(entry.DN)
	if err != nil {
		log.Printf("Warning: failed to get user groups: %v", err)
	} else {
		user.Roles = ldap.mapGroupsToRoles(groups)
	}

	return user, nil
}

// getUserGroups retrieves user's group memberships
func (auth *LDAPAuthenticator) getUserGroups(userDN string) ([]string, error) {
	conn, err := auth.dial()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// Bind with service account
	if auth.config.BindDN != "" {
		err = conn.Bind(auth.config.BindDN, auth.config.BindPassword)
		if err != nil {
			return nil, fmt.Errorf("failed to bind service account: %w", err)
		}
	}

	filter := fmt.Sprintf(auth.config.GroupFilter, userDN)
	searchRequest := ldap.NewSearchRequest(
		auth.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,
		int(auth.config.Timeout.Seconds()),
		false,
		filter,
		auth.config.GroupAttributes,
		nil,
	)

	sr, err := conn.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	var groups []string
	for _, entry := range sr.Entries {
		if groupName := getLDAPAttribute(entry, "cn", "name"); groupName != "" {
			groups = append(groups, groupName)
		}
	}

	return groups, nil
}

// mapGroupsToRoles maps LDAP groups to RBAC roles
func (ldap *LDAPAuthenticator) mapGroupsToRoles(groups []string) []RBACRole {
	var roles []RBACRole

	// Default role
	roles = append(roles, RBACRoleViewer)

	for _, group := range groups {
		switch strings.ToLower(group) {
		case "llm-admin", "administrators":
			roles = append(roles, RBACRoleAdmin)
		case "llm-analyst", "analysts":
			roles = append(roles, RBACRoleAnalyst)
		case "llm-operator", "operators":
			roles = append(roles, RBACRoleOperator)
		}
	}

	return roles
}

// getLDAPAttribute gets the first available attribute value
func getLDAPAttribute(entry *ldap.Entry, attrs ...string) string {
	for _, attr := range attrs {
		if values := entry.GetAttributeValues(attr); len(values) > 0 {
			return values[0]
		}
	}
	return ""
}

// ValidateLDAPConfig validates LDAP configuration
func ValidateLDAPConfig(config LDAPConfig) error {
	if !config.Enabled {
		return nil
	}

	if config.URL == "" {
		return fmt.Errorf("LDAP URL is required when LDAP is enabled")
	}

	if config.BaseDN == "" {
		return fmt.Errorf("LDAP BaseDN is required when LDAP is enabled")
	}

	if config.UserFilter == "" {
		return fmt.Errorf("LDAP UserFilter is required when LDAP is enabled")
	}

	return nil
}

// Note: MockLDAPUsers has been moved to ldap_mock_test.go for test-only access
