# Image Processing Microservices Platform

## Overview

A cloud-native, Kubernetes-ready image processing platform built with Go microservices. Users upload images for processing (resize, filters, thumbnails, watermarks), with automatic horizontal scaling based on workload.

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Kubernetes Cluster                              │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌─────────────┐      ┌─────────────┐      ┌─────────────────────────┐    │
│   │   Ingress   │─────▶│  Frontend   │      │      Worker Pool        │    │
│   │  (nginx)    │      │  (Go/htmx)  │      │  ┌───────┐ ┌───────┐   │    │
│   └─────────────┘      └──────┬──────┘      │  │Worker1│ │Worker2│   │    │
│                               │             │  └───┬───┘ └───┬───┘   │    │
│                               ▼             │      │         │       │    │
│                        ┌─────────────┐      │  ┌───┴─────────┴───┐   │    │
│                        │   API       │◀────▶│  │    WorkerN      │   │    │
│                        │  Gateway    │      │  └───────┬─────────┘   │    │
│                        │   (Go)      │      └──────────┼─────────────┘    │
│                        └──────┬──────┘                 │                  │
│                               │                        │                  │
│              ┌────────────────┼────────────────────────┤                  │
│              │                │                        │                  │
│              ▼                ▼                        ▼                  │
│       ┌─────────────┐  ┌─────────────┐         ┌─────────────┐           │
│       │ PostgreSQL  │  │    Redis    │         │    MinIO    │           │
│       │ (metadata)  │  │   (queue)   │         │  (storage)  │           │
│       └─────────────┘  └─────────────┘         └─────────────┘           │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Services

### 1. Frontend Service
- **Purpose:** Web UI for uploading images and viewing results
- **Tech:** Go + templ (type-safe templates) + htmx (reactive updates)
- **Port:** 8080
- **Scaling:** 2-5 replicas (CPU-based HPA)

### 2. API Gateway Service
- **Purpose:** REST API, job management, orchestration
- **Tech:** Go + chi router + structured logging
- **Port:** 8081
- **Scaling:** 2-10 replicas (CPU/request-based HPA)

### 3. Worker Service
- **Purpose:** Image processing (CPU-intensive)
- **Tech:** Go + imaging libraries (disintegration/imaging, nfnt/resize)
- **Port:** 8082 (health/metrics only)
- **Scaling:** 1-20 replicas (queue-depth-based HPA via KEDA)

### 4. PostgreSQL
- **Purpose:** Job metadata, user data, processing history
- **Scaling:** Single instance (or HA with Patroni for production)

### 5. Redis
- **Purpose:** Job queue, pub/sub for real-time updates, caching
- **Scaling:** Single instance (or Redis Cluster for production)

### 6. MinIO
- **Purpose:** S3-compatible object storage for images
- **Scaling:** Single instance (or distributed mode for production)

## Tech Stack

| Component       | Technology                    | Why                                      |
|-----------------|-------------------------------|------------------------------------------|
| Language        | Go 1.22+                      | Performance, concurrency, K8s native     |
| HTTP Router     | chi                           | Lightweight, idiomatic, middleware       |
| Templates       | templ                         | Type-safe, fast, Go-native               |
| Frontend UX     | htmx + Alpine.js              | Reactive without heavy JS frameworks     |
| CSS             | Tailwind CSS                  | Rapid styling, utility-first             |
| Image Processing| disintegration/imaging        | Pure Go, no CGO dependencies             |
| Queue           | Redis Streams                 | Reliable, supports consumer groups       |
| Database        | PostgreSQL 16                 | Robust, JSON support, battle-tested      |
| Object Storage  | MinIO                         | S3-compatible, K8s native                |
| Containers      | Docker + multi-stage builds   | Small images, security                   |
| Orchestration   | Kubernetes                    | Scaling, self-healing, declarative       |
| Autoscaling     | HPA + KEDA                    | Scale workers based on queue depth       |
| Observability   | Prometheus + Grafana          | Metrics, dashboards                      |
| Tracing         | OpenTelemetry                 | Distributed tracing across services      |

## Project Structure

