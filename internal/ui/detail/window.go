package detail

import (
	"fmt"
	"net/url"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/yosh3289/speedforce/internal/core"
	"github.com/yosh3289/speedforce/internal/i18n"
)

type Window struct {
	app        fyne.App
	i18n       *i18n.Translator
	bus        *core.StateBus
	onSettings func()

	mu        sync.Mutex
	win       fyne.Window
	modeLbl   *widget.Label
	ipLbl     *widget.Label
	probesBox *fyne.Container
	spBox     *fyne.Container
}

func New(app fyne.App, tr *i18n.Translator, bus *core.StateBus, onSettings func()) *Window {
	return &Window{app: app, i18n: tr, bus: bus, onSettings: onSettings}
}

func (w *Window) Show() {
	w.mu.Lock()
	if w.win != nil {
		w.win.Show()
		w.win.RequestFocus()
		w.mu.Unlock()
		return
	}
	w.win = w.app.NewWindow(w.i18n.T("detail.title"))
	w.win.Resize(fyne.NewSize(420, 520))
	w.win.SetOnClosed(func() {
		w.mu.Lock()
		w.win = nil
		w.mu.Unlock()
	})

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
	w.win.Show()
	w.mu.Unlock()

	go w.subscribe()
}

func (w *Window) subscribe() {
	sub := w.bus.Subscribe()
	w.update(w.bus.Snapshot())
	for s := range sub {
		w.update(s)
	}
}

func (w *Window) update(s core.State) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.win == nil {
		return
	}
	if s.Mode == core.TickFast {
		w.modeLbl.SetText(w.i18n.T("detail.mode.fast"))
	} else {
		w.modeLbl.SetText(w.i18n.T("detail.mode.normal"))
	}
	w.ipLbl.SetText(fmt.Sprintf("%s (%s / %s / %s)  LAN: %s",
		s.IP.PublicIP, s.IP.Country, s.IP.City, s.IP.ISP, s.IP.LANIP))

	w.probesBox.Objects = nil
	for _, p := range s.HTTPS {
		status := "🔴"
		if p.IsUp() {
			if p.LatencyMs > 1000 {
				status = "🟡"
			} else {
				status = "🔵"
			}
		}
		line := widget.NewLabel(fmt.Sprintf("%s  %s — %d ms (HTTP %d)", status, p.Name, p.LatencyMs, p.StatusCode))
		w.probesBox.Add(line)
	}

	w.spBox.Objects = nil
	for _, sp := range s.Statuspage {
		badge := "🔵"
		switch sp.Indicator {
		case core.StatuspageMinor:
			badge = "🟡"
		case core.StatuspageMajor, core.StatuspageCritical:
			badge = "🔴"
		case core.StatuspageMaintenance:
			badge = "🟡"
		}
		text := fmt.Sprintf("%s  %s — %s", badge, sp.Name, sp.Description)
		if sp.Err != nil {
			text = fmt.Sprintf("⚪  %s — unavailable", sp.Name)
		}
		w.spBox.Add(widget.NewLabel(text))
	}
}

func (w *Window) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.win != nil {
		w.win.Close()
		w.win = nil
	}
}
