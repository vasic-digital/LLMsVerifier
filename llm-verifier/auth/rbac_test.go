package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRBACManager(t *testing.T) {
	rbac := NewRBACManager()

	assert.NotNil(t, rbac)
	assert.NotNil(t, rbac.roles)
	assert.NotNil(t, rbac.permissions)

	// Should have default roles
	assert.Contains(t, rbac.roles, "admin")
	assert.Contains(t, rbac.roles, "editor")
	assert.Contains(t, rbac.roles, "viewer")
	assert.Contains(t, rbac.roles, "auditor")
}

func TestRBACManager_DefaultRoles(t *testing.T) {
	rbac := NewRBACManager()

	// Check admin role
	admin, err := rbac.GetRole("admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", admin.Name)
	assert.Equal(t, "Full system administrator", admin.Description)
	assert.Contains(t, admin.Permissions, "admin:system")

	// Check editor role
	editor, err := rbac.GetRole("editor")
	require.NoError(t, err)
	assert.Equal(t, "editor", editor.Name)
	assert.Contains(t, editor.Permissions, "read:models")
	assert.Contains(t, editor.Permissions, "write:models")

	// Check viewer role
	viewer, err := rbac.GetRole("viewer")
	require.NoError(t, err)
	assert.Equal(t, "viewer", viewer.Name)
	assert.Contains(t, viewer.Permissions, "read:models")
	assert.NotContains(t, viewer.Permissions, "write:models")
}

func TestRBACManager_DefaultPermissions(t *testing.T) {
	rbac := NewRBACManager()

	permissions := rbac.GetPermissions()

	assert.Contains(t, permissions, "read:models")
	assert.Contains(t, permissions, "write:models")
	assert.Contains(t, permissions, "delete:models")
	assert.Contains(t, permissions, "read:providers")
	assert.Contains(t, permissions, "write:providers")
	assert.Contains(t, permissions, "admin:system")
	assert.Contains(t, permissions, "read:audit")
}

func TestRBACManager_CreateRole(t *testing.T) {
	rbac := NewRBACManager()

	err := rbac.CreateRole("custom", "Custom role", []string{"read:models"})
	require.NoError(t, err)

	role, err := rbac.GetRole("custom")
	require.NoError(t, err)
	assert.Equal(t, "custom", role.Name)
	assert.Equal(t, "Custom role", role.Description)
	assert.Contains(t, role.Permissions, "read:models")
}

