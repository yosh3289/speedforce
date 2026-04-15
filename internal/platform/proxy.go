package platform

import (
	"net/http"
	"net/url"
)

// SystemProxyFunc returns a function suitable for use as http.Transport.Proxy.
// When no proxy is configured or detection fails, returns a function that returns nil.
func SystemProxyFunc() func(*http.Request) (*url.URL, error) {
	return systemProxyImpl()
}
