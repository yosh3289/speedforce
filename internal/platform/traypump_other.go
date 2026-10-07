//go:build !windows

package platform

// TrayPumpResponsive is a no-op on non-Windows platforms: the systray
// thread-affinity failure it guards against is Windows-specific. Always
// reports responsive so the watchdog only ever acts on the fyne probe.
func TrayPumpResponsive(timeoutMs uint32) bool { return true }
