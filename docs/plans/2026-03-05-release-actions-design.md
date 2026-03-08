# Release GitHub Actions Design

## Trigger

Push a semver tag (`v*.*.*`) to trigger the release workflow.

## Build Matrix

Three parallel build jobs using `wails build`:

| Platform | Runner | Output | Archive |
|----------|--------|--------|---------|
| Linux | `ubuntu-latest` | `rook` binary | `rook-linux-amd64.tar.gz` |
| macOS | `macos-latest` | `rook.app` bundle | `rook-darwin-amd64.tar.gz` |
| Windows | `windows-latest` | `rook.exe` | `rook-windows-amd64.zip` |

Each build job:

1. Checkout code
2. Setup Go (from `go.mod`) + Node (lts/*)
3. Install Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
4. Install platform deps (Linux: libgtk-3-dev, libwebkit2gtk-4.1-dev)
5. Run `wails build` (with `-tags webkit2_41` on Linux)
6. Archive the output (tar.gz for Linux/macOS, zip for Windows)
7. Upload archive as workflow artifact

## Release Job

A final job that depends on all three build jobs:

1. Downloads all artifacts
2. Creates a GitHub Release using `softprops/action-gh-release`
3. Attaches the three archives to the release
4. Uses the tag as the release title
5. Auto-generates release notes from commits since last tag

## Out of Scope

- No test step (trusting CI has already passed on main)
- No platform installers (DMG, NSIS, AppImage)
- No code signing
- No arm64 builds
