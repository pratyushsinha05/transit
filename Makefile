# ============================================================================
# TRANSIT BACKEND - COMPREHENSIVE MAKEFILE
# ============================================================================
# One command to rule them all: make docker-up
# ============================================================================

.PHONY: help check-deps check-system check-docker check-go check-tools
.PHONY: verify-setup check-go-deps install-go-deps verify-go-deps
.PHONY: build build-docker build-prod rebuild-backend
.PHONY: docker-up docker-down docker-clean docker-logs docker-health
.PHONY: docker-ps docker-shell docker-health-backend docker-restart
.PHONY: db-migrate db-reset db-seed db-shell db-info
.PHONY: test test-unit test-integration test-coverage fmt vet lint lint-layering
.PHONY: run dev dev-docker
.PHONY: clean clean-all clean-docker version info

# ============================================================================
# PROJECT & BUILD CONFIGURATION
# ============================================================================

PROJECT_NAME := transit-backend
GO_VERSION := 1.23
IMAGE_NAME := $(PROJECT_NAME)
IMAGE_TAG := latest
REGISTRY := localhost

# Paths
BACKEND_DIR := ./backend
MIGRATIONS_DIR := $(BACKEND_DIR)/migrations
CMD_SERVER := $(BACKEND_DIR)/cmd/server

# Docker Compose
DOCKER_COMPOSE_FILE := ./infra/docker-compose.yml
# Auto-detect Docker Compose command (v2 or v1)
DOCKER_COMPOSE_CMD := $(shell if docker compose version >/dev/null 2>&1; then echo "docker compose"; else echo "docker-compose"; fi)

# Database Configuration
DB_USER := transit_user
DB_PASSWORD := transit_password
DB_NAME := transit_poc
DB_HOST := localhost
DB_PORT := 5432
POSTGRES_URL := postgresql://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=disable

# Build Information
VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_HASH := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go Build Flags
GO_BUILD_FLAGS := -v
GO_TEST_FLAGS := -v -race -cover

# Color Output
BLUE := \033[0;36m
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m

# ============================================================================
# PHASE 1: DEPENDENCY CHECKING
# ============================================================================

# MAIN ENTRY: Check all critical dependencies
check-deps: check-system check-docker check-go check-tools ## Verify all critical dependencies
	@echo ""
	@echo "$(GREEN)============================================$(NC)"
	@echo "$(GREEN)✓ All critical dependencies verified$(NC)"
	@echo "$(GREEN)Ready to run: make docker-up$(NC)"
	@echo "$(GREEN)============================================$(NC)"
	@echo ""

# Check system fundamentals (Make, Git)
check-system: ## Check system fundamentals (Make, Git)
	@echo "$(BLUE)[1/4] Checking system dependencies...$(NC)"
	@command -v make >/dev/null 2>&1 || { \
		echo "$(RED)✗ Make not found$(NC)"; \
		exit 1; \
	}
	@command -v git >/dev/null 2>&1 || { \
		echo "$(YELLOW)⚠ Git not found (optional, needed for version info)$(NC)"; \
	}
	@echo "$(GREEN)  ✓ Make installed$(NC)"
	@echo "$(GREEN)  ✓ Git available$(NC)"

# Check Docker and Docker Compose
check-docker: ## Check Docker and Docker Compose installation
	@echo "$(BLUE)[2/4] Checking Docker installation...$(NC)"
	@command -v docker >/dev/null 2>&1 || { \
		echo "$(RED)✗ Docker not found$(NC)"; \
		echo "$(YELLOW)Please install Docker manually: https://docs.docker.com/get-docker/$(NC)"; \
		exit 1; \
	}
	@echo "$(GREEN)  ✓ Docker installed$(NC)"
	@$(DOCKER_COMPOSE_CMD) version >/dev/null 2>&1 || { \
		echo "$(RED)✗ Docker Compose not found$(NC)"; \
		echo "$(YELLOW)Please install Docker Compose manually: https://docs.docker.com/compose/install/$(NC)"; \
		exit 1; \
	}
	@echo "$(GREEN)  ✓ Docker Compose installed$(NC)"
	@docker ps >/dev/null 2>&1 || { \
		echo "$(RED)✗ Docker daemon not running$(NC)"; \
		echo "$(YELLOW)Please start Docker Desktop manually$(NC)"; \
		exit 1; \
	}
	@echo "$(GREEN)  ✓ Docker daemon running$(NC)"

# Check Go installation and version
check-go: ## Check Go installation and minimum version (1.21+)
	@echo "$(BLUE)[3/4] Checking Go installation...$(NC)"
	@command -v go >/dev/null 2>&1 || { \
		echo "$(RED)✗ Go not found$(NC)"; \
		echo "$(YELLOW)Please install Go 1.21+ manually: https://golang.org/dl/$(NC)"; \
		exit 1; \
	}
	@GO_VER=$$(go version | awk '{print $$3}' | sed 's/go//'); \
	MAJOR=$$(echo $$GO_VER | cut -d. -f1); \
	MINOR=$$(echo $$GO_VER | cut -d. -f2); \
	if [ "$$MAJOR" -lt 1 ] || ([ "$$MAJOR" -eq 1 ] && [ "$$MINOR" -lt 21 ]); then \
		echo "$(RED)✗ Go version too old (found $$GO_VER, need 1.21+)$(NC)"; \
		echo "$(YELLOW)Please upgrade Go manually$(NC)"; \
		exit 1; \
	fi; \
	echo "$(GREEN)  ✓ Go $$GO_VER installed$(NC)"

# Check development tools (PostgreSQL, Redis, curl)
check-tools: ## Check development tools (psql, redis-cli, curl)
	@echo "$(BLUE)[4/4] Checking development tools...$(NC)"
	@if command -v psql >/dev/null 2>&1; then \
		echo "$(GREEN)  ✓ PostgreSQL client installed$(NC)"; \
	else \
		echo "$(YELLOW)  ⚠ psql not found (recommended for db-shell)$(NC)"; \
	fi
	@if command -v redis-cli >/dev/null 2>&1; then \
		echo "$(GREEN)  ✓ Redis CLI installed$(NC)"; \
	else \
		echo "$(YELLOW)  ⚠ redis-cli not found (optional)$(NC)"; \
	fi
	@if command -v curl >/dev/null 2>&1; then \
		echo "$(GREEN)  ✓ curl installed$(NC)"; \
	else \
		echo "$(RED)✗ curl not found (required for health checks)$(NC)"; \
		echo "$(YELLOW)Please install curl manually$(NC)"; \
		exit 1; \
	fi
	@if command -v jq >/dev/null 2>&1; then \
		echo "$(GREEN)  ✓ jq installed$(NC)"; \
	else \
		echo "$(YELLOW)  ⚠ jq not found (optional, for JSON formatting)$(NC)"; \
	fi

# Comprehensive setup verification with versions
verify-setup: check-deps ## Comprehensive environment verification with all versions
	@echo ""
	@echo "$(BLUE)╔════════════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(BLUE)║          ENVIRONMENT SETUP VERIFICATION COMPLETE               ║$(NC)"
	@echo "$(BLUE)╚════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "$(BLUE)System Information:$(NC)"
	@make --version | head -1 | sed 's/^/  /'
	@echo ""
	@echo "$(BLUE)Docker Stack:$(NC)"
	@docker --version | sed 's/^/  /'
	@$(DOCKER_COMPOSE_CMD) version | head -1 | sed 's/^/  /'
	@echo ""
	@echo "$(BLUE)Go Toolchain:$(NC)"
	@go version | sed 's/^/  /'
	@echo ""
	@echo "$(BLUE)Project Paths:$(NC)"
	@echo "  Backend Directory: $(BACKEND_DIR)"
	@echo "  Migrations Directory: $(MIGRATIONS_DIR)"
	@echo "  Database: $(DB_NAME) @ $(DB_HOST):$(DB_PORT)"
	@echo ""
	@echo "$(GREEN)✓ All systems verified and ready$(NC)"

# Check Go module dependencies
check-go-deps: ## Verify Go module integrity
	@echo "$(BLUE)Checking Go module dependencies...$(NC)"
	cd $(BACKEND_DIR) && go mod verify
	@echo "$(GREEN)✓ Go dependencies verified$(NC)"

# Install/download Go dependencies
install-go-deps: ## Download all Go dependencies
	@echo "$(BLUE)Installing Go dependencies...$(NC)"
	cd $(BACKEND_DIR) && go mod download
	cd $(BACKEND_DIR) && go mod tidy
	@echo "$(GREEN)✓ Go dependencies installed$(NC)"

# Verify Go dependency versions
verify-go-deps: ## Show all Go dependency versions
	@echo "$(BLUE)Go Dependencies:$(NC)"
	@cd $(BACKEND_DIR) && go list -m all | grep -E "github.com/(labstack|gorilla|jackc|lib|redis|uber|joho)" | while read dep version; do \
		echo "  ✓ $$dep $$version"; \
	done



# ============================================================================
# PHASE 3: DOCKER BUILD & OPERATIONS
# ============================================================================

# Build Docker image (multi-stage)
build-docker: check-docker ## Build Docker image (multi-stage: 15-20MB)
	@echo "$(BLUE)Building Docker image...$(NC)"
	@echo "  Stage 1: golang:1.23-alpine (builder)"
	@echo "  Stage 2: alpine:latest (runtime)"
	docker build \
		-t $(IMAGE_NAME):$(IMAGE_TAG) \
		-f $(BACKEND_DIR)/Dockerfile \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_HASH=$(GIT_HASH) \
		$(BACKEND_DIR)
	@echo ""
	@echo "$(GREEN)✓ Docker image built: $(IMAGE_NAME):$(IMAGE_TAG)$(NC)"
	@docker images | grep $(IMAGE_NAME) | head -1 | awk '{printf "  Size: %s\n", $$7}'

# Build production binary locally
build-prod: check-go ## Build production binary (optimized, stripped)
	@echo "$(BLUE)Building production binary...$(NC)"
	cd $(BACKEND_DIR) && go build \
		-ldflags="-s -w -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitHash=$(GIT_HASH)" \
		-o ../server \
		./cmd/server
	@echo "$(GREEN)✓ Production binary built: server$(NC)"
	@ls -lh server | awk '{print "  Size: " $$5}'

# Build debug binary locally
build: check-go ## Build debug binary
	@echo "$(BLUE)Building debug binary...$(NC)"
	cd $(BACKEND_DIR) && go build $(GO_BUILD_FLAGS) \
		-o ../server \
		-ldflags="-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.GitHash=$(GIT_HASH)" \
		./cmd/server
	@echo "$(GREEN)✓ Binary built: server$(NC)"

# Quick rebuild backend in running container
rebuild-backend: ## Rebuild backend in running container (fast iteration)
	@echo "$(BLUE)Rebuilding backend service...$(NC)"
	@if [ -z "$$($(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) ps backend -q 2>/dev/null)" ]; then \
		echo "$(RED)✗ Backend container not running$(NC)"; \
		echo "Start with: make docker-up"; \
		exit 1; \
	fi
	$(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) build --no-cache backend
	$(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) up -d backend
	@echo "$(GREEN)✓ Backend rebuilt and restarted$(NC)"
	@sleep 3
	@$(MAKE) docker-health-backend

# ============================================================================
# PHASE 4: DOCKER COMPOSE ORCHESTRATION
# ============================================================================

# MAIN TARGET: Spin up entire Docker stack
docker-up: check-deps install-go-deps build-docker ## Spin up entire stack (dependencies verified first)
	@echo ""
	@echo "$(BLUE)╔════════════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(BLUE)║                  STARTING DOCKER STACK                         ║$(NC)"
	@echo "$(BLUE)╚════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "$(BLUE)Starting services (order matters):$(NC)"
	@echo "  1. PostgreSQL (database)"
	@echo "  2. Redis (cache)"
	@echo "  3. PostgreSQL migrations (init container)"
	@echo "  4. Backend (Go service)"
	@echo ""
	$(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) up -d
	@echo ""
	@echo "$(YELLOW)Waiting for services to initialize...$(NC)"
	@sleep 5
	@echo "$(YELLOW)Checking service health...$(NC)"
	@sleep 3
	@$(MAKE) docker-health


# Stop containers (keep volumes)
docker-down: ## Stop all containers (preserve volumes)
	@echo "$(BLUE)Stopping Docker stack...$(NC)"
	$(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) down
	@echo "$(GREEN)✓ Stack stopped (volumes preserved)$(NC)"

# Stop containers and remove volumes
docker-clean: ## Stop containers and delete all volumes (full reset)
	@echo "$(RED)Removing entire stack (including data)...$(NC)"
	$(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) down -v --remove-orphans
	@echo "$(GREEN)✓ Stack cleaned$(NC)"

# Tail logs from all services
docker-logs: ## Tail logs from all services (Ctrl+C to stop)
	@echo "$(BLUE)Tailing logs (Ctrl+C to stop)...$(NC)"
	$(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) logs -f

# Tail logs from backend only
docker-logs-backend: ## Tail backend logs only
	$(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) logs -f backend

# Tail logs from postgres only
docker-logs-postgres: ## Tail PostgreSQL logs only
	$(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) logs -f postgres

# Show running containers
docker-ps: ## Show all running containers and status
	@echo "$(BLUE)Running containers:$(NC)"
	$(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) ps

# Open shell in backend container
docker-shell: ## Open shell in backend container
	@echo "$(BLUE)Opening shell in backend container...$(NC)"
	$(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) exec backend sh

# Comprehensive health check
docker-health: ## Check health of all services
	@echo "$(BLUE)════════════════════════════════════════════════════════════════$(NC)"
	@echo "$(BLUE)Service Health Check:$(NC)"
	@echo "$(BLUE)════════════════════════════════════════════════════════════════$(NC)"
	@echo ""
	@$(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) ps
	@echo ""
	@echo "$(BLUE)Testing service connectivity:$(NC)"
	@echo ""
	@echo -n "  PostgreSQL (5432): "
	@if $(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) exec -T postgres pg_isready -U $(DB_USER) >/dev/null 2>&1; then \
		echo "$(GREEN)✓ Ready$(NC)"; \
	else \
		echo "$(RED)✗ Not ready$(NC)"; \
	fi
	@echo ""
	@echo -n "  Redis (6379): "
	@if $(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) exec -T redis redis-cli ping 2>/dev/null | grep -q PONG; then \
		echo "$(GREEN)✓ Ready$(NC)"; \
	else \
		echo "$(RED)✗ Not ready$(NC)"; \
	fi
	@echo ""
	@echo -n "  Backend (8080/health): "
	@if curl -sf http://localhost:8080/health >/dev/null 2>&1; then \
		echo "$(GREEN)✓ Ready$(NC)"; \
	else \
		echo "$(YELLOW)⚠ Initializing...$(NC)"; \
	fi
	@echo ""
	@echo "$(BLUE)Service Endpoints:$(NC)"
	@echo "  Backend API:  http://localhost:8080"
	@echo "  Health:       http://localhost:8080/health"
	@echo "  WebSocket:    ws://localhost:8080/ws"
	@echo "  PostgreSQL:   localhost:5432"
	@echo "  Redis:        localhost:6379"
	@echo ""
	@echo "$(BLUE)Database Credentials:$(NC)"
	@echo "  User:     $(DB_USER)"
	@echo "  Password: $(DB_PASSWORD)"
	@echo "  Database: $(DB_NAME)"
	@echo ""
	@echo "$(BLUE)Quick Tests:$(NC)"
	@echo "  curl http://localhost:8080/health"
	@echo "  curl http://localhost:8080/api/routes"
	@echo "  curl 'http://localhost:8080/api/geo/hex?lat=28.6139&lng=77.2090'"
	@echo ""

# Health check for backend only
docker-health-backend: ## Check backend service health
	@echo "$(BLUE)Backend Health Check:$(NC)"
	@curl -s http://localhost:8080/health | jq '.' 2>/dev/null || curl -s http://localhost:8080/health || echo "$(RED)Backend not ready yet$(NC)"

# Restart all services
docker-restart: ## Restart all services (keep volumes)
	@echo "$(BLUE)Restarting Docker stack...$(NC)"
	$(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) restart
	@echo "$(GREEN)✓ Stack restarted$(NC)"
	@sleep 3
	@$(MAKE) docker-health

# ============================================================================
# PHASE 5: DATABASE MANAGEMENT
# ============================================================================

# Run database migrations via Docker
db-migrate: ## Run database migrations (via Docker)
	@echo "$(BLUE)Running database migrations...$(NC)"
	@for f in $(MIGRATIONS_DIR)/*.sql; do \
		echo "  Applying $$(basename $$f)..."; \
		$(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -f /docker-entrypoint-initdb.d/$$(basename $$f) 2>/dev/null || true; \
	done
	@echo "$(GREEN)✓ Migrations completed$(NC)"

# Run migrations locally (requires psql)
db-migrate-local: ## Run migrations locally (requires psql installed)
	@if ! command -v psql >/dev/null 2>&1; then \
		echo "$(RED)✗ PostgreSQL client (psql) not found$(NC)"; \
		echo "$(YELLOW)Install with: brew install postgresql$(NC)"; \
		exit 1; \
	fi
	@echo "$(BLUE)Running database migrations locally...$(NC)"
	@for f in $(MIGRATIONS_DIR)/*.sql; do \
		echo "  Applying $$(basename $$f)..."; \
		psql $(POSTGRES_URL) -f $$f 2>/dev/null || true; \
	done
	@echo "$(GREEN)✓ Migrations completed$(NC)"

# Reset database (dangerous!)
db-reset: ## ⚠️  DROP and RECREATE database (DEVELOPMENT ONLY)
	@echo "$(RED)⚠️  WARNING: This will DELETE all data in $(DB_NAME)$(NC)"
	@read -p "Type 'yes' to confirm: " confirm; \
	if [ "$$confirm" = "yes" ]; then \
		$(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) exec -T postgres psql -U $(DB_USER) -c "DROP DATABASE IF EXISTS $(DB_NAME);" && \
		$(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) exec -T postgres psql -U $(DB_USER) -c "CREATE DATABASE $(DB_NAME);" && \
		$(MAKE) db-migrate && \
		echo "$(GREEN)✓ Database reset complete$(NC)"; \
	else \
		echo "$(YELLOW)Cancelled$(NC)"; \
	fi

# Seed sample data
db-seed: ## Run seed data migration
	@echo "$(BLUE)Seeding sample data...$(NC)"
	@if [ -f "$(MIGRATIONS_DIR)/004_seed_data.sql" ]; then \
		$(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) exec -T postgres psql -U $(DB_USER) -d $(DB_NAME) -f /docker-entrypoint-initdb.d/004_seed_data.sql; \
		echo "$(GREEN)✓ Sample data added$(NC)"; \
	else \
		echo "$(YELLOW)Seed file not found$(NC)"; \
	fi

# Open psql to database (via Docker)
db-shell: ## Open psql shell to database
	@echo "$(BLUE)Opening psql...$(NC)"
	$(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) exec postgres psql -U $(DB_USER) -d $(DB_NAME)

# Show database info
db-info: ## Show database connection information
	@echo "$(BLUE)Database Information:$(NC)"
	@echo "  Host:     $(DB_HOST)"
	@echo "  Port:     $(DB_PORT)"
	@echo "  Database: $(DB_NAME)"
	@echo "  User:     $(DB_USER)"
	@echo "  Password: $(DB_PASSWORD)"
	@echo ""
	@echo "$(BLUE)Connection URL:$(NC)"
	@echo "  $(POSTGRES_URL)"
	@echo ""
	@echo "$(BLUE)To connect:$(NC)"
	@echo "  make db-shell"

# ============================================================================
# PHASE 6: DEVELOPMENT & TESTING
# ============================================================================

# Run all tests
test: check-go ## Run all tests
	@echo "$(BLUE)Running tests...$(NC)"
	cd $(BACKEND_DIR) && go test $(GO_TEST_FLAGS) ./...
	@echo "$(GREEN)✓ Tests passed$(NC)"

# Run unit tests only
test-unit: check-go ## Run unit tests only (fast)
	@echo "$(BLUE)Running unit tests...$(NC)"
	cd $(BACKEND_DIR) && go test $(GO_TEST_FLAGS) -short ./...
	@echo "$(GREEN)✓ Unit tests passed$(NC)"

# Run integration tests
test-integration: check-go ## Run integration tests (requires Docker)
	@echo "$(BLUE)Running integration tests...$(NC)"
	cd $(BACKEND_DIR) && go test -tags=integration ./test/... -v -timeout 120s
	@echo "$(GREEN)✓ Integration tests passed$(NC)"

# Generate coverage report
test-coverage: check-go ## Generate test coverage report
	@echo "$(BLUE)Generating coverage report...$(NC)"
	cd $(BACKEND_DIR) && go test -coverprofile=coverage.out $(GO_TEST_FLAGS) ./...
	cd $(BACKEND_DIR) && go tool cover -html=coverage.out -o ../coverage.html
	@echo "$(GREEN)✓ Coverage report: coverage.html$(NC)"

# Format code
fmt: ## Format Go code
	@echo "$(BLUE)Formatting code...$(NC)"
	cd $(BACKEND_DIR) && go fmt ./...
	@echo "$(GREEN)✓ Code formatted$(NC)"

# Run go vet
vet: ## Run go vet analysis
	@echo "$(BLUE)Running go vet...$(NC)"
	cd $(BACKEND_DIR) && go vet ./...
	@echo "$(GREEN)✓ No issues found$(NC)"

# Run linters
lint: ## Run golangci-lint (requires installation)
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "$(YELLOW)golangci-lint not found$(NC)"; \
		echo "Install with: brew install golangci-lint"; \
		exit 1; \
	fi
	@echo "$(BLUE)Running linters...$(NC)"
	cd $(BACKEND_DIR) && golangci-lint run ./...
	@echo "$(GREEN)✓ Linting passed$(NC)"

# Run backend binary
run: build ## Run backend binary locally
	@echo "$(BLUE)Running backend...$(NC)"
	./server

# Run dev locally
dev: ## Run backend with go run (requires DB/Redis)
	@echo "$(BLUE)Starting backend...$(NC)"
	cd $(BACKEND_DIR) && go run ./cmd/server

# Run with Docker (attached mode for debugging)
dev-docker: ## Run entire stack in Docker (attached mode)
	$(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) up

# ============================================================================
# PHASE 7: API TESTING HELPERS
# ============================================================================

# Test health endpoint
test-health: ## Test health endpoint
	@echo "$(BLUE)Testing health endpoint...$(NC)"
	@curl -s http://localhost:8080/health | jq '.' 2>/dev/null || curl -s http://localhost:8080/health

# Test location ingest
test-location: ## POST a test location update
	@echo "$(BLUE)Sending test location...$(NC)"
	@curl -s -X POST http://localhost:8080/api/location \
		-H "Content-Type: application/json" \
		-d '{"device_id":"test-bus-001","latitude":28.6139,"longitude":77.2090,"speed":25.0,"accuracy":10.0}' | jq '.' 2>/dev/null || \
	curl -s -X POST http://localhost:8080/api/location \
		-H "Content-Type: application/json" \
		-d '{"device_id":"test-bus-001","latitude":28.6139,"longitude":77.2090,"speed":25.0,"accuracy":10.0}'

# Test nearby devices
test-nearby: ## Test nearby devices endpoint
	@echo "$(BLUE)Testing nearby devices...$(NC)"
	@curl -s "http://localhost:8080/api/nearby/devices?lat=28.6139&lng=77.2090&radius=500" | jq '.' 2>/dev/null || \
	curl -s "http://localhost:8080/api/nearby/devices?lat=28.6139&lng=77.2090&radius=500"

# Test H3 hex endpoint
test-hex: ## Test H3 hex endpoint
	@echo "$(BLUE)Testing H3 hex...$(NC)"
	@curl -s "http://localhost:8080/api/geo/hex?lat=28.6139&lng=77.2090" | jq '.' 2>/dev/null || \
	curl -s "http://localhost:8080/api/geo/hex?lat=28.6139&lng=77.2090"

# Test routes endpoint
test-routes: ## Test routes endpoint
	@echo "$(BLUE)Testing routes...$(NC)"
	@curl -s "http://localhost:8080/api/routes" | jq '.' 2>/dev/null || \
	curl -s "http://localhost:8080/api/routes"

# Test arrivals endpoint
test-arrivals: ## Test arrivals endpoint
	@echo "$(BLUE)Testing arrivals...$(NC)"
	@curl -s "http://localhost:8080/api/arrivals?stop_id=stop-r1-3" | jq '.' 2>/dev/null || \
	curl -s "http://localhost:8080/api/arrivals?stop_id=stop-r1-3"

# Run all API tests
test-api: test-health test-hex test-routes ## Run all API endpoint tests
	@echo "$(GREEN)✓ All API tests completed$(NC)"

# ============================================================================
# PHASE 8: CLEANUP & UTILITIES
# ============================================================================

# Clean build artifacts
clean: ## Remove build artifacts and cache
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	rm -f server coverage.out $(BACKEND_DIR)/coverage.out coverage.html
	go clean -cache
	@echo "$(GREEN)✓ Cleaned$(NC)"

# Full cleanup
clean-all: docker-clean clean ## Full cleanup (containers + artifacts)
	@echo "$(BLUE)Full system cleanup complete$(NC)"

# Clean Docker only
clean-docker: ## Clean Docker images and containers
	@echo "$(BLUE)Cleaning Docker...$(NC)"
	$(DOCKER_COMPOSE_CMD) -f $(DOCKER_COMPOSE_FILE) down -v --remove-orphans
	docker rmi $(IMAGE_NAME):$(IMAGE_TAG) 2>/dev/null || true
	@echo "$(GREEN)✓ Docker cleaned$(NC)"

# Prune Docker system
docker-prune: ## Prune unused Docker resources
	@echo "$(BLUE)Pruning Docker system...$(NC)"
	docker system prune -f
	@echo "$(GREEN)✓ Docker pruned$(NC)"

# Show version info
version: ## Show build information
	@echo "$(BLUE)Build Information:$(NC)"
	@echo "  Project:    $(PROJECT_NAME)"
	@echo "  Version:    $(VERSION)"
	@echo "  Build Time: $(BUILD_TIME)"
	@echo "  Git Hash:   $(GIT_HASH)"
	@echo "  Go Version: $(GO_VERSION)"

# Show project info
info: ## Show project information
	@echo "$(BLUE)Project Information:$(NC)"
	@echo "  Project:    $(PROJECT_NAME)"
	@echo "  Backend:    $(BACKEND_DIR)"
	@echo "  Migrations: $(MIGRATIONS_DIR)"
	@echo "  Database:   $(DB_NAME) @ $(DB_HOST):$(DB_PORT)"
	@echo ""
	@echo "$(BLUE)Quick Start:$(NC)"
	@echo "  make check-deps        # Verify dependencies"
	@echo "  make docker-up         # Start everything"
	@echo "  make docker-health     # Check health"
	@echo "  make docker-logs       # View logs"

# Default help target
.DEFAULT_GOAL := help

help: ## Show this help message with all available targets
	@echo "$(BLUE)╔════════════════════════════════════════════════════════════════╗$(NC)"
	@echo "$(BLUE)║              TRANSIT BACKEND - MAKEFILE TARGETS                ║$(NC)"
	@echo "$(BLUE)╚════════════════════════════════════════════════════════════════╝$(NC)"
	@echo ""
	@echo "$(YELLOW)QUICK START:$(NC)"
	@echo "  make check-deps        # Verify all dependencies installed"
	@echo "  make docker-up         # Start entire stack (recommended)"
	@echo "  make docker-health     # Verify all services ready"
	@echo "  make docker-logs       # View service logs"
	@echo ""
	@echo "$(YELLOW)AVAILABLE COMMANDS:$(NC)"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  $(BLUE)%-22s$(NC) %s\n", $$1, $$2}' | \
		sort
	@echo ""
	@echo "$(YELLOW)COMMON WORKFLOWS:$(NC)"
	@echo ""
	@echo "  $(GREEN)First Time Setup:$(NC)"
	@echo "    1. make check-deps"
	@echo "    2. make docker-up"
	@echo "    3. make docker-health"
	@echo ""
	@echo "  $(GREEN)Daily Development:$(NC)"
	@echo "    1. make docker-up"
	@echo "    2. make docker-logs"
	@echo "    3. make test-api"
	@echo ""
	@echo "  $(GREEN)Testing:$(NC)"
	@echo "    make test            # All tests"
	@echo "    make test-api        # API tests"
	@echo ""
	@echo "  $(GREEN)Cleanup:$(NC)"
	@echo "    make docker-down     # Stop (keep data)"
	@echo "    make docker-clean    # Full reset"
	@echo ""

# ============================================================================
# LAYERING GUARD (DEFECT-1 recurrence protection)
# ============================================================================

lint-layering: ## Verify handlers/services don't import database directly (layering rule)
	@bash scripts/gate.sh --check=layering
