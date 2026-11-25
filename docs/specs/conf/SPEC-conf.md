# Spec: conf

Module id: `conf`. Status: draft for review.

## Assumptions I'm making

These are the choices the idea doc left open. Correct any of them before
approval:

1. **Package name `conf`.** Module `github.com/fikrimohammad/go-servicekit/conf`;
   provider implementations live under `conf/internal/providers`. Users choose
   a source through `conf.File` or `conf.Etcd`. If verbosity matters more than
   brevity, `configuration` is the alternative, but every call site gets
   longer.
2. **The provider contract is internal** (`internal/core.Provider`,
   `Watcher`, `Formatter`, and `Event`). `Source` is the public seam; there is
   no third-party provider ecosystem and no standalone provider API.
3. **`EtcdConfig` lives in the root `conf` package** and uses a single
   `Endpoint` plus optional `*tls.Config`. Multi-endpoint etcd is out of scope.
4. **Telemetry uses the `conf.` namespace:** `conf.reload.duration`,
   `conf.reload.count`, `conf.last_success.timestamp`; spans `conf.load` and
   `conf.reload`.
5. **The file provider walks the document tree** with the requested key list; a
   `Getter[SomeStruct]` leaf is one key, not a folder.
6. **The snapshot is copy-on-write.** Config sizes are tens of keys; each
   applied update clones the map and swaps a pointer.
7. **TOML is in scope** because you asked. It adds
   `github.com/pelletier/go-toml/v2`. If TOML is not actually required, drop it
   and keep YAML/JSON only.
8. **Formats are declared, never guessed.** `FileConfig.Format` is required and
   owns the whole document. Etcd values require a `format=` tag on every leaf;
   there is no default.
9. **The file provider preserves the source format and owns its parsing.** It
   implements the internal `Formatter` capability with `Parse(raw, type)`,
   decodes each leaf with `Format`, and returns raw source-format values. There
   is no normalization to YAML, so format-specific unmarshalers keep working,
   and TOML scalars/arrays decode through a synthetic single-key document.

## Objective

Give every internal service one configuration module that is correct under
change. `conf` replaces `confloader` with a narrow model: one loader binds one
provider, every leaf is a typed `Getter`, reads never touch the network, boot is
strict, and the loader tells the truth about whether hot config is live.

Two populations, never merged:

- **Boot config** comes from a file provider, is loaded once, and never changes.
- **Hot config** comes from an etcd provider, is loaded at boot, then follows
  the watch.

Consumers: internal service teams. Owner: developer experience.

Success: a hot change is visible in under a second (5s worst case); a missing or
unparseable key fails boot; a bad value never replaces a good one; a read is
always a complete value; `State()` and OTel expose the truth.

### User stories

- As a service team, I declare a struct of `Getter` fields, pick a provider, and
  read values with `Get` or `GetWithDefault`.
- As an operator, I create a hot key before deploying the code that reads it,
  and I can see from `State()` and metrics whether the loader is live or
  degraded.
- As an on-call engineer, a parse failure logs the key, keeps the last good
  value, and does not crash the service.
- As a developer, a type with `UnmarshalJSON`, `UnmarshalYAML`, or
  `UnmarshalTOML` keeps its custom behavior when its file is loaded.
- As a developer, renaming a field fails the golden-key test instead of
  silently reading the wrong key.
- As a maintainer, a new storage backend adds an internal provider and a
  `Source` constructor and changes nothing in the loader.

## Tech Stack

- Go 1.26.5 (from `go.work`); module
  `github.com/fikrimohammad/go-servicekit/conf`.
- `go.etcd.io/etcd/client/v3` v3.7.1 (`internal/providers/etcd` only).
- `gopkg.in/yaml.v3` v3.0.1 in `internal/core` (value decoding) and in
  `internal/providers/file` (document walking).
- `github.com/pelletier/go-toml/v2` v2.x for TOML values and files.
- `github.com/fikrimohammad/go-servicekit/observability` (`metrics.Client`,
  `tracer.Client`, `logs.Logger`), with local `replace` directives like the
  sibling modules. `identity` is a test-only dependency for the observability
  constructors.
- `go.opentelemetry.io/otel` v1.45.0: `trace`, `codes`,
  `semconv/v1.43.0` (for `error.type`), `sdk/trace/tracetest` in tests.
- Stdlib for everything else.

## Commands

Run from `conf/`:

