#!/usr/bin/env bash
set -euo pipefail

AUR_REPO="ssh://aur@aur.archlinux.org/newscid.git"
CLONE_DIR="/tmp/aur-newscid"
VERSION=$(cat VERSION)

echo "==> Pushing NewScid v${VERSION} to AUR (${AUR_REPO})..."

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
