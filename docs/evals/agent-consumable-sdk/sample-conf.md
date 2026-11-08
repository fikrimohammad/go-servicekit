# Sample: conf

## Goal

Implement `loadDebug` in `loader_test.go` using the SDK's `conf` package.

- Build a `conf.New[appConfig](ctx, p)`.
- `fake` implements `conf.Provider`; it is the backend.
- Read `Data().Debug.Get()` and return it.
- Close the loader before returning.

Do not change the test or add dependencies.

## Allowed files

- `loader_test.go`

## Done command

```bash
go test ./...
```

## Stop rule

If the `conf` contract lacks something this goal needs, stop and report the
gap. Do not add a shim, do not edit the SDK, and do not work around it with
another package.

## Grader checklist

- [ ] Imports `github.com/fikrimohammad/go-servicekit/conf`.
- [ ] Calls `conf.New[appConfig]` with the fake provider.
- [ ] Reads through `Getter.Get`, not a raw map.
- [ ] Calls `Close` on the loader.
- [ ] Zero SDK-usage corrections.
