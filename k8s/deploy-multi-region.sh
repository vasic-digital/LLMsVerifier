#!/bin/bash

# LLM Verifier Multi-Region Deployment Script
# This script deploys LLM Verifier across multiple regions with global load balancing

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
REGIONS=("us-east-1" "us-west-2" "eu-west-1" "ap-southeast-1")
PRIMARY_REGION="us-east-1"

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

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check if kubectl is installed
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed. Please install kubectl first."
        exit 1
    fi

    # Check if helm is installed
    if ! command -v helm &> /dev/null; then
        log_error "helm is not installed. Please install helm first."
        exit 1
    fi

    # Check if docker is installed
    if ! command -v docker &> /dev/null; then
        log_error "docker is not installed. Please install docker first."
        exit 1
    fi

    log_success "Prerequisites check passed"
}

# Setup cloud provider CLI
setup_cloud_cli() {
    local provider=$1

    case $provider in
        "aws")
            if ! command -v aws &> /dev/null; then
                log_error "AWS CLI is not installed"
                exit 1
            fi
            aws configure
            ;;
        "gcp")
            if ! command -v gcloud &> /dev/null; then
                log_error "Google Cloud SDK is not installed"
                exit 1
            fi
            gcloud auth login
            ;;
        "azure")
            if ! command -v az &> /dev/null; then
                log_error "Azure CLI is not installed"
                exit 1
            fi
            az login
            ;;
        *)
            log_warning "Unknown cloud provider: $provider"
            ;;
    esac
}

# Build and push Docker image
build_and_push_image() {
    local registry=$1
    local tag=${2:-latest}

    log_info "Building Docker image..."

    cd "$PROJECT_ROOT"

    # Build the application
    if [ -f "llm-verifier/go.mod" ]; then
        cd llm-verifier
        GOOS=linux GOARCH=amd64 go build -o ../llm-verifier-app ./cmd
        cd ..
    fi

    # Build Docker image
    docker build -t "$registry/llm-verifier:$tag" .

    # Push to registry
    log_info "Pushing image to registry..."
    docker push "$registry/llm-verifier:$tag"

    log_success "Image built and pushed: $registry/llm-verifier:$tag"
}

# Deploy to a specific region
deploy_to_region() {
    local region=$1
    local context=$2
    local registry=$3

    log_info "Deploying to region: $region"

    # Switch kubectl context
    kubectl config use-context "$context"

    # Create namespace if it doesn't exist
    kubectl create namespace llm-verifier --dry-run=client -o yaml | kubectl apply -f -

    # Deploy regional secrets
    if [ -f "$SCRIPT_DIR/secrets-$region.yaml" ]; then
        kubectl apply -f "$SCRIPT_DIR/secrets-$region.yaml"
    else
        log_warning "Regional secrets file not found: $SCRIPT_DIR/secrets-$region.yaml"
    fi

    # Update deployment with region-specific config
    sed -e "s/\${REGION}/$region/g" \
        -e "s/\${CLUSTER_NAME}/$context/g" \
        -e "s|llm-verifier:latest|$registry/llm-verifier:latest|g" \
        "$SCRIPT_DIR/multi-region-deployment.yaml" | kubectl apply -f -

    # Wait for rollout
    kubectl rollout status deployment/llm-verifier -n llm-verifier --timeout=300s

    log_success "Deployment completed for region: $region"
}

