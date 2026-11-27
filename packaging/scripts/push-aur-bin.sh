#!/usr/bin/env bash
set -euo pipefail

AUR_REPO="ssh://aur@aur.archlinux.org/newscid-bin.git"
CLONE_DIR="/tmp/aur-newscid-bin"
VERSION=$(tr -d ' \n\r' < VERSION)

echo "==> Pushing NewScid-bin v${VERSION} to AUR (${AUR_REPO})..."

# Fetch live SHA256 directly from GitHub Release asset
ASSET_URL="https://github.com/chrisfair/newscid/releases/download/v${VERSION}/newscid-${VERSION}-linux-x86_64.tar.gz"
echo "==> Verifying live release asset checksum from ${ASSET_URL}..."
SUM=$(curl -sfL "${ASSET_URL}" | sha256sum | awk '{print $1}')
if [ -z "${SUM}" ] || [ "${#SUM}" -ne 64 ]; then
    echo "Error: failed to fetch release asset ${ASSET_URL}" >&2
    exit 1
fi

sed -i -E "s/^pkgver=.*/pkgver=${VERSION}/" packaging/pacman-bin/PKGBUILD
sed -i -E "s/^sha256sums_x86_64=\('[a-f0-9]+'\)/sha256sums_x86_64=('${SUM}')/" packaging/pacman-bin/PKGBUILD

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

if [ -n "$(git -C "${CLONE_DIR}" status --porcelain)" ]; then
    git -C "${CLONE_DIR}" add PKGBUILD .SRCINFO
    git -C "${CLONE_DIR}" commit -m "Update newscid-bin to v${VERSION}"
    git -C "${CLONE_DIR}" push -u origin HEAD:master
    echo "✔ Successfully updated AUR newscid-bin to v${VERSION}"
else
    echo "✔ AUR newscid-bin is already up to date with v${VERSION}"
fi
