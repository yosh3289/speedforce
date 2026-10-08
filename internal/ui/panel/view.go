package panel

import (
	"fmt"
	"image/color"
	"net/url"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/yosh3289/speedforce/internal/core"
)

var (
	barBlue   = color.NRGBA{R: 41, G: 121, B: 255, A: 255}
	barYellow = color.NRGBA{R: 255, G: 196, B: 0, A: 255}
	barRed    = color.NRGBA{R: 213, G: 0, B: 0, A: 255}
	barTrack  = color.NRGBA{R: 255, G: 255, B: 255, A: 0x26}
	dotGray   = color.NRGBA{R: 158, G: 158, B: 158, A: 255}
)

// latencyScaleMs is the latency that fills the bar; slower probes are also
// shown in yellow (the detail window's old "slow" threshold).
const latencyScaleMs = 3000

// statusPages are the providers' official status pages, opened in the browser
// from the buttons at the bottom of the panel.
var statusPages = []struct {
	labelKey string
	url      string
}{
	{"panel.button.open_claude_status", "https://status.claude.com"},
	{"panel.button.open_openai_status", "https://status.openai.com"},
	{"panel.button.open_gemini_status", "https://aistudio.google.com/status"},
}

// latencyBar returns how full a probe's bar is (0..1) and its colour: failed
// probes are a full red bar, slow ones yellow, the rest blue.
func latencyBar(p core.ProbeResult) (float32, color.Color) {
	if !p.IsUp() {
		return 1, barRed
	}
	frac := float32(p.LatencyMs) / latencyScaleMs
	if frac > 1 {
		frac = 1
	}
	if p.LatencyMs > latencyScaleMs {
		return frac, barYellow
	}
	return frac, barBlue
}

// buildContent builds the whole panel from the current state and returns it
// with the size the window should have. The footer is pinned; everything above
// it scrolls. The height is always that of the body with the network details
// collapsed, so expanding them scrolls inside the panel instead of growing it
// (a docked panel that grows moves its top edge).
func (p *Panel) buildContent() (fyne.CanvasObject, fyne.Size) {
	th := newDarkTheme()
	body := p.body(p.detailsOpen)
	footer := p.footer()
	p.scroll = container.NewVScroll(body)
	card := container.NewStack(canvas.NewRectangle(panelBackground),
		container.NewPadded(container.NewBorder(nil, footer, nil, nil, p.scroll)))
	content := container.NewThemeOverride(card, th)

	natural := body
	if p.detailsOpen {
		natural = p.body(false)
		container.NewThemeOverride(natural, th) // measure it with the panel theme
	}
	pad := th.Size(theme.SizeNamePadding)
	bodyMin, footMin := natural.MinSize(), footer.MinSize()
	w := fyne.Max(panelWidth, fyne.Max(bodyMin.Width, footMin.Width)+2*pad)
	h := bodyMin.Height + footMin.Height + 3*pad // padding above, between, below
	return content, fyne.NewSize(w, h)
}

func (p *Panel) body(detailsOpen bool) *fyne.Container {
	s := p.state
	return container.NewVBox(
		p.header(s),
		widget.NewSeparator(),
		sectionTitle(p.i18n.T("panel.section.probes")),
		p.probeRows(s),
		widget.NewSeparator(),
		sectionTitle(p.i18n.T("panel.section.statuspage")),
		p.statuspageRows(s),
		widget.NewSeparator(),
		p.networkDetails(s, detailsOpen),
	)
}

