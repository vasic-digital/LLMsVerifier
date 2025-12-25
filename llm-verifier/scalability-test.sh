#!/bin/bash

# LLM Verifier - Scalability Testing Script
# Tests system performance under various load conditions

set -e

# Configuration
API_URL=${API_URL:-"http://localhost:8080"}
MAX_CONCURRENT=${MAX_CONCURRENT:-50}
TEST_DURATION=${TEST_DURATION:-30}
STEP_SIZE=${STEP_SIZE:-5}

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $1"
}

# Check system health
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if hey is installed
    if ! command -v hey &> /dev/null; then
        log_error "hey (HTTP load testing tool) is not installed."
        log_info "Install with: go install github.com/rakyll/hey@latest"
        exit 1
    fi

    # Check if system is healthy
    if ! curl -s -f --max-time 10 "${API_URL}/api/health" > /dev/null; then
        log_error "System is not healthy. Cannot run scalability tests."
        exit 1
    fi

    log_success "Prerequisites check passed"
}

# Run scalability test for specific concurrent users
run_scalability_test() {
    local concurrent=$1
    local duration=$2

    log_info "Testing with ${concurrent} concurrent users for ${duration} seconds..."

    # Run load test
    hey -n 10000 -c $concurrent -q 10 -z ${duration}s "${API_URL}/api/health" > "scalability_${concurrent}_users.txt" 2>&1

    # Extract metrics
    local avg_response=$(grep "Average" "scalability_${concurrent}_users.txt" | awk '{print $2}')
    local p95_response=$(grep "95%" "scalability_${concurrent}_users.txt" | awk '{print $2}')
    local requests_per_sec=$(grep "Requests/sec" "scalability_${concurrent}_users.txt" | awk '{print $2}')
    local errors=$(grep "Non-2xx" "scalability_${concurrent}_users.txt" | awk '{print $2}' | tr -d '(')

    # Log results
    echo "${concurrent},${avg_response},${p95_response},${requests_per_sec},${errors}" >> scalability_results.csv

    log_success "Results for ${concurrent} users:"
    echo "  Average response: ${avg_response}ms"
    echo "  95th percentile: ${p95_response}ms"
    echo "  Requests/sec: ${requests_per_sec}"
    echo "  Errors: ${errors}"

    # Check performance thresholds
    if (( $(echo "$p95_response > 1000" | bc -l) )); then
        log_warning "95th percentile response time > 1000ms at ${concurrent} users"
    fi

    if [ "$errors" -gt 0 ]; then
        log_warning "${errors} errors detected at ${concurrent} users"
    fi
}

# Generate scalability report
generate_report() {
    log_info "Generating scalability report..."

    cat > scalability_report.md << EOF
# LLM Verifier Scalability Test Report

Generated: $(date)
API Endpoint: ${API_URL}
Test Duration per Load Level: ${TEST_DURATION} seconds

## Test Results

| Concurrent Users | Avg Response (ms) | 95th Percentile (ms) | Requests/sec | Errors |
|------------------|-------------------|----------------------|--------------|--------|
$(tail -n +2 scalability_results.csv | while IFS=',' read -r users avg p95 rps errors; do
    echo "| $users | $avg | $p95 | $rps | $errors |"
done)

## Analysis

### Performance Scaling
- **Linear Scaling**: Check if response times remain stable as load increases
- **Breaking Point**: Identify when performance degrades significantly
- **Error Threshold**: Monitor when errors start occurring

### Recommendations

#### Current System Capacity
Based on the test results, the system can handle approximately X concurrent users
with acceptable performance (95th percentile < 500ms, error rate < 1%).

#### Scaling Recommendations
1. **Vertical Scaling**: Increase CPU/memory if response times degrade
2. **Horizontal Scaling**: Add more instances behind a load balancer
3. **Database Optimization**: Implement connection pooling and query optimization
4. **Caching**: Add Redis for frequently accessed data

#### Monitoring Thresholds
- Alert when 95th percentile > 500ms for > 10 minutes
- Alert when error rate > 1% for > 5 minutes
- Alert when concurrent users > 80% of capacity

## Next Steps

1. Implement recommended optimizations
2. Re-run scalability tests after changes
3. Set up continuous performance monitoring
4. Plan for production scaling requirements

EOF

    log_success "Scalability report generated: scalability_report.md"
}

# Main execution
main() {
    log_info "Starting LLM Verifier scalability testing..."
    log_info "API URL: ${API_URL}"
    log_info "Max Concurrent Users: ${MAX_CONCURRENT}"
    log_info "Test Duration per Level: ${TEST_DURATION} seconds"

    check_prerequisites

    # Initialize results file
    echo "Concurrent_Users,Avg_Response_ms,P95_Response_ms,Requests_per_sec,Errors" > scalability_results.csv

    # Run scalability tests with increasing load
    for ((users=STEP_SIZE; users<=MAX_CONCURRENT; users+=STEP_SIZE)); do
        run_scalability_test $users $TEST_DURATION

        # Brief pause between tests
        sleep 2
    done

    generate_report

    log_success "Scalability testing completed!"
    log_info "Review scalability_report.md for detailed analysis"
    log_info "Raw results available in scalability_results.csv"
}

# Run main function
main "$@"