package tests

import (
	"path/filepath"
	"testing"
	"time"

	"llm-verifier/database"
)

// setupTestDatabase creates a test database
func setupTestDatabase(t *testing.T) *database.Database {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := database.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	return db
}

// cleanupTestDatabase cleans up the test database
func cleanupTestDatabase(t *testing.T, db *database.Database) {
	if db != nil {
		db.Close()
	}
}

// TestDatabaseConnection tests database connection and initialization
func TestDatabaseConnection(t *testing.T) {
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	// Test connection is alive
	err := db.Ping()
	if err != nil {
		t.Fatalf("Database connection failed: %v", err)
	}
}

// TestProviderCRUD tests all provider CRUD operations
func TestProviderCRUD(t *testing.T) {
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	t.Run("CreateProvider", func(t *testing.T) {
		provider := &database.Provider{
			Name:        "TestProvider",
			Endpoint:    "https://api.testprovider.com",
			Description: "Test provider for unit tests",
			IsActive:    true,
		}

		err := db.CreateProvider(provider)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		// Verify provider was created by listing providers
		providers, err := db.ListProviders(map[string]interface{}{})
		if err != nil {
			t.Fatalf("Failed to get providers: %v", err)
		}

		found := false
		for _, p := range providers {
			if p.Name == "TestProvider" {
				found = true
				if p.Endpoint != "https://api.testprovider.com" {
					t.Errorf("Expected endpoint %s, got %s", "https://api.testprovider.com", p.Endpoint)
				}
				break
			}
		}

		if !found {
			t.Error("Created provider not found")
		}
	})

	t.Run("GetProvider", func(t *testing.T) {
		// First create a provider
		provider := createTestProvider(t, db, "GetTestProvider")

		retrieved, err := db.GetProvider(provider.ID)
		if err != nil {
			t.Fatalf("Failed to get provider: %v", err)
		}

		if retrieved.ID != provider.ID {
			t.Errorf("Expected ID %d, got %d", provider.ID, retrieved.ID)
		}

		if retrieved.Name != provider.Name {
			t.Errorf("Expected name %s, got %s", provider.Name, retrieved.Name)
		}
	})

	t.Run("UpdateProvider", func(t *testing.T) {
		provider := createTestProvider(t, db, "UpdateTestProvider")

		// Update provider
		provider.Description = "Updated description"
		provider.ReliabilityScore = 95.0

		err := db.UpdateProvider(provider)
		if err != nil {
			t.Fatalf("Failed to update provider: %v", err)
		}

		// Verify update
		retrieved, err := db.GetProvider(provider.ID)
		if err != nil {
			t.Fatalf("Failed to get updated provider: %v", err)
		}

		if retrieved.Description != "Updated description" {
			t.Errorf("Expected description 'Updated description', got %s", retrieved.Description)
		}

		if retrieved.ReliabilityScore != 95.0 {
			t.Errorf("Expected reliability score 95.0, got %f", retrieved.ReliabilityScore)
		}
	})

	t.Run("DeleteProvider", func(t *testing.T) {
		provider := createTestProvider(t, db, "DeleteTestProvider")

		err := db.DeleteProvider(provider.ID)
		if err != nil {
			t.Fatalf("Failed to delete provider: %v", err)
		}

		// Verify provider is deleted
		_, err = db.GetProvider(provider.ID)
		if err == nil {
			t.Error("Expected error when getting deleted provider")
		}
	})
}

