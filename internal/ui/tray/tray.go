package tray

import (
	"sync"

	"github.com/getlantern/systray"
	"github.com/yosh3289/speedforce/internal/core"
)

type Tray struct {
	mu       sync.Mutex
	onQuit   func()
	onDetail func()
}

func New(onDetail, onQuit func()) *Tray {
	return &Tray{onDetail: onDetail, onQuit: onQuit}
}

func (t *Tray) Run() {
	systray.Run(t.onReady, t.onExit)
}

func (t *Tray) onReady() {
	systray.SetIcon(IconFor(core.StatusUnknown))
	systray.SetTooltip("SpeedForce")

	mDetail := systray.AddMenuItem("Show Details", "Open detail window")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Exit SpeedForce")

	go func() {
		for {
			select {
			case <-mDetail.ClickedCh:
				if t.onDetail != nil {
					t.onDetail()
				}
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func (t *Tray) onExit() {
	if t.onQuit != nil {
		t.onQuit()
	}
}

func (t *Tray) SetStatus(status core.OverallStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()
	systray.SetIcon(IconFor(status))
}
