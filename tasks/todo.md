# Tasks: agent-consumable-sdk (docs-only)

Plan: [plan.md](plan.md). Spec:
[SPEC-agent-consumable-sdk](../docs/specs/agent-consumable-sdk/SPEC-agent-consumable-sdk.md).
Intent: [docs/intent/agent-consumable-sdk.md](../docs/intent/agent-consumable-sdk.md).

**Module checks.** Run in every module a task touches:

- `gofmt -l .`
- `go vet ./...`
- `go test -count=1 ./...` (add `-race` at checkpoints)
- `golangci-lint run ./...` (confloader: `--new-from-merge-base=main`)
- `GOWORK=off go run ../.agents/skills/go-best-practices/scripts/gostyle.go ./...`
  (strict modules only)
- `go mod tidy -diff`

**Doc checks.** From the repo root:

- `.github/scripts/check-doc-contracts.sh` (all packages) or
  `.github/scripts/check-doc-contracts.sh <import-path>...` (focused).
- `go doc -all` in each package directory, to confirm the headings render.

**TDD rule.** Only T4 adds behavior (the check script); it is test-first. Every
other task is documentation, shell glue, or a version bump.

Each task is one commit, with the subject `<type>(<area>): <summary> (T<n>)`.

---

## Task 1: Commit the re-scoped docs and this plan; open a draft PR

**Description:** Create `feat/agent-consumable-sdk` from `main`. Commit the three
re-scoped docs, this plan, and this task list. Open a draft PR that states the
docs-only scope, links the spec, and lists the six sample evals.

**Acceptance criteria:**
- [ ] The branch has one commit containing only the three docs and `tasks/`.
- [ ] The draft PR links the spec and states: docs-only, no API changes,
      `v0.1.1`, catalog migration out of scope.
- [ ] `main` is untouched and the working tree is clean.

**Verification:**
- [ ] `git show --stat HEAD` lists exactly the five docs/task files.
- [ ] `git status --short` is empty.
- [ ] `gh pr view --json isDraft,title` shows the draft.

**Dependencies:** None
**Files likely touched:**
- `docs/ideas/agent-consumable-sdk.md`
- `docs/intent/agent-consumable-sdk.md`
- `docs/specs/agent-consumable-sdk/SPEC-agent-consumable-sdk.md`
- `tasks/plan.md`
- `tasks/todo.md`

**Estimated scope:** S

---

## Task 2: Root import map and doc-only `nats`/`rest` roots

**Description:** Replace the README "Modules" table with a task→package import
map for the 21 import paths. Add `nats/doc.go` and `rest/doc.go` that say the
module root has no importable API and name the subpackages. Reword the
root parity-test headers that claim the roots have no other Go files.

**Acceptance criteria:**
- [ ] The README has one `| I need | Import | Notes |` table with exactly the 21
      spec paths; a separate note lists `.../nats` and `.../rest` as module
      roots with no importable package.
- [ ] `nats/doc.go` and `rest/doc.go` start with `Package nats` / `Package rest`,
      name the subpackages, and render under `go doc`.
- [ ] The root parity tests still pass and their headers no longer say the root
      has no `.go` files.

**Verification:**
- [ ] `awk` count of table rows in `README.md` is 21.
- [ ] `(cd nats && go test ./... && go doc .)` and `(cd rest && go test ./... && go doc .)`.
- [ ] `go vet ./...` in both modules.

**Dependencies:** T1
**Files likely touched:**
- `README.md`
- `nats/doc.go`
- `rest/doc.go`
- `nats/parity_test.go`
- `rest/parity_test.go`

**Estimated scope:** S

---

## Task 3: Contract shape and the `db` exemplar

**Description:** Establish the template every later contract follows. Create
`db/doc.go` with the six headings and move the existing package comment out of
`db/db.go`. The content covers connect/close, named queries, transactions, and
the sqlmock test recipe.

**Acceptance criteria:**
- [ ] `db/doc.go` starts `// Package db `, has `# When to use`, `# Import`,
      `# Minimal example`, `# Lifecycle`, `# Common mistakes`, `# Test recipe`
      in that order, and is 150–250 words excluding code.
- [ ] The snippet mirrors `db/example_test.go` and uses only the public API.
- [ ] `db` has exactly one `// Package ` comment, and it is in `doc.go`.

