# Implementation Plan: agent-consumable-sdk (docs-only)

Spec: [SPEC-agent-consumable-sdk](../docs/specs/agent-consumable-sdk/SPEC-agent-consumable-sdk.md)
(re-scoped 2026-11-08). Intent:
[docs/intent/agent-consumable-sdk.md](../docs/intent/agent-consumable-sdk.md). Idea:
[docs/ideas/agent-consumable-sdk.md](../docs/ideas/agent-consumable-sdk.md). Tasks:
[todo.md](todo.md).

## Overview

One PR on branch `feat/agent-consumable-sdk`, one commit per task. It ships
documentation only:

- **Contracts:** a `doc.go` for all 21 public import paths with the six headings,
  plus a task→package root README and doc-only `nats`/`rest` roots.
- **Guard:** `.github/scripts/check-doc-contracts.sh`, its test harness, and a CI
  job.
- **Proof:** a self-contained eval kit with six in-process sample evals.
- **Release:** `v0.1.1`, docs only.

No exported API changes. A capability an eval finds missing is recorded as a
finding, not fixed here; the catalog migration is a separate project.

This plan replaces the completed guardrails plan (all tasks checked; PR #19
merged). The previous plan stays in git history.

## Execution

- Branch `feat/agent-consumable-sdk` off `main`; one commit per task, subject
  `<type>(<area>): <summary> (T<n>)`.
- Run the touched module's verification before each commit; tick the task in
  [todo.md](todo.md).
- Checkpoints are review gates: the owner (or a reviewer agent) signs off before
  the next phase.
- Tasks within a wave touch disjoint files and can run in parallel; see
  Parallelization.
- **Eval runner:** delegated by the owner (2026-11-08); T22–T23 use a fresh
  agent that sees only the kit and the workdir.

## Architecture Decisions

- **One package comment, in `doc.go`.** Every package in the set gets a
  `doc.go`; where the comment currently lives in `<pkg>.go`, it moves. The check
  enforces exactly one `// Package ` comment per package.
- **The check derives the package set.** It runs `go list` per module with
  `GOWORK=off`, excludes `/internal/` and test-only packages, and accepts
  optional import paths so a mid-project task can check only its own packages.
  README coverage is checked by exact path; `.../nats` and `.../rest` are the
  declared doc-only roots, never table rows.
- **The import map lists 21 rows.** `nats` and `rest` appear only in a "module
  roots are not importable" note; `observability` stays a row because it is the
  wiring guide (no API).
- **The eval kit is prose + `setup.sh` + static fixtures.** `setup.sh <sample>`
  copies a fixture to a temp dir, adds `replace` directives for all 11 SDK
  modules, runs `go mod download`, prints the workdir. No harness, no CI agent
  runs.
- **Evals run against the working tree; the tag is spot-checked.** Fixtures use
  local replaces pre-release; after `v0.1.1`, two samples re-run from the proxy
  without replaces.
- **The release bump is atomic.** `VERSION`, 11 sibling `require`s, and the
  README version line change together; that is a mechanical exception to the
  5-file task guideline.
- **Order:** contracts before prompts, prompts before evals, evals before the
  release. Catalog is out.

## Task List

Details, acceptance criteria, and verification are in [todo.md](todo.md).

### Phase 1: Foundation
- [x] T1: Commit the re-scoped docs and this plan; open a draft PR
- [x] T2: Root import map and doc-only `nats`/`rest` roots
- [x] T3: Contract shape and the `db` exemplar
- [x] T4: `check-doc-contracts.sh` plus its test harness

### Checkpoint A: foundation
- [x] README map lists 21 paths; `nats`/`rest` roots render under `go doc`
- [x] The `db` contract renders all six headings
- [x] The check script's tests pass; its only repo findings are the 20 unwritten
      contracts
- [x] Every module still passes vet and test

