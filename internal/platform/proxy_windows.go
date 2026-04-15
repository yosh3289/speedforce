//go:build windows

package platform

import (
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func systemProxyImpl() func(*http.Request) (*url.URL, error) {
	return func(req *http.Request) (*url.URL, error) {
		return readWindowsProxy()
	}
}

func readWindowsProxy() (*url.URL, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.QUERY_VALUE)
	if err != nil {
		return nil, nil
	}
	defer k.Close()

	enabled, _, err := k.GetIntegerValue("ProxyEnable")
	if err != nil || enabled == 0 {
		return nil, nil
	}
	server, _, err := k.GetStringValue("ProxyServer")
	if err != nil || server == "" {
		return nil, nil
	}
	// ProxyServer may be "host:port" or "http=host:port;https=host:port"
	if strings.Contains(server, "=") {
		for _, part := range strings.Split(server, ";") {
			if strings.HasPrefix(part, "https=") {
				return url.Parse("http://" + strings.TrimPrefix(part, "https="))
			}
		}
		for _, part := range strings.Split(server, ";") {
			if strings.HasPrefix(part, "http=") {
				return url.Parse("http://" + strings.TrimPrefix(part, "http="))
			}
		}
	}
	return url.Parse("http://" + server)
}
