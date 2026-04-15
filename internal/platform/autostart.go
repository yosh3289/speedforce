package platform

func SetAutoStart(enabled bool, exePath string) error {
	return setAutoStartImpl(enabled, exePath)
}

func IsAutoStart() (bool, error) {
	return isAutoStartImpl()
}