```
image-processor/
├── cmd/                           # Application entrypoints
│   ├── frontend/
│   │   └── main.go
│   ├── api/
│   │   └── main.go
│   └── worker/
│       └── main.go
│
├── internal/                      # Private application code
│   ├── config/                    # Configuration loading
│   │   └── config.go
│   ├── models/                    # Domain models
│   │   ├── job.go
│   │   └── image.go
│   ├── queue/                     # Redis queue abstraction
│   │   ├── producer.go
│   │   └── consumer.go
│   ├── storage/                   # MinIO storage abstraction
│   │   └── minio.go
│   ├── database/                  # PostgreSQL repository
│   │   ├── db.go
│   │   └── jobs.go
│   ├── processor/                 # Image processing logic
│   │   ├── processor.go
│   │   ├── resize.go
│   │   ├── filters.go
│   │   ├── thumbnail.go
│   │   └── watermark.go
│   └── api/                       # HTTP handlers
│       ├── handlers.go
│       ├── middleware.go
│       └── routes.go
│
├── web/                           # Frontend assets
│   ├── templates/                 # templ templates
│   │   ├── layouts/
│   │   │   └── base.templ
│   │   ├── pages/
│   │   │   ├── home.templ
│   │   │   ├── upload.templ
│   │   │   └── gallery.templ
│   │   └── components/
│   │       ├── navbar.templ
│   │       ├── job_card.templ
│   │       └── progress.templ
│   └── static/
│       ├── css/
│       └── js/
│
├── deployments/
│   ├── docker/
│   │   ├── Dockerfile.frontend
│   │   ├── Dockerfile.api
│   │   └── Dockerfile.worker
│   └── kubernetes/
│       ├── base/                  # Kustomize base
│       │   ├── kustomization.yaml
│       │   ├── namespace.yaml
│       │   ├── frontend/
│       │   │   ├── deployment.yaml
│       │   │   └── service.yaml
│       │   ├── api/
│       │   │   ├── deployment.yaml
│       │   │   └── service.yaml
│       │   ├── worker/
│       │   │   ├── deployment.yaml
│       │   │   └── scaledobject.yaml  # KEDA
│       │   ├── postgres/
│       │   │   ├── statefulset.yaml
│       │   │   ├── service.yaml
│       │   │   └── pvc.yaml
│       │   ├── redis/
│       │   │   ├── deployment.yaml
│       │   │   └── service.yaml
│       │   ├── minio/
│       │   │   ├── deployment.yaml
│       │   │   ├── service.yaml
│       │   │   └── pvc.yaml
│       │   └── ingress.yaml
│       └── overlays/
│           ├── dev/
│           │   └── kustomization.yaml
│           └── prod/
│               └── kustomization.yaml
│
├── scripts/
│   ├── load-test.sh              # Generate load for demo
│   └── setup-cluster.sh          # K8s cluster setup
│
├── migrations/                    # Database migrations
│   ├── 001_init.up.sql
│   └── 001_init.down.sql
│
├── docker-compose.yaml           # Local development
├── Makefile                      # Build automation
├── go.mod
├── go.sum
└── README.md
```

## Processing Operations

Each image can undergo multiple CPU-intensive operations:

| Operation       | Description                           | CPU Intensity |
|-----------------|---------------------------------------|---------------|
| Resize          | Scale to multiple dimensions          | Medium        |
| Thumbnail       | Generate small preview (150x150)      | Low           |
| Blur            | Gaussian blur (configurable radius)   | High          |
| Sharpen         | Unsharp mask filter                   | High          |
| Grayscale       | Convert to grayscale                  | Low           |
| Sepia           | Apply sepia tone                      | Medium        |
| Watermark       | Overlay text/image watermark          | Medium        |
| Rotate          | Rotate by degrees                     | Low           |
| Flip            | Horizontal/vertical flip              | Low           |
| Brightness      | Adjust brightness                     | Medium        |
| Contrast        | Adjust contrast                       | Medium        |
| Saturation      | Adjust color saturation               | Medium        |

**Heavy Processing Mode:** For demo purposes, we'll add options to:
- Process at high resolution (4K+)
- Apply all filters sequentially
- Run multiple iterations (artificial load)

