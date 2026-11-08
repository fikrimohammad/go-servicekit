# Sample: observability

## Goal

Implement `wire` in `wiring_test.go` using the SDK's `observability` packages.

- Create a logger with `logs.New` (JSON format, `io.Discard` writers) and
  install it with `logs.SetDefault`.
- Create metrics with `metrics.New`, install the provider with
  `otel.SetMeterProvider`.
- Create a tracer with `tracer.New`, install it with `tracer.SetGlobal`.
- Return a stop function that stops both clients.
- Use endpoint `127.0.0.1:4317`, `Insecure: true`, and a one-second exporter
  timeout: `Timeout` on the metrics config and `ExportTimeout` on the tracer
  config, so the test does not wait for a collector.

Do not change the test or add dependencies.

## Allowed files

- `wiring_test.go`

## Done command

```bash
go test ./...
```

## Stop rule

If the `observability` contracts lack something this goal needs, stop and
report the gap. Do not add a shim, do not edit the SDK, and do not work around
it with another package.

## Grader checklist

- [ ] Imports the `logs`, `metrics`, and `tracer` subpackages, not the root.
- [ ] Installs the logger, meter provider, and tracer via their documented
      install calls.
- [ ] Stops both clients through the returned function.
- [ ] Zero SDK-usage corrections.
