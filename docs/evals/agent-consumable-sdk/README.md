# Agent-consumable SDK eval kit

This kit measures one thing: whether a coding agent that sees only the SDK's
`go doc` contracts can wire a package correctly. It is harness-agnostic and
self-contained. An eval runner reads this file, picks a sample, and does not
need any other context about the SDK.

## What is here

| Path | Purpose |
| --- | --- |
| `sample-<name>.md` | The prompt, allowed files, done command, and grader checklist for one sample |
| `fixtures/<name>/` | A non-module template copied into a temporary workdir |
| `setup.sh` | Copies a fixture, points its SDK modules at this working tree, downloads deps, prints the workdir |
| `correction-log.md` | The template the grader fills for every run |

Samples: `db`, `errs`, `confloader`, `rest-server`, `observability`,
`nats-consumer`.

## Protocol

1. **Prepare.** Run `./setup.sh <sample>`. It prints a temporary workdir.
   Copy nothing else into it.
2. **Prompt.** Give the eval agent only `sample-<sample>.md` and the workdir.
   The prompt names the goal, the allowed files, the done command, and the stop
   rule.
3. **Work.** The agent edits only the allowed files and may run any Go command.
   It must not edit the SDK, add a shim, or edit this kit.
4. **Done.** Run the done command from the prompt. It must pass.
5. **Grade.** Diff the workdir against the fixture. Classify every edit, then
   fill one row in `correction-log.md` per SDK-usage correction. Record a
   finding for every capability the SDK lacked.

## Correction vs. not a correction

A **correction** is a human edit to SDK usage:

- an import path;
- constructor or `Config` wiring;
- lifecycle calls (`Start`, `Stop`, `Close`, `Shutdown`);
- SDK error handling, including sentinels and `errors.Is`/`As`;
- telemetry wiring.

Formatting, comments, test fixtures, and business logic are **not**
corrections. A run with zero corrections means the contract was sufficient.

## Stop rule

If the SDK has no API for what the prompt requires, stop. Report the gap in the
correction log as a finding: sample, package, missing capability. Do not add a
shim, do not edit the SDK, and do not work around it with a different package.
A finding is a successful outcome for the eval; it is how the SDK learns what
to add next.

## Rules

- One sample per run, in a fresh workdir, with an agent that has not seen the
  SDK before.
- The fixture is pinned by the SDK commit that contains this kit.
- Network is required for `go mod download`; no external service is needed.
- The workdir is a standalone module. If it sits inside a Go workspace, run Go
  commands with `GOWORK=off`, and give the runner a writable `GOCACHE`.
- Do not fix the SDK during a run, and do not change the prompt or fixture.
