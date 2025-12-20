#!/bin/bash

# LLM Verifier Production Deployment Script
# This script handles the complete production deployment of LLM Verifier

set -e

# Configuration
NAMESPACE="llm-verifier"
VERSION=${1:-"v1.0.0"}
REGISTRY=${2:-"your-registry.com"}
ENVIRONMENT=${3:-"production"}
CLUSTER_NAME=${4:-"production-cluster"}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
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

# Pre-deployment checks
check_prerequisites() {
    log_info "Checking deployment prerequisites..."
    
    # Check if kubectl is available
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is required but not installed"
        exit 1
    fi
    
    # Check if helm is available (optional)
    if command -v helm &> /dev/null; then
        log_info "Helm is available"
    else
        log_warning "Helm is not available (optional)"
    fi
    
    # Check cluster connectivity
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster"
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Create namespace
create_namespace() {
    log_info "Creating namespace: $NAMESPACE"
    kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -
    log_success "Namespace created/updated"
}

# Apply secrets
apply_secrets() {
    log_info "Applying Kubernetes secrets..."
    
    # Create secrets from environment variables
    kubectl apply -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: llm-verifier-secrets
  namespace: $NAMESPACE
type: Opaque
data:
  db-host: $(echo -n "postgres" | base64)
  db-port: $(echo -n "5432" | base64)
  db-name: $(echo -n "llm_verifier" | base64)
  db-user: $(echo -n "llm_user" | base64)
  db-password: $(echo -n "${POSTGRES_PASSWORD:-secure_password}" | base64)
  jwt-secret: $(echo -n "${JWT_SECRET:-default-secret-change-in-production-32charslong}" | base64)
  openai-api-key: $(echo -n "${OPENAI_API_KEY:-}" | base64)
  anthropic-api-key: $(echo -n "${ANTHROPIC_API_KEY:-}" | base64)
  redis-password: $(echo -n "${REDIS_PASSWORD:-}" | base64)
EOF
    
    log_success "Secrets applied"
}

# Apply ConfigMaps
apply_configmaps() {
    log_info "Applying ConfigMaps..."
    
    kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: llm-verifier-config
  namespace: $NAMESPACE
data:
  production.yaml: |
    environment: $ENVIRONMENT
    debug: false
    log_level: info
    server:
      host: 0.0.0.0
      port: 8080
      read_timeout: 30s
      write_timeout: 30s
      idle_timeout: 60s
      tls:
        enabled: true
        cert_file: /app/ssl/tls.crt
        key_file: /app/ssl/tls.key
    performance:
      max_workers: 50
      worker_timeout: 5m
      queue_size: 1000
      memory_limit: 2147483648
      cpu_quota: 4
      enable_profiling: false
      enable_metrics: true
    monitoring:
      enabled: true
      prometheus:
        enabled: true
        port: 9090
        path: /metrics
        namespace: llm_verifier
      tracing:
        enabled: true
        provider: jaeger
        endpoint: http://jaeger:14268/api/traces
        service: llm-verifier
      logging:
        level: info
        format: json
        output: file
        max_size: 100
        max_backups: 10
        max_age: 30
        compress: true
    enterprise:
      rbac:
        enabled: true
        default_role: user
        admin_role: admin
        super_admin_role: super_admin
      multi_tenant:
        enabled: true
        default_tenant: default
        tenant_header: X-Tenant-ID
        isolation_mode: strict
      audit_logging:
        enabled: true
        storage: database
        retention: 2160h
        compression: true
    features:
      ai_assistant:
        enabled: true
      plugins:
        enabled: true
      analytics:
        enabled: true
      enterprise_features:
        enabled: true
      caching:
        enabled: true
      rate_limiting:
        enabled: true
    database:
      port: "5432"
EOF
    
    log_success "ConfigMaps applied"
}

# Apply Persistent Volumes
apply_volumes() {
    log_info "Applying PersistentVolumeClaims..."
    
    kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: llm-verifier-data
  namespace: $NAMESPACE
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
  storageClassName: fast-ssd
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: llm-verifier-logs
  namespace: $NAMESPACE
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
  storageClassName: fast-ssd
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: llm-verifier-ssl
  namespace: $NAMESPACE
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
  storageClassName: fast-ssd
EOF
    
    log_success "PersistentVolumeClaims applied"
}

# Deploy the application
deploy_application() {
    log_info "Deploying LLM Verifier application..."
    
    # Update the deployment with current image version
    sed "s|image: llm-verifier:latest|image: $REGISTRY/llm-verifier:$VERSION|g" k8s/deployment.yaml | kubectl apply -f -
    
    log_success "Application deployed"
}

# Apply services and ingress
apply_networking() {
    log_info "Applying services and ingress..."
    
    kubectl apply -f k8s/service.yaml
    kubectl apply -f k8s/ingress.yaml
    
    log_success "Networking configured"
}

# Deploy monitoring stack
deploy_monitoring() {
    log_info "Deploying monitoring stack..."
    
    # Create monitoring namespace if it doesn't exist
    kubectl create namespace monitoring --dry-run=client -o yaml | kubectl apply -f -
    
    # Deploy Prometheus
    if command -v helm &> /dev/null; then
        helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
        helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
            --namespace monitoring \
            --create-namespace \
            --set prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage=50Gi \
            --set grafana.adminPassword=${GRAFANA_PASSWORD:-admin123} \
            --set prometheus.prometheusSpec.serviceMonitor.enabled=true
    else
        kubectl apply -f monitoring/prometheus.yaml
    fi
    
    log_success "Monitoring stack deployed"
}

# Wait for deployment
wait_for_deployment() {
    log_info "Waiting for deployment to be ready..."
    
    # Wait for the deployment to be ready
    kubectl wait --for=condition=available --timeout=300s deployment/llm-verifier -n $NAMESPACE
    
    # Wait for the pods to be ready
    kubectl wait --for=condition=ready --timeout=300s pod -l app=llm-verifier -n $NAMESPACE
    
    log_success "Deployment is ready"
}

# Health check
health_check() {
    log_info "Performing health check..."
    
    # Get the external IP
    EXTERNAL_IP=""
    for i in {1..30}; do
        EXTERNAL_IP=$(kubectl get svc llm-verifier-service-external -n $NAMESPACE -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
        if [[ "$EXTERNAL_IP" != "" ]]; then
            break
        fi
        sleep 2
    done
    
    if [[ "$EXTERNAL_IP" != "" ]]; then
        # Perform health check
        if curl -f -s http://$EXTERNAL_IP/health > /dev/null; then
            log_success "Health check passed - Application is accessible at http://$EXTERNAL_IP"
        else
            log_error "Health check failed - Application not responding"
            exit 1
        fi
    else
        log_warning "External IP not available - health check skipped"
    fi
}

# Show deployment status
show_status() {
    log_info "Deployment Status:"
    echo ""
    echo "Namespace: $NAMESPACE"
    echo "Version: $VERSION"
    echo "Cluster: $CLUSTER_NAME"
    echo ""
    echo "Pods:"
    kubectl get pods -n $NAMESPACE -l app=llm-verifier
    echo ""
    echo "Services:"
    kubectl get services -n $NAMESPACE
    echo ""
    echo "Ingress:"
    kubectl get ingress -n $NAMESPACE
}

# Cleanup function
cleanup() {
    log_warning "Cleaning up deployment..."
    kubectl delete namespace $NAMESPACE --ignore-not-found=true
    log_success "Cleanup completed"
}

# Main deployment flow
main() {
    echo ""
    echo "${BLUE}🚀 LLM Verifier Production Deployment${NC}"
    echo "${BLUE}=========================================${NC}"
    echo ""
    
    case "${1:-deploy}" in
        "deploy")
            check_prerequisites
            create_namespace
            apply_secrets
            apply_configmaps
            apply_volumes
            deploy_application
            apply_networking
            deploy_monitoring
            wait_for_deployment
            health_check
            show_status
            ;;
        "update")
            deploy_application
            wait_for_deployment
            health_check
            ;;
        "monitoring")
            deploy_monitoring
            ;;
        "status")
            show_status
            ;;
        "cleanup")
            cleanup
            ;;
        *)
            echo "Usage: $0 {deploy|update|monitoring|status|cleanup} [version] [registry] [cluster-name]"
            echo ""
            echo "Commands:"
            echo "  deploy     - Full deployment (default)"
            echo "  update     - Update existing deployment"
            echo "  monitoring - Deploy monitoring stack only"
            echo "  status     - Show deployment status"
            echo "  cleanup    - Remove all resources"
            echo ""
            echo "Example: $0 deploy v1.0.0 your-registry.com production-cluster"
            exit 1
            ;;
    esac
    
    echo ""
    echo "${GREEN}✅ Deployment completed successfully!${NC}"
}

# Trap to handle cleanup on exit
trap cleanup EXIT

# Execute main function with all arguments
main "$@"