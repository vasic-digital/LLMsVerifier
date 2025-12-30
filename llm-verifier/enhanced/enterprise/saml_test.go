package enterprise

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createValidSAMLResponse creates a valid SAML response for testing
func createValidSAMLResponse(t *testing.T, userEmail, username string) string {
	now := time.Now().UTC()
	notBefore := now.Add(-5 * time.Minute).Format(time.RFC3339)
	notOnOrAfter := now.Add(5 * time.Minute).Format(time.RFC3339)

	samlXML := `<?xml version="1.0" encoding="UTF-8"?>
<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
                xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
                ID="_response123"
                Version="2.0"
                IssueInstant="` + now.Format(time.RFC3339) + `"
                Destination="https://sp.example.com/acs">
    <saml:Issuer>https://idp.example.com</saml:Issuer>
    <samlp:Status>
        <samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
    </samlp:Status>
    <saml:Assertion ID="_assertion123" Version="2.0" IssueInstant="` + now.Format(time.RFC3339) + `">
        <saml:Issuer>https://idp.example.com</saml:Issuer>
        <saml:Subject>
            <saml:NameID Format="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">` + userEmail + `</saml:NameID>
            <saml:SubjectConfirmation Method="urn:oasis:names:tc:SAML:2.0:cm:bearer">
                <saml:SubjectConfirmationData NotOnOrAfter="` + notOnOrAfter + `" Recipient="https://sp.example.com/acs"/>
            </saml:SubjectConfirmation>
        </saml:Subject>
        <saml:Conditions NotBefore="` + notBefore + `" NotOnOrAfter="` + notOnOrAfter + `">
            <saml:AudienceRestriction>
                <saml:Audience>https://sp.example.com</saml:Audience>
            </saml:AudienceRestriction>
        </saml:Conditions>
        <saml:AuthnStatement AuthnInstant="` + now.Format(time.RFC3339) + `" SessionIndex="_session123">
            <saml:AuthnContext>
                <saml:AuthnContextClassRef>urn:oasis:names:tc:SAML:2.0:ac:classes:Password</saml:AuthnContextClassRef>
            </saml:AuthnContext>
        </saml:AuthnStatement>
        <saml:AttributeStatement>
            <saml:Attribute Name="http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress">
                <saml:AttributeValue>` + userEmail + `</saml:AttributeValue>
            </saml:Attribute>
            <saml:Attribute Name="http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name">
                <saml:AttributeValue>` + username + `</saml:AttributeValue>
            </saml:Attribute>
            <saml:Attribute Name="http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname">
                <saml:AttributeValue>Test</saml:AttributeValue>
            </saml:Attribute>
            <saml:Attribute Name="http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname">
                <saml:AttributeValue>User</saml:AttributeValue>
            </saml:Attribute>
            <saml:Attribute Name="http://schemas.xmlsoap.org/claims/Group">
                <saml:AttributeValue>analyst</saml:AttributeValue>
            </saml:Attribute>
        </saml:AttributeStatement>
    </saml:Assertion>
</samlp:Response>`

	return base64.StdEncoding.EncodeToString([]byte(samlXML))
}

// TestProcessSAMLResponse_ValidResponse tests processing a valid SAML response
func TestProcessSAMLResponse_ValidResponse(t *testing.T) {
	config := SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
		EntityID:            "https://sp.example.com",
		AllowedAudiences:    []string{"https://sp.example.com"},
	}
	saml := NewSAMLAuthenticator(config)

	samlResponse := createValidSAMLResponse(t, "testuser@example.com", "testuser")

	user, err := saml.ProcessSAMLResponse(samlResponse)
	require.NoError(t, err)
	require.NotNil(t, user)

	assert.Equal(t, "testuser@example.com", user.Email)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "Test", user.FirstName)
	assert.Equal(t, "User", user.LastName)
	assert.True(t, user.Enabled)
	assert.Contains(t, user.Roles, RBACRoleAnalyst)
}

// TestProcessSAMLResponse_MissingConfig tests SAML with missing configuration
func TestProcessSAMLResponse_MissingConfig(t *testing.T) {
	config := SAMLConfig{
		// No IdentityProviderURL
	}
	saml := NewSAMLAuthenticator(config)

	samlResponse := createValidSAMLResponse(t, "test@example.com", "testuser")

	user, err := saml.ProcessSAMLResponse(samlResponse)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "not properly configured")
}

// TestProcessSAMLResponse_InvalidBase64 tests SAML with invalid base64
func TestProcessSAMLResponse_InvalidBase64(t *testing.T) {
	config := SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
	}
	saml := NewSAMLAuthenticator(config)

	user, err := saml.ProcessSAMLResponse("not-valid-base64!!!")
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "decode")
}

// TestProcessSAMLResponse_InvalidXML tests SAML with invalid XML
func TestProcessSAMLResponse_InvalidXML(t *testing.T) {
	config := SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
	}
	saml := NewSAMLAuthenticator(config)

	invalidXML := base64.StdEncoding.EncodeToString([]byte("<invalid>xml"))

	user, err := saml.ProcessSAMLResponse(invalidXML)
	assert.Error(t, err)
	assert.Nil(t, user)
}

