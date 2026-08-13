# Build specification - Unified Document Viewer

**Scenario D** | Domain: Operate | Implemented layer: **Backend (Go)**
Version: 2.0 | Status: **awaiting confirmation**

> **Provenance.** [`DRAFT.md`](DRAFT.md) is the source of truth for the goal, assumptions (A1-A11),
> scope, requirements (FR1-FR10, NFR1-NFR8) and persistence design. This document does **not**
> restate them. It specifies how the build is executed: commands, structure, style, testing, data
> handling, boundaries and success criteria.

## Document set

| Document | Holds | Authority |
|---|---|---|
| [`DRAFT.md`](DRAFT.md) | Goal, assumptions, scope, requirements, persistence design | **Source of truth** |
| [`SYSTEM_DESIGN.md`](SYSTEM_DESIGN.md) | The Part 1 deliverable: **architecture selection: the author's option from DRAFT §6 confirmed against five alternatives (§2)**, architecture, components, data flow, technology, observability, GenAI narrative | Derived from DRAFT |
| **This document** | Build specification | Derived from DRAFT |
| [`TASKS.md`](TASKS.md) | Ordered execution plan with gates | Derived from all of the above |

If any document contradicts `DRAFT.md`, `DRAFT.md` wins and the other document is wrong.

---

## 1. Objective and users

The objective is `DRAFT.md` §1. Who is served by which requirement:

| User | Need | Requirement |
|---|---|---|
| Dealership service advisor | All paperwork for a vehicle in one view | FR1, FR4 |
| Dealership manager / compliance | Evidence of who accessed which vehicle's documents | FR8, NFR8 |
| Client application developer | A contract that states whether the list is complete | FR7, the `sources[]` array |
| Operator on call | Degradation attributed to a specific upstream | NFR5 |

---

## 2. Interface

One endpoint. Field naming is snake_case throughout, following `DRAFT.md` A4 (`issued_at`).

```
GET /api/v1/vehicles/{vin}/documents
Header: X-Actor-Id     (optional; recorded in the audit trail per A2, FR8)
Header: X-Request-Id   (optional; generated when absent)
```

| Status | When | Body |
|---|---|---|
| `200` | Any result was produced, complete or not | Documents + `sources[]` + `partial` + `served_from_cache` + `stale` |
| `400` | VIN fails the A1 format check | `{"error":{"code":"VIN_INVALID"}}` |
| `503` | All sources failed and no cached entry exists (FR10) | `{"error":{"code":"ALL_SOURCES_UNAVAILABLE"}}` |

Response shape and the full status decision flow are in `SYSTEM_DESIGN.md` §5.2 and §5.4.

Supporting endpoints: `/healthz`, `/readyz`, `/metrics`.

**Error code vocabulary.** `VIN_INVALID` and `ALL_SOURCES_UNAVAILABLE` come from `DRAFT.md` §8.
`UPSTREAM_TIMEOUT` and `UPSTREAM_ERROR` are added for per-source reporting under FR7. No other codes
without updating this list.

---

## 3. Tech stack

Justifications and rejected alternatives: `SYSTEM_DESIGN.md` §7. A8 fixes the persistence choice.

| Concern | Choice |
|---|---|
| Language | Go 1.26 (verified `go1.26.5`) |
| Routing | stdlib `net/http.ServeMux` |
| Concurrency | `golang.org/x/sync/errgroup`, `context` |
| Persistence | PostgreSQL 17 via `jackc/pgx/v5` - **A8** |
| Data access | `pgx` + golang-migrate pairs in `migrations/`, applied by `make migrate-up` (not on boot) |
| Logging | stdlib `log/slog`, JSON handler |
| Metrics | `prometheus/client_golang` |
| Tracing | OpenTelemetry Go SDK, stdout exporter |
| Testing | stdlib `testing`, table-driven, `httptest`; `mockgen` available for larger ports |
| Contract | OpenAPI 3.1 generated from handler annotations by `swag`, drift-checked in CI |
| Config / request ID | `godotenv` (load `.env`), `google/uuid` (`X-Request-Id` when absent) |

