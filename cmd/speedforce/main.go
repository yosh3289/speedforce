package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2/app"

	"github.com/yosh3289/speedforce/internal/config"
	"github.com/yosh3289/speedforce/internal/core"
	"github.com/yosh3289/speedforce/internal/i18n"
	"github.com/yosh3289/speedforce/internal/platform"
	"github.com/yosh3289/speedforce/internal/probe"
	"github.com/yosh3289/speedforce/internal/ui/detail"
	"github.com/yosh3289/speedforce/internal/ui/settings"
	"github.com/yosh3289/speedforce/internal/ui/tray"
)

func main() {
	cfgPath := configPath()

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	tr, err := i18n.New(cfg.Language)
	if err != nil {
		log.Fatalf("i18n: %v", err)
	}

	client := probe.NewClient(probe.ClientOptions{
		Timeout:       time.Duration(cfg.Network.TimeoutMs) * time.Millisecond,
		ProxyMode:     cfg.Network.Proxy.Mode,
		ProxyURL:      cfg.Network.Proxy.ManualURL,
		SystemProxyFn: platform.SystemProxyFunc(),
	})

	httpsProber := probe.NewHTTPSProber(client)
	ipProber := probe.NewIPProber(client, cfg.Probes.IP.PublicIPAPI, cfg.Probes.IP.GeoAPI)
	spProber := probe.NewStatuspageProber(client)

	bus := core.NewStateBus()
	defer bus.Close()

	httpsTargets := make([]core.Target, 0, len(cfg.Probes.HTTPS))
	for _, p := range cfg.Probes.HTTPS {
		httpsTargets = append(httpsTargets, core.Target{Name: p.Name, URL: p.URL})
	}
	spTargets := make([]core.StatuspageTarget, 0, len(cfg.Probes.Statuspage.Sources))
	for _, s := range cfg.Probes.Statuspage.Sources {
		spTargets = append(spTargets, core.StatuspageTarget{Name: s.Name, URL: s.URL})
	}

	sch := core.NewScheduler(core.SchedulerConfig{
		Bus:                   bus,
		HTTPS:                 httpsProber,
		HTTPSTargets:          httpsTargets,
		IP:                    ipProber,
		IPRefreshEveryTicks:   cfg.Probes.IP.RefreshEveryTicks,
		Statuspage:            spProber,
		StatuspageTargets:     spTargets,
		StatuspageIntervalSec: cfg.Probes.Statuspage.RefreshIntervalSec,
		TickInterval:          time.Duration(cfg.Network.TickInterval) * time.Second,
		FastInterval:          time.Duration(cfg.Network.AdaptiveTick.FastIntervalSec) * time.Second,
		AdaptiveEnabled:       cfg.Network.AdaptiveTick.Enabled,
		ThresholdMs:           int64(cfg.Network.AdaptiveTick.ThresholdMs),
		RecoveryMs:            int64(cfg.Network.AdaptiveTick.RecoveryThresholdMs),
		MaxFastDuration:       time.Duration(cfg.Network.AdaptiveTick.MaxFastDurationSec) * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	go sch.Run(ctx)

	fyneApp := app.NewWithID("com.speedforce")

	var detailWin *detail.Window
	var settingsWin *settings.Window

	showSettings := func() {
		if settingsWin == nil {
			settingsWin = settings.New(fyneApp, tr, cfg,
				func(newCfg *config.Config) error { return nil },
				func() (string, error) { return "", nil },
			)
		}
		settingsWin.Show()
	}

	showDetail := func() {
		if detailWin == nil {
			detailWin = detail.New(fyneApp, tr, bus, showSettings)
		}
		detailWin.Show()
	}

	t := tray.New(tr, tray.Callbacks{
		OnDetail:   showDetail,
		OnSettings: showSettings,
		OnQuit: func() {
			cancel()
			os.Exit(0)
		},
	})

	go func() {
		sub := bus.Subscribe()
		for s := range sub {
			overall := core.ComputeOverall(s.HTTPS, s.Statuspage)
			t.SetStatus(overall)
		}
	}()

	t.Run()
}

func configPath() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = "."
	}
	dir := filepath.Join(appData, "SpeedForce")
	_ = os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "config.yaml")
}
