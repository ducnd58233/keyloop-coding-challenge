# AGENTS.md

Go backend for Scenario D: one VIN lookup that queries two mocked back-office systems in parallel,
normalises two dissimilar payloads into a single document model, and stays useful when one upstream
is down.

## Where things live

Three homes, no overlap. Look in the right one, and add to the right one.

| Kind of thing | Home | How to use it |
|---|---|---|
| What is being built, and why | `docs/` | `DRAFT.md` is the source of truth for goal, assumptions and requirements. Every other document there states in its own header what it derives from |
| Anything runnable | `Makefile` | `make help` lists every target with its purpose |
| How code is written here | this file | The rules below |

**Do not restate content from `docs/` or the `Makefile` in this file.** A rule that exists in
two places is a rule that will eventually disagree with itself. Commands change in the `Makefile`
only; requirements change in `DRAFT.md` only.

Identifiers are greppable: `FR*`, `NFR*`, `A*` are requirements and assumptions in `DRAFT.md`;
`DD-*` are design decisions in `SYSTEM_DESIGN.md`; `R*` are the rules below.

## Definition of done

`make lint`, `make test-race` and `make openapi-check` all pass. Nothing is finished before that.

## Architectural rules

**R1 - Modules interact only through ports.** A module never imports another module's packages. If
one needs something another owns, it declares an interface in its own `app/ports.go` and the
composition root injects an adapter. An import between modules is invisible coupling that compiles
fine and makes both untestable in isolation.

**R2 - Dependencies point one way.** `api` → `app` → `domain`. `infra` implements the ports declared
in `app`. `domain` imports nothing from the layers above it, and `shared` never imports `modules`.
This is what keeps `domain` and `app` free of I/O, which is what makes the failure tests runnable
without a network.

**R3 - One repository, one table.** A repository that grows to serve several tables becomes the place
every schema change lands, and its tests stop being able to state what they cover. A query spanning
two tables belongs in a dedicated read model, not bolted onto either repository.

**R4 - Transactions belong to the use case, never the repository.** A repository never calls `Begin`,
`Commit` or `Rollback`. Only the use case knows what one business operation is; a repository that
opens its own transaction cannot be composed with another without silently splitting the atomicity
the caller assumed. When a flow needs atomicity, open a Unit of Work and let repositories join it
through the context:

```go
err := uow.Within(ctx, func(ctx context.Context) error {
    if err := cacheRepo.Store(ctx, vin, result); err != nil { return err }
    return counterRepo.Increment(ctx, vin)   // both, or neither
})
```

No current flow needs this - cache upsert and audit insert are one statement each, and are
deliberately *not* atomic with each other, because an audit record must survive a cache write
failure (FR8). The rule still holds.

**R5 - No SQL outside `infra/persistence`.** Query text, driver types and `pgx` imports stay in the
adapters. SQL leaking upward is how a database-agnostic design quietly becomes database-shaped.

**R6 - Each binary has one composition root.** `internal/<service>/app/` constructs every adapter
for that binary and wires it to a port. Modules for that binary live in
`internal/<service>/modules/<module>/`. `cmd/<service>` only handles signals. One directory per
running system to read, one directory to change to swap an implementation.

## Code style

- `gofmt` is authoritative; `golangci-lint` must be clean.
- `context.Context` is the first parameter of anything that performs or bounds I/O.
- Declare interfaces in the consuming package (`app/ports.go`), never beside the implementation.
- Wrap errors with `%w`. Put sentinel errors in `domain/` when callers branch on them.
- Table-driven tests, colocated with the code they cover. Hand-written fakes are fine for ports this
  small; `make generate` runs `mockgen` if one grows.
- Name things after the domain, not the pattern: `documents.Aggregate`, not `DocumentServiceImpl`.
- Comments explain *why*. No commented-out code, no banner art.

## Boundaries

**Always**

- Obey R1-R6; they are review gates, not preferences.
- Read every environment variable in `configs/` and nowhere else.
- Regenerate and commit the contract when a handler or DTO changes (`make openapi`).
- Write an audit record for every request, including rejected and failed ones (FR8).
- Return the `sources[]` per-source status array on every aggregate response (FR7).
- Run commands through their real CLI — `make <target>` when one exists (`make help` lists them),
  the tool's own CLI otherwise.

**Ask first**

- Any `git commit`, push, branch deletion or pull request.
- Adding a dependency, or any exception to R1-R6.
- Editing `docs/design/DRAFT.md` - it is the source of truth, not a working file.
- Changing the public API contract once `api/<service>/http/docs/` exists.

**Never**

- Log, trace or label a full VIN. Use the last 4 characters plus a salted hash instead.
- Return internal error detail, hostnames, driver errors or stack traces to a client. Map upstream
  failures to the fixed error code set instead.
- Cache a partial or degraded result (NFR6). Write to cache only when every source succeeded.
- Let one source's failure cancel another's in-flight request (NFR1). Record the failure as data and
  return `nil` from the goroutine instead.
- Add an update or delete path to the audit store (NFR8). It is append-only.
- Commit a credential or a populated database volume.
- Hand-type or recall a command from memory when a CLI can produce or verify it. Flags and syntax
  drift between tool versions; `make help` and the tool's own `--help` are authoritative, memory
  is not.
- Suppress a linter with `//nolint` or an equivalent ignore. Fix the code.

## Before you write the aggregator

`errgroup.WithContext` cancels the shared context on the **first** non-nil error, so returning a
source failure from a goroutine aborts the healthy sibling and the service returns zero documents
whenever either upstream is unhealthy - the exact inverse of NFR1. It fails silently; happy-path
tests stay green. Read `SYSTEM_DESIGN.md` DD-2 before touching the fan-out, and keep the isolation
test as a gate.
