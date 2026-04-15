//go:build windows

package platform

import "golang.org/x/sys/windows/registry"

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const valueName = "SpeedForce"

func setAutoStartImpl(enabled bool, exePath string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if enabled {
		return k.SetStringValue(valueName, exePath)
	}
	return k.DeleteValue(valueName)
}

func isAutoStartImpl() (bool, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer k.Close()
	_, _, err = k.GetStringValue(valueName)
	if err == registry.ErrNotExist {
		return false, nil
	}
	return err == nil, err
}
