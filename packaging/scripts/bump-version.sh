#!/usr/bin/env bash
set -euo pipefail

VERSION_FILE="VERSION"
WAILS_FILE="wails.json"
PKG_VERSION_FILE="pkg/version/version.go"
FRONTEND_PKG="frontend/package.json"

if [ ! -f "${VERSION_FILE}" ]; then
    echo "1.0.0" > "${VERSION_FILE}"
fi

CURRENT_VERSION=$(tr -d ' \n\r' < "${VERSION_FILE}")

# Parse SemVer
IFS='.' read -r MAJOR MINOR PATCH <<< "${CURRENT_VERSION}"
MAJOR=${MAJOR:-1}
MINOR=${MINOR:-0}
PATCH=${PATCH:-0}

ACTION="${1:-get}"

case "${ACTION}" in
    get|current)
        echo "${CURRENT_VERSION}"
        exit 0
        ;;
    patch)
        PATCH=$((PATCH + 1))
        NEW_VERSION="${MAJOR}.${MINOR}.${PATCH}"
        ;;
    minor)
        MINOR=$((MINOR + 1))
        PATCH=0
        NEW_VERSION="${MAJOR}.${MINOR}.${PATCH}"
        ;;
    major)
        MAJOR=$((MAJOR + 1))
        MINOR=0
        PATCH=0
        NEW_VERSION="${MAJOR}.${MINOR}.${PATCH}"
        ;;
    set)
        if [ -z "${2:-}" ]; then
            echo "Error: 'set' requires version argument (e.g. bump-version.sh set 1.2.3)" >&2
            exit 1
        fi
        NEW_VERSION="$2"
        ;;
    *)
        echo "Usage: $0 [get|patch|minor|major|set <version>]"
        exit 1
        ;;
esac

echo "==> Bumping version: ${CURRENT_VERSION} -> ${NEW_VERSION}"

# 1. Update VERSION file
echo "${NEW_VERSION}" > "${VERSION_FILE}"

# 2. Update wails.json
if [ -f "${WAILS_FILE}" ]; then
    sed -i -E "s/\"productVersion\": *\"[^\"]+\"/\"productVersion\": \"${NEW_VERSION}\"/" "${WAILS_FILE}"
fi

# 3. Update pkg/version/version.go default
if [ -f "${PKG_VERSION_FILE}" ]; then
    sed -i -E "s/Version *= *\"[^\"]+\"/Version = \"${NEW_VERSION}\"/" "${PKG_VERSION_FILE}"
fi

# 4. Update frontend/package.json
if [ -f "${FRONTEND_PKG}" ]; then
    sed -i -E "s/\"version\": *\"[^\"]+\"/\"version\": \"${NEW_VERSION}\"/" "${FRONTEND_PKG}"
fi

# 5. Update packaging metadata files
if [ -f packaging/chocolatey/newscid.nuspec ]; then
    sed -i -E "s/<version>[^<]+<\/version>/<version>${NEW_VERSION}<\/version>/" packaging/chocolatey/newscid.nuspec
fi

if [ -f packaging/deb/DEBIAN/control ]; then
    sed -i -E "s/^Version: .*/Version: ${NEW_VERSION}-1/" packaging/deb/DEBIAN/control
fi

if [ -f packaging/snap/snapcraft.yaml ]; then
    sed -i -E "s/^version: .*/version: '${NEW_VERSION}'/" packaging/snap/snapcraft.yaml
fi

if [ -f packaging/rpm/newscid.spec ]; then
    sed -i -E "s/^Version: .*/Version:        ${NEW_VERSION}/" packaging/rpm/newscid.spec
fi

if [ -f packaging/pacman/PKGBUILD ]; then
    sed -i -E "s/^pkgver=.*/pkgver=${NEW_VERSION}/" packaging/pacman/PKGBUILD
fi

if [ -f packaging/pacman/.SRCINFO ]; then
    sed -i -E "s/pkgver = .*/pkgver = ${NEW_VERSION}/" packaging/pacman/.SRCINFO
    sed -i -E "s/tag=v[0-9.]+/tag=v${NEW_VERSION}/" packaging/pacman/.SRCINFO
fi

if [ -f packaging/man/newscid.1 ]; then
    sed -i -E "s/NewScid [0-9.]+/NewScid ${NEW_VERSION}/" packaging/man/newscid.1
fi

echo "✔ Version synchronized across VERSION, wails.json, pkg/version, frontend, and packaging manifests: v${NEW_VERSION}"