**Verification:**
- [ ] `(cd db && gofmt -l . && go vet ./... && go test -count=1 ./...)`.
- [ ] `(cd db && go doc -all .)` shows all six sections.
- [ ] `grep -rn '^// Package ' db --include='*.go'` returns one line, in
      `db/doc.go`.

**Dependencies:** T1
**Files likely touched:**
- `db/doc.go`
- `db/db.go`

**Estimated scope:** S

---

## Task 4: `check-doc-contracts.sh` plus its test harness

**Description:** Write the repository check and its tests. The script derives the
public package set per module with `go list`, excludes `/internal/` and
test-only packages, treats `.../nats` and `.../rest` as the declared doc-only
roots (never table rows), and validates the six headings, the single package
comment, and README coverage. It accepts optional import paths for focused
checks.

**Acceptance criteria:**
- [ ] The script fails on each of: missing `doc.go`; a missing heading; a package
      comment outside `doc.go` or a duplicate; a public path absent from the
      README table; `.../nats` or `.../rest` listed as a table row.
- [ ] The test harness builds a temp module and README per failure mode and
      asserts exit code and message; it passes.
- [ ] On the current tree the script fails and lists exactly the 20 contracts
      not yet written; focused mode passes on `db`.

**Verification:**
- [ ] `bash -n .github/scripts/check-doc-contracts.sh`.
- [ ] `bash .github/scripts/check-doc-contracts_test.sh`.
- [ ] `bash .github/scripts/check-doc-contracts.sh github.com/fikrimohammad/go-servicekit/db`
      exits 0.

**Dependencies:** T2, T3
**Files likely touched:**
- `.github/scripts/check-doc-contracts.sh`
- `.github/scripts/check-doc-contracts_test.sh`

**Estimated scope:** S

---

### Checkpoint A: foundation

- [ ] The README map and both doc-only roots are in place.
- [ ] The `db` exemplar renders all six sections.
- [ ] The check script's tests pass; its only repo findings are unwritten
      contracts.
- [ ] Every module passes vet and test.
- [ ] Reviewer approves the contract shape before the parallel wave.

---

## Task 5: `errs`, `errgroup`, `identity` contracts

**Description:** Write the three foundation contracts. `errs` keeps its current
substance (codes, `errors.Is`, `SlogAttr`) and gains the six headings;
`errgroup` (limit, panic safety) and `identity` (name/version/env) move their
comments into `doc.go`.

**Acceptance criteria:**
- [ ] Each `doc.go` has the six headings, 150–250 words excluding code, and
      starts `Package <name>`.
- [ ] Each package has exactly one `// Package ` comment, in `doc.go`.
- [ ] Each snippet mirrors that package's `example_test.go`.

**Verification:**
- [ ] Module checks in `errs`, `errgroup`, `identity`.
- [ ] `go doc -all .` in each package directory.
- [ ] `check-doc-contracts.sh` with the three import paths exits 0.

**Dependencies:** T3
**Files likely touched:**
- `errs/doc.go`, `errgroup/doc.go`, `errgroup/errgroup.go`
- `identity/doc.go`, `identity/identity.go`

**Estimated scope:** M

---

## Task 6: `redis`, `s3` contracts

**Description:** Document the two storage clients. `redis` covers construction,
pool lifecycle, and metrics/tracing; `s3` covers the client, multipart upload,
and presigned URLs.

**Acceptance criteria:**
- [ ] Both contracts have the six headings, 150–250 words excluding code, and
      start `Package <name>`.
- [ ] Each package has exactly one `// Package ` comment, in `doc.go`.
- [ ] Each snippet mirrors its package's `example_test.go`.

**Verification:**
- [ ] Module checks in `redis` and `s3`.
- [ ] `go doc -all .` in each package directory.
- [ ] `check-doc-contracts.sh` with both import paths exits 0.

**Dependencies:** T3
**Files likely touched:**
- `redis/doc.go`, `redis/redis.go`
- `s3/doc.go`, `s3/s3.go`

**Estimated scope:** S

---

## Task 7: `cron`, `cron/engine` contracts

**Description:** Document the job model and the engine: scheduling, start/stop,
middleware, and the engine's lifecycle.

