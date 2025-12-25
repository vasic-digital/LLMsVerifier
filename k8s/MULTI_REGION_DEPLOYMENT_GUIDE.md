# LLM Verifier Multi-Region Deployment Guide

This guide covers deploying LLM Verifier across multiple regions with global load balancing, high availability, and geo-based traffic routing.

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   US East       │    │   US West       │    │   EU West       │
│   (Primary)     │    │   (Backup)      │    │   (Backup)      │
│                 │    │                 │    │                 │
│ ┌─────────────┐ │    │ ┌─────────────┐ │    │ ┌─────────────┐ │
│ │ LLM Verifier│ │    │ │ LLM Verifier│ │    │ │ LLM Verifier│ │
│ │  Pods       │ │    │ │  Pods       │ │    │ │ LLM Verifier│ │
│ └─────────────┘ │    │ └─────────────┘ │    │ │  Pods       │ │
└─────────────────┘    └─────────────────┘    │ └─────────────┘ │
                                              └─────────────────┘
┌─────────────────────────────────────────────────────────────┐
│                  Global Load Balancer                      │
│  • Geo-based routing                                       │
│  • Health-based failover                                   │
│  • SSL termination                                         │
│  • DDoS protection                                         │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Global DNS                               │
│  • api.llm-verifier.com                                     │
│  • Geo-aware resolution                                     │
│  • Health-based routing                                     │
└─────────────────────────────────────────────────────────────┘
```

## Supported Cloud Providers

- **AWS**: EKS with Global Accelerator and Route 53
- **Google Cloud**: GKE with Global Load Balancer and Cloud DNS
- **Azure**: AKS with Front Door and Traffic Manager
- **Multi-cloud**: Hybrid deployments across providers

## Prerequisites

### System Requirements
- kubectl 1.24+
- helm 3.8+
- Docker 20.10+
- Cloud provider CLI (aws, gcloud, az)

### Cloud Resources
- 4 Kubernetes clusters (one per region)
- Global load balancer
- DNS domain (llm-verifier.com)
- SSL certificates
- Object storage buckets
- Managed databases (optional)

## Quick Start

### 1. Clone and Setup

```bash
git clone <repository>
cd llm-verifier/k8s
```

### 2. Configure Deployment

Edit the deployment script parameters:

```bash
# AWS Deployment
./deploy-multi-region.sh \
  --cloud-provider aws \
  --registry 123456789.dkr.ecr.us-east-1.amazonaws.com

# GCP Deployment
./deploy-multi-region.sh \
  --cloud-provider gcp \
  --registry gcr.io/my-project

# Azure Deployment
./deploy-multi-region.sh \
  --cloud-provider azure \
  --registry myregistry.azurecr.io
```

### 3. Deploy

```bash
# Full deployment with monitoring
./deploy-multi-region.sh --cloud-provider aws --registry <your-registry>

# Minimal deployment (skip optional components)
./deploy-multi-region.sh \
  --cloud-provider aws \
  --registry <your-registry> \
  --skip-monitoring \
  --skip-service-mesh
```

## Manual Deployment Steps

### Step 1: Build and Push Images

```bash
# Build for multiple architectures
docker buildx create --use
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t <your-registry>/llm-verifier:latest \
  --push .
```

### Step 2: Setup Secrets

Create region-specific secrets:

```bash
# For each region
kubectl create secret generic llm-verifier-secrets \
  --from-literal=openai-api-key=$OPENAI_API_KEY \
  --from-literal=anthropic-api-key=$ANTHROPIC_API_KEY \
  --from-literal=jwt-secret=$JWT_SECRET \
  --namespace llm-verifier
```

### Step 3: Deploy to Primary Region

```bash
# Switch to primary region context
kubectl config use-context aws-us-east-1

# Deploy base infrastructure
kubectl apply -f multi-region-deployment.yaml

# Wait for rollout
kubectl rollout status deployment/llm-verifier -n llm-verifier
```

### Step 4: Deploy to Other Regions

```bash
# Deploy to US West
kubectl config use-context aws-us-west-2
kubectl apply -f multi-region-deployment.yaml

# Deploy to EU West
kubectl config use-context aws-eu-west-1
kubectl apply -f multi-region-deployment.yaml

# Deploy to Asia Pacific
kubectl config use-context aws-ap-southeast-1
kubectl apply -f multi-region-deployment.yaml
```

### Step 5: Setup Global Load Balancing

```bash
# Switch back to primary region
kubectl config use-context aws-us-east-1

# Deploy global load balancer
kubectl apply -f global-load-balancer.yaml

# Get global IP
kubectl get svc llm-verifier-global -n llm-verifier
```

### Step 6: Configure DNS

Update your DNS provider:

```
api.llm-verifier.com     A     <global-load-balancer-ip>
*.llm-verifier.com        CNAME api.llm-verifier.com
```

## Traffic Routing Configuration

### Geo-Based Routing

Traffic is automatically routed based on user location:

- **North America**: US East (primary), US West (backup)
- **Europe**: EU West (primary), US East (backup)
- **Asia**: Asia Pacific (primary), US East (backup)
- **Other regions**: US East (default)

### Health-Based Failover

Automatic failover occurs when:
- Region health checks fail (3 consecutive failures)
- Response latency > 5 seconds
- Error rate > 5%

### Load Balancing Algorithms

- **Round Robin**: Even distribution across healthy instances
- **Least Connections**: Route to least loaded instances
- **Geo-aware**: Prefer closer regions
- **Weighted**: Manual traffic distribution

## Monitoring and Observability

### Metrics Collection

```bash
# Access Grafana
kubectl port-forward svc/prometheus-grafana 3000:80 -n monitoring

