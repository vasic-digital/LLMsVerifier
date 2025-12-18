# Kubernetes Deployment Files

This directory contains Kubernetes manifests for deploying LLM Verifier.

## Files

- `namespace.yaml` - Kubernetes namespace
- `secrets.yaml` - Application secrets (JWT secret, etc.)
- `pvc.yaml` - Persistent volume claim for data storage
- `deployment.yaml` - Application deployment with 3 replicas
- `service.yaml` - Internal service for the application
- `ingress.yaml` - External access with TLS termination
- `hpa.yaml` - Horizontal pod autoscaling configuration

## Deployment

1. **Create secrets:**
   ```bash
   # Generate and encode your JWT secret
   echo "your-jwt-secret" | base64
   # Update secrets.yaml with the encoded value
   ```

2. **Apply manifests:**
   ```bash
   kubectl apply -f namespace.yaml
   kubectl apply -f secrets.yaml
   kubectl apply -f pvc.yaml
   kubectl apply -f deployment.yaml
   kubectl apply -f service.yaml
   kubectl apply -f ingress.yaml
   kubectl apply -f hpa.yaml
   ```

3. **Verify deployment:**
   ```bash
   kubectl get pods -n llm-verifier
   kubectl get svc -n llm-verifier
   kubectl get ingress -n llm-verifier
   ```

## Configuration

### Environment Variables
- `LLM_VERIFIER_API_PORT`: API server port (default: 8080)
- `LLM_VERIFIER_PROFILE`: Environment profile (production/development)
- `LLM_VERIFIER_DATABASE_PATH`: SQLite database path
- `LLM_VERIFIER_API_JWT_SECRET`: JWT signing secret

### Resources
- **Memory**: 512Mi request, 1Gi limit per pod
- **CPU**: 250m request, 500m limit per pod
- **Storage**: 10Gi persistent volume
- **Replicas**: 3 minimum, autoscales to 10

### Autoscaling
- **CPU Target**: 70% utilization
- **Memory Target**: 80% utilization
- **Min Replicas**: 3
- **Max Replicas**: 10

### Health Checks
- **Liveness**: `/health` endpoint, 30s delay, 10s period
- **Readiness**: `/ready` endpoint, 5s delay, 5s period

## Security

- **TLS**: Automatic certificate management via cert-manager
- **Secrets**: Sensitive data in Kubernetes secrets
- **Network Policies**: Restrict pod communication as needed
- **RBAC**: Implement least-privilege access
