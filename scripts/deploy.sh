#!/bin/bash

set -euo pipefail

# Game Development VCS Deployment Script
# Supports both local development and production Kubernetes deployment

VERSION=${VERSION:-1.0.0}
ENVIRONMENT=${ENVIRONMENT:-development}
NAMESPACE=${NAMESPACE:-vcs-system}
REGISTRY=${REGISTRY:-yourstudio}

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

# Help function
show_help() {
    cat << EOF
Game Development VCS Deployment Script

Usage: $0 [COMMAND] [OPTIONS]

Commands:
  dev                 Start development environment with Docker Compose
  build               Build Docker images
  push                Push Docker images to registry
  deploy              Deploy to Kubernetes (production)
  destroy             Destroy deployment
  status              Show deployment status
  logs                Show logs
  backup              Create backup
  restore             Restore from backup

Options:
  -e, --environment   Environment (development, staging, production) [default: development]
  -n, --namespace     Kubernetes namespace [default: vcs-system]
  -r, --registry      Docker registry [default: yourstudio]
  -v, --version       Version tag [default: 1.0.0]
  -h, --help          Show this help

Examples:
  $0 dev                          # Start local development
  $0 build                        # Build Docker images
  $0 deploy -e production         # Deploy to production
  $0 status -n vcs-system         # Check status
  $0 logs vcs-api                 # Show API logs

EOF
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    local missing_tools=()
    
    if ! command -v docker &> /dev/null; then
        missing_tools+=("docker")
    fi
    
    if ! command -v docker-compose &> /dev/null && [ "$ENVIRONMENT" == "development" ]; then
        missing_tools+=("docker-compose")
    fi
    
    if ! command -v kubectl &> /dev/null && [ "$ENVIRONMENT" != "development" ]; then
        missing_tools+=("kubectl")
    fi
    
    if ! command -v make &> /dev/null; then
        missing_tools+=("make")
    fi
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        log_error "Please install the missing tools and try again."
        exit 1
    fi
    
    log_success "All prerequisites satisfied"
}

# Development environment
start_dev() {
    log_info "Starting development environment..."
    
    # Create necessary directories
    mkdir -p data/{redis,clickhouse,storage,logs}
    
    # Start services with Docker Compose
    docker-compose -f docker/docker-compose.dev.yml up -d
    
    log_success "Development environment started!"
    log_info "Services available at:"
    log_info "  • VCS API:      http://localhost:8080"
    log_info "  • ClickHouse:   http://localhost:8123"
    log_info "  • Redis:        localhost:6379"
    log_info "  • Adminer:      http://localhost:8081"
    log_info "  • Redis GUI:    http://localhost:8082"
    
    log_info "To view logs: docker-compose -f docker/docker-compose.dev.yml logs -f"
    log_info "To stop: docker-compose -f docker/docker-compose.dev.yml down"
}

# Build Docker images
build_images() {
    log_info "Building Docker images..."
    
    # Build server image
    log_info "Building VCS server image..."
    docker build -f docker/Dockerfile.server -t ${REGISTRY}/vcs-server:${VERSION} .
    docker tag ${REGISTRY}/vcs-server:${VERSION} ${REGISTRY}/vcs-server:latest
    
    # Build CLI image
    log_info "Building VCS CLI image..."
    docker build -f docker/Dockerfile.cli -t ${REGISTRY}/vcs-cli:${VERSION} .
    docker tag ${REGISTRY}/vcs-cli:${VERSION} ${REGISTRY}/vcs-cli:latest
    
    log_success "Docker images built successfully!"
    log_info "Images:"
    log_info "  • ${REGISTRY}/vcs-server:${VERSION}"
    log_info "  • ${REGISTRY}/vcs-cli:${VERSION}"
}

# Push images to registry
push_images() {
    log_info "Pushing images to registry..."
    
    docker push ${REGISTRY}/vcs-server:${VERSION}
    docker push ${REGISTRY}/vcs-server:latest
    docker push ${REGISTRY}/vcs-cli:${VERSION}
    docker push ${REGISTRY}/vcs-cli:latest
    
    log_success "Images pushed to registry!"
}

# Deploy to Kubernetes
deploy_k8s() {
    log_info "Deploying to Kubernetes (${ENVIRONMENT})..."
    
    # Check if kubectl is configured
    if ! kubectl cluster-info &> /dev/null; then
        log_error "kubectl is not configured or cluster is not accessible"
        exit 1
    fi
    
    # Create namespace if it doesn't exist
    kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -
    
    # Apply configurations in order
    log_info "Applying namespace and configuration..."
    kubectl apply -f k8s/namespace.yaml
    kubectl apply -f k8s/configmap.yaml
    
    log_info "Deploying Redis..."
    kubectl apply -f k8s/redis.yaml
    
    log_info "Deploying ClickHouse..."
    kubectl apply -f k8s/clickhouse.yaml
    
    # Wait for databases to be ready
    log_info "Waiting for databases to be ready..."
    kubectl wait --for=condition=available --timeout=300s deployment/redis -n ${NAMESPACE}
    kubectl wait --for=condition=ready --timeout=300s statefulset/clickhouse -n ${NAMESPACE}
    
    log_info "Deploying VCS API..."
    # Update image tags in the deployment
    sed "s|yourstudio/vcs-server:1.0.0|${REGISTRY}/vcs-server:${VERSION}|g" k8s/vcs-api.yaml | kubectl apply -f -
    
    log_info "Deploying Ingress..."
    kubectl apply -f k8s/ingress.yaml
    
    # Wait for deployment to be ready
    kubectl wait --for=condition=available --timeout=300s deployment/vcs-api -n ${NAMESPACE}
    
    log_success "Deployment completed!"
    show_status
}