# Access Prometheus
kubectl port-forward svc/prometheus-kube-prometheus-prometheus 9090:9090 -n monitoring

# Access Jaeger
kubectl port-forward svc/jaeger-query 16686:16686 -n observability
```

### Key Metrics to Monitor

- **Request Latency**: P50, P95, P99
- **Error Rates**: 4xx, 5xx responses
- **Throughput**: Requests per second
- **Regional Health**: Per-region availability
- **Cross-region Traffic**: Inter-region communication

### Alerting Rules

```yaml
# Example alert for high latency
- alert: HighLatency
  expr: histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m])) > 2
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "High request latency detected"

# Example alert for regional failure
- alert: RegionalFailure
  expr: up{region=~".+"} == 0
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "Regional deployment failure detected"
```

## Service Mesh Configuration

### Istio Setup

The deployment includes Istio service mesh for:

- **Traffic Management**: Advanced routing and load balancing
- **Security**: mTLS encryption between services
- **Observability**: Distributed tracing and metrics
- **Resilience**: Circuit breakers and retry logic

### Traffic Policies

```yaml
# Circuit breaker configuration
apiVersion: networking.istio.io/v1beta1
kind: DestinationRule
metadata:
  name: llm-verifier-circuit-breaker
spec:
  host: llm-verifier
  trafficPolicy:
    connectionPool:
      tcp:
        maxConnections: 100
      http:
        http1MaxPendingRequests: 10
        maxRequestsPerConnection: 10
    outlierDetection:
      consecutive5xxErrors: 3
      interval: 10s
      baseEjectionTime: 30s
```

## Scaling and Performance

### Horizontal Pod Autoscaling

```yaml
# CPU-based scaling
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: llm-verifier-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: llm-verifier
  minReplicas: 3
  maxReplicas: 50
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

### Vertical Pod Autoscaling

```yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: llm-verifier-vpa
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: llm-verifier
  updatePolicy:
    updateMode: "Auto"
```

## Security Configuration

### Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: llm-verifier-security
spec:
  podSelector:
    matchLabels:
      app: llm-verifier
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
```

### RBAC Configuration

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: llm-verifier-operator
rules:
- apiGroups: ["apps", "extensions"]
  resources: ["deployments", "replicasets"]
  verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
```

## Backup and Disaster Recovery

### Database Backup

```bash
# Automated backups using Velero
velero backup create llm-verifier-db-backup \
  --include-namespaces llm-verifier \
  --selector app=llm-verifier
```

### Cross-Region Failover

The system automatically fails over when:
1. Primary region becomes unhealthy
2. Database replication lag > 30 seconds
3. Network partition detected

### Recovery Procedures

1. **Regional Failure**: Traffic automatically routes to healthy regions
2. **Cluster Failure**: Deploy new cluster in unaffected region
3. **Global Failure**: Fallback to static disaster recovery site

## Cost Optimization

### Resource Optimization

- **Spot Instances**: Use for non-critical workloads
- **Auto-scaling**: Scale down during low-traffic periods
- **Regional Pricing**: Deploy in cost-effective regions

### Monitoring Costs

```yaml
# Cost allocation tags
metadata:
  labels:
    team: ml-platform
    environment: production
    cost-center: ai-research
```

## Troubleshooting

### Common Issues

1. **DNS Propagation Delay**
   ```bash
   # Check DNS resolution
   dig api.llm-verifier.com
   nslookup api.llm-verifier.com
   ```

2. **Load Balancer Health Checks Failing**
   ```bash
   # Check pod health
   kubectl get pods -n llm-verifier
   kubectl logs deployment/llm-verifier -n llm-verifier
   ```

3. **Cross-Region Traffic Issues**
   ```bash
   # Check service mesh configuration
   istioctl proxy-config routes deploy/llm-verifier.llm-verifier
   ```

### Debug Commands

```bash
# Check global connectivity
curl -H "Host: api.llm-verifier.com" http://<global-lb-ip>/api/health

# Check regional deployments
for region in us-east us-west eu-west asia; do
  kubectl config use-context $region
  kubectl get pods -n llm-verifier
done

# Check service mesh
istioctl proxy-status
istioctl proxy-config listeners deploy/llm-verifier.llm-verifier
```

## Support and Maintenance

### Regular Maintenance Tasks

- **Certificate Renewal**: Automatic via cert-manager
- **Security Updates**: Automated via CI/CD
- **Performance Tuning**: Monthly review
- **Cost Optimization**: Weekly reports

### Contact Information

- **DevOps Team**: devops@llm-verifier.com
- **Security Team**: security@llm-verifier.com
- **Customer Support**: support@llm-verifier.com

## Appendix

### Region Mappings

| Region Code | Location | Cloud Provider |
|-------------|----------|----------------|
| us-east-1  | Virginia, USA | AWS |
| us-west-2  | Oregon, USA | AWS |
| eu-west-1  | Ireland, EU | AWS |
| ap-southeast-1 | Singapore, Asia | AWS |

### Service Ports

| Service | Port | Protocol | Description |
|---------|------|----------|-------------|
| API | 8080 | HTTP/HTTPS | REST API |
| Metrics | 9090 | HTTP | Prometheus metrics |
| Health | 8080 | HTTP | Health checks |
| Tracing | 9411 | HTTP | Jaeger tracing |

### Environment Variables

| Variable | Description | Required |
|----------|-------------|----------|
| REGION | Current region | Yes |
| CLUSTER_NAME | Cluster identifier | Yes |
| OPENAI_API_KEY | OpenAI API key | No |
| ANTHROPIC_API_KEY | Anthropic API key | No |
| JWT_SECRET | JWT signing secret | Yes |