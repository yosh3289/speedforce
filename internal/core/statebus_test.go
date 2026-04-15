package core

import (
	"sync"
	"testing"
	"time"
)

func TestStateBus_PublishSubscribe(t *testing.T) {
	bus := NewStateBus()
	defer bus.Close()

	sub := bus.Subscribe()

	go func() {
		bus.Publish(State{
			HTTPS:     []ProbeResult{{Name: "Claude", StatusCode: 200}},
			UpdatedAt: time.Now(),
		})
	}()

	select {
	case s := <-sub:
		if len(s.HTTPS) != 1 || s.HTTPS[0].Name != "Claude" {
			t.Errorf("unexpected state: %+v", s)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for state")
	}
}

func TestStateBus_MultipleSubscribers(t *testing.T) {
	bus := NewStateBus()
	defer bus.Close()

	s1 := bus.Subscribe()
	s2 := bus.Subscribe()

	var wg sync.WaitGroup
	wg.Add(2)
	got := make([]State, 2)

	go func() { defer wg.Done(); got[0] = <-s1 }()
	go func() { defer wg.Done(); got[1] = <-s2 }()

	bus.Publish(State{Mode: TickFast})

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}

	for i, g := range got {
		if g.Mode != TickFast {
			t.Errorf("subscriber %d got mode %v", i, g.Mode)
		}
	}
}

func TestStateBus_Snapshot(t *testing.T) {
	bus := NewStateBus()
	defer bus.Close()
	bus.Publish(State{Mode: TickFast, UpdatedAt: time.Now()})
	time.Sleep(20 * time.Millisecond)
	s := bus.Snapshot()
	if s.Mode != TickFast {
		t.Errorf("snapshot mode = %v, want TickFast", s.Mode)
	}
}
