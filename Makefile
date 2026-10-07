# NewScid Makefile
# Automated dependency management, testing, and building

.DEFAULT_GOAL := help
.PHONY: help deps setup dev build build-frontend generate test test-coverage lint clean run

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
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-18s$(RESET) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@printf "\n$(BOLD)Quick Start:$(RESET)\n"
	@printf "  1. $(YELLOW)make deps$(RESET)    # Install mise toolchain and dependencies\n"
	@printf "  2. $(YELLOW)make test$(RESET)    # Run all unit and integration tests\n"
	@printf "  3. $(YELLOW)make build$(RESET)   # Build standalone production binary\n"
	@printf "  4. $(YELLOW)make dev$(RESET)     # Launch interactive development mode\n\n"

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
	@printf "$(BOLD)$(BLUE)==> Building NewScid production desktop application...$(RESET)\n"
	@wails build
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

clean: ## Remove build artifacts, coverage outputs, and temporary files
	@printf "$(BOLD)$(YELLOW)==> Cleaning build artifacts...$(RESET)\n"
	@rm -rf build/bin coverage.out
	@rm -rf frontend/dist
	@printf "$(BOLD)$(GREEN)✔ Clean completed.$(RESET)\n"
