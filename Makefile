# Makefile for contextify - cross-platform builds

# Application name
APP_NAME := contextify

# Version (can be overridden)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Build directory
BUILD_DIR := build

# Go build flags
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

# Default target
.PHONY: all
all: clean build

# Build for all platforms
.PHONY: build
build: build-darwin-amd64 build-darwin-arm64 build-windows-amd64

# Clean build directory
.PHONY: clean
clean:
	@echo "Cleaning build directory..."
	@rm -rf $(BUILD_DIR)
	@mkdir -p $(BUILD_DIR)

# macOS Intel (amd64)
.PHONY: build-darwin-amd64
build-darwin-amd64:
	@echo "Building for macOS Intel (amd64)..."
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 .

# macOS Apple Silicon (arm64)
.PHONY: build-darwin-arm64
build-darwin-arm64:
	@echo "Building for macOS Apple Silicon (arm64)..."
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 .

# Windows (amd64)
.PHONY: build-windows-amd64
build-windows-amd64:
	@echo "Building for Windows (amd64)..."
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe .

# Build for current platform only
.PHONY: build-local
build-local:
	@echo "Building for current platform..."
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) .

# Install locally (builds and copies to GOPATH/bin or GOBIN)
.PHONY: install
install:
	@echo "Installing $(APP_NAME)..."
	@go install $(LDFLAGS) .

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@go test -v ./...

# Run linter (requires golangci-lint)
.PHONY: lint
lint:
	@echo "Running linter..."
	@golangci-lint run

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Tidy dependencies
.PHONY: tidy
tidy:
	@echo "Tidying dependencies..."
	@go mod tidy

# Create release archives
.PHONY: package
package: build
	@echo "Creating release packages..."
	@cd $(BUILD_DIR) && \
	tar -czf $(APP_NAME)-darwin-amd64.tar.gz $(APP_NAME)-darwin-amd64 && \
	tar -czf $(APP_NAME)-darwin-arm64.tar.gz $(APP_NAME)-darwin-arm64 && \
	zip $(APP_NAME)-windows-amd64.zip $(APP_NAME)-windows-amd64.exe
	@echo "Packages created in $(BUILD_DIR)/"

# Show build info
.PHONY: info
info:
    @echo "Application: $(APP_NAME)"
    @echo "Version: $(VERSION)"
    @echo "Go version: $(shell go version)"
    @echo "Build directory: $(BUILD_DIR)"

# Development workflow
.PHONY: dev
dev: fmt tidy test build-local

# Release workflow
.PHONY: release
release: clean fmt tidy test build package

# Help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all              - Clean and build for all platforms"
	@echo "  build            - Build for all platforms"
	@echo "  build-local      - Build for current platform only"
	@echo "  build-darwin-amd64 - Build for macOS Intel"
	@echo "  build-darwin-arm64 - Build for macOS Apple Silicon"
	@echo "  build-windows-amd64 - Build for Windows"
	@echo "  clean            - Clean build directory"
	@echo "  install          - Install locally"
	@echo "  test             - Run tests"
	@echo "  lint             - Run linter (requires golangci-lint)"
	@echo "  fmt              - Format code"
	@echo "  tidy             - Tidy dependencies"
	@echo "  package          - Create release archives"
	@echo "  dev              - Development workflow (fmt, tidy, test, build-local)"
	@echo "  release          - Release workflow (clean, fmt, tidy, test, build, package)"
	@echo "  info             - Show build information"
	@echo "  help             - Show this help"
