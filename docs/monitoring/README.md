# LLMsVerifier Monitoring Setup Guide

## Overview

LLMsVerifier provides comprehensive monitoring and observability through Prometheus metrics, Grafana dashboards, and structured logging.

## Quick Start

### Local Development

```bash
# Start monitoring stack
docker-compose -f docker-compose.monitoring.yml up -d

# Access services
# Grafana: http://localhost:3000 (admin/admin)
# Prometheus: http://localhost:9090
```

### Production Deployment

```bash
# Deploy with Kubernetes
kubectl apply -f monitoring/

# Or with Docker Compose
docker-compose --profile monitoring up -d
```

## Health Check Endpoints

| Endpoint | Description |
|----------|-------------|
| `/health` | Application health status |
| `/ready` | Readiness probe for load balancers |
| `/metrics` | Prometheus-compatible metrics |

### Health Check Example

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "healthy",
  "checks": {
    "database": "ok",
    "redis": "ok",
    "providers": "ok"
  },
  "version": "2.0.0",
  "uptime": "24h5m30s"
}
```

## Prometheus Metrics

### Application Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `http_requests_total` | Counter | HTTP request count by method/path/status |
| `http_request_duration_seconds` | Histogram | Request latency |
| `verification_jobs_total` | Counter | Verification job count by status |
| `verification_duration_seconds` | Histogram | Verification job duration |
| `models_verified_total` | Counter | Total models verified |

### System Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `cpu_usage_percent` | Gauge | CPU utilization |
| `memory_usage_bytes` | Gauge | Memory usage |
| `goroutines_count` | Gauge | Active goroutines |
| `gc_duration_seconds` | Histogram | Garbage collection duration |

### Business Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `models_verified_total` | Counter | Total verified models |
| `models_failed_total` | Counter | Failed verification count |
| `models_average_score` | Gauge | Average verification score |
| `providers_active_count` | Gauge | Active provider count |

## Grafana Dashboards

Pre-built dashboards are available in `llm-verifier/monitoring/grafana/`:

1. **Overview Dashboard**: System overview with key metrics
2. **Performance Dashboard**: Detailed performance metrics
3. **Application Dashboard**: Application-specific metrics
4. **Verification Dashboard**: Model verification metrics

### Importing Dashboards

```bash
# Using Grafana API
curl -X POST \
  -H "Content-Type: application/json" \
  -d @monitoring/grafana/dashboards/overview.json \
  http://admin:admin@localhost:3000/api/dashboards/db
```

## Alerting

### Alert Rules

Prometheus alerting rules are defined in `llm-verifier/monitoring/alert_rules.yml`:

```yaml
groups:
- name: llm-verifier.rules
  rules:
  - alert: LLMVerifierDown
    expr: up{job="llm-verifier"} == 0
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "LLM Verifier is down"

  - alert: HighErrorRate
    expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
    for: 5m
    labels:
      severity: warning

  - alert: HighVerificationFailures
    expr: rate(models_failed_total[5m]) > 0.2
    for: 10m
    labels:
      severity: warning
```

### AlertManager Configuration

```yaml
receivers:
- name: 'slack'
  slack_configs:
  - channel: '#alerts'
    api_url: 'https://hooks.slack.com/services/...'

- name: 'email'
  email_configs:
  - to: 'ops@company.com'

route:
  receiver: 'slack'
  routes:
  - match:
      severity: critical
    receiver: 'email'
```

## Logging

### Configuration

```yaml
logging:
  level: "info"
  format: "json"
  output: "file"
  file: "/var/log/llm-verifier/app.log"
  max_size: "100MB"
  max_backups: 5
  rotate_daily: true
```

### Log Aggregation

Integrate with external logging services:

```yaml
logging:
  services:
    elasticsearch:
      enabled: true
      url: "http://elasticsearch:9200"
      index: "llm-verifier"

    loki:
      enabled: true
      url: "http://loki:3100"
```

### Structured Log Fields

| Field | Description |
|-------|-------------|
| `timestamp` | Event timestamp |
| `level` | Log level (info/warn/error) |
| `component` | Application component |
| `request_id` | Request correlation ID |
| `message` | Log message |

## Distributed Tracing

### Jaeger Integration

```yaml
tracing:
  enabled: true
  service_name: "llm-verifier"
  jaeger:
    endpoint: "http://jaeger:14268/api/traces"
    sample_rate: 0.1
```

Access Jaeger UI at `http://localhost:16686`.

## Docker Compose Configuration

```yaml
# docker-compose.monitoring.yml
version: '3.8'
services:
  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    volumes:
      - ./monitoring/grafana:/etc/grafana/provisioning

  alertmanager:
    image: prom/alertmanager:latest
    ports:
      - "9093:9093"
```

## Kubernetes Deployment

```yaml
# monitoring/prometheus-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
    scrape_configs:
      - job_name: 'llm-verifier'
        static_configs:
          - targets: ['llm-verifier:8080']
```

## Debug Commands

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

## Best Practices

1. **Set Up Alerts**: Configure alerts for critical metrics
2. **Monitor SLAs**: Track service level agreements
3. **Use Dashboards**: Visualize metrics effectively
4. **Retain Data**: Configure appropriate data retention
5. **Regular Reviews**: Review and update monitoring setup

## Related Documentation

- [Detailed Monitoring Documentation](../../llm-verifier/monitoring/README.md)
- [Deployment Guide](../scoring/guides/DEPLOYMENT.md)
- [API Documentation](../API_DOCUMENTATION_UPDATED.md)
