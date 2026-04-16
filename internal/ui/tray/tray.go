package tray

import (
	"fmt"
	"strings"
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

func (t *Tray) UpdateTooltip(s core.State) {
	t.mu.Lock()
	defer t.mu.Unlock()

	var lines []string
	lines = append(lines, "SpeedForce ⚡")

	if s.IP.PublicIP != "" {
		ip := s.IP.PublicIP
		if s.IP.Country != "" {
			ip = fmt.Sprintf("%s (%s)", s.IP.PublicIP, s.IP.Country)
		}
		lines = append(lines, "IP: "+ip)
	}

	up, down := 0, 0
	for _, p := range s.HTTPS {
		if p.IsUp() {
			up++
		} else {
			down++
		}
	}
	lines = append(lines, fmt.Sprintf("Services: %d/%d up", up, up+down))

	if down > 0 {
		var names []string
		for _, p := range s.HTTPS {
			if !p.IsUp() {
				names = append(names, p.Name)
			}
		}
		lines = append(lines, "Down: "+strings.Join(names, ", "))
	}

	systray.SetTooltip(strings.Join(lines, "\n"))
}
