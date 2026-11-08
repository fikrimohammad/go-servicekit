# Spec: agent-consumable-sdk

Module id: `agent-consumable-sdk`. Status: approved 2026-11-08; docs-only
re-scope of the 2026-11-03 draft. Scope:
[confirmed intent](../../intent/agent-consumable-sdk.md); reasoning:
[idea doc](../../ideas/agent-consumable-sdk.md).

## Scope Note

This is one capability with staged workstreams, not a capability map. The 21
contracts, the eval kit, the sample evals, and the release are not independently
shippable modules: the kit needs the contracts, the evals need both, and the
release needs the eval proof. The workstream table below is the build order; it
is not a set of separate specs.

This project is documentation-only. It adds no exported API, no dependency, and
no module. A capability an eval finds missing is recorded as a finding and left
for a separate project; a real-service migration is likewise out of scope.

## Objective

Make the SDK self-describing for coding agents in other repos. Every one of the
SDK's 21 public import paths gets a `doc.go` contract that an agent can read from
`go doc` alone: exact import path, when to use it, a minimal correct snippet,
lifecycle, config defaults, common mistakes, and a test recipe. The root README
becomes a task→package map, and the `nats`/`rest` module roots get doc-only
packages so an agent cannot import a root that has no API.

Prove it with six sample evals — `db`, `rest/server`, `nats/consumer`,
`observability`, `confloader`, `errs` — run by a separate agent from a
self-contained kit, with zero SDK-usage corrections. Every sample runs
in-process: sqlmock for `db`, the embedded NATS server for `nats/consumer`, a
fake client for `confloader`, and `httptest` for `rest/server`. No fixture needs
a database, Redis, S3, NATS, or a collector as an external service.

Migrating the SDK into a real service is out of scope. It needs the SDK
implemented in a real project and is a separate project that will consume this
kit and these contracts. The release is tagged `v0.1.1` inside five weeks.

### Users

- **Primary:** coding agents working in other repos.
- **Secondary:** human reviewers who need the same contract without reading
  3,000 lines of module READMEs.
- **Owner:** the SDK maintainer, who runs the eval kit with another agent and
  reviews the findings.

### User stories

- As an agent in another repo, I run `go doc <package>` and get the import path,
  the one thing it is for, a compiling snippet, its lifecycle, its common
  mistakes, and a test recipe without opening the README.
- As an agent, I open the root README with a task ("add persistence") and land
  on `.../db`; I never import `.../nats` or `.../rest` expecting a root API.
- As an eval runner, I hand another agent a fixed prompt, a fixed fixture, an
  allowed-files list, and a done command, then grade SDK-usage corrections
  against a checklist.
- As the SDK owner, I see gaps reported with the package and the eval that found
  them, and I do not change the SDK to ship the docs.
- As a consumer, I `go get` the tagged version and the contracts are in the
  module cache.

## Workstreams and Build Order

| Workstream | Responsibility | Depends on | Weeks |
|---|---|---|---|
| `contracts` | 21 `doc.go` contracts; root import map; doc-only `nats`/`rest` roots; contract check | — | 1–4 |
| `eval-kit` | Fixed fixtures, prompts, allowed files, done commands, grader checklists, correction log; six sample kits | `contracts` (prompts need the final package surface) | 2–4 |
| `sample-evals` | Six manual runs against the local SDK; findings logged | `contracts`, `eval-kit` | 4–5 |
| `release` | Bump `VERSION`/sibling requires, tag every module at `v0.1.1`, verify proxy installs | `contracts`, `sample-evals` | 5 |

## Assumptions (correct any of these before approval)

1. The **21 public import paths** are the ones in "Package Set"; the `nats` and
   `rest` roots are doc-only additions, not part of the 21.
2. The six sample fixtures live in this repo as non-module templates;
   `setup.sh <sample>` copies one into a temporary working directory.
3. The sample evals run against the **local working tree** through a `replace`
   directive; the release is verified separately by proxy install, and the tag
   is spot-checked where a fixture allows it.
4. The eval agent has Go 1.26.5 and network access for module downloads. It
   needs no Docker, database, collector, or context from this conversation.
5. The project changes no exported API. A capability an eval finds missing is a
   finding, not work here.
6. `confloader` stays a LAX module until its redesign; its contract documents
   the current API.
7. The release is a docs-only `v0.1.1`.
8. "Correction" means a human edit to SDK usage. Business-logic, formatting, or
   fixture edits are not corrections.
9. The sample evals are the only empirical proof; CI runs the contract check and
   the module tests, never an agent.

## Tech Stack

