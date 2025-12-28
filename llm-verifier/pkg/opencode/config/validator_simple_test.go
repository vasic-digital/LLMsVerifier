package opencode_config

import (
	"testing"
	
	"github.com/stretchr/testify/assert"
)

func TestCreateDefaultConfig(t *testing.T) {
	config := CreateDefaultConfig()
	
	assert.NotNil(t, config)
	assert.NotNil(t, config.Provider)
	assert.NotNil(t, config.Agent)
	
	assert.Contains(t, config.Provider, "openai")
	assert.Contains(t, config.Agent, "build")
}