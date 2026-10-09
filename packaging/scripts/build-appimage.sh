#!/usr/bin/env bash
set -euo pipefail

# Configuration
VERSION="${APP_VERSION:-1.0.0}"
BINARY_PATH="${1:-build/bin/newscid}"
DIST_DIR="${2:-build/dist}"
STAGING_DIR="build/staging/appimage/AppDir"
CACHE_DIR="build/.cache"

echo "==> Packaging AppImage for NewScid v${VERSION}..."

if [ ! -f "${BINARY_PATH}" ]; then
    echo "Error: Binary not found at ${BINARY_PATH}. Run 'make build' first." >&2
    exit 1
fi

# Clean and create directories
rm -rf "${STAGING_DIR}"
mkdir -p "${STAGING_DIR}/usr/bin"
mkdir -p "${STAGING_DIR}/usr/share/applications"
mkdir -p "${STAGING_DIR}/usr/share/metainfo"
mkdir -p "${DIST_DIR}"
mkdir -p "${CACHE_DIR}"

# 1. Copy application binary
cp "${BINARY_PATH}" "${STAGING_DIR}/usr/bin/newscid"
chmod 755 "${STAGING_DIR}/usr/bin/newscid"

# 2. Copy launcher AppRun
cp packaging/appimage/AppRun "${STAGING_DIR}/AppRun"
chmod 755 "${STAGING_DIR}/AppRun"

# 3. Copy desktop entry
cp packaging/desktop/newscid.desktop "${STAGING_DIR}/newscid.desktop"
cp packaging/desktop/newscid.desktop "${STAGING_DIR}/usr/share/applications/newscid.desktop"

# 4. Copy AppStream metadata
cp packaging/desktop/org.newscid.NewScid.metainfo.xml "${STAGING_DIR}/usr/share/metainfo/org.newscid.NewScid.metainfo.xml"

# 5. Generate and install icons
python3 packaging/scripts/generate-icons.py build/appicon.png build/staging/icons
cp build/staging/icons/256x256.png "${STAGING_DIR}/newscid.png"

for size in 16 32 48 64 128 256 512 1024; do
    icon_dir="${STAGING_DIR}/usr/share/icons/hicolor/${size}x${size}/apps"
    mkdir -p "${icon_dir}"
    if [ -f "build/staging/icons/${size}x${size}.png" ]; then
        cp "build/staging/icons/${size}x${size}.png" "${icon_dir}/newscid.png"
    fi
done

# 6. Locate or download appimagetool
APPIMAGETOOL=""
if command -v appimagetool >/dev/null 2>&1; then
    APPIMAGETOOL="appimagetool"
elif [ -x "${CACHE_DIR}/appimagetool" ]; then
    APPIMAGETOOL="${CACHE_DIR}/appimagetool"
else
    echo "--> Downloading appimagetool..."
    curl -fsSL -L -o "${CACHE_DIR}/appimagetool" \
        "https://github.com/AppImage/AppImageKit/releases/download/continuous/appimagetool-x86_64.AppImage"
    chmod +x "${CACHE_DIR}/appimagetool"
    APPIMAGETOOL="${CACHE_DIR}/appimagetool"
fi

OUTPUT_FILE="${DIST_DIR}/NewScid-${VERSION}-x86_64.AppImage"
rm -f "${OUTPUT_FILE}"

echo "--> Generating AppImage with appimagetool..."
# Run appimagetool with -n (skip online network checks)
ARCH=x86_64 "${APPIMAGETOOL}" --appimage-extract-and-run -n "${STAGING_DIR}" "${OUTPUT_FILE}"

chmod +x "${OUTPUT_FILE}"
echo "✔ AppImage successfully built: ${OUTPUT_FILE} ($(du -h "${OUTPUT_FILE}" | cut -f1))"
