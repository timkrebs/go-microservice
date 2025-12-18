.PHONY: all build test clean dev-api dev-worker up down logs migrate rebuild load-test k8s-apply-dev k8s-apply-prod

# Variables
DOCKER_COMPOSE = docker compose
GO = go
GOFLAGS = -v
API_BINARY = bin/api
WORKER_BINARY = bin/worker

# Default target
all: build

# Build binaries
build: build-api build-worker

build-api:
	$(GO) build $(GOFLAGS) -o $(API_BINARY) ./cmd/api

build-worker:
	$(GO) build $(GOFLAGS) -o $(WORKER_BINARY) ./cmd/worker

# Test
test:
	$(GO) test -v ./...

test-coverage:
	$(GO) test -v -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

# Clean
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Local development
dev-api:
	$(GO) run ./cmd/api

dev-worker:
	$(GO) run ./cmd/worker

# Docker Compose commands
up:
	$(DOCKER_COMPOSE) up -d postgres redis minio

down:
	$(DOCKER_COMPOSE) down

down-volumes:
	$(DOCKER_COMPOSE) down -v

logs:
	$(DOCKER_COMPOSE) logs -f

logs-api:
	$(DOCKER_COMPOSE) logs -f api

logs-worker:
	$(DOCKER_COMPOSE) logs -f worker

# Build and start all services
rebuild:
	$(DOCKER_COMPOSE) up -d --build

# Start specific services
start-api:
	$(DOCKER_COMPOSE) up -d api

start-worker:
	$(DOCKER_COMPOSE) up -d worker

# Database migrations (runs via Docker)
migrate:
	@echo "Running migrations..."
	@docker compose exec -T postgres psql -U postgres -d imageprocessor < migrations/001_init.up.sql

migrate-down:
	@echo "Rolling back migrations..."
	@docker compose exec -T postgres psql -U postgres -d imageprocessor < migrations/001_init.down.sql

# Create database (runs via Docker)
createdb:
	@docker compose exec -T postgres psql -U postgres -c "CREATE DATABASE imageprocessor;" || true

# Load testing
load-test:
	./scripts/load-test.sh

load-test-heavy:
	HEAVY_MODE=true TOTAL_JOBS=100 CONCURRENT=5 ./scripts/load-test.sh

# Kubernetes commands
k8s-apply-dev:
	kubectl apply -k deployments/kubernetes/overlays/dev

k8s-apply-prod:
	kubectl apply -k deployments/kubernetes/overlays/prod

k8s-delete-dev:
	kubectl delete -k deployments/kubernetes/overlays/dev

k8s-delete-prod:
	kubectl delete -k deployments/kubernetes/overlays/prod

# Docker image builds
docker-build-api:
	docker build -t image-processor-api:latest -f deployments/docker/Dockerfile.api .

docker-build-worker:
	docker build -t image-processor-worker:latest -f deployments/docker/Dockerfile.worker .

docker-build: docker-build-api docker-build-worker

# Linting
lint:
	golangci-lint run ./...

# Format code
fmt:
	$(GO) fmt ./...

# Download dependencies
deps:
	$(GO) mod download
	$(GO) mod tidy

# Generate mocks (if needed)
generate:
	$(GO) generate ./...

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build all binaries"
	@echo "  test           - Run tests"
	@echo "  clean          - Remove build artifacts"
	@echo "  dev-api        - Run API server locally"
	@echo "  dev-worker     - Run worker locally"
	@echo "  up             - Start infrastructure (Postgres, Redis, MinIO)"
	@echo "  down           - Stop all containers"
	@echo "  logs           - Follow container logs"
	@echo "  migrate        - Run database migrations"
	@echo "  rebuild        - Rebuild and start all services"
	@echo "  load-test      - Run load test"
	@echo "  k8s-apply-dev  - Deploy to Kubernetes (dev)"
	@echo "  k8s-apply-prod - Deploy to Kubernetes (prod)"
	@echo "  docker-build   - Build Docker images"