```bash
gofmt -l .
go vet ./...
go test -count=1 -shuffle=on ./...
go test -count=1 -race -shuffle=on ./...
golangci-lint run ./...
GOWORK=off go run ../.github/scripts/sdkcheck/sdkcheck.go ./...
GOWORK=off go run ../.github/scripts/otelcheck/otelcheck.go ./...
go mod tidy -diff
GOWORK=off go doc -all .
GOWORK=off go doc -all ./internal/core
GOWORK=off go doc -all ./internal/providers/file
GOWORK=off go doc -all ./internal/providers/etcd
```

Integration:

```bash
CONF_ETCD_ENDPOINT=127.0.0.1:2379 go test -count=1 -run 'Integration' ./...
```

## Project Structure

```text
conf/
  go.mod                       module github.com/fikrimohammad/go-servicekit/conf
  README.md                    quickstart; links to doc.go
  doc.go                       package overview in the house format (see below)
  conf.go                      Option, New, Loader, Data, State, Close, getterLeaf
  conf_test.go
  config.go                    FileConfig, EtcdConfig, SetDefaults, Validate
  config_test.go
  source.go                    Source, File, Etcd, internal provider wiring
  source_test.go
  getter.go                    Getter[T], Get, GetWithDefault, bind
  getter_test.go
  clone.go                     copy-on-read for reference values
  clone_test.go
  key.go                       key derivation, tag parsing, snake_case,
                               collision detection, format policy
  key_test.go
  format.go                    public Format alias and constants
  state.go                     State, constants, String
  state_test.go
  errors.go                    public sentinel aliases
  telemetry.go                 metric descriptors, span and log helpers
  telemetry_test.go
  helpers_test.go              fixture structs, fake provider, recording clients
  example_test.go              runnable file example; compile-only etcd example
  integration_test.go          loader end-to-end through conf.Etcd
  testdata/                    sample YAML/JSON/TOML and golden key sets
  internal/
    core/
      core.go                  package comment; internal only
      provider.go              Provider, Watcher, Formatter, Event
      format.go                Format, constants, validation, parsing
      format_test.go
      errors.go                sentinels
    providers/
      file/
        doc.go
        config.go              internal Config
        file.go                Provider, New, Load, Parse, Source, Close
        file_test.go
        testdata/
      etcd/
        doc.go
        config.go              internal Config
        etcd.go                Provider, New, Load, Watch, Source, Close
        etcd_test.go
        integration_test.go    Load and Watch against a real etcd
```

Changes outside `conf/`:

- `go.work`: add `./conf`, remove `./confloader`.
- Delete `confloader/`.
- `.github/workflows/ci.yml`: remove `confloader` from `LAX_MODULES`; the list
  becomes empty.
- Root `README.md`: replace the `confloader` row with `conf`.
- `VERSION` and every sibling `require`: the new module ships at the current
  shared version; `release.yml` tags it `conf/vX.Y.Z`.

### `doc.go`

`conf/doc.go` carries the only package comment, in the same shape as
`observability/doc.go` and `identity/doc.go`:

- `Package conf ...` one-line summary.
- `# When to use` — one loader per store; boot vs hot; no secrets; no env layer.
- `# Import` — the conf import path.
- `# Minimal example` — `conf.File` and `conf.Etcd` loaders.
- `# Key derivation` — the snake_case path, `folder=`, `name=`, `format=`,
  `conf:"-"`, and the `checkouts/ttl` example.
- `# Formats` — the file source owns one document format via the internal
  `Formatter.Parse`; every etcd leaf declares its format; there is no default
  and no sniffing.
- `# Lifecycle` — `New` loads strictly, starts the watch, `State` reports
  health, `Close` invalidates the loader.
- `# Common mistakes` — `conf:"-"` on a `Getter`; a missing boot key; a missing
  etcd `format=` tag; expecting cross-key atomicity; logging values; treating
  env as a layer.
- `# Test recipe` — a temp file for the file provider; a fake provider for
  loader tests; `CONF_ETCD_ENDPOINT` for integration.

`internal/providers/file/doc.go` and `internal/providers/etcd/doc.go` are
internal package comments; they are not public API docs.

## Public API