**Acceptance criteria:**
- [ ] Both contracts have the six headings, 150–250 words excluding code, and
      start `Package <name>`.
- [ ] Each package has exactly one `// Package ` comment, in `doc.go`.
- [ ] Each snippet mirrors that package's `example_test.go`.

**Verification:**
- [ ] Module checks in `cron`.
- [ ] `go doc -all .` in `cron` and `cron/engine`.
- [ ] `check-doc-contracts.sh` with both import paths exits 0.

**Dependencies:** T3
**Files likely touched:**
- `cron/doc.go`, `cron/cron.go`
- `cron/engine/doc.go`, `cron/engine/engine.go`

**Estimated scope:** S

---

## Task 8: `confloader`, `confloader/client` contracts

**Description:** Document the loader (file + dynamic + secrets, `New`, `Data`)
and the backend contract interface. `confloader/client` has no
`example_test.go`; derive its snippet from the package tests and README.

**Acceptance criteria:**
- [ ] Both contracts have the six headings, 150–250 words excluding code, and
      start `Package <name>`.
- [ ] `confloader/client` names the contract role and the concrete backends
      without documenting their constructors.
- [ ] Each package has exactly one `// Package ` comment, in `doc.go`.

**Verification:**
- [ ] Module checks in `confloader` (LAX lint command).
- [ ] `go doc -all .` in both package directories.
- [ ] `check-doc-contracts.sh` with both import paths exits 0.

**Dependencies:** T3
**Files likely touched:**
- `confloader/doc.go`, `confloader/confloader.go`
- `confloader/client/doc.go`, `confloader/client/client.go`

**Estimated scope:** S

---

## Task 9: `confloader/client/etcd`, `confloader/client/infisical` contracts

**Description:** Document the two backends: constructors, config defaults,
authentication, and `Close`. Neither has `example_test.go`; derive snippets from
the package tests and README.

**Acceptance criteria:**
- [ ] Both contracts have the six headings, 150–250 words excluding code, and
      start `Package <name>`.
- [ ] Each states its config defaults and lifecycle explicitly.
- [ ] Each package has exactly one `// Package ` comment, in `doc.go`.

**Verification:**
- [ ] Module checks in `confloader`.
- [ ] `go doc -all .` in both package directories.
- [ ] `check-doc-contracts.sh` with both import paths exits 0.

**Dependencies:** T3
**Files likely touched:**
- `confloader/client/etcd/doc.go`, `confloader/client/etcd/etcd.go`
- `confloader/client/infisical/doc.go`,
  `confloader/client/infisical/infisical.go`

**Estimated scope:** S

---

## Task 10: `rest/client`, `rest/server` contracts

**Description:** Document the HTTP boundary. `rest/server` covers middleware
order, `Error`/problem+json, request ID, and `Run`; `rest/client` covers `New`,
retry, timeouts, the `RoundTripper`, and the no-propagation rule.

**Acceptance criteria:**
- [ ] Both contracts have the six headings, 150–250 words excluding code, and
      start `Package <name>`.
- [ ] The server states the fixed middleware order and the 5xx/4xx log levels;
      the client states that it never propagates `traceparent` or
      `X-Request-ID` and redacts query strings.
- [ ] Each package has exactly one `// Package ` comment, in `doc.go`.

**Verification:**
- [ ] Module checks in `rest`.
- [ ] `go doc -all .` in `rest/client` and `rest/server`.
- [ ] `check-doc-contracts.sh` with both import paths exits 0.

**Dependencies:** T3
**Files likely touched:**
- `rest/client/doc.go`, `rest/client/client.go`
- `rest/server/doc.go`, `rest/server/server.go`

**Estimated scope:** M

---

## Task 11: `nats/consumer`, `nats/producer` contracts

**Description:** Document JetStream consumption (durable consumer, handler,
lifecycle) and production (publish, headers, flush). The test recipe names the
in-process NATS server used by the module's tests.

**Acceptance criteria:**
- [ ] Both contracts have the six headings, 150–250 words excluding code, and
      start `Package <name>`.
- [ ] The consumer states its ack/retry behavior; the producer states the
      publish and flush lifecycle.
- [ ] Each package has exactly one `// Package ` comment, in `doc.go`.

