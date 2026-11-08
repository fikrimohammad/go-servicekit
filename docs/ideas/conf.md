# conf: boot and hot configuration without lying

Direction record. The detailed contract lives in
[docs/specs/conf/SPEC-conf.md](../specs/conf/SPEC-conf.md).

## Problem Statement

How might we give a Go service typed access to two kinds of configuration — a
boot set loaded once from a file, and a hot set read from etcd without polling
— while guaranteeing that every read is a complete value, that boot fails
loudly, that custom unmarshalers keep working, and that the health of the hot
set is always visible?

## Recommended Direction

A new strict module, `conf`, replacing `confloader`. One loader binds one
provider, and providers live next to the loader:

```text
conf/
  conf.go             New, Loader, Getter, State, errors, telemetry
  providers/file/     strict document loader, owns one format
  providers/etcd/     boot Get, watch, resync, per-key formats
```

```go
boot, err := conf.New[Boot](ctx, file.New(file.Config{
	Path:   "disburseapp.prod.yaml",
	Format: conf.FormatYAML,
}))
hot, err := conf.New[Hot](ctx, etcd.New(etcd.Config{
	Endpoint:  os.Getenv("ETCD_ENDPOINT"),
	Namespace: "disburseapp",
}))
```

The config type is always a tree of `Getter[T]` leaves, and nested plain
structs are folders. Keys are the snake_case field path, so
`Checkout.MaxRetries` is `checkout/max_retries`. Any segment can be renamed:
`conf:"folder=..."` on a nested struct renames its folder segment, and
`conf:"name=..."` on a leaf renames its config name. `conf:"-"` excludes a
non-`Getter` field; on a `Getter` it is an initialization error, because the
loader guarantees every `Getter` is bound.

```go
type Hot struct {
	Checkout struct {
		Timeout    conf.Getter[time.Duration] `conf:"name=ttl,format=scalar"`
		MaxRetries conf.Getter[int]           `conf:"name=max_retry,format=scalar"`
	} `conf:"folder=checkouts"`
	KillSwitch conf.Getter[bool] `conf:"name=toggle_switch,format=scalar"`
}
```

That produces `checkouts/ttl`, `checkouts/max_retry`, and `toggle_switch`.
There is no default-folder fallback and no cross-source merge. One value lives
in one store, and one loader sees one store.

### Formats are declared, never sniffed

A raw string is ambiguous: `true` is a YAML bool, a JSON string if quoted, and
a TOML boolean. The format is therefore explicit.

- **File:** `file.Config.Format` is required and owns the whole document. The
  file provider implements the optional `Formatter` capability
  (`Format() conf.Format`), returns every leaf in that format, and the loader
  parses with it. A `format` tag on a file leaf is an error. There is no
  extension detection and no default.
- **Etcd:** the provider does not implement `Formatter`; every leaf must carry
  `format=yaml|json|toml|scalar`. A missing tag is `ErrMissingFormat`, an
  unknown one is `ErrInvalidFormat`. There is no fallback.
- **Why `Formatter` exists:** the file config tells the provider how to read
  the document, but only the loader knows the target `T`, so only the loader
  can call `json.Unmarshal`, `yaml.Unmarshal`, or `toml.Unmarshal` into `T`.
  `Formatter` carries the format across that boundary. Without it, the loader
  would normalize to YAML and silently skip `UnmarshalJSON`, `UnmarshalTOML`,
  and large-integer fidelity.
- **Custom unmarshalers:** the declared parser runs, so `json.Unmarshaler`,
  `yaml.Unmarshaler`, `toml.Unmarshaler`, and `encoding.TextUnmarshaler` all
  behave as their authors intended. `FormatScalar` uses `strconv`,
  `time.ParseDuration`, and `TextUnmarshaler` for bare etcd values.

Reads are pure: `Get() (T, error)` and `GetWithDefault(def T) T`, with no `ctx`
and no I/O. The loader holds an immutable snapshot behind an atomic pointer. A
file loader produces one snapshot at `New` and never changes. An etcd loader
produces one at `New`, then applies watch events per key.

Atomicity is per key. A `Get` is a single load of that key's current value: it
never returns a value parsed from a partial update, and a failed parse never
replaces the last good value. Reads of different keys are not a transaction;
two `Get`s can observe different revisions. This is the highest-priority
correctness property in the design.

Boot is strict; runtime is lenient. `New` loads the whole store and requires
every derived key to exist and parse. A missing key, an unknown field in the
file, a tag error, or an unparseable value fails construction. After that, an
unparseable update keeps that key's last good value and moves the loader to
`Degraded`; every other key in the update still applies. Deleting a key leaves
it absent: `GetWithDefault` returns the call-site default, `Get` returns
`ErrNotSet`, until the key comes back.

The hot path is etcd `Watch`, resumed from the boot `Get` revision.
`WithProgressNotify` is the liveness check, not a value poll; a compaction
triggers one prefix `Get` to resync, and reconnects back off. The target is
under one second, five seconds worst case.

