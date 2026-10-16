#!/usr/bin/env bash
set -euo pipefail

VERSION="${APP_VERSION:-1.0.0}"
RELEASE="${APP_RELEASE:-1}"
BINARY_PATH="${1:-build/bin/newscid}"
DIST_DIR="${2:-build/dist}"
ARCH="x86_64"
PKG_NAME="newscid-${VERSION}-${RELEASE}.${ARCH}"
STAGE_DIR="build/staging/rpm/stage"
OUTPUT_RPM="${DIST_DIR}/${PKG_NAME}.rpm"

echo "==> Packaging RPM package for NewScid v${VERSION}-${RELEASE}..."

if [ ! -f "${BINARY_PATH}" ]; then
    echo "Error: Binary not found at ${BINARY_PATH}. Run 'make build' first." >&2
    exit 1
fi

rm -rf "${STAGE_DIR}"
mkdir -p "${STAGE_DIR}/usr/bin"
mkdir -p "${STAGE_DIR}/usr/share/applications"
mkdir -p "${STAGE_DIR}/usr/share/metainfo"
mkdir -p "${STAGE_DIR}/usr/share/mime/packages"
mkdir -p "${STAGE_DIR}/usr/share/doc/newscid"
mkdir -p "${STAGE_DIR}/usr/share/licenses/newscid"
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

if [ -f packaging/man/newscid.1 ]; then
    cp packaging/man/newscid.1 "${STAGE_DIR}/usr/share/man/man1/newscid.1"
    gzip -9n -f "${STAGE_DIR}/usr/share/man/man1/newscid.1"
    chmod 644 "${STAGE_DIR}/usr/share/man/man1/newscid.1.gz"
fi

if [ -f LICENSE ]; then
    cp LICENSE "${STAGE_DIR}/usr/share/licenses/newscid/LICENSE"
    chmod 644 "${STAGE_DIR}/usr/share/licenses/newscid/LICENSE"
fi

if [ -f README.md ]; then
    cp README.md "${STAGE_DIR}/usr/share/doc/newscid/README.md"
    chmod 644 "${STAGE_DIR}/usr/share/doc/newscid/README.md"
fi

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

# 4. Check for packaging tool (nfpm or rpmbuild)
NFPM_BIN=""
if command -v nfpm >/dev/null 2>&1; then
    NFPM_BIN="nfpm"
elif [ -x "${HOME}/go/bin/nfpm" ]; then
    NFPM_BIN="${HOME}/go/bin/nfpm"
fi

if [ -n "${NFPM_BIN}" ]; then
    echo "--> Building RPM via nfpm..."
    NFPM_CONFIG="build/staging/rpm/nfpm.yaml"
    cat << EOF > "${NFPM_CONFIG}"
name: "newscid"
arch: "amd64"
platform: "linux"
version: "${VERSION}"
release: "${RELEASE}"
section: "games"
priority: "optional"
maintainer: "Christopher Patrick Fair <christopherpatrickfair@gmail.com>"
description: |
  NewScid is a fresh, database-compatible rewrite of Scid vs. PC featuring a modern,
  intuitive user interface built with Wails v2 (Go backend) and Vue 3 (TypeScript + Vite).
vendor: "NewScid Project"
homepage: "https://github.com/chrisfair/newscid"
license: "GPL-3.0-or-later"
depends:
  - gtk3
  - webkit2gtk4.1
recommends:
  - stockfish
contents:
  - src: ${STAGE_DIR}/
    dst: /
    type: tree
overrides:
  rpm:
    scripts:
      postinstall: packaging/rpm/scripts/postinstall.sh
      postremove: packaging/rpm/scripts/postremove.sh
EOF

    "${NFPM_BIN}" package -f "${NFPM_CONFIG}" -p rpm -t "${OUTPUT_RPM}"
elif command -v rpmbuild >/dev/null 2>&1; then
    echo "--> Building RPM via rpmbuild..."
    RPMBUILD_DIR="build/staging/rpm/rpmbuild"
    mkdir -p "${RPMBUILD_DIR}"/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
    cp packaging/rpm/newscid.spec "${RPMBUILD_DIR}/SPECS/newscid.spec"
    
    # Create source tarball
    TAR_NAME="newscid-${VERSION}.tar.gz"
    tar --transform "s|^${STAGE_DIR}|newscid-${VERSION}|" -czf "${RPMBUILD_DIR}/SOURCES/${TAR_NAME}" -C "${STAGE_DIR}" .
    
    rpmbuild --define "_topdir $(pwd)/${RPMBUILD_DIR}" -bb "${RPMBUILD_DIR}/SPECS/newscid.spec"
    cp "${RPMBUILD_DIR}/RPMS/${ARCH}"/newscid-*.rpm "${OUTPUT_RPM}"
else
    echo "Error: Neither nfpm nor rpmbuild found. Please install nfpm ('go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest') or rpm-tools." >&2
    exit 1
fi

chmod 644 "${OUTPUT_RPM}"
echo "✔ RPM package successfully built: ${OUTPUT_RPM} ($(du -h "${OUTPUT_RPM}" | cut -f1))"
