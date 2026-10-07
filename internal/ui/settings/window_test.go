package settings

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/yosh3289/speedforce/internal/config"
	"github.com/yosh3289/speedforce/internal/i18n"
)

// Settings must open again after its window was closed. fyne silently ignores
// Show on a closed window, so holding on to it left Settings unopenable until
// the app restarted.
func TestSettingsReopensAfterWindowClosed(t *testing.T) {
	a := test.NewApp()
	tr, err := i18n.New("zh")
	if err != nil {
		t.Fatal(err)
	}
	s := New(a, tr, config.Default(), func(*config.Config) error { return nil }, nil)

	s.show()
	s.win.Close()
	s.show()

	if s.win == nil || !isOpen(a.Driver().AllWindows(), s.win) {
		t.Fatal("settings window did not reopen after being closed")
	}
}

func isOpen(open []fyne.Window, w fyne.Window) bool {
	for _, o := range open {
		if o == w {
			return true
		}
	}
	return false
}
