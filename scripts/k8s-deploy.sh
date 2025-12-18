#!/bin/bash

# Kubernetes Deployment Script for Image Processor
# Usage: ./scripts/k8s-deploy.sh [OPTIONS]
# 
# Options:
#   --registry <registry>    Container registry (default: docker.io/yourusername)
#   --tag <tag>              Image tag (default: latest)
#   --env <environment>      Environment: dev or prod (default: dev)
#   --skip-build            Skip building images
#   --skip-push             Skip pushing images
#   --skip-deploy           Skip Kubernetes deployment
#   --dry-run               Show commands without executing

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default configuration
IMAGE_REGISTRY="${IMAGE_REGISTRY:-docker.io/yourusername}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
ENVIRONMENT="${ENVIRONMENT:-dev}"
SKIP_BUILD=false
SKIP_PUSH=false
SKIP_DEPLOY=false
DRY_RUN=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --registry)
            IMAGE_REGISTRY="$2"
            shift 2
            ;;
        --tag)
            IMAGE_TAG="$2"
            shift 2
            ;;
        --env)
            ENVIRONMENT="$2"
            shift 2
            ;;
        --skip-build)
            SKIP_BUILD=true
            shift
            ;;
        --skip-push)
            SKIP_PUSH=true
            shift
            ;;
        --skip-deploy)
            SKIP_DEPLOY=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --help)
            echo "Kubernetes Deployment Script for Image Processor"
            echo ""
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --registry <registry>    Container registry (default: docker.io/yourusername)"
            echo "  --tag <tag>              Image tag (default: latest)"
            echo "  --env <environment>      Environment: dev or prod (default: dev)"
            echo "  --skip-build            Skip building images"
            echo "  --skip-push             Skip pushing images"
            echo "  --skip-deploy           Skip Kubernetes deployment"
            echo "  --dry-run               Show commands without executing"
            echo "  --help                  Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0 --registry gcr.io/my-project --tag v1.0.0 --env prod"
            echo "  $0 --env dev --skip-push"
            echo "  $0 --dry-run"
            exit 0
            ;;
        *)
            echo -e "${RED}Error: Unknown option $1${NC}"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Validate environment
if [[ "$ENVIRONMENT" != "dev" && "$ENVIRONMENT" != "prod" ]]; then
    echo -e "${RED}Error: Environment must be 'dev' or 'prod'${NC}"
    exit 1
fi

# Function to execute or print command
run_cmd() {
    if [ "$DRY_RUN" = true ]; then
        echo -e "${BLUE}[DRY-RUN]${NC} $*"
    else
        echo -e "${BLUE}[EXEC]${NC} $*"
        eval "$@"
    fi
}

# Print configuration
echo -e "${GREEN}═══════════════════════════════════════════════════${NC}"
echo -e "${GREEN}Kubernetes Deployment Configuration${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════${NC}"
echo -e "Registry:     ${YELLOW}${IMAGE_REGISTRY}${NC}"
echo -e "Tag:          ${YELLOW}${IMAGE_TAG}${NC}"
echo -e "Environment:  ${YELLOW}${ENVIRONMENT}${NC}"
echo -e "Skip Build:   ${YELLOW}${SKIP_BUILD}${NC}"
echo -e "Skip Push:    ${YELLOW}${SKIP_PUSH}${NC}"
echo -e "Skip Deploy:  ${YELLOW}${SKIP_DEPLOY}${NC}"
echo -e "Dry Run:      ${YELLOW}${DRY_RUN}${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════${NC}"
echo ""

# Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"

if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: docker is not installed${NC}"
    exit 1
fi

if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}Error: kubectl is not installed${NC}"
    exit 1
fi

# Check kubectl connection
if ! kubectl cluster-info &> /dev/null; then
    echo -e "${RED}Error: kubectl cannot connect to cluster${NC}"
    echo "Please configure kubectl with: kubectl config use-context <context>"
    exit 1
fi

echo -e "${GREEN}✓ Prerequisites check passed${NC}"
echo ""

