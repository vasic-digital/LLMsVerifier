# Deployment Guide

This guide covers deploying LLM Verifier in production environments.

## Prerequisites

- Go 1.21+ or Docker
- SQLite 3.x (if not using container)
- 4GB RAM minimum, 8GB recommended (for supervisor system)
- 20GB storage minimum, 100GB recommended (for backups and logs)
- Cloud provider credentials (optional, for backup functionality)

## Environment Setup

### 1. Secure Configuration

Set required environment variables:

```bash
# Required: Generate secure JWT secret
export LLM_VERIFIER_API_JWT_SECRET=$(openssl rand -base64 32)

# Optional: Database path
export LLM_VERIFIER_DATABASE_PATH="/app/data/llm-verifier.db"

# Optional: API configuration
export LLM_VERIFIER_API_PORT="8080"
export LLM_VERIFIER_API_RATE_LIMIT="1000"
export LLM_VERIFIER_PROFILE="prod"

# Optional: Cloud backup configuration (if using cloud backup)
export AWS_ACCESS_KEY_ID="your-aws-key"
export AWS_SECRET_ACCESS_KEY="your-aws-secret"
export AWS_REGION="us-east-1"
export GCP_SERVICE_ACCOUNT_JSON="path/to/service-account.json"
export AZURE_STORAGE_KEY="your-azure-key"

# Optional: Supervisor system configuration
export LLM_VERIFIER_SUPERVISOR_ENABLED="true"
export LLM_VERIFIER_SUPERVISOR_MAX_WORKERS="5"

# Optional: Context management configuration
export LLM_VERIFIER_CONTEXT_LONG_TERM_ENABLED="true"
export LLM_VERIFIER_CONTEXT_SUMMARIZATION_ENABLED="true"
```

### 2. Create Required Directories

```bash
# Create data directory
sudo mkdir -p /app/data
sudo chown $USER:$USER /app/data
chmod 755 /app/data

# Create logs directory
sudo mkdir -p /app/logs
sudo chown $USER:$USER /app/logs
chmod 755 /app/logs

# Create backups directory (if using local backups)
sudo mkdir -p /app/backups
sudo chown $USER:$USER /app/backups
chmod 755 /app/backups

# Create context storage directory (for LLM context management)
sudo mkdir -p /app/context
sudo chown $USER:$USER /app/context
chmod 755 /app/context
```

## Deployment Options

### Option 1: Docker Deployment (Recommended)

#### 1. Create Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o llm-verifier ./cmd

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/llm-verifier .
COPY --from=builder /app/config.yaml .

EXPOSE 8080
CMD ["./llm-verifier", "server"]
```

#### 2. Create docker-compose.yml

```yaml
version: '3.8'

services:
  llm-verifier:
    build: .
    ports:
      - "8080:8080"
    environment:
      - LLM_VERIFIER_API_JWT_SECRET=${JWT_SECRET}
      - LLM_VERIFIER_DATABASE_PATH=/app/data/llm-verifier.db
      - LLM_VERIFIER_PROFILE=prod
      - LLM_VERIFIER_API_PORT=8080
    volumes:
      - ./data:/app/data
      - ./logs:/app/logs
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

#### 3. Deploy

```bash
# Set environment variables
export JWT_SECRET=$(openssl rand -base64 32)

# Start services
docker-compose up -d

# Check status
docker-compose ps

# View logs
docker-compose logs -f llm-verifier
```

### Option 2: Kubernetes Deployment

#### 1. Create Namespace

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: llm-verifier
```

#### 2. Create Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: llm-verifier-secrets
  namespace: llm-verifier
type: Opaque
stringData:
  jwt-secret: "your-secure-random-jwt-secret-here"
```

#### 3. Create ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: llm-verifier-config
  namespace: llm-verifier
data:
  config.yaml: |
    profile: prod
    global:
      max_retries: 3
      timeout: 30s
    database:
      path: "/app/data/llm-verifier.db"
    api:
      port: "8080"
      rate_limit: 1000
      enable_cors: true
      cors_origins: "https://yourdomain.com"
    logging:
      level: info
      format: json
      output: file
      file_path: "/app/logs/llm-verifier.log"
