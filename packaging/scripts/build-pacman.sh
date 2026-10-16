#!/usr/bin/env bash
set -euo pipefail

VERSION="${APP_VERSION:-1.0.0}"
RELEASE="${APP_RELEASE:-1}"
BINARY_PATH="${1:-build/bin/newscid}"
DIST_DIR="${2:-build/dist}"
STAGE_DIR="build/staging/pacman"

echo "==> Packaging Pacman (.pkg.tar.*) package for NewScid v${VERSION}-${RELEASE}..."

if [ ! -f "${BINARY_PATH}" ]; then
    echo "Error: Binary not found at ${BINARY_PATH}. Run 'make build' first." >&2
    exit 1
fi

rm -rf "${STAGE_DIR}"
mkdir -p "${STAGE_DIR}/icons"
mkdir -p "${DIST_DIR}"

# 1. Copy source files for PKGBUILD
cp "${BINARY_PATH}" "${STAGE_DIR}/newscid"
chmod 755 "${STAGE_DIR}/newscid"

cp packaging/desktop/newscid.desktop "${STAGE_DIR}/newscid.desktop"
cp packaging/desktop/org.newscid.NewScid.metainfo.xml "${STAGE_DIR}/org.newscid.NewScid.metainfo.xml"
cp packaging/desktop/newscid.mime.xml "${STAGE_DIR}/newscid.mime.xml"
cp packaging/man/newscid.1 "${STAGE_DIR}/newscid.1"
cp packaging/pacman/newscid.install "${STAGE_DIR}/newscid.install"

# 2. Copy icons
python3 packaging/scripts/generate-icons.py build/appicon.png "${STAGE_DIR}/icons"

# 3. Generate PKGBUILD for local staged binary
cat <<EOF > "${STAGE_DIR}/PKGBUILD"
# Maintainer: Christopher Patrick Fair <christopherpatrickfair@gmail.com>
pkgname=newscid
pkgver=${VERSION}
pkgrel=${RELEASE}
pkgdesc="Modern Chess Information Database (Scid vs. PC Rewrite)"
arch=('x86_64')
url="https://github.com/chrisfair/newscid"
license=('GPL-3.0-or-later')
depends=('gtk3' 'webkit2gtk-4.1' 'hicolor-icon-theme')
optdepends=(
    'stockfish: strong default UCI chess engine'
    'fruit: legacy UCI chess engine'
)
install="newscid.install"

package() {
    install -Dm755 "\${srcdir}/../newscid" "\${pkgdir}/usr/bin/newscid"
    install -Dm644 "\${srcdir}/../newscid.desktop" "\${pkgdir}/usr/share/applications/newscid.desktop"
    install -Dm644 "\${srcdir}/../org.newscid.NewScid.metainfo.xml" "\${pkgdir}/usr/share/metainfo/org.newscid.NewScid.metainfo.xml"
    install -Dm644 "\${srcdir}/../newscid.mime.xml" "\${pkgdir}/usr/share/mime/packages/newscid.xml"
    if [ -f "\${srcdir}/../newscid.1" ]; then
        install -Dm644 "\${srcdir}/../newscid.1" "\${pkgdir}/usr/share/man/man1/newscid.1"
    fi
    for size in 16 32 48 64 128 256 512 1024; do
        if [ -f "\${srcdir}/../icons/\${size}x\${size}.png" ]; then
            install -Dm644 "\${srcdir}/../icons/\${size}x\${size}.png" "\${pkgdir}/usr/share/icons/hicolor/\${size}x\${size}/apps/newscid.png"
        fi
    done
}
EOF

# 4. Check for makepkg
if ! command -v makepkg >/dev/null 2>&1; then
    echo "Error: makepkg not found. Install pacman/base-devel." >&2
    exit 1
fi

# 5. Build package with makepkg
echo "--> Running makepkg..."
(
    cd "${STAGE_DIR}"
    # Run makepkg skipping dependency checks since runtime dependencies are handled by pacman at install time
    makepkg -f --nodeps
)

# 6. Copy output package to dist
PKG_FILE=$(find "${STAGE_DIR}" -maxdepth 1 -name "newscid-${VERSION}-${RELEASE}-*.pkg.tar.*" | head -n 1)

if [ -n "${PKG_FILE}" ] && [ -f "${PKG_FILE}" ]; then
    cp "${PKG_FILE}" "${DIST_DIR}/"
    OUTPUT_PKG="${DIST_DIR}/$(basename "${PKG_FILE}")"
    echo "✔ Pacman package successfully built: ${OUTPUT_PKG} ($(du -h "${OUTPUT_PKG}" | cut -f1))"
else
    echo "Error: Failed to find generated pacman package in ${STAGE_DIR}" >&2
    exit 1
fi