# Build images
if [ "$SKIP_BUILD" = false ]; then
    echo -e "${YELLOW}Building Docker images...${NC}"
    
    API_IMAGE="${IMAGE_REGISTRY}/image-processor-api:${IMAGE_TAG}"
    WORKER_IMAGE="${IMAGE_REGISTRY}/image-processor-worker:${IMAGE_TAG}"
    FRONTEND_IMAGE="${IMAGE_REGISTRY}/image-processor-frontend:${IMAGE_TAG}"
    
    run_cmd "docker build -f deployments/docker/Dockerfile.api -t ${API_IMAGE} ."
    run_cmd "docker build -f deployments/docker/Dockerfile.worker -t ${WORKER_IMAGE} ."
    run_cmd "docker build -f deployments/docker/Dockerfile.frontend -t ${FRONTEND_IMAGE} ."
    
    # Tag as latest
    run_cmd "docker tag ${API_IMAGE} ${IMAGE_REGISTRY}/image-processor-api:latest"
    run_cmd "docker tag ${WORKER_IMAGE} ${IMAGE_REGISTRY}/image-processor-worker:latest"
    run_cmd "docker tag ${FRONTEND_IMAGE} ${IMAGE_REGISTRY}/image-processor-frontend:latest"
    
    echo -e "${GREEN}✓ Images built successfully${NC}"
    echo ""
else
    echo -e "${YELLOW}Skipping image build${NC}"
    echo ""
fi

# Push images
if [ "$SKIP_PUSH" = false ]; then
    echo -e "${YELLOW}Pushing Docker images to registry...${NC}"
    
    # Check if logged in (for Docker Hub)
    if [[ "$IMAGE_REGISTRY" == *"docker.io"* ]]; then
        if ! docker info | grep -q "Username"; then
            echo -e "${YELLOW}Warning: Not logged in to Docker Hub${NC}"
            echo "Run: docker login"
            if [ "$DRY_RUN" = false ]; then
                read -p "Continue anyway? (y/n) " -n 1 -r
                echo
                if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                    exit 1
                fi
            fi
        fi
    fi
    
    API_IMAGE="${IMAGE_REGISTRY}/image-processor-api:${IMAGE_TAG}"
    WORKER_IMAGE="${IMAGE_REGISTRY}/image-processor-worker:${IMAGE_TAG}"
    FRONTEND_IMAGE="${IMAGE_REGISTRY}/image-processor-frontend:${IMAGE_TAG}"
    
    run_cmd "docker push ${API_IMAGE}"
    run_cmd "docker push ${WORKER_IMAGE}"
    run_cmd "docker push ${FRONTEND_IMAGE}"
    
    run_cmd "docker push ${IMAGE_REGISTRY}/image-processor-api:latest"
    run_cmd "docker push ${IMAGE_REGISTRY}/image-processor-worker:latest"
    run_cmd "docker push ${IMAGE_REGISTRY}/image-processor-frontend:latest"
    
    echo -e "${GREEN}✓ Images pushed successfully${NC}"
    echo ""
else
    echo -e "${YELLOW}Skipping image push${NC}"
    echo ""
fi

# Deploy to Kubernetes
if [ "$SKIP_DEPLOY" = false ]; then
    echo -e "${YELLOW}Deploying to Kubernetes (${ENVIRONMENT})...${NC}"
    
    # Create temporary kustomization patch for images
    OVERLAY_DIR="deployments/kubernetes/overlays/${ENVIRONMENT}"
    IMAGES_PATCH="${OVERLAY_DIR}/images-patch.yaml"
    
    if [ "$DRY_RUN" = false ]; then
        cat > "${IMAGES_PATCH}" <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

images:
  - name: image-processor-api
    newName: ${IMAGE_REGISTRY}/image-processor-api
    newTag: ${IMAGE_TAG}
  - name: image-processor-worker
    newName: ${IMAGE_REGISTRY}/image-processor-worker
    newTag: ${IMAGE_TAG}
  - name: image-processor-frontend
    newName: ${IMAGE_REGISTRY}/image-processor-frontend
    newTag: ${IMAGE_TAG}