# Setup global load balancer
setup_global_load_balancer() {
    local primary_context=$1

    log_info "Setting up global load balancer..."

    # Switch to primary region context
    kubectl config use-context "$primary_context"

    # Deploy global load balancer
    kubectl apply -f "$SCRIPT_DIR/global-load-balancer.yaml"

    # Wait for load balancer to get external IP
    log_info "Waiting for load balancer external IP..."
    local attempts=0
    local max_attempts=30

    while [ $attempts -lt $max_attempts ]; do
        EXTERNAL_IP=$(kubectl get svc llm-verifier -n llm-verifier -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
        if [ -n "$EXTERNAL_IP" ]; then
            log_success "Global load balancer ready with IP: $EXTERNAL_IP"
            echo "Global Load Balancer IP: $EXTERNAL_IP" > "$SCRIPT_DIR/global-ip.txt"
            return 0
        fi

        attempts=$((attempts + 1))
        log_info "Waiting for external IP (attempt $attempts/$max_attempts)..."
        sleep 10
    done

    log_error "Failed to get external IP for global load balancer"
    return 1
}

# Setup monitoring and observability
setup_monitoring() {
    log_info "Setting up monitoring and observability..."

    # Deploy Prometheus
    helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
    helm repo update

    helm upgrade --install prometheus prometheus-community/kube-prometheus-stack \
        --namespace monitoring \
        --create-namespace \
        --set grafana.enabled=true \
        --set prometheus.serviceMonitor.enabled=true

    # Deploy Jaeger for tracing
    helm repo add jaegertracing https://jaegertracing.github.io/helm-charts

    helm upgrade --install jaeger jaegertracing/jaeger \
        --namespace observability \
        --create-namespace \
        --set allInOne.enabled=true

    log_success "Monitoring setup completed"
}

# Setup service mesh (Istio)
setup_service_mesh() {
    log_info "Setting up service mesh..."

    # Install Istio
    if ! command -v istioctl &> /dev/null; then
        log_info "Installing istioctl..."
        curl -L https://istio.io/downloadIstio | sh -
        export PATH="$PATH:$HOME/.istioctl/bin"
    fi

    # Install Istio with minimal profile
    istioctl install --set profile=minimal -y

    # Enable Istio injection for namespace
    kubectl label namespace llm-verifier istio-injection=enabled --overwrite

    # Restart deployments to pick up injection
    kubectl rollout restart deployment llm-verifier -n llm-verifier

    log_success "Service mesh setup completed"
}

# Setup cross-region DNS
setup_cross_region_dns() {
    log_info "Setting up cross-region DNS..."

    # Install External-DNS
    helm repo add external-dns https://kubernetes-sigs.github.io/external-dns/
    helm repo update

    helm upgrade --install external-dns external-dns/external-dns \
        --namespace external-dns \
        --create-namespace \
        --set provider=aws \  # or gcp, azure
        --set aws.zoneType=public \
        --set domainFilters[0]=llm-verifier.com

    log_success "Cross-region DNS setup completed"
}

# Run health checks across all regions
run_health_checks() {
    log_info "Running health checks across all regions..."

    local failed_regions=()

    for region in "${REGIONS[@]}"; do
        log_info "Checking health in region: $region"

        # Get service endpoint for this region
        local endpoint=""
        case $region in
            "us-east-1")
                endpoint="http://us-east.llm-verifier.com/api/health"
                ;;
            "us-west-2")
                endpoint="http://us-west.llm-verifier.com/api/health"
                ;;
            "eu-west-1")
                endpoint="http://eu.llm-verifier.com/api/health"
                ;;
            "ap-southeast-1")
                endpoint="http://asia.llm-verifier.com/api/health"
                ;;
        esac

        if [ -n "$endpoint" ]; then
            if curl -f -s --max-time 10 "$endpoint" > /dev/null; then
                log_success "Health check passed for $region"
            else
                log_error "Health check failed for $region"
                failed_regions+=("$region")
            fi
        fi
    done

    if [ ${#failed_regions[@]} -gt 0 ]; then
        log_error "Health checks failed for regions: ${failed_regions[*]}"
        return 1
    fi

    log_success "All regional health checks passed"
}

# Main deployment function
main() {
    log_info "Starting LLM Verifier Multi-Region Deployment"

    # Parse command line arguments
    local cloud_provider="aws"  # default
    local registry=""
    local skip_monitoring=false
    local skip_service_mesh=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            --cloud-provider)
                cloud_provider="$2"
                shift 2
                ;;
            --registry)
                registry="$2"
                shift 2
                ;;
            --skip-monitoring)
                skip_monitoring=true
                shift
                ;;
            --skip-service-mesh)
                skip_service_mesh=true
                shift
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --cloud-provider PROVIDER    Cloud provider (aws, gcp, azure) [default: aws]"
                echo "  --registry REGISTRY          Docker registry URL"
                echo "  --skip-monitoring            Skip monitoring setup"
                echo "  --skip-service-mesh          Skip service mesh setup"
                echo "  --help                       Show this help"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    # Validate inputs
    if [ -z "$registry" ]; then
        log_error "Registry is required. Use --registry to specify Docker registry."
        exit 1
    fi

    # Run deployment steps
    check_prerequisites
    setup_cloud_cli "$cloud_provider"
    build_and_push_image "$registry"

    # Deploy to each region
    for region in "${REGIONS[@]}"; do
        # In a real setup, you'd have different kubectl contexts for each region
        local context="${cloud_provider}-${region}"
        deploy_to_region "$region" "$context" "$registry"
    done

    # Setup global infrastructure
    local primary_context="${cloud_provider}-${PRIMARY_REGION}"
    setup_global_load_balancer "$primary_context"

    if [ "$skip_monitoring" = false ]; then
        setup_monitoring
    fi

    if [ "$skip_service_mesh" = false ]; then
        setup_service_mesh
    fi

    setup_cross_region_dns
    run_health_checks

    log_success "Multi-region deployment completed successfully!"
    log_info "Global endpoint: https://api.llm-verifier.com"
    log_info "Monitoring: https://grafana.llm-verifier.com"
    log_info "Tracing: https://jaeger.llm-verifier.com"
}

# Run main function with all arguments
main "$@"