## API Endpoints

### Public API (API Gateway)

```
POST   /api/v1/jobs                Create processing job
GET    /api/v1/jobs                List all jobs (paginated)
GET    /api/v1/jobs/:id            Get job status
DELETE /api/v1/jobs/:id            Cancel job

GET    /api/v1/images/:id          Get processed image
GET    /api/v1/images/:id/download Download original/processed

GET    /api/v1/health              Health check
GET    /api/v1/metrics             Prometheus metrics
```

### Internal API (Worker)

```
GET    /health                     Worker health
GET    /metrics                    Worker metrics
```

## Job Lifecycle

```
┌─────────┐    ┌─────────┐    ┌────────────┐    ┌───────────┐    ┌──────────┐
│ PENDING │───▶│ QUEUED  │───▶│ PROCESSING │───▶│ COMPLETED │    │  FAILED  │
└─────────┘    └─────────┘    └────────────┘    └───────────┘    └──────────┘
     │              │               │                                  ▲
     │              │               └──────────────────────────────────┘
     │              │                        (on error)
     └──────────────┴─────────────────────────────────────────────────▶│
                              (on cancel)                         ┌──────────┐
                                                                  │ CANCELLED│
                                                                  └──────────┘
```

## Implementation Phases

### Phase 1: Foundation (Day 1-2)
- [ ] Initialize Go modules
- [ ] Set up project structure
- [ ] Implement config loading (env vars)
- [ ] Create domain models
- [ ] Set up PostgreSQL schema and migrations
- [ ] Implement database repository

### Phase 2: Core Services (Day 3-4)
- [ ] Implement MinIO storage client
- [ ] Implement Redis queue (producer/consumer)
- [ ] Build API Gateway handlers
- [ ] Build Worker processing loop
- [ ] Implement image processing operations

### Phase 3: Frontend (Day 5)
- [ ] Set up templ templates
- [ ] Build upload page with drag-drop
- [ ] Build gallery view
- [ ] Add htmx for real-time updates
- [ ] Style with Tailwind CSS

### Phase 4: Containerization (Day 6)
- [ ] Create multi-stage Dockerfiles
- [ ] Set up docker-compose for local dev
- [ ] Test full stack locally
- [ ] Optimize image sizes

### Phase 5: Kubernetes (Day 7-8)
- [ ] Create Kubernetes manifests
- [ ] Set up Kustomize overlays
- [ ] Configure HPA for frontend/api
- [ ] Set up KEDA for worker scaling
- [ ] Create Ingress configuration

### Phase 6: Observability (Day 9)
- [ ] Add Prometheus metrics
- [ ] Create Grafana dashboards
- [ ] Implement structured logging
- [ ] Add OpenTelemetry tracing

### Phase 7: Load Testing & Demo (Day 10)
- [ ] Create load testing script
- [ ] Document demo scenarios
- [ ] Fine-tune scaling parameters
- [ ] Record demo video (optional)

## Scaling Strategy

### Horizontal Pod Autoscaler (HPA)

**Frontend & API:**
```yaml
metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
minReplicas: 2
maxReplicas: 10
```

### KEDA ScaledObject (Workers)

```yaml
triggers:
  - type: redis-streams
    metadata:
      address: redis:6379
      stream: image-jobs
      consumerGroup: workers
      lagCount: "5"  # Scale up when 5+ pending jobs
minReplicaCount: 1
maxReplicaCount: 20
```

## Security Considerations

1. **Secrets Management:** Use Kubernetes Secrets (or HashiCorp Vault)
2. **Network Policies:** Restrict pod-to-pod communication
3. **Image Scanning:** Scan containers with Trivy
4. **RBAC:** Minimal service account permissions
5. **Input Validation:** Validate file types, sizes
6. **Rate Limiting:** Protect API from abuse

## Future Enhancements (Optional)

- [ ] HashiCorp Vault integration for secrets
- [ ] HashiCorp Consul for service discovery
- [ ] Istio service mesh
- [ ] GitOps with ArgoCD
- [ ] Multi-tenant support
- [ ] Webhook notifications
- [ ] S3 direct upload (presigned URLs)
