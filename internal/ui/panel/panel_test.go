package panel

import (
	"net/url"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/yosh3289/speedforce/internal/i18n"
)

// recordingApp is a test app that records the URLs it is asked to open.
type recordingApp struct {
	fyne.App
	opened []string
}

func (a *recordingApp) OpenURL(u *url.URL) error {
	a.opened = append(a.opened, u.String())
	return nil
}

// fakeClock is a manually advanced clock for the tap debounce.
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time          { return c.t }
func (c *fakeClock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newTestPanel(t *testing.T, locale string) (*Panel, *recordingApp, *fakeClock) {
	t.Helper()
	tr, err := i18n.New(locale)
	if err != nil {
		t.Fatal(err)
	}
	a := &recordingApp{App: test.NewApp()}
	clk := &fakeClock{t: time.Unix(1_000_000, 0)}
	p := newPanel(a, tr, nil)
	p.now = clk.now
	return p, a, clk
}

func TestTapOpensThenClosesPanel(t *testing.T) {
	p, _, _ := newTestPanel(t, "en")

	p.toggle()
	if !p.visible {
		t.Fatal("first tap did not open the panel")
	}
	p.toggle()
	if p.visible {
		t.Fatal("second tap did not close the panel")
	}
}

// Clicking the tray icon while the panel is open first takes focus away from
// the panel (mouse down hides it), then delivers the tap (mouse up). That tap
// must not immediately reopen the panel.
func TestTapRightAfterFocusLossDoesNotReopen(t *testing.T) {
	p, _, clk := newTestPanel(t, "en")
	p.toggle()

	p.focusLost()
	if p.visible {
		t.Fatal("losing focus did not hide the panel")
	}
	clk.advance(100 * time.Millisecond)
	p.toggle()
	if p.visible {
		t.Fatal("tap right after focus loss reopened the panel")
	}
	clk.advance(400 * time.Millisecond)
	p.toggle()
	if !p.visible {
		t.Fatal("a later tap did not open the panel")
	}
}

// Restyling the window on first show hides and re-shows it natively, which
// fires a focus loss mid-show; that must not close the panel being opened.
func TestFocusLossWhileStylingIsIgnored(t *testing.T) {
	p, _, _ := newTestPanel(t, "en")
	p.toggle()

	p.styling = true
	p.focusLost()
	p.styling = false

	if !p.visible {
		t.Fatal("focus loss caused by restyling hid the panel")
	}
}

func TestEscapeHidesPanel(t *testing.T) {
	p, _, _ := newTestPanel(t, "en")
	p.toggle()

	p.win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyEscape})

	if p.visible {
		t.Fatal("Escape did not hide the panel")
	}
}
