package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
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
	release, singletonErr := platform.AcquireSingleton(`Global\SpeedForce.Singleton`)
	if singletonErr != nil {
		log.Fatalf("another instance already running: %v", singletonErr)
	}
	defer release()

	fakeDown := flag.String("fake-down", "", "comma-separated service names to simulate as down")
	tickOverride := flag.Int("tick", 0, "override tick interval in seconds (debug)")
	flag.Parse()

	cfgPath := configPath()

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if *tickOverride > 0 {
		cfg.Network.TickInterval = *tickOverride
	}

	exePath, _ := os.Executable()
	if err := platform.SetAutoStart(cfg.UI.AutoStart, exePath); err != nil {
		log.Printf("autostart: %v", err)
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

	var httpsProber core.HTTPSProbeFunc = probe.NewHTTPSProber(client)
	if *fakeDown != "" {
		set := make(map[string]bool)
		for _, n := range strings.Split(*fakeDown, ",") {
			set[strings.TrimSpace(n)] = true
		}
		httpsProber = &fakeDownProber{
			inner:   probe.NewHTTPSProber(client),
			downSet: set,
		}
	}
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

type fakeDownProber struct {
	inner   *probe.HTTPSProber
	downSet map[string]bool
}

func (f *fakeDownProber) Probe(ctx context.Context, name, url string) core.ProbeResult {
	if f.downSet[name] {
		return core.ProbeResult{Name: name, URL: url, Err: errFakeDown, Timestamp: time.Now()}
	}
	return f.inner.Probe(ctx, name, url)
}

var errFakeDown = errors.New("fake-down")
