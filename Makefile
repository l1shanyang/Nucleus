APP_NAME    := nucleus-api
MODULE      := nucleus
MIGRATION_DIR := sql/migrations
SQLC_CONFIG := sqlc.yaml
DB_URL      ?= postgres://nucleus:nucleus@localhost:5432/nucleus?sslmode=disable
DOCKER_DB_URL ?= postgres://nucleus:nucleus@db:5432/nucleus?sslmode=disable

# Build info
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME  ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS     := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.BuildTime=$(BUILD_TIME)

.PHONY: help deps db-up db-down migrate-up migrate-down sqlc-gen \
	run build docker-build test cover fmt lint vuln check tidy clean

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

# ---------------------------------------------------------------------------
# Dependencies
# ---------------------------------------------------------------------------

deps: ## Install local tools (sqlc, migrate, golangci-lint)
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# ---------------------------------------------------------------------------
# Database
# ---------------------------------------------------------------------------

db-up: ## Start PostgreSQL via Docker Compose
	docker compose up -d db

db-down: ## Stop all Docker Compose services
	docker compose down

migrate-up: ## Apply all migrations
	@if command -v migrate >/dev/null 2>&1; then \
		echo "Using local migrate binary"; \
		migrate -path $(MIGRATION_DIR) -database "$(DB_URL)" up; \
	else \
		echo "Local migrate not found, using docker compose migrate service"; \
		docker compose run --rm migrate -path=/migrations -database "$(DOCKER_DB_URL)" up; \
	fi

migrate-down: ## Rollback one migration
	@if command -v migrate >/dev/null 2>&1; then \
		echo "Using local migrate binary"; \
		migrate -path $(MIGRATION_DIR) -database "$(DB_URL)" down 1; \
	else \
		echo "Local migrate not found, using docker compose migrate service"; \
		docker compose run --rm migrate -path=/migrations -database "$(DOCKER_DB_URL)" down 1; \
	fi

# ---------------------------------------------------------------------------
# Code generation
# ---------------------------------------------------------------------------

sqlc-gen: ## Regenerate Go code from SQL queries
	sqlc generate -f $(SQLC_CONFIG)

# ---------------------------------------------------------------------------
# Build & Run
# ---------------------------------------------------------------------------

build: ## Build the API binary to bin/
	@mkdir -p bin
	go build -ldflags '$(LDFLAGS)' -o bin/$(APP_NAME) ./cmd/api
	@echo "Built: bin/$(APP_NAME) ($(VERSION), $(COMMIT))"

run: ## Run API server locally (or via Docker if Go not found)
	@if command -v go >/dev/null 2>&1; then \
		echo "Using local Go toolchain"; \
		go run ./cmd/api; \
	else \
		echo "Local Go not found, using docker compose api service"; \
		docker compose up api; \
	fi

docker-build: ## Build production Docker image
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(APP_NAME):$(VERSION) \
		-t $(APP_NAME):latest \
		.
	@echo "Built image: $(APP_NAME):$(VERSION)"

# ---------------------------------------------------------------------------
# Quality
# ---------------------------------------------------------------------------

test: ## Run all tests
	go test -v -count=1 ./...

cover: ## Run tests with coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1
	@echo "Open HTML report: go tool cover -html=coverage.out"

fmt: ## Format all Go files
	gofmt -w $$(find . -name '*.go' -type f)

lint: ## Run go vet
	go vet ./...

vuln: ## Check for known vulnerabilities
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	govulncheck ./...

check: fmt lint test vuln ## Run fmt + lint + test + vuln (full quality gate)

tidy: ## Tidy and verify go.mod
	go mod tidy
	go mod verify

clean: ## Remove build artifacts
	rm -rf bin/
	rm -f *.out coverage.html
	@echo "Cleaned"
