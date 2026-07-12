package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"digital.vasic.llmsverifier/database"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// HXC-135 guard: publish the six CONST-040 capability flags (MCP, LSP, ACP,
// RAG/embedding-retrieval, Skills, Plugins) in the verifier's own live model
// API responses.
//
// RED baseline (§11.4.115, established by code inspection this session,
// captured in the HXC-135 commit): before this change, ListModelsHandler /
// GetModelHandler / VerifyModelHandler built their response maps purely from
// database.Model + buildCapabilitiesList — neither of which carries
// SupportsMCPs/SupportsLSPs/SupportsACPs/SupportsRAG/SupportsSkills/
// SupportsPlugins at all, so a consumer reading these responses always saw
// the six flags as absent/default-false regardless of the verifier's actual
// computed capability data (stored on database.VerificationResult and
// populated by verification.Verifier.Verify's real C4 probes). These tests
// assert the fields are now present and sourced from that real computed
// data — with a MIXED true/false profile so a hardcoded constant of either
// polarity would fail this test — never fabricated.
// ---------------------------------------------------------------------------

// mixedCapabilityVerificationResult returns a completed VerificationResult
// with a deliberately mixed capability profile (three true, three false) so
// the guard cannot be satisfied by a single hardcoded boolean.
func mixedCapabilityVerificationResult(modelID int64) *database.VerificationResult {
	completedAt := time.Now()
	return &database.VerificationResult{
		ModelID:             modelID,
		VerificationType:    "capability",
		StartedAt:           completedAt.Add(-time.Second),
		CompletedAt:         &completedAt,
		Status:              "completed",
		SupportsMCPs:        true,
		SupportsLSPs:        false,
		SupportsACPs:        true,
		SupportsRAG:         false,
		SupportsSkills:      true,
		SupportsPlugins:     false,
		CodeLanguageSupport: []string{},
	}
}

func assertMixedCapabilityFields(t *testing.T, resp map[string]interface{}) {
	t.Helper()
	assert.Equal(t, true, resp["supports_mcps"], "supports_mcps must be sourced from the computed VerificationResult (true)")
	assert.Equal(t, false, resp["supports_lsps"], "supports_lsps must be sourced from the computed VerificationResult (false)")
	assert.Equal(t, true, resp["supports_acps"], "supports_acps must be sourced from the computed VerificationResult (true)")
	assert.Equal(t, false, resp["supports_rag"], "supports_rag must be sourced from the computed VerificationResult (false)")
	assert.Equal(t, true, resp["supports_skills"], "supports_skills must be sourced from the computed VerificationResult (true)")
	assert.Equal(t, false, resp["supports_plugins"], "supports_plugins must be sourced from the computed VerificationResult (false)")
}

func assertHonestFalseCapabilityFields(t *testing.T, resp map[string]interface{}) {
	t.Helper()
	for _, key := range []string{
		"supports_mcps", "supports_lsps", "supports_acps",
		"supports_rag", "supports_skills", "supports_plugins",
	} {
		v, ok := resp[key]
		require.Truef(t, ok, "expected capability key %q to be present (even if honest-false)", key)
		assert.Equalf(t, false, v, "expected %q to be the honest zero value (no verification result exists)", key)
	}
}