**Verification:**
- [ ] Module checks in `nats`.
- [ ] `go doc -all .` in both package directories.
- [ ] `check-doc-contracts.sh` with both import paths exits 0.

**Dependencies:** T3
**Files likely touched:**
- `nats/consumer/doc.go`, `nats/consumer/consumer.go`
- `nats/producer/doc.go`, `nats/producer/producer.go`

**Estimated scope:** M

---

## Task 12: `observability` root, `attributes`, `logs` contracts

**Description:** Rewrite the root as the wiring guide (no API): which package
installs what and in which order. Document `attributes` key normalization and
`logs` (logger install, `trace_id`).

**Acceptance criteria:**
- [ ] The root says it has no API and is the wiring guide; all three have the
      six headings and 150–250 words excluding code.
- [ ] Each package has exactly one `// Package ` comment, in `doc.go`.
- [ ] Snippets mirror `example_test.go` in each package.

**Verification:**
- [ ] Module checks in `observability`.
- [ ] `go doc -all .` in all three package directories.
- [ ] `check-doc-contracts.sh` with the three import paths exits 0.

**Dependencies:** T3
**Files likely touched:**
- `observability/doc.go`
- `observability/attributes/doc.go`, `observability/attributes/attributes.go`
- `observability/logs/doc.go`, `observability/logs/logs.go`

**Estimated scope:** M

---

## Task 13: `observability/metrics`, `observability/tracer` contracts

**Description:** Document the metrics and tracer clients: construction, config
defaults, lifecycle, and the OTLP/gRPC exporter. No HTTP exporter option.

**Acceptance criteria:**
- [ ] Both contracts have the six headings, 150–250 words excluding code, and
      start `Package <name>`.
- [ ] Each states its config defaults and that the exporter is OTLP/gRPC.
- [ ] Each package has exactly one `// Package ` comment, in `doc.go`.

**Verification:**
- [ ] Module checks in `observability`.
- [ ] `go doc -all .` in both package directories.
- [ ] `check-doc-contracts.sh` with both import paths exits 0.

**Dependencies:** T3
**Files likely touched:**
- `observability/metrics/doc.go`, `observability/metrics/metrics.go`
- `observability/tracer/doc.go`, `observability/tracer/tracer.go`

**Estimated scope:** S

---

### Checkpoint B: contracts complete

- [ ] `check-doc-contracts.sh` exits 0 on the whole tree.
- [ ] `go doc -all` renders all 21 contracts; word counts are in range.
- [ ] No exported signature changed (`git diff --stat` shows `doc.go` moves and
      comment removals only).
- [ ] Every module passes the full verification, including `-race`.
- [ ] A reviewer spot-checks three contracts against the code.

---

## Task 14: Wire the contract check into CI

**Description:** Add a `doc-contracts` job to `.github/workflows/ci.yml`:
checkout, Go 1.26.5, run the script. No matrix, no Docker. Mention it in the
README's CI section. Approved by the owner on 2026-11-08 (the spec lists CI
workflow changes under "Ask first").

**Acceptance criteria:**
- [ ] The job runs on PRs and pushes to `main`, alongside the existing jobs,
      and the README's CI section names it.
- [ ] It invokes the script from the repo root and fails the PR on a contract
      violation.
- [ ] The approval is recorded (plan, 2026-11-08), and the PR notes it.

**Verification:**
- [ ] The workflow YAML parses.
- [ ] The PR's CI run shows a green `doc-contracts` job.
- [ ] Reverting one heading locally makes the job fail (checked in a scratch
      branch).

**Dependencies:** T4, T5–T13
**Files likely touched:**
- `.github/workflows/ci.yml`
- `README.md`

**Estimated scope:** S

---

## Task 15: Eval-kit protocol, correction log, `setup.sh`

**Description:** Write the kit protocol (`docs/evals/agent-consumable-sdk/README.md`),
the correction-log template, and `setup.sh`. The protocol is self-contained and
harness-agnostic: fixed fixture, prompt, allowed files, done command, grader
checklist, stop rule, correction definition.

**Acceptance criteria:**
- [ ] The README defines all protocol terms, names the six samples, and contains
      no link to this conversation or the spec's internal debates.
- [ ] `setup.sh <sample>` validates the sample, copies `fixtures/<sample>` to a
      temp dir, adds replaces for all 11 modules found via `find -maxdepth 2
      -name go.mod`, runs `go mod download`, and prints the workdir; no args or
      an unknown sample exits 2 with usage.