Health is part of the API: `State()` is `Live | Degraded | Closed`. `Degraded`
means a key's last parse failed or the watch is disconnected and the loader is
serving last-known-good; a successful update, reset, or heartbeat returns
`Live`. `Closed` is terminal. The module emits `conf.load` / `conf.reload`
spans, a reload counter, a reload duration histogram, and a last-success
timestamp. `source` is the only metric attribute; keys and values never
appear. File mode emits the load span and nothing else.

Bootstrap is out of the loader. The caller reads endpoint, credentials, and app
environment from env — injected by the platform from the secret store — and
builds the provider config. `conf` never talks to a secret store. Validation is
the consumer's job: the library guarantees a value parses into `T`, not that it
makes sense.

### Operations

- **Adding a key:** create it in etcd first, then deploy the code that reads
  it. Boot is strict, so the reverse order crash-loops the pod. There is no
  optional-key tag.
- **Deleting a key:** the loader stays `Live`; the key becomes absent. `Get`
  returns `ErrNotSet`, `GetWithDefault` returns the call-site default.
- **Closing:** `Close` makes the loader unusable. `Get` returns `ErrClosed`,
  `GetWithDefault` returns the default, `State()` is `Closed`, and `Close` is
  idempotent. Reads after close are misuse, not a shutdown path.

### API sketch

```go
type Hot struct {
	Checkout struct {
		Timeout    conf.Getter[time.Duration] `conf:"name=ttl,format=scalar"`
		MaxRetries conf.Getter[int]           `conf:"name=max_retry,format=scalar"`
	} `conf:"folder=checkouts"`
	KillSwitch conf.Getter[bool] `conf:"name=toggle_switch,format=scalar"`
}

loader, err := conf.New[Hot](ctx, etcd.New(etcd.Config{
	Endpoint:  os.Getenv("ETCD_ENDPOINT"),
	Username:  os.Getenv("ETCD_USERNAME"),
	Password:  os.Getenv("ETCD_PASSWORD"),
	Namespace: "disburseapp",
}))
if err != nil {
	return err // boot is strict
}
defer loader.Close()

hot := loader.Data()
timeout := hot.Checkout.Timeout.GetWithDefault(30 * time.Second)
enabled, err := hot.KillSwitch.Get() // ErrNotSet if deleted, ErrClosed after Close
```

Derived keys: `checkouts/ttl`, `checkouts/max_retry`, `toggle_switch`.

## Decisions

| Decision | Why |
| --- | --- |
| One loader, one provider | A key lives in one place; a loader that sees two stores re-invents merge and precedence. |
| Providers under `providers/` | File and etcd storage code stays out of the loader; a new backend adds a package and changes nothing in the core. |
| Every leaf a `Getter[T]` | File and hot config share one access shape: `Get` / `GetWithDefault`. |
| Derived snake_case path, per-segment rename | Nested structs carry the grouping, so field names stay short; `folder=` and `name=` rename one segment each, and `conf:"-"` is the only other tag. |
| `Formatter` for files, `format=` for etcd | A file is one document with one format; an etcd key is an independent value with its own format. |
| No format sniffing | A raw string is ambiguous; declared formats give deterministic errors and preserve custom unmarshalers. |
| Per-key atomicity | A read is one complete value; cross-key reads are not a transaction. This keeps the `Getter` shape and avoids whole-struct swapping. |
| Strict boot, lenient runtime | A misconfigured deploy must not start; an operator typo must not take down a running one. |
| Create the key before the code | Strict boot only works if the key exists first; an optional-key tag would let a typo pass at boot. |
| Parse failure keeps last good | A bad value is a mistake, not an intentional revert. |
| Deletion falls to the call-site default | `GetWithDefault` makes the fallback local and explicit. |
| No `ctx` on reads | Reads are a pointer load; a context would be a lie. |
| No `Initializing` state | `New` is synchronous and strict; no caller can observe construction. |
| `Close` invalidates the loader | A closed loader is misuse; returning `ErrClosed` is honest, while serving a frozen snapshot hides the bug. |
| Consumer-side validation | The library guarantees parsing, not policy; semantic rules would have to run against call-site defaults. |
| No callbacks | Every hot value is read at use, and anything needing reinitiation is a deploy. OTel covers the audit need. |
| File is read once | Env selects the file; a change arrives with a new process. |
| etcd Watch, not polling | The sub-second target is the point; the old 30s ticker is the failure being replaced. |

## Key Assumptions to Validate

- [x] Hot reload is real: feature toggles and per-request timeouts, not pool
      sizes or consumer counts.
- [x] Boot config is read once; a file change arrives with a deploy.
- [x] Values are YAML, JSON, TOML, or bare scalars; the format is declared, not
      sniffed.
