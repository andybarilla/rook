# GUI CD Build Design

## Overview

Add GUI (`rook-gui`) binaries to the existing release pipeline. The CLI build (goreleaser) stays unchanged. A new matrix job builds the GUI natively on each platform and attaches binaries to the same GitHub Release. The install script is updated to optionally install the GUI.

## Approach

Goreleaser handles CLI as-is. A separate `gui` job in the release workflow builds the GUI natively on each platform using `wails build -production`, then uploads archives to the release goreleaser created.

## Release Workflow Changes

### New `gui` job

Runs after the `release` job (needs the GitHub Release to exist). Matrix across platforms.

**Matrix:**

| Runner | OS label | Arch | Platform deps |
|--------|----------|------|---------------|
| `ubuntu-latest` | `linux` | `amd64` | `libgtk-3-dev`, `libwebkit2gtk-4.1-dev` |
| `macos-13` | `darwin` | `amd64` | none (WebKit built-in) |
| `macos-latest` | `darwin` | `arm64` | none (WebKit built-in) |
| `windows-latest` | `windows` | `amd64` | none (WebView2 bundled by Wails) |

Note: `macos-latest` resolves to Apple Silicon (arm64) runners. `macos-13` is needed for Intel (amd64) builds.

**Steps per matrix entry:**

1. `actions/checkout@v4`
2. `actions/setup-go@v5` (go 1.22)
3. `actions/setup-node@v4` (for frontend build)
4. Install platform deps (Linux only): `sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev`
5. Install Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0`
6. `wails build -production -tags webkit2_41 -ldflags "-s -w"` from the repository root (where `wails.json` lives). The `-tags webkit2_41` is needed on Linux for webkit2gtk 4.1 compatibility; it is harmless on other platforms.
7. Built binary is at `build/bin/rook-gui` (`build/bin/rook-gui.exe` on Windows)
8. Archive the binary:
   - Linux/macOS: `tar czf rook-gui_<version>_<os>_<arch>.tar.gz -C build/bin rook-gui`
   - Windows: zip `rook-gui_<version>_windows_<arch>.zip` containing `rook-gui.exe`
9. Upload archive to the release via `gh release upload` (requires `GITHUB_TOKEN` / `GH_TOKEN` env var)

**Version extraction:** Strip `v` prefix from the git tag (e.g., `v0.1.0` -> `0.1.0`), matching goreleaser convention.

### Checksums (`gui-checksums` job)

A separate `gui-checksums` job runs after all `gui` matrix entries complete (`needs: gui`). It:

1. Downloads all `rook-gui_*` archives from the release via `gh release download`
2. Computes SHA256 checksums for each archive
3. Writes `checksums-gui.txt` and uploads it to the release

This avoids race conditions from matrix runners writing to the same file.

## Archive Naming

Follows CLI convention:

- `rook-gui_<version>_linux_amd64.tar.gz`
- `rook-gui_<version>_darwin_amd64.tar.gz`
- `rook-gui_<version>_darwin_arm64.tar.gz`
- `rook-gui_<version>_windows_amd64.zip`

## Install Script Changes

After installing the CLI:

1. Check if stdin is a terminal (`[ -t 0 ]`)
2. If interactive, prompt: `"Would you also like to install rook-gui? [y/N]"`
3. If `ROOK_GUI=1` is set, skip prompt and install GUI (for non-interactive use)
4. If yes, download `rook-gui_<version>_<os>_<arch>.tar.gz` and extract to `$ROOK_INSTALL_DIR`
5. If non-interactive and `ROOK_GUI` not set, skip GUI silently

## Scope Limitations

- No `linux/arm64` GUI build (no ARM Linux GitHub runners available by default)
- No Windows support in the install script (matches existing behavior — Windows users download zip manually)
- No macOS code signing or notarization (Gatekeeper will warn on first launch; standard for unsigned dev tools)

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
