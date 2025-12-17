package enterprise

import (
	"crypto/tls"
	"fmt"
	"log"

	"github.com/go-ldap/ldap/v3"
)

// LDAPConfig holds LDAP configuration
type LDAPConfig struct {
	Host         string   `json:"host"`
	Port         int      `json:"port"`
	UseSSL       bool     `json:"use_ssl"`
	BaseDN       string   `json:"base_dn"`
	BindUser     string   `json:"bind_user"`
	BindPassword string   `json:"bind_password"`
	UserFilter   string   `json:"user_filter"`
	GroupFilter  string   `json:"group_filter"`
	Attributes   []string `json:"attributes"`
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

// Authenticate authenticates a user against LDAP
func (la *LDAPAuthenticator) Authenticate(username, password string) (*LDAPUser, error) {
	if password == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}

	// Connect to LDAP server
	conn, err := la.connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP server: %w", err)
	}
	defer conn.Close()

	// Bind as the user to verify credentials
	userDN, err := la.findUserDN(conn, username)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Attempt to bind with user credentials
	err = conn.Bind(userDN, password)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// Get user details
	user, err := la.getUserDetails(conn, userDN)
	if err != nil {
		return nil, fmt.Errorf("failed to get user details: %w", err)
	}

	return user, nil
}

// connect establishes a connection to the LDAP server
func (la *LDAPAuthenticator) connect() (*ldap.Conn, error) {
	var l *ldap.Conn
	var err error

	address := fmt.Sprintf("%s:%d", la.config.Host, la.config.Port)

	if la.config.UseSSL {
		// Use LDAPS
		l, err = ldap.DialTLS("tcp", address, &tls.Config{InsecureSkipVerify: true})
	} else {
		// Use LDAP with StartTLS
		l, err = ldap.Dial("tcp", address)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to LDAP server: %w", err)
		}

		// Start TLS if not using LDAPS
		err = l.StartTLS(&tls.Config{InsecureSkipVerify: true})
		if err != nil {
			l.Close()
			return nil, fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP server: %w", err)
	}

	log.Printf("LDAP connection established to %s:%d", la.config.Host, la.config.Port)
	return l, nil
}

// findUserDN finds the DN for a username
func (la *LDAPAuthenticator) findUserDN(conn *ldap.Conn, username string) (string, error) {
	// Bind with service account if configured
	if la.config.BindUser != "" && la.config.BindPassword != "" {
		err := conn.Bind(la.config.BindUser, la.config.BindPassword)
		if err != nil {
			return "", fmt.Errorf("failed to bind with service account: %w", err)
		}
	}

	// Search for the user
	searchRequest := ldap.NewSearchRequest(
		la.config.BaseDN,
		ldap.ScopeWholeSubtree,
		ldap.NeverDerefAliases,
		0,     // SizeLimit
		0,     // TimeLimit
		false, // TypesOnly
		fmt.Sprintf(la.config.UserFilter, username), // Filter
		la.config.Attributes,                        // Attributes
		nil,                                         // Controls
	)

	searchResult, err := conn.Search(searchRequest)
	if err != nil {
		return "", fmt.Errorf("LDAP search failed: %w", err)
	}

	if len(searchResult.Entries) == 0 {
		return "", fmt.Errorf("user not found: %s", username)
	}

	if len(searchResult.Entries) > 1 {
		return "", fmt.Errorf("multiple users found for: %s", username)
	}

	return searchResult.Entries[0].DN, nil
}

// getUserDetails retrieves user details from LDAP
func (la *LDAPAuthenticator) getUserDetails(conn *ldap.Conn, userDN string) (*LDAPUser, error) {
	// Search for user details
	searchRequest := ldap.NewSearchRequest(
		userDN,
		ldap.ScopeBaseObject,
		ldap.NeverDerefAliases,
		0,                                        // SizeLimit
		0,                                        // TimeLimit
		false,                                    // TypesOnly
		"(objectClass=*)",                        // Filter
		append(la.config.Attributes, "memberOf"), // Attributes
		nil,                                      // Controls
	)

	searchResult, err := conn.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to get user details: %w", err)
	}

	if len(searchResult.Entries) == 0 {
		return nil, fmt.Errorf("user details not found")
	}

	entry := searchResult.Entries[0]

	// Extract user information
	user := &LDAPUser{
		DN:       entry.DN,
		Username: entry.GetAttributeValue("sAMAccountName"),
		Email:    entry.GetAttributeValue("mail"),
		FullName: entry.GetAttributeValue("displayName"),
		Groups:   entry.GetAttributeValues("memberOf"),
	}

	// Fallback to cn if username is empty
	if user.Username == "" {
		user.Username = entry.GetAttributeValue("cn")
	}

	return user, nil
}

// LDAPUser represents a user from LDAP
type LDAPUser struct {
	DN       string   `json:"dn"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	FullName string   `json:"full_name"`
	Groups   []string `json:"groups"`
}

// GetUserInfo retrieves user information without authentication
func (la *LDAPAuthenticator) GetUserInfo(username string) (*LDAPUser, error) {
	conn, err := la.connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LDAP server: %w", err)
	}
	defer conn.Close()

	userDN, err := la.findUserDN(conn, username)
	if err != nil {
		return nil, err
	}

	return la.getUserDetails(conn, userDN)
}

// ValidateConnection tests the LDAP connection
func (la *LDAPAuthenticator) ValidateConnection() error {
	conn, err := la.connect()
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	// Test bind with service account if configured
	if la.config.BindUser != "" && la.config.BindPassword != "" {
		err = conn.Bind(la.config.BindUser, la.config.BindPassword)
		if err != nil {
			return fmt.Errorf("failed to bind with service account: %w", err)
		}
	}

	log.Printf("LDAP connection validation successful")
	return nil
}
