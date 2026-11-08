# Sample: rest/server

## Goal

Implement `newServer` in `server_test.go` using the SDK's `rest/server`
package.

- Create a server with `server.New(server.Config{Addr: addr})`.
- Register GET `/orders/:id` on `srv.Hertz()`.
- The handler answers with `server.Error(c, http.StatusNotFound,
  errs.New(404, "order not found"))`.

Do not change the test or add dependencies.

## Allowed files

- `server_test.go`

## Done command

```bash
go test ./...
```

## Stop rule

If the `rest/server` contract lacks something this goal needs, stop and report
the gap. Do not add a shim, do not edit the SDK, and do not work around it with
another package.

## Grader checklist

- [ ] Imports `github.com/fikrimohammad/go-servicekit/rest/server` and `errs`.
- [ ] Builds with `server.New` and registers the route on `Hertz()`.
- [ ] Answers with `server.Error`, not hand-written JSON.
- [ ] Zero SDK-usage corrections.
