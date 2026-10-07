package detail

import (
	"net/url"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

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

// The detail window has a button per provider that opens its official status
// page, labelled in every supported language.
func TestStatusButtonsOpenOfficialStatusPages(t *testing.T) {
	want := []struct{ labelKey, url string }{
		{"detail.button.open_claude_status", "https://status.claude.com"},
		{"detail.button.open_openai_status", "https://status.openai.com"},
		{"detail.button.open_gemini_status", "https://aistudio.google.com/status"},
	}
	for _, locale := range []string{"en", "zh"} {
		tr, err := i18n.New(locale)
		if err != nil {
			t.Fatal(err)
		}
		a := &recordingApp{App: test.NewApp()}
		w := &Window{app: a, i18n: tr}
		w.win = a.NewWindow("")
		w.buildContentLocked()

		for _, p := range want {
			label := tr.T(p.labelKey)
			if label == p.labelKey {
				t.Errorf("%s: %s has no translation", locale, p.labelKey)
				continue
			}
			btn := findButton(w.win.Content(), label)
			if btn == nil {
				t.Errorf("%s: no %q button", locale, label)
				continue
			}
			a.opened = nil
			test.Tap(btn)
			if len(a.opened) != 1 || a.opened[0] != p.url {
				t.Errorf("%s: %q opened %v, want [%s]", locale, label, a.opened, p.url)
			}
		}
	}
}

func findButton(o fyne.CanvasObject, text string) *widget.Button {
	switch v := o.(type) {
	case *widget.Button:
		if v.Text == text {
			return v
		}
	case *fyne.Container:
		for _, c := range v.Objects {
			if b := findButton(c, text); b != nil {
				return b
			}
		}
	}
	return nil
}
