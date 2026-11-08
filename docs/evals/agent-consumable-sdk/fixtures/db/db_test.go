package eval

import (
	"context"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"

	"github.com/fikrimohammad/go-servicekit/db"
)

// newMock registers a sqlmock database and returns its DSN. The test passes the
// DSN to loadName; use driver "sqlmock" and that DSN.
func newMock(t *testing.T) (string, sqlmock.Sqlmock) {
	t.Helper()
	dsn := "eval_db_" + t.Name()
	sqlDB, mock, err := sqlmock.NewWithDSN(dsn,
		sqlmock.MonitorPingsOption(true),
		sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("sqlmock.NewWithDSN: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return dsn, mock
}

// loadName returns the name of the user with id.
//
// TODO: connect with the SDK's db package (driver "sqlmock", the given DSN),
// run SELECT name FROM users WHERE id = :id with NamedGetContext, scan into a
// struct with a db:"name" tag, and close the pool.
func loadName(_ context.Context, _ string, _ int64) (string, error) {
	_ = db.Config{}
	return "", nil
}

func TestLoadName(t *testing.T) {
	dsn, mock := newMock(t)
	mock.ExpectPing()
	mock.ExpectQuery("SELECT name FROM users WHERE id = ?").
		WithArgs(int64(42)).
		WillReturnRows(sqlmock.NewRows([]string{"name"}).AddRow("ada"))

	name, err := loadName(context.Background(), dsn, 42)
	if err != nil {
		t.Fatalf("loadName: %v", err)
	}
	if name != "ada" {
		t.Errorf("name = %q, want ada", name)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
