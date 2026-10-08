package probe

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPickLANIP(t *testing.T) {
	ips := func(ss ...string) []net.IP {
		var out []net.IP
		for _, s := range ss {
			out = append(out, net.ParseIP(s))
		}
		return out
	}
	// A real machine: Wi-Fi, Bluetooth and virtual adapters without DHCP get
	// 169.254.x link-local addresses; only Ethernet carries traffic.
	machine := ips("169.254.18.137", "169.254.49.91", "172.17.128.1",
		"169.254.126.71", "192.168.147.168", "169.254.123.95", "127.0.0.1")

	tests := []struct {
		name  string
		route net.IP
		cands []net.IP
		want  string
	}{
		{"address of the default route wins", net.ParseIP("192.168.147.168"), machine, "192.168.147.168"},
		{"no route: first private, never link-local", nil, ips("169.254.123.95", "192.168.1.5"), "192.168.1.5"},
		{"link-local route is ignored", net.ParseIP("169.254.123.95"), ips("169.254.123.95", "10.0.0.7"), "10.0.0.7"},
		{"only link-local and loopback: nothing", nil, ips("169.254.1.1", "127.0.0.1"), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pickLANIP(tt.route, tt.cands); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIPProber_FetchesPublicAndGeo(t *testing.T) {
	ipSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ip":"203.0.113.1"}`))
	}))
	defer ipSrv.Close()

	geoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"country":"United States","city":"San Francisco","isp":"Cloudflare"}`))
	}))
	defer geoSrv.Close()

	p := NewIPProber(
		NewClient(ClientOptions{Timeout: 2 * time.Second, ProxyMode: "none"}),
		ipSrv.URL,
		geoSrv.URL+"/{ip}",
	)

	info, err := p.Probe(context.Background())
	if err != nil {
		t.Fatalf("probe err: %v", err)
	}
	if info.PublicIP != "203.0.113.1" {
		t.Errorf("ip = %q", info.PublicIP)
	}
	if info.Country != "United States" {
		t.Errorf("country = %q", info.Country)
	}
	if info.City != "San Francisco" {
		t.Errorf("city = %q", info.City)
	}
	if info.ISP != "Cloudflare" {
		t.Errorf("isp = %q", info.ISP)
	}
	if info.LANIP == "" {
		t.Error("LANIP should be set")
	}
}
