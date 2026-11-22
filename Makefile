# Common project commands. Run `make` (or `make help`) to list them.
#
# Targets are grouped by component: api-* (Go backend in api/) and web-*
# (Astro site in web/). The targets without a prefix act on the whole project.
API_DIR := api
WEB_DIR := web

# Version of a data release. Defaults to today; override with
# `make datasets VERSION=2026.11.21`.
VERSION ?= $(shell date +%Y.%m.%d)

.DEFAULT_GOAL := help
.PHONY: help setup fmt test check datasets \
	api-run api-build api-test api-lint api-fmt api-tidy api-vuln api-check \
	web-install web-dev web-build web-fmt web-lint web-typecheck web-test web-check

help: ## Show available commands
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make \033[36m<target>\033[0m\n"} \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } \
		/^[a-zA-Z_-]+:.*##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

##@ Project

setup: web-install ## Prepare a fresh clone: git hooks and site dependencies
	lefthook install

fmt: api-fmt web-fmt ## Format all code

test: api-test web-test ## Run all tests

check: api-check web-check ## Run every check of the project (same as CI)

##@ Data (data/)

datasets: ## Build the downloadable files of every dataset into dist/ (VERSION=YYYY.MM.DD, default: today)
	rm -rf dist
	cd $(API_DIR) && go run ./cmd/datasets -version $(VERSION) -data ../data -out ../dist

##@ API (api/)

api-run: ## Run the API server locally (http://localhost:8080)
	cd $(API_DIR) && go run ./cmd/server

api-build: ## Build the server binary into api/bin/server
	cd $(API_DIR) && go build -o bin/server ./cmd/server

api-test: ## Run API tests with the race detector
	cd $(API_DIR) && go test -race ./...

api-lint: ## Lint Go code with golangci-lint (also reports formatting)
	cd $(API_DIR) && golangci-lint run

api-fmt: ## Format Go code (gofmt + goimports)
	cd $(API_DIR) && golangci-lint fmt

api-tidy: ## Clean up go.mod and go.sum
	cd $(API_DIR) && go mod tidy

api-vuln: ## Check Go dependencies for known vulnerabilities
	cd $(API_DIR) && go tool govulncheck ./...

api-check: api-lint api-test api-vuln ## Run all API checks: lint, tests, vulnerabilities

##@ Site (web/)

web-install: ## Install site dependencies exactly as locked
	cd $(WEB_DIR) && pnpm install --frozen-lockfile

web-dev: ## Run the site dev server (http://localhost:4321)
	cd $(WEB_DIR) && pnpm dev

web-build: ## Build the static site into web/dist
	cd $(WEB_DIR) && pnpm build

web-fmt: ## Format site code with Prettier
	cd $(WEB_DIR) && pnpm format

web-lint: ## Lint site code with ESLint (includes a11y rules)
	cd $(WEB_DIR) && pnpm lint

web-typecheck: ## Type-check site code with astro check
	cd $(WEB_DIR) && pnpm typecheck

web-test: ## Run site unit tests with Vitest
	cd $(WEB_DIR) && pnpm test

web-check: ## Run all site checks: format, lint, types, tests
	cd $(WEB_DIR) && pnpm format:check && pnpm lint && pnpm typecheck && pnpm test
