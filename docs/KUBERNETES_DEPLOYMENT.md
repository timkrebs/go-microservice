# Kubernetes Deployment Guide

This guide provides comprehensive instructions for deploying the Image Processor microservice to a Kubernetes cluster.

## Architecture Overview

The application consists of:
- **API Service**: REST API for job submission (Go, port 8080)
- **Worker Service**: Background image processing (Go)
- **Frontend Service**: Web UI for job management (Go, port 8082)
- **PostgreSQL**: Job metadata storage (StatefulSet)
- **Redis**: Job queue via streams (Deployment)
- **MinIO**: S3-compatible object storage (Deployment)

## Prerequisites

### Required Components

1. **Kubernetes Cluster** (v1.24+)
   - Local: Minikube, Kind, Docker Desktop
   - Cloud: GKE, EKS, AKS, DigitalOcean

2. **kubectl** configured with cluster access
   ```bash
   kubectl version --client
   kubectl cluster-info
   ```

3. **NGINX Ingress Controller**
   ```bash
   kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.9.5/deploy/static/provider/cloud/deploy.yaml
   ```

4. **KEDA (Kubernetes Event-Driven Autoscaler)** v2.12+
   ```bash
   kubectl apply -f https://github.com/kedacore/keda/releases/download/v2.12.0/keda-2.12.0.yaml
   ```

5. **Metrics Server** (for HPA)
   ```bash
   kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml
   ```

6. **Container Registry** (one of):
   - Docker Hub: `docker.io/<username>`
   - GitHub Container Registry: `ghcr.io/<username>`
   - Google Container Registry: `gcr.io/<project-id>`
   - AWS ECR: `<account>.dkr.ecr.<region>.amazonaws.com`
   - Self-hosted: `registry.example.com`

7. **Docker** for building images
   ```bash
   docker version
   ```

### Verify Prerequisites

```bash
# Check KEDA installation
kubectl get deployment -n keda

# Check NGINX Ingress
kubectl get pods -n ingress-nginx

# Check Metrics Server
kubectl get deployment metrics-server -n kube-system

# Check default StorageClass
kubectl get storageclass
```

## Quick Start

### Option 1: Automated Deployment Script

```bash
# Configure registry (edit script or set environment variable)
export IMAGE_REGISTRY="docker.io/yourusername"
export IMAGE_TAG="v1.0.0"
export ENVIRONMENT="dev"  # or "prod"

# Run deployment script
./scripts/k8s-deploy.sh
```

### Option 2: Manual Deployment

Follow the detailed steps below for full control over the deployment process.

## Detailed Deployment Steps

### 1. Configure Container Registry

#### Option A: Docker Hub
```bash
# Login to Docker Hub
docker login

# Set registry variables
export IMAGE_REGISTRY="docker.io/yourusername"
export IMAGE_TAG="v1.0.0"
```

#### Option B: GitHub Container Registry
```bash
# Login to GHCR
echo $GITHUB_TOKEN | docker login ghcr.io -u USERNAME --password-stdin

# Set registry variables
export IMAGE_REGISTRY="ghcr.io/yourusername"
export IMAGE_TAG="v1.0.0"
```

#### Option C: Google Container Registry
```bash
# Configure Docker for GCR
gcloud auth configure-docker

# Set registry variables
export IMAGE_REGISTRY="gcr.io/your-project-id"
export IMAGE_TAG="v1.0.0"
```

### 2. Build and Push Docker Images

```bash
# Build images
docker build -f deployments/docker/Dockerfile.api \
  -t ${IMAGE_REGISTRY}/image-processor-api:${IMAGE_TAG} .

docker build -f deployments/docker/Dockerfile.worker \
  -t ${IMAGE_REGISTRY}/image-processor-worker:${IMAGE_TAG} .

docker build -f deployments/docker/Dockerfile.frontend \
  -t ${IMAGE_REGISTRY}/image-processor-frontend:${IMAGE_TAG} .

# Push images
docker push ${IMAGE_REGISTRY}/image-processor-api:${IMAGE_TAG}
docker push ${IMAGE_REGISTRY}/image-processor-worker:${IMAGE_TAG}
docker push ${IMAGE_REGISTRY}/image-processor-frontend:${IMAGE_TAG}

# Tag as latest (optional)
docker tag ${IMAGE_REGISTRY}/image-processor-api:${IMAGE_TAG} ${IMAGE_REGISTRY}/image-processor-api:latest
docker push ${IMAGE_REGISTRY}/image-processor-api:latest
```