```

#### 4. Create Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: llm-verifier
  namespace: llm-verifier
spec:
  replicas: 2
  selector:
    matchLabels:
      app: llm-verifier
  template:
    metadata:
      labels:
        app: llm-verifier
    spec:
      containers:
      - name: llm-verifier
        image: llm-verifier:latest
        ports:
        - containerPort: 8080
        env:
        - name: LLM_VERIFIER_API_JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: llm-verifier-secrets
              key: jwt-secret
        - name: LLM_VERIFIER_PROFILE
          value: "prod"
        volumeMounts:
        - name: config-volume
          mountPath: /app/config.yaml
          subPath: config.yaml
        - name: data-volume
          mountPath: /app/data
        - name: logs-volume
          mountPath: /app/logs
        resources:
          requests:
            memory: "512Mi"
            cpu: "250m"
          limits:
            memory: "1Gi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config-volume
        configMap:
          name: llm-verifier-config
      - name: data-volume
        persistentVolumeClaim:
          claimName: llm-verifier-data
      - name: logs-volume
        persistentVolumeClaim:
          claimName: llm-verifier-logs
```

#### 5. Create Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: llm-verifier-service
  namespace: llm-verifier
spec:
  selector:
    app: llm-verifier
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: ClusterIP
```

#### 6. Create Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: llm-verifier-ingress
  namespace: llm-verifier
  annotations:
    kubernetes.io/ingress.class: nginx
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  tls:
  - hosts:
    - llm-verifier.yourdomain.com
    secretName: llm-verifier-tls
  rules:
  - host: llm-verifier.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: llm-verifier-service
            port:
              number: 80
```

#### 7. Deploy to Kubernetes

```bash
# Apply all manifests
kubectl apply -f k8s/

# Check deployment status
kubectl get pods -n llm-verifier
kubectl get services -n llm-verifier
kubectl get ingress -n llm-verifier

# View logs
kubectl logs -f deployment/llm-verifier -n llm-verifier
```

### Option 3: Binary Deployment

#### 1. Build Binary

```bash
# Clone repository
git clone https://github.com/your-org/llm-verifier.git
cd llm-verifier

# Build binary
go build -ldflags="-s -w" -o llm-verifier ./cmd

# Verify binary
./llm-verifier --version
```

#### 2. Systemd Service

Create `/etc/systemd/system/llm-verifier.service`:

```ini
[Unit]
Description=LLM Verifier API Server
After=network.target

[Service]
Type=simple
User=llm-verifier
WorkingDirectory=/opt/llm-verifier
ExecStart=/opt/llm-verifier/llm-verifier server
Restart=always
RestartSec=5
Environment=LLM_VERIFIER_PROFILE=prod
Environment=LLM_VERIFIER_API_JWT_SECRET=your-secure-secret
Environment=LLM_VERIFIER_DATABASE_PATH=/opt/llm-verifier/data/llm-verifier.db

[Install]
WantedBy=multi-user.target
```

#### 3. Start Service

```bash
# Create user
sudo useradd --system --shell /bin/false llm-verifier

# Setup directories
sudo mkdir -p /opt/llm-verifier/{data,logs}
sudo chown -R llm-verifier:llm-verifier /opt/llm-verifier

# Install binary
sudo cp llm-verifier /opt/llm-verifier/
sudo chmod +x /opt/llm-verifier/llm-verifier

# Install service
sudo cp llm-verifier.service /etc/systemd/system/
sudo systemctl daemon-reload

# Start service
sudo systemctl enable llm-verifier
sudo systemctl start llm-verifier

# Check status
sudo systemctl status llm-verifier
```

## Monitoring and Maintenance

### Health Checks

```bash
# API Health
curl -f http://localhost:8080/health

# Service Status (systemd)
sudo systemctl status llm-verifier

# Container Status
docker-compose ps
kubectl get pods -n llm-verifier
```

### Log Management

```bash
# View application logs
tail -f /app/logs/llm-verifier.log

# Rotate logs (logrotate)
sudo logrotate /etc/logrotate.d/llm-verifier
```

### Backup Strategy

```bash
# Database backup
sqlite3 /app/data/llm-verifier.db ".backup backup-$(date +%Y%m%d).db"

# Configuration backup
cp /app/config.yaml /app/backups/config-$(date +%Y%m%d).yaml
```

## Security Considerations

1. **Network Security**: Use TLS/HTTPS in production
2. **Firewall**: Open only necessary ports (80/443)
3. **Secrets Management**: Use proper secret management
4. **Access Control**: Limit API access with authentication
5. **Regular Updates**: Keep dependencies updated

## Performance Tuning

### Database Optimization
```bash
# SQLite settings
export LLM_VERIFIER_DATABASE_PATH="/app/data/llm-verifier.db?cache=shared&mode=rwc"
```

### Resource Allocation
- **CPU**: 250m-500m per container
- **Memory**: 512Mi-1Gi per container
- **Storage**: SSD with 50+ IOPS

### Load Balancing
Use multiple replicas with load balancer for high availability.