// TestProcessSAMLResponse_FailedStatus tests SAML with failed status
func TestProcessSAMLResponse_FailedStatus(t *testing.T) {
	config := SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
	}
	saml := NewSAMLAuthenticator(config)

	failedSAML := `<?xml version="1.0"?>
<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" Version="2.0">
    <saml:Issuer xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">https://idp.example.com</saml:Issuer>
    <samlp:Status>
        <samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Requester"/>
    </samlp:Status>
</samlp:Response>`

	encoded := base64.StdEncoding.EncodeToString([]byte(failedSAML))

	user, err := saml.ProcessSAMLResponse(encoded)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "failed")
}

// TestProcessSAMLResponse_WrongVersion tests SAML with wrong version
func TestProcessSAMLResponse_WrongVersion(t *testing.T) {
	config := SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
	}
	saml := NewSAMLAuthenticator(config)

	wrongVersionSAML := `<?xml version="1.0"?>
<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" Version="1.0">
    <saml:Issuer xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">https://idp.example.com</saml:Issuer>
    <samlp:Status>
        <samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
    </samlp:Status>
</samlp:Response>`

	encoded := base64.StdEncoding.EncodeToString([]byte(wrongVersionSAML))

	user, err := saml.ProcessSAMLResponse(encoded)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "unsupported SAML version")
}

// TestProcessSAMLResponse_NoAssertion tests SAML without assertion
func TestProcessSAMLResponse_NoAssertion(t *testing.T) {
	config := SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
	}
	saml := NewSAMLAuthenticator(config)

	noAssertionSAML := `<?xml version="1.0"?>
<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol" Version="2.0">
    <saml:Issuer xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion">https://idp.example.com</saml:Issuer>
    <samlp:Status>
        <samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
    </samlp:Status>
</samlp:Response>`

	encoded := base64.StdEncoding.EncodeToString([]byte(noAssertionSAML))

	user, err := saml.ProcessSAMLResponse(encoded)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "no assertion")
}

// TestProcessSAMLResponse_ExpiredAssertion tests SAML with expired assertion
func TestProcessSAMLResponse_ExpiredAssertion(t *testing.T) {
	config := SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
	}
	saml := NewSAMLAuthenticator(config)

	expired := time.Now().Add(-1 * time.Hour).UTC()
	notBefore := expired.Add(-5 * time.Minute).Format(time.RFC3339)
	notOnOrAfter := expired.Format(time.RFC3339) // Already expired

	expiredSAML := `<?xml version="1.0"?>
<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
                xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
                Version="2.0">
    <saml:Issuer>https://idp.example.com</saml:Issuer>
    <samlp:Status>
        <samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
    </samlp:Status>
    <saml:Assertion Version="2.0">
        <saml:Issuer>https://idp.example.com</saml:Issuer>
        <saml:Subject>
            <saml:NameID>test@example.com</saml:NameID>
        </saml:Subject>
        <saml:Conditions NotBefore="` + notBefore + `" NotOnOrAfter="` + notOnOrAfter + `"/>
    </saml:Assertion>
</samlp:Response>`

	encoded := base64.StdEncoding.EncodeToString([]byte(expiredSAML))

	user, err := saml.ProcessSAMLResponse(encoded)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "expired")
}

// TestProcessSAMLResponse_NotYetValid tests SAML that's not yet valid
func TestProcessSAMLResponse_NotYetValid(t *testing.T) {
	config := SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
		MaxClockSkew:        1 * time.Minute, // Reduce clock skew for testing
	}
	saml := NewSAMLAuthenticator(config)

	future := time.Now().Add(1 * time.Hour).UTC()
	notBefore := future.Format(time.RFC3339)
	notOnOrAfter := future.Add(5 * time.Minute).Format(time.RFC3339)

	futureSAML := `<?xml version="1.0"?>
<samlp:Response xmlns:samlp="urn:oasis:names:tc:SAML:2.0:protocol"
                xmlns:saml="urn:oasis:names:tc:SAML:2.0:assertion"
                Version="2.0">
    <saml:Issuer>https://idp.example.com</saml:Issuer>
    <samlp:Status>
        <samlp:StatusCode Value="urn:oasis:names:tc:SAML:2.0:status:Success"/>
    </samlp:Status>
    <saml:Assertion Version="2.0">
        <saml:Issuer>https://idp.example.com</saml:Issuer>
        <saml:Subject>
            <saml:NameID>test@example.com</saml:NameID>
        </saml:Subject>
        <saml:Conditions NotBefore="` + notBefore + `" NotOnOrAfter="` + notOnOrAfter + `"/>
    </saml:Assertion>
</samlp:Response>`

	encoded := base64.StdEncoding.EncodeToString([]byte(futureSAML))

	user, err := saml.ProcessSAMLResponse(encoded)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "NotBefore")
}

