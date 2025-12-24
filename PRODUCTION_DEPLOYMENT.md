# Production Deployment Guide

## Overview

This guide provides comprehensive instructions for deploying the LLM Verifier system to production environments. The LLM Verifier is an enterprise-grade platform for verifying and managing Large Language Model (LLM) APIs across multiple providers.

## Prerequisites

### System Requirements
- **OS**: Linux (Ubuntu 20.04+ or RHEL/CentOS 8+)
- **CPU**: 4+ cores (8+ recommended)
- **RAM**: 8GB minimum (16GB+ recommended)
- **Storage**: 50GB+ SSD storage
- **Network**: 1Gbps+ connection

### Software Dependencies
- Go 1.21+
- PostgreSQL 15+
- Redis 7+ (optional, for enhanced performance)
- Docker & Docker Compose (for containerized deployment)
- Nginx or Apache (reverse proxy)
- SSL certificate (Let's Encrypt recommended)

### Network Requirements
- **Inbound**: HTTPS (443), HTTP (80 for redirects)
- **Outbound**: Access to LLM provider APIs (OpenAI, Anthropic, Google, etc.)
- **Internal**: Database access, Redis access (if used)

## Deployment Options

### Option 1: Docker Compose (Recommended)

#### 1. Environment Setup

```bash
# Clone the repository
git clone https://github.com/your-org/llm-verifier.git
cd llm-verifier

# Create environment file
cp .env.example .env.production
```

#### 2. Configure Environment Variables

Edit `.env.production`:

```bash
# Database Configuration
DATABASE_URL=postgresql://llm_verifier:secure_password@localhost:5432/llm_verifier_prod?sslmode=require

# JWT Configuration
JWT_SECRET=your-256-bit-secret-key-here

# API Keys (encrypted in production)
OPENAI_API_KEY=encrypted_key_here
ANTHROPIC_API_KEY=encrypted_key_here

# Redis (optional)
REDIS_URL=redis://localhost:6379

# Application Settings
APP_ENV=production
LOG_LEVEL=info
API_PORT=8080

# Rate Limiting
RATE_LIMIT_REQUESTS_PER_MINUTE=1000
RATE_LIMIT_BURST_SIZE=100

# Monitoring
PROMETHEUS_ENABLED=true
GRAFANA_ENABLED=true
```

#### 3. Database Setup

```bash
# Start only PostgreSQL
docker-compose -f docker-compose.prod.yml up -d postgres

# Wait for database to be ready
sleep 30

# Run database migrations
docker-compose -f docker-compose.prod.yml run --rm app migrate up

# Create initial admin user
docker-compose -f docker-compose.prod.yml run --rm app create-admin-user
```

#### 4. Deploy Application

```bash
# Start all services
docker-compose -f docker-compose.prod.yml up -d

# Verify deployment
curl -k https://your-domain.com/health
```

### Option 2: Kubernetes Deployment

#### 1. Prerequisites
- Kubernetes cluster (1.24+)
- kubectl configured
- Helm 3+

#### 2. Deploy with Helm

```bash
# Add Helm repository
helm repo add llm-verifier https://charts.your-org.com
helm repo update

# Install LLM Verifier
helm install llm-verifier llm-verifier/llm-verifier \
  --namespace llm-verifier \
  --create-namespace \
  --set database.external.enabled=true \
  --set database.external.host=your-postgres-host \
  --set ingress.enabled=true \
  --set ingress.hosts[0]=your-domain.com
```

#### 3. Verify Deployment

```bash
# Check pod status
kubectl get pods -n llm-verifier

# Check services
kubectl get services -n llm-verifier

# Check ingress
kubectl get ingress -n llm-verifier
```

## Configuration

### Database Configuration

The system supports PostgreSQL with the following schema:

```sql
-- Main tables created by migrations
CREATE TABLE providers (...);
CREATE TABLE models (...);
CREATE TABLE verification_results (...);
CREATE TABLE users (...);
CREATE TABLE api_keys (...);
CREATE TABLE notifications (...);
```

### Security Configuration

#### JWT Tokens
- Use 256-bit secret keys
- Set appropriate token expiration (default: 24 hours)
- Rotate keys regularly

#### API Keys
- Store encrypted in database
- Use separate keys for different providers
- Implement key rotation policies

#### Rate Limiting
- Configure per-client limits
- Implement exponential backoff
- Monitor for abuse patterns

## Testing Procedures

### Pre-Deployment Testing

#### 1. Unit Tests
```bash
# Run all unit tests
go test ./... -v

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

#### 2. Integration Tests
```bash
# Run integration tests
go test -tags integration ./testing/... -v

# Run specific test suites
go test -tags integration -run TestSecuritySuite ./testing/... -v
go test -tags integration -run TestEndToEndWorkflowSuite ./testing/... -v
```

#### 3. Performance Benchmarks
```bash
# Run performance benchmarks
go test -tags integration -bench=. -benchmem ./testing/... -v

# Generate benchmark reports
go test -tags integration -bench=. -benchmem ./testing/... > benchmarks.txt
```

#### 4. Security Testing
```bash
# Run security tests
go test -tags integration -run TestSecuritySuite ./testing/... -v

# SQL injection tests
go test -tags integration -run "SQLInjection" ./testing/... -v

# Authentication tests
go test -tags integration -run "Authentication" ./testing/... -v
```

### Production Validation Tests

#### 1. Health Checks
```bash
# API health check
curl -f https://your-domain.com/health

# Database connectivity
curl -f https://your-domain.com/health/database

# External API connectivity
curl -f https://your-domain.com/health/providers
```

#### 2. Functional Tests
```bash
# Test provider verification
curl -X POST https://your-domain.com/api/v1/providers/verify \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Test model verification
curl -X POST https://your-domain.com/api/v1/models/verify \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"

# Test configuration export
curl -X GET https://your-domain.com/api/v1/config/export \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### 3. Load Testing
```bash
# Install hey for load testing
go install github.com/rakyll/hey@latest

# Test API endpoints under load
hey -n 1000 -c 10 https://your-domain.com/health

# Test verification endpoints
hey -n 100 -c 5 -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  https://your-domain.com/api/v1/providers
```

#### 4. Security Validation
```bash
# Test authentication
curl -X POST https://your-domain.com/auth/login \
  -d '{"username":"test","password":"test"}'

# Test unauthorized access
curl -f https://your-domain.com/api/v1/admin/users

# Test rate limiting
for i in {1..150}; do
  curl -s https://your-domain.com/api/v1/providers > /dev/null &
done
wait
```

## Monitoring Setup

### Application Monitoring

#### Prometheus Metrics
The application exposes metrics at `/metrics`:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'llm-verifier'
    static_configs:
      - targets: ['localhost:8080']
    scrape_interval: 15s
```

#### Key Metrics to Monitor
- `llm_verifier_requests_total`: Total API requests
- `llm_verifier_request_duration_seconds`: Request latency
- `llm_verifier_verification_success_total`: Successful verifications
- `llm_verifier_verification_errors_total`: Verification errors
- `llm_verifier_database_connections_active`: Active DB connections

### System Monitoring

#### Grafana Dashboards
Import the provided dashboard JSON:

```bash
# Access Grafana at http://your-domain.com/grafana
# Default credentials: admin/admin
```

#### Alert Rules
```yaml
# alert_rules.yml
groups:
  - name: llm-verifier
    rules:
      - alert: HighErrorRate
        expr: rate(llm_verifier_verification_errors_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High verification error rate"
```

### Logging

#### Log Aggregation
```yaml
# docker-compose.prod.yml logging configuration
version: '3.8'
services:
  app:
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"
```

#### Log Analysis
```bash
# View application logs
docker-compose -f docker-compose.prod.yml logs -f app

# Search for errors
docker-compose -f docker-compose.prod.yml logs app | grep ERROR

# Monitor verification activities
docker-compose -f docker-compose.prod.yml logs app | grep "verification"
```

## Backup and Recovery

### Database Backup

#### Automated Backups
```bash
# Create backup script
#!/bin/bash
BACKUP_DIR="/opt/llm-verifier/backups"
DATE=$(date +%Y%m%d_%H%M%S)

pg_dump -h localhost -U llm_verifier llm_verifier_prod > $BACKUP_DIR/backup_$DATE.sql

# Keep only last 7 days
find $BACKUP_DIR -name "backup_*.sql" -mtime +7 -delete
```

#### Schedule Backups
```bash
# Add to crontab
0 2 * * * /opt/llm-verifier/scripts/backup.sh
```

### Configuration Backup

```bash
# Backup configuration files
tar -czf /opt/backups/config_$(date +%Y%m%d).tar.gz \
  /opt/llm-verifier/.env.production \
  /opt/llm-verifier/config/
```

### Recovery Procedures

#### Database Recovery
```bash
# Stop the application
docker-compose -f docker-compose.prod.yml stop app

# Restore database
psql -h localhost -U llm_verifier llm_verifier_prod < backup_file.sql

# Restart application
docker-compose -f docker-compose.prod.yml start app
```

#### Application Rollback
```bash
# Rollback to previous version
docker-compose -f docker-compose.prod.yml pull app
docker-compose -f docker-compose.prod.yml up -d app
```

## Scaling Considerations

### Horizontal Scaling

#### Load Balancer Configuration
```nginx
# nginx.conf
upstream llm_verifier_backend {
    server app1:8080;
    server app2:8080;
    server app3:8080;
}

server {
    listen 443 ssl;
    server_name your-domain.com;

    location / {
        proxy_pass http://llm_verifier_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

#### Database Scaling
- Use read replicas for analytics queries
- Implement connection pooling
- Monitor query performance

### Vertical Scaling

#### Resource Allocation
```yaml
# docker-compose.prod.yml
services:
  app:
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 4G
        reservations:
          cpus: '1.0'
          memory: 2G
```

## Security Hardening

### Network Security

#### Firewall Configuration
```bash
# UFW rules
ufw allow ssh
ufw allow 80
ufw allow 443
ufw --force enable
```

#### SSL/TLS Configuration
```nginx
# nginx ssl configuration
server {
    listen 443 ssl http2;
    server_name your-domain.com;

    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;

    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;
    ssl_prefer_server_ciphers off;
}
```

### Application Security

#### Environment Variables
- Store secrets in environment variables
- Use Docker secrets or Kubernetes secrets
- Never commit secrets to version control

#### API Security
- Implement proper CORS policies
- Use HTTPS only
- Implement request size limits
- Add security headers

## Troubleshooting

### Common Issues

#### Database Connection Issues
```bash
# Check database connectivity
docker-compose -f docker-compose.prod.yml exec postgres pg_isready

# Check application logs
docker-compose -f docker-compose.prod.yml logs app | grep "database"
```

#### High Memory Usage
```bash
# Monitor memory usage
docker stats

# Check for memory leaks
go tool pprof http://localhost:8080/debug/pprof/heap
```

#### Slow Performance
```bash
# Check slow queries
docker-compose -f docker-compose.prod.yml exec postgres psql -U llm_verifier -d llm_verifier_prod -c "SELECT * FROM pg_stat_activity;"

# Profile application
go tool pprof http://localhost:8080/debug/pprof/profile
```

### Support Contacts

- **Technical Support**: support@your-company.com
- **Security Issues**: security@your-company.com
- **Documentation**: https://docs.your-company.com/llm-verifier

## Maintenance Procedures

### Regular Maintenance Tasks

#### Weekly Tasks
- Review application logs
- Check disk usage
- Update dependencies
- Run security scans

#### Monthly Tasks
- Database optimization
- Certificate renewal
- Performance benchmarking
- Security updates

#### Quarterly Tasks
- Full system backup validation
- Disaster recovery testing
- Architecture review
- Compliance audits

### Updates and Upgrades

#### Application Updates
```bash
# Update application
docker-compose -f docker-compose.prod.yml pull
docker-compose -f docker-compose.prod.yml up -d

# Run database migrations if needed
docker-compose -f docker-compose.prod.yml run --rm app migrate up
```

#### Dependency Updates
```bash
# Update Go dependencies
go mod tidy
go mod download

# Rebuild and redeploy
docker-compose -f docker-compose.prod.yml build --no-cache
docker-compose -f docker-compose.prod.yml up -d
```

---

## Testing Checklist

Before going live, ensure all items are checked:

### Pre-Deployment
- [ ] Unit tests pass (100% success rate)
- [ ] Integration tests pass
- [ ] Security tests pass
- [ ] Performance benchmarks meet requirements
- [ ] Code coverage > 80% for critical components

### Deployment
- [ ] Environment variables configured
- [ ] Database migrations applied
- [ ] SSL certificates installed
- [ ] DNS records updated
- [ ] Firewall configured

### Post-Deployment
- [ ] Health checks pass
- [ ] Basic functionality verified
- [ ] Load testing completed
- [ ] Monitoring configured
- [ ] Backup procedures tested
- [ ] Rollback procedures documented

### Production Validation
- [ ] End-to-end workflows tested
- [ ] External API integrations working
- [ ] User authentication working
- [ ] Rate limiting functional
- [ ] Error handling verified

---

*This deployment guide is comprehensive and should be reviewed by your DevOps and Security teams before production deployment.*