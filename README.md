# Image Processing Microservices Platform

A cloud-native, Kubernetes-ready image processing platform built with Go microservices. Users upload images for processing (resize, filters, thumbnails, watermarks), with automatic horizontal scaling based on workload.

## Features

- **Scalable Architecture**: Frontend, API, and Worker services scale independently
- **Queue-Based Processing**: Redis Streams for reliable job distribution
- **Multiple Image Operations**: Resize, blur, sharpen, grayscale, sepia, rotate, and more
- **S3-Compatible Storage**: MinIO for image storage
- **Kubernetes Native**: Full K8s manifests with HPA and KEDA autoscaling
- **Observability Ready**: Prometheus metrics, structured logging

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Kubernetes Cluster                       │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│   Ingress ──▶ API Gateway ──▶ Redis Queue ──▶ Worker Pool  │
│                    │                              │         │
│                    ▼                              ▼         │
│              PostgreSQL                        MinIO        │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.22+
- Docker & Docker Compose
- (Optional) Kubernetes cluster with KEDA installed

### Local Development

1. **Clone and setup**
   ```bash
   git clone https://github.com/timohirt/image-processor
   cd image-processor
   go mod download
   ```

2. **Start infrastructure**
   ```bash
   make up
   ```

3. **Run migrations**
   ```bash
   make migrate
   ```

4. **Start services**
   ```bash
   # Terminal 1: API
   make dev-api

   # Terminal 2: Worker
   make dev-worker
   ```

5. **Test the API**
   ```bash
   # Upload and process an image
   curl -X POST http://localhost:8080/api/v1/jobs \
     -F "image=@/path/to/image.jpg" \
     -F 'operations=[{"operation":"resize","parameters":{"width":800}},{"operation":"thumbnail"}]'
   ```

### Docker Compose

```bash
# Build and start all services
make rebuild

# View logs
make logs

# Stop services
make down
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/jobs` | Create processing job (upload image) |
| GET | `/api/v1/jobs` | List all jobs (paginated) |
| GET | `/api/v1/jobs/:id` | Get job status |
| DELETE | `/api/v1/jobs/:id` | Cancel job |
| GET | `/api/v1/images/:id` | Get/download image |
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/stats/queue` | Queue statistics |

## Available Operations

| Operation | Parameters | Description |
|-----------|------------|-------------|
| `resize` | `width`, `height` | Scale image |
| `thumbnail` | `size` | Create square thumbnail |
| `blur` | `sigma` | Gaussian blur |
| `sharpen` | `sigma` | Unsharp mask |
| `grayscale` | - | Convert to grayscale |
| `sepia` | - | Apply sepia tone |
| `rotate` | `angle` | Rotate by degrees |
| `flip` | `horizontal` | Flip horizontally/vertically |
| `brightness` | `amount` | Adjust brightness (-100 to 100) |
| `contrast` | `amount` | Adjust contrast |
| `saturation` | `amount` | Adjust saturation |

## Kubernetes Deployment

### Prerequisites

- Kubernetes cluster (1.24+)
- kubectl configured
- KEDA installed (for worker autoscaling)
- Ingress controller (nginx recommended)

### Deploy

```bash
# Development environment
make k8s-apply-dev

# Production environment
make k8s-apply-prod
```

### KEDA Autoscaling

Workers automatically scale based on Redis queue depth:

```yaml
triggers:
  - type: redis-streams
    metadata:
      stream: image-jobs
      consumerGroup: workers
      lagCount: "5"  # Scale up when 5+ pending jobs
minReplicaCount: 1
maxReplicaCount: 20
```

## Load Testing

Generate load to see autoscaling in action:

```bash
# Run load test (50 jobs, 5 concurrent)
make load-test

# Custom load test
TOTAL_JOBS=100 CONCURRENT=10 HEAVY_MODE=true ./scripts/load-test.sh
```

## Configuration

Environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `HTTP_PORT` | 8080 | API server port |
| `DATABASE_URL` | - | PostgreSQL connection string |
| `REDIS_ADDR` | localhost:6379 | Redis address |
| `MINIO_ENDPOINT` | localhost:9000 | MinIO endpoint |
| `MINIO_ACCESS_KEY` | minioadmin | MinIO access key |
| `MINIO_SECRET_KEY` | minioadmin | MinIO secret key |
| `MINIO_BUCKET` | images | Storage bucket name |
| `WORKER_CONCURRENCY` | 4 | Worker goroutine count |
| `MAX_UPLOAD_SIZE` | 52428800 | Max upload size (50MB) |

## Project Structure

```
image-processor/
├── cmd/                    # Application entrypoints
│   ├── api/               # API Gateway service
│   └── worker/            # Processing worker service
├── internal/              # Private application code
│   ├── api/              # HTTP handlers & routes
│   ├── config/           # Configuration
│   ├── database/         # PostgreSQL repository
│   ├── models/           # Domain models
│   ├── processor/        # Image processing logic
│   ├── queue/            # Redis queue
│   └── storage/          # MinIO storage
├── deployments/
│   ├── docker/           # Dockerfiles
│   └── kubernetes/       # K8s manifests (Kustomize)
├── migrations/           # Database migrations
├── scripts/              # Utility scripts
├── docker-compose.yaml
├── Makefile
└── README.md
```

## Future Enhancements

- [ ] HashiCorp Vault for secrets management
- [ ] HashiCorp Consul for service discovery
- [ ] Frontend UI with real-time updates (htmx)
- [ ] Webhook notifications
- [ ] Multi-tenant support
- [ ] Prometheus + Grafana dashboards

## License

MIT
