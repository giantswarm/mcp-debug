.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z%\\\/_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build

.PHONY: build
build: ## Build the binary for the current platform
	go build -o mcp-debug

.PHONY: install
install: build ## Install the binary
	mv mcp-debug $(GOPATH)/bin/mcp-debug

##@ Development

.PHONY: run
run: ## Run the agent in REPL mode
	go run . agent --repl

.PHONY: run-mcp
run-mcp: ## Run the agent as MCP server
	go run . agent --mcp-server

.PHONY: run-verbose
run-verbose: ## Run the agent with verbose logging
	go run . agent --verbose --json-rpc

##@ Testing

.PHONY: test
test: ## Run go test and go vet
	@echo "Running Go tests..."
	@go test -cover ./...
	@echo "Running go vet..."
	@go vet ./...

.PHONY: clean
clean: ## Clean build artifacts
	rm -f mcp-debug
	rm -rf dist/ 