EOF
        echo -e "${GREEN}✓ Created image patch: ${IMAGES_PATCH}${NC}"
        
        # Add to kustomization if not already present
        if ! grep -q "images-patch.yaml" "${OVERLAY_DIR}/kustomization.yaml"; then
            echo "  - images-patch.yaml" >> "${OVERLAY_DIR}/kustomization.yaml"
            echo -e "${GREEN}✓ Added images-patch.yaml to kustomization${NC}"
        fi
    else
        echo -e "${BLUE}[DRY-RUN]${NC} Would create ${IMAGES_PATCH}"
    fi
    
    # Apply kustomization
    echo ""
    echo -e "${YELLOW}Applying Kubernetes manifests...${NC}"
    run_cmd "kubectl apply -k ${OVERLAY_DIR}"
    
    if [ "$DRY_RUN" = false ]; then
        echo ""
        echo -e "${YELLOW}Waiting for deployments to be ready...${NC}"
        
        # Wait for namespace to be created
        kubectl wait --for=jsonpath='{.status.phase}'=Active namespace/image-processor --timeout=30s || true
        
        # Wait for jobs to complete
        echo -e "${YELLOW}Waiting for initialization jobs...${NC}"
        kubectl wait --for=condition=complete job/minio-init -n image-processor --timeout=120s || \
            echo -e "${YELLOW}Warning: MinIO init job did not complete in time${NC}"
        kubectl wait --for=condition=complete job/db-migration -n image-processor --timeout=120s || \
            echo -e "${YELLOW}Warning: DB migration job did not complete in time${NC}"
        
        # Wait for deployments
        echo -e "${YELLOW}Waiting for deployments...${NC}"
        kubectl rollout status statefulset/postgres -n image-processor --timeout=300s
        kubectl rollout status deployment/redis -n image-processor --timeout=300s
        kubectl rollout status deployment/minio -n image-processor --timeout=300s
        kubectl rollout status deployment/api -n image-processor --timeout=300s
        kubectl rollout status deployment/worker -n image-processor --timeout=300s
        kubectl rollout status deployment/frontend -n image-processor --timeout=300s
    fi
    
    echo -e "${GREEN}✓ Deployment completed${NC}"
    echo ""
else
    echo -e "${YELLOW}Skipping Kubernetes deployment${NC}"
    echo ""
fi

# Show deployment status
if [ "$DRY_RUN" = false ] && [ "$SKIP_DEPLOY" = false ]; then
    echo -e "${GREEN}═══════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}Deployment Status${NC}"
    echo -e "${GREEN}═══════════════════════════════════════════════════${NC}"
    
    echo ""
    echo -e "${YELLOW}Pods:${NC}"
    kubectl get pods -n image-processor
    
    echo ""
    echo -e "${YELLOW}Services:${NC}"
    kubectl get svc -n image-processor
    
    echo ""
    echo -e "${YELLOW}Ingress:${NC}"
    kubectl get ingress -n image-processor
    
    echo ""
    echo -e "${YELLOW}HPA:${NC}"
    kubectl get hpa -n image-processor
    
    echo ""
    echo -e "${YELLOW}ScaledObject:${NC}"
    kubectl get scaledobject -n image-processor
    
    echo ""
    echo -e "${GREEN}═══════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}Deployment Complete!${NC}"
    echo -e "${GREEN}═══════════════════════════════════════════════════${NC}"
    
    echo ""
    echo -e "${YELLOW}Access the application:${NC}"
    echo ""
    echo "1. Add to /etc/hosts (for local clusters):"
    echo "   kubectl get ingress -n image-processor -o jsonpath='{.items[0].status.loadBalancer.ingress[0].ip}' | xargs -I {} echo {} image-processor.local | sudo tee -a /etc/hosts"
    echo ""
    echo "2. Or use port forwarding:"
    echo "   kubectl port-forward -n image-processor svc/frontend 8082:8082"
    echo "   kubectl port-forward -n image-processor svc/api 8080:8080"
    echo ""
    echo "3. Access endpoints:"
    echo "   Frontend: http://image-processor.local or http://localhost:8082"
    echo "   API:      http://image-processor.local/api/v1/health or http://localhost:8080/api/v1/health"
    echo "   Metrics:  http://localhost:8080/metrics"
    echo ""
    echo -e "${YELLOW}View logs:${NC}"
    echo "   kubectl logs -n image-processor -l app=api -f"
    echo "   kubectl logs -n image-processor -l app=worker -f"
    echo ""
    echo -e "${YELLOW}Cleanup:${NC}"
    echo "   kubectl delete -k ${OVERLAY_DIR}"
    echo "   kubectl delete namespace image-processor"
    echo ""
fi

echo -e "${GREEN}✓ Script completed successfully${NC}"
