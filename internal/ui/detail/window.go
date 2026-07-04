package detail

import (
	"fmt"
	"image/color"
	"net/url"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/yosh3289/speedforce/internal/core"
	"github.com/yosh3289/speedforce/internal/i18n"
)

var (
	dotBlue   = color.NRGBA{R: 41, G: 121, B: 255, A: 255}
	dotYellow = color.NRGBA{R: 255, G: 196, B: 0, A: 255}
	dotRed    = color.NRGBA{R: 213, G: 0, B: 0, A: 255}
	dotGray   = color.NRGBA{R: 158, G: 158, B: 158, A: 255}
)

func dotRow(c color.Color, text string) fyne.CanvasObject {
	circle := canvas.NewCircle(c)
	dotCell := container.NewGridWrap(fyne.NewSize(14, 14), circle)
	// NewCenter inside the left region of a Border layout vertically
	// centers the dot against the label's natural line height.
	return container.NewBorder(nil, nil, container.NewCenter(dotCell), nil, widget.NewLabel(text))
}

type Window struct {
	app        fyne.App
	i18n       *i18n.Translator
	bus        *core.StateBus
	onSettings func()

	mu          sync.Mutex
	win         fyne.Window
	visible     bool
	builtLocale string
	modeLbl     *widget.Label
	ipLbl       *widget.Label
	probesBox   *fyne.Container
	spBox       *fyne.Container
}

func New(app fyne.App, tr *i18n.Translator, bus *core.StateBus, onSettings func()) *Window {
	w := &Window{app: app, i18n: tr, bus: bus, onSettings: onSettings}
	go w.subscribe()
	return w
}

// Show creates the window on first use and reuses it thereafter. Closing the
// window (the X button) hides it rather than destroying it, so the underlying
// GLFW window and its OpenGL context are reused on the next open.
//
// Previously every open called NewWindow and every close destroyed it; on
// Windows that leaked ~2 MB of native (GL/cgo) memory per open/close cycle
// because the driver does not fully release the destroyed context. Reusing a
// single hidden window avoids the churn entirely.
func (w *Window) Show() {
	fyne.Do(func() {
		w.mu.Lock()
		if w.win == nil {
			w.win = w.app.NewWindow(w.i18n.T("detail.title"))
			w.win.Resize(fyne.NewSize(420, 520))
			w.win.SetCloseIntercept(func() {
				w.mu.Lock()
				w.visible = false
				w.mu.Unlock()
				w.win.Hide()
			})
			w.buildContentLocked()
		} else if w.builtLocale != w.i18n.Locale() {
			// Language switched at runtime: rebuild content in place. No new
			// window is created, so there is no GL-context churn.
			w.buildContentLocked()
		}
		w.visible = true
		w.win.Show()
		w.win.RequestFocus()
		snap := w.bus.Snapshot()
		w.mu.Unlock()

		// Populate the (possibly freshly built) widgets immediately.
		go w.update(snap)
	})
}

// buildContentLocked builds the widgets and sets window content. Caller holds mu.
func (w *Window) buildContentLocked() {
	w.win.SetTitle(w.i18n.T("detail.title"))
	w.modeLbl = widget.NewLabel(w.i18n.T("detail.mode.normal"))
	w.ipLbl = widget.NewLabel("...")
	w.probesBox = container.NewVBox()
	w.spBox = container.NewVBox()

	geminiBtn := widget.NewButton(w.i18n.T("detail.button.open_gemini_status"), func() {
		u, _ := url.Parse("https://aistudio.google.com/status")
		_ = w.app.OpenURL(u)
	})

	settingsBtn := widget.NewButton(w.i18n.T("detail.button.settings"), func() {
		if w.onSettings != nil {
			w.onSettings()
		}
	})

	content := container.NewVBox(
		container.NewHBox(widget.NewLabel(w.i18n.T("app_name")+" — "), w.modeLbl),
		widget.NewSeparator(),
		widget.NewLabel(w.i18n.T("detail.section.ip")),
		w.ipLbl,
		widget.NewSeparator(),
		widget.NewLabel(w.i18n.T("detail.section.probes")),
		w.probesBox,
		widget.NewSeparator(),
		widget.NewLabel(w.i18n.T("detail.section.statuspage")),
		w.spBox,
		widget.NewSeparator(),
		container.NewHBox(geminiBtn, settingsBtn),
	)
	w.win.SetContent(content)
	w.builtLocale = w.i18n.Locale()
}

func (w *Window) subscribe() {
	sub := w.bus.Subscribe()
	w.update(w.bus.Snapshot())
	for s := range sub {
		w.update(s)
	}
}

func (w *Window) update(s core.State) {
	fyne.Do(func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		if w.win == nil || !w.visible {
			return
		}
		if s.Mode == core.TickFast {
			w.modeLbl.SetText(w.i18n.T("detail.mode.fast"))
		} else {
			w.modeLbl.SetText(w.i18n.T("detail.mode.normal"))
		}
		ipText := "loading..."
		if s.IP.PublicIP != "" {
			geo := "loading geo..."
			if s.IP.Country != "" {
				geo = fmt.Sprintf("%s / %s / %s", s.IP.Country, s.IP.City, s.IP.ISP)
			}
			ipText = fmt.Sprintf("Public: %s (%s)\nLAN: %s", s.IP.PublicIP, geo, s.IP.LANIP)
		}
		w.ipLbl.SetText(ipText)

		w.probesBox.Objects = nil
		for _, p := range s.HTTPS {
			c := dotRed
			if p.IsUp() {
				if p.LatencyMs > 3000 {
					c = dotYellow
				} else {
					c = dotBlue
				}
			}
			text := fmt.Sprintf("%s — %d ms (HTTP %d)", p.Name, p.LatencyMs, p.StatusCode)
			w.probesBox.Add(dotRow(c, text))
		}
		w.probesBox.Refresh()

		w.spBox.Objects = nil
		for _, sp := range s.Statuspage {
			c := dotBlue
			switch sp.Indicator {
			case core.StatuspageMinor, core.StatuspageMaintenance:
				c = dotYellow
			case core.StatuspageMajor, core.StatuspageCritical:
				c = dotRed
			}
			text := fmt.Sprintf("%s — %s", sp.Name, sp.Description)
			if sp.Err != nil {
				c = dotGray
				text = fmt.Sprintf("%s — unavailable", sp.Name)
			}
			w.spBox.Add(dotRow(c, text))
		}
		w.spBox.Refresh()
	})
}

// Close destroys the window. Used only on app shutdown; normal user close is
// intercepted to Hide (see Show).
func (w *Window) Close() {
	w.mu.Lock()
	win := w.win
	w.win = nil
	w.visible = false
	w.mu.Unlock()
	if win != nil {
		win.Close()
	}
}
