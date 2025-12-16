# Deployment Guide

This guide covers deploying LLM Verifier in production environments.

## Prerequisites

- Go 1.21+ or Docker
- SQLite 3.x (if not using container)
- 2GB RAM minimum, 4GB recommended
- 10GB storage minimum, 50GB recommended

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