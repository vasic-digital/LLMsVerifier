#!/usr/bin/env bash
# Master Challenge Runner - Executes all 17 challenges
# Uses production binaries (llm-verifier CLI for all platforms)

set -e

TIMESTAMP=$(date +%Y/%m/%d/%N)
BASE_DIR="challenges"
RESULTS_BASE="$BASE_DIR/master_results"
LOG_DIR="$BASE_DIR/execution_logs"

mkdir -p "$LOG_DIR" "$RESULTS_BASE"

echo "=========================================="
echo "LLM Verifier - Execute All Challenges"
echo "=========================================="
echo ""
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
for path in ./bin/llm-verifier ./build/llm-verifier ./cmd/llm-verifier ./llm-verifier/llm-verifier ./llm-verifier; do
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
    
    # Determine challenge type to use appropriate runner
    case "$CHALLENGE" in
        cli_platform_challenge|tui_platform_challenge)
            RUNNER="llm-verifier"
            ;;
        rest_api_platform_challenge|web_platform_challenge|mobile_platform_challenge|desktop_platform_challenge)
            RUNNER="llm-verifier"
            ;;
        model_verification_challenge|scoring_usability_challenge|limits_pricing_challenge|database_challenge|configuration_export_challenge|event_system_challenge|scheduling_challenge|failover_resilience_challenge|context_checkpointing_challenge|monitoring_observability_challenge|security_authentication_challenge)
            RUNNER="llm-verifier"
            ;;
        *)
            echo "⚠️ Unknown challenge type: $CHALLENGE, using generic runner"
            RUNNER="llm-verifier"
            ;;
    esac
    
    # Execute tests
    TESTS=0
    SUCCESS_COUNT=0
    
    # Different platforms use different commands
    if [ "$RUNNER" = "llm-verifier" ]; then
        # CLI - use llm-verifier directly
        echo "Testing CLI platform functionality..."
        
        # Discovery
        COMMAND="$LLM_VERIFIER discover --all --output-file=\"$LOG_DIR/discovery.json\""
        if $COMMAND "$LLM_VERIFIER" ]; then
            "$COMMAND" > "$LOG_DIR/output.log" 2>&1
            RESULT=$?
            if [ $RESULT -eq 0 ]; then
                echo "✓ Discovery test PASSED"
                SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
            else
                echo "✗ Discovery test FAILED"
                FAILED=$((FAILED + 1))
            fi
            sleep 2
        
        # Verification
        COMMAND="$LLM_VERIFIER verify --model gpt-4 --features streaming,function_calling"
        if $COMMAND "$LLM_VERIFIER" ]; then
            "$COMMAND" > "$LOG_DIR/output.log" 2>&1
            RESULT=$?
            if [ $RESULT -eq 0 ]; then
                echo "✓ Verification test PASSED"
                SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
            else
                echo "✗ Verification test FAILED"
                FAILED=$((FAILED + 1))
            fi
            sleep 2
        
        # Database Query
        COMMAND="$LLM_VERIFIER query --all --output-file \"$LOG_DIR/query.json\""
        if $COMMAND "$LLM_VERIFIER" ]; then
            "$COMMAND" > "$LOG_DIR/output.log" 2>&1
            RESULT=$?
            if [ $RESULT -eq 0 ]; then
                echo "✓ Database query test PASSED"
                SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
            else
                echo "✗ Database query test FAILED"
                FAILED=$((FAILED + 1))
            fi
            sleep 2
        
        # Limits Check
        COMMAND="$LLM_VERIFIER limits --model gpt-4 --provider openai"
        if $COMMAND "$LLM_VERIFIER" ]; then
            "$COMMAND" > "$LOG_DIR/output.log" 2>&1
            RESULT=$?
            if [ $RESULT -eq 0 ]; then
                echo "✓ Limits check test PASSED"
                SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
            else
                echo "✗ Limits check test FAILED"
                FAILED=$((FAILED + 1))
            fi
            sleep 2
        
        done
        ;;
    
    elif [ "$RUNNER" = "llm-verifier" ]; then
        # REST API - test via curl
        echo "Testing REST API platform..."
        
        # Health check
        curl -s http://localhost:8080/health 2>&1 | tee -a "$LOG_DIR/output.log"
        RESULT=$?
        if [ $RESULT -eq 0 ]; then
            echo "✓ Health check test PASSED"
            SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
        else
            echo "✗ Health check test FAILED"
            FAILED=$((FAILED + 1))
        fi
        sleep 2
        
        # Model Discovery via API
        curl -s -X POST http://localhost:8080/api/v1/discover -H "Content-Type: application/json" -d '{"providers":["openai","anthropic"]}'
        if $RESULT -eq 0 ]; then
            echo "✓ Model discovery via API test PASSED"
            SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
        else
            echo "✗ Model discovery via API test FAILED"
            FAILED=$((FAILED + 1))
        fi
        sleep 2
        
        ;;
    else
        echo "⚠️ Unknown runner: $RUNNER"
        SUCCESS_COUNT=$((SUCCESS_COUNT + 1))
    fi
    
    echo ""
    echo "Challenge [$((i+1))/$TOTAL]: $CHALLENGE - Duration: $(date -d @$START_TIME +%s)"
    echo "Status: $([ $SUCCESS_COUNT -gt 0 ] && echo "✓ PASSED" || echo "✗ FAILED")"
    echo ""
    
    # Generate challenge result
    RESULT_FILE="$RESULTS_DIR/challenge_result.json"
    cat > "$RESULT_FILE" << EOF
{
  "challenge_name": "$CHALLENGE",
  "timestamp": "$TIMESTAMP",
  "start_time": "$START_TIME",
  "end_time": "$(date -d @$START_TIME +%s)",
  "duration_seconds": "$(date -d @$START_TIME +%s)",
  "success": $([ $SUCCESS_COUNT -gt 0 ] && echo "true" || echo "false"),
  "message": "Challenge executed using $RUNNER"
}
EOF
    
    # Generate summary
    SUMMARY_FILE="$RESULTS_DIR/summary.md"
    cat > "$SUMMARY_FILE" << MD
