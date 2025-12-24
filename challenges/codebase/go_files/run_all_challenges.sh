#!/bin/bash
# Master Challenge Runner
# Executes all challenges in proper order

set -e

echo "=========================================="
echo "LLM Verifier - All Challenges Runner"
echo "=========================================="
echo ""

BASE_DIR="challenges"
TIMESTAMP=$(date +%Y/%m/%d/%N)

# List of all challenges in execution order
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

for i in "${!CHALLENGES[@]}"; do
    CHALLENGE="${CHALLENGES[$i]}"
    echo ""
    echo "=========================================="
    echo "Challenge [$((i+1))/$TOTAL]: $CHALLENGE"
    echo "=========================================="
    
    # Create challenge directory
    CHALLENGE_DIR="$BASE_DIR/$CHALLENGE/$TIMESTAMP"
    mkdir -p "$CHALLENGE_DIR/logs" "$CHALLENGE_DIR/results"
    
    # Run challenge
    if [ -f "$BASE_DIR/codebase/go_files/${CHALLENGE}.go" ]; then
        go run "$BASE_DIR/codebase/go_files/${CHALLENGE}.go" "$CHALLENGE_DIR" > "$CHALLENGE_DIR/logs/output.log" 2>&1
        RESULT=$?
    elif [ -f "$BASE_DIR/codebase/go_files/${CHALLENGE}.sh" ]; then
        bash "$BASE_DIR/codebase/go_files/${CHALLENGE}.sh" "$CHALLENGE_DIR" > "$CHALLENGE_DIR/logs/output.log" 2>&1
        RESULT=$?
    else
        echo "Challenge implementation not found: $CHALLENGE"
        RESULT=1
    fi
    
    if [ $RESULT -eq 0 ]; then
        echo "✓ Challenge PASSED"
        PASSED=$((PASSED + 1))
    else
        echo "✗ Challenge FAILED"
        FAILED=$((FAILED + 1))
    fi
done

echo ""
echo "=========================================="
echo "ALL CHALLENGES COMPLETE"
echo "=========================================="
echo "Total: $TOTAL"
echo "Passed: $PASSED"
echo "Failed: $FAILED"
echo "Success Rate: $(echo "scale=2; $PASSED * 100 / $TOTAL" | bc)%"
echo ""
echo "See results in: $BASE_DIR"
echo "=========================================="

# Generate master summary
JSON_OUT="challenges/master_summary_$TIMESTAMP.json"
MD_OUT="challenges/master_summary_$TIMESTAMP.md"

# Generate JSON summary
echo '{' > "$JSON_OUT"
echo '  "timestamp": "'$TIMESTAMP'",' >> "$JSON_OUT"
echo '  "total_challenges": '$TOTAL',' >> "$JSON_OUT"
echo '  "passed": '$PASSED',' >> "$JSON_OUT"
echo '  "failed": '$FAILED',' >> "$JSON_OUT"
echo '  "success_rate": '$(echo "scale=4; $PASSED * 100 / $TOTAL" | bc)',' >> "$JSON_OUT"
echo '  "challenges": [' >> "$JSON_OUT"

for i in "${!CHALLENGES[@]}"; do
    CHALLENGE_DIR="$BASE_DIR/${CHALLENGES[$i]}/$TIMESTAMP"
    STATUS="not_run"
    RESULTS_LINK="[N/A]"
    
    if [ -d "$CHALLENGE_DIR/results" ]; then
        STATUS="passed"
        RESULTS_LINK="[Results](${CHALLENGES[$i]}/$TIMESTAMP/results)"
    fi
    
    echo '    {"name": "'${CHALLENGES[$i]}'", "status": "'$STATUS'", "results_dir": "'$CHALLENGE_DIR/results'"},' >> "$JSON_OUT"
done

echo '  ]' >> "$JSON_OUT"
echo '}' >> "$JSON_OUT"

# Generate Markdown summary
echo "# LLM Verifier - All Challenges Master Summary" > "$MD_OUT"
echo "" >> "$MD_OUT"
echo "## Execution Summary" >> "$MD_OUT"
echo "- **Timestamp**: $TIMESTAMP" >> "$MD_OUT"
echo "- **Total Challenges**: $TOTAL" >> "$MD_OUT"
echo "- **Passed**: $PASSED" >> "$MD_OUT"
echo "- **Failed**: $FAILED" >> "$MD_OUT"
echo "- **Success Rate**: $(echo "scale=2; $PASSED * 100 / $TOTAL" | bc)%" >> "$MD_OUT"
echo "" >> "$MD_OUT"
echo "## Challenge Results" >> "$MD_OUT"
echo "" >> "$MD_OUT"
echo "| # | Challenge | Status | Results |" >> "$MD_OUT"
echo "|---|-----------|--------|---------|" >> "$MD_OUT"

for i in "${!CHALLENGES[@]}"; do
    CHALLENGE_DIR="$BASE_DIR/${CHALLENGES[$i]}/$TIMESTAMP"
    STATUS_ICON="⚠️"
    RESULTS_LINK="[N/A]"
    
    if [ -d "$CHALLENGE_DIR/results" ]; then
        STATUS_ICON="✅"
        RESULTS_LINK="[Results](${CHALLENGES[$i]}/$TIMESTAMP/results)"
    fi
    
    echo "| $((i+1)) | ${CHALLENGES[$i]} | $STATUS_ICON | $RESULTS_LINK |" >> "$MD_OUT"
done

echo "" >> "$MD_OUT"
echo "## Detailed Results" >> "$MD_OUT"
echo "Each challenge directory contains:" >> "$MD_OUT"
echo "- \`logs/\` - All execution logs" >> "$MD_OUT"
echo "- \`results/\` - JSON and Markdown reports" >> "$MD_OUT"
echo "" >> "$MD_OUT"
echo "## Running All Challenges" >> "$MD_OUT"
echo "To run all challenges:" >> "$MD_OUT"
echo "\`\`\`bash" >> "$MD_OUT"
echo "cd $(pwd) && bash challenges/codebase/go_files/run_all_challenges.sh" >> "$MD_OUT"
echo "\`\`\`" >> "$MD_OUT"
echo "" >> "$MD_OUT"
echo "## Running Individual Challenges" >> "$MD_OUT"
echo "To run a specific challenge:" >> "$MD_OUT"
echo "\`\`\`bash" >> "$MD_OUT"
echo "cd $(pwd) && go run challenges/codebase/go_files/<challenge_name>.go <challenge_dir>" >> "$MD_OUT"
echo "\`\`\`" >> "$MD_OUT"

echo ""
echo "Master summary created:"
echo "  - JSON: $JSON_OUT"
echo "  - Markdown: $MD_OUT"
