.PHONY: help dev-up dev-down run-server run-web build build-cli build-server test test-web lint lint-web audit-web migrate-up migrate-down clean

BINARY_SERVER=bin/atlas-server
BINARY_CLI=bin/atlas
GO_MODULE=github.com/nesbite/atlas
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

# --- Development ---

dev-up: ## Start PostgreSQL via Docker Compose
	docker compose -f deploy/docker-compose.yml up -d

dev-down: ## Stop PostgreSQL
	docker compose -f deploy/docker-compose.yml down

run-server: ## Run the API server
	go run ./cmd/atlas-server

run-web: ## Run the frontend dev server
	cd web && pnpm dev

# --- Build ---

build: build-server build-cli ## Build all binaries

build-server: ## Build the API server
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY_SERVER) ./cmd/atlas-server

build-cli: ## Build the CLI
	go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY_CLI) ./cmd/atlas

# --- Test ---

test: ## Run all Go tests
	go test ./... -v -race -count=1

test-web: ## Run frontend tests
	cd web && pnpm test

# --- Lint ---

lint: ## Lint Go code
	golangci-lint run ./...

lint-web: ## Lint frontend code
	cd web && pnpm lint

# --- Security ---

audit-web: ## Audit frontend dependencies for vulnerabilities
	cd web && pnpm audit --audit-level=high

# --- Database ---

migrate-up: ## Run database migrations
	@echo "TODO: implement with golang-migrate"

migrate-down: ## Rollback last migration
	@echo "TODO: implement with golang-migrate"

# --- Cleanup ---

clean: ## Remove build artifacts
	rm -rf bin/ dist/
	cd web && rm -rf dist/ node_modules/
