# Correction log

One row per SDK-usage correction. Leave the tables empty when a run needs none;
an empty log is a pass.

**Correction kinds:** `import`, `config`, `lifecycle`, `error-handling`,
`telemetry`.

| Sample | File:line | Kind | Reason | Resolution |
| --- | --- | --- | --- | --- |

## Findings

A finding is a capability the SDK lacked. It stops the run instead of turning
into a correction.

| Sample | Package | Missing capability | Notes |
| --- | --- | --- | --- |

## Runs

| Sample | Date | Runner | Done command | Outcome |
| --- | --- | --- | --- | --- |
| errs | 2026-11-08 | fresh agent | `GOWORK=off go test ./...` | pass, zero corrections |
