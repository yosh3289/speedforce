# SpeedForce Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a lightweight Windows tray app (SpeedForce) that monitors reachability to Claude/OpenAI/Gemini through the user's network/proxy and surfaces service status.

**Architecture:** Go application with three layers — UI (systray + fyne), Core (scheduler, state bus, notifier, config, logger, i18n), and Probes (HTTPS, IP, Statuspage). UI subscribes to a single StateBus. Platform-specific code isolated behind Go build tags for future macOS/Linux support.

**Tech Stack:** Go 1.22+, `fyne.io/fyne/v2` (UI), `getlantern/systray` (tray), `gen2brain/beeep` (notifications), `goccy/go-yaml` (YAML w/ comment round-trip), `fsnotify/fsnotify` (config watch), `rs/zerolog` (structured logging). Build target: Windows amd64 first; platform stubs for darwin/linux.

---

## File Structure

**Package layout (all paths relative to `speedforce/`):**

| Path | Responsibility |
|---|---|
| `cmd/speedforce/main.go` | Entry point; wires Config → Logger → Scheduler → UI |
| `internal/core/types.go` | Shared types: `ProbeResult`, `State`, `OverallStatus`, `TickMode` |
| `internal/core/statebus.go` | In-memory state store + pub/sub channels |
| `internal/core/scheduler.go` | Tick loop; dispatches probers; owns adaptive tick state machine |
| `internal/core/notifier.go` | State diff → toast dispatch with cooldown |
| `internal/probe/httpclient.go` | Proxy-aware `*http.Client` factory |
| `internal/probe/https.go` | `HTTPSProber` |
| `internal/probe/ip.go` | `IPProber` (public IP + geo + LAN IP) |
| `internal/probe/statuspage.go` | `StatuspageProber` (Statuspage.io JSON API) |
| `internal/config/config.go` | Load / validate / hot-reload config |
| `internal/config/defaults.go` | Default config generation |
| `internal/i18n/i18n.go` | Locale loader + `T(key)` lookup |
| `internal/i18n/locales/zh.json` | Chinese strings |
| `internal/i18n/locales/en.json` | English strings |
| `internal/ui/tray/tray.go` | Systray icon and menu |
| `internal/ui/tray/icons.go` | Embedded `.ico` bytes (blue/yellow/red) |
| `internal/ui/detail/window.go` | Detail window (fyne) |
| `internal/ui/settings/window.go` | Settings window (fyne) |
| `internal/platform/proxy_windows.go` | Windows proxy detection via registry |
| `internal/platform/proxy_darwin.go` | Stub (build tag: darwin) |
| `internal/platform/proxy_linux.go` | Stub (build tag: linux) |
| `internal/platform/autostart_windows.go` | Windows autostart via HKCU\Run |
| `internal/platform/singleton_windows.go` | Named-mutex singleton check |
| `internal/logger/logger.go` | Structured logging + daily rotation |

---

## Milestones Overview

| Milestone | Tasks | Outcome |
|---|---|---|
| M1: Skeleton | 1-6 | Builds, tray icon changes color based on a single HTTPS probe |
| M2: Core Complete | 7-14 | All probes + StateBus + adaptive tick + config loading |
| M3: UI | 15-19 | Detail + Settings windows, i18n, full icon switching |
| M4: Feature-Complete | 20-24 | Hot-reload, notifications, autostart, log export, singleton |
| M5: Polish + Release | 25-30 | Integration tests, CI, README, v0.1 release |

---

## M1 — Skeleton

### Task 1: Initialize Go module and directory layout

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `README.md` (stub)

- [ ] **Step 1: Initialize Go module**

Run:
```bash
cd E:/0-syb/dev/speedforce
go mod init github.com/<your-username>/speedforce
```
Expected: `go.mod` created with module path.

- [ ] **Step 2: Create directory scaffold**

Run:
```bash
mkdir -p cmd/speedforce internal/core internal/probe internal/config internal/i18n/locales internal/ui/tray internal/ui/detail internal/ui/settings internal/platform internal/logger assets/icons
```

- [ ] **Step 3: Write .gitignore**

```gitignore
/speedforce
/speedforce.exe
/dist/
*.log
coverage.out
.idea/
.vscode/
```

- [ ] **Step 4: Write README.md stub**

```markdown
# SpeedForce ⚡

Lightweight tray app that monitors reachability to Claude, OpenAI, and Gemini.

Status: under development. See `docs/superpowers/specs/` for the design.
```

- [ ] **Step 5: Commit**

```bash
git add go.mod .gitignore README.md
git commit -m "chore: init go module and scaffold"
```

---

### Task 2: Core shared types

**Files:**
- Create: `internal/core/types.go`
- Test: `internal/core/types_test.go`

- [ ] **Step 1: Write failing test**

`internal/core/types_test.go`:
```go
package core

import "testing"

func TestOverallStatus_Color(t *testing.T) {
	cases := []struct {
		name     string
		status   OverallStatus
		wantHex  string
	}{
		{"healthy is blue", StatusHealthy, "#2979FF"},
		{"degraded is yellow", StatusDegraded, "#FFC400"},
		{"down is red", StatusDown, "#D50000"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.status.ColorHex(); got != c.wantHex {
				t.Errorf("got %s, want %s", got, c.wantHex)
			}
		})
	}
}

func TestProbeResult_IsUp(t *testing.T) {
	up := ProbeResult{StatusCode: 401, Err: nil}
	if !up.IsUp() {
		t.Error("401 should be considered up (reachable)")
	}
	down := ProbeResult{StatusCode: 0, Err: errTestTimeout}
	if down.IsUp() {
		t.Error("timeout should be considered down")
	}
	serverErr := ProbeResult{StatusCode: 503, Err: nil}
	if serverErr.IsUp() {
		t.Error("5xx should be considered down")
	}
}

var errTestTimeout = &testErr{msg: "timeout"}

type testErr struct{ msg string }

func (e *testErr) Error() string { return e.msg }
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/core/... -run TestOverallStatus_Color
```
Expected: FAIL — undefined `OverallStatus`, `StatusHealthy`, etc.

- [ ] **Step 3: Implement types**

`internal/core/types.go`:
```go
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
	PublicIP string
	LANIP    string
	Country  string
	City     string
	ISP      string
	FetchedAt time.Time
}

type StatuspageIndicator string

const (
	StatuspageOperational   StatuspageIndicator = "none"
	StatuspageMinor         StatuspageIndicator = "minor"
	StatuspageMajor         StatuspageIndicator = "major"
	StatuspageCritical      StatuspageIndicator = "critical"
	StatuspageMaintenance   StatuspageIndicator = "maintenance"
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
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/core/... -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/core/types.go internal/core/types_test.go
git commit -m "feat(core): add shared types for probes and status"
```

---

### Task 3: Proxy-aware HTTP client factory

**Files:**
- Create: `internal/probe/httpclient.go`
- Test: `internal/probe/httpclient_test.go`

- [ ] **Step 1: Write failing test**

`internal/probe/httpclient_test.go`:
```go
package probe

import (
	"net/url"
	"testing"
	"time"
)

func TestNewClient_NoProxy(t *testing.T) {
	c := NewClient(ClientOptions{Timeout: 2 * time.Second, ProxyMode: "none"})
	if c == nil {
		t.Fatal("client is nil")
	}
	if c.Timeout != 2*time.Second {
		t.Errorf("timeout = %v, want 2s", c.Timeout)
	}
}

func TestNewClient_ManualProxy(t *testing.T) {
	c := NewClient(ClientOptions{
		Timeout:   2 * time.Second,
		ProxyMode: "manual",
		ProxyURL:  "http://127.0.0.1:7890",
	})
	if c == nil {
		t.Fatal("client is nil")
	}
	tr, ok := c.Transport.(*proxyTransport)
	if !ok {
		t.Fatalf("transport not *proxyTransport: %T", c.Transport)
	}
	u, _ := url.Parse("http://example.com")
	got, err := tr.proxyFunc(&httpRequest{URL: u})
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.String() != "http://127.0.0.1:7890" {
		t.Errorf("proxy = %v, want http://127.0.0.1:7890", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/probe/... -run TestNewClient
```
Expected: FAIL — undefined `NewClient`, `ClientOptions`, etc.

- [ ] **Step 3: Implement client factory**

`internal/probe/httpclient.go`:
```go
package probe

import (
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type ProxyMode string

const (
	ProxyAuto   ProxyMode = "auto"
	ProxyManual ProxyMode = "manual"
	ProxyNone   ProxyMode = "none"
)

type ClientOptions struct {
	Timeout   time.Duration
	ProxyMode string
	ProxyURL  string
	SystemProxyFn func(*http.Request) (*url.URL, error)
}

type httpRequest = http.Request

type proxyTransport struct {
	*http.Transport
	proxyFunc func(*httpRequest) (*url.URL, error)
}

func NewClient(opts ClientOptions) *http.Client {
	if opts.Timeout == 0 {
		opts.Timeout = 5 * time.Second
	}

	tr := &http.Transport{
		DisableKeepAlives:     true,
		TLSHandshakeTimeout:   opts.Timeout,
		ResponseHeaderTimeout: opts.Timeout,
	}

	var proxyFn func(*http.Request) (*url.URL, error)

	switch ProxyMode(opts.ProxyMode) {
	case ProxyManual:
		u, err := url.Parse(opts.ProxyURL)
		if err != nil {
			proxyFn = func(*http.Request) (*url.URL, error) {
				return nil, fmt.Errorf("invalid manual proxy URL: %w", err)
			}
		} else {
			proxyFn = func(*http.Request) (*url.URL, error) { return u, nil }
		}
	case ProxyNone:
		proxyFn = nil
	default:
		if opts.SystemProxyFn != nil {
			proxyFn = opts.SystemProxyFn
		} else {
			proxyFn = http.ProxyFromEnvironment
		}
	}

	tr.Proxy = proxyFn

	return &http.Client{
		Timeout:   opts.Timeout,
		Transport: &proxyTransport{Transport: tr, proxyFunc: proxyFn},
	}
}

func (t *proxyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.Transport.RoundTrip(req)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/probe/... -v -run TestNewClient
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/probe/httpclient.go internal/probe/httpclient_test.go
git commit -m "feat(probe): add proxy-aware HTTP client factory"
```

---

### Task 4: HTTPSProber

**Files:**
- Create: `internal/probe/https.go`
- Test: `internal/probe/https_test.go`

- [ ] **Step 1: Write failing test using httptest**

`internal/probe/https_test.go`:
```go
package probe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPSProber_2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	p := NewHTTPSProber(NewClient(ClientOptions{Timeout: 2 * time.Second, ProxyMode: "none"}))
	res := p.Probe(context.Background(), "test", srv.URL)
	if !res.IsUp() {
		t.Errorf("expected up, got err=%v code=%d", res.Err, res.StatusCode)
	}
	if res.StatusCode != 200 {
		t.Errorf("status = %d, want 200", res.StatusCode)
	}
}

func TestHTTPSProber_401IsUp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer srv.Close()

	p := NewHTTPSProber(NewClient(ClientOptions{Timeout: 2 * time.Second, ProxyMode: "none"}))
	res := p.Probe(context.Background(), "test", srv.URL)
	if !res.IsUp() {
		t.Errorf("401 should be up (network path OK, auth missing); got down")
	}
}

func TestHTTPSProber_503IsDown(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(503)
	}))
	defer srv.Close()

	p := NewHTTPSProber(NewClient(ClientOptions{Timeout: 2 * time.Second, ProxyMode: "none"}))
	res := p.Probe(context.Background(), "test", srv.URL)
	if res.IsUp() {
		t.Errorf("503 should be down")
	}
}

func TestHTTPSProber_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer srv.Close()

	p := NewHTTPSProber(NewClient(ClientOptions{Timeout: 100 * time.Millisecond, ProxyMode: "none"}))
	res := p.Probe(context.Background(), "test", srv.URL)
	if res.IsUp() {
		t.Error("timeout should be down")
	}
	if res.Err == nil {
		t.Error("expected error")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/probe/... -run TestHTTPSProber
```
Expected: FAIL — undefined `NewHTTPSProber`

- [ ] **Step 3: Implement HTTPSProber**

`internal/probe/https.go`:
```go
package probe

import (
	"context"
	"net/http"
	"time"

	"github.com/<your-username>/speedforce/internal/core"
)

type HTTPSProber struct {
	client *http.Client
}

func NewHTTPSProber(client *http.Client) *HTTPSProber {
	return &HTTPSProber{client: client}
}

func (p *HTTPSProber) Probe(ctx context.Context, name, rawURL string) core.ProbeResult {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, rawURL, nil)
	if err != nil {
		return core.ProbeResult{Name: name, URL: rawURL, Err: err, Timestamp: time.Now()}
	}
	req.Header.Set("User-Agent", "SpeedForce/1.0 (+probe)")

	resp, err := p.client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return core.ProbeResult{
			Name: name, URL: rawURL, Err: err,
			LatencyMs: latency, Timestamp: time.Now(),
		}
	}
	defer resp.Body.Close()
	return core.ProbeResult{
		Name: name, URL: rawURL, StatusCode: resp.StatusCode,
		LatencyMs: latency, Timestamp: time.Now(),
	}
}
```

Replace `<your-username>` with the actual module path used in Task 1.

