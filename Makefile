.PHONY: all build test clean dev-api dev-worker up down logs migrate rebuild load-test \
	k8s-apply-dev k8s-apply-prod lint lint-fix test-unit test-race test-ci test-coverage \
	test-integration ci-local pre-commit docker-build security-scan help

# Variables
DOCKER_COMPOSE = docker compose
GO = go
GOFLAGS = -v
API_BINARY = bin/api
WORKER_BINARY = bin/worker
FRONTEND_BINARY = bin/frontend
GOLANGCI_LINT_VERSION = v1.64.8
MIN_COVERAGE = 60

# Git info for versioning
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
VERSION ?= $(GIT_COMMIT)

# Build flags
LDFLAGS = -ldflags="-w -s -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"

# Default target
all: build

# ============================================================================
# Build targets
# ============================================================================

build: build-api build-worker build-frontend

build-api:
	@echo "Building API server..."
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(API_BINARY) ./cmd/api

build-worker:
	@echo "Building worker..."
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(WORKER_BINARY) ./cmd/worker

build-frontend:
	@echo "Building frontend..."
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(FRONTEND_BINARY) ./cmd/frontend

# Cross-compilation targets
build-linux-amd64:
	@echo "Building for linux/amd64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o bin/api-linux-amd64 ./cmd/api
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o bin/worker-linux-amd64 ./cmd/worker
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o bin/frontend-linux-amd64 ./cmd/frontend

build-linux-arm64:
	@echo "Building for linux/arm64..."
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o bin/api-linux-arm64 ./cmd/api
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o bin/worker-linux-arm64 ./cmd/worker
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o bin/frontend-linux-arm64 ./cmd/frontend

build-all: build-linux-amd64 build-linux-arm64

# ============================================================================
# Test targets
# ============================================================================

test:
	@echo "Running tests..."
	$(GO) test -v ./...

test-unit:
	@echo "Running unit tests..."
	$(GO) test -short -v ./...

test-race:
	@echo "Running tests with race detector..."
	$(GO) test -race -short -timeout=5m ./...

test-ci:
	@echo "Running tests exactly as CI does..."
	$(GO) test -v -race -coverprofile=coverage.out -covermode=atomic -timeout=10m ./...
	@echo ""
	@echo "Coverage report generated: coverage.out"
	@$(GO) tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'

test-coverage:
	@echo "Running tests with coverage..."
	$(GO) test -v -coverprofile=coverage.out -timeout=10m ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo ""
	@$(GO) tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'
	@echo "HTML coverage report: coverage.html"

