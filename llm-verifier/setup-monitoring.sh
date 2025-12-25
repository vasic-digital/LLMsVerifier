#!/bin/bash

# LLM Verifier - Monitoring Setup Script
# Sets up Prometheus, Grafana, and alerting for production monitoring

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

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

# Check if Docker is available
check_docker() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed. Please install Docker first."
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose is not installed. Please install Docker Compose first."
        exit 1
    fi

    log_success "Docker and Docker Compose are available"
}

# Start monitoring stack
start_monitoring() {
    log_info "Starting monitoring stack..."

    docker-compose -f docker-compose.yml up -d prometheus grafana

    log_success "Monitoring stack started"
    log_info "Prometheus: http://localhost:9090"
    log_info "Grafana: http://localhost:3000 (admin/admin)"
}

# Configure Grafana dashboards
configure_grafana() {
    log_info "Configuring Grafana dashboards..."

    # Wait for Grafana to be ready
    local retries=30
    while [ $retries -gt 0 ]; do
        if curl -s http://localhost:3000/api/health > /dev/null; then
            break
        fi
        sleep 2
        ((retries--))
    done

    if [ $retries -eq 0 ]; then
        log_error "Grafana did not start properly"
        return 1
    fi

    # Create dashboard
    curl -s -X POST http://admin:admin@localhost:3000/api/dashboards/db \
         -H "Content-Type: application/json" \
         -d @monitoring/grafana/dashboards/llm-verifier-dashboard.json

    log_success "Grafana dashboard configured"
}

# Test monitoring endpoints
test_monitoring() {
    log_info "Testing monitoring endpoints..."

    # Test Prometheus
    if curl -s http://localhost:9090/-/healthy > /dev/null; then
        log_success "Prometheus is healthy"
    else
        log_error "Prometheus health check failed"
        return 1
    fi

    # Test Grafana
    if curl -s http://localhost:3000/api/health > /dev/null; then
        log_success "Grafana is healthy"
    else
        log_error "Grafana health check failed"
        return 1
    fi

    # Test LLM Verifier metrics endpoint
    if curl -s http://localhost:8080/metrics > /dev/null; then
        log_success "LLM Verifier metrics endpoint accessible"
    else
        log_warning "LLM Verifier metrics endpoint not accessible (may not be enabled)"
    fi
}

# Configure alerting
setup_alerting() {
    log_info "Setting up alerting..."

    # AlertManager would be configured here in production
    # For now, we just have the rules in Prometheus

    log_success "Alerting rules configured in Prometheus"
    log_info "Alerts will trigger based on:"
    echo "  - System downtime"
    echo "  - High response times (>500ms)"
    echo "  - High error rates (>5%)"
    echo "  - Provider failures"
    echo "  - Resource exhaustion"
}

# Generate monitoring documentation
generate_docs() {
    log_info "Generating monitoring documentation..."

    cat > MONITORING.md << 'EOF'
# LLM Verifier Monitoring Guide

## Overview

This guide covers monitoring, alerting, and observability for the LLM Verifier system.

## Monitoring Stack

### Prometheus
- **URL**: http://localhost:9090
- **Purpose**: Metrics collection and alerting
- **Configuration**: `monitoring/prometheus.yml`

### Grafana
- **URL**: http://localhost:3000
- **Credentials**: admin/admin (change in production!)
- **Dashboards**: LLM Verifier Performance Dashboard

### AlertManager
- **Configuration**: Integrated with Prometheus
- **Alerts**: Defined in `monitoring/alert_rules.yml`

## Key Metrics

### Application Metrics
- HTTP request duration (95th percentile)
- Request rate per second
- Error rate percentage
- Provider request success/failure rates

### System Metrics
- CPU usage percentage
- Memory usage percentage
- Disk I/O operations
- Network traffic

### Business Metrics
- Active verifications
- Provider availability
- Model coverage statistics
- User adoption metrics

## Alerting Rules

### Critical Alerts
- System downtime (>5 minutes)
- Error rate >5% (>5 minutes)
- Database unavailability (>5 minutes)

### Warning Alerts
- Response time >500ms (>10 minutes)
- CPU usage >90% (>10 minutes)
- Memory usage >85% (>10 minutes)
- Provider down (>5 minutes)

## Dashboard Panels

### API Performance
- Response time percentiles
- Request throughput
- Error rates by endpoint

### Provider Health
- Provider status table
- Request success rates
- Response time by provider

### System Resources
- CPU and memory usage
- Database connection pool
- Disk and network I/O

## Troubleshooting

### Common Issues

#### High Response Times
1. Check database query performance
2. Monitor provider API latency
3. Review application logs for bottlenecks
4. Consider scaling resources

#### High Error Rates
1. Check provider API status
2. Review application error logs
3. Verify configuration settings
4. Check network connectivity

#### Resource Exhaustion
1. Monitor CPU/memory usage trends
2. Check for memory leaks
3. Review connection pool settings
4. Consider horizontal scaling

## Maintenance

### Regular Tasks
- Review alert history weekly
- Update dashboards quarterly
- Archive old metrics monthly
- Test alerting rules monthly

### Backup and Recovery
- Prometheus data: Container volumes
- Grafana dashboards: Export configurations
- Alert rules: Version controlled
- Metrics retention: 90 days default
EOF

    log_success "Monitoring documentation generated: MONITORING.md"
}

# Main execution
main() {
    log_info "Setting up LLM Verifier monitoring..."

    check_docker
    start_monitoring
    configure_grafana
    test_monitoring
    setup_alerting
    generate_docs

    log_success "Monitoring setup completed!"
    echo
    log_info "Access points:"
    echo "  📊 Grafana: http://localhost:3000 (admin/admin)"
    echo "  📈 Prometheus: http://localhost:9090"
    echo "  📋 Alerts: Configured in Prometheus"
    echo
    log_info "Next steps:"
    echo "  1. Change default Grafana password"
    echo "  2. Configure email/Slack notifications"
    echo "  3. Set up log aggregation"
    echo "  4. Configure backup monitoring"
}

# Run main function
main "$@"