**Dependency budget.** Runtime groups in use or planned: `godotenv`, `google/uuid`, `x/sync`,
`jackc/pgx/v5`, `prometheus/client_golang`, `go.opentelemetry.io/otel`. Adding another requires a
decision recorded in `SYSTEM_DESIGN.md` §6.9.

---

## 4. Commands

Every workflow is a `make` target. The `Makefile` is the single source of truth and describes itself:
run **`make help`**. This document does not list targets, so it cannot fall out of step with them.

Two capabilities are **specification requirements** rather than conveniences, and the build is not
complete without them, whatever they end up being called:

- **A single command that brings a reviewer to a working system** - database, schema, both mock
  upstreams and the API. A reviewer must not have to assemble it from instructions.
- **A command that runs the system with one upstream deliberately down.** Without it, FR7 is a claim
  rather than a demonstration.

Tooling is pinned into `./bin` via `GOBIN` rather than installed globally, so versions cannot drift
between machines: `golangci-lint`, `golang-migrate`, `swag`, `mockgen`.

---

## 5. Project structure and architectural rules

Vertical slices under `internal/modules/`, cross-cutting technical concerns under `internal/shared/`,
configuration as a package at the repository root. Full annotated tree and the reasoning behind the
module split: `SYSTEM_DESIGN.md` §3.3. This is the **target** layout; files land per `TASKS.md`
(T2 keeps empty packages as `doc.go` stubs).

```
api/http/docs/                     generated OpenAPI 3.1: docs.go, swagger.yaml, swagger.json
cmd/
  api/                             main.go + docs.go (swag general annotations)
  mock-sales/                      Sales System mock, port 9100    (A5 - separate server)
  mock-service/                    Service System mock, port 9101  (A5 - separate server)
configs/                           package configs - repository root, not internal
  config.go  env.go  http.go  sources.go  cache.go  database.go  log.go
internal/
  app/                             composition root: bootstrap.go, http.go
  modules/
    documents/                     aggregation slice
      api/                         routes.go - the module's public surface
      app/                         ports.go + use_cases/
      domain/                      document, source, vin, merge, errors    (no I/O)
      dto/                         wire shapes, snake_case
      infra/http/                  Sales + Service clients and normalisers
      infra/persistence/           cache repository (document_cache only, R3)
    audit/                         compliance slice
      app/                         ports.go + use_cases/
      domain/                      access_event
      infra/persistence/           append-only audit repository (search_audit only, R3)
  shared/
    common/                        clock, salted hashing
    infra/httpserver/              server.go, response.go, context.go, middleware/
    infra/postgres/                pgxpool connection, Unit of Work (R4)
    observability/                 logger, metrics, tracing
deployments/docker/docker-compose.yaml   PostgreSQL 17, healthcheck, named volume (A8)
migrations/                        golang-migrate pairs, applied by `make migrate-up`
bin/                               tools installed by `make tools` (gitignored)
docs/design/                       DRAFT.md, SPEC.md, SYSTEM_DESIGN.md, TASKS.md
examples/curl.md
.env.example                       committed template
.env                               local overrides, gitignored; copy from .env.example
Makefile
AGENTS.md
README.md
```

**Three binaries, not two.** A5 specifies two separate mock servers; each gets its own `cmd/` entry.

**Two modules, not one.** `audit` is a separate bounded context with its own invariant (NFR8:
append-only) and no dependency from `documents`. A single module would make `modules/` decorative.

**`modules/*/domain/` and `modules/documents/app/use_cases/` contain zero I/O.** That constraint is
what makes the failure matrix in §7 testable without a network, and it is where the business logic
the brief asks to be validated by tests actually lives. Tests are colocated with the code they cover.

### 5.1 Architectural rules

Six rules govern how these packages may depend on each other: modules interact only through ports,
one repository per table, transactions owned by the use case, no SQL above `infra`, and a single
composition root. They are **enforcement rules rather than design rationale**, so they live in
[`AGENTS.md`](../../AGENTS.md) next to the code, not here. Referenced throughout as **R1**-**R6**.

---

## 6. Configuration

