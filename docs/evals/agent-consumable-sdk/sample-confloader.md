# Sample: confloader

## Goal

Implement `loadDebug` in `loader_test.go` using the SDK's `confloader`
package.

- Build a `confloader.New[appConfig](ctx, cfg, confloader.WithClient(client))`.
- Use a config the loader accepts (`ProviderEtcd`, a non-empty endpoint,
  credentials, and namespace); the fake client is the backend.
- Read `Data().Debug.Get(ctx)` and return it.
- Stop the loader before returning.

Do not change the test or add dependencies.

## Allowed files

- `loader_test.go`

## Done command

```bash
go test ./...
```

## Stop rule

If the `confloader` contract lacks something this goal needs, stop and report
the gap. Do not add a shim, do not edit the SDK, and do not work around it with
another package.

## Grader checklist

- [ ] Imports `github.com/fikrimohammad/go-servicekit/confloader`.
- [ ] Calls `confloader.New[appConfig]` with `confloader.WithClient`.
- [ ] Reads through `Getter.Get`, not a raw map.
- [ ] Calls `Stop` on the loader.
- [ ] Zero SDK-usage corrections.
