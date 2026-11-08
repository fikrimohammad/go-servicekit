# Tasks: conf

Plan: [plan.md](plan.md). Spec:
[SPEC-conf](../docs/specs/conf/SPEC-conf.md). Idea:
[docs/ideas/conf.md](../docs/ideas/conf.md).

## Module checks

Run from `conf/` for every task that touches Go code:

- `gofmt -l .`
- `go vet ./...`
- `go test -count=1 ./...`
- `go test -count=1 -race ./...` at checkpoints
- `golangci-lint run ./...`
- `GOWORK=off go run ../.agents/skills/go-best-practices/scripts/gostyle.go ./...`
- `go mod tidy -diff`

Focused tests: `go test -count=1 ./providers/file`,
`go test -count=1 ./providers/etcd`, or `go test -count=1 -run <Pattern> .`.

Integration: `CONF_ETCD_ENDPOINT=127.0.0.1:2379 go test -count=1 -run Integration ./...`

**TDD rule.** Write the test with the task. Table-driven where there is more
than one case. Tests mirror source files.

One commit per task, subject `feat(conf): <summary> (T<n>)`.

---

## Task 1: Create the `conf` module skeleton and `go.work` entry

**Description:** Create `conf/go.mod` with module
`github.com/fikrimohammad/go-servicekit/conf`, Go 1.26.5, the sibling
`require`/`replace` shape, and the new dependencies (YAML, TOML, etcd, OTel).
Add a minimal `conf/doc.go` with `Package conf ...`; the full doc lands in T20.
Add `./conf` to `go.work`.

**Acceptance criteria:**
- [ ] `conf/go.mod` has the correct module path, `go 1.26.5`, and local
      `replace` directives for `observability`, `identity`, and `errs`.
- [ ] `conf/doc.go` contains a single package comment and no code.
- [ ] `go.work` lists `./conf` and no longer lists `./confloader` (deletion is
      T21; remove the entry there instead if the module must stay momentarily).
- [ ] `cd conf && go build ./...` passes with the skeleton.

**Verification:**
- [ ] `cd conf && gofmt -l . && go vet ./... && go build ./...`
- [ ] `git diff -- go.work conf/go.mod conf/go.sum`

**Dependencies:** None.

**Files likely touched:**
- `conf/go.mod`
- `conf/go.sum`
- `conf/doc.go`
- `go.work`

**Estimated scope:** XS

---

## Task 2: `Format` type, parsing, and scalar decoder

**Description:** Add `Format` with `FormatYAML`, `FormatJSON`, `FormatTOML`,
and `FormatScalar`; `Validate`; and `parseValue(raw, reflect.Type, Format)`.
YAML uses `yaml.Unmarshal`, JSON `encoding/json`, TOML `go-toml/v2`. Scalar
uses `encoding.TextUnmarshaler`, `time.ParseDuration`, `strconv`, and the
primitive kinds; an unsupported target is `ErrInvalidType`. Error text never
includes the raw value.

**Acceptance criteria:**
- [ ] Each format parses a scalar and a document into a concrete target.
- [ ] `FormatScalar` handles string, bool, int, uint, float, `time.Duration`,
      and a custom `TextUnmarshaler`; a struct target is `ErrInvalidType`.
- [ ] A custom `UnmarshalJSON`, `UnmarshalYAML`, and `UnmarshalTOML` each run
      under its own format.
- [ ] A JSON integer above 2^53 parses exactly into an `int64`.
- [ ] A TOML local date parses into its `go-toml/v2` type.
- [ ] Parse errors wrap `ErrParse` and contain no value text.

**Verification:**
- [ ] `cd conf && go test -count=1 -run 'TestFormat|TestParse' ./...`
- [ ] Module checks.

**Dependencies:** T1.

**Files likely touched:**
- `conf/format.go`
- `conf/format_test.go`
- `conf/errors.go`
- `conf/go.mod`

**Estimated scope:** M

---

## Task 3: `Getter[T]` and the `getterLeaf` binding marker

**Description:** Add `Getter[T]` with the opaque `Core func() (any, error)`
field, `Get`, `GetWithDefault`, `bind`, and the unexported `getterLeaf` marker
in `conf.go`.

