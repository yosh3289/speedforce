//go:build linux

package platform

import (
	"net/http"
	"net/url"
)

func systemProxyImpl() func(*http.Request) (*url.URL, error) {
	return http.ProxyFromEnvironment
}