- [ ] `correction-log.md` has one row per correction: sample, file:line, kind,
      reason, resolution.

**Verification:**
- [ ] `bash -n docs/evals/agent-consumable-sdk/setup.sh`.
- [ ] `setup.sh` and `setup.sh nope` both exit 2 with usage.
- [ ] `grep -c '^|' correction-log.md` shows the template row.

**Dependencies:** T1
**Files likely touched:**
- `docs/evals/agent-consumable-sdk/README.md`
- `docs/evals/agent-consumable-sdk/correction-log.md`
- `docs/evals/agent-consumable-sdk/setup.sh`

**Estimated scope:** S

---

## Task 16: `errs` sample (first complete kit)

**Description:** Add `sample-errs.md` and `fixtures/errs`: a `go.mod` and a
failing test that asks the agent to construct, wrap, and match an error by code,
then log it with `SlogAttr`. This proves `setup.sh` end to end.

**Acceptance criteria:**
- [ ] `setup.sh errs` prints a workdir where `go mod download` succeeded and
      `go test ./...` fails on the untouched fixture.
- [ ] The prompt states goal, allowed file, done command, and stop rule; the
      fixture needs only the SDK and the standard library.
- [ ] The grader checklist names what counts as an SDK-usage correction for this
      sample.

**Verification:**
- [ ] `setup.sh errs`, then `go test ./...` in the printed dir — fails.
- [ ] `grep -rn 'spec\|conversation\|previous' sample-errs.md fixtures/errs` is
      empty.
- [ ] `go doc` in the workdir shows the `errs` contract.

**Dependencies:** T15, T5
**Files likely touched:**
- `docs/evals/agent-consumable-sdk/sample-errs.md`
- `docs/evals/agent-consumable-sdk/fixtures/errs/go.mod`
- `docs/evals/agent-consumable-sdk/fixtures/errs/errors_test.go`

**Estimated scope:** S

---

## Task 17: `db` sample

**Description:** Add `sample-db.md` and `fixtures/db`: connect through the
sqlmock driver, run a named query, and commit a transaction. The fixture pins
`go-sqlmock` and fails until the agent wires the SDK correctly.

**Acceptance criteria:**
- [ ] `setup.sh db` yields a workdir where `go test ./...` fails on the untouched
      fixture and the only SDK import needed is `.../db`.
- [ ] The prompt states goal, allowed file, done command, stop rule, and grader
      checklist.
- [ ] The fixture pins `github.com/DATA-DOG/go-sqlmock` and documents the
      network requirement.

**Verification:**
- [ ] `setup.sh db`, then `go test ./...` — fails as expected.
- [ ] `grep -rn 'minio\|pgx' fixtures/db` is empty (no leaked old client).
- [ ] The prompt names the sqlmock recipe from the `db` contract.

**Dependencies:** T15, T3
**Files likely touched:**
- `docs/evals/agent-consumable-sdk/sample-db.md`
- `docs/evals/agent-consumable-sdk/fixtures/db/go.mod`
- `docs/evals/agent-consumable-sdk/fixtures/db/db_test.go`

**Estimated scope:** S

---

## Task 18: `confloader` sample

**Description:** Add `sample-confloader.md` and `fixtures/confloader`: implement a
fake `client.Client`, load a struct through `New`/`Data`, and read a nested key.

**Acceptance criteria:**
- [ ] `setup.sh confloader` yields a workdir whose `go test ./...` fails until
      the agent wires the loader.
- [ ] The fixture's fake client is the only backend; no network.
- [ ] The prompt states goal, allowed file, done command, stop rule, and grader
      checklist.

**Verification:**
- [ ] `setup.sh confloader`, then `go test ./...` — fails as expected.
- [ ] `go test` in the workdir does not dial etcd or Infisical.
- [ ] The prompt references the `confloader/client` contract.

**Dependencies:** T15, T8
**Files likely touched:**
- `docs/evals/agent-consumable-sdk/sample-confloader.md`
- `docs/evals/agent-consumable-sdk/fixtures/confloader/go.mod`
- `docs/evals/agent-consumable-sdk/fixtures/confloader/loader_test.go`

