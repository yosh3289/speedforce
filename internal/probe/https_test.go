package probe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPSProber_2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	p := NewHTTPSProber(NewClient(ClientOptions{Timeout: 2 * time.Second, ProxyMode: "none"}))
	res := p.Probe(context.Background(), "test", srv.URL)
	if !res.IsUp() {
		t.Errorf("expected up, got err=%v code=%d", res.Err, res.StatusCode)
	}
	if res.StatusCode != 200 {
		t.Errorf("status = %d, want 200", res.StatusCode)
	}
}

func TestHTTPSProber_401IsUp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer srv.Close()

	p := NewHTTPSProber(NewClient(ClientOptions{Timeout: 2 * time.Second, ProxyMode: "none"}))
	res := p.Probe(context.Background(), "test", srv.URL)
	if !res.IsUp() {
		t.Errorf("401 should be up (network path OK, auth missing); got down")
	}
}

func TestHTTPSProber_503IsDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()

	p := NewHTTPSProber(NewClient(ClientOptions{Timeout: 2 * time.Second, ProxyMode: "none"}))
	res := p.Probe(context.Background(), "test", srv.URL)
	if res.IsUp() {
		t.Errorf("503 should be down")
	}
}

func TestHTTPSProber_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer srv.Close()

	p := NewHTTPSProber(NewClient(ClientOptions{Timeout: 100 * time.Millisecond, ProxyMode: "none"}))
	res := p.Probe(context.Background(), "test", srv.URL)
	if res.IsUp() {
		t.Error("timeout should be down")
	}
	if res.Err == nil {
		t.Error("expected error")
	}
}