- [x] Latency target is under 1s ideal, 5s maximum.
- [x] Secrets are out of scope; env carries only bootstrap and environment.
- [x] etcd Watch survives the target clusters: long-lived gRPC through their
      proxies and network policy. Confirmed on the production network path.
- [x] The keyspace matches derived paths except where a folder or a leaf name
      differs. Confirmed: several existing fields already need a rename.
- [x] Existing prod files map to the struct through the same per-segment
      overrides and the declared file format.
- [x] Adding a hot key by creating it in etcd before deploying is an
      acceptable SOP.
- [x] A type with `UnmarshalJSON`, `UnmarshalYAML`, or `UnmarshalTOML` must
      keep its custom behavior; the file provider preserves the source format.

## MVP Scope

1. `conf`: `New[T](ctx, provider, opts...)`, `Getter[T]`, `Get`,
   `GetWithDefault`, `Data`, `State`, `Close`, `Provider`, `Watcher`,
   `Formatter`.
2. `providers/file`: `Config{Path, Format}`, `Formatter`, raw source values,
   strict missing/unknown keys, no watch.
3. `providers/etcd`: boot prefix `Get`, watch from revision, progress notify,
   compaction resync, reconnect backoff, per-key last-good, per-key formats.
4. Key derivation with per-segment `conf:"folder=..."` (struct),
   `conf:"name=..."` and `conf:"format=..."` (leaf), `conf:"-"` opt-out for
   non-`Getter` fields, an init error for `conf:"-"` on a `Getter`, and
   collision detection at `New`. Each override is a single snake_case segment.
5. Formats: `FormatYAML`, `FormatJSON`, `FormatTOML`, `FormatScalar`; parsing
   honors each format's custom unmarshaler.
6. Health and telemetry: `State` (`Live | Degraded | Closed`), last sync,
   version, `conf.load` / `conf.reload` spans, counter, histogram.
7. Errors: `ErrNotSet`, `ErrClosed`, `ErrUnbound`, `ErrInvalidConfig`,
   `ErrInvalidTag`, `ErrInvalidType`, `ErrDuplicateKey`, `ErrMissingKey`,
   `ErrUnknownKey`, `ErrParse`, `ErrConnect`, `ErrInvalidFormat`,
   `ErrMissingFormat`.
8. Tests: derivation, per-segment overrides, `conf:"-"` on a `Getter`, strict
   boot, per-key atomicity under `-race`, parse-failure keeps last good,
   deletion → `ErrNotSet` / default, `Close` → `ErrClosed`, watch resume,
   compaction, custom unmarshalers, large JSON ints, TOML dates, etcd
   integration skipped without `CONF_ETCD_ENDPOINT`.
9. Golden-key test: a checked-in key set per config type so a field rename or
   a dropped tag fails CI.
10. New module at `go-servicekit/conf`, added to `go.work`; delete
    `confloader` and remove it from `LAX_MODULES`.

## Not Doing (and Why)

- **Secrets storage** — no provider notifies reliably; credentials are
  bootstrap's job.
- **Disk cache** — in-memory only; cold start is one fetch.
- **Polling** — progress notify and a resync on compaction replace it.
- **File watching** — the file is boot-only; `fsnotify` and the ConfigMap
  symlink problem leave the design.
- **Cross-source merge / precedence** — a key lives in exactly one store.
- **Multiple sources per loader** — one loader, one store; compose in the app.
- **Automatic format detection** — ambiguous for raw values; formats are
  declared in `file.Config` or the leaf tag.
- **A third-party provider ecosystem** — the provider interfaces are public
  because the providers are subpackages, not a compatibility promise.
- **Absolute or multi-segment overrides** — per-segment renames compose to any
  path; an absolute override would let a key contradict the struct shape.
- **Optional-key tags** — create the key before deploying; an optional tag
  would let a misspelled required key pass boot.
- **Whole-struct snapshots / cross-key transactions** — per-key atomicity is
  the guarantee; whole-struct swapping fights the `Getter` field shape.
- **Serving reads after `Close`** — a closed loader is unusable; `ErrClosed`
  is the contract.
- **An `Initializing` state** — `New` is synchronous; the state is
  unreachable.
- **An HTTP webhook server or `OnChange`** — no callback use case survived;
  `State()` and OTel cover operations.
- **Codegen / schema files** — negative ROI at this size.
- **Plain-field accessors** — every leaf is a `Getter`.
- **Library-side semantic validation** — consumer's job.
- **Env as a config source** — env selects the file and carries bootstrap; it
  is not a layer.
- **Keeping `confloader` or a shim** — no in-repo importers, pre-1.0, delete
  it.

## Open Questions

None. Resolved: etcd Watch is confirmed on the production network path;
mismatched folder and config names are handled by per-segment `folder=` and
`name=` overrides; strict file decoding stays, with any key that has no field
treated as rollout cleanup; a new hot key is created before its code deploys;
formats are declared, never sniffed; the file provider implements `Formatter`
so custom unmarshalers keep working.
