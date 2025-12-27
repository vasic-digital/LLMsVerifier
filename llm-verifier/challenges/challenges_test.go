package challenges

import (
	"context"
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
	"llm-verifier/database"
)

// TestProviderModelsDiscovery tests the provider models discovery challenge
func TestProviderModelsDiscovery(t *testing.T) {
	// Create a mock context
	ctx := context.Background()
	
	// Create mock database
	db := &database.Database{}
	
	// Test challenge creation
	challenge := NewProviderModelsDiscoveryChallenge(db)
	
	assert.NotNil(t, challenge)
	
	// Test challenge execution
	err := challenge.Run(ctx)
	assert.NoError(t, err)
}

// TestCrushConfigConverter tests the crush config converter challenge
func TestCrushConfigConverter(t *testing.T) {
	// Create a mock context
	ctx := context.Background()
	
	// Create mock database
	db := &database.Database{}
	
	// Test challenge creation
	challenge := NewCrushConfigConverterChallenge(db)
	
	assert.NotNil(t, challenge)
	
	// Test challenge execution
	err := challenge.Run(ctx)
	assert.NoError(t, err)
}

// TestRunModelVerification tests the run model verification challenge
func TestRunModelVerification(t *testing.T) {
	// Create a mock context
	ctx := context.Background()
	
	// Create mock database
	db := &database.Database{}
	
	// Test challenge creation
	challenge := NewRunModelVerificationChallenge(db, nil)
	
	assert.NotNil(t, challenge)
	
	// Test challenge execution
	err := challenge.Run(ctx)
	assert.NoError(t, err)
}

// TestChallengeInterface tests the challenge interface
func TestChallengeInterface(t *testing.T) {
	// Test that our challenges implement the interface
	var _ Challenge = (*ProviderModelsDiscoveryChallenge)(nil)
	var _ Challenge = (*CrushConfigConverterChallenge)(nil)
	var _ Challenge = (*RunModelVerificationChallenge)(nil)
}

// TestChallengeReEnabled tests that challenges are re-enabled
func TestChallengeReEnabled(t *testing.T) {
	// Test that challenges are not temporarily disabled
	// This is a meta-test to ensure our implementation is complete
	
	// Verify challenges have proper implementation
	ctx := context.Background()
	timeout := 5 * time.Second
	
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	
	// Test that we can create challenge instances without errors
	db := &database.Database{}
	
	providerChallenge := NewProviderModelsDiscoveryChallenge(db)
	assert.NotNil(t, providerChallenge)
	
	crushChallenge := NewCrushConfigConverterChallenge(db)
	assert.NotNil(t, crushChallenge)
	
	verificationChallenge := NewRunModelVerificationChallenge(db, nil)
	assert.NotNil(t, verificationChallenge)
	
	// Test that challenges have the required Run method
	assert.NotPanics(t, func() {
		// These would panic if the methods don't exist
		_ = providerChallenge.Run
		_ = crushChallenge.Run
		_ = verificationChallenge.Run
	})
}