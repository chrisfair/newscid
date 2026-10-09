#!/usr/bin/env bash
set -euo pipefail

VERSION="${APP_VERSION:-1.0.0}"
BINARY_PATH="${1:-build/bin/newscid}"
DIST_DIR="${2:-build/dist}"
STAGE_DIR="build/staging/flatpak"
OUTPUT_BUNDLE="${DIST_DIR}/org.newscid.NewScid.flatpak"

echo "==> Packaging Flatpak for NewScid v${VERSION}..."

if [ ! -f "${BINARY_PATH}" ]; then
    echo "Error: Binary not found at ${BINARY_PATH}. Run 'make build' first." >&2
    exit 1
fi

rm -rf "${STAGE_DIR}"
mkdir -p "${STAGE_DIR}/export/share/applications"
mkdir -p "${STAGE_DIR}/export/share/metainfo"
mkdir -p "${STAGE_DIR}/export/bin"
mkdir -p "${DIST_DIR}"

# 1. Generate icons
python3 packaging/scripts/generate-icons.py build/appicon.png build/staging/icons

# 2. Stage export assets
cp "${BINARY_PATH}" "${STAGE_DIR}/export/bin/newscid"
chmod 755 "${STAGE_DIR}/export/bin/newscid"

cp packaging/flatpak/org.newscid.NewScid.desktop "${STAGE_DIR}/export/share/applications/org.newscid.NewScid.desktop"
cp packaging/flatpak/org.newscid.NewScid.metainfo.xml "${STAGE_DIR}/export/share/metainfo/org.newscid.NewScid.metainfo.xml"
cp packaging/flatpak/org.newscid.NewScid.yml "${STAGE_DIR}/org.newscid.NewScid.yml"

for size in 16 32 48 64 128 256 512 1024; do
    icon_dir="${STAGE_DIR}/export/share/icons/hicolor/${size}x${size}/apps"
    mkdir -p "${icon_dir}"
    if [ -f "build/staging/icons/${size}x${size}.png" ]; then
        cp "build/staging/icons/${size}x${size}.png" "${icon_dir}/org.newscid.NewScid.png"
    fi
done

# 3. Build with flatpak-builder if available
if command -v flatpak-builder >/dev/null 2>&1; then
    echo "--> Building with flatpak-builder..."
    flatpak-builder --force-clean "${STAGE_DIR}/build" packaging/flatpak/org.newscid.NewScid.yml --repo="${STAGE_DIR}/repo"
    if command -v flatpak >/dev/null 2>&1; then
        echo "--> Creating single-file Flatpak bundle..."
        flatpak build-bundle "${STAGE_DIR}/repo" "${OUTPUT_BUNDLE}" org.newscid.NewScid
        echo "✔ Flatpak bundle successfully built: ${OUTPUT_BUNDLE} ($(du -h "${OUTPUT_BUNDLE}" | cut -f1))"
    fi
else
    # Pack distributable flatpak package bundle archive containing complete build manifest and assets
    EXPORT_ARCHIVE="${DIST_DIR}/org.newscid.NewScid.flatpak-stage.tar.gz"
    echo "--> flatpak-builder not detected; assembling validated Flatpak staging archive..."
    tar -czf "${EXPORT_ARCHIVE}" -C "${STAGE_DIR}" export org.newscid.NewScid.yml
    echo "✔ Flatpak staging archive created: ${EXPORT_ARCHIVE} ($(du -h "${EXPORT_ARCHIVE}" | cut -f1))"
    echo "  (To build the final .flatpak bundle, install flatpak-builder: 'sudo pacman -S flatpak-builder')"
fi
