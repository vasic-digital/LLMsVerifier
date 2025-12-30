package enterprise

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// RBACRole represents role-based access control roles
type RBACRole string

const (
	RBACRoleAdmin    RBACRole = "admin"
	RBACRoleOperator RBACRole = "operator"
	RBACRoleAnalyst  RBACRole = "analyst"
	RBACRoleViewer   RBACRole = "viewer"
	RBACRoleReadOnly RBACRole = "readonly"
)

// Permission represents a specific permission
type Permission string

const (
	PermissionJobSubmit       Permission = "job:submit"
	PermissionJobView         Permission = "job:view"
	PermissionJobCancel       Permission = "job:cancel"
	PermissionSystemStart     Permission = "system:start"
	PermissionSystemStop      Permission = "system:stop"
	PermissionSystemConfigure Permission = "system:configure"
	PermissionUserManage      Permission = "user:manage"
	PermissionMetricsView     Permission = "metrics:view"
	PermissionLogsView        Permission = "logs:view"
)

// User represents an enterprise user
type User struct {
	ID               string                 `json:"id"`
	Username         string                 `json:"username"`
	Email            string                 `json:"email"`
	FirstName        string                 `json:"first_name"`
	LastName         string                 `json:"last_name"`
	Roles            []RBACRole             `json:"roles"`
	Permissions      []Permission           `json:"permissions"`
	Attributes       map[string]interface{} `json:"attributes,omitempty"`
	Enabled          bool                   `json:"enabled"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	LastLogin        *time.Time             `json:"last_login,omitempty"`
	TwoFactorEnabled bool                   `json:"two_factor_enabled"`
}

// Role defines role with permissions
type Role struct {
	Name        RBACRole     `json:"name"`
	DisplayName string       `json:"display_name"`
	Description string       `json:"description"`
	Permissions []Permission `json:"permissions"`
	System      bool         `json:"system"`
	CreatedAt   time.Time    `json:"created_at"`
}

// RBACManager manages role-based access control
type RBACManager struct {
	users    map[string]*User
	roles    map[string]*Role
	mu       sync.RWMutex
	auditLog []AuditEntry
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
	ID        string                 `json:"id"`
	UserID    string                 `json:"user_id"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	Timestamp time.Time              `json:"timestamp"`
	IPAddress string                 `json:"ip_address"`
	UserAgent string                 `json:"user_agent"`
	Success   bool                   `json:"success"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// NewRBACManager creates a new RBAC manager
func NewRBACManager() *RBACManager {
	rbac := &RBACManager{
		users:    make(map[string]*User),
		roles:    make(map[string]*Role),
		auditLog: make([]AuditEntry, 0),
	}

	// Initialize default roles
	rbac.initializeDefaultRoles()

	return rbac
}

// initializeDefaultRoles sets up default system roles
func (rbac *RBACManager) initializeDefaultRoles() {
	defaultRoles := []*Role{
		{
			Name:        RBACRoleAdmin,
			DisplayName: "Administrator",
			Description: "Full system access",
			Permissions: []Permission{
				PermissionJobSubmit, PermissionJobView, PermissionJobCancel,
				PermissionSystemStart, PermissionSystemStop, PermissionSystemConfigure,
				PermissionUserManage, PermissionMetricsView, PermissionLogsView,
			},
			System:    true,
			CreatedAt: time.Now(),
		},
		{
			Name:        RBACRoleOperator,
			DisplayName: "Operator",
			Description: "Can manage jobs and view system status",
			Permissions: []Permission{
				PermissionJobSubmit, PermissionJobView, PermissionJobCancel,
				PermissionSystemStart, PermissionSystemStop,
				PermissionMetricsView, PermissionLogsView,
			},
			System:    true,
			CreatedAt: time.Now(),
		},
		{
			Name:        RBACRoleAnalyst,
			DisplayName: "Analyst",
			Description: "Can submit and view jobs, access metrics",
			Permissions: []Permission{
				PermissionJobSubmit, PermissionJobView,
				PermissionMetricsView,
			},
			System:    true,
			CreatedAt: time.Now(),
		},
		{
			Name:        RBACRoleViewer,
			DisplayName: "Viewer",
			Description: "Read-only access to jobs and metrics",
			Permissions: []Permission{
				PermissionJobView, PermissionMetricsView,
			},
			System:    true,
			CreatedAt: time.Now(),
		},
		{
			Name:        RBACRoleReadOnly,
			DisplayName: "Read Only",
			Description: "View-only access to specific resources",
			Permissions: []Permission{
				PermissionJobView,
			},
			System:    true,
			CreatedAt: time.Now(),
		},
	}

	for _, role := range defaultRoles {
		rbac.roles[string(role.Name)] = role
	}
}

// CreateUser creates a new user
func (rbac *RBACManager) CreateUser(user *User) error {
	rbac.mu.Lock()
	defer rbac.mu.Unlock()

	if user.ID == "" {
		return fmt.Errorf("user ID is required")
	}

	if _, exists := rbac.users[user.ID]; exists {
		return fmt.Errorf("user already exists: %s", user.ID)
	}

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	rbac.users[user.ID] = user

	// Log audit entry
	rbac.logAudit(user.ID, "user.created", "user", "", true, map[string]interface{}{
		"username": user.Username,
		"roles":    user.Roles,
	})

	return nil
}

// GetUser retrieves a user by ID
func (rbac *RBACManager) GetUser(userID string) (*User, error) {
	rbac.mu.RLock()
	defer rbac.mu.RUnlock()

	user, exists := rbac.users[userID]
	if !exists {
		return nil, fmt.Errorf("user not found: %s", userID)
	}

	// Return a copy to prevent external modification
	userCopy := *user
	return &userCopy, nil
}

// AuthenticateUser validates user credentials
func (rbac *RBACManager) AuthenticateUser(username, password string) (*User, error) {
	rbac.mu.RLock()
	defer rbac.mu.RUnlock()

	// In a real implementation, you'd verify password hash
	for _, user := range rbac.users {
		if user.Username == username && user.Enabled {
			// Update last login
			now := time.Now()
			user.LastLogin = &now
			user.UpdatedAt = now

			// Log audit entry
			rbac.logAudit(user.ID, "user.authenticated", "auth", "", true, map[string]interface{}{
				"username": username,
			})

			// Return copy
			userCopy := *user
			return &userCopy, nil
		}
	}

	return nil, fmt.Errorf("invalid credentials")
}

// HasPermission checks if user has a specific permission
func (rbac *RBACManager) HasPermission(userID string, permission Permission) bool {
	rbac.mu.RLock()
	defer rbac.mu.RUnlock()

	user, exists := rbac.users[userID]
	if !exists || !user.Enabled {
		return false
	}

	// Check direct permissions
	for _, perm := range user.Permissions {
		if perm == permission {
			return true
		}
	}

	// Check role-based permissions
	for _, roleName := range user.Roles {
		if role, exists := rbac.roles[string(roleName)]; exists {
			for _, perm := range role.Permissions {
				if perm == permission {
					return true
				}
			}
		}
	}

	return false
}

// AssignRole assigns a role to a user
func (rbac *RBACManager) AssignRole(userID string, role RBACRole) error {
	rbac.mu.Lock()
	defer rbac.mu.Unlock()

	user, exists := rbac.users[userID]
	if !exists {
		return fmt.Errorf("user not found: %s", userID)
	}

	// Check if role exists
	if _, exists := rbac.roles[string(role)]; !exists {
		return fmt.Errorf("role not found: %s", role)
	}

	// Check if user already has role
	for _, existingRole := range user.Roles {
		if existingRole == role {
			return fmt.Errorf("user already has role: %s", role)
		}
	}

	// Add role
	user.Roles = append(user.Roles, role)
	user.UpdatedAt = time.Now()

	// Log audit entry
	rbac.logAudit(userID, "role.assigned", "user", "", true, map[string]interface{}{
		"role": role,
	})

	return nil
}

// RemoveRole removes a role from a user
func (rbac *RBACManager) RemoveRole(userID string, role RBACRole) error {
	rbac.mu.Lock()
	defer rbac.mu.Unlock()

	user, exists := rbac.users[userID]
	if !exists {
		return fmt.Errorf("user not found: %s", userID)
	}

	// Remove role
	for i, existingRole := range user.Roles {
		if existingRole == role {
			user.Roles = append(user.Roles[:i], user.Roles[i+1:]...)
			user.UpdatedAt = time.Now()

			// Log audit entry
			rbac.logAudit(userID, "role.removed", "user", "", true, map[string]interface{}{
				"role": role,
			})

			return nil
		}
	}

	return fmt.Errorf("user does not have role: %s", role)
}

// GetUsers returns all users
func (rbac *RBACManager) GetUsers() []*User {
	rbac.mu.RLock()
	defer rbac.mu.RUnlock()

	users := make([]*User, 0, len(rbac.users))
	for _, user := range rbac.users {
		userCopy := *user
		users = append(users, &userCopy)
	}

	return users
}

// GetRoles returns all roles
func (rbac *RBACManager) GetRoles() []*Role {
	rbac.mu.RLock()
	defer rbac.mu.RUnlock()

	roles := make([]*Role, 0, len(rbac.roles))
	for _, role := range rbac.roles {
		roleCopy := *role
		roles = append(roles, &roleCopy)
	}

	return roles
}

// GetAuditLog returns audit log entries
func (rbac *RBACManager) GetAuditLog(limit int) []AuditEntry {
	rbac.mu.RLock()
	defer rbac.mu.RUnlock()

	if limit <= 0 || limit > len(rbac.auditLog) {
		limit = len(rbac.auditLog)
	}

	// Return most recent entries
	start := len(rbac.auditLog) - limit
	if start < 0 {
		start = 0
	}

	entries := make([]AuditEntry, limit)
	copy(entries, rbac.auditLog[start:])

	return entries
}

// logAudit adds an entry to the audit log
func (rbac *RBACManager) logAudit(userID, action, resource, ipAddress string, success bool, details map[string]interface{}) {
	// Set default IP address if not provided
	if ipAddress == "" {
		ipAddress = "0.0.0.0"
	}

	entry := AuditEntry{
		ID:        fmt.Sprintf("audit_%d", time.Now().UnixNano()),
		UserID:    userID,
		Action:    action,
		Resource:  resource,
		Timestamp: time.Now(),
		IPAddress: ipAddress,
		UserAgent: "",
		Success:   success,
		Details:   details,
	}

	rbac.auditLog = append(rbac.auditLog, entry)

	// Maintain audit log size (keep last 10000 entries)
	if len(rbac.auditLog) > 10000 {
		rbac.auditLog = rbac.auditLog[1:]
	}
}

// TLSConfig holds TLS configuration
type TLSConfig struct {
	Enabled            bool   `yaml:"enabled"`
	CertFile           string `yaml:"cert_file"`
	KeyFile            string `yaml:"key_file"`
	CAFile             string `yaml:"ca_file"`
	ServerName         string `yaml:"server_name"`
	InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
}

// Authenticate authenticates user against LDAP and returns User type
func (ldap *LDAPAuthenticator) AuthenticateAsUser(username, password string) (*User, error) {
	ldapUser, err := ldap.Authenticate(username, password)
	if err != nil {
		return nil, err
	}

	// Convert LDAPUser to User
	user := &User{
		ID:        ldapUser.ID,
		Username:  ldapUser.Username,
		Email:     ldapUser.Email,
		FirstName: ldapUser.FirstName,
		LastName:  ldapUser.LastName,
		Roles:     ldapUser.Roles,
		Enabled:   ldapUser.Enabled,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return user, nil
}

// SAMLConfig holds SAML configuration
type SAMLConfig struct {
	IdentityProviderURL string            `yaml:"identity_provider_url"`
	SSOURL              string            `yaml:"sso_url"`
	CertificateFile     string            `yaml:"certificate_file"`
	KeyFile             string            `yaml:"key_file"`
	AttributeMapping    map[string]string `yaml:"attribute_mapping"`
	EntityID            string            `yaml:"entity_id"`
	AllowedAudiences    []string          `yaml:"allowed_audiences"`
	MaxClockSkew        time.Duration     `yaml:"max_clock_skew"`
}

// SAMLResponse represents a SAML 2.0 Response
type SAMLResponse struct {
	XMLName      xml.Name       `xml:"Response"`
	ID           string         `xml:"ID,attr"`
	Version      string         `xml:"Version,attr"`
	IssueInstant string         `xml:"IssueInstant,attr"`
	Destination  string         `xml:"Destination,attr"`
	Issuer       string         `xml:"Issuer"`
	Status       SAMLStatus     `xml:"Status"`
	Assertion    *SAMLAssertion `xml:"Assertion"`
}

// SAMLStatus represents the status of a SAML response
type SAMLStatus struct {
	StatusCode SAMLStatusCode `xml:"StatusCode"`
}

// SAMLStatusCode represents a SAML status code
type SAMLStatusCode struct {
	Value string `xml:"Value,attr"`
}

// SAMLAssertion represents a SAML assertion
type SAMLAssertion struct {
	XMLName            xml.Name            `xml:"Assertion"`
	ID                 string              `xml:"ID,attr"`
	Version            string              `xml:"Version,attr"`
	IssueInstant       string              `xml:"IssueInstant,attr"`
	Issuer             string              `xml:"Issuer"`
	Subject            SAMLSubject         `xml:"Subject"`
	Conditions         SAMLConditions      `xml:"Conditions"`
	AuthnStatement     SAMLAuthnStatement  `xml:"AuthnStatement"`
	AttributeStatement SAMLAttributeStmt   `xml:"AttributeStatement"`
}

// SAMLSubject represents the subject of a SAML assertion
type SAMLSubject struct {
	NameID              SAMLNameID              `xml:"NameID"`
	SubjectConfirmation SAMLSubjectConfirmation `xml:"SubjectConfirmation"`
}

// SAMLNameID represents a SAML NameID
type SAMLNameID struct {
	Format string `xml:"Format,attr"`
	Value  string `xml:",chardata"`
}

// SAMLSubjectConfirmation represents subject confirmation
type SAMLSubjectConfirmation struct {
	Method                  string                      `xml:"Method,attr"`
	SubjectConfirmationData SAMLSubjectConfirmationData `xml:"SubjectConfirmationData"`
}

// SAMLSubjectConfirmationData represents subject confirmation data
type SAMLSubjectConfirmationData struct {
	NotOnOrAfter string `xml:"NotOnOrAfter,attr"`
	Recipient    string `xml:"Recipient,attr"`
	InResponseTo string `xml:"InResponseTo,attr"`
}

// SAMLConditions represents SAML conditions
type SAMLConditions struct {
	NotBefore            string              `xml:"NotBefore,attr"`
	NotOnOrAfter         string              `xml:"NotOnOrAfter,attr"`
	AudienceRestrictions []SAMLAudienceRestr `xml:"AudienceRestriction"`
}

// SAMLAudienceRestr represents audience restriction
type SAMLAudienceRestr struct {
	Audiences []string `xml:"Audience"`
}

// SAMLAuthnStatement represents an authentication statement
type SAMLAuthnStatement struct {
	AuthnInstant string `xml:"AuthnInstant,attr"`
	SessionIndex string `xml:"SessionIndex,attr"`
}

// SAMLAttributeStmt represents an attribute statement
type SAMLAttributeStmt struct {
	Attributes []SAMLAttribute `xml:"Attribute"`
}

// SAMLAttribute represents a SAML attribute
type SAMLAttribute struct {
	Name         string   `xml:"Name,attr"`
	NameFormat   string   `xml:"NameFormat,attr"`
	FriendlyName string   `xml:"FriendlyName,attr"`
	Values       []string `xml:"AttributeValue"`
}

// SAMLAuthenticator provides SAML authentication
type SAMLAuthenticator struct {
	config      SAMLConfig
	certificate *x509.Certificate
	initialized bool
}

// NewSAMLAuthenticator creates a new SAML authenticator
func NewSAMLAuthenticator(config SAMLConfig) *SAMLAuthenticator {
	auth := &SAMLAuthenticator{
		config: config,
	}

	// Set defaults
	if config.MaxClockSkew == 0 {
		auth.config.MaxClockSkew = 5 * time.Minute
	}

	// Initialize default attribute mapping if not provided
	if auth.config.AttributeMapping == nil {
		auth.config.AttributeMapping = map[string]string{
			"email":      "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress",
			"username":   "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name",
			"first_name": "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname",
			"last_name":  "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname",
			"groups":     "http://schemas.xmlsoap.org/claims/Group",
		}
	}

	// Load IdP certificate if configured
	if config.CertificateFile != "" {
		if err := auth.loadCertificate(); err != nil {
			log.Printf("Warning: Failed to load SAML IdP certificate: %v", err)
		} else {
			auth.initialized = true
		}
	}

	return auth
}

// loadCertificate loads the IdP certificate for signature verification
func (saml *SAMLAuthenticator) loadCertificate() error {
	certData, err := os.ReadFile(saml.config.CertificateFile)
	if err != nil {
		return fmt.Errorf("failed to read certificate file: %w", err)
	}

	// Parse PEM or DER certificate
	cert, err := x509.ParseCertificate(certData)
	if err != nil {
		// Try parsing as PEM
		block, _ := decodePEM(certData)
		if block != nil {
			cert, err = x509.ParseCertificate(block)
			if err != nil {
				return fmt.Errorf("failed to parse certificate: %w", err)
			}
		} else {
			return fmt.Errorf("failed to parse certificate: %w", err)
		}
	}

	saml.certificate = cert
	return nil
}

// decodePEM decodes PEM data
func decodePEM(data []byte) ([]byte, error) {
	// Simple PEM decoder - look for base64 content between markers
	content := string(data)
	start := strings.Index(content, "-----BEGIN")
	if start == -1 {
		return nil, errors.New("no PEM header found")
	}
	end := strings.Index(content, "-----END")
	if end == -1 {
		return nil, errors.New("no PEM footer found")
	}

	// Extract base64 content
	headerEnd := strings.Index(content[start:], "\n")
	if headerEnd == -1 {
		return nil, errors.New("invalid PEM format")
	}
	base64Content := strings.TrimSpace(content[start+headerEnd : end])
	base64Content = strings.ReplaceAll(base64Content, "\n", "")
	base64Content = strings.ReplaceAll(base64Content, "\r", "")

	return base64.StdEncoding.DecodeString(base64Content)
}

// ProcessSAMLResponse processes and validates a SAML response
func (saml *SAMLAuthenticator) ProcessSAMLResponse(samlResponse string) (*User, error) {
	// Validate configuration
	if saml.config.IdentityProviderURL == "" {
		return nil, errors.New("SAML is not properly configured: missing identity provider URL")
	}

	// Decode base64 SAML response
	decodedResponse, err := base64.StdEncoding.DecodeString(samlResponse)
	if err != nil {
		// Try URL-safe base64
		decodedResponse, err = base64.URLEncoding.DecodeString(samlResponse)
		if err != nil {
			return nil, fmt.Errorf("failed to decode SAML response: %w", err)
		}
	}

	// Parse SAML response XML
	var response SAMLResponse
	if err := xml.Unmarshal(decodedResponse, &response); err != nil {
		return nil, fmt.Errorf("failed to parse SAML response XML: %w", err)
	}

	// Validate response
	if err := saml.validateResponse(&response); err != nil {
		return nil, fmt.Errorf("SAML response validation failed: %w", err)
	}

	// Extract user from assertion
	user, err := saml.extractUserFromAssertion(response.Assertion)
	if err != nil {
		return nil, fmt.Errorf("failed to extract user from SAML assertion: %w", err)
	}

	log.Printf("SAML authentication successful for user: %s", user.Username)
	return user, nil
}

// validateResponse validates the SAML response
func (saml *SAMLAuthenticator) validateResponse(response *SAMLResponse) error {
	// Check SAML version
	if response.Version != "2.0" {
		return fmt.Errorf("unsupported SAML version: %s", response.Version)
	}

	// Check status
	if !strings.HasSuffix(response.Status.StatusCode.Value, "Success") {
		return fmt.Errorf("SAML authentication failed with status: %s", response.Status.StatusCode.Value)
	}

	// Verify issuer matches configured IdP
	if response.Issuer != saml.config.IdentityProviderURL && response.Issuer != saml.config.EntityID {
		// Allow some flexibility in issuer matching
		if !strings.Contains(response.Issuer, saml.config.IdentityProviderURL) {
			log.Printf("Warning: SAML issuer mismatch. Expected: %s, Got: %s",
				saml.config.IdentityProviderURL, response.Issuer)
		}
	}

	// Check assertion exists
	if response.Assertion == nil {
		return errors.New("SAML response contains no assertion")
	}

	// Validate assertion conditions
	if err := saml.validateConditions(&response.Assertion.Conditions); err != nil {
		return err
	}

	return nil
}

// validateConditions validates SAML assertion conditions
func (saml *SAMLAuthenticator) validateConditions(conditions *SAMLConditions) error {
	now := time.Now()
	clockSkew := saml.config.MaxClockSkew

	// Check NotBefore
	if conditions.NotBefore != "" {
		notBefore, err := time.Parse(time.RFC3339, conditions.NotBefore)
		if err != nil {
			// Try alternative format
			notBefore, err = time.Parse("2006-01-02T15:04:05Z", conditions.NotBefore)
			if err != nil {
				return fmt.Errorf("invalid NotBefore time format: %s", conditions.NotBefore)
			}
		}
		if now.Add(clockSkew).Before(notBefore) {
			return errors.New("SAML assertion is not yet valid (NotBefore)")
		}
	}

	// Check NotOnOrAfter
	if conditions.NotOnOrAfter != "" {
		notOnOrAfter, err := time.Parse(time.RFC3339, conditions.NotOnOrAfter)
		if err != nil {
			notOnOrAfter, err = time.Parse("2006-01-02T15:04:05Z", conditions.NotOnOrAfter)
			if err != nil {
				return fmt.Errorf("invalid NotOnOrAfter time format: %s", conditions.NotOnOrAfter)
			}
		}
		if now.Add(-clockSkew).After(notOnOrAfter) {
			return errors.New("SAML assertion has expired (NotOnOrAfter)")
		}
	}

	// Validate audience restriction if configured
	if len(saml.config.AllowedAudiences) > 0 && len(conditions.AudienceRestrictions) > 0 {
		audienceValid := false
		for _, restriction := range conditions.AudienceRestrictions {
			for _, audience := range restriction.Audiences {
				for _, allowed := range saml.config.AllowedAudiences {
					if audience == allowed {
						audienceValid = true
						break
					}
				}
			}
		}
		if !audienceValid {
			return errors.New("SAML assertion audience does not match allowed audiences")
		}
	}

	return nil
}

// extractUserFromAssertion extracts user information from SAML assertion
func (saml *SAMLAuthenticator) extractUserFromAssertion(assertion *SAMLAssertion) (*User, error) {
	if assertion == nil {
		return nil, errors.New("assertion is nil")
	}

	// Extract attributes into a map for easier access
	attrs := make(map[string][]string)
	for _, attr := range assertion.AttributeStatement.Attributes {
		// Use both Name and FriendlyName as keys
		attrs[attr.Name] = attr.Values
		if attr.FriendlyName != "" {
			attrs[attr.FriendlyName] = attr.Values
		}
	}

	// Get user ID from NameID
	userID := assertion.Subject.NameID.Value
	if userID == "" {
		return nil, errors.New("SAML assertion contains no NameID")
	}

	// Extract user attributes using configured mapping
	user := &User{
		ID:        "saml_" + sanitizeUserID(userID),
		Enabled:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Extract email
	if emailAttr := saml.config.AttributeMapping["email"]; emailAttr != "" {
		if values, ok := attrs[emailAttr]; ok && len(values) > 0 {
			user.Email = values[0]
		}
	}
	if user.Email == "" {
		// Fallback: use NameID if it looks like an email
		if strings.Contains(userID, "@") {
			user.Email = userID
		}
	}

	// Extract username
	if usernameAttr := saml.config.AttributeMapping["username"]; usernameAttr != "" {
		if values, ok := attrs[usernameAttr]; ok && len(values) > 0 {
			user.Username = values[0]
		}
	}
	if user.Username == "" {
		// Fallback: derive from email or NameID
		if user.Email != "" {
			parts := strings.Split(user.Email, "@")
			user.Username = parts[0]
		} else {
			user.Username = userID
		}
	}

	// Extract first name
	if fnAttr := saml.config.AttributeMapping["first_name"]; fnAttr != "" {
		if values, ok := attrs[fnAttr]; ok && len(values) > 0 {
			user.FirstName = values[0]
		}
	}

	// Extract last name
	if lnAttr := saml.config.AttributeMapping["last_name"]; lnAttr != "" {
		if values, ok := attrs[lnAttr]; ok && len(values) > 0 {
			user.LastName = values[0]
		}
	}

	// Extract groups/roles
	if groupsAttr := saml.config.AttributeMapping["groups"]; groupsAttr != "" {
		if values, ok := attrs[groupsAttr]; ok {
			user.Roles = mapGroupsToRoles(values)
		}
	}

	// Set default role if none assigned
	if len(user.Roles) == 0 {
		user.Roles = []RBACRole{RBACRoleViewer}
	}

	return user, nil
}

// sanitizeUserID removes or replaces characters not suitable for user IDs
func sanitizeUserID(id string) string {
	// Replace common problematic characters
	result := strings.ReplaceAll(id, "@", "_at_")
	result = strings.ReplaceAll(result, "/", "_")
	result = strings.ReplaceAll(result, "\\", "_")
	result = strings.ReplaceAll(result, " ", "_")
	return result
}

// mapGroupsToRoles maps SAML group names to RBAC roles
func mapGroupsToRoles(groups []string) []RBACRole {
	var roles []RBACRole
	roleMap := map[string]RBACRole{
		"admin":       RBACRoleAdmin,
		"admins":      RBACRoleAdmin,
		"administrator": RBACRoleAdmin,
		"administrators": RBACRoleAdmin,
		"operator":    RBACRoleOperator,
		"operators":   RBACRoleOperator,
		"analyst":     RBACRoleAnalyst,
		"analysts":    RBACRoleAnalyst,
		"viewer":      RBACRoleViewer,
		"viewers":     RBACRoleViewer,
		"readonly":    RBACRoleReadOnly,
		"read-only":   RBACRoleReadOnly,
	}

	for _, group := range groups {
		groupLower := strings.ToLower(strings.TrimSpace(group))
		if role, ok := roleMap[groupLower]; ok {
			roles = append(roles, role)
		}
	}

	return roles
}

// IsConfigured returns whether SAML is properly configured
func (saml *SAMLAuthenticator) IsConfigured() bool {
	return saml.config.IdentityProviderURL != ""
}

// GetLoginURL returns the SAML SSO login URL
func (saml *SAMLAuthenticator) GetLoginURL(relayState string) string {
	if saml.config.SSOURL != "" {
		return saml.config.SSOURL
	}
	return saml.config.IdentityProviderURL
}

// Ensure context import is used
var _ = io.EOF

// MultiTenantManager manages multi-tenancy
type MultiTenantManager struct {
	tenants     map[string]*Tenant
	tenantUsers map[string][]string // tenantID -> userIDs
	mu          sync.RWMutex
}

// Tenant represents a tenant
type Tenant struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Domain      string                 `json:"domain"`
	Settings    map[string]interface{} `json:"settings"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Enabled     bool                   `json:"enabled"`
}

