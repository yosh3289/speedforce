package core

import "time"

type OverallStatus int

const (
	StatusUnknown OverallStatus = iota
	StatusHealthy
	StatusDegraded
	StatusDown
)

func (s OverallStatus) ColorHex() string {
	switch s {
	case StatusHealthy:
		return "#2979FF"
	case StatusDegraded:
		return "#FFC400"
	case StatusDown:
		return "#D50000"
	default:
		return "#9E9E9E"
	}
}

type ProbeResult struct {
	Name       string
	URL        string
	StatusCode int
	LatencyMs  int64
	Err        error
	Timestamp  time.Time
}

func (p ProbeResult) IsUp() bool {
	if p.Err != nil {
		return false
	}
	return p.StatusCode >= 200 && p.StatusCode < 500
}

type TickMode int

const (
	TickNormal TickMode = iota
	TickFast
)

type IPInfo struct {
	PublicIP  string
	LANIP     string
	Country   string
	City      string
	ISP       string
	FetchedAt time.Time
}

type StatuspageIndicator string

const (
	StatuspageOperational StatuspageIndicator = "none"
	StatuspageMinor       StatuspageIndicator = "minor"
	StatuspageMajor       StatuspageIndicator = "major"
	StatuspageCritical    StatuspageIndicator = "critical"
	StatuspageMaintenance StatuspageIndicator = "maintenance"
)

type StatuspageResult struct {
	Name        string
	Indicator   StatuspageIndicator
	Description string
	FetchedAt   time.Time
	Err         error
}

type State struct {
	HTTPS      []ProbeResult
	IP         IPInfo
	Statuspage []StatuspageResult
	Mode       TickMode
	UpdatedAt  time.Time
}