- [ ] **Step 4: Run tests to verify pass**

```bash
go test ./internal/probe/... -v -run TestHTTPSProber
```
Expected: PASS on all four test cases

- [ ] **Step 5: Commit**

```bash
git add internal/probe/https.go internal/probe/https_test.go
git commit -m "feat(probe): add HTTPS reachability prober"
```

---

### Task 5: Embed lightning bolt icons

**Files:**
- Create: `assets/icons/lightning-blue.ico`
- Create: `assets/icons/lightning-yellow.ico`
- Create: `assets/icons/lightning-red.ico`
- Create: `internal/ui/tray/icons.go`

- [ ] **Step 1: Generate the three ICO files**

Use any vector → .ico workflow. Simplest: create three 32×32 PNGs (one per color) of a lightning bolt glyph, convert to `.ico`:

```bash
# Using ImageMagick (install first if needed)
magick convert assets/icons/lightning-blue.png assets/icons/lightning-blue.ico
magick convert assets/icons/lightning-yellow.png assets/icons/lightning-yellow.ico
magick convert assets/icons/lightning-red.png assets/icons/lightning-red.ico
```

Alternative: download three free lightning SVGs (e.g., from `https://iconify.design`), tint to `#2979FF`, `#FFC400`, `#D50000`, then convert via an online .ico converter.

Verify:
```bash
ls -la assets/icons/
```
Expected: three `.ico` files, each ~1-10 KB.

- [ ] **Step 2: Write icons.go with go:embed**

`internal/ui/tray/icons.go`:
```go
package tray

import (
	_ "embed"

	"github.com/<your-username>/speedforce/internal/core"
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
```

- [ ] **Step 3: Copy icons into embed path**

```bash
mkdir -p internal/ui/tray/icons
cp assets/icons/lightning-*.ico internal/ui/tray/icons/
```

- [ ] **Step 4: Build to confirm embed compiles**

```bash
go build ./internal/ui/tray/...
```
Expected: no output (success)

- [ ] **Step 5: Commit**

```bash
git add assets/icons/ internal/ui/tray/icons/ internal/ui/tray/icons.go
git commit -m "feat(ui): embed blue/yellow/red lightning tray icons"
```

---

### Task 6: Minimal tray with color switching and main entry point

**Files:**
- Create: `internal/ui/tray/tray.go`
- Create: `cmd/speedforce/main.go`

- [ ] **Step 1: Add systray dependency**

```bash
go get github.com/getlantern/systray
```

- [ ] **Step 2: Write tray.go**

`internal/ui/tray/tray.go`:
```go
package tray

import (
	"sync"

	"github.com/getlantern/systray"
	"github.com/<your-username>/speedforce/internal/core"
)

type Tray struct {
	mu       sync.Mutex
	onQuit   func()
	onDetail func()
}

func New(onDetail, onQuit func()) *Tray {
	return &Tray{onDetail: onDetail, onQuit: onQuit}
}

func (t *Tray) Run() {
	systray.Run(t.onReady, t.onExit)
}

func (t *Tray) onReady() {
	systray.SetIcon(IconFor(core.StatusUnknown))
	systray.SetTooltip("SpeedForce")

	mDetail := systray.AddMenuItem("Show Details", "Open detail window")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Exit SpeedForce")

	go func() {
		for {
			select {
			case <-mDetail.ClickedCh:
				if t.onDetail != nil {
					t.onDetail()
				}
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func (t *Tray) onExit() {
	if t.onQuit != nil {
		t.onQuit()
	}
}

func (t *Tray) SetStatus(status core.OverallStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()
	systray.SetIcon(IconFor(status))
}
```

- [ ] **Step 3: Write main.go wiring a single HTTPS probe loop**

`cmd/speedforce/main.go`:
```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/<your-username>/speedforce/internal/core"
	"github.com/<your-username>/speedforce/internal/probe"
	"github.com/<your-username>/speedforce/internal/ui/tray"
)

func main() {
	client := probe.NewClient(probe.ClientOptions{
		Timeout:   5 * time.Second,
		ProxyMode: string(probe.ProxyAuto),
	})
	prober := probe.NewHTTPSProber(client)

	t := tray.New(
		func() { log.Println("Detail requested (window not implemented yet)") },
		func() { log.Println("Quit") },
	)

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		tick := func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			res := prober.Probe(ctx, "Claude", "https://claude.ai")
			status := core.StatusHealthy
			if !res.IsUp() {
				status = core.StatusDown
			}
			t.SetStatus(status)
			log.Printf("probe: up=%v latency=%dms err=%v", res.IsUp(), res.LatencyMs, res.Err)
		}
		tick()
		for range ticker.C {
			tick()
		}
	}()

	t.Run()
}
```

- [ ] **Step 4: Build and run**

```bash
go mod tidy
go build -o speedforce.exe ./cmd/speedforce
./speedforce.exe
```
Expected: tray icon appears; check log output every 30s; icon stays blue while network is OK, turns red if you disconnect.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/tray/tray.go cmd/speedforce/main.go go.mod go.sum
git commit -m "feat: minimal tray app with one HTTPS probe loop (M1 complete)"
```

---

## M2 — Core Complete

### Task 7: StateBus (pub/sub state store)

**Files:**
- Create: `internal/core/statebus.go`
- Test: `internal/core/statebus_test.go`

- [ ] **Step 1: Write failing test**

`internal/core/statebus_test.go`:
```go
package core

import (
	"sync"
	"testing"
	"time"
)

func TestStateBus_PublishSubscribe(t *testing.T) {
	bus := NewStateBus()
	defer bus.Close()

	sub := bus.Subscribe()

	go func() {
		bus.Publish(State{
			HTTPS:     []ProbeResult{{Name: "Claude", StatusCode: 200}},
			UpdatedAt: time.Now(),
		})
	}()

	select {
	case s := <-sub:
		if len(s.HTTPS) != 1 || s.HTTPS[0].Name != "Claude" {
			t.Errorf("unexpected state: %+v", s)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for state")
	}
}

func TestStateBus_MultipleSubscribers(t *testing.T) {
	bus := NewStateBus()
	defer bus.Close()

	s1 := bus.Subscribe()
	s2 := bus.Subscribe()

	var wg sync.WaitGroup
	wg.Add(2)
	got := make([]State, 2)

	go func() { defer wg.Done(); got[0] = <-s1 }()
	go func() { defer wg.Done(); got[1] = <-s2 }()

	bus.Publish(State{Mode: TickFast})

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}

	for i, g := range got {
		if g.Mode != TickFast {
			t.Errorf("subscriber %d got mode %v", i, g.Mode)
		}
	}
}

func TestStateBus_Snapshot(t *testing.T) {
	bus := NewStateBus()
	defer bus.Close()
	bus.Publish(State{Mode: TickFast, UpdatedAt: time.Now()})
	time.Sleep(20 * time.Millisecond)
	s := bus.Snapshot()
	if s.Mode != TickFast {
		t.Errorf("snapshot mode = %v, want TickFast", s.Mode)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/core/... -run TestStateBus
```
Expected: FAIL — undefined `NewStateBus`

- [ ] **Step 3: Implement StateBus**

`internal/core/statebus.go`:
```go
package core

import "sync"

type StateBus struct {
	mu       sync.RWMutex
	current  State
	subs     []chan State
	closed   bool
}

func NewStateBus() *StateBus {
	return &StateBus{}
}

func (b *StateBus) Subscribe() <-chan State {
	b.mu.Lock()
	defer b.mu.Unlock()
	ch := make(chan State, 8)
	b.subs = append(b.subs, ch)
	return ch
}

func (b *StateBus) Publish(s State) {
	b.mu.Lock()
	b.current = s
	subs := make([]chan State, len(b.subs))
	copy(subs, b.subs)
	b.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- s:
		default:
			// drop if subscriber slow; they'll get the next Publish
		}
	}
}

func (b *StateBus) Snapshot() State {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.current
}

func (b *StateBus) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	for _, ch := range b.subs {
		close(ch)
	}
	b.subs = nil
}
```

- [ ] **Step 4: Run tests with -race**

```bash
go test -race ./internal/core/... -v -run TestStateBus
```
Expected: PASS, no race detected

- [ ] **Step 5: Commit**

```bash
git add internal/core/statebus.go internal/core/statebus_test.go
git commit -m "feat(core): add StateBus pub/sub"
```

---

### Task 8: IPProber

**Files:**
- Create: `internal/probe/ip.go`
- Test: `internal/probe/ip_test.go`

- [ ] **Step 1: Write failing test**

`internal/probe/ip_test.go`:
```go
package probe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIPProber_FetchesPublicAndGeo(t *testing.T) {
	ipSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ip":"203.0.113.1"}`))
	}))
	defer ipSrv.Close()

	geoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"country_name":"United States","city":"San Francisco","org":"Cloudflare"}`))
	}))
	defer geoSrv.Close()

	p := NewIPProber(
		NewClient(ClientOptions{Timeout: 2 * time.Second, ProxyMode: "none"}),
		ipSrv.URL,
		geoSrv.URL+"/{ip}",
	)

	info, err := p.Probe(context.Background())
	if err != nil {
		t.Fatalf("probe err: %v", err)
	}
	if info.PublicIP != "203.0.113.1" {
		t.Errorf("ip = %q", info.PublicIP)
	}
	if info.Country != "United States" {
		t.Errorf("country = %q", info.Country)
	}
	if info.City != "San Francisco" {
		t.Errorf("city = %q", info.City)
	}
	if info.ISP != "Cloudflare" {
		t.Errorf("isp = %q", info.ISP)
	}
	if info.LANIP == "" {
		t.Error("LANIP should be set")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/probe/... -run TestIPProber
```
Expected: FAIL — undefined `NewIPProber`

- [ ] **Step 3: Implement IPProber**

`internal/probe/ip.go`:
```go
package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/<your-username>/speedforce/internal/core"
)

type IPProber struct {
	client       *http.Client
	publicIPURL  string
	geoURLTmpl   string // contains "{ip}" placeholder
}

func NewIPProber(client *http.Client, publicIPURL, geoURLTmpl string) *IPProber {
	return &IPProber{client: client, publicIPURL: publicIPURL, geoURLTmpl: geoURLTmpl}
}

func (p *IPProber) Probe(ctx context.Context) (core.IPInfo, error) {
	info := core.IPInfo{LANIP: firstLANIP(), FetchedAt: time.Now()}

	ip, err := p.fetchPublicIP(ctx)
	if err != nil {
		return info, fmt.Errorf("public ip: %w", err)
	}
	info.PublicIP = ip

	geo, err := p.fetchGeo(ctx, ip)
	if err != nil {
		return info, fmt.Errorf("geo: %w", err)
	}
	info.Country = geo.Country
	info.City = geo.City
	info.ISP = geo.Org
	return info, nil
}

type ipifyResp struct {
	IP string `json:"ip"`
}

type geoResp struct {
	Country string `json:"country_name"`
	City    string `json:"city"`
	Org     string `json:"org"`
}

func (p *IPProber) fetchPublicIP(ctx context.Context) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, p.publicIPURL, nil)
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var r ipifyResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", err
	}
	return r.IP, nil
}

func (p *IPProber) fetchGeo(ctx context.Context, ip string) (geoResp, error) {
	url := strings.Replace(p.geoURLTmpl, "{ip}", ip, 1)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := p.client.Do(req)
	if err != nil {
		return geoResp{}, err
	}
	defer resp.Body.Close()
	var r geoResp
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return geoResp{}, err
	}
	return r, nil
}

func firstLANIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}
		if ip4 := ipNet.IP.To4(); ip4 != nil {
			return ip4.String()
		}
	}
	return ""
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
go test ./internal/probe/... -v -run TestIPProber
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/probe/ip.go internal/probe/ip_test.go
git commit -m "feat(probe): add IP info prober (public + LAN + geo)"
```

---

### Task 9: StatuspageProber

**Files:**
- Create: `internal/probe/statuspage.go`
- Test: `internal/probe/statuspage_test.go`
- Create: `internal/probe/testdata/statuspage_operational.json`
- Create: `internal/probe/testdata/statuspage_major.json`

- [ ] **Step 1: Create fixtures**

`internal/probe/testdata/statuspage_operational.json`:
```json
{
  "page": {"id": "abc", "name": "Anthropic"},
  "status": {"indicator": "none", "description": "All Systems Operational"}
}
```

`internal/probe/testdata/statuspage_major.json`:
```json
{
  "page": {"id": "def", "name": "OpenAI"},
  "status": {"indicator": "major", "description": "Major service disruption"}
}
```

- [ ] **Step 2: Write failing test**

