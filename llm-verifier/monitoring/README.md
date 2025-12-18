# Monitoring and Observability Setup

This document covers the monitoring and observability setup for LLM Verifier.

## Components

### 1. Health Checks

The application provides comprehensive health check endpoints:

- **Health Endpoint**: `/health`
  - Application status
  - Database connectivity
  - Dependencies status
  - System metrics
  
- **Readiness Endpoint**: `/ready`
  - Checks if application is ready to serve traffic
  
- **Metrics Endpoint**: `/metrics`
  - Prometheus-compatible metrics
  - Performance counters
  - Error rates

### 2. Metrics Collected

#### Application Metrics
- `http_requests_total` - HTTP request count by method, path, status
- `http_request_duration_seconds` - Request latency histogram
- `database_connections_active` - Active database connections
- `database_connections_pool` - Connection pool size
- `verification_jobs_total` - Verification job count by status
- `verification_duration_seconds` - Verification job duration

#### System Metrics
- `cpu_usage_percent` - CPU utilization
- `memory_usage_bytes` - Memory usage
- `disk_usage_bytes` - Disk space usage
- `goroutines_count` - Go goroutine count
- `gc_duration_seconds` - Garbage collection duration

#### Business Metrics
- `models_verified_total` - Number of models verified
- `models_failed_total` - Number of failed verifications
- `models_average_score` - Average verification score
- `providers_active_count` - Active providers count

### 3. Alerting Rules

Prometheus alerting rules are defined for critical metrics:

```yaml
groups:
- name: llm-verifier.rules
  rules:
  # Application Health
  - alert: LLMVerifierDown
    expr: up{job="llm-verifier"} == 0
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "LLM Verifier is down"
      description: "LLM Verifier has been down for more than 5 minutes"
      
  # High Error Rate
  - alert: HighErrorRate
    expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High error rate detected"
      description: "Error rate is above 10% for more than 5 minutes"
      
  # High Memory Usage
  - alert: HighMemoryUsage
    expr: memory_usage_bytes / (1024*1024*1024) > 1000
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "High memory usage"
      description: "Memory usage is above 1GB for more than 10 minutes"
      
  # Database Issues
  - alert: DatabaseConnectionFailure
    expr: database_connections_active == 0
    for: 2m
    labels:
      severity: critical
    annotations:
      summary: "Database connection failure"
      description: "No active database connections for 2 minutes"
```

### 4. Grafana Dashboards

Pre-built Grafana dashboards are available:

- **Overview Dashboard**: System overview with key metrics
- **Performance Dashboard**: Detailed performance metrics
- **Application Dashboard**: Application-specific metrics
- **Infrastructure Dashboard**: System resource metrics

### 5. Log Management

#### Structured Logging
The application uses structured logging with the following fields:
- `timestamp`: Event timestamp
- `level`: Log level (info, warn, error)
- `component`: Application component
- `message`: Log message
- `request_id`: Request correlation ID
- `user_id`: User ID (if available)
- `error`: Error details (if applicable)

#### Log Aggregation
Configure log aggregation with:

```yaml
logging:
  level: "info"
  format: "json"
  output: "file"
  file: "/var/log/llm-verifier/app.log"
  max_size: "100MB"
  max_backups: 5
  rotate_daily: true
  
  # External logging services
  services:
    elasticsearch:
      enabled: false
      url: "http://elasticsearch:9200"
      index: "llm-verifier"
      
    loki:
      enabled: false
      url: "http://loki:3100"
      
    cloudwatch:
      enabled: false
      region: "us-east-1"
      log_group: "/llm-verifier"
```

### 6. Distributed Tracing

For distributed tracing, configure Jaeger:

```yaml
tracing:
  enabled: true
  service_name: "llm-verifier"
  jaeger:
    endpoint: "http://jaeger:14268/api/traces"
    sample_rate: 0.1
```

### 7. Setup Instructions

#### 1. Local Development

```bash
# Start monitoring stack
docker-compose -f docker-compose.monitoring.yml up -d

# Access services
# Grafana: http://localhost:3000 (admin/admin)
# Prometheus: http://localhost:9090
# Jaeger: http://localhost:16686
```

#### 2. Production Setup

```bash
# Deploy monitoring components
kubectl apply -f monitoring/

# Configure Prometheus
kubectl apply -f monitoring/prometheus/

# Configure Grafana
kubectl apply -f monitoring/grafana/

# Configure AlertManager
kubectl apply -f monitoring/alertmanager/
```

### 8. Monitoring Best Practices

1. **Set Up Alerts**: Configure alerts for critical metrics
2. **Monitor SLAs**: Track service level agreements
3. **Use Dashboards**: Visualize metrics effectively
4. **Log Everything**: Ensure comprehensive logging
5. **Retain Data**: Configure appropriate data retention
6. **Monitor Costs**: Track monitoring infrastructure costs
7. **Regular Reviews**: Review and update monitoring setup

### 9. Troubleshooting

#### Common Issues

- **Missing Metrics**: Check metrics endpoint configuration
- **High Memory**: Review memory usage patterns
- **Slow Queries**: Analyze database performance
- **Alert Fatigue**: Review alert thresholds
- **Data Gaps**: Check monitoring infrastructure health

#### Debug Commands

```bash
# Check health status
curl http://localhost:8080/health

# View metrics
curl http://localhost:8080/metrics

# Check logs
tail -f /var/log/llm-verifier/app.log

# Test alerts
amtool alert --alertmanager.url=http://localhost:9093
```

## Contact

For monitoring and observability support:
- Documentation: [../docs/](../docs/)
- Issues: [GitHub Issues](https://github.com/vasic-digital/LLMsVerifier/issues)
- Community: [Discussions](https://github.com/vasic-digital/LLMsVerifier/discussions)
