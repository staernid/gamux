#!/usr/bin/env bash
set -euo pipefail

GUI_BIN="${1:-bin/gamux-gui}"
OUTPUT="${2:-bin/gamux-gui-x86_64.AppImage}"
BUILD_DIR="$(mktemp -d -t gamux-appdir-XXXXXX)"

cleanup() {
    rm -rf "$BUILD_DIR"
}
trap cleanup EXIT

if [[ ! -f "$GUI_BIN" ]]; then
    echo "Error: GUI binary not found at $GUI_BIN. Build it first with 'make build-gui'." >&2
    exit 1
fi

echo "==> Preparing AppDir in $BUILD_DIR..."
APPDIR="$BUILD_DIR/AppDir"
mkdir -p "$APPDIR/usr/bin"
mkdir -p "$APPDIR/usr/share/applications"
mkdir -p "$APPDIR/usr/share/icons/hicolor/scalable/apps"

# Copy binary and assets
cp "$GUI_BIN" "$APPDIR/usr/bin/gamux-gui"
chmod +x "$APPDIR/usr/bin/gamux-gui"

cp assets/io.github.staernid.gamux.desktop "$APPDIR/usr/share/applications/io.github.staernid.gamux.desktop"
cp assets/io.github.staernid.gamux.desktop "$APPDIR/gamux-gui.desktop"
cp assets/io.github.staernid.gamux.desktop "$APPDIR/io.github.staernid.gamux.desktop"

cp assets/io.github.staernid.gamux.svg "$APPDIR/usr/share/icons/hicolor/scalable/apps/io.github.staernid.gamux.svg"
cp assets/io.github.staernid.gamux.svg "$APPDIR/io.github.staernid.gamux.svg"
cp assets/io.github.staernid.gamux.svg "$APPDIR/gamux-gui.svg"
cp assets/io.github.staernid.gamux.svg "$APPDIR/.DirIcon"

# Create standard AppRun launcher
cat <<'EOF' > "$APPDIR/AppRun"
#!/usr/bin/env bash
HERE="$(dirname "$(readlink -f "${0}")")"
export PATH="${HERE}/usr/bin:${PATH}"
export LD_LIBRARY_PATH="${HERE}/usr/lib:${HERE}/usr/lib/x86_64-linux-gnu:${LD_LIBRARY_PATH:-}"
export XDG_DATA_DIRS="${HERE}/usr/share:${XDG_DATA_DIRS:-/usr/local/share:/usr/share}"
exec "${HERE}/usr/bin/gamux-gui" "$@"
EOF
chmod +x "$APPDIR/AppRun"

mkdir -p "$(dirname "$OUTPUT")"

# Check for appimagetool
APPIMAGETOOL="${APPIMAGETOOL:-appimagetool}"
if ! command -v "$APPIMAGETOOL" &>/dev/null; then
    echo "==> appimagetool not found on PATH. Downloading standalone appimagetool..."
    TOOL_PATH="$BUILD_DIR/appimagetool"
    curl -fsSL -o "$TOOL_PATH" "https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage"
    chmod +x "$TOOL_PATH"
    export APPIMAGE_EXTRACT_AND_RUN=1
    APPIMAGETOOL="$TOOL_PATH"
fi

echo "==> Packaging AppImage with $APPIMAGETOOL..."
ARCH=x86_64 "$APPIMAGETOOL" "$APPDIR" "$OUTPUT"
echo "==> Successfully created $OUTPUT"
