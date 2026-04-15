package core

import (
	"context"
	"sync"
	"time"
)

type HTTPSProbeFunc interface {
	Probe(ctx context.Context, name, url string) ProbeResult
}

type Target struct {
	Name string
	URL  string
}

type SchedulerConfig struct {
	Bus          *StateBus
	HTTPS        HTTPSProbeFunc
	HTTPSTargets []Target

	TickInterval    time.Duration
	FastInterval    time.Duration
	AdaptiveEnabled bool
	ThresholdMs     int64
	RecoveryMs      int64
	MaxFastDuration time.Duration
}

type Scheduler struct {
	cfg SchedulerConfig

	mu            sync.Mutex
	mode          TickMode
	fastSince     time.Time
	stableTicks   int
	lastLatencies map[string]int64
}

func NewScheduler(cfg SchedulerConfig) *Scheduler {
	return &Scheduler{
		cfg:           cfg,
		mode:          TickNormal,
		lastLatencies: make(map[string]int64),
	}
}

func (s *Scheduler) currentInterval() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mode == TickFast {
		return s.cfg.FastInterval
	}
	return s.cfg.TickInterval
}

func (s *Scheduler) Run(ctx context.Context) {
	timer := time.NewTimer(s.currentInterval())
	defer timer.Stop()

	s.doTick(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			s.doTick(ctx)
			timer.Reset(s.currentInterval())
		}
	}
}

func (s *Scheduler) doTick(ctx context.Context) {
	var wg sync.WaitGroup
	results := make([]ProbeResult, len(s.cfg.HTTPSTargets))
	for i, t := range s.cfg.HTTPSTargets {
		wg.Add(1)
		go func(i int, t Target) {
			defer wg.Done()
			results[i] = s.cfg.HTTPS.Probe(ctx, t.Name, t.URL)
		}(i, t)
	}
	wg.Wait()

	s.updateAdaptive(results)

	s.cfg.Bus.Publish(State{
		HTTPS:     results,
		Mode:      s.currentMode(),
		UpdatedAt: time.Now(),
	})
}

func (s *Scheduler) currentMode() TickMode {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mode
}

// updateAdaptive: real implementation added in Task 12. For Task 11 it's a no-op.
func (s *Scheduler) updateAdaptive(results []ProbeResult) {
	// TODO: Task 12 implements adaptive tick state machine
}