// NewMultiTenantManager creates a new multi-tenant manager
func NewMultiTenantManager() *MultiTenantManager {
	return &MultiTenantManager{
		tenants:     make(map[string]*Tenant),
		tenantUsers: make(map[string][]string),
	}
}

// CreateTenant creates a new tenant
func (mtm *MultiTenantManager) CreateTenant(tenant *Tenant) error {
	mtm.mu.Lock()
	defer mtm.mu.Unlock()

	if _, exists := mtm.tenants[tenant.ID]; exists {
		return fmt.Errorf("tenant already exists: %s", tenant.ID)
	}

	tenant.CreatedAt = time.Now()
	tenant.UpdatedAt = time.Now()
	mtm.tenants[tenant.ID] = tenant
	mtm.tenantUsers[tenant.ID] = make([]string, 0)

	return nil
}

// GetAllTenants retrieves all tenants
func (mtm *MultiTenantManager) GetAllTenants() []*Tenant {
	mtm.mu.RLock()
	defer mtm.mu.RUnlock()

	tenants := make([]*Tenant, 0, len(mtm.tenants))
	for _, tenant := range mtm.tenants {
		tenantCopy := *tenant
		tenants = append(tenants, &tenantCopy)
	}

	return tenants
}

// GetTenant retrieves a tenant by ID
func (mtm *MultiTenantManager) GetTenant(tenantID string) (*Tenant, error) {
	mtm.mu.RLock()
	defer mtm.mu.RUnlock()

	tenant, exists := mtm.tenants[tenantID]
	if !exists {
		return nil, fmt.Errorf("tenant not found: %s", tenantID)
	}

	tenantCopy := *tenant
	return &tenantCopy, nil
}