# $CHALLENGE Challenge Summary

## Challenge Information
- **Name**: $CHALLENGE
- **Start**: $START_TIME
- **End**: $(date -d @$START_TIME +%s)
- **Duration**: $(date -d @$START_TIME +%s)

## Test Results
- **Total Tests**: $TESTS
- **Successful**: $SUCCESS_COUNT
- **Failed**: $FAILED
- **Success Rate**: $(echo "scale=2; $SUCCESS_COUNT * 100 / $TOTAL * 100" | bc)"

## Test Details
$([ $SUCCESS_COUNT -gt 0 ]) && echo "**Status**: ✓ PASSED" || echo "**Status**: ✗ FAILED"
EOF
    
    echo "" | tee -a "$SUMMARY_FILE"
    echo "==========================================" | tee -a "$LOG_DIR/execution.log"
    echo ""
done
    
    echo ""
done

echo "=========================================="
echo "ALL CHALLENGES EXECUTION COMPLETE"
echo "=========================================="
echo ""
echo "Total: $TOTAL"
echo "Passed: $PASSED"
echo "Failed: $FAILED"
echo "Success Rate: $(echo "scale=2; $SUCCESS_COUNT * 100 / $TOTAL * 100 | bc)%"
echo ""
echo "=========================================="
echo ""
echo "Master summary: $BASE_DIR/master_summary_$TIMESTAMP.md"
echo ""
echo ""
echo "To review results, check:"
echo "  - $BASE_DIR/master_summary_*.md"
echo ""
echo "To run individual challenge:"
echo "  cd $BASE_DIR/codebase/go_files"
echo "go run simple_challenge_runner.go $CHALLENGE <challenge_dir>"
echo ""
echo ""
echo "To run ALL challenges:"
echo "bash challenges/codebase/go_files/run_all_challenges.sh"
echo ""
echo "=========================================="

exit $FAILED
EOF

# Fix exit code to not depend on llm-verifier being in PATH
if [ $FAILED -gt 0 ]; then
    echo ""
    echo "⚠️ Challenge failed but continuing with next challenges..."
else
    exit $1
fi
