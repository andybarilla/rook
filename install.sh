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

main
