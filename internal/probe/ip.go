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
	Country string `json:"country"`
	City    string `json:"city"`
	Org     string `json:"isp"`
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

// firstLANIP returns the machine's LAN address: the local address of the
// interface outbound traffic uses. Taking the first interface address instead
// picks unpredictable adapters, e.g. a disconnected Wi-Fi card's 169.254.x.
func firstLANIP() string {
	var cands []net.IP
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, a := range addrs {
			if ipNet, ok := a.(*net.IPNet); ok {
				cands = append(cands, ipNet.IP)
			}
		}
	}
	return pickLANIP(defaultRouteIP(), cands)
}

// defaultRouteIP is the local address the OS would use to reach the internet.
// Connecting a UDP socket only consults the routing table; nothing is sent.
func defaultRouteIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return nil
	}
	defer func() { _ = conn.Close() }()
	if a, ok := conn.LocalAddr().(*net.UDPAddr); ok {
		return a.IP
	}
	return nil
}

// pickLANIP prefers the default route's address, else the first private
// address, else any other usable one. Loopback, link-local (169.254.x) and
// non-IPv4 addresses are never chosen.
func pickLANIP(route net.IP, cands []net.IP) string {
	usable := func(ip net.IP) bool {
		ip4 := ip.To4()
		return ip4 != nil && !ip4.IsLoopback() && !ip4.IsLinkLocalUnicast() && !ip4.IsUnspecified()
	}
	if usable(route) {
		return route.To4().String()
	}
	fallback := ""
	for _, ip := range cands {
		if !usable(ip) {
			continue
		}
		if ip.IsPrivate() {
			return ip.To4().String()
		}
		if fallback == "" {
			fallback = ip.To4().String()
		}
	}
	return fallback
}
