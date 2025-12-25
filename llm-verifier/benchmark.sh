#!/bin/bash

# LLM Verifier - Performance Benchmarking Script
# Runs comprehensive performance tests against the deployed system

set -e

# Configuration
API_URL=${API_URL:-"http://localhost:8080"}
CONCURRENT_USERS=${CONCURRENT_USERS:-10}
TEST_DURATION=${TEST_DURATION:-60}  # seconds
WARMUP_TIME=${WARMUP_TIME:-10}

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

# Check if system is healthy
check_health() {
    log_info "Checking system health..."
    if ! curl -s -f --max-time 10 "${API_URL}/api/health" > /dev/null; then
        log_error "System is not healthy. Aborting benchmark."
        exit 1
    fi
    log_success "System health check passed"
}

# Benchmark health endpoint
benchmark_health() {
    log_info "Benchmarking health endpoint..."

    # Warmup
    log_info "Warming up for ${WARMUP_TIME} seconds..."
    timeout ${WARMUP_TIME} curl -s "${API_URL}/api/health" > /dev/null &

    # Run benchmark
    log_info "Running health endpoint benchmark (${TEST_DURATION}s)..."
    hey -n 1000 -c ${CONCURRENT_USERS} -q 10 "${API_URL}/api/health" > health_benchmark.txt 2>&1

    # Parse results
    local avg_response=$(grep "Average" health_benchmark.txt | awk '{print $2}')
    local p95_response=$(grep "95%" health_benchmark.txt | awk '{print $2}')
    local requests_per_sec=$(grep "Requests/sec" health_benchmark.txt | awk '{print $2}')

    log_success "Health endpoint benchmark results:"
    echo "  Average response time: ${avg_response}ms"
    echo "  95th percentile: ${p95_response}ms"
    echo "  Requests/sec: ${requests_per_sec}"

    # Check thresholds
    if (( $(echo "$avg_response > 200" | bc -l) )); then
        log_warning "Average response time exceeds 200ms threshold"
    fi
}

# Benchmark providers endpoint
benchmark_providers() {
    log_info "Benchmarking providers endpoint..."

    hey -n 500 -c 5 -q 5 "${API_URL}/api/providers" > providers_benchmark.txt 2>&1

    local avg_response=$(grep "Average" providers_benchmark.txt | awk '{print $2}')
    local success_rate=$(grep "Status code distribution" providers_benchmark.txt -A 5 | grep "200" | awk '{print $2}' | tr -d '%')

    log_success "Providers endpoint benchmark results:"
    echo "  Average response time: ${avg_response}ms"
    echo "  Success rate: ${success_rate}%"

    if (( $(echo "$success_rate < 99.9" | bc -l) )); then
        log_warning "Success rate below 99.9% SLA"
    fi
}

# Benchmark models endpoint
benchmark_models() {
    log_info "Benchmarking models endpoint..."

    hey -n 300 -c 3 -q 3 "${API_URL}/api/models" > models_benchmark.txt 2>&1

    local avg_response=$(grep "Average" models_benchmark.txt | awk '{print $2}')
    local data_transfer=$(grep "Total data" models_benchmark.txt | awk '{print $3, $4}')

    log_success "Models endpoint benchmark results:"
    echo "  Average response time: ${avg_response}ms"
    echo "  Data transferred: ${data_transfer}"
}

# Run memory and CPU profiling
run_profiling() {
    log_info "Running system profiling..."

    # Memory profiling
    log_info "Capturing memory profile..."
    go tool pprof -png -output memory_profile.png http://localhost:8080/debug/pprof/heap > /dev/null 2>&1 || true

    # CPU profiling
    log_info "Capturing CPU profile..."
    go tool pprof -png -output cpu_profile.png http://localhost:8080/debug/pprof/profile?seconds=30 > /dev/null 2>&1 || true

    if [ -f memory_profile.png ] && [ -f cpu_profile.png ]; then
        log_success "Profiling completed - check memory_profile.png and cpu_profile.png"
    else
        log_warning "Profiling not available (debug endpoints not enabled)"
    fi
}

# Generate performance report
generate_report() {
    log_info "Generating performance report..."

    cat > performance_report.md << EOF
# LLM Verifier Performance Benchmark Report

Generated: $(date)
Test Duration: ${TEST_DURATION} seconds
Concurrent Users: ${CONCURRENT_USERS}

## Health Endpoint Performance
$(cat health_benchmark.txt 2>/dev/null || echo "No data available")

## Providers Endpoint Performance
$(cat providers_benchmark.txt 2>/dev/null || echo "No data available")

## Models Endpoint Performance
$(cat models_benchmark.txt 2>/dev/null || echo "No data available")

## Recommendations

### Performance Targets Met:
- [ ] Average response time < 200ms
- [ ] 95th percentile < 500ms
- [ ] Error rate < 1%
- [ ] Throughput > 100 req/sec

### Optimization Opportunities:
- Database query optimization
- Caching implementation
- Connection pooling tuning
- Horizontal scaling evaluation

EOF

    log_success "Performance report generated: performance_report.md"
}

# Main execution
main() {
    log_info "Starting LLM Verifier performance benchmarking..."
    log_info "API URL: ${API_URL}"
    log_info "Concurrent Users: ${CONCURRENT_USERS}"
    log_info "Test Duration: ${TEST_DURATION} seconds"

    check_health

    benchmark_health
    benchmark_providers
    benchmark_models

    run_profiling
    generate_report

    log_success "Performance benchmarking completed!"
    log_info "Review performance_report.md for detailed results"
}

# Check if hey (HTTP load testing tool) is available
if ! command -v hey &> /dev/null; then
    log_error "hey (HTTP load testing tool) is not installed."
    log_info "Install with: go install github.com/rakyll/hey@latest"
    exit 1
fi

# Run main function
main "$@"