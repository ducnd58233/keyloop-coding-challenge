# Unified Document Viewer

Keyloop Scenario D (Operate). One VIN lookup queries Sales and Service in parallel, normalises
two dissimilar payloads into one list, and stays useful when one upstream is down.

Backend in Go. The client is stubbed with an OpenAPI 3.1 contract and Swagger UI.

## Where things live

Three homes, no overlap. Read the right one; do not copy a rule into two places.

| Kind of thing | Home | How to use it |
|---|---|---|
| What is being built, and why | [`docs/design/`](docs/design/) | [`DRAFT.md`](docs/design/DRAFT.md) is the source of truth for goal, assumptions and requirements. [`SPEC.md`](docs/design/SPEC.md), [`SYSTEM_DESIGN.md`](docs/design/SYSTEM_DESIGN.md) and [`TASKS.md`](docs/design/TASKS.md) each state in their header what they derive from |
| Anything runnable | [`Makefile`](Makefile) | `make help` lists every target with its purpose. Commands change there only |
| How code is written | [`AGENTS.md`](AGENTS.md) | Architectural rules (R1–R7) for humans and coding tools. Not restated here |

Identifiers: `FR*`, `NFR*`, `A*` in DRAFT; `DD-*` in SYSTEM_DESIGN; `R*` in AGENTS.md.

## Prerequisites

| Tool | Why | Download |
|---|---|---|
| [Docker](https://docs.docker.com/get-docker/) | Full stack (`make stack-up` / `make stack-down`). Compose is included with Docker Desktop | [Get Docker](https://docs.docker.com/get-docker/) |
| [Go](https://go.dev/dl/) 1.26+ | Host tests, `make tools`, `make dev`. CI-style targets assume `go1.26.5` | [Download Go](https://go.dev/dl/) |

`make` is required for the targets above (Git Bash on Windows). Full stack only needs Docker plus `make migrate-up` after `stack-up`.

## Full stack (reviewer path)

This is the intended run. `make stack-up` / `make stack-down` own the whole compose project.

```bash
make stack-up       # Postgres 18 + sales + service + documentviewer (--build, waits healthy)
make migrate-up     # schema is not applied on container boot; do this once
# try APIs (below)
make stack-down     # stop everything
```

Ports: viewer **8000**, sales **9100**, service **9101**, Postgres **5432**.

`make db-reset` drops the Postgres volume after a major bump or a bad migration. Then `make stack-up` and `make migrate-up` again.

Optional host run (binaries on the laptop, only Postgres in Docker): `make tools` → `make dev`. Ctrl+C stops the binaries. `make demo-degraded` is the same with one mock down (FR7 `partial: true`).

## OpenAPI

Try-it-out is the client stub. After `stack-up`, open these:

| Service | Swagger UI | Contract |
|---|---|---|
| Document viewer | http://localhost:8000/docs | http://localhost:8000/openapi.json |
| Sales mock | http://localhost:9100/docs | http://localhost:9100/openapi.json |
| Service mock | http://localhost:9101/docs | http://localhost:9101/openapi.json |
| Health | http://localhost:8000/healthz | |

Viewer route: `GET /api/v1/vehicles/{vin}/documents`. Optional headers: `X-Actor-Id`, `X-Request-Id`.

Errors: HTTP status is the code. Body is `{"message":"..."}` and, on validation only, `"details": {"field":"..."}`. `sources[].error.code` is FR7 only.

Compose mocks use live chaos by default (~timeout/error). For a stable demo, restart sales/service with `-deterministic`, or use `make demo-degraded` on the host.

## Seed VINs (A1, 10 alphanumeric)

Use these in `/docs` Try it out.

| VIN | Expected |
|---|---|
| `1HGCM82633` | Documents from both Sales and Service |
| `3N1AB7AP1D` | `200`, `documents: []` (FR6, not 404) |
| `JHMCM56557` | Sales only, both sources `OK` |
| `WBA3A5C59E` | Service only |
| `1HGCM8263` | `400`, `details.vin` set (invalid length) |

## Commands

`make help` is authoritative. Common order:

| Step | Command | When |
|---|---|---|
| 1 | `make tools` | Once, for lint / swag / migrate / mockgen in `./bin` |
| 2 | `make stack-up` | Full Docker run |
| 3 | `make migrate-up` | After first `stack-up`, or after `db-reset` |
| 4 | Open `/docs` | Manual API check |
| 5 | `make test` · `make lint` · `make openapi-check` | Before review |
| 6 | `make test-integration` | Adapter tests; Testcontainers Postgres 18 + `migrations/`, never compose `DATABASE_URL` |
| 7 | `make test-race` | Concurrency gate (needs cgo; otherwise **UNVERIFIED**) |
| 8 | `make stack-down` | Tear down the full stack |

Also: `make openapi` regenerates `api/<service>/http/docs/`. `make infra-up` / `infra-down` is Postgres only.

## AI Collaboration Narrative

The brief scores how GenAI was directed, verified, and owned. This is that process.

**Design starts as a human document, not a generated one.** I wrote `docs/design/DRAFT.md` myself: assumptions, in/out of scope, functional and non-functional requirements, and a first architecture with options. That file is the source of truth. Everything else must name what it derives from. If an agent contradicted DRAFT, DRAFT won. Identifiers (`A*`, `FR*`, `NFR*`, `DD-*`, `R1–R7`) stay greppable so drift is a mismatch, not a plausible rewrite.

**Agents expand design; they do not replace the author.** I discussed DRAFT with GenAI, then had agents run a spec pass and a plan pass to produce `SPEC.md`, `TASKS.md`, and a complete `SYSTEM_DESIGN.md` (architecture, data flow, tech justifications, observability, GenAI-in-design section). We kept discussing on SYSTEM_DESIGN until the technical story held: failure isolation (NFR1 / DD-2), cache policy (NFR6/NFR7), audit (FR8/NFR8), status selection, and what was explicitly out of scope.

**Coding is task-shaped and rule-bound.** Implementation follows `TASKS.md` one slice at a time. Custom skills and commands (`/build`, `/test`, `/review`, `/ship`, and others) plus `AGENTS.md` are the contract the coding tools must obey: hexagonal slices, ports in the consuming package, one composition root per binary, no full VIN in logs, no SQL above `infra/persistence`, named literals (R7), Testcontainers for integration, Makefile as the only command list. The agent does not get a second, conflicting rulebook.

**I review before anything lands.** The agent writes tests and implementation, then runs `make test` / `make lint` / `make openapi-check`. I read the diff before commit. No AI co-author trailers. Work stays on a task branch; `main` only via PR.

**A second agent reviews before merge.** After my review, custom `/review` and `/test` go to another agent so the same authoring session is not the only reader. `/ship` is the merge gate: it checks verification evidence, not a model asserting “done”. GO plus my explicit approval is what merges. Then the next task starts on a new branch from `main`.

**Verification that actually bites.** Happy path is not enough. `errgroup.WithContext` cancelling a healthy sibling inverts NFR1 and fails silently; that isolation test is a gate. Partial results are never cached (NFR6). Cache read errors fail open (NFR7). Audit is append-only and written on invalid VIN and total failure (FR8). Logs assert a VIN suffix, not the full VIN. Plausible scaffolding that violated DRAFT or AGENTS (combined mock binary, extra A4 fields, GORM, `context.Background` on timeouts, integration tests on compose `DATABASE_URL`) was rejected, not “fixed later”.
