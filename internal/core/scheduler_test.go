package core

import (
	"context"
	"sync"
	"testing"
	"time"
)

type fakeProber struct {
	mu        sync.Mutex
	calls     int
	latencyMs int64
}

func (f *fakeProber) Probe(ctx context.Context, name, url string) ProbeResult {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return ProbeResult{Name: name, URL: url, StatusCode: 200, LatencyMs: f.latencyMs}
}

func TestScheduler_TickFiresProbes(t *testing.T) {
	bus := NewStateBus()
	defer bus.Close()

	fp := &fakeProber{}
	sch := NewScheduler(SchedulerConfig{
		Bus:             bus,
		HTTPS:           fp,
		HTTPSTargets:    []Target{{Name: "a", URL: "http://a"}, {Name: "b", URL: "http://b"}},
		TickInterval:    50 * time.Millisecond,
		AdaptiveEnabled: false,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sch.Run(ctx)

	time.Sleep(140 * time.Millisecond)
	fp.mu.Lock()
	if fp.calls < 4 {
		t.Errorf("expected >=4 probe calls, got %d", fp.calls)
	}
	fp.mu.Unlock()
}

type fakeIPProber struct{ calls int }

func (f *fakeIPProber) Probe(ctx context.Context) (IPInfo, error) {
	f.calls++
	return IPInfo{PublicIP: "1.2.3.4"}, nil
}

type fakeStatuspageProber struct{ calls int }

func (f *fakeStatuspageProber) Probe(ctx context.Context, name, url string) StatuspageResult {
	f.calls++
	return StatuspageResult{Name: name, Indicator: StatuspageOperational}
}

func TestScheduler_IPRunsEveryNTicks(t *testing.T) {
	bus := NewStateBus()
	defer bus.Close()

	fp := &fakeProber{}
	fip := &fakeIPProber{}
	sch := NewScheduler(SchedulerConfig{
		Bus:                 bus,
		HTTPS:               fp,
		HTTPSTargets:        []Target{{Name: "a", URL: "http://a"}},
		IP:                  fip,
		IPRefreshEveryTicks: 3,
		TickInterval:        30 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sch.Run(ctx)

	time.Sleep(200 * time.Millisecond)
	if fip.calls < 2 {
		t.Errorf("IP probe calls = %d, want >= 2", fip.calls)
	}
	if fip.calls >= 6 {
		t.Errorf("IP probe ran too often: %d", fip.calls)
	}
}
