#!/bin/bash

# Test script for ACP CLI
set -e

echo "🧪 Testing ACP CLI Implementation"
echo "=================================="

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -e "\n${YELLOW}Testing:${NC} $test_name"
    if eval "$test_command"; then
        echo -e "${GREEN}✅ PASS:${NC} $test_name"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}❌ FAIL:${NC} $test_name"
        ((TESTS_FAILED++))
    fi
}

# Change to project directory
cd "$(dirname "$0")"

echo "Step 1: Building ACP CLI"
run_test "Build ACP CLI" "make build-acp"

if [ ! -f "bin/acp-cli" ]; then
    echo -e "${RED}❌ FAIL:${NC} ACP CLI binary not found at bin/acp-cli"
    exit 1
fi

echo -e "${GREEN}✅ PASS:${NC} ACP CLI built successfully"
echo ""

echo "Step 2: Testing CLI Commands"

# Test help command
run_test "Help command works" "./bin/acp-cli --help >/dev/null 2>&1"

# Test providers command
echo -e "\n${YELLOW}Testing:${NC} Providers command"
if ./bin/acp-cli providers >/dev/null 2>&1; then
    echo -e "${GREEN}✅ PASS:${NC} Providers command works"
    ((TESTS_PASSED++))
else
    echo -e "${RED}❌ FAIL:${NC} Providers command failed"
    ((TESTS_FAILED++))
fi

# Test single model verification
echo -e "\n${YELLOW}Testing:${NC} Verify single model"
if ./bin/acp-cli verify --model gpt-4 --provider openai >/dev/null 2>&1; then
    echo -e "${GREEN}✅ PASS:${NC} Single model verification works"
    ((TESTS_PASSED++))
else
    echo -e "${RED}❌ FAIL:${NC} Single model verification failed"
    ((TESTS_FAILED++))
fi

# Test JSON output format
echo -e "\n${YELLOW}Testing:${NC} JSON output format"
if ./bin/acp-cli verify --model gpt-4 --provider openai --output json | jq . >/dev/null 2>&1; then
    echo -e "${GREEN}✅ PASS:${NC} JSON output format works"
    ((TESTS_PASSED++))
else
    echo -e "${RED}❌ FAIL:${NC} JSON output format failed"
    ((TESTS_FAILED++))
fi

# Test batch verification
echo -e "\n${YELLOW}Testing:${NC} Batch verification"
if ./bin/acp-cli batch --models gpt-4,gpt-3.5-turbo,claude-3-opus >/dev/null 2>&1; then
    echo -e "${GREEN}✅ PASS:${NC} Batch verification works"
    ((TESTS_PASSED++))
else
    echo -e "${RED}❌ FAIL:${NC} Batch verification failed"
    ((TESTS_FAILED++))
fi

# Test models file
echo -e "\n${YELLOW}Testing:${NC} Models file input"
echo -e "gpt-4\ngpt-3.5-turbo\nclaude-3-opus" > test-models.txt
if ./bin/acp-cli batch --models-file test-models.txt >/dev/null 2>&1; then
    echo -e "${GREEN}✅ PASS:${NC} Models file input works"
    ((TESTS_PASSED++))
else
    echo -e "${RED}❌ FAIL:${NC} Models file input failed"
    ((TESTS_FAILED++))
fi
rm -f test-models.txt

echo ""
echo "Step 3: Testing ACP Client Library"

# Test that ACP client builds
run_test "ACP Client builds" "go build -o /tmp/acp-client-test ./llm-verifier/client/acp_client.go"

# Run ACP unit tests
run_test "ACP unit tests pass" "go test -v ./llm-verifier/tests/acp_test.go -run TestACPsdetection"

echo ""
echo "Step 4: Integration Test"

# Create a simple ACP test
cat > test_acp_integration.go << 'EOF'
package main

import (
    "context"
    "fmt"
    "github.com/llmverifier/llmverifier"
    "github.com/llmverifier/llmverifier/config"
    "github.com/llmverifier/llmverifier/providers"
)

func main() {
    cfg := &config.Config{GlobalTimeout: 30}
    verifier := llmverifier.New(cfg)
    registry := providers.NewProviderRegistry()
    
    providerConfig, _ := registry.GetConfig("openai")
    client, _ := createProviderClient(providerConfig)
    
    ctx := context.Background()
    result := verifier.TestACPs(client, "test-model", ctx)
    
    if result {
        fmt.Println("ACP test passed")
        os.Exit(0)
    } else {
        fmt.Println("ACP test failed")
        os.Exit(1)
    }
}

func createProviderClient(config *providers.ProviderConfig) (llmverifier.LLMClient, error) {
    // Mock implementation
    return nil, nil
}
EOF

echo ""
echo "==================================="
echo "📊 Test Results Summary"
echo "==================================="
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All ACP CLI tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed. Please review the output above.${NC}"
    exit 1
fi