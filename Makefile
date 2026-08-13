APP=unified-document-viewer

# Override if needed, e.g. make tools GO=/path/to/go
GO ?= go
GOINSTALL ?= $(GO) install
GOTOOLCHAIN ?= go1.26.5
export GOTOOLCHAIN

TOOLS_DIR ?= $(CURDIR)/bin
export GOBIN := $(TOOLS_DIR)
export PATH := $(TOOLS_DIR):$(PATH)

ifeq ($(OS),Windows_NT)
EXE := .exe
else
EXE :=
endif

COMPOSE=deployments/docker/docker-compose.yaml

MIGRATIONS_DIR=migrations
DATABASE_URL ?= postgres://viewer:viewer@localhost:5432/viewer?sslmode=disable

GOLANGCI_LINT=$(TOOLS_DIR)/golangci-lint$(EXE)
MIGRATE=$(TOOLS_DIR)/migrate$(EXE)
MOCKGEN=$(TOOLS_DIR)/mockgen$(EXE)
SWAG=$(TOOLS_DIR)/swag$(EXE)

DOCKER_IMAGE ?= $(APP):local
VERSION ?= dev

# Ports for the two mock upstreams (A5 - separate servers)
SALES_ADDR ?= :9100
SERVICE_ADDR ?= :9101

# Directories swag scans for annotations: general info first, then module surfaces.
SWAG_DIRS = cmd/api,internal/modules/documents/api,internal/shared/infra/httpserver

.DEFAULT_GOAL := help

.PHONY: help tools run mocks dev demo-degraded \
	test test-race test-integration cover vet fmt tidy lint generate \
	openapi openapi-check \
	new-migrate migrate-up migrate-down \
	db-up db-down db-reset build clean

# ---------------------------------------------------------------------------
# Help - this Makefile is the single source of truth for commands.
# Every target carries its own '##' description; `make help` prints them.
# ---------------------------------------------------------------------------

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  %-17s %s%s", $$1, $$2, ORS}' $(MAKEFILE_LIST)

# ---------------------------------------------------------------------------
# Tools - installed on demand into ./bin, never into the user's GOPATH
# ---------------------------------------------------------------------------

tools: $(GOLANGCI_LINT) $(MIGRATE) $(MOCKGEN) $(SWAG) ## Install golangci-lint, migrate, mockgen, swag into ./bin

$(GOLANGCI_LINT):
	$(GOINSTALL) github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

$(MIGRATE):
	$(GOINSTALL) -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

$(MOCKGEN):
	$(GOINSTALL) go.uber.org/mock/mockgen@latest

$(SWAG):
	$(GOINSTALL) github.com/swaggo/swag/v2/cmd/swag@latest

# ---------------------------------------------------------------------------
# Run / develop
# ---------------------------------------------------------------------------

run: ## Run the API alone (expects db + mocks already up)
	$(GO) run ./cmd/api

# Both mock upstreams, foreground. Use a second terminal, or use `make dev`.
mocks: ## Run both mock upstreams in the foreground
	$(GO) run ./cmd/mock-sales -addr $(SALES_ADDR) & \
	$(GO) run ./cmd/mock-service -addr $(SERVICE_ADDR) & \
	wait

# The one command a reviewer needs: database, schema, mocks, API.
dev: db-up migrate-up ## db + migrations + both mocks + API. The one command to run the system
	$(GO) run ./cmd/mock-sales -addr $(SALES_ADDR) & \
	$(GO) run ./cmd/mock-service -addr $(SERVICE_ADDR) & \
	$(GO) run ./cmd/api

# Same, with the Service upstream forced down, to demonstrate FR7 partial results.
demo-degraded: db-up migrate-up ## Same as dev but with the Service upstream down (demonstrates FR7)
	$(GO) run ./cmd/mock-sales -addr $(SALES_ADDR) & \
	$(GO) run ./cmd/mock-service -addr $(SERVICE_ADDR) -down & \
	$(GO) run ./cmd/api

# ---------------------------------------------------------------------------
# Test / quality
# ---------------------------------------------------------------------------

test: ## Unit tests
	$(GO) test ./...

# The gate for concurrency work: DD-2 failure isolation, FR3 parallelism.
test-race: ## Unit tests under -race. Gate for anything concurrent
	$(GO) test -race ./...

# Needs a running database: make db-up migrate-up first.
test-integration: ## Integration tests (build tag), needs a live database
	$(GO) test -tags=integration ./internal/...

cover: ## Coverage profile and per-function summary
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

vet: ## go vet
	$(GO) vet ./...

fmt: ## Format all Go files in place
	gofmt -w $$(find . -name '*.go' -not -path './bin/*')

tidy: ## go mod tidy
	$(GO) mod tidy

lint: $(GOLANGCI_LINT) ## golangci-lint run ./...
	$(GOLANGCI_LINT) run ./...

generate: $(MOCKGEN) ## Run go:generate (mockgen)
	GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) generate ./internal/...

# ---------------------------------------------------------------------------
# OpenAPI - annotations become api/http/docs/{docs.go,swagger.yaml,swagger.json}
# ---------------------------------------------------------------------------

openapi: $(SWAG) ## Regenerate api/http/docs from handler annotations
	$(SWAG) init --v3.1 -g docs.go -d $(SWAG_DIRS) -o ./api/http/docs --outputTypes go,yaml,json

# CI gate: fails if the committed contract has drifted from the handlers.
openapi-check: openapi ## Fail if the committed contract has drifted from the code
	@git diff --exit-code -- api/http/docs/docs.go api/http/docs/swagger.yaml api/http/docs/swagger.json \
		|| (echo "openapi out of date; run make openapi and commit" && exit 1)

# ---------------------------------------------------------------------------
# Database migrations
# ---------------------------------------------------------------------------

# usage: make new-migrate name=add_document_cache
new-migrate: $(MIGRATE) ## Create a migration pair. usage: make new-migrate name=add_foo
ifndef name
	$(error usage: make new-migrate name=<migration_name>)
endif
	$(MIGRATE) create -ext sql -dir $(MIGRATIONS_DIR) -seq $(name)

migrate-up: $(MIGRATE) ## Apply all pending migrations
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" up

migrate-down: $(MIGRATE) ## Roll back one migration
	$(MIGRATE) -path $(MIGRATIONS_DIR) -database "$(DATABASE_URL)" down 1

# ---------------------------------------------------------------------------
# Database container
# ---------------------------------------------------------------------------

db-up: ## Start PostgreSQL and wait for its healthcheck
	docker compose -f $(COMPOSE) up -d --wait

db-down: ## Stop PostgreSQL
	docker compose -f $(COMPOSE) down

# Drops the volume too - use when a migration needs a clean slate.
db-reset: ## Stop PostgreSQL, drop the volume, start clean
	docker compose -f $(COMPOSE) down -v
	docker compose -f $(COMPOSE) up -d --wait

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------

build: ## Build all three binaries into ./bin
	$(GO) build -ldflags "-X main.version=$(VERSION)" -o bin/api$(EXE) ./cmd/api
	$(GO) build -o bin/mock-sales$(EXE) ./cmd/mock-sales
	$(GO) build -o bin/mock-service$(EXE) ./cmd/mock-service

clean: ## Remove ./bin and coverage output
	rm -rf bin coverage.out
