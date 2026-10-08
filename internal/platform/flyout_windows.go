//go:build windows

package platform

import (
	"syscall"
	"unsafe"
)

var (
	fyUser32             = syscall.NewLazyDLL("user32.dll")
	fyGetCursorPos       = fyUser32.NewProc("GetCursorPos")
	fyMonitorFromPoint   = fyUser32.NewProc("MonitorFromPoint")
	fyGetMonitorInfoW    = fyUser32.NewProc("GetMonitorInfoW")
	fyGetWindowRect      = fyUser32.NewProc("GetWindowRect")
	fyGetDpiForWindow    = fyUser32.NewProc("GetDpiForWindow")
	fySetWindowPos       = fyUser32.NewProc("SetWindowPos")
	fyShowWindow         = fyUser32.NewProc("ShowWindow")
	fyIsWindowVisible    = fyUser32.NewProc("IsWindowVisible")
	fyGetWindowLongPtrW  = fyUser32.NewProc("GetWindowLongPtrW")
	fySetWindowLongPtrW  = fyUser32.NewProc("SetWindowLongPtrW")
	fyDwmapi             = syscall.NewLazyDLL("dwmapi.dll")
	fyDwmSetWindowAttrib = fyDwmapi.NewProc("DwmSetWindowAttribute")
)

const fyHWNDTopmost = ^uintptr(0) // (HWND)-1

const (
	fyMonitorDefaultToNearest = 2
	fySWPNoSize               = 0x0001
	fySWPNoMove               = 0x0002
	fySWPNoZOrder             = 0x0004
	fySWPNoActivate           = 0x0010
	fySWPFrameChanged         = 0x0020
	fyWSExToolWindow          = 0x00000080
	fyWSExAppWindow           = 0x00040000
	fySWHide                  = 0
	fySWShow                  = 5
	fyDWMWACornerPreference   = 33
	fyDWMWCPRound             = 2
	fyFlyoutGapDIP            = 8
)

type fyPoint struct{ X, Y int32 }

type fyRect struct{ Left, Top, Right, Bottom int32 }

type fyMonitorInfo struct {
	Size    uint32
	Monitor fyRect
	Work    fyRect
	Flags   uint32
}

func (r fyRect) rect() Rect {
	return Rect{Left: int(r.Left), Top: int(r.Top), Right: int(r.Right), Bottom: int(r.Bottom)}
}

// PositionFlyout docks the window next to the taskbar at the mouse cursor (see
// PlaceFlyout); use it when the flyout opens. Call it on the window's thread
// after the window is shown at its final size. Failures leave the window where
// it is.
func PositionFlyout(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	var pt fyPoint
	if r, _, _ := fyGetCursorPos.Call(uintptr(unsafe.Pointer(&pt))); r == 0 {
		return
	}
	dockFlyout(hwnd, pt)
}

// RedockFlyout re-docks the window around its own centre, e.g. after it was
// resized: it stays where it is along the taskbar and only its distance from
// the taskbar is corrected. (Using the cursor here would slide the flyout
// towards wherever the user just clicked inside it.)
func RedockFlyout(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	var wr fyRect
	if r, _, _ := fyGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr))); r == 0 {
		return
	}
	dockFlyout(hwnd, fyPoint{X: (wr.Left + wr.Right) / 2, Y: (wr.Top + wr.Bottom) / 2})
}

// dockFlyout places the window with PlaceFlyout, anchored at pt on the monitor
// containing pt.
func dockFlyout(hwnd uintptr, pt fyPoint) {
	// MonitorFromPoint takes the POINT by value: x and y packed into one
	// 64-bit argument.
	packed := uintptr(uint32(pt.X)) | uintptr(uint32(pt.Y))<<32
	mon, _, _ := fyMonitorFromPoint.Call(packed, fyMonitorDefaultToNearest)
	mi := fyMonitorInfo{Size: uint32(unsafe.Sizeof(fyMonitorInfo{}))}
	if r, _, _ := fyGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi))); r == 0 {
		return
	}
	var wr fyRect
	if r, _, _ := fyGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&wr))); r == 0 {
		return
	}
	dpi := uintptr(96)
	if fyGetDpiForWindow.Find() == nil {
		if d, _, _ := fyGetDpiForWindow.Call(hwnd); d != 0 {
			dpi = d
		}
	}
	gap := int(fyFlyoutGapDIP * dpi / 96)
	x, y := PlaceFlyout(int(pt.X), int(pt.Y), mi.Monitor.rect(), mi.Work.rect(),
		int(wr.Right-wr.Left), int(wr.Bottom-wr.Top), gap)
	// Topmost, like the shell's own flyouts: the panel must sit above other
	// windows even when Windows refuses to activate it.
	_, _, _ = fySetWindowPos.Call(hwnd, fyHWNDTopmost, uintptr(x), uintptr(y), 0, 0,
		fySWPNoSize|fySWPNoActivate)
}

// StyleFlyout makes the window a flyout: no taskbar button or Alt-Tab entry
// (WS_EX_TOOLWINDOW) and, on Windows 11, rounded corners. Call it once on the
// window's thread after the window first exists.
func StyleFlyout(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	gwlExStyle := -20
	ex, _, _ := fyGetWindowLongPtrW.Call(hwnd, uintptr(gwlExStyle))
	ex = (ex | fyWSExToolWindow) &^ fyWSExAppWindow
	_, _, _ = fySetWindowLongPtrW.Call(hwnd, uintptr(gwlExStyle), ex)
	if fyDwmSetWindowAttrib.Find() == nil {
		pref := int32(fyDWMWCPRound)
		_, _, _ = fyDwmSetWindowAttrib.Call(hwnd, fyDWMWACornerPreference,
			uintptr(unsafe.Pointer(&pref)), unsafe.Sizeof(pref))
	}
	_, _, _ = fySetWindowPos.Call(hwnd, 0, 0, 0, 0, 0,
		fySWPNoMove|fySWPNoSize|fySWPNoZOrder|fySWPNoActivate|fySWPFrameChanged)
	// The taskbar only drops an existing button when the window is re-shown.
	if v, _, _ := fyIsWindowVisible.Call(hwnd); v != 0 {
		_, _, _ = fyShowWindow.Call(hwnd, fySWHide)
		_, _, _ = fyShowWindow.Call(hwnd, fySWShow)
	}
}
