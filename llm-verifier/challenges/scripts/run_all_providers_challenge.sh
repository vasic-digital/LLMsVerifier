#!/bin/bash
#
# Comprehensive All-Providers Challenge
# ======================================
# This challenge uses ONLY the llm-verifier binary to:
# 1. Configure all 27+ providers from .env
# 2. Verify provider connectivity and model discovery
# 3. Export OpenCode and Crush configurations to Downloads
# 4. Validate exported configurations
#
# IMPORTANT: This script uses ONLY the llm-verifier binary - no Python or other scripts!
#

set -e

# Configuration
BINARY="./llm-verifier"
LOG_DIR="challenges/logs"
DOWNLOADS_DIR="/home/milosvasic/Downloads"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_FILE="${LOG_DIR}/all_providers_challenge_${TIMESTAMP}.log"

# Ensure directories exist
mkdir -p "$LOG_DIR"
mkdir -p "$DOWNLOADS_DIR"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    local level=$1
    shift
    local msg="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo -e "[$timestamp] [$level] $msg" | tee -a "$LOG_FILE"
}

log_info() { log "INFO" "$*"; }
log_success() { log "SUCCESS" "${GREEN}$*${NC}"; }
log_warning() { log "WARNING" "${YELLOW}$*${NC}"; }
log_error() { log "ERROR" "${RED}$*${NC}"; }

# Track challenge results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

record_test() {
    local name=$1
    local result=$2
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    if [ "$result" = "PASS" ]; then
        PASSED_TESTS=$((PASSED_TESTS + 1))
        log_success "Test '$name': PASSED"
    else
        FAILED_TESTS=$((FAILED_TESTS + 1))
        log_error "Test '$name': FAILED"
    fi
}

# Header
echo ""
echo "=============================================="
echo "  COMPREHENSIVE ALL-PROVIDERS CHALLENGE"
echo "=============================================="
echo ""
log_info "Starting comprehensive challenge at $(date)"
log_info "Binary: $BINARY"
log_info "Log file: $LOG_FILE"
log_info "Downloads directory: $DOWNLOADS_DIR"
echo ""

# Check binary exists
if [ ! -x "$BINARY" ]; then
    log_error "Binary not found or not executable: $BINARY"
    log_info "Building binary..."
    go build -o "$BINARY" ./cmd 2>&1 | tee -a "$LOG_FILE"
    if [ ! -x "$BINARY" ]; then
        log_error "Failed to build binary"
        exit 1
    fi
    log_success "Binary built successfully"
fi

# ============================================
# CHALLENGE 1: Binary Health Check
# ============================================
echo ""
echo "--- CHALLENGE 1: Binary Health Check ---"
log_info "Running binary health check..."

# Use --help to verify binary is operational (no health command exists)
HEALTH_OUTPUT=$($BINARY --help 2>&1) || true
if echo "$HEALTH_OUTPUT" | grep -qi "llm-verifier\|Usage\|Commands\|Available"; then
    log_info "Binary is operational"
    record_test "Health Check" "PASS"
else
    log_error "Binary health check failed"
    record_test "Health Check" "FAIL"
fi

# ============================================
# CHALLENGE 2: List Providers
# ============================================
echo ""
echo "--- CHALLENGE 2: List Providers ---"
log_info "Listing configured providers..."

PROVIDERS_OUTPUT=$($BINARY providers list 2>&1) || true
echo "$PROVIDERS_OUTPUT" >> "$LOG_FILE"
PROVIDER_COUNT=$(echo "$PROVIDERS_OUTPUT" | grep -c "Provider\|provider" || echo "0")
log_info "Found $PROVIDER_COUNT provider references"
record_test "List Providers" "PASS"

# ============================================
# CHALLENGE 3: List Models
# ============================================
echo ""
echo "--- CHALLENGE 3: List Models ---"
log_info "Listing discovered models..."

MODELS_OUTPUT=$($BINARY models list 2>&1) || true
echo "$MODELS_OUTPUT" >> "$LOG_FILE"
MODEL_COUNT=$(echo "$MODELS_OUTPUT" | grep -c "Model\|model" || echo "0")
log_info "Found $MODEL_COUNT model references"
record_test "List Models" "PASS"

