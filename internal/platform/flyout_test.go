package platform

import "testing"

func TestPlaceFlyout(t *testing.T) {
	monitor := Rect{Left: 0, Top: 0, Right: 1920, Bottom: 1080}
	const w, h, gap = 340, 500, 8

	tests := []struct {
		name   string
		cx, cy int
		work   Rect
		x, y   int
	}{
		{"taskbar bottom: above taskbar, centred on click",
			1000, 1060, Rect{0, 0, 1920, 1032}, 1000 - w/2, 1032 - h - gap},
		{"taskbar bottom, click near right edge: clamped inside",
			1900, 1060, Rect{0, 0, 1920, 1032}, 1920 - w - gap, 1032 - h - gap},
		{"taskbar top: below taskbar",
			1000, 20, Rect{0, 48, 1920, 1080}, 1000 - w/2, 48 + gap},
		{"taskbar left: right of taskbar, centred vertically on click",
			20, 900, Rect{60, 0, 1920, 1080}, 60 + gap, 1080 - h - gap},
		{"taskbar right: left of taskbar",
			1900, 300, Rect{0, 0, 1860, 1080}, 1860 - w - gap, 300 - h/2},
		{"auto-hide taskbar (work == monitor): treated as bottom",
			1000, 1075, Rect{0, 0, 1920, 1080}, 1000 - w/2, 1080 - h - gap},
	}
	// Re-docking anchors at the flyout's own centre: that must leave a docked
	// flyout exactly where it is.
	for _, tt := range tests {
		t.Run(tt.name+" / redock in place", func(t *testing.T) {
			x, y := PlaceFlyout(tt.x+w/2, tt.y+h/2, monitor, tt.work, w, h, gap)
			if x != tt.x || y != tt.y {
				t.Errorf("redock moved it: (%d,%d) -> (%d,%d)", tt.x, tt.y, x, y)
			}
		})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x, y := PlaceFlyout(tt.cx, tt.cy, monitor, tt.work, w, h, gap)
			if x != tt.x || y != tt.y {
				t.Errorf("got (%d,%d), want (%d,%d)", x, y, tt.x, tt.y)
			}
		})
	}
}