`internal/probe/statuspage_test.go`:
```go
package probe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/<your-username>/speedforce/internal/core"
)

func serveFile(t *testing.T, path string) *httptest.Server {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
}

func TestStatuspageProber_Operational(t *testing.T) {
	srv := serveFile(t, "testdata/statuspage_operational.json")
	defer srv.Close()

	p := NewStatuspageProber(NewClient(ClientOptions{Timeout: 2 * time.Second, ProxyMode: "none"}))
	res := p.Probe(context.Background(), "Anthropic", srv.URL)
	if res.Indicator != core.StatuspageOperational {
		t.Errorf("indicator = %q, want operational(none)", res.Indicator)
	}
	if res.Description == "" {
		t.Error("description empty")
	}
}

func TestStatuspageProber_MajorOutage(t *testing.T) {
	srv := serveFile(t, "testdata/statuspage_major.json")
	defer srv.Close()

	p := NewStatuspageProber(NewClient(ClientOptions{Timeout: 2 * time.Second, ProxyMode: "none"}))
	res := p.Probe(context.Background(), "OpenAI", srv.URL)
	if res.Indicator != core.StatuspageMajor {
		t.Errorf("indicator = %q, want major", res.Indicator)
	}
}

func TestStatuspageProber_FetchError(t *testing.T) {
	p := NewStatuspageProber(NewClient(ClientOptions{Timeout: 200 * time.Millisecond, ProxyMode: "none"}))
	res := p.Probe(context.Background(), "x", "http://127.0.0.1:1") // port 1 refuses
	if res.Err == nil {
		t.Error("expected error for unreachable endpoint")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./internal/probe/... -run TestStatuspageProber
```
Expected: FAIL — undefined `NewStatuspageProber`

- [ ] **Step 4: Implement**

`internal/probe/statuspage.go`:
```go
package probe

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/<your-username>/speedforce/internal/core"
)

type StatuspageProber struct {
	client *http.Client
}

func NewStatuspageProber(client *http.Client) *StatuspageProber {
	return &StatuspageProber{client: client}
}

type statuspageDoc struct {
	Status struct {
		Indicator   string `json:"indicator"`
		Description string `json:"description"`
	} `json:"status"`
}

func (p *StatuspageProber) Probe(ctx context.Context, name, url string) core.StatuspageResult {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := p.client.Do(req)
	if err != nil {
		return core.StatuspageResult{Name: name, Err: err, FetchedAt: time.Now()}
	}
	defer resp.Body.Close()

	var doc statuspageDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return core.StatuspageResult{Name: name, Err: err, FetchedAt: time.Now()}
	}
	return core.StatuspageResult{
		Name:        name,
		Indicator:   core.StatuspageIndicator(doc.Status.Indicator),
		Description: doc.Status.Description,
		FetchedAt:   time.Now(),
	}
}
```

- [ ] **Step 5: Run tests and commit**

```bash
go test ./internal/probe/... -v -run TestStatuspageProber
git add internal/probe/statuspage.go internal/probe/statuspage_test.go internal/probe/testdata/
git commit -m "feat(probe): add Statuspage.io prober"
```

---

### Task 10: Config types and defaults

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/defaults.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: Add YAML dependency**

```bash
go get github.com/goccy/go-yaml
```

- [ ] **Step 2: Write failing test**

`internal/config/config_test.go`:
```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_UsesDefaultsWhenMissing(t *testing.T) {
	dir := t.TempDir()
	cfg, err := Load(filepath.Join(dir, "missing.yaml"))
	if err != nil {
		t.Fatalf("load err: %v", err)
	}
	if cfg.Network.TimeoutMs != 5000 {
		t.Errorf("timeout default = %d, want 5000", cfg.Network.TimeoutMs)
	}
	if len(cfg.Probes.HTTPS) != 6 {
		t.Errorf("got %d HTTPS probes, want 6 (default set)", len(cfg.Probes.HTTPS))
	}
}

func TestLoad_ParsesYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.yaml")
	err := os.WriteFile(path, []byte(`
version: 1
language: en
network:
  proxy:
    mode: manual
    manual_url: http://127.0.0.1:7890
  timeout_ms: 3000
  tick_interval_sec: 30
  adaptive_tick:
    enabled: false
    fast_interval_sec: 10
    threshold_ms: 1000
    recovery_threshold_ms: 300
    max_fast_duration_sec: 600
probes:
  https:
    - name: Test
      url: https://example.com
      notify_on_down: true
  statuspage:
    refresh_interval_sec: 300
    notify_on_major_outage: true
    sources: []
  ip:
    public_ip_api: https://api.ipify.org?format=json
    geo_api: https://ipapi.co/{ip}/json/
    refresh_every_ticks: 5
ui:
  auto_start: true
  minimize_to_tray_on_close: true
  theme: dark
logging:
  level: debug
  retention_days: 7
`), 0600)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Language != "en" {
		t.Errorf("language = %q", cfg.Language)
	}
	if cfg.Network.Proxy.Mode != "manual" {
		t.Errorf("mode = %q", cfg.Network.Proxy.Mode)
	}
	if cfg.UI.Theme != "dark" {
		t.Errorf("theme = %q", cfg.UI.Theme)
	}
}

func TestLoad_RollbackOnInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	os.WriteFile(path, []byte("::: not yaml :::"), 0600)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected parse error")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./internal/config/...
```
Expected: FAIL — undefined `Load`

- [ ] **Step 4: Implement config.go**

`internal/config/config.go`:
```go
package config

import (
	"errors"
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Version  int      `yaml:"version"`
	Language string   `yaml:"language"`
	Network  Network  `yaml:"network"`
	Probes   Probes   `yaml:"probes"`
	UI       UI       `yaml:"ui"`
	Logging  Logging  `yaml:"logging"`
}

type Network struct {
	Proxy        Proxy        `yaml:"proxy"`
	TimeoutMs    int          `yaml:"timeout_ms"`
	TickInterval int          `yaml:"tick_interval_sec"`
	AdaptiveTick AdaptiveTick `yaml:"adaptive_tick"`
}

type Proxy struct {
	Mode      string `yaml:"mode"`
	ManualURL string `yaml:"manual_url"`
}

type AdaptiveTick struct {
	Enabled             bool `yaml:"enabled"`
	FastIntervalSec     int  `yaml:"fast_interval_sec"`
	ThresholdMs         int  `yaml:"threshold_ms"`
	RecoveryThresholdMs int  `yaml:"recovery_threshold_ms"`
	MaxFastDurationSec  int  `yaml:"max_fast_duration_sec"`
}

type Probes struct {
	HTTPS      []HTTPSProbe     `yaml:"https"`
	Statuspage StatuspageBlock  `yaml:"statuspage"`
	IP         IPBlock          `yaml:"ip"`
}

type HTTPSProbe struct {
	Name         string `yaml:"name"`
	URL          string `yaml:"url"`
	NotifyOnDown bool   `yaml:"notify_on_down"`
}

type StatuspageBlock struct {
	RefreshIntervalSec   int                 `yaml:"refresh_interval_sec"`
	NotifyOnMajorOutage  bool                `yaml:"notify_on_major_outage"`
	Sources              []StatuspageSource  `yaml:"sources"`
}

type StatuspageSource struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type IPBlock struct {
	PublicIPAPI       string `yaml:"public_ip_api"`
	GeoAPI            string `yaml:"geo_api"`
	RefreshEveryTicks int    `yaml:"refresh_every_ticks"`
}

type UI struct {
	AutoStart              bool   `yaml:"auto_start"`
	MinimizeToTrayOnClose  bool   `yaml:"minimize_to_tray_on_close"`
	Theme                  string `yaml:"theme"`
}

type Logging struct {
	Level         string `yaml:"level"`
	RetentionDays int    `yaml:"retention_days"`
}

func Load(path string) (*Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if writeErr := os.WriteFile(path, []byte(DefaultYAML), 0600); writeErr != nil {
				return cfg, fmt.Errorf("write default: %w", writeErr)
			}
			return cfg, nil
		}
		return cfg, err
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	return cfg, nil
}
```

- [ ] **Step 5: Implement defaults.go**

`internal/config/defaults.go`:
```go
package config

const DefaultYAML = `version: 1
language: zh

network:
  proxy:
    mode: auto
    manual_url: ""
  timeout_ms: 5000
  tick_interval_sec: 60
  adaptive_tick:
    enabled: true
    fast_interval_sec: 10
    threshold_ms: 1000
    recovery_threshold_ms: 300
    max_fast_duration_sec: 600

probes:
  https:
    - name: Claude API
      url: https://api.anthropic.com
      notify_on_down: true
    - name: Claude Web
      url: https://claude.ai
      notify_on_down: false
    - name: OpenAI API
      url: https://api.openai.com
      notify_on_down: true
    - name: ChatGPT Web
      url: https://chatgpt.com
      notify_on_down: false
    - name: Gemini API
      url: https://generativelanguage.googleapis.com
      notify_on_down: true
    - name: Gemini Web
      url: https://gemini.google.com
      notify_on_down: false

  statuspage:
    refresh_interval_sec: 300
    notify_on_major_outage: true
    sources:
      - name: Anthropic
        url: https://status.anthropic.com/api/v2/status.json
      - name: OpenAI
        url: https://status.openai.com/api/v2/status.json

  ip:
    public_ip_api: https://api.ipify.org?format=json
    geo_api: https://ipapi.co/{ip}/json/
    refresh_every_ticks: 5

ui:
  auto_start: false
  minimize_to_tray_on_close: true
  theme: auto

logging:
  level: info
  retention_days: 30
`

func Default() *Config {
	cfg := &Config{}
	if err := yamlUnmarshal([]byte(DefaultYAML), cfg); err != nil {
		panic("default YAML invalid: " + err.Error())
	}
	return cfg
}
```

And add helper at the top (or inline the import):

```go
package config

import "github.com/goccy/go-yaml"

func yamlUnmarshal(b []byte, v any) error { return yaml.Unmarshal(b, v) }
```

Or simply use `yaml.Unmarshal` directly in `Default()`.

- [ ] **Step 6: Run tests and commit**

```bash
go test ./internal/config/... -v
git add internal/config/ go.mod go.sum
git commit -m "feat(config): YAML loading with defaults"
```

---

### Task 11: Scheduler (fixed tick, dispatches probes)

**Files:**
- Create: `internal/core/scheduler.go`
- Test: `internal/core/scheduler_test.go`

- [ ] **Step 1: Write failing test**

`internal/core/scheduler_test.go`:
```go
package core

import (
	"context"
	"sync"
	"testing"
	"time"
)

type fakeProber struct {
	mu    sync.Mutex
	calls int
	latencyMs int64
}

func (f *fakeProber) Probe(ctx context.Context, name, url string) ProbeResult {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return ProbeResult{Name: name, URL: url, StatusCode: 200, LatencyMs: f.latencyMs}
}

func TestScheduler_TickFiresProbes(t *testing.T) {
	bus := NewStateBus()
	defer bus.Close()

	fp := &fakeProber{}
	sch := NewScheduler(SchedulerConfig{
		Bus: bus,
		HTTPS: fp,
		HTTPSTargets: []Target{{Name: "a", URL: "http://a"}, {Name: "b", URL: "http://b"}},
		TickInterval: 50 * time.Millisecond,
		AdaptiveEnabled: false,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sch.Run(ctx)

	time.Sleep(140 * time.Millisecond)
	fp.mu.Lock()
	if fp.calls < 4 { // 2 targets × ~2 ticks
		t.Errorf("expected >=4 probe calls, got %d", fp.calls)
	}
	fp.mu.Unlock()
}
```

- [ ] **Step 2: Run test, expect fail**

```bash
go test ./internal/core/... -run TestScheduler
```
Expected: FAIL — undefined `NewScheduler`

- [ ] **Step 3: Implement scheduler with fixed-tick-only (no adaptive yet)**

`internal/core/scheduler.go`:
```go
package core

import (
	"context"
	"sync"
	"time"
)

type HTTPSProbeFunc interface {
	Probe(ctx context.Context, name, url string) ProbeResult
}

type Target struct {
	Name string
	URL  string
}

type SchedulerConfig struct {
	Bus          *StateBus
	HTTPS        HTTPSProbeFunc
	HTTPSTargets []Target

	TickInterval    time.Duration
	FastInterval    time.Duration
	AdaptiveEnabled bool
	ThresholdMs     int64
	RecoveryMs      int64
	MaxFastDuration time.Duration
}

type Scheduler struct {
	cfg SchedulerConfig

	mu             sync.Mutex
	mode           TickMode
	fastSince      time.Time
	stableTicks    int
	lastLatencies  map[string]int64
}

func NewScheduler(cfg SchedulerConfig) *Scheduler {
	return &Scheduler{
		cfg:           cfg,
		mode:          TickNormal,
		lastLatencies: make(map[string]int64),
	}
}

func (s *Scheduler) currentInterval() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mode == TickFast {
		return s.cfg.FastInterval
	}
	return s.cfg.TickInterval
}

func (s *Scheduler) Run(ctx context.Context) {
	timer := time.NewTimer(s.currentInterval())
	defer timer.Stop()

	s.doTick(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			s.doTick(ctx)
			timer.Reset(s.currentInterval())
		}
	}
}

func (s *Scheduler) doTick(ctx context.Context) {
	var wg sync.WaitGroup
	results := make([]ProbeResult, len(s.cfg.HTTPSTargets))
	for i, t := range s.cfg.HTTPSTargets {
		wg.Add(1)
		go func(i int, t Target) {
			defer wg.Done()
			results[i] = s.cfg.HTTPS.Probe(ctx, t.Name, t.URL)
		}(i, t)
	}
	wg.Wait()

	s.updateAdaptive(results)

	s.cfg.Bus.Publish(State{
		HTTPS:     results,
		Mode:      s.currentMode(),
		UpdatedAt: time.Now(),
	})
}

func (s *Scheduler) currentMode() TickMode {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mode
}

func (s *Scheduler) updateAdaptive(results []ProbeResult) {
	// Full adaptive logic is added in Task 12
}
```

