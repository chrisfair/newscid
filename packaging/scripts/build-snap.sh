#!/usr/bin/env bash
set -euo pipefail

VERSION="${APP_VERSION:-1.0.0}"
BINARY_PATH="${1:-build/bin/newscid}"
DIST_DIR="${2:-build/dist}"
STAGE_DIR="build/staging/snap/stage"
SNAP_DIR="build/staging/snap/snap-root"
OUTPUT_SNAP="${DIST_DIR}/newscid_${VERSION}_amd64.snap"

echo "==> Packaging Snap for NewScid v${VERSION}..."

if [ ! -f "${BINARY_PATH}" ]; then
    echo "Error: Binary not found at ${BINARY_PATH}. Run 'make build' first." >&2
    exit 1
fi

rm -rf "${STAGE_DIR}" "${SNAP_DIR}"
mkdir -p "${STAGE_DIR}/usr/bin"
mkdir -p "${STAGE_DIR}/usr/share/applications"
mkdir -p "${STAGE_DIR}/usr/share/metainfo"
mkdir -p "${SNAP_DIR}/meta/gui"
mkdir -p "${SNAP_DIR}/usr/bin"
mkdir -p "${SNAP_DIR}/usr/share/applications"
mkdir -p "${SNAP_DIR}/usr/share/metainfo"
mkdir -p "${DIST_DIR}"

# 1. Copy binary and desktop assets to stage
cp "${BINARY_PATH}" "${STAGE_DIR}/usr/bin/newscid"
chmod 755 "${STAGE_DIR}/usr/bin/newscid"
cp packaging/desktop/newscid.desktop "${STAGE_DIR}/usr/share/applications/newscid.desktop"
cp packaging/desktop/org.newscid.NewScid.metainfo.xml "${STAGE_DIR}/usr/share/metainfo/org.newscid.NewScid.metainfo.xml"

# Copy to snap root layout
cp "${STAGE_DIR}/usr/bin/newscid" "${SNAP_DIR}/usr/bin/newscid"
cp "${STAGE_DIR}/usr/share/applications/newscid.desktop" "${SNAP_DIR}/usr/share/applications/newscid.desktop"
cp "${STAGE_DIR}/usr/share/metainfo/org.newscid.NewScid.metainfo.xml" "${SNAP_DIR}/usr/share/metainfo/org.newscid.NewScid.metainfo.xml"

# 2. Icons
python3 packaging/scripts/generate-icons.py build/appicon.png build/staging/icons
cp build/staging/icons/256x256.png "${SNAP_DIR}/meta/gui/icon.png"

for size in 16 32 48 64 128 256 512 1024; do
    mkdir -p "${STAGE_DIR}/usr/share/icons/hicolor/${size}x${size}/apps"
    mkdir -p "${SNAP_DIR}/usr/share/icons/hicolor/${size}x${size}/apps"
    if [ -f "build/staging/icons/${size}x${size}.png" ]; then
        cp "build/staging/icons/${size}x${size}.png" "${STAGE_DIR}/usr/share/icons/hicolor/${size}x${size}/apps/newscid.png"
        cp "build/staging/icons/${size}x${size}.png" "${SNAP_DIR}/usr/share/icons/hicolor/${size}x${size}/apps/newscid.png"
    fi
done

# 3. Create meta/snap.yaml
cat << EOF > "${SNAP_DIR}/meta/snap.yaml"
name: newscid
version: '${VERSION}'
summary: Modern Chess Information Database
description: |
  NewScid is a fresh, database-compatible rewrite of Scid vs. PC featuring a modern,
  intuitive user interface built with Wails v2 and Vue 3.
architectures:
  - amd64
base: core24
confinement: strict
grade: stable
apps:
  newscid:
    command: usr/bin/newscid
    desktop: meta/gui/newscid.desktop
    plugs:
      - desktop
      - desktop-legacy
      - wayland
      - x11
      - home
      - network
      - opengl
      - audio-playback
EOF

# Desktop entry for meta/gui
cat << EOF > "${SNAP_DIR}/meta/gui/newscid.desktop"
[Desktop Entry]
Name=NewScid
GenericName=Chess Database
Comment=Modern Chess Information Database (Scid vs. PC Rewrite)
Exec=newscid %F
Icon=\${SNAP}/meta/gui/icon.png
Terminal=false
Type=Application
Categories=Game;BoardGame;
EOF

# 4. Build snap package
if command -v snapcraft >/dev/null 2>&1; then
    echo "--> Building snap using snapcraft..."
    snapcraft --destructive-mode --output="${OUTPUT_SNAP}"
elif command -v mksquashfs >/dev/null 2>&1; then
    echo "--> Building snap using mksquashfs..."
    rm -f "${OUTPUT_SNAP}"
    mksquashfs "${SNAP_DIR}" "${OUTPUT_SNAP}" -noappend -comp xz -all-root -no-xattrs
else
    echo "Error: Neither snapcraft nor mksquashfs found." >&2
    exit 1
fi

echo "✔ Snap successfully built: ${OUTPUT_SNAP} ($(du -h "${OUTPUT_SNAP}" | cut -f1))"
