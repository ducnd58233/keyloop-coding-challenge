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

COMPOSE_DIR=deployments/docker
COMPOSE_INFRA=$(COMPOSE_DIR)/docker-compose.infra.yaml
COMPOSE_ALL=$(COMPOSE_DIR)/docker-compose.yaml
COMPOSE_PROJECT=$(APP)

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

SWAG_DOCUMENTVIEWER = cmd/documentviewer,internal/documentviewer/modules/documents/api,internal/shared/infra/httpserver
SWAG_SALES = cmd/sales,internal/sales/modules/sales
SWAG_SERVICE = cmd/service,internal/service/modules/service

.DEFAULT_GOAL := help

.PHONY: help tools run mocks dev dev-stop demo-degraded \
	test test-race test-integration cover vet fmt tidy lint hook-test \
	generate mocks-gen mocks-gen-documentviewer mocks-gen-sales mocks-gen-service \
	openapi openapi-check openapi-documentviewer openapi-sales openapi-service \
	new-migrate migrate-up migrate-down \
	infra-up infra-down stack-up stack-down \
	db-up db-down db-reset build clean

# ---------------------------------------------------------------------------
# Help - this Makefile is the single source of truth for commands.
# Every target carries its own '##' description; `make help` prints them.
# ---------------------------------------------------------------------------

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  %-22s %s%s", $$1, $$2, ORS}' $(MAKEFILE_LIST)

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

run: ## Run the document-viewer API alone (expects db + mocks already up)
	$(GO) run ./cmd/documentviewer

# Built binaries (not `go run`) so Ctrl+C / trap hits the listener, not a wrapper
# that leaves the compiled exe bound on Windows.
# Both mock upstreams, foreground. Use a second terminal, or use `make dev`.
mocks: build ## Run both mock upstreams until Ctrl+C
	@pids=""; \
	cleanup() { for pid in $$pids; do kill -TERM $$pid 2>/dev/null || true; done; wait 2>/dev/null || true; }; \
	trap cleanup INT TERM EXIT; \
	./bin/sales$(EXE) -addr $(SALES_ADDR) & pids="$$pids $$!"; \
	./bin/service$(EXE) -addr $(SERVICE_ADDR) & pids="$$pids $$!"; \
	wait $$pids

# The one command a reviewer needs: database, schema, mocks, API.
dev: infra-up migrate-up build ## db + migrations + both mocks + API. Ctrl+C stops all three
	@pids=""; \
	cleanup() { for pid in $$pids; do kill -TERM $$pid 2>/dev/null || true; done; wait 2>/dev/null || true; }; \
	trap cleanup INT TERM EXIT; \
	./bin/sales$(EXE) -addr $(SALES_ADDR) & pids="$$pids $$!"; \
	./bin/service$(EXE) -addr $(SERVICE_ADDR) & pids="$$pids $$!"; \
	./bin/documentviewer$(EXE) & pids="$$pids $$!"; \
	wait $$pids

# Same, with the Service upstream forced down, to demonstrate FR7 partial results.
demo-degraded: infra-up migrate-up build ## Same as dev but with the Service upstream down (demonstrates FR7)
	@pids=""; \
	cleanup() { for pid in $$pids; do kill -TERM $$pid 2>/dev/null || true; done; wait 2>/dev/null || true; }; \
	trap cleanup INT TERM EXIT; \
	./bin/sales$(EXE) -addr $(SALES_ADDR) & pids="$$pids $$!"; \
	./bin/service$(EXE) -addr $(SERVICE_ADDR) -down & pids="$$pids $$!"; \
	./bin/documentviewer$(EXE) & pids="$$pids $$!"; \
	wait $$pids

dev-stop: ## Kill leftover sales/service/documentviewer (and old mock-*) processes
ifeq ($(OS),Windows_NT)
	@powershell.exe -NoProfile -Command "Get-CimInstance Win32_Process | Where-Object { $$_.Name -match '^(sales|service|documentviewer|mock-sales|mock-service|api)\.exe$$' } | ForEach-Object { Stop-Process -Id $$_.ProcessId -Force -ErrorAction SilentlyContinue }"
else
	-@pkill -TERM -x sales$(EXE) || true
	-@pkill -TERM -x service$(EXE) || true
	-@pkill -TERM -x documentviewer$(EXE) || true
endif

# ---------------------------------------------------------------------------
# Test / quality
# ---------------------------------------------------------------------------

test: ## Unit tests
	$(GO) test ./...

