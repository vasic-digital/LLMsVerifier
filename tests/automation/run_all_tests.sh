#!/bin/bash

# Comprehensive Test Automation Script
# Runs all test suites with proper reporting and validation

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
TEST_RESULTS_DIR="${PROJECT_ROOT}/test_results"
COVERAGE_DIR="${PROJECT_ROOT}/coverage"
LOGS_DIR="${PROJECT_ROOT}/logs"

# Test configuration
UNIT_TEST_TIMEOUT=300
INTEGRATION_TEST_TIMEOUT=600
E2E_TEST_TIMEOUT=900
PERFORMANCE_TEST_TIMEOUT=300
SECURITY_TEST_TIMEOUT=300

# Counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0
SKIPPED_TESTS=0

# Initialize directories
init_directories() {
    echo -e "${BLUE}📁 Initializing test directories...${NC}"
    mkdir -p "$TEST_RESULTS_DIR" "$COVERAGE_DIR" "$LOGS_DIR"
    mkdir -p "$TEST_RESULTS_DIR/unit" "$TEST_RESULTS_DIR/integration" \
             "$TEST_RESULTS_DIR/e2e" "$TEST_RESULTS_DIR/performance" "$TEST_RESULTS_DIR/security"
}

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Test execution functions
run_unit_tests() {
    log_info "Running unit tests..."
    local start_time=$(date +%s)
    local test_log="$LOGS_DIR/unit_tests.log"
    
    cd "$PROJECT_ROOT"
    
    # Run unit tests with coverage
    if go test -v -race -coverprofile="$COVERAGE_DIR/unit_coverage.out" \
               -timeout="${UNIT_TEST_TIMEOUT}s" \
               ./tests/unit/... > "$test_log" 2>&1; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_success "Unit tests completed in ${duration}s"
        
        # Generate coverage report
        go tool cover -html="$COVERAGE_DIR/unit_coverage.out" -o "$COVERAGE_DIR/unit_coverage.html"
        
        # Extract test results
        extract_test_results "$test_log" "unit"
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_error "Unit tests failed after ${duration}s"
        extract_test_results "$test_log" "unit"
        return 1
    fi
}

run_integration_tests() {
    log_info "Running integration tests..."
    local start_time=$(date +%s)
    local test_log="$LOGS_DIR/integration_tests.log"
    
    cd "$PROJECT_ROOT"
    
    # Start test dependencies
    start_test_dependencies
    
    # Run integration tests
    if go test -v -race -tags=integration \
               -coverprofile="$COVERAGE_DIR/integration_coverage.out" \
               -timeout="${INTEGRATION_TEST_TIMEOUT}s" \
               ./tests/integration/... > "$test_log" 2>&1; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_success "Integration tests completed in ${duration}s"
        
        # Generate coverage report
        go tool cover -html="$COVERAGE_DIR/integration_coverage.out" -o "$COVERAGE_DIR/integration_coverage.html"
        
        extract_test_results "$test_log" "integration"
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_error "Integration tests failed after ${duration}s"
        extract_test_results "$test_log" "integration"
        return 1
    fi
}

run_e2e_tests() {
    log_info "Running end-to-end tests..."
    local start_time=$(date +%s)
    local test_log="$LOGS_DIR/e2e_tests.log"
    
    cd "$PROJECT_ROOT"
    
    # Run E2E tests
    if go test -v -race -tags=e2e \
               -timeout="${E2E_TEST_TIMEOUT}s" \
               ./tests/e2e/... > "$test_log" 2>&1; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_success "E2E tests completed in ${duration}s"
        extract_test_results "$test_log" "e2e"
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_error "E2E tests failed after ${duration}s"
        extract_test_results "$test_log" "e2e"
        return 1
    fi
}

run_performance_tests() {
    log_info "Running performance tests..."
    local start_time=$(date +%s)
    local test_log="$LOGS_DIR/performance_tests.log"
    
    cd "$PROJECT_ROOT"
    
    # Run performance tests
    if go test -v -race -tags=performance \
               -timeout="${PERFORMANCE_TEST_TIMEOUT}s" \
               -bench=. -benchmem \
               ./tests/performance/... > "$test_log" 2>&1; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_success "Performance tests completed in ${duration}s"
        extract_test_results "$test_log" "performance"
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_error "Performance tests failed after ${duration}s"
        extract_test_results "$test_log" "performance"
        return 1
    fi
}

