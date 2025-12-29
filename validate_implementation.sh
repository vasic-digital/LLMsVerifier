#!/bin/bash

# Quick validation script for the model verification implementation

echo "🔍 Validating Model Verification Implementation"
echo "=============================================="

# Check Go syntax by building packages
echo "Checking Go syntax by building packages..."

cd llm-verifier

ERRORS=0

# Check providers package
echo -n "Checking providers package... "
if go build -o /dev/null ./providers/... 2>/dev/null; then
    echo "✅ OK"
else
    echo "❌ FAILED"
    ERRORS=$((ERRORS + 1))
fi

# Check model-verification CLI
echo -n "Checking model-verification CLI... "
if go build -o /dev/null ./cmd/model-verification/... 2>/dev/null; then
    echo "✅ OK"
else
    echo "❌ FAILED"
    ERRORS=$((ERRORS + 1))
fi

cd ..

# Check for missing imports or dependencies
echo ""
echo "Checking for missing dependencies..."
cd llm-verifier

if go mod tidy; then
    echo "✅ Dependencies resolved successfully"
else
    echo "❌ Failed to resolve dependencies"
    ERRORS=$((ERRORS + 1))
fi

# Check for compilation errors
echo ""
echo "Checking compilation..."
if go build -o /dev/null ./providers/... 2>/dev/null; then
    echo "✅ Providers package compiles successfully"
else
    echo "❌ Providers package compilation failed"
    ERRORS=$((ERRORS + 1))
fi

if go build -o /dev/null ./cmd/model-verification/... 2>/dev/null; then
    echo "✅ Model verification CLI compiles successfully"
else
    echo "❌ Model verification CLI compilation failed"
    ERRORS=$((ERRORS + 1))
fi

cd ..

# Summary
echo ""
echo "=============================================="
if [ $ERRORS -eq 0 ]; then
    echo "✅ All validation checks passed!"
else
    echo "❌ Found $ERRORS validation errors"
fi
echo "=============================================="

exit $ERRORS