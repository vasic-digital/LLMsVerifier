#!/bin/bash
# Day 1 Implementation Verification Script

echo "🔍 Verifying Day 1 implementation completion..."
echo "==============================================="
echo "📅 Date: $(date)"
echo ""

cd /media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier/llm-verifier

# Check if tests are now passing
echo "✅ Running re-enabled tests..."
echo ""

# Test API package
echo "🧪 Testing API package..."
if go test ./api -v; then
    echo "✅ API tests passing"
    API_TESTS_PASSED=true
else
    echo "❌ API tests failed"
    API_TESTS_PASSED=false
fi

echo ""

# Test events package
echo "🧪 Testing events package..."
if go test ./events -v; then
    echo "✅ Events tests passing"
    EVENTS_TESTS_PASSED=true
else
    echo "❌ Events tests failed"
    EVENTS_TESTS_PASSED=false
fi

echo ""

# Test challenges package
echo "🧪 Testing challenges package..."
if go test ./challenges -v; then
    echo "✅ Challenges tests passing"
    CHALLENGES_TESTS_PASSED=true
else
    echo "❌ Challenges tests failed"
    CHALLENGES_TESTS_PASSED=false
fi

echo ""

# Test notifications package
echo "🧪 Testing notifications package..."
if go test ./notifications -v; then
    echo "✅ Notifications tests passing"
    NOTIFICATIONS_TESTS_PASSED=true
else
    echo "❌ Notifications tests failed"
    NOTIFICATIONS_TESTS_PASSED=false
fi

echo ""

# Check if challenges are re-enabled
echo "✅ Verifying challenges are re-enabled..."
DISABLED_COUNT=$(grep -r "temporarily disabled" . --include="*.go" | wc -l)
if [ "$DISABLED_COUNT" -eq 0 ]; then
    echo "✅ No 'temporarily disabled' found - challenges should be re-enabled"
    CHALLENGES_REENABLED=true
else
    echo "⚠️  Found $DISABLED_COUNT 'temporarily disabled' references - review needed"
    CHALLENGES_REENABLED=false
fi

echo ""

# Check for score suffix format in code
echo "✅ Verifying score suffix format implementation..."
SCORE_SUFFIX_COUNT=$(grep -r "(SC:" . --include="*.go" | wc -l)
if [ "$SCORE_SUFFIX_COUNT" -gt 0 ]; then
    echo "✅ Found $SCORE_SUFFIX_COUNT score suffix implementations"
    SCORE_SUFFIX_IMPLEMENTED=true
else
    echo "⚠️  No score suffix implementations found"
    SCORE_SUFFIX_IMPLEMENTED=false
fi

echo ""

# Test build integrity
echo "🔨 Testing build integrity..."
if go build ./...; then
    echo "✅ Project builds successfully"
    BUILD_SUCCESS=true
else
    echo "❌ Project build failed"
    BUILD_SUCCESS=false
fi

echo ""

# Generate test coverage report
echo "📊 Generating test coverage report..."
go test ./... -coverprofile=coverage.out -covermode=atomic 2>/dev/null
if [ -f coverage.out ]; then
    COVERAGE_PERCENTAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
    echo "✅ Current test coverage: $COVERAGE_PERCENTAGE"
    COVERAGE_REPORTED=true
else
    echo "⚠️  Could not generate coverage report"
    COVERAGE_PERCENTAGE="0%"
    COVERAGE_REPORTED=false
fi

echo ""

# Summary report
echo "📋 DAY 1 VERIFICATION SUMMARY"
echo "=============================="
echo ""
echo "✅ Test Results:"
echo "  API Tests: $([ "$API_TESTS_PASSED" = true ] && echo "PASSED" || echo "FAILED")"
echo "  Events Tests: $([ "$EVENTS_TESTS_PASSED" = true ] && echo "PASSED" || echo "FAILED")"
echo "  Challenges Tests: $([ "$CHALLENGES_TESTS_PASSED" = true ] && echo "PASSED" || echo "FAILED")"
echo "  Notifications Tests: $([ "$NOTIFICATIONS_TESTS_PASSED" = true ] && echo "PASSED" || echo "FAILED")"
echo ""
echo "✅ Implementation Status:"
echo "  Challenges Re-enabled: $([ "$CHALLENGES_REENABLED" = true ] && echo "YES" || echo "NO")"
echo "  Score Suffix Format: $([ "$SCORE_SUFFIX_IMPLEMENTED" = true ] && echo "IMPLEMENTED" || echo "NOT FOUND")"
echo "  Build Status: $([ "$BUILD_SUCCESS" = true ] && echo "SUCCESS" || echo "FAILED")"
echo "  Test Coverage: $COVERAGE_PERCENTAGE"
echo ""

# Calculate success rate
SUCCESS_COUNT=0
TOTAL_CHECKS=6

[ "$API_TESTS_PASSED" = true ] && SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
[ "$EVENTS_TESTS_PASSED" = true ] && SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
[ "$CHALLENGES_TESTS_PASSED" = true ] && SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
[ "$NOTIFICATIONS_TESTS_PASSED" = true ] && SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
[ "$CHALLENGES_REENABLED" = true ] && SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
[ "$BUILD_SUCCESS" = true ] && SUCCESS_COUNT=$((SUCCESS_COUNT + 1))

SUCCESS_RATE=$((SUCCESS_COUNT * 100 / TOTAL_CHECKS))

echo "🎯 Overall Success Rate: $SUCCESS_RATE% ($SUCCESS_COUNT/$TOTAL_CHECKS)"
echo ""

if [ "$SUCCESS_RATE" -ge 80 ]; then
    echo "🎉 DAY 1 IMPLEMENTATION COMPLETED SUCCESSFULLY!"
    echo "✅ Ready to proceed to Week 2 implementation"
    exit 0
else
    echo "⚠️  DAY 1 IMPLEMENTATION NEEDS ATTENTION"
    echo "📝 Please review failed tests and fix issues before continuing"
    exit 1
fi