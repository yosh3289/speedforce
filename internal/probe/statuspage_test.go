package probe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/yosh3289/speedforce/internal/core"
)

func serveFile(t *testing.T, path string) *httptest.Server {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
}

func TestStatuspageProber_Operational(t *testing.T) {
	srv := serveFile(t, "testdata/statuspage_operational.json")
	defer srv.Close()

	p := NewStatuspageProber(NewClient(ClientOptions{Timeout: 2 * time.Second, ProxyMode: "none"}))
	res := p.Probe(context.Background(), "Anthropic", srv.URL)
	if res.Indicator != core.StatuspageOperational {
		t.Errorf("indicator = %q, want operational(none)", res.Indicator)
	}
	if res.Description == "" {
		t.Error("description empty")
	}
}

func TestStatuspageProber_MajorOutage(t *testing.T) {
	srv := serveFile(t, "testdata/statuspage_major.json")
	defer srv.Close()

	p := NewStatuspageProber(NewClient(ClientOptions{Timeout: 2 * time.Second, ProxyMode: "none"}))
	res := p.Probe(context.Background(), "OpenAI", srv.URL)
	if res.Indicator != core.StatuspageMajor {
		t.Errorf("indicator = %q, want major", res.Indicator)
	}
}

func TestStatuspageProber_FetchError(t *testing.T) {
	p := NewStatuspageProber(NewClient(ClientOptions{Timeout: 200 * time.Millisecond, ProxyMode: "none"}))
	res := p.Probe(context.Background(), "x", "http://127.0.0.1:1")
	if res.Err == nil {
		t.Error("expected error for unreachable endpoint")
	}
}