```go
package conf

// Format identifies a value encoding.
type Format string

const (
	// FormatYAML parses YAML documents and plain scalars.
	FormatYAML Format = "yaml"
	// FormatJSON parses strict JSON.
	FormatJSON Format = "json"
	// FormatTOML parses TOML.
	FormatTOML Format = "toml"
	// FormatScalar parses one raw value into the target type without a
	// document parser. It is valid for etcd leaves only.
	FormatScalar Format = "scalar"
)

// Option configures a Loader at construction time.
type Option func(*options) error

// WithMetrics, WithTracer, and WithLogger inject telemetry clients. A nil
// client falls back to the package default, read per operation.
func WithMetrics(m metrics.Client) Option
func WithTracer(t tracer.Client) Option
func WithLogger(l logs.Logger) Option

// FileConfig configures a file source. Both fields are required.
type FileConfig struct {
	Path   string
	Format Format
}
func (c FileConfig) SetDefaults() FileConfig
func (c FileConfig) Validate() error

// EtcdConfig configures an etcd source. The value format is not here: every
// leaf declares its own format.
type EtcdConfig struct {
	Endpoint    string
	Username    string
	Password    string
	Namespace   string
	DialTimeout time.Duration
	TLS         *tls.Config
}
func (c EtcdConfig) SetDefaults() EtcdConfig
func (c EtcdConfig) Validate() error

// Source describes one storage backend. Create it with File or Etcd.
type Source struct{ /* unexported */ }

func File(cfg FileConfig) Source
func Etcd(cfg EtcdConfig) Source

// New builds a Loader for T, loads the source, and starts the watch when the
// source supports one. T must be a struct whose leaves are Getter fields.
// New takes ownership of the provider created by source. Canceling ctx stops
// the boot load but not the watch; Loader.Close stops the watch and closes the
// provider.
func New[T any](ctx context.Context, source Source, opts ...Option) (*Loader[T], error)

type Loader[T any] struct{ /* ... */ }

// Data returns the config struct with every Getter bound. Each call returns a
// copy whose Getters read the loader's current snapshot.
func (l *Loader[T]) Data() T

// State reports the loader's health and lifecycle state.
func (l *Loader[T]) State() State

// Close stops the watch and closes the provider. It is idempotent. After
// Close, reads return ErrClosed and State returns StateClosed.
func (l *Loader[T]) Close() error

// Getter is a typed accessor for one config key, created and bound only by a
// Loader. It has no exported fields, and its zero value is unbound.
type Getter[T any] struct{ /* unexported */ }

func (g Getter[T]) Get() (T, error)
func (g Getter[T]) GetWithDefault(def T) T

type State string

const (
	StateLive     State = "live"
	StateDegraded State = "degraded"
	StateClosed   State = "closed"
)
```

### Sources

The provider implementations live in `internal/providers` and are not public
API. Callers choose one through `Source`:

```go
loader, err := conf.New[Boot](ctx, conf.File(conf.FileConfig{
	Path:   "app.yaml",
	Format: conf.FormatYAML,
}))
loader, err := conf.New[Hot](ctx, conf.Etcd(conf.EtcdConfig{
	Endpoint:  "127.0.0.1:2379",
	Namespace: "disburseapp",
}))
```

`File` and `Etcd` validate their config immediately and store the error in the
returned `Source`; `New` reports it. A `Source` creates a fresh provider for
each `New`, so one `Source` can be used for more than one Loader.

Behaviour:

- `New` requires a usable `Source`; a zero `Source` or an invalid config is
  `ErrInvalidConfig` or the config's own sentinel.
- `New` creates the provider from `source` and takes ownership of it: it
  closes the provider on every error path, and `Loader.Close` closes it on
  success. Canceling the `ctx` passed to `New` stops the boot load but not the
  watch; only `Close` stops the watch.
- `New` derives the keys, calls `Load` once, checks that every derived key is
  present, parses every value, and binds the `Getter` fields.
- **Format policy:** the internal file provider implements the internal
  `Formatter` interface, so the loader calls `Parse` for every value and a leaf
  `format` tag is `ErrInvalidTag`. The internal etcd provider does not, so every
  leaf must carry `conf:"format=..."`; a missing tag is `ErrMissingFormat`; an
  unknown value is `ErrInvalidFormat`. There is no default either way.
- The internal etcd provider implements the internal `Watcher` interface; `New`
  calls `Watch` with the load revision. A watch that cannot start fails `New`;
  a transient disconnect after startup is reported as an internal `Event.Err`
  and does not fail the loader.
- `Data` is safe for concurrent use; the struct and its `Getter` fields are
  immutable after `New`. A Getter cannot be rebound from another package.
- A zero `Getter` returns `ErrUnbound` from `Get` and `def` from
  `GetWithDefault`. The loader never produces one.
- `Get` returns `ErrNotSet` for a deleted or absent key and `ErrClosed` after
  `Close`. It never returns a parse error: bad values never enter a snapshot.
  It deep-copies reference values, so callers may mutate the result.
