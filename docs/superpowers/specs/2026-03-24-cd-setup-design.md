# CD Setup for Rook CLI

## Overview

Automate building and publishing cross-platform CLI binaries as GitHub Releases on version tags, plus a curl-pipe install script for end users.

## Trigger

```yaml
on:
  push:
    tags:
      - 'v*'
```

Fires only on tag push (e.g., `git tag v0.1.0 && git push --tags`), not on branch push or PR.

## Components

### 1. Goreleaser Config (`.goreleaser.yml`)

**Builds:**
- Source: `./cmd/rook`
- Binary name: `rook`
- Targets: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64
- ldflags: `-s -w` (strip debug symbols for smaller binaries)

**Archives:**
- Format: `tar.gz` for linux and macOS, `zip` for windows
- Naming: `rook_{{ .Version }}_{{ .Os }}_{{ .Arch }}`
- `{{ .Version }}` is the tag with the `v` prefix stripped (e.g., tag `v0.1.0` -> version `0.1.0`)
- Contents: binary only (no README, LICENSE, etc. unless they exist)

**Extra files:**
- `install.sh` attached to the release via goreleaser's `extra_files` config

**Checksums:**
- `checksums.txt` with SHA256 hashes for all archives

**Excluded:**
- No Homebrew tap
- No Docker image
- No snapcraft/scoop
- No changelog generation (release body left empty)

### 2. Release Workflow (`.github/workflows/release.yml`)

Single job — goreleaser handles everything including uploading `install.sh` via `extra_files`.

**Job: `release`**
- Runs on: `ubuntu-latest`
- Permissions: `contents: write`
- Steps:
  1. `actions/checkout@v4` with `fetch-depth: 0` (goreleaser needs full history for version detection)
  2. `actions/setup-go@v5` with go-version `1.22`
  3. `goreleaser/goreleaser-action@v6` with `goreleaser release`
  4. Uses `GITHUB_TOKEN` for release creation (no additional secrets needed)

### 3. Install Script (`install.sh`)

Adapted from the jackdaw install script pattern.

**Error handling:** `set -eu` at top (POSIX sh — `pipefail` is not portable). Failures (bad HTTP response, unsupported platform, missing tools) exit with a descriptive error message.

**Behavior:**
1. Detect OS (`uname -s` -> `linux`, `darwin`) and arch (`uname -m` -> `amd64`, `arm64`). Exit on unsupported OS or arch.
2. Fetch latest release tag from GitHub API. Try `curl -fsSL` first, fall back to `wget -qO-`.
3. Strip `v` prefix from tag to get version (e.g., `v0.1.0` -> `0.1.0`).
4. Construct archive name: `rook_<version>_<os>_<arch>.tar.gz`
5. Download archive, extract `rook` binary to `$ROOK_INSTALL_DIR` (default: `~/.local/bin`).
6. `chmod +x` the binary.
7. Warn if install dir is not in `$PATH`.

**Supported platforms:**
- Linux: amd64, arm64
- macOS: amd64, arm64
- Windows: not supported via install script (users download zip manually)

**Dependencies:** `curl` or `wget`, `tar`

## Archive Naming Convention

The install script and goreleaser must agree on naming. Goreleaser's `{{ .Os }}` and `{{ .Arch }}` produce:
- `linux`, `darwin`, `windows` for OS
- `amd64`, `arm64` for arch

Version is the tag with `v` stripped: tag `v0.1.0` -> version `0.1.0`.

Archives: `rook_<version>_<os>_<arch>.tar.gz` (or `.zip` for windows)

Example: tag `v0.1.0` -> `rook_0.1.0_linux_amd64.tar.gz`

## What Changes

| File | Action |
|------|--------|
| `.goreleaser.yml` | New |
| `.github/workflows/release.yml` | New |
| `install.sh` | New |

## What Doesn't Change

- `.github/workflows/test.yml` — unchanged
- GUI (`cmd/rook-gui/`) — excluded from releases
- `Makefile` — unchanged
