package settings

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/yosh3289/speedforce/internal/config"
	"github.com/yosh3289/speedforce/internal/i18n"
)

type SaveFunc func(*config.Config) error
type ExportLogsFunc func() (path string, err error)

type Window struct {
	app      fyne.App
	i18n     *i18n.Translator
	cfg      *config.Config
	onSave   SaveFunc
	onExport ExportLogsFunc

	mu  sync.Mutex
	win fyne.Window
}

func New(app fyne.App, tr *i18n.Translator, cfg *config.Config, save SaveFunc, export ExportLogsFunc) *Window {
	return &Window{app: app, i18n: tr, cfg: cfg, onSave: save, onExport: export}
}

func (w *Window) Show() {
	fyne.Do(w.show)
}

func (w *Window) show() {
	w.mu.Lock()
	if w.win != nil {
		w.win.Show()
		w.win.RequestFocus()
		w.mu.Unlock()
		return
	}
	w.win = w.app.NewWindow(w.i18n.T("settings.title"))
	w.win.Resize(fyne.NewSize(420, 480))

	langSel := widget.NewSelect([]string{"zh", "en"}, nil)
	langSel.Selected = w.cfg.Language

	proxyMode := widget.NewSelect([]string{"auto", "manual", "none"}, nil)
	proxyMode.Selected = w.cfg.Network.Proxy.Mode

	proxyURL := widget.NewEntry()
	proxyURL.SetText(w.cfg.Network.Proxy.ManualURL)

	tickEntry := widget.NewEntry()
	tickEntry.SetText(intToStr(w.cfg.Network.TickInterval))

	autostart := widget.NewCheck("", nil)
	autostart.Checked = w.cfg.UI.AutoStart

	notifCheckboxes := make([]*widget.Check, len(w.cfg.Probes.HTTPS))
	notifBox := container.NewVBox()
	for i, p := range w.cfg.Probes.HTTPS {
		c := widget.NewCheck(p.Name, nil)
		c.Checked = p.NotifyOnDown
		notifCheckboxes[i] = c
		notifBox.Add(c)
	}

	exportBtn := widget.NewButton(w.i18n.T("settings.export_logs"), func() {
		if w.onExport == nil {
			return
		}
		if path, err := w.onExport(); err == nil {
			dialogInfo(w.app, "Exported to "+path)
		} else {
			dialogInfo(w.app, "Export failed: "+err.Error())
		}
	})

	saveBtn := widget.NewButton(w.i18n.T("settings.save"), func() {
		w.cfg.Language = langSel.Selected
		w.cfg.Network.Proxy.Mode = proxyMode.Selected
		w.cfg.Network.Proxy.ManualURL = proxyURL.Text
		w.cfg.Network.TickInterval = strToInt(tickEntry.Text, w.cfg.Network.TickInterval)
		w.cfg.UI.AutoStart = autostart.Checked
		for i, c := range notifCheckboxes {
			w.cfg.Probes.HTTPS[i].NotifyOnDown = c.Checked
		}
		if err := w.onSave(w.cfg); err != nil {
			dialogInfo(w.app, "Save failed: "+err.Error())
			return
		}
		w.close()
	})

	cancelBtn := widget.NewButton(w.i18n.T("settings.cancel"), w.close)

	form := container.NewVBox(
		labelRow(w.i18n.T("settings.language"), langSel),
		labelRow(w.i18n.T("settings.proxy.mode"), proxyMode),
		labelRow(w.i18n.T("settings.proxy.url"), proxyURL),
		labelRow(w.i18n.T("settings.tick_interval"), tickEntry),
		labelRow(w.i18n.T("settings.autostart"), autostart),
		widget.NewSeparator(),
		widget.NewLabel(w.i18n.T("settings.notifications")),
		notifBox,
		widget.NewSeparator(),
		exportBtn,
		widget.NewSeparator(),
		container.NewHBox(saveBtn, cancelBtn),
	)
	w.win.SetContent(form)
	w.win.Show()
	w.mu.Unlock()
}

func (w *Window) close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.win != nil {
		w.win.Close()
		w.win = nil
	}
}

func labelRow(label string, input fyne.CanvasObject) fyne.CanvasObject {
	return container.NewBorder(nil, nil, widget.NewLabel(label), nil, input)
}

func intToStr(i int) string { return fmtInt(i) }

func strToInt(s string, fallback int) int {
	v, err := parseInt(s)
	if err != nil {
		return fallback
	}
	return v
}

func dialogInfo(app fyne.App, msg string) {
	win := app.NewWindow("Info")
	win.SetContent(widget.NewLabel(msg))
	win.Resize(fyne.NewSize(300, 100))
	win.Show()
}
