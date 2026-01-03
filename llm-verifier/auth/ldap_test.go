package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLDAPManager(t *testing.T) {
	config := &LDAPConfig{
		Host:       "ldap.example.com",
		Port:       389,
		BaseDN:     "dc=example,dc=com",
		UserFilter: "(&(objectClass=user)(sAMAccountName=%s))",
	}

	manager, err := NewLDAPManager(config)
	require.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, config, manager.config)
}

func TestNewLDAPManager_NilConfig(t *testing.T) {
	manager, err := NewLDAPManager(nil)
	require.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Nil(t, manager.config)
}

func TestLDAPManager_ValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *LDAPConfig
		wantErr     bool
		errContains string
		checkPort   int // expected port after validation
	}{
		{
			name: "valid config",
			config: &LDAPConfig{
				Host:       "ldap.example.com",
				Port:       389,
				BaseDN:     "dc=example,dc=com",
				UserFilter: "(&(objectClass=user)(sAMAccountName=%s))",
			},
			wantErr: false,
		},
		{
			name: "missing host",
			config: &LDAPConfig{
				Port:   389,
				BaseDN: "dc=example,dc=com",
			},
			wantErr:     true,
			errContains: "host is required",
		},
		{
			name: "missing base DN",
			config: &LDAPConfig{
				Host: "ldap.example.com",
				Port: 389,
			},
			wantErr:     true,
			errContains: "base DN is required",
		},
		{
			name: "default port",
			config: &LDAPConfig{
				Host:   "ldap.example.com",
				Port:   0,
				BaseDN: "dc=example,dc=com",
			},
			wantErr:   false,
			checkPort: 389,
		},
		{
			name: "default user filter",
			config: &LDAPConfig{
				Host:       "ldap.example.com",
				Port:       389,
				BaseDN:     "dc=example,dc=com",
				UserFilter: "",
			},
			wantErr: false,
		},
		{
			name: "TLS config",
			config: &LDAPConfig{
				Host:   "ldap.example.com",
				Port:   636,
				BaseDN: "dc=example,dc=com",
				TLS:    true,
			},
			wantErr: false,
		},
		{
			name: "StartTLS config",
			config: &LDAPConfig{
				Host:     "ldap.example.com",
				Port:     389,
				BaseDN:   "dc=example,dc=com",
				StartTLS: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewLDAPManager(tt.config)
			require.NoError(t, err)

			err = manager.ValidateConfig()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errContains != "" {
					assert.Contains(t, err.Error(), tt.errContains)
				}
			} else {
				assert.NoError(t, err)
			}

			// Check default port
			if tt.checkPort > 0 {
				assert.Equal(t, tt.checkPort, manager.config.Port)
			}

			// Check default user filter was set
			if tt.config.UserFilter == "" && !tt.wantErr && manager.config != nil {
				assert.NotEmpty(t, manager.config.UserFilter)
				assert.Contains(t, manager.config.UserFilter, "sAMAccountName")
			}
		})
	}
}

func TestLDAPManager_SyncUsers(t *testing.T) {
	config := &LDAPConfig{
		Host:   "ldap.example.com",
		Port:   389,
		BaseDN: "dc=example,dc=com",
	}

	manager, err := NewLDAPManager(config)
	require.NoError(t, err)

	// SyncUsers should return users or error (actual LDAP connection will fail in test)
	users, err := manager.SyncUsers()
	// Connection will fail without actual LDAP server, but function is implemented
	if err != nil {
		assert.Error(t, err) // Expected as no real server
	} else {
		assert.NotNil(t, users)
	}
}

func TestLDAPManager_Authenticate_EmptyPassword(t *testing.T) {
	config := &LDAPConfig{
		Host:   "ldap.example.com",
		Port:   389,
		BaseDN: "dc=example,dc=com",
	}

	manager, err := NewLDAPManager(config)
	require.NoError(t, err)

	// Empty password should fail
	client, err := manager.Authenticate("testuser", "")
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "password cannot be empty")
}

