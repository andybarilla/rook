# GUI CD Build Design

## Overview

Add GUI (`rook-gui`) binaries to the existing release pipeline. The CLI build (goreleaser) stays unchanged. A new matrix job builds the GUI natively on each platform and attaches binaries to the same GitHub Release. The install script is updated to optionally install the GUI.

## Approach

Goreleaser handles CLI as-is. A separate `gui` job in the release workflow builds the GUI natively on ubuntu-latest, macos-latest, and windows-latest using `wails build -production`, then uploads archives to the release goreleaser created.

## Release Workflow Changes

### New `gui` job

Runs after the `release` job (needs the GitHub Release to exist). Matrix across three OSes.

**Matrix:**

| Runner | OS label | Arch | Platform deps |
|--------|----------|------|---------------|
| `ubuntu-latest` | `linux` | `amd64` | `libgtk-3-dev`, `libwebkit2gtk-4.0-dev` |
| `macos-latest` | `darwin` | `amd64` | none (WebKit built-in) |
| `windows-latest` | `windows` | `amd64` | none (WebView2 bundled by Wails) |

**Steps per matrix entry:**

1. `actions/checkout@v4`
2. `actions/setup-go@v5` (go 1.22)
3. `actions/setup-node@v4` (for frontend build)
4. Install platform deps (Linux only)
5. Install Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
6. `wails build -production` in `cmd/rook-gui/`
7. Archive the binary:
   - Linux/macOS: `rook-gui_<version>_<os>_<arch>.tar.gz`
   - Windows: `rook-gui_<version>_windows_<arch>.zip`
8. Compute SHA256 checksum
9. Upload archive + checksum entry to the release via `gh release upload`

**Version extraction:** Strip `v` prefix from the git tag (e.g., `v0.1.0` -> `0.1.0`), matching goreleaser convention.

### Checksums

GUI checksums go in a separate `checksums-gui.txt` file to avoid race conditions from three matrix runners appending to the same file. Each runner uploads its own checksum line; a final step (or the last runner) consolidates them, or they're uploaded as individual files and combined. Simplest: each runner uploads `checksums-gui-<os>.txt`, then a post-matrix step downloads and merges into `checksums-gui.txt`.

Alternative: each runner just includes the checksum in a predictable filename and the install script computes its own check. Given the complexity of merging across matrix runners, the simplest approach is: each runner uploads its archive, and a `gui-checksums` job that `needs: gui` downloads all `rook-gui_*` archives, computes checksums, and uploads a single `checksums-gui.txt`.

## Archive Naming

Follows CLI convention:

- `rook-gui_<version>_linux_amd64.tar.gz`
- `rook-gui_<version>_darwin_amd64.tar.gz`
- `rook-gui_<version>_windows_amd64.zip`

## Install Script Changes

After installing the CLI:

1. Check if stdin is a terminal (`[ -t 0 ]`)
2. If interactive, prompt: `"Would you also like to install rook-gui? [y/N]"`
3. If `ROOK_GUI=1` is set, skip prompt and install GUI (for non-interactive use)
4. If yes, download `rook-gui_<version>_<os>_<arch>.tar.gz` and extract to `$ROOK_INSTALL_DIR`
5. If non-interactive and `ROOK_GUI` not set, skip GUI silently

## What Changes

| File | Action |
|------|--------|
| `.github/workflows/release.yml` | Add `gui` matrix job + `gui-checksums` job after `release` |
| `install.sh` | Add GUI prompt and download logic |

## What Doesn't Change

| File | Reason |
|------|--------|
| `.goreleaser.yml` | CLI build unchanged |
| `Makefile` | Local build unchanged |
| `cmd/rook-gui/` | No source changes |
