package eval

import (
	"errors"
	"testing"

	"github.com/fikrimohammad/go-servicekit/errs"
)

// errDBDown is the cause loadUser must wrap for the "db-down" case.
var errDBDown = errors.New("database unavailable")

// loadUser returns the display name for id.
//
// Implement it with the SDK's errs package:
//   - "missing" returns an error with code 404, message "user not found",
//     metadata id=missing, and a non-empty debug detail;
//   - "db-down" wraps errDBDown with code 503 and message "load user";
//   - any other id returns "user-"+id and a nil error.
//
// The tests below define the exact behaviour. Do not change them.
func loadUser(id string) (string, error) {
	_ = id
	return "", errors.New("not implemented")
}

func TestLoadUserMissing(t *testing.T) {
	_, err := loadUser("missing")
	if err == nil {
		t.Fatal("loadUser(\"missing\") error = nil, want code 404")
	}
	if !errors.Is(err, errs.New(404, "")) {
		t.Fatalf("loadUser(\"missing\") error = %v, want code 404", err)
	}
	e := errs.AsError(err)
	if e == nil {
		t.Fatalf("loadUser(\"missing\") error = %T, want *errs.Error", err)
	}
	if got := e.Meta()["id"]; got != "missing" {
		t.Errorf("meta id = %v, want missing", got)
	}
	if e.Debug() == "" {
		t.Error("debug detail is empty, want a non-empty detail")
	}
}

func TestLoadUserDBDown(t *testing.T) {
	_, err := loadUser("db-down")
	if !errors.Is(err, errDBDown) {
		t.Fatalf("loadUser(\"db-down\") error = %v, want it to wrap errDBDown", err)
	}
	e := errs.AsError(err)
	if e == nil {
		t.Fatalf("loadUser(\"db-down\") error = %T, want *errs.Error", err)
	}
	if e.Code() != 503 {
		t.Errorf("code = %d, want 503", e.Code())
	}
}

func TestLoadUserOK(t *testing.T) {
	name, err := loadUser("42")
	if err != nil {
		t.Fatalf("loadUser(\"42\") error = %v, want nil", err)
	}
	if name != "user-42" {
		t.Errorf("name = %q, want user-42", name)
	}
}
