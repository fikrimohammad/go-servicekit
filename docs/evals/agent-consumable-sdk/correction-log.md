# Correction log

One row per SDK-usage correction. Leave the tables empty when a run needs none;
an empty log is a pass.

**Correction kinds:** `import`, `config`, `lifecycle`, `error-handling`,
`telemetry`.

| Sample | File:line | Kind | Reason | Resolution |
| --- | --- | --- | --- | --- |

No corrections.

## Findings

A finding is a capability the SDK lacked. It stops the run instead of turning
into a correction.

| Sample | Package | Missing capability | Notes |
| --- | --- | --- | --- |

No findings. Every sample was solvable from the contracts.

## Runs

| Sample | Date | Runner | Done command | Outcome |
| --- | --- | --- | --- | --- |
| errs (dry run) | 2026-11-08 | fresh agent | `GOWORK=off go test ./...` | pass, zero corrections |
| errs | 2026-11-08 | fresh agent | `GOWORK=off go test ./...` | pass, zero corrections |
| db | 2026-11-08 | fresh agent | `GOWORK=off go test ./...` | pass, zero corrections |
| confloader | 2026-11-08 | fresh agent | `GOWORK=off go test ./...` | pass, zero corrections |
| rest-server | 2026-11-08 | fresh agent | `GOWORK=off go test ./...` | pass, zero corrections |
| observability | 2026-11-08 | fresh agent | `GOWORK=off go test ./...` | pass, zero corrections |
| nats-consumer | 2026-11-08 | fresh agent | `GOWORK=off go test ./...` | pass, zero corrections |
