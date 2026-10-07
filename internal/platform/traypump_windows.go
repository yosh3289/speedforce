//go:build windows

package platform

import (
	"sync"
	"syscall"
	"unsafe"
)

// Liveness probe for the fyne.io/systray Win32 message pump. The pump runs
// on a dedicated OS thread that owns the hidden "SystrayClass" window; if it
// ever stops servicing that window's message queue (historically because the
// pump goroutine migrated off the window's thread), the tray icon stays painted
// but the menu goes dead. This probe detects that from the outside by sending a
// no-op message and checking whether the pump processes it.

var (
	tpUser32                 = syscall.NewLazyDLL("user32.dll")
	tpEnumWindows            = tpUser32.NewProc("EnumWindows")
	tpGetClassNameW          = tpUser32.NewProc("GetClassNameW")
	tpGetWindowThreadProcess = tpUser32.NewProc("GetWindowThreadProcessId")
	tpSendMessageTimeoutW    = tpUser32.NewProc("SendMessageTimeoutW")

	tpKernel32          = syscall.NewLazyDLL("kernel32.dll")
	tpGetCurrentProcess = tpKernel32.NewProc("GetCurrentProcessId")
)

const (
	tpWMNull          = 0x0000
	tpSMTOAbortIfHung = 0x0002
	tpTrayClassName   = "SystrayClass"
)

// Guards the shared enumeration state below. TrayPumpResponsive is only called
// from the watchdog goroutine today, but the mutex keeps findTrayWindow correct
// if that ever changes. syscall.NewCallback must be created once (its slots are
// never freed), so the callback and its result state are package-level.
var (
	tpFindMu    sync.Mutex
	tpFoundHwnd uintptr
	tpMyPID     uintptr
)

var tpEnumCallback = syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
	var pid uint32
	_, _, _ = tpGetWindowThreadProcess.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	if uintptr(pid) != tpMyPID {
		return 1 // not ours; continue enumeration
	}
	var buf [64]uint16
	n, _, _ := tpGetClassNameW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n > 0 && syscall.UTF16ToString(buf[:n]) == tpTrayClassName {
		tpFoundHwnd = hwnd
		return 0 // found it; stop enumeration
	}
	return 1
})

// findTrayWindow returns the HWND of this process's systray window, or 0.
func findTrayWindow() uintptr {
	tpFindMu.Lock()
	defer tpFindMu.Unlock()
	myPID, _, _ := tpGetCurrentProcess.Call()
	tpMyPID = myPID
	tpFoundHwnd = 0
	_, _, _ = tpEnumWindows.Call(tpEnumCallback, 0)
	return tpFoundHwnd
}

// TrayPumpResponsive reports whether the systray message pump is still
// processing its window's messages. It sends WM_NULL with SMTO_ABORTIFHUNG and
// the given timeout: a healthy pump replies in well under a millisecond, while a
// pump that has stopped servicing its queue returns false. Returns true when no
// tray window exists yet (nothing to probe) so it never false-positives at
// startup.
func TrayPumpResponsive(timeoutMs uint32) bool {
	hwnd := findTrayWindow()
	if hwnd == 0 {
		return true
	}
	var result uintptr
	r1, _, _ := tpSendMessageTimeoutW.Call(
		hwnd,
		uintptr(tpWMNull),
		0,
		0,
		uintptr(tpSMTOAbortIfHung),
		uintptr(timeoutMs),
		uintptr(unsafe.Pointer(&result)),
	)
	return r1 != 0
}