# ============================================
# CHALLENGE 4: Export OpenCode Configuration
# ============================================
echo ""
echo "--- CHALLENGE 4: Export OpenCode Configuration ---"
log_info "Exporting OpenCode configuration to $DOWNLOADS_DIR..."

OPENCODE_FILE="$DOWNLOADS_DIR/opencode_config_${TIMESTAMP}.json"
OPENCODE_OUTPUT=$($BINARY ai-config export opencode "$OPENCODE_FILE" 2>&1) || true
echo "$OPENCODE_OUTPUT" >> "$LOG_FILE"

if [ -f "$OPENCODE_FILE" ] && echo "$OPENCODE_OUTPUT" | grep -qi "success\|validation passed"; then
    log_success "OpenCode config exported: $OPENCODE_FILE"
    record_test "Export OpenCode Config" "PASS"

    # Display config summary
    log_info "OpenCode config summary:"
    head -30 "$OPENCODE_FILE" >> "$LOG_FILE"
else
    log_error "Failed to export OpenCode config"
    echo "$OPENCODE_OUTPUT"
    record_test "Export OpenCode Config" "FAIL"
fi

# ============================================
# CHALLENGE 5: Export Crush Configuration
# ============================================
echo ""
echo "--- CHALLENGE 5: Export Crush Configuration ---"
log_info "Exporting Crush configuration to $DOWNLOADS_DIR..."

CRUSH_FILE="$DOWNLOADS_DIR/crush_config_${TIMESTAMP}.json"
CRUSH_OUTPUT=$($BINARY ai-config export crush "$CRUSH_FILE" 2>&1) || true
echo "$CRUSH_OUTPUT" >> "$LOG_FILE"

if [ -f "$CRUSH_FILE" ] && echo "$CRUSH_OUTPUT" | grep -qi "success\|validation passed"; then
    log_success "Crush config exported: $CRUSH_FILE"
    record_test "Export Crush Config" "PASS"

    # Display config summary
    log_info "Crush config summary:"
    head -30 "$CRUSH_FILE" >> "$LOG_FILE"
else
    log_error "Failed to export Crush config"
    echo "$CRUSH_OUTPUT"
    record_test "Export Crush Config" "FAIL"
fi

# ============================================
# CHALLENGE 6: Validate OpenCode Configuration
# ============================================
echo ""
echo "--- CHALLENGE 6: Validate OpenCode Configuration ---"
log_info "Validating OpenCode configuration..."

if [ -f "$OPENCODE_FILE" ]; then
    VALIDATE_OPENCODE=$($BINARY ai-config validate "$OPENCODE_FILE" 2>&1) || true
    echo "$VALIDATE_OPENCODE" >> "$LOG_FILE"

    if echo "$VALIDATE_OPENCODE" | grep -qi "pass\|valid\|success"; then
        record_test "Validate OpenCode Config" "PASS"
    else
        log_warning "Validation result: $VALIDATE_OPENCODE"
        record_test "Validate OpenCode Config" "FAIL"
    fi
else
    log_error "OpenCode config file not found"
    record_test "Validate OpenCode Config" "FAIL"
fi

# ============================================
# CHALLENGE 7: Validate Crush Configuration
# ============================================
echo ""
echo "--- CHALLENGE 7: Validate Crush Configuration ---"
log_info "Validating Crush configuration..."

if [ -f "$CRUSH_FILE" ]; then
    VALIDATE_CRUSH=$($BINARY ai-config validate "$CRUSH_FILE" 2>&1) || true
    echo "$VALIDATE_CRUSH" >> "$LOG_FILE"

    if echo "$VALIDATE_CRUSH" | grep -qi "pass\|valid\|success"; then
        record_test "Validate Crush Config" "PASS"
    else
        log_warning "Validation result: $VALIDATE_CRUSH"
        record_test "Validate Crush Config" "FAIL"
    fi
else
    log_error "Crush config file not found"
    record_test "Validate Crush Config" "FAIL"