The `configs` package at the repository root is the **only** place an environment variable is read.
One `Load()` returns a `Config` composed of per-concern structs, one file per concern; `env.go`
holds the `env` / `duration` / `integer` / `float` / `boolean` helpers so parsing and defaulting
are not repeated.

```go
// configs/config.go
type Config struct {
    HTTP     HTTP
    Sources  Sources
    Cache    Cache
    Database Database
    Log      Log
}

func Load() (Config, error)   // godotenv.Load() then the environment
```

`Load()` calls `godotenv.Load()`, which reads `.env` from the process working directory when the
file is present. A missing `.env` is not an error. Copy `.env.example` to `.env` for local
overrides; `.env` is gitignored.

Being outside `internal/` is deliberate: `cmd/mock-sales` and `cmd/mock-service` load their ports
and fault-injection settings through the same loader as `cmd/api`, so all three binaries share one
definition of every variable and its default.

Safe defaults throughout, so `make dev` works with nothing set.

| Variable | Default | File | Notes |
|---|---|---|---|
| `HTTP_ADDR` | `:8080` | `http.go` | |
| `REQUEST_TIMEOUT` | `3s` | `http.go` | Derived; must exceed `AGGREGATE_TIMEOUT` |
| `SALES_BASE_URL` | `http://localhost:9100` | `sources.go` | Also resolves relative `download_path` values |
| `SERVICE_BASE_URL` | `http://localhost:9101` | `sources.go` | |
| `PER_SOURCE_TIMEOUT` | `2s` | `sources.go` | `DRAFT.md` §7, §8 |
| `AGGREGATE_TIMEOUT` | `2500ms` | `sources.go` | Derived; must exceed `PER_SOURCE_TIMEOUT` |
| `CACHE_TTL` | `60s` | `cache.go` | A9 |
| `DATABASE_URL` | `postgres://viewer:viewer@localhost:5432/viewer?sslmode=disable` | `database.go` | Matches the compose file. Password is a local development default and is never a real credential |
| `DB_MAX_CONNS` | `10` | `database.go` | Pool ceiling; see SYSTEM_DESIGN §9.2 |
| `VIN_HASH_SALT` | `dev-only-not-a-secret` | `log.go` | **Secret in production.** Never logged. See §8 |
| `LOG_LEVEL` | `info` | `log.go` | |
| `MOCK_LATENCY_MS` | `0` | `sources.go` | Fault injection for the mock servers (T3) |
| `MOCK_ERROR_RATE` | `0` | `sources.go` | |
| `MOCK_DOWN` | `false` | `sources.go` | |

`Load()` **fails fast** if the timeout ordering is violated. An inverted budget is a silent bug: the
outer layer fires first and the service loses the ability to report which dependency was slow
(`SYSTEM_DESIGN.md` DD-3).

---

## 7. Testing strategy

Runner: `go test`. No framework dependency. Table-driven throughout. `-race` on concurrency tests.

The full matrix is `SYSTEM_DESIGN.md` §9.4, traced per requirement in §11.2. The cases that gate
completion:

| Gate | Proves |
|---|---|
| **Failure isolation** - a failing source must not cancel its healthy sibling | NFR1. The `errgroup` trap in DD-2. Fails *silently*: the happy path still passes |
| **Parallelism** - two sources each delayed 500ms complete in under 1s | FR3, NFR3 |
| **Cache never stores a partial result** | NFR6 |
| **Cache read error falls through to upstreams** rather than failing the request | NFR7 |
| **Audit written on all four outcomes**: success, partial, invalid VIN, total failure | FR8 |
| **Log capture contains the VIN suffix, never the full VIN** | §8 |
| All green under `-race` | NFR4 |

---

## 8. Data classification

Every place data leaves the process, and what is allowed there. A specification that never says
which fields are sensitive produces code that guesses.

