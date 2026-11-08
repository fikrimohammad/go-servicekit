# Agent-Consumable SDK

**Status:** idea, refined with resolved decisions (2026-11-03); re-scoped to
docs-only (2026-11-08). Scope:
[docs/intent/agent-consumable-sdk.md](../intent/agent-consumable-sdk.md).

## Problem Statement

How might we let a coding agent in a business repo go from a task like "add
Postgres to this service" to correct, tested SDK usage using only the Go
toolchain and module cache — with no access to this repo and no human
correction?

## Recommended Direction

Make `doc.go` the consumer contract for **all 21 public import paths**, not a
golden-path subset. Each package gets a 150–250 word package doc: when to use
it, the exact import path, a minimal correct snippet, lifecycle, config
defaults, common mistakes, and a test recipe. The five-field handover brief
(context, packages, pattern, files, done, stop if) is the authoring checklist;
the rendered headings are plain Go doc, not process jargon.

Fix the import map in the same pass. The root README calls modules "packages"
and lists `.../nats` and `.../rest`, which have no importable files. Rewrite it
as a task→package map and add doc-only roots for `nats` and `rest` that point
at their subpackages.

Prove the contracts with **six sample evals** run by a different agent: `db`,
`rest/server`, `nats/consumer`, `observability`, `confloader`, and `errs`. Each
is a fixed fixture, a fixed prompt, a done command, a grader checklist, and a
correction log. Every sample runs in-process — sqlmock for `db`, the embedded
NATS server for `nats/consumer`, a fake client for `confloader` — so there is
no compose stack and no external service.

**No API changes.** This project ships documentation. When an eval shows the
SDK lacks a capability, the gap is recorded as a finding; changing the API is a
separate project. Migrating the SDK into a real service — the catalog service
of monster-hunter-wilds-builder is the candidate — is also a separate project:
it needs the SDK implemented in a real repo, and this one ships the contracts
and the kit that project will consume.

Eight weeks was the original budget with a catalog migration. Without it the
plan is five weeks, including the release.

## Decisions (2026-11-03, amended 2026-11-08)

- **Surface:** toolchain-first (`go get`, `go doc`, module cache); web is a
  bonus.
- **Coverage:** all 21 public import paths, including implementation packages
  and the API-less roots.
- **Proof:** six sample evals, run by another agent from a self-contained kit;
  the real-repo eval is a separate project.
- **Correction:** only a human edit to **SDK usage** counts — imports,
  constructor/config wiring, lifecycle calls, SDK error handling, telemetry
  wiring. Business-logic or formatting edits do not.
- **Shims:** none. Where the SDK is used, it is called directly; the eval fails
  if a compatibility wrapper keeps the old client underneath.
- **API:** docs-only. No additive or breaking API change; a missing capability
  is a finding, not work.
- **Fixtures:** in-repo non-module templates under
  `docs/evals/agent-consumable-sdk/fixtures/`.
- **Automation:** one `setup.sh <sample>` plus prose; no harness, no CI agent
  runs.
- **Sample evals:** `db`, `rest/server`, `nats/consumer`, `observability`,
  `confloader`, `errs`.
- **Timeline:** five weeks, including the release/tag.
- **Execution:** import map → 21 contracts → eval kit → sample evals → release.
- **Five-field brief:** authoring checklist only. Public docs use plain
  headings: When to use, Import, Minimal example, Lifecycle, Common mistakes,
  Test recipe.
- **Observability:** unchanged; OTLP/gRPC stays the standard. A future
  real-repo migration adapts its exporter config instead of growing the SDK.
- **Migration runner:** out of scope. Schema tooling is untouched and never
  mentioned in the contracts.

## Key Assumptions to Validate

Checked means accepted by the owner on 2026-11-03; re-scoped on 2026-11-08. The
evals remain the empirical test.

- [x] Agents consult `go doc` or the module cache. Test: record commands in the
      eval transcripts; if they never do, the surface choice is wrong.
- [x] All 21 contracts can be written in three weeks without losing accuracy.
      Test: every public import path exists by week 4; `go doc -all` renders.
- [x] A 150–250 word contract plus one snippet prevents wrong wiring. Test:
      zero SDK-usage corrections on the six sample evals.
- [x] Doc-only roots fix the `nats`/`rest` trap without alias packages. Test:
      no root imports in the eval diffs.