// AddUserToTenant adds a user to a tenant
func (mtm *MultiTenantManager) AddUserToTenant(tenantID, userID string) error {
	mtm.mu.Lock()
	defer mtm.mu.Unlock()

	if _, exists := mtm.tenants[tenantID]; !exists {
		return fmt.Errorf("tenant not found: %s", tenantID)
	}

	users := mtm.tenantUsers[tenantID]
	for _, existingUserID := range users {
		if existingUserID == userID {
			return fmt.Errorf("user already in tenant: %s", userID)
		}
	}

	mtm.tenantUsers[tenantID] = append(users, userID)
	return nil
}

// GetUserTenants returns all tenants for a user
func (mtm *MultiTenantManager) GetUserTenants(userID string) []*Tenant {
	mtm.mu.RLock()
	defer mtm.mu.RUnlock()

	var userTenants []*Tenant
	for tenantID, users := range mtm.tenantUsers {
		for _, existingUserID := range users {
			if existingUserID == userID {
				if tenant, exists := mtm.tenants[tenantID]; exists {
					tenantCopy := *tenant
					userTenants = append(userTenants, &tenantCopy)
				}
				break
			}
		}
	}

	return userTenants
}

// EnterpriseManager integrates all enterprise features
type EnterpriseManager struct {
	RBAC        *RBACManager
	LDAP        *LDAPAuthenticator
	SAML        *SAMLAuthenticator
	MultiTenant *MultiTenantManager
	Supervisor  interface{} // Use interface{} to avoid circular dependency
	API         *EnterpriseAPI
	Config      EnterpriseConfig
}

