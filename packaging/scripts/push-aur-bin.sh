#!/usr/bin/env bash
set -euo pipefail

AUR_REPO="ssh://aur@aur.archlinux.org/newscid-bin.git"
CLONE_DIR="/tmp/aur-newscid-bin"
VERSION=$(tr -d ' \n\r' < VERSION)

echo "==> Pushing NewScid-bin v${VERSION} to AUR (${AUR_REPO})..."

# Fetch live SHA256 directly from GitHub Release asset
PKGREL=$(grep -E '^pkgrel=' packaging/pacman-bin/PKGBUILD | cut -d= -f2)
ASSET_URL="https://github.com/chrisfair/newscid/releases/download/v${VERSION}/newscid-${VERSION}-${PKGREL}-linux-x86_64.tar.gz"
echo "==> Verifying live release asset checksum from ${ASSET_URL}..."
SUM=$(curl -sfL "${ASSET_URL}" | sha256sum | awk '{print $1}')
if [ -z "${SUM}" ] || [ "${#SUM}" -ne 64 ]; then
    echo "Error: failed to fetch release asset ${ASSET_URL}" >&2
    exit 1
fi

sed -i -E "s/^pkgver=.*/pkgver=${VERSION}/" packaging/pacman-bin/PKGBUILD
sed -i -E "s/'[a-f0-9]{64}'/'${SUM}'/" packaging/pacman-bin/PKGBUILD

if command -v makepkg >/dev/null 2>&1; then
    (cd packaging/pacman-bin && makepkg --printsrcinfo > .SRCINFO)
fi

if [ ! -d "${CLONE_DIR}/.git" ]; then
    rm -rf "${CLONE_DIR}"
    git clone "${AUR_REPO}" "${CLONE_DIR}"
else
    git -C "${CLONE_DIR}" fetch origin || true
    git -C "${CLONE_DIR}" reset --hard origin/master || true
fi

cp packaging/pacman-bin/PKGBUILD "${CLONE_DIR}/PKGBUILD"
cp packaging/pacman-bin/.SRCINFO "${CLONE_DIR}/.SRCINFO"
if [ -d packaging/pacman-bin/keys ]; then
    rm -rf "${CLONE_DIR}/keys"
    cp -a packaging/pacman-bin/keys "${CLONE_DIR}/keys"
fi

if [ -n "$(git -C "${CLONE_DIR}" status --porcelain)" ]; then
    git -C "${CLONE_DIR}" add PKGBUILD .SRCINFO
    [ -d "${CLONE_DIR}/keys" ] && git -C "${CLONE_DIR}" add keys
    git -C "${CLONE_DIR}" commit -m "Update newscid-bin to v${VERSION}"
    git -C "${CLONE_DIR}" push -u origin HEAD:master
    echo "✔ Successfully updated AUR newscid-bin to v${VERSION}"
else
    echo "✔ AUR newscid-bin is already up to date with v${VERSION}"
fi
