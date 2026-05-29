APP_NAME    := nucleus-api
MODULE      := nucleus
MIGRATION_DIR := sql/migrations
SQLC_CONFIG := sqlc.yaml
DB_URL      ?= postgres://nucleus:nucleus@localhost:5432/nucleus?sslmode=disable
DOCKER_DB_URL ?= postgres://nucleus:nucleus@db:5432/nucleus?sslmode=disable
GO_VERSION := 1.26
GOLANGCI_LINT_VERSION := v2.12.2
SQLC_VERSION := v1.31.1
MIGRATE_VERSION := v4.19.1
GOVULNCHECK_VERSION := v1.3.0

# Build info
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT      ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME  ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS     := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.BuildTime=$(BUILD_TIME)

.PHONY: help versions deps db-up db-down migrate-up migrate-down sqlc-gen \
	run build docker-build test cover fmt lint vuln check tidy clean

help: ## Show this help
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

versions: ## Print pinned tool versions
	@echo "Go: $(GO_VERSION)"
	@echo "golangci-lint: $(GOLANGCI_LINT_VERSION)"
	@echo "sqlc: $(SQLC_VERSION)"
	@echo "migrate: $(MIGRATE_VERSION)"
	@echo "govulncheck: $(GOVULNCHECK_VERSION)"

# ---------------------------------------------------------------------------
# Dependencies
# ---------------------------------------------------------------------------

deps: ## Install pinned local tools
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION)
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

# ---------------------------------------------------------------------------
# Database
# ---------------------------------------------------------------------------

db-up: ## Start PostgreSQL via Docker Compose
	docker compose up -d db

db-down: ## Stop all Docker Compose services
	docker compose down

migrate-up: ## Apply all migrations with pinned migrate
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION) \
		-path $(MIGRATION_DIR) -database "$(DB_URL)" up

migrate-down: ## Rollback one migration with pinned migrate
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION) \
		-path $(MIGRATION_DIR) -database "$(DB_URL)" down 1

# ---------------------------------------------------------------------------
# Code generation
# ---------------------------------------------------------------------------

sqlc-gen: ## Regenerate Go code from SQL queries with pinned sqlc
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION) generate -f $(SQLC_CONFIG)

# ---------------------------------------------------------------------------
# Build & Run
# ---------------------------------------------------------------------------

build: ## Build the API binary to bin/
	@mkdir -p bin
	go build -ldflags '$(LDFLAGS)' -o bin/$(APP_NAME) ./cmd/api
	@echo "Built: bin/$(APP_NAME) ($(VERSION), $(COMMIT))"

run: ## Run API server locally
	go run ./cmd/api

docker-build: ## Build production Docker image
	docker build \
		--build-arg GO_VERSION=$(GO_VERSION) \
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

lint: ## Run pinned golangci-lint
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run

vuln: ## Check for known vulnerabilities
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

check: fmt lint test vuln ## Run fmt + lint + test + vuln

tidy: ## Tidy and verify go.mod
	go mod tidy
	go mod verify

clean: ## Remove build artifacts
	rm -rf bin/
	rm -f *.out coverage.html
	@echo "Cleaned"
