# Quick Reference: Kubernetes Deployment

## Prerequisites Installation

### macOS (Homebrew)
```bash
# Install kubectl
brew install kubectl

# Install Docker Desktop (includes Kubernetes)
brew install --cask docker

# Install KEDA
helm repo add kedacore https://kedacore.github.io/charts
helm repo update
helm install keda kedacore/keda --namespace keda --create-namespace

# Install NGINX Ingress
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.9.5/deploy/static/provider/cloud/deploy.yaml
```

### Linux
```bash
# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Enable Kubernetes in Docker Desktop or install Minikube
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube
minikube start
```

## Quick Deploy Commands

### Development (Local Cluster)
```bash
# Option 1: Use deployment script
export IMAGE_REGISTRY="docker.io/youruser"
./scripts/k8s-deploy.sh --env dev

# Option 2: Manual with local images (no registry push)
docker build -f deployments/docker/Dockerfile.api -t image-processor-api:latest .
docker build -f deployments/docker/Dockerfile.worker -t image-processor-worker:latest .
docker build -f deployments/docker/Dockerfile.frontend -t image-processor-frontend:latest .

# For Minikube: Load images into cluster
minikube image load image-processor-api:latest
minikube image load image-processor-worker:latest
minikube image load image-processor-frontend:latest

# For Kind: Load images into cluster
kind load docker-image image-processor-api:latest
kind load docker-image image-processor-worker:latest
kind load docker-image image-processor-frontend:latest

# Deploy
kubectl apply -k deployments/kubernetes/overlays/dev

# Access via port-forward
kubectl port-forward -n image-processor svc/frontend 8082:8082
kubectl port-forward -n image-processor svc/api 8080:8080
```

### Production (Cloud Cluster)
```bash
# Build and push to registry
export IMAGE_REGISTRY="gcr.io/my-project"  # or ghcr.io/username
export IMAGE_TAG="v1.0.0"

./scripts/k8s-deploy.sh --registry $IMAGE_REGISTRY --tag $IMAGE_TAG --env prod

# Or manually
docker build -f deployments/docker/Dockerfile.api -t ${IMAGE_REGISTRY}/image-processor-api:${IMAGE_TAG} .
docker push ${IMAGE_REGISTRY}/image-processor-api:${IMAGE_TAG}
# Repeat for worker and frontend...

kubectl apply -k deployments/kubernetes/overlays/prod
```

## Common Tasks

### Check Deployment Status
```bash
# All resources
kubectl get all -n image-processor

# Pods
kubectl get pods -n image-processor -o wide

# Logs
kubectl logs -n image-processor -l app=api --tail=100 -f
kubectl logs -n image-processor -l app=worker --tail=100 -f

# Events
kubectl get events -n image-processor --sort-by='.lastTimestamp'
```

### Access Application
```bash
# Get ingress IP
kubectl get ingress -n image-processor

# Port forwarding
kubectl port-forward -n image-processor svc/api 8080:8080
kubectl port-forward -n image-processor svc/frontend 8082:8082

# Test API
curl http://localhost:8080/api/v1/health

# Test metrics
curl http://localhost:8080/metrics
```

### Database Operations
```bash
# Connect to PostgreSQL
kubectl exec -it -n image-processor postgres-0 -- psql -U postgres imageprocessor

# Run migration manually
kubectl delete job db-migration -n image-processor
kubectl apply -k deployments/kubernetes/overlays/dev

# Backup database
kubectl exec -n image-processor postgres-0 -- \
  pg_dump -U postgres imageprocessor > backup.sql

# Restore database
kubectl exec -i -n image-processor postgres-0 -- \
  psql -U postgres imageprocessor < backup.sql
```

### Scaling
```bash
# Manual scale
kubectl scale deployment/api --replicas=5 -n image-processor

# Check HPA status
kubectl get hpa -n image-processor

# Check KEDA scaling
kubectl get scaledobject -n image-processor
kubectl describe scaledobject worker-scaler -n image-processor
```

### Update Images
```bash
# Set new image
kubectl set image deployment/api \
  api=image-processor-api:v1.1.0 \
  -n image-processor

# Rollout status
kubectl rollout status deployment/api -n image-processor

# Rollout history
kubectl rollout history deployment/api -n image-processor

# Rollback
kubectl rollout undo deployment/api -n image-processor
```

### Troubleshooting
```bash
# Describe pod for errors
kubectl describe pod -n image-processor <pod-name>

# Check logs of crashed pod
kubectl logs -n image-processor <pod-name> --previous

# Execute into pod
kubectl exec -it -n image-processor <pod-name> -- /bin/sh

# Check resource usage
kubectl top pods -n image-processor
kubectl top nodes

# Check KEDA operator
kubectl logs -n keda -l app=keda-operator

# Check ingress controller
kubectl logs -n ingress-nginx -l app.kubernetes.io/component=controller
```

### Cleanup
```bash
# Delete application
kubectl delete -k deployments/kubernetes/overlays/dev

# Delete namespace (removes everything)
kubectl delete namespace image-processor

# Delete PVCs (data loss!)
kubectl delete pvc --all -n image-processor
```

## Registry Setup

### Docker Hub
```bash
docker login
export IMAGE_REGISTRY="docker.io/yourusername"
```

### GitHub Container Registry
```bash
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin
export IMAGE_REGISTRY="ghcr.io/yourusername"
```

### Google Container Registry
```bash
gcloud auth configure-docker
export IMAGE_REGISTRY="gcr.io/your-project-id"
```

### AWS ECR
```bash
aws ecr get-login-password --region region | docker login --username AWS --password-stdin aws_account_id.dkr.ecr.region.amazonaws.com
export IMAGE_REGISTRY="aws_account_id.dkr.ecr.region.amazonaws.com"
```

## Environment Variables

Set these before deploying:
```bash
export IMAGE_REGISTRY="docker.io/youruser"  # Container registry
export IMAGE_TAG="v1.0.0"                   # Image version tag
export ENVIRONMENT="dev"                    # dev or prod
```

## File Locations

- **Deployment Script**: `scripts/k8s-deploy.sh`
- **Documentation**: `docs/KUBERNETES_DEPLOYMENT.md`
- **Base Manifests**: `deployments/kubernetes/base/`
- **Dev Overlay**: `deployments/kubernetes/overlays/dev/`
- **Prod Overlay**: `deployments/kubernetes/overlays/prod/`
- **Migrations**: `migrations/`
- **Dockerfiles**: `deployments/docker/`
