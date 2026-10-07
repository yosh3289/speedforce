package tray

import (
	"fmt"
	"runtime"
	"strings"
	"sync"

	"fyne.io/systray"

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

	// tapCh carries left-clicks on the tray icon from the message pump to
	// handleEvents.
	tapCh chan struct{}

	mu sync.Mutex
}

func New(tr *i18n.Translator, cb Callbacks) *Tray {
	return &Tray{i18n: tr, cb: cb, tapCh: make(chan struct{}, 1)}
}

func (t *Tray) Run() {
	// Pin this goroutine to its OS thread for its entire life. systray.Run creates
	// the tray window and runs a Win32 message pump (GetMessage) on this goroutine,
	// and Win32 delivers a window's messages ONLY to the thread that created it.
	// This goroutine is not the main goroutine (fyne owns that for its GL context),
	// and systray's own init() only LockOSThreads goroutine-1 — so without this,
	// the Go scheduler eventually migrates the pump off the thread that created
	// the tray window. After that migration the pump blocks in GetMessage on a
	// thread that owns no window and never sees the tray's clicks: the icon stays
	// but the menu goes permanently dead (confirmed via minidump, 2026-07-07).
	// LockOSThread prevents the migration. It is never unlocked; when systray.Run
	// returns on quit this goroutine exits and the runtime reclaims the thread.
	runtime.LockOSThread()
	// Left-click opens the detail window; right-click keeps showing the menu.
	// Registered before Run so it is in place before the pump starts.
	systray.SetOnTapped(t.onTapped)
	systray.Run(t.onReady, t.onExit)
}

func (t *Tray) onReady() {
	systray.SetIcon(IconFor(core.StatusUnknown))
	systray.SetTooltip(t.i18n.T("tray.tooltip"))

	mDetail := systray.AddMenuItem(t.i18n.T("tray.menu.show_details"), "")
	mSettings := systray.AddMenuItem(t.i18n.T("tray.menu.settings"), "")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem(t.i18n.T("tray.menu.quit"), "")

	go t.handleEvents(mDetail.ClickedCh, mSettings.ClickedCh, mQuit.ClickedCh, systray.Quit)
}

// onTapped handles a left-click on the tray icon. It runs synchronously inside
// the tray's window procedure on the message-pump thread, so it must never
// block — a stalled pump is exactly what kills the tray menu. It only records a
// pending request (repeats coalesce) and handleEvents opens the window.
func (t *Tray) onTapped() {
	select {
	case t.tapCh <- struct{}{}:
	default: // a request is already pending
	}
}

// handleEvents runs the callbacks for all tray clicks on this one goroutine.
func (t *Tray) handleEvents(detail, settings, quit <-chan struct{}, quitTray func()) {
	for {
		select {
		case <-t.tapCh:
			if t.cb.OnDetail != nil {
				t.cb.OnDetail()
			}
		case <-detail:
			if t.cb.OnDetail != nil {
				t.cb.OnDetail()
			}
		case <-settings:
			if t.cb.OnSettings != nil {
				t.cb.OnSettings()
			}
		case <-quit:
			quitTray()
			return
		}
	}
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
