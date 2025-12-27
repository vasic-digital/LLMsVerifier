# 🚀 LLM Verifier Scoring System - Production Deployment Guide

Complete guide for deploying the scoring system in production environments.

## 📋 Table of Contents

1. [Pre-Deployment Checklist](#pre-deployment-checklist)
2. [Environment Setup](#environment-setup)
3. [Installation Methods](#installation-methods)
4. [Configuration](#configuration)
5. [Database Setup](#database-setup)
6. [Security Hardening](#security-hardening)
7. [Monitoring & Alerting](#monitoring--alerting)
8. [Load Balancing & Scaling](#load-balancing--scaling)
9. [Backup & Recovery](#backup--recovery)
10. [Maintenance Procedures](#maintenance-procedures)
11. [Troubleshooting](#troubleshooting)

## Pre-Deployment Checklist

### ✅ System Requirements

- [ ] Go 1.21+ installed on target system
- [ ] SQLite3 with SQL Cipher support
- [ ] Sufficient disk space (minimum 10GB)
- [ ] Network connectivity to models.dev API
- [ ] SSL/TLS certificates for HTTPS
- [ ] Database backup storage
- [ ] Monitoring infrastructure
- [ ] Log aggregation system

### ✅ Security Requirements

- [ ] Database encryption keys generated
- [ ] API authentication configured
- [ ] Network security policies defined
- [ ] SSL/TLS certificates obtained
- [ ] Rate limiting configured
- [ ] Audit logging enabled
- [ ] Vulnerability scanning completed

### ✅ Performance Requirements

- [ ] Load testing completed
- [ ] Performance benchmarks established
- [ ] Scaling requirements defined
- [ ] Resource limits configured
- [ ] Caching strategy implemented

## Environment Setup

### 1. Production Environment Variables

```bash
# Create production environment file
sudo tee /etc/llm-verifier/environment << 'EOF'
# Application Environment
export LLM_ENV=production
export LLM_LOG_LEVEL=INFO
export LLM_LOG_FORMAT=json

# Database Configuration
export LLM_DATABASE_PATH=/var/lib/llm-verifier/data.db
export LLM_ENCRYPTION_KEY_FILE=/etc/llm-verifier/encryption.key
export LLM_DATABASE_MAX_CONNECTIONS=25
export LLM_DATABASE_TIMEOUT=30s

# API Configuration
export LLM_API_PORT=8080
export LLM_API_HOST=0.0.0.0
export LLM_API_READ_TIMEOUT=30s
export LLM_API_WRITE_TIMEOUT=30s
export LLM_API_IDLE_TIMEOUT=120s

# Models.dev API Configuration
export LLM_MODELS_DEV_URL=https://models.dev
export LLM_MODELS_DEV_TIMEOUT=30s
export LLM_MODELS_DEV_MAX_RETRIES=3
export LLM_MODELS_DEV_RETRY_DELAY=1s

# Security Configuration
export LLM_API_RATE_LIMIT=1000
export LLM_API_RATE_LIMIT_WINDOW=60s
export LLM_CORS_ALLOWED_ORIGINS=https://yourdomain.com
export LLM_CSRF_PROTECTION=true

# Performance Configuration
export LLM_CACHE_ENABLED=true
export LLM_CACHE_TTL=3600s
export LLM_CACHE_MAX_SIZE=1000
export LLM_BATCH_SIZE=100
export LLM_WORKER_POOL_SIZE=10

# Monitoring Configuration
export LLM_METRICS_ENABLED=true
export LLM_METRICS_PORT=9090
export LLM_HEALTH_CHECK_INTERVAL=30s
export LLM_ALERTING_ENABLED=true
EOF

# Load environment variables
source /etc/llm-verifier/environment
```

### 2. System User Creation

```bash
# Create system user for the application
sudo useradd -r -s /bin/false -d /var/lib/llm-verifier llm-verifier

# Create necessary directories
sudo mkdir -p /var/lib/llm-verifier/{data,logs,backups}
sudo mkdir -p /etc/llm-verifier
sudo mkdir -p /var/log/llm-verifier

# Set permissions
sudo chown -R llm-verifier:llm-verifier /var/lib/llm-verifier
sudo chown -R llm-verifier:llm-verifier /var/log/llm-verifier
sudo chmod 750 /var/lib/llm-verifier
sudo chmod 750 /var/log/llm-verifier
```

## Installation Methods

### 1. Binary Installation

```bash
# Download latest release
wget https://github.com/your-org/llm-verifier/releases/latest/download/llm-verifier-linux-amd64.tar.gz

# Extract binary
sudo tar -xzf llm-verifier-linux-amd64.tar.gz -C /usr/local/bin/
sudo chmod +x /usr/local/bin/llm-verifier

# Verify installation
llm-verifier --version
```

### 2. Docker Installation

```dockerfile
# Dockerfile
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o llm-verifier ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates sqlite-libs

WORKDIR /root/

# Create non-root user
RUN addgroup -g 1000 -S llmverifier && \
    adduser -u 1000 -S llmverifier -G llmverifier

COPY --from=builder /app/llm-verifier .
COPY --from=builder /app/config ./config

USER llmverifier

EXPOSE 8080 9090

ENTRYPOINT ["./llm-verifier"]
CMD ["server", "--config=/config/production.yaml"]
```

```yaml
# docker-compose.yml
version: '3.8'
services:
  llm-verifier:
    build: .
    ports:
      - "8080:8080"
      - "9090:9090"
    environment:
      - LLM_ENV=production
      - LLM_DATABASE_PATH=/data/llm-verifier.db
      - LLM_ENCRYPTION_KEY=${ENCRYPTION_KEY}
    volumes:
      - ./data:/data
      - ./logs:/logs
      - ./config:/config
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

### 3. Kubernetes Installation

```bash
# Create namespace
kubectl create namespace llm-verifier

# Create secrets
kubectl create secret generic llm-secrets \
  --from-literal=encryption-key=$(openssl rand -base64 32) \
  --namespace llm-verifier

# Apply configurations
kubectl apply -f k8s-configs/ -n llm-verifier
```

## Configuration

### 1. Production Configuration File

```yaml
# /etc/llm-verifier/production.yaml
server:
  host: 0.0.0.0
  port: 8080
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s
  max_request_size: 10MB

database:
  path: /var/lib/llm-verifier/data.db
  encryption_key_file: /etc/llm-verifier/encryption.key
  max_connections: 25
  connection_timeout: 30s
  busy_timeout: 5s
  
scoring:
  enabled: true
  default_config: production
  configs:
    production:
      weights:
        response_speed: 0.25
        model_efficiency: 0.20
        cost_effectiveness: 0.25
        capability: 0.20
        recency: 0.10
      thresholds:
        min_score: 0.0
        max_score: 10.0
      enabled: true
      cache_ttl: 3600s
      
    high-performance:
      weights:
        response_speed: 0.4
        model_efficiency: 0.3
        cost_effectiveness: 0.1
        capability: 0.15
        recency: 0.05
      enabled: true
      cache_ttl: 1800s

models_dev:
  base_url: https://models.dev
  timeout: 30s
  max_retries: 3
  retry_delay: 1s
  use_http3: true
  use_brotli: true
  
cache:
  enabled: true
  type: memory
  ttl: 3600s
  max_size: 1000
  
performance:
  worker_pool_size: 10
  batch_size: 100
  max_concurrent_requests: 1000
  
security:
  rate_limit: 1000
  rate_limit_window: 60s
  cors_allowed_origins:
    - https://yourdomain.com
  csrf_protection: true
  
monitoring:
  enabled: true
  metrics_port: 9090
  health_check_path: /health
  metrics_path: /metrics
```

### 2. Environment-specific Configurations

```bash
# Development environment
cp production.yaml development.yaml
# Edit development.yaml for dev settings

# Staging environment  
cp production.yaml staging.yaml
# Edit staging.yaml for staging settings
```

## Database Setup

### 1. Database Initialization

```bash
# Initialize database with scoring schema
sudo -u llm-verifier llm-verifier migrate up --config=/etc/llm-verifier/production.yaml

# Verify database setup
sudo -u llm-verifier llm-verifier database verify --config=/etc/llm-verifier/production.yaml

# Create initial backup
sudo -u llm-verifier llm-verifier backup create --path=/var/lib/llm-verifier/backups/initial-$(date +%Y%m%d).db
```

### 2. Database Schema

```sql
-- Scoring-specific tables
CREATE TABLE IF NOT EXISTS model_scores (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    model_id INTEGER NOT NULL,
    overall_score REAL NOT NULL,
    speed_score REAL NOT NULL,
    efficiency_score REAL NOT NULL,
    cost_score REAL NOT NULL,
    capability_score REAL NOT NULL,
    recency_score REAL NOT NULL,
    score_suffix TEXT NOT NULL,
    calculation_hash TEXT NOT NULL,
    calculation_details TEXT,
    last_calculated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    valid_until TIMESTAMP,
    is_active BOOLEAN DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE
);

-- Performance metrics table
CREATE TABLE IF NOT EXISTS model_performance_metrics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    model_id INTEGER NOT NULL,
    metric_type TEXT NOT NULL,
    metric_value REAL NOT NULL,
    metric_unit TEXT,
    sample_count INTEGER DEFAULT 1,
    p50_value REAL,
    p95_value REAL,
    p99_value REAL,
    min_value REAL,
    max_value REAL,
    std_dev REAL,
    measured_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    measurement_window_seconds INTEGER DEFAULT 3600,
    metadata TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (model_id) REFERENCES models(id) ON DELETE CASCADE
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_model_scores_model ON model_scores(model_id);
CREATE INDEX IF NOT EXISTS idx_model_scores_overall ON model_scores(overall_score);
CREATE INDEX IF NOT EXISTS idx_model_scores_active ON model_scores(is_active);
CREATE INDEX IF NOT EXISTS idx_model_scores_calculated ON model_scores(last_calculated);
CREATE INDEX IF NOT EXISTS idx_model_performance_metrics_model ON model_performance_metrics(model_id);
CREATE INDEX IF NOT EXISTS idx_model_performance_metrics_type ON model_performance_metrics(metric_type);
CREATE INDEX IF NOT EXISTS idx_model_performance_metrics_measured ON model_performance_metrics(measured_at);
```

## Security Hardening

### 1. SSL/TLS Configuration

```nginx
# /etc/nginx/sites-available/llm-verifier
server {
    listen 443 ssl http2;
    server_name api.llm-verifier.com;
    
    ssl_certificate /etc/ssl/certs/llm-verifier.crt;
    ssl_certificate_key /etc/ssl/private/llm-verifier.key;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES128-GCM-SHA256:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-SHA256:ECDHE-RSA-AES256-SHA384;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    
    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Rate limiting
        limit_req zone=api burst=20 nodelay;
        limit_req_status 429;
        
        # CORS headers
        add_header Access-Control-Allow-Origin "https://yourdomain.com" always;
        add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS" always;
        add_header Access-Control-Allow-Headers "Content-Type, Authorization" always;
    }
    
    location /health {
        access_log off;
        proxy_pass http://localhost:8080/health;
    }
}

# Redirect HTTP to HTTPS
server {
    listen 80;
    server_name api.llm-verifier.com;
    return 301 https://$server_name$request_uri;
}
```

### 2. Firewall Configuration

```bash
# Configure UFW firewall
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP redirect
sudo ufw allow 443/tcp   # HTTPS
sudo ufw allow 9090/tcp  # Metrics (internal only)
sudo ufw --force enable

# Configure iptables for additional security
sudo iptables -A INPUT -i lo -j ACCEPT
sudo iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 22 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 80 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 443 -j ACCEPT
sudo iptables -A INPUT -j DROP
```

### 3. API Security

```go
// Security middleware
func securityMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Rate limiting
        if !rateLimiter.Allow(getClientIP(r)) {
            http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
            return
        }
        
        // Input validation
        if err := validateRequest(r); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        
        // CORS handling
        w.Header().Set("Access-Control-Allow-Origin", "https://yourdomain.com")
        w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
        w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
        
        if r.Method == "OPTIONS" {
            w.WriteHeader(http.StatusOK)
            return
        }
        
        // Security headers
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        w.Header().Set("X-XSS-Protection", "1; mode=block")
        w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
        
        next.ServeHTTP(w, r)
    })
}
```

## Monitoring & Alerting

### 1. Prometheus Metrics

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'llm-verifier'
    static_configs:
      - targets: ['localhost:9090']
    metrics_path: /metrics
    scrape_interval: 15s
    
  - job_name: 'llm-verifier-health'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: /health
    scrape_interval: 30s
```

### 2. Grafana Dashboards

```json
{
  "dashboard": {
    "title": "LLM Verifier Scoring System",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])",
            "legendFormat": "{{method}} {{status}}"
          }
        ]
      },
      {
        "title": "Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "95th percentile"
          },
          {
            "expr": "histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "99th percentile"
          }
        ]
      },
      {
        "title": "Score Distribution",
        "type": "heatmap",
        "targets": [
          {
            "expr": "histogram_quantile(0.5, rate(scoring_overall_score_bucket[5m]))",
            "legendFormat": "50th percentile"
          }
        ]
      }
    ]
  }
}
```

### 3. Alerting Rules

```yaml
# alerting-rules.yml
groups:
- name: llm-verifier-alerts
  rules:
  - alert: HighErrorRate
    expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High error rate detected"
      description: "Error rate is {{ $value | humanizePercentage }} for the last 5 minutes"
      
  - alert: HighResponseTime
    expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High response time detected"
      description: "95th percentile response time is {{ $value }}s"
      
  - alert: DatabaseConnectionFailure
    expr: up{job="llm-verifier-health"} == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "Database connection failure"
      description: "LLM Verifier health check is failing"
```

## Load Balancing & Scaling

### 1. Nginx Load Balancer Configuration

```nginx
upstream llm_verifier {
    least_conn;
    server 127.0.0.1:8081 max_fails=3 fail_timeout=30s;
    server 127.0.0.1:8082 max_fails=3 fail_timeout=30s;
    server 127.0.0.1:8083 max_fails=3 fail_timeout=30s;
    
    keepalive 32;
}

server {
    listen 80;
    server_name api.llm-verifier.com;
    
    location / {
        proxy_pass http://llm_verifier;
        proxy_http_version 1.1;
        proxy_set_header Connection "";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Health check
        proxy_next_upstream error timeout invalid_header http_500 http_502 http_503 http_504;
        proxy_next_upstream_tries 3;
        proxy_next_upstream_timeout 10s;
    }
}
```

### 2. Auto-scaling Configuration

```yaml
# docker-compose.scale.yml
version: '3.8'
services:
  llm-verifier:
    build: .
    environment:
      - LLM_INSTANCE_ID={{.Task.Slot}}
      - LLM_REPLICA_ID={{.Service.Name}}-{{.Task.Slot}}
    deploy:
      replicas: 3
      update_config:
        parallelism: 1
        delay: 10s
        failure_action: rollback
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
      resources:
        limits:
          cpus: '1.0'
          memory: 1G
        reservations:
          cpus: '0.5'
          memory: 512M
          
  haproxy:
    image: haproxy:alpine
    ports:
      - "80:80"
      - "8404:8404"
    volumes:
      - ./haproxy.cfg:/usr/local/etc/haproxy/haproxy.cfg
    deploy:
      replicas: 2
      placement:
        constraints:
          - node.role == manager
```

## Backup & Recovery

### 1. Automated Backup Script

```bash
#!/bin/bash
# /usr/local/bin/llm-verifier-backup.sh

set -e

BACKUP_DIR="/var/lib/llm-verifier/backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=30

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Stop writes (if using WAL mode)
systemctl reload llm-verifier

# Create backup
sudo -u llm-verifier sqlite3 /var/lib/llm-verifier/data.db ".backup '$BACKUP_DIR/backup_$TIMESTAMP.db'"

# Compress backup
gzip "$BACKUP_DIR/backup_$TIMESTAMP.db"

# Upload to S3 (optional)
if [ -n "$S3_BUCKET" ]; then
    aws s3 cp "$BACKUP_DIR/backup_$TIMESTAMP.db.gz" "s3://$S3_BUCKET/backups/"
fi

# Clean old backups
find "$BACKUP_DIR" -name "backup_*.db.gz" -mtime +$RETENTION_DAYS -delete

# Log completion
echo "Backup completed: backup_$TIMESTAMP.db.gz"
```

### 2. Recovery Procedures

```bash
#!/bin/bash
# /usr/local/bin/llm-verifier-restore.sh

set -e

BACKUP_FILE="$1"
if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file>"
    exit 1
fi

echo "🔄 Starting recovery process..."

# Stop service
sudo systemctl stop llm-verifier

# Backup current data
cp /var/lib/llm-verifier/data.db /var/lib/llm-verifier/data.db.backup.$(date +%Y%m%d_%H%M%S)

# Restore from backup
sudo -u llm-verifier gunzip -c "$BACKUP_FILE" | sqlite3 /var/lib/llm-verifier/data.db ".restore 'main'"

# Verify database integrity
if sudo -u llm-verifier sqlite3 /var/lib/llm-verifier/data.db "PRAGMA integrity_check;" | grep -q "ok"; then
    echo "✅ Database integrity verified"
else
    echo "❌ Database integrity check failed"
    exit 1
fi

# Start service
sudo systemctl start llm-verifier

# Verify service health
if systemctl is-active --quiet llm-verifier; then
    echo "✅ Service restored successfully"
else
    echo "❌ Service failed to start"
    exit 1
fi

echo "✅ Recovery completed successfully"
```

## Maintenance Procedures

### 1. Regular Maintenance Schedule

```bash
# /etc/cron.d/llm-verifier-maintenance
# Daily maintenance at 2 AM
0 2 * * * llm-verifier /usr/local/bin/llm-verifier-daily-maintenance.sh

# Weekly maintenance on Sunday at 3 AM
0 3 * * 0 llm-verifier /usr/local/bin/llm-verifier-weekly-maintenance.sh

# Monthly maintenance on 1st at 4 AM
0 4 1 * * llm-verifier /usr/local/bin/llm-verifier-monthly-maintenance.sh
```

### 2. Daily Maintenance Script

```bash
#!/bin/bash
# /usr/local/bin/llm-verifier-daily-maintenance.sh

echo "Starting daily maintenance..."

# Database optimization
sudo -u llm-verifier sqlite3 /var/lib/llm-verifier/data.db "PRAGMA integrity_check;"
sudo -u llm-verifier sqlite3 /var/lib/llm-verifier/data.db "PRAGMA optimize;"

# Log rotation
find /var/log/llm-verifier -name "*.log" -mtime +7 -exec gzip {} \;
find /var/log/llm-verifier -name "*.log.gz" -mtime +30 -delete

# Check disk space
DISK_USAGE=$(df /var/lib/llm-verifier | tail -1 | awk '{print $5}' | sed 's/%//')
if [ "$DISK_USAGE" -gt 80 ]; then
    echo "⚠️ Disk usage is ${DISK_USAGE}% - consider cleanup"
fi

# Check service status
if ! systemctl is-active --quiet llm-verifier; then
    echo "❌ Service is not running"
    systemctl status llm-verifier
fi

echo "✅ Daily maintenance completed"
```

### 3. Database Maintenance

```bash
#!/bin/bash
# /usr/local/bin/llm-verifier-database-maintenance.sh

echo "Starting database maintenance..."

# Weekly optimization
sudo -u llm-verifier sqlite3 /var/lib/llm-verifier/data.db <<EOF
PRAGMA integrity_check;
PRAGMA optimize;
VACUUM;
ANALYZE;
EOF

# Update statistics
sudo -u llm-verifier sqlite3 /var/lib/llm-verifier/data.db "UPDATE sqlite_sequence SET seq = (SELECT MAX(id) FROM model_scores) WHERE name = 'model_scores';"

# Check for long-running queries
LONG_QUERIES=$(sudo -u llm-verifier sqlite3 /var/lib/llm-verifier/data.db "SELECT count(*) FROM sqlite_master WHERE type='table' AND name LIKE 'sqlite_temp%';")
if [ "$LONG_QUERIES" -gt 0 ]; then
    echo "⚠️ Found temporary tables - consider cleanup"
fi

# Update indexes statistics
sudo -u llm-verifier sqlite3 /var/lib/llm-verifier/data.db "PRAGMA index_info('idx_model_scores_model');"
sudo -u llm-verifier sqlite3 /var/lib/llm-verifier/data.db "PRAGMA index_info('idx_model_scores_overall');"

echo "✅ Database maintenance completed"
```

## Troubleshooting

### 1. Common Issues and Solutions

#### High Memory Usage

```bash
# Check memory usage
ps aux | grep llm-verifier
top -p $(pgrep llm-verifier)

# Check for memory leaks
go tool pprof http://localhost:6060/debug/pprof/heap

# Force garbage collection
curl -X POST http://localhost:6060/debug/gc
```

#### Database Connection Issues

```bash
# Check database connectivity
sudo -u llm-verifier sqlite3 /var/lib/llm-verifier/data.db "PRAGMA integrity_check;"

# Check connection pool
sudo -u llm-verifier sqlite3 /var/lib/llm-verifier/data.db "PRAGMA compile_options;"

# Monitor active connections
sudo lsof -u llm-verifier | grep sqlite
```

#### API Performance Issues

```bash
# Check API response times
curl -w "@curl-format.txt" -o /dev/null -s http://localhost:8080/health

# Monitor request rates
curl http://localhost:9090/metrics | grep http_requests_total

# Check for slow queries
sudo -u llm-verifier sqlite3 /var/lib/llm-verifier/data.db "SELECT * FROM sqlite_stat1;"
```

### 2. Diagnostic Tools

```bash
#!/bin/bash
# /usr/local/bin/llm-verifier-diagnostics.sh

echo "🔍 Running diagnostics..."

# System resources
echo "=== System Resources ==="
free -h
df -h
uptime

# Service status
echo "\n=== Service Status ==="
systemctl status llm-verifier
journalctl -u llm-verifier --since="1 hour ago" --no-pager

# Database health
echo "\n=== Database Health ==="
sudo -u llm-verifier sqlite3 /var/lib/llm-verifier/data.db "PRAGMA integrity_check;"
sudo -u llm-verifier sqlite3 /var/lib/llm-verifier/data.db "SELECT COUNT(*) FROM model_scores;"

# Network connectivity
echo "\n=== Network Connectivity ==="
curl -I https://models.dev
netstat -tlnp | grep :8080

# Log analysis
echo "\n=== Recent Errors ==="
grep -i "error\|fail\|exception" /var/log/llm-verifier/app.log | tail -20

echo "✅ Diagnostics completed"
```

### 3. Recovery Procedures

#### Service Recovery

```bash
# Service won't start
sudo systemctl status llm-verifier
sudo journalctl -u llm-verifier --since="1 hour ago"

# Check configuration
sudo -u llm-verifier llm-verifier validate-config --config=/etc/llm-verifier/production.yaml

# Reinstall service
sudo systemctl disable llm-verifier
sudo rm /etc/systemd/system/llm-verifier.service
sudo systemctl daemon-reload
# Reinstall from package
sudo systemctl enable llm-verifier
sudo systemctl start llm-verifier
```

#### Database Recovery

```bash
# Database corruption
sudo systemctl stop llm-verifier
sudo -u llm-verifier sqlite3 /var/lib/llm-verifier/data.db "PRAGMA integrity_check;"

# If corruption detected
sudo -u llm-verifier sqlite3 /var/lib/llm-verifier/data.db ".recover" > /tmp/recovered.sql
sudo -u llm-verifier sqlite3 /var/lib/llm-verifier/data.db.recovered < /tmp/recovered.sql

# Restore from backup
sudo /usr/local/bin/llm-verifier-restore.sh /path/to/backup.db.gz
```

---

## 📚 Additional Resources

- [Advanced Configuration](./ADVANCED.md)
- [Performance Tuning](./PERFORMANCE.md)
- [Security Guide](./SECURITY.md)
- [Monitoring Setup](./MONITORING.md)
- [Backup Procedures](./BACKUP.md)

---

**🎉 Your LLM Verifier Scoring System is now production-ready!**

*For ongoing support and updates, monitor the system health and refer to the maintenance procedures regularly.*