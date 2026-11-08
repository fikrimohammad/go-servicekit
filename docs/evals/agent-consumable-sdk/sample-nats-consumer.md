# Sample: nats/consumer

## Goal

Implement `consume` in `consumer_test.go` using the SDK's `nats/consumer`
package.

- Create a consumer with `consumer.New(identity.Info{Name: "eval"})`.
- Register the topic and group from the arguments, with
  `ConsumeFromWhere: consumer.ConsumeFromFirst`, so a message published before
  the consumer starts is delivered.
- Call the handler for each message; return nil after the first message is
  handled.
- Shut the consumer down before returning, and return `ctx.Err()` if the
  context ends first.

Do not change the test or add dependencies.

## Allowed files

- `consumer_test.go`

## Done command

```bash
go test ./...
```

## Stop rule

If the `nats/consumer` contract lacks something this goal needs, stop and
report the gap. Do not add a shim, do not edit the SDK, and do not work around
it with another package.

## Grader checklist

- [ ] Imports `github.com/fikrimohammad/go-servicekit/nats/consumer`.
- [ ] Calls `New`, `Register`, `Start`, and `Shutdown` in that order.
- [ ] Sets `ConsumeFromWhere: consumer.ConsumeFromFirst`.
- [ ] Zero SDK-usage corrections.
