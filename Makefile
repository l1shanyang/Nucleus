APP_NAME    := nucleus-api
MODULE      := nucleus
include versions.env
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

.PHONY: help versions version-check deps db-up db-down migrate-up migrate-down sqlc-gen \
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
	@echo "PostgreSQL: $(POSTGRES_VERSION)"

version-check: ## Verify local and checked-in tool versions match versions.env
	@test "$$(go env GOVERSION)" = "go$(GO_VERSION)" || \
		(echo "Go version mismatch: expected go$(GO_VERSION), got $$(go env GOVERSION)"; exit 1)
	@grep -q '^go 1.26.0$$' go.mod || \
		(echo "go.mod language version mismatch: expected go 1.26.0"; exit 1)
	@grep -q '^ARG GO_VERSION=$(GO_VERSION)$$' Dockerfile || \
		(echo "Dockerfile GO_VERSION mismatch: expected $(GO_VERSION)"; exit 1)
	@grep -q 'image: golang:$(GO_VERSION)' docker-compose.yml || \
		(echo "docker-compose Go image mismatch: expected golang:$(GO_VERSION)"; exit 1)
	@grep -q 'image: migrate/migrate:$(MIGRATE_IMAGE_VERSION)' docker-compose.yml || \
		(echo "docker-compose migrate image mismatch: expected migrate/migrate:$(MIGRATE_IMAGE_VERSION)"; exit 1)
	@grep -q 'image: postgres:$(POSTGRES_VERSION)' docker-compose.yml || \
		(echo "docker-compose PostgreSQL image mismatch: expected postgres:$(POSTGRES_VERSION)"; exit 1)
	@grep -q 'image: postgres:$(POSTGRES_VERSION)' .github/workflows/ci.yml || \
		(echo "CI PostgreSQL image mismatch: expected postgres:$(POSTGRES_VERSION)"; exit 1)

# ---------------------------------------------------------------------------
# Dependencies
# ---------------------------------------------------------------------------

deps: version-check ## Install pinned local tools
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

migrate-up: version-check ## Apply all migrations with pinned migrate
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION) \
		-path $(MIGRATION_DIR) -database "$(DB_URL)" up

migrate-down: version-check ## Rollback one migration with pinned migrate
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION) \
		-path $(MIGRATION_DIR) -database "$(DB_URL)" down 1

# ---------------------------------------------------------------------------
# Code generation
# ---------------------------------------------------------------------------

sqlc-gen: version-check ## Regenerate Go code from SQL queries with pinned sqlc
	go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION) generate -f $(SQLC_CONFIG)

# ---------------------------------------------------------------------------
# Build & Run
# ---------------------------------------------------------------------------

build: version-check ## Build the API binary to bin/
	@mkdir -p bin
	go build -ldflags '$(LDFLAGS)' -o bin/$(APP_NAME) ./cmd/api
	@echo "Built: bin/$(APP_NAME) ($(VERSION), $(COMMIT))"

run: version-check ## Run API server locally
	go run ./cmd/api

docker-build: version-check ## Build production Docker image
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

test: version-check ## Run all tests
	go test -v -count=1 ./...

cover: version-check ## Run tests with coverage report
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1
	@echo "Open HTML report: go tool cover -html=coverage.out"

fmt: ## Format all Go files
	gofmt -w $$(find . -name '*.go' -type f)

lint: version-check ## Run pinned golangci-lint
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run

vuln: version-check ## Check for known vulnerabilities
	go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

check: version-check fmt lint test vuln ## Run version-check + fmt + lint + test + vuln

tidy: version-check ## Tidy and verify go.mod
	go mod tidy
	go mod verify

clean: ## Remove build artifacts
	rm -rf bin/
	rm -f *.out coverage.html
	@echo "Cleaned"