- `GetWithDefault` returns `def` on `ErrNotSet`, `ErrClosed`, `ErrUnbound`, or
  a wrong stored type, and otherwise returns the same deep copy as `Get`.
- Exported errors: `ErrNotSet`, `ErrClosed`, `ErrUnbound`, `ErrInvalidConfig`,
  `ErrInvalidTag`, `ErrInvalidType`, `ErrDuplicateKey`, `ErrMissingKey`,
  `ErrUnknownKey`, `ErrParse`, `ErrConnect`, `ErrInvalidFormat`,
  `ErrMissingFormat`.

## Provider Contract

The loader and the internal providers meet at four types defined in
`internal/core`: `Provider`, `Watcher`, `Formatter`, and `Event`. They are not
public API; `Source` is the public seam.

- `Load(ctx, keys)` returns raw strings. A provider that implements
  `Formatter` owns their decoding; otherwise the loader decodes with the leaf's
  `format`.
- `keys` is the derived key set. The file provider uses it to decide where a
  document subtree ends and to detect unknown keys. The etcd provider uses it
  to filter the prefix result.
- `rev` is opaque. The file provider returns 0 and does not implement
  `Watcher`; the etcd provider returns the header revision and continues from
  it.
- `Watch` is called at most once, after `Load`, and must return promptly. The
  loader detaches its context from the boot context so the watch survives
  `New`'s context; `Close` cancels it.
- **Format policy:** `Formatter` is the parsing-ownership signal.
  - The provider implements `Formatter` → it owns decoding for every value;
    the loader calls `Parse` and a leaf `format` tag is `ErrInvalidTag`. The
    file provider is this case.
  - The provider does not implement `Formatter` → every leaf must carry
    `format=`, and the loader decodes with it. The etcd provider is this case.
  No `Source()` value influences behavior; `Source()` is telemetry only.
- `Formatter.Parse` must return a value assignable to the requested type; the
  loader converts it to that type before storing. It may return nil only when
  the target can be nil. The loader rejects a mismatched result at boot and on
  every update.
- `Event` carries exactly one kind of change: `Err`, `Heartbeat`, `Reset`,
  `Deleted`, or a put. If more than one is set, only the first in that order is
  applied and the others are ignored. A provider must not mutate an `Event` or
  its `Reset` map after sending it.
- Providers never log values. They may return errors; the loader classifies
  them for telemetry.
- The sentinels are the shared vocabulary: providers wrap connection failures
  with `ErrConnect`, read failures with `ErrInvalidConfig`, and the file
  provider returns `ErrUnknownKey` for extra nodes.

### `internal/providers/file`

- `New(cfg)` stores `Config{Path, Format}`. The public `FileConfig.Validate`
  checks both fields before the provider is created; `Load` reads the file once
  and parses it with `Format`.
- One file is one document. Decode it with `Format`, walk it by the requested
  key paths, and return each leaf as raw bytes in `Format`. JSON numbers are
  preserved as JSON, YAML nodes as YAML, TOML tables and date types as TOML.
  A second JSON value or YAML document is `ErrParse`. There is no
  normalization to YAML.
- A requested path that is absent is left out of the map; the loader reports
  `ErrMissingKey`.
- A node not consumed by a requested path is `ErrUnknownKey`, joined across all
  extras.
- `Format` accepts YAML, JSON, and TOML; `FormatScalar` is `ErrInvalidFormat`
  because a file is a document.
- `Parse(raw, t)` decodes with `Config.Format`. YAML and JSON parse directly;
  TOML tries the document as a table first, then decodes a synthetic
  single-key document for a scalar or array leaf. `Source()` returns `"file"`;
  `Close()` is a no-op. No watch.

### `internal/providers/etcd`

- `New(cfg)` stores the config; the public `EtcdConfig.Validate` checks it
  before the provider is created. `Load` opens one `clientv3` client and
  performs one prefix `Get`.
- Keys are `<namespace>/<key>`; the prefix is `<namespace>/`. Only requested
  keys are returned; other keys in the prefix are ignored.
- Values stay raw strings; the loader parses each with the leaf's mandatory
  `format` tag.
- `Watch` owns reconnect, backoff, progress notify, compaction, and resync:
  - A `PUT` becomes `Event{Key, Value, Revision}`.
  - A `DELETE` becomes `Event{Key, Deleted: true, Revision}`.
  - A progress notification becomes `Event{Heartbeat: true, Revision}`.
  - A disconnect becomes `Event{Err: err}`; the provider keeps retrying.
  - A compaction or reconnect resync becomes `Event{Reset: values, Revision}`.