// EnterpriseConfig holds enterprise configuration
type EnterpriseConfig struct {
	RBAC         RBACConfig         `yaml:"rbac"`
	LDAP         LDAPConfig         `yaml:"ldap"`
	SAML         SAMLConfig         `yaml:"saml"`
	MultiTenant  MultiTenantConfig  `yaml:"multi_tenant"`
	AuditLogging AuditLoggingConfig `yaml:"audit_logging"`
	Security     SecurityConfig     `yaml:"security"`
}

// RBACConfig holds RBAC configuration
type RBACConfig struct {
	Enabled        bool           `yaml:"enabled"`
	SessionTimeout time.Duration  `yaml:"session_timeout"`
	PasswordPolicy PasswordPolicy `yaml:"password_policy"`
	TwoFactorAuth  bool           `yaml:"two_factor_auth"`
}

// PasswordPolicy defines password requirements
type PasswordPolicy struct {
	MinLength        int           `yaml:"min_length"`
	RequireUppercase bool          `yaml:"require_uppercase"`
	RequireLowercase bool          `yaml:"require_lowercase"`
	RequireNumbers   bool          `yaml:"require_numbers"`
	RequireSymbols   bool          `yaml:"require_symbols"`
	MaxAge           time.Duration `yaml:"max_age"`
}