- [ ] **Step 4: Run test, expect pass**

```bash
go test -race ./internal/core/... -v -run TestScheduler
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/core/scheduler.go internal/core/scheduler_test.go
git commit -m "feat(core): basic scheduler with fixed tick interval"
```

---

### Task 12: Adaptive tick logic

**Files:**
- Modify: `internal/core/scheduler.go` (replace `updateAdaptive`)
- Test: `internal/core/scheduler_adaptive_test.go`

- [ ] **Step 1: Write failing test**

`internal/core/scheduler_adaptive_test.go`:
```go
package core

import (
	"testing"
	"time"
)

func newSchedulerForAdaptive() *Scheduler {
	return NewScheduler(SchedulerConfig{
		TickInterval:    60 * time.Second,
		FastInterval:    10 * time.Second,
		AdaptiveEnabled: true,
		ThresholdMs:     1000,
		RecoveryMs:      300,
		MaxFastDuration: 10 * time.Minute,
	})
}

func TestAdaptive_EnterFastOnLatencySpike(t *testing.T) {
	s := newSchedulerForAdaptive()
	s.lastLatencies["a"] = 100

	s.updateAdaptive([]ProbeResult{{Name: "a", URL: "x", StatusCode: 200, LatencyMs: 1500}})

	if s.currentMode() != TickFast {
		t.Error("should enter fast mode on >1000ms delta")
	}
}

func TestAdaptive_EnterFastOnStateFlip(t *testing.T) {
	s := newSchedulerForAdaptive()
	s.lastLatencies["a"] = 100

	// previous was up (StatusCode 200 in map), now down
	s.updateAdaptive([]ProbeResult{{Name: "a", URL: "x", StatusCode: 0, Err: &testErr{"timeout"}}})

	if s.currentMode() != TickFast {
		t.Error("should enter fast mode on up→down transition")
	}
}

func TestAdaptive_ExitAfterStableTicks(t *testing.T) {
	s := newSchedulerForAdaptive()
	s.mode = TickFast
	s.fastSince = time.Now()
	s.lastLatencies["a"] = 100

	for i := 0; i < 3; i++ {
		s.updateAdaptive([]ProbeResult{{Name: "a", URL: "x", StatusCode: 200, LatencyMs: 120}})
	}

	if s.currentMode() != TickNormal {
		t.Error("should exit fast mode after 3 stable ticks")
	}
}

func TestAdaptive_MaxFastDurationCap(t *testing.T) {
	s := newSchedulerForAdaptive()
	s.mode = TickFast
	s.fastSince = time.Now().Add(-11 * time.Minute) // beyond 10min cap
	s.lastLatencies["a"] = 100

	s.updateAdaptive([]ProbeResult{{Name: "a", URL: "x", StatusCode: 200, LatencyMs: 1500}}) // would normally keep fast

	if s.currentMode() != TickNormal {
		t.Error("should forcibly exit after max_fast_duration")
	}
}
```

Add to `scheduler_test.go` or adjacent:
```go
type testErr struct{ msg string }

func (e *testErr) Error() string { return e.msg }
```

(If already defined in `types_test.go`, skip.)

- [ ] **Step 2: Run test, expect fail**

```bash
go test ./internal/core/... -run TestAdaptive
```
Expected: FAIL — state never transitions

- [ ] **Step 3: Implement updateAdaptive**

Replace the stub in `internal/core/scheduler.go`:
```go
func (s *Scheduler) updateAdaptive(results []ProbeResult) {
	if !s.cfg.AdaptiveEnabled {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check max-fast-duration cap first
	if s.mode == TickFast && !s.fastSince.IsZero() &&
		time.Since(s.fastSince) >= s.cfg.MaxFastDuration {
		s.mode = TickNormal
		s.fastSince = time.Time{}
		s.stableTicks = 0
		// record latencies before returning so next tick has a reference
		s.recordLatencies(results)
		return
	}

	trigger := false
	allStable := true

	for _, r := range results {
		prev, seen := s.lastLatencies[r.Name]
		curr := r.LatencyMs

		if seen {
			delta := absInt64(curr - prev)
			if delta > s.cfg.ThresholdMs {
				trigger = true
			}
			if delta >= s.cfg.RecoveryMs {
				allStable = false
			}

			// State flip check
			prevUp := prev > 0 // we only record successful latencies; treat 0 as unknown/down
			currUp := (ProbeResult{StatusCode: r.StatusCode, Err: r.Err}).IsUp()
			if prevUp != currUp {
				trigger = true
			}
		} else {
			// first seen; don't flap
			allStable = false
		}
	}

	switch s.mode {
	case TickNormal:
		if trigger {
			s.mode = TickFast
			s.fastSince = time.Now()
			s.stableTicks = 0
		}
	case TickFast:
		if trigger {
			s.stableTicks = 0
		} else if allStable {
			s.stableTicks++
			if s.stableTicks >= 3 {
				s.mode = TickNormal
				s.fastSince = time.Time{}
				s.stableTicks = 0
			}
		} else {
			s.stableTicks = 0
		}
	}

	s.recordLatencies(results)
}

func (s *Scheduler) recordLatencies(results []ProbeResult) {
	for _, r := range results {
		if r.Err == nil && r.LatencyMs > 0 {
			s.lastLatencies[r.Name] = r.LatencyMs
		} else {
			// mark as "was down" with 0
			s.lastLatencies[r.Name] = 0
		}
	}
}

func absInt64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
```

- [ ] **Step 4: Run tests, expect pass**

```bash
go test -race ./internal/core/... -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/core/scheduler.go internal/core/scheduler_adaptive_test.go
git commit -m "feat(core): adaptive tick state machine (normal↔fast)"
```

---

### Task 13: Extend Scheduler to also run IP + Statuspage probes

**Files:**
- Modify: `internal/core/scheduler.go`
- Test: `internal/core/scheduler_test.go` (add test)

- [ ] **Step 1: Add test**

Append to `internal/core/scheduler_test.go`:
```go
type fakeIPProber struct{ calls int }

func (f *fakeIPProber) Probe(ctx context.Context) (IPInfo, error) {
	f.calls++
	return IPInfo{PublicIP: "1.2.3.4"}, nil
}

type fakeStatuspageProber struct{ calls int }

func (f *fakeStatuspageProber) Probe(ctx context.Context, name, url string) StatuspageResult {
	f.calls++
	return StatuspageResult{Name: name, Indicator: StatuspageOperational}
}

func TestScheduler_IPRunsEveryNTicks(t *testing.T) {
	bus := NewStateBus()
	defer bus.Close()

	fp := &fakeProber{}
	fip := &fakeIPProber{}
	sch := NewScheduler(SchedulerConfig{
		Bus:                bus,
		HTTPS:              fp,
		HTTPSTargets:       []Target{{Name: "a", URL: "http://a"}},
		IP:                 fip,
		IPRefreshEveryTicks: 3,
		TickInterval:       30 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sch.Run(ctx)

	time.Sleep(200 * time.Millisecond)
	if fip.calls < 2 { // ~6 ticks / 3 = 2 IP probes minimum
		t.Errorf("IP probe calls = %d, want >= 2", fip.calls)
	}
	if fip.calls >= 6 {
		t.Errorf("IP probe ran too often: %d", fip.calls)
	}
}
```

- [ ] **Step 2: Run, expect fail**

```bash
go test ./internal/core/... -run TestScheduler_IPRunsEveryNTicks
```
Expected: FAIL — `IP` field missing on `SchedulerConfig`

- [ ] **Step 3: Extend SchedulerConfig and Run loop**

Edit `internal/core/scheduler.go`:

Add to `SchedulerConfig`:
```go
type IPProbeFunc interface {
	Probe(ctx context.Context) (IPInfo, error)
}

type StatuspageProbeFunc interface {
	Probe(ctx context.Context, name, url string) StatuspageResult
}

type StatuspageTarget struct {
	Name string
	URL  string
}
```

Add fields to `SchedulerConfig`:
```go
	IP                   IPProbeFunc
	IPRefreshEveryTicks  int

	Statuspage              StatuspageProbeFunc
	StatuspageTargets       []StatuspageTarget
	StatuspageIntervalSec   int
```

Add tick counter to Scheduler:
```go
type Scheduler struct {
	// ...existing fields...
	tickNum          int64
	lastStatuspageAt time.Time
	lastIP           IPInfo
	lastStatuspage   []StatuspageResult
}
```

Extend `doTick`:
```go
func (s *Scheduler) doTick(ctx context.Context) {
	s.mu.Lock()
	s.tickNum++
	tick := s.tickNum
	s.mu.Unlock()

	var wg sync.WaitGroup
	httpsResults := make([]ProbeResult, len(s.cfg.HTTPSTargets))
	for i, t := range s.cfg.HTTPSTargets {
		wg.Add(1)
		go func(i int, t Target) {
			defer wg.Done()
			httpsResults[i] = s.cfg.HTTPS.Probe(ctx, t.Name, t.URL)
		}(i, t)
	}

	shouldIP := s.cfg.IP != nil && s.cfg.IPRefreshEveryTicks > 0 &&
		(tick == 1 || tick%int64(s.cfg.IPRefreshEveryTicks) == 0)
	if shouldIP {
		wg.Add(1)
		go func() {
			defer wg.Done()
			info, err := s.cfg.IP.Probe(ctx)
			if err == nil {
				s.mu.Lock()
				s.lastIP = info
				s.mu.Unlock()
			}
		}()
	}

	shouldStatuspage := s.cfg.Statuspage != nil &&
		(s.lastStatuspageAt.IsZero() ||
			time.Since(s.lastStatuspageAt) >= time.Duration(s.cfg.StatuspageIntervalSec)*time.Second)
	if shouldStatuspage {
		spResults := make([]StatuspageResult, len(s.cfg.StatuspageTargets))
		for i, t := range s.cfg.StatuspageTargets {
			wg.Add(1)
			go func(i int, t StatuspageTarget) {
				defer wg.Done()
				spResults[i] = s.cfg.Statuspage.Probe(ctx, t.Name, t.URL)
			}(i, t)
		}
		wg.Wait()
		s.mu.Lock()
		s.lastStatuspage = spResults
		s.lastStatuspageAt = time.Now()
		s.mu.Unlock()
	} else {
		wg.Wait()
	}

	s.updateAdaptive(httpsResults)

	s.mu.Lock()
	ip := s.lastIP
	sp := append([]StatuspageResult(nil), s.lastStatuspage...)
	s.mu.Unlock()

	s.cfg.Bus.Publish(State{
		HTTPS:      httpsResults,
		IP:         ip,
		Statuspage: sp,
		Mode:       s.currentMode(),
		UpdatedAt:  time.Now(),
	})
}
```

- [ ] **Step 4: Run tests**

```bash
go test -race ./internal/core/... -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/core/scheduler.go internal/core/scheduler_test.go
git commit -m "feat(core): scheduler runs IP (N-tick) and Statuspage (interval) probes"
```

---

### Task 14: Platform-aware system proxy detection (Windows)

**Files:**
- Create: `internal/platform/proxy_windows.go`
- Create: `internal/platform/proxy_darwin.go`
- Create: `internal/platform/proxy_linux.go`
- Create: `internal/platform/proxy.go` (cross-platform interface)
- Test: `internal/platform/proxy_windows_test.go`

- [ ] **Step 1: Write cross-platform interface**

`internal/platform/proxy.go`:
```go
package platform

import (
	"net/http"
	"net/url"
)

// SystemProxyFunc returns a function suitable for use as http.Transport.Proxy.
// When no proxy is configured or detection fails, returns a nil-proxy function.
func SystemProxyFunc() func(*http.Request) (*url.URL, error) {
	return systemProxyImpl()
}
```

- [ ] **Step 2: Windows implementation**

`internal/platform/proxy_windows.go`:
```go
//go:build windows

package platform

import (
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func systemProxyImpl() func(*http.Request) (*url.URL, error) {
	return func(req *http.Request) (*url.URL, error) {
		return readWindowsProxy()
	}
}

func readWindowsProxy() (*url.URL, error) {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Internet Settings`, registry.QUERY_VALUE)
	if err != nil {
		return nil, nil
	}
	defer k.Close()

	enabled, _, err := k.GetIntegerValue("ProxyEnable")
	if err != nil || enabled == 0 {
		return nil, nil
	}
	server, _, err := k.GetStringValue("ProxyServer")
	if err != nil || server == "" {
		return nil, nil
	}
	// ProxyServer may be "host:port" or "http=host:port;https=host:port"
	if strings.Contains(server, "=") {
		for _, part := range strings.Split(server, ";") {
			if strings.HasPrefix(part, "https=") {
				return url.Parse("http://" + strings.TrimPrefix(part, "https="))
			}
		}
		// fall back to http=
		for _, part := range strings.Split(server, ";") {
			if strings.HasPrefix(part, "http=") {
				return url.Parse("http://" + strings.TrimPrefix(part, "http="))
			}
		}
	}
	return url.Parse("http://" + server)
}
```

Install dep:
```bash
go get golang.org/x/sys/windows/registry
```

- [ ] **Step 3: Darwin and Linux stubs**

`internal/platform/proxy_darwin.go`:
```go
//go:build darwin

