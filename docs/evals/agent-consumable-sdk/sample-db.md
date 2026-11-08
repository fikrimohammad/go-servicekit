# Sample: db

## Goal

Implement `loadName` in `db_test.go` using the SDK's `db` package.

- Connect with `db.Connect`, driver `"sqlmock"`, and the DSN the test provides.
- Run the named query `SELECT name FROM users WHERE id = :id` with `:id` = `id`.
- Scan into a struct and return the name.
- Close the pool before returning.

Do not change the test or add dependencies.

## Allowed files

- `db_test.go`

## Done command

```bash
go test ./...
```

## Stop rule

If the `db` contract lacks something this goal needs, stop and report the gap.
Do not add a shim, do not edit the SDK, and do not work around it with another
package.

## Grader checklist

- [ ] Imports `github.com/fikrimohammad/go-servicekit/db`.
- [ ] Calls `db.Connect` with `Config{Driver: "sqlmock", DSN: dsn}`.
- [ ] Uses `NamedGetContext` with a named placeholder and a `db`-tagged struct.
- [ ] Calls `Close` on the pool.
- [ ] Zero SDK-usage corrections.