- [x] Another agent can run the eval kit without this conversation. Test: a
      fresh agent reads only the kit and completes the run; record its
      questions as kit defects.
- [x] Five weeks is enough with contracts first. Test: week-4 contracts,
      week-5 samples and release; if the contracts slip, the release slips.
- [x] Docs stay true without a generator. Test: import-map coverage check;
      snippets come from the sample evals.

## MVP Scope

1. **Import map** (week 1). Root README task→package table; doc-only `doc.go`
   for `nats`, `rest`, and the `observability` root; an explicit "module roots
   are not importable" note.
2. **21 contracts** (weeks 2–4). Every public import path gets the same
   structure: when to use, import path, minimal snippet, lifecycle, config
   defaults, common mistakes, test recipe. Plain headings; the five-field brief
   is the checklist.
3. **Eval kit** (weeks 2–4, alongside the contracts). Six sample kits: fixed
   fixture, prompt, allowed files, done command, grader checklist, and
   correction log. Harness-agnostic; no context from this session.
4. **Sample evals** (weeks 4–5). Six manual runs with fixed prompts and
   fixtures; each pass requires zero SDK-usage corrections. Gaps become
   findings, not code changes.
5. **Release** (week 5). Tag every module at `v0.1.1`; verify install from the
   proxy; spot-check the sample evals against the tag where the fixture allows
   it.

## Schedule Risk

Contracts-first makes the critical path three weeks of contracts followed by
the eval kit and the samples. There is no catalog migration to absorb slack, so
a contract slip moves the release one-for-one. The mitigation is running the
sample kits in parallel with the last contracts and keeping the project free of
API work.

## Not Doing (and Why)

- **Migrating the SDK into a real service.** Catalog migration is its own
  project, with its own owner, schedule, and integration gate. This project
  ships the contracts and the kit that project will consume.
- **Additive API changes.** Docs-only. A gap found by an eval is recorded as a
  finding; changing the SDK is a different project.
- **An OTLP/HTTP option.** `observability` is unchanged and gRPC is the
  standard; the real-repo project adapts its exporter config.
- **Migration-runner work.** Schema tooling is not an SDK consumer concern.
- **Uniform API normalization.** Breaking across 11 modules; docs-only does not
  open that door.
- **Rewriting the 3,000 lines of module READMEs.** The README stays the
  long-form appendix; the contract lives in `doc.go`.
- **A docs site, `llms.txt`, or generated docs.** Toolchain-first; a fourth
  copy of the truth.
- **Alias or facade packages for `nats` and `rest`.** A new API surface to
  maintain; doc-only roots fix discovery without code.
- **A full eval harness or CI-triggered agent evals.** Non-deterministic, slow,
  and coupled to one agent CLI; `setup.sh` plus prose keeps the kit
  harness-agnostic and manual.
- **Temporary adapter shims.** They would hide the SDK and invalidate the eval.
- **A `servicekit` CLI or generator.** The 10x version; later, only if the eval
  shows the contract is still not enough.
- **A synthetic `examples/gateway` service.** The sample fixtures are the eval;
  no service stands in for a repo.
- **Per-package evals for all 21.** The contracts cover all 21; the evals
  sample the hard shapes.
- **A `confloader` redesign.** Its contract documents the current API; the
  redesign is a separate track.
- **A LICENSE.** Not on the critical path; add it separately if the repo stays
  public.

## Resolved Details (2026-11-08)

- **Scope:** docs-only; no API changes.
- **Real-repo eval:** out; a separate project.
- **Release:** `v0.1.1`, tagged inside the five weeks.
- **Fixtures:** in-repo non-module templates.
- **Automation:** `setup.sh` plus prose.
- **Observability:** unchanged, OTLP/gRPC.
- **Migration runner:** untouched.
- **Open questions:** none.

## Success

- All 21 public import paths have a contract; the import map has no dead roots.
- The six sample evals pass with zero SDK-usage corrections.
- The agent never imports `.../nats`, `.../rest`, or `.../observability`
  expecting a root API.
- The eval kit is self-contained: a fresh agent can run it without asking a
  question the kit should have answered.
- Every SDK gap found by an eval is recorded as a finding with its package and
  the eval that surfaced it.
- The release is tagged, and every module installs from the proxy.
