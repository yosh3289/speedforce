---
name: SpeedForce Design
description: Lightweight Windows tray app that monitors connectivity to Claude, OpenAI, and Gemini. Planned for cross-platform extension.
type: design
date: 2026-04-15
status: approved
---

# SpeedForce — Design Document

## 1. Overview

**SpeedForce** is a lightweight system tray application that continuously monitors a user's ability to reach three major AI service providers (Anthropic/Claude, OpenAI/ChatGPT, Google/Gemini) from their current network. It is primarily targeted at users who rely on HTTP/SOCKS proxies to access these services and need fast signal when connectivity degrades.

The app runs quietly in the Windows tray as a colored lightning bolt icon (blue/yellow/red — a Flash/Savitar color theme) and surfaces detailed status on demand through a hybrid menu + detail-window UI.

**Name origin:** "Speed Force" is the energy source powering every speedster in *The Flash*. The lightning bolt icon and color scheme reference the show's speedsters.

## 2. Goals & Non-Goals

### Goals

- Continuously verify that the user's network (including any active proxy) can reach Anthropic, OpenAI, and Google AI endpoints
- Show current public IP, geolocation, and ISP (reflects proxy exit node when a proxy is in use)
- Surface official platform status from Anthropic and OpenAI Statuspage
- Alert the user via OS notifications when services they care about go down
- Stay lightweight: ~10-12 MB resident memory when idle, ~30-40 MB when the detail window is open
- Runnable as a single `.exe` with zero install dependencies
- Bilingual UI (Chinese / English)
- Open-source with a clear path to cross-platform support (macOS, Linux)

### Non-Goals (v1)

- macOS or Linux support (code is structured to allow this later, but not shipped in v1)
- Measuring throughput, packet loss, or running speedtest-style benchmarks
- Authenticated API health probes (we do not require valid API keys; reachability is what matters)
- Historical charts or long-term trend analysis (in-memory 1-hour cache only; no persistent history)
- Replacing official status pages for Gemini (no reliable public status API exists; we link to `aistudio.google.com/status` instead)

## 3. Architecture

```
┌──────────────────────────────────────────────────┐
│                   Tray App (Go)                  │
│                                                  │
│  ┌──────────┐  ┌──────────┐  ┌────────────────┐  │
│  │  Tray    │←→│  Core    │←→│  Detail Window │  │
│  │ (systray)│  │ (engine) │  │    (fyne)      │  │
│  └──────────┘  └──────────┘  └────────────────┘  │
│                     ↓                            │
│       ┌─────────────┼─────────────┐              │
│       ↓             ↓             ↓              │
│  ┌────────┐   ┌──────────┐  ┌─────────┐          │
│  │Probers │   │ IP Probe │  │ Status  │          │
│  │(HTTPS) │   │  (IPv4)  │  │ Fetcher │          │
│  └────────┘   └──────────┘  └─────────┘          │
│       ↓             ↓             ↓              │
│  ┌─────────────────────────────────────┐         │
│  │      Proxy-Aware HTTP Client        │         │
│  └─────────────────────────────────────┘         │
│                     ↓                            │
└─────────────────────┼────────────────────────────┘
                      ↓
           Internet (through proxy if configured)
```

**Three layers:**

- **UI Layer** — `Tray` and `DetailWindow` are independent UI components subscribing to state changes from Core
- **Core Layer** — Scheduler, StateBus (state distribution), ConfigManager, I18n, Logger, Notifier
- **Probe Layer** — Three independent prober types (HTTPS, IP, Statuspage) sharing a proxy-aware HTTP client

**Key design principles:**

- UI decoupled from Core — swapping the UI library (fyne → something else) does not affect business logic
- Core decoupled from probes via interfaces — probes can be mocked for unit tests
- Single `StateBus` is the canonical source of truth; UI components subscribe instead of polling
- All platform-specific code isolated in `internal/platform/` behind Go build tags

## 4. Core Components

### 4.1 Core Layer