// TestListModelsHandler_CapabilityFields_SourcedFromComputedVerificationResult
// is the HXC-135 GREEN guard for ListModelsHandler.
func TestListModelsHandler_CapabilityFields_SourcedFromComputedVerificationResult(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	provider := &database.Provider{Name: "TestProvider", Endpoint: "https://api.test.com/v1", IsActive: true}
	require.NoError(t, db.CreateProvider(provider))

	verifiedModel := &database.Model{
		ProviderID:         provider.ID,
		ModelID:            "capability-model",
		Name:               "Capability Model",
		VerificationStatus: "verified",
	}
	require.NoError(t, db.CreateModel(verifiedModel))
	require.NoError(t, db.CreateVerificationResult(mixedCapabilityVerificationResult(verifiedModel.ID)))

	unverifiedModel := &database.Model{
		ProviderID:         provider.ID,
		ModelID:            "no-verification-model",
		Name:               "No Verification Model",
		VerificationStatus: "pending",
	}
	require.NoError(t, db.CreateModel(unverifiedModel))

	server := createTestServerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/models", nil)
	w := httptest.NewRecorder()
	server.ListModelsHandler(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	models := response["models"].([]interface{})
	require.Len(t, models, 2)

	byName := map[string]map[string]interface{}{}
	for _, raw := range models {
		m := raw.(map[string]interface{})
		byName[m["name"].(string)] = m
	}

	require.Contains(t, byName, "Capability Model")
	assertMixedCapabilityFields(t, byName["Capability Model"])

	require.Contains(t, byName, "No Verification Model")
	assertHonestFalseCapabilityFields(t, byName["No Verification Model"])
}

// TestGetModelHandler_CapabilityFields_SourcedFromComputedVerificationResult
// is the HXC-135 GREEN guard for GetModelHandler.
func TestGetModelHandler_CapabilityFields_SourcedFromComputedVerificationResult(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	provider := &database.Provider{Name: "TestProvider", Endpoint: "https://api.test.com/v1", IsActive: true}
	require.NoError(t, db.CreateProvider(provider))

	model := &database.Model{
		ProviderID:         provider.ID,
		ModelID:            "capability-model",
		Name:               "Capability Model",
		VerificationStatus: "verified",
	}
	require.NoError(t, db.CreateModel(model))
	require.NoError(t, db.CreateVerificationResult(mixedCapabilityVerificationResult(model.ID)))

	server := createTestServerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/models/1", nil)
	w := httptest.NewRecorder()
	server.GetModelHandler(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assertMixedCapabilityFields(t, response)
}

// TestGetModelHandler_CapabilityFields_HonestFalse_WhenNoVerificationResult
// proves GetModelHandler never fabricates a capability for a model with no
// completed VerificationResult yet.
func TestGetModelHandler_CapabilityFields_HonestFalse_WhenNoVerificationResult(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	provider := &database.Provider{Name: "TestProvider", Endpoint: "https://api.test.com/v1", IsActive: true}
	require.NoError(t, db.CreateProvider(provider))

	model := &database.Model{
		ProviderID:         provider.ID,
		ModelID:            "no-verification-model",
		Name:               "No Verification Model",
		VerificationStatus: "pending",
	}
	require.NoError(t, db.CreateModel(model))

	server := createTestServerWithDB(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/models/1", nil)
	w := httptest.NewRecorder()
	server.GetModelHandler(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assertHonestFalseCapabilityFields(t, response)
}

// TestVerifyModelHandler_CapabilityFields_SourcedFromComputedVerificationResult
// is the HXC-135 GREEN guard for VerifyModelHandler: it must publish the six
// capability flags from the model's latest COMPLETED VerificationResult (the
// C4-probe-populated record created by verification.Verifier.Verify), not
// merely the status/score pair the ModelVerifier seam returns.
func TestVerifyModelHandler_CapabilityFields_SourcedFromComputedVerificationResult(t *testing.T) {
	db, cleanup := setupIntegrationTestDB(t)
	defer cleanup()

	provider := &database.Provider{Name: "TestProvider", Endpoint: "https://api.test.com/v1", IsActive: true}
	require.NoError(t, db.CreateProvider(provider))

	model := &database.Model{
		ProviderID:         provider.ID,
		ModelID:            "capability-model",
		Name:               "Capability Model",
		VerificationStatus: "pending",
	}
	require.NoError(t, db.CreateModel(model))
	// A prior, real capability-probing verification run already computed and
	// persisted the mixed profile — VerifyModelHandler's own stubbed run below
	// only updates status/score, it must NOT clobber or fabricate the
	// capability flags; it must surface the latest completed computed data.
	require.NoError(t, db.CreateVerificationResult(mixedCapabilityVerificationResult(model.ID)))

	server := createTestServerWithDB(t, db)
	server.verifier = &stubModelVerifier{status: "verified", score: 0.91}

	req := httptest.NewRequest(http.MethodPost, "/api/models/1/verify", nil)
	w := httptest.NewRecorder()
	server.VerifyModelHandler(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assertMixedCapabilityFields(t, response)
}
