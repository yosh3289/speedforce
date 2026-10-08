// Package panel is the tray flyout: a borderless dark card that opens next to
// the taskbar on a left-click of the tray icon and hides when it loses focus.
package panel

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/yosh3289/speedforce/internal/core"
	"github.com/yosh3289/speedforce/internal/i18n"
	"github.com/yosh3289/speedforce/internal/platform"
)

const (
	// panelWidth is the flyout width in fyne units; it grows if the content
	// needs more.
	panelWidth = 340
	// tapDebounce: clicking the tray icon while the panel is open first takes
	// focus away from the panel (hiding it on mouse down), then delivers the
	// tap on mouse up. A tap this soon after such a hide is that same click and
	// must not reopen the panel.
	tapDebounce = 300 * time.Millisecond
	// windowTitle is never displayed (the window has no decorations) but
	// identifies the window to the OS and to diagnostics.
	windowTitle = "SpeedForce Panel"
)

// Panel is the tray flyout. Its exported methods may be called from any
// goroutine; everything else runs on the fyne main thread, which is why the
// fields below need no lock.
type Panel struct {
	app        fyne.App
	i18n       *i18n.Translator
	onSettings func()
	now        func() time.Time

	win         fyne.Window
	visible     bool
	styled      bool
	styling     bool // StyleFlyout re-shows the window; ignore the focus loss that causes
	detailsOpen bool
	scroll      *container.Scroll // the scrolling body of the current content
	scrollToEnd bool              // scroll to the bottom on the next render

	// placeAtCursor docks the window next to the taskbar at the mouse cursor
	// (on open); redock re-docks it around its current position (after a
	// re-render). Fields so tests can observe which one runs.
	placeAtCursor func()
	redock        func()
	state         core.State
	lastAutoHide  time.Time
}

// New creates the panel and keeps it updated from bus. The window itself is
// created on first show and then reused: closing only hides it.
func New(app fyne.App, tr *i18n.Translator, bus *core.StateBus, onSettings func()) *Panel {
	p := newPanel(app, tr, onSettings)
	go p.subscribe(bus)
	return p
}

func newPanel(app fyne.App, tr *i18n.Translator, onSettings func()) *Panel {
	p := &Panel{app: app, i18n: tr, onSettings: onSettings, now: time.Now}
	p.placeAtCursor = func() { p.native(platform.PositionFlyout) }
	p.redock = func() { p.native(platform.RedockFlyout) }
	return p
}

// Toggle opens the panel, or closes it if it is open (tray left-click).
func (p *Panel) Toggle() { fyne.Do(p.toggle) }

// Show opens the panel (tray menu "Show Details").
func (p *Panel) Show() { fyne.Do(p.show) }

// FocusLost hides the panel. It is wired to the app's
// Lifecycle.SetOnExitedForeground, so it already runs on the main thread.
func (p *Panel) FocusLost() { p.focusLost() }

func (p *Panel) subscribe(bus *core.StateBus) {
	sub := bus.Subscribe()
	apply := func(s core.State) {
		fyne.Do(func() {
			p.state = s
			if p.visible {
				p.render()
			}
		})
	}
	apply(bus.Snapshot())
	for s := range sub {
		apply(s)
	}
}

func (p *Panel) toggle() {
	if p.visible {
		p.hide()
		return
	}
	if p.now().Sub(p.lastAutoHide) < tapDebounce {
		return
	}
	p.show()
}

func (p *Panel) focusLost() {
	if !p.visible || p.styling {
		return
	}
	p.hide()
	p.lastAutoHide = p.now()
}

func (p *Panel) show() {
	if p.win == nil {
		p.win = p.newWindow()
	}
	p.visible = true
	p.render()
	p.win.Show()
	p.win.RequestFocus()
	p.native(func(hwnd uintptr) {
		if !p.styled {
			// StyleFlyout hides and re-shows the native window (so the taskbar
			// drops its button); the hide fires a focus loss synchronously,
			// which must not hide the panel we are opening.
			p.styling = true
			platform.StyleFlyout(hwnd)
			p.styling = false
			p.styled = true
		}
	})
	p.placeAtCursor()
}

func (p *Panel) hide() {
	p.visible = false
	if p.win != nil {
		p.win.Hide()
	}
}

func (p *Panel) newWindow() fyne.Window {
	var w fyne.Window
	if drv, ok := p.app.Driver().(desktop.Driver); ok {
		w = drv.CreateSplashWindow() // borderless
	} else {
		w = p.app.NewWindow(windowTitle)
	}
	w.SetTitle(windowTitle)
	w.SetCloseIntercept(p.hide) // e.g. Alt+F4: hide, never destroy
	w.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		if e.Name == fyne.KeyEscape {
			p.hide()
		}
	})
	return w
}

// render rebuilds the content from the current state and sizes the window.
// The scroll position survives the rebuild (state updates arrive every few
// seconds). While the panel is open it is re-docked around its current
// position, so a size change (e.g. a new probe row) keeps it against the
// taskbar without sliding it towards the mouse cursor.
func (p *Panel) render() {
	var offset fyne.Position
	if p.scroll != nil {
		offset = p.scroll.Offset
	}
	content, size := p.buildContent()
	p.win.SetContent(content)
	p.win.Resize(size)
	if p.scrollToEnd {
		p.scroll.ScrollToBottom()
		p.scrollToEnd = false
	} else {
		p.scroll.Offset = offset
		p.scroll.Refresh()
	}
	if p.visible {
		p.redock()
	}
}

// native runs f with the window's Win32 handle, if there is one.
func (p *Panel) native(f func(hwnd uintptr)) {
	nw, ok := p.win.(driver.NativeWindow)
	if !ok {
		return
	}
	nw.RunNative(func(ctx any) {
		if c, ok := ctx.(driver.WindowsWindowContext); ok && c.HWND != 0 {
			f(c.HWND)
		}
	})
}