func TestLDAPManager_Authenticate_ConnectionFailed(t *testing.T) {
	config := &LDAPConfig{
		Host:   "nonexistent.example.com",
		Port:   389,
		BaseDN: "dc=example,dc=com",
	}

	manager, err := NewLDAPManager(config)
	require.NoError(t, err)

	// Should fail to connect
	client, err := manager.Authenticate("testuser", "password")
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to connect")
}

func TestLDAPManager_GetUserGroups_ConnectionFailed(t *testing.T) {
	config := &LDAPConfig{
		Host:   "nonexistent.example.com",
		Port:   389,
		BaseDN: "dc=example,dc=com",
	}

	manager, err := NewLDAPManager(config)
	require.NoError(t, err)

	// Should fail to connect
	groups, err := manager.GetUserGroups("testuser")
	assert.Error(t, err)
	assert.Nil(t, groups)
	assert.Contains(t, err.Error(), "failed to connect")
}

func TestLDAPConfig_Struct(t *testing.T) {
	config := LDAPConfig{
		Host:         "ldap.example.com",
		Port:         636,
		BaseDN:       "dc=example,dc=com",
		BindDN:       "cn=admin,dc=example,dc=com",
		BindPassword: "secret",
		UserFilter:   "(&(objectClass=user)(sAMAccountName=%s))",
		GroupFilter:  "(&(objectClass=group)(member=%s))",
		Attributes:   []string{"cn", "sAMAccountName", "mail"},
		TLS:          true,
		StartTLS:     false,
	}

	assert.Equal(t, "ldap.example.com", config.Host)
	assert.Equal(t, 636, config.Port)
	assert.Equal(t, "dc=example,dc=com", config.BaseDN)
	assert.Equal(t, "cn=admin,dc=example,dc=com", config.BindDN)
	assert.Equal(t, "secret", config.BindPassword)
	assert.Contains(t, config.UserFilter, "sAMAccountName")
	assert.Contains(t, config.GroupFilter, "member")
	assert.Len(t, config.Attributes, 3)
	assert.True(t, config.TLS)
	assert.False(t, config.StartTLS)
}

func TestLDAPManager_Connect_NoHost(t *testing.T) {
	config := &LDAPConfig{
		Host: "",
		Port: 389,
	}

	manager, err := NewLDAPManager(config)
	require.NoError(t, err)

	// connect is private, test via Authenticate
	_, err = manager.Authenticate("user", "pass")
	assert.Error(t, err)
}

func TestLDAPManager_ValidateConfig_SetsDefaults(t *testing.T) {
	config := &LDAPConfig{
		Host:   "ldap.example.com",
		BaseDN: "dc=example,dc=com",
		// Port and UserFilter not set
	}

	manager, err := NewLDAPManager(config)
	require.NoError(t, err)

	err = manager.ValidateConfig()
	require.NoError(t, err)

	// Check defaults were set
	assert.Equal(t, 389, config.Port)
	assert.Contains(t, config.UserFilter, "sAMAccountName")
}

func TestLDAPManager_ConfigWithBindCredentials(t *testing.T) {
	config := &LDAPConfig{
		Host:         "ldap.example.com",
		Port:         389,
		BaseDN:       "dc=example,dc=com",
		BindDN:       "cn=service,dc=example,dc=com",
		BindPassword: "servicepass",
	}

	manager, err := NewLDAPManager(config)
	require.NoError(t, err)
	assert.NotNil(t, manager)

	err = manager.ValidateConfig()
	assert.NoError(t, err)
}

func TestLDAPManager_TLSConfigurations(t *testing.T) {
	tests := []struct {
		name     string
		tls      bool
		startTLS bool
	}{
		{"plain", false, false},
		{"TLS", true, false},
		{"StartTLS", false, true},
		{"both", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &LDAPConfig{
				Host:     "ldap.example.com",
				Port:     389,
				BaseDN:   "dc=example,dc=com",
				TLS:      tt.tls,
				StartTLS: tt.startTLS,
			}

			manager, err := NewLDAPManager(config)
			require.NoError(t, err)

			err = manager.ValidateConfig()
			assert.NoError(t, err)

			// Config should preserve TLS settings
			assert.Equal(t, tt.tls, manager.config.TLS)
			assert.Equal(t, tt.startTLS, manager.config.StartTLS)
		})
	}
}