# The gate for concurrency work: DD-2 failure isolation, FR3 parallelism.
test-race: ## Unit tests under -race. Gate for anything concurrent
	$(GO) test -race ./...

# Needs a running database: make infra-up migrate-up first.
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

hook-test: ## Cursor hook: refuse direct/forced push to main
	python3 .cursor/hooks/block_direct_main_test.py

# ---------------------------------------------------------------------------
# Codegen - mocks (go:generate mockgen) and OpenAPI (swag)
# ---------------------------------------------------------------------------

generate: mocks-gen openapi ## Generate everything: mockgen mocks + OpenAPI docs

mocks-gen: mocks-gen-documentviewer mocks-gen-sales mocks-gen-service ## Run go:generate mockgen per microservice

mocks-gen-documentviewer: $(MOCKGEN) ## mockgen for documentviewer modules
	GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) generate ./internal/documentviewer/...

mocks-gen-sales: $(MOCKGEN) ## mockgen for sales modules (no-op until ports grow)
	GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) generate ./internal/sales/...

mocks-gen-service: $(MOCKGEN) ## mockgen for service modules (no-op until ports grow)
	GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) generate ./internal/service/...

# ---------------------------------------------------------------------------
# OpenAPI - annotations become api/<service>/http/docs/{docs.go,swagger.yaml,swagger.json}
# ---------------------------------------------------------------------------

openapi: openapi-documentviewer openapi-sales openapi-service ## Regenerate all service OpenAPI contracts

openapi-documentviewer: $(SWAG) ## Regenerate api/documentviewer/http/docs
	$(SWAG) init --v3.1 -g docs.go -d $(SWAG_DOCUMENTVIEWER) -o ./api/documentviewer/http/docs --outputTypes go,yaml,json

openapi-sales: $(SWAG) ## Regenerate api/sales/http/docs
	$(SWAG) init --v3.1 -g docs.go -d $(SWAG_SALES) -o ./api/sales/http/docs --outputTypes go,yaml,json

openapi-service: $(SWAG) ## Regenerate api/service/http/docs
	$(SWAG) init --v3.1 -g docs.go -d $(SWAG_SERVICE) -o ./api/service/http/docs --outputTypes go,yaml,json

# CI gate: fails if the committed contract has drifted from the handlers.
openapi-check: openapi ## Fail if any committed contract has drifted from the code
	@git diff --exit-code -- \
		api/documentviewer/http/docs/docs.go api/documentviewer/http/docs/swagger.yaml api/documentviewer/http/docs/swagger.json \
		api/sales/http/docs/docs.go api/sales/http/docs/swagger.yaml api/sales/http/docs/swagger.json \
		api/service/http/docs/docs.go api/service/http/docs/swagger.yaml api/service/http/docs/swagger.json \
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
# Docker - infra only vs full stack
# ---------------------------------------------------------------------------

infra-up: ## Start infrastructure only (PostgreSQL)
	docker compose -p $(COMPOSE_PROJECT) -f $(COMPOSE_INFRA) up -d --wait

infra-down: ## Stop infrastructure
	docker compose -p $(COMPOSE_PROJECT) -f $(COMPOSE_INFRA) down

stack-up: ## Build and start infrastructure + all services
	docker compose -p $(COMPOSE_PROJECT) -f $(COMPOSE_ALL) up -d --wait --build

stack-down: ## Stop infrastructure + all services
	docker compose -p $(COMPOSE_PROJECT) -f $(COMPOSE_ALL) down --remove-orphans

db-up: infra-up ## Alias: start PostgreSQL (same as infra-up)

db-down: infra-down ## Alias: stop PostgreSQL (same as infra-down)

# Drops the volume too - use when a migration needs a clean slate.
db-reset: ## Stop PostgreSQL, drop the volume, start clean
	docker compose -p $(COMPOSE_PROJECT) -f $(COMPOSE_INFRA) down -v
	docker compose -p $(COMPOSE_PROJECT) -f $(COMPOSE_INFRA) up -d --wait

# ---------------------------------------------------------------------------
# Build
# ---------------------------------------------------------------------------

build: ## Build all three binaries into ./bin
	$(GO) build -ldflags "-X main.version=$(VERSION)" -o bin/documentviewer$(EXE) ./cmd/documentviewer
	$(GO) build -o bin/sales$(EXE) ./cmd/sales
	$(GO) build -o bin/service$(EXE) ./cmd/service

clean: ## Remove ./bin and coverage output
	rm -rf bin coverage.out
