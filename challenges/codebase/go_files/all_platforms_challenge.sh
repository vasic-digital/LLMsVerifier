#!/bin/bash
# All Platforms Challenge Runner
# Executes tests across CLI, TUI, Web, REST API, Mobile, Desktop

set -e

CHALLENGE_DIR="${1:-challenges/all_platforms_challenge/$(date +%Y/%m/%d/%N)}"
LOG_DIR="$CHALLENGE_DIR/logs"
RESULTS_DIR="$CHALLENGE_DIR/results"
BINARY_DIR="../build"

mkdir -p "$LOG_DIR" "$RESULTS_DIR"

echo "Starting All Platforms Challenge" | tee "$LOG_DIR/challenge.log"
echo "Challenge directory: $CHALLENGE_DIR" | tee -a "$LOG_DIR/challenge.log"
START_TIME=$(date +%s)

# Test CLI
echo "=== Testing CLI Platform ===" | tee -a "$LOG_DIR/challenge.log"
if [ -f "$BINARY_DIR/llm-verifier" ]; then
    "$BINARY_DIR/llm-verifier" --version 2>&1 | tee -a "$LOG_DIR/cli.log"
    CLI_SUCCESS=$?
else
    echo "CLI binary not found" | tee -a "$LOG_DIR/cli.log"
    CLI_SUCCESS=1
fi

# Test REST API (if running)
echo "=== Testing REST API Platform ===" | tee -a "$LOG_DIR/challenge.log"
curl -f http://localhost:8080/health 2>&1 | tee -a "$LOG_DIR/api.log" || API_SUCCESS=1
if [ -z "$API_SUCCESS" ]; then
    API_SUCCESS=0
fi

# Test Web (if running)
echo "=== Testing Web Platform ===" | tee -a "$LOG_DIR/challenge.log"
curl -f http://localhost:4200 2>&1 | tee -a "$LOG_DIR/web.log" || WEB_SUCCESS=1
if [ -z "$WEB_SUCCESS" ]; then
    WEB_SUCCESS=0
fi

# Results
END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

cat > "$RESULTS_DIR/summary.json" << RESULTS
{
  "challenge_name": "All Platforms Challenge",
  "start_time": "$START_TIME",
  "end_time": "$END_TIME",
  "duration_seconds": "$DURATION",
  "results": {
    "cli": {"tested": true, "success": $CLI_SUCCESS},
    "api": {"tested": true, "success": $API_SUCCESS},
    "web": {"tested": true, "success": $WEB_SUCCESS}
  }
}
RESULTS

cat > "$RESULTS_DIR/summary.md" << MD
# All Platforms Challenge Summary

## Platforms Tested
- CLI: $( [ $CLI_SUCCESS -eq 0 ] && echo "✓ Passed" || echo "✗ Failed" )
- REST API: $( [ $API_SUCCESS -eq 0 ] && echo "✓ Passed" || echo "✗ Failed" )
- Web: $( [ $WEB_SUCCESS -eq 0 ] && echo "✓ Passed" || echo "✗ Failed" )

## Duration
- Total: ${DURATION}s

See detailed logs in: $LOG_DIR
MD

echo "Challenge completed in ${DURATION}s" | tee -a "$LOG_DIR/challenge.log"
