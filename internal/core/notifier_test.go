package core

import (
	"sync"
	"testing"
	"time"
)

type fakeToast struct {
	mu    sync.Mutex
	calls []string
}

func (f *fakeToast) Notify(title, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, title+"|"+body)
	return nil
}

func (f *fakeToast) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func TestNotifier_FiresOnUpToDown(t *testing.T) {
	ft := &fakeToast{}
	n := NewNotifier(NotifierConfig{
		Toast:       ft,
		NotifyHTTPS: map[string]bool{"Claude": true},
	})

	prev := State{HTTPS: []ProbeResult{{Name: "Claude", StatusCode: 200, LatencyMs: 100}}}
	curr := State{HTTPS: []ProbeResult{{Name: "Claude", Err: &testErr{"timeout"}}}}
	n.Observe(prev)
	n.Observe(curr)

	if ft.count() != 1 {
		t.Errorf("expected 1 toast, got %d", ft.count())
	}
}

func TestNotifier_SkipsServiceWithNotifyFalse(t *testing.T) {
	ft := &fakeToast{}
	n := NewNotifier(NotifierConfig{
		Toast:       ft,
		NotifyHTTPS: map[string]bool{"Claude": false},
	})

	prev := State{HTTPS: []ProbeResult{{Name: "Claude", StatusCode: 200, LatencyMs: 100}}}
	curr := State{HTTPS: []ProbeResult{{Name: "Claude", Err: &testErr{"timeout"}}}}
	n.Observe(prev)
	n.Observe(curr)

	if ft.count() != 0 {
		t.Errorf("expected no toast, got %d", ft.count())
	}
}

func TestNotifier_NoToastOnRecovery(t *testing.T) {
	ft := &fakeToast{}
	n := NewNotifier(NotifierConfig{
		Toast:       ft,
		NotifyHTTPS: map[string]bool{"Claude": true},
	})

	prev := State{HTTPS: []ProbeResult{{Name: "Claude", Err: &testErr{"t"}}}}
	curr := State{HTTPS: []ProbeResult{{Name: "Claude", StatusCode: 200, LatencyMs: 100}}}
	n.Observe(prev)
	n.Observe(curr)

	if ft.count() != 0 {
		t.Errorf("recovery should not fire toast, got %d", ft.count())
	}
}

func TestNotifier_FlapCooldown(t *testing.T) {
	ft := &fakeToast{}
	n := NewNotifier(NotifierConfig{
		Toast:       ft,
		NotifyHTTPS: map[string]bool{"Claude": true},
		CooldownDur: 5 * time.Minute,
	})

	upState := State{HTTPS: []ProbeResult{{Name: "Claude", StatusCode: 200, LatencyMs: 100}}}
	downState := State{HTTPS: []ProbeResult{{Name: "Claude", Err: &testErr{"t"}}}}

	n.Observe(upState)
	n.Observe(downState)
	n.Observe(upState)
	n.Observe(downState)

	if ft.count() != 1 {
		t.Errorf("expected 1 toast (cooldown suppressed second), got %d", ft.count())
	}
}

func TestNotifier_StatuspageMajorFires(t *testing.T) {
	ft := &fakeToast{}
	n := NewNotifier(NotifierConfig{
		Toast:                 ft,
		NotifyStatuspageMajor: true,
	})
	prev := State{Statuspage: []StatuspageResult{{Name: "Anthropic", Indicator: StatuspageOperational}}}
	curr := State{Statuspage: []StatuspageResult{{Name: "Anthropic", Indicator: StatuspageMajor}}}
	n.Observe(prev)
	n.Observe(curr)
	if ft.count() != 1 {
		t.Errorf("expected major-outage toast, got %d", ft.count())
	}
}
