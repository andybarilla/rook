#!/bin/sh
set -eu

REPO="andybarilla/rook"
INSTALL_DIR="${ROOK_INSTALL_DIR:-$HOME/.local/bin}"

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
    tagfile="$1/release.json"
    download "https://api.github.com/repos/$REPO/releases/latest" "$tagfile"
    grep '"tag_name"' "$tagfile" | cut -d'"' -f4
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

    extract_dir="$tmpdir/${name}"
    mkdir -p "$extract_dir"
    download "$url" "$extract_dir/${name}.tar.gz"
    tar -xzf "$extract_dir/${name}.tar.gz" -C "$extract_dir"
    chmod +x "$extract_dir/$name"
    mv "$extract_dir/$name" "$INSTALL_DIR/$name"

    echo "Installed $name to $INSTALL_DIR/$name"
}

install_desktop_entry() {
    desktop_dir="${XDG_DATA_HOME:-$HOME/.local/share}/applications"
    mkdir -p "$desktop_dir"
    cat > "$desktop_dir/rook-gui.desktop" << DESKTOP
[Desktop Entry]
Name=Rook
Comment=Local development workspace manager
Exec=$INSTALL_DIR/rook-gui
Type=Application
Categories=Development;Utility;
StartupNotify=true
DESKTOP
    echo "Installed desktop entry to $desktop_dir/rook-gui.desktop"
}

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

    tmpdir=$(mktemp -d)
    trap 'rm -rf "$tmpdir"' EXIT

    tag=$(fetch_tag "$tmpdir")
    if [ -z "$tag" ]; then
        echo "Error: could not determine latest release"
        exit 1
    fi

    version="${tag#v}"

    mkdir -p "$INSTALL_DIR"

    install_binary "rook" "$version" "$tag" "$os" "$arch" "$tmpdir"

    # GUI install: prompt if interactive, or honor ROOK_GUI env var
    install_gui=false
    if [ "${ROOK_GUI:-}" = "1" ]; then
        install_gui=true
    elif [ -n "${ROOK_GUI:-}" ]; then
        install_gui=false
    elif [ -t 0 ]; then
        printf "Would you also like to install rook-gui? [y/N] "
        read -r answer
        case "$answer" in
            [yY]|[yY][eE][sS]) install_gui=true ;;
        esac
    fi

    if [ "$install_gui" = true ]; then
        install_binary "rook-gui" "$version" "$tag" "$os" "$arch" "$tmpdir"

        if [ "$os" = "linux" ]; then
            install_desktop_entry
        fi
    fi

    # Check PATH
    case ":$PATH:" in
        *":$INSTALL_DIR:"*) ;;
        *) echo "Warning: $INSTALL_DIR is not in your PATH. Add it with:"
           echo "  export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
    esac
}

main
