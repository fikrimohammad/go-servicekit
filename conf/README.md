# conf

Typed configuration with two lifetimes: a boot set loaded once from a file,
and a hot set watched in etcd. One loader binds one provider. Reads never touch
the network, boot is strict, and `State()` tells the truth.

The detailed contract is in [SPEC-conf.md](../docs/specs/conf/SPEC-conf.md).
Package-level docs are in `doc.go` for `conf`, `providers/file`, and
`providers/etcd`.

## Install

```bash
go get github.com/fikrimohammad/go-servicekit/conf
```

## File (boot config)

```go
type Boot struct {
    Checkout struct {
        TTL conf.Getter[time.Duration] `conf:"name=ttl"`
    } `conf:"folder=checkouts"`
}

loader, err := conf.New[Boot](ctx, file.New(file.Config{
    Path:   "disburseapp.prod.yaml",
    Format: conf.FormatYAML,
}))
if err != nil {
    return err
}
defer loader.Close()

ttl := loader.Data().Checkout.TTL.GetWithDefault(30 * time.Second)
```

`file.Config.Format` is required: `FormatYAML`, `FormatJSON`, or `FormatTOML`.
There is no extension detection. The file provider implements
`conf.Formatter`, so leaves must not carry a `format` tag.

## Etcd (hot config)

```go
type Hot struct {
    Checkout struct {
        TTL        conf.Getter[time.Duration] `conf:"name=ttl,format=scalar"`
        MaxRetries conf.Getter[int]           `conf:"name=max_retry,format=scalar"`
    } `conf:"folder=checkouts"`
    Toggle conf.Getter[bool] `conf:"name=toggle_switch,format=scalar"`
}

loader, err := conf.New[Hot](ctx, etcd.New(etcd.Config{
    Endpoint:  os.Getenv("ETCD_ENDPOINT"),
    Username:  os.Getenv("ETCD_USERNAME"),
    Password:  os.Getenv("ETCD_PASSWORD"),
    Namespace: "disburseapp",
}))
```

Every etcd leaf must declare `format=yaml|json|toml|scalar`; there is no
default. `scalar` reads one bare value into string, bool, int, uint, float,
`time.Duration`, or an `encoding.TextUnmarshaler`.

## Keys

Keys are the snake_case field path. A nested struct is a folder; a `Getter` is a
leaf.

```go
type Hot struct {
    Checkout struct {
        MaxRetries conf.Getter[int] `conf:"name=max_retry,format=scalar"`
    } `conf:"folder=checkouts"`
}
// key: checkouts/max_retry
```

`conf:"name=..."` renames a leaf segment, `conf:"folder=..."` renames a struct
segment, and `conf:"-"` skips a field or subtree. `conf:"-"` on a `Getter` is an
error: every Getter must be bound. A golden-key test catches a renamed field or
a dropped tag.

## Lifecycle

- `New` loads the provider once, requires every key to exist and decode, and
  binds the Getters. A missing or malformed key fails construction.
- An etcd provider then watches from the load revision. A bad update keeps the
  last good value and moves `State()` to `Degraded`; a later good value returns
  it to `Live`.
- A deleted key is absent: `Get` returns `conf.ErrNotSet`, `GetWithDefault`
  returns its argument.
- `Close` makes the loader unusable: `Get` returns `conf.ErrClosed`,
  `GetWithDefault` returns the default, `State()` is `Closed`. It is
  idempotent.
- Reads are per-key atomic. Two reads can span a reload; a set of reads is not
  a transaction.

## Adding a hot key

Create the key in etcd first, then deploy the code that reads it. Boot is
strict; the reverse order crash-loops the pod.

## Telemetry

- Metrics: `conf.reload.duration` (s), `conf.reload.count` (`{reload}`),
  `conf.last_success.timestamp` (s), with `conf.source` and, on failure,
  `error.type`.
- Spans: `conf.load`, `conf.reload` (internal), same attributes plus
  `conf.revision`.
- Logs include keys and revisions, never values.

## Tests

```bash
cd conf
go test -count=1 -race ./...
CONF_ETCD_ENDPOINT=127.0.0.1:2379 go test -count=1 -run Integration ./...
```
