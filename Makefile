# NewScid Makefile
# Automated dependency management, testing, and building

.DEFAULT_GOAL := help
.PHONY: help deps setup dev build build-frontend generate test test-coverage lint clean clean-dist run serve \
        install install-user uninstall \
        appimage package-appimage snap package-snap deb package-deb pacman package-pacman \
        chocolatey choco package-chocolatey package-choco brew homebrew package-brew \
        flatpak package-flatpak android android-apk packages package package-all package-checksums \
        version bump-patch bump-minor bump-major tag-release

# Application Metadata & Versioning (Single source of truth: VERSION)
VERSION_FILE := VERSION
APP_NAME     ?= newscid
APP_TITLE    ?= NewScid
APP_VERSION  ?= $(shell cat $(VERSION_FILE) 2>/dev/null || echo 1.0.0)
APP_RELEASE  ?= 1
APP_ARCH     ?= amd64
APP_DESC     ?= Modern Chess Information Database (Scid vs. PC Rewrite)
APP_AUTHOR   ?= Christopher Patrick Fair
APP_EMAIL    ?= christopherpatrickfair@gmail.com
APP_URL      ?= https://github.com/christopherpatrickfair/newscid

# Build & Git Metadata
GIT_COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "development")
BUILD_DATE   ?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
LDFLAGS      := -X newscid/pkg/version.Version=$(APP_VERSION) \
                -X newscid/pkg/version.GitCommit=$(GIT_COMMIT) \
                -X newscid/pkg/version.BuildDate=$(BUILD_DATE)

# Build Directories & Artifacts
BIN_DIR     := build/bin
DIST_DIR    := build/dist
STAGING_DIR := build/staging
BINARY      := $(BIN_DIR)/$(APP_NAME)

