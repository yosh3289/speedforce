//go:build linux

package platform

func setAutoStartImpl(enabled bool, exePath string) error { return nil }
func isAutoStartImpl() (bool, error)                      { return false, nil }
