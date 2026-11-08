// Package db provides a small, standardized named-query API for MySQL and
// PostgreSQL on top of [sqlx.DB], recording OpenTelemetry traces and metrics
// for each executed statement.
//
// # When to use
//
// Use db when a service needs SQL with named parameters, a managed connection
// pool, and consistent telemetry. It is not an ORM, and it does not manage
// schema migrations.
//
// # Import
//
//	import "github.com/fikrimohammad/go-servicekit/db"
//
// # Minimal example
//
//	conn, err := db.Connect(ctx, db.Config{Driver: "pgx", DSN: dsn})
//	if err != nil {
//		return err
//	}
//	defer conn.Close()
//
//	var user struct {
//		ID    int64  `db:"id"`
//		Email string `db:"email"`
//	}
//	err = conn.NamedGetContext(ctx, &user,
//		"SELECT id, email FROM users WHERE id = :id", map[string]any{"id": 42})
//
// # Lifecycle
//
// [Connect] applies [Config.SetDefaults] and [Config.Validate], opens the pool,
// applies the pool settings, and pings it. [DB.Close] releases the pool. A
// transaction from [DB.Begin] must be committed or rolled back; deferring
// [Tx.Rollback] is safe after a successful Commit.
//
// # Common mistakes
//
//   - Forgetting Close, or forgetting to roll back a transaction on an early
//     return.
//   - Passing a query with positional placeholders, or argument keys that do
//     not match the :placeholders.
//   - Using a driver Connect does not know without also setting DSN.
//
// # Test recipe
//
// Register a sqlmock database with sqlmock.NewWithDSN, then call Connect with
// Driver "sqlmock" and that DSN. The driver is not built in, so Connect opens
// it as-is and no database server is needed.
package db