- **SDK:** Go 1.26.5 (`go.work`), 11 modules. Existing dependencies only; no new
  third-party module or dependency. `confloader` stays LAX until its redesign.
- **Eval:** harness-agnostic coding agent with Go 1.26.5; no agent-specific SDK
  or plugin. Fixtures compile against the local SDK through a `replace`
  directive; no service containers.
- **Tools:** `gofmt`, `go vet`, `go test -race`, `golangci-lint` v2.15.0,
  gostyle from `.agents/skills/go-best-practices/scripts`, `go mod tidy`. No new
  linter or formatter.

## Package Set

Each row gets a `doc.go` in its package directory. If the package comment
currently lives in another file, move it so there is exactly one package
comment.

| # | Import path | Module | Notes |
|---|---|---|---|
| 1 | `.../errs` | `errs` | Errors, slog attribute. |
| 2 | `.../errgroup` | `errgroup` | Concurrency, panic error. |
| 3 | `.../identity` | `identity` | Service identity. |
| 4 | `.../observability` | `observability` | Root has no API; it is the wiring guide. |
| 5 | `.../observability/attributes` | `observability` | Key normalization. |
| 6 | `.../observability/logs` | `observability` | Logger. |
| 7 | `.../observability/metrics` | `observability` | Metrics client. |
| 8 | `.../observability/tracer` | `observability` | Tracer client. |
| 9 | `.../confloader` | `confloader` | Loader; LAX module. |
| 10 | `.../confloader/client` | `confloader` | Backend contract. |
| 11 | `.../confloader/client/etcd` | `confloader` | etcd backend. |
| 12 | `.../confloader/client/infisical` | `confloader` | Infisical backend. |
| 13 | `.../rest/client` | `rest` | Hertz client wrapper. |
| 14 | `.../rest/server` | `rest` | Hertz server. |
| 15 | `.../cron` | `cron` | Job model. |
| 16 | `.../cron/engine` | `cron` | Engine. |
| 17 | `.../db` | `db` | sqlx named-query client. |
| 18 | `.../nats/consumer` | `nats` | JetStream consumer. |
| 19 | `.../nats/producer` | `nats` | JetStream producer. |
| 20 | `.../redis` | `redis` | Redis client. |
| 21 | `.../s3` | `s3` | AWS S3 client. |

Doc-only roots: `.../nats` and `.../rest`. Their `doc.go` says the module root
has no importable API and names the subpackages.

## Commands

### SDK module verification

Run inside the changed module. `confloader` is a LAX module: use
`--new-from-merge-base=main` and skip gostyle until its redesign lands.

```bash
gofmt -l .
go vet ./...
go test -count=1 -shuffle=on ./...
go test -count=1 -race -shuffle=on ./...
golangci-lint run ./...
GOWORK=off go run ../.agents/skills/go-best-practices/scripts/gostyle.go ./...
go mod tidy -diff
go doc -all .
```

### Contract and import-map check

New repository check, no new Go module:

```bash
./.github/scripts/check-doc-contracts.sh
```

It fails when a public package has no `doc.go` contract, a required heading is
missing, a package comment is duplicated, a public import path is absent from
the root import map, or a module root is listed as importable. The script and
its tests live outside the release train.

### Eval kit setup

The kit is prose plus one setup script; it is not a Go module in the workspace.

```bash
docs/evals/agent-consumable-sdk/setup.sh db
# Hand the printed working directory and sample-db.md to the eval agent.
# The agent runs the done command; the grader fills correction-log.md.
```

### Release

```bash
# Bump VERSION and every sibling require to v0.1.1, then:
.github/scripts/release.sh "$(cat VERSION)" --check
gh workflow run release.yml
# The workflow tags every module and verifies proxy installs.
```

## Project Structure

```
docs/specs/agent-consumable-sdk/
  SPEC-agent-consumable-sdk.md       this spec
docs/evals/agent-consumable-sdk/
  README.md                          protocol: fixed fixtures, no shims,
                                     correction definition, stop rule
  sample-db.md                       six sample prompts and checklists
  sample-rest-server.md
  sample-nats-consumer.md
  sample-observability.md
  sample-confloader.md
  sample-errs.md
  correction-log.md                  correction-log template
  setup.sh                           copies a fixture to a temp dir, adds the
                                     local replace, prints the workdir
  fixtures/                          non-module templates for the six samples
README.md                            task→package import map
.github/scripts/check-doc-contracts.sh
<module>/<package>/doc.go            one per package in the set
nats/doc.go, rest/doc.go             doc-only roots
```

No service tree, no compose stack, no catalog checkout.

## Code Style

