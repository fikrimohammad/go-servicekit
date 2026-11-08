package eval

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/fikrimohammad/go-servicekit/errs"
	"github.com/fikrimohammad/go-servicekit/rest/server"
)

// freeAddr returns a loopback address that is free right now.
func freeAddr(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer l.Close()
	return l.Addr().String()
}

// newServer builds a server on addr with GET /orders/:id answering 404 through
// the SDK's problem+json error helper.
//
// TODO: call server.New(server.Config{Addr: addr}), register the route on
// srv.Hertz(), and answer with server.Error(c, http.StatusNotFound,
// errs.New(404, "order not found")).
func newServer(_ string) (*server.Server, error) {
	_ = server.Config{}
	_ = errs.New(0, "")
	return nil, errors.New("not implemented")
}

func TestNotFoundProblem(t *testing.T) {
	addr := freeAddr(t)
	srv, err := newServer(addr)
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errc := make(chan error, 1)
	go func() { errc <- srv.Run(ctx) }()
	defer func() {
		cancel()
		<-errc
	}()

	url := "http://" + addr + "/orders/42"
	var resp *http.Response
	for i := 0; i < 200; i++ {
		resp, err = http.Get(url)
		if err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
	if got := resp.Header.Get("Content-Type"); got != "application/problem+json" {
		t.Fatalf("Content-Type = %q, want application/problem+json", got)
	}

	var body struct {
		Status    int    `json:"status"`
		Title     string `json:"title"`
		Detail    string `json:"detail"`
		Code      int    `json:"code"`
		RequestID string `json:"request_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode problem: %v", err)
	}
	if body.Status != http.StatusNotFound {
		t.Errorf("body.status = %d, want 404", body.Status)
	}
	if body.Title != http.StatusText(http.StatusNotFound) {
		t.Errorf("body.title = %q, want %q", body.Title, http.StatusText(http.StatusNotFound))
	}
	if body.Detail != "order not found" {
		t.Errorf("body.detail = %q, want order not found", body.Detail)
	}
	if body.Code != 404 {
		t.Errorf("body.code = %d, want 404", body.Code)
	}
	if body.RequestID == "" {
		t.Error("body.request_id is empty")
	}
}
