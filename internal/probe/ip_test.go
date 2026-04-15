package probe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIPProber_FetchesPublicAndGeo(t *testing.T) {
	ipSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ip":"203.0.113.1"}`))
	}))
	defer ipSrv.Close()

	geoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"country_name":"United States","city":"San Francisco","org":"Cloudflare"}`))
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
