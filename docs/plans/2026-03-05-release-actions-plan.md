# Release GitHub Actions Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Create a GitHub Actions workflow that builds Flock for Linux, macOS, and Windows on tag push, then publishes a GitHub Release with the binaries.

**Architecture:** Single workflow file with a build matrix (3 OS jobs) followed by a release job that collects artifacts and creates a GitHub Release. Uses `wails build` for cross-platform desktop builds.

**Tech Stack:** GitHub Actions, Wails CLI, Go, Node.js, `softprops/action-gh-release`

---

### Task 1: Create the release workflow file

**Files:**
- Create: `.github/workflows/release.yml`

**Step 1: Create the workflow file**

```yaml
name: Release

on:
  push:
    tags:
      - "v*.*.*"

permissions:
  contents: write

jobs:
  build:
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            platform: linux
            archive_name: flock-linux-amd64.tar.gz
          - os: macos-latest
            platform: darwin
            archive_name: flock-darwin-amd64.tar.gz
          - os: windows-latest
            platform: windows
            archive_name: flock-windows-amd64.zip
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - uses: actions/setup-node@v4
        with:
          node-version: lts/*

      - name: Install Wails CLI
        run: go install github.com/wailsapp/wails/v2/cmd/wails@latest

      - name: Install Linux dependencies
        if: matrix.platform == 'linux'
        run: |
          sudo apt-get update
          sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev

      - name: Build (Linux)
        if: matrix.platform == 'linux'
        run: wails build -tags webkit2_41

      - name: Build (macOS/Windows)
        if: matrix.platform != 'linux'
        run: wails build

      - name: Archive (Linux/macOS)
        if: matrix.platform != 'windows'
        working-directory: build/bin
        run: tar czf ${{ matrix.archive_name }} *

      - name: Archive (Windows)
        if: matrix.platform == 'windows'
        working-directory: build/bin
        run: Compress-Archive -Path * -DestinationPath ${{ matrix.archive_name }}

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: ${{ matrix.archive_name }}
          path: build/bin/${{ matrix.archive_name }}

  release:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - name: Download all artifacts
        uses: actions/download-artifact@v4
        with:
          path: artifacts
          merge-multiple: true

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          generate_release_notes: true
          files: artifacts/*
```

**Step 2: Validate the YAML syntax**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))"`
Expected: No output (valid YAML)

If python3/yaml unavailable, use: `go run github.com/mikefarah/yq/v4@latest eval '.jobs | keys' .github/workflows/release.yml`
Expected: Prints `- build` and `- release`

**Step 3: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "feat: add release workflow for cross-platform builds

Builds Linux, macOS, and Windows binaries on tag push (v*.*.*),
then creates a GitHub Release with the archives attached."
```

### Task 2: Test the workflow with a dry-run tag

**Step 1: Push the branch and create a test tag**

Push the branch with the new workflow, then create a lightweight tag to trigger it:

```bash
git tag v0.0.1-rc.1
git push origin v0.0.1-rc.1
```

**Step 2: Monitor the workflow run**

```bash
gh run list --workflow=release.yml --limit=1
gh run watch  # watch the latest run
```

Expected: All 3 build jobs succeed, release job creates a GitHub Release at the tag.

**Step 3: Verify the release**

```bash
gh release view v0.0.1-rc.1
```

Expected: Release exists with 3 assets: `flock-linux-amd64.tar.gz`, `flock-darwin-amd64.tar.gz`, `flock-windows-amd64.zip`.

**Step 4: Clean up test release (optional)**

```bash
gh release delete v0.0.1-rc.1 --yes
git push --delete origin v0.0.1-rc.1
git tag -d v0.0.1-rc.1
```