// TestProcessSAMLResponse_WrongAudience tests SAML with wrong audience
func TestProcessSAMLResponse_WrongAudience(t *testing.T) {
	config := SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
		AllowedAudiences:    []string{"https://other-sp.example.com"},
	}
	saml := NewSAMLAuthenticator(config)

	samlResponse := createValidSAMLResponse(t, "test@example.com", "testuser")

	user, err := saml.ProcessSAMLResponse(samlResponse)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "audience")
}

// TestMapGroupsToRoles tests group to role mapping
func TestMapGroupsToRoles(t *testing.T) {
	testCases := []struct {
		name     string
		groups   []string
		expected []RBACRole
	}{
		{
			name:     "Admin group",
			groups:   []string{"admin"},
			expected: []RBACRole{RBACRoleAdmin},
		},
		{
			name:     "Multiple groups",
			groups:   []string{"admin", "analyst"},
			expected: []RBACRole{RBACRoleAdmin, RBACRoleAnalyst},
		},
		{
			name:     "Case insensitive",
			groups:   []string{"ADMIN", "Analyst"},
			expected: []RBACRole{RBACRoleAdmin, RBACRoleAnalyst},
		},
		{
			name:     "Unknown groups ignored",
			groups:   []string{"unknown", "admin", "other"},
			expected: []RBACRole{RBACRoleAdmin},
		},
		{
			name:     "Plural forms",
			groups:   []string{"admins", "operators", "viewers"},
			expected: []RBACRole{RBACRoleAdmin, RBACRoleOperator, RBACRoleViewer},
		},
		{
			name:     "Empty groups",
			groups:   []string{},
			expected: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			roles := mapGroupsToRoles(tc.groups)
			assert.Equal(t, tc.expected, roles)
		})
	}
}

// TestSanitizeUserID tests user ID sanitization
func TestSanitizeUserID(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"user@example.com", "user_at_example.com"},
		{"user/name", "user_name"},
		{"user\\name", "user_name"},
		{"user name", "user_name"},
		{"simple", "simple"},
		{"complex@user/with\\spaces name", "complex_at_user_with_spaces_name"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := sanitizeUserID(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestSAMLAuthenticator_IsConfigured tests configuration check
func TestSAMLAuthenticator_IsConfigured(t *testing.T) {
	t.Run("Configured", func(t *testing.T) {
		config := SAMLConfig{
			IdentityProviderURL: "https://idp.example.com",
		}
		saml := NewSAMLAuthenticator(config)
		assert.True(t, saml.IsConfigured())
	})

	t.Run("Not configured", func(t *testing.T) {
		config := SAMLConfig{}
		saml := NewSAMLAuthenticator(config)
		assert.False(t, saml.IsConfigured())
	})
}

// TestSAMLAuthenticator_GetLoginURL tests login URL generation
func TestSAMLAuthenticator_GetLoginURL(t *testing.T) {
	t.Run("With SSO URL", func(t *testing.T) {
		config := SAMLConfig{
			IdentityProviderURL: "https://idp.example.com",
			SSOURL:              "https://idp.example.com/sso",
		}
		saml := NewSAMLAuthenticator(config)
		assert.Equal(t, "https://idp.example.com/sso", saml.GetLoginURL(""))
	})

	t.Run("Without SSO URL", func(t *testing.T) {
		config := SAMLConfig{
			IdentityProviderURL: "https://idp.example.com",
		}
		saml := NewSAMLAuthenticator(config)
		assert.Equal(t, "https://idp.example.com", saml.GetLoginURL(""))
	})
}

// TestDefaultAttributeMapping tests default attribute mapping
func TestDefaultAttributeMapping(t *testing.T) {
	config := SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
		// No AttributeMapping provided
	}
	saml := NewSAMLAuthenticator(config)

	assert.NotNil(t, saml.config.AttributeMapping)
	assert.Equal(t, "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress", saml.config.AttributeMapping["email"])
	assert.Equal(t, "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name", saml.config.AttributeMapping["username"])
}

// TestDefaultMaxClockSkew tests default clock skew setting
func TestDefaultMaxClockSkew(t *testing.T) {
	config := SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
		// No MaxClockSkew provided
	}
	saml := NewSAMLAuthenticator(config)

	assert.Equal(t, 5*time.Minute, saml.config.MaxClockSkew)
}

// TestExtractUserFromAssertion_NoNameID tests extraction with missing NameID
func TestExtractUserFromAssertion_NoNameID(t *testing.T) {
	config := SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
	}
	saml := NewSAMLAuthenticator(config)

	assertion := &SAMLAssertion{
		Subject: SAMLSubject{
			NameID: SAMLNameID{Value: ""}, // Empty NameID
		},
	}

	user, err := saml.extractUserFromAssertion(assertion)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "NameID")
}

// TestExtractUserFromAssertion_NilAssertion tests extraction with nil assertion
func TestExtractUserFromAssertion_NilAssertion(t *testing.T) {
	config := SAMLConfig{
		IdentityProviderURL: "https://idp.example.com",
	}
	saml := NewSAMLAuthenticator(config)

	user, err := saml.extractUserFromAssertion(nil)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "nil")
}