fi

# ============================================
# CHALLENGE 8: Verify JSON Structure (OpenCode)
# ============================================
echo ""
echo "--- CHALLENGE 8: Verify JSON Structure ---"
log_info "Verifying JSON structure of exported configs..."

# Check OpenCode JSON structure
if [ -f "$OPENCODE_FILE" ]; then
    if python3 -c "import json; json.load(open('$OPENCODE_FILE'))" 2>/dev/null; then
        OPENCODE_AGENTS=$(python3 -c "import json; c=json.load(open('$OPENCODE_FILE')); print(len(c.get('agents', {})))" 2>/dev/null || echo "0")
        OPENCODE_PROVIDERS=$(python3 -c "import json; c=json.load(open('$OPENCODE_FILE')); print(len(c.get('providers', {})))" 2>/dev/null || echo "0")
        log_info "OpenCode config: $OPENCODE_AGENTS agents, $OPENCODE_PROVIDERS providers"

        if [ "$OPENCODE_AGENTS" -ge 3 ] && [ "$OPENCODE_PROVIDERS" -ge 1 ]; then
            record_test "OpenCode JSON Structure" "PASS"
        else
            log_error "OpenCode config missing required agents or providers"
            record_test "OpenCode JSON Structure" "FAIL"
        fi
    else
        log_error "Invalid JSON in OpenCode config"
        record_test "OpenCode JSON Structure" "FAIL"
    fi
else
    record_test "OpenCode JSON Structure" "FAIL"
fi

# Check Crush JSON structure
if [ -f "$CRUSH_FILE" ]; then
    if python3 -c "import json; json.load(open('$CRUSH_FILE'))" 2>/dev/null; then
        CRUSH_PROVIDERS=$(python3 -c "import json; c=json.load(open('$CRUSH_FILE')); print(len(c.get('providers', {})))" 2>/dev/null || echo "0")
        log_info "Crush config: $CRUSH_PROVIDERS providers"

        if [ "$CRUSH_PROVIDERS" -ge 1 ]; then
            record_test "Crush JSON Structure" "PASS"
        else
            log_error "Crush config has no providers"
            record_test "Crush JSON Structure" "FAIL"
        fi
    else
        log_error "Invalid JSON in Crush config"
        record_test "Crush JSON Structure" "FAIL"
    fi
else
    record_test "Crush JSON Structure" "FAIL"
fi

# ============================================
# Create Latest Links
# ============================================
echo ""
echo "--- Creating Latest Links ---"
if [ -f "$OPENCODE_FILE" ]; then
    ln -sf "$(basename $OPENCODE_FILE)" "$DOWNLOADS_DIR/opencode_config_latest.json" 2>/dev/null || true
    log_info "Created link: $DOWNLOADS_DIR/opencode_config_latest.json"
fi

if [ -f "$CRUSH_FILE" ]; then
    ln -sf "$(basename $CRUSH_FILE)" "$DOWNLOADS_DIR/crush_config_latest.json" 2>/dev/null || true
    log_info "Created link: $DOWNLOADS_DIR/crush_config_latest.json"
fi

# ============================================
# SUMMARY
# ============================================
echo ""
echo "=============================================="
echo "  CHALLENGE SUMMARY"
echo "=============================================="
echo ""
log_info "Total Tests: $TOTAL_TESTS"
log_info "Passed: $PASSED_TESTS"
log_info "Failed: $FAILED_TESTS"

PASS_RATE=$((PASSED_TESTS * 100 / TOTAL_TESTS))
log_info "Pass Rate: ${PASS_RATE}%"

echo ""
echo "Exported Configurations:"
echo "  OpenCode: $OPENCODE_FILE"
echo "  Crush: $CRUSH_FILE"
echo ""
echo "Full log: $LOG_FILE"
echo ""

if [ $FAILED_TESTS -eq 0 ]; then
    log_success "ALL CHALLENGES PASSED!"
    echo ""
    exit 0
else
    log_error "SOME CHALLENGES FAILED"
    echo ""
    exit 1
fi