The contract is a Go package comment, not Markdown. It is 150–250 words,
excluding code. It starts with `Package <name> ...`, uses `[Link]` syntax, and
renders under `go doc` without backticks or HTML.

```go
// Package db provides a small, standardized named-query API for MySQL and
// PostgreSQL with OpenTelemetry telemetry.
//
// # When to use
//
// Use db when a service needs SQL with named parameters and a managed pool.
// It is not an ORM.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/db"
//
// # Minimal example
//
//	conn, err := db.Connect(ctx, db.Config{Driver: "pgx", DSN: dsn})
//	if err != nil {
//		return err
//	}
//	defer conn.Close()
//
// # Lifecycle
//
// Connect opens the pool; Close releases it. A Tx must be committed or rolled
// back.
//
// # Common mistakes
//
//   - Importing the module root instead of the package.
//   - Forgetting Close.
//
// # Test recipe
//
// Use sqlmock through Connect with Driver "sqlmock"; see the README.
package db
```

Import map style:

```markdown
| I need | Import | Notes |
| --- | --- | --- |
| Persistence | `github.com/fikrimohammad/go-servicekit/db` | MySQL/PostgreSQL |
| HTTP server | `github.com/fikrimohammad/go-servicekit/rest/server` | Hertz |
```

`nats` and `rest` are module roots with no importable API; the map says so
explicitly. No alias/facade packages.

Eval-kit style: harness-agnostic, self-contained, exact commands, checkbox
checklists, no links to this conversation. The prompt states the stop rule: if
the SDK lacks a required API, stop and report the gap; do not add a shim and do
not edit the SDK.

## Testing Strategy

- **Contracts:** no unit tests. The contract check validates presence, headings,
  single package comment, and import-map coverage; `go doc -all` validates
  rendering. Truth is validated by the evals, not by prose tests.
- **SDK modules:** existing unit tests plus the full module verification above.
  No new third-party dependencies; no coverage threshold change.
- **Eval kit:** manual by design. One dry run with a fresh agent proves the kit
  is self-contained; the dry run's questions become kit defects, not eval
  corrections.
- **Sample evals:** six manual runs with fixed prompts and fixtures; each pass
  requires zero SDK-usage corrections. A missing API is a finding, not a failure
  to fix.
- **Release:** every module installs from the proxy; the sample evals are
  spot-checked against the tag where a fixture allows it.
- **Not tested in CI:** agent behavior. CI runs the contract check and the
  existing module tests only.

## Boundaries

**Always do**

- Run the module verification before proposing a change.
- Keep the contract truthful; a contract that disagrees with the code is a bug.
- Update the import map when a public package is added, removed, or renamed.
- Log every SDK-usage correction with file, line, and reason.
- Log every missing capability as a finding with the package and the eval that
  found it.
- Pin each sample's fixture and prompt in the kit.

**Ask first**

- Adding a sample eval or fixture.
- Changing the eval-kit protocol or the correction definition.
- A CI workflow change.
- The release version or the release date.

**Never do**

- Change the SDK's API in this project — additive or breaking.
- Breaking API normalization.
- Migrate the SDK into a real service; that is a separate project.
- Build a harness, a docs site, `llms.txt`, or generated docs.
- Rewrite the 3,000-line module READMEs.
- Add a module or dependency for Kitex, pgx, MinIO, or gocron.
- Adapter shims that hide the old client.
- Add an OTLP/HTTP option or touch the migration runner.
- Skip the contract check or weaken an eval.
- Commit secrets.

## Success Criteria

1. All 21 public import paths have a `doc.go` contract with the six headings;
   `go doc -all` renders each; `check-doc-contracts.sh` passes.
2. The root import map lists all 21 import paths plus the doc-only roots, and no
   module root is listed as importable.
3. Every sample eval either passes with zero SDK-usage corrections or stops on a
   recorded finding; the correction log is complete.
4. The eval kit is self-contained: a fresh agent completes the dry run without
   asking a question the kit should have answered.
5. Every missing capability found by an eval is recorded as a finding with its
   package and the eval that surfaced it.
6. `v0.1.1` is tagged for every module, and every module installs from the
   proxy.
7. No exported signature changed; the diff contains documentation, the import
   map, the doc-only roots, the contract check, and the eval kit only.
8. The work lands inside five weeks; if the contracts slip, the release slips
   rather than the contract check being cut.

## Open Questions

None. Resolved 2026-11-08: no OTLP/HTTP option (gRPC stays); the migration runner
is untouched and out of scope; the release is `v0.1.1`; fixtures are in-repo
templates; the kit is `setup.sh` plus prose; the real-repo eval is a separate
project; API changes are out of scope.