package platform

import (
	"net/http"
	"net/url"
)

func systemProxyImpl() func(*http.Request) (*url.URL, error) {
	// TODO: macOS implementation (future milestone)
	return http.ProxyFromEnvironment
}
```

`internal/platform/proxy_linux.go`:
```go
//go:build linux

package platform

import (
	"net/http"
	"net/url"
)

func systemProxyImpl() func(*http.Request) (*url.URL, error) {
	return http.ProxyFromEnvironment
}
```

- [ ] **Step 4: Windows test**

`internal/platform/proxy_windows_test.go`:
```go
//go:build windows

package platform

import (
	"testing"
)

func TestSystemProxyFunc_ReturnsFunction(t *testing.T) {
	fn := SystemProxyFunc()
	if fn == nil {
		t.Fatal("got nil proxy func")
	}
	// Should not panic regardless of registry state
	_, _ = fn(nil)
}
```

- [ ] **Step 5: Build all platforms and commit**

```bash
go build ./internal/platform/...
GOOS=darwin go build ./internal/platform/...
GOOS=linux go build ./internal/platform/...
go test ./internal/platform/...

git add internal/platform/ go.mod go.sum
git commit -m "feat(platform): system proxy detection (Windows registry; stubs for darwin/linux)"
```

**M2 complete.** Confirm by running the full test suite:
```bash
go test -race ./...
```

---

## M3 — UI

### Task 15: i18n module

**Files:**
- Create: `internal/i18n/i18n.go`
- Create: `internal/i18n/locales/zh.json`
- Create: `internal/i18n/locales/en.json`
- Test: `internal/i18n/i18n_test.go`

- [ ] **Step 1: Create locale files**

`internal/i18n/locales/zh.json`:
```json
{
  "app_name": "SpeedForce",
  "tray.tooltip": "SpeedForce — 网络状态",
  "tray.menu.show_details": "显示详情",
  "tray.menu.settings": "设置",
  "tray.menu.quit": "退出",
  "detail.title": "SpeedForce 详情",
  "detail.section.ip": "IP 信息",
  "detail.section.probes": "连接状态",
  "detail.section.statuspage": "官方服务状态",
  "detail.mode.normal": "🐢 Normal (60s)",
  "detail.mode.fast": "⚡ Fast (10s)",
  "detail.button.refresh": "刷新",
  "detail.button.settings": "设置",
  "detail.button.open_gemini_status": "查看 Gemini 官方状态",
  "settings.title": "SpeedForce 设置",
  "settings.language": "界面语言",
  "settings.proxy": "代理设置",
  "settings.proxy.mode": "模式",
  "settings.proxy.mode.auto": "自动（系统代理）",
  "settings.proxy.mode.manual": "手动",
  "settings.proxy.mode.none": "禁用",
  "settings.proxy.url": "代理地址",
  "settings.tick_interval": "检测频率（秒）",
  "settings.notifications": "通知",
  "settings.autostart": "开机自启动",
  "settings.export_logs": "导出日志",
  "settings.save": "保存",
  "settings.cancel": "取消",
  "notify.service_down": "{{name}} 连接失败",
  "notify.statuspage_major": "{{name}} 报告重大故障",
  "notify.gemini_down": "Gemini 连接失败（我方探测）"
}
```

`internal/i18n/locales/en.json`:
```json
{
  "app_name": "SpeedForce",
  "tray.tooltip": "SpeedForce — Network Status",
  "tray.menu.show_details": "Show Details",
  "tray.menu.settings": "Settings",
  "tray.menu.quit": "Quit",
  "detail.title": "SpeedForce Details",
  "detail.section.ip": "IP Information",
  "detail.section.probes": "Connectivity",
  "detail.section.statuspage": "Official Status",
  "detail.mode.normal": "🐢 Normal (60s)",
  "detail.mode.fast": "⚡ Fast (10s)",
  "detail.button.refresh": "Refresh",
  "detail.button.settings": "Settings",
  "detail.button.open_gemini_status": "View Gemini Official Status",
  "settings.title": "SpeedForce Settings",
  "settings.language": "Language",
  "settings.proxy": "Proxy",
  "settings.proxy.mode": "Mode",
  "settings.proxy.mode.auto": "Auto (system proxy)",
  "settings.proxy.mode.manual": "Manual",
  "settings.proxy.mode.none": "Disabled",
  "settings.proxy.url": "Proxy URL",
  "settings.tick_interval": "Tick Interval (s)",
  "settings.notifications": "Notifications",
  "settings.autostart": "Start with Windows",
  "settings.export_logs": "Export Logs",
  "settings.save": "Save",
  "settings.cancel": "Cancel",
  "notify.service_down": "{{name}} is unreachable",
  "notify.statuspage_major": "{{name}} reports a major outage",
  "notify.gemini_down": "Gemini unreachable (self-probe)"
}
```

- [ ] **Step 2: Write failing test**

`internal/i18n/i18n_test.go`:
```go
package i18n

import "testing"

func TestT_ReturnsKey(t *testing.T) {
	tr, err := New("en")
	if err != nil {
		t.Fatal(err)
	}
	if got := tr.T("app_name"); got != "SpeedForce" {
		t.Errorf("got %q", got)
	}
}

func TestT_ChineseLoad(t *testing.T) {
	tr, err := New("zh")
	if err != nil {
		t.Fatal(err)
	}
	if got := tr.T("tray.menu.quit"); got != "退出" {
		t.Errorf("got %q", got)
	}
}

func TestT_FallbackToEn(t *testing.T) {
	tr, err := New("zz") // unknown locale
	if err != nil {
		t.Fatal(err)
	}
	if got := tr.T("app_name"); got != "SpeedForce" {
		t.Errorf("expected fallback: got %q", got)
	}
}

func TestT_MissingKeyReturnsKey(t *testing.T) {
	tr, _ := New("en")
	if got := tr.T("nonexistent.key"); got != "nonexistent.key" {
		t.Errorf("missing key should return key itself, got %q", got)
	}
}

