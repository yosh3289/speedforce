package core

import (
	"testing"
	"time"
)

func newSchedulerForAdaptive() *Scheduler {
	return NewScheduler(SchedulerConfig{
		TickInterval:    60 * time.Second,
		FastInterval:    10 * time.Second,
		AdaptiveEnabled: true,
		ThresholdMs:     1000,
		RecoveryMs:      300,
		MaxFastDuration: 10 * time.Minute,
	})
}

func TestAdaptive_EnterFastOnLatencySpike(t *testing.T) {
	s := newSchedulerForAdaptive()
	s.lastLatencies["a"] = 100

	s.updateAdaptive([]ProbeResult{{Name: "a", URL: "x", StatusCode: 200, LatencyMs: 1500}})

	if s.currentMode() != TickFast {
		t.Error("should enter fast mode on >1000ms delta")
	}
}

func TestAdaptive_EnterFastOnStateFlip(t *testing.T) {
	s := newSchedulerForAdaptive()
	s.lastLatencies["a"] = 100

	s.updateAdaptive([]ProbeResult{{Name: "a", URL: "x", StatusCode: 0, Err: &testErr{"timeout"}}})

	if s.currentMode() != TickFast {
		t.Error("should enter fast mode on up→down transition")
	}
}

func TestAdaptive_ExitAfterStableTicks(t *testing.T) {
	s := newSchedulerForAdaptive()
	s.mode = TickFast
	s.fastSince = time.Now()
	s.lastLatencies["a"] = 100

	for i := 0; i < 3; i++ {
		s.updateAdaptive([]ProbeResult{{Name: "a", URL: "x", StatusCode: 200, LatencyMs: 120}})
	}

	if s.currentMode() != TickNormal {
		t.Error("should exit fast mode after 3 stable ticks")
	}
}

func TestAdaptive_MaxFastDurationCap(t *testing.T) {
	s := newSchedulerForAdaptive()
	s.mode = TickFast
	s.fastSince = time.Now().Add(-11 * time.Minute)
	s.lastLatencies["a"] = 100

	s.updateAdaptive([]ProbeResult{{Name: "a", URL: "x", StatusCode: 200, LatencyMs: 1500}})

	if s.currentMode() != TickNormal {
		t.Error("should forcibly exit after max_fast_duration")
	}
}