run_security_tests() {
    log_info "Running security tests..."
    local start_time=$(date +%s)
    local test_log="$LOGS_DIR/security_tests.log"
    
    cd "$PROJECT_ROOT"
    
    # Run security tests
    if go test -v -race -tags=security \
               -timeout="${SECURITY_TEST_TIMEOUT}s" \
               ./tests/security/... > "$test_log" 2>&1; then
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_success "Security tests completed in ${duration}s"
        extract_test_results "$test_log" "security"
        return 0
    else
        local end_time=$(date +%s)
        local duration=$((end_time - start_time))
        log_error "Security tests failed after ${duration}s"
        extract_test_results "$test_log" "security"
        return 1
    fi
}

start_test_dependencies() {
    log_info "Starting test dependencies..."
    
    # Start mock services, databases, etc.
    # This is a placeholder - implement based on your specific needs
    
    # Example: Start a test database
    # docker run -d --name test-db -p 5432:5432 -e POSTGRES_DB=test postgres:13
    
    # Wait for services to be ready
    sleep 5
}

stop_test_dependencies() {
    log_info "Stopping test dependencies..."
    
    # Stop mock services, databases, etc.
    # This is a placeholder - implement based on your specific needs
    
    # Example: Stop test database
    # docker stop test-db && docker rm test-db
}

