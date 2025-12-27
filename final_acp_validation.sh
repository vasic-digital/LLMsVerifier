#!/bin/bash

# Final ACP Implementation Validation
set -e

echo "🔍 Final ACP Implementation Validation"
echo "======================================="

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

TESTS_PASSED=0
TESTS_FAILED=0

cd /media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier

echo "1. Testing ACP CLI Binary..."
if [ -f "bin/acp-cli" ]; then
    echo -e "${GREEN}✅${NC} ACP CLI binary exists"
    ((TESTS_PASSED++))
else
    echo -e "${RED}❌${NC} ACP CLI binary not found"
    ((TESTS_FAILED++))
fi

echo "2. Testing ACP CLI Help Command..."
if ./bin/acp-cli --help >/dev/null 2>&1; then
    echo -e "${GREEN}✅${NC} ACP CLI help command works"
    ((TESTS_PASSED++))
else
    echo -e "${RED}❌${NC} ACP CLI help command failed"
    ((TESTS_FAILED++))
fi

echo "3. Testing ACP CLI List Command..."
if ./bin/acp-cli list >/dev/null 2>&1; then
    echo -e "${GREEN}✅${NC} ACP CLI list command works"
    ((TESTS_PASSED++))
else
    echo -e "${RED}❌${NC} ACP CLI list command failed"
    ((TESTS_FAILED++))
fi

echo "4. Testing ACP CLI Verify Command..."
if ./bin/acp-cli verify --models gpt-4 --output json >/dev/null 2>&1; then
    echo -e "${GREEN}✅${NC} ACP CLI verify command works"
    ((TESTS_PASSED++))
else
    echo -e "${RED}❌${NC} ACP CLI verify command failed"
    ((TESTS_FAILED++))
fi

echo "5. Testing ACP Method in Verifier..."
cd llm-verifier
if /usr/lib/go-1.22/bin/go test -v ./tests/unit_test.go -run TestACPsDetection >/dev/null 2>&1; then
    echo -e "${GREEN}✅${NC} ACP TestACPs method exists and works"
    ((TESTS_PASSED++))
else
    echo -e "${RED}❌${NC} ACP TestACPs method test failed"
    ((TESTS_FAILED++))
fi

cd ..

echo "6. Checking ACP Feature Detection in Output..."
if ./bin/acp-cli verify --models gpt-4 --output json | grep -q '"acps"'; then
    echo -e "${GREEN}✅${NC} ACP feature detection field present in output"
    ((TESTS_PASSED++))
else
    echo -e "${RED}❌${NC} ACP feature detection field missing from output"
    ((TESTS_FAILED++))
fi

echo ""
echo "======================================="
echo "📊 Final Validation Results"
echo "======================================="
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 ACP Implementation Validation SUCCESSFUL!${NC}"
    echo "All core ACP components are working correctly."
    echo ""
    echo "✅ ACP CLI Tool: Functional"
    echo "✅ ACP TestACPs Method: Implemented"
    echo "✅ ACP Feature Detection: Integrated"
    echo "✅ ACP Output Format: Correct"
    exit 0
else
    echo -e "${RED}❌ ACP Implementation Validation FAILED${NC}"
    echo "Some components need attention."
    exit 1
fi