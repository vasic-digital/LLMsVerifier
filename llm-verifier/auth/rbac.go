// Package auth provides role-based access control (RBAC)
package auth

import (
	"fmt"
	"strings"
)

// Role represents a user role with associated permissions
type Role struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// Permission represents a granular permission
type Permission struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
	Scope    string `json:"scope,omitempty"` // global, tenant, user
}

// RBACManager manages role-based access control
type RBACManager struct {
	roles       map[string]*Role
	permissions map[string]*Permission
}

// NewRBACManager creates a new RBAC manager
func NewRBACManager() *RBACManager {
	rbac := &RBACManager{
		roles:       make(map[string]*Role),
		permissions: make(map[string]*Permission),
	}

	// Initialize default roles
	rbac.initializeDefaultRoles()

	return rbac
}

// initializeDefaultRoles sets up default roles and permissions
func (rbac *RBACManager) initializeDefaultRoles() {
	// Define permissions
	rbac.permissions["read:models"] = &Permission{Resource: "models", Action: "read"}
	rbac.permissions["write:models"] = &Permission{Resource: "models", Action: "write"}
	rbac.permissions["delete:models"] = &Permission{Resource: "models", Action: "delete"}
	rbac.permissions["read:providers"] = &Permission{Resource: "providers", Action: "read"}
	rbac.permissions["write:providers"] = &Permission{Resource: "providers", Action: "write"}
	rbac.permissions["admin:system"] = &Permission{Resource: "system", Action: "admin"}
	rbac.permissions["read:audit"] = &Permission{Resource: "audit", Action: "read"}

	// Define roles
	rbac.roles["admin"] = &Role{
		Name:        "admin",
		Description: "Full system administrator",
		Permissions: []string{"read:models", "write:models", "delete:models", "read:providers", "write:providers", "admin:system", "read:audit"},
	}

	rbac.roles["editor"] = &Role{
		Name:        "editor",
		Description: "Can read and modify models and providers",
		Permissions: []string{"read:models", "write:models", "read:providers", "write:providers"},
	}

	rbac.roles["viewer"] = &Role{
		Name:        "viewer",
		Description: "Read-only access to models and providers",
		Permissions: []string{"read:models", "read:providers"},
	}

	rbac.roles["auditor"] = &Role{
		Name:        "auditor",
		Description: "Can view audit logs",
		Permissions: []string{"read:audit"},
	}
}

// CreateRole creates a new custom role
func (rbac *RBACManager) CreateRole(name, description string, permissions []string) error {
	if _, exists := rbac.roles[name]; exists {
		return fmt.Errorf("role %s already exists", name)
	}

	// Validate permissions exist
	for _, perm := range permissions {
		if _, exists := rbac.permissions[perm]; !exists {
			return fmt.Errorf("permission %s does not exist", perm)
		}
	}

	rbac.roles[name] = &Role{
		Name:        name,
		Description: description,
		Permissions: permissions,
	}

	return nil
}

// DeleteRole removes a role
func (rbac *RBACManager) DeleteRole(name string) error {
	if name == "admin" || name == "editor" || name == "viewer" {
		return fmt.Errorf("cannot delete built-in role: %s", name)
	}

	if _, exists := rbac.roles[name]; !exists {
		return fmt.Errorf("role %s does not exist", name)
	}

	delete(rbac.roles, name)
	return nil
}

// AssignRole assigns a role to a client
func (rbac *RBACManager) AssignRole(client *Client, roleName string) error {
	role, exists := rbac.roles[roleName]
	if !exists {
		return fmt.Errorf("role %s does not exist", roleName)
	}

	// Add role permissions to client (avoid duplicates)
	permissionMap := make(map[string]bool)
	for _, perm := range client.Permissions {
		permissionMap[perm] = true
	}

	for _, perm := range role.Permissions {
		if !permissionMap[perm] {
			client.Permissions = append(client.Permissions, perm)
			permissionMap[perm] = true
		}
	}

	return nil
}

// CheckPermission checks if a client has a specific permission
func (rbac *RBACManager) CheckPermission(client *Client, resource, action string) error {
	requiredPerm := fmt.Sprintf("%s:%s", action, resource)

	// Check direct permissions
	for _, perm := range client.Permissions {
		if perm == requiredPerm || perm == "admin:system" {
			return nil
		}

		// Check wildcard permissions
		if strings.HasSuffix(perm, ":*") {
			resourcePrefix := strings.TrimSuffix(perm, ":*")
			if strings.HasPrefix(requiredPerm, resourcePrefix+":") {
				return nil
			}
		}
	}

	return fmt.Errorf("permission denied: %s requires %s", client.Name, requiredPerm)
}

// GetRoles returns all available roles
func (rbac *RBACManager) GetRoles() map[string]*Role {
	return rbac.roles
}

// GetRole returns a specific role
func (rbac *RBACManager) GetRole(name string) (*Role, error) {
	role, exists := rbac.roles[name]
	if !exists {
		return nil, fmt.Errorf("role %s not found", name)
	}
	return role, nil
}

// GetPermissions returns all available permissions
func (rbac *RBACManager) GetPermissions() map[string]*Permission {
	return rbac.permissions
}

// AddPermission adds a new custom permission
func (rbac *RBACManager) AddPermission(resource, action, scope string) error {
	permKey := fmt.Sprintf("%s:%s", action, resource)
	if scope != "" {
		permKey = fmt.Sprintf("%s:%s:%s", action, resource, scope)
	}

	if _, exists := rbac.permissions[permKey]; exists {
		return fmt.Errorf("permission %s already exists", permKey)
	}

	rbac.permissions[permKey] = &Permission{
		Resource: resource,
		Action:   action,
		Scope:    scope,
	}

	return nil
}

// ValidatePermissions validates that all permissions in a list exist
func (rbac *RBACManager) ValidatePermissions(permissions []string) error {
	for _, perm := range permissions {
		if _, exists := rbac.permissions[perm]; !exists {
			return fmt.Errorf("permission %s does not exist", perm)
		}
	}
	return nil
}