| Data class | Example | Allowed | **Never** |
|---|---|---|---|
| **VIN** - quasi-identifier, personal data in context | `1HGCM82633` | Request path; response body (the caller supplied it); cache table as the key; **last 4 chars + salted SHA-256** in logs, traces and the audit table | **Full VIN in logs, trace attributes, or metric labels.** Metric labels additionally because it is unbounded cardinality |
| **Actor identity** - personal data | `X-Actor-Id` | Audit table (FR8); server-side request logs | Response body; metric labels |
| **Document metadata** - internal business data | Titles, URLs | Response body; cache table (A11) | Logs; trace attributes; metric labels |
| **Hash salt** - secret | `VIN_HASH_SALT` | Environment variable, read once at startup | Logs, responses, repository, metrics, error messages, test fixtures |
| **Internal structure** | Upstream URLs, hostnames, DB path, driver errors, stack traces | Server-side logs; config | **Client-facing responses.** Upstream errors map to the fixed code set in §2 |
| **Document bytes** | - | Nowhere - not fetched, not stored | Cache (A11), logs, responses |
| **Upstream credentials** | none in scope (A2) | If introduced: environment only | Repository, logs, responses, client-visible config |

**Why the VIN rule.** A VIN identifies a specific vehicle and, combined with dealer records, an
identifiable person; it is treated as personal data in the connected-vehicle context under GDPR.
Logs are the highest-fan-out, longest-retained and least access-controlled surface in the system.
`DRAFT.md` §5.4 sets the audit-table rule - salted hash plus last 4 characters - and the same rule
applies to logs and traces. Four rather than six because at the 10-character length in A1, a
six-character suffix would expose most of the identifier and defeat the hashing.

**Enforced by** a test that captures log output and asserts the full VIN is absent.

---

## 9. Boundaries

What must always happen, what needs approval, and what is forbidden while working in this repository
is stated in [`AGENTS.md`](../../AGENTS.md) - it belongs beside the code an agent or contributor is
editing, not in a design document. The data-handling rules it enforces are specified in §8 above.

---

## 10. Success criteria

1. Every `DRAFT.md` requirement FR1-FR10 and NFR1-NFR8 is verified by a passing test, or explicitly listed in the `TASKS.md` cut list.
2. `make test-race` and `make lint` pass clean.
3. `make dev` gives a working demo from a clean clone with Docker as the only prerequisite (A8).
4. `make demo-degraded` visibly returns `200` with `partial: true` naming the failed source (FR7).
5. A trace shows the two upstream spans **overlapping** - the visual proof of FR3 and NFR3.
6. `SYSTEM_DESIGN.md` covers all six Part 1 elements.
7. `README.md` documents build/run/test and contains the AI Collaboration Narrative.
8. The T9 demo script has been rehearsed end to end inside the 5-10 minute video budget.

---

## 11. Open questions

| # | Question | Default if unanswered |
|---|---|---|
| Q1 | Keep `singleflight` request collapsing? | First on the cut list; it protects against a stampede a demo will never produce |
| Q2 | Does the video need the OTel trace view, given a stdout exporter renders poorly on camera? | Show `/metrics` and the degraded curl; describe the trace design verbally |
| Q3 | Keep docs under `docs/design/`, or move to `docs/<slug>/` per the toolkit convention? | Keep `docs/design/` - it exists and splitting the set helps nobody |
| Q4 | Should the mock servers share a package, or be fully independent? | Share a small internal package for seed data; keep the HTTP surfaces fully independent so the payload dissimilarity A5 requires cannot accidentally converge |

### 11.1 Accepted risk: VIN length

Recorded so the decision is visible rather than looking like an oversight.

ISO 3779 defines a VIN as 17 characters excluding `I`, `O` and `Q`. `DRAFT.md` A1 specifies 10,
matching the synthetic identifiers in the mock upstreams. The risk is presentational: an
automotive-domain reviewer may read a 10-character VIN rule as a domain misunderstanding rather than
a scoping choice.

Two things contain it. A1 is stated as a simplification with the production form named, so the
reader sees a decision rather than a gap. And the validator is built as one configurable rule over
length and alphabet, so `modules/documents/domain/vin.go` moves to the ISO form by changing a constant and a character
set - no logic change, no test rewrite.

**Reversing this costs one line plus a test-table update.**

---

*No open question blocks implementation. Q1-Q4 have defaults.*
