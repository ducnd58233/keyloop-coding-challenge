# AGENTS.md

Go backend for Scenario D: one VIN lookup that queries two mocked back-office systems in parallel,
normalises two dissimilar payloads into a single document model, and stays useful when one upstream
is down.

<scope>

This file is how code is written here. It is not the product spec and not the command list.

</scope>

<precedence>

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

</precedence>

<verification>

## Definition of done

`make lint`, `make test-race` and `make openapi-check` all pass. Nothing is finished before that.
If `test-race` cannot run (no cgo), say **UNVERIFIED**. Do not report a skipped race gate as a pass.

</verification>

<required>

## Architectural rules

**R1 - Modules interact only through ports.** A module never imports another module's packages. If
one needs something another owns, it declares an interface in its own `app/ports.go` and the
composition root injects an adapter. An import between modules is invisible coupling that compiles
fine and makes both untestable in isolation. A binary never imports `internal/<other-service>/`.

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
for that binary (logger, HTTP server, DB pool, upstream clients, handlers) and wires it to a port.
Modules for that binary live in `internal/<service>/modules/<module>/`. `cmd/<service>` parses flags
and installs signal handlers. It does not call `NewLogger`, open a database, or build an
`http.Handler`. After `Run` returns, `main` may print to stderr; it must not log through a logger
`Run` already closed. One directory per running system to read, one directory to change to swap an
implementation.

## Always

- Obey R1-R6; they are review gates, not preferences.
- Read every environment variable in `configs/` and nowhere else. Mock outage and live chaos are
  process flags on the mock binaries, not `.env` keys.
- Regenerate and commit the contract when a handler or DTO changes (`make openapi`).
- Write an audit record for every request, including rejected and failed ones (FR8).
- Return the `sources[]` per-source status array on every aggregate response (FR7).
- Run commands through their real CLI. `make <target>` when one exists (`make help` lists them),
  the tool's own CLI otherwise.
- Log listen/start only after bind succeeds.
- Land work on a task branch. The only path onto `main` is `gh pr create` on origin, then
  `gh pr merge`. After merge, `git checkout main && git pull`. "merge", "ship", or `/vibe-ship`
  GO is not permission to update `main` directly.

## Never

- Log, trace or label a full VIN. Do not log `r.URL.Path`, `r.RequestURI`, raw query strings, or
  path values that contain a VIN. Use a constant route template plus the last 4 characters
  (`vin_suffix`); salted hash when an identifier must reach audit or traces.
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
- Suppress a linter with `//nolint`, `#nosec`, an ignore file, or a weaker file mode to silence
  gosec. Fix the code (`0o750` dirs, `0o600` log files, `os.OpenRoot` for variable paths).
- Add a `Co-authored-by` line or any AI trailer to a commit.
- Call `os.Exit` while a logger, DB, or listener still needs `Close`. The composition root `defer`s
  Close; `os.Exit` skips that.
- Push commits directly to `main` or `master` (`git push origin main`, `HEAD:main`, `git push`
  while on `main`, `--all`, `--mirror`).
- Force-push `main`/`master` (`--force`, `--force-with-lease`, `-f`, `+main`), including to invent
  a PR after a direct push, unless the user explicitly requests a rewind.

</required>

<rules>

## Code style

- `gofmt` is authoritative; `golangci-lint` must be clean.
- `context.Context` is the first parameter of anything that performs or bounds I/O. Request-scoped
  wait, dial, upstream call, and shutdown take the request or signal context, not
  `context.Background()` or `context.TODO()`. Tests that need a non-cancelled ctx may use Background.
- Declare interfaces in the consuming package (`app/ports.go`), never beside the implementation.
- Wrap errors with `%w`. Put sentinel errors in `domain/` when callers branch on them.
- Table-driven tests, colocated with the code they cover. Ports in `app/ports.go` are mocked only
  via `make generate` (`go.uber.org/mock`). Do not hand-write fakes for those ports. Do not mockgen
  `Logger`; VIN and log-redaction assertions use `slog.NewTextHandler` on a `bytes.Buffer`.
  Reusable helpers that cannot be generated live in `internal/testutil/` (or `<root>/test/` if they
  must stay outside module packages). Do not add `<root>/tests/`. Unit tests stay colocated
  `*_test.go` with no build tag; do not rename them to `*_unit_test.go`. Live-database tests use
  `//go:build integration` and `*_integration_test.go` beside the adapter. Process harnesses live
  under `<root>/test/e2e` with `//go:build e2e`. Never put live-database or live-process I/O in an
  untagged `*_test.go` file.
- Name things after the domain, not the pattern: `documents.Aggregate`, not `DocumentServiceImpl`.
- Comments explain *why* (constraint, trap, requirement id). Do not restate the next line. Exported
  godoc that revive requires starts with the identifier and states a constraint, not a paraphrase.
- No commented-out code, no banner art, no emoji, no em-dash in comments or commit messages.
- Mock randomness goes through `internal/shared/randutil` (`crypto/rand`). Do not import `math/rand`.

</rules>

<escalation>

## Ask first

- Any `git commit`, feature-branch push, or branch deletion, unless the user already authorized
  that ship in this conversation.
- Adding a dependency, or any exception to R1-R6.
- Editing `docs/design/DRAFT.md` - it is the source of truth, not a working file.
- Changing the public API contract once `api/<service>/http/docs/` exists.

## Main only via PR

Do not ask whether to skip the PR. Create it on origin (`gh pr create`), then merge it
(`gh pr merge`). Fast-forwarding local `main` and pushing, or force-pushing `main` to attach a
PR after the fact, is a workflow violation even when the user said "merge and push".

## Harness

This file is the in-repo delivery harness. Do not add Cursor hooks, git hooks, or Make targets to
refuse `git push`. If origin later supports GitHub branch protection (require a pull request, deny
force-push to `main`), enable that on origin; private GitHub Free cannot. Optional vibe-agent
`merge_approved` stays out of tree and is not required here.

</escalation>

<antipatterns>

## Traps already paid for

These compiled, or the happy path stayed green. Do not reintroduce them.

- **`r.URL.Path` in mock logs** leaked the full VIN on the Service route. Route template + `vin_suffix`.
- **Logger constructed in `cmd/`** violated R6; `os.Exit(1)` skipped `Close`.
- **`context.Background()` on a timeout hang** made request cancel a no-op.
- **`math/rand` + `//nolint:gosec`** instead of `randutil`.
- **mockgen / `Capture` for `Logger`** cannot assert redaction. Use a slog text buffer.
- **`http server started` before `Listen`** reported a bound port that never accepted.
- **Direct `git push origin main` after ship GO** skipped the PR. Repair was rewind + PR #1.
  `gh pr create` then `gh pr merge`.
- **`--force-with-lease` on `main`** to attach a PR retroactively. Still a force-push to main.

## Before you write the aggregator

`errgroup.WithContext` cancels the shared context on the **first** non-nil error, so returning a
source failure from a goroutine aborts the healthy sibling and the service returns zero documents
whenever either upstream is unhealthy - the exact inverse of NFR1. It fails silently; happy-path
tests stay green. Read `SYSTEM_DESIGN.md` DD-2 before touching the fan-out, and keep the isolation
test as a gate.

</antipatterns>
