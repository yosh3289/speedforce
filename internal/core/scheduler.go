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

func (s *Scheduler) updateAdaptive(results []ProbeResult) {
	if !s.cfg.AdaptiveEnabled {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check max-fast-duration cap first
	if s.mode == TickFast && !s.fastSince.IsZero() &&
		time.Since(s.fastSince) >= s.cfg.MaxFastDuration {
		s.mode = TickNormal
		s.fastSince = time.Time{}
		s.stableTicks = 0
		s.recordLatencies(results)
		return
	}

	trigger := false
	allStable := true

	for _, r := range results {
		prev, seen := s.lastLatencies[r.Name]
		curr := r.LatencyMs

		if seen {
			delta := absInt64(curr - prev)
			if delta > s.cfg.ThresholdMs {
				trigger = true
			}
			if delta >= s.cfg.RecoveryMs {
				allStable = false
			}

			prevUp := prev > 0
			currUp := (ProbeResult{StatusCode: r.StatusCode, Err: r.Err}).IsUp()
			if prevUp != currUp {
				trigger = true
			}
		} else {
			allStable = false
		}
	}

	switch s.mode {
	case TickNormal:
		if trigger {
			s.mode = TickFast
			s.fastSince = time.Now()
			s.stableTicks = 0
		}
	case TickFast:
		if trigger {
			s.stableTicks = 0
		} else if allStable {
			s.stableTicks++
			if s.stableTicks >= 3 {
				s.mode = TickNormal
				s.fastSince = time.Time{}
				s.stableTicks = 0
			}
		} else {
			s.stableTicks = 0
		}
	}

	s.recordLatencies(results)
}

func (s *Scheduler) recordLatencies(results []ProbeResult) {
	for _, r := range results {
		if r.Err == nil && r.LatencyMs > 0 {
			s.lastLatencies[r.Name] = r.LatencyMs
		} else {
			s.lastLatencies[r.Name] = 0
		}
	}
}

func absInt64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
