APP := nucleus-api
MODULE := nucleus

DB_URL ?= postgres://nucleus:nucleus@localhost:5432/nucleus?sslmode=disable
MIGRATIONS := sql/migrations
SQLC_CONFIG := sqlc.yaml

GO_VERSION := 1.26.4
GOLANGCI_LINT_VERSION := v2.12.2
SQLC_VERSION := v1.31.1
MIGRATE_VERSION := v4.19.1
GOVULNCHECK_VERSION := v1.3.0

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -s -w \
	-X $(MODULE)/internal/version.Version=$(VERSION) \
	-X $(MODULE)/internal/version.Commit=$(COMMIT) \
	-X $(MODULE)/internal/version.BuildTime=$(BUILD_TIME)

LINT := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
MIGRATE := go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION)
SQLC := go run github.com/sqlc-dev/sqlc/cmd/sqlc@$(SQLC_VERSION)
VULN := go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

.PHONY: help version dev db down migrate rollback sqlc run build docker test it fmt lint check clean wait-db
.PHONY: versions db-up db-down migrate-up migrate-down sqlc-gen docker-build test-integration tidy vuln

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*## "; print "Usage: make <target>\n"} /^[a-zA-Z_-]+:.*## / {printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

version: ## Print tool versions
	@echo "Go: $(GO_VERSION)"
	@echo "golangci-lint: $(GOLANGCI_LINT_VERSION)"
	@echo "sqlc: $(SQLC_VERSION)"
	@echo "migrate: $(MIGRATE_VERSION)"
	@echo "govulncheck: $(GOVULNCHECK_VERSION)"

dev: db wait-db migrate run ## Start DB, migrate, then run API

db: ## Start PostgreSQL
	docker compose up -d db

wait-db:
	@printf "Waiting for database"
	@until docker compose exec -T db pg_isready -U nucleus -d nucleus >/dev/null 2>&1; do \
		printf "."; \
		sleep 1; \
	done
	@echo " ready"

down: ## Stop Docker Compose services
	docker compose down

migrate: ## Apply database migrations
	$(MIGRATE) -path $(MIGRATIONS) -database "$(DB_URL)" up

rollback: ## Roll back one database migration
	$(MIGRATE) -path $(MIGRATIONS) -database "$(DB_URL)" down 1

sqlc: ## Regenerate sqlc code
	$(SQLC) generate -f $(SQLC_CONFIG)

run: ## Run API locally
	go run ./cmd/api

build: ## Build API binary
	@mkdir -p bin
	go build -ldflags '$(LDFLAGS)' -o bin/$(APP) ./cmd/api
	@echo "Built: bin/$(APP) ($(VERSION), $(COMMIT))"

docker: ## Build production Docker image
	docker build \
		--build-arg GO_VERSION=$(GO_VERSION) \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		-t $(APP):$(VERSION) \
		-t $(APP):latest \
		.

test: ## Run unit tests
	go test -v -count=1 ./...

it: ## Run integration tests
	@if [ -z "$(TEST_DATABASE_URL)" ]; then \
		echo "TEST_DATABASE_URL is required"; \
		echo "Example: TEST_DATABASE_URL=postgres://user:pass@localhost:5432/nucleus_test?sslmode=disable make it"; \
		exit 1; \
	fi
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -v -count=1 -tags=integration ./...

fmt: ## Format Go files
	gofmt -w $$(find . -name '*.go' -type f)

lint: ## Run linter
	$(LINT) run

check: fmt lint test ## Run local quality gate
	$(VULN) ./...

clean: ## Remove build artifacts
	rm -rf bin/
	rm -f *.out coverage.html

# Hidden compatibility aliases for older docs/sessions.
versions: version
db-up: db
db-down: down
migrate-up: migrate
migrate-down: rollback
sqlc-gen: sqlc
docker-build: docker
test-integration: it

tidy:
	go mod tidy
	go mod verify

vuln:
	$(VULN) ./...
