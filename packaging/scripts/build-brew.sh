#!/usr/bin/env bash
set -euo pipefail

VERSION="${APP_VERSION:-1.0.0}"
BINARY_PATH="${1:-build/bin/newscid}"
DIST_DIR="${2:-build/dist}"
STAGE_DIR="build/staging/brew"
APP_DIR="${STAGE_DIR}/NewScid.app"
CONTENTS_DIR="${APP_DIR}/Contents"
ZIP_OUTPUT="${DIST_DIR}/NewScid-${VERSION}-darwin.zip"

echo "==> Packaging Homebrew formula & macOS distribution bundle for NewScid v${VERSION}..."

rm -rf "${STAGE_DIR}"
mkdir -p "${CONTENTS_DIR}/MacOS"
mkdir -p "${CONTENTS_DIR}/Resources"
mkdir -p "${DIST_DIR}"

# 1. Binary or launcher
if [ -f "${BINARY_PATH}" ]; then
    cp "${BINARY_PATH}" "${CONTENTS_DIR}/MacOS/newscid"
    chmod 755 "${CONTENTS_DIR}/MacOS/newscid"
else
    # Create placeholder binary if building formula without mac binary
    cat << 'EOF' > "${CONTENTS_DIR}/MacOS/newscid"
#!/bin/sh
echo "NewScid - Modern Chess Information Database"
EOF
    chmod 755 "${CONTENTS_DIR}/MacOS/newscid"
fi

# 2. PkgInfo
echo -n "APPL????" > "${CONTENTS_DIR}/PkgInfo"

# 3. Info.plist
if [ -f "build/darwin/Info.plist" ]; then
    sed -e "s/{{.Info.ProductName}}/NewScid/g" \
        -e "s/{{.OutputFilename}}/newscid/g" \
        -e "s/com.wails.{{.Name}}/com.wails.newscid/g" \
        -e "s/{{.Info.ProductVersion}}/${VERSION}/g" \
        -e "s/{{.Info.Comments}}/Modern Chess Information Database/g" \
        -e "s/{{.Info.Copyright}}/Copyright 2026 Christopher Patrick Fair/g" \
        build/darwin/Info.plist > "${CONTENTS_DIR}/Info.plist"
else
    cat << EOF > "${CONTENTS_DIR}/Info.plist"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleName</key>
    <string>NewScid</string>
    <key>CFBundleExecutable</key>
    <string>newscid</string>
    <key>CFBundleIdentifier</key>
    <string>com.wails.newscid</string>
    <key>CFBundleVersion</key>
    <string>${VERSION}</string>
    <key>CFBundleShortVersionString</key>
    <string>${VERSION}</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.13.0</string>
    <key>NSHighResolutionCapable</key>
    <true/>
</dict>
</plist>
EOF
fi

# 4. Resources Icon
cp build/appicon.png "${CONTENTS_DIR}/Resources/iconfile.png"

# 5. Archive macOS app into zip
(
    cd "${STAGE_DIR}"
    python3 -m zipfile -c "${OLDPWD}/${ZIP_OUTPUT}" "NewScid.app"
)

# 6. Compute SHA-256
SHA256=$(sha256sum "${ZIP_OUTPUT}" | cut -d' ' -f1)

# 7. Render Cask and Formula
CASK_OUTPUT="${DIST_DIR}/newscid.cask.rb"
FORMULA_OUTPUT="${DIST_DIR}/newscid.rb"

sed -e "s/{{VERSION}}/${VERSION}/g" \
    -e "s/{{SHA256}}/${SHA256}/g" \
    packaging/brew/newscid.cask.rb.template > "${CASK_OUTPUT}"

sed -e "s/{{VERSION}}/${VERSION}/g" \
    -e "s/{{SOURCE_SHA256}}/${SHA256}/g" \
    packaging/brew/newscid.rb.template > "${FORMULA_OUTPUT}"

echo "✔ Homebrew macOS bundle built: ${ZIP_OUTPUT} ($(du -h "${ZIP_OUTPUT}" | cut -f1))"
echo "✔ Homebrew Cask generated: ${CASK_OUTPUT} (SHA-256: ${SHA256})"
echo "✔ Homebrew Formula generated: ${FORMULA_OUTPUT}"