**Acceptance criteria:**
- [ ] `Get` returns the typed value on success, `ErrUnbound` for a zero
      `Getter`, and the underlying error otherwise.
- [ ] `GetWithDefault` returns the default on any error and never panics.
- [ ] `bind` sets `Core` on `*Getter[T]`.
- [ ] Table tests cover bound, zero, error, and `GetWithDefault` cases.

**Verification:**
- [ ] `cd conf && go test -count=1 -run TestGetter ./...`
- [ ] Module checks.

**Dependencies:** T1.

**Files likely touched:**
- `conf/getter.go`
- `conf/getter_test.go`
- `conf/conf.go`
- `conf/errors.go`

**Estimated scope:** S

---

## Task 4: `Provider`, `Watcher`, `Formatter`, `Event`, and sentinels

**Description:** Define the public provider contract and the sentinel errors in
`errors.go`. `Event` carries Key, Value, Deleted, Reset, Revision, Heartbeat,
and Err, each documented.

**Acceptance criteria:**
- [ ] The interfaces match the spec exactly, with doc comments.
- [ ] `Event` fields are documented and have the specified types.
- [ ] All exported sentinels exist: `ErrNotSet`, `ErrClosed`, `ErrUnbound`,
      `ErrInvalidConfig`, `ErrInvalidTag`, `ErrInvalidType`, `ErrDuplicateKey`,
      `ErrMissingKey`, `ErrUnknownKey`, `ErrParse`, `ErrConnect`,
      `ErrInvalidFormat`, `ErrMissingFormat`.
- [ ] `go doc .` renders the contract without stray brackets.

**Verification:**
- [ ] `cd conf && go build ./... && GOWORK=off go doc -all .`
- [ ] Module checks.

**Dependencies:** T1.

**Files likely touched:**
- `conf/conf.go`
- `conf/provider.go`
- `conf/errors.go`

**Estimated scope:** S

---

## Task 5: Key derivation, tags, collisions, and the golden-key test

**Description:** Walk `T` with reflection. A leaf is a field whose pointer
implements `getterLeaf`; a plain struct is a folder. Derive snake_case path
segments, apply `folder=`/`name=`/`format=`, honor `conf:"-"`, reject malformed
tags, detect duplicate keys, and apply the format policy: a `Formatter`
provider forbids leaf `format` tags; a non-`Formatter` provider requires them.
Add `testdata/keys.golden`.

**Acceptance criteria:**
- [ ] `Checkout.MaxRetries` → `checkout/max_retries`; `DBHost` → `db_host`;
      `HTTPTimeout` → `http_timeout`.
- [ ] `folder=`, `name=`, and `format=` overrides produce the expected paths.
- [ ] `conf:"-"` skips a struct or non-`Getter`; on a `Getter` it is
      `ErrInvalidTag`.
- [ ] Unknown tags, duplicate tags, bad segment names, non-struct folders, and
      duplicate keys return the documented sentinels.
- [ ] The golden-key test fails on a field rename or a dropped tag.

**Verification:**
- [ ] `cd conf && go test -count=1 -run 'TestDerive|TestGolden' ./...`
- [ ] Module checks.

**Dependencies:** T2, T3, T4.

**Files likely touched:**
- `conf/key.go`
- `conf/key_test.go`
- `conf/testdata/keys.golden`

**Estimated scope:** M

---

## Checkpoint 1: Foundation

- [ ] `conf` builds; all foundation tests pass.
- [ ] The derivation and senders are locked by tests.
- [ ] Review with the owner before the three parallel streams start.

---

## Task 6: `conf.New` with a fake provider

**Description:** Implement `New[T](ctx, p Provider, opts ...Option)`: validate T
and the provider, derive keys, call `Load` once, join missing keys, parse,
bind, publish the first snapshot, and return the loader. Add `Data`, `State`,
and `Close`. Add a fake provider in `helpers_test.go`.

**Acceptance criteria:**
- [ ] A valid fake load produces a bound `Data()` with all leaves.
- [ ] Missing keys return `ErrMissingKey` with every key joined.
- [ ] A parse failure returns `ErrParse` and binds nothing.
- [ ] The file/remote format policy is enforced at `New`.
- [ ] `Close` is idempotent; `Get` after close is `ErrClosed`;
      `GetWithDefault` returns the default.
