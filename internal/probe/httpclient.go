package probe

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type ProxyMode string

const (
	ProxyAuto   ProxyMode = "auto"
	ProxyManual ProxyMode = "manual"
	ProxyNone   ProxyMode = "none"
)

type ClientOptions struct {
	Timeout       time.Duration
	ProxyMode     string
	ProxyURL      string
	SystemProxyFn func(*http.Request) (*url.URL, error)
}

type httpRequest = http.Request

type proxyTransport struct {
	*http.Transport
	proxyFunc func(*httpRequest) (*url.URL, error)
}

func NewClient(opts ClientOptions) *http.Client {
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Second
	}

	tr := &http.Transport{
		DisableKeepAlives:     true,
		TLSHandshakeTimeout:   opts.Timeout,
		ResponseHeaderTimeout: opts.Timeout,
	}

	var proxyFn func(*http.Request) (*url.URL, error)

	switch ProxyMode(opts.ProxyMode) {
	case ProxyManual:
		u, err := url.Parse(opts.ProxyURL)
		if err != nil {
			proxyFn = func(*http.Request) (*url.URL, error) {
				return nil, fmt.Errorf("invalid manual proxy URL: %w", err)
			}
		} else {
			proxyFn = func(*http.Request) (*url.URL, error) { return u, nil }
		}
	case ProxyNone:
		proxyFn = nil
	default:
		if opts.SystemProxyFn != nil {
			proxyFn = opts.SystemProxyFn
		} else {
			proxyFn = http.ProxyFromEnvironment
		}
	}

	tr.Proxy = proxyFn

	return &http.Client{
		Timeout:   opts.Timeout,
		Transport: &proxyTransport{Transport: tr, proxyFunc: proxyFn},
	}
}

func (t *proxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.Transport.RoundTrip(req)
}
