package tray

import (
	_ "embed"

	"github.com/yosh3289/speedforce/internal/core"
)

//go:embed icons/lightning-blue.ico
var iconBlue []byte

//go:embed icons/lightning-yellow.ico
var iconYellow []byte

//go:embed icons/lightning-red.ico
var iconRed []byte

func IconFor(status core.OverallStatus) []byte {
	switch status {
	case core.StatusHealthy:
		return iconBlue
	case core.StatusDegraded:
		return iconYellow
	case core.StatusDown:
		return iconRed
	default:
		return iconBlue
	}
}
