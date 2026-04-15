package core

import (
	"fmt"
	"sync"
	"time"
)

type ToastSender interface {
	Notify(title, body string) error
}

type NotifierConfig struct {
	Toast                 ToastSender
	NotifyHTTPS           map[string]bool
	NotifyStatuspageMajor bool
	CooldownDur           time.Duration
}

type Notifier struct {
	cfg NotifierConfig

	mu        sync.Mutex
	prev      *State
	lastFired map[string]time.Time
}

func NewNotifier(cfg NotifierConfig) *Notifier {
	if cfg.CooldownDur == 0 {
		cfg.CooldownDur = 5 * time.Minute
	}
	return &Notifier{cfg: cfg, lastFired: make(map[string]time.Time)}
}

func (n *Notifier) Observe(curr State) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.prev == nil {
		snap := curr
		n.prev = &snap
		return
	}

	n.checkHTTPS(curr)
	n.checkStatuspage(curr)

	snap := curr
	n.prev = &snap
}

func (n *Notifier) checkHTTPS(curr State) {
	prevMap := make(map[string]bool)
	for _, p := range n.prev.HTTPS {
		prevMap[p.Name] = p.IsUp()
	}
	for _, p := range curr.HTTPS {
		if !n.cfg.NotifyHTTPS[p.Name] {
			continue
		}
		wasUp, seen := prevMap[p.Name]
		if !seen {
			continue
		}
		if wasUp && !p.IsUp() {
			n.fireIfCooled("https:"+p.Name,
				fmt.Sprintf("%s unreachable", p.Name),
				fmt.Sprintf("SpeedForce cannot reach %s.", p.URL))
		}
	}
}

func (n *Notifier) checkStatuspage(curr State) {
	if !n.cfg.NotifyStatuspageMajor {
		return
	}
	prevMap := make(map[string]StatuspageIndicator)
	for _, s := range n.prev.Statuspage {
		prevMap[s.Name] = s.Indicator
	}
	for _, s := range curr.Statuspage {
		prev, seen := prevMap[s.Name]
		if !seen {
			continue
		}
		wasMajor := prev == StatuspageMajor || prev == StatuspageCritical
		isMajor := s.Indicator == StatuspageMajor || s.Indicator == StatuspageCritical
		if !wasMajor && isMajor {
			n.fireIfCooled("statuspage:"+s.Name,
				fmt.Sprintf("%s reports major outage", s.Name),
				s.Description)
		}
	}
}

func (n *Notifier) fireIfCooled(key, title, body string) {
	last, seen := n.lastFired[key]
	if seen && time.Since(last) < n.cfg.CooldownDur {
		return
	}
	if n.cfg.Toast != nil {
		_ = n.cfg.Toast.Notify(title, body)
	}
	n.lastFired[key] = time.Now()
}