## Advanced Features Deployment

### Supervisor System Configuration

The supervisor system provides intelligent task breakdown and parallel processing. Configure it for production use:

```yaml
# Supervisor configuration
supervisor:
  enabled: true
  max_workers: 10  # Scale based on CPU cores
  task_timeout: "30m"
  retry_attempts: 3

  # Database connection pooling for worker efficiency
  database:
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: "1h"

  # Quality assurance
  quality_checks:
    enabled: true
    validation_required: true
    human_review_threshold: 0.9
```

**Resource Requirements**:
- Additional 2-4GB RAM for supervisor workers
- CPU cores should be 2x number of max_workers
- Database connections: max_workers + buffer

### Context Management Deployment

For long-term context management with LLM summarization:

```yaml
# Context management configuration
context:
  long_term:
    enabled: true
    max_age: "168h"  # 7 days
    summarization_interval: "1h"
    compression_threshold: 0.8

  summarization:
    enabled: true
    provider: "anthropic"  # Dedicated summarization model
    model: "claude-3-haiku-20240307"
    batch_size: 10
    quality_preservation: true

  storage:
    type: "file"  # or "redis", "postgresql"
    path: "/app/context"
    max_size: "10GB"
    compression: true
```

**Storage Requirements**:
- 1-5GB for context storage depending on usage
- SSD storage recommended for performance
- Backup context data separately from main database

### Cloud Backup Integration

Configure automated backups to cloud storage:

```yaml
# Cloud backup configuration
backup:
  enabled: true
  provider: "aws"
  bucket: "llm-verifier-prod-backups"
  region: "us-east-1"
  prefix: "automated/"

  schedule: "0 */4 * * *"  # Every 4 hours
  compression: true
  encryption: true

  include:
    database: true
    configurations: true
    context: true  # Include context data
    reports: true

  retention:
    days: 30
    max_backups: 50
```

**Cloud Provider Permissions**:
- **AWS**: `s3:PutObject`, `s3:GetObject`, `s3:ListBucket`, `s3:DeleteObject`
- **GCP**: `Storage Object Admin` role
- **Azure**: `Storage Blob Data Contributor` role

### Vector Database Integration

For advanced RAG (Retrieval-Augmented Generation) capabilities:

```yaml
# Vector database configuration
vector:
  enabled: true
  provider: "cognee"  # or "pinecone", "weaviate", "qdrant"
  endpoint: "http://localhost:8000"
  api_key: "${COGNEE_API_KEY}"

  # Indexing configuration
  index:
    name: "llm-verifier-knowledge"
    dimension: 1536  # Match your embedding model
    metric: "cosine"

  # Embedding configuration
  embedding:
    provider: "openai"
    model: "text-embedding-3-large"
    batch_size: 100

  # Retrieval settings
  retrieval:
    top_k: 5
    score_threshold: 0.7
    rerank: true
```

**Additional Dependencies**:
- Vector database server (if using self-hosted)
- Embedding model API access
- Additional storage for vector indexes (5-20GB)

### Failover and Circuit Breaker Configuration

For production resilience:

```yaml
# Failover configuration
failover:
  enabled: true
  circuit_breaker:
    failure_threshold: 5
    recovery_timeout: "30s"
    monitoring_period: "1m"

  latency_routing:
    enabled: true
    max_latency: "5s"
    measurement_window: "5m"

  health_checking:
    interval: "30s"
    timeout: "10s"
    unhealthy_threshold: 3
    healthy_threshold: 2

  weighted_routing:
    cost_weight: 0.7
    performance_weight: 0.3
    update_interval: "5m"
```

**Monitoring Requirements**:
- External health check endpoints
- Metrics collection for latency and error rates
- Alerting on circuit breaker state changes

## Troubleshooting

### Common Issues

1. **JWT Secret Error**: Set `LLM_VERIFIER_API_JWT_SECRET`
2. **Database Permission**: Check file/directory permissions
3. **Port Conflict**: Ensure port 8080 is available
4. **Memory Issues**: Increase memory allocation
5. **Network Issues**: Check firewall and DNS

### Debug Mode

```bash
# Enable debug logging
export LLM_VERIFIER_LOGGING_LEVEL="debug"
export LLM_VERIFIER_PROFILE="dev"

# Verbose output
./llm-verifier server --verbose
```

### Log Analysis

```bash
# Error patterns
grep -i error /app/logs/llm-verifier.log

# Performance metrics
grep -i "slow" /app/logs/llm-verifier.log

# Security events
grep -i "auth" /app/logs/llm-verifier.log
```