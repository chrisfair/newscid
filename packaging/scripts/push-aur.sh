#!/usr/bin/env bash
set -euo pipefail

AUR_REPO="ssh://aur@aur.archlinux.org/newscid.git"
CLONE_DIR="/tmp/aur-newscid"
VERSION=$(cat VERSION)

echo "==> Pushing NewScid v${VERSION} to AUR (${AUR_REPO})..."

# Ensure sha256sums and .SRCINFO accurately match the published tag tarball
echo "==> Verifying source checksums for v${VERSION}..."
if command -v updpkgsums >/dev/null 2>&1; then
    (cd packaging/pacman && updpkgsums && rm -rf src pkg newscid-*.tar.gz)
else
    TAG_URL="https://github.com/chrisfair/newscid/archive/refs/tags/v${VERSION}.tar.gz"
    SUM=$(curl -sL "${TAG_URL}" | sha256sum | awk '{print $1}')
    if [ -n "${SUM}" ] && [ "${#SUM}" -eq 64 ]; then
        sed -i -E "s/sha256sums=\('[a-f0-9]+'\)/sha256sums=('${SUM}')/" packaging/pacman/PKGBUILD
    fi
fi
if command -v makepkg >/dev/null 2>&1; then
    (cd packaging/pacman && makepkg --printsrcinfo > .SRCINFO)
fi

if [ ! -d "${CLONE_DIR}/.git" ]; then
    rm -rf "${CLONE_DIR}"
    git clone "${AUR_REPO}" "${CLONE_DIR}"
else
    git -C "${CLONE_DIR}" fetch origin
    git -C "${CLONE_DIR}" reset --hard origin/master
fi

cp packaging/pacman/PKGBUILD "${CLONE_DIR}/PKGBUILD"
cp packaging/pacman/.SRCINFO "${CLONE_DIR}/.SRCINFO"

if [ -n "$(git -C "${CLONE_DIR}" status --porcelain)" ]; then
    git -C "${CLONE_DIR}" commit -am "Update to v${VERSION}"
    git -C "${CLONE_DIR}" push origin master
    echo "✔ Successfully updated AUR package to v${VERSION}"
else
    echo "✔ AUR package is already up to date with v${VERSION}"
fi
