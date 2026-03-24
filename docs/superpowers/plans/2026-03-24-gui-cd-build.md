# GUI CD Build Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add cross-platform GUI binary builds to the release pipeline and update the install script to optionally install the GUI.

**Architecture:** Goreleaser continues to build the CLI. A new matrix job builds the GUI natively on Linux, macOS (Intel + ARM), and Windows using `wails build`. A checksums job consolidates GUI checksums. The install script prompts for optional GUI installation.

**Tech Stack:** GitHub Actions, Wails v2 CLI, `gh` CLI for release uploads

**Spec:** `docs/superpowers/specs/2026-03-24-gui-cd-build-design.md`

---

### Task 1: Add GUI matrix job to release workflow

**Files:**
- Modify: `.github/workflows/release.yml`

- [ ] **Step 1: Add the `gui` job with matrix strategy**

Add after the existing `release` job. The job needs `release` to complete first so the GitHub Release exists. Matrix includes 4 entries for the platform/arch combinations.

```yaml
  gui:
    needs: release
    strategy:
      matrix:
        include:
          - runner: ubuntu-24.04
            os: linux
            arch: amd64
          - runner: macos-13
            os: darwin
            arch: amd64
          - runner: macos-15
            os: darwin
            arch: arm64
          - runner: windows-latest
            os: windows
            arch: amd64
    runs-on: ${{ matrix.runner }}
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"

      - uses: actions/setup-node@v4
        with:
          node-version: "20"

      - name: Install Linux deps
        if: matrix.os == 'linux'
        run: sudo apt-get update && sudo apt-get install -y libgtk-3-dev libwebkit2gtk-4.1-dev

      - name: Install Wails CLI
        run: go install github.com/wailsapp/wails/v2/cmd/wails@v2.11.0

      - name: Build GUI
        run: wails build -production -tags webkit2_41 -ldflags "-s -w"

      - name: Archive (unix)
        if: matrix.os != 'windows'
        run: |
          VERSION="${GITHUB_REF_NAME#v}"
          tar czf "rook-gui_${VERSION}_${{ matrix.os }}_${{ matrix.arch }}.tar.gz" -C build/bin rook-gui

      - name: Archive (windows)
        if: matrix.os == 'windows'
        shell: pwsh
        run: |
          $VERSION = "${{ github.ref_name }}".TrimStart("v")
          Compress-Archive -Path "build/bin/rook-gui.exe" -DestinationPath "rook-gui_${VERSION}_windows_${{ matrix.arch }}.zip"

      - name: Upload to release
        shell: bash
        run: gh release upload --clobber "${{ github.ref_name }}" rook-gui_*
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 2: Verify YAML syntax**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))"`
Expected: No output (valid YAML)

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "feat: add GUI build matrix to release workflow"
```

---

### Task 2: Add GUI checksums job to release workflow

**Files:**
- Modify: `.github/workflows/release.yml`

- [ ] **Step 1: Add the `gui-checksums` job**

Add after the `gui` job. This job downloads all GUI archives from the release, computes SHA256 checksums, and uploads `checksums-gui.txt`.

```yaml
  gui-checksums:
    needs: gui
    runs-on: ubuntu-24.04
    steps:
      - uses: actions/checkout@v4

      - name: Download GUI archives
        run: gh release download "${{ github.ref_name }}" --pattern "rook-gui_*" --dir artifacts
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      - name: Generate checksums
        run: cd artifacts && sha256sum rook-gui_* > checksums-gui.txt

      - name: Upload checksums
        run: gh release upload --clobber "${{ github.ref_name }}" artifacts/checksums-gui.txt
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

- [ ] **Step 2: Verify YAML syntax**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))"`
Expected: No output (valid YAML)

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "feat: add GUI checksums job to release workflow"
```

---

### Task 3: Update install script with GUI prompt

**Files:**
- Modify: `install.sh`

- [ ] **Step 1: Add `download` helper function**

Extract the download logic into a reusable function since both CLI and GUI need it. Add after the `INSTALL_DIR` line, before `main()`:

```sh
download() {
    url="$1"
    dest="$2"
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url" -o "$dest"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$dest" "$url"
    else
        echo "Error: curl or wget required"
        exit 1
    fi
}

fetch_tag() {
    tmpfile=$(mktemp)
    download "https://api.github.com/repos/$REPO/releases/latest" "$tmpfile"
    grep '"tag_name"' "$tmpfile" | cut -d'"' -f4
    rm -f "$tmpfile"
}

install_binary() {
    name="$1"
    version="$2"
    tag="$3"
    os="$4"
    arch="$5"
    tmpdir="$6"

    artifact="${name}_${version}_${os}_${arch}.tar.gz"
    url="https://github.com/$REPO/releases/download/$tag/$artifact"

    echo "Installing $name $tag for $os/$arch..."

    download "$url" "$tmpdir/${name}.tar.gz"
    tar -xzf "$tmpdir/${name}.tar.gz" -C "$tmpdir"
    chmod +x "$tmpdir/$name"
    mv "$tmpdir/$name" "$INSTALL_DIR/$name"

    echo "Installed $name to $INSTALL_DIR/$name"
}
```

- [ ] **Step 2: Rewrite `main()` to use helpers and add GUI prompt**

```sh
main() {
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m)

    case "$os" in
        linux)  ;;
        darwin) ;;
        *)      echo "Unsupported OS: $os"; exit 1 ;;
    esac

    case "$arch" in
        x86_64|amd64) arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        *) echo "Unsupported architecture: $arch"; exit 1 ;;
    esac

    tag=$(fetch_tag)
    if [ -z "$tag" ]; then
        echo "Error: could not determine latest release"
        exit 1
    fi

    version="${tag#v}"

    mkdir -p "$INSTALL_DIR"

    tmpdir=$(mktemp -d)
    trap 'rm -rf "$tmpdir"' EXIT

    install_binary "rook" "$version" "$tag" "$os" "$arch" "$tmpdir"

    # GUI install: prompt if interactive, or honor ROOK_GUI env var
    install_gui=false
    if [ "${ROOK_GUI:-}" = "1" ]; then
        install_gui=true
    elif [ -t 0 ]; then
        printf "Would you also like to install rook-gui? [y/N] "
        read -r answer
        case "$answer" in
            [yY]|[yY][eE][sS]) install_gui=true ;;
        esac
    fi

    if [ "$install_gui" = true ]; then
        install_binary "rook-gui" "$version" "$tag" "$os" "$arch" "$tmpdir"
    fi

    # Check PATH
    case ":$PATH:" in
        *":$INSTALL_DIR:"*) ;;
        *) echo "Warning: $INSTALL_DIR is not in your PATH. Add it with:"
           echo "  export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
    esac
}
```

- [ ] **Step 3: Test the script parses correctly**

Run: `sh -n install.sh`
Expected: No output (no syntax errors)

- [ ] **Step 4: Commit**

```bash
git add install.sh
git commit -m "feat: add optional GUI install to install script"
```

---

### Task 4: End-to-end validation

- [ ] **Step 1: Validate the complete workflow YAML**

Run: `python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))"`
Expected: No output (valid YAML)

- [ ] **Step 2: Validate install script syntax**

Run: `sh -n install.sh`
Expected: No output

- [ ] **Step 3: Dry-run review**

Read through both files in their final state. Verify:
- `gui` job has `needs: release`
- `gui-checksums` job has `needs: gui`
- All `GH_TOKEN` env vars are set
- Archive naming matches between workflow and install script
- Install script handles both interactive and non-interactive modes