// TestModelCRUD tests all model CRUD operations
func TestModelCRUD(t *testing.T) {
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	provider := createTestProvider(t, db, "ModelTestProvider")

	t.Run("CreateModel", func(t *testing.T) {
		model := &database.Model{
			ProviderID:         provider.ID,
			ModelID:            "test-model-1",
			Name:               "Test Model 1",
			Description:        "A test model for unit testing",
			VerificationStatus: "pending",
			OverallScore:       85.5,
		}

		err := db.CreateModel(model)
		if err != nil {
			t.Fatalf("Failed to create model: %v", err)
		}

		// Verify model was created
		models, err := db.ListModels(map[string]interface{}{"provider_id": provider.ID})
		if err != nil {
			t.Fatalf("Failed to get models: %v", err)
		}

		found := false
		for _, m := range models {
			if m.Name == "Test Model 1" {
				found = true
				if m.ModelID != "test-model-1" {
					t.Errorf("Expected model ID %s, got %s", "test-model-1", m.ModelID)
				}
				break
			}
		}

		if !found {
			t.Error("Created model not found")
		}
	})

	t.Run("GetModel", func(t *testing.T) {
		model := createTestModel(t, db, provider.ID, "GetTestModel")

		retrieved, err := db.GetModel(model.ID)
		if err != nil {
			t.Fatalf("Failed to get model: %v", err)
		}

		if retrieved.ID != model.ID {
			t.Errorf("Expected ID %d, got %d", model.ID, retrieved.ID)
		}

		if retrieved.Name != model.Name {
			t.Errorf("Expected name %s, got %s", model.Name, retrieved.Name)
		}
	})

	t.Run("UpdateModel", func(t *testing.T) {
		model := createTestModel(t, db, provider.ID, "UpdateTestModel")

		model.Description = "Updated description"
		model.OverallScore = 95.0

		err := db.UpdateModel(model)
		if err != nil {
			t.Fatalf("Failed to update model: %v", err)
		}

		// Verify update
		retrieved, err := db.GetModel(model.ID)
		if err != nil {
			t.Fatalf("Failed to get updated model: %v", err)
		}

		if retrieved.Description != "Updated description" {
			t.Errorf("Expected description 'Updated description', got %s", retrieved.Description)
		}

		if retrieved.OverallScore != 95.0 {
			t.Errorf("Expected score 95.0, got %f", retrieved.OverallScore)
		}
	})

	t.Run("DeleteModel", func(t *testing.T) {
		model := createTestModel(t, db, provider.ID, "DeleteTestModel")

		err := db.DeleteModel(model.ID)
		if err != nil {
			t.Fatalf("Failed to delete model: %v", err)
		}

		// Verify model is deleted
		_, err = db.GetModel(model.ID)
		if err == nil {
			t.Error("Expected error when getting deleted model")
		}
	})
}

// TestVerificationResultCRUD tests verification result CRUD operations
func TestVerificationResultCRUD(t *testing.T) {
	// Test re-enabled after database column mismatch fix

	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	provider := createTestProvider(t, db, "VerificationTestProvider")
	model := createTestModel(t, db, provider.ID, "VerificationTestModel")

	t.Run("CreateVerificationResult", func(t *testing.T) {
		result := &database.VerificationResult{
			ModelID:      model.ID,
			Status:       "completed",
			OverallScore: 87.5,
			AvgLatencyMs: 250,
			StartedAt:    time.Now().Add(-time.Minute * 15),
		}

		err := db.CreateVerificationResult(result)
		if err != nil {
			t.Fatalf("Failed to create verification result: %v", err)
		}

		// Verify result was created
		results, err := db.ListVerificationResults(map[string]interface{}{"model_id": model.ID})
		if err != nil {
			t.Fatalf("Failed to get verification results: %v", err)
		}

		found := false
		for _, r := range results {
			if r.Status == "completed" && r.OverallScore == 87.5 {
				found = true
				break
			}
		}

		if !found {
			t.Error("Created verification result not found")
		}
	})

	t.Run("GetVerificationResult", func(t *testing.T) {
		result := createTestVerificationResult(t, db, model.ID)

		retrieved, err := db.GetVerificationResult(result.ID)
		if err != nil {
			t.Fatalf("Failed to get verification result: %v", err)
		}

		if retrieved.ID != result.ID {
			t.Errorf("Expected ID %d, got %d", result.ID, retrieved.ID)
		}

		if retrieved.Status != result.Status {
			t.Errorf("Expected status %s, got %s", result.Status, retrieved.Status)
		}
	})
}

