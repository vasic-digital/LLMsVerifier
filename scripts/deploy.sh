#!/bin/bash

# Enhanced LLM Verifier - Production Deployment Script
# Version: 1.0.0

set -euo pipefail

# Configuration
NAMESPACE="llm-verifier"
ENVIRONMENT="production"
DOCKER_REGISTRY="ghcr.io/your-org/llm-verifier"
HELM_CHART_PATH="./helm/llm-verifier"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${2}[$(date +'%Y-%m-%d %H:%M:%S')] ${NC} $1${NC}"
}

error() {
    echo -e "${RED}[$(date +'%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
    exit 1
}

success() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')] SUCCESS: $1${NC}"
}

warning() {
    echo -e "${YELLOW}[$(date +'%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

info() {
    echo -e "${BLUE}[$(date +'%Y-%m-%d %H:%M:%S')] INFO: $1${NC}"
}

# Pre-deployment checks
check_prerequisites() {
    log "Checking prerequisites..."
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        error "kubectl is not installed or not in PATH"
    fi
    
    # Check kubectl cluster access
    if ! kubectl cluster-info &> /dev/null; then
        error "Cannot access Kubernetes cluster"
    fi
    
    # Check helm
    if ! command -v helm &> /dev/null; then
        error "helm is not installed or not in PATH"
    fi
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        error "docker is not installed or not in PATH"
    fi
    
    # Check required files
    local required_files=(
        "config/production.yaml"
        "k8s/deployment.yaml"
        "k8s/namespace.yaml"
        "k8s/secrets.yaml"
        "monitoring/prometheus.yml"
        "monitoring/alert_rules.yml"
    )
    
    for file in "${required_files[@]}"; do
        if [[ ! -f "$file" ]]; then
            error "Required file not found: $file"
        fi
    done
    
    success "Prerequisites check completed"
}

# Environment validation
validate_environment() {
    log "Validating environment variables..."
    
    local required_vars=(
        "DB_PASSWORD"
        "JWT_SECRET"
        "OPENAI_API_KEY"
    )
    
    local missing_vars=()
    
    for var in "${required_vars[@]}"; do
        if [[ -z "${!var:-}" ]]; then
            missing_vars+=("$var")
        fi
    done
    
    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        error "Missing required environment variables: ${missing_vars[*]}"
    fi
    
    success "Environment validation completed"
}

# Namespace creation
create_namespace() {
    log "Creating namespace: $NAMESPACE"
    
    if kubectl get namespace "$NAMESPACE" &> /dev/null; then
        warning "Namespace $NAMESPACE already exists"
    else
        kubectl create namespace "$NAMESPACE"
        success "Namespace $NAMESPACE created"
    fi
    
    # Apply labels
    kubectl label namespace "$NAMESPACE" \
        name=llm-verifier \
        environment=$ENVIRONMENT \
        managed-by=llm-verifier-deploy \
        --overwrite
}

# Secrets management
create_secrets() {
    log "Creating secrets..."
    
    # Database secret
    kubectl create secret generic llm-verifier-secrets \
        --from-literal=db-host="${DB_HOST:-postgres}" \
        --from-literal=db-port="${DB_PORT:-5432}" \
        --from-literal=db-name="${DB_NAME:-llm_verifier}" \
        --from-literal=db-user="${DB_USER:-llm_verifier}" \
        --from-literal=db-password="$DB_PASSWORD" \
        --namespace="$NAMESPACE" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    # JWT secret
    kubectl create secret generic llm-verifier-jwt \
        --from-literal=jwt-secret="$JWT_SECRET" \
        --namespace="$NAMESPACE" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    # API secrets
    kubectl create secret generic llm-verifier-api-keys \
        --from-literal=openai-api-key="$OPENAI_API_KEY" \
        --from-literal=anthropic-api-key="${ANTHROPIC_API_KEY:-}" \
        --from-literal=google-api-key="${GOOGLE_API_KEY:-}" \
        --from-literal=azure-api-key="${AZURE_API_KEY:-}" \
        --namespace="$NAMESPACE" \
        --dry-run=client -o yaml | kubectl apply -f -
    
    # SSL secret (if certificates exist)
    if [[ -f "ssl/tls.crt" ]] && [[ -f "ssl/tls.key" ]]; then
        kubectl create secret tls llm-verifier-ssl \
            --cert=ssl/tls.crt \
            --key=ssl/tls.key \
            --namespace="$NAMESPACE" \
            --dry-run=client -o yaml | kubectl apply -f -
    fi
    
    success "Secrets created successfully"
}

# ConfigMap creation
create_configmaps() {
    log "Creating ConfigMaps..."
    
    # Application configuration
    kubectl apply -f k8s/configmap.yaml --namespace="$NAMESPACE"
    
    # Monitoring configuration
    kubectl apply -f monitoring/configmap.yaml --namespace="$NAMESPACE"
    
    success "ConfigMaps created successfully"
}

# Database deployment
deploy_database() {
    log "Deploying database..."
    
    # Deploy PostgreSQL
    kubectl apply -f k8s/postgres.yaml --namespace="$NAMESPACE"
    
    # Wait for database to be ready
    kubectl wait --for=condition=available --timeout=300s \
        deployment/postgres --namespace="$NAMESPACE"
    
    # Run database migrations
    kubectl wait --for=condition=ready --timeout=60s \
        pod -l app=postgres --namespace="$NAMESPACE"
    
    kubectl exec -n "$NAMESPACE" deployment/postgres -- \
        psql -U postgres -d llm_verifier -c "
            CREATE TABLE IF NOT EXISTS schema_migrations (
                version VARCHAR(50) PRIMARY KEY,
                applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
            );
        "
    
    success "Database deployed successfully"
}

# Redis deployment
deploy_redis() {
    log "Deploying Redis..."
    
    kubectl apply -f k8s/redis.yaml --namespace="$NAMESPACE"
    
    # Wait for Redis to be ready
    kubectl wait --for=condition=available --timeout=180s \
        deployment/redis --namespace="$NAMESPACE"
    
    success "Redis deployed successfully"
}

# Monitoring stack deployment
deploy_monitoring() {
    log "Deploying monitoring stack..."
    
    # Deploy Prometheus
    kubectl apply -f k8s/prometheus.yaml --namespace="$NAMESPACE"
    kubectl wait --for=condition=available --timeout=180s \
        deployment/prometheus --namespace="$NAMESPACE"
    
    # Deploy Grafana
    kubectl apply -f k8s/grafana.yaml --namespace="$NAMESPACE"
    kubectl wait --for=condition=available --timeout=180s \
        deployment/grafana --namespace="$NAMESPACE"
    
    # Deploy Jaeger
    kubectl apply -f k8s/jaeger.yaml --namespace="$NAMESPACE"
    kubectl wait --for=condition=available --timeout=180s \
        deployment/jaeger --namespace="$NAMESPACE"
    
    # Apply alert rules
    kubectl apply -f monitoring/alert_rules.yaml --namespace="$NAMESPACE"
    
    success "Monitoring stack deployed successfully"
}

# Application deployment
deploy_application() {
    log "Deploying application..."
    
    # Deploy the main application
    kubectl apply -f k8s/deployment.yaml --namespace="$NAMESPACE"
    
    # Wait for deployment to be ready
    kubectl wait --for=condition=available --timeout=300s \
        deployment/llm-verifier --namespace="$NAMESPACE"
    
    success "Application deployed successfully"
}

# Blue-green deployment
blue_green_deploy() {
    log "Performing blue-green deployment..."
    
    local current_version=$(kubectl get service llm-verifier-service \
        -n "$NAMESPACE" -o jsonpath='{.spec.selector.version}' 2>/dev/null || echo "blue")
    local new_version="green"
    
    if [[ "$current_version" == "green" ]]; then
        new_version="blue"
    fi
    
    # Update the inactive environment
    kubectl set image deployment/llm-verifier-$new_version \
        llm-verifier=$DOCKER_REGISTRY:$IMAGE_TAG \
        --namespace="$NAMESPACE"
    
    # Wait for the new version to be ready
    kubectl wait --for=condition=available --timeout=300s \
        deployment/llm-verifier-$new_version --namespace="$NAMESPACE"
    
    # Run health checks
    log "Running health checks on $new_version environment..."
    local health_check_passed=false
    
    for i in {1..10}; do
        if kubectl exec -n "$NAMESPACE" deployment/llm-verifier-$new_version \
            -- curl -f http://localhost:8080/health; then
            health_check_passed=true
            break
        fi
        sleep 10
    done
    
    if [[ "$health_check_passed" == "true" ]]; then
        # Switch traffic to the new version
        kubectl patch service llm-verifier-service -n "$NAMESPACE" \
            -p "{\"spec\":{\"selector\":{\"version\":\"$new_version\"}}"
        
        success "Traffic switched to $new_version environment"
        
        # Wait for verification
        sleep 30
        
        # Verify traffic routing
        local switch_check_passed=false
        for i in {1..5}; do
            if curl -f "https://api.llm-verifier.com/health"; then
                switch_check_passed=true
                break
            fi
            sleep 5
        done
        
        if [[ "$switch_check_passed" == "true" ]]; then
            success "Blue-green deployment completed successfully"
            
            # Update the other version for next deployment
            kubectl set image deployment/llm-verifier-$current_version \
                llm-verifier=$DOCKER_REGISTRY:$IMAGE_TAG \
                --namespace="$NAMESPACE"
        else
            error "Traffic switch verification failed"
        fi
    else
        error "Health check failed for $new_version environment"
    fi
}

# Health verification
verify_deployment() {
    log "Verifying deployment..."
    
    # Check pod status
    local pod_status=$(kubectl get pods -n "$NAMESPACE" -l app=llm-verifier \
        -o jsonpath='{.items[0].status.phase}' 2>/dev/null)
    
    if [[ "$pod_status" != "Running" ]]; then
        error "Application pod is not running: $pod_status"
    fi
    
    # Check service endpoints
    local service_url=$(kubectl get service llm-verifier-service-external \
        -n "$NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].hostname}' 2>/dev/null)
    
    if [[ -z "$service_url" ]]; then
        error "External service endpoint not available"
    fi
    
    # Run comprehensive health checks
    local health_endpoints=(
        "/health"
        "/ready"
        "/api/v1/health"
        "/metrics"
    )
    
    for endpoint in "${health_endpoints[@]}"; do
        if ! curl -f --max-time 30 "https://$service_url$endpoint"; then
            error "Health check failed for endpoint: $endpoint"
        fi
    done
    
    success "Deployment verification completed"
}

# Performance testing
run_performance_tests() {
    log "Running performance tests..."
    
    local service_url=$(kubectl get service llm-verifier-service-external \
        -n "$NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
    
    # Install k6 if not present
    if ! command -v k6 &> /dev/null; then
        log "Installing k6 for performance testing..."
        curl -L https://github.com/grafana/k6/releases/download/v0.47.0/k6-v0.47.0-linux-amd64.tar.gz | tar xz
        sudo mv k6-v0.47.0-linux-amd64/k6 /usr/local/bin/k6
        sudo chmod +x /usr/local/bin/k6
    fi
    
    # Run performance test
    k6 run --out json=results.json tests/performance/load_test.js \
        --vus 100 \
        --duration 5m \
        --http-url "https://$service_url"
    
    # Analyze results
    if [[ -f "results.json" ]]; then
        local avg_response_time=$(jq -r '.metrics.http_req_duration.avg' results.json)
        local p95_response_time=$(jq -r '.metrics.http_req_duration["p(95)"]' results.json)
        local error_rate=$(jq -r '.metrics.http_req_failed.rate' results.json)
        
        info "Performance Test Results:"
        info "  Average Response Time: ${avg_response_time}s"
        info "  95th Percentile: ${p95_response_time}s"
        info "  Error Rate: ${error_rate}"
        
        # Check against thresholds
        if (( $(echo "$p95_response_time > 2.0" | bc -l) )); then
            warning "P95 response time exceeds 2.0s threshold"
        fi
        
        if (( $(echo "$error_rate > 0.01" | bc -l) )); then
            warning "Error rate exceeds 1% threshold"
        fi
    fi
    
    success "Performance testing completed"
}

# Security scanning
run_security_scan() {
    log "Running security scan..."
    
    local service_url=$(kubectl get service llm-verifier-service-external \
        -n "$NAMESPACE" -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
    
    # Run OWASP ZAP Baseline Scan
    if command -v zap-baseline &> /dev/null; then
        zap-baseline -t $service_url -j zap-report.json
        
        if [[ -f "zap-report.json" ]]; then
            local high_alerts=$(jq -r '.site.alerts[] | select(.riskid=="High") | length' zap-report.json)
            local medium_alerts=$(jq -r '.site.alerts[] | select(.riskid=="Medium") | length' zap-report.json)
            
            info "Security Scan Results:"
            info "  High Risk Alerts: $high_alerts"
            info "  Medium Risk Alerts: $medium_alerts"
            
            if [[ $high_alerts -gt 0 ]]; then
                warning "High risk vulnerabilities detected"
            fi
        fi
    else
        warning "ZAP security scanner not available"
    fi
    
    success "Security scanning completed"
}

# Rollback procedure
rollback_deployment() {
    log "Performing rollback..."
    
    # Get previous deployment info
    local previous_image=$(kubectl rollout history deployment/llm-verifier \
        -n "$NAMESPACE" --revision=1 --template='{{.metadata.annotations.kubernetes.io/revision-image}}')
    
    if [[ -z "$previous_image" ]]; then
        error "No previous deployment found for rollback"
    fi
    
    # Perform rollback
    kubectl rollout undo deployment/llm-verifier --namespace="$NAMESPACE"
    
    # Wait for rollback to complete
    kubectl wait --for=condition=available --timeout=300s \
        deployment/llm-verifier --namespace="$NAMESPACE"
    
    success "Rollback completed successfully"
}

# Backup procedure
backup_deployment() {
    log "Creating deployment backup..."
    
    local backup_dir="backups/$(date +%Y%m%d_%H%M%S)"
    mkdir -p "$backup_dir"
    
    # Backup Kubernetes resources
    kubectl get all -n "$NAMESPACE" -o yaml > "$backup_dir/k8s-resources.yaml"
    
    # Backup configuration
    cp -r config/ "$backup_dir/"
    
    # Backup database
    kubectl exec -n "$NAMESPACE" deployment/postgres -- \
        pg_dump llm_verifier | gzip > "$backup_dir/database.sql.gz"
    
    # Create backup metadata
    cat > "$backup_dir/metadata.json" << EOF
{
    "timestamp": "$(date -Iseconds)",
    "namespace": "$NAMESPACE",
    "environment": "$ENVIRONMENT",
    "image_tag": "$IMAGE_TAG",
    "backup_type": "manual"
}
EOF
    
    success "Backup created in $backup_dir"
}

# Cleanup procedure
cleanup_deployment() {
    log "Performing cleanup..."
    
    # Remove failed pods
    kubectl delete pods -n "$NAMESPACE" \
        -l app=llm-verifier \
        --field-selector=status.phase=Failed \
        --ignore-not-found=true
    
    # Clean up old images
    docker image prune -f --filter "label=org.opencontainers.image.source=https://github.com/your-org/llm-verifier"
    
    # Clean up old ConfigMaps and Secrets
    kubectl delete configmaps -n "$NAMESPACE" \
        -l managed-by=llm-verifier-deploy \
        --older-than=24h \
        --ignore-not-found=true
    
    success "Cleanup completed"
}

# Main deployment function
deploy() {
    log "Starting production deployment..."
    
    check_prerequisites
    validate_environment
    create_namespace
    create_secrets
    create_configmaps
    deploy_database
    deploy_redis
    deploy_monitoring
    
    if [[ "${DEPLOYMENT_STRATEGY:-blue-green}" == "blue-green" ]]; then
        blue_green_deploy
    else
        deploy_application
    fi
    
    verify_deployment
    run_performance_tests
    run_security_scan
    
    success "Production deployment completed successfully"
}

# Signal handlers
trap 'error "Script interrupted"; exit 1' INT TERM

# Help function
show_help() {
    cat << EOF
Enhanced LLM Verifier Production Deployment Script

Usage: $0 [OPTIONS]

OPTIONS:
    -h, --help              Show this help message
    -e, --environment       Environment (default: production)
    -n, --namespace          Kubernetes namespace (default: llm-verifier)
    -t, --tag              Docker image tag (required)
    -s, --strategy           Deployment strategy (default: rolling)
                           Options: rolling, blue-green
    --skip-tests           Skip performance and security tests
    --rollback              Perform rollback to previous deployment
    --backup                Create deployment backup
    --cleanup               Perform cleanup operations

EXAMPLES:
    $0 -t v1.0.0 -e production -s blue-green
    $0 --rollback -n llm-verifier
    $0 --backup -n llm-verifier

ENVIRONMENT VARIABLES:
    DB_PASSWORD           PostgreSQL password
    JWT_SECRET             JWT signing secret
    OPENAI_API_KEY        OpenAI API key
    ANTHROPIC_API_KEY     Anthropic API key
    GOOGLE_API_KEY         Google AI API key
    AZURE_API_KEY          Azure API key
    DB_HOST               Database host
    DB_PORT               Database port
    DB_NAME               Database name
    DB_USER               Database username

EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -e|--environment)
            ENVIRONMENT="$2"
            shift 2
            ;;
        -n|--namespace)
            NAMESPACE="$2"
            shift 2
            ;;
        -t|--tag)
            IMAGE_TAG="$2"
            shift 2
            ;;
        -s|--strategy)
            DEPLOYMENT_STRATEGY="$2"
            shift 2
            ;;
        --skip-tests)
            SKIP_TESTS=true
            shift
            ;;
        --rollback)
            ROLLBACK=true
            shift
            ;;
        --backup)
            BACKUP=true
            shift
            ;;
        --cleanup)
            CLEANUP=true
            shift
            ;;
        *)
            error "Unknown option: $1"
            ;;
    esac
done

# Check required arguments
if [[ -z "${IMAGE_TAG:-}" ]] && [[ -z "${ROLLBACK:-}" ]] && [[ -z "${BACKUP:-}" ]] && [[ -z "${CLEANUP:-}" ]]; then
    error "Image tag is required. Use -t option or specify --rollback/--backup/--cleanup"
fi

# Execute main function based on arguments
if [[ "${ROLLBACK:-}" == "true" ]]; then
    rollback_deployment
elif [[ "${BACKUP:-}" == "true" ]]; then
    backup_deployment
elif [[ "${CLEANUP:-}" == "true" ]]; then
    cleanup_deployment
else
    deploy
fi