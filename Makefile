.PHONY: help build test clean install fmt lint vet run benchmark docker-build docker-run release release-checksums

VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.commit=$(COMMIT)"

GO := go
GOFLAGS := -v
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

BINARY_NAME := luakit
DIST_DIR := dist
RELEASE_ARTIFACTS := $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64 \
                   $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-arm64 \
                   $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-amd64 \
                   $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-arm64

help:
	@echo "Available targets:"
	@echo "  make build          - Build the binary for the current platform"
	@echo "  make build-all      - Build binaries for all target platforms"
	@echo "  make release        - Build release artifacts for all platforms"
	@echo "  make checksums      - Generate SHA256 checksums for release artifacts"
	@echo "  make test           - Run tests"
	@echo "  make clean          - Clean build artifacts"
	@echo "  make install        - Install the binary to GOPATH/bin"
	@echo "  make fmt            - Format code"
	@echo "  make lint           - Run linter"
	@echo "  make vet            - Run go vet"
	@echo "  make benchmark      - Run benchmarks"
	@echo "  make docker-build   - Build Docker image"
	@echo "  make docker-run     - Run Docker container"

build:
	@echo "Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(DIST_DIR)
	$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME) ./cmd/luakit
	@echo "Binary built: $(DIST_DIR)/$(BINARY_NAME)"

build-all:
	@echo "Building binaries for all target platforms..."
	@mkdir -p $(DIST_DIR)
	@GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/luakit
	@GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/luakit
	@GOOS=darwin GOARCH=amd64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/luakit
	@GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/luakit
	@echo "All binaries built successfully"

release: build-all checksums

checksums:
	@echo "Generating SHA256 checksums..."
	@cd $(DIST_DIR) && shasum -a 256 $(BINARY_NAME)-* > checksums.txt
	@echo "Checksums generated: $(DIST_DIR)/checksums.txt"

test:
	@echo "Running tests..."
	$(GO) test -v -race -coverprofile=coverage.out -covermode=atomic ./...

test-integration:
	@echo "Running integration tests..."
	@if [ -z "$$BUILDKIT_HOST" ] && [ ! -S /run/buildkit/buildkitd.sock ]; then \
		echo "Error: BuildKit daemon not running"; \
		echo "Start with: docker run -d --name buildkitd --privileged -p 127.0.0.1:1234:1234 moby/buildkit:latest --addr tcp://0.0.0.0:1234"; \
		echo "Then set: export BUILDKIT_HOST=tcp://127.0.0.1:1234"; \
		exit 1; \
	fi
	$(GO) test -tags=e2e -v -timeout=15m ./test/integration/...

test-coverage:
	@echo "Generating coverage report..."
	$(GO) test -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(DIST_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

install:
	@echo "Installing $(BINARY_NAME) to $(shell go env GOPATH)/bin..."
	$(GO) install $(GOFLAGS) $(LDFLAGS) ./cmd/luakit
	@echo "Installed successfully"

fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	@echo "Formatting complete"

lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found. Install it from https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

vet:
	@echo "Running go vet..."
	$(GO) vet ./...

run: build
	@$(DIST_DIR)/$(BINARY_NAME) version

benchmark:
	@echo "Running benchmarks..."
	$(GO) test -bench=. -benchmem -benchtime=10s ./...

docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):latest .
	@echo "Docker image built: $(BINARY_NAME):latest"

docker-run: docker-build
	@echo "Running Docker container..."
	docker run --rm $(BINARY_NAME):latest version

verify-release:
	@echo "Verifying release artifacts..."
	@cd $(DIST_DIR) && \
	for artifact in $(BINARY_NAME)-*; do \
		echo "Verifying $$artifact..."; \
		$$artifact version; \
	done
	@echo "All release artifacts verified"

.PHONY: verify-release