extract_test_results() {
    local log_file="$1"
    local test_type="$2"
    local results_file="$TEST_RESULTS_DIR/${test_type}_results.json"
    
    # Extract test counts from log
    local passed=$(grep -oP '\d+(?= passed)' "$log_file" | tail -1 || echo "0")
    local failed=$(grep -oP '\d+(?= failed)' "$log_file" | tail -1 || echo "0")
    local skipped=$(grep -oP '\d+(?= skipped)' "$log_file" | tail -1 || echo "0")
    
    # Create results JSON
    cat > "$results_file" << EOF
{
    "test_type": "$test_type",
    "passed": $passed,
    "failed": $failed,
    "skipped": $skipped,
    "total": $((passed + failed + skipped)),
    "log_file": "$log_file",
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
    
    # Update global counters
    TOTAL_TESTS=$((TOTAL_TESTS + passed + failed + skipped))
    PASSED_TESTS=$((PASSED_TESTS + passed))
    FAILED_TESTS=$((FAILED_TESTS + failed))
    SKIPPED_TESTS=$((SKIPPED_TESTS + skipped))
}

generate_coverage_report() {
    log_info "Generating combined coverage report..."
    
    cd "$PROJECT_ROOT"
    
    # Combine coverage files
    echo "mode: atomic" > "$COVERAGE_DIR/combined_coverage.out"
    
    for coverage_file in "$COVERAGE_DIR"/*_coverage.out; do
        if [ -f "$coverage_file" ] && [ "$coverage_file" != "$COVERAGE_DIR/combined_coverage.out" ]; then
            tail -n +2 "$coverage_file" >> "$COVERAGE_DIR/combined_coverage.out"
        fi
    done
    
    # Generate HTML report
    go tool cover -html="$COVERAGE_DIR/combined_coverage.out" -o "$COVERAGE_DIR/combined_coverage.html"
    
    # Generate coverage summary
    go tool cover -func="$COVERAGE_DIR/combined_coverage.out" | tail -n 1 > "$COVERAGE_DIR/coverage_summary.txt"
    
    local coverage_percent=$(cat "$COVERAGE_DIR/coverage_summary.txt" | awk '{print $3}')
    log_info "Total coverage: $coverage_percent"
}

generate_test_report() {
    log_info "Generating comprehensive test report..."
    
    local report_file="$TEST_RESULTS_DIR/test_report.html"
    local timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    
    cat > "$report_file" << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>LLM Verifier Test Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 20px; border-radius: 5px; }
        .summary { margin: 20px 0; }
        .test-results { margin: 20px 0; }
        .passed { color: green; }
        .failed { color: red; }
        .skipped { color: orange; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
        .coverage { margin: 20px 0; }
    </style>
</head>
<body>
    <div class="header">
        <h1>LLM Verifier Test Report</h1>
        <p>Generated: TIMESTAMP</p>
    </div>
    
    <div class="summary">
        <h2>Test Summary</h2>
        <table>
            <tr><th>Metric</th><th>Value</th></tr>
            <tr><td>Total Tests</td><td>TOTAL_TESTS</td></tr>
            <tr><td class="passed">Passed</td><td>PASSED_TESTS</td></tr>
            <tr><td class="failed">Failed</td><td>FAILED_TESTS</td></tr>
            <tr><td class="skipped">Skipped</td><td>SKIPPED_TESTS</td></tr>
            <tr><td>Success Rate</td><td>SUCCESS_RATE%</td></tr>
        </table>
    </div>
    
    <div class="test-results">
        <h2>Detailed Test Results</h2>
        <table>
            <tr><th>Test Type</th><th>Passed</th><th>Failed</th><th>Skipped</th><th>Total</th></tr>
EOF
    
    # Add test results
    for results_file in "$TEST_RESULTS_DIR"/*_results.json; do
        if [ -f "$results_file" ]; then
            local test_type=$(jq -r '.test_type' "$results_file")
            local passed=$(jq -r '.passed' "$results_file")
            local failed=$(jq -r '.failed' "$results_file")
            local skipped=$(jq -r '.skipped' "$results_file")
            local total=$(jq -r '.total' "$results_file")
            
            echo "            <tr>" >> "$report_file"
            echo "                <td>$test_type</td>" >> "$report_file"
            echo "                <td class=\"passed\">$passed</td>" >> "$report_file"
            echo "                <td class=\"failed\">$failed</td>" >> "$report_file"
            echo "                <td class=\"skipped\">$skipped</td>" >> "$report_file"
            echo "                <td>$total</td>" >> "$report_file"
            echo "            </tr>" >> "$report_file"
        fi
    done
    
    cat >> "$report_file" << 'EOF'
        </table>
    </div>
    
    <div class="coverage">
        <h2>Coverage Report</h2>
        <p>Total Coverage: COVERAGE_PERCENT</p>
        <p><a href="combined_coverage.html">View Detailed Coverage Report</a></p>
    </div>
    
    <div class="logs">
        <h2>Test Logs</h2>
        <ul>
EOF
    
    # Add log links
    for log_file in "$LOGS_DIR"/*.log; do
        if [ -f "$log_file" ]; then
            local log_name=$(basename "$log_file")
            echo "            <li><a href=\"../logs/$log_name\">$log_name</a></li>" >> "$report_file"
        fi
    done
    
    cat >> "$report_file" << 'EOF'
        </ul>
    </div>
</body>
</html>
EOF
    
    # Replace placeholders
    local success_rate=0
    if [ $TOTAL_TESTS -gt 0 ]; then
        success_rate=$((PASSED_TESTS * 100 / TOTAL_TESTS))
    fi
    
    local coverage_percent="0%"
    if [ -f "$COVERAGE_DIR/coverage_summary.txt" ]; then
        coverage_percent=$(cat "$COVERAGE_DIR/coverage_summary.txt" | awk '{print $3}')
    fi
    
    sed -i "s/TIMESTAMP/$timestamp/g" "$report_file"
    sed -i "s/TOTAL_TESTS/$TOTAL_TESTS/g" "$report_file"
    sed -i "s/PASSED_TESTS/$PASSED_TESTS/g" "$report_file"
    sed -i "s/FAILED_TESTS/$FAILED_TESTS/g" "$report_file"
    sed -i "s/SKIPPED_TESTS/$SKIPPED_TESTS/g" "$report_file"
    sed -i "s/SUCCESS_RATE/$success_rate/g" "$report_file"
    sed -i "s/COVERAGE_PERCENT/$coverage_percent/g" "$report_file"
    
    log_success "Test report generated: $report_file"
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check Go installation
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi
    
    # Check Go version
    local go_version=$(go version | awk '{print $3}')
    log_info "Go version: $go_version"
    
    # Check for required tools
    local required_tools=("jq" "curl")
    for tool in "${required_tools[@]}"; do
        if ! command -v "$tool" &> /dev/null; then
            log_warning "$tool is not installed"
        fi
    done
    
    # Check project structure
    if [ ! -f "$PROJECT_ROOT/go.mod" ]; then
        log_error "go.mod not found in project root"
        exit 1
    fi
    
    log_success "Prerequisites check completed"
}

validate_test_results() {
    log_info "Validating test results..."
    
    # Check if any critical tests failed
    if [ $FAILED_TESTS -gt 0 ]; then
        log_error "$FAILED_TESTS tests failed"
        
        # Check for critical failures
        for log_file in "$LOGS_DIR"/*.log; do
            if [ -f "$log_file" ]; then
                if grep -q "CRITICAL\|FATAL\|PANIC" "$log_file"; then
                    log_error "Critical failures found in $log_file"
                    return 1
                fi
            fi
        done
    fi
    
    # Check coverage threshold (minimum 95%)
    if [ -f "$COVERAGE_DIR/coverage_summary.txt" ]; then
        local coverage=$(cat "$COVERAGE_DIR/coverage_summary.txt" | awk '{print $3}' | sed 's/%//')
        if (( $(echo "$coverage < 95" | bc -l) )); then
            log_error "Coverage $coverage% is below minimum threshold of 95%"
            return 1
        fi
        log_success "Coverage $coverage% meets minimum threshold of 95%"
    fi
    
    # Check success rate (minimum 100% for critical tests)
    local success_rate=0
    if [ $TOTAL_TESTS -gt 0 ]; then
        success_rate=$((PASSED_TESTS * 100 / TOTAL_TESTS))
    fi
    
    if [ $success_rate -lt 100 ]; then
        log_error "Success rate $success_rate% is below required 100%"
        return 1
    fi
    
    log_success "All test validations passed"
    return 0
}

print_summary() {
    echo -e "\n${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}                     TEST EXECUTION SUMMARY                     ${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "Total Tests:    ${YELLOW}$TOTAL_TESTS${NC}"
    echo -e "Passed:         ${GREEN}$PASSED_TESTS${NC}"
    echo -e "Failed:         ${RED}$FAILED_TESTS${NC}"
    echo -e "Skipped:        ${YELLOW}$SKIPPED_TESTS${NC}"
    
    if [ $TOTAL_TESTS -gt 0 ]; then
        local success_rate=$((PASSED_TESTS * 100 / TOTAL_TESTS))
        echo -e "Success Rate:   ${GREEN}$success_rate%${NC}"
    fi
    
    if [ -f "$COVERAGE_DIR/coverage_summary.txt" ]; then
        local coverage=$(cat "$COVERAGE_DIR/coverage_summary.txt" | awk '{print $3}')
        echo -e "Coverage:       ${BLUE}$coverage${NC}"
    fi
    
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "Test Results:   ${BLUE}$TEST_RESULTS_DIR/test_report.html${NC}"
    echo -e "Coverage Report:${BLUE}$COVERAGE_DIR/combined_coverage.html${NC}"
    echo -e "Test Logs:      ${BLUE}$LOGS_DIR/${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

# Main execution
main() {
    echo -e "${BLUE}🚀 LLM Verifier Comprehensive Test Suite${NC}"
    echo -e "${BLUE}========================================${NC}\n"
    
    # Parse command line arguments
    local run_unit=true
    local run_integration=true
    local run_e2e=true
    local run_performance=true
    local run_security=true
    local skip_cleanup=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --unit-only)
                run_integration=false
                run_e2e=false
                run_performance=false
                run_security=false
                shift
                ;;
            --integration-only)
                run_unit=false
                run_e2e=false
                run_performance=false
                run_security=false
                shift
                ;;
            --e2e-only)
                run_unit=false
                run_integration=false
                run_performance=false
                run_security=false
                shift
                ;;
            --performance-only)
                run_unit=false
                run_integration=false
                run_e2e=false
                run_security=false
                shift
                ;;
            --security-only)
                run_unit=false
                run_integration=false
                run_e2e=false
                run_performance=false
                shift
                ;;
            --skip-cleanup)
                skip_cleanup=true
                shift
                ;;
            --help)
                echo "Usage: $0 [options]"
                echo "Options:"
                echo "  --unit-only          Run only unit tests"
                echo "  --integration-only   Run only integration tests"
                echo "  --e2e-only          Run only end-to-end tests"
                echo "  --performance-only   Run only performance tests"
                echo "  --security-only      Run only security tests"
                echo "  --skip-cleanup       Skip cleanup after tests"
                echo "  --help              Show this help message"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Initialize
    init_directories
    check_prerequisites
    
    # Start test dependencies
    start_test_dependencies
    
    # Run test suites
    local exit_code=0
    
    if [ "$run_unit" = true ]; then
        run_unit_tests || exit_code=1
    fi
    
    if [ "$run_integration" = true ]; then
        run_integration_tests || exit_code=1
    fi
    
    if [ "$run_e2e" = true ]; then
        run_e2e_tests || exit_code=1
    fi
    
    if [ "$run_performance" = true ]; then
        run_performance_tests || exit_code=1
    fi
    
    if [ "$run_security" = true ]; then
        run_security_tests || exit_code=1
    fi
    
    # Generate reports
    generate_coverage_report
    generate_test_report
    
    # Validate results
    if ! validate_test_results; then
        exit_code=1
    fi
    
    # Cleanup
    if [ "$skip_cleanup" = false ]; then
        stop_test_dependencies
    fi
    
    # Print summary
    print_summary
    
    if [ $exit_code -eq 0 ]; then
        log_success "🎉 All tests completed successfully!"
    else
        log_error "❌ Some tests failed!"
    fi
    
    exit $exit_code
}

# Trap to ensure cleanup on script exit
trap 'stop_test_dependencies' EXIT

# Run main function
main "$@"