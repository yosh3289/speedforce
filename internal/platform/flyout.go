package platform

// Rect is a screen rectangle in physical pixels (right/bottom exclusive).
type Rect struct {
	Left, Top, Right, Bottom int
}

// PlaceFlyout returns the top-left position for a w×h flyout opened by a click
// at (cx, cy). The taskbar edge is the side where the monitor's work area is
// smaller than the monitor itself; the flyout docks against that edge, gap
// pixels inside the work area, centred on the click along the taskbar and
// clamped to the work area. An auto-hidden taskbar (work == monitor) is
// treated as being at the bottom.
func PlaceFlyout(cx, cy int, monitor, work Rect, w, h, gap int) (x, y int) {
	alongX := clamp(cx-w/2, work.Left+gap, work.Right-w-gap)
	alongY := clamp(cy-h/2, work.Top+gap, work.Bottom-h-gap)
	switch {
	case work.Top > monitor.Top:
		return alongX, work.Top + gap
	case work.Left > monitor.Left:
		return work.Left + gap, alongY
	case work.Right < monitor.Right:
		return work.Right - w - gap, alongY
	default: // bottom, or auto-hidden
		return alongX, work.Bottom - h - gap
	}
}

func clamp(v, lo, hi int) int {
	if v > hi {
		v = hi
	}
	if v < lo {
		v = lo
	}
	return v
}