# Color definitions
CYAN   := \033[36m
GREEN  := \033[32m
YELLOW := \033[33m
BLUE   := \033[34m
BOLD   := \033[1m
RESET  := \033[0m

help: ## Display this help menu with all available targets
	@printf "$(BOLD)$(CYAN)========================================================================$(RESET)\n"
	@printf "$(BOLD)$(CYAN)  NewScid - Modern Chess Information Database (Scid vs. PC Rewrite)$(RESET)\n"
	@printf "$(BOLD)$(CYAN)========================================================================$(RESET)\n"
	@printf "$(BOLD)Usage:$(RESET) make $(GREEN)<target>$(RESET)\n\n"
	@printf "$(BOLD)Available Targets:$(RESET)\n"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-20s$(RESET) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@printf "\n$(BOLD)Quick Start:$(RESET)\n"
	@printf "  1. $(YELLOW)make deps$(RESET)        # Install mise toolchain and dependencies\n"
	@printf "  2. $(YELLOW)make test$(RESET)        # Run all unit and integration tests\n"
	@printf "  3. $(YELLOW)make build$(RESET)       # Build standalone production binary\n"
	@printf "  4. $(YELLOW)make packages$(RESET)    # Build all distribution packages\n"
	@printf "  5. $(YELLOW)make dev$(RESET)         # Launch interactive development mode\n\n"

deps: setup ## Alias for setup (installs all toolchain and library dependencies)

setup: ## Setup toolchain via mise, download Go modules, and install npm packages
	@printf "$(BOLD)$(BLUE)==> Installing toolchain via mise...$(RESET)\n"
	@mise install
	@printf "$(BOLD)$(BLUE)==> Downloading Go modules...$(RESET)\n"
	@go mod tidy
	@go mod download
	@printf "$(BOLD)$(BLUE)==> Installing frontend dependencies via npm...$(RESET)\n"
	@cd frontend && npm install
	@printf "$(BOLD)$(GREEN)✔ Dependencies installed successfully.$(RESET)\n"

dev: ## Run live development server with hot-reload (Wails Dev)
	@printf "$(BOLD)$(BLUE)==> Starting Wails development server...$(RESET)\n"
	@wails dev

build: ## Build the standalone production binary (build/bin/newscid)
	@printf "$(BOLD)$(BLUE)==> Building NewScid production desktop application (v$(APP_VERSION), commit $(GIT_COMMIT))...$(RESET)\n"
	@wails build -ldflags "$(LDFLAGS)"
	@printf "$(BOLD)$(GREEN)✔ Binary built at build/bin/newscid$(RESET)\n"


build-frontend: ## Compile the Vue 3 + Vite frontend assets
	@printf "$(BOLD)$(BLUE)==> Compiling Vue 3 frontend...$(RESET)\n"
	@cd frontend && npm run build
	@printf "$(BOLD)$(GREEN)✔ Frontend compiled.$(RESET)\n"

generate: ## Regenerate TypeScript bindings from Go App service
	@printf "$(BOLD)$(BLUE)==> Generating Wails TypeScript bindings...$(RESET)\n"
	@wails generate module
	@printf "$(BOLD)$(GREEN)✔ Bindings generated in frontend/wailsjs/$(RESET)\n"

test: ## Run all Go unit and integration tests with testify
	@printf "$(BOLD)$(BLUE)==> Running test suite across all packages...$(RESET)\n"
	@go test -v -count=1 ./...
	@printf "$(BOLD)$(GREEN)✔ All tests passed successfully.$(RESET)\n"

test-coverage: ## Run tests with code coverage analysis
	@printf "$(BOLD)$(BLUE)==> Running tests with coverage...$(RESET)\n"
	@go test -count=1 -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out
	@printf "$(BOLD)$(GREEN)✔ Coverage profile saved to coverage.out$(RESET)\n"

lint: ## Run static analysis and linting on Go and TypeScript code
	@printf "$(BOLD)$(BLUE)==> Checking Go code (go vet)...$(RESET)\n"
	@go vet ./...
	@printf "$(BOLD)$(BLUE)==> Typechecking Vue 3 code (vue-tsc)...$(RESET)\n"
	@cd frontend && npx vue-tsc --noEmit
	@printf "$(BOLD)$(GREEN)✔ Linting and typechecks passed.$(RESET)\n"

run: ## Run the built production application binary
	@if [ -f build/bin/newscid ]; then \
		printf "$(BOLD)$(BLUE)==> Launching build/bin/newscid...$(RESET)\n"; \
		./build/bin/newscid; \
	else \
		printf "$(BOLD)$(YELLOW)Binary not found. Building first...$(RESET)\n"; \
		$(MAKE) build && ./build/bin/newscid; \
	fi

serve: build ## Run NewScid in standalone web / mobile server mode (for Android / browser / PWA)
	@if [ -f build/bin/newscid ]; then \
		printf "$(BOLD)$(BLUE)==> Launching NewScid Server on http://0.0.0.0:8080...$(RESET)\n"; \
		./build/bin/newscid serve --host 0.0.0.0 --port 8080; \
	else \
		$(MAKE) build && ./build/bin/newscid serve --host 0.0.0.0 --port 8080; \
	fi

# ==============================================================================
# Versioning & Release Management
# ==============================================================================

version: ## Display current version, git commit, and build metadata
	@printf "$(BOLD)Application:$(RESET) $(CYAN)$(APP_TITLE)$(RESET)\n"
	@printf "$(BOLD)Version:    $(RESET) $(GREEN)v$(APP_VERSION)$(RESET) (from $(VERSION_FILE))\n"
	@printf "$(BOLD)Git Commit: $(RESET) $(YELLOW)$(GIT_COMMIT)$(RESET)\n"
	@printf "$(BOLD)Build Date: $(RESET) $(BLUE)$(BUILD_DATE)$(RESET)\n"

bump-patch: ## Bump semantic patch version (e.g. 1.0.0 -> 1.0.1)
	@./packaging/scripts/bump-version.sh patch

bump-minor: ## Bump semantic minor version (e.g. 1.0.0 -> 1.1.0)
	@./packaging/scripts/bump-version.sh minor

bump-major: ## Bump semantic major version (e.g. 1.0.0 -> 2.0.0)
	@./packaging/scripts/bump-version.sh major

tag-release: ## Create and annotate git release tag for current version
	@printf "$(BOLD)$(BLUE)==> Creating git release tag v$(APP_VERSION)...$(RESET)\n"
	@if [ -n "$$(git status --porcelain)" ]; then \
		printf "$(BOLD)$(YELLOW)Warning: Working directory has uncommitted changes. Please commit first.$(RESET)\n"; \
		exit 1; \
	fi
	@git tag -a "v$(APP_VERSION)" -m "Release v$(APP_VERSION)"
	@printf "$(BOLD)$(GREEN)✔ Tag v$(APP_VERSION) created successfully.$(RESET)\n"
	@printf "  Run $(YELLOW)git push origin v$(APP_VERSION)$(RESET) to publish tag.\n"

# ==============================================================================
# Packaging Targets
# ==============================================================================

appimage: build ## Build standalone Linux AppImage package (.AppImage)
	@APP_VERSION=$(APP_VERSION) ./packaging/scripts/build-appimage.sh $(BINARY) $(DIST_DIR)

package-appimage: appimage ## Alias for appimage

snap: build ## Build Canonical Snap package (.snap)
	@APP_VERSION=$(APP_VERSION) ./packaging/scripts/build-snap.sh $(BINARY) $(DIST_DIR)

package-snap: snap ## Alias for snap

deb: build ## Build Debian/Ubuntu package (.deb)
	@APP_VERSION=$(APP_VERSION) APP_RELEASE=$(APP_RELEASE) ./packaging/scripts/build-deb.sh $(BINARY) $(DIST_DIR)

package-deb: deb ## Alias for deb

pacman: build ## Build Arch Linux pacman package (.pkg.tar.*)
	@APP_VERSION=$(APP_VERSION) APP_RELEASE=$(APP_RELEASE) ./packaging/scripts/build-pacman.sh $(BINARY) $(DIST_DIR)

package-pacman: pacman ## Alias for pacman

chocolatey: ## Build Windows Chocolatey package (.nupkg)
	@APP_VERSION=$(APP_VERSION) ./packaging/scripts/build-chocolatey.py $(DIST_DIR)

choco: chocolatey ## Alias for chocolatey
package-chocolatey: chocolatey ## Alias for chocolatey
package-choco: chocolatey ## Alias for chocolatey

brew: build ## Build macOS Homebrew Cask and Formula (.rb + .zip)
	@APP_VERSION=$(APP_VERSION) ./packaging/scripts/build-brew.sh $(BINARY) $(DIST_DIR)

homebrew: brew ## Alias for brew
package-brew: brew ## Alias for brew

flatpak: build ## Build Flatpak bundle or staged export package
	@APP_VERSION=$(APP_VERSION) ./packaging/scripts/build-flatpak.sh $(BINARY) $(DIST_DIR)

package-flatpak: flatpak ## Alias for flatpak

android: android-apk ## Build standalone installable Android APK and Termux ARM64 binary

android-apk: ## Build standalone installable Android APK package (.apk)
	@./packaging/scripts/build-android.sh

packages: appimage snap deb pacman chocolatey brew flatpak android package-checksums ## Build all distribution packages (Linux, Windows, macOS, Android)
	@printf "\n$(BOLD)$(GREEN)========================================================================$(RESET)\n"
	@printf "$(BOLD)$(GREEN)✔ All packages successfully generated in $(DIST_DIR)/$(RESET)\n"
	@printf "$(BOLD)$(GREEN)========================================================================$(RESET)\n"
	@ls -lh $(DIST_DIR)

package: packages ## Alias for packages
package-all: packages ## Alias for packages

package-checksums: ## Generate SHA-256 checksums for all packages in build/dist/
	@printf "$(BOLD)$(BLUE)==> Generating SHA-256 checksums in $(DIST_DIR)/SHA256SUMS...$(RESET)\n"
	@mkdir -p $(DIST_DIR)
	@cd $(DIST_DIR) && rm -f SHA256SUMS && sha256sum * > SHA256SUMS 2>/dev/null || true
	@printf "$(BOLD)$(GREEN)✔ SHA256SUMS generated.$(RESET)\n"

# ==============================================================================
# Installation Targets
# ==============================================================================

install: install-user ## Install NewScid to user local environment (~/.local)

install-user: build ## Install binary, desktop entry, mime info, and icons to ~/.local
	@printf "$(BOLD)$(BLUE)==> Installing NewScid to $(HOME)/.local...$(RESET)\n"
	@mkdir -p $(HOME)/.local/bin $(HOME)/.local/share/applications $(HOME)/.local/share/metainfo $(HOME)/.local/share/mime/packages $(HOME)/.local/share/man/man1
	@install -Dm755 $(BINARY) $(HOME)/.local/bin/$(APP_NAME)
	@install -Dm644 packaging/desktop/newscid.desktop $(HOME)/.local/share/applications/newscid.desktop
	@install -Dm644 packaging/desktop/org.newscid.NewScid.metainfo.xml $(HOME)/.local/share/metainfo/org.newscid.NewScid.metainfo.xml
	@install -Dm644 packaging/desktop/newscid.mime.xml $(HOME)/.local/share/mime/packages/newscid.xml
	@install -Dm644 packaging/man/newscid.1 $(HOME)/.local/share/man/man1/newscid.1
	@python3 packaging/scripts/generate-icons.py build/appicon.png build/staging/icons
	@for size in 16 32 48 64 128 256 512 1024; do \
		if [ -f "build/staging/icons/$${size}x$${size}.png" ]; then \
			install -Dm644 "build/staging/icons/$${size}x$${size}.png" "$(HOME)/.local/share/icons/hicolor/$${size}x$${size}/apps/newscid.png"; \
		fi \
	done
	@update-desktop-database $(HOME)/.local/share/applications 2>/dev/null || true
	@update-mime-database $(HOME)/.local/share/mime 2>/dev/null || true
	@gtk-update-icon-cache -q -t -f $(HOME)/.local/share/icons/hicolor 2>/dev/null || true
	@printf "$(BOLD)$(GREEN)✔ NewScid v$(APP_VERSION) successfully installed to $(HOME)/.local/bin/$(APP_NAME)$(RESET)\n"

uninstall: ## Remove NewScid from user local environment (~/.local)
	@printf "$(BOLD)$(YELLOW)==> Uninstalling NewScid from $(HOME)/.local...$(RESET)\n"
	@rm -f $(HOME)/.local/bin/$(APP_NAME)
	@rm -f $(HOME)/.local/share/applications/newscid.desktop
	@rm -f $(HOME)/.local/share/metainfo/org.newscid.NewScid.metainfo.xml
	@rm -f $(HOME)/.local/share/mime/packages/newscid.xml
	@rm -f $(HOME)/.local/share/man/man1/newscid.1
	@for size in 16 32 48 64 128 256 512 1024; do \
		rm -f $(HOME)/.local/share/icons/hicolor/$${size}x$${size}/apps/newscid.png; \
	done
	@update-desktop-database $(HOME)/.local/share/applications 2>/dev/null || true
	@update-mime-database $(HOME)/.local/share/mime 2>/dev/null || true
	@printf "$(BOLD)$(GREEN)✔ NewScid successfully uninstalled from $(HOME)/.local.$(RESET)\n"

clean-dist: ## Remove all packaged distribution artifacts in build/dist/ and staging
	@printf "$(BOLD)$(YELLOW)==> Cleaning distribution packages...$(RESET)\n"
	@rm -rf $(DIST_DIR) $(STAGING_DIR)
	@printf "$(BOLD)$(GREEN)✔ Distribution artifacts removed.$(RESET)\n"

clean: clean-dist ## Remove build artifacts, coverage outputs, and temporary files
	@printf "$(BOLD)$(YELLOW)==> Cleaning build artifacts...$(RESET)\n"
	@rm -rf build/bin coverage.out
	@rm -rf frontend/dist
	@printf "$(BOLD)$(GREEN)✔ Clean completed.$(RESET)\n"

