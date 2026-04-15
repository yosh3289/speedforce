package core

import "testing"

func TestOverallStatus_Color(t *testing.T) {
	cases := []struct {
		name     string
		status   OverallStatus
		wantHex  string
	}{
		{"healthy is blue", StatusHealthy, "#2979FF"},
		{"degraded is yellow", StatusDegraded, "#FFC400"},
		{"down is red", StatusDown, "#D50000"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.status.ColorHex(); got != c.wantHex {
				t.Errorf("got %s, want %s", got, c.wantHex)
			}
		})
	}
}

func TestProbeResult_IsUp(t *testing.T) {
	up := ProbeResult{StatusCode: 401, Err: nil}
	if !up.IsUp() {
		t.Error("401 should be considered up (reachable)")
	}
	down := ProbeResult{StatusCode: 0, Err: errTestTimeout}
	if down.IsUp() {
		t.Error("timeout should be considered down")
	}
	serverErr := ProbeResult{StatusCode: 503, Err: nil}
	if serverErr.IsUp() {
		t.Error("5xx should be considered down")
	}
}

var errTestTimeout = &testErr{msg: "timeout"}

type testErr struct{ msg string }

func (e *testErr) Error() string { return e.msg }
