# Intent: Agent-Consumable SDK

**Status:** confirmed, 2026-11-03; re-scoped 2026-11-08
**Related:** [docs/ideas/agent-consumable-sdk.md](../ideas/agent-consumable-sdk.md)

The scope is docs-only: 21 contracts, the import map, doc-only roots, six
sample evals, and a `v0.1.1` release. Migrating the SDK into a real service is a
separate project.

## Confirmed Statement of Intent

- **Outcome:** Every one of the SDK's 21 public import paths has a
  self-contained `doc.go` contract — import path, when to use, minimal correct
  snippet, lifecycle, config defaults, common mistakes, test recipe — plus a
  root import map and doc-only roots for `nats`/`rest`, so an agent can choose
  and wire the package from `go doc` alone.
- **Proof:** Six sample evals (`db`, `rest/server`, `nats/consumer`,
  `observability`, `confloader`, `errs`) run by a separate agent from a
  self-contained kit — fixed fixture, prompt, allowed files, done command,
  grader checklist, correction log — with zero SDK-usage corrections. The
  real-repo migration is a separate project.
- **User:** Coding agents working in other repos — not human readers.
- **Why now:** The SDK's READMEs are rich but the toolchain surface is thin, and
  agents are the consumers; nothing meets them where they look.
- **Success:** Zero SDK-usage corrections on the six sample evals; all 21
  contracts exist; the import map has no dead roots; the eval kit is
  self-contained; `v0.1.1` is tagged and every module installs from the proxy.
- **Constraint:** Docs-only: no API changes, no new dependencies, no breaking
  normalization; no temporary adapter shims; manual evals; five weeks, including
  the release/tag.
- **Out of scope:** Migrating the SDK into a real service (catalog), additive
  API fixes (logged as findings), a harness, an OTLP/HTTP option,
  migration-runner work, a docs site, `llms.txt`, generated docs, rewriting the
  3,000-line READMEs, CI agent evals, new modules.

## Resolved Details (2026-11-08)

- **Runner:** another agent; the SDK prepares the kit and checklists.
- **Correction:** a human edit to SDK usage only; business-logic or formatting
  edits do not count.
- **Shims:** none; where the SDK is used, it is called directly.
- **Fixtures:** in-repo non-module templates under
  `docs/evals/agent-consumable-sdk/fixtures/`.
- **Automation:** one `setup.sh <sample>` plus prose; no harness and no CI agent
  runs.
- **Release:** `v0.1.1`, docs-only, inside the five weeks.
- **Order:** all 21 contracts exist before the sample evals start.
- **Observability:** unchanged; OTLP/gRPC stays the standard.
- **Migration runner:** untouched; not part of this project.
- **Five-field brief:** the five labels are an internal authoring checklist;
  public docs use plain headings (When to use, Import, Minimal example,
  Lifecycle, Common mistakes, Test recipe).

## What Changed from the 2026-11-03 Intent

- The catalog migration moved out; it needs the SDK implemented in a real
  project and is its own project.
- Additive API changes were removed; a missing capability is a finding, not
  work.
- The release is a docs-only `v0.1.1`.
- The budget went from eight weeks to five.
- The five open questions closed: gRPC stays, migrations are untouched,
  `v0.1.1`, in-repo fixtures, `setup.sh` plus prose.
