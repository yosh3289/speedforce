package core

import "sync"

type StateBus struct {
	mu      sync.RWMutex
	current State
	subs    []chan State
	closed  bool
}

func NewStateBus() *StateBus {
	return &StateBus{}
}

func (b *StateBus) Subscribe() <-chan State {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan State, 8)
	b.subs = append(b.subs, ch)
	return ch
}

func (b *StateBus) Publish(s State) {
	b.mu.Lock()
	b.current = s
	subs := make([]chan State, len(b.subs))
	copy(subs, b.subs)
	b.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- s:
		default:
			// drop if subscriber slow; they'll get the next Publish
		}
	}
}

func (b *StateBus) Snapshot() State {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.current
}

func (b *StateBus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for _, ch := range b.subs {
		close(ch)
	}
	b.subs = nil
}
