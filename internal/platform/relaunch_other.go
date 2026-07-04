//go:build !windows

package platform

import (
	"os"
	"os/exec"
)

// RelaunchDetached starts a fresh copy of exePath with the current process's
// arguments and environment. Used by the UI watchdog to recover from a wedged
// event loop.
func RelaunchDetached(exePath string) error {
	cmd := exec.Command(exePath, os.Args[1:]...)
	cmd.Env = os.Environ()
	return cmd.Start()
}