test-coverage-check:
	@echo "Checking coverage meets minimum threshold..."
	@$(GO) test -coverprofile=coverage.out -timeout=10m ./... > /dev/null 2>&1
	@COVERAGE=$$($(GO) tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	if [ $$(echo "$$COVERAGE < $(MIN_COVERAGE)" | bc -l) -eq 1 ]; then \
		echo "❌ Coverage $$COVERAGE% is below minimum threshold of $(MIN_COVERAGE)%"; \
		exit 1; \
	else \
		echo "✅ Coverage $$COVERAGE% meets minimum threshold of $(MIN_COVERAGE)%"; \
	fi

test-integration:
	@echo "Starting test services..."
	$(DOCKER_COMPOSE) -f docker-compose.test.yml up -d
	@echo "Waiting for services to be ready..."
	@sleep 15
	@echo "Running integration tests..."
	TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5433/imageprocessor?sslmode=disable" \
	TEST_MINIO_ENDPOINT="localhost:9001" \
	$(GO) test -v -timeout=10m ./... || ($(DOCKER_COMPOSE) -f docker-compose.test.yml down && exit 1)
	$(DOCKER_COMPOSE) -f docker-compose.test.yml down
	@echo "✅ Integration tests complete"

test-integration-keep:
	@echo "Starting test services (will keep running)..."
	$(DOCKER_COMPOSE) -f docker-compose.test.yml up -d
	@echo "Waiting for services to be ready..."
	@sleep 15
	@echo "Running integration tests..."
	TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5433/imageprocessor?sslmode=disable" \
	TEST_MINIO_ENDPOINT="localhost:9001" \
	$(GO) test -v -timeout=10m ./...
	@echo "✅ Integration tests complete (services still running)"
	@echo "Run 'make test-integration-cleanup' to stop services"

test-integration-cleanup:
	@echo "Stopping test services..."
	$(DOCKER_COMPOSE) -f docker-compose.test.yml down -v
	@echo "✅ Test services stopped"

# ============================================================================
# Linting and formatting
# ============================================================================

lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m; \
	else \
		echo "golangci-lint not found. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
		golangci-lint run --timeout=5m; \
	fi

lint-fix:
	@echo "Running linters with auto-fix..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --timeout=5m --fix; \
	else \
		echo "golangci-lint not found. Installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
		golangci-lint run --timeout=5m --fix; \
	fi

fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w -local github.com/timkrebs/image-processor .; \
	fi

fmt-check:
	@echo "Checking code formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "❌ Code is not formatted. Run 'make fmt' to fix."; \
		gofmt -d .; \
		exit 1; \
	else \
		echo "✅ Code is properly formatted"; \
	fi

# ============================================================================
# Security scanning
# ============================================================================

security-scan:
	@echo "Running security scans..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec -quiet ./...; \
	else \
		echo "gosec not found. Installing..."; \
		go install github.com/securego/gosec/v2/cmd/gosec@latest; \
		gosec -quiet ./...; \
	fi
	@echo ""
	@echo "Running vulnerability check..."
	@if command -v govulncheck >/dev/null 2>&1; then \
		govulncheck ./...; \
	else \
		echo "govulncheck not found. Installing..."; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
		govulncheck ./...; \
	fi

# ============================================================================
# CI simulation
# ============================================================================

ci-local: deps-verify fmt-check lint test-race test-ci security-scan
	@echo ""
	@echo "============================================"
	@echo "✅ All CI checks passed locally!"
	@echo "============================================"

ci-quick: fmt-check lint test-unit
	@echo ""
	@echo "============================================"
	@echo "✅ Quick CI checks passed!"
	@echo "============================================"

# Pre-commit checks
pre-commit:
	@echo "Running pre-commit checks..."
	@if command -v pre-commit >/dev/null 2>&1; then \
		pre-commit run --all-files || (echo "❌ Pre-commit checks failed" && exit 1); \
	else \
		echo "pre-commit not installed. Running manual checks..."; \
		$(MAKE) fmt-check lint test-unit; \
	fi
	@echo "✅ Pre-commit checks passed"

# ============================================================================
# Dependency management
# ============================================================================

deps:
	@echo "Downloading dependencies..."
	$(GO) mod download

deps-tidy:
	@echo "Tidying dependencies..."
	$(GO) mod tidy

deps-verify:
	@echo "Verifying dependencies..."
	$(GO) mod verify
	@echo "Checking go.mod tidiness..."
	@$(GO) mod tidy
	@if [ -n "$$(git diff --name-only go.mod go.sum)" ]; then \
		echo "❌ go.mod or go.sum is not tidy. Run 'go mod tidy' and commit changes."; \
		exit 1; \
	else \
		echo "✅ Dependencies are tidy"; \
	fi

deps-update:
	@echo "Updating dependencies..."
	$(GO) get -u ./...
	$(GO) mod tidy
	@echo "✅ Dependencies updated"

# ============================================================================
# Clean
# ============================================================================

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f *.sarif
	@echo "✅ Clean complete"

clean-all: clean test-integration-cleanup
	@echo "Cleaning all artifacts including Docker..."
	$(DOCKER_COMPOSE) down -v --remove-orphans 2>/dev/null || true
	@echo "✅ Full clean complete"

# ============================================================================
# Local development
# ============================================================================

dev-api:
	$(GO) run ./cmd/api

dev-worker:
	$(GO) run ./cmd/worker

dev-frontend:
	$(GO) run ./cmd/frontend

# ============================================================================
# Docker Compose commands
# ============================================================================

up:
	$(DOCKER_COMPOSE) up -d postgres redis minio

up-all:
	$(DOCKER_COMPOSE) up -d

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

rebuild:
	$(DOCKER_COMPOSE) up -d --build

start-api:
	$(DOCKER_COMPOSE) up -d api

start-worker:
	$(DOCKER_COMPOSE) up -d worker

# ============================================================================
# Database migrations
# ============================================================================

migrate:
	@echo "Running migrations..."
	@if command -v migrate >/dev/null 2>&1; then \
		migrate -path ./migrations -database "postgres://postgres:postgres@localhost:5432/imageprocessor?sslmode=disable" up; \
	else \
		docker compose exec -T postgres psql -U postgres -d imageprocessor < migrations/001_init.up.sql; \
		docker compose exec -T postgres psql -U postgres -d imageprocessor < migrations/002_performance_indexes.up.sql; \
		docker compose exec -T postgres psql -U postgres -d imageprocessor < migrations/003_add_users.up.sql; \
		docker compose exec -T postgres psql -U postgres -d imageprocessor < migrations/004_add_cleanup.up.sql; \
	fi
	@echo "✅ Migrations complete"

migrate-down:
	@echo "Rolling back migrations..."
	@docker compose exec -T postgres psql -U postgres -d imageprocessor < migrations/001_init.down.sql

createdb:
	@docker compose exec -T postgres psql -U postgres -c "CREATE DATABASE imageprocessor;" || true

# ============================================================================
# Load testing
# ============================================================================

load-test:
	./scripts/load-test.sh

load-test-heavy:
	HEAVY_MODE=true TOTAL_JOBS=100 CONCURRENT=5 ./scripts/load-test.sh

# ============================================================================
# Kubernetes commands
# ============================================================================

k8s-apply-dev:
	kubectl apply -k deployments/kubernetes/overlays/dev

k8s-apply-staging:
	kubectl apply -k deployments/kubernetes/overlays/staging

k8s-apply-prod:
	kubectl apply -k deployments/kubernetes/overlays/prod

k8s-delete-dev:
	kubectl delete -k deployments/kubernetes/overlays/dev

k8s-delete-staging:
	kubectl delete -k deployments/kubernetes/overlays/staging

k8s-delete-prod:
	kubectl delete -k deployments/kubernetes/overlays/prod

# ============================================================================
# Docker image builds
# ============================================================================

docker-build-api:
	docker build -t image-processor-api:$(VERSION) -f deployments/docker/Dockerfile.api .

docker-build-worker:
	docker build -t image-processor-worker:$(VERSION) -f deployments/docker/Dockerfile.worker .

docker-build-frontend:
	docker build -t image-processor-frontend:$(VERSION) -f deployments/docker/Dockerfile.frontend .

docker-build: docker-build-api docker-build-worker docker-build-frontend
	@echo "✅ All Docker images built"

# ============================================================================
# Generate
# ============================================================================

generate:
	$(GO) generate ./...

# ============================================================================
# Tools installation
# ============================================================================

tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install github.com/securego/gosec/v2/cmd/gosec@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	@echo "✅ Tools installed"

# ============================================================================
# Help
# ============================================================================

help:
	@echo "Available targets:"
	@echo ""
	@echo "Build:"
	@echo "  build              - Build all binaries"
	@echo "  build-api          - Build API server"
	@echo "  build-worker       - Build worker"
	@echo "  build-frontend     - Build frontend"
	@echo "  build-all          - Cross-compile for all platforms"
	@echo ""
	@echo "Test:"
	@echo "  test               - Run all tests"
	@echo "  test-unit          - Run unit tests only"
	@echo "  test-race          - Run tests with race detector"
	@echo "  test-ci            - Run tests exactly as CI does"
	@echo "  test-coverage      - Run tests with coverage report"
	@echo "  test-coverage-check- Check coverage meets threshold"
	@echo "  test-integration   - Run integration tests with Docker services"
	@echo ""
	@echo "Code Quality:"
	@echo "  lint               - Run linters"
	@echo "  lint-fix           - Run linters with auto-fix"
	@echo "  fmt                - Format code"
	@echo "  fmt-check          - Check code formatting"
	@echo "  security-scan      - Run security scans"
	@echo ""
	@echo "CI:"
	@echo "  ci-local           - Run full CI pipeline locally"
	@echo "  ci-quick           - Run quick CI checks"
	@echo "  pre-commit         - Run pre-commit checks"
	@echo ""
	@echo "Dependencies:"
	@echo "  deps               - Download dependencies"
	@echo "  deps-tidy          - Tidy dependencies"
	@echo "  deps-verify        - Verify dependencies"
	@echo "  deps-update        - Update dependencies"
	@echo ""
	@echo "Development:"
	@echo "  dev-api            - Run API server locally"
	@echo "  dev-worker         - Run worker locally"
	@echo "  dev-frontend       - Run frontend locally"
	@echo "  up                 - Start infrastructure (Postgres, Redis, MinIO)"
	@echo "  down               - Stop all containers"
	@echo "  logs               - Follow container logs"
	@echo "  migrate            - Run database migrations"
	@echo "  rebuild            - Rebuild and start all services"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build       - Build all Docker images"
	@echo "  docker-build-api   - Build API Docker image"
	@echo "  docker-build-worker- Build worker Docker image"
	@echo ""
	@echo "Kubernetes:"
	@echo "  k8s-apply-dev      - Deploy to Kubernetes (dev)"
	@echo "  k8s-apply-staging  - Deploy to Kubernetes (staging)"
	@echo "  k8s-apply-prod     - Deploy to Kubernetes (prod)"
	@echo ""
	@echo "Other:"
	@echo "  clean              - Remove build artifacts"
	@echo "  clean-all          - Remove all artifacts including Docker"
	@echo "  tools              - Install development tools"
	@echo "  load-test          - Run load test"
	@echo "  help               - Show this help"