func (p *Panel) header(s core.State) fyne.CanvasObject {
	up := 0
	for _, r := range s.HTTPS {
		if r.IsUp() {
			up++
		}
	}
	online := p.i18n.T("panel.online", map[string]string{
		"up": strconv.Itoa(up), "total": strconv.Itoa(len(s.HTTPS)),
	})
	title := widget.NewLabelWithStyle("SpeedForce  "+online, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	mode := p.i18n.T("panel.mode.normal")
	if s.Mode == core.TickFast {
		mode = p.i18n.T("panel.mode.fast")
	}
	ip := p.i18n.T("panel.loading")
	if s.IP.PublicIP != "" {
		ip = s.IP.PublicIP
		if s.IP.Country != "" {
			ip += " · " + s.IP.Country
		}
	}
	return container.NewVBox(
		container.NewBorder(nil, nil, nil, widget.NewLabel(mode), title),
		secondaryLabel(ip),
	)
}

func (p *Panel) probeRows(s core.State) fyne.CanvasObject {
	if len(s.HTTPS) == 0 {
		return widget.NewLabel(p.i18n.T("panel.loading"))
	}
	rows := container.New(layout.NewFormLayout())
	for _, r := range s.HTTPS {
		frac, col := latencyBar(r)
		text := fmt.Sprintf("%d ms", r.LatencyMs)
		if !r.IsUp() {
			text = p.i18n.T("panel.failed")
		}
		ms := widget.NewLabel(text)
		ms.Alignment = fyne.TextAlignTrailing
		msCell := container.NewGridWrap(fyne.NewSize(76, ms.MinSize().Height), ms)
		rows.Add(widget.NewLabel(r.Name))
		rows.Add(container.NewBorder(nil, nil, nil, msCell, newBar(frac, col)))
	}
	return rows
}

func (p *Panel) statuspageRows(s core.State) fyne.CanvasObject {
	if len(s.Statuspage) == 0 {
		return widget.NewLabel(p.i18n.T("panel.loading"))
	}
	box := container.NewVBox()
	for _, sp := range s.Statuspage {
		c := color.Color(barBlue)
		switch sp.Indicator {
		case core.StatuspageMinor, core.StatuspageMaintenance:
			c = barYellow
		case core.StatuspageMajor, core.StatuspageCritical:
			c = barRed
		}
		text := sp.Name + " — " + sp.Description
		if sp.Err != nil {
			c = dotGray
			text = sp.Name + " — " + p.i18n.T("panel.unavailable")
		}
		box.Add(dotRow(c, text))
	}
	return box
}

// networkDetails is a collapsible section (like codexbar's "Usage details")
// with what the compact rows leave out.
func (p *Panel) networkDetails(s core.State, open bool) fyne.CanvasObject {
	arrow := "▸ "
	if open {
		arrow = "▾ "
	}
	toggle := widget.NewButton(arrow+p.i18n.T("panel.network_details"), func() {
		p.detailsOpen = !p.detailsOpen
		p.scrollToEnd = p.detailsOpen // bring the newly shown details into view
		p.render()
	})
	toggle.Importance = widget.LowImportance
	toggle.Alignment = widget.ButtonAlignLeading
	if !open {
		return toggle
	}

	lines := []string{p.i18n.T("panel.lan") + ": " + orDash(s.IP.LANIP)}
	var place []string
	for _, v := range []string{s.IP.City, s.IP.Country} {
		if v != "" {
			place = append(place, v)
		}
	}
	location := orDash(strings.Join(place, ", "))
	if s.IP.ISP != "" {
		location += " · " + s.IP.ISP
	}
	lines = append(lines, p.i18n.T("panel.location")+": "+location)
	for _, r := range s.HTTPS {
		code := "HTTP " + strconv.Itoa(r.StatusCode)
		if r.Err != nil {
			code = p.i18n.T("panel.failed")
		}
		lines = append(lines, r.Name+" — "+code)
	}
	box := container.NewVBox(toggle)
	for _, l := range lines {
		box.Add(secondaryLabel(l))
	}
	return box
}

// secondaryLabel is dimmer, word-wrapped text, so long values (an ISP name)
// wrap instead of widening the panel.
func secondaryLabel(text string) *widget.Label {
	l := widget.NewLabel(text)
	l.Importance = widget.LowImportance
	l.Wrapping = fyne.TextWrapWord
	return l
}

func (p *Panel) footer() fyne.CanvasObject {
	buttons := container.NewGridWithColumns(len(statusPages))
	for _, sp := range statusPages {
		u, _ := url.Parse(sp.url)
		buttons.Add(widget.NewButton(p.i18n.T(sp.labelKey), func() {
			_ = p.app.OpenURL(u)
		}))
	}
	settings := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		if p.onSettings != nil {
			p.onSettings()
		}
	})
	return container.NewBorder(nil, nil, nil, settings, buttons)
}

func sectionTitle(text string) fyne.CanvasObject {
	return widget.NewLabelWithStyle(text, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
}

func dotRow(c color.Color, text string) fyne.CanvasObject {
	dot := container.NewGridWrap(fyne.NewSize(10, 10), canvas.NewCircle(c))
	return container.NewBorder(nil, nil, container.NewCenter(dot), nil, widget.NewLabel(text))
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// newBar is a thin horizontal bar: a dim track with the first frac of its
// width filled in col, vertically centred in its row.
func newBar(frac float32, col color.Color) fyne.CanvasObject {
	return container.New(barLayout{frac: frac}, canvas.NewRectangle(barTrack), canvas.NewRectangle(col))
}

type barLayout struct{ frac float32 }

const barHeight = 6

func (l barLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	y := (size.Height - barHeight) / 2
	objs[0].Move(fyne.NewPos(0, y))
	objs[0].Resize(fyne.NewSize(size.Width, barHeight))
	objs[1].Move(fyne.NewPos(0, y))
	objs[1].Resize(fyne.NewSize(size.Width*l.frac, barHeight))
}

func (l barLayout) MinSize([]fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(60, barHeight)
}
