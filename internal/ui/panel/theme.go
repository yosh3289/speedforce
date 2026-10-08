package panel

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// panelBackground is the flyout's card colour (close to Windows' own dark
// flyouts).
var panelBackground = color.NRGBA{R: 0x20, G: 0x20, B: 0x22, A: 0xff}

// darkTheme renders the panel in the default theme's dark variant whatever the
// system setting is, so it reads as a flyout. It is applied only to the panel's
// content (container.NewThemeOverride); other windows keep the app theme.
type darkTheme struct{ fyne.Theme }

func newDarkTheme() fyne.Theme { return darkTheme{theme.DefaultTheme()} }

// secondaryText colours low-importance labels (IP line, network details). The
// dark variant's own "disabled" grey is too dim to read on the card; the panel
// has no disabled widgets, so that colour is repurposed.
var secondaryText = color.NRGBA{R: 0x9a, G: 0x9a, B: 0xa0, A: 0xff}

func (d darkTheme) Color(n fyne.ThemeColorName, _ fyne.ThemeVariant) color.Color {
	switch n {
	case theme.ColorNameBackground:
		return panelBackground
	case theme.ColorNameDisabled:
		return secondaryText
	}
	return d.Theme.Color(n, theme.VariantDark)
}

// Size tightens the inner padding so the rows sit closer, like a flyout.
func (d darkTheme) Size(n fyne.ThemeSizeName) float32 {
	if n == theme.SizeNameInnerPadding {
		return 5
	}
	return d.Theme.Size(n)
}