- `Close` cancels the watch, waits for the goroutine, and closes the client.
  It is idempotent.

## Behaviour

### Value formats

There is no content sniffing. A raw string is ambiguous — `true` is a YAML
bool, a JSON string if quoted, and a TOML boolean — so the format is declared
once and applied deterministically.

| Format | Parser | Covers | Where |
| --- | --- | --- | --- |
| `FormatYAML` | `yaml.v3` | YAML documents and plain scalars (`30s`, `true`, `16`) | File documents; etcd leaves |
| `FormatJSON` | `encoding/json` | Strict JSON only | File documents; etcd leaves |
| `FormatTOML` | `go-toml/v2` | TOML documents | File documents; etcd leaves |
| `FormatScalar` | `strconv` + `encoding.TextUnmarshaler` | One bare value: string, bool, int, uint, float, `time.Duration`, or any `TextUnmarshaler` | etcd leaves only |

- **Files:** `FileConfig.Format` is required and applies to the whole
  document. There is no extension detection and no default. File leaves must
  not carry a `format` tag.
- **Etcd:** every leaf must carry `conf:"format=..."`; a missing tag is
  `ErrMissingFormat`. There is no default.
- **Custom unmarshalers:** the loader uses the declared parser, so the
  format-specific interface runs. `FormatJSON` honors `json.Unmarshaler`,
  `FormatYAML` honors `yaml.Unmarshaler`, `FormatTOML` honors
  `toml.Unmarshaler`, and all three honor `encoding.TextUnmarshaler`.
  `FormatScalar` uses `encoding.TextUnmarshaler`, `strconv`, and
  `time.ParseDuration`.
- `FormatScalar` reads the raw text directly. It special-cases `time.Duration`
  and handles `string`, `bool`, signed and unsigned integers, and floats with
  `strconv`. Any other target is `ErrInvalidType`.
- A YAML `null` into a `Getter[string]` becomes `""`; use `FormatScalar` or
  quote the value when exact text matters.

### Key derivation

`T` is walked with reflection. A field is a leaf when its pointer implements the
unexported `getterLeaf` marker (`*Getter[T]` does). For each exported field:

| Field kind | Meaning | Segment |
| --- | --- | --- |
| `Getter[T]` | leaf | `snake_case` of the field name, or `conf:"name=..."` |
| plain struct | folder | `snake_case` of the field name, or `conf:"folder=..."` |
| anything else | not config | left untouched |

- The key is the folder segments joined with `/`, then the leaf segment:
  `Hot.Checkout.MaxRetries` → `checkout/max_retries`.
- `conf:"folder=checkouts"` on the `Checkout` field and
  `conf:"name=max_retry"` on the leaf produce `checkouts/max_retry`.
- A leaf may add `format=yaml|json|toml|scalar`, e.g.
  `conf:"name=rules,format=json"`.
- `conf:"-"` skips a non-`Getter` field or a whole subtree. On a `Getter` it is
  `ErrInvalidTag`, because every `Getter` must be bound.
- Any other tag, an unknown tag key, or a duplicate tag key is `ErrInvalidTag`.
  `folder=` is only valid on a struct; `name=` and `format=` are only valid on
  a `Getter`.
- An override is a single snake_case segment matching
  `^[a-z][a-z0-9_]*$`; slashes, empty values, and uppercase are
  `ErrInvalidTag`.
- Two fields producing the same key is `ErrDuplicateKey`.
- Pointers, maps, slices, and other non-struct kinds in a folder position are
  `ErrInvalidType` unless tagged `conf:"-"`.
- Embedded structs use the embedded type name as their segment.
- Unexported fields are ignored, except a `Getter`: it can never be bound, so
  an unexported `Getter` and a `conf` tag on any unexported field are
  `ErrInvalidType`.

Snake case splits on case transitions and keeps initialisms as one word:
`Checkout` → `checkout`, `MaxRetries` → `max_retries`, `DBHost` → `db_host`,
`HTTPTimeout` → `http_timeout`.

### Load and reload

- `New` derives the keys, applies the format policy, calls `Load`, and requires
  every key to be present. Missing keys are `ErrMissingKey` with all missing
  keys joined.
- Every value is parsed with its effective format before the first snapshot is
  published. A parse failure is `ErrParse`; a bad value never enters a snapshot.
- A file load is complete at `New`. An etcd load starts a watch from the load
  revision.