| Component | Responsibility | Key Data |
|---|---|---|
| `Scheduler` | Triggers probe tasks on configured intervals; implements adaptive tick (see §6) | current tick interval, mode (normal/fast) |
| `StateBus` | In-memory state store; all probe results land here; UI subscribes to change events | rolling 1-hour result cache |
| `ConfigManager` | Reads, validates, and hot-reloads `config.yaml` | file path, last-known-good config |
| `Notifier` | Compares successive states; filters by `notify_on_down` per service; triggers OS toasts | notification cooldown map |
| `Logger` | Structured logging, rotating daily files, 30-day retention | log file handle |
| `I18n` | Loads language packs; exposes translation lookups; supports runtime switching | current locale, fallback chain |

### 4.2 Probe Layer

| Prober | How It Works | Output |
|---|---|---|
| `HTTPSProber` | Sends HEAD requests to 6 endpoints. Treats 2xx/3xx/4xx as "reachable" (indicates network path works, even if unauthenticated). Only timeout / TLS failure / 5xx count as "down". | latency (ms) + status code + error |
| `IPProber` | Queries `api.ipify.org` for public IP, then `ipapi.co/{ip}/json/` for geo/ISP. Runs every 5 ticks (not every tick). | IP, country, city, ISP, LAN IP |
| `StatuspageProber` | Fetches Statuspage JSON API for Anthropic and OpenAI. Runs every 5 minutes (fixed, independent of adaptive tick). | per-service status (operational / degraded / partial_outage / major_outage) |

**Probed endpoints:**

- **API:** `api.anthropic.com`, `api.openai.com`, `generativelanguage.googleapis.com`
- **Web:** `claude.ai`, `chatgpt.com`, `gemini.google.com`

**Status sources:**

- Anthropic: `https://status.anthropic.com/api/v2/status.json`
- OpenAI: `https://status.openai.com/api/v2/status.json`
- Gemini: **no public JSON API.** Detail window shows a "View Official Status" button that opens `https://aistudio.google.com/status` in the default browser. The app's own HTTPS probe is the only automated signal for Gemini.

### 4.3 UI Layer

| Component | Responsibility |
|---|---|
| `Tray` | Lightning bolt icon (blue / yellow / red `.ico` files embedded via `go:embed`); right-click menu; left-click opens `DetailWindow` |
| `DetailWindow` | On-demand fyne window showing full IP info, per-service probe results, Statuspage status; destroyed on close to release memory |
| `SettingsWindow` | On-demand fyne window for proxy / notification / interval / language / theme / autostart configuration |

### 4.4 External Dependencies

- `fyne.io/fyne/v2` — cross-platform GUI (detail + settings windows)
- `getlantern/systray` — cross-platform tray icon and menu
- `gen2brain/beeep` — cross-platform OS notifications
- `goccy/go-yaml` or `go-yaml/yaml` v3 — YAML parsing with comment round-trip support (viper discards comments, so we avoid it here; the implementation plan will validate the specific library choice)
- `fsnotify/fsnotify` — file change detection for config hot-reload
- `sirupsen/logrus` or `rs/zerolog` — structured logging
- Standard library: `net/http`, `encoding/json`, `context`, `sync`, `embed`

## 5. Data Flow

**Single tick cycle (every 60s by default):**

```
[Scheduler Tick]
       │
       ├──► [HTTPSProber] ──► Concurrent probes of 6 endpoints (goroutines)
       │                       │
       │                       └─► Each result pushed to StateBus
       │
       ├──► [IPProber] (every 5 ticks only) ─► public IP + geo → StateBus
       │
       └──► [StatuspageProber] (fixed 5-min cadence) ─► Anthropic + OpenAI → StateBus

       StateBus receives new state
             │
             ├──► Tray: recompute overall status color → update lightning icon
             │
             ├──► Notifier: diff vs previous state → fire toast if a "notify_on_down" service transitioned down
             │
             ├──► DetailWindow (if open): refresh UI
             │
             └──► Logger: write structured log entry
```

### 5.1 Overall Status Color Rules

- 🔵 **Blue (Savitar)** — all 6 HTTPS probes succeed AND both Statuspage sources report `operational`
- 🟡 **Yellow (Reverse Flash)** — any probe latency > 1s, OR any Statuspage reports `degraded` / `partial_outage`, OR 1-2 HTTPS probes fail
- 🔴 **Red** — 3 or more HTTPS probes fail, OR any Statuspage reports `major_outage`

### 5.2 Concurrency Model

