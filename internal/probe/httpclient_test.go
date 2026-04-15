package probe

import (
	"net/url"
	"testing"
	"time"
)

func TestNewClient_NoProxy(t *testing.T) {
	c := NewClient(ClientOptions{Timeout: 2 * time.Second, ProxyMode: "none"})
	if c == nil {
		t.Fatal("client is nil")
	}
	if c.Timeout != 2*time.Second {
		t.Errorf("timeout = %v, want 2s", c.Timeout)
	}
}

func TestNewClient_ManualProxy(t *testing.T) {
	c := NewClient(ClientOptions{
		Timeout:   2 * time.Second,
		ProxyMode: "manual",
		ProxyURL:  "http://127.0.0.1:7890",
	})
	if c == nil {
		t.Fatal("client is nil")
	}
	tr, ok := c.Transport.(*proxyTransport)
	if !ok {
		t.Fatalf("transport not *proxyTransport: %T", c.Transport)
	}
	u, _ := url.Parse("http://example.com")
	got, err := tr.proxyFunc(&httpRequest{URL: u})
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.String() != "http://127.0.0.1:7890" {
		t.Errorf("proxy = %v, want http://127.0.0.1:7890", got)
	}
}
