# Sample: errs

## Goal

Implement `loadUser` in `errors_test.go` using the SDK's `errs` package.

- `"missing"` returns an error with code 404, the message `user not found`,
  metadata `id=missing`, and a non-empty debug detail.
- `"db-down"` wraps `errDBDown` with code 503 and the message `load user`.
- Any other id returns `"user-" + id` and a nil error.

Do not change the tests or add dependencies.

## Allowed files

- `errors_test.go`

## Done command

```bash
go test ./...
```

## Stop rule

If the `errs` contract lacks something this goal needs, stop and report the gap.
Do not add a shim, do not edit the SDK, and do not work around it with another
package.

## Grader checklist

- [ ] Imports `github.com/fikrimohammad/go-servicekit/errs`, not `fmt.Errorf`.
- [ ] Uses `errs.New`/`errs.Wrap` and attaches metadata with `WithMeta`.
- [ ] `errors.Is` matches the 404 code and the wrapped 503 cause.
- [ ] Tests and dependencies are unchanged.
- [ ] Zero SDK-usage corrections.
