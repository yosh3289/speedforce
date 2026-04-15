//go:build windows

package platform

import "testing"

func TestSystemProxyFunc_ReturnsFunction(t *testing.T) {
	fn := SystemProxyFunc()
	if fn == nil {
		t.Fatal("got nil proxy func")
	}
	// Should not panic regardless of registry state
	_, _ = fn(nil)
}
