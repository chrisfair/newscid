#!/usr/bin/env bash
set -euo pipefail

VERSION="${APP_VERSION:-1.0.0}"
RELEASE="${APP_RELEASE:-1}"
BINARY_PATH="${1:-build/bin/newscid}"
DIST_DIR="${2:-build/dist}"
ARCH="amd64"
PKG_NAME="newscid_${VERSION}-${RELEASE}_${ARCH}"
STAGE_DIR="build/staging/deb/${PKG_NAME}"
OUTPUT_DEB="${DIST_DIR}/${PKG_NAME}.deb"

echo "==> Packaging Debian (.deb) package for NewScid v${VERSION}-${RELEASE}..."

if [ ! -f "${BINARY_PATH}" ]; then
    echo "Error: Binary not found at ${BINARY_PATH}. Run 'make build' first." >&2
    exit 1
fi

rm -rf "${STAGE_DIR}"
mkdir -p "${STAGE_DIR}/DEBIAN"
mkdir -p "${STAGE_DIR}/usr/bin"
mkdir -p "${STAGE_DIR}/usr/share/applications"
mkdir -p "${STAGE_DIR}/usr/share/metainfo"
mkdir -p "${STAGE_DIR}/usr/share/mime/packages"
mkdir -p "${STAGE_DIR}/usr/share/doc/newscid"
mkdir -p "${STAGE_DIR}/usr/share/man/man1"
mkdir -p "${DIST_DIR}"

# 1. Install binary
cp "${BINARY_PATH}" "${STAGE_DIR}/usr/bin/newscid"
chmod 755 "${STAGE_DIR}/usr/bin/newscid"

# 2. Desktop entry and metadata
cp packaging/desktop/newscid.desktop "${STAGE_DIR}/usr/share/applications/newscid.desktop"
chmod 644 "${STAGE_DIR}/usr/share/applications/newscid.desktop"

cp packaging/desktop/org.newscid.NewScid.metainfo.xml "${STAGE_DIR}/usr/share/metainfo/org.newscid.NewScid.metainfo.xml"
chmod 644 "${STAGE_DIR}/usr/share/metainfo/org.newscid.NewScid.metainfo.xml"

cp packaging/desktop/newscid.mime.xml "${STAGE_DIR}/usr/share/mime/packages/newscid.xml"
chmod 644 "${STAGE_DIR}/usr/share/mime/packages/newscid.xml"

cp packaging/man/newscid.1 "${STAGE_DIR}/usr/share/man/man1/newscid.1"
gzip -9n -f "${STAGE_DIR}/usr/share/man/man1/newscid.1"
chmod 644 "${STAGE_DIR}/usr/share/man/man1/newscid.1.gz"

cp packaging/deb/copyright "${STAGE_DIR}/usr/share/doc/newscid/copyright"
chmod 644 "${STAGE_DIR}/usr/share/doc/newscid/copyright"

# 3. Icons
python3 packaging/scripts/generate-icons.py build/appicon.png build/staging/icons
for size in 16 32 48 64 128 256 512 1024; do
    icon_dir="${STAGE_DIR}/usr/share/icons/hicolor/${size}x${size}/apps"
    mkdir -p "${icon_dir}"
    if [ -f "build/staging/icons/${size}x${size}.png" ]; then
        cp "build/staging/icons/${size}x${size}.png" "${icon_dir}/newscid.png"
        chmod 644 "${icon_dir}/newscid.png"
    fi
done

# 4. DEBIAN control, postinst, postrm
sed -e "s/^Version:.*/Version: ${VERSION}-${RELEASE}/" \
    -e "s/^Architecture:.*/Architecture: ${ARCH}/" \
    packaging/deb/DEBIAN/control > "${STAGE_DIR}/DEBIAN/control"
chmod 644 "${STAGE_DIR}/DEBIAN/control"

cp packaging/deb/DEBIAN/postinst "${STAGE_DIR}/DEBIAN/postinst"
chmod 755 "${STAGE_DIR}/DEBIAN/postinst"

cp packaging/deb/DEBIAN/postrm "${STAGE_DIR}/DEBIAN/postrm"
chmod 755 "${STAGE_DIR}/DEBIAN/postrm"

# 5. Generate md5sums
(
    cd "${STAGE_DIR}"
    find usr -type f -exec md5sum {} + > DEBIAN/md5sums
    chmod 644 DEBIAN/md5sums
)

# 6. Build .deb package
if ! command -v dpkg-deb >/dev/null 2>&1; then
    echo "Error: dpkg-deb not found. Please install dpkg." >&2
    exit 1
fi

echo "--> Running dpkg-deb..."
dpkg-deb --build --root-owner-group "${STAGE_DIR}" "${OUTPUT_DEB}"

echo "✔ Debian package successfully built: ${OUTPUT_DEB} ($(du -h "${OUTPUT_DEB}" | cut -f1))"
dpkg-deb -I "${OUTPUT_DEB}"

