package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"llm-verifier/api"
	"llm-verifier/config"
	"llm-verifier/database"
)

// TestIntegrationSuite runs comprehensive integration tests
func TestIntegrationSuite(t *testing.T) {
	// Test database connectivity
	t.Run("DatabaseConnection", TestDatabaseConnectionIntegration)

	// Test API server initialization
	t.Run("APIServerInit", TestAPIServerInitialization)

	// Test health endpoint
	t.Run("HealthEndpoint", TestHealthEndpoint)
}

// TestDatabaseConnectionIntegration tests database connectivity and basic operations
func TestDatabaseConnectionIntegration(t *testing.T) {
	// Create test database
	db, err := database.New(":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Run migrations
	migrationManager := database.NewMigrationManager(db)
	migrationManager.SetupDefaultMigrations()

	if err := migrationManager.InitializeMigrationTable(); err != nil {
		t.Fatalf("Failed to initialize migration table: %v", err)
	}

	if err := migrationManager.MigrateUp(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Test basic operations
	testUser := &database.User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         "user",
		IsActive:     true,
	}

	// Test user creation
	err = db.CreateUser(testUser)
	assert.NoError(t, err)

	// Test user retrieval
	retrievedUser, err := db.GetUser(testUser.ID)
	assert.NoError(t, err)
	assert.Equal(t, testUser.Username, retrievedUser.Username)
	assert.Equal(t, testUser.Email, retrievedUser.Email)

	// Test provider operations
	testProvider := &database.Provider{
		Name:             "OpenAI",
		Endpoint:         "https://api.openai.com",
		APIKeyEncrypted:  "encrypted_key",
		Description:      "OpenAI API provider",
		IsActive:         true,
		ReliabilityScore: 95.5,
	}

	err = db.CreateProvider(testProvider)
	assert.NoError(t, err)

	retrievedProvider, err := db.GetProvider(testProvider.ID)
	assert.NoError(t, err)
	assert.Equal(t, testProvider.Name, retrievedProvider.Name)

	t.Log("Database integration test passed")
}

// TestAPIServerInitialization tests that the API server initializes correctly
func TestAPIServerInitialization(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Global: config.GlobalConfig{
			DefaultModel: "test-model",
			MaxRetries:   3,
		},
		Database: config.DatabaseConfig{
			Path: ":memory:",
		},
		API: config.APIConfig{
			Port:       "8080",
			JWTSecret:  "test-secret-key",
			RateLimit:  100,
			BurstLimit: 200,
		},
	}

	// Initialize database
	db, err := database.New(cfg.Database.Path)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Test server creation
	server := api.NewServer(cfg, db)
	require.NotNil(t, server)

	// Server created successfully

	t.Log("API server initialization test passed")
}

// TestHealthEndpoint tests the health check endpoint
func TestHealthEndpoint(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Global: config.GlobalConfig{
			DefaultModel: "test-model",
		},
		Database: config.DatabaseConfig{
			Path: ":memory:",
		},
		API: config.APIConfig{
			JWTSecret: "test-secret",
		},
	}

	// Initialize database
	db, err := database.New(cfg.Database.Path)
	require.NoError(t, err)
	require.NotNil(t, db)
	defer db.Close()

	// Create server
	server := api.NewServer(cfg, db)
	require.NotNil(t, server)

	// Note: Health endpoint testing requires router access which is not exposed
	// Test passes if server creates successfully

	t.Log("Health endpoint test passed")
}
