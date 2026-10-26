#!/usr/bin/env sh
set -e

# Default installation prefix (rootless default: ~/.local)
PREFIX="${PREFIX:-$HOME/.local}"
UNINSTALL=0

usage() {
    cat <<EOF
Usage: $(basename "$0") [options]

Options:
  --prefix <dir>   Installation directory prefix (default: \$HOME/.local)
  --uninstall      Uninstall gamux binaries, desktop entry, and icons
  -h, --help       Show this help message
EOF
    exit 0
}

while [ "$#" -gt 0 ]; do
    case "$1" in
        --prefix)
            PREFIX="$2"
            shift 2
            ;;
        --uninstall)
            UNINSTALL=1
            shift
            ;;
        -h|--help)
            usage
            ;;
        *)
            echo "Unknown option: $1" >&2
            usage
            ;;
    esac
done

# Resolve source directory where script is located
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$SCRIPT_DIR"

# Locate binaries and assets
if [ -f "$SCRIPT_DIR/gamux-gui" ]; then
    SRC_BIN_GUI="$SCRIPT_DIR/gamux-gui"
    SRC_BIN_CLI="$SCRIPT_DIR/gamux"
    SRC_ASSETS="$SCRIPT_DIR/assets"
elif [ -f "$SCRIPT_DIR/bin/gamux-gui" ]; then
    SRC_BIN_GUI="$SCRIPT_DIR/bin/gamux-gui"
    SRC_BIN_CLI="$SCRIPT_DIR/bin/gamux"
    SRC_ASSETS="$SCRIPT_DIR/assets"
elif [ -f "$SCRIPT_DIR/../bin/gamux-gui" ]; then
    SRC_BIN_GUI="$SCRIPT_DIR/../bin/gamux-gui"
    SRC_BIN_CLI="$SCRIPT_DIR/../bin/gamux"
    SRC_ASSETS="$SCRIPT_DIR/../assets"
else
    SRC_BIN_GUI=""
    SRC_BIN_CLI=""
    SRC_ASSETS="$SCRIPT_DIR/assets"
fi

BIN_DIR="$PREFIX/bin"
APP_DIR="$PREFIX/share/applications"
ICON_SCALABLE_DIR="$PREFIX/share/icons/hicolor/scalable/apps"
ICON_256_DIR="$PREFIX/share/icons/hicolor/256x256/apps"

if [ "$UNINSTALL" -eq 1 ]; then
    echo "==> Uninstalling gamux from $PREFIX..."
    rm -f "$BIN_DIR/gamux"
    rm -f "$BIN_DIR/gamux-gui"
    rm -f "$APP_DIR/io.github.staernid.gamux.desktop"
    rm -f "$ICON_SCALABLE_DIR/io.github.staernid.gamux.svg"
    rm -f "$ICON_256_DIR/io.github.staernid.gamux.png"

    if command -v update-desktop-database >/dev/null 2>&1; then
        update-desktop-database "$APP_DIR" >/dev/null 2>&1 || true
    fi
    if command -v gtk-update-icon-cache >/dev/null 2>&1; then
        gtk-update-icon-cache -q -t "$PREFIX/share/icons/hicolor" >/dev/null 2>&1 || true
    fi
    echo "==> Successfully uninstalled gamux."
    exit 0
fi

if [ -z "$SRC_BIN_GUI" ] || [ ! -f "$SRC_BIN_GUI" ]; then
    echo "Error: gamux-gui binary not found. Run 'make build' first or execute from the extracted release archive." >&2
    exit 1
fi

echo "==> Installing gamux to $PREFIX..."

mkdir -p "$BIN_DIR"
mkdir -p "$APP_DIR"
mkdir -p "$ICON_SCALABLE_DIR"
mkdir -p "$ICON_256_DIR"

# Install binaries
install -m 755 "$SRC_BIN_GUI" "$BIN_DIR/gamux-gui"
if [ -f "$SRC_BIN_CLI" ]; then
    install -m 755 "$SRC_BIN_CLI" "$BIN_DIR/gamux"
fi

# Install desktop entry and icons
if [ -f "$SRC_ASSETS/io.github.staernid.gamux.desktop" ]; then
    install -m 644 "$SRC_ASSETS/io.github.staernid.gamux.desktop" "$APP_DIR/io.github.staernid.gamux.desktop"
fi
if [ -f "$SRC_ASSETS/io.github.staernid.gamux.svg" ]; then
    install -m 644 "$SRC_ASSETS/io.github.staernid.gamux.svg" "$ICON_SCALABLE_DIR/io.github.staernid.gamux.svg"
fi
if [ -f "$SRC_ASSETS/io.github.staernid.gamux.png" ]; then
    install -m 644 "$SRC_ASSETS/io.github.staernid.gamux.png" "$ICON_256_DIR/io.github.staernid.gamux.png"
fi

# Update desktop & icon caches
if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database "$APP_DIR" >/dev/null 2>&1 || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1; then
    gtk-update-icon-cache -q -t "$PREFIX/share/icons/hicolor" >/dev/null 2>&1 || true
fi

echo "==> Installation complete!"
echo "    Binaries: $BIN_DIR/gamux, $BIN_DIR/gamux-gui"
echo "    Desktop:  $APP_DIR/io.github.staernid.gamux.desktop"

case ":$PATH:" in
    *":$BIN_DIR:"*) ;;
    *)
        echo ""
        echo "Notice: $BIN_DIR is not currently in your \$PATH."
        echo "You can launch the GUI via your application menu or add it to ~/.bashrc or ~/.zshrc:"
        echo "  export PATH=\"$BIN_DIR:\$PATH\""
        ;;
esac
