# Common project commands. Run `make` (or `make help`) to list them.
API_DIR := api
WEB_DIR := web

.DEFAULT_GOAL := help
.PHONY: help setup run build test lint fmt tidy vuln check web-install web-dev web-build web-fmt web-lint web-typecheck web-check

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

setup: ## Install git hooks (run once after cloning)
	lefthook install

run: ## Run the API server locally
	cd $(API_DIR) && go run ./cmd/server

build: ## Build the server binary into api/bin/server
	cd $(API_DIR) && go build -o bin/server ./cmd/server

test: ## Run tests with the race detector
	cd $(API_DIR) && go test -race ./...

lint: ## Run golangci-lint
	cd $(API_DIR) && golangci-lint run

fmt: ## Format Go code (gofmt + goimports)
	cd $(API_DIR) && golangci-lint fmt

tidy: ## Clean up go.mod and go.sum
	cd $(API_DIR) && go mod tidy

vuln: ## Check dependencies for known vulnerabilities
	cd $(API_DIR) && go tool govulncheck ./...

check: lint test vuln ## Run all checks (same as CI)

web-install: ## Install web dependencies exactly as locked
	cd $(WEB_DIR) && pnpm install --frozen-lockfile

web-dev: ## Run the web dev server (http://localhost:4321)
	cd $(WEB_DIR) && pnpm dev

web-build: ## Build the static site into web/dist
	cd $(WEB_DIR) && pnpm build

web-fmt: ## Format web code with Prettier
	cd $(WEB_DIR) && pnpm format

web-lint: ## Lint web code with ESLint (includes a11y rules)
	cd $(WEB_DIR) && pnpm lint

web-typecheck: ## Type-check web code with astro check
	cd $(WEB_DIR) && pnpm typecheck

web-check: ## Run all web checks: format, lint, types (same as CI)
	cd $(WEB_DIR) && pnpm format:check && pnpm lint && pnpm typecheck
