package tray

import (
	"sync"

	"github.com/getlantern/systray"

	"github.com/yosh3289/speedforce/internal/core"
	"github.com/yosh3289/speedforce/internal/i18n"
)

type Callbacks struct {
	OnDetail   func()
	OnSettings func()
	OnQuit     func()
}

type Tray struct {
	i18n *i18n.Translator
	cb   Callbacks

	mu sync.Mutex
}

func New(tr *i18n.Translator, cb Callbacks) *Tray {
	return &Tray{i18n: tr, cb: cb}
}

func (t *Tray) Run() {
	systray.Run(t.onReady, t.onExit)
}

func (t *Tray) onReady() {
	systray.SetIcon(IconFor(core.StatusUnknown))
	systray.SetTooltip(t.i18n.T("tray.tooltip"))

	mDetail := systray.AddMenuItem(t.i18n.T("tray.menu.show_details"), "")
	mSettings := systray.AddMenuItem(t.i18n.T("tray.menu.settings"), "")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem(t.i18n.T("tray.menu.quit"), "")

	go func() {
		for {
			select {
			case <-mDetail.ClickedCh:
				if t.cb.OnDetail != nil {
					t.cb.OnDetail()
				}
			case <-mSettings.ClickedCh:
				if t.cb.OnSettings != nil {
					t.cb.OnSettings()
				}
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func (t *Tray) onExit() {
	if t.cb.OnQuit != nil {
		t.cb.OnQuit()
	}
}

func (t *Tray) SetStatus(status core.OverallStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()
	systray.SetIcon(IconFor(status))
}