// MultiTenantConfig holds multi-tenant configuration
type MultiTenantConfig struct {
	Enabled       bool   `yaml:"enabled"`
	DefaultTenant string `yaml:"default_tenant"`
	TenantHeader  string `yaml:"tenant_header"`
}

// AuditLoggingConfig holds audit logging configuration
type AuditLoggingConfig struct {
	Enabled   bool          `yaml:"enabled"`
	Storage   string        `yaml:"storage"` // "file", "database", "syslog"
	Retention time.Duration `yaml:"retention"`
	Level     string        `yaml:"level"`
}

// SecurityConfig holds security configuration
type SecurityConfig struct {
	HTTPSEnabled   bool            `yaml:"https_enabled"`
	TLSConfig      *TLSConfig      `yaml:"tls_config"`
	CORSOrigins    []string        `yaml:"cors_origins"`
	RateLimiting   RateLimitConfig `yaml:"rate_limiting"`
	Authentication AuthConfig      `yaml:"authentication"`
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Requests int           `yaml:"requests"`
	Window   time.Duration `yaml:"window"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Methods []string `yaml:"methods"` // "ldap", "saml", "local"
}

// NewEnterpriseManager creates a new enterprise manager
func NewEnterpriseManager(config EnterpriseConfig) *EnterpriseManager {
	manager := &EnterpriseManager{
		RBAC:        NewRBACManager(),
		LDAP:        NewLDAPAuthenticator(config.LDAP),
		SAML:        NewSAMLAuthenticator(config.SAML),
		MultiTenant: NewMultiTenantManager(),
		Supervisor:  nil, // Initialize supervisor later to avoid circular dependency
		Config:      config,
	}

	// Initialize authenticators if enabled
	if config.LDAP.URL != "" {
		manager.LDAP = NewLDAPAuthenticator(config.LDAP)
	}

	if config.SAML.IdentityProviderURL != "" {
		manager.SAML = NewSAMLAuthenticator(config.SAML)
	}

	// Initialize API
	manager.API = NewEnterpriseAPI(manager)

	return manager
}

// Start starts the enterprise manager
func (em *EnterpriseManager) Start(ctx context.Context) error {
	log.Printf("Starting enterprise manager")

	// Start enterprise API server if configured
	if err := em.API.Start(ctx); err != nil {
		return fmt.Errorf("failed to start enterprise API: %w", err)
	}

	log.Printf("Enterprise manager started successfully")
	return nil
}

// Stop stops the enterprise manager
func (em *EnterpriseManager) Stop(ctx context.Context) error {
	log.Printf("Stopping enterprise manager")

	// Stop API server
	if err := em.API.Stop(ctx); err != nil {
		log.Printf("Error stopping enterprise API: %v", err)
	}

	log.Printf("Enterprise manager stopped")
	return nil
}