- [ ] A nil provider is `ErrInvalidConfig`.

**Verification:**
- [ ] `cd conf && go test -count=1 -run 'TestNew|TestClose' ./...`
- [ ] Module checks.

**Dependencies:** T2, T3, T4, T5.

**Files likely touched:**
- `conf/conf.go`
- `conf/conf_test.go`
- `conf/helpers_test.go`

**Estimated scope:** M

---

## Task 7: `State` and degradation tracking

**Description:** Add `State` (`Live | Degraded | Closed`) and the per-key /
provider degradation bookkeeping. A parse failure adds the key; a successful
update or delete removes it; a provider event error sets provider degradation;
a heartbeat, put, delete, or reset clears it.

**Acceptance criteria:**
- [ ] `State` is `Live` after `New`.
- [ ] A parse failure sets `Degraded`; a later valid value clears it.
- [ ] A provider `Error` sets `Degraded`; a heartbeat or reset clears it.
- [ ] Deleting a failed key clears its degradation and does not degrade.
- [ ] `Close` sets `Closed` and it is terminal.

**Verification:**
- [ ] `cd conf && go test -count=1 -run TestState ./...`
- [ ] Module checks.

**Dependencies:** T6.

**Files likely touched:**
- `conf/state.go`
- `conf/state_test.go`
- `conf/conf.go`

**Estimated scope:** S

---

## Task 8: Watch consumption

**Description:** Start the consumer when the provider implements `Watcher`.
Apply put, delete, reset, heartbeat, and error events to the copy-on-write
snapshot. A failed parse keeps the last good value and marks the key degraded.

**Acceptance criteria:**
- [ ] A put updates only its key; a delete removes only its key.
- [ ] A reset replaces the whole snapshot and clears degraded keys that parse
      or are absent.
- [ ] A heartbeat changes no value but clears provider degradation.
- [ ] A bad put keeps the old value and sets `Degraded`.
- [ ] Concurrent `Get` during reloads passes `-race`; no read is partial.
- [ ] `Close` cancels the watch and waits for the consumer.

**Verification:**
- [ ] `cd conf && go test -count=1 -race -run TestWatch ./...`
- [ ] Module checks.

**Dependencies:** T6, T7.

**Files likely touched:**
- `conf/conf.go`
- `conf/conf_test.go`
- `conf/helpers_test.go`

**Estimated scope:** M

---

## Task 9: Telemetry: spans, metrics, logs, defaults

**Description:** Implement the Telemetry Contract: `conf.load` and
`conf.reload` spans, `conf.reload.duration`, `conf.reload.count`,
`conf.last_success.timestamp`, and the documented log points. Add
`WithMetrics`, `WithTracer`, and `WithLogger`; nil clients fall back to the
package defaults per operation.

**Acceptance criteria:**
- [ ] Metric names, units, instrument kinds, and attributes match the spec
      exactly.
- [ ] Span names, kind, status, and attributes match the spec exactly.
- [ ] No key or value appears on a metric or span; keys may appear in logs.
- [ ] A nil client falls back to `metrics.Default()`/`tracer.Default()`/
      `logs.Default()`.
- [ ] The contract tests fail if a name, unit, or attribute changes.

**Verification:**
- [ ] `cd conf && go test -count=1 -run TestTelemetry ./...`
- [ ] Module checks.

**Dependencies:** T6.

**Files likely touched:**
- `conf/telemetry.go`
- `conf/telemetry_test.go`
- `conf/conf.go`

**Estimated scope:** M

---

## Checkpoint 2: Loader

- [ ] Loader tests pass with the fake provider, including `-race`.
- [ ] The telemetry contract is asserted.
- [ ] Review with the owner before the provider streams merge.

---

## Task 10: `file.Config` defaults and validation

**Description:** Add `providers/file.Config{Path, Format}`, `SetDefaults`, and
`Validate`. `Path` and `Format` are required; `FormatScalar` is rejected.

**Acceptance criteria:**
- [ ] Empty path → `ErrInvalidConfig`.
- [ ] Empty, scalar, or unknown format → `ErrInvalidFormat`.
- [ ] YAML, JSON, and TOML are accepted.
- [ ] `SetDefaults` leaves the required fields untouched.