func TestT_Interpolation(t *testing.T) {
	tr, _ := New("en")
	got := tr.T("notify.service_down", map[string]string{"name": "Claude"})
	if got != "Claude is unreachable" {
		t.Errorf("got %q", got)
	}
}
```

- [ ] **Step 3: Run, expect fail**

```bash
go test ./internal/i18n/...
```
Expected: FAIL — undefined `New`

- [ ] **Step 4: Implement i18n**

`internal/i18n/i18n.go`:
```go
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed locales/*.json
var localesFS embed.FS

type Translator struct {
	mu      sync.RWMutex
	locale  string
	primary map[string]string
	fallback map[string]string
}

func New(locale string) (*Translator, error) {
	fallback, err := loadLocale("en")
	if err != nil {
		return nil, err
	}
	primary := fallback
	if locale != "en" {
		p, err := loadLocale(locale)
		if err == nil {
			primary = p
		}
		// if load fails, primary remains fallback (en)
	}
	return &Translator{
		locale:   locale,
		primary:  primary,
		fallback: fallback,
	}, nil
}

func loadLocale(locale string) (map[string]string, error) {
	data, err := localesFS.ReadFile(fmt.Sprintf("locales/%s.json", locale))
	if err != nil {
		return nil, err
	}
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// T returns translation for key. Optional params map does {{key}} substitution.
func (tr *Translator) T(key string, params ...map[string]string) string {
	tr.mu.RLock()
	val, ok := tr.primary[key]
	if !ok {
		val, ok = tr.fallback[key]
	}
	tr.mu.RUnlock()
	if !ok {
		return key
	}
	if len(params) > 0 {
		for k, v := range params[0] {
			val = strings.ReplaceAll(val, "{{"+k+"}}", v)
		}
	}
	return val
}

func (tr *Translator) SetLocale(locale string) error {
	primary, err := loadLocale(locale)
	if err != nil {
		return err
	}
	tr.mu.Lock()
	tr.locale = locale
	tr.primary = primary
	tr.mu.Unlock()
	return nil
}

func (tr *Translator) Locale() string {
	tr.mu.RLock()
	defer tr.mu.RUnlock()
	return tr.locale
}
```

- [ ] **Step 5: Run tests, commit**

```bash
go test ./internal/i18n/... -v
git add internal/i18n/
git commit -m "feat(i18n): zh/en locales with fallback and interpolation"
```

---

### Task 16: Detail window (fyne)

**Files:**
- Create: `internal/ui/detail/window.go`

- [ ] **Step 1: Add fyne dependency**

```bash
go get fyne.io/fyne/v2
```

- [ ] **Step 2: Write window.go**

`internal/ui/detail/window.go`:
```go
package detail

import (
	"fmt"
	"net/url"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/<your-username>/speedforce/internal/core"
	"github.com/<your-username>/speedforce/internal/i18n"
)

type Window struct {
	app      fyne.App
	i18n     *i18n.Translator
	bus      *core.StateBus
	onSettings func()

	mu       sync.Mutex
	win      fyne.Window
	modeLbl  *widget.Label
	ipLbl    *widget.Label
	probesBox *fyne.Container
	spBox    *fyne.Container
}

func New(app fyne.App, i18n *i18n.Translator, bus *core.StateBus, onSettings func()) *Window {
	return &Window{app: app, i18n: i18n, bus: bus, onSettings: onSettings}
}

func (w *Window) Show() {
	w.mu.Lock()
	if w.win != nil {
		w.win.Show()
		w.win.RequestFocus()
		w.mu.Unlock()
		return
	}
	w.win = w.app.NewWindow(w.i18n.T("detail.title"))
	w.win.Resize(fyne.NewSize(420, 520))
	w.win.SetOnClosed(func() {
		w.mu.Lock()
		w.win = nil
		w.mu.Unlock()
	})

	w.modeLbl = widget.NewLabel(w.i18n.T("detail.mode.normal"))
	w.ipLbl = widget.NewLabel("...")
	w.probesBox = container.NewVBox()
	w.spBox = container.NewVBox()

	geminiBtn := widget.NewButton(w.i18n.T("detail.button.open_gemini_status"), func() {
		u, _ := url.Parse("https://aistudio.google.com/status")
		_ = w.app.OpenURL(u)
	})

	settingsBtn := widget.NewButton(w.i18n.T("detail.button.settings"), func() {
		if w.onSettings != nil {
			w.onSettings()
		}
	})

	content := container.NewVBox(
		container.NewHBox(widget.NewLabel(w.i18n.T("app_name")+" — "), w.modeLbl),
		widget.NewSeparator(),
		widget.NewLabel(w.i18n.T("detail.section.ip")),
		w.ipLbl,
		widget.NewSeparator(),
		widget.NewLabel(w.i18n.T("detail.section.probes")),
		w.probesBox,
		widget.NewSeparator(),
		widget.NewLabel(w.i18n.T("detail.section.statuspage")),
		w.spBox,
		widget.NewSeparator(),
		container.NewHBox(geminiBtn, settingsBtn),
	)
	w.win.SetContent(content)
	w.win.Show()
	w.mu.Unlock()

	go w.subscribe()
}

func (w *Window) subscribe() {
	sub := w.bus.Subscribe()
	w.update(w.bus.Snapshot())
	for s := range sub {
		w.update(s)
	}
}

func (w *Window) update(s core.State) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.win == nil {
		return
	}
	if s.Mode == core.TickFast {
		w.modeLbl.SetText(w.i18n.T("detail.mode.fast"))
	} else {
		w.modeLbl.SetText(w.i18n.T("detail.mode.normal"))
	}
	w.ipLbl.SetText(fmt.Sprintf("%s (%s / %s / %s)  LAN: %s",
		s.IP.PublicIP, s.IP.Country, s.IP.City, s.IP.ISP, s.IP.LANIP))

	w.probesBox.Objects = nil
	for _, p := range s.HTTPS {
		status := "🔴"
		if p.IsUp() {
			if p.LatencyMs > 1000 {
				status = "🟡"
			} else {
				status = "🔵"
			}
		}
		line := widget.NewLabel(fmt.Sprintf("%s  %s — %d ms (HTTP %d)", status, p.Name, p.LatencyMs, p.StatusCode))
		w.probesBox.Add(line)
	}

	w.spBox.Objects = nil
	for _, sp := range s.Statuspage {
		badge := "🔵"
		switch sp.Indicator {
		case core.StatuspageMinor:
			badge = "🟡"
		case core.StatuspageMajor, core.StatuspageCritical:
			badge = "🔴"
		case core.StatuspageMaintenance:
			badge = "🟡"
		}
		text := fmt.Sprintf("%s  %s — %s", badge, sp.Name, sp.Description)
		if sp.Err != nil {
			text = fmt.Sprintf("⚪  %s — unavailable", sp.Name)
		}
		w.spBox.Add(widget.NewLabel(text))
	}
}

func (w *Window) Close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.win != nil {
		w.win.Close()
		w.win = nil
	}
}
```

- [ ] **Step 3: Build**

```bash
go build ./internal/ui/detail/...
```
Expected: success

- [ ] **Step 4: Commit**

```bash
git add internal/ui/detail/ go.mod go.sum
git commit -m "feat(ui): detail window with live StateBus subscription"
```

---

### Task 17: Settings window (fyne)

**Files:**
- Create: `internal/ui/settings/window.go`

- [ ] **Step 1: Write settings/window.go**

`internal/ui/settings/window.go`:
```go
package settings

import (
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/<your-username>/speedforce/internal/config"
	"github.com/<your-username>/speedforce/internal/i18n"
)

type SaveFunc func(*config.Config) error
type ExportLogsFunc func() (path string, err error)

type Window struct {
	app        fyne.App
	i18n       *i18n.Translator
	cfg        *config.Config
	onSave     SaveFunc
	onExport   ExportLogsFunc

	mu  sync.Mutex
	win fyne.Window
}

func New(app fyne.App, tr *i18n.Translator, cfg *config.Config, save SaveFunc, export ExportLogsFunc) *Window {
	return &Window{app: app, i18n: tr, cfg: cfg, onSave: save, onExport: export}
}

func (w *Window) Show() {
	w.mu.Lock()
	if w.win != nil {
		w.win.Show()
		w.win.RequestFocus()
		w.mu.Unlock()
		return
	}
	w.win = w.app.NewWindow(w.i18n.T("settings.title"))
	w.win.Resize(fyne.NewSize(420, 480))

	langSel := widget.NewSelect([]string{"zh", "en"}, nil)
	langSel.Selected = w.cfg.Language

	proxyMode := widget.NewSelect([]string{"auto", "manual", "none"}, nil)
	proxyMode.Selected = w.cfg.Network.Proxy.Mode

	proxyURL := widget.NewEntry()
	proxyURL.SetText(w.cfg.Network.Proxy.ManualURL)

	tickEntry := widget.NewEntry()
	tickEntry.SetText(intToStr(w.cfg.Network.TickInterval))

	autostart := widget.NewCheck("", nil)
	autostart.Checked = w.cfg.UI.AutoStart

	notifCheckboxes := make([]*widget.Check, len(w.cfg.Probes.HTTPS))
	notifBox := container.NewVBox()
	for i, p := range w.cfg.Probes.HTTPS {
		c := widget.NewCheck(p.Name, nil)
		c.Checked = p.NotifyOnDown
		notifCheckboxes[i] = c
		notifBox.Add(c)
	}

	exportBtn := widget.NewButton(w.i18n.T("settings.export_logs"), func() {
		if w.onExport == nil {
			return
		}
		if path, err := w.onExport(); err == nil {
			dialogInfo(w.app, "Exported to "+path)
		} else {
			dialogInfo(w.app, "Export failed: "+err.Error())
		}
	})

	saveBtn := widget.NewButton(w.i18n.T("settings.save"), func() {
		w.cfg.Language = langSel.Selected
		w.cfg.Network.Proxy.Mode = proxyMode.Selected
		w.cfg.Network.Proxy.ManualURL = proxyURL.Text
		w.cfg.Network.TickInterval = strToInt(tickEntry.Text, w.cfg.Network.TickInterval)
		w.cfg.UI.AutoStart = autostart.Checked
		for i, c := range notifCheckboxes {
			w.cfg.Probes.HTTPS[i].NotifyOnDown = c.Checked
		}
		if err := w.onSave(w.cfg); err != nil {
			dialogInfo(w.app, "Save failed: "+err.Error())
			return
		}
		w.close()
	})

	cancelBtn := widget.NewButton(w.i18n.T("settings.cancel"), w.close)

	form := container.NewVBox(
		labelRow(w.i18n.T("settings.language"), langSel),
		labelRow(w.i18n.T("settings.proxy.mode"), proxyMode),
		labelRow(w.i18n.T("settings.proxy.url"), proxyURL),
		labelRow(w.i18n.T("settings.tick_interval"), tickEntry),
		labelRow(w.i18n.T("settings.autostart"), autostart),
		widget.NewSeparator(),
		widget.NewLabel(w.i18n.T("settings.notifications")),
		notifBox,
		widget.NewSeparator(),
		exportBtn,
		widget.NewSeparator(),
		container.NewHBox(saveBtn, cancelBtn),
	)
	w.win.SetContent(form)
	w.win.Show()
	w.mu.Unlock()
}

func (w *Window) close() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.win != nil {
		w.win.Close()
		w.win = nil
	}
}

func labelRow(label string, input fyne.CanvasObject) fyne.CanvasObject {
	return container.NewBorder(nil, nil, widget.NewLabel(label), nil, input)
}

func intToStr(i int) string {
	return fmtInt(i)
}

func strToInt(s string, fallback int) int {
	v, err := parseInt(s)
	if err != nil {
		return fallback
	}
	return v
}

func dialogInfo(app fyne.App, msg string) {
	win := app.NewWindow("Info")
	win.SetContent(widget.NewLabel(msg))
	win.Resize(fyne.NewSize(300, 100))
	win.Show()
}
```

Add small helper file `internal/ui/settings/numfmt.go`:
```go
package settings

import "strconv"

func fmtInt(i int) string          { return strconv.Itoa(i) }
func parseInt(s string) (int, error) { return strconv.Atoi(s) }
```

- [ ] **Step 2: Build**

```bash
go build ./internal/ui/settings/...
```
Expected: success

- [ ] **Step 3: Commit**

```bash
git add internal/ui/settings/
git commit -m "feat(ui): settings window with language/proxy/tick/notify controls"
```

---

### Task 18: Extend Tray with menu i18n and detail/settings callbacks

**Files:**
- Modify: `internal/ui/tray/tray.go`

- [ ] **Step 1: Replace tray.go**

`internal/ui/tray/tray.go`:
```go
package tray

import (
	"sync"

	"github.com/getlantern/systray"

	"github.com/<your-username>/speedforce/internal/core"
	"github.com/<your-username>/speedforce/internal/i18n"
)

type Callbacks struct {
	OnDetail   func()
	OnSettings func()
	OnQuit     func()
}

type Tray struct {
	i18n *i18n.Translator
	cb   Callbacks

	mu sync.Mutex
}

func New(tr *i18n.Translator, cb Callbacks) *Tray {
	return &Tray{i18n: tr, cb: cb}
}

func (t *Tray) Run() {
	systray.Run(t.onReady, t.onExit)
}

func (t *Tray) onReady() {
	systray.SetIcon(IconFor(core.StatusUnknown))
	systray.SetTooltip(t.i18n.T("tray.tooltip"))

	mDetail := systray.AddMenuItem(t.i18n.T("tray.menu.show_details"), "")
	mSettings := systray.AddMenuItem(t.i18n.T("tray.menu.settings"), "")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem(t.i18n.T("tray.menu.quit"), "")

	go func() {
		for {
			select {
			case <-mDetail.ClickedCh:
				if t.cb.OnDetail != nil {
					t.cb.OnDetail()
				}
			case <-mSettings.ClickedCh:
				if t.cb.OnSettings != nil {
					t.cb.OnSettings()
				}
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func (t *Tray) onExit() {
	if t.cb.OnQuit != nil {
		t.cb.OnQuit()
	}
}

func (t *Tray) SetStatus(status core.OverallStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()
	systray.SetIcon(IconFor(status))
}
```

- [ ] **Step 2: Build**

```bash
go build ./internal/ui/tray/...
```

- [ ] **Step 3: Commit**

```bash
git add internal/ui/tray/tray.go
git commit -m "feat(ui): tray uses i18n strings + settings callback"
```

---

### Task 19: Overall-status computation + wire everything in main.go

**Files:**
- Create: `internal/core/overall.go`
- Test: `internal/core/overall_test.go`
- Modify: `cmd/speedforce/main.go`

- [ ] **Step 1: Write failing test**

`internal/core/overall_test.go`:
```go
package core

import "testing"

func TestComputeOverall_AllUp(t *testing.T) {
	probes := []ProbeResult{
		{Name: "a", StatusCode: 200, LatencyMs: 100},
		{Name: "b", StatusCode: 200, LatencyMs: 200},
	}
	sp := []StatuspageResult{{Indicator: StatuspageOperational}}
	if got := ComputeOverall(probes, sp); got != StatusHealthy {
		t.Errorf("got %v, want healthy", got)
	}
}

func TestComputeOverall_SlowIsDegraded(t *testing.T) {
	probes := []ProbeResult{
		{Name: "a", StatusCode: 200, LatencyMs: 1500},
	}
	if got := ComputeOverall(probes, nil); got != StatusDegraded {
		t.Errorf("got %v, want degraded", got)
	}
}

func TestComputeOverall_ThreePlusDownIsRed(t *testing.T) {
	probes := []ProbeResult{
		{Name: "a", Err: &testErr{"x"}},
		{Name: "b", Err: &testErr{"x"}},
		{Name: "c", Err: &testErr{"x"}},
		{Name: "d", StatusCode: 200, LatencyMs: 100},
	}
	if got := ComputeOverall(probes, nil); got != StatusDown {
		t.Errorf("got %v, want down", got)
	}
}

func TestComputeOverall_MajorOutageIsRed(t *testing.T) {
	probes := []ProbeResult{{StatusCode: 200, LatencyMs: 100}}
	sp := []StatuspageResult{{Indicator: StatuspageMajor}}
	if got := ComputeOverall(probes, sp); got != StatusDown {
		t.Errorf("got %v, want down on major outage", got)
	}
}

func TestComputeOverall_OneOrTwoDownIsYellow(t *testing.T) {
	probes := []ProbeResult{
		{Name: "a", Err: &testErr{"x"}},
		{Name: "b", StatusCode: 200, LatencyMs: 100},
		{Name: "c", StatusCode: 200, LatencyMs: 100},
	}
	if got := ComputeOverall(probes, nil); got != StatusDegraded {
		t.Errorf("got %v, want degraded", got)
	}
}
```

- [ ] **Step 2: Run, expect fail**

```bash
go test ./internal/core/... -run TestComputeOverall
```
Expected: FAIL

- [ ] **Step 3: Implement**

`internal/core/overall.go`:
```go
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
```

- [ ] **Step 4: Run tests, expect pass**

```bash
go test ./internal/core/... -v -run TestComputeOverall
```

- [ ] **Step 5: Rewrite main.go to wire everything**

Replace `cmd/speedforce/main.go`:
```go
package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"fyne.io/fyne/v2/app"

	"github.com/<your-username>/speedforce/internal/config"
	"github.com/<your-username>/speedforce/internal/core"
	"github.com/<your-username>/speedforce/internal/i18n"
	"github.com/<your-username>/speedforce/internal/platform"
	"github.com/<your-username>/speedforce/internal/probe"
	"github.com/<your-username>/speedforce/internal/ui/detail"
	"github.com/<your-username>/speedforce/internal/ui/settings"
	"github.com/<your-username>/speedforce/internal/ui/tray"
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
				func(newCfg *config.Config) error { return nil }, // hot-reload hooked in M4
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

	// Background loop: map StateBus → tray icon color
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
```

- [ ] **Step 6: Build and smoke-run**

```bash
go build -o speedforce.exe ./cmd/speedforce
./speedforce.exe
```

Expected: tray icon appears, cycles through colors based on state; right-click shows i18n'd menu; clicking "Show Details" opens the fyne window.

- [ ] **Step 7: Commit**

```bash
git add internal/core/overall.go internal/core/overall_test.go cmd/speedforce/main.go go.mod go.sum
git commit -m "feat: wire config + i18n + scheduler + tray + detail (M3 complete)"
```

---

## M4 — Feature-Complete

### Task 20: Config hot-reload with fsnotify

**Files:**
- Modify: `internal/config/config.go` (add `Watch`)
- Test: `internal/config/watch_test.go`

- [ ] **Step 1: Add fsnotify**

```bash
go get github.com/fsnotify/fsnotify
```

- [ ] **Step 2: Write failing test**

`internal/config/watch_test.go`:
```go
package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatch_FiresOnChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.yaml")
	if err := os.WriteFile(path, []byte(DefaultYAML), 0600); err != nil {
		t.Fatal(err)
	}

	w, err := Watch(path)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	// modify file
	go func() {
		time.Sleep(50 * time.Millisecond)
		modified := DefaultYAML + "\n# extra comment\n"
		_ = os.WriteFile(path, []byte(modified), 0600)
	}()

	select {
	case cfg := <-w.Changes():
		if cfg == nil {
			t.Fatal("got nil config")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no change event received")
	}
}

func TestWatch_RollbackOnBadYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "c.yaml")
	os.WriteFile(path, []byte(DefaultYAML), 0600)

	w, err := Watch(path)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()

	go func() {
		time.Sleep(50 * time.Millisecond)
		os.WriteFile(path, []byte(":: not yaml ::"), 0600)
	}()

	// We expect no valid change pushed — instead an error on Errors()
	select {
	case <-w.Changes():
		t.Fatal("bad YAML should not push a config")
	case err := <-w.Errors():
		if err == nil {
			t.Fatal("expected error")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for error")
	}
}
```

- [ ] **Step 3: Run, expect fail**

```bash
go test ./internal/config/... -run TestWatch
```

- [ ] **Step 4: Implement Watcher**

Append to `internal/config/config.go` (or create `watch.go`):
```go
package config

import (
	"github.com/fsnotify/fsnotify"
)

type Watcher struct {
	fs      *fsnotify.Watcher
	path    string
	changes chan *Config
	errors  chan error
	done    chan struct{}
}

func Watch(path string) (*Watcher, error) {
	fs, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := fs.Add(path); err != nil {
		fs.Close()
		return nil, err
	}
	w := &Watcher{
		fs:      fs,
		path:    path,
		changes: make(chan *Config, 1),
		errors:  make(chan error, 1),
		done:    make(chan struct{}),
	}
	go w.loop()
	return w, nil
}

func (w *Watcher) Changes() <-chan *Config { return w.changes }
func (w *Watcher) Errors() <-chan error    { return w.errors }

func (w *Watcher) Close() {
	close(w.done)
	w.fs.Close()
}

func (w *Watcher) loop() {
	for {
		select {
		case <-w.done:
			return
		case ev, ok := <-w.fs.Events:
			if !ok {
				return
			}
			if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0 {
				cfg, err := Load(w.path)
				if err != nil {
					select {
					case w.errors <- err:
					default:
					}
					continue
				}
				select {
				case w.changes <- cfg:
				default:
				}
			}
		case err, ok := <-w.fs.Errors:
			if !ok {
				return
			}
			select {
			case w.errors <- err:
			default:
			}
		}
	}
}
```

- [ ] **Step 5: Save helper to write config back with comments preserved**

Append to `internal/config/config.go`:
```go
func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
```

Note: `goccy/go-yaml` preserves map ordering but not original comments. For v0.1 we accept that the settings UI's "save" will rewrite the file without user-added comments. Document this in README.

- [ ] **Step 6: Run tests, commit**

```bash
go test ./internal/config/... -v
git add internal/config/ go.mod go.sum
git commit -m "feat(config): hot-reload via fsnotify"
```

---

### Task 21: Logger with daily rotation

**Files:**
- Create: `internal/logger/logger.go`
- Test: `internal/logger/logger_test.go`

- [ ] **Step 1: Add zerolog**

```bash
go get github.com/rs/zerolog
```

- [ ] **Step 2: Write failing test**

`internal/logger/logger_test.go`:
```go
package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNew_WritesToFile(t *testing.T) {
	dir := t.TempDir()
	lg, err := New(Options{Dir: dir, Level: "info", RetentionDays: 30})
	if err != nil {
		t.Fatal(err)
	}
	defer lg.Close()

	lg.Info().Str("key", "value").Msg("hello")
	lg.Sync()

	entries, _ := os.ReadDir(dir)
	if len(entries) == 0 {
		t.Fatal("no log file written")
	}
	data, _ := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if !strings.Contains(string(data), "hello") || !strings.Contains(string(data), "value") {
		t.Errorf("log content unexpected: %s", data)
	}
}

func TestPrune_RemovesOldFiles(t *testing.T) {
	dir := t.TempDir()
	old := filepath.Join(dir, "probe-2000-01-01.log")
	os.WriteFile(old, []byte("x"), 0600)
	oldTime := time.Now().AddDate(0, 0, -60)
	os.Chtimes(old, oldTime, oldTime)

	lg, err := New(Options{Dir: dir, Level: "info", RetentionDays: 30})
	if err != nil {
		t.Fatal(err)
	}
	defer lg.Close()

	lg.Prune()

	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Error("old file should be pruned")
	}
}
```

- [ ] **Step 3: Implement**

`internal/logger/logger.go`:
```go
package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

type Options struct {
	Dir           string
	Level         string
	RetentionDays int
}

type Logger struct {
	zerolog.Logger
	opts Options

	mu      sync.Mutex
	current *os.File
	curDate string
}

func New(opts Options) (*Logger, error) {
	if err := os.MkdirAll(opts.Dir, 0755); err != nil {
		return nil, err
	}
	lg := &Logger{opts: opts}
	if err := lg.rotate(); err != nil {
		return nil, err
	}
	return lg, nil
}

func parseLevel(s string) zerolog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return zerolog.DebugLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	default:
		return zerolog.InfoLevel
	}
}

func (l *Logger) rotate() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	date := time.Now().Format("2006-01-02")
	if date == l.curDate && l.current != nil {
		return nil
	}
	if l.current != nil {
		l.current.Close()
	}
	path := filepath.Join(l.opts.Dir, fmt.Sprintf("probe-%s.log", date))
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	l.current = f
	l.curDate = date
	l.Logger = zerolog.New(io.MultiWriter(f)).Level(parseLevel(l.opts.Level)).With().Timestamp().Logger()
	return nil
}

func (l *Logger) Sync() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.current != nil {
		l.current.Sync()
	}
}

func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.current != nil {
		l.current.Close()
		l.current = nil
	}
}

func (l *Logger) Prune() {
	entries, err := os.ReadDir(l.opts.Dir)
	if err != nil {
		return
	}
	cutoff := time.Now().AddDate(0, 0, -l.opts.RetentionDays)
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "probe-") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(l.opts.Dir, e.Name()))
		}
	}
}
```

- [ ] **Step 4: Run tests, commit**

```bash
go test ./internal/logger/... -v
git add internal/logger/ go.mod go.sum
git commit -m "feat(logger): daily-rotated structured logs with retention prune"
```

---

### Task 22: Notifier

**Files:**
- Create: `internal/core/notifier.go`
- Test: `internal/core/notifier_test.go`

- [ ] **Step 1: Add beeep**

```bash
go get github.com/gen2brain/beeep
```

- [ ] **Step 2: Write failing test**

`internal/core/notifier_test.go`:
```go
package core

import (
	"sync"
	"testing"
	"time"
)

type fakeToast struct {
	mu    sync.Mutex
	calls []string
}

func (f *fakeToast) Notify(title, body string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, title+"|"+body)
	return nil
}

func (f *fakeToast) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func TestNotifier_FiresOnUpToDown(t *testing.T) {
	ft := &fakeToast{}
	n := NewNotifier(NotifierConfig{
		Toast:       ft,
		NotifyHTTPS: map[string]bool{"Claude": true},
	})

	prev := State{HTTPS: []ProbeResult{{Name: "Claude", StatusCode: 200, LatencyMs: 100}}}
	curr := State{HTTPS: []ProbeResult{{Name: "Claude", Err: &testErr{"timeout"}}}}
	n.Observe(prev)
	n.Observe(curr)

	if ft.count() != 1 {
		t.Errorf("expected 1 toast, got %d", ft.count())
	}
}

func TestNotifier_SkipsServiceWithNotifyFalse(t *testing.T) {
	ft := &fakeToast{}
	n := NewNotifier(NotifierConfig{
		Toast:       ft,
		NotifyHTTPS: map[string]bool{"Claude": false},
	})

	prev := State{HTTPS: []ProbeResult{{Name: "Claude", StatusCode: 200, LatencyMs: 100}}}
	curr := State{HTTPS: []ProbeResult{{Name: "Claude", Err: &testErr{"timeout"}}}}
	n.Observe(prev)
	n.Observe(curr)

	if ft.count() != 0 {
		t.Errorf("expected no toast, got %d", ft.count())
	}
}

func TestNotifier_NoToastOnRecovery(t *testing.T) {
	ft := &fakeToast{}
	n := NewNotifier(NotifierConfig{
		Toast:       ft,
		NotifyHTTPS: map[string]bool{"Claude": true},
	})

	prev := State{HTTPS: []ProbeResult{{Name: "Claude", Err: &testErr{"t"}}}}
	curr := State{HTTPS: []ProbeResult{{Name: "Claude", StatusCode: 200, LatencyMs: 100}}}
	n.Observe(prev)
	n.Observe(curr)

	if ft.count() != 0 {
		t.Errorf("recovery should not fire toast, got %d", ft.count())
	}
}

func TestNotifier_FlapCooldown(t *testing.T) {
	ft := &fakeToast{}
	n := NewNotifier(NotifierConfig{
		Toast:         ft,
		NotifyHTTPS:   map[string]bool{"Claude": true},
		CooldownDur:   5 * time.Minute,
	})

	upState := State{HTTPS: []ProbeResult{{Name: "Claude", StatusCode: 200, LatencyMs: 100}}}
	downState := State{HTTPS: []ProbeResult{{Name: "Claude", Err: &testErr{"t"}}}}

	n.Observe(upState)
	n.Observe(downState)    // fires
	n.Observe(upState)      // recovery, no fire
	n.Observe(downState)    // would fire but within cooldown

	if ft.count() != 1 {
		t.Errorf("expected 1 toast (cooldown suppressed second), got %d", ft.count())
	}
}

func TestNotifier_StatuspageMajorFires(t *testing.T) {
	ft := &fakeToast{}
	n := NewNotifier(NotifierConfig{
		Toast:                    ft,
		NotifyStatuspageMajor:    true,
	})
	prev := State{Statuspage: []StatuspageResult{{Name: "Anthropic", Indicator: StatuspageOperational}}}
	curr := State{Statuspage: []StatuspageResult{{Name: "Anthropic", Indicator: StatuspageMajor}}}
	n.Observe(prev)
	n.Observe(curr)
	if ft.count() != 1 {
		t.Errorf("expected major-outage toast, got %d", ft.count())
	}
}
```

- [ ] **Step 3: Implement**

`internal/core/notifier.go`:
```go
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
	NotifyHTTPS           map[string]bool // name -> notify?
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
```

- [ ] **Step 4: Create beeep adapter**

`internal/core/beeep_adapter.go`:
```go
package core

import "github.com/gen2brain/beeep"

type BeeepSender struct{}

func (BeeepSender) Notify(title, body string) error {
	return beeep.Notify(title, body, "")
}
```

- [ ] **Step 5: Run tests, commit**

```bash
go test ./internal/core/... -v -run TestNotifier
git add internal/core/notifier.go internal/core/notifier_test.go internal/core/beeep_adapter.go go.mod go.sum
git commit -m "feat(core): notifier with per-service toggles and flap cooldown"
```

---

### Task 23: Windows autostart + singleton guard

**Files:**
- Create: `internal/platform/autostart_windows.go`
- Create: `internal/platform/autostart_darwin.go`
- Create: `internal/platform/autostart_linux.go`
- Create: `internal/platform/autostart.go` (interface)
- Create: `internal/platform/singleton_windows.go`
- Create: `internal/platform/singleton_other.go`

- [ ] **Step 1: Interface**

`internal/platform/autostart.go`:
```go
package platform

func SetAutoStart(enabled bool, exePath string) error {
	return setAutoStartImpl(enabled, exePath)
}

func IsAutoStart() (bool, error) {
	return isAutoStartImpl()
}
```

- [ ] **Step 2: Windows impl**

`internal/platform/autostart_windows.go`:
```go
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
```

- [ ] **Step 3: Darwin/Linux stubs**

`internal/platform/autostart_darwin.go`:
```go
//go:build darwin

package platform

func setAutoStartImpl(enabled bool, exePath string) error { return nil }
func isAutoStartImpl() (bool, error)                       { return false, nil }
```

`internal/platform/autostart_linux.go`:
```go
//go:build linux

package platform

func setAutoStartImpl(enabled bool, exePath string) error { return nil }
func isAutoStartImpl() (bool, error)                       { return false, nil }
```

- [ ] **Step 4: Singleton (Windows named mutex)**

`internal/platform/singleton_windows.go`:
```go
//go:build windows

package platform

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutex = kernel32.NewProc("CreateMutexW")
)

func AcquireSingleton(name string) (release func(), err error) {
	ptr, _ := syscall.UTF16PtrFromString(name)
	r1, _, callErr := procCreateMutex.Call(0, 0, uintptr(unsafe.Pointer(ptr)))
	if r1 == 0 {
		return nil, callErr
	}
	if callErr == syscall.Errno(183) { // ERROR_ALREADY_EXISTS
		syscall.CloseHandle(syscall.Handle(r1))
		return nil, fmt.Errorf("already running")
	}
	handle := syscall.Handle(r1)
	return func() { syscall.CloseHandle(handle) }, nil
}
```

`internal/platform/singleton_other.go`:
```go
//go:build !windows

package platform

func AcquireSingleton(name string) (release func(), err error) {
	return func() {}, nil
}
```

- [ ] **Step 5: Wire autostart + singleton in main.go**

At top of `main()` in `cmd/speedforce/main.go`, before config load:
```go
	release, err := platform.AcquireSingleton("Global\\SpeedForce.Singleton")
	if err != nil {
		log.Fatalf("another instance already running: %v", err)
	}
	defer release()
```

After `cfg, err := config.Load(cfgPath)`:
```go
	exePath, _ := os.Executable()
	if err := platform.SetAutoStart(cfg.UI.AutoStart, exePath); err != nil {
		log.Printf("autostart: %v", err)
	}
```

- [ ] **Step 6: Build all platforms and commit**

```bash
go build ./cmd/speedforce
GOOS=darwin go build ./internal/platform/...
GOOS=linux go build ./internal/platform/...
git add internal/platform/ cmd/speedforce/main.go
git commit -m "feat(platform): windows autostart and singleton-instance guard"
```

---

### Task 24: Log export (zip last 7 days)

**Files:**
- Modify: `internal/logger/logger.go` (add `ExportZip`)
- Test: `internal/logger/export_test.go`

- [ ] **Step 1: Write failing test**

`internal/logger/export_test.go`:
```go
package logger

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExportZip_IncludesRecentLogs(t *testing.T) {
	dir := t.TempDir()
	recent := filepath.Join(dir, "probe-"+time.Now().Format("2006-01-02")+".log")
	old := filepath.Join(dir, "probe-2000-01-01.log")
	os.WriteFile(recent, []byte("recent log"), 0600)
	os.WriteFile(old, []byte("old log"), 0600)
	oldTime := time.Now().AddDate(0, 0, -60)
	os.Chtimes(old, oldTime, oldTime)

	lg, err := New(Options{Dir: dir, Level: "info", RetentionDays: 30})
	if err != nil {
		t.Fatal(err)
	}
	defer lg.Close()

	outDir := t.TempDir()
	outPath, err := lg.ExportZip(outDir, 7)
	if err != nil {
		t.Fatal(err)
	}

	zr, err := zip.OpenReader(outPath)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()

	var recentFound, oldFound bool
	for _, f := range zr.File {
		if filepath.Base(f.Name) == filepath.Base(recent) {
			recentFound = true
		}
		if filepath.Base(f.Name) == filepath.Base(old) {
			oldFound = true
		}
	}
	if !recentFound {
		t.Error("recent log not in zip")
	}
	if oldFound {
		t.Error("old log should not be in 7-day export")
	}
}
```

- [ ] **Step 2: Implement ExportZip**

Append to `internal/logger/logger.go`:
```go
import (
	"archive/zip"
	"io"
)

func (l *Logger) ExportZip(outDir string, days int) (string, error) {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", err
	}
	outPath := filepath.Join(outDir, fmt.Sprintf("speedforce-logs-%s.zip", time.Now().Format("20060102-150405")))
	f, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	entries, err := os.ReadDir(l.opts.Dir)
	if err != nil {
		return "", err
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), "probe-") {
			continue
		}
		info, err := e.Info()
		if err != nil || info.ModTime().Before(cutoff) {
			continue
		}
		src, err := os.Open(filepath.Join(l.opts.Dir, e.Name()))
		if err != nil {
			continue
		}
		w, err := zw.Create(e.Name())
		if err != nil {
			src.Close()
			return "", err
		}
		_, _ = io.Copy(w, src)
		src.Close()
	}
	return outPath, nil
}
```

Merge the new `import` additions into the existing imports block at the top of `logger.go` (don't create two `import` blocks).

- [ ] **Step 3: Wire into main.go / settings callback**

Replace the stub export callback in `cmd/speedforce/main.go`:
```go
	// replace the existing ExportLogsFunc stub
	exportFn := func() (string, error) {
		return lg.ExportZip(filepath.Join(appDataDir, "exports"), 7)
	}
```

(Assumes you capture `lg *logger.Logger` and `appDataDir` earlier — add those.)

- [ ] **Step 4: Test and commit**

```bash
go test ./internal/logger/... -v -run TestExportZip
git add internal/logger/logger.go internal/logger/export_test.go cmd/speedforce/main.go
git commit -m "feat(logger): export recent logs as zip (M4 complete)"
```

---

## M5 — Polish + Release

### Task 25: Integration test (end-to-end mock pipeline)

**Files:**
- Create: `internal/core/integration_test.go`

- [ ] **Step 1: Write integration test**

`internal/core/integration_test.go`:
```go
package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

type httpsProberImpl struct {
	client *http.Client
}

func (p *httpsProberImpl) Probe(ctx context.Context, name, url string) ProbeResult {
	req, _ := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	start := time.Now()
	resp, err := p.client.Do(req)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		return ProbeResult{Name: name, URL: url, Err: err, LatencyMs: latency, Timestamp: time.Now()}
	}
	defer resp.Body.Close()
	return ProbeResult{Name: name, URL: url, StatusCode: resp.StatusCode, LatencyMs: latency, Timestamp: time.Now()}
}

func TestIntegration_SchedulerToBusToNotifier(t *testing.T) {
	var upSwitch atomic.Bool
	upSwitch.Store(true)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if upSwitch.Load() {
			w.WriteHeader(200)
			return
		}
		// simulate down: sleep past timeout
		time.Sleep(300 * time.Millisecond)
	}))
	defer srv.Close()

	bus := NewStateBus()
	defer bus.Close()

	client := &http.Client{Timeout: 100 * time.Millisecond}
	sch := NewScheduler(SchedulerConfig{
		Bus:             bus,
		HTTPS:           &httpsProberImpl{client: client},
		HTTPSTargets:    []Target{{Name: "Service", URL: srv.URL}},
		TickInterval:    30 * time.Millisecond,
		AdaptiveEnabled: false,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go sch.Run(ctx)

	fakeToast := &fakeToast{}
	notifier := NewNotifier(NotifierConfig{
		Toast:       fakeToast,
		NotifyHTTPS: map[string]bool{"Service": true},
	})

	sub := bus.Subscribe()

	// observe for a bit while up
	done := make(chan struct{})
	go func() {
		timeout := time.After(3 * time.Second)
		for {
			select {
			case s, ok := <-sub:
				if !ok {
					close(done)
					return
				}
				notifier.Observe(s)
			case <-timeout:
				close(done)
				return
			}
		}
	}()

	time.Sleep(150 * time.Millisecond)
	upSwitch.Store(false) // flip to down
	time.Sleep(300 * time.Millisecond)

	<-done

	if fakeToast.count() == 0 {
		t.Error("expected at least one toast after up→down flip")
	}
}
```

(`fakeToast` and `testErr` live in the test files from earlier tasks — this test shares the package.)

- [ ] **Step 2: Run with race**

```bash
go test -race ./internal/core/... -run TestIntegration -v
```
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/core/integration_test.go
git commit -m "test(core): integration test wiring scheduler → bus → notifier"
```

---

### Task 26: Debug flags (--fake-down, --tick)

**Files:**
- Modify: `cmd/speedforce/main.go`

- [ ] **Step 1: Add flag parsing at top of main()**

Edit `cmd/speedforce/main.go` — add after the existing variable declarations at the top of `main()`:
```go
import "flag"

// ...

	fakeDown := flag.String("fake-down", "", "comma-separated service names to simulate as down")
	tickOverride := flag.Int("tick", 0, "override tick interval in seconds (debug)")
	flag.Parse()
```

- [ ] **Step 2: Apply overrides**

After loading config, apply overrides:
```go
	if *tickOverride > 0 {
		cfg.Network.TickInterval = *tickOverride
	}
```

For `--fake-down`, add a wrapper prober:
```go
type fakeDownProber struct {
	inner probe.HTTPSProber
	downSet map[string]bool
}

func (f *fakeDownProber) Probe(ctx context.Context, name, url string) core.ProbeResult {
	if f.downSet[name] {
		return core.ProbeResult{Name: name, URL: url, Err: errFakeDown, Timestamp: time.Now()}
	}
	return f.inner.Probe(ctx, name, url)
}

var errFakeDown = errors.New("fake-down")
```

Then when constructing the prober in `main()`:
```go
	var httpsProber core.HTTPSProbeFunc = probe.NewHTTPSProber(client)
	if *fakeDown != "" {
		set := make(map[string]bool)
		for _, n := range strings.Split(*fakeDown, ",") {
			set[strings.TrimSpace(n)] = true
		}
		httpsProber = &fakeDownProber{inner: *probe.NewHTTPSProber(client), downSet: set}
	}
```

- [ ] **Step 3: Smoke test**

```bash
go build -o speedforce.exe ./cmd/speedforce
./speedforce.exe --fake-down="Claude API" --tick=5
```
Expected: runs with Claude API always showing as down; tick every 5s.

- [ ] **Step 4: Commit**

```bash
git add cmd/speedforce/main.go
git commit -m "feat(debug): --fake-down and --tick CLI flags for development"
```

---

### Task 27: Prune dev log files + validate coverage

- [ ] **Step 1: Run coverage and inspect**

```bash
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -30
```

- [ ] **Step 2: Add tests for any Core module below 80% coverage**

If `internal/core/overall.go` or `internal/core/notifier.go` shows below 80%, add specific case tests targeting uncovered branches.

- [ ] **Step 3: Commit any new tests**

```bash
git add internal/core/
git commit -m "test: raise core layer coverage to ≥80%"
```

---

### Task 28: GitHub Actions CI

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Write CI workflow**

`.github/workflows/ci.yml`:
```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true

      - name: Install golangci-lint
        run: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

      - name: Lint
        run: golangci-lint run ./...

      - name: Test
        run: go test -race -coverprofile=coverage.out ./...

      - name: Coverage summary
        run: go tool cover -func=coverage.out

      - name: Build
        run: go build -o speedforce.exe ./cmd/speedforce

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: speedforce-windows
          path: speedforce.exe
```

- [ ] **Step 2: Add golangci-lint config**

`.golangci.yml`:
```yaml
linters:
  enable:
    - gofmt
    - govet
    - staticcheck
    - ineffassign
    - errcheck
run:
  timeout: 5m
```

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/ci.yml .golangci.yml
git commit -m "ci: github actions for test/lint/build on windows-latest"
```

---

### Task 29: GitHub Actions release workflow

**Files:**
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: Write release workflow**

`.github/workflows/release.yml`:
```yaml
name: Release

on:
  push:
    tags: ['v*']

jobs:
  release:
    runs-on: windows-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
          cache: true

      - name: Build
        run: |
          go build -ldflags="-H windowsgui -s -w" -o speedforce.exe ./cmd/speedforce

      - name: Zip
        run: Compress-Archive -Path speedforce.exe,README.md,LICENSE -DestinationPath speedforce-${{ github.ref_name }}-windows-amd64.zip
        shell: pwsh

      - name: Release
        uses: softprops/action-gh-release@v2
        with:
          files: speedforce-*.zip
          draft: false
          prerelease: false
          generate_release_notes: true
```

The `-H windowsgui` flag hides the console window when the exe runs (important for a tray app). `-s -w` strips debug info to shrink the binary.

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci: release workflow builds windows binary on tag push"
```

---

### Task 30: README, LICENSE, and v0.1 release

**Files:**
- Create: `LICENSE` (MIT)
- Overwrite: `README.md`

- [ ] **Step 1: Add MIT license**

`LICENSE`:
```
MIT License

Copyright (c) 2026 <your-name>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

- [ ] **Step 2: Write README.md**

`README.md`:
```markdown
# SpeedForce ⚡

A lightweight Windows tray app that watches your ability to reach Claude, OpenAI, and Gemini — built for people who access them through a proxy.

![lightning blue](assets/icons/lightning-blue.ico) all good &nbsp;
![lightning yellow](assets/icons/lightning-yellow.ico) degraded &nbsp;
![lightning red](assets/icons/lightning-red.ico) down

## Features

- 6 HTTPS reachability probes (3 API endpoints + 3 consumer web endpoints)
- Public IP + LAN IP + geolocation + ISP
- Anthropic and OpenAI Statuspage integration; direct link to Gemini / AI Studio status
- Per-service opt-in notifications (OS toasts)
- Adaptive polling: 60s by default, 10s on instability, 60s once stable for 3 ticks
- Auto-detect system proxy; manual proxy override
- Bilingual (zh / en)
- Log export (last 7 days zipped)
- Single-instance; starts with Windows (optional)
- ~10 MB idle memory, ~30 MB with detail window open

## Install

Download the latest `speedforce-vX.Y.Z-windows-amd64.zip` from the [releases page](https://github.com/<your-username>/speedforce/releases) and run `speedforce.exe`.

## Configuration

Config lives at `%APPDATA%\SpeedForce\config.yaml`. It is created with defaults on first run and hot-reloaded on change. See the [design doc](docs/superpowers/specs/2026-04-15-speedforce-design.md) for the full schema.

## Building from Source

```bash
git clone https://github.com/<your-username>/speedforce.git
cd speedforce
go build -ldflags="-H windowsgui -s -w" -o speedforce.exe ./cmd/speedforce
```

Requires Go 1.22+.

## Development

```bash
go test -race ./...
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

Debug flags:
- `--tick=5` — override tick interval (seconds)
- `--fake-down="Claude API,OpenAI API"` — simulate services being down

## Roadmap

See [design doc §12](docs/superpowers/specs/2026-04-15-speedforce-design.md#12-future-work-post-v01) for planned work: macOS + Linux support, persistent history, sparklines, custom endpoints.

## License

MIT. See [LICENSE](LICENSE).
```

- [ ] **Step 3: Final test sweep**

```bash
go test -race ./...
go vet ./...
go build -o speedforce.exe ./cmd/speedforce
```
All should pass cleanly.

- [ ] **Step 4: Tag and push**

```bash
git add LICENSE README.md
git commit -m "docs: add README and MIT license"
git tag v0.1.0
git push origin main
git push origin v0.1.0
```

The release workflow (Task 29) will build and attach `speedforce-v0.1.0-windows-amd64.zip` to the GitHub Release automatically.

- [ ] **Step 5: Manual smoke from §9.3 of spec**

Run through the seven-step smoke checklist from the spec:
1. Double-click `.exe` → blue lightning appears
2. Disconnect network → icon red within 10-20s + toast
3. Reconnect → blue (no toast)
4. Toggle proxy on/off → icon updates
5. Open DetailWindow → close → memory released
6. Edit `config.yaml` → change applies without restart
7. Switch language zh↔en → UI updates

Fix anything that fails before announcing v0.1.

---

**🎉 M5 complete — SpeedForce v0.1 ready for release.**





