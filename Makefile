# Game Development VCS Makefile

.PHONY: help build build-cli build-server run-server run-cli clean test deps docker-build docker-run integration-test

# Variables
BINARY_NAME_CLI=vcs
BINARY_NAME_SERVER=vcs-server
BUILD_DIR=build
VERSION=1.0.0
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
REGISTRY=yourstudio

# Default target
help: ## Show this help message
	@echo "Game Development VCS Build System"
	@echo ""
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Dependencies
deps: ## Download Go dependencies
	@echo "📦 Downloading dependencies..."
	go mod download
	go mod tidy

# Build targets
build: build-cli build-server ## Build both CLI and server

build-cli: deps ## Build the CLI tool
	@echo "🔨 Building CLI..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_CLI) ./cmd/vcs
	@echo "✅ CLI built: $(BUILD_DIR)/$(BINARY_NAME_CLI)"

build-server: deps ## Build the server
	@echo "🔨 Building server..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_SERVER) ./cmd/vcs-server
	@echo "✅ Server built: $(BUILD_DIR)/$(BINARY_NAME_SERVER)"

# Cross-compilation targets
build-all-platforms: deps ## Build for all platforms
	@echo "🔨 Building for all platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Windows
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_CLI)-windows-amd64.exe ./cmd/vcs
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_SERVER)-windows-amd64.exe ./cmd/vcs-server
	
	# macOS
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_CLI)-darwin-amd64 ./cmd/vcs
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_SERVER)-darwin-amd64 ./cmd/vcs-server
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_CLI)-darwin-arm64 ./cmd/vcs
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_SERVER)-darwin-arm64 ./cmd/vcs-server
	
	# Linux
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_CLI)-linux-amd64 ./cmd/vcs
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_SERVER)-linux-amd64 ./cmd/vcs-server
	
	@echo "✅ All platforms built"

# Run targets
run-server: build-server ## Run the server locally
	@echo "🚀 Starting VCS server..."
	./$(BUILD_DIR)/$(BINARY_NAME_SERVER)

run-cli: build-cli ## Run the CLI (show help)
	@echo "🖥️  VCS CLI:"
	./$(BUILD_DIR)/$(BINARY_NAME_CLI) --help

# Development targets
dev-server: ## Run server in development mode
	@echo "🔧 Running server in development mode..."
	go run ./cmd/vcs-server

dev-cli: ## Run CLI in development mode
	@echo "🔧 Running CLI in development mode..."
	go run ./cmd/vcs --help

# Testing
test: ## Run unit tests
	@echo "🧪 Running unit tests..."
	go test -v ./...

test-coverage: ## Run tests with coverage
	@echo "🧪 Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report: coverage.html"

integration-test: build ## Run integration tests
	@echo "🔧 Running integration tests..."
	@chmod +x scripts/test-integration.sh
	./scripts/test-integration.sh

test-all: test integration-test ## Run all tests

# Development environment
dev-up: ## Start development environment with Docker Compose
	@echo "🚀 Starting development environment..."
	@chmod +x scripts/deploy.sh
	./scripts/deploy.sh dev

dev-down: ## Stop development environment
	@echo "🛑 Stopping development environment..."
	docker-compose -f docker/docker-compose.dev.yml down

dev-logs: ## Show development environment logs
	@echo "📋 Development environment logs..."
	docker-compose -f docker/docker-compose.dev.yml logs -f

dev-clean: ## Clean development environment
	@echo "🧹 Cleaning development environment..."
	docker-compose -f docker/docker-compose.dev.yml down -v
	docker system prune -f

dev-reset: dev-clean dev-up ## Reset development environment

# Docker targets
docker-build: ## Build Docker images
	@echo "🐳 Building Docker images..."
	docker build -f docker/Dockerfile.server -t $(REGISTRY)/vcs-server:$(VERSION) .
	docker build -f docker/Dockerfile.cli -t $(REGISTRY)/vcs-cli:$(VERSION) .
	docker tag $(REGISTRY)/vcs-server:$(VERSION) $(REGISTRY)/vcs-server:latest
	docker tag $(REGISTRY)/vcs-cli:$(VERSION) $(REGISTRY)/vcs-cli:latest

docker-push: docker-build ## Push Docker images to registry
	@echo "🐳 Pushing Docker images..."
	docker push $(REGISTRY)/vcs-server:$(VERSION)
	docker push $(REGISTRY)/vcs-server:latest
	docker push $(REGISTRY)/vcs-cli:$(VERSION)
	docker push $(REGISTRY)/vcs-cli:latest

docker-run: docker-build ## Run server in Docker
	@echo "🐳 Running server in Docker..."
	docker run -p 8080:8080 -e ENVIRONMENT=development $(REGISTRY)/vcs-server:$(VERSION)

# Production deployment
deploy-prod: ## Deploy to production Kubernetes
	@echo "🚀 Deploying to production..."
	./scripts/deploy.sh deploy -e production

deploy-staging: ## Deploy to staging Kubernetes
	@echo "🚀 Deploying to staging..."
	./scripts/deploy.sh deploy -e staging

