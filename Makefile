# ZStack CLI Makefile

# Variables
BINARY_NAME := goznp
BINARY_DIR := bin
CMD_DIR := ./cmd/goznp
GO := go

# Build info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Default target
.DEFAULT_GOAL := build

# Phony targets
.PHONY: all build build-daemon clean test test-verbose test-coverage test-integration fmt vet lint check install help

## build: Build the CLI binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	$(GO) build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Built $(BINARY_DIR)/$(BINARY_NAME)"

## build-daemon: Build the goznpd daemon binary
build-daemon:
	@echo "Building goznpd..."
	@mkdir -p $(BINARY_DIR)
	$(GO) build $(LDFLAGS) -o $(BINARY_DIR)/goznpd ./cmd/goznpd
	@echo "Built $(BINARY_DIR)/goznpd"

## build-all: Build CLI and daemon binaries
build-all: build build-daemon

## all: Run fmt, vet, test, and build
all: fmt vet test build

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BINARY_DIR)
	@$(GO) clean -testcache
	@echo "Done"

## test: Run tests
test:
	@echo "Running tests..."
	$(GO) test ./...

## test-verbose: Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	$(GO) test -v ./...

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(BINARY_DIR)
	$(GO) test -coverprofile=$(BINARY_DIR)/coverage.out ./...
	$(GO) tool cover -html=$(BINARY_DIR)/coverage.out -o $(BINARY_DIR)/coverage.html
	@echo "Coverage report: $(BINARY_DIR)/coverage.html"

## test-integration: Run hardware integration tests (requires GOZNP_PORT)
test-integration:
ifndef GOZNP_PORT
	@echo "Error: GOZNP_PORT environment variable is required"
	@echo "Usage: GOZNP_PORT=/dev/ttyUSB0 make test-integration"
	@exit 1
endif
	@echo "Running integration tests on $(GOZNP_PORT)..."
	$(GO) test -v -tags=integration ./pkg/adapter/...

## fmt: Format code
fmt:
	@echo "Formatting code..."
	$(GO) fmt ./...
	$(GO) run golang.org/x/tools/cmd/goimports@latest -w .

## vet: Run go vet
vet:
	@echo "Running go vet..."
	$(GO) vet ./...

## lint: Run golangci-lint
lint:
	@echo "Running linter..."
	$(GO) run github.com/golangci/golangci-lint/cmd/golangci-lint@latest run

## check: Run fmt, vet, and lint checks
check: fmt vet lint

## install: Install binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	$(GO) install $(LDFLAGS) $(CMD_DIR)
	@echo "Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

## run-daemon: Run the daemon
run-daemon:
	@if [ -z "$(GOZNP_PORT)" ]; then \
		echo "Error: GOZNP_PORT environment variable is required"; \
		echo "Usage: GOZNP_PORT=/dev/ttyUSB0 make run-daemon"; \
		exit 1; \
	fi
	@echo "Running goznpd..."
	@$(GO) run ./cmd/goznpd

## deps: Download and tidy dependencies
deps:
	@echo "Downloading dependencies..."
	$(GO) mod download
	$(GO) mod tidy

## help: Show this help message
help:
	@echo "ZStack CLI - Available targets:"
	@echo ""
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/  /'
	@echo ""
	@echo "Examples:"
	@echo "  make                                        # Build the binary"
	@echo "  make test                                   # Run unit tests"
	@echo "  make all                                    # Format, vet, test, and build"
	@echo "  make clean build                            # Clean and rebuild"
	@echo "  GOZNP_PORT=/dev/ttyUSB0 make test-integration  # Run hardware tests"
