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

// countryCode reduces a full country name to a 2-letter code heuristic
// (e.g. "United States" → "US"). ip-api.com only returns the full name.
func countryCode(name string) string {
	if name == "" {
		return ""
	}
	// Take initials of words, cap at 3 letters
	words := strings.Fields(name)
	if len(words) >= 2 {
		code := ""
		for _, w := range words {
			if len(code) >= 3 {
				break
			}
			code += strings.ToUpper(string(w[0]))
		}
		return code
	}
	if len(name) >= 3 {
		return strings.ToUpper(name[:3])
	}
	return strings.ToUpper(name)
}

// maxTooltipLen is the Windows NOTIFYICONDATA.szTip limit (128 chars
// including terminator; keep a safety margin).
const maxTooltipLen = 120

func (t *Tray) UpdateTooltip(s core.State) {
	t.mu.Lock()
	defer t.mu.Unlock()

	up, down := 0, 0
	for _, p := range s.HTTPS {
		if p.IsUp() {
			up++
		} else {
			down++
		}
	}

	var lines []string
	header := fmt.Sprintf("SpeedForce ⚡ %d/%d up", up, up+down)
	lines = append(lines, header)

	if s.IP.PublicIP != "" {
		ipLine := s.IP.PublicIP
		if cc := countryCode(s.IP.Country); cc != "" {
			ipLine = fmt.Sprintf("%s %s", s.IP.PublicIP, cc)
		}
		lines = append(lines, ipLine)
	}

	if down > 0 {
		for _, p := range s.HTTPS {
			if !p.IsUp() {
				lines = append(lines, "✗ "+p.Name)
			}
		}
	}

	text := strings.Join(lines, "\n")
	if len([]rune(text)) > maxTooltipLen {
		runes := []rune(text)
		text = string(runes[:maxTooltipLen-1]) + "…"
	}
	systray.SetTooltip(text)
}
