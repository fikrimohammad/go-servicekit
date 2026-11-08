# Implementation Plan: conf

Spec: [SPEC-conf](../docs/specs/conf/SPEC-conf.md). Idea:
[docs/ideas/conf.md](../docs/ideas/conf.md). Tasks: [todo.md](todo.md).

## Overview

Build the `conf` module: one loader binds one provider, every leaf is a typed
`Getter`, reads are I/O-free, boot is strict, hot config follows an etcd watch,
and the file provider preserves its source format. Replace `confloader` and
empty `LAX_MODULES`.

This plan replaces the completed agent-consumable-sdk plan (all tasks done).

**Status 2026-11-08:** T1–T22 are implemented on `feat/conf-module`. Unit,
race, lint, gostyle, and doc-contract checks pass. The etcd integration tests
pass against a local etcd 3.5.17 (root loader, watch, compaction resync).

## Architecture Decisions

- **One loader, one provider.** `conf.New[T](ctx, p, opts...)`; `providers/file`
  and `providers/etcd` are separate packages. No merge or precedence.
- **Provider capabilities.** `Provider` (Source, Load, Close) is required;
  `Watcher` (Watch) and `Formatter` (Format) are optional type assertions.
- **Format ownership.** `Formatter` means the provider owns one format for every
  value (file). Its absence means every leaf must carry `format=` (etcd). No
  default and no sniffing.
- **Reads are pure.** `Getter[T].Get` / `GetWithDefault` load an immutable
  copy-on-write snapshot; per-key atomicity, no cross-key transaction.
- **Strict boot, lenient runtime.** `New` fails on missing/unparseable keys;
  a bad watch value keeps last-known-good and sets `StateDegraded`.
- **No callbacks.** `State()` and OTel carry health; values are read at use.
- **Delete `confloader`.** No shim; the SDK is pre-1.0 and has no in-repo
  importers.

## Task List

### Phase 1: Foundation

- [x] T1: Create the `conf` module skeleton and `go.work` entry
- [x] T2: `Format` type, parsing, and scalar decoder
- [x] T3: `Getter[T]` and the `getterLeaf` binding marker
- [x] T4: `Provider`, `Watcher`, `Formatter`, `Event`, and sentinels
- [x] T5: Key derivation, tags, collisions, and the golden-key test

**Checkpoint 1:** `conf` builds; format/getter/key tests pass; derivation is
locked by `testdata/keys.golden`.

### Phase 2: Loader

- [x] T6: `conf.New` with a fake provider: load, strict boot, bind, Data, Close
- [x] T7: `State` and degradation tracking
- [x] T8: Watch consumption: put, delete, reset, heartbeat, error
- [x] T9: Telemetry: spans, metrics, logs, defaults

**Checkpoint 2:** loader end-to-end with a fake provider; `-race` passes;
telemetry contract tests pass.

### Phase 3: File provider

- [x] T10: `file.Config` defaults and validation
- [x] T11: `file.Provider`, `Formatter`, strict document walk, raw values
- [x] T12: File tests: YAML/JSON/TOML, custom unmarshalers, large ints, TOML dates
- [x] T13: File example and `doc.go`

**Checkpoint 3:** a real file loads end-to-end through `conf.New`; missing,
unknown, and unparseable keys fail boot.

### Phase 4: Etcd provider

- [x] T14: `etcd.Config` defaults and validation
- [x] T15: `etcd.Provider.Load`: client, prefix `Get`, key filter, revision
- [x] T16: `etcd.Provider.Watch`: events, heartbeat, backoff, compaction resync
- [x] T17: Etcd integration tests, example, and `doc.go`

**Checkpoint 4:** a real etcd loads and watches; a `PUT` is visible in under 1s;
a `DELETE` is absent; a compaction resync recovers.

### Phase 5: Acceptance and migration

- [x] T18: Loader end-to-end integration through the etcd provider
- [x] T19: Package examples and README
- [x] T20: `doc.go` for `conf` and both providers in the house format
- [x] T21: Delete `confloader`; update `go.work`, `LAX_MODULES`, root README
- [x] T22: Final verification and spec success-criteria audit

**Checkpoint 5:** every Command passes; the spec's Success Criteria are checked;
the module is ready for review.

## Risks and Mitigations

| Risk | Impact | Mitigation |
| --- | --- | --- |
| Etcd compaction/reconnect is the hardest part | High | T16 is isolated, integration-tested, and lands before acceptance; force compaction in T17 |
| `Getter` reflection binding | Medium | T3/T5 test the marker, binding, and zero value before the loader exists |
| Format-specific unmarshalers regress | High | T12 asserts `UnmarshalJSON`/`UnmarshalYAML`/`UnmarshalTOML`, a large JSON int, and a TOML date |
| Telemetry names drift from the contract | Medium | T9 asserts exact names, units, kinds, and attributes; T22 re-runs it |
| Strict lint on a new module | Medium | T1 uses the sibling `go.mod` shape; run `golangci-lint` and gostyle from T1 |
| Deleting `confloader` breaks an unseen consumer | Low | No in-repo importers; pre-1.0; release notes call it out |

## Parallelization

After Checkpoint 1, three streams are independent because the loader uses a
fake provider:

- **Stream A:** T6 → T7 → T8 → T9 (loader).
- **Stream B:** T10 → T11 → T12 → T13 (file provider).
- **Stream C:** T14 → T15 → T16 → T17 (etcd provider).

T18 needs T8 and T16. T21 needs T6, T11, and T15. T19/T20 need both providers.
T22 is last.

## Open Questions

None from the spec. Release tagging is manual and out of scope for this plan;
the module ships at the shared `VERSION` when the owner runs `release.yml`.
