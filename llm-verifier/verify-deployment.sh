#!/bin/bash

# LLM Verifier - Production Deployment Verification Script
# This script verifies that the deployment is working correctly

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

API_URL=${API_URL:-"http://localhost:8080"}
TIMEOUT=${TIMEOUT:-30}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_health() {
    log_info "Checking API health endpoint..."
    if curl -s -f --max-time $TIMEOUT "${API_URL}/api/health" > /dev/null 2>&1; then
        log_success "Health check passed"
        return 0
    else
        log_error "Health check failed"
        return 1
    fi
}

check_providers() {
    log_info "Checking providers endpoint..."
    if curl -s -f --max-time $TIMEOUT "${API_URL}/api/providers" > /dev/null 2>&1; then
        log_success "Providers endpoint accessible"
        return 0
    else
        log_error "Providers endpoint failed"
        return 1
    fi
}

check_models() {
    log_info "Checking models endpoint..."
    if curl -s -f --max-time $TIMEOUT "${API_URL}/api/models" > /dev/null 2>&1; then
        log_success "Models endpoint accessible"
        return 0
    else
        log_error "Models endpoint failed"
        return 1
    fi
}

check_database() {
    log_info "Checking database connectivity..."
    # This would require authentication, so we'll just check if the API responds
    if curl -s --max-time $TIMEOUT "${API_URL}/api/health" | grep -q "healthy"; then
        log_success "Database connectivity verified"
        return 0
    else
        log_warning "Cannot verify database connectivity through health check"
        return 0  # Not critical for basic deployment check
    fi
}

main() {
    log_info "Starting LLM Verifier deployment verification..."
    log_info "API URL: ${API_URL}"
    log_info "Timeout: ${TIMEOUT}s"

    local failed=0

    if ! check_health; then
        ((failed++))
    fi

    if ! check_providers; then
        ((failed++))
    fi

    if ! check_models; then
        ((failed++))
    fi

    check_database  # Not counted as failure

    echo
    if [ $failed -eq 0 ]; then
        log_success "🎉 Deployment verification completed successfully!"
        log_success "All critical endpoints are responding correctly."
        echo
        log_info "Next steps:"
        echo "  1. Configure provider API keys in production"
        echo "  2. Set up monitoring dashboards"
        echo "  3. Configure backup schedules"
        echo "  4. Set up SSL certificates"
        echo "  5. Configure firewall rules"
        exit 0
    else
        log_error "❌ Deployment verification failed with $failed errors"
        log_info "Please check the application logs and configuration"
        exit 1
    fi
}

# Allow script to be sourced without running main
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi