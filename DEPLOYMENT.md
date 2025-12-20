# LLM Verifier Production Deployment Guide

This comprehensive guide covers the production deployment of the Enhanced LLM Verifier platform v1.0.0.

## 📋 Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start](#quick-start)
3. [Configuration](#configuration)
4. [Deployment Methods](#deployment-methods)
5. [Security Setup](#security-setup)
6. [Monitoring Setup](#monitoring-setup)
7. [Troubleshooting](#troubleshooting)
8. [Maintenance](#maintenance)

## 🔧 Prerequisites

### Required Tools

- **Docker** >= 20.10.0
- **Docker Compose** >= 2.0.0
- **Kubernetes** >= 1.24.0 (for K8s deployment)
- **kubectl** configured for cluster access
- **Helm** >= 3.8.0 (optional, for monitoring stack)

### Required Resources

- **Minimum**: 2 CPU cores, 4GB RAM, 20GB storage
- **Recommended**: 4 CPU cores, 8GB RAM, 50GB storage
- **Production**: 8+ CPU cores, 16GB+ RAM, 100GB+ storage

### Network Requirements

- **Ports**: 80, 443, 8080, 5432, 6379, 9090, 3000
- **Load Balancer**: Required for external access
- **SSL/TLS**: Recommended for production

## 🚀 Quick Start

### Option 1: Docker Compose (Recommended for Testing)

```bash
# Clone repository
git clone https://github.com/your-org/llm-verifier.git
cd llm-verifier

# Copy production environment
cp .env.production.example .env.production

# Edit environment variables
nano .env.production

# Deploy with Docker Compose
docker-compose -f docker-compose.prod.yml up -d

# Check status
docker-compose -f docker-compose.prod.yml ps
```

### Option 2: Kubernetes (Production Recommended)

```bash
# Make deployment script executable
chmod +x deploy.sh

# Deploy to Kubernetes
./deploy.sh deploy v1.0.0 your-registry.com production-cluster

# Check deployment status
./deploy.sh status
```

## ⚙️ Configuration

### Environment Variables

Copy `.env.production.example` to `.env.production` and configure:

#### Core Settings
```bash
# Application
TAG=v1.0.0
PORT=8080
GIN_MODE=release
TZ=UTC

# Security
JWT_SECRET=your-super-secure-jwt-secret-32-chars-long
DATABASE_ENCRYPTION_KEY=your-32-character-encryption-key-here
```

#### Database Configuration

```bash
# SQLite (default)
LLM_DB_PATH=/app/data/llm-verifier.db

# PostgreSQL (uncomment)
POSTGRES_CONNECTION_STRING=postgres://user:password@postgres:5432/llm_verifier?sslmode=require
```

#### API Keys

```bash
# Configure your LLM providers
OPENAI_API_KEY=sk-your-openai-key
ANTHROPIC_API_KEY=sk-ant-your-anthropic-key
GOOGLE_API_KEY=your-google-api-key
META_API_KEY=your-meta-api-key
```

### Security Configuration

#### JWT Authentication
- Generate secure JWT secret (32+ characters)
- Set proper token expiration
- Enable token refresh mechanism

#### Database Encryption
- Enable SQL Cipher for data at rest
- Rotate encryption keys regularly
- Backup encryption keys securely

#### Rate Limiting
- Configure per-IP and per-user limits
- Set appropriate burst limits
- Monitor and adjust based on usage

## 📦 Deployment Methods

### Docker Compose Deployment

#### Single Node Deployment

```bash
# Start all services
docker-compose -f docker-compose.prod.yml up -d

# Scale specific services
docker-compose -f docker-compose.prod.yml up -d --scale llm-verifier=3

# View logs
docker-compose -f docker-compose.prod.yml logs -f llm-verifier
```

#### High Availability Deployment

```yaml
# docker-compose.ha.yml
version: '3.8'

services:
  # Load balancer
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
    volumes:
      - ./nginx/ha.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      - llm-verifier-1
      - llm-verifier-2

  # Application instances
  llm-verifier-1:
    image: llm-verifier:v1.0.0
    environment:
      - INSTANCE_ID=1
    volumes:
      - llm_verifier_data_1:/app/data

  llm-verifier-2:
    image: llm-verifier:v1.0.0
    environment:
      - INSTANCE_ID=2
    volumes:
      - llm_verifier_data_2:/app/data
```

### Kubernetes Deployment

#### Namespace Setup

```bash
# Create dedicated namespace
kubectl create namespace llm-verifier
kubectl config set-context --current --namespace=llm-verifier
```

#### Secrets Management

```bash
# Create secrets from environment file
kubectl apply -f k8s/secrets.yaml

# Or using kubectl create secret
kubectl create secret generic llm-verifier-secrets \
  --from-literal=jwt-secret="$JWT_SECRET" \
  --from-literal=db-password="$DB_PASSWORD" \
  --from-literal=openai-api-key="$OPENAI_API_KEY"
```

#### Deployment Strategy

```yaml
# k8s/production-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: llm-verifier
  namespace: llm-verifier
  labels:
    app: llm-verifier
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
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
        image: your-registry/llm-verifier:v1.0.0
        ports:
        - name: http
          containerPort: 8080
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "2000m"
        livenessProbe:
          httpGet:
            path: /health
            port: http
            initialDelaySeconds: 30
            periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: http
            initialDelaySeconds: 5
            periodSeconds: 5
```

## 🔒 Security Setup

### SSL/TLS Configuration

#### Self-Signed Certificates (Development)

```bash
# Generate certificates
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365

# Create secret
kubectl create secret tls llm-verifier-tls \
  --key-file=key.pem \
  --cert-file=cert.pem
```

#### Let's Encrypt (Production)

```yaml
# cert-manager/certificate.yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: llm-verifier-tls
  namespace: llm-verifier
spec:
  secretName: llm-verifier-tls
  issuerRef:
    name: letsencrypt-prod
  dnsNames:
  - api.llm-verifier.com
```

### Network Security

#### Network Policies

```yaml
# k8s/network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: llm-verifier-netpol
  namespace: llm-verifier
spec:
  podSelector:
    matchLabels:
      app: llm-verifier
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
```

### Pod Security Policies

```yaml
# k8s/pod-security-policy.yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: llm-verifier-psp
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
  - ALL
  volumes:
  - configMap
  - emptyDir
  - projected
  - secret
  - downwardAPI
  - persistentVolumeClaim
  runAsUser:
    rule: MustRunAsNonRoot
  fsGroup:
    rule: MustRunAs
```

## 📊 Monitoring Setup

### Prometheus Configuration

```yaml
# monitoring/prometheus.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
      evaluation_interval: 15s
      
    scrape_configs:
      - job_name: 'llm-verifier'
        static_configs:
          - targets: ['llm-verifier-service:9090']
        metrics_path: /metrics
        scrape_interval: 10s
        
    rule_files:
      - "alert_rules.yml"
      
    alerting:
      alertmanagers:
        - static_configs:
          - targets:
            - alertmanager:9093
```

### Grafana Dashboards

```json
// monitoring/grafana/dashboards/llm-verifier-dashboard.json
{
  "dashboard": {
    "title": "LLM Verifier Dashboard",
    "tags": ["llm-verifier"],
    "timezone": "browser",
    "panels": [
      {
        "title": "API Response Time",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))",
            "legendFormat": "99th percentile"
          }
        ]
      },
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])",
            "legendFormat": "requests/sec"
          }
        ]
      }
    ]
  }
}
```

### Alerting Rules

```yaml
# monitoring/alert-rules.yml
groups:
  - name: llm-verifier.rules
    rules:
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) > 0.1
        for: 2m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} errors per second"
          
      - alert: HighLatency
        expr: histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m])) > 2
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "High API latency"
          description: "99th percentile latency is {{ $value }} seconds"
```

## 🔧 Troubleshooting

### Common Issues

#### Application Not Starting

```bash
# Check logs
kubectl logs -n llm-verifier deployment/llm-verifier

# Check events
kubectl get events -n llm-verifier --sort-by='.lastTimestamp'

# Describe pod
kubectl describe pod -n llm-verifier <pod-name>

# Check configuration
kubectl get configmap llm-verifier-config -o yaml
```

#### Database Connection Issues

```bash
# Check database pod
kubectl get pods -n llm-verifier -l app=postgres

# Test database connection
kubectl exec -it postgres-xxx -- psql -U llm_user -d llm_verifier

# Check secrets
kubectl get secret llm-verifier-secrets -o yaml
```

#### High Memory/CPU Usage

```bash
# Check resource usage
kubectl top pods -n llm-verifier

# Check limits
kubectl describe pod -n llm-verifier <pod-name>

# Add more resources
kubectl patch deployment llm-verifier -p '{"spec":{"template":{"spec":{"containers":[{"name":"llm-verifier","resources":{"limits":{"memory":"4Gi"}}]}}}}'
```

#### External Access Issues

```bash
# Check service
kubectl get svc -n llm-verifier

# Check ingress
kubectl get ingress -n llm-verifier

# Test connectivity
kubectl port-forward svc/llm-verifier-service 8080:8080

# Check DNS
nslookup api.llm-verifier.com
```

### Health Checks

```bash
# Application health
curl http://api.llm-verifier.com/health

# Readiness check
curl http://api.llm-verifier.com/ready

# Metrics endpoint
curl http://api.llm-verifier.com/metrics

# Detailed health information
curl http://api.llm-verifier.com/health?detailed=true
```

## 🔧 Maintenance

### Regular Tasks

#### Daily

```bash
# Check system health
./deploy.sh status

# Review logs for errors
kubectl logs --since=24h -n llm-verifier deployment/llm-verifier | grep ERROR

# Backup data
kubectl exec -it deployment/llm-verifier -- /app/backup.sh
```

#### Weekly

```bash
# Update application
./deploy.sh update v1.0.1

# Rotate secrets
./deploy.sh rotate-secrets

# Clean up old resources
kubectl delete pods -n llm-verifier --field-selector=status.phase=Succeeded --since=72h
```

#### Monthly

```bash
# Security audit
kubectl exec -it deployment/llm-verifier -- security-audit.sh

# Performance review
./deploy.sh performance-review

# Capacity planning
./deploy.sh capacity-plan
```

### Backup and Recovery

#### Automated Backups

```bash
# Enable database backups
kubectl annotate deployment llm-verifier backup.enabled=true

# Configure backup schedule
kubectl patch configmap llm-verifier-config -p '{"data":{"backup_schedule":"0 2 * * *"}}'

# Backup to external storage
kubectl apply -f - <<EOF
apiVersion: v1
kind: CronJob
metadata:
  name: llm-verifier-backup
  namespace: llm-verifier
spec:
  schedule: "0 2 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: your-registry/backup-tools:v1.0.0
            command: ["/backup.sh"]
            env:
              - name: DB_CONNECTION
                valueFrom:
                  secretKeyRef:
                    name: llm-verifier-secrets
                    key: postgres-connection
            volumeMounts:
              - name: backup-storage
                mountPath: /backups
          volumes:
            - name: backup-storage
              persistentVolumeClaim:
                claimName: backup-pvc
EOF
```

#### Disaster Recovery

```bash
# Disaster recovery checklist
echo "1. Verify backup integrity"
echo "2. Prepare recovery environment"
echo "3. Restore from latest backup"
echo "4. Validate restored data"
echo "5. Update DNS records"
echo "6. Monitor system performance"
echo "7. Notify users of recovery"
```

## 📈 Scaling and Performance

### Horizontal Pod Autoscaler

```yaml
# k8s/hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: llm-verifier-hpa
  namespace: llm-verifier
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: llm-verifier
  minReplicas: 2
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 100
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
```

### Cluster Autoscaling

```yaml
# k8s/cluster-autoscaler.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: cluster-autoscaler-config
data:
  config.yaml: |
    nodeGroups:
    - name: llm-verifier-group
      maxNodes: 10
      minNodes: 2
      scaleDownDelayAfterAdd: 10m
      scaleDownUnneededTime: 10m
```

## 🎯 Production Best Practices

### Security Checklist

- [ ] Enable mutual TLS for all services
- [ ] Use network policies for pod communication
- [ ] Implement RBAC with least privilege
- [ ] Enable audit logging for all actions
- [ ] Regular security scanning of images
- [ ] Secrets rotation policy in place
- [ ] DDoS protection enabled
- [ ] Input validation and sanitization
- [ ] SQL injection protection enabled

### Performance Checklist

- [ ] Connection pooling configured
- [ ] Query optimization indexes in place
- [ ] Caching layer enabled and configured
- [ ] Resource limits appropriate for workload
- [ ] Monitoring and alerting active
- [ ] Load testing performed
- [ ] Performance baseline established
- [ ] Autoscaling policies configured

### Reliability Checklist

- [ ] Health checks implemented
- [ ] Graceful shutdown handling
- [ ] Retry logic for external calls
- [ ] Circuit breakers for external services
- [ ] Backup and restore procedures
- [ ] Multi-zone deployment
- [ ] Failover mechanisms tested
- [ ] Monitoring coverage complete

---

## 📞 Support

For production deployment support, contact:

- **Documentation**: [https://docs.llm-verifier.com](https://docs.llm-verifier.com)
- **GitHub Issues**: [https://github.com/your-org/llm-verifier/issues](https://github.com/your-org/llm-verifier/issues)
- **Community**: [https://discord.gg/llm-verifier](https://discord.gg/llm-verifier)
- **Enterprise Support**: enterprise@llm-verifier.com

## 🔄 Updates and Maintenance

- **Patch Releases**: Weekly on Tuesdays
- **Minor Releases**: Monthly on first Friday
- **Major Releases**: Quarterly
- **Maintenance Windows**: Saturdays 2:00-4:00 UTC
- **Security Updates**: As needed (immediate for critical)

---

**Version**: v1.0.0  
**Last Updated**: 2024-01-01  
**Next Review**: 2024-02-01