deploy-status: ## Show deployment status
	@echo "📊 Deployment status..."
	./scripts/deploy.sh status

deploy-destroy: ## Destroy deployment
	@echo "💥 Destroying deployment..."
	./scripts/deploy.sh destroy

# Kubernetes helpers
k8s-logs: ## Show Kubernetes logs
	kubectl logs -f deployment/vcs-api -n vcs-system

k8s-shell: ## Get shell in VCS API pod
	kubectl exec -it deployment/vcs-api -n vcs-system -- /bin/sh

k8s-port-forward: ## Port forward to local machine
	kubectl port-forward service/vcs-api 8080:8080 -n vcs-system

# Backup and restore
backup: ## Create backup
	@echo "💾 Creating backup..."
	./scripts/deploy.sh backup

# Utilities
clean: ## Clean build artifacts
	@echo "🧹 Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	go clean

format: ## Format Go code
	@echo "🎨 Formatting code..."
	go fmt ./...

lint: ## Run linter
	@echo "🔍 Running linter..."
	golangci-lint run

mod-tidy: ## Tidy Go modules
	@echo "📦 Tidying modules..."
	go mod tidy

# Installation
install-cli: build-cli ## Install CLI to system PATH
	@echo "📦 Installing CLI..."
	sudo cp $(BUILD_DIR)/$(BINARY_NAME_CLI) /usr/local/bin/
	@echo "✅ CLI installed to /usr/local/bin/$(BINARY_NAME_CLI)"

uninstall-cli: ## Uninstall CLI from system PATH
	@echo "🗑️  Uninstalling CLI..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME_CLI)
	@echo "✅ CLI uninstalled"

# Quick development workflow
quick-start: clean build dev-up ## Quick start: clean, build, and start dev environment
	@echo "⚡ Quick start completed!"
	@echo "🌐 Server: http://localhost:8080"
	@echo "🧪 Test with: make integration-test"

quick-test: build integration-test ## Quick test: build and run integration tests

demo: quick-start ## Start demo environment and run integration tests
	@echo "⏳ Waiting for services to start..."
	@sleep 10
	@echo "🧪 Running integration tests..."
	@make integration-test || true
	@echo ""
	@echo "🎮 Demo environment is ready!"
	@echo "🌐 VCS API: http://localhost:8080"
	@echo "🗄️  ClickHouse: http://localhost:8123"
	@echo "📊 Redis Commander: http://localhost:8082"
	@echo ""
	@echo "Try these commands:"
	@echo "  ./$(BUILD_DIR)/$(BINARY_NAME_CLI) init demo-project"
	@echo "  ./$(BUILD_DIR)/$(BINARY_NAME_CLI) add README.md"
	@echo "  ./$(BUILD_DIR)/$(BINARY_NAME_CLI) status --team"

# Monitoring and debugging
logs: ## Show all development logs
	@echo "📋 Showing all logs..."
	docker-compose -f docker/docker-compose.dev.yml logs --tail=100

health-check: ## Check health of all services
	@echo "🩺 Checking service health..."
	@echo "API Server:"
	@curl -f http://localhost:8080/health 2>/dev/null | jq '.' || echo "❌ API Server not responding"
	@echo ""
	@echo "Redis:"
	@docker-compose -f docker/docker-compose.dev.yml exec -T redis redis-cli ping || echo "❌ Redis not responding"
	@echo ""
	@echo "ClickHouse:"
	@curl -f http://localhost:8123/ping 2>/dev/null || echo "❌ ClickHouse not responding"

debug-storage: ## Show storage debugging info
	@echo "💾 Storage debugging info..."
	@echo "Storage stats:"
	@curl -s http://localhost:8080/api/v1/system/storage/stats | jq '.' || echo "❌ Cannot get storage stats"

debug-presence: ## Show team presence debugging info
	@echo "👥 Team presence debugging info..."
	@curl -s "http://localhost:8080/api/v1/collaboration/default/presence" | jq '.' || echo "❌ Cannot get presence info"

# Performance testing
perf-test: build ## Run performance tests
	@echo "🚀 Running performance tests..."
	@echo "Creating large test file..."
	@dd if=/dev/zero of=/tmp/large-test-file.bin bs=1M count=10 2>/dev/null
	@echo "Testing large file upload..."
	@time ./$(BUILD_DIR)/$(BINARY_NAME_CLI) add /tmp/large-test-file.bin || true
	@rm -f /tmp/large-test-file.bin
	@echo "✅ Performance test completed"

# Show version
version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Go version: $(shell go version)"
	@echo "Git commit: $(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
	@echo "Build date: $(shell date -u '+%Y-%m-%d %H:%M:%S UTC')"

# Development shortcuts
start: dev-up ## Alias for dev-up
stop: dev-down ## Alias for dev-down
restart: dev-reset ## Alias for dev-reset
check: health-check ## Alias for health-check

# All-in-one targets
all: clean deps build test integration-test ## Build everything and run all tests
ci: clean deps build test ## CI pipeline: clean, deps, build, test
release: clean deps build-all-platforms docker-build ## Prepare release: build all platforms and Docker images