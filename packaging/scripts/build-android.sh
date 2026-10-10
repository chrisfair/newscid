#!/usr/bin/env bash
# ==============================================================================
# NewScid - Android Packaging Script
# Builds:
#   1. Standalone installable Android APK (packaging/android)
#   2. Standalone Android ARM64 server binary (for Termux / CLI / PWA)
# ==============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

VERSION_FILE="${ROOT_DIR}/VERSION"
APP_VERSION="$(cat "${VERSION_FILE}" 2>/dev/null || echo "1.0.0")"
DIST_DIR="${ROOT_DIR}/build/dist"
mkdir -p "${DIST_DIR}"

echo -e "\033[1;36m==> Building NewScid for Android (v${APP_VERSION})...\033[0m"

# 1. Detect Java JDK (compatible with Gradle 8.x, e.g. Java 17-21)
if [ -z "${JAVA_HOME:-}" ]; then
    if [ -d "/opt/android-studio/jbr" ]; then
        export JAVA_HOME="/opt/android-studio/jbr"
    elif [ -d "/usr/lib/jvm/java-21-openjdk" ] && [ -x "/usr/lib/jvm/java-21-openjdk/bin/javac" ]; then
        export JAVA_HOME="/usr/lib/jvm/java-21-openjdk"
    elif [ -d "/usr/lib/jvm/java-17-openjdk" ]; then
        export JAVA_HOME="/usr/lib/jvm/java-17-openjdk"
    fi
fi
if [ -n "${JAVA_HOME:-}" ]; then
    export PATH="${JAVA_HOME}/bin:${PATH}"
    echo "  Using JDK: ${JAVA_HOME}"
fi

# 2. Detect Android SDK
if [ -z "${ANDROID_HOME:-}" ]; then
    if [ -d "${HOME}/.android-sdk" ]; then
        export ANDROID_HOME="${HOME}/.android-sdk"
    elif [ -d "/opt/android-sdk" ]; then
        export ANDROID_HOME="/opt/android-sdk"
    fi
fi
if [ -n "${ANDROID_HOME:-}" ]; then
    echo "  Using Android SDK: ${ANDROID_HOME}"
fi

# 3. Build Web Frontend
echo -e "\033[1;34m==> Compiling web frontend with mobile touch optimizations...\033[0m"
cd "${ROOT_DIR}/frontend"
npm run build

# 4. Cross-compile Go backend for Android ARM64
echo -e "\033[1;34m==> Cross-compiling Go backend for Android (arm64-v8a)...\033[0m"
cd "${ROOT_DIR}"
ASSETS_DIR="${ROOT_DIR}/packaging/android/app/src/main/assets"
mkdir -p "${ASSETS_DIR}"

GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build \
    -ldflags="-s -w -X newscid/pkg/version.Version=${APP_VERSION} -X newscid/pkg/version.GitCommit=$(git rev-parse --short HEAD 2>/dev/null || echo release)" \
    -o "${ASSETS_DIR}/newscid-server-arm64" .

# Also copy to build/dist for direct Termux / Android CLI use
cp "${ASSETS_DIR}/newscid-server-arm64" "${DIST_DIR}/newscid-v${APP_VERSION}-android-arm64"
chmod +x "${DIST_DIR}/newscid-v${APP_VERSION}-android-arm64"
echo -e "  \033[1;32m✔ Standalone Android ARM64 binary created:\033[0m ${DIST_DIR}/newscid-v${APP_VERSION}-android-arm64"

# 5. Assemble Android APK via Gradle
echo -e "\033[1;34m==> Assembling Android APK package...\033[0m"
cd "${ROOT_DIR}/packaging/android"

if [ -f "./gradlew" ]; then
    GRADLE_CMD="./gradlew"
elif command -v gradle >/dev/null 2>&1; then
    GRADLE_CMD="gradle"
elif [ -x "${HOME}/.local/share/mise/installs/gradle/8.10.2/gradle-8.10.2/bin/gradle" ]; then
    GRADLE_CMD="${HOME}/.local/share/mise/installs/gradle/8.10.2/gradle-8.10.2/bin/gradle"
else
    echo -e "  \033[1;33mWarning: Gradle not found. APK assembly skipped, binary is ready.\033[0m"
    exit 0
fi

${GRADLE_CMD} assembleDebug

APK_SRC="${ROOT_DIR}/packaging/android/app/build/outputs/apk/debug/app-debug.apk"
if [ -f "${APK_SRC}" ]; then
    APK_DEST="${DIST_DIR}/newscid-v${APP_VERSION}-android-arm64.apk"
    cp "${APK_SRC}" "${APK_DEST}"
    echo -e "  \033[1;32m✔ Android APK successfully built:\033[0m ${APK_DEST}"
    ls -lh "${APK_DEST}"
    
    # Checksum
    cd "${DIST_DIR}"
    sha256sum "newscid-v${APP_VERSION}-android-arm64.apk" > "newscid-v${APP_VERSION}-android-arm64.apk.sha256"
fi

echo -e "\n\033[1;32m========================================================================\033[0m"
echo -e "\033[1;32m✔ Android Packaging Complete!\033[0m"
echo -e "\033[1;32m========================================================================\033[0m"
echo -e "To install on your Android device:"
echo -e "  1. Sideload APK via ADB: \033[1;33madb install -r build/dist/newscid-v${APP_VERSION}-android-arm64.apk\033[0m"
echo -e "  2. Or transfer the .apk file to your device (via USB, cloud, or email) and tap to install."
echo -e "  3. For Termux: copy \033[1;33mnewscid-v${APP_VERSION}-android-arm64\033[0m to Termux, chmod +x, and run \033[1;33m./newscid serve\033[0m!"
