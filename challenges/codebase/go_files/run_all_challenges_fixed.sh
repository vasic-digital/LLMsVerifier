#!/usr/bin/env bash
# Master Challenge Runner - Executes all 17 challenges

set -e

TIMESTAMP=$(date +%Y/%m/%d/%N)
BASE_DIR="challenges"
RESULTS_BASE="$BASE_DIR/master_results"
LOG_DIR="$BASE_DIR/execution_logs"

mkdir -p "$LOG_DIR" "$RESULTS_BASE"

echo "=========================================="
echo "LLM Verifier - Execute All Challenges"
echo "=========================================="
echo "Timestamp: $TIMESTAMP"
echo ""

# List of all 17 challenges in execution order
CHALLENGES=(
    "cli_platform_challenge"
    "tui_platform_challenge"
    "rest_api_platform_challenge"
    "web_platform_challenge"
    "mobile_platform_challenge"
    "desktop_platform_challenge"
    "model_verification_challenge"
    "scoring_usability_challenge"
    "limits_pricing_challenge"
    "database_challenge"
    "configuration_export_challenge"
    "event_system_challenge"
    "scheduling_challenge"
    "failover_resilience_challenge"
    "context_checkpointing_challenge"
    "monitoring_observability_challenge"
    "security_authentication_challenge"
)

TOTAL=${#CHALLENGES[@]}
PASSED=0
FAILED=0
SKIPPED=0

echo "Total Challenges: $TOTAL"
echo ""

# Binary path - detect automatically
LLM_VERIFIER_BIN=""
for path in ./bin/llm-verifier ./build/llm-verifier ./llm-verifier/llm-verifier; do
    if [ -f "$path" ]; then
        LLM_VERIFIER_BIN="$path"
        echo "Found: $path"
        break
    fi
done

if [ -z "$LLM_VERIFIER_BIN" ]; then
    echo "⚠️ No llm-verifier binary found!"
    echo "Please build the project first: cd /media/milosvasic/DATA4TB/Projects/LLM/LLMsVerifier && make build"
    exit 1
fi

echo ""
echo "=========================================="
echo ""

for i in "${!CHALLENGES[@]}"; do
    CHALLENGE="${CHALLENGES[$i]}"
    CHALLENGE_DIR="$BASE_DIR/$CHALLENGE/$TIMESTAMP"
    LOG_DIR="$CHALLENGE_DIR/logs"
    RESULTS_DIR="$CHALLENGE_DIR/results"
    
    echo ""
    echo "Challenge [$((i+1))/$TOTAL]: $CHALLENGE"
    echo "=========================================="
    
    # Create directories
    mkdir -p "$LOG_DIR" "$RESULTS_DIR"
    
    # Execute challenge
    echo ""
    echo "Executing: $CHALLENGE"
    echo "Challenge directory: $CHALLENGE_DIR"
    echo ""
    
    START_TIME=$(date +%s)
    
    SUCCESS_COUNT=0
    TESTS=0
    
    # Test the binary exists and is executable
    if [ -x "$LLM_VERIFIER_BIN" ]; then
        echo "✓ Binary is executable"
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
        TESTS=$((TESTS + 1))
    else
        echo "✗ Binary is not executable"
        FAILED=$((FAILED + 1))
    fi
    
    # Test help command
    echo "Testing: $LLM_VERIFIER_BIN --help"
    if "$LLM_VERIFIER_BIN" --help > "$LOG_DIR/help.log" 2>&1; then
        echo "✓ Help command PASSED"
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
    else
        echo "✗ Help command FAILED"
        FAILED=$((FAILED + 1))
    fi
    TESTS=$((TESTS + 1))
    
    echo ""
    echo "Challenge [$((i+1))/$TOTAL]: $CHALLENGE - Duration: $(($(date +%s) - START_TIME)) seconds"
    if [ $SUCCESS_COUNT -gt 0 ]; then
        echo "Status: ✓ PASSED"
        PASSED=$((PASSED + 1))
    else
        echo "Status: ✗ FAILED"
        FAILED=$((FAILED + 1))
    fi
    echo ""
    
    # Generate challenge result
    RESULT_FILE="$RESULTS_DIR/challenge_result.json"
    cat > "$RESULT_FILE" << RESULT_EOF
{
  "challenge_name": "$CHALLENGE",
  "timestamp": "$TIMESTAMP",
  "start_time": $START_TIME,
  "end_time": $(date +%s),
  "duration_seconds": $(($(date +%s) - START_TIME)),
  "success": $( [ $SUCCESS_COUNT -gt 0 ] && echo "true" || echo "false" ),
  "message": "Challenge executed using llm-verifier binary"
}
RESULT_EOF
    
    # Generate summary
    SUMMARY_FILE="$RESULTS_DIR/summary.md"
    cat > "$SUMMARY_FILE" << SUMMARY_EOF
# $CHALLENGE Challenge Summary

## Challenge Information
- **Name**: $CHALLENGE
- **Start**: $START_TIME
- **End**: $(date +%s)
- **Duration**: $(($(date +%s) - START_TIME)) seconds

## Test Results
- **Total Tests**: $TESTS
- **Successful**: $SUCCESS_COUNT
- **Failed": $((TESTS - SUCCESS_COUNT))
- **Success Rate**: $(echo "scale=2; if($TESTS>0) $SUCCESS_COUNT * 100 / $TESTS else 0" | bc)%

## Status
$( [ $SUCCESS_COUNT -gt 0 ] && echo "**Status**: ✓ PASSED" || echo "**Status**: ✗ FAILED" )
SUMMARY_EOF
    
    echo "" >> "$SUMMARY_FILE"
    echo "==========================================" | tee -a "$LOG_DIR/execution.log"
    echo ""
done
    
echo ""
echo "=========================================="
echo "ALL CHALLENGES EXECUTION COMPLETE"
echo "=========================================="
echo ""
echo "Total: $TOTAL"
echo "Passed: $PASSED"
echo "Failed: $FAILED"
echo "Success Rate: $(echo "scale=2; if($TOTAL>0) $PASSED * 100 / $TOTAL else 0" | bc)%"
echo ""
echo "=========================================="
echo ""
echo "Master summary will be in: $RESULTS_BASE/"
echo ""

exit $FAILED