**Verification:**
- [ ] `cd conf && go test -count=1 ./providers/file`
- [ ] Module checks.

**Dependencies:** T2.

**Files likely touched:**
- `providers/file/config.go`
- `providers/file/config_test.go`

**Estimated scope:** S

---

## Task 11: `file.Provider`, `Formatter`, strict document walk

**Description:** Add `New(cfg)`, `Source`, `Format`, `Load`, and `Close`. `Load`
reads the file once, parses it with `Format`, walks it by the requested key
paths, returns each leaf in the source format, errors on unknown nodes, and
leaves missing keys absent. A `Getter[SomeStruct]` leaf is one key.

**Acceptance criteria:**
- [ ] `Provider` satisfies `conf.Provider` and `conf.Formatter` at compile time.
- [ ] `Format()` returns `cfg.Format`; `Source()` returns `"file"`.
- [ ] A requested leaf returns the raw value in the source format.
- [ ] An unknown node returns `ErrUnknownKey`; a missing leaf is absent.
- [ ] A struct-valued leaf is one key, not a folder.
- [ ] `Close` is a no-op.

**Verification:**
- [ ] `cd conf && go test -count=1 ./providers/file`
- [ ] Module checks.

**Dependencies:** T4, T5, T10.

**Files likely touched:**
- `providers/file/file.go`
- `providers/file/file_test.go`

**Estimated scope:** M

---

## Task 12: File tests: formats, custom unmarshalers, edge types

**Description:** Complete the file provider tests with YAML, JSON, and TOML
documents; nested folders; missing and unknown keys; a custom `UnmarshalJSON`,
`UnmarshalYAML`, and `UnmarshalTOML`; a large JSON integer; a TOML local date;
and `Format`/`Source`.

**Acceptance criteria:**
- [ ] Each format loads a nested fixture and returns the expected keys.
- [ ] Each format-specific unmarshaler runs for its own format.
- [ ] A JSON integer above 2^53 round-trips exactly.
- [ ] A TOML local date round-trips.
- [ ] Missing, unknown, and malformed values return the documented sentinels.
- [ ] `FormatScalar` is rejected.

**Verification:**
- [ ] `cd conf && go test -count=1 ./providers/file`
- [ ] Module checks.

**Dependencies:** T11.

**Files likely touched:**
- `providers/file/file_test.go`
- `providers/file/testdata/*`

**Estimated scope:** M

---

## Task 13: File example and `doc.go`

**Description:** Add a runnable `example_test.go` with `// Output:` and a
`doc.go` in the house format.

**Acceptance criteria:**
- [ ] The example runs with `go test` and no external service.
- [ ] `doc.go` has `Package file ...` and the house headings.
- [ ] `go doc -all ./providers/file` renders.

**Verification:**
- [ ] `cd conf && go test -count=1 ./providers/file`
- [ ] `go doc -all ./providers/file`

**Dependencies:** T11.

**Files likely touched:**
- `providers/file/example_test.go`
- `providers/file/doc.go`

**Estimated scope:** S

---

## Checkpoint 3: File provider

- [ ] A real file loads end-to-end through `conf.New`.
- [ ] Missing, unknown, and unparseable keys fail boot.
- [ ] Custom unmarshalers, large ints, and TOML dates are covered.
- [ ] Review with the owner.

---

## Task 14: `etcd.Config` defaults and validation

**Description:** Add `providers/etcd.Config` with Endpoint, Username, Password,
Namespace, DialTimeout, and TLS; `SetDefaults` (5s dial); `Validate` (endpoint
host:port, namespace, auth both-or-neither).

**Acceptance criteria:**
- [ ] Missing endpoint or namespace returns `ErrInvalidConfig`.
- [ ] Username without password (or the reverse) returns `ErrInvalidConfig`.
- [ ] `DialTimeout` defaults to 5s.
- [ ] A nil TLS config is valid.

**Verification:**
- [ ] `cd conf && go test -count=1 ./providers/etcd`
- [ ] Module checks.

**Dependencies:** T1.

**Files likely touched:**
- `providers/etcd/config.go`
- `providers/etcd/config_test.go`

