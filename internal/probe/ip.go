package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/yosh3289/speedforce/internal/core"
)

type IPProber struct {
	client      *http.Client
	publicIPURL string
	geoURLTmpl  string // contains "{ip}" placeholder
}

func NewIPProber(client *http.Client, publicIPURL, geoURLTmpl string) *IPProber {
	return &IPProber{client: client, publicIPURL: publicIPURL, geoURLTmpl: geoURLTmpl}
}

func (p *IPProber) Probe(ctx context.Context) (core.IPInfo, error) {
	info := core.IPInfo{LANIP: firstLANIP(), FetchedAt: time.Now()}

	ip, err := p.fetchPublicIP(ctx)
	if err != nil {
		return info, fmt.Errorf("public ip: %w", err)
	}
	info.PublicIP = ip

	geo, err := p.fetchGeo(ctx, ip)
	if err != nil {
		return info, fmt.Errorf("geo: %w", err)
	}
	info.Country = geo.Country
	info.City = geo.City
	info.ISP = geo.Org
	return info, nil
}

type ipifyResp struct {
	IP string `json:"ip"`
}

type geoResp struct {
	Country string `json:"country_name"`
	City    string `json:"city"`
	Org     string `json:"org"`
}

func (p *IPProber) fetchPublicIP(ctx context.Context) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, p.publicIPURL, nil)
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var r ipifyResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}
	return r.IP, nil
}

func (p *IPProber) fetchGeo(ctx context.Context, ip string) (geoResp, error) {
	url := strings.Replace(p.geoURLTmpl, "{ip}", ip, 1)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := p.client.Do(req)
	if err != nil {
		return geoResp{}, err
	}
	defer resp.Body.Close()
	var r geoResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return geoResp{}, err
	}
	return r, nil
}

func firstLANIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		if ip4 := ipNet.IP.To4(); ip4 != nil {
			return ip4.String()
		}
	}
	return ""
}