### 3. Update Kubernetes Manifests

Create a kustomization patch to override image names:

```bash
cat > deployments/kubernetes/overlays/dev/images.yaml <<EOF
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

# Add to kustomization.yaml
echo "  - images.yaml" >> deployments/kubernetes/overlays/dev/kustomization.yaml
```

### 4. Configure Secrets

**CRITICAL**: Update production secrets before deploying.

```bash
# Generate secure passwords
POSTGRES_PASSWORD=$(openssl rand -base64 32)
MINIO_ACCESS_KEY=$(openssl rand -base64 16)
MINIO_SECRET_KEY=$(openssl rand -base64 32)

# Create secret (recommended over editing YAML)
kubectl create secret generic app-secrets \
  --from-literal=POSTGRES_PASSWORD=${POSTGRES_PASSWORD} \
  --from-literal=MINIO_ACCESS_KEY=${MINIO_ACCESS_KEY} \
  --from-literal=MINIO_SECRET_KEY=${MINIO_SECRET_KEY} \
  --from-literal=REDIS_PASSWORD="" \
  --namespace=image-processor \
  --dry-run=client -o yaml > deployments/kubernetes/overlays/prod/secrets.yaml
```

For production, consider using:
- [Sealed Secrets](https://github.com/bitnami-labs/sealed-secrets)
- [External Secrets Operator](https://external-secrets.io/)
- Cloud provider secret managers (AWS Secrets Manager, GCP Secret Manager, Azure Key Vault)

### 5. Deploy to Kubernetes

#### Development Environment
```bash
# Deploy all resources
kubectl apply -k deployments/kubernetes/overlays/dev

# Watch deployment progress
kubectl get pods -n image-processor -w
```

#### Production Environment
```bash
# Deploy all resources
kubectl apply -k deployments/kubernetes/overlays/prod

# Monitor rollout
kubectl rollout status deployment/api -n image-processor
kubectl rollout status deployment/worker -n image-processor
kubectl rollout status deployment/frontend -n image-processor
```

### 6. Verify Deployment

```bash
# Check all pods are running
kubectl get pods -n image-processor

# Check services
kubectl get svc -n image-processor

# Check ingress
kubectl get ingress -n image-processor

# Check HPA
kubectl get hpa -n image-processor

# Check KEDA ScaledObject
kubectl get scaledobject -n image-processor

# View logs
kubectl logs -n image-processor -l app=api --tail=50
kubectl logs -n image-processor -l app=worker --tail=50
```

### 7. Database Migrations

Migrations run automatically via Kubernetes Job during deployment. To run manually:

```bash
# Check migration job status
kubectl get jobs -n image-processor

# View migration logs
kubectl logs -n image-processor job/db-migration

# Manual migration (if needed)
kubectl delete job db-migration -n image-processor
kubectl apply -k deployments/kubernetes/overlays/dev
```

### 8. Access the Application

#### Local Cluster (Minikube/Kind)
```bash
# Add to /etc/hosts
echo "$(kubectl get ingress -n image-processor image-processor-ingress -o jsonpath='{.status.loadBalancer.ingress[0].ip}') image-processor.local" | sudo tee -a /etc/hosts

# Or use port forwarding
kubectl port-forward -n image-processor svc/api 8080:8080
kubectl port-forward -n image-processor svc/frontend 8082:8082

# Access application
curl http://localhost:8080/api/v1/health
open http://localhost:8082
```

#### Cloud Cluster
```bash
# Get LoadBalancer IP
kubectl get ingress -n image-processor

# Configure DNS A record
# image-processor.yourdomain.com -> <EXTERNAL-IP>

# Access application
curl http://image-processor.yourdomain.com/api/v1/health
open http://image-processor.yourdomain.com
```

## Configuration

### Environment-Specific Settings

The deployment uses Kustomize overlays for environment-specific configuration:

| Setting | Dev | Prod |
|---------|-----|------|
| API Replicas | 1 | 3 |
| API HPA Max | 3 | 20 |
| Worker HPA Max | 5 | 50 |
| CPU Request (API) | 200m | 250m |
| Memory Request (API) | 256Mi | 256Mi |
| CPU Limit (Worker) | 1000m | 2000m |
| Memory Limit (Worker) | 512Mi | 2Gi |
| MinIO Storage | 20Gi | 100Gi |

### Scaling Behavior

**API Service** (HPA):
- Scales based on CPU (70%) and Memory (80%)
- Development: 1-3 replicas
- Production: 3-20 replicas

**Worker Service** (KEDA):
- Scales based on Redis Stream lag
- Trigger: `image-jobs` stream, lag threshold = 5
- Development: 0-5 replicas
- Production: 2-50 replicas
- Polling interval: 15s, cooldown: 60s

### Resource Requests and Limits

```yaml
# API (Production)
resources:
  requests:
    cpu: 250m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 1Gi

# Worker (Production)
resources:
  requests:
    cpu: 500m
    memory: 512Mi
  limits:
    cpu: 2000m
    memory: 2Gi
```

## Monitoring and Observability

### Prometheus Metrics

The API service exposes Prometheus metrics at `/metrics`:

```bash
# Port forward to access metrics
kubectl port-forward -n image-processor svc/api 8080:8080
curl http://localhost:8080/metrics
```

Key metrics:
- `image_processor_api_http_requests_total` - Total HTTP requests
- `image_processor_api_http_request_duration_seconds` - Request latency
- `image_processor_api_jobs_total` - Job processing metrics
- `image_processor_api_storage_operations_total` - Storage operations
- `image_processor_api_db_queries_total` - Database query performance

### Health Checks

```bash
# API health check
kubectl exec -n image-processor deployment/api -- \
  curl -s http://localhost:8080/api/v1/health | jq

# Expected response
{
  "status": "healthy",
  "checks": {
    "database": {
      "status": "healthy",
      "open_connections": 1,
      "in_use": 0,
      "idle": 1
    },
    "redis": {
      "status": "healthy"
    },
    "storage": {
      "status": "healthy"
    }
  }
}
```

### Logs

```bash
# Stream API logs
kubectl logs -n image-processor -l app=api -f --tail=100

# Stream worker logs
kubectl logs -n image-processor -l app=worker -f --tail=100

# Query specific pod logs
kubectl logs -n image-processor <pod-name> --previous
```

## Troubleshooting

### Pods Not Starting

```bash
# Check pod status
kubectl describe pod -n image-processor <pod-name>

# Common issues:
# 1. Image pull errors - verify registry access
kubectl get events -n image-processor --sort-by='.lastTimestamp'

# 2. Resource constraints - check node capacity
kubectl describe nodes

# 3. PVC pending - check StorageClass
kubectl get pvc -n image-processor
kubectl get storageclass
```

### Database Connection Issues

```bash
# Check PostgreSQL pod
kubectl logs -n image-processor postgres-0

# Test connection from API pod
kubectl exec -it -n image-processor deployment/api -- \
  psql "postgres://postgres:postgres@postgres:5432/imageprocessor?sslmode=disable"

# Check service endpoints
kubectl get endpoints -n image-processor postgres
```

### Worker Not Processing Jobs

```bash
# Check KEDA operator
kubectl get pods -n keda

# Check ScaledObject status
kubectl describe scaledobject -n image-processor worker-scaler

# Check Redis connection
kubectl exec -it -n image-processor deployment/redis -- redis-cli
> XINFO STREAM image-jobs
> XINFO GROUPS image-jobs

# Check worker logs
kubectl logs -n image-processor -l app=worker --tail=100
```

### Ingress Not Accessible

```bash
# Check ingress controller
kubectl get pods -n ingress-nginx

# Check ingress configuration
kubectl describe ingress -n image-processor

# Check service endpoints
kubectl get endpoints -n image-processor api

# Test service directly
kubectl port-forward -n image-processor svc/api 8080:8080
curl http://localhost:8080/api/v1/health
```

### Image Pull Errors

```bash
# Verify image exists
docker pull ${IMAGE_REGISTRY}/image-processor-api:${IMAGE_TAG}

# Create image pull secret (for private registries)
kubectl create secret docker-registry regcred \
  --docker-server=${IMAGE_REGISTRY} \
  --docker-username=${REGISTRY_USERNAME} \
  --docker-password=${REGISTRY_PASSWORD} \
  --namespace=image-processor

# Add to deployment
kubectl patch serviceaccount default \
  -p '{"imagePullSecrets": [{"name": "regcred"}]}' \
  -n image-processor
```

### Storage Issues

```bash
# Check MinIO pod
kubectl logs -n image-processor deployment/minio

# Check bucket initialization
kubectl logs -n image-processor job/minio-init

# Test MinIO access
kubectl port-forward -n image-processor svc/minio 9000:9000
# Access console at http://localhost:9000
```

## Maintenance

### Rolling Updates

```bash
# Update image version
kubectl set image deployment/api \
  api=${IMAGE_REGISTRY}/image-processor-api:v1.1.0 \
  -n image-processor

# Monitor rollout
kubectl rollout status deployment/api -n image-processor

# Rollback if needed
kubectl rollout undo deployment/api -n image-processor
```

### Database Backups

```bash
# Backup PostgreSQL
kubectl exec -n image-processor postgres-0 -- \
  pg_dump -U postgres imageprocessor > backup-$(date +%Y%m%d).sql

# Restore
kubectl exec -i -n image-processor postgres-0 -- \
  psql -U postgres imageprocessor < backup-20251218.sql
```

### Scaling

```bash
# Manual scale (temporary)
kubectl scale deployment/api --replicas=5 -n image-processor

# Update HPA limits
kubectl edit hpa/api -n image-processor

# Update KEDA limits
kubectl edit scaledobject/worker-scaler -n image-processor
```

### Cleanup

```bash
# Delete application
kubectl delete -k deployments/kubernetes/overlays/dev

# Delete namespace (removes all resources)
kubectl delete namespace image-processor

# Delete PVCs (data loss!)
kubectl delete pvc --all -n image-processor
```

## Security Considerations

### Production Checklist

- [ ] Use secure passwords for PostgreSQL, MinIO, Redis
- [ ] Enable TLS/SSL on ingress with cert-manager
- [ ] Use Pod Security Standards (restricted profile)
- [ ] Enable Network Policies to restrict pod communication
- [ ] Use separate service accounts for each component
- [ ] Enable RBAC with least privilege
- [ ] Scan images for vulnerabilities (Trivy, Snyk)
- [ ] Use private container registry
- [ ] Enable audit logging
- [ ] Implement resource quotas per namespace
- [ ] Use external secret management (not ConfigMaps)
- [ ] Enable PostgreSQL SSL/TLS
- [ ] Configure MinIO with TLS
- [ ] Implement Pod Disruption Budgets
- [ ] Use read-only root filesystems where possible

### TLS Configuration

```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# Create ClusterIssuer for Let's Encrypt
kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-prod
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    email: your-email@example.com
    privateKeySecretRef:
      name: letsencrypt-prod
    solvers:
    - http01:
        ingress:
          class: nginx
EOF

# Update ingress with TLS
kubectl annotate ingress image-processor-ingress \
  cert-manager.io/cluster-issuer=letsencrypt-prod \
  -n image-processor
```

## Performance Tuning

### PostgreSQL Optimization

```bash
# Increase connection pool (in configmap)
MAX_OPEN_CONNS=25
MAX_IDLE_CONNS=5

# Add PostgreSQL performance tuning
kubectl exec -n image-processor postgres-0 -- psql -U postgres -c "
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;
ALTER SYSTEM SET random_page_cost = 1.1;
ALTER SYSTEM SET effective_io_concurrency = 200;
"

# Restart PostgreSQL
kubectl rollout restart statefulset/postgres -n image-processor
```

### Redis Optimization

```bash
# Enable persistence (add to redis deployment)
# --appendonly yes --appendfsync everysec
```

### Worker Optimization

```bash
# Adjust KEDA trigger sensitivity
kubectl edit scaledobject worker-scaler -n image-processor

# Reduce lag threshold for faster scaling
lagThreshold: "3"  # Scale when 3+ jobs pending

# Adjust polling interval
pollingInterval: 10  # Check every 10 seconds
```

## Additional Resources

- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [KEDA Documentation](https://keda.sh/docs/)
- [Kustomize Documentation](https://kustomize.io/)
- [NGINX Ingress Controller](https://kubernetes.github.io/ingress-nginx/)
- [Prometheus Operator](https://prometheus-operator.dev/)

## Support

For issues specific to this deployment:
1. Check application logs: `kubectl logs -n image-processor -l app=<component>`
2. Review [TROUBLESHOOTING.md](./TROUBLESHOOTING.md)
3. Verify [PRODUCTION_ENHANCEMENTS.md](./PRODUCTION_ENHANCEMENTS.md) for metrics and health checks
4. Check database migrations in `migrations/` directory
