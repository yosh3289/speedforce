package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type httpsProberImpl struct {
	client *http.Client
}

func (p *httpsProberImpl) Probe(ctx context.Context, name, url string) ProbeResult {
	req, _ := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	start := time.Now()
	resp, err := p.client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return ProbeResult{Name: name, URL: url, Err: err, LatencyMs: latency, Timestamp: time.Now()}
	}
	defer resp.Body.Close()
	return ProbeResult{Name: name, URL: url, StatusCode: resp.StatusCode, LatencyMs: latency, Timestamp: time.Now()}
}

func TestIntegration_SchedulerToBusToNotifier(t *testing.T) {
	var upSwitch atomic.Bool
	upSwitch.Store(true)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if upSwitch.Load() {
			w.WriteHeader(200)
			return
		}
		time.Sleep(300 * time.Millisecond)
	}))
	defer srv.Close()

	bus := NewStateBus()
	defer bus.Close()

	client := &http.Client{Timeout: 100 * time.Millisecond}
	sch := NewScheduler(SchedulerConfig{
		Bus:             bus,
		HTTPS:           &httpsProberImpl{client: client},
		HTTPSTargets:    []Target{{Name: "Service", URL: srv.URL}},
		TickInterval:    30 * time.Millisecond,
		AdaptiveEnabled: false,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sch.Run(ctx)

	ft := &fakeToast{}
	notifier := NewNotifier(NotifierConfig{
		Toast:       ft,
		NotifyHTTPS: map[string]bool{"Service": true},
	})

	sub := bus.Subscribe()

	done := make(chan struct{})
	go func() {
		timeout := time.After(3 * time.Second)
		for {
			select {
			case s, ok := <-sub:
				if !ok {
					close(done)
					return
				}
				notifier.Observe(s)
			case <-timeout:
				close(done)
				return
			}
		}
	}()

	time.Sleep(150 * time.Millisecond)
	upSwitch.Store(false)
	time.Sleep(300 * time.Millisecond)

	<-done

	if ft.count() == 0 {
		t.Error("expected at least one toast after up→down flip")
	}
}
