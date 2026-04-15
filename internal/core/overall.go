package core

func ComputeOverall(probes []ProbeResult, sp []StatuspageResult) OverallStatus {
	for _, s := range sp {
		if s.Indicator == StatuspageMajor || s.Indicator == StatuspageCritical {
			return StatusDown
		}
	}

	downCount := 0
	slow := false
	for _, p := range probes {
		if !p.IsUp() {
			downCount++
		} else if p.LatencyMs > 1000 {
			slow = true
		}
	}
	if downCount >= 3 {
		return StatusDown
	}

	for _, s := range sp {
		if s.Indicator == StatuspageMinor || s.Indicator == StatuspageMaintenance {
			return StatusDegraded
		}
	}

	if downCount > 0 || slow {
		return StatusDegraded
	}
	return StatusHealthy
}