**Estimated scope:** S

---

## Task 19: `rest/server` sample

**Description:** Add `sample-rest-server.md` and `fixtures/rest-server`: build a
server with one route and one `Error` response, then assert the problem+json body
and the request ID header with `httptest`.

**Acceptance criteria:**
- [ ] `setup.sh rest-server` yields a workdir whose `go test ./...` fails until
      the agent wires the server.
- [ ] The test asserts status, `application/problem+json`, and the `request_id`
      extension member.
- [ ] The prompt states goal, allowed file, done command, stop rule, and grader
      checklist.

**Verification:**
- [ ] `setup.sh rest-server`, then `go test ./...` — fails as expected.
- [ ] The workdir test binds only `httptest`; no fixed port.
- [ ] The prompt references the `rest/server` contract.

**Dependencies:** T15, T10
**Files likely touched:**
- `docs/evals/agent-consumable-sdk/sample-rest-server.md`
- `docs/evals/agent-consumable-sdk/fixtures/rest-server/go.mod`
- `docs/evals/agent-consumable-sdk/fixtures/rest-server/server_test.go`

**Estimated scope:** S

---

## Task 20: `observability` sample

**Description:** Add `sample-observability.md` and `fixtures/observability`: wire
logs, metrics, and tracer from config, emit one of each, and assert no exporter
is required.

**Acceptance criteria:**
- [ ] `setup.sh observability` yields a workdir whose `go test ./...` fails until
      the agent wires the three clients.
- [ ] The fixture needs no collector; export errors are ignored by design.
- [ ] The prompt states goal, allowed file, done command, stop rule, and grader
      checklist.

**Verification:**
- [ ] `setup.sh observability`, then `go test ./...` — fails as expected.
- [ ] The test does not bind a network port.
- [ ] The prompt references the root wiring guide and the three client
      contracts.

**Dependencies:** T15, T12
**Files likely touched:**
- `docs/evals/agent-consumable-sdk/sample-observability.md`
- `docs/evals/agent-consumable-sdk/fixtures/observability/go.mod`
- `docs/evals/agent-consumable-sdk/fixtures/observability/wiring_test.go`

**Estimated scope:** S

---

## Task 21: `nats/consumer` sample

**Description:** Add `sample-nats-consumer.md` and `fixtures/nats-consumer`:
start the in-process NATS server with JetStream, publish one message, consume it
with the SDK consumer, and ack it.

**Acceptance criteria:**
- [ ] `setup.sh nats-consumer` yields a workdir whose `go test ./...` fails until
      the agent wires the consumer.
- [ ] The fixture pins `nats-server/v2` and starts it in-process with
      `Port: -1`.
- [ ] The prompt states goal, allowed file, done command, stop rule, and grader
      checklist.

**Verification:**
- [ ] `setup.sh nats-consumer`, then `go test ./...` — fails as expected.
- [ ] The workdir test opens no external connection.
- [ ] The prompt references the `nats/consumer` contract.

**Dependencies:** T15, T11
**Files likely touched:**
- `docs/evals/agent-consumable-sdk/sample-nats-consumer.md`
- `docs/evals/agent-consumable-sdk/fixtures/nats-consumer/go.mod`
- `docs/evals/agent-consumable-sdk/fixtures/nats-consumer/consumer_test.go`

**Estimated scope:** S

---

### Checkpoint C: kit complete

- [ ] All six samples run through `setup.sh`; each done command fails untouched.
- [ ] The protocol, correction log, and prompts are self-contained.
- [ ] The CI `doc-contracts` job is green on the PR.
- [ ] Reviewer approves the kit before the eval runs.

---

## Task 22: Fresh-agent dry run

**Description:** Hand the `errs` kit to a fresh agent that sees only the kit and
the workdir. Record every question and every doc defect. Fix kit defects; file
contract defects for the owning task.

**Acceptance criteria:**
- [ ] The agent completes the sample, or stops per the stop rule, without asking
      a question the kit should have answered.
- [ ] Every question is recorded and either fixed in the kit or filed as a
      finding.
- [ ] The dry run needs no SDK-usage correction.

**Verification:**
- [ ] The transcript is linked in the PR.
- [ ] `git diff` of kit fixes is limited to `docs/evals/`.
- [ ] The agent's commands include `go doc` (recorded in the transcript).

