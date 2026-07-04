//go:build windows

package platform

import (
	"os"
	"os/exec"
	"syscall"
)

// detachedProcess starts the child without inheriting the parent console so it
// survives the parent exiting.
const detachedProcess = 0x00000008 // DETACHED_PROCESS

// RelaunchDetached starts a fresh copy of exePath with the current process's
// arguments and environment, detached from this process. Used by the UI
// watchdog to recover from a wedged event loop.
func RelaunchDetached(exePath string) error {
	cmd := exec.Command(exePath, os.Args[1:]...)
	cmd.Env = os.Environ()
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: detachedProcess}
	return cmd.Start()
}
