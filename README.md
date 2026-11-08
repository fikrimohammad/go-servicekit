# go-servicekit

Personal go-to SDK for building Go applications.

## Import map

The SDK is 11 Go modules. These are the public import paths, by task:

| I need | Import | Notes |
| --- | --- | --- |
| An error with a code, cause, metadata, and stack | `github.com/fikrimohammad/go-servicekit/errs` | Structured errors; slog attribute |
| Concurrency with a limit and panic safety | `github.com/fikrimohammad/go-servicekit/errgroup` | Wraps `golang.org/x/sync/errgroup` |
| A standard service name, version, and env | `github.com/fikrimohammad/go-servicekit/identity` | Used by observability |
| To wire logs, metrics, and traces | `github.com/fikrimohammad/go-servicekit/observability` | Wiring guide; no API |
| To normalize attribute keys | `github.com/fikrimohammad/go-servicekit/observability/attributes` | Key normalization |
| A logger with trace context | `github.com/fikrimohammad/go-servicekit/observability/logs` | slog handler; trace_id |
| Counters, histograms, gauges | `github.com/fikrimohammad/go-servicekit/observability/metrics` | Metrics client |
| Spans and a tracer client | `github.com/fikrimohammad/go-servicekit/observability/tracer` | Tracer client |
| Layered config and secrets | `github.com/fikrimohammad/go-servicekit/confloader` | File + dynamic + secrets |
| To implement a config backend | `github.com/fikrimohammad/go-servicekit/confloader/client` | Backend contract |
| etcd-backed dynamic config | `github.com/fikrimohammad/go-servicekit/confloader/client/etcd` | etcd backend |
| Infisical-backed secrets | `github.com/fikrimohammad/go-servicekit/confloader/client/infisical` | Infisical backend |
| An HTTP client for vendors | `github.com/fikrimohammad/go-servicekit/rest/client` | Hertz; retry, timeouts, RoundTripper |
| An HTTP server at the edge | `github.com/fikrimohammad/go-servicekit/rest/server` | Hertz; problem+json, request ID |
| To schedule recurring jobs | `github.com/fikrimohammad/go-servicekit/cron` | Job model |
| To run the scheduler | `github.com/fikrimohammad/go-servicekit/cron/engine` | Engine |
| SQL persistence | `github.com/fikrimohammad/go-servicekit/db` | MySQL/PostgreSQL; named queries |
| To consume JetStream messages | `github.com/fikrimohammad/go-servicekit/nats/consumer` | Durable consumer |
| To publish JetStream messages | `github.com/fikrimohammad/go-servicekit/nats/producer` | Producer |
| A Redis client | `github.com/fikrimohammad/go-servicekit/redis` | Metrics and tracing |
| S3 objects and presigned URLs | `github.com/fikrimohammad/go-servicekit/s3` | Multipart upload |

`github.com/fikrimohammad/go-servicekit/nats` and
`github.com/fikrimohammad/go-servicekit/rest` are module roots with no
importable package; use the subpackages above. Importing a root compiles to an
empty package only so `go doc` can point you to them.

## Development

The repository is a Go workspace; all modules are listed in `go.work`.

```bash
# build every module
go build ./...

# test every module
go test -count=1 ./...

# run inside a single module
(cd errs && go test ./...)
```

## CI / Releases

GitHub Actions workflows live in `.github/workflows/`:

- **`ci.yml`** — on PR + push to main. Discovers every Go module and runs a
  matrix job per module: `gofmt`, `golangci-lint` (config at repo root,
  `.golangci.yml`), `go vet`, `go test` (plus `-race`), and a
  `go mod tidy -diff` check. Every module except those in `LAX_MODULES` (today
  only `confloader`) is linted on the full tree and checked by gostyle.
  gostyle also has its own test job.
- **`doc-contracts`** job in `ci.yml` runs
  `.github/scripts/check-doc-contracts.sh` and its tests: every public package
  must have a `doc.go` contract and a row in the import map.
- **`coverage-report.yml`** — on PR. Runs tests with coverage per module and
  posts a per-module coverage table as a PR comment.
- **`db-integration`** job in `ci.yml` runs the MySQL and PostgreSQL integration
  tests against service containers.
- **`release-dry-run`** job in `ci.yml` runs the release script's unit tests,
  checks that every sibling `require` matches `VERSION`, and dry-runs the
  release while `VERSION` is unreleased.
- **`release.yml`** — manual only (`workflow_dispatch`); nothing is tagged on
  push. Run `gh workflow run release.yml` on main. It releases the version in
  `VERSION`, tags every module at that version in one atomic push, and then
  checks that each module installs from proxy.golang.org. It refuses a version
  that already exists as a tag or in proxy.golang.org.

CI does not rely on `go.work` (it is gitignored); each module builds standalone
via its `replace` directives.

## Versioning

All modules share one version, set in `VERSION` (currently `v0.1.1`). Every
sibling `require` must equal it, and CI checks this. Each release tags every
module as `<module>/<version>`. All modules are in development (v0), so any
release may break the API. Never reuse a version: fix a bad release forward
with a new patch version. See `AGENTS.md` for the full release steps.