**Estimated scope:** S

---

## Task 15: `etcd.Provider.Load`

**Description:** Add `New(cfg)`, `Source`, `Load`, and `Close`. `Load` validates
the config, opens one `clientv3` client, performs one prefix `Get`, filters to
the requested keys, and returns the raw values plus the header revision.

**Acceptance criteria:**
- [ ] `Provider` satisfies `conf.Provider` at compile time; it does not
      implement `conf.Formatter`.
- [ ] `Source()` returns `"etcd"`.
- [ ] Keys are `<namespace>/<key>`; the prefix is `<namespace>/`.
- [ ] Only requested keys are returned; other keys are ignored.
- [ ] The revision is the header revision.
- [ ] `Close` is idempotent and safe after a failed `Load`.

**Verification:**
- [ ] `cd conf && go test -count=1 ./providers/etcd`
- [ ] Module checks.

**Dependencies:** T4, T14.

**Files likely touched:**
- `providers/etcd/etcd.go`
- `providers/etcd/etcd_test.go`

**Estimated scope:** M

---

## Task 16: `etcd.Provider.Watch`

**Description:** Implement the watch loop: start at `rev+1`, map PUT/DELETE to
events, emit a heartbeat on progress notify, emit `Err` on a disconnect and
retry with exponential backoff, and emit a `Reset` after a compaction resync.

**Acceptance criteria:**
- [ ] PUT and DELETE map to the documented events.
- [ ] A progress notification emits a heartbeat with the revision.
- [ ] A disconnect emits `Err` and reconnects; the channel stays open.
- [ ] A compaction triggers one prefix `Get` and emits `Reset`.
- [ ] `Close` cancels the loop and waits for it.
- [ ] The event mapping has unit tests.

**Verification:**
- [ ] `cd conf && go test -count=1 ./providers/etcd`
- [ ] Module checks.

**Dependencies:** T15.

**Files likely touched:**
- `providers/etcd/etcd.go`
- `providers/etcd/etcd_test.go`

**Estimated scope:** M

---

## Task 17: Etcd integration tests, example, and `doc.go`

**Description:** Add `integration_test.go` against a real etcd, a compile-only
example, and the package doc. The integration test skips without
`CONF_ETCD_ENDPOINT`.

**Acceptance criteria:**
- [ ] Boot loads every requested key; a missing key is absent.
- [ ] A `PUT` is visible in under 1s.
- [ ] A `DELETE` is reported.
- [ ] A compaction produces a `Reset` and the loader recovers.
- [ ] `Close` stops the watch.
- [ ] The test skips when `CONF_ETCD_ENDPOINT` is unset.
- [ ] `doc.go` follows the house format.

**Verification:**
- [ ] `CONF_ETCD_ENDPOINT=127.0.0.1:2379 go test -count=1 -run Integration ./providers/etcd`
- [ ] `GOWORK=off go doc -all ./providers/etcd`

**Dependencies:** T16.

**Files likely touched:**
- `providers/etcd/integration_test.go`
- `providers/etcd/example_test.go`
- `providers/etcd/doc.go`

**Estimated scope:** M

---

## Checkpoint 4: Etcd provider

- [ ] Real etcd load/watch passes; `PUT <1s`, `DELETE`, compaction resync.
- [ ] The provider does not implement `Formatter`.
- [ ] Review with the owner.

---

## Task 18: Loader end-to-end integration through etcd

**Description:** Add `conf/integration_test.go` in `package conf_test` that
builds a loader with the real etcd provider and asserts derivation, snapshot,
state transitions, and close.

**Acceptance criteria:**
- [ ] `conf.New` loads the fixture struct from etcd.
- [ ] A `PUT` is visible through `Get` in under 1s.
- [ ] A `DELETE` gives `ErrNotSet` / the default.
- [ ] A bad value keeps last good and sets `Degraded`.
- [ ] `Close` gives `ErrClosed` and releases the client.
- [ ] The test skips without `CONF_ETCD_ENDPOINT`.

**Verification:**
- [ ] `CONF_ETCD_ENDPOINT=127.0.0.1:2379 go test -count=1 -run Integration ./...`
- [ ] Module checks.

**Dependencies:** T8, T16.