- A `PUT` parses into the leaf's type. On success, clone the snapshot, replace
  the value, swap. On failure, keep the old value, mark that key degraded, log
  the key, and record telemetry. Other keys are unaffected.
- A `DELETE` removes the key, swaps, and clears its degraded mark.
- A `Reset` parses every requested key in the reset map; a key that is absent is
  removed; a key that fails keeps its old value and stays degraded. One swap
  publishes the whole reset.
- A `Heartbeat` proves the watch is alive; it clears a provider-level
  degradation but changes no values.

### Read semantics

- The loader holds `atomic.Pointer[snapshot]`, where a snapshot is an immutable
  `map[string]any` of parsed values.
- `Get` checks closed, loads the pointer, looks up the key, type-asserts, and
  deep-copies reference values. It performs no I/O.
- Per-key atomicity: a value is parsed before the snapshot is published, and a
  failed parse never replaces a good value.
- A value that references other data (a map, slice, pointer, array, or a struct
  with exported fields) is deep-copied on every read. Callers may mutate the
  result without changing the snapshot; types that hide mutable state behind
  unexported fields are copied shallowly.
- Cross-key atomicity is deliberately absent: two `Get` calls can see different
  snapshots. A set of reads is not a transaction.
- A deleted key is absent from the map. `Get` returns `ErrNotSet`;
  `GetWithDefault` returns the argument.

### State

| State | Meaning |
| --- | --- |
| `StateLive` | The last load or update succeeded; the snapshot is current as of the last event. |
| `StateDegraded` | A parse failure kept last-known-good for at least one key, or the provider's watch is disconnected. |
| `StateClosed` | `Close` ran. Terminal. |

Transitions: `Live → Degraded` on a parse failure or `Event.Err`;
`Degraded → Live` when the failed key is fixed or deleted and a heartbeat,
put, delete, or reset proves the watch is alive; any state → `Closed` on
`Close`. Deletion never degrades.

### Operations

- **Adding a key:** create it in etcd before deploying the code that reads it.
  Boot is strict; the reverse order crash-loops the pod. There is no
  optional-key tag.
- **Deleting a key:** the loader stays `Live`; the value becomes absent.
- **Closing:** `Close` sets `StateClosed`, cancels the watch, waits for the
  consumer goroutine, closes the provider, and is idempotent. Reads after close
  are misuse and return `ErrClosed`.

## Telemetry Contract

Public API for dashboards and alerts. `conf.source` is a custom attribute under
the package namespace; `error.type` comes from `semconv/v1.43.0` and is a
low-cardinality class.

### Metrics

| Name | Instrument | Unit | Attributes |
| --- | --- | --- | --- |
| `conf.reload.duration` | Histogram (`metrics.Record`) | `s` | `conf.source`, `error.type` (failure only) |
| `conf.reload.count` | Counter (`metrics.Add`) | `{reload}` | `conf.source`, `error.type` (failure only) |
| `conf.last_success.timestamp` | Gauge (`metrics.RecordGauge`) | `s` | `conf.source` |

- `conf.source` is the provider's `Source()`: `"file"` or `"etcd"`.
- `error.type` is `"config"`, `"key"`, `"parse"`, `"connect"`, `"watch"`, or
  `"_OTHER"`. `"key"` reports a missing or unknown key at boot.
- The boot load is recorded once. Successful batches, resets, and heartbeats
  update `conf.last_success.timestamp`. Batches, resets, parse failures, and
  watch disconnects record duration and count; heartbeats do not, and failures
  do not update the timestamp.
- No key, value, revision, endpoint, namespace, or format appears in a metric
  attribute.
- A nil metrics client falls back to `metrics.Default()`, read per operation.

### Spans

| Name | Kind | Attributes |
| --- | --- | --- |
| `conf.load` | Internal | `conf.source`, `conf.revision` (success), `error.type` (failure) |
| `conf.reload` | Internal | `conf.source`, `conf.revision` (success), `error.type` (failure) |

- Instrumentation scope `conf`.
- Status Error on failure; Unset on success.
- `conf.revision` is an int64 etcd revision, or absent for a file load.
- No key or value ever appears on a span. Keys may appear in logs only.
- A nil tracer client falls back to `tracer.Default()`, read per operation.

### Logs

Use `logs.Logger`, falling back to `logs.Default()`.

- Info: load succeeded (`conf.source`, `conf.revision`); watch recovered.
- Warn: reload parse failure (`conf.source`, `key`, `error.type`); watch
  disconnected (`conf.source`, `error.type`).
