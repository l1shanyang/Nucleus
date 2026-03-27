APP_NAME := nucleus-api
MIGRATION_DIR := sql/migrations
SQLC_CONFIG := sqlc.yaml
DB_URL ?= postgres://nucleus:nucleus@localhost:5432/nucleus?sslmode=disable
DOCKER_DB_URL ?= postgres://nucleus:nucleus@db:5432/nucleus?sslmode=disable

.PHONY: help deps db-up db-down migrate-up migrate-down sqlc-gen run test fmt lint

help:
	@echo "Available targets:"
	@echo "  make deps          - install local tools (sqlc, migrate)"
	@echo "  make db-up         - start PostgreSQL by docker compose"
	@echo "  make db-down       - stop PostgreSQL"
	@echo "  make migrate-up    - apply migrations (local migrate or docker fallback)"
	@echo "  make migrate-down  - rollback one migration (local migrate or docker fallback)"
	@echo "  make sqlc-gen      - generate Go code from SQL"
	@echo "  make run           - run API server (local go or docker fallback)"
	@echo "  make test          - run tests"
	@echo "  make fmt           - gofmt all Go files"
	@echo "  make lint          - run go vet"

deps:
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

db-up:
	docker compose up -d db

db-down:
	docker compose down

migrate-up:
	@if command -v migrate >/dev/null 2>&1; then \
		echo "Using local migrate binary"; \
		migrate -path $(MIGRATION_DIR) -database "$(DB_URL)" up; \
	else \
		echo "Local migrate not found, using docker compose migrate service"; \
		docker compose run --rm migrate -path=/migrations -database "$(DOCKER_DB_URL)" up; \
	fi

migrate-down:
	@if command -v migrate >/dev/null 2>&1; then \
		echo "Using local migrate binary"; \
		migrate -path $(MIGRATION_DIR) -database "$(DB_URL)" down 1; \
	else \
		echo "Local migrate not found, using docker compose migrate service"; \
		docker compose run --rm migrate -path=/migrations -database "$(DOCKER_DB_URL)" down 1; \
	fi

sqlc-gen:
	sqlc generate -f $(SQLC_CONFIG)

run:
	@if command -v go >/dev/null 2>&1; then \
		echo "Using local Go toolchain"; \
		go run ./cmd/api; \
	else \
		echo "Local Go not found, using docker compose api service"; \
		docker compose up api; \
	fi

test:
	go test ./...

fmt:
	gofmt -w $$(find . -name '*.go' -type f)

lint:
	go vet ./...
