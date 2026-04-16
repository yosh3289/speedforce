package core

import "testing"

func TestComputeOverall_AllUp(t *testing.T) {
	probes := []ProbeResult{
		{Name: "a", StatusCode: 200, LatencyMs: 100},
		{Name: "b", StatusCode: 200, LatencyMs: 200},
	}
	sp := []StatuspageResult{{Indicator: StatuspageOperational}}
	if got := ComputeOverall(probes, sp); got != StatusHealthy {
		t.Errorf("got %v, want healthy", got)
	}
}

func TestComputeOverall_SlowIsDegraded(t *testing.T) {
	probes := []ProbeResult{
		{Name: "a", StatusCode: 200, LatencyMs: 3500},
	}
	if got := ComputeOverall(probes, nil); got != StatusDegraded {
		t.Errorf("got %v, want degraded", got)
	}
}

func TestComputeOverall_ThreePlusDownIsRed(t *testing.T) {
	probes := []ProbeResult{
		{Name: "a", Err: &testErr{"x"}},
		{Name: "b", Err: &testErr{"x"}},
		{Name: "c", Err: &testErr{"x"}},
		{Name: "d", StatusCode: 200, LatencyMs: 100},
	}
	if got := ComputeOverall(probes, nil); got != StatusDown {
		t.Errorf("got %v, want down", got)
	}
}

func TestComputeOverall_MajorOutageIsRed(t *testing.T) {
	probes := []ProbeResult{{StatusCode: 200, LatencyMs: 100}}
	sp := []StatuspageResult{{Indicator: StatuspageMajor}}
	if got := ComputeOverall(probes, sp); got != StatusDown {
		t.Errorf("got %v, want down on major outage", got)
	}
}

func TestComputeOverall_OneOrTwoDownIsYellow(t *testing.T) {
	probes := []ProbeResult{
		{Name: "a", Err: &testErr{"x"}},
		{Name: "b", StatusCode: 200, LatencyMs: 100},
		{Name: "c", StatusCode: 200, LatencyMs: 100},
	}
	if got := ComputeOverall(probes, nil); got != StatusDegraded {
		t.Errorf("got %v, want degraded", got)
	}
}