- Single `main` goroutine running `Scheduler`
- Per-tick: up to 9 probe goroutines (6 HTTPS + 1 IP + 2 Statuspage)
- `StateBus` uses channels for result delivery to avoid mutex contention
- UI layer subscribes to StateBus via a separate buffered channel

### 5.3 Why Reduced IP Probe Frequency?

Proxy exit nodes rarely change during a session. Querying every 60s wastes bandwidth and consumes free-tier quota on third-party geo APIs (typical limit: 1000 requests/day). Every 5 minutes is plenty responsive for detecting proxy node switches.

## 6. Adaptive Tick (Normal ⇄ Fast Mode)

Observations from the user's workflow: connectivity problems through a proxy are often intermittent. A fixed 60-second tick misses transient spikes; a fixed 10-second tick wastes resources when things are stable. The adaptive model gets both benefits.

### 6.1 Entering Fast Mode (60s → 10s)

Trigger if **either** condition holds on any HTTPS probe, comparing consecutive ticks:

- `|latency_current - latency_previous| > 1000ms`
- State transition: up→down or down→up

### 6.2 Exiting Fast Mode (10s → 60s)

Return to normal tick when **either** condition holds:

- 3 consecutive ticks where **all** probes show `|Δlatency| < 300ms` AND no state transitions
- Fast mode has been active for 10 consecutive minutes (hard safety cap to prevent runaway polling if something is permanently flaky)

### 6.3 Scope

Adaptive tick applies **only** to `HTTPSProber`. `IPProber` and `StatuspageProber` keep their slower fixed cadences because:

- Third-party APIs (Statuspage, ipapi.co) enforce rate limits; 10s polling would get us banned
- Their measurements don't benefit from high frequency (platform-wide incidents aren't 10-second events)

### 6.4 UI Feedback

The detail window shows a small badge in the top-right corner:

- `🐢 Normal (60s)` — default
- `⚡ Fast (10s)` — active adaptive mode

Entering fast mode also writes a log entry with the triggering probe name and Δlatency value.

## 7. Error Handling & Edge Cases

### 7.1 Probe Failures

| Scenario | Behavior |
|---|---|
| HTTPS request timeout | Record as "down", retry next tick, do not block other probes |
| TLS handshake failure | Record as "down", log with TLS error detail (helps diagnose cert / MITM issues) |
| DNS resolution failure | Record as "down", log as "DNS error" (common proxy DNS misconfiguration signal) |
| Statuspage API returns 5xx / times out | Show "Status Unknown" (grey indicator); does not affect the self-probe results |
| IP geo API rate-limited (429) | Retain last-known values, log warning, retry next cycle |
| Proxy unreachable (all probes fail) | Red lightning + "Proxy may be down" toast (if notifications enabled) |

### 7.2 Config & Startup

| Scenario | Behavior |
|---|---|
| Config file missing | Create default `config.yaml`, continue running |
| Config file malformed | Show dialog with error line number, fall back to defaults for this session |
| Hot-reloaded config invalid | Roll back to previous valid config, log warning |
| Second instance launched | Detect existing instance (named mutex on Windows), activate its DetailWindow, exit second instance |

### 7.3 Network Changes

- When switching between proxy and direct connection, OS proxy settings change. SpeedForce re-reads system proxy on every tick (cheap), and triggers an immediate probe on detected change rather than waiting for the next tick.
- On Windows, this uses a combination of re-reading the registry and responding to `InternetGetConnectedState` events.

### 7.4 Notification Rules

Two independent notification sources:

1. **HTTPS probe failures** — fire a toast when a service transitions from reachable to unreachable, filtered by its per-service `notify_on_down` flag in config
2. **Statuspage major outages** — fire a toast when Anthropic or OpenAI Statuspage reports `major_outage`, controlled by the single `probes.statuspage.notify_on_major_outage` flag (`degraded` / `partial_outage` are shown in UI but do not notify, to avoid alert fatigue)

**De-duplication (applies to both sources):**

- A given service going down fires one notification
- Subsequent state oscillation (flapping) within 5 minutes does not re-notify
- Recovery (down → up) does **not** fire a notification — users don't want to be woken at night by "everything's fine" messages

### 7.5 Gemini-Specific

Because no official Gemini status API exists, notifications and status messages must be carefully worded:

