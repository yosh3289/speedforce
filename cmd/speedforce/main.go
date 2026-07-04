package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
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
	if isAuto, _ := platform.IsAutoStart(); isAuto != cfg.UI.AutoStart {
		if err := platform.SetAutoStart(cfg.UI.AutoStart, exePath); err != nil {
			log.Printf("autostart: %v", err)
		}
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
				func(newCfg *config.Config) error {
					if err := config.Save(cfgPath, newCfg); err != nil {
						return err
					}
					_ = tr.SetLocale(newCfg.Language)
					return nil
				},
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

	var quitting atomic.Bool
	t := tray.New(tr, tray.Callbacks{
		OnDetail:   showDetail,
		OnSettings: showSettings,
		OnQuit: func() {
			quitting.Store(true)
			cancel()
			fyneApp.Quit()
		},
	})

	go func() {
		sub := bus.Subscribe()
		for s := range sub {
			overall := core.ComputeOverall(s.HTTPS, s.Statuspage)
			t.SetStatus(overall)
			t.UpdateTooltip(s)
		}
	}()

	go t.Run()

	// Hidden master window keeps fyne event loop alive even when no
	// detail/settings window is visible. fyne.Run() exits if no windows
	// are shown.
	master := fyneApp.NewWindow("SpeedForce")
	master.SetMaster()
	master.SetCloseIntercept(func() { master.Hide() })
	master.Resize(fyne.NewSize(1, 1))
	master.Hide()

	// Watchdog: recover if the fyne UI event loop stops servicing work (e.g.
	// the GL context is lost after the machine sleeps). It probes loop
	// liveness and, on a sustained stall, dumps a goroutine trace and relaunches.
	if os.Getenv("SF_NO_WATCHDOG") == "" {
		go runWatchdog(exePath, release, &quitting)
	}

	fyneApp.Run()
}

func appDataDir() string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = "."
	}
	return filepath.Join(appData, "SpeedForce")
}

func configPath() string {
	dir := appDataDir()
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, "config.yaml")
}

// runWatchdog probes the fyne UI event loop for liveness. A healthy loop runs
// queued funcs within milliseconds; if two consecutive probes time out the loop
// is wedged, so we capture a goroutine dump and relaunch to recover. Two
// consecutive failures (not one) avoids false positives around sleep/resume.
func runWatchdog(exePath string, release func(), quitting *atomic.Bool) {
	const probeEvery = 20 * time.Second
	const probeTimeout = 30 * time.Second
	stalls := 0
	for {
		time.Sleep(probeEvery)
		if quitting.Load() {
			return
		}
		if uiResponsive(probeTimeout) {
			stalls = 0
			continue
		}
		if quitting.Load() {
			return
		}
		stalls++
		log.Printf("watchdog: UI loop unresponsive (%d/2)", stalls)
		if stalls >= 2 {
			recoverWedge(exePath, release)
			return
		}
	}
}

// uiResponsive reports whether the fyne main loop runs a queued func within timeout.
func uiResponsive(timeout time.Duration) bool {
	done := make(chan struct{})
	fyne.Do(func() { close(done) })
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

func recoverWedge(exePath string, release func()) {
	dumpGoroutines()
	if !allowRestart() {
		log.Printf("watchdog: too many restarts within 10m; not relaunching again")
		return // keep holding the single-instance mutex; leave process as-is
	}
	log.Printf("watchdog: relaunching to recover the UI")
	// Release the single-instance mutex BEFORE spawning the child, otherwise
	// the child's AcquireSingleton sees ERROR_ALREADY_EXISTS and exits, which
	// combined with our os.Exit would leave no instance running.
	if release != nil {
		release()
	}
	if err := platform.RelaunchDetached(exePath); err != nil {
		log.Printf("watchdog: relaunch failed: %v", err)
	}
	os.Exit(1)
}

// dumpGoroutines writes a full goroutine trace next to the config file, so a
// wedge occurring in real use leaves behind the stuck loop's stack.
func dumpGoroutines() {
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	dir := appDataDir()
	_ = os.MkdirAll(dir, 0o755)
	path := filepath.Join(dir, fmt.Sprintf("wedge-%s.log", time.Now().Format("20060102-150405")))
	if err := os.WriteFile(path, buf[:n], 0o644); err != nil {
		log.Printf("watchdog: failed to write dump: %v", err)
		return
	}
	log.Printf("watchdog: wrote goroutine dump to %s", path)
}

// allowRestart is a restart-storm guard: at most 3 relaunches within a rolling
// 10-minute window, tracked across restarts via inherited env vars.
func allowRestart() bool {
	now := time.Now().Unix()
	gen, _ := strconv.Atoi(os.Getenv("SF_WD_GEN"))
	first, _ := strconv.ParseInt(os.Getenv("SF_WD_FIRST"), 10, 64)
	if first == 0 || now-first > 600 {
		gen = 0
		first = now
	}
	gen++
	_ = os.Setenv("SF_WD_GEN", strconv.Itoa(gen))
	_ = os.Setenv("SF_WD_FIRST", strconv.FormatInt(first, 10))
	return gen <= 3
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