func TestRBACManager_CreateRole_AlreadyExists(t *testing.T) {
	rbac := NewRBACManager()

	err := rbac.CreateRole("admin", "Duplicate admin", []string{"read:models"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestRBACManager_CreateRole_InvalidPermission(t *testing.T) {
	rbac := NewRBACManager()

	err := rbac.CreateRole("custom", "Custom role", []string{"invalid:permission"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestRBACManager_DeleteRole(t *testing.T) {
	rbac := NewRBACManager()

	// Create a custom role first
	err := rbac.CreateRole("custom", "Custom role", []string{"read:models"})
	require.NoError(t, err)

	// Delete it
	err = rbac.DeleteRole("custom")
	require.NoError(t, err)

	// Should not exist anymore
	_, err = rbac.GetRole("custom")
	assert.Error(t, err)
}

func TestRBACManager_DeleteRole_BuiltIn(t *testing.T) {
	rbac := NewRBACManager()

	// Try to delete built-in roles
	err := rbac.DeleteRole("admin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete built-in role")

	err = rbac.DeleteRole("editor")
	assert.Error(t, err)

	err = rbac.DeleteRole("viewer")
	assert.Error(t, err)
}

func TestRBACManager_DeleteRole_NotExists(t *testing.T) {
	rbac := NewRBACManager()

	err := rbac.DeleteRole("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestRBACManager_AssignRole(t *testing.T) {
	rbac := NewRBACManager()

	client := &Client{
		Name:        "test-client",
		Permissions: []string{},
	}

	err := rbac.AssignRole(client, "viewer")
	require.NoError(t, err)

	assert.Contains(t, client.Permissions, "read:models")
	assert.Contains(t, client.Permissions, "read:providers")
}

func TestRBACManager_AssignRole_NoDuplicates(t *testing.T) {
	rbac := NewRBACManager()

	client := &Client{
		Name:        "test-client",
		Permissions: []string{"read:models"}, // Already has this permission
	}

	err := rbac.AssignRole(client, "viewer")
	require.NoError(t, err)

	// Should not have duplicate permissions
	count := 0
	for _, perm := range client.Permissions {
		if perm == "read:models" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestRBACManager_AssignRole_NotExists(t *testing.T) {
	rbac := NewRBACManager()

	client := &Client{
		Name:        "test-client",
		Permissions: []string{},
	}

	err := rbac.AssignRole(client, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestRBACManager_CheckPermission(t *testing.T) {
	rbac := NewRBACManager()

	client := &Client{
		Name:        "test-client",
		Permissions: []string{"read:models", "write:models"},
	}

	// Should have permission
	err := rbac.CheckPermission(client, "models", "read")
	assert.NoError(t, err)

	err = rbac.CheckPermission(client, "models", "write")
	assert.NoError(t, err)

	// Should not have permission
	err = rbac.CheckPermission(client, "models", "delete")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestRBACManager_CheckPermission_Admin(t *testing.T) {
	rbac := NewRBACManager()

	client := &Client{
		Name:        "admin-client",
		Permissions: []string{"admin:system"},
	}

	// Admin has all permissions
	err := rbac.CheckPermission(client, "models", "read")
	assert.NoError(t, err)

	err = rbac.CheckPermission(client, "models", "delete")
	assert.NoError(t, err)

	err = rbac.CheckPermission(client, "anything", "anything")
	assert.NoError(t, err)
}

func TestRBACManager_CheckPermission_Wildcard(t *testing.T) {
	rbac := NewRBACManager()

	client := &Client{
		Name:        "wildcard-client",
		Permissions: []string{"read:*"},
	}

	// Wildcard should match any read permission
	err := rbac.CheckPermission(client, "models", "read")
	assert.NoError(t, err)

	err = rbac.CheckPermission(client, "providers", "read")
	assert.NoError(t, err)
}

func TestRBACManager_GetRoles(t *testing.T) {
	rbac := NewRBACManager()

	roles := rbac.GetRoles()

	assert.NotEmpty(t, roles)
	assert.Contains(t, roles, "admin")
	assert.Contains(t, roles, "editor")
	assert.Contains(t, roles, "viewer")
	assert.Contains(t, roles, "auditor")
}

func TestRBACManager_GetRole(t *testing.T) {
	rbac := NewRBACManager()

	role, err := rbac.GetRole("admin")
	require.NoError(t, err)
	assert.Equal(t, "admin", role.Name)

	_, err = rbac.GetRole("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRBACManager_AddPermission(t *testing.T) {
	rbac := NewRBACManager()

	err := rbac.AddPermission("custom", "read", "")
	require.NoError(t, err)

	permissions := rbac.GetPermissions()
	assert.Contains(t, permissions, "read:custom")
}

func TestRBACManager_AddPermission_WithScope(t *testing.T) {
	rbac := NewRBACManager()

	err := rbac.AddPermission("custom", "read", "tenant")
	require.NoError(t, err)

	permissions := rbac.GetPermissions()
	assert.Contains(t, permissions, "read:custom:tenant")
}

func TestRBACManager_AddPermission_AlreadyExists(t *testing.T) {
	rbac := NewRBACManager()

	// Try to add existing permission
	err := rbac.AddPermission("models", "read", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestRBACManager_ValidatePermissions(t *testing.T) {
	rbac := NewRBACManager()

	// Valid permissions
	err := rbac.ValidatePermissions([]string{"read:models", "write:models"})
	assert.NoError(t, err)

	// Invalid permissions
	err = rbac.ValidatePermissions([]string{"read:models", "invalid:permission"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestRole_Struct(t *testing.T) {
	role := Role{
		Name:        "test-role",
		Description: "Test description",
		Permissions: []string{"read:models", "write:models"},
	}

	assert.Equal(t, "test-role", role.Name)
	assert.Equal(t, "Test description", role.Description)
	assert.Len(t, role.Permissions, 2)
}

func TestPermission_Struct(t *testing.T) {
	perm := Permission{
		Resource: "models",
		Action:   "read",
		Scope:    "tenant",
	}

	assert.Equal(t, "models", perm.Resource)
	assert.Equal(t, "read", perm.Action)
	assert.Equal(t, "tenant", perm.Scope)
}