// TestQueryOptimizations tests optimized query functions
func TestQueryOptimizations(t *testing.T) {
	db := setupTestDatabase(t)
	defer cleanupTestDatabase(t, db)

	// Create test data
	provider := createTestProvider(t, db, "OptimizationTestProvider")
	model := createTestModel(t, db, provider.ID, "OptimizationTestModel")

	t.Run("GetModelsWithStats", func(t *testing.T) {
		models, err := db.GetModelsWithStats(10)
		if err != nil {
			t.Fatalf("Failed to get models with stats: %v", err)
		}

		// Should find our created model
		found := false
		for _, m := range models {
			if m.ID == model.ID {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected to find created model in results")
		}
	})

	t.Run("BatchInsertModels", func(t *testing.T) {
		models := []*database.Model{
			{
				ProviderID:         provider.ID,
				ModelID:            "batch-model-1",
				Name:               "Batch Model 1",
				VerificationStatus: "pending",
			},
			{
				ProviderID:         provider.ID,
				ModelID:            "batch-model-2",
				Name:               "Batch Model 2",
				VerificationStatus: "pending",
			},
		}

		err := db.BatchInsertModels(models)
		if err != nil {
			t.Fatalf("Failed to batch insert models: %v", err)
		}

		// Verify models were created
		for _, m := range models {
			retrieved, err := db.ListModels(map[string]interface{}{"model_id": m.ModelID})
			if err != nil {
				t.Fatalf("Failed to retrieve batch inserted model %s: %v", m.ModelID, err)
			}
			if len(retrieved) > 0 && retrieved[0].Name != m.Name {
				t.Errorf("Expected name %s, got %s", m.Name, retrieved[0].Name)
			}
		}
	})
}

// Helper functions for creating test data

func createTestProvider(t *testing.T, db *database.Database, name string) *database.Provider {
	provider := &database.Provider{
		Name:        name,
		Endpoint:    "https://api.test.com",
		Description: "Test provider",
		IsActive:    true,
	}

	err := db.CreateProvider(provider)
	if err != nil {
		t.Fatalf("Failed to create test provider: %v", err)
	}

	// Get the created provider
	providers, err := db.ListProviders(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to get providers: %v", err)
	}

	for _, p := range providers {
		if p.Name == name {
			return p
		}
	}

	t.Fatalf("Created provider not found")
	return nil
}

func createTestModel(t *testing.T, db *database.Database, providerID int64, name string) *database.Model {
	model := &database.Model{
		ProviderID:         providerID,
		ModelID:            name + "-id",
		Name:               name,
		Description:        "Test model for " + name,
		VerificationStatus: "pending",
		OverallScore:       75.0,
	}

	err := db.CreateModel(model)
	if err != nil {
		t.Fatalf("Failed to create test model: %v", err)
	}

	// Get the created model
	models, err := db.ListModels(map[string]interface{}{"provider_id": providerID})
	if err != nil {
		t.Fatalf("Failed to get models: %v", err)
	}

	for _, m := range models {
		if m.Name == name {
			return m
		}
	}

	t.Fatalf("Created model not found")
	return nil
}

func createTestVerificationResult(t *testing.T, db *database.Database, modelID int64) *database.VerificationResult {
	result := &database.VerificationResult{
		ModelID:      modelID,
		Status:       "completed",
		OverallScore: 80.0,
		AvgLatencyMs: 200,
		StartedAt:    time.Now(),
	}

	err := db.CreateVerificationResult(result)
	if err != nil {
		t.Fatalf("Failed to create test verification result: %v", err)
	}

	// Get the created result
	results, err := db.ListVerificationResults(map[string]interface{}{"model_id": modelID})
	if err != nil {
		t.Fatalf("Failed to get verification results: %v", err)
	}

	// Return the most recent one
	if len(results) > 0 {
		return results[0]
	}

	t.Fatalf("Created verification result not found")
	return nil
}