- Error: `Close` failed.
- Never log a value.

## Code Style

Follow the repository Go design rules in `AGENTS.md` and the sibling modules.

```go
// FileConfig configures a file source. Both fields are required; there is no
// extension detection and no default format.
type FileConfig struct {
	// Path is the config file path.
	Path string
	// Format is the document format: YAML, JSON, or TOML.
	Format Format
}

func (c FileConfig) Validate() error {
	var errs []error
	if c.Path == "" {
		errs = append(errs, ErrInvalidConfig)
	}
	switch c.Format {
	case FormatYAML, FormatJSON, FormatTOML:
	default:
		errs = append(errs, ErrInvalidFormat)
	}
	return errors.Join(errs...)
}

// Parse implements the internal core.Formatter. It decodes a raw file value
// with the document format and rejects per-leaf format tags.
func (p *Provider) Parse(raw string, t reflect.Type) (any, error) {
	// YAML and JSON decode directly. TOML tries the document as a table,
	// then falls back to a synthetic "v = ..." document for scalar and
	// array leaves.
}
```

```go
// EtcdConfig configures an etcd source. Build it from environment variables;
// the source never reads them itself. The value format is not here: every leaf
// declares its own format.
type EtcdConfig struct {
	// Endpoint is the etcd host:port. Required.
	Endpoint string
	// Username and Password are optional; set both or neither.
	Username string
	Password string
	// Namespace is the key prefix, without a trailing slash. Required.
	Namespace string
	// DialTimeout bounds the initial connection. Defaults to 5s.
	DialTimeout time.Duration
	// TLS, when non-nil, secures the connection.
	TLS *tls.Config
}
```

- Internal providers wrap the root sentinels, so their error strings start with
  the sentinel's `conf:` prefix; provider-specific context follows it.
- Package comments live only in each package's `doc.go`; no duplicate comment
  in the main file.
- Doc comments on every exported declaration and field.
- No mockey, no `//go:noinline`. `Provider`, `Watcher`, `Formatter`, and
  `Event` are internal to `internal/core`; `Source` is the only public seam.

## Testing Strategy

- Stdlib `testing`, table-driven where there is more than one case. No
  goconvey, no mockey.
- Tests mirror source files (`key.go` → `key_test.go`). Shared fixtures go in
  `helpers_test.go`: fixture structs, a fake `Provider`/`Watcher`, a recording
  `metrics.Client`, a `tracetest.SpanRecorder`, and a log buffer installed with
  `logs.SetDefault`.
- Required tests:
  - **Derivation:** every table row above, including initialisms, both
    overrides, `format=` on a leaf, `conf:"-"` on a struct, `conf:"-"` on a
    `Getter`, `ErrInvalidTag`, `ErrInvalidType`, and `ErrDuplicateKey`.
  - **Format policy:** a `Formatter` provider rejects a `format` tag; a
    non-`Formatter` provider requires one on every leaf (`ErrMissingFormat`);
    an unknown format is `ErrInvalidFormat`; `FormatScalar` handles string,
    bool, ints, floats, `time.Duration`, and `TextUnmarshaler`; an unsupported
    target is `ErrInvalidType`.
  - **Formatter validation:** a wrong result type and nil for a non-nilable
    target fail boot and watch updates; nil for a nilable target is accepted.
  - **Custom unmarshalers:** a type with `UnmarshalJSON` keeps its JSON
    behavior through a JSON file; the same for `UnmarshalYAML` and
    `UnmarshalTOML`; a `TextUnmarshaler` works under all three; a large JSON
    integer round-trips exactly; a TOML local date round-trips.
  - **File provider:** YAML, JSON, and TOML documents; a missing `Path` or
    `Format`; `FormatScalar` rejected; nested values, `time.Duration`,
    `net.IP`, a custom `TextUnmarshaler`, a `Getter` holding a struct, missing
    keys, unknown keys, a value that does not decode, a trailing JSON or YAML
    document, `Parse()`, and `Source()`.
  - **Etcd provider integration:** boot loads every key; a missing key fails; a
    `PUT` is visible in under 1s; a `DELETE` yields `ErrNotSet` / the default; a
    bad value keeps last good and sets `StateDegraded`; a compaction triggers a
    resync and returns to `StateLive`; `Close` stops the watch.
  - **Loader end-to-end:** the acceptance test uses the etcd provider and
    asserts the key derivation, the snapshot, the state transitions, and
    `Close`.
  - **Atomicity:** concurrent `Get` during reloads under `-race`; no read
    returns a partially decoded value; two keys may span snapshots.
  - **Copy-on-read:** maps, slices, pointers, arrays, nested exported fields,
    shared pointers, cycles, and nil/empty containers are isolated from the
    snapshot.
  - **Lifecycle:** `Close` is idempotent; `Get` → `ErrClosed`;
    `GetWithDefault` → default; `State` → `StateClosed`; a zero `Getter`
    returns `ErrUnbound`.
  - **Telemetry contract:** exact metric names, units, instrument kinds, and
    attribute sets; exact span names, kinds, statuses, and attributes; no key
    or value on a metric or span.
  - **Golden keys:** a checked-in key set per fixture struct under `testdata/`
    fails on a field rename or a dropped tag.