# Show deployment status
show_status() {
    if [ "$ENVIRONMENT" == "development" ]; then
        log_info "Development environment status:"
        docker-compose -f docker/docker-compose.dev.yml ps
    else
        log_info "Kubernetes deployment status:"
        kubectl get all -n ${NAMESPACE}
        echo
        log_info "Ingress status:"
        kubectl get ingress -n ${NAMESPACE}
        echo
        log_info "Storage status:"
        kubectl get pvc -n ${NAMESPACE}
    fi
}

# Show logs
show_logs() {
    local service=${1:-vcs-api}
    
    if [ "$ENVIRONMENT" == "development" ]; then
        docker-compose -f docker/docker-compose.dev.yml logs -f ${service}
    else
        kubectl logs -f deployment/${service} -n ${NAMESPACE}
    fi
}

# Destroy deployment
destroy_deployment() {
    log_warning "This will destroy the entire deployment. Are you sure? (y/N)"
    read -r confirmation
    
    if [ "$confirmation" != "y" ] && [ "$confirmation" != "Y" ]; then
        log_info "Aborted."
        exit 0
    fi
    
    if [ "$ENVIRONMENT" == "development" ]; then
        log_info "Stopping development environment..."
        docker-compose -f docker/docker-compose.dev.yml down -v
        docker system prune -f
    else
        log_info "Destroying Kubernetes deployment..."
        kubectl delete namespace ${NAMESPACE}
    fi
    
    log_success "Deployment destroyed!"
}

# Create backup
create_backup() {
    local backup_name="vcs-backup-$(date +%Y%m%d-%H%M%S)"
    
    log_info "Creating backup: ${backup_name}"
    
    if [ "$ENVIRONMENT" == "development" ]; then
        # Backup development data
        mkdir -p backups/${backup_name}
        docker-compose -f docker/docker-compose.dev.yml exec -T redis redis-cli BGSAVE
        docker cp vcs-redis-dev:/data backups/${backup_name}/redis
        docker cp vcs-clickhouse-dev:/var/lib/clickhouse backups/${backup_name}/clickhouse
        tar -czf backups/${backup_name}.tar.gz -C backups ${backup_name}
        rm -rf backups/${backup_name}
    else
        # Backup Kubernetes data
        kubectl exec -n ${NAMESPACE} deployment/redis -- redis-cli BGSAVE
        # TODO: Implement ClickHouse backup and S3 upload
    fi
    
    log_success "Backup created: ${backup_name}"
}

# Main script logic
main() {
    local command=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            dev|build|push|deploy|destroy|status|logs|backup|restore)
                command="$1"
                shift
                ;;
            -e|--environment)
                ENVIRONMENT="$2"
                shift 2
                ;;
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            -r|--registry)
                REGISTRY="$2"
                shift 2
                ;;
            -v|--version)
                VERSION="$2"
                shift 2
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                if [ -z "$command" ]; then
                    log_error "Unknown command: $1"
                    show_help
                    exit 1
                else
                    # This might be an argument for the command
                    break
                fi
                ;;
        esac
    done
    
    if [ -z "$command" ]; then
        log_error "No command specified"
        show_help
        exit 1
    fi
    
    # Set environment-specific defaults
    if [ "$ENVIRONMENT" != "development" ]; then
        export KUBECONFIG=${KUBECONFIG:-~/.kube/config}
    fi
    
    log_info "Environment: ${ENVIRONMENT}"
    log_info "Version: ${VERSION}"
    log_info "Registry: ${REGISTRY}"
    
    if [ "$ENVIRONMENT" != "development" ]; then
        log_info "Namespace: ${NAMESPACE}"
    fi
    
    echo
    
    check_prerequisites
    
    # Execute command
    case $command in
        dev)
            start_dev
            ;;
        build)
            build_images
            ;;
        push)
            push_images
            ;;
        deploy)
            if [ "$ENVIRONMENT" == "development" ]; then
                start_dev
            else
                deploy_k8s
            fi
            ;;
        destroy)
            destroy_deployment
            ;;
        status)
            show_status
            ;;
        logs)
            show_logs "$@"
            ;;
        backup)
            create_backup
            ;;
        restore)
            log_error "Restore functionality not yet implemented"
            exit 1
            ;;
        *)
            log_error "Unknown command: $command"
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"