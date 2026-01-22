.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.PHONY: proto-lint
proto-lint: ## Lint proto files
	buf lint

.PHONY: proto-breaking
proto-breaking: ## Check for breaking changes in proto files
	buf breaking --against '.git#branch=main'

# Binary list
BINARIES := bin/shoes-vz-server bin/shoes-vz-agent bin/shoes-vz-runner-agent bin/shoes-vz-client

# Source files
GO_SOURCES := $(shell find . -type f -name '*.go' -not -path './gen/*' 2>/dev/null || true)
PROTO_SOURCES := $(shell find apis/proto -type f -name '*.proto' 2>/dev/null || true)

# Generated sources
GENERATED_SOURCES := $(wildcard gen/go/**/*.go)

.PHONY: generate
generate: ## Generate all code (proto, etc.)
	buf generate

# Ensure generated code exists before building
.PHONY: ensure-generated
ensure-generated:
	@if [ ! -d gen/go ] || [ -z "$$(ls -A gen/go 2>/dev/null)" ]; then \
		echo "Generating code from proto files..."; \
		$(MAKE) generate; \
	fi

.PHONY: build
build: $(BINARIES) ## Build all binaries

# Create bin directory
bin:
	mkdir -p bin

# Build rules for each binary
bin/shoes-vz-server: ensure-generated $(GO_SOURCES) $(GENERATED_SOURCES) shoes-vz.entitlements | bin
	go build -o $@ ./cmd/shoes-vz-server
	codesign -s - --entitlements shoes-vz.entitlements --force $@

bin/shoes-vz-agent: ensure-generated $(GO_SOURCES) $(GENERATED_SOURCES) shoes-vz.entitlements | bin
	go build -o $@ ./cmd/shoes-vz-agent
	codesign -s - --entitlements shoes-vz.entitlements --force $@

bin/shoes-vz-runner-agent: ensure-generated $(GO_SOURCES) $(GENERATED_SOURCES) shoes-vz.entitlements | bin
	go build -o $@ ./cmd/shoes-vz-runner-agent
	codesign -s - --entitlements shoes-vz.entitlements --force $@

bin/shoes-vz-client: ensure-generated $(GO_SOURCES) $(GENERATED_SOURCES) shoes-vz.entitlements | bin
	go build -o $@ ./cmd/shoes-vz-client
	codesign -s - --entitlements shoes-vz.entitlements --force $@

.PHONY: test
test: ## Run tests
	go test -v -race -cover ./...

.PHONY: fmt
fmt: ## Format Go code
	go fmt ./...

.PHONY: lint
lint: ## Run golangci-lint
	golangci-lint run

.PHONY: tidy
tidy: ## Tidy go modules
	go mod tidy

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf bin/
	rm -rf gen/

.PHONY: deps
deps: ## Install dependencies
	go tool -n github.com/bufbuild/buf/cmd/buf
	go tool -n google.golang.org/protobuf/cmd/protoc-gen-go
	go tool -n google.golang.org/grpc/cmd/protoc-gen-go-grpc

.DEFAULT_GOAL := help