**Files likely touched:**
- `conf/integration_test.go`

**Estimated scope:** M

---

## Task 19: Package examples and README

**Description:** Add `conf/example_test.go` (runnable file example, compile-only
etcd example) and `conf/README.md` (quickstart, key derivation, formats,
operations, telemetry).

**Acceptance criteria:**
- [ ] The file example runs with `// Output:`.
- [ ] The etcd example compiles without a service.
- [ ] The README documents both providers, `folder=`/`name=`/`format=`,
      strict boot, `State`, and `Close`.
- [ ] The README links to the spec and the provider docs.

**Verification:**
- [ ] `cd conf && go test -count=1 -run Example ./...`
- [ ] `GOWORK=off go doc -all .`

**Dependencies:** T13, T17.

**Files likely touched:**
- `conf/example_test.go`
- `conf/README.md`

**Estimated scope:** M

---

## Task 20: Package docs in the house format

**Description:** Finish `conf/doc.go`, `providers/file/doc.go`, and
`providers/etcd/doc.go` with the house sections: When to use, Import, Minimal
example, Key derivation/Formats, Lifecycle, Common mistakes, Test recipe.

**Acceptance criteria:**
- [ ] Each package has exactly one package comment, in `doc.go`.
- [ ] `go doc -all` renders every package without stray brackets.
- [ ] The `conf` doc explains `Formatter` vs `format=`, strict boot,
      per-key atomicity, and `Close`.
- [ ] No duplicate package comment remains in `conf.go`.

**Verification:**
- [ ] `cd conf && GOWORK=off go doc -all . && GOWORK=off go doc -all ./providers/file && GOWORK=off go doc -all ./providers/etcd`
- [ ] Module checks.

**Dependencies:** T19.

**Files likely touched:**
- `conf/doc.go`
- `providers/file/doc.go`
- `providers/etcd/doc.go`

**Estimated scope:** S

---

## Task 21: Delete `confloader`; update `go.work`, CI, root README

**Description:** Delete the `confloader/` module and its `go.work` entry, remove
it from `LAX_MODULES`, and replace its row in the root `README.md` with `conf`.

**Acceptance criteria:**
- [ ] `confloader/` is gone; `go.work` has no `./confloader`.
- [ ] `.github/workflows/ci.yml` has `LAX_MODULES: '[]'`.
- [ ] The root README lists `conf` instead of `confloader`.
- [ ] No module in `go.work` imports `confloader`; the frozen
      `docs/evals/agent-consumable-sdk` fixture is excluded.
- [ ] Each workspace module builds; `cd conf && go build ./...` passes.

**Verification:**
- [ ] `grep -R "confloader" --include='*.go' --exclude-dir=docs .` returns
      nothing.
- [ ] `cd conf && go build ./...`
- [ ] `git diff --stat`

**Dependencies:** T6, T11, T15.

**Files likely touched:**
- `go.work`
- `.github/workflows/ci.yml`
- `README.md`
- `confloader/` (deleted)

**Estimated scope:** S

---

## Task 22: Final verification and spec success-criteria audit

**Description:** Run every Command on the final tree, walk the spec's Success
Criteria, and record any deviation. No code changes unless a check fails.

**Acceptance criteria:**
- [ ] `gofmt`, `go vet`, `go test`, `go test -race`, `golangci-lint`, and
      gostyle report zero findings in `conf/`.
- [ ] `go mod tidy -diff` is clean.
- [ ] The etcd integration tests pass with `CONF_ETCD_ENDPOINT` and skip
      without it.
- [ ] Every Success Criterion in the spec is checked or explicitly waived.
- [ ] The task list is fully ticked.

**Verification:**
- [ ] Run the full Commands from the spec.
- [ ] `cd conf && GOWORK=off go doc -all . && GOWORK=off go doc -all ./providers/file && GOWORK=off go doc -all ./providers/etcd`
- [ ] Review the spec checklist with the owner.

**Dependencies:** T1–T21.

**Files likely touched:**
- `tasks/todo.md`

**Estimated scope:** S

---

## Checkpoint 5: Complete

- [ ] All tasks checked; full verification green.
- [ ] The spec's Success Criteria are audited.
- [ ] The owner approves the branch for review/merge.
