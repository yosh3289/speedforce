package tray

import (
	"testing"
	"time"
)

// A left-click on the tray icon opens the detail window, the same as the
// "Show Details" menu item.
func TestLeftClickOpensDetail(t *testing.T) {
	opened := make(chan struct{}, 1)
	tr := New(nil, Callbacks{OnDetail: func() { opened <- struct{}{} }})
	quit := make(chan struct{})
	defer close(quit)
	go tr.handleEvents(nil, nil, quit, func() {})

	tr.onTapped()

	select {
	case <-opened:
	case <-time.After(2 * time.Second):
		t.Fatal("left-click did not open the detail window")
	}
}

// The tap callback runs synchronously inside the tray's window procedure on the
// message-pump thread, so it must return at once even when nothing is consuming
// clicks (e.g. the event goroutine is busy opening a window). A blocked pump is
// exactly the old dead-tray-menu symptom.
func TestLeftClickNeverBlocksMessagePump(t *testing.T) {
	tr := New(nil, Callbacks{})
	done := make(chan struct{})
	go func() {
		for i := 0; i < 100; i++ {
			tr.onTapped()
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("left-click blocked the tray message pump")
	}
}
