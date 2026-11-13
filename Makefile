# Common project commands. Run `make` (or `make help`) to list them.
API_DIR := api

.DEFAULT_GOAL := help
.PHONY: help setup run build test lint fmt tidy vuln check

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-8s\033[0m %s\n", $$1, $$2}'

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
