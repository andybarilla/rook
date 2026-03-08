<p align="center">
  <h1 align="center">Rook</h1>
  <p align="center">A cross-platform, open-source local development environment manager.</p>
  <p align="center">
    <a href="https://github.com/andybarilla/rook/actions/workflows/ci.yml"><img src="https://github.com/andybarilla/rook/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
    <a href="https://github.com/andybarilla/rook/releases/latest"><img src="https://img.shields.io/github/v/release/andybarilla/rook?include_prereleases&sort=semver" alt="Release"></a>
    <a href="https://github.com/andybarilla/rook/blob/main/LICENSE"><img src="https://img.shields.io/github/license/andybarilla/rook" alt="License"></a>
    <a href="https://goreportcard.com/report/github.com/andybarilla/rook"><img src="https://goreportcard.com/badge/github.com/andybarilla/rook" alt="Go Report Card"></a>
  </p>
  <p align="center">
    <a href="https://getrook.dev">Website</a> &middot;
    <a href="https://github.com/andybarilla/rook/releases/latest">Download</a> &middot;
    <a href="https://github.com/andybarilla/rook/issues">Issues</a>
  </p>
</p>

---

Rook is a community alternative to [Laravel Herd](https://herd.laravel.com/) — a native desktop app that manages local vhosts, SSL certificates, PHP runtimes, database services, and more. Built with [Go](https://go.dev) + [Wails](https://wails.io) + [Caddy](https://caddyserver.com), it runs on macOS, Linux, and Windows.

## Features

- **Automatic SSL** — Local HTTPS via [mkcert](https://github.com/FiloSottile/mkcert), zero configuration
- **PHP Management** — Multiple PHP versions with per-site FPM pools
- **Database Services** — MySQL, PostgreSQL, and Redis managed from the GUI
- **Node.js Support** — Per-site Node version selection
- **Plugin Architecture** — Extensible to any language stack via a plugin API
- **Cross-Platform** — Native desktop app for macOS, Linux, and Windows
- **System Tray** — Runs quietly in the background, always accessible

## Quick Start

### Prerequisites

- [Go](https://go.dev/dl/) 1.23+
- [Node.js](https://nodejs.org/) (LTS)
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)

**Linux (Debian/Ubuntu):**
```bash
sudo apt-get install libgtk-3-dev libwebkit2gtk-4.1-dev
```

**Linux (Fedora 43+):**
```bash
sudo dnf install gtk3-devel webkit2gtk4.1-devel
```

### Development

```bash
# Clone the repo
git clone https://github.com/andybarilla/rook.git
cd rook

# Install frontend dependencies
cd frontend && npm install && cd ..

# Run in dev mode (hot reload)
wails dev -tags webkit2_41    # Fedora 43+ (webkit2gtk 4.1)
wails dev                      # other Linux / macOS / Windows

# Build for production
wails build -tags webkit2_41  # Fedora 43+
wails build                    # other platforms
```

### Running Tests

```bash
# Go tests
go test ./...

# Frontend tests
cd frontend && npm test
```

## Architecture

```
rook/
├── internal/
│   ├── core/          # App lifecycle and wiring
│   ├── caddy/         # Embedded Caddy server management
│   ├── registry/      # Site registry (sites.json)
│   ├── plugin/        # Plugin host and interfaces
│   ├── ssl/           # mkcert SSL plugin
│   ├── php/           # PHP-FPM plugin
│   ├── databases/     # MySQL, PostgreSQL, Redis plugin
│   ├── node/          # Node.js plugin
│   ├── discovery/     # Plugin discovery and loading
│   ├── external/      # External plugin support
│   └── config/        # Configuration management
├── frontend/          # Svelte + Tailwind + DaisyUI
└── build/             # Build assets and packaging
```

**Core layers:**

1. **Plugin Host** — discovers, loads, and manages plugin lifecycle
2. **Caddy Manager** — embeds Caddy as a Go library; manages vhosts and TLS
3. **Site Registry** — persists local sites to `~/.config/rook/sites.json`
4. **Wails GUI** — native webview with Svelte frontend

See [`docs/plans/2026-03-03-rook-core-design.md`](docs/plans/2026-03-03-rook-core-design.md) for the full architecture document.

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Commit your changes
4. Push to the branch (`git push origin feature/my-feature`)
5. Open a Pull Request

Check the [roadmap](docs/ROADMAP.md) for planned features and current status.

## License

[MIT](LICENSE) — [Andy Barilla](https://andybarilla.com)