- `conf/example_test.go` has a runnable file example with `// Output:` and a
  compile-only etcd example. Internal packages are exempt.
- `integration_test.go` is the only file in a package using a real etcd; it
  calls `t.Skip` when `CONF_ETCD_ENDPOINT` is unset. CI sets the variable and
  fails if the tests skip.
- Coverage: every exported function and each table row above. No numeric
  coverage gate.

## Boundaries

- **Always:** run the Commands before each commit; update this spec first when
  a decision changes; keep tests in mirror files; keep emitted telemetry
  exactly as in the Telemetry Contract; keep `go test ./...` hermetic.
- **Ask first:** any new dependency; any change to a metric or span name, unit,
  instrument, or attribute; exporting API beyond Public API; changing CI or
  `release.yml`; adding `EtcdConfig` fields.
- **Never:** load, store, or log a config value; record an unbounded value as a
  metric attribute; use mockey or `//go:noinline`; keep `confloader`; reuse a
  version that proxy.golang.org already has.

## Success Criteria

- [ ] `conf/` exists with `doc.go` and internal file/etcd providers; every
      Command passes; `confloader/` is deleted; `go.work` and `LAX_MODULES` are
      updated; the strict lint, sdkcheck, and otelcheck checks report zero findings.
- [ ] `FileConfig` requires both `Path` and `Format`; YAML, JSON, and TOML
      documents load, including `time.Duration`, `net.IP`, and
      `TextUnmarshaler` values; `FormatScalar` is rejected for files.
- [ ] `Source`, `File`, and `Etcd` are the only public provider seam; provider
      implementations are under `internal/providers` and cannot be imported by
      users.
- [ ] The file provider implements the internal `Formatter.Parse` and decodes
      raw values with `Format`; a `format` tag on a file-loaded leaf is
      `ErrInvalidTag`; a TOML scalar leaf and a TOML table leaf both load.
- [ ] Every etcd leaf requires a `format=` tag; a missing tag is
      `ErrMissingFormat`; an unknown format is `ErrInvalidFormat`.
- [ ] `FormatScalar` parses string, bool, ints, uints, floats,
      `time.Duration`, and `TextUnmarshaler` targets; an unsupported target is
      `ErrInvalidType`.
- [ ] A type with `UnmarshalJSON`, `UnmarshalYAML`, or `UnmarshalTOML` keeps its
      custom behavior through the matching file format; a large JSON integer
      round-trips exactly; a TOML local date round-trips.
- [ ] Missing keys, unknown file keys, malformed tags, `conf:"-"` on a
      `Getter`, and duplicate keys fail `conf.New` with the documented
      sentinels.
- [ ] Etcd boot is strict; unknown keys in the prefix are ignored; a `PUT` is
      visible in under 1s; a `DELETE` yields `ErrNotSet` / the default; a bad
      value keeps last good and sets `StateDegraded`; a compaction resync
      recovers to `StateLive`.
- [ ] Concurrent reads during reloads pass `-race`, no read is partial, and a
      returned map, slice, or pointer can be mutated without touching the
      snapshot.
- [ ] `Close` is idempotent; `Get` → `ErrClosed`; `GetWithDefault` → default;
      `State` → `StateClosed`; a zero `Getter` returns `ErrUnbound`.
- [ ] Telemetry contract tests assert every metric and span name, unit,
      instrument kind, and attribute exactly.
- [ ] The golden-key test fails on a field rename or a dropped tag.
- [ ] `go test ./...` is hermetic; the etcd integration tests skip without
      `CONF_ETCD_ENDPOINT`.
- [ ] `conf/doc.go` follows the house format; internal provider package docs
      exist, and `conf/README.md` is a quickstart.

## Open Questions

None. The items in "Assumptions I'm making" are the only decisions not fixed by
the idea doc; confirm or change them before approval.
