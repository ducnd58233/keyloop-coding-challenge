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

# cmd/<name>/ is the service list. Override: make SERVICES="sales service documentviewer"
empty :=
space := $(empty) $(empty)
SERVICES ?= $(notdir $(patsubst %/,%,$(wildcard cmd/*/)))
VIEWER ?= documentviewer
MOCKS ?= $(filter-out $(VIEWER),$(SERVICES))
DEVSTOP_REGEX = $(subst $(space),|,$(SERVICES))|mock-sales|mock-service|api

# Ports for the two mock upstreams (A5 - separate servers)
SALES_ADDR ?= :9100
SERVICE_ADDR ?= :9101
ADDR_sales ?= $(SALES_ADDR)
ADDR_service ?= $(SERVICE_ADDR)
DEMO_DOWN ?= service

.DEFAULT_GOAL := help

.PHONY: help tools run mocks dev dev-stop demo-degraded \
	test test-race test-integration cover vet fmt tidy lint \
	generate mocks-gen openapi openapi-check \
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
# Mock upstreams, foreground. Use a second terminal, or use `make dev`.
mocks: build ## Run mock upstreams until Ctrl+C
	@pids=""; \
	cleanup() { for pid in $$pids; do kill -TERM $$pid 2>/dev/null || true; done; wait 2>/dev/null || true; }; \
	trap cleanup INT TERM EXIT; \
	$(foreach s,$(MOCKS),./bin/$(s)$(EXE) -addr $(ADDR_$(s)) & pids="$$pids $$!";) \
	wait $$pids

# The one command a reviewer needs: database, schema, mocks, API.
dev: infra-up migrate-up build ## db + migrations + mocks + API. Ctrl+C stops every cmd/ binary
	@pids=""; \
	cleanup() { for pid in $$pids; do kill -TERM $$pid 2>/dev/null || true; done; wait 2>/dev/null || true; }; \
	trap cleanup INT TERM EXIT; \
	$(foreach s,$(MOCKS),./bin/$(s)$(EXE) -addr $(ADDR_$(s)) & pids="$$pids $$!";) \
	./bin/$(VIEWER)$(EXE) & pids="$$pids $$!"; \
	wait $$pids

# Same, with DEMO_DOWN forced down, to demonstrate FR7 partial results.
demo-degraded: infra-up migrate-up build ## Same as dev but DEMO_DOWN (default service) is -down (FR7)
	@pids=""; \
	cleanup() { for pid in $$pids; do kill -TERM $$pid 2>/dev/null || true; done; wait 2>/dev/null || true; }; \
	trap cleanup INT TERM EXIT; \
	$(foreach s,$(MOCKS),./bin/$(s)$(EXE) -addr $(ADDR_$(s)) $(if $(filter $(s),$(DEMO_DOWN)),-down) & pids="$$pids $$!";) \
	./bin/$(VIEWER)$(EXE) & pids="$$pids $$!"; \
	wait $$pids

dev-stop: ## Kill leftover cmd/ binaries (and old mock-* names)
ifeq ($(OS),Windows_NT)
	@powershell.exe -NoProfile -Command "Get-CimInstance Win32_Process | Where-Object { $$_.Name -match '^($(DEVSTOP_REGEX))\.exe$$' } | ForEach-Object { Stop-Process -Id $$_.ProcessId -Force -ErrorAction SilentlyContinue }"
else
	@for s in $(SERVICES); do pkill -TERM -x $$s$(EXE) || true; done
endif

# ---------------------------------------------------------------------------
# Test / quality
# ---------------------------------------------------------------------------

test: ## Unit tests
	$(GO) test ./...

# The gate for concurrency work: DD-2 failure isolation, FR3 parallelism.
test-race: ## Unit tests under -race. Gate for anything concurrent
	$(GO) test -race ./...

# Isolated Testcontainers Postgres 18 + migrations/. Does not use compose DATABASE_URL.
# -p 1: Windows Docker Desktop rejects concurrent Testcontainers providers (rootless error).
test-integration: ## Integration tests (build tag); starts its own Postgres, no infra-up
	$(GO) test -p 1 -tags=integration ./internal/...

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

# ---------------------------------------------------------------------------
# Codegen - mocks (go:generate mockgen) and OpenAPI (swag)
# ---------------------------------------------------------------------------

generate: mocks-gen openapi ## Generate everything: mockgen mocks + OpenAPI docs

mocks-gen: $(MOCKGEN) ## go:generate mockgen for every cmd/ service
	@for s in $(SERVICES); do \
		GOTOOLCHAIN=$(GOTOOLCHAIN) $(GO) generate ./internal/$$s/...; \
	done

# ---------------------------------------------------------------------------
# OpenAPI - annotations become api/<service>/http/docs/{docs.go,swagger.yaml,swagger.json}
# ---------------------------------------------------------------------------

openapi: $(SWAG) ## Regenerate OpenAPI contracts for every cmd/ service
	@for s in $(SERVICES); do \
		$(SWAG) init --v3.1 -g docs.go \
			-d cmd/$$s,internal/$$s \
			-o ./api/$$s/http/docs --outputTypes go,yaml,json; \
	done

# CI gate: fails if the committed contract has drifted from the handlers.
openapi-check: openapi ## Fail if any committed contract has drifted from the code
	@fail=0; \
	for s in $(SERVICES); do \
		git diff --exit-code -- \
			api/$$s/http/docs/docs.go api/$$s/http/docs/swagger.yaml api/$$s/http/docs/swagger.json \
			|| fail=1; \
	done; \
	if [ $$fail -ne 0 ]; then echo "openapi out of date; run make openapi and commit" && exit 1; fi

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

build: ## Build every cmd/ binary into ./bin
	@mkdir -p bin
	@for s in $(SERVICES); do \
		$(GO) build -ldflags "-X main.version=$(VERSION)" -o bin/$$s$(EXE) ./cmd/$$s; \
	done

clean: ## Remove ./bin and coverage output
	rm -rf bin coverage.out
