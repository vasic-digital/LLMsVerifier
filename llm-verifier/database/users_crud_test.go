package database

import (
	"strings"
	"testing"
)

func TestCreateUser(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	user := &User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "password123",
		FullName:     "Test User",
		Role:         "user",
		IsActive:     true,
	}

	err := db.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	if user.ID == 0 {
		t.Error("User ID should be set after creation")
	}

	// Verify user was created
	retrieved, err := db.GetUserByUsername("testuser")
	if err != nil {
		t.Fatalf("Failed to retrieve user: %v", err)
	}

	if retrieved.Username != user.Username {
		t.Errorf("Expected username %s, got %s", user.Username, retrieved.Username)
	}

	if retrieved.Email != user.Email {
		t.Errorf("Expected email %s, got %s", user.Email, retrieved.Email)
	}
}

func TestGetUserByID(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	user := &User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "password123",
		FullName:     "Test User",
		Role:         "user",
		IsActive:     true,
	}

	err := db.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	retrieved, err := db.GetUser(user.ID)
	if err != nil {
		t.Fatalf("Failed to get user by ID: %v", err)
	}

	if retrieved.Username != user.Username {
		t.Errorf("Expected username %s, got %s", user.Username, retrieved.Username)
	}

	if retrieved.ID != user.ID {
		t.Errorf("Expected ID %d, got %d", user.ID, retrieved.ID)
	}
}

func TestGetUserByUsername(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	user := &User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "password123",
		FullName:     "Test User",
		Role:         "user",
		IsActive:     true,
	}

	err := db.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	retrieved, err := db.GetUserByUsername("testuser")
	if err != nil {
		t.Fatalf("Failed to get user by username: %v", err)
	}

	if retrieved.Username != user.Username {
		t.Errorf("Expected username %s, got %s", user.Username, retrieved.Username)
	}
}

func TestUpdateUser(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	user := &User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "password123",
		FullName:     "Test User",
		Role:         "user",
		IsActive:     true,
	}

	err := db.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Update user
	user.FullName = "Updated Name"
	user.Role = "admin"

	err = db.UpdateUser(user)
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}

	// Verify update
	retrieved, err := db.GetUser(user.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve updated user: %v", err)
	}

	if retrieved.FullName != "Updated Name" {
		t.Errorf("Expected full name 'Updated Name', got '%s'", retrieved.FullName)
	}

	if retrieved.Role != "admin" {
		t.Errorf("Expected role 'admin', got '%s'", retrieved.Role)
	}
}

func TestDeleteUser(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	user := &User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "password123",
		FullName:     "Test User",
		Role:         "user",
		IsActive:     true,
	}

	err := db.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	userID := user.ID

	err = db.DeleteUser(userID)
	if err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	// Verify user is deleted
	_, err = db.GetUser(userID)
	if err == nil {
		t.Error("Expected error when retrieving deleted user")
	}

	if err == nil || !strings.Contains(err.Error(), "user not found") {
		t.Errorf("Expected error containing 'user not found', got %v", err)
	}
}

func TestListUsers(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	// Create test users
	users := []*User{
		{
			Username:     "user1",
			Email:        "user1@example.com",
			PasswordHash: "password123",
			FullName:     "User One",
			Role:         "user",
			IsActive:     true,
		},
		{
			Username:     "user2",
			Email:        "user2@example.com",
			PasswordHash: "password123",
			FullName:     "User Two",
			Role:         "admin",
			IsActive:     true,
		},
	}

	for _, user := range users {
		err := db.CreateUser(user)
		if err != nil {
			t.Fatalf("Failed to create user: %v", err)
		}
	}

	// List users
	allUsers, err := db.ListUsers(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Failed to list users: %v", err)
	}

	if len(allUsers) < 2 {
		t.Errorf("Expected at least 2 users, got %d", len(allUsers))
	}
}

func TestUpdateLastLogin(t *testing.T) {
	db := setupTestDB(t)
	defer cleanupTestDB(t, db)

	user := &User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "password123",
		FullName:     "Test User",
		Role:         "user",
		IsActive:     true,
	}

	err := db.CreateUser(user)
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Update last login
	err = db.UpdateUserLastLogin(user.ID)
	if err != nil {
		t.Fatalf("Failed to update last login: %v", err)
	}

	// Verify update
	retrieved, err := db.GetUser(user.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve user: %v", err)
	}

	if retrieved.LastLogin == nil {
		t.Error("Expected LastLogin to be set")
	}
}

// Helper functions for testing
func setupTestDB(t *testing.T) *Database {
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Initialize schema
	err = db.initializeSchema()
	if err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	return db
}

func cleanupTestDB(t *testing.T, db *Database) {
	err := db.Close()
	if err != nil {
		t.Logf("Warning: failed to close test database: %v", err)
	}
}