- Notification text: `"Gemini connection failed (self-probe)"` — NOT `"Gemini is down"` (we don't know that)
- Detail window label: `"Gemini — reachability: ✓ | Official status: click to open"` — makes the distinction explicit

### 7.6 Logs

- One file per day: `logs/probe-YYYY-MM-DD.log`
- Auto-delete files older than 30 days (configurable)
- "Export Logs" button in settings → zip the last 7 days into `speedforce-logs-<timestamp>.zip` for bug reports

## 8. Configuration

### 8.1 Storage Location

**Windows:** `%APPDATA%\SpeedForce\`

```
%APPDATA%\SpeedForce\
├── config.yaml
├── locales/
│   ├── zh.json
│   └── en.json
└── logs/
    └── probe-YYYY-MM-DD.log
```

### 8.2 `config.yaml` Schema

```yaml
version: 1
language: zh  # zh | en

network:
  proxy:
    mode: auto       # auto | manual | none
    manual_url: ""   # used only when mode=manual, e.g. "http://127.0.0.1:7890"
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
    refresh_interval_sec: 300   # fixed cadence; independent of adaptive tick
    notify_on_major_outage: true  # notify regardless of per-service HTTPS notify flags
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
  theme: auto  # auto | light | dark

logging:
  level: info  # debug | info | warn | error
  retention_days: 30
```

### 8.3 Configuration Principles

- **Everything configurable** in YAML, but the Settings UI exposes only commonly-needed items (proxy, notifications, interval, language, autostart, theme). Advanced keys remain YAML-only.
- **Hot-reload** via `fsnotify`: saved changes take effect immediately without restart
- **Version field** enables future schema migrations
- **Comment preservation**: when the Settings UI writes changes, user-authored YAML comments are preserved (use a YAML library that supports this, or a round-trip parser)

### 8.4 What Is NOT Persisted

- Probe result history (in-memory 1-hour ring buffer only)
- IP query results (always fetched fresh)
- Notification cooldown state (reset on restart)

## 9. Testing Strategy

Testing is layered: deep unit coverage, integration tests for wired-up flows, and a small manual smoke checklist before each release.

### 9.1 Unit Tests

| Module | Test Focus |
|---|---|
| `Scheduler` | Adaptive tick transitions: given a sequence of mock latencies, assert correct entry / exit of fast mode, correct 10-minute cap |
| `StateBus` | Subscribe/publish semantics, no data races (`go test -race`) |
| `Notifier` | State transition detection, 5-minute flap cooldown, filtering by `notify_on_down` |
| `HTTPSProber` | Use `httptest.Server` to simulate 2xx/3xx/4xx/5xx/timeout/TLS failures; assert correct classification |
| `StatuspageProber` | Sample JSON fixtures for operational / degraded / major_outage states |
| `ConfigManager` | YAML parsing, hot-reload, rollback on invalid config, comment preservation |
| `I18n` | Locale switching, fallback to default language on missing key |

### 9.2 Integration Tests

- Spin up a local mock HTTP server in place of all three providers; run the full probe → StateBus → Notifier chain
- Run with `-race` to verify no deadlocks or races across the ~9 concurrent goroutines

### 9.3 Manual Smoke Checklist (Pre-Release)

1. Double-click `.exe` → blue lightning appears in tray
2. Disconnect network → icon turns red within 10-20s + toast fires
3. Reconnect → returns to blue (no toast; recovery suppressed by design)
4. Toggle proxy on/off → icon state updates correctly
5. Open DetailWindow → close → Task Manager shows memory release
6. Edit `config.yaml` → change takes effect without restart
7. Switch language zh↔en → all UI text updates

### 9.4 CI

- GitHub Actions: `go test -race ./...` + `go build` + `golangci-lint run`
- Every push to `main` produces a Windows build artifact
- Tagged commits (v0.x.y) trigger a release workflow that attaches the `.exe` to a GitHub Release

### 9.5 Coverage Targets

- Core layer ≥ 80%
- Probe layer ≥ 70%
- UI layer: not measured; manual smoke is more reliable

### 9.6 Developer Debug Flags (Not in Production Builds)

- `--fake-down=claude-api` — simulate a service being down (for testing UI + notifications)
- `--tick=1` — shorten tick interval to 1 second (for testing adaptive transitions)

## 10. Project Structure

```
speedforce/
├── cmd/
│   └── speedforce/
│       └── main.go                 # entrypoint: init + start Scheduler + Tray
├── internal/
│   ├── core/
│   │   ├── scheduler.go            # adaptive-tick scheduler
│   │   ├── statebus.go             # state store and pub/sub
│   │   ├── notifier.go             # notification dispatch
│   │   └── types.go                # shared structs (ProbeResult, State, etc.)
│   ├── probe/
│   │   ├── https.go                # HTTPSProber
│   │   ├── ip.go                   # IPProber
│   │   ├── statuspage.go           # StatuspageProber
│   │   └── httpclient.go           # proxy-aware shared HTTP client
│   ├── config/
│   │   ├── config.go               # load / hot-reload / validate
│   │   └── defaults.go             # default config generation
│   ├── i18n/
│   │   ├── i18n.go                 # language-pack loading and lookups
│   │   └── locales/
│   │       ├── zh.json
│   │       └── en.json
│   ├── ui/
│   │   ├── tray/
│   │   │   ├── tray.go             # tray icon + menu
│   │   │   └── icons.go            # 3 lightning ICOs embedded via go:embed
│   │   ├── detail/
│   │   │   └── window.go           # detail window (fyne)
│   │   └── settings/
│   │       └── window.go           # settings window (fyne)
│   ├── platform/
│   │   ├── proxy_windows.go        # read Windows registry for system proxy (build tag: windows)
│   │   ├── proxy_darwin.go         # placeholder (build tag: darwin)
│   │   ├── proxy_linux.go          # placeholder (build tag: linux)
│   │   ├── autostart_windows.go    # autostart via registry
│   │   └── autostart_*.go          # placeholders for other platforms
│   └── logger/
│       └── logger.go               # structured logging + rotation
├── assets/
│   ├── icons/                      # source PNG/SVG (for embed)
│   └── speedforce.ico              # exe icon
├── docs/
│   ├── superpowers/specs/          # design documents
│   └── README.md                   # open-source landing page
├── .github/workflows/
│   ├── ci.yml                      # test + lint
│   └── release.yml                 # tag-triggered build
├── go.mod
├── go.sum
├── LICENSE                         # MIT (open-source friendly)
└── README.md
```

### 10.1 Cross-Platform Readiness

- `internal/platform/` uses Go build tags; `darwin`/`linux` files are stubs now
- Core business logic and UI use fyne/systray (cross-platform libraries), so ~95% of code compiles as-is on macOS and Linux
- Future macOS support requires: filling in `proxy_darwin.go` + `autostart_darwin.go` + producing a `.icns` icon

## 11. Milestones

| Milestone | Scope | Estimate (solo, full-time) |
|---|---|---|
| **M1: Skeleton** | Project scaffold, HTTPS prober unit, minimal tray (color changes only, no windows) | 2-3 days |
| **M2: Core Complete** | StateBus, Scheduler (with adaptive tick), full probe layer (IP + Statuspage) | 3-5 days |
| **M3: UI** | DetailWindow, SettingsWindow, three-color lightning icon switching, i18n (zh + en) | 3-5 days |
| **M4: Feature-Complete** | Config persistence + hot-reload, notifications, autostart, log export | 2-3 days |
| **M5: Polish + Release** | Test coverage, docs, GitHub Actions, v0.1 release | 2-3 days |

**Total: ~12-19 days full-time, or ~5-8 weeks at 2 hours/day.**

The user has opted to wait until M5 completes before taking the build, rather than using intermediate drops.

## 12. Future Work (Post v0.1)

- **macOS and Linux support** (fill `platform/*_darwin.go` + `*_linux.go`, provide `.icns` and Linux equivalents)
- **Persistent history** (SQLite, last 7 days of probe results)
- **Mini sparklines** in DetailWindow showing short-term latency trend
- **Custom probe endpoints** — let users add their own URLs beyond the built-in 6
- **Per-service thresholds** — different "slow" definitions per endpoint
- **Icon theme packs** — allow users to skin the lightning with alternate color themes

## 13. Open Questions

None at spec-approval time. All major design decisions are settled; implementation plan will enumerate remaining tactical choices (specific library versions, exact `fyne` layout primitives, etc.).
