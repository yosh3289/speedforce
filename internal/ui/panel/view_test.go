package panel

import (
	"errors"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/yosh3289/speedforce/internal/core"
)

// The panel has a button per provider that opens its official status page,
// labelled in every supported language.
func TestStatusButtonsOpenOfficialStatusPages(t *testing.T) {
	want := []struct{ labelKey, url string }{
		{"panel.button.open_claude_status", "https://status.claude.com"},
		{"panel.button.open_openai_status", "https://status.openai.com"},
		{"panel.button.open_gemini_status", "https://aistudio.google.com/status"},
	}
	for _, locale := range []string{"en", "zh"} {
		p, a, _ := newTestPanel(t, locale)
		p.toggle()

		for _, s := range want {
			label := p.i18n.T(s.labelKey)
			if label == s.labelKey {
				t.Errorf("%s: %s has no translation", locale, s.labelKey)
				continue
			}
			btn := findButton(p.win.Content(), label)
			if btn == nil {
				t.Errorf("%s: no %q button", locale, label)
				continue
			}
			a.opened = nil
			test.Tap(btn)
			if len(a.opened) != 1 || a.opened[0] != s.url {
				t.Errorf("%s: %q opened %v, want [%s]", locale, label, a.opened, s.url)
			}
		}
	}
}

func TestHeaderShowsOnlineCount(t *testing.T) {
	p, _, _ := newTestPanel(t, "zh")
	p.state = core.State{HTTPS: []core.ProbeResult{
		{Name: "Claude API", StatusCode: 404, LatencyMs: 700},
		{Name: "OpenAI API", StatusCode: 421, LatencyMs: 600},
		{Name: "Gemini API", Err: errors.New("timeout")},
	}}
	p.toggle()

	if findLabel(p.win.Content(), "2/3 在线") == nil {
		t.Error(`header does not show "2/3 在线"`)
	}
}

func TestFailedProbeShowsFailed(t *testing.T) {
	p, _, _ := newTestPanel(t, "en")
	p.state = core.State{HTTPS: []core.ProbeResult{{Name: "Gemini API", Err: errors.New("timeout")}}}
	p.toggle()

	if findLabel(p.win.Content(), "Failed") == nil {
		t.Error(`failed probe row does not say "Failed"`)
	}
}

func TestLatencyBar(t *testing.T) {
	tests := []struct {
		name string
		p    core.ProbeResult
		frac float32
		col  any
	}{
		{"fast", core.ProbeResult{StatusCode: 200, LatencyMs: 0}, 0, barBlue},
		{"half scale", core.ProbeResult{StatusCode: 200, LatencyMs: 1500}, 0.5, barBlue},
		{"at scale", core.ProbeResult{StatusCode: 200, LatencyMs: 3000}, 1, barBlue},
		{"slow: clamped, yellow", core.ProbeResult{StatusCode: 200, LatencyMs: 5000}, 1, barYellow},
		{"failed: full, red", core.ProbeResult{Err: errors.New("x")}, 1, barRed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frac, col := latencyBar(tt.p)
			if frac != tt.frac || col != tt.col {
				t.Errorf("got (%v, %v), want (%v, %v)", frac, col, tt.frac, tt.col)
			}
		})
	}
}

func TestNetworkDetailsExpand(t *testing.T) {
	p, _, _ := newTestPanel(t, "en")
	p.state = core.State{IP: core.IPInfo{PublicIP: "1.2.3.4", LANIP: "192.168.1.5", Country: "US"}}
	p.toggle()
	if findLabel(p.win.Content(), "192.168.1.5") != nil {
		t.Fatal("network details visible before expanding")
	}

	test.Tap(findButton(p.win.Content(), "▸ "+p.i18n.T("panel.network_details")))

	if findLabel(p.win.Content(), "192.168.1.5") == nil {
		t.Fatal("expanding network details does not show the LAN IP")
	}
}

// Expanding or collapsing the details must not resize the panel: it is docked
// against the taskbar, so a height change would move its top edge.
func TestExpandingDetailsKeepsPanelSize(t *testing.T) {
	p, _, _ := newTestPanel(t, "en")
	p.state = core.State{
		HTTPS: []core.ProbeResult{{Name: "Claude API", StatusCode: 200, LatencyMs: 700}},
		IP:    core.IPInfo{PublicIP: "1.2.3.4", LANIP: "192.168.1.5", City: "Riverside", Country: "US", ISP: "Some ISP"},
	}
	p.toggle()
	closed := p.win.Canvas().Size()

	test.Tap(findButton(p.win.Content(), "▸ "+p.i18n.T("panel.network_details")))
	if got := p.win.Canvas().Size(); got != closed {
		t.Fatalf("expanding details resized the panel: %v -> %v", closed, got)
	}
	test.Tap(findButton(p.win.Content(), "▾ "+p.i18n.T("panel.network_details")))
	if got := p.win.Canvas().Size(); got != closed {
		t.Fatalf("collapsing details resized the panel: %v -> %v", closed, got)
	}
}

// Only opening the panel may place it at the mouse cursor. Re-docking after a
// re-render (expanding the details) must keep it where it is; placing it at
// the cursor — which is over the details button — slid it sideways each time.
func TestTogglingDetailsNeverMovesPanelToCursor(t *testing.T) {
	p, _, _ := newTestPanel(t, "en")
	atCursor := 0
	p.placeAtCursor = func() { atCursor++ }
	p.redock = func() {}
	p.toggle()

	test.Tap(findButton(p.win.Content(), "▸ "+p.i18n.T("panel.network_details")))
	test.Tap(findButton(p.win.Content(), "▾ "+p.i18n.T("panel.network_details")))

	if atCursor != 1 {
		t.Fatalf("panel was placed at the cursor %d times, want once (on open)", atCursor)
	}
}

// walk visits every canvas object in the tree, including inside theme overrides.
func walk(o fyne.CanvasObject, visit func(fyne.CanvasObject)) {
	visit(o)
	switch v := o.(type) {
	case *fyne.Container:
		for _, c := range v.Objects {
			walk(c, visit)
		}
	case *container.ThemeOverride:
		walk(v.Content, visit)
	case *container.Scroll:
		walk(v.Content, visit)
	}
}

func findButton(root fyne.CanvasObject, text string) *widget.Button {
	var found *widget.Button
	walk(root, func(o fyne.CanvasObject) {
		if b, ok := o.(*widget.Button); ok && found == nil && b.Text == text {
			found = b
		}
	})
	return found
}

func findLabel(root fyne.CanvasObject, substr string) *widget.Label {
	var found *widget.Label
	walk(root, func(o fyne.CanvasObject) {
		if l, ok := o.(*widget.Label); ok && found == nil && strings.Contains(l.Text, substr) {
			found = l
		}
	})
	return found
}
