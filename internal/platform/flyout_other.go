//go:build !windows

package platform

// PositionFlyout is a no-op outside Windows; the window stays where the
// toolkit put it.
func PositionFlyout(hwnd uintptr) {}

// RedockFlyout is a no-op outside Windows.
func RedockFlyout(hwnd uintptr) {}

// StyleFlyout is a no-op outside Windows.
func StyleFlyout(hwnd uintptr) {}
