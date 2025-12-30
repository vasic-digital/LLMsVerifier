package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ==================== ConfigExport CRUD Tests ====================

func TestCreateConfigExport(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	targetModels := "gpt-4,claude-3"
	targetProviders := "openai,anthropic"
	createdBy := "user123"

	configExport := &ConfigExport{
		ExportType:      "model_config",
		Name:            "Test Export",
		Description:     "Test config export",
		ConfigData:      `{"key": "value"}`,
		TargetModels:    &targetModels,
		TargetProviders: &targetProviders,
		IsVerified:      true,
		CreatedBy:       &createdBy,
	}

	err := db.CreateConfigExport(configExport)
	require.NoError(t, err)
	assert.NotZero(t, configExport.ID)

	// Verify creation
	retrieved, err := db.GetConfigExport(configExport.ID)
	require.NoError(t, err)
	assert.Equal(t, "Test Export", retrieved.Name)
	assert.Equal(t, "model_config", retrieved.ExportType)
	assert.True(t, retrieved.IsVerified)
}

func TestGetConfigExport(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create first
	configExport := &ConfigExport{
		ExportType:  "provider_config",
		Name:        "Get Test Export",
		Description: "For retrieval test",
		ConfigData:  `{"setting": true}`,
	}
	err := db.CreateConfigExport(configExport)
	require.NoError(t, err)

	// Get by ID
	retrieved, err := db.GetConfigExport(configExport.ID)
	require.NoError(t, err)
	assert.Equal(t, configExport.Name, retrieved.Name)
	assert.Equal(t, configExport.ConfigData, retrieved.ConfigData)

	// Get non-existent
	_, err = db.GetConfigExport(99999)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestUpdateConfigExport(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create
	configExport := &ConfigExport{
		ExportType:  "model_config",
		Name:        "Original Name",
		Description: "Original Description",
		ConfigData:  `{"version": 1}`,
		IsVerified:  false,
	}
	err := db.CreateConfigExport(configExport)
	require.NoError(t, err)

	// Update
	newNotes := "Verified by admin"
	configExport.Name = "Updated Name"
	configExport.Description = "Updated Description"
	configExport.ConfigData = `{"version": 2}`
	configExport.IsVerified = true
	configExport.VerificationNotes = &newNotes

	err = db.UpdateConfigExport(configExport)
	require.NoError(t, err)

	// Verify update
	retrieved, err := db.GetConfigExport(configExport.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", retrieved.Name)
	assert.Equal(t, "Updated Description", retrieved.Description)
	assert.True(t, retrieved.IsVerified)
}

func TestDeleteConfigExport(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create
	configExport := &ConfigExport{
		ExportType:  "model_config",
		Name:        "Delete Test",
		Description: "Will be deleted",
		ConfigData:  `{}`,
	}
	err := db.CreateConfigExport(configExport)
	require.NoError(t, err)

	// Delete
	err = db.DeleteConfigExport(configExport.ID)
	require.NoError(t, err)

	// Verify deletion
	_, err = db.GetConfigExport(configExport.ID)
	assert.Error(t, err)
}

func TestListConfigExports(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create multiple exports
	for i := 0; i < 5; i++ {
		configExport := &ConfigExport{
			ExportType:  "model_config",
			Name:        "List Test",
			Description: "For listing",
			ConfigData:  `{}`,
		}
		err := db.CreateConfigExport(configExport)
		require.NoError(t, err)
	}

	// List all
	exports, err := db.ListConfigExports(nil)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(exports), 5)
}

func TestListConfigExports_WithFilters(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create exports with different types
	types := []string{"model_config", "provider_config", "model_config"}
	for i, exportType := range types {
		configExport := &ConfigExport{
			ExportType:  exportType,
			Name:        "Filter Test",
			Description: "For filter test",
			ConfigData:  `{}`,
			IsVerified:  i%2 == 0, // Alternate verified
		}
		err := db.CreateConfigExport(configExport)
		require.NoError(t, err)
	}

	// Filter by type
	filters := map[string]any{"export_type": "model_config"}
	exports, err := db.ListConfigExports(filters)
	require.NoError(t, err)
	for _, exp := range exports {
		assert.Equal(t, "model_config", exp.ExportType)
	}
}

func TestIncrementDownloadCount(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create export
	configExport := &ConfigExport{
		ExportType:  "model_config",
		Name:        "Download Test",
		Description: "For download count test",
		ConfigData:  `{}`,
	}
	err := db.CreateConfigExport(configExport)
	require.NoError(t, err)

	// Initial count should be 0
	retrieved, err := db.GetConfigExport(configExport.ID)
	require.NoError(t, err)
	assert.Equal(t, 0, retrieved.DownloadCount)

	// Increment
	err = db.IncrementDownloadCount(configExport.ID)
	require.NoError(t, err)

	// Verify increment
	retrieved, err = db.GetConfigExport(configExport.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, retrieved.DownloadCount)

	// Increment again
	err = db.IncrementDownloadCount(configExport.ID)
	require.NoError(t, err)

	retrieved, err = db.GetConfigExport(configExport.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, retrieved.DownloadCount)
}

func TestGetConfigExportsByType(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create exports with different types
	for _, exportType := range []string{"model_config", "provider_config", "model_config"} {
		configExport := &ConfigExport{
			ExportType:  exportType,
			Name:        "Type Test",
			Description: "For type test",
			ConfigData:  `{}`,
		}
		err := db.CreateConfigExport(configExport)
		require.NoError(t, err)
	}

	// Get by type
	exports, err := db.GetConfigExportsByType("model_config")
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(exports), 2)
	for _, exp := range exports {
		assert.Equal(t, "model_config", exp.ExportType)
	}
}

func TestGetVerifiedConfigExports(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create verified and unverified exports
	verified := &ConfigExport{
		ExportType:  "model_config",
		Name:        "Verified Export",
		Description: "This is verified",
		ConfigData:  `{}`,
		IsVerified:  true,
	}
	err := db.CreateConfigExport(verified)
	require.NoError(t, err)

	unverified := &ConfigExport{
		ExportType:  "model_config",
		Name:        "Unverified Export",
		Description: "This is not verified",
		ConfigData:  `{}`,
		IsVerified:  false,
	}
	err = db.CreateConfigExport(unverified)
	require.NoError(t, err)

	// Get verified only
	exports, err := db.GetVerifiedConfigExports()
	require.NoError(t, err)
	for _, exp := range exports {
		assert.True(t, exp.IsVerified)
	}
}
