# SpeedForce ⚡

A lightweight Windows tray app that watches your ability to reach Claude, OpenAI, and Gemini — built for people who access them through a proxy.

🔵 all good &nbsp; 🟡 degraded &nbsp; 🔴 down

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

Download the latest `speedforce-vX.Y.Z-windows-amd64.zip` from the [releases page](https://github.com/yosh3289/speedforce/releases) and run `speedforce.exe`.

## Configuration

Config lives at `%APPDATA%\SpeedForce\config.yaml`. It is created with defaults on first run and hot-reloaded on change. See the [design doc](docs/superpowers/specs/2026-04-15-speedforce-design.md) for the full schema.

## Building from Source

```bash
git clone https://github.com/yosh3289/speedforce.git
cd speedforce
go build -ldflags="-H windowsgui -s -w" -o speedforce.exe ./cmd/speedforce
```

Requires Go 1.22+ and a C compiler (fyne's OpenGL driver uses CGO). On Windows, install MinGW-w64; on macOS/Linux, the system `cc` is usually sufficient.

## Development

```bash
go test ./...
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

Debug flags:
- `--tick=5` — override tick interval (seconds)
- `--fake-down="Claude API,OpenAI API"` — simulate services being down

## Roadmap

See [design doc §12](docs/superpowers/specs/2026-04-15-speedforce-design.md#12-future-work-post-v01) for planned work: macOS + Linux support, persistent history, sparklines, custom endpoints.

## License

MIT. See [LICENSE](LICENSE).