**Dependencies:** T16–T21
**Files likely touched:** `docs/evals/agent-consumable-sdk/**`
**Estimated scope:** S

---

## Task 23: Run the six sample evals

**Description:** Run each sample with a fresh agent, grade the diff against the
checklist, and fill `correction-log.md`. Findings name the package and the
missing capability; nothing is fixed in the SDK.

**Acceptance criteria:**
- [ ] Each sample's done command passes.
- [ ] Zero SDK-usage corrections, or every correction has a finding naming the
      package and the missing API.
- [ ] No shim in any diff; the correction log is complete.

**Verification:**
- [ ] `correction-log.md` has a row per run, even when empty.
- [ ] `grep -rn 'type .*shim\|adapter'` over the diffs is empty.
- [ ] The PR lists the findings.

**Dependencies:** T22
**Files likely touched:** `docs/evals/agent-consumable-sdk/correction-log.md`
**Estimated scope:** S

---

### Checkpoint D: eval proof

- [ ] Zero corrections, or every correction logged as a finding.
- [ ] No shim in any diff.
- [ ] Findings are attached to the PR and will be in the release notes.

---

## Task 24: Bump `VERSION` and sibling requires to `v0.1.1`

**Description:** Bump `VERSION`, all 11 sibling `require`s, and the README
version line. The diff is mechanical and atomic; it is an accepted exception to
the 5-file guideline.

**Acceptance criteria:**
- [ ] `release.sh "$(cat VERSION)" --check` passes.
- [ ] Every sibling `require` is `v0.1.1`; no `v0.1.0` sibling remains.
- [ ] The README "Versioning" section says `v0.1.1`.

**Verification:**
- [ ] `.github/scripts/release.sh "$(cat VERSION)" --check`.
- [ ] `grep -rn 'go-servicekit/[a-z/]* v0.1.0' --include=go.mod .` is empty.
- [ ] `git diff --stat` lists `VERSION`, 11 `go.mod` files, and `README.md`.

**Dependencies:** T23
**Files likely touched:**
- `VERSION`
- `errs/go.mod`, `errgroup/go.mod`, `identity/go.mod`, `observability/go.mod`,
  `confloader/go.mod`, `rest/go.mod`, `cron/go.mod`, `db/go.mod`,
  `nats/go.mod`, `redis/go.mod`, `s3/go.mod`
- `README.md`

**Estimated scope:** M (mechanical)

---

## Task 25: Tag every module and verify proxy installs

**Description:** Run the release workflow on `main` for `v0.1.1`; wait for the
verify-install job. Draft release notes listing the six samples and any findings.

**Acceptance criteria:**
- [ ] 11 tags `<module>/v0.1.1` exist on one commit.
- [ ] The `verify-install` job is green; every module installs from
      proxy.golang.org.
- [ ] Release notes state docs-only and list the findings.

**Verification:**
- [ ] `gh workflow run release.yml` completes; the run URL is in the PR.
- [ ] `git ls-remote --tags origin '*/v0.1.1'` lists 11 tags.
- [ ] In a temp module, `go get github.com/fikrimohammad/go-servicekit/db@v0.1.1`
      succeeds.

**Dependencies:** T24
**Files likely touched:** None (tags and release notes)
**Estimated scope:** S

---

## Task 26: Spot-check `db` and `errs` against `v0.1.1`

**Description:** Re-run two samples from the proxy version: remove the replaces,
`go get @v0.1.1`, and run the done command. This proves the contracts travel in
the module cache.

**Acceptance criteria:**
- [ ] Both done commands pass without local replaces.
- [ ] `go doc` from the proxy version shows all six headings.
- [ ] No corrections; any gap is logged.

**Verification:**
- [ ] The two temp workdirs have no `replace` lines.
- [ ] `go test ./...` passes in both.
- [ ] `go doc -all github.com/fikrimohammad/go-servicekit/db` shows the headings.

**Dependencies:** T25
**Files likely touched:** None (temporary copies)
**Estimated scope:** S

---

### Checkpoint E: release

- [ ] 11 tags at `v0.1.1` on one commit; verify-install green.
- [ ] Spot checks pass with no replaces.
- [ ] Spec success criteria 1–8 hold; the PR is merged.