### Phase 2: Contracts (parallel)
- [x] T5: `errs`, `errgroup`, `identity`
- [x] T6: `redis`, `s3`
- [x] T7: `cron`, `cron/engine`
- [x] T8: `confloader`, `confloader/client`
- [x] T9: `confloader/client/etcd`, `confloader/client/infisical`
- [x] T10: `rest/client`, `rest/server`
- [x] T11: `nats/consumer`, `nats/producer`
- [x] T12: `observability` root, `observability/attributes`, `observability/logs`
- [x] T13: `observability/metrics`, `observability/tracer`

### Checkpoint B: contracts complete
- [x] `check-doc-contracts.sh` passes on the whole tree
- [x] All 21 contracts render under `go doc -all`; each is 150–250 words with no
      code counted
- [x] No exported API changed; every module passes vet, test, and lint
- [ ] Reviewer spot-checks three contracts against the code

### Phase 3: CI and eval kit
- [x] T14: Wire the contract check into CI (owner approval required)
- [x] T15: Eval-kit protocol, correction log, `setup.sh`
- [x] T16: `errs` sample (first complete kit)
- [x] T17: `db` sample
- [x] T18: `confloader` sample
- [x] T19: `rest/server` sample
- [x] T20: `observability` sample
- [x] T21: `nats/consumer` sample

### Checkpoint C: kit complete
- [x] `setup.sh <sample>` works for all six; each done command fails on its
      untouched fixture
- [x] The protocol, correction log, and prompts are self-contained
- [x] The CI doc-contracts job is green on the PR

### Phase 4: Proof
- [x] T22: Fresh-agent dry run; record kit defects
- [x] T23: Run the six sample evals and fill the correction log

### Checkpoint D: eval proof
- [ ] Zero SDK-usage corrections, or every one logged as a finding
- [ ] No shim in any diff
- [ ] Findings are attached to the PR

### Phase 5: Release
- [ ] T24: Bump `VERSION` and sibling requires to `v0.1.1`
- [ ] T25: Tag every module; verify proxy installs
- [ ] T26: Spot-check `db` and `errs` against `v0.1.1`

### Checkpoint E: release
- [ ] 11 tags at `v0.1.1` on one commit; verify-install job green
- [ ] Spot checks pass with no replaces
- [ ] Success criteria 1–8 of the spec hold

## Parallelization

- **Wave 1:** T2 and T3 in parallel (disjoint files); T4 after both.
- **Wave 2:** T5–T13 in parallel, capped at four concurrent tasks; each touches
  only its own packages.
- **Wave 3:** T14 and T15 in parallel; then T16–T21 in parallel, capped at
  three; each sample depends on its package contract.
- **Wave 4:** T22 → T23 sequential; they need the owner's eval runner.
- **Release:** T24 → T25 → T26 sequential.

## Risks and Mitigations

| Risk | Impact | Mitigation |
| --- | --- | --- |
| A contract drifts from the real API | High | Author from `go doc -all` and `example_test.go`; reviewer spot-checks three; sample evals catch wiring errors |
| The check is brittle (test-only packages, internals, roots) | Med | Test harness covers each failure mode and each exclusion; optional path args; README rules stated explicitly |
| `setup.sh` misses a transitive sibling replace | Med | It writes replaces for all 11 modules unconditionally; each sample smoke-tests it |
| Fixtures need network and pinned deps (sqlmock, nats-server) | Low | `go mod download` in setup; versions pinned in each fixture; the kit states the Go-only requirement |
| Eval runner variance masks a doc defect | Med | Fixed prompt, allowed files, done command, dry run first; every question becomes a kit defect |
| CI wiring is declined or delayed | Low | T14 is gated; the check still runs from the PR body and locally |
| 21 contracts are the critical path; a slip moves the release | Med | Parallel waves; Checkpoint B gates the kit; spec says the release slips, not the check |
| The version bump misses a sibling `require` | Low | `release.sh "$(cat VERSION)" --check`; one atomic commit |

## Open Questions

None. Resolved 2026-11-08: the CI workflow change is approved; the eval runner
is delegated by the owner (T22–T23 use a fresh agent with kit-only context).
