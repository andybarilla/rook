#!/bin/sh
set -eu

REPO="andybarilla/rook"
INSTALL_DIR="${ROOK_INSTALL_DIR:-$HOME/.local/bin}"

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

    # Get latest release tag
    if command -v curl >/dev/null 2>&1; then
        tag=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
    elif command -v wget >/dev/null 2>&1; then
        tag=$(wget -qO- "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
    else
        echo "Error: curl or wget required"
        exit 1
    fi

    if [ -z "$tag" ]; then
        echo "Error: could not determine latest release"
        exit 1
    fi

    # Strip v prefix for archive name
    version="${tag#v}"
    artifact="rook_${version}_${os}_${arch}.tar.gz"
    url="https://github.com/$REPO/releases/download/$tag/$artifact"

    echo "Installing rook $tag for $os/$arch..."
    mkdir -p "$INSTALL_DIR"

    # Download and extract
    tmpdir=$(mktemp -d)
    trap 'rm -rf "$tmpdir"' EXIT

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url" -o "$tmpdir/rook.tar.gz"
    else
        wget -qO "$tmpdir/rook.tar.gz" "$url"
    fi

    tar -xzf "$tmpdir/rook.tar.gz" -C "$tmpdir"
    chmod +x "$tmpdir/rook"
    mv "$tmpdir/rook" "$INSTALL_DIR/rook"

    echo "Installed rook to $INSTALL_DIR/rook"

    # Check PATH
    case ":$PATH:" in
        *":$INSTALL_DIR:"*) ;;
        *) echo "Warning: $INSTALL_DIR is not in your PATH. Add it with:"
           echo "  export PATH=\"$INSTALL_DIR:\$PATH\"" ;;
    esac
